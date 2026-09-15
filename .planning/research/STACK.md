# Technology Stack

**Project:** Gortex Mainframe Engine -- v1.0 Deterministic COBOL Graph Extraction and AI Enrichment Foundation
**Researched:** 2026-09-15
**Overall confidence:** HIGH for the native stack and storage decision; MEDIUM for a future synchronized Neo4j projection because no measured query workload yet requires it

## Recommendation

Keep the existing Go, Tree-sitter, and embedded SQLite architecture. Add domain schema and lifecycle behavior inside the current graph model before adding any infrastructure. The milestone does **not** need Neo4j, a second graph abstraction, a new LLM SDK, a vector database, or a workflow/orchestration service.

Use the authoritative COBOL parser baseline from `tree-sitter-cobol-upgrade/main` merge `97ac9f1`. The current ignored local `go.work` is suitable for development, but it is not a reproducible release dependency. Before milestone acceptance, publish or otherwise pin the `forest-shim/cobol` module at that exact source revision through `go.mod` and `go.sum`.

SQLite remains authoritative. The existing Cypher exporter is a useful manual snapshot/export seam, but its current `CREATE`-only output is not an incremental synchronization protocol. Defer the Neo4j Go driver unless measured modernization queries prove that a live projection is worth its operational and consistency cost.

## Existing Stack to Retain

### Core Runtime and Parsing

| Technology | Current version/baseline | Purpose | Decision and rationale |
|---|---:|---|---|
| Go | `go 1.27.0` in `go.mod` | Engine, daemon, CLI, MCP, HTTP, extraction and analysis | Retain. The milestone is an extension of a mature Go engine; another runtime would duplicate lifecycle, scoping, and query infrastructure. |
| `github.com/tree-sitter/go-tree-sitter` | `v0.25.0` | Tree-sitter runtime and source-positioned AST access | Retain. Exact named-node ranges are the deterministic evidence boundary. |
| COBOL forest shim | `tree-sitter-cobol-upgrade/main` merge `97ac9f1` | Authoritative COBOL, DATA DIVISION, IDMS, CICS, SQL, unparsed-tail, ERROR, and MISSING node contract | Pin this exact baseline. Parser work is out of scope unless graph integration exposes a proven regression. Do not fall back to the older generic `github.com/alexaandru/go-sitter-forest/cobol v1.9.1` grammar behavior. |
| Existing language extractor/indexer interfaces | current repository code | Convert parser output into `graph.Node` and `graph.Edge`, resolve references, and patch changed files | Extend rather than create a parallel COBOL ingestion service. This preserves native CLI/MCP/API behavior and incremental re-indexing. |

### Authoritative Graph and Persistence

| Technology | Current version | Purpose | Decision and rationale |
|---|---:|---|---|
| `modernc.org/sqlite` | `v1.58.0` | Embedded durable graph store | Keep as the sole authority for v1.0. Current architecture writes every mutation to SQLite, queries it in place, restores it on startup, and performs scoped file-level reconciliation. Replacing it would cut across the entire engine. |
| Existing `graph.Node` / `graph.Edge` schema | current repository code | Typed graph data model | Extend minimally. Nodes already carry stable ID, kind, exact lines and columns, language, metadata, repository, workspace, and project. Edges already carry source file/line, origin, confidence, confidence label, metadata, and cross-repo status. Add only concepts that cannot be represented honestly with current kinds and metadata. |
| Existing SQLite schema/migrations and mutation receipts | current repository code | Atomic mutation, durable identity changes, resolution frontier, incremental repair | Reuse. Mainframe facts, gaps, claims, and reviews must participate in the same mutation and eviction lifecycle rather than using sidecar JSON or a second database. |
| Existing XDG data layout | current repository code | Durable local state | Retain. The fork already runs against isolated `~/.gortex-fork`; no new database service is required for deterministic indexing. |

### Query and Integration Surfaces

| Technology | Current version | Purpose | Decision and rationale |
|---|---:|---|---|
| Existing query engine | current repository code | Traversals, search, usages, dataflow, analyzers | Make COBOL facts visible through this layer first. A graph fact that only exists in a special exporter is not integrated. |
| MCP, Cobra CLI, HTTP API | current repository code | Native access surfaces | Reuse. The first tracer should prove one parser node reaches SQLite and all relevant native surfaces. |
| Existing Cypher exporter | current repository code | Manual Neo4j/Memgraph-compatible snapshot | Retain and harden only as needed for schema round-trip tests. It currently exports typed labels, stable node IDs, scope fields, edge origin/confidence, source location, and flattened metadata. |

