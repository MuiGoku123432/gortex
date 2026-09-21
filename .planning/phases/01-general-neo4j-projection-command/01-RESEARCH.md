# Phase 1: General Neo4j Projection Command - Research

**Researched:** 2026-09-21
**Domain:** Explicit, scoped, one-way SQLite-to-Neo4j snapshot projection in Go
**Confidence:** HIGH for repository architecture and locked behavior; MEDIUM for the new Neo4j transport and generation protocol

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

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

### the agent's Discretion
- Exact Neo4j Go driver and supported server-version floor.
- Exact profile field names, environment-variable naming convention, batch defaults, timeout defaults, and progress cadence.
- Deterministic property-name escaping and collision encoding, provided round-trip identity and all selected data are preserved.
- Internal package layout and whether pending generations use labels, properties, or an internal projection manifest node.

### Deferred Ideas (OUT OF SCOPE)
- Continuous or incremental Neo4j synchronization, change-data capture, projection lag monitoring, and automatic background refresh belong to a future phase after the manual projection proves stable.
- Neo4j as an authoritative store or reverse-write source remains rejected for this milestone.
- COBOL parser integration and the paused tracer discussion resume in Phase 2 when grammar readiness is established.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NEO-01 | Equivalent explicit CLI and MCP projection | One shared projection service with thin Cobra and MCP adapters |
| NEO-02 | Explicit workspace/project/repository scope; fail closed | Reuse strict scope resolution, then require a non-empty concrete repository allow-set |
| NEO-03 | Explicit connection config and secret redaction | Machine-level named profiles with environment references only for credentials |
| NEO-04 | Stable node projection keys | Namespace/scope plus authoritative `Node.ID`, separated from generation-specific physical identity |
| NEO-05 | Stable relationship projection keys | Hash the authoritative edge identity fields and origin, preserving material occurrence distinctions |
| NEO-06 | Verify identity constraints before writes | Named node and per-relationship-type uniqueness constraints plus capability inspection |
| NEO-07 | Idempotent upsert | Parameterized `UNWIND` plus `MERGE` under uniqueness constraints |
| NEO-08 | Bounded batches | Keyset-paged SQLite snapshot and bounded Neo4j managed transactions |
| NEO-09 | Non-mutating dry-run | Connectivity/schema/scope/target census only; no DDL or DML |
| NEO-10 | Progress and final result | Shared event/result contracts, phase counts, redacted errors |
| NEO-11 | Prompt cancellation/error stop | Context-aware reader and driver calls; explicit incomplete result |
| NEO-12 | Safe partial failure/retry | Pending generation and idempotent replay; no activation before all batches pass |
| NEO-13 | Preserve scope and provenance properties | Dedicated mapper covers every native node/edge field plus safe metadata |
| NEO-14 | Scoped stale reconciliation | Generation ownership and atomic activation confined to one namespace/scope key |
| NEO-15 | Never mutate SQLite or reverse write | Read-only service dependency; hash SQLite and sidecars before/after tests |
| NEO-16 | Automated acceptance | Protocol fake for exhaustive failures plus disposable Neo4j acceptance lane |
| NEO-17 | Neo4j optional for ordinary Gortex | Driver constructed only inside explicit push; ordinary tests run with no server |
| STORE-01 | SQLite sole authority | No Neo4j-backed implementation of graph store/query interfaces |
| STORE-02 | No index-time dual write | Projection package is unreachable from indexer mutation paths |
| STORE-03 | Manual rebuildable projection | Full selected-snapshot push remains the only operation |
| STORE-04 | No continuous synchronization | No watcher, outbox, scheduler, or daemon startup dependency |
| STORE-05 | No ordinary Neo4j dependency or reverse writes | Optional connection and integration-test boundary only |
</phase_requirements>

## Summary

The phase should add a new projection subsystem, not extend raw Cypher-file replay into a remote loader. The existing exporter snapshots into memory and emits literal `CREATE` statements, while the SQLite store already has bounded scoped iterators. Specifically, `WriteCypher` calls `snapshot`, then writes nodes and edges; node and relationship output uses `CREATE`. [VERIFIED: internal/exporter/cypher.go:28-48,51-54,97-131] The scoped store uses the verbatim page constant `const scopedProjectionPage = 256`, keyset-pages nodes, and freezes an edge scan at the current `view_gen`/maximum edge ID. [VERIFIED: internal/graph/store_sqlite/scoped_projection.go:10-24,52-70,132-175]

