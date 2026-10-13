---
phase: 04-stable-incremental-and-cross-repository-lifecycle
verified: 2026-10-13T22:56:01Z
status: gaps_found
score: 114/116 must-haves verified (4/5 roadmap success criteria; 110/111 plan truths)
covered_files: [".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-17-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-17-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-18-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-18-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-19-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-19-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-20-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-20-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-21-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-21-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-22-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-22-SUMMARY.md","cmd/gortex/daemon_state.go","internal/indexer/cobol_lifecycle_crossrepo_test.go","internal/indexer/indexer.go","internal/indexer/multi.go","internal/indexer/nested_tracked_root_test.go","internal/indexer/repository_untrack.go","internal/indexer/workspace_resolve.go","internal/mcp/explore_artifact_intent.go","internal/mcp/overlay_view.go","internal/mcp/source_scope.go","internal/mcp/source_scope_test.go","internal/mcp/tools_explore.go","internal/mcp/tools_fileops.go","internal/mcp/tools_find_declaration.go","internal/mcp/tools_find_declaration_test.go","internal/mcp/tools_generate_skill.go","internal/mcp/tools_generate_skill_test.go","internal/mcp/tools_move_inline.go","internal/mcp/tools_move_inline_test.go","internal/mcp/tools_simulate.go"]
covered_digest: "v3:sha256:949b8aa149c3aa2231d2766d00200d071f022e93314175fee4f5fe63bf2f56e3"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 93/97
  gaps_closed:
    - "SC-5 / SEC-01 / SEC-02 (explore artifact lane): an unbound two-repository explore is refused before the lane runs, and an alpha-scoped explore returns no nested beta bytes in either index state (04-19)"
    - "SC-4 / ID-09 / SEC-08 (nested topology): untracking a nested repository admits its files into the enclosing repository, tracking it later evicts the outer copies in the same call, and the projects:-map and inactive-entry shapes give one owner (04-20, 04-21)"
  gaps_remaining: []
  regressions: []
  new_gaps:
    - "SC-5 / SEC-01: the per-entry walk gate added in 04-22 fails open for a symlink whose target lies outside every tracked root, so generate_skill, move_symbol and inline_symbol admit bytes that read_file refuses for the same path and scope (review CR-04, verifier probe-confirmed)"
gaps:
  - truth: "SC-5 / SEC-01 / 04-17 truth 3: every source surface admits the same bytes for one allow-set and fails closed when the physical owner of a location cannot be established (no source surface admits a location that read_file refuses for the same scope)"
    status: failed
    reason: "requirePathSourceScope returns nil when the symlink-resolved location has no registered owner (source_scope.go:124, `physical == \"\"`). Its doc comment says it relies on the caller having confined the path first ('Root confinement accepts a location inside any tracked root'). resolveFilePath and resolveNodePath do confine (guardSymlinkWithinRepo). The three filesystem walks that 04-22 gated do not, so for them the gate fails open on an owner-less location. Verifier probe (go test -overlay, tree untouched), fixture newNestedGoModuleServer with alpha/a/link.txt -> <tmp>/secret.txt and alpha/a/zlink.go -> <tmp>/x.go: (1) read_file {path: alpha/a/link.txt, repo: alpha} is refused ('resolves to ... outside every indexed repository root'). (2) generate_skill {directory: alpha/, repo: alpha} lists a/link.txt (25 bytes) and a/zlink.go, and writes alpha/.claude/skills/alphaskill/references/a/link.txt containing TOP-SECRET-OUTSIDE-ROOTS. (3) read_file then serves that bundled copy (isError=false, secret present). (4) move_symbol alpha/a/a.go::Foo rewrites alpha/a/zlink.go: the symlink becomes a regular in-repository file holding the outside file's bytes (with the import rewritten); the outside file is unchanged. The same leak appears with sibling (non-nested) repositories (newTwoRepoServer: generate_skill dry_run lists alpha/link.txt). The behavior in generate_skill predates the phase (upstream 47fceda9), but 04-22 rewrote these walks specifically to close 'a symlinked file' and its comments claim that coverage; the files changed in this round."
    artifacts:
      - path: "internal/mcp/source_scope.go"
        issue: "requirePathSourceScope (lines 107-128): `physical == \"\"` returns nil, so an owner-less (outside-every-root) location is attributed to the lexical repository instead of being refused"
      - path: "internal/mcp/tools_generate_skill.go"
        issue: "WalkDir callback (lines ~158-205): per-entry gate is requirePathSourceScope only; no guardSymlinkWithinRepo; os.ReadFile follows the symlink and the bytes are written into the skill bundle"
      - path: "internal/mcp/tools_move_inline.go"
        issue: "move_symbol and inline_symbol module walks (lines ~254-283, ~1148-1172) and the callerFiles read loops: same fail-open; AtomicWriteFile replaces the link with an in-repository copy of the out-of-tree file"
      - path: "internal/mcp/tools_generate_skill_test.go"
        issue: "TestGenerateSkill_NestedTrackedRootExcluded has no symlink case; tools_move_inline_test.go likewise"
    missing:
      - "Refuse an owner-less location in the walk gates: either make requirePathSourceScope refuse when physical == \"\" and guardResolvedPathWithinRepo(ctx, absPath, real) fails (every caller inherits confinement), or call guardSymlinkWithinRepo on each walk entry before reading it (generate_skill, move_symbol, inline_symbol, and the callerFiles read loops)"
      - "Make the fix hold in single-repository mode too (requirePathSourceScope short-circuits there, so the confinement check must not depend on IsMultiRepo)"
      - "Tests: a symlink-out-of-every-root case for generate_skill (not listed, not copied), move_symbol and inline_symbol (link not rewritten or replaced), plus a single-repository variant"
