package resolver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
)

// placeholderMeta is the Meta the extractor stamps on every literal CALL and
// non-IDMS COPY edge: its original unresolved target (04-02).
func placeholderMeta(placeholder string) map[string]any {
	return map[string]any{"cobol_placeholder": placeholder}
}

// cobolWriteCounter counts the writes the reconcile pass issues (edge moves,
// Meta writes, batch adds, file evictions), so a test can assert that an
// unchanged decision writes nothing.
type cobolWriteCounter struct {
	*store_sqlite.Store
	reindexes, persists, adds, evicts int
}

func (s *cobolWriteCounter) reset() { s.reindexes, s.persists, s.adds, s.evicts = 0, 0, 0, 0 }

func (s *cobolWriteCounter) writes() int { return s.reindexes + s.persists + s.adds + s.evicts }

func (s *cobolWriteCounter) AddBatch(nodes []*graph.Node, edges []*graph.Edge) {
	s.adds++
	s.Store.AddBatch(nodes, edges)
}

func (s *cobolWriteCounter) EvictFile(filePath string) (int, int) {
	s.evicts++
	return s.Store.EvictFile(filePath)
}

func (s *cobolWriteCounter) ReindexEdges(batch []graph.EdgeReindex) {
	s.reindexes += len(batch)
	s.Store.ReindexEdges(batch)
}

func (s *cobolWriteCounter) PersistEdgeAttributesBatch(edges []*graph.Edge) {
	s.persists += len(edges)
	s.Store.PersistEdgeAttributesBatch(edges)
}

