---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 08
subsystem: indexer
tags: [cobol, incremental, reconcile, restub, lifecycle, graphdump, sqlite]

requires:
  - phase: 04-01
    provides: COBOL lifecycle harness (newCobolLifecycleEnv, cleanCobolDump, requireSameDump) and graphdump
  - phase: 04-06
    provides: Resolver.ReconcileCobol(changed), the one exported reconcile entry (takes the resolve lock)
  - phase: 04-14
    provides: orphaned shared-resource eviction inside the same reconcile pass
  - phase: 04-07
    provides: index.cobol policy lookup on every resolver, including newMasterResolver
provides:
  - CR-01 COBOL-first restub in restubIncomingRefs and restubIncomingRefsFromView
  - cobolMutationTouched, Indexer.reconcileCobolAfterMutation, MultiIndexer.reconcileCobolRepos
  - COBOL reconcile at every incremental mutation tail, catch-up kind cobol
  - TestCobolLifecycle_TailsReached/IdenticalRerun/Edit/Delete/Restore
  - graphdump-volatile/2 allowlist (source_*_fingerprint keys)
affects: [04-15, 04-16, warm-restart reconcile, affected-by escalation]

actuals:
  tokens: 4729
  tasks: 1
  commits: 2
plan_head_before: 2de11b5368e2dc81b686de39d2e0c2025f696589

tech-stack:
  added: []
  patterns:
    - "COBOL in-edges restub to their own cobol_placeholder before eviction; the reconcile pass, never the generic resolver, rebinds them"
    - "Every incremental mutation tail runs the COBOL reconcile after (outside) its resolution branch"

key-files:
  created:
    - internal/indexer/cobol_lifecycle_incremental_test.go
  modified:
    - internal/indexer/cobol_reconcile.go
    - internal/indexer/indexer.go
    - internal/indexer/incremental_batch.go
    - internal/indexer/incremental_watcher_batch.go
    - internal/testutil/graphdump/graphdump.go
    - internal/testutil/graphdump/graphdump_test.go

key-decisions:
  - "COBOL reconcile runs right after each tail's resolution branch and before semantic, derived, and cross-repository passes, matching the clean build where reconcileCobol runs inside ResolveAll"
  - "cobolMutationTouched is checked before the incrementalResolveFilesHook seam, so non-COBOL mutations under that seam do not observe a cobol tail"
  - "graphdump DeterministicVolatile bumped to graphdump-volatile/2 with the seven source_*_fingerprint keys: the incremental delta-probe cache is stamped only by an incremental patch, never by a cold index, so it records index history and not inputs"
  - "An unprefixed (standalone) indexer reconciles with a nil repository set (every edge), mirroring the existing repoPrefix==\"\" scope rule in resolveIncrementalRepoMutationMode"

patterns-established:
  - "COBOL-first restub branch is inline at both restub sites with a // note: naming the other (rule of three not met)"

requirements-completed: [ID-07, ID-09]

coverage:
  - id: D1
    description: "A COBOL save and a COBOL delete through IncrementalReindexRepo each reach the cobol reconcile tail exactly once"
    requirement: ID-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_TailsReached"
        status: pass
    human_judgment: false
  - id: D2
    description: "An unchanged incremental reindex, run twice, and a repeated reconcile leave the clean-build dump digest unchanged"
    requirement: ID-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_IdenticalRerun"
        status: pass
    human_judgment: false
  - id: D3
    description: "Editing the bound CALL target and adding a CALL converges to the clean build; no row carries restub_prev_"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_Edit"
        status: pass
    human_judgment: false
  - id: D4
    description: "Deleting a copybook returns its COPY edge to the placeholder with reason missing; deleting the last reference evicts the shared table; both equal the clean build"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_Delete"
        status: pass
    human_judgment: false
  - id: D5
    description: "Restoring the original bytes reproduces the original clean-build rows byte for byte"
    requirement: ID-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_incremental_test.go#TestCobolLifecycle_Restore"
        status: pass
    human_judgment: false
  - id: D6
    description: "Pre-existing Restub and Incremental indexer tests, the full indexer/resolver/graphdump packages, and cmd/gortex COBOL tests still pass"
    verification:
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestCobolLifecycle_|Restub|Incremental'"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -count=1 -timeout 30m ./internal/indexer/ ./internal/resolver/ ./internal/testutil/graphdump/"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -count=1 ./cmd/gortex/ -run 'Cobol|COBOL'"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 08: COBOL Incremental Convergence (CR-01 Restub and Reconcile Tail) Summary

