---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 14
subsystem: resolver
tags: [cobol, resolver, reconcile, graph-scope, allowlist, copybook-library, db2, idms, gc]

requires:
  - phase: 04-06
    provides: reconcileCobol / ReconcileCobol, decideCobol, CobolPolicy types, SetCobolPolicyLookup
  - phase: 04-03
    provides: graph.NewScope / Scope.Allows fail-closed repository boundary
  - phase: 04-02
    provides: cobol_placeholder and cobol_copy_library_key edge Meta
  - phase: 04-01
    provides: shared resources with FilePath == ID and prov_range_absence shared_resource
provides:
  - "Directed cross-repository COBOL binding: caller scope = itself + policy ResolveFrom (D-09), one candidate union (D-19)"
  - "Reverse reconcile-set expansion: a change in B re-decides every repository resolving from B (Pitfall 5)"
  - "Ordered copybook library search with OF/IN narrowing; edge Meta cobol_copy_search_rank / cobol_copy_search_rule cobol-copy-search/1 (D-10)"
  - "reconcileCobolResources: orphan shared-resource GC by EvictFile on the virtual path (CR-02, D-05)"
  - "D-11 EdgeAliases link from _unqualified DB2/IDMS resource to existing qualified node, Meta cobol_qualifier_rule cobol-default-qualifier/1"
affects: [04-07, 04-08, 04-15]

actuals:
  tokens: 10189   # chars/4 over the realized diff (40757 chars)
  tasks: 2
  commits: 4
plan_head_before: f5163424bdce3e01ba42322513ecd3c8b897b347

tech-stack:
  added: []
  patterns:
    - "Pass-scoped policy cache: cobolPolicyFor(policies, repo) consults the lookup once per repository per pass"
    - "Rule-keyed edge ownership: the pass only removes EdgeAliases edges that carry its own cobol_qualifier_rule key"

key-files:
  created:
    - internal/resolver/cobol_resources.go
    - internal/resolver/cobol_resources_test.go
  modified:
    - internal/resolver/cobol_references.go
    - internal/resolver/cobol_policy.go
    - internal/resolver/cobol_reconcile_test.go
    - internal/graph/edge.go

key-decisions:
  - "Program union (D-19): a contained candidate in the caller repository wins outright; otherwise the caller-repository tier (same-file, else other-file) is joined with every top-level program of the other allowed repositories, so same-file never shadows an allowed repository"
  - "Reconcile-set expansion reads the policy of every repository holding a COBOL program or copybook only when changed is non-empty; an empty changed still reconciles every edge, as in 04-06"
  - "cobolPolicyFor now takes the pass cache map, so decideCobol and reconcileCobolResources share one lookup per repository per pass; the 04-06 nolint is gone"
  - "cobol_copy_search_rank comparison tolerates int / int64 / float64 so a ranked bind is a true no-op on the second SQLite pass (asserted zero writes)"
  - "Orphan collection runs before linking: a qualified node reached only by a D-11 alias is itself an orphan, so it is evicted and its link goes with it"
  - "Default schema / subschema values are used as configured (TrimSpace only); 04-07 must pass them in the producer's normalized ID form (cobol-name-norm/1) or the qualified node will never be found"

patterns-established:
  - "COBOL shared-resource lifecycle is owned by the reconcile pass: it ends with reconcileCobolResources over the same repository set"

requirements-completed: [SEC-03, ID-04, ID-07]

