# Requirements: v1.0 Neo4j Projection and Mainframe Graph Foundation

**Defined:** 2026-09-15
**Status:** Draft for roadmap planning
**Source baseline:** `tree-sitter-cobol-upgrade/main` merge `97ac9f1`

## Product Principle

Deterministic extraction establishes facts. Missing evidence remains explicit. AI may propose evidence-linked claims, but it must not rewrite parser truth, invent unavailable source, or become a dependency of indexing and deterministic queries.

## Current Milestone

### General Neo4j Projection

- [x] **NEO-01**: A user MUST be able to explicitly invoke a language-agnostic projection through both a CLI command and an MCP tool with equivalent scope and behavior.
- [x] **NEO-02**: The projection MUST read a selected snapshot from authoritative SQLite under explicit workspace, project, and repository scope and MUST fail closed when required scope is absent or ambiguous.
- [x] **NEO-03**: Neo4j connection and credential configuration MUST be explicit, and credentials or secret-bearing connection details MUST NOT appear in command output, logs, plans, progress, or result summaries.
- [x] **NEO-04**: Projected nodes MUST use stable projection keys derived from authoritative Gortex identity rather than language-specific or Neo4j-generated identity.
- [x] **NEO-05**: Projected relationships MUST use stable projection keys that preserve endpoint identity, relationship kind, and occurrence/provenance distinctions needed to avoid collapsing material evidence.
- [x] **NEO-06**: The projection MUST establish or verify the Neo4j constraints required to enforce its stable node and relationship identity contract before applying records.
- [x] **NEO-07**: Projection writes MUST use idempotent `MERGE`/upsert semantics so retrying identical work does not create duplicate nodes or relationships.
- [x] **NEO-08**: Projection work MUST use bounded batches and transactions rather than loading an unbounded selected graph in one transaction.
- [x] **NEO-09**: A dry-run or plan mode MUST report intended scope and materialization actions without mutating Neo4j or SQLite.
- [x] **NEO-10**: An invoked projection MUST expose progress and a final result summary with scoped counts and failures while redacting secrets.
- [x] **NEO-11**: Cancellation or a Neo4j error MUST stop further work promptly, leave SQLite unchanged, and return an actionable non-success result.
- [x] **NEO-12**: Partial failure and retry MUST be safe: completed batches may be replayed without duplication, and the result MUST identify incomplete work without claiming a complete snapshot.
- [x] **NEO-13**: Projected records MUST preserve selected repository/workspace/project scope plus relevant provenance, evidence, origin, confidence, lifecycle, and source-location properties present in SQLite.
- [x] **NEO-14**: Stale records for the selected scope MUST be identifiable and reconcilable through explicit snapshot ownership/generation or equivalent replace-scope semantics without affecting records outside that scope.
- [x] **NEO-15**: Projection MUST never mutate SQLite, accept reverse writes from Neo4j, or participate in synchronous index-time dual-write.
- [x] **NEO-16**: Automated acceptance MUST exercise success, retry, stale-record, cancellation/error, and scope-isolation behavior against a disposable Neo4j instance or a protocol seam with equivalent observable guarantees.
- [x] **NEO-17**: Neo4j availability MUST be required only when the explicit projection command/tool or its integration tests run; indexing, native queries, daemon startup, and all other Gortex operations MUST remain functional without Neo4j.

### Parser Baseline

- [x] **BASE-01**: The production COBOL extraction path MUST use parser behavior reproducibly equivalent to `tree-sitter-cobol-upgrade/main` merge `97ac9f1`.
- [x] **BASE-02**: A clean build with `GOWORK=off` MUST either resolve the approved parser baseline or fail without silently falling back to the stock COBOL grammar.
- [x] **BASE-03**: Each COBOL extraction generation MUST record a parser grammar content identifier that can be compared across runs.
- [ ] **BASE-04**: Acceptance tests MUST verify named COBOL, DATA DIVISION, IDMS, CICS, SQL, bounded unparsed-tail, `ERROR`, and `MISSING` behavior required from baseline `97ac9f1`.

### Thin Deterministic Tracer

