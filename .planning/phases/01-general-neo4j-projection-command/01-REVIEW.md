---
phase: 01-general-neo4j-projection-command
reviewed: 2026-09-22T13:08:55Z
depth: deep
files_reviewed: 32
files_reviewed_list:
  - SECURITY.md
  - cmd/gortex/cli_daemon.go
  - cmd/gortex/neo4j.go
  - cmd/gortex/neo4j_test.go
  - go.mod
  - internal/config/global.go
  - internal/config/global_test.go
  - internal/daemon/mutating.go
  - internal/graph/scoped_projection.go
  - internal/graph/store_sqlite/projection_fingerprint_test.go
  - internal/graph/store_sqlite/scoped_projection.go
  - internal/graph/store_sqlite/scoped_projection_test.go
  - internal/mcp/facade_registry.go
  - internal/mcp/facade_tools_test.go
  - internal/mcp/server.go
  - internal/mcp/tools_neo4j.go
  - internal/mcp/tools_neo4j_test.go
  - internal/neo4jprojection/identity.go
  - internal/neo4jprojection/neo4j.go
  - internal/neo4jprojection/neo4j_integration_test.go
  - internal/neo4jprojection/neo4j_test.go
  - internal/neo4jprojection/properties.go
  - internal/neo4jprojection/properties_test.go
  - internal/neo4jprojection/protocol_test.go
  - internal/neo4jprojection/security_docs_test.go
  - internal/neo4jprojection/service.go
  - internal/neo4jprojection/service_test.go
  - internal/neo4jprojection/testdata/properties.golden.json
  - internal/neo4jprojection/tracer_integration_test.go
  - internal/testutil/graphfixture/fixture.go
  - internal/testutil/graphfixture/fixture_test.go
  - scripts/test-neo4j.sh
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Phase 01: Code Review Report

**Reviewed:** 2026-09-22T13:08:55Z
**Depth:** deep
**Files Reviewed:** 32
**Status:** clean

## Summary

Final adversarial review of Phase 01 at current HEAD `c944732e`, including fixes in `d7999ebe`, `64e91498`, `05a9b8c9`, and `c944732e`. I re-read the complete Phase 01 planning, summary, context, research, pattern, coverage, validation, prior review/fix artifacts, current implementation, adapters, and tests. I traced the source snapshot, mapping, target plan, acquisition, staging, independent completion census, activation, cleanup, abort, close, and public result paths.

No blocker or ordinary warning remains. All historical findings are resolved in the current code:

- **Cleanup lease loss and release:** every cleanup mutation requires the exact owner/operation/active-generation/attempt, an unexpired cleanup lease, and returns an explicit fence sentinel. Empty results are rejected as lease loss. Final release uses the same fence and atomically requires zero non-active exact-owner nodes and relationships before marking cleanup complete and returning the manifest to idle. Lease loss is reported as cleanup-incomplete with a fresh census and a takeover-safe manifest.
- **Dry-run physical cleanup:** target planning separately reports all current exact-owner physical records that a fresh attempt will make stale and logical records absent from the intended snapshot. Identical-target dry-run therefore reports physical cleanup and `delete_stale_records` without mutating Neo4j.
- **Stage fencing:** each node and relationship mutation transaction atomically validates owner, operation, pending generation, attempt, active state, and lease, then renews the lease before `UNWIND`/`MERGE`. Superseded or expired attempts cannot continue staging.
- **Intended count mismatch:** the service independently deduplicates intended node and edge identities. `MarkComplete` independently counts target records, records intended and observed counts, and permits activation only when both match. Missing-node and missing-relationship regressions preserve the prior active generation.
- **Omission parity:** mapper omission counts are aggregated for dry-run and apply, retained in the shared `Result`, and asserted through MCP JSON and CLI JSON paths.
- **Bounded close:** transport close uses a fresh five-second context. The blocking-close regression proves bounded return and preserves both the operation and close errors.
- **Cypher syntax boundary:** both cleanup queries include the required `WITH m` between lease-renewing `SET` and `CALL (m)`. The Neo4j 5.26 production tracer and full disposable suite execute this boundary successfully.
- **Historical lifecycle and security findings:** exact scope, endpoint hydration, collision-safe identities, secret filtering, metadata-envelope bounds, unique attempt generations, prior-generation preservation, cleanup truth, bounded abort/stop handling, duration caps, TLS guidance, adapter normalization, and no-server startup remain intact.

All reviewed files meet the Phase 01 quality and correctness bar. No issues found.

## Verification

The following final gates passed at current HEAD:

```text
GOWORK=off go test -race ./internal/config ./internal/graph/store_sqlite \
  ./internal/neo4jprojection ./internal/mcp ./cmd/gortex \
  -run '^(TestNeo4jCleanupDeleteDistinguishesFenceLossFromZeroRows|TestNeo4jCleanupQueriesFenceReleaseAndAssertNoStaleRecords|TestNeo4jStageQueriesFenceEveryMutationTransaction|TestProjectionMissingPendingRecordBlocksActivation|TestProjectionDryRunReportsMapperOmissions|TestProjectionCloseIsBoundedAndPreservesOperationError|TestNeo4jParity|TestNeo4jPushJSON|TestNeo4jProductionTracer)$' \
  -count=1

scripts/test-neo4j.sh --timeout-seconds 240

GOWORK=off go build -o /tmp/gortex-phase1-final-review ./cmd/gortex/
bash -n scripts/test-neo4j.sh
git diff --check d7999ebe^..HEAD
```

The focused race suite passed across all five changed packages. The complete disposable Neo4j suite passed in `5.471s`. CLI build, shell syntax, and diff hygiene passed.

## Narrative Findings (AI reviewer)

No Critical, Warning, or Info findings.

## Defense-in-Depth Residual (Non-finding)

### Combined public relay composition

There is still no single acceptance test that composes `CLI -> daemon relay -> registered MCP handler -> named profile -> production service/official driver -> disposable Neo4j`. The individual CLI, daemon/MCP, profile, normalization/result, cancellation, service, official-driver, and disposable Neo4j seams are covered and passed. No incorrect behavior is demonstrated, so this remains an explicitly separated defense-in-depth residual rather than an ordinary warning and does not prevent a clean verdict.

---

_Reviewed: 2026-09-22T13:08:55Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: deep_
