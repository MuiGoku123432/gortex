---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 15
subsystem: indexer
tags: [cobol, incremental, reconcile, warm-restart, affected-by, cross-repo, lifecycle, sqlite]

requires:
  - phase: 04-08
    provides: reconcileCobolRepos, the COBOL reconcile at every incremental mutation tail, the CR-01 restub, and the incremental harness helpers
  - phase: 04-07
    provides: crossa/crossb cross-repository fixtures and the resolve_from allowlist
  - phase: 04-06
    provides: Resolver.ReconcileCobol and its rebound/stamped write split
  - phase: 04-01
    provides: COBOL lifecycle harness (newCobolLifecycleEnv, cleanCobolDump, requireSameDump)
provides:
  - COBOL reconcile at the end of MultiIndexer.ReconcileRepoCtx (warm restart)
  - cross_repo_calls parallel edges for COBOL CALLs the reconcile binds across repositories
  - D-08 escalation at both affected-by sites ("affected-by: cap exceeded, resolving full affected set")
  - TestCobolLifecycle_CrossFile/CrossRepoIncremental/Interrupted/OverCap, TestAffectedBy_CapEscalates
affects: [04-16, phase-6 generic cross-repository re-check, warm restart, affected-by pass]

actuals:
  tokens: 4614
  tasks: 2
  commits: 4
plan_head_before: a8c1413b03168fa1af1c3962f2b6ab6addb262bf

tech-stack:
  added: []
  patterns:
    - "A warm restart always ends with the COBOL reconcile for its repository, whatever route the census chose"
    - "Every resolver that rebinds a calls edge runs DetectCrossRepoEdgesForReindexes on its rebound batch"
    - "The affected-by cap is an escalation threshold, not a truncation"

key-files:
  created: []
  modified:
    - internal/indexer/multi.go
    - internal/resolver/cobol_references.go
    - internal/indexer/affected_by.go
    - internal/indexer/incremental_batch.go
    - internal/config/config.go
    - internal/indexer/cobol_lifecycle_incremental_test.go
    - internal/indexer/affected_by_e2e_test.go

key-decisions:
  - "ReconcileRepoCtx reconciles COBOL unconditionally after its resolve and derived tails, because an interrupted reindex has already recorded its mtimes and the next census can report no change"
  - "Resolver.reconcileCobol materializes the cross_repo_calls parallel edge for its rebound batch, which is the same settle step CrossRepoResolver runs, so the clean track path matches the incremental derived pass"
  - "Above AffectedByReresolveMax both affected-by sites log at Info (file, affected, cap) and resolve the full set (D-08). builder_closure.go keeps the same constant as its own cap, unchanged"
  - "A COBOL copybook edit never reaches the affected-by pass, because programs reference a copybook only through the imports edge to its file node and the reconcile owns that edge. TestCobolLifecycle_OverCap therefore also renames a CALL target that three programs reference, to drive the escalation through COBOL"

patterns-established:
  - "Two-site escalation log with a // note: at each site (rule of three not met)"

requirements-completed: [ID-07, ID-09]

coverage:
  - id: D1
    description: "A new candidate or an edited target in another lifea file re-decides main.cbl's unchanged CALL (ambiguous, missing, then rebinds to the identical ID). Each step equals its clean build"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_CrossFile"
        status: pass
    human_judgment: false
  - id: D2
    description: "Adding and deleting XPGM through an incremental reindex of crossb alone binds crossa's allowed CALL and then returns it to missing. Each step equals its clean build, including the cross_repo_calls row"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_CrossRepoIncremental"
        status: pass
    human_judgment: false
  - id: D3
    description: "An interrupted reindex leaves placeholders and no wrong binding. A fresh MultiIndexer over the same SQLite file, run through ReconcileRepoCtx with persisted mtimes, converges to the clean dump"
    requirement: ID-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_Interrupted"
        status: pass
    human_judgment: false
  - id: D4
    description: "Above the affected-by cap, one Info escalation entry carries affected=3 and cap=1, and all three callers bind to the fresh F. The deferred-batch path still logs nothing"
    requirement: ID-09
    verification:
      - kind: unit
        ref: "internal/indexer/affected_by_e2e_test.go#TestAffectedBy_CapEscalates"
        status: pass
      - kind: unit
        ref: "internal/indexer/affected_by_e2e_test.go#TestAffectedBy_DeferredBatchPath_NoFanout"
        status: pass
    human_judgment: false
  - id: D5
    description: "With index.affected_by_reresolve_max 1, a copybook layout edit and a removed CALL target with three referencing programs resolve the full affected set and equal the clean build"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_OverCap"
        status: pass
    human_judgment: false
  - id: D6
    description: "No regressions in the full indexer, resolver, graphdump, and config packages or in the cmd/gortex COBOL tests"
    verification:
      - kind: unit
        ref: "GOWORK=off go test -count=1 -timeout 40m ./internal/indexer/ ./internal/resolver/ ./internal/testutil/graphdump/ ./internal/config/"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -count=1 ./cmd/gortex/ -run 'Cobol|COBOL'"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestCobolLifecycle_|Reconcile|Checkout|^TestAffectedBy_'"
        status: pass
    human_judgment: false

