---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 12
subsystem: api
tags: [mcp, security, scope, multi-repo, sec-02, d-13, d-20]

requires:
  - phase: 04-03
    provides: graph.Scope (NewScope, AllowsRepo) shared scope type (D-16)
provides:
  - requireSourceScope fail-closed source gate for multi-repository sessions (D-13, D-20)
  - withSourceScopeRequest / sourceScopeRequestFromContext request-in-ctx seam from wrapToolHandlerMode
  - errSourceScopeRequired (narrowing guidance) and errSourceOutOfScope (not-found parity) refusals
  - resolveConfinedFilePath (ungated anchoring + confinement) for graph-key-only callers
affects: [04-10 cross-surface acceptance, phase 07 AI context assembly, phase 08 claim storage]

actuals:
  tokens: 6434
  tasks: 1
  commits: 2
plan_head_before: 1cd324b759f404d9c1a779015e13fef823abf5ee

tech-stack:
  added: []
  patterns:
    - "Source-byte choke points (resolveFilePath, resolveNodePath, resolveGraphPath, committed-view reads) carry the scope gate; per-tool gates only where a handler would otherwise swallow the refusal"
    - "Out-of-scope refusals reuse the missing-file / missing-symbol text so existence is not probeable"

key-files:
  created:
    - internal/mcp/source_scope.go
    - internal/mcp/source_scope_test.go
  modified:
    - internal/mcp/overlay.go
    - internal/mcp/tools_fileops.go
    - internal/mcp/tools_coding.go
    - internal/mcp/change_contract.go
    - internal/mcp/tools_fileops_worktree_test.go

key-decisions:
  - "Source gate resolves scope with IntentReach, not IntentLocate: a bound session keeps its whole workspace for source as before; only the unbound fallback changes"
  - "A resolved scope with an empty RepoAllow but a WorkspaceID (workspace arg, workspace-bound session, server default workspace) expands to that workspace's repositories, so the workspace narrowing the refusal text recommends actually works"
  - "No stored request in ctx is treated as an empty request (no explicit narrowing), so a bound session still resolves its own scope when a handler is invoked outside the middleware"
  - "The gate is a thin resolveFilePath wrapper over resolveConfinedFilePath; graphRelPath and workspaceEditVerificationFiles use the confined resolver because they only derive graph keys / repository ownership and never touch bytes"
  - "rename_symbol refuses the whole rename when any reference site is gated, rather than writing a partial cross-repository rename"

patterns-established:
  - "requireSourceScope(ctx, repoPrefix) is the single source-access rule; phases 7 and 8 must reuse it with graph.Scope (SEC-01)"

requirements-completed: [SEC-02, SEC-01]

coverage:
  - id: D1
    description: "Unbound multi-repo source reads refused with narrowing guidance and no other-repository names or bytes (read_file, get_symbol_source, batch_symbols, get_editing_context, smart_context, export_context)"
    requirement: SEC-02
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_UnboundMultiRepoRefused"
        status: pass
    human_judgment: false
  - id: D2
    description: "Narrowed or bound requests read in-scope source; out-of-scope file and symbol requests answer like missing ones"
    requirement: SEC-02
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_NarrowedAllowed"
        status: pass
    human_judgment: false
  - id: D3
    description: "Unbound multi-repo writes (edit_file, write_file, rename_symbol) refused with on-disk bytes unchanged"
    requirement: SEC-02
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_WriteToolsGated"
        status: pass
    human_judgment: false
  - id: D4
    description: "Single tracked repository: gate is a no-op"
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_SingleRepoNoop"
        status: pass
    human_judgment: false
  - id: D5
    description: "Graph-metadata tools keep the all-repositories default for unbound sessions"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_MetadataUnchanged"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/mcp/"
        status: pass
    human_judgment: false

duration: 19min
completed: 2026-10-12
status: complete
---

