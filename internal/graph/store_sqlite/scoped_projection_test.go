package store_sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/testutil/graphfixture"
)

func TestScopedProjectionTracer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.db")
	store, err := Open(path)
	require.NoError(t, err)
	store.AddBatch([]*graph.Node{
		{ID: graphfixture.ScopedNodeID, RepoPrefix: "fixture/repo", FilePath: "fixture/repo/main.go", Kind: graph.NodeKind("function"), Name: "Run"},
		{ID: graphfixture.ScopedTargetID, RepoPrefix: "fixture/repo", FilePath: "fixture/repo/main.go", Kind: graph.NodeKind("type"), Name: "Store"},
		{ID: "neighbor/repo/main.go::Run", RepoPrefix: "neighbor/repo", FilePath: "neighbor/repo/main.go", Kind: graph.NodeKind("function"), Name: "Run"},
	}, []*graph.Edge{{From: graphfixture.ScopedNodeID, To: graphfixture.ScopedTargetID, Kind: graph.EdgeKind("references"), FilePath: "fixture/repo/main.go"}})
	require.NoError(t, store.CheckpointWAL())
	before := projectionFileBytes(t, path)

	_, err = store.OpenScopedProjectionSnapshot(context.Background(), graph.ProjectionScope{})
	require.ErrorContains(t, err, "repository allow-set is required")

	snapshot, err := store.OpenScopedProjectionSnapshot(context.Background(), graph.ProjectionScope{Repositories: []string{"fixture/repo"}})
	require.NoError(t, err)
	require.Equal(t, int64(0), snapshot.Descriptor().SourceGeneration)

	nodes, err := snapshot.ReadNodes(context.Background())
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	require.Equal(t, []string{graphfixture.ScopedNodeID, graphfixture.ScopedTargetID}, []string{nodes[0].ID, nodes[1].ID})

	edges, err := snapshot.ReadEdges(context.Background())
	require.NoError(t, err)
	require.Len(t, edges, 1)
	require.Equal(t, graph.EdgeKind("references"), edges[0].Edge.Kind)
	require.Equal(t, graphfixture.ScopedNodeID, edges[0].Source.ID)
	require.Equal(t, graphfixture.ScopedTargetID, edges[0].Target.ID)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = snapshot.ReadNodes(cancelled)
	require.True(t, errors.Is(err, context.Canceled), "got %v", err)
	require.NoError(t, snapshot.Close())
	require.ErrorContains(t, snapshot.Close(), "closed")

	require.NoError(t, store.CheckpointWAL())
	after := projectionFileBytes(t, path)
	require.Equal(t, before, after)
	require.NoError(t, store.Close())
}

func projectionFileBytes(t *testing.T, path string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		data, err := os.ReadFile(path + suffix)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)
		result[suffix] = data
	}
	return result
}

var _ = time.Second
