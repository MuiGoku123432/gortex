---
phase: 03-deterministic-cobol-and-mainframe-breadth
verified: 2026-10-02T13:57:00Z
status: gaps_found
score: 11/13 must-haves verified
covered_files:
  - .planning/REQUIREMENTS.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-01-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-01-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-02-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-02-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-03-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-03-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-04-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-04-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-05-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-05-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-06-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-06-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-07-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-07-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-08-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-08-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-09-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-09-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-10-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-10-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-11-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-11-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-12-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-12-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-13-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-13-SUMMARY.md
  - docs/languages.md
  - go.mod
  - internal/indexer/incremental_resolve.go
  - internal/indexer/source_revision.go
  - internal/parser/languages/cobol_grammar.go
  - internal/parser/languages/cobol_grammar_data.go
  - internal/parser/languages/cobol_grammar_data_clauses.go
  - internal/parser/languages/cobol_grammar_dedupe.go
  - internal/parser/languages/cobol_grammar_flow.go
  - internal/parser/languages/cobol_grammar_identity.go
  - internal/parser/languages/cobol_grammar_mainframe.go
  - internal/parser/languages/cobol_grammar_procedure.go
  - internal/parser/languages/cobol_grammar_refs.go
  - internal/parser/languages/cobol_grammar_sql.go
  - internal/parser/languages/cobol_grammar_statements.go
  - internal/parser/languages/cobol_grammar_walk.go
  - internal/resolver/cobol_references.go
  - internal/resolver/cross_repo.go
  - internal/resolver/resolver.go
covered_digest: "v1:sha256:6a15fc8cdcbd846f1e2a9681577fd7a57095b24f69ec6a1389f10649bcacf9cc"
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Phase goal clause 'without false deterministic resolution': a literal CALL is bound only to a program a separately compiled caller can actually reach"
    status: failed
    reason: "resolveCobolReferences indexes nested (contained) programs as repo-wide candidates. A CALL 'INNER' in a different file binds to OUTER/INNER whenever that nested program is the only INNER in the repository. COBOL scoping makes a nested program uncallable from outside its containing program, so this binding is false. Confirmed by a non-mutating overlay probe: a/caller.cbl CALL 'INNER' -> a/nest.cbl::OUTER/INNER with no cobol_unresolved_reason."
    artifacts:
      - path: "internal/resolver/cobol_references.go"
        issue: "cobolCandidateIndex includes every cobol_kind=program node. The same-file filter (lines 81-92) prefers same-file candidates but falls back to nested programs from other files"
      - path: "internal/parser/languages/cobol_grammar.go"
        issue: "every program node, including contained programs, gets cobol_name_key, and no nested/top-level marker is stamped for the resolver to filter on"
    missing:
      - "Only offer nested programs (QualName contains '/', or a cobol_nested=true Meta stamped at extraction) as candidates for callers in the same file. A cross-file caller sees only top-level programs. A sole nested match then gives missing, not a binding"
      - "A resolver test: cross-file CALL to the only nested INNER stays on unresolved::cobol::program::INNER with cobol_unresolved_reason=missing"
  - truth: "DET-05 / SC3: copybook and include references keep the referenced member name and source occurrence, including COPY statements inside an analyzed copybook"
    status: partial
    reason: "Plan 03-13 turned on copybook analysis, so the parser now exposes the copy_statement observations inside a copybook. The copybook branch of Extract runs only cobolDataSection and cobolDataBind and never calls cobolCopy. A 'COPY INNERCPY.' inside outer.cpy only sets cobol_contains_copy=true on an item. No imports edge is emitted, so the member name INNERCPY and its source range are lost. Confirmed by overlay probe: outer.cpy emitted only defines/member_of edges. Plan 03-13 assigned this to Phase 4, but neither the Phase 4 roadmap success criteria nor the CONTEXT deferred list mention copybook-to-copybook edges. CONTEXT defers only library search order."
    artifacts:
      - path: "internal/parser/languages/cobol_grammar.go"
        issue: "the copybook branch (lines 220-241) never emits COPY or EXEC SQL INCLUDE reference edges"
    missing:
      - "In the copybook branch, call cobolCopy(w, i) for every copy_statement and exec_sql_include observation. cobolCopy already emits from the file node when no program owns the COPY"
      - "Extend TestCobolGrammar_CopybookNestedCopy to assert the imports edge to unresolved::cobol::copybook::<key> with cobol_copy_member_raw and the statement range keys"