advisory:
  - finding: "WR-10: preview_edit and simulate_chain resolve WorkspaceEdit paths through ungated resolveOverlayAbsPath (accepts any absolute path) and read the file from disk"
    category: security
    reason: "Verifier probe: unbound and alpha-scoped preview_edit of beta/main.go both return symbols_removed ['beta/main.go::ZebraWidget']; read_file refuses the same alpha-scoped path. No raw source line was returned, and an absolute path outside every root produced no symbol. So the observed disclosure is graph metadata (symbol IDs) to a bound out-of-scope request, plus a theoretical LSP-diagnostic quote channel and simulate_chain keep:true overlay promotion. tools_simulate.go and overlay_view.go were not modified this round, and 04-19 recorded this as a known residual. Resolve by routing groupEditByFile through resolveFilePath and adding preview_edit/simulate_chain to the TestSourceScope_ inventories."
    evidence_status: "probe shows metadata disclosure only; no source-byte leak demonstrated"
  - finding: "WR-09: the view-checkout override in requirePathSourceScope (and the copied rootRepo rule in generate_skill) beats a more specific registered owner tracked inside a linked worktree"
    category: security
    reason: "Code-read confirmed (source_scope.go:118-122 overwrites physical unconditionally when the view contains the path). Requires a repository tracked inside another repository's materialized worktree checkout. Not probed; no deterministic evidence. All source surfaces share this attribution, so they agree with each other, but the attribution is wrong for that topology."
    evidence_status: "none provided"
human_verification:
  - test: "Decide the refusal-versus-skip policy for out-of-scope references across rename_symbol, move_symbol and inline_symbol (prior WR-04 / IN-06, new WR-08)"
    expected: "Either accept rename_symbol's distinct 'some references lie outside the resolved scope' message formally, and decide whether move/inline should refuse the same way or emit a fixed advisory; or replace all three with generic text. Today move_symbol/inline_symbol silently skip out-of-scope callers and report success while leaving them non-compiling (verifier probe: beta/use.go still calls a.Foo() after Foo moved to package b; the result carries no scope advisory)"
    why_human: "Policy decision between an existence oracle (refuse with a distinct message) and a silent broken refactor (skip). The 04-22 plan truths require the skip; review WR-08 calls it a correctness defect."
  - test: "Review the lifecycle failure-path and concurrency warnings: WR-06 (declared roots over-prune when a declared repository fails to register; nothing re-admits), WR-07 (enclosing-reconcile failures only logged; track/untrack report success), REVIEW-FIX WR-01 (cobolReconcileMu, deferred reconcile), prior re-review WR-01/WR-02 (untrack restub under contention; reconcile sites outside the reach gate), IN-05 (hidden mi.mu.RLock on the walk path)"
    expected: "Accept as warnings with follow-up items, or schedule fixes. WR-06 verifier probe: DeclarePendingRoots([alpha, beta]) then TrackRepoCtx(alpha) then release() without beta registering leaves alpha/nested/main.go absent while RepoForFile(beta/main.go) = alpha and shouldExclude is false; only an explicit IncrementalReindexRepo restores it"
    why_human: "These are error paths and timing windows. A deterministic failure reproduces the same state on a clean rebuild, so SC-4 equivalence is not violated for identical inputs; a transient failure diverges until the next reindex or restart. Whether that is acceptable for Phase 4 is a product decision."
