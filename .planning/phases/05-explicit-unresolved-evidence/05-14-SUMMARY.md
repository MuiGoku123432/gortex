---
phase: 05-explicit-unresolved-evidence
plan: 14
subsystem: cobol-findings
tags: [cobol, finding, ledger, finding_history, concurrency, reach, topology-writer, gap-closure, d-23, d-24, wr-01, wr-02]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "finding ledger and change-record drain (05-06, 05-09, 05-11), cobolReconcileMu on every resolver (05-09), name-stable finding identity (05-13)"
provides:
  - "(*MultiIndexer).recordCobolSettleLedger: the one MultiIndexer COBOL settle ledger, holding the reach topology writer (taken unless the caller holds it) and cobolReconcileMu across drain, read, diff, append"
  - "reconcileCobolRepos takes the writer (finish(false)); reconcileCobolReposTopologyHeld for writer-held callers"
  - "TrackRepoCtx cold tail branches on activeRepositoryTopologyBatch(ctx, mi): held variant inside a topology batch, taking variant outside"
  - "recordCobolFindingLedger gains reconcileMu *sync.Mutex; lock order writer -> cobolReconcileMu -> findingLedgerMu"
  - "reach.TopologyWriterActive; Resolver.CobolReconcileLock; MultiIndexer.cobolSettleHook test seam (stages committed, ledger)"
  - "runMasterResolveHookedContext ledgers its drain at its own step (D-24); runMasterResolveTopologyHeld; resolveDeferredMutationsMode behind the unchanged 4-argument resolveDeferredMutations; DeferredPassesRun.topologyHeld; runPreEnrichResolve behind the unchanged RunPreEnrichResolve"
  - "TestCobolFindings_LedgerExcludesTornMutation (track_tail, master_drain) and TestCobolFindings_MasterDrainClosesBeforeLaterEdit, each with a recorded failing mutation"
affects: [05-16, 05-17]

actuals:
  tokens: 8850
  tasks: 3
  commits: 7

tech-stack:
  added: []
  patterns:
    - "COBOL settle point = reconcile, drain, ledger, search sync under one exclusion (recordCobolSettleLedger)"
    - "TopologyHeld convention for every reconcile and master-resolve path: writer state is an explicit bool, never re-acquired"
    - "Deterministic interleaving tests park goroutines on a nil-in-production hook and decide release order from one snapshot read; no sleeps or timed waits"

key-files:
  created:
    - internal/indexer/cobol_findings_ledger_exclusion_test.go
  modified:
    - internal/indexer/cobol_reconcile.go
    - internal/indexer/finding_ledger.go
    - internal/indexer/multi.go
    - internal/indexer/incremental_watcher_batch.go
    - internal/indexer/repository_untrack.go
    - internal/reach/reach.go
    - internal/resolver/cobol_references.go
    - internal/indexer/cobol_findings_lifecycle_test.go

key-decisions:
  - "D-23 shipped: every MultiIndexer COBOL ledger runs through recordCobolSettleLedger under the reach topology writer and cobolReconcileMu; lock order writer -> cobolReconcileMu -> findingLedgerMu everywhere"
  - "Writer-held sites (incremental lanes, warm snapshot, untrack purge, TrackRepoCtx inside RunRepositoryTopologyBatch, incremental no-frontier fallback, IndexAll pre-enrich resolve and cold deferred tail) use TopologyHeld variants; EndBatch, ResetBatch, RefreshRepoConfigs, TrackRepoCtx outside a batch, daemon warmup, and public deferred paths take the writer"
  - "D-24 shipped: the master-resolve drain ledgers at its own step; resolveDeferredMutations keeps its 4-argument signature so deferred_receipt_test.go and deferred_receipt_import_test.go are unchanged"
  - "Cold-track commit-to-tail gap kept and documented as benign (new repository only; every ledger read of it follows a reconcile of it); one residual stale-record corner recorded under T-05-73 (accepted), not treated as a Rule 4 stop"
  - "GAP-06 and GAP-07 are addressed here but not marked complete: the orchestrator reserves that for phase verification"

