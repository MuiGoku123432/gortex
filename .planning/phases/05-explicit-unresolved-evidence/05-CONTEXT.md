# Phase 5: Explicit Unresolved Evidence - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can see two kinds of deterministic finding for COBOL programs and copybooks, and tell them apart:

- **`PARSE_UNRESOLVED`**: the source exists, but the parser could not fully interpret it.
- **`EXTERNAL_UNRESOLVED`**: the source references an artifact that was never obtained.

Each finding is persisted with these fields:
- a stable ID
- exact scope and range (or an explicit absence reason)
- the containing symbol
- resolution class
- normalized reason
- severity
- parser version
- lifecycle status

Unchanged evidence keeps its finding ID. Changed or resolved evidence closes or supersedes only the finding it applies to. A missing target gets a placeholder that carries no fabricated content. An uncontained parser recovery cascade stays a deterministic regression failure.

Requirements: GAP-01..GAP-08.

Also in scope (deferred here from Phase 3, see `03-VERIFICATION.md` `deferred:`):
- **WR-06:** a copybook analysis failure is distinguishable from a missing copybook.
- **WINDOWS.md 4-6:** SQL unparsed tails, `DECLARE CURSOR` in WORKING-STORAGE, and `DECLARE ... WITH HOLD` produce explicit parse findings.
- **WR-07, WR-09, IN-08:** unresolved reasons on edges are scope-accurate.

Out of this phase:
- Native query verbs that filter, aggregate, or rank findings and unavailable artifacts, and lower-bound disclosure (Phase 6: QUERY-08, QUERY-09, QUERY-10).
- Monitoring of findings by class and reason (Phase 6: EVAL-06).
- AI review, claims citing findings, and human review (Phases 7-8).
- Findings for languages other than COBOL. The `finding` kind and its meta are language-neutral, so Phase 6.1 (VB.NET) or later can emit them.

Depends on Phase 4. As of 2026-10-14, Phase 4's UAT is complete but its verification has not been closed out. Phase 3 still needs re-verification after `9f6c1973`.

</domain>

<decisions>
## Implementation Decisions

### Carried forward (do not re-decide)
- Provenance lives in flat `prov_*` and namespaced Meta keys, with no new `graph.Node` fields or SQLite columns on nodes or edges (P2 D-08).
- Never invent coordinates. A gap with no range carries the producer's `AbsenceReason`, surfaced as-is (readiness §4).
- Do not pin the producer's `gaps[].id` or `ledger.configuration_id` in tests (readiness §4).
- Nodes and edges are hard-evicted by `file_path`. The node and edge store holds no inactive rows. Lifecycle state applies to findings only (P4 D-05).
- Line numbers never discriminate identity (P2 D-10, ID-02).
- Unbound edges keep their `cobol_unresolved_reason` vocabulary (P3 D-12). Phase 5 builds findings on top of it and does not replace it.
- Binding follows P3 D-09/D-11, P4 D-09 and P4 D-19:
  - CALL and COPY bind only on an exact single match within the same repository plus the allowed-repository union.
  - `missing` means no candidate in that union.
  - `COPY IDMS` members are never bound (P3 D-19).
- Incremental indexing must equal a clean rebuild, compared on the canonical dump, which includes all deterministic meta (P4 D-01..D-04).
- Ownership health and SEC-08 (P4 D-14, D-15): every new node must be owned or sit in an enumerated exempt namespace.
- Fixtures are invented COBOL only, because the repo is public (P2 D-14).

### Finding identity
- **D-01:** A finding ID is anchored to:
  - the containing symbol ID
  - the resolution class
  - the gap kind
  - the node kind or expected family
  - a digest of the gap's own original source text

  A `#k` ordinal is added only when identical evidence text appears more than once in the same container. Edits elsewhere in the file and line shifts keep the ID. Editing the gap's own text produces a new ID.

  This deliberately overrides two weaker schemes:
  - the producer's `gapID`, which hashes the whole document's content ID plus byte offsets;
  - the "source revision in the key" direction in `GRAPH-SCHEMA.md`'s Layered Identities table.

  — **Reversibility:** costly — finding IDs are persisted, projected to Neo4j, and will be cited by Phase 8 claims. Changing the scheme closes and reopens every finding in the estate.
