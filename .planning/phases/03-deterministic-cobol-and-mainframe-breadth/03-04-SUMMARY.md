---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 04
subsystem: resolver
tags: [cobol, resolver, unresolved-namespace, exactly-one, same-repo, sqlite, copybook]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-01 unresolved::cobol:: placeholder contract (program, copybook, dynamic, label, data, idms_scope) and cobol-name-norm/1 keys"
provides:
  - "cobol:: owned-namespace exits in Resolver.resolveEdge, CrossRepoResolver.resolveEdge, identifierFromTarget, reuseIdentifier"
  - "Resolver.resolveCobolReferences: exactly-one same-repo binding for unresolved::cobol::program:: and unresolved::cobol::copybook:: targets"
  - "Edge Meta cobol_unresolved_reason (missing | ambiguous) and cobol_candidate_ids (sorted full node IDs)"
affects: [03-06, 03-07, 03-09, 04-lifecycle, 05-findings, 06-queries]

actuals:
  tokens: 4580
  tasks: 2
  commits: 4
plan_head_before: 47ca3ed2e90231fd13ee940b7562b92e331b18dc

tech-stack:
  added: []
  patterns:
    - "COBOL placeholders live under unresolved::cobol:: and every generic resolver path exits on a cobol:: prefix; resolveCobolReferences is the single owner"
    - "Candidate index keyed by RepoPrefix + NUL + cobol_name_key, so a lookup can never see another repository"
    - "Rebinds go through ReindexEdges (persists Meta too); missing/ambiguous stamps go through EdgeMetaBatchPersister when the store implements it"

key-files:
  created:
    - internal/resolver/cobol_references.go
    - internal/resolver/cobol_references_test.go
  modified:
    - internal/resolver/resolver.go
    - internal/resolver/cross_repo.go
    - internal/resolver/lsp_resolve.go
    - internal/indexer/incremental_resolve.go
    - internal/indexer/incremental_resolve_test.go

key-decisions:
  - "resolveCobolReferences reads the target through graph.UnresolvedName, the same helper resolveEdge uses, so both the bare and repo-prefixed unresolved forms are handled"
  - "Copybook rebinds call noteImportEdgeReindexes before ReindexEdges, as resolveLuaRequires does, so the import-adjacency retention stays coherent"
  - "Stamps are rewritten on every full resolve (no change detection): the values are identical, and SQLite decodes cobol_candidate_ids as []any, which would make an equality check fragile"
  - "Same-file narrowing applies to programs only; a copybook is always another file"

patterns-established:
  - "Pattern: an owned resolver namespace needs one exit per generic path (resolveEdge, CrossRepoResolver.resolveEdge, identifierFromTarget, reuseIdentifier) plus one whole-graph pass in the ResolveAllContext settle window"

requirements-completed: [DET-03, DET-04, DET-06]

coverage:
  - id: D1
    description: "No generic resolver path binds, LSP-probes, classifies terminal, cross-repo-stitches, or reuses a cobol:: target"
    requirement: DET-04
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolNamespaceExclusions"
        status: pass
      - kind: unit
        ref: "internal/indexer/incremental_resolve_test.go#TestReuseIdentifierSkipsCobolNamespace"
        status: pass
    human_judgment: false
  - id: D2
    description: "Literal program calls bind only to exactly one same-repo program, same-file first; missing and ambiguous are stamped with reason and sorted candidate IDs; never across repositories; second resolve is identical; rebinds and stamps survive a SQLite close and reopen"
    requirement: DET-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReferences_ProgramBinding"
        status: pass
    human_judgment: false
  - id: D3
    description: "Copybook references bind only to exactly one same-repo copybook; ambiguous and cross-repo cases are stamped; COPY IDMS, dynamic, label, data, and idms_scope targets keep their To and extractor Meta unchanged"
    requirement: DET-06
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReferences_Copybook"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 04: COBOL Resolver Namespace and Exactly-One Binding Summary

