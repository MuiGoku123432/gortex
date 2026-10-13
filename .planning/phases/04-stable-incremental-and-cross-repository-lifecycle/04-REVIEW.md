---
phase: 04-stable-incremental-and-cross-repository-lifecycle
reviewed: 2026-10-13T22:39:52Z
depth: standard
review_kind: incremental re-review of gap-closure round 2, plans 04-19..04-22 (diff base c9b6fd1a, HEAD 88fd602d)
files_reviewed: 16
files_reviewed_list:
  - cmd/gortex/daemon_state.go
  - internal/indexer/cobol_lifecycle_crossrepo_test.go
  - internal/indexer/indexer.go
  - internal/indexer/multi.go
  - internal/indexer/nested_tracked_root_test.go
  - internal/indexer/repository_untrack.go
  - internal/indexer/workspace_resolve.go
  - internal/mcp/explore_artifact_intent.go
  - internal/mcp/source_scope_test.go
  - internal/mcp/tools_explore.go
  - internal/mcp/tools_find_declaration.go
  - internal/mcp/tools_find_declaration_test.go
  - internal/mcp/tools_generate_skill.go
  - internal/mcp/tools_generate_skill_test.go
  - internal/mcp/tools_move_inline.go
  - internal/mcp/tools_move_inline_test.go
findings:
  critical: 1
  warning: 7
  info: 10
  total: 18
status: issues_found
---

# Phase 4: Code Review Report (re-review of 04-19..04-22)

**Reviewed:** 2026-10-13T22:39:52Z
**Depth:** standard (incremental: `git diff c9b6fd1a..HEAD` over the 16 files, with call chains followed into `requirePathSourceScope`, `filterTextMatchesByResolvedScope`, `GrepRegexpForRepos`, `trackRepoSourceCtx`, `reach.BeginTopologyMutation`, `indexMultiRepo` publication, the warmup worker pool, `resolveOverlayAbsPath`, and `AtomicWriteFile`)
**Files Reviewed:** 16
**Status:** issues_found

## Narrative Findings (AI reviewer)

## Summary

Round 2 makes four changes:

- **04-19:** explore's artifact lane now refuses unbound multi-repo requests and passes its matches through the physical-owner filter.
- **04-20:** the per-Indexer config seed is replaced by live registered roots plus reference-counted declared roots.
- **04-21:** tracking or untracking a repository now runs a scoped reconcile of every repository that encloses it.
- **04-22:** each entry of the move, inline and generate_skill walks is gated, and the `find_declaration` regex branch filters before it caps.

I ran throwaway probes with `go test -overlay`. The tree was not modified.

**What holds:**

- **Round-1 CR-01, WR-01 and WR-02 are closed for the shapes they cited.** I probed a warm restart in which alpha is reconciled from persisted mtimes and a newly configured nested beta is cold-tracked in the same declared batch. Alpha's copies are dropped and beta owns the file. In a second probe, nested files arrived through the watcher's Indexer-level `IncrementalReindexPaths` before beta was tracked. The committed `FileMtimes` snapshot did include them, and `TrackRepoCtx` evicted alpha's copy.
- **Lock order.** `DeclarePendingRoots` and its release take `mi.mu.Lock` only for map updates. The daemon declares inside `RunRepositoryTopologyBatch` (gate, then `mi.mu`), which matches the established order. `reconcileEnclosingRepos`:
  - runs only after `trackRepoSourceCtx` or `untrackRepoTeardown` has returned, so every lane, gate and topology-writer defer has already run;
  - holds `mi.mu.RLock` only while it reads an immutable committed `FileMtimes` map;
  - is skipped inside a topology batch.

  Concurrent tracks stay serialized by the exclusive reach topology writer (`reach.go:276-288`), so the "outer not yet published" race I looked for cannot occur while `WritesBaseTopology` is true. I found no recursion path from `IncrementalReindexRepo` back into track or untrack.
- **Release and refcount.** Release is `sync.Once`-wrapped and deferred immediately at both declaration sites, so lane errors and panics are covered. Duplicate roots are counted and released symmetrically.
- **Explore gate.** The refusal is decided from `classifyExploreArtifactIntent(task)` alone, and its text names no repository.

**What does not hold:**

