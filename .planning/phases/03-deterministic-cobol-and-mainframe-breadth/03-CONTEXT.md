# Phase 3: Deterministic COBOL and Mainframe Breadth - Context

**Gathered:** 2026-10-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Rebuild COBOL graph breadth from the enhanced grammar's handoff `Facts.Observations` (Phase 2 shipped only the program node and its source-file edge). Deliver source-positioned divisions, sections, paragraphs, statements, and data items; explicit `PERFORM` / `PERFORM THRU` / `GO TO` paragraph flow; literal versus dynamic `CALL`; `COPY`/`INCLUDE` references; IDMS record/set, CICS program/transaction/map/mapset, and DB2 table read/write references. The rule throughout is **no false deterministic resolution**: anything the source does not establish stays on a distinguishable unresolved placeholder.

Not in this phase: incremental lifecycle and idempotency proofs (Phase 4), `PARSE_UNRESOLVED` / `EXTERNAL_UNRESOLVED` finding records and their lifecycle (Phase 5), native query acceptance QUERY-01..10 (Phase 6). Requirements: DET-01..DET-12.

</domain>

<decisions>
## Implementation Decisions

### Carried forward (Phase 2, do not re-decide)
- Grammar-only extraction; the regex `CobolExtractor` stays unregistered (P2 D-06, D-16). Every breadth fact comes from handoff `Observations`; names are read from `Generated[span]`; every persisted range is original coordinates via the handoff `Map` (P2 D-10).
- Provenance in flat `prov_*` Meta keys, no new `graph.Node` fields or SQLite columns (P2 D-08). Every new node and edge carries the same `prov_*` contract the program node has.
- IDs extend `<repo>/<path>::OUTER/INNER#k`; names as written (not upcased); line numbers never discriminate (P2 D-10, D-20).
- Amber/red documents and `Affected=true` observations are emitted and marked degraded, never dropped (P2 D-07).
- Invented COBOL fixtures only; the repo is public (P2 D-14).
- `prov_extractor_version` (`gortex-cobol-grammar/1`) must be bumped, because the emitted nodes and edges change (P2 02-03 note).

### Graph vocabulary
- **D-01:** **Reuse existing node kinds plus `cobol_kind` meta. Add no new kinds.** Programs and paragraphs → `KindFunction`. Data items → `KindField`/`KindVariable`. Copybooks → `KindModule` or the existing file node. Sections and divisions → an existing containment kind. Each carries `cobol_kind=program|division|section|paragraph|statement|data_item|copybook`. Research picks the exact kind per construct, with existing query and analyzer compatibility as the tiebreaker. — **Reversibility:** costly — kinds are persisted in SQLite, projected as Neo4j labels, and relied on by Phase 4–6 queries; changing them needs a full reindex and reprojection.
- **D-02:** Paragraph control flow uses **`EdgeCalls` with `cobol_flow=perform|perform_thru|goto` meta**, so `get_call_chain` and `get_callers` trace paragraph execution without changes. — **Reversibility:** costly — same persistence and projection reach as D-01.
- **D-03:** `PERFORM A THRU C` **expands to one edge per paragraph from A through C** in source order within the owning program. All of those edges share the same statement occurrence, and each is marked `cobol_flow=perform_thru`. If C precedes A or either label is not found, emit no expansion and leave the reference unresolved with a reason (see D-12).
- **D-04:** **Explicit transfers only. No implicit fall-through edges.** `GO TO ... DEPENDING ON` emits one edge per listed target. Record each paragraph's source-order ordinal in meta so fall-through can be derived later.

### Statement & data granularity
- **D-05:** Statement nodes: **the user leans toward ALL named statements as nodes**, contained by their paragraph. **Research makes the final call**, starting from "all". It must measure the node, edge, and SQLite/Neo4j projection volume impact (aggregate counts only, no estate content) and may narrow to evidence-bearing statements (CALL, PERFORM, GO TO, COPY, EXEC CICS, EXEC SQL, IDMS DML) only with demonstrated, unacceptable cost. Whatever the choice, every relationship edge must point back to its statement occurrence (range plus statement node ID where one exists).
- **D-06:** Data items: **everything**. That means levels 01–49, 66 (RENAMES), 77, 88 (condition names), and FILLER. The hierarchy comes from level nesting through containment edges. REDEFINES and RENAMES are relationships to the target item. Raw level, PIC, USAGE, OCCURS, VALUE, and the raw name are kept as meta.
- **D-07:** Data-item IDs use the **full group path from the 01 level**: `<owner>::PROG/REC-A/WS-DATE`, extending P2 D-10. A `#k` ordinal applies only to true same-path duplicates and to FILLERs. — **Reversibility:** costly — IDs are persisted and projected, and Phase 4 identity (ID-03/ID-04) builds on them.
- **D-08:** Copybook contents. Copybooks currently skip `Analyze` (`prov_analysis_absence=copybook_standalone_analysis_unsupported`). **Research first determines whether the fork's preprocessor and/or grammar can already parse a copybook (for example as a data-description fragment).** If it can, copybook data items are owned by the copybook node, not duplicated per including program. **If it cannot, research reports back to the user before any workaround is chosen.** Neither a Gortex-side synthetic wrapper nor a producer-side fork change gets planned without the user's decision. A fork change would be scheduled as a producer prerequisite, like P2 D-03.

