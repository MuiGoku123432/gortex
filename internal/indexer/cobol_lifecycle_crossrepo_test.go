//go:build darwin || linux

package indexer

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/testutil/graphdump"
)

// The cross-repository corpus (D-04, D-09): invented fixed-format COBOL.
// crossa's MAINA calls XPGM (only crossb defines it), calls SUBPGM (both
// define it), and copies XCPY (only crossb holds it).
var cobolCrossRepoA = map[string]string{
	"cobol/main.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. MAINA.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY XCPY.
       PROCEDURE DIVISION.
       MAIN-PARA.
           CALL 'XPGM'.
           CALL 'SUBPGM'.
           GOBACK.
`,
	"cobol/sub.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SUBPGM.
       PROCEDURE DIVISION.
       SUB-PARA.
           GOBACK.
`,
}

var cobolCrossRepoB = map[string]string{
	"cobol/xpgm.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. XPGM.
       PROCEDURE DIVISION.
       X-PARA.
           GOBACK.
`,
	"cobol/sub.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SUBPGM.
       PROCEDURE DIVISION.
       SUB-PARA.
           GOBACK.
`,
	"copy/xcpy.cpy": `       01 X-REC.
          05 X-ID PIC X(8).
`,
}

// cobolCrossRepos returns the two-repository corpus with extra files merged
// into crossa (a .gortex.yaml is just another fixture entry).
func cobolCrossRepos(extraA map[string]string) map[string]map[string]string {
	a := maps.Clone(cobolCrossRepoA)
	maps.Copy(a, extraA)
	return map[string]map[string]string{"crossa": a, "crossb": cobolCrossRepoB}
}

const cobolCrossAllowlist = "index:\n  cobol:\n    resolve_from: [crossb]\n"

// cobolDumpEdge is one dumped edge row.
type cobolDumpEdge struct {
	from, to, kind string
	meta           map[string]any
}

// cobolDumpEdges parses the edge rows of a dump.
func cobolDumpEdges(t *testing.T, res graphdump.Result) []cobolDumpEdge {
	t.Helper()
	var out []cobolDumpEdge
	for _, line := range res.Rows {
		fields := strings.Split(line, "\x1f")
		if fields[0] != "E" {
			continue
		}
		var meta map[string]any
		require.NoError(t, json.Unmarshal([]byte(fields[len(fields)-1]), &meta), line)
		out = append(out, cobolDumpEdge{from: fields[1], to: fields[2], kind: fields[3], meta: meta})
	}
	return out
}

// cobolPlaceholderEdge returns the one edge from repo whose cobol_placeholder
// is placeholder.
func cobolPlaceholderEdge(t *testing.T, edges []cobolDumpEdge, repo, placeholder string) cobolDumpEdge {
	t.Helper()
	var found []cobolDumpEdge
	for _, e := range edges {
		if strings.HasPrefix(e.from, repo+"/") && e.meta["cobol_placeholder"] == placeholder {
			found = append(found, e)
		}
	}
	require.Len(t, found, 1, "edges from %s with placeholder %s", repo, placeholder)
	return found[0]
}

// requireCobolMissingNode asserts the dump holds the missing-artifact
// placeholder node id (D-11) with artifact_status not_obtained.
func requireCobolMissingNode(t *testing.T, res graphdump.Result, id string) {
	t.Helper()
	for _, line := range res.Rows {
		fields := strings.Split(line, "\x1f")
		if fields[0] != "N" || fields[1] != id {
			continue
		}
		var meta map[string]any
		require.NoError(t, json.Unmarshal([]byte(fields[len(fields)-1]), &meta), line)
		assert.Equal(t, "not_obtained", meta["artifact_status"], id)
		return
	}
	t.Fatalf("no placeholder node %s in the dump", id)
}

