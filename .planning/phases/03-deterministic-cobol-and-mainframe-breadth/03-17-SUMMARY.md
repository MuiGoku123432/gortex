---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 17
subsystem: cobol-extractor
tags: [cobol, copybook, idms, cics, db2, exec-sql, wr-04, gap-closure, tdd]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-08 IDMS/CICS walkers; 03-09 SQL walker; 03-15 cobolCopybookRoot and w.id; 03-16 procedure copybook structure, statements, and flow"
provides:
  - "cobolMainframe and cobolSQL build From and owner IDs through w.id; cobolCopybook runs them after cobolReferences"
  - "IDMS, CICS, and DB2 edges from procedure copybooks, From the paragraph or the copybook file node; dynamic placeholders keyed by the copybook path"
  - "cobol_grammar_data_refs.go: cobolDataBind, qualifiedName, lookup, cobolDataReference (moved verbatim)"
  - "cobol_unresolved_reason=not_declared_in_copybook for data names a copybook references but does not declare"
  - "Copybook file node keys cobol_copybook_wrapper (data|procedure) and prov_analysis_absence=copybook_content_unmapped (non-green and nothing mapped)"
  - "docs/languages.md COBOL row: nested scope, copybook COPY family, procedure copybooks, no COPY expansion, WR-04 window"
affects: [phase-04-incremental-cobol, phase-05-parse-findings, neo4j-projection, verify-work-03]

actuals:
  tokens: 7872
  tasks: 3
  commits: 7
plan_head_before: 2c6e3d7aa40cbbfefb0330ec1cdefd8de22e318f

tech-stack:
  added: []
  patterns:
    - "Walker owner IDs go through w.id everywhere (procedure, statements, flow, references, mainframe, SQL), so one rule names the owner in program and copybook documents"
    - "Producer-handoff facts that Gortex cannot fix are recorded on the file node (cobol_copybook_wrapper) and a degraded empty mapping is stamped as an explicit absence, never presented as an empty member"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_data_refs.go
  modified:
    - internal/parser/languages/cobol_grammar_mainframe.go
    - internal/parser/languages/cobol_grammar_sql.go
    - internal/parser/languages/cobol_grammar_copybook.go
    - internal/parser/languages/cobol_grammar_data.go
    - internal/parser/languages/cobol_grammar_copybook_test.go
    - cmd/gortex/cobol_breadth_test.go
    - docs/languages.md

key-decisions:
  - "A copybook data name with zero matches is not_declared_in_copybook (keyed on w.root); program documents keep not_declared_in_program, and ambiguous is unchanged"
  - "WR-04 stays a producer window: cobol_copybook_wrapper is read from the wrapper program's direct data_division/procedure_division observation, and copybook_content_unmapped requires both a non-green grade and only the file node with no edge"
  - "The data-name resolution move was committed as its own refactor (verbatim, diff-checked) before the reason change, so the behavior change is isolated"

patterns-established:
  - "Copybook absence stamp: a non-green copybook analysis that mapped nothing carries prov_analysis_absence=copybook_content_unmapped; a partial mapping (a leading COPY's imports edge) shows its loss only through grade, reasons, and wrapper"

requirements-completed: [DET-01, DET-04, DET-07, DET-08, DET-09, DET-10, DET-11, DET-12]

