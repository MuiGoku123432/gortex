# Roadmap: Gortex Mainframe Engine v1.0

## Overview

This milestone now begins with a general, manually invoked projection that materializes an explicitly scoped snapshot of Gortex's authoritative SQLite graph into Neo4j through equivalent CLI and MCP surfaces. The projection is language-agnostic, retry-safe, and rebuildable; it never joins indexing, writes back, or changes SQLite authority. The existing COBOL tracer, grammar-dependent extraction, unresolved-evidence, and AI phases remain coherent but are deferred until parser/grammar readiness.

## Phases

**Phase Numbering:**

- Integer phases are planned milestone work.
- Decimal phases are reserved for urgent insertions.

- [x] **Phase 1: General Neo4j Projection Command** - Materialize a selected SQLite graph snapshot into Neo4j through equivalent explicit CLI and MCP operations. (completed 2026-09-22)
- [x] **Phase 2: Thin Deterministic Native Tracer and Minimum Contracts** - Deliver one trustworthy native COBOL trace after parser/grammar readiness. (completed 2026-10-01)
- [ ] **Phase 3: Deterministic COBOL and Mainframe Breadth** - Expand native extraction across COBOL structure, control flow, data, calls, copybooks, IDMS, CICS, and SQL.
- [ ] **Phase 4: Stable Incremental and Cross-Repository Lifecycle** - Make identity, reconciliation, and scoped resolution stable across reruns, changes, and repositories.
- [ ] **Phase 5: Explicit Unresolved Evidence** - Persist parser gaps and unavailable artifacts as distinct, lifecycle-aware deterministic findings.
- [ ] **Phase 6: Native Deterministic Acceptance and Pre-AI Release Gate** - Prove native queries, deterministic fixtures, lifecycle equivalence, monitoring, and unresolved-boundary behavior before AI.
- [ ] **Phase 7: AI Contract, Policy, and Evaluation Foundation** - Define and enforce the bounded, redacted, provider-authorized AI boundary before claim implementation.
- [ ] **Phase 8: Evidence-Linked Claims and Human Review** - Persist validated AI claims separately and support auditable review and lifecycle transitions.
- [ ] **Phase 9: End-to-End AI Evaluation and Projection Safety** - Prove AI release invariants and projection disclosure boundaries while keeping SQLite authoritative.

## Phase Details

### Phase 1: General Neo4j Projection Command

**Goal**: Users can explicitly materialize a selected, scoped snapshot of Gortex's authoritative SQLite graph into Neo4j through equivalent CLI and MCP operations without making Neo4j authoritative or required by ordinary Gortex workflows.
**Depends on**: Nothing (first phase)
**Requirements**: NEO-01, NEO-02, NEO-03, NEO-04, NEO-05, NEO-06, NEO-07, NEO-08, NEO-09, NEO-10, NEO-11, NEO-12, NEO-13, NEO-14, NEO-15, NEO-16, NEO-17, STORE-01, STORE-02, STORE-03, STORE-04, STORE-05
**Success Criteria** (what must be TRUE):

  1. A user can invoke equivalent CLI and MCP projection operations for an explicit workspace/project/repository scope, while missing or ambiguous scope fails closed and output never exposes credentials.
  2. The selected snapshot materializes language-agnostic nodes and relationships with stable keys, required Neo4j constraints, and preserved scope, provenance, evidence, origin, confidence, lifecycle, and source-location properties.
  3. Repeating or resuming a projection uses bounded transactions and idempotent upserts without duplicates; stale records are reconciled only within the explicitly owned scope/generation, including after partial failure.
  4. A user can preview the projection without mutation, observe redacted progress and final scoped counts, cancel work, and receive actionable failure or incomplete-snapshot results.
  5. Disposable-Neo4j or protocol-seam acceptance proves success, retry, stale-record, cancellation/error, and scope isolation while SQLite remains byte-for-byte unmodified and every non-projection Gortex operation works without Neo4j.

**Plans**: 7/7 plans executed

Plans:
**Wave 1**

