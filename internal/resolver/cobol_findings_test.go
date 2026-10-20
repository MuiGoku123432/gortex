package resolver

import (
	"path/filepath"
	"sort"
	"strings"
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

// seedCobolFinding adds a finding node of class owned by filePath, with its
// ID from graph.FindingSymbol, and returns the ID.
func seedCobolFinding(g graph.Store, repo, filePath, container, class string) string {
	id := filePath + "::" + graph.FindingSymbol(container, class, "unparsed-tail", "statement", graph.FindingEvidenceDigest("ZQX"))
	g.AddNode(&graph.Node{
		ID: id, Kind: graph.KindFinding, Name: "unparsed-tail", FilePath: filePath,
		Language: "cobol", RepoPrefix: repo, Meta: map[string]any{graph.MetaEvidenceClass: class},
	})
	return id
}

func TestCobolReconcile_PlaceholderNeverCandidate(t *testing.T) {
	const callNo = "unresolved::cobol::program::NOPGM"
	const copyNo = "unresolved::cobol::copybook::NOCPY"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", "r1/a.cbl::APGM", graph.KindFunction, "program", "APGM")
		seedCobol(g, "r1", "r1/b.cbl::BPGM", graph.KindFunction, "program", "BPGM")
		cobolEdge(g, "r1/a.cbl::APGM", 1, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		cobolEdge(g, "r1/a.cbl::APGM", 2, graph.EdgeImports, copyNo, cobolCopyMeta("copy", copyNo))
		New(g).ReconcileCobol(nil)
		require.Len(t, cobolMissingNodes(g), 2)

		// Only the placeholders exist for NOPGM and NOCPY now.
		cobolEdge(g, "r1/b.cbl::BPGM", 3, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		cobolEdge(g, "r1/b.cbl::BPGM", 4, graph.EdgeImports, copyNo, cobolCopyMeta("copy", copyNo))
		New(g).ReconcileCobol(nil)

		for line, to := range map[int]string{3: "r1/cobol-missing::program/NOPGM", 4: "r1/cobol-missing::copybook/NOCPY"} {
			e := cobolEdgeAt(t, g, "r1/b.cbl::BPGM", line)
			assert.Equal(t, to, e.To, "line %d", line)
			assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"], "line %d: a placeholder is never a candidate", line)
			assert.NotContains(t, e.Meta, "cobol_candidate_ids", "line %d", line)
		}
		assert.Len(t, cobolMissingNodes(g), 2, "the second caller shares its repository's placeholders")
	})
}

func TestCobolReconcile_PlaceholderSweep(t *testing.T) {
	const callNo = "unresolved::cobol::program::NOPGM"
	const placeholder = "r1/cobol-missing::program/NOPGM"
	seed := func(g graph.Store, callers ...string) {
		for i, c := range callers {
			seedCobol(g, "r1", c, graph.KindFunction, "program", c[len(c)-4:])
			cobolEdge(g, c, i+1, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		}
		New(g).ReconcileCobol(nil)
	}

	t.Run("last_reference_removed", func(t *testing.T) {
		cobolOnStores(t, func(t *testing.T, g graph.Store) {
			seed(g, "r1/a.cbl::APGM")
			require.NotNil(t, g.GetNode(placeholder))
			require.True(t, g.RemoveEdge("r1/a.cbl::APGM", placeholder, graph.EdgeCalls))
			New(g).ReconcileCobol(nil)
			assert.Nil(t, g.GetNode(placeholder), "an unreferenced placeholder is evicted")
		})
	})

	t.Run("still_referenced", func(t *testing.T) {
		cobolOnStores(t, func(t *testing.T, g graph.Store) {
			seed(g, "r1/a.cbl::APGM", "r1/b.cbl::BPGM")
			require.True(t, g.RemoveEdge("r1/a.cbl::APGM", placeholder, graph.EdgeCalls))
			New(g).ReconcileCobol(nil)
			assert.NotNil(t, g.GetNode(placeholder), "one code reference keeps the placeholder")
			assert.Equal(t, placeholder, cobolEdgeAt(t, g, "r1/b.cbl::BPGM", 2).To)
		})
	})

	t.Run("finding_only", func(t *testing.T) {
		cobolOnStores(t, func(t *testing.T, g graph.Store) {
			seed(g, "r1/a.cbl::APGM")
			finding := seedCobolFinding(g, "r1", "r1/a.cbl", "a.cbl::APGM", graph.FindingClassExternal)
			g.AddEdge(&graph.Edge{From: finding, To: placeholder, Kind: graph.EdgeReferences, FilePath: "r1/a.cbl"})
			require.True(t, g.RemoveEdge("r1/a.cbl::APGM", placeholder, graph.EdgeCalls))
			New(g).ReconcileCobol(nil)
			assert.Nil(t, g.GetNode(placeholder), "a finding's references edge never keeps its placeholder")
		})
	})

	t.Run("code_references_kept", func(t *testing.T) {
		cobolOnStores(t, func(t *testing.T, g graph.Store) {
			// EXEC CICS LOAD PROGRAM('NOPGM'): a literal program reference
			// that is not a call.
			seedCobol(g, "r1", "r1/a.cbl::APGM", graph.KindFunction, "program", "APGM")
			cobolEdge(g, "r1/a.cbl::APGM", 1, graph.EdgeReferences, callNo, placeholderMeta(callNo))
			New(g).ReconcileCobol(nil)
			New(g).ReconcileCobol(nil)
			assert.NotNil(t, g.GetNode(placeholder), "a code references edge keeps the placeholder")
			assert.Equal(t, placeholder, cobolEdgeAt(t, g, "r1/a.cbl::APGM", 1).To)
		})
	})

	t.Run("outside_reconcile_set", func(t *testing.T) {
		g := graph.New()
		seed(g, "r1/a.cbl::APGM")
		require.True(t, g.RemoveEdge("r1/a.cbl::APGM", placeholder, graph.EdgeCalls))
		seedCobol(g, "r2", "r2/z.cbl::ZPGM", graph.KindFunction, "program", "ZPGM")
		New(g).ReconcileCobol(map[string]struct{}{"r2": {}})
		assert.NotNil(t, g.GetNode(placeholder), "a placeholder outside the reconcile set is not swept")
		New(g).ReconcileCobol(map[string]struct{}{"r1": {}})
		assert.Nil(t, g.GetNode(placeholder))
	})
}

func TestCobolReconcile_PlaceholderRebindOnRetrieve(t *testing.T) {
	const callNo = "unresolved::cobol::program::NOPGM"
	const copyNo = "unresolved::cobol::copybook::NOCPY"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", "r1/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "r2", "r2/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
		cobolEdge(g, "r1/main.cbl::MAINPGM", 1, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		cobolEdge(g, "r1/main.cbl::MAINPGM", 2, graph.EdgeImports, copyNo, cobolCopyMeta("copy", copyNo))
		cobolEdge(g, "r2/main.cbl::MAINPGM", 1, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		parse := seedCobolFinding(g, "r1", "r1/main.cbl", "main.cbl::MAINPGM", graph.FindingClassParse)
		New(g).ReconcileCobol(nil)
		require.Len(t, cobolMissingNodes(g), 3)

		// The missing program is retrieved into r1.
		seedCobol(g, "r1", "r1/nopgm.cbl::NOPGM", graph.KindFunction, "program", "NOPGM")
		New(g).ReconcileCobol(map[string]struct{}{"r1": {}})

		e := cobolEdgeAt(t, g, "r1/main.cbl::MAINPGM", 1)
		assert.Equal(t, "r1/nopgm.cbl::NOPGM", e.To, "the edge rebinds to the retrieved program")
		assert.NotContains(t, e.Meta, "cobol_unresolved_reason")
		assert.Nil(t, g.GetNode("r1/cobol-missing::program/NOPGM"), "its placeholder is removed")
		assert.NotNil(t, g.GetNode("r2/cobol-missing::program/NOPGM"), "another repository's placeholder stays")
		assert.Equal(t, "r2/cobol-missing::program/NOPGM", cobolEdgeAt(t, g, "r2/main.cbl::MAINPGM", 1).To)
		assert.NotNil(t, g.GetNode("r1/cobol-missing::copybook/NOCPY"), "an unrelated placeholder stays")
		assert.NotNil(t, g.GetNode(parse), "a parse finding is never touched")
		assert.NotNil(t, g.GetNode("r1/nopgm.cbl::NOPGM"))
	})
}

// A placeholder-targeted edge is not pending, so a ResolveAll with nothing
// pending must still run the COBOL reconcile to bind a retrieved target.
func TestCobolReconcile_PlaceholderResolveAllNoPending(t *testing.T) {
	const callNo = "unresolved::cobol::program::NOPGM"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", "r1/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
		cobolEdge(g, "r1/main.cbl::MAINPGM", 1, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		New(g).ResolveAll()
		require.Equal(t, "r1/cobol-missing::program/NOPGM", cobolEdgeAt(t, g, "r1/main.cbl::MAINPGM", 1).To)

		seedCobol(g, "r1", "r1/nopgm.cbl::NOPGM", graph.KindFunction, "program", "NOPGM")
		New(g).ResolveAll()

		assert.Equal(t, "r1/nopgm.cbl::NOPGM", cobolEdgeAt(t, g, "r1/main.cbl::MAINPGM", 1).To)
		assert.Empty(t, cobolMissingNodes(g))
	})
}

func TestCobolReconcile_PlaceholderPerRepository(t *testing.T) {
	const callNo = "unresolved::cobol::program::NOPGM"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		for _, repo := range []string{"r1", "r2"} {
			g.AddNode(&graph.Node{
				ID: repo + "/main.cbl::MAINPGM", Kind: graph.KindFunction, Name: "MAINPGM",
				FilePath: repo + "/main.cbl", Language: "cobol", RepoPrefix: repo,
				WorkspaceID: "ws-" + repo, ProjectID: "proj-" + repo,
				Meta: map[string]any{"cobol_kind": "program", "cobol_name_key": "MAINPGM", "cobol_nested": false},
			})
			cobolEdge(g, repo+"/main.cbl::MAINPGM", 1, graph.EdgeCalls, callNo, placeholderMeta(callNo))
		}

		New(g).ReconcileCobol(nil)

		missing := cobolMissingNodes(g)
		require.Len(t, missing, 2, "one placeholder per referencing repository")
		for _, repo := range []string{"r1", "r2"} {
			id := repo + "/cobol-missing::program/NOPGM"
			n := missing[id]
			require.NotNil(t, n, id)
			assert.Equal(t, repo, n.RepoPrefix)
			assert.Equal(t, "ws-"+repo, n.WorkspaceID)
			assert.Equal(t, "proj-"+repo, n.ProjectID)
			assert.Equal(t, id, cobolEdgeAt(t, g, repo+"/main.cbl::MAINPGM", 1).To, "%s targets its own placeholder", repo)
		}
	})
}

