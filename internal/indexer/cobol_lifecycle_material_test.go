//go:build darwin || linux

package indexer

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/indexer/merkle"
)

// The material half of the Phase 4 lifecycle harness (D-04, D-06): an
// index.cobol edit with unchanged COBOL bytes must restage the COBOL files
// and converge to a clean build with the new config, and an edit to an
// unrelated config key must restage none of them.

// cobolMaterialSQL reads ORDERS unqualified and as MYSCH.ORDERS, so
// db2_default_schema MYSCH adds an aliases edge between them.
// note: second copy of the default_schema fixture in
// cobol_lifecycle_crossrepo_test.go; it stays duplicated (rule of three).
const cobolMaterialSQL = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SQLPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-X PIC X(10).
       PROCEDURE DIVISION.
       SQL-PARA.
           EXEC SQL SELECT A INTO :WS-X FROM ORDERS END-EXEC.
           EXEC SQL SELECT A INTO :WS-X FROM MYSCH.ORDERS END-EXEC.
           GOBACK.
`

// reconfigure writes repo's .gortex.yaml, pushes it into the live indexers,
// and runs a full-root incremental reindex of repo.
func (env *cobolLifecycleEnv) reconfigure(t *testing.T, repo, yaml string) *IndexResult {
	t.Helper()
	env.write(t, repo, ".gortex.yaml", yaml)
	env.mi.RefreshRepoConfigs()
	res, err := env.mi.IncrementalReindexRepo(repo, nil)
	require.NoError(t, err)
	return res
}

// cobolFileCount returns how many of repo's files are COBOL.
func (env *cobolLifecycleEnv) cobolFileCount(repo string) int {
	n := 0
	for rel := range env.repos[repo] {
		if ExtractorLangForFile(rel) == "cobol" {
			n++
		}
	}
	return n
}

// storedCobolMaterial returns the _cobol_material value persisted on repo's
// index state row, and whether the row carries it.
func (env *cobolLifecycleEnv) storedCobolMaterial(t *testing.T, repo string) (int, bool) {
	t.Helper()
	st, found, err := env.store.GetRepoIndexState(repo)
	require.NoError(t, err)
	require.True(t, found, "index state of %s", repo)
	var versions map[string]int
	require.NoError(t, json.Unmarshal([]byte(st.ExtractorVersions), &versions))
	v, ok := versions[cobolMaterialKey]
	return v, ok
}

// TestCobolLifecycle_ConfigChange proves D-06 for config changes: an
// index.cobol edit with unchanged COBOL bytes, pushed by RefreshRepoConfigs,
// restages every COBOL file on the next full-root reindex and yields the
// clean build of the new config; a clean pass re-stamps the fingerprint so
// the restage happens once.
func TestCobolLifecycle_ConfigChange(t *testing.T) {
	t.Run("default_schema", func(t *testing.T) {
		env := newCobolLifecycleEnv(t, cobolCrossRepos(map[string]string{"cobol/sql.cbl": cobolMaterialSQL}))
		env.track(t, "crossa", "crossb")

		res := env.reconfigure(t, "crossa", "index:\n  cobol:\n    db2_default_schema: MYSCH\n")
		assert.GreaterOrEqual(t, res.StaleFileCount, env.cobolFileCount("crossa"))
		got := env.requireCleanEquivalent(t)
		var aliases []cobolDumpEdge
		for _, e := range cobolDumpEdges(t, got) {
			if e.kind == string(graph.EdgeAliases) && e.from == "crossa/db2::_unqualified/ORDERS" {
				aliases = append(aliases, e)
			}
		}
		require.Len(t, aliases, 1)
		assert.Equal(t, "crossa/db2::MYSCH/ORDERS", aliases[0].to)
	})

	env := newCobolLifecycleEnv(t, cobolCrossRepos(nil))
	env.track(t, "crossa", "crossb")

	t.Run("allowlist", func(t *testing.T) {
		res := env.reconfigure(t, "crossa", cobolCrossAllowlist)
		assert.GreaterOrEqual(t, res.StaleFileCount, env.cobolFileCount("crossa"))
		got := env.requireCleanEquivalent(t)
		xpgm := cobolPlaceholderEdge(t, cobolDumpEdges(t, got), "crossa", "unresolved::cobol::program::XPGM")
		assert.Equal(t, "crossb/cobol/xpgm.cbl::XPGM", xpgm.to)
	})

	t.Run("restamped", func(t *testing.T) {
		idx := env.mi.GetIndexer("crossa")
		require.NotNil(t, idx)
		current, ok := idx.cobolMaterial()
		require.True(t, ok, "COBOL material fingerprint available")
		stored, ok := env.storedCobolMaterial(t, "crossa")
		require.True(t, ok, "stored %s", cobolMaterialKey)
		assert.Equal(t, int(current), stored)

		res, err := env.mi.IncrementalReindexRepo("crossa", nil)
		require.NoError(t, err)
		assert.Zero(t, res.StaleFileCount)
	})
}

// TestCobolLifecycle_ConfigReloadRedecides proves a config reload alone, with
// no reindex, re-decides the bindings of a repository whose index.cobol
// fingerprint changed (WR-03): crossa's new resolve_from binds its CALL and
// COPY into crossb, and dropping it returns them to missing.
func TestCobolLifecycle_ConfigReloadRedecides(t *testing.T) {
	const callX = "unresolved::cobol::program::XPGM"
	const missingX = "crossa/cobol-missing::program/XPGM"
	env := newCobolLifecycleEnv(t, cobolCrossRepos(nil))
	env.track(t, "crossa", "crossb")
	res := env.dump(t)
	call := cobolPlaceholderEdge(t, cobolDumpEdges(t, res), "crossa", callX)
	require.Equal(t, missingX, call.to)
	requireCobolMissingNode(t, res, missingX)

	// Written straight to disk: the reload, not a reindex, must re-decide.
	cfgPath := filepath.Join(env.roots["crossa"], ".gortex.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(cobolCrossAllowlist), 0o600))
	env.mi.RefreshRepoConfigs()
	edges := cobolDumpEdges(t, env.dump(t))
	assert.Equal(t, "crossb/cobol/xpgm.cbl::XPGM", cobolPlaceholderEdge(t, edges, "crossa", callX).to)
	assert.Equal(t, "crossb/copy/xcpy.cpy",
		cobolPlaceholderEdge(t, edges, "crossa", "unresolved::cobol::copybook::XCPY").to)

	require.NoError(t, os.WriteFile(cfgPath, []byte("index:\n  cobol: {}\n"), 0o600))
	env.mi.RefreshRepoConfigs()
	res = env.dump(t)
	call = cobolPlaceholderEdge(t, cobolDumpEdges(t, res), "crossa", callX)
	assert.Equal(t, missingX, call.to)
	assert.Equal(t, "missing", call.meta["cobol_unresolved_reason"])
	requireCobolMissingNode(t, res, missingX)
}

// TestCobolLifecycle_UnrelatedConfig proves the fingerprint reads only the
// index.cobol block (D-06): changing another index key restages no COBOL
// file and leaves the stored fingerprint as it was.
func TestCobolLifecycle_UnrelatedConfig(t *testing.T) {
	env := newCobolLifecycleEnv(t, cobolCrossRepos(nil))
	env.track(t, "crossa", "crossb")
	before, ok := env.storedCobolMaterial(t, "crossa")
	require.True(t, ok, "stored %s after track", cobolMaterialKey)

	res := env.reconfigure(t, "crossa", "index:\n  affected_by_reresolve_max: 50\n")
	// Only .gortex.yaml itself changed on disk.
	assert.Equal(t, 1, res.StaleFileCount)
	after, ok := env.storedCobolMaterial(t, "crossa")
	require.True(t, ok)
	assert.Equal(t, before, after)
	env.requireCleanEquivalent(t)
}

// TestCobolLifecycle_VersionChange proves D-07 for an extractor or parser
// identity change. That identity is pinned at compile time, so the test
// plays the store an older binary left behind: a different stored
// _cobol_material (and, in Merkle mode, a different fingerprint in every
// COBOL leaf salt) plus one program node carrying the previous
// prov_extractor_version. One full-root reindex must restage exactly the
// COBOL files, restore the current values, re-stamp the mode's record of
// the fingerprint, and equal the clean build; a second pass restages
// nothing.
func TestCobolLifecycle_VersionChange(t *testing.T) {
	for _, mode := range []string{"mtime", "merkle"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "merkle" {
				t.Setenv("GORTEX_MERKLE", "1")
			}
			repos := map[string]map[string]string{
				"lifea": maps.Clone(cobolLifecycleRepoA),
				"lifeb": maps.Clone(cobolLifecycleRepoB),
			}
			env := newCobolLifecycleEnv(t, repos)
			env.track(t, "lifea", "lifeb")
			idx := env.mi.GetIndexer("lifea")
			require.NotNil(t, idx)
			require.Equal(t, mode == "merkle", idx.merkleEnabled())
			current, ok := idx.cobolMaterial()
			require.True(t, ok, "COBOL material fingerprint available")
			older := current + 1

			st, found, err := env.store.GetRepoIndexState("lifea")
			require.NoError(t, err)
			require.True(t, found)
			versions := decodeExtractorVersions(st.ExtractorVersions)
			versions[cobolMaterialKey] = int(older)
			encoded, err := json.Marshal(versions)
			require.NoError(t, err)
			st.ExtractorVersions = string(encoded)
			require.NoError(t, env.store.SetRepoIndexState(st))

			treePath := merkleTreeFile(env.roots["lifea"])
			if mode == "merkle" {
				tree, err := merkle.Load(treePath)
				require.NoError(t, err)
				require.NotNil(t, tree)
				for rel, leaf := range tree.Files {
					if ExtractorLangForFile(rel) != "cobol" {
						continue
					}
					want := "|cobol-material@" + strconv.FormatInt(current, 10)
					require.Contains(t, leaf.Salt, want, "salt of %s", rel)
					leaf.Salt = strings.Replace(leaf.Salt, want, "|cobol-material@"+strconv.FormatInt(older, 10), 1)
					tree.Files[rel] = leaf
				}
				require.NoError(t, merkle.MergeScope(nil, tree, nil).Save(treePath))
			}

			const programID = "lifea/cobol/main.cbl::MAINPGM"
			program := env.store.GetNode(programID)
			require.NotNil(t, program)
			currentVersion, _ := program.Meta["prov_extractor_version"].(string)
			prefix, number, ok := strings.Cut(currentVersion, "/")
			require.True(t, ok, "extractor version %q", currentVersion)
			n, err := strconv.Atoi(number)
			require.NoError(t, err)
			stale := *program
			stale.Meta = maps.Clone(program.Meta)
			stale.Meta["prov_extractor_version"] = prefix + "/" + strconv.Itoa(n-1)
			env.store.AddBatch([]*graph.Node{&stale}, nil)

			res, err := env.mi.IncrementalReindexRepo("lifea", nil)
			require.NoError(t, err)
			assert.Equal(t, env.cobolFileCount("lifea"), res.StaleFileCount)
			restored := env.store.GetNode(programID)
			require.NotNil(t, restored)
			assert.Equal(t, currentVersion, restored.Meta["prov_extractor_version"])
			// Each mode re-stamps its own record: the row on the mtime path,
			// the leaf salt in Merkle mode (which never reads the row).
			if mode == "merkle" {
				tree, err := merkle.Load(treePath)
				require.NoError(t, err)
				assert.Contains(t, tree.Files["cobol/main.cbl"].Salt, "|cobol-material@"+strconv.FormatInt(current, 10))
			} else {
				stored, ok := env.storedCobolMaterial(t, "lifea")
				require.True(t, ok)
				assert.Equal(t, int(current), stored)
			}
			env.requireCleanEquivalent(t)
			again, err := env.mi.IncrementalReindexRepo("lifea", nil)
			require.NoError(t, err)
			assert.Zero(t, again.StaleFileCount, "the restage happens once")
		})
	}
}

// newCobolGitEnv tracks lifea as a git repository whose one commit holds the
// corpus; edits are written to the working tree after that commit, before
// tracking, so they start out uncommitted.
func newCobolGitEnv(t *testing.T, edits map[string]string) *cobolLifecycleEnv {
	t.Helper()
	requireGit(t)
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"lifea": maps.Clone(cobolLifecycleRepoA)})
	root := env.roots["lifea"]
	gitInitRepo(t, root)
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "init")
	for rel, src := range edits {
		env.write(t, "lifea", rel, src)
	}
	env.track(t, "lifea")
	return env
}

// headMoveTail runs what the git watcher and the poller run after HEAD
// moves: lifea's reconcileRepoIndexState.
func (env *cobolLifecycleEnv) headMoveTail(t *testing.T) {
	t.Helper()
	idx := env.mi.GetIndexer("lifea")
	require.NotNil(t, idx)
	idx.reconcileRepoIndexState(context.Background(), idx.RootPath())
}

// requireRootCleanEquivalent requires env's dump to equal a clean build, in
// a fresh store, of the same repository roots, so the clean build sees the
// same git HEAD and working tree.
func (env *cobolLifecycleEnv) requireRootCleanEquivalent(t *testing.T) {
	t.Helper()
	clean := newCobolLifecycleEnv(t, nil)
	clean.roots, clean.repos = env.roots, env.repos
	clean.track(t, slices.Sorted(maps.Keys(env.repos))...)
	requireSameDump(t, clean.dump(t), env.dump(t))
}

// requireRevision checks the one revision stamp a node carries: commit, or
// the absence reason when commit is "".
func (env *cobolLifecycleEnv) requireRevision(t *testing.T, id, commit, absence string) {
	t.Helper()
	n := env.store.GetNode(id)
	require.NotNil(t, n, id)
	if commit != "" {
		assert.Equal(t, commit, n.Meta["prov_vcs_commit"], id)
		assert.NotContains(t, n.Meta, "prov_vcs_commit_absence", id)
		return
	}
	assert.Equal(t, absence, n.Meta["prov_vcs_commit_absence"], id)
	assert.NotContains(t, n.Meta, "prov_vcs_commit", id)
}

// TestCobolLifecycle_HeadMove proves D-17: when HEAD moves and a file's
// bytes do not, the HEAD-move tail re-stamps prov_vcs_commit or
// prov_vcs_commit_absence without re-extraction, and the dump (which
// compares prov_vcs_commit) equals a clean build at the new HEAD.
func TestCobolLifecycle_HeadMove(t *testing.T) {
	const mainID, subID = "lifea/cobol/main.cbl::MAINPGM", "lifea/cobol/sub.cbl::SUBPGM"

	t.Run("commit_other_file", func(t *testing.T) {
		env := newCobolGitEnv(t, nil)
		root := env.roots["lifea"]
		env.write(t, "lifea", "cobol/sub.cbl", cobolLifecycleEdits["cobol/sub.cbl"])
		runGit(t, root, "commit", "-q", "-am", "edit sub")
		env.reindex(t, "lifea")
		env.headMoveTail(t)
		head := headCommit(t, root)
		env.requireRevision(t, mainID, head, "")
		env.requireRevision(t, subID, head, "")
		env.requireRootCleanEquivalent(t)
	})

	t.Run("commit_without_change", func(t *testing.T) {
		env := newCobolGitEnv(t, map[string]string{"cobol/main.cbl": cobolLifecycleEdits["cobol/main.cbl"]})
		env.requireRevision(t, mainID, "", "working_tree_differs_from_head")
		root := env.roots["lifea"]
		runGit(t, root, "commit", "-q", "-am", "commit main")
		env.headMoveTail(t)
		env.requireRevision(t, mainID, headCommit(t, root), "")
		env.requireRootCleanEquivalent(t)
	})

	t.Run("dirty_file_stays_absent", func(t *testing.T) {
		env := newCobolGitEnv(t, map[string]string{"cobol/sub.cbl": cobolLifecycleEdits["cobol/sub.cbl"]})
		root := env.roots["lifea"]
		runGit(t, root, "commit", "-q", "--allow-empty", "-m", "move head")
		env.headMoveTail(t)
		env.requireRevision(t, mainID, headCommit(t, root), "")
		env.requireRevision(t, subID, "", "working_tree_differs_from_head")
		env.requireRootCleanEquivalent(t)
	})

	t.Run("no_reextraction", func(t *testing.T) {
		env := newCobolGitEnv(t, nil)
		before := env.dump(t)
		runGit(t, env.roots["lifea"], "commit", "-q", "--allow-empty", "-m", "move head")
		env.headMoveTail(t)
		after := env.dump(t)
		require.Len(t, after.Rows, len(before.Rows))
		split := func(rows []string) map[string]map[string]any {
			out := make(map[string]map[string]any, len(rows))
			for _, row := range rows {
				cut := strings.LastIndex(row, "\x1f")
				var meta map[string]any
				require.NoError(t, json.Unmarshal([]byte(row[cut+1:]), &meta))
				out[row[:cut]] = meta
			}
			return out
		}
		was, now := split(before.Rows), split(after.Rows)
		changed := 0
		for id, old := range was {
			cur, ok := now[id]
			require.True(t, ok, "row %q kept", id)
			if old["prov_vcs_commit"] != cur["prov_vcs_commit"] {
				changed++
			}
			for _, key := range []string{"prov_vcs_commit", "prov_vcs_commit_absence"} {
				delete(old, key)
				delete(cur, key)
			}
			assert.Equal(t, old, cur, "row %q changed beyond its revision stamp", id)
		}
		assert.Positive(t, changed, "the tail re-stamps at least one row")
	})
}
