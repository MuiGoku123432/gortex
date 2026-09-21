# Architecture Patterns

**Project:** Gortex Mainframe Engine -- v1.0 Neo4j Projection and Mainframe Graph Foundation
**Domain:** Scoped language-agnostic graph projection and later provenance-aware mainframe graph
**Researched:** 2026-09-15; reconciled 2026-09-21
**Overall confidence:** HIGH for current Gortex architecture and Phase 1 seams; MEDIUM for intentionally open projection transport/UX details and later claim-ledger details

## Executive Decision

Extend the existing Gortex pipeline. Phase 1 is a general, manually invoked, explicitly scoped SQLite-to-Neo4j projection exposed through equivalent CLI and MCP operations. Do not create a parallel authority, second identity system, continuous synchronization service, reverse-write path, or index-time dual-write.

SQLite remains the authoritative store for native graph facts and query behavior. The current Cypher exporter remains a one-way serialization boundary and evidence source. Its `CREATE` statements are not retry-safe projection semantics, so Phase 1 must add or wrap a bounded apply path with stable keys, target constraints, idempotent upserts, stale selected-scope reconciliation, dry-run, progress, cancellation, and incomplete-result reporting.

The earlier conclusion that Neo4j should wait for measured native-query inadequacy is **superseded by the user's 2026-09-21 priority decision**. That benchmark gate now applies only to continuous synchronization or replacement, not to the approved manual projection.

Deterministic extraction and AI enrichment must be two separate write paths:

1. The deterministic path parses source, emits source-positioned observations, resolves references where evidence permits, reconciles changed files, and publishes the active graph through existing CLI/MCP/query surfaces.
2. The enrichment path reads a bounded graph context and unresolved findings, invokes an already-configured `llm.Provider`, validates structured output, and writes claims without modifying deterministic nodes or edges.

The first implementation phase must be language-agnostic and independent of parser readiness: project the graph already committed to SQLite. Ground it in `internal/exporter/cypher.go`, `internal/graph/store_sqlite/scoped_projection.go`, current CLI/MCP export entry points, native node/edge identity and scope fields, and SQLite generation/mutation behavior. The thin COBOL tracer remains the first grammar-dependent implementation phase after parser readiness.

## Current Architecture Evidence