func TestCobolReconcile_PlaceholderPrefixDiagnostics(t *testing.T) {
	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	for _, repo := range []string{"r1", "r2"} {
		caller := repo + "/main.cbl::MAINPGM"
		seedCobol(store, repo, caller, graph.KindFunction, "program", "MAINPGM")
		cobolEdge(store, caller, 1, graph.EdgeCalls, "unresolved::cobol::program::NOPGM",
			placeholderMeta("unresolved::cobol::program::NOPGM"))
		cobolEdge(store, caller, 2, graph.EdgeImports, "unresolved::cobol::copybook::INCMEM",
			cobolCopyMeta("inc", "unresolved::cobol::copybook::INCMEM"))
		cobolEdge(store, caller, 3, graph.EdgeImports, "unresolved::cobol::copybook::idms/RECORD/EMP-REC", cobolIDMSMeta())
	}

	New(store).ReconcileCobol(nil)
	require.Len(t, cobolMissingNodes(store), 6)

	diag, ok := graph.ReadPrefixDiagnostics(store, 10)
	require.True(t, ok)
	assert.Zero(t, diag.UnownedCodeNodes, "unowned: %v", diag.UnownedSamples)
	assert.Zero(t, diag.MisprefixedNodes, "misprefixed: %v", diag.MisprefixedSamples)
	assert.GreaterOrEqual(t, diag.OwnedCodeNodes, 8, "the placeholders are audited as owned nodes")
}