---

# Phase 4: Stable Incremental and Cross-Repository Lifecycle Verification Report

**Phase Goal:** Users receive the same canonical scoped deterministic graph from identical final inputs regardless of reruns, edits, deletions, restores, or clean rebuilds.
**Verified:** 2026-10-13T22:56:01Z (HEAD 2c5d9354)
**Status:** gaps_found
**Re-verification:** Yes. This follows gap-closure round 2 (plans 04-19, 04-20, 04-21, 04-22).

## Summary of this pass

- **Both prior gaps are closed.** I checked each one with the round-2 tests (re-run under `-race`) and by re-running my round-1 probes against HEAD.
  - **Explore (prior gap 1).** An unbound explore is now refused. An alpha-scoped explore in the nested `outer_indexed_first` state returns no beta bytes.
  - **Nested topology (prior gap 2).** Untracking beta admits alpha/nested through incremental reconcile, matching the clean build. Tracking beta later evicts alpha's copies with no manual reindex. The `projects:` shape indexes the file once. With an inactive entry, alpha owns the file and RepoForFile agrees.
- **One new gap blocks: review CR-04, which I confirmed.** The per-entry walk gate 04-22 added fails open for a symlink whose target lies outside every tracked root.
  - `generate_skill` copies that file's bytes into the repository's skill bundle, and `read_file` then serves the copy.
  - `move_symbol` and `inline_symbol` replace a `.go` symlink with an in-repository copy of the outside file.
  - `read_file` refuses the same path for the same scope, so the source surfaces disagree.
- **The other new review findings are warnings, advisories or human decisions.** WR-06, WR-07 and WR-08 are human decisions. WR-09 and WR-10 are advisories. Rationale is below.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Stable, non-colliding identities and provenance | ✓ VERIFIED (regression check) | No identity code touched in round 2. The orchestrator's full `-race` suite is green: 157 packages ok. |
| 2 | Identical reindex reuses identities, no duplicates | ✓ VERIFIED (regression check) | Same as row 1. TestCobolLifecycle_* is in the green indexer run. |
| 3 | Edit/delete/restore/version/retrieval/config changes invalidate stale observations | ✓ VERIFIED (regression check) | Same as row 1. |
| 4 | Incremental equals clean for identical final inputs | ✓ VERIFIED (prior partial now closed) | `TestCobolLifecycle_NestedTrackLater` and `TestCobolLifecycle_NestedUntrack` (graphdump digest equality) pass under -race, as do `TestNestedTrackedRoot_TrackLaterEvicts`, `UntrackAdmits/{UntrackRepo,UntrackRepoChecked}`, `LifecycleTrackAndUntrack`, `ProjectsMapIndexedOnce`, `OuterOwnsUnindexedNestedEntry/*` and `DeclaredRootsPruneWarmupOrder` (verifier run). Re-running the round-1 probes at HEAD: projects: map, alpha/nested=false and beta present; inactive entry, alpha/nested=true and RepoForFile=alpha; untrack, shouldExclude=false and nested present (matching clean); track-later, nested evicted with no manual IndexRepo. |
| 5 | All downstream surfaces fail closed on missing scope; no same-name cross-repo binding; zero unowned/misprefixed for the acceptance corpus | ✗ FAILED (new gap, CR-04) | Binding refusal, the ownership audit and the explore gate hold. But `requirePathSourceScope` fails open when a location has no owner. Through the walks of generate_skill, move_symbol and inline_symbol, an alpha-scoped request receives bytes from outside every tracked root, while `read_file` refuses the same path for the same scope. |

**Score:** 114/116 must-haves (roadmap 4/5; plan truths 110/111). No truth is present but behavior-unverified.

### Plan Must-Have Truths: round 2 in full, earlier plans by regression

