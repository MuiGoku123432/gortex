---
phase: 01-general-neo4j-projection-command
plan: 07
subsystem: database
tags: [neo4j, cobra, mcp, adapter-parity, progress, cancellation]
requires:
  - phase: 01-06
    provides: atomic generation activation, bounded cleanup, and truthful incomplete results
provides:
  - Explicit `gortex neo4j push` command with human and shared JSON output
  - Mutating `neo4j_push` MCP tool over the same normalized request and result model
  - Cancellation-aware daemon relay and invocation-local Neo4j driver construction
affects: [phase-verification, neo4j-operations]
actuals:
  tokens: 8412
  tasks: 2
  commits: 4
tech-stack:
  added: []
  patterns: [shared adapter normalization, fail-closed concrete scope, invocation-local external transport, structured incomplete result]
key-files:
  created:
    - cmd/gortex/neo4j.go
    - cmd/gortex/neo4j_test.go
    - internal/mcp/tools_neo4j.go
    - internal/mcp/tools_neo4j_test.go
  modified:
    - cmd/gortex/cli_daemon.go
    - internal/daemon/mutating.go
    - internal/mcp/facade_registry.go
    - internal/mcp/facade_tools_test.go
    - internal/mcp/server.go
    - internal/neo4jprojection/neo4j.go
    - internal/neo4jprojection/service.go
key-decisions:
  - "Normalize CLI and MCP input through one neo4jprojection.Request contract before any profile resolution, snapshot read, or driver construction."
  - "Classify neo4j_push as an external write while retaining dry-run as an argument-level non-mutating path with no second confirmation field."
patterns-established:
  - "CLI JSON is the shared Result JSON; human output is a concise presentation of the same result."
  - "Profile resolution and official-driver construction occur only inside an explicit MCP push invocation."
requirements-completed: [NEO-01, NEO-02, NEO-03, NEO-09, NEO-10, NEO-11, NEO-12, NEO-17, STORE-03, STORE-04, STORE-05]
coverage:
  - id: D1
    description: "Explicit CLI push with required profile/scope, human and JSON output, cancellation, and incomplete failure handling"
    requirement: NEO-01
    verification:
      - kind: unit
        ref: "cmd/gortex/neo4j_test.go#TestNeo4jPushFlags|TestNeo4jPushJSON|TestNeo4jPushProgress|TestNeo4jPushIncomplete"
        status: pass
    human_judgment: false
  - id: D2
    description: "CLI/MCP normalized request and Result parity with mutating intent and ordinary no-server startup"
    requirement: NEO-02
    verification:
      - kind: integration
        ref: "internal/mcp/tools_neo4j_test.go#TestNeo4jParity|TestNeo4jScope|TestNeo4jMutation|TestNeo4jNoServer"
        status: pass
      - kind: integration
        ref: "scripts/test-neo4j.sh --timeout-seconds 240"
        status: pass
    human_judgment: false
duration: 37 min
completed: 2026-09-21
status: complete
---

# Phase 1 Plan 7: Neo4j CLI/MCP Projection Adapters Summary

**Explicit CLI and MCP Neo4j pushes now share one fail-closed request/result contract, safe output, cancellation, progress, dry-run, and truthful activation-versus-cleanup status.**

## Performance

- **Duration:** 37 min
- **Started:** 2026-09-21T22:45:50Z
- **Completed:** 2026-09-21T23:23:00Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- Added `gortex neo4j push` with required named profile, namespace, workspace, project, and one-or-more repositories plus bounded timeout/batch flags.
- Added shared JSON output, concise secret-safe human output, command-context cancellation, and non-zero incomplete outcomes with ordinary-rerun guidance.
- Registered `neo4j_push` as an external mutation, attached standard MCP progress, preserved dry-run without confirmation, and delayed profile/driver work until explicit invocation.
- Proved adapter request/result parity, concrete-scope failure, mutation intent, ordinary no-server startup, focused regressions, build, and disposable Neo4j acceptance.

## Task Commits

1. **Task 1 RED: Add failing Neo4j push CLI tests** - `1f4e9355` (test)
2. **Task 1 GREEN: Add Neo4j push CLI** - `8b28aba3` (feat)
3. **Task 2: Add Neo4j push MCP parity** - `15b9234c` (feat)
4. **Task 2 fix: Complete adapter integration** - `36e2becf` (fix)

