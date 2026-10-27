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
	meta := map[string]any{"cobol_kind": cobolKind, "cobol_name_key": key}
	if cobolKind == "program" {
		meta["cobol_nested"] = false // top-level, as the extractor stamps it
	}
	g.AddNode(&graph.Node{
		ID: id, Kind: kind, Name: key, QualName: name, FilePath: filePath,
		Language: "cobol", RepoPrefix: repo, Meta: meta,
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
		assert.Equal(t, "a::cobol-missing::program/SUBPGM", e.To, "never bound to another repository's program")
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

	// seedNested seeds a contained program in repository a and stamps the
	// cobol_nested and cobol_container Meta the extractor writes on it. The
	// in-memory graph hands out the live node, so the stamps land on the
	// stored node.
	seedNested := func(g graph.Store, id, key, container string) {
		seedCobol(g, "a", id, graph.KindFunction, "program", key)
		g.GetNode(id).Meta["cobol_nested"] = true
		g.GetNode(id).Meta["cobol_container"] = container
	}
	// callFrom is the literal CALL Meta naming the calling program.
	callFrom := func(program string) map[string]any {
		return map[string]any{"cobol_caller_program": program}
	}
	const crossCaller = "a/caller.cbl::CALLER"
	const innerTarget = "unresolved::cobol::program::INNER"

	t.Run("nested_cross_file_missing", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", crossCaller, graph.KindFunction, "program", "CALLER")
		seedCobol(g, "a", "a/nest.cbl::OUTER", graph.KindFunction, "program", "OUTER")
		seedNested(g, "a/nest.cbl::OUTER/INNER", "INNER", "a/nest.cbl::OUTER")
		cobolEdge(g, crossCaller, 5, graph.EdgeCalls, innerTarget, nil)

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, crossCaller, 5)
		assert.Equal(t, "a::cobol-missing::program/INNER", e.To, "a contained program is not callable from another file")
		assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
		assert.NotContains(t, e.Meta, "cobol_candidate_ids")
	})

	t.Run("nested_cross_file_ignored", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", crossCaller, graph.KindFunction, "program", "CALLER")
		seedCobol(g, "a", "a/nest.cbl::OUTER", graph.KindFunction, "program", "OUTER")
		seedNested(g, "a/nest.cbl::OUTER/INNER", "INNER", "a/nest.cbl::OUTER")
		seedCobol(g, "a", "a/other.cbl::INNER", graph.KindFunction, "program", "INNER")
		cobolEdge(g, crossCaller, 5, graph.EdgeCalls, innerTarget, nil)

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, crossCaller, 5)
		assert.Equal(t, "a/other.cbl::INNER", e.To, "the top-level program is the only cross-file candidate")
		assert.NotContains(t, e.Meta, "cobol_unresolved_reason")
	})

	t.Run("nested_same_file", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", "a/nest.cbl::OUTER", graph.KindFunction, "program", "OUTER")
		seedNested(g, "a/nest.cbl::OUTER/INNER", "INNER", "a/nest.cbl::OUTER")
		cobolEdge(g, "a/nest.cbl::OUTER", 9, graph.EdgeCalls, innerTarget, callFrom("a/nest.cbl::OUTER"))

		New(g).ResolveAll()

		require.Equal(t, true, g.GetNode("a/nest.cbl::OUTER/INNER").Meta["cobol_nested"])
		e := cobolEdgeAt(t, g, "a/nest.cbl::OUTER", 9)
		assert.Equal(t, "a/nest.cbl::OUTER/INNER", e.To, "a contained program binds within its own file")
		assert.NotContains(t, e.Meta, "cobol_unresolved_reason")
	})

	t.Run("nested_excluded_from_candidates", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", crossCaller, graph.KindFunction, "program", "CALLER")
		seedCobol(g, "a", "a/d1.cbl::DUP", graph.KindFunction, "program", "DUP")
		seedCobol(g, "a", "a/d2.cbl::DUP", graph.KindFunction, "program", "DUP")
		seedCobol(g, "a", "a/nest.cbl::OUTER", graph.KindFunction, "program", "OUTER")
		seedNested(g, "a/nest.cbl::OUTER/DUP", "DUP", "a/nest.cbl::OUTER")
		cobolEdge(g, crossCaller, 5, graph.EdgeCalls, "unresolved::cobol::program::DUP", nil)

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, crossCaller, 5)
		assert.Equal(t, "ambiguous", e.Meta["cobol_unresolved_reason"])
		assert.Equal(t, []string{"a/d1.cbl::DUP", "a/d2.cbl::DUP"}, e.Meta["cobol_candidate_ids"],
			"a contained program in another file is never listed as a candidate")
	})

	// WR-08: a contained program is visible only from its direct container.
	t.Run("nested_unrelated_same_file", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", "a/batch.cbl::A", graph.KindFunction, "program", "A")
		seedNested(g, "a/batch.cbl::A/UTIL", "UTIL", "a/batch.cbl::A")
		seedCobol(g, "a", "a/batch.cbl::B", graph.KindFunction, "program", "B")
		seedCobol(g, "a", "a/util.cbl::UTIL", graph.KindFunction, "program", "UTIL")
		cobolEdge(g, "a/batch.cbl::B", 7, graph.EdgeCalls, "unresolved::cobol::program::UTIL", callFrom("a/batch.cbl::B"))

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, "a/batch.cbl::B", 7)
		assert.Equal(t, "a/util.cbl::UTIL", e.To, "B cannot reach A's contained UTIL, so the top-level UTIL binds")
	})

	t.Run("nested_sibling_missing", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", "a/p.cbl::P", graph.KindFunction, "program", "P")
		seedNested(g, "a/p.cbl::P/X", "X", "a/p.cbl::P")
		seedNested(g, "a/p.cbl::P/Y", "Y", "a/p.cbl::P")
		cobolEdge(g, "a/p.cbl::P/Y", 12, graph.EdgeCalls, "unresolved::cobol::program::X", callFrom("a/p.cbl::P/Y"))

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, "a/p.cbl::P/Y", 12)
		assert.Equal(t, "a::cobol-missing::program/X", e.To, "a sibling is unreachable unless COMMON, which is not observed")
		assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
	})

	t.Run("nested_flag_missing_fails_closed", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "a", crossCaller, graph.KindFunction, "program", "CALLER")
		seedCobol(g, "a", "a/old.cbl::INNER", graph.KindFunction, "program", "INNER")
		delete(g.GetNode("a/old.cbl::INNER").Meta, "cobol_nested") // indexed before the flag existed
		cobolEdge(g, crossCaller, 5, graph.EdgeCalls, innerTarget, callFrom(crossCaller))

		New(g).ResolveAll()

		e := cobolEdgeAt(t, g, crossCaller, 5)
		assert.Equal(t, "a::cobol-missing::program/INNER", e.To, "a program without cobol_nested is never a candidate")
		assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
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

	// A COPY IDMS member is never bound, even where a same-named copybook
	// exists; it moves onto its dictionary-member placeholder node (D-11).
	cobolEdge(g, caller, 4, graph.EdgeImports, "unresolved::cobol::copybook::idms/RECORD/EMP", cobolIDMSMeta())
	// Owned targets the pass must never rebind or restamp, even where a
	// same-named copybook or program exists.
	untouched := map[int]string{
		5: "unresolved::cobol::dynamic::program::a/main.cbl::MAINPGM::EMP",
		6: "unresolved::cobol::label::a/main.cbl::MAINPGM::EMP",
		7: "unresolved::cobol::data::a/main.cbl::MAINPGM::CUSTREC",
		8: "unresolved::cobol::idms_scope::_unqualified/EMP",
	}
	reasons := map[int]string{5: "dynamic", 6: "label_not_found", 7: "missing", 8: "scope_role_unknown"}
	for line, to := range untouched {
		cobolEdge(g, caller, line, graph.EdgeReferences, to, map[string]any{"cobol_unresolved_reason": reasons[line]})
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
	assert.Equal(t, "a::cobol-missing::copybook/ONLYB", other.To, "no cross-repository copybook binding")
	assert.Equal(t, "missing", other.Meta["cobol_unresolved_reason"])

	idms := cobolEdgeAt(t, g, caller, 4)
	assert.Equal(t, "a::cobol-missing::idms_dictionary_member/RECORD/EMP", idms.To, "an IDMS member targets its placeholder node")
	assert.Equal(t, cobolIDMSMeta(), idms.Meta, "the IDMS edge Meta is unchanged")

	for line, to := range untouched {
		e := cobolEdgeAt(t, g, caller, line)
		assert.Equal(t, to, e.To, "line %d stays unresolved", line)
		assert.Equal(t, map[string]any{"cobol_unresolved_reason": reasons[line]}, e.Meta, "line %d Meta unchanged", line)
	}
}

