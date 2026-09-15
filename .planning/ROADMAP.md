# Roadmap: Gortex Mainframe Engine v1.0

## Overview

This milestone begins with a working thin deterministic tracer, carrying one named COBOL program through the real parser, extractor, SQLite, CLI, and MCP path under the minimum contracts needed to trust that result. It then expands deterministic mainframe coverage, stabilizes identity and lifecycle behavior, models unresolved evidence explicitly, and passes a native deterministic release gate before any AI work begins. The final three phases retain the existing AI policy, claims/review, and evaluation/storage-projection roles.

## Phases

**Phase Numbering:**
- Integer phases are planned milestone work.
- Decimal phases are reserved for urgent insertions.

- [ ] **Phase 1: Thin Deterministic Native Tracer and Minimum Contracts** - Deliver one trustworthy native COBOL trace through production persistence and query surfaces.
- [ ] **Phase 2: Deterministic COBOL and Mainframe Breadth** - Expand native extraction across COBOL structure, control flow, data, calls, copybooks, IDMS, CICS, and SQL.
- [ ] **Phase 3: Stable Incremental and Cross-Repository Lifecycle** - Make identity, reconciliation, and scoped resolution stable across reruns, changes, and repositories.
- [ ] **Phase 4: Explicit Unresolved Evidence** - Persist parser gaps and unavailable artifacts as distinct, lifecycle-aware deterministic findings.
- [ ] **Phase 5: Native Deterministic Acceptance and Pre-AI Release Gate** - Prove native queries, deterministic fixtures, lifecycle equivalence, monitoring, and unresolved-boundary behavior before AI.
- [ ] **Phase 6: AI Contract, Policy, and Evaluation Foundation** - Define and enforce the bounded, redacted, provider-authorized AI boundary before claim implementation.
- [ ] **Phase 7: Evidence-Linked Claims and Human Review** - Persist validated AI claims separately and support auditable review and lifecycle transitions.
- [ ] **Phase 8: End-to-End Evaluation and Storage Projection Decision** - Prove AI release invariants and decide the projection boundary from measured evidence while keeping SQLite authoritative.

## Phase Details

### Phase 1: Thin Deterministic Native Tracer and Minimum Contracts
**Goal**: A user can retrieve one trustworthy named COBOL program observation through existing native Gortex surfaces after it traverses the real production indexing and SQLite path.
**Depends on**: Nothing (first phase)
**Requirements**: BASE-01, BASE-02, BASE-03, TRACE-01, TRACE-02, TRACE-03, TRACE-04, TRACE-05, TRACE-06, TRACE-07, PROV-01, PROV-02, PROV-03, PROV-04, PROV-05, PROV-06, ID-01, ID-02, STORE-01, STORE-02
**Success Criteria** (what must be TRUE):
  1. A clean or `GOWORK=off` build uses parser behavior equivalent to baseline `97ac9f1`, reports its grammar content identifier, or fails instead of silently selecting the stock grammar.
  2. Indexing the tracer fixture emits one named COBOL program as a native node and its source-file relationship as a native edge through the registered extractor, `ExtractionResult`, repository prefixing, indexer lifecycle, and SQLite `AddBatch` transaction.
  3. The traced observation exposes stable repository-prefixed identity, exact source range, scope, revision, parser/extractor versions, evidence class, origin, and confidence without using line number as its sole identity discriminator.
  4. A user can retrieve the same persisted node through an existing CLI query and an existing MCP query.
  5. The tracer acceptance passes with all AI providers disabled and no Neo4j installation, with SQLite as the sole authoritative store and no deterministic dual-write.
**Plans**: TBD

### Phase 2: Deterministic COBOL and Mainframe Breadth
**Goal**: Users can inspect source-positioned COBOL structure, control flow, data, calls, copybooks, IDMS, CICS, and SQL without false deterministic resolution.
**Depends on**: Phase 1
**Requirements**: DET-01, DET-02, DET-03, DET-04, DET-05, DET-06, DET-07, DET-08, DET-09, DET-10, DET-11, DET-12
**Success Criteria** (what must be TRUE):
  1. Extraction produces source-positioned programs, copybooks, divisions, sections, paragraphs, statements, and data items from named parser nodes.
  2. Supported `PERFORM`, `THRU`, and `GO TO` constructs produce paragraph control-flow relationships without claiming unsupported resolution.
  3. Literal calls, dynamic call identifiers, copybook references, missing targets, and ambiguous library resolution remain distinguishable in the graph.
  4. IDMS record/set and CICS program/transaction/map/mapset references retain exact source evidence.
  5. DB2 reads and writes retain statement-level evidence while aliases, CTEs, includes, and dynamic sources remain distinguishable from physical tables.
