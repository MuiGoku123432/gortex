# Phase 1: General Neo4j Projection Command - Pattern Map

**Mapped:** 2026-09-21
**Files analyzed:** 18 likely new/modified files
**Analogs found:** 16 / 18

## File Classification

The exact split inside `internal/neo4jprojection` is planner discretion, but these are the smallest single-responsibility files implied by CONTEXT.md and RESEARCH.md.

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/gortex/neo4j.go` | controller / CLI command | request-response | `cmd/gortex/db.go` + `cmd/gortex/provider.go` | exact role-match |
| `internal/mcp/tools_neo4j.go` | controller / MCP adapter | request-response | `internal/mcp/tools_export.go` | exact role-match |
| `internal/mcp/server.go` | config / registration | request-response | existing `NewServer` registration sweep | exact |
| `internal/config/global.go` | config / model | CRUD | existing `GlobalConfig`, `LoadGlobal`, `Save` | exact |
| `internal/neo4jprojection/model.go` | model | request-response | `internal/graph/store_sqlite/mutation_receipt.go` result model | role-match |
| `internal/neo4jprojection/service.go` | service | batch + request-response | `internal/mcp/tools_export.go::exportByFormat` plus scoped iterators | partial composite |
| `internal/neo4jprojection/identity.go` | utility | transform | mutation receipt deterministic sorted evidence | role-match |
| `internal/neo4jprojection/properties.go` | utility | transform | `internal/exporter/exporter.go` property/label functions | exact role-match, stricter contract needed |
| `internal/neo4jprojection/transport.go` | service protocol | batch | `internal/semantic/lsp/client.go::Transport` | exact seam pattern |
| `internal/neo4jprojection/neo4j.go` | service / transport | batch + request-response | `internal/semantic/lsp/client.go` transport implementation | role-match |
| `internal/graph/scoped_projection.go` | provider contract | streaming | existing `ScopedProjectionSequencer` | exact modification seam |
| `internal/graph/store_sqlite/scoped_projection.go` | provider / store | streaming + batch | existing scoped keyset iterators | exact modification seam |
| `internal/neo4jprojection/service_test.go` | test | batch + event-driven failure injection | LSP transport injection pattern | role-match |
| `internal/neo4jprojection/properties_test.go` | test | transform | exporter property conversion behavior | role-match |
| `internal/config/global_test.go` or neighboring config test | test | CRUD | `GlobalConfig` load/save seams | role-match |
| `internal/mcp/tools_neo4j_test.go` | test | request-response | MCP handler/registration conventions | role-match |
| `cmd/gortex/neo4j_test.go` | test | request-response | provider/export command adapter conventions | role-match |
| `internal/neo4jprojection/neo4j_integration_test.go` | integration test | batch + request-response | none in repository | no analog |
| `go.mod`, `go.sum` | config | dependency resolution | existing module files | exact |

## Pattern Assignments

### `cmd/gortex/neo4j.go` (CLI controller, request-response)

**Primary analogs:** `cmd/gortex/db.go`, `cmd/gortex/provider.go`, and `cmd/gortex/export.go`

**Nested Cobra command pattern** (`cmd/gortex/db.go:20-39`):

```go
var dbCmd = &cobra.Command{
    Use:   "db",
    Short: "Live database connectors",
}

var dbSchemaCmd = &cobra.Command{
    Use:   "schema",
    Short: "Introspect a live database and emit its schema as CREATE TABLE DDL",
    RunE:  runDBSchema,
}

func init() {
    dbSchemaCmd.Flags().StringVar(&dbPostgresDSN, "postgres", "", "...")
    dbCmd.AddCommand(dbSchemaCmd)
    rootCmd.AddCommand(dbCmd)
}
```

Copy this shape as `neo4jCmd` containing `neo4jPushCmd`. Keep `Args: cobra.NoArgs`, require `--profile`, `--namespace`, and an explicit scope selector, and use `cmd.Context()` rather than `context.Background()` so cancellation reaches the shared service.

**Named profile flag and validation pattern** (`cmd/gortex/provider.go:61-93,96-124`):

```go
providerAddCmd.Flags().StringVar(&providerAPIKeyEnv, "api-key-env", "",
    "env var holding the bearer key (omit for keyless local endpoints)")

