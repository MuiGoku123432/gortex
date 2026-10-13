---
phase: 04-stable-incremental-and-cross-repository-lifecycle
verified: 2026-10-13T19:39:56Z
status: gaps_found
score: 93/97 must-haves verified (3/5 roadmap success criteria; 90/92 plan truths)
covered_files: [".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-17-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-17-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-18-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-18-SUMMARY.md","internal/config/global.go","internal/config/manager.go","internal/indexer/indexer.go","internal/indexer/multi.go","internal/indexer/nested_tracked_root_test.go","internal/indexer/walk_source.go","internal/indexer/watcher.go","internal/indexer/workspace_resolve.go","internal/mcp/explore_artifact_intent.go","internal/mcp/source_scope.go","internal/mcp/source_scope_test.go","internal/mcp/tools_coding.go","internal/mcp/tools_explore.go","internal/mcp/tools_fileops.go","internal/mcp/tools_find_declaration.go","internal/mcp/tools_search_text.go","internal/mcp/tools_search_text_pathsep_test.go","internal/mcp/tools_search_text_test.go"]
covered_digest: "v3:sha256:5d7af645ff584d7aba14c00d59d35541600a5e8da5f35435a3f9f977601dcff3"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 86/89
  gaps_closed:
    - "04-12 truth 2 (D-20 write gate, nested case): an alpha-scoped rename_symbol of a nested beta symbol is refused and beta's main.go is unchanged on disk"
    - "Nested-checkout source gate on read_file, get_symbol_source, batch_symbols, search_text, find_declaration, rename_symbol in both index states (tracked_together, outer_indexed_first)"
    - "Nested tracked root is indexed once when the top-level repos: config lists both repositories (no alpha/nested nodes; audit clean)"
  gaps_remaining:
    - "SC-5 / SEC-01 / SEC-02: every source surface enforces the same scope and refuses unbound multi-repo requests (explore artifact lane leaks source lines)"
  regressions:
    - "SC-4 / ID-09 in the nested-repository topology: after 04-18, untracking a nested repository or tracking one later leaves an incremental graph that differs from a clean build of the same final inputs until the outer repository is rebuilt or the daemon restarts"