// seedCobolNode adds a COBOL node of repository r1 with exactly meta. Unlike
// seedCobol it never mutates a stored node, so it seeds every store.
func seedCobolNode(g graph.Store, id string, kind graph.NodeKind, meta map[string]any) {
	filePath, name := id, id
	if i := strings.Index(id, "::"); i >= 0 {
		filePath, name = id[:i], id[i+2:]
	}
	key, _ := meta["cobol_name_key"].(string)
	g.AddNode(&graph.Node{
		ID: id, Kind: kind, Name: key, QualName: name, FilePath: filePath,
		Language: "cobol", RepoPrefix: "r1", Meta: meta,
	})
}

// IN-08: a CALL whose only same-named program is nested under END PROGRAM
// containment the extractor could not resolve is containment_unresolved, not
// missing: that program exists, so no placeholder or finding claims it was
// never obtained (D-09). Without the stamp the same CALL is missing.
func TestCobolReconcile_ContainmentUnresolved(t *testing.T) {
	const caller = "r1/a.cbl::CALLER"
	const target = "unresolved::cobol::program::INNER"
	const inner = "r1/b.cbl::OUTER/INNER"
	seed := func(g graph.Store, stamped bool) {
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "CALLER")
		seedCobol(g, "r1", "r1/b.cbl::OUTER", graph.KindFunction, "program", "OUTER")
		meta := map[string]any{
			"cobol_kind": "program", "cobol_name_key": "INNER",
			"cobol_nested": true, "cobol_container": "r1/b.cbl::OUTER",
		}
		if stamped {
			meta["cobol_nested_unresolved"] = true
		}
		seedCobolNode(g, inner, graph.KindFunction, meta)
	}
	call := func(g graph.Store) {
		meta := cobolAt(cobolCallMeta(target, "'INNER'"), 4, 400)
		meta["cobol_caller_program"] = caller
		cobolEdge(g, caller, 5, graph.EdgeCalls, target, meta)
	}
	findings := func(g graph.Store) int {
		count := 0
		for n := range g.NodesByKind(graph.KindFinding) {
			if n != nil {
				count++
			}
		}
		return count
	}
	assertContainmentUnresolved := func(t *testing.T, g graph.Store) {
		t.Helper()
		e := cobolEdgeAt(t, g, caller, 5)
		assert.Equal(t, target, e.To, "the edge stays on its unresolved target")
		assert.Equal(t, "containment_unresolved", e.Meta["cobol_unresolved_reason"])
		assert.NotContains(t, e.Meta, "cobol_candidate_ids")
		assert.Empty(t, cobolMissingNodes(g), "no not-obtained placeholder")
		assert.Zero(t, findings(g), "no finding of any class")
	}

	t.Run("stamped", func(t *testing.T) {
		cobolOnStores(t, func(t *testing.T, g graph.Store) {
			seed(g, true)
			call(g)
			New(g).ReconcileCobol(nil)
			assertContainmentUnresolved(t, g)
			New(g).ReconcileCobol(nil)
			assertContainmentUnresolved(t, g)
		})
	})

	t.Run("unstamped_missing", func(t *testing.T) {
		cobolOnStores(t, func(t *testing.T, g graph.Store) {
			seed(g, false)
			call(g)
			New(g).ReconcileCobol(nil)
			e := cobolEdgeAt(t, g, caller, 5)
			assert.Equal(t, "r1::cobol-missing::program/INNER", e.To)
			assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
			assert.Len(t, cobolMissingNodes(g), 1)
			assert.Len(t, cobolExternalNodes(g), 1)
		})
	})

	// A re-extraction that stamps the nested program moves a missing CALL
	// off its placeholder, and its placeholder and finding go with it.
	t.Run("missing_then_stamped", func(t *testing.T) {
		cobolOnStores(t, func(t *testing.T, g graph.Store) {
			seed(g, false)
			call(g)
			New(g).ReconcileCobol(nil)
			require.Len(t, cobolExternalNodes(g), 1)
			seed(g, true)
			New(g).ReconcileCobol(nil)
			assertContainmentUnresolved(t, g)
		})
	})
}

