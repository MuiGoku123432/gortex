---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 21
subsystem: indexer
tags: [nested-checkout, wr-01, wr-02, ownership, topology-change, multi-repo, lock-order, id-09, sec-08]

requires:
  - phase: 04-20
    provides: live nested-root pruning (registered + declared pending roots), trackNestedLater / requireNestedIndexedOnce fixtures
  - phase: 04-18
    provides: nested-root pruning predicate and its lock-order audit
provides:
  - "(*MultiIndexer).reconcileEnclosingRepos(ctx, root, evicting): scoped IncrementalReindexRepo(outer, [root]) of every strictly-enclosing registered repository, skipped inside a topology batch"
  - "TrackRepoCtx evicts enclosing copies after trackRepoSourceCtx returns"
  - "untrackRepoChecked is a wrapper (capture root, untrackRepoTeardown, admit); UntrackRepo / UntrackRepoChecked / purgeRepoChecked inherit it"
  - "TestNestedTrackedRoot_TrackLaterEvicts, _TrackInsideTopologyBatchCompletes, _UntrackAdmits/{UntrackRepo,UntrackRepoChecked}, _LifecycleTrackAndUntrack"
  - "TestCobolLifecycle_NestedTrackLater, TestCobolLifecycle_NestedUntrack, helper newNestedCobolEnv"
  - "MCP newNestedCheckoutServer(trackBetaLater=true) registers beta through ReconcileRepoCtx"
affects: [phase-04-verification, 04-22, source-scope, ownership-audit, checkout-lifecycle]

actuals:
  tokens: 4080
  tasks: 3
  commits: 4
plan_head_before: ed6b5a1debab903d47f7d5e654602fb2e47163f4
plan_head_after: 0448ac9f1526d37dc46549a9eee2c4870952040e

tech-stack:
  added: []
  patterns:
    - "A topology change is a mutation of every enclosing repository: reconcile the enclosing subtree with the existing scoped reconcile, after the trigger has released all locks, never inside a topology batch"

key-files:
  created: []
  modified:
    - internal/indexer/multi.go
    - internal/indexer/repository_untrack.go
    - internal/indexer/nested_tracked_root_test.go
    - internal/indexer/cobol_lifecycle_crossrepo_test.go
    - internal/mcp/source_scope_test.go

key-decisions:
  - "One existing reconcile serves both directions: IncrementalReindexRepo(outer, [nestedRoot]) purges present-but-excluded copies on track and admits the subtree on untrack; no new eviction code"
  - "The enclosing reconcile runs only after the triggering track/untrack returns (all of mi.mu, lane, batch gate and topology writer released) and returns at once inside RunRepositoryTopologyBatch"
  - "Track-side reconcile runs only when the outer repository's committed FileMtimes holds a key under the nested root (T-04-76); untrack always admits"
  - "ReconcileRepoCtx (warm-restart registration) deliberately does not reconcile enclosing repositories; the warmup converges via 04-20 declared roots and the outer census, and the MCP fixture uses this path to keep exercising the physical-owner gate"

patterns-established:
  - "Thin wrapper over a lock-holding teardown: capture inputs under mi.mu.RLock, run the teardown (whose defers release the gate/writer), then do follow-up mutations"

requirements-completed: []  # ID-09 / SEC-08 deliberately NOT marked: phase re-verification decides

