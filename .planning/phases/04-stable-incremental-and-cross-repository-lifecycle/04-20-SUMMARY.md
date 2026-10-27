---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 20
subsystem: indexer
tags: [nested-checkout, cr-01, wr-01, ownership, walk-pruning, watcher, multi-repo, lock-order, warmup]

requires:
  - phase: 04-18
    provides: nested-root pruning predicate (Indexer.inNestedTrackedRoot, MultiIndexer.nestedTrackedRootContains under mi.mu.RLock), its lock-order audit, and the nested_tracked_root_test.go fixtures
provides:
  - MultiIndexer.pendingRoots (reference-counted roots declared by in-flight batches, guarded by mi.mu)
  - exported MultiIndexer.DeclarePendingRoots(repos) (release func()), sync.Once-idempotent release
  - indexMultiRepo declares its resolved repository list before the batch gate
  - warmupDaemonState declares the exact list its parallel pool tracks
  - nestedTrackedRootContains tests registered roots plus declared pending roots; the per-Indexer config copy (04-18 seed) is removed
  - TestNestedTrackedRoot_ProjectsMapIndexedOnce, TestNestedTrackedRoot_OuterOwnsUnindexedNestedEntry/{other_active_project,project_filter}, TestNestedTrackedRoot_UntrackStopsExcluding, TestNestedTrackedRoot_DeclaredRootsPruneWarmupOrder; helper newNestedTrackedMultiFromConfig
affects: [phase-04-verification, 04-21, source-scope, ownership-audit, watcher, daemon-warmup]

actuals:
  tokens: 4383
  tasks: 2
  commits: 2
plan_head_before: 5d4cc638a77bed620fdff60a84bde76c01cd1313
plan_head_after: 13d9d1eb6e66c9801fcf3016d6b93f66e1b21476

tech-stack:
  added: []
  patterns:
    - "Declared, not copied: a batch declares the repositories it is about to index; pruning reads declared + registered roots live under mi.mu.RLock"

key-files:
  created: []
  modified:
    - internal/indexer/multi.go
    - internal/indexer/workspace_resolve.go
    - internal/indexer/indexer.go
    - internal/indexer/nested_tracked_root_test.go
    - cmd/gortex/daemon_state.go

key-decisions:
  - "Nested-root pruning reads only live MultiIndexer state (mi.repos plus mi.pendingRoots under one RLock); the global config is demoted to an input of the batch that declares and is never consulted by pruning"
  - "pendingRoots is reference counted per declaration so concurrent disjoint indexMultiRepo batches and the daemon warmup cannot release each other's roots; release is sync.Once-wrapped"
  - "indexMultiRepo declares before taking batchMutationGate; the daemon declares inside RunRepositoryTopologyBatch, which follows the existing gate -> mi.mu order"

patterns-established:
  - "Batch-scoped pending state on MultiIndexer: declare under mi.mu.Lock for map updates only, defer the release at the declaration site"

requirements-completed: []  # SEC-08 / ID-09 deliberately NOT marked: 04-21 and 04-22 still claim them; phase re-verification decides

coverage:
  - id: D1
    description: "A nested pair defined only under projects: (active_project) is indexed once by IndexScoped with a clean ownership audit"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_ProjectsMapIndexedOnce"
        status: pass
    human_judgment: false
  - id: D2
    description: "A nested repos: entry outside the indexed batch (other active project, or project filter) never prunes alpha; alpha owns the file, RepoForFile agrees, audit clean"
    requirement: ID-09
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_OuterOwnsUnindexedNestedEntry"
        status: pass
    human_judgment: false
  - id: D3
    description: "After UntrackRepo(beta) the same alpha Indexer stops excluding beta's former root on walk, probe and watcher, and IncrementalReindexRepo indexes the file"
    requirement: ID-09
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_UntrackStopsExcluding"
        status: pass
    human_judgment: false
  - id: D4
    description: "Daemon warmup declares the list it feeds its parallel pool, so alpha-before-beta tracking inside one topology batch still leaves beta's file single-owned; registered roots keep pruning after release"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_DeclaredRootsPruneWarmupOrder"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol|^TestEmbeddedIndex_|^TestWarmup'"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 20: Nested-root pruning from declared + registered roots Summary

