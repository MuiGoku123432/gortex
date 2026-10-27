# Phase 2: Thin Deterministic Native Tracer and Minimum Contracts - Pattern Map

**Mapped:** 2026-09-30
**Files analyzed:** 20 (4 fork-side, 16 Gortex-side incl. docs/CI)
**Analogs found:** 17 / 20 (the 3 without a code analog are planning/docs text edits)

Path conventions in this document:
- Gortex paths are repo-relative to the Gortex checkout.
- `<fork>/…` means the producer repo `github.com/MuiGoku123432/tree-sitter-cobol-upgrade` (local checkout named in CONTEXT.md). Only its module paths and source files are cited; no estate content.
- Every analog path below was checked with `git ls-files` (Gortex: 53 files, fork: 10 files, all tracked). No mirror or gitignored paths are named.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `<fork>/preprocessor/internal/tsadapter/parser.go` (add `AttestEmbedded`) | utility (attestation) | transform | same file `Attest` L177-193 | exact |
| `<fork>/preprocessor/analyze.go` (add `NewEmbeddedAnalyzer`) | service constructor | request-response | same file `NewAnalyzer` L37-45 | exact |
| `<fork>/preprocessor/pipeline.go` (split `validateSelectedShim`; optional Replace-aware version) | utility | transform | same file L461-501 | exact |
| `<fork>` tests (`analyze_test.go`, `internal/tsadapter/parser_test.go`) | test | request-response | `analyze_test.go` L35-100; `parser_test.go` L20-28, L92-109 | exact |
| `go.mod` / `go.sum` | config | — | `go.mod` L353-366 (commented `replace` blocks) | exact |
| `internal/parser/languages/cobol_grammar.go` (new, `//go:build !windows`) | extractor (parser tier) | transform (bytes → nodes/edges) | `internal/parser/languages/cobol.go` (shape) + `extractor_plugin.go` (stateful struct) | role-match |
| `internal/parser/languages/cobol_grammar_windows.go` (new, `//go:build windows`) | extractor stub | request-response (fail) | `internal/parser/languages/grammar_load_static.go` | exact |
| `internal/parser/languages/cobol_grammar_test.go` (new, `//go:build !windows`) | test | transform | `internal/parser/languages/cobol_test.go` + `grammar_load_static_test.go` (build header) | exact |
| `internal/parser/languages/cobol_grammar_windows_test.go` (optional, `//go:build windows`) | test | request-response | `internal/parser/languages/grammar_load_static_test.go` L19-27 | exact |
| `internal/parser/languages/register.go` L119 | config (registration) | — | same line | exact |
| `internal/indexer/source_revision.go` (new) | indexer hook | transform (stamp Meta) | `internal/indexer/indexer.go` L1642-1651 (once-cached repo state) + L1687-1703 (stamp file-node Meta) | role-match |
| `internal/indexer/indexer.go` (one call in `applyCoverageDomains`) | indexer | transform | same function L1664-1745 | exact |
| `internal/indexer/source_revision_test.go` (new) | test | file-I/O (temp git repo) | `internal/indexer/git_watcher_integration_test.go` L20-33 + `worktree_instance_test.go` L19-28 + `coverage_strip_test.go` L15-33 | exact |
| `cmd/gortex/cobol_tracer_test.go` (new, `//go:build !windows`) | integration test | request-response + CRUD (SQLite) | `cmd/gortex/daemon_integration_test.go` L44-181 + `cmd/gortex/daemon_mcp_checkout_cwd_test.go` L33-119 + `internal/indexer/direct_cold_symbol_fts_test.go` | role-match (composite) |
| `internal/neo4jprojection/properties_test.go` (add `TestProjectNode_CobolProvenance`) | test | transform | same file L57-74, L209-221 | exact |
| `.github/workflows/ci.yml` (+ `init-smoke.yml`, `skill-drift.yml`, `security.yml`, `bench-arm.yml`, `release.yml`, `publish-claude-plugin.yml`) | config (CI) | batch | `.github/workflows/release.yml` L590-600 (secret in step `env:` + guard) | role-match |
| `internal/parser/forest/cobolprobe/README.md` | docs | — | — (text edit; scrub L9-25) | n/a |
| `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/PROJECT.md` | docs | — | — | no analog |
| README / docs COBOL-paragraph claims | docs | — | — | no analog |

## Pattern Assignments

### `<fork>/preprocessor/internal/tsadapter/parser.go` — add `AttestEmbedded` (utility, transform)

**Analog:** same file, `Attest` (L177-193). Copy it and drop the `CheckoutIdentity(shimRoot)` read:
```go
func Attest(shimRoot string) (Identity, error) {
	compiled, err := CompiledIdentity()
	if err != nil {
		return Identity{}, err
	}
	checkout, err := CheckoutIdentity(shimRoot)
	if err != nil {
		return Identity{}, err
	}
	if compiled != expectedCompiledLanguageID {
		return Identity{}, errors.New("compiled forest language identity mismatch")
	}
	if checkout != expectedCheckoutArtifactID {
		return Identity{}, errors.New("forest shim checkout identity mismatch")
	}
	return Identity{CompiledLanguageID: compiled, CheckoutArtifactID: checkout}, nil
}
```
- The constants come from the generated `expected_identity.go` L5-8 (`expectedCompiledLanguageID = "f97452e1…"`). Keep the error string verbatim (`"compiled forest language identity mismatch"`). Gortex tests and the CI stock-grammar step match on it.
- Negative-test seam: package var `languagePointer = cobol.GetLanguage` (L26-29).

