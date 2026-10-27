---
phase: 1
slug: general-neo4j-projection-command
status: passed_with_warning
nyquist_compliant: true
wave_0_complete: true
validated: 2026-09-22
verdict: PASS WITH WARNING
validated_commit: c944732e
---

# Phase 1 - Final Clean Nyquist Audit

## GAPS FILLED

**Verdict:** PASS WITH WARNING

Phase 01 is cleanly certifiable at current HEAD `c944732e`. Every exact PLAN selector was rediscovered before execution and passed. The previously open adversarial gaps now have executable behavioral coverage for cleanup lease expiry, pending-count mismatch, omission-result parity, stale identical-target dry-run planning, and bounded close. Focused race, CLI build, focused disposable production tracer, and the complete pinned disposable Neo4j suite all passed.

The combined public CLI-to-daemon-to-MCP-to-disposable-Neo4j relay remains a non-blocking defense-in-depth warning. Its independent seams are green, and no observed failure justifies blocking Phase 01.

No production code or tests were modified. This validation artifact is the only intentional audit edit.

## Audit Scope

Read and assessed current Phase 01 PLANs, SUMMARYs, requirements, coverage/review/fix/validation artifacts, and the relevant current tests in:

- `internal/neo4jprojection`
- `internal/mcp`
- `cmd/gortex`
- `internal/config`
- `internal/graph/store_sqlite`
- `internal/testutil/graphfixture`
- `scripts/test-neo4j.sh`

No project-local `.claude/skills/` or `.agents/skills/` directory exists.

## Gap Resolution

| Gap | Behavioral evidence at current HEAD | Result |
|---|---|---|
| Cleanup lease expiry | Disposable `TestNeo4jProductionTracer` expires the production cleanup lease before the first relationship delete and before final release. Both cases require `ErrCleanupIncomplete`, preserve activated completeness, reject false cleanup success, and leave takeover-safe manifest state. | FILLED |
| Intended/observed count mismatch | `TestProjectionMissingPendingRecordBlocksActivation/node` and `/relationship` independently remove one observed pending record, require non-success, preserve the prior active generation, and prove activation did not run. | FILLED |
| Omission parity | `TestProjectionDryRunReportsMapperOmissions` proves service aggregation. `TestNeo4jParity` requires `secret_omissions` and `unsupported_omissions` to survive shared Result JSON through MCP, while `TestNeo4jPushJSON` verifies the CLI shared JSON path. | FILLED |
| Stale dry-run planning | Disposable `TestNeo4jProductionTracer` seeds an active owner, dry-runs an identical target, requires physical former-generation node/edge counts and `delete_stale_records`, requires zero logical removals, and proves the target census is unchanged. | FILLED |
| Bounded close | `TestProjectionCloseIsBoundedAndPreservesOperationError` supplies a close implementation blocked until context expiry, observes the five-second bound, and requires the operation error, close error, and deadline error to remain joined. | FILLED |
| Cleanup fence/result distinction | `TestNeo4jCleanupDeleteDistinguishesFenceLossFromZeroRows` rejects an empty result as fence loss while accepting a fenced zero-delete result. The disposable lease-expiry cases prove the production behavior. | FILLED |
| Changed-package race | Focused `-race` execution across config, scoped SQLite, projection, MCP, and CLI packages passed. | FILLED |
| CLI build and script hygiene | `GOWORK=off go build`, `bash -n scripts/test-neo4j.sh`, and `git diff --check` passed. | FILLED |
| Focused disposable tracer | Exact `TestNeo4jProductionTracer` disposable command passed in `4.209s`. | FILLED |
| Full disposable suite | `scripts/test-neo4j.sh --timeout-seconds 240` passed in `4.227s`; this independently confirms the supplied prior full-suite pass evidence. | FILLED |
| Combined public relay | No single test composes the entire CLI-to-daemon-to-MCP-to-disposable target path. Adapter normalization, daemon cancellation, MCP registration/handler, service, driver, output, and disposable target behavior are independently green. | WARNING -- non-blocking defense in depth |

## Exact PLAN Selector Audit

Every named selector was first listed with its anchored expression, checked as an exact line, and then executed with the anchored aggregate selector.

