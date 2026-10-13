package indexer

import "github.com/zzet/gortex/internal/graph"

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
// stands in for an interrupted resolve.
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
}

// reconcileCobolRepos runs the COBOL reconcile over repos (every edge when
// repos is empty) through a master resolver, so the decision sees every
// tracked repository's candidates and policy.
func (mi *MultiIndexer) reconcileCobolRepos(repos map[string]struct{}) {
	master := mi.newMasterResolver(false)
	if master == nil {
		return
	}
	master.ReconcileCobol(repos)
}