### `<fork>/preprocessor/analyze.go` — add `NewEmbeddedAnalyzer` (service constructor)

**Analog:** `NewAnalyzer` (L37-45):
```go
func NewAnalyzer(repoRoot string, catalog transform.Catalog) (Analyzer, error) {
	tool, err := validateSelectedShim(repoRoot)
	if err != nil {
		return Analyzer{}, err
	}
	return Analyzer{tool: tool, catalog: catalog, budget: tsadapter.DefaultBudget, ceiling: parseWallClockCeiling}, nil
}
```
- Mirror it exactly and swap only the validation call. The `Analyzer` fields stay unexported (L26-35 comment: "a caller cannot claim a parser identity it did not attest").
- `Analyze` (L86-105) is unchanged. It already fails with `"analyzer has no attested tool identity"` on a zero tool.

### `<fork>/preprocessor/pipeline.go` — split `validateSelectedShim` (utility)

**Analog:** L461-487 (build-info half L463-474, attest call L475, ToolIdentity assembly L479-486) and `dependencyVersion` L489-501.
- Split at L475: keep the build-info checks (`forestModulePath`, `goTreeSitterVersion = "v0.25.0"`, constants L29-32) shared, and let the embedded path call `tsadapter.AttestEmbedded()`.
- Optional (fork's call): in `dependencyVersion`, when `dependency.Replace != nil`, return `Replace.Path + "@" + Replace.Version` so `ForestModuleVersion` is not the misleading `v1.9.1`.

### `<fork>` tests (test)

**Analog 1:** `<fork>/preprocessor/analyze_test.go` `TestTracerAnalyzeDocument` (L35-100). This is the invented-fixture tracer. For the equivalence assertion, copy L90-99:
```go
wantTool, err := validateSelectedShim(repoRoot)
...
if h.Tool != wantTool { t.Fatalf("Tool = %+v, want %+v", h.Tool, wantTool) }
if h.ToolID != h.Tool.ID() { ... }
```
The new test compares `NewEmbeddedAnalyzer` handoff `Tool` against `validateSelectedShim(repoRoot)`.

**Analog 2:** `<fork>/preprocessor/internal/tsadapter/parser_test.go`:
- `TestParserSmokeRejectsUnavailableLanguage` L20-28 is the stub-`languagePointer` negative:
```go
original := languagePointer
languagePointer = func() unsafe.Pointer { return nil }
t.Cleanup(func() { languagePointer = original })
```
- `TestCompiledIdentityMatchesCheckout` L92-109 is the positive `Attest` pattern that `AttestEmbedded` should be checked against.

---

### `go.mod` / `go.sum` (config)

**Analog:** existing commented replace blocks, `go.mod` L353-366:
```
// In-tree sharded drop-in for github.com/mattn/go-pointer. The upstream
// guards a single map with one global RWMutex; ...
replace github.com/mattn/go-pointer => ./internal/thirdparty/go-pointer
```
- Add the forest `replace` in the same style, with a `//` comment explaining the reason (D-02: the shim declares the upstream path, and the enhanced grammar is pinned by go.sum).
- The stock require stays at `go.mod` L28 (`github.com/alexaandru/go-sitter-forest/cobol v1.9.1`).
- Add the new `require …/preprocessor` in the direct `require (` block (L5-290).
- Run everything with `GOWORK=off`. A stale `go.work`/`go.work.sum` exists in the repo root; it is gitignored at `.gitignore` L23-24. Deleting it is a user-local action, not a commit.

---

### `internal/parser/languages/cobol_grammar.go` (extractor, transform) — `//go:build !windows`

**Analog A (shape):** `internal/parser/languages/cobol.go`.

Imports (L1-9). Keep the same package and import style, and add the fork packages as full module paths with no aliases:
```go
package languages

import (
	"regexp"
	"strings"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)
```

Language/Extensions (L68-71). Claim the identical extension set so the registry swap is total:
```go
func (e *CobolExtractor) Language() string { return "cobol" }
func (e *CobolExtractor) Extensions() []string {
	return []string{".cob", ".cbl", ".cpy", ".COB", ".CBL", ".CPY"}
}
```

File node + program node + `EdgeDefines` (L73-103). Copy this ID/edge shape exactly; only the source of name/range changes:
```go
fileNode := &graph.Node{
	ID: filePath, Kind: graph.KindFile, Name: filePath,
	FilePath: filePath, StartLine: 1, EndLine: len(lines),
	Language: "cobol",
}
...
id := filePath + "::" + name
result.Nodes = append(result.Nodes, &graph.Node{
	ID: id, Kind: kind, Name: name,
	FilePath: filePath, StartLine: start, EndLine: end,
	Language: "cobol", Meta: meta,
})
result.Edges = append(result.Edges, &graph.Edge{
	From: fileNode.ID, To: id, Kind: graph.EdgeDefines,
	FilePath: filePath, Line: start,
})
```
- The extractor emits **unprefixed** IDs (`relPath::SYMBOL`). `Indexer.applyRepoPrefix` (`internal/indexer/indexer.go` L2255-2324) prefixes `n.ID`, `n.FilePath`, and edge endpoints, and sets `RepoPrefix`/`WorkspaceID`/`ProjectID`.
- For grammar-derived nodes, the symbol is the containment path (`OUTER/INNER`, `NAME#2`), `QualName` is the same, and `Meta["cobol_kind"]="program"` plus the `prov_*` keys are set.
- Interface assertion at the file tail (L317): `var _ parser.Extractor = (*CobolExtractor)(nil)`. Add the same for the new type in the `!windows` file and in the windows stub.

**Analog B (stateful extractor struct + constructor):** `internal/parser/languages/extractor_plugin.go` L375-407:
```go
type SubprocessExtractor struct {
	language string
	exts     []string
	...
	dropOnce sync.Once
}

func NewSubprocessExtractor(spec config.ExtractorPluginSpec, log *zap.Logger) *SubprocessExtractor { ... }

func (e *SubprocessExtractor) Language() string     { return e.language }
func (e *SubprocessExtractor) Extensions() []string { return e.exts }
```
- Use this for the `sync.Once` + cached `initErr` field layout (RESEARCH Pattern 1). The constructor stays argument-free (`NewCobolGrammarExtractor()`) so `register.go` needs no config.
- **Divergence (D-05):** `SubprocessExtractor.Extract` (L424-434) degrades failures to a file-only result and returns `nil` error. Do **not** copy that. The grammar extractor returns `nil, fmt.Errorf("…: %w", err)`, and the indexer then records a parse-failed skip node (`indexer.go` L3589-3607).

**Edge provenance fields** (RESEARCH Q5): set origin explicitly as in `internal/parser/languages/cpp_reffrm.go` L125-133:
```go
result.Edges = append(result.Edges, &graph.Edge{
	From: ownerID, To: "unresolved::" + t, Kind: graph.EdgeReferences,
	FilePath: filePath, Line: line,
	Origin:   graph.OriginASTResolved,
	...
})
```
For the `EdgeDefines` file→program edge, add `Confidence: 1.0, ConfidenceLabel: "EXTRACTED"` (constants/helpers: `internal/graph/edge.go` L698 `OriginASTResolved = "ast_resolved"`, `DefaultOriginFor` L827, `ConfidenceLabelFor` L1146).

**Producer API to call** (`<fork>/preprocessor/analyze.go`):
- `NewEmbeddedAnalyzer(transform.NewCatalog(nil))` once, inside `sync.Once`.
- `NewSourceID(relPath)` (L47-50).
- `an.Analyze(ctx, sourceID, src)` (L86-105), called inside a capacity-1 semaphore.
- Handoff fields (`<fork>/preprocessor/handoff/handoff.go` L33-48): `SourceID`, `OriginalContentID`, `Generated`, `Tool`, `ToolID`, `ParseConfigurationID`, `Ledger.ConfigurationID`, `Facts.Observations`, `Assessment{PolicyVersion, Grade, Reasons}` (`grade/grade.go` L102-107).
- Observation fields (`parsefacts/facts.go` L74-80): `Kind`, `Generated` span, `Original *sourcemap.Projection`, `Parent int`, `Affected bool`.

**Parser-module identity** (D-04 "commit"): no Gortex analog. `debug.ReadBuildInfo` is not used anywhere in Gortex source. Copy the loop shape from `<fork>/preprocessor/pipeline.go` L489-501 (`for _, dependency := range info.Deps { if dependency.Path != path { continue } … }`) and read `dependency.Replace`.

**Line helper:** `lineAt` exists in `internal/parser/languages/helpers_lines.go` L6. Do not re-implement it. The grammar path should not need it anyway, because ranges come from `Observation.Original`.

---

### `internal/parser/languages/cobol_grammar_windows.go` (extractor stub) — `//go:build windows`

**Analog:** `internal/parser/languages/grammar_load_static.go` (whole file, 31 lines). It uses a platform-gated sentinel error with an actionable message and a same-signature stub:
```go
//go:build !windows && static_link

package languages

import (
	"errors"
	"unsafe"
)

// errNoDlopen explains why a statically linked build cannot load a custom
// grammar, and what to do about it. ...
var errNoDlopen = errors.New(
	"custom tree-sitter grammars need dlopen, which the statically linked release binary does not have; " +
		"build gortex from source (go build ./cmd/gortex/) to use them, or use one of the built-in grammars")

// loadGrammarLanguage refuses to load a custom grammar in a static build.
// ... (long RATIONALE comment explaining the platform constraint)
func loadGrammarLanguage(_, _ string) (unsafe.Pointer, error) {
	return nil, errNoDlopen
}
```
- Keep the same exported constructor name, the same `Language()`/`Extensions()`, and `Extract` returning `nil, errCobolParserUnavailableOnWindows`.
- The sibling pair convention in the same package is `grammar_load_unix.go` (`//go:build !windows && !static_link`) and `grammar_load_windows.go` (`//go:build windows`). Name the pair `cobol_grammar.go` (`!windows`) and `cobol_grammar_windows.go` (the Go `_windows` suffix also implies the constraint, but the repo writes the tag explicitly anyway).
- If the approved-grammar constant is needed by both files, put it in the `!windows` file only. The stub never compares IDs.

---

### `internal/parser/languages/cobol_grammar_test.go` (test) — `//go:build !windows`

**Analog:** `internal/parser/languages/cobol_test.go`.

Imports (L1-10):
```go
package languages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)
```

Inline invented fixed-format fixture + Extract + scan (L12-54):
```go
src := []byte(`       IDENTIFICATION DIVISION.
       PROGRAM-ID. HELLO-WORLD.
       ...
`)
e := NewCobolExtractor()
require.Equal(t, "cobol", e.Language())
res, err := e.Extract("HELLO.cob", src)
require.NoError(t, err)
for _, n := range res.Nodes { switch n.Name { ... } }
```
- Edge lookup helper: `findEdgeTo(edges, to)` already exists at `cobol_test.go` L287. It is in the same package, so reuse it and do not redefine it.
- Build-tag header on a test file: copy line 1 style from `grammar_load_static_test.go` (`//go:build !windows && static_link`), using `//go:build !windows`.
- Test naming: `TestCobolGrammar_<Case>`, matching RESEARCH's Validation map (`_ProgramNode`, `_NestedIDs`, `_DuplicateOrdinal`, `_IDIgnoresLines`, `_DegradedAmber`, `_ApprovedGrammarPinned`, `_GrammarMismatchFails`, copybook skip).
- The mismatch test constructs the extractor with a wrong unexported `approvedGrammarID` field. This needs same-package access, which `package languages` provides.

### `internal/parser/languages/cobol_grammar_windows_test.go` (optional) — `//go:build windows`

**Analog:** `grammar_load_static_test.go` L19-27. It asserts the error, the `ErrorIs` sentinel, and actionable message text:
```go
ptr, err := loadGrammarLanguage("/any/libtree-sitter-thing.so", "tree_sitter_thing")
require.Error(t, err)
assert.Nil(t, ptr)
assert.ErrorIs(t, err, errNoDlopen)
assert.Contains(t, err.Error(), "build gortex from source")
```

---

### `internal/parser/languages/register.go` L119 (registration)

Current (L118-120):
```go
	reg.Register(NewPascalExtractor())
	reg.Register(NewCobolExtractor())
	reg.Register(NewJCLExtractor())
```
Replace only L119 with `reg.Register(NewCobolGrammarExtractor())` (D-16). Do not touch `cobol.go` or `cobol_test.go`. Neighbouring registrations use a `//` comment above when order matters (for example L18-29), and a one-line comment here is consistent.

---

### `internal/indexer/source_revision.go` (indexer hook, transform) — new

**Analog 1 (repo-scoped state cached on the Indexer):** `internal/indexer/indexer.go` L1642-1651, with fields at L493-499:
```go
func (idx *Indexer) loadCodeownersRules() []codeowners.Rule {
	idx.codeownersOnce.Do(func() {
		rules, _, ok := codeowners.LoadFromRepo(idx.rootPath)
		if !ok {
			return
		}
		idx.codeownersRules = rules
	})
	return idx.codeownersRules
}
```
- `idx.rootPath` is at `indexer.go` L246.
- RESEARCH wants a snapshot per bulk pass and a fresh one per incremental call, so a lifetime `sync.Once` is not enough on its own. Keep the per-pass cache minimal (for example, reset at the start of `IndexCtx`), or sample on each call if that is cheap enough, and say which in the plan.

**Analog 2 (stamp Meta onto the file-shaped node inside the hook):** `indexer.go` L1687-1703 (codegen):
```go
for _, n := range result.Nodes {
	if n.Kind == graph.KindFile && n.FilePath == relPath {
		if n.Meta == nil {
			n.Meta = map[string]any{}
		}
		codegen.MarkFileNode(n.Meta, marker)
		break
	}
}
```
The stamp iterates `result.Nodes` and touches only nodes whose `Meta` already has `prov_revision_content_id`. It returns immediately when there are none, so other languages pay one loop.

**Git snapshot:** `internal/gitstate/dirty.go`.
- `SampleDirty(ctx, dir)` (L268-286) returns `DirtySnapshot{HeadCommit, Entries []DirtyEntry{Path, Kind, …}}` (L58-94).
- `Entries[].Path` is relative to the **worktree root** (L59-61 and L274-276). Map it to `relPath`, which is relative to `idx.rootPath`.
- Errors wrap `ErrDirtyUnavailable` (L26). Use `errors.Is(err, gitstate.ErrDirtyUnavailable)` to choose between the absence reasons `not_a_git_worktree` and `git_unavailable`.
- An empty `HeadCommit` maps to `no_head_commit`, and a matching dirty entry maps to `working_tree_differs_from_head`.
- Other in-package gitstate import sites that can be used as style references: `internal/indexer/worktree_prefix.go` (L3-10 import block, L85 `gitstate.Inventory(ctx, root)`).

### `internal/indexer/indexer.go` — one call in `applyCoverageDomains`

**Insertion point:** `applyCoverageDomains` L1664-1745. Existing gates look like:
```go
if idx.config.Coverage.IsEnabled("codegen") { ... }
entrypoints.Detect(relPath, lang, result.Nodes, result.Edges)
if !idx.config.Coverage.IsEnabled("function_shape") {
	stripFunctionShape(result)
}
```
Add one unconditional line (for example `idx.stampSourceRevision(relPath, result)`) next to `entrypoints.Detect` (L1720). This is an ungated call, and D-18 approves exactly one additive line.

**Gotcha:** the bulk path calls `applyCoverageDomains` only when `!skipped && !omitSecondarySourceScans` (L3618-3626). Incremental paths call it at `file_delta.go` L161 and `indexer.go` L4658. A COBOL file that is quarantined or timed out gets no VCS stamp. That is acceptable because it gets no program node either, but it should be noted in the plan.

### `internal/indexer/source_revision_test.go` (test, temp git repo)

**Analog 1 (git runner isolated from dev config):** `internal/indexer/git_watcher_integration_test.go` L20-33. `runGit` is already defined in package `indexer`, so reuse it:
```go
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, string(out))
}
```
**Analog 2:** `internal/indexer/worktree_instance_test.go` L19-28 (`gitInitRepo(t, dir)`, already defined, so reuse it) and L37-44 (`exec.LookPath("git")` skip guard, `writeFile`, `add`, `commit`).

**Analog 3 (direct unit on an `ExtractionResult`):** `internal/indexer/coverage_strip_test.go` L15-33. It builds a `parser.ExtractionResult` literal and calls the pass directly. Do the same with a node carrying `Meta{"prov_revision_content_id": "…"}`, which keeps this test independent of the COBOL grammar and Windows-safe.

Helpers available in package `indexer`: `writeFile` (`indexer_test.go` L63-66) and `newTestIndexer(g)` (`indexer_test.go` L68-74).

---

### `cmd/gortex/cobol_tracer_test.go` (integration test) — `//go:build !windows`

This is a composite of four analogs.

**Analog 1 — real daemon harness:** `cmd/gortex/daemon_integration_test.go` `spinUpDaemonWithConfig` L44-124:
- L49-53: `testenv.ShortTempDir(t)`, `t.Setenv("GORTEX_DAEMON_SOCKET", …)`, `t.Setenv("GORTEX_DAEMON_PIDFILE", …)`.
- L62-73: `g := graph.New()`, `languages.RegisterAll(reg)`, `indexer.New(g, reg, config.Default().Index, zap.NewNop())`, `config.NewConfigManager(configPath)`, `indexer.NewMultiIndexer(g, reg, idx.Search(), cm, zap.NewNop())`. **Swap `g` for the SQLite store.**
- L75: `mi.TrackRepoCtx(context.Background(), config.RepoEntry{Path: trackedRoot})`.
- L78-82: `gortexmcp.NewServer(eng, g, idx, nil, zap.NewNop(), nil, gortexmcp.MultiRepoOptions{MultiIndexer: mi, ConfigManager: cm})`. Nothing sets an LLM service here, which is the TRACE-07 proof point.
- L88-104: `indexer.NewCheckoutLifecycle(...)`, `daemon.New(socket, "test", zap.NewNop())`, `&realController{…}`, `newMCPDispatcher(srv, mi, zap.NewNop())`.
- L106-123: `d.Listen()`, `go d.Serve()`, cleanup via `d.Shutdown()`, `require.Eventually(daemon.IsRunningAt(socket))`.

MCP frame round-trip, from `TestDaemon_EndToEnd_GraphStatsOverMCPProxy` L131-181:
```go
client, err := daemon.DialTo(socket, daemon.Handshake{
	Mode: daemon.ModeMCP, CWD: trackedRoot, ClientName: "integration-test",
	Tools: cliLegacyToolSurface, ToolsMode: cliLegacyToolMode,
})
initFrame := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"integration","version":"0"}}}`)
...
toolFrame := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"graph_stats","arguments":{}}}`)
...
content, ok := resp.Result["content"].([]any)
text, ok := content[0].(map[string]any)["text"].(string)
```
Replace `graph_stats` with `get_symbol` (`{"id": "<prefixed id>", "detail": "full"}`) and with `search_symbols` (`{"query": "<PROGRAM>"}`).
- **Use the `cliLegacyToolSurface` handshake.** The default facade surface hides legacy tools (`cmd/gortex/daemon_mcp_test.go` L512-529 asserts `graph_stats` is hidden).

