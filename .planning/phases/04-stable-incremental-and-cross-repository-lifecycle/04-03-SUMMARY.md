---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 03
subsystem: graph
tags: [scope, projection, neo4j, sqlite, security, fail-closed]

requires:
  - phase: 01
    provides: "Fail-closed ProjectionScope validation in store_sqlite.OpenScopedProjectionSnapshot and neo4jprojection.NormalizeRequest"
provides:
  - "graph.Scope (Workspace, Project, Repositories): one fail-closed scope type (D-16)"
  - "graph.NewScope validator: trims, rejects an empty set and a blank entry, dedupes, sorts"
  - "(Scope).RequireWorkspaceProject, (Scope).Allows, (Scope).AllowsRepo: never admit an empty prefix or the zero Scope"
  - "type ProjectionScope = Scope, so every projection consumer compiles unchanged"
affects: [04-06 reconcile pass, 04-10 cross-surface acceptance, 04-12 source read/write gate, phases 7-8 AI context and claim storage]

actuals:
  tokens: 3299
  tasks: 1
  commits: 2
plan_head_before: 054582deeec0ae198fb84e728c1ab036eaea5ca6

tech-stack:
  added: []
  patterns:
    - "Build scope values with graph.NewScope; check membership with Scope.Allows/AllowsRepo (fail-closed, unlike repoNarrowAdmits and QueryOptions.ScopeAllows)"

key-files:
  created:
    - internal/graph/scope.go
    - internal/graph/scope_test.go
    - internal/neo4jprojection/normalize_scope_test.go
  modified:
    - internal/graph/scoped_projection.go
    - internal/graph/store_sqlite/scoped_projection.go
    - internal/neo4jprojection/service.go

key-decisions:
  - "04-03: graph.Scope is the one fail-closed scope type (D-16); ProjectionScope is an alias. Allows/AllowsRepo reject an empty RepoPrefix, the zero Scope, and (when set) a WorkspaceID/ProjectID mismatch, with no RepoPrefix fallback for the workspace."
  - "04-03: neo4jprojection.NormalizeRequest no longer drops blank repository entries. A blank entry fails closed with 'neo4j projection scope: scope: repository allow-set contains an empty value'; the combined missing-field error is unchanged."

patterns-established:
  - "Scope validation: graph.NewScope, then RequireWorkspaceProject only where the surface is keyed by workspace and project (projection)"

requirements-completed: [SEC-01]

coverage:
  - id: D1
    description: "graph.Scope with NewScope (trim, empty-set and blank-entry rejection, dedupe, sort) and RequireWorkspaceProject"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/graph/scope_test.go#TestScope_NewScope"
        status: pass
      - kind: unit
        ref: "internal/graph/scope_test.go#TestScope_RequireWorkspaceProject"
        status: pass
    human_judgment: false
  - id: D2
    description: "Scope.Allows and Scope.AllowsRepo never admit a nil node, an empty prefix, an out-of-set repository, or the zero Scope, and enforce workspace and project when set"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/graph/scope_test.go#TestScope_Allows"
        status: pass
      - kind: unit
        ref: "internal/graph/scope_test.go#TestScope_AllowsRepo"
        status: pass
    human_judgment: false
  - id: D3
    description: "OpenScopedProjectionSnapshot validates through graph.NewScope and keeps its existing error text"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/graph/store_sqlite/scoped_projection_test.go#TestScopedProjectionTracer"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/graph/store_sqlite/ -run ScopedProjection"
        status: pass
    human_judgment: false
  - id: D4
    description: "NormalizeRequest builds request.Scope with graph.NewScope; a blank repository entry fails closed and duplicates are deduped"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/normalize_scope_test.go#TestNormalizeRequest_BlankRepositoryFailsClosed"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/neo4jprojection/ && GOWORK=off go test -race -count=1 ./internal/mcp/ -run Neo4j"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 03: Shared Fail-Closed graph.Scope Summary

**Plan 03 adds `graph.Scope`, one fail-closed scope type (`NewScope`, `RequireWorkspaceProject`, `Allows`, `AllowsRepo`). The SQLite projection snapshot and the Neo4j request normalizer now validate through it, and a blank repository entry in a projection request is now rejected instead of silently dropped.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-10-13T01:39:34Z
- **Completed:** 2026-10-13T01:44:57Z
- **Tasks:** 1 (TDD: RED + GREEN commits, no refactor needed)
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments

