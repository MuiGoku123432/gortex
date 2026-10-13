package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/indexer"
	"github.com/zzet/gortex/internal/query"
)

// TestIndexHealth_RepoOwnershipStatus pins D-15: index_health reports
// repository ownership as healthy or unhealthy from the strict audit (D-14),
// names the unowned nodes whenever there are any, and still answers — health
// reports, it does not refuse.
func TestIndexHealth_RepoOwnershipStatus(t *testing.T) {
	readOwnership := func(t *testing.T, nodes []*graph.Node) map[string]any {
		t.Helper()
		g := graph.New()
		g.AddBatch(nodes, nil)
		idx := indexer.New(g, testRegistry(), config.Default().Index, zap.NewNop())
		srv := NewServer(query.NewEngine(g), g, idx, nil, zap.NewNop(), nil)
		baseline, err := srv.buildIndexHealthBasePayloadCtx(context.Background())
		require.NoError(t, err)
		srv.indexHealth.mu.Lock()
		srv.indexHealth.payload = baseline
		srv.indexHealth.updatedAt = time.Now()
		srv.indexHealth.mu.Unlock()

		result, err := srv.handleIndexHealth(context.Background(), makeReq("index_health", map[string]any{"format": "json"}))
		require.NoError(t, err)
		require.False(t, result.IsError, "index_health must answer whatever the ownership status")
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcplib.TextContent).Text), &payload))
		ownership, ok := payload["repo_ownership"].(map[string]any)
		require.True(t, ok, "index_health must carry a repo_ownership block")
		return ownership
	}
	global := &graph.Node{ID: "dep::go.uber.org/zap::Logger", Kind: graph.KindFunction, Name: "Logger"}

	t.Run("healthy", func(t *testing.T) {
		ownership := readOwnership(t, []*graph.Node{
			{ID: "r1/a.go::A", Kind: graph.KindFunction, Name: "A", FilePath: "r1/a.go", RepoPrefix: "r1"},
			global,
		})
		require.Equal(t, "healthy", ownership["status"])
		require.Equal(t, true, ownership["consistent"])
		require.NotContains(t, ownership, "unowned_samples")
	})

	t.Run("unhealthy_unowned", func(t *testing.T) {
		ownership := readOwnership(t, []*graph.Node{
			{ID: "a.go::A", Kind: graph.KindFunction, Name: "A", FilePath: "a.go"},
			global,
		})
		require.Equal(t, "unhealthy", ownership["status"])
		require.Equal(t, float64(1), ownership["unowned_code_nodes"])
		require.Equal(t, []any{"a.go::A"}, ownership["unowned_samples"])
		require.NotContains(t, ownership, "mixed", "a wholly unowned graph is not the mixed ghost graph")
	})
}
