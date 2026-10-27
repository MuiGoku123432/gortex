---
phase: 04-stable-incremental-and-cross-repository-lifecycle
fixed_at: 2026-10-13T10:45:00Z
review_path: .planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 6
skipped: 1
status: partial
---

# Phase 4: Code Review Fix Report

**Fixed at:** 2026-10-13T10:45:00Z
**Source review:** .planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (CR-01..CR-03, WR-01..WR-04; Info items out of scope)
- Fixed: 6
- Skipped: 1 (WR-04, accepted risk)

Every fix has a regression test that failed before the fix and passes after it. The pre-fix failure for each is quoted below.

## Fixed Issues

### CR-01: The source gate assigns a repository from the lexical path, so a symlink or nested checkout bypasses the scope

**Files modified:** `internal/mcp/source_scope.go`, `internal/mcp/tools_fileops.go`, `internal/mcp/source_scope_test.go`
**Commit:** 261ee576
**Applied fix:** Added `requirePathSourceScope(ctx, lexical, absPath)` next to `requireSourceScope`. It first gates the lexical repository. It then resolves the path with `filepath.EvalSymlinks`, falling back to `resolveNearestExistingAncestor`, and gates the repository that physically owns that location. The owner is `MultiIndexer.RepoForFile`, which picks the longest tracked root and also handles canonical roots. If the request's own view checkout contains the location, the view's repository is the owner instead. `resolveFilePath` now calls this helper. So does the absolute-path branch of `resolveGraphPath`; its relative branch is unchanged.
**Regression tests:** `TestSourceScope_SymlinkIntoOtherRepoRefused` and `TestSourceScope_NestedCheckoutRefused`. The first covers read_file (repo-prefixed and absolute paths), edit_file and write_file, which must leave beta's bytes unchanged, and the absolute branch of resolveGraphPath. The second covers a nested checkout.
**Pre-fix evidence:** `read_file alpha/vendor/main.go through a symlink into beta must be refused: {"content":"package main\n\nfunc ZebraWidget() {}\n",...}`. The nested case returned `ZebraWidget` for `alpha/nested/main.go`.

### CR-02: `search_text` and `find_declaration` return source lines from every tracked repository with no scope

**Files modified:** `internal/mcp/tools_search_text.go`, `internal/mcp/tools_find_declaration.go`, `internal/mcp/source_scope_test.go`, `internal/mcp/tools_search_text_test.go`, `internal/mcp/tools_find_declaration_test.go`
**Commit:** bfd543d9
**Applied fix:** Both tools now refuse an unbound multi-repo request with `errSourceScopeRequired` before any search runs. The refusal does not depend on whether anything matched, so it cannot be used to learn whether a string exists. For bound requests, `search_text` keeps the narrowing it already had through `filterTextMatchesByResolvedScope`. `findUseSiteMatches` now takes `ctx` and drops any use site whose repository fails `requireSourceScope`, caching the result per repository. The literal branch filters before applying the limit; the regex branch filters after.

Two existing tests asserted the old unbound fan-out: `TestSearchText_MultiRepoFanout` and `TestFindDeclaration_MultiRepoFanout`. Under D-13 that behaviour is not allowed, so both now pass `repo` through the production middleware.
**Regression test:** `TestSourceScope_TextSearchToolsGated`. Unbound requests are refused for a literal match, a literal no-match, regex `.`, and find_declaration. Bound requests for search_text and both arms of find_declaration return only alpha's lines.
**Pre-fix evidence:** `search_text map[format:json query:func ] must refuse an unbound multi-repo request, got: {"count":2,"matches":[{"path":"alpha/main.go",...},{"path":"beta/main.go",...`

### CR-03: Untracking a repository deletes other repositories' bound COBOL CALL/COPY edges