// cobolCallMeta is the Meta the extractor stamps on a literal CALL edge:
// the call form, the as-written target, and the original target.
func cobolCallMeta(target, raw string) map[string]any {
	return map[string]any{
		"cobol_call": "literal", "cobol_target_raw": raw,
		"cobol_name_norm": "cobol-name-norm/1", "cobol_placeholder": target,
	}
}

// cobolAt adds the occurrence range keys the extractor stamps on every
// relationship edge, starting at 0-based row and byte start, to meta.
func cobolAt(meta map[string]any, row, start int) map[string]any {
	meta["prov_start_row"], meta["prov_start_column"] = row, 11
	meta["prov_end_row"], meta["prov_end_column"] = row, 25
	meta["prov_start_byte"], meta["prov_end_byte"] = start, start+14
	meta["prov_range_exact"] = true
	return meta
}

// cobolExternalNodes returns every EXTERNAL_UNRESOLVED finding node by ID.
func cobolExternalNodes(g graph.Store) map[string]*graph.Node {
	out := map[string]*graph.Node{}
	for n := range g.NodesByKind(graph.KindFinding) {
		if n != nil && n.Meta[graph.MetaEvidenceClass] == graph.FindingClassExternal {
			out[n.ID] = n
		}
	}
	return out
}

// cobolExternalID is the unsuffixed external finding ID of a reference in
// filePath from container (repository-relative) to an artifact of
// artifactType, whose evidence parts are joined with NUL.
func cobolExternalID(filePath, container, artifactType string, evidence ...string) string {
	digest := graph.FindingEvidenceDigest(strings.Join(evidence, "\x00"))
	return filePath + "::" + graph.FindingSymbol(container, graph.FindingClassExternal, "external-reference", artifactType, digest)
}

// cobolFindingTargets returns a finding's out-edge targets by edge kind and
// fails unless it has exactly one member_of and one references edge.
func cobolFindingTargets(t *testing.T, g graph.Store, id string) map[graph.EdgeKind]string {
	t.Helper()
	out := map[graph.EdgeKind]string{}
	for _, e := range g.GetOutEdges(id) {
		_, dup := out[e.Kind]
		require.False(t, dup, "%s has one %s edge", id, e.Kind)
		out[e.Kind] = e.To
	}
	require.Len(t, out, 2, "%s has a member_of and a references edge: %v", id, out)
	return out
}