| Plan | Truth | Status | Evidence |
|------|-------|--------|----------|
| 04-19 #1 | Unbound explore of the artifact task is refused with scope text, no repo name, no bytes; beta-scoped positive control | ✓ VERIFIED | `TestSourceScope_UnboundMultiRepoRefused/explore` PASS (-race). Verifier probe: unbound explore isError=true, leak=false. Code: tools_explore.go:2753-2757 (decided from `artifactIntent` only) |
| 04-19 #2 | Artifact lane admits only through filterTextMatchesByResolvedScope; nested both states | ✓ VERIFIED | `TestSourceScope_NestedCheckoutRefused/{tracked_together,outer_indexed_first}/explore` PASS. explore_artifact_intent.go:400 |
| 04-19 #3 | No over-refusal for ordinary code tasks | ✓ VERIFIED | Asserted in UnboundMultiRepoRefused, which passes |
| 04-19 #4 | Structural source-channel inventory enforced | ✓ VERIFIED | `TestSourceScope_SourceChannelsInventoried` PASS (go/parser over non-test files). Limitation: name-keyed and Grep*/Walk* only, so it cannot see the os.ReadFile and readSimFile channels (IN-08) |
| 04-19 #5 | search_symbols content fallback gated | ✓ VERIFIED | `TestSourceScope_SearchSymbolsContentFallbackGated` PASS |
| 04-20 #1 | projects:-map nested pair indexed once, audit clean | ✓ VERIFIED | `TestNestedTrackedRoot_ProjectsMapIndexedOnce` PASS, plus verifier probe |
| 04-20 #2 | Entry outside the batch: alpha owns the file, RepoForFile agrees | ✓ VERIFIED | `OuterOwnsUnindexedNestedEntry/{other_active_project,project_filter}` PASS, plus probe |
| 04-20 #3 | After UntrackRepo(beta), the same Indexer stops excluding | ✓ VERIFIED | `TestNestedTrackedRoot_UntrackStopsExcluding` PASS, plus probe (sameIndexer=true, shouldExclude=false) |
| 04-20 #4 | Daemon warmup declares its exact list before the pool | ✓ VERIFIED | daemon_state.go:644-645 `DeclarePendingRoots(repos)` with `defer release()` inside the batch callback; `DeclaredRootsPruneWarmupOrder` PASS |
| 04-20 #5 | Pruning reads only live state; no per-Indexer copy; no per-file config read | ✓ VERIFIED | The `Indexer.trackedRepoRoots` field and seed are deleted (no indexer match for `trackedRepoRoots`). workspace_resolve.go reads `mi.repos` and `mi.pendingRoots` under RLock |
| 04-21 #1 | Track nested later evicts outer copies in the same call; COBOL digest equal either order | ✓ VERIFIED | `TrackLaterEvicts` and `TestCobolLifecycle_NestedTrackLater` PASS. multi.go:2916-2924 |
| 04-21 #2 | Untrack admits subtree in the same call (UntrackRepo, UntrackRepoChecked, purge); COBOL digest equals clean | ✓ VERIFIED | `UntrackAdmits/*`, `LifecycleTrackAndUntrack` and `TestCobolLifecycle_NestedUntrack` PASS. repository_untrack.go:44-61 |
| 04-21 #3 | Reconcile runs after locks are released, is skipped inside a topology batch, and the SUMMARY lists callers | ✓ VERIFIED | multi.go:2952 `activeRepositoryTopologyBatch` check; `TrackInsideTopologyBatchCompletes` PASS; 04-21-SUMMARY lines 176-206 list callers and the locks they hold |
| 04-21 #4 | MCP outer_indexed_first fixture registers beta through ReconcileRepoCtx | ✓ VERIFIED | source_scope_test.go:340; NestedCheckoutRefused PASS in both states |
| 04-22 #1 | Alpha-scoped move_symbol leaves nested beta byte-identical | ✓ VERIFIED | `TestMoveSymbol_NestedTrackedRootNotRewritten` PASS. Side effect: beta is left non-compiling (WR-08, human item 1) |
| 04-22 #2 | Alpha-scoped inline_symbol leaves beta byte-identical | ✓ VERIFIED | `TestInlineSymbol_NestedTrackedRootNotRewritten` PASS |
| 04-22 #3 | generate_skill lists no nested beta file | ✓ VERIFIED | `TestGenerateSkill_NestedTrackedRootExcluded` PASS |
| 04-22 #4 | Each walk admits entries only through requirePathSourceScope; no nested root or symlink brings **another repository's** bytes in | ✓ VERIFIED (literal) | Verifier probe: alpha/a/blink.txt pointing to beta/use.go is not bundled. The truth names "another repository's bytes", and an outside-every-root target is no repository's, so the literal truth holds. The 04-22 code comments claim the whole symlink class, and that wider claim is false (gap). |
| 04-22 #5 | find_declaration regex filters before cap | ✓ VERIFIED | `TestFindDeclaration_RegexFiltersBeforeLimit` PASS. The per-repository cap before the physical-owner filter remains (WR-05 residual; a false negative, not a leak) |
| 04-17 #3 | "no two source surfaces disagree for one allow-set" | ✗ FAILED | For allow-set {alpha} and path alpha/a/link.txt: read_file refuses, while generate_skill copies the bytes and move_symbol materializes them (probe) |
| 04-12 #1 | Every source-returning tool refuses an unbound multi-repo request and an out-of-scope repository | ✓ VERIFIED (prior gap closed) | Explore now refuses. No other unbound leak of source bytes was found (preview_edit returns symbol IDs only, see WR-10 advisory) |
| 04-01..04-18 (others) | as in prior reports | ✓ VERIFIED (regression) | Full suite green; round 2 touched only the files listed |