| Concern | Current implementation | Architectural consequence | Confidence |
|---|---|---|---|
| COBOL production extraction | `internal/parser/languages/cobol.go::CobolExtractor.Extract` is regex-based and owns `.cbl`, `.cob`, and `.cpy`; `RegisterAll` registers it before forest grammars | The completed Tree-sitter grammar is not yet the production extractor. Integration must replace or evolve this registered extractor, not add a competing extension owner | HIGH |
| Enhanced parser availability | `internal/parser/forest/cobolprobe/README.md` states that the enhanced parser is consumed through the local `go.work` forest shim at parser baseline `97ac9f1`; the probe is measurement-only | Make parser selection reproducible before broad extraction. CI cannot depend silently on an ignored local workspace | HIGH |
| Existing COBOL IDs | Current extractor emits file ID `path`, symbol ID `path::name`, unresolved calls `unresolved::<name>`, and dynamic calls `unresolved::dyncall::<identifier>` | Preserve the established symbol-ID grammar for the tracer; do not encode line numbers into declaration identity | HIGH |
| Multi-repo IDs | `Indexer.applyRepoPrefix` rewrites node IDs and file paths as `<repo_prefix>/<existing-id>`, stamps `RepoPrefix`, `WorkspaceID`, and `ProjectID`, and deliberately leaves `unresolved::` targets unprefixed | Canonical IDs begin with current repository prefixing. Resolution scope belongs in typed fields and resolver policy, not an incompatible COBOL ID namespace | HIGH |
| Node schema | `internal/graph/node.go` defines generic code/resource kinds, source line and column fields, metadata, and workspace/project ownership; it has no COBOL/mainframe domain kinds | Use an existing kind for the tracer. Add domain kinds only after the identity/schema contract is tested end-to-end | HIGH |
| Edge schema | `internal/graph/edge.go::Edge` stores endpoints, kind, source file/line, numeric confidence, confidence label, origin, tier, cross-repo marker, and metadata | Reuse edge provenance. Evidence class must remain separate from confidence; AI status cannot be inferred from a high confidence number | HIGH |
| Provenance defaults | `DefaultOriginFor` classifies structural edges such as `defines`, `contains`, and `imports` as `ast_resolved`; resolution-derived edges fall back by confidence. Current dynamic COBOL calls are explicitly `ast_inferred` | Direct parser observations should be `ast_resolved`; dynamic/value-derived bindings remain inferred even when confidence is high | HIGH |
| SQLite persistence | `store_sqlite.Store.AddBatch` inserts nodes and edges transactionally using bounded multi-row writes. Schema includes generation-scoped nodes/edges, indexes, and FTS sidecars | Keep one atomic graph write path. Do not dual-write SQLite and Neo4j during indexing | HIGH |
| Incremental lifecycle | `reindexIncrementalFilesBatched` evicts deleted files, reparses stale files, retries watcher failures, and feeds derived invalidation. `captureIncrementalState` retains reusable resolutions and unresolved edges | Mainframe extraction must participate in file ownership/eviction rather than implementing its own reconciliation loop | HIGH |
| Placeholder reconciliation | Current extractors emit `unresolved::` endpoints; `reconcilePlaceholderSources` repoints source-owned placeholders after edge reindex, and cross-repo resolution has its own scoped resolver | Represent missing external artifacts using the existing unresolved convention first, then layer explicit findings over it. Do not mint fake resolved targets | HIGH |
| Query behavior | `query.Engine.bfs` reads the graph store, enforces workspace/project scope, excludes unresolved endpoints from ordinary traversal, and reports dropped call targets as epistemic boundaries | Native query surfaces already expose known graph structure and honest unresolved boundaries. The tracer should exercise these surfaces directly | HIGH |
| LLM extension | `provider.New` selects configured providers; `Service.RunAgent` builds graph tools with a scope and creates `agent.New`; MCP `handleAsk` passes repo/project/ref scope. `SetupLLM` omits the feature cleanly when disabled or misconfigured | AI review must reuse provider/service setup and scope, but should have a dedicated structured-review operation rather than smuggling claim writes through free-form `ask` | HIGH |
| Neo4j boundary | `exporter.WriteCypher` snapshots graph nodes/edges to plain `CREATE` and `MATCH ... CREATE`; current MCP `export_graph` exposes format/output/repository/kind/language filters and delegates Cypher to `WriteCypher` | Reuse serialization/property behavior and CLI/MCP surface conventions, but do not replay raw `CREATE` as Phase 1. Add a separate bounded projection service/operation with equivalent CLI/MCP semantics |
| Scoped SQLite projection reads | `store_sqlite.NodesInScopeSeq` and `EdgesInScopeSeq` keyset-page current `viewGen` rows by requested repository/file frontier and kind | Use these existing language-neutral read seams or an equivalent snapshot abstraction; do not load the entire graph unbounded or infer scope from node language |
| SQLite mutation/generation | SQLite mutation receipts identify changed IDs/names/files and scoped reads bind to the store view generation | Capture and report the authoritative generation/snapshot selected for projection; projection must never write mutation receipts or alter SQLite state | HIGH |

## Recommended Architecture

### Phase 1 projection flow

```text
explicit CLI or MCP invocation
          |
          v
fail-closed scope resolution
 workspace + project + repository
          |
          v
authoritative SQLite snapshot/view generation
          |
          v
bounded scoped node/edge readers
          |
          v
projection planner
 stable keys + properties + constraints + stale ownership
          |
          +------ dry-run ------> redacted plan/result
          |
          v
Neo4j apply boundary
 bounded idempotent transactions + cancellation/retry
          |
          v
progress + complete/incomplete result
```

SQLite does not receive a reverse edge in this diagram. Neo4j can be deleted and rebuilt. Exact command name, credential syntax, transport/driver, and default stale-record policy are phase-discussion decisions.

