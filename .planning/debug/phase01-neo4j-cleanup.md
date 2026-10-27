---
status: awaiting_human_verify
trigger: "Investigate and fix the Phase 01 mandatory disposable regression at current HEAD. scripts/test-neo4j.sh --timeout-seconds 240 fails in TestNeo4jProductionTracer at tracer_integration_test.go:108 with neo4j projection cleanup incomplete after commits d7999ebe, 64e91498, 05a9b8c9."
created: 2026-09-22T04:36:50Z
updated: 2026-09-22T05:25:00Z
---

## Current Focus

hypothesis: Confirmed: both hardened cleanup mutation queries omit the mandatory WITH boundary between SET and CALL, so Neo4j rejects cleanup before any stale census or release logic executes.
test: Run focused/race tests, CLI build, exact disposable tracer, full 240-second disposable suite, shell syntax, diff checks, and fix-acceptance guardrails.
expecting: All requested gates pass with the exact fence predicates, lease renewals, bounded LIMIT deletes, final zero-stale release, and security redaction unchanged.
next_action: Await human confirmation; phase remains incomplete because the disposable environment became unstable on repeated confirmation runs and prior VALIDATION blockers remain.
reasoning_checkpoint:
  hypothesis: Reconcile causes the reported cleanup-incomplete because its relationship and node cleanup Cypher place CALL directly after SET, which Neo4j 5.26 rejects with a syntax error.
  confirming_evidence:
    - The exact disposable tracer reproduces at the first Push cleanup and joins ErrCleanupIncomplete with the classified Neo4j operation error.
    - A temporary diagnostic exposed Neo4jError Neo.ClientError.Statement.SyntaxError stating WITH is required between SET and CALL at Reconcile's CALL (m).
  falsification_test: If adding only WITH m after each cleanup lease SET does not make the focused regression and exact disposable tracer pass, or if Neo4j reports a different remaining failure, this hypothesis is incomplete or wrong.
  fix_rationale: WITH m is the required Cypher clause boundary and preserves m as the imported variable for the existing scoped subquery; it changes neither fence predicates, lease renewal, delete bounds, stale census, nor atomic release.
  blind_spots: The full disposable suite may reveal independent failures in other tests, and a successful focused run does not alone prove all race/security controls.
  candidate_causes:
    - code: malformed Reconcile Cypher introduced by the cleanup-hardening commit omits WITH between SET and CALL.
    - environment: Neo4j image/version compatibility could differ, but the pinned required 5.26 image directly and deterministically rejects the query.
    - data: stale fixture records could produce nonzero census, but the observed path is a parser error before deletion or census evaluation.
  and_gate: no -- the malformed query alone fails the first cleanup on an empty prior generation; lease loss and fixture contamination are independently ruled out by the parser error.
known_pattern_candidate: Prior Phase 01 lifecycle fixes establish that exact attempt ownership and an unexpired lease must remain fenced through cleanup and atomic zero-stale release.
bug_class: Bohrbug unless repeated disposable runs establish timing sensitivity.

## Symptoms

expected: scripts/test-neo4j.sh --timeout-seconds 240 completes successfully at current HEAD while preserving exact attempt fencing, intended counts, bounded cleanup, stale dry-run, and security controls.
actual: The mandatory disposable suite fails in TestNeo4jProductionTracer at tracer_integration_test.go:108.
errors: neo4j projection cleanup incomplete
reproduction: Run scripts/test-neo4j.sh --timeout-seconds 240 against its disposable Neo4j fixture.
started: Current HEAD after commits d7999ebe, 64e91498, and 05a9b8c9.

## Eliminated

## Evidence

- timestamp: 2026-09-22T04:36:50Z
  checked: Gortex session memories and localization
  found: Reconcile in internal/neo4jprojection/neo4j.go is the highest-ranked production site; prior invariants require exact attempt and unexpired lease through cleanup and distinguish logical activation from physical cleanup truth.
  implication: The known lifecycle invariant is a hypothesis candidate, not proof; direct disposable state is required.
- timestamp: 2026-09-22T04:36:50Z
  checked: Working tree status
  found: Existing user changes are present in .planning/config.json, 01-VALIDATION.md, state.json, milestone.lock, 01-REVIEW-FIX.md, and 01-REVIEW.md.
  implication: Investigation and commit must not overwrite or accidentally stage unrelated existing work; REVIEW-FIX may only be updated deliberately after inspection.
- timestamp: 2026-09-22T04:44:00Z
  checked: Complete Reconcile, Service.Push, disposable tracer, test script, REVIEW, VALIDATION, and REVIEW-FIX
  found: Reconcile retains the exact cleanup fence, deletes relationships before disconnected stale nodes, then atomically releases only on a zero stale census. The production tracer performs two same-owner pushes before line 108 and currently reports only the generic error because t.Fatal(err) discards the Result.
  implication: The code preserves truthful failure semantics; the next experiment must identify the specific Push and remaining census rather than weakening ErrCleanupIncomplete.
