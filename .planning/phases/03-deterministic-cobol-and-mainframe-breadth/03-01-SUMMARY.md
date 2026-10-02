---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 01
subsystem: parser
tags: [cobol, tree-sitter-cobol-upgrade, handoff, provenance, sqlite, mcp, get_callers]

requires:
  - phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
    provides: CobolGrammarExtractor program node, prov_* document contract, cobolProgramSymbols IDs, cobol tracer harness
provides:
  - cobolWalk shared observation walk (children index, subtree ends, owning program, container and flow-source lookups)
  - cobolNameNorm / cobolNameNormRule (cobol-name-norm/1), cobolIDSegment, cobolStampMeta, cobolStampExtent, cobolDedupeEdges
  - symbolNode / defines / memberOf / relationEdge builders implementing the phase provenance contract
  - cobolProcedure: division nodes (KindPackage), procedure sections and paragraphs (KindFunction, cobol_ordinal, Pitfall 1 extents)
  - single-label PERFORM calls edges (cobol_flow=perform) or unresolved::cobol::label targets with label_not_found / ambiguous
  - prov_extractor_version gortex-cobol-grammar/3
affects: [03-03, 03-05, 03-06, 03-07, 03-08, 03-09, 03-10, 03-13]

actuals:
  tokens: 7725
  tasks: 2
  commits: 3
plan_head_before: e927b6a2963704eb7830e9681424c28959b6357a

tech-stack:
  added: []
  patterns:
    - "Every breadth walker reads one cobolWalk; IDs are <path>::<symbol>, name segments pass through cobolIDSegment"
    - "Structural edges (defines, member_of) carry nil Meta; relationship edges carry cloned docMeta plus prov_observation_kind, prov_affected, and occurrence range keys, Line = prov_start_row + 1"
    - "cobolDedupeEdges runs last in Extract so SQLite and the in-memory graph persist the same edge"
    - "w.container / w.flowFrom record, per observation inside a procedure division, its membership container and flow source symbols"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_walk.go
    - internal/parser/languages/cobol_grammar_procedure.go
    - internal/parser/languages/cobol_grammar_procedure_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_test.go
    - cmd/gortex/cobol_tracer_test.go

key-decisions:
  - "Breadth (newCobolWalk + cobolProcedure) runs before the document-flag block, so prov_containment_unresolved and prov_unnamed_program_count stay on the file node only"
  - "PERFORM edges record cobol_name_norm=cobol-name-norm/1 (D-18 rule ID in meta)"
  - "Paragraph and section extents end at the last procedure-division child before the ending header; that child's span covers its descendants"
  - "Division Name is the observation kind (for example procedure_division), matching its @<observation kind> ID segment"

patterns-established:
  - "Pattern: one occurrences map per document for #k across divisions, sections, and paragraphs with identical paths"
  - "Pattern: PERFORM targets are matched after every header of the program is known (forward references)"

requirements-completed: [DET-01, DET-02]

coverage:
  - id: D1
    description: "Paragraphs MAIN-PARA and WORK-PARA persist through MultiIndexer and SQLite AddBatch; MCP get_callers on WORK-PARA returns MAIN-PARA over the real daemon relay; the persisted calls edge carries cobol_flow=perform, the provenance contract, and Line = prov_start_row + 1"
    requirement: DET-02
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_ParagraphFlow"
        status: pass
    human_judgment: false
  - id: D2
    description: "Identification, environment, data, and procedure divisions emitted as KindPackage nodes with cobol_division, file defines and program member_of edges with nil Meta"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_procedure_test.go#TestCobolGrammar_Divisions"
        status: pass
    human_judgment: false
  - id: D3
    description: "Procedure sections and paragraphs follow the phase ID grammar (@proc/[SECT/]PARA, #k for duplicates), keep names as written, carry int cobol_ordinal, and stamp Pitfall 1 extents; nested programs keep their own @proc paths"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_procedure_test.go#TestCobolGrammar_Paragraphs"
        status: pass
    human_judgment: false
  - id: D4
    description: "Single-label PERFORM binds only to exactly one same-program label under cobol-name-norm/1; missing and ambiguous labels stay on unresolved::cobol::label targets with reasons and sorted candidate symbols; no cross-program binding"
    requirement: DET-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_procedure_test.go#TestCobolGrammar_PerformEdge"
        status: pass
    human_judgment: false
  - id: D5
    description: "Relationship edges sharing From, To, Kind, and Line collapse to one survivor with cobol_occurrence_count"
    requirement: DET-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_procedure_test.go#TestCobolGrammar_EdgeDedupe"
        status: pass
    human_judgment: false
  - id: D6
    description: "cobolNameNorm (quote strip, un-double, ASCII upper-fold) and cobolIDSegment percent-encoding of %, :, /, #, and control bytes"
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_procedure_test.go#TestCobolGrammar_NameNorm"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 01: Paragraph Tracer and Shared COBOL Walk Summary

