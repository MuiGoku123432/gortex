package indexer

import (
	"maps"
	"slices"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/reach"
	"github.com/zzet/gortex/internal/resolver"
)

// cobolMutationPaths returns the sorted COBOL paths an incremental mutation
// lists, from the derived invalidation plan and the receipt's changed,
// unresolved, and definition files.
func cobolMutationPaths(result *IndexResult, receipt *graph.MutationReceipt) []string {
	var lists [][]string
	if result != nil {
		lists = append(lists, result.DerivedInvalidation.Files)
	}
	if receipt != nil {
		lists = append(lists, receipt.ChangedFiles, receipt.UnresolvedFiles, receipt.DefinitionFiles)
	}
	var paths []string
	for _, list := range lists {
		for _, p := range list {
			if ExtractorLangForFile(p) == "cobol" {
				paths = append(paths, p)
			}
		}
	}
	return appendUniqueSorted(nil, paths...)
}

// cobolMutationTouched reports whether an incremental mutation may change a
// COBOL binding. It is deliberately conservative: any listed COBOL path, any
// deleted file, or a stale result with no file list at all triggers the
// reconcile, which is a no-op on a graph without COBOL nodes.
func cobolMutationTouched(result *IndexResult, receipt *graph.MutationReceipt) bool {
	if len(cobolMutationPaths(result, receipt)) > 0 {
		return true
	}
	if result == nil {
		return false
	}
	if result.DeletedFileCount > 0 {
		return true
	}
	listed := len(result.DerivedInvalidation.Files) > 0
	if receipt != nil {
		listed = listed || len(receipt.ChangedFiles) > 0 ||
			len(receipt.UnresolvedFiles) > 0 || len(receipt.DefinitionFiles) > 0
	}
	return result.StaleFileCount > 0 && !listed
}

// reconcileCobolAfterMutation is the standalone Indexer's COBOL tail. It
// runs after the resolution branch whatever the receipt says, because a
// receipt that is not ResolutionRelevant can still move or remove a COBOL
// target (WR-01). The pass is observable as catch-up kind cobol and is
// skipped while incrementalResolveFilesHook is set, the test seam that
// stands in for an interrupted resolve. It ends with the finding ledger on
// the durable store over its own repository and every repository the
// drained resolver reconciled (an unprefixed indexer trusts no ID prefix),
// then syncs symbol search from the drained IDs (Pitfall 4).
func (idx *Indexer) reconcileCobolAfterMutation(result *IndexResult, receipt *graph.MutationReceipt) {
	if idx == nil || !cobolMutationTouched(result, receipt) {
		return
	}
	idx.observeIncrementalCatchup("cobol", cobolMutationPaths(result, receipt))
	if idx.incrementalResolveFilesHook != nil || idx.resolver == nil {
		return
	}
	// note: the same repository scope is built inline in
	// MultiIndexer.resolveIncrementalRepoMutationMode; an unprefixed indexer
	// reconciles every edge.
	var repos map[string]struct{}
	if idx.repoPrefix != "" {
		repos = map[string]struct{}{idx.repoPrefix: {}}
	}
	idx.resolver.ReconcileCobol(repos)
	added, evicted := recordCobolFindingLedger(idx.graph, []string{idx.repoPrefix}, nil, nil, idx.logger, idx.resolver)
	idx.syncCobolFindingFTS(added, evicted)
}

// reconcileCobolRepos is reconcileCobolReposTopologyHeld for a caller that
// does not hold the reach topology writer: it takes the writer for the
// reconcile, drain, ledger, and search sync, so the ledger never reads a
// repository lane between its file commit and its reconcile (D-23, WR-01).
// finish(false) keeps the reach index valid, as before this pass took the
// writer. A caller inside a writer section (an incremental lane, the warm
// snapshot closure, untrack, or a repository topology batch callback) must
// call reconcileCobolReposTopologyHeld instead: the gate is not reentrant.
//
// KEYWORDS — cobol, reconcile, ledger, settle point, reach topology writer,
// TopologyHeld convention, D-23, WR-01.
func (mi *MultiIndexer) reconcileCobolRepos(repos map[string]struct{}) {
	finish := reach.BeginTopologyMutation(mi.graph)
	defer finish(false)
	mi.reconcileCobolReposTopologyHeld(repos)
}

// reconcileCobolReposTopologyHeld runs the COBOL reconcile over repos (every
// edge when repos is empty) through a master resolver, so the decision sees
// every tracked repository's candidates and policy, then the finding ledger
// on the durable store. The ledger drains the master and the resolver of
// every tracked repository in its base (repos, or every tracked repository
// when repos is empty), which catches a cold, full-retrack, or fallback
// ResolveAll whose reconcile this master only repeats. The drained IDs then
// sync symbol search, each through its own repository's Indexer. The caller
// holds the reach topology writer (D-23).
func (mi *MultiIndexer) reconcileCobolReposTopologyHeld(repos map[string]struct{}) {
	master := mi.newMasterResolver(false)
	if master == nil {
		return
	}
	master.ReconcileCobol(repos)
	base := slices.Sorted(maps.Keys(repos))
	if len(base) == 0 {
		base = mi.RepoPrefixes()
	}
	resolvers := []*resolver.Resolver{master}
	for _, p := range base {
		if idx := mi.GetIndexer(p); idx != nil {
			resolvers = append(resolvers, idx.resolver)
		}
	}
	mi.recordCobolSettleLedger(base, true, resolvers...)
}

// recordCobolSettleLedger is the finding ledger and search sync of every
// MultiIndexer COBOL settle point: it drains resolvers, ledgers base plus the
// repositories the drained records name, and syncs symbol search from the
// drained IDs.
//
// PURPOSE — the ledger is a diff of the store's current findings against
// their latest recorded rows, so it must read only settled state (D-23,
// WR-01). A repository lane evicts a file's external findings at its file
// commit and re-adds them at its reconcile; a ledger reading in between
// records a false active -> closed, then closed -> active, for unchanged
// evidence. Every lane (incremental, warm snapshot, untrack, standalone
// watcher) holds the reach topology writer from its commit through its
// reconcile, so the ledger holds the writer too; it also holds the shared
// cobolReconcileMu across drain, read, diff, and append, which excludes the
// reconciles that run outside any writer section (master ResolveAll during
// daemon warmup and deferred catch-up).
//
// RATIONALE — reach.BeginTopologyMutation is not reentrant, so writer state
// is explicit (the runGlobalGraphPasses / TopologyHeld convention):
// topologyHeld true means the caller holds the writer, or a repository
// topology batch holds it for the caller; false takes it here for the
// ledger's duration and releases it with finish(false), which leaves the
// reach index valid. Lock order everywhere: reach topology writer, then
// cobolReconcileMu, then findingLedgerMu.
//
// KEYWORDS — cobol, finding, ledger, settle point, exclusion, reach topology
// writer, cobolReconcileMu, lock order, TopologyHeld convention, D-23, D-24,
// WR-01, WR-02.
func (mi *MultiIndexer) recordCobolSettleLedger(base []string, topologyHeld bool, resolvers ...*resolver.Resolver) {
	if !topologyHeld {
		finish := reach.BeginTopologyMutation(mi.graph)
		defer finish(false)
	}
	tracked := func(prefix string) bool { return mi.GetIndexer(prefix) != nil }
	added, evicted := recordCobolFindingLedger(mi.graph, base, tracked, &mi.cobolReconcileMu, mi.logger, resolvers...)
	mi.syncCobolFindingFTSByRepo(added, evicted)
}
