---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 16
subsystem: cobol-extractor
tags: [cobol, copybook, procedure, paragraph, perform, gap-closure, tdd]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-06 procedure/flow walkers; 03-13 copybook analysis branch and breadth real-path harness; 03-15 cobolCopybookRoot, cobolWalk.root, and w.id"
provides:
  - "cobolProcedure copybook-root mode: only the wrapper's procedure division is read, no division node; the copybook file node is the procedure container"
  - "Copybook procedure IDs <path>::@proc/<SECT>, <path>::@proc/<PARA>, <path>::@proc/<SECT>/<PARA>, statements <container>/@<kind>[#k] and <path>::@proc/@<kind>[#k]"
  - "Member-local PERFORM / THRU / GO TO in copybooks; unresolved labels go to unresolved::cobol::label::<path>::<key> with cobol_unresolved_reason=label_not_in_copybook"
  - "CALL and COPY inside procedure copybooks flow From the paragraph; get_callers reaches copybook paragraphs over the daemon MCP relay"
affects: [03-17, phase-04-incremental-cobol, phase-04-identity, neo4j-projection]

actuals:
  tokens: 6266
  tasks: 2
  commits: 3
plan_head_before: 1eec479f2a74d1a8c4f02b63ca592ef1ff5c1fce

tech-stack:
  added: []
  patterns:
    - "Root mode in cobolProcedure: when the program symbol is w.root, the file node (w.filePath) replaces the division node as container, and label bases are <root>/<NAME>; finish skips member_of when the parent is the file node"
    - "Flow From and unresolved-label prefixes go through w.id, so one rule names the owner in program and copybook documents"

key-files:
  created: []
  modified:
    - internal/parser/languages/cobol_grammar_procedure.go
    - internal/parser/languages/cobol_grammar_statements.go
    - internal/parser/languages/cobol_grammar_flow.go
    - internal/parser/languages/cobol_grammar_copybook.go
    - internal/parser/languages/cobol_grammar_copybook_test.go
    - cmd/gortex/cobol_breadth_test.go

key-decisions:
  - "Copybook procedure symbols start with the copybook root @proc (<path>::@proc/<PARA>), mirroring a program's <prog>/@proc/<PARA>; '@' cannot begin a COBOL word, so they never collide with copybook data-item IDs"
  - "A label a copybook does not define is label_not_in_copybook (only label_not_found is rewritten; ambiguous and thru_range_invalid keep their reasons), and it is never bound to a paragraph of another file"
  - "Top-level copybook sections and paragraphs carry only the file defines edge, like copybook 01/77 items; no wrapper program or division node is emitted"

patterns-established:
  - "cobolProcedure root mode: a walk root that is a synthetic wrapper contributes only its procedure division, owned by the file node"

requirements-completed: [DET-02, DET-03, DET-05]

coverage:
  - id: D1
    description: "A procedure copybook persists its paragraphs (kind function, cobol_ordinal, member rows 0 and 5, exact range, copybook grade policy) and statements; PERFORM CHECK-PARA binds member-locally; PERFORM MAIN-PARA stays unresolved with label_not_in_copybook although ORDPGM defines MAIN-PARA; CALL 'SUBPGM' binds; MCP get_callers on CHECK-PARA and SUBPGM returns VALIDATE-PARA; no program/division node or CPYWRAP ID"
    requirement: DET-02
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_CopybookProcedure"
        status: pass
    human_judgment: false
  - id: D2
    description: "Procedure fragment member: exactly three nodes (file, paragraph, move statement); defines from the file; paragraph has no member_of; statement member_of the paragraph; full document provenance on every node"
    requirement: DET-05
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookProcedureFragment"
        status: pass
    human_judgment: false
  - id: D3
    description: "Statements before any header are <path>::@proc/@<kind>, member_of the copybook file node; their PERFORM flows From the file node"
    requirement: DET-05
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookProcedureRoot"
        status: pass
    human_judgment: false
  - id: D4
    description: "Copybook section and paragraphs, PERFORM THRU expansion, GO TO, and label_not_in_copybook for an unknown label and a THRU range end; no calls edge leaves the member"
    requirement: DET-03
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookProcedureFlow"
        status: pass
    human_judgment: false
  - id: D5
    description: "Literal CALL, dynamic CALL (plus call_target data reference keyed under the copybook path), and COPY inside a copybook paragraph come From the paragraph"
    requirement: DET-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookProcedureCalls"
        status: pass
    human_judgment: false
  - id: D6
    description: "Program documents unchanged: every TestCobol* test in internal/parser/languages, internal/resolver, and cmd/gortex passes; full languages and resolver packages pass under -race"
    verification:
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/parser/languages/ ./internal/resolver/"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol'"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 16: Procedure Copybooks First-Class Summary