**Files modified:** `internal/indexer/repository_untrack.go`, `internal/indexer/cobol_lifecycle_incremental_test.go`
**Commit:** f8dfaf61
**Applied fix:** Added `restubCobolEdgesInto(repoPrefix)`, which runs in `purgeRepositoryPayload` before the purge. It collects the repository's COBOL program and copybook nodes with `GetRepoNodesByLanguage`. Every in-edge from another repository that carries `cobol_placeholder` goes back to that placeholder, with `CrossRepo` set to false, and the whole batch is written with one `ReindexEdges` call. This matches what `restubIncomingRefs` already does on file eviction. Once the purge succeeds, a deferred `reconcileCobolRepos` covers the untracked prefix and each restubbed caller repository. The untracked prefix expands to every repository whose `resolve_from` names it, because its mutation lane is still attached during the purge.
**Regression test:** `TestCobolLifecycle_CrossRepoUntrack` (D-04 sequence). It tracks crossa and crossb with crossa allowed to resolve from crossb, and checks that XPGM and XCPY bind into crossb. It then untracks crossb and compares with a clean build of crossa alone: XPGM and XCPY must be `missing`, and SUBPGM must bind back to crossa's own program. Finally it re-tracks crossb and checks that the edges rebind.
**Pre-fix evidence:** the dump digest differed from the clean build. The XPGM and XCPY edges were missing entirely, and SUBPGM stayed `ambiguous` with a stale `crossb/cobol/sub.cbl::SUBPGM` candidate.

### WR-01: COBOL reconcile passes are not serialised, and the warm-restart pass ignores `deferResolve`

**Files modified:** `internal/indexer/multi.go`, `internal/resolver/cobol_references.go`, `internal/resolver/resolver.go`, `internal/indexer/cobol_lifecycle_incremental_test.go`
**Commit:** d3965000
**Status:** fixed: requires human verification (concurrency and ordering logic)
**Applied fix:**
- **Serialisation.** Added `MultiIndexer.cobolReconcileMu`, following the existing `reconcileMu` pattern. A new `Resolver.SetCobolReconcileLock(*sync.Mutex)` installs it on the master resolver (`newMasterResolver`) and on every per-repository resolver, and `reconcileCobol` holds it for the whole pass. This covers the per-repo `ResolveAll` path the review called out, not only `reconcileCobolRepos`. The lock is taken in one place only, inside the resolver, so the lock order is always `r.mu` then `cobolReconcileMu` and no re-entry is possible.
- **Deferred warm restart.** Under `batchMode.deferResolve`, `ReconcileRepoCtx` now records the prefix in `mi.batchCobolRepos` instead of reconciling at once. `EndBatch` and `ResetBatch` take that set under `mi.mu` and run one reconcile over it after every repository is registered. `ResetBatch` is included because the warm-restart fast path (nothing changed) is exactly the case where an interrupted reindex left placeholders behind (ID-07).

The restart wiring in `TestCobolLifecycle_Interrupted` was extracted into `env.restart` / `env.warmReconcile`, because the new test is its third use (rule of three).
**Regression tests:**
- `TestCobolLifecycle_ParallelWarmRestart` warm-restarts under `BeginParallelBatch` and reconciles crossa before crossb is registered. Crossa's binding into crossb must not be rewritten, and the batch end must leave the clean build.
- `TestCobolLifecycle_ReconcileSerialized` holds `cobolReconcileMu` and checks that both a master pass and a per-repository pass block until it is released.
**Pre-fix evidence:** `expected: "crossb/cobol/xpgm.cbl::XPGM" actual: "unresolved::cobol::program::XPGM" — decided before crossb was registered`. `TestCobolLifecycle_ReconcileSerialized` did not compile before the fix because the lock it holds did not exist. I first tried an in-memory `-race` concurrency test, but the graph's shard locks hide the race from the detector, so it passed pre-fix and I replaced it with the blocking test.

### WR-02: The v22 sidecar sweep deletes every empty-prefix sidecar row on every v21 store

