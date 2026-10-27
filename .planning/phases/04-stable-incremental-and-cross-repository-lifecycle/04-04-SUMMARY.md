---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 04
subsystem: resolver
tags: [go, resolver, cross-repo, determinism, sqlite, D-12, SEC-03]

requires:
  - phase: 03
    provides: COBOL cobol_unresolved_reason / cobol_candidate_ids Meta-only stamp and EdgeMetaBatchPersister pattern (cobol_references.go)
provides:
  - "pickCrossRepoCandidate: deterministic fail-closed choice for every generic cross-repository fallback tier"
  - "Edge Meta resolution_unresolved_reason=ambiguous + resolution_candidate_ids (sorted) on refused generic cross-repo calls"
  - "Meta-only ambiguity stamps persisted via EdgeMetaBatchPersister in ResolveAllContext and resolveScopedLocked"
affects: [04-06, 04-07, 04-10, 04-14, cross-repo resolution, neo4j projection of unresolved edges]

actuals:
  tokens: 4950
  tasks: 2
  commits: 4
plan_head_before: e268312867f8cedc8563d463ca6adccfd7b3956f

tech-stack:
  added: []
  patterns:
    - "Cross-repo fallback tiers pass their gates as an accept closure to pickCrossRepoCandidate; ambiguity returns sorted IDs and stops the tier walk"
    - "Bind in CrossRepoResolver.resolveEdge clears resolution_* keys centrally; ReindexEdges persists the cleared Meta"

key-files:
  created: []
  modified:
    - internal/resolver/cross_repo.go
    - internal/resolver/cross_repo_test.go

key-decisions:
  - "Clearing of resolution_unresolved_reason / resolution_candidate_ids on bind lives once in CrossRepoResolver.resolveEdge (e.To != oldTo) instead of at each of the four call sites; it covers every bind of the pass, and ReindexEdges persists the full Meta"
  - "stampCrossRepoAmbiguous is a package-private function shared by the four fallback tiers (rule of three met); the two persistence blocks (ResolveAllContext, resolveScopedLocked) stay duplicated inline with note comments"
  - "SEC-03 not marked complete: plans 04-06, 04-07, 04-10, and 04-14 also declare it and have no SUMMARY yet (shared-ID gate)"

patterns-established:
  - "D-12 generic ambiguity: eligible candidates spanning more than one non-empty repository bind nothing; one repository binds lowest ID; synthetic no-repo nodes keep precedence"

requirements-completed: [SEC-03]

coverage:
  - id: D1
    description: "resolveFunctionCall cross-repo fallback refuses multi-repository matches, stamps ambiguous with sorted candidate IDs, binds the lowest ID within one repository, and is identical across 20 runs"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/resolver/cross_repo_test.go#TestCrossRepoResolveAll_Ambiguous/two_repos"
        status: pass
      - kind: unit
        ref: "internal/resolver/cross_repo_test.go#TestCrossRepoResolveAll_Ambiguous/one_repo_two_candidates"
        status: pass
      - kind: unit
        ref: "internal/resolver/cross_repo_test.go#TestCrossRepoResolveAll_Ambiguous/deterministic"
        status: pass
    human_judgment: false
  - id: D2
    description: "Binding clears stale resolution_* keys, and Meta-only ambiguity stamps survive a SQLite close/reopen"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/resolver/cross_repo_test.go#TestCrossRepoResolveAll_Ambiguous/clears_stale_keys"
        status: pass
      - kind: integration
        ref: "internal/resolver/cross_repo_test.go#TestCrossRepoResolveAll_Ambiguous/persisted"
        status: pass
    human_judgment: false
  - id: D3
    description: "All three resolveMethodCall cross-repo tiers (exact type, method name, function) refuse ambiguity without falling through; same-repo binds unchanged"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/resolver/cross_repo_test.go#TestCrossRepoResolveAll_AmbiguousMethod"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/resolver/ && GOWORK=off go test -race -count=1 ./internal/indexer/ -run 'CrossRepo|Workspace'"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 04: Deterministic Fail-Closed Cross-Repository Fallback Summary

**pickCrossRepoCandidate replaces first-match binding in the generic cross-repository fallback (one function tier and three method tiers). When matches span two or more repositories, the edge stays unresolved and gets a persisted `resolution_unresolved_reason=ambiguous` stamp plus sorted candidate IDs. Matches within one repository bind to the lowest node ID.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-10-13T01:46:58Z
- **Completed:** 2026-10-13T01:53:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `CrossRepoResolver.pickCrossRepoCandidate(candidates, accept)` handles candidate choice the same way in every generic fallback tier:
  - An eligible synthetic no-repo node still wins (lowest ID).
  - If all eligible candidates are in one repository, the lowest ID wins.
  - If they span two or more repositories, nothing binds and the sorted IDs are returned.
