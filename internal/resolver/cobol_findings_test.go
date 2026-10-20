package resolver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
)

// cobolOnStores runs fn on an in-memory graph and on a fresh SQLite store.
func cobolOnStores(t *testing.T, fn func(t *testing.T, g graph.Store)) {
	t.Run("memory", func(t *testing.T) { fn(t, graph.New()) })
	t.Run("sqlite", func(t *testing.T) {
		s, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, s.Close()) })
		fn(t, s)
	})
}

// cobolMissingNodes returns every missing-artifact placeholder in g by ID.
func cobolMissingNodes(g graph.Store) map[string]*graph.Node {
	out := map[string]*graph.Node{}
	for n := range g.NodesByKind(graph.KindResource) {
		if n != nil && n.Meta["cobol_kind"] == "missing_artifact" {
			out[n.ID] = n
		}
	}
	return out
}

// cobolCopyMeta is the Meta the extractor stamps on a COPY-family edge of
// form copy, inc, or sql_include: the form and its original target.
func cobolCopyMeta(form, placeholder string) map[string]any {
	return map[string]any{"cobol_copy_form": form, "cobol_placeholder": placeholder}
}

// cobolIDMSMeta is the Meta the extractor stamps on a COPY IDMS edge: no
// cobol_placeholder, the dictionary-member reason from the start.
func cobolIDMSMeta() map[string]any {
	return map[string]any{"cobol_copy_form": "idms", "cobol_unresolved_reason": "idms_dictionary_member"}
}

func TestCobolReconcile_PlaceholderEdgeTargets(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const callNo = "unresolved::cobol::program::NOPGM"
	const copyNo = "unresolved::cobol::copybook::NOCPY"
	const incNo = "unresolved::cobol::copybook::INCMEM"
	const incYes = "unresolved::cobol::copybook::PRESENT"
	const idms = "unresolved::cobol::copybook::idms/RECORD/EMP-REC"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "r1", "r1/copy/present.cpy", graph.KindFile, "copybook", "PRESENT")
		for _, line := range []int{1, 2, 3} { // CALL, LINK, XCTL
			cobolEdge(g, caller, line, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		}
		cobolEdge(g, caller, 4, graph.EdgeImports, copyNo, cobolCopyMeta("copy", copyNo))
		cobolEdge(g, caller, 5, graph.EdgeImports, incNo, cobolCopyMeta("inc", incNo))
		cobolEdge(g, caller, 6, graph.EdgeImports, incYes, cobolCopyMeta("inc", incYes))
		cobolEdge(g, caller, 7, graph.EdgeImports, idms, cobolIDMSMeta())

		New(g).ReconcileCobol(nil)

		want := map[int][2]string{
			1: {"r1/cobol-missing::program/NOPGM", "missing"},
			2: {"r1/cobol-missing::program/NOPGM", "missing"},
			3: {"r1/cobol-missing::program/NOPGM", "missing"},
			4: {"r1/cobol-missing::copybook/NOCPY", "missing"},
			5: {"r1/cobol-missing::inc_member/INCMEM", "missing"},
			7: {"r1/cobol-missing::idms_dictionary_member/RECORD/EMP-REC", "idms_dictionary_member"},
		}
		for line, w := range want {
			e := cobolEdgeAt(t, g, caller, line)
			assert.Equal(t, w[0], e.To, "line %d targets its placeholder node", line)
			assert.Equal(t, w[1], e.Meta["cobol_unresolved_reason"], "line %d keeps its reason", line)
			assert.False(t, e.CrossRepo, "line %d", line)
			require.NotNil(t, g.GetNode(w[0]), "line %d placeholder node exists", line)
		}
		bound := cobolEdgeAt(t, g, caller, 6)
		assert.Equal(t, "r1/copy/present.cpy", bound.To, "an -INC member present as a copybook binds (D-17)")
		assert.NotContains(t, bound.Meta, "cobol_unresolved_reason")

		missing := cobolMissingNodes(g)
		assert.Len(t, missing, 4, "one placeholder per (repository, type, key)")
		assert.NotContains(t, missing, "r1/cobol-missing::inc_member/PRESENT")
	})
}

