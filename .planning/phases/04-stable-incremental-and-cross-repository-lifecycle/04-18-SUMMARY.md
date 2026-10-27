---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 18
subsystem: indexer
tags: [nested-checkout, cr-01, ownership, walk-pruning, watcher, multi-repo, lock-order]

requires:
  - phase: 04-10
    provides: physical-owner rule (MultiIndexer.RepoForFile longest root) and the nested-checkout CR-01 finding
  - phase: 04-17
    provides: physical-owner source gate that keeps source surfaces consistent while outer-repo duplicates still exist
provides:
  - Indexer.trackedRepoRoots seeded once per Indexer from GlobalConfig.ReposSnapshot in newPerRepoIndexerGuardedWithMode
  - Indexer.inNestedTrackedRoot (config seed, then live registered roots) consulted by shouldExclude
  - MultiIndexer.nestedTrackedRootContains over mi.repos under mi.mu.RLock
  - Watcher.isExcluded refuses paths under a nested tracked root
  - admitWalkFileKnownType regular-file fast path refuses nested-root files (re-include descent, PathIndexability)
  - TestNestedTrackedRoot_IndexedOnce/{alpha_first,beta_first}, TestNestedTrackedRoot_OuterReindexDropsDuplicates, TestNestedTrackedRoot_WatcherRefusesNestedEvents
affects: [phase-04-verification, source-scope, ownership-audit, watcher, warm-reconcile]

actuals:
  tokens: 3544
  tasks: 2
  commits: 4
plan_head_before: 366930a0ce8f0d6e4f544cabc14688a532ecf6fe
plan_head_after: a9fecc764e984d3be3fda54ab1a46bd1e94fb9ba

tech-stack:
  added: []
  patterns:
    - "Walk and watcher admission stop at any tracked root nested strictly inside the walk root; one path, one owner (longest tracked root)"
    - "Config roots copied once per Indexer at construction; registered roots read live under mi.mu.RLock by callers that never hold mi.mu"

key-files:
  created:
    - internal/indexer/nested_tracked_root_test.go
  modified:
    - internal/indexer/indexer.go
    - internal/indexer/multi.go
    - internal/indexer/workspace_resolve.go
    - internal/indexer/watcher.go
    - internal/indexer/walk_source.go

key-decisions:
  - "Kept the plan's live mi.mu.RLock read of registered roots instead of pushing roots to Indexers at publication points: pushing would need per-Indexer synchronization on every track (the watcher case needs the later-tracked root on an already-built Indexer), which is more code than one RLock read. The lock-order audit found no caller holding mi.mu."
  - "admitWalkFileKnownType's regular-file fast path (which skips shouldExclude to avoid a second Lstat) now also calls inNestedTrackedRoot; otherwise a `!` re-include under the nested root, or PathIndexability (which never prunes), re-admits beta's files to alpha."
  - "OuterReindexDropsDuplicates passes on the config seed alone because TrackRepoCtx calls Global().AddRepo, so IndexRepo(alpha) builds a fresh Indexer whose seed lists beta; the watcher test is the one that exercises the live registered-root read."

patterns-established:
  - "Any admission path that bypasses Indexer.shouldExclude must call inNestedTrackedRoot itself"

requirements-completed: [SEC-08, SEC-01]

coverage:
  - id: D1
    description: "Nested pair tracked together (both config orders): no alpha/nested/main.go node, beta and alpha symbols present, ownership audit clean; alpha's shouldExclude and PathIndexability refuse beta's root and file; beta's indexer admits its own file"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_IndexedOnce"
        status: pass
    human_judgment: false
  - id: D2
    description: "Outer repository indexed before the nested repository was tracked drops its duplicates on its next IndexRepo; audit clean"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_OuterReindexDropsDuplicates"
        status: pass
    human_judgment: false
  - id: D3
    description: "Outer watcher refuses betaDir and betaDir/main.go right after beta is tracked (no alpha reindex) and admits alphaDir/main.go; beta's watcher admits betaDir/main.go"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_WatcherRefusesNestedEvents"
        status: pass
    human_judgment: false
  - id: D4
    description: "No regression: full internal/indexer package, MCP source-scope suite, and cmd/gortex COBOL/embedded suites under -race"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/indexer/ -timeout 60m"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/mcp/ -run '^TestSourceScope_'"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol|^TestEmbeddedIndex_'"
        status: pass
    human_judgment: false

