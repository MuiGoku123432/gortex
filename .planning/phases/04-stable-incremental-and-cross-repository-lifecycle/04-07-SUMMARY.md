---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 07
subsystem: indexer
tags: [cobol, config, allowlist, resolve_from, copybook-library, db2, idms, multi-repo, reload]

requires:
  - phase: 04-14
    provides: reconcile pass honouring ResolveFrom (union ambiguity), ordered CopybookLibraries, default-qualifier alias links, resource GC
  - phase: 04-06
    provides: resolver.CobolPolicy / CobolLibrary / CobolPolicyLookup and SetCobolPolicyLookup
  - phase: 04-01
    provides: COBOL lifecycle harness (newCobolLifecycleEnv, cleanCobolDump, requireSameDump)
provides:
  - "index.cobol per-repository config (CobolConfig, CobolCopybookLibrary, IndexConfig.Cobol) with fail-closed CobolConfig.Validate"
  - "Validation in validateWorkspaceSchema and readWorkspaceConfig (invalid block keeps last good parse or defaults)"
  - "MultiIndexer.cobolPolicyLookup shared by the master and every per-repository resolver; untracked repos dropped with a Warn"
  - "Indexer.SetCobolConfig pushed from RefreshRepoConfigs; Indexer.cobolConfig reader for 04-09's material fingerprint"
  - "TestCobolLifecycle_CrossRepoClean: cross-repository clean-build sequences on the real track path"
affects: [04-08, 04-09, 04-15]

actuals:
  tokens: 7048   # chars/4 over the realized diff (28190 chars)
  tasks: 2
  commits: 4
plan_head_before: 7723a18161a19688c6f4a58b273030c391ee0492

tech-stack:
  added: []
  patterns:
    - "Per-call tracked check: the COBOL policy lookup reads tracking at each call (once per repository per pass), so a long-lived per-repository resolver never holds a stale repository set"
    - "Fail-closed config block: an invalid index.cobol rejects the whole .gortex.yaml read, the same way a malformed file is rejected"

key-files:
  created:
    - internal/config/cobol_config_test.go
    - internal/indexer/cobol_lifecycle_crossrepo_test.go
  modified:
    - internal/config/config.go
    - internal/config/manager.go
    - internal/indexer/indexer.go
    - internal/indexer/multi.go
    - internal/indexer/workspace_resolve.go

key-decisions:
  - "A repository counts as tracked for the COBOL policy when it is registered in mi.repos or holds a live mutation lane (existingRepositoryMutationCoordinator). TrackRepoCtx registers a repository only after its own resolve pass, so a registered-only check drops crossb from crossa's allowlist during crossb's pass, and crossa then crossb loses the binding (verified by experiment)."
  - "Tracking is read per lookup call, not snapshotted when the lookup is built: the per-repository resolver's lookup is installed at Indexer construction and outlives later tracks and untracks."
  - "Library names and default qualifiers are validated as already-normalized cobol-name-norm/1 keys (no lower-case ASCII, quotes, %, :, /, #, space, or control bytes), so the resolver's exact comparison finds the producer's IDs; delimited DB2 schemas cannot be defaults."
  - "With a nil ConfigManager (zero-value fixtures) cobolPolicyLookup returns nil, which means same-repository binding only."

patterns-established:
  - "Config reload into a live per-repository Indexer: RefreshRepoConfigs pushes each reloadable block through a mutex-guarded setter next to SetExcludePatterns"

requirements-completed: [SEC-03, ID-04, ID-09]

coverage:
  - id: D1
    description: "index.cobol config types and fail-closed validation: wildcard/blank resolve_from, empty/absolute/../wildcard library paths, library repos outside resolve_from, and non-normalized names/defaults are rejected with joined errors; global schema validation includes the block"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/config/cobol_config_test.go#TestCobolConfig_Validate"
        status: pass
    human_judgment: false
  - id: D2
    description: "Per-repository .gortex.yaml loads index.cobol; an invalid block keeps the last good parse, and on a first load leaves no policy"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/config/cobol_config_test.go#TestCobolConfig_WorkspaceLoad"
        status: pass
    human_judgment: false
  - id: D3
    description: "Cross-repository clean builds: no binding without an allowlist, exactly-one binding with it, SUBPGM ambiguous across both, wildcard config fails closed, equal digests for opposite track orders"
    requirement: SEC-03
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_crossrepo_test.go#TestCobolLifecycle_CrossRepoClean/(no_allowlist|allowlist|track_order|invalid_config)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Configured copybook library order binds copy/prod with cobol_copy_search_rank 0, and db2_default_schema yields one cobol-default-qualifier/1 aliases edge"
    requirement: ID-04
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_crossrepo_test.go#TestCobolLifecycle_CrossRepoClean/(search_order|default_schema)"
        status: pass
    human_judgment: false
  - id: D5
    description: "RefreshRepoConfigs pushes the reloaded index.cobol block into the live per-repository Indexer"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_crossrepo_test.go#TestCobolLifecycle_CrossRepoClean/refresh_pushes_config"
        status: pass
    human_judgment: false
  - id: D6
    description: "No regression in the config package, the indexer lifecycle/workspace/refresh tests, resolver COBOL tests, and the cmd/gortex COBOL Tracer/Breadth suites"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/config/"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer/ -run '^TestCobolLifecycle_|RefreshRepoConfigs|Workspace'"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/resolver/ -run Cobol"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 07: index.cobol Configuration and Cross-Repository Policy Wiring Summary

