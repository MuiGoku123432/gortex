---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 22
subsystem: mcp
tags: [source-scope, sec-01, sec-02, wr-05, nested-checkout, filesystem-walk, multi-repo, d-13, d-20]
status: complete

requires:
  - phase: 04-19
    provides: source-surface audit and structural inventory naming the move/inline/generate_skill walks
  - phase: 04-17
    provides: requirePathSourceScope physical-owner gate
provides:
  - "handleMoveSymbol and handleInlineSymbol module walks gate every entry with requirePathSourceScope(ctx, node.RepoPrefix, p): refused directories pruned (SkipDir), refused Go files skipped"
  - "handleGenerateSkill gates every WalkDir entry against the walk root's repository (RepoForFile(absDir), overridden by the request's view checkout); refused directories pruned, refused files counted in skipped"
  - "findUseSiteMatches regex branch: GrepRegexpForRepos(useSite, pathPrefix, nil, limit), scope filter, then cap at limit (WR-05)"
  - "Test helper newNestedGoModuleServer (alpha Go module, beta tracked at alpha/nested calling a.Foo())"
  - "TestMoveSymbol_NestedTrackedRootNotRewritten, TestInlineSymbol_NestedTrackedRootNotRewritten, TestGenerateSkill_NestedTrackedRootExcluded, TestFindDeclaration_RegexFiltersBeforeLimit"
affects: [phase-04-verification, source-scope, refactor-tools, generate-skill, find-declaration]

actuals:
  tokens: 3454
  tasks: 3
  commits: 3
plan_head_before: 450e248fcdd5b42010df99cce7546c8847867703
plan_head_after: b23a688f39d205a044579929357e2d8b206c71d9

tech-stack:
  added: []
  patterns:
    - "A filesystem walk under a gated root gates each entry by its physical owner, not only the root: requirePathSourceScope(ctx, <root repo>, entry), SkipDir on a refused directory"

key-files:
  created: []
  modified:
    - internal/mcp/tools_move_inline.go
    - internal/mcp/tools_move_inline_test.go
    - internal/mcp/tools_generate_skill.go
    - internal/mcp/tools_generate_skill_test.go
    - internal/mcp/tools_find_declaration.go
    - internal/mcp/tools_find_declaration_test.go

key-decisions:
  - "Skip, do not refuse: an out-of-scope walk entry is silently skipped, matching the edge loop's existing continue on a resolveNodePath error; whether a scoped refactor should refuse is the pending WR-04/IN-06 policy question"
  - "generate_skill's root repository is RepoForFile(absDir), overridden by the request's view checkout when it contains absDir (the same attribution requirePathSourceScope uses), so a worktree-view request is not refused wholesale"
  - "WR-05 keeps the scan cost: GrepRegexpForRepos still caps each repository at limit; only the global cap that ran before the scope filter is gone"

patterns-established:
  - "Per-entry physical-owner gate inside filepath.Walk/WalkDir callbacks (three call sites, one existing function, nothing extracted)"

requirements-completed: []  # SEC-01 / SEC-02 deliberately NOT marked: phase re-verification decides

duration: ~15min
completed: 2026-10-13
---

# Phase 04 Plan 22: Per-entry source gate for the move/inline/generate_skill walks Summary

**move_symbol, inline_symbol and generate_skill now gate every filesystem-walk entry by its physical owner via requirePathSourceScope, so an alpha-scoped request never rewrites or bundles a nested tracked repository's files; find_declaration's regex branch filters by scope before capping (WR-05).**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-10-13T22:13Z
- **Completed:** 2026-10-13
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- move_symbol's and inline_symbol's module-walk fallback no longer pulls in Go files from a tracked root nested under the module (or a symlinked file). RED showed beta's `use.go` rewritten to `b.Foo()` by move and inlined to `return 42` by inline, and `beta/use.go` named in `touched`. GREEN: beta byte-identical, alpha's `c/c.go` still rewritten or inlined.
- generate_skill's directory walk no longer lists or copies `nested/use.go` into an alpha-scoped bundle. RED listed it in `references`.
- find_declaration's regex use-site search: with two repositories and limit 1, the beta-scoped search returned 0 matches (RED). Now each scope returns exactly its own match.

## Task Commits

