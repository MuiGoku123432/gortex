---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 13
subsystem: cli
tags: [mcp, embedded, multi-indexer, ownership, watcher, sec-08, d-14]

requires:
  - phase: 04-05
    provides: strict ownership audit (graph.ReadPrefixDiagnostics, PrefixDiagnostics.Clean) and index_health repo_ownership status
  - phase: 04-12
    provides: fail-closed source gate that is a no-op with one tracked repository
provides:
  - "cmd/gortex/mcp.go: indexEmbeddedRepo(ctx, mi, path) (*indexer.IndexResult, string, error), which indexes the embedded tree through MultiIndexer.TrackRepoCtx and returns the assigned prefix"
  - "runMCP indexes --index through indexEmbeddedRepo and watches through mi.GetIndexer(prefix)"
  - "TestEmbeddedIndex_OwnedByRepo (cmd/gortex/mcp_index_test.go)"
affects: [04-15, 04-16, embedded-mcp, index_health]

actuals:
  tokens: 1911
  tasks: 1
  commits: 2
plan_head_before: 5c438110c98b84dcf5fa8edee489bb68c35f02fc

tech-stack:
  added: []
  patterns:
    - "Embedded single-repo mode is a workspace of one: index with MultiIndexer.TrackRepoCtx and read the prefix back with RepoForFile"
    - "The embedded watcher uses the prefixed per-repo Indexer. With no indexed repository, no watcher starts."

key-files:
  created: []
  modified:
    - cmd/gortex/mcp.go
    - cmd/gortex/mcp_index_test.go

key-decisions:
  - "indexEmbeddedRepo reads the prefix back with mi.RepoForFile(path) instead of result.RepoPrefix. This covers the already-tracked case, where TrackRepoCtx returns (nil, nil). An empty prefix is an error."
  - "The nil-MultiIndexer guard lives inside indexEmbeddedRepo. It returns an error that names the cause, so the guard has a direct test and nothing falls back to the bare ss.Indexer. The bare-Indexer contract-registry else-branch was removed because that indexer never indexes anything now."
  - "The background index uses context.Background(), not cmd.Context(). runMCP already treats cmd.Context() as possibly nil, and the old bare Index call was not cancellable either."
  - "The watcher is built directly with indexer.NewWatcher(mi.GetIndexer(prefix), ...) as the plan specifies. It is not a MultiWatcher. Smoke-tested: a file edit reindexed to a prefixed node."

requirements-completed: [SEC-08]

coverage:
  - id: D1
    description: "The embedded gortex mcp --index path writes only repository-owned nodes, and the strict audit is clean on its store"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "cmd/gortex/mcp_index_test.go#TestEmbeddedIndex_OwnedByRepo"
        status: pass
      - kind: other
        ref: "smoke: sandboxed `gortex mcp --index <tree>`; workspace(operation:index) reported repo_ownership healthy, owned_code_nodes 4, unowned 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "The embedded watcher reindexes through the prefixed per-repo Indexer, and does not start without an indexed repository"
    requirement: SEC-08
    verification:
      - kind: other
        ref: "smoke: `gortex mcp --index <tree> --watch`; after editing main.go, search(symbols, addedLater) returned tree/main.go::addedLater with repo_prefix tree. `gortex mcp --watch` from a refused cwd printed the watch-mode-needs-an-indexed-repository line."
        status: pass
    human_judgment: false
  - id: D3
    description: "A nil MultiIndexer is an error that names the cause, not an unprefixed fallback"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "cmd/gortex/mcp_index_test.go#TestEmbeddedIndex_OwnedByRepo/nil_MultiIndexer_is_an_error,_not_an_unprefixed_index"
        status: pass
    human_judgment: false
  - id: D4
    description: "The pre-existing embedded and MCP command tests and internal/serverstack still pass, with blast-radius counts recorded"
    verification:
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run 'Embedded|ResolveEmbeddedIndex|TrackedReposUnder|MCP|Mcp' && go test -race -count=1 ./internal/serverstack/"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -count=1 -timeout 20m ./cmd/gortex (full package)"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 13: Embedded MCP Index as a Workspace of One Summary

**`gortex mcp --index` now indexes through `MultiIndexer.TrackRepoCtx` and watches through `mi.GetIndexer(prefix)`, so the embedded server writes only repository-prefixed nodes and its strict ownership audit is clean.**

## Performance

- **Duration:** about 6 min from the recorded start. Reading context took several more minutes before that.
- **Started:** 2026-10-13T03:56:24Z
- **Completed:** 2026-10-13T04:03:00Z
- **Tasks:** 1 (TDD: RED + GREEN, no refactor needed)
- **Files modified:** 2

## Accomplishments
- `indexEmbeddedRepo` tracks the embedded tree with `MultiIndexer.TrackRepoCtx(ctx, config.RepoEntry{Path: path})` and reads the assigned prefix back with `mi.RepoForFile(path)`. Every node the embedded server writes carries that prefix.
- The embedded watcher is `indexer.NewWatcher(mi.GetIndexer(prefix), …)`, so incremental reindexes also write prefixed nodes. When no repository was indexed (no `--index`, or a refused launch cwd), the watcher does not start. One stderr line says watch mode needs an indexed repository. The old `"."` watch-path fallback is gone.
- A nil MultiIndexer returns an error that names the cause. The unprefixed bare `ss.Indexer` path is gone from runMCP.
- An end-to-end smoke test of the real binary, in a sandboxed HOME and XDG dirs with `mcp.allow_embedded: true` and no daemon, passed:
  - `search(symbols)` returned `tree/main.go::helperThing`.
  - `read(file)` still serves source with one tracked repository, so the 04-12 gate stays a no-op.
  - After a live edit, the watcher produced `tree/main.go::addedLater`.
  - `index_health` reported `repo_ownership: healthy, owned 4, unowned 0`.

