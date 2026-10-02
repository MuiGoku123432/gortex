---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 05
subsystem: parser
tags: [cobol, tree-sitter, data-items, level-numbers, redefines, renames, filler, copybook]

requires:
  - phase: 03-03
    provides: "Data-division section nodes (<prog>/@<kind>_section), relationEdge with statement provenance, span-placed COPY"
provides:
  - "Every data_description with a D-06 level in a named program's data-division section as a node: 01/77 KindVariable, 02-49/66/88/FILLER KindField, cobol_kind=data_item, full group-path ID (D-07)"
  - "Raw clause Meta: cobol_level, cobol_name_raw, cobol_pic, cobol_usage, cobol_occurs_min/max/depending, cobol_value, cobol_redefines, cobol_renames"
  - "cobolDataIndex (per document) with lookup(owner, name, qualifiers) and the shared cobolDataReference path that 03-07, 03-08, and 03-09 reuse"
  - "REDEFINES/RENAMES references edges with cobol_ref_role and honest unresolved::cobol::data:: targets"
  - "cobol_contains_copy on the innermost open item or section node; cobol_unleveled_entry_count on sections"
  - "cobolDataSection(w, idx, owner, idPrefix, parentID, section) entry point for the D-16 copybook plan"
affects: [03-07, 03-08, 03-09, 03-13, phase-04-identity]

actuals:
  tokens: 6135
  tasks: 2
  commits: 4
plan_head_before: f7a228217e4e0400849c63df184796059971e624

tech-stack:
  added: []
  patterns:
    - "Level-number stack walk over flat data_description observations (01/77 reset, 02-49 pop, 66 to the 01 record, 88 to the preceding non-88 entry)"
    - "Reserved-symbol set: program symbols are taken first, so a colliding 01 record takes the next free #k"
    - "Section COPY span runs past the section's own span to the next non-COPY observation"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_data.go
    - internal/parser/languages/cobol_grammar_data_clauses.go
    - internal/parser/languages/cobol_grammar_data_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_walk.go
    - internal/parser/languages/cobol_grammar_procedure.go

key-decisions:
  - "A level outside 01-49, 66, 77, 88 (such as 78 or 50) and a CONSTANT entry (its level sits under constant_entry) get no node and are counted in cobol_unleveled_entry_count. Nothing is placed by guess."
  - "A COPY after a section's last entry lies outside the section's generated span, so the section's COPY window runs to the first later observation that is not a COPY. Otherwise the most common trailing COPY would be silent."
  - "Item #k ordinals skip any symbol already taken by a program, not only the first program symbol. This keeps the record ID distinct even when nested programs carry their own #k."
  - "cobolWalk gains a sections map (filled by cobolProcedure), so the data walker reuses the exact section node and its #k instead of re-deriving the symbol."
  - "Edges carry cobol_name_norm=cobol-name-norm/1 because binding depends on the matching rule (D-18). PERFORM edges do the same."

patterns-established:
  - "Data names resolve only through cobolDataReference(w, idx, from, occurrence, owner, name, qualifiers, role)"
  - "Qualifiers match as an inner-to-outer subsequence of the item's non-filler ancestor keys"

requirements-completed: [DET-01]

coverage:
  - id: D1
    description: "Data-item nodes with the D-06 level hierarchy, D-07 full-path IDs (FILLER#k, #k duplicates, nested-program collision), kinds, member_of/defines, raw clause Meta, exact ranges, and deterministic order"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_data_test.go#TestCobolGrammar_DataItems"
        status: pass
    human_judgment: false
  - id: D2
    description: "REDEFINES/RENAMES references edges with cobol_ref_role and full provenance, case-insensitive owner-local binding, IN/OF qualification, not_declared_in_program and ambiguous unresolved targets with candidates"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_data_test.go#TestCobolGrammar_DataReferences"
        status: pass
    human_judgment: false
  - id: D3
    description: "COPY inside a group, before any record, and after a section's last entry sets cobol_contains_copy on the open item or the section node, and later entries still nest by level"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_data_test.go#TestCobolGrammar_DataContainsCopy"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolTracer_'"
        status: pass
    human_judgment: false

duration: 14min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 05: COBOL Data Items, Level Hierarchy, and REDEFINES/RENAMES Summary

**Every data description in a named program's data division is now a node under its full group-path ID. 01/77 items are KindVariable and nested, 66, 88, and FILLER items are KindField. Raw PIC/USAGE/OCCURS/VALUE Meta comes along. REDEFINES and RENAMES bind through one owner-local, cobol-name-norm/1 data-reference path that resolves IN/OF qualifiers and records honest unresolved::cobol::data:: outcomes.**

## Performance