func TestCobolReconcile_ExternalMissingCall(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const callNo = "unresolved::cobol::program::NOPGM"
	const placeholder = "r1/cobol-missing::program/NOPGM"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		// CALL, LINK, and XCTL, seeded out of source order.
		for _, line := range []int{3, 1, 2} {
			cobolEdge(g, caller, line, graph.EdgeCalls, callNo, cobolAt(cobolCallMeta(callNo, "'NOPGM'"), line-1, line*100))
		}

		New(g).ReconcileCobol(nil)

		found := cobolExternalNodes(g)
		require.Len(t, found, 3, "one finding per reference occurrence")
		base := cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM", "program", "literal", "'NOPGM'")
		for k, id := range []string{base, base + "#2", base + "#3"} {
			n := found[id]
			require.NotNil(t, n, "finding %s", id)
			assert.Equal(t, graph.KindFinding, n.Kind)
			assert.Equal(t, k+1, n.StartLine, "%s is the occurrence at byte order %d", id, k+1)
			assert.Equal(t, "missing", n.Meta[graph.MetaFindingReason])
			assert.Equal(t, "r1", n.RepoPrefix)
			assert.Equal(t, map[graph.EdgeKind]string{
				graph.EdgeMemberOf: caller, graph.EdgeReferences: placeholder,
			}, cobolFindingTargets(t, g, id))
		}
	})
}

func TestCobolReconcile_ExternalMissingCopyAndInc(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const copyNo = "unresolved::cobol::copybook::NOCPY"
	const incNo = "unresolved::cobol::copybook::INCMEM"
	const incYes = "unresolved::cobol::copybook::PRESENT"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "r1", "r1/copy/present.cpy", graph.KindFile, "copybook", "PRESENT")
		withRaw := func(meta map[string]any, raw string) map[string]any {
			meta["cobol_copy_member_raw"] = raw
			return meta
		}
		cobolEdge(g, caller, 4, graph.EdgeImports, copyNo, withRaw(cobolCopyMeta("copy", copyNo), "NOCPY"))
		cobolEdge(g, caller, 5, graph.EdgeImports, incNo, withRaw(cobolCopyMeta("inc", incNo), "INCMEM"))
		cobolEdge(g, caller, 6, graph.EdgeImports, incYes, withRaw(cobolCopyMeta("inc", incYes), "PRESENT"))

		New(g).ReconcileCobol(nil)

		found := cobolExternalNodes(g)
		require.Len(t, found, 2, "a bound -INC member has no finding (D-17)")
		want := map[string][3]string{
			cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM", "copybook", "copy", "NOCPY"): {
				"copybook", "NOCPY", "r1/cobol-missing::copybook/NOCPY",
			},
			cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM", "inc_member", "inc", "INCMEM"): {
				"inc_member", "INCMEM", "r1/cobol-missing::inc_member/INCMEM",
			},
		}
		for id, w := range want {
			n := found[id]
			require.NotNil(t, n, "finding %s", id)
			assert.Equal(t, w[0], n.Meta[graph.MetaFindingArtifactType])
			assert.Equal(t, w[1], n.Meta[graph.MetaFindingTargetKey])
			assert.Equal(t, "missing", n.Meta[graph.MetaFindingReason])
			assert.Equal(t, w[2], cobolFindingTargets(t, g, id)[graph.EdgeReferences])
		}
	})
}

func TestCobolReconcile_ExternalIDMS(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const idms = "unresolved::cobol::copybook::idms/RECORD/EMP-REC"
	opened, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
	require.NoError(t, err)
	store := &cobolWriteCounter{Store: opened}
	t.Cleanup(func() { require.NoError(t, opened.Close()) })
	seedCobol(store, "r1", caller, graph.KindFunction, "program", "MAINPGM")
	meta := cobolIDMSMeta()
	meta["cobol_copy_member_raw"], meta["cobol_copy_idms_kind"] = "EMP-REC", "RECORD"
	cobolEdge(store, caller, 7, graph.EdgeImports, idms, meta)

	New(store).ReconcileCobol(nil)
	id := cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM", "idms_dictionary_member", "idms", "RECORD", "EMP-REC")
	found := cobolExternalNodes(store)
	require.Len(t, found, 1)
	n := found[id]
	require.NotNil(t, n, "finding %s", id)
	assert.Equal(t, "idms_dictionary_member", n.Meta[graph.MetaFindingReason])
	assert.Equal(t, "idms_dictionary_member", n.Meta[graph.MetaFindingArtifactType])
	assert.Equal(t, "RECORD/EMP-REC", n.Meta[graph.MetaFindingTargetKey])
	assert.Equal(t, "r1/cobol-missing::idms_dictionary_member/RECORD/EMP-REC",
		cobolFindingTargets(t, store, id)[graph.EdgeReferences])

	store.reset()
	New(store).ReconcileCobol(nil)
	assert.Equal(t, []string{id}, mapKeys(cobolExternalNodes(store)), "a second pass keeps the ID")
	assert.Zero(t, store.writes(), "a second pass writes nothing")
}

