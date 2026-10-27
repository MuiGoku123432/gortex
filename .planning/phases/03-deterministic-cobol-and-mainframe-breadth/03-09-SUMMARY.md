---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 09
subsystem: parser
tags: [cobol, db2, exec-sql, kindtable, cursor, dynamic-sql, cte, alias, reads, writes]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-08 (w *cobolWalk) resourceNode and cobolUnquote; 03-05 cobolDataItems index and cobolDataReference; 03-03 statement nodes and flow sources; 03-07 EXEC SQL INCLUDE on the COPY path; cobolDedupeEdges"
provides:
  - "cobolSQL(w, data): DB2 table, cursor, and dynamic SQL edges for every EXEC SQL statement of a named program, called from Extract after cobolMainframe"
  - "KindTable IDs db2::<schema key|_unqualified>/<table key> with Meta table, dialect=db2, schema (only when written), cobol_name_norm, emitted once per document through resourceNode"
  - "Edge Meta sql_verb, sql_aliases, sql_location, sql_cursor, sql_via_cursor; cobol_ref_role=sql_dynamic_source; dynamic target unresolved::cobol::dynamic::sql::<programID>::<key>"
  - "Statement Meta sql_verb, sql_ctes, sql_cte_refs, sql_derived_aliases, sql_cursor, sql_dynamic, cobol_unresolved_reason=cursor_not_declared"
  - "cobolSQLName (SQL clause of cobol-name-norm/1) and cobolSQLParts (dot split outside double quotes) in cobol_grammar_identity.go"
affects: [03-13, 04-lifecycle, 05-findings, 06-queries]

actuals:
  tokens: 6512
  tasks: 2
  commits: 4
plan_head_before: 9b6fef87df3a31fbbde18b859180ae2835566c3b

tech-stack:
  added: []
  patterns:
    - "Physical tables come only from sql_source_candidate observations; the candidate's parent kind decides writes, reads, or references"
    - "One edge per statement, table, and kind; alias texts collect on it as sql_aliases"
    - "Cursor names are read from the statement text after the verb, because underscore identifiers are hidden tokens"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_sql.go
    - internal/parser/languages/cobol_grammar_sql_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_identity.go

key-decisions:
  - "A KindTable's Name and table Meta are the table part as written (go_sql precedent), schema Meta is the schema part as written, and the ID keys use the SQL clause of cobol-name-norm/1 (delimited identifiers verbatim with quotes, ordinary ones upper-folded) through cobolIDSegment"
  - "Cursor names are taken from the statement text (first token after the verb that is not an orientation keyword, host variable, or number), not from the first WORD: underscore names such as C_EMP are hidden tokens, and FETCH ABSOLUTE :hv FROM c would otherwise pick the host variable"
  - "A DECLARE whose cursor cannot be read registers nothing, so a cursor use with no readable name never binds; cursor-use edges always have kind reads, sql_verb of the using statement, the DECLARE's sql_location, and no aliases"
  - "Resolved OPEN, FETCH, and CLOSE statements also carry sql_cursor on their statement node, not only the unresolved ones"
  - "An sql_source_candidate whose parent is not listed in cobolSQLKinds references its table (honest default per D-14)"

patterns-established:
  - "Pattern: every DB2 table node is emitted through w.resourceNode(id, table, graph.KindTable, meta); aliases, CTE names, derived tables, cursors, prepared statements, and dynamic sources never reach it"

requirements-completed: [DET-11, DET-12]

coverage:
  - id: D1
    description: "Physical tables named by sql_source_candidate are KindTable db2::<schema|_unqualified>/<table> nodes with table, dialect=db2, schema only when written, cobol_name_norm, and shared-resource provenance; the whole sql_table_name span is used (MYSCH.AUDIT_LOG gives schema MYSCH and table AUDIT_LOG), a three-part name keeps sql_location on the edge, and a delimited identifier keeps its quotes and case with only ID metacharacters percent-encoded"
    requirement: DET-11
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/select"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/tables_exact"
        status: pass
    human_judgment: false
  - id: D2
    description: "INSERT, UPDATE, DELETE, and MERGE targets get writes, FROM/JOIN and MERGE USING sources get reads, and CREATE, ALTER, and DROP TABLE get references, each from the statement's flow source with sql_verb as written and the statement occurrence range"
    requirement: DET-11
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/writes"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/ddl"
        status: pass
    human_judgment: false
  - id: D3
    description: "Aliases collect as sql_aliases on one deduplicated edge and are never nodes; a same-statement CTE name is no table and gets no edge (sql_ctes, sql_cte_refs), while tables inside the CTE body are read; a derived-table alias goes to sql_derived_aliases"
    requirement: DET-12
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/aliases"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/cte"
        status: pass
    human_judgment: false
  - id: D4
    description: "DECLARE c CURSOR edges carry sql_cursor; OPEN, FETCH (plain, NEXT FROM, ABSOLUTE :hv FROM), and CLOSE read the tables of the one same-program DECLARE with sql_via_cursor from their own paragraph; zero or two DECLAREs give cursor_not_declared and sql_cursor on the statement and no edge"
    requirement: DET-12
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/cursor"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/cursor_unknown"
        status: pass
    human_judgment: false
  - id: D5
    description: "PREPARE s FROM :hv references unresolved::cobol::dynamic::sql::<path>::<prog>::<hv> with reason dynamic plus a sql_dynamic_source data reference; EXECUTE and EXECUTE IMMEDIATE set sql_dynamic=true; no table exists for a cursor, prepared statement, or dynamic source"
    requirement: DET-12
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/prepare"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/execute"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/cursor_tables_exact"
        status: pass
    human_judgment: false
  - id: D6
    description: "SQL-bearing programs grade amber (unparsed-tail) and every SQL edge and node is still emitted with prov_document_grade and prov_affected"
    requirement: DET-11
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/amber"
        status: pass
    human_judgment: false