deferred:
  - truth: "Incremental re-index keeps bound COBOL CALL/COPY edges in the cobol:: namespace and does not drop inbound COPY edges (review CR-01, WR-01)"
    addressed_in: "Phase 4"
    evidence: "Phase 4 SC3 'Edit, delete, restore ... changes remove or invalidate stale owned observations', SC4 'Incremental reconciliation and a clean rebuild produce the same canonical active deterministic subgraph', SC5 'Resolution ... fail closed ... reject unsupported same-name ... binding'"
  - truth: "Shared DB2/IDMS/CICS resource nodes survive eviction of the first-sighting file without deleting other programs' edges (review CR-02)"
    addressed_in: "Phase 4"
    evidence: "Phase 4 SC1 'namespaced resources ... retain stable, non-colliding identities', SC3, SC4. Plan 03-08 Pitfall 7 assigned shared-node ownership to Phase 4"
  - truth: "Unprefixed per-program unresolved placeholders (dynamic/label/data) cannot collide across repositories with the same file path"
    addressed_in: "Phase 4"
    evidence: "Phase 4 SC5 'report zero unowned or misprefixed active nodes for the acceptance corpus'"
  - truth: "Copybook analysis failure is distinguishable from a missing copybook (review WR-06), and SQL unparsed tails, WORKING-STORAGE DECLARE CURSOR, and DECLARE ... WITH HOLD produce explicit parse findings (WINDOWS.md 4-6)"
    addressed_in: "Phase 5"
    evidence: "Phase 5 goal 'distinguish source that could not be parsed from referenced artifacts that were not retrieved'; SC1 'Named unparsed tails and localized ERROR or MISSING nodes produce ... PARSE_UNRESOLVED findings'"
---

# Phase 3: Deterministic COBOL and Mainframe Breadth Verification Report

**Phase Goal:** Users can inspect source-positioned COBOL structure, control flow, data, calls, copybooks, IDMS, CICS, and SQL without false deterministic resolution.
**Verified:** 2026-10-02T13:57:00Z
**Status:** gaps_found
**Re-verification:** No (initial verification)

## Goal Achievement

The core breadth is real. It is substantive, wired end-to-end (extractor, indexer hook, SQLite, resolver, daemon MCP relay, CLI relay), and behaviorally tested on a two-repository real-path fixture. Two observable defects remain. Both are small and both are in Phase 3 scope:

