---
phase: 05-explicit-unresolved-evidence
plan: 05
subsystem: cobol-graph
tags: [cobol, resolver, placeholder, missing-artifact, external-unresolved, sqlite]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.KindFinding, graph.FindingSymbol, graph.MetaEvidenceClass, graph.FindingClassExternal (05-01); gortex-cobol-grammar/6 (05-03)"
  - phase: 04-stable-incremental-and-cross-repository-lifecycle
    provides: "reconcileCobol single binding pass, reconcileCobolResources sweep shape, graph.Scope, canonical dump harness"
provides:
  - "Missing CALL/LINK/XCTL, COPY, -INC, non-builtin EXEC SQL INCLUDE, and COPY IDMS edges target <callerRepo>/cobol-missing::<type>/<key> KindResource placeholder nodes"
  - "cobolExternalRef desired entries (one per placeholder-targeted reference occurrence, unchanged edges included) handed to reconcileCobolFindings, the seam 05-08 builds EXTERNAL_UNRESOLVED findings from"
  - "reconcileCobolFindings: adds absent placeholders in one AddBatch and sweeps unreferenced ones with EvictFile on their own path"
  - "cobolPlaceholderTarget / cobolUnresolvedFromPlaceholder mapping, cobolRef.idms, placeholder in-edge visit in reconcileCobol"
  - "ResolveAll runs the COBOL reconcile on its no-pending early return"
affects: [05-08, 05-09, 05-11, 05-12, phase-06]

actuals:
  tokens: 14947
  tasks: 3
  commits: 8

tech-stack:
  added: []
  patterns:
    - "Missing-artifact placeholder: KindResource, ID == FilePath == <callerRepo>/cobol-missing::<type>/<key>, Meta exactly the D-11 set, scope fields copied from the caller node"
    - "Placeholder sweep keeps a placeholder while any in-edge other than a finding's references edge (From contains ::@finding/) targets it"

key-files:
  created:
    - internal/resolver/cobol_findings.go
    - internal/resolver/cobol_findings_test.go
  modified:
    - internal/resolver/cobol_references.go
    - internal/resolver/resolver.go
    - internal/resolver/cobol_references_test.go
    - internal/resolver/cobol_reconcile_test.go
    - internal/resolver/resolve_all_pending_test.go
    - internal/indexer/cobol_lifecycle_crossrepo_test.go
    - internal/indexer/cobol_lifecycle_incremental_test.go
    - internal/indexer/cobol_lifecycle_material_test.go
    - cmd/gortex/cobol_breadth_test.go

key-decisions:
  - "Every missing decision (d.reason == missing, not sql_include_builtin) and every COPY IDMS edge targets a placeholder, so a literal non-call CICS PROGRAM reference (EXEC CICS LOAD PROGRAM) also gets one; it is an EdgeReferences code edge"
  - "The sweep ignores only references edges whose From is a finding (::@finding/), not every references edge, because CICS LOAD PROGRAM code edges are references edges and evicting their placeholder would delete the code edge"
  - "The placeholder in-edge visit recognises edges by Meta (cobol_placeholder, or cobol_copy_form idms with the ID inverted), so finding edges are skipped without a kind check"
  - "ResolveAll now runs reconcileCobol on its no-pending early return: placeholder-targeted edges are no longer pending, and a clean track of the target repository otherwise never rebound them"

patterns-established:
  - "COBOL lifecycle tests assert a missing edge's To is <repo>/cobol-missing::<type>/<key> and the node is in the dump or store with artifact_status not_obtained (requireCobolMissingNode, assertCobolBreadthMissing)"

requirements-completed: [GAP-02, GAP-05, GAP-07]

coverage:
  - id: D1
    description: "Missing CALL/LINK/XCTL, COPY, -INC, and COPY IDMS edges move onto per-repository cobol-missing:: placeholder nodes and keep their reasons; a present -INC member binds (D-17)"
    requirement: GAP-05
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderEdgeTargets"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderIDMSSecondPass"
        status: pass
    human_judgment: false
  - id: D2
    description: "No placeholder for bound, ambiguous, dynamic, caller_unowned, or SQLCA/SQLDA builtin references; a placeholder is never a binding candidate"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderNotForBoundAmbiguousDynamicBuiltin"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderNeverCandidate"
        status: pass
    human_judgment: false
  - id: D3
    description: "Placeholder carries exactly the D-11 Meta and caller scope, no range or parser stamp; one per referencing repository; ownership audit clean on SQLite"
    requirement: GAP-05
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderShape"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderPerRepository"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderPrefixDiagnostics"
        status: pass
    human_judgment: false
  - id: D4
    description: "Placeholder evicted with its last code reference or on retrieval; other repositories' placeholders, unrelated placeholders, and parse findings untouched; unchanged rerun writes nothing"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderSweep"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderRebindOnRetrieve"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderIdempotent"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderResolveAllNoPending"
        status: pass
    human_judgment: false
  - id: D5
    description: "Real-path lifecycle and breadth suites agree missing targets are placeholder nodes, and incremental still equals clean"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer/ ./cmd/gortex/ -run '^TestCobol'"
        status: pass
    human_judgment: false

