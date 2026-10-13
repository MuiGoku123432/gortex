//go:build darwin || linux

package indexer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
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
	requireSameDump(t, cleanCobolDump(t, env.repos, "lifea", "lifeb"), got)
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
	assert.Equal(t, placeholder, copyEdge[2])
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
