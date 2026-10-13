---
phase: 04-stable-incremental-and-cross-repository-lifecycle
reviewed: 2026-10-13T16:14:12Z
depth: standard
review_kind: incremental re-review of fix commits (diff base fec9f8e9)
files_reviewed: 15
files_reviewed_list:
  - internal/graph/store_sqlite/migration_unprefixed_purge.go
  - internal/graph/store_sqlite/migration_unprefixed_purge_test.go
  - internal/indexer/cobol_lifecycle_incremental_test.go
  - internal/indexer/cobol_lifecycle_material_test.go
  - internal/indexer/multi.go
  - internal/indexer/repository_untrack.go
  - internal/mcp/source_scope.go
  - internal/mcp/source_scope_test.go
  - internal/mcp/tools_fileops.go
  - internal/mcp/tools_find_declaration.go
  - internal/mcp/tools_find_declaration_test.go
  - internal/mcp/tools_search_text.go
  - internal/mcp/tools_search_text_test.go
  - internal/resolver/cobol_references.go
  - internal/resolver/resolver.go
findings:
  critical: 1
  warning: 4
  info: 7
  total: 12
status: issues_found
---

# Phase 4: Code Review Report (re-review after fixes)

**Reviewed:** 2026-10-13T16:14:12Z
**Depth:** standard (incremental: `git diff fec9f8e9..HEAD`, with call chains followed into `resolveNodePath`, `Indexer.shouldPruneDir`, `store_sqlite.ReindexEdges`, `reach.BeginTopologyMutation`, `Store.ResolveMutex`, and the daemon warmup batch end)
**Files Reviewed:** 15
**Status:** issues_found

## Narrative Findings (AI reviewer)

## Summary

This pass checked each fix commit against the finding it claims to close, then looked for new defects introduced by the fixes.

| Prior finding | Verdict |
|---|---|
| CR-01: symlink and nested-checkout scope bypass | **Partially closed.** The symlink case is closed on `resolveFilePath` and on the absolute branch of `resolveGraphPath`. The nested-checkout case is closed only for `read_file`, `edit_file`, and `write_file`. It is still open on every graph-backed source surface. I confirmed this with a probe test run through `go test -overlay`, which left the tree untouched. See CR-01 below. |
| CR-02: `search_text` and `find_declaration` scope gating | **Closed for unbound requests.** Both tools now refuse before searching. Bound narrowing is correct apart from the nested-checkout attribution (CR-01) and a truncation bug in the `find_declaration` regex branch (WR-03). |
| CR-03: untrack restub and reconcile | **Closed on the normal path.** The restub uses a best-effort write that can silently drop the batch, after which the purge still deletes the edges (WR-01). |
| WR-01: COBOL reconcile serialization, `ResetBatch`/`EndBatch` deferral | **The deferral is closed.** The new lock order holds: `r.mu` (store `ResolveMutex`), then `cobolReconcileMu`, then `mi.mu.RLock` and `repositoryMutationMu`. No caller takes `cobolReconcileMu` before any of those. I found no deadlock or lock-order inversion. The serialization half was already provided: every resolver on a store shares `store.ResolveMutex()` (`resolver.go:519-525`), so the new mutex is redundant for same-store passes (IN-07). The new deferred and reload call sites run outside the topology gate (WR-02). |
| WR-02: v22 sidecar sweep | **Closed.** The `RowsAffected` early return matches the narrowed node rule, and the regression test asserts that the sidecar row survives. |
| WR-03: reload re-decision | **Closed functionally.** The new reconcile runs outside the batch gate and the reach topology gate (WR-02). |
| WR-04: `rename_symbol` existence oracle | **Not fixed.** Carried forward as accepted risk (04-12). |
| IN-01..IN-06 | **Not addressed.** Carried forward; line numbers updated where code moved. |

## Critical Issues

### CR-01: Nested-checkout scope bypass is still open on graph-backed source reads, text search, and rename writes (incomplete fix of prior CR-01)

