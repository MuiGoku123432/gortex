# Phase 3: Deterministic COBOL and Mainframe Breadth - Research

**Researched:** 2026-10-01
**Domain:** Mapping the enhanced COBOL grammar's handoff `Facts.Observations` onto Gortex graph nodes, edges, and resolver passes, covering structure, control flow, calls, copybooks, IDMS, CICS, and DB2.
**Confidence:** HIGH for observation shapes and the Gortex internals involved (probed against the pinned producer and read from source this session). MEDIUM for volume projections (measured on a public suite, not the private estate). LOW for the COBOL, IDMS, and CICS language semantics tagged `[ASSUMED]`.

> Privacy: every fixture, name, and count below is invented or comes from the public NIST COBOL-85 suite that ships in the fork's `test/` directory. No estate content, member names, private paths, or run IDs appear in this file.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Carried forward (Phase 2, do not re-decide)
- Grammar-only extraction; the regex `CobolExtractor` stays unregistered (P2 D-06, D-16). Every breadth fact comes from handoff `Observations`; names are read from `Generated[span]`; every persisted range is original coordinates via the handoff `Map` (P2 D-10).
- Provenance in flat `prov_*` Meta keys, no new `graph.Node` fields or SQLite columns (P2 D-08). Every new node and edge carries the same `prov_*` contract the program node has.
- IDs extend `<repo>/<path>::OUTER/INNER#k`; names as written (not upcased); line numbers never discriminate (P2 D-10, D-20).
- Amber/red documents and `Affected=true` observations are emitted and marked degraded, never dropped (P2 D-07).
- Invented COBOL fixtures only; the repo is public (P2 D-14).
- `prov_extractor_version` (`gortex-cobol-grammar/1`) must be bumped, because the emitted nodes and edges change (P2 02-03 note).

#### Graph vocabulary
- **D-01:** **Reuse existing node kinds plus `cobol_kind` meta. Add no new kinds.** Programs and paragraphs → `KindFunction`. Data items → `KindField`/`KindVariable`. Copybooks → `KindModule` or the existing file node. Sections and divisions → an existing containment kind. Each carries `cobol_kind=program|division|section|paragraph|statement|data_item|copybook`. Research picks the exact kind per construct, with existing query and analyzer compatibility as the tiebreaker. — **Reversibility:** costly — kinds are persisted in SQLite, projected as Neo4j labels, and relied on by Phase 4–6 queries; changing them needs a full reindex and reprojection.
- **D-02:** Paragraph control flow uses **`EdgeCalls` with `cobol_flow=perform|perform_thru|goto` meta**, so `get_call_chain` and `get_callers` trace paragraph execution without changes. — **Reversibility:** costly — same persistence and projection reach as D-01.
- **D-03:** `PERFORM A THRU C` **expands to one edge per paragraph from A through C** in source order within the owning program. All of those edges share the same statement occurrence, and each is marked `cobol_flow=perform_thru`. If C precedes A or either label is not found, emit no expansion and leave the reference unresolved with a reason (see D-12).
- **D-04:** **Explicit transfers only. No implicit fall-through edges.** `GO TO ... DEPENDING ON` emits one edge per listed target. Record each paragraph's source-order ordinal in meta so fall-through can be derived later.

#### Statement & data granularity
- **D-05:** Statement nodes: **the user leans toward ALL named statements as nodes**, contained by their paragraph. **Research makes the final call**, starting from "all". It must measure the node, edge, and SQLite/Neo4j projection volume impact (aggregate counts only, no estate content) and may narrow to evidence-bearing statements (CALL, PERFORM, GO TO, COPY, EXEC CICS, EXEC SQL, IDMS DML) only with demonstrated, unacceptable cost. Whatever the choice, every relationship edge must point back to its statement occurrence (range plus statement node ID where one exists).
- **D-06:** Data items: **everything**. That means levels 01–49, 66 (RENAMES), 77, 88 (condition names), and FILLER. The hierarchy comes from level nesting through containment edges. REDEFINES and RENAMES are relationships to the target item. Raw level, PIC, USAGE, OCCURS, VALUE, and the raw name are kept as meta.
- **D-07:** Data-item IDs use the **full group path from the 01 level**: `<owner>::PROG/REC-A/WS-DATE`, extending P2 D-10. A `#k` ordinal applies only to true same-path duplicates and to FILLERs. — **Reversibility:** costly — IDs are persisted and projected, and Phase 4 identity (ID-03/ID-04) builds on them.
- **D-08:** Copybook contents. Copybooks currently skip `Analyze` (`prov_analysis_absence=copybook_standalone_analysis_unsupported`). **Research first determines whether the fork's preprocessor and/or grammar can already parse a copybook (for example as a data-description fragment).** If it can, copybook data items are owned by the copybook node, not duplicated per including program. **If it cannot, research reports back to the user before any workaround is chosen.** Neither a Gortex-side synthetic wrapper nor a producer-side fork change gets planned without the user's decision. A fork change would be scheduled as a producer prerequisite, like P2 D-03.