// requireCrossRepoUnbound asserts the same-repository-only outcome: XPGM and
// XCPY missing on crossa's placeholder nodes, SUBPGM bound to crossa's own
// program.
func requireCrossRepoUnbound(t *testing.T, res graphdump.Result) {
	t.Helper()
	edges := cobolDumpEdges(t, res)
	xpgm := cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::program::XPGM")
	assert.Equal(t, "crossa/cobol-missing::program/XPGM", xpgm.to)
	assert.Equal(t, "missing", xpgm.meta["cobol_unresolved_reason"])
	requireCobolMissingNode(t, res, xpgm.to)
	xcpy := cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::copybook::XCPY")
	assert.Equal(t, "crossa/cobol-missing::copybook/XCPY", xcpy.to)
	assert.Equal(t, "missing", xcpy.meta["cobol_unresolved_reason"])
	requireCobolMissingNode(t, res, xcpy.to)
	sub := cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::program::SUBPGM")
	assert.Equal(t, "crossa/cobol/sub.cbl::SUBPGM", sub.to)
	assert.NotContains(t, sub.meta, "cobol_unresolved_reason")
}

// TestCobolLifecycle_CrossRepoClean proves the cross-repository clean-build
// sequences of D-04 on the real track path: configured allowlists, library
// order, and default qualifiers from each repository's .gortex.yaml drive
// binding (D-09, D-10, D-11), an invalid block fails closed, opposite track
// orders give one dump (D-19), and a config refresh reaches the live indexer.
func TestCobolLifecycle_CrossRepoClean(t *testing.T) {
	t.Run("no_allowlist", func(t *testing.T) {
		requireCrossRepoUnbound(t, cleanCobolDump(t, cobolCrossRepos(nil), "crossa", "crossb"))
	})

	allowRepos := cobolCrossRepos(map[string]string{".gortex.yaml": cobolCrossAllowlist})
	allowed := cleanCobolDump(t, allowRepos, "crossa", "crossb")

	t.Run("allowlist", func(t *testing.T) {
		edges := cobolDumpEdges(t, allowed)
		xpgm := cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::program::XPGM")
		assert.Equal(t, "crossb/cobol/xpgm.cbl::XPGM", xpgm.to)
		assert.NotContains(t, xpgm.meta, "cobol_unresolved_reason")
		xcpy := cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::copybook::XCPY")
		assert.Equal(t, "crossb/copy/xcpy.cpy", xcpy.to)
		assert.NotContains(t, xcpy.meta, "cobol_unresolved_reason")
		sub := cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::program::SUBPGM")
		assert.Equal(t, "unresolved::cobol::program::SUBPGM", sub.to)
		assert.Equal(t, "ambiguous", sub.meta["cobol_unresolved_reason"])
		assert.Equal(t, []any{"crossa/cobol/sub.cbl::SUBPGM", "crossb/cobol/sub.cbl::SUBPGM"},
			sub.meta["cobol_candidate_ids"])
	})

	t.Run("track_order", func(t *testing.T) {
		requireSameDump(t, allowed, cleanCobolDump(t, allowRepos, "crossb", "crossa"))
	})

	t.Run("search_order", func(t *testing.T) {
		repos := cobolCrossRepos(map[string]string{
			".gortex.yaml": "index:\n  cobol:\n    copybook_libraries:\n      - path: copy/prod\n      - path: copy/test\n",
			"cobol/rec.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. RECPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CUSTREC.
       PROCEDURE DIVISION.
       REC-PARA.
           GOBACK.
`,
			"copy/prod/custrec.cpy": "       01 CUST-REC.\n          05 CUST-ID PIC X(8).\n",
			"copy/test/custrec.cpy": "       01 CUST-REC.\n          05 CUST-ID PIC X(9).\n",
		})
		edges := cobolDumpEdges(t, cleanCobolDump(t, repos, "crossa", "crossb"))
		cpy := cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::copybook::CUSTREC")
		assert.Equal(t, "crossa/copy/prod/custrec.cpy", cpy.to)
		assert.Equal(t, float64(0), cpy.meta["cobol_copy_search_rank"])
		assert.Equal(t, "cobol-copy-search/1", cpy.meta["cobol_copy_search_rule"])
	})

	t.Run("default_schema", func(t *testing.T) {
		repos := cobolCrossRepos(map[string]string{
			".gortex.yaml": "index:\n  cobol:\n    db2_default_schema: MYSCH\n",
			"cobol/sql.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SQLPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-X PIC X(10).
       PROCEDURE DIVISION.
       SQL-PARA.
           EXEC SQL SELECT A INTO :WS-X FROM ORDERS END-EXEC.
           EXEC SQL SELECT A INTO :WS-X FROM MYSCH.ORDERS END-EXEC.
           GOBACK.
`,
		})
		var aliases []cobolDumpEdge
		for _, e := range cobolDumpEdges(t, cleanCobolDump(t, repos, "crossa", "crossb")) {
			if e.kind == string(graph.EdgeAliases) && e.from == "crossa/db2::_unqualified/ORDERS" {
				aliases = append(aliases, e)
			}
		}
		require.Len(t, aliases, 1)
		assert.Equal(t, "crossa/db2::MYSCH/ORDERS", aliases[0].to)
		assert.Equal(t, "cobol-default-qualifier/1", aliases[0].meta["cobol_qualifier_rule"])
	})

	t.Run("invalid_config", func(t *testing.T) {
		repos := cobolCrossRepos(map[string]string{
			".gortex.yaml": "index:\n  cobol:\n    resolve_from: [\"*\"]\n",
		})
		requireCrossRepoUnbound(t, cleanCobolDump(t, repos, "crossa", "crossb"))
	})

	t.Run("refresh_pushes_config", func(t *testing.T) {
		env := newCobolLifecycleEnv(t, cobolCrossRepos(nil))
		env.track(t, "crossa", "crossb")
		idx := env.mi.GetIndexer("crossa")
		require.NotNil(t, idx)
		assert.Equal(t, config.CobolConfig{}, idx.cobolConfig())

		require.NoError(t, os.WriteFile(filepath.Join(env.roots["crossa"], ".gortex.yaml"),
			[]byte(cobolCrossAllowlist), 0o600))
		env.mi.RefreshRepoConfigs()
		assert.Equal(t, config.CobolConfig{ResolveFrom: []string{"crossb"}}, idx.cobolConfig())
	})
}