**Plans**: TBD

### Phase 3: Stable Incremental and Cross-Repository Lifecycle
**Goal**: Users receive the same canonical scoped deterministic graph from identical final inputs regardless of reruns, edits, deletions, restores, or clean rebuilds.
**Depends on**: Phase 2
**Requirements**: ID-03, ID-04, ID-05, ID-06, ID-07, ID-08, ID-09, SEC-01, SEC-02, SEC-03, SEC-08
**Success Criteria** (what must be TRUE):
  1. Program-owned paragraphs, namespaced resources, and repeated relationship occurrences retain stable, non-colliding identities and distinguishable provenance.
  2. Reindexing identical inputs reuses active deterministic identities and creates no duplicate active facts.
  3. Edit, delete, restore, parser-version, extractor-version, retrieval, and material-configuration changes remove or invalidate stale owned observations.
  4. Incremental reconciliation and a clean rebuild produce the same canonical active deterministic subgraph for identical final inputs.
  5. Resolution and all downstream surfaces fail closed on missing scope, reject unsupported same-name cross-repository binding, and report zero unowned or misprefixed active nodes for the acceptance corpus.
**Plans**: TBD

### Phase 4: Explicit Unresolved Evidence
**Goal**: Users can see and distinguish source that could not be parsed from referenced artifacts that were not retrieved, without fabricated evidence.
**Depends on**: Phase 3
**Requirements**: GAP-01, GAP-02, GAP-03, GAP-04, GAP-05, GAP-06, GAP-07, GAP-08
**Success Criteria** (what must be TRUE):
  1. Named unparsed tails and localized `ERROR` or `MISSING` nodes produce stable, source-positioned `PARSE_UNRESOLVED` findings, while unavailable targets produce separate `EXTERNAL_UNRESOLVED` findings.
  2. Each finding exposes stable identity, exact scope and range, containing symbol, resolution class, normalized reason, severity, parser version, and lifecycle status.
  3. Missing-target placeholders preserve the observed target reference without fabricated source, body, or attributes.
  4. Unchanged evidence reuses finding identities; corrected source/parser evidence or retrieved artifacts close or supersede only the applicable finding class.
  5. Destructive parser-recovery cascades remain deterministic regression failures regardless of any later AI output.
**Plans**: TBD

### Phase 5: Native Deterministic Acceptance and Pre-AI Release Gate
**Goal**: Users can answer the required native graph questions, and the deterministic system demonstrably satisfies reproducibility, lifecycle, monitoring, and unresolved-boundary invariants before AI is introduced.
**Depends on**: Phase 4
**Requirements**: BASE-04, QUERY-01, QUERY-02, QUERY-03, QUERY-04, QUERY-05, QUERY-06, QUERY-07, QUERY-08, QUERY-09, QUERY-10, EVAL-01, EVAL-02, EVAL-03, EVAL-06
**Success Criteria** (what must be TRUE):
  1. Committed fixtures pass the required named COBOL, DATA DIVISION, IDMS, CICS, SQL, bounded-tail, `ERROR`, `MISSING`, and missing-artifact acceptance cases using canonical identities and provenance.
  2. Native CLI and MCP queries answer program, copybook, paragraph-path, caller, data-item, DB2, IDMS, CICS, unresolved-finding, and unavailable-artifact impact questions with exact evidence.
  3. Query results disclose unresolved boundaries and identify affected counts or paths as lower bounds rather than presenting incomplete results as complete.
  4. Identical rerun, edit, delete, restore, parser-version change, and interrupted-retry acceptance sequences prove canonical active deterministic equivalence.
  5. Monitoring reports deterministic extraction failures and unresolved findings by class and reason, and all deterministic release gates pass with AI disabled before Phase 6 begins.
**Plans**: TBD