**File:** `internal/mcp/tools_fileops.go:509` (`resolveNodePath`), `internal/mcp/tools_fileops.go:596-598` (relative branch of `resolveGraphPath`), `internal/mcp/tools_search_text.go:240-275` (`filterTextMatchesByResolvedScope`), `internal/mcp/tools_find_declaration.go:150-162` (`allowed`). Root cause: `internal/indexer/indexer.go:5819-5835` (`shouldPruneDir`).

**Issue:** The fix added `requirePathSourceScope` only to `resolveFilePath` and to the absolute branch of `resolveGraphPath`. The outer repository's indexer still walks into a nested tracked root, because `shouldPruneDir` has no tracked-root check. Beta's files are therefore indexed a second time as **alpha-owned** nodes, for example `alpha/nested/main.go::ZebraWidget` with `RepoPrefix=alpha`. Every surface that gets its repository from the node or from the lexical match path then attributes beta's bytes to alpha:

- `resolveNodePath` gates only `node.RepoPrefix`. This path serves `get_symbol_source`, `batch_symbols`, `get_editing_context`, and `smart_context` bodies.
- The relative branch of `resolveGraphPath` gates only `matchedRepoPrefix(graphPath)`. This path serves `rename_symbol` reads and **writes**, and scaffold.
- `search_text` and `find_declaration` attribute trigram matches with `strings.Cut(m.Path, "/")` and `matchedRepoPrefix(m.Path)`.

I ran a probe with the same fixture as `TestSourceScope_NestedCheckoutRefused`: alpha at `<tmp>/alpha`, beta at `<tmp>/alpha/nested`. It was injected with `go test -overlay`, so the tree was not modified. Results:

```
NODE alpha/nested/main.go::ZebraWidget repo=alpha file=alpha/nested/main.go
NODE beta/main.go::ZebraWidget repo=beta file=beta/main.go
search_text {repo:alpha, query:Zebra}  -> err=false {"matches":[{"path":"alpha/nested/main.go","text":"func ZebraWidget() {}",...}]}
get_symbol_source {repo:alpha, id:alpha/nested/main.go::ZebraWidget} -> err=false {"source":"package main\n\nfunc ZebraWidget() {}",...}
```

So `read_file alpha/nested/main.go` with `repo:alpha` is refused, but `get_symbol_source` on the same file returns the same bytes. This is the D-13/D-20 bypass the prior CR-01 described, now reachable through other tools. The regression test passes only because it exercises `read_file` alone. The double ownership is also a SEC-08 gap: the ownership audit counts these rows as correctly prefixed alpha nodes, so `index_health` stays green while one file has two owners.

**Fix:** Close it at the root by not letting a per-repository walk descend into another tracked root. The MultiIndexer knows every root, so it can hand each per-repo Indexer the sibling roots to prune. This also removes the double ownership:
```go
// indexer.go shouldPruneDir, before the exclude rules:
if idx.isOtherTrackedRoot(path) { // set by MultiIndexer: absolute roots of every other tracked repo
    return true
}
```
If that is too invasive for fork hygiene, apply the physical-owner check at the remaining choke points instead:
```go
// resolveNodePath, after computing abs:
if err := s.requirePathSourceScope(ctx, node.RepoPrefix, abs); err != nil { return "", err }
// resolveGraphPath relative branch, after computing abs: same call with matchedRepoPrefix(graphPath)
// search_text / find_declaration: attribute a match with
//   s.multiIndexer.RepoForFile(s.multiIndexer.ResolveFilePath(m.Path))
// and require both the lexical and the physical repository to be in scope.
```
Add `get_symbol_source`, `search_text` (bound to alpha), and `rename_symbol` cases to `TestSourceScope_NestedCheckoutRefused`.

## Warnings

### WR-01: The untrack restub is best-effort, but the purge after it is destructive, so prior CR-03 can still recur

