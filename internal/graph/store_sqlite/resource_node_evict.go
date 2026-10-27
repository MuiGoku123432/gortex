package store_sqlite

import "github.com/zzet/gortex/internal/graph"

// EvictResourceNodesByIDs removes only KindResource nodes from the supplied
// bounded ID set and every incident edge in one transaction. The kind guard in
// every statement keeps every other node safe even when a wrong ID list names
// it; the COBOL resolver sweeps missing-artifact placeholders, which own no
// file path, through it instead of file eviction (D-27, review WR-05).
//
// Scoped to the handle's payload view generation: the resolver computes the
// stale placeholder set from the corpus this handle indexed, so a resource
// still produced in another generation must survive.
func (s *Store) EvictResourceNodesByIDs(ids []string) (nodesRemoved, edgesRemoved int) {
	return s.evictNodesOfKindByIDs(graph.KindResource, ids)
}

var _ graph.ResourceNodeBatchEvicter = (*Store)(nil)