**Nested-root pruning now follows the repositories a batch is actually indexing (indexMultiRepo's resolved list, the daemon warmup's pool list) plus the live registry, read under one mi.mu.RLock. That closes CR-01 under- and over-prune and WR-01's stale exclusion after untrack.**

## Performance

- **Duration:** ~35 min (including a 987s full-package `-race` run)
- **Completed:** 2026-10-13
- **Tasks:** 2/2
- **Files modified:** 5

## Accomplishments

- Removed 04-18's per-Indexer `trackedRepoRoots` config copy and its seed block in `newPerRepoIndexerGuardedWithMode`. The constructor now reads nothing and still never re-enters `mi.mu`.
- Added `MultiIndexer.pendingRoots map[string]int` and the exported `DeclarePendingRoots(repos) (release func())`. Paths are cleaned and made absolute (same spelling as `mi.repos`). Each declaration bumps a count under `mi.mu.Lock`. The release decrements and deletes at zero, wrapped in `sync.Once`.
- `indexMultiRepo` declares `r.entry` of every resolved repository after the empty-batch early return and before `batchMutationGate.RLock`, with `defer release()`.
- `nestedTrackedRootContains` checks the three-part containment against `mi.repos` and then `mi.pendingRoots`, using one local closure. The doc is rewritten and the stale `// note:` removed.
- `Indexer.inNestedTrackedRoot` keeps the empty-root guard and delegates to the owner. An Indexer without an owner prunes nothing.
- `warmupDaemonState` calls `state.multiIndexer.DeclarePendingRoots(repos)` at the top of the `RunRepositoryTopologyBatch` callback, before any worker starts, with `defer release()`. `repos` (L460) is the same slice the job loop feeds (L766). It is not reassigned in between.

## Task Commits

1. **Task 1 (tracer): nested-root pruning follows the repositories actually being indexed, read live**: `6b299de3` (fix)
2. **Task 2: the daemon warmup declares the repositories its parallel pool tracks**: `13d9d1eb` (fix)

## TDD Evidence

**Task 1 RED** (tests written first, run before any implementation change):
- `TestNestedTrackedRoot_ProjectsMapIndexedOnce` failed: `Expected nil, but got: &graph.Node{ID:"alpha/nested/main.go::ZebraWidget" ...}` (under-prune: the config seed missed the projects: entry).
- `TestNestedTrackedRoot_OuterOwnsUnindexedNestedEntry/{other_active_project,project_filter}` both failed: `Expected value not to be nil` for alpha/nested/main.go::ZebraWidget (over-prune: the seed included beta although no batch indexed it).
- `TestNestedTrackedRoot_UntrackStopsExcluding` failed at the `shouldExclude` assertion: `Should be false` (stale copy after untrack).
- `IndexedOnce/{alpha_first,beta_first}`, `OuterReindexDropsDuplicates`, `WatcherRefusesNestedEvents` stayed green.

**Task 1 GREEN:** the Task 1 verify command passed, covering all six `TestNestedTrackedRoot_` names, `TestShouldExclude*`, `TestWatcherIsExcluded*`, `TestIndexSkipsSymlinkOutOfRepo`, `TestWatcher_*` and `TestIndexAllWorkersDoNotReenterRegistryLock` under `-race`. gofmt was clean.

**Task 2 RED:** a temporary copy of the test file had `release := mi.DeclarePendingRoots(repos)` replaced with `release := func() {}`, which is the warmup after Task 1 and before Task 2. `TestNestedTrackedRoot_DeclaredRootsPruneWarmupOrder` then failed with `Expected nil, but got: &graph.Node{ID:"alpha/nested/main.go::ZebraWidget" ...}`. The call was restored (grep count 1) and the test passed.