- **D-02:** The parser or extractor version is **not** part of the ID. If a new parser version still produces the same gap, the finding keeps its ID and its parser-version fields are re-stamped. Claims handle their own parser-version staleness (REVIEW-05).
- **D-03:** Closed and superseded are distinguished deterministically:
  - **Closed:** the evidence is gone. Either the gap is no longer produced, or the missing target was retrieved or bound.
  - **Superseded:** a different active finding of the same class and kind now sits in the same container. The old record names its successor ID.

  Never match across containers, and never guess.
- **D-04:** There is one `EXTERNAL_UNRESOLVED` finding per reference occurrence (each `CALL 'PGMX'` or `COPY X` statement), with its exact range and containing symbol. Artifact-level counts are aggregated from findings or from the placeholder's in-edges.

### Storage and history
- **D-05:** Add one new node kind, **`finding`**. This is justified by the `GRAPH-SCHEMA.md` rule, because a generic kind would imply false semantics. Each active finding is:
  - a repository-prefixed node owned by its source file (`file_path` eviction applies);
  - carrying its fields as Meta;
  - searchable, included in the canonical dump, covered by the ownership audit, and projected to Neo4j through the existing projection.

  — **Reversibility:** costly — the kind is persisted in SQLite, projected as a Neo4j label, and consumed by Phase 6 queries.
- **D-06:** Closed and superseded findings do not stay as nodes. Every close or supersede transition is appended to a new sidecar SQLite table, **`finding_history`**. Each row holds:
  - finding ID
  - prior status and new status
  - successor ID (for supersede)
  - evidence and parser stamps
  - repository, workspace, and project scope

  The node store therefore never holds inactive rows. The `file_index_failures` table is the precedent for an additive sidecar table.

  — **Reversibility:** costly — this is a schema migration (v23 or later) plus a new persisted table that later claim lineage reads.
- **D-07:** Finding history is best-effort and not canonical:
  - It survives normal reindexing, incremental updates, and daemon restarts.
  - A store rebuild (a schema bump or a fresh store) starts it empty.
  - It is excluded from the incremental-equals-clean comparison. Only active finding nodes are canonical.
- **D-08:** Users see findings in Phase 5 through existing surfaces only:
  - `get_symbol` by exact ID, through both MCP and CLI (the P2 D-11 acceptance pattern);
  - search by name or kind;
  - edges to the containing symbol and to the placeholder;
  - `index_health`, which reports active finding counts by class.

  A list or filter verb waits for Phase 6 (QUERY-08).

### What becomes a finding
- **D-09:** `EXTERNAL_UNRESOLVED` covers missing artifacts only:
  - the edge reason `missing` (a CALL, LINK, XCTL, or COPY target with no candidate in the allowed scope);
  - `idms_dictionary_member` (the IDMS dictionary was not retrieved);
  - the producer's `external-reference` gaps (an `-INC` member that was not obtained, or a cyclic one). **Amended by D-17.**

  `ambiguous` and `dynamic` stay edge-only reasons. The artifact either exists more than once, or the target cannot be known statically. Neither is "not retrieved". Phase 6 QUERY-03 already distinguishes them through edge reasons.
- **D-10:** `PARSE_UNRESOLVED` comes from the producer's gap list only:
  - `error`, `missing` node, `unparsed-tail`, `lost-capture`, `unmapped-capture`;
  - the document-scope gaps `no-tree`, `budget-exceeded`, and `invocation-failure`;
  - the cascade finding in D-14.

  Gortex's in-source reasons (`label_not_found`, `label_not_in_copybook`, `not_declared_in_*`, `cursor_*`, `scope_role_unknown`, `caller_unowned`) stay edge reasons. Phase 5 makes them scope-accurate, as deferred from Phase 3:
  - **WR-07:** a twice-declared cursor is `ambiguous`, not `cursor_not_declared`.
  - **WR-09:** a procedure copybook's cursor use gets a copybook-scoped reason.
  - **IN-08:** a program nested under unresolved containment does not report a plain `missing`.

  GAP-01 makes every named unparsed tail a finding. That includes the tail every `EXEC SQL` statement leaves under the pinned producer (WINDOWS.md 6), which yields one low-severity finding per SQL statement. **Amended by D-18** (the producer emits one tail gap per token, not per statement).