- [x] 01-01-PLAN.md -- Wave 0 protocol, SQLite, and mandatory disposable-Neo4j harness
- [x] 01-02-PLAN.md -- Named profile and immutable exact-scope snapshot contracts

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-03-PLAN.md -- Real official-driver SQLite-to-Neo4j tracer with manifest activation

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-04-PLAN.md -- Stable identities, typed graph shape, and safe property envelope
- [x] 01-05-PLAN.md -- Immutable cancellable SQLite snapshot expansion and bounded batching

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-06-PLAN.md -- Full official-driver constraints, activation, bounded reconciliation, and acceptance

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 01-07-PLAN.md -- Equivalent CLI/MCP adapters, progress, parity, and final regression gate

### Phase 2: Thin Deterministic Native Tracer and Minimum Contracts

**Goal**: A user can retrieve one trustworthy named COBOL program observation through existing native Gortex surfaces after it traverses the real production indexing and SQLite path.
**Depends on**: Phase 1 and parser/grammar readiness
**Requirements**: BASE-01, BASE-02, BASE-03, TRACE-01, TRACE-02, TRACE-03, TRACE-04, TRACE-05, TRACE-06, TRACE-07, PROV-01, PROV-02, PROV-03, PROV-04, PROV-05, PROV-06, ID-01, ID-02
**Success Criteria** (what must be TRUE):

  1. A clean or `GOWORK=off` build uses the pinned `tree-sitter-cobol-upgrade` v0.26.0-line commit `f9eaf99c34a9` (behavioral floor `97ac9f1`), records its grammar content identifier, or fails instead of silently selecting the stock grammar.
  2. Indexing the tracer fixture emits one named COBOL program as a native node and its source-file relationship as a native edge through the registered extractor, `ExtractionResult`, repository prefixing, indexer lifecycle, and SQLite `AddBatch` transaction.
  3. The traced observation exposes stable repository-prefixed identity, exact source range, scope, revision, parser/extractor versions, evidence class, origin, and confidence without using line number as its sole identity discriminator.
  4. A user can retrieve the same persisted node through an existing CLI query and an existing MCP query.
  5. The tracer acceptance passes with all AI providers disabled and no Neo4j installation, with SQLite as the sole authoritative store and no deterministic dual-write.

**Plans**: 6/6 plans executed

Plans:
**Wave 1**

- [x] 02-01-PLAN.md -- Fork prerequisite: embedded grammar attestation, content-ID wrapper, pushed on the v0.26.0 line (D-03)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md -- Tracer: go.mod pin, grammar-only extractor, register swap, SQLite AddBatch to MCP/CLI get_symbol acceptance

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 02-03-PLAN.md -- Containment-path IDs, degraded/copybook/empty/concurrency guards, and fail-closed grammar checks
- [x] 02-04-PLAN.md -- VCS revision stamp in applyCoverageDomains and prov_* Neo4j projection contract test

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 02-05-PLAN.md -- CI read credential, disabled module caches, stock-grammar negative, credential-free release container

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 02-06-PLAN.md -- Requirement/roadmap/project amendments, doc scrub, full phase gate, CI secret and green-CI checkpoint

### Phase 3: Deterministic COBOL and Mainframe Breadth

**Goal**: Users can inspect source-positioned COBOL structure, control flow, data, calls, copybooks, IDMS, CICS, and SQL without false deterministic resolution.
**Depends on**: Phase 2
**Requirements**: DET-01, DET-02, DET-03, DET-04, DET-05, DET-06, DET-07, DET-08, DET-09, DET-10, DET-11, DET-12
**Success Criteria** (what must be TRUE):

  1. Extraction produces source-positioned programs, copybooks, divisions, sections, paragraphs, statements, and data items from named parser nodes.
  2. Supported `PERFORM`, `THRU`, and `GO TO` constructs produce paragraph control-flow relationships without claiming unsupported resolution.
  3. Literal calls, dynamic call identifiers, copybook references, missing targets, and ambiguous library resolution remain distinguishable in the graph.
  4. IDMS record/set and CICS program/transaction/map/mapset references retain exact source evidence.
  5. DB2 reads and writes retain statement-level evidence while aliases, CTEs, includes, and dynamic sources remain distinguishable from physical tables.

