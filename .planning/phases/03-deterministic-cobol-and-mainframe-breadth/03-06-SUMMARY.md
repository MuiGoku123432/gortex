---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 06
subsystem: parser
tags: [cobol, tree-sitter-cobol-upgrade, control-flow, perform, thru, goto, depending-on, calls]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth (03-01)
    provides: cobolWalk, relationEdge, cobolProcedure headers with cobol_ordinal, the tracer's single-label PERFORM edge
  - phase: 03-deterministic-cobol-and-mainframe-breadth (03-03)
    provides: statement registry (statementSymbol, statementNode), cobol_statement_symbol on every relationEdge, cobolDedupeEdges
provides:
  - cobol_grammar_flow.go with cobolFlow(w), called by cobolProcedure after cobolStatements
  - One program-local label resolver (qualified X OF/IN S, own section first, else unique in program) shared by PERFORM, THRU, and GO TO
  - PERFORM THRU expansion (cobol_flow=perform_thru, cobol_thru_start, cobol_thru_end) or one unresolved ::thru:: target with label_not_found, ambiguous, or thru_range_invalid
  - GO TO edges (cobol_flow=goto), DEPENDING ON metadata (cobol_goto_depending, cobol_depending_positions, cobol_goto_depending_on), and cobol_goto_alterable on alterable GO TO statements
affects: [03-07, 03-08, 03-13, 06-QUERY-02]

actuals:
  tokens: 6141
  tasks: 2
  commits: 4
plan_head_before: c69617b47c9758908545fb7d08671c512f45b690

tech-stack:
  added: []
  patterns:
    - "Flow walkers rebuild the per-division label table from header observations: w.container of a header is its own node symbol"
    - "A statement's own section comes from w.flowFrom: a section label, else the paragraph label's enclosing section"
    - "GO TO targets are grouped in the extractor by target ID, so a repeated DEPENDING ON target is one edge whatever the backend"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_flow.go
    - internal/parser/languages/cobol_grammar_flow_test.go
  modified:
    - internal/parser/languages/cobol_grammar_procedure.go

key-decisions:
  - "cobolFlow(w) rebuilds the label table from header observations instead of a new cobolWalk field: cobol_grammar_walk.go is at 298/300 lines and the plan touches only three files"
  - "An unresolved THRU edge reports the first failing endpoint: start is checked before end, so its reason and candidates win"
  - "Own section is tracked by label index, not section name, so two same-named sections never share own-section lookups"
  - "A qualified label never falls back to a program-wide match: X OF S is label_not_found unless S holds a paragraph X"
  - "cobol_thru_start and cobol_thru_end hold the label text as generated (for example A OF S1), not the normalized key"

patterns-established:
  - "Pattern: every control-flow edge goes through cobolFlowEdge, which uses w.relationEdge and stamps cobol_flow and cobol_name_norm"

requirements-completed: [DET-02]

coverage:
  - id: D1
    description: "Unqualified PERFORM labels resolve to the paragraph in the statement's own section first, else the unique section or paragraph in the program; X OF S and X IN S resolve only inside section S; PERFORM <section> and numeric labels resolve; inner programs never resolve outer labels"
    requirement: DET-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_flow_test.go#TestCobolGrammar_ControlFlow/{own_section_first,qualified,section_target,numeric_labels,nested_isolation}"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_procedure_test.go#TestCobolGrammar_PerformEdge (unmodified)"
        status: pass
    human_judgment: false
  - id: D2
    description: "PERFORM A THRU C expands to one perform_thru edge per same-kind procedure from A to C in source order, crossing section boundaries, all sharing Line, statement symbol, and the thru endpoints as written; invalid, mixed-kind, or unresolved ranges emit exactly one unresolved ::thru:: edge and no partial expansion"
    requirement: DET-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_flow_test.go#TestCobolGrammar_ControlFlow/{thru_paragraphs,thru_sections,thru_invalid}"
        status: pass
    human_judgment: false
  - id: D3
    description: "GO TO emits one goto edge per distinct target; DEPENDING ON adds cobol_goto_depending, 1-based cobol_depending_positions, and cobol_goto_depending_on; unresolved targets carry label_not_found; an alterable GO TO emits no edge and marks its statement cobol_goto_alterable; no fall-through edge exists"
    requirement: DET-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_flow_test.go#TestCobolGrammar_ControlFlow/{goto,goto_depending,goto_unresolved,goto_alterable,no_fallthrough}"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolTracer_'"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 06: COBOL Control Flow (Qualified Labels, PERFORM THRU, GO TO) Summary

**Program-local COBOL label resolution (X OF/IN S, own section first, then unique in program) now drives PERFORM, exact-or-nothing PERFORM THRU expansion, and GO TO / GO TO DEPENDING ON calls edges, with unresolved label targets that carry label_not_found, ambiguous, or thru_range_invalid**

## Performance

- **Duration:** about 9 min
- **Started:** 2026-10-02T07:25:54Z
- **Completed:** 2026-10-02T07:34:48Z
- **Tasks:** 2
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments
- New `cobol_grammar_flow.go` holds `cobolFlow(w)`. `cobolProcedure` now calls it after `cobolStatements(w)`, and the tracer's inline `cobolPerform` and `cobolLabel` are gone from `cobol_grammar_procedure.go` (226 to 148 lines).
- `cobolResolve` is the one resolver for all three statement kinds:
  - `X OF S` / `X IN S` (second WORD or second number of the label) matches only paragraph X in a section S of the same program.
  - An unqualified X matches the paragraph X in the statement's own section first. If there is none, it matches the unique section or paragraph X in the program.
  - Zero matches give `label_not_found`. Several give `ambiguous` with sorted `cobol_candidate_symbols`. Keys use cobol-name-norm/1.