**File:** `internal/indexer/repository_untrack.go:209-221,232-275`; `internal/graph/store_sqlite/store.go:1918-1931`

**Issue:** `restubCobolEdgesInto` returns the cross-repository edges to their placeholders with `mi.graph.ReindexEdges(batch)`. On SQLite, `ReindexEdges` deliberately **swallows** `errReindexWriterGateContended` and drops the whole batch (store.go:1924-1927). The design assumes "a later resolve pass rebinds them". That holds when the targets survive. Here, `PurgeRepo(repoPrefix)` runs straight afterwards and deletes every edge whose `to_id` is in the purged repository. A dropped restub batch therefore leaves A's CALL/COPY edges pointing at B, and the purge deletes them. The deferred reconcile then has nothing to decide. This is the original CR-03 loss: the statement-occurrence edge is gone until A's file is re-extracted, and incremental no longer equals clean. The contention window is a checkpoint or a sibling writer holding `writeMu`, which is plausible during the purge's own heavy writes.

Two smaller gaps:
- The restub does not hold `store.ResolveMutex()`, so a reconcile running at the same time can read the bound edge before the restub and write the binding back afterwards. `RefreshRepoConfigs` is not excluded by the untrack's topology gate (see WR-02).
- When `PurgeRepo` fails, the restubbed edges keep their bound-decision meta on a placeholder target until a retry succeeds, because the reconcile runs only on `err == nil`.

**Fix:** Make the restub part of the destructive step, or make it fail closed:
```go
// Option A (preferred): inside PurgeRepo's transaction, before the edge delete,
//   UPDATE edges SET to_id = json_extract(meta, '$.cobol_placeholder'), cross_repo = 0
//   WHERE to_id IN (<B node ids>) AND repo_prefix_of(from_id) <> ? AND json_extract(meta,'$.cobol_placeholder') <> ''
// Option B: use a ReindexEdges variant that returns the contention error, and
// abort purgeRepositoryPayload (return the error so the saga retries) when the
// restub did not land. Hold mi.graph.ResolveMutex() across restub + PurgeRepo,
// releasing it before the deferred reconcileCobolRepos runs.
```

### WR-02: The new reconcile call sites run outside the reach topology gate and the batch gate

**File:** `internal/indexer/multi.go:2528-2532` (`RefreshRepoConfigs`), `internal/indexer/multi.go:1643-1648` (`ResetBatch`), `internal/indexer/multi.go:1608-1611` (`EndBatch`, before `runGlobalGraphPasses` takes the gate)

**Issue:** Every reconcile that existed before the fixes runs inside `reach.BeginTopologyMutation`: the watcher tails at `incremental_watcher_batch.go:241/296`, `ReconcileRepoCtx` through `beginRepositoryTopologyMutation`, and untrack. That gate is the only thing that calls `reach.InvalidateIndex()` (reach.go:276-301), and it keeps reach builders from seeing half-applied rewires. The three new call sites take neither the gate nor `batchMutationGate`:

- **Reload (`RefreshRepoConfigs`).** On a running daemon, a reload that changes `resolve_from` rebinds COBOL CALL/COPY edges and can `EvictFile`/`AddNode` D-11 alias resources. The precomputed `reach_d1..d3` tiers are never invalidated, so impact answers stay stale until an unrelated topology mutation. A reload during a parallel warmup (`deferResolve`) also decides immediately, before every repository is registered. The policy drops the unregistered repository, and bindings into it are rewritten to `missing`. That is the exact WR-01 registration-order bug, through a new door.
- **`ResetBatch`.** The warm fast path rebinds the edges of an interrupted reindex and never invalidates reach.
- **`EndBatch`.** Invalidation follows from `runGlobalGraphPasses`, but reach readers can still see the reconcile half-applied.

