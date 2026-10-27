package graph

// FindingNodeBatchEvicter removes a bounded set of finding nodes and every
// incident edge in one backend operation. It is intentionally finding-specific:
// an external finding is owned by its referencing source file, which did not
// change when the finding stops applying, so file eviction cannot retire it.
type FindingNodeBatchEvicter interface {
	EvictFindingNodesByIDs(ids []string) (nodesRemoved, edgesRemoved int)
}

// EvictFindingNodesByIDs dispatches only to the set-oriented capability.
//
// Findings are removed by ID only when they are no longer produced (D-05,
// D-06); closed and superseded findings never stay as nodes. The kind guard
// keeps every non-finding node safe even when a wrong ID list names it.
// Production Graph and SQLite stores implement the capability; adapter stores
// report unsupported instead of hiding an N+1 point-delete fallback.
func EvictFindingNodesByIDs(store Store, ids []string) (nodesRemoved, edgesRemoved int, supported bool) {
	if store == nil || len(ids) == 0 {
		return 0, 0, true
	}
	evicter, ok := store.(FindingNodeBatchEvicter)
	if !ok {
		return 0, 0, false
	}
	nodesRemoved, edgesRemoved = evicter.EvictFindingNodesByIDs(ids)
	return nodesRemoved, edgesRemoved, true
}

// EvictFindingNodesByIDs implements the bounded in-memory capability through
// the kind-guarded synthetic-node evicter. Unknown and non-finding IDs are
// ignored.
func (g *Graph) EvictFindingNodesByIDs(ids []string) (nodesRemoved, edgesRemoved int) {
	return g.evictSyntheticNodesByIDs(ids, map[NodeKind]struct{}{KindFinding: {}})
}

var _ FindingNodeBatchEvicter = (*Graph)(nil)