**Analog 2 — SQLite store under a real MultiIndexer + MCP server in `cmd/gortex`:** `cmd/gortex/daemon_mcp_checkout_cwd_test.go` `checkoutCWDSetup` L33-119:
```go
store, err := store_sqlite.Open(filepath.Join(dir, "store.sqlite"))
require.NoError(t, err)
t.Cleanup(func() { _ = store.Close() })
...
reg := parser.NewRegistry()
languages.RegisterAll(reg)
idx := indexer.New(store, reg, config.Default().Index, zap.NewNop())
mi := indexer.NewMultiIndexer(store, reg, search.NewNull(), cm, zap.NewNop())
t.Cleanup(func() { _ = mi.Close(context.Background()) })
res, err := mi.TrackRepoCtx(ctx, config.RepoEntry{Path: primaryRoot})
prefix := res.RepoPrefix
...
srv := gortexmcp.NewServer(query.NewEngine(store), store, idx, nil, zap.NewNop(), nil,
	gortexmcp.MultiRepoOptions{MultiIndexer: mi, ConfigManager: cm})
```
Import block L3-26 is the reference for `store_sqlite`, `search`, and `query` imports in `package main`. Use `res.RepoPrefix` to build the expected ID, and never hard-code a prefix.

**Analog 3 — shadow disabled + reopen proves persistence:**
- `internal/indexer/direct_cold_symbol_fts_test.go` L20-41: `t.Setenv("GORTEX_SHADOW_MAX_FILES", "0") // refuse the shadow: direct disk parse`.
- `internal/indexer/clone_sqlite_persistence_test.go` L53-78: open, index, `store.Close()`, `store_sqlite.Open(dbPath)` again, then `reopened.GetNode(id)` with `node.Meta[...]` assertions.