- [x] **TRACE-01**: The first implementation phase MUST trace one named COBOL program-definition node from the enhanced parser through the registered production extractor.
- [x] **TRACE-02**: The tracer MUST emit the program as a native Gortex graph node and its source-file relationship as a native Gortex graph edge.
- [x] **TRACE-03**: The tracer MUST pass through the existing `ExtractionResult`, repository-prefixing, and indexer lifecycle rather than a parallel ingestion path.
- [x] **TRACE-04**: The tracer MUST persist through the existing SQLite `AddBatch` transaction path.
- [x] **TRACE-05**: The traced node MUST be retrievable through at least one existing CLI query.
- [x] **TRACE-06**: The traced node MUST be retrievable through at least one existing MCP query.
- [ ] **TRACE-07**: The tracer acceptance test MUST pass with all AI providers disabled and no Neo4j installation.

### Exact Provenance

- [x] **PROV-01**: Every deterministic source observation MUST retain repository, workspace, and project scope.
- [x] **PROV-02**: Every deterministic source observation MUST retain its normalized repository-relative source path.
- [x] **PROV-03**: Every deterministic source observation MUST retain exact start and end row and column positions from the parser.
- [x] **PROV-04**: Every deterministic source observation MUST retain the source or retrieval revision that supplied its bytes.
- [x] **PROV-05**: Every deterministic source observation MUST retain parser and extractor version identifiers.
- [x] **PROV-06**: Evidence class, extraction origin, and confidence MUST remain separately queryable dimensions.

### Deterministic COBOL and Mainframe Breadth

- [ ] **DET-01**: The extractor MUST emit source-positioned program, division, section, paragraph, statement, and data-item observations from named AST nodes.
- [ ] **DET-02**: The extractor MUST emit paragraph control-flow relationships for `PERFORM`, `THRU`, and `GO TO` constructs supported by the parser baseline.
- [ ] **DET-03**: Literal program calls MUST be represented separately from dynamic call identifiers.
- [ ] **DET-04**: Dynamic call identifiers MUST remain unresolved unless deterministic value-flow or resolution evidence establishes a target.
- [ ] **DET-05**: Copybook and include references MUST retain the referenced member name and source occurrence.
- [ ] **DET-06**: Copybook resolution MUST preserve ambiguity when available library, repository, or search-order evidence does not identify one target.
- [ ] **DET-07**: IDMS statements MUST emit source-positioned record references exposed by the parser baseline.
- [ ] **DET-08**: IDMS statements MUST emit source-positioned set references exposed by the parser baseline.
- [ ] **DET-09**: CICS statements MUST emit source-positioned program and transaction references exposed by the parser baseline.
- [ ] **DET-10**: CICS statements MUST emit source-positioned map and mapset references exposed by the parser baseline.
- [ ] **DET-11**: SQL statements MUST emit source-positioned read and write relationships to deterministically identified physical tables.
- [ ] **DET-12**: SQL aliases, CTEs, includes, and dynamic sources MUST remain distinguishable from physical tables.

### Stable Identity and Incremental Idempotency

- [x] **ID-01**: Source-file and declaration identities MUST extend Gortex's existing repository-prefixed identity convention.
- [x] **ID-02**: Stable declaration identity MUST NOT use source line number as its sole discriminator.
- [ ] **ID-03**: Paragraph identity MUST include its owning program context.
- [ ] **ID-04**: Resource identity MUST include the minimum available namespace needed to avoid merging distinct same-named artifacts.
- [ ] **ID-05**: Multiple source occurrences supporting the same semantic relationship MUST retain distinguishable provenance.
- [ ] **ID-06**: Reindexing identical source, parser, extractor, and material configuration MUST produce the same active deterministic node, edge, and finding identities.
- [ ] **ID-07**: Incremental reconciliation for a changed or deleted file MUST remove stale active observations owned by the prior file generation.
- [ ] **ID-08**: A parser, extractor, retrieval, or material configuration change MUST invalidate affected observations even when source bytes are unchanged.
- [ ] **ID-09**: For identical final inputs, incremental reconciliation MUST produce the same canonical active deterministic subgraph as a clean rebuild.

### Explicit Unresolved Evidence