### AI Foundation

| Technology | Current version | Purpose | Decision and rationale |
|---|---:|---|---|
| Existing `internal/llm` provider interface and factory | current repository code | Optional model invocation across local, enterprise, hosted, and CLI providers | Reuse. Do not introduce LangChain, an agent framework, or provider-specific SDKs. Provider construction already fails soft so deterministic features remain available. |
| Existing OpenAI-compatible provider | current repository code | Strict JSON Schema, JSON object, or prompt-only structured output | Reuse as the common structured-output seam for OpenAI, Azure, custom gateways, and compatible local/enterprise deployments. |
| Existing Anthropic, Bedrock, Gemini, Ollama, local llama.cpp, and CLI adapters | current repository code | Deployment-policy choices | Reuse, but add a claim-specific capability gate. Raw proprietary source must be allowed only for explicitly approved providers; local or enterprise-controlled endpoints should be the default policy. |
| `github.com/santhosh-tekuri/jsonschema/v6` | `v6.0.3` indirect today | Deterministic JSON Schema validation | Promote to direct use if claim validation imports it. Validate every AI response before persistence; provider-native structured output is not sufficient by itself. |

## Actual Stack Additions Needed

| Addition | Needed in v1.0? | Why | Recommended form |
|---|---|---|---|
| Reproducibly pinned COBOL grammar module | **Yes** | The current local `go.work` boundary does not encode the authoritative parser revision in the repository's module graph. CI and other machines must build the same grammar. | Publish/tag the `forest-shim/cobol` module or require a commit-derived pseudo-version tied to `97ac9f1`; retain the workspace override only for parser development. |
| Mainframe graph kinds and metadata constants | **Yes** | Existing generic kinds cover files, tables, variables, references, reads/writes, calls, and resources, but explicit parser findings, missing artifacts, reviewable claims, and some mainframe resources need honest query semantics. | Add a small set of first-class kinds for `finding`, `missing_artifact`, and `claim` if schema reconciliation confirms no existing kind fits. Prefer metadata or flavors for program/paragraph/data-item/CICS/IDMS distinctions when generic structural semantics remain correct. |
| Stable content-derived identity helpers | **Yes** | Findings, source occurrences, claims, and reviews need idempotent reruns and supersession. | Extend existing `<repo_prefix>/<path>::<Symbol>` identity conventions. Include canonical source-relative identity plus semantic occurrence key; keep source revision, parser version, extractor version, prompt version, and model version as provenance, not arbitrary identity churn. |
| SQLite migrations/indexes for lifecycle queries | **Yes, likely** | Active/superseded/contradicted status and exact evidence links need efficient filtering without overloading unindexed JSON metadata. | Add normalized columns/tables only for fields used in identity, lifecycle, joins, or common filters. Keep descriptive details in metadata. Use the existing migration framework and transaction boundary. |
| Deterministic claim-output schema and validator | **Yes** | AI output must not enter the graph unchecked. | Versioned Go structs plus JSON Schema validation, mapped to first-class claim records and evidence edges. |
| Provider data-governance policy | **Yes** | Existing providers answer connectivity, not whether proprietary source is allowed to leave the machine or be logged. | Add explicit allow/deny policy by provider and model, redaction/context limits, retention settings, and auditable invocation metadata. Keep AI off by default. |
| Neo4j Go driver | **No, defer** | No current requirement proves a synchronized projection is necessary. Adding it creates a remote service, credentials, retries, lag, deletion, and scope-consistency obligations. | Add `github.com/neo4j/neo4j-go-driver/v6` only after a decision gate and measured query benchmark selects synchronized projection. Current official docs list v6 as current. |
| Queue/event bus for projection | **No** | SQLite has one writer and existing durable mutation receipts. A new broker would be unjustified complexity for a local personal engine. | If projection is later approved, first use a durable SQLite outbox/change cursor consumed by a single worker. Do not add Kafka/NATS merely for replication. |
| Vector database or embedding service | **No** | The milestone's claims are bounded by deterministic graph and source evidence; Gortex already has search/reranking infrastructure. | Reuse current retrieval and graph traversal. Revisit only with measured recall failures. |

