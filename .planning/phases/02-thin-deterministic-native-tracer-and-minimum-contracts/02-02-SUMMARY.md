---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
plan: 02
subsystem: parser
tags: [cobol, tree-sitter, go-modules, provenance, sqlite, mcp, tracer]
requires:
  - phase: 02-01
    provides: "Pushed fork commit f9eaf99c34a92db6f5487d316cf9977295a62a7d with preprocessor.NewEmbeddedAnalyzer and NewContentID"
provides:
  - "go.mod pin: require .../preprocessor and replace go-sitter-forest/cobol => .../forest-shim/cobol, both at v0.0.0-20260930215433-f9eaf99c34a9"
  - "CobolGrammarExtractor (!windows) registered at register.go:119: file node + KindFunction program node (cobol_kind=program) + EdgeDefines, with the prov_* Meta contract"
  - "Windows stub CobolGrammarExtractor whose Extract returns errCobolParserUnavailableOnWindows"
  - "cmd/gortex/cobol_tracer_test.go: SQLite AddBatch recorder, reopen, MCP get_symbol, CLI get_symbol, search_symbols, AI-off acceptance"
affects: [02-03, 02-04, 02-05, 02-06, cobol-extractor, parser-provenance, neo4j-projection]
actuals:
  tokens: 8221
  tasks: 2
  commits: 2
plan_head_before: 94ebade0e4cbeb70fc96af4cd71fa681a2a76bbb
tech-stack:
  added:
    - "github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor v0.0.0-20260930215433-f9eaf99c34a9 (private, user-owned; D-02)"
    - "github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol v0.0.0-20260930215433-f9eaf99c34a9 (replace for go-sitter-forest/cobol)"
  patterns:
    - "Lazy sync.Once attestation with a cached init error; every Extract fails the same way (D-05)"
    - "Package-level capacity-1 channel serializes Analyze across extractor instances"
    - "Embedded-*store_sqlite.Store recorder overriding AddBatch/AddBatchChecked to prove the direct write path"
key-files:
  created:
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_windows.go
    - cmd/gortex/cobol_tracer_test.go
  modified:
    - go.mod
    - go.sum
    - internal/parser/languages/register.go
key-decisions:
  - "Pinned pseudo-version v0.0.0-20260930215433-f9eaf99c34a9 (fork commit f9eaf99c34a92db6f5487d316cf9977295a62a7d), confirmed with go list -m -json; preprocessor and forest-shim share it."
  - "prov_parser_module is read from build info as Replace.Path@Replace.Version, giving github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol@v0.0.0-20260930215433-f9eaf99c34a9 rather than the stock v1.9.1 the fork's ToolIdentity reports."
  - "The program name comes from the program_name observation whose parent is the identification_division whose parent is the program_definition; end_program names never match. Names are verbatim."
  - "The pre-existing go mod tidy drift that promotes neo4j-go-driver to a direct require was reverted, not committed, to keep the go.mod diff scoped; it is logged in deferred-items.md."
requirements-completed: [BASE-01, BASE-02, BASE-03, TRACE-01, TRACE-02, TRACE-03, TRACE-04, TRACE-05, TRACE-06, TRACE-07, PROV-01, PROV-02, PROV-03, PROV-05, PROV-06, ID-01]
coverage:
  - id: D1
    description: "GOWORK=off build resolves go-sitter-forest/cobol to the fork forest-shim at the preprocessor's pseudo-version; no go.work/go.work.sum; build succeeds; Windows closure excludes the fork preprocessor"
    requirement: BASE-01
    verification:
      - kind: other
        ref: "Task 1 verify command 1 (go list -m replace/version checks, register.go diff count, gofmt, go build, GOOS=windows go list -deps)"
        status: pass
    human_judgment: false
  - id: D2
    description: "One invented DEMOPGM program is persisted through AddBatch into SQLite with every prov_* key, survives close and reopen, and carries RepoPrefix/WorkspaceID/ProjectID equal to its file node's values"
    requirement: TRACE-04
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_PersistsThroughAddBatch (go test -race)"
        status: pass
    human_judgment: false
  - id: D3
    description: "MCP get_symbol detail=full over the real daemon relay returns the prefixed program node, its provenance, and a defines in-edge from the file node"
    requirement: TRACE-06
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_MCPGetSymbol (go test -race)"
        status: pass
    human_judgment: false
  - id: D4
    description: "gortex call get_symbol through the real relay (no stub) returns the same node with provenance"
    requirement: TRACE-05
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_CLIGetSymbol (go test -race)"
        status: pass
    human_judgment: false
  - id: D5
    description: "search_symbols DEMOPGM returns the program ID"
    requirement: TRACE-06
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_SearchSymbolsByName (go test -race)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The slice runs with no LLM service, an empty GORTEX_LLM_PROVIDER, no ask tool, and no Neo4j driver in the indexer/parser/SQLite store dependency closure"
    requirement: TRACE-07
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_NoAINoNeo4j (go test -race)"
        status: pass
      - kind: other
        ref: "Task 2 verify: go list -deps ./internal/indexer/ ./internal/parser/languages/ ./internal/graph/store_sqlite/ has no neo4j-go-driver"
        status: pass
    human_judgment: false
  - id: D7
    description: "Regex CobolExtractor and its tests are untouched (byte-identical to 3238e524) and the nine TestCobolExtractor_* tests still pass"
    requirement: TRACE-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_test.go#TestCobolExtractor_* (go test -race); git diff --quiet 3238e524 -- cobol.go cobol_test.go"
        status: pass
    human_judgment: false
  - id: D8
    description: "Windows stub compiles and fails every Extract with errCobolParserUnavailableOnWindows"
    requirement: BASE-02
    human_judgment: true
    rationale: "No Windows cgo toolchain on the development host, and internal/parser needs cgo, so the stub was not compiled locally. The first real compile is the Windows CI shard (Plan 02-05) and Plan 02-03's TestCobolGrammarWindowsStub."