- [ ] **GAP-01**: A named unparsed tail, localized Tree-sitter `ERROR`, or localized `MISSING` node with surviving context MUST produce a persisted `PARSE_UNRESOLVED` finding.
- [ ] **GAP-02**: A deterministic reference whose target artifact was not retrieved MUST produce an `EXTERNAL_UNRESOLVED` finding.
- [ ] **GAP-03**: `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` MUST remain distinct in storage and query results.
- [ ] **GAP-04**: Each finding MUST retain a stable ID, source scope, exact range, containing symbol, resolution class, normalized reason, severity, parser version, and lifecycle status.
- [ ] **GAP-05**: An external unresolved finding MUST preserve a placeholder for the observed target without fabricating target source, body, or attributes.
- [ ] **GAP-06**: Reprocessing unchanged evidence MUST reuse the active finding identity rather than create a duplicate.
- [ ] **GAP-07**: A finding MUST close or be superseded when its source, parser, or retrieved-artifact evidence resolves or replaces it.
- [ ] **GAP-08**: An uncontained parser recovery cascade that removes expected downstream structure MUST remain a deterministic regression failure even if AI proposes a recovery.

### Native Query Outcomes

- [ ] **QUERY-01**: Native CLI and MCP queries MUST list present COBOL programs and copybooks with stable IDs, scope, and exact source locations.
- [ ] **QUERY-02**: Native queries MUST return paragraph execution paths with unresolved control-flow boundaries disclosed.
- [ ] **QUERY-03**: Native queries MUST return callers of a program while distinguishing resolved literal calls, dynamic identifiers, and missing targets.
- [ ] **QUERY-04**: Native queries MUST return programs that depend on a copybook and disclose unresolved member or library ambiguity.
- [ ] **QUERY-05**: Native queries MUST return data items contained by a program or copybook with exact declaration ranges.
- [ ] **QUERY-06**: Native queries MUST return DB2 table reads and writes with statement-level evidence and nonphysical SQL sources distinguished.
- [ ] **QUERY-07**: Native queries MUST return IDMS record/set and CICS program/transaction/map/mapset usage with source evidence.
- [ ] **QUERY-08**: Native queries MUST filter and aggregate unresolved findings by class, reason, severity, containing symbol, parser version, and lifecycle status.
- [ ] **QUERY-09**: Native queries MUST rank unavailable artifacts by inbound references and affected programs.
- [ ] **QUERY-10**: Query results that cross unresolved boundaries MUST state that returned counts or paths are lower bounds.
- [ ] **QUERY-11**: Native queries MUST distinguish deterministic, unresolved, AI-inferred, human-confirmed, contradicted, and superseded outcomes.

### Storage Authority and Cypher Boundary

- [x] **STORE-01**: SQLite MUST remain the sole authoritative read/write store for active graph facts, findings, claims, and review state in v1.0.
- [x] **STORE-02**: Deterministic indexing MUST perform one authoritative SQLite commit and MUST NOT dual-write to Neo4j.
- [x] **STORE-03**: Neo4j materialization MUST remain a one-way, manually invoked, rebuildable projection derived from SQLite.
- [x] **STORE-04**: The explicit projection MUST NOT turn continuous synchronization, index-time dual-write, or Neo4j replacement into a v1.0 requirement.
- [x] **STORE-05**: Neo4j availability MUST NOT be required outside explicit projection invocation and projection-specific tests, and no reverse-write path from Neo4j may exist.

### Optional AI Context and Provider Policy

- [ ] **AI-01**: AI enrichment MUST be disabled by default.
- [ ] **AI-02**: Disabling AI MUST leave deterministic indexing, persistence, reconciliation, and native queries fully functional.
- [ ] **AI-03**: AI review MUST use the existing Gortex LLM provider abstraction rather than a separate provider framework.
- [ ] **AI-04**: Each AI request MUST be authorized by an explicit policy for provider, model or deployment, workspace/project/repository scope, data class, and retention mode.
- [ ] **AI-05**: Provider connectivity configuration alone MUST NOT authorize proprietary source transmission.
- [ ] **AI-06**: The context builder MUST deterministically bound source and graph context to the selected finding and approved scope.
- [ ] **AI-07**: Each AI context package MUST declare unavailable artifacts and omitted context relevant to the requested review.
- [ ] **AI-08**: Secrets, credentials, protected literals, and disallowed source content MUST be removed before provider request construction.
- [ ] **AI-09**: Each AI invocation MUST record provider, model, policy version, source scope, and context digest without requiring raw source retention.