1. A cross-file literal CALL binds falsely to a nested program.
2. Copybook-mode extraction drops the member name of a nested COPY.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: source-positioned programs, copybooks, divisions, sections, paragraphs, statements, and data items come from named parser nodes | ✓ VERIFIED | `cobol_grammar_procedure.go`, `cobol_grammar_statements.go`, and `cobol_grammar_data.go` build every node from `h.Facts.Observations` and stamp original-coordinate ranges through `cobolStampRange`/`cobolStampExtent`. The tests pass: `TestCobolGrammar_Structure/Statements/Paragraphs/Divisions/DataItems` and `TestCobolGrammar_CopybookDataItems`. On the real path, `TestCobolBreadth_StructureAndFlow` runs MCP `get_symbol` on a field, a variable, a statement, and a division, and `TestCobolTracer_CopybookDataItems` passes. Caveat: procedure-fragment copybooks yield only a file node (see Human Verification 1) |
| 2 | SC2: PERFORM, THRU, and GO TO produce paragraph control flow without claiming unsupported resolution | ✓ VERIFIED | `cobol_grammar_flow.go`: labels resolve program-locally; THRU expands only for a valid range and otherwise goes to one `unresolved::cobol::label::…::thru::…` with a reason; GO TO DEPENDING ON keeps positions; an alterable GO TO marks its node; no fall-through edges. `TestCobolGrammar_ControlFlow` has 14 subtests, including `thru_invalid`, `nested_isolation`, and `no_fallthrough`. Real path: `get_call_chain` from MAIN-PARA reaches STEP-A..E, and MCP plus CLI `get_callers` on STEP-B return MAIN-PARA |
| 3 | SC3: literal calls, dynamic identifiers, copybook references, missing targets, and ambiguous resolution stay distinguishable | ✓ VERIFIED (program-origin references) | `cobolCallTarget` keeps literal calls (`unresolved::cobol::program::`) apart from dynamic ones (`unresolved::cobol::dynamic::program::<prog>::`, reason dynamic). `cobolCopy` handles the copy, inc, idms, and sql_include forms. The resolver stamps missing or ambiguous plus `cobol_candidate_ids`. `TestCobolBreadth_CallsAndCopy` on the persisted graph covers: SUBPGM bound; TWINPGM ambiguous with both IDs; ONLYB missing even though it exists in repo B; WS-PGM dynamic; COPY IDMS `idms_dictionary_member`. References inside copybooks are not covered; see truth 7 |
| 4 | SC4: IDMS record/set and CICS program/transaction/map/mapset references keep exact source evidence | ✓ VERIFIED | `cobol_grammar_mainframe.go` has the verb table, `_unqualified` subschema and mapset, `scope_role_unknown` for WITHIN, and CICS PROGRAM handled through `cobolCallTarget`. `TestCobolGrammar_IDMS` (9 subtests) and `TestCobolGrammar_CICS` (7 subtests) pass. On the real path, the OBTAIN reads `idms::_unqualified/record/EMP-REC` and the SEND writes `cics::map/MSET1/MAP1`, each with verb or command Meta, `cobol_statement_symbol`, and Line = prov_start_row+1 |
| 5 | SC5: DB2 reads and writes keep statement-level evidence, while aliases, CTEs, includes, and dynamic sources stay distinct from physical tables | ✓ VERIFIED | `cobol_grammar_sql.go`: tables come only from `sql_source_candidate`; CTE names are subtracted; aliases and derived aliases go to Meta; PREPARE goes to a dynamic placeholder; EXECUTE is marked `sql_dynamic`. `TestCobolGrammar_SQL` (13 subtests) covers delimited identifiers kept verbatim and the `tables_exact`/`cursor_tables_exact` sets. On the real path, the persisted KindTable set is exactly {MYSCH/ORDERS, _unqualified/AUDIT_LOG, _unqualified/ORDHIST}. Producer limits are documented in WINDOWS.md 4-6; no table is invented and affected documents grade amber or red |
| 6 | Goal clause: no false deterministic resolution on a full index | ✗ FAILED | Overlay probe: `a/caller.cbl` with CALL 'INNER' binds to `a/nest.cbl::OUTER/INNER`, a nested program a separately compiled caller cannot reach. `cobolCandidateIndex` does not distinguish nested programs from top-level ones (also flagged by code review WR-02) |
| 7 | DET-05: copybook/include references keep the member name and occurrence, including references inside analyzed copybooks | ✗ FAILED (partial) | Overlay probe: `outer.cpy` containing `COPY INNERCPY.` emits no imports edge, only `cobol_contains_copy=true`. The copybook branch never calls `cobolCopy` (code review WR-05) |
| 8 | 03-02 CR-01: BOM byte offsets shift by 3 on nodes and edges; any other mismatch is marked inexact; VCS stamp on edges | ✓ VERIFIED | `stampSourceRevision` stamps edges and calls `alignProvByteOffsets` (`source_revision.go:85`). It is reached from `applyCoverageDomains` on 3 indexer paths. `TestSourceRevision*` and `TestCobolBOMByteOffsets` pass. Real path: `TestCobolBreadth_BOMEdgeOffsets` checks that the PERFORM edge's prov_start_byte equals the raw-file offset with the BOM included |
| 9 | 03-04: owned `cobol::` namespace; same-repo exactly-one binding; dynamic/label/data/idms_scope/COPY IDMS never rebound | ✓ VERIFIED | `resolver.go:3379`, `cross_repo.go:1121`, and `incremental_resolve.go:453` exclude `cobol::`. `resolveCobolReferences` is called at `resolver.go:1662`, skips `copybook::idms/`, and handles only `program::`/`copybook::`. `TestCobolNamespaceExclusions`, `TestCobolReferences_*`, and `TestReuseIdentifierSkipsCobolNamespace` pass |
| 10 | Provenance contract: full prov_* on analyzed nodes and relationship edges; structural edges have nil Meta; Meta value types are restricted; no secret-like keys; extractor version /3 | ✓ VERIFIED | `relationEdge`/`symbolNode` clone docMeta, and `structural` sets no Meta. `TestCobolGrammar_ProvKeys` passes; real-path tests assert `gortex-cobol-grammar/3` |
| 11 | Amber and red documents still emit breadth, with grade keys and per-observation prov_affected | ✓ VERIFIED | `TestCobolGrammar_DegradedBreadth` passes. The SQL `amber` subtest asserts that edges carry the amber grade and prov_affected |
| 12 | Neo4j projection keeps Phase 3 node Meta keys as meta_ properties | ✓ VERIFIED | `TestProjectNode_CobolBreadthMeta` passes. Info: the fixture's `cobol_renames` type differs from the extractor's []string (review IN-05) |
| 13 | 03-11/12/13: fork copybook mode is pushed and pinned; copybooks go through AnalyzeCopybook under the approved-grammar check; copybook-owned data items keep member coordinates; no wrapper nodes | ✓ VERIFIED | `go.mod` requires preprocessor `v0.0.0-20261002100409-e64234cc493c`; the forest-shim replace is unchanged at f9eaf99. The module cache copy contains `AnalyzeCopybook`, `RunCopybook`, `PolicyCopybookV1`, and `AnchorsByConstruction`. `cobolApprovedGrammarID` is unchanged since 81d26400 (Phase 2). `Extract` runs the grammar check after both analysis branches. `TestCobolGrammar_CopybookGrammarPinned/ProcedureFragment/ReplacingDegrades` and `TestCobolTracer_CopybookDataItems` pass. Orchestrator-verified: ls-remote shows the fork commit on origin |