**Plans**: 17/17 plans executed (2 gap-closure plans pending)

Plans:
**Wave 1**

- [x] 03-01-PLAN.md -- Tracer: paragraphs and PERFORM through SQLite to get_callers; shared walk, ID grammar, edge provenance, dedupe, version /3
- [x] 03-02-PLAN.md -- CR-01 BOM byte offsets and VCS stamp on relationship edges in the indexer hook

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 03-03-PLAN.md -- Data/environment sections and all named statements (KindLocal) with statement provenance on edges
- [x] 03-04-PLAN.md -- Resolver: owned cobol:: namespace and same-repo exactly-one CALL/COPY binding pass

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 03-05-PLAN.md -- Data items: levels, FILLER/66/77/88, full-path IDs, REDEFINES/RENAMES, shared data lookup
- [x] 03-06-PLAN.md -- Control flow: qualified labels, PERFORM THRU expansion, GO TO and DEPENDING ON

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 03-07-PLAN.md -- Literal and dynamic CALL, COPY/-INC/COPY IDMS/SQL INCLUDE, binding keys, real-path binding test

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 03-08-PLAN.md -- IDMS record/set and CICS program/transaction/map/mapset references

**Wave 6** *(blocked on Wave 5 completion)*

- [x] 03-09-PLAN.md -- DB2 SQL reads/writes, aliases, CTEs, cursors, dynamic SQL

**Wave 7** *(blocked on Wave 6 completion)*

- [x] 03-10-PLAN.md -- Two-repository real-path acceptance, contract tests, Neo4j key check, language docs

**Wave 8** *(blocked on Wave 7 completion)*

- [x] 03-11-PLAN.md -- Fork prerequisite (D-16): copybook analysis mode, committed locally

**Wave 9** *(blocked on Wave 8 completion)*

- [x] 03-12-PLAN.md -- User-confirmed push of the copybook-mode fork commit

**Wave 10** *(blocked on Wave 9 completion)*

- [x] 03-13-PLAN.md -- Preprocessor-only re-pin, copybook-owned data items, phase gate

**Wave 11** *(gap closure; blocked on Wave 10 completion)*

- [x] 03-14-PLAN.md -- Gap 1: nested programs are CALL candidates only in their own file (cobol_nested stamp + resolver rule)

**Wave 12** *(gap closure; blocked on Wave 11 completion)*

- [x] 03-15-PLAN.md -- Gap 2: COPY/-INC/COPY IDMS/SQL INCLUDE inside copybooks become imports edges; copybook root ownership

**Wave 13** *(gap closure; blocked on Wave 12 completion)*

- [x] 03-16-PLAN.md -- Procedure copybooks: paragraphs, sections, statements, PERFORM/THRU/GO TO, CALL and COPY owned by the copybook

**Wave 14** *(gap closure; blocked on Wave 13 completion)*

- [x] 03-17-PLAN.md -- Procedure copybooks: IDMS/CICS/SQL, not_declared_in_copybook, WR-04 window stamps, docs, phase gate

### Phase 4: Stable Incremental and Cross-Repository Lifecycle

**Goal**: Users receive the same canonical scoped deterministic graph from identical final inputs regardless of reruns, edits, deletions, restores, or clean rebuilds.
**Depends on**: Phase 3
**Requirements**: ID-03, ID-04, ID-05, ID-06, ID-07, ID-08, ID-09, SEC-01, SEC-02, SEC-03, SEC-08
**Success Criteria** (what must be TRUE):

  1. Program-owned paragraphs, namespaced resources, and repeated relationship occurrences retain stable, non-colliding identities and distinguishable provenance.
  2. Reindexing identical inputs reuses active deterministic identities and creates no duplicate active facts.
  3. Edit, delete, restore, parser-version, extractor-version, retrieval, and material-configuration changes remove or invalidate stale owned observations.
  4. Incremental reconciliation and a clean rebuild produce the same canonical active deterministic subgraph for identical final inputs.
  5. Resolution and all downstream surfaces fail closed on missing scope, reject unsupported same-name cross-repository binding, and report zero unowned or misprefixed active nodes for the acceptance corpus.