### Deferred grammar-dependent and AI flow

```text
retrieved/preprocessed estate
          |
          v
registered COBOL extractor
  Tree-sitter named nodes + exact ranges
          |
          v
deterministic observation mapper
  Node/Edge + extraction diagnostics
          |
          v
existing index lifecycle
  repo/workspace/project stamps
  file eviction + incremental resolution
          |
          v
authoritative SQLite graph
  active deterministic facts
  unresolved placeholders/findings
          |
          +-------------------+
          |                   |
          v                   v
native query engine      bounded context builder
CLI / MCP / API                |
                              v
                     configured LLM provider
                              |
                     schema validator + policy
                              |
                              v
                      SQLite claim ledger
                              |
                      reviewed claim projection
                              |
          +-------------------+
          v
optional rebuildable Cypher export
```

The key boundary is between **observations** and **claims**:

- An observation is reproducibly derived from a particular source revision, parser version, extractor version, and configuration.
- A resolved deterministic fact is an observation whose target is established by deterministic resolver policy.
- A finding records that evidence exists but deterministic interpretation or retrieval is incomplete.
- A claim is a versioned interpretation proposed by a model or accepted by a human. It never updates an observation in place.

## Component Boundaries

| Component | Responsibility | Owns | Must not own | Communicates with |
|---|---|---|---|---|
| Parser adapter | Run the pinned enhanced COBOL grammar and expose named nodes, fields, errors, missing nodes, and byte/point ranges | Grammar invocation and parser-version fingerprint | Graph IDs, repository scope, AI recovery | COBOL observation mapper |
| COBOL observation mapper | Convert selected named AST nodes into `graph.Node`/`graph.Edge` records with exact ranges and deterministic metadata | Parser-node-to-graph mapping | Persistence transactions, cross-repo policy, model calls | Parser adapter, `parser.ExtractionResult` |
| Indexer integration | Apply repository/workspace/project identity, persist batches, evict stale file generations, schedule resolution/invalidation | Active-file lifecycle and mutation ordering | COBOL semantics | Registry, SQLite store, resolver |
| Deterministic resolver | Bind placeholders using explicit library/repository rules; preserve ambiguity and missing targets | Resolution policy and provenance upgrades | Invented target contents | Native graph, cross-repo resolver, placeholder reconciliation |
| Finding normalizer | Convert named unparsed tails, localized parser errors, and unavailable targets into stable deterministic findings | Gap classification and source evidence | AI interpretation | Parser adapter, resolver, SQLite |
| Native graph store | Authoritative active deterministic nodes/edges and findings; source/query indexes | Current graph generation | Remote graph synchronization | Indexer, query engine, exporter |
| Query surfaces | Expose nodes, usages, paths, gaps, and evidence filters through existing engine/CLI/MCP/API | Scope enforcement and response projection | Separate COBOL data model | SQLite `graph.Store`, MCP/CLI handlers |
| AI context builder | Deterministically gather minimal source and graph evidence for one finding under repo/project policy | Context selection, redaction manifest, unavailable-context list | Provider choice, claim persistence | Query engine, finding store, policy |
| AI review service | Call an approved existing `llm.Provider` with a versioned schema; validate response before persistence | Prompt/schema versions and validation | Direct graph mutation | Existing `svc.Service`/provider, context builder, claim ledger |
| Claim ledger | Retain proposed, confirmed, contradicted, and superseded claims with evidence/model/review lineage | Immutable claim revisions and review events | Deterministic facts | AI review service, optional graph projection |
| Existing Cypher exporter | Preserve current language-neutral property serialization and file/inline interchange | Serialization only | Remote authority, retries, stale state | `graph.Reader`, CLI/MCP export tool |
| Projection scope resolver | Resolve explicit workspace/project/repository selection and fail closed on absence or ambiguity | Selected authority scope and snapshot descriptor | Credential parsing, Neo4j writes | Workspace/project catalog, SQLite store |
| Projection planner | Stream bounded scoped rows, derive stable projected keys/properties/constraints, and identify selected-scope stale ownership actions | Dry-run plan and expected counts | SQLite mutation, default stale policy | Scoped SQLite readers, apply boundary |
| Neo4j apply boundary | Apply constraints and bounded parameterized idempotent batches; stop on cancellation/error and report completion state | Remote transactions, retry classification, progress | Indexing, reverse writes, authority | Projection planner, selected transport/client |
| CLI/MCP projection adapters | Expose equivalent arguments, scope behavior, redaction, progress, cancellation, and result semantics | User/tool protocol mapping | Business logic duplication | Shared projection service |