- `PERFORM A THRU C` (THROUGH too) expands over the program's headers in source order. Only procedures of the endpoints' kind are included, and paragraph ranges cross section boundaries. Every edge gets `cobol_flow=perform_thru` and shares the statement's Line and `cobol_statement_symbol`. Each edge also carries `cobol_thru_start` and `cobol_thru_end` as written. A missing or ambiguous endpoint, mixed kinds, or C before A gives exactly one edge to `unresolved::cobol::label::<file>::<prog>::<start>::thru::<end>`.
- `GO TO` emits one `cobol_flow=goto` edge per distinct target, in order of first listing.
  - With DEPENDING ON, each edge also gets `cobol_goto_depending=true`, `cobol_depending_positions` ([]string, 1-based), and `cobol_goto_depending_on` (the identifier as written, for example `WS-IX`).
  - `GO TO.` emits no edge and sets `cobol_goto_alterable=true` on its statement node.
- No fall-through, ALTER, STOP RUN, GOBACK, or EXIT edge is emitted.

## Task Commits

1. **Task 1: Qualified and own-section label resolution, and PERFORM THRU expansion** - `4b979aad` (test, RED), `aced1907` (feat, GREEN)
2. **Task 2: GO TO, GO TO DEPENDING ON, and alterable GO TO** - `fdc23236` (test, RED), `b4b7ef08` (feat, GREEN)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_flow.go` (287 lines) - `cobolLabel`, `cobolLabelRef`, `cobolFlow`, `cobolProcedureLabels`, `cobolLabelOf`, `cobolResolve`, `cobolFlowEdge`, `cobolPerform`, `cobolGoTo`
- `internal/parser/languages/cobol_grammar_flow_test.go` - invented fixtures (OWNPGM, THRUPGM, GOTOPGM, plus the existing nested fixture); `TestCobolGrammar_ControlFlow` with 13 subtests
- `internal/parser/languages/cobol_grammar_procedure.go` (148 lines) - PERFORM emission and the label table removed; calls `cobolFlow(w)`

## TDD Evidence
- **Task 1 RED (`4b979aad`):** `GOWORK=off go test -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_ControlFlow$'` exited 1. `own_section_first` failed on its target assertion: it expected `src/own.cbl::OWNPGM/@proc/S1/P-X` and got the `ambiguous` label target. `qualified`, `thru_paragraphs`, `thru_sections`, and `thru_invalid` each found 0 edges where 1 to 3 were expected. `section_target`, `numeric_labels`, and `nested_isolation` already passed on the tracer's single-label code, which those behaviors pin. **GREEN (`aced1907`):** the Task 1 verify command passes under `-race`.
- **Task 2 RED (`fdc23236`):** `goto`, `goto_depending`, and `goto_unresolved` found 0 edges. `goto_alterable` failed on the missing `cobol_goto_alterable`. `no_fallthrough` already passed, since no GO TO edges existed yet. **GREEN (`b4b7ef08`):** the Task 2 verify command passes (languages and cmd/gortex tracer under `-race`, go vet, gofmt, and both files at or under 300 lines).
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output. As in 03-01 through 03-05, these RED records were captured by hand.

## Decisions Made
- **No new walk field.** `cobolFlow(w)` keeps the plan's one-argument signature. It rebuilds the label table from each procedure division's header observations, where `w.container[header]` is the header's own node symbol. The header-name rule (first field, minus `.`) now appears in two places, marked with a `// note:` comment. `cobol_grammar_walk.go` stays at 298 lines.
- **Own section by index.** The statement's own section is a label index found from `w.flowFrom`, not a section name. Two same-named sections therefore never share own-section lookups.
- **THRU failure reporting.** The start endpoint is resolved first. If it fails, its reason and candidates go on the single unresolved edge. Otherwise the end endpoint's do.
- **No fallback for qualified labels.** A qualified label never falls back to a program-wide match.

## Deviations from Plan

None - plan executed exactly as written.

## TDD Gate Compliance
- Task 1: RED `4b979aad` (`test(03-06)`) precedes GREEN `aced1907` (`feat(03-06)`).
- Task 2: RED `fdc23236` (`test(03-06)`) precedes GREEN `b4b7ef08` (`feat(03-06)`).
- No refactor commit was needed.

## Issues Encountered
- `gofmt -l internal/parser/languages/` lists about 20 files that were already unformatted before this plan, none of them COBOL files. They are out of scope and were left untouched. The three planned files are gofmt-clean.

## Known Stubs
None.

## Threat Flags
None.
- **T-03-19:** resolution is program-local. It binds only on exactly one match and records a reason otherwise. Invalid or unresolved THRU ranges emit one edge and never expand partially.
- **T-03-20:** every label-target key segment passes through `cobolIDSegment`, so label text cannot forge `::thru::`, `/`, or `#k` structure.
- **T-03-21 (accepted):** expansion is linear in the program's own header count.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 6 QUERY-02 can trace paragraph execution through `get_callers` and `get_call_chain` over `calls` edges, using `cobol_flow` values perform, perform_thru, and goto.
- Existing SQLite stores need a reindex to pick up the THRU and GO TO edges. The extractor version stays at `gortex-cobol-grammar/3`, as the phase contract requires.

## Self-Check: PASSED
- FOUND: internal/parser/languages/cobol_grammar_flow.go, cobol_grammar_flow_test.go, cobol_grammar_procedure.go
- FOUND commits: 4b979aad, aced1907, fdc23236, b4b7ef08
- Both task verify commands pass. The full `internal/parser/languages` package passes. `go build ./...` passes. `git diff --name-only` since plan start lists only the three planned files.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
