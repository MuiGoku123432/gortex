---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 07
subsystem: parser
tags: [cobol, call, copy, copybook, endevor-inc, copy-idms, sql-include, unresolved-namespace, resolver, mcp]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-04 resolveCobolReferences (exactly-one same-repo binding on cobol_kind + cobol_name_key); 03-05 cobolDataItems index and cobolDataReference; 03-03 span-placed COPY statements and flow sources"
provides:
  - "cobolReferences(w, data): CALL, COPY, -INC, COPY IDMS, and EXEC SQL INCLUDE reference edges"
  - "cobolCallTarget(w, data, from, occurrence, programID, target, kind, role): literal vs dynamic program target plus data reference, reusable by CICS LINK/XCTL/PROGRAM (03-08)"
  - "Node Meta cobol_name_key + cobol_name_norm on program nodes; cobol_kind=copybook + cobol_name_key + cobol_name_norm on copybook file nodes"
  - "Edge Meta cobol_call, cobol_target_raw, cobol_copy_form, cobol_copy_member_raw, cobol_copy_library, cobol_copy_idms_kind, sql_include_builtin, cobol_name_norm; reasons dynamic and idms_dictionary_member; cobol_ref_role=call_target"
affects: [03-08, 03-09, 03-13, 04-lifecycle, 05-findings, 06-queries]

actuals:
  tokens: 6933
  tasks: 2
  commits: 3
plan_head_before: 6f3dec0e26e1c4b8da623fa662815b79e242b2d7

tech-stack:
  added: []
  patterns:
    - "Cross-file COBOL references always target an owned unresolved::cobol:: namespace; the extractor never binds them, only resolveCobolReferences does"
    - "A COPY's owning program is found by generated span (innermost named program_definition containing its start), never by its Parent (Pitfall 3)"
    - "ASCII-only case folding (cobolHasPrefixFold) for keyword detection, never Go's Unicode (?i) folding"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_refs.go
    - internal/parser/languages/cobol_grammar_refs_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - cmd/gortex/cobol_tracer_test.go

key-decisions:
  - "COPY IDMS is detected by matching the copy text against the grammar's own _COPY_IDMS token (COPY [ \\t]+ IDMS, ASCII fold), with no trailing-whitespace requirement, so the extractor agrees with how the parser tokenized it"
  - "The EXEC SQL INCLUDE member comes from the sql_include_name child only: the grammar aliases the INCLUDE keyword as a WORD child, so a first-WORD rule would read INCLUDE as the member"
  - "A FUNCTION call target is dynamic and keyed by its whole text; its func_args identifiers get no call_target data reference"
  - "COPY ownership uses the innermost named program by generated span for exec_sql_include too, so every COPY-family form shares one From rule"

patterns-established:
  - "Pattern: an operand-to-target helper (cobolCallTarget) returns every edge it emits, so CICS can reuse it with its own edge kind and cobol_ref_role"

requirements-completed: [DET-03, DET-04, DET-05, DET-06]

coverage:
  - id: D1
    description: "Literal CALL (string or number) emits a calls edge to unresolved::cobol::program::<key> with cobol_call=literal, cobol_target_raw, cobol_name_norm, statement symbol, and full provenance; ID metacharacters are percent-encoded"
    requirement: DET-03
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_refs_test.go#TestCobolGrammar_Calls/literal"
        status: pass
    human_judgment: false
  - id: D2
    description: "Identifier and FUNCTION CALLs emit per-program unresolved::cobol::dynamic::program:: targets with reason dynamic; identifiers also emit a call_target data reference (resolved, qualified, or not_declared_in_program); same identifier in two programs gets two targets"
    requirement: DET-04
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_refs_test.go#TestCobolGrammar_Calls/dynamic"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_refs_test.go#TestCobolGrammar_Calls/qualified_and_program_scoped"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_refs_test.go#TestCobolGrammar_Calls/function"
        status: pass
    human_judgment: false
  - id: D3
    description: "COPY, OF/IN library, quoted .cpy member, -INC, COPY IDMS (with and without kind), and EXEC SQL INCLUDE (SQLCA builtin) emit imports edges to unresolved::cobol::copybook:: targets with the right form and Meta; From is paragraph, program, nested program, or file node by span; cobol-name-norm/1 encoding probe passes"
    requirement: DET-05
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_refs_test.go#TestCobolGrammar_Copy"
        status: pass
    human_judgment: false
  - id: D4
    description: "Program nodes carry cobol_name_key; copybook file nodes carry cobol_kind=copybook and cobol_name_key for lower- and upper-case paths"
    requirement: DET-06
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_refs_test.go#TestCobolGrammar_CopybookKey"
        status: pass
    human_judgment: false
  - id: D5
    description: "Through the real MultiIndexer, SQLite, and production resolver, CALL 'SUBPGM' binds to <prefix>/cobol/subpgm.cbl::SUBPGM, COPY CUSTREC binds to <prefix>/copy/custrec.cpy, and MCP get_callers on SUBPGM returns MAIN-PARA over the daemon relay"
    requirement: DET-03
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_CallAndCopyBinding"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolTracer_'"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 07: COBOL CALL and COPY-Family Reference Edges Summary

