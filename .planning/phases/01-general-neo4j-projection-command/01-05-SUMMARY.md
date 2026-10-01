---
phase: 01-general-neo4j-projection-command
plan: 05
subsystem: database
tags: [sqlite, neo4j, keyset-pagination, batching, cancellation, progress]
requires:
  - phase: 01-02
    provides: immutable exact-scope SQLite snapshot contract
  - phase: 01-03
    provides: production official-driver tracer and generation activation
  - phase: 01-04
    provides: canonical projected identities and safe properties
provides:
  - Estate-scale immutable node and all-edge snapshot paging with frozen boundaries
  - DB/WAL/SHM and canonical graph immutability coverage
  - Bounded 500-record projection batches with dry-run, deadline, cancellation, and exact progress cadence
affects: [01-06, 01-07]
actuals:
  tokens: 9498
  tasks: 2
  commits: 2
tech-stack:
  added: []
  patterns: [callback page streaming, frozen edge boundary, bounded transport aggregation, monotonic progress milestones]
key-files:
  created:
    - internal/graph/store_sqlite/projection_fingerprint_test.go
  modified:
    - internal/graph/scoped_projection.go
    - internal/graph/store_sqlite/scoped_projection.go
    - internal/graph/store_sqlite/scoped_projection_test.go
    - internal/testutil/graphfixture/fixture.go
    - internal/neo4jprojection/service.go
    - internal/neo4jprojection/service_test.go
    - internal/neo4jprojection/tracer_integration_test.go
key-decisions:
  - "Expose callback-based snapshot pages so SQLite cursors close before transport work while one read transaction preserves a coherent source view."
  - "Count and stage nodes before edges in batches capped at 500 records, with progress emitted only after successful transport calls."
patterns-established:
  - "Snapshot page callbacks are the bounded handoff between SQLite and remote projection work."
  - "Progress reports phase boundaries and exact crossed 1000-record milestones outside retryable transport callbacks."
requirements-completed: [NEO-02, NEO-08, NEO-11, NEO-13, NEO-15, NEO-17, STORE-01, STORE-02, STORE-05]
coverage:
  - id: D1
    description: "Exact multi-repository snapshots keyset-page deterministic nodes and all edge kinds under one immutable descriptor"
    requirement: NEO-08
    verification:
      - kind: integration
        ref: "internal/graph/store_sqlite/scoped_projection_test.go#TestScopedProjectionEstatePagination"
        status: pass
    human_judgment: false
  - id: D2
    description: "Snapshot success and cancellation preserve DB/WAL/SHM fingerprints and canonical graph content"
    requirement: NEO-15
    verification:
      - kind: integration
        ref: "internal/graph/store_sqlite/projection_fingerprint_test.go#TestScopedProjectionFingerprintCoverage|TestScopedProjectionCancellation"
        status: pass
    human_judgment: false
  - id: D3
    description: "Projection service streams bounded batches with deadlines, cancellation, dry-run census, and exact progress milestones"
    requirement: NEO-11
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/service_test.go#TestProjectionBatchAggregation|TestProjectionProgressCadence|TestProjectionBoundedReads"
        status: pass
    human_judgment: false
duration: 17 min
completed: 2026-09-21
status: complete
---

# Phase 1 Plan 5: Estate-Scale Projection Streaming Summary

**Immutable exact-scope SQLite snapshots now feed Neo4j through cursor-closed keyset pages and bounded 500-record transport batches with cancellation, dry-run, deadlines, and deterministic progress.**

## Performance

- **Duration:** 17 min
- **Started:** 2026-09-21T21:36:11Z
- **Completed:** 2026-09-21T21:53:09Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- Replaced whole-snapshot node and edge reads with callback-based 256-row keyset pages bound to one read transaction, exact canonical repository allow-set, and a frozen edge boundary.
- Added broad estate fixtures proving deterministic multi-page traversal of nodes and all edge kinds, prompt cancellation, consumer error propagation, cursor closure, and DB/WAL/SHM plus canonical graph immutability.
- Added default 500-record and configured smaller transport batches, a 30-minute operation deadline, non-mutating dry-run census, cancellation state, phase boundaries, and exact 1000-record progress milestones.

## Task Commits

1. **Task 1: Extend immutable snapshot across estate-scale fixtures** - `cb4281cc` (feat)
2. **Task 2: Stream real snapshot through bounded service batches** - `850055cd` (feat)

## Files Created/Modified

- `internal/graph/scoped_projection.go` - Bounded callback page contract for immutable snapshots.
- `internal/graph/store_sqlite/scoped_projection.go` - Context-aware keyset node/edge pages, frozen edge boundary, and page-wise endpoint hydration.
- `internal/graph/store_sqlite/scoped_projection_test.go` - Broad exact-scope and all-edge pagination contract.
- `internal/graph/store_sqlite/projection_fingerprint_test.go` - Cancellation and DB/WAL/SHM plus canonical immutability coverage.
- `internal/testutil/graphfixture/fixture.go` - Canonical fingerprint support for both fixture and production graph schemas.
- `internal/neo4jprojection/service.go` - Bounded batching, timeout, dry-run, cancellation, and progress orchestration.
- `internal/neo4jprojection/service_test.go` - Exact batch, cadence, bounded-read, cancellation, and dry-run tests.
- `internal/neo4jprojection/tracer_integration_test.go` - Production tracer adapted to the bounded snapshot contract.

## Decisions Made

- Page callbacks replace whole-graph slice returns because the service must never retain an estate-sized graph and SQLite cursors must close before remote calls.
- The batch cap counts combined node and edge records; nodes and edges are staged separately because relationship staging depends on node existence.
- Dry-run opens the same immutable source snapshot and performs target inspection, but skips lock, staging, and activation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Extended canonical fingerprints to production store schema**
- **Found during:** Task 1 fingerprint verification
- **Issue:** Wave 0 `graphfixture.Canonical` only understood its compact fixture schema and could not fingerprint the production SQLite node/edge schema required by the plan.
- **Fix:** Added schema detection and complete deterministic canonical queries for production nodes and edges while retaining the Wave 0 fixture path.
- **Files modified:** `internal/testutil/graphfixture/fixture.go`
- **Verification:** `TestScopedProjectionFingerprintCoverage` passes with DB/WAL/SHM and canonical equality.
- **Committed in:** `cb4281cc`

**Total deviations:** 1 auto-fixed (1 missing critical functionality).
**Impact on plan:** The extension was necessary to reuse the mandated fingerprint API against the authoritative production store. No production dependency or write path was added.

## Issues Encountered

- Reopening a production SQLite store can repair missing planner statistics and legitimately change database bytes. The fingerprint test stabilizes the store before its baseline, then compares the read-only projection paths from that stable state.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-06 can reconcile inactive generations while using bounded, cancellation-aware service results.
- Plan 01-07 can expose batch, timeout, dry-run, progress, and cancellation semantics through CLI and MCP adapters.

## Self-Check: PASSED

- All eight created or modified source/test files exist.
- Task commits `cb4281cc` and `850055cd` exist in order.
- Both exact test-discovery guards and focused suites pass.
- `GOWORK=off go test ./internal/graph/store_sqlite ./internal/neo4jprojection -run 'Projection|Snapshot' -count=1` passes.
- Gortex search confirms no `neo4jprojection` import exists in `internal/indexer`.
- No TODO, FIXME, placeholder, or skipped-test stubs were introduced.

---
*Phase: 01-general-neo4j-projection-command*
*Completed: 2026-09-21*