func runProviderAdd(cmd *cobra.Command, args []string) error {
    name := strings.TrimSpace(args[0])
    cp := llm.CustomProvider{
        APIKeyEnv: strings.TrimSpace(providerAPIKeyEnv),
    }
    if err := registry.Add(name, cp); err != nil {
        return fmt.Errorf("provider add %q: %w", name, err)
    }
    // render through cmd.OutOrStdout()
    return nil
}
```

**Thin daemon/MCP adapter pattern** (`cmd/gortex/export.go:65-121`):

```go
toolArgs := map[string]any{
    "format": format,
    "repo":   exportRepo,
}

out, err := requireDaemonTool(repoPath, "export_graph", toolArgs)
if err != nil {
    return err
}
_, err = cmd.OutOrStdout().Write(out)
return err
```

For `neo4j push`, either call the same shared operation directly or route to `neo4j_push`, but do not duplicate projection orchestration. Human output goes to `cmd.ErrOrStderr()` while `--json` writes the shared result to `cmd.OutOrStdout()`.

---

### `internal/mcp/tools_neo4j.go` and `internal/mcp/server.go` (MCP adapter/registration, request-response)

**Analog:** `internal/mcp/tools_export.go`

**Tool registration pattern** (`internal/mcp/tools_export.go:18-34`):

```go
func (s *Server) registerExportTools() {
    s.addTool(
        mcp.NewTool("export_graph",
            mcp.WithDescription("..."),
            mcp.WithString("format", mcp.Description("...")),
            mcp.WithString("repo", mcp.Description("...")),
            mcp.WithBoolean("no_synthetic", mcp.Description("...")),
        ),
        s.handleExportGraph,
    )
}
```

Register `neo4j_push` with explicit `profile`, `namespace`, scope selectors, `dry_run`, batch/timeout overrides, and no confirmation parameter. Mark it through the repository's mutating-tool intent/scope mechanism because `dry_run=false` changes Neo4j.

**Handler parsing and error/result pattern** (`internal/mcp/tools_export.go:53-85,135-164`):

```go
args := req.GetArguments()
format := strings.ToLower(stringArgOrDefault(args, "format", "cypher"))

if g == nil {
    return mcp.NewToolResultError("export: graph is not initialised"), nil
}

return s.respondJSONOrTOON(ctx, req, map[string]any{
    "nodes": st.NodesWritten,
    "edges": st.EdgesWritten,
})
```

The new handler should parse into `neo4jprojection.Request`, resolve a concrete scope before opening the transport, call the shared service once, and return the service's `Result` directly through `respondJSONOrTOON`. Safe operation failures belong in the structured result; malformed arguments and unavailable graph/profile use the normal tool error response.

**Progress attachment pattern** (`internal/mcp/progress.go:29-51,72-82`):

```go
func newProgressReporter(ctx context.Context, sender notificationSender,
    token mcp.ProgressToken) progress.Reporter {
    if sender == nil || token == nil {
        return nil
    }
    return &mcpProgressReporter{ctx: ctx, sender: sender, token: token}
}