1. The per-entry walk gate that 04-22 added says it covers "a symlinked file", but it admits any symlink whose target is **outside every tracked root**. I confirmed with a probe that `generate_skill` copied an out-of-tree file's bytes into the repository's skill bundle (CR-04).
2. Declared roots over-prune when a declared repository fails to register, and nothing re-admits the files afterwards (WR-06).
3. Enclosing-reconcile failures are only logged (WR-07).
4. The 04-19 residual (`preview_edit`/`simulate_chain` reading through the ungated `resolveOverlayAbsPath`) is still open. That resolver also accepts absolute paths with no confinement (WR-10).

### Status of round-1 findings (review at base d1d1a688)

| Round-1 finding | Status after 04-19..04-22 |
|---|---|
| CR-01: nested-root walk seed comes from the wrong repository list | **Resolved** (04-20). Pruning reads `mi.repos` plus `pendingRoots` live, and batches declare the list they actually index. Tests cover `projects:`, another active project, and the project filter. Residual on failed registration: WR-06 (new). |
| WR-01: seed never refreshes after untrack | **Resolved** (04-20, 04-21). The per-Indexer copy is gone, and `untrackRepoChecked` admits the subtree into enclosing repositories. |
| WR-02: tracking a nested repository later freezes outer duplicates | **Resolved** for `TrackRepoCtx` (04-21) and for the warm-restart batch (probe). `indexMultiRepo` still has no enclosing eviction (IN-10, new). |
| WR-03: nested-root containment is lexical only | **Open, extended.** `DeclarePendingRoots`, `nestedTrackedRootContains` and the new `reconcileEnclosingRepos` are all lexical. See Prior review. |
| WR-04: explore artifact lane emits source lines without a gate | **Resolved** (04-19). Unbound refusal plus `filterTextMatchesByResolvedScope`. Tested in both nested states, with positive controls. |
| WR-05: `find_declaration` regex caps before filtering | **Partially resolved** (04-22). The cross-repository case is fixed. The per-repository cap still runs before the physical-owner filter. See Prior review. |
| IN-01: scope re-resolved and `EvalSymlinks` per path | **Open, broadened.** Every walk entry in move, inline and generate_skill now pays for it (performance; noted only). |
| IN-02: view-routed `search_text` anchored to the canonical root | **Open.** File not touched. |
| IN-03: worktree fallback requires the worktree prefix in scope | **Open.** File not touched. |
| IN-04: test gaps let CR-01 and WR-01..03 pass | **Partially resolved.** The CR-01, WR-01 and WR-02 shapes and the positive controls are now tested. WR-03 (symlink or alternate spelling) and the CR-04 symlink class have no test. |
| IN-05: "caller must not hold mi.mu" not enforced | **Open.** The predicate now also iterates `pendingRoots` under the same RLock. Still documented, not enforced. |
| IN-06: `rename_symbol` refusal reveals nested owners (accepted risk) | **Open (accepted).** The skip-versus-refuse policy now also governs the move and inline walks (WR-08). |

## Critical Issues

### CR-04: Per-entry walk gate admits symlinks whose target is outside every tracked root, so generate_skill copies arbitrary file bytes into the repository

**File:** `internal/mcp/tools_generate_skill.go:146-205`; `internal/mcp/tools_move_inline.go:254-283,1148-1172`; root cause `internal/mcp/source_scope.go:114-127`

**Issue:** 04-22's comments say the per-entry gate stops "a tracked root nested under absDir, **or a symlinked file**". `requirePathSourceScope` resolves the entry with `EvalSymlinks`, and when `RepoForFile(real)` is `""` (the target lies outside every tracked root) it returns nil (`physical == "" → return nil`, line 124). `filepath.WalkDir` reports a symlink as a non-directory, and `os.ReadFile(path)` (line 198) follows it.

Probe: in `newNestedGoModuleServer`, I created `alpha/a/link.txt -> <tmp>/secret.txt` and called `generate_skill {directory:"alpha/", repo:"alpha"}`. The result was `references` containing `{"rel_path":"a/link.txt","bytes_written":25}`, and `alpha/.claude/skills/alphaskill/references/a/link.txt` held `TOP-SECRET-OUTSIDE-ROOTS`.

