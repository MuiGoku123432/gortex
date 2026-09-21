---
gsd_state_version: "1.0"
milestone: v1.0
current_phase: 1
current_phase_name: General Neo4j Projection Command
status: planning
stopped_at: Phase 1 context gathered
last_updated: "2026-09-21T17:54:05.491Z"
last_activity: 2026-09-21
last_activity_desc: Immediate priority changed to a general SQLite-to-Neo4j projection; 120/120 requirements mapped, with COBOL phases deferred for grammar readiness
state_head: aab0141840b5ac45bd4ee5224067f7c285cd00c7
progress:
  total_phases: 9
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
milestone_name: Neo4j Projection and Mainframe Graph Foundation
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-21)

**Core value:** A trustworthy graph representation of a mainframe estate that modernization cutover decisions can be made against -- deterministic and reproducible first, enriched and simulated later.
**Current focus:** Phase 1 -- General Neo4j Projection Command

## Current Position

Phase: 1 of 9 (General Neo4j Projection Command)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-09-21 -- Immediate priority changed to a general SQLite-to-Neo4j projection; 120/120 requirements mapped, with COBOL phases deferred for grammar readiness

Progress: [░░░░░░░░░░] 0%

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

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Phase 1]: The immediate priority is a general, language-agnostic CLI/MCP operation that projects an explicitly scoped SQLite snapshot into Neo4j.
- [Phase 1]: SQLite remains sole authority; projection is manual and rebuildable, with no index-time dual-write, reverse writes, or Neo4j dependency for ordinary Gortex operations.
- [Phase 2]: Parser behavior must be reproducibly equivalent to baseline `97ac9f1`; the tracer resumes only when parser/grammar integration is ready.
- [Phase 5]: `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` remain distinct deterministic findings.
- [Phases 7-9]: AI-facing work requires the AI-SPEC workflow before implementation planning; existing artifacts must be renumbered by the orchestrator.

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

Last session: 2026-09-21T17:54:05.483Z
Stopped at: Phase 1 context gathered
Resume file: .planning/phases/01-general-neo4j-projection-command/01-CONTEXT.md