func (s *Server) progressCtx(ctx context.Context, req mcp.CallToolRequest) context.Context {
    // attach reporter only when _meta.progressToken exists
    return progress.WithReporter(ctx, r)
}
```

Call the service with `s.progressCtx(ctx, req)`. The shared service should use `progress.FromContext(ctx).Report(...)`; the MCP adapter must not invent a second progress contract.

**Server registration sweep** (`internal/mcp/server.go:1880-1910`):

```go
s.registerWikiTools()
s.registerExportTools()
s.registerAuditTool()
```

Add `s.registerNeo4jTools()` beside export registration. Do not construct a Neo4j driver in `NewServer`; NEO-17 requires driver creation only for an explicit push.

---

### `internal/config/global.go` (machine-level named profiles, CRUD)

**Primary analogs:** existing `GlobalConfig` and `internal/llm/registry/registry.go`

**Machine-level placement pattern** (`internal/config/global.go:68-99`):

```go
// GlobalConfig is the user-level config at ~/.gortex/config.yaml.
type GlobalConfig struct {
    Projects map[string]ProjectConfig `mapstructure:"projects" yaml:"projects,omitempty"`
    // MCP carries machine-level client startup policy. It intentionally lives
    // in the user config rather than a repo's .gortex.yaml.
    MCP GlobalMCPConfig `mapstructure:"mcp" yaml:"mcp,omitempty"`
}
```

Add a machine-owned named map, for example:

```go
type Neo4jProfile struct {
    URI         string `mapstructure:"uri" yaml:"uri"`
    Database    string `mapstructure:"database" yaml:"database,omitempty"`
    UsernameEnv string `mapstructure:"username_env" yaml:"username_env"`
    PasswordEnv string `mapstructure:"password_env" yaml:"password_env"`
}

type GlobalConfig struct {
    Neo4j map[string]Neo4jProfile `mapstructure:"neo4j" yaml:"neo4j,omitempty"`
}
```

Also add `neo4j` to `knownGlobalTopLevelKeys` (`internal/config/global.go:185-188`). Do not put profiles in repo-local `.gortex.yaml`.

**Environment-reference model** (`internal/llm/config.go:181-194`):

```go
type RemoteConfig struct {
    // APIKeyEnv names the environment variable holding the API key.
    // The key itself is never stored in the config file.
    APIKeyEnv string `mapstructure:"api_key_env" yaml:"api_key_env,omitempty"`
    BaseURL   string `mapstructure:"base_url" yaml:"base_url,omitempty"`
}
```

Resolve `os.Getenv(profile.UsernameEnv)` and `os.Getenv(profile.PasswordEnv)` only at invocation. Validate that the references and values are non-empty, reject URI userinfo, and return errors naming the profile/env variable but never the secret or raw credential-bearing URI.

**Load/save behavior** (`internal/config/global.go:239-295`):

```go
func LoadGlobal(configPath ...string) (*GlobalConfig, error) {
    // missing file returns an empty GlobalConfig
    if err := yaml.Unmarshal(data, gc); err != nil {
        return nil, fmt.Errorf("parsing global config: %w", err)
    }
    return gc, nil
}

func (gc *GlobalConfig) Save() error {
    globalConfigMu.Lock()
    defer globalConfigMu.Unlock()
    return saveGlobalConfigLocked(gc)
}
```

Use this existing config path and validation flow. Do not create a second profile registry unless the planner deliberately chooses the already-proven `providers.json` registry pattern.

---

### `internal/neo4jprojection/model.go` and `service.go` (shared service/result, batch + request-response)

**Composite analogs:** mutation receipt result semantics, `progress.Reporter`, and scoped iterators.

**Complete/incomplete result pattern** (`internal/graph/store_sqlite/mutation_receipt.go:16-62`):

```go
type sqliteMutationReceiptAccumulator struct {
    complete         bool
    incompleteReason string
    changedFiles     map[string]struct{}
}

func (a *sqliteMutationReceiptAccumulator) noteIncomplete(reason string) {
    a.complete = false
    if a.incompleteReason == "" {
        a.incompleteReason = reason // retain first cause
    }
}
```

The projection `Result` should similarly always exist once execution starts and retain the first safe failure cause. Include `Complete`, `Phase`, `DryRun`, `Cancelled`, operation/profile/namespace/scope IDs, source/pending/active generations, expected/applied/skipped/stale counts, elapsed duration, and a sanitized error code/message.

**Progress contract** (`internal/progress/progress.go:11-36`):

```go
type Reporter interface {
    Report(stage string, current, total int)
}