coverage:
  - id: D1
    description: "Tracking a nested repository after its outer one evicts the outer copies in the same TrackRepoCtx call: no alpha/nested node, no alpha text match, clean audit"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_TrackLaterEvicts"
        status: pass
    human_judgment: false
  - id: D2
    description: "COBOL: outer-then-nested track order yields the same graphdump digest as nested-then-outer"
    requirement: ID-09
    verification:
      - kind: unit
        ref: "internal/indexer/cobol_lifecycle_crossrepo_test.go#TestCobolLifecycle_NestedTrackLater"
        status: pass
    human_judgment: false
  - id: D3
    description: "Untracking a nested repository (UntrackRepo, UntrackRepoChecked) admits its files into the enclosing repository in the same call; RepoForFile names alpha; audit clean"
    requirement: ID-09
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_UntrackAdmits"
        status: pass
    human_judgment: false
  - id: D4
    description: "COBOL: after untracking the nested repository the outer graph equals a clean outer-only build holding the nested files"
    requirement: ID-09
    verification:
      - kind: unit
        ref: "internal/indexer/cobol_lifecycle_crossrepo_test.go#TestCobolLifecycle_NestedUntrack"
        status: pass
    human_judgment: false
  - id: D5
    description: "No deadlock: TrackRepoCtx inside RunRepositoryTopologyBatch, and CheckoutLifecycle Register/Untrack (purgeRepoChecked), each finish within a bounded wait under -race"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_TrackInsideTopologyBatchCompletes"
        status: pass
      - kind: unit
        ref: "internal/indexer/nested_tracked_root_test.go#TestNestedTrackedRoot_LifecycleTrackAndUntrack"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol|^TestEmbeddedIndex_|^TestWarmup'"
        status: pass
    human_judgment: false
  - id: D6
    description: "MCP outer_indexed_first still reproduces alpha holding beta's copies (via ReconcileRepoCtx); all sixteen NestedCheckoutRefused sub-subtests pass"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_NestedCheckoutRefused"
        status: pass
    human_judgment: false

duration: 38min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 21: Enclosing-repository reconcile on nested track/untrack Summary

**Tracking or untracking a repository nested inside another one now runs a scoped `IncrementalReindexRepo(outer, [nestedRoot])` on each enclosing repository in the same call. The run happens after the trigger has released every lock and never inside a topology batch, so the outer graph equals a clean build (graphdump-proven for COBOL) and every nested file has exactly one owner.**

## Performance

- **Duration:** ~38 min, including a 995s full `internal/indexer` `-race` run and a 514s full `internal/mcp` `-race` run
- **Started:** 2026-10-13T21:34:14Z
- **Completed:** 2026-10-13T22:11:35Z
- **Tasks:** 3/3, plus one optional Rule 2 test commit
- **Files modified:** 5, exactly the plan's `files_modified`

## Accomplishments

- `reconcileEnclosingRepos(ctx, root, evicting)` in `multi.go`, next to TrackRepoCtx, with a PURPOSE / RATIONALE / KEYWORDS doc:
  - It returns at once for an empty root or when `activeRepositoryTopologyBatch(ctx, mi) != nil`.
  - Under one `mi.mu.RLock` it collects the prefix of every registered repository whose cleaned root strictly contains `root` (`HasPathPrefix` and not `EqualPaths`). When evicting, it keeps a prefix only if `FileMtimes` holds `rel` or `rel/...`.
  - It then unlocks, sorts the prefixes, and calls `IncrementalReindexRepo(prefix, []string{root})` for each one. An error is logged as a Warn and the loop continues.
- `TrackRepoCtx` keeps its `trackRepoSourceCtx(ctx, entry, nil)` call. After a successful non-nil result it reads the new repository's RootPath through `GetMetadata` and runs the evicting reconcile, then returns the original result and error.
- In `repository_untrack.go`, the old body is renamed `untrackRepoTeardown`; only its name changed, and it had no doc comment. The new `untrackRepoChecked`:
  - captures the root under `mi.mu.RLock`, from `mi.repos` or, on a retry, from `pendingRepositoryUntracks[prefix].metadata` (set once at construction under `mi.mu.Lock`, never mutated);
  - runs the teardown;
  - on a nil error, runs the admitting reconcile.
- The MCP fixture now registers beta with `ReconcileRepoCtx(ctx, beta, {"main.go": 1})`. That is the warm-restart path, which reconciles no enclosing repository, so the physical-owner gate is still tested in the duplicate-copy window.

