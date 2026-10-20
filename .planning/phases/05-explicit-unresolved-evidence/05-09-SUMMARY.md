---
phase: 05-explicit-unresolved-evidence
plan: 09
subsystem: cobol-graph
tags: [cobol, finding, ledger, finding_history, resolver, sqlite, shadow, batch, lifecycle]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.FindingLedgerTransitions, FindingHistoryReader.LatestFindingTransitions, FindingHistoryAppender, and the v23 finding_history sidecar (05-06); EXTERNAL_UNRESOLVED findings and the placeholder sweep in reconcileCobolFindings (05-05, 05-08)"
provides:
  - "(*resolver.Resolver).CobolFindingChanges: sorted, cleared-on-read union of reconciled repositories and added/evicted finding and placeholder IDs, under the leaf mutex cobolFindingMu"
  - "internal/indexer/finding_ledger.go: findingLedgerMu, recordFindingLedger, findingTransitionOf, recordCobolFindingLedger"
  - "Ledger tails in reconcileCobolAfterMutation and reconcileCobolRepos (covers incremental, EndBatch, ResetBatch, warm restart, config change, untrack)"
  - "Cold TrackRepoCtx tail: reconcileCobolRepos({repo}) inline, or batchCobolRepos[repo] while resolution is deferred"
  - "Test helpers cobolShadowAdmissions, cobolLedgerFindings, cobolHistory, requireCobolOpenRows, cobolLedgerRepos"
affects: [05-10, 05-11, 05-12]

actuals:
  tokens: 10400
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Every COBOL settle point drains each resolver's change record before the ledger and ledgers own ∪ reconciled ∪ tracked-ID-prefix repositories"
    - "A cold track ends with a master COBOL reconcile on the durable store (the warm-restart tail's shape), deferred into batchCobolRepos under mi.mu while a parallel batch defers resolution"

key-files:
  created:
    - internal/indexer/finding_ledger.go
    - internal/indexer/cobol_findings_lifecycle_test.go
  modified:
    - internal/resolver/resolver.go
    - internal/resolver/cobol_findings.go
    - internal/resolver/cobol_findings_test.go
    - internal/indexer/cobol_reconcile.go
    - internal/indexer/multi.go

key-decisions:
  - "Undesired external findings join the evicted set only when EvictFindingNodesByIDs reports the capability; swept placeholders always join it"
  - "The tracked-prefix predicate is mi.GetIndexer(prefix) != nil; no reusable predicate existed (cobolPolicyLookup's is a closure)"
  - "findingTransitionOf reads prov_start_byte inline with a -1 default; metaToInt and metaInt default to 0, a real byte offset"
  - "TrackRepoCtx checks deferResolve and inserts into batchCobolRepos in one mi.mu section after reconcileEnclosingRepos, then runs the inline tail outside the lock"

patterns-established:
  - "Real-path history tests read FindingHistory (full log) and locate finding IDs through the store, never by recomputing hashes"
  - "A shadow=on test re-applies GORTEX_SHADOW_MAX_FILES after every newCobolLifecycleEnv call and proves the shadow by the admissions delta"

requirements-completed: [GAP-02, GAP-06, GAP-07]

coverage:
  - id: D1
    description: "Resolver change record: reconciled repositories (widened by resolve_from) and added/evicted IDs, accumulated across passes, sorted and cleared by each drain"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_FindingChangesDrain"
        status: pass
    human_judgment: false
  - id: D2
    description: "A track records one open row per finding of both classes on SQLite, on the direct and the shadow path"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryOpensOnTrack"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryShadowPath"
        status: pass
    human_judgment: false
  - id: D3
    description: "Identical re-extraction, direct reconcile passes, and parser re-stamps keep IDs and append no row, from a ledger tail proven to run"
    requirement: GAP-06
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryIdenticalRerun"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryParserRestamp"
        status: pass
    human_judgment: false
  - id: D4
    description: "Fix closes only the fixed finding, a token edit supersedes with the successor ID, deleting the last COBOL file closes all, restoring reopens the same IDs"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryFixCloses"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistorySupersede"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryRestoreReopens"
        status: pass
    human_judgment: false
  - id: D5
    description: "Dependent-repository transitions land at the causing step (incremental and both cold-track orders, shadow off and on); a shadow-path track never leaves a fabricated finding, placeholder, or history row"
    requirement: GAP-02
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryDependentRepo"
        status: pass
    human_judgment: false
  - id: D6
    description: "Parallel-batch tracks ledger at EndBatch, plain-batch tracks inline; interrupted reconciles record nothing until the warm-restart reconcile; concurrent passes append once"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryBatchDeferredTrack"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryInterrupted"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_LedgerSerialized"
        status: pass
    human_judgment: false
  - id: D7
    description: "Every Phase 4/5 COBOL lifecycle, CLI/MCP COBOL, and resolver suite still passes with the cold-track tail on every track"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer/ -run '^TestCobolFindings_|^TestCobolLifecycle_'"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./cmd/gortex/ -run '^TestCobol'"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/resolver/"
        status: pass
    human_judgment: false