coverage:
  - id: D1
    description: "Through the real MultiIndexer, SQLite store, and resolver, copy/valpara.cpy's EXEC SQL persists one reads edge from @proc/VALIDATE-PARA to db2::MYSCH/ORDERS (sql_verb SELECT, statement symbol @proc/VALIDATE-PARA/@exec_sql_statement, ast_resolved, row 4); repository A's table set is unchanged"
    requirement: DET-11
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_CopybookProcedureReferences"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_MainframeAndSQL"
        status: pass
    human_judgment: false
  - id: D2
    description: "IDMS OBTAIN, CICS LINK and RETURN TRANSID(data-name), EXEC SQL SELECT with alias, and PREPARE in a procedure copybook emit the program-equivalent edges From the paragraph; root-level IDMS and SQL flow From the copybook file node; no KindTable for O, S1, WS-SQLTXT, or SQLCA; every node and edge passes assertCobolMetaContract"
    requirement: DET-07
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookProcedureMainframe"
        status: pass
    human_judgment: false
  - id: D3
    description: "CICS, PREPARE, dynamic CALL, and REDEFINES data names a copybook does not declare carry not_declared_in_copybook; a declared REDEFINES target binds with no reason; program documents keep not_declared_in_program"
    requirement: DET-04
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookDataNames"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookProcedureCalls"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_data_test.go + cobol_grammar_refs_test.go (not_declared_in_program assertions)"
        status: pass
    human_judgment: false
  - id: D4
    description: "WR-04 window recorded: six members (EXEC CICS, FD, leading COPY, pseudo-text, data, procedure) carry the producer's wrapper; copybook_content_unmapped only on degraded members that mapped nothing; TestCobolTracer_CopybookDataItems still sees no absence on a green data copybook"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookWrapperWindow"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookReplacingDegrades"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_CopybookDataItems"
        status: pass
    human_judgment: false
  - id: D5
    description: "docs/languages.md COBOL row states nested CALL scope, copybook COPY-family imports, procedure copybook ownership with label_not_in_copybook / not_declared_in_copybook, no COPY expansion, and the WR-04 window; the Windows sentence is kept"
    verification:
      - kind: other
        ref: "grep '^| COBOL |' docs/languages.md for the six required terms (Task 3 verify)"
        status: pass
    human_judgment: true
    rationale: "Wording quality and the absence of estate content in public docs need a human read"
  - id: D6
    description: "Phase 3 gate: GOWORK=off go test -race -timeout=45m ./... (156 packages ok), GOWORK=off go build ./cmd/gortex, Windows dependency closure has no fork preprocessor package, go.mod/go.sum unchanged"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -timeout=45m ./..."
        status: pass
      - kind: other
        ref: "GOOS=windows CGO_ENABLED=1 GOWORK=off go list -deps ./cmd/gortex/ | grep tree-sitter-cobol-upgrade/preprocessor (no match)"
        status: pass
    human_judgment: false

duration: 26min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 17: Copybook Mainframe Edges, Honest Data Names, and the WR-04 Window Summary

**IDMS, CICS, and DB2 statements in procedure copybooks now produce the same verb-classified edges as in programs. Each edge comes From the paragraph, or from the copybook file node when the statement precedes any header, and the real path persists `copy/valpara.cpy::@proc/VALIDATE-PARA -> db2::MYSCH/ORDERS`. A data name a copybook does not declare now carries `not_declared_in_copybook`. The producer's misclassified wrapper (review WR-04) is now visible: the file node records `cobol_copybook_wrapper`, and a degraded member that mapped nothing gets `copybook_content_unmapped`. The full Phase 3 gate is green.**

## Performance

- **Duration:** 26 min (about 15 of it the full race suite)
- **Started:** 2026-10-12T15:33:53Z
- **Completed:** 2026-10-12T16:00:45Z
- **Tasks:** 3
- **Files modified:** 8 (1 created)

## Accomplishments