- **D-11:** The GAP-05 placeholder is a **repository-scoped node**, one per (referencing repository, artifact type, normalized key). Artifact types are program, copybook, IDMS dictionary member, and `-INC` member.

  It carries:
  - the normalized key (`cobol-name-norm/1`);
  - the artifact type;
  - a "not obtained" status.

  It carries no source, body, range, or attributes. The as-written spelling stays on the referencing edges, as in P4 D-18.

  Each per-occurrence `EXTERNAL_UNRESOLVED` finding and its reference edge point to the placeholder. When the last reference disappears, the placeholder is removed, following the 04-14 shared-resource eviction pattern. It is owned by the referencing repository, so it is scope-safe (SEC-01) and covered by the audit (SEC-08).

  A placeholder must never satisfy a query that requires parsed implementation (`GRAPH-SCHEMA.md` Required Verification).

  — **Reversibility:** costly — this changes today's unprefixed, edge-only `unresolved::cobol::program|copybook::` targets into persisted, projected, owned nodes.
- **D-12:** Severity comes from the producer's static table (`cobol-handoff-v1`):
  - **high:** document-scope gaps, lost capture, unmapped capture;
  - **medium:** error, missing node, external reference;
  - **low:** unparsed tail.

  Gortex-derived `EXTERNAL_UNRESOLVED` findings (`missing` CALL/COPY, `idms_dictionary_member`) are **medium**, matching the producer's external-reference severity. A versioned severity rule ID is recorded in Meta.

### Cascade regression gate (GAP-08)
- **D-13:** Cascades are detected from the producer's grade signals. Gortex does not re-derive them. A document whose grade reasons include the cascade family (`division-scale-recovery`, `retained-structure-collapse`, `missing-required-anchor`, `lost-capture`) gets one **high**-severity, document-scope `PARSE_UNRESOLVED` finding with reason `uncontained-cascade`, carrying the producer's reasons. Its localized gaps are still emitted individually (GAP-01).
- **D-14:** "Regression failure" means a committed fixture gate:
  - It uses invented fixtures and runs on the real path (registered extractor, indexer, SQLite temp store).
  - It runs in the default `go test -race`, with no flag.
  - Injected cascade shapes must produce the `uncontained-cascade` finding.
  - Contained shapes (negative controls) must keep their downstream paragraphs and data items.

  In a real estate, cascades are reported through findings and `index_health` counts. They never block indexing and never mark a repository unhealthy (the same posture as P4 D-15).
- **D-15:** Port the injection cases from `internal/parser/forest/cobolprobe/cascade_test.go` `TestErrorCascade` into the D-14 gate:
  - 20 data items plus paragraphs;
  - IDMS DML;
  - `EXEC CICS` shapes.

  Keep `cobolprobe` as a flag-gated measurement harness for real corpora. It is not the gate.
- **D-16:** A finding closes only through deterministic evidence: a source edit, a parser or extractor change that stops producing it, or retrieval or binding of the target. No API or tool may close or alter a finding in any other way. Phase 8 claims may cite findings but never change their status. There is no `closable_by` field, and nothing AI-specific is built in this phase.

### Post-research amendments (2026-10-19, user-confirmed)
Research (05-RESEARCH.md Pitfalls 1, 2, 6, 7) ran the pinned producer and found facts that contradict or leave open D-09/D-10 as written. The user confirmed each amendment below (D-17..D-20); they override the original text where they conflict.