func TestCobolReconcile_PlaceholderNotForBoundAmbiguousDynamicBuiltin(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const orphan = "r1/orphan.cbl::ORPHAN"
	const callSub = "unresolved::cobol::program::SUBPGM"
	const callDup = "unresolved::cobol::program::DUP"
	const callNo = "unresolved::cobol::program::NOPGM"
	const dynamic = "unresolved::cobol::dynamic::program::r1/main.cbl::MAINPGM::WS-PGM"
	const sqlca = "unresolved::cobol::copybook::SQLCA"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "", orphan, graph.KindFunction, "program", "ORPHAN")
		seedCobol(g, "r1", "r1/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		seedCobol(g, "r1", "r1/d1.cbl::DUP", graph.KindFunction, "program", "DUP")
		seedCobol(g, "r1", "r1/d2.cbl::DUP", graph.KindFunction, "program", "DUP")
		cobolEdge(g, caller, 1, graph.EdgeCalls, callSub, placeholderMeta(callSub))
		cobolEdge(g, caller, 2, graph.EdgeCalls, callDup, placeholderMeta(callDup))
		cobolEdge(g, caller, 3, graph.EdgeCalls, dynamic, map[string]any{"cobol_unresolved_reason": "dynamic"})
		builtin := cobolCopyMeta("sql_include", sqlca)
		builtin["sql_include_builtin"] = true
		cobolEdge(g, caller, 4, graph.EdgeImports, sqlca, builtin)
		cobolEdge(g, orphan, 5, graph.EdgeCalls, callNo, placeholderMeta(callNo))

		New(g).ReconcileCobol(nil)

		assert.Equal(t, "r1/sub.cbl::SUBPGM", cobolEdgeAt(t, g, caller, 1).To, "bound")
		amb := cobolEdgeAt(t, g, caller, 2)
		assert.Equal(t, callDup, amb.To, "ambiguous keeps its unresolved target")
		assert.Equal(t, "ambiguous", amb.Meta["cobol_unresolved_reason"])
		dyn := cobolEdgeAt(t, g, caller, 3)
		assert.Equal(t, dynamic, dyn.To, "dynamic is never touched")
		assert.Equal(t, "dynamic", dyn.Meta["cobol_unresolved_reason"])
		sql := cobolEdgeAt(t, g, caller, 4)
		assert.Equal(t, sqlca, sql.To, "a builtin include keeps its unresolved target (D-19)")
		assert.Equal(t, "missing", sql.Meta["cobol_unresolved_reason"])
		unowned := cobolEdgeAt(t, g, orphan, 5)
		assert.Equal(t, callNo, unowned.To, "an unowned caller gets no placeholder")
		assert.Equal(t, "caller_unowned", unowned.Meta["cobol_unresolved_reason"])

		assert.Empty(t, cobolMissingNodes(g), "no placeholder for a bound, ambiguous, dynamic, unowned, or builtin reference")
	})
}

func TestCobolReconcile_PlaceholderIDMSSecondPass(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const idms = "unresolved::cobol::copybook::idms/RECORD/EMP-REC"
	const placeholder = "r1/cobol-missing::idms_dictionary_member/RECORD/EMP-REC"

	t.Run("unchanged", func(t *testing.T) {
		opened, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
		require.NoError(t, err)
		store := &cobolWriteCounter{Store: opened}
		t.Cleanup(func() { require.NoError(t, opened.Close()) })
		seedCobol(store, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		cobolEdge(store, caller, 7, graph.EdgeImports, idms, cobolIDMSMeta())

		New(store).ReconcileCobol(nil)
		require.Equal(t, placeholder, cobolEdgeAt(t, store, caller, 7).To)
		store.reset()
		New(store).ReconcileCobol(nil)

		e := cobolEdgeAt(t, store, caller, 7)
		assert.Equal(t, placeholder, e.To)
		assert.Equal(t, "idms_dictionary_member", e.Meta["cobol_unresolved_reason"])
		assert.Zero(t, store.writes(), "a second pass over the IDMS edge writes nothing")
	})

	t.Run("revisited", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		cobolEdge(g, caller, 7, graph.EdgeImports, idms, cobolIDMSMeta())
		New(g).ReconcileCobol(nil)
		e := cobolEdgeAt(t, g, caller, 7)
		require.Equal(t, placeholder, e.To)

		// A stale reason proves the second pass reaches the edge through the
		// placeholder's in-edges: only a visited edge is restamped.
		e.Meta["cobol_unresolved_reason"] = "stale"
		New(g).ReconcileCobol(nil)

		e = cobolEdgeAt(t, g, caller, 7)
		assert.Equal(t, placeholder, e.To)
		assert.Equal(t, "idms_dictionary_member", e.Meta["cobol_unresolved_reason"])
	})
}