// mapKeys returns the keys of m in sorted order.
func mapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestCobolReconcile_ExternalNotForBoundAmbiguousDynamicBuiltin(t *testing.T) {
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
		cobolEdge(g, caller, 1, graph.EdgeCalls, callSub, cobolCallMeta(callSub, "'SUBPGM'"))
		cobolEdge(g, caller, 2, graph.EdgeCalls, callDup, cobolCallMeta(callDup, "'DUP'"))
		cobolEdge(g, caller, 3, graph.EdgeCalls, dynamic, map[string]any{
			"cobol_call": "dynamic", "cobol_target_raw": "WS-PGM", "cobol_unresolved_reason": "dynamic",
		})
		builtin := cobolCopyMeta("sql_include", sqlca)
		builtin["sql_include_builtin"], builtin["cobol_copy_member_raw"] = true, "SQLCA"
		cobolEdge(g, caller, 4, graph.EdgeImports, sqlca, builtin)
		cobolEdge(g, orphan, 5, graph.EdgeCalls, callNo, cobolCallMeta(callNo, "'NOPGM'"))

		New(g).ReconcileCobol(nil)

		require.Equal(t, "caller_unowned", cobolEdgeAt(t, g, orphan, 5).Meta["cobol_unresolved_reason"])
		var findings []string
		for n := range g.NodesByKind(graph.KindFinding) {
			findings = append(findings, n.ID)
		}
		assert.Empty(t, findings, "no finding for a bound, ambiguous, dynamic, unowned, or builtin reference")
	})
}

func TestCobolReconcile_ExternalOrdinalAndLineShift(t *testing.T) {
	const para = "r1/main.cbl::MAINPGM.PARA-A"
	const callNo = "unresolved::cobol::program::NOPGM"
	// shift moves both CALLs down by rows lines and bytes bytes, as an edit
	// above them does, and returns each finding's ID by its start line.
	run := func(t *testing.T, g graph.Store, rows, bytes int) map[int]string {
		seedCobol(g, "r1", "r1/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "r1", para, graph.KindFunction, "paragraph", "PARA-A")
		// The later CALL is seeded first: byte order decides the ordinal.
		for _, at := range [][2]int{{9, 200}, {5, 100}} {
			meta := cobolAt(cobolCallMeta(callNo, "'NOPGM'"), at[0]+rows-1, at[1]+bytes)
			meta["cobol_caller_program"] = "r1/main.cbl::MAINPGM"
			cobolEdge(g, para, at[0]+rows, graph.EdgeCalls, callNo, meta)
		}
		New(g).ReconcileCobol(nil)
		out := map[int]string{}
		for id, n := range cobolExternalNodes(g) {
			out[n.StartLine] = id
		}
		return out
	}
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		base := cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM.PARA-A", "program", "literal", "'NOPGM'")
		assert.Equal(t, map[int]string{5: base, 9: base + "#2"}, run(t, g, 0, 0),
			"identical references in one container are ordered by start byte")
	})
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		base := cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM.PARA-A", "program", "literal", "'NOPGM'")
		assert.Equal(t, map[int]string{8: base, 12: base + "#2"}, run(t, g, 3, 57),
			"a line shift elsewhere in the file keeps both IDs")
	})
}

func TestCobolReconcile_ExternalContainerSeparation(t *testing.T) {
	const callNo = "unresolved::cobol::program::NOPGM"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", "r1/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
		for i, p := range []string{"PARA-A", "PARA-B"} {
			seedCobol(g, "r1", "r1/main.cbl::MAINPGM."+p, graph.KindFunction, "paragraph", p)
			cobolEdge(g, "r1/main.cbl::MAINPGM."+p, i+1, graph.EdgeCalls, callNo,
				cobolAt(cobolCallMeta(callNo, "'NOPGM'"), i, (i+1)*100))
		}

		New(g).ReconcileCobol(nil)

		found := cobolExternalNodes(g)
		require.Len(t, found, 2)
		containers := map[string]string{}
		for id, n := range found {
			assert.NotContains(t, id, "#", "identical references in two containers are both unsuffixed")
			containers[n.Meta[graph.MetaFindingContainer].(string)] = id
		}
		require.Len(t, containers, 2, "two containers, never one pairing group (D-03)")
		for _, p := range []string{"PARA-A", "PARA-B"} {
			id := containers["main.cbl::MAINPGM."+p]
			assert.Equal(t, cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM."+p, "program", "literal", "'NOPGM'"), id)
			assert.Equal(t, "r1/main.cbl::MAINPGM."+p, cobolFindingTargets(t, g, id)[graph.EdgeMemberOf])
		}
	})
}