coverage:
  - id: D1
    description: "Cross-repository COBOL CALL/COPY binding only through a directed ResolveFrom allowlist, with union ambiguity, foreign nested programs excluded, and unknown repositories behaving like no policy"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_Allowlist"
        status: pass
    human_judgment: false
  - id: D2
    description: "A change in repository B re-decides every repository whose ResolveFrom names B"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_Allowlist/reverse_fanout"
        status: pass
    human_judgment: false
  - id: D3
    description: "Ordered copybook libraries: rank wins, OF/IN narrowing, unranked dropped, ties ambiguous, cross-repository library, rank keys removed without libraries; idempotent on SQLite"
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_SearchOrder"
        status: pass
    human_judgment: false
  - id: D4
    description: "Orphaned shared resources (none or alias-only in-edges) are evicted; real-file nodes and resources outside the reconcile set survive"
    requirement: ID-07
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_resources_test.go#TestCobolReconcile_ResourceGC"
        status: pass
    human_judgment: false
  - id: D5
    description: "Configured default schema/subschema keeps exactly one rule-tagged alias edge to an existing qualified node, removes it when the default goes, never creates or renames a node, never touches foreign alias edges"
    requirement: ID-04
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_resources_test.go#TestCobolReconcile_Alias"
        status: pass
    human_judgment: false
  - id: D6
    description: "No regression in the resolver package, graph edge tests, and the Phase 2/3 COBOL real-path suites"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/resolver/"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 14: COBOL Allowlist, Library Order, Qualifier Links, and Resource GC Summary

**The COBOL reconcile pass now binds across repositories only through a directed `ResolveFrom` allowlist, with one candidate union scoped by `graph.NewScope`. It ranks copybooks by an ordered library list and records `cobol_copy_search_rank`. It also evicts orphaned shared resources by their virtual path and keeps `cobol-default-qualifier/1` alias edges from `_unqualified` DB2/IDMS resources to existing qualified nodes.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-10-13T03:28:56Z
- **Completed:** 2026-10-13T03:37:39Z
- **Tasks:** 2 (TDD: RED + GREEN each)
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments
- Per-caller allowed scope is the caller repository plus its trimmed `ResolveFrom`, built with `graph.NewScope` and applied with `Scope.Allows`. With no policy the pass is same-repository only, as before.
- Candidates from every allowed repository form one union, so a same-repository program or copybook never silently shadows an allowed-repository one (two candidates give `ambiguous` with both IDs). Programs in other repositories count only when top-level (`cobol_nested=false`).
- A non-empty `changed` set grows by every repository whose `ResolveFrom` names a changed repository (`cobolReconcileRepos`).
- Copybook libraries: a candidate ranks by the first library (in configured order, narrowed to the `cobol_copy_library_key` Name when present) whose repository and exact directory hold it. Unranked candidates drop out, and a unique minimum binds with `cobol_copy_search_rank` and `cobol_copy_search_rule`. Every other decision deletes both keys.
- New `reconcileCobolResources` runs at the end of `reconcileCobol` over the same set. It evicts a shared resource (`FilePath == ID`) whose in-edges are none or only `EdgeAliases`. It then keeps exactly one rule-tagged `EdgeAliases` link per `_unqualified` resource whose policy default names an existing qualified node, and removes stale rule-tagged links.

## Task Commits

1. **Task 1 RED: allowlist and search-order tests** - `c142ee8c` (test)
2. **Task 1 GREEN: allowlist union and ordered libraries** - `6f58efe0` (feat)
3. **Task 2 RED: resource GC and alias tests** - `cb2fb1ea` (test)
4. **Task 2 GREEN: resource GC and qualifier links** - `da372896` (feat)

REFACTOR: none needed.

## TDD Gate Compliance

- **Task 1 RED** (`c142ee8c`): `GOWORK=off go test -count=1 -v ./internal/resolver/ -run 'TestCobolReconcile_(Allowlist|SearchOrder)'` exited 1. Nine subtests failed on assertions:
  - Allowlist `allowed`: To stayed `unresolved::cobol::program::XPGM`.
  - `union_ambiguous`: bound to `repoA/sub.cbl::SUBPGM` instead of ambiguous.
  - `reverse_fanout`: repoA's edge was not re-decided.
  - SearchOrder `rank_wins` and `of_qualifier`: ambiguous, with no rank.
  - `unranked_dropped`: bound to `repoA/other/custrec.cpy`.
  - `cross_repo_library`: missing.
  - `no_libraries`: rank keys kept.

  Five guard subtests passed in RED because they hold under the same-repository rule: `no_policy`, `directed`, `foreign_nested_excluded`, `unknown_repo`, `tie_ambiguous`.
