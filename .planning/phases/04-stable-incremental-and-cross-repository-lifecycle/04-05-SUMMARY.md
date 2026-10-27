---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 05
subsystem: graph-ownership
tags: [sec-08, d-14, d-15, prefix-diagnostics, sqlite, migration, index-health]

requires:
  - phase: 04-03
    provides: repo-prefix ownership audit (PrefixDiagnostics, SQLite predicates, storetest fence)
  - phase: 04-12
    provides: fail-closed source gate in internal/mcp (same package edited and tested here)
provides:
  - "Strict PrefixDiagnostics.Clean: false whenever any audited node is unowned or misprefixed (D-14)"
  - "graph.OwnershipExemptKinds / OwnershipExemptPathPrefixes / OwnershipGlobalIDNamespaces: the only ownership exemptions"
  - "SQLite ownership predicates generated from the same Go lists"
  - "index_health.repo_ownership.status healthy|unhealthy with unowned_samples whenever unowned > 0 (D-15)"
  - "Schema migration v22 (purge unprefixed solo-repo rows, embedded); currentSchemaVersion = 22"
affects: [04-13, 04-10, phase-06, phase-07]

actuals:
  tokens: 9000    # chars/4 over the realized diff (36,000 chars across 12 files)
  tasks: 2
  commits: 4
plan_head_before: 6df22dfbda47248ec63c671589ed27a0d62c1a5d

tech-stack:
  added: []
  patterns:
    - "One enumerated exemption list per exemption kind in Go; SQL predicates rendered from it once at package init"
    - "Migration purge selects exactly the audit's unowned file-backed population (shared predicate)"

key-files:
  created:
    - internal/mcp/index_health_ownership_test.go
  modified:
    - internal/graph/prefix_diagnostics.go
    - internal/graph/store_sqlite/prefix_diagnostics.go
    - internal/graph/prefix_diagnostics_test.go
    - internal/graph/storetest/storetest.go
    - internal/mcp/tools_enhancements.go
    - internal/mcp/index_health_path_liveness_test.go
    - cmd/gortex/daemon_state.go
    - internal/graph/store_sqlite/schema_version.go
    - internal/graph/store_sqlite/migration_unprefixed_purge.go
    - internal/graph/store_sqlite/migration_unprefixed_purge_test.go
    - internal/graph/store_sqlite/coverage_spelling_purge_test.go

key-decisions:
  - "unresolved:: (graph.UnresolvedMarker) is enumerated in OwnershipGlobalIDNamespaces as the extractor's sentinel namespace; the pinned TestClassifyNodePrefix case already treated it as a global"
  - "purgeUnprefixedRepoRows now deletes only the audit's unowned file-backed rows (repo_prefix='' AND file_path<>'' AND auditable), because v22 runs on current stores where external::-path modules and workspace-scoped rationale nodes legitimately carry an empty repo_prefix"
  - "SQL predicate rendering reuses the package's existing quoteSQLLiteral rather than adding a second quoting helper"

patterns-established:
  - "Ownership exemptions: add a namespace to OwnershipGlobalIDNamespaces with a one-line justification; never a catch-all empty-path rule"

requirements-completed: [SEC-08]

