---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 15
subsystem: cobol-extractor
tags: [cobol, copybook, copy, imports, gap-closure, tdd]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-07 cobolCopy COPY vocabulary and From rule; 03-13 copybook analysis branch and breadth real-path harness; 03-14 latest extractor state"
provides:
  - "cobolCopybook (cobol_grammar_copybook.go): copybook-mode mapping moved out of Extract, now running cobolReferences"
  - "cobolCopybookRoot = \"@proc\": the wrapper program's root symbol, which the copybook file node stands in for"
  - "cobolWalk.root and (*cobolWalk).id: From and owner IDs map the walk root to the file node ID"
  - "COPY, -INC, COPY IDMS, and EXEC SQL INCLUDE inside a copybook emit imports edges from the copybook file node"
  - "Copybook-to-copybook COPY binds on the persisted real-path graph under the unchanged exactly-one rule"
affects: [03-16, 03-17, phase-04-incremental-cobol, neo4j-projection]

actuals:
  tokens: 7188
  tasks: 2
  commits: 3
plan_head_before: 57a0d6056e60d806235c7a13a5ace4b1bb4603cb

tech-stack:
  added: []
  patterns:
    - "A copybook walk registers the producer's wrapper program_definition as its root; cobolWalk.id maps that root to the file node ID, so no edge or owner ID names the wrapper"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_copybook.go
    - internal/parser/languages/cobol_grammar_copybook_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_walk.go
    - internal/parser/languages/cobol_grammar_refs.go
    - internal/parser/languages/cobol_grammar_test.go
    - cmd/gortex/cobol_breadth_test.go

key-decisions:
  - "The copybook wrapper is registered as the walk root (cobolCopybookRoot \"@proc\") instead of special-casing copybook mode in cobolCopy; cobolWalk.id maps it to the file node, so cobolReferences and cobolCopy run unchanged in both modes"
  - "No new Meta key, reason, or binding path: copybook COPY edges reuse the 03-07 vocabulary and the unchanged exactly-one same-repository copybook index; extractor version stays gortex-cobol-grammar/3"

patterns-established:
  - "Edge From and owner IDs in COBOL walkers go through w.id(symbol), so a walk root can stand in for a synthetic container"

requirements-completed: [DET-05, DET-06]

coverage:
  - id: D1
    description: "A COPY inside a copybook persists as one imports edge from the copybook file node, bound to the copied copybook under the unchanged exactly-one rule, with form, member, range, and observation Meta; ORDPGM's COPY CUSTREC still binds"
    requirement: DET-06
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_CopybookNestedCopy"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_CallsAndCopy"
        status: pass
    human_judgment: false
  - id: D2
    description: "Nested COPY in a copybook: exactly two imports edges from the file node (ADDRREC row 2, TAILREC row 4) with member as written, norm rule, every range key, Line = start row + 1, and none of cobol_kind, cobol_name_key, cobol_contains_copy; COPY marks unchanged"
    requirement: DET-05
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookNestedCopy"
        status: pass
    human_judgment: false
  - id: D3
    description: "-INC (inexact range with real coordinates), COPY IDMS (idms_dictionary_member, never bound), EXEC SQL INCLUDE SQLCA (builtin), and COPY OF library inside a copybook each emit an imports edge from the file node; a COPY on the member's first code line does too"
    requirement: DET-05
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookIncludeForms"
        status: pass
    human_judgment: false
  - id: D4
    description: "The wrapper program is never a node or edge endpoint; copybook data items unchanged (exactly three, no wrapper node)"
    requirement: DET-05
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookDataItems"
        status: pass
    human_judgment: false

duration: 7min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 15: COPY Edges Inside Copybooks Summary

**COPY, -INC, COPY IDMS, and EXEC SQL INCLUDE written inside a copybook now become imports edges from the copybook file node. The edges keep the member name as written and the statement's original range. A copybook-to-copybook COPY binds on the persisted graph under the unchanged exactly-one rule.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-10-12T14:45:18Z
- **Completed:** 2026-10-12T14:52:20Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- `cobolCopybook` in the new `cobol_grammar_copybook.go` takes over `Extract`'s copybook branch. It now calls `cobolReferences`, so `cobolCopy` emits the COPY family from the file node (D-11, D-19). This closes VERIFICATION gap 2.
- `cobolWalk.root` and `(*cobolWalk).id` were added. The producer's wrapper `program_definition` is registered as the root `@proc`, and `w.id` maps it to the file node ID. The CALL and COPY From and owner IDs now go through `w.id`. Program documents keep identical IDs because their root is empty.
- The copybook tests moved unchanged to `cobol_grammar_copybook_test.go`. `TestCobolGrammar_CopybookNestedCopy` now asserts both imports edges, and the new `TestCobolGrammar_CopybookIncludeForms` covers every COPY form plus a COPY on the member's first code line.
- `TestCobolBreadth_CopybookNestedCopy` proves the binding through MultiIndexer, SQLite, and the production resolver: `copy/custall.cpy` → `copy/custrec.cpy`, with no `cobol_unresolved_reason`.
- `cobol_grammar.go` is back to 285 lines, down from 302. `cobol_grammar_walk.go` is at 294.

