---
phase: 05-explicit-unresolved-evidence
plan: 08
subsystem: cobol-graph
tags: [cobol, resolver, finding, external-unresolved, placeholder, sqlite, provenance]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.FindingSymbol, graph.FindingEvidenceDigest, finding Meta constants (05-01); graph.EvictFindingNodesByIDs (05-02); cobolExternalRef desired entries, placeholders, and the placeholder sweep in reconcileCobolFindings (05-05)"
provides:
  - "One EXTERNAL_UNRESOLVED KindFinding node per placeholder-targeted reference occurrence (missing CALL/LINK/XCTL/CICS PROGRAM, COPY, -INC, non-builtin EXEC SQL INCLUDE, every COPY IDMS), built by the single binding pass"
  - "External finding ID `<edge FilePath>::@finding/external/<hex>[#k]`, with #k ordered by prov_start_byte, Line, then edge Kind"
  - "Full RESEARCH Pattern 3 Meta on every external finding (every prov_* key of the edge plus the finding_* contract), member_of edge to the edge's From node, references edge to the placeholder"
  - "Undesired external findings evicted with graph.EvictFindingNodesByIDs before the placeholder sweep; existing findings read from the placeholders' references in-edges (one GetInEdgesByNodeIDs shared with the sweep)"
  - "cobolWriteCounter counts EvictFindingNodesByIDs (resolver test helper)"
affects: [05-09, 05-10, 05-11, 05-12, phase-06]

actuals:
  tokens: 10625
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "External finding Meta: copy every prov_* key of the referencing edge verbatim, then override prov_evidence_class and set the finding_* keys through graph.Meta* constants; never copy cobol_* keys or the as-written target"
    - "Existing resolver-owned findings are discovered through the in-edges of the nodes they reference, never by streaming every KindFinding node"

key-files:
  created: []
  modified:
    - internal/resolver/cobol_findings.go
    - internal/resolver/cobol_findings_test.go
    - internal/resolver/cobol_reconcile_test.go
    - cmd/gortex/cobol_findings_test.go

key-decisions:
  - "External evidence is the whitespace-collapsed cobol_call (else cobol_copy_form), cobol_copy_idms_kind when present, and cobol_target_raw (else cobol_copy_member_raw), joined with NUL; CALL, LINK, and XCTL to one target from one container therefore share a base ID and take #2, #3"
  - "QualName carries the #k suffix, as the extractor's parse findings do"
  - "If a store lacks the finding evicter, the pass logs a warning and does not fall back to point deletes (05-02 contract)"
  - "The sweep keeps 05-05's discriminator: only references edges whose From contains ::@finding/ are ignored, so CICS LOAD PROGRAM code edges keep their placeholder"

patterns-established:
  - "Resolver tests seed extractor-shaped edge Meta with cobolCallMeta and cobolAt (prov range keys) and compute expected IDs with cobolExternalID via graph.FindingSymbol"

requirements-completed: [GAP-02, GAP-03, GAP-04, GAP-06, GAP-07]