## Task Commits

1. **Task 1 (tracer): tracking a nested repository evicts the outer copies**: `8068f1cf` (fix)
2. **Task 2: untracking a nested repository admits its files into the enclosing repository**: `874d422a` (fix)
3. **Task 3: the MCP nested fixture registers beta through the warm-restart path**: `e9c32528` (test)
4. **Optional Rule 2 (orchestrator note): deadline-bounded `-race` test through CheckoutLifecycle Register/Untrack**: `0448ac9f` (test)

## TDD Evidence

**Task 1 RED** (tests written and run before `multi.go` changed):
- `TestNestedTrackedRoot_TrackLaterEvicts` failed in `requireNestedIndexedOnce`: `Expected nil, but got: &graph.Node{ID:"alpha/nested/main.go::ZebraWidget" ...}`.
- `TestCobolLifecycle_NestedTrackLater` digests differed. The first diff rows were `+ [8] E | lifea/nested/cobol/inner.cbl | lifea/nested/cobol/inner.cbl::INNERPGM | defines ...` and the `lifea/nested/cobol/inner.cbl` file and program nodes.
- `TestNestedTrackedRoot_TrackInsideTopologyBatchCompletes` passed. It is a guard against a deadlock the change could introduce, not a reproduction.

**Task 1 GREEN:** the Task 1 verify command exited 0 (`-race`, 139.9s) and gofmt was clean.

**Task 2 RED:** `TestNestedTrackedRoot_UntrackAdmits/{UntrackRepo,UntrackRepoChecked}` both failed with `Expected value not to be nil` (no alpha/nested node). `TestCobolLifecycle_NestedUntrack` digests differed, with `- [8] E | lifea/nested/cobol/inner.cbl ...` rows missing.

**Task 2 GREEN:** the Task 2 verify command exited 0 (`-race`, 160.0s; covers nested-root, COBOL lifecycle, untrack, apply/preview-untrack and repository-cleanup tests) and gofmt was clean.

**Task 3 RED:** `TestSourceScope_NestedCheckoutRefused/outer_indexed_first` failed at its precondition: `the fixture must reproduce a store indexed before the nested repository was tracked: alpha/nested/main.go::ZebraWidget is missing`. `tracked_together` passed.

**Task 3 GREEN:** all sixteen `--- PASS: TestSourceScope_NestedCheckoutRefused/<state>/<surface>` lines were present, `TestSourceScope_` passed under `-race`, gofmt was clean, and `grep -c 'ReconcileRepoCtx(' internal/mcp/source_scope_test.go` = 1.

## Plan-level verification

| Command | Result |
|---|---|
| `GOWORK=off go build ./... && GOWORK=off go vet ./internal/indexer/ ./internal/mcp/` | pass |
| `GOWORK=off go test -race -count=1 ./internal/indexer/ -timeout 45m` | ok (994.6s) |
| `GOWORK=off go test -race -count=1 ./internal/mcp/ -timeout 45m` | ok (514.0s) |
| `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol\|^TestEmbeddedIndex_\|^TestWarmup' -timeout 40m` | ok (38.4s) |
| `GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestNestedTrackedRoot_LifecycleTrackAndUntrack$'` | ok. The test was added after the full-package run had compiled, so it was run on its own |
| `git diff --name-only ed6b5a1d..HEAD` | exactly the 5 `files_modified` files |

## Lock-order audit (enclosing reconcile)

The invariant (SEC-08 prohibition): `reconcileEnclosingRepos` must never run while its caller holds `mi.mu`, a repository lane, `batchMutationGate` (read or write), the reach topology writer, or CheckoutLifecycle `l.mu`, and must never run inside a repository topology batch.

