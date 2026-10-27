---
phase: 01-general-neo4j-projection-command
plan: 01
subsystem: testing
tags: [neo4j, sqlite, integration-testing, fixtures]
requires: []
provides:
  - Package-local projection protocol recorder with failure injection and batch bounds
  - Importable immutable SQLite graph fixture with DB/WAL/SHM fingerprints
  - Pinned disposable Neo4j 5.26 exact-test gate
affects: [01-02, 01-03, 01-05, 01-06, 01-07]
actuals:
  tokens: 3994
  tasks: 2
  commits: 3
tech-stack:
  added: []
  patterns: [test-only protocol recorder, canonical SQLite fixture fingerprint, exact named integration gate]
key-files:
  created:
    - internal/neo4jprojection/protocol_test.go
    - internal/testutil/graphfixture/fixture.go
    - internal/testutil/graphfixture/fixture_test.go
    - internal/neo4jprojection/neo4j_integration_test.go
    - scripts/test-neo4j.sh
  modified: []
key-decisions:
  - "Keep graphfixture independent of store_sqlite so downstream same-package tests can open the fixture without an import cycle."
  - "Register future real-server scenarios now, and make the final mandatory gate fail while any remain skipped."
patterns-established:
  - "Exact test gate: validate identifier grammar, list exact test, require grep -qx match, then run anchored selector."
  - "Fixture immutability: compare DB/WAL/SHM bytes and canonical sorted graph rows."
requirements-completed: [NEO-01, NEO-02, NEO-03, NEO-04, NEO-05, NEO-06, NEO-07, NEO-08, NEO-09, NEO-10, NEO-11, NEO-12, NEO-13, NEO-14, NEO-15, NEO-16, NEO-17, STORE-01, STORE-02, STORE-03, STORE-04, STORE-05]
coverage:
  - id: D1
    description: "Projection protocol and authoritative scoped SQLite fixtures"
    verification:
      - kind: unit
        ref: "GOWORK=off go test ./internal/testutil/graphfixture ./internal/neo4jprojection -run '^(TestProjectionFixture|TestProjectionProtocolHarness)$' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Pinned disposable Neo4j gate with strict exact-test selection"
    verification:
      - kind: integration
        ref: "bash -n scripts/test-neo4j.sh && GOWORK=off go test ./internal/neo4jprojection -run '^(TestNeo4jIntegrationGate|TestNeo4jScriptExactRunExecutesKnownTest|TestNeo4jScriptExactRunRejectsMissingTest)$' -count=1"
        status: pass
    human_judgment: false
duration: 6 min
completed: 2026-09-21
status: complete
---

# Phase 1 Plan 1: Wave 0 Neo4j Validation Harness Summary

**Deterministic projection protocol and scoped SQLite fixtures now feed a pinned, timeout-bounded Neo4j 5.26 integration gate with exact named-test enforcement.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-09-21T20:47:46Z
- **Completed:** 2026-09-21T20:53:15Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added a projection protocol recorder covering ordered phases, transient/permanent/activation/cleanup/commit-uncertain failures, cancellation, secret canaries, and maximum batch size.
- Added a raw SQLite fixture that separates exact-owner and neighboring-owner data and compares main/WAL/SHM fingerprints plus canonical graph rows.
- Added a self-cleaning disposable Neo4j 5.26 script that rejects invalid or absent named tests before Docker startup and runs exact anchored selectors under bounded deadlines.

## Task Commits

1. **Task 1: Create protocol and authoritative SQLite fixtures** - `e66eeb16` (test)
2. **Task 2: Create the mandatory disposable Neo4j gate** - `0bc80c07` (test)
3. **Task 2 correction: Fail incomplete mandatory scenarios** - `7fac3d45` (fix)

## Files Created/Modified

- `internal/neo4jprojection/protocol_test.go` - Protocol operation recorder and injected-outcome contract.
- `internal/testutil/graphfixture/fixture.go` - Raw SQLite fixture, canonical snapshot, and sidecar fingerprint API.
- `internal/testutil/graphfixture/fixture_test.go` - Determinism, ownership isolation, and fingerprint self-test.
- `internal/neo4jprojection/neo4j_integration_test.go` - Integration scenario registry and script contract tests.
- `scripts/test-neo4j.sh` - Pinned isolated disposable Neo4j gate.

## Decisions Made

- Kept `graphfixture` independent of `store_sqlite`; consumers construct production objects from its path without introducing package cycles.
- The Wave 0 gate permits registered future scenarios to skip during ordinary tests, but `GORTEX_NEO4J_REQUIRE_SCENARIOS=1` makes those skips fatal for the final mandatory phase gate.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stabilized physical fingerprints after canonical reads**
- **Found during:** Task 1 verification
- **Issue:** Opening the fixture read-only could initialize a WAL sidecar before physical hashing, making two snapshots differ.
- **Fix:** Canonicalize first, then hash the settled DB/WAL/SHM state.
- **Files modified:** `internal/testutil/graphfixture/fixture.go`
- **Verification:** `TestProjectionFixture` passes repeatedly.
- **Committed in:** `e66eeb16`

**2. [Rule 2 - Missing Critical] Made mandatory scenario skips observable failures**
- **Found during:** Task 2 self-review
- **Issue:** Registered production scenarios could remain skipped and let a future mandatory gate report success.
- **Fix:** Added a required-scenarios mode that turns every unimplemented scenario into a failure.
- **Files modified:** `internal/neo4jprojection/neo4j_integration_test.go`, `scripts/test-neo4j.sh`
- **Verification:** Focused script contract suite passes; the final gate can opt into fail-on-skip behavior.
- **Committed in:** `7fac3d45`

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical functionality)
**Impact on plan:** Both fixes enforce the fixture stability and no-silent-skip contracts without adding production behavior.

## Known Stubs

- `internal/neo4jprojection/neo4j_integration_test.go:27` - Real-server scenario bodies are intentionally registered for Plans 01-03 and 01-06; required-scenarios mode fails until those consumers implement them.

## Issues Encountered

- Gortex change detection did not include newly untracked files before their commits, so verification relied on focused tests, diff checks, and post-commit graph refresh rather than its initial unstaged report.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 01-02 and 01-03 can import the scoped fixture and target the package-local protocol harness.
- The disposable gate is ready for production tracer execution; its mandatory scenario flag deliberately remains red until later plans implement all registered scenarios.

## Self-Check: PASSED

- All five created files exist.
- Task commits `e66eeb16`, `0bc80c07`, and `7fac3d45` exist.
- Both task verification commands and the combined focused package suite pass.
- Production import scan found no importer of `internal/testutil/graphfixture`.

---
*Phase: 01-general-neo4j-projection-command*
*Completed: 2026-09-21*
