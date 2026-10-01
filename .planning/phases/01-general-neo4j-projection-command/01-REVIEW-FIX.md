---
phase: 01-general-neo4j-projection-command
fixed_at: 2026-09-22T05:17:00Z
review_path: .planning/phases/01-general-neo4j-projection-command/01-REVIEW.md
iteration: 6
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

**Fixed at:** 2026-09-22T05:17:00Z
**Source review:** `.planning/phases/01-general-neo4j-projection-command/01-REVIEW.md`
**Iteration:** 6

**Summary:**
- Findings in scope: 1
- Fixed: 1
- Skipped: 0

## Fixed Issues

### CR-01: Reject expired cleanup fences and atomically release clean owners

**Files modified:** `internal/neo4jprojection/neo4j.go`, `internal/neo4jprojection/service.go`, `internal/neo4jprojection/neo4j_test.go`, `internal/neo4jprojection/tracer_integration_test.go`
**Commits:** `d7999ebe`, `64e91498`
**Applied fix:** Cleanup transactions now return an explicit fence sentinel, and final release requires the same unexpired exact owner/operation/active generation/attempt while atomically proving zero non-active exact-owner records. Lease loss reports cleanup-incomplete with truthful counts/action and leaves takeover-safe state. Focused and disposable tests cover expiry before the first delete and before final release.
**Status:** fixed: requires human verification

### WR-01: Report physical former-generation cleanup in dry-run

**Files modified:** `internal/neo4jprojection/neo4j.go`, `internal/neo4jprojection/service.go`, `internal/neo4jprojection/tracer_integration_test.go`
**Commits:** `d7999ebe`, `64e91498`
**Applied fix:** Target planning reports physical exact-owner records that a fresh generation will make non-active separately from logical removals. Dry-run emits deterministic cleanup work for existing active/abandoned generations while remaining non-mutating. Disposable coverage includes identical-target counts/actions and zero writes.
**Status:** fixed: requires human verification

## Additional Nyquist Tests

- Production cleanup expiry interleavings at both required points.
- Missing pending node and relationship census mismatch preserving prior active state.
- CLI and MCP omission-count JSON parity.
- Blocking close five-second bound and joined error preservation.
- Stale identical-target dry-run action/count and no-mutation coverage.

## Verification

- Verification ran in the isolated review-fix worktree.
- Focused and changed-package race suites passed for config, scoped SQLite, projection, MCP, and CLI packages.
- Exact new selectors passed; the CLI build, shell syntax, and diff checks passed.
- Disposable Neo4j execution remains environment-blocked because the Docker VM reports `No space left on device`. The user declined pruning unused Docker data, so no disposable pass is claimed.
- Final Gortex change/diff graph calls timed out while the daemon was busy; focused behavioral gates and git diff hygiene passed.

WR-02 full public relay remains optional defense-in-depth and was not implemented. Phase completion state was not changed.

## Iteration 6: Disposable Cleanup Cypher Regression

### CR-02: Restore the Cypher clause boundary before cleanup subqueries

**Files modified:** `internal/neo4jprojection/neo4j.go`, `internal/neo4jprojection/neo4j_test.go`
**Applied fix:** Both lease-fenced cleanup mutation queries now carry `m` through an explicit `WITH m` between `SET m.lease_until` and `CALL (m)`. Neo4j 5.26 had rejected the new query shape with `Neo.ClientError.Statement.SyntaxError: WITH is required between SET and CALL`, which the service correctly surfaced as cleanup-incomplete. The fix preserves the exact owner/operation/generation/attempt fence, lease renewal, bounded relationship-first cleanup, final stale census, and atomic zero-stale release.
**Regression:** `TestNeo4jCleanupQueriesFenceReleaseAndAssertNoStaleRecords` now requires the valid lease-renewal/subquery boundary. It failed before the fix and passes after it.
**Status:** fixed; phase remains blocked pending stable final disposable rerun and the pre-existing audit gaps.

### Iteration 6 Verification

- Focused regression and changed-package race selectors passed.
- CLI build, `bash -n scripts/test-neo4j.sh`, and `git diff --check` passed.
- `scripts/test-neo4j.sh --timeout-seconds 240 --run TestNeo4jProductionTracer` passed after the fix in `3.106s`.
- `scripts/test-neo4j.sh --timeout-seconds 240` passed after the fix in `3.241s`.
- Later confirmation attempts failed before test execution at `Neo4j readiness timed out`; Docker reports substantial reclaimable local-volume usage, consistent with the existing environment-capacity blocker. No pass is claimed for those attempts.
- Phase completion state was not changed.

---

_Fixed: 2026-09-22T05:17:00Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 6_