**COBOL in-edges now restub to their own `cobol_placeholder` before eviction, and every incremental mutation tail runs `ReconcileCobol` outside its resolution branch. As a result, COBOL edits, deletes, restores, and identical reruns produce the same dump as a clean build of the same files.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-10-13T04:05:01Z
- **Completed:** 2026-10-13T04:18:51Z
- **Tasks:** 1 (tracer, TDD: RED + GREEN)
- **Files modified:** 7 (1 created, 6 modified)

## Tail spike (for 04-15)

The spike was `TestCobolLifecycle_TailsReached`, with `incrementalCatchupHook` recording kinds on the `lifea` indexer. Both mutations go through `MultiIndexer.IncrementalReindexRepo`, which leads to `incrementalReindexRepoRawMode` and then `resolveIncrementalRepoMutationMode`. Both the save and the delete arrived with `needed=true`, so neither was skipped by a receipt that was not `ResolutionRelevant` in this corpus.

- Before this plan: a `.cbl` save gave `[resolve dataflow ref_facts cross_repo derived]`, and so did a `.cpy` delete. No COBOL reconcile ran at all.
- After this plan, both give `[resolve dataflow ref_facts cobol cross_repo derived]`, with `cobol` exactly once.
- Paths: `result.DerivedInvalidation.Files` carries repository-prefixed graph paths (`lifea/cobol/main.cbl`). `ExtractorLangForFile` classifies them by extension. A delete also sets `DeletedFileCount`.
- Fixed in the spike and not reached by `IncrementalReindexRepo`: the standalone tails `Indexer.incrementalWatcherPaths` and `Indexer.incrementalEvictWatcherPath`. `MultiIndexer.incrementalEvictRepoRaw` reaches the reconcile through `resolveIncrementalRepoMutation` when `DeletedFileCount > 0`. The warm-restart tail is still 04-15's job.

## Accomplishments

- **CR-01 restub.** `restubIncomingRefs` also fetches in-edges of COBOL `KindFile` nodes (copybooks). Both restub sites now handle COBOL first. An in-edge whose source survives and whose Meta `cobol_placeholder` is non-empty gets `To` set back to that placeholder (skipped when already equal) and is never handed to the generic restub. This holds for any edge kind and any node kind. The branch does not call `StashRestubProvenance` and does not change Origin. Only referenceable nodes fall through to the existing generic restub.
- **Reconcile tail.** New `cobol_reconcile.go`:
  - `cobolMutationTouched` is conservative. It fires on any COBOL path in the derived plan or the receipt's changed, unresolved, or definition files; on any deleted file; or on a stale result with no file list.
  - `Indexer.reconcileCobolAfterMutation` observes `cobol` with the touched paths and returns early under `incrementalResolveFilesHook`. Otherwise it calls `idx.resolver.ReconcileCobol` scoped to the indexer's repository.
  - `MultiIndexer.reconcileCobolRepos` builds `newMasterResolver(false)`, so every tracked repository's candidates and policy are visible, and calls `ReconcileCobol`.
- **Wiring.** The reconcile runs in all three tails, after the resolution branch and outside the `needed` condition:
  - `incrementalWatcherPaths` and `incrementalEvictWatcherPath` call `reconcileCobolAfterMutation`.
  - `resolveIncrementalRepoMutationMode` runs it after the master-resolve branch and before semantic and cross-repository work, and honours the same hook seam.
