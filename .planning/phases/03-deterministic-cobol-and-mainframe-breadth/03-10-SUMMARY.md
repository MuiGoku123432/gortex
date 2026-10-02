---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 10
subsystem: testing
tags: [cobol, mcp, cli, sqlite, resolver, neo4j-projection, provenance, db2, idms, cics]

requires:
  - phase: 03-02
    provides: BOM byte-offset alignment on nodes and relationship edges
  - phase: 03-04
    provides: resolveCobolReferences exactly-one same-repository binding
  - phase: 03-06
    provides: PERFORM THRU and GO TO DEPENDING ON calls edges
  - phase: 03-07
    provides: CALL and COPY-family reference edges
  - phase: 03-08
    provides: IDMS and CICS resource references
  - phase: 03-09
    provides: DB2 table reads/writes, cursors, dynamic SQL
provides:
  - Real-path acceptance over two tracked repositories through MCP (get_symbol, get_call_chain, get_callers) and the CLI relay (gortex call get_callers)
  - COBOL embedded-SQL KindTable nodes and their DB2 edges now survive the default-off sql coverage gate
  - Locked contracts for provenance keys, Meta value types, Neo4j-safe key names, and degraded documents
  - Projection check that every Phase 3 node Meta key reaches Neo4j unchanged
  - docs/languages.md COBOL row describing Phase 3 breadth
affects: [03-11, 03-12, 03-13, phase-05-producer, phase-06-query]

actuals:
  tokens: 9335
  tasks: 2
  commits: 3
plan_head_before: c48bfb7a2254310cf8ed81c72942c647ed69e60f

tech-stack:
  added: []
  patterns:
    - "Two repositories tracked by one MultiIndexer into one SQLite store, B before A, so A's resolve runs while the cross-repo decoy exists"
    - "serveCobolTestDaemon: one daemon-over-relay helper shared by the tracer and breadth tests"

key-files:
  created:
    - cmd/gortex/cobol_breadth_test.go
    - internal/parser/languages/cobol_grammar_contract_test.go
  modified:
    - internal/indexer/indexer.go
    - internal/indexer/coverage_strip_test.go
    - cmd/gortex/cobol_tracer_test.go
    - internal/neo4jprojection/properties_test.go
    - docs/languages.md

key-decisions:
  - "COBOL embedded-SQL tables (KindTable with Language cobol) survive stripSQLArtifacts like migration and ORM tables: they come from parsed EXEC SQL, not string-literal guesses"
  - "The daemon wiring was split out of startCobolTracerDaemon into serveCobolTestDaemon instead of copied, because a breadth copy would have been the third occurrence of that wiring"
  - "Degraded nodes are matched to their handoff observation by (prov_observation_kind, original start byte)"

patterns-established:
  - "Acceptance tests assert unresolved COBOL targets by their unprefixed IDs (unresolved::cobol::...), and dynamic targets keep the repo-relative path"

requirements-completed: [DET-01, DET-02, DET-03, DET-04, DET-05, DET-06, DET-07, DET-08, DET-09, DET-10, DET-11, DET-12]

coverage:
  - id: D1
    description: "Structure and control flow (data item, statement, division, THRU range, GO TO DEPENDING ON) through MCP get_symbol/get_call_chain/get_callers and the CLI get_callers relay, with no LLM service"
    requirement: DET-02
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_StructureAndFlow"
        status: pass
    human_judgment: false
  - id: D2
    description: "CALL/COPY outcomes on the persisted graph: bound single same-repo candidate, ambiguous with sorted candidate IDs, missing despite a same-named program in repository B, dynamic, idms_dictionary_member"
    requirement: DET-09
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_CallsAndCopy"
        status: pass
    human_judgment: false
  - id: D3
    description: "IDMS, CICS, DB2 edges with verb/command Meta and statement ranges; persisted table set equals the physical tables named"
    requirement: DET-12
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_MainframeAndSQL"
        status: pass
      - kind: unit
        ref: "internal/indexer/coverage_strip_test.go#TestStripSQLArtifacts_PreservesCobolSQLTables"
        status: pass
    human_judgment: false
  - id: D4
    description: "BOM-prefixed file: relationship edge prov_start_byte is the raw-file offset, BOM included (CR-01)"
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_BOMEdgeOffsets"
        status: pass
    human_judgment: false
  - id: D5
    description: "Provenance keys, nil Meta on structural edges, Meta value types, Neo4j-safe key names, degraded amber/red breadth"
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_contract_test.go#TestCobolGrammar_ProvKeys"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_contract_test.go#TestCobolGrammar_DegradedBreadth"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every Phase 3 node Meta key projects unchanged as meta_<key>"
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/properties_test.go#TestProjectNode_CobolBreadthMeta"
        status: pass
    human_judgment: false
  - id: D7
    description: "docs/languages.md COBOL row reads accurately to a user and names no estate content"
    verification: []
    human_judgment: true
    rationale: "Documentation wording and accuracy is a reader judgment; grep only proves the row mentions paragraphs and no longer says 'temporarily not extracted'"