**Plans**: 22/22 plans executed (4 gap-closure plans pending, round 2)

Plans 04-11 to 04-16 are the second halves of 04-01, 04-03, 04-05, 04-06, 04-08, and 04-09, split so every plan fits the 100k smart-zone budget; run them in wave order, not number order.

Plans:
**Wave 1**

- [x] 04-01-PLAN.md -- Tracer: canonical dump + BLAKE3 digest, real-path harness, deterministic clean builds (CR-02 shared resources)
- [x] 04-02-PLAN.md -- Extractor identity: encoded program IDs (IN-03), cobol_placeholder and library key, occurrence provenance proof
- [x] 04-03-PLAN.md -- One scope type (graph.Scope) for projection and export (D-16)
- [x] 04-04-PLAN.md -- Generic cross-repository ambiguity refusal in function and method fallbacks (D-12)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 04-11-PLAN.md -- Extractor /5 bump; shared-resource and canonical-dump contract tests
- [x] 04-12-PLAN.md -- Fail-closed source read/write gate for multi-repository sessions (D-13, D-20)
- [x] 04-06-PLAN.md -- One COBOL reconcile pass: re-decided bindings, caller-scoped through graph.Scope, idempotent

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 04-05-PLAN.md -- Strict ownership audit, enumerated exemptions, index_health status, v22 purge
- [x] 04-14-PLAN.md -- Reconcile policy: directed allowlist with union ambiguity, library order, qualifier links, orphan GC

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 04-07-PLAN.md -- index.cobol config, validation, policy lookup wiring, live reload, cross-repository clean builds
- [x] 04-13-PLAN.md -- Embedded gortex mcp --index as a workspace of one through MultiIndexer

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 04-08-PLAN.md -- Incremental tracer: CR-01 restub, reconcile at every mutation tail, rerun/edit/delete/restore sequences

**Wave 6** *(blocked on Wave 5 completion)*

- [x] 04-15-PLAN.md -- Warm-restart reconcile, affected-by escalation (D-08), cross-file, cross-repository, interrupted, and over-cap sequences

**Wave 7** *(blocked on Wave 6 completion)*

- [x] 04-09-PLAN.md -- Material fingerprint (D-06/D-07) in RepoIndexState, config-change sequences

**Wave 8** *(blocked on Wave 7 completion)*

- [x] 04-16-PLAN.md -- Merkle salt, extractor-identity change sequence, VCS re-stamp on HEAD move (D-17)

**Wave 9** *(blocked on Wave 8 completion)*

- [x] 04-10-PLAN.md -- Cross-surface scope proof, ownership acceptance, D-04 completeness, Meta-key screen, docs, full race suite

**Wave 10** *(gap closure; blocked on Wave 9 completion; 04-17 and 04-18 share no file and run in parallel)*

- [x] 04-17-PLAN.md -- Gap CR-01 (surfaces): resolveNodePath, the relative resolveGraphPath branch, search_text, and find_declaration gate by the physical owner (requirePathSourceScope); nested-checkout test in both indexing orders
- [x] 04-18-PLAN.md -- Gap CR-01 (root cause): walk and watcher stop at a nested tracked root (config seed + registered roots), so each file has one owner and the ownership audit stays honest

**Wave 11** *(gap closure round 2; blocked on Wave 10 completion; 04-19 and 04-20 share no file and run in parallel)*

- [x] 04-19-PLAN.md -- Gap 1 (SEC-01/SEC-02): explore artifact lane refuses unbound multi-repo requests and admits matches by the physical owner; parse-level source-channel inventory test; source-surface audit
- [x] 04-20-PLAN.md -- Gap 2 seed (SEC-08/ID-09): nested-root pruning reads declared batch roots plus live registered roots (DeclarePendingRoots in indexMultiRepo and the daemon warmup); no per-Indexer config copy