duration: 24min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 18: Nested Tracked Root Walk and Watcher Pruning Summary

**An outer repository's walk, census, explicit-path probe, and watcher now stop at any tracked root nested inside it. The config roots are seeded once per Indexer and the registered roots are read live, so a nested checkout has exactly one owner and the alpha/nested duplicates that let the ownership audit pass with two owners are no longer created (CR-01).**

## Performance

- **Duration:** 24 min
- **Started:** 2026-10-13T18:36:13Z
- **Completed:** 2026-10-13T19:00:31Z
- **Tasks:** 2
- **Files modified:** 6 (5 planned + walk_source.go)

## Accomplishments
- `Indexer.inNestedTrackedRoot(path, root)` reports whether `path` lies under a tracked root that sits strictly inside `root`. It checks the config seed first (no lock, no allocation), then `MultiIndexer.nestedTrackedRootContains`. A root equal to or above the walk root never excludes anything.
- `shouldExclude` calls it after symlink confinement and before the rule-based re-includes. `shouldPruneDir`, the cold walk, the warm census, and explicit-path incremental sites all inherit it.
- `newPerRepoIndexerGuardedWithMode` copies `GlobalConfig.ReposSnapshot()` roots into `trackedRepoRoots` once per Indexer. A parallel warmup therefore prunes a sibling that is not yet in `mi.repos`, and this does not re-enter `mi.mu`.
- `Watcher.isExcluded` refuses nested-root paths, so an edit to beta's file is never re-indexed under alpha.
- `admitWalkFileKnownType`'s regular-file fast path also checks the nested root (see Deviations).

## Task Commits

1. **Task 1 (tracer): nested tracked root indexed once when tracked together**
   - `13189aac` test(04-18): nested tracked root must be indexed once (RED)
   - `799bdf60` fix(04-18): stop an outer repository's walk at a nested tracked root
   - Tracer gate (interactive, end-of-phase, automated-only verify): verify re-run green, then expanded.
2. **Task 2: registered roots and the live watcher stop at a nested tracked root**
   - `aabfdf3b` test(04-18): later-tracked nested root, watcher and explicit-path probe (RED)
   - `a9fecc76` fix(04-18): honour registered nested roots in walks, watcher and probe

## Files Created/Modified
- `internal/indexer/indexer.go`: the `trackedRepoRoots` field, `inNestedTrackedRoot`, and the call in `shouldExclude`.
- `internal/indexer/multi.go`: config-root seed in `newPerRepoIndexerGuardedWithMode`.
- `internal/indexer/workspace_resolve.go`: `nestedTrackedRootContains`, with a `// note:` on the duplicated three-part test (two uses).
- `internal/indexer/watcher.go`: the `isExcluded` nested-root check and its doc.
- `internal/indexer/walk_source.go`: the nested-root check on the `admitWalkFileKnownType` regular-file path.
- `internal/indexer/nested_tracked_root_test.go`: the fixture helpers and the three tests.

## Lock-order audit (Task 2 step 4)

`nestedTrackedRootContains` takes `mi.mu.RLock`. `mi.mu` is unexported, so only code in `internal/indexer` can hold it. External callers (`internal/mcp/freshness_rider.go` PathIndexability; `cmd/gortex/daemon_controller_view.go` PathIndexability/ScopeIndexability; MCP `EnqueueFileMutation` via `MultiWatcher`) get the Indexer through `GetIndexer`, `RepoPrefixes`, or `IndexerForFile`, and each of those releases `mi.mu` before returning.

