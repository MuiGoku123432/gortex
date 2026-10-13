---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 10
subsystem: testing
tags: [cobol, scope, ownership, neo4j, multi-repo, acceptance, docs]

requires:
  - phase: 04-03
    provides: graph.NewScope / Scope.Allows and OpenScopedProjectionSnapshot
  - phase: 04-05
    provides: strict ReadPrefixDiagnostics Clean and index_health.repo_ownership
  - phase: 04-12
    provides: requireSourceScope gate behind get_symbol_source
  - phase: 04-04
    provides: generic cross-repository ambiguous refusal (resolution_unresolved_reason)
  - phase: 04-09
    provides: COBOL material fingerprint restage on index.cobol change
  - phase: 04-16
    provides: the full D-04 TestCobolLifecycle_ sequence set (HeadMove last)
provides:
  - TestCobolScope_CrossSurface (SEC-01, D-16) - projection, Scope.Allows, and the MCP source gate agree on one COBOL node set
  - TestCobolLifecycle_Ownership (SEC-08, D-15) - strict ownership audit clean after the whole lifecycle
  - TestCobolLifecycle_PlaceholderCollision (SEC-03) - same-path dynamic CALL placeholders never merge facts
  - TestSensitiveMetadataKey_Phase4Keys - every new Phase 4 Meta key survives the Neo4j secret-key screen
  - Phase 4 behavior documented in docs/languages.md and docs/multi-repo.md
affects: [phase-05, phase-06, phase-07, neo4j-projection, multi-repo]

actuals:
  tokens: 5830
  tasks: 3
  commits: 3
plan_head_before: 79343f1ee7f5ae6d95231007c2100bb86e0fb0bc

tech-stack:
  added: []
  patterns:
    - "Acceptance tests over already-shipped behavior are proven able to fail by a mutation run (red) before being committed green"

key-files:
  created:
    - cmd/gortex/cobol_scope_test.go
    - internal/indexer/cobol_lifecycle_acceptance_test.go
    - internal/neo4jprojection/properties_phase4_test.go
  modified:
    - cmd/gortex/cobol_tracer_test.go
    - docs/languages.md
    - docs/multi-repo.md

key-decisions:
  - "TestCobolScope_CrossSurface calls get_symbol_source from the directory above both checkouts, which reaches both repositories, so only the explicit repo argument narrows the read; called from repository A's checkout the session binding alone already excluded B, and a mutation that dropped the repo argument did not go red"
  - "callCobolTracerTool now delegates to a new callCobolToolResult that returns (text, isError), so the cross-surface test reads the not-found refusal without a second relay helper"
  - "SEC-03 placeholder collision is closed by proof, as the plan flagged: the shared unresolved::cobol::dynamic::program::cobol/same.cbl::SAMEPGM::WS-PGM target has no node, and the two edges carry samea/ and sameb/ prefixed From IDs"

patterns-established:
  - "Mutation-proof for green-on-arrival acceptance tests: temporarily break the input (drop narrowing, inject an unowned node, prefix the target) and record the red output"

requirements-completed: [SEC-01, SEC-03, SEC-08, ID-09]

coverage:
  - id: D1
    description: "One repository allow-set yields the same COBOL program set on the projection snapshot, Scope.Allows, and get_symbol_source over the daemon relay; B programs answer 'symbol not found'; every bound CALL/COPY edge from A targets an A node"
    requirement: SEC-01
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_scope_test.go#TestCobolScope_CrossSurface"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Scope|Tracer|Breadth)_'"
        status: pass
    human_judgment: false
  - id: D2
    description: "After track, edit, delete, restore, config change, and cross-repository change, ReadPrefixDiagnostics reports owned > 0, unowned 0, misprefixed 0, Clean true"
    requirement: SEC-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_acceptance_test.go#TestCobolLifecycle_Ownership"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two repositories with the same path, program, and dynamic CALL keep two edges with distinct repository-prefixed From IDs and no node for the shared unresolved target"
    requirement: SEC-03
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_acceptance_test.go#TestCobolLifecycle_PlaceholderCollision"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every D-04 sequence has a passing TestCobolLifecycle_ test (all fifteen names present) under -race"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestCobolLifecycle_' (plus -list name check)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every new Phase 4 Meta key passes the Neo4j projection secret-key screen"
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/properties_phase4_test.go#TestSensitiveMetadataKey_Phase4Keys"
        status: pass
    human_judgment: false
  - id: D6
    description: "docs/languages.md and docs/multi-repo.md describe Phase 4 behavior accurately and concisely"
    verification:
      - kind: other
        ref: "grep resolve_from docs/languages.md docs/multi-repo.md; grep repo_ownership docs/multi-repo.md"
        status: pass
    human_judgment: true
    rationale: "The greps prove the keywords are present; whether the prose is accurate and readable is a reviewer judgment"
  - id: D7
    description: "The whole repository passes under the race detector"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -timeout 90m ./... (157 ok, 0 FAIL, exit 0)"
        status: pass
    human_judgment: false