**Wave 12** *(blocked on Wave 11 completion; 04-21 and 04-22 share no file and run in parallel)*

- [x] 04-21-PLAN.md -- Gap 2 topology (ID-09/SEC-08): track of a nested root evicts the enclosing repository's copies and untrack admits them (reconcileEnclosingRepos); COBOL graphdump equivalence; MCP window fixture via ReconcileRepoCtx
- [x] 04-22-PLAN.md -- Audit follow-up (SEC-01/SEC-02): per-entry source gate in move_symbol, inline_symbol and generate_skill walks; find_declaration regex filter-before-limit (WR-05)

### Phase 5: Explicit Unresolved Evidence

**Goal**: Users can see and distinguish source that could not be parsed from referenced artifacts that were not retrieved, without fabricated evidence.
**Depends on**: Phase 4
**Requirements**: GAP-01, GAP-02, GAP-03, GAP-04, GAP-05, GAP-06, GAP-07, GAP-08
**Success Criteria** (what must be TRUE):

  1. Named unparsed tails and localized `ERROR` or `MISSING` nodes produce stable, source-positioned `PARSE_UNRESOLVED` findings, while unavailable targets produce separate `EXTERNAL_UNRESOLVED` findings.
  2. Each finding exposes stable identity, exact scope and range, containing symbol, resolution class, normalized reason, severity, parser version, and lifecycle status.
  3. Missing-target placeholders preserve the observed target reference without fabricated source, body, or attributes.
  4. Unchanged evidence reuses finding identities; corrected source/parser evidence or retrieved artifacts close or supersede only the applicable finding class.
  5. Destructive parser-recovery cascades remain deterministic regression failures regardless of any later AI output.

**Plans**: 4/12 plans executed

Plans:
**Wave 1**

- [x] 05-01-PLAN.md -- Tracer: finding kind, D-01 identity, program parse findings through SQLite to MCP/CLI get_symbol and search

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 05-02-PLAN.md -- Kind-guarded EvictFindingNodesByIDs on both backends, generation fence, storetest conformance
- [x] 05-03-PLAN.md -- Uncontained-cascade document finding (D-13), copybook findings, extractor version /6
- [x] 05-04-PLAN.md -- index_health finding counts by class (never scored), projection Meta proof, schema resource entry

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 05-05-PLAN.md -- Resolver: missing and IDMS edges retarget onto owned missing-artifact placeholders; downstream expectations
- [ ] 05-06-PLAN.md -- Ledger diff (close vs supersede) and finding_history sidecar with latest-row index, schema v23
- [ ] 05-07-PLAN.md -- D-14 cascade regression gate on the real path in the default race suite

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 05-08-PLAN.md -- Resolver: EXTERNAL_UNRESOLVED findings with the full Pattern 3 Meta contract; real-path persistence

**Wave 5** *(blocked on Wave 4 completion)*

- [ ] 05-09-PLAN.md -- Resolver change record and durable ledger at every COBOL settle point, including dependent repositories, batch-deferred tracks, and cold shadow-path tracks re-decided on the durable store; real-path history sequences
- [ ] 05-10-PLAN.md -- Deferred Phase 3 reasons: WR-06 copybook-only analysis failure (D-20), WR-07, WR-09, IN-08

**Wave 6** *(blocked on Wave 5 completion)*

- [ ] 05-11-PLAN.md -- Search sync for resolver-created nodes (incl. dependent repositories and fallback master resolves), external real-path lifecycle, ledger scale benchmark

**Wave 7** *(blocked on Wave 6 completion)*

- [ ] 05-12-PLAN.md -- External/placeholder MCP+CLI acceptance, canonical equivalence, cross-container ledger case, D-16 writer inventory, docs (incl. D-20 limitation), full race gate

### Phase 6: Native Deterministic Acceptance and Pre-AI Release Gate