Use the official Neo4j Go driver v6 and set the supported server floor to Neo4j 5.26 LTS. The official current install guide says v6 supports Neo4j 4.4, 5.x, 2025.x, and 2026.x, while the driver manual identifies 5.26 as LTS. [CITED: https://neo4j.com/docs/go-manual/current/install/] [CITED: https://neo4j.com/docs/go-manual/current/] The registry query returned `github.com/neo4j/neo4j-go-driver/v6 v6.3.0`, published 2026-09-21, and requiring Go 1.24; the repository declares `go 1.27.0`. [VERIFIED: Go module proxy query] [VERIFIED: go.mod:1-5] The 5.26 floor provides one stable server family while retaining relationship uniqueness constraints and dynamic labels/types, but the implementation should generate static, sanitized labels/types per batch to avoid the documented dynamic-token performance caveats. [CITED: https://neo4j.com/docs/cypher-manual/current/clauses/merge/]

The core correctness challenge is generation visibility. Do not overwrite active physical records while staging the next snapshot. Give each projected entity a stable logical key and a generation-specific physical key, stage all records under a pending manifest, then switch the manifest/active flags and reconcile the former generation in one final transaction. This is the only straightforward way to satisfy the locked rule that the old complete snapshot remains visible while multiple new batches are written. [ASSUMED]

**Primary recommendation:** Implement a shared `internal/neo4jprojection` service around a small transport protocol, a context-aware immutable SQLite snapshot reader, and the official v6 driver; stage generation-specific records, then atomically activate and clean only the owned namespace/scope.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI/MCP request parsing | API / Backend | -- | Adapters validate protocol fields and call one service |
| Scope resolution | API / Backend | Database / Storage | Resolve workspace/project/repository before any graph read |
| Authoritative snapshot | Database / Storage | API / Backend | SQLite owns facts; service owns orchestration only |
| Property/identity mapping | API / Backend | Database / Storage | Pure mapping from authoritative records to projection records |
| Batch writes and constraints | Database / Storage | API / Backend | Neo4j transaction semantics own remote atomicity |
| Generation activation/reconciliation | Database / Storage | API / Backend | Target manifest and ownership predicates prevent cross-scope deletion |
| Progress/result/redaction | API / Backend | CLI / MCP client | One structured model feeds both presentation surfaces |

## Project Constraints (from CLAUDE.md)

- Prefer Gortex graph tools over Read/Grep/Glob for indexed source investigation. [VERIFIED: CLAUDE.md:19-29]
- Build with `go build -o gortex ./cmd/gortex/` and gate the full repository with `go test -race ./...`. [VERIFIED: CLAUDE.md:5-10]
- Reuse existing symbols and folder conventions; do not create a second implementation. [VERIFIED: CLAUDE.md:156-173]
- No new dependency, pattern, or architectural layer without plan justification and confirmation. [VERIFIED: CLAUDE.md:175-179]
- Keep single-use logic local, apply the rule of three, and avoid speculative abstraction. [VERIFIED: CLAUDE.md:142-154]
- No repository `AGENTS.md` exists, so there are no additional project-local AGENTS directives. [VERIFIED: filesystem read attempt]

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.27.x | Service, adapters, tests | Repository module declares `go 1.27.0`. [VERIFIED: go.mod:1-5] |
| `github.com/neo4j/neo4j-go-driver/v6` | v6.3.0 | Bolt connectivity, sessions, managed transactions, error classification | Official Neo4j driver; current official docs use v6. [CITED: https://neo4j.com/docs/go-manual/current/] [VERIFIED: Go module proxy query] |
| Existing SQLite store | repository version | Sole authoritative snapshot | Scoped iteration already binds current store generation and bounds pages. [VERIFIED: internal/graph/store_sqlite/scoped_projection.go:10-24,52-70] |
| Existing Cobra and `mcp-go` surfaces | repository versions | `gortex neo4j push` and `neo4j_push` adapters | Existing command/tool architecture is already used by export. [VERIFIED: cmd/gortex/export.go:23-44; internal/mcp/tools_export.go:17-34] |

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| `context` | Go stdlib | Cancellation/deadlines | Every snapshot and Neo4j operation |
| `crypto/sha256` + canonical encoding | Go stdlib | Stable logical/physical projection keys | Relationship identity, generation ID, scope owner key |
| Protocol fake | in-repo | Deterministic transaction/failure tests | Unit and contract tests without Neo4j |
| Official Neo4j Community Docker image | 5.26 pinned patch | Disposable acceptance | Integration lane only; never ordinary build/startup [CITED: https://neo4j.com/docs/operations-manual/current/docker/introduction/] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Official v6 driver | Shelling out to `cypher-shell` | Harder cancellation, error typing, transaction control, secret handling, and test seams |
| Static sanitized token statements | Dynamic label/type `$(expr)` | Dynamic syntax is safer than string concatenation but has version-dependent index-planner caveats. [CITED: https://neo4j.com/docs/cypher-manual/current/clauses/merge/] |
| Protocol fake plus disposable server | Docker-only tests | Docker-only failure injection is slow and brittle; fake-only tests cannot prove real Cypher/constraint behavior |

**Installation:**

```bash
GOWORK=off go get github.com/neo4j/neo4j-go-driver/v6@v6.3.0
GOWORK=off go mod tidy
```

The `GOWORK=off` prefix is necessary in this checkout because the current local `go.work` references missing local modules; an unqualified module query failed on those paths during research. [VERIFIED: environment probe]

## Package Legitimacy Audit

The GSD legitimacy seam currently accepts only `npm`, `pypi`, and `crates`, so it cannot issue an `OK/SUS/SLOP` verdict for a Go module. [VERIFIED: package-legitimacy command response] The package is nevertheless named by Neo4j's official documentation and resolves through the Go module proxy. [CITED: https://neo4j.com/docs/go-manual/current/install/] [VERIFIED: Go module proxy query]

| Package | Registry | Version / Publish Date | Source | Verdict | Disposition |
|---------|----------|------------------------|--------|---------|-------------|
| `github.com/neo4j/neo4j-go-driver/v6` | Go module proxy | v6.3.0 / 2026-09-21 | Official Neo4j docs and GitHub organization | Gate unsupported for Go; authoritative provenance verified | Approved with planner noting seam limitation |

**Packages removed due to SLOP verdict:** none.
**Packages flagged as suspicious:** none; no Go verdict was available.

## Architecture Patterns

### System Architecture Diagram

```text
CLI: gortex neo4j push             MCP: neo4j_push
             \                         /
              +--> request validation + redaction
                            |
                            v
                 explicit scope resolver
             workspace + project + repositories
                            |
                 absent/ambiguous? --yes--> fail closed
                            |
                            v
                  named profile resolver
             non-secret config + env secret refs
                            |
                            v
               read-only SQLite snapshot handle
                            |
              +-------------+-------------+
              |                           |
           dry-run                     apply
              |                           |
      target census/schema       acquire namespace lock
      capability validation               |
              |                  create pending manifest
              |                           |
              |              bounded nodes -> transactions
              |              bounded edges -> transactions
              |                           |
              |                  all batches succeeded?
              |                    /             \
              |                  no               yes
              |                  |                 |
              |          incomplete result    atomic activation
              |          old active intact    + old-gen cleanup
              |                                    |
              +--------------------> shared redacted result/progress
```

### Recommended Project Structure

```text
internal/neo4jprojection/
├── service.go          # orchestration, dry-run/apply, result contract
├── model.go            # request, scope, progress, result, mapped records
├── identity.go         # stable owner/entity/relationship/generation keys
├── properties.go       # safe deterministic property projection
├── transport.go        # minimal protocol seam used by service and tests
└── neo4j.go            # official-driver implementation and Cypher
internal/config/        # machine-level named Neo4j profiles
internal/mcp/           # thin neo4j_push registration/handler
cmd/gortex/             # thin neo4j push Cobra adapter
```

This is a recommended mapping, not an existing path. [ASSUMED]

### Pattern 1: Logical identity separated from staged physical identity

**What:** Define `owner_key = hash(namespace, workspace, project, sorted repositories)`, `logical_node_key = hash(owner_key, Node.ID)`, and `physical_node_key = hash(logical_node_key, generation)`. Relationship keys include owner, authoritative endpoints, original kind, file path, line, origin, and any additional occurrence discriminator preserved by metadata. [ASSUMED]

**Why:** The repository's current logical edge key is verbatim `From`, `To`, `Kind`, `FilePath`, and `Line`; its provenance-bearing identity additionally folds in `Origin`. [VERIFIED: internal/graph/graph.go:72-81,139-155] Reuse that semantic contract instead of reducing identity to `(from,to,type)`.

**Activation:** Keep `active_generation` on one manifest per owner. Stage generation-specific node/relationship copies. In the final transaction, verify the pending manifest is complete and lock-owned, switch the active generation, mark/delete the previous generation only under the exact owner key, and release the lock. [ASSUMED]

### Pattern 2: Static-token, parameterized batch Cypher

Use one statement per sanitized node label or relationship type and pass rows as a parameter list. `UNWIND` turns a parameter list into rows, and `MERGE` provides match-or-create behavior. [CITED: https://neo4j.com/docs/cypher-manual/current/clauses/unwind/] [CITED: https://neo4j.com/docs/cypher-manual/current/clauses/merge/]

```cypher
UNWIND $rows AS row
MERGE (n:GortexNode:Function {physical_key: row.physical_key})
SET n += row.properties,
    n.logical_key = row.logical_key,
    n.owner_key = $owner_key,
    n.generation = $generation
```

The label `Function` is an example derived from the existing sanitizer, whose documented mapping says `"function" → "Function"`; all runtime tokens must come only from a strict sanitizer, never raw user/data text. [VERIFIED: internal/exporter/exporter.go:261-273]

### Pattern 3: Shared structured operation contract

The service request should contain profile, namespace, explicit scope selectors, dry-run, batch size, and timeout. The result should always contain operation ID, profile name (not URI), owner/scope, source snapshot generation, pending/active generation IDs, phase, complete boolean, dry-run boolean, expected/applied/skipped/stale counts, elapsed time, cancellation, and sanitized error code/message. [ASSUMED]

CLI human progress is a renderer over progress events. CLI `--json` and MCP return the same final result struct. MCP cancellation is the handler context; CLI uses `cmd.Context()` and signal cancellation. [ASSUMED]

### Pattern 4: Machine-level profiles, environment-only secrets

Add profiles to user-level config, not checked-in repo config, because the existing user configuration is explicitly machine-level and already keeps machine policy outside repository control. [VERIFIED: internal/config/global.go:68-99] Recommended fields are `uri`, `database`, `username_env`, `password_env`, optional `realm`, `encrypted`, `connection_timeout`, `transaction_timeout`, and `max_retry_time`; every invocation requires `--profile`/`profile`. [ASSUMED]

Reject URI userinfo, reject missing env references or missing env values, and never include resolved credentials or the raw URI in result/error/progress objects. The official driver docs support `NewDriver`, `BasicAuth`, `VerifyConnectivity`, and explicit database selection. [CITED: https://neo4j.com/docs/go-manual/current/] [CITED: https://neo4j.com/docs/go-manual/current/query-simple/]

### Pattern 5: Deterministic property envelope

Project every typed native field, including node columns currently omitted by the file exporter (`start_column`, `end_column`, node `origin`, `stub`, `fetched_at`) and edge fields (`kind`, `tier`, `alias`, plus safe metadata). The native `Node` struct's verbatim typed fields include `ID`, `Kind`, `Name`, `QualName`, `FilePath`, `StartLine`, `EndLine`, `StartColumn`, `EndColumn`, `Language`, `Meta`, `RepoPrefix`, `WorkspaceID`, and `ProjectID`. [VERIFIED: internal/graph/node.go:310-347] The native edge struct includes `From`, `To`, `Kind`, `FilePath`, `Line`, `Confidence`, `ConfidenceLabel`, `Origin`, `Tier`, `CrossRepo`, `Context`, `ReturnUsage`, `Via`, `Alias`, `NameOnly`, and `Meta`. [VERIFIED: internal/graph/edge.go:563-630]

Reserve a `gortex_` prefix for system fields. Encode metadata keys with a reversible percent/hex escape; on collision after normalization, append a deterministic short hash of the original key and store a key-map property as canonical JSON. Preserve supported scalar and homogeneous scalar-list values natively; encode nested/maps/mixed lists as canonical JSON strings plus an encoding marker. Reject/omit non-finite floats and unsupported runtime values with counted warnings, never `fmt.Sprint` them silently. [ASSUMED]

Apply a case-insensitive sensitive-key deny policy (`password`, `passwd`, `secret`, `token`, `credential`, `private_key`, `api_key`, `authorization`, and close normalized variants) before encoding, and test canary values across output, logs, errors, and Neo4j. [ASSUMED] OWASP says passwords, access tokens, database connection strings, encryption keys, and primary secrets should usually not be logged. [CITED: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html]

### Anti-Patterns to Avoid

- **Mutating active records during staging:** violates prior-generation visibility on failure.
- **Using the file exporter as the apply protocol:** it loads an in-memory snapshot and emits `CREATE`. [VERIFIED: internal/exporter/cypher.go:28-48,51-54,97-131]
- **`MERGE` without constraints:** official docs warn constraints are needed to prevent unintended/concurrent duplicates. [CITED: https://neo4j.com/docs/cypher-manual/current/clauses/merge/]
- **String-concatenating raw labels/types:** sanitize to a closed identifier grammar; parameterize all values.
- **Defaulting scope/profile:** this directly contradicts D-02/D-04.
- **Deleting `MATCH (n:GortexNode)`:** current export comments present this as a reset mechanism, but Phase 1 must never touch another namespace/scope. [VERIFIED: internal/exporter/cypher.go:11-14]
- **Depending on Neo4j at daemon startup:** construct the driver only inside explicit push.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bolt protocol/pooling/routing | Custom network client | Official Neo4j Go driver v6 | Connectivity, sessions, retries, errors, TLS/routing |
| Retry classification | String matching error messages | Managed transactions and typed Neo4j errors | Official docs say messages are unstable and codes are stable. [CITED: https://neo4j.com/docs/go-manual/current/query-simple/] |
| Transaction retries | Ad hoc infinite loop | Driver managed transaction with bounded `MaxTransactionRetryTime` | Callbacks may be retried and must be idempotent. [CITED: https://neo4j.com/docs/go-manual/current/transactions/] |
| Batch expansion | One query per record | Parameter list plus `UNWIND` | Fewer round trips and bounded transaction units. [CITED: https://neo4j.com/docs/cypher-manual/current/clauses/unwind/] |
| Scope semantics | New Neo4j-specific selector | Existing scope resolver tightened to require explicit concrete scope | Prevents divergence between native and projection boundaries |
| Secret redaction | Scattered log cleanup | One safe profile/result/error model that never carries secret values | Redaction after formatting is too late |

**Key insight:** Keep custom code in the domain protocol -- scope ownership, identity, generation activation, and property mapping. Delegate transport, transaction, retry, and Cypher execution behavior to the official driver.

## Common Pitfalls

### Pitfall 1: Existing scoped iterators are not yet an immutable cancellable snapshot
**What goes wrong:** Nodes and edges can be observed from different moments, cancellation cannot interrupt SQLite queries promptly, and fatal read errors are converted through `panicOnFatal`.
**Why it happens:** Current methods call `s.db.Query`, use an internal fixed page, and node/edge scans are separate. [VERIFIED: internal/graph/store_sqlite/scoped_projection.go:87-129,145-175]
**How to avoid:** Plan a read-only snapshot handle using one SQLite read transaction/view generation, `QueryContext`, explicit repository allow-set, and service-controlled batch aggregation. Ensure node and edge iteration share the same snapshot descriptor. [ASSUMED]
**Warning signs:** Projection service directly calls current iterators without context or snapshot-lifetime tests.

### Pitfall 2: Empty edge-kind input currently means no edges
**What goes wrong:** A naive call intended to mean “all edge kinds” projects zero edges.
**Why it happens:** `EdgesInScopeSeq` returns an empty sequence when `len(kinds) == 0`. [VERIFIED: internal/graph/store_sqlite/scoped_projection.go:132-144]
**How to avoid:** Add an explicit all-kinds snapshot path or enumerate authoritative edge kinds from the store schema; do not rely on node iterator semantics.
**Warning signs:** Node counts are non-zero and edge counts are always zero.

### Pitfall 3: Relationship constraints are type-specific
**What goes wrong:** One generic constraint is assumed to cover all typed relationships.
**Why it happens:** Cypher relationship uniqueness constraints name a concrete relationship type. [CITED: https://neo4j.com/docs/cypher-manual/current/schema/constraints/create-constraints/]
**How to avoid:** Discover selected sanitized types during planning, verify/create one named uniqueness constraint per type for `physical_key`, and report constraint actions. Dry-run verifies but performs no DDL. [ASSUMED]
**Warning signs:** Only a node constraint exists or constraints are created after data batches.

### Pitfall 4: Stable key collapses evidence occurrences
**What goes wrong:** Two source occurrences between the same endpoints and kind become one relationship.
**Why it happens:** `(from,to,type)` omits file/line/origin.
**How to avoid:** Base the key on the existing authoritative identity contract and include collision-relevant metadata when current storage distinguishes it. [VERIFIED: internal/graph/graph.go:72-81,139-155]
**Warning signs:** Projected relationship count is lower than scoped SQLite count without an explicit omission reason.

### Pitfall 5: Constraint setup mutates dry-run
**What goes wrong:** `CREATE CONSTRAINT IF NOT EXISTS` is run during dry-run.
**How to avoid:** Dry-run uses `SHOW CONSTRAINTS`, connectivity, version/edition checks, and target census only. It reports missing/incompatible constraints as planned actions or failure according to capability contract. [ASSUMED]

### Pitfall 6: Activation is split across transactions
**What goes wrong:** New and old generations are both active, or neither is active, after interruption.
**How to avoid:** Keep final owner-manifest validation, activation pointer/flags, and stale reconciliation in one transaction. If target size makes deletion too large, atomically switch active generation first and perform physical garbage collection later without changing logical visibility. [ASSUMED]

### Pitfall 7: Driver automatic retry duplicates side effects
**What goes wrong:** Progress counters or manifest state advance twice when a transaction callback reruns.
**How to avoid:** Keep callback effects database-idempotent and update local progress only after `ExecuteWrite` returns success. Official docs state managed transaction callbacks may rerun. [CITED: https://neo4j.com/docs/go-manual/current/transactions/]

### Pitfall 8: SQLite “immutability” test ignores sidecars
**What goes wrong:** Main DB hash is unchanged while WAL/SHM or mutation receipts change.
**How to avoid:** Record existence, size, and SHA-256 for the database plus `-wal` and `-shm` after fixture setup and before invocation; compare after success, dry-run, cancellation, and remote failure. Also compare canonical node/edge queries. [ASSUMED]

## Code Examples

### Driver lifecycle and connectivity

```go
// Source: https://neo4j.com/docs/go-manual/current/
driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(user, password, ""))
if err != nil { return err }
defer driver.Close(ctx)
if err := driver.VerifyConnectivity(ctx); err != nil { return err }
```

The production code must source `uri`, `user`, and `password` from the resolved named profile/env references and must not include them in returned errors. [ASSUMED]

### Managed bounded write

```go
// Source: https://neo4j.com/docs/go-manual/current/transactions/
session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: database})
defer session.Close(ctx)
_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
    result, err := tx.Run(ctx, statement, map[string]any{"rows": rows})
    if err != nil { return nil, err }
    return result.Consume(ctx)
}, neo4j.WithTxTimeout(transactionTimeout))
```

### Constraint shape

```cypher
// Source: https://neo4j.com/docs/cypher-manual/current/schema/constraints/create-constraints/
CREATE CONSTRAINT gortex_node_physical_key IF NOT EXISTS
FOR (n:GortexNode) REQUIRE n.physical_key IS UNIQUE
```

Generate equivalent named relationship constraints for each sanitized selected relationship type. [ASSUMED]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Neo4j Go driver v5 | Official Go driver v6 | Current docs in 2026 | Use `/v6`; v6 supports servers 4.4 through 2026.x. [CITED: https://neo4j.com/docs/go-manual/current/install/] |
| Literal `CREATE` export | Constrained `MERGE` batches | Phase 1 | Retry-safe remote materialization instead of disposable script replay |
| Whole exporter snapshot | Keyset-paged scoped SQLite read | Existing store seam | Memory unit is page/batch, not graph [VERIFIED: internal/graph/store_sqlite/scoped_projection.go:10-24] |
| Dynamic token concatenation | Dynamic token syntax exists from Neo4j 5.26, but static sanitized statements remain preferable here | Neo4j 5.26+ | Avoid injection and documented dynamic planner caveats. [CITED: https://neo4j.com/docs/cypher-manual/current/clauses/merge/] |

**Deprecated/outdated:**
- Treating raw `CREATE` Cypher as an idempotent integration is wrong for this phase. [VERIFIED: internal/exporter/cypher.go:51-54,97-131]
- Using Neo4j-generated internal IDs as durable application identity is outside the locked authoritative-identity contract. [VERIFIED: REQUIREMENTS.md:18-20]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Generation-specific physical records plus manifest activation are the selected implementation | Summary / Patterns | Alternative may satisfy D-07 with less storage, but must prove old snapshot visibility |
| A2 | Recommended internal package/file layout | Architecture | Planner may adapt to neighboring conventions |
| A3 | Exact profile fields/default timeout/batch settings | Profile pattern | Must be fixed in plan/API contract before implementation |
| A4 | Reversible property encoding and sensitive-key deny list | Property pattern | Data loss, collision, or secret disclosure if incomplete |
| A5 | SQLite sidecar hash procedure fully proves no authoritative mutation | Pitfalls/Validation | Platform journaling behavior may require canonical DB comparison too |

## Open Questions

1. **What defaults should the public contract use?**
   - What we know: CONTEXT delegates batch, timeout, and cadence values.
   - Recommendation: lock `batch_size=500`, operation timeout `30m`, transaction timeout `30s`, driver retry ceiling `30s`, and progress at phase boundaries plus every 1,000 records or 1 second. [ASSUMED]
2. **Delete or retain inactive generations?**
   - What we know: D-05 permits inactive or removed records and D-07 requires prior active retention until success.
   - Recommendation: activation transaction switches visibility and deletes the former generation for the exact owner; failed pending generations remain inactive and are deleted by the next same-owner rerun before restaging. [ASSUMED]
3. **How should unsupported metadata be preserved?**
   - What we know: D-11 requires all safely representable data and deterministic collision handling.
   - Recommendation: native scalars/lists plus canonical JSON fallback and explicit encoding/key-map properties; count omissions and fail if an identity/scope/provenance field cannot be represented. [ASSUMED]

These are resolved recommendations for planning, not blockers. The planner should make them explicit task/API acceptance criteria rather than reopen broad alternatives.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | Build/tests | Yes | 1.27.1 | -- |
| Neo4j Go driver module | Compile | Resolvable | v6.3.0 | Module download during implementation |
| Docker CLI | Disposable acceptance | Yes, daemon unavailable | 29.8.0 | Protocol-seam tests; CI/service starts disposable Neo4j |
| `cypher-shell` | Manual inspection only | Yes | 2026.08.1 | Official Go driver |
| Neo4j server | Integration acceptance | No running server detected | -- | Pinned Docker service in integration lane |

**Missing dependencies with no fallback:** A real Neo4j 5.26 server is required for the final disposable-server acceptance lane; it is not required for development unit tests or ordinary Gortex workflows. [ASSUMED]

**Missing dependencies with fallback:** Local Docker daemon is stopped; use protocol tests locally and start Docker/CI service only for integration acceptance.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing`, existing `testify`/`rapid` where useful |
| Config file | `go.mod`; no separate test config |
| Quick run command | `GOWORK=off go test ./internal/neo4jprojection ./internal/mcp ./cmd/gortex -run 'Neo4j|Projection'` [ASSUMED] |
| Full suite command | `GOWORK=off go test -race ./...` |

### Phase Requirements to Test Map

| Req IDs | Behavior | Test Type | Automated Command | File Exists? |
|---------|----------|-----------|-------------------|-------------|
| NEO-01, NEO-10 | CLI/MCP parity and shared result | contract | quick run | No -- Wave 0 |
| NEO-02, NEO-14 | explicit fail-closed and scope isolation | unit/integration | projection package tests | No -- Wave 0 |
| NEO-03 | profile validation and redaction canaries | unit | projection/config tests | No -- Wave 0 |
| NEO-04..NEO-07 | stable keys, constraints, repeated upsert | unit + Neo4j | protocol and integration lanes | No -- Wave 0 |
| NEO-08 | bounded transaction sizes | protocol fake | projection package tests | No -- Wave 0 |
| NEO-09 | dry-run performs zero writes | protocol fake + Neo4j | projection/integration tests | No -- Wave 0 |
| NEO-11, NEO-12 | cancellation/error, replay, incomplete result | protocol fake | projection package tests | No -- Wave 0 |
| NEO-13 | all fields/property collision/unsupported values | table/property tests | projection package tests | No -- Wave 0 |
| NEO-15, STORE-01..05 | SQLite bytes unchanged; no coupling/reverse path | integration + architecture | projection tests plus full suite without Neo4j | No -- Wave 0 |
| NEO-16 | success/retry/stale/cancel/scope acceptance | disposable Neo4j | tagged integration command | No -- Wave 0 |
| NEO-17 | ordinary operation without Neo4j | regression | `GOWORK=off go test -race ./...` with no Neo4j env/server | Existing suite plus new gate |

### Required Acceptance Scenarios

1. Empty target success; verify counts, labels, typed relationships, properties, constraints, active manifest.
2. Identical rerun; exact one logical record per key and no count growth.
3. Changed snapshot; stale records disappear only in selected owner scope.
4. Neighbor namespace/workspace/project/repository canaries survive.
5. Failure after node and edge batches but before activation; old generation remains active, result is incomplete.
6. Cancellation during SQLite read and Neo4j write; prompt stop and old generation remains active.
7. Rerun after failed pending generation; cleanup/replay succeeds without duplicates.
8. Dry-run against missing and present constraints; zero target and SQLite mutations.
9. Secret canaries never appear in CLI human/JSON, MCP, logs, errors, transaction metadata, or target properties.
10. SQLite main file, WAL, and SHM existence/size/hash unchanged after every path, plus canonical node/edge equality.
11. Build, daemon startup, indexing, native queries, and full tests succeed with no Neo4j server/profile.

### Sampling Rate

- **Per task commit:** focused package/adapter tests under 30 seconds.
- **Per wave merge:** `GOWORK=off go test -race` for changed packages; protocol suite always.
- **Phase gate:** full `GOWORK=off go test -race ./...` plus disposable Neo4j acceptance green.

### Wave 0 Gaps

- [ ] Protocol fake with query/transaction recording, bounded-batch assertion, injected transient/permanent failures, blocking cancellation, and commit uncertainty.
- [ ] SQLite scoped fixture and byte/sidecar fingerprint helper.
- [ ] Property/identity golden fixtures covering collisions, unresolved targets, nested metadata, unsupported values, and secret canaries.
- [ ] CLI/MCP parity harness comparing normalized request and final JSON result.
- [ ] Disposable Neo4j 5.26 integration script/CI service, isolated by build tag or explicit environment gate so ordinary tests never require it.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Yes | Driver auth token built from env-referenced secrets; no URL credentials |
| V3 Session Management | No | No end-user web session; close driver sessions deterministically |
| V4 Access Control | Yes | Explicit scope owner predicates and least-privileged Neo4j account |
| V5 Input Validation | Yes | Strict profile/scope/namespace/batch bounds; sanitized labels/types; parameterized values |
| V6 Cryptography | Yes | Driver-supported encrypted transport and certificate verification; never custom crypto |
| V7 Error/Logging | Yes | Typed safe errors, secret-free structured results, log-injection sanitization |
| V8 Data Protection | Yes | Metadata secret filtering and no reverse write |

### Known Threat Patterns for Go/Neo4j Projection

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cypher injection via label/type/property key | Tampering | Closed sanitizer for tokens; parameters for all values [CITED: https://neo4j.com/docs/go-manual/current/query-simple/] |
| Credential disclosure in URI/output/log | Information Disclosure | Reject URI userinfo; env refs; safe profile/result/error types; canary tests |
| Cross-scope stale deletion | Tampering/Elevation | Exact owner key in every mutation and cleanup predicate; neighboring canaries |
| Namespace lock bypass | Tampering | Uniqueness-constrained manifest/lock plus compare-and-set transaction [ASSUMED] |
| Resource exhaustion via huge batch/metadata | Denial of Service | Min/max batch validation, transaction/operation deadlines, bounded values/progress |
| Malicious metadata log injection | Spoofing | Structured logs and CR/LF sanitization; OWASP validation guidance [CITED: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html] |
| Over-privileged target account | Elevation | Separate projection principal limited to selected database, schema inspection/constraint and data writes required by command |

## Sources

### Primary Repository Evidence (HIGH confidence)

- `internal/exporter/cypher.go` -- current in-memory `CREATE` serializer and projected fields.
- `internal/graph/store_sqlite/scoped_projection.go` -- current 256-row keyset paging and scope behavior.
- `internal/graph/node.go`, `internal/graph/edge.go`, `internal/graph/graph.go` -- authoritative fields and edge identity.
- `internal/indexer/workspace_resolve.go`, `internal/mcp/scope_resolve.go`, `internal/mcp/tools_core.go` -- current scope resolution and fail-closed foundations.
- `internal/config/global.go`, `internal/config/config.go` -- machine and repository configuration boundaries.
- `.planning/REQUIREMENTS.md`, phase CONTEXT, STORAGE-ADR, ARCHITECTURE, STACK, PITFALLS -- locked product and architecture boundary.

### Official Documentation (MEDIUM/LOW per configured classifier)

- https://neo4j.com/docs/go-manual/current/ -- official v6 driver lifecycle and connectivity.
- https://neo4j.com/docs/go-manual/current/install/ -- v6 install and server compatibility.
- https://neo4j.com/docs/go-manual/current/query-simple/ -- parameters, database selection, retries, typed errors.
- https://neo4j.com/docs/go-manual/current/transactions/ -- managed transactions, idempotent callbacks, timeout/session rules.
- https://neo4j.com/docs/cypher-manual/current/schema/constraints/create-constraints/ -- node/relationship constraints and backing indexes.
- https://neo4j.com/docs/cypher-manual/current/clauses/merge/ -- `MERGE`, constraints, dynamic token caveats.
- https://neo4j.com/docs/cypher-manual/current/clauses/unwind/ -- parameter-list batching.
- https://neo4j.com/docs/operations-manual/current/docker/introduction/ -- official disposable image behavior.
- https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html -- secret exclusion and logging validation.

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM -- official driver and current registry version verified; GSD package-legitimacy seam lacks Go support.
- Architecture: HIGH for existing seams and required boundaries; MEDIUM for the recommended physical-generation model until implemented against Neo4j.
- Pitfalls: HIGH -- grounded in current source contracts and official transaction/constraint behavior.

**Research date:** 2026-09-21
**Valid until:** 2026-10-21 for architecture; recheck driver/server patch versions immediately before dependency installation.
