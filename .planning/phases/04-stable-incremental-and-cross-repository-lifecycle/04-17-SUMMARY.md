---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 17
subsystem: mcp
tags: [source-scope, nested-checkout, cr-01, security, multi-repo, rename, search_text, find_declaration]

requires:
  - phase: 04-10
    provides: requirePathSourceScope (lexical + physical-owner gate) in resolveFilePath and the absolute branch of resolveGraphPath
  - phase: 04-12
    provides: requireSourceScope, errSourceOutOfScope, withSourceScopeRequest, callSourceTool test helper
provides:
  - resolveNodePath admits its returned path through requirePathSourceScope (get_symbol_source, batch_symbols, get_editing_context, smart_context, withAbsPath)
  - resolveGraphPath relative branch admits its anchored path through requirePathSourceScope (rename_symbol reads/writes, scaffold)
  - filterTextMatchesByResolvedScope and findUseSiteMatches gate ResolveFilePath(m.Path) per path
  - requirePathSourceScope skips symlink resolution when !IsMultiRepo()
  - handleGetSymbolSource maps errSourceOutOfScope from resolveNodePath to "symbol not found: <id>"
  - newNestedCheckoutServer(t, trackBetaLater) fixture and TestSourceScope_NestedCheckoutRefused/{tracked_together,outer_indexed_first}/{7 surfaces}
affects: [phase-04-verification, 04-18, source-scope, rename_symbol, search_text, find_declaration, search_symbols content fallback]

actuals:
  tokens: 5000
  tasks: 2
  commits: 3
plan_head_before: 5c946e91af29a68f8cca1365b66b86e2ff67b7d8
plan_head_after: 090b2b2688b3ed03fdf5a01e0b0c92360b9be329

tech-stack:
  added: []
  patterns:
    - "Every source surface admits a path only through requirePathSourceScope: lexical repository first, then the physical owner (MultiIndexer.RepoForFile on the symlink-resolved location)"
    - "Text-match filters anchor a match with MultiIndexer.ResolveFilePath(m.Path) and gate it, verdict cached per path (two copies with // note:, below the rule of three)"

key-files:
  created: []
  modified:
    - internal/mcp/source_scope.go
    - internal/mcp/tools_fileops.go
    - internal/mcp/tools_coding.go
    - internal/mcp/tools_search_text.go
    - internal/mcp/tools_find_declaration.go
    - internal/mcp/source_scope_test.go
    - internal/mcp/tools_search_text_pathsep_test.go
    - internal/mcp/tools_search_text_test.go

key-decisions:
  - "Gate fix and 04-18 walk fix are independent: the gate is required on its own because an outer repository's pre-existing duplicate nodes for a nested checkout persist until its next reindex; outer_indexed_first tracks beta via TrackRepoCtx without a config entry so the duplicates exist even after 04-18."
  - "filterTextMatchesByResolvedScope's per-path gate reads the request from ctx (as handleSearchText's own requireSourceScope does), so direct callers must carry the request via withSourceScopeRequest; two direct-call unit tests were updated rather than adding a second ownership rule keyed on the ResolvedScope argument."
  - "requirePathSourceScope returns after the lexical gate when !IsMultiRepo(), so single-repo withAbsPath listings pay no EvalSymlinks per node."

patterns-established:
  - "No source surface decides ownership by lexical repository prefix alone"

requirements-completed: [SEC-01, SEC-02]

coverage:
  - id: D1
    description: "Alpha-scoped get_symbol_source / batch_symbols on a nested beta checkout return no beta bytes, in both indexing orders; get_symbol_source answers 'symbol not found'"
    requirement: SEC-02
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_NestedCheckoutRefused/*/get_symbol_source,batch_symbols"
        status: pass
    human_judgment: false
  - id: D2
    description: "Alpha-scoped rename_symbol of a beta-owned symbol is refused and beta's main.go is byte-identical afterwards; alpha's own dry-run rename succeeds"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_NestedCheckoutRefused/*/rename_symbol"
        status: pass
    human_judgment: false
  - id: D3
    description: "Alpha-scoped search_text and find_declaration (literal and regex) return no beta lines; beta-scoped search_text still finds beta/main.go"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_NestedCheckoutRefused/*/search_text,find_declaration"
        status: pass
    human_judgment: false
  - id: D4
    description: "No over-refusal: beta-scoped and alpha-own reads succeed; full internal/mcp package and TestCobolScope_ pass"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_NestedCheckoutRefused/*/in_scope_reads"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/mcp/ -timeout 40m"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolScope_'"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 17: Nested-checkout physical-owner source gate Summary

**Every source surface (resolveNodePath, both resolveGraphPath branches, search_text and find_declaration filters) now admits a path only through requirePathSourceScope, so an alpha-scoped request can neither read nor rename-write a nested beta checkout's bytes, even when alpha indexed beta's files before beta was tracked.**

## Performance