duration: 23min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 15: Warm-Restart COBOL Reconcile and Affected-By Escalation Summary

**`ReconcileRepoCtx` now ends with the COBOL reconcile, so an interrupted reindex converges on the next warm restart. `ReconcileCobol` now materializes the `cross_repo_calls` parallel edge for CALLs it binds across repositories. Both affected-by sites log an Info escalation and resolve the full set where they used to truncate. Cross-file, cross-repository, interrupted, and over-cap sequences each equal their clean build.**

## Performance

- **Duration:** 23 min
- **Started:** 2026-10-13T04:21:16Z
- **Completed:** 2026-10-13T04:44:46Z
- **Tasks:** 2 (TDD, RED + GREEN each)
- **Files modified:** 7

## Accomplishments

- Warm restart: `mi.reconcileCobolRepos(map[string]struct{}{prefix: {}})` runs at the end of `ReconcileRepoCtx`, after the resolve and derived tails, on every route (ID-07, T-04-61).
- Cross-repository parallel edges: the reconcile's rebound batch goes through `DetectCrossRepoEdgesForReindexes`, so a COBOL CALL bound across repositories carries `cross_repo_calls` on the clean track path as well as on the incremental path.
- D-08: `reresolveAffectedBy` and `planAffectedByStages` log `affected-by: cap exceeded, resolving full affected set` at Info with `file`, `affected`, and `cap`, and no longer slice the set. The `AffectedByReresolveMax` doc describes the escalation.
- Five tests close the remaining D-04 incremental sequences.

## Task Commits

1. **Task 1 RED:** cross-file, cross-repository, and interrupted tests - `829235ee` (test)
2. **Task 1 GREEN:** warm-restart reconcile + cross-repo parallel edges - `9efcad7f` (feat)
3. **Task 2 RED:** affected-by escalation tests - `7d5e82c7` (test)
4. **Task 2 GREEN:** escalate instead of truncating - `ab894074` (feat)

## TDD Gate Compliance

- **Task 1 RED** (`829235ee`): `GOWORK=off go test -count=1 ./internal/indexer/ -run '^TestCobolLifecycle_(CrossFile|CrossRepoIncremental|Interrupted)$'` exited 1 with two failures:
  - `TestCobolLifecycle_Interrupted`: digest mismatch after the warm restart. The clean build has `- [7] ... lifea/copy/custrec.cpy | imports` and `- [13] ... lifea/cobol/sub.cbl::SUBPGM | calls`. After the restart the same edges are still on `unresolved::cobol::copybook::CUSTREC` and `unresolved::cobol::program::SUBPGM`, because the census saw no change and nothing reconciled. The before-restart assertions (dump differs from clean, no wrong binding) already passed.
  - `TestCobolLifecycle_CrossRepoIncremental`: digest mismatch after adding XPGM. The incremental dump has `+ [12] E | crossa/cobol/main.cbl::MAINA/@proc/MAIN-PARA | crossb/cobol/xpgm.cbl::XPGM | cross_repo_calls`, and the clean build lacks it.
  - `TestCobolLifecycle_CrossFile` passed in RED as a guard. 04-08's reconcile tail already re-decides the CALL when another file in the same repository changes.
- **Task 1 GREEN** (`9efcad7f`): all three pass. The plan's verify command passes under `-race` (`TestCobolLifecycle_|Reconcile|Checkout`, 232s).
- **Task 2 RED** (`7d5e82c7`): `-run '^(TestAffectedBy_CapEscalates|TestAffectedBy_DeferredBatchPath_NoFanout|TestCobolLifecycle_OverCap)$'` exited 1:
  - `TestAffectedBy_CapEscalates`: `"[]" should have 1 item(s), but has 0`. No escalation entry was logged.
  - `TestCobolLifecycle_OverCap`: expected `[main.cbl progb.cbl progc.cbl]` and got `[lifea/cobol/main.cbl]`. The affected set was truncated to the cap.
- **Task 2 GREEN** (`ab894074`): all pass. The plan's verify command passes under `-race`, and so do the config package and gofmt.
- `gsd-tools check tdd-red-evidence` cannot parse `go test` output, so the evidence above is copied from the actual failure output.

