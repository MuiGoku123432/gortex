package indexer

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/excludes"
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
	return newNestedTrackedMultiFromConfig(t, &config.GlobalConfig{Repos: repos})
}

// newNestedTrackedMultiFromConfig saves gc as the global config and builds a
// MultiIndexer over a real SQLite store.
func newNestedTrackedMultiFromConfig(t *testing.T, gc *config.GlobalConfig) (*MultiIndexer, *store_sqlite.Store) {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
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
			// The explicit-path probe skips directory pruning and takes the
			// regular-file fast path, so it must refuse the nested file too.
			skip, ok := alphaIdx.PathIndexability("nested/main.go")
			require.True(t, ok)
			require.True(t, skip.Skipped && skip.ByRule, "alpha must report beta's file as excluded")

			betaIdx := mi.GetIndexer("beta")
			require.NotNil(t, betaIdx)
			require.False(t, betaIdx.shouldExclude(filepath.Join(betaDir, "main.go"), betaDir, false),
				"an ancestor root must never exclude the nested repository's own files")
		})
	}
}

// trackNestedLater indexes alpha alone, then tracks beta without touching the
// config. Alpha indexes the nested files because beta is neither declared by
// alpha's batch nor registered when that batch runs.
func trackNestedLater(t *testing.T) (mi *MultiIndexer, store *store_sqlite.Store, alphaDir, betaDir string) {
	t.Helper()
	alphaDir, betaDir = writeNestedTrackedRepos(t)
	mi, store = newNestedTrackedMulti(t, []config.RepoEntry{
		{Path: alphaDir, Name: "alpha", Workspace: "shared"},
	})
	_, err := mi.IndexScoped("", "")
	require.NoError(t, err)
	require.NotNil(t, store.GetNode("alpha/nested/main.go::ZebraWidget"),
		"precondition: alpha indexed the nested files before beta was tracked")
	_, err = mi.TrackRepoCtx(context.Background(),
		config.RepoEntry{Path: betaDir, Name: "beta", Workspace: "shared"})
	require.NoError(t, err)
	return mi, store, alphaDir, betaDir
}

// TestNestedTrackedRoot_OuterReindexDropsDuplicates: alpha was indexed before
// beta was tracked, so it holds duplicates of beta's file. Its next full
// reindex consults the registered roots live and drops them (D-16, SEC-01).
func TestNestedTrackedRoot_OuterReindexDropsDuplicates(t *testing.T) {
	mi, store, _, _ := trackNestedLater(t)
	_, err := mi.IndexRepo("alpha")
	require.NoError(t, err)
	requireNestedIndexedOnce(t, store)
}

// TestNestedTrackedRoot_WatcherRefusesNestedEvents: the live path does not
// route through shouldExclude, so alpha's watcher must refuse beta's paths on
// its own, immediately after beta is tracked and without an alpha reindex.
func TestNestedTrackedRoot_WatcherRefusesNestedEvents(t *testing.T) {
	mi, _, alphaDir, betaDir := trackNestedLater(t)

	wa := &Watcher{indexer: mi.GetIndexer("alpha"), excludes: excludes.New(excludes.Builtin)}
	require.True(t, wa.isExcluded(betaDir))
	require.True(t, wa.isExcluded(filepath.Join(betaDir, "main.go")))
	require.False(t, wa.isExcluded(filepath.Join(alphaDir, "main.go")))

	wb := &Watcher{indexer: mi.GetIndexer("beta"), excludes: excludes.New(excludes.Builtin)}
	require.False(t, wb.isExcluded(filepath.Join(betaDir, "main.go")),
		"beta's own watcher must keep admitting its files")
}

// TestNestedTrackedRoot_ProjectsMapIndexedOnce: the nested pair is defined
// only under a projects: entry selected by active_project. The batch declares
// the repositories it indexes, so alpha's walk stops at beta's root even
// though the top-level repos: list is empty (CR-01 under-prune, SEC-08).
func TestNestedTrackedRoot_ProjectsMapIndexedOnce(t *testing.T) {
	alphaDir, betaDir := writeNestedTrackedRepos(t)
	mi, store := newNestedTrackedMultiFromConfig(t, &config.GlobalConfig{
		Projects: map[string]config.ProjectConfig{"p1": {Repos: []config.RepoEntry{
			{Path: alphaDir, Name: "alpha", Workspace: "shared"},
			{Path: betaDir, Name: "beta", Workspace: "shared"},
		}}},
		ActiveProject: "p1",
	})
	_, err := mi.IndexScoped("", "")
	require.NoError(t, err)

	prefixes := mi.RepoPrefixes()
	sort.Strings(prefixes)
	require.Equal(t, []string{"alpha", "beta"}, prefixes)
	requireNestedIndexedOnce(t, store)
}