duration: 17min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 09: Embedded DB2 SQL Tables, Cursors, and Dynamic SQL Summary

**Embedded DB2 SQL statements now read, write, or reference shared `KindTable` nodes `db2::<schema|_unqualified>/<table>`. Tables come only from source and carry statement-level provenance. Aliases, CTE names, and derived tables are statement Meta and never tables. Cursor use resolves through the one same-program DECLARE, and PREPARE goes to a per-program dynamic placeholder that is never bound.**

## Performance

- **Duration:** about 17 min
- **Started:** 2026-10-02T08:41:00Z
- **Completed:** 2026-10-02T08:59:00Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- `cobolSQL(w, data)` runs in `Extract` right after `cobolMainframe(w, data)`. It visits every `exec_sql_statement` of a named program. `exec_sql_include` stays on the COPY path from 03-07.
- **Verb:** the verb is the first `WORD` of the statement's `sql_*` subnode. A WITH clause is skipped, so its verb is the SELECT. The fallback form (OPEN, FETCH, CLOSE, EXECUTE, COMMIT, …) keeps a bare `WORD`. `sql_verb` is kept as written on the statement node and on every edge.
- **Tables:**
  - Table text is the whole `sql_table_name` span, split on dots outside double quotes. One part gives `_unqualified`. Two parts give schema and table. With three or more parts, the earlier parts become `sql_location` on the edge.
  - Delimited identifiers keep their quotes and case. Ordinary identifiers are upper-folded. Every key segment goes through `cobolIDSegment`, so `"A/B"."x""y"` becomes `db2::"A%2FB"/"x""y"`.
- **Classification:** the candidate's parent kind decides the edge, through `cobolSQLKinds`:
  - INSERT, UPDATE, DELETE, and MERGE targets get `writes`.
  - FROM, JOIN, and MERGE USING sources get `reads`.
  - CREATE, ALTER, and DROP TABLE get `references`.

  Each statement has one edge per (table, kind), and `sql_aliases` collects the alias texts for that table in source order.
- **CTE and derived tables:** an unqualified candidate that matches a same-statement `sql_cte_name` is dropped and recorded in `sql_cte_refs`. `sql_ctes` lists the CTE names. An alias not under a source candidate becomes `sql_derived_aliases`.
- **Cursors:** DECLARE edges carry `sql_cursor`. OPEN, FETCH, and CLOSE read every table of the single same-program DECLARE, from their own paragraph, with `sql_via_cursor`. With zero or several DECLAREs, the statement gets `cobol_unresolved_reason=cursor_not_declared` and `sql_cursor`, and no edge.
- **Dynamic SQL:**
  - PREPARE references `unresolved::cobol::dynamic::sql::<programID>::<hv key>` with reason `dynamic`, plus `cobolDataReference(..., "sql_dynamic_source")`, which resolved to the `WS-SQLTXT` data item in the test.
  - EXECUTE and EXECUTE IMMEDIATE set `sql_dynamic=true` on the statement node.

## Task Commits

