---
phase: 1
slug: general-neo4j-projection-command
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-21
---

# Phase 1 - Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing`, existing `testify`/`rapid` where useful |
| **Config file** | `go.mod` |
| **Quick run command** | `GOWORK=off go test ./internal/neo4jprojection ./internal/mcp ./cmd/gortex -run 'Neo4j|Projection'` |
| **Full suite command** | `GOWORK=off go test -race ./...` |
| **Estimated runtime** | Quick under 30 seconds; full suite runtime measured during execution |

## Sampling Rate

- **After every task commit:** Run the focused package or adapter test named by the task.
- **After every plan wave:** Run `GOWORK=off go test -race` for changed packages and the protocol suite.
- **Before `/gsd-verify-work`:** Run `GOWORK=off go test -race ./...` and the disposable Neo4j acceptance lane.
- **Max feedback latency:** 30 seconds for task-level checks.

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-W0-01 | TBD | 0 | NEO-01..17, STORE-01..05 | T-01-SC | Deterministic protocol fake and immutable SQLite fixture | unit/contract | `GOWORK=off go test ./internal/neo4jprojection -run 'Protocol|Projection'` | No - W0 | pending |
| 01-CLI-MCP | TBD | TBD | NEO-01, NEO-02, NEO-03, NEO-09, NEO-10, NEO-11, NEO-17 | T-01-01 | Fail-closed scope/profile parsing and redacted equivalent result | contract | `GOWORK=off go test ./internal/mcp ./cmd/gortex -run 'Neo4j|Projection'` | No - W0 | pending |
| 01-MAP | TBD | TBD | NEO-04, NEO-05, NEO-06, NEO-13 | T-01-02 | Stable collision-aware identities and secret-safe properties | unit/golden | `GOWORK=off go test ./internal/neo4jprojection -run 'Identity|Properties|Constraint'` | No - W0 | pending |
| 01-GEN | TBD | TBD | NEO-07, NEO-08, NEO-12, NEO-14 | T-01-03 | Idempotent bounded generation activation and owner-scoped cleanup | protocol/integration | `GOWORK=off go test ./internal/neo4jprojection -run 'Generation|Retry|Stale|Cancel'` | No - W0 | pending |
| 01-ACC | TBD | TBD | NEO-15, NEO-16, NEO-17, STORE-01..05 | T-01-04 | SQLite unchanged and Neo4j optional outside explicit push | integration/regression | `GOWORK=off go test -race ./...` plus explicit disposable-Neo4j test command defined by the plan | No - W0 | pending |

## Wave 0 Requirements

- [ ] Protocol fake with transaction recording, bounded-batch assertions, injected failures, cancellation, and commit uncertainty.
- [ ] SQLite scoped fixture and database/WAL/SHM fingerprint helper.
- [ ] Identity/property golden fixtures covering collisions, unresolved targets, nested metadata, unsupported values, and secret canaries.
- [ ] CLI/MCP parity harness comparing normalized requests and final structured results.
- [ ] Disposable Neo4j 5.26 integration script or CI service behind an explicit integration gate.

## Manual-Only Verifications

All phase behaviors have automated verification. A locally stopped Docker daemon may defer disposable-server execution to CI, but does not replace that phase gate.

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies.
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify.
- [ ] Wave 0 covers all MISSING references.
- [ ] No watch-mode flags.
- [ ] Feedback latency under 30 seconds for focused checks.
- [ ] `nyquist_compliant: true` set after validation audit.

**Approval:** pending
