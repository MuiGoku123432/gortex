---
phase: 04-stable-incremental-and-cross-repository-lifecycle
reviewed: 2026-10-13T19:23:03Z
depth: standard
review_kind: incremental re-review of gap-closure plans 04-17 and 04-18 (diff base d1d1a688)
files_reviewed: 14
files_reviewed_list:
  - internal/indexer/indexer.go
  - internal/indexer/multi.go
  - internal/indexer/nested_tracked_root_test.go
  - internal/indexer/walk_source.go
  - internal/indexer/watcher.go
  - internal/indexer/workspace_resolve.go
  - internal/mcp/source_scope.go
  - internal/mcp/source_scope_test.go
  - internal/mcp/tools_coding.go
  - internal/mcp/tools_fileops.go
  - internal/mcp/tools_find_declaration.go
  - internal/mcp/tools_search_text.go
  - internal/mcp/tools_search_text_test.go
  - internal/mcp/tools_search_text_pathsep_test.go
findings:
  critical: 1
  warning: 5
  info: 6
  total: 12
status: issues_found
---

# Phase 4: Code Review Report (re-review of 04-17 / 04-18)

**Reviewed:** 2026-10-13T19:23:03Z
**Depth:** standard (incremental: `git diff d1d1a688..HEAD`, with call chains followed into `GlobalConfig.ReposSnapshot`, `ConfigManager.ActiveRepos`, `MultiIndexer.indexMultiRepo`, `UntrackRepo`, `RepoForFile`, `LinkedWorktreeRoots`, `worktreeRootedPath`, and the explore text lanes)
**Files Reviewed:** 14
**Status:** issues_found

## Narrative Findings (AI reviewer)

## Summary

Plan 04-17 adds a physical-owner check (`requirePathSourceScope`) to `resolveNodePath`, the relative branch of `resolveGraphPath`, `search_text`/`search_symbols` content fallback (`filterTextMatchesByResolvedScope`), and `find_declaration` (`findUseSiteMatches`). Plan 04-18 stops an outer repository's walk, admission probe, and watcher at a nested tracked root.

**What holds:**

- **Read gating.** The prior CR-01 read bypass is closed for every graph-backed reader that goes through `resolveNodePath` or `resolveGraphPath`: `get_symbol_source`, `batch_symbols`, `smart_context`, `get_editing_context`, nav, cfg, dataflow, manifest, and `rename_symbol`/scaffold. `search_text` and `find_declaration` are closed as well. All four new tests pass.
- **Refusal text.** Refusals do not reveal whether an out-of-scope file exists. `errSourceOutOfScope` is the same generic not-found text whether the lexical gate or the physical-owner gate refuses, and `get_symbol_source` maps it to `symbol not found`.
- **Lock order.** The new `mi.mu.RLock` in `nestedTrackedRootContains` is safe. I checked every `mi.mu.Lock` and `mi.mu.RLock` critical section in `internal/indexer`. None of them walks or calls `shouldExclude`, `PathIndexability`, or `Watcher.isExcluded`. The cold-index collector no longer holds `mi.mu` across the result channel (`multi.go:2297-2314`). `MultiWatcher.EnqueueFileMutation` releases `mw.mu` before it calls into the per-repo watcher. Nothing enforces this, though (IN-05).

**What does not hold:** the walk-pruning half (04-18) takes its roots from the wrong source, and its state goes stale. I wrote throwaway probe tests and ran them with `go test -overlay`, so the tree was not modified. They show that:

1. A nested repository defined under `projects:` is still indexed twice.
2. An inactive config entry makes the nested files disappear from the graph entirely.
3. After `UntrackRepo("beta")`, alpha keeps excluding beta's former files.

The tests only exercise the config shape where the seed happens to be correct.

### Status of prior findings (base fec9f8e9 review)