## Storage Options Compared

| Option | Authority | Incremental behavior | Provenance/identity fit | Multi-repo fit | Operational cost | v1.0 verdict |
|---|---|---|---|---|---|---|
| **Native SQLite only** | SQLite | Native per-file eviction, scoped reparse/re-resolution, warm restart reconciliation, and mutation receipts | Best. Existing node/edge fields and metadata already persist source and resolution evidence | Best. Workspace/project/repo boundaries and checkout generations are native | Lowest; embedded and offline | **Recommended baseline and default shipping mode** |
| **Existing Cypher export** | SQLite | Snapshot only; rerunning current output duplicates data unless target is reset | Carries IDs, scope, locations, origin, confidence, labels, and metadata, but relationships have no explicit stable exported edge ID | Can export scoped snapshots, but the target does not enforce Gortex query boundaries by itself | Low and optional | **Keep as manual analysis/interchange path**; do not call it synchronization |
| **Synchronized Neo4j projection** | SQLite | Feasible only with durable change/outbox records, stable IDs, idempotent upserts, tombstones, retries, lag reporting, and rebuild tooling | Good if projection preserves all evidence fields and never invents authority | Requires workspace/project/repo fields in every merge key or mandatory predicate and safe handling of checkout generations | Medium/high; adds driver, server, credentials, migrations, monitoring, and consistency tests | **Conditional future addition**, gated by measured query value |
| **Replace SQLite with Neo4j** | Neo4j | Requires reimplementing indexing transactions, startup recovery, scoped reconcile, watcher behavior, checkout generations, and native query tools | High migration risk and no demonstrated benefit for identity/provenance | Highest risk because current isolation semantics are embedded throughout native storage/query paths | Highest; loses embedded/offline simplicity | **Reject for v1.0** |

### Why the Existing Cypher Export Is Not a Live Projection

`internal/exporter/cypher.go::WriteCypher` snapshots all selected nodes and edges. `writeCypherNode` emits `CREATE`, while `writeCypherEdge` performs endpoint `MATCH` followed by relationship `CREATE`. This is intentionally portable and simple, but it has no target-side uniqueness DDL, no `MERGE`, no deletions/tombstones, no checkpoint, no retry cursor, and no atomic correspondence with a SQLite mutation. It is appropriate for reset-and-load exploration, not continuous synchronization.

If a synchronized projection is later justified:

1. Keep SQLite authoritative.
2. Give every projected node an application-assigned stable `gortex_id` with a uniqueness constraint.
3. Give every projected relationship a stable identity or a uniqueness-safe tuple that includes edge kind, endpoints, occurrence/provenance identity, and scope. Parallel evidence-bearing edges must not collapse accidentally.
4. Use parameterized, static-label/type Cypher and managed write transactions.
5. Make each transaction idempotent because the official Neo4j Go driver retries transient failures.
6. Persist outbox sequence/checkpoint state in SQLite in the same authoritative transaction as graph mutation where possible.
7. Project tombstones and supersession, not just creates/updates.
8. Expose projection lag and rebuild from SQLite; never reverse-sync Neo4j into native truth.

Neo4j's official Cypher documentation explicitly says `MERGE` alone does not guarantee uniqueness under concurrent loads and recommends constraints on identifying properties. This makes stable IDs and constraints prerequisites, not optional tuning.

## Schema Reconciliation Guidance

### Reuse Existing Typed Fields