**Procedure-only copybooks now emit their sections, paragraphs, statements, and PERFORM/THRU/GO TO/CALL/COPY edges. The copybook file node owns them at the member's own coordinates, and their IDs have the form `<path>::@proc/<PARA>`. Labels resolve only within the member, and a label the member does not define gets the reason `label_not_in_copybook`. The wrapper program never appears in the graph, and `get_callers` now reaches copybook paragraphs through SQLite and the daemon MCP relay.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-10-12T15:08:42Z
- **Completed:** 2026-10-12T15:17:44Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- `cobolProcedure` has a copybook-root mode. When the program is `w.root`, only the wrapper's `procedure_division` is read and no division node is emitted. The container ID is the file path, and labels are based at `@proc/<NAME>`. `finish` skips the member_of edge when the parent is the file node, so top-level copybook sections and paragraphs have only the defines edge.
- `cobolStatements` builds member_of through `w.id(w.container[i])`, so a statement before any header is member_of the copybook file node.
- `cobolPerform` and `cobolGoTo` compute From and the unresolved-label prefix through `w.id`. Program IDs are unchanged. `cobolFlowEdge` rewrites `label_not_found` to `label_not_in_copybook` when the walk has a root.
- `cobolCopybook` calls `cobolProcedure(w)` before the data sections, so statements are registered before `cobolReferences` builds CALL and COPY edges. A data member is unaffected because its wrapper has no procedure division.
- The real-path test `TestCobolBreadth_CopybookProcedure` covers the new `copy/valpara.cpy` fixture in repository A.
- Four unit tests lock the structure, root-level statements, flow, and calls. They share a helper, `assertCobolNoWrapper`.

## Task Commits

1. **Task 1 RED: failing real-path test for a procedure copybook** - `d0318ca0` (test)
2. **Task 1 GREEN: emit procedure copybook paragraphs, statements, and flow** - `46b221b1` (feat)
3. **Task 2: lock copybook procedure structure, flow, and calls** - `b427ba6e` (test)

## TDD Evidence