- **Duration:** about 45 min (two full-package runs of about 8.5 min each)
- **Tasks:** 2/2
- **Files modified:** 8

## Accomplishments

- `requirePathSourceScope` returns after the lexical gate when `!IsMultiRepo()`.
- `resolveNodePath` gates the path it returns (both the MultiIndexer branch and the absolute-FilePath branch). This covers get_symbol_source, batch_symbols, get_editing_context, smart_context, and withAbsPath.
- `handleGetSymbolSource` maps `errSourceOutOfScope` from `resolveNodePath` to `symbol not found: <id>`, so the refusal does not reveal which repository owns the file.
- The relative branch of `resolveGraphPath` gates the anchored, re-rooted path. rename_symbol already refuses the whole rename on a scope error, and scaffold inherits the gate.
- `filterTextMatchesByResolvedScope` and the `findUseSiteMatches` `allowed` closure anchor each match with `ResolveFilePath(m.Path)` and gate it. A path that doesn't anchor fails closed, and each path's verdict is cached.
- New fixture `newNestedCheckoutServer(t, trackBetaLater)`. `TestSourceScope_NestedCheckoutRefused` now runs 7 surfaces under `tracked_together` and under `outer_indexed_first`, which asserts the duplicate `alpha/nested/main.go::ZebraWidget` node exists.

## Task Commits

1. **Task 1 (tracer): get_symbol_source / batch_symbols nested gate.** Commit `d4796f32` (fix). RED confirmed before the change: both orderings returned beta's bytes.
2. **Task 2: rename writes, search_text, find_declaration.** Commit `5ed6d45e` (fix). RED confirmed before the change: search_text leaked `alpha/nested/main.go`, find_declaration returned nested use sites, and rename rewrote beta's main.go.
3. **Deviation fix: test contexts for direct filter calls.** Commit `090b2b26` (test).

## TDD Gate Compliance

For each task the tests were written first and run RED, and the implementation turned them GREEN. The test and implementation for each task were committed together in that task's commit, as the plan's per-task commit scope requires.

## Decisions Made

See frontmatter `key-decisions`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug, test fixtures] Direct `filterTextMatchesByResolvedScope` callers carried no request**
- **Found during:** plan-level verification (full `internal/mcp` run)
- **Issue:** `TestFilterTextMatchesByResolvedScope_BelowRepoRoot/prefixed_node_ids` and `TestFilterTextMatchesByResolvedScope_FailsClosed` call the filter with a bare `context.Background()`. The new per-path gate reads the request that the middleware stores in ctx, the same way `handleSearchText` already calls `requireSourceScope(ctx, "")`. Without a stored request, a multi-repo call looks unbound and every match is dropped.
- **Fix:** both tests now pass `withSourceScopeRequest(ctx, makeReq("search_text", {"repo": "beta"}))`, matching how production calls the filter (the same pattern `TestSourceScope_TextSearchToolsGated` uses for `findUseSiteMatches`). The assertions are unchanged.
- **Files modified:** `internal/mcp/tools_search_text_pathsep_test.go`, `internal/mcp/tools_search_text_test.go`. Neither file is listed in the plan's `files_modified`.
- **Commit:** `090b2b26`

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: behavior-change | internal/mcp/search_symbols_content_fallback.go | `searchSymbolsContentFallback` also calls `filterTextMatchesByResolvedScope`, so the search_symbols content-fallback match lines (source text) are now gated by the physical owner too. For a multi-repo request with no source scope, where the Locate default narrows but Reach resolves nothing, the fallback now drops its lines instead of returning them. That is fail-closed and consistent with D-13. No test covers it, and the file itself was not edited. |

## Verification

- Task 1 verify: all 8 sub-subtest PASS lines present, the regression run passed, gofmt clean.
- Task 2 verify: all 14 sub-subtest PASS lines present, the regression run (`TestSourceScope_|TestSearchText|TestFindDeclaration|TestRenameSymbol|TestIndexHealth`) passed, gofmt clean.
- `GOWORK=off go build ./... && GOWORK=off go vet ./internal/mcp/`: OK.
- `GOWORK=off go test -race -count=1 ./internal/mcp/ -timeout 40m`: ok (507s) after the deviation fix.
- `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolScope_'`: ok.
- Acceptance greps: tools_fileops non-comment `requirePathSourceScope(` = 5; search_text = 1; find_declaration = 1; `IsMultiRepo()` in source_scope.go = 1; `errSourceOutOfScope` in tools_coding.go = 4.

## Known Stubs

None.

## Deferred (unchanged, carried)

- WR-04 rename refusal text (human decision); concurrency warnings WR-01/WR-02; WR-03 regex-branch limit-before-filter false negative.
- Pre-existing gofmt drift in unrelated `internal/mcp` files (e.g. manifest.go, notebook.go) is out of scope.

## Self-Check: PASSED

- All 8 modified files exist; commits d4796f32, 5ed6d45e, 090b2b26 are ancestors of HEAD.