| Prior finding | Status after 04-17/04-18 |
|---|---|
| CR-01: nested-checkout bypass on graph-backed reads, text search, rename writes | **Closed for the source-byte gate.** Root-cause pruning is only partial (see CR-01, WR-01..WR-03 below). One residual source surface is outside the changed files (WR-04). |
| WR-01: untrack restub is best-effort before a destructive purge | **Open.** `repository_untrack.go` was not touched. |
| WR-02: new COBOL reconcile call sites run outside the reach topology gate and batch gate | **Open.** Not touched. |
| WR-03: `find_declaration` regex branch caps before the scope filter | **Open, and worse.** Re-raised as WR-05. |
| WR-04: `rename_symbol` existence oracle (accepted risk, 04-12) | **Open (accepted).** It now also reveals whether a *nested* physical owner holds a reference. |
| IN-01..IN-07 | **Open.** None of the touched code addresses them (IN-06 `get_editing_context` view branch still gates only `anchor.RepoPrefix`, `tools_coding.go:571-576`). |

## Critical Issues

### CR-01: The nested-root walk seed comes from the wrong repository list: project-defined repos are still double-indexed, and inactive entries make files disappear

**File:** `internal/indexer/multi.go:376-392` (seed), consumed by `internal/indexer/indexer.go:5697-5710` (`inNestedTrackedRoot`)

**Issue:** The seed is built from `mi.configMgr.Global().ReposSnapshot()`, which copies only the top-level `gc.Repos` (`config/global.go:456-463`). The set a warmup actually indexes is `ConfigManager.ActiveRepos()` (`config/manager.go:447-459`), optionally narrowed by `filterReposByScope`. During a parallel warmup the live fallback (`nestedTrackedRootContains`) sees nothing, because `mi.repos` is published only after every worker has drained (`multi.go:2314-2322`). The seed is therefore the **only** guard on the cold path, and the two lists disagree in both directions:

- **Under-prune (SEC-08 still open).** When `active_project` names a `projects:` entry, `ActiveRepos` returns `gc.Projects[p].Repos`, which are not in `gc.Repos`. Alpha's walk then descends into beta. Probe result: `repos=[alpha beta]`, `alpha/nested/main.go::ZebraWidget present: true`, so one file has two owners. This is the exact CR-01/SEC-08 state the plan claims to close.
- **Over-prune (graph data loss).** `gc.Repos` lists entries that are not being indexed: another project's entries when `active_project` is set, entries filtered out by `IndexScoped(workspace, project)`, and entries that fail to open or index (`resolveErrors`). Alpha still prunes their roots, so nobody indexes those files. Probe result: `active_project: p1`, beta tagged `p2`, so `repos=[alpha]`, `alpha/nested node present: false`, `beta node present: false`. Meanwhile `RepoForFile` attributes `alpha/nested/main.go` to alpha, so `read_file` serves it while every graph tool says it does not exist.

**Fix:** Seed from the list the batch is actually indexing, not from the raw config. Do it once, before the workers start, and pass it in:
```go
// indexMultiRepo, after `resolved` is built (absPath already filepath.Abs'd and Stat'd):
roots := make([]string, 0, len(resolved))
for _, r := range resolved { roots = append(roots, filepath.Clean(r.absPath)) }
// newPerRepoIndexerGuardedWithMode(cfg, mode, roots) -> idx.trackedRepoRoots = roots
```
Seed `TrackRepoCtx`/`IndexRepo` constructors from the registered `mi.repos` roots, taken before construction under `mi.mu.RLock`. Drop `ReposSnapshot()` from the constructor. Add tests for a `projects:`-map config, an `active_project` that excludes the nested entry, and a `filterReposByScope` run that excludes it.

## Warnings

### WR-01: The seed never refreshes, so after untracking the nested repository the outer repository keeps excluding its files

**File:** `internal/indexer/indexer.go:529-533,5701-5705`; `internal/indexer/watcher.go:2953`

**Issue:** `trackedRepoRoots` is documented as "read-only after construction". `UntrackRepo` deletes beta from `mi.repos` and from the config (`multi.go:3449`), but alpha's long-lived `Indexer` keeps its seed, and so does the `Watcher` that points at that Indexer. Probe result after `mi.UntrackRepo("beta")`: `repos=[alpha]`, same alpha Indexer, `shouldExclude(betaDir/main.go) == true`.

