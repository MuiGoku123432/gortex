---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 08
subsystem: parser
tags: [cobol, idms, cics, kindresource, resource_type, mainframe, reads, writes, unresolved-namespace]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-07 cobolCallTarget (literal vs dynamic program target plus data reference, returns its edges); 03-05 cobolDataItems index and cobolDataReference; 03-03 statement nodes and flow sources; cobolDedupeEdges"
provides:
  - "cobolMainframe(w, data): IDMS verb-classified record, set, and scope edges and CICS command-classified program, transaction, map, and mapset edges, called from Extract after cobolReferences"
  - "(w *cobolWalk) resourceNode(id, name, kind, meta) string: per-document first-sighting shared node with prov_range_absence=shared_resource and no structural edge (03-09 reuses it for KindTable)"
  - "KindResource IDs idms::_unqualified/{record,set}/<key>, cics::transaction/<key>, cics::map/<mapset key|_unqualified>/<key>, cics::mapset/<key>; resource_type idms_record|idms_set|cics_transaction|cics_map|cics_mapset"
  - "Statement Meta idms_verb, idms_area, idms_erase_scope, cics_command; edge Meta idms_verb, idms_erase_scope, cics_command, cobol_ref_role cics_program|cics_transaction|cics_map|cics_mapset; reason scope_role_unknown"
  - "cobolUnquote(raw): quote strip and un-double without case folding (cobolNameNorm now calls it)"
affects: [03-09, 03-13, 04-lifecycle, 05-findings, 06-queries]

actuals:
  tokens: 6993
  tasks: 2
  commits: 4
plan_head_before: 81cacbf383b4c27ff28128d145dd2e2f2f62bc36

tech-stack:
  added: []
  patterns:
    - "Shared mainframe resources are emitted once per document through w.resourceNode and owned by no file (Pitfall 7); every relationship edge keeps its own statement provenance"
    - "Operand classification by observation kind: string/number child = literal resource, WORD child = per-program dynamic target plus data reference"
    - "Verb tables are keyed by the grammar's verb kind, while Meta keeps the verb text as written"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_mainframe.go
    - internal/parser/languages/cobol_grammar_mainframe_test.go
  modified:
    - internal/parser/languages/cobol_grammar_walk.go
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_identity.go

key-decisions:
  - "IDMS verbs are classified by the verb observation's kind (OBTAIN, FIND, …), so case-folded source still classifies, and idms_verb keeps the written text (for example obtain)"
  - "idms_erase_scope joins only the observed ALL/PERMANENT/SELECTIVE/MEMBERS keywords of an ERASE; COMMIT ALL and idms_unparsed_tail never feed it"
  - "A literal CICS resource's Name is cobolUnquote of the literal (case as written, quotes dropped); its ID key is cobol-name-norm/1, so 'tx01' and 'TX01' share one node"
  - "A map's qualifier is the first literal MAPSET key in the same statement, wherever it appears in the option list; a data-name MAPSET leaves the map _unqualified"
  - "cobol_name_norm is stamped on every IDMS and CICS edge, including the idms_scope and dynamic placeholders, because each key uses the rule"
  - "cobolNameNorm, cobolIDSegment, and cobolNameNormRule moved from cobol_grammar_walk.go to cobol_grammar_identity.go (name-to-ID rules sit with program identity), keeping the walk at 280/300 lines"

patterns-established:
  - "Pattern: resourceNode is the only way a COBOL walker emits a shared node; it returns the ID so callers write w.relationEdge(from, w.resourceNode(...), kind, stmt)"

requirements-completed: [DET-07, DET-08, DET-09, DET-10]

