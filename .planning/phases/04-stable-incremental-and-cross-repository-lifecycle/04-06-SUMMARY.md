---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 06
subsystem: resolver
tags: [cobol, resolver, reconcile, graph-scope, determinism, incremental]

requires:
  - phase: 04-02
    provides: cobol_placeholder edge Meta on every literal CALL and non-IDMS COPY edge
  - phase: 04-03
    provides: graph.NewScope / Scope.Allows fail-closed repository boundary
provides:
  - "Resolver.reconcileCobol(changed) / exported Resolver.ReconcileCobol(changed): one COBOL binding pass for full and incremental paths"
  - "Bound-edge re-decision (WR-01): bound COBOL edges return to their placeholder when a second candidate appears or the target is gone"
  - "caller_unowned reason: a caller with an empty RepoPrefix never binds (SC5 fail closed)"
  - "CobolPolicy / CobolLibrary / CobolPolicyLookup types and Resolver.SetCobolPolicyLookup (nil = same-repository only)"
affects: [04-07, 04-08, 04-14, 04-15]

actuals:
  tokens: 7037   # chars/4 over the realized diff (28147 chars)
  tasks: 1
  commits: 2
plan_head_before: 98dcf868af8ebb531a5757a717dac4d79578bc6f

tech-stack:
  added: []
  patterns:
    - "Reconcile-by-repo-set: decide every edge from (placeholder, candidate set, caller scope) and write only diffs"
    - "Per-caller-repo graph.Scope built once per pass and cached in a local map"

key-files:
  created:
    - internal/resolver/cobol_policy.go
    - internal/resolver/cobol_reconcile_test.go
  modified:
    - internal/resolver/cobol_references.go
    - internal/resolver/resolver.go

key-decisions:
  - "With an empty changed set (full ResolveAll, unscoped) the pass reconciles every COBOL edge rather than filtering by repositories that hold COBOL nodes, so an unowned caller is still stamped caller_unowned in a full pass; for owned callers the two sets are equivalent"
  - "A legacy unresolved edge without cobol_placeholder uses its own e.To as the placeholder; an unresolved edge with cobol_placeholder is classified by the Meta value, so a generic restub to a bare name is still reconciled"
  - "Origin and CrossRepo take part in the diff: a To or CrossRepo change goes through ReindexEdges, a Meta- or Origin-only change through PersistEdgeAttributesBatch, no change writes nothing"
  - "cobol_candidate_ids comparison tolerates []any (SQLite decode) vs []string (memory) so the second pass is a true no-op on disk stores"
  - "cobolPolicyFor carries //nolint:unused until 04-14 reads ResolveFrom in decideCobol (CI enforces the unused linter)"

patterns-established:
  - "COBOL bindings are owned by reconcileCobol: the full path calls reconcileCobol(r.scope) under ResolveAll's lock; incremental tails call ReconcileCobol(changed)"

requirements-completed: [SEC-03, ID-07]

coverage:
  - id: D1
    description: "One reconcile pass re-decides placeholders and already-bound COBOL edges (ambiguous on a second candidate, missing when the target is gone, rebinds to the same ID when it returns)"
    requirement: ID-07
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_RecheckBound"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_TargetGone"
        status: pass
    human_judgment: false
  - id: D2
    description: "The pass is idempotent on a SQLite store: second run leaves To/Meta identical and issues zero reindex and zero Meta writes"
    requirement: ID-07
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_Idempotent"
        status: pass
    human_judgment: false
  - id: D3
    description: "Binding is confined to the caller repository via graph.Scope; the reconcile set limits which edges are touched; unowned callers never bind"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_ScopedRepos"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_reconcile_test.go#TestCobolReconcile_CallerUnowned"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolNamespaceExclusions"
        status: pass
    human_judgment: false
  - id: D4
    description: "Existing COBOL binding behavior through ResolveAll is unchanged (program ladder, nested programs, copybooks, SQLite persistence)"
    verification:
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/resolver/ -run Cobol"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/resolver/"
        status: pass
    human_judgment: false
  - id: D5
    description: "CobolPolicy types and SetCobolPolicyLookup exist for 04-07 to wire config into (no behavior yet)"
    verification:
      - kind: other
        ref: "GOWORK=off go build ./..."
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 06: COBOL Reconcile Pass Summary

**`reconcileCobol(changed)` replaces the placeholder-only `resolveCobolReferences`. It re-decides placeholder and already-bound COBOL CALL/COPY edges from `cobol_placeholder`, and filters candidates through a per-caller `graph.Scope`. Unowned callers fail closed with `caller_unowned`, and only changed edges are written, so a second run is a no-op.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-10-13T01:56:14Z
- **Completed:** 2026-10-13T02:02:38Z
- **Tasks:** 1 (TDD: RED + GREEN)
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- One function decides COBOL bindings for both paths. `ResolveAllContext` calls `r.reconcileCobol(r.scope)`, and incremental tails (04-08) call the exported `ReconcileCobol(changed)`, which takes `r.mu`.
- Bound edges are re-checked on every pass (WR-01). A second candidate returns the edge to its placeholder as `ambiguous` with sorted IDs. A vanished target gives `missing`. When the target returns, the edge rebinds to the same ID and the reason and candidate IDs are cleared.
- Fail closed on missing ownership. A caller node with no `RepoPrefix` keeps the placeholder with `cobol_unresolved_reason=caller_unowned`, and an already-bound edge from such a caller is unbound.
- Candidates pass through `graph.NewScope("", "", [callerRepo])` and `Scope.Allows` (D-16). 04-14 only has to add `ResolveFrom` to the allowed set.
- The pass writes only diffs. `To`/`CrossRepo` changes go through `ReindexEdges` (with `noteImportEdgeReindexes`), Meta/Origin-only changes go through `PersistEdgeAttributesBatch`, and no change writes nothing. On SQLite, the second-run write count is verified as zero.
- `CobolPolicy`, `CobolLibrary`, `CobolPolicyLookup`, and `SetCobolPolicyLookup` are in place for 04-07. A nil lookup means same-repository only.

