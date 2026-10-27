---
phase: 01-general-neo4j-projection-command
verified: 2026-09-22T13:20:14Z
status: passed
score: 26/26 must-haves verified
behavior_unverified: 0
overrides_applied: 0
residual_warnings:
  - "Defense in depth: no single test composes CLI -> daemon relay -> registered MCP handler -> named profile -> production service/official driver -> disposable Neo4j. Each seam and the real target path are independently tested; actual wiring is present."
  - "The broad repository-wide race suite has a pre-existing timeout/user-cache isolation problem outside Phase 01. The five changed packages pass the focused race suite."
---

# Phase 1: General Neo4j Projection Command Verification Report

**Phase Goal:** Users can explicitly materialize a selected, scoped snapshot of Gortex's authoritative SQLite graph into Neo4j through equivalent CLI and MCP operations without making Neo4j authoritative or required by ordinary Gortex workflows.

**Verified:** 2026-09-22T13:20:14Z
**Status:** passed
**Re-verification:** No -- initial goal-backward verification
**Code baseline:** `dcee2266`; production and test code are unchanged from reviewed baseline `c944732e` (`git diff c944732e..HEAD` contains planning artifacts only).

## Goal Achievement

### Roadmap Success Criteria

| # | Truth | Status | Actual code and behavioral evidence |
|---|---|---|---|
| 1 | Equivalent CLI and MCP operations require explicit workspace/project/repository scope and do not expose credentials. | VERIFIED | `cmd/gortex/neo4j.go:26-98` normalizes required fields and calls `neo4j_push`; `internal/mcp/tools_neo4j.go:16-60` declares the same fields and checks canonical scope equality; `internal/config/global.go:111-186` stores env references, excludes resolved secrets from YAML, rejects URI userinfo, and resolves credentials only on explicit invocation. `TestNeo4jPushFlags`, `TestNeo4jParity`, `TestNeo4jScope`, and profile tests passed. |
| 2 | The selected snapshot materializes language-agnostic nodes/relationships with stable keys, constraints, scope, provenance, evidence, origin, confidence, lifecycle, and source location. | VERIFIED | `identity.go:17-61` derives length-delimited owner/logical/generation/physical keys; `properties.go:38-75` maps all native node/edge fields and unresolved labels; `neo4j.go:136-145,227-354` creates node/manifest/per-type relationship constraints and parameterized MERGE batches. Identity/property/golden/unresolved/secret tests and disposable target assertions passed. |
| 3 | Retry/resume is bounded and idempotent; stale records reconcile only inside the exact owner/generation after partial failure. | VERIFIED | `service.go:382-529` creates unique attempt generations, renews leases, stages bounded batches, independently completes/counts, atomically activates, then reconciles; `neo4j.go:149-225,356-525` fences acquire/stage/completion/activation/abort/cleanup by owner, operation, generation, attempt, state, and lease. Generation, retry, stale, cancellation, mismatch, fence-loss, and disposable rerun tests passed. |
| 4 | Dry-run is non-mutating and reports target actions/counts; progress, cancellation, and actionable incomplete results work without leaking secrets. | VERIFIED | `service.go:317-380` reads the immutable source, calls target `Plan`, and reports physical stale counts, logical removals, missing constraints, and planned actions without acquire/stage/activate; `service.go:190-206,411-529` reports phases/counts and safe failure states; adapters preserve the shared Result. Dry-run omissions/plan, progress, incomplete, cancellation, redaction, and disposable zero-write assertions passed. |
| 5 | Disposable/protocol acceptance proves success, retry, stale cleanup, cancellation/error, scope isolation, SQLite immutability, and ordinary no-server behavior. | VERIFIED | Independent verifier run of `scripts/test-neo4j.sh --timeout-seconds 240` passed against pinned `neo4j:5.26-community`. Focused changed-package race tests, all exact PLAN selectors, `TestNeo4jNoServer`, SQLite fingerprint tests, and CLI build passed. |

