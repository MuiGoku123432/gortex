//go:build darwin || linux

package indexer

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/testutil/graphdump"
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

// requireConfigCleanEquivalent is requireCleanEquivalent for a sequence that
// added repo's .gortex.yaml on the incremental path. graphdump strips the
// incremental-only source_*_fingerprint keys, so a file node whose Meta held
// nothing else dumps as {} where a clean build dumps null; that holds for
// every non-COBOL file and predates this plan (04-09 deferred-items.md). Only
// that row of repo's .gortex.yaml is normalized; every other row, COBOL
// included, must match the clean build exactly.
func (env *cobolLifecycleEnv) requireConfigCleanEquivalent(t *testing.T, repo string) graphdump.Result {
	t.Helper()
	got := env.dump(t)
	want := cleanCobolDump(t, env.repos, slices.Sorted(maps.Keys(env.repos))...)
	configRow := "N\x1f" + repo + "/.gortex.yaml\x1f"
	rows := slices.Clone(got.Rows)
	for i, line := range rows {
		if strings.HasPrefix(line, configRow) && strings.HasSuffix(line, "\x1f{}") {
			rows[i] = strings.TrimSuffix(line, "{}") + "null"
		}
	}
	require.Equal(t, want.Rows, rows, graphdump.FirstDiff(want.Rows, rows, 20))
	for _, line := range rows {
		require.NotContains(t, line, "restub_prev_")
	}
	return got
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
		got := env.requireConfigCleanEquivalent(t, "crossa")
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
		got := env.requireConfigCleanEquivalent(t, "crossa")
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
	env.requireConfigCleanEquivalent(t, "crossa")
}