duration: 28min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 10: Cross-Surface Scope, Ownership, and Phase Acceptance Summary

**Acceptance gate for Phase 4: one repository allow-set gives the same COBOL node set on the projection snapshot, Scope.Allows, and the get_symbol_source gate; the lifecycle corpus ends with a clean ownership audit; same-path dynamic CALL placeholders stay separate per repository; new Meta keys pass the Neo4j screen; the docs are updated; and the full race suite is green.**

## Performance

- **Duration:** 28 min
- **Started:** 2026-10-13T05:31:56Z
- **Completed:** 2026-10-13T05:59:43Z
- **Tasks:** 3
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments

- `TestCobolScope_CrossSurface` indexes the two breadth repositories into one SQLite store and serves them over the real daemon relay. For `graph.NewScope(ws, project, [prefixA])` it compares three COBOL program-ID sets: projection snapshot `ReadNodePages`, `Scope.Allows` over every node, and `get_symbol_source` with `repo: prefixA`. All three are the same 8 A programs. `b/cobol/onlyb.cbl::ONLYB` returns exactly `symbol not found: <id>`. Every bound `cobol_placeholder` edge leaving A targets an A node.
- `TestCobolLifecycle_Ownership` drives one store through edit, delete, restore, an `index.cobol` change (`resolve_from: [lifeb]` + `db2_default_schema`), and a lifeb-only cross-repository delete. The final dump equals a clean build, and `ReadPrefixDiagnostics` is clean (owned 44, unowned 0, misprefixed 0).
- `TestCobolLifecycle_PlaceholderCollision`: `samea` and `sameb` both hold `cobol/same.cbl` with SAMEPGM and `CALL WS-PGM`. Two edges reach the same unresolved target string, with `samea/` and `sameb/` From IDs, and `GetNode(target)` is nil.
- `TestSensitiveMetadataKey_Phase4Keys`: all eight Phase 4 keys are kept by `sensitiveMetadataKey`.
- Docs: the COBOL row in `docs/languages.md` now covers normalized shared resources, re-decided placeholders, `resolve_from`, union ambiguity, library rank, `aliases` qualifiers, and IN-01. `docs/multi-repo.md` gains a "COBOL resolution scope" subsection (YAML + validation rules), a "Source read and write scope" subsection, the generic `ambiguous` refusal, `repo_ownership.status`, and the upgrade note. The boundary note now points at the source gate.

## Task Commits

1. **Task 1: One allow-set, one node set across projection, resolution, and source reads** - `2e08ec50` (test)
2. **Task 2: Ownership, placeholder isolation, D-04 completeness, and Meta-key screening** - `a45d29ae` (test)
3. **Task 3: Document Phase 4 behavior and pass the full suite** - `8960950c` (docs)

## TDD Gate Compliance

Tasks 1 and 2 are `tdd="true"`, but they are acceptance tests for behavior that plans 04-03 through 04-16 already shipped, so they passed on arrival (tdd.md: "feature may already exist"). Instead of a separate failing commit, each assertion was proven able to fail with a temporary mutation, then reverted. Observed `go test` output:

- **Task 1:** I removed the `repo` argument and called from the workspace-root CWD. `--- FAIL: TestCobolScope_CrossSurface ... Messages: projection vs get_symbol_source` (actual set gained `b/cobol/onlyb.cbl::ONLYB`, len 9 vs 8).
- **Task 2 ownership:** I injected an unowned `cobol/ghost.cbl::GHOST` node before the audit. `--- FAIL: TestCobolLifecycle_Ownership ... Should be zero, but was 1 ... (owned=44 unowned=1 e.g. cobol/ghost.cbl::GHOST)`.
- **Task 2 collision:** I changed the target to the `lifea::unresolved::…` form. `--- FAIL: TestCobolLifecycle_PlaceholderCollision ... "[]" should have 2 item(s), but has 0`.

