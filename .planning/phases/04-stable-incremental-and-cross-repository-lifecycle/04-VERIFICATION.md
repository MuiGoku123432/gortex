---
phase: 04-stable-incremental-and-cross-repository-lifecycle
verified: 2026-10-14T03:42:42Z
status: human_needed
score: 116/116 must-haves verified (5/5 roadmap success criteria; 111/111 plan truths)
covered_files: [".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-01-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-02-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-03-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-04-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-05-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-06-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-07-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-08-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-09-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-10-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-11-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-12-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-13-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-14-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-15-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-16-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-17-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-17-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-18-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-18-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-19-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-19-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-20-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-20-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-21-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-21-SUMMARY.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-22-PLAN.md",".planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-22-SUMMARY.md","cmd/gortex/daemon_state.go","internal/indexer/cobol_lifecycle_crossrepo_test.go","internal/indexer/indexer.go","internal/indexer/multi.go","internal/indexer/nested_tracked_root_test.go","internal/indexer/repository_untrack.go","internal/indexer/workspace_resolve.go","internal/mcp/explore_artifact_intent.go","internal/mcp/overlay_view.go","internal/mcp/source_scope.go","internal/mcp/source_scope_test.go","internal/mcp/tools_explore.go","internal/mcp/tools_fileops.go","internal/mcp/tools_find_declaration.go","internal/mcp/tools_find_declaration_test.go","internal/mcp/tools_generate_skill.go","internal/mcp/tools_generate_skill_test.go","internal/mcp/tools_move_inline.go","internal/mcp/tools_move_inline_test.go","internal/mcp/tools_simulate.go"]
covered_digest: "v3:sha256:2ce3b3678ecd31eaa573f204e960ae86d1bf87971a387d5dcb4e29278cd2d4be"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 114/116
  gaps_closed:
    - "SC-5 / SEC-01 / 04-17 truth 3 (CR-04): each walk file entry in move_symbol, inline_symbol and generate_skill now passes guardSymlinkWithinRepo before requirePathSourceScope (fix 5b4e4597), so a symlink pointing outside every tracked root is excluded exactly as read_file refuses it, in multi-repo and single-repo mode"
  gaps_remaining: []
  regressions: []
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
**Verified:** 2026-10-14T03:42:42Z (HEAD 5b4e4597)
**Status:** human_needed
**Re-verification:** Yes, round 4. A targeted re-check of the single CR-04 gap after fix commit 5b4e4597. The round-3 pass (HEAD 2c5d9354) verified everything else; those findings are kept below.

## Round 4: CR-04 re-verification (fix 5b4e4597)

**The CR-04 gap is closed.** Fix 5b4e4597 changes three lines. Each walk FILE entry now calls `s.guardSymlinkWithinRepo(ctx, p)` before `requirePathSourceScope`, at `tools_move_inline.go:273` (move_symbol), `tools_move_inline.go:1166` (inline_symbol) and `tools_generate_skill.go:180`. `read_file` confines paths with the same function. `guardSymlinkWithinRepo` resolves symlinks and checks the result against `guardRepoRoots`, which includes every registered multi-repo root, `s.indexer.RootPath()` and the view root. It does not depend on `IsMultiRepo`.

I checked it with a probe run through `go test -overlay`; the repository tree was not touched. The probe files are `scratchpad/probe3/zz_verifier_r3_test.go` and `scratchpad/probe4/zz_vr4_test.go`.