- **Task 1 RED:** `TestCobolBreadth_CopybookProcedure` failed on its first target assertion: `missing a/copy/valpara.cpy::@proc/VALIDATE-PARA`. I converted the `go test -json` output to TAP, and `gsd-tools check tdd-red-evidence` returned `RED_EVIDENCE_OK / target_test_failed`.
- **Task 1 GREEN:** the test passes under `-race`. All `TestCobol(Tracer|Breadth)_`, `TestCobol(Grammar_|Extractor_)`, and resolver `TestCobol` tests also pass.
- **Tracer gate:** the run was interactive in `end-of-phase` mode and the verify was automated-only, so I re-ran the tracer's `<verify>`. It passed, and I continued to Task 2.
- **Task 2 (mutation check):** Task 1 had already shipped the behavior these tests lock, so they passed on their first run. To show they can fail, I temporarily disabled the `label_not_in_copybook` rewrite in `cobolFlowEdge` (`&& false`). `TestCobolGrammar_CopybookProcedureFlow` then failed on both label assertions (expected `label_not_in_copybook`, actual `label_not_found`), and the checker returned `RED_EVIDENCE_OK`. I restored the file with `git checkout -- internal/parser/languages/cobol_grammar_flow.go`, confirmed `git diff --stat` showed no change to it, and committed the tests.
- No REFACTOR commit was needed.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar_procedure.go`: root mode (`divID`, `divSymbol`, `labelBase`), the `finish` member_of skip, and an updated doc comment.
- `internal/parser/languages/cobol_grammar_statements.go`: member_of goes through `w.id`.
- `internal/parser/languages/cobol_grammar_flow.go`: From and label placeholders go through `w.id`, and the code writes `label_not_in_copybook`. The file is 294 lines, within the 300-line limit.
- `internal/parser/languages/cobol_grammar_copybook.go`: `cobolProcedure(w)` call and doc comment.
- `internal/parser/languages/cobol_grammar_copybook_test.go`: the rewritten `ProcedureFragment` test, the new `ProcedureRoot`, `ProcedureFlow`, and `ProcedureCalls` tests, and `assertCobolNoWrapper`.
- `cmd/gortex/cobol_breadth_test.go`: `cobolBreadthValParaFixture` and `TestCobolBreadth_CopybookProcedure`.

## Decisions Made

- Copybook procedure IDs are rooted at `@proc`, following the plan's ID decision. This reuses `cobolCopybookRoot` from 03-15, so `w.id` maps every root-level owner to the file node.
- Only `label_not_found` is rewritten in copybooks. `ambiguous` and `thru_range_invalid` keep their meaning, because they describe the member's own headers.
- The wrapper's name (CPYWRAP, from the pinned producer's `transform/copybook.go`) is used only in test assertions as the string that must never appear. Production code never names it.

## Deviations from Plan

None in implementation: the plan was executed as written.

- **Process note (TDD order, Task 2):** the Task 2 tests could not be committed failing before an implementation existed, because Task 1 had already shipped the behavior. The mutation check under TDD Evidence stands in for the RED gate.
- **Process note (tooling):** the Gortex MCP tools were not mounted in this session, so I read source with shell commands, as in 03-14 and 03-15.

**Total deviations:** 0 auto-fixed.
**Impact on plan:** none.

## Issues Encountered

None.

## Known Limitations (flagged in plan, owned by 03-17 / Phase 4)

- A dynamic CALL's data reference in a copybook still carries `not_declared_in_program`. Plan 03-17 introduces `not_declared_in_copybook`. The unit test asserts only that the reason is non-empty.
- EXEC SQL, EXEC CICS, and IDMS statements in copybooks get statement nodes only. Plan 03-17 adds the edges they would produce. The valpara fixture's EXEC SQL line is already in place for that plan.
- COPY is still not expanded into including programs. A program's PERFORM of a paragraph that only a copybook defines stays `label_not_found`, and cross-member labels are deferred to Phase 4.
- Copybook paragraphs are `KindFunction`, so the deferred incremental-reindex hazard (review CR-01, Phase 4) applies to them as it does to program paragraphs.

## Threat Model Check

- T-03-51: labels resolve only over the member's headers, and anything else goes to `unresolved::cobol::label::<path>::…` with `label_not_in_copybook`. Both `ProcedureFlow` and the real-path MAIN-PARA check assert this.
- T-03-52: no program or division node exists, and no ID contains CPYWRAP. `assertCobolNoWrapper` checks this in all four unit tests, and the real-path test scans the persisted copybook nodes.
- T-03-53: the tests pin member rows 0 and 5 with `prov_range_exact` true. The fragment paragraph's extent ends at row 1.
- T-03-54: header names still pass through `cobolIDSegment`.
- T-03-SC: `go.mod` and `go.sum` are unchanged, no package was installed, and the fork was not touched.
- No new threat surface.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- Plan 03-17 can add `not_declared_in_copybook` and run the IDMS, CICS, and SQL walkers on copybooks. Their statement nodes and `w.id` routing are already in place.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-12*

## Self-Check: PASSED

- FOUND: cmd/gortex/cobol_breadth_test.go, internal/parser/languages/cobol_grammar_copybook_test.go, and the four modified walker files
- FOUND: d0318ca0, 46b221b1, b427ba6e
- Verification: both Task 1 verify commands and the Task 2 verify command passed. The full `internal/parser/languages` and `internal/resolver` packages pass under `-race`, and `go.mod` and `go.sum` are unchanged.