gaps:
  - truth: "SC-5 / SEC-01 / SEC-02 / D-13 / D-16: every tool that returns source bytes refuses a request with no resolved scope, and every source surface admits the same bytes for one allow-set (04-12 truth 1; 04-17 truth 3 'no two source surfaces disagree'; 04-10 prohibition; carried forward from the prior gap)"
    status: failed
    reason: "The explore tool's artifact lane (gatherExploreArtifactLane) returns matched source lines as target snippets with no source gate. Verifier probe (go test -overlay, tree untouched): (1) Unbound two-repository server (the same fixture shape as TestSourceScope_UnboundMultiRepoRefused): explore {task: 'Set ZEBRA_SECRET_TOKEN in deploy.yaml for the deployment pipeline config'} returned isError=false with 'beta/deploy.yaml ... ZEBRA_SECRET_TOKEN: hunter2zebraxyz'. search_text for the same token was refused. (2) Nested outer_indexed_first, explore {repo: alpha}: returned 'alpha/nested/deploy.yaml ... ZEBRA_SECRET_TOKEN: hunter2zebraxyz' while read_file and search_text {repo: alpha} refused or filtered it. Explore's main lane is gated (manifestSymbolSource goes through resolveNodePath); only the artifact lane bypasses the gate. This is review WR-04, confirmed, and the unbound leak is wider than the review stated."
    artifacts:
      - path: "internal/mcp/explore_artifact_intent.go"
        issue: "gatherExploreArtifactLane (lines ~386-431): GrepText / GrepTextForRepos, then enrichTextMatchesContext (lexical node scope only), then hit.snippet = truncateExploreArtifactSnippet(match.Text). No requireSourceScope or requirePathSourceScope call. When scope.RepoAllow is nil it fans out across every repository."
      - path: "internal/mcp/tools_explore.go"
        issue: "handleExplore (~2730-2757) resolves scope with IntentLocate and calls the artifact lane without the unbound refusal that search_text and find_declaration apply (requireSourceScope(ctx, \"\"))"
      - path: "internal/mcp/source_scope_test.go"
        issue: "TestSourceScope_UnboundMultiRepoRefused and TestSourceScope_NestedCheckoutRefused do not include explore"
    missing:
      - "Before the artifact lane runs, refuse an unbound multi-repo request or drop its snippets (requireSourceScope(ctx, \"\") returns errSourceScopeRequired), as search_text does"
      - "Pass the lane's matches through filterTextMatchesByResolvedScope (physical-owner gate) before taking snippets, or gate each kept hit with requirePathSourceScope"
      - "Add explore (artifact-intent task plus a yaml fixture) to TestSourceScope_UnboundMultiRepoRefused and to both TestSourceScope_NestedCheckoutRefused states"
  - truth: "SC-4 / ID-09 / SEC-08 (nested topology): incremental reconciliation equals a clean build for identical final inputs, and a file under a nested tracked root has exactly one owner (04-18 prohibition 'MUST NOT let one repository's walk or watcher index a file under another tracked repository's nested root')"
    status: partial
    reason: "04-18 seeds Indexer.trackedRepoRoots once at construction from GlobalConfig.ReposSnapshot() (top-level repos: only). It never refreshes the seed, and the seed does not match the set being indexed. Verifier probes (go test -overlay): (a) Untrack: index alpha and nested beta, then UntrackRepo('beta'). The same alpha Indexer is kept. shouldExclude(beta/main.go)=true, and IncrementalReindexRepo('alpha', [beta/main.go]) leaves alpha/nested absent. A clean alpha-only build has alpha/nested present, so incremental differs from clean. Only IndexRepo('alpha') or a restart recovers. Before 04-18, incremental and clean agreed here; 04-18 introduced this regression (review WR-01). (b) Track later: alpha's alpha/nested duplicates stay after TrackRepoCtx(beta), and alpha's watcher now refuses their update and delete events (review WR-02). A clean build of {alpha, beta} has none. An explicit IncrementalReindexRepo or the warm-restart census evicts them, but the live daemon path does not. (c) projects:-defined pair with active_project: IndexScoped indexes both, and alpha/nested is present, so one file has two owners (review CR-01 under-prune). (d) An inactive repos: entry: neither repository indexes alpha/nested, while RepoForFile says alpha (CR-01 over-prune). (c) and (d) go through IndexScoped, which has no non-test caller in this repo; the daemon warms Global().Repos, which matches the seed. (a) and (b) are reachable from the daemon."
    artifacts:
      - path: "internal/indexer/multi.go"
        issue: "newPerRepoIndexerGuardedWithMode (lines 376-392) seeds trackedRepoRoots from ReposSnapshot() instead of the batch's resolved repos; the seed is never refreshed on untrack or config reload"
      - path: "internal/indexer/indexer.go"
        issue: "inNestedTrackedRoot (5697-5711) consults the stale seed before live mi.repos, so an untracked nested root keeps excluding files"
      - path: "internal/indexer/nested_tracked_root_test.go"
        issue: "Covers only the top-level repos: shape. No untrack, projects:, inactive-entry, or track-later-without-manual-IndexRepo test."
    missing:
      - "Seed nested-root pruning from the list indexMultiRepo or TrackRepoCtx is actually indexing, and use live mi.repos after publication (drop the per-Indexer config copy, or refresh it on untrack and reload)"
      - "On untrack of a nested root, reindex (or admit) the enclosing repository's subtree. On track of a nested root, evict the enclosing repository's file nodes under it."
      - "Tests: untrack beta, then alpha's incremental pass equals a clean alpha-only dump; track beta later, then alpha/nested is evicted without a manual IndexRepo; projects:-map and inactive-entry IndexScoped shapes"
human_verification:
  - test: "Decide whether the rename_symbol out-of-scope refusal message is acceptable (prior WR-04 / re-review IN-06, accepted risk in 04-12)"
    expected: "Either accept the risk formally (override the 04-12 privacy prohibition) or replace the message with the generic not-found text"
    why_human: "Policy decision. A scoped dry_run rename returns 'some references lie outside the resolved scope', which reveals that out-of-scope references exist. That message now also covers nested physical owners."
  - test: "Review the concurrency and ordering warnings: REVIEW-FIX WR-01 (cobolReconcileMu, deferred reconcile in EndBatch/ResetBatch), prior re-review WR-01 (untrack restub dropped under writer-gate contention) and WR-02 (reconcile sites outside the reach topology gate), and new IN-05 (hidden mi.mu.RLock in shouldExclude, not reentrant once a writer is queued)"
    expected: "Accept them as warnings or schedule fixes"
    why_human: "Contention and timing windows that deterministic tests do not reach"
