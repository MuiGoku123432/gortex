//go:build darwin || linux

package indexer

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
	"github.com/zzet/gortex/internal/parser"
	"github.com/zzet/gortex/internal/parser/languages"
	"github.com/zzet/gortex/internal/testutil/graphdump"
)

// The Phase 4 lifecycle harness (D-04): every fixture below is invented
// fixed-format COBOL. It must never be replaced with estate-derived source:
// this repository is public.

// cobolLifecycleRepoA is repository lifea. MAINPGM copies CUSTREC, calls
// SUBPGM, reads MySch.Orders, names IDMS record EMP-REC, and sends a CICS
// map with a literal mapset; the procedure copybook VALPARA names the same
// table spelled mysch.orders and the same IDMS record, so two documents
// write each shared resource.
var cobolLifecycleRepoA = map[string]string{
	"cobol/main.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. MAINPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CUSTREC.
       01 WS-X PIC X(10).
       PROCEDURE DIVISION.
       MAIN-PARA.
           CALL 'SUBPGM'.
           OBTAIN CALC EMP-REC.
           EXEC CICS SEND MAP('MAP1') MAPSET('MSET1') END-EXEC.
           EXEC SQL SELECT A INTO :WS-X FROM MySch.Orders END-EXEC.
           GOBACK.
`,
	"cobol/sub.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SUBPGM.
       PROCEDURE DIVISION.
       SUB-PARA.
           GOBACK.
`,
	"copy/custrec.cpy": `       01 CUST-REC.
          05 CUST-ID PIC X(8).
`,
	"copy/valpara.cpy": `       VALIDATE-PARA.
           OBTAIN CALC EMP-REC.
           EXEC SQL SELECT A INTO :WS-X FROM mysch.orders END-EXEC.
`,
}

// cobolLifecycleRepoB is repository lifeb: a second SUBPGM and a caller
// that reads the unqualified ORDERS table.
var cobolLifecycleRepoB = map[string]string{
	"cobol/sub.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SUBPGM.
       PROCEDURE DIVISION.
       SUB-PARA.
           GOBACK.
`,
	"cobol/caller.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. CALLER.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-X PIC X(10).
       PROCEDURE DIVISION.
       CALL-PARA.
           CALL 'SUBPGM'.
           EXEC SQL SELECT A INTO :WS-X FROM ORDERS END-EXEC.
           GOBACK.
`,
}

// cobolLifecycleRepos is the two-repository corpus every clean build uses.
var cobolLifecycleRepos = map[string]map[string]string{
	"lifea": cobolLifecycleRepoA,
	"lifeb": cobolLifecycleRepoB,
}

// cobolLifecycleEnv is one SQLite store with a MultiIndexer over a set of
// repositories written under dir, keyed by repository name.
type cobolLifecycleEnv struct {
	dir   string
	store *store_sqlite.Store
	cm    *config.ConfigManager
	mi    *MultiIndexer
	roots map[string]string
	repos map[string]map[string]string
}

// newCobolLifecycleEnv writes every repository under a fresh directory and
// opens a SQLite store, the registered extractors, and a MultiIndexer over
// it, with the in-memory shadow disabled and AI providers off.
func newCobolLifecycleEnv(t *testing.T, repos map[string]map[string]string) *cobolLifecycleEnv {
	t.Helper()
	t.Setenv("GORTEX_SHADOW_MAX_FILES", "0")
	t.Setenv("GORTEX_LLM_PROVIDER", "")
	dir := t.TempDir()
	env := &cobolLifecycleEnv{dir: dir, roots: map[string]string{}, repos: repos}
	for name, files := range repos {
		root := filepath.Join(dir, name)
		for rel, src := range files {
			writeRepoFile(t, root, rel, src)
		}
		env.roots[name] = root
	}

	// note: third occurrence of the store and indexer wiring (the
	// cmd/gortex COBOL tracer and breadth tests hold the other two); it
	// stays duplicated because those copies live in package main.
	store, err := store_sqlite.Open(filepath.Join(dir, "g.sqlite"))
	require.NoError(t, err)
	reg := parser.NewRegistry()
	languages.RegisterAll(reg)
	idx := New(store, reg, config.Default().Index, zap.NewNop())
	env.store = store
	env.cm, err = config.NewConfigManager(filepath.Join(dir, "config.yaml"))
	require.NoError(t, err)
	env.mi = NewMultiIndexer(store, reg, idx.Search(), env.cm, zap.NewNop())
	t.Cleanup(func() {
		_ = env.mi.Close(context.Background())
		_ = store.Close()
	})
	return env
}

// track tracks the named repositories in the given order, each pinned to
// workspace and project phase4.
func (env *cobolLifecycleEnv) track(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		res, err := env.mi.TrackRepoCtx(context.Background(), config.RepoEntry{
			Path: env.roots[name], Name: name, Workspace: "phase4", Project: "phase4",
		})
		require.NoError(t, err)
		require.Equal(t, name, res.RepoPrefix)
	}
}

// dump returns the canonical dump of every repository in env, after
// checking that a node of each reads back workspace and project phase4.
func (env *cobolLifecycleEnv) dump(t *testing.T) graphdump.Result {
	t.Helper()
	var names []string
	for name, files := range env.repos {
		names = append(names, name)
		var paths []string
		for rel := range files {
			paths = append(paths, rel)
		}
		sort.Strings(paths)
		n := env.store.GetNode(name + "/" + paths[0])
		require.NotNil(t, n, "file node of %s", name)
		require.Equal(t, "phase4", n.WorkspaceID)
		require.Equal(t, "phase4", n.ProjectID)
	}
	sort.Strings(names)
	res, err := graphdump.Dump(context.Background(), env.store,
		graph.ProjectionScope{Workspace: "phase4", Project: "phase4", Repositories: names},
		graphdump.DeterministicVolatile)
	require.NoError(t, err)
	require.NotEmpty(t, res.Rows)
	return res
}

