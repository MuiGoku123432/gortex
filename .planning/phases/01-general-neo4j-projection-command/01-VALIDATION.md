---
phase: 1
slug: general-neo4j-projection-command
status: planned
nyquist_compliant: true
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
| 01-01-T1 | 01-01 | 0 | NEO-02..15, STORE-01..05 | T-01 Wave 0 | Protocol recorder and immutable SQLite fixture | unit/contract | `GOWORK=off go test ./internal/graph/store_sqlite ./internal/neo4jprojection -run 'TestProjectionFixture|TestProjectionProtocolHarness' -count=1` | Planned | pending execution |
| 01-01-T2 | 01-01 | 0 | NEO-16, NEO-17 | T-01 Wave 0 | Mandatory pinned disposable-server harness | gate self-test | `bash -n scripts/test-neo4j.sh && GOWORK=off go test ./internal/neo4jprojection -run 'TestNeo4jIntegrationGate' -count=1` | Planned | pending execution |
| 01-02-T1 | 01-02 | 1 | NEO-02..08, NEO-11, NEO-12, NEO-14, NEO-15, NEO-17 | T-01 tracer | Real scoped SQLite through official driver and manifest switch | contract | `GOWORK=off go test ./internal/config ./internal/graph/store_sqlite ./internal/neo4jprojection -run 'TestNeo4jTracerContract|TestScopedProjectionTracer' -count=1` | Consumer of 01-01 | pending execution |
| 01-02-T2 | 01-02 | 1 | NEO-16 | T-01 tracer | Real disposable one-node/one-relationship activation | integration | `scripts/test-neo4j.sh --timeout-seconds 180 --run 'TestNeo4jProductionTracer'` | Consumer of 01-01 | pending execution |
| 01-03-T1/T2 | 01-03 | 2 | NEO-04..06, NEO-13 | T-01 mapping | Stable identities and secret-safe properties | unit/golden | `GOWORK=off go test ./internal/neo4jprojection -run 'Identity|Properties|Golden|Unresolved|Secret' -count=1` | Consumer of 01-01 | pending execution |
| 01-04-T1/T2 | 01-04 | 2 | NEO-02, NEO-08, NEO-09, NEO-11, NEO-13, NEO-15, STORE-01, STORE-02, STORE-05 | T-01 snapshot | Immutable source and bounded service batches | unit/contract | `GOWORK=off go test ./internal/graph/store_sqlite ./internal/neo4jprojection -run 'Projection|Snapshot' -count=1` | Consumer of 01-01 | pending execution |
| 01-05-T1 | 01-05 | 3 | NEO-06..08, NEO-11, NEO-12, NEO-14, NEO-15, NEO-17 | T-01 transport | Complete official-driver constraints, bounded activation, and reconciliation | protocol | `GOWORK=off go test ./internal/neo4jprojection -run 'Test.*Neo4j.*Constraint|Test.*Neo4j.*Generation|Test.*Neo4j.*Retry|Test.*Neo4j.*Stale|Test.*Neo4j.*Cancel' -count=1` | Consumer of 01-01, 01-03, 01-04 | pending execution |
| 01-05-T2 | 01-05 | 3 | NEO-06..08, NEO-11, NEO-12, NEO-14..17 | T-01 acceptance | Complete no-skip real-server suite after transport implementation | integration | `bash -n scripts/test-neo4j.sh && scripts/test-neo4j.sh --timeout-seconds 240` | Consumer of 01-05-T1 | pending execution |
| 01-06-T1/T2 | 01-06 | 4 | NEO-01..03, NEO-09..12, NEO-17, STORE-03..05 | T-01 adapters | CLI/MCP parity and no-Neo4j regression | contract/regression | `GOWORK=off go test ./internal/mcp ./cmd/gortex -run 'Neo4j|Projection' -count=1 && GOWORK=off go test -race ./...` | Consumer of 01-01 | pending execution |

## Wave 0 Requirements

- [ ] Plan 01-01 Task 1 creates the protocol recorder and SQLite DB/WAL/SHM fixture before Plans 01-02 through 01-05 consume them.
- [ ] Plan 01-01 Task 2 creates the disposable Neo4j 5.26 gate before Plans 01-02 and 01-05 execute it.
- [ ] Plan 01-03 creates identity/property golden fixtures before the full driver consumes complete mappings.
- [ ] Plan 01-06 completes CLI/MCP parity after the shared production service is stable.

## Manual-Only Verifications

All phase behaviors have automated verification. A locally stopped Docker daemon may defer disposable-server execution to CI, but does not replace that phase gate.

## Validation Sign-Off

- [x] All tasks have `<automated>` verify and consumers depend on Wave 0.
- [x] Sampling continuity: every task has an automated verify.
- [x] Wave 0 covers all prerequisite fixtures and the disposable gate.
- [x] No watch-mode flags.
- [x] Focused checks target under 30 seconds; disposable phase gates have explicit 180s/240s bounds.
- [x] `nyquist_compliant: true` set after plan/task/wave audit.

**Approval:** planned and Nyquist-audited; `wave_0_complete` remains false until Plan 01-01 executes.