## Canonical Schema Direction

### 1. Identity is layered, not universal

Use separate canonical-key rules by entity class while preserving the existing repository prefix:

| Entity | Canonical key direction | Stability rule |
|---|---|---|
| Source file | `<repo_prefix>/<normalized-repo-relative-path>` | Stable across content edits; changes only on path/repository identity changes |
| Declaration | `<file-id>::<scope-qualified-logical-name>` | Stable across movement within the same file; do not include line number |
| Occurrence | Hash of `repo + path + revision + parser-node-family + exact byte range + normalized payload` | Range/revision identity is appropriate because an occurrence is source-specific |
| External logical artifact | Typed canonical namespace plus normalized qualified name and resolution domain | Do not use only a short program/copybook name when library search order matters |
| Deterministic edge | Existing edge identity: `from + to + kind + file_path + line`, augmented with exact columns/range metadata where needed | Repeated extraction produces one active row; source movement intentionally changes occurrence identity |
| Finding | Hash of `repo + path + source revision + parser version + range + resolution class + reason` | Same evidence produces same finding; changed source/parser creates a successor |
| AI claim | Hash of `subject/predicate/object proposal + evidence-set digest + model + prompt/context/schema versions` | Rerunning identical inputs is idempotent; changed model or evidence creates a new revision |

Do not put a retrieval revision into declaration IDs. Revision belongs in observation/finding metadata. Otherwise every commit destroys graph continuity and makes incremental reconciliation equivalent to full replacement.

### 2. Domain kinds should be additive and delayed until after the tracer

The current generic mapping (`PROGRAM-ID` and paragraphs both `KindFunction`, sections as `KindMethod`, divisions as `KindType`) is too lossy for a mainframe engine. However, changing the full vocabulary before proving persistence is the wrong order.

Recommended progression:

1. **Tracer:** `PROGRAM-ID` -> existing `KindFunction`, `cobol_kind=program`, exact range, file `EdgeDefines` program. This has zero new schema dependencies and is visible to existing search/read/query tools.
2. **Canonical deterministic schema:** introduce only domain concepts that need independent filtering, identity, or relationships. Strong candidates are `program`, `paragraph`, `data_item`, `copybook`, `finding`, and `claim`. Continue to reuse `KindTable` for DB2 physical tables and generic file/source kinds.
3. **Relationships:** reuse `defines`, `contains/member_of`, `calls`, `imports`, `reads`, `writes`, and dataflow edges when their semantics match. Add `performs`, `uses_record`, `uses_set`, `uses_transaction`, and `uses_map` only when collapsing them into generic edges would prevent required queries.

This avoids both extremes: permanently overloading language-neutral kinds until queries become metadata scans, or adding dozens of mainframe-specific kinds before use cases justify them.

### 3. Source provenance must be typed at the edge/observation boundary

For every deterministic graph row, record:

- repository/workspace/project
- normalized source path
- start/end line and start/end column, using existing typed node columns
- parser grammar fingerprint/version
- extractor mapping version
- source/retrieval revision
- evidence class (`DETERMINISTIC`, `PARSE_UNRESOLVED`, or `EXTERNAL_UNRESOLVED`)
- origin (`ast_resolved`, `ast_inferred`, and existing resolver upgrades)

`Origin`, `Confidence`, and evidence class are orthogonal:

- `origin` answers how the edge was derived.
- `confidence` ranks uncertainty within that derivation.
- `evidence_class` controls whether the row is an active fact, unresolved evidence, or an opt-in claim.