## Task Commits

1. **Task 1 RED: failing real-path test** - `1c7776bf` (test)
2. **Task 1 GREEN: emit COPY imports edges from a copybook's file node** - `e03af2c0` (feat)
3. **Task 2: move copybook tests, assert every COPY form** - `548bccf7` (test)

## TDD Evidence

- **Task 1 RED:** `TestCobolBreadth_CopybookNestedCopy` failed on its target assertion ("imports edges a/copy/custall.cpy -> a/copy/custrec.cpy" had 0 items). I converted the `go test -json` output to TAP and ran it through `gsd-tools check tdd-red-evidence`, which returned `RED_EVIDENCE_OK / target_test_failed`.
- **Task 1 GREEN:** the same test passes under `-race`, and so do all `TestCobol(Tracer|Breadth)_`, `TestCobol(Grammar_|Extractor_)`, and resolver `TestCobol` tests.
- **Task 2:** these tests lock behavior that Task 1 already shipped, so they passed on their first run. To show they can fail, I commented out the `cobolReferences(w, idx)` call in `cobolCopybook` and reran them. `NestedCopy` (0 of 2 edges), `IncludeForms/data_member` (0 of 4), and `IncludeForms/leading_copy_member` (0 of 1) all failed on their planned assertions, and the checker returned `RED_EVIDENCE_OK` for both test names. I then restored the file with `git checkout -- <file>` and confirmed it had no diff before committing the tests.
- No REFACTOR commit was needed.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar_copybook.go` (new): `cobolCopybookRoot` and `cobolCopybook`.
- `internal/parser/languages/cobol_grammar.go`: the copybook branch is now a single `cobolCopybook` call.
- `internal/parser/languages/cobol_grammar_walk.go`: the `root` field and the `id` method.
- `internal/parser/languages/cobol_grammar_refs.go`: CALL From/program IDs and COPY From go through `w.id`.
- `internal/parser/languages/cobol_grammar_copybook_test.go` (new): the moved copybook tests, the extended `NestedCopy` test, `IncludeForms`, and the `cobolCopybookImports` helper.
- `internal/parser/languages/cobol_grammar_test.go`: the copybook block removed. `cobolCopybookFixture`, `cobolWrongGrammarID`, and `cobolGrammarHex64` stay here.
- `cmd/gortex/cobol_breadth_test.go`: the `cobolBreadthCustAllFixture` fixture (`copy/custall.cpy`) and `TestCobolBreadth_CopybookNestedCopy`.

## Decisions Made

- Copybook handling uses the walk root instead of a copybook-specific branch in `cobolCopy`. One `w.id` rule names the owner in both modes, and Plan 03-16 can reuse `cobolCopybookRoot` as the `@proc` prefix for copybook paragraph IDs.
- No new vocabulary: the edges reuse the 03-07 COPY Meta, and binding stays in the unchanged `resolveCobolReferences` exactly-one same-repository index. COPY IDMS keeps its never-bound reason (D-19).

## Deviations from Plan

None in implementation: the plan was executed as written.

- **Process note (TDD order, Task 2):** Task 2's tests could not be committed failing ahead of an implementation, because Task 1 had already shipped the behavior they lock. The mutation check described under TDD Evidence stands in for the RED gate.
- **Process note (tooling):** the Gortex MCP tools were not mounted in this session, so I read source with shell commands, as 03-14 did.

**Total deviations:** 0 auto-fixed.
**Impact on plan:** none.

## Issues Encountered

- `gofmt -l internal/parser/languages/` lists about 20 unrelated files that were already unformatted before this plan. All files this plan touched are gofmt-clean. These are out of scope and were not changed.

## Threat Model Check

- T-03-47: copybook-to-copybook edges bind only through the unchanged exactly-one index. COPY IDMS keeps `idms_dictionary_member` (asserted in `IncludeForms`).
- T-03-48: keys still pass through `cobolIDSegment(cobolNameNorm(...))` in the unchanged `cobolCopy`. The raw member is kept only in Meta.
- T-03-49: the walk clones docMeta before the binding keys are written. `NestedCopy` asserts the edges carry no `cobol_kind`, `cobol_name_key`, or `cobol_contains_copy`.
- T-03-50: the wrapper exists only as the walk root and maps to the file node. `CopybookDataItems` still asserts that no wrapper node exists.
- No new threat surface beyond the plan's threat model.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- Plans 03-16 and 03-17 can build procedure copybooks on `cobolCopybookRoot` and `w.id`. `TestCobolGrammar_CopybookProcedureFragment` now holds only the file-node assertions, ready for 03-16 to add procedure-structure assertions.
- Known deferred hazard (review CR-01, Phase 4): re-indexing a copybook drops inbound COPY edges until the files that include it are re-indexed. This now also covers copybook-to-copybook edges.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-12*

## Self-Check: PASSED

- FOUND: internal/parser/languages/cobol_grammar_copybook.go
- FOUND: internal/parser/languages/cobol_grammar_copybook_test.go
- FOUND: 1c7776bf, e03af2c0, 548bccf7
- Verification: both Task 1 verify commands and the Task 2 verify command passed. Full `internal/parser/languages` and `internal/resolver` packages also pass.