// cleanCobolDump builds repos into a fresh store, tracking in order, and
// returns the dump.
func cleanCobolDump(t *testing.T, repos map[string]map[string]string, order ...string) graphdump.Result {
	t.Helper()
	env := newCobolLifecycleEnv(t, repos)
	env.track(t, order...)
	return env.dump(t)
}

// requireSameDump fails with the first differing rows when the digests of
// want and got differ.
func requireSameDump(t *testing.T, want, got graphdump.Result) {
	t.Helper()
	require.Equal(t, want.Digest, got.Digest, graphdump.FirstDiff(want.Rows, got.Rows, 20))
}

// TestCobolLifecycle_CleanDeterminism proves ID-06 for clean builds: the
// same corpus gives one dump across rebuilds and opposite track orders, and
// shared DB2, IDMS, and CICS nodes do not depend on their writer (D-18).
func TestCobolLifecycle_CleanDeterminism(t *testing.T) {
	want := cleanCobolDump(t, cobolLifecycleRepos, "lifea", "lifeb")
	dumps := []graphdump.Result{want}

	t.Run("rebuild", func(t *testing.T) {
		got := cleanCobolDump(t, cobolLifecycleRepos, "lifea", "lifeb")
		dumps = append(dumps, got)
		requireSameDump(t, want, got)
		assert.Equal(t, want.Rows, got.Rows)
		assert.Empty(t, graphdump.FirstDiff(want.Rows, got.Rows, 20))
	})

	t.Run("track_order", func(t *testing.T) {
		got := cleanCobolDump(t, cobolLifecycleRepos, "lifeb", "lifea")
		dumps = append(dumps, got)
		requireSameDump(t, want, got)
	})

	t.Run("shared_resources", func(t *testing.T) {
		type row struct {
			fields []string
			meta   map[string]any
		}
		shared := map[string]row{}
		var edges []row
		for _, line := range want.Rows {
			fields := strings.Split(line, "\x1f")
			var meta map[string]any
			require.NoError(t, json.Unmarshal([]byte(fields[len(fields)-1]), &meta), line)
			r := row{fields: fields, meta: meta}
			switch {
			case fields[0] == "E":
				edges = append(edges, r)
			case meta["prov_range_absence"] == "shared_resource" &&
				(fields[2] == string(graph.KindResource) || fields[2] == string(graph.KindTable)):
				shared[fields[1]] = r
			}
		}
		for _, id := range []string{
			"lifea/db2::MYSCH/ORDERS", "lifea/idms::_unqualified/record/EMP-REC",
			"lifea/cics::map/MSET1/MAP1", "lifea/cics::mapset/MSET1", "lifeb/db2::_unqualified/ORDERS",
		} {
			assert.Contains(t, shared, id)
		}
		for id, r := range shared {
			assert.Equal(t, id, r.fields[4], "FilePath of %s", id)
			assert.Equal(t, id[strings.LastIndex(id, "/")+1:], r.fields[3], "Name of %s", id)
			for _, key := range []string{
				"prov_source_path", "prov_source_id", "prov_revision_content_id",
				"prov_document_grade", "prov_document_grade_reasons", "prov_grade_policy",
				"prov_parse_config_id", "prov_transform_config_id", "prov_vcs_commit", "prov_vcs_commit_absence",
			} {
				assert.NotContains(t, r.meta, key, id)
			}
		}
		orders := shared["lifea/db2::MYSCH/ORDERS"]
		assert.Equal(t, "ORDERS", orders.meta["table"])
		assert.Equal(t, "MYSCH", orders.meta["schema"])

		raw := map[string][]string{}
		for _, e := range edges {
			to := e.fields[2]
			if _, ok := shared[to]; !ok {
				continue
			}
			assert.Contains(t, e.meta, "cobol_resource_raw", "edge %s -> %s", e.fields[1], to)
			if s, ok := e.meta["cobol_resource_raw"].(string); ok {
				raw[to] = append(raw[to], s)
			}
		}
		sort.Strings(raw["lifea/db2::MYSCH/ORDERS"])
		assert.Equal(t, []string{"MySch.Orders", "mysch.orders"}, raw["lifea/db2::MYSCH/ORDERS"])
	})

	t.Run("no_stash_keys", func(t *testing.T) {
		for _, d := range dumps {
			for _, line := range d.Rows {
				assert.NotContains(t, line, "restub_prev_")
			}
		}
	})

	// The allowlist must be versioned and every key justified (D-01).
	env := newCobolLifecycleEnv(t, cobolLifecycleRepos)
	scope := graph.ProjectionScope{Workspace: "phase4", Project: "phase4", Repositories: []string{"lifea"}}
	_, err := graphdump.Dump(context.Background(), env.store, scope, graphdump.Volatile{Keys: map[string]string{}})
	assert.Error(t, err)
	_, err = graphdump.Dump(context.Background(), env.store, scope,
		graphdump.Volatile{Version: "v", Keys: map[string]string{"k": ""}})
	assert.Error(t, err)
}