## Files Created/Modified

- `internal/indexer/multi.go`: the warm-restart reconcile in `ReconcileRepoCtx`.
- `internal/resolver/cobol_references.go`: `DetectCrossRepoEdgesForReindexes(r.graph, rebound)` after `ReindexEdges`.
- `internal/indexer/affected_by.go`: Info escalation without truncation in `reresolveAffectedBy`, plus the `defaultAffectedByMax` and `affectedFilesFor` doc wording.
- `internal/indexer/incremental_batch.go`: Info escalation without truncation in `planAffectedByStages`.
- `internal/config/config.go`: the `AffectedByReresolveMax` doc.
- `internal/indexer/cobol_lifecycle_incremental_test.go`: four new tests. `requireCleanEquivalent` now builds the clean dump over the env's own repositories.
- `internal/indexer/affected_by_e2e_test.go`: `TestAffectedBy_CapEscalates` replaces `TestAffectedBy_CapBoundsFanout`, and the `DeferredBatchPath_NoFanout` filter now uses the new message.

## Decisions Made

See `key-decisions`. In addition:
- Concurrency (edge probe ID-07): the warm-restart reconcile runs inside `coordinateRepositoryTopologyMutation`, so it is serialized with the repository's other mutations. Each `reconcileCobolRepos` builds a fresh master resolver and `ReconcileCobol` holds that resolver's mutex. No concurrent-reconcile test was added, because the plan's behaviors do not call for one.
- Cost: on a COBOL graph, every warm-restart reconcile scans the unresolved-target edges and candidate in-edges of its repository set once. The scan is a no-op (`graphHasLanguage`) without COBOL. This matches the plan's "unconditionally".

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The COBOL reconcile bound cross-repository CALLs without their `cross_repo_calls` parallel edge**
- **Found during:** Task 1 RED (`TestCobolLifecycle_CrossRepoIncremental`)
- **Issue:** The incremental derived pass (`DetectCrossRepoEdgesForFiles`) emitted `cross_repo_calls` for crossa's CALL to crossb's XPGM. The clean `TrackRepoCtx` path never did, in either track order: `ReconcileCobol` set `Edge.CrossRepo` but did not run the parallel-layer settle step that every other cross-repository binder runs. A scratch dump of the 04-07 allowlisted clean build confirmed that no `cross_repo_calls` row existed.
- **Fix:** One call to `DetectCrossRepoEdgesForReindexes(r.graph, rebound)` after `ReindexEdges` in `reconcileCobol`. This mirrors `CrossRepoResolver`'s settle at `cross_repo.go:619`.
- **Files modified:** internal/resolver/cobol_references.go (outside the plan's file list)
- **Verification:** `TestCobolLifecycle_CrossRepoIncremental`, `TestCobolLifecycle_CrossRepoClean` (both track orders), the full resolver package, and the full indexer package all pass.
- **Committed in:** 9efcad7f

**2. [Rule 2 - Missing critical] `TestCobolLifecycle_OverCap` also drives a removed CALL target**
- **Found during:** Task 2 RED
- **Issue:** The planned copybook edit never reaches the affected-by pass. Programs reference a copybook only through the `imports` edge to its file node, so there is no symbol delta. As a result, the planned test passed under truncation and did not exercise the cap.
- **Fix:** The test keeps the planned copybook step. It then renames SUBPGM, which three lifea programs call, and asserts that the `affected_by` catch-up saw all three files. This step fails under truncation and passes with the escalation.
- **Files modified:** internal/indexer/cobol_lifecycle_incremental_test.go
- **Committed in:** 7d5e82c7

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing test strength)
**Impact on plan:** Both were needed for the plan's must-haves to be true. The production diff is one line outside the planned files.

## Issues Encountered

- A gap related to deviation 1 is recorded in `deferred-items.md`: when a cross-repository COBOL CALL unbinds and no file is evicted, its `cross_repo_calls` edge goes stale. One example is a same-named program added to the caller's repository, which makes the CALL ambiguous. No D-04 sequence reaches this case, and the generic cross-repository resolver has the same limitation (Phase 6, Open Question 4).
- `internal/indexer/file_delta.go` is still not gofmt-clean. That is pre-existing and was already noted by 04-08.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 04-16 (D-17 HEAD movement) can rely on the warm-restart reconcile and on full affected-by re-resolution.
- Threat register: T-04-60 is covered by `CrossFile/target_edited`. T-04-61 is covered by the warm-restart reconcile plus `Interrupted`. T-04-35 is accepted, and the Info log carries the counts. T-04-62: the escalation log carries only the repository-relative path and counts.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*

## Self-Check: PASSED