func FromContext(ctx context.Context) Reporter {
    if r, ok := ctx.Value(ctxKey{}).(Reporter); ok && r != nil {
        return r
    }
    return Nop{}
}
```

The service reports phase boundaries and bounded counters through this existing context contract. Keep progress side effects outside Neo4j managed transaction callbacks because callbacks may retry.

**Service sequencing to implement:**

1. Validate request, concrete scope, and profile without carrying resolved secrets into `Result`.
2. Open one authoritative read snapshot.
3. Connect and inspect server/capabilities/constraints/target census.
4. For dry-run, stop before DDL/DML and return planned counts/actions.
5. For apply, acquire same-owner namespace lock and create/reuse pending generation.
6. Stream mapped nodes and edges in bounded batches through `Transport`.
7. Activate and reconcile stale records in one final exact-owner transaction.
8. On any error/cancellation, return an incomplete result and leave the prior active generation untouched.

---

### `internal/graph/scoped_projection.go` and `internal/graph/store_sqlite/scoped_projection.go` (snapshot provider, streaming)

**Exact analog:** current scoped projection sequencer.

**Capability seam** (`internal/graph/scoped_projection.go:17-25`):

```go
type ScopedProjectionSequencer interface {
    NodesInScopeSeq(repoPrefixes, filePaths []string, kinds ...NodeKind) iter.Seq[*Node]
    EdgesInScopeSeq(repoPrefixes, filePaths []string, kinds ...EdgeKind) iter.Seq[ScopedEdgeRow]
    NodesLightInScopeSeq(repoPrefixes, filePaths []string) iter.Seq[*Node]
}
```

Do not make the Neo4j service depend on concrete SQLite internals. Add a narrow read-snapshot/cancellable capability or adapter that returns errors rather than relying on iterator panic behavior.

**Bounded keyset page pattern** (`internal/graph/store_sqlite/scoped_projection.go:10-24,55-71,84-128`):

```go
const scopedProjectionPage = 256

func (s *Store) NodesInScopeSeq(...) iter.Seq[*graph.Node] {
    return func(yield func(*graph.Node) bool) {
        for _, kind := range kindValues {
            query, args, ok := scopedNodeProjectionQuery(...)
            if !s.streamScopedNodes(query, args, false, yield) {
                return
            }
        }
    }
}

pageArgs := append(append([]any(nil), args...), lastID, scopedProjectionPage)
rows, err := s.db.Query(query, pageArgs...)
// close rows before yielding the bounded page
```

Preserve keyset paging and the rule that SQLite cursors close before yielding. The new snapshot API must use `QueryContext`, bind nodes and edges to one immutable read transaction/view generation, and expose cancellation/errors explicitly.

**Frozen edge boundary and hydration pattern** (`internal/graph/store_sqlite/scoped_projection.go:132-175,181-234`):

```go
var maxID int64
if err := s.db.QueryRow(
    `SELECT COALESCE(MAX(id), 0) FROM edges WHERE view_gen = ?`, s.viewGen,
).Scan(&maxID); err != nil { ... }

