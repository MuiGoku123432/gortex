---
phase: 05-explicit-unresolved-evidence
plan: 11
subsystem: cobol-graph
tags: [cobol, finding, placeholder, symbol-fts, search, ledger, resolve_from, shadow, master-resolve, benchmark, pitfall-4]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "recordCobolFindingLedger returning the drained added/evicted IDs, (*resolver.Resolver).CobolFindingChanges, the cold TrackRepoCtx tail through reconcileCobolRepos, the finda and fdepa/fdepb fixtures, cobolShadowAdmissions (05-09); EXTERNAL_UNRESOLVED findings (05-08); finding_history and recordFindingLedger (05-06); missing-artifact placeholders (05-05)"
provides:
  - "(*Indexer).syncCobolFindingFTS: deletes evicted IDs from symbol FTS, upserts added IDs whose node exists and passes shouldIndexForSearch (ftsTokensFor tokens, SymbolFTSBatchUpserter)"
  - "(*MultiIndexer).syncCobolFindingFTSByRepo: groups IDs by graph.RepoPrefixOfID and writes each group only through that tracked repository's Indexer"
  - "Search sync at both COBOL settle-point tails (reconcileCobolAfterMutation, reconcileCobolRepos, which the cold TrackRepoCtx tail reaches)"
  - "runMasterResolveHookedContext drains master.CobolFindingChanges() after ResolveAllContext and syncs search (fallback masters no longer discard changes)"
  - "Seven real-path tests (search after incremental, after a fallback master, across a resolve_from dependency incl. both cold-track orders with the shadow off and on; external closes on retrieve; placeholder swept on last reference; history survives restart; history purged on untrack)"
  - "BenchmarkRecordFindingLedger_LargeRepo (FirstPass, NoChangePass) and its recorded baseline"
affects: [05-12, phase-06]

actuals:
  tokens: 4920
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Every place that drains a resolver's COBOL change record also syncs symbol FTS from the drained IDs; the ledger stays at the two 05-09 settle points"
    - "Cross-repository search writes are routed by graph.RepoPrefixOfID to the owning tracked Indexer; empty or untracked prefixes are skipped"
    - "COBOL search tests assert both tiers: the bare identifier (exact-name tier, node table) and 'missing <name>' (ranks the symbol FTS corpus)"

key-files:
  created: []
  modified:
    - internal/indexer/finding_ledger.go
    - internal/indexer/cobol_reconcile.go
    - internal/indexer/multi.go
    - internal/indexer/cobol_findings_lifecycle_test.go

key-decisions:
  - "syncCobolFindingFTS writes only the store's symbol FTS, not idx.search: initialSearchBackend gives an indexer either SymbolSearcherBackend (Add/Remove are no-ops) or the Null backend, so the incremental stage path's idx.search writes have no effect there either"
  - "Delete runs before upsert and an added ID is upserted only while its node exists, so an ID in both drained lists (e.g. a shadow-built node the tail's master evicts) ends absent from search"
  - "runMasterResolveHookedContext drains and syncs whatever ResolveAllContext returns (a reconcile that ran before a cancellation still changed the store) and records no ledger; history for fallback masters is recorded at the next settle point (state-based ledger)"
  - "COBOL search assertions use two queries per name because the bare identifier is answered from the node table and cannot see the FTS index"

patterns-established:
  - "Drain-then-sync: any new resolver entry that reconciles COBOL must drain CobolFindingChanges and call syncCobolFindingFTSByRepo, or its findings and placeholders go stale in search"

requirements-completed: [GAP-05, GAP-06, GAP-07]

coverage:
  - id: D1
    description: "An incremental edit that adds CALL 'NEWPGM' makes the new placeholder and external finding searchable in both tiers; removing the call takes both out of the store and search"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_SearchAfterIncremental"
        status: pass
    human_judgment: false
  - id: D2
    description: "A master ResolveAll (the incremental no-frontier fallback's exact call) removes a stale placeholder and finding from the store and from search with no other settle point; the next reconcileCobolRepos appends the active -> closed row"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_SearchAfterFallbackMasterResolve"
        status: pass
    human_judgment: false
  - id: D3
    description: "Across a resolve_from dependency, adding/removing NOPGM in fdepa closes/reopens fdepb's same finding ID with a ledger row and removes/restores fdepb's placeholder and finding in search at that step; both cold-track orders hold with the shadow off and on (no not-obtained node searchable)"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_CrossRepoDependentLedgerAndSearch (incremental, shadow=off/{fdepb-then-fdepa,fdepa-then-fdepb}, shadow=on/{fdepb-then-fdepa,fdepa-then-fdepb})"
        status: pass
    human_judgment: false
  - id: D4
    description: "Adding the missing copybook closes only the COPY NOCPY external finding (one row) and removes its placeholder; the missing CALL and parse findings get no row"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_ExternalClosesOnRetrieve"
        status: pass
    human_judgment: false
  - id: D5
    description: "Removing the last CALL 'NOPGM' closes its external finding and sweeps finda/cobol-missing::program/NOPGM"
    requirement: GAP-05
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_PlaceholderSweptOnLastReference"
        status: pass
    human_judgment: false
  - id: D6
    description: "Finding history survives a daemon restart plus warm reconcile with no row appended (identity reused), and is purged when the repository is untracked"
    requirement: GAP-06
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistorySurvivesRestart"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_HistoryPurgedOnUntrack"
        status: pass
    human_judgment: false
  - id: D7
    description: "Ledger cost over 100000 findings is measured: FirstPass appends 100000 rows, NoChangePass appends 0"
    verification:
      - kind: other
        ref: "GOWORK=off go test -count=1 -timeout 30m -run '^$' -bench '^BenchmarkRecordFindingLedger_LargeRepo$' -benchtime 1x ./internal/indexer/"
        status: pass
    human_judgment: false