**Task 2 GREEN:** the Task 2 verify command exited 0. It covered the list check, the nested-root, topology-batch and parallel-batch run under `-race`, `go build ./cmd/gortex/`, the `DeclarePendingRoots(repos)` count of 1, `TestWarmup*` under `-race`, and gofmt.

## Plan-level verification

| Command | Result |
|---|---|
| `GOWORK=off go build ./... && GOWORK=off go vet ./internal/indexer/ ./cmd/gortex/` | pass |
| `GOWORK=off go test -race -count=1 ./internal/indexer/ -timeout 45m` | ok (986.7s) |
| `GOWORK=off go test -race -count=1 ./internal/mcp/ -run '^TestSourceScope_' -timeout 20m` | ok (13.6s); `outer_indexed_first` still sees its duplicates, as the plan expected |
| `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol\|^TestEmbeddedIndex_\|^TestWarmup' -timeout 40m` | ok (38.0s) |
| `git diff --name-only 5d4cc638..HEAD` | exactly the 5 `files_modified` files |

No existing test depended on the removed config seed. No test expectation was changed.

## Lock-order audit

The invariant: no `mi.mu` holder walks, probes or admits, and `DeclarePendingRoots` holds `mi.mu` only for map updates.

This plan changes the lock picture in only three ways:
- The per-file predicate now also iterates `mi.pendingRoots` inside the RLock it already held. No new acquisition.
- The per-Indexer slice is removed, which involved no lock.
- There are two new `mi.mu.Lock` spans, `DeclarePendingRoots` and its release. Each does only map increments or decrements. Neither walks, waits on a lane, WaitGroup or channel, or calls out.

The predicate call sites from 04-18's table were re-checked at current line numbers. `multi.go` sites after `indexMultiRepo` moved by +49 lines and `indexer.go` `shouldExclude` moved by -5. No lock acquisition or release around any of them changed.

| Call site reaching the predicate / declaration | Caller path | `mi.mu` state at the call |
|---|---|---|
| `indexer.go:5670` `shouldExclude` -> `inNestedTrackedRoot` (L5693) -> `nestedTrackedRootContains` (`workspace_resolve.go:152`) | every walk / probe below | Not held: the predicate takes the RLock itself |
| `walk_source.go:78` `admitWalkFileKnownType` | walk, `PathIndexability` (`admission.go`) | Not held (unchanged from 04-18) |
| `watcher.go:2953` `Watcher.isExcluded` | watcher event loop / fswatcher filter / `EnqueueFileMutation` | Not held (unchanged from 04-18) |
| `multi.go:2310` worker `idx.indexCtxRaw` | `indexMultiRepo` | Not held. Workers run unlocked. Publication `mi.mu.Lock()` L2364..`Unlock()` L2372 only assigns maps, after the workers drain |
| `multi.go:2508` `idx.indexCtxRaw` | `indexRepoRaw` (IndexRepo) | Not held (RLock released before the walk, as in 04-18) |
| `multi.go:2769` `incrementalReindexPathsWithReceiptMode` | `incrementalReindexRepoRawMode` (IncrementalReindexRepo*) | Not held |
| `multi.go:3040` `idx.indexCtxRaw` | `trackRepoSourceCtx` | Not held; the publication Lock comes after the walk |
| `multi.go:3222/3229/3251/3263/3268/3274/3280` | `ReconcileRepoCtx` | Not held; the publication Lock comes after |
| `intake_manifest.go`, `admission.go`, `incremental_watcher_batch.go`, `watcher.go:971/1040/1648/1723`, `watcher_path_filter.go:21` | as in 04-18 | Not held (files untouched by this plan) |
| **new** `multi.go:2262` `DeclarePendingRoots(declared)` | `indexMultiRepo` <- `IndexScoped` (L2055) <- `IndexAll` (L2020) / external callers | Not held. `resolveTrackPrefix` releases its RLock (L2871/L2873) inside the resolve loop. The declaration precedes `batchMutationGate.RLock` (L2267). The deferred release runs after the gate RUnlock and after the publication Unlock (L2372) |
| **new** `cmd/gortex/daemon_state.go:644` `DeclarePendingRoots(repos)` | `warmupDaemonState` inside the `RunRepositoryTopologyBatch` callback | Not held (`mi.mu` is unexported; `cmd/gortex` cannot hold it). It runs under the `batchMutationGate` writer and topology writer. gate -> `mi.mu` is the established order (indexMultiRepo and trackRepoSourceCtx publication). Every `batchMutationGate` acquirer was checked (`incremental_derived.go:144`, `multi.go:351/507/531/1599/1633/2267`, `repository_mutation_coordinator.go:167/284`, `repository_untrack.go:92`, `repository_topology_batch.go:68`): none holds `mi.mu`, so no inverse edge exists |