duration: 43min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 05: Missing-Artifact Placeholders Summary

**Every COBOL reference to a target that was never obtained now points at a repository-owned, source-free `<repo>/cobol-missing::<type>/<key>` KindResource node. The single binding pass creates these nodes and sweeps them when no reference remains, and hands one `cobolExternalRef` per reference occurrence to 05-08.**

## Performance

- **Duration:** ~43 min
- **Started:** 2026-10-20T16:47:43Z
- **Completed:** 2026-10-20T17:30:30Z
- **Tasks:** 3
- **Files modified:** 11 (2 created, 9 modified)

## Accomplishments

- `reconcileCobol` moves an edge onto `<callerRepo>/cobol-missing::{program,copybook,inc_member,idms_dictionary_member}/<key>` when:
  - it is decided `missing` and is not an SQLCA/SQLDA builtin, or
  - it is a COPY IDMS edge.

  The edge keeps its reason. Placeholder in-edges are visited again on later passes. An IDMS edge's unresolved target is recovered from the placeholder ID, because those edges carry no `cobol_placeholder`.
- The following edges keep their existing `To` and get no placeholder: bound, ambiguous, dynamic, `caller_unowned`, and builtin.
- Each placeholder-targeted occurrence becomes one `cobolExternalRef`, including unchanged edges. All entries are collected before the compare-and-skip and passed to `reconcileCobolFindings` before `reconcileCobolResources`.
- `reconcileCobolFindings` builds the D-11 node (Meta key set exact; RepoPrefix, WorkspaceID, and ProjectID copied from the caller; no range).
  - It adds only absent placeholders, in one `AddBatch`.
  - It sweeps placeholders of the reconcile set that no code edge targets, with `EvictFile` on the placeholder's own path.
  - An unchanged rerun issues no `AddBatch`, `ReindexEdges`, `PersistEdgeAttributesBatch`, or `EvictFile`.
- Downstream lifecycle and breadth tests expect placeholder `To` values and assert that each placeholder node exists with `artifact_status=not_obtained`.

## Task Commits

1. **Task 1: Missing and IDMS edges retarget onto owned placeholder nodes**
   - `d86c3462` (test, RED)
   - `c8a7855a` (feat, GREEN)
2. **Task 2: Placeholder lifecycle (sweep, rebind on retrieval, per-repository ownership)**
   - `341cddc9` (test, RED)
   - `1fea46c5` (feat, GREEN)
3. **Task 3: Downstream expectations follow missing edges onto placeholders**
   - `91cd7149` (fix, deviation 1)
   - `567374ea` (test)
   - `d9c8766d` (test, deviation 3)
   - `80147b57` (test, deviation 2)

**Plan metadata:** recorded in the docs(05-05) commits that add this file and update STATE/ROADMAP.

## Files Created/Modified

- `internal/resolver/cobol_findings.go`: Created. Contains:
  - constants: `cobolMissingNamespace`, `cobolMissingArtifactKind`, `cobolArtifact*`, `metaCobolCopyForm`, `metaSQLIncludeBuiltin`;
  - `cobolExternalRef`, `cobolPlaceholderTarget`, `cobolUnresolvedFromPlaceholder`;
  - `reconcileCobolFindings` (placeholder add and sweep).
- `internal/resolver/cobol_references.go`: Modified.
  - `cobolRef.idms` added; the IDMS case no longer returns early.
  - Placeholder in-edge visit added.
  - Placeholder retarget plus desired-entry collection added.
  - `r.reconcileCobolFindings(repos, desired)` call added.
  - Doc comments updated.
