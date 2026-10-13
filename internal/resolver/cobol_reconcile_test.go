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

// cobolWriteCounter counts the edge writes the reconcile pass issues, so a test
// can assert that an unchanged decision writes nothing.
type cobolWriteCounter struct {
	*store_sqlite.Store
	reindexes, persists int
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
	assert.Equal(t, placeholder, e.To)
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