#### Unresolved & ambiguous targets
- **D-09:** Literal `CALL 'PGMX'` emits `calls` → an `unresolved::` placeholder in a COBOL-program namespace. **The existing resolver binds it only to a COBOL program in the same repository scope, and only when exactly one candidate exists.** Zero candidates → reason `missing`. Two or more → reason `ambiguous`, with the candidate IDs in meta, and nothing gets picked. No cross-repository binding by name. Nested and contained programs in the same file resolve within the file first.
- **D-10:** Dynamic `CALL WS-PGM-NAME` emits a **distinct dyncall placeholder scoped to the calling program** (so same-named identifiers in different programs don't merge), plus `cobol_call=dynamic` and a reference edge to the identifier's data-item node. **It is never bound in Phase 3** (DET-04). Literal calls carry `cobol_call=literal`. CICS `LINK`/`XCTL PROGRAM('X')` with a literal uses the same literal-call path. With an identifier it uses the dyncall path.
- **D-11:** `COPY X` resolution: **bind only to a copybook in the same repository whose member name matches exactly once.** The case and extension normalization rule is versioned and recorded. Zero matches → reason `missing`. Two or more → reason `ambiguous`, with candidate IDs in meta, and nothing gets picked. `COPY ... OF/IN <library>` and `COPY IDMS [kind] name` qualifiers are recorded as written. Library search order is Phase 4.
- **D-12:** **Phase 3 / Phase 5 line.** Phase 3 emits each reference edge to its placeholder with a normalized `cobol_unresolved_reason` (`missing | ambiguous | dynamic | label_not_found`, and research may extend the set). It does not create finding nodes and leaves the handoff `Gaps[]` untouched. Phase 5 builds the `PARSE_UNRESOLVED` / `EXTERNAL_UNRESOLVED` findings on top. Nothing that can't be resolved is dropped.

#### Mainframe resource modeling
- **D-13:** IDMS records and sets and CICS transactions, maps, and mapsets use **`KindResource` with `resource_type=idms_record|idms_set|cics_transaction|cics_map|cics_mapset`**. DB2 physical tables reuse **`KindTable`**. CICS programs are COBOL-program call targets (D-10), not resources. — **Reversibility:** costly — persisted and projected like D-01.
- **D-14:** Edges are classified by verb and reuse existing kinds: IDMS `OBTAIN`/`FIND`/`GET` → `reads`. `STORE`/`MODIFY`/`ERASE`/`CONNECT`/`DISCONNECT` → `writes`. CICS `RECEIVE MAP` → `reads`, `SEND MAP` → `writes`. Transaction references (`START`/`RETURN TRANSID`) → `references`. SQL `SELECT`/`FETCH`/cursor reads → `reads`, `INSERT`/`UPDATE`/`DELETE`/`MERGE` → `writes`. The raw verb or command is always kept in meta (`idms_verb`, `cics_command`, `sql_verb`). Research supplies the complete verb table, and any verb it cannot classify honestly becomes `references`.
- **D-15:** Resource identity is **repository-scoped, with qualifiers taken only from source**. Examples: `<repo>::db2/<SCHEMA|_unqualified>/TABLE`, `<repo>::idms/<subschema|_unqualified>/RECORD`, `<repo>::cics/mapset/NAME`. An unknown qualifier is an explicit `_unqualified` marker, never a guess. Qualified and unqualified same-named resources stay distinct until Phase 4 resolution domains can justify a merge. — **Reversibility:** costly — the IDs are persisted and projected, and Phase 4 ID-04 depends on them.

### Claude's Discretion
- **SQL non-physical sources (DET-12):** research picks the representation from what `queries/sql.scm` actually exposes (`sql_table_alias`, `sql_cte_definition`, `sql_dynamic_source`, `sql_include_name`, `sql_source_candidate`). Hard constraints: an alias, CTE, include, or dynamic source is **never merged into or represented as a physical `KindTable`**, and no table is invented for dynamic SQL. The suggested default: resolve aliases onto the physical table's edge as meta, make CTEs statement-scoped and marked `sql_role=cte`, treat `EXEC SQL INCLUDE` like COPY, and give dynamic SQL an `unresolved::` placeholder.
- The final statement granularity under D-05.
- The exact existing kind per construct under D-01, placeholder namespace strings, and meta key names (`cobol_*` vs `prov_*`, following the P2 conventions).
- The section and division containment kind and edge (`EdgeContains` vs `EdgeDefines`, matching existing language extractors).
- The plan slicing and wave order. Suggested: structure and data first, then control flow and calls/COPY, then IDMS, CICS, and SQL.

### Deferred Ideas (OUT OF SCOPE)
- Implicit paragraph fall-through and terminator analysis (STOP RUN/GOBACK/EXIT/ALTER) — behavioral twin stage.
- Dynamic CALL value-flow resolution (for example, a single literal VALUE with no observed writes) — after Phase 3, under DET-04's deterministic-evidence rule.
- Copybook library search order and consuming a `cobol-repo-architect` retrieval manifest — Phase 4.
- Cross-repository CALL and COPY binding — Phase 4 resolution domains. Never by name alone.
- Merging qualified and `_unqualified` same-named DB2, IDMS, and CICS resources — Phase 4 (ID-04).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DET-01 | Source-positioned program, division, section, paragraph, statement, and data-item observations from named AST nodes | Observation shapes probed (§Role derivation). Kind map (§Kind and edge map). ID grammar (§ID grammar). Statement granularity measured (§D-05). Extents for flat headers (Pitfall 1). |
| DET-02 | Paragraph control flow for PERFORM, THRU, GO TO | `perform_procedure → label [THRU label]` and `goto_statement → label* [qualified_word]` confirmed by probe. Label-resolution rules (§Control flow). Edge dedupe (§Edge identity). |
| DET-03 | Literal calls separate from dynamic identifiers | The first child of `call_statement` is `string`/`number` (literal) or `qualified_word`/`function_` (identifier). Confirmed by probe. |
| DET-04 | Dynamic identifiers stay unresolved | `unresolved::cobol::dynamic::…` namespace, never bound. The resolver leaves the `cobol::` namespace untouched (§Resolver). |
| DET-05 | Copybook/include references keep member name and occurrence | `copy_statement` (WORD/string book, WORD/string library), `-INC` rewrite, `COPY IDMS`, `exec_sql_include`. Text-based form detection (§Role derivation). |
| DET-06 | Copybook resolution keeps ambiguity | New `resolveCobolReferences` pass: same repo, exactly-one, candidate IDs on ambiguity (§Resolver). |
| DET-07 | IDMS record references | `idms_record_name` under navigation/update/session/accept statements (§Verb table). |
| DET-08 | IDMS set references | `idms_set_name`. `idms_scope_name` stays a placeholder (set or area) (§Verb table). |
| DET-09 | CICS program and transaction references | `cics_program_name`, `cics_transaction_name`, each a literal (`string`) or a data name (`WORD`) (§Verb table). |
| DET-10 | CICS map and mapset references | `cics_map_name`, `cics_mapset_name` (§Verb table). |
| DET-11 | SQL reads and writes to physical tables | `sql_source_candidate` parent decides write vs read. Table text comes from the `sql_table_name` span (§SQL). |
| DET-12 | Aliases, CTEs, includes, dynamic sources stay distinguishable | Alias → edge meta. CTE names subtracted per statement into statement meta. INCLUDE → COPY path. PREPARE → dynamic placeholder (§SQL non-physical representation). |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

Extracted from `./CLAUDE.md`, `./.claude/CLAUDE.md`, and the global engineering standards. They have the same authority as locked decisions:

- **Build/test:** `go build -o gortex ./cmd/gortex/` needs CGO. `go test -race ./...` must pass.
- **Simplicity:** single-use code stays inline. Extract only on the third use (rule of three). No speculative generality, managers, factories, or plugin layers. YAGNI.
- **Reuse first:** before adding a function or type, check whether one already exists. Extend it instead of forking it.
- **File size:** aim for < 300 lines per module, one responsibility per file, following the neighbouring layout (`internal/parser/languages/cobol_grammar*.go`).
- **Dependencies:** no new dependency, pattern, or layer without written justification and confirmation. **This phase needs no new dependency.**
- **Fork hygiene:** prefer additive files over invasive upstream edits. Each upstream edit (resolver, indexer) must be minimal and justified.
- **Comments:** every exported function gets a doc comment. Use the PURPOSE/RATIONALE/KEYWORDS style for non-obvious code.
- **Errors:** wrap with `%w`. Never swallow an error silently.
- **Privacy (public repo):** invented COBOL only. Never put estate names or paths in code, tests, or docs.
- **GSD workflow:** edits happen inside `/gsd-execute-phase`.
- **Organization (Uniti):** OWASP Top 10 practices. Parameterized SQL only (this phase writes no SQL text; it uses Store APIs). No credentials in code. Flag SOX/CPNI exposure (none found: the estate stays local, and only aggregate counts ever leave the developer machine).

## Summary

The handoff has every structural fact Phase 3 needs, but in a **flatter shape than the CONTEXT assumes**. A probe built against the exact pinned producer (`preprocessor@v0.0.0-20260930215433-f9eaf99c34a9`, fork HEAD `f9eaf99`) showed four things:
1. `paragraph_header` and `section_header` are **siblings** of the statements under `procedure_division`, not containers.
2. `data_description` entries are **flat siblings** under `working_storage_section`, so hierarchy has to come from level numbers.
3. Paragraph and section **names are not observations**: `field('name', choice($._WORD, $.integer))` uses a hidden token. The name comes from the header's own `Generated` text.
4. `copy_statement` is a grammar *extra*, so its `Parent` is whatever node it happens to sit in.

Every role the CONTEXT asked about can be derived from child kind plus order, with **no fork change**. The CALL target is the first child: `string`/`number` is a literal, `qualified_word` is an identifier. The THRU end is the second `label` under `perform_procedure`. GO TO DEPENDING shows up as a direct `qualified_word` child. `IN/OF` qualification is the second `WORD` in a `qualified_word`. There are two exceptions that need a `Generated`-text check rather than kinds alone:
- `COPY IDMS kind name` versus `COPY name OF lib`. Both have two `WORD` children.
- SQL table names. Underscore identifiers are hidden tokens, so `MYSCH.AUDIT_LOG` yields only the `WORD` `MYSCH`.

**D-08 finding: a standalone copybook CANNOT be parsed today.** The grammar's start rule accepts only `program_definition`, and `Analyzer.Analyze` has no fragment mode. An invented data-description copybook grades red, with one `ERROR` gap and no `data_description` observations. The options are laid out in the section below. **The user must decide before any copybook-contents work is planned.**

**D-05 recommendation: ALL named statements as nodes.** On the public NIST suite (382 files, 398 programs, 24.6 MB), "all" means 142,470 statement nodes against 59,585 evidence-bearing ones. Total COBOL nodes go from about 159k to about 242k (×1.52). That growth is linear and not disproportionate. The real storage cost is the full `prov_*` contract cloned onto every node (about 1.4 KB each), and it applies equally to both options. Use `graph.KindLocal` for statements: it is excluded from BM25 search, it is not a dead-code candidate, and it attaches through `member_of` with no `defines` edge. That keeps the cost of "all" contained.

**The resolver does not enforce D-09/D-11 today.** `resolveFunctionCall` takes the *first* same-file candidate, then falls back to "the first same-repo function/method match", and `CrossRepoResolver` lifts bare names across repos. The minimal change has three parts:
- an owned placeholder namespace `unresolved::cobol::…`;
- four one-line exclusions (`resolveEdge`, `identifierFromTarget`, `reuseIdentifier`, `CrossRepoResolver.resolveEdge`);
- one whole-graph pass, `resolveCobolReferences`, placed next to `resolveLuaRequires` in `ResolveAllContext`.

**Primary recommendation:** Start with a tracer slice: paragraph nodes plus one PERFORM edge, run through the real indexer, SQLite, and `get_callers`. It establishes a shared observation-walk helper and the `@proc` ID grammar. Then fan out to structure, data, control flow, calls/COPY with the resolver pass, IDMS/CICS, and SQL. Fix CR-01 in parallel inside the existing D-18 indexer hook.

## USER DECISION REQUIRED: Copybook contents

**Finding (VERIFIED by probe this session):** the pinned producer cannot parse a standalone copybook without a workaround or a fork change.

- The grammar's start rule is `start: $ => repeat( choice( $.program_definition, ...` (fork `grammar.js:36-41`). No data-description or procedure-fragment entry exists.
- `Analyzer.Analyze(ctx, sourceID, original)` has no options. Its only constructors are `NewAnalyzer(repoRoot, catalog)` and `NewEmbeddedAnalyzer(catalog)` (fork `preprocessor/analyze.go`). Nothing in the preprocessor mentions copybooks or fragments.
- Probe: the invented copybook `01 CPY-REC. 05 CPY-A PIC X(4). 05 CPY-B PIC 9(2).` gave `grade=red reasons=[missing-required-anchor division-scale-recovery error-node affected-capture]`, 5 observations (`start`, one `level_number`, three `comment_entry`, all `Affected`), and one `ERROR` gap covering the body. No `data_description` was observed. A procedure-fragment copybook behaved the same way.
- `COPY` is **not expanded** in programs either. The catalog only models Endevor `-INC` membership, and `-INC` lines are rewritten symbolically to `COPY member.` So the program-side data hierarchy never contains copybook-supplied items.

Phase 3 can satisfy DET-01..DET-12 **without** copybook contents. DET-01 covers data items in programs, and DET-05/06 cover references and resolution only. What is lost without contents: data items declared in copybooks, `REDEFINES`/`RENAMES` targets and dynamic-CALL identifiers that live in copybooks (they stay unresolved with reason `not_declared_in_program`), and copybook-owned data identity.

| Option | What it is | Accuracy / coordinates | Cost and risk |
|---|---|---|---|
| **A. Producer copybook mode, no grammar change** | Fork adds an entry such as `AnalyzeCopybook(ctx, id, src)` that wraps the member in a synthetic `IDENTIFICATION DIVISION. PROGRAM-ID. <synthetic>. DATA DIVISION. WORKING-STORAGE SECTION.` header. The wrapper is recorded as `sourcemap.KindInsert` segments, so the real bytes keep exact original coordinates and `OriginalContentID` stays the member's own bytes. | High for data-description members. Gortex's `cobolStampRange` already marks synthetic projections (`prov_range_absence=synthetic_projection`). | Fork change, then a user-approved push and a Gortex re-pin of `cobolApprovedPreprocessorModule` (grammar ID unchanged). Grade policy needs a copybook-aware anchor rule. Procedure-fragment members need a second wrapper variant. `REPLACING` pseudo-text members (`==:TAG:==`) still fail: the probe showed that pseudo-text triggers an `ERROR` cascade even inside a program. Schedule it as a producer prerequisite, like P2 D-03. |
| **B. Producer grammar change** | Add a fragment alternative to `start` (for example a `copybook_body` of data descriptions). | High. Coordinates are exact natively. | Regenerates the grammar, so the grammar ID changes. That means re-attesting and re-pinning both modules and re-running the fork's v0.26.0 acceptance (touches P2 D-01 baseline). Parse-table conflict risk. Most invasive. |
| **C. Gortex-side synthetic wrapper** | Gortex prepends wrapper bytes before `Analyze` and subtracts the offsets afterwards. | Fragile. The handoff's `Map`, `OriginalContentID`, and grade describe the *wrapped* bytes, not the file, which breaks the D-09 revision identity and PROV-03 exactness unless every coordinate is re-based by hand. The wrapper must itself be valid fixed-format. | No fork change, but it invents coordinates on the Gortex side, against readiness §4 ("never invent coordinates"). Not recommended. |
| **D. Defer (status quo)** | Copybooks stay file nodes only (`cobol_kind=copybook`). COPY edges still bind to them. | Honest: absence is explicit. | Zero. Program-side analysis stays incomplete until a later phase. |

Do not pick a workaround. Planning for copybook **contents** waits on this answer. Planning for copybook **references and resolution** (DET-05/06) does not.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|---|---|---|---|
| Parse, source map, gaps, grading | Producer (fork `preprocessor`) | — | Readiness §4 / D10: the producer emits facts and gaps only. |
| Observation → nodes/edges, IDs, in-file label and data resolution | Extractor (`internal/parser/languages/cobol_grammar*.go`) | — | All of the evidence is inside one handoff, and it is deterministic. |
| Repo prefixing, BOM coordinate correction, VCS revision | Indexer (`internal/indexer`, D-18 hook) | — | Only the indexer knows the root path and the raw bytes. |
| Cross-file binding (literal CALL → program, COPY → copybook) | Resolver (`internal/resolver`, new whole-graph pass) | — | Needs the whole repo's graph. Same repo, exactly one candidate. |
| Persistence, edge identity | Graph store (SQLite `AddBatch`) | In-memory graph | `UNIQUE(from_id, to_id, kind, file_path, line, view_gen)`. |
| Query surface (`get_callers`, `get_call_chain`, `get_symbol`) | Query/MCP (unchanged) | CLI | D-02 relies on the existing `calls` traversal. |
| Neo4j labels and properties | `internal/neo4jprojection` (unchanged) | — | Projects kinds and meta. Meta key names must avoid secret-like substrings. |

## Standard Stack

### Core (all already in `go.mod`; nothing new)

| Library | Version | Purpose | Why Standard |
|---|---|---|---|
| `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor` | `v0.0.0-20260930215433-f9eaf99c34a9` | Handoff producer | Pinned by P2 D-02: `cobolApprovedPreprocessorModule = cobolPreprocessorModulePath + "@v0.0.0-20260930215433-f9eaf99c34a9"` [VERIFIED: internal/parser/languages/cobol_grammar.go:39; probe built against the module cache] |
| `github.com/alexaandru/go-sitter-forest/cobol` → fork `forest-shim/cobol` (replace) | same pseudo-version | Enhanced grammar | P2 D-02: `replace github.com/alexaandru/go-sitter-forest/cobol => github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol v0.0.0-20260930215433-f9eaf99c34a9` [VERIFIED: go.mod:375] |
| `internal/graph`, `internal/resolver`, `internal/indexer` | in-repo | Kinds, edges, binding, prefixing | Existing seams. |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions | Already used by `cobol_grammar_test.go`. |

**Installation:** none. **Version verification:** not applicable. No package is added, removed, or bumped, unless the user picks copybook Option A or B, which would re-pin the fork pseudo-versions.

### Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---|---|---|---|---|---|---|
| (none) | — | — | — | — | — | No external packages are installed in this phase. |

**Packages removed due to [SLOP] verdict:** none. **Packages flagged [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
COBOL file bytes ──► indexer transforms (BOM strip) ──► CobolGrammarExtractor.Extract
                                                           │
                                     .cpy? ──yes──► file node only (cobol_kind=copybook)   [D-08 gate]
                                                           │ no
                                                           ▼
                                        preprocessor.Analyze (cobolAnalyzeSlot, 1 at a time)
                                                           │ handoff.Handoff
                                                           ▼
                        ┌──────────── shared observation walk (children index, text(i)) ────────────┐
                        ▼                    ▼                     ▼                    ▼            ▼
                 programs (P2)     divisions/sections/      data items (levels,   refs: CALL/COPY/   mainframe:
                                   paragraphs/statements    REDEFINES/RENAMES)    INCLUDE/dynamic    IDMS/CICS/SQL
                                   + label resolution
                                   + PERFORM/THRU/GO TO
                        └──────────────────────────────┬─────────────────────────────────────────────┘
                                                       ▼ ExtractionResult (repo-relative IDs, prov_* meta,
                                                         edges dedup by From/To/Kind/Line)
                     applyRepoPrefix ──► applyCoverageDomains (D-18: VCS stamp + CR-01 BOM byte shift)
                                                       ▼
                                           SQLite AddBatch (INSERT OR IGNORE edges)
                                                       ▼
                         Resolver.ResolveAllContext ──► resolveEdge: "cobol::" left untouched
                                                       └► resolveCobolReferences (whole graph):
                                                            program/copybook placeholders → bind if
                                                            exactly 1 same-repo candidate, else reason
                                                       ▼
                                       get_callers / get_call_chain / get_symbol / Neo4j projection
```

### Recommended file layout (all `//go:build darwin || linux`, next to the existing files)

```
internal/parser/languages/
├── cobol_grammar.go             # Extract: existing; gains calls into the walkers below, version → /3
├── cobol_grammar_identity.go    # existing program symbols (P2); unchanged
├── cobol_grammar_walk.go        # shared: children index, text(i), owning program, stamp helpers (used by every walker)
├── cobol_grammar_procedure.go   # divisions, sections, paragraphs, statements, labels, PERFORM/THRU/GO TO
├── cobol_grammar_data.go        # data items, level nesting, REDEFINES/RENAMES
├── cobol_grammar_refs.go        # CALL, COPY/-INC/INCLUDE, dynamic placeholders, data-item references
├── cobol_grammar_mainframe.go   # IDMS + CICS
└── cobol_grammar_sql.go         # EXEC SQL tables/aliases/CTE/cursor/dynamic
internal/resolver/cobol_references.go   # new whole-graph binding pass (+ 1-line edits in 4 files)
internal/indexer/source_revision.go     # CR-01: BOM byte shift inside the existing D-18 hook
```
The walk helper is used by five or more walkers, so extracting it satisfies the rule of three. Each walker file has one responsibility.

### Role derivation from Observations (research question 3)

Every row below was **confirmed by running the pinned producer** on invented fixtures. Observations carry only named nodes. Anonymous keywords (`TO`, `OF`, `DEPENDING`, `INTO`) never appear, but named keyword rules (`THRU`, `MAP`, `PROGRAM`, `TRANSID`, IDMS verbs) do.

| Construct | Observed shape (child kinds in span order) | Derivation | Fork change? |
|---|---|---|---|
| Paragraph / section name | `paragraph_header` text `MAIN-PARA.`; `section_header` text `MAIN-SECT SECTION.`; numeric `0200.` | Name is hidden (`_WORD`). Use the first whitespace-delimited token of `text(header)`, minus the trailing `.`. | No |
| Paragraph membership | headers and statements are **siblings** under `procedure_division` | A statement belongs to the nearest preceding header within the same `procedure_division`. Before any header, it belongs to the procedure division itself. | No |
| CALL literal vs identifier | `call_statement` → first child `string` (`'SUBPGM'`, `"DQPGM"`) or `number` → literal; `qualified_word` → identifier; `function_` → identifier-like | Kind of the **first** child. `call_param_arg` children come after it. | No |
| CALL identifier qualification | `qualified_word` → `WORD`(`WS-PGM`) `WORD`(`WS-REC`) for `WS-PGM OF WS-REC` | First WORD is the data name, the rest are qualifiers, innermost first. | No |
| PERFORM target / THRU end | `perform_statement_call_proc` → `perform_procedure` → `label`, [`THRU`, `label`]; `THROUGH` also yields kind `THRU` | First label = start, second label = THRU end. | No |
| PERFORM with options | `perform_option` is a sibling of `perform_procedure` (`3 TIMES`, `UNTIL …`) | Ignore it for edge purposes. | No |
| Inline PERFORM | `perform_statement_loop` (no `perform_procedure`) | No target. Statement node only. | No |
| GO TO targets | `goto_statement` → `label`+ | One target per label. | No |
| GO TO … DEPENDING ON | direct child `qualified_word` after the labels (not wrapped in `label`) | A direct `qualified_word` child means DEPENDING ON. | No |
| Alterable `GO TO.` | `goto_statement` with no children | No edge. `cobol_goto_alterable=true`. | No |
| Label `IN/OF` section | `label` → `qualified_word` → `WORD`(para) `WORD`(section); numeric labels: `label` → `number` [`number`] | Second element = section qualifier. | No |
| COPY member / library | `copy_statement` → `WORD`\|`string` (member) [`WORD`\|`string` (library)] | First = member, second = library (`OF`/`IN` is anonymous). | No |
| `COPY IDMS [kind] name` | `copy_statement` → `WORD` [`WORD`] (`COPY IDMS RECORD EMPLOYEE` → `RECORD`,`EMPLOYEE`) | **Kinds alone are ambiguous with `COPY X OF L`.** Test `text(copy)` against `^COPY[ \t]+IDMS[ \t]` (ASCII case-fold) — the `_COPY_IDMS` token. If it matches, two WORDs are kind + name and one WORD is the name. | No (text check) |
| Endevor `-INC member` | Generated holds `COPY INCMEM.`; the `copy_statement` original projection has `exact=false` | `cobol_copy_form=inc` when the original bytes at `Original.Start.Byte` start with `-INC` | No |
| `EXEC SQL INCLUDE m` | `exec_sql_include` → `EXEC`,`SQL`,`WORD`(INCLUDE),`sql_include_name`→`WORD` | Treat as COPY with `cobol_copy_form=sql_include` | No |
| Data entry | `data_description` → `level_number`, `entry_name` (text `FILLER` for filler), clauses | Flat. Nesting comes from levels (§Data items). | No |
| PIC / USAGE | `picture_clause` → `picture_x`/`picture_9`/`picture_a`/`picture_n`/`picture_edit`; `usage_clause` → e.g. `COMP_3`, `COMP` | Raw text of the child. | No |
| OCCURS | `occurs_clause` → `integer` [`integer`] [`qualified_word`] [keys] | Two integers = min/max (`TO` is hidden). A `qualified_word` = DEPENDING ON. | No |
| VALUE | `value_clause` → `value_item`+ → literal [`THRU` literal] | Raw text of each item. | No |
| REDEFINES / RENAMES | `redefines_clause` → `qualified_word`; `renames_clause` → `qualified_word` [`qualified_word`] | Two qualified words = `RENAMES A THRU B`. | No |
| IDMS verb | first child: `FIND`/`OBTAIN`/`STORE`/`MODIFY`/`ERASE`/`CONNECT`/`DISCONNECT`/`GET`/`BIND`/`READY`/`FINISH`/`COMMIT`/`ROLLBACK`/`ACCEPT` | Kind of the first child. | No |
| IDMS operands | `idms_record_name`, `idms_set_name`, `idms_area_name`, `idms_scope_name` (WITHIN set-or-area) | One role per node kind. | No |
| IDMS subschema qualifier | `DB <ss> WITHIN <schema>.` lines are neutralized to comments by the transform. No named node. | Always `_unqualified` | Yes, to expose it (optional; Phase 4) |
| CICS command | `exec_cics_statement` → `EXEC`,`CICS`,`WORD`(command), … | First `WORD` child after `CICS`. | No |
| CICS operands | keyword node (`PROGRAM`/`TRANSID`/`MAP`/`MAPSET`) then `cics_*_name` → `string` (literal) or `WORD` (data name) | Child kind decides literal vs dynamic. | No |
| SQL verb | `exec_sql_statement` → `EXEC`,`SQL`, then one of `sql_select_query`, `sql_cte_clause`+`sql_select_query`, `sql_insert_statement`, `sql_update_statement`, `sql_delete_statement`, `sql_merge_statement`, `sql_create/alter/drop_table_statement`, `sql_prepare_statement`, `sql_declare_cursor_statement`, or a bare `WORD` (fallback: OPEN, FETCH, CLOSE, EXECUTE, COMMIT, …) | First `WORD` of the subnode, else the first `WORD` after `SQL`. | No |
| SQL table name | `sql_source_candidate` → `sql_table_name` [`sql_table_alias`→`WORD`] | **Use `text(sql_table_name)`, not its WORD children**: `MYSCH.AUDIT_LOG` yields only `WORD` `MYSCH` because `_sql_underscore_identifier` is hidden. Split on `.` outside `"…"`. | No (text) |
| SQL write vs read | candidate's parent is `sql_insert/update/delete/merge_statement` → target (write); parent `sql_source_reference` (FROM/JOIN) or `sql_using_reference` → source (read) | Parent kind. | No |
| CTE | `sql_cte_definition` → `sql_cte_name`→`WORD`, `sql_select_query`; the main select then names the CTE as an ordinary `sql_table_name` | Subtract same-statement CTE names before treating a name as physical. | No |
| Dynamic SQL | `sql_prepare_statement` → `WORD`(PREPARE),`WORD`(stmt),`WORD`(FROM),`sql_dynamic_source`→`WORD`(host var) | `EXECUTE IMMEDIATE :x` has **no** named source node (the tail is a gap). | No; known producer limit |
| Cursor | `sql_declare_cursor_statement` → `WORD`(DECLARE),`WORD`(cursor),…,`sql_select_query`; `FETCH C1` is fallback, so the cursor is a `WORD` child of the statement (inside a tail gap) | Program-local cursor → DECLARE match. | No |

**Producer limitations seen (not role gaps; Phase 5 owns them as gaps):** aliases without `AS` (`FROM CUST C`) stay in `sql_unparsed_tail`. `COPY … REPLACING ==:P:== BY ==X==` and `ENTRY 'X'` each caused an `ERROR` cascade that turned an invented program red. Every EXEC SQL statement produced `unparsed-tail` gaps, so SQL-bearing programs grade **amber**. `sql_unparsed_tail`/`cics_unparsed_tail`/`idms_unparsed_tail` are gaps, but the `WORD` nodes *inside* them are still observations, parented to the enclosing statement.

### Kind and edge map (research question 4)

| Construct | `graph.Kind` | `cobol_kind` | Structural edges | Why this kind (compatibility tiebreak) |
|---|---|---|---|---|
| Program | `KindFunction` (P2) | `program` | file `defines` → program (P2) | Unchanged. |
| Division (identification/environment/data/procedure) | `KindPackage` | `division` (+`cobol_division=data|procedure|…`) | `defines` from file; `member_of` → program | `KindPackage` is the namespace container that C++/C#/Erlang/PHP/Rust extractors already use. It is in `neverDeadCodeKinds` and excluded from the tier-0 search shortcut. `KindType` would be a dead-code candidate and would show up in class hierarchy. |
| Data-division section (WORKING-STORAGE, LINKAGE, FILE, LOCAL-STORAGE) | `KindPackage` | `section` (+`cobol_section_kind=working_storage|linkage|…` from the observation kind) | `defines`; `member_of` → data division | Same as above. |
| Procedure section | `KindFunction` | `section` | `defines`; `member_of` → procedure division | `PERFORM <section>` is a legal target [ASSUMED], so sections must be callable for `get_callers`. |
| Paragraph | `KindFunction` | `paragraph` (+`cobol_ordinal`) | `defines`; `member_of` → section or procedure division | D-01/D-02. |
| Statement | `KindLocal` | `statement` (+`cobol_statement_kind=<observation kind>`) | **`member_of` → container only** (no `defines`) | `KindLocal` is excluded from BM25 (`shouldIndexForSearch`), excluded from get_file_summary definitions, and is not a dead-code candidate. Go locals attach the same way (`member_of` only). `KindClosure` would imply callable semantics. See Assumption A1. |
| Data item 01 / 77 | `KindVariable` | `data_item` | `defines`; `member_of` → data section | D-01. Dead-code for variables is opt-in, so there is no noise. |
| Data item 02–49, 66, 88, FILLER | `KindField` | `data_item` | `defines`; `member_of` → parent item | `KindField` + `EdgeMemberOf` is the documented "member of owner" convention. |
| Copybook | existing `KindFile` | `copybook` (file meta) | — | `KindModule` means external `(ecosystem,name,version)` packages (`module::<eco>:<name>@<ver>`). Reusing it would mislead module analyzers. |
| DB2 table | `KindTable` (+meta `dialect=db2`, `schema`, `table`) | — | relationship edges only | The meta keys match those the existing orphan/unreferenced-table analyzers read. |
| IDMS record/set, CICS transaction/map/mapset | `KindResource` + `resource_type` | — | relationship edges only | D-13. `KindResource` is in `neverDeadCodeKinds`. |

Kind and edge constants confirmed this session: `KindFile NodeKind = "file"`, `KindPackage NodeKind = "package"`, `KindFunction NodeKind = "function"`, `KindVariable NodeKind = "variable"` [VERIFIED: internal/graph/node.go:27-33]; `KindField NodeKind = "field"` [VERIFIED: internal/graph/node.go:44]; `KindLocal NodeKind = "local"` [VERIFIED: internal/graph/node.go:76]; `KindModule NodeKind = "module"`, `KindTable NodeKind = "table"` [VERIFIED: internal/graph/node.go:107-111]; `KindResource NodeKind = "resource"` [VERIFIED: internal/graph/node.go:176]; `EdgeContains EdgeKind = "contains"`, `EdgeDefines EdgeKind = "defines"` [VERIFIED: internal/graph/edge.go:25-26]; `EdgeCalls EdgeKind = "calls"` [VERIFIED: internal/graph/edge.go:27]; `EdgeReferences EdgeKind = "references"` [VERIFIED: internal/graph/edge.go:31]; `EdgeMemberOf  EdgeKind = "member_of"` [VERIFIED: internal/graph/edge.go:38]; `EdgeReads  EdgeKind = "reads"`, `EdgeWrites EdgeKind = "writes"` [VERIFIED: internal/graph/edge.go:101-102]; `OriginASTResolved = "ast_resolved"` [VERIFIED: internal/graph/edge.go:698]. `EdgeImports EdgeKind = "imports"` is the existing kind for COPY (regex era) [VERIFIED: internal/graph/edge.go:6].

**Containment edge choice:** use `defines` (file → symbol) plus `member_of` (child → parent). This is what the Go and Java extractors do (`java.go:768-774` emits file `EdgeDefines` and method `EdgeMemberOf` → class). `EdgeContains` is documented as "file node to its non-symbol children (imports)", so it is not the right fit.

**Relationship edges, all From = the innermost procedure container (paragraph > section > program node for statements before any header; program node for data-division EXEC SQL):**

| Fact | Edge | To | Key meta |
|---|---|---|---|
| PERFORM X | `calls` | paragraph/section ID, or label placeholder | `cobol_flow=perform` |
| PERFORM A THRU C | `calls` ×N | each procedure A..C | `cobol_flow=perform_thru`, `cobol_thru_start`, `cobol_thru_end` |
| GO TO X / DEPENDING | `calls` | each target | `cobol_flow=goto`, `cobol_goto_depending=true`, `cobol_depending_positions=[…]` |
| CALL 'P' / LINK/XCTL literal | `calls` | `unresolved::cobol::program::<NORM>` | `cobol_call=literal`, `cobol_target_raw`, `cics_command` (CICS) |
| CALL ident / LINK/XCTL ident | `calls` | `unresolved::cobol::dynamic::program::<callerProgramID>::<IDENT>` | `cobol_call=dynamic`, `cobol_unresolved_reason=dynamic` |
| identifier → data item | `references` | data item ID, or `unresolved::cobol::data::<programID>::<NAME>` | `cobol_ref_role=call_target|cics_program|…` |
| COPY / -INC / SQL INCLUDE | `imports` | `unresolved::cobol::copybook::<NORM>` (then bound to a copybook file node) | `cobol_copy_form=copy|inc|idms|sql_include`, `cobol_copy_member_raw`, `cobol_copy_library`, `cobol_copy_idms_kind`, `cobol_copy_norm` |
| REDEFINES / RENAMES | `references` | target data item, or `unresolved::cobol::data::…` | `cobol_ref_role=redefines|renames_from|renames_thru` |
| IDMS / CICS / SQL | `reads`/`writes`/`references`/`calls` | resource / table / placeholder | `idms_verb`, `cics_command`, `sql_verb`, `sql_alias(es)` |

Every relationship edge carries the full `prov_*` contract plus `cobol_statement_symbol` (the statement's QualName, so `edge.FilePath + "::" + symbol` is the statement node ID in both single- and multi-repo modes, Pitfall 5), the statement occurrence range (`prov_start_row`…`prov_end_byte`, `prov_range_exact`), and `prov_affected`. `Edge.Line` = the statement's 1-based original start line. `Origin=OriginASTResolved` and `Confidence=1.0` for all extractor facts. The fact is deterministic even when its target is an unresolved placeholder.

### ID grammar (extends P2 D-10 and D-07; Claude's discretion for non-data segments)

COBOL user-defined words use only letters, digits, and hyphens [ASSUMED]. So a segment starting with `@` cannot collide with a data or program name. D-07 locks data paths **unmarked** directly under the program, so procedure-side constructs take a marked segment instead:

| Construct | Repo-relative ID (indexer prepends `<repo>/`) |
|---|---|
| Program (P2) | `<path>::OUTER/INNER#k` |
| Division | `<path>::<prog>/@<observation kind>`, e.g. `…::DEMOPGM/@procedure_division` |
| Data-division section | `<path>::<prog>/@working_storage_section` |
| Data item (D-07, locked) | `<path>::<prog>/WS-REC/WS-DATE/WS-YY`; FILLER → `…/FILLER#k` (k = 1-based among FILLER siblings); a true duplicate → `#k` for k ≥ 2 |
| Procedure section | `<path>::<prog>/@proc/MAIN-SECT` |
| Paragraph | `<path>::<prog>/@proc/[<SECT>/]MAIN-PARA` (`#k` for same-path duplicates, k ≥ 2) |
| Statement | `<container ID>/@<observation kind>` then `#k` for k ≥ 2 among same-kind statements in that container, e.g. `…/@proc/MAIN-SECT/MAIN-PARA/@call_statement#2` |
| DB2 table | `db2::<SCHEMA\|_unqualified>/<TABLE>` |
| IDMS | `idms::<SUBSCHEMA\|_unqualified>/record/<NAME>`, `idms::<…>/set/<NAME>` |
| CICS | `cics::transaction/<NAME>`, `cics::mapset/<NAME>`, `cics::map/<MAPSET\|_unqualified>/<MAP>` |

The resource IDs follow Gortex's `<scheme>::…` convention for shared, non-file nodes: `KindTable` documents ``db::<dialect>::<schema>.<table>`` [VERIFIED: internal/graph/node.go:108-110] and `KindResource` documents ``k8s::<kind>::<namespace>::<name>`` [VERIFIED: internal/graph/node.go:170-171]. They keep D-15's components. D-15's `<repo>::` comes from `applyRepoPrefix`, which writes `<repo>/` in front of every node ID. IDMS gets an extra `record`/`set` segment because a record and a set may share a spelling [ASSUMED]. Line numbers and byte offsets never enter an ID.

**Known collision the planner must surface (Open Question 2):** D-07's unmarked data path and P2's unmarked nested-program path share the namespace under `<prog>/`. A nested program `INNER` inside `OUTER` and an `01 INNER` record in `OUTER` would both be `…::OUTER/INNER`. That is rare but legal [ASSUMED]. The P2/D-07 `#k` rule would give the second one `#2`, which is deterministic but mixes kinds.

### Name normalization (versioned, recorded as meta; needs confirmation, see Open Question 1)

Node names and IDs stay **as written** (P2 D-20). Matching and shared-resource IDs need a canonical form, because every reference to a shared resource spells it independently. The recommended rule is `cobol-name-norm/1`:
- COBOL user-defined words (paragraphs, data names, IDMS, COPY members): ASCII upper-fold.
- Quoted literals (CALL, CICS operands, COPY): strip one level of quotes, un-double embedded quotes, then upper-fold. Strip a trailing registered copybook extension (`.cpy`) for COPY.
- SQL ordinary identifiers: upper-fold. Delimited `"…"` identifiers: kept verbatim with quotes.
- Record `cobol_name_norm=cobol-name-norm/1` on every edge and resource node that uses it.

Upper-folding literal program names assumes the PGMNAME(COMPAT) behaviour, which is compiler-option dependent [ASSUMED].

### Resolver: current behaviour and the minimal D-09/D-11 change (research question 4)

**Current behaviour (VERIFIED this session):**
- `applyRepoPrefix` leaves `unresolved::` targets unprefixed ("Unresolved targets carry no prefix", indexer.go:2316-2319, read via sed). `UnresolvedMarker = "unresolved::"` [VERIFIED: internal/graph/stub.go:142].
- `resolveEdge` handles bare names with `resolveFunctionCall`. That function filters candidates to the caller's repo, then binds the **first** same-file match even when there are several, then runs locality cascades, and finally: "Fall back to the first same-repo function/method match … tag it text_matched" [VERIFIED: internal/resolver/resolver.go:4046-4063]. Names are compared case-sensitively. **This violates D-09's exactly-one rule.** The regex-era `unresolved::PGMX` literal-call edges went through exactly this path.
- `CrossRepoResolver.resolveEdge` routes non-`import::`/`*.` targets to its own `resolveFunctionCall` and may stitch across repos (cross_repo.go:1087-1125, read via sed). That violates "no cross-repo binding by name".
- Owned namespaces already exist as precedent. `resolveEdge` returns untouched for `grpc::` and `pyrel::`: `case strings.HasPrefix(target, "grpc::"): … return oldTo, false` / `case strings.HasPrefix(target, "pyrel::"): … return oldTo, false` [VERIFIED: internal/resolver/resolver.go:3362-3376]. Whole-graph passes (`resolveRelativeImports`, `resolveLuaRequires`) then bind them inside `ResolveAllContext` (resolver.go:1651-1660).
- The identifier extractors treat `import::`, `pyrel::`, and `grpc::` as owned and return `""`. Quoting `identifierFromTarget`: `case strings.HasPrefix(target, "import::"), strings.HasPrefix(target, "pyrel::"), strings.HasPrefix(target, "grpc::"): … return ""` [VERIFIED: internal/resolver/lsp_resolve.go:517-522]. `reuseIdentifier` has the same case list [VERIFIED: internal/indexer/incremental_resolve.go:450-453]. `identifierFromTarget` feeds terminal stamping, LSP, cross-repo seeding, and repo-language lookup.

**Minimal change:**
1. Use `unresolved::cobol::` for **every** COBOL placeholder: `program::`, `copybook::`, `dynamic::`, `label::`, `data::`, `idms_scope::`.
2. Add a `strings.HasPrefix(target, "cobol::")` case that returns untouched in `Resolver.resolveEdge` (next to `grpc::`/`pyrel::`) and in `CrossRepoResolver.resolveEdge`. Add `cobol::` to `identifierFromTarget` and `reuseIdentifier` so it is never classified terminal or reused.
3. Add a new file `internal/resolver/cobol_references.go` containing `resolveCobolReferences()`, called right after `resolveLuaRequires()` in `ResolveAllContext`. It:
   - returns early unless `r.graphHasLanguage("cobol")`;
   - indexes COBOL program nodes (`KindFunction`, `Meta["cobol_kind"]=="program"`) by `(RepoPrefix, normName)`, and copybook file nodes by `(RepoPrefix, normBasenameWithoutExt)` (via `graph.FileLanguageNodesSeq` + node meta);
   - for each pending edge with a `unresolved::cobol::program::` or `unresolved::cobol::copybook::` target, scopes candidates to the caller repo (`r.callerRepoPrefix(e)`). For programs, it narrows to same-file candidates first when any exist.
   - **Exactly 1:** sets `e.To` and appends `graph.EdgeReindex{Edge: e, OldTo: oldTo}`.
   - **0:** stamps `cobol_unresolved_reason=missing`.
   - **≥2:** stamps `ambiguous` and `cobol_candidate_ids` (sorted).
   - persists with `r.graph.ReindexEdges(batch)` for To changes, and with `graph.EdgeMetaBatchPersister.PersistEdgeAttributesBatch` for meta-only stamps (the persister the terminal pass uses, terminal.go:285-289).
   - leaves `dynamic::`, `label::`, `data::`, and `idms_scope::` alone (never bound in Phase 3).
4. Leave `unresolved::cobol::idms_copy` / `COPY IDMS` bound or unbound per Open Question 3.

`EdgeImports` with `unresolved::cobol::copybook::` never reaches `resolveImport`, so it never becomes an `external::` stub that module attribution would pick up (module attribution only scans `external::` imports, module_attribution.go:53-54).

**Incremental path caveat:** whole-graph passes run only in `ResolveAllContext`. `resolveRelativeImports` has the same limitation. A single-file reindex leaves new COBOL placeholders unbound until the next full resolve. That is Phase 4 (lifecycle). Record it, don't fix it here.

### Unresolved reason vocabulary (D-12, extended)

| Reason | Set by | When |
|---|---|---|
| `missing` | resolver | No same-repo candidate (program/copybook). |
| `ambiguous` | resolver or extractor | ≥2 candidates (program, copybook, label, data name); `cobol_candidate_ids` lists them. |
| `dynamic` | extractor | Identifier-valued CALL/CICS program/transaction/map/mapset, PREPARE FROM host var. |
| `label_not_found` | extractor | PERFORM/GO TO label not in the owning program. |
| `thru_range_invalid` (new) | extractor | THRU end precedes start, or start/end are different procedure kinds. No expansion (D-03). |
| `not_declared_in_program` (new) | extractor | Data name not among the program's observed items (often copybook-supplied). |
| `scope_role_unknown` (new) | extractor | IDMS `WITHIN <scope>`: grammar cannot tell set from area. |
| `cursor_not_declared` (new) | extractor | FETCH/OPEN/CLOSE cursor with zero or ≥2 same-program DECLAREs. |

### Control flow: label resolution rules (extractor, program-local)

These are COBOL procedure-name rules [CITED: ibm.com/docs/en/SS6SG3_6.5/lr/ref/rlrefqua.html — paragraph names are unique within a section, and the section name is the only qualifier]:
1. `X OF S` / `X IN S`: the paragraph `X` in section `S` of the same program. Otherwise `label_not_found`.
2. Unqualified `X`: the paragraph `X` in the **referencing statement's own section**, if present. Otherwise the unique section or paragraph named `X` in the program. Two or more → `ambiguous`. Zero → `label_not_found`.
3. Procedure names never resolve across programs (nested programs included) [ASSUMED]. Each `procedure_division` belongs to its parent `program_definition`. Every `program_definition` hangs off `start`, as P2 found.
4. THRU expansion (D-03): over the program's procedure headers in source order, **same kind as the endpoints**. Paragraph endpoints expand over every paragraph from A to C inclusive, crossing section boundaries. Section endpoints expand over sections. Mixed kinds, or C before A → `thru_range_invalid` and no expansion. Every expanded edge carries the same `cobol_statement_symbol`.
5. `cobol_ordinal` = 1-based source order of procedure headers (sections and paragraphs, one sequence per program) for later fall-through (D-04).

### Data items: level nesting

Walk `data_description` observations in order within each data-division section:
- `01`/`77` reset the stack. 77 is standalone (`KindVariable`).
- `02`–`49`: pop while `top.level >= level`. The parent is the new top.
- `66`: parent = the current 01 record (bottom of the stack) [ASSUMED: RENAMES belongs to the preceding record].
- `88`: parent = the immediately preceding non-88 entry, without popping.
- Compare levels numerically (`5` == `05`).
- FILLER gets `#k`. Duplicate same-path names get `#k` (k ≥ 2).

Meta: `cobol_level`, `cobol_name_raw`, `cobol_pic`, `cobol_usage`, `cobol_occurs_min/max`, `cobol_occurs_depending`, `cobol_value` (list of raw item texts), `cobol_redefines`, `cobol_renames`. REDEFINES/RENAMES targets resolve by the same qualified-name lookup as dynamic-CALL identifiers (innermost WORD plus `OF` qualifiers matched against ancestors).

**COPY inside a group breaks the hierarchy silently.** In `01 WS-REC. COPY X. 05 Y`, the copybook's items are absent, so `Y` nests under `WS-REC` correctly but the gap is invisible. Mark the group with `cobol_contains_copy=true`.

### D-14 complete verb table (research question 5)

**IDMS** (`idms_verb` = first-child kind):

| Verb | Record operand | Set operand | Scope (`WITHIN x`) | Notes |
|---|---|---|---|---|
| OBTAIN, FIND | `reads` | `reads` | `references` → `unresolved::cobol::idms_scope::_unqualified/<NAME>` (`scope_role_unknown`) | D-14 locked. FIND only establishes currency without moving data [ASSUMED]. The raw verb is kept. |
| GET | `reads` | — | — | Grammar places GET under `idms_update_statement`. Classify by verb, not by node kind. |
| STORE, MODIFY, ERASE | `writes` | — | — | `ERASE … ALL MEMBERS` → `idms_erase_scope` meta. |
| CONNECT, DISCONNECT | `writes` | `writes` | — | Both operands change. |
| BIND | `references` | — | — | `BIND RUN-UNIT` has no operand. |
| ACCEPT … FROM rec CURRENCY | `references` | — | — | Reads currency (db-key), not record data [ASSUMED]. |
| READY | — | — | — | `idms_area` meta on the statement. No resource (not in D-13). |
| FINISH, COMMIT, ROLLBACK | — | — | — | Statement node only. |

**CICS** (`cics_command` = first `WORD` after `CICS`; operand literal = `string` child, otherwise dynamic):

| Command + operand | Edge | Target |
|---|---|---|
| LINK / XCTL + PROGRAM | `calls` | literal → `unresolved::cobol::program::`; identifier → dynamic placeholder + `references` → data item (D-10) |
| any other command + PROGRAM (LOAD, RELEASE, HANDLE ABEND, INQUIRE, SET, …) | `references` | same program placeholder namespace |
| any command + TRANSID (START, RETURN, CANCEL, …) | `references` | `cics::transaction/<NAME>` or a dynamic placeholder |
| SEND + MAP | `writes` | `cics::map/<MAPSET\|_unqualified>/<MAP>` |
| RECEIVE + MAP | `reads` | same |
| other command + MAP | `references` | same |
| any + MAPSET | `references` | `cics::mapset/<NAME>` |
| anything else | — | statement node + `cics_command` only (FILE/QUEUE operands are not named nodes) |

When MAPSET is omitted, the map stays `_unqualified`. No defaulting rule is applied (D-15).

**SQL** (`sql_verb` = first `WORD` of the statement subnode or fallback):

| Verb | Edges |
|---|---|
| SELECT (incl. SELECT INTO, nested and derived queries, CTE bodies) | `reads` → each physical source candidate |
| DECLARE … CURSOR FOR SELECT | `reads` → its sources; `sql_cursor` meta |
| OPEN / FETCH / CLOSE c | `reads` → the tables of the unique same-program `DECLARE c`, with `sql_via_cursor=c`. Otherwise statement only + `cursor_not_declared`. |
| INSERT INTO t / UPDATE t / DELETE FROM t | `writes` → t |
| MERGE INTO t USING s | `writes` → t, `reads` → s |
| CREATE / ALTER / DROP TABLE t | `references` → t (DDL is not a data read or write) |
| PREPARE s FROM :hv | `references` → `unresolved::cobol::dynamic::sql::<programID>::<hv>` (`dynamic`) + `references` → data item hv |
| EXECUTE / EXECUTE IMMEDIATE | statement only, `sql_dynamic=true` (no named source node) |
| INCLUDE m | `imports` → copybook placeholder (`cobol_copy_form=sql_include`); `sql_include_builtin=true` for SQLCA/SQLDA [CITED: ibm.com/docs/SSEPEK_12.0.0/sqlref/src/tpc/db2z_sql_include.html — `INCLUDE SQLCA \| SQLDA \| member-name`] |
| everything else (COMMIT, ROLLBACK, LOCK, SET, WHENEVER, CONNECT, DECLARE TABLE) | statement node only |

### SQL non-physical representation (research question 6, Claude's discretion)

- **Alias:** `sql_alias` on the edge to the physical table. If one statement references the same table twice (self-join), collect `sql_aliases=[…]` on the deduped edge. Derived-table aliases (`(SELECT …) AS D`) are `sql_derived_aliases` on the statement node and never a table.
- **CTE:** statement-scoped. Statement node meta `sql_ctes=[names]`, `sql_cte_refs=[names]`. **No node and no edge.** Physical tables inside the CTE body still get `reads`. A same-statement CTE name shadows a physical table of the same name.
- **Include:** COPY path, so it is never a `KindTable`.
- **Dynamic:** dynamic placeholder (never bound) plus a data-item reference. No table is invented.

This satisfies DET-12 with no new node types and no fake tables.

### Edge identity (research question 7)

Both stores key an edge on `(From, To, Kind, FilePath, Line)`: `type edgeKey struct { From string; To string; Kind EdgeKind; FilePath string; Line int }` [VERIFIED: internal/graph/graph.go:72-82]; `UNIQUE(from_id, to_id, kind, file_path, line, view_gen)` [VERIFIED: internal/graph/store_sqlite/schema.go:871]. They handle a duplicate differently. **SQLite uses `INSERT OR IGNORE INTO edges` (the first one wins and later meta is dropped)** (add_batch_set.go:436, read via grep). **The in-memory graph replaces the slot in place (the last one wins)** (graph.go:2750-2776). So:
- Two PERFORMs of the same paragraph on **different lines** stay distinct.
- THRU expansion edges share From and Line but have different `To`, so they stay distinct.
- **They collapse when they share From, To, Kind, and Line:** `PERFORM P-ONE. PERFORM P-ONE.` on one line (observed in the edge-case probe), and `GO TO A B A DEPENDING ON X` (a repeated target in one statement). Without handling, which occurrence's meta survives depends on the backend.
- **Fix in the extractor:** dedupe by `(From, To, Kind, Line)` before returning. Keep the first occurrence's meta, append later statement symbols to `cobol_statement_symbols` (list), set `cobol_occurrence_count`, and for DEPENDING keep `cobol_depending_positions`. That makes the outcome deterministic on both backends.
- Node IDs (statement `#k` ordinals, FILLER `#k`) are unique by construction. Resource nodes are **meant** to collide across files (shared ID). Follow the `go_sql.go` precedent: emit per file with that file's `FilePath` ("first sighting; not authoritative"), deduped inside one result.

### D-05 statement granularity: measurement and recommendation (research question 1)

The private corpus was **not reachable** through environment variables (`PREPROCESSOR_CORPUS_ROOT` is unset). The measurement used the **public NIST COBOL-85 suite** (`test/cobol85` in the fork) and the pinned producer, with aggregate counts only.

| Measure (382 files, 398 programs, 24,560,344 bytes; all graded red only for `missing-required-anchor`, with 9 `error-node`) | Count |
|---|---|
| Observations | 1,406,278 |
| Divisions / procedure sections / paragraphs / data items | 1,581 / 1,478 / 37,254 / 58,885 |
| Statements: the 45 named `_statement` kinds (fork grammar.js:1384-1430, excluding the `ERROR` alias) + `perform_statement_loop` | **142,470** (median 291/file, p90 846, max 1,871) |
| Evidence-bearing statements (CALL, PERFORM, GO TO, EXEC CICS/SQL/INCLUDE, IDMS DML) | 59,585 (PERFORM 40,746, GO TO 18,721, CALL 118) |
| COBOL nodes, "all" vs "evidence" | ~242,066 vs ~159,181 (×1.52) |
| Structural edges (symbols `defines`+`member_of`, statements `member_of` only) | ~341,700 vs ~258,800 (+82,885) |
| Full `prov_*` meta per node (sample, JSON) | ~1,430 bytes |

Per program: about 608 nodes ("all") against about 400 ("evidence"). At the readiness doc's ~1,369-program scale that is roughly 830k against 550k nodes. Node meta is about 1.2 GB against 0.8 GB, before edge rows.

**Recommendation: ALL named statements.** The extra cost is a bounded, linear ×1.5. The dominant cost, the cloned prov contract, is paid under both options. All of these keep "all" affordable:
- `KindLocal`, so there is no BM25/FTS row or dead-code noise.
- `member_of` only, so there is one edge per statement, not two.
- Structural edges (`defines`/`member_of`) carry `Origin`/`Confidence` but **no `prov_*` Meta**, as the P2 program `defines` edge already does (Assumption A2). The full contract goes on nodes and relationship edges.

Neo4j projection scales linearly (one node per statement, ~30 properties each), well under the 10,000-element / 1 MiB per-entity metadata budget (`maxMetadataElements = 10_000`, `maxMetadataBytes = 1 << 20`, neo4jprojection/properties.go:19-20, read via grep).

Optional, before execution: the developer can rerun the scratch probe's aggregate mode against `$PREPROCESSOR_CORPUS_ROOT` locally (it prints counts only) to confirm the ratio on the real estate. That is not required to plan.

### CR-01 residual: BOM byte offsets (research question 8)

**Cause (VERIFIED):** the indexer's always-on `bomStripTransform` (`matches(string) bool { return true }`, and `stripBOM` returns `src[3:]` for `0xEF 0xBB 0xBF`) [VERIFIED: internal/indexer/transform.go:278-291] runs before extraction at 8 call sites (`file_delta.go:137`, `indexer.go:3577`, and others). The extractor never sees the BOM, so `prov_start_byte`/`prov_end_byte` are relative to the stripped bytes. You cannot fix this inside the extractor, and the preprocessor cannot take the BOM either: its fixed-format zone check rejects a leading UTF-8 BOM (fork `transform/fixed_test.go:189`, "leading UTF-8 BOM disagrees").

**Fix location:** the existing D-18 hook. `applyCoverageDomains` → `stampSourceRevision(relPath, src, result)` (indexer.go:1721) already compares `src` with the on-disk file: `if disk, err := os.ReadFile(...); err != nil || !bytes.Equal(disk, src) { return "", "indexed_bytes_differ_from_file" }` [VERIFIED: internal/indexer/source_revision.go:170-171]. Add a small sibling in `source_revision.go`, called from the same hook. If the disk bytes equal `EF BB BF` + `src`, add 3 to `prov_start_byte`/`prov_end_byte` on every node **and edge** whose Meta carries them. Leave rows and columns alone: Gortex line and column coordinates are BOM-excluded for every language, which matches editor semantics (Assumption A6). For any other mismatch (command transform, UTF-16), set `prov_range_exact=false` and `prov_range_absence=indexed_bytes_differ_from_file` on byte-bearing entries rather than claiming file offsets. One `os.ReadFile` per COBOL file. Test with an invented BOM-prefixed program through the real indexer, because the extractor alone cannot observe the BOM.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| COBOL parsing or keyword scanning | regexes over source (the regex era) | handoff `Observations` + `text(i)` on spans | Comments, continuations, and sequence areas. The regex era matched 2,242 of 3,768 COPYs in comments (cobol.go comment). |
| Original coordinates | arithmetic on `Generated` offsets | `Observation.Original` (`sourcemap.Projection`) + existing `cobolStampRange` | Rewrites, inserts, and `-INC` make generated and original offsets diverge. |
| Edge persistence and dedupe | custom SQL | `ExtractionResult` → `AddBatch`; `ReindexEdges` / `PersistEdgeAttributesBatch` | Generation, view, and FTS sidecars are store-managed. |
| Cross-file binding | name matching inside the extractor | resolver whole-graph pass | The extractor sees one file. The resolver owns persistence and scope. |
| Repo scoping | building `<repo>/` into extractor IDs | `applyRepoPrefix` | Extractors emit repo-relative IDs (P2). |
| Search and dead-code exclusion of statements | new skip-config entries | `KindLocal` (already excluded) | Already enforced in `shouldIndexForSearch` and `isNonDefinitionNode`. |

**Key insight:** every hard problem in this phase (coordinates, recovery, grading, persistence, binding) already has an owner. Phase 3 is a mapping layer plus one resolver pass.

## Runtime State Inventory

This phase changes what the COBOL extractor emits, so existing stores hold stale shapes.

| Category | Items Found | Action Required |
|---|---|---|
| Stored data | SQLite stores indexed under `gortex-cobol-grammar/2` hold file and program nodes only. Older stores may still hold regex-era COBOL nodes (P2 D-15). | Full reindex after deploy (code change only; no migration script). Bump the version to `/3`. **Note:** the code says `cobolGrammarExtractorVersion = "gortex-cobol-grammar/2"` [VERIFIED: internal/parser/languages/cobol_grammar.go:42], not `/1` as CONTEXT states. |
| Live service config | Neo4j projections built from P2 stores | Reproject after reindex (Phase 1 command). No config change. |
| OS-registered state | None. Verified: no daemons or tasks embed COBOL shapes. | None |
| Secrets / env vars | `COBOL_PARSER_READ_PAT` (CI). Unchanged unless the copybook Option A/B re-pins the fork. | None, or re-pin only |
| Build artifacts | Module cache holds the pinned fork pseudo-versions. Verified: the probe built with `GOWORK=off GOPROXY=off`. | None, unless D-08 A/B |

## Common Pitfalls

### Pitfall 1: Treating headers as containers
**What goes wrong:** paragraph nodes get one-line ranges and no statements, because `paragraph_header` spans only `MAIN-PARA.`.
**Why:** the procedure division is flat in this grammar.
**How to avoid:** derive membership from position. The paragraph extent runs from the header's original start to the original end of the last observation before the next header in the same `procedure_division`. Combine projections only when both are exact and non-synthetic. Otherwise stamp only the header and set `prov_range_absence=paragraph_extent_unmapped`. This needs a two-projection variant of `cobolStampRange`, because the existing one takes a single `*sourcemap.Projection`.
**Warning signs:** `get_symbol_source` on a paragraph shows one line.

### Pitfall 2: Reading names from child WORDs
**What goes wrong:** paragraph names are missing (the hidden `_WORD`), and SQL tables with underscores get truncated (`MYSCH.AUDIT_LOG` → `MYSCH`).
**How to avoid:** read names from the span text of `paragraph_header`, `section_header`, and `sql_table_name`, not from child WORDs.

### Pitfall 3: Trusting `Parent` for COPY
**What goes wrong:** COPY appears under `program_definition`, `data_division`, or `start` depending on where the extra lands (the probe saw all three).
**How to avoid:** the owning program = the nearest `program_definition` ancestor through `Parent`. If there is none, From = the file node. Use span order for "where in the data hierarchy".

### Pitfall 4: Backend-dependent edge collapse
See §Edge identity. Always dedupe in the extractor.

### Pitfall 5: Node IDs inside Meta are not repo-prefixed
**What goes wrong:** `applyRepoPrefix` rewrites only `Node.ID`, `Edge.From/To`, and `FilePath`. A meta value such as `cobol_statement_id="cobol/x.cbl::P/@proc/A/@call_statement"` would not equal the real (prefixed) ID in multi-repo mode.
**How to avoid:** store `cobol_statement_symbol` (the QualName only). Consumers join on `edge.FilePath + "::" + symbol`. Resolver-written `cobol_candidate_ids` are already full IDs, because the resolver sees prefixed nodes.

### Pitfall 6: COBOL placeholders leaking into generic passes
**What goes wrong:** without the `cobol::` exclusions, terminal stamping marks them `no_definition`, LSP and cross-repo passes probe them, and a bare `unresolved::NAME` would be bound by the "first same-repo match" fallback.
**How to avoid:** use the four one-line exclusions plus the owned namespace. Never emit bare `unresolved::<NAME>` from COBOL.

### Pitfall 7: Shared resource nodes and file eviction
**What goes wrong:** a `KindTable`/`KindResource` node shared across files has the `file_path` of its last writer (SQLite upsert sets `file_path=excluded.file_path`). Reindexing that file evicts the node while other files' edges still point at it, and if the file no longer references it, it is not re-added. This already happens for `go_sql` tables.
**How to avoid:** accept it in Phase 3 and document it. Phase 4 (lifecycle and ID-04) owns shared-node ownership.

### Pitfall 8: Dead-code and caller noise from fall-through
**What goes wrong:** paragraphs reached only by fall-through (and the first paragraph) have no incoming `calls`, so `analyze dead_code` reports them as dead.
**How to avoid:** this is expected under D-04. Record `cobol_ordinal` so Phase 6 queries can derive fall-through. Do not add implicit edges.

### Pitfall 9: Literal names with ID metacharacters
**What goes wrong:** a literal `PROGRAM-ID. 'a/b::unresolved::c'` or `CALL 'x#2'` produces IDs that `IsUnresolvedTarget` (`Contains "::unresolved::"`) or `/`/`#` parsing would misread.
**How to avoid:** percent-encode `:`, `/`, `#`, `%`, and control bytes in **literal-derived** ID segments only. Keep the raw form in `cobol_*_raw` meta. COBOL words cannot contain these characters [ASSUMED].

### Pitfall 10: IDMS COPY bound to an ordinary copybook
**What goes wrong:** `COPY IDMS EMPLOYEE` copies from the IDMS dictionary, not a source library [ASSUMED]. Binding it to `EMPLOYEE.cpy` by name could be a false resolution.
**How to avoid:** see Open Question 3. Default to a separate unbound form.

## Code Examples

These are skeletons for the planner. Identifiers marked as existing were read this session. Everything else is a proposal.

### Shared walk helper (proposal; `cobol_grammar_walk.go`)
```go
// cobolWalk indexes one handoff's observations once. Used by every breadth walker.
type cobolWalk struct {
	h        handoff.Handoff
	children [][]int // observation index -> child indexes in span order
}

func newCobolWalk(h handoff.Handoff) *cobolWalk {
	w := &cobolWalk{h: h, children: make([][]int, len(h.Facts.Observations))}
	for i, o := range h.Facts.Observations { // Parent < i is guaranteed by parsefacts.Validate
		if o.Parent >= 0 {
			w.children[o.Parent] = append(w.children[o.Parent], i)
		}
	}
	return w
}

// text returns the generated bytes of observation i (names come from Generated, P2 D-10).
func (w *cobolWalk) text(i int) string {
	g := w.h.Facts.Observations[i].Generated
	return string(w.h.Generated[g.Start:g.End])
}
```

### Procedure headers and PERFORM targets (proposal)
```go
for _, c := range w.children[procDiv] {
	o := w.h.Facts.Observations[c]
	switch o.Kind {
	case "section_header", "paragraph_header":
		name := strings.TrimSuffix(strings.Fields(w.text(c))[0], ".") // hidden _WORD: read header text
		// open a new container; statements that follow belong to it
	case "perform_statement_call_proc":
		for _, p := range w.children[c] {
			if w.h.Facts.Observations[p].Kind != "perform_procedure" {
				continue
			}
			var labels []int
			for _, l := range w.children[p] {
				if w.h.Facts.Observations[l].Kind == "label" {
					labels = append(labels, l)
				}
			}
			// labels[0] = start; len(labels)==2 → labels[1] = THRU end
		}
	}
}
```

### Resolver dispatch exclusion (existing switch in `resolveEdge`, one new case)
```go
case strings.HasPrefix(target, "grpc::"):  // existing [VERIFIED resolver.go:3362]
	return oldTo, false
case strings.HasPrefix(target, "cobol::"): // new: owned by resolveCobolReferences
	return oldTo, false
```

### Binding pass shape (proposal; follows `resolveLuaRequires`)
```go
func (r *Resolver) resolveCobolReferences() {
	if !r.graphHasLanguage("cobol") {
		return
	}
	// build (repo, norm) -> []id for programs and copybooks, then:
	var batch []graph.EdgeReindex
	var stamped []*graph.Edge
	for e := range r.graph.EdgesWithUnresolvedTarget() {
		// filter unresolved::cobol::program:: / ::copybook::; scope r.callerRepoPrefix(e)
		// exactly one -> oldTo := e.To; e.To = id; batch = append(batch, graph.EdgeReindex{Edge: e, OldTo: oldTo})
		// zero / many -> e.Meta["cobol_unresolved_reason"] = "missing"|"ambiguous"; stamped = append(stamped, e)
	}
	if len(batch) > 0 {
		r.graph.ReindexEdges(batch)
	}
	if p, ok := r.graph.(graph.EdgeMetaBatchPersister); ok && len(stamped) > 0 {
		p.PersistEdgeAttributesBatch(stamped)
	}
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| Regex `CobolExtractor`: divisions `KindType`, sections `KindMethod`, `unresolved::NAME` calls, `unresolved::import::` COPY, `unresolved::dyncall::ID` | Grammar handoff: owned `unresolved::cobol::` namespace, `KindPackage` divisions, `KindFunction` sections/paragraphs, `KindLocal` statements | Phase 2 (program only) → Phase 3 (breadth) | The regex file stays unregistered and serves as a parity checklist only. |
| Bare-name resolver binding (first candidate) | Exactly-one, same-repo COBOL pass | Phase 3 | D-09/D-11 semantics. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|---|---|---|
| A1 | `KindLocal` is acceptable for statement nodes, despite the doc's "intra-function binding" wording | Kind map | Persisted kind (costly). Alternative is `KindClosure` (callable semantics) or meta-only statements. **Confirm with the user.** |
| A2 | Structural `defines`/`member_of` edges may omit `prov_*` Meta, as the P2 program edge does, while nodes and relationship edges carry the full contract | D-05 | If the user reads "every edge" literally, storage roughly doubles for statements. **Confirm.** |
| A3 | COBOL user-defined words use only letters, digits, and hyphens, so `@`-prefixed segments cannot collide | ID grammar | ID collision. IBM 6.x also allows `_`, which is still safe. |
| A4 | `PERFORM <section>` is legal, and procedure names never resolve across nested programs | Control flow | Missing or wrong edges. |
| A5 | Upper-fold matching (`cobol-name-norm/1`) for program, copybook, and resource names | Normalization | Over- or under-merging. PGMNAME(LONGMIXED) shops are case-sensitive. |
| A6 | Rows and columns stay BOM-excluded and only byte offsets are shifted | CR-01 | Column mismatch on row 0 if the user wants raw-file columns. |
| A7 | IDMS FIND does not move data; ACCEPT CURRENCY reads only the db-key; `COPY IDMS` comes from the dictionary | Verb table, Pitfall 10 | Misclassified edges. The raw verb is always kept. |
| A8 | Level-66 belongs to the preceding 01 record, and 88 to the preceding non-88 entry | Data items | Wrong containment for 66/88. |
| A9 | NIST suite ratios (×1.52) are representative of the estate | D-05 | The cost estimate is off. Rerun the aggregate probe locally to confirm. |

## Open Questions (RESOLVED)

1. **Name normalization rule (`cobol-name-norm/1`).** Upper-folding is needed for shared resource IDs and for D-09/D-11 matching. CONTEXT locks "names as written" for node names and IDs and "qualifiers only from source". Recommendation: keep node names and IDs as written for definitions, and use folded canonical IDs only for shared resources and for matching. Record the rule version. Confirm before the resource plans. **RESOLVED:** CONTEXT D-18 (user-confirmed, `cobol-name-norm/1` = ASCII uppercase; display names as written).
2. **Program-versus-data path collision** (`OUTER/INNER` nested program vs `01 INNER`). Recommendation: accept the `#k` fallback for Phase 3 and log it for Phase 4 ID-03. Alternatively, amend D-07 with a data marker such as `@data`, which is costly to reverse. **RESOLVED:** the `#k` fallback was adopted as a research default (CONTEXT post-research confirmations).
3. **`COPY IDMS` binding.** D-11 records the qualifiers but does not say whether IDMS-form copies bind to `.cpy` files. Recommendation: use a separate unbound placeholder `unresolved::cobol::copybook::idms/<KIND|_>/<NAME>` with reason `idms_dictionary_member` (one more reason value). Confirm. **RESOLVED:** CONTEXT D-19 (user-confirmed, never bound, reason `idms_dictionary_member`).
4. **FETCH-via-cursor `reads`.** Recommended (deterministic and program-local). The simpler alternative attributes reads only at DECLARE, which loses the paragraph context when DECLARE sits in WORKING-STORAGE. **RESOLVED:** adopted as a research default (CONTEXT post-research confirmations).
5. **D-08 copybook contents.** See the USER DECISION section. It blocks only the copybook-contents plan. **RESOLVED:** CONTEXT D-16 (user chose Option A, a fork copybook mode).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| Go toolchain | build/test | ✓ | go1.27.1 (go.mod `go 1.27.0`) | — |
| CGO + C compiler | tree-sitter | ✓ | `CGO_ENABLED=1`, `/usr/bin/cc` | — |
| Pinned fork modules | grammar/preprocessor | ✓ (module cache; `GOPRIVATE` covers the fork) | `v0.0.0-20260930215433-f9eaf99c34a9` | — |
| git | VCS stamp, tracer test fixtures | ✓ | Homebrew git | — |
| Fork checkout | only for D-08 options A/B | ✓ (HEAD `f9eaf99`, matches the pin) | — | — |
| Private corpus env (`PREPROCESSOR_CORPUS_ROOT`) | optional volume confirmation | ✗ (unset) | — | Public NIST suite (used) |

**Missing dependencies with no fallback:** none. Existing `TestCobolGrammar*` tests pass locally (`ok … 2.887s`).

## Validation Architecture

### Test Framework

| Property | Value |
|---|---|
| Framework | Go `testing` + testify v1.11.1 |
| Config file | none (build tags `darwin \|\| linux` on COBOL files) |
| Quick run command | `go test -race ./internal/parser/languages/ -run 'TestCobolGrammar' -count=1` |
| Resolver run | `go test -race ./internal/resolver/ -run 'TestCobol' -count=1` |
| Indexer (BOM) run | `go test -race ./internal/indexer/ -run 'TestCobol\|TestStampSourceRevision' -count=1` |
| Acceptance run | `go test -race ./cmd/gortex/ -run 'TestCobolTracer' -count=1` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|---|---|---|---|---|
| DET-01 | Divisions/sections/paragraphs/statements/data items with exact ranges, kinds, IDs, `cobol_kind` | unit | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_Structure -count=1` | ❌ Wave 0 |
| DET-01 | Level nesting, FILLER/88/66/77, REDEFINES/RENAMES meta and edges | unit | `… -run TestCobolGrammar_DataItems` | ❌ |
| DET-02 | PERFORM, THRU expansion (incl. invalid range), GO TO, DEPENDING, IN/OF, label_not_found, same-line dedupe | unit | `… -run TestCobolGrammar_ControlFlow` | ❌ |
| DET-02 | `get_callers` / `get_call_chain` over paragraphs through real indexer + SQLite + MCP | integration | `go test -race ./cmd/gortex/ -run TestCobolTracer_ParagraphFlow -count=1` | ❌ |
| DET-03/04 | Literal vs dynamic CALL; dynamic placeholder scoped per program, never bound; reference to data item | unit + resolver | `… -run TestCobolGrammar_Calls` ; `go test -race ./internal/resolver/ -run TestCobolReferences -count=1` | ❌ |
| DET-05/06 | COPY/-INC/IDMS/INCLUDE forms; bind exactly one same-repo; missing/ambiguous with candidates; no cross-repo | unit + resolver | `… -run TestCobolGrammar_Copy` ; `… -run TestCobolReferences_Copybook` | ❌ |
| DET-07/08 | IDMS record/set reads/writes per verb; scope placeholder | unit | `… -run TestCobolGrammar_IDMS` | ❌ |
| DET-09/10 | CICS program (literal/dynamic), transaction, map, mapset | unit | `… -run TestCobolGrammar_CICS` | ❌ |
| DET-11/12 | SQL reads/writes, schema qualifier, underscores, alias meta, CTE subtraction, cursor, PREPARE dynamic, INCLUDE | unit | `… -run TestCobolGrammar_SQL` | ❌ |
| CR-01 | BOM-prefixed file → `prov_start_byte` +3 on nodes and edges | integration (indexer) | `go test -race ./internal/indexer/ -run TestCobolBOMByteOffsets -count=1` | ❌ |
| Cross-cutting | Degraded (`Affected`, amber/red) facts emitted and marked; extractor version `/3`; no meta key with secret-like substring | unit | `… -run 'TestCobolGrammar_(Degraded\|ProvKeys)'` | partial (`TestCobolGrammar_DegradedAmber` exists) |

### Sampling Rate
- **Per task commit:** the quick run command for the touched package.
- **Per wave merge:** `go test -race ./internal/parser/languages/ ./internal/resolver/ ./internal/indexer/ ./cmd/gortex/ -count=1`
- **Phase gate:** full `go test -race ./...` green before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] Shared invented fixtures (one multi-construct program, one with sections and nested programs, IDMS, CICS, SQL, BOM variant) as test constants in `cobol_grammar_test.go` or a sibling `_test.go`. Invented only; the probe fixtures in this research are suitable templates.
- [ ] `internal/resolver/cobol_references_test.go`: in-memory graph with two repos, duplicate copybooks, nested same-file programs.
- [ ] Extend `cmd/gortex/cobol_tracer_test.go` with paragraph-flow and call-binding assertions (the real path).
- [ ] Framework install: none.

## Security Domain

### Applicable ASVS Categories (Level 1)

| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | no | — (local indexing) |
| V3 Session Management | no | — |
| V4 Access Control | no | Repo scoping is integrity, covered by D-09/D-11 (no cross-repo binding) |
| V5 Input Validation | **yes** | Untrusted COBOL text becomes IDs and meta. Bounded by producer budgets and the 2-minute ceiling. Percent-encode literal-derived ID segments (Pitfall 9). No source text in error messages (the fork's T-06-04 discipline). |
| V6 Cryptography | no | — (content IDs come from the producer) |
| V7 Logging | yes | Never log estate source or names. Tests are invented-only. |
| V14 Configuration | yes | Meta keys must not contain `password`, `passwd`, `secret`, `token`, `credential`, `privatekey`, `apikey`, or `authorization` (the Neo4j filter drops them, properties.go:174). Avoid names like `cobol_token_*`. |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| ID injection via literal names (`::unresolved::`, `/`, `#`) | Tampering | Percent-encode literal-derived segments, keep raw values in meta. |
| False cross-repo binding (wrong program or copybook) | Tampering / Repudiation | Owned namespace, same-repo + exactly-one pass, candidates on ambiguity. |
| Estate content leaking into the public repo | Information disclosure | Invented fixtures only; aggregate-only measurement; this document holds no names or paths. |
| Pathological parse memory (C-side recovery) | DoS | Existing `cobolAnalyzeSlot` serialization. WR-04 per-process limit is deferred to Phase 4. |
| SQL injection | Tampering | Not applicable: no SQL text is built; Store APIs only. Org rule (parameterized SQL) satisfied. |

## Suggested Plan Slicing (research question 9)

| Wave | Plan | Scope | Depends on |
|---|---|---|---|
| 1 | **03-01 Tracer: paragraphs + PERFORM** | `cobol_grammar_walk.go`; paragraph `KindFunction` nodes with `@proc` IDs; one `calls` edge with `cobol_flow=perform`; extractor version `/3`; walker call sites in `Extract`, so later plans touch only their own files; real-path test asserting `get_callers` on a paragraph | — |
| 1 | 03-02 CR-01 BOM byte shift | `source_revision.go` sibling + indexer test | — |
| 2 | 03-03 Structure | divisions, data sections, procedure sections, all statements (`KindLocal`), extents, `cobol_ordinal`, statement symbols, edge dedupe helper | 03-01 |
| 2 | 03-04 Data items | levels, FILLER/66/77/88, meta, REDEFINES/RENAMES (in-program lookup) | 03-01 |
| 3 | 03-05 Control flow | label rules, THRU expansion, GO TO/DEPENDING, label placeholders and reasons | 03-03 |
| 3 | 03-06 CALL/COPY + resolver | literal/dynamic CALL, COPY/-INC/INCLUDE forms, data-item references, `cobol::` exclusions, `resolveCobolReferences` | 03-03, 03-04 |
| 4 | 03-07 IDMS + CICS | resources, verb table, CICS LINK/XCTL via the 03-06 placeholders | 03-06 |
| 4 | 03-08 SQL | tables, qualifiers, aliases, CTE, cursor, PREPARE, INCLUDE | 03-06 |
| 5 | 03-09 Phase acceptance | multi-construct invented fixture through the real indexer, SQLite, resolver, MCP, and CLI; Neo4j property-sanitization smoke check | all |
| gated | Copybook contents | **only after the USER DECISION.** Options A/B add a producer-prerequisite plan (fork change, user-approved push, re-pin) before a copybook data-items plan. | user |

Wave 2 plans run in parallel only if each owns its walker file and 03-01 has already added the call sites in `Extract`.

## Sources

### Primary (HIGH confidence)
- Pinned producer, probed this session: scratch probe built against `preprocessor@v0.0.0-20260930215433-f9eaf99c34a9` with `GOWORK=off GOPROXY=off`. Invented fixtures for core structure, CICS/SQL, IDMS, copybooks, and edge cases. NIST aggregate counts.
- Fork source: `grammar.js` (lines 36-41, 1366-1431, 1481-1496, 1709-1722, 1973-1982, 2980-2994, plus the CICS/SQL/IDMS rules), `queries/{idms,cics,sql}.scm`, `preprocessor/parsefacts/{facts,extract}.go`, `preprocessor/analyze.go`, `preprocessor/transform/{idms,endevor,catalog,fixed_test}.go`, `preprocessor/sourcemap/sourcemap.go`.
- Gortex source: `internal/parser/languages/cobol_grammar*.go`, `cobol.go`, `register.go:119`; `internal/graph/{node,edge,graph,stub}.go`; `internal/graph/store_sqlite/{schema,add_batch_set,meta_json}.go`; `internal/resolver/{resolver,lsp_resolve,cross_repo,relative_imports,lua_imports,terminal,module_attribution}.go`; `internal/indexer/{indexer,transform,source_revision,incremental_resolve}.go`; `internal/neo4jprojection/properties.go`; `internal/analysis/deadcode.go`; `internal/parser/languages/{go_sql,go_orm,go_dataflow,java}.go`.

### Secondary (MEDIUM confidence)
- [IBM Enterprise COBOL 6.5 — Qualification](https://www.ibm.com/docs/en/SS6SG3_6.5/lr/ref/rlrefqua.html): paragraph uniqueness within a section; the section is the only qualifier.
- [IBM Db2 for z/OS 12 — INCLUDE statement](https://www.ibm.com/docs/SSEPEK_12.0.0/sqlref/src/tpc/db2z_sql_include.html): `INCLUDE SQLCA | SQLDA | member-name`.
- [IBM CICS — SEND MAP](https://www.ibm.com/docs/en/SSNAQ8_10.1.0/com.ibm.cics.tx.doc/reference/r_send_map.html): MAPSET option semantics. Not relied on (no defaulting is applied).

### Tertiary (LOW confidence)
- Training-based COBOL, IDMS, and CICS semantics tagged `[ASSUMED]` (Assumptions Log).

## Metadata

**Confidence breakdown:**
- Observation shapes and role derivation: HIGH. Probed against the exact pinned producer.
- Gortex integration (kinds, edges, resolver, identity, CR-01): HIGH. Read from source this session.
- Volume (D-05): MEDIUM. Public suite only; the private corpus was not reachable.
- COBOL, IDMS, and CICS semantics for verb classification and label scope: MEDIUM to LOW. Partly cited, partly assumed, and the raw verb is always kept.

**Research date:** 2026-10-01
**Valid until:** 2026-10-31, or until the fork pin changes, whichever comes first.