// The nested corpus (D-04): invented fixed-format COBOL. nestb's root lies
// inside lifea at nested/; neither program calls or copies the other.
var nestedCobolOuter = map[string]string{
	"cobol/outer.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OUTERPGM.
       PROCEDURE DIVISION.
       OUTER-PARA.
           GOBACK.
`,
}

var nestedCobolInner = map[string]string{
	"cobol/inner.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. INNERPGM.
       PROCEDURE DIVISION.
       INNER-PARA.
           GOBACK.
`,
}

// newNestedCobolEnv writes lifea and nests nestb's files under
// <lifea>/nested, registering nestb's root and corpus in env.
func newNestedCobolEnv(t *testing.T) *cobolLifecycleEnv {
	t.Helper()
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"lifea": nestedCobolOuter})
	root := filepath.Join(env.roots["lifea"], "nested")
	for rel, src := range nestedCobolInner {
		writeRepoFile(t, root, rel, src)
	}
	env.roots["nestb"] = root
	env.repos["nestb"] = nestedCobolInner
	return env
}

// TestCobolLifecycle_NestedTrackLater proves ID-09 for a nested topology:
// tracking the outer repository first, then the nested one, gives the same
// canonical graph as tracking the nested repository first (WR-02, SC-4).
func TestCobolLifecycle_NestedTrackLater(t *testing.T) {
	wantEnv := newNestedCobolEnv(t)
	wantEnv.track(t, "nestb", "lifea")
	want := wantEnv.dump(t)

	gotEnv := newNestedCobolEnv(t)
	gotEnv.track(t, "lifea", "nestb")
	requireSameDump(t, want, gotEnv.dump(t))
}

// TestCobolLifecycle_NestedUntrack proves ID-09 for an untrack: after the
// nested repository is untracked, the outer repository's graph equals a
// clean build of the outer repository holding the nested files (WR-01, SC-4).
func TestCobolLifecycle_NestedUntrack(t *testing.T) {
	merged := maps.Clone(nestedCobolOuter)
	for rel, src := range nestedCobolInner {
		merged["nested/"+rel] = src
	}
	single := map[string]map[string]string{"lifea": merged}
	want := cleanCobolDump(t, single, "lifea")

	env := newNestedCobolEnv(t)
	env.track(t, "nestb", "lifea")
	env.mi.UntrackRepo("nestb")
	env.repos = single
	requireSameDump(t, want, env.dump(t))
}