duration: 25min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 10: Real-Path COBOL Breadth Acceptance and Cross-Cutting Contracts Summary

**Two invented repositories tracked through one MultiIndexer into SQLite now show every DET-01..DET-12 behavior through the existing MCP tools and CLI relay, binding nothing beyond a single same-repository candidate. The run exposed and fixed a real gap: the default-off `sql` coverage gate was silently deleting every COBOL DB2 table and SQL edge before persistence.**

## Performance

- **Duration:** about 25 min
- **Started:** 2026-10-02T09:16:05Z
- **Completed:** 2026-10-02T09:41:10Z
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments
- `TestCobolBreadth_StructureAndFlow`: over the daemon relay, `get_symbol` (detail full) returns a data item (`field`, `data_item`), an 01 record (`variable`), a statement (`local`, `statement`), and the procedure division (`package`). `get_call_chain` from MAIN-PARA reaches STEP-A..STEP-E, which covers both the THRU range and the GO TO DEPENDING ON targets. `get_callers` on STEP-B returns MAIN-PARA over MCP and over `runCall(cmd, ["get_callers"])` with no stub. The server has no LLM service.
- `TestCobolBreadth_CallsAndCopy`: on the persisted graph, `CALL 'SUBPGM'` binds to `<A>/cobol/subpgm.cbl::SUBPGM`. `CALL 'TWINPGM'` keeps `unresolved::cobol::program::TWINPGM` with `ambiguous` and both sorted full IDs. `CALL 'ONLYB'` keeps its target with `missing` while `<B>/cobol/onlyb.cbl::ONLYB` exists. Repository B is tracked first, so A's resolve runs while ONLYB is in the graph. `CALL WS-PGM` keeps its dynamic target. COPY CUSTREC binds to the copybook, and COPY IDMS keeps `idms_dictionary_member`.
- `TestCobolBreadth_MainframeAndSQL`: the IDMS reads edge carries `idms_verb=OBTAIN`, the CICS map writes edge carries `cics_command=SEND`, the ORDERS reads edge carries `sql_aliases=[O]`, AUDIT_LOG gets a writes edge, and the FETCH edge carries `sql_via_cursor=C1`. Each edge has `Line == prov_start_row + 1`. Repository A's table set is exactly `MYSCH/ORDERS`, `_unqualified/AUDIT_LOG`, `_unqualified/ORDHIST`. Repository B has no tables.
- `TestCobolBreadth_BOMEdgeOffsets`: the PERFORM edge in the BOM-prefixed file has `prov_start_byte` equal to the index of `PERFORM` in the raw bytes (BOM included), and the matching 0-based row.
- `TestCobolGrammar_ProvKeys` locks five rules on a fixture that reaches every walker. Every analyzed node carries the document keys and `gortex-cobol-grammar/3`. Every relationship edge also carries `prov_observation_kind`, `prov_affected`, and `prov_range_exact`. `defines` and `member_of` edges have nil Meta. Every value is a string, bool, int, float64, or []string. No key contains a secret-filter word.
- `TestCobolGrammar_DegradedBreadth`: amber and red documents still emit sections, paragraphs, statements, and data items. Each carries the grade keys and its own observation's `prov_affected`.
- `TestProjectNode_CobolBreadthMeta`: all 35 node keys in the 03-01 contract list project as `meta_<key>`, with the key and the value unchanged.
- The `docs/languages.md` COBOL row describes Phase 3 breadth. It notes that copybook data items are not yet extracted, and it keeps the Windows note.

## Task Commits

1. **Task 1: Real-path breadth acceptance across two repositories through MCP and the CLI relay**
   - `2d3aeda5` (fix): keep COBOL embedded-SQL tables through the default-off sql gate
   - `8361c96f` (test): breadth acceptance tests and the shared daemon helper
2. **Task 2: Cross-cutting contract tests, Neo4j projection check, and the COBOL language row** - `1a8bd599` (test)

## Files Created/Modified
- `cmd/gortex/cobol_breadth_test.go` - fixtures, `indexCobolBreadthRepos`, and the four `TestCobolBreadth_*` tests
- `cmd/gortex/cobol_tracer_test.go` - `serveCobolTestDaemon` split out of `startCobolTracerDaemon` (no behavior change)
- `internal/indexer/indexer.go` - `isCobolSQLTableNode`; `stripSQLArtifacts` keeps COBOL tables
- `internal/indexer/coverage_strip_test.go` - `TestStripSQLArtifacts_PreservesCobolSQLTables`
- `internal/parser/languages/cobol_grammar_contract_test.go` - `TestCobolGrammar_ProvKeys`, `TestCobolGrammar_DegradedBreadth`
- `internal/neo4jprojection/properties_test.go` - `TestProjectNode_CobolBreadthMeta`
- `docs/languages.md` - COBOL row