- **D-17 (amends D-09 bullet 3):** The producer's `external-reference` gaps are **not** mapped to findings. Gortex builds the analyzer with an empty `-INC` catalog (`transform.NewCatalog(nil)`), so the producer reports every `-INC` member as not obtained, including members the resolver later binds to a present copybook. `-INC` `EXTERNAL_UNRESOLVED` findings and placeholders come **only** from the resolver, when the `inc` edge (`cobol_copy_form=inc`) is decided `missing`. A file whose `-INC` edge is bound must have no `-INC` finding. (`cyclic` cannot fire with an empty catalog.)
- **D-18 (amends D-10 last paragraph):** One `PARSE_UNRESOLVED` finding per producer `unparsed-tail` gap (the producer emits one per token), not one per SQL statement. Gortex does not aggregate tails. Identical tail text in the same container is disambiguated by the D-01 `#k` ordinal. Tests and `index_health` counts are sized to per-token volume.
- **D-19 (narrows D-09 bullet 1 and D-11):** Edges with `sql_include_builtin=true` (`EXEC SQL INCLUDE SQLCA` / `SQLDA`) produce **no** `EXTERNAL_UNRESOLVED` finding and **no** placeholder. The edge keeps its existing `missing` reason unchanged.
- **D-20 (accepted limitation, RESEARCH A7):** The WR-06 analysis-failure fallback (identity-only node plus an `analysis-failed` `PARSE_UNRESOLVED` document finding) applies to **copybooks only**. A **program** whose analysis fails keeps today's behavior: extraction fails hard and the failure is recorded in `file_index_failures`. Its callers may therefore resolve to `missing` and receive an `EXTERNAL_UNRESOLVED` finding and a "not obtained" placeholder even though the file exists. Phase 5 does not guess a program identity from a filename. This limitation MUST be documented in `docs/languages.md` and left for Phase 6.

### Gap-closure amendments (2026-10-21, user-confirmed)
Source: 05-VERIFICATION.md (gaps_found, SC4) and 05-REVIEW.md. These override earlier text where they conflict. All persisted-ID changes below ship together under one rule bump, so the estate re-IDs once.

- **D-21 (amends D-01 anchor; closes CR-01 and WR-07):** The finding ID and `finding_container` (the D-03 ledger pairing key) anchor to a **name-stable** container: paragraph, section, data or environment section, program, or file. Never a statement ordinal (`@<kind>#k`). The statement may remain the `member_of` target. Document-scope findings (`no-tree`, `budget-exceeded`, `invocation-failure`, `uncontained-cascade`, `analysis-failed`, gaps outside every program) always anchor to the **file node**, never to "the single program when exactly one exists". `graph.FindingIDRule` bumps to `gortex-finding-id/2`. Regression tests MUST insert and delete a same-kind statement above existing gaps, and add and remove a second program, and assert unchanged IDs and no ledger rows for unchanged evidence.
- **D-22 (WR-04, accepted risk):** The evidence digest stays **unkeyed** so IDs remain reproducible across machines and clean rebuilds (core value). The residual dictionary-recovery risk for single-token evidence MUST be documented in `docs/languages.md`, and the doc comment in `internal/graph/finding.go` claiming source never leaks through the ID or projection MUST be corrected. — **Reversibility:** costly (changing later re-IDs every finding).
- **D-23 (WR-01):** The finding ledger MUST run under the same exclusion as the finding mutations it observes (resolver reconcile and repository file commits), so no ledger can read between a file commit and its reconcile. A concurrency test MUST interleave a ledger with a reconcile and a file commit and assert no false `active→closed→active` pair.
- **D-24 (WR-02):** The master-resolve drain (`runMasterResolveHookedContext`) MUST record the ledger for the repositories it drained, not discard them. Closed-vs-superseded classification must not depend on timing.
- **D-25 (WR-03):** External finding evidence MUST include the reference form: edge kind, `prov_observation_kind`, and the CICS command (stamped on the edge if absent), so `CALL 'X'` and `EXEC CICS XCTL PROGRAM('X')` produce different IDs. The evidence rule bumps to `gortex-finding-evidence/2`.
- **D-26 (WR-06, enforces D-04):** One `EXTERNAL_UNRESOLVED` finding per reference **occurrence**, even when same-line duplicates collapse into one edge. An edge with `cobol_occurrence_count = n` yields `n` findings in a deterministic order, disambiguated by the existing `#k` rule.
- **D-27 (WR-05, amends D-11 ID shape):** Placeholders live in an ID namespace no repository file path can produce, with an empty or virtual FilePath. The sweep evicts placeholders **by ID** with a kind and `cobol_kind=missing_artifact` guard, never via `EvictFile`. Creating a placeholder over an existing non-placeholder node MUST be refused.
- **Out of scope for this gap cycle:** WR-08 (verifier judged defensible under COBOL semantics; revisit in Phase 6) and IN-01..IN-07 (recorded in 05-REVIEW.md).