**Analog 4 — AddBatch-recording wrapper:** embed the concrete pointer so optional-interface assertions keep working (`direct_cold_symbol_fts_test.go` L83-86):
```go
type ftsFailingStore struct {
	*store_sqlite.Store
	writes int
}
```
Override method shape from `internal/indexer/full_topology_transaction_test.go` L232-243 (mutex because parse workers are concurrent under `-race`):
```go
type countingIndexAllStore struct {
	graph.Store
	mu      sync.Mutex
	batches int
}
func (s *countingIndexAllStore) AddBatch(nodes []*graph.Node, edges []*graph.Edge) {
	s.mu.Lock()
	s.batches++
	s.mu.Unlock()
	s.Store.AddBatch(nodes, edges)
}
```
- The production direct path calls `b.store.AddBatch(nodes, edges)` (`internal/indexer/parse_graph_batch.go` L130). `store_sqlite.Store` also has `AddBatchChecked` (`store.go` L1562). Record node IDs in both overrides so a later switch to the checked path cannot make the test pass vacuously.
- Use `*store_sqlite.Store` embedding, not `graph.Store`, so `Catalog()`, `BulkLoader`, and `SymbolSearcher` stay reachable (other examples: `internal/indexer/multi_cold_orchestration_test.go` L342-347, `internal/indexer/fts_normalization_lifecycle_test.go` L27).