1. **Task 1: SQL verbs, physical tables, read and write classification, aliases, and CTEs** - `364c2a4d` (test, RED), `e41c726c` (feat, GREEN)
2. **Task 2: Cursors, PREPARE, EXECUTE, and DDL** - `6e0a9254` (test, RED), `7ac0943f` (feat, GREEN)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_sql.go` - `cobolSQLKinds`, `cobolSQLTable`, `cobolSQLOrientation`, `cobolSQL`, `cobolSQLCursor`, `cobolSQLVerb`, `cobolSQLTables`, `cobolSQLAlias` (297 lines)
- `internal/parser/languages/cobol_grammar_sql_test.go` - invented fixtures `cobolSQLFixture` and `cobolSQLCursorFixture`; `TestCobolGrammar_SQL` with 12 subtests; helpers `cobolTableIDs`, `assertCobolTable`, `cobolEdgeTargets`
- `internal/parser/languages/cobol_grammar.go` - `cobolSQL(w, data)` call (293 lines). The extractor version stays `gortex-cobol-grammar/3` (phase contract).
- `internal/parser/languages/cobol_grammar_identity.go` - `cobolSQLName` and `cobolSQLParts` beside the other name-to-ID rules (220 lines)

## TDD Evidence
- **RED Task 1** (`364c2a4d`): `GOWORK=off go test ./internal/parser/languages/ -run '^TestCobolGrammar_SQL$'` exited 1. All six subtests failed on their own assertions, for example `"[]" should have 1 item(s)`, `expected: []string{"writes db2::_unqualified/T1"} actual: []string(nil)`, and `expected []string{"RECENT"} actual <nil>`. Statement-node lookups succeeded, so the fixture parsed and only the behavior was missing. The first RED run panicked on an index into an empty edge list. The test was changed to `require.Equal` before indexing, so the committed RED fails on assertions only.
- **GREEN Task 1** (`e41c726c`): the full Task 1 verify passed: `-list`, `go test -race ./internal/parser/languages/ -run '^TestCobol'`, and gofmt.
- **RED Task 2** (`6e0a9254`): it exited 1 with missing `sql_cursor`, `sql_via_cursor`, `cursor_not_declared`, PREPARE edges, and `sql_dynamic`. `ddl` and `cursor_tables_exact` passed before GREEN, as expected: Task 1 already classifies DDL and creates no non-physical tables.
- **GREEN Task 2** (`7ac0943f`): the full Task 2 verify passed: `-race` on `^TestCobol` and on `cmd/gortex` `^TestCobolTracer_`, `go vet`, gofmt, and the 300-line cap (297).
- Before the tests were written, a throwaway observation probe (deleted, never committed) confirmed the shapes described in Deviations below.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so the RED records were captured by hand, as in 03-01, 03-04, 03-07, and 03-08.

## TDD Gate Compliance
- Task 1: RED `364c2a4d` (`test(03-09)`) precedes GREEN `e41c726c` (`feat(03-09)`).
- Task 2: RED `6e0a9254` (`test(03-09)`) precedes GREEN `7ac0943f` (`feat(03-09)`).
- No refactor commits were needed.

## Decisions Made
- A table's Name and `table` Meta are the table part as written, as in the go_sql precedent. Meta values keep the source spelling; IDs use the SQL name rule.
- Cursor names come from the statement text, not the first WORD (see deviation 2).
- Cursor-use edges take the DECLARE's `sql_location` but not its aliases. Aliases belong to the declaring statement.
- An unlisted candidate parent references its table, the honest default for a verb that cannot be classified (D-14).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] DECLARE CURSOR fixture moved from WORKING-STORAGE to the procedure division**
- **Found during:** Task 2, from the pre-test probe
- **Issue:** The pinned grammar admits only `exec_sql_declare_section` and `exec_sql_include` in WORKING-STORAGE (`grammar.js` working_storage_section). `EXEC SQL DECLARE C1 CURSOR FOR SELECT …` there becomes stray `EXEC`/`SQL`/`comment_entry` observations, not an `exec_sql_statement`, and grades the document red. The behavior "a reads edge From the program node with no cobol_statement_symbol" cannot be produced.
- **Fix:** The fixture declares its cursors in paragraph P0. The DECLARE edge comes from P0 and carries a statement symbol. `cobolSQL` keeps a program-node flow source for an EXEC SQL statement outside any procedure division, per the plan truth, but the pinned producer never reaches it. It is recorded as WINDOWS.md entry 4 (unrun-verify).
- **Files modified:** internal/parser/languages/cobol_grammar_sql_test.go, internal/parser/languages/cobol_grammar_sql.go
- **Verification:** `/cursor` passes, and the cursor fixture grades amber
- **Committed in:** 6e0a9254, 7ac0943f

**2. [Rule 1 - Bug] Cursor names read from statement text instead of the first WORD**
- **Found during:** Task 2, from the pre-test probe
- **Issue:** The plan's rule takes the second WORD of DECLARE and the first non-orientation WORD after OPEN, FETCH, or CLOSE. Underscore identifiers are hidden tokens, so `DECLARE C_EMP CURSOR` has `CURSOR` as its second WORD, and `FETCH C_EMP INTO :WS-X` has `INTO` as its first WORD. `FETCH ABSOLUTE :WS-N FROM C1` would also pick the host variable `WS-N`. Each case records a wrong `sql_cursor`.
- **Fix:** `cobolSQLCursor` scans the generated text after the verb. It skips orientation keywords, `:host` tokens, and numbers, and stops at END-EXEC. A DECLARE with no readable name registers nothing.
- **Files modified:** internal/parser/languages/cobol_grammar_sql.go
- **Verification:** `/cursor` covers C_EMP (DECLARE and FETCH) and `FETCH ABSOLUTE :WS-N FROM C1`
- **Committed in:** 7ac0943f

**3. [Rule 3 - Blocking] Line budget: SQL name rules moved to cobol_grammar_identity.go**
- **Found during:** Task 2
- **Issue:** `cobol_grammar_sql.go` reached 325 lines, and the Task 2 verify caps it at 300.
- **Fix:** `cobolSQLName` (the SQL clause of cobol-name-norm/1) and `cobolSQLParts` (the qualified-name split) moved unchanged next to `cobolNameNorm`, `cobolUnquote`, and `cobolIDSegment`, as 03-08 did. The file is now 297 lines.
- **Files modified:** internal/parser/languages/cobol_grammar_identity.go (outside `files_modified`)
- **Verification:** every `TestCobol*` test passes under `-race`, and `go build ./...` is clean
- **Committed in:** 7ac0943f

**4. [Plan expectation not deliverable] INSERT INTO T1 SELECT … FROM T2 does not read T2**
- **Found during:** Task 1, from the pre-test probe
- **Issue:** The producer models only the INSERT target. The trailing SELECT is plain `WORD`s inside `sql_unparsed_tail`, with no `sql_source_candidate`.
- **Fix:** No guess, matching the plan's own rule for aliases without AS: T2 inside the tail is not an observation. The test asserts writes T1 only, and `tables_exact` confirms no T2 table exists. This is a Phase 5 producer gap, recorded as WINDOWS.md entry 5.
- **Files modified:** internal/parser/languages/cobol_grammar_sql_test.go
- **Committed in:** 364c2a4d

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking) and 1 plan expectation the producer cannot deliver
**Impact on plan:** One file outside `files_modified` changed: `cobol_grammar_identity.go` received two moved helpers. The test fixture differs from the plan in two places, both forced by producer limits. No false table and no guessed edge was added.

## Issues Encountered
- `DECLARE … CURSOR WITH HOLD FOR` produces an ERROR cascade in the pinned grammar (the probe graded it red), so the fixture avoids it. This is a producer limit and belongs to Phase 5.

## Known Limits (carried from the plan and the probe, not gaps in this plan)
- Aliases written without AS (`FROM CUST C`) stay in the unparsed tail and are not recorded.
- `EXECUTE IMMEDIATE :x` has no dynamic-source observation and only marks `sql_dynamic`.
- A delimited cursor name containing spaces would split at the space in `cobolSQLCursor`.
- A shared table node's FilePath is its last writer in SQLite (Pitfall 7). Ownership is Phase 4 (ID-04). Existing stores need a reindex to pick up the new nodes and edges.

## Known Stubs
None.

## Threat Flags
None. T-03-28: every key segment goes through `cobolSQLName` and `cobolIDSegment`, and delimited identifiers keep their quotes (`"A/B"` becomes `"A%2FB"`, the `/select` test). T-03-29: aliases, CTE names, derived tables, cursors, prepared statements, and dynamic sources never reach `resourceNode`. The `tables_exact` and `cursor_tables_exact` tests assert the exact KindTable sets. T-03-30 is accepted as planned: no SQL is built or executed.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 03-13 phase acceptance can run DB2 table edges through the real indexer, SQLite, and MCP. Tables are `KindTable` with `dialect=db2`, `schema`, and `table`, the keys the existing orphan and unreferenced-table analyzers read.
- Phase 6 queries can filter SQL use by edge kind, `sql_verb`, `sql_via_cursor`, and the `unresolved::cobol::dynamic::sql::` namespace.

## Self-Check: PASSED
- FOUND: internal/parser/languages/cobol_grammar_sql.go, cobol_grammar_sql_test.go, cobol_grammar.go, cobol_grammar_identity.go
- FOUND: 364c2a4d, e41c726c, 6e0a9254, 7ac0943f (`git rev-list --count 9b6fef87..HEAD` = 4)
- Both task verify commands pass under `-race`. `go build ./...` and `go vet` are clean. The COBOL tests in `internal/resolver`, `internal/indexer`, `internal/neo4jprojection`, and all of `internal/parser/...` pass. `git diff --name-only 9b6fef87..HEAD` lists the three planned files plus `cobol_grammar_identity.go`. The acceptance greps pass: line 1 is the build tag, `cobolSQL(w, data)` appears once, `graph.KindTable` appears once, and `sql_via_cursor` and `cursor_not_declared` are present.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