### Claude's Discretion
- **Meta key names** for finding fields. Follow the P2/P3 conventions (`prov_*` for provenance, plus a `finding_*` namespace or similar). Avoid secret-like substrings, because the Neo4j projection filters them.
- **Edge kinds:**
  - finding to containing symbol (reuse `member_of` or similar existing kinds);
  - finding and reference edge to placeholder. Reuse existing kinds. Add `blocked_by` only if research shows a required query is obscured without it.
- **Placeholder node kind and ID grammar.** It must be repository-owned, distinguishable from parsed programs and copybooks, and never mistakable for an implementation. Choose how resolver binding and closure interact with it. The resolver currently rewrites edge `To` on bind, and the edge `To` today is the unprefixed `unresolved::cobol::` ID.
- **Evidence-digest details:**
  - text normalization before hashing;
  - the digest algorithm (prefer the existing BLAKE3 dependency);
  - what to digest when the original range is absent (for example node kind plus absence reason). Never invent coordinates.
- **Containing symbol:**
  - map the producer's `Container` observation index to a Gortex node ID through the existing walk (`cobolWalk.id`, `statementSymbol`);
  - attach document-scope gaps to the file node, or to the program node when exactly one exists.
- **Supersede pairing** when several old and new findings of one class and kind share a container: pair them in source order, then close or open the leftovers.
- **`finding_history`:** schema, migration (`storeNeedsRebuild` or an in-place migration), and indexes. Scope columns follow the `file_index_failures` precedent.
- **Where the extractor and resolver emit findings, and where lifecycle is diffed:**
  - parse findings come from `Extract`;
  - external findings and placeholders come from `reconcileCobol`;
  - lifecycle diffing must see the prior active finding set before eviction, on both the incremental and the clean path.
- **WR-06 representation:** how a copybook whose analysis failed is distinguished from a missing copybook. Expected: a `PARSE_UNRESOLVED` document finding on the present copybook, never an `EXTERNAL_UNRESOLVED`.
- **`index_health` shape** for the finding counts.
- **Extractor version:** bump `prov_extractor_version` past `gortex-cobol-grammar/5`, once for the phase, because the emitted nodes change.
- **Plan slicing and wave order.** Suggested order:
  1. finding kind plus parse findings (tracer);
  2. external findings plus placeholders;
  3. lifecycle and history;
  4. Phase 3 reason fixes;
  5. cascade gate plus incremental-equals-clean coverage.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` §"Phase 5: Explicit Unresolved Evidence": the goal and success criteria 1-5.
- `.planning/REQUIREMENTS.md`:
  - GAP-01..GAP-08 (this phase);
  - QUERY-08, QUERY-09, QUERY-10 and EVAL-06 (the Phase 6 consumers of these findings);
  - SEC-01 and SEC-08 (scope and ownership);
  - the Out of Scope list ("Inventing contents or attributes for unavailable programs...").
- `.planning/PROJECT.md` §Key Decisions and §Constraints: fork hygiene, SQLite as sole authority, and the parser pin.

### Finding contract and schema direction
- `docs/ai-enhanced-cobol-graph-handoff.md`:
  - §6 evidence classes;
  - §8 parser and retrieval gap contract, including outcome 4, the uncontained cascade;
  - §9 finding contract (minimum fields);
  - §12 stable identity.
- `.planning/research/GRAPH-SCHEMA.md`:
  - the "Parser finding" and "Missing artifact" rows;
  - Provenance Vocabulary (evidence class vs lifecycle);
  - Finding Lifecycle rules 1-7;
  - Incremental Reconciliation;
  - Required Verification.
  - Its Layered Identities "Finding" key direction (source revision in the key) is **superseded by D-01/D-02**.

### Producer handoff (the source of `PARSE_UNRESOLVED`)
- `.planning/research/COBOL-PARSER-READINESS.md`:
  - §3, the `Gaps[]` to Phase 5 mapping;
  - §4 boundary rules: the producer emits facts and gaps only, never invent coordinates, do not pin gap or config IDs;
  - §6 producer limits.
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/handoff/gaps.go`:
  - `Gap` struct, `ResolutionClass`, `Kind`, `Severity`, `Scope` constants;
  - `severityTable` (`cobol-handoff-v1`), `deriveGaps`, `deriveExternalGaps` (`-INC`), `gapID` (do not reuse; see D-01).
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/parsefacts/facts.go`: `Gap` (Parent index, Division, CrossesDivision), `Observation`, `AnchorFact`.
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/grade/grade.go`: the grade `Reason*` constants used by D-13.

