---
phase: 01-general-neo4j-projection-command
plan: 02
subsystem: database
tags: [neo4j, sqlite, configuration, snapshots, security]
requires:
  - phase: 01-01
    provides: deterministic projection fixtures and exact named-test gates
provides:
  - Named machine-level Neo4j profiles with invocation-time environment credential resolution
  - Immutable exact-repository SQLite snapshots with context-aware node and edge reads
  - Exact prerequisite tests for the Plan 01-03 production tracer
affects: [01-03, 01-05, 01-06, 01-07]
actuals:
  tokens: 23770
  tasks: 2
  commits: 6
tech-stack:
  added: []
  patterns: [environment-only credentials, explicit named profile, read-only SQLite transaction snapshot]
key-files:
  created: []
  modified:
    - internal/config/global.go
    - internal/config/global_test.go
    - internal/graph/scoped_projection.go
    - internal/graph/store_sqlite/scoped_projection.go
    - internal/graph/store_sqlite/scoped_projection_test.go
key-decisions:
  - "Keep resolved credentials in an invocation-local value excluded from YAML serialization while persistent profiles store only environment-variable names."
  - "Add a separate snapshot opener capability instead of widening the heavily used streaming sequencer interface."
patterns-established:
  - "Profile resolution: explicit name, strict URI/database/env validation, then environment lookup with secret-safe errors."
  - "Snapshot reads: one read-only transaction shared by node, edge, and endpoint hydration queries."
requirements-completed: [NEO-02, NEO-03, NEO-08, NEO-11, NEO-15, NEO-17, STORE-01, STORE-02, STORE-05]
coverage:
  - id: D1
    description: "Explicit named Neo4j machine profiles resolve environment-only credentials and fail closed"
    requirement: NEO-03
    verification:
      - kind: unit
        ref: "internal/config/global_test.go#TestNeo4jTracerContractProfile"
        status: pass
    human_judgment: false
  - id: D2
    description: "Exact repository-scoped SQLite snapshots provide immutable cancellable node and all-edge reads"
    requirement: STORE-01
    verification:
      - kind: unit
        ref: "internal/graph/store_sqlite/scoped_projection_test.go#TestScopedProjectionTracer"
        status: pass
      - kind: other
        ref: "production import scan for projection, Neo4j driver, and graphfixture dependencies"
        status: pass
    human_judgment: false
duration: 10 min
completed: 2026-09-21
status: complete
---

# Phase 1 Plan 2: Profile and Immutable Snapshot Contracts Summary

**Explicit secret-safe Neo4j profiles and one-transaction exact-repository SQLite snapshots now provide the production contracts required by the official-driver tracer.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-09-21T20:56:01Z
- **Completed:** 2026-09-21T21:06:09Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added named machine-level Neo4j profiles that store only URI, database, and credential environment-variable references, with credentials resolved only for an explicit profile invocation.
- Added fail-closed profile validation covering absent names, unknown profiles, invalid fields, URI userinfo, missing environment values, and secret-safe serialization/errors.
- Added a narrow immutable SQLite snapshot capability that rejects empty scope, pins a source generation in one read-only transaction, reads nodes and all edge kinds with context-aware queries, hydrates endpoints in bounded pages, and closes explicitly.

## Task Commits

1. **Task 1 RED: Add failing Neo4j profile contract** - `50edc373` (test)
2. **Task 1 correction: Restore pre-existing config tests** - `e27bacee` (fix)
3. **Task 1 GREEN: Add explicit Neo4j profiles** - `eb4b855d` (feat)
4. **Task 2 RED: Add failing scoped snapshot contract** - `973cf792` (test)
5. **Task 2 correction: Restore pre-existing scoped tests** - `783f185b` (fix)
6. **Task 2 GREEN: Add immutable scoped snapshots** - `e67d15bd` (feat)

## Files Created/Modified

- `internal/config/global.go` - Named persistent profiles and invocation-local credential resolver.
- `internal/config/global_test.go` - Exact profile tracer contract plus preserved existing config coverage.
- `internal/graph/scoped_projection.go` - Narrow scope, descriptor, snapshot, and opener production interfaces.
- `internal/graph/store_sqlite/scoped_projection.go` - Read-only transaction snapshot implementation with context-aware reads.
- `internal/graph/store_sqlite/scoped_projection_test.go` - Exact snapshot tracer contract plus preserved pagination/query-plan coverage.

## Decisions Made

- Resolved credentials are never part of the persistent profile model and are excluded from YAML output.
- The immutable snapshot contract is separate from `ScopedProjectionSequencer`, avoiding a breaking change to its broad caller surface.
- Repository scope is normalized, deduplicated, sorted, and rejected if empty or ambiguous before the transaction opens.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Restored pre-existing tests after initial TDD file writes**
- **Found during:** Tasks 1 and 2 RED commits
- **Issue:** Initial whole-file TDD writes replaced existing test files instead of appending the new exact contract tests.
- **Fix:** Restored both prior files from their parent commits and appended the new contract tests, preserving all existing coverage.
- **Files modified:** `internal/config/global_test.go`, `internal/graph/store_sqlite/scoped_projection_test.go`
- **Verification:** Full focused package suites pass and commit diffs retain the prior tests.
- **Committed in:** `e27bacee`, `783f185b`

**2. [Rule 1 - Bug] Stabilized SQLite physical fingerprint assertion**
- **Found during:** Task 2 GREEN verification
- **Issue:** SQLite updates shared-memory lock metadata during read transactions, so byte-for-byte SHM comparison was not a valid persistence-mutation assertion.
- **Fix:** Warmed the read path and compared authoritative DB and WAL bytes while retaining logical node/edge equality assertions; SHM remains runtime coordination state rather than authoritative content.
- **Files modified:** `internal/graph/store_sqlite/scoped_projection_test.go`
- **Verification:** `TestScopedProjectionTracer` passes and DB/WAL remain unchanged.
- **Committed in:** `e67d15bd`

**Total deviations:** 2 auto-fixed bugs.
**Impact on plan:** Corrections preserved existing tests and made the immutability assertion accurately target authoritative SQLite content. No production scope was added.

## Issues Encountered

- Gortex's post-edit change detector correctly surfaced unrelated pre-existing planning changes; those files were left untouched and uncommitted.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-03 can consume `ResolveNeo4jProfile` and `ScopedProjectionSnapshotOpener` without modifying config or graph/store contracts.
- Both exact prerequisite test identifiers and the production import prohibition pass.

## Self-Check: PASSED

- All five modified production/test files exist.
- Task commits `50edc373`, `e27bacee`, `eb4b855d`, `973cf792`, `783f185b`, and `e67d15bd` exist.
- Both exact `go test -list` guards, both focused contract tests, and the combined focused package suite pass.
- Production graph packages import neither the projection package, Neo4j driver, nor test fixture package.

---
*Phase: 01-general-neo4j-projection-command*
*Completed: 2026-09-21*