**Literal and dynamic CALLs, and every COPY form (COPY, Endevor -INC, COPY IDMS, EXEC SQL INCLUDE), are now separate, source-positioned edges to owned `unresolved::cobol::` targets. Program and copybook nodes carry the `cobol_name_key` that `resolveCobolReferences` matches on. A literal CALL and a COPY bind across files on the real indexer, SQLite, and resolver path, and `get_callers` sees the call.**

## Performance

- **Duration:** about 10 min
- **Started:** 2026-10-02T07:50:57Z
- **Completed:** 2026-10-02T08:01:20Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- `cobolReferences(w, data)` walks every `call_statement` in a named program's procedure division and every `copy_statement` / `exec_sql_include`. `Extract` now captures `data := cobolDataItems(w)` and passes it on.
- `cobolCallTarget`: a `string` or `number` operand becomes a `cobol_call=literal` edge to `unresolved::cobol::program::<cobolIDSegment(cobolNameNorm(text))>`. A `qualified_word` or bare `WORD` operand becomes a `cobol_call=dynamic` edge to `unresolved::cobol::dynamic::program::<programID>::<key>[/<qualifier key>…]` with reason `dynamic`, plus a `cobol_ref_role=call_target` data reference through `cobolDataReference`. A `function_` operand is dynamic and keyed by its text, with no data reference. Each call edge records `cobol_target_raw` (as written) and `cobol_name_norm`. The helper returns its edges so 03-08 can reuse it for CICS.
- `cobolCopy`: the member and library are the first and second `WORD`/`string` children. The member key is `cobolIDSegment` of `cobolNameNorm(member)` with one trailing `.CPY` removed. The forms are:
  - `inc`: the original bytes at the observation start begin with `-INC` (ASCII fold).
  - `idms`: the text matches the grammar's `_COPY_IDMS` token. The target is `copybook::idms/<kind key or _>/<key>`, with `cobol_copy_idms_kind` only when a kind is written, and reason `idms_dictionary_member`.
  - `sql_include`: the member comes from `sql_include_name`, and `sql_include_builtin` is set for SQLCA and SQLDA.
  - `cobol_copy_library` is written only when OF/IN is present.
- Ownership (Pitfall 3): the program is the innermost named `program_definition` whose generated span contains the COPY's start. From is the procedure flow source inside a procedure division, else that program, else the file node.
- Node keys: each program node carries `cobol_name_key` and `cobol_name_norm`. Each copybook file node carries `cobol_kind=copybook`, `cobol_name_key` (basename without extension), and `cobol_name_norm`. Its other keys are unchanged.
- Tracer: the fixture repository now has an invented `SUBPGM` program and `CUSTREC` copybook. `TestCobolTracer_CallAndCopyBinding` shows tracking's production resolver binding both, and MCP `get_callers` on SUBPGM returning MAIN-PARA. The test never calls `resolveCobolReferences` directly.

## Task Commits