patterns-established:
  - "A settle point that claims a held writer on a base-topology store while none is active logs a repository-only warning and continues (D-07 best effort)"

requirements-completed: []  # Plan requirements are [GAP-06, GAP-07]; left unmarked per orchestrator instruction (phase verification decides).

coverage:
  - id: D1
    description: "Every MultiIndexer COBOL ledger holds the reach topology writer and cobolReconcileMu; writer-held paths, the repository topology batch included, use the held variant and do not self-deadlock"
    requirement: GAP-06
    verification:
      - kind: integration
        ref: "Task 1 verify selection: ^TestNestedTrackedRoot_|^TestRunRepositoryTopologyBatch|9 COBOL call-site tests (-race, -timeout 300s)"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 60m ./internal/indexer/"
        status: pass
    human_judgment: false
  - id: D2
    description: "Deterministic interleaving test: a settle-point ledger can no longer read an incremental lane between its file commit and its reconcile (track_tail and master_drain), no false active -> closed -> active pair"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_ledger_exclusion_test.go#TestCobolFindings_LedgerExcludesTornMutation (-race -count=3)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The fallback master records its close at its own step; the next settle point and an unchanged second pass append zero rows; a later same-paragraph edit only opens NEWPGM, never supersedes (WR-02)"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_SearchAfterFallbackMasterResolve"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_MasterDrainClosesBeforeLaterEdit"
        status: pass
    human_judgment: false
  - id: D4
    description: "Deferred-receipt callers compile unchanged and the master-path topologyHeld plumbing holds for IndexAll, daemon warmup, and deferred catch-up"
    verification:
      - kind: integration
        ref: "Task 3 verify selection incl. ^TestResolveDeferredMutations|^TestFinishTailResult|^TestMutationReceiptEvict"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./cmd/gortex/ -run '^TestCobol'"
        status: pass
    human_judgment: false
  - id: D5
    description: "Cold-track commit-to-tail gap documented as benign (code-reading argument)"
    verification: []
    human_judgment: true
    rationale: "A read-set argument over recordCobolFindingLedger and the change-record producers, not an executable check; the verifier should confirm the reasoning and the residual stale-record corner recorded below."

duration: 62min
completed: 2026-10-23
status: complete
---

# Phase 5 Plan 14: Settled-State Finding Ledger (D-23, D-24) Summary

**Every COBOL finding ledger now runs under the reach topology writer plus the shared reconcile lock through one helper, so it can't read a lane between its file commit and its reconcile. Fallback master resolves now record their drain at their own step, so whether a finding is closed or superseded no longer depends on timing. Both changes are backed by deterministic interleaving tests and failing mutation runs.**

Execution resumed after an executor API disconnect. The first executor committed all three tasks (16:11Z-16:23Z) and lost its connection before verification or SUMMARY. This close-out (19:56Z-20:35Z) audited the five commits, ran every task verify, the plan-level verification, and the mutation proofs, found no gaps, and added no code commits.

## Performance

- **Duration:** ~62 min active (first executor ~26 min, 15:57Z-16:23Z; close-out ~36 min, 19:56Z-20:35Z)
- **Started:** 2026-10-23T15:57Z (after the 05-13 completion commit)
- **Completed:** 2026-10-23T20:35Z
- **Tasks:** 3 of 3
- **Files modified:** 9 (8 modified, 1 created), the nine planned files exactly

## Accomplishments