// TestNestedTrackedRoot_OuterOwnsUnindexedNestedEntry: beta is a repos: entry
// the batch does not index (another project is active, or IndexScoped's
// project filter drops it). Beta's root must not prune alpha's walk, so alpha
// owns the nested file and the graph agrees with RepoForFile (CR-01
// over-prune, ID-09).
func TestNestedTrackedRoot_OuterOwnsUnindexedNestedEntry(t *testing.T) {
	cases := []struct {
		name          string
		activeProject string
		projectFilter string
	}{
		{name: "other_active_project", activeProject: "p1"},
		{name: "project_filter", projectFilter: "p1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alphaDir, betaDir := writeNestedTrackedRepos(t)
			mi, store := newNestedTrackedMultiFromConfig(t, &config.GlobalConfig{
				Repos: []config.RepoEntry{
					{Path: alphaDir, Name: "alpha", Workspace: "shared", Project: "p1"},
					{Path: betaDir, Name: "beta", Workspace: "shared", Project: "p2"},
				},
				ActiveProject: tc.activeProject,
			})
			_, err := mi.IndexScoped("", tc.projectFilter)
			require.NoError(t, err)

			require.Equal(t, []string{"alpha"}, mi.RepoPrefixes())
			require.NotNil(t, store.GetNode("alpha/nested/main.go::ZebraWidget"),
				"an unindexed nested entry must not erase alpha's files")
			require.Equal(t, "alpha", mi.RepoForFile(filepath.Join(betaDir, "main.go")))

			d, ok := graph.ReadPrefixDiagnostics(store, 5)
			require.True(t, ok, "the SQLite store must answer the ownership audit")
			require.Positive(t, d.OwnedCodeNodes, "sanity: owned code nodes (%s)", d.Summary())
			require.Zero(t, d.UnownedCodeNodes, "unowned nodes (%s)", d.Summary())
			require.Zero(t, d.MisprefixedNodes, "misprefixed nodes: %v (%s)", d.MisprefixedSamples, d.Summary())
			require.True(t, d.Clean(), "repo ownership audit must be clean (%s)", d.Summary())
		})
	}
}

// TestNestedTrackedRoot_UntrackStopsExcluding: after beta is untracked, the
// same alpha Indexer stops excluding beta's former root at once, on the walk,
// the explicit-path probe and the watcher, and an incremental pass indexes the
// file under alpha (WR-01, ID-09, SC-4).
func TestNestedTrackedRoot_UntrackStopsExcluding(t *testing.T) {
	alphaDir, betaDir := writeNestedTrackedRepos(t)
	mi, store := newNestedTrackedMulti(t, []config.RepoEntry{
		{Path: alphaDir, Name: "alpha", Workspace: "shared"},
		{Path: betaDir, Name: "beta", Workspace: "shared"},
	})
	_, err := mi.IndexScoped("", "")
	require.NoError(t, err)
	alphaBefore := mi.GetIndexer("alpha")
	require.NotNil(t, alphaBefore)

	mi.UntrackRepo("beta")

	require.Same(t, alphaBefore, mi.GetIndexer("alpha"))
	betaFile := filepath.Join(betaDir, "main.go")
	require.False(t, alphaBefore.shouldExclude(betaFile, alphaDir, false),
		"alpha must stop excluding an untracked root immediately")
	skip, ok := alphaBefore.PathIndexability("nested/main.go")
	require.True(t, ok)
	require.False(t, skip.Skipped, "alpha must report the formerly nested file as indexable")
	wa := &Watcher{indexer: alphaBefore, excludes: excludes.New(excludes.Builtin)}
	require.False(t, wa.isExcluded(betaFile))

	_, err = mi.IncrementalReindexRepo("alpha", []string{betaFile})
	require.NoError(t, err)
	require.NotNil(t, store.GetNode("alpha/nested/main.go::ZebraWidget"))
}

// TestNestedTrackedRoot_DeclaredRootsPruneWarmupOrder mirrors the daemon
// warmup: the pool tracks the configured repositories inside one topology
// batch, alpha before beta, so beta is not registered when alpha walks. The
// warmup declares the list it tracks, which stops alpha at beta's root; after
// release the registered roots alone keep pruning (CR-01, SEC-08).
func TestNestedTrackedRoot_DeclaredRootsPruneWarmupOrder(t *testing.T) {
	alphaDir, betaDir := writeNestedTrackedRepos(t)
	repos := []config.RepoEntry{
		{Path: alphaDir, Name: "alpha", Workspace: "shared"},
		{Path: betaDir, Name: "beta", Workspace: "shared"},
	}
	mi, store := newNestedTrackedMulti(t, repos)

	release := mi.DeclarePendingRoots(repos)
	err := mi.RunRepositoryTopologyBatch(context.Background(), func(batchCtx context.Context) error {
		for _, entry := range repos {
			if _, err := mi.TrackRepoCtx(batchCtx, entry); err != nil {
				return err
			}
		}
		return nil
	})
	release()
	require.NoError(t, err)

	requireNestedIndexedOnce(t, store)
	alphaIdx := mi.GetIndexer("alpha")
	require.NotNil(t, alphaIdx)
	require.True(t, alphaIdx.shouldExclude(filepath.Join(betaDir, "main.go"), alphaDir, false),
		"after release the registered roots alone must keep pruning")
}