# Phase 04 Plan 12: Fail-Closed Multi-Repository Source Gate Summary

**`requireSourceScope` refuses any source read or write that has no resolved scope while several repositories are tracked. A repository outside the scope gets the same answer as a missing file or symbol. Membership is decided by `graph.NewScope` plus `AllowsRepo`.**

## Performance

- **Duration:** 19 min
- **Started:** 2026-10-13T02:14:36Z
- **Completed:** 2026-10-13T02:34:07Z
- **Tasks:** 1 (TDD: RED + GREEN, no refactor needed)
- **Files modified:** 7

## Accomplishments
- With more than one tracked repository, an unbound request is refused by every tool that returns source bytes. The refusal says to pass repo, project, or workspace and names no repository. Graph-metadata tools are unchanged.
- The same gate covers every write path: `resolveFilePath` (edit_file, write_file and all other callers) and `resolveGraphPath` (rename_symbol, scaffold).
- Out-of-scope reads answer `no such file or directory` before anything is checked on disk, so an existing out-of-scope file looks the same as a missing one. Out-of-scope symbols (get_symbol_source, rename_symbol) answer `symbol not found: <id>`.
- With one tracked repository, or with no MultiIndexer, the gate does nothing.

## TDD Gates

- **RED** `9ff5b49b`: the five `TestSourceScope_*` tests. Running `go test ./internal/mcp/ -run '^TestSourceScope_' -v` before any implementation, the behavior assertions failed for the planned reasons:
  - Unbound read_file returned `{"content":"package main\n\nfunc AlphaThing() {}\n",...}`.
  - get_symbol_source, batch_symbols, get_editing_context, smart_context and export_context all returned source.
  - `beta/main.go` with `repo: alpha` was served.
  - Unbound edit_file reported `"disk_status":"committed"`.
  - SingleRepoNoop and MetadataUnchanged passed, because they pin behavior that must not change.
- **GREEN** `ac45e19f`: all five pass, and the whole `internal/mcp` package passes under `-race` (439s).
- **REFACTOR:** none needed. The two repeated gate snippets appear twice each, which is under the rule of three.

## Task Commits

1. **Task 1 RED: failing tests for the source gate**: `9ff5b49b` (test)
2. **Task 1 GREEN: fail-closed source gate**: `ac45e19f` (feat)

## Files Created/Modified
- `internal/mcp/source_scope.go`: `requireSourceScope`, `errSourceScopeRequired`, `errSourceOutOfScope`, and the request-in-ctx key (`withSourceScopeRequest` / `sourceScopeRequestFromContext`).
- `internal/mcp/source_scope_test.go`: the five `TestSourceScope_*` tests, run through `srv.wrapToolHandler`.
- `internal/mcp/overlay.go`: `wrapToolHandlerMode` stores the request in ctx after view-selector stripping and parameter reconciliation (one line).
- `internal/mcp/tools_fileops.go`:
  - `resolveFilePath` is now a gated wrapper over `resolveConfinedFilePath`.
  - Gates added to `resolveNodePath`, `resolveGraphPath` and the read_file committed-view read.
  - `graphRelPath` uses the confined resolver.
- `internal/mcp/tools_coding.go`: gates in get_symbol_source (with symbol-not-found parity), the get_editing_context `compress_bodies` read, the batch_symbols `include_source` read, the smart_context source embed, and rename_symbol (target parity plus a whole-rename refusal).
- `internal/mcp/change_contract.go`: `workspaceEditVerificationFiles` uses the confined resolver. It only derives repository ownership and never reads bytes.
- `internal/mcp/tools_fileops_worktree_test.go`: three linked-worktree write tests now pass an explicit `repo` through the middleware.