func TestCobolReconcile_RecheckBound(t *testing.T) {
	g := graph.New()
	const caller = "a/main.cbl::MAINPGM"
	const placeholder = "unresolved::cobol::program::SUBPGM"
	seedCobol(g, "a", caller, graph.KindFunction, "program", "MAINPGM")
	seedCobol(g, "a", "a/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
	cobolEdge(g, caller, 4, graph.EdgeCalls, "a/sub.cbl::SUBPGM", placeholderMeta(placeholder))

	// A second top-level SUBPGM appears in the same repository.
	seedCobol(g, "a", "a/other.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
	New(g).ReconcileCobol(map[string]struct{}{"a": {}})

	e := cobolEdgeAt(t, g, caller, 4)
	assert.Equal(t, placeholder, e.To, "a bound edge returns to its placeholder when a second candidate appears")
	assert.Equal(t, "ambiguous", e.Meta["cobol_unresolved_reason"])
	assert.Equal(t, []string{"a/other.cbl::SUBPGM", "a/sub.cbl::SUBPGM"}, e.Meta["cobol_candidate_ids"])
	assert.Equal(t, placeholder, e.Meta["cobol_placeholder"], "the placeholder Meta is kept")
	assert.Empty(t, g.GetInEdges("a/sub.cbl::SUBPGM"), "the old target no longer has the in-edge")
}

func TestCobolReconcile_TargetGone(t *testing.T) {
	g := graph.New()
	const caller = "a/main.cbl::MAINPGM"
	const placeholder = "unresolved::cobol::copybook::CUSTREC"
	seedCobol(g, "a", caller, graph.KindFunction, "program", "MAINPGM")
	// The CR-01 restub leaves a deleted copybook's in-edge on its placeholder.
	cobolEdge(g, caller, 2, graph.EdgeImports, placeholder, placeholderMeta(placeholder))

	New(g).ReconcileCobol(map[string]struct{}{"a": {}})

	e := cobolEdgeAt(t, g, caller, 2)
	assert.Equal(t, "a/cobol-missing::copybook/CUSTREC", e.To, "a missing copybook targets its placeholder node")
	assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
	assert.NotContains(t, e.Meta, "cobol_candidate_ids")

	// The copybook comes back with the same ID.
	seedCobol(g, "a", "a/copy/custrec.cpy", graph.KindFile, "copybook", "CUSTREC")
	New(g).ReconcileCobol(map[string]struct{}{"a": {}})

	e = cobolEdgeAt(t, g, caller, 2)
	assert.Equal(t, "a/copy/custrec.cpy", e.To, "binds to the same ID again")
	assert.Equal(t, graph.OriginASTResolved, e.Origin)
	assert.NotContains(t, e.Meta, "cobol_unresolved_reason")
	assert.NotContains(t, e.Meta, "cobol_candidate_ids")
	assert.Equal(t, placeholder, e.Meta["cobol_placeholder"])
}

func TestCobolReconcile_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cobol.sqlite")
	opened, err := store_sqlite.Open(path)
	require.NoError(t, err)
	store := &cobolWriteCounter{Store: opened}
	t.Cleanup(func() { require.NoError(t, opened.Close()) })

	const caller = "a/main.cbl::MAINPGM"
	seedCobol(store, "a", caller, graph.KindFunction, "program", "MAINPGM")
	seedCobol(store, "a", "a/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
	seedCobol(store, "a", "a/d1.cbl::DUP", graph.KindFunction, "program", "DUP")
	seedCobol(store, "a", "a/d2.cbl::DUP", graph.KindFunction, "program", "DUP")
	seedCobol(store, "a", "a/copy/custrec.cpy", graph.KindFile, "copybook", "CUSTREC")
	for line, to := range map[int]string{
		1: "unresolved::cobol::program::SUBPGM",
		2: "unresolved::cobol::program::NOPE",
		3: "unresolved::cobol::program::DUP",
	} {
		cobolEdge(store, caller, line, graph.EdgeCalls, to, placeholderMeta(to))
	}
	// Already bound, and still the single candidate.
	cobolEdge(store, caller, 4, graph.EdgeImports, "a/copy/custrec.cpy",
		placeholderMeta("unresolved::cobol::copybook::CUSTREC"))

	type snap struct {
		To   string
		Meta map[string]any
	}
	capture := func() map[int]snap {
		out := map[int]snap{}
		for _, e := range store.GetOutEdges(caller) {
			out[e.Line] = snap{To: e.To, Meta: e.Meta}
		}
		return out
	}

	changed := map[string]struct{}{"a": {}}
	New(store).ReconcileCobol(changed)
	first := capture()
	store.reindexes, store.persists = 0, 0
	New(store).ReconcileCobol(changed)
	second := capture()

	assert.Equal(t, first, second, "a second reconcile leaves every edge identical")
	assert.Zero(t, store.reindexes, "no second reindex")
	assert.Zero(t, store.persists, "no second Meta write")
	assert.Equal(t, "a/sub.cbl::SUBPGM", first[1].To)
	assert.Equal(t, "missing", first[2].Meta["cobol_unresolved_reason"])
	assert.Equal(t, "ambiguous", first[3].Meta["cobol_unresolved_reason"])
	assert.Equal(t, "a/copy/custrec.cpy", first[4].To)
}

func TestCobolReconcile_ScopedRepos(t *testing.T) {
	g := graph.New()
	const callerA = "a/main.cbl::MAINPGM"
	const callerB = "b/main.cbl::MAINPGM"
	const placeholder = "unresolved::cobol::program::SUBPGM"
	seedCobol(g, "a", callerA, graph.KindFunction, "program", "MAINPGM")
	seedCobol(g, "a", "a/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
	seedCobol(g, "b", callerB, graph.KindFunction, "program", "MAINPGM")
	cobolEdge(g, callerA, 4, graph.EdgeCalls, placeholder, placeholderMeta(placeholder))
	cobolEdge(g, callerB, 4, graph.EdgeCalls, placeholder, placeholderMeta(placeholder))

	New(g).ReconcileCobol(map[string]struct{}{"a": {}})

	assert.Equal(t, "a/sub.cbl::SUBPGM", cobolEdgeAt(t, g, callerA, 4).To, "repoA's edge is re-decided")
	b := cobolEdgeAt(t, g, callerB, 4)
	assert.Equal(t, placeholder, b.To, "repoB's edge is outside the reconcile set")
	assert.Equal(t, placeholderMeta(placeholder), b.Meta, "no reason stamped outside the reconcile set")
}

func TestCobolReconcile_CallerUnowned(t *testing.T) {
	g := graph.New()
	const caller = "a/orphan.cbl::ORPHAN"
	const placeholder = "unresolved::cobol::program::SUBPGM"
	seedCobol(g, "", caller, graph.KindFunction, "program", "ORPHAN")
	seedCobol(g, "a", "a/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
	cobolEdge(g, caller, 4, graph.EdgeCalls, placeholder, placeholderMeta(placeholder))
	cobolEdge(g, caller, 5, graph.EdgeCalls, "a/sub.cbl::SUBPGM", placeholderMeta(placeholder))

	New(g).ReconcileCobol(nil)

	for _, line := range []int{4, 5} {
		e := cobolEdgeAt(t, g, caller, line)
		assert.Equal(t, placeholder, e.To, "line %d: an unowned caller never binds", line)
		assert.Equal(t, "caller_unowned", e.Meta["cobol_unresolved_reason"], "line %d", line)
		assert.NotContains(t, e.Meta, "cobol_candidate_ids", "line %d", line)
	}
}

// cobolPolicies returns a policy lookup over a fixed per-repository map.
func cobolPolicies(m map[string]CobolPolicy) CobolPolicyLookup {
	return func(repo string) CobolPolicy { return m[repo] }
}

func TestCobolReconcile_Allowlist(t *testing.T) {
	const caller = "repoA/main.cbl::MAINPGM"
	const callX = "unresolved::cobol::program::XPGM"
	const copyX = "unresolved::cobol::copybook::XCPY"
	allowB := cobolPolicies(map[string]CobolPolicy{"repoA": {ResolveFrom: []string{"repoB"}}})

	seed := func() *graph.Graph {
		g := graph.New()
		seedCobol(g, "repoA", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "repoB", "repoB/x.cbl::XPGM", graph.KindFunction, "program", "XPGM")
		seedCobol(g, "repoB", "repoB/copy/xcpy.cpy", graph.KindFile, "copybook", "XCPY")
		cobolEdge(g, caller, 1, graph.EdgeCalls, callX, placeholderMeta(callX))
		cobolEdge(g, caller, 2, graph.EdgeImports, copyX, placeholderMeta(copyX))
		return g
	}

	t.Run("no_policy", func(t *testing.T) {
		g := seed()
		New(g).ReconcileCobol(nil)
		for _, line := range []int{1, 2} {
			e := cobolEdgeAt(t, g, caller, line)
			assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"], "line %d binds only inside repoA", line)
			assert.False(t, e.CrossRepo)
		}
	})

	t.Run("allowed", func(t *testing.T) {
		g := seed()
		r := New(g)
		r.SetCobolPolicyLookup(allowB)
		r.ReconcileCobol(nil)
		call := cobolEdgeAt(t, g, caller, 1)
		assert.Equal(t, "repoB/x.cbl::XPGM", call.To)
		assert.True(t, call.CrossRepo)
		assert.NotContains(t, call.Meta, "cobol_unresolved_reason")
		cp := cobolEdgeAt(t, g, caller, 2)
		assert.Equal(t, "repoB/copy/xcpy.cpy", cp.To)
		assert.True(t, cp.CrossRepo)
	})

	t.Run("union_ambiguous", func(t *testing.T) {
		g := seed()
		const callSub = "unresolved::cobol::program::SUBPGM"
		seedCobol(g, "repoA", "repoA/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		seedCobol(g, "repoB", "repoB/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		cobolEdge(g, caller, 3, graph.EdgeCalls, callSub, placeholderMeta(callSub))
		r := New(g)
		r.SetCobolPolicyLookup(allowB)
		r.ReconcileCobol(nil)
		e := cobolEdgeAt(t, g, caller, 3)
		assert.Equal(t, callSub, e.To, "the same-repository candidate does not shadow the allowed one")
		assert.Equal(t, "ambiguous", e.Meta["cobol_unresolved_reason"])
		assert.Equal(t, []string{"repoA/sub.cbl::SUBPGM", "repoB/sub.cbl::SUBPGM"}, e.Meta["cobol_candidate_ids"])
	})

	t.Run("directed", func(t *testing.T) {
		g := seed()
		r := New(g)
		r.SetCobolPolicyLookup(cobolPolicies(map[string]CobolPolicy{"repoB": {ResolveFrom: []string{"repoA"}}}))
		r.ReconcileCobol(nil)
		for _, line := range []int{1, 2} {
			e := cobolEdgeAt(t, g, caller, line)
			assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"], "line %d: repoB's list does not widen repoA", line)
		}
	})

	t.Run("foreign_nested_excluded", func(t *testing.T) {
		g := seed()
		const callNested = "unresolved::cobol::program::INNER"
		seedCobol(g, "repoB", "repoB/x.cbl::XPGM/INNER", graph.KindFunction, "program", "INNER")
		inner := g.GetNode("repoB/x.cbl::XPGM/INNER")
		inner.Meta["cobol_nested"] = true
		inner.Meta["cobol_container"] = "repoB/x.cbl::XPGM"
		meta := placeholderMeta(callNested)
		meta["cobol_caller_program"] = "repoB/x.cbl::XPGM"
		cobolEdge(g, caller, 3, graph.EdgeCalls, callNested, meta)
		r := New(g)
		r.SetCobolPolicyLookup(allowB)
		r.ReconcileCobol(nil)
		e := cobolEdgeAt(t, g, caller, 3)
		assert.Equal(t, "repoA/cobol-missing::program/INNER", e.To, "a nested program in another repository is never a candidate")
		assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
	})

	t.Run("reverse_fanout", func(t *testing.T) {
		g := graph.New()
		seedCobol(g, "repoA", caller, graph.KindFunction, "program", "MAINPGM")
		seedCobol(g, "repoB", "repoB/main.cbl::BMAIN", graph.KindFunction, "program", "BMAIN")
		cobolEdge(g, caller, 1, graph.EdgeCalls, callX, placeholderMeta(callX))
		r := New(g)
		r.SetCobolPolicyLookup(allowB)
		r.ReconcileCobol(nil)
		require.Equal(t, "missing", cobolEdgeAt(t, g, caller, 1).Meta["cobol_unresolved_reason"])

		seedCobol(g, "repoB", "repoB/x.cbl::XPGM", graph.KindFunction, "program", "XPGM")
		r.ReconcileCobol(map[string]struct{}{"repoB": {}})
		e := cobolEdgeAt(t, g, caller, 1)
		assert.Equal(t, "repoB/x.cbl::XPGM", e.To, "a change in repoB re-decides repoA, which resolves from repoB")
		assert.True(t, e.CrossRepo)
	})

	t.Run("unknown_repo", func(t *testing.T) {
		g := seed()
		const callSub = "unresolved::cobol::program::SUBPGM"
		seedCobol(g, "repoA", "repoA/sub.cbl::SUBPGM", graph.KindFunction, "program", "SUBPGM")
		cobolEdge(g, caller, 3, graph.EdgeCalls, callSub, placeholderMeta(callSub))
		r := New(g)
		r.SetCobolPolicyLookup(cobolPolicies(map[string]CobolPolicy{"repoA": {ResolveFrom: []string{"ghost", " "}}}))
		r.ReconcileCobol(nil)
		assert.Equal(t, "missing", cobolEdgeAt(t, g, caller, 1).Meta["cobol_unresolved_reason"])
		assert.Equal(t, "missing", cobolEdgeAt(t, g, caller, 2).Meta["cobol_unresolved_reason"])
		assert.Equal(t, "repoA/sub.cbl::SUBPGM", cobolEdgeAt(t, g, caller, 3).To, "same-repository binding is unchanged")
	})
}

func TestCobolReconcile_SearchOrder(t *testing.T) {
	const caller = "repoA/main.cbl::MAINPGM"
	const copyCust = "unresolved::cobol::copybook::CUSTREC"
	const prod = "repoA/copy/prod/custrec.cpy"
	const test = "repoA/copy/test/custrec.cpy"
	libs := func(l ...CobolLibrary) CobolPolicyLookup {
		return cobolPolicies(map[string]CobolPolicy{"repoA": {CopybookLibraries: l}})
	}
	seed := func(g graph.Store, members ...string) {
		seedCobol(g, "repoA", caller, graph.KindFunction, "program", "MAINPGM")
		for _, m := range members {
			seedCobol(g, graph.RepoPrefixOfID(m), m, graph.KindFile, "copybook", "CUSTREC")
		}
	}
	assertRanked := func(t *testing.T, e *graph.Edge, to string, rank int) {
		t.Helper()
		assert.Equal(t, to, e.To)
		assert.NotContains(t, e.Meta, "cobol_unresolved_reason")
		assert.EqualValues(t, rank, e.Meta["cobol_copy_search_rank"])
		assert.Equal(t, "cobol-copy-search/1", e.Meta["cobol_copy_search_rule"])
	}

	t.Run("rank_wins", func(t *testing.T) {
		opened, err := store_sqlite.Open(filepath.Join(t.TempDir(), "cobol.sqlite"))
		require.NoError(t, err)
		store := &cobolWriteCounter{Store: opened}
		t.Cleanup(func() { require.NoError(t, opened.Close()) })
		seed(store, prod, test)
		cobolEdge(store, caller, 2, graph.EdgeImports, copyCust, placeholderMeta(copyCust))
		r := New(store)
		r.SetCobolPolicyLookup(libs(CobolLibrary{Path: "copy/prod"}, CobolLibrary{Path: "copy/test/"}))
		r.ReconcileCobol(nil)
		assertRanked(t, cobolEdgeAt(t, store, caller, 2), prod, 0)

		store.reindexes, store.persists = 0, 0
		r.ReconcileCobol(nil)
		assertRanked(t, cobolEdgeAt(t, store, caller, 2), prod, 0)
		assert.Zero(t, store.reindexes, "no second reindex")
		assert.Zero(t, store.persists, "no second Meta write")
	})

	t.Run("of_qualifier", func(t *testing.T) {
		g := graph.New()
		seed(g, prod, test)
		meta := placeholderMeta(copyCust)
		meta["cobol_copy_library_key"] = "TESTLIB"
		cobolEdge(g, caller, 2, graph.EdgeImports, copyCust, meta)
		r := New(g)
		r.SetCobolPolicyLookup(libs(CobolLibrary{Name: "PRODLIB", Path: "copy/prod"}, CobolLibrary{Name: "TESTLIB", Path: "copy/test"}))
		r.ReconcileCobol(nil)
		assertRanked(t, cobolEdgeAt(t, g, caller, 2), test, 1)
	})

	t.Run("unranked_dropped", func(t *testing.T) {
		g := graph.New()
		seed(g, "repoA/other/custrec.cpy")
		cobolEdge(g, caller, 2, graph.EdgeImports, copyCust, placeholderMeta(copyCust))
		r := New(g)
		r.SetCobolPolicyLookup(libs(CobolLibrary{Path: "copy/prod"}))
		r.ReconcileCobol(nil)
		e := cobolEdgeAt(t, g, caller, 2)
		assert.Equal(t, "repoA/cobol-missing::copybook/CUSTREC", e.To, "a copybook outside every library is not a candidate")
		assert.Equal(t, "missing", e.Meta["cobol_unresolved_reason"])
		assert.NotContains(t, e.Meta, "cobol_copy_search_rank")
	})

	t.Run("tie_ambiguous", func(t *testing.T) {
		g := graph.New()
		seed(g, prod, "repoA/copy/prod/CUSTREC.cpy")
		cobolEdge(g, caller, 2, graph.EdgeImports, copyCust, placeholderMeta(copyCust))
		r := New(g)
		r.SetCobolPolicyLookup(libs(CobolLibrary{Path: "copy/prod"}))
		r.ReconcileCobol(nil)
		e := cobolEdgeAt(t, g, caller, 2)
		assert.Equal(t, copyCust, e.To)
		assert.Equal(t, "ambiguous", e.Meta["cobol_unresolved_reason"])
		assert.Equal(t, []string{"repoA/copy/prod/CUSTREC.cpy", prod}, e.Meta["cobol_candidate_ids"])
		assert.NotContains(t, e.Meta, "cobol_copy_search_rank")
		assert.NotContains(t, e.Meta, "cobol_copy_search_rule")
	})

	t.Run("cross_repo_library", func(t *testing.T) {
		g := graph.New()
		seed(g, "repoB/copy/custrec.cpy")
		cobolEdge(g, caller, 2, graph.EdgeImports, copyCust, placeholderMeta(copyCust))
		r := New(g)
		r.SetCobolPolicyLookup(cobolPolicies(map[string]CobolPolicy{"repoA": {
			ResolveFrom:       []string{"repoB"},
			CopybookLibraries: []CobolLibrary{{Path: "copy"}, {Repo: "repoB", Path: "copy"}},
		}}))
		r.ReconcileCobol(nil)
		e := cobolEdgeAt(t, g, caller, 2)
		assertRanked(t, e, "repoB/copy/custrec.cpy", 1)
		assert.True(t, e.CrossRepo)
	})

	t.Run("no_libraries", func(t *testing.T) {
		g := graph.New()
		seed(g, prod, test)
		meta := placeholderMeta(copyCust)
		meta["cobol_copy_search_rank"] = 0
		meta["cobol_copy_search_rule"] = "cobol-copy-search/1"
		cobolEdge(g, caller, 2, graph.EdgeImports, prod, meta)
		New(g).ReconcileCobol(nil)
		e := cobolEdgeAt(t, g, caller, 2)
		assert.Equal(t, copyCust, e.To)
		assert.Equal(t, "ambiguous", e.Meta["cobol_unresolved_reason"])
		assert.Equal(t, []string{prod, test}, e.Meta["cobol_candidate_ids"])
		assert.NotContains(t, e.Meta, "cobol_copy_search_rank", "a decision that is not a library bind drops the rank")
		assert.NotContains(t, e.Meta, "cobol_copy_search_rule")
	})
}