1. **Task 1 (tracer): move_symbol walk gate.** `068014d8` `fix(04-22): gate each move_symbol module-walk entry by its physical owner`. Tracer gate: the interactive end-of-phase mode with an automated-only `<verify>` was re-run and passed, so expansion continued.
2. **Task 2: inline_symbol and generate_skill walk gates.** `09a4ff59` `fix(04-22): gate inline_symbol and generate_skill walk entries by their physical owner`
3. **Task 3: WR-05 regex filter-before-limit.** `b23a688f` `fix(04-22): filter find_declaration regex use sites by scope before capping (WR-05)`

The RED and GREEN steps of each TDD task were committed together. RED was observed and recorded before each implementation.

## Files Created/Modified

- `internal/mcp/tools_move_inline.go`: per-entry gate in both module-walk callbacks (4 call sites, `node.RepoPrefix`).
- `internal/mcp/tools_generate_skill.go`: root repository computed once, then a per-entry gate in the WalkDir callback.
- `internal/mcp/tools_find_declaration.go`: regex branch uses GrepRegexpForRepos and breaks at limit after `allowed()`.
- `internal/mcp/tools_move_inline_test.go`: newNestedGoModuleServer plus the move and inline nested tests.
- `internal/mcp/tools_generate_skill_test.go`: TestGenerateSkill_NestedTrackedRootExcluded.
- `internal/mcp/tools_find_declaration_test.go`: TestFindDeclaration_RegexFiltersBeforeLimit.

## Decisions Made

See `key-decisions` in the frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] generate_skill test directory spelling**
- **Found during:** Task 2 (RED run)
- **Issue:** The plan's `directory: "alpha"` is refused by resolveFilePath ("does not start with a known repo prefix"), so the test could not reach the walk.
- **Fix:** The test passes `directory: "alpha/"`, which resolves to alpha's root.
- **Files modified:** internal/mcp/tools_generate_skill_test.go
- **Commit:** 09a4ff59

**2. [Rule 2 - Correctness] generate_skill root repository honors the request's view checkout**
- **Found during:** Task 2
- **Issue:** The plan computes the root repository with `RepoForFile(absDir)` alone. For a worktree-view request whose checkout sits outside every tracked root, that returns "". `requireSourceScope(ctx, "")` refuses in multi-repo mode, so every entry would have been dropped.
- **Fix:** When the request's view checkout contains absDir, it overrides the repository. This is the same attribution requirePathSourceScope applies to the physical owner.
- **Files modified:** internal/mcp/tools_generate_skill.go
- **Commit:** 09a4ff59

**3. [Minor] Stronger inline positive assertion**
- The plan asked the executor to confirm the inlined text from the RED run. RED showed `return 42`, and the inline test now asserts that alpha's `c/c.go` contains it.

## Verification

- Task 1 verify: `-run '^TestMoveSymbol_|^TestInlineSymbol_|^TestSourceScope_'` under `-race` passed; gofmt clean.
- Task 2 verify: `-run '^TestMoveSymbol_|^TestInlineSymbol_|^TestGenerateSkill|^TestSourceScope_'` under `-race` passed, including the 04-19 inventory guard; gofmt clean.
- Task 3 verify: `-run '^TestFindDeclaration|^TestSourceScope_'` under `-race` passed; gofmt clean.
- Acceptance greps: move_inline `requirePathSourceScope(ctx, node.RepoPrefix` = 4 (≥2); generate_skill `requirePathSourceScope(` = 2 (≥1); find_declaration `GrepRegexpForRepos(` = 1.
- `GOWORK=off go build ./... && GOWORK=off go vet ./internal/mcp/` passed.
- Full package `GOWORK=off go test -race -count=1 ./internal/mcp/ -timeout 45m`: `ok github.com/zzet/gortex/internal/mcp 513.085s`
- `git diff --name-only 450e248f..HEAD` lists exactly the six `files_modified`.

## Issues Encountered

None.

## Known Stubs

None.

## Threat Flags

None. No new surface: the change only narrows what three existing walks admit and which already-scanned matches survive.

## Next Phase Readiness

- Mitigations T-04-77, T-04-78 and T-04-79 are in place with invariant tests. SEC-01 and SEC-02 are left for phase re-verification to mark.
- Follow-ups (deferred by plan):
  - a shared multi-repository test-server builder (newTwoRepoServer, newNestedCheckoutServer, newNestedGoModuleServer);
  - the refuse-vs-skip policy for scoped refactors (WR-04/IN-06).

## Self-Check: PASSED

- All six modified files exist.
- Commits 068014d8, 09a4ff59 and b23a688f are ancestors of HEAD.
- The ledger count (3) matches the commits listed.