func TestCobolReconcile_PlaceholderShape(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const callNo = "unresolved::cobol::program::NOPGM"
	const id = "r1/cobol-missing::program/NOPGM"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		g.AddNode(&graph.Node{
			ID: caller, Kind: graph.KindFunction, Name: "MAINPGM", QualName: "MAINPGM",
			FilePath: "r1/main.cbl", StartLine: 1, EndLine: 9, Language: "cobol",
			RepoPrefix: "r1", WorkspaceID: "ws1", ProjectID: "proj1",
			Meta: map[string]any{"cobol_kind": "program", "cobol_name_key": "MAINPGM", "cobol_nested": false},
		})
		meta := placeholderMeta(callNo)
		meta["cobol_target_raw"] = "'nopgm'"
		meta["prov_start_row"] = 4
		meta["prov_parser_tool_id"] = "tool"
		cobolEdge(g, caller, 5, graph.EdgeCalls, callNo, meta)

		New(g).ReconcileCobol(nil)

		n := g.GetNode(id)
		require.NotNil(t, n)
		assert.Equal(t, graph.KindResource, n.Kind)
		assert.Equal(t, id, n.FilePath, "the placeholder owns its own virtual path")
		assert.Equal(t, "NOPGM", n.Name)
		assert.Equal(t, "cobol", n.Language)
		assert.Equal(t, "r1", n.RepoPrefix)
		assert.Equal(t, "ws1", n.WorkspaceID)
		assert.Equal(t, "proj1", n.ProjectID)
		assert.Zero(t, n.StartLine)
		assert.Zero(t, n.EndLine)
		assert.Equal(t, map[string]any{
			"cobol_kind":          "missing_artifact",
			"artifact_type":       "program",
			"artifact_key":        "NOPGM",
			"artifact_status":     "not_obtained",
			"synthetic":           true,
			"prov_evidence_class": "EXTERNAL_UNRESOLVED",
			"prov_range_exact":    false,
			"prov_range_absence":  "not_obtained",
			"cobol_name_norm":     "cobol-name-norm/1",
		}, n.Meta, "exactly the D-11 Meta")
		for _, k := range []string{"prov_start_row", "prov_parser_tool_id", "cobol_target_raw", "cobol_name_key"} {
			assert.NotContains(t, n.Meta, k, "no parser stamp, range, or as-written spelling")
		}
	})
}

func TestCobolReconcile_PlaceholderIdempotent(t *testing.T) {
	opened, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
	require.NoError(t, err)
	store := &cobolWriteCounter{Store: opened}
	t.Cleanup(func() { require.NoError(t, opened.Close()) })

	const caller = "r1/main.cbl::MAINPGM"
	seedCobol(store, "r1", caller, graph.KindFunction, "program", "MAINPGM")
	seedCobol(store, "r1", "r1/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
	for line, to := range map[int]string{
		1: "unresolved::cobol::program::NOPGM",
		2: "unresolved::cobol::program::SUBPGM",
	} {
		cobolEdge(store, caller, line, graph.EdgeCalls, to, placeholderMeta(to))
	}
	cobolEdge(store, caller, 3, graph.EdgeImports, "unresolved::cobol::copybook::NOCPY",
		cobolCopyMeta("copy", "unresolved::cobol::copybook::NOCPY"))
	cobolEdge(store, caller, 4, graph.EdgeImports, "unresolved::cobol::copybook::INCMEM",
		cobolCopyMeta("inc", "unresolved::cobol::copybook::INCMEM"))
	cobolEdge(store, caller, 5, graph.EdgeImports, "unresolved::cobol::copybook::idms/RECORD/EMP-REC", cobolIDMSMeta())

	changed := map[string]struct{}{"r1": {}}
	New(store).ReconcileCobol(changed)
	require.Len(t, cobolMissingNodes(store), 4)
	require.NotZero(t, store.adds, "the first pass adds the placeholders")
	store.reset()
	New(store).ReconcileCobol(changed)

	assert.Zero(t, store.adds, "no second AddBatch")
	assert.Zero(t, store.reindexes, "no second ReindexEdges")
	assert.Zero(t, store.persists, "no second Meta write")
	assert.Zero(t, store.evicts, "no EvictFile")
	assert.Len(t, cobolMissingNodes(store), 4)
}