| Concern | Existing capability | Recommendation |
|---|---|---|
| Source location | Node start/end line and zero-based start/end column; edge file and line | Use typed fields for primary occurrence location. Add precise end ranges or evidence-range records only where edge-level line precision is insufficient. |
| Repository identity | `RepoPrefix` and prefixed node IDs | Preserve as the starting identity domain. Do not create global program IDs that collapse same-named programs across libraries or repos. |
| Query isolation | `WorkspaceID` hard boundary and `ProjectID` soft sub-boundary | Stamp every mainframe node, finding, claim, and projection row. Placeholder targets must remain scope-aware. |
| Provenance | Edge `Origin`, `Confidence`, `ConfidenceLabel`, `Meta`; node `Meta` | Reuse for extraction mechanics, but do not encode evidence class solely as confidence. `AI_INFERRED` remains non-factual at confidence 1.0. |
| Deterministic relationships | Existing calls, contains, defines, references, reads/writes, queries, table/column and dataflow edges | Reuse where semantics match. Add domain edge kinds only where generic kinds would make modernization queries ambiguous or false. |
| External placeholders | Existing unresolved/external conventions and opt-in synthesized external calls | Extend with explicit missing-artifact type and reason. A known reference to an unavailable target is deterministic even though target content is unknown. |
| Incremental replacement | File eviction, batch mutation, re-resolution, warm reconciliation, mutation receipts | Ensure new facts and findings are owned by a source file/extraction generation so scoped reindex removes stale active state. Audit history needs append-only lineage separate from the active graph view. |

### Evidence and Claim Model

Use first-class claim nodes for material AI assertions. Connect them to deterministic nodes and exact source evidence through typed edges such as `supported_by`, `asserts`, `contradicts`, and `supersedes`. Human confirmation changes claim review state or emits a review record; it must not rewrite the original model provenance.

Do not overload existing edge `Origin` with the full evidence-class lifecycle. Existing origins describe extraction/resolution methods such as AST inferred/resolved or LSP resolved. Add an explicit evidence class/status field with a closed enum for deterministic facts, unresolved findings, AI inferences, human confirmations, contradictions, and supersession.

## Identity and Incremental Lifecycle

- **Domain identity:** derive from workspace/repository/library context plus canonical artifact name. Same-named COBOL members in different concatenation libraries must remain distinct until deterministic resolution proves the binding.
- **Occurrence identity:** derive from repository, path, containing symbol, parser node family, and stable semantic occurrence discriminator. Avoid raw line number as the sole key because unrelated edits shift lines.
- **Finding identity:** derive from occurrence identity, resolution class, normalized reason, and parser/extractor contract version. An unchanged rerun reuses the ID; changed source/parser creates a successor and closes/supersedes the prior active finding.
- **Edge identity:** do not assume `(from,to,kind)` is enough. Multiple statements can establish the same relationship with different source ranges and evidence. Preserve occurrence/provenance identity where auditability matters.
- **Claim identity:** derive from finding/evidence package plus prompt-template/context-builder/model versions and normalized assertion. Retries of one invocation must deduplicate; genuinely different model runs remain separately auditable.
- **Revision handling:** source commit/retrieval revision and parser version are provenance dimensions. They govern validity and supersession but should not force churn in stable domain entity IDs.
- **Incremental ownership:** every emitted fact, finding, and claim dependency must be traceable to source file/retrieval generation so the existing eviction and scoped reconcile machinery can identify stale active records.

## Provider and AI Stack Decision

The provider layer is broad enough already. The missing stack is policy and domain contracts, not another model client.

Recommended order:

1. Deterministic context builder over native graph reads.
2. Redaction and unavailable-context manifest.
3. Provider-policy gate that defaults to deny for raw proprietary source and defaults AI off globally.
4. Versioned structured claim schema.
5. Provider call through existing `llm.Provider`.
6. Local deterministic JSON Schema validation and evidence-ID validation.
7. Persist claim separately from deterministic facts.
8. Human review and lifecycle updates.

Prefer `local`, `ollama`, Azure OpenAI, Bedrock with approved enterprise controls, or a registered internal OpenAI-compatible gateway according to actual data policy. Do not claim any hosted provider is approved merely because Gortex supports it. CLI subscription adapters are poor defaults for auditable production enrichment because their surrounding service policy, retention, model identity, and reproducibility are less explicit than direct configured APIs.

OWASP guidance supports sanitization, strict input validation, least-privilege data access, restricted data sources, tokenization/redaction, and transparent retention policy. Apply those controls before provider invocation, not only in the prompt.

## Build and Dependency Reproducibility

The parser is currently consumed through a local `go.work` integration boundary. Official Go documentation confirms that workspace replacements can override module resolution and that module versions in `go.mod` are the durable, canonical build inputs. Therefore:

- Treat `97ac9f1` as the semantic parser baseline.
- Materialize it as a tagged or pseudo-versioned module dependency before release/CI acceptance.
- Keep `go.work` for local co-development only.
- Record parser grammar content/version in extraction provenance so a parser upgrade can invalidate or supersede findings deterministically.
- Do not vendor the entire parser repository into Gortex unless publishing the shim is impossible; that increases fork maintenance and obscures provenance.

## Alternatives Considered

| Category | Recommended | Alternative | Why not now |
|---|---|---|---|
| Graph authority | Embedded SQLite | Neo4j authority | Would require replacing proven persistence, query, restart, watcher, checkout, and scope behavior without a measured requirement. |
| Neo4j integration | Manual export now; synchronized projection only after gate | Immediate dual-write | Dual-write creates split-brain and partial-failure hazards. A projection needs a durable outbox and rebuild semantics first. |
| Projection transport | SQLite outbox/cursor if needed | Kafka/NATS | The product is a local engine with one authoritative writer; a broker is unjustified operational weight. |
| Mainframe schema | Extend native typed graph conservatively | Separate COBOL graph service/schema | A parallel graph would bypass existing resolution, analyzers, scoping, and access surfaces. |
| AI integration | Existing provider interface plus policy/validation | LangChain or provider SDK per model | Existing adapters already cover required providers and structured output. New frameworks add dependency and behavior surface without solving trust. |
| Claim persistence | First-class claim/evidence nodes in SQLite | Mutating deterministic edges or dumping JSON sidecars | Mutation destroys epistemic separation; sidecars bypass native queries and incremental lifecycle. |
| Parser dependency | Published/pseudo-version pinned to `97ac9f1` | Developer-local `go.work` only | Local workspaces are not reproducible across CI or other machines. |
| Retrieval | Native graph traversal and search | New vector DB | No measured retrieval gap yet; adds another consistency domain. |

## Installation and Dependency Impact

### Required for the initial deterministic phases

```bash
# No new runtime service and likely no new third-party package.
# Replace the generic COBOL module requirement with a reproducible module
# version that resolves to the forest-shim generated from parser commit 97ac9f1.
go mod tidy

go build -o gortex ./cmd/gortex/
go test -race ./...
```

### Required when structured AI claims are implemented

```bash
# Prefer the JSON Schema package already present in the module graph.
# It becomes a direct dependency only when imported by claim validation.
go get github.com/santhosh-tekuri/jsonschema/v6@v6.0.3
```

### Conditional only if synchronized Neo4j projection passes the decision gate

```bash
go get github.com/neo4j/neo4j-go-driver/v6
```

Do not add the Neo4j driver merely to improve the existing file exporter. Export remains database-driver-free by design.

## Decision Gates

### Gate 1 -- New Node or Edge Kind

Add a kind only if all are true:

- Existing generic semantics would produce misleading queries.
- The concept is queried independently, not merely displayed.
- It has a stable identity and lifecycle.
- It can be stamped with workspace/project/repository scope.
- Incremental eviction and re-resolution behavior are defined.

### Gate 2 -- Synchronized Neo4j Projection

Add Neo4j only if a benchmark corpus demonstrates at least one important modernization query that is materially impractical through native queries and if the project accepts:

- a separately operated Neo4j 5.26 LTS or supported current deployment,
- driver credentials and TLS policy,
- uniqueness constraints and projection migrations,
- retry-safe idempotent writes,
- tombstones and rebuild behavior,
- projection lag monitoring,
- workspace/project access controls,
- and explicit non-authoritative status.

Without that evidence, ship SQLite plus export.

### Gate 3 -- Hosted AI Provider

Allow raw source only after explicit approval records provider, model/deployment, endpoint, retention/training policy, region, redaction rules, and log behavior. Connectivity configuration alone is not authorization.

## Roadmap Implications

1. **Schema and identity reconciliation first:** decide native kinds, metadata, evidence classes, stable IDs, source ownership, and lifecycle before broad extraction.
2. **Thin deterministic tracer second:** prove one parser node from baseline `97ac9f1` reaches native SQLite and CLI/MCP queries with exact source provenance.
3. **Broaden deterministic extraction and gaps:** reuse existing kinds where honest; add explicit parser and missing-artifact findings.
4. **Prove idempotency and incremental supersession:** identical reruns, changed files, parser version changes, deleted files, and cross-repo resolution must converge.
5. **Evaluate storage boundary:** benchmark native queries and the existing export. Implement synchronized Neo4j only if the gate passes; otherwise this phase should close with an ADR and export validation, not code.
6. **Add AI policy and validated claims:** reuse provider infrastructure after deterministic context and evidence contracts exist.

