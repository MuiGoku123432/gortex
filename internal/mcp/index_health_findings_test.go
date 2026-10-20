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

// readIndexHealth builds the index_health payload for a graph holding nodes
// and edges, through the same cache-seed path the ownership test uses.
func readIndexHealth(t *testing.T, nodes []*graph.Node, edges []*graph.Edge) map[string]any {
	t.Helper()
	g := graph.New()
	g.AddBatch(nodes, edges)
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
	require.False(t, result.IsError)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcplib.TextContent).Text), &payload))
	return payload
}

// findingHealthBase is a two-repository COBOL graph with no findings: one
// file and one program per repository, joined by a defines edge so the
// payload's edges_ok check stays green with or without findings.
func findingHealthBase() ([]*graph.Node, []*graph.Edge) {
	var nodes []*graph.Node
	var edges []*graph.Edge
	for _, repo := range []string{"r1", "r2"} {
		file := repo + "/PGM.cbl"
		program := file + "::PGM"
		nodes = append(nodes,
			&graph.Node{ID: file, Kind: graph.KindFile, Name: "PGM.cbl", FilePath: file, RepoPrefix: repo, Language: "cobol"},
			&graph.Node{ID: program, Kind: graph.KindFunction, Name: "PGM", FilePath: file, RepoPrefix: repo, Language: "cobol"},
		)
		edges = append(edges, &graph.Edge{From: file, To: program, Kind: graph.EdgeDefines, FilePath: file})
	}
	return nodes, edges
}

// findingHealthNode returns an active finding of class in repo, owned by the
// repository's program, with a member_of edge to it.
func findingHealthNode(repo, class, kind string) (*graph.Node, *graph.Edge) {
	file := repo + "/PGM.cbl"
	program := file + "::PGM"
	id := file + "::" + graph.FindingSymbol("PGM.cbl::PGM", class, kind, "", graph.FindingEvidenceDigest(kind))
	node := &graph.Node{
		ID: id, Kind: graph.KindFinding, Name: kind, FilePath: file, RepoPrefix: repo, Language: "cobol",
		Meta: map[string]any{
			graph.MetaEvidenceClass: class,
			graph.MetaFindingKind:   kind,
			graph.MetaFindingStatus: graph.FindingStatusActive,
		},
	}
	return node, &graph.Edge{From: id, To: program, Kind: graph.EdgeMemberOf, FilePath: file}
}

// findingHealthFindings is the finding set of TestIndexHealth_FindingCounts:
// two parse and one external finding in r1, none in r2.
func findingHealthFindings() ([]*graph.Node, []*graph.Edge) {
	var nodes []*graph.Node
	var edges []*graph.Edge
	for _, f := range []struct{ class, kind string }{
		{graph.FindingClassParse, "unparsed-tail"},
		{graph.FindingClassParse, "error"},
		{graph.FindingClassExternal, "missing"},
	} {
		n, e := findingHealthNode("r1", f.class, f.kind)
		nodes = append(nodes, n)
		edges = append(edges, e)
	}
	return nodes, edges
}

func findingCounts(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	findings, ok := payload["findings"].(map[string]any)
	require.True(t, ok, "index_health must carry a findings block, got %v", payload["findings"])
	return findings
}

// TestIndexHealth_FindingCounts pins D-08 and GAP-03: index_health reports
// active finding counts by prov_evidence_class over the payload's repository
// set, and both class keys are always present.
func TestIndexHealth_FindingCounts(t *testing.T) {
	t.Run("counts_by_class", func(t *testing.T) {
		nodes, edges := findingHealthBase()
		fNodes, fEdges := findingHealthFindings()
		nodes = append(nodes, fNodes...)
		edges = append(edges, fEdges...)

		// A finding outside the scanned repository set and a node with an
		// unknown class value must not be counted.
		outside, outsideEdge := findingHealthNode("r9", graph.FindingClassParse, "error")
		unknown, unknownEdge := findingHealthNode("r2", graph.FindingClassParse, "missing")
		unknown.Meta[graph.MetaEvidenceClass] = "RESOLVED"
		nodes = append(nodes, outside, unknown)
		edges = append(edges, outsideEdge, unknownEdge)

		findings := findingCounts(t, readIndexHealth(t, nodes, edges))
		require.Equal(t, map[string]any{
			graph.FindingClassParse:    float64(2),
			graph.FindingClassExternal: float64(1),
		}, findings)
	})

	t.Run("none", func(t *testing.T) {
		nodes, edges := findingHealthBase()
		findings := findingCounts(t, readIndexHealth(t, nodes, edges))
		require.Equal(t, map[string]any{
			graph.FindingClassParse:    float64(0),
			graph.FindingClassExternal: float64(0),
		}, findings)
	})
}

// TestIndexHealth_FindingsDoNotScore pins D-14 and P4 D-15: findings are
// reported, never enforced, so the same store with and without findings has
// the same health_score and recommendation.
func TestIndexHealth_FindingsDoNotScore(t *testing.T) {
	nodes, edges := findingHealthBase()
	without := readIndexHealth(t, nodes, edges)

	fNodes, fEdges := findingHealthFindings()
	with := readIndexHealth(t, append(nodes, fNodes...), append(edges, fEdges...))

	require.Equal(t, float64(3), findingCounts(t, with)[graph.FindingClassParse].(float64)+
		findingCounts(t, with)[graph.FindingClassExternal].(float64), "the store must hold findings")
	require.Equal(t, without["health_score"], with["health_score"])
	require.Equal(t, without["recommendation"], with["recommendation"])
}