endpointIDs := make([]string, 0, len(page)*2)
endpoints := s.GetNodesByIDs(endpointIDs)
```

Retain a frozen edge boundary and page-wise endpoint hydration. Fix the current semantic trap: `EdgesInScopeSeq` returns no edges when no kinds are supplied. The projection snapshot needs an explicit all-edge-kinds path.

---

### `internal/neo4jprojection/identity.go` and `properties.go` (transform utilities)

**Analog:** `internal/exporter/exporter.go`, with important required corrections.

**Label conversion baseline** (`internal/exporter/exporter.go:261-273`):

```go
func nodeLabel(kind graph.NodeKind) string {
    if kind == "" {
        return "Unknown"
    }
    parts := strings.Split(string(kind), "_")
    for i, p := range parts {
        parts[i] = strings.ToUpper(p[:1]) + p[1:]
    }
    return strings.Join(parts, "")
}
```

Reuse the established visible shape (`function` -> `Function`) but enforce a strict closed Neo4j token grammar for both labels and relationship types. Add `GortexNode` universally and `GortexUnresolved` for unresolved placeholders.

**Property sanitizer baseline** (`internal/exporter/exporter.go:234-256`):

```go
func sanitizePropertyName(name string) string {
    var b strings.Builder
    for i, r := range name {
        ok := r == '_' || isASCIIAlpha(r) || (i > 0 && isDigit(r))
        if ok { b.WriteRune(r) } else { b.WriteRune('_') }
    }
    return b.String()
}
```

Do not copy this behavior unchanged: replacement with `_` is collision-prone. Phase 1 requires deterministic, collision-aware, reversible property encoding, reserved `gortex_` system fields, a key-map for escaped metadata, canonical JSON fallback, non-finite/unsupported value handling, and sensitive-key filtering before encoding.

**Deterministic output baseline** (`internal/exporter/exporter.go:196-210`):

```go
for k, v := range meta {
    safeKey := sanitizePropertyName(k)
    if safeKey == "" { continue }
    out = append(out, metaEntry{Key: safeKey, Value: coerceValue(v)})
}
sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
```

Keep sorted deterministic output and replace silent dropping/coercion with explicit warning/omission counts.

**Identity semantics:** hash a canonical length-delimited encoding, not string concatenation. Node logical identity includes exact owner key plus authoritative `Node.ID`; edge identity must preserve the existing authoritative occurrence fields (`From`, `To`, `Kind`, `FilePath`, `Line`, `Origin`, and any persisted discriminator). Physical identity additionally includes generation.

---

### `internal/neo4jprojection/transport.go` and `neo4j.go` (protocol and official-driver adapter)

**Analog:** `internal/semantic/lsp/client.go:73-92,94-152,277-299`

**Small injected protocol pattern:**

```go
type Transport interface {
    Start() (io.WriteCloser, io.Reader, error)
    Stop() error
    SendsShutdown() bool
    Description() string
}