duration: ~38min (~18 min original execution 2026-10-20 + ~19 min close-out 2026-10-21)
completed: 2026-10-21
status: complete
---

# Phase 5 Plan 11: COBOL Finding Search Sync and External Lifecycle Summary

**Symbol FTS now changes in the same step as the resolver. The finding and placeholder IDs that `recordCobolFindingLedger` and every master `ResolveAll` drain are deleted from or upserted into each owning repository's FTS. The changed repository, `resolve_from` dependents, cold tracks with the shadow off and on, and fallback masters all stay correct. Seven real-path tests prove the external-finding lifecycle, and a 100000-finding ledger baseline is recorded.**

## Performance

- **Duration:** about 38 min in total.
  - About 18 min was the original execution on 2026-10-20, from 19:21Z to 19:39Z.
  - About 19 min was the close-out on 2026-10-21, from 16:25:59Z to 16:45Z.
- **Started:** 2026-10-20T19:21:42Z (after 05-10)
- **Completed:** 2026-10-21T16:45:07Z
- **Tasks:** 2 of 2
- **Files modified:** 4 (0 created)
- **Resumed execution:** the first executor committed all three task commits, then died from an SSE stream timeout. It never ran verification or wrote this SUMMARY. A second executor audited the commits against the plan, ran every verify and the plan-level verification, confirmed RED, and closed out the plan. No code changes were needed.

## Accomplishments

- **Search sync helper.** `(*Indexer).syncCobolFindingFTS(added, evicted)` (`finding_ledger.go`) makes the same store writes as the incremental stage path:
  - `deleteSymbolFTS(evicted)`;
  - then `SymbolFTSBatchUpserter.BatchUpsertSymbolFTS` with `ftsTokensFor` tokens, for added IDs whose node is in the store and passes `shouldIndexForSearch`.
- **Per-repository routing.** `(*MultiIndexer).syncCobolFindingFTSByRepo` groups both lists by `graph.RepoPrefixOfID`. It writes each group only through `mi.GetIndexer(prefix)`, in sorted prefix order, and skips empty or untracked prefixes (T-05-48).
- **Settle-point tails.**
  - `reconcileCobolAfterMutation` captures the drained IDs and calls `idx.syncCobolFindingFTS`.
  - `reconcileCobolRepos` captures them and calls `mi.syncCobolFindingFTSByRepo`.
  - The cold `TrackRepoCtx` tail reaches the second through its `reconcileCobolRepos` call, so `TrackRepoCtx` is unchanged.
- **Master-resolve drain.** `runMasterResolveHookedContext` drains `master.CobolFindingChanges()` right after `ResolveAllContext`, whatever the error, and syncs search. This one site covers all three fallback callers: the incremental no-frontier fallback, the deferred-mutation full fallback, and `RunPreEnrichResolve`. It adds no ledger, no field, and no pending set.
- **External lifecycle on the real path.**
  - Retrieving the missing copybook closes only its finding.
  - Removing the last CALL sweeps its placeholder.
  - History survives a restart with a warm reconcile and is purged on untrack.

## Task Commits

1. **Task 1: Search sync at every settle point and after every master resolve (TDD)**
   - `97a499b9` (test, RED): SearchAfterIncremental, SearchAfterFallbackMasterResolve, and CrossRepoDependentLedgerAndSearch, including the incremental case and both cold-track orders with the shadow off and on.
   - `12160d37` (feat, GREEN): `syncCobolFindingFTS` and `syncCobolFindingFTSByRepo`, the two tail call sites, and the master-resolve drain.