duration: 23min
completed: 2026-09-30
status: complete
---

# Phase 2 Plan 02: Thin COBOL Grammar Tracer Summary

**One invented COBOL program now travels from the pinned enhanced grammar (fork `f9eaf99`, `v0.0.0-20260930215433-f9eaf99c34a9`) through the registered `CobolGrammarExtractor` and MultiIndexer repository prefixing, into SQLite via `AddBatch`. It survives a reopen with every `prov_*` key, and comes back out through MCP `get_symbol`, `gortex call get_symbol`, and `search_symbols`. No AI provider and no Neo4j are involved.**

## Pinned Pseudo-Version

| Field | Value |
|-------|-------|
| Fork commit | `f9eaf99c34a92db6f5487d316cf9977295a62a7d` |
| Pseudo-version (confirmed with `go list -m -json`) | `v0.0.0-20260930215433-f9eaf99c34a9` |
| `require` | `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor v0.0.0-20260930215433-f9eaf99c34a9` |
| `replace` | `github.com/alexaandru/go-sitter-forest/cobol => github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol v0.0.0-20260930215433-f9eaf99c34a9` |
| Persisted `prov_parser_module` | `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol@v0.0.0-20260930215433-f9eaf99c34a9` |
| Approved grammar ID (`cobolApprovedGrammarID`) | `f97452e11a2b80b92acb7edf776131470bc2e1ffd7c077f4ade4ce3c3a47503d` |

## Performance

- **Duration:** about 23 min
- **Started:** 2026-09-30T22:22Z
- **Completed:** 2026-09-30T22:45Z
- **Tasks:** 2/2
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments

- Pinned the private fork in go.mod: the `preprocessor` require plus a commented `forest-shim/cobol` replace, with the stock `cobol v1.9.1` require kept. go.sum pins h1 hashes for both fork modules.
- Deleted the stale git-ignored `go.work` and `go.work.sum` (D-20) and set the developer `GOPRIVATE` with `go env -w`. Global and repo git config were not touched: module fetches used per-command `GIT_CONFIG_*` environment variables.
- Added `CobolGrammarExtractor` in `cobol_grammar.go`, built for `!windows`:
  - `NewEmbeddedAnalyzer(transform.NewCatalog(nil))` runs once on the first Extract, and a failure is cached.
  - Parses are serialized through `cobolAnalyzeSlot`.
  - A handoff whose grammar ID differs from `cobolApprovedGrammarID` fails with `errCobolGrammarNotApproved`.
  - It emits the file node, one `KindFunction` program node (`cobol_kind=program`, `QualName`, native 1-based lines and 0-based columns), and an `EdgeDefines` edge with `ast_resolved`, confidence 1.0, and `EXTRACTED`.
  - Every node carries the D-08 `prov_*` keys, typed as string, bool, int, float64, or []string. No generated source, ledger entries, or snippets go into Meta.
- Added the Windows stub `cobol_grammar_windows.go`, which imports nothing from the fork.
- Swapped `register.go:119` to `NewCobolGrammarExtractor()`. That is a single-line change, and `cobol.go` and `cobol_test.go` are unchanged.
- Added five `TestCobolTracer_*` acceptance tests on a real SQLite store, git-initialized repo, MultiIndexer, and in-process daemon relay. All pass under `-race`.

## Task Commits

1. **Task 1 (tracer): pinned grammar to SQLite to MCP get_symbol**: `81d26400` (feat)
2. **Task 2: CLI get_symbol, search_symbols, AI-off/Neo4j-free**: `a6724dd3` (test)