// WR-06: a copybook whose analysis failed is still present, so a COPY of it
// binds to the extractor's identity-only file node and no
// EXTERNAL_UNRESOLVED finding claims it was never obtained; its
// analysis-failed parse finding stays.
func TestCobolReconcile_AnalysisFailedCopybookBinds(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const target = "unresolved::cobol::copybook::BROKENC"
	const copybook = "r1/copy/brokenc.cpy"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobolNode(g, copybook, graph.KindFile, map[string]any{
			"cobol_kind": "copybook", "cobol_name_key": "BROKENC",
			"cobol_name_norm": "cobol-name-norm/1", "prov_analysis_absence": "copybook_analysis_failed",
		})
		parse := copybook + "::" + graph.FindingSymbol("copy/brokenc.cpy", graph.FindingClassParse,
			"analysis-failed", "", graph.FindingEvidenceDigest("analysis-failed"))
		g.AddNode(&graph.Node{
			ID: parse, Kind: graph.KindFinding, Name: "analysis-failed", FilePath: copybook,
			Language: "cobol", RepoPrefix: "r1", Meta: map[string]any{
				graph.MetaEvidenceClass: graph.FindingClassParse, graph.MetaFindingKind: "analysis-failed",
				graph.MetaFindingReason: "analysis-error",
			},
		})
		g.AddEdge(&graph.Edge{From: parse, To: copybook, Kind: graph.EdgeMemberOf, FilePath: copybook})
		cobolEdge(g, caller, 4, graph.EdgeImports, target, cobolAt(cobolCopyMeta("copy", target), 3, 300))

		New(g).ReconcileCobol(nil)

		e := cobolEdgeAt(t, g, caller, 4)
		assert.Equal(t, copybook, e.To, "the COPY binds to the present copybook")
		assert.NotContains(t, e.Meta, "cobol_unresolved_reason")
		assert.Empty(t, cobolExternalNodes(g), "no EXTERNAL_UNRESOLVED finding")
		assert.Empty(t, cobolMissingNodes(g), "no not-obtained placeholder")
		require.NotNil(t, g.GetNode(parse), "the analysis-failed parse finding stays")
	})
}