**Files modified:** `internal/graph/store_sqlite/migration_unprefixed_purge.go`, `internal/graph/store_sqlite/migration_unprefixed_purge_test.go`
**Commit:** 62a99511
**Applied fix:** `purgeUnprefixedRepoRows` now checks `RowsAffected` on the node delete and returns before the sidecar sweep if it removed no unowned file-backed node. A store with real unprefixed residue is still swept exactly as before; v7 and the existing v22 tests are unchanged and still pass.
**Regression test:** `TestMigrationV22_KeepsSidecarsWhenNoNodePurged`. A v21 store holds only an empty-path `dep::` global plus an empty-prefix `file_mtimes` row, and the row must survive the open.
**Pre-fix evidence:** `empty-prefix sidecar rows: got 0 ([]), want 1 ([:shared/notes.md])`

### WR-03: `daemon reload` changes the live COBOL policy, but nothing reconciles existing bindings

**Files modified:** `internal/indexer/multi.go`, `internal/indexer/cobol_lifecycle_material_test.go`
**Commit:** 788690e9
**Applied fix:** `RefreshRepoConfigs` now computes `idx.cobolMaterial()` before and after `SetCobolConfig`, and collects each repository whose fingerprint changed. After the loop it runs one `reconcileCobolRepos` over that set. That reconcile re-decides CALL/COPY bindings (allowlist, library order) and the D-11 default-qualifier alias edges (`reconcileCobolResources`). The stored `_cobol_material` still triggers its one-time restage on the next full-root reindex, as before.
**Regression test:** `TestCobolLifecycle_ConfigReloadRedecides`. It writes `.gortex.yaml` straight to disk and calls only `RefreshRepoConfigs`, with no reindex. Crossa's XPGM and XCPY must bind into crossb, then return to `missing` once `resolve_from` is removed.
**Pre-fix evidence:** `expected: "crossb/cobol/xpgm.cbl::XPGM" actual: "unresolved::cobol::program::XPGM"` (XCPY the same)

## Skipped Issues

### WR-04: The `rename_symbol` refusal reveals that out-of-scope references exist

**File:** `internal/mcp/tools_coding.go:2860-2866`
**Reason:** accepted risk (04-12). It is recorded in plan 04-12's SUMMARY Threat Flags; the orchestrator said not to change it.
**Original issue:** `rename_symbol` gives a distinct refusal message when any reference lies outside the resolved scope. A caller can use that message to learn whether other repositories reference a symbol.

## Verification

Verification ran in the isolated review-fix worktree, `.claude/worktrees/rf-04-3086-1791902963` (branch `gsd-reviewfix/04-3086`), which the cleanup tail then removed. That worktree was not checked again from the main checkout after the fast-forward, but the fast-forward brings the main checkout to the same commits.

| Command | Result |
|---|---|
| `GOWORK=off go build ./...` | ok |
| `GOWORK=off go vet ./internal/mcp ./internal/indexer ./internal/resolver ./internal/graph/store_sqlite` | ok |
| `GOWORK=off go test -race -count=1 ./internal/mcp -timeout 40m` | ok (531.5s) |
| `GOWORK=off go test -race -count=1 ./internal/resolver -timeout 40m` | ok (35.4s) |
| `GOWORK=off go test -race -count=1 ./internal/indexer -timeout 60m` (full package) | ok (1029.5s) |
| `GOWORK=off go test -race -count=1 ./internal/graph/store_sqlite -timeout 60m` (full package) | ok (685.6s) |
| `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolLifecycle_' -timeout 40m` | ok (128.4s) |
| `GOWORK=off go test -race -count=1 ./internal/mcp -run 'TestSourceScope_'` | ok (10.8s) |
| `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol' -timeout 40m` | ok (37.4s) |
| `GOWORK=off go test -count=1 ./cmd/gortex -timeout 40m` (full package, no race) | ok (50.4s) |

`gofmt -l` is clean on every file these commits touched. It does list some other files in `internal/mcp` and `internal/resolver`, but those were unformatted before this run.

---

_Fixed: 2026-10-13T10:45:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