**Plan metadata:** recorded in the docs commit that follows this SUMMARY.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar.go`: grammar extractor, approved-grammar pin, prov_* mapping, program-name lookup (231 lines).
- `internal/parser/languages/cobol_grammar_windows.go`: Windows stub and `errCobolParserUnavailableOnWindows`.
- `internal/parser/languages/register.go`: line 119 registers the grammar extractor.
- `cmd/gortex/cobol_tracer_test.go`: `cobolAddBatchRecorder`, `indexCobolTracerRepo`, `startCobolTracerDaemon`, `callCobolTracerTool`, `assertCobolTracerProgram`, and the five tracer tests.
- `go.mod` / `go.sum`: fork require and replace, plus fork sum lines. The stock `cobol v1.9.1` sum lines were dropped by tidy because the module is replaced.

## Decisions Made

- Verbatim program name from `program_definition` → `identification_division` → `program_name` (RESEARCH "Program-name lookup"), implemented as one bounds-checked pass over the observations.
- `prov_document_grade_reasons` is always a non-nil `[]string`, so the key is present even for a green document.
- Close/reopen: the shared index helper registers idempotent `mi.Close` and `store.Close` cleanups, so the persistence test can close early and reopen the same path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The Canonical row check used a 0x1f byte, but Canonical emits the literal text `\x1f`**
- **Found during:** Task 1 (GREEN run)
- **Issue:** The plan said the record starts with `node`, "the 0x1f separator", and the program ID. `graphfixture.Canonical` builds records with SQLite `printf('node\x1f%s…')`. SQLite's printf does not interpret `\x` escapes, so the separator is the four characters `\x1f`, and the byte-based check could never match.
- **Fix:** The test matches `"\nnode\\x1f" + programID + "\\x1f"`, with a comment explaining why.
- **Files modified:** `cmd/gortex/cobol_tracer_test.go`
- **Commit:** `81d26400`

**2. [Scope] go mod tidy drift left out of the go.mod diff**
- **Found during:** Task 1 (module pin)
- **Issue:** `go mod tidy` also promoted `neo4j-go-driver/v6` from indirect to direct. The same thing happens at `94ebade0` before any 02-02 change.
- **Fix:** The unrelated hunk was reverted so the diff holds only the planned require, the commented replace, and sum lines. It is logged in `deferred-items.md`.
- **Files modified:** `go.mod`
- **Commit:** `81d26400`

**3. [Engineering standard] A two-use get_symbol argument helper was inlined**
- **Found during:** Task 2 (self-review)
- **Issue:** A `cobolTracerGetSymbolArgs` helper had only two callers, which is below the rule-of-three threshold.
- **Fix:** Inlined it with a `// note:` comment. `callCobolTracerTool` (three callers) and `startCobolTracerDaemon` (four callers, required by the plan) stay as private helpers in the test file.
- **Commit:** `a6724dd3`

---

**Total deviations:** 3 (1 Rule 1 test fix, 1 scope-preserving revert, 1 standards cleanup). **Impact:** none on behavior. The planned contract and file set are unchanged.

## TDD Gate Compliance

The plan is `type: execute`, and the tracer task asked for RED first. `cobol_tracer_test.go` was written before the extractor, and its first run failed at the assertion level: the regex extractor produced `DEMOPGM` with no `prov_*` keys. That run also showed the harness itself worked, with the AddBatch recorder seeing the ID and the daemon relay finding the node. Per the plan, the test and the implementation landed in one tracer commit.

## Issues Encountered

- `detect_changes` compares the working tree, not a commit range, so it reported only the pre-existing uncommitted `.planning/config.json` and `.planning/state.json` edits. `git diff --name-only 94ebade0..HEAD` confirms exactly the six planned files.
- The Windows stub could not be compiled locally. There is no Windows cgo toolchain here, and `internal/parser` needs cgo. See coverage D8.

## Known Stubs

None. The Windows `CobolGrammarExtractor` is an intentional fail-closed stand-in (D-05, D-17), not a data stub.

## Threat Flags

None beyond the plan's threat model. The new private module dependency (T-02-SC) is pinned by pseudo-version and go.sum h1 hashes. No new endpoints or auth paths were added.

## Runtime Notes (not repo changes)

- A running `gortex` daemon built before this change still uses the regex extractor. Existing stores keep regex-era COBOL nodes until a full reindex. The user's daemon was not restarted.
- Every Go command in this checkout now needs `GOPRIVATE` covering the fork (set via `go env -w`), plus git access to the fork. Locally that is the `github-personal` SSH alias. CI is Plan 02-05.

## Next Phase Readiness

- Plan 02-03 can add containment-path and ordinal identity, the copybook skip, and `cobol_grammar_test.go` on top of `CobolGrammarExtractor`. The field `approvedGrammarID` is the injection point for the mismatch test.
- Plan 02-04 can key its VCS stamp on `prov_revision_content_id`, which every analyzed node already carries.
- Plan 02-05 needs the two `deferred-items.md` notes: the pre-existing tidy drift, and the BASE-02 step's missing stock go.sum lines.

## Self-Check: PASSED

- FOUND: internal/parser/languages/cobol_grammar.go, internal/parser/languages/cobol_grammar_windows.go, cmd/gortex/cobol_tracer_test.go
- FOUND: commits 81d26400 and a6724dd3 on planning/ai-enhanced-cobol-graph
- PASS: Task 1 verify 1 and 2, Task 2 verify, and every acceptance criterion (register.go:119 single line, build tags, 29 prov_ occurrences with no `prov.`, one NewEmbeddedAnalyzer, no NewCobolExtractor, go.mod comment above the replace, stock require kept, five tracer tests under -race, Windows closure without the fork preprocessor)
- PASS: adjacent packages under -race: internal/embedding, internal/mcp, internal/parser/forest/..., internal/cfg, internal/parser/languages, and cobolprobe with -enhanced-parser