## Files Created/Modified

- `cmd/gortex/neo4j.go` - Nested push command, explicit flags, daemon adapter, and human/JSON rendering.
- `cmd/gortex/neo4j_test.go` - Exact CLI flag, JSON, progress, cancellation, and incomplete tests.
- `cmd/gortex/cli_daemon.go` - Context cancellation now interrupts blocking daemon tool calls.
- `internal/mcp/tools_neo4j.go` - Tool schema, exact-scope validation, profile/driver construction, progress, and structured results.
- `internal/mcp/tools_neo4j_test.go` - Adapter parity, scope, mutation, and no-server guards.
- `internal/mcp/server.go` - Normal registration plus injected test seam; no startup driver construction.
- `internal/daemon/mutating.go` - External-write intent classification.
- `internal/mcp/facade_registry.go` - External-write catalog mapping.
- `internal/mcp/facade_tools_test.go` - Updated complete legacy catalog count.
- `internal/neo4jprojection/service.go` - Public normalized request/result JSON contract and safe incomplete metadata.
- `internal/neo4jprojection/neo4j.go` - Invocation-specific transaction and retry timeout wiring.

## Decisions Made

- Both adapters normalize one request model, including deterministic repository ordering and exact-owner derivation, before invoking the service.
- The MCP tool is always write-classified because `dry_run=false` mutates Neo4j; no confirmation argument exists.
- Activation completeness remains independent from cleanup completeness, so post-activation cleanup interruptions stay truthful.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Propagated cancellation through the CLI daemon relay**
- **Found during:** Task 1
- **Issue:** The existing daemon executor ignored its context while waiting for an MCP response, preventing CLI cancellation from reaching the caller promptly.
- **Fix:** Made `daemonExecutor.CallTool` close the connection and return the context error when cancelled.
- **Files modified:** `cmd/gortex/cli_daemon.go`
- **Verification:** `TestNeo4jPushIncomplete` cancellation case passes.
- **Committed in:** `8b28aba3`

**2. [Rule 3 - Blocking] Added new tool to the complete facade migration catalog**
- **Found during:** Full race suite
- **Issue:** The repository-wide catalog guard rejected the newly registered legacy tool until it had an external-write facade mapping.
- **Fix:** Mapped `neo4j_push` into the existing external-write facade and updated the exact catalog count.
- **Files modified:** `internal/mcp/facade_registry.go`, `internal/mcp/facade_tools_test.go`
- **Verification:** `TestFacadeRegistryCoversRegisteredLegacyCatalog` and `TestFacadeEffectBoundaryParity` pass.
- **Committed in:** `36e2becf`

**Total deviations:** 2 auto-fixed (1 missing critical functionality, 1 blocking integration guard).
**Impact on plan:** Both fixes were required for cancellation correctness and repository tool-catalog integrity; no product scope was added.

## Issues Encountered

- The broad `GOWORK=off go test -race ./...` gate again hit the pre-existing 10-minute timeouts in `internal/graph/store_sqlite` and `internal/indexer`. Before the facade fix it also exposed the new catalog guard, which was fixed and focused-tested. Changed-package focused suites, build, ordinary no-server behavior, and the mandatory disposable Neo4j lane pass.

## User Setup Required

None - ordinary Gortex startup remains Neo4j-independent. Actual pushes require an explicitly configured named profile and its referenced credential environment variables.

## Next Phase Readiness

- Phase 1 implementation is complete and ready for phase verification.
- The known unrelated broad-race timeout remains visible for project-level follow-up.

## Self-Check: PASSED

- All four created adapter/test files and seven modified integration files exist.
- Commits `1f4e9355`, `8b28aba3`, `15b9234c`, and `36e2becf` exist in order.
- All eight exact CLI/MCP tests are discoverable and pass.
- Focused Neo4j/projection regressions, CLI build, facade catalog guards, and disposable Neo4j acceptance pass.
- No credential/URI flags, resume command, TODO, FIXME, placeholder, or ordinary-startup Neo4j dependency was introduced.

---
*Phase: 01-general-neo4j-projection-command*
*Completed: 2026-09-21*