- `internal/graph/scope.go` holds the D-16 scope type. The validation body moved here from `OpenScopedProjectionSnapshot`, and the error text moved with it, under a `scope:` prefix.
- `Allows`/`AllowsRepo` reject what the fail-open predicates admit: an empty `RepoPrefix` and an empty allow-set. When `Workspace` or `Project` is set, a node's `WorkspaceID`/`ProjectID` must match it exactly.
- `ProjectionScope` is now `= Scope`, so no projection consumer had to change.
- `OpenScopedProjectionSnapshot` keeps the substrings `repository allow-set is required`, `repository allow-set contains an empty value`, and `workspace and project are required` (now `scoped projection: scope: ...`).
- `NormalizeRequest` fails closed on a blank repository entry (T-04-12). It no longer trims the caller's `Repositories` slice in place. The owner key still sees the deduped, sorted list.

## Task Commits

1. **Task 1 RED: failing tests for the shared scope**: `c632b6ac` (test)
2. **Task 1 GREEN: graph.Scope and the two call sites**: `17bb0ce7` (feat)

REFACTOR: none needed.

## Files Created/Modified

- `internal/graph/scope.go`: `Scope`, `NewScope`, `RequireWorkspaceProject`, `AllowsRepo` (binary search on the sorted set), `Allows`
- `internal/graph/scope_test.go`: `TestScope_NewScope`, `TestScope_RequireWorkspaceProject`, `TestScope_Allows`, `TestScope_AllowsRepo`
- `internal/graph/scoped_projection.go`: `type ProjectionScope = Scope`
- `internal/graph/store_sqlite/scoped_projection.go`: `graph.NewScope` + `RequireWorkspaceProject`, wrapped as `scoped projection: %w`; the descriptor uses the validated Scope; the unused `sort` import is removed
- `internal/neo4jprojection/service.go`: `graph.NewScope` after the combined missing-field check, wrapped as `neo4j projection scope: %w`; `request.Scope = scope`
- `internal/neo4jprojection/normalize_scope_test.go`: `TestNormalizeRequest_BlankRepositoryFailsClosed`

## Decisions Made

- `Allows` compares `WorkspaceID`/`ProjectID` exactly. Unlike `LocalizationNodeScope.Allows` and `QueryOptions.ScopeAllows`, it does not fall back to `RepoPrefix` for an empty `WorkspaceID`. The plan specifies strict equality, and a fallback would weaken the fail-closed contract.
- `NewScope` copies its input, never mutates it, and does not require workspace or project. Only projection calls `RequireWorkspaceProject`, so the source gate (04-12) can decide its own rule.
- `LocalizationNodeScope` was left alone. It is a fail-open query-time filter with different semantics (kinds, tests, file exclusions), so it is not a duplicate of `Scope`.

## Deviations from Plan

None. The plan executed as written.

## TDD Gate Compliance

- RED `c632b6ac` precedes GREEN `17bb0ce7`.
- RED for `TestNormalizeRequest_BlankRepositoryFailsClosed` was an intentional assertion failure: `normalize_scope_test.go:16: NormalizeRequest with a blank repository entry succeeded, want an error`.
- RED for the four `TestScope_*` tests was a build failure (`undefined: Scope`, `NewScope`), which is unavoidable in Go for a brand-new API.
- `gsd-tools check tdd-red-evidence` could not classify the run: it parses only Node TAP summaries and reports `zero_tests_discovered` for `go test` output. `workflow.tdd_mode` is `false`, so the gate is advisory here. The Go output above is the RED evidence.

## Issues Encountered

- The shell is zsh, so an unquoted `$F` file list did not word-split. The first `gofmt -w` silently did nothing, and the gofmt `test -z` passed for the wrong reason. I reran with the files listed explicitly: `gofmt -l` prints nothing, and the exact verify command passes under `bash -c`.

## Verification

- The plan verify command (run under `bash -c`) passes, printing `VERIFY_PASS`: all four `TestScope_*` names are in the `-list` output; `-race` passes for `^TestScope_`, store_sqlite `ScopedProjection` (including `TestScopedProjectionTracer`), the full `internal/neo4jprojection` package, and `internal/mcp -run Neo4j` (`TestNeo4jScope` still rejects `["", " "]`); gofmt is clean.
- Extra regression runs: the full `./internal/graph/` package passes, and `./cmd/gortex/ -run Neo4j|neo4j` passes. `go build ./...` and `go vet` on the touched packages are clean.
- Acceptance greps: `func NewScope(` (1 line), `type ProjectionScope = Scope` (1 line), and `graph.NewScope(` (1 line in each call-site file).

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 04-12 (source read/write gate), 04-06 (reconcile pass), and 04-10 (cross-surface acceptance) can consume `graph.Scope` / `NewScope` / `Allows` directly.
- Phases 7 and 8 (AI context assembly, claim storage) must reuse `graph.Scope` for SEC-01.

## Self-Check: PASSED

- FOUND: internal/graph/scope.go, internal/graph/scope_test.go, internal/neo4jprojection/normalize_scope_test.go
- FOUND: c632b6ac, 17bb0ce7

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*