coverage:
  - id: D1
    description: "Every missing CALL/LINK/XCTL, COPY, -INC, and COPY IDMS reference gets exactly one EXTERNAL_UNRESOLVED finding with member_of and references edges; a bound -INC member gets none (D-17)"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalMissingCall"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalMissingCopyAndInc"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalIDMS"
        status: pass
    human_judgment: false
  - id: D2
    description: "No finding for bound, ambiguous, dynamic, caller_unowned, or SQLCA/SQLDA builtin references (D-09, D-19)"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalNotForBoundAmbiguousDynamicBuiltin"
        status: pass
    human_judgment: false
  - id: D3
    description: "Full RESEARCH Pattern 3 Meta (prov_* copied verbatim, finding_* contract, no cobol_* key, no as-written spelling, allowed value types), exact and absent ranges"
    requirement: GAP-04
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalMetaContract"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_ExternalPersisted"
        status: pass
    human_judgment: false
  - id: D4
    description: "IDs are stable under line shifts, #k ordinals follow byte order, and identical references in two containers stay distinct and unsuffixed (D-01, D-03)"
    requirement: GAP-06
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalOrdinalAndLineShift"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalContainerSeparation"
        status: pass
    human_judgment: false
  - id: D5
    description: "Binding or deleting a reference evicts only its external finding before the sweep; parse findings and other repositories' findings stay; an unchanged rerun writes nothing (ID-07)"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalEvictedWhenUndesired"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalIdempotent"
        status: pass
    human_judgment: false
  - id: D6
    description: "External findings are created only by the resolver, owned per repository, and pass the SQLite prefix audit (GAP-03, SEC-01, SEC-08)"
    requirement: GAP-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalPerRepositoryPrefixDiagnostics"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_ExternalPersisted"
        status: pass
    human_judgment: false
  - id: D7
    description: "Every Phase 2-5 real-path COBOL suite agrees (incremental equals clean with external findings on both sides)"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer/ ./cmd/gortex/ -run '^TestCobol'"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 08: External Unresolved Findings Summary

**Every COBOL reference to a never-obtained program, copybook, -INC member, or IDMS dictionary member now persists as one `EXTERNAL_UNRESOLVED` finding node, `<file>::@finding/external/<hex>[#k]`. The single binding pass builds each finding, keeps it, and removes it. Each finding carries the full RESEARCH Pattern 3 Meta and has a `member_of` edge to its container and a `references` edge to the repository's placeholder.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-10-20T18:11:30Z
- **Completed:** 2026-10-20T18:31:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- `reconcileCobolFindings` now runs this order:
  1. Collect the reconcile set's placeholders.
  2. Read their in-edges once.
  3. Take the existing external findings from the `references` in-edges whose `From` contains `::@finding/external/`.
  4. Build the desired findings.
  5. Evict the undesired ones with `graph.EvictFindingNodesByIDs`.
  6. Add the absent placeholders and findings, with their edges, in one `AddBatch`.
  7. Sweep placeholders over the same in-edge snapshot.
- `newCobolExternalFinding` builds each finding:
  - **Container:** the edge's From node without the repository prefix.
  - **Digest:** covers the call or copy form, the IDMS kind, and the as-written target.
  - **ID:** `graph.FindingSymbol(container, EXTERNAL_UNRESOLVED, external-reference, artifact type, digest)`.
  - **Meta:** every `prov_*` key of the edge, then the `finding_*` contract.
  - **Scope:** RepoPrefix, WorkspaceID, and ProjectID come from the caller node.
  - **Lines:** StartLine is the edge Line. EndLine is `prov_end_row + 1`.
- Identical references in one container get `#k` in order of `prov_start_byte`, then Line, then edge Kind. A `// note:` points to the extractor's ordinal loop.
- Ten `TestCobolReconcile_External*` tests run on the memory and SQLite stores, plus a real-path SQLite persistence test.

## Task Commits

1. **Task 1: EXTERNAL_UNRESOLVED findings inside reconcileCobolFindings**
   - `556cdf57` (test, RED: 9 of 10 failed; `NotForBoundAmbiguousDynamicBuiltin` is a negative control)
   - `284c7869` (feat, GREEN)
2. **Task 2: External findings persist with the full contract on the real path**
   - `142bc6db` (test). It passes against Task 1 code. It fails with "has 0" when `cobol_findings.go` is temporarily reverted to the pre-Task-1 version, then restored.

**Plan metadata:** the docs(05-08) commits that add this file and update STATE/ROADMAP.

## Files Created/Modified

- `internal/resolver/cobol_findings.go`:
  - constants `cobolExternalKind` and the edge-evidence Meta keys;
  - `reconcileCobolFindings`, rewritten for external findings, with an extended doc block covering the Pattern 3 contract, eviction before the sweep, and D-03 container separation;
  - the private builder `newCobolExternalFinding`.