func TestCobolReconcile_ExternalMetaContract(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const callNo = "unresolved::cobol::program::NOPGM"
	const raw = "'nopgm'"
	stamps := map[string]any{
		"prov_parser_tool_id": "tree-sitter-cobol-upgrade", "prov_parser_grammar_id": "cobol-grammar-x",
		"prov_extractor_version": "gortex-cobol-grammar/6", "prov_evidence_class": "DETERMINISTIC",
		"prov_observation_kind": "call_statement", "prov_affected": false,
	}
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		g.AddNode(&graph.Node{
			ID: caller, Kind: graph.KindFunction, Name: "MAINPGM", QualName: "MAINPGM",
			FilePath: "r1/main.cbl", StartLine: 1, EndLine: 20, Language: "cobol",
			RepoPrefix: "r1", WorkspaceID: "ws1", ProjectID: "proj1",
			Meta: map[string]any{"cobol_kind": "program", "cobol_name_key": "MAINPGM", "cobol_nested": false},
		})
		exact := cobolAt(cobolCallMeta(callNo, raw), 4, 300)
		exact["prov_end_row"] = 5
		exact["cobol_caller_program"], exact["cobol_statement_symbol"] = caller, "main.cbl::MAINPGM/stmt/3"
		absent := cobolCallMeta(callNo, "'NOPGM'")
		absent["prov_range_exact"], absent["prov_range_absence"] = false, "no_original_projection"
		for k, v := range stamps {
			exact[k], absent[k] = v, v
		}
		cobolEdge(g, caller, 5, graph.EdgeCalls, callNo, exact)
		cobolEdge(g, caller, 0, graph.EdgeCalls, callNo, absent)

		New(g).ReconcileCobol(nil)

		found := cobolExternalNodes(g)
		require.Len(t, found, 2)
		exactID := cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM", "program", "literal", raw)
		n := found[exactID]
		require.NotNil(t, n, "finding %s", exactID)
		symbol := exactID[len("r1/main.cbl::"):]
		assert.Equal(t, "NOPGM", n.Name, "Name is the normalized key")
		assert.Equal(t, symbol, n.QualName)
		assert.Equal(t, "r1/main.cbl", n.FilePath)
		assert.Equal(t, "cobol", n.Language)
		assert.Equal(t, [3]string{"r1", "ws1", "proj1"}, [3]string{n.RepoPrefix, n.WorkspaceID, n.ProjectID})
		assert.Equal(t, 5, n.StartLine)
		assert.Equal(t, 6, n.EndLine, "EndLine is prov_end_row + 1")
		want := map[string]any{
			"prov_start_row": 4, "prov_start_column": 11, "prov_end_row": 5, "prov_end_column": 25,
			"prov_start_byte": 300, "prov_end_byte": 314, "prov_range_exact": true,
			"prov_parser_tool_id": "tree-sitter-cobol-upgrade", "prov_parser_grammar_id": "cobol-grammar-x",
			"prov_extractor_version": "gortex-cobol-grammar/6",
			"prov_observation_kind":  "call_statement", "prov_affected": false,
			graph.MetaEvidenceClass:         graph.FindingClassExternal,
			graph.MetaFindingKind:           "external-reference",
			graph.MetaFindingReason:         "missing",
			graph.MetaFindingSeverity:       "medium",
			graph.MetaFindingSeverityRule:   graph.FindingSeverityRule,
			graph.MetaFindingScope:          "unavailable",
			graph.MetaFindingExpectedFamily: "program",
			graph.MetaFindingContainer:      "main.cbl::MAINPGM",
			graph.MetaFindingStatus:         graph.FindingStatusActive,
			graph.MetaFindingEvidenceDigest: graph.FindingEvidenceDigest("literal\x00" + raw),
			graph.MetaFindingEvidenceRule:   graph.FindingEvidenceRule,
			graph.MetaFindingIDRule:         graph.FindingIDRule,
			graph.MetaFindingArtifactType:   "program",
			graph.MetaFindingTargetKey:      "NOPGM",
		}
		assert.Equal(t, want, n.Meta, "the full Pattern 3 contract, prov_* copied verbatim")
		for _, k := range []string{graph.MetaFindingKind, graph.MetaFindingContainer, graph.MetaFindingEvidenceDigest} {
			assert.NotEmpty(t, n.Meta[k], "%s is never empty (D-03)", k)
		}

		for id, f := range found {
			for k, v := range f.Meta {
				assert.False(t, strings.HasPrefix(k, "cobol_"), "%s: no cobol_* key, found %s", id, k)
				assert.NotEqual(t, raw, v, "%s: the as-written spelling is hashed, never stored (%s)", id, k)
				switch v.(type) {
				case string, bool, int, float64, []string:
				default:
					t.Errorf("%s: Meta %s has disallowed type %T", id, k, v)
				}
			}
		}
		var other *graph.Node
		for id, f := range found {
			if id != exactID {
				other = f
			}
		}
		require.NotNil(t, other)
		assert.Equal(t, false, other.Meta["prov_range_exact"])
		assert.Equal(t, "no_original_projection", other.Meta["prov_range_absence"])
		assert.NotContains(t, other.Meta, "prov_start_row")
		assert.Zero(t, other.EndLine)
	})
}