- `recordCobolSettleLedger` is the only MultiIndexer ledger entry. It takes the reach topology writer unless its caller holds it, and through `recordCobolFindingLedger` it holds `cobolReconcileMu` from before the drain until `recordFindingLedger` returns.
- `reconcileCobolRepos` takes the writer and `reconcileCobolReposTopologyHeld` serves writer-held callers. The TrackRepoCtx tail branches on `activeRepositoryTopologyBatch(ctx, mi)`, so a track inside `RunRepositoryTopologyBatch` cannot self-deadlock.
- `runMasterResolveHookedContext` ledgers its drain (D-24) for the incremental no-frontier fallback, the deferred full fallback, and the pre-enrich resolve, with writer state threaded explicitly.
- Deterministic and sleep-free proof: `TestCobolFindings_LedgerExcludesTornMutation` (track_tail, master_drain) and `TestCobolFindings_MasterDrainClosesBeforeLaterEdit`. All three mutations fail as intended, and the code was restored to an empty diff.

## Task Commits

1. **Task 1 (tracer): every MultiIndexer ledger under the topology writer and reconcile lock** - `25e22584` (fix)
2. **Task 2: writer probe, reconcile lock accessor, settle seam, interleaving test** - `6a92ea55` (feat), `4e1465d8` (test)
3. **Task 3: master-resolve drain ledgers at its own step** - `9f08c856` (test), `3990ce26` (fix)

**Plan metadata:** SUMMARY commit and STATE/ROADMAP commit follow this file. No fix commits were needed in the close-out.

## Files Created/Modified

- `internal/indexer/cobol_reconcile.go` - `reconcileCobolRepos` (takes the writer), `reconcileCobolReposTopologyHeld`, `recordCobolSettleLedger` (writer, held-writer warning, `ledger` hook, `cobolReconcileMu`); the Indexer tail passes `idx.resolver.CobolReconcileLock()`
- `internal/indexer/finding_ledger.go` - `recordCobolFindingLedger(store, base, tracked, reconcileMu, logger, resolvers...)` holds reconcileMu across drain, read, diff, and append
- `internal/indexer/multi.go` - `cobolSettleHook` seam (`committed` at :2827, `ledger` in cobol_reconcile.go:176); batch-aware TrackRepoCtx tail; held variant at the warm snapshot; master drain ledger; `runMasterResolveTopologyHeld`; `resolveDeferredMutationsMode`; `DeferredPassesRun.topologyHeld`; `runPreEnrichResolve`
- `internal/indexer/incremental_watcher_batch.go` - COBOL tail on the held variant; no-frontier fallback on `runMasterResolveTopologyHeld`
- `internal/indexer/repository_untrack.go` - purge reconcile on the held variant
- `internal/reach/reach.go` - `TopologyWriterActive()`
- `internal/resolver/cobol_references.go` - `(*Resolver).CobolReconcileLock()`
- `internal/indexer/cobol_findings_ledger_exclusion_test.go` (new) - the interleaving protocol and its two subtests
- `internal/indexer/cobol_findings_lifecycle_test.go` - `SearchAfterFallbackMasterResolve` updated, `MasterDrainClosesBeforeLaterEdit` added

## Call-Site Table (audited at HEAD 3990ce26)

Writer means the process-global reach topology writer (`reach.BeginTopologyMutation`). "Held" variants never take it, and "taking" variants take it once and release it with `finish(false)`.

### Reconcile and ledger sites

