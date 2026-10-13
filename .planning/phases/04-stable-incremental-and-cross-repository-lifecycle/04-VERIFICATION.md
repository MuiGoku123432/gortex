---
phase: 04-stable-incremental-and-cross-repository-lifecycle
verified: 2026-10-13T16:45:00Z
status: gaps_found
score: 86/89 must-haves verified (4/5 roadmap success criteria; 82/84 plan truths)
covered_files: [".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-SUMMARY.md","cmd/gortex/cobol_scope_test.go","cmd/gortex/daemon_state.go","cmd/gortex/mcp.go","internal/config/config.go","internal/config/manager.go","internal/graph/edge.go","internal/graph/prefix_diagnostics.go","internal/graph/scope.go","internal/graph/scoped_projection.go","internal/graph/store_sqlite/migration_unprefixed_purge.go","internal/graph/store_sqlite/prefix_diagnostics.go","internal/graph/store_sqlite/schema_version.go","internal/graph/store_sqlite/scoped_projection.go","internal/graph/storetest/storetest.go","internal/indexer/affected_by.go","internal/indexer/cobol_lifecycle_acceptance_test.go","internal/indexer/cobol_lifecycle_incremental_test.go","internal/indexer/cobol_lifecycle_material_test.go","internal/indexer/cobol_lifecycle_test.go","internal/indexer/cobol_material.go","internal/indexer/cobol_reconcile.go","internal/indexer/extractor_version.go","internal/indexer/incremental_batch.go","internal/indexer/incremental_watcher_batch.go","internal/indexer/index_state.go","internal/indexer/indexer.go","internal/indexer/merkle_reindex.go","internal/indexer/multi.go","internal/indexer/repository_untrack.go","internal/indexer/source_revision.go","internal/indexer/workspace_resolve.go","internal/mcp/change_contract.go","internal/mcp/overlay.go","internal/mcp/source_scope.go","internal/mcp/source_scope_test.go","internal/mcp/tools_coding.go","internal/mcp/tools_enhancements.go","internal/mcp/tools_fileops.go","internal/mcp/tools_find_declaration.go","internal/mcp/tools_search_text.go","internal/neo4jprojection/service.go","internal/parser/languages/cobol_grammar.go","internal/parser/languages/cobol_grammar_identity.go","internal/parser/languages/cobol_grammar_mainframe.go","internal/parser/languages/cobol_grammar_refs.go","internal/parser/languages/cobol_grammar_sql.go","internal/parser/languages/cobol_grammar_walk.go","internal/resolver/cobol_policy.go","internal/resolver/cobol_references.go","internal/resolver/cobol_resources.go","internal/resolver/cross_repo.go","internal/resolver/resolver.go","internal/testutil/graphdump/graphdump.go"]
covered_digest: "v3:sha256:b542823983962d360c495668018fbcc06b5e10e80e0179d4a0911a367fc7b260"
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "SC-5 / SEC-01 / D-13 / D-20: every source surface enforces the same repository scope; a repository outside the resolved scope is refused for source reads and writes (04-12 truths 1 and 2; 04-10 prohibition 'MUST NOT let two surfaces disagree about which nodes one repository allow-set admits')"
    status: failed
    reason: "Open review finding CR-01 confirmed. When a tracked repository's root is nested inside another tracked root, the outer repository's walk descends into it (Indexer.shouldPruneDir has no tracked-root check), so the nested repository's files are indexed a second time as outer-repository nodes. The CR-01 fix gates the physical owner only in resolveFilePath and the absolute branch of resolveGraphPath. resolveNodePath, the relative branch of resolveGraphPath, search_text, and find_declaration still gate only the lexical/node repository. A verifier probe (go test -overlay; tree untouched) with alpha at <tmp>/alpha and beta at <tmp>/alpha/nested showed: read_file {repo:alpha, path:alpha/nested/main.go} is refused, but get_symbol_source {repo:alpha} returns beta's bytes, search_text {repo:alpha} returns beta's line, and rename_symbol {repo:alpha, dry_run:false} rewrote beta's main.go on disk (\"func ZebraRenamed() {}\"). PrefixDiagnostics reported Clean=true (6 owned, 0 unowned, 0 misprefixed) while one file has two owners."
    artifacts:
      - path: "internal/indexer/indexer.go"
        issue: "shouldPruneDir (line ~5819) does not prune another tracked repository's root, so a nested tracked checkout is double-indexed under the outer repository's prefix (root cause; double ownership invisible to the SEC-08 audit)"
      - path: "internal/mcp/tools_fileops.go"
        issue: "resolveNodePath (line ~509) calls requireSourceScope(node.RepoPrefix) only; relative branch of resolveGraphPath (line ~598) calls requireSourceScope(matchedRepoPrefix(graphPath)) only; neither applies requirePathSourceScope's physical-owner check"
      - path: "internal/mcp/tools_search_text.go"
        issue: "filterTextMatchesByResolvedScope attributes matches by the lexical path prefix, not the physical owner"
      - path: "internal/mcp/tools_find_declaration.go"
        issue: "allowed() attributes use sites with matchedRepoPrefix(m.Path) (lexical)"
      - path: "internal/mcp/source_scope_test.go"
        issue: "TestSourceScope_NestedCheckoutRefused exercises read_file only, so it passes while get_symbol_source, search_text, and rename_symbol leak"
    missing:
      - "Prune other tracked roots in the per-repository walk (MultiIndexer hands each Indexer the sibling roots), which removes the double ownership at the source; OR apply requirePathSourceScope (physical owner) in resolveNodePath, the relative branch of resolveGraphPath, and the search_text / find_declaration attribution"
      - "Extend TestSourceScope_NestedCheckoutRefused with get_symbol_source, batch_symbols include_source, search_text bound to alpha, find_declaration, and a non-dry-run rename_symbol that must leave beta's file unchanged"
      - "If the root-cause fix is chosen, assert the nested fixture's PrefixDiagnostics has no alpha/nested/... nodes (SEC-08 double ownership)"