| Plan task | Exact selectors | Result |
|---|---|---|
| 01-01-T1 | `TestProjectionFixture`, `TestProjectionProtocolHarness` | PASS |
| 01-01-T2 | `TestNeo4jIntegrationGate`, `TestNeo4jScriptExactRunExecutesKnownTest`, `TestNeo4jScriptExactRunRejectsMissingTest` | PASS |
| 01-02-T1 | `TestNeo4jTracerContractProfile` | PASS |
| 01-02-T2 | `TestScopedProjectionTracer` | PASS |
| 01-03-T1 | `TestNeo4jTracerContractProfile`, `TestScopedProjectionTracer`, `TestNeo4jTracerContract` | PASS |
| 01-03-T2 | Exact disposable `TestNeo4jProductionTracer` | PASS |
| 01-04-T1 | `TestProjectionIdentity`, `TestProjectionOwnerIdentity`, `TestProjectionTokenSanitization` | PASS |
| 01-04-T2 | `TestProjectionProperties`, `TestProjectionPropertiesGolden`, `TestProjectionUnresolved`, `TestProjectionSecretFiltering` | PASS |
| 01-05-T1 | `TestScopedProjectionEstatePagination`, `TestScopedProjectionFingerprintCoverage`, `TestScopedProjectionCancellation` | PASS |
| 01-05-T2 | `TestProjectionBatchAggregation`, `TestProjectionProgressCadence`, `TestProjectionBoundedReads` | PASS |
| 01-06-T1 | `TestNeo4jConstraints`, `TestNeo4jGenerationActivation`, `TestNeo4jRetryBounds`, `TestNeo4jStaleReconciliation`, `TestNeo4jCancellation` | PASS |
| 01-06-T2 | Complete pinned disposable Neo4j suite | PASS |
| 01-07-T1 | `TestNeo4jPushFlags`, `TestNeo4jPushJSON`, `TestNeo4jPushProgress`, `TestNeo4jPushIncomplete` | PASS |
| 01-07-T2 | `TestNeo4jParity`, `TestNeo4jScope`, `TestNeo4jMutation`, `TestNeo4jNoServer`; CLI build | PASS |

## Verification Map Updates

| Requirement behavior | Test type | Automated command | Status |
|---|---|---|---|
| Exact PLAN selectors | unit/integration | Anchored `go test -list`, exact-line checks, and anchored package runs for all 01-01 through 01-07 selectors | green |
| Lease-expiry and stale dry-run production behavior | disposable integration | `scripts/test-neo4j.sh --timeout-seconds 240 --run TestNeo4jProductionTracer` | green, `4.209s` |
| Count mismatch, omission parity, bounded close | unit/integration | `GOWORK=off go test ./internal/neo4jprojection ./internal/mcp ./cmd/gortex -run '^(TestProjectionMissingPendingRecordBlocksActivation|TestProjectionDryRunReportsMapperOmissions|TestProjectionCloseIsBoundedAndPreservesOperationError|TestNeo4jCleanupDeleteDistinguishesFenceLossFromZeroRows|TestNeo4jCleanupQueriesFenceReleaseAndAssertNoStaleRecords|TestNeo4jParity|TestNeo4jPushJSON)$' -count=1 -v` | green |
| Focused changed-package race | integration/race | `GOWORK=off go test -race ./internal/config ./internal/graph/store_sqlite ./internal/neo4jprojection ./internal/mcp ./cmd/gortex -run '^(TestNeo4jTracerContractProfile|TestScopedProjectionTracer|TestScopedProjectionEstatePagination|TestScopedProjectionFingerprintCoverage|TestScopedProjectionCancellation|TestProjectionMissingPendingRecordBlocksActivation|TestProjectionDryRunReportsMapperOmissions|TestProjectionCloseIsBoundedAndPreservesOperationError|TestNeo4jCleanupDeleteDistinguishesFenceLossFromZeroRows|TestNeo4jParity|TestNeo4jPushJSON|TestNeo4jNoServer)$' -count=1` | green |
| CLI build and harness hygiene | smoke | `GOWORK=off go build -o /tmp/gortex-phase1-c944732e ./cmd/gortex/ && bash -n scripts/test-neo4j.sh && git diff --check` | green |
| Complete mandatory real-server acceptance | disposable integration | `scripts/test-neo4j.sh --timeout-seconds 240` | green, `4.227s` |

## Observed Focused Results

- Exact selector aggregate: all six involved package lanes passed.
- Adversarial selector run: all named tests and mismatch subtests passed; bounded close completed in `5.00s`.
- Focused race: config `1.193s`, scoped SQLite `16.518s`, projection `7.042s`, MCP `4.860s`, CLI `4.419s`.
- CLI build, shell syntax, and diff hygiene: passed with no output.
- Focused disposable tracer: passed in `4.209s`.
- Full disposable suite: passed in `4.227s`.

## Requirements Coverage

- `NEO-01..17`: certified by exact selectors plus the focused adversarial and disposable evidence above.
- `STORE-01..05`: certified. SQLite remains authoritative and unchanged; projection is explicit, one-way, rebuildable, and absent from ordinary startup requirements.
- `D-01..D-17`: certified for Phase 01. The only remaining caveat is the optional combined public-relay composition test.

## Final Sign-Off

- [x] All exact PLAN selectors exist and pass.
- [x] Cleanup lease expiry is behaviorally forced at both critical interleavings.
- [x] Missing pending node and relationship counts block activation.
- [x] Omission counts survive service and public Result JSON paths.
- [x] Stale identical-target dry-run reports physical cleanup and performs zero writes.
- [x] Blocking close is bounded and preserves joined errors.
- [x] Focused changed-package race passes.
- [x] CLI build and script hygiene pass.
- [x] Focused disposable tracer passes.
- [x] Full disposable suite passes.
- [x] Combined public relay remains a non-blocking warning.

**Final Nyquist verdict:** PASS WITH WARNING. Phase 01 is Nyquist-compliant and cleanly certifiable at `c944732e`; the only remaining item is the accepted non-blocking defense-in-depth public relay composition warning.

## Files for Commit

- `.planning/phases/01-general-neo4j-projection-command/01-VALIDATION.md`