- `cobolMainframe` and `cobolSQL` now compute From, the CICS program ID, and the SQL program ID through `w.id`. `cobolCopybook` calls both after `cobolReferences`, the same order `Extract` uses. Program IDs are byte-identical, because `root` is empty for programs.
- In copybooks, CICS and PREPARE dynamic placeholders and data references are keyed under the copybook path (for example `unresolved::cobol::dynamic::transaction::copy/mf.cpy::WS-TRN`). Cursors are scoped the same way.
- Data-name resolution now lives in `cobol_grammar_data_refs.go`. The move was verbatim, checked with `diff`, and `cobol_grammar_data.go` went from 312 lines to 212. `cobolDataReference` returns `not_declared_in_copybook` when the walk has a copybook root.
- `cobolCopybook` writes `cobol_copybook_wrapper` from the wrapper program's direct division observation. It writes `prov_analysis_absence=copybook_content_unmapped` only when the grade is not green and the result holds just the file node with no edge.
- The docs COBOL row now covers nested scope, copybook COPY-family imports, procedure-copybook ownership and its two unresolved reasons, that COPY is not expanded, and the WR-04 window.
- Phase gate: the full repository race suite, the GOWORK=off build, the Windows closure check, and the go.mod/go.sum check all pass.

## Task Commits

1. **Task 1 RED: failing real-path test for copybook EXEC SQL edges** - `67864e0b` (test)
2. **Task 1 GREEN: emit IDMS, CICS, and DB2 edges from procedure copybooks** - `f798a4e7` (feat)
3. **Task 2 move: data-name resolution into its own file** - `93454327` (refactor)
4. **Task 2 RED: failing tests for copybook mainframe edges and data names** - `d569e7c2` (test)
5. **Task 2 GREEN: not_declared_in_copybook** - `8156827a` (feat)
6. **Task 3 RED: failing tests for the WR-04 wrapper window** - `0f910d0d` (test)
7. **Task 3 GREEN: wrapper and absence stamps, docs row** - `265a1ef6` (feat)

## TDD Evidence