coverage:
  - id: D1
    description: "IDMS records and sets are shared KindResource nodes idms::_unqualified/{record,set}/<key> (resource_type, cobol_name_norm, prov_range_absence=shared_resource, no defines edge); verbs classify edges: OBTAIN/FIND/GET read, STORE/MODIFY/ERASE/CONNECT/DISCONNECT write, BIND/ACCEPT reference; idms_verb on statement and edges"
    requirement: DET-07
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/obtain"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/set_operand"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/update"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/accept"
        status: pass
    human_judgment: false
  - id: D2
    description: "WITHIN scope references unresolved::cobol::idms_scope::_unqualified/<key> with scope_role_unknown; BIND RUN-UNIT, FINISH, COMMIT, ROLLBACK, READY yield statement nodes with idms_verb and no edge; READY records idms_area; ERASE records idms_erase_scope"
    requirement: DET-08
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/find_within"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/session"
        status: pass
    human_judgment: false
  - id: D3
    description: "Edge probe: a record and a set spelled EMP-DEPT are two nodes; two same-line reads of one set collapse to one edge with cobol_occurrence_count 2 and both statement symbols; a different line stays separate; resource nodes appear once in first-sighting order; two extractions give identical node and edge lists; lower-case source folds to the same node"
    requirement: DET-08
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/adjacency"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/deterministic"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_IDMS/name_norm"
        status: pass
    human_judgment: false
  - id: D4
    description: "CICS LINK/XCTL PROGRAM literal calls unresolved program targets through cobolCallTarget; a data-name PROGRAM calls the per-program dynamic target plus a cics_program data reference; LOAD PROGRAM references; no CICS program is a resource node"
    requirement: DET-09
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_CICS/link_literal"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_CICS/link_dynamic"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_CICS/load"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_CICS/resources"
        status: pass
    human_judgment: false
  - id: D5
    description: "CICS TRANSID references cics::transaction/<key> or a dynamic transaction target plus cics_transaction data reference; SEND MAP writes, RECEIVE MAP reads, other commands reference cics::map/<literal mapset|_unqualified>/<key>; MAPSET references cics::mapset/<key> or a dynamic mapset target; cics_command on every statement node and edge"
    requirement: DET-10
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_CICS/transid"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_CICS/maps"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_mainframe_test.go#TestCobolGrammar_CICS/other"
        status: pass
    human_judgment: false

duration: 11min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 08: IDMS and CICS Mainframe References Summary

**IDMS records and sets and CICS transactions, maps, and mapsets are now shared `KindResource` nodes with source-only qualifiers (`idms::_unqualified/...`, `cics::map/<literal mapset or _unqualified>/...`). Each statement reaches them through edges classified by verb or command, with full provenance. CICS LINK and XCTL reuse the CALL path, and data-name operands go to per-program dynamic targets that are never bound.**

## Performance

- **Duration:** about 11 min
- **Started:** 2026-10-02T08:17:54Z
- **Completed:** 2026-10-02T08:28:39Z
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments
- `cobolMainframe(w, data)` runs in `Extract` right after `cobolReferences(w, data)`. It visits every IDMS statement (`idms_navigation_statement`, `idms_update_statement`, `idms_session_statement`, `idms_accept_statement`) and every `exec_cics_statement` that has a flow source in a named program.
- **IDMS:** the edge kind comes from the verb's observation kind, through `cobolIDMSKinds`:
  - OBTAIN and FIND read the record and the set.
  - GET reads the record.
  - STORE, MODIFY, and ERASE write the record.
  - CONNECT and DISCONNECT write the record and the set.
  - BIND, ACCEPT … CURRENCY, and any unlisted verb reference their operands.

  Records and sets become `idms::_unqualified/{record,set}/<key>`. A WITHIN `idms_scope_name` references `unresolved::cobol::idms_scope::_unqualified/<key>` with `scope_role_unknown`. READY records `idms_area` on its statement node. ERASE records its observed scope keywords as `idms_erase_scope` (for example `PERMANENT MEMBERS`) on the statement and the edge. FINISH, COMMIT, ROLLBACK, and BIND RUN-UNIT produce a statement node with `idms_verb` and no edge.
- **CICS:** the command is the first `WORD` after `CICS`, and `cics_command` keeps it as written on the statement and every edge.
  - A PROGRAM operand goes through `cobolCallTarget`: `calls` for LINK and XCTL, `references` otherwise.
  - TRANSID and MAPSET operands are `references`.
  - A MAP is `writes` for SEND, `reads` for RECEIVE, and `references` for anything else.

  A `string` or `number` operand becomes a shared resource. A `WORD` operand becomes `unresolved::cobol::dynamic::{transaction,map,mapset}::<programID>::<key>` with reason `dynamic`, plus a `cobolDataReference` with role `cics_transaction`, `cics_map`, or `cics_mapset`.