| Check | Result at HEAD 5b4e4597 |
|-------|-------------------------|
| Round-3 probe, nested fixture: `alpha/a/link.txt` and `alpha/a/zlink.go` point outside every root | `read_file` refuses link.txt. `generate_skill` bundles only `[a/a.go b/b.go c/c.go go.mod]`, with no copy of link.txt on disk. Reading the would-be bundle path through `read_file` is refused (the file does not exist). After `move_symbol`, zlink.go is still a symlink and the outside x.go is byte-identical. Before the fix, the link was bundled and rewritten. |
| Sibling-repo symlink (`alpha/a/blink.txt` → `beta/use.go`) | Still gated. It is not bundled, because the confinement passes and `requirePathSourceScope` then refuses the physical owner beta. In the two-repo sibling fixture, `generate_skill` dry_run no longer lists link.txt (round 3 listed it). |
| `inline_symbol`, nested fixture | zlink.go stays a symlink, the outside file is unchanged, and alpha's own caller c.go is inlined. |
| Symlinked DIRECTORY (`alpha/c/sdir` → outside dir holding s.go and secret.txt) | Not descended. `generate_skill` lists nothing under sdir, and `inline_symbol` leaves sdir/s.go unchanged. This matches the stdlib contract: `filepath.Walk` uses Lstat and `filepath.WalkDir` uses DirEntry types, so neither follows a symlinked directory below the root. A symlinked directory reaches the file branch as a non-directory and passes the same guard. The walk roots come from `resolveFilePath` (generate_skill) and `findGoModRoot` of a resolved source file (move/inline). |
| Single-repo mode (`setupMoveInlineRepo`, no multiIndexer) | `read_file link.txt` is refused. `generate_skill` bundles `[go.mod pkga/a.go pkgb/b.go pkgc/c.go]` and does not copy link.txt. `move_symbol` and `inline_symbol` both rewrite the in-repo caller, keep `pkgc/zlink.go` a symlink, and leave the outside file byte-identical. |
| Single repo whose go.mod lives in an untracked parent (the module walk leaves the root) | `move_symbol` leaves `../other/o.go`, which lies outside the tracked root and calls `pkga.Foo()`, byte-identical. The guard refuses ownerless files in this case too. |
| Stale-graph swap (an indexed caller is replaced by an outside symlink before reindex), through the edge loop and `resolveNodePath` | move_symbol and inline_symbol leave the link and the outside file unchanged. `get_symbol_source` and `read_file` refuse. |
| Named tests, `-race` | `TestMoveInline_SymlinkOutsideRootsNotRewritten/{move_symbol,inline_symbol}`, `TestGenerateSkill_SymlinkOutsideRootsExcluded` and `TestSourceScope_SourceChannelsInventoried` PASS. |
| gofmt and debt markers on the 4 changed files | Clean. |

**No other surface in internal/mcp has the same ownerless-path hole.**
- The only `filepath.Walk` and `WalkDir` sites are the three gated above. `TestSourceScope_SourceChannelsInventoried` enforces that inventory and passes.
- The other `requirePathSourceScope` callers confine first or take indexed paths only:
  - `resolveFilePath` → `resolveConfinedFilePath`.
  - `resolveNodePath` and `resolveGraphPath` take graph paths. The indexer's `shouldExclude` → `pathguard.SymlinkEscapes` never admits an escaping symlinked file.
  - The trigram filters in `find_declaration` and `search_text` only see indexed paths.
- The only `os.ReadDir` site is `inferGoPackageForDir` (`tools_move_inline.go:633`). It reads just the package clause of a sibling file in the confined target directory and emits no bytes (ℹ️ Info).

ℹ️ The `walkGates` descriptions in `TestSourceScope_SourceChannelsInventoried` still say "per-entry requirePathSourceScope (04-22)". They do not mention the added confinement. This is cosmetic.

## Round 3 summary (HEAD 2c5d9354, carried)

- **Both prior gaps are closed.** I checked each one with the round-2 tests (re-run under `-race`) and by re-running my round-1 probes against HEAD.
  - **Explore (prior gap 1).** An unbound explore is now refused. An alpha-scoped explore in the nested `outer_indexed_first` state returns no beta bytes.
  - **Nested topology (prior gap 2).** Untracking beta admits alpha/nested through incremental reconcile, matching the clean build. Tracking beta later evicts alpha's copies with no manual reindex. The `projects:` shape indexes the file once. With an inactive entry, alpha owns the file and RepoForFile agrees.
- **One new gap blocked in round 3: review CR-04 (closed in round 4, see above).** The per-entry walk gate 04-22 added fails open for a symlink whose target lies outside every tracked root.
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
| 5 | All downstream surfaces fail closed on missing scope; no same-name cross-repo binding; zero unowned/misprefixed for the acceptance corpus | ✓ VERIFIED (round 4, CR-04 closed) | Binding refusal, the ownership audit and the explore gate hold. The three walk tools now confine each file entry with `guardSymlinkWithinRepo` before the owner gate, so an outside-root symlink is excluded wherever `read_file` refuses it. Checked by probe in nested, sibling and single-repo modes, and by the new tests under -race. |

**Score:** 116/116 must-haves (roadmap 5/5; plan truths 111/111). No truth is present but behavior-unverified.

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
| 04-22 #4 | Each walk admits entries only through requirePathSourceScope; no nested root or symlink brings **another repository's** bytes in | ✓ VERIFIED | Sibling-repo symlink not bundled (probe). Since 5b4e4597 the code's wider claim also holds: outside-every-root symlinks are excluded by `guardSymlinkWithinRepo` (round-4 probe) |
| 04-22 #5 | find_declaration regex filters before cap | ✓ VERIFIED | `TestFindDeclaration_RegexFiltersBeforeLimit` PASS. The per-repository cap before the physical-owner filter remains (WR-05 residual; a false negative, not a leak) |
| 04-17 #3 | "no two source surfaces disagree for one allow-set" | ✓ VERIFIED (round 4) | For allow-set {alpha} and alpha/a/link.txt, all four surfaces now refuse or exclude: read_file, generate_skill, move_symbol and inline_symbol. The same holds in single-repo mode (probe) |
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
| `internal/mcp/tools_move_inline.go`, `tools_generate_skill.go` | per-entry guardSymlinkWithinRepo + requirePathSourceScope | ✓ VERIFIED | `verify.artifacts` 04-22 6/6. Lines 273/1166 and 180 confine before the owner gate (5b4e4597) |
| `internal/mcp/tools_find_declaration.go` | GrepRegexpForRepos, filter then cap | ✓ VERIFIED | |