### Phase 6: AI Contract, Policy, and Evaluation Foundation
**Goal**: An explicitly authorized user can prepare a minimal, redacted, reproducible AI review request without weakening AI-off behavior or exposing proprietary source by default.
**Depends on**: Phase 5
**Requirements**: AI-01, AI-02, AI-03, AI-04, AI-05, AI-06, AI-07, AI-08, AI-09, SEC-04, SEC-05, SEC-07
**Success Criteria** (what must be TRUE):
  1. AI remains disabled by default, and disabling every provider leaves deterministic indexing, reconciliation, persistence, and native queries fully functional.
  2. Provider connectivity alone cannot transmit source; each request requires explicit provider/model, scope, data-class, and retention authorization through the existing Gortex provider abstraction.
  3. A context manifest deterministically identifies bounded in-scope evidence, omitted or unavailable context, redactions, and a digest before provider dispatch.
  4. Default logs and persisted audit metadata contain no raw prompts, source context, or responses; confidential audit files use restrictive permissions and policy-bounded retention.
  5. Policy denials and provider or audit-path failures are observable with provider/model/policy/scope metadata but without protected source.
**Plans**: TBD
**AI-SPEC workflow**: Required before implementation planning.

### Phase 7: Evidence-Linked Claims and Human Review
**Goal**: Users can opt into validated AI claims, inspect their evidence and lineage, and review them without changing deterministic graph truth.
**Depends on**: Phase 6
**Requirements**: CLAIM-01, CLAIM-02, CLAIM-03, CLAIM-04, CLAIM-05, CLAIM-06, CLAIM-07, CLAIM-08, CLAIM-09, REVIEW-01, REVIEW-02, REVIEW-03, REVIEW-04, REVIEW-05, REVIEW-06, QUERY-11
**Success Criteria** (what must be TRUE):
  1. Only output conforming to a closed versioned schema and citing existing in-scope graph IDs or exact source ranges can become an `AI_INFERRED` claim.
  2. Each claim exposes stable identity, assertion, confidence, rationale, missing context, evidence, and provider/model/prompt/context/schema lineage; identical retries do not duplicate active claims.
  3. Deterministic and trusted views exclude unreviewed claims by default, and AI-on versus AI-off operation leaves the canonical deterministic subgraph unchanged.
  4. An authorized reviewer can confirm, contradict, reject, or supersede a claim while retaining reviewer identity, timestamp, rationale, originating claim, and model provenance.
  5. Users can distinguish deterministic, unresolved, AI-inferred, human-confirmed, contradicted, stale, rejected, and superseded outcomes; conflicting claims coexist without last-write-wins domain mutation.
**Plans**: TBD
**AI-SPEC workflow**: Required before implementation planning.

### Phase 8: End-to-End Evaluation and Storage Projection Decision
**Goal**: AI behavior and disclosure boundaries pass representative evaluation, and the storage projection boundary is decided from measured evidence without adding Neo4j synchronization.
**Depends on**: Phase 7
**Requirements**: STORE-03, STORE-04, STORE-05, EVAL-04, EVAL-05, EVAL-07, EVAL-08, EVAL-09, SEC-06
**Success Criteria** (what must be TRUE):
  1. A curated labeled AI reference set meets documented thresholds and reports schema validity, citation validity, unsupported-claim rate, and review outcome distribution before claims leave experimental use.
  2. Monitoring reports AI policy, validation, and provider failures plus claim lifecycle counts without logging proprietary source; recurring gap families are ranked by frequency and graph impact.
  3. End-to-end canaries prove protected source cannot escape through unauthorized provider requests, logs, claims, temporary files, or Cypher exports, and AI-on/off runs produce identical active deterministic subgraphs.
  4. Measured native query and estate-scale evidence produces a final storage projection ADR; SQLite remains authoritative, Cypher remains a scoped one-way rebuildable snapshot, and no v1.0 acceptance path requires Neo4j availability, synchronization, or reverse writes.
**Plans**: TBD
**AI-SPEC workflow**: Required before implementation planning because this phase evaluates AI behavior and policy invariants.

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Thin Deterministic Native Tracer and Minimum Contracts | 0/TBD | Not started | - |
| 2. Deterministic COBOL and Mainframe Breadth | 0/TBD | Not started | - |
| 3. Stable Incremental and Cross-Repository Lifecycle | 0/TBD | Not started | - |
| 4. Explicit Unresolved Evidence | 0/TBD | Not started | - |
| 5. Native Deterministic Acceptance and Pre-AI Release Gate | 0/TBD | Not started | - |
| 6. AI Contract, Policy, and Evaluation Foundation | 0/TBD | Not started | - |
| 7. Evidence-Linked Claims and Human Review | 0/TBD | Not started | - |
| 8. End-to-End Evaluation and Storage Projection Decision | 0/TBD | Not started | - |