---

# Phase 4: Stable Incremental and Cross-Repository Lifecycle Verification Report

**Phase Goal:** Users receive the same canonical scoped deterministic graph from identical final inputs regardless of reruns, edits, deletions, restores, or clean rebuilds.
**Verified:** 2026-10-13T19:39:56Z (HEAD e6d94a98)
**Status:** gaps_found
**Re-verification:** Yes. This follows gap closure by plans 04-17 and 04-18.

## Summary of this pass

- **04-17 (source gate) is achieved for the surfaces it names.** read_file, get_symbol_source, batch_symbols, search_text, find_declaration, and rename_symbol now gate on the physical owner in both index states. The not-found text matches. I re-ran the tests under `-race` and they are green.
- **One source surface was never gated: explore's artifact lane.** It serves source lines:
  - to an **unbound** multi-repository request, which violates SEC-02 independent of nesting;
  - to an alpha-scoped request in the nested `outer_indexed_first` state.
  
  So the carried-forward truth, "every source surface enforces the same scope", still fails. The prior verification missed explore because its evidence list (TestSourceScope_UnboundMultiRepoRefused) does not include it.
- **04-18 (walk pruning) holds only for the config shape its tests use.** The construction-time seed is stale and does not match the indexed set. I reproduced review CR-01, WR-01, and WR-02. Untracking or later tracking a nested repository now leaves an incremental graph that differs from a clean build until a rebuild or restart. 04-18 introduced this, so it is a regression of SC-4 in the nested topology.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Stable, non-colliding identities and provenance | ✓ VERIFIED (regression check) | No identity code was touched since the prior pass. The orchestrator's full `-race` suite passes (indexer, parser/languages, resolver). |
| 2 | Identical reindex reuses identities, no duplicates | ✓ VERIFIED (regression check) | Same as row 1. TestCobolLifecycle_IdenticalRerun and CleanDeterminism are in the green indexer run. |
| 3 | Edit/delete/restore/version/retrieval/config changes invalidate stale observations | ✓ VERIFIED (regression check) | Same as row 1. |
| 4 | Incremental equals clean for identical final inputs | ✗ PARTIAL (gap 2) | Holds for every D-04 COBOL acceptance sequence (all TestCobolLifecycle_* tests green). It fails for the nested-repository topology. Probe: track alpha and nested beta, then untrack beta. Alpha's incremental pass leaves `alpha/nested/main.go` absent, while a clean alpha-only build has it. Tracking beta later leaves frozen alpha/nested duplicates that a clean {alpha, beta} build lacks. 04-18 introduced this. |
| 5 | All downstream surfaces fail closed on missing scope; no same-name cross-repo binding; zero unowned/misprefixed for the acceptance corpus | ✗ FAILED (gap 1) | Binding refusal and the acceptance-corpus ownership audit still hold. The fail-closed part does not: explore returns `beta/deploy.yaml` lines to an **unbound** two-repository request (search_text refuses the same request), and returns nested beta lines to an alpha-scoped request. |

**Score:** 93/97 must-haves. Roadmap criteria: 3/5. Plan truths: 90/92. The failing plan truths are 04-12 truth 1 and 04-17 truth 3. No truth is present but behavior-unverified.

### Plan Must-Have Truths (gap-closure plans in full; others by regression)