- **Proven sequences.** Identical rerun (twice, plus a double reconcile), edit, delete (copybook placeholder `missing`; the last reference to `lifeb/db2::_unqualified/ORDERS` deleted, table node evicted by 04-14's resource GC), and restore (rows equal to the original clean dump). Every row is free of `restub_prev_`.

## Task Commits

1. **Task 1 RED: failing COBOL incremental lifecycle tests** - `8533c30d` (test)
2. **Task 1 GREEN: CR-01 restub, reconcile tail, volatile allowlist v2** - `09d9df13` (feat)

REFACTOR: none needed.

## TDD Gate Compliance

- **RED** (`8533c30d`): `GOWORK=off go test -count=1 ./internal/indexer/ -run '^TestCobolLifecycle_'` exited 1. Four target tests failed on assertions:
  - `TailsReached`: `cobol` count expected 1, actual 0, for both the save and the delete.
  - `Edit`: digest mismatch. `main.cbl`'s `CALL 'SUBPGM'` and `COPY CUSTREC` were left on `unresolved::cobol::...` placeholders after re-extraction.
  - `Delete`: digest mismatch. The bound `COPY CUSTREC` edge was evicted together with the copybook instead of returning to its placeholder (`- [7] ... unresolved::cobol::copybook::CUSTREC ... "cobol_unresolved_reason":"missing"`).
  - `Restore`: digest mismatch, same rows as `Edit`.

  `IdenticalRerun` passed in RED as a guard, because an unchanged rerun already converged. The RED commit carried a no-op `reconcileCobolRepos` stub so the test file compiled (deviation 1). `gsd-tools check tdd-red-evidence` cannot parse `go test` output, so this evidence is copied from the actual failure output.
- **GREEN** (`09d9df13`): all five tests pass. The plan's verify command passes under `-race`.

## Files Created/Modified

- `internal/indexer/cobol_lifecycle_incremental_test.go` (new): the five sequence tests, plus `write`/`remove`/`reindex`/`requireCleanEquivalent` env helpers over a private copy of the 04-01 corpus.
- `internal/indexer/cobol_reconcile.go`: `cobolMutationPaths`, `cobolMutationTouched`, `Indexer.reconcileCobolAfterMutation`, and `MultiIndexer.reconcileCobolRepos`.
- `internal/indexer/indexer.go`: CR-01 branch and doc paragraph in `restubIncomingRefs`.
- `internal/indexer/incremental_batch.go`: CR-01 branch in `restubIncomingRefsFromView`.
- `internal/indexer/incremental_watcher_batch.go`: the reconcile at the three mutation tails.
- `internal/testutil/graphdump/graphdump.go`: `DeterministicVolatile` v2 with the fingerprint keys and their justification.
- `internal/testutil/graphdump/graphdump_test.go`: version assertion updated to `graphdump-volatile/2`.

## Decisions Made

See `key-decisions` in the frontmatter. Two of them matter most for 04-15:

- The reconcile runs before the semantic, derived, and cross-repository passes, so they see final COBOL bindings. This mirrors the clean build.
- The hook seam is checked after `cobolMutationTouched`, so the planned interrupted test needs a COBOL mutation in order to observe `cobol`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] RED compile stub for `reconcileCobolRepos`**
- **Found during:** Task 1 RED
- **Issue:** `TestCobolLifecycle_IdenticalRerun` calls `mi.reconcileCobolRepos`, which did not exist yet, so the RED file could not compile.
- **Fix:** Added a no-op stub in `cobol_reconcile.go` for the RED commit. GREEN replaced it.
- **Files modified:** internal/indexer/cobol_reconcile.go
- **Committed in:** 8533c30d

**2. [Rule 3 - Blocking] graphdump volatile allowlist v2 (files outside the plan list)**
- **Found during:** Task 1 GREEN
- **Issue:** After CR-01 and the reconcile tail, every edge row converged. Only the file-node rows of incrementally re-extracted files still differed: they carry seven `source_*_fingerprint` Meta keys (`source_core`, `semantic`, `metadata`, and `derived_{decl,import,runtime,artifact}`). These are the incremental delta-probe cache. `prepareFileDelta` documents that cold indexing deliberately does not stamp them, so no clean build can carry them, and Restore could never be byte-identical with them in.
- **Fix:** Added the seven keys to `graphdump.DeterministicVolatile` with a justification and bumped its version to `graphdump-volatile/2`. This is the 04-01 rule exactly: a harness diff proved the keys volatile. The version assertion in `graphdump_test.go` was updated to match. No production indexing behaviour changed.
- **Files modified:** internal/testutil/graphdump/graphdump.go, internal/testutil/graphdump/graphdump_test.go
- **Verification:** graphdump package tests pass, and all COBOL lifecycle tests pass, including 04-01's `CleanDeterminism` and the cross-repository clean tests.
- **Committed in:** 09d9df13

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both were needed to reach the plan's success criteria. No production scope creep. The plan's verification line ("only the 5 planned files") is exceeded by the two graphdump files, recorded here.

## Issues Encountered

- `gofmt -l internal/indexer/` lists `internal/indexer/file_delta.go`. That file is untouched by this plan and was already unformatted, so it is out of scope. The plan's gofmt check over its own five files is clean.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 04-15 can build the warm-restart reconcile, the D-08 affected-by escalation, and the cross-file, cross-repository, interrupted, and over-cap sequences on top of `reconcileCobolAfterMutation`, `reconcileCobolRepos`, and the `cobol` catch-up kind.
- Plan-level threat mitigations:
  - T-04-32: COBOL edges never restub to a bare name.
  - T-04-33: no stash keys on COBOL edges, and every harness test asserts none appear.
  - T-04-34: the reconcile runs outside the `needed` branch.
  - T-04-36: the reconcile adds no logging.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*

## Self-Check: PASSED
