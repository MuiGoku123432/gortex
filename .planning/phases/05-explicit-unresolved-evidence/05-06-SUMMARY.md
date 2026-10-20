---
phase: 05-explicit-unresolved-evidence
plan: 06
subsystem: graph-store
tags: [finding, ledger, lifecycle, sqlite, sidecar, migration, v23]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.FindingStatusActive/Closed/Superseded and the finding ID grammar (05-01); kind-guarded finding eviction (05-02)"
provides:
  - "graph.FindingTransition, graph.FindingHistoryReader, graph.FindingHistoryAppender (optional capabilities)"
  - "graph.FindingLedgerTransitions: pure, deterministic open / close / supersede / reopen diff (D-03)"
  - "SQLite sidecar finding_history (schema v23, in-place additive migration 'persist finding history') with index finding_history_by_finding"
  - "(*store_sqlite.Store).FindingHistory, LatestFindingTransitions, AppendFindingTransitions"
  - "Registry entries: viewGenSidecars (with index), purgeSidecarTables, orphanScanTables, rekeyDropTables, capability checklist rows"
affects: [05-09, 05-11, 05-12, phase-08]

actuals:
  tokens: 7519
  tasks: 3
  commits: 7

tech-stack:
  added: []
  patterns:
    - "Finding history records every transition, opens included, so the open set is 'latest row per finding is active' and the ledger needs no pre-eviction read"
    - "Per-repository seq (MAX + 1 inside the write transaction) orders rows; no wall-clock value is stored"
    - "Latest-row read is one grouped query; CROSS JOIN pins the index-then-primary-key loop order on stores without planner statistics"

key-files:
  created:
    - internal/graph/finding_ledger.go
    - internal/graph/finding_ledger_test.go
    - internal/graph/store_sqlite/store_finding_history.go
    - internal/graph/store_sqlite/store_finding_history_test.go
  modified:
    - internal/graph/store_sqlite/schema.go
    - internal/graph/store_sqlite/schema_version.go
    - internal/graph/store_sqlite/schema_sidecar_view_gen_test.go
    - internal/graph/store_sqlite/store_purge.go
    - internal/graph/store_sqlite/store_generation_read_test.go
    - internal/graph/store_sqlite/migration_unprefixed_purge_test.go

key-decisions:
  - "FindingLedgerTransitions compares IDs only; a gone finding is superseded by the k-th newly opened finding of the same (container, class, kind) in (StartByte, FindingID) order, leftovers close or open alone; reopened findings count as newly opened"
  - "LatestFindingTransitions uses CROSS JOIN with the grouped subquery first: without planner statistics SQLite otherwise picked a plan that rescans the subquery once per history row"
  - "finding_history stays out of the EvictRepo sweep (D-07) but is purged on untrack, dropped on rekey, and swept with generations through viewGenSidecars"

patterns-established:
  - "A new generation-keyed sidecar needs: viewGenSidecars entry (with indexes), a schema bump plus in-place migration, both v14 special cases in schema_sidecar_view_gen_test.go, a seedSidecarRows row, purge/orphan/rekey registry entries, and capability checklist rows"

requirements-completed: [GAP-06, GAP-07]

coverage:
  - id: D1
    description: "Pure ledger diff opens new findings, closes gone ones, supersedes within (container, class, kind) in source order, reopens closed IDs, and writes nothing for identical or re-stamped input; full log and latest-row projection agree"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_OpensNew"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_ClosesGone"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_SupersedePairsInSourceOrder"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_NeverPairsAcrossContainerOrClass"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_ReopenAfterClose"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_LatestRowsEqualFullHistory"
        status: pass
    human_judgment: false
  - id: D2
    description: "Unchanged evidence (identical IDs, including re-stamped parser or extractor versions) yields zero transitions"
    requirement: GAP-06
    verification:
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_IdenticalInputNoTransitions"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_ledger_test.go#TestFindingLedger_RestampSameIDNoTransition"
        status: pass
    human_judgment: false
  - id: D3
    description: "finding_history v23 sidecar persists, reads (full log and latest row per finding), migrates in place from v22 with its index, and survives EvictRepo"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/graph/store_sqlite/store_finding_history_test.go#TestFindingHistory_AppendAndRead"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/store_finding_history_test.go#TestFindingHistory_LatestPerFinding"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/store_finding_history_test.go#TestFindingHistory_SurvivesEvictRepo"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/store_finding_history_test.go#TestFindingHistory_MigrationV23"
        status: pass
    human_judgment: false
  - id: D4
    description: "Untrack purges history, rekey drops it, orphan scan sees it, and every store registry test knows the sidecar"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/graph/store_sqlite/store_finding_history_test.go#TestFindingHistory_PurgedWithRepo"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/graph/store_sqlite/"
        status: pass
    human_judgment: false

