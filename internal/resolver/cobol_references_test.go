package resolver

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
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

func TestCobolReferences_ProgramBinding(t *testing.T) {
	const caller = "a/main.cbl::MAINPGM"
	const target = "unresolved::cobol::program::SUBPGM"

	t.Run("exactly_one", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "a", "a/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		seedCobol(g, "b", "b/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		// Stale stamps from an earlier pass are cleared on binding.
		cobolEdge(g, caller, 4, graph.EdgeCalls, target, map[string]any{
			"cobol_call":              "literal",
			"cobol_unresolved_reason": "ambiguous",
			"cobol_candidate_ids":     []string{"x", "y"},
		})

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, caller, 4)
		assert.Equal(t, "a/sub.cbl::SUBPGM", e.To, "binds the single same-repo program")
		assert.Equal(t, graph.OriginASTResolved, e.Origin)
		assert.NotContains(t, e.Meta, "cobol_unresolved_reason")
		assert.NotContains(t, e.Meta, "cobol_candidate_ids")
		assert.Equal(t, "literal", e.Meta["cobol_call"], "extractor Meta is kept")
	})

	t.Run("missing", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "b", "b/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		cobolEdge(g, caller, 4, graph.EdgeCalls, target, nil)

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, caller, 4)
		assert.Equal(t, target, e.To, "never bound to another repository's program")
		assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
		assert.NotContains(t, e.Meta, "cobol_candidate_ids")
	})

	t.Run("ambiguous", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "a", "a/s2.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		seedCobol(g, "a", "a/s1.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		cobolEdge(g, caller, 4, graph.EdgeCalls, target, nil)

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, caller, 4)
		assert.Equal(t, target, e.To, "nothing is picked among two candidates")
		assert.Equal(t, "ambiguous", e.Meta["cobol_unresolved_reason"])
		assert.Equal(t, []string{"a/s1.cbl::SUBPGM", "a/s2.cbl::SUBPGM"}, e.Meta["cobol_candidate_ids"])
	})

	t.Run("same_file_first", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", "a/nest.cbl::OUTER", graph.KindFunction, "program", "OUTER")
		seedCobol(g, "a", "a/nest.cbl::OUTER/INNER", graph.KindFunction, "program", "INNER")
		seedCobol(g, "a", "a/other.cbl::INNER", graph.KindFunction, "program", "INNER")
		cobolEdge(g, "a/nest.cbl::OUTER", 9, graph.EdgeCalls, "unresolved::cobol::program::INNER", nil)

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, "a/nest.cbl::OUTER", 9)
		assert.Equal(t, "a/nest.cbl::OUTER/INNER", e.To, "contained program resolves within the file first")
	})

	t.Run("sqlite_persisted", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cobol.sqlite")
		store, err := store_sqlite.Open(path)
		require.NoError(t, err)
		seedCobol(store, "a", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(store, "a", "a/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		seedCobol(store, "a", "a/d1.cbl::DUP", graph.KindFunction, "program", "DUP")
		seedCobol(store, "a", "a/d2.cbl::DUP", graph.KindFunction, "program", "DUP")
		cobolEdge(store, caller, 1, graph.EdgeCalls, target, nil)
		cobolEdge(store, caller, 2, graph.EdgeCalls, "unresolved::cobol::program::NOPE", nil)
		cobolEdge(store, caller, 3, graph.EdgeCalls, "unresolved::cobol::program::DUP", nil)
		New(store).ResolveAll()
		require.NoError(t, store.Close())

		reopened, err := store_sqlite.Open(path)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reopened.Close()) })
		assert.Equal(t, "a/sub.cbl::SUBPGM", cobolEdgeAt(t, reopened, caller, 1).To, "rebind persisted")
		assert.Equal(t, "missing", cobolEdgeAt(t, reopened, caller, 2).Meta["cobol_unresolved_reason"], "missing stamp persisted")
		amb := cobolEdgeAt(t, reopened, caller, 3)
		assert.Equal(t, "ambiguous", amb.Meta["cobol_unresolved_reason"], "ambiguous stamp persisted")
		assert.Len(t, amb.Meta["cobol_candidate_ids"], 2)
	})

	t.Run("idempotent", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "a", "a/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		seedCobol(g, "a", "a/d1.cbl::DUP", graph.KindFunction, "program", "DUP")
		seedCobol(g, "a", "a/d2.cbl::DUP", graph.KindFunction, "program", "DUP")
		cobolEdge(g, caller, 1, graph.EdgeCalls, target, nil)
		cobolEdge(g, caller, 2, graph.EdgeCalls, "unresolved::cobol::program::NOPE", nil)
		cobolEdge(g, caller, 3, graph.EdgeCalls, "unresolved::cobol::program::DUP", nil)

		type snap struct {
			To   string
			Meta map[string]any
		}
		capture := func() map[int]snap {
			out := map[int]snap{}
			for _, e := range g.GetOutEdges(caller) {
				m := map[string]any{}
				for k, v := range e.Meta {
					m[k] = v
				}
				out[e.Line] = snap{To: e.To, Meta: m}
			}
			return out
		}

		New(g).ResolveAll()
		first := capture()
		New(g).ResolveAll()
		second := capture()

		assert.Equal(t, first, second, "a second resolve leaves every edge identical")
		assert.Equal(t, "a/sub.cbl::SUBPGM", first[1].To)
		assert.Equal(t, "missing", first[2].Meta["cobol_unresolved_reason"])
		assert.Equal(t, "ambiguous", first[3].Meta["cobol_unresolved_reason"])
	})
}