### Schema-Valid Separate Claims

- [ ] **CLAIM-01**: AI output MUST be validated against a closed, versioned schema before persistence.
- [ ] **CLAIM-02**: A schema-valid AI assertion MUST be persisted as a separate `AI_INFERRED` claim rather than as a deterministic node or edge mutation.
- [ ] **CLAIM-03**: Each material claim MUST retain a stable claim ID, assertion payload, confidence, concise rationale, and missing-context declaration.
- [ ] **CLAIM-04**: Each material claim MUST cite existing in-scope graph IDs or exact source ranges that support it.
- [ ] **CLAIM-05**: Each material claim MUST retain provider, model, prompt-template, context-builder, and output-schema versions.
- [ ] **CLAIM-06**: Claim validation MUST reject unknown evidence IDs and out-of-scope evidence.
- [ ] **CLAIM-07**: Identical retries of the same claim input MUST NOT create duplicate active claims.
- [ ] **CLAIM-08**: Unreviewed `AI_INFERRED` claims MUST be excluded from deterministic and trusted query views by default regardless of confidence.
- [ ] **CLAIM-09**: Running AI enrichment MUST NOT change the canonical active deterministic subgraph.

### Human Review and Claim Lifecycle

- [ ] **REVIEW-01**: An authorized reviewer MUST be able to confirm, contradict, reject, or supersede an AI claim.
- [ ] **REVIEW-02**: Each review action MUST retain reviewer identity, timestamp, decision, and optional rationale.
- [ ] **REVIEW-03**: Human confirmation MUST preserve the originating AI claim and its model provenance.
- [ ] **REVIEW-04**: Contradicted, rejected, stale, and superseded claims MUST remain available for audit but inactive in trusted views.
- [ ] **REVIEW-05**: A claim MUST become stale or superseded when cited evidence, source revision, parser version, context-builder version, prompt version, or model version invalidates its basis.
- [ ] **REVIEW-06**: Multiple conflicting claims MUST coexist without last-write-wins mutation of domain entities.

### Evaluation and Monitoring

- [ ] **EVAL-01**: The milestone MUST include committed representative fixtures for COBOL structure, DATA DIVISION, IDMS, CICS, SQL, parser gaps, and missing artifacts.
- [ ] **EVAL-02**: Deterministic acceptance MUST compare canonical active identities and provenance rather than only aggregate node or error counts.
- [ ] **EVAL-03**: Lifecycle acceptance MUST cover identical rerun, edit, delete, restore, parser-version change, and interrupted retry sequences.
- [ ] **EVAL-04**: AI acceptance MUST use a curated labeled reference set with documented quality thresholds before claims are exposed beyond experimental use.
- [ ] **EVAL-05**: AI evaluation MUST report schema-validity rate, evidence-citation validity, unsupported-claim rate, and review outcome distribution.
- [ ] **EVAL-06**: Monitoring MUST report deterministic extraction failures and unresolved-finding counts by class and reason.
- [ ] **EVAL-07**: Monitoring MUST report AI policy denials, validation failures, provider failures, and claim lifecycle counts without logging raw proprietary source by default.
- [ ] **EVAL-08**: Recurring AI-reviewed gap families MUST be ranked for deterministic parser or extractor improvement using frequency and graph impact.
- [ ] **EVAL-09**: Milestone acceptance MUST verify that AI-on and AI-off runs produce identical active deterministic subgraphs.

### Security and Scope

- [ ] **SEC-01**: Extraction, resolution, query, AI context, claim storage, logging, and export MUST enforce the same workspace/project/repository scope.
- [ ] **SEC-02**: Ambiguous or absent scope MUST fail closed for source reads and AI context assembly.
- [ ] **SEC-03**: Cross-repository resolution MUST NOT bind same-named programs, copybooks, or resources without explicit allowed scope and deterministic resolution evidence.
- [ ] **SEC-04**: Raw prompts, source context, and model responses MUST NOT be written to logs by default.
- [ ] **SEC-05**: Persisted AI audit metadata MUST use restrictive file permissions and bounded retention defined by policy.
- [ ] **SEC-06**: Canary tests MUST verify that protected source does not appear in unauthorized provider requests, logs, claims, temporary files, or Cypher exports.
- [ ] **SEC-07**: AI provider or audit-path health failures MUST be observable without echoing protected source.
- [ ] **SEC-08**: Repository ownership health MUST report zero unowned or misprefixed nodes for the acceptance corpus before AI context or export acceptance.