duration: 26min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 06: Finding Lifecycle Ledger Summary

**A pure `graph.FindingLedgerTransitions` diff (open, close, supersede within one container/class/kind in source order, reopen) plus the append-only SQLite `finding_history` sidecar (schema v23, latest-row-per-finding read through `finding_history_by_finding`) that survives reindexing and is purged on untrack, ready for the 05-09 durable ledger pass.**

## Performance

- **Duration:** ~26 min
- **Started:** 2026-10-20T17:33:50Z
- **Completed:** 2026-10-20T17:59:51Z
- **Tasks:** 3
- **Files modified:** 10 (4 created, 6 modified)

## Accomplishments

- `internal/graph/finding_ledger.go`: `FindingTransition`, the optional `FindingHistoryReader` (`FindingHistory` in append order, `LatestFindingTransitions` one row per finding by ID) and `FindingHistoryAppender`, and `FindingLedgerTransitions`, which compares IDs only (D-02), never pairs across container, class, or kind (D-03), and returns nil when nothing changed (GAP-06).
- `internal/graph/store_sqlite/store_finding_history.go`: the 17-column `finding_history` table keyed `(view_gen, repo_prefix, seq)`, the `finding_history_by_finding` index, a single-query latest-row read (`MAX(seq) ... GROUP BY finding_id`), and an atomic, chunked append (50 rows x 17 parameters) that numbers rows after the repository's highest `seq`.
- Schema v23 in-place migration "persist finding history"; the sidecar is in `viewGenSidecars` (fresh DDL, generation copy, payload sweep, index), `purgeSidecarTables`, `orphanScanTables`, and `rekeyDropTables`, and not in the `EvictRepo` sweep (D-07).
- The whole `internal/graph/store_sqlite` package passes under `-race` (783 s).

## Task Commits

1. **Task 1: Pure ledger diff with deterministic close and supersede** - `7680599f` (test, RED), `cb975775` (feat, GREEN)
2. **Task 2: finding_history sidecar table and v23 in-place migration** - `0133ef21` (test, RED), `839269b1` (feat, GREEN)
3. **Task 3: Purge, orphan, rekey, and capability registries** - `0354c010` (test, RED), `1c139fe5` (feat, GREEN)

**Plan metadata:** recorded in the docs(05-06) commit that adds this file.

## Files Created/Modified

- `internal/graph/finding_ledger.go` - transition row, history capabilities, pure ledger diff
- `internal/graph/finding_ledger_test.go` - eight `TestFindingLedger_*` tests
- `internal/graph/store_sqlite/store_finding_history.go` - table body, index, v23 step, reads, append
- `internal/graph/store_sqlite/store_finding_history_test.go` - five `TestFindingHistory_*` tests
- `internal/graph/store_sqlite/schema.go` - `viewGenSidecars` entry with the finding index
- `internal/graph/store_sqlite/schema_version.go` - `currentSchemaVersion = 23` and the v23 migration
- `internal/graph/store_sqlite/schema_sidecar_view_gen_test.go` - v14 special cases extended to `finding_history`; `seedSidecarRows` seeds one history row
- `internal/graph/store_sqlite/store_purge.go` - purge, orphan-scan, and rekey-drop entries
- `internal/graph/store_sqlite/store_generation_read_test.go` - two capability checklist rows
- `internal/graph/store_sqlite/migration_unprefixed_purge_test.go` - v22 test asserts `currentSchemaVersion` instead of a literal 22

## Decisions Made

