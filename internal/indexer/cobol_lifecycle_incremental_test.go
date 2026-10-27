//go:build darwin || linux

package indexer

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

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

// The incremental half of the Phase 4 lifecycle harness (D-04): each test
// mutates the 04-01 corpus on disk, runs the real IncrementalReindexRepo
// path, and compares the dump with a clean build of the same final files.

// cobolLifecycleEdits are the edited bytes the sequences write: SUBPGM's
// paragraph renamed, and a second CALL added to MAINPGM.
var cobolLifecycleEdits = map[string]string{
	"cobol/sub.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SUBPGM.
       PROCEDURE DIVISION.
       SUB-LOOP.
           DISPLAY 'X'.
           GOBACK.
`,
	"cobol/main.cbl": strings.Replace(cobolLifecycleRepoA["cobol/main.cbl"],
		"           CALL 'SUBPGM'.\n",
		"           CALL 'SUBPGM'.\n           CALL 'OTHERPGM'.\n", 1),
}

// newCobolIncrementalEnv tracks a private copy of the two-repository
// corpus, so the sequences can mutate env.repos to mirror the disk.
func newCobolIncrementalEnv(t *testing.T) *cobolLifecycleEnv {
	t.Helper()
	repos := make(map[string]map[string]string, len(cobolLifecycleRepos))
	for name, files := range cobolLifecycleRepos {
		repos[name] = make(map[string]string, len(files))
		for rel, src := range files {
			repos[name][rel] = src
		}
	}
	env := newCobolLifecycleEnv(t, repos)
	env.track(t, "lifea", "lifeb")
	return env
}

// write replaces one file on disk and in env.repos.
func (env *cobolLifecycleEnv) write(t *testing.T, repo, rel, src string) {
	t.Helper()
	bumpMtime(t, filepath.Join(env.roots[repo], rel), src)
	env.repos[repo][rel] = src
}

// remove deletes one file from disk and from env.repos.
func (env *cobolLifecycleEnv) remove(t *testing.T, repo, rel string) {
	t.Helper()
	require.NoError(t, os.Remove(filepath.Join(env.roots[repo], rel)))
	delete(env.repos[repo], rel)
}

// reindex runs the coordinated incremental reindex of one repository.
func (env *cobolLifecycleEnv) reindex(t *testing.T, repo string) {
	t.Helper()
	_, err := env.mi.IncrementalReindexRepo(repo, nil)
	require.NoError(t, err)
}

// requireCleanEquivalent dumps env and requires it to equal a clean build
// of env's current files, with no restub stash key on any row.
func (env *cobolLifecycleEnv) requireCleanEquivalent(t *testing.T) graphdump.Result {
	t.Helper()
	got := env.dump(t)
	requireSameDump(t, cleanCobolDump(t, env.repos, slices.Sorted(maps.Keys(env.repos))...), got)
	for _, line := range got.Rows {
		require.NotContains(t, line, "restub_prev_")
	}
	return got
}

// TestCobolLifecycle_TailsReached is the tail spike kept as a regression
// test: a COBOL save and a COBOL delete through IncrementalReindexRepo each
// reach the COBOL reconcile tail exactly once.
func TestCobolLifecycle_TailsReached(t *testing.T) {
	env := newCobolIncrementalEnv(t)
	idx := env.mi.GetIndexer("lifea")
	require.NotNil(t, idx)
	var mu sync.Mutex
	var kinds []string
	idx.incrementalCatchupHook = func(kind string, _ []string) {
		mu.Lock()
		defer mu.Unlock()
		kinds = append(kinds, kind)
	}
	take := func() []string {
		mu.Lock()
		defer mu.Unlock()
		out := kinds
		kinds = nil
		return out
	}
	count := func(list []string, kind string) int {
		n := 0
		for _, k := range list {
			if k == kind {
				n++
			}
		}
		return n
	}

	env.write(t, "lifea", "cobol/main.cbl", cobolLifecycleEdits["cobol/main.cbl"])
	env.reindex(t, "lifea")
	save := take()
	t.Logf("tails after a .cbl save: %v", save)
	assert.Equal(t, 1, count(save, "cobol"), "save tails %v", save)

	env.remove(t, "lifea", "copy/custrec.cpy")
	env.reindex(t, "lifea")
	del := take()
	t.Logf("tails after a .cpy delete: %v", del)
	assert.Equal(t, 1, count(del, "cobol"), "delete tails %v", del)
}

// TestCobolLifecycle_IdenticalRerun proves an unchanged incremental reindex
// and a repeated reconcile leave the clean-build dump unchanged (ID-07).
func TestCobolLifecycle_IdenticalRerun(t *testing.T) {
	env := newCobolIncrementalEnv(t)
	want := env.requireCleanEquivalent(t)
	for i := 0; i < 2; i++ {
		env.reindex(t, "lifea")
		requireSameDump(t, want, env.requireCleanEquivalent(t))
	}
	for i := 0; i < 2; i++ {
		env.mi.reconcileCobolRepos(map[string]struct{}{"lifea": {}, "lifeb": {}})
		requireSameDump(t, want, env.dump(t))
	}
}

// TestCobolLifecycle_Edit edits the bound target and adds a CALL; the
// incremental dump equals a clean build of the edited files.
func TestCobolLifecycle_Edit(t *testing.T) {
	env := newCobolIncrementalEnv(t)
	for _, rel := range []string{"cobol/sub.cbl", "cobol/main.cbl"} {
		env.write(t, "lifea", rel, cobolLifecycleEdits[rel])
	}
	env.reindex(t, "lifea")
	env.requireCleanEquivalent(t)
}

// TestCobolLifecycle_Delete deletes a copybook (its COPY edge returns to the
// placeholder as missing) and the last reference to a shared table (the
// table node is evicted); each dump equals a clean build without the file.
func TestCobolLifecycle_Delete(t *testing.T) {
	env := newCobolIncrementalEnv(t)

	env.remove(t, "lifea", "copy/custrec.cpy")
	env.reindex(t, "lifea")
	got := env.requireCleanEquivalent(t)
	placeholder := graph.UnresolvedMarker + "cobol::copybook::CUSTREC"
	var copyEdge []string
	for _, line := range got.Rows {
		fields := strings.Split(line, "\x1f")
		if fields[0] == "E" && strings.HasPrefix(fields[1], "lifea/cobol/main.cbl") &&
			strings.Contains(fields[6], `"cobol_placeholder":"`+placeholder+`"`) {
			copyEdge = fields
		}
	}
	require.NotNil(t, copyEdge, "COPY CUSTREC edge of main.cbl")
	assert.Equal(t, "lifea::cobol-missing::copybook/CUSTREC", copyEdge[2])
	requireCobolMissingNode(t, got, copyEdge[2])
	var meta map[string]any
	require.NoError(t, json.Unmarshal([]byte(copyEdge[6]), &meta))
	assert.Equal(t, "missing", meta["cobol_unresolved_reason"])

	const table = "lifeb/db2::_unqualified/ORDERS"
	require.NotNil(t, env.store.GetNode(table))
	env.remove(t, "lifeb", "cobol/caller.cbl")
	env.reindex(t, "lifeb")
	assert.Nil(t, env.store.GetNode(table))
	env.requireCleanEquivalent(t)
}

// TestCobolLifecycle_Restore restores the original bytes of the edited and
// deleted files; the dump is byte-identical to the original clean build.
func TestCobolLifecycle_Restore(t *testing.T) {
	env := newCobolIncrementalEnv(t)
	want := env.dump(t)

	for _, rel := range []string{"cobol/sub.cbl", "cobol/main.cbl"} {
		env.write(t, "lifea", rel, cobolLifecycleEdits[rel])
	}
	env.remove(t, "lifea", "copy/custrec.cpy")
	env.reindex(t, "lifea")
	env.requireCleanEquivalent(t)

	for _, rel := range []string{"cobol/sub.cbl", "cobol/main.cbl", "copy/custrec.cpy"} {
		env.write(t, "lifea", rel, cobolLifecycleRepoA[rel])
	}
	env.reindex(t, "lifea")
	got := env.requireCleanEquivalent(t)
	requireSameDump(t, want, got)
	assert.Equal(t, want.Rows, got.Rows)
}

// TestCobolLifecycle_CrossFile changes another file of lifea; the unchanged
// CALL 'SUBPGM' in main.cbl is decided again, and each step equals its
// clean build (D-04).
func TestCobolLifecycle_CrossFile(t *testing.T) {
	const callSub = "unresolved::cobol::program::SUBPGM"
	callEdge := func(t *testing.T, res graphdump.Result) cobolDumpEdge {
		t.Helper()
		return cobolPlaceholderEdge(t, cobolDumpEdges(t, res), "lifea", callSub)
	}

	t.Run("new_candidate", func(t *testing.T) {
		env := newCobolIncrementalEnv(t)
		bound := callEdge(t, env.dump(t)).to
		require.Equal(t, "lifea/cobol/sub.cbl::SUBPGM", bound)

		env.write(t, "lifea", "cobol/sub2.cbl", cobolLifecycleRepoA["cobol/sub.cbl"])
		env.reindex(t, "lifea")
		call := callEdge(t, env.requireCleanEquivalent(t))
		assert.Equal(t, callSub, call.to)
		assert.Equal(t, "ambiguous", call.meta["cobol_unresolved_reason"])

		env.remove(t, "lifea", "cobol/sub2.cbl")
		env.reindex(t, "lifea")
		assert.Equal(t, bound, callEdge(t, env.requireCleanEquivalent(t)).to)
	})

	t.Run("target_edited", func(t *testing.T) {
		env := newCobolIncrementalEnv(t)
		bound := callEdge(t, env.dump(t)).to

		renamed := strings.Replace(cobolLifecycleRepoA["cobol/sub.cbl"],
			"PROGRAM-ID. SUBPGM.", "PROGRAM-ID. SUBPGX.", 1)
		env.write(t, "lifea", "cobol/sub.cbl", renamed)
		env.reindex(t, "lifea")
		res := env.requireCleanEquivalent(t)
		call := callEdge(t, res)
		assert.Equal(t, "lifea::cobol-missing::program/SUBPGM", call.to)
		assert.Equal(t, "missing", call.meta["cobol_unresolved_reason"])
		requireCobolMissingNode(t, res, call.to)

		env.write(t, "lifea", "cobol/sub.cbl", cobolLifecycleRepoA["cobol/sub.cbl"])
		env.reindex(t, "lifea")
		assert.Equal(t, bound, callEdge(t, env.requireCleanEquivalent(t)).to)
	})
}

// TestCobolLifecycle_CrossRepoIncremental adds and deletes XPGM in crossb
// through an incremental reindex of crossb alone; crossa's unchanged CALL,
// allowed by its resolve_from, binds and returns to missing (D-09).
func TestCobolLifecycle_CrossRepoIncremental(t *testing.T) {
	const callX = "unresolved::cobol::program::XPGM"
	const missingX = "crossa::cobol-missing::program/XPGM"
	repos := cobolCrossRepos(map[string]string{".gortex.yaml": cobolCrossAllowlist})
	repos["crossb"] = maps.Clone(repos["crossb"])
	xpgm := repos["crossb"]["cobol/xpgm.cbl"]
	delete(repos["crossb"], "cobol/xpgm.cbl")
	env := newCobolLifecycleEnv(t, repos)
	env.track(t, "crossa", "crossb")
	var res graphdump.Result
	callEdge := func() cobolDumpEdge {
		res = env.requireCleanEquivalent(t)
		return cobolPlaceholderEdge(t, cobolDumpEdges(t, res), "crossa", callX)
	}
	call := callEdge()
	require.Equal(t, missingX, call.to)
	require.Equal(t, "missing", call.meta["cobol_unresolved_reason"])
	requireCobolMissingNode(t, res, missingX)

	env.write(t, "crossb", "cobol/xpgm.cbl", xpgm)
	env.reindex(t, "crossb")
	call = callEdge()
	assert.Equal(t, "crossb/cobol/xpgm.cbl::XPGM", call.to)
	assert.NotContains(t, call.meta, "cobol_unresolved_reason")

	env.remove(t, "crossb", "cobol/xpgm.cbl")
	env.reindex(t, "crossb")
	call = callEdge()
	assert.Equal(t, missingX, call.to)
	assert.Equal(t, "missing", call.meta["cobol_unresolved_reason"])
	requireCobolMissingNode(t, res, missingX)
}

// TestCobolLifecycle_CrossRepoUntrack binds crossa's CALL and COPY into
// crossb through its resolve_from, then untracks crossb: crossa's edges go
// back to their placeholders as missing, the result equals a clean build of
// crossa alone (ID-05, ID-09), and re-tracking crossb binds them again.
func TestCobolLifecycle_CrossRepoUntrack(t *testing.T) {
	repos := cobolCrossRepos(map[string]string{".gortex.yaml": cobolCrossAllowlist})
	env := newCobolLifecycleEnv(t, repos)
	env.track(t, "crossa", "crossb")
	edges := cobolDumpEdges(t, env.requireCleanEquivalent(t))
	require.Equal(t, "crossb/cobol/xpgm.cbl::XPGM",
		cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::program::XPGM").to)
	require.Equal(t, "crossb/copy/xcpy.cpy",
		cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::copybook::XCPY").to)

	_, _, err := env.mi.UntrackRepoChecked(context.Background(), "crossb")
	require.NoError(t, err)
	crossb := env.repos["crossb"]
	delete(env.repos, "crossb")
	res := env.requireCleanEquivalent(t)
	edges = cobolDumpEdges(t, res)
	for placeholder, missing := range map[string]string{
		"unresolved::cobol::program::XPGM":  "crossa::cobol-missing::program/XPGM",
		"unresolved::cobol::copybook::XCPY": "crossa::cobol-missing::copybook/XCPY",
	} {
		e := cobolPlaceholderEdge(t, edges, "crossa", placeholder)
		assert.Equal(t, missing, e.to)
		assert.Equal(t, "missing", e.meta["cobol_unresolved_reason"])
		requireCobolMissingNode(t, res, missing)
	}

	env.repos["crossb"] = crossb
	env.track(t, "crossb")
	edges = cobolDumpEdges(t, env.requireCleanEquivalent(t))
	assert.Equal(t, "crossb/cobol/xpgm.cbl::XPGM",
		cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::program::XPGM").to)
}

// TestCobolLifecycle_Interrupted interrupts a reindex after parse, evict, and
// AddBatch but before its resolver tail: the store holds placeholders, never
// a wrong binding, and a warm restart over the same SQLite file converges to
// the clean build of the edited files (ID-07).
func TestCobolLifecycle_Interrupted(t *testing.T) {
	env := newCobolIncrementalEnv(t)
	idx := env.mi.GetIndexer("lifea")
	require.NotNil(t, idx)
	idx.incrementalResolveFilesHook = func([]string) {}

	env.write(t, "lifea", "cobol/main.cbl", cobolLifecycleEdits["cobol/main.cbl"])
	env.reindex(t, "lifea")

	clean := cleanCobolDump(t, env.repos, "lifea", "lifeb")
	got := env.dump(t)
	require.NotEqual(t, clean.Digest, got.Digest, "the interrupted reindex left no placeholder")
	bound := map[string]string{}
	for _, e := range cobolDumpEdges(t, clean) {
		if p, _ := e.meta["cobol_placeholder"].(string); p != "" {
			bound[e.from+"\x00"+p] = e.to
		}
	}
	for _, e := range cobolDumpEdges(t, got) {
		p, _ := e.meta["cobol_placeholder"].(string)
		if p == "" || e.to == p {
			continue
		}
		assert.Equal(t, bound[e.from+"\x00"+p], e.to, "binding of %s %s", e.from, p)
	}

	// Daemon restart: reconcile each repository from its persisted mtimes.
	env.restart(t)
	for _, name := range []string{"lifea", "lifeb"} {
		env.warmReconcile(t, name)
	}
	requireSameDump(t, clean, env.requireCleanEquivalent(t))
}

// restart drops env's stack and opens a fresh one over the same SQLite file,
// as a daemon restart does.
func (env *cobolLifecycleEnv) restart(t *testing.T) {
	t.Helper()
	require.NoError(t, env.mi.Close(context.Background()))
	require.NoError(t, env.store.Close())
	// note: the same store and MultiIndexer wiring as newCobolLifecycleEnv
	// (two occurrences in this package).
	store, err := store_sqlite.Open(filepath.Join(env.dir, "g.sqlite"))
	require.NoError(t, err)
	reg := parser.NewRegistry()
	languages.RegisterAll(reg)
	cm, err := config.NewConfigManager(filepath.Join(env.dir, "config.yaml"))
	require.NoError(t, err)
	mi := NewMultiIndexer(store, reg, New(store, reg, config.Default().Index, zap.NewNop()).Search(), cm, zap.NewNop())
	t.Cleanup(func() {
		_ = mi.Close(context.Background())
		_ = store.Close()
	})
	env.store, env.cm, env.mi = store, cm, mi
}

// warmReconcile reconciles one repository from its persisted mtimes, the
// daemon's warm-restart route.
func (env *cobolLifecycleEnv) warmReconcile(t *testing.T, name string) {
	t.Helper()
	prior := env.store.LoadFileMtimes(name)
	require.NotEmpty(t, prior, name)
	_, err := env.mi.ReconcileRepoCtx(context.Background(), config.RepoEntry{
		Path: env.roots[name], Name: name, Workspace: "phase4", Project: "phase4",
	}, prior)
	require.NoError(t, err)
}

// TestCobolLifecycle_ParallelWarmRestart restarts the cross-repository store
// under a parallel warmup batch and reconciles crossa before crossb is
// registered. The COBOL reconcile waits for the batch end (WR-01): crossa's
// binding into crossb is not rewritten to missing while crossb is merely
// unregistered, and the batch end leaves the clean build.
func TestCobolLifecycle_ParallelWarmRestart(t *testing.T) {
	const callX = "unresolved::cobol::program::XPGM"
	env := newCobolLifecycleEnv(t, cobolCrossRepos(map[string]string{".gortex.yaml": cobolCrossAllowlist}))
	env.track(t, "crossa", "crossb")
	env.restart(t)

	env.mi.BeginParallelBatch()
	env.warmReconcile(t, "crossa")
	call := cobolPlaceholderEdge(t, cobolDumpEdges(t, env.dump(t)), "crossa", callX)
	assert.Equal(t, "crossb/cobol/xpgm.cbl::XPGM", call.to, "decided before crossb was registered")
	env.warmReconcile(t, "crossb")
	env.mi.ResetBatch()

	call = cobolPlaceholderEdge(t, cobolDumpEdges(t, env.requireCleanEquivalent(t)), "crossa", callX)
	assert.Equal(t, "crossb/cobol/xpgm.cbl::XPGM", call.to)
}

// TestCobolLifecycle_OverCap caps lifea's affected-by fan-out at one file.
// Three programs include CUSTREC and call SUBPGM. Editing the copybook's
// member layout never reaches the affected-by pass (a program references a
// copybook only through its imports edge, which the COBOL reconcile owns);
// renaming SUBPGM removes a symbol three files reference, which exceeds the
// cap. Either way the pass must not truncate (D-08) and each dump equals the
// clean build of the edited files.
func TestCobolLifecycle_OverCap(t *testing.T) {
	repos := make(map[string]map[string]string, len(cobolLifecycleRepos))
	for name, files := range cobolLifecycleRepos {
		repos[name] = maps.Clone(files)
	}
	repos["lifea"][".gortex.yaml"] = "index:\n  affected_by_reresolve_max: 1\n"
	for _, name := range []string{"PROGB", "PROGC"} {
		repos["lifea"]["cobol/"+strings.ToLower(name)+".cbl"] = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. ` + name + `.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CUSTREC.
       PROCEDURE DIVISION.
       ` + name + `-PARA.
           CALL 'SUBPGM'.
           GOBACK.
`
	}
	env := newCobolLifecycleEnv(t, repos)
	env.track(t, "lifea", "lifeb")
	idx := env.mi.GetIndexer("lifea")
	require.NotNil(t, idx)
	var mu sync.Mutex
	var affected []string
	idx.incrementalCatchupHook = func(kind string, files []string) {
		if kind != "affected_by" {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		affected = append(affected, files...)
	}

	env.write(t, "lifea", "copy/custrec.cpy", `       01 CUST-REC.
          05 CUST-KEY PIC X(8).
          05 CUST-ID PIC X(12).
`)
	env.reindex(t, "lifea")
	env.requireCleanEquivalent(t)

	env.write(t, "lifea", "cobol/sub.cbl", strings.Replace(cobolLifecycleRepoA["cobol/sub.cbl"],
		"PROGRAM-ID. SUBPGM.", "PROGRAM-ID. SUBPGX.", 1))
	env.reindex(t, "lifea")
	env.requireCleanEquivalent(t)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"lifea/cobol/main.cbl", "lifea/cobol/progb.cbl", "lifea/cobol/progc.cbl"},
		appendUniqueSorted(nil, affected...), "the affected set above the cap of 1 must not be truncated")
}