func TestCobolReferences_Copybook(t *testing.T) {
	g := graph.New()
	const caller = "a/main.cbl::MAINPGM"
	seedCobol(g, "a", caller, graph.KindFunction, "program", "MAINPGM")
	seedCobol(g, "a", "a/copy/custrec.cpy", graph.KindFile, "copybook", "CUSTREC")
	seedCobol(g, "a", "a/c2/x.cpy", graph.KindFile, "copybook", "X")
	seedCobol(g, "a", "a/c1/x.cpy", graph.KindFile, "copybook", "X")
	seedCobol(g, "a", "a/copy/emp.cpy", graph.KindFile, "copybook", "EMP")
	seedCobol(g, "b", "b/copy/onlyb.cpy", graph.KindFile, "copybook", "ONLYB")

	cobolEdge(g, caller, 1, graph.EdgeImports, "unresolved::cobol::copybook::CUSTREC", nil)
	cobolEdge(g, caller, 2, graph.EdgeImports, "unresolved::cobol::copybook::X", nil)
	cobolEdge(g, caller, 3, graph.EdgeImports, "unresolved::cobol::copybook::ONLYB", nil)

	// Owned targets the pass must never rebind or restamp, even where a
	// same-named copybook or program exists.
	untouched := map[int]string{
		4: "unresolved::cobol::copybook::idms/RECORD/EMP",
		5: "unresolved::cobol::dynamic::program::a/main.cbl::MAINPGM::EMP",
		6: "unresolved::cobol::label::a/main.cbl::MAINPGM::EMP",
		7: "unresolved::cobol::data::a/main.cbl::MAINPGM::CUSTREC",
		8: "unresolved::cobol::idms_scope::_unqualified/EMP",
	}
	reasons := map[int]string{4: "idms_dictionary_member", 5: "dynamic", 6: "label_not_found", 7: "missing", 8: "scope_role_unknown"}
	for line, to := range untouched {
		kind := graph.EdgeReferences
		if line == 4 {
			kind = graph.EdgeImports
		}
		cobolEdge(g, caller, line, kind, to, map[string]any{"cobol_unresolved_reason": reasons[line]})
	}

	New(g).ResolveAll()

	bound := cobolEdgeAt(t, g, caller, 1)
	assert.Equal(t, "a/copy/custrec.cpy", bound.To, "binds the single same-repo copybook")
	assert.Equal(t, graph.OriginASTResolved, bound.Origin)
	assert.NotContains(t, bound.Meta, "cobol_unresolved_reason")

	amb := cobolEdgeAt(t, g, caller, 2)
	assert.Equal(t, "unresolved::cobol::copybook::X", amb.To)
	assert.Equal(t, "ambiguous", amb.Meta["cobol_unresolved_reason"])
	assert.Equal(t, []string{"a/c1/x.cpy", "a/c2/x.cpy"}, amb.Meta["cobol_candidate_ids"])

	other := cobolEdgeAt(t, g, caller, 3)
	assert.Equal(t, "unresolved::cobol::copybook::ONLYB", other.To, "no cross-repository copybook binding")
	assert.Equal(t, "missing", other.Meta["cobol_unresolved_reason"])

	for line, to := range untouched {
		e := cobolEdgeAt(t, g, caller, line)
		assert.Equal(t, to, e.To, "line %d stays unresolved", line)
		assert.Equal(t, map[string]any{"cobol_unresolved_reason": reasons[line]}, e.Meta, "line %d Meta unchanged", line)
	}
}
