---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 03
subsystem: parser
tags: [cobol, tree-sitter-cobol-upgrade, handoff, statements, sections, provenance, KindLocal]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth (03-01)
    provides: cobolWalk (children, last, owner, container, flowFrom), symbolNode/defines/memberOf/relationEdge builders, cobolProcedure, cobolDedupeEdges
provides:
  - Data and environment section nodes (KindPackage, cobol_kind=section, cobol_section_kind) with a file defines edge and member_of their division
  - cobolStatementKinds and cobolStatements in cobol_grammar_statements.go: KindLocal statement nodes <container>/@<kind>[#k], member_of only
  - Statement registry on the walk, statementSymbol(i) and statementNode(i)
  - relationEdge stamps cobol_statement_symbol (a QualName); same-line duplicates list cobol_statement_symbols
  - copy_statement placed by generated span (Pitfall 3)
  - cobolDedupeEdges moved to cobol_grammar_dedupe.go
affects: [03-05, 03-06, 03-07, 03-08, 03-09, 03-10, 03-13]

actuals:
  tokens: 5657
  tasks: 2
  commits: 4
plan_head_before: c694d507fb2eb45c22668333493c8957adfc6de4

tech-stack:
  added: []
  patterns:
    - "Statement nodes are KindLocal with one member_of edge to the header container (paragraph, else section, else procedure division) and no defines edge"
    - "Statement IDs use fixed @<observation kind> segments with #k ordinals in source order, so no source text enters them"
    - "cobolStatements runs inside cobolProcedure before any relationship edge, so every relationEdge can look up its statement"
    - "w.container/w.flowFrom are also set on the procedure division observation itself, which gives span-placed COPY a fallback container"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_statements.go
    - internal/parser/languages/cobol_grammar_dedupe.go
    - internal/parser/languages/cobol_grammar_structure_test.go
  modified:
    - internal/parser/languages/cobol_grammar_procedure.go
    - internal/parser/languages/cobol_grammar_walk.go

key-decisions:
  - "cobolDedupeEdges moved unchanged into cobol_grammar_dedupe.go so cobol_grammar_walk.go stays within 300 lines (294) after the statement registry was added"
  - "The procedure division observation itself carries container=division node and flowFrom=program, so a span-placed COPY with no earlier division child falls back to the division"
  - "copy_statement is always placed by generated span, even when its Parent chain is inside a procedure division; span wins over ancestry"
  - "A COPY after the last statement of a procedure division lies outside the division's generated span and gets no statement node (the plan's outside-every-division rule)"
  - "prov_extractor_version stays gortex-cobol-grammar/3, as the phase contract in 03-01 requires"

patterns-established:
  - "Pattern: later walkers read the innermost statement through w.statementSymbol and annotate statement Meta only through w.statementNode"

requirements-completed: [DET-01]

coverage:
  - id: D1
    description: "CONFIGURATION, WORKING-STORAGE, and LINKAGE sections are KindPackage nodes <prog>/@<kind> with cobol_section_kind, document keys, own ranges, a nil-Meta file defines edge, and a nil-Meta member_of to their division; same-named sections in nested programs stay distinct"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_structure_test.go#TestCobolGrammar_Structure"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every named statement in a named program's procedure division is a KindLocal node with Name and cobol_statement_kind equal to its observation kind, document keys, own range, one member_of edge to its paragraph, section, or division, no defines edge, and no statement text; #k ordinals in source order; header-free, empty-paragraph, nested-program, and empty-source cases"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_structure_test.go#TestCobolGrammar_Statements"
        status: pass
    human_judgment: false
  - id: D3
    description: "Relationship edges carry cobol_statement_symbol (a QualName, so FilePath + '::' + symbol is the statement node ID); same-line duplicate PERFORMs collapse with both symbols in cobol_statement_symbols; COPY is placed by generated span and statementNode returns its node"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_structure_test.go#TestCobolGrammar_StatementSymbolOnEdges"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolTracer_'"
        status: pass
    human_judgment: false

duration: 11min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 03: COBOL Sections, Statement Nodes, and Statement Provenance on Edges Summary

**Data and environment sections become KindPackage containers, every named procedure statement becomes a KindLocal node attached only by member_of (`<container>/@<kind>[#k]`), and every relationship edge built through relationEdge now carries `cobol_statement_symbol` pointing back to its statement, with grammar-extra COPY statements placed by generated span**

## Performance

- **Duration:** about 11 min
- **Started:** 2026-10-02T06:28:59Z
- **Completed:** 2026-10-02T06:39:45Z
- **Tasks:** 2
- **Files modified:** 5 (3 created, 2 modified)

## Accomplishments
- `cobolProcedure` emits every `*_section` child of a named program's non-procedure division as `<prog>/@<kind>` (KindPackage, `cobol_kind=section`, `cobol_section_kind` such as `working_storage`), with a file `defines` edge and `member_of` its division.
- New `cobol_grammar_statements.go`: `cobolStatementKinds` lists the 44 members of the pinned fork grammar's `_statement` choice (the `ERROR` alias excluded) plus `perform_statement_loop` and `copy_statement`. `cobolStatements` emits one KindLocal node per statement, ordered by `Generated.Start` (stable on index), with `#k` for repeated kinds in one container. Each node has `cobol_statement_kind`, the cloned document keys, its own range, and one `member_of` edge. It has no `defines` edge and stores no statement text.
- The walk now has a statement registry: `statementSymbol(i)` returns the QualName of the innermost registered statement among i and its ancestors, and `statementNode(i)` returns the node or nil. Both are bounds-checked. `relationEdge` stamps `cobol_statement_symbol`, and the existing dedupe already merges same-line duplicates into `cobol_statement_symbols`.
- `copy_statement` is placed by generated span. It goes into the procedure division whose span contains its start, under the container of that division's last other child that starts at or before it. Its subtree's `container`/`flowFrom` are recorded the same way as for tree-contained observations.
- `cobolDedupeEdges` moved unchanged to `cobol_grammar_dedupe.go`, so `cobol_grammar_walk.go` is 294 lines.

## Task Commits

1. **Task 1: Data and environment section containers and all named statement nodes** - `d5789ace` (test, RED), `7a89515a` (feat, GREEN)
2. **Task 2: Statement registry, cobol_statement_symbol on relationship edges, and span-placed COPY statements** - `b9bfbb68` (test, RED, with stub signatures), `37f128ac` (feat, GREEN)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_statements.go` - `cobolStatementKinds`, `cobolStatements` (COPY span placement, ordering, `#k`, node emission, registration)
- `internal/parser/languages/cobol_grammar_dedupe.go` - `cobolDedupeEdges`, moved verbatim from the walk file
- `internal/parser/languages/cobol_grammar_structure_test.go` - invented fixtures; `TestCobolGrammar_Structure`, `TestCobolGrammar_Statements`, `TestCobolGrammar_StatementSymbolOnEdges`
- `internal/parser/languages/cobol_grammar_procedure.go` - section emission, division-level container/flow, `cobolStatements(w)` before PERFORM edges
- `internal/parser/languages/cobol_grammar_walk.go` - `statement` registry field, `statementSymbol`, `statementNode`, `cobol_statement_symbol` in `relationEdge`

## TDD Evidence
- **Task 1 RED:** `GOWORK=off go test -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_(Structure|Statements)$'` exited 1. Both target tests failed on their own `no node` assertions, for `src/sect.cbl::SECTPGM/@configuration_section` and `src/stmt.cbl::STMTPGM/@procedure_division/@display_statement`. **GREEN:** `7a89515a`. The Task 1 verify command passes under `-race`.
- **Task 2 RED:** the test first failed to compile because `statementNode`/`statementSymbol` did not exist. That is INVALID_RED, so the RED commit adds stub signatures returning nil/"". With the stubs, `TestCobolGrammar_StatementSymbolOnEdges` failed on its first target assertion (expected `SYMPGM/@proc/P1/@perform_statement_call_proc`, actual `""`). **GREEN:** `37f128ac`. The Task 2 verify command (languages and cmd/gortex tracer, both under `-race`) passes.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so these RED records were captured by hand, as in 03-01 and 03-02.

## Decisions Made
- The dedupe function moved to its own file. It is a post-pass over the result's edges, not a walk method, so it is a real seam. The plan's grep criteria still hold, because `relationEdge` and `statementNode` stay in the walk file.
- The procedure division observation now carries its own container (the division node) and flow source (the program). This gives span placement a fallback without recomputing division symbols, and `cobolStatements` uses it to find named programs' procedure divisions.
- COPY placement uses span even when ancestry would agree. In the fixtures, ancestry and span give the same answer for every COPY inside a division. Span is the rule the plan specifies.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Split cobolDedupeEdges into cobol_grammar_dedupe.go**
- **Found during:** Task 2
- **Issue:** `cobol_grammar_walk.go` was already 297 lines. The registry plus edge stamping would have taken it past the 300-line acceptance limit.
- **Fix:** Moved `cobolDedupeEdges` verbatim into the new `cobol_grammar_dedupe.go` (`//go:build darwin || linux`). The orchestrator suggested this seam. There is no behavior change.
- **Files modified:** internal/parser/languages/cobol_grammar_walk.go, internal/parser/languages/cobol_grammar_dedupe.go
- **Verification:** `TestCobolGrammar_EdgeDedupe` and all `TestCobol*` pass under -race. walk.go is 294 lines.
- **Committed in:** 37f128ac

**2. [Rule 3 - Blocking] Stub signatures in the Task 2 RED commit**
- **Found during:** Task 2
- **Issue:** Without the methods, the test did not compile. Under tdd.md, a build failure is INVALID_RED.
- **Fix:** The RED commit carries `statementSymbol`/`statementNode` stubs that return zero values. GREEN replaces them.
- **Files modified:** internal/parser/languages/cobol_grammar_walk.go
- **Committed in:** b9bfbb68

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** One extra file (`cobol_grammar_dedupe.go`) beyond the four planned. No behavior change and no scope creep.

## TDD Gate Compliance
- Task 1: RED `d5789ace` (`test(03-03)`) precedes GREEN `7a89515a` (`feat(03-03)`).
- Task 2: RED `b9bfbb68` (`test(03-03)`) precedes GREEN `37f128ac` (`feat(03-03)`).
- No refactor commit was needed.

## Issues Encountered
- The grammar treats a COPY after the last statement of a procedure division as outside the division span (its Parent is `start` or the program). Under the plan's rule it gets no statement node. 03-07, which handles copybook includes, should not expect a statement for that position.

## Known Stubs
None. The Task 2 RED stubs were replaced in GREEN.

## Threat Flags
None. T-03-09: statement Meta holds identities, kinds, names, and coordinates only, and the tests assert that no Meta string contains fixture statement text. T-03-10: statements are KindLocal with only a member_of edge. T-03-11: IDs use fixed `@<kind>` segments with deterministic `#k`. Fixtures are invented.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 03-05 (data items) can attach to `<prog>/@working_storage_section` and the other section nodes.
- 03-06 through 03-09 get `cobol_statement_symbol` automatically on every edge built through `relationEdge`, and annotate statements through `w.statementNode`.
- Existing SQLite stores need a reindex to pick up section and statement nodes. The extractor version stays at `/3`, as the phase contract requires.

## Self-Check: PASSED
- FOUND: internal/parser/languages/cobol_grammar_statements.go, cobol_grammar_dedupe.go, cobol_grammar_structure_test.go, cobol_grammar_procedure.go, cobol_grammar_walk.go
- FOUND commits: d5789ace, 7a89515a, b9bfbb68, 37f128ac
- Both task verify commands pass under -race. gofmt is clean on the touched files. `go build ./...` passes, as does the full `internal/parser/languages` package.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
