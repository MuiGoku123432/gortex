package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
	"github.com/zzet/gortex/internal/search"
)

// writeNestedTrackedRepos lays out alpha at <tmp>/alpha and beta at
// <tmp>/alpha/nested, each with one Go file.
// note: internal/mcp/source_scope_test.go (newNestedCheckoutServer) holds the
// same layout for the source-gate tests; it lives in another package.
func writeNestedTrackedRepos(t *testing.T) (alphaDir, betaDir string) {
	t.Helper()
	alphaDir = filepath.Join(t.TempDir(), "alpha")
	betaDir = filepath.Join(alphaDir, "nested")
	require.NoError(t, os.MkdirAll(betaDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(alphaDir, "main.go"),
		[]byte("package main\n\nfunc AlphaThing() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(betaDir, "main.go"),
		[]byte("package main\n\nfunc ZebraWidget() {}\n"), 0o644))
	return alphaDir, betaDir
}

// newNestedTrackedMulti saves repos as the global config and builds a
// MultiIndexer over a real SQLite store.
func newNestedTrackedMulti(t *testing.T, repos []config.RepoEntry) (*MultiIndexer, *store_sqlite.Store) {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	gc := &config.GlobalConfig{Repos: repos}
	gc.SetConfigPath(cfgPath)
	require.NoError(t, gc.Save())
	cm, err := config.NewConfigManager(cfgPath)
	require.NoError(t, err)
	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "g.sqlite"))
	require.NoError(t, err)
	mi := NewMultiIndexer(store, newTestRegistry(), search.NewNull(), cm, zap.NewNop())
	t.Cleanup(func() {
		_ = mi.Close(context.Background())
		_ = store.Close()
	})
	return mi, store
}

// requireNestedIndexedOnce asserts beta's file has one owner (beta) and the
// ownership audit is clean.
func requireNestedIndexedOnce(t *testing.T, store *store_sqlite.Store) {
	t.Helper()
	require.Nil(t, store.GetNode("alpha/nested/main.go::ZebraWidget"),
		"alpha must not index a file under the nested tracked root")
	require.Empty(t, store.GetFileNodes("alpha/nested/main.go"))
	require.NotNil(t, store.GetNode("beta/main.go::ZebraWidget"))
	require.NotNil(t, store.GetNode("alpha/main.go::AlphaThing"))

	d, ok := graph.ReadPrefixDiagnostics(store, 5)
	require.True(t, ok, "the SQLite store must answer the ownership audit")
	require.Positive(t, d.OwnedCodeNodes, "sanity: owned code nodes (%s)", d.Summary())
	require.Zero(t, d.UnownedCodeNodes, "unowned nodes (%s)", d.Summary())
	require.Zero(t, d.MisprefixedNodes, "misprefixed nodes: %v (%s)", d.MisprefixedSamples, d.Summary())
	require.True(t, d.Clean(), "repo ownership audit must be clean (%s)", d.Summary())
}

// TestNestedTrackedRoot_IndexedOnce: beta's root lies inside alpha's and the
// config tracks both. Whichever order the config lists them, alpha's walk
// stops at beta's root, so beta's file is owned once (CR-01, SEC-08).
func TestNestedTrackedRoot_IndexedOnce(t *testing.T) {
	for _, order := range []string{"alpha_first", "beta_first"} {
		t.Run(order, func(t *testing.T) {
			alphaDir, betaDir := writeNestedTrackedRepos(t)
			alpha := config.RepoEntry{Path: alphaDir, Name: "alpha", Workspace: "shared"}
			beta := config.RepoEntry{Path: betaDir, Name: "beta", Workspace: "shared"}
			repos := []config.RepoEntry{alpha, beta}
			if order == "beta_first" {
				repos = []config.RepoEntry{beta, alpha}
			}
			mi, store := newNestedTrackedMulti(t, repos)
			_, err := mi.IndexScoped("", "")
			require.NoError(t, err)

			requireNestedIndexedOnce(t, store)

			alphaIdx := mi.GetIndexer("alpha")
			require.NotNil(t, alphaIdx)
			require.True(t, alphaIdx.shouldExclude(betaDir, alphaDir, true))
			require.True(t, alphaIdx.shouldExclude(filepath.Join(betaDir, "main.go"), alphaDir, false))
			require.False(t, alphaIdx.shouldExclude(filepath.Join(alphaDir, "main.go"), alphaDir, false))

			betaIdx := mi.GetIndexer("beta")
			require.NotNil(t, betaIdx)
			require.False(t, betaIdx.shouldExclude(filepath.Join(betaDir, "main.go"), betaDir, false),
				"an ancestor root must never exclude the nested repository's own files")
		})
	}
}