**CLI verb (TRACE-05):** `cmd/gortex/call_test.go` `newCallTestCmd` L17-35 resets the package flag vars and binds a buffer:
```go
callIndex = "."
callJSON = ""
callArgs = nil
callFormat = "json"
...
cmd := &cobra.Command{Use: "call", RunE: runCall}
cmd.SetOut(buf)
```
- For the real relay, do **not** stub `callDaemonTool` (`call.go` L32 `var callDaemonTool = requireDaemonTool`). Set `callIndex = trackedRoot` (the default `"."` is the test process cwd) and `callArgs = []string{"id=<prefixed id>", "detail=full"}`, then `runCall(cmd, []string{"get_symbol"})`.
- `runCall` flow is at `call.go` L95-150.
- `GORTEX_DAEMON_SOCKET` is already set by the harness. `newCallTestCmd` is in the same package, so reuse it.

**Retrieval gotchas** (`internal/mcp/tools_core.go` L1709-1752):
- `detail` defaults to `"brief"` (L1741), and `Brief()` has no Meta. Always pass `detail=full`; the node is under `"node"` (L1750-1751).
- `ensureFresh([]string{parts[0]})` (L1716-1718) may re-index the file on lookup. Assert the AddBatch recorder *before* the MCP/CLI calls, or tolerate extra calls.