func NewClientWithTransport(t Transport, logger *zap.Logger) (*Client, error) {
    stdin, stdout, err := t.Start()
    if err != nil { return nil, err }
    // client owns protocol, transport owns carrier lifecycle
}
```

Define only operations the service needs: connect/capability inspection, target census, constraint verification/creation, owner lock/pending generation, node batch, edge batch, atomic activation/reconciliation, and close. Keep Cypher and official-driver types behind the adapter so unit tests use a deterministic fake.

Production transport rules:

- Construct the official v6 driver only inside explicit push.
- Use explicit database selection and context-aware calls.
- Parameterize every value and use `UNWIND $rows` plus `MERGE`.
- Generate labels/types only from the closed sanitizer.
- Treat managed transaction callbacks as retryable and side-effect-free outside Neo4j.
- Never include resolved credentials or raw sensitive connection material in errors, progress, transaction metadata, or results.

---

### Tests (unit, contract, adapter, integration)

**Protocol fake pattern:** follow the injected-interface seam from `internal/semantic/lsp/client.go`. The fake should record ordered operations and rows, enforce maximum batch sizes, block until context cancellation, and inject transient, permanent, pre-activation, activation, and commit-uncertain failures. Keep it in `_test.go` unless production code needs it.

**Config tests:** use `t.Setenv` because `DefaultGlobalConfigPath` resolves environment fresh on every call (`internal/config/global.go:227-237`). Verify named lookup, missing profile, missing env reference/value, URI userinfo rejection, and that secret canaries never appear in errors or marshaled results.

**Snapshot/store tests:** use the real SQLite test fixture. Assert one snapshot generation for nodes and edges, deterministic ordering, bounded pages, explicit all-edge-kind semantics, prompt cancellation, and unchanged main DB/WAL/SHM fingerprints plus canonical node/edge equality.

**CLI/MCP parity tests:** normalize the request created by each adapter and compare final structured JSON. The test must prove both surfaces call the same service contract and that `dry_run=false` needs no second confirmation.

**Disposable Neo4j integration:** no close repository analog exists. Add an explicitly gated lane (build tag or required environment variable) using Neo4j 5.26. Ordinary `go test -race ./...` must pass with no profile/server. Cover empty push, identical rerun, changed snapshot stale cleanup, neighboring owner canaries, failure before activation, rerun recovery, cancellation, dry-run zero writes, typed labels/relationships, and secret canaries.

## Shared Patterns

### Scope Resolution and Fail-Closed Behavior

**Source:** `internal/mcp/scope_resolve.go:89-214`

```go
intersected := intersectRepoSets(narrowed, wsRepos)
if len(intersected) == 0 {
    return ResolvedScope{}, fmt.Errorf(
        "repo/project/ref filter resolves to nothing inside the active workspace ...")
}
resolved.RepoAllow = intersected
```

Apply to both CLI and MCP before snapshot reads. For projection, strengthen this further: the final repository allow-set must be explicit, concrete, non-empty, and identical for dry-run/apply. Never interpret nil as all repositories.

### Authentication and Secret Handling

**Source:** `internal/llm/config.go:181-194` and `internal/llm/registry/registry.go:54-88`

- Store env variable names, never values.
- Trust machine-level configuration by default; do not accept checked-in endpoint profiles.
- Validate endpoint scheme/URI and profile name before use.
- Result objects carry profile name, not credentials or raw URI.

### Progress

**Source:** `internal/progress/progress.go` and `internal/mcp/progress.go`

- Service emits through `progress.FromContext(ctx)`.
- MCP translates to monotonic `notifications/progress` ticks.
- CLI supplies a human renderer or no-op reporter.
- Reporting failures do not fail the projection.

### Error and Completion Semantics

**Source:** `internal/graph/store_sqlite/mutation_receipt.go:31-62`

- Preserve the first incomplete reason.
- Return deterministic sorted evidence/counts.
- Cancellation and remote failures produce an incomplete result rather than losing partial statistics.
- Prior active generation remains visible unless final activation succeeds.

### Bounded Data Access

**Source:** `internal/graph/store_sqlite/scoped_projection.go`

- Keyset paging, fixed bounded pages, cursor closed before yield.
- Context-aware read transaction shared by node and edge scans.
- Remote writes grouped by sanitized label/type and bounded by configured batch size.

### Registration and Optional Dependency

**Source:** `internal/mcp/server.go::NewServer`

Register the MCP tool during the normal registration sweep, but instantiate no Neo4j driver and make no network call at daemon startup. CLI/MCP ordinary startup and all unrelated tests remain Neo4j-independent.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/neo4jprojection/neo4j_integration_test.go` | integration test | batch + request-response | Repository has no disposable Neo4j/testcontainers lane or integration build-tag pattern. Use RESEARCH.md acceptance scenarios and an explicit gate. |
| generation activation Cypher inside `internal/neo4jprojection/neo4j.go` | service/transport | transactional batch | Existing exporter emits literal `CREATE` and destructive global wipe instructions; it is explicitly unsafe as an apply/reconciliation analog. Use official driver transaction patterns from RESEARCH.md. |

## Patterns Explicitly Not to Copy

- Do not copy `cmd/gortex/db.go` accepting credentials in a DSN; Phase 1 requires named profiles and env-secret references.
- Do not copy `internal/exporter/cypher.go` whole-graph snapshot or literal `CREATE` statements.
- Do not copy `printLoadInstructions` global `MATCH (n:GortexNode) DETACH DELETE n`; cleanup must use exact owner and generation predicates.
- Do not call current scoped iterators unchanged from the service: they lack `QueryContext`, explicit errors, one shared immutable snapshot, and all-edge-kind behavior.
- Do not add Neo4j as a graph store, indexer dependency, daemon-startup dependency, watcher, outbox, or reverse-write path.

## Metadata

**Analog search scope:** `cmd/gortex`, `internal/mcp`, `internal/config`, `internal/llm`, `internal/graph`, `internal/graph/store_sqlite`, `internal/exporter`, `internal/progress`, `internal/semantic/lsp`

**Strong analog files read:** 13

**Pattern extraction date:** 2026-09-21