human_verification:
  - test: "Decide whether the rename_symbol out-of-scope refusal message is acceptable (review WR-04, recorded as accepted risk in 04-12)"
    expected: "Either accept the risk formally (add an override for the 04-12 privacy prohibition) or change the message to the generic errSourceScopeRequired / not-found text"
    why_human: "The 04-12 judgment-tier prohibition says refusals MUST NOT reveal whether something exists outside the caller's scope. A scoped dry_run rename returns a distinct 'some references lie outside the resolved scope' message, so it reveals that out-of-scope references exist. Accepting this is a policy decision."
  - test: "Review the concurrency fixes the fixer marked 'requires human verification' (REVIEW-FIX WR-01: cobolReconcileMu, deferred reconcile in EndBatch/ResetBatch) together with re-review WR-01 (untrack restub dropped under writer-gate contention) and WR-02 (new reconcile sites outside the reach topology gate)"
    expected: "Agree that these remain warnings (convergence still holds on every tested sequence), or schedule fixes"
    why_human: "These are contention- and ordering-dependent paths that the passing deterministic tests do not exercise (writer-gate timeout, reload during parallel warmup, reach cache staleness)"
---

# Phase 4: Stable Incremental and Cross-Repository Lifecycle Verification Report

**Phase Goal:** Users receive the same canonical scoped deterministic graph from identical final inputs regardless of reruns, edits, deletions, restores, or clean rebuilds.
**Verified:** 2026-10-13T16:45:00Z
**Status:** gaps_found
**Re-verification:** No. This is the initial verification. There was no prior 04-VERIFICATION.md.

## Goal Achievement

The lifecycle core of the phase is achieved and backed by real tests. These hold for the COBOL acceptance corpus:
- Incremental reconciliation equals a clean build.
- Deterministic identities are stable.
- Stale observations are invalidated.
- Cross-repository resolution fails closed.