**Result: no caller holds `mi.mu` when the predicate or a declaration runs, and the new `mi.mu.Lock` spans do map work only. No deviation to report.** The full package passes under `-race`.

## Decisions Made

- The deferred release is placed right after the declaration in `indexMultiRepo`, so it also covers the lane-acquisition error path and panics. Defers run LIFO, so the release runs after `batchMutationGate.RUnlock`, which needs no particular order.
- `DeclarePendingRoots` builds its root list before taking `mi.mu`, so `filepath.Abs` (a possible `os.Getwd`) never runs under the lock.

## Deviations from Plan

### Acceptance-criterion scope note (no code change)

**1. `grep -rn 'trackedRepoRoots' internal/ cmd/` is not empty**
- **Found during:** Task 1 acceptance check
- **Issue:** the grep matches `internal/mcp/server.go:1550/1552/1590`, which is `func (s *Server) trackedRepoRoots()`. That is an unrelated upstream MCP-server method (introduced in `49a15756`, "match MCP session cwd to tracked repos case-insensitively"), not the removed Indexer field.
- **Resolution:** left untouched. It is a different symbol, outside `files_modified`, and in the package 04-19 just changed. The intent of the criterion holds: `grep -rn trackedRepoRoots internal/indexer cmd/` prints nothing, and no comment names the removed field.

Otherwise, the plan was executed exactly as written.

## Issues Encountered

- `gofmt -l internal/indexer/` lists `internal/indexer/file_delta.go`. That issue predates this plan and the file is not in it, so it was not touched (out of scope).
- The Task 2 verify's `go build ./cmd/gortex/` rewrites the repo-root `gortex` binary, which is gitignored (`/gortex`). No tracked file was affected.

## Flagged assumptions (carried from the plan, confirmed)

- Until 04-21, untracking a nested root makes alpha admit beta's files again, but nothing reindexes them automatically. `UntrackStopsExcluding` therefore calls `IncrementalReindexRepo` explicitly.
- `trackNestedLater` still produces alpha/nested duplicates, because IndexScoped declares only alpha. Its doc comment now says so without referring to a config seed. MCP `outer_indexed_first` is unaffected, and `TestSourceScope_` passes.

## Next Phase Readiness

- 04-21 can add enclosing-repository reconcile on track and untrack on top of live pruning. Pruning has no copy left to refresh.
- SEC-08 and ID-09 are deliberately not marked Complete. 04-21 and 04-22 still claim them, and the phase re-verification decides.

## Self-Check: PASSED

- FOUND: internal/indexer/multi.go (DeclarePendingRoots at L2143, declaration at L2262)
- FOUND: internal/indexer/workspace_resolve.go (pendingRoots read in nestedTrackedRootContains)
- FOUND: internal/indexer/indexer.go (inNestedTrackedRoot delegates to the owner)
- FOUND: internal/indexer/nested_tracked_root_test.go (7 TestNestedTrackedRoot_ functions)
- FOUND: cmd/gortex/daemon_state.go (DeclarePendingRoots(repos) count = 1)
- FOUND: 6b299de3 (ancestor of HEAD)
- FOUND: 13d9d1eb (ancestor of HEAD)