### Plan Must-Have Truths

| Plan | Must-have truths | Status | Evidence |
|---|---|---|---|
| 01-01 | Executable fixtures precede implementation; pinned bounded exact-test disposable gate; no ordinary Neo4j dependency. | VERIFIED | Fixture/protocol/script files are substantive. Exact script guards and harness tests passed. Script validates strict test identifiers, exact listing, pinned image, bounded readiness/execution, random credentials, and cleanup. |
| 01-02 | Explicit named env-backed profile; immutable cancellable exact-scope SQLite snapshot; exact guards. | VERIFIED | `ResolveNeo4jProfile` and `OpenScopedProjectionSnapshot` are production implementations, not fakes. Profile and snapshot exact tests passed. |
| 01-03 | Real scoped SQLite node/edge reaches disposable Neo4j through official driver; prior generation remains visible before atomic activation; mandatory gate fails closed. | VERIFIED | Service -> snapshot -> official transport data flow is real. `TestNeo4jProductionTracer` ran in the full disposable suite. |
| 01-04 | Stable logical/physical identities preserve occurrence evidence; all safe fields survive with collision-safe names; secret/unsupported values are counted omissions. | VERIFIED | `identity.go` and `properties.go` implement these contracts. Seven exact identity/property tests passed. |
| 01-05 | Exact-scope snapshot keyset pagination scales; all result paths preserve SQLite; service uses bounded batches/reads/progress/cancellation. | VERIFIED | Read-only transaction and 256-row pages are in `scoped_projection.go`; service cap is 500. Pagination, fingerprint, cancellation, batch, cadence, and bounded-read tests passed. |
| 01-06 | Invocation-local official driver enforces 5.26+, constraints, bounded parameterized MERGE; exact-owner replay/activation/cleanup is fenced and resumable; full disposable matrix passes. | VERIFIED | Production Cypher and lifecycle predicates were inspected. All exact transport tests and full disposable suite passed. |
| 01-07 | CLI/MCP share defaults, fail-closed scope, mutation/progress/result semantics; output is safe; dry-run and failures are actionable. | VERIFIED | CLI calls registered `neo4j_push`; MCP normalizes through the same request and service, and is classified `EffectExternalWrite`. Eight exact adapter tests and CLI build passed. |

**Score:** 26/26 truths verified (5 roadmap criteria plus 21 plan truths; 0 behavior-unverified).

## Required Artifacts

| Artifact group | Levels 1-3 | Wiring/data flow | Status |
|---|---|---|---|
| `internal/testutil/graphfixture/*`, `protocol_test.go`, `scripts/test-neo4j.sh` | Present and substantive | Imported only from tests; script runs the real integration package | VERIFIED |
| `internal/config/global.go` | Present and substantive | MCP resolves a named profile only inside explicit push | VERIFIED |
| `internal/graph/scoped_projection.go`, `internal/graph/store_sqlite/scoped_projection.go` | Present and substantive | Service opens one real read-only SQLite transaction and streams pages | VERIFIED |
| `internal/neo4jprojection/service.go` | Present and substantive | Shared orchestration consumed by MCP and all transport tests | VERIFIED |
| `internal/neo4jprojection/neo4j.go` | Present and substantive | Official v6 driver receives mapped records and executes target queries | VERIFIED |
| `identity.go`, `properties.go`, golden fixture | Present and substantive | Stage calls `projectNode`/`projectEdge`; values flow into parameterized MERGE rows | VERIFIED |
| `cmd/gortex/neo4j.go` | Present and substantive | Registered Cobra command calls daemon tool `neo4j_push` | VERIFIED |
| `internal/mcp/tools_neo4j.go` | Present and substantive | Registered by `Server.registerNeo4jTools` and routes to `Service.Push` | VERIFIED |
| Integration and focused test files | Present and substantive | Enumerated exact selectors execute against production seams | VERIFIED |

## Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| CLI | daemon/MCP | `callNeo4jPushTool(..., "neo4j_push", args)` | WIRED | Same normalized field names and shared Result JSON. |
| MCP registration | handler | `registerNeo4jTools` -> `handleNeo4jPush` | WIRED | Registered from `internal/mcp/server.go:1909`. |
| MCP handler | profile/service | `ResolveNeo4jProfile` -> `NewService` -> `Push` | WIRED | Driver construction is invocation-local. |
| Service | SQLite | `OpenScopedProjectionSnapshot` and page callbacks | WIRED | Exact workspace/project/repository predicates and one read-only transaction. |
| Service | mapper/transport | `projectNode`/`projectEdge`, then `Stage` | WIRED | Real authoritative rows become target records. |
| Transport | Neo4j | official driver `ExecuteQuery`, constraints, parameterized MERGE | WIRED | Verified by disposable Neo4j suite. |
| Lifecycle | manifest and records | acquire/renew/complete/activate/reconcile/abort | WIRED | Exact owner/operation/generation/attempt/lease predicates are present and behaviorally tested. |

## Data-Flow Trace (Level 4)

| Data | Source | Transformation | Sink/result | Status |
|---|---|---|---|---|
| Scoped nodes/edges | Authoritative SQLite `nodes`/`edges` under read-only snapshot | Stable keys, closed labels/types, typed properties, secret filtering | Parameterized Neo4j `MERGE` rows and final counts | FLOWING |
| Dry-run target plan | Real source census plus live target manifest/record census and `SHOW CONSTRAINTS` | Physical cleanup, logical removals, missing constraints, planned actions | Shared CLI/MCP Result, no acquire/stage/activation | FLOWING |
| Progress/failure | Service phase transitions, counts, context errors, cleanup census | Safe codes/messages and rerun action | MCP progress/shared Result/CLI output | FLOWING |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Every exact PLAN selector exists and passes | Exact `go test -list`, `grep -qx`, and aggregate run across six packages | All 34 named tests passed | PASS |
| Focused race for changed packages and adversarial lifecycle cases | `GOWORK=off go test -race ./internal/config ./internal/graph/store_sqlite ./internal/neo4jprojection ./internal/mcp ./cmd/gortex -run '^(...)$' -count=1` | All five packages passed | PASS |
| CLI builds without Neo4j server/profile | `GOWORK=off go build -o /var/.../gortex-phase1-verify ./cmd/gortex/` | Exit 0 | PASS |
| Harness syntax | `bash -n scripts/test-neo4j.sh` | Exit 0 | PASS |
| Full exact disposable Neo4j suite | `scripts/test-neo4j.sh --timeout-seconds 240` | Passed in verifier process (`internal/neo4jprojection`, 6.556s) | PASS |

## Probe Execution

No `probe-*.sh` is declared for this phase. The declared mandatory executable gate is `scripts/test-neo4j.sh`; it was run directly and passed as recorded above.

## Requirements Coverage