The copy is a regular file inside alpha. `read_file` then serves it, and an agent may commit it. A repository that ships a symlink such as `docs/notes.md -> ../../.ssh/id_rsa` can therefore exfiltrate any file the daemon can read. This breaks the SECURITY.md confinement invariant that `guardSymlinkWithinRepo` enforces for `read_file`.

move_symbol and inline_symbol have the same hole for `*.go` symlinks. The target's bytes are read and rewritten, and `AtomicWriteFile`'s rename replaces the link with an in-repository copy of the out-of-tree file. The behavior predates this round, but these files were rewritten in this round specifically to close the symlink class. `TestGenerateSkill_NestedTrackedRootExcluded` has no symlink case.

**Fix:** Confine each entry before gating its owner, and fail closed on non-regular files:
```go
// generate_skill WalkDir callback, before reading:
if d.Type()&fs.ModeSymlink != 0 {
    if s.guardSymlinkWithinRepo(ctx, path) != nil { skipped++; return nil }
}
if s.requirePathSourceScope(ctx, rootRepo, path) != nil { skipped++; return nil }
```
Apply the same `guardSymlinkWithinRepo` check in both move/inline walk callbacks and in the `callerFiles` read loops (`tools_move_inline.go:285,1188`). Better still, make `requirePathSourceScope` refuse `physical == ""` whenever `guardResolvedPathWithinRepo(ctx, absPath, real)` fails, so every caller inherits the confinement. Add a symlink-out-of-tree case to the generate_skill, move and inline tests.

## Warnings

### WR-06: Declared roots over-prune when a declared repository fails to register, and nothing re-admits the files

**File:** `cmd/gortex/daemon_state.go:644-645,712-755`; `internal/indexer/multi.go:2257-2263,2347-2366`

**Issue:** The warmup declares every `gc.Repos` entry before the pool starts, and `indexMultiRepo` declares every resolved entry before its workers run. If beta's `TrackRepoCtx` or `ReconcileRepoCtx` then fails, the failure is logged ("startup track failed" or "startup reconcile failed"); the same happens if the per-entry panic guard fires or the `indexMultiRepo` worker returns `indexErr`. In every case, beta never reaches `mi.repos`. Alpha's walk, or its warm census, has already pruned or purged beta's subtree. The release then drops the declaration, but:

- the warmup skips `reconcileEnclosingRepos` because it runs inside a topology batch;
- `indexMultiRepo` has no enclosing reconcile at all.

Alpha therefore never re-admits the files. They disappear from the graph while `RepoForFile` attributes them to alpha. This is the over-prune half of round-1 CR-01, moved from "entries not in the batch" to "entries in the batch that fail".

The same window also affects an already-registered outer repository whose watcher runs while a batch is in flight. The watcher drops events under a declared root. If that root never registers, the dropped events are never replayed.

**Fix:** After the batch, compare the declared roots with `mi.repos`. For each declared root that did not register, run `reconcileEnclosingRepos(context.Background(), root, false)` after `release()`. For the daemon, do this after `RunRepositoryTopologyBatch` returns. For `indexMultiRepo`, do it after the lane and gate are released. Add a test that declares beta, forces its track to fail (for example an unreadable root, or a hook seam), and asserts that `alpha/nested/main.go` is indexed afterwards.

### WR-07: Enclosing-reconcile failures are only logged: track and untrack report success while the outer repository keeps duplicates or misses files

**File:** `internal/indexer/multi.go:2987-2994`; `internal/indexer/repository_untrack.go:56-60`

**Issue:** `reconcileEnclosingRepos` turns every `IncrementalReindexRepo` error into a `Warn` log. `TrackRepoCtx` and `untrackRepoChecked` then return the trigger's own success.

- On track, a failed eviction leaves `alpha/nested/*` duplicates. Alpha's watcher now refuses events under that root, so the duplicates are frozen (round-1 WR-02's symptom) and nothing retries.
- On untrack, a failed admission leaves the subtree out of the graph until alpha's next full reindex.

Neither outcome reaches the caller, `index_health` or the lifecycle catalog. The function also takes a `ctx` that it ignores past the batch check, so an MCP request that is cancelled mid-reconcile keeps running.

**Fix:** Return an aggregated error, or a `DeferredEnclosingReconcile` list, from `reconcileEnclosingRepos`. Surface it on `IndexResult` (or as a joined error) without failing the already-committed track or untrack. Also record the outer prefix as stale, so the next watcher tick or warmup census retries it.