- `resourceNode` emits each shared node once per document, at first sighting. The node carries Name as first written, the cloned document provenance, `prov_range_exact=false`, and `prov_range_absence=shared_resource`. It gets no `defines` or `member_of` edge.

## Task Commits

1. **Task 1: Resource nodes and IDMS record, set, and scope references by verb** - `a61159ad` (test, RED), `863c7dc9` (feat, GREEN)
2. **Task 2: CICS program, transaction, map, and mapset references by command** - `1abff964` (test, RED), `427d9c18` (feat, GREEN)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_mainframe.go` - `cobolIDMSKinds`, `cobolMainframe`, `cobolIDMS`, `cobolCICS` (203 lines)
- `internal/parser/languages/cobol_grammar_mainframe_test.go` - invented fixtures; `TestCobolGrammar_IDMS`, `TestCobolGrammar_CICS`; helpers `cobolStatementEdges`, `cobolResourceIDs`, `assertCobolResource`
- `internal/parser/languages/cobol_grammar_walk.go` - `resources` set on the walk and `resourceNode` (280 lines)
- `internal/parser/languages/cobol_grammar.go` - `cobolMainframe(w, data)` call (292 lines). The extractor version stays `gortex-cobol-grammar/3` (phase contract).
- `internal/parser/languages/cobol_grammar_identity.go` - now holds `cobolNameNormRule`, `cobolNameNorm`, `cobolIDSegment` (moved), and the new `cobolUnquote` (191 lines)

## TDD Evidence
- **RED Task 1** (`a61159ad`): `GOWORK=off go test ./internal/parser/languages/ -run '^TestCobolGrammar_IDMS$'` exited 1. Each subtest failed on its own assertion: `"[]" should have 1 item(s), but has 0` for the missing edges, and `expected "BIND" … actual <nil>` for the missing `idms_verb`. The statement-node lookups succeeded, which shows the fixture parsed and only the behavior was missing. `deterministic` passed before GREEN, as expected for an identity check.
- **GREEN Task 1** (`863c7dc9`): the Task 1 verify command passed: `-list`, `go test -race ./internal/parser/languages/ -run '^TestCobol'`, and gofmt.
- **RED Task 2** (`1abff964`): `-run '^TestCobolGrammar_CICS$'` exited 1 with missing-edge assertions, `expected "RETURN" … actual <nil>`, and `resources` returning nil. The first RED run also exposed an off-by-one in the test's own statement count (14 instead of the fixture's 15). That was fixed before the RED commit, so the committed RED fails only on missing behavior.
- **GREEN Task 2** (`427d9c18`): the Task 2 verify command passed: `-list`, `-race`, `go vet`, gofmt, and the 300-line cap.
- A throwaway probe (deleted, never committed) confirmed the observation shapes before the tests were written. Verbs and IDMS keywords such as `PERMANENT`, `MEMBERS`, and `RUN_UNIT` are observed kinds. Operands are direct children of the statement. Each `cics_*_name` wraps one `string` or `WORD` child. All fixtures grade green.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so these RED records were captured by hand, as in 03-01, 03-04, and 03-07.

## TDD Gate Compliance
- Task 1: RED `a61159ad` (`test(03-08)`) precedes GREEN `863c7dc9` (`feat(03-08)`).
- Task 2: RED `1abff964` (`test(03-08)`) precedes GREEN `427d9c18` (`feat(03-08)`).
- No refactor commits were needed.

## Decisions Made
- IDMS verbs are classified by observation kind, so mixed-case source still classifies, and `idms_verb` keeps the written text.
- `idms_erase_scope` is built only from the ALL, PERMANENT, SELECTIVE, and MEMBERS keywords of an ERASE. COMMIT ALL never feeds it.
- A literal CICS resource is named by `cobolUnquote(literal)` and keyed by cobol-name-norm/1. `'tx01'` and `'TX01'` share one node, which keeps the first spelling.
- A map's qualifier is the first literal MAPSET of the same statement, wherever that option appears. A data-name MAPSET leaves the map `_unqualified` and is never defaulted.
- `cobol_name_norm` is stamped on every IDMS and CICS edge, the placeholders included.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Line budget: moved the name-to-ID rules out of cobol_grammar_walk.go**
- **Found during:** Task 1
- **Issue:** `cobol_grammar_walk.go` was at 298/300 lines, and `resourceNode` plus its dedupe set adds about 25.
- **Fix:** `cobolNameNormRule`, `cobolNameNorm`, and `cobolIDSegment` moved, unchanged, to `cobol_grammar_identity.go`, where the program-identity rules already live. The walk is now at 280 lines.
- **Files modified:** internal/parser/languages/cobol_grammar_walk.go, internal/parser/languages/cobol_grammar_identity.go
- **Verification:** `go build ./...` and every `TestCobol*` test pass under `-race`
- **Committed in:** 863c7dc9

**2. [Rule 2 - Missing critical] cobolUnquote for the names of literal resources**
- **Found during:** Task 2
- **Issue:** "Name as first written" for a quoted literal would have kept the quotes (`'TX01'`). The only existing unquoting was inside `cobolNameNorm`, which also upper-folds.
- **Fix:** The quote handling moved out of `cobolNameNorm` into `cobolUnquote`. `cobolNameNorm` calls it, so its behavior is unchanged, and CICS uses it for resource names. This avoids a second unquote implementation.
- **Files modified:** internal/parser/languages/cobol_grammar_identity.go
- **Verification:** the existing `TestCobolGrammar_Copy` encoding probe (`'cpé''x'` → `CPé'X`) and the new `transid` lower-case case pass
- **Committed in:** 427d9c18

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** One file outside `files_modified` changed: `cobol_grammar_identity.go`, which received the moved helpers. No behavior changed in existing code, and nothing was added beyond the plan's scope.

