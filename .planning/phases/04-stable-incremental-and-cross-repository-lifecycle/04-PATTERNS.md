# Phase 4: Stable Incremental and Cross-Repository Lifecycle - Pattern Map

**Mapped:** 2026-10-12
**Files analyzed:** 41 (9 new source/test files, 32 modified)
**Analogs found:** 40 / 41 (every file has an in-repo analog. The D-04 harness is a composite of three analogs.)

All paths are relative to `/Users/e1001547-mbp-it/repos/mine/GoApps/gortex` and were checked with `git ls-files`. None is a mirror path. Line numbers were read on 2026-10-12.

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/graph/scope.go` (NEW) | model + validator | request-response | `internal/graph/store_sqlite/scoped_projection.go:22-62` (validation body) + `internal/graph/scoped_projection.go:18-23` (type) | exact (generalize) |
| `internal/graph/scope_test.go` (NEW) | test | — | `internal/graph/prefix_diagnostics_test.go` | role-match |
| `internal/graph/scoped_projection.go` (MOD) | model | — | itself (alias `ProjectionScope = Scope`) | exact |
| `internal/graph/store_sqlite/scoped_projection.go` (MOD) | store | request-response | itself, lines 22-50 become a `graph.NewScope` call | exact |
| `internal/neo4jprojection/service.go` (MOD) | service | request-response | itself, `NormalizeRequest` :133-177 | exact |
| `internal/graph/prefix_diagnostics.go` (MOD) | utility (audit) | transform | itself, :70-74, :117-149 | exact |
| `internal/graph/store_sqlite/prefix_diagnostics.go` (MOD) | store predicate | batch | itself, :26-29 `auditableRepoSourceNodePredicate` | exact |
| `internal/graph/prefix_diagnostics_test.go`, `internal/graph/storetest/storetest.go` (MOD) | test | — | themselves (flip the fences) | exact |
| `internal/resolver/cobol_references.go` (MOD → `reconcileCobol`) | service (resolver pass) | batch / transform | itself, :68-167 | exact |
| `internal/resolver/cobol_policy.go` (NEW) | config lookup + setter | request-response | `internal/resolver/cross_repo.go:43-47,220-228` (`CrossWorkspaceDepLookup`) | exact |
| `internal/resolver/cobol_resources.go` (NEW) | service (alias edges + GC) | batch | `internal/resolver/cobol_references.go:155-166` (write path) + `graph.Store.EvictFile` | role-match |
| `internal/resolver/resolver.go` (MOD) | service | — | call site :1660-1662 | exact |
| `internal/resolver/cross_repo.go` (MOD, D-12) | service | transform | `pickQualNameCandidate` :324-352 + `lowestIDQualNameCandidate` (`qual_name_candidates.go:7-18`) | exact |
| `internal/indexer/cobol_material.go` (NEW) | utility (fingerprint) | transform | `indexConfigHash` (`checkout_coordinator.go:1820-1835`) + `contentHashForSource` (`file_meta.go:65-68`) | exact |
| `internal/indexer/extractor_version.go` (MOD) | config/utility | transform | itself, :37-50, :123-138, :172-215 | exact |
| `internal/indexer/merkle_reindex.go` (MOD) | indexer | file-I/O | call sites :92, :118 | exact |
| `internal/indexer/index_state.go` (MOD) | indexer → store | CRUD | itself, `persistRepoIndexState` :34-56, `persistExtractorVersion` :61-94, `reconcileRepoIndexState` :105-155 | exact |
| `internal/graph/index_state.go` + `store_sqlite/store_index_state.go` (MOD, only if the A3 column is chosen) | model + store | CRUD | `store_index_state.go:12-37`, `schema_version.go:426-439` | exact |
| `internal/indexer/source_revision.go` (MOD, D-17) | indexer | file-I/O | `stampSourceRevision` :44-86 | exact |
| `internal/indexer/affected_by.go`, `internal/indexer/incremental_batch.go` (MOD, D-08) | indexer | batch | themselves, :556-563 / :1494-1499 | exact |
| `internal/indexer/affected_by_e2e_test.go` (MOD) | test | — | `TestAffectedBy_CapBoundsFanout` :207-236 | exact |
| `internal/indexer/indexer.go` `restubIncomingRefs` + `incremental_batch.go` `restubIncomingRefsFromView` (MOD, CR-01) | indexer | transform | themselves, :5218-5271 / :957-987 | exact |
| `internal/indexer/incremental_watcher_batch.go` (MOD, reconcile trigger) | indexer tail | event-driven | itself, :240-278, :291-328, :376-440 | exact |
| `internal/indexer/multi.go` (MOD) | indexer (config push) | event-driven | `RefreshRepoConfigs` :2457-2486 + `SetExcludePatterns` (`indexer.go:5832-5837`) | exact |
| `internal/indexer/workspace_resolve.go` (MOD, policy wiring) | indexer → resolver wiring | request-response | `newCrossRepoResolver` :492-505, `buildCrossWorkspaceRuleSnapshot` :565-587, `collectCrossWorkspaceRules` :599+ | exact |
| `internal/config/config.go` (MOD, `index.cobol`) | config | — | `CrossWorkspaceDep` :191-195, `IndexConfig` :632+, `validateWorkspaceSchema` :1914-1949 | exact |
| `internal/parser/languages/cobol_grammar.go` (MOD: `/5`, `MaterialIdentity`) | extractor | transform | itself, :25-49; optional capability idiom `parser.BoundedExtractor` (`extraction_limits.go:100-106`) | exact |
| `internal/parser/languages/cobol_grammar_unsupported.go` (MOD) | extractor stub | — | itself, :28-49 | exact |
| `internal/parser/languages/cobol_grammar_identity.go` (MOD, IN-03) | extractor | transform | itself, :107-118 (program symbol) and :212-226 (`cobolIDSegment`) | exact |
| `internal/parser/languages/cobol_grammar_walk.go` (MOD, CR-02) | extractor | transform | `resourceNode` :230-243 | exact |
| `internal/parser/languages/cobol_grammar_refs.go` (MOD, `cobol_placeholder`) | extractor | transform | `cobolCallTarget` :53-92, `cobolCopy` :107-175 | exact |
| `internal/mcp/source_scope.go` (NEW, D-13/D-20) | middleware (guard) | request-response | `nodeInSessionScope` (`server.go:2456-2465`) + `repoPrefixVisible` (`scope_resolve.go:516-526`) + `soleTrackedRepo` (`tools_fileops.go:202-215`) | role-match |
| `internal/mcp/tools_fileops.go`, `internal/mcp/view_files.go` (MOD, gate call sites) | middleware call sites | request-response | `resolveFilePath` :68-110 (deferred guard), `resolveNodePath` :474-517, `readViewFile` :203-213 | exact |
| `internal/mcp/source_scope_test.go` (NEW) | test | — | `newTwoRepoServer` (`scope_resolve_test.go:804-855`) | exact |
| `internal/mcp/tools_enhancements.go` (MOD, D-15) | controller (`index_health`) | request-response | itself, :3483-3501 | exact |
| `cmd/gortex/daemon_state.go` (MOD, D-14 log) | controller | — | `logRepoOwnershipAudit` :1268-1300 | exact |
| `cmd/gortex/mcp.go` (MOD, embedded single repo) | entry point | — | itself, :485-498; `internal/serverstack/shared_server.go:577-579`; `multi.go:2395-2406` | exact |
| `internal/graph/store_sqlite/schema_version.go` (MOD, v22) | migration | batch | itself, :37, :97-103, :120 + `migration_unprefixed_purge.go:8-50` | exact |
| `internal/testutil/graphdump/graphdump.go` (NEW) | test utility | transform | `internal/testutil/graphfixture/fixture.go` (package shape) + `canonicalGraph` (`incremental_reindex_test.go:83-104`) | role-match |
| `internal/indexer/cobol_lifecycle_test.go` (NEW, D-04) | test (real path) | batch | `indexCobolBreadthRepos` (`cmd/gortex/cobol_breadth_test.go:191-240`) + `TestIncrementalReindex_ConvergesToFullIndex` + `cobol_bom_test.go` | composite |
| `cmd/gortex/cobol_scope_test.go` (NEW, SEC-01 cross-surface) | test (integration) | request-response | `cmd/gortex/cobol_breadth_test.go` + `internal/mcp/scope_resolve_test.go:804` | role-match |
| `internal/resolver/cobol_references_test.go`, `cross_repo_test.go` (MOD/extend) | test | — | themselves (`seedCobol` :18-34, `cobolEdge` :37-47, `TestCrossRepoResolveAll_CrossRepoFallback` :57-80) | exact |
| `cmd/gortex/cobol_tracer_test.go`, `cmd/gortex/cobol_breadth_test.go`, `internal/parser/languages/cobol_grammar_contract_test.go` (MOD) | test | — | they pin `gortex-cobol-grammar/4`; the pins move to `/5` | exact |

---

## Pattern Assignments

### `internal/graph/scope.go` (NEW, model + validator, D-16)

**Analog:** `internal/graph/store_sqlite/scoped_projection.go:22-62`. This is the only fail-closed validator in the codebase. Move its body here and do not rewrite it.

**Core validation to lift** (lines 22-50):
```go
func (s *Store) OpenScopedProjectionSnapshot(ctx context.Context, scope graph.ProjectionScope) (graph.ScopedProjectionSnapshot, error) {
	if len(scope.Repositories) == 0 {
		return nil, fmt.Errorf("scoped projection: repository allow-set is required")
	}
	scope.Workspace = strings.TrimSpace(scope.Workspace)
	scope.Project = strings.TrimSpace(scope.Project)
	if scope.Workspace == "" || scope.Project == "" {          // -> Scope.RequireWorkspaceProject()
		return nil, fmt.Errorf("scoped projection: workspace and project are required")
	}
	...
	seen := make(map[string]struct{}, len(scope.Repositories))
	repos := make([]string, 0, len(scope.Repositories))
	for _, repo := range scope.Repositories {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			return nil, fmt.Errorf("scoped projection: repository allow-set contains an empty value")
		}
		if _, ok := seen[repo]; ok { continue }
		seen[repo] = struct{}{}
		repos = append(repos, repo)
	}
	if len(repos) == 0 { return nil, fmt.Errorf("scoped projection: repository allow-set is required") }
	sort.Strings(repos)
