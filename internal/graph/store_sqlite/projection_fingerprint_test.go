package store_sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/testutil/graphfixture"
)

func TestScopedProjectionFingerprintCoverage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fingerprint.sqlite")
	store, err := Open(path)
	require.NoError(t, err)
	seedEstateProjection(t, store, scopedProjectionPage+17)
	require.NoError(t, store.CheckpointWAL())
	warm, err := store.OpenScopedProjectionSnapshot(context.Background(), graph.ProjectionScope{Workspace: "ws", Project: "project", Repositories: []string{"estate/a", "estate/b"}})
	require.NoError(t, err)
	require.NoError(t, warm.ReadNodePages(context.Background(), func([]*graph.Node) error { return nil }))
	require.NoError(t, warm.ReadEdgePages(context.Background(), func([]graph.ScopedEdgeRow) error { return nil }))
	require.NoError(t, warm.Close())
	require.NoError(t, store.Close())

	store, err = Open(path)
	require.NoError(t, err)
	require.NoError(t, store.Close())
	before, err := graphfixture.Snapshot(path)
	require.NoError(t, err)
	store, err = Open(path)
	require.NoError(t, err)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_, ok := before.Files[suffix]
		require.True(t, ok, "missing %q fingerprint", suffix)
	}

	for _, tc := range []struct {
		name string
		ctx  func() context.Context
	}{
		{name: "success", ctx: context.Background},
		{name: "cancelled", ctx: func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, openErr := store.OpenScopedProjectionSnapshot(context.Background(), graph.ProjectionScope{Workspace: "ws", Project: "project", Repositories: []string{"estate/a", "estate/b"}})
			require.NoError(t, openErr)
			err := snapshot.ReadNodePages(tc.ctx(), func([]*graph.Node) error { return nil })
			if tc.name == "cancelled" {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.NoError(t, err)
				require.NoError(t, snapshot.ReadEdgePages(tc.ctx(), func([]graph.ScopedEdgeRow) error { return nil }))
			}
			require.NoError(t, snapshot.Close())
		})
	}
	require.NoError(t, store.CheckpointWAL())
	settle, err := store.OpenScopedProjectionSnapshot(context.Background(), graph.ProjectionScope{Workspace: "ws", Project: "project", Repositories: []string{"estate/a", "estate/b"}})
	require.NoError(t, err)
	require.NoError(t, settle.ReadNodePages(context.Background(), func([]*graph.Node) error { return nil }))
	require.NoError(t, settle.ReadEdgePages(context.Background(), func([]graph.ScopedEdgeRow) error { return nil }))
	require.NoError(t, settle.Close())
	require.NoError(t, store.Close())
	after, err := graphfixture.Snapshot(path)
	require.NoError(t, err)
	require.Equal(t, before.Canonical, after.Canonical)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		require.Equal(t, before.Files[suffix], after.Files[suffix], "projection read changed %q fingerprint", suffix)
	}
}

func TestScopedProjectionCancellation(t *testing.T) {
	store := openScopedProjectionTestStore(t)
	seedEstateProjection(t, store, scopedProjectionPage*2+11)
	snapshot, err := store.OpenScopedProjectionSnapshot(context.Background(), graph.ProjectionScope{Workspace: "ws", Project: "project", Repositories: []string{"estate/a"}})
	require.NoError(t, err)
	defer snapshot.Close()

	ctx, cancel := context.WithCancel(context.Background())
	pages := 0
	err = snapshot.ReadNodePages(ctx, func(page []*graph.Node) error {
		pages++
		require.LessOrEqual(t, len(page), scopedProjectionPage)
		cancel()
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, pages)

	sentinel := errors.New("consumer stopped")
	err = snapshot.ReadEdgePages(context.Background(), func([]graph.ScopedEdgeRow) error { return sentinel })
	require.ErrorIs(t, err, sentinel)
}

func seedEstateProjection(t *testing.T, store *Store, perRepo int) {
	t.Helper()
	nodes := make([]*graph.Node, 0, perRepo*2)
	edges := make([]*graph.Edge, 0, perRepo*4)
	for _, repo := range []string{"estate/a", "estate/b"} {
		for i := 0; i < perRepo; i++ {
			id := fmt.Sprintf("%s/file-%04d.go::Fn", repo, i)
			target := fmt.Sprintf("%s/file-%04d.go::Type", repo, i)
			nodes = append(nodes,
				&graph.Node{ID: id, RepoPrefix: repo, WorkspaceID: "ws", ProjectID: "project", FilePath: fmt.Sprintf("%s/file-%04d.go", repo, i), Kind: graph.KindFunction, Name: "Fn"},
				&graph.Node{ID: target, RepoPrefix: repo, WorkspaceID: "ws", ProjectID: "project", FilePath: fmt.Sprintf("%s/file-%04d.go", repo, i), Kind: graph.KindType, Name: "Type"},
			)
			edges = append(edges,
				&graph.Edge{From: id, To: target, Kind: graph.EdgeCalls, FilePath: fmt.Sprintf("%s/file-%04d.go", repo, i), Line: i + 1},
				&graph.Edge{From: id, To: target, Kind: graph.EdgeReferences, FilePath: fmt.Sprintf("%s/file-%04d.go", repo, i), Line: i + 1},
			)
		}
	}
	store.AddBatch(nodes, edges)
}