2. **Task 2: External finding lifecycle and the ledger scale benchmark**
   - `e9d71cf1` (test): ExternalClosesOnRetrieve, PlaceholderSweptOnLastReference, HistorySurvivesRestart, HistoryPurgedOnUntrack, and `BenchmarkRecordFindingLedger_LargeRepo` (FirstPass and NoChangePass).
   - It has no feat commit: these tests prove code from 05-05, 05-06, 05-08, and 05-09, and they passed on the real path.

**Plan metadata:** the `docs(05-11)` SUMMARY commit, then the `docs(05-11)` STATE/ROADMAP/REQUIREMENTS commit.

## Files Created/Modified

- `internal/indexer/finding_ledger.go`: `syncCobolFindingFTS` and `syncCobolFindingFTSByRepo`, each with a PURPOSE / RATIONALE / KEYWORDS doc block citing Pitfall 4 and D-03 across repositories.
- `internal/indexer/cobol_reconcile.go`: both tails consume `recordCobolFindingLedger`'s `(added, evicted)` and sync search. The doc sentences are updated.
- `internal/indexer/multi.go`: the drain and sync in `runMasterResolveHookedContext`, with a comment naming the three fallback callers.
- `internal/indexer/cobol_findings_lifecycle_test.go`:
  - helpers `cobolSearchHits`, `assertCobolSearch`, and `cobolExternalFindingFor`;
  - seven `TestCobolFindings_*` tests;
  - `BenchmarkRecordFindingLedger_LargeRepo`.

## Ledger scale

`BenchmarkRecordFindingLedger_LargeRepo` is run with `-benchtime 1x`, no `-race`. The fixture is one repository on a temp SQLite store, seeded outside the timer with 100000 `KindFinding` nodes over 1000 files (100 findings per file). FirstPass opens a fresh store per iteration, also outside the timer.

| Sub-benchmark | ns/op | Wall time (timed op) | Rows appended |
|---|---|---|---|
| `FirstPass` (empty history) | 1,587,097,334 | 1.59 s | 100000 (`rows/op` 100000) |
| `NoChangePass` (after a first pass) | 730,103,167 | 0.73 s | 0 (`rows/op` 0) |

- **Whole benchmark run:** the package reported `ok ... 21.995s`, including seeding and setup. `go test` wall time was 26.77 s (user 15.41 s, sys 9.81 s).
- **Toolchain:** go1.27.1 darwin/arm64.
- **Machine:** Apple M5, 10 cores, 16 GiB RAM, macOS 27.0 (26A428), arm64.
- **Reading:** a no-change pass over 100k findings costs about 0.73 s, which is the steady-state cost at every settle point for a repository this size. This is the baseline for 05-09 T-05-45 (T-05-51 mitigated).

## Decisions Made

- **No `idx.search` writes.** `syncCobolFindingFTS` does not write the in-process `idx.search` backend. The close-out confirmed this matches the stage path's effect:
  - `initialSearchBackend` (indexer.go:733) gives an indexer either `SymbolSearcherBackend`, whose `Add`/`Remove` are empty (symbolsearcher_backend.go:149,153), or the Null backend, which indexes nothing.
  - So the stage path's `idx.search.Add`/`Remove` (incremental_batch.go:950-952) are no-ops in both cases.
  - The plan's "mirror exactly" requirement is therefore met by the FTS writes alone, and no new index is introduced.
- **Delete before upsert.** An ID in both drained lists ends in the correct final state. For example, a shadow-built node that the tail's master evicts is deleted and not re-upserted.
- **Master drain ignores the error.** The master drain runs whatever `ResolveAllContext` returns, and records no ledger. History for a fallback master is a state-based diff recorded at the repository's next settle point (D-07 best effort). The test checks this: `reconcileCobolRepos` appends the active -> closed row afterwards.
- **Two search tiers.** COBOL search assertions query both tiers. A bare identifier is answered from the node table and would pass even with stale FTS. The query `missing <name>` ranks the FTS corpus.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The data volume was full, so verification could not build**
- **Found during:** Task 1 verify, the first command run at close-out.
- **Issue:** `/System/Volumes/Data` was at 100% with 1.0 GiB free. The Go linker failed with `write $WORK/b001/indexer.test~: no space left on device`.
  - TMPDIR held 11,149 leaked hidden `.<hash>-00000000.dylib` files from Bun native-library extraction, about 39 GiB, written Oct 19-20.
  - The Go build cache was 16 GiB.
  - The disk filling up plausibly contributed to the first executor's stream timeout.
- **Fix:** the user approved "Both" through the question tool.
  - The executor deleted the 11,080 dylibs that were older than 6 hours and not open by any process. `lsof` was re-checked right before deletion.
  - It then ran `go clean -cache`.
  - Free space went from 1.0 GiB to 56 GiB.
- **Files modified:** none in the repository. This was an environment-only change outside the project.
- **Verification:** every verify command then ran to completion, starting from a cold build cache.
- **Committed in:** n/a (no repository change).