## Current Code Evidence

- `internal/graph/node.go`: current node kinds, stable ID conventions, exact source columns, `RepoPrefix`, `WorkspaceID`, `ProjectID`, metadata, and proxy-origin semantics.
- `internal/graph/edge.go`: current structural, database, dataflow, cross-repo, origin, confidence, and incremental-edge conventions.
- `internal/graph/store_sqlite/mutation_receipt.go::recordSQLiteChangedNodeIdentity`: SQLite mutations already track changed identities, names, files, and incomplete resolution frontiers.
- `internal/graph/store_sqlite/reindex_receipt.go::Store.publishSQLiteReindexReceiptLocked`: reindex deltas feed the native mutation receipt path.
- `internal/indexer/indexer.go::Indexer.IncrementalReindexPaths`: current scoped incremental re-indexing entry point.
- `internal/exporter/cypher.go::WriteCypher`, `writeCypherNode`, `writeCypherEdge`: current Cypher path is a full `CREATE` snapshot, not synchronization.
- `internal/llm/provider/provider.go::New` and `newCustom`: existing provider factory and OpenAI-compatible custom-provider seam.
- `internal/llm/provider/openaicompat/openaicompat.go::SchemaMode`: current strict JSON Schema, JSON-object, and prompt-only modes.
- `internal/semantic/manager.go::Manager.RepoEnrichmentMarkerState`: persisted per-repository enrichment state demonstrates a reusable pattern for versioned optional enrichment, though AI claims need a separate lifecycle.
- `go.mod`: Go 1.27.0, Tree-sitter runtime v0.25.0, generic COBOL forest module v1.9.1, SQLite v1.58.0, and existing JSON Schema validator v6.0.3; no Neo4j driver today.

## Sources and Confidence

| Source | What it supports | Confidence |
|---|---|---|
| `.planning/PROJECT.md` and `docs/ai-enhanced-cobol-graph-handoff.md` | Milestone constraints, parser authority, deterministic-first rule, evidence classes, Neo4j decision gate | HIGH -- project authority |
| `docs/architecture.md`, `docs/llm.md`, `docs/multi-repo.md` | Current persistence, provider, incremental indexing, identity, workspace/project and checkout behavior | HIGH -- current repository documentation cross-checked against code |
| Current code files listed above | Actual types, dependencies, exporter behavior, mutation receipts, and provider seams | HIGH -- primary implementation evidence |
| [Go Modules Reference](https://go.dev/ref/mod#workspaces) | Workspace overrides versus durable module version selection and pseudo-versions | LOW per research confidence classifier, but official primary documentation and used only to support the pinning recommendation |
| [Neo4j Cypher Manual -- MERGE](https://neo4j.com/docs/cypher-manual/current/clauses/merge/) | `MERGE` behavior and need for constraints to prevent concurrent duplicates | LOW per research confidence classifier, but official primary documentation |
| [Neo4j Go Driver Manual -- transactions](https://neo4j.com/docs/go-manual/current/transactions/) | Current driver major version, ACID transactions, managed retries, idempotent callback requirement | LOW per research confidence classifier, but official primary documentation |
| [OWASP LLM02:2025 Sensitive Information Disclosure](https://genai.owasp.org/llmrisk/llm022025-sensitive-information-disclosure/) | Sanitization, input validation, least privilege, restricted sources, redaction, retention transparency | LOW per research confidence classifier; corroborates project policy rather than determining architecture |

## Open Questions Requiring Phase-Specific Evidence

- Which candidate mainframe entities truly need new `NodeKind`/`EdgeKind` values after query examples are written?
- Does the current edge deduplication key preserve multiple source occurrences of the same semantic relation, or is a first-class evidence/occurrence node required?
- Which native modernization queries exceed acceptable latency on the DCC and larger estate corpus?
- What are the actual enterprise-approved model endpoints, retention policies, and source classifications?
- How should retrieval/library concatenation order disambiguate same-named copybooks and programs across repositories?
- What exact parser grammar content identifier should be persisted when the generated forest shim and parser source live in separate repositories?