As a result:
- Alpha's watcher refuses every create or edit under `alpha/nested/`.
- Nothing re-adopts the files. The untrack path does not reindex the enclosing repository.
- The files are absent from the graph until alpha's Indexer is rebuilt by a full `IndexRepo` or a daemon restart. `RepoForFile` still says alpha owns them, which is the same inconsistency as CR-01's over-prune.

The same staleness applies to a config reload that removes an entry.

**Fix:** Drop the per-Indexer copy and keep the live `mi.repos` check as the only source after warmup. Have warmup pass its batch roots (CR-01) in a field that `indexMultiRepo` clears once it publishes `mi.repos`. Then, on untrack of a root nested in another tracked root, schedule a scoped reindex of the enclosing repository for that subtree, or at least of the files under it. Add a regression test: track both, untrack beta, then assert that alpha's watcher admits `betaDir/main.go` and that alpha's next incremental pass indexes it.

### WR-02: Tracking a nested repository later freezes the outer repository's stale duplicates, because the watcher now refuses their update and delete events

**File:** `internal/indexer/watcher.go:2945-2955`; `internal/indexer/nested_tracked_root_test.go:127-136`

**Issue:** When beta is tracked after alpha was indexed, `alpha/nested/*` nodes remain in the graph (`trackNestedLater` asserts this precondition). After 04-18, alpha's watcher refuses every event under beta's root, including deletes. Those alpha-owned duplicates can therefore never be updated or evicted until alpha's next full reindex or warm-restart census. In the meantime:
- `search_symbols`, `find_usages` and the other graph-metadata tools, which are not source-gated, keep returning `alpha/nested/...` symbols that drift from disk.
- The ownership audit counts them as correctly prefixed alpha nodes, so `index_health` stays green while one file has two owners.

`TestNestedTrackedRoot_OuterReindexDropsDuplicates` passes only because it calls `mi.IndexRepo("alpha")` by hand. No production track path does that.

**Fix:** In `TrackRepoCtx`, after beta is registered, evict the enclosing repository's file nodes under beta's root. Every outer repository whose root strictly contains beta's root needs this, for example `EvictFile` for each `GetFileNodes` path with prefix `<outer>/<rel(beta)>/`. Do it under the same topology gate the track already holds. Assert this in `trackNestedLater` without the manual `IndexRepo`.

### WR-03: Nested-root containment is lexical only, so a symlinked or alternate spelling of the nested root is not pruned

**File:** `internal/indexer/indexer.go:5701-5705`; `internal/indexer/workspace_resolve.go:152-168`; `internal/indexer/multi.go:386-390`

**Issue:** Both checks use `pathkey.HasPathPrefix` on `filepath.Abs` spellings, which folds case only. If beta is tracked through a symlink to `alpha/nested`, or through any alternate spelling (`/var` vs `/private/var`, a bind mount), the check does not match. Probe result: alpha at `<tmp>/alpha`, beta tracked as `<tmp2>/betalink -> <tmp>/alpha/nested`, and `alpha/nested/main.go::ZebraWidget` is present. Elsewhere the codebase already treats roots by canonical identity: `pathkey.SamePathIdentity` in duplicate-root detection (`multi.go:2187`), and `CanonicalHasPathPrefix`/`CanonicalPath` in `bestRepoPrefixForPathLocked`.

**Fix:** Canonicalize the roots once when seeding or registering them, and compare with the canonical form:
```go
// seed / registration: r := pathkey.CanonicalPath(abs)
// check: compare on canonical root identity, keeping the walk path lexical-fast:
if pathkey.HasPathPrefix(path, r) || pathkey.CanonicalHasPathPrefix(path, r) { ... }
```
At minimum, also store `filepath.EvalSymlinks(root)` for each tracked root and test against both spellings.

### WR-04: The explore artifact lane emits matched source lines without any source gate, so the physical-owner fix is not applied everywhere