// TestCobolLifecycle_ReconcileSerialized proves every COBOL reconcile pass a
// MultiIndexer drives, through a master resolver or a per-repository one,
// takes the shared cobolReconcileMu, so two passes never decide the same
// edges from interleaved snapshots (WR-01).
func TestCobolLifecycle_ReconcileSerialized(t *testing.T) {
	env := newCobolLifecycleEnv(t, cobolCrossRepos(map[string]string{".gortex.yaml": cobolCrossAllowlist}))
	env.track(t, "crossa", "crossb")
	idx := env.mi.GetIndexer("crossa")
	require.NotNil(t, idx)
	crossa := map[string]struct{}{"crossa": {}}

	for name, pass := range map[string]func(){
		"master":         func() { env.mi.reconcileCobolRepos(crossa) },
		"per_repository": func() { idx.resolver.ReconcileCobol(crossa) },
	} {
		t.Run(name, func(t *testing.T) {
			env.mi.cobolReconcileMu.Lock()
			done := make(chan struct{})
			go func() {
				defer close(done)
				pass()
			}()
			select {
			case <-done:
				env.mi.cobolReconcileMu.Unlock()
				t.Fatal("the reconcile pass ran while another pass held the lock")
			case <-time.After(200 * time.Millisecond):
			}
			env.mi.cobolReconcileMu.Unlock()
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				t.Fatal("the reconcile pass did not finish after the lock was released")
			}
		})
	}
}