duration: 28min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 09: Durable Finding Ledger Summary

**Every COBOL settle point now drains the resolvers' change record and appends finding_history rows on the durable store for every repository the reconcile touched, and a cold track ends with a master COBOL reconcile, so a shadow-path track can no longer persist a not-obtained finding that a resolve_from repository disproves.**

## Performance

- **Duration:** ~28 min
- **Started:** 2026-10-20T18:33:00Z
- **Completed:** 2026-10-20T19:01:00Z
- **Tasks:** 3
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- **Resolver change record.** `reconcileCobolFindings` unions its repository set, the IDs it passed to `AddBatch`, and the IDs it evicted into `cobolFindingRepos/Added/Evicted`. `CobolFindingChanges()` takes only `cobolFindingMu` (a leaf lock), returns sorted slices, and clears them. A nil-scope pass records IDs only.
- **Ledger.** `recordFindingLedger` probes both history capabilities, holds `findingLedgerMu` across read-diff-append, and per repository reads `LatestFindingTransitions`, collects `NodesInScopeSeq(..., KindFinding)`, and appends `FindingLedgerTransitions`. Errors are logged with the repository only. The pass is not gated on COBOL presence.
- **Settle points.** `recordCobolFindingLedger` drains every resolver first, then ledgers base ∪ drained repositories ∪ tracked ID prefixes, and returns the drained IDs (05-11 search sync). Both tails call it:
  - `reconcileCobolAfterMutation`: own prefix, `tracked` nil.
  - `reconcileCobolRepos`: sorted `repos` (or every tracked repository), draining the master plus each base repository's indexer resolver.
- **Cold track.** `TrackRepoCtx` checks `deferResolve` under `mi.mu`. When set, the repository joins `batchCobolRepos` for `EndBatch`/`ResetBatch`. Otherwise it runs `reconcileCobolRepos({repo})` after `reconcileEnclosingRepos`.
- **Real-path proof.** 11 `TestCobolFindings_*` tests, including the shadow path, which failed before the tail (fabricated `EXTERNAL_UNRESOLVED` finding and placeholder, CALL left on the placeholder).

## Task Commits

1. **Task 1: Resolver change record** - `b2ac5713` (test, RED: build failure on the missing accessor), `5fab4ad6` (feat, GREEN)
2. **Task 2: Durable ledger at every COBOL settle point** - `6935653c` (test, RED: no rows on every test; the shadow=on fdepa-then-fdepb order showed the fabricated finding), `9c052308` (feat, GREEN)
3. **Task 3: Batch deferral, shadow path, interruption, parser re-stamp** - `a026382c` (test)

**Plan metadata:** the docs(05-09) commits that add this file and update STATE/ROADMAP/REQUIREMENTS.

## Files Created/Modified

- `internal/resolver/resolver.go` - change-record fields and `cobolFindingMu`
- `internal/resolver/cobol_findings.go` - record write at the end of `reconcileCobolFindings`; `CobolFindingChanges`
- `internal/resolver/cobol_findings_test.go` - `TestCobolReconcile_FindingChangesDrain`
- `internal/indexer/finding_ledger.go` - ledger mutex, pass, transition mapping, COBOL settle-point ledger
- `internal/indexer/cobol_reconcile.go` - ledger calls in both tails
- `internal/indexer/multi.go` - cold-track tail and deferral in `TrackRepoCtx`; `batchCobolRepos` comment
- `internal/indexer/cobol_findings_lifecycle_test.go` - the 11 real-path tests and helpers

## Decisions Made