- `resolveFunctionCall` step 2 and the exact-type, method-name, and function cross-repo tiers of `resolveMethodCall` all use the helper. An ambiguous tier stops the method walk, so a weaker tier cannot bind what it refused.
- Meta-only ambiguity stamps (where `To` is unchanged) are collected per worker in `CrossRepoStats.ambiguous`. They are persisted with `PersistEdgeAttributesBatch` after each super-chunk merge and after the scoped (repo/file) pass.
- A bind during the pass deletes any earlier `resolution_*` keys.
- The `cobol::` owned-namespace exit in `resolveEdge` is unchanged.

## Task Commits

1. **Task 1 RED: failing tests for function fallback** - `7d9d68b2` (test)
2. **Task 1 GREEN: pickCrossRepoCandidate + resolveFunctionCall + persistence** - `c9d73734` (feat)
3. **Task 2 RED: failing tests for method tiers** - `abfa7f8a` (test)
4. **Task 2 GREEN: three resolveMethodCall tiers** - `47b75526` (feat)

No REFACTOR commits were needed.

## TDD Evidence

- **Task 1 RED:** `GOWORK=off go test -count=1 ./internal/resolver/ -run 'TestCrossRepoResolveAll_Ambiguous$'` exited 1. All five subtests failed on behavior assertions, not on compilation:
  - two_repos and persisted: `To` was bound to repoB instead of staying unresolved, and the stamp was nil.
  - one_repo_two_candidates: got `z.go` instead of the lowest ID, `a.go`.
  - deterministic: the stamp was nil on run 0.
  - clears_stale_keys: the stale keys were still present after the bind.
- **Task 1 GREEN:** all five subtests pass, and so does `-race -run 'CrossRepo|Cobol'`.
- **Task 2 RED:** `-run 'TestCrossRepoResolveAll_AmbiguousMethod$'` exited 1.
  - exact_type, method_name, and function_tier failed because `To` was bound to repoB and the stamp was nil.
  - same_repo_unchanged passed as intended; it guards behavior that must not change.
- **Task 2 GREEN:** all four subtests pass. The full `./internal/resolver` package passes under `-race`, as do the `./internal/indexer -run 'CrossRepo|Workspace'` tests.
- `gsd-tools check tdd-red-evidence` cannot parse `go test` output, so the evidence above comes from the actual test runs.

## Files Created/Modified
- `internal/resolver/cross_repo.go`:
  - Adds the `ambiguous` field on `CrossRepoStats`, `pickCrossRepoCandidate`, and `stampCrossRepoAmbiguous`.
  - Changes four call sites.
  - Clears the keys on bind in `resolveEdge`.
  - Adds two persistence blocks.
- `internal/resolver/cross_repo_test.go`:
  - Adds the `seedTwoRepoHelper` and `callEdgeAt` fixtures.
  - Adds `TestCrossRepoResolveAll_Ambiguous` (5 subtests) and `TestCrossRepoResolveAll_AmbiguousMethod` (4 subtests).

## Decisions Made
- The bind-time key clearing happens once, in `resolveEdge` (when `e.To != oldTo`), instead of separately at each of the four pick sites. It covers every bind in the cross-repo pass, same-repository binds included. `ReindexEdges` already persists the full edge payload, so the cleared Meta reaches disk with no extra write.
- `stampCrossRepoAmbiguous` is shared by four call sites, so it meets the rule of three. The persistence block is used in only two places, so it stays inline with `note:` comments.
- `CrossRepoStats.ambiguous` is not merged into the top-level stats of `ResolveAllContext`. Edges are persisted per super-chunk, so they are not retained for the whole pass.

## Deviations from Plan

None in behavior. The one structural choice is that key deletion on bind is centralized in `resolveEdge` instead of written at each call site (see Decisions). It has the same effect and slightly broader coverage.

**Total deviations:** 0 auto-fixed
**Impact on plan:** None.

## Issues Encountered
None. Two out-of-scope observations are logged in `deferred-items.md`:
- A generic ambiguity stamp goes stale when a later pass finds zero candidates.
- `tryRemoteStitch` still runs after a local ambiguity refusal. This behavior already existed, and its evidence gate refuses bare names.

## Next Phase Readiness
- SEC-03 is still open at the requirement level until 04-06, 04-07, 04-10, and 04-14 complete. 04-04's share (generic cross-repo determinism) is done.
- Downstream readers must treat `resolution_unresolved_reason` as meaningful only while `To` is still unresolved (Pitfall 8).

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*

## Self-Check: PASSED