**Score:** 11/13 truths verified (0 present, behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Incremental re-index restubs bound COBOL CALLs to bare `unresolved::<Name>` (then generic resolution) and drops inbound COPY edges; `resolveCobolReferences` does not run on the incremental path (review CR-01, WR-01) | Phase 4 | SC3, SC4 (incremental = clean rebuild), SC5 (fail closed, reject unsupported same-name binding) |
| 2 | Shared DB2/IDMS/CICS nodes are owned by the first or last sighting file; evicting that file deletes other programs' edges (review CR-02) | Phase 4 | SC1 namespaced resource identities, SC3/SC4 stale-observation reconciliation; Plan 03-08 Pitfall 7 |
| 3 | Unprefixed per-program placeholder IDs can collide across repositories that share a relative path | Phase 4 | SC5 "zero unowned or misprefixed active nodes" |
| 4 | Copybook analysis failure drops the file node, so COPYs report `missing` (review WR-06); SQL unparsed tails, WORKING-STORAGE DECLARE CURSOR, and WITH HOLD (WINDOWS.md 4-6) | Phase 5 | Goal: distinguish unparsed source from unretrieved artifacts; SC1 PARSE_UNRESOLVED findings |

Deferral note: items 1 and 2 are real hazards for any user who edits files after the initial index. They do not affect the full-index path this phase delivers. Phase 4 must close them before COBOL is used under a watcher.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/parser/languages/cobol_grammar.go` | Extract orchestration, /3 version, copybook branch, pins | ✓ VERIFIED | 301 lines; calls `newCobolWalk`, `cobolProcedure`, `cobolDataItems`, `cobolReferences(w, data)`, `cobolMainframe(w, data)`, `cobolSQL(w, data)`, `cobolDedupeEdges`, `AnalyzeCopybook`, `cobolDataSection` |
| `internal/parser/languages/cobol_grammar_walk.go` | shared walk, relationEdge, resourceNode, stamping | ✓ VERIFIED | 280 lines; `cobolDedupeEdges` lives in sibling `cobol_grammar_dedupe.go` (49 lines), not in walk.go as planned. This is a harmless relocation |
| `internal/parser/languages/cobol_grammar_procedure.go` | divisions, sections, paragraphs, ordinals, extents | ✓ VERIFIED | calls `cobolStatements(w)` and then `cobolFlow(w)` |
| `internal/parser/languages/cobol_grammar_statements.go` | KindLocal statements, COPY span placement | ✓ VERIFIED | |
| `internal/parser/languages/cobol_grammar_data.go` (+ `_data_clauses.go`) | data items, hierarchy, REDEFINES/RENAMES, data index | ✓ VERIFIED | |
| `internal/parser/languages/cobol_grammar_flow.go` | label resolver, THRU, GO TO | ✓ VERIFIED | all edges are built through `relationEdge` |
| `internal/parser/languages/cobol_grammar_refs.go` | CALL, COPY, -INC, COPY IDMS, SQL INCLUDE | ✓ VERIFIED | not wired into the copybook branch (gap 2) |
| `internal/parser/languages/cobol_grammar_mainframe.go` | IDMS verb table, CICS commands | ✓ VERIFIED | CICS PROGRAM goes through `cobolCallTarget` |
| `internal/parser/languages/cobol_grammar_sql.go` | DB2 tables, cursors, dynamic SQL | ✓ VERIFIED | |
| `internal/resolver/cobol_references.go` | exactly-one same-repo binding pass | ⚠️ VERIFIED with defect | wired at `resolver.go:1662`; the nested-program candidate leak is gap 1 |
| `internal/indexer/source_revision.go` | alignProvByteOffsets and edge stamping | ✓ VERIFIED | |
| `cmd/gortex/cobol_breadth_test.go`, `cmd/gortex/cobol_tracer_test.go` | real-path acceptance | ✓ VERIFIED | 431 and 649 lines; all 12 tests pass |
| `internal/parser/languages/cobol_grammar_contract_test.go` | ProvKeys, DegradedBreadth | ✓ VERIFIED | |
| `internal/neo4jprojection/properties_test.go` | TestProjectNode_CobolBreadthMeta | ✓ VERIFIED | |
| `docs/languages.md` | COBOL Phase 3 row | ✓ VERIFIED | Info: it does not state the copybook limits (no nested COPY edges, no procedure-fragment structure) |
| `go.mod` | preprocessor re-pin, forest-shim unchanged | ✓ VERIFIED | |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| cobol_grammar.go | cobol_grammar_procedure.go | `cobolProcedure(w)` | ✓ WIRED | |
| cobol_grammar.go | cobol_grammar_dedupe.go | `cobolDedupeEdges(result.Edges)` on both branches | ✓ WIRED | |
| cobol_grammar_walk.go | cobol_grammar_statements.go | `statementSymbol` in `relationEdge` | ✓ WIRED | |
| cobol_grammar.go | cobol_grammar_data.go | `cobolDataItems(w)` / copybook `cobolDataSection` | ✓ WIRED | |
| cobol_grammar_flow.go | cobol_grammar_walk.go | `relationEdge(` | ✓ WIRED | |
| cobol_grammar_refs.go | resolver/cobol_references.go | `unresolved::cobol::program::` / `copybook::` plus node `cobol_name_key` | ✓ WIRED | proven bound on the persisted graph |
| cobol_grammar_refs.go | cobol_grammar_data.go | `cobolDataReference(` for dynamic CALL | ✓ WIRED | |
| cobol_grammar_mainframe.go | cobol_grammar_refs.go | `cobolCallTarget(` for CICS PROGRAM | ✓ WIRED | |
| cobol_grammar_sql.go | cobol_grammar_walk.go | `resourceNode(` KindTable | ✓ WIRED | |
| resolver.go | cobol_references.go | `r.resolveCobolReferences()` in ResolveAllContext | ✓ WIRED | full resolve only; incremental path deferred to Phase 4 |
| source_revision.go | indexer.go | `alignProvByteOffsets(` inside `stampSourceRevision` | ✓ WIRED | |
| cobol_grammar.go | fork preprocessor | `e.analyzer.AnalyzeCopybook(` under `cobolAnalyzeSlot` | ✓ WIRED | |
| cobol_tracer_test.go | tools_core get_callers | daemon relay `"get_callers"` | ✓ WIRED | |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| Persisted COBOL graph (SQLite) | nodes/edges | handoff Observations → Extract → MultiIndexer → store_sqlite | Yes; read back through `store.GetOutEdges`, `NodesByKinds`, MCP `get_symbol`/`get_callers`/`get_call_chain` | ✓ FLOWING |
| Resolver stamps | cobol_unresolved_reason / cobol_candidate_ids | `resolveCobolReferences` → `ReindexEdges` / `PersistEdgeAttributesBatch` | Yes; asserted on SQLite-persisted edges | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Extractor unit and contract suite | `GOWORK=off go test -count=1 -run TestCobol ./internal/parser/languages/` | ok | ✓ PASS |
| Resolver, indexer, and BOM | `go test -run 'TestCobol\|TestReuseIdentifierSkipsCobolNamespace\|TestSourceRevision' ./internal/resolver/ ./internal/indexer/` | ok, ok | ✓ PASS |
| Neo4j projection | `go test -run TestProjectNode_CobolBreadthMeta ./internal/neo4jprojection/` | ok | ✓ PASS |
| Real-path two-repo acceptance | `go test -v -run 'TestCobolTracer\|TestCobolBreadth' ./cmd/gortex/` | 12/12 PASS | ✓ PASS |
| Cross-file CALL to sole nested program (overlay probe, no repo change) | `go test -overlay … -run TestVerifierProbe_NestedCrossFile ./internal/resolver/` | `CALL 'INNER' -> a/nest.cbl::OUTER/INNER reason=<nil>` | ✗ FAIL (false binding) |
| COPY inside a copybook (overlay probe) | `go test -overlay … -run TestVerifierProbe_NestedCopyInCopybook ./internal/parser/languages/` | no imports edge; `cobol_contains_copy=true` on A-FLD only | ✗ FAIL (member name lost) |
| Procedure-fragment copybook (overlay probe) | same probe | `nodes=1 edges=0 grade=amber absence=<nil>` for PERFORM/CALL/EXEC SQL | ? see Human Verification 1 |
| Full suite (orchestrator) | `go test -race -timeout 30m ./...` | 156 ok packages, exit 0 | ✓ PASS (relied on, not re-run) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared or present for this phase. Step 7c: SKIPPED (no declared probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| DET-01 | 03-01,02,03,05,10,11,12,13 | source-positioned program/division/section/paragraph/statement/data-item observations | ✓ SATISFIED | Truths 1, 8, 10, 11, 13 |
| DET-02 | 03-01,06,10 | PERFORM/THRU/GO TO control flow | ✓ SATISFIED | Truth 2 |
| DET-03 | 03-04,07,10 | literal calls separate from dynamic identifiers | ✓ SATISFIED | Truth 3. The separation itself holds; the false nested binding is tracked under the goal clause (truth 6) |
| DET-04 | 03-04,07,10 | dynamic identifiers stay unresolved | ✓ SATISFIED | `cobolCallTarget` dynamic branch; resolver skips `dynamic::`; CICS data-name operands are dynamic; real-path WS-PGM stays dynamic |
| DET-05 | 03-07,10 | copybook/include references keep member name and occurrence | ✗ BLOCKED (partial) | Program-origin COPY/-INC/SQL INCLUDE/COPY IDMS are satisfied. COPY inside an analyzed copybook loses the member name (gap 2) |
| DET-06 | 03-04,07,10 | copybook ambiguity preserved | ✓ SATISFIED | Exactly-one rule with ambiguous plus candidate IDs. The `OF/IN` library is recorded but not used for binding; library search order is deferred per CONTEXT D-11 |
| DET-07 | 03-08,10 | IDMS record references | ✓ SATISFIED | Truth 4 |
| DET-08 | 03-08,10 | IDMS set references | ✓ SATISFIED | Truth 4 (`set_operand`, `adjacency`) |
| DET-09 | 03-08,10 | CICS program/transaction references | ✓ SATISFIED | Truth 4 (`link_literal`, `link_dynamic`, `transid`) |
| DET-10 | 03-08,10 | CICS map/mapset references | ✓ SATISFIED | Truth 4 (`maps`, real-path SEND MAP) |
| DET-11 | 03-09,10 | SQL read/write to deterministically identified tables | ✓ SATISFIED | Truth 5. INSERT…SELECT source reads and WORKING-STORAGE cursors are producer windows (WINDOWS.md 4-6) and are never guessed |
| DET-12 | 03-09,10 | aliases/CTEs/includes/dynamic sources distinguishable | ✓ SATISFIED | Truth 5 |

No orphaned requirements. All of DET-01..DET-12 are claimed by at least one plan, and REQUIREMENTS.md maps no other ID to Phase 3. Note: REQUIREMENTS.md already marks DET-05 `[x] Complete`; that mark is premature until gap 2 closes.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (all phase source files) | – | TBD/FIXME/XXX/TODO/HACK | none found | Debt marker gate clean |
| internal/resolver/cobol_references.go | 54, 81-92 | nested programs indexed as repo-wide candidates | 🛑 Blocker | False deterministic CALL binding (gap 1) |
| internal/parser/languages/cobol_grammar.go | 220-241 | copybook branch skips reference emission | 🛑 Blocker | Nested COPY member names lost (gap 2) |
| internal/parser/languages/cobol_grammar_sql.go | 151-156 | two DECLAREs of one cursor reported as `cursor_not_declared` | ⚠️ Warning | Misleading reason vocabulary (review WR-07); matches the plan truth as written |
| internal/parser/languages/cobol_grammar_data.go | 64-90, 111-131 | REDEFINES/RENAMES looked up program-wide | ⚠️ Warning | False `ambiguous` for routine repeated child names (review WR-03); conservative, not a false binding |
| fork preprocessor transform/copybook.go | 77-94 | copybook header heuristic (`EXEC` → data header; FD/SD, leading COPY/-INC → procedure header) | ⚠️ Warning | Common members are misclassified and degrade (review WR-04); producer-side |
| internal/parser/languages/cobol_grammar_dedupe.go | 26-46 | PERFORM and GO TO on one line collapse to one cobol_flow | ℹ️ Info | review IN-04 |
| internal/parser/languages/cobol_grammar_identity.go | 102-112 | program symbol not passed through cobolIDSegment | ℹ️ Info | review IN-03 |

### Human Verification Required

Status is `gaps_found`, so these items are listed here but do not set the status. Resolve them alongside gap closure.

#### 1. Procedure-fragment copybooks emit no structure and no absence reason

**Test:** Index a `.cpy` containing only procedure statements (PERFORM, CALL 'X', EXEC SQL SELECT … FROM T1).
**Expected:** Either paragraphs and statements with their CALL/SQL/CICS edges are emitted, or the file node carries an explicit `prov_analysis_absence` reason (for example `copybook_procedure_member_unsupported`).
**Why human:** Overlay probe result: `nodes=1 edges=0 grade=amber`, no absence key. Programs do not expand COPY either, so CALL, SQL, and CICS usage inside procedure copybooks is invisible to impact queries. Plan 03-13 assigned this to Phase 4, but no Phase 4 success criterion names it, and ownership of copybook paragraphs is an open identity-design decision. Decide whether to add it to Phase 4 or close it now.

#### 2. Judgment-tier prohibitions (non-authoritative LLM-judge verdicts, flagged for human review)

All 20 prohibitions across the 13 plans are `verification: judgment` with no wired enforcement, so each is `unverified-prohibition — human review recommended`:

- No implicit fall-through or guessed PERFORM labels (03-01, 03-06). Judge: holds. No fall-through code exists, and unresolved targets carry reasons.
- No estate-derived names, paths, or run IDs in fixtures, tests, docs, or the fork (03-01, 02, 03, 10, 11, 13). Judge: holds as far as visible. Fixtures use invented names (ORDPGM, SUBPGM, CUSTREC, MAIN-PARA). Estate provenance cannot be proven by inspection.
- No byte offset presented as exact when indexed bytes differ from the file (03-02). Judge: holds (`alignProvByteOffsets`, tests).
- No statement source text in statement Meta (03-03). Judge: holds. Only kinds, verbs, cursor names, and CTE names are stored.
- No fabricated level, parent, or copybook items; no PIC/VALUE text in errors (03-05). Judge: holds.
- No choosing among 2+ candidates and no cross-repo binding by name (03-04). Judge: holds for repositories. The cross-file nested-program binding (gap 1) is a related but distinct defect.
- No binding of dynamic, label, data, IDMS-scope, or COPY IDMS targets, and no VALUE heuristic (03-04, 03-07). Judge: holds.
- No bare `unresolved::<name>` and no `import::`/`dyncall::` from the extractor (03-07). Judge: holds on extraction (grep finds only a comment). The incremental restub path can still produce a bare stub (deferred CR-01).
- No guessed IDMS subschema, WITHIN role, or CICS mapset; no binding of data-named CICS operands (03-08). Judge: holds.
- No KindTable for an alias, CTE, include, derived table, or dynamic source; no guessed schema or case fold of delimited identifiers (03-09). Judge: holds (`tables_exact` tests).
- No change to grammar, transform.Run, ConfigurationID, or PolicyV1; fork commit touches exactly 7 files (03-11). Judge: plausibly holds. `cobolApprovedGrammarID` is unchanged and the forest-shim pin is unchanged. The 7-file claim was not re-checked in the fork repository.
- Push only with explicit confirmation, no force-push, only the recorded commit (03-12). Judge: cannot be verified from code; the orchestrator confirmed via ls-remote that the recorded hash is on origin.
- No re-pin of forest-shim, no change to the grammar ID, no copybook bypass of the grammar check, no wrapper nodes; no URL rewrite or credential in global git config (03-13). Judge: holds. Global git config has 0 `url.*` entries (counted only, values not read).

### Gaps Summary

The phase delivers its breadth on the full-index path, with honest unresolved placeholders, and the real-path tests prove it end to end. Two Phase 3-scoped defects remain. Both are cheap to fix and both are confirmed by non-mutating probes:

1. **False deterministic binding to nested programs (goal clause).** `resolveCobolReferences` treats contained programs as repo-wide candidates. Restricting nested programs to same-file callers fixes it.
2. **Nested COPY member names dropped in copybook mode (DET-05).** The copybook branch never calls `cobolCopy`. Calling it for `copy_statement` and `exec_sql_include` observations fixes it; the file-node `From` fallback already exists.

**This may be intentional for gap 2.** Plan 03-13 consciously assigned copybook-to-copybook edges to Phase 4. If you accept that, add an override and add the item to the Phase 4 roadmap so it is not lost:

```yaml
overrides:
  - must_have: "DET-05 / SC3: copybook and include references keep the referenced member name and source occurrence, including COPY statements inside an analyzed copybook"
    reason: "Copybook-to-copybook import edges deferred to Phase 4 with library search order; Phase 4 roadmap updated to include them"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

Gap 1 should not be overridden. It is a false deterministic resolution, which the phase goal forbids.

The incremental-lifecycle hazards (review CR-01, CR-02, WR-01) are real but belong to Phase 4. The parse-boundary findings (WR-06, WINDOWS.md 4-6) belong to Phase 5. They are recorded as deferred. The WINDOWS.md entries are honest documented producer windows: no table is invented, and affected documents grade amber or red. They are not goal gaps.

---

_Verified: 2026-10-02T13:57:00Z_
_Verifier: Claude (gsd-verifier)_