## Decisions Made
- COBOL embedded-SQL tables survive the `sql` gate. The gate exists for noisy string-literal SQL guesses in code. EXEC SQL tables are parsed statements, the same reasoning that already keeps migration and ORM tables. The predicate is `Kind == KindTable && Language == "cobol"`. No other COBOL extractor emits tables.
- The daemon wiring is shared, not copied. `startCobolTracerDaemon` was already the second copy of `spinUpDaemonWithConfig`'s wiring, so a breadth copy would have been the third. The repository and store wiring is duplicated (second occurrence) and marked with a `// note:` comment, as the plan says.
- Degraded nodes are matched to their observation by kind and original start byte. Every emitted section, paragraph, statement, and data item in the fixtures has an exact range.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Default coverage config deleted every COBOL DB2 table and SQL edge**
- **Found during:** Task 1 (first probe of the persisted graph)
- **Issue:** The `sql` coverage domain is off by default. `stripSQLArtifacts` then removes every `KindTable` node that is not of migration or ORM origin, plus every edge touching one. The 03-09 tables and their reads/writes/cursor edges were in the extractor output but never reached SQLite. That broke DET-12 on the real path, and unit tests could not see it.
- **Fix:** Added `isCobolSQLTableNode` to the survival condition and updated the doc comment. Added a unit test.
- **Files modified:** internal/indexer/indexer.go, internal/indexer/coverage_strip_test.go
- **Verification:** With the predicate temporarily removed, `TestCobolBreadth_MainframeAndSQL` fails (`"[]" should have 1 item(s)`). With it, the test passes. `internal/indexer` passes under `-race`.
- **Committed in:** 2d3aeda5

**2. [Rule 3 - Blocking / CLAUDE.md rule of three] Daemon wiring shared instead of duplicated**
- **Found during:** Task 1
- **Issue:** The plan said to duplicate the `startCobolTracerDaemon` wiring with a `// note:` comment. That wiring is already a second copy of `spinUpDaemonWithConfig`, so another copy would be the third occurrence.
- **Fix:** Split the daemon part into `serveCobolTestDaemon(t, dir, store, idx, mi, cm)`. `startCobolTracerDaemon` now calls it. The repository and store wiring is still duplicated with a note, as planned.
- **Files modified:** cmd/gortex/cobol_tracer_test.go (outside `files_modified`)
- **Verification:** All seven `TestCobolTracer_*` tests pass under `-race`.
- **Committed in:** 8361c96f

**3. [Plan expectation not deliverable] The acceptance program grades amber, not green**
- **Found during:** Task 1 (probe)
- **Issue:** Under the pinned producer, every `EXEC SQL` statement leaves an `sql_unparsed_tail`, so any SQL-bearing document grades amber (`unparsed-tail`). The plan says to keep fixtures parseable "so documents grade green", but ORDPGM must carry SQL.
- **Fix:** None. No assertion depends on a green grade, and every edge is still emitted. BOMPGM, SUBPGM, the twins, and ONLYB grade green. Recorded as WINDOWS.md entry 6 (deviation, Phase 5 producer gap).
- **Committed in:** n/a

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking/reuse) and 1 recorded producer limitation.
**Impact on plan:** Fix 1 is required for DET-12 to hold on the real path, and only an end-to-end test could catch it. Fix 2 removes duplication. There is no scope creep.

## Issues Encountered
- In this producer, no emitted section, paragraph, statement, or data-item node has an affected observation. Only sub-expression and `comment_entry` observations are affected. So `TestCobolGrammar_DegradedBreadth` compares nodes whose expected value is `false`. It still fails if a node's `prov_affected` ever diverges from its observation.

## Known Stubs
None.

## Threat Flags
None. T-03-31: all fixtures are invented, and the docs name no estate content. T-03-32: `TestCobolGrammar_ProvKeys` and `TestProjectNode_CobolBreadthMeta` are in place. T-03-33: `TestCobolBreadth_CallsAndCopy` proves there is no cross-repository binding by name and that two same-repository candidates stay ambiguous. The strip change widens which nodes persist only to COBOL-language tables. It adds no new input surface.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 3 breadth is proven end to end. 03-11..03-13, including the D-16 copybook producer prerequisite, can build on it.
- Existing SQLite stores that indexed COBOL with SQL need a reindex to pick up the DB2 tables and edges the strip used to delete.
- Verification: `GOWORK=off go test -race -count=1 ./internal/parser/languages/ ./internal/resolver/ ./internal/indexer/ ./internal/neo4jprojection/ ./cmd/gortex/` exited 0. `go build ./...` is clean.

## Self-Check: PASSED
- FOUND: cmd/gortex/cobol_breadth_test.go, internal/parser/languages/cobol_grammar_contract_test.go
- FOUND commits: 2d3aeda5, 8361c96f, 1a8bd599
- Both task verify commands and the per-wave suite pass. `grep -c resolveCobolReferences cmd/gortex/cobol_breadth_test.go` prints 0. `grep -n 'temporarily not extracted' docs/languages.md` prints nothing.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