**File:** `internal/mcp/explore_artifact_intent.go:386-410,431` (outside the changed files; the review brief asked whether every source surface is consistent)

**Issue:** The artifact probe runs `GrepTextForRepos`/`GrepText`, enriches the matches, and stores `truncateExploreArtifactSnippet(match.Text)` as the target's `source`. No `explore*.go` file calls `requireSourceScope` or `requirePathSourceScope`. Attribution goes only through `enrichTextMatchesContext`'s lexical node scope. In the `outer_indexed_first` state, which WR-02 keeps alive indefinitely, an alpha-scoped `explore` returns snippet bytes from beta's file through the `alpha/nested/...` node. When `scope.RepoAllow` is nil, the search fans out across all repositories with no unbound refusal at all.

**Fix:** Route the matches through the same gate as `search_text` before taking snippets:
```go
matches = s.filterTextMatchesByResolvedScope(ctx, matches, ResolvedScope{WorkspaceID: scope.WorkspaceID, ProjectID: scope.ProjectID, RepoAllow: scope.RepoAllow})
```
Refuse up front with `requireSourceScope(ctx, "")`, as `search_text` does. Add `explore` to `TestSourceScope_NestedCheckoutRefused`.

### WR-05: `find_declaration`'s regex branch still caps before filtering, and the new physical-owner filter makes false "no use sites matched" answers more likely (prior WR-03)

**File:** `internal/mcp/tools_find_declaration.go:169-185`

**Issue:** `GrepRegexp(useSite, pathPrefix, limit)` stops at `limit` (default 20) across every repository. `allowed` runs afterwards and now also drops matches whose physical owner is out of scope. A bound caller whose first `limit` raw hits are out of scope gets `"no use sites matched"` even though in-scope matches exist. The literal branch avoids this by scanning with limit 0 and capping after the filter.

**Fix:** Mirror the literal branch:
```go
matches, err = s.multiIndexer.GrepRegexp(useSite, pathPrefix, 0) // or GrepRegexpForRepos with the resolved allow-set
kept := matches[:0]
for _, m := range matches {
    if allowed(m) { kept = append(kept, m); if len(kept) >= limit { break } }
}
```
Add a test where an out-of-scope repository holds more than `limit` regex matches.

## Info

### IN-01: Scope is re-resolved and `EvalSymlinks` runs once per unique match path

**File:** `internal/mcp/tools_find_declaration.go:153-167`; `internal/mcp/tools_search_text.go:252-284`; `internal/mcp/tools_fileops.go:513-548`; `internal/mcp/source_scope.go:107-128`

**Issue:** The cache moved from per-repository to per-path. Each new path pays `resolveScopeForRequest` once or twice, plus `EvalSymlinks` and `RepoForFile` when the lexical repository is in scope. That cost is invariant per request. The literal `find_declaration` branch calls `GrepText(useSite, 0)`, so the number of unique paths is unbounded. `resolveNodePath` also runs `requireSourceScope` twice per node, once early and once inside `requirePathSourceScope`, for every `withAbsPaths` listing row. The comment at `tools_search_text.go:270` ("the searcher stops at limit, so the cache is bounded") does not hold for the `find_declaration` copy.