**Optional store-independent row check:** `internal/testutil/graphfixture/fixture.go` `Canonical(path)` L104-146. Its non-`owner_key` branch (L120-124) reads the real `nodes` schema including `hex(meta)`. Call it on the closed DB path.

---

### `internal/neo4jprojection/properties_test.go` — add `TestProjectNode_CobolProvenance`

**Analog:** same file, `TestProjectionProperties` L57-74 and `TestProjectionSecretFiltering` L209-221:
```go
projected, warnings, err := projectNode("owner", "generation", &graph.Node{ID: "n", Kind: graph.KindFunction, Meta: map[string]any{...}})
if err != nil { t.Fatal(err) }
...
if warnings.Unsupported != 2 || warnings.Secret != 1 { t.Fatalf(...) }
```
- The file uses stdlib `testing` with `t.Fatalf`, not testify (imports L3-14). Keep that style.
- Assert `projected.Properties["meta_prov_<key>"]` for each key, `warnings.Secret == 0`, and `warnings.Unsupported == 0`. Property naming is `"meta_" + encodeMetadataKey(key)` (`properties.go` L101-106), and the key map is `gortex_meta_key_map` (L146).
- `sensitiveMetadataKey` (L171-180) **strips `_`, `-`, `.`, and spaces before substring matching**. Check the concatenated form of every key (for example `prov_parser_tool_id` becomes `provparsertoolid`) for `token|secret|credential|password|passwd|privatekey|apikey|authorization`. That makes a table-driven "no key is sensitive" assertion over the full `prov_*` list worthwhile.