**COBOL divisions, procedure sections, and paragraphs from handoff observations, with single-label PERFORM as calls edges that persist through SQLite and come back out of the unchanged MCP get_callers, on a shared walk that fixes the procedure ID grammar, the relationship-edge provenance contract, edge dedupe, and cobol-name-norm/1 (extractor version gortex-cobol-grammar/3)**

## Performance

- **Duration:** about 8 min of task execution (after context loading)
- **Started:** 2026-10-02T05:42:13Z
- **Completed:** 2026-10-02T05:49:40Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- A real indexed paragraph is traceable: `get_callers` on `…/cobol/demopgm.cbl::DEMOPGM/@proc/WORK-PARA` returns `…::DEMOPGM/@proc/MAIN-PARA` over the daemon relay, and the persisted SQLite edge carries `cobol_flow=perform`, `prov_observation_kind=perform_statement_call_proc`, `prov_extractor_version=gortex-cobol-grammar/3`, Origin `ast_resolved`, and `Line == prov_start_row + 1`.
- `cobol_grammar_walk.go` gives every later Phase 3 walker one children index, subtree ends, owning-program map, container/flow-source lookups, and the node and edge builders.
- Divisions (`KindPackage`), procedure sections and paragraphs (`KindFunction`) carry final IDs, `cobol_ordinal`, and Pitfall 1 extents; labels bind only on exactly one same-program match.
- `cobolDedupeEdges` makes edge Meta identical on SQLite (first wins) and the in-memory graph (last wins).

## Task Commits