func TestCobolReconcile_ExternalEvictedWhenUndesired(t *testing.T) {
	const caller = "r1/main.cbl::MAINPGM"
	const callNo = "unresolved::cobol::program::NOPGM"
	const copyNo = "unresolved::cobol::copybook::NOCPY"
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seedCobol(g, "r1", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "r2", "r2/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
		cobolEdge(g, caller, 1, graph.EdgeCalls, callNo, cobolCallMeta(callNo, "'NOPGM'"))
		copyMeta := cobolCopyMeta("copy", copyNo)
		copyMeta["cobol_copy_member_raw"] = "NOCPY"
		cobolEdge(g, caller, 2, graph.EdgeImports, copyNo, copyMeta)
		cobolEdge(g, "r2/main.cbl::MAINPGM", 1, graph.EdgeCalls, callNo, cobolCallMeta(callNo, "'NOPGM'"))
		parse := seedCobolFinding(g, "r1", "r1/main.cbl", "main.cbl::MAINPGM", graph.FindingClassParse)
		New(g).ReconcileCobol(nil)

		callID := cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM", "program", "literal", "'NOPGM'")
		copyID := cobolExternalID("r1/main.cbl", "main.cbl::MAINPGM", "copybook", "copy", "NOCPY")
		otherID := cobolExternalID("r2/main.cbl", "main.cbl::MAINPGM", "program", "literal", "'NOPGM'")
		require.ElementsMatch(t, []string{callID, copyID, otherID}, mapKeys(cobolExternalNodes(g)))

		// The missing program is retrieved into r1: the CALL binds.
		seedCobol(g, "r1", "r1/nopgm.cbl::NOPGM", graph.KindFunction, "program", "NOPGM")
		New(g).ReconcileCobol(map[string]struct{}{"r1": {}})

		require.Equal(t, "r1/nopgm.cbl::NOPGM", cobolEdgeAt(t, g, caller, 1).To)
		assert.ElementsMatch(t, []string{copyID, otherID}, mapKeys(cobolExternalNodes(g)),
			"only the bound reference's finding is evicted")
		assert.Nil(t, g.GetNode("r1/cobol-missing::program/NOPGM"), "then its placeholder is swept")
		assert.NotNil(t, g.GetNode(parse), "a parse finding is never touched")
		assert.NotNil(t, g.GetNode("r2/cobol-missing::program/NOPGM"))

		// The COPY is edited away.
		require.True(t, g.RemoveEdge(caller, "r1/cobol-missing::copybook/NOCPY", graph.EdgeImports))
		New(g).ReconcileCobol(map[string]struct{}{"r1": {}})

		assert.Equal(t, []string{otherID}, mapKeys(cobolExternalNodes(g)))
		assert.Nil(t, g.GetNode("r1/cobol-missing::copybook/NOCPY"))
		assert.NotNil(t, g.GetNode(parse))
	})
}

func TestCobolReconcile_ExternalIdempotent(t *testing.T) {
	opened, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
	require.NoError(t, err)
	store := &cobolWriteCounter{Store: opened}
	t.Cleanup(func() { require.NoError(t, opened.Close()) })

	const caller = "r1/main.cbl::MAINPGM"
	const callNo = "unresolved::cobol::program::NOPGM"
	seedCobol(store, "r1", caller, graph.KindFunction, "program", "MAINPGM")
	for _, line := range []int{1, 2} { // two identical CALLs: one ordinal
		cobolEdge(store, caller, line, graph.EdgeCalls, callNo, cobolAt(cobolCallMeta(callNo, "'NOPGM'"), line-1, line*100))
	}
	cobolEdge(store, caller, 3, graph.EdgeImports, "unresolved::cobol::copybook::NOCPY",
		cobolCopyMeta("copy", "unresolved::cobol::copybook::NOCPY"))
	cobolEdge(store, caller, 4, graph.EdgeImports, "unresolved::cobol::copybook::INCMEM",
		cobolCopyMeta("inc", "unresolved::cobol::copybook::INCMEM"))
	cobolEdge(store, caller, 5, graph.EdgeImports, "unresolved::cobol::copybook::idms/RECORD/EMP-REC", cobolIDMSMeta())

	changed := map[string]struct{}{"r1": {}}
	New(store).ReconcileCobol(changed)
	first := mapKeys(cobolExternalNodes(store))
	require.Len(t, first, 5)
	store.reset()
	New(store).ReconcileCobol(changed)

	assert.Zero(t, store.adds, "no second AddBatch")
	assert.Zero(t, store.reindexes, "no second ReindexEdges")
	assert.Zero(t, store.persists, "no second Meta write")
	assert.Zero(t, store.evicts, "no EvictFile")
	assert.Zero(t, store.findingEvicts, "no EvictFindingNodesByIDs")
	assert.Equal(t, first, mapKeys(cobolExternalNodes(store)))
}