---

### CI workflows (config, batch)

**Analog for passing a secret into one step:** `.github/workflows/release.yml` L590-600:
```yaml
        shell: bash
        env:
          SCOOP_BUCKET_TOKEN: ${{ secrets.SCOOP_BUCKET_TOKEN }}
        run: |
          set -euo pipefail
          if [ -z "${SCOOP_BUCKET_TOKEN:-}" ]; then
            echo "SCOOP_BUCKET_TOKEN not set; skipping scoop manifest publish"
            exit 0
          fi
```
- Same shape, but **fail** instead of skip (D-13 says no skip-when-missing). The RESEARCH Q8 block is the step body.
- `shell: bash` is already used on Windows-capable steps (`release.yml` L487, L504, L531). `ci.yml` L41-44 documents a PowerShell tokenisation pitfall, so keep the credential step on `shell: bash`.
- Job-level `env:` precedent: `publish-claude-plugin.yml` L49-53. Put `GOPRIVATE` at job or workflow level.

**Every setup-go site that needs the step and `cache: false`.** All use the pinned `actions/setup-go@b7ad1dad…` with `go-version-file`:
- `ci.yml`: L23 (test, 3 OS), L70 (build-linux-static, which also runs `go test … ./internal/parser/languages/` at L91), L111 (lint, including `go mod tidy` diff at L127-134), L141 (build-onnx), L179 (benchmark).
- `bench-arm.yml` L17, L56. `init-smoke.yml` L27. `skill-drift.yml` L35. `release.yml` L33 (build-darwin), L478 (release-windows). `publish-claude-plugin.yml` L78 (`go-version-file: gortex/go.mod`, so the checkout is in a subdirectory).
- `security.yml` L21-24 **explicitly sets `cache: true`**, so flip it. The `golang/govulncheck-action` at L25-28 runs its own setup-go (`go-version-file: go.mod`). Verify whether it caches and needs the credential [ASSUMED].

**Release gotcha (not covered in RESEARCH):** the linux release runs inside `docker run … ghcr.io/goreleaser/goreleaser-cross:v1.27 release --clean` (`release.yml` L239-246) with only `-e GITHUB_TOKEN` passed. The host `git config --global insteadOf` and `GOPRIVATE` do **not** reach the container. The plan must pass `-e GOPRIVATE` plus credentials, or pre-download on the host and mount `GOMODCACHE` with `GOFLAGS=-mod=mod` and `GOPROXY=off`. Otherwise the release breaks at module download. D-19 accepts the embedding but not a broken release job, so surface this to the user.

**Stock-grammar negative step (BASE-02):** ubuntu-only step in `ci.yml` `test`, gated by `if: matrix.os == 'ubuntu-latest'` (same style as L31/L46/L50). The body is RESEARCH Validation row BASE-02 (`go mod edit -modfile=… -dropreplace …` then `! go test -modfile=…`).

---

### Docs

- `internal/parser/forest/cobolprobe/README.md`: L9 names `97ac9f1`, L11-12 has go.work activation with an absolute developer path, L19-20 has corpus paths, L25 has go.work rationale. Rewrite these to the go.mod pin and `GOWORK=off go test ./internal/parser/forest/cobolprobe/ -enhanced-parser` (flag defined at `cascade_test.go` L14). **Do not carry any absolute path or corpus name forward.**
- `.planning/REQUIREMENTS.md` (BASE-01, BASE-03, "Out of Scope" baseline line), `.planning/ROADMAP.md` (Phase 2 SC1), `.planning/PROJECT.md` (baseline statements): name the new pinned fork commit and cite `f19029f` as the accepted v0.26.0 line (RESEARCH Q4 "Consequence for D-01").
- README/docs "COBOL paragraphs" claims: mark them temporarily unavailable until Phase 3 (D-15).

## Shared Patterns