**Repositories can now declare `index.cobol` in their `.gortex.yaml`: `resolve_from`, ordered `copybook_libraries`, `db2_default_schema`, and `idms_default_subschema`. The block is validated fail-closed in both the global and per-repository load paths. One tracked-aware policy lookup feeds the master resolver and every per-repository resolver, and `RefreshRepoConfigs` pushes reloads into live indexers. Cross-repository clean builds bind only within the allowlist and give the same digest in either track order.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-10-13T03:40:13Z
- **Completed:** 2026-10-13T03:53:00Z
- **Tasks:** 2 (TDD: RED + GREEN each)
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments
- `CobolConfig` / `CobolCopybookLibrary` under `index.cobol`, with documented D-09/D-10/D-11 semantics. `CobolConfig.Validate` collects every problem and joins them with "; ". `validCobolKey` is shared by library names and both defaults.
- Fail-closed loading:
  - `validateWorkspaceSchema` appends `index.cobol` errors.
  - `readWorkspaceConfig` logs a Warn with the prefix, path, and error, then returns `(nil, false)`. The last good parse stays in force, or the defaults (same-repository only) on a first load.
- `cobolPolicyFromConfig` and `MultiIndexer.cobolPolicyLookup`:
  - Each call reads the repository's merged config.
  - `ResolveFrom` entries and library repositories that are not tracked are dropped, with one Warn listing the prefixes only.
  - The master resolver (`newMasterResolver`) and every per-repository resolver (`newPerRepoIndexerGuardedWithMode`) install it.
- `Indexer.SetCobolConfig` and `cobolConfig` run under `cobolMu`. `RefreshRepoConfigs` calls `SetCobolConfig(cfg.Index.Cobol)` beside `SetExcludePatterns`. A standalone `Indexer` installs a default lookup that honours its own policy only.

## Task Commits

1. **Task 1 RED: index.cobol validation and workspace load tests** - `bd3fd53f` (test)
2. **Task 1 GREEN: fail-closed validation in global and per-repository config** - `8d6fb25e` (feat)
3. **Task 2 RED: cross-repository clean-build tests** - `c3159baa` (test)
4. **Task 2 GREEN: policy lookup wiring and live reload** - `a164679e` (feat)

REFACTOR: none needed.

## TDD Gate Compliance

- **Task 1 RED** (`bd3fd53f`): the first run failed to compile, which is not a valid RED. Minimal scaffolding fixed that: the types, the `IndexConfig.Cobol` field, and a no-op `Validate`. Then `GOWORK=off go test -count=1 -v ./internal/config/ -run 'TestCobolConfig_'` exited 1 with 19 assertion failures:
  - 15 invalid-input Validate subtests, plus `errors_joined` and `global_schema`.
  - `WorkspaceLoad/invalid_keeps_last_good` and `/invalid_first_load`.

  The valid-input guard subtests and `WorkspaceLoad/valid` passed.
- **Task 1 GREEN** (`8d6fb25e`): the verify command exited 0, and the whole config package passes under `-race`.
- **Task 2 RED** (`c3159baa`): the RED commit includes the `cobolMu` field and the `cobolConfig` reader so the test compiles. `GOWORK=off go test -count=1 -v ./internal/indexer/ -run '^TestCobolLifecycle_CrossRepoClean$'` exited 1 with these assertion failures:
  - `allowlist`: XPGM To `unresolved::cobol::program::XPGM` with reason `missing`; XCPY missing; SUBPGM bound to crossa instead of ambiguous.
  - `search_order`: CUSTREC unresolved, with no rank.
  - `default_schema`: 0 alias edges.
  - `refresh_pushes_config`: Cobol block still empty.

  `no_allowlist`, `invalid_config`, and `track_order` passed as guards.