- **Task 1 GREEN** (`6f58efe0`): all 13 subtests and the 04-06 COBOL tests pass under `-race`.
- **Task 2 RED** (`cb2fb1ea`): same command for `(ResourceGC|Alias)` exited 1. Five subtests failed on assertions:
  - `orphan` and `alias_only_evicted`: node not evicted.
  - `default_schema`, `policy_removed`, `idms_shape`: expected 1 alias edge, got 0.

  Guards (`referenced_kept`, `legacy_guard`, `outside_set`, `no_target`, `go_alias_untouched`) passed in RED.
- **Task 2 GREEN** (`da372896`): all 10 subtests pass. The full resolver package passes under `-race`.

## Files Created/Modified
- `internal/resolver/cobol_references.go`:
  - Reconcile-set expansion and the `cobolRef` type, now at package level.
  - `decideCobol`: allowed union, program tiers with allowed-repository joins, library ranking.
  - `cobolLibraryRank`, `cobolRankEqual`, the rank Meta writes, and the `reconcileCobolResources` call.
- `internal/resolver/cobol_policy.go`: `cobolCopySearchRule`, `cobolQualifierRule`, the cached `cobolPolicyFor(policies, repo)`, and `cobolReconcileRepos`.
- `internal/resolver/cobol_resources.go` (new): `reconcileCobolResources` and `cobolQualifiedID`.
- `internal/resolver/cobol_reconcile_test.go`: `TestCobolReconcile_Allowlist` and `TestCobolReconcile_SearchOrder`, plus the `cobolPolicies` helper.
- `internal/resolver/cobol_resources_test.go` (new): `TestCobolReconcile_ResourceGC` and `TestCobolReconcile_Alias`.
- `internal/graph/edge.go`: one doc line on `EdgeAliases` for the D-11 COBOL use.

## Decisions Made
See `key-decisions` in the frontmatter. One matters for 04-07: default qualifiers are compared as configured, with only whitespace trimmed. Config wiring must normalize them the way the producer forms resource IDs.

## Deviations from Plan

None. The plan executed as written. Additions within plan scope:
- The `rank_wins` subtest runs on SQLite and asserts zero second-pass writes, which covers the "stays idempotent" truth for rank Meta.
- `cobolPolicyFor` takes the pass cache as a parameter, which satisfies "policies read once per repository per pass" across both functions.

## Issues Encountered
- `golangci-lint` is not installed locally, so the removal of `//nolint:unused` was checked only by `go vet` and build. `cobolPolicyFor` now has two callers.
- `gofmt -l internal/resolver/` still flags the pre-existing `csharp_verbatim_type_ref_test.go`, which this plan did not touch.
- Other COBOL tests showed no regressions: `internal/indexer`, `internal/neo4jprojection`, `internal/parser/languages`, and the `cmd/gortex` Tracer/Breadth suites were each run alone.

## User Setup Required
None. No external service configuration required.

## Next Phase Readiness
- 04-07 can wire `index.cobol` config into `SetCobolPolicyLookup`: `ResolveFrom`, `CopybookLibraries`, and the default schema/subschema in normalized form.
- 04-08 and 04-15 get orphan collection for free from every `ReconcileCobol(changed)` tail. Deleting the last reference to a shared table removes the table node.

## Self-Check: PASSED
- FOUND: internal/resolver/cobol_resources.go, internal/resolver/cobol_resources_test.go, internal/resolver/cobol_references.go, internal/resolver/cobol_policy.go, internal/resolver/cobol_reconcile_test.go, internal/graph/edge.go
- FOUND commits: c142ee8c, 6f58efe0, cb2fb1ea, da372896
- Both task verify commands exited 0. Acceptance greps matched: `cobol-copy-search/1` (1 line), `graph.NewScope(` (1 line), `func (r *Resolver) reconcileCobolResources(` (1 line), `cobol-default-qualifier/1` (1 line). `git diff --name-only` since the plan base lists only the 6 planned files.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*