How the hook sites guarantee it:
- **TrackRepoCtx** (`multi.go:2922`) calls it after `trackRepoSourceCtx` has returned. Inside `trackRepoSourceCtx`, the lane and topology writer are held only inside the `coordinateRepositoryTopologyMutation` callback and released by its defers, and `mi.mu` is taken only in short RLock/Lock spans. Nothing escapes the return.
- **untrackRepoChecked** (`repository_untrack.go:58`) calls it after `untrackRepoTeardown` has returned. The teardown's `defer mi.batchMutationGate.RUnlock()` and `defer finishTopologyMutation(...)` have run by then. The root is captured under an RLock that is released before the teardown starts.
- Lock primitives outside the indexer: `mi.mu`, the lanes, `batchMutationGate` and CheckoutLifecycle `l.mu` are unexported, so `internal/mcp` and `cmd/gortex` cannot hold them. `reach.BeginTopologyMutation` is called only from `internal/indexer` (verified by grep).
- Every CheckoutLifecycle `l.mu` span (`checkout_lifecycle.go` L346-384 setters/getters, L2781-2819 `notifyTrackedSetChanged` / `beginBatch` / `watcher`) is a field read or write that never calls into `mi`.

### TrackRepoCtx callers

| Call site | Caller path | Locks held at the call | Reconcile runs? |
|---|---|---|---|
| `internal/mcp/tools_multi.go:532` | `resolveRepoPrefixOrReconcile`, called from `tools_core.go:1453/1554` MCP handlers | none (MCP request goroutine; indexer locks unexported) | yes |
| `internal/mcp/auto_index.go:64` | `autoIndexCWDBackground` (background goroutine, `context.Background()`) | none | yes |
| `internal/indexer/checkout_promote.go:286` | `indexPromotedCorpus` <- `promoteCheckoutTransition` (L205) <- `executePromotionTransition` (transition worker) | none. `defer l.beginBatch()()` takes `l.mu` only to bump `batchDepth`, then releases it. Catalog calls hold no indexer lock | yes |
| `internal/indexer/checkout_lifecycle.go:461` | `CheckoutLifecycle.Register` | none. Same `beginBatch` pattern; `l.mu` is not held across the call | yes (covered by `TestNestedTrackedRoot_LifecycleTrackAndUntrack`) |
| `internal/indexer/multi.go:2816` | `TrackRepo` (no non-test callers) | none | yes |
| `internal/indexer/multi.go:3230` | `ReconcileRepoCtx` fallback (empty `priorMtimes`) | none. The already-tracked check RLock is released at L3218 before the call | yes, unless ctx is a batch ctx |
| `cmd/gortex/mcp.go:571` | `indexEmbeddedRepo` (embedded single-repo mode) | none (`cmd` cannot hold indexer locks) | yes |
| `cmd/gortex/daemon_state.go:742` | `warmupDaemonState` pool worker, inside `RunRepositoryTopologyBatch` (the batch holds `batchMutationGate.Lock` + topology writer) | batch gate (write) and topology writer, held by the batch owner | **no**: `activeRepositoryTopologyBatch(batchCtx, mi) != nil` returns at once (`TestNestedTrackedRoot_TrackInsideTopologyBatchCompletes`) |

### Untrack callers (all reach `untrackRepoChecked`)

| Call site | Caller path | Locks held at the call | Reconcile runs? |
|---|---|---|---|
| `internal/indexer/multi.go:3570` | `UntrackRepo` (no non-test callers outside `internal/indexer`) | none | yes |
| `internal/indexer/repository_untrack.go:18` | `UntrackRepoChecked` <- `checkout_promote.go:302` (`indexPromotedCorpus` resample) | none (as for L286 above) | yes |
| `internal/indexer/repository_untrack.go:31` | `purgeRepoChecked` <- `checkout_promote.go:376` (`rollbackPromotion` <- `rollbackPromotionOrLog` <- `promoteCheckoutTransition`) | none | yes |
| `internal/indexer/repository_untrack.go:31` | `purgeRepoChecked` <- `checkout_lifecycle.go:2686` `evictRepoChecked`. Its callers are `evictRepo` (`Untrack`/`applyUntrack` L959/L1060, `retireOnReload` L1217/L1227), the reconciler saga hook `ReleaseGraph` (L2652, `reconcile/saga.go:421`), `checkout_untrack.go:358` (demotion after `CommitAuthorizedDemotion`), and `checkout_transition_worker.go:186/217` (demotion transitions) | none. `detachWatcherContext` and the catalog calls return before the purge; `AllMetadata()` in the reload loop returns a copy | yes (covered by `TestNestedTrackedRoot_LifecycleTrackAndUntrack` via `lc.Untrack`) |