**Goal**: Users can answer the required native graph questions, and the deterministic system demonstrably satisfies reproducibility, lifecycle, monitoring, and unresolved-boundary invariants before AI is introduced.
**Depends on**: Phase 5
**Requirements**: BASE-04, QUERY-01, QUERY-02, QUERY-03, QUERY-04, QUERY-05, QUERY-06, QUERY-07, QUERY-08, QUERY-09, QUERY-10, EVAL-01, EVAL-02, EVAL-03, EVAL-06
**Success Criteria** (what must be TRUE):

  1. Committed fixtures pass the required named COBOL, DATA DIVISION, IDMS, CICS, SQL, bounded-tail, `ERROR`, `MISSING`, and missing-artifact acceptance cases using canonical identities and provenance.
  2. Native CLI and MCP queries answer program, copybook, paragraph-path, caller, data-item, DB2, IDMS, CICS, unresolved-finding, and unavailable-artifact impact questions with exact evidence.
  3. Query results disclose unresolved boundaries and identify affected counts or paths as lower bounds rather than presenting incomplete results as complete.
  4. Identical rerun, edit, delete, restore, parser-version change, and interrupted-retry acceptance sequences prove canonical active deterministic equivalence.
  5. Monitoring reports deterministic extraction failures and unresolved findings by class and reason, and all deterministic release gates pass with AI disabled before Phase 7 begins.

**Plans**: TBD

### Phase 06.1: Deterministic VB.NET Extraction via Pinned tree-sitter-vbnet Grammar (INSERTED)

**Goal:** A user can index VB.NET source into the same authoritative SQLite graph as the mainframe estate, with source-positioned declarations and references extracted deterministically from the user's `MuiGoku123432/tree-sitter-vbnet` grammar and retrievable through existing CLI and MCP surfaces.
**Requirements**: TBD (to be defined at discuss/plan time; reuse the Phase 2 BASE/PROV/ID contract shapes for VB.NET)
**Depends on:** Phase 6 (deterministic gate) and the Phase 2 pinned-grammar pattern
**Success Criteria** (what must be TRUE):

  1. The VB.NET grammar ships Go bindings and is pinned in `go.mod` at an exact commit, so a `GOWORK=off` build is reproducible; a missing or mismatched grammar fails VB.NET extraction instead of silently degrading.
  2. A registered VB.NET extractor emits namespaces, modules, classes, structures, interfaces, enums, and methods/properties as native nodes with repository-prefixed identity and exact source ranges, through the existing `ExtractionResult`, indexer lifecycle, and SQLite path.
  3. `Imports`, inheritance/`Implements`, and call references produce edges without claiming resolution the grammar cannot support; unresolved targets stay distinguishable.
  4. `.vbproj` projects are recognised alongside existing `.sln`/`.csproj` handling.
  5. Acceptance passes on invented VB.NET fixtures with AI providers disabled and no Neo4j installation.

**Plans:** 0 plans

Plans:

- [ ] TBD (run /gsd-plan-phase 06.1 to break down)

### Phase 7: AI Contract, Policy, and Evaluation Foundation

**Goal**: An explicitly authorized user can prepare a minimal, redacted, reproducible AI review request without weakening AI-off behavior or exposing proprietary source by default.
**Depends on**: Phase 6
**Requirements**: AI-01, AI-02, AI-03, AI-04, AI-05, AI-06, AI-07, AI-08, AI-09, SEC-04, SEC-05, SEC-07
**Success Criteria** (what must be TRUE):

  1. AI remains disabled by default, and disabling every provider leaves deterministic indexing, reconciliation, persistence, and native queries fully functional.
  2. Provider connectivity alone cannot transmit source; each request requires explicit provider/model, scope, data-class, and retention authorization through the existing Gortex provider abstraction.
  3. A context manifest deterministically identifies bounded in-scope evidence, omitted or unavailable context, redactions, and a digest before provider dispatch.
  4. Default logs and persisted audit metadata contain no raw prompts, source context, or responses; confidential audit files use restrictive permissions and policy-bounded retention.
  5. Policy denials and provider or audit-path failures are observable with provider/model/policy/scope metadata but without protected source.