- **Duration:** about 14 min
- **Started:** 2026-10-02T07:09:17Z
- **Completed:** 2026-10-02T07:23:13Z
- **Tasks:** 2
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments
- `cobolDataItems(w)` walks each named program's data-division sections in observation order and returns the document's `cobolDataIndex`. `Extract` calls it right after `cobolProcedure(w)`.
- `cobolDataSection` nests `data_description` entries by numeric level: `5` equals `05`. 01 and 77 reset the stack, and 77 is standalone. 02-49 pop while the top level is >= the new one. 66 attaches to the current 01 record, and 88 to the preceding non-88 entry. IDs are `<prog>/<01>/<child>/…`. Fillers, including implicit ones with no entry name (named `FILLER`), are `FILLER#k` among siblings. Any other repeated path gets `#k`, with k >= 2.
- Each item gets the document provenance keys, an exact range, a file `defines` edge, and `member_of` to its parent item or section. Its raw clause Meta is the PIC string and USAGE value (without their keywords), the OCCURS bounds (one integer sets both) and DEPENDING ON name, each VALUE item, and the REDEFINES/RENAMES operands.
- `qualifiedName`, `cobolDataIndex.lookup`, and `cobolDataReference` form the single data-name path. Exactly one owner-local, non-filler match binds. Zero matches go to `unresolved::cobol::data::<owner>::<key>[/<qualifier>…]` with `not_declared_in_program`. Several matches go to the same target with `ambiguous` and sorted `cobol_candidate_symbols`.
- REDEFINES (`redefines`) and RENAMES (`renames_from`, `renames_thru`) edges are emitted after every item is registered, using the clause observation as the occurrence.
- A COPY in a data section sets `cobol_contains_copy` on the stack top, or on the section node when the stack is empty. Entries that cannot be placed are counted in `cobol_unleveled_entry_count`.

## Task Commits

1. **Task 1: Data-item nodes, level hierarchy, full-path IDs, and raw clause Meta** - `afccfd50` (test, RED), `12799eb8` (feat, GREEN)
2. **Task 2: REDEFINES/RENAMES relationships, the shared data-reference helper, and COPY-in-group marking** - `233baee0` (test, RED), `14f7372b` (feat, GREEN)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_data.go` - `cobolDataItem`, `cobolDataIndex` (`lookup`), `cobolDataItems`, walk method `qualifiedName`, `cobolDataReference`, `cobolDataFrame`, `cobolDataSection` (300 lines)
- `internal/parser/languages/cobol_grammar_data_clauses.go` - `cobolDataClauses`, the raw clause Meta reader
- `internal/parser/languages/cobol_grammar_data_test.go` - invented fixtures; `TestCobolGrammar_DataItems`, `TestCobolGrammar_DataReferences`, `TestCobolGrammar_DataContainsCopy`
- `internal/parser/languages/cobol_grammar.go` - Extract keeps the walk and calls `cobolDataItems(w)` after `cobolProcedure(w)`. The version stays at `gortex-cobol-grammar/3`.
- `internal/parser/languages/cobol_grammar_walk.go` - `sections map[int]*graph.Node` field (298 lines)
- `internal/parser/languages/cobol_grammar_procedure.go` - records each data or environment section node in `w.sections`

## TDD Evidence
- **Task 1 RED:** `GOWORK=off go test -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_DataItems$'` exited 1. The target test failed on its first assertion: the expected 12 data-item IDs against `[]string(nil)`. **GREEN:** `12799eb8`. The Task 1 verify command passes under `-race`.
- **Task 2 RED:** `-run '^TestCobolGrammar_Data(References|ContainsCopy)$'` exited 1. `TestCobolGrammar_DataReferences` failed on its first target assertion, which expects one REDEFINES edge `WS-ALT -> WS-REC` and found 0. `TestCobolGrammar_DataContainsCopy` passed at RED, because the marking had already shipped in Task 1 (see Deviations). **GREEN:** `14f7372b`. The Task 2 verify command passes: `-list`, `-race` on `^TestCobol`, `go vet`, and gofmt.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so these RED records were captured by hand, as in 03-01 through 03-04.

## Decisions Made
- Levels outside D-06 (78, 50 and so on) and CONSTANT entries are counted rather than placed. The count reuses the contract key `cobol_unleveled_entry_count`.
- The section's COPY window runs past the section's own generated span, up to the next non-COPY observation. A probe showed `01 WS-REC.` / `COPY X.` at the end of WORKING-STORAGE parents the COPY to the data division, or even to the program before PROCEDURE DIVISION.
- `#k` ordinals skip symbols already reserved by programs, so the ID stays unique when nested programs carry their own `#k`.
- The walk records section nodes in a map instead of re-deriving `#k` section symbols.
- Data-reference edges carry `cobol_name_norm`, matching the PERFORM edges.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] COPY after a section's last entry would be silent**
- **Found during:** Task 1 (fixture probe)
- **Issue:** The plan's rule only counts a COPY whose generated start falls inside the section's span. The grammar ends the section span at its last entry, so a trailing COPY parents to the division or the program and would mark nothing. That breaks the must-have "the missing copybook-supplied items are never silent".
- **Fix:** The window runs from the section start to the start of the first later observation that is not a COPY.
- **Files modified:** internal/parser/languages/cobol_grammar_data.go
- **Verification:** `TestCobolGrammar_DataContainsCopy` (trailing COPY in WORKING-STORAGE marks `WS-T/WS-TA`; before PROCEDURE DIVISION it marks `LK-REC`)
- **Committed in:** 12799eb8