| Requirement | Status | Evidence |
|---|---|---|
| NEO-01 | SATISFIED | Explicit equivalent CLI/MCP adapters; parity tests pass. |
| NEO-02 | SATISFIED | Exact non-empty workspace/project/repository scope at normalization, canonical MCP resolution, and SQLite predicates. |
| NEO-03 | SATISFIED | Named profiles, env-only secrets, userinfo rejection, safe errors/results/properties. |
| NEO-04 | SATISFIED | Stable node logical/physical keys derive from authoritative identity. |
| NEO-05 | SATISFIED | Edge identity binds owner, endpoints, kind, provenance, and occurrence. |
| NEO-06 | SATISFIED | Node, manifest, and sanitized per-relationship physical-key uniqueness constraints. |
| NEO-07 | SATISFIED | Parameterized `MERGE` upserts and identical-rerun acceptance. |
| NEO-08 | SATISFIED | 256-row source pages, <=500-record target batches, 30s transaction/retry caps. |
| NEO-09 | SATISFIED | Dry-run inspects real server/source/target and plans actions without mutation. |
| NEO-10 | SATISFIED | Redacted progress and shared scoped Result counts. |
| NEO-11 | SATISFIED | Context-aware reads/driver calls, bounded close/abort, actionable non-success. |
| NEO-12 | SATISFIED | Unique attempts, exact lease fencing, independent observed counts, replay/recovery tests. |
| NEO-13 | SATISFIED | Typed graph fields, metadata envelope, scope/provenance/origin/confidence/lifecycle/location retained. |
| NEO-14 | SATISFIED | Exact-owner generation census and bounded relationship-first stale reconciliation. |
| NEO-15 | SATISFIED | Read-only SQLite snapshot and physical/canonical fingerprint acceptance; no reverse mutation. |
| NEO-16 | SATISFIED | Full pinned disposable suite plus protocol seam covers required scenarios. |
| NEO-17 | SATISFIED | `TestNeo4jNoServer`, focused ordinary package tests, and CLI build pass without target/profile. |
| STORE-01 | SATISFIED | SQLite remains the only authoritative store; Neo4j records are rebuildable projections. |
| STORE-02 | SATISFIED | No `neo4jprojection`/driver import or call exists in `internal/indexer`; indexing has no dual-write hook. |
| STORE-03 | SATISFIED | Projection is one-way and manually invoked only by CLI/MCP push. |
| STORE-04 | SATISFIED | No watcher, scheduler, CDC, outbox, or background synchronization path was found. |
| STORE-05 | SATISFIED | No ordinary startup requirement and no Neo4j-to-SQLite reverse-write path exists. |

No Phase 1 requirement is orphaned: all NEO-01..17 and STORE-01..05 appear in plan frontmatter and have implementation/test evidence.

## Security and Review Gates

| Gate | Current result | Verification assessment |
|---|---|---|
| Code review | `01-REVIEW.md`: clean, 0 critical/warning/info | Production/test diff after reviewed baseline is empty; focused and disposable checks independently reconfirm behavior. |
| Nyquist | `01-VALIDATION.md`: pass with warning, compliant | Exact selectors and current behavior were independently rerun. Residual relay composition warning retained below. |
| Security | `01-SECURITY.md`: secured, 47/47 closed, `threats_open: 0` | Current code still enforces scope, lease fencing, redaction, TLS policy, metadata bounds, and optional target behavior. |

## Anti-Patterns Found

| Scope | Pattern | Severity | Result |
|---|---|---|---|
| Phase implementation files | `TBD`, `FIXME`, `XXX`, `TODO`, `HACK`, `PLACEHOLDER`, not-implemented markers | None | No matches. |
| Production dependency graph | Index-time Neo4j call/import | None | No match under `internal/indexer`. |
| Authority boundary | Reverse write, watcher, scheduler, dual-write hook | None | No implementation evidence found; data flow is SQLite -> explicit projection only. |

## Residual Warnings

1. **Combined public relay composition:** No single disposable test composes the whole CLI -> daemon relay -> registered MCP handler -> named profile -> production service/driver -> Neo4j chain. This is not a gap: the actual CLI call, tool registration, handler/service construction, source flow, and target flow are wired in production code, and each seam plus the disposable target path passed independently.
2. **Broad repository suite:** The previously documented broad `go test -race ./...` timeout/user-cache isolation issue is outside the Phase 01 changed-package evidence. It was not used to weaken this verdict. The focused race suite for all five changed packages passed.

## Human Verification Required

None. All state-transition, cancellation, cleanup, ordering, and external-target truths have named automated behavioral evidence. No visual or subjective behavior is in scope.

## Gaps Summary

No blocking or uncertain must-have remains. All five roadmap criteria, all 21 plan truths, and all 22 requirements are verified against current implementation and independently executed tests. Phase goal achieved.

---

_Verified: 2026-09-22T13:20:14Z_
_Verifier: the agent (gsd-verifier)_