| Plan | Truth | Status | Evidence |
|------|-------|--------|----------|
| 04-17 #1 | Alpha scope gets none of beta's bytes from read_file, get_symbol_source, batch_symbols, search_text, find_declaration, in both states; not-found parity | ✓ VERIFIED | `TestSourceScope_NestedCheckoutRefused` PASS under -race (verifier run). Code: tools_fileops.go:519/545 (resolveNodePath), tools_coding.go:1027 (not-found mapping), tools_search_text.go:278, tools_find_declaration.go:163 |
| 04-17 #2 | Alpha-scoped non-dry-run rename of beta's file refused, bytes unchanged | ✓ VERIFIED | rename_symbol subtest PASS (both states) |
| 04-17 #3 | The listed choke points admit only through requirePathSourceScope, "so no two source surfaces disagree for one allow-set" | ✗ FAILED | The listed sites are wired (tools_fileops.go:83, 519, 545, 614, 642; search_text:278; find_declaration:163). Explore's artifact lane disagrees: probe (2) above. |
| 04-17 #4 | No over-refusal; single-repo short-circuit | ✓ VERIFIED | in_scope_reads PASS; source_scope.go:111 `!IsMultiRepo()`; all 8 TestSourceScope_* PASS |
| 04-18 #1 | Both in top-level config: nested indexed once, audit clean, either order | ✓ VERIFIED | `TestNestedTrackedRoot_IndexedOnce/{alpha_first,beta_first}` PASS (verifier run). Holds only for that shape (see gap 2). |
| 04-18 #2 | Outer reindex after later track drops duplicates | ✓ VERIFIED | `TestNestedTrackedRoot_OuterReindexDropsDuplicates` PASS. Needs a manual `IndexRepo("alpha")`; no production track path calls it (gap 2b). |
| 04-18 #3 | Outer watcher refuses nested paths; beta's watcher admits | ✓ VERIFIED | `TestNestedTrackedRoot_WatcherRefusesNestedEvents` PASS; watcher.go:2953 |
| 04-18 #4 | Ancestor root never excludes nested repo's own files; package green | ✓ VERIFIED | beta `shouldExclude` false in IndexedOnce; the orchestrator's `-timeout 45m` indexer run is ok |
| 04-12 #1 | Every source-returning tool refuses an unbound multi-repo request and an out-of-scope repository | ✗ FAILED | explore (artifact lane) answers an unbound request with source bytes (probe 1). All other gated tools refuse. |
| 04-12 #2 | D-20 write gate | ✓ VERIFIED (gap closed) | Nested rename refused, beta unchanged (04-17 #2) |
| 04-01..04-16 (other truths) | as in the prior report | ✓ VERIFIED (regression) | Files in those plans were not touched by 04-17/04-18 beyond the listed sites. The full suite is green per the orchestrator. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/mcp/source_scope.go` | `IsMultiRepo()` short-circuit | ✓ VERIFIED | `verify.artifacts` 04-17: 5/5 pass |
| `internal/mcp/tools_fileops.go` | physical-owner gate in resolveNodePath and the relative resolveGraphPath branch | ✓ VERIFIED | Lines 519, 545, 642 |
| `internal/mcp/tools_search_text.go`, `tools_find_declaration.go` | per-path requirePathSourceScope | ✓ VERIFIED | Lines 278, 163 |
| `internal/mcp/source_scope_test.go` | outer_indexed_first and tracked_together subtests | ✓ VERIFIED | Present and passing. Explore is missing from the inventory (gap 1). |
| `internal/indexer/indexer.go` | trackedRepoRoots, inNestedTrackedRoot, call in shouldExclude | ✓ VERIFIED (wiring) / ⚠️ stale data | `verify.artifacts` 04-18: 5/5 pass. The data source is wrong (gap 2). |
| `internal/indexer/multi.go` | seed from ReposSnapshot | ⚠️ HOLLOW for non-top-level shapes | Seed is `gc.Repos` only, never refreshed |
| `internal/indexer/workspace_resolve.go` | nestedTrackedRootContains | ✓ VERIFIED | Lines 152-168 |
| `internal/indexer/watcher.go`, `walk_source.go` | nested check on the live path and the fast path | ✓ VERIFIED | watcher.go:2953, walk_source.go:78 |
| `internal/mcp/explore_artifact_intent.go` | (not in a plan; it is a source surface) | ✗ UNGATED | gap 1 |

### Key Link Verification

| From | To | Via | Status |
|------|----|-----|--------|
| 04-17: tools_fileops / search_text / find_declaration / tools_coding | source_scope.go | requirePathSourceScope, ResolveFilePath, errSourceOutOfScope | ✓ WIRED (`verify.key-links` 4/4) |
| 04-18: shouldExclude, Watcher.isExcluded, inNestedTrackedRoot, constructor | inNestedTrackedRoot, nestedTrackedRootContains, ReposSnapshot | | ✓ WIRED (`verify.key-links` 4/4) |
| explore artifact lane | source_scope.go | (none) | ✗ NOT_WIRED (gap 1) |

### Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|----------|------|--------|-----------|--------|
| `Indexer.trackedRepoRoots` | nested roots that prune the walk | `GlobalConfig.ReposSnapshot()` at construction | Top-level `repos:` only; not the indexed set; never refreshed | ⚠️ STATIC (gap 2) |
| `nestedTrackedRootContains` | registered roots | live `mi.repos` under RLock | Yes. During parallel warmup it is empty until publication. | ✓ FLOWING (post-warmup) |
| explore artifact snippet | `match.Text` | `GrepText` across all repos when RepoAllow is nil | Real bytes, ungated | ✗ leaks (gap 1) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| 04-17 source gate (all TestSourceScope_) | `GOWORK=off go test -race -count=1 -run '^TestSourceScope_' ./internal/mcp/` | ok 11.5s; 8/8 PASS incl. NestedCheckoutRefused | ✓ PASS |
| 04-18 nested pruning | `GOWORK=off go test -race -count=1 -run '^TestNestedTrackedRoot_' ./internal/indexer/` | ok 6.2s; IndexedOnce (2), OuterReindexDropsDuplicates, WatcherRefusesNestedEvents PASS | ✓ PASS |
| Full suite | (orchestrator) `go test -race ./...` with `-timeout 45m` for indexer and store_sqlite | all ok | ✓ PASS |
| gofmt / debt markers on the 04-17/04-18 diff | `gofmt -l` on 12 files; grep TBD/FIXME/XXX/TODO/HACK on added lines | clean / none | ✓ PASS |
| Explore unbound and nested leak (probe) | `go test -overlay … -run TestVerifierProbe_ExploreArtifactLane ./internal/mcp/` | unbound: isError=false, leak=true (`beta/deploy.yaml` line); nested alpha-scoped: leak=true; sibling alpha-scoped: no leak; search_text unbound: refused | ✗ FAIL |
| Nested seed probes | `go test -overlay … -run TestVerifierProbe_ ./internal/indexer/` | projects: map: repos=[alpha beta], alpha/nested present=true. Inactive entry: repos=[alpha], alpha/nested=false, beta=false, RepoForFile=alpha. Untrack: same Indexer, shouldExclude=true, incremental leaves nested absent vs clean present; IndexRepo recovers. Track-later: duplicate present until explicit IncrementalReindexRepo evicts it. Symlinked beta root: beta file nodes=0, alpha nested=2, RepoForFile=alpha (single owner). | ✗ FAIL (a-d), ℹ️ symlink |

The probe sources are outside the tree and were run only through `-overlay`, so the repository was not modified:
- `/private/tmp/claude-501/-Users-e1001547-mbp-it-repos-mine-GoApps/6332945b-c602-4ced-bd7d-35c7715acfa1/scratchpad/probe/verifier_probe_explore_test.go`
- `/private/tmp/claude-501/-Users-e1001547-mbp-it-repos-mine-GoApps/6332945b-c602-4ced-bd7d-35c7715acfa1/scratchpad/probe/verifier_probe_nested_test.go`

### Probe Execution

Step 7c: no `scripts/*/tests/probe-*.sh` is declared or present. Not applicable.

### Review claim adjudication (04-REVIEW.md, re-review of 04-17/04-18)

| Finding | Verifier result | Classification |
|---------|-----------------|----------------|
| CR-01: seed from the wrong list (under-prune and over-prune) | **Confirmed** by probe. Both shapes go only through `IndexScoped`/`IndexAll`, which have no non-test caller. The daemon warms `Global().Repos` (daemon_state.go:460), which equals the seed. | Part of gap 2 (library path; real defect, lower production reach) |
| WR-01: seed never refreshes after untrack | **Confirmed**. Incremental differs from clean for identical final inputs. The only write to `trackedRepoRoots` is multi.go:390. | Gap 2 (production-reachable SC-4 regression) |
| WR-02: track-later duplicates frozen | **Confirmed**. Duplicates persist; an explicit incremental or the warm census evicts them; the watcher refuses them. | Gap 2 |
| WR-03: symlinked nested root not pruned | **Partly refuted.** Beta tracked through a symlink indexes **zero** files (pre-existing symlink confinement), so alpha is the only owner and RepoForFile agrees. No double ownership. | ℹ️ Info |
| WR-04: explore artifact lane ungated | **Confirmed, and wider than stated.** It also leaks to unbound multi-repo requests with no nesting at all. | Gap 1 (BLOCKER) |
| WR-05: find_declaration regex caps before filtering | Not re-probed. Code unchanged (tools_find_declaration.go:169-185). False negative, not a leak. | ⚠️ Warning |
| IN-01..IN-06 | Noted. IN-05 is added to the concurrency human item. | ℹ️ Info |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| ID-03 | 04-02 | Paragraph identity includes owning program | ✓ SATISFIED | Unchanged; suite green |
| ID-04 | 04-01, 04-07, 04-11, 04-14 | Resource identity namespace | ✓ SATISFIED | Unchanged |
| ID-05 | 04-02 | Distinguishable occurrence provenance | ✓ SATISFIED | Unchanged |
| ID-06 | 04-01, 04-11 | Identical inputs give identical identities | ✓ SATISFIED | Unchanged |
| ID-07 | 04-06, 04-08, 04-14, 04-15 | Stale observations removed on change or delete | ✓ SATISFIED | Unchanged |
| ID-08 | 04-09, 04-16 | Parser, extractor, retrieval, or config change invalidates | ✓ SATISFIED | Unchanged |
| ID-09 | 04-07..04-10, 04-15, 04-16 | Incremental equals clean | ⚠️ PARTIAL | All acceptance sequences equal clean. The nested-repository untrack and track-later sequences diverge after 04-18 (gap 2). |
| SEC-01 | 04-03, 04-10, 04-12, 04-17, 04-18 | Same scope across surfaces | ✗ BLOCKED (partial) | Explore disagrees with the other surfaces in the nested state (gap 1). |
| SEC-02 | 04-12, 04-17 | Absent scope fails closed for source reads and AI context | ✗ BLOCKED | Explore serves source lines to an unbound multi-repo request (gap 1). |
| SEC-03 | 04-04, 04-06, 04-07, 04-10, 04-14 | No same-name cross-repo binding without allowlist and evidence | ✓ SATISFIED | Unchanged |
| SEC-08 | 04-05, 04-10, 04-13, 04-18 | Zero unowned/misprefixed for the acceptance corpus | ✓ SATISFIED (acceptance corpus) | TestCobolLifecycle_Ownership green. Caveat: nested double ownership persists in the projects:/track-later shapes and the audit cannot see it (gap 2). |

Every phase requirement ID appears in at least one plan's `requirements:`, so none is orphaned. **REQUIREMENTS.md currently marks SEC-01, SEC-02, and SEC-08 `[x]` / Complete.** With the gaps above, SEC-01 and SEC-02 are premature and should revert to Gaps Found.

### Prohibitions (judgment-tier; non-authoritative LLM verdicts)

| Plan | Prohibition | Verdict |
|------|-------------|---------|
| 04-17 SEC-01 | MUST NOT admit a source path on any source surface by its lexical prefix alone | ✗ Violated by the explore artifact lane (lexical node scope only) |
| 04-17 SEC-02 | MUST NOT reveal via a get_symbol_source refusal that an out-of-scope repo owns the file | ✓ Holds (tools_coding.go:1027 maps to "symbol not found") |
| 04-18 SEC-08 | MUST NOT let one repo's walk or watcher index a file under another tracked nested root | ✗ Violated in the projects:-map IndexScoped shape (probe c) |
| 04-18 SEC-01 | MUST NOT read the global config per file or event | ✓ Holds (seed copied once; the per-file check reads `mi.repos`, not the config) |
| 04-12 SEC-02 | MUST NOT return source bytes for a request with no resolved scope (multi-repo) | ✗ Violated by explore (probe 1) |
| 04-12 SEC-02 privacy | MUST NOT reveal out-of-scope existence through a refusal | ⚠️ Flagged (rename_symbol message; human item 1) |
| Others (04-01..04-16) | as in the prior report | ✓ Consistent |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/mcp/explore_artifact_intent.go | ~386-431 | Source snippet emitted with no source gate | 🛑 Blocker | Gap 1. In contract (carried-forward gap truth) and backed by a reproducible probe. |
| internal/indexer/multi.go | 376-392 | Construction-time config snapshot used as authoritative ownership data | 🛑 Blocker | Gap 2. File modified since the prior verification (regression) and backed by probes. |
| internal/mcp/tools_find_declaration.go | 169-185 | Regex branch caps at `limit` before the scope filter | ⚠️ Warning | WR-05 false negative |
| internal/indexer/workspace_resolve.go | 152-154 | Hidden `mi.mu.RLock` on every walked entry | ℹ️ Info | IN-05. The lock-order audit in 04-18-SUMMARY found no caller holding mi.mu. |
| (04-17/04-18 added lines) | | TBD/FIXME/XXX/TODO/HACK | none | |

### Human Verification Required

### 1. rename_symbol existence oracle (prior WR-04 / IN-06)

**Test:** Bind a request to alpha, then run `rename_symbol` with `dry_run` on an alpha symbol that beta, or a nested beta checkout, references.
**Expected:** Accept the distinct "some references lie outside the resolved scope" message under the 04-12 privacy prohibition, or replace it with the generic text.
**Why human:** Policy acceptance of a recorded risk.

### 2. Concurrency and ordering warnings

**Test:** Review these together:
- REVIEW-FIX WR-01: cobolReconcileMu and the deferred reconcile.
- Prior re-review WR-01: the untrack restub is dropped under writer-gate contention.
- Prior re-review WR-02: reconcile sites run outside the reach topology gate.
- New IN-05: a non-reentrant RLock on the walk path.

**Expected:** Accept them as warnings or schedule fixes.
**Why human:** Deterministic tests cannot reach these timing windows.

### Gaps Summary

There are two gaps.

**Gap 1 (BLOCKER: SC-5, SEC-02, SEC-01). Explore's artifact lane is a source surface with no source gate.** It is a pre-existing upstream feature that the 04-12 inventory and test list never included. That is why both the prior verification and 04-17 missed it.

The probe shows two leaks:
- An **unbound** two-repository `explore` returns another repository's config-file line, for example a token in `deploy.yaml`, while `search_text` refuses the same request.
- In the nested `outer_indexed_first` state, an alpha-scoped `explore` returns beta's bytes.

The fix is small and local, and the review's WR-04 sketch fits it:
- apply the `requireSourceScope(ctx, "")` unbound refusal before the lane runs;
- route the lane's matches through `filterTextMatchesByResolvedScope` before taking snippets;
- add explore to both TestSourceScope_ inventories.

**Gap 2 (SC-4/ID-09 regression and the SEC-08 caveat in the nested topology).** 04-18 prunes nested roots using a config snapshot taken when each Indexer is built. That snapshot:
- names only top-level `repos:`;
- is never refreshed;
- is consulted before the live registered roots.

As a result, untracking a nested repository silently drops those files from the outer repository's graph. A clean build of the same final inputs includes them. Tracking a nested repository later freezes the outer repository's duplicates until a restart or a manual reindex.

Both cases converge on a daemon restart or a full `IndexRepo`, and the 04-17 gate keeps the read surfaces safe while they persist. Explore is the exception (gap 1). So the impact is limited to graph-metadata tools and incremental-equals-clean equivalence. 04-18 deferred these two cases in-plan ("fails closed"). No later roadmap phase covers them: Phase 6 SC-4 lists edit, delete, restore, parser-version, and interrupted-retry sequences, not repository topology changes. Under the deferral rules they therefore stay gaps.

**If you decide that nested-repository topology changes (track or untrack of a repository nested inside another) only need to converge on restart in Phase 4**, you can accept gap 2 with this override and leave gap 1 as the only blocker:

```yaml
overrides:
  - must_have: "SC-4 / ID-09 / SEC-08 (nested topology): incremental reconciliation equals a clean build for identical final inputs, and a file under a nested tracked root has exactly one owner"
    reason: "Nested-repository track/untrack converges on the next full reindex or daemon restart; the 04-17 source gate keeps read surfaces consistent meanwhile; library-only IndexScoped shapes (projects:, inactive entries) are not used by the daemon"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

I do not suggest an override for gap 1. It returns source bytes to a request with no scope, which is exactly what SEC-02 forbids.

Everything else holds and is test-backed. The 04-17 gate on the six named surfaces is correct in both index states. The rename write leak from the prior report is closed. The lifecycle core (SC-1 to SC-3, and SC-4 on the acceptance sequences) is unchanged and green.

---

_Verified: 2026-10-13T19:39:56Z_
_Verifier: Claude (gsd-verifier)_