- Reopened findings (`closed -> active`) count as newly opened, so they can be the successor of a gone finding in the same container, class, and kind. This follows "an unpaired newly opened finding" in the plan truth.
- 05-01's `#k` sibling ordinals need no special case: inserting an identical earlier sibling shifts IDs so the diff sees one extra open (the highest ordinal), and removing one sees one close. The diff stays ID-only per D-01.
- `LatestFindingTransitions` puts the grouped subquery first under `CROSS JOIN`. `EXPLAIN QUERY PLAN` on a table without statistics showed `SCAN latest` nested inside an index search of `h`, which rescans the subquery once per history row. The pinned order always gives: index search for the subquery, then a primary-key lookup per finding.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Hard-coded schema version in TestMigrationV22_PurgesUnprefixedRows**
- **Found during:** Task 2 (verify run, `Migration` subset)
- **Issue:** The test asserted `user_version == 22` after open, so the v23 bump failed it.
- **Fix:** Assert `currentSchemaVersion`, as the neighbouring migration tests do.
- **Files modified:** internal/graph/store_sqlite/migration_unprefixed_purge_test.go (not in the plan's file list)
- **Verification:** Task 2 verify subset passes under `-race`
- **Committed in:** 839269b1

**2. [Rule 3 - Blocking] seedSidecarRows had no finding_history row**
- **Found during:** Task 2 (verify run)
- **Issue:** `TestSchemaSidecarViewGenerationMigrationIsIdempotent` requires every `viewGenSidecars` table to hold rows after seeding.
- **Fix:** `seedSidecarRows` appends one history row. The v14 downgrade test already drops the table through the extended special case.
- **Files modified:** internal/graph/store_sqlite/schema_sidecar_view_gen_test.go (in the plan's file list)
- **Verification:** sidecar view-generation tests pass under `-race`
- **Committed in:** 839269b1

**3. [Rule 1 - Bug] Latest-row query plan without statistics**
- **Found during:** Task 2 (query plan check)
- **Issue:** A plain `JOIN` let SQLite choose a nested `SCAN latest` on an unanalyzed table.
- **Fix:** `CROSS JOIN` with the subquery first. Same SQL semantics, same single `GROUP BY finding_id`.
- **Files modified:** internal/graph/store_sqlite/store_finding_history.go
- **Verification:** `EXPLAIN QUERY PLAN` shows the covering-index search, then `SEARCH h USING PRIMARY KEY`, with and without statistics; `TestFindingHistory_LatestPerFinding` passes
- **Committed in:** 839269b1

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 bug)
**Impact on plan:** One test file outside the planned list changed (one assertion). No scope creep.

## TDD Gate Compliance

All three tasks followed RED then GREEN. `7680599f`, `0133ef21`, and `0354c010` failed (build failures for the missing symbols, then a real assertion failure for the purge test). `cb975775`, `839269b1`, and `1c139fe5` made them pass. No refactor commits were needed.

## Issues Encountered

- Between Task 2 and Task 3, `TestGenerationCapabilityChecklistIsComplete` failed by design: the new capabilities were asserted on `*Store` before their checklist rows existed. Task 3 added the rows.

## Verification

- Task 1 verify: all eight ledger tests listed and passing under `-race`; gofmt clean; signature grep prints 1.
- Task 2 verify: four tests listed; the `TestFindingHistory_|SidecarViewGen|Migration|SchemaVersion|ColdMaintenance` subset passes under `-race`; gofmt clean. Acceptance greps: 1, 1, 2 (>= 1), 1, 1, 0.
- Task 3 verify: the registry-sensitive subset passes under `-race` (including `TestPurgeRepo_ClearsEveryTable_LeavesOthersAndGlobals`, `TestOrphanRepoPrefixes_SidecarOnlyResidue`, `TestRekeyRepoPrefix_MovesProvenanceDropsNodeIDKeyed`, and `TestGenerationCapabilityChecklistIsComplete`); acceptance greps 3 and 2.
- Plan verification: `GOWORK=off go test -race -count=1 -timeout 30m ./internal/graph/store_sqlite/` passed (782.9 s); `GOWORK=off go test -race -count=1 -timeout 30m ./internal/graph/ -run '^TestFindingLedger_'` passed; `go build ./...` passed.
- `git diff --name-only 28557857..HEAD` lists the 9 planned files plus `migration_unprefixed_purge_test.go` (deviation 1).

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 05-09 can probe `graph.FindingHistoryReader` / `graph.FindingHistoryAppender` on the durable store. It should call `LatestFindingTransitions`, pass the result and the active finding records to `FindingLedgerTransitions`, and append the rows. In-memory `*graph.Graph` does not implement the capabilities, so the pass is a no-op there.
- 05-11 should measure the first-pass append volume (`BenchmarkRecordFindingLedger_LargeRepo`). Every active finding gets one open row, inside one transaction.
- GAP-06 and GAP-07 are also declared by sibling plans; the shared-ID gate decides when they are marked complete.

## Self-Check: PASSED

- FOUND: internal/graph/finding_ledger.go, internal/graph/finding_ledger_test.go, internal/graph/store_sqlite/store_finding_history.go, internal/graph/store_sqlite/store_finding_history_test.go
- FOUND commits: 7680599f, cb975775, 0133ef21, 839269b1, 0354c010, 1c139fe5
- All three task verify commands and the plan verification passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