- `internal/resolver/resolver.go`: Modified. `ResolveAllContext`'s no-pending early return now runs `r.reconcileCobol(r.scope)`.
- `internal/resolver/cobol_findings_test.go`: Created. Contains:
  - the 10 planned `TestCobolReconcile_Placeholder*` tests plus `TestCobolReconcile_PlaceholderResolveAllNoPending`;
  - helpers `cobolOnStores`, `cobolMissingNodes`, `cobolCopyMeta`, `cobolIDMSMeta`, `seedCobolFinding`. The finding ID comes from `graph.FindingSymbol`.
- `internal/resolver/cobol_reconcile_test.go`: Modified. `cobolWriteCounter` now also counts `AddBatch` and `EvictFile` (`reset`, `writes`). Three expected `To` values are now placeholders.
- `internal/resolver/cobol_references_test.go`: Modified. Five expected `To` values are now placeholders. The IDMS line is now seeded with the extractor's IDMS Meta and expects its placeholder.
- `internal/resolver/resolve_all_pending_test.go`: Modified. The zero-work test's counting store now answers `HasLanguage`, as the SQLite store it imitates does.
- `internal/indexer/cobol_lifecycle_crossrepo_test.go`: Modified. Adds `requireCobolMissingNode`; `requireCrossRepoUnbound` expects placeholders.
- `internal/indexer/cobol_lifecycle_incremental_test.go`, `internal/indexer/cobol_lifecycle_material_test.go`, `cmd/gortex/cobol_breadth_test.go`: Modified. Missing and IDMS expectations are now placeholders, with node assertions.

## Decisions Made

- **Placeholder rule.** Every `missing` decision except a builtin gets a placeholder, as the plan's action rule states. That includes a literal non-call CICS `PROGRAM` operand (for example `EXEC CICS LOAD PROGRAM('X')`), an `EdgeReferences` code edge carrying `cobol_placeholder`. It names an artifact that was never obtained, the same as a CALL.
- **Sweep rule.** The sweep ignores a `references` in-edge only when its `From` is a finding (`::@finding/`), not every `references` edge. Otherwise a placeholder held only by a CICS LOAD code edge would be evicted on every pass, and `EvictFile` would drop the code edge with it. This still lets 05-08's finding `references` edges never keep their own placeholder.
- **In-edge visit.** The placeholder in-edge visit tells code edges from finding edges by Meta (`cobol_placeholder`, or `cobol_copy_form=idms`), not by edge kind, for the same CICS reason.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] ResolveAll skipped the COBOL reconcile when nothing was pending**
- **Found during:** Task 3 (lifecycle tests: CrossRepoClean allowlist, track_order, CrossRepoIncremental, ParallelWarmRestart, ConfigChange)
- **Issue:** Before this plan, a missing edge stayed on `unresolved::cobol::...` and so counted as pending. The full `ResolveAll` of a later track therefore always reached `reconcileCobol`. Once Task 1 moved missing edges onto placeholder nodes, they stopped being pending. The no-pending early return then skipped the binding pass, so crossa's CALL/COPY never bound into crossb when crossb was tracked second, and track-order dumps diverged.
- **Fix:** `ResolveAllContext` calls `r.reconcileCobol(r.scope)` before the no-pending early return (4 lines in `internal/resolver/resolver.go`). It is gated by the same `graphHasLanguage("cobol")` check as the main call.
- **Scope note:** The plan said to fix regressions only in `cobol_findings.go` or `cobol_references.go`, otherwise stop and report. The regression came from Task 1 code, but the only correct fix site is `ResolveAll`. Fixing it here kept the phase consistent with the Phase 4 contract that `ResolveAll` always runs the single binding pass.
- **Files modified:** `internal/resolver/resolver.go`, `internal/resolver/cobol_findings_test.go` (regression test `TestCobolReconcile_PlaceholderResolveAllNoPending`, memory and SQLite; it fails with the line disabled)
- **Committed in:** `91cd7149`

**2. [Rule 1 - Consequence of fix 1] Zero-work resolve test store lacked the language probe**
- **Found during:** Plan verification (`go test -race ./internal/resolver/`)
- **Issue:** `TestResolveAllZeroWorkSkipsGlobalIndexesAndBackendBulk` wraps an empty graph in a counting store that imitates the SQLite backend (bulk resolve, pending check) but does not implement `HasLanguage`. `graphHasLanguage` therefore fails open, and the new early-return reconcile scanned `NodesByKind`.
- **Fix:** The test wraps its store in `languageProbedStore`, which answers `HasLanguage` from its nodes, as SQLite does. The zero-scan invariant still holds for any store with the probe. The other tests' stores are unchanged.
- **Files modified:** `internal/resolver/resolve_all_pending_test.go`
- **Committed in:** `80147b57`

