package graph

// ResourceNodeBatchEvicter removes a bounded set of resource nodes and every
// incident edge in one backend operation. It is intentionally resource-specific:
// a missing-artifact placeholder owns no file path, so file eviction cannot
// retire it without risking a repository path that collides with it.
type ResourceNodeBatchEvicter interface {
	EvictResourceNodesByIDs(ids []string) (nodesRemoved, edgesRemoved int)
}

// EvictResourceNodesByIDs dispatches only to the set-oriented capability.
//
// It removes a bounded set of KindResource nodes by ID with every incident
// edge. The kind guard keeps every other node safe even when a wrong ID list
// names it. The COBOL resolver uses it to sweep missing-artifact placeholders,
// which own no file path (D-27, review WR-05). Production Graph and SQLite
// stores implement the capability; adapter stores report unsupported instead
// of hiding an N+1 point-delete fallback.
func EvictResourceNodesByIDs(store Store, ids []string) (nodesRemoved, edgesRemoved int, supported bool) {
	if store == nil || len(ids) == 0 {
		return 0, 0, true
	}
	evicter, ok := store.(ResourceNodeBatchEvicter)
	if !ok {
		return 0, 0, false
	}
	nodesRemoved, edgesRemoved = evicter.EvictResourceNodesByIDs(ids)
	return nodesRemoved, edgesRemoved, true
}

// EvictResourceNodesByIDs implements the bounded in-memory capability through
// the kind-guarded synthetic-node evicter. Unknown and non-resource IDs are
// ignored.
func (g *Graph) EvictResourceNodesByIDs(ids []string) (nodesRemoved, edgesRemoved int) {
	return g.evictSyntheticNodesByIDs(ids, map[NodeKind]struct{}{KindResource: {}})
}

var _ ResourceNodeBatchEvicter = (*Graph)(nil)