- `internal/resolver/cobol_findings_test.go`: the ten `TestCobolReconcile_External*` tests and the helpers `cobolCallMeta`, `cobolAt`, `cobolExternalNodes`, `cobolExternalID`, `cobolFindingTargets`, `mapKeys`.
- `internal/resolver/cobol_reconcile_test.go`: `cobolWriteCounter` now also counts `EvictFindingNodesByIDs` (in `reset` and `writes`).
- `cmd/gortex/cobol_findings_test.go`: `TestCobolFindings_ExternalPersisted`.

## Decisions Made

- **Evidence.** The evidence is `cobol_call` (else `cobol_copy_form`), then `cobol_copy_idms_kind` when present, then `cobol_target_raw` (else `cobol_copy_member_raw`), each whitespace-collapsed and joined with NUL. A literal CALL, LINK, and XCTL all carry `cobol_call=literal`. When they name the same target from one container, they get the same base ID with `#2`/`#3`.
- **QualName.** QualName includes the `#k` suffix, matching the extractor's parse findings.
- **Unsupported store.** The pass logs a warning when the store cannot evict findings. It never falls back to point deletes.
- **Eviction set.** The eviction set comes only from `::@finding/external/` IDs on the reconcile set's own placeholders, and the evicter is kind-guarded. A parse finding, another repository's finding, or a non-finding node can therefore never be evicted (T-05-41).

## Deviations from Plan

None. The plan executed as written. No pre-existing assertion in `cobol_references_test.go`, `cobol_reconcile_test.go`, the breadth tests, or the lifecycle tests needed a count adjustment for the new finding nodes or edges. Of the planned files, only `cobol_reconcile_test.go` (write counter) and the two finding test files changed.

## Issues Encountered

- `internal/resolver/csharp_verbatim_type_ref_test.go` is still not gofmt-clean at HEAD. This is unrelated and already in `deferred-items.md`.

## Verification

- **Task 1 verify:** passed.
  - All ten tests are listed.
  - `go test -race ./internal/resolver/ -run Cobol` is ok.
  - gofmt is clean.
  - Acceptance greps: EvictFindingNodesByIDs/FindingSymbol 2; Meta constants 7; first `EvictFindingNodesByIDs(` at line 235, before `r.graph.EvictFile(` at line 302; `nodesByKindLang(graph.KindFinding` 0; `GetInEdgesByNodeIDs(` 1.
  - The 11 `TestCobolReconcile_Placeholder*` tests pass.
- **Task 2 verify:** passed.
  - The test is listed.
  - The cmd `TestCobolFindings_*` and `TestCobolBreadth_(CallsAndCopy|CopybookProcedure|NestedProgramScope)` tests pass under race.
  - `TestCobolLifecycle_(CrossRepoClean|CrossFile|ConfigChange)` pass under race.
  - gofmt is clean.
- **Plan verification:**
  - `go build ./...` ok.
  - `go test -race ./internal/resolver/` (whole package) ok.
  - `go test -race -timeout 30m ./internal/indexer/ ./cmd/gortex/ -run '^TestCobol'` ok (181 s and 46 s).
  - `go test -race ./internal/graph/... -run 'PrefixDiagnostics|Finding'` ok.
  - `./internal/mcp/ -run 'Cobol|COBOL|Finding'` ok.
  - `git diff --name-only c598e32d..HEAD` lists only the 4 planned files.

## User Setup Required

None. No external service configuration is required. On the next binding pass, existing stores gain external findings for references already on placeholders.

## Next Phase Readiness

- 05-09 (ledger) can read `finding_kind`, `finding_container`, `finding_evidence_digest`, `prov_start_byte`, and the parser stamps on every external finding.
- 05-11 must sync FTS for these resolver-created findings and placeholders on incremental tails (Pitfall 4).

## Self-Check: PASSED

- FOUND: internal/resolver/cobol_findings.go, internal/resolver/cobol_findings_test.go, internal/resolver/cobol_reconcile_test.go, cmd/gortex/cobol_findings_test.go
- FOUND commits: 556cdf57, 284c7869, 142bc6db
- Both task verify commands, every acceptance criterion, and the plan-level verification passed.

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