The phase does not fully achieve the "scoped" part of the goal. A repository-scoped request still gets, and can write, another tracked repository's source bytes when one tracked root is nested inside another. I confirmed this end to end with a probe at HEAD (d1d1a688).

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Program-owned paragraphs, namespaced resources, and repeated relationship occurrences retain stable, non-colliding identities and distinguishable provenance | ✓ VERIFIED | `cobolIDSegment` in `cobol_grammar_identity.go`. `resourceNode` emits writer-independent shared resources (D-18). `cobol_placeholder` and `cobol_copy_library_key` are in `cobol_grammar_refs.go`. The extractor is `gortex-cobol-grammar/5`. The tests TestCobolGrammar_ProgramIDEncoding, _Occurrences, _SharedResource, and TestCobolLifecycle_CleanDeterminism (the shared-resource subtest) passed in the orchestrator's HEAD run of `./internal/parser/languages/` and in my `-race` run of the indexer. |
| 2 | Reindexing identical inputs reuses active deterministic identities and creates no duplicate active facts | ✓ VERIFIED | TestCobolLifecycle_IdenticalRerun passed under `-race`. It runs two reindexes and two reconciles, and each dump digest matches the clean build. TestCobolLifecycle_CleanDeterminism passed: it rebuilds in opposite track order and gets the same digest. |
| 3 | Edit, delete, restore, parser-version, extractor-version, retrieval, and material-configuration changes remove or invalidate stale owned observations | ✓ VERIFIED | These lifecycle tests passed under `-race`, and every step is compared with a clean build: Edit, Delete, Restore (byte-identical rows), VersionChange (Merkle and non-Merkle), ConfigChange, UnrelatedConfig, ConfigReloadRedecides, and HeadMove (D-17 re-stamp of `prov_vcs_commit`). The `_cobol_material` fingerprint is in `cobol_material.go`, and the Merkle salt is in `Indexer.merkleSaltFor`. "Retrieval" is covered by the D-17 VCS re-stamp. No retrieval manifest exists yet; this is recorded in the `cobol_material.go` comment and in 04-RESEARCH ID-08. |
| 4 | Incremental reconciliation and a clean rebuild produce the same canonical active deterministic subgraph for identical final inputs | ✓ VERIFIED | `requireCleanEquivalent` compares the BLAKE3 digest of the `graphdump.Dump` rows against `cleanCobolDump` and rejects any `restub_prev_` key. All 21 TestCobolLifecycle_* tests passed under `-race`, including CrossFile, CrossRepoIncremental, CrossRepoUntrack, Interrupted, OverCap, and ParallelWarmRestart. The volatile allowlist (`graphdump-volatile/3`) holds only seven `source_*_fingerprint` keys, each with a justification. Warnings WR-01 and WR-02 are contention and ordering risks that no deterministic sequence reaches. They are listed below. |
| 5 | Resolution and all downstream surfaces fail closed on missing scope, reject unsupported same-name cross-repository binding, and report zero unowned or misprefixed active nodes for the acceptance corpus | ✗ FAILED | Three parts hold. (a) Missing scope: an unbound multi-repo request is refused by every gated tool, proven by TestSourceScope_UnboundMultiRepoRefused and _TextSearchToolsGated. (b) Same-name binding: D-12 `pickCrossRepoCandidate` and the COBOL union-ambiguity rule are in place, proven by TestCrossRepoResolveAll_Ambiguous, TestCobolReconcile_Allowlist, and CrossRepoClean. (c) Ownership: TestCobolLifecycle_Ownership reports zero unowned and zero misprefixed nodes. One part fails: **"all downstream surfaces" do not enforce the same scope.** For a nested tracked checkout, read_file refuses, but get_symbol_source, search_text, and rename_symbol serve or write the other repository's bytes under an alpha scope (CR-01, reproduced). |

**Score:** 4/5 roadmap success criteria. 82/84 plan must-have truths are verified. The two that fail are 04-12 truths 1 and 2, in the nested-checkout case. No truth is present but behavior-unverified.

### Plan Must-Have Truths (summary by plan)

