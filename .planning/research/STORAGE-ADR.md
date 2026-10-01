# Storage Authority and Projection ADR

**Milestone:** v1.0 Neo4j Projection and Mainframe Graph Foundation
**Status:** Accepted for milestone planning
**Date:** 2026-09-21
**Decision owners:** Milestone planning; implementation remains subject to phase acceptance gates

**Supersedes:** The 2026-09-15 research recommendation to defer all Neo4j implementation until a native-query benchmark failed. That historical recommendation remains useful evidence against continuous synchronization and replacement, but no longer governs the manual projection timing.

## Decision

SQLite is the sole authoritative read/write store for v1.0 deterministic facts, unresolved findings, and any future claim ledger. A user-approved, manually invoked, rebuildable Neo4j projection is included in v1.0 as the immediate priority. Users may select explicit workspace/project/repository scope and materialize that SQLite snapshot through equivalent CLI and MCP operations.

This projection is not continuous synchronization, change data capture, backup authority, index-time dual-write, or a reverse-write path. Those remain deferred or rejected. Neo4j availability is required only for explicit projection invocation and projection-specific integration tests; indexing, daemon startup, native queries, and all other Gortex operations remain independent of Neo4j. Replacing SQLite with Neo4j is rejected for this milestone.

Phase 1 discussion must decide the exact command/tool naming, credential/config syntax, driver or transport choice, and default stale-record action. This ADR deliberately does not lock those gray areas. It does lock the equivalent CLI/MCP behavior, fail-closed explicit scope, non-disclosure, stable keys/constraints, bounded idempotent apply, dry-run/progress/cancellation/results, selected-scope stale reconciliation capability, one-way authority, and Neo4j-optional ordinary operation.

This decision supersedes the 2026-09-15 defer-only conclusion. The authority and anti-dual-write reasoning remains accepted; only the timing of the bounded manual projection changed by explicit user priority.

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
| `internal/exporter/cypher.go::WriteCypher` | Export takes a graph snapshot and emits one Cypher statement per node and edge. It has no remote client, acknowledgement, constraint setup, stale ownership, or retry protocol. |
| `internal/graph/store_sqlite/scoped_projection.go::Store.NodesInScopeSeq` and `EdgesInScopeSeq` | SQLite already exposes keyset-paged, language-neutral, current-generation node/edge reads by requested repository/file frontier and kind. These are stronger Phase 1 foundations than loading an unbounded exporter snapshot. |
| `internal/mcp/tools_export.go::Server.registerExportTools` and `exportByFormat` | MCP already exposes an `export_graph` entry point and delegates Cypher serialization to `WriteCypher`; Phase 1 should preserve CLI/MCP parity but share one projection service rather than duplicate remote-write logic in adapters. |
| SQLite mutation receipts and view generations | Authoritative mutations identify changed identities/files and scoped reads bind to a current view generation. Phase 1 should report the selected snapshot/generation without writing back to this lifecycle. |
| `cmd/gortex/export.go` | Loading exported Cypher is a separate manual operation, not part of indexing or persistence. |

The current exporter uses `CREATE`-oriented serialization and emits no authoritative schema or synchronization state. Replaying it into a non-empty target can duplicate data. Calling this path "Neo4j integration" without qualification would overstate its guarantees.

## Decision Drivers

1. Preserve one source of truth for deterministic evidence.
2. Retain current atomic indexing, generation, eviction, resolution, and restart behavior.
3. Keep AI optional and keep Neo4j absent from indexing and ordinary Gortex availability paths.
4. Preserve workspace/project/repository scope in all reads and exports.
5. Avoid split-brain writes, distributed transaction assumptions, and untestable partial-failure states.
6. Bound the approved Neo4j use to an explicit, user-invoked projection rather than treating it as a second operational authority.
7. Keep the graph reconstructable from retrieved source, parser/extractor versions, and configuration.

## Options Considered

| Option | Decision | Evidence-based assessment |
|---|---|---|
| SQLite authoritative; native queries only | Accepted for v1.0 authority | Matches current indexer, resolver, query, analyzer, and persistence architecture. Lowest consistency risk. |
| SQLite authoritative; manually invoked rebuildable Neo4j projection | Accepted for v1.0 | Provides explicit CLI/MCP materialization with stable keys, constraints, bounded idempotent upserts, stale-scope reconciliation, progress, and retry safety without joining indexing. |
| SQLite authoritative; continuous/incremental Neo4j synchronization | Deferred behind a separate ADR | Requires change capture, checkpoints, lag semantics, operational ownership, and a broader failure model beyond the approved explicit snapshot push. |
| Neo4j replaces SQLite authority | Rejected for v1.0 | Cuts across indexing, generations, eviction, resolution, FTS, analyzers, CLI/MCP behavior, packaging, tests, and offline operation without a demonstrated workload need. |
| Synchronous dual-write from indexing | Rejected | No atomic transaction spans SQLite and Neo4j. Partial success would make graph truth depend on which store a consumer reads. |

## Authority Contract

For v1.0:

- A fact is authoritative only after the native SQLite transaction commits.
- Native Gortex queries define current active-state behavior.
- Deterministic source observations and findings use existing file/generation reconciliation semantics.
- Claim and review history, if implemented, remains SQLite-owned and append-only; it does not become authoritative merely because it is exported.
- Neo4j projected data is derived. It can be deleted and rebuilt from SQLite.
- No Cypher-side edit, review, merge, or correction flows back into SQLite.
- Projection failure cannot roll back or invalidate a committed SQLite generation.
- Projection results must disclose snapshot time, scope, source generation, evidence/status filters, completion status, and failures without exposing credentials.
- Stable node and relationship keys, Neo4j constraints, idempotent upserts, bounded transactions, and explicit stale-record ownership/reconciliation semantics are required for the selected scope.
- The existence of stale-record reconciliation is fixed; whether Phase 1 defaults to report-only, replace-scope, or another safe explicit behavior is left to discuss-phase.
- Partial retries may replay completed work but must never claim a complete snapshot while selected work remains incomplete.

## Continuous Synchronization Reconsideration Gate

A future planning cycle may authorize continuous or incremental synchronization only when all criteria below have evidence:

1. **Named query gap:** At least one approved modernization query cannot meet an agreed latency or expressiveness target through native indexed traversal.
2. **Representative scale:** Benchmarks use a representative estate, relationship mix, unresolved density, and concurrent workload rather than synthetic node counts alone.
3. **Quantified threshold:** Baseline p50/p95 latency, memory, graph size, and refresh cost are recorded. Exact targets are an unresolved product decision, not invented here.
4. **Stable keys:** The v1.0 manual projection's node, relationship, scope, and generation keys are proven and any additional finding/claim/review keys are migration-tested.
5. **Replay protocol:** The synchronization design adds SQLite-committed change capture, tombstones/supersession, per-scope checkpoints, lag reporting, retries, and full rebuild beyond v1.0 snapshot semantics.
6. **Consistency contract:** Read authority, acceptable projection lag, stale-read labeling, failure recovery, and schema migration are explicit.
7. **Security parity:** Projection cannot widen repository/project scope or export evidence classes and source text beyond policy.
8. **Operational ownership:** Deployment, credentials, upgrades, backup, monitoring, and incident response have named owners and accepted cost.

Passing the gate authorizes a separately planned projection. It does not authorize replacing SQLite. Replacement would require a new ADR and migration proof covering every native graph capability.

## Consequences

### Positive

- One authoritative generation and one recovery model remain intact.
- Deterministic indexing works offline with AI and Neo4j disabled.
- Existing native tools immediately benefit from COBOL facts and findings.
- Projection cannot contaminate source truth.
- Neo4j utility is available before COBOL grammar-dependent phases without coupling the projection to any language.

### Negative

- Native SQLite traversal must be benchmarked and may eventually expose workload limits.
- External graph users receive manually refreshed snapshots rather than live changes.
- Gortex must own stable-key, constraint, idempotency, stale-record, and partial-retry behavior for the explicit projection command.
- Claim-ledger retention and indexing still require SQLite schema planning.

### Risks Accepted

- v1.0 may not optimize every exploratory multi-hop visualization workload.
- Snapshot freshness is bounded by export cadence and must be labeled.
- Projection fidelity must carry selected exact ranges, evidence class, lifecycle state, provenance, and scope.
- Neo4j credentials and network availability add an operational boundary for explicit projection only.

## Verification Gates

- Clean rebuild and incremental reconciliation produce the same canonical active deterministic graph.
- Repeated indexing does not create duplicate active facts.
- Native CLI and MCP queries return the tracer without Neo4j installed.
- CLI and MCP projection operations are behaviorally equivalent for the same explicit scope.
- Dry-run reports the intended projection without mutating SQLite or Neo4j.
- Projection is demonstrably one-way and rebuildable from SQLite; no index-time or reverse-write path exists.
- Replaying identical batches is idempotent, and stale-record reconciliation is confined to the selected snapshot owner/scope.
- Disposable-Neo4j or protocol-seam fixtures prove constraints, bounded batching, retry/partial failure, cancellation, repository/workspace/project isolation, and property preservation.
- Failure during projection leaves SQLite authority unchanged and reports an incomplete result rather than a complete snapshot.
- Indexing, daemon startup, native queries, and non-projection tests pass without Neo4j.
- Any proposal for continuous synchronization satisfies every reconsideration criterion in a separate ADR.

## Unresolved Measurements

- Representative estate node/edge/finding/claim volume.
- Named modernization query suite and accepted p50/p95 targets.
- Maximum acceptable manually refreshed snapshot age for downstream analysis.
- Whether reviewed claims are included by default in any projection.
- Projection retention and source-text policy.

These placeholders do not block the v1.0 authority decision or manual projection. They block continuous synchronization or should be settled during Phase 1 discussion where they affect explicit projection behavior.

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
- `internal/graph/store_sqlite/scoped_projection.go::Store.NodesInScopeSeq`
- `internal/graph/store_sqlite/scoped_projection.go::Store.EdgesInScopeSeq`
- `internal/mcp/tools_export.go::Server.registerExportTools`
- `internal/mcp/tools_export.go::exportByFormat`
- `cmd/gortex/export.go`