## Future Requirements

- **FUT-01**: Benchmark named modernization queries and estate-scale graph sizes before reconsidering the storage boundary.
- **FUT-02**: Add continuous or incremental Neo4j synchronization only after a separate accepted ADR defines change capture, tombstones, checkpoints, lag reporting, failure recovery, and operational ownership beyond the v1.0 manual snapshot projection.
- **FUT-03**: Add full JCL parsing, PROC expansion, include resolution, symbolic substitution, and execution relationships in a later milestone.
- **FUT-04**: Add behavioral execution modeling only after the static deterministic graph and identity lifecycle are stable.
- **FUT-05**: Add runtime-fed or live-synchronized digital-twin behavior only after static and behavioral stages prove operational value.
- **FUT-06**: Add broad business-concept, data-lineage, and end-to-end transaction inference only after the narrow claim workflow meets evaluation thresholds.
- **FUT-07**: Add automatic claim promotion only if a later governance decision defines evidence and review controls that preserve deterministic truth.
- **FUT-08**: Add UI-specific review workflows only after CLI/MCP evidence and lifecycle semantics are stable.

## Out of Scope

- Replacing SQLite with Neo4j or any other graph store.
- Building or requiring continuous/live Neo4j synchronization.
- Dual-writing deterministic index mutations to SQLite and Neo4j.
- Reverse-synchronizing Neo4j edits or reviews into Gortex.
- Parsing every COBOL dialect or every mainframe language.
- Changing parser baseline `97ac9f1` unless graph integration proves a release-blocking parser defect.
- Treating AI output, model confidence, or free-form `ask` responses as deterministic graph truth.
- Inventing contents or attributes for unavailable programs, copybooks, JCL members, schemas, or subsystem definitions.
- Full JCL symbolic resolution, behavioral simulation, or a live digital twin.
- Reusing `cobol-ingestor` or `mainframe-viewer` internals.
- Adding a vector database, orchestration framework, parallel graph service, or new LLM provider framework.
- Opening upstream pull requests or resolving the fork's upstream-contribution policy.

## Traceability

