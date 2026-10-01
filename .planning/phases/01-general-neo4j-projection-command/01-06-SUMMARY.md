---
phase: 01-general-neo4j-projection-command
plan: 06
subsystem: database
tags: [neo4j, generation-protocol, constraints, reconciliation, integration-testing]
requires:
  - phase: 01-03
    provides: official-driver production tracer and owner manifest activation
  - phase: 01-04
    provides: canonical projection identities and closed graph tokens
  - phase: 01-05
    provides: bounded immutable snapshot streaming and progress semantics
provides:
  - Exact-owner generation lock with idempotent same-operation replay
  - Neo4j 5.26 capability checks, constraints, bounded MERGE writes, and 30-second transaction/retry limits
  - Atomic manifest activation followed by bounded resumable exact-owner cleanup
  - Mandatory disposable Neo4j acceptance with no scenario-level skips
affects: [01-07]
actuals:
  tokens: 9450
  tasks: 2
  commits: 4
tech-stack:
  added: []
  patterns: [exact-owner compare-and-switch, generation-specific staging, post-activation bounded reconciliation, truthful partial-cleanup results]
key-files:
  created:
    - internal/neo4jprojection/neo4j_test.go
  modified:
    - internal/neo4jprojection/neo4j.go
    - internal/neo4jprojection/service.go
    - internal/neo4jprojection/service_test.go
    - internal/neo4jprojection/neo4j_integration_test.go
    - internal/neo4jprojection/tracer_integration_test.go
    - scripts/test-neo4j.sh
key-decisions:
  - "Treat manifest activation as logical completion, then report physical reconciliation independently so a cleanup interruption never misstates the active snapshot."
  - "Use the same operation and generation as the idempotent lock identity; reject every competing operation for that exact owner."
patterns-established:
  - "All transport transactions use a 30-second server timeout and the driver uses a 30-second managed retry ceiling."
  - "Former generations are deleted relationships-first in exact-owner batches no larger than the requested batch size."
requirements-completed: [NEO-06, NEO-07, NEO-08, NEO-11, NEO-12, NEO-14, NEO-15, NEO-16, NEO-17, STORE-01, STORE-02, STORE-03, STORE-04, STORE-05]
coverage:
  - id: D1
    description: "Exact-owner generation locking, bounded idempotent writes, atomic activation, and truthful resumable cleanup"
    requirement: NEO-12
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/neo4j_test.go#TestNeo4jConstraints|TestNeo4jGenerationActivation|TestNeo4jRetryBounds|TestNeo4jStaleReconciliation|TestNeo4jCancellation"
        status: pass
    human_judgment: false
  - id: D2
    description: "Disposable Neo4j 5.26 production path covers dry-run, idempotency, isolation, failure, cancellation, cleanup interruption and rerun, redaction, and SQLite immutability"
    requirement: NEO-16
    verification:
      - kind: integration
        ref: "scripts/test-neo4j.sh --timeout-seconds 240"
        status: pass
    human_judgment: false
duration: 38 min
completed: 2026-09-21
status: complete
---

# Phase 1 Plan 6: Atomic Neo4j Generation Protocol Summary

**Exact-owner generation locks now guard bounded idempotent Neo4j writes, one atomic manifest activation, and resumable physical cleanup with truthful partial-cleanup results.**

## Performance

- **Duration:** 38 min
- **Started:** 2026-09-21T21:55:10Z
- **Completed:** 2026-09-21T22:33:10Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added Neo4j 5.26 minimum-version enforcement, node/manifest/per-relationship constraints, parameterized `UNWIND`/`MERGE` writes, and explicit 30-second transaction and retry bounds.
- Added exact-owner lock ownership, generation compare-and-switch activation, bounded relationships-first stale cleanup, remaining-count census, and ordinary-rerun recovery.
- Made results distinguish activated snapshot completeness from physical cleanup completeness, including cancellation and actionable rerun guidance.
- Converted the disposable gate to execute the complete production tracer by default and expanded it across dry-run, idempotency, neighboring data, pre-activation failure, cancellation, cleanup interruption/rerun, graph shape, redaction, and SQLite immutability.