**Plans**: TBD
**AI-SPEC workflow**: Required before implementation planning.

### Phase 8: Evidence-Linked Claims and Human Review

**Goal**: Users can opt into validated AI claims, inspect their evidence and lineage, and review them without changing deterministic graph truth.
**Depends on**: Phase 7
**Requirements**: CLAIM-01, CLAIM-02, CLAIM-03, CLAIM-04, CLAIM-05, CLAIM-06, CLAIM-07, CLAIM-08, CLAIM-09, REVIEW-01, REVIEW-02, REVIEW-03, REVIEW-04, REVIEW-05, REVIEW-06, QUERY-11
**Success Criteria** (what must be TRUE):

  1. Only output conforming to a closed versioned schema and citing existing in-scope graph IDs or exact source ranges can become an `AI_INFERRED` claim.
  2. Each claim exposes stable identity, assertion, confidence, rationale, missing context, evidence, and provider/model/prompt/context/schema lineage; identical retries do not duplicate active claims.
  3. Deterministic and trusted views exclude unreviewed claims by default, and AI-on versus AI-off operation leaves the canonical deterministic subgraph unchanged.
  4. An authorized reviewer can confirm, contradict, reject, or supersede a claim while retaining reviewer identity, timestamp, rationale, originating claim, and model provenance.
  5. Users can distinguish deterministic, unresolved, AI-inferred, human-confirmed, contradicted, stale, rejected, and superseded outcomes; conflicting claims coexist without last-write-wins domain mutation.

**Plans**: TBD
**AI-SPEC workflow**: Required before implementation planning.

### Phase 9: End-to-End AI Evaluation and Projection Safety

**Goal**: AI behavior and all external disclosure boundaries pass representative evaluation without changing SQLite authority or widening the approved manual projection into synchronization.
**Depends on**: Phase 8
**Requirements**: EVAL-04, EVAL-05, EVAL-07, EVAL-08, EVAL-09, SEC-06
**Success Criteria** (what must be TRUE):

  1. A curated labeled AI reference set meets documented thresholds and reports schema validity, citation validity, unsupported-claim rate, and review outcome distribution before claims leave experimental use.
  2. Monitoring reports AI policy, validation, and provider failures plus claim lifecycle counts without logging proprietary source; recurring gap families are ranked by frequency and graph impact.
  3. End-to-end canaries prove protected source cannot escape through unauthorized provider requests, logs, claims, temporary files, or Cypher exports, and AI-on/off runs produce identical active deterministic subgraphs.
  4. End-to-end acceptance confirms SQLite remains authoritative, projection is manually invoked and one-way, and continuous synchronization or reverse writes remain absent.

**Plans**: TBD
**AI-SPEC workflow**: Required before implementation planning because this phase evaluates AI behavior and policy invariants.

## Progress

**Execution Order:**
Phase 1 starts immediately. Phases 2-9 retain numeric dependency order but remain deferred until parser/grammar readiness: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. General Neo4j Projection Command | 7/7 | Complete    | 2026-09-22 |
| 2. Thin Deterministic Native Tracer and Minimum Contracts | 6/6 | Complete    | 2026-10-01 |
| 3. Deterministic COBOL and Mainframe Breadth | 17/17 | In Progress|  |
| 4. Stable Incremental and Cross-Repository Lifecycle | 22/22 | In Progress|  |
| 5. Explicit Unresolved Evidence | 4/12 | In Progress|  |
| 6. Native Deterministic Acceptance and Pre-AI Release Gate | 0/TBD | Deferred: Phase 5 | - |
| 7. AI Contract, Policy, and Evaluation Foundation | 0/TBD | Deferred: Phase 6 | - |
| 8. Evidence-Linked Claims and Human Review | 0/TBD | Deferred: Phase 7 | - |
| 9. End-to-End AI Evaluation and Projection Safety | 0/TBD | Deferred: Phase 8 | - |