- Undesired external findings count as evicted only when the store reports the evicter capability, so 05-11 never deletes search rows for findings still present.
- No tracked-prefix predicate existed, so `reconcileCobolRepos` uses `mi.GetIndexer(prefix) != nil` inline.
- `findingTransitionOf` reads `prov_start_byte` with a -1 default inline, with a `// note:` naming the two 0-default helpers.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test bug] Identical-rerun tail proof assumed the path is always listed**
- **Found during:** Task 2 (verify run)
- **Issue:** The plan's behavior required each identical reindex to observe catch-up kind `cobol` with a path ending in `cobol/main.cbl`. The first identical rewrite after a cold track lists it (legacy fallback). The second is re-extracted (`StaleFileCount` 1) but classified inert (`InertFiles` 1, no listed files). `cobolMutationTouched` deliberately runs the tail through its "stale result with no file list" branch, so the tail runs and observes `cobol` with an empty list. No production code is wrong.
- **Fix:** The test calls `IncrementalReindexRepo` directly. On every reindex it requires `StaleFileCount == 1`, exactly one `cobol` catch-up, and any listed path to end in `cobol/main.cbl`. Zero-row and same-ID assertions are unchanged, and the positive control still appends rows.
- **Files modified:** internal/indexer/cobol_findings_lifecycle_test.go
- **Verification:** `TestCobolFindings_HistoryIdenticalRerun` passes under `-race`; stored as workspace memory `mem615be6ab26c41e5f` (gotcha).
- **Committed in:** 9c052308

---

**Total deviations:** 1 auto-fixed (1 test bug)
**Impact on plan:** The tail-ran proof is kept (re-extraction count plus exactly one tail run per reindex). No scope change; only the 7 planned files changed.

## TDD Gate Compliance

Tasks 1 and 2 followed RED then GREEN (`b2ac5713` -> `5fab4ad6`, `6935653c` -> `9c052308`). Before `6935653c` was committed, a temporary no-op `recordFindingLedger` stub (deleted, never committed) showed assertion-level failures for the history tests. Task 3 is `type="auto"` and adds tests only. A temporary mutation that forced the inline tail made its `parallel` subtest fail, then `multi.go` was restored from HEAD.

## Issues Encountered

- `internal/resolver/csharp_verbatim_type_ref_test.go` is still not gofmt-clean at HEAD (pre-existing, already in `deferred-items.md`).

## Verification

- Task 1 verify: test listed, `go test -race ./internal/resolver/ -run Cobol` ok, gofmt clean. Acceptance greps: 1, 1, 0, 1.
- Task 2 verify: 7 tests listed and passing under `-race` (DependentRepo: incremental, shadow=off and shadow=on, both orders each), gofmt clean. Acceptance greps: cobol_reconcile 2 / multi 0; TrackRepoCtx tail 1; `batchCobolRepos[result.RepoPrefix]` 1; `recordFindingLedger(` 0 and 0; `CobolFindingChanges()` line 17 before `recordFindingLedger(` line 43; `RepoPrefixOfID(` 1; `LatestFindingTransitions(` 1 and `.FindingHistory(` 0; the only non-graph `AppendFindingTransitions(` caller is `internal/indexer/finding_ledger.go`; `findingLedgerMu.Lock()` 1.
- Task 3 verify: 4 tests listed, they and `TestCobolLifecycle_ParallelWarmRestart` pass under `-race`, gofmt clean. Acceptance greps 1 and 1.
- Plan verification:
  - `./internal/indexer/ -run '^TestCobolFindings_|^TestCobolLifecycle_'` ok (184 s)
  - `./cmd/gortex/ -run '^TestCobol'` ok (49 s)
  - `./internal/resolver/` ok (56 s)
  - `./internal/graph/... ./internal/resolver ./internal/indexer -run 'Finding|Cobol'` ok
  - Extra regression sweeps: `./internal/indexer/ -run 'Track|Batch|Shadow|Warm|Untrack|MultiIndexer|Multi'` ok; `./internal/mcp/ -run 'Cobol|COBOL|Finding|TrackRepo|IndexHealth'` ok.
  - `git diff --name-only 6a33eb86..HEAD` lists only the 7 planned files.

## User Setup Required

None. Existing stores get one open row per active finding at each repository's next COBOL settle point.

## Next Phase Readiness

- 05-11 can sync search from the `(added, evicted)` that `recordCobolFindingLedger` returns (both callers discard them today). It must also drain the masters `runMasterResolve` discards. Reuse `cobolShadowAdmissions`, `cobolLedgerFindings`, and `cobolHistory`.
- Every cold track outside a batch now pays one master COBOL reconcile (early return without COBOL) plus one ledger read. 05-11's benchmark measures the ledger.

## Self-Check: PASSED

- FOUND: internal/indexer/finding_ledger.go, internal/indexer/cobol_findings_lifecycle_test.go, and the 5 modified files
- FOUND commits: b2ac5713, 5fab4ad6, 6935653c, 9c052308, a026382c
- All three task verify commands, every acceptance criterion, and the plan verification passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