### WR-08: Scoped move_symbol and inline_symbol now silently leave out-of-scope callers unrewritten and report success

**File:** `internal/mcp/tools_move_inline.go:254-283,1148-1172`; `internal/mcp/tools_move_inline_test.go` (`TestMoveSymbol_NestedTrackedRootNotRewritten`)

**Issue:** The new tests prove that `beta/use.go`, which imports `example.com/alpha/a` and calls `a.Foo()`, is left byte-identical while `Foo` moves to package `b`. The tool returns a non-error result whose `touched` list omits beta, and it gives no signal that callers were skipped. Beta no longer compiles.

Before 04-22 the walk leaked by rewriting out-of-scope files. Now it silently produces a broken refactor. The summary defers this as the "skip vs refuse" policy (tied to IN-06). It is still a correctness defect in shipped behavior, not only a policy note.

**Fix:** Pick one of two options:
- Refuse the move or inline with the generic `errSourceOutOfScope` text when any walk entry or edge-loop caller is refused. This is consistent with `rename_symbol`'s accepted-risk refusal.
- Always emit a fixed advisory, for example `"callers outside the resolved scope are not rewritten"`, whenever the walk refused any entry, with no count and no path, so it does not reveal which files exist.

Add an assertion for whichever behavior is chosen.

### WR-09: View attribution overrides a more specific registered owner in the physical-owner gate, and generate_skill's 04-22 deviation copies the same rule

**File:** `internal/mcp/source_scope.go:118-122`; `internal/mcp/tools_generate_skill.go:150-156`

**Issue:** The gate sets `physical = RepoForFile(real)`, then overwrites it unconditionally with `view.repoPrefix` whenever the request's view checkout contains `real`. Suppose a repository is tracked at a root nested inside a materialized worktree checkout, for example `alpha-wt/vendor-clone` tracked as `gamma`. `RepoForFile` returns `gamma`, but the override relabels it as alpha. An alpha-scoped view request then reads, bundles or rewrites gamma's files.

This is the same nested-owner bypass the phase closed for the primary checkout, reopened through the view path. 04-22's "generate_skill root repository honors the request's view checkout" deviation applies the identical override to `rootRepo`.

**Fix:** Apply the view override only when the registered owner is empty or is the view's own repository, or a worktree instance of it:
```go
if view := requestViewPathRoot(ctx); view.contains(real) && (physical == "" || physical == view.repoPrefix || sameMainRepo(physical, view.repoPrefix)) {
    physical = view.repoPrefix
}
```
Do the same for `rootRepo`. Add a test with a tracked root nested inside a worktree view.

### WR-10: preview_edit and simulate_chain still read through ungated `resolveOverlayAbsPath`, which also accepts any absolute path (04-19 residual)

**File:** `internal/mcp/overlay_view.go:465-483`; `internal/mcp/tools_simulate.go:810-860,1005-1040`; `internal/mcp/tools_review.go:1401-1405`

**Issue:** 04-19's own threat flags record this residual, but no plan closed it. `groupEditByFile` resolves every WorkspaceEdit URI through `resolveOverlayAbsPath`. That function returns `filepath.Clean(p)` for any absolute path and calls neither `requireSourceScope` nor `guardSymlinkWithinRepo`. The simulation then:
- reads the file (`readSimFile`);
- parses it into the speculative graph diff, which returns symbol names, signatures and line ranges;
- pushes it to an LSP server, whose diagnostics are returned.

As a result:
- An unbound or alpha-scoped caller can learn the structure of beta's files, including nested ones.
- Any caller can learn the structure of files outside every tracked root.
- With `keep: true`, the caller can promote that content into a session overlay.

The 04-19 inventory test cannot see this, because it tracks only `Grep*` and `Walk*` call sites (IN-08).

**Fix:** Have `groupEditByFile` resolve paths through `resolveFilePath(ctx, path)`, which applies the lexical gate, the physical-owner gate and confinement. Treat a refusal as a tool error carrying the generic not-found text. Keep the bare-path fallback only for new files inside an in-scope root. Add `preview_edit` and `simulate_chain` cases to `TestSourceScope_UnboundMultiRepoRefused` and `TestSourceScope_NestedCheckoutRefused`.