- **Task 2 GREEN** (`a164679e`): all 7 subtests and `TestCobolLifecycle_CleanDeterminism` pass under `-race`, and the full verify command exited 0.

## Files Created/Modified
- `internal/config/config.go`: `IndexConfig.Cobol`, `CobolConfig`, `CobolCopybookLibrary`, `CobolConfig.Validate`, `validCobolKey`, and the `validateWorkspaceSchema` hook.
- `internal/config/manager.go`: `readWorkspaceConfig` validates `index.cobol` after unmarshal and keeps the last good parse on error.
- `internal/config/cobol_config_test.go` (new): `TestCobolConfig_Validate`, `TestCobolConfig_WorkspaceLoad`.
- `internal/indexer/indexer.go`: `cobolMu`, `SetCobolConfig`, `cobolConfig`, and the default lookup in `New`.
- `internal/indexer/multi.go`: the lookup in the master and per-repository resolvers, and `SetCobolConfig` in `RefreshRepoConfigs`.
- `internal/indexer/workspace_resolve.go`: `cobolPolicyFromConfig`, `cobolPolicyLookup`.
- `internal/indexer/cobol_lifecycle_crossrepo_test.go` (new): `TestCobolLifecycle_CrossRepoClean`, with an invented crossa/crossb corpus.

## Decisions Made
See `key-decisions` in the frontmatter. The tracked-repository rule is the one later plans need to know.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The tracked-prefix check is read per call and includes repositories with a live mutation lane**
- **Found during:** Task 2
- **Issue:** The plan said to snapshot the tracked prefixes once when the lookup is built. The per-repository lookup is built when the Indexer is constructed and lives for the Indexer's lifetime, so a construction-time snapshot goes stale. Separately, `TrackRepoCtx` adds a repository to `mi.repos` only after that repository's own resolve pass. So during crossb's pass, crossa's `resolve_from: [crossb]` was dropped as untracked, and tracking crossa then crossb left XPGM and XCPY missing. With the lane check removed as an experiment, the `allowlist` and `track_order` subtests fail.
- **Fix:** The closure checks tracking on each call, which happens once per repository per pass. A repository is tracked when it is in `mi.repos` or `existingRepositoryMutationCoordinator(prefix) != nil`. The `mi.mu.RLock` inside a resolver callback follows the existing `crossWorkspaceLookup` precedent. Untracked prefixes are still dropped with a Warn, and nothing is treated as everything.
- **Files modified:** `internal/indexer/workspace_resolve.go`
- **Commit:** `a164679e`

**2. [Rule 3 - Blocking] RED commits include compile scaffolding**
- **Found during:** Tasks 1 and 2
- **Issue:** Tests against types and methods that did not exist yet failed to compile, which is not a valid RED.
- **Fix:** Each RED commit includes the smallest scaffolding that lets the target tests fail on assertions: types plus a no-op `Validate` in Task 1, and `cobolMu` plus `cobolConfig` in Task 2.
- **Commits:** `bd3fd53f`, `c3159baa`

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking). **Impact:** None on scope. The change to the tracked definition is what makes the allowlist work on the real track path, in either track order.

## Issues Encountered
- `golangci-lint` is not installed locally. `go vet` passes on the touched packages.
- The `cmd/gortex` failure without a named test, seen in an earlier plan, did not recur. The Tracer/Breadth run passed on its own.

## User Setup Required
None. No external service configuration is required.

## Next Phase Readiness
- 04-09 can hash the live `index.cobol` through `Indexer.cobolConfig()`. `RefreshRepoConfigs` keeps it current.
- 04-08 and 04-15 get configured allowlists, library order, and defaults on every reconcile pass, both per-repository and master.

## Self-Check: PASSED
- FOUND: internal/config/cobol_config_test.go, internal/indexer/cobol_lifecycle_crossrepo_test.go, internal/config/config.go, internal/config/manager.go, internal/indexer/indexer.go, internal/indexer/multi.go, internal/indexer/workspace_resolve.go
- FOUND commits: bd3fd53f, 8d6fb25e, c3159baa, a164679e
- Both task verify commands exited 0. The acceptance greps matched:
  - `type CobolConfig struct`: 1 line.
  - `mapstructure:"cobol"`: 1 line.
  - `Cobol.Validate()`: 1 line each in config.go and manager.go.
  - `SetCobolPolicyLookup(mi.cobolPolicyLookup())`: 2 lines in multi.go.
  - `SetCobolConfig(cfg.Index.Cobol)`: 1 line.
- `git diff --name-only` since the plan base lists only the 7 planned files.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*