| Plan | Truths | Status | Notes |
|------|--------|--------|-------|
| 04-01 | 7 | ✓ all | graphdump: scoped snapshot only, BLAKE3, allowlist validation, FirstDiff; CleanDeterminism passes |
| 04-02 | 5 | ✓ all | Encoded program IDs, placeholder meta, occurrence provenance |
| 04-03 | 3 | ✓ all | `graph.NewScope`, `type ProjectionScope = Scope`; used by store_sqlite and neo4jprojection |
| 04-04 | 6 | ✓ all | `pickCrossRepoCandidate` wired into function and method fallbacks; `PersistEdgeAttributesBatch` stamps |
| 04-05 | 6 | ✓ all | Strict `Clean()`, enumerated exemptions, `currentSchemaVersion = 22`, index_health status |
| 04-06 | 6 | ✓ all | `ReconcileCobol` with one decision function; caller_unowned fails closed |
| 04-07 | 6 | ✓ all | `CobolConfig.Validate`, `SetCobolPolicyLookup(mi.cobolPolicyLookup())` (multi.go:393, 980), `SetCobolConfig(cfg.Index.Cobol)` (multi.go:2521) |
| 04-08 | 5 | ✓ all | COBOL restub to `cobol_placeholder` (incremental_batch.go:981, indexer.go), reconcile at the mutation tails |
| 04-09 | 5 | ✓ all | `_cobol_material` written through `repoExtractorVersionsSnapshot()` (index_state.go:41); hand-bumped cobol entry removed |
| 04-10 | 7 | ✓ all | Cross-surface test passes for the COBOL corpus. The full suite was reported green at a45d29ae; each package the fixes touched was re-run green at HEAD |
| 04-11 | 5 | ✓ all | Extractor `/5`, SharedResource, graphdump unit tests |
| 04-12 | 4 | ✗ 2 failed | Truth 1 ("refuses a repository outside the resolved scope") and truth 2 (D-20 write gate) fail for nested tracked checkouts: get_symbol_source and search_text leak, and rename_symbol writes. Truths 3 and 4 hold. |
| 04-13 | 3 | ✓ all | `indexEmbeddedRepo` goes through TrackRepoCtx (mcp.go:495, 567) |
| 04-14 | 7 | ✓ all | Union ambiguity, library ranking `cobol-copy-search/1`, alias edges, resource GC |
| 04-15 | 4 | ✓ all | Warm-restart reconcile (multi.go:3305) with deferral under deferResolve; D-08 Info log at affected_by.go:559 and incremental_batch.go:1511 |
| 04-16 | 5 | ✓ all | Merkle salt, `restampSourceRevisions` called from index_state.go:113 |

### Required Artifacts

`gsd-tools verify.artifacts` checked all 16 plans: every declared artifact exists and contains its required pattern, 52 of 52.

| Artifact (representative) | Status | Details |
|---------------------------|--------|---------|
| `internal/testutil/graphdump/graphdump.go` | ✓ VERIFIED | Reads only the scoped snapshot. Its allowlist is versioned and every key is justified. |
| `internal/graph/scope.go` | ✓ VERIFIED | NewScope, Allows, AllowsRepo |
| `internal/resolver/cobol_references.go`, `cobol_policy.go`, `cobol_resources.go` | ✓ VERIFIED | One reconcile pass for both the clean and incremental paths |
| `internal/indexer/cobol_reconcile.go`, `cobol_material.go`, `source_revision.go` | ✓ VERIFIED | Wired into the watcher tails, ReconcileRepoCtx, EndBatch, ResetBatch, RefreshRepoConfigs, and untrack |
| `internal/mcp/source_scope.go` | ⚠️ PARTIAL | The gate exists and is wired at 21 call sites. The physical-owner check is missing from resolveNodePath, the relative branch of resolveGraphPath, and the text-search attribution (gap 1). |
| `internal/graph/prefix_diagnostics.go` + `store_sqlite/prefix_diagnostics.go` | ✓ VERIFIED | Strict Clean and enumerated exemptions. The audit cannot detect double ownership (see gap 1). |

### Key Link Verification

`verify.key-links` passed for 27 links. The remaining 9 links name a symbol in `from:`, which the tool cannot parse, so I verified them by hand with grep:

| From | To | Status | Evidence |
|------|----|--------|----------|
| cross_repo.go resolveFunctionCall / resolveMethodCall | pickCrossRepoCandidate | ✓ WIRED | cross_repo.go:1383, 1635, 1679, 1705 |
| cross_repo.go ResolveAllContext | EdgeMetaBatchPersister | ✓ WIRED | cross_repo.go:625, 765 `PersistEdgeAttributesBatch` |
| multi.go newMasterResolver | SetCobolPolicyLookup | ✓ WIRED | multi.go:393, 980 |
| multi.go RefreshRepoConfigs | SetCobolConfig | ✓ WIRED | multi.go:2521 |
| config/manager.go readWorkspaceConfig | CobolConfig.Validate | ✓ WIRED | manager.go:212 |
| incremental_batch.go restubIncomingRefsFromView | cobol_placeholder | ✓ WIRED | incremental_batch.go:981 |
| index_state.go persistRepoIndexState | repoExtractorVersionsSnapshot | ✓ WIRED | index_state.go:41 |
| multi.go ReconcileRepoCtx | reconcileCobolRepos | ✓ WIRED | multi.go:3305 |
| index_state.go reconcileRepoIndexState | restampSourceRevisions | ✓ WIRED | index_state.go:113 |

### Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|----------|------|--------|-----------|--------|
| graphdump.Dump | node and edge rows | `OpenScopedProjectionSnapshot` → ReadNodePages / ReadEdgePages on a real SQLite store | Yes | ✓ FLOWING |
| index_health repo_ownership.status | Clean() | `graph.ReadPrefixDiagnostics` over the store | Yes | ✓ FLOWING (it does not detect a nested double owner) |
| COBOL policy | ResolveFrom / libraries / qualifiers | per-repo `.gortex.yaml` → `cobolPolicyFromConfig` → resolver lookup | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| D-04 sequences plus fingerprint and cap escalation | `GOWORK=off go test -race -count=1 -run 'TestCobolLifecycle_\|TestCobolMaterialFingerprint_\|TestAffectedBy_CapEscalates' ./internal/indexer/` | ok, 131.6s. 21 TestCobolLifecycle_* tests, 4 fingerprint tests, and CapEscalates all PASS | ✓ PASS |
| Source gate and text-search gating | `go test -race -run 'TestSourceScope_\|TestSearchText\|TestFindDeclaration\|TestIndexHealth' ./internal/mcp/` | ok, 24.6s | ✓ PASS |
| Scope, graphdump, config | `go test -race ./internal/graph/ ./internal/testutil/... ./internal/config/` | ok | ✓ PASS |
| v22 migration and prefix diagnostics | `go test -race -run 'Migration\|Prefix\|Scoped\|Unprefixed\|V22' ./internal/graph/store_sqlite/` | ok, 63.2s | ✓ PASS |
| D-04 name completeness | grep for the 13 required TestCobolLifecycle_ names | none missing | ✓ PASS |
| Nested-checkout scope (CR-01 probe, overlay only) | `go test -overlay … -run TestVerifierProbe_NestedCheckout ./internal/mcp/` | get_symbol_source and search_text `{repo:alpha}` returned beta's `ZebraWidget`. rename_symbol `{repo:alpha, dry_run:false}` rewrote beta/main.go. read_file was refused. PrefixDiagnostics Clean=true. | ✗ FAIL |

The orchestrator already observed these packages green with `-race` at HEAD: `./cmd/gortex`, `./internal/neo4jprojection/...`, `./internal/parser/languages/`, and `./internal/resolver/`.

### Probe Execution