**Fix:** Run each of these under the same gates the watcher uses. Take `mi.batchMutationGate.RLock()` first, then the topology gate, which is the same order watchers use, so no inversion is introduced. Under `deferResolve`, record the repositories in `batchCobolRepos` instead of reconciling:
```go
// RefreshRepoConfigs, replacing the direct call:
if len(cobolChanged) > 0 {
    mi.batchMutationGate.RLock()
    defer mi.batchMutationGate.RUnlock()
    mi.mu.Lock()
    if mi.deferResolve {
        if mi.batchCobolRepos == nil { mi.batchCobolRepos = map[string]struct{}{} }
        maps.Copy(mi.batchCobolRepos, cobolChanged)
        mi.mu.Unlock()
        return len(roots)
    }
    mi.mu.Unlock()
    finish := reach.BeginTopologyMutation(mi.graph)
    mi.reconcileCobolRepos(cobolChanged)
    finish(true)
}
// ResetBatch / EndBatch (already hold batchMutationGate.Lock):
finish := reach.BeginTopologyMutation(mi.graph)
mi.reconcileCobolRepos(cobolRepos)
finish(true)
```

### WR-03: `find_declaration`'s regex branch applies `limit` before the scope filter, so a bound caller can get "no use sites matched" when matches exist

**File:** `internal/mcp/tools_find_declaration.go:164-184`

**Issue:** In the regex branch, `s.multiIndexer.GrepRegexp(useSite, pathPrefix, limit)` searches **every** repository and stops at `limit`, which defaults to 20. The new `allowed` filter runs only afterwards. A request bound to alpha, in a workspace where beta holds more than `limit` matches that sort first, gets zero or few use sites back. The tool then answers `"note": "no use sites matched"`. That is a silent wrong answer, and the fix report acknowledges the order ("the regex branch filters after"). The literal branch is correct because it scans with limit 0 and caps after filtering. `search_text` avoids the problem with `GrepRegexpForRepos(..., resolved.RepoAllow, limit)` plus `limitTextMatches`.

**Fix:** Narrow the search itself, as `search_text` does, or filter before capping:
```go
if s.multiIndexer != nil {
    matches, err = s.multiIndexer.GrepRegexp(useSite, pathPrefix, 0) // or GrepRegexpForRepos with the resolved allow-set
}
...
kept := matches[:0]
for _, m := range matches {
    if allowed(m) {
        kept = append(kept, m)
        if len(kept) >= limit { break }
    }
}
```
Add a test where the out-of-scope repository holds more than `limit` regex matches.

### WR-04: The `rename_symbol` refusal reveals that out-of-scope references exist. Accepted risk (04-12), not fixed

**File:** `internal/mcp/tools_coding.go:2864-2867`

**Issue:** Unchanged from the prior review. When any reference lies outside the resolved scope, `rename_symbol` returns a distinct message: "some references lie outside the resolved scope; widen it with repo, project, or workspace…". A caller scoped to `alpha` can run a `dry_run` rename of an alpha symbol and learn whether other repositories reference it. This is recorded as an accepted risk in plan 04-12's Threat Flags and was intentionally skipped by the fixer. It is carried forward only so the risk stays visible.

**Fix (if the acceptance is revisited):** Return the generic `errSourceScopeRequired` guidance, or the not-found text that `get_symbol_source` returns for an out-of-scope target.

## Info

### IN-01: `cobolPolicyLookup` logs a Warn for each repository on every reconcile pass

**File:** `internal/indexer/workspace_resolve.go:749-754`

**Issue:** Unchanged. The fixes add more reconcile entry points (`EndBatch`, `ResetBatch`, `RefreshRepoConfigs`, untrack), so a repository whose `resolve_from` names an untracked repository now repeats the Warn more often.

**Fix:** Log once per (repo, untracked-set) change with a `sync.Map` keyed by `repo+"\x00"+strings.Join(dropped, ",")`, or demote repeats to Debug.

### IN-02: Untrimmed `lib.Repo`/`lib.Path` in `cobolLibraryRank` disagree with the trimmed validation and tracking checks

**File:** `internal/resolver/cobol_references.go:420-424`