func TestCobolReconcile_ExternalPerRepositoryPrefixDiagnostics(t *testing.T) {
	const callNo = "unresolved::cobol::program::NOPGM"
	seed := func(g graph.Store) {
		for _, repo := range []string{"r1", "r2"} {
			seedCobol(g, repo, repo+"/main.cbl::MAINPGM", graph.KindFunction, "program", "MAINPGM")
			cobolEdge(g, repo+"/main.cbl::MAINPGM", 1, graph.EdgeCalls, callNo, cobolCallMeta(callNo, "'NOPGM'"))
		}
		New(g).ReconcileCobol(nil)
	}
	cobolOnStores(t, func(t *testing.T, g graph.Store) {
		seed(g)
		found := cobolExternalNodes(g)
		require.Len(t, found, 2, "one finding per referencing repository")
		for _, repo := range []string{"r1", "r2"} {
			id := cobolExternalID(repo+"/main.cbl", "main.cbl::MAINPGM", "program", "literal", "'NOPGM'")
			n := found[id]
			require.NotNil(t, n, "finding %s", id)
			assert.Equal(t, repo, n.RepoPrefix, "owned by its own repository")
			assert.Equal(t, map[graph.EdgeKind]string{
				graph.EdgeMemberOf:   repo + "/main.cbl::MAINPGM",
				graph.EdgeReferences: repo + "/cobol-missing::program/NOPGM",
			}, cobolFindingTargets(t, g, id))
		}
	})

	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	seed(store)
	diag, ok := graph.ReadPrefixDiagnostics(store, 10)
	require.True(t, ok)
	assert.Zero(t, diag.UnownedCodeNodes, "unowned: %v", diag.UnownedSamples)
	assert.Zero(t, diag.MisprefixedNodes, "misprefixed: %v", diag.MisprefixedSamples)
}

// TestCobolReconcile_FindingChangesDrain pins the change record the indexer
// drains before its ledger pass: the reconciled repositories (widened by
// resolve_from) and the finding and placeholder IDs added or evicted,
// accumulated across passes and cleared by each drain.
func TestCobolReconcile_FindingChangesDrain(t *testing.T) {
	g := graph.New()
	r := New(g)
	r.SetCobolPolicyLookup(cobolPolicies(map[string]CobolPolicy{"r2": {ResolveFrom: []string{"r1"}}}))
	for _, c := range []struct{ repo, program, para, target string }{
		{"r2", "B2PGM", "B2-PARA", "NOPGM"},
		{"r3", "B3PGM", "B3-PARA", "NOPGM3"},
	} {
		file := c.repo + "/b.cbl"
		seedCobol(g, c.repo, file+"::"+c.program, graph.KindFunction, "program", c.program)
		para := file + "::" + c.program + "/" + c.para
		seedCobol(g, c.repo, para, graph.KindFunction, "paragraph", c.para)
		callNo := "unresolved::cobol::program::" + c.target
		cobolEdge(g, para, 1, graph.EdgeCalls, callNo, cobolAt(cobolCallMeta(callNo, "'"+c.target+"'"), 0, 100))
	}
	externalOf := func(repo string) string {
		t.Helper()
		var ids []string
		for id, n := range cobolExternalNodes(g) {
			if n.RepoPrefix == repo {
				ids = append(ids, id)
			}
		}
		require.Len(t, ids, 1, "one external finding in %s", repo)
		return ids[0]
	}

	r.ReconcileCobol(nil)
	repos, added, evicted := r.CobolFindingChanges()
	f2, f3 := externalOf("r2"), externalOf("r3")
	assert.Empty(t, repos, "a pass over every edge names no repository set")
	want := []string{f2, f3, "r2/cobol-missing::program/NOPGM", "r3/cobol-missing::program/NOPGM3"}
	sort.Strings(want)
	assert.Equal(t, want, added)
	assert.Empty(t, evicted)
	repos, added, evicted = r.CobolFindingChanges()
	assert.Empty(t, repos, "a drain clears the record")
	assert.Empty(t, added)
	assert.Empty(t, evicted)

	// NOPGM is obtained in r1; r2 resolves from r1, r3 does not.
	seedCobol(g, "r1", "r1/nopgm.cbl::NOPGM", graph.KindFunction, "program", "NOPGM")
	changed := map[string]struct{}{"r1": {}}
	r.ReconcileCobol(changed)
	r.ReconcileCobol(changed) // a no-op pass must not hide the first one
	repos, added, evicted = r.CobolFindingChanges()
	assert.Equal(t, []string{"r1", "r2"}, repos, "the reconcile set widened by resolve_from")
	want = []string{f2, "r2/cobol-missing::program/NOPGM"}
	sort.Strings(want)
	assert.Equal(t, want, evicted)
	assert.Empty(t, added)
	assert.Equal(t, []string{f3}, mapKeys(cobolExternalNodes(g)), "r3's finding is untouched")
	assert.NotNil(t, g.GetNode("r3/cobol-missing::program/NOPGM3"), "r3's placeholder is untouched")
}