coverage:
  - id: D1
    description: "Strict Clean plus enumerated exemptions, identical on the in-memory and SQLite backends"
    requirement: SEC-08
    verification:
      - kind: unit
        ref: "internal/graph/prefix_diagnostics_test.go#TestPrefixDiagnostics_StrictClean"
        status: pass
      - kind: unit
        ref: "internal/graph/prefix_diagnostics_test.go#TestPrefixDiagnostics_EnumeratedExemptions"
        status: pass
      - kind: integration
        ref: "internal/graph/storetest/storetest.go#testPrefixDiagnostics (TestMemoryStoreConformance, TestSQLiteStoreConformance)"
        status: pass
      - kind: integration
        ref: "internal/indexer/repo_prefix_parity_test.go#TestRepoPrefixParityAfterIndexing"
        status: pass
    human_judgment: false
  - id: D2
    description: "index_health.repo_ownership.status healthy|unhealthy with unowned samples, never failing the call"
    requirement: SEC-08
    verification:
      - kind: integration
        ref: "internal/mcp/index_health_ownership_test.go#TestIndexHealth_RepoOwnershipStatus"
        status: pass
    human_judgment: false
  - id: D3
    description: "Migration v22 purges unprefixed file-backed rows, keeps globals and legitimately empty-prefix rows, idempotent on reopen"
    requirement: SEC-08
    verification:
      - kind: integration
        ref: "internal/graph/store_sqlite/migration_unprefixed_purge_test.go#TestMigrationV22_PurgesUnprefixedRows"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/coverage_spelling_purge_test.go#TestOpenLeavesUnprefixedSyntheticPathsUntouched"
        status: pass
    human_judgment: false
  - id: D4
    description: "Daemon ownership log no longer describes a wholly unowned graph as healthy single-repo mode"
    requirement: SEC-08
    verification: []
    human_judgment: true
    rationale: "Doc-comment and log-message wording change; no test asserts log text. cmd/gortex builds and its tests pass."

duration: 49min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 05: Strict Repository Ownership Summary

**Ownership audit is now strict and visible: `Clean()` fails on any unowned or misprefixed node, every exemption is one named Go list that also generates the SQLite predicates, index_health reports `repo_ownership.status` healthy/unhealthy without blocking queries, and migration v22 purges unprefixed file-backed rows from existing stores.**

## Performance

- **Duration:** 49 min
- **Started:** 2026-10-13T02:36:34Z
- **Completed:** 2026-10-13T03:25:27Z
- **Tasks:** 2
- **Files modified:** 12 (1 created, 11 modified)

## Accomplishments

- `PrefixDiagnostics.Clean` returns `d.UnownedCodeNodes == 0 && d.MisprefixedNodes == 0`. A wholly unowned (single-repo) graph is no longer clean (D-14).
- `OwnershipExemptKinds` (contract, contract_bridge, topic, rationale), `OwnershipExemptPathPrefixes` (external::, external-call::) and `OwnershipGlobalIDNamespaces` (dep::, module::, stdlib::, external::, builtin::, string::, unresolved::) each carry a per-entry justification. An empty-FilePath node is exempt only when it is unowned in a listed namespace, or owned under its own `<RepoPrefix>::`.
- The SQLite `auditableRepoSourceNodePredicate` is rendered once from those lists, so the backends cannot drift. The storetest fence proves both backends count a `mystery::` empty-path node as unowned and ignore an `r1::` synthetic node.
- `index_health.repo_ownership` carries `status`. `unowned_samples` appears whenever `unowned_code_nodes > 0`, and `mixed` is still set for the ghost-graph case. The call never fails because of ownership (D-15).
- `logRepoOwnershipAudit` doc and Error message now state the D-14 rule.
- Migration v22 re-runs `purgeUnprefixedRepoRows`, and `currentSchemaVersion = 22`. The purge deletes exactly the audit's unowned file-backed rows.

## Task Commits

1. **Task 1 RED:** `16ae3817` (test) added failing tests for the strict ownership audit and the enumerated exemptions.
2. **Task 1 GREEN:** `e4a8abac` (feat) made the ownership audit strict, with one enumerated exemption list.
3. **Task 2 RED:** `f1d513c4` (test) added failing tests for the ownership status and migration v22.
4. **Task 2 GREEN:** `be70b2bf` (feat) added the ownership status to index_health, made the daemon log honest, and added migration v22.

RED evidence came from `go test` output, because `gsd-tools check tdd-red-evidence` cannot parse it:
- Task 1: `TestPrefixDiagnostics_StrictClean` failed with `unowned=1 clean=true, want 1/false`. `TestPrefixDiagnostics_EnumeratedExemptions` failed with `ClassifyNodePrefix(mystery::x) = "", want "unowned_code_node"`. The flipped `TestPrefixDiagnosticsMixed` subtest failed with `Clean() = true for a wholly-unowned graph`. The fence failed with `UnownedCodeNodes = 1, want 2` on both `TestMemoryStoreConformance` and `TestSQLiteStoreConformance`.
- Task 2: `TestIndexHealth_RepoOwnershipStatus/healthy` and `/unhealthy_unowned` failed with `expected "healthy"/"unhealthy", actual <nil>`. `TestMigrationV22_PurgesUnprefixedRows` failed with `got 2 ([dep::… main.go::main]), want 1`.