## Task Commits

1. **Task 1 RED: failing reconcile tests** - `3a4567b0` (test)
2. **Task 1 GREEN: reconcile pass + policy types** - `e3e727ec` (feat)

REFACTOR: none needed.

## TDD Gate Compliance

- **RED** (`3a4567b0`): `GOWORK=off go test -count=1 ./internal/resolver/ -run 'TestCobolReconcile_'` exited 1. RecheckBound failed: the bound edge was not re-decided. Idempotent failed: "no second Meta write", expected 0, got 2. ScopedRepos failed: repoB's edge was stamped `missing`. CallerUnowned failed: reason `missing` instead of `caller_unowned`, and the bound line-5 edge stayed bound. TargetGone passed in RED because the old pass already handled placeholder→missing→bind; its new assertions only hold in GREEN through the shared function. RED used a stub `ReconcileCobol` that delegated to the old pass, so the tests compiled and failed on assertions rather than on a build error.
- **GREEN** (`e3e727ec`): all five `TestCobolReconcile_*` tests and every pre-existing COBOL resolver test pass under `-race`.

## Files Created/Modified
- `internal/resolver/cobol_references.go`: `ReconcileCobol` / `reconcileCobol` (enumeration + diff-only writes), `decideCobol` (scope filter + unchanged program ladder + 0/1/≥2 rule), `cobolDecision`, `cobolIDsEqual`, and the `metaCobolPlaceholder` const. The PURPOSE/RATIONALE/KEYWORDS doc was rewritten for reconcile semantics.
- `internal/resolver/cobol_policy.go` (new): policy types, the setter, and `cobolPolicyFor`.
- `internal/resolver/resolver.go`: the `cobolPolicyLookup` field, the call site `r.reconcileCobol(r.scope)`, and one ownership comment updated.
- `internal/resolver/cobol_reconcile_test.go` (new): the five `TestCobolReconcile_*` tests plus a `cobolWriteCounter` SQLite wrapper that counts writes.

## Decisions Made
See `key-decisions` in the frontmatter. The main interpretation: with an empty `changed`, the pass reconciles all COBOL edges. The plan's set is "repositories holding COBOL nodes". For owned callers this is equivalent, and it ensures unowned callers are still stamped in a full pass.

## Deviations from Plan

**1. [Rule 3 - Blocking] Test helper renamed to avoid a package-level collision**
- **Found during:** Task 1 RED
- **Issue:** The test package already declares `countingStore` in `scoped_global_passes_test.go`.
- **Fix:** Named the new SQLite write counter `cobolWriteCounter`.
- **Files modified:** internal/resolver/cobol_reconcile_test.go
- **Committed in:** 3a4567b0

**2. [Rule 3 - Blocking] `//nolint:unused` on `cobolPolicyFor`**
- **Found during:** Task 1 GREEN
- **Issue:** The plan requires the private `cobolPolicyFor`, but this plan's allowed set is the caller repository alone, so nothing reads it yet. CI's `standard` golangci-lint set includes `unused`.
- **Fix:** Added a scoped `//nolint:unused` directive naming 04-14 as the consumer. 04-14 should remove it when `decideCobol` reads `ResolveFrom`.
- **Files modified:** internal/resolver/cobol_policy.go
- **Committed in:** e3e727ec

---

**Total deviations:** 2 auto-fixed (2 blocking).
**Impact on plan:** None on behavior. Both were mechanical.

## Issues Encountered
- Comments in files outside this plan still name `resolveCobolReferences`: `internal/resolver/lsp_resolve.go:520`, `internal/resolver/cross_repo.go:1194`, `internal/indexer/incremental_resolve.go:453`, `internal/indexer/incremental_resolve_test.go:295`, and `internal/parser/languages/cobol_grammar_refs.go:20` / `cobol_grammar_copybook.go:70`. They were left untouched to keep the diff to planned files, and the acceptance criterion allows comments. 04-14 or a later doc sweep can rename them.
- `gofmt -l internal/resolver/` flags `csharp_verbatim_type_ref_test.go`. This is pre-existing and not touched by this plan.
- Neither the full `./internal/resolver/` package under `-race` (33 s) nor the COBOL-related tests in `internal/indexer`, `cmd/gortex`, `internal/neo4jprojection`, and `internal/mcp` showed regressions. The intermittent `FAIL ./cmd/gortex` noted by earlier plans did not occur; that package was run alone.

## User Setup Required
None. No external service configuration required.

## Next Phase Readiness
- 04-07 can wire config into `SetCobolPolicyLookup`.
- 04-08 can call `ReconcileCobol(changed)` from every mutation tail.
- 04-14 extends `decideCobol` with `ResolveFrom` union ambiguity, library order, and qualifier links. It also adds the reverse-allowlist expansion of the reconcile set and should drop the `nolint`.

## Self-Check: PASSED
- FOUND: internal/resolver/cobol_policy.go, internal/resolver/cobol_reconcile_test.go, internal/resolver/cobol_references.go
- FOUND commits: 3a4567b0, e3e727ec
- Plan verify command exit 0. Acceptance greps: `ReconcileCobol(` 1 line, `r.reconcileCobol(r.scope)` 1 line, `resolveCobolReferences` in comments only.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*