### Required Artifacts (round 2)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/mcp/tools_explore.go` | unbound refusal before artifact lane | ✓ VERIFIED | `verify.artifacts` 04-19 3/3 |
| `internal/mcp/explore_artifact_intent.go` | filterTextMatchesByResolvedScope on lane matches | ✓ VERIFIED | line 400 |
| `internal/mcp/source_scope_test.go` | explore cases, inventory, content-fallback tests | ✓ VERIFIED | all PASS |
| `internal/indexer/multi.go` | pendingRoots, DeclarePendingRoots, reconcileEnclosingRepos, seed removed | ✓ VERIFIED | `verify.artifacts` 04-20 5/5, 04-21 5/5 |
| `internal/indexer/workspace_resolve.go`, `indexer.go` | live registered plus pending roots | ✓ VERIFIED | |
| `internal/indexer/repository_untrack.go` | admit on untrack | ✓ VERIFIED | |
| `cmd/gortex/daemon_state.go` | warmup declares its list | ✓ VERIFIED | |
| `internal/mcp/tools_move_inline.go`, `tools_generate_skill.go` | per-entry requirePathSourceScope | ⚠️ PRESENT, FAILS OPEN | `verify.artifacts` 04-22 6/6. Wired, but the gate admits owner-less (outside-root) symlink targets (gap) |
| `internal/mcp/tools_find_declaration.go` | GrepRegexpForRepos, filter then cap | ✓ VERIFIED | |

### Key Link Verification

| Plan | Links | Status |
|------|-------|--------|
| 04-19 | handleExplore→requireSourceScope; lane→filterTextMatchesByResolvedScope; inventory→gatherExploreArtifactLane | ✓ WIRED (`verify.key-links` 3/3) |
| 04-20 | inNestedTrackedRoot→nestedTrackedRootContains; pendingRoots; daemon DeclarePendingRoots(repos) | ✓ WIRED (3/3) |
| 04-21 | TrackRepoCtx→reconcileEnclosingRepos; untrack→reconcileEnclosingRepos; batch skip | ✓ WIRED (3/3) |
| 04-22 | move/inline/generate_skill walks→requirePathSourceScope; find_declaration→GrepRegexpForRepos | ✓ WIRED (3/3). The wired gate is incomplete for owner-less locations (gap) |

### Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|----------|------|--------|-----------|--------|
| nested-root pruning | registered and declared roots | live `mi.repos` and `mi.pendingRoots` under RLock | Yes; refreshed on track, untrack and release | ✓ FLOWING (residual: WR-06 failed-registration path) |
| explore artifact snippet | `match.Text` | GrepText, then physical-owner filter | Yes, gated | ✓ FLOWING |
| generate_skill bundle bytes | `os.ReadFile(path)` per walk entry | filesystem, following symlinks | Gated for registered owners; ungated for owner-less targets | ✗ leaks outside-root bytes (gap) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Round-2 MCP tests | `GOWORK=off go test -race -count=1 -run '^TestSourceScope_\|^TestMoveSymbol_Nested…\|^TestInlineSymbol_Nested…\|^TestGenerateSkill_Nested…\|^TestFindDeclaration_RegexFiltersBeforeLimit$\|^TestSearchSymbols_ContentFallback' ./internal/mcp/` | ok 18.3s; 41 PASS lines including explore subtests in both nested states | ✓ PASS |
| Round-2 indexer tests | `GOWORK=off go test -race -count=1 -run '^TestNestedTrackedRoot_\|^TestCobolLifecycle_Nested' ./internal/indexer/` | ok 20.7s; 13 top-level PASS | ✓ PASS |
| Full suite | (orchestrator) `GOWORK=off go test -race -count=1 -timeout 45m ./...` | 157 packages ok | ✓ PASS |
| gofmt / debt markers on the 16 round-2 files | `gofmt -l`; grep TBD/FIXME/XXX | clean; only feature-code literals in indexer.go:1631-1637 | ✓ PASS |
| Round-1 explore probe at HEAD | `go test -overlay … -run '^TestVerifierProbe_' ./internal/mcp/` | unbound explore refused; nested alpha-scoped no leak; sibling no leak | ✓ PASS (gap 1 closed) |
| Round-1 nested probes at HEAD | `go test -overlay … -run '^TestVerifierProbe_' ./internal/indexer/` | projects:, inactive, untrack, track-later all converge (see SC-4 row) | ✓ PASS (gap 2 closed) |
| CR-04 probe | `go test -overlay … -run '^TestVerifierR3_CR04' ./internal/mcp/` | read_file refuses alpha/a/link.txt; generate_skill bundles it (25 bytes, secret on disk); read_file serves the copy; move_symbol turns zlink.go into an in-repo copy of the outside file; sibling layout also lists link.txt | ✗ FAIL |
| WR-10 probe | `… -run '^TestVerifierR3_WR10' ./internal/mcp/` | unbound and alpha-scoped preview_edit of beta/main.go: isError=false, symbols_removed=[beta/main.go::ZebraWidget]; no raw lines; outside-root absolute path yields no symbols | ⚠️ metadata disclosure (advisory) |
| WR-06 probe | `… -run '^TestVerifierR3_WR06' ./internal/indexer/` | declared-but-unregistered beta: alpha/nested absent, RepoForFile=alpha, shouldExclude=false; explicit incremental restores | ⚠️ failure-path divergence (human item 2) |

Probe sources live outside the tree and ran only through `-overlay`; the repository was not modified:
- `/private/tmp/claude-501/-Users-e1001547-mbp-it-repos-mine-GoApps/6332945b-c602-4ced-bd7d-35c7715acfa1/scratchpad/probe3/zz_verifier_r3_test.go`
- `/private/tmp/claude-501/-Users-e1001547-mbp-it-repos-mine-GoApps/6332945b-c602-4ced-bd7d-35c7715acfa1/scratchpad/probe3/zz_vr3_idx_test.go`
- the round-1 probes in `.../scratchpad/probe/`

### Probe Execution

Step 7c: no `scripts/*/tests/probe-*.sh` is declared or present. Not applicable.

### Review claim adjudication (04-REVIEW.md, round 2)

