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
| 01-01-T1 | 01-01 | 0 | NEO-02..15, STORE-01..05 | T-01 Wave 0 | Protocol recorder plus importable test-only SQLite fixture/fingerprint package with no production importers | unit/contract | `GOWORK=off go test ./internal/testutil/graphfixture ./internal/neo4jprojection -run 'TestProjectionFixture|TestProjectionProtocolHarness' -count=1 && imports_file=$(mktemp) && trap 'rm -f "$imports_file"' EXIT && GOWORK=off go list -f '{{if not .ForTest}}{{.ImportPath}}{{"\t"}}{{join .Imports " "}}{{end}}' ./... > "$imports_file" && awk -F '\t' '$1 != "github.com/zzet/gortex/internal/testutil" && index($1, "github.com/zzet/gortex/internal/testutil/") != 1 { count=split($2, imports, " "); for (i=1; i<=count; i++) if (imports[i] == "github.com/zzet/gortex/internal/testutil/graphfixture") { print > "/dev/stderr"; violation=1 } } END { exit violation ? 1 : 0 }' "$imports_file"` | Planned | pending execution |
| 01-01-T2 | 01-01 | 0 | NEO-16, NEO-17 | T-01 Wave 0 | Mandatory pinned disposable-server harness | gate self-test | `bash -n scripts/test-neo4j.sh && GOWORK=off go test ./internal/neo4jprojection -run 'TestNeo4jIntegrationGate' -count=1` | Planned | pending execution |
| 01-02-T1 | 01-02 | 1 | NEO-03, NEO-17, STORE-05 | T-01 contracts | Exact named-profile/env-reference and redaction contract | unit/contract | `config_tests=$(GOWORK=off go test ./internal/config -list '^TestNeo4jTracerContractProfile$') && printf '%s\n' "$config_tests" | grep -qx 'TestNeo4jTracerContractProfile' && GOWORK=off go test ./internal/config -run '^TestNeo4jTracerContractProfile$' -count=1` | Consumer of 01-01 | pending execution |
| 01-02-T2 | 01-02 | 1 | NEO-02, NEO-08, NEO-11, NEO-15, STORE-01, STORE-02, STORE-05 | T-01 contracts | Exact immutable scoped SQLite snapshot contract plus import prohibition | unit/contract | `store_tests=$(GOWORK=off go test ./internal/graph/store_sqlite -list '^TestScopedProjectionTracer$') && printf '%s\n' "$store_tests" | grep -qx 'TestScopedProjectionTracer' && GOWORK=off go test ./internal/graph/store_sqlite -run '^TestScopedProjectionTracer$' -count=1 && imports_file=$(mktemp) && trap 'rm -f "$imports_file"' EXIT && GOWORK=off go list -f '{{if not .ForTest}}{{.ImportPath}}{{"\t"}}{{join .Imports " "}}{{end}}' ./internal/graph/... > "$imports_file" && awk -F '\t' '{ count=split($2, imports, " "); for (i=1; i<=count; i++) if (imports[i] ~ /internal\/neo4jprojection|neo4j-go-driver|internal\/testutil\/graphfixture/) { print > "/dev/stderr"; violation=1 } } END { exit violation ? 1 : 0 }' "$imports_file"` | Consumer of 01-01 | pending execution |
| 01-03-T1 | 01-03 | 2 | NEO-04..08, NEO-11, NEO-12, NEO-14, NEO-15, NEO-17 | T-01 tracer | Real scoped SQLite through official driver and manifest switch | contract | `config_tests=$(GOWORK=off go test ./internal/config -list '^TestNeo4jTracerContractProfile$') && printf '%s\n' "$config_tests" | grep -qx 'TestNeo4jTracerContractProfile' && store_tests=$(GOWORK=off go test ./internal/graph/store_sqlite -list '^TestScopedProjectionTracer$') && printf '%s\n' "$store_tests" | grep -qx 'TestScopedProjectionTracer' && service_tests=$(GOWORK=off go test ./internal/neo4jprojection -list '^TestNeo4jTracerContract$') && printf '%s\n' "$service_tests" | grep -qx 'TestNeo4jTracerContract' && GOWORK=off go test ./internal/config ./internal/graph/store_sqlite ./internal/neo4jprojection -run '^(TestNeo4jTracerContractProfile|TestScopedProjectionTracer|TestNeo4jTracerContract)$' -count=1` | Consumers of 01-01 and 01-02 | pending execution |
| 01-03-T2 | 01-03 | 2 | NEO-16 | T-01 tracer | Real disposable one-node/one-relationship activation | integration | `scripts/test-neo4j.sh --timeout-seconds 180 --run 'TestNeo4jProductionTracer'` | Consumer of 01-01 and 01-03-T1 | pending execution |
| 01-04-T1/T2 | 01-04 | 3 | NEO-04..06, NEO-13 | T-01 mapping | Stable identities and secret-safe properties | unit/golden | `GOWORK=off go test ./internal/neo4jprojection -run 'Identity|Properties|Golden|Unresolved|Secret' -count=1` | Consumer of 01-03 | pending execution |
| 01-05-T1/T2 | 01-05 | 3 | NEO-02, NEO-08, NEO-09, NEO-11, NEO-13, NEO-15, STORE-01, STORE-02, STORE-05 | T-01 snapshot | Immutable source expansion and bounded service batches | unit/contract | `GOWORK=off go test ./internal/graph/store_sqlite ./internal/neo4jprojection -run 'Projection|Snapshot' -count=1` | Consumer of 01-01 and 01-03 | pending execution |
| 01-06-T1 | 01-06 | 4 | NEO-06..08, NEO-11, NEO-12, NEO-14, NEO-15, NEO-17 | T-01 transport | Complete official-driver constraints, bounded activation, and reconciliation | protocol | `GOWORK=off go test ./internal/neo4jprojection -run 'Test.*Neo4j.*Constraint|Test.*Neo4j.*Generation|Test.*Neo4j.*Retry|Test.*Neo4j.*Stale|Test.*Neo4j.*Cancel' -count=1` | Consumer of 01-01, 01-04, 01-05 | pending execution |
| 01-06-T2 | 01-06 | 4 | NEO-06..08, NEO-11, NEO-12, NEO-14..17 | T-01 acceptance | Complete no-skip real-server suite after transport implementation | integration | `bash -n scripts/test-neo4j.sh && scripts/test-neo4j.sh --timeout-seconds 240` | Consumer of 01-06-T1 | pending execution |
| 01-07-T1 | 01-07 | 5 | NEO-01, NEO-03, NEO-09..11, NEO-17, STORE-03..05 | T-01 adapters | CLI flags, output, progress, and incomplete-result behavior | contract | `GOWORK=off go test ./cmd/gortex -run 'Test.*Neo4jPush.*Flags|Test.*Neo4jPush.*JSON|Test.*Neo4jPush.*Progress|Test.*Neo4jPush.*Incomplete' -count=1` | Planned | pending execution |
| 01-07-T2 | 01-07 | 5 | NEO-01..03, NEO-09..12, NEO-17, STORE-03..05 | T-01 adapters | CLI/MCP parity and no-Neo4j startup/build regression | contract/regression | `GOWORK=off go test ./internal/mcp ./cmd/gortex -run 'Test.*Neo4j.*Parity|Test.*Neo4j.*Scope|Test.*Neo4j.*Mutation|Test.*Neo4j.*NoServer' -count=1 && GOWORK=off go build -o /tmp/gortex-phase1 ./cmd/gortex/` | Consumer of 01-01 | pending execution |

## Wave 0 Requirements

- [ ] Plan 01-01 Task 1 creates the package-local protocol recorder and importable `internal/testutil/graphfixture` DB/WAL/SHM fixture APIs before Plans 01-02, 01-03, 01-05, and 01-06 consume them from `_test.go` files.
- [ ] Wave 0's `go list` production-import scan proves no ordinary package imports `internal/testutil/graphfixture`; package-specific tests open its returned database path through their own production package.
- [ ] Plan 01-01 Task 2 creates the disposable Neo4j 5.26 gate before Plans 01-03 and 01-06 execute it.
- [ ] Plan 01-02 creates exact profile/snapshot contracts before Plan 01-03 imports the official driver or service.
- [ ] Plan 01-04 creates identity/property golden fixtures before the full driver consumes complete mappings.
- [ ] Plan 01-07 completes CLI/MCP parity after the shared production service is stable.

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