Step 7c: no `scripts/*/tests/probe-*.sh` is declared or present for this phase. Not applicable.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| ID-03 | 04-02 | Paragraph identity includes its owning program | ✓ SATISFIED | cobolIDSegment, program-encoded paragraph IDs, ProgramIDEncoding test |
| ID-04 | 04-01, 04-07, 04-11, 04-14 | Resource identity carries its namespace | ✓ SATISFIED | D-18 shared resources, `_unqualified` kept distinct, alias edges only |
| ID-05 | 04-02 | Occurrences keep distinguishable provenance | ✓ SATISFIED | Occurrences test (`cobol_statement_symbols`, Line) |
| ID-06 | 04-01, 04-11 | Identical inputs give identical identities | ✓ SATISFIED | CleanDeterminism, IdenticalRerun |
| ID-07 | 04-06, 04-08, 04-14, 04-15 | Changed or deleted files remove stale observations | ✓ SATISFIED | Edit/Delete/Restore/Interrupted. WR-01 contention risk on untrack noted. |
| ID-08 | 04-09, 04-16 | Parser, extractor, retrieval, or config change invalidates | ✓ SATISFIED | VersionChange, ConfigChange, HeadMove. No retrieval manifest exists yet. |
| ID-09 | 04-07, 04-08, 04-09, 04-10, 04-15, 04-16 | Incremental equals clean | ✓ SATISFIED | All D-04 sequences equal the clean digest. WR-01/WR-02 are warnings. |
| SEC-01 | 04-03, 04-10, 04-12 | Same scope across extraction, resolution, query, and source surfaces | ✗ BLOCKED (partial) | graph.Scope is shared. In the nested-checkout case the surfaces disagree (gap 1). |
| SEC-02 | 04-12 | Ambiguous or absent scope fails closed for source reads | ⚠️ PARTIAL | Absent scope is refused on every gated tool. Bytes with two owners (nested root) are served under one owner's scope without failing closed (gap 1). |
| SEC-03 | 04-04, 04-06, 04-07, 04-10, 04-14 | No same-name cross-repo binding without an allowlist and evidence | ✓ SATISFIED | D-12 refusal, COBOL union ambiguity, Validate rejects wildcards and untracked repositories |
| SEC-08 | 04-05, 04-10, 04-13 | Zero unowned or misprefixed nodes for the acceptance corpus | ✓ SATISFIED (acceptance corpus) | TestCobolLifecycle_Ownership and TestEmbeddedIndex_OwnedByRepo. Caveat: the audit counts nested double-owned nodes as correctly prefixed (gap 1). |

Every phase requirement ID appears in at least one plan's `requirements:`. No requirement is orphaned.

### Prohibitions (all judgment-tier; non-authoritative LLM judgment)

| Plan | Prohibition | Verdict |
|------|-------------|---------|
| 04-10 | MUST NOT let two surfaces disagree about which nodes one allow-set admits | ✗ Violated in the nested-checkout case (gap 1) |
| 04-12 | MUST NOT return or write source bytes for a request with no resolved scope | ✓ Holds: unbound requests are refused, including search_text and find_declaration |
| 04-12 | MUST NOT reveal through a refusal whether something exists outside scope | ⚠️ Flagged: rename_symbol's distinct message (WR-04, accepted risk). Needs a human decision. |
| 04-01, 04-11 | MUST NOT exclude a Meta key without a justified versioned entry; shared resource must not depend on writer | ✓ Consistent |
| 04-03, 04-04, 04-06, 04-07, 04-14 (SEC-03/SEC-01) | No empty-prefix or empty-set widening; no map-order binding; no cross-repo bind without an allowlist | ✓ Consistent |
| 04-05, 04-13 (SEC-08) | No healthy report while unowned; no unprefixed embedded index | ✓ Consistent for the acceptance corpus; see the gap 1 caveat |
| 04-08, 04-14, 04-15 (ID-07/ID-09) | No stale COBOL binding; no deletion of real-file nodes in GC; no silent cap truncation | ✓ Consistent; see WR-01 |
| 04-09 (ID-08) | No serving of observations from a stale extractor or config | ✓ Consistent |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (phase-added lines in 47 non-test Go files) | | TBD, FIXME, XXX, TODO, HACK | none found | |
| internal/mcp/tools_fileops.go | 509, 598 | Lexical-only scope attribution at node and relative-path choke points | 🛑 Blocker | Gap 1 (CR-01) |
| internal/indexer/indexer.go | 5819 | shouldPruneDir does not stop at other tracked roots | 🛑 Blocker (root cause) | Double ownership (CR-01) |
| internal/indexer/repository_untrack.go | 209-275 | Best-effort `ReindexEdges` restub followed by a destructive `PurgeRepo` | ⚠️ Warning | WR-01, confirmed: store.go:1924 swallows `errReindexWriterGateContended`. Under contention the restub is dropped and the purge deletes other repositories' bound COBOL edges, which breaks ID-09 until re-extraction. Not deterministically reachable. |
| internal/indexer/multi.go | 1608, 1645, 2531 | `reconcileCobolRepos` runs outside `reach.BeginTopologyMutation` / `batchMutationGate` | ⚠️ Warning | WR-02, confirmed. The reach tiers can stay stale after a reload or ResetBatch rebind. A reload during parallel warmup decides early, and EndBatch's batch reconcile later corrects it. |
| internal/mcp/tools_find_declaration.go | 164-184 | Regex branch caps at `limit` before the scope filter | ⚠️ Warning | WR-03, confirmed. A bound caller can get "no use sites matched" when in-scope matches exist. This is a false negative, not a leak. |
| internal/mcp/tools_coding.go | 2864 | Distinct out-of-scope rename refusal | ⚠️ Warning | WR-04 (accepted risk), routed to human review |