| Call site reaching the predicate | Caller path | `mi.mu` state at the call |
|---|---|---|
| `indexer.go:2398` `current.indexCtxRaw` (walkSource → admitWalkEntry/admitWalkFileKnownType → shouldPruneDir/shouldExclude) | `Indexer` public index entry | Not held. `owner.mu.Lock()` is taken only afterwards at `indexer.go:2414`, after the walk returns. |
| `multi.go:2261` worker `idx.indexCtxRaw` | `indexMultiRepo` | Not held. Workers run with no lock; the collector drains `resultCh` (L2305) unlocked after `wg.Wait()` (L2293), and `mi.mu.Lock()` L2315..`Unlock()` L2323 only assigns maps. |
| `multi.go:2459` `idx.indexCtxRaw` | `indexRepoRaw` (IndexRepo) | Released at `multi.go:2427` (RLock L2424); next Lock is L2468, after the walk. |
| `multi.go:2991` `idx.indexCtxRaw` | `trackRepoSourceCtx` | Released at L2924/L2929/L2934 (RLock L2922) and L2984 (RLock L2974); publication Lock L3001 is after the walk. |
| `multi.go:3173` `indexCtxRaw`, `3180` `changedSinceMtimesCensus`, `3202/3214/3219` `incrementalReindexPathsWithReceipt`, `3225/3231` `incrementalReindexPathsWithReceiptMode` | `ReconcileRepoCtx` | Released at L3091 (RLock L3088) and L3146 (RLock L3136); no `mi.mu` op between L3146 and L3240. Publication Lock L3250 comes afterwards. |
| `multi.go:2720` `incrementalReindexPathsWithReceiptMode` (→ `incrementalReindexPathsMode` `indexer.go:6313/6460`) | `incrementalReindexRepoRawMode` (IncrementalReindexRepo*, `reindexPointMutationRaw` `indexer.go:4452`) | Released at L2705 (RLock L2702); next Lock L2735, after the call. |
| `incremental_watcher_batch.go:300` via `incrementalWatcherPaths` (`indexer.go:4461`, L230/234/286), `incremental_watcher_batch.go:351/361`, `watcher.go:1459` `IncrementalReindexPaths` | Watcher / standalone Indexer paths | Not held. These run on watcher or caller goroutines with no `mi.mu`. |
| `indexer.go:9161` `changedSinceMtimesCensus` | `ChangedSinceMtimes` (no in-package caller holding `mi.mu`) | Not held. |
| `intake_manifest.go:117/135` | `dryRunIntake` on `newIntakeIndexer` | No owner, so no lock is taken. |
| `admission.go:49/109/117` | `PathIndexability` / `ScopeIndexability` (external callers above) | Not held. |
| `watcher.go:971/1040` | `reconcileInitialReplayEvents` / `matchingInitialReplayReceipts` (watcher startup goroutine, L890/L927) | Not held. |
| `watcher.go:1648` | `handleEvent` (watcher event loop, L1334) | Not held. |
| `watcher.go:1723` | `EnqueueFileMutation` ← `MultiWatcher.EnqueueFileMutation` (`multi_watcher.go:1093`) | Not held. `mw.mu` (a different mutex) is released at `multi_watcher.go:~1089` before the call. |
| `watcher_path_filter.go:21` `watcherExcludeFilter.ShouldInclude` | fswatcher filter goroutine | Not held. |

Lock holders were also checked in the other direction. Every defer-style `mi.mu` holder (`refreshColdCensusMetadata`, `boundedRepoIndexer`, `discardRetargetedTestCallFiles`, `persistScopedInlinedContracts`, `grepRepoJobs`, `LinkedWorktreeRoots`, `ScopeForCWD`, `ContainedReposScope`, `ReposInWorkspace`, and the trivial getters) and every explicit `Lock`..`Unlock` span in the package only does map/metadata work. The span calls are `ResolveWorktree`, `time.Now`, the atomic `applyMultiIndexerBatchMode` setters, `detachBatchGlobalPassStateLocked`, `ContractRegistry`, and contract shape/bare-type passes. None runs a walk, waits on a repository lane (`runExclusive`), waits on a WaitGroup or channel, or calls into the watcher. `owner.mu` in `repositoryMutationRootPath` (`repository_mutation_paths.go:111`) is a short RLock with no walk, which is the same pattern the new check uses.

**Result: no caller holds `mi.mu` when the predicate runs, and no `mi.mu` writer waits on a walk or lane, so there is no deviation to report.** The full package passes under `-race`. Recorded as a pinned Gortex invariant memory on `Indexer.inNestedTrackedRoot` / `MultiIndexer.nestedTrackedRootContains`.