## Files Created/Modified

- `internal/graph/prefix_diagnostics.go`: holds the strict Clean, the three exported exemption lists, and the list-driven `IsAuditableRepoSourceNode` / `IsAuditableRepoSourcePath`.
- `internal/graph/store_sqlite/prefix_diagnostics.go`: the predicates are now package vars built from the Go lists by `buildAuditableRepoSourceNodePredicate`, using the existing `quoteSQLLiteral`.
- `internal/graph/prefix_diagnostics_test.go`: adds the two new tests and the flipped single-repo subtest.
- `internal/graph/storetest/storetest.go`: the fence gains a `mystery::` node (unowned) and an `r1::` synthetic node (no change), and the order-insensitive samples check now expects two entries.
- `internal/mcp/tools_enhancements.go`: adds `repo_ownership.status` and unowned samples whenever any node is unowned.
- `internal/mcp/index_health_ownership_test.go` (new): end-to-end handler test for healthy and unhealthy ownership.
- `internal/mcp/index_health_path_liveness_test.go`: the clean-workspace test now asserts on the liveness message only (see Deviations).
- `cmd/gortex/daemon_state.go`: D-14 wording in the doc comment and Error message.
- `internal/graph/store_sqlite/schema_version.go`: adds step v22 and sets `currentSchemaVersion = 22`.
- `internal/graph/store_sqlite/migration_unprefixed_purge.go`: the purge predicate is now the audit's unowned file-backed population (see Deviations).
- `internal/graph/store_sqlite/migration_unprefixed_purge_test.go`: adds `TestMigrationV22_PurgesUnprefixedRows`, which runs Open twice and pins the survival of external::-path modules and rationale nodes.
- `internal/graph/store_sqlite/coverage_spelling_purge_test.go`: the v19 single-repo test now opens as a v21 binary (see Deviations).

## Decisions Made

- `unresolved::` (`graph.UnresolvedMarker`) is enumerated as a global namespace. The pre-existing pinned case `TestClassifyNodePrefix/"unresolved sentinel is not audited"` already treated it as a global, and `repo_prefix.go` documents it beside `dep::` as a synthetic namespace. Its multi-repo form `<repo>::unresolved::…` is covered by the owned rule.
- The purge shares the audit's predicate instead of keeping its own `file_path <> ''` test, so "what v22 deletes" is by construction "what the audit calls unowned and file-backed".

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The v22 purge would have deleted live, legitimately empty-prefix rows on every current store**
- **Found during:** Task 2 (full `internal/graph/store_sqlite` run)
- **Issue:** `purgeUnprefixedRepoRows` selected `repo_prefix = '' AND file_path <> ''`. That also matches semantic externals and their `module::go:…` nodes under `external::` paths, which made `TestOpenLeavesUnprefixedSyntheticPathsUntouched` fail. It also matches rationale nodes for workspace-scoped memories, which `rationale_projection.go` mints with `RepoPrefix: ""` over `.gortex/rationale`, and contracts whose `c.RepoPrefix` is empty. v7 only ran on pre-v7 stores, but v22 runs on every v21 store, so this would have been real data loss. That contradicts T-04-19 ("keeps globals").
- **Fix:** The purge now uses `unprefixedFileBackedNodePredicate = repo_prefix = '' AND file_path <> '' AND (auditableRepoSourceNodePredicate)` for its edge, vector and node deletes. This narrows v7 to its stated intent: "exactly the file-backed code nodes … never the shared stub namespace". The registry entry for v7 is unchanged.
- **Files modified:** `internal/graph/store_sqlite/migration_unprefixed_purge.go`, `migration_unprefixed_purge_test.go`. The v22 test now also seeds and asserts the survival of an external::-path module and a rationale row.
- **Verification:** `TestOpenV6PurgesUnprefixedSoloRepoRows`, `TestPurgeUnprefixedRepoRowsIsIdempotent`, `TestMigrationV22_PurgesUnprefixedRows`, `TestOpenLeavesUnprefixedSyntheticPathsUntouched` and the whole package pass under `-race`.
- **Committed in:** `be70b2bf`

