package resolver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// seedCobol adds a COBOL node carrying the repository prefix and the
// cobol_kind / cobol_name_key Meta the binding pass reads. seedFile sets
// neither, so COBOL tests seed through this helper. Fixtures are invented.
func seedCobol(g graph.Store, repo, id string, kind graph.NodeKind, cobolKind, key string) {
	filePath := id
	name := id
	if i := strings.Index(id, "::"); i >= 0 {
		filePath = id[:i]
		name = id[i+2:]
	}
	g.AddNode(&graph.Node{
		ID: id, Kind: kind, Name: key, QualName: name, FilePath: filePath,
		Language: "cobol", RepoPrefix: repo,
		Meta: map[string]any{"cobol_kind": cobolKind, "cobol_name_key": key},
	})
}

// cobolEdge adds an edge from a seeded caller to an unresolved COBOL target.
// The edge FilePath is the caller's file, the way the extractor stamps it.
func cobolEdge(g graph.Store, from string, line int, kind graph.EdgeKind, to string, meta map[string]any) {
	filePath := from
	if i := strings.Index(from, "::"); i >= 0 {
		filePath = from[:i]
	}
	g.AddEdge(&graph.Edge{
		From: from, To: to, Kind: kind, FilePath: filePath, Line: line,
		Origin: graph.OriginASTResolved, Meta: meta,
	})
}

// cobolEdgeAt returns the live edge leaving from at line.
func cobolEdgeAt(t *testing.T, g graph.Store, from string, line int) *graph.Edge {
	t.Helper()
	for _, e := range g.GetOutEdges(from) {
		if e.Line == line {
			return e
		}
	}
	t.Fatalf("no edge from %s at line %d", from, line)
	return nil
}

func TestCobolNamespaceExclusions(t *testing.T) {
	t.Run("identifier", func(t *testing.T) {
		for _, target := range []string{
			"cobol::program::PGMX",
			"cobol::copybook::CPYA",
			"cobol::dynamic::program::a.cbl::P::WS-PGM",
		} {
			assert.Equal(t, "", identifierFromTarget(target), target)
		}
	})

	t.Run("generic", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", "a/a.cbl::P", graph.KindFunction, "program", "P")
		seedCobol(g, "a", "a/b.cbl::WS-PGM", graph.KindFunction, "program", "WS-PGM")
		g.AddNode(&graph.Node{
			ID: "a/x.go::WS-PGM", Kind: graph.KindFunction, Name: "WS-PGM",
			FilePath: "a/x.go", Language: "go", RepoPrefix: "a",
		})
		target := "unresolved::cobol::dynamic::program::a.cbl::P::WS-PGM"
		cobolEdge(g, "a/a.cbl::P", 7, graph.EdgeCalls, target, map[string]any{"cobol_unresolved_reason": "dynamic"})

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, "a/a.cbl::P", 7)
		assert.Equal(t, target, e.To, "dynamic COBOL target is never bound by a generic pass")
		assert.False(t, edgeTerminalFlag(e), "COBOL target is never classified terminal")
		assert.Equal(t, "dynamic", e.Meta["cobol_unresolved_reason"])
	})

	t.Run("crossrepo", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", "a/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "b", "b/pgmx.cbl::PGMX", graph.KindFunction, "program", "PGMX")
		target := "unresolved::cobol::program::PGMX"
		cobolEdge(g, "a/main.cbl::MAINPGM", 3, graph.EdgeCalls, target, nil)

		stats := NewCrossRepo(g).ResolveAll()

		e := cobolEdgeAt(t, g, "a/main.cbl::MAINPGM", 3)
		assert.Equal(t, target, e.To, "no cross-repository binding by name")
		assert.False(t, e.CrossRepo)
		require.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.Unresolved, 1, "owned COBOL target counts as unresolved")
	})
}
