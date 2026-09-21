# Roadmap: Gortex Mainframe Engine v1.0

## Overview

This milestone now begins with a general, manually invoked projection that materializes an explicitly scoped snapshot of Gortex's authoritative SQLite graph into Neo4j through equivalent CLI and MCP surfaces. The projection is language-agnostic, retry-safe, and rebuildable; it never joins indexing, writes back, or changes SQLite authority. The existing COBOL tracer, grammar-dependent extraction, unresolved-evidence, and AI phases remain coherent but are deferred until parser/grammar readiness.

## Phases

**Phase Numbering:**
- Integer phases are planned milestone work.
- Decimal phases are reserved for urgent insertions.

- [ ] **Phase 1: General Neo4j Projection Command** - Materialize a selected SQLite graph snapshot into Neo4j through equivalent explicit CLI and MCP operations.
- [ ] **Phase 2: Thin Deterministic Native Tracer and Minimum Contracts** - Deliver one trustworthy native COBOL trace after parser/grammar readiness.
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
**Plans**: 6 plans

Plans:
- [ ] 01-01-PLAN.md -- Wave 0 protocol, SQLite, and mandatory disposable-Neo4j harness
- [ ] 01-02-PLAN.md -- Real production SQLite-to-Neo4j tracer with manifest activation
- [ ] 01-03-PLAN.md -- Stable identities, typed graph shape, and safe property envelope
- [ ] 01-04-PLAN.md -- Immutable cancellable SQLite snapshot and bounded batching
- [ ] 01-05-PLAN.md -- Full official-driver constraints, activation, bounded reconciliation, and acceptance
- [ ] 01-06-PLAN.md -- Equivalent CLI/MCP adapters, progress, parity, and final regression gate

### Phase 2: Thin Deterministic Native Tracer and Minimum Contracts
**Goal**: A user can retrieve one trustworthy named COBOL program observation through existing native Gortex surfaces after it traverses the real production indexing and SQLite path.
**Depends on**: Phase 1 and parser/grammar readiness
**Requirements**: BASE-01, BASE-02, BASE-03, TRACE-01, TRACE-02, TRACE-03, TRACE-04, TRACE-05, TRACE-06, TRACE-07, PROV-01, PROV-02, PROV-03, PROV-04, PROV-05, PROV-06, ID-01, ID-02
**Success Criteria** (what must be TRUE):
  1. A clean or `GOWORK=off` build uses parser behavior equivalent to baseline `97ac9f1`, reports its grammar content identifier, or fails instead of silently selecting the stock grammar.
  2. Indexing the tracer fixture emits one named COBOL program as a native node and its source-file relationship as a native edge through the registered extractor, `ExtractionResult`, repository prefixing, indexer lifecycle, and SQLite `AddBatch` transaction.
  3. The traced observation exposes stable repository-prefixed identity, exact source range, scope, revision, parser/extractor versions, evidence class, origin, and confidence without using line number as its sole identity discriminator.
  4. A user can retrieve the same persisted node through an existing CLI query and an existing MCP query.
  5. The tracer acceptance passes with all AI providers disabled and no Neo4j installation, with SQLite as the sole authoritative store and no deterministic dual-write.
**Plans**: TBD

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
**Plans**: TBD

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
**Plans**: TBD

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
**Plans**: TBD

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
| 1. General Neo4j Projection Command | 0/TBD | Not started | - |
| 2. Thin Deterministic Native Tracer and Minimum Contracts | 0/TBD | Deferred: parser/grammar readiness | - |
| 3. Deterministic COBOL and Mainframe Breadth | 0/TBD | Deferred: parser/grammar readiness | - |
| 4. Stable Incremental and Cross-Repository Lifecycle | 0/TBD | Deferred: Phase 3 | - |
| 5. Explicit Unresolved Evidence | 0/TBD | Deferred: Phase 4 | - |
| 6. Native Deterministic Acceptance and Pre-AI Release Gate | 0/TBD | Deferred: Phase 5 | - |
| 7. AI Contract, Policy, and Evaluation Foundation | 0/TBD | Deferred: Phase 6 | - |
| 8. Evidence-Linked Claims and Human Review | 0/TBD | Deferred: Phase 7 | - |
| 9. End-to-End AI Evaluation and Projection Safety | 0/TBD | Deferred: Phase 8 | - |
