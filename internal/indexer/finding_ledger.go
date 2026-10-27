package indexer

import (
	"maps"
	"slices"
	"sync"

	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/resolver"
)

// findingLedgerMu serializes every finding ledger pass in the process, so
// two passes over one repository never both read the same latest rows and
// append the same transitions twice.
var findingLedgerMu sync.Mutex

// recordFindingLedger appends, for each repository in repos, the finding
// lifecycle rows that move its recorded history to its current finding
// nodes in store.
//
// PURPOSE — finding_history is the append-only log of observed transitions
// (D-06). For each repository, in sorted unique order, the pass reads only
// the latest row per finding (LatestFindingTransitions, one indexed query,
// never the full log), collects the repository's KindFinding nodes as the
// active set, and appends graph.FindingLedgerTransitions of the two: opens,
// closes, supersedes, and reopens; identical input appends nothing (GAP-06).
//
// RATIONALE — it runs only on the durable store, after a COBOL settle point,
// and never inside the resolver: the cold path resolves on an in-memory
// shadow that keeps no history (Pitfall 3). It is not gated on COBOL
// presence, so deleting a repository's last COBOL file still closes every
// finding of it (Pitfall 8). History is best effort (D-07): a store without
// both capabilities keeps none, and a read or append error is logged with
// the repository only, never Meta or source, and the next repository runs.
// findingLedgerMu is held across the whole read-diff-append.
//
// KEYWORDS — finding, ledger, history, lifecycle, transition, durable store,
// shadow, D-06, D-07, GAP-06, GAP-07, Pitfall 3, Pitfall 8.
func recordFindingLedger(store graph.Store, repos []string, logger *zap.Logger) {
	reader, ok := store.(graph.FindingHistoryReader)
	if !ok {
		return
	}
	appender, ok := store.(graph.FindingHistoryAppender)
	if !ok {
		return
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	findingLedgerMu.Lock()
	defer findingLedgerMu.Unlock()
	for _, repo := range slices.Compact(slices.Sorted(slices.Values(repos))) {
		latest, err := reader.LatestFindingTransitions(repo)
		if err != nil {
			logger.Warn("indexer: reading finding history failed", zap.String("repo", repo), zap.Error(err))
			continue
		}
		var active []graph.FindingTransition
		for n := range graph.NodesInScopeSeq(store, []string{repo}, nil, graph.KindFinding) {
			if n != nil {
				active = append(active, findingTransitionOf(n))
			}
		}
		rows := graph.FindingLedgerTransitions(latest, active)
		if len(rows) == 0 {
			continue
		}
		if err := appender.AppendFindingTransitions(repo, rows); err != nil {
			logger.Warn("indexer: appending finding history failed", zap.String("repo", repo), zap.Error(err))
		}
	}
}

// findingTransitionOf maps a finding node onto the history row the ledger
// diff compares: its ID and scope, the pairing key (class, kind, container,
// start byte, -1 when the node has no range), the evidence digest, and the
// parser stamps.
func findingTransitionOf(n *graph.Node) graph.FindingTransition {
	str := func(key string) string {
		s, _ := n.Meta[key].(string)
		return s
	}
	// note: metaToInt (affected_by.go) and metaInt (content_split.go) read
	// the same forms but default to 0, which is a real byte offset here.
	start := -1
	switch v := n.Meta["prov_start_byte"].(type) {
	case int:
		start = v
	case int64:
		start = int(v)
	case float64:
		start = int(v)
	}
	return graph.FindingTransition{
		FindingID:        n.ID,
		RepoPrefix:       n.RepoPrefix,
		WorkspaceID:      n.WorkspaceID,
		ProjectID:        n.ProjectID,
		Class:            str(graph.MetaEvidenceClass),
		Kind:             str(graph.MetaFindingKind),
		Container:        str(graph.MetaFindingContainer),
		StartByte:        start,
		EvidenceDigest:   str(graph.MetaFindingEvidenceDigest),
		ParserToolID:     str("prov_parser_tool_id"),
		ParserGrammarID:  str("prov_parser_grammar_id"),
		ExtractorVersion: str("prov_extractor_version"),
	}
}

// recordCobolFindingLedger is the finding ledger of a COBOL settle point. It
// drains every given resolver's change record, ledgers the union of base,
// the drained reconciled repositories, and the tracked repositories the
// drained added and evicted IDs name, and returns the drained added and
// evicted IDs, each sorted.
//
// PURPOSE — one reconcile can open or close findings outside the repository
// that changed: cobolReconcileRepos widens the set to every repository whose
// resolve_from names a changed one, and a per-repository cold ResolveAll
// reconciles every COBOL edge of its store (no scope). Ledgering only the
// changed repository would leave a dependent repository's transition
// unrecorded, and its next pass could pair the gone finding with an
// unrelated new one in the same container (a false supersede). So the
// dependent repository's row lands at the step that caused it (D-03 across
// repositories, roadmap SC4).
//
// RATIONALE — the drain runs before the ledger, so the reconciled set
// covers an earlier pass on another resolver (a discarded master, a cold or
// fallback ResolveAll) that this tail's own no-op pass merely repeats, and
// the ID prefixes cover the scope-nil pass. An ID's repository prefix
// (cobolOwnerPrefix) is trusted only when tracked reports it (an unprefixed
// ID yields a directory name), so a nil tracked ignores prefixes.
// The ledger runs on store, the durable graph, only (Pitfall 3, D-06, D-07);
// the returned IDs feed search sync.
//
// The ledger observes resolver reconciles and file commits, so it holds
// their exclusions (D-23, WR-01): reconcileMu, when non-nil, is the reconcile
// lock the given resolvers share (SetCobolReconcileLock) and is held from
// before the drain until recordFindingLedger returns, so no reconcile's
// evict, add, and sweep writes land between the drain and the read; the
// caller holds the reach topology writer, which every repository lane holds
// from its file commit through its reconcile. Lock order everywhere: reach
// topology writer, then reconcileMu, then findingLedgerMu.
//
// KEYWORDS — cobol, finding, ledger, drain, change record, dependent
// repository, resolve_from, settle point, exclusion, lock order, D-03, D-06,
// D-07, D-23, WR-01, Pitfall 3.
func recordCobolFindingLedger(
	store graph.Store,
	base []string,
	tracked func(prefix string) bool,
	reconcileMu *sync.Mutex,
	logger *zap.Logger,
	resolvers ...*resolver.Resolver,
) (added, evicted []string) {
	if reconcileMu != nil {
		reconcileMu.Lock()
		defer reconcileMu.Unlock()
	}
	ledger := make(map[string]struct{}, len(base))
	for _, repo := range base {
		ledger[repo] = struct{}{}
	}
	addedSet, evictedSet := map[string]struct{}{}, map[string]struct{}{}
	for _, r := range resolvers {
		if r == nil {
			continue
		}
		repos, a, e := r.CobolFindingChanges()
		for _, repo := range repos {
			ledger[repo] = struct{}{}
		}
		for _, id := range a {
			addedSet[id] = struct{}{}
		}
		for _, id := range e {
			evictedSet[id] = struct{}{}
		}
	}
	if tracked != nil {
		prefixes := map[string]struct{}{}
		for _, ids := range []map[string]struct{}{addedSet, evictedSet} {
			for id := range ids {
				if p := cobolOwnerPrefix(id); p != "" {
					prefixes[p] = struct{}{}
				}
			}
		}
		for p := range prefixes {
			if tracked(p) {
				ledger[p] = struct{}{}
			}
		}
	}
	recordFindingLedger(store, slices.Sorted(maps.Keys(ledger)), logger)
	return slices.Sorted(maps.Keys(addedSet)), slices.Sorted(maps.Keys(evictedSet))
}

// syncCobolFindingFTS brings this indexer's symbol search index in step with
// the finding and placeholder IDs COBOL reconciles added and evicted.
//
// PURPOSE — symbol FTS is written by the indexer, never by AddBatch or the
// evicters, so a node reconcileCobolFindings creates stays unsearchable and
// one it removes stays searchable until something rewrites it (Pitfall 4).
// Evicted IDs are deleted from FTS; added IDs whose node is in the store and
// passes shouldIndexForSearch are upserted with ftsTokensFor tokens: the
// writes the incremental stage path makes (deleteSymbolFTS,
// upsertIncrementalFTS), invoking no new index.
//
// RATIONALE — the delete runs first and an added ID is upserted only while
// its node exists, so an ID in both lists (created and removed, or the
// reverse, between two drains) ends in the right state, including a
// shadow-built node a master pass evicted. The upsert is idempotent, so
// re-writing a node the cold path already indexed is harmless. The
// in-process text backend (idx.search) is not written: on a store with a
// symbol searcher its Add and Remove are no-ops, and the evicted node that
// removeFromSearch's admission gate reads is already gone. Write errors are
// logged like the incremental path's, with no Meta or source.
//
// KEYWORDS — cobol, finding, placeholder, search, symbol FTS, sync, settle
// point, Pitfall 4.
func (idx *Indexer) syncCobolFindingFTS(added, evicted []string) {
	if len(evicted) > 0 {
		idx.deleteSymbolFTS(evicted)
	}
	batcher, ok := idx.graph.(graph.SymbolFTSBatchUpserter)
	if !ok || len(added) == 0 {
		return
	}
	nodes := idx.graph.GetNodesByIDs(added)
	var items []graph.SymbolFTSItem
	for _, id := range added {
		if n := nodes[id]; n != nil && idx.shouldIndexForSearch(n) {
			items = append(items, graph.SymbolFTSItem{NodeID: n.ID, Tokens: ftsTokensFor(n, idx.projectName)})
		}
	}
	if len(items) == 0 {
		return
	}
	if err := batcher.BatchUpsertSymbolFTS(items); err != nil {
		idx.logger.Debug("indexer: backend FTS batch upsert failed",
			zap.Int("symbols", len(items)), zap.Error(err))
	}
}

// syncCobolFindingFTSByRepo syncs search from drained added and evicted
// finding and placeholder IDs, each through the Indexer of the repository
// its ID names.
//
// PURPOSE — one reconcile writes nodes other repositories own: a change in
// one repository closes or reopens findings and placeholders of every
// repository whose resolve_from names it, and a master ResolveAll, including
// the fallback masters, reconciles its whole scope. Routing each ID to its
// own repository makes those nodes searchable or gone at the step that
// drained them (Pitfall 4, D-03 across repositories).
//
// RATIONALE — the IDs are grouped by cobolOwnerPrefix, so a placeholder ID
// (`<repo>::...`) routes to its owner like a finding ID (`<repo>/...`), and
// written only through a tracked prefix's Indexer, whose projectName yields
// the tokens; an empty or untracked prefix is skipped, because an
// unprefixed ID yields a directory name.
//
// KEYWORDS — cobol, finding, placeholder, search, symbol FTS, dependent
// repository, resolve_from, master resolve, D-03, Pitfall 4.
func (mi *MultiIndexer) syncCobolFindingFTSByRepo(added, evicted []string) {
	addedBy, evictedBy := map[string][]string{}, map[string][]string{}
	prefixes := map[string]struct{}{}
	for _, id := range added {
		p := cobolOwnerPrefix(id)
		addedBy[p] = append(addedBy[p], id)
		prefixes[p] = struct{}{}
	}
	for _, id := range evicted {
		p := cobolOwnerPrefix(id)
		evictedBy[p] = append(evictedBy[p], id)
		prefixes[p] = struct{}{}
	}
	for _, p := range slices.Sorted(maps.Keys(prefixes)) {
		if p == "" {
			continue
		}
		if idx := mi.GetIndexer(p); idx != nil {
			idx.syncCobolFindingFTS(addedBy[p], evictedBy[p])
		}
	}
}

// cobolOwnerPrefix returns the repository prefix a finding or placeholder
// ID of a COBOL reconcile names: the owner resolver.ParseCobolPlaceholderID
// parses from a missing-artifact placeholder ID
// (`<repo>::cobol-missing::<type>/<key>`, D-27), else graph.RepoPrefixOfID
// (a file-owned finding, `<repo>/<path>::...`). A repository prefix may
// contain "::", so neither form is cut at its first "::" (review CR-01).
func cobolOwnerPrefix(id string) string {
	if repo, _, _, ok := resolver.ParseCobolPlaceholderID(id); ok {
		return repo
	}
	return graph.RepoPrefixOfID(id)
}