- **Task 1 RED:** `TestCobolBreadth_CopybookProcedureReferences` failed on its target assertion, `reads edges a/copy/valpara.cpy::@proc/VALIDATE-PARA -> a/db2::MYSCH/ORDERS`, with 0 edges where 1 was expected. I converted the `go test -json` output to TAP, and `check tdd-red-evidence` returned `RED_EVIDENCE_OK / target_test_failed`.
- **Task 1 GREEN and tracer gate:** the run was interactive, in `end-of-phase` mode, and the verify was automated-only. I re-ran both verify commands after the commit and both passed, so the plan continued to its next task.
- **Task 2 RED:** `TestCobolGrammar_CopybookDataNames` (target) and the tightened `ProcedureCalls` failed only on the reason assertion: expected `not_declared_in_copybook`, actual `not_declared_in_program`. The checker returned `RED_EVIDENCE_OK`. `TestCobolGrammar_CopybookProcedureMainframe` passed at RED because Task 1 had already shipped that behavior, so it is a lock test.
- **Task 3 RED:** `TestCobolGrammar_CopybookWrapperWindow` (target) and `ReplacingDegrades` failed only on the two new keys. The grade, node-count, and edge-count assertions already held, which confirms the producer behavior the plan describes. The checker returned `RED_EVIDENCE_OK`.
- **REFACTOR:** the data-name move is a refactor commit, but it ran before the Task 2 RED tests. Its purpose was to keep the reason change in a clean diff. The `TestCobol(Grammar_|Extractor_)` suite passed before and after the move.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar_mainframe.go`: From and CICS program ID go through `w.id`; doc comment updated.
- `internal/parser/languages/cobol_grammar_sql.go`: program ID and From go through `w.id`; doc comment updated. Still 297 lines.
- `internal/parser/languages/cobol_grammar_copybook.go`: calls `cobolMainframe` and `cobolSQL`, writes the wrapper and absence stamps, and documents the WR-04 window. Imports the pinned `preprocessor/grade`. 86 lines.
- `internal/parser/languages/cobol_grammar_data_refs.go` (new): the moved data-name resolution plus `not_declared_in_copybook`.
- `internal/parser/languages/cobol_grammar_data.go`: the moved code is removed (212 lines).
- `internal/parser/languages/cobol_grammar_copybook_test.go`: `ProcedureMainframe` (subtests `member` and `root_sql`), `DataNames` (subtests `procedure_member` and `data_member`), `WrapperWindow` (6 subtests), the tightened `ProcedureCalls`, and the absence stamp in `ReplacingDegrades`. The blanket check in `assertCobolCopybookFile` was replaced by explicit no-absence assertions in `DataItems`, `NestedCopy`, `IncludeForms/data_member`, and `ProcedureFragment`.
- `cmd/gortex/cobol_breadth_test.go`: `TestCobolBreadth_CopybookProcedureReferences`.
- `docs/languages.md`: the COBOL row.

## Decisions Made

- `not_declared_in_copybook` is chosen on `w.root != ""`, which is set only in the copybook branch. `ambiguous` keeps its meaning in both modes.
- The wrapper value is the producer's observed division. Gortex never infers what the wrapper should have been. When neither division was observed, the key stays unset.
- The absence condition counts every node and edge, including shared resource and table nodes. Anything mapped therefore suppresses the stamp, which is why the leading-COPY member (one imports edge) carries none.

## Deviations from Plan

- **Root-level IDMS assertion (orchestrator advisory, additive):** the `root_sql` fixture also contains `OBTAIN CALC EMP-REC.` before any header. The test asserts a reads edge From the copybook file node with `idms_verb` OBTAIN. That covers the must-have truth that IDMS statements before any header flow from the file node, which the plan's SQL-only fixture did not reach.
- **Refactor commit:** Task 2's verbatim move is its own `refactor(03-17)` commit, ahead of the RED tests, not part of the GREEN commit. Behavior is unchanged and the suite was green on both sides of it.
- **Tooling:** the Gortex MCP tools were not mounted, so I read source with shell commands, as in 03-14 to 03-16.

**Total deviations:** 0 auto-fixed (1 additive test assertion, 2 process notes).
**Impact on plan:** none. Scope is unchanged.

## Issues Encountered

None.

## Known Limitations (owned by later phases)

- WR-04 itself is not fixed. A member opening with `EXEC CICS`, a non-declarative `EXEC SQL`, `FD`/`SD`, a leading `COPY`, or `-INC` can still get the wrong wrapper. If such a member mapped something (for example a leading COPY), its loss shows only through its grade, its reasons, and `cobol_copybook_wrapper`. Phase 5 parse findings (`PARSE_UNRESOLVED`) can localize it. A real fix needs a separate, user-confirmed producer change.
- COPY is still not expanded into including programs, so a copybook's unresolved data names and labels never bind to an including program. Cross-member binding is deferred to Phase 4.
- Copybook paragraphs are still `KindFunction`, so the deferred incremental-reindex hazard (review CR-01, Phase 4) applies to them.

## Threat Model Check

- T-03-55: copybook SQL goes through the unchanged `cobolSQL` rules. `ProcedureMainframe` asserts that the only table is `db2::MYSCH/ORDERS` and that no node is named O, S1, WS-SQLTXT, or SQLCA. The real path keeps repository A's table set exact.
- T-03-56: the absence stamp requires a non-green grade and a result with only the file node and no edge. The leading-COPY, data, and procedure subtests show it is not stamped otherwise.
- T-03-57: there was no fork edit, commit, push, or re-pin, and the gate confirmed `go.mod` and `go.sum` are unchanged.
- T-03-58: `assertCobolMetaContract` runs over every node and edge of the mainframe fixture. The new keys (`cobol_copybook_wrapper`, `prov_analysis_absence`) contain no secret-like substring.
- No new threat surface. Fixtures and docs are invented and contain no estate content.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- This was the last plan of Phase 3, which is ready for `/gsd-verify-work 03`. The full repository race suite is green.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-12*

## Self-Check: PASSED

- FOUND: cobol_grammar_data_refs.go, cobol_grammar_copybook.go, cobol_breadth_test.go, docs/languages.md, and this SUMMARY
- FOUND: all seven commits (67864e0b, f798a4e7, 93454327, d569e7c2, 8156827a, 0f910d0d, 265a1ef6)