**COBOL placeholders get their own `unresolved::cobol::` namespace. Every generic resolver path now skips it. A new whole-graph pass, `resolveCobolReferences`, binds a literal CALL or COPY only when exactly one program or copybook in the caller's repository has the same `cobol_name_key`, and stamps `missing` or `ambiguous` (with sorted candidate IDs) otherwise.**

## Performance

- **Duration:** about 9 min
- **Started:** 2026-10-02T06:42:13Z
- **Completed:** 2026-10-02T06:50:47Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Four one-line `cobol::` exits: `Resolver.resolveEdge` returns the edge untouched, `CrossRepoResolver.resolveEdge` counts it unresolved before `resolveFunctionCall`, and `identifierFromTarget` and `reuseIdentifier` return "". Because `identifierFromTarget` returns "", the inline LSP path (`tryResolveViaLSP`), the deferred bulk LSP path (`lspDeferTarget`), and terminal classification all skip COBOL targets with no extra guard.
- `internal/resolver/cobol_references.go` (154 lines): candidate indexes for programs (`KindFunction`) and copybooks (`KindFile`), keyed by RepoPrefix and `cobol_name_key`. The pass sweeps `EdgesWithUnresolvedTarget`, narrows programs to same-file candidates first, then binds, stamps `missing`, or stamps `ambiguous`. It never touches `copybook::idms/`, `dynamic::`, `label::`, `data::`, or `idms_scope::` targets.
- Called from `ResolveAllContext` right after `resolveLuaRequires`.
- A rebind clears stale `cobol_unresolved_reason` / `cobol_candidate_ids` and sets Origin `ast_resolved`. A SQLite subtest closes and reopens the store and confirms the rebind and both stamps were persisted. Disabling the stamp persistence makes that subtest fail.

## Task Commits

1. **Task 1: Own the cobol:: namespace in every generic resolution path** - `a1b9005f` (test, RED), `09eaa915` (feat, GREEN)
2. **Task 2: resolveCobolReferences binds literal calls and copybooks to exactly one same-repository candidate** - `c0611aae` (test, RED), `2b5f6373` (feat, GREEN)

## Files Created/Modified
- `internal/resolver/cobol_references.go` - the binding pass and `cobolCandidateIndex`
- `internal/resolver/cobol_references_test.go` - `seedCobol` / `cobolEdge` / `cobolEdgeAt` helpers, `TestCobolNamespaceExclusions`, `TestCobolReferences_ProgramBinding` (exactly_one, missing, ambiguous, same_file_first, sqlite_persisted, idempotent), `TestCobolReferences_Copybook`
- `internal/resolver/resolver.go` - `cobol::` case in `resolveEdge` (+2 lines), `resolveCobolReferences()` call site (+2 lines)
- `internal/resolver/cross_repo.go` - `cobol::` case counting unresolved (+2 lines)
- `internal/resolver/lsp_resolve.go` - `cobol::` added to the owned case list (+2/-1)
- `internal/indexer/incremental_resolve.go` - `cobol::` added to `reuseIdentifier` (+2/-1)
- `internal/indexer/incremental_resolve_test.go` - `TestReuseIdentifierSkipsCobolNamespace`

## TDD Evidence
- **Task 1 RED** (`a1b9005f`): `go test ./internal/resolver/ -run '^TestCobolNamespaceExclusions$'` exited 1. The `identifier` subtest failed on its own assertion (expected "", actual `cobol::program::PGMX`, and likewise for the other two targets). The `generic` and `crossrepo` subtests already passed before the change: the generic and cross-repo name lookups find no node named `cobol::…`, so with today's fixtures they are regression guards, not RED drivers. **GREEN** (`09eaa915`): the Task 1 verify command passed under `-race`.
- **Task 2 RED** (`c0611aae`): all five `TestCobolReferences_ProgramBinding` subtests and `TestCobolReferences_Copybook` failed on assertions (for example, expected `a/sub.cbl::SUBPGM`, actual `unresolved::cobol::program::SUBPGM`; expected reason `missing`, actual nil). `TestReuseIdentifierSkipsCobolNamespace` failed with actual `cobol::copybook::Y`. None of the failures were panics or compile errors. **GREEN** (`2b5f6373`): the whole `internal/resolver` package and the selected indexer tests passed under `-race`.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so these RED records were captured by hand (same as 03-01).