| Requirement | Roadmap Phase | Status |
|-------------|---------------|--------|
| NEO-01 | Phase 1 | Complete |
| NEO-02 | Phase 1 | Complete |
| NEO-03 | Phase 1 | Complete |
| NEO-04 | Phase 1 | Complete |
| NEO-05 | Phase 1 | Complete |
| NEO-06 | Phase 1 | Complete |
| NEO-07 | Phase 1 | Complete |
| NEO-08 | Phase 1 | Complete |
| NEO-09 | Phase 1 | Complete |
| NEO-10 | Phase 1 | Complete |
| NEO-11 | Phase 1 | Complete |
| NEO-12 | Phase 1 | Complete |
| NEO-13 | Phase 1 | Complete |
| NEO-14 | Phase 1 | Complete |
| NEO-15 | Phase 1 | Complete |
| NEO-16 | Phase 1 | Complete |
| NEO-17 | Phase 1 | Complete |
| BASE-01 | Phase 2 | Complete |
| BASE-02 | Phase 2 | Complete |
| BASE-03 | Phase 2 | Complete |
| BASE-04 | Phase 6 | Pending |
| TRACE-01 | Phase 2 | Complete |
| TRACE-02 | Phase 2 | Complete |
| TRACE-03 | Phase 2 | Complete |
| TRACE-04 | Phase 2 | Complete |
| TRACE-05 | Phase 2 | Complete |
| TRACE-06 | Phase 2 | Complete |
| TRACE-07 | Phase 2 | Pending |
| PROV-01 | Phase 2 | Complete |
| PROV-02 | Phase 2 | Complete |
| PROV-03 | Phase 2 | Complete |
| PROV-04 | Phase 2 | Complete |
| PROV-05 | Phase 2 | Complete |
| PROV-06 | Phase 2 | Complete |
| DET-01 | Phase 3 | Pending |
| DET-02 | Phase 3 | Pending |
| DET-03 | Phase 3 | Pending |
| DET-04 | Phase 3 | Pending |
| DET-05 | Phase 3 | Pending |
| DET-06 | Phase 3 | Pending |
| DET-07 | Phase 3 | Pending |
| DET-08 | Phase 3 | Pending |
| DET-09 | Phase 3 | Pending |
| DET-10 | Phase 3 | Pending |
| DET-11 | Phase 3 | Pending |
| DET-12 | Phase 3 | Pending |
| ID-01 | Phase 2 | Complete |
| ID-02 | Phase 2 | Complete |
| ID-03 | Phase 4 | Pending |
| ID-04 | Phase 4 | Pending |
| ID-05 | Phase 4 | Pending |
| ID-06 | Phase 4 | Pending |
| ID-07 | Phase 4 | Pending |
| ID-08 | Phase 4 | Pending |
| ID-09 | Phase 4 | Pending |
| GAP-01 | Phase 5 | Pending |
| GAP-02 | Phase 5 | Pending |
| GAP-03 | Phase 5 | Pending |
| GAP-04 | Phase 5 | Pending |
| GAP-05 | Phase 5 | Pending |
| GAP-06 | Phase 5 | Pending |
| GAP-07 | Phase 5 | Pending |
| GAP-08 | Phase 5 | Pending |
| QUERY-01 | Phase 6 | Pending |
| QUERY-02 | Phase 6 | Pending |
| QUERY-03 | Phase 6 | Pending |
| QUERY-04 | Phase 6 | Pending |
| QUERY-05 | Phase 6 | Pending |
| QUERY-06 | Phase 6 | Pending |
| QUERY-07 | Phase 6 | Pending |
| QUERY-08 | Phase 6 | Pending |
| QUERY-09 | Phase 6 | Pending |
| QUERY-10 | Phase 6 | Pending |
| QUERY-11 | Phase 8 | Pending |
| STORE-01 | Phase 1 | Complete |
| STORE-02 | Phase 1 | Complete |
| STORE-03 | Phase 1 | Complete |
| STORE-04 | Phase 1 | Complete |
| STORE-05 | Phase 1 | Complete |
| AI-01 | Phase 7 | Pending |
| AI-02 | Phase 7 | Pending |
| AI-03 | Phase 7 | Pending |
| AI-04 | Phase 7 | Pending |
| AI-05 | Phase 7 | Pending |
| AI-06 | Phase 7 | Pending |
| AI-07 | Phase 7 | Pending |
| AI-08 | Phase 7 | Pending |
| AI-09 | Phase 7 | Pending |
| CLAIM-01 | Phase 8 | Pending |
| CLAIM-02 | Phase 8 | Pending |
| CLAIM-03 | Phase 8 | Pending |
| CLAIM-04 | Phase 8 | Pending |
| CLAIM-05 | Phase 8 | Pending |
| CLAIM-06 | Phase 8 | Pending |
| CLAIM-07 | Phase 8 | Pending |
| CLAIM-08 | Phase 8 | Pending |
| CLAIM-09 | Phase 8 | Pending |
| REVIEW-01 | Phase 8 | Pending |
| REVIEW-02 | Phase 8 | Pending |
| REVIEW-03 | Phase 8 | Pending |
| REVIEW-04 | Phase 8 | Pending |
| REVIEW-05 | Phase 8 | Pending |
| REVIEW-06 | Phase 8 | Pending |
| EVAL-01 | Phase 6 | Pending |
| EVAL-02 | Phase 6 | Pending |
| EVAL-03 | Phase 6 | Pending |
| EVAL-04 | Phase 9 | Pending |
| EVAL-05 | Phase 9 | Pending |
| EVAL-06 | Phase 6 | Pending |
| EVAL-07 | Phase 9 | Pending |
| EVAL-08 | Phase 9 | Pending |
| EVAL-09 | Phase 9 | Pending |
| SEC-01 | Phase 4 | Pending |
| SEC-02 | Phase 4 | Pending |
| SEC-03 | Phase 4 | Pending |
| SEC-04 | Phase 7 | Pending |
| SEC-05 | Phase 7 | Pending |
| SEC-06 | Phase 9 | Pending |
| SEC-07 | Phase 7 | Pending |
| SEC-08 | Phase 4 | Pending |