## Task Commits

1. **Task 1 RED: Add failing generation protocol tests** - `1b456a27` (test)
2. **Task 1 GREEN: Enforce atomic generation protocol** - `a3ab51c6` (feat)
3. **Task 2: Enforce mandatory Neo4j acceptance** - `c3efd224` (test)
4. **Task 2 fix: Isolate disposable tracer from ordinary tests** - `df3553bd` (fix)

## Files Created/Modified

- `internal/neo4jprojection/neo4j.go` - Official-driver constraints, lock, staging, activation, error safety, and bounded cleanup.
- `internal/neo4jprojection/neo4j_test.go` - Focused protocol tests for constraints, activation, retries, cleanup, and cancellation.
- `internal/neo4jprojection/service.go` - Cleanup transport contract and truthful activation-versus-cleanup result semantics.
- `internal/neo4jprojection/service_test.go` - Existing transport fakes adapted to reconciliation lifecycle.
- `internal/neo4jprojection/neo4j_integration_test.go` - Mandatory gate no longer contains scenario-level placeholders or skips.
- `internal/neo4jprojection/tracer_integration_test.go` - Real-server dry-run, cancellation, cleanup interruption/rerun, isolation, redaction, shape, and source immutability coverage.
- `scripts/test-neo4j.sh` - Production tracer is the mandatory default and required-scenario mode defaults on.

## Decisions Made

- Activation defines logical snapshot completeness; physical cleanup is a subsequent bounded phase with independent status and stale counts.
- Same-owner replay is accepted only when both operation ID and pending generation match; competing operations are rejected.
- Cleanup deletes relationships before now-unconnected nodes and predicates every operation by exact owner and non-active generation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected production relationship assertion**
- **Found during:** Task 2 disposable-server verification
- **Issue:** The tracer asserted a historical relationship type name that did not match the mapper's closed generated type.
- **Fix:** Asserted owner/generation-scoped relationships independent of presentation type while retaining shape checks.
- **Files modified:** `internal/neo4jprojection/tracer_integration_test.go`
- **Verification:** Mandatory disposable Neo4j suite passes.
- **Committed in:** `c3efd224`

**2. [Rule 2 - Missing Critical] Preserved ordinary no-server test behavior**
- **Found during:** Full repository race suite
- **Issue:** The production tracer failed rather than skipped outside the explicit disposable integration lane, violating NEO-17.
- **Fix:** Restored the environment gate while keeping the mandatory script responsible for enabling and executing the test.
- **Files modified:** `internal/neo4jprojection/tracer_integration_test.go`
- **Verification:** `go test -race ./internal/neo4jprojection` and the mandatory disposable suite both pass.
- **Committed in:** `df3553bd`

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical functionality).
**Impact on plan:** Both fixes preserve the specified acceptance semantics without broadening product scope.

## Issues Encountered

- The full `GOWORK=off go test -race ./...` run reached unrelated pre-existing 10-minute package timeouts in `internal/graph/store_sqlite` and `internal/indexer`. Both packages time out when run independently as well. The changed `internal/neo4jprojection` race suite and mandatory disposable Neo4j suite pass.

## User Setup Required

None - the integration harness provisions and removes its disposable Neo4j container.

## Next Phase Readiness

- Plan 01-07 can expose stable cleanup status, stale counts, cancellation, and failure semantics through CLI and MCP.
- No blocker exists in the Neo4j projection package; unrelated repository race-suite timeout behavior remains outside this plan.

## Self-Check: PASSED

- All seven created or modified implementation/test files exist.
- Commits `1b456a27`, `a3ab51c6`, `c3efd224`, and `df3553bd` exist in order.
- All five exact focused protocol tests are discoverable and pass.
- `scripts/test-neo4j.sh --timeout-seconds 240` passes against pinned Neo4j 5.26 with no scenario-level skips.
- `GOWORK=off go test -race ./internal/neo4jprojection` passes without a Neo4j server.
- No TODO, FIXME, placeholder, skipped protocol scenario, or production test stub was introduced.

---
*Phase: 01-general-neo4j-projection-command*
*Completed: 2026-09-21*