## Info

### IN-07: `DeclarePendingRoots` was inserted between `indexMultiRepo`'s doc line and its function

**File:** `internal/indexer/multi.go:2126-2180`

**Issue:** The line `// indexMultiRepo indexes multiple repos concurrently with repo prefixes.` now opens `DeclarePendingRoots`'s godoc, and `indexMultiRepo` has no doc comment. The doc also says roots are spelled "like mi.repos", but `trackRepoSourceCtx` additionally applies `pathkey.NormalizeVolume`. That is harmless on POSIX and only case-folded on Windows.

**Fix:** Move the `indexMultiRepo` line back above `func (mi *MultiIndexer) indexMultiRepo`, and either drop "like mi.repos" from the spelling note or normalize the volume.

### IN-08: The source-channel inventory test is name-keyed and covers only `Grep*` and `Walk*` selector calls

**File:** `internal/mcp/source_scope_test.go:519-597`

**Issue:** The inventory misses four kinds of call site:
- a new call site inside an already-listed function;
- method values (`g := s.multiIndexer.GrepText; g(...)`);
- package-level function literals;
- every `os.ReadFile`, `os.Open` and `readSimFile` channel.

The last gap is how WR-10's ungated reads stayed invisible. The 04-19 summary's `os.ReadFile` audit is manual and unenforced. The 04-22 tests also have no symlink case, although the code comments claim symlink coverage (CR-04).

**Fix:** Key the inventory on `file:function:callee` counts. Add `os.ReadFile`, `os.Open` and `readSimFile` to the tracked callees with the path source each one uses. Add symlink cases to the walk tests.

### IN-09: Explore's artifact lane still caps the text probe before the scope filter when `RepoAllow` is nil

**File:** `internal/mcp/explore_artifact_intent.go:389-400`

**Issue:** For a workspace- or project-bound request with no explicit repository allow-set, the lane calls `GrepText(probe, exploreArtifactTextHitLimit)`, a global cap across all repositories. `filterTextMatchesByResolvedScope` runs only after that. Out-of-workspace matches can fill the cap and starve in-scope hits, which gives false "nothing found" answers. This is the same pattern as WR-05.

**Fix:** Expand the workspace or project to its repository set (`ReposInWorkspace`) and call `GrepTextForRepos`. Alternatively, scan uncapped and cap after the filter, as `find_declaration`'s literal branch does.

### IN-10: `indexMultiRepo` registers a nested repository without evicting an already-registered enclosing repository outside the batch

**File:** `internal/indexer/multi.go:2347-2366`

**Issue:** Only `TrackRepoCtx` and untrack call `reconcileEnclosingRepos`. Suppose `IndexScoped("", "p2")` indexes beta while alpha (project p1) is already registered and holds `alpha/nested/*`. Beta registers, alpha's watcher starts refusing that subtree, and alpha's copies stay frozen. This is round-1 WR-02's state. `IndexScoped` and `IndexAll` currently have no production callers, so this is latent.

**Fix:** After publication and release, run `reconcileEnclosingRepos(ctx, meta.RootPath, true)` for each newly registered prefix whose enclosing repository was not in the batch.

## Prior review (round-1 findings still open)

### WR-03: Nested-root containment is lexical only; it now also covers declared roots and the enclosing reconcile

**File:** `internal/indexer/workspace_resolve.go:152-175`; `internal/indexer/multi.go:2143-2178,2951-2985`

**Issue:** This is unchanged from round 1, and the new code inherits it:
- `DeclarePendingRoots` stores `filepath.Clean(filepath.Abs(p))`.
- `nestedTrackedRootContains` compares with `pathkey.HasPathPrefix`, which folds case only.
- `reconcileEnclosingRepos` selects enclosing repositories with the case-folding `HasPathPrefix`, then computes `sub` with the case-sensitive `filepath.Rel(r, root)` (line 2967). With a case-only spelling difference on a case-insensitive filesystem, `Rel` yields `../Alpha/nested`, `held` is false, and the eviction is skipped.

A symlinked spelling (`/var` vs `/private/var`, or a link to `alpha/nested`) is still not pruned. There is still no test.