1. **Task 1 (tracer): Paragraphs and one PERFORM from the handoff through SQLite to MCP get_callers** - `e5c9baff` (feat)
2. **Task 2: Division nodes, ordinals, extents, label outcomes, dedupe, and the name rules under unit test** - `ec7913de` (test, RED), `bbeb0ccd` (feat, GREEN)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_walk.go` - shared observation walk, name and ID rules, range stamping, edge builders, dedupe (297 lines)
- `internal/parser/languages/cobol_grammar_procedure.go` - divisions, procedure sections, paragraphs, extents, ordinals, single-label PERFORM edges (207 lines)
- `internal/parser/languages/cobol_grammar_procedure_test.go` - the five procedure unit tests with invented fixtures
- `internal/parser/languages/cobol_grammar.go` - version `/3`, `cobolStampRange` delegates to `cobolStampMeta`, Extract calls the walk, `cobolProcedure`, and `cobolDedupeEdges`
- `internal/parser/languages/cobol_grammar_test.go` - `ProgramNode` and `NestedIDs` count only the file-to-program `defines` edges
- `cmd/gortex/cobol_tracer_test.go` - fixture gains `PERFORM WORK-PARA` and `WORK-PARA`; version literal `/3`; `TestCobolTracer_ParagraphFlow`

## TDD Evidence
- **Task 1 RED:** `GOWORK=off go test -count=1 ./cmd/gortex -run '^TestCobolTracer_'` exited 1. `TestCobolTracer_ParagraphFlow` failed on its own assertions (get_callers returned no nodes for the WORK-PARA ID; get_symbol found no MAIN-PARA), and the program contract tests failed on the `/3` version literal. **GREEN:** the same command passed after the walk and procedure files landed.
- **Task 2 RED:** `go test ./internal/parser/languages/ -run '^TestCobolGrammar_(Divisions|...)$'` exited 1 with only `TestCobolGrammar_Divisions` failing, on its "no node" assertion for `src/div.cbl::DIVPGM/@identification_division` (expected: four division nodes; actual: procedure division only). The other four passed, because Task 1 already implemented that behavior. **GREEN:** `bbeb0ccd`.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so the RED records above were captured by hand.

## Decisions Made
- Breadth runs before the document-flag block in Extract, so the document-only flags stay on the file node, as the existing comment intends.
- PERFORM edges record `cobol_name_norm=cobol-name-norm/1`, so the D-18 rule ID is in meta wherever a label match depends on it.
- An extent ends at the last procedure-division child before the ending header. That child's span covers its descendants, so the end is never cut short by a nested token.
- A division's Name is its observation kind, matching its ID segment.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Breadth placed before the document-flag block**
- **Found during:** Task 1
- **Issue:** The plan placed the walk after the document-flag block. Cloning `docMeta` there would copy `prov_containment_unresolved` and `prov_unnamed_program_count` into every breadth node and edge, which the existing code deliberately keeps on the file node only.
- **Fix:** `cobolProcedure(newCobolWalk(...))` runs right after the program loop, before those flags are set. `cobolDedupeEdges` is still the last step before return.
- **Files modified:** internal/parser/languages/cobol_grammar.go
- **Verification:** `TestCobolGrammar_ContainmentUnresolved` and `TestCobolGrammar_UnnamedProgramCounted` pass; all TestCobol* pass under -race.
- **Committed in:** e5c9baff

**2. [Rule 2 - Missing critical] Name-rule ID recorded on PERFORM edges**
- **Found during:** Task 1
- **Issue:** D-18 requires the matching rule ID in meta, but the PERFORM step in the plan did not write it.
- **Fix:** Every PERFORM edge now carries `cobol_name_norm=cobol-name-norm/1`. The key is already in the phase meta-key list.
- **Files modified:** internal/parser/languages/cobol_grammar_procedure.go
- **Verification:** `TestCobolGrammar_PerformEdge` asserts it.
- **Committed in:** e5c9baff

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both keep the provenance contract honest. No scope creep.

## Issues Encountered
- Interim risk, owned by Plan 03-04: until the `cobol::` resolver exclusions land, the generic resolver sees `unresolved::cobol::label::…` targets. The namespaced form makes a by-name bind unlikely, but 03-04 is what guarantees it.

## Known Stubs
None.

## Threat Flags
None. The new surface is extractor-internal and is covered by T-03-01..T-03-05. IDs pass through `cobolIDSegment`, labels bind only on a unique same-program match, edges are deduped, and fixtures are invented.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Ready for 03-02 (BOM offsets, edge VCS stamping) and the wave-2 walkers. 03-03 statements can read `w.container`, `w.flowFrom`, and `relationEdge`. 03-06 replaces the inline PERFORM emission in `cobolPerform` (qualified labels, THRU, GO TO).
- `cobol_grammar_walk.go` is at 297 of the 300-line target. Later helpers (`statementSymbol`, `qualifiedName`, `resourceNode`) will need to split it along a real seam, for example moving the edge builders and dedupe into their own file.

## Self-Check: PASSED
- FOUND: internal/parser/languages/cobol_grammar_walk.go
- FOUND: internal/parser/languages/cobol_grammar_procedure.go
- FOUND: internal/parser/languages/cobol_grammar_procedure_test.go
- FOUND: e5c9baff, ec7913de, bbeb0ccd
- Task 1 verify (both commands) and Task 2 verify pass under -race. gofmt is clean on the touched files, and `git grep 'gortex-cobol-grammar/2' -- internal cmd` is empty. `cobol.go`, `cobol_test.go`, and `register.go` are untouched, and `TestCobolGrammar_RangeNeverInvented` is unmodified.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
