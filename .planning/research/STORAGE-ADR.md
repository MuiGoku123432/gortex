# Storage Authority and Projection ADR

**Milestone:** v1.0 Deterministic COBOL Graph Extraction and AI Enrichment Foundation
**Status:** Accepted for milestone planning
**Date:** 2026-09-15
**Decision owners:** Milestone planning; implementation remains subject to phase acceptance gates

## Decision

SQLite is the sole authoritative read/write store for v1.0 deterministic facts, unresolved findings, and any future claim ledger. The current Cypher export is a scoped, rebuildable snapshot only. It is not synchronization, change data capture, backup authority, or a supported reverse-write path.

Neo4j synchronization is deferred. It may be reconsidered only after named modernization queries and representative estate scale demonstrate a material deficiency in native Gortex query latency, traversal capability, or operational usability. Replacing SQLite with Neo4j is rejected for this milestone.

## Context

The milestone's core value is a trustworthy graph whose deterministic facts are reproducible before enrichment. Gortex already indexes into a persistent SQLite graph and serves that graph through its native CLI, MCP, API, resolver, analyzer, and incremental-indexing paths. The planning record explicitly says to evolve this architecture rather than create a parallel engine (`.planning/PROJECT.md`, "Constraints" and "Context"; `docs/ai-enhanced-cobol-graph-handoff.md`, sections 4 and 13).

Current implementation evidence:

| Evidence | What it establishes |
|---|---|
| `internal/graph/store_sqlite/store.go::Store.AddBatch` | Nodes and edges are inserted in one SQLite transaction using bounded multi-row statements while preserving existing upsert/ignore semantics. |
| `internal/indexer/incremental_batch.go::Indexer.commitStructuralIncrementalBatch` | Structural updates evict stale file-owned state and coordinate incremental commit behavior around the native graph. |
| `internal/resolver/cross_repo_incremental.go::CrossRepoResolver.ResolveFilesAndIncoming` | Cross-repository resolution participates in the mutation lifecycle rather than operating as an independent store. |
| `internal/graph/node.go::Node` | Repository, workspace, project, source range, language, and metadata are native graph properties. |
| `internal/indexer/indexer.go::Indexer.applyRepoPrefix` | Repository identity and workspace/project scope are stamped before persistence; unresolved targets deliberately retain their unresolved namespace. |
| `internal/exporter/cypher.go::WriteCypher` | Export takes a graph snapshot and emits one Cypher statement per node and edge. It has no remote client, acknowledgement, checkpoint, tombstone, or retry protocol. |
| `cmd/gortex/export.go` | Loading exported Cypher is a separate manual operation, not part of indexing or persistence. |

The current exporter uses `CREATE`-oriented serialization and emits no authoritative schema or synchronization state. Replaying it into a non-empty target can duplicate data. Calling this path "Neo4j integration" without qualification would overstate its guarantees.

## Decision Drivers

1. Preserve one source of truth for deterministic evidence.
2. Retain current atomic indexing, generation, eviction, resolution, and restart behavior.
3. Keep AI optional and keep Neo4j absent from the indexing availability path.
4. Preserve workspace/project/repository scope in all reads and exports.
5. Avoid split-brain writes, distributed transaction assumptions, and untestable partial-failure states.
6. Require measured workload evidence before adding an operational database.
7. Keep the graph reconstructable from retrieved source, parser/extractor versions, and configuration.

## Options Considered

| Option | Decision | Evidence-based assessment |
|---|---|---|
| SQLite authoritative; native queries only | Accepted for v1.0 authority | Matches current indexer, resolver, query, analyzer, and persistence architecture. Lowest consistency risk. |
| SQLite authoritative; rebuildable Cypher snapshot | Accepted as the current external boundary | Useful for disposable analysis copies when export scope and evidence properties are preserved. No synchronization claim is made. |
| SQLite authoritative; synchronized Neo4j projection | Deferred behind criteria | Potentially useful at proven scale, but requires stable identities, an outbox/change feed, idempotent apply, tombstones, checkpoints, lag visibility, scope enforcement, and rebuild tests that do not exist today. |
| Neo4j replaces SQLite authority | Rejected for v1.0 | Cuts across indexing, generations, eviction, resolution, FTS, analyzers, CLI/MCP behavior, packaging, tests, and offline operation without a demonstrated workload need. |
| Synchronous dual-write from indexing | Rejected | No atomic transaction spans SQLite and Neo4j. Partial success would make graph truth depend on which store a consumer reads. |

## Authority Contract

For v1.0:

- A fact is authoritative only after the native SQLite transaction commits.
- Native Gortex queries define current active-state behavior.
- Deterministic source observations and findings use existing file/generation reconciliation semantics.
- Claim and review history, if implemented, remains SQLite-owned and append-only; it does not become authoritative merely because it is exported.
- Cypher output is derived data. It can be deleted and rebuilt from SQLite.
- No Cypher-side edit, review, merge, or correction flows back into SQLite.
- Export failure cannot roll back or invalidate a committed SQLite generation.
- Export consumers must disclose snapshot time, scope, source generation, and evidence/status filters.

## Neo4j Reconsideration Gate

A future planning cycle may authorize an incremental projection only when all criteria below have evidence:

1. **Named query gap:** At least one approved modernization query cannot meet an agreed latency or expressiveness target through native indexed traversal.
2. **Representative scale:** Benchmarks use a representative estate, relationship mix, unresolved density, and concurrent workload rather than synthetic node counts alone.
3. **Quantified threshold:** Baseline p50/p95 latency, memory, graph size, and refresh cost are recorded. Exact targets are an unresolved product decision, not invented here.
4. **Stable keys:** Node, edge occurrence, finding, claim, review, scope, and generation identities are finalized and migration-tested.
5. **Replay protocol:** The design includes SQLite-committed outbox records, idempotent upsert, tombstones/supersession, per-scope checkpoints, retries, and full rebuild.
6. **Consistency contract:** Read authority, acceptable projection lag, stale-read labeling, failure recovery, and schema migration are explicit.
7. **Security parity:** Projection cannot widen repository/project scope or export evidence classes and source text beyond policy.
8. **Operational ownership:** Deployment, credentials, upgrades, backup, monitoring, and incident response have named owners and accepted cost.

Passing the gate authorizes a separately planned projection. It does not authorize replacing SQLite. Replacement would require a new ADR and migration proof covering every native graph capability.

## Consequences

### Positive

- One authoritative generation and one recovery model remain intact.
- Deterministic indexing works offline with AI and Neo4j disabled.
- Existing native tools immediately benefit from COBOL facts and findings.
- Projection experiments cannot contaminate source truth.
- The milestone spends effort on identity, provenance, gaps, and reconciliation before infrastructure.

### Negative

- Native SQLite traversal must be benchmarked and may eventually expose workload limits.
- External graph users receive snapshots rather than live changes.
- Export consumers must clear/rebuild or otherwise manage duplicate prevention themselves.
- Claim-ledger retention and indexing still require SQLite schema planning.

### Risks Accepted

- v1.0 may not optimize every exploratory multi-hop visualization workload.
- Snapshot freshness is bounded by export cadence and must be labeled.
- Export fidelity may need additive work to carry exact ranges, evidence class, lifecycle state, and scope.

## Verification Gates

- Clean rebuild and incremental reconciliation produce the same canonical active deterministic graph.
- Repeated indexing does not create duplicate active facts.
- Native CLI and MCP queries return the tracer without Neo4j installed.
- Export is demonstrably one-way and rebuildable from SQLite.
- Replaying a snapshot is never documented as idempotent synchronization.
- Export fixtures prove repository/workspace/project scope and evidence/status filters are preserved.
- Failure during export leaves SQLite authority unchanged.
- Any proposal for synchronization includes benchmark results and satisfies every reconsideration criterion.

## Unresolved Measurements

- Representative estate node/edge/finding/claim volume.
- Named modernization query suite and accepted p50/p95 targets.
- Maximum acceptable snapshot age for downstream analysis.
- Whether reviewed claims are included by default in any projection.
- Projection retention and source-text policy.

These placeholders do not block the v1.0 authority decision. They block Neo4j synchronization.

## Sources

- `.planning/PROJECT.md`
- `docs/ai-enhanced-cobol-graph-handoff.md`, especially sections 4, 12, 13, 16, and 18
- `.planning/research/SUMMARY.md`
- `.planning/research/ARCHITECTURE.md`, especially "Storage and Synchronization Decision"
- `.planning/research/PITFALLS.md`, especially pitfalls 2 and 9
- `internal/graph/store_sqlite/store.go::Store.AddBatch`
- `internal/indexer/incremental_batch.go::Indexer.commitStructuralIncrementalBatch`
- `internal/resolver/cross_repo_incremental.go::CrossRepoResolver.ResolveFilesAndIncoming`
- `internal/indexer/indexer.go::Indexer.applyRepoPrefix`
- `internal/exporter/cypher.go::WriteCypher`
- `cmd/gortex/export.go`