**Fix:** As in round 1, canonicalize the roots at registration and declaration time and compare with `pathkey.CanonicalHasPathPrefix`. In `reconcileEnclosingRepos`, compute `rel` from canonical spellings, or use `relativeWithinRoot`-style alias-aware logic.

### WR-05: `find_declaration`'s regex branch still caps each repository before the physical-owner filter (narrowed)

**File:** `internal/mcp/tools_find_declaration.go:173-195`

**Issue:** `GrepRegexpForRepos(useSite, pathPrefix, nil, limit)` caps **each repository** at `limit` before `allowed()` runs. Two cases still give false "no use sites matched" answers:
- an in-scope repository whose first `limit` raw hits all lie under a nested tracked root or behind symlinks into an out-of-scope repository;
- an alpha-scoped regex while alpha still holds copies (WR-06 / WR-07 states).

Passing `nil` as the allow-set also greps repositories that are out of scope. The new test exercises only the cross-repository case.

**Fix:** Pass the resolved allow-set to `GrepRegexpForRepos`. Within each repository, either scan uncapped and cap after `allowed()`, or loop with a growing per-repository limit until `limit` filtered hits are found or the repository is exhausted. Add a test where alpha's first `limit` regex hits lie under the nested root.

### IN-01: Scope is re-resolved and `EvalSymlinks` runs once per path, and now once per walk entry

**File:** `internal/mcp/source_scope.go:107-128`; `internal/mcp/tools_move_inline.go:265,273,1158,1166`; `internal/mcp/tools_generate_skill.go:175,180`

**Issue:** This is unchanged from round 1. In addition, each directory and file in a module walk now pays `resolveScopeForRequest` twice, plus `EvalSymlinks` and `RepoForFile`. It is noted for maintainability only, since performance is out of scope.

**Fix:** Same as round 1: resolve the allow-set once per request and reuse a checker closure.

### IN-02: View-routed `search_text` matches are anchored to the canonical root, not the view's checkout

**File:** `internal/mcp/tools_search_text.go:274-279`

**Issue:** Unchanged; the file was not touched.

**Fix:** Same as round 1: re-root through `requestViewPathRoot(ctx).rooted(...)` before `requirePathSourceScope`.

### IN-03: Graph-backed reads require the linked worktree's prefix to be in scope when the worktree fallback re-roots

**File:** `internal/mcp/tools_fileops.go:540-548`

**Issue:** Unchanged; there is still no test.

**Fix:** Same as round 1.

### IN-04: Test gaps (partially resolved)

**File:** `internal/indexer/nested_tracked_root_test.go`; `internal/mcp/tools_generate_skill_test.go`; `internal/mcp/tools_move_inline_test.go`

**Issue:** The CR-01, WR-01 and WR-02 shapes, the positive controls and the content-fallback test are now present. These cases remain untested:
- WR-03, a symlinked or alternate spelling of the nested root;
- the CR-04 symlink-out-of-tree class;
- the WR-06 failed registration;
- WR-05's intra-repository physical-owner case.

**Fix:** Add the tests listed in CR-04, WR-03, WR-05 and WR-06.

### IN-05: The "caller must not hold mi.mu" invariant on walk and watcher admission is documented but not enforced

**File:** `internal/indexer/workspace_resolve.go:147-175`; `internal/indexer/indexer.go:5693-5701`

**Issue:** Unchanged. The per-file predicate now iterates `pendingRoots` as well, still under a hidden `mi.mu.RLock` on every walked entry. `DeclarePendingRoots` adds two more `mi.mu.Lock` writers, which makes reentrant-RLock deadlocks more likely if a future helper walks under `mi.mu`.

**Fix:** Same as round 1. Publish `registered ∪ pending` roots as an `atomic.Pointer[[]string]` that is rebuilt wherever `mi.repos` or `pendingRoots` change, and drop the lock from the per-file path.

### IN-06: Carried-forward accepted risk: the `rename_symbol` refusal reveals nested physical owners

**File:** `internal/mcp/tools_coding.go:2705-2730`

**Issue:** Unchanged (accepted 04-12). The related skip-versus-refuse decision for the move and inline walks is now a correctness issue (WR-08).

**Fix:** Decide the policy once for rename, move and inline together.

---

_Reviewed: 2026-10-13T22:39:52Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
