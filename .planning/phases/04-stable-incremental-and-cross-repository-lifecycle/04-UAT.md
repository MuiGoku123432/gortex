---
status: complete
phase: 04-stable-incremental-and-cross-repository-lifecycle
source: [04-VERIFICATION.md]
started: 2026-10-14T03:44:24Z
updated: 2026-10-14T18:52:38Z
---

## Current Test

[testing complete]

## Tests

### 1. Refuse-versus-skip policy for out-of-scope references (WR-04/IN-06, WR-08)
expected: One consistent, accepted policy for rename_symbol / move_symbol / inline_symbol when references lie outside the resolved scope
result: issue
reported: "go with your recommendations" (accepting: issue, not a policy question; all three refuse; keep rename's message)
severity: major
resolution: fixed in 17d43f34. Policy is refuse, for all three tools. move_symbol (cross-package only) and inline_symbol now refuse, writing nothing, when an indexed reference lies outside the resolved scope, using rename_symbol's message via the shared refsOutOfScopeResult. The existence-revealing message is accepted (it names no repository, path or content; same trade-off as 04-12). Limit: the filesystem-walk fallback still skips unindexed out-of-scope files, because deciding whether they reference the symbol would read gated bytes (D-13/D-20); the tool descriptions say so.

### 2. Lifecycle failure-path and concurrency warnings (WR-06, WR-07, REVIEW-FIX WR-01, re-review WR-01/WR-02, IN-05)
expected: Each accepted as a warning with a follow-up, or scheduled for a fix (declared-root over-prune on registration failure; log-only enclosing-reconcile failure; contention/ordering paths; unenforced "no mi.mu holder walks" rule)
result: pass
note: Accepted as warnings by the owner. WR-06 is probe-confirmed and WR-07 code-confirmed by the verifier; both trigger only after an already-logged indexing or reconcile failure, and a clean rebuild recovers. The concurrency items are timing-dependent and not reliably reachable by a deterministic test. Follow-up recorded in deferred-items.md.

## Summary

total: 2
passed: 1
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "rename_symbol, move_symbol and inline_symbol share one policy for references outside the resolved scope"
  status: resolved
  reason: "Owner-confirmed probe: an alpha-scoped move_symbol / inline_symbol reported success while skipping beta's nested/use.go caller, leaving the module unbuildable (undefined: a.Foo); rename_symbol refused"
  severity: major
  test: 1
  root_cause: "The FindUsages edge loops in handleMoveSymbol and handleInlineSymbol treated a source-gate refusal from resolveNodePath as 'skip this caller' (continue) instead of refusing the refactor as handleRenameSymbol does"
  artifacts: [internal/mcp/tools_move_inline.go, internal/mcp/source_scope.go, internal/mcp/tools_coding.go, internal/mcp/tools_move_inline_test.go]
  missing: []
  debug_session: ""
  fix: "17d43f34 -- TestMoveSymbol_OutOfScopeReferenceRefuses / TestInlineSymbol_OutOfScopeCallsiteRefuses (refused, nothing written; workspace scope succeeds). Both fail without the fix."