**2. [Rule 3 - Blocking] Section node registry on the walk**
- **Found during:** Task 1
- **Issue:** `cobolDataItems` needs each data section's node. Its symbol can carry a `#k` that only `cobolProcedure` knows.
- **Fix:** Added a `sections map[int]*graph.Node` field (walk file at 298/300 lines). `cobolProcedure` fills it.
- **Files modified:** internal/parser/languages/cobol_grammar_walk.go, internal/parser/languages/cobol_grammar_procedure.go
- **Committed in:** 12799eb8

**3. [Rule 3 - Blocking] cobolDataClauses split into cobol_grammar_data_clauses.go**
- **Found during:** Task 2
- **Issue:** With the reference path added, `cobol_grammar_data.go` reached 314 lines. The acceptance criteria cap it at 300 and pin `lookup` and `cobolDataReference` to that file.
- **Fix:** Moved the self-contained clause-Meta reader to its own file and tightened the code. The file is now at exactly 300 lines.
- **Files modified:** internal/parser/languages/cobol_grammar_data.go, internal/parser/languages/cobol_grammar_data_clauses.go
- **Committed in:** 14f7372b

**4. [Sequencing] COPY-in-group marking landed in Task 1**
- **Found during:** Task 1
- **Issue:** COPY marking is part of the same merged entry walk as the level stack, so it went in with Task 1's GREEN commit instead of Task 2's.
- **Effect:** `TestCobolGrammar_DataContainsCopy` is a characterization test that passed at Task 2's RED. That gate rests on `TestCobolGrammar_DataReferences`, which failed for the right reason.
- **Committed in:** 12799eb8

**5. [Minor] Extract discards the returned index for now**
- **Issue:** The plan writes `data := cobolDataItems(w)`. An unused variable does not compile in Go.
- **Fix:** Extract calls `cobolDataItems(w)` with a comment. 03-07 captures the return when it first passes the index on. The acceptance grep still matches.
- **Committed in:** 12799eb8

---

**Total deviations:** 5 (1 missing-critical, 2 blocking, 1 sequencing, 1 minor)
**Impact on plan:** One extra file and one walk field. No behavior beyond the plan except the trailing-COPY window, which the must-haves require.

## TDD Gate Compliance
- Task 1: RED `afccfd50` (`test(03-05)`) precedes GREEN `12799eb8` (`feat(03-05)`).
- Task 2: RED `233baee0` (`test(03-05)`) precedes GREEN `14f7372b` (`feat(03-05)`). The ContainsCopy test was green at RED (deviation 4).
- No refactor commit was needed.

## Issues Encountered
- In the fork grammar, `88` parses as an ordinary `level_number`, because the `level_number_88` alternative is commented out. Implicit fillers have no `entry_name` child, and lower-case `filler` is kept as written. All of these are handled and covered.
- A malformed data entry (an invented `05 LK-B PIC X OF LK-G.` in a probe) set off error recovery that parents later entries to the program, outside every section. Those entries get no node. This is a producer-side limit, not a role gap.

## Known Stubs
None.

## Threat Flags
None. T-03-16: name segments pass through `cobolIDSegment`, and FILLER and duplicate ordinals follow source order. Entry names match `[A-Za-z0-9-]`, so they cannot collide with `@` segments. T-03-17: no Meta key contains a Neo4j secret-filter substring (password, secret, token, credential, privatekey, apikey, authorization), no error message carries clause text, and fixtures are invented. T-03-18: binding requires exactly one owner-local match; zero or several go to an unresolved data target with a reason and candidates.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 03-07, 03-08, and 03-09 should capture `data := cobolDataItems(w)` in Extract and resolve every data name through `cobolDataReference(w, data, from, occurrence, owner, name, qualifiers, role)`. `owner` is `w.filePath + "::" + prog`, and `w.qualifiedName(i)` splits a `qualified_word`.
- 03-13 can call `cobolDataSection` with the copybook as owner. The REDEFINES/RENAMES pass in `cobolDataItems` would then need to run for that owner too.
- Existing SQLite stores need a reindex to pick up data items. The extractor version stays at `/3`, as the phase contract requires.

## Self-Check: PASSED
- FOUND: internal/parser/languages/cobol_grammar_data.go, cobol_grammar_data_clauses.go, cobol_grammar_data_test.go, cobol_grammar.go, cobol_grammar_walk.go, cobol_grammar_procedure.go
- FOUND commits: afccfd50, 12799eb8, 233baee0, 14f7372b
- Both task verify commands pass under -race. `go build ./...` passes. The COBOL tracer acceptance tests (`./cmd/gortex -run '^TestCobolTracer_'`) and the resolver COBOL tests pass. gofmt is clean on the touched files. `cobol_grammar_data.go` is 300 lines.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