Do not encode evidence class solely in `Meta` forever. The tracer may use metadata, but filtering and validation should gain typed constants/fields before AI claims are queryable.

### 4. Findings and claims require different lifecycle semantics

Deterministic findings belong to the file-owned active graph because they should close when a file is reparsed successfully. Claim and review history must survive file eviction. Therefore:

- Persist active findings as native graph nodes/edges tied to their source file and generation.
- Persist AI claims and review events in an append-only SQLite claim ledger keyed by stable IDs.
- Project only policy-eligible claims into native query views. Unreviewed `AI_INFERRED` claims remain excluded from deterministic views by default.
- On source/parser/context changes, mark prior claims superseded or stale; never rewrite them as deterministic facts.

The exact claim-ledger tables need a phase design after retention and query requirements are fixed. The architectural boundary does not: history cannot share file-eviction semantics with active extracted facts.

## End-to-End Data Flow

### Deterministic indexing

1. The registry selects one COBOL extractor for the file extension.
2. The parser adapter runs the pinned grammar and captures parser fingerprint plus exact named-node ranges.
3. The observation mapper emits a deterministic `ExtractionResult`.
4. The indexer applies `<repo_prefix>/` to local node IDs and file paths, while unresolved targets retain `unresolved::` form.
5. Existing incremental processing captures reusable resolutions, evicts stale file-owned rows, inserts the replacement batch transactionally, and reruns affected resolution/reconciliation.
6. SQLite indexes and FTS make the node immediately visible to native search and traversal.
7. CLI/MCP/API handlers query the same graph store under workspace/project/repository scope.

### Unresolved evidence

1. A parser tail/error produces a source-positioned `PARSE_UNRESOLVED` finding.
2. A deterministic reference with no available target keeps its unresolved edge and produces an `EXTERNAL_UNRESOLVED` finding.
3. Existing call traversal continues to report unresolved call endpoints as epistemic boundaries.
4. Impact/ranking can count findings by dependents without claiming knowledge of missing contents.

### Optional AI enrichment

1. A policy-gated job selects one persisted finding.
2. The context builder queries only the active, scoped native graph and builds a deterministic evidence manifest.
3. The existing provider factory supplies the configured approved provider; no provider means the feature is absent and indexing remains healthy.
4. The model returns a versioned structured claim envelope.
5. A deterministic validator rejects unknown evidence IDs, out-of-scope targets, malformed schemas, and prohibited promotions.
6. The claim ledger stores the proposal and lineage. Human review appends confirmation, contradiction, or supersession.
7. Query surfaces expose claims only through an explicit evidence/status filter; deterministic views never silently include unreviewed claims.

## Storage and Synchronization Decision

### Decision: SQLite authoritative, manual Neo4j projection first

Use the current SQLite graph as the sole authoritative active graph for this milestone. Add the approved explicit Phase 1 projection as a downstream operation.

Reasons:

- The indexer, file eviction, generation model, resolver, FTS, analyzers, MCP server, and query engine already depend on `graph.Store` and its SQLite implementation.
- `AddBatch` already provides the required atomic persistence seam.
- Multi-repo scope is embedded in node fields and query options.
- The current Cypher path serializes a snapshot using `CREATE`; it has no uniqueness constraint setup, `MERGE`, cursor/checkpoint, deletion feed, retry ledger, or transaction acknowledgement from Neo4j.
- A synchronous dual-write would create split-brain failure modes inside the most trust-sensitive deterministic path.

For Phase 1, evolve beyond clear-and-load `CREATE`: establish stable keys and constraints, stream bounded scope, apply idempotently, expose dry-run/progress/cancellation/results, and reconcile stale records only within selected ownership. This remains a manual snapshot projection, not synchronization. The planner must not invent the exact command name, credential syntax, driver choice, or default stale policy before discuss-phase.

### Future decision gate for continuous synchronization

Consider continuous or automatically refreshed Neo4j synchronization only if all are demonstrated:

1. A named modernization query is materially impractical on native Gortex traversal.
2. Dataset size and measured latency justify another operational system.
3. Stable node, edge, finding, and claim keys are finalized.
4. The projection protocol includes idempotent upsert, tombstones, checkpoints, scope preservation, and replay after partial failure.
5. SQLite remains sufficient to reconstruct the projection from scratch.

Replacing SQLite authority is outside this milestone and should remain presumptively rejected.

## Patterns to Follow

### Pattern 1: Walking-skeleton tracer

Start with one real named parser node and no new infrastructure:

```text
program_definition / PROGRAM-ID named node
  -> existing graph.Node{KindFunction, cobol_kind: program, exact range}
  -> EdgeDefines from file
  -> applyRepoPrefix
  -> SQLite AddBatch
  -> search_symbols/read_file/find_usages or graph traversal via CLI and MCP
```

Acceptance must prove:

- the production extractor actually invokes the enhanced Tree-sitter parser
- stable ID is unchanged across identical runs
- source range survives SQLite round-trip
- one changed file replaces, rather than duplicates, the node
- repository/workspace/project scope is correct
- the same node is discoverable through at least one CLI and one MCP/native query surface
- behavior remains valid with LLM disabled and with no Neo4j installation

### Pattern 2: File-owned deterministic reconciliation

Every extracted observation must have a canonical `FilePath` so existing eviction removes stale nodes and edges. Resolver-produced rows must retain enough source ownership to be re-derived from the changed frontier. Avoid global rebuilds for single-file edits.

### Pattern 3: Explicit epistemic boundary

Keep unresolved targets unresolved. A placeholder is evidence that a reference exists, not a synthetic implementation of its target. Findings enrich the unresolved edge with reason, impact, and source range; they do not replace it.

### Pattern 4: Schema-validated claim envelope

AI output should use a closed schema containing claim ID inputs, subject/predicate/object proposal, evidence IDs, missing-context declaration, confidence, rationale, and model/prompt/context versions. Reject before storage if evidence does not resolve within the request scope.

### Pattern 5: One-way projection

External graph stores receive a projection from SQLite. No edits or reviews flow back through Cypher. Human review writes through the native claim API so lineage and scope remain authoritative in one place.

## Anti-Patterns to Avoid

### Parallel COBOL graph package

**Why bad:** It bypasses existing prefixing, resolver, incremental eviction, search, analyzers, and MCP scope.
**Instead:** Emit standard `graph.Node`/`graph.Edge` records and add only necessary domain vocabulary.

### Broad schema-first implementation

**Why bad:** New kinds, claims, Neo4j, and AI can all appear correct while the enhanced parser is still not the registered production path.
**Instead:** Require the one-node walking skeleton as Phase 1.

### Line-number declaration IDs

**Why bad:** Moving code produces delete/recreate churn, breaks lineage, and invalidates claims unnecessarily.
**Instead:** Use logical scoped declaration IDs; ranges are provenance.

### Confidence as truth class

**Why bad:** A 0.99 model output is still inferred, while a certain unresolved reference still lacks target contents.
**Instead:** Separate evidence class, origin, confidence, and review status.

### Dual-write indexing

**Why bad:** Partial SQLite/Neo4j success creates divergent active graphs and retry ambiguity.
**Instead:** Commit SQLite once, then build a replayable downstream projection if later justified.

### Free-form `ask` as the enrichment API

**Why bad:** `ask` is designed to answer questions, not enforce claim identity, validation, evidence linking, or review lineage.
**Instead:** Reuse the provider/service foundation behind a dedicated structured-review boundary.

### Claims owned by source-file eviction

**Why bad:** Reindexing a file would erase audit history and human decisions.
**Instead:** Store append-only claim/review history separately and project current eligible state.

## Build Order

1. **General manual Neo4j projection**
   - Resolve explicit workspace/project/repository scope and snapshot generation from SQLite.
   - Share one language-agnostic projection service between CLI and MCP adapters.
   - Reuse current serializer and scoped-read behavior where correct, but replace raw `CREATE` application with stable keys, constraints, bounded idempotent transactions, and explicit completion state.
   - Add dry-run, redacted progress/results, cancellation, retry/partial-failure safety, and selected-scope stale reconciliation.
   - Keep exact naming, credentials, transport/driver, and stale default open for discuss-phase.