## Decisions Made
- I kept the planned live RLock read instead of the orchestrator's suggested alternative (handing roots to Indexers at the publication points). The watcher case needs a root tracked later to reach an Indexer that already exists. Pushing roots would mean taking every Indexer's lock on each track, which is more code and more lock surface than one RLock read whose callers are audited.
- The config seed skips empty and non-Abs-able paths and stores `filepath.Clean(filepath.Abs(path))`. This matches how `indexCtxRaw` and `mi.repos` spell roots (Abs, no EvalSymlinks), so `HasPathPrefix` compares like with like.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Regular-file fast path bypassed the nested-root check**
- **Found during:** Task 2 (call-site audit)
- **Issue:** `admitWalkFileKnownType` sends regular files straight to `shouldExcludeRules` and skips `shouldExclude`, to avoid a second Lstat for symlink confinement. `shouldPruneDir` keeps descending an excluded directory when a `!` re-include targets a descendant, and `PathIndexability` (explicit-path probe, `admission.go:49`) never prunes. In either case alpha re-admits beta's file. Proven RED: `alpha.PathIndexability("nested/main.go")` reported indexable.
- **Fix:** `if idx.inNestedTrackedRoot(absPath, root) || idx.shouldExcludeRules(absPath, root, false)` on the fast path, plus an assertion in `TestNestedTrackedRoot_IndexedOnce`.
- **Files modified:** `internal/indexer/walk_source.go` (a sixth file, outside `files_modified`), `internal/indexer/nested_tracked_root_test.go`
- **Verification:** IndexedOnce passes in both orders; the full indexer package passes under -race.
- **Committed in:** `aabfdf3b` (test), `a9fecc76` (fix)

**2. [Plan expectation] OuterReindexDropsDuplicates was not RED before Task 2's code**
- **Found during:** Task 2 RED run
- **Issue:** The plan expected this test to fail until the registered-root read existed. `trackRepoSourceCtx` calls `mi.configMgr.Global().AddRepo(entry)` (`multi.go` ~L3020), so after `TrackRepoCtx` the in-memory config lists beta. `IndexRepo("alpha")` then builds a fresh Indexer whose config seed already includes beta.
- **Fix:** None needed. The test is kept as a behavioral guard (the plan's truth #2 holds). The live registered-root read is exercised by `TestNestedTrackedRoot_WatcherRefusesNestedEvents`, which was RED (alpha's Indexer captured its seed before the track).
- **Committed in:** `aabfdf3b`

---

**Total deviations:** 1 auto-fixed (Rule 2) and 1 plan-expectation note.
**Impact on plan:** The Rule 2 fix is required for the "every admission path" success criterion. No scope creep.

## Issues Encountered
- The plan's flagged assumption about tests that relied on double ownership did not occur. No existing test in `internal/indexer`, `internal/mcp` (TestSourceScope_), or `cmd/gortex` (TestCobol*, TestEmbeddedIndex_*) needed changes. 04-17's `outer_indexed_first` still sees alpha's pre-track duplicates, as designed.
- Out of scope: `gofmt -l internal/indexer/` lists `file_delta.go`, an upstream file this plan does not touch. Logged in `deferred-items.md`.

## Verification Run
- Task 1 verify: PASS (IndexedOnce/alpha_first and /beta_first, symlink-confinement regression, gofmt).
- Task 2 verify: PASS (all three TestNestedTrackedRoot_, TestShouldExclude*, TestWatcherIsExcluded*, TestIndexSkipsSymlinkOutOfRepo, TestWatcher_* under -race; gofmt).
- `GOWORK=off go build ./... && GOWORK=off go vet ./internal/indexer/`: PASS
- `GOWORK=off go test -race -count=1 ./internal/indexer/ -timeout 60m`: ok (992s)
- `GOWORK=off go test -race -count=1 ./internal/mcp/ -run '^TestSourceScope_'`: ok (11.6s)
- `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol|^TestEmbeddedIndex_'`: ok (38.4s)

## Threat Flags
None. No new endpoints, auth paths, or schema changes. The change only narrows what a walk or watcher admits (T-04-51..54 mitigated as planned; T-04-55 accepted as in the plan).

## User Setup Required
None. No external service configuration is required.

## Next Phase Readiness
VERIFICATION `gaps[0]` is closed at the root cause (alongside 04-17's source gate). The deferred items stand as written in the plan: eviction at track time, and the effect of untracking a nested repository on a config seed already captured by an outer Indexer.

## Self-Check: PASSED

All created files exist; commits 13189aac, 799bdf60, aabfdf3b, a9fecc76 are ancestors of HEAD; acceptance greps (nestedTrackedRootContains definition = 1, watcher inNestedTrackedRoot call = 1, indexer nestedTrackedRootContains call = 1) match.