1. **Task 1: CALL and COPY-family reference edges and the binding keys on program and copybook nodes** - `cbcbb584` (test, RED), `8e9569c7` (feat, GREEN)
2. **Task 2: Prove CALL and COPY binding through the real indexer, SQLite, resolver, and get_callers** - `d23c1204` (test)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_refs.go` - `cobolReferences`, `cobolCallTarget`, `cobolCopy`, `cobolCopyIDMS`, `cobolHasPrefixFold` (207 lines)
- `internal/parser/languages/cobol_grammar_refs_test.go` - invented fixtures; `TestCobolGrammar_Calls`, `TestCobolGrammar_Copy`, `TestCobolGrammar_CopybookKey`
- `internal/parser/languages/cobol_grammar.go` - program and copybook node keys; `data := cobolDataItems(w)` then `cobolReferences(w, data)` (291 lines). The extractor version stays at `gortex-cobol-grammar/3`.
- `cmd/gortex/cobol_tracer_test.go` - three-file fixture repository, `TestCobolTracer_CallAndCopyBinding`, and a ParagraphFlow filter fix

## TDD Evidence
- **RED** (`cbcbb584`): `GOWORK=off go test ./internal/parser/languages/ -run '^TestCobolGrammar_(Calls|Copy|CopybookKey)$'` exited 1. Every subtest failed on its own assertion: `"[]" should have 1 item(s), but has 0` for the missing edges, and `expected "DEMOPGM" / "copybook" / "CUSTREC", actual <nil>` for the missing node keys. Nothing failed by panic, compile error, or zero-test discovery. Before committing, a throwaway probe (deleted, never committed) confirmed that all three fixtures grade green and that the sibling programs get the IDs the tests expect (`CALLPGM`, `OTHERPGM`, `OUTERPGM/INNERPGM`).
- **GREEN** (`8e9569c7`): the Task 1 verify command passed. That covers `-list` of the three exact names, `go test -race ./internal/parser/languages/ -run '^TestCobol'`, `go vet`, and gofmt.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so this RED record was captured by hand (same as 03-01 and 03-04).

## TDD Gate Compliance
- Task 1: RED `cbcbb584` (`test(03-07)`) precedes GREEN `8e9569c7` (`feat(03-07)`). No refactor commit was needed.
- Task 2 is `type="auto"` (not TDD). Its commit is `test(03-07)` because it changes only a test file.

## Decisions Made
- COPY IDMS detection mirrors the grammar's `_COPY_IDMS` token exactly, so the extractor's form always agrees with the parse.
- The SQL INCLUDE member is read only from `sql_include_name`, because the grammar aliases the INCLUDE keyword as a `WORD` child.
- A FUNCTION target is dynamic, keyed by its text, and gets no data reference: its argument identifiers are not the program name.
- One span-based From rule covers every COPY-family form, `exec_sql_include` included.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] TestCobolTracer_ParagraphFlow counted every calls edge from MAIN-PARA**
- **Found during:** Task 2
- **Issue:** The plan puts `CALL 'SUBPGM'` in MAIN-PARA, so MAIN-PARA now has two `calls` edges: the PERFORM and the CALL. ParagraphFlow's `require.Len(calls, 1)` was meant to find the one PERFORM edge, and the extra edge would break it.
- **Fix:** The filter now keeps only `calls` edges that carry `cobol_flow` (control flow). Every assertion on that edge is unchanged. The fixture still grades green.
- **Files modified:** cmd/gortex/cobol_tracer_test.go
- **Verification:** all seven `TestCobolTracer_*` tests pass under `-race`
- **Committed in:** d23c1204

**2. [Rule 2 - Missing critical] DET-05 encoding probe pinned by a test**
- **Found during:** Task 1 (GREEN)
- **Issue:** The must-have "member equality is byte equality after cobol-name-norm/1 (embedded doubled quotes un-doubled, non-ASCII bytes unchanged)" had no test case in the planned behavior list.
- **Fix:** Added `COPY 'cpé''x'.` to the Copy fixture. It asserts target key `CPé'X`, raw member `'cpé''x'`, and `cobol_name_norm`.
- **Files modified:** internal/parser/languages/cobol_grammar_refs_test.go
- **Committed in:** 8e9569c7

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** The only files changed are the four planned ones. Neither fix adds scope.

## Issues Encountered
- In the pinned grammar, `exec_sql_include` aliases its INCLUDE keyword as a `WORD` child. The first draft took the member from the WORD/string children and was corrected before GREEN.
- The `-INC` copy observation's original projection is not exact (the Endevor transform rewrote the line), but its start byte is the column-1 `-INC`. Its edge therefore carries `prov_range_exact=false` and real coordinates.

## Known Limits (carried from the plan, not gaps)
- Dynamic CALL targets are never bound in Phase 3. No VALUE clause or value-flow evidence is read (DET-04 prohibition).
- `COPY … OF/IN <library>` is recorded in `cobol_copy_library`, but the resolver does not narrow candidates by it. Library search order is Phase 4.
- As 03-03 noted, a COPY after a procedure division's last statement falls outside that division's span. It is then emitted from its program node, with no statement symbol.

## Known Stubs
None.

## Threat Flags
None. T-03-22: every key segment passes through `cobolIDSegment`, and the `CALL 'A/B#1'` test covers it. T-03-23: only the owned `unresolved::cobol::` namespaces are emitted (the acceptance grep finds 0 regex-era namespaces, and both tests assert the target prefixes). Dynamic and IDMS forms carry reasons that the resolver never rebinds. T-03-24: the tracer's extra program and copybook are invented.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 03-08 can call `cobolCallTarget(w, data, from, stmt, programID, operand, kind, "cics_program")` for CICS LINK/XCTL/PROGRAM operands. A bare `WORD` operand is already handled.
- 03-13 can extend the copybook branch. Its file node already carries the binding keys.
- Existing SQLite stores need a reindex to pick up the new edges and keys. The extractor version stays at `/3`, as the phase contract requires.

## Self-Check: PASSED
- FOUND: internal/parser/languages/cobol_grammar_refs.go, cobol_grammar_refs_test.go, cobol_grammar.go, cmd/gortex/cobol_tracer_test.go
- FOUND: cbcbb584, 8e9569c7, d23c1204
- Both task verify commands pass under `-race`. `go build ./...` and `go vet` are clean. `git diff --name-only 6f3dec0e..HEAD` lists only the four planned files. The COBOL tests in `internal/resolver`, `internal/indexer`, and `internal/parser/...` pass.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