### Key Link Verification

| Plan | Links | Status |
|------|-------|--------|
| 04-19 | handleExplore→requireSourceScope; lane→filterTextMatchesByResolvedScope; inventory→gatherExploreArtifactLane | ✓ WIRED (`verify.key-links` 3/3) |
| 04-20 | inNestedTrackedRoot→nestedTrackedRootContains; pendingRoots; daemon DeclarePendingRoots(repos) | ✓ WIRED (3/3) |
| 04-21 | TrackRepoCtx→reconcileEnclosingRepos; untrack→reconcileEnclosingRepos; batch skip | ✓ WIRED (3/3) |
| 04-22 | move/inline/generate_skill walks→requirePathSourceScope; find_declaration→GrepRegexpForRepos | ✓ WIRED (3/3), with guardSymlinkWithinRepo ahead of the gate since 5b4e4597 |

### Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|----------|------|--------|-----------|--------|
| nested-root pruning | registered and declared roots | live `mi.repos` and `mi.pendingRoots` under RLock | Yes; refreshed on track, untrack and release | ✓ FLOWING (residual: WR-06 failed-registration path) |
| explore artifact snippet | `match.Text` | GrepText, then physical-owner filter | Yes, gated | ✓ FLOWING |
| generate_skill bundle bytes | `os.ReadFile(path)` per walk entry | filesystem, following symlinks | Confined to tracked roots, then gated by physical owner | ✓ FLOWING (gated) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Round-2 MCP tests | `GOWORK=off go test -race -count=1 -run '^TestSourceScope_\|^TestMoveSymbol_Nested…\|^TestInlineSymbol_Nested…\|^TestGenerateSkill_Nested…\|^TestFindDeclaration_RegexFiltersBeforeLimit$\|^TestSearchSymbols_ContentFallback' ./internal/mcp/` | ok 18.3s; 41 PASS lines including explore subtests in both nested states | ✓ PASS |
| Round-2 indexer tests | `GOWORK=off go test -race -count=1 -run '^TestNestedTrackedRoot_\|^TestCobolLifecycle_Nested' ./internal/indexer/` | ok 20.7s; 13 top-level PASS | ✓ PASS |
| Full suite | (orchestrator) `GOWORK=off go test -race -count=1 -timeout 45m ./...` | 157 packages ok | ✓ PASS |
| gofmt / debt markers on the 16 round-2 files | `gofmt -l`; grep TBD/FIXME/XXX | clean; only feature-code literals in indexer.go:1631-1637 | ✓ PASS |
| Round-1 explore probe at HEAD | `go test -overlay … -run '^TestVerifierProbe_' ./internal/mcp/` | unbound explore refused; nested alpha-scoped no leak; sibling no leak | ✓ PASS (gap 1 closed) |
| Round-1 nested probes at HEAD | `go test -overlay … -run '^TestVerifierProbe_' ./internal/indexer/` | projects:, inactive, untrack, track-later all converge (see SC-4 row) | ✓ PASS (gap 2 closed) |
| CR-04 probe (round 3, at 2c5d9354) | `go test -overlay … -run '^TestVerifierR3_CR04' ./internal/mcp/` | generate_skill bundled link.txt; move_symbol replaced zlink.go | ✗ FAIL (superseded) |
| CR-04 probe (round 4, at 5b4e4597) | `go test -overlay probe4/overlay.json -run '^TestVerifierR3_CR04\|^TestVR4_' ./internal/mcp/` | All refused or excluded: nested, sibling, inline, symlinked dir, single-repo, parent-module walk, stale-graph swap (see Round 4) | ✓ PASS |
| CR-04 named tests | `GOWORK=off go test -race -count=1 -run '^(TestMoveInline_SymlinkOutsideRootsNotRewritten\|TestGenerateSkill_SymlinkOutsideRootsExcluded\|TestSourceScope_SourceChannelsInventoried)$' ./internal/mcp/` | ok 8.8s; 3 PASS plus 2 subtests | ✓ PASS |
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
| CR-04: walk gate admits outside-root symlinks | **Closed in round 4** by 5b4e4597 (probe and tests above). Round-3 finding: **Confirmed** by probe for generate_skill (copy plus read_file serving it) and move_symbol (link replaced by an in-repo copy). Surfaces disagree: read_file refuses the same path for the same scope (guardSymlinkWithinRepo). It predates the phase in generate_skill (upstream 47fceda9), but the gate's fail-open on `physical == ""` is phase-4 code (04-17), and 04-22 reused it on paths that were never confined while claiming symlink coverage. Files changed this round and a deterministic probe exists, so the evidence gate keeps it blocking. | ✓ Closed (was 🛑 Gap) |
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
| SEC-01 | 04-03, 04-10, 04-12, 04-17, 04-18, 04-19, 04-22 | Same scope across surfaces | ✓ SATISFIED | Walk tools now confine like read_file (5b4e4597; round-4 probe) |
| SEC-02 | 04-12, 04-17, 04-19, 04-22 | Absent scope fails closed for source reads and AI context | ✓ SATISFIED | Explore and every inventoried source surface refuse unbound multi-repo requests. CR-04 is an out-of-tree attribution failure under a bound scope, not an unbound read; it is tracked under SEC-01 |
| SEC-03 | 04-04, 04-06, 04-07, 04-10, 04-14 | No same-name cross-repo binding without allowlist and evidence | ✓ SATISFIED | Unchanged |
| SEC-08 | 04-05, 04-10, 04-13, 04-18, 04-20, 04-21 | Zero unowned/misprefixed for the acceptance corpus | ✓ SATISFIED | Ownership audit green; nested double ownership closed in all four probed shapes |

