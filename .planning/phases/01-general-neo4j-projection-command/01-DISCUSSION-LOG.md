# Phase 1: General Neo4j Projection Command - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md; this log preserves the alternatives considered.

**Date:** 2026-09-21
**Phase:** 1-General Neo4j Projection Command
**Areas discussed:** Command and configuration, Snapshot replacement, Projected graph shape, Run experience

---

## Command and configuration

| Decision | Alternatives considered | Selected |
|----------|-------------------------|----------|
| CLI command | `gortex neo4j push`; `gortex export --to neo4j`; `gortex graph push` | `gortex neo4j push` |
| MCP tool | `neo4j_push`; `push_graph`; extend `export_graph` | `neo4j_push` |
| Connection settings | Named profiles + environment secrets; flags + environment secrets; connection URL | Named profiles + environment secrets |
| Missing profile | Require explicit profile; configured default; localhost fallback | Require explicit profile |

**Notes:** The command must be clearly external and mutating. Connection credentials must not appear in command history, output, logs, plans, or summaries.

---

## Snapshot replacement

| Decision | Alternatives considered | Selected |
|----------|-------------------------|----------|
| Stale records | Replace selected scope; upsert only; explicit `--prune` | Replace selected scope |
| Ownership | Projection namespace; dedicated database; labels only | Projection namespace |
| Failed replacement | Keep prior active snapshot; expose partial generation; clear then rebuild | Keep prior active snapshot |
| Concurrent pushes | Reject second push; queue; last completion wins | Reject second push |

**Notes:** Replacement is constrained to records owned by the exact projection namespace and selected Gortex scope. A pending generation cannot become active until every batch succeeds.

---

## Projected graph shape

| Decision | Alternatives considered | Selected |
|----------|-------------------------|----------|
| Node labels | Base + kind label; base only; kind only | Base + kind label |
| Relationships | Typed relationship; generic edge type; duplicate both | Typed relationship |
| Properties | All supported; stable core only; core + allow-list | All supported |
| Unresolved targets | Project labeled placeholders; omit; flatten onto source | Project labeled placeholders |

**Notes:** Preserve Neo4j exploration ergonomics without losing the original Gortex kind or epistemic-boundary information.

---

## Run experience

| Decision | Alternatives considered | Selected |
|----------|-------------------------|----------|
| Dry run | Connect, inspect, and plan; local plan only; syntax only | Connect, inspect, and plan |
| Progress | Human + JSON; final summary only; per-batch verbose | Human + JSON |
| Retry | Rerun same command; explicit resume ID; restart generation | Rerun same command |
| MCP mutation | Explicit confirmation; mutate unless dry-run; dry-run only | Mutate unless `dry_run` is true |
| Incomplete push | Failure with prior active retained; warning success; partial success | Failure with prior active retained |

**Notes:** CLI and MCP use one result model. Failed intent is reported as failure even though the previous complete snapshot remains available.

## Agent's Discretion

- Driver choice, package layout, profile field names, timeout/batch defaults, progress cadence, and internal pending-generation representation.

## Deferred Ideas

- Continuous/incremental synchronization and change-data capture.
- Neo4j authority or reverse writes.
- COBOL graph extraction until grammar readiness.