| # | Call site | Calls | Writer at the call (where taken) | Variant | Covering test |
|---|-----------|-------|----------------------------------|---------|---------------|
| 1 | TrackRepoCtx tail, inside a topology batch (multi.go:3013) | `reconcileCobolReposTopologyHeld` | Yes. `RunRepositoryTopologyBatch` holds it for the whole callback (repository_topology_batch.go:69) | held | `TestNestedTrackedRoot_TrackInsideTopologyBatchCompletes`, `TestNestedTrackedRoot_DeclaredRootsPruneWarmupOrder`, `TestRunRepositoryTopologyBatchAllowsParallelTrackAndBlocksReach` |
| 2 | TrackRepoCtx tail, outside a batch (multi.go:3015) | `reconcileCobolRepos` | No. The closure's writer (multi.go:3188, `beginRepositoryTopologyMutation`) is released when the closure returns | taking (cobol_reconcile.go:103) | `TestCobolFindings_HistoryOpensOnTrack`, `LedgerExcludesTornMutation/track_tail` (the same call) |
| 3 | `EndBatch` (multi.go:1668) | `reconcileCobolRepos` | No. Only batchMutationGate is held. The daemon calls it after `RunRepositoryTopologyBatch` returns (daemon_state.go:1160 vs :638) | taking | `TestCobolFindings_HistoryBatchDeferredTrack` |
| 4 | `ResetBatch` (multi.go:1705) | `reconcileCobolRepos` | No. Only batchMutationGate is held (daemon_state.go:1152, :1156) | taking | `TestCobolLifecycle_ParallelWarmRestart` |
| 5 | `RefreshRepoConfigs` (multi.go:2650) | `reconcileCobolRepos` | No. The caller is `CheckoutLifecycle.ApplyReload` (checkout_lifecycle.go:1151), which holds only its session fan-out batch | taking | `TestCobolLifecycle_ConfigReloadRedecides` |
| 6 | Warm snapshot `ReconcileRepoCtx`, non-deferred (multi.go:3546) | `reconcileCobolReposTopologyHeld` | Yes. multi.go:3350 (`beginRepositoryTopologyMutation`, or the batch's writer) | held | `TestCobolFindings_HistorySurvivesRestart` (whole-package run); the deferred branch is covered by `TestCobolLifecycle_ParallelWarmRestart` |
| 7 | `resolveIncrementalRepoMutationMode` COBOL tail (incremental_watcher_batch.go:440) | `reconcileCobolReposTopologyHeld` | Yes. Every caller holds it: `incrementalEvictRepoRaw` (writer multi.go:2761, call :2773), `incrementalReindexRepoRawMode` (writer :2818, call :2835), warm closure (writer :3350, call :3519) | held | `TestCobolFindings_HistoryFixCloses`, `TestCobolLifecycle_ReconcileSerialized`, lane B of `LedgerExcludesTornMutation` |
| 8 | `purgeRepositoryPayload` deferred reconcile (repository_untrack.go:248) | `reconcileCobolReposTopologyHeld` | Yes. `untrackRepoTeardown` takes it at repository_untrack.go:123, and the `refresh` closure (:182) that reaches the purge runs while it is held | held | `TestCobolFindings_HistoryPurgedOnUntrack`; `TestCobolLifecycle_CrossRepoUntrack` (whole-package run) |
| 9 | `reconcileCobolReposTopologyHeld` -> `recordCobolSettleLedger(base, true, ...)` (cobol_reconcile.go:133) | settle ledger | Inherited from rows 1-8 | held | all of the above |
| 10 | `recordCobolSettleLedger` -> `recordCobolFindingLedger(..., &mi.cobolReconcileMu, ...)` (cobol_reconcile.go:179) | ledger | Taken at :169 when `topologyHeld` is false; otherwise inherited, with a warning if absent on a base-topology store (:171) | n/a | all settle points |
| 11 | Standalone Indexer tail `reconcileCobolAfterMutation` -> `recordCobolFindingLedger(..., idx.resolver.CobolReconcileLock(), ...)` (cobol_reconcile.go:86) | ledger | Yes. `incrementalWatcherPaths` holds it (incremental_watcher_batch.go:241 / :296) | n/a (no MultiIndexer helper) | whole-package run |

### Master-resolve sites (`runMasterResolveHookedContext` -> `recordCobolSettleLedger(nil, topologyHeld, master)`, multi.go:1070)

| # | Entry | topologyHeld | Writer at the call (where taken) | Covering test |
|---|-------|--------------|----------------------------------|---------------|
| 12 | `runMasterResolve` (multi.go:1013) | false | No production caller remains; tests only | `SearchAfterFallbackMasterResolve`, `MasterDrainClosesBeforeLaterEdit`, `LedgerExcludesTornMutation/master_drain` |
| 13 | `runMasterResolveTopologyHeld` from the incremental no-frontier fallback (incremental_watcher_batch.go:413) | true | Yes. The lane writer, as in row 7 | whole-package run |
| 14 | `resolveDeferredMutationsMode` full fallback (multi.go:925) via `FinishTailResult` (multi.go:834) | `r.topologyHeld` | true only for `finishColdDeferredPasses` (multi.go:617-620, called at :2466 inside IndexAll's writer from :2328). false for `BeginDeferredPasses` (multi.go:718; daemon_state.go:909, :999) and `RunDeferredPassesAllResult` (multi.go:609; daemon_state.go:1001), which are writer-free | `TestFinishTailResult*`, whole-package run (IndexAll) |
| 15 | `resolveDeferredMutations` 4-argument delegate (multi.go:907) | false | Its only callers are the 14 direct tests in deferred_receipt_test.go and deferred_receipt_import_test.go (both unchanged), and none of them holds the writer | `TestResolveDeferredMutations*` |
| 16 | `runPreEnrichResolve` from IndexAll (multi.go:2459) | true | Yes. multi.go:2328 (held through :2466) | whole-package run |
| 17 | `RunPreEnrichResolve` (multi.go:1182) | false | No. daemon_state.go:949 runs after the topology batch has joined | `./cmd/gortex -run '^TestCobol'`, whole-package run |

Two other master paths have no ledger site. `runMasterResolveFiles` (`ResolveFilesAndIncoming`, also reached through `RunPreEnrichResolveFiles` at daemon_state.go:947) and `runMasterResolveNames` (`ResolveIncomingForNames`) run no COBOL reconcile, because `reconcileCobol` runs only inside `ResolveAllContext` (resolver.go:746, :1691). Their change records stay empty.

Reach read sections: `beginTopologyRead` and `beginTopologyReadContext` have no caller outside package `reach`. No caller of a writer-free master path can therefore hold a read section across the ledger's writer acquisition.

## Cold-Track Gap

`trackRepoSourceCtx`'s closure commits the new repository R under the writer (multi.go:3188) and releases it on return. `TrackRepoCtx` then runs `reconcileEnclosingRepos` and only then the COBOL tail, which retakes the writer (row 2). The gap stays, for the reasons below, confirmed against `recordCobolFindingLedger`'s read set (finding_ledger.go:162-197): base, the repositories each drained resolver reports, and the tracked prefixes of the drained added and evicted IDs.

1. **The commit evicts no ledgered finding.** The closure installs only an untracked repository. It returns early for a tracked prefix (:3144, :3196) or a same-identity path (:3148-3152, :3198-3203). R therefore has no prior finding rows: untrack purges history (`TestCobolFindings_HistoryPurgedOnUntrack`). A read in the gap cannot produce the D-23 tear, which is a false active -> closed for a finding the commit evicted and the reconcile re-adds.
2. **Every read of R in the gap follows a reconcile of R.**
   - If R is in a settle point's base, that settle point's master reconciles R inside the same writer section before its ledger reads it (cobol_reconcile.go:122 then :133).
   - If R is not in the base, only a drained change record can name R. Records come from `reconcileCobolFindings`.
     - The settle point's own master reconciles inside the same writer section, after R's commit.
     - Per-repository resolvers are drained only when their repository is in the base (cobol_reconcile.go:128-131). R's own resolver therefore drains only with R in the base, which is the case above.
     - Another repository's resolver can name R's IDs only through a reconcile that saw R's nodes, which exist only after R's commit.
   - The tail's later pass then sees the same evidence and appends zero rows. Zero-row repeat: `SearchAfterFallbackMasterResolve` runs two unchanged `reconcileCobolRepos` passes after the master drain and appends nothing.
3. **Why the tail is not moved into the closure.** Inside the closure the tail would run under R's lane and before `reconcileEnclosingRepos`. That would reorder the enclosing-repository eviction and move the `deferResolve` handoff to `batchCobolRepos` under the lane. The change would be larger than the gap warrants.

**Residual corner (recorded, accepted under T-05-73, not a Rule 4 stop).** Part 2 assumes no per-repository change record outlives its owner's next settle point. One construction breaks that assumption:

- Repository X cold-tracks on the direct, non-shadow path. Its per-repository `ResolveAll` (indexer.go:4021) has nil scope and reconciles every COBOL edge in the shared store, so X's record can name an already-tracked R.
- Inside X's short commit-to-tail window, R is untracked and re-tracked.
- X's tail then drains that record during R's re-track gap and ledgers R before R's master tail.

What that read sees is R's own resolver's reconciled commit state, not a torn lane. R has no prior rows after the untrack purge, so the worst case is real transitions between two reconciled states (R-only decisions, then the master's cross-repository refinement). It cannot produce a false close and reopen of unchanged ledgered evidence. Under a parallel warmup batch (deferResolve) the per-repository resolve is skipped (indexer.go:4005), so the window exists only for non-batched direct cold tracks.

## Decisions Made

- No code decisions were made in the close-out. It followed the committed design: the plan's D-23 and D-24 choices, explicit TopologyHeld variants, and the documented cold-track gap.
- GAP-06 and GAP-07 are left unmarked in REQUIREMENTS.md per the orchestrator. `requirements-completed` is empty on purpose.

## Audit of the Prior Executor's Commits

| Check | Result |
|-------|--------|
| TrackRepoCtx tail branches on `activeRepositoryTopologyBatch(ctx, mi)`: held variant inside a batch, `reconcileCobolRepos` outside | Present (multi.go:3012-3016) |
| `resolveDeferredMutations` keeps its 4-argument signature; `resolveDeferredMutationsMode` carries `topologyHeld` | Present (multi.go:907-908, :914) |
| deferred_receipt_test.go and deferred_receipt_import_test.go unchanged | `git diff --name-only eb9ca9f2 HEAD` lists neither |
| Interleaving test reads `reach.TopologyWriterActive()` once, right after `aParked` and before goroutine B starts | Yes, in order: `held := reach.TopologyWriterActive()` (exclusion_test.go:91) precedes `go func() {` that runs `IncrementalReindexRepo("fdepb", nil)` (:95); grep count 1 |
| No slash-form placeholder literal (`/cobol-missing::`) in the new exclusion test | grep count 0 |
| No `time.Sleep` or `time.After` in the exclusion test; no `require` inside goroutines; hook reset in `t.Cleanup` | Confirmed |
| Every Task 1/2/3 action and acceptance grep | All met (counts: cobol_reconcile TopologyHeld 5, iwb TopologyHeld 1, multi TopologyHeld 2, `activeRepositoryTopologyBatch(ctx, mi)` 2, cobol_reconcile `BeginTopologyMutation` 3, settle ledger def 1, `cobolReconcileMu` 4, finding_ledger `reconcileMu` 6, `TopologyWriterActive` def 1, `CobolReconcileLock` def 1, `cobolSettleHook` 4, `recordCobolSettleLedger` in multi 1, `runMasterResolveTopologyHeld` in iwb 1, `resolveDeferredMutations(receipt` def 1, `resolveDeferredMutationsMode(` 3, "no ledger at the master drain" 0) |
| `git diff --name-only` vs base eb9ca9f2 | Exactly the nine planned files |

Nothing was missing, so the close-out added no fix commits.

## Verification Results

| Command | Result |
|---------|--------|
| Task 1 verify (build, vet, 22 tests: `^TestNestedTrackedRoot_`, `^TestRunRepositoryTopologyBatch`, 9 COBOL call-site tests, `-race -timeout 300s`, gofmt) | ok (indexer 33.5 s); no timeout |
| Task 2 verify (`-list` check, `LedgerExcludesTornMutation -race -count=3 -timeout 300s`, reach `Topology`, resolver `Cobol`, gofmt) | ok: track_tail and master_drain PASS 3/3 (14.8 s); reach 3.2 s; resolver 23.2 s |
| Task 3 verify (`-list` for 3 tests, build, vet, 21 tests incl. `^TestResolveDeferredMutations`, `^TestFinishTailResult`, `^TestMutationReceiptEvict`, `-race -timeout 300s`, gofmt) | ok (26.3 s); no timeout |
| `df -h /System/Volumes/Data` before plan verification | 21 GiB free (15 GiB after the runs); above the 10 GiB floor |
| `GOWORK=off go test -race -count=1 -timeout 60m ./internal/indexer/` | ok (1153.7 s); no timeout, no deadlock |
| `GOWORK=off go test -race -count=1 ./internal/reach/ ./internal/resolver/` | ok (3.6 s, 52.1 s) |
| `GOWORK=off go test -race -count=1 -timeout 30m ./cmd/gortex/ -run '^TestCobol'` | ok (52.8 s, 25 tests) |
| `GOWORK=off go build ./...` | ok |

### Mutation Proofs (all restored; `git diff -- internal cmd` empty afterward)

| Mutation | Test | Result |
|----------|------|--------|
| `reconcileCobolRepos` calls the held variant without taking the writer (Task 2; orchestrator step 3) | `LedgerExcludesTornMutation/track_tail` | FAIL 3/3 under `-race`: fdepb gains `fdepb/cobol/b.cbl::@finding/external/c02f93cd...` active -> closed (StartByte 110), then closed -> active (StartByte 138); message "writer held at the settle point: false" |
| `recordCobolSettleLedger` skips the writer when `topologyHeld` is false (Task 3, proof 2) | `LedgerExcludesTornMutation/master_drain` | FAIL 3/3: the same active -> closed -> active pair on fdepb |
| Master drain keeps drain and search sync but skips the ledger (Task 3, proof 1) | `TestCobolFindings_MasterDrainClosesBeforeLaterEdit` | FAIL: no row at the master step, then `finda/...@finding/external/1696cb10...` active -> superseded with successor NEWPGM (`requireNoSuperseded` also fails) |

## Deviations from Plan

None. The plan was executed as written, and the close-out found nothing to fix.

## TDD Gate Compliance

- **Task 2 (tdd=true):** `6a92ea55` (feat) precedes `4e1465d8` (test) within the same second. The test cannot compile without the `cobolSettleHook` seam the feat adds, and Task 1 (`25e22584`) had already shipped the exclusion. A RED run against HEAD was therefore impossible, and the first executor never showed one. The close-out supplies RED as the mutation run in row 1 above (3/3 failures).
- **Task 3 (tdd=true):** `9f08c856` (test) precedes `3990ce26` (fix) within the same second, and no RED run was recorded. Reading the code at `9f08c856`: `MasterDrainClosesBeforeLaterEdit` would fail, and `master_drain` would block at `aParked` because the master emitted no `ledger` stage then. This was inferred, not run. The close-out supplies RED as the mutation runs in rows 2 and 3.

## Issues Encountered

- Execution resumed after an executor API disconnect. The prior executor's commits were audited rather than redone.
- **Observation:** if a regression removed the master-drain ledger, `master_drain` would block on `aParked` until the package `-timeout` instead of failing quickly. The plan forbids timeout channels in this test, so this is by design: the timeout is the deadlock signal. Noted here for the verifier; no change made.
- **Residual cold-track corner:** see "Cold-Track Gap" (accepted under T-05-73).

## Threat Flags

None. No new network, auth, file-access, or schema surface. The new warning names repositories only (T-05-63).

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 05-16 (D-27 placeholder namespace) and 05-17 (docs for D-21..D-27, including the D-23/D-24 history timing and this cold-track note) are ready.
- No blockers.

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-23*

## Self-Check: PASSED

- All nine plan files and this SUMMARY exist on disk.
- Commits 25e22584, 6a92ea55, 4e1465d8, 9f08c856, and 3990ce26 exist in history.
- The source tree is clean after the mutation runs (`git diff -- internal cmd` is empty).
