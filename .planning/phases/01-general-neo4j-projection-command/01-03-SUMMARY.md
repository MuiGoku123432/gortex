---
phase: 01-general-neo4j-projection-command
plan: 03
subsystem: database
tags: [neo4j, sqlite, projection, integration-testing, official-driver]
requires:
  - phase: 01-01
    provides: disposable Neo4j exact-test gate and SQLite fixture contracts
  - phase: 01-02
    provides: named Neo4j profiles and immutable scoped SQLite snapshots
provides:
  - Production SQLite-to-Neo4j projection service using the official v6 driver
  - Exact-owner generation staging and atomic manifest activation
  - Mandatory disposable-server tracer proving activation, failure preservation, redaction, and SQLite immutability
affects: [01-04, 01-05, 01-06, 01-07]
actuals:
  tokens: 7274
  tasks: 2
  commits: 3
tech-stack:
  added: [github.com/neo4j/neo4j-go-driver/v6 v6.3.0]
  patterns: [invocation-local transport, immutable source snapshot, generation staging, atomic owner manifest activation]
key-files:
  created:
    - internal/neo4jprojection/service.go
    - internal/neo4jprojection/service_test.go
    - internal/neo4jprojection/neo4j.go
    - internal/neo4jprojection/tracer_integration_test.go
  modified:
    - internal/neo4jprojection/neo4j_integration_test.go
    - scripts/test-neo4j.sh
    - go.mod
    - go.sum
key-decisions:
  - "Keep the official Neo4j driver invocation-local behind a narrow transport lifecycle; ordinary Gortex startup remains Neo4j-free."
  - "Stage generation-specific physical records and make only one exact-owner manifest pointer switch visible."
patterns-established:
  - "Projection lifecycle: immutable snapshot, inspect, owner lock, stage, atomic activate, explicit close."
  - "Disposable integration tests publish Bolt only on random loopback ports and retain strict exact-test selection."
requirements-completed: [NEO-04, NEO-05, NEO-06, NEO-07, NEO-08, NEO-10, NEO-11, NEO-12, NEO-14, NEO-15, NEO-16, NEO-17, STORE-01, STORE-02, STORE-03, STORE-04, STORE-05]
coverage:
  - id: D1
    description: "Production service projects an immutable exact-scope SQLite snapshot through the official driver with atomic owner activation"
    requirement: NEO-04
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/service_test.go#TestNeo4jTracerContract"
        status: pass
    human_judgment: false
  - id: D2
    description: "Disposable Neo4j tracer proves owner/generation activation, idempotency, neighboring-data preservation, failure preservation, redaction, and SQLite immutability"
    requirement: NEO-17
    verification:
      - kind: integration
        ref: "scripts/test-neo4j.sh --timeout-seconds 180 --run 'TestNeo4jProductionTracer'"
        status: pass
    human_judgment: false
duration: 13 min
completed: 2026-09-21
status: complete
---

# Phase 1 Plan 3: Production Neo4j Tracer Summary

**A real immutable SQLite scope now stages generation-specific graph records through Neo4j's official v6 driver and atomically activates them under an exact-owner manifest.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-09-21T21:10:47Z
- **Completed:** 2026-09-21T21:23:16Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- Added the permanent projection service and invocation-local official-driver transport with connectivity/version inspection, constraints, owner exclusion, generation staging, and compare-and-switch activation.
- Proved the production path against disposable Neo4j 5.26 using a real production SQLite store containing two scoped nodes and one relationship.
- Proved identical reruns remain idempotent, neighboring records survive, pre-activation failure preserves the prior generation, credentials and source secret canaries do not reach Neo4j or results, and SQLite DB/WAL/SHM plus canonical rows remain unchanged.

## Task Commits

1. **Task 1 RED: Add failing production tracer contract** - `e3cd2489` (test)
2. **Task 1 GREEN: Implement production projection tracer** - `881d3594` (feat)
3. **Task 2: Execute the real disposable-server tracer gate** - `3f29f8d1` (test)

## Files Created/Modified

- `internal/neo4jprojection/service.go` - Shared request/result, immutable snapshot orchestration, and transport lifecycle.
- `internal/neo4jprojection/service_test.go` - Ordering, lifecycle, prior-active preservation, and lazy driver construction contract.
- `internal/neo4jprojection/neo4j.go` - Official-driver inspection, owner lock, physical staging, and atomic activation.
- `internal/neo4jprojection/tracer_integration_test.go` - Real disposable-server production tracer and assertions.
- `internal/neo4jprojection/neo4j_integration_test.go` - Exact-run script fixture updated for published Bolt ports.
- `scripts/test-neo4j.sh` - Random loopback Bolt publication for host-side Go tests.
- `go.mod`, `go.sum` - Pinned and verified Neo4j Go driver v6.3.0.

## Decisions Made

- Driver creation stays behind the service transport factory and occurs only during an explicit push.
- The production tracer uses the real `store_sqlite.Store` rather than adapting the Wave 0 raw fixture schema, while reusing Wave 0 identity and ownership constants.
- A failed activation intentionally leaves staged records unreachable and preserves the manifest's prior active pointer; cleanup belongs to the later reconciliation plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Published disposable Bolt to the host**
- **Found during:** Task 2 disposable gate
- **Issue:** The script passed a Docker-network-only hostname to a Go test running on the host, so DNS resolution failed.
- **Fix:** Published container port 7687 on a random loopback port, discovered it through `docker port`, and retained the isolated Docker network.
- **Files modified:** `scripts/test-neo4j.sh`, `internal/neo4jprojection/neo4j_integration_test.go`
- **Verification:** The exact script contract suite and real disposable tracer both pass.
- **Committed in:** `3f29f8d1`

**2. [Rule 1 - Bug] Corrected Neo4j 5.26 existence assertions**
- **Found during:** Task 2 disposable gate
- **Issue:** Pattern `exists(...)` syntax used by the initial assertion query was invalid in Neo4j 5.26.
- **Fix:** Replaced it with supported `count { MATCH ... }` existence subqueries.
- **Files modified:** `internal/neo4jprojection/tracer_integration_test.go`
- **Verification:** The mandatory disposable tracer passes against pinned Neo4j 5.26.
- **Committed in:** `3f29f8d1`

**Total deviations:** 2 auto-fixed (1 blocking issue, 1 bug).
**Impact on plan:** Both fixes were required to execute the specified production path against the pinned disposable server. No product scope was added.

## Issues Encountered

- Gortex post-change detection and diff-context calls timed out while the daemon was reindexing the newly created integration test. Earlier and later graph checks localized the intended service/transport surface; `git diff --check`, focused suites, and the mandatory real-server gate passed.

## User Setup Required

None - no persistent external service configuration required.

## Next Phase Readiness

- Plans 01-04 through 01-06 can expand mapping, batching, retry, and reconciliation on the proven service and transport boundary.
- Plan 01-07 can expose the shared request/result without importing Neo4j driver types into CLI or MCP adapters.

## Self-Check: PASSED

- All four created files and four modified dependency/harness files exist.
- Task commits `e3cd2489`, `881d3594`, and `3f29f8d1` exist in order.
- Exact prerequisite contract selectors and their focused package suite pass.
- `scripts/test-neo4j.sh --timeout-seconds 180 --run 'TestNeo4jProductionTracer'` passes without skipping.
- The disposable assertion proves nodes, relationship, activation, idempotency, neighboring canaries, pre-activation preservation, redaction, and SQLite DB/WAL/SHM plus canonical immutability.

---
*Phase: 01-general-neo4j-projection-command*
*Completed: 2026-09-21*