**2. [Rule 1 - Bug] A v19 test assumed an unprefixed single-repo store survives Open**
- **Found during:** Task 2
- **Issue:** `TestOpenPurgesLegacyCoverageSpellingsSingleRepo` seeds an unprefixed store at v12 and asserts that the native todo node survives Open. Under D-14, v22 purges that row by design, which would hide what v19 alone does.
- **Fix:** The test opens with `openWith(path, 21, <migrations ≤ v21>, false)`, so it still covers v19's unprefixed arm. v22 is covered by its own test.
- **Files modified:** `internal/graph/store_sqlite/coverage_spelling_purge_test.go`
- **Committed in:** `be70b2bf`

**3. [Rule 1 - Bug] A path-liveness test asserted that index_health makes no recommendation at all**
- **Found during:** Task 1 (impact check on `internal/mcp`)
- **Issue:** `TestIndexHealth_PathLivenessCleanWorkspace` uses `setupTestServer`, whose plain `Indexer` produces a wholly unowned graph. Under strict Clean that graph now correctly gets the ownership recommendation.
- **Fix:** The test asserts that the recommendation does not contain the liveness message ("no longer exist on disk"), which is its actual subject.
- **Files modified:** `internal/mcp/index_health_path_liveness_test.go`
- **Committed in:** `e4a8abac`

**4. [Rule 2 - Correctness] Enumerated `unresolved::` in OwnershipGlobalIDNamespaces** (see Decisions). It is named and justified, not a catch-all.

---

**Total deviations:** 4 auto-fixed (3 Rule 1, 1 Rule 2). Three files outside the planned nine were touched: `migration_unprefixed_purge.go`, `coverage_spelling_purge_test.go` and `index_health_path_liveness_test.go`.
**Impact on plan:** Deviation 1 is required for v22 to be safe on any current store. The others keep existing tests truthful under D-14. There is no scope creep.

## Issues Encountered

- The plan's Task 1 verify regex (`-run 'PrefixDiagnostics|Prefix'`) does not select the conformance suites (`TestMemoryStoreConformance` / `TestSQLiteStoreConformance`). The fence was therefore run explicitly with `-run Conformance`, and it passes on both backends under `-race`.
- The full `internal/graph/store_sqlite` package under `-race` takes about 680s, which is over Go's default 10-minute timeout, so the plan's verify needs `-timeout 30m`. This is pre-existing and the same class as the indexer note. With the timeout it passes.
- `gofmt -l internal/graph/store_sqlite/` lists four pre-existing unformatted files that this plan did not touch (`language_gate.go`, `language_gate_test.go`, `proxy_filter_test.go`, `store_generation_write_test.go`). They are out of scope and were left alone.
- `FAIL ./cmd/gortex` without a named test did not occur: `go test ./cmd/gortex/` passed on its own (47s).

## Verification Run

- Task 1 verify: exit 0. Task 2 verify (with `-timeout 30m` on store_sqlite, and `go build -o /dev/null`): exit 0.
- `go test -race ./internal/graph/ ./internal/graph/storetest/`: ok.
- `go test ./internal/mcp/ ./cmd/gortex/` (full packages, no race): ok.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 04-13 can route the embedded `gortex mcp --index` path through `MultiIndexer`. Until then, embedded and `setupTestServer`-style single-repo graphs report `repo_ownership.status: unhealthy`, which is the intended D-14 signal.
- Phases 6 and 7 can gate on `repo_ownership.status`.

## Self-Check: PASSED

- All created and modified files exist on disk.
- Commits `16ae3817`, `e4a8abac`, `f1d513c4` and `be70b2bf` are present in `git log`.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*