## Decisions Made
See `key-decisions` in the frontmatter. The two with the most effect:
- **IntentReach instead of IntentLocate.** The plan named Locate, but Locate would have narrowed a bound session to its own repository for source. That would have broken workspace-wide reads that bound sessions have today.
- **A workspace counts as a resolved scope.** `resolveScopeForRequest` leaves `RepoAllow` nil when only a workspace is supplied. If the gate had refused that case, the refusal would have told callers to "pass workspace" and then rejected it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Workspace narrowing would have been refused**
- **Found during:** Task 1 (GREEN design)
- **Issue:** `resolveUnboundScope` returns `RepoAllow == nil` when only a `workspace` argument is given, so the planned "empty RepoAllow means refuse" rule would also refuse the very narrowing the error text recommends.
- **Fix:** When `RepoAllow` is empty and the resolved scope has a `WorkspaceID`, the allow-set becomes `multiIndexer.ReposInWorkspace(WorkspaceID)`.
- **Files modified:** internal/mcp/source_scope.go
- **Committed in:** ac45e19f

**2. [Rule 1 - Bug] Bound sessions refused when there is no stored request**
- **Found during:** Task 1 (full-package run: `TestAnalyzeScope_GateShapes_RepoNarrows`, `TestAnalyzeScope_WorkspaceIsolation_GateShapes`)
- **Issue:** The plan said "no request in ctx means no resolved scope". That refused bound sessions whose handlers run without the middleware (for example the analyze AST scan through `resolveNodePath`).
- **Fix:** A missing request is treated as an empty request, so session binding still resolves. An unbound session still resolves to nothing and is refused.
- **Files modified:** internal/mcp/source_scope.go
- **Committed in:** ac45e19f

**3. [Rule 1 - Bug] Metadata path normalization was gated**
- **Found during:** Task 1 (`TestGetFileSummary_MultiRepoRelativePath`, `TestLowerEditSourceRefusesTwoRepoNewFilesActionably`)
- **Issue:** `graphRelPath` (get_file_summary, get_editing_context key lookup) and `workspaceEditVerificationFiles` call `resolveFilePath` only to get a graph key or a repository owner. Gating them broke metadata tools for unbound sessions, which contradicts D-13.
- **Fix:** The gate moved from the deferred block into a thin `resolveFilePath` wrapper over `resolveConfinedFilePath`, which keeps the existing body, symlink confinement included. Those two callers now use the confined resolver. Every other caller is still gated.
- **Files modified:** internal/mcp/tools_fileops.go, internal/mcp/change_contract.go
- **Committed in:** ac45e19f

**4. [Rule 1 - Bug] Linked-worktree view reads were attributed to no repository**
- **Found during:** Task 1 (`TestWorktreePathSelectorReadsFileThroughFacade`, `TestCheckoutSourcePrimitive*`)
- **Issue:** An absolute path inside the request's selected checkout has no tracked prefix, so the gate saw the repository `""` and refused it as out of scope.
- **Fix:** When `relPath` has no tracked prefix and `requestViewPathRoot(ctx).contains(absPath)`, the view's `repoPrefix` is used. This runs after the confinement guard.
- **Files modified:** internal/mcp/tools_fileops.go
- **Committed in:** ac45e19f

**5. [Rule 2 - Missing critical] `resolveGraphPath` was a fourth, ungated source choke point**
- **Found during:** Task 1 (inventory review)
- **Issue:** rename_symbol (via `planRenameWrites`) and scaffold read and write source through `resolveGraphPath`, which the plan did not list. Unbound rename was a D-20 hole. Once the gate was in place, rename's resolve closure also silently skipped refused sites. Under narrowing that could have produced a partial cross-repository rename, and an unbound rename reported a misleading `no_edits`.
- **Fix:**
  - Gated `resolveGraphPath`.
  - rename_symbol records the first gate refusal and returns an error for the whole rename instead of applying the rest.
  - An out-of-scope rename target answers `symbol not found: <id>`.
  - Added rename cases to `TestSourceScope_WriteToolsGated` and `TestSourceScope_NarrowedAllowed`.