**Result: no caller outside a topology batch holds `mi.mu`, a repository lane, `batchMutationGate`, the topology writer, or CheckoutLifecycle `l.mu` when `reconcileEnclosingRepos` runs. The single in-batch caller (daemon warmup) is skipped by the batch check. No deviation to report.**

Lane ordering (the issue 04-18 deferred): the reconcile takes only the outer repository's lane, through `IncrementalReindexRepo` -> `coordinator.reconcile`, and only once the nested repository's lane has been released (track) or closed and drained (untrack). No lane is held across another.

## Decisions Made

- `reconcileEnclosingRepos` takes `root` from the registered metadata (`GetMetadata(result.RepoPrefix).RootPath`), not from `entry.Path`. This gives the same absolute, volume-normalised spelling that `mi.repos` uses for containment.
- The untrack wrapper reconciles only on a nil teardown error. On a pending (failed) teardown the registry entry is already hidden but the lane stays closed, and the retry captures the root from the continuation and admits it then.
- In the MCP fixture, the comment on the config branch was trimmed to its current reason ("so IndexScoped indexes alpha alone"). Its old text cited 04-18's config seed, which 04-20 removed. The config itself is unchanged, as the plan specified.

## Deviations from Plan

### Auto-added (Rule 2, at the orchestrator's request)

**1. [Rule 2 - Safety test] Deadline-bounded `-race` test through the CheckoutLifecycle track and untrack paths**
- **Found during:** post-Task 3. The orchestrator noted that only the in-batch skip had a failing test behind it.
- **Change:** added `TestNestedTrackedRoot_LifecycleTrackAndUntrack`. It runs `Register(outer)`, `Register(inner)` and `Untrack(inner)` through `newLifecycleFixture`, each inside a 2-minute select. It asserts eviction after the inner track, and after the untrack it asserts admission and `RepoForFile = outer`.
- **Files modified:** `internal/indexer/nested_tracked_root_test.go` (already in `files_modified`)
- **Commit:** `0448ac9f`

Otherwise, the plan was executed exactly as written.

## Issues Encountered

- `gofmt -l internal/indexer/` still lists `internal/indexer/file_delta.go`. This predates the plan (noted in 04-20) and the file is outside its scope, so it was left untouched.

## Flagged assumptions (carried from the plan, confirmed)

- The COBOL nested dumps matched without widening the volatile allowlist. No row depended on the root.
- `ReconcileRepoCtx` with `{"main.go": 1}` registered beta without the TrackRepoCtx fallback, and alpha kept its copies, so the `outer_indexed_first` precondition held.
- `IncrementalReindexRepo` ignores ctx. The reconcile honours ctx only through the batch check.

## Next Phase Readiness

- VERIFICATION `gaps[1]` missing items 2 and 3 (untrack admission, track-later eviction) now have passing tests. 04-22 (walk gates, WR-05) is independent of this change.
- ID-09 and SEC-08 are deliberately not marked Complete; the phase re-verification decides.

## Self-Check: PASSED

- All 5 modified files and this SUMMARY exist.
- Commits 8068f1cf, 874d422a, e9c32528, 0448ac9f are ancestors of HEAD; `git rev-list --count ed6b5a1d..HEAD` = 4 before the docs commit.