## Blast Radius (research Open Question 7)

| Suite | Before | After |
|-------|--------|-------|
| `./cmd/gortex -run 'MCP\|Mcp\|Embedded'` | 17 PASS / 0 FAIL / 0 SKIP | 18 PASS / 0 FAIL (17 + `TestEmbeddedIndex_OwnedByRepo`) |
| verify filter `Embedded\|ResolveEmbeddedIndex\|TrackedReposUnder\|MCP\|Mcp` under `-race` | n/a | 19 PASS / 0 FAIL |
| `./internal/serverstack/` | 18 PASS / 0 FAIL / 0 SKIP | 18 PASS / 0 FAIL (`-race`) |
| full `./cmd/gortex` | not measured | `ok` (46s) |

The before runs were without `-race`, and the after runs used `-race` as the verify command requires. No test changed state.

## TDD Gate Compliance

- **RED** (`d567852c`): the test was committed with a no-op `indexEmbeddedRepo` (returning `nil, "", nil`) as compile scaffolding, following the 04-07 precedent. Without the stub, a missing function only fails to compile, which is not a valid RED. `GOWORK=off go test -count=1 -v ./cmd/gortex -run '^TestEmbeddedIndex_OwnedByRepo$'` exited 1. `TestEmbeddedIndex_OwnedByRepo` failed at `mcp_index_test.go:207` on `require.NotNil(t, result)`, the planned assertion: nothing was indexed under a prefix.
- **GREEN** (`7bc57576`): the target test and its nil-MultiIndexer subtest pass, and the full verify command exited 0.
- **REFACTOR:** none needed.

## Task Commits

1. **Task 1 RED: failing test for embedded index repository ownership**: `d567852c` (test)
2. **Task 1 GREEN: index the embedded tree as a workspace of one**: `7bc57576` (feat)

## Files Created/Modified
- `cmd/gortex/mcp.go`:
  - Adds `indexEmbeddedRepo`.
  - The background goroutine indexes through it and builds the watcher on `mi.GetIndexer(prefix)`.
  - The watcher refuses to start when no repository was indexed.
  - The unused `idx := ss.Indexer` and the bare-Indexer contract-registry branch are removed.
- `cmd/gortex/mcp_index_test.go`: adds `TestEmbeddedIndex_OwnedByRepo`. It uses a temp Go file plus an invented COBOL program `EMBPGM`, a SQLite store, a real MultiIndexer, the strict `ReadPrefixDiagnostics` / `Clean` audit, a prefixed program-node lookup, and a nil-MultiIndexer subtest.

## Decisions Made
See `key-decisions` in the frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] RED needed compile scaffolding**
- **Found during:** Task 1 (RED)
- **Issue:** A test that calls a function that does not exist yet only fails to compile, and that is INVALID_RED.
- **Fix:** The RED commit carries a no-op `indexEmbeddedRepo` stub, so the test fails on its planned assertion. GREEN replaced the stub.
- **Files modified:** cmd/gortex/mcp.go
- **Committed in:** d567852c

---

**Total deviations:** 1 auto-fixed (1 blocking).
**Impact on plan:** None on scope. Only the 2 planned source files changed.

## Issues Encountered

These behavior changes follow from routing through `TrackRepoCtx`. They are intended and listed for the verifier:
- **Unsafe roots:** an explicit `--index` that names an unsafe root, such as the home directory or `/`, is now refused by `TrackRepoCtx`'s unsafe-root guard ("pass force to track it anyway"). Before, the bare Indexer crawled it. Inferred roots were already refused by `resolveEmbeddedIndex`.
- **Index config source:** the per-repository indexer takes its index config from the repository's `.gortex.yaml` through the ConfigManager, the same way the daemon does. It no longer uses the `--config` file's `index:` block, which the bare `ss.Indexer` used.
- **Watcher wiring:** the embedded watcher is a plain `Watcher` on the per-repo Indexer, as the plan specifies. It does not get the MultiWatcher-only hooks (repository mutation coordinator, `IncrementalReindexRepo` batch path). Those are unexported in `internal/indexer`. Incremental reindex through the prefixed indexer was smoke-verified.

## Threat Flags

None. T-04-20 is mitigated as planned: indexing and watching both go through the prefixed MultiIndexer path, no unprefixed watcher starts, a nil MultiIndexer is an error, and `TestEmbeddedIndex_OwnedByRepo` pins it.

## Known Stubs

None.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness
- The embedded path was the last writer of unowned nodes (SEC-08). Daemon, MultiIndexer and embedded stores now all write repository-owned nodes only.
- Ready for the remaining Phase 04 plans (04-08, 04-09, 04-10, 04-15, 04-16).

## Self-Check: PASSED
- FOUND: cmd/gortex/mcp.go, cmd/gortex/mcp_index_test.go
- FOUND commits: d567852c, 7bc57576
- Key links: `TrackRepoCtx(` in mcp.go (1), `ReadPrefixDiagnostics(` in mcp_index_test.go (1)
- Acceptance: `indexEmbeddedRepo(` prints the definition (L567) and the runMCP call (L495). `idx.Index(mcpIndex)` prints nothing.