| Finding | Verifier result | Classification |
|---------|-----------------|----------------|
| CR-04: walk gate admits outside-root symlinks | **Confirmed** by probe for generate_skill (copy plus read_file serving it) and move_symbol (link replaced by an in-repo copy). Surfaces disagree: read_file refuses the same path for the same scope (guardSymlinkWithinRepo). It predates the phase in generate_skill (upstream 47fceda9), but the gate's fail-open on `physical == ""` is phase-4 code (04-17), and 04-22 reused it on paths that were never confined while claiming symlink coverage. Files changed this round and a deterministic probe exists, so the evidence gate keeps it blocking. | 🛑 Gap (SC-5, SEC-01; 04-17 #3; 04-22 SEC-02 prohibition) |
| WR-06: declared roots over-prune on failed registration | **Confirmed** by probe (mechanism). It is an error path, logged; a deterministic failure reproduces on a clean rebuild. | ⚠️ Warning, human item 2 |
| WR-07: enclosing-reconcile failure only logged | **Confirmed** by code (multi.go:2988-2993 Warn only). Error path. | ⚠️ Warning, human item 2 |
| WR-08: move/inline silently leave out-of-scope callers broken | **Confirmed** by probe (beta/use.go still calls a.Foo(); no advisory in the result). Required by the 04-22 truths. | Human item 1 (policy) |
| WR-09: view override beats a nested registered owner | Code-read confirmed; not probed; needs a repository tracked inside a linked worktree. | 📋 Advisory |
| WR-10: preview_edit/simulate_chain ungated read | **Partly confirmed.** It reads out-of-scope and absolute paths, but returns graph metadata (symbol IDs), not raw bytes. Files unchanged since the prior pass. | 📋 Advisory |
| WR-03, WR-05 residual, IN-01..IN-10 | Noted. WR-03's symlinked-root shape still yields a single owner (round-1 probe at HEAD). | ℹ️ Info |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| ID-03 | 04-02 | Paragraph identity includes owning program | ✓ SATISFIED | Unchanged; suite green |
| ID-04 | 04-01, 04-07, 04-11, 04-14 | Resource identity namespace | ✓ SATISFIED | Unchanged |
| ID-05 | 04-02 | Distinguishable occurrence provenance | ✓ SATISFIED | Unchanged |
| ID-06 | 04-01, 04-11 | Identical inputs give identical identities | ✓ SATISFIED | Unchanged |
| ID-07 | 04-06, 04-08, 04-14, 04-15 | Stale observations removed on change or delete | ✓ SATISFIED | Unchanged |
| ID-08 | 04-09, 04-16 | Parser/extractor/retrieval/config change invalidates | ✓ SATISFIED | Unchanged |
| ID-09 | 04-07..04-10, 04-15, 04-16, 04-20, 04-21 | Incremental equals clean | ✓ SATISFIED | Nested track-later and untrack now digest-equal to clean (TestCobolLifecycle_Nested*). Failure-path caveat WR-06/WR-07 (human item 2) |
| SEC-01 | 04-03, 04-10, 04-12, 04-17, 04-18, 04-19, 04-22 | Same scope across surfaces | ✗ BLOCKED | Walk tools admit owner-less symlink targets that read_file refuses (gap) |
| SEC-02 | 04-12, 04-17, 04-19, 04-22 | Absent scope fails closed for source reads and AI context | ✓ SATISFIED | Explore and every inventoried source surface refuse unbound multi-repo requests. CR-04 is an out-of-tree attribution failure under a bound scope, not an unbound read; it is tracked under SEC-01 |
| SEC-03 | 04-04, 04-06, 04-07, 04-10, 04-14 | No same-name cross-repo binding without allowlist and evidence | ✓ SATISFIED | Unchanged |
| SEC-08 | 04-05, 04-10, 04-13, 04-18, 04-20, 04-21 | Zero unowned/misprefixed for the acceptance corpus | ✓ SATISFIED | Ownership audit green; nested double ownership closed in all four probed shapes |

Every phase requirement ID appears in at least one plan's `requirements:`, so none is orphaned. REQUIREMENTS.md currently marks all eleven "Gaps Found", which matches this report: SEC-01 stays open.

### Prohibitions (judgment tier; non-authoritative LLM verdicts)