## Issues Encountered
- The "two references from one statement to the same set" adjacency case cannot happen in the grammar: no IDMS statement form has two set operands. Edges collapse by (From, To, Kind, Line), so the test puts two same-paragraph statements on one line instead, which is the case `cobolDedupeEdges` handles. The survivor lists both statement symbols.

## Known Limits (carried from the plan, not gaps)
- The IDMS subschema is never observed (the transform neutralizes `DB … WITHIN …`), so every IDMS resource is `_unqualified`.
- FILE and QUEUE CICS operands are not named observations and get no edge. Content inside `cics_unparsed_tail` or `idms_unparsed_tail` is not read.
- A shared resource node's FilePath is its last writer in SQLite (Pitfall 7). Ownership is Phase 4 (ID-04).
- Existing stores need a reindex to pick up the new nodes and edges.

## Known Stubs
None.

## Threat Flags
None. T-03-25: every key passes through `cobolNameNorm` and `cobolIDSegment`, and every qualifier is a source literal or `_unqualified`. T-03-26: data-name operands go only to `unresolved::cobol::dynamic::` targets with reason `dynamic`, never to a resource (the `link_dynamic`, `transid`, and `maps` tests). T-03-27 is accepted as planned.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 03-09 can call `w.resourceNode(id, name, graph.KindTable, meta)` for DB2 tables. It returns the ID and dedupes per document.
- Phase 6 queries can filter mainframe use by `resource_type`, the edge kind, `idms_verb`, and `cics_command`.

## Self-Check: PASSED
- FOUND: internal/parser/languages/cobol_grammar_mainframe.go, cobol_grammar_mainframe_test.go, cobol_grammar_walk.go, cobol_grammar.go, cobol_grammar_identity.go
- FOUND: a61159ad, 863c7dc9, 1abff964, 427d9c18 (`git rev-list --count 81cacbf3..HEAD` = 4)
- Both task verify commands pass under `-race`. `go build ./...` and `go vet` are clean. The `TestCobolTracer_*` tests in `cmd/gortex`, the COBOL tests in `internal/resolver` and `internal/indexer`, and `internal/parser/...` pass. `git diff --name-only 81cacbf3..HEAD` lists the four planned files plus `cobol_grammar_identity.go`. The acceptance greps pass: `cobolCallTarget(` appears 1 time and the program-namespace string 0 times in cobol_grammar_mainframe.go.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