- timestamp: 2026-09-22T04:44:00Z
  checked: Phase 01 validation history
  found: The focused disposable tracer previously passed before commits d7999ebe, 64e91498, and 05a9b8c9; the new regression is therefore likely in the newly hardened cleanup queries or their test fixture assumptions.
  implication: Differential debugging against those commits is appropriate after reproducing current HEAD.
- timestamp: 2026-09-22T04:47:00Z
  checked: Exact disposable TestNeo4jProductionTracer at current HEAD
  found: Reproduced deterministically in 1.34s at tracer_integration_test.go:108 with joined errors ErrCleanupIncomplete and "neo4j projection operation failed".
  implication: Reconcile encountered a Neo4j query error and then truthfully converted it to cleanup-incomplete; this is not the plain nonzero-census branch and does not currently point to lease expiry.
- timestamp: 2026-09-22T04:51:00Z
  checked: Temporary diagnostic preserving the original execution path but exposing the wrapped Neo4j driver error
  found: Neo4j 5.26 reports "WITH is required between SET and CALL" at the first Reconcile cleanup query's CALL (m) subquery.
  implication: The root cause is a deterministic Cypher syntax defect introduced by d7999ebe; lease, final census, physical planning, and fixture contamination are not needed to produce this failure.
- timestamp: 2026-09-22T04:51:00Z
  checked: Diagnostic source cleanup
  found: classifyNeo4jError was restored byte-for-byte after collecting the underlying error.
  implication: No temporary error-detail/security behavior remains in the working tree.
- timestamp: 2026-09-22T04:57:00Z
  checked: New TestNeo4jCleanupQueriesFenceReleaseAndAssertNoStaleRecords assertion
  found: The focused test fails before the production fix because neither cleanup mutation query contains the required SET/ WITH m / CALL clause boundary.
  implication: The regression test is red for the exact malformed query shape and will guard both cleanup loops without requiring Docker.
- timestamp: 2026-09-22T05:00:00Z
  checked: Minimal production fix and focused verification
  found: Adding WITH m after both lease-renewing SET clauses makes the focused regression pass and the exact disposable TestNeo4jProductionTracer pass in 3.106s.
  implication: The fix addresses the observed Neo4j 5.26 parser failure while retaining the cleanup protocol structure.
- timestamp: 2026-09-22T05:08:00Z
  checked: Fix-acceptance revert-and-reconfirm and mutation-equivalent check
  found: Removing both WITH m clauses makes the agent-authored focused regression fail; restoring only those clauses makes it pass. The diff adds behavior and strengthens an assertion rather than deleting or bypassing cleanup.
  implication: The regression directly kills the fix-site mutant and the minimal change is causally responsible for correcting the defect.
- timestamp: 2026-09-22T05:17:00Z
  checked: Requested focused/race tests, CLI build, shell/diff hygiene
  found: Focused race selectors passed across internal/config, internal/graph/store_sqlite, internal/neo4jprojection, internal/mcp, and cmd/gortex; CLI build, bash -n, and git diff --check passed.
  implication: Adjacent code and repository hygiene gates are green.
- timestamp: 2026-09-22T05:17:00Z
  checked: Post-fix disposable verification
  found: The exact disposable tracer passed once immediately after the fix (3.106s), and the full default 240-second suite passed once (3.241s). Subsequent isolated reruns failed at Neo4j readiness before Go test execution; Docker reports 44.46GB of local volumes with 28.42GB reclaimable, matching the prior no-space/readiness environment blocker.
  implication: The code regression is fixed and both mandatory commands have successful post-fix evidence, but disposable startup is not currently stable; phase completion must remain blocked.
- timestamp: 2026-09-22T05:25:00Z
  checked: Atomic commit and final package race test
  found: Commit c944732e contains only the two code/test changes plus the requested REVIEW-FIX iteration 6 update; go test -race ./internal/neo4jprojection passes in 8.471s. Pre-existing planning changes remain unstaged.
  implication: The fix is atomically committed without absorbing unrelated user work.

## Resolution

root_cause: d7999ebe introduced two Reconcile Cypher queries that execute CALL (m) immediately after SET without the mandatory WITH m clause boundary; Neo4j 5.26 rejects the first cleanup transaction, and truthful cleanup handling reports ErrCleanupIncomplete.
fix: Inserted WITH m between lease renewal and each scoped cleanup subquery; strengthened the cleanup query regression to require that exact valid clause boundary.
verification:
  target_test: {result: pass}
  mutation_check: {result: pass, reason_if_skipped: "Stryker is not applicable to Go; direct fix-site deletion mutant was applied and killed by the agent-authored regression", mutant_killed: true}
  no_op_deletion: {result: pass, deletion_justified_by_rca: false}
  adjacent_tests: {result: pass, suites_run: ["focused changed-package race selectors", "CLI build", "exact disposable tracer", "full disposable suite", "shell syntax", "diff check"]}
  revert_and_reconfirm: {result: pass, bug_returned_on_revert: true, fixed_on_reapply: true}
  guardrail_verdict: accepted
oracle_type: specified
files_changed:
  - internal/neo4jprojection/neo4j.go
  - internal/neo4jprojection/neo4j_test.go
