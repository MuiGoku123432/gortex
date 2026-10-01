# Phase 1: General Neo4j Projection Command - Context

**Gathered:** 2026-09-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver a language-agnostic, explicitly invoked projection that materializes a selected workspace/project/repository snapshot from Gortex's authoritative SQLite graph into Neo4j through equivalent CLI and MCP operations. This phase includes safe replacement of the selected projection namespace, stable projected identities, dry-run planning, progress reporting, and retry-safe failure behavior. It does not add continuous synchronization, index-time dual-write, reverse writes, or Neo4j authority.

</domain>

<decisions>
## Implementation Decisions

### Command and configuration
- **D-01:** The CLI entry point is `gortex neo4j push`.
- **D-02:** The matching MCP tool is `neo4j_push`.
- **D-03:** Neo4j connections use named profiles for non-secret settings and environment-variable references for secrets. Credentials must not be accepted in a connection URL that can leak through shell history or output.
- **D-04:** Every push must explicitly name a profile. There is no implicit default or localhost fallback.

### Snapshot replacement
- **D-05:** A successful push replaces the selected projection scope rather than merely accumulating upserts. Records absent from the new SQLite snapshot become inactive or are removed only within that owned scope.
- **D-06:** Every projected record is owned by an explicit Gortex projection namespace plus workspace/project/repository scope. Gortex must not alter unrelated Neo4j data.
- **D-07:** Multi-batch writes use a pending generation. The prior complete generation remains active until all batches succeed; activation and stale-record cleanup happen only after successful completion.
- **D-08:** Concurrent pushes to the same projection namespace are rejected. Different namespaces may proceed independently if the implementation safely supports it.

### Projected graph shape
- **D-09:** Every projected node has a common `GortexNode` label plus a sanitized label derived from its Gortex node kind.
- **D-10:** Gortex edge kinds become sanitized Neo4j relationship types, and the original edge kind is also retained as a property.
- **D-11:** Project all serializable native node and edge fields and metadata properties by default, excluding secrets and values Neo4j cannot safely represent. Property sanitization must be deterministic and collision-aware.
- **D-12:** Unresolved placeholders are projected as graph nodes and carry a `GortexUnresolved` label so Neo4j paths preserve known epistemic boundaries.

### Run experience
- **D-13:** `--dry-run` connects to Neo4j, validates the selected profile and server capabilities/constraints, inspects the target namespace, and reports planned counts/actions without mutating SQLite or Neo4j.
- **D-14:** CLI output has a concise human progress mode and a structured `--json` mode. MCP uses the same structured result model.
- **D-15:** Recovery from interruption or failure is an ordinary rerun of the same command. Stable generation and idempotent batches make a separate resume command unnecessary.
- **D-16:** MCP `neo4j_push` mutates when `dry_run` is false. It does not require a second confirmation field.
- **D-17:** If a complete new generation cannot be activated, CLI/MCP returns failure with incomplete counts while retaining the prior active snapshot.

### Agent's Discretion
- Exact Neo4j Go driver and supported server-version floor.
- Exact profile field names, environment-variable naming convention, batch defaults, timeout defaults, and progress cadence.
- Deterministic property-name escaping and collision encoding, provided round-trip identity and all selected data are preserved.
- Internal package layout and whether pending generations use labels, properties, or an internal projection manifest node.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Product and requirements
- `.planning/PROJECT.md` - Current milestone priority, SQLite authority, and deferred COBOL work.
- `.planning/REQUIREMENTS.md` - `NEO-01` through `NEO-17` and `STORE-01` through `STORE-05` acceptance contract.
- `.planning/ROADMAP.md` - Phase boundary, dependencies, and observable success criteria.

### Architecture and risk
- `.planning/research/STORAGE-ADR.md` - Accepted manual projection boundary and rejected authority/synchronization alternatives.
- `.planning/research/ARCHITECTURE.md` - Current graph, scope, export, and projection seams.
- `.planning/research/PITFALLS.md` - Identity, partial-failure, scope-leakage, and dual-write risks.
- `.planning/research/STACK.md` - Existing stack and dependency guidance for the projection.

### Existing implementation
- `internal/exporter/cypher.go` - Current one-way Cypher serializer and property conversion behavior.
- `internal/graph/store_sqlite/scoped_projection.go` - Scoped node/edge sequence APIs over authoritative SQLite.
- `internal/mcp/tools_export.go` - Existing MCP export registration, scope handling, and format dispatch.
- `cmd/gortex/export.go` - Existing CLI export behavior and manual Neo4j load instructions.
- `internal/graph/store_sqlite/mutation_receipt.go` - Existing stable mutation identity and generation evidence relevant to projection keys.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `graph.ScopedProjectionSequencer`: streams nodes and edges under repository/file/kind scope without loading the full graph.
- `store_sqlite.Store.NodesInScopeSeq` and `Store.EdgesInScopeSeq`: authoritative SQLite snapshot iterators suitable for bounded projection batches.
- `exporter.WriteCypher`: established node/edge property serialization behavior that can inform, but not directly provide, idempotent remote writes.
- `Server.registerExportTools` and `exportByFormat`: existing MCP scope resolution and export response patterns.
- SQLite mutation receipts and graph identities: starting evidence for stable projected keys and generation manifests.

### Established Patterns
- CLI and MCP should share one service-level operation rather than implement projection behavior twice.
- Workspace/project/repository scope is resolved before graph reads and must be applied identically to dry-run and mutation paths.
- SQLite mutations remain independent; remote projection reads a committed snapshot and cannot participate in indexing transactions.
- Long-running work uses bounded iteration, context cancellation, structured statistics, and explicit incomplete results.

### Integration Points
- Add a `neo4j` Cobra command family alongside existing export/query commands.
- Register `neo4j_push` as an explicitly mutating MCP tool with the same request/result contract as the CLI service.
- Extend configuration with named Neo4j profiles containing non-secret connection metadata and secret environment-variable references.
- Introduce a projection service between scoped graph readers and the Neo4j transport so dry-run, CLI, MCP, and tests share behavior.

</code_context>

<specifics>
## Specific Ideas

- The command should feel like materializing a disposable external view, not migrating Gortex's database.
- A normal push should leave users with one complete active snapshot for the selected projection namespace, never a visibly half-written replacement.
- Neo4j should remain useful for native graph exploration through kind labels and typed relationships while retaining original Gortex properties.

</specifics>

<deferred>
## Deferred Ideas

- Continuous or incremental Neo4j synchronization, change-data capture, projection lag monitoring, and automatic background refresh belong to a future phase after the manual projection proves stable.
- Neo4j as an authoritative store or reverse-write source remains rejected for this milestone.
- COBOL parser integration and the paused tracer discussion resume in Phase 2 when grammar readiness is established.

</deferred>

---

*Phase: 1-General Neo4j Projection Command*
*Context gathered: 2026-09-21*