### Prior phase decisions
- `.planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-CONTEXT.md`:
  - D-07 (degraded marking);
  - D-08 (provenance meta);
  - D-09 (revision and absence reasons);
  - D-10 (identity);
  - D-11 and D-12 (acceptance pattern);
  - D-14 (invented fixtures).
- `.planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-CONTEXT.md`:
  - D-09 to D-12 (placeholders, reasons, and the Phase 3/5 line);
  - D-18 (`cobol-name-norm/1`);
  - D-19 (`COPY IDMS` never bound).
- `.planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-VERIFICATION.md` frontmatter `deferred:`. These are the WR-06, WINDOWS 4-6, WR-07, WR-09 and IN-08 items assigned to Phase 5.
- `.planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-CONTEXT.md`:
  - D-01 to D-05 (canonical dump and hard-evict);
  - D-09 and D-19 (allowed-scope union);
  - D-14 and D-15 (ownership and health posture);
  - D-18 (normalized names, as-written spelling on edges).
- `.planning/WINDOWS.md` entries 4, 5 and 6: the SQL producer gaps that become findings.

### Codebase maps
- `.planning/codebase/ARCHITECTURE.md`: the indexer, resolver and graph layers.
- `.planning/codebase/TESTING.md`: test conventions, `graph/storetest`, and `-race`.

</canonical_refs>

<code_context>
## Existing Code Insights

Scouted 2026-10-14. Graph IDs live under `gortex/...`. A sibling checkout, `GoApps/gortex-phase4-sql`, is also indexed and duplicates many symbols, so do not confuse the two.

### Reusable Assets
- **Grade and degraded stamps:** `internal/parser/languages/cobol_grammar.go:239-240` already stamps `prov_document_grade` and `prov_document_grade_reasons`. The cascade signals for D-13 are therefore already on the file node. `prov_affected` is stamped per observation (`cobol_grammar.go:267`, `cobol_grammar_walk.go:173,209`).
- **Container mapping:** `cobol_grammar_walk.go` `cobolWalk.id` (:130) and `statementSymbol` (:140) map an observation index to a node ID or QualName. The producer's gap `Container` and `Parent` indexes map the same way.
- **Range absence pattern:** `cobolStampRange` (`prov_range_exact` / `prov_range_absence`) is the existing pattern for a range that is absent by design.
- **Binding pass:** `internal/resolver/cobol_references.go:124` `Resolver.reconcileCobol` is the single COBOL binding pass, on both the full and the incremental path. It is where `missing` is decided, so external findings and placeholders are created, kept, or closed here.
- **Placeholder removal precedent:** `internal/resolver/cobol_resources.go:38` `reconcileCobolResources` evicts shared nodes that have no in-edges. D-11 placeholder removal follows this.
- **Ownership audit:** `internal/graph/prefix_diagnostics.go` lists the exempt namespaces (`OwnershipGlobalIDNamespaces`, including `UnresolvedMarker`). Placeholder and finding nodes must pass the audit.
- **Sidecar table precedent:** `internal/graph/store_sqlite/store_file_index_failures.go` is an additive table with `view_gen`, `repo_prefix`, `workspace_id` and `project_id`. `schema_version.go:37` sets `currentSchemaVersion = 22`.
- **Canonical dump:** `internal/testutil/graphdump` (`graphdump-volatile/3`). Active finding and placeholder nodes are compared; `finding_history` is not.
- **Health reporting:** `index_health` is in `internal/mcp/tools_enhancements.go` (around :3270, next to `ReadPrefixDiagnostics`). Finding counts go here.
- **Cascade cases to port:** `internal/parser/forest/cobolprobe/cascade_test.go` `TestErrorCascade`, flag-gated by `-enhanced-parser`.