| Plan | Prohibition | Verdict |
|------|-------------|---------|
| 04-19 SEC-02 | MUST NOT decide explore's unbound refusal from search results | ✓ Holds (decided from `artifactIntent` before any grep) |
| 04-19 SEC-01 | MUST NOT emit a snippet or raise a score from an out-of-scope physical owner | ✓ Holds (filter runs before enrichment and scoring) |
| 04-20 SEC-08 | MUST NOT index a nested root that the batch indexes or that is registered; MUST NOT exclude a root that is neither | ✓ Holds on the success path. ⚠️ Flagged on a failed registration (WR-06) |
| 04-20 SEC-01 | MUST NOT read the config per file or event, or copy roots into an Indexer | ✓ Holds |
| 04-21 ID-09 | MUST NOT leave the outer graph different from clean after a nested track or untrack | ✓ Holds (digest tests). ⚠️ Flagged on reconcile failure (WR-07) |
| 04-21 SEC-08 | MUST NOT reconcile while holding mi.mu, a lane, the gate, the writer, or inside a batch | ✓ Holds (code plus SUMMARY caller audit) |
| 04-22 SEC-01 | MUST NOT write a file whose physical owner is outside scope from a walk | ✗ Violated in spirit: move_symbol writes outside-root bytes into an alpha file by replacing the symlink (the owner is "none", which the gate treats as alpha) |
| 04-22 SEC-02 | MUST NOT copy or list a file whose physical owner is outside scope from generate_skill's walk | ✗ Violated: an owner-less outside-root file is listed and copied |
| 04-12 SEC-02 privacy | MUST NOT reveal out-of-scope existence through a refusal | ⚠️ Flagged (rename message; human item 1) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/mcp/source_scope.go | 124 | Fail-open on an unresolved physical owner (`physical == ""` → nil) | 🛑 Blocker | Gap. Callers that do not confine first leak outside-root bytes |
| internal/mcp/tools_generate_skill.go | ~146-205 | Comment claims symlink coverage the gate does not provide; os.ReadFile follows the link | 🛑 Blocker | Gap (same root cause) |
| internal/mcp/tools_move_inline.go | ~248-283, ~1143-1172 | Same claim; rename-write replaces the link with outside bytes | 🛑 Blocker | Gap (same root cause) |
| internal/indexer/multi.go | 2988-2993 | Error swallowed to a Warn log | ⚠️ Warning | WR-07 |
| internal/indexer/multi.go | 2126-2127 | `indexMultiRepo` doc line now sits above `DeclarePendingRoots` | ℹ️ Info | IN-07 |
| (round-2 added lines) | | TBD/FIXME/XXX/TODO/HACK | none | |

### Human Verification Required

### 1. Out-of-scope reference policy for rename, move and inline (WR-04/IN-06 carried, WR-08 new)

**Test:** Bind a request to alpha. Run `rename_symbol` (dry_run), `move_symbol` and `inline_symbol` on an alpha symbol that nested beta references.
**Expected:** Decide one policy for all three:
- refuse with a message, either generic or the current distinct one; or
- skip with a fixed advisory.

Today rename refuses with a distinct message, while move and inline silently skip and report success, leaving beta non-compiling.
**Why human:** This is a policy trade-off. Refusing reveals that references exist; skipping silently breaks the refactor.

### 2. Lifecycle failure paths and concurrency warnings

**Test:** Review these items:
- WR-06: declared-but-failed registration over-prunes (probe-confirmed).
- WR-07: enclosing-reconcile errors are only logged.
- Carried items: REVIEW-FIX WR-01, prior re-review WR-01 and WR-02, and IN-05.

**Expected:** Accept each as a warning with a follow-up item, or schedule a fix.
**Why human:** These are error-path and timing windows. They do not violate incremental-equals-clean for identical inputs on the success path.

### Gaps Summary

There is **one gap (BLOCKER: SC-5, SEC-01)**, and it has a single root cause.

`requirePathSourceScope` treats a location with no registered owner as belonging to the lexical repository. It was written for callers that confine the path first, and `resolveFilePath` and `resolveNodePath` do. 04-22 then used it as the only per-entry gate in three filesystem walks that do not confine. As a result:
- an alpha-scoped `generate_skill` bundles a symlinked file from anywhere the daemon can read;
- `move_symbol` and `inline_symbol` materialize such a file inside alpha.

`read_file` refuses the same path for the same scope, so the surfaces disagree. The walk behavior predates the phase, but these walks were rewritten in this round to close "a symlinked file" and they did not. This is an OWASP A01 path-traversal and confinement issue, and it breaks the SECURITY.md invariant. No later phase in the roadmap covers it.

The fix is small and local:
- confine each walk entry with `guardSymlinkWithinRepo`, or make `requirePathSourceScope` refuse an owner-less location that fails `guardResolvedPathWithinRepo`;
- make the check independent of `IsMultiRepo`;
- add symlink-out-of-root tests for the three tools, including a single-repository variant.

I do not suggest an override. The tool writes bytes from outside every tracked repository into a repository and then serves them.

**Everything else holds.**
- Both prior gaps are closed and backed by tests: the explore gate (04-19) and the nested-topology lifecycle (04-20, 04-21), including digest equivalence for nested track-later and untrack.
- The lifecycle core is unchanged and green: SC-1 to SC-4, ID-03 to ID-09, SEC-03 and SEC-08.

---

_Verified: 2026-10-13T22:56:01Z_
_Verifier: Claude (gsd-verifier)_