**Fix:** Resolve the allowed repository set once per request (a small `sourceScopeChecker` closure built from `requireSourceScope`'s allow computation) and reuse it for both the lexical and the physical check. Skip `EvalSymlinks` when `RepoForFile(abs)` already equals the lexical repository and `os.Lstat` shows a non-symlink.

### IN-02: View-routed `search_text` matches are anchored to the canonical root, not the view's checkout

**File:** `internal/mcp/tools_search_text.go:274-279`

**Issue:** `searchTextInView` returns matches from the view's working copy. The new gate anchors them with `ResolveFilePath`, which joins against the primary root, and never applies `requestViewPathRoot(ctx).rooted(...)` the way `resolveGraphPath` does. Suppose a path is a nested tracked root in the primary checkout but not in the worktree. The view's own bytes are then refused as beta's, which is a false refusal. The error goes in the fail-closed direction, but it is inconsistent with `resolveGraphPath`.

**Fix:** Re-root with `requestViewPathRoot(ctx).rooted(abs, root)` before calling `requirePathSourceScope`, as `resolveGraphPath` does at `tools_fileops.go:626-635`.

### IN-03: Graph-backed reads now require the linked worktree's prefix to be in scope when the worktree fallback re-roots

**File:** `internal/mcp/tools_fileops.go:540-548` (with `worktreeRootedPath`, `tools_fileops.go:274-308`)

**Issue:** `worktreeRootedPath` re-roots a missing main-checkout file onto a registered linked worktree (`LinkedWorktreeRoots` lists `mi.repos` entries with `IsWorktree`). `RepoForFile` then returns the worktree's prefix, which differs from `node.RepoPrefix`, so a request scoped to `repo: alpha` now needs `alpha@wt` in scope as well. Otherwise `get_symbol_source` answers "symbol not found". `resolveFilePath` already behaved this way. The graph-backed surfaces newly inherit the behavior, and no test covers it.

**Fix:** Confirm that `repo: alpha` expands to its worktree instances. If it does not, treat a worktree of the same main repository as the lexical owner in `requirePathSourceScope`. Add a test either way.

### IN-04: Test gaps let the CR-01, WR-01, WR-02 and WR-03 defects pass

**File:** `internal/indexer/nested_tracked_root_test.go:74-154`; `internal/mcp/source_scope_test.go:328-355`

**Issue:**
- `IndexedOnce` lists both repositories in top-level `repos:`, the only shape where the seed is right. No test covers a `projects:` map, an inactive entry, untrack, a symlinked spelling, or the live `mi.repos` path during a walk. The 04-18 summary concedes that `OuterReindexDropsDuplicates` "passes on the config seed alone".
- The `find_declaration` subtest only asserts that no surviving match is nested. With no positive control (for example, a beta-scoped call returns `beta/main.go`), an empty result passes, including the `tracked_together` case where pruning leaves nothing to gate.
- The `batch_symbols` subtest only asserts `NotContains`.
- The search_symbols content-fallback behavior change flagged in 04-17-SUMMARY (fail-closed drop for Reach-unbound requests) has no test.

**Fix:** Add the probes described in CR-01, WR-01, WR-02 and WR-03 as tests. Add positive controls to the `find_declaration` and `batch_symbols` subtests. Add one test that asserts the content fallback's behavior under an unbound multi-repo request.

### IN-05: The "caller must not hold mi.mu" invariant on walk and watcher admission is documented but not enforced

**File:** `internal/indexer/workspace_resolve.go:146-154`; `internal/indexer/indexer.go:5706-5708`

**Issue:** The audit above found no caller holding `mi.mu`. But `shouldExclude` is now a hidden `mi.mu.RLock` acquisition on every walked entry. `sync.RWMutex` read locks are not reentrant once a writer is queued. Any future helper that walks or probes `PathIndexability` under `mi.mu.RLock`, which is a natural thing to write in `multi.go`, would deadlock intermittently under load.

**Fix:** State the invariant on `shouldExclude`, `PathIndexability` and `Watcher.isExcluded`, not only on the helper. Alternatively, publish registered roots as an `atomic.Pointer[[]string]` that is updated wherever `mi.repos` is written, and drop the lock from the per-file path entirely. That also resolves WR-01's staleness.

### IN-06: Carried-forward accepted risk: the `rename_symbol` refusal now also reveals nested physical owners (prior WR-04)

**File:** `internal/mcp/tools_coding.go:2705-2730` (`resolvePath` / `scopeErr`)

**Issue:** An alpha-scoped `dry_run` rename of an alpha symbol now also tells the caller whether a file under a nested out-of-scope root references it, through the distinct "some references lie outside the resolved scope" refusal. This is still the 04-12 accepted risk, now with a slightly wider surface.

**Fix:** If the acceptance is revisited, return the generic not-found text.

---

_Reviewed: 2026-10-13T19:23:03Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
