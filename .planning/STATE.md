---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: Neo4j Projection and Mainframe Graph Foundation
current_phase: 01
current_phase_name: General Neo4j Projection Command
status: executing
stopped_at: Completed 01-01-PLAN.md
last_updated: "2026-09-21T20:54:43.675Z"
last_activity: 2026-09-21
last_activity_desc: Phase 01 execution started
progress:
  total_phases: 9
  completed_phases: 0
  total_plans: 7
  completed_plans: 1
  percent: 0
state_head: 3c44f1fac75c26bd579d1fd96270e74f36e7d699
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-21)

**Core value:** A trustworthy graph representation of a mainframe estate that modernization cutover decisions can be made against -- deterministic and reproducible first, enriched and simulated later.
**Current focus:** Phase 01 — General Neo4j Projection Command

## Current Position

Phase: 01 (General Neo4j Projection Command) — EXECUTING
Plan: 2 of 7
Status: Ready to execute
Last activity: 2026-09-21 — Phase 01 execution started

Progress: [█░░░░░░░░░] 14%

## Performance Metrics

**Velocity:**

- Total plans completed: 0
- Average duration: -
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**

- Last 5 plans: none
- Trend: No execution data

*Updated after each plan completion*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P01 | 6 min | 2 tasks | 5 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Phase 1]: The immediate priority is a general, language-agnostic CLI/MCP operation that projects an explicitly scoped SQLite snapshot into Neo4j.
- [Phase 1]: SQLite remains sole authority; projection is manual and rebuildable, with no index-time dual-write, reverse writes, or Neo4j dependency for ordinary Gortex operations.
- [Phase 2]: Parser behavior must be reproducibly equivalent to baseline `97ac9f1`; the tracer resumes only when parser/grammar integration is ready.
- [Phase 5]: `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` remain distinct deterministic findings.
- [Phases 7-9]: AI-facing work requires the AI-SPEC workflow before implementation planning; existing artifacts must be renumbered by the orchestrator.
- [Phase 01]: Keep graphfixture independent of store_sqlite so downstream same-package tests can open the fixture without an import cycle.
- [Phase 01]: Register future real-server scenarios now, and make the final mandatory gate fail while any remain skipped.

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 1]: Exact command name, connection/driver syntax, and default stale-record deletion policy remain discuss-phase decisions; requirements define behavior, not those details.
- [Phase 1]: Projection design must prove stable relationship keys, Neo4j constraint support, bounded retry behavior, and scope-safe stale-record reconciliation.
- [Phase 2]: COBOL parser/grammar readiness blocks the tracer and all later grammar-dependent work.
- [Phase 4]: Copybook/library search order and canonical cross-repository resolution domains need corpus evidence.
- [Phases 7-9]: Provider authorization, retention, claim-ledger, reviewer authority, and evaluation thresholds remain policy/design decisions.

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Storage | Continuous Neo4j synchronization or Neo4j authority | Deferred; v1.0 includes only explicit scoped projection | v1.0 reprioritization |
| COBOL graph | Tracer and grammar-dependent extraction phases | Deferred until parser/grammar readiness | v1.0 reprioritization |
| Mainframe breadth | Full JCL symbolic resolution and behavioral/live twin | Future milestone | v1.0 planning |

## Session Continuity

Last session: 2026-09-21T20:54:43.668Z
Stopped at: Completed 01-01-PLAN.md
Resume file: None