### Unresolved & ambiguous targets
- **D-09:** Literal `CALL 'PGMX'` emits `calls` → an `unresolved::` placeholder in a COBOL-program namespace. **The existing resolver binds it only to a COBOL program in the same repository scope, and only when exactly one candidate exists.** Zero candidates → reason `missing`. Two or more → reason `ambiguous`, with the candidate IDs in meta, and nothing gets picked. No cross-repository binding by name. Nested and contained programs in the same file resolve within the file first.
- **D-10:** Dynamic `CALL WS-PGM-NAME` emits a **distinct dyncall placeholder scoped to the calling program** (so same-named identifiers in different programs don't merge), plus `cobol_call=dynamic` and a reference edge to the identifier's data-item node. **It is never bound in Phase 3** (DET-04). Literal calls carry `cobol_call=literal`. CICS `LINK`/`XCTL PROGRAM('X')` with a literal uses the same literal-call path. With an identifier it uses the dyncall path.
- **D-11:** `COPY X` resolution: **bind only to a copybook in the same repository whose member name matches exactly once.** The case and extension normalization rule is versioned and recorded. Zero matches → reason `missing`. Two or more → reason `ambiguous`, with candidate IDs in meta, and nothing gets picked. `COPY ... OF/IN <library>` and `COPY IDMS [kind] name` qualifiers are recorded as written. Library search order is Phase 4.
- **D-12:** **Phase 3 / Phase 5 line.** Phase 3 emits each reference edge to its placeholder with a normalized `cobol_unresolved_reason` (`missing | ambiguous | dynamic | label_not_found`, and research may extend the set). It does not create finding nodes and leaves the handoff `Gaps[]` untouched. Phase 5 builds the `PARSE_UNRESOLVED` / `EXTERNAL_UNRESOLVED` findings on top. Nothing that can't be resolved is dropped.

### Mainframe resource modeling
- **D-13:** IDMS records and sets and CICS transactions, maps, and mapsets use **`KindResource` with `resource_type=idms_record|idms_set|cics_transaction|cics_map|cics_mapset`**. DB2 physical tables reuse **`KindTable`**. CICS programs are COBOL-program call targets (D-10), not resources. — **Reversibility:** costly — persisted and projected like D-01.
- **D-14:** Edges are classified by verb and reuse existing kinds: IDMS `OBTAIN`/`FIND`/`GET` → `reads`. `STORE`/`MODIFY`/`ERASE`/`CONNECT`/`DISCONNECT` → `writes`. CICS `RECEIVE MAP` → `reads`, `SEND MAP` → `writes`. Transaction references (`START`/`RETURN TRANSID`) → `references`. SQL `SELECT`/`FETCH`/cursor reads → `reads`, `INSERT`/`UPDATE`/`DELETE`/`MERGE` → `writes`. The raw verb or command is always kept in meta (`idms_verb`, `cics_command`, `sql_verb`). Research supplies the complete verb table, and any verb it cannot classify honestly becomes `references`.
- **D-15:** Resource identity is **repository-scoped, with qualifiers taken only from source**. Examples: `<repo>::db2/<SCHEMA|_unqualified>/TABLE`, `<repo>::idms/<subschema|_unqualified>/RECORD`, `<repo>::cics/mapset/NAME`. An unknown qualifier is an explicit `_unqualified` marker, never a guess. Qualified and unqualified same-named resources stay distinct until Phase 4 resolution domains can justify a merge. — **Reversibility:** costly — the IDs are persisted and projected, and Phase 4 ID-04 depends on them.

### Post-research confirmations (2026-10-01, user-confirmed after 03-RESEARCH.md)
- **D-16:** **Copybook contents use Option A, a producer copybook mode in the fork.** This resolves D-08. Research verified that the pinned producer cannot parse a standalone copybook (the `start` rule accepts only programs, and `Analyze` takes no options).
  - The fork adds a copybook entry point (for example `AnalyzeCopybook`). It wraps the member in a synthetic program/DATA DIVISION header recorded as `sourcemap.KindInsert` segments, so original coordinates and `OriginalContentID` stay the member's own. It also adds a copybook-aware grade/anchor rule.
  - The grammar is unchanged, so the grammar ID stays the same. Only the preprocessor module is re-pinned (`cobolApprovedPreprocessorModule`).
  - Copybook data items are owned by the copybook node and are not duplicated per including program.
  - Procedure-fragment members need a second wrapper variant. `REPLACING` pseudo-text members are expected to fail and surface as explicit gaps or degraded output.
  - **Scheduling:** treat it as a producer prerequisite in Phase 3's **last wave**, like P2 D-03: fork commit, then a **user-confirmed push checkpoint** (outward action), then the Gortex re-pin, then copybook data-item extraction. No other DET work depends on it. — **Reversibility:** costly — a fork API plus a re-pin; copybook-owned data-item IDs are persisted and projected.
- **D-17:** Statement nodes (D-05 resolved to **all named statements**) use **`graph.KindLocal`** with `cobol_kind=statement` and `cobol_statement_kind=<observation kind>`. They attach to their paragraph through `member_of` only (no `defines`). `KindLocal` is already excluded from BM25 search, file-summary definitions, and dead-code results. Structural edges carry no full `prov_*` blob, matching the Phase 2 program edge. — **Reversibility:** costly — the kind is persisted and projected.
- **D-18:** Name matching for CALL/COPY binding and for shared DB2/IDMS/CICS resource IDs uses a versioned rule, **`cobol-name-norm/1` = ASCII uppercase**. The rule ID is recorded in meta. Node display names stay as written (P2 D-20).
- **D-19:** **`COPY IDMS ...` members are never bound.** They stay on their placeholder with a distinct reason (for example `idms_dictionary_member`), so they are never confused with missing copybooks.
- Research defaults adopted, unless the planner finds a blocker:
  - COBOL placeholders get their own `unresolved::cobol::` namespace, which the generic resolver passes skip.
  - One whole-repo `resolveCobolReferences` pass enforces D-09/D-11.
  - The extractor dedupes same-key edges itself, because SQLite keeps the first copy and the in-memory graph the last.
  - The CR-01 BOM fix goes in the D-18 indexer hook (`source_revision.go`).
  - `prov_extractor_version` bumps from `gortex-cobol-grammar/2` to `/3`.
  - A nested program colliding with an 01 record at the same path uses the `#k` fallback.
  - FETCH-via-cursor emits `reads` edges.

### Claude's Discretion
- **SQL non-physical sources (DET-12):** research picks the representation from what `queries/sql.scm` actually exposes (`sql_table_alias`, `sql_cte_definition`, `sql_dynamic_source`, `sql_include_name`, `sql_source_candidate`). Hard constraints: an alias, CTE, include, or dynamic source is **never merged into or represented as a physical `KindTable`**, and no table is invented for dynamic SQL. The suggested default: resolve aliases onto the physical table's edge as meta, make CTEs statement-scoped and marked `sql_role=cte`, treat `EXEC SQL INCLUDE` like COPY, and give dynamic SQL an `unresolved::` placeholder.
- The exact existing kind per construct under D-01, placeholder namespace strings, and meta key names (`cobol_*` vs `prov_*`, following the P2 conventions).
- The section and division containment kind and edge (`EdgeContains` vs `EdgeDefines`, matching existing language extractors).
- The plan slicing and wave order. Suggested: structure and data first, then control flow and calls/COPY, then IDMS, CICS, and SQL.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Parser handoff and producer (primary)
- `.planning/research/COBOL-PARSER-READINESS.md` — handoff fields → Gortex mapping (§3), boundary rules D10 (§4: the producer emits facts and gaps only; do not pin `ledger.configuration_id` / `gaps[].id`), producer limits (§6: memory, low concurrency), privacy (§7)
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade/grammar.js` — rule shapes: `call_statement` (target field `x` = `_id_or_lit_or_func`), `perform_procedure` (`label [THRU label]`), `goto_statement` (`to` repeat label, `DEPENDING ON`), `label` (`qualified_word` or `LITERAL [IN|OF LITERAL]`), `copy_statement`
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade/queries/{idms,cics,sql}.scm` — named node vocabulary for IDMS (`idms_record_name`, `idms_set_name`, navigation/update/session/accept statements), CICS (`cics_program_name`, `cics_transaction_name`, `cics_map_name`, `cics_mapset_name`), and SQL (`sql_table_name`, `sql_table_alias`, `sql_cte_definition`, `sql_dynamic_source`, `sql_include_name`, `sql_source_candidate`)
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/parsefacts/facts.go` — `Observation{Kind, Generated, Original, Parent, Affected}`. **Note: observations carry no tree-sitter field names.** Roles (CALL target, THRU end, GO TO targets) must be derived from child kinds and order.
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/handoff/testdata/consumer-v1.golden.json` — consumer contract fixture (assert structure, not IDs)

### Milestone contract
- `.planning/REQUIREMENTS.md` — DET-01..DET-12 (this phase), ID-03/ID-04 (Phase 4, but shaped by D-07/D-15), GAP-* (Phase 5 boundary, D-12), QUERY-* (Phase 6 consumers of this graph)
- `.planning/ROADMAP.md` — Phase 3 goal and success criteria 1–5
- `.planning/PROJECT.md` — constraints (fork hygiene, sequencing), Key Decisions
- `docs/ai-enhanced-cobol-graph-handoff.md` §6 (evidence classes), §12 (stable identity)

### Graph schema and prior phase
- `.planning/research/GRAPH-SCHEMA.md` — candidate concept mapping, layered identities, provenance vocabulary. Its "additive kinds" direction is **superseded by D-01** for this phase.
- `.planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-CONTEXT.md` — D-01..D-20 (provenance keys, ID grammar, degraded marking, copybook absence)
- `.planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/deferred-items.md` — CR-01 residual (BOM byte-offset bug, targeted at Phase 3), WR-04 (Phase 4)
- `.planning/research/PITFALLS.md`, `.planning/research/ARCHITECTURE.md`

### Prior regex baseline (behavior reference only, never re-registered)
- `internal/parser/languages/cobol.go` — what the regex extractor emitted (paragraphs, sections, divisions, COPY, CALL, the `dyncall` namespace, `COPY IDMS`). Use it as a parity checklist, not a source of truth.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/parser/languages/cobol_grammar.go` — `CobolGrammarExtractor.Extract` already runs `Analyze`, walks the program `Observations`, builds the `prov_*` meta, and stamps ranges with `cobolStampRange` (`prov_range_exact` / `prov_range_absence`). Breadth extends this walk rather than adding a new extractor.
- `internal/parser/languages/cobol_grammar_identity.go` — program ID and containment-path logic (`::OUTER/INNER#k`). D-07 data-item paths extend it.
- `graph.KindFunction`, `KindField`, `KindVariable`, `KindModule`, `KindTable`, `KindResource`; `EdgeCalls`, `EdgeContains`, `EdgeDefines`, `EdgeImports`, `EdgeReads`, `EdgeWrites`, `EdgeReferences` — all already exist (`internal/graph/node.go`, `edge.go`).
- The `unresolved::` placeholder convention, left unprefixed by `Indexer.applyRepoPrefix` and bound later by the resolver (`internal/resolver/`), is the binding seam for D-09 and D-11.

### Established Patterns
- Repository prefixing is applied by the indexer, not the extractor. Extractors emit repository-relative IDs.
- `graph.OriginASTResolved` / `OriginASTInferred` on edges. The `prov_origin` and `prov_confidence` contract comes from Phase 2.
- Copybooks (`.cpy`/`.CPY`) currently get a file node only (P2 D-20). D-08 may change this.

### Integration Points
- `internal/parser/languages/register.go:119` — the grammar extractor registration (unchanged).
- The resolver needs a COBOL-aware same-repo, unique-candidate rule for `cobol_program` and copybook placeholders. Check whether the existing resolver rules already enforce single-candidate binding before adding anything.
- The Neo4j projection (Phase 1) projects whatever kinds and meta land in SQLite, so new meta keys must avoid secret-like substrings (P2 D-20).
- Producer memory limit: parse one file at a time or at low concurrency (readiness §6, `cobolAnalyzeSlot`).

</code_context>

<specifics>
## Specific Ideas

- On statements, the user said: "I am feeling all statements… let research decide but I am leaning all."
- On copybooks, the user said: "research… if it can't do it let me know. I think it and/or the grammar may be able to." That is an explicit report-back gate, not permission to build a workaround.
- Accuracy over convenience remains the tiebreaker (carried from Phase 2).
- Phase 2's open flag (static vs dynamic CALL) is answered by the scout: the grammar's `call_statement` target is `_id_or_lit_or_func`, so a literal and an identifier are distinguishable by child node kind. Research must still confirm this through `Observations` (no field names).
- The regex era measured 829 dynamic versus 352 literal CALLs on the DCC corpus (PROJECT.md). Dynamic calls are the majority, so D-10's placeholder modeling matters.

</specifics>

<deferred>
## Deferred Ideas

- Implicit paragraph fall-through and terminator analysis (STOP RUN/GOBACK/EXIT/ALTER) — behavioral twin stage.
- Dynamic CALL value-flow resolution (for example, a single literal VALUE with no observed writes) — after Phase 3, under DET-04's deterministic-evidence rule.
- Copybook library search order and consuming a `cobol-repo-architect` retrieval manifest — Phase 4.
- Cross-repository CALL and COPY binding — Phase 4 resolution domains. Never by name alone.
- Merging qualified and `_unqualified` same-named DB2, IDMS, and CICS resources — Phase 4 (ID-04).

</deferred>

---

*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Context gathered: 2026-10-01*