2. **Thin deterministic parser-to-query tracer**
   - Pin and make reproducible the enhanced parser dependency.
   - Route one named program node through the registered COBOL extractor.
   - Reuse `KindFunction`, `EdgeDefines`, current IDs, prefixing, `AddBatch`, and native query surfaces.
   - This is the mandatory first implementation phase.

3. **Canonical identity and provenance contract**
   - Specify normalization, declaration/occurrence/finding IDs, parser and extractor versions, exact ranges, evidence class, and scope.
   - Add round-trip and identical-run tests before expanding extraction.

4. **Deterministic COBOL schema and mapper expansion**
   - Introduce the minimal domain node/edge kinds justified by required queries.
   - Add paragraphs, data items, copybooks, IDMS, CICS, and SQL in vertical slices, each visible through native queries.

5. **Incremental and cross-repo resolution lifecycle**
   - Prove file replacement, deletion, rename behavior, reusable resolution, copybook library scope, duplicate names, and incoming cross-repo references.
   - Do not proceed while identical runs create duplicate active facts.

6. **Explicit gap model**
   - Persist parser and external findings separately, rank their impact, and close/supersede them deterministically.
   - Preserve existing unresolved edges and epistemic-boundary behavior.

7. **AI contract and policy foundation**
   - Design bounded contexts, redaction, allowed-provider policy, closed output schema, evaluations, and deterministic validation.
   - Reuse provider setup and scope; AI remains optional/off by default.

8. **Claim ledger and review projection**
   - Add immutable claims/reviews, evidence links, confirmation/contradiction/supersession, and default-excluded inferred views.

9. **Continuous synchronization evaluation, if later requested**
   - Measure actual freshness/query needs after the manual projection ships.
   - Build change capture/checkpoints/lag semantics only if the future decision gate passes in a separate ADR.

## Phase-Specific Risks

| Phase | Risk | Required mitigation |
|---|---|---|
| Tracer | Tests pass through `go.work` but committed dependency still uses the old grammar | Make parser provenance visible in test output and run acceptance with the exact intended dependency mode |
| Schema | Generic and domain kinds coexist with ambiguous semantics | Publish canonical mapping and compatibility rules; avoid changing existing IDs during kind refinement |
| Extraction | Named parser nodes are mistaken for resolved domain facts | Distinguish syntax observation from resolver output in origin/evidence class |
| Incremental | File eviction deletes too much or derived edges survive stale | Assert active-subgraph equality between clean rebuild and incremental sequence |
| Multi-repo | Short COBOL member names bind across wrong libraries/repos | Resolve under explicit ordered library/workspace/project scope and retain ambiguity |
| Findings | Parser errors and unavailable artifacts collapse into one category | Keep `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` distinct |
| AI | Model output leaks into deterministic query defaults | Separate ledger/write path and require explicit evidence/status filters |
| Neo4j | Snapshot export is mistaken for synchronized authority | Name it projection/export; document rebuild semantics and prohibit reverse writes |

## Verification Strategy

The architectural invariant is:

```text
clean_rebuild(source, parser, config)
  == incremental_reconcile(same source, parser, config)
```

Compare canonical active deterministic subgraphs after sorting away storage order. Include node IDs/kinds/ranges/scope, edge endpoints/kinds/provenance, and findings. Wall-clock timestamps and append-only claim history are outside this equality; active claim projections are tested separately.

For the first tracer, use a tiny committed COBOL fixture and verify the chain at four boundaries:

1. AST named node and exact point range.
2. `ExtractionResult` node/edge and stable local ID.
3. SQLite round-trip after repo prefixing and after repeated/incremental indexing.
4. Native CLI and MCP query responses under correct scope.

## Scalability Considerations

