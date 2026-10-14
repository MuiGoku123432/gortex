---
status: testing
phase: 04-stable-incremental-and-cross-repository-lifecycle
source: [04-VERIFICATION.md]
started: 2026-10-14T03:44:24Z
updated: 2026-10-14T03:44:24Z
---

## Current Test

number: 1
name: Refuse-versus-skip policy for out-of-scope references (rename_symbol, move_symbol, inline_symbol; WR-04/IN-06, WR-08)
expected: |
  One consistent policy across the three refactor tools. Today rename_symbol refuses with a distinct
  "some references lie outside the resolved scope" message, while move_symbol and inline_symbol skip
  out-of-scope callers silently and report success (leaving e.g. beta calling a moved a.Foo()).
  Decide: refuse, or skip with an explicit advisory — and accept or change the existence-revealing message.
awaiting: user response

## Tests

### 1. Refuse-versus-skip policy for out-of-scope references (WR-04/IN-06, WR-08)
expected: One consistent, accepted policy for rename_symbol / move_symbol / inline_symbol when references lie outside the resolved scope
result: [pending]

### 2. Lifecycle failure-path and concurrency warnings (WR-06, WR-07, REVIEW-FIX WR-01, re-review WR-01/WR-02, IN-05)
expected: Each accepted as a warning with a follow-up, or scheduled for a fix (declared-root over-prune on registration failure; log-only enclosing-reconcile failure; contention/ordering paths; unenforced "no mi.mu holder walks" rule)
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
