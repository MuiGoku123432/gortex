package resolver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// seedSharedResource adds a COBOL shared resource the way the extractor
// emits it: FilePath equals the ID and prov_range_absence is shared_resource.
// Fixtures are invented.
func seedSharedResource(g graph.Store, repo, id string, kind graph.NodeKind) {
	g.AddNode(&graph.Node{
		ID: id, Kind: kind, Name: id[strings.LastIndex(id, "/")+1:], FilePath: id,
		Language: "cobol", RepoPrefix: repo,
		Meta: map[string]any{"prov_range_absence": "shared_resource"},
	})
}

// cobolAliasesFrom returns the EdgeAliases edges leaving id.
func cobolAliasesFrom(g graph.Store, id string) []*graph.Edge {
	var out []*graph.Edge
	for _, e := range g.GetOutEdges(id) {
		if e.Kind == graph.EdgeAliases {
			out = append(out, e)
		}
	}
	return out
}

func TestCobolReconcile_ResourceGC(t *testing.T) {
	const caller = "repoA/main.cbl::MAINPGM"
	repoA := map[string]struct{}{"repoA": {}}

	t.Run("orphan", func(t *testing.T) {
		g := graph.New()
		seedSharedResource(g, "repoA", "repoA/db2::_unqualified/OLD", graph.KindTable)
		New(g).ReconcileCobol(repoA)
		assert.Nil(t, g.GetNode("repoA/db2::_unqualified/OLD"), "an unreferenced shared resource is collected")
	})

	t.Run("referenced_kept", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "repoA", caller, graph.KindFunction, "program", "MAINPGM")
		seedSharedResource(g, "repoA", "repoA/db2::_unqualified/ORDERS", graph.KindTable)
		cobolEdge(g, caller, 3, graph.EdgeReads, "repoA/db2::_unqualified/ORDERS", nil)
		New(g).ReconcileCobol(repoA)
		assert.NotNil(t, g.GetNode("repoA/db2::_unqualified/ORDERS"))
	})

	t.Run("alias_only_evicted", func(t *testing.T) {
		g := graph.New()
		const unq, qual = "repoA/db2::_unqualified/ORDERS", "repoA/db2::MYSCH/ORDERS"
		seedCobol(g, "repoA", caller, graph.KindFunction, "program", "MAINPGM")
		seedSharedResource(g, "repoA", unq, graph.KindTable)
		seedSharedResource(g, "repoA", qual, graph.KindTable)
		cobolEdge(g, caller, 3, graph.EdgeReads, unq, nil)
		g.AddEdge(&graph.Edge{From: unq, To: qual, Kind: graph.EdgeAliases, FilePath: unq})
		New(g).ReconcileCobol(repoA)
		assert.Nil(t, g.GetNode(qual), "a resource reached only through an alias edge is collected")
		assert.Empty(t, cobolAliasesFrom(g, unq), "its alias edge goes with it")
		assert.NotNil(t, g.GetNode(unq))
	})

	t.Run("legacy_guard", func(t *testing.T) {
		g := graph.New()
		const legacy = "repoA/db2::_unqualified/LEGACY"
		seedCobol(g, "repoA", "repoA/legacy.cbl::LEGACYPGM", graph.KindFunction, "program", "LEGACYPGM")
		g.AddNode(&graph.Node{
			ID: legacy, Kind: graph.KindTable, Name: "LEGACY", FilePath: "repoA/legacy.cbl",
			Language: "cobol", RepoPrefix: "repoA",
			Meta: map[string]any{"prov_range_absence": "shared_resource"},
		})
		New(g).ReconcileCobol(repoA)
		assert.NotNil(t, g.GetNode(legacy), "a resource owned by a real file is never evicted")
		assert.NotNil(t, g.GetNode("repoA/legacy.cbl::LEGACYPGM"), "that file's nodes survive")
	})

	t.Run("outside_set", func(t *testing.T) {
		g := graph.New()
		seedSharedResource(g, "repoB", "repoB/db2::_unqualified/OLD", graph.KindTable)
		New(g).ReconcileCobol(repoA)
		assert.NotNil(t, g.GetNode("repoB/db2::_unqualified/OLD"), "repoB is outside the reconcile set")
	})
}