| Concern | Initial estate | Larger multi-repo estate | Projection-scale estate |
|---|---|---|---|
| Writes | Existing bounded `AddBatch` transaction | File-batched reconciliation and derived frontier | Keep SQLite commit independent; project asynchronously |
| Identity | Logical declaration IDs | Repository + workspace/project + library resolution domain | Same canonical IDs in exported properties |
| Query | Native indexed SQLite traversal | Push kind/scope predicates into store capabilities | Measure before adding Neo4j |
| Findings | Per-file active nodes | Impact ranking and deduplicated stable IDs | Export only scoped/current findings unless audit requested |
| Claims | Small append-only ledger | Index by subject, evidence digest, status, model/version | Project reviewed/current state; retain full audit in SQLite |
| Export | Full scoped snapshot | Partition by repo/project and evidence status | Add checkpoints/tombstones only if synchronized projection is approved |

## Open Questions Requiring Phase Research

- The exact enhanced-parser Go API and named-node field names should be frozen from baseline `97ac9f1` during the tracer phase.
- Copybook identity must incorporate actual preprocessing/library search order; a short member name alone is not globally canonical.
- Determine whether domain kinds are all first-class `NodeKind` values or whether some remain generic kinds with typed `cobol_kind` metadata based on required query ergonomics and migration cost.
- Define claim-ledger retention, encryption, and whether raw prompt/context is stored at all.
- Define which human-confirmed claims may enter default graph views and whether confirmation is scoped to source/parser/model versions.
- Measure native SQLite query latency and graph size before any Neo4j synchronization design.

## Sources

### Primary project evidence

- `.planning/PROJECT.md` -- milestone goals, constraints, sequencing, and SQLite authority.
- `docs/ai-enhanced-cobol-graph-handoff.md` -- parser contract, evidence classes, gap/claim requirements, identity starting point, and mandatory thin tracer.
- `internal/graph/node.go` -- current `NodeKind`, node ranges, metadata, and multi-repo ownership fields.
- `internal/graph/edge.go` -- current `EdgeKind`, edge provenance/origin/confidence model, and default provenance rules.
- `internal/parser/languages/cobol.go` -- currently registered regex COBOL extractor, IDs, unresolved targets, and dynamic-call provenance.
- `internal/parser/languages/register.go::RegisterAll` -- extractor registration and extension ownership.
- `internal/parser/forest/cobolprobe/README.md` -- enhanced parser baseline and current non-production integration boundary.
- `internal/indexer/indexer.go::Indexer.applyRepoPrefix` -- canonical repository prefixing and scope stamping.
- `internal/indexer/incremental_batch.go::Indexer.reindexIncrementalFilesBatched` -- current incremental eviction/reparse/retry lifecycle.
- `internal/indexer/incremental_resolve.go::captureIncrementalState` -- reusable resolved references and unresolved-edge carry-forward.
- `internal/resolver/placeholder_sources.go::reconcilePlaceholderSources` -- placeholder repointing after edge reconciliation.
- `internal/graph/store_sqlite/schema.go` and `internal/graph/store_sqlite/store.go::Store.AddBatch` -- authoritative SQLite schema, indexes, generations, and atomic batch persistence.
- `internal/query/engine.go::Engine.bfs` and `internal/query/subgraph.go::QueryOptions.ScopeAllows` -- native traversal, unresolved boundaries, and scope enforcement.
- `internal/indexer/workspace_resolve.go::MultiIndexer.ScopeForCWD` and `internal/mcp/tools_core.go::Server.resolveRepoFilterArgs` -- workspace/project/repository selection.
- `internal/llm/provider/provider.go::New`, `internal/llm/svc/service.go::Service.RunAgent`, and `internal/mcp/tools_llm.go::Server.handleAsk` -- provider, agent, and scoped MCP ask path.
- `internal/exporter/cypher.go::WriteCypher` and `internal/mcp/tools_export.go::Server.registerExportTools` -- current snapshot-only Cypher boundary.

### Confidence note

All decisive architecture claims above were verified directly against the current repository and cross-checked between planning documents and implementation. External documentation lookup was not needed to establish the recommendation; the installed Context7 CLI returned no usable result, so no unsupported external claim is used as authority.