- **Files modified:** internal/mcp/tools_fileops.go, internal/mcp/tools_coding.go, internal/mcp/source_scope_test.go
- **Committed in:** ac45e19f

**6. [Rule 3 - Blocking] Existing linked-worktree write tests made unbound two-prefix writes**
- **Found during:** Task 1 (`TestEditFile_LandsInLinkedWorktree`, `TestWriteFile_LandsInLinkedWorktree`, `TestEditFile_MainCheckoutPrefixStillEditsMainCheckout`)
- **Issue:** These tests track a main checkout and its linked worktree as two prefixes and wrote without narrowing. D-20 now refuses that by design.
- **Fix:** They now pass `repo` through the production middleware (`callSourceTool`). What the tests assert is unchanged.
- **Files modified:** internal/mcp/tools_fileops_worktree_test.go
- **Committed in:** ac45e19f

**Other changes made inside the planned scope:**
- **Out-of-scope read parity.** The refusal text is `errSourceOutOfScope` ("...: no such file or directory"), with no path and no repository name. The parity test checks that an existing out-of-scope file and a missing out-of-scope file get identical text. It also checks that this text and an in-scope missing file both report `no such file or directory`; the in-scope message from `os.Stat` includes the absolute path, so the two cannot be byte-identical.
- **Where the tool gates sit.** For get_symbol_source and get_editing_context, the gate runs once, before the branch between the committed view and the disk read. That covers both of the plan's `readViewFile` sites with the node's repository.
- **get_editing_context in the unbound test.** The test passes `compress_bodies: true`, because that is the only mode in which the tool returns source bytes. Without it, the tool returns metadata only and stays ungated.

---

**Total deviations:** 6 auto-fixed (4 bugs, 1 missing critical, 1 blocking)
**Impact on plan:** All six were needed to meet the plan's own truths: metadata unchanged, every source write gated, refusals that reveal nothing. Scope did not grow beyond the source choke points.

## Issues Encountered
- The first full-package run on the initial GREEN code had 10 failures, all explained by deviations 2, 3, 4 and 6. After the fixes, the full `internal/mcp` package passes under `-race`, and `cmd/gortex`, `internal/server`, `internal/serverstack`, `internal/hooks`, `internal/agents/codex` and `internal/mcp/streamable` pass (non-race). The `FAIL ./cmd/gortex` with no named test, reported by earlier plans, did not reproduce.
- `gofmt -l internal/mcp/` lists pre-existing unformatted files that this plan did not touch, such as `manifest.go` and `tools_citation.go`. All files changed in this plan are gofmt-clean.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: info-disclosure (accepted) | internal/mcp/tools_coding.go | When a narrowed rename_symbol targets an in-scope symbol that has references in another repository, it refuses with "some references lie outside the resolved scope". That reveals that outside references exist, though not their names or paths. Refusing the whole rename was chosen over a silent partial rename, which would break correctness. |

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness
- 04-10's cross-surface acceptance can drive `get_symbol_source` through this gate.
- Phases 7 (AI context assembly) and 8 (claim storage) must call `requireSourceScope` and use `graph.Scope` (SEC-01).
- Clients that relied on unbound multi-repo source reads now need to pass `repo`, `project` or `workspace`, or bind a session. The CLI binds a session from its cwd, so it is unaffected.

## Self-Check: PASSED
- FOUND: internal/mcp/source_scope.go, internal/mcp/source_scope_test.go
- FOUND: commits 9ff5b49b (test), ac45e19f (feat)
- Acceptance: `grep -c 'requireSourceScope(' tools_fileops.go` = 4 (at least 3 required); `tools_coding.go` = 5 (at least 2 required); `withSourceScopeRequest(` appears on one line in overlay.go; all five `TestSourceScope_*` names are present in `-list`; `GOWORK=off go test -race -count=1 ./internal/mcp/` exits 0; gofmt is clean on the changed files.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-12*