**Issue:** Unchanged; the lines moved. `" shared"` or `copy\prod` passes validation and then never ranks, so every COPY silently resolves to `missing`.

**Fix:** Normalize once in `cobolPolicyFromConfig`: `Repo: strings.TrimSpace(lib.Repo)`, `Path: path.Clean(filepath.ToSlash(strings.TrimSpace(lib.Path)))`.

### IN-03: The material fingerprint truncates on 32-bit builds and depends on list order

**File:** `internal/indexer/cobol_material.go:38-42,68`

**Issue:** Unchanged. `RefreshRepoConfigs` now compares fingerprints too (`multi.go:2520-2523`). Reordering `resolve_from` in a reload therefore also triggers a needless reconcile pass, on top of the restage.

**Fix:** Sort a clone of `ResolveFrom` before hashing. Keep the stored value as `int64` or a string.

### IN-04: The canonical dump omits edge and node fields that the reconcile pass mutates

**File:** `internal/testutil/graphdump/graphdump.go:101,115`

**Issue:** Unchanged. The new untrack restub sets `CrossRepo=false`, and the new equivalence tests (`TestCobolLifecycle_CrossRepoUntrack`, `TestCobolLifecycle_ParallelWarmRestart`, `TestCobolLifecycle_ConfigReloadRedecides`) cannot see a stale `CrossRepo` or `Origin`.

**Fix:** Add `Origin`, `CrossRepo`, and `RepoPrefix` to the rows under a new `graphdump-volatile/4` version.

### IN-05: Bare unprefixed `Indexer` paths no longer bind any COBOL CALL or COPY

**File:** `internal/indexer/indexer.go:598-603`, `internal/resolver/cobol_references.go` (`caller_unowned` branch of `decideCobol`)

**Issue:** Unchanged. `gortex eval-server --index` and the `init` temporary store still index through a bare `indexer.New`.

**Fix:** Route those paths through `indexEmbeddedRepo` / MultiIndexer, or document the behavior.

### IN-06: `get_editing_context` gates the anchor's repository but reads view bytes from the view's repository

**File:** `internal/mcp/tools_coding.go:571-590`

**Issue:** Unchanged.

**Fix:** In the view branch, also call `s.requireSourceScope(ctx, files.repoPrefix)` before `readViewFile`.

### IN-07: `cobolReconcileMu` duplicates the store-wide resolve mutex for same-store passes, and its regression test only proves the duplicate is taken

**File:** `internal/indexer/multi.go:146-152,394,981`; `internal/resolver/cobol_references.go:128-131`; `internal/indexer/cobol_lifecycle_incremental_test.go:479-515`

**Issue:** `resolver.New(g)` sets `r.mu = g.ResolveMutex()`. The SQLite store and the in-memory `Graph` each return one mutex per store (`store.go:391-396`, `graph.go:693-695`). The master resolver and every per-repository resolver over `mi.graph` therefore already serialize `ReconcileCobol` and `ResolveAll` end to end. The premise of the prior WR-01, that a new master resolver excludes nothing, was wrong for the shared store. The new mutex changes behavior in only one case: a per-repository pass running on an in-memory **shadow** store during a cold index. That pass now blocks reconciles on the disk store, which it never touches. The CR-03 fix also adds a new lock edge: `vectorUpdateMu` → `ResolveMutex` → `cobolReconcileMu`, because the deferred reconcile runs inside `SerializeVectorUpdate`. I found no inverse acquisition, so this is not a deadlock today. It is still coupling a future change could invert. `TestCobolLifecycle_ReconcileSerialized` holds the new mutex directly, so it would also pass with the mutex removed and the store mutex still serializing the passes. It also relies on a 200 ms negative wait.

**Fix:** Either remove `cobolReconcileMu` and rely on `ResolveMutex()`, or keep it and record in the field comment which cross-store case it exists for. Run the untrack reconcile after `SerializeVectorUpdate` returns rather than inside it.

---

_Reviewed: 2026-10-13T16:14:12Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