```
After the move, the store keeps only the `ctx.Err()` check, `BeginTx`, and the descriptor. Keep the existing error substrings: `TestScopedProjectionTracer` asserts `"repository allow-set is required"` (`store_sqlite/scoped_projection_test.go:88`).

**Type to rename** (`internal/graph/scoped_projection.go:18-23`):
```go
// ProjectionScope is the exact non-empty source boundary for a projection.
type ProjectionScope struct {
	Workspace    string
	Project      string
	Repositories []string
}
```
Rename it to `Scope` in `scope.go` and leave `type ProjectionScope = Scope` in `scoped_projection.go`. `ScopedProjectionSnapshotOpener` (:41-43) and `ProjectionSnapshotDescriptor.Scope` (:28) keep compiling unchanged.

**`Allows` must NOT copy `repoNarrowAdmits`.** `repoNarrowAdmits` admits empty prefixes (`internal/mcp/tools_core.go:805-810`):
```go
func repoNarrowAdmits(allowed map[string]bool, repoPrefix string) bool {
	if len(allowed) == 0 { return true }
	return repoPrefix == "" || allowed[repoPrefix]   // <- fail-open on "" ; Scope.Allows must reject ""
}
```

**Behavior divergence to resolve:** `neo4jprojection.NormalizeRequest` (`service.go:138-145`) silently **drops** blank repo entries (`slices.DeleteFunc(... repo == "")`), while the store **rejects** them. If `NormalizeRequest` calls `NewScope`, a blank entry becomes an error. That matches D-16's fail-closed intent, but neo4jprojection tests that pass blank repos will change. The planner should pick one behavior explicitly.

**Doc style:** use the house `PURPOSE — / RATIONALE — / KEYWORDS —` block (example: `cobol_references.go:32-67`).

---

### `internal/graph/prefix_diagnostics.go` + `store_sqlite/prefix_diagnostics.go` (MOD, D-14 strict)

**Analog:** the files themselves.

**Change site: `Clean()`** (lines 70-74):
```go
// Clean reports whether the audit found no ownership problems. It tolerates
// a wholly-unowned graph, which is what single-repo mode looks like.
func (d PrefixDiagnostics) Clean() bool {
	return !d.Mixed() && d.MisprefixedNodes == 0
}
```
The new rule is `UnownedCodeNodes == 0 && MisprefixedNodes == 0`. Also update the struct doc at :34-38 ("a lone-repo daemon legitimately reports the whole graph as unowned").

**Exemption list, implicit today. It lives in two places that must stay in lockstep** (Go :117-149):
```go
const (
	externalSemanticFilePathPrefix = "external::"
	externalCallFilePathPrefix     = "external-call::"
)
func IsAuditableRepoSourcePath(filePath string) bool {
	return filePath != "" &&
		!strings.HasPrefix(filePath, externalSemanticFilePathPrefix) &&
		!strings.HasPrefix(filePath, externalCallFilePathPrefix)
}
func IsAuditableRepoSourceNode(n *Node) bool {
	...
	switch n.Kind {
	case KindContract, KindContractBridge, KindTopic, KindRationale:
		return false
```
SQLite mirror (`store_sqlite/prefix_diagnostics.go:26-29`):
```go
auditableRepoSourceNodePredicate = `file_path <> '' AND ` +
	`kind NOT IN ('contract', 'contract_bridge', 'topic', 'rationale') AND ` +
	`instr(file_path, 'external::') <> 1 AND ` +
	`instr(file_path, 'external-call::') <> 1`
```
Name the exemption list once in Go, with a doc comment. Keep the SQL literal equal to it. The existing comment "The exact same path prefixes are excluded by the SQLite ownership predicates" (:122-124) is the contract. The storetest fence is the parity test.

**Tests to flip:**
- `prefix_diagnostics_test.go` subtest `"wholly unprefixed is single-repo mode, not a defect"` (around file line 115) currently asserts `!d.Mixed()` and `d.Clean()` for `{unowned, global}`. Under D-14, `Clean()` must be false.
- `storetest.go` `testPrefixDiagnostics` (around line 204). Its fixture of exempt nodes (`dep::`, `external::`, `external-call::`, contract, contract-bridge, topic, rationale) is the enumerated exemption list. Keep those rows. Only the `Clean()` / unowned expectations change.

---

### `internal/resolver/cobol_references.go` (MOD → `reconcileCobol`, batch)

**Analog:** itself (:68-188). Keep the structure: candidate index, then per-edge decide, then two write paths.

**Existing reason vocabulary to reuse, not rename** (:19-21):
```go
metaCobolUnresolvedReason = "cobol_unresolved_reason"
metaCobolCandidateIDs     = "cobol_candidate_ids"
```
RESEARCH A1 proposes `resolution_unresolved_reason` / `resolution_candidate_ids` only for the **generic** D-12 path. The COBOL path already has persisted, projected names. Keep them.

**Candidate index keyed by repo** (:171-188). For D-09, look up `index[repo+"\x00"+key]` for every repo in the allow-set instead of only `r.callerRepoPrefix(e)`:
```go
k := n.RepoPrefix + "\x00" + key
index[k] = append(index[k], n)
```

**Program preference ladder, kept inside the caller repo** (:100-126): contained, then sameFile, then otherFile. Unknown `cobol_nested` fails closed (:106-107).

**Bind / stamp / persist** (:128-166). This is the exact write pattern for the reconcile diffs:
```go
if len(candidates) == 1 {
	oldTo := e.To
	e.To = candidates[0].ID
	e.Origin = graph.OriginASTResolved
	delete(e.Meta, metaCobolUnresolvedReason)
	delete(e.Meta, metaCobolCandidateIDs)
	rebound = append(rebound, graph.EdgeReindex{Edge: e, OldTo: oldTo})
	continue
}
...  // 0 -> "missing"; >=2 -> "ambiguous" + sort.Strings(ids)
if len(rebound) > 0 {
	r.noteImportEdgeReindexes(rebound)
	r.graph.ReindexEdges(rebound)
}
if len(stamped) > 0 {
	if p, ok := r.graph.(graph.EdgeMetaBatchPersister); ok {
		p.PersistEdgeAttributesBatch(stamped)
	}
}
```
New: a bound edge that must return to unresolved (a new second candidate, or a deleted target) goes through `ReindexEdges` with `To = e.Meta["cobol_placeholder"]`. It is not a meta-only stamp, because `To` changes.

**Edge enumeration:** placeholders come from `r.graph.EdgesWithUnresolvedTarget()` (:77). Bound edges come from `r.graph.GetInEdgesByNodeIDs(candidateIDs)`, the same primitive `restubIncomingRefs` uses (`indexer.go:5240`). Filter both by `cobol_placeholder` present and caller repo in S.

**Call site:** `internal/resolver/resolver.go:1660-1662`, which runs serially after `resolveLuaRequires`:
```go
r.resolveLuaRequires()
...
r.resolveCobolReferences()
```

**Namespace exclusions that must stay:** `resolver.go:3379` and `cross_repo.go:1121` (`case strings.HasPrefix(target, "cobol::"): stats.Unresolved++`).

**Test helpers to reuse** (`internal/resolver/cobol_references_test.go:18-58`): `seedCobol(g, repo, id, kind, cobolKind, key)`, `cobolEdge(g, from, line, kind, to, meta)`, and `cobolEdgeAt(t, g, from, line)`. New `TestCobolReconcile_*` tests go in the same file or a sibling `_test.go` in `package resolver`.

---

### `internal/resolver/cobol_policy.go` (NEW, per-repo policy lookup)

**Analog:** `CrossWorkspaceDepLookup` (`internal/resolver/cross_repo.go:43-47, 220-228`):
```go
// CrossWorkspaceDepLookup returns the list of declared cross-workspace
// dependencies for a *source* workspace. Empty / nil result means ...
type CrossWorkspaceDepLookup func(sourceWorkspaceID string) []CrossWorkspaceDepRule

// SetCrossWorkspaceDepLookup wires the boundary rule. After this
// call, the resolver will refuse cross-workspace candidates that
// aren't covered by an explicit declaration ...
func (cr *CrossRepoResolver) SetCrossWorkspaceDepLookup(lookup CrossWorkspaceDepLookup) {
	cr.crossWorkspaceLookup = lookup
}
```
Mirror this as `type CobolPolicyLookup func(repoPrefix string) CobolPolicy` and `(*Resolver).SetCobolPolicyLookup`. Neighboring setters on `Resolver` are at `resolver.go:525-562` (`SetLogger`, `SetScope`, `SetStampTerminal`, `SetGraph`). A nil lookup means same-repo only, which is today's behavior.

**Wiring and frozen-per-pass snapshot analog** (`internal/indexer/workspace_resolve.go:492-505, 565-587`):
```go
cr := resolver.NewCrossRepo(mi.graph)
cr.SetLogger(mi.logger)
cr.SetCrossWorkspaceDepLookup(mi.crossWorkspacePassLookup())
...
func (mi *MultiIndexer) buildCrossWorkspaceRuleSnapshot() *crossWorkspaceRuleSnapshot {
	...
	for {
		revision := mi.configMgr.Revision()
		rulesByWorkspace, repoCount := mi.collectCrossWorkspaceRules()
		if revision == mi.configMgr.Revision() {
			return &crossWorkspaceRuleSnapshot{revision: revision, repoCount: repoCount, rulesByWorkspace: rulesByWorkspace}
		}
	}
}
```
`collectCrossWorkspaceRules` (:599+) iterates sorted repo prefixes under `mi.mu.RLock()` and reads `mi.configMgr` per repo. Use the same shape to build `map[repo]CobolPolicy` and the reverse-allowlist fan-out set S (RESEARCH Pitfall 5).

---

### `internal/resolver/cobol_resources.go` (NEW, D-11 alias edges + CR-02 GC)

**Analog:** the write tail of `cobol_references.go:155-166` (above) for edges, plus `graph.Store.EvictFile` for GC. There is no node-delete primitive (RESEARCH, `graph/store.go:160-237`).

**Edge kind:** `graph.EdgeAliases` already exists (`internal/graph/edge.go:257-260`, listed in the kind set at :841). Today it means `type X = Y`. Reusing it for `_unqualified → qualified` needs a doc-comment line added at edge.go:257.

**Projection screen for every new meta key** (`internal/neo4jprojection/properties.go:171-180`): it lower-cases the key, strips `-_. `, and rejects `password|passwd|secret|token|credential|privatekey|apikey|authorization`. `cobol_placeholder`, `cobol_resource_raw`, `cobol_copy_search_rank`, `cobol_copy_search_rule`, and `cobol_qualifier_rule` all pass. Add an assertion in a unit test.

---

### `internal/resolver/cross_repo.go` (MOD, D-12 minimal edit)

**Analog in the same file:** `pickQualNameCandidate` (:324-352) already refuses more than one eligible foreign candidate:
```go
var eligible *graph.Node
for _, candidate := range candidates {
	if candidate == nil || !cr.crossWorkspaceEligible(callerWS, candidateWorkspaceID(candidate), qualName) {
		continue
	}
	if eligible != nil {
		return nil, true      // ambiguous: stay unresolved rather than bind by row order
	}
	eligible = candidate
}
return eligible, false
```
Deterministic tie-break helper (`internal/resolver/qual_name_candidates.go:7-18`):
```go
func lowestIDQualNameCandidate(candidates []*graph.Node, accept func(*graph.Node) bool) *graph.Node {
	var picked *graph.Node
	for _, candidate := range candidates {
		if candidate == nil || !accept(candidate) { continue }
		if picked == nil || candidate.ID < picked.ID { picked = candidate }
	}
	return picked
}
```

**Edit site 1, `resolveFunctionCall` step 2** (:1297-1323). It returns the first match in `scopedCandidates` order, and that order comes from `for r := range reachableRepos` (:1036, map iteration):
```go
for _, c := range candidates {
	if c.Kind != graph.KindFunction && c.Kind != graph.KindMethod { continue }
	if !reachable(c.RepoPrefix) { continue }
	if !cr.crossWorkspaceEligible(callerWS, candidateWorkspaceID(c), "") { continue }
	e.To = c.ID
	stats.Resolved++
	if isCrossRepoHop(callerRepo, c.RepoPrefix) { e.CrossRepo = true; stats.CrossRepoEdges++; stats.ByRepo[c.RepoPrefix]++ }
	return
}
stats.Unresolved++
```
Collect the eligible candidates instead. If they span more than one `RepoPrefix`, stamp the reason and sorted IDs and count as unresolved. Within one repo, use `lowestIDQualNameCandidate`.

**Edit site 2, `resolveMethodCall`** (:1525-1655) has **three** cross-repo fallback loops with the same `reachable(c.RepoPrefix)` then `e.To = c.ID` shape: around :1556-1566, :1600-1610, and :1626-1636. RESEARCH mentions only one. Each has a same-repo loop before it (:1543, :1593, :1619) that stays unchanged. Three identical edits make a private helper justified (rule of three) if the planner wants one.

**Persisting the meta stamp:** `resolveEdge` (:1087-1138) only batches when `e.To != oldTo`. A meta-only ambiguous stamp therefore needs its own path, such as collecting into a slice and calling `graph.EdgeMetaBatchPersister.PersistEdgeAttributesBatch`, the same as `cobol_references.go:162-166`.

**Tests:** extend `internal/resolver/cross_repo_test.go`. `TestCrossRepoResolveAll_CrossRepoFallback` (:57-80) and its `wireImport(g, callerFile, targetRepo, targetFile)` helper are the template. Two `wireImport`s to two repos with the same name should give `ambiguous`.

---

### `internal/indexer/cobol_material.go` (NEW, D-06/D-07 fingerprint)

**Analog:** `indexConfigHash` (`internal/indexer/checkout_coordinator.go:1820-1835`). It is the existing canonical-JSON config hash, including the sort/compact normalization and an "unencodable means unique" fallback:
```go
func indexConfigHash(cfg config.IndexConfig) string {
	normalized := cfg
	if cfg.FrameworkSynthesizers != nil {
		names := slices.Clone(*cfg.FrameworkSynthesizers)
		slices.Sort(names)
		names = slices.Compact(names)
		normalized.FrameworkSynthesizers = &names
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "unhashable-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	sum := sha256.Sum256(encoded)
```
Swap in BLAKE3 the way `file_meta.go:65-68` does it:
```go
func contentHashForSource(src []byte) string {
	sum := blake3.Sum256(src)
	return hex.EncodeToString(sum[:])
}
```
Hash only `idx.config.Cobol` plus the extractor's `MaterialIdentity()`, never the whole `IndexConfig` (D-06).

**Extractor lookup without importing `languages`** follows `internal/indexer/file_delta.go:127`: `ext, _ := idx.registry.GetByLanguage(lang)`, then type-assert the optional capability. The same assertion idiom appears at `indexer.go:3494-3495` (`walkExt.(parser.StreamingExtractor)`).

---

### `internal/indexer/extractor_version.go` + `merkle_reindex.go` (MOD)

**Analog:** itself.

**Salt function to extend** (:127-138):
```go
func merkleSaltFor(rel string) string {
	lang := extractorSaltExtLang[strings.ToLower(filepath.Ext(rel))]
	if lang == "" { return "" }
	policySalt := postExtractionPolicySnapshotKey + "@" + strconv.Itoa(postExtractionPolicyVersion)
	v := extractorVersionForLang(lang)
	if v <= 1 { return policySalt }
	return policySalt + "|" + lang + "@" + strconv.Itoa(v)
}
```
There are only two callers, and both pass the free function: `merkle_reindex.go:92` `merkle.Build(rootAbs, rels, prior, merkleSaltFor)` and `:118` `merkle.BuildWithKnownFiles(..., merkleSaltFor, knownFiles)`. Change them to a closure over `idx` that appends `|cobol-material@<fp>` when `lang == "cobol"`.

**Map entry to drop** (:48): `"cobol": 2, // nested-program CALL scope ... (gortex-cobol-grammar/4)`. Keep the `.cob/.cbl/.cpy → "cobol"` mapping (:101-103), because it carries the global policy salt.

**Reserved-key precedent** (if A3 picks "no migration"): `postExtractionPolicySnapshotKey` (:15-21) and its special case in `staleLangsBetween` (:179-189). A material-fingerprint key needs `!=` semantics there, not the `<` used for versions at :204.

**Non-Merkle stale-set reader** (:223-248), `extractorVersionStaleLangSet`, is where a stored-vs-current fingerprint mismatch adds `"cobol"` to the set. Fail-safe pattern: on read error, Warn and return nil.

---

### `internal/indexer/index_state.go` (+ optional column) (MOD, CRUD)

**Analog:** itself.

**Write path** (:34-56). The fingerprint is persisted beside `ExtractorVersions`:
```go
vers, _ := json.Marshal(extractorVersionsSnapshot())
st := graph.RepoIndexState{
	RepoPrefix: idx.repoPrefix, IndexedSHA: sha, Dirty: dirty, IndexedAt: time.Now().Unix(),
	WorkspaceFP: workspaceFP, NodeCount: nodes, EdgeCount: edges, ExtractorVersions: string(vers),
}
if err := w.SetRepoIndexState(st); err != nil {
	idx.logger.Warn("persist repo index state failed", zap.String("repo", idx.repoPrefix), zap.Error(err))
}
```
**Read-modify-write precedent** (:61-94), `persistExtractorVersion`. Use it as the template for "advance only the COBOL fingerprint after a scoped restage".

**If the A3 column is chosen**, the full change set is: `graph.RepoIndexState` (`internal/graph/index_state.go:13-22`), INSERT and SELECT in `internal/graph/store_sqlite/store_index_state.go:19-37` (column lists at :21 and :32), `read_index_state.go:61-65`, `schema.go`, and an idempotent `ADD COLUMN` step following `addEdgeViewGenerationColumn` (`schema_version.go:426-439`: probe `pragma_table_xinfo`, then `ALTER TABLE ... ADD COLUMN`). The reserved-key option touches only `extractor_version.go`.

**D-17 (re-stamp `prov_vcs_commit` on HEAD move):** the HEAD-move path ends in `reconcileRepoIndexState` (:105-155). The per-file stamp logic to reuse is `stampSourceRevision` (`source_revision.go:44-86`). It uses `classifySourceRevision(idx.rootPath, relPath, src)` and then a closure that sets `prov_vcs_commit` or `prov_vcs_commit_absence` on every Meta that carries `prov_revision_content_id`. A metadata-only refresh needs those same two keys written to the stored node and edge Meta. For edges, persist through `EdgeMetaBatchPersister`.

---

### `internal/indexer/affected_by.go` + `incremental_batch.go` (MOD, D-08)

**Analog:** the two identical truncation sites.

`affected_by.go:556-566`:
```go
if maxFiles := idx.affectedByMaxFiles(); len(files) > maxFiles {
	idx.logger.Debug("affected-by: re-resolve set truncated",
		zap.String("file", changedPath),
		zap.Int("affected", len(files)),
		zap.Int("cap", maxFiles),
		zap.Int("dropped", len(files)-maxFiles))
	files = files[:maxFiles]
}
idx.resolver.ResolveFilesAndIncoming(files)
resolver.SynthesizeExternalCallsForFiles(idx.graph, idx.externalCallSynthesisEnabled(), files)
idx.persistRefFactsForFiles(files)
```
`incremental_batch.go:1494-1499` has the same block inside the `filesByChanged` loop.

The edit is the same at both sites: `Debug` becomes `Info`, the message becomes `"affected-by: cap exceeded, resolving full affected set"` with `affected` and `cap`, and the `files = files[:maxFiles]` line is removed. Leave `builder_closure.go:85-91` alone (RESEARCH).

**Doc to update:** `IndexConfig.AffectedByReresolveMax` (`internal/config/config.go:762-772`) says "the affected set is truncated at this bound".

**Test to rewrite:** `TestAffectedBy_CapBoundsFanout` (`affected_by_e2e_test.go:207-236`). Keep the zap observer pattern:
```go
core, observed := observer.New(zap.DebugLevel)
idx := New(g, reg, cfg, zap.New(core))
...
entries := observed.FilterMessage("affected-by: re-resolve set truncated").All()
require.Len(t, entries, 1)
fields := entries[0].ContextMap()
assert.EqualValues(t, 3, fields["affected"])
```
Also update `TestAffectedBy_DeferredBatchPath_NoFanout` (:242-266), which filters on the old message string.

---

### `internal/indexer/indexer.go` `restubIncomingRefs` + `incremental_batch.go` `restubIncomingRefsFromView` (MOD, CR-01)

**Analog:** the two functions themselves. They are near-duplicates, so the COBOL branch goes into both, with a `// note:` comment or a package-private helper in `indexer`.

`indexer.go:5218-5271` (excerpt):
```go
for _, n := range nodes {
	if n == nil || n.Name == "" || !graph.IsReferenceableSymbol(n.Kind) {   // excludes KindFile copybooks
		continue
	}
	stub := graph.UnresolvedMarker + n.Name
	for _, e := range inEdges[n.ID] {
		if e == nil || !graph.IsResolvableRefEdge(e.Kind) {                   // excludes EdgeImports (COPY)
			continue
		}
		if _, fromEvicted := evicted[e.From]; fromEvicted { continue }
		if graph.IsUnresolvedTarget(e.To) { continue }
		oldTo := e.To
		graph.StashRestubProvenance(e)        // MUST NOT run for COBOL (restub_prev_* leak into dump)
		e.To = stub
		batch = append(batch, graph.EdgeReindex{Edge: e, OldTo: oldTo})
	}
}
```
`incremental_batch.go:957-987` has the same loop over `stage.priorNodes` and `view.inByNode`. **Caveat:** `view.inByNode` is filled from every prior node ID (`incremental_batch.go:403-419`), so copybook `KindFile` in-edges are already fetched there. In `restubIncomingRefs`, `refIDs` (:5224-5233) is filtered by `IsReferenceableSymbol`, so COBOL `KindFile` nodes need to be added to `refIDs` explicitly.

Stash keys being avoided: `internal/graph/restub_provenance.go:20-25` (`restub_prev_to`, `restub_prev_origin`, `restub_prev_tier`, `restub_prev_conf`).

---

### `internal/indexer/incremental_watcher_batch.go` (MOD, reconcile trigger)

**Analog:** the three tail functions. Each has the same "frontier, then hook-or-resolve" shape, and the test seam is honored at every site:
```go
files, needed, exact := incrementalResolutionFrontier(result, receipt)
if needed && len(files) > 0 {
	idx.runIncrementalResolutionCatchup(files, batch, func(frontier []string) {
		if idx.incrementalResolveFilesHook != nil {
			idx.incrementalResolveFilesHook(frontier)
			return
		}
		idx.resolver.ResolveFilesAndIncoming(frontier)
	})
	...
} else if needed && !exact {
	idx.observeIncrementalCatchup("resolve", nil)
	idx.ResolveAll()
}
```
Sites: `incrementalEvictWatcherPath` :240-278, `incrementalWatcherPaths` :291-328, and `MultiIndexer.resolveIncrementalRepoMutationMode` :376-440. The multi path dispatches through `mi.runMasterResolveFiles` and `mi.runCrossRepoResolveFiles`. Place the COBOL reconcile call **outside** the `needed` branch, because `needed=false` skips resolution (RESEARCH anti-pattern 1). Use `idx.observeIncrementalCatchup("cobol", ...)` to make it observable to `incrementalCatchupHook` (`indexer.go:530-533`). COBOL detection: `ExtractorLangForFile(rel) == "cobol"` (`extractor_version.go:110-112`).

---

### `internal/indexer/multi.go` (MOD, config push)

**Analog:** `RefreshRepoConfigs` (:2457-2486) together with the `SetExcludePatterns` setter (`indexer.go:5832-5837`):
```go
if idx != nil {
	idx.SetExcludePatterns(cfg.Index.Exclude)
}
...
func (idx *Indexer) SetExcludePatterns(patterns []string) {
	idx.excludeMu.Lock()
	defer idx.excludeMu.Unlock()
	idx.config.Exclude = patterns
	idx.excludes.Store(excludes.New(effectiveExcludePatterns(patterns)))
}
```
Add one line, `idx.SetCobolConfig(cfg.Index.Cobol)`, and a mutex-guarded setter with the same shape. Prefix stamping for all repos, including a lone repo, is already uniform at :2395-2406 (`idx.SetRepoPrefix(repoPrefix)`, `SetWorkspaceID`, `SetProjectID`).

---

### `internal/config/config.go` (MOD, `index.cobol`)

**Analog:** `CrossWorkspaceDep` (:191-195), a per-repo `.gortex.yaml` block with `mapstructure`+`yaml` tags:
```go
type CrossWorkspaceDep struct {
	Workspace string   `mapstructure:"workspace" yaml:"workspace"`
	Modules   []string `mapstructure:"modules"   yaml:"modules"`
	Mode      string   `mapstructure:"mode"      yaml:"mode"`
}
```
Put the field on `IndexConfig` (:632+), shaped like `Coverage CoverageConfig \`mapstructure:"coverage" yaml:"coverage,omitempty"\`` (:789).

**Validation analog:** `validateWorkspaceSchema` (:1914-1949) concatenates errors:
```go
for _, dep := range c.CrossWorkspaceDeps {
	if dep.Workspace == "" {
		errs = append(errs, "config: cross_workspace_deps[].workspace is required")
	}
	...
}
if len(errs) == 0 { return nil }
return fmt.Errorf("%s", strings.Join(errs, "; "))
```
Add `index.cobol` checks with the same `errs = append(...)` style: no `*`, no `..`, not absolute, and defaults non-empty after trimming.

---

### `internal/parser/languages/cobol_grammar*.go` (MOD, extractor `/5`)

**Version constants** (`cobol_grammar.go:25-49`): `cobolApprovedGrammarID` :32, `cobolApprovedPreprocessorModule` :39, `cobolGrammarExtractorVersion = "gortex-cobol-grammar/4"` :42. Name-norm rule: `cobolNameNormRule = "cobol-name-norm/1"` (`cobol_grammar_identity.go:152`).

**Optional-capability idiom** (`internal/parser/extraction_limits.go:100-106`):
```go
// BoundedExtractor is an optional Extractor capability for request-scoped
// cancellation and hard output/result envelopes. Legacy callers continue to
// use Extract unchanged.
type BoundedExtractor interface {
	Extractor
	ExtractBounded(context.Context, string, []byte, ExtractionLimits) (*ExtractionResult, ExtractionUsage, error)
}
```
If the interface is declared in `parser`, follow this exact doc and shape. The `!darwin && !linux` stub (`cobol_grammar_unsupported.go:28-49`) ends with `var _ parser.Extractor = (*CobolGrammarExtractor)(nil)`. Add the method there too, or make the indexer tolerate a missing capability.

**IN-03 site** (`cobol_grammar_identity.go:107-118`). `name` is used raw:
```go
base, container := name, ""
if len(open) > 0 {
	container = open[len(open)-1].symbol
	base = container + "/" + name
	...
}
occurrences[base]++
symbol := base
if k := occurrences[base]; k >= 2 { symbol += "#" + strconv.Itoa(k) }
```
Wrap `name` with `cobolIDSegment` (:212-226, which escapes `% : / #` and control bytes). The same encoding is already used for placeholder keys in `cobol_grammar_refs.go:60,138`.

**CR-02 site** (`cobol_grammar_walk.go:230-243`):
```go
m := maps.Clone(w.docMeta)     // writer-dependent prov_* -> must be filtered
maps.Copy(m, meta)
m["prov_range_exact"] = false
m["prov_range_absence"] = "shared_resource"
w.result.Nodes = append(w.result.Nodes, &graph.Node{
	ID: id, Kind: kind, Name: name, FilePath: w.filePath, Language: "cobol", Meta: m,   // FilePath -> id ; Name -> normalized key (D-18)
})
```
The doc comment above it (:223-229) already says "shared-node ownership is Phase 4". Rewrite it.

**`cobol_placeholder` sites:** `cobolCallTarget` (`cobol_grammar_refs.go:53-92`) sets `edge.Meta["cobol_call"]`, `cobol_target_raw`, and `cobol_name_norm`. Literal branch :84-86: add `edge.Meta["cobol_placeholder"] = to` there. `cobolCopy` (:107-175) sets `cobol_copy_form` and `cobol_copy_member_raw` at :166-169. Add it there for non-IDMS forms only (IDMS is never bound, D-19).

**Pinned `/4` assertions to move to `/5`:** `cmd/gortex/cobol_breadth_test.go`, `cmd/gortex/cobol_tracer_test.go`, `internal/parser/languages/cobol_grammar_contract_test.go` (grep-verified).

---

### `internal/mcp/source_scope.go` (NEW, guard) + call sites in `tools_fileops.go` / `view_files.go`

**Analogs:**
- Fail-open to replace (`internal/mcp/server.go:2456-2465`):
  ```go
  func (s *Server) nodeInSessionScope(ctx context.Context, n *graph.Node) bool {
  	opts, bound := s.sessionScopeOptions(ctx)
  	if !bound {
  		return true          // unbound session => everything (the SEC-02 hole)
  	}
  	...
  	return opts.ScopeAllows(n)
  }
  ```
- Single-repo no-op test (`tools_fileops.go:202-215`): `soleTrackedRepo(mi)` uses `mi.RepoPrefixes()`. `len != 1` means not solo. Use `len(s.multiIndexer.RepoPrefixes()) <= 1` for the no-op.
- Bound-session and allow-set test (`scope_resolve.go:516-526`, `repoPrefixVisible`): `s.sessionWorkspaceRepoSet(ctx)` then `repoAllowFromContext(ctx)` (`session_ctx.go:192-209`). Note that `withRepoAllow` stores nothing for an empty set, so "no allow-set in ctx" is the unbound signal.
- Unbound resolution with a nil allow-set (`scope_resolve.go:216-265`). `resolveUnboundScope` returns `RepoAllow = nil` for `repo:"*"` (:244-248) and when no narrowing applies.

**Choke points to gate:**
- `resolveFilePath` (`tools_fileops.go:68`). It already runs a **deferred** post-resolution guard, and that is the place to add the scope check without touching any return path:
  ```go
  defer func() {
  	if err == nil {
  		if guardErr := s.guardSymlinkWithinRepo(ctx, absPath); guardErr != nil {
  			absPath, relPath, err = "", "", guardErr
  		}
  	}
  }()
  ```
  Write tools pass through here too (D-20).
- `resolveNodePath` (`tools_fileops.go:474-517`). Insert after the nil and `FilePath` checks (:475-483) and before the `s.multiIndexer` branch (:487). `node.RepoPrefix` is available.
- `readViewFile` (`view_files.go:203-213`) is a **free function** `readViewFile(ctx, files, raw)`, not a `*Server` method. The gate needs a ctx-carried decision or a call at each of its 4 callers. Plan for this explicitly.

**Error-message parity** so the boundary cannot be probed (`tools_coding.go:958-963`):
```go
if !s.nodeInSessionScope(ctx, node) {
	return mcp.NewToolResultError("symbol not found: " + id), nil
}
```

---

### `internal/mcp/source_scope_test.go` (NEW) and `cmd/gortex/cobol_scope_test.go` (NEW)

**Analog:** `newTwoRepoServer` (`internal/mcp/scope_resolve_test.go:804-855`). It builds a real SQLite store with two repos under `GlobalConfig`, `NewMultiIndexer`, and `NewServer(..., MultiRepoOptions{MultiIndexer: mi, ConfigManager: cm, ...})`. Bound vs unbound sessions come from `sessionCtx("s-note", alphaDir)` vs `context.Background()`. Requests come from `makeReq("tool", map[string]any{...})` and a direct `srv.handleX(ctx, req)` call (:862-869). Inventory test: loop a table of source-returning handlers with an unbound ctx and assert `res.IsError`.

For the SEC-01 cross-surface test in `cmd/gortex`, the two-repo COBOL corpus helper is `indexCobolBreadthRepos` (`cmd/gortex/cobol_breadth_test.go:191-240`). It carries `//go:build darwin || linux`.

---

### `internal/mcp/tools_enhancements.go` (MOD, D-15) + `cmd/gortex/daemon_state.go` (MOD)

**Analog** (`tools_enhancements.go:3483-3501`):
```go
if prefixAuditOK {
	ownership := map[string]any{
		"owned_code_nodes":   prefixAudit.OwnedCodeNodes,
		"unowned_code_nodes": prefixAudit.UnownedCodeNodes,
		"consistent":         prefixAudit.Clean(),
	}
	if prefixAudit.MisprefixedNodes > 0 { ... }
	// Only surface samples when they point at a defect; on a
	// single-repo daemon every node is legitimately unowned ...
	if prefixAudit.Mixed() {
		ownership["mixed"] = true
		ownership["unowned_samples"] = prefixAudit.UnownedSamples
	}
	result["repo_ownership"] = ownership
}
```
Add `"status": "healthy"|"unhealthy"` derived from the strict `Clean()`. Surface `unowned_samples` whenever unowned is greater than 0, and rewrite the comment.

`logRepoOwnershipAudit` (`daemon_state.go:1268-1300`): the doc line :1277 ("A wholly-unowned graph is single-repo mode, not a defect") becomes wrong. The function already branches on `d.Clean()` (:1293), so the logic follows automatically once `Clean()` is strict.

---

### `cmd/gortex/mcp.go` (MOD, D-14 embedded single repo)

**Site** (:485-498): `result, err := idx.Index(mcpIndex)` runs with no `SetRepoPrefix`. **Analog to route through:** `internal/serverstack/shared_server.go:577-579` already builds `mi = indexer.NewMultiIndexer(g, reg, idx.Search(), cm, logger)`. `multi.go:2395-2406` shows uniform prefixing. **Blast radius to measure first:** `resolveNodePath`'s single-repo fallback (`tools_fileops.go:510-514`) joins `root + node.FilePath` without stripping a prefix.

---

### `internal/graph/store_sqlite/schema_version.go` (MOD, v22)

**Analog** (:37, :97-103, :113-120):
```go
const currentSchemaVersion = 21
...
{version: 7, name: "purge unprefixed solo-repo rows", inPlace: purgeUnprefixedRepoRows},
...
{version: 21, name: "persist per-file indexing failures", inPlace: createFileIndexFailuresTable},
```
Append `{version: 22, name: "purge unprefixed solo-repo rows (embedded)", inPlace: purgeUnprefixedRepoRows}` and bump to 22. The function's doc (`migration_unprefixed_purge.go:8-38`) explains why in-place beats rebuild, and the `file_path <> ''` guard keeps `dep::` externals. Rule (:45-46): "Steps are append-only and ascending; never edit or renumber a shipped one. Any inPlace step must be idempotent." If the A3 column is chosen as well, it becomes v23.

---

### `internal/testutil/graphdump/graphdump.go` (NEW, test-only helper)

**Package-shape analog:** `internal/testutil/graphfixture/fixture.go:1-3`:
```go
// Package graphfixture provides immutable SQLite graph fixtures for tests.
// Production packages must not import it.
package graphfixture
```
**Do not copy its `Canonical` (:105)**. That function reads raw SQL with `hex(meta)` and rowids (RESEARCH "Alternatives"). Read through `graph.ScopedProjectionSnapshotOpener` (`graph/scoped_projection.go:31-43`: `ReadNodePages`, `ReadEdgePages` yielding `[]ScopedEdgeRow{Edge, Source, Target}`).

**Rendering analog:** `canonicalGraph` (`internal/indexer/incremental_reindex_test.go:83-104`), which renders sorted `N|...` / `E|...` lines and joins them. Extend it with range columns, edge file and line, and canonical Meta (`json.Marshal` of a map sorts keys). Add a BLAKE3 digest using `blake3.Sum256` + `hex.EncodeToString` (`file_meta.go:65-68`).

---

### `internal/indexer/cobol_lifecycle_test.go` (NEW, D-04 harness, real path)

Composite of three analogs:
1. **Build tag, registry, and fixture rules:** `internal/indexer/cobol_bom_test.go:1-45`. It uses `//go:build darwin || linux`, `reg.Register(languages.NewCobolGrammarExtractor())`, and invented-COBOL constants with the comment "It must never be replaced with estate-derived source: this repository is public." Use `writeRepoFile(t, root, rel, content)`.
2. **Two repos, one SQLite store, MultiIndexer:** `cmd/gortex/cobol_breadth_test.go:191-240`:
   ```go
   store, err := store_sqlite.Open(filepath.Join(dir, "store.sqlite"))
   reg := parser.NewRegistry(); languages.RegisterAll(reg)
   repos.idx = indexer.New(store, reg, config.Default().Index, zap.NewNop())
   repos.cm, err = config.NewConfigManager(filepath.Join(dir, "config.yaml"))
   repos.mi = indexer.NewMultiIndexer(store, reg, repos.idx.Search(), repos.cm, zap.NewNop())
   t.Cleanup(func() { _ = repos.mi.Close(context.Background()); _ = store.Close() })
   resB, err := repos.mi.TrackRepoCtx(context.Background(), config.RepoEntry{Path: repos.rootB})
   resA, err := repos.mi.TrackRepoCtx(context.Background(), config.RepoEntry{Path: repos.rootA})
   ```
   That helper already carries `// note: second occurrence of indexCobolTracerRepo's store and indexer wiring`. A copy in `internal/indexer` would be the **third**, but it is in a different package (`main` vs `indexer`), so it cannot be shared without a `testutil` package. The planner decides: either a `// note:` third copy, or move the wiring into `internal/testutil` (rule of three met).
   Restart and retry over the same SQLite file: `checkout_lifecycle_test.go:157-184` (`f.open()` rebuilds the store, config manager, and MultiIndexer over the same `dbPath`).
3. **Edit, delete, and convergence:** `bumpMtime` (`incremental_reindex_test.go:106-114`, which writes the file and then `os.Chtimes(now+2s)`) and the shape of `TestIncrementalReindex_ConvergesToFullIndex` (:122-164): path A incremental, path B cold, then `assert.Equal(canonical(B), canonical(A))`.

Interruption seam: `idx.incrementalResolveFilesHook` (`indexer.go:526-529`), which every tail site honors (see incremental_watcher_batch excerpt above). Cap escalation: `cfg.AffectedByReresolveMax = 1`, as in `affected_by_e2e_test.go:220`.

---

## Shared Patterns

### Fail-closed scope validation (D-16)
**Source:** `internal/graph/store_sqlite/scoped_projection.go:22-50` → `graph.NewScope`
**Apply to:** projection snapshot, `neo4jprojection.NormalizeRequest`, the `reconcileCobol` allow-set, and `internal/mcp/source_scope.go`.
Rules: trim, reject an empty set or a blank entry, dedupe, sort, and never admit `RepoPrefix == ""`.

### Deterministic candidate decision
**Source:** `internal/resolver/cobol_references.go:128-152` (0 → missing, 1 → bind, ≥2 → ambiguous + `sort.Strings(ids)`) and `qual_name_candidates.go:7-18` (lowest ID).
**Apply to:** `reconcileCobol` (D-09/D-10/D-19) and `cross_repo.go` D-12 (function and the three method-call loops).

### Batched edge persistence
**Source:** `cobol_references.go:155-166`. `ReindexEdges` when `To` changes, `EdgeMetaBatchPersister.PersistEdgeAttributesBatch` for meta-only.
**Apply to:** reconcile, D-11 alias edges, D-12 ambiguous stamps, and the D-17 VCS re-stamp.

### Per-pass config snapshot into a resolver
**Source:** `workspace_resolve.go:492-505, 565-587` + `cross_repo.go:43-47, 220-228`
**Apply to:** COBOL policy (allowlist, libraries, default qualifiers), plus `multi.go:2457-2486` for live reload.

### Optional capability by type assertion
**Source:** `parser/extraction_limits.go:100-106`, `indexer.go:3494-3495`, `graph/prefix_diagnostics.go:227-237` (`ReadPrefixDiagnostics` probes for a capability, then falls back)
**Apply to:** `MaterialIdentity()`, fingerprint persistence, and edge meta persistence.

### Logging and errors
- zap structured fields with counts and prefixes only, never source text. Examples: `affected_by.go:557-561`, `index_state.go:53-54`.
- Wrap errors with `%w` (`store_sqlite/scoped_projection.go:53`). Config errors are concatenated (`config.go:1945-1948`).
- Warn on a fail-safe read error instead of silently skipping (`extractor_version.go:228-235`).

### Hashing
**Source:** `file_meta.go:65-68` (`blake3.Sum256` + hex) and `indexConfigHash` (`checkout_coordinator.go:1820-1835`, canonical JSON).
**Apply to:** the COBOL material fingerprint and the graphdump digest.

### New meta-key screening
**Source:** `neo4jprojection/properties.go:171-180` `sensitiveMetadataKey`
**Apply to:** every new `cobol_*` / `resolution_*` key. Add a test assertion.

### Test conventions
- COBOL tests use `//go:build darwin || linux` and invented fixtures only (`cobol_bom_test.go:1,18-24`).
- Real path: `store_sqlite.Open(filepath.Join(t.TempDir(), "g.sqlite"))` with `t.Cleanup(store.Close)` (`affected_by_e2e_test.go:21-32`).
- Use testify `require`/`assert`. Run with `GOWORK=off go test -race -count=1`.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| (partial) `internal/indexer/cobol_lifecycle_test.go` "interrupted reindex + retry" sequence | test | batch | No existing test combines `incrementalResolveFilesHook` (no-op) with a restart over the same SQLite file. The pieces exist (`indexer.go:526-529`, `checkout_lifecycle_test.go:157-184`), but the combination is new. RESEARCH lists it as a Wave 0 spike. |

Every other file has a direct in-repo analog. There is no external pattern source.

---

## Planner Flags (found while mapping)

1. **COBOL reason keys already exist:** `cobol_unresolved_reason` / `cobol_candidate_ids` (`cobol_references.go:20-21`). Do not introduce `resolution_*` names on COBOL edges. Those names are only for the generic D-12 path.
2. **`resolveMethodCall` has three cross-repo first-match loops** (≈:1556, :1600, :1626), not one.
3. **`readViewFile` is a free function** (`view_files.go:203`), so the D-13 gate cannot be a one-line `s.require...` call inside it.
4. **`NormalizeRequest` drops blank repos, but the store rejects them.** Unifying on `NewScope` changes neo4jprojection behavior.
5. **`restubIncomingRefs` refIDs exclude `KindFile`** (`indexer.go:5230`), but `restubIncomingRefsFromView` fetches in-edges for all prior nodes (`incremental_batch.go:403-419`). The CR-01 branch must close the first gap explicitly.
6. **The A3 column costs more than it looks.** It touches `store_index_state.go` INSERT/SELECT, `read_index_state.go`, `schema.go`, and a migration. The reserved-key option touches only `extractor_version.go` (with `!=` semantics in `staleLangsBetween`).
7. **The two-repo test-wiring helper is already duplicated twice** (`cobol_breadth_test.go:214` `// note:`). The D-04 harness would be the third occurrence.

## Metadata

**Analog search scope:** `internal/graph`, `internal/graph/store_sqlite`, `internal/graph/storetest`, `internal/resolver`, `internal/indexer`, `internal/parser`, `internal/parser/languages`, `internal/mcp`, `internal/config`, `internal/neo4jprojection`, `internal/serverstack`, `internal/testutil`, `cmd/gortex`
**Files scanned:** ~55 (source and `_test.go`, read directly)
**Pattern extraction date:** 2026-10-12