### Platform build tags (Windows)
**Source:** `internal/parser/languages/grammar_load_static.go`, `grammar_load_windows.go`, `internal/mcp/physical_evidence_open_{nonwindows,windows}.go`, `internal/indexer/unreadable_file{,_windows}_test.go`.
**Apply to:** `cobol_grammar.go`, `cobol_grammar_test.go`, `cmd/gortex/cobol_tracer_test.go` (`//go:build !windows`), and `cobol_grammar_windows.go` (`//go:build windows`).
- Line 1 is the tag, then a blank line, then `package`.
- Both halves define the same symbol names, so callers such as `register.go` compile unchanged.
- Nothing outside the `!windows` files may import `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/...`, because its `internal/securefs` is `darwin || linux` only.
- `internal/indexer/source_revision*.go` must **not** import the fork. They key only on the `prov_revision_content_id` Meta string, so they stay untagged.

### Fail-closed errors
**Source:** `grammar_load_static.go` L14-16 (package-level `errors.New` sentinel with an actionable message) and CLAUDE.md (`fmt.Errorf("…: %w", err)`).
**Apply to:** the grammar extractor (cached `initErr`, grammar-ID mismatch, `Analyze` error) and the Windows stub.
- Returning `(nil, err)` from `Extract` is the supported failure contract. The indexer appends an `IndexError` and a `parseFailedFiles` skip node (`internal/indexer/indexer.go` L3589-3607).
- Never degrade to file-only the way `extractor_plugin.go` L424-434 does.
- Producer errors never carry source bytes, and wrapped messages should not add any.

### Identity and prefixing
**Source:** `cobol.go` L77-99 (`filePath` / `filePath + "::" + name`) and `indexer.go` L2255-2324 (`applyRepoPrefix`).
**Apply to:** the extractor (emits unprefixed IDs) and the tracer test (expects `res.RepoPrefix + "/" + relPath + "::" + symbol`).
- `unresolved::` targets are never prefixed (L2315-2318). Phase 2 emits none.

### Meta / provenance keys
**Source:** `internal/neo4jprojection/properties.go` L154-180 (key encoding + sensitive filter) and `internal/graph/node.go` L314-325 (`QualName`, `StartColumn` `omitempty`, `Meta`), L349-355 (`Origin` reserved for proxy nodes).
**Apply to:** the extractor, `source_revision.go`, and all provenance assertions.
- Use the `prov_` underscore prefix, with values of type string / bool / int / float64 / []string only. Convert `uint64` to `int`, since the SQLite flat codec has exact tags for those types.
- Assert the `prov_*` subset, not full Meta equality, because the indexer adds `search_*` keys.

### Test style
**Source:** `internal/parser/languages/cobol_test.go` and `cmd/gortex/*_test.go` use testify `require`/`assert`. `internal/neo4jprojection/properties_test.go` uses stdlib `t.Fatalf`.
**Apply to:** each new test. Match the style of the package it lives in.
- Invented COBOL only, inline raw-string fixtures, 7-space fixed format (as in `cobol_test.go` L13-22 and the fork `analyze_test.go` L20-33).
- Never pin `ToolID`, `Ledger.ConfigurationID`, or `gaps[].id`. Assert 64-hex shape instead.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `.planning/REQUIREMENTS.md` / `ROADMAP.md` / `PROJECT.md` amendments | docs | — | Text edits; use RESEARCH Q4 "Consequence for D-01" wording |
| README/docs COBOL-capability wording | docs | — | Text edit (D-15 temporary regression note) |
| Parser build-info lookup inside `cobol_grammar.go` | utility (inline) | transform | No `debug.ReadBuildInfo` use exists in Gortex source; copy the loop from `<fork>/preprocessor/pipeline.go` L489-501 / RESEARCH "Parser module identity" example, and keep it inline and unexported (single use) |

The END PROGRAM containment-stack algorithm (RESEARCH Q6) and the program-name lookup (RESEARCH Code Examples) have no in-repo analog either. They are grammar-specific; implement them from RESEARCH inside `cobol_grammar.go` as unexported helpers.

## Metadata

**Analog search scope:** `internal/parser/languages/`, `internal/indexer/`, `internal/gitstate/`, `internal/neo4jprojection/`, `internal/testutil/graphfixture/`, `internal/graph/{edge.go,node.go,store_sqlite/store.go}`, `internal/mcp/tools_core.go`, `cmd/gortex/` (call, daemon integration, dispatcher tests), `.github/workflows/*.yml`, `go.mod`, `.gitignore`. Fork: `preprocessor/{analyze.go,pipeline.go,analyze_test.go}`, `preprocessor/internal/tsadapter/{parser.go,parser_test.go,expected_identity.go}`, `preprocessor/{handoff,parsefacts,grade,internal/contract}`.
**Files scanned:** ~45
**Tracked-source gate:** all cited Gortex analogs pass `git ls-files` (53 checked); all cited fork analogs pass `git ls-files` in the fork (10 checked).
**Pattern extraction date:** 2026-09-30