Every phase requirement ID appears in at least one plan's `requirements:`, so none is orphaned. All eleven are now satisfied. REQUIREMENTS.md still marks them "Gaps Found", and the orchestrator should update that.

### Prohibitions (judgment tier; non-authoritative LLM verdicts)

| Plan | Prohibition | Verdict |
|------|-------------|---------|
| 04-19 SEC-02 | MUST NOT decide explore's unbound refusal from search results | ✓ Holds (decided from `artifactIntent` before any grep) |
| 04-19 SEC-01 | MUST NOT emit a snippet or raise a score from an out-of-scope physical owner | ✓ Holds (filter runs before enrichment and scoring) |
| 04-20 SEC-08 | MUST NOT index a nested root that the batch indexes or that is registered; MUST NOT exclude a root that is neither | ✓ Holds on the success path. ⚠️ Flagged on a failed registration (WR-06) |
| 04-20 SEC-01 | MUST NOT read the config per file or event, or copy roots into an Indexer | ✓ Holds |
| 04-21 ID-09 | MUST NOT leave the outer graph different from clean after a nested track or untrack | ✓ Holds (digest tests). ⚠️ Flagged on reconcile failure (WR-07) |
| 04-21 SEC-08 | MUST NOT reconcile while holding mi.mu, a lane, the gate, the writer, or inside a batch | ✓ Holds (code plus SUMMARY caller audit) |
| 04-22 SEC-01 | MUST NOT write a file whose physical owner is outside scope from a walk | ✓ Holds (round 4: outside-root symlinks are no longer rewritten) |
| 04-22 SEC-02 | MUST NOT copy or list a file whose physical owner is outside scope from generate_skill's walk | ✓ Holds (round 4) |
| 04-12 SEC-02 privacy | MUST NOT reveal out-of-scope existence through a refusal | ⚠️ Flagged (rename message; human item 1) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/mcp/source_scope.go | 124 | `physical == ""` → nil, by design: it relies on the caller confining first | ℹ️ Info | Every caller now confines first or takes only indexed paths (round 4 audit) |
| internal/mcp/source_scope_test.go | 531-533 | `walkGates` descriptions omit the added confinement | ℹ️ Info | Cosmetic |
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

**No gaps remain.** Fix 5b4e4597 closes CR-04, the only round-3 blocker. The three filesystem walks now confine each file entry to the tracked roots with `guardSymlinkWithinRepo`, as `read_file` does, before the physical-owner gate. I checked this in five setups: nested, sibling, single-repo, a parent-module walk and a stale-graph swap. Symlinked directories are not descended. No other walk or surface in internal/mcp has the same ownerless-path hole.

All automated must-haves pass (116/116). The status is **human_needed** because of the two carried human decisions above:
1. The refusal-versus-skip policy for rename, move and inline (WR-04, IN-06, WR-08).
2. The failure-path and concurrency warnings: WR-06, WR-07, REVIEW-FIX WR-01, re-review WR-01/WR-02 and IN-05.

WR-09 and WR-10 stay advisories.

---

_Verified: 2026-10-14T03:42:42Z_
_Verifier: Claude (gsd-verifier)_