## Decisions Made
- The target is read through `graph.UnresolvedName`, the same helper `resolveEdge` uses, so repo-prefixed unresolved forms are covered too.
- Copybook rebinds are `imports` edges, so the batch goes through `noteImportEdgeReindexes` before `ReindexEdges`, matching `resolveLuaRequires`.
- Stamps are rewritten on every full resolve. Re-stamping writes identical values, and after a SQLite round trip `cobol_candidate_ids` decodes as `[]any`, so an equality check would be fragile. Cost is one batch per full resolve, gated on COBOL being present (T-03-15, accepted).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] Import-adjacency invalidation on copybook rebinds**
- **Found during:** Task 2
- **Issue:** The plan's pass shape called `ReindexEdges` directly. Copybook edges are `EdgeImports`, and the resolver keeps an import-adjacency retention that `resolveLuaRequires` invalidates with `noteImportEdgeReindexes` before reindexing. Skipping that call would leave stale import adjacency for the caller's file.
- **Fix:** `r.noteImportEdgeReindexes(rebound)` runs before `r.graph.ReindexEdges(rebound)`. It ignores non-import rows, so program rebinds are unaffected.
- **Files modified:** internal/resolver/cobol_references.go
- **Verification:** the whole `internal/resolver` package passes under `-race`
- **Committed in:** 2b5f6373

**2. [Rule 2 - Missing critical] SQLite persistence subtest**
- **Found during:** Task 2
- **Issue:** The truth "persists To changes through ReindexEdges and Meta-only stamps through EdgeMetaBatchPersister" was only exercised on the in-memory graph. That graph never implements the persister, so it would not catch a dropped stamp write.
- **Fix:** Added `TestCobolReferences_ProgramBinding/sqlite_persisted`. It resolves against `store_sqlite`, closes the store, reopens it, and asserts the rebind and both stamps. A mutation check that disables `PersistEdgeAttributesBatch` makes it fail.
- **Files modified:** internal/resolver/cobol_references_test.go
- **Committed in:** 2b5f6373

---

**Total deviations:** 2 auto-fixed (2 missing critical)
**Impact on plan:** Both keep cache and persistence correct. No scope creep; the seven planned files are the only files changed.

## Issues Encountered
- `gofmt -l internal/resolver/` lists `internal/resolver/csharp_verbatim_type_ref_test.go`. That file is pre-existing and was not touched by this plan; every file this plan touches is gofmt-clean.

## Known Limits (carried from the plan, not gaps)
- Whole-graph passes run only in `ResolveAllContext`, so a single-file incremental reindex leaves new COBOL targets unbound until the next full resolve. Lifecycle is Phase 4.
- `COPY … OF/IN <library>` qualifiers do not narrow candidates. Library search order is Phase 4.

## Known Stubs
None.

## Threat Flags
None. T-03-12 (repo-scoped, exactly-one, sorted candidate IDs) and T-03-13 (owned-namespace exits) are mitigated as planned and covered by the tests above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 03-06 and 03-07 can emit `unresolved::cobol::program::<KEY>` / `unresolved::cobol::copybook::<KEY>` targets and node Meta `cobol_kind` + `cobol_name_key`, and binding works with no further resolver change.
- The extractor must keep keys exactly as normalized under cobol-name-norm/1. The resolver compares by string equality and has no normalization of its own.

## Self-Check: PASSED
- FOUND: internal/resolver/cobol_references.go
- FOUND: internal/resolver/cobol_references_test.go
- FOUND: a1b9005f, 09eaa915, c0611aae, 2b5f6373
- Task 1 and Task 2 verify commands pass under `-race`. `git diff --name-only 47ca3ed2..HEAD` lists only the seven planned files.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