### Established Patterns
- **The producer's gap list is not consumed anywhere in Gortex today.** No code references `handoff.Gap`, `PARSE_UNRESOLVED` or `EXTERNAL_UNRESOLVED`.
- **Unbound references live only on edges.** Each carries `cobol_unresolved_reason`, set in `cobol_grammar_refs.go`, `_flow.go`, `_data_refs.go`, `_mainframe.go`, `_sql.go` and `resolver/cobol_references.go`.
- **Unresolved targets have no node today.** They are edge `To` IDs only, and `unresolved::cobol::program|copybook::<key>` is unprefixed, so every repository shares it.
- **Extractors emit repo-relative IDs.** `Indexer.applyRepoPrefix` (`internal/indexer/indexer.go:2278`) prefixes them and leaves unresolved targets unprefixed.
- **Eviction is by `file_path`**, so a finding owned by a file disappears on reindex. The history diff must read the prior active set first.

### Integration Points
- **Parse findings:** emitted in `CobolGrammarExtractor.Extract` (programs and the copybook path in `cobol_grammar_copybook.go`), as file-owned nodes.
- **External findings and placeholders:** emitted in `reconcileCobol`. This crosses files and repositories, so it must run on the incremental path too, under the P4 D-08 escalation rule.
- **Lifecycle diff and history writes:** at the file-generation commit (`AddBatch` / the structural incremental commit) and at the resolver pass. Incremental and clean must produce the same active set.
- **Neo4j projection:** projects the new kind automatically. Check the meta-key filter.

</code_context>

<specifics>
## Specific Ideas

- The user accepted every recommended option. The overriding preference is "accuracy without churn": an ID must not change because of unrelated edits or a parser bump, and findings must mean exactly "the parser failed" or "the artifact was not obtained", nothing broader.
- Expect a high volume of low-severity findings. Every `EXEC SQL` statement leaves an unparsed tail under the pinned producer (WINDOWS.md 6), and GAP-01 makes each one a finding. That is accepted, not a reason to filter.
- The regex era measured 829 dynamic versus 352 literal CALLs on the DCC corpus. Dynamic calls stay out of findings (D-09), so the blocker ranking stays meaningful.

</specifics>

<deferred>
## Deferred Ideas

- **A `list_findings` MCP/CLI verb** with class and file filters: Phase 6 (QUERY-08).
- **Finding history that survives store rebuilds:** revisit in Phase 8 if claim lineage needs it.
- **Impact-based severity** (for example, a missing called program as high): consider for Phase 6 ranking or monitoring.
- **`ambiguous` and `dynamic` references as findings:** not now. They stay edge reasons, distinguished by Phase 6 QUERY-03.
- **Promoting in-source misses** (`label_not_found`, `not_declared_*`) to `PARSE_UNRESOLVED` when they fall inside an Affected cascade region: possible later precision work.
- **An explicit `closable_by` field on findings:** Phase 8, if claim validation needs it.
- **Marking a repository unhealthy, or failing indexing, on an uncontained cascade:** rejected for now (D-14). Phases 6 and 7 may revisit for export or AI gating.
- **Findings for VB.NET and other languages:** Phase 6.1 or later, reusing the language-neutral `finding` kind.

</deferred>

---

*Phase: 05-explicit-unresolved-evidence*
*Context gathered: 2026-10-14*