`gsd-tools check tdd-red-evidence` cannot parse `go test` output, so the evidence above is the raw failure text.

## Files Created/Modified

- `cmd/gortex/cobol_scope_test.go` - TestCobolScope_CrossSurface (SEC-01, D-16)
- `cmd/gortex/cobol_tracer_test.go` - callCobolTracerTool now delegates to new callCobolToolResult (returns tool-error results)
- `internal/indexer/cobol_lifecycle_acceptance_test.go` - TestCobolLifecycle_Ownership, TestCobolLifecycle_PlaceholderCollision
- `internal/neo4jprojection/properties_phase4_test.go` - TestSensitiveMetadataKey_Phase4Keys
- `docs/languages.md` - COBOL row: Phase 4 binding, naming, and IN-01 behavior
- `docs/multi-repo.md` - COBOL resolution scope, source read/write scope, ambiguity, ownership health, upgrade note

## Decisions Made

- The cross-surface source reads come from the directory above both checkouts. From repository A's checkout, the session binding already excluded B, so a test there could not tell whether the `repo` argument works. From the workspace root, only `repo` narrows the read, and the mutation proves it.
- The relay helper was extended, not forked: `callCobolTracerTool` keeps its signature and existing callers, and wraps `callCobolToolResult`.
- SEC-03 placeholder collision is closed by proof, not by prefixing target strings, as the plan flagged.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Extended the shared relay helper in cmd/gortex/cobol_tracer_test.go**
- **Found during:** Task 1
- **Issue:** `callCobolTracerTool` fails the test on any tool error, so the plan's check that a B program returns the not-found text could not be written with it.
- **Fix:** Moved the body into `callCobolToolResult(t, socket, cwd, tool, args) (text, isError)`. `callCobolTracerTool` now calls it and keeps `require.False(isError)`. Existing callers are unchanged.
- **Files modified:** cmd/gortex/cobol_tracer_test.go (outside the plan's files_modified list)
- **Verification:** `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Scope|Tracer|Breadth)_'` ok
- **Committed in:** 2e08ec50

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Test-only helper change needed for the planned assertion. No product code changed, and no defect turned up in any surface.

## Issues Encountered

- The first Task 1 mutation (dropping `repo` while calling from repository A's checkout) stayed green, because the session binding already narrowed reads. I moved the call to the workspace-root CWD so the test checks the `repo` argument itself. This is the decision above.

## Verification

- Task 1 verify: name in `-list`, `-race` COBOL Scope/Tracer/Breadth tests ok (36s), gofmt clean.
- Task 2 verify: all fifteen `TestCobolLifecycle_` names present, `TestSensitiveMetadataKey_Phase4Keys` present, `-race` lifecycle tests ok (113s), `-race` neo4jprojection ok, gofmt clean.
- Task 3 verify: the doc greps match (`resolve_from` 1 in languages.md and 3 in multi-repo.md; `repo_ownership` at multi-repo.md:440). `go build ./...` and the listed `go vet` packages pass. Full `GOWORK=off go test -race -timeout 90m ./...` finished with 157 packages ok, 0 FAIL lines, exit 0. Slow packages: indexer 1018s, store_sqlite 725s, mcp 536s, cmd/gortex 170s. `-timeout 90m` was passed because these exceed Go's default 10-minute limit. The run was not split, and `./cmd/gortex` passed inside it.
- Plan-level: the diff from the base (`79343f1e`) touches the five planned files plus `cmd/gortex/cobol_tracer_test.go` (deviation 1).

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 4 success criteria SC1 to SC5 now have tests behind them, and the full race suite is green.
- Open items in `deferred-items.md` from 04-04 and 04-15 are unchanged: a stale ambiguous stamp with zero candidates, `tryRemoteStitch` after a local refusal, and a stale `cross_repo_calls` edge when a cross-repo CALL becomes unbound without eviction. They are recorded for phase verification. This plan did not touch them.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*

## Self-Check: PASSED
