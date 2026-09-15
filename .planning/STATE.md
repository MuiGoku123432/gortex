---
gsd_state_version: "1.0"
milestone: v1.0
milestone_name: Deterministic COBOL Graph Extraction and AI Enrichment Foundation
status: planning
last_updated: "2026-09-15T19:00:00Z"
last_activity: 2026-09-15
progress:
  total_phases: 8
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-15)

**Core value:** A trustworthy graph representation of a mainframe estate that modernization cutover decisions can be made against -- deterministic and reproducible first, enriched and simulated later.
**Current focus:** Phase 1 -- Thin Deterministic Native Tracer and Minimum Contracts

## Current Position

Phase: 1 of 8 (Thin Deterministic Native Tracer and Minimum Contracts)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-09-15 -- Roadmap revised so the first implementation phase delivers the thin deterministic tracer; 103/103 requirements remain mapped

Progress: [----------] 0%

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

- [Phase 1]: The first implementation phase delivers the thin native tracer through the existing extractor, SQLite, CLI, and MCP, with only the minimum parser reproducibility, identity, provenance, and SQLite-authority contracts needed to trust it.
- [Phase 1]: Parser behavior must be reproducibly equivalent to baseline `97ac9f1`; silent fallback is forbidden.
- [Phase 4]: `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` remain distinct deterministic findings.
- [Phase 5]: Deterministic extraction, stable identity, explicit unresolved modeling, native query acceptance, and deterministic lifecycle gates must pass before AI work begins.
- [Phases 6-8]: AI-facing work requires the AI-SPEC workflow before implementation planning.
- [Phase 8]: SQLite remains authoritative; v1.0 may validate a one-way Cypher snapshot but will not implement Neo4j synchronization.

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 1]: Portable parser dependency, grammar fingerprint equivalent to `97ac9f1`, and the minimum trustworthy tracer contract require a concrete design.
- [Phase 3]: Copybook/library search order and canonical cross-repository resolution domains need corpus evidence.
- [Phase 6]: Approved providers, deployments, retention, redaction, and authorization ownership remain policy decisions.
- [Phase 7]: Claim-ledger retention, encryption, reviewer authority, and version-scoped confirmation require AI-SPEC decisions.
- [Phase 8]: Native query thresholds and representative estate scale must be defined before the projection decision.

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Storage | Neo4j synchronization or authority | Deferred pending measured native query deficiency and a future ADR | v1.0 planning |
| Mainframe breadth | Full JCL symbolic resolution and behavioral/live twin | Future milestone | v1.0 planning |

## Session Continuity

Last session: 2026-09-15
Stopped at: Roadmap reordered around a tracer-first Phase 1 with 103/103 requirements mapped; Phase 1 is ready for planning
Resume file: None