### Human Verification Required

### 1. rename_symbol existence oracle (WR-04)

**Test:** Bind a request to alpha and run `rename_symbol` with `dry_run` on an alpha symbol that beta references.
**Expected:** Decide whether the message "some references lie outside the resolved scope" is acceptable under the 04-12 privacy prohibition, or replace it with the generic text.
**Why human:** This is a policy acceptance of a recorded risk. The code behaves as written.

### 2. Concurrency and ordering paths (REVIEW-FIX WR-01; re-review WR-01, WR-02)

**Test:** Review the untrack restub under writer-gate contention, a config reload during a parallel warmup, and reach-cache invalidation after a ResetBatch or reload reconcile.
**Expected:** Accept these as warnings or schedule fixes. Option A in the review (restub inside the PurgeRepo transaction) and running the reconcile under the topology gate are the suggested remedies.
**Why human:** Deterministic tests cannot reach these timing windows.

### Gaps Summary

There is one gap, and it has one root cause. It touches SC-5, SEC-01, SEC-02, and SEC-08 (caveat only).

Prior review finding CR-01 is only partly fixed. The fix commit 261ee576 decided that the physical owner of bytes under a nested tracked root is the inner repository, and it made `read_file`, `edit_file`, and `write_file` enforce that. Other surfaces disagree with that decision:
- `get_symbol_source`, `batch_symbols`, `get_editing_context`, and `smart_context` go through `resolveNodePath`.
- `rename_symbol` writes and scaffold go through the relative branch of `resolveGraphPath`.
- `search_text` and `find_declaration` attribute matches by the lexical path.

These surfaces still trust the outer repository's duplicate nodes. Those nodes exist because the outer repository's walk never stops at another tracked root. As a result, one allow-set admits different bytes on different surfaces, and a scoped write can modify a repository outside the scope. I reproduced both against HEAD. The ownership audit cannot see the double ownership.

The fix is small and localized. Either prune sibling tracked roots in `shouldPruneDir`, which is preferred because it also removes the double ownership, or call `requirePathSourceScope` at the three remaining choke points. Then extend `TestSourceScope_NestedCheckoutRefused` to cover those tools.

No later milestone phase covers nested tracked roots or source-gate attribution. Phase 7 reuses this gate rather than replacing it. So this gap is not deferred.

Everything else holds and is test-backed under `-race`: identity stability, invalidation, convergence of incremental with clean builds, cross-repository ambiguity refusal, and strict ownership for the acceptance corpus. WR-01 to WR-03 are confirmed warnings that do not by themselves defeat a success criterion on the tested sequences.

**If you decide nested tracked roots are out of scope for Phase 4**, you can accept the deviation by adding this override to the frontmatter, and the status would then be human_needed:

```yaml
overrides:
  - must_have: "SC-5 / SEC-01 / D-13 / D-20: every source surface enforces the same repository scope; a repository outside the resolved scope is refused for source reads and writes"
    reason: "Nested tracked roots are unsupported in Phase 4; tracked as a follow-up"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

I do not recommend this override, because the leak includes an on-disk write into the out-of-scope repository.

---

_Verified: 2026-10-13T16:45:00Z_
_Verifier: Claude (gsd-verifier)_