---

**Total deviations:** 1 environment blocker, resolved with user authorization.
**Impact on plan:** none on code or scope. The plan diff (`git diff --name-only 34f316ab..HEAD`) is exactly the 4 planned files.

## TDD Gate Compliance

- **Task 1:** RED `97a499b9`, then GREEN `12160d37`.
  - The close-out re-checked RED in a detached throwaway worktree at `97a499b9`, since removed.
  - SearchAfterIncremental failed: the FTS tier did not return the new nodes.
  - SearchAfterFallbackMasterResolve failed: the removed nodes were still searchable after the master.
  - CrossRepoDependentLedgerAndSearch failed in incremental, shadow=off, and shadow=on: fdepb's closed nodes were still searchable.
  - Each failure is the reason the plan predicted.
- **Task 2:** `type="auto"` with no TDD gate. Its tests and benchmark prove existing behavior and passed on the first run, so it has no feat commit.

## Verification

- **Task 1 verify** (exact command): exit 0.
  - All three tests are listed.
  - `go test -race` passed on SearchAfterIncremental, SearchAfterFallbackMasterResolve, CrossRepoDependentLedgerAndSearch, HistoryDependentRepo, and HistoryIdenticalRerun (30.0 s, cold cache).
  - gofmt is clean.
- **Task 2 verify** (exact command): exit 0.
  - All four tests and the benchmark are listed.
  - `go test -race` passed on the four tests (13.4 s).
  - gofmt is clean.
- **Acceptance greps:**
  - `syncCobolFindingFTS(` def 1, `syncCobolFindingFTSByRepo(` def 1.
  - `idx.syncCobolFindingFTS(` in cobol_reconcile.go 1, `syncCobolFindingFTSByRepo(` in cobol_reconcile.go 1, and in multi.go 1.
  - In the `runMasterResolveHookedContext` window: `master.CobolFindingChanges()` 1, `syncCobolFindingFTSByRepo(` 1.
  - Ledger calls in multi.go: 0.
  - `RepoPrefixOfID(` in finding_ledger.go: 3.
  - Benchmark def 1, `b.Run("FirstPass"` 1, `b.Run("NoChangePass"` 1.
  - `^func TestCobolFindings_`: 18.
- **Plan verification:**
  - `go test -race -count=1 -timeout 30m ./internal/indexer/ -run '^TestCobolFindings_|^TestCobolLifecycle_'` is ok (219.5 s):
    - 18/18 `TestCobolFindings_*` and 23/23 `TestCobolLifecycle_*` pass;
    - all six cross-repo subtests pass;
    - 0 FAIL, 0 SKIP.
  - `go test -race -count=1 -timeout 30m ./internal/indexer/ -run 'FTS|Search'` is ok (50.7 s; 38 top-level PASS, 0 FAIL, 0 SKIP).
  - `go test -race -count=1 -timeout 30m ./cmd/gortex/ -run '^TestCobol'` is ok (55.7 s; 21 top-level PASS, 0 FAIL, 0 SKIP).
  - The benchmark command ran to completion (numbers above).
  - `go build ./...` is ok.
  - `git diff --name-only HEAD` lists only the pre-existing, unrelated `.planning/config.json` and `.planning/state.json`.
  - `git diff --name-only 34f316ab..HEAD` lists exactly the 4 planned files.

## Issues Encountered

- The first executor died from an SSE stream timeout after its last task commit. The close-out found nothing missing in the three commits.
- The data volume was full. The user authorized the cleanup, which is recorded under Deviations.

## User Setup Required

None. No external service configuration is required. Running daemons pick up the search sync when rebuilt.

## Next Phase Readiness

- 05-12 can rely on these behaviors:
  - COBOL findings and placeholders are consistent between the store and symbol search at every settle point and after every master resolve, across `resolve_from` dependents.
  - The ledger has a recorded 100k-finding baseline.
- Any future code path that calls `ResolveAll` or `ReconcileCobol` outside `runMasterResolveHookedContext` and the two settle-point tails must drain `CobolFindingChanges()` and call `syncCobolFindingFTSByRepo`, or its nodes go stale in search.
- **Environment watch item:** Bun-extracted `.<hash>-00000000.dylib` files leak into TMPDIR at about 3.5 MB per launch (about 39 GiB over two days). Long agent runs on this machine can fill the disk again.

## Self-Check: PASSED

- FOUND: internal/indexer/finding_ledger.go, internal/indexer/cobol_reconcile.go, internal/indexer/multi.go, internal/indexer/cobol_findings_lifecycle_test.go
- FOUND commits: 97a499b9, 12160d37, e9d71cf1
- Both task verify commands, every acceptance criterion, and every plan-level verification command passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-21*