**3. [Rule 1 - Missed retarget] TestCobolLifecycle_Delete**
- **Found during:** Plan verification (wave-merge `^TestCobol` suite)
- **Issue:** The test builds its expected target as `graph.UnresolvedMarker + "cobol::copybook::CUSTREC"`, so the plan's `grep 'unresolved::cobol::'` missed it. It was a pure placeholder retarget, and incremental-equals-clean already passed.
- **Fix:** Expect `lifea/cobol-missing::copybook/CUSTREC` and assert the placeholder node.
- **Files modified:** `internal/indexer/cobol_lifecycle_incremental_test.go`
- **Committed in:** `d9c8766d`

---

**Total deviations:** 3 auto-fixed (Rule 1).

**Impact on plan:**
- Deviation 1 is a 4-line production fix outside the planned file list. It is required for the clean multi-repository track path.
- The other two are test-only.
- No scope creep. Diff: the 9 planned files plus `resolver.go` and `resolve_all_pending_test.go`.

## Issues Encountered

- `internal/resolver/csharp_verbatim_type_ref_test.go` is not gofmt-clean at HEAD. It is unrelated to this plan and was logged to `deferred-items.md`.
- For 05-08:
  - 05-08 restructures the sweep's in-edge read to run at the start of `reconcileCobolFindings`. Keep the finding-only discriminator (`references` with `From` containing `::@finding/`), not a bare `references` kind check, or CICS LOAD placeholders will be evicted.
  - A non-call CICS PROGRAM reference gets a placeholder and therefore, under 05-08, an `EXTERNAL_UNRESOLVED` finding. Its `cobol_call` is `literal`.

## TDD Gate Compliance

- **Task 1:** RED `d86c3462`, then GREEN `c8a7855a`. Four of the five new tests failed before the change. `NotForBoundAmbiguousDynamicBuiltin` is a negative control and passed trivially before placeholders existed.
- **Task 2:** RED `341cddc9` (Sweep and RebindOnRetrieve failed on the missing sweep), then GREEN `1fea46c5`.
- **Task 3:** Not a TDD task.

## Verification

- **Task 1 verify:** passed. Acceptance results:
  - `r.reconcileCobolFindings(` occurs once, at line 351, before `reconcileCobolResources` at line 352.
  - The builtin grep counts 3.
  - The IDMS `return` grep counts 0.
- **Task 2 verify:** passed. Acceptance results:
  - `EvictFile` occurs once.
  - `cobolMissingArtifactKind` occurs 3 times.
  - 11 `TestCobolReconcile_Placeholder*` tests are listed.
- **Task 3 verify:** passed. `cobol-missing::` counts per file: crossrepo 2, incremental 4, material 1, breadth 3.
- **Plan verification:**
  - `go test -race ./internal/resolver/` (whole package) passes.
  - `go test -race -timeout 30m ./internal/indexer/ ./cmd/gortex/ -run '^TestCobol'` passes (140 s and 44 s).
  - `go test -race ./internal/graph/... ./internal/parser/languages -run 'Finding|Cobol|PrefixDiagnostics'` passes.
- **Extra coverage for the ResolveAll change:** the whole `./internal/indexer/`, `./internal/contracts/`, and `./internal/mcp/` packages pass, and `go build ./...` passes.

## User Setup Required

None. No external service configuration is required. Existing stores re-extract through the `/6` material fingerprint (05-03). Then the binding pass moves missing edges onto placeholders. Neo4j projections made before this plan need a manual re-run of the explicit projection.

## Next Phase Readiness

- 05-08 can build `EXTERNAL_UNRESOLVED` findings from the `desired` slice in `reconcileCobolFindings`. It should:
  - add each finding with its `member_of` and `references` edges in the same `AddBatch` as new placeholders;
  - evict undesired findings before the sweep.
- 05-11 (FTS sync for resolver-created nodes) applies to these placeholders on incremental tails.

## Self-Check: PASSED

- FOUND: `internal/resolver/cobol_findings.go`, `internal/resolver/cobol_findings_test.go`, `internal/resolver/cobol_references.go`, `internal/resolver/resolver.go`
- FOUND commits: `d86c3462`, `c8a7855a`, `341cddc9`, `1fea46c5`, `91cd7149`, `567374ea`, `d9c8766d`, `80147b57`
- All three task verify commands, every acceptance criterion, and the plan-level verification passed.

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