func TestCobolReconcile_Alias(t *testing.T) {
	const caller = "repoA/main.cbl::MAINPGM"
	const unq, qual = "repoA/db2::_unqualified/ORDERS", "repoA/db2::MYSCH/ORDERS"
	repoA := map[string]struct{}{"repoA": {}}
	schema := cobolPolicies(map[string]CobolPolicy{"repoA": {DB2DefaultSchema: "MYSCH"}})

	// seed adds a program that reads every given resource, so none is an orphan.
	seed := func(kind graph.NodeKind, ids ...string) *graph.Graph {
		g := graph.New()
		seedCobol(g, "repoA", caller, graph.KindFunction, "program", "MAINPGM")
		for i, id := range ids {
			seedSharedResource(g, "repoA", id, kind)
			cobolEdge(g, caller, 10+i, graph.EdgeReads, id, nil)
		}
		return g
	}
	assertLink := func(t *testing.T, g graph.Store, from, to string) {
		t.Helper()
		aliases := cobolAliasesFrom(g, from)
		require.Len(t, aliases, 1)
		assert.Equal(t, to, aliases[0].To)
		assert.Equal(t, "cobol-default-qualifier/1", aliases[0].Meta["cobol_qualifier_rule"])
	}

	t.Run("default_schema", func(t *testing.T) {
		g := seed(graph.KindTable, unq, qual)
		r := New(g)
		r.SetCobolPolicyLookup(schema)
		r.ReconcileCobol(repoA)
		assertLink(t, g, unq, qual)
		r.ReconcileCobol(repoA)
		assertLink(t, g, unq, qual)
		assert.Len(t, g.GetInEdges(qual), 2, "a second pass adds nothing")
	})

	t.Run("no_target", func(t *testing.T) {
		g := seed(graph.KindTable, unq)
		r := New(g)
		r.SetCobolPolicyLookup(schema)
		r.ReconcileCobol(repoA)
		assert.Empty(t, cobolAliasesFrom(g, unq))
		assert.Nil(t, g.GetNode(qual), "no qualified node is created")
	})

	t.Run("policy_removed", func(t *testing.T) {
		g := seed(graph.KindTable, unq, qual)
		r := New(g)
		r.SetCobolPolicyLookup(schema)
		r.ReconcileCobol(repoA)
		assertLink(t, g, unq, qual)

		r.SetCobolPolicyLookup(nil)
		r.ReconcileCobol(repoA)
		assert.Empty(t, cobolAliasesFrom(g, unq), "the alias goes when the default goes")
		for id, name := range map[string]string{unq: "ORDERS", qual: "ORDERS"} {
			n := g.GetNode(id)
			require.NotNil(t, n, id)
			assert.Equal(t, name, n.Name, "node %s is never renamed", id)
		}
	})

	t.Run("idms_shape", func(t *testing.T) {
		const iu, iq = "repoA/idms::_unqualified/record/EMP-REC", "repoA/idms::EMPSS/record/EMP-REC"
		g := seed(graph.KindResource, iu, iq)
		r := New(g)
		r.SetCobolPolicyLookup(cobolPolicies(map[string]CobolPolicy{"repoA": {IDMSDefaultSubschema: "EMPSS"}}))
		r.ReconcileCobol(repoA)
		assertLink(t, g, iu, iq)
	})

	t.Run("go_alias_untouched", func(t *testing.T) {
		g := seed(graph.KindTable, unq, qual)
		g.AddEdge(&graph.Edge{From: unq, To: qual, Kind: graph.EdgeAliases, FilePath: unq})
		New(g).ReconcileCobol(repoA)
		aliases := cobolAliasesFrom(g, unq)
		require.Len(t, aliases, 1, "an alias edge without cobol_qualifier_rule is never removed")
		assert.NotContains(t, aliases[0].Meta, "cobol_qualifier_rule")
	})
}
