//go:build darwin || linux

package indexer

import (
	"encoding/json"
	"maps"
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
