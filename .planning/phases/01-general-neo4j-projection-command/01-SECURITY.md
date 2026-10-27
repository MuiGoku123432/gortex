---
phase: 01-general-neo4j-projection-command
status: secured
threats_total: 47
threats_closed: 47
threats_open: 0
asvs_level: 1
blocking_threshold: high
audited: 2026-09-22
head: c944732e
---

# Phase 01 Security Verification

## Verdict

**SECURED** -- all high and critical threats are mitigated. The three low-severity accepted risks are recorded below with owners and review conditions.

## Verified Controls

- Exact workspace, project, repository, owner, generation, operation, and attempt scope is enforced at SQLite reads and Neo4j mutations.
- Every staging, activation, reconciliation, and abort mutation is fenced by exact attempt ownership and lease state.
- Pending completion compares independent intended counts with observed target counts before activation.
- Cleanup is bounded, relationships-first, cancellation-aware, and reports fence loss or stale records truthfully.
- Remote Neo4j profiles require CA-validated encrypted schemes; insecure and self-signed schemes are loopback-only.
- Credentials are environment-referenced and excluded from config serialization, errors, progress, results, logs, and projected properties.
- Metadata is bounded by depth, element count, and exact retained-envelope bytes before driver dispatch.
- Transaction/retry limits are capped at 30 seconds, operation timeout at 30 minutes, and projection batches at 500 records.
- SQLite remains read-only authority. No index-time dual write, reverse write, background synchronization, or ordinary startup dependency exists.
- The pinned Neo4j 5.26 disposable suite, focused race tests, and CLI build pass.

## Least-Privilege Deployment

Use a dedicated projection principal restricted to the configured Neo4j database. Grant only database access, constraint inspection/creation, and graph writes required by `neo4j_push`. Do not grant DBMS-wide administration, user management, cross-database access, or authority over unrelated data. Keep credentials in profile-referenced environment variables, use separate credentials per environment, and require CA-validated encryption for non-loopback targets.

## Accepted Risks

| Threat ID | Severity | Risk | Rationale | Owner | Review condition |
|---|---|---|---|---|---|
| 01-01-E | low | The disposable local Neo4j test principal has broad authority inside its temporary database. | The harness creates an isolated random container/database, exposes no reusable credentials, and destroys the environment after the bounded test. | Phase 01 maintainers | Revisit if the harness targets a shared or persistent Neo4j environment. |
| T-01-09E | low | The pure mapping package performs no authorization checks. | Mapping is deterministic and side-effect free; authorization and scope enforcement occur before mapping at service/store boundaries. Reserved owner fields cannot be overwritten by metadata. | Neo4j projection maintainers | Revisit if mapping gains I/O, policy decisions, or direct request handling. |
| T-01-25 | low | CLI and MCP presentation adapters add no independent authorization layer. | Both adapters require explicit fields and route through shared normalization, canonical scope resolution, and one projection service without bypass controls. | CLI/MCP maintainers | Revisit if an adapter gains a separate transport, credential source, or service bypass. |

## Residual Defense-In-Depth

A single disposable composition test spanning CLI, daemon relay, registered MCP handler, named profile, and Neo4j is not yet present. All component seams and the real driver path are independently covered; this is retained as a non-blocking defense-in-depth test opportunity.

## Evidence

- `01-REVIEW.md`: clean, zero critical and zero ordinary warnings.
- `01-VALIDATION.md`: Nyquist pass with the public-relay defense-in-depth warning.
- `scripts/test-neo4j.sh --timeout-seconds 240`: passed at final verification.
- Focused changed-package race tests and CLI build: passed.
