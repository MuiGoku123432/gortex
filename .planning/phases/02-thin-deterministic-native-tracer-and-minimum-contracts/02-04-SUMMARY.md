---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
plan: 04
subsystem: indexer
tags: [cobol, provenance, git, vcs, neo4j, projection, tdd]
requires:
  - phase: 02-02
    provides: "prov_* Meta contract with prov_revision_content_id on COBOL file/program/copybook nodes, and the TestCobolTracer_* acceptance harness"
  - phase: 02-03
    provides: "Containment-path program IDs and copybook identity (the tracer's single DEMOPGM ID is unchanged)"
provides:
  - "Indexer.stampSourceRevision: prov_vcs_commit (HEAD) or prov_vcs_commit_absence on every node carrying prov_revision_content_id"
  - "classifySourceRevision: git_unavailable / not_a_git_worktree / no_head_commit / not_under_version_control / working_tree_differs_from_head, from gitstate.NewDirtySampler porcelain plus ls-files --error-unmatch"
  - "One additive call in applyCoverageDomains (D-18) covering the bulk, watcher-delta, and incremental index paths"
  - "TestSourceRevisionStamp (10 subtests over temp git repositories)"
  - "Tracer acceptance asserts prov_vcs_commit equals the fixture HEAD on the program (live, reopened, MCP, CLI) and the reopened file node"
  - "TestProjectNode_CobolProvenance: every contract key projects to its own unescaped meta_{key} Neo4j property with zero secret and zero unsupported warnings"
affects: [02-05, 02-06, phase-4-invalidation, neo4j-projection]
actuals:
  tokens: 4223
  tasks: 2
  commits: 3
plan_head_before: 9719925714a8269b44bb953b754d766cf43cceca
tech-stack:
  added: []
  patterns:
    - "Indexer-side provenance stamp keyed only on a Meta string, so it needs no import of the COBOL fork and costs other languages one loop"
    - "Per-call git sampling (accuracy over reuse) bounded by a 30s context"
key-files:
  created:
    - internal/indexer/source_revision.go
    - internal/indexer/source_revision_test.go
  modified:
    - internal/indexer/indexer.go
    - cmd/gortex/cobol_tracer_test.go
    - internal/neo4jprojection/properties_test.go
key-decisions:
  - "Git is sampled on every stamp call rather than cached per bulk pass: the snapshot reflects the worktree when that file is stamped, and only files carrying prov_revision_content_id pay for it."
  - "ls-files runs under --literal-pathspecs with the path after --, so a COBOL file name containing glob or pathspec-magic characters cannot match a different tracked file and earn a commit it does not have (T-02-16, T-02-17)."
  - "A context timeout during rev-parse or ls-files reports git_unavailable, not not_a_git_worktree or not_under_version_control, so a slow git never produces a false claim about the repository."
  - "The stamp deletes the opposite key when it writes one, so a node can never carry both prov_vcs_commit and prov_vcs_commit_absence even if it is stamped twice."
requirements-completed: [PROV-04, PROV-06]
coverage:
  - id: D1
    description: "A committed, unmodified file gets prov_vcs_commit equal to git rev-parse HEAD on the file and program nodes, even when another file in the worktree is dirty"
    requirement: PROV-04
    verification:
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionStamp/clean_committed (go test -race)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Modified and staged-only files get working_tree_differs_from_head; untracked and ignored files get not_under_version_control; never a commit"
    requirement: PROV-04
    verification:
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionStamp/{modified,staged_new,untracked,ignored} (go test -race)"
        status: pass
    human_judgment: false
  - id: D3
    description: "An unborn repository gives no_head_commit, a plain directory gives not_a_git_worktree, and a missing git binary gives git_unavailable"
    requirement: PROV-04
    verification:
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionStamp/{unborn,not_git,git_unavailable} (go test -race)"
        status: pass
    human_judgment: false
  - id: D4
    description: "An index root that is a symlinked subdirectory of the worktree still maps the file to its worktree path and gets the commit; nodes without prov_revision_content_id are untouched even with git unavailable"
    requirement: PROV-04
    verification:
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionStamp/{subdirectory_root,untouched} (go test -race)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The persisted DEMOPGM program and its file node carry prov_vcs_commit equal to the fixture HEAD through the real indexer, AddBatch, SQLite reopen, MCP get_symbol, and CLI get_symbol"
    requirement: PROV-04
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_* (5 tests, go test -race)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every prov_* contract key, plus cobol_kind, projects into Neo4j as its own meta_{key} property under its unchanged name, with values round-tripping and zero secret or unsupported warnings"
    requirement: PROV-06
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/properties_test.go#TestProjectNode_CobolProvenance (go test -race)"
        status: pass
    human_judgment: false
duration: 5min
completed: 2026-09-30
status: complete
---

# Phase 2 Plan 04: VCS Revision Stamp and Neo4j Provenance Projection Summary

**Every COBOL node that carries a content revision now also records either the HEAD commit or one of five exact absence reasons. The indexer writes this through a single approved hook in `applyCoverageDomains`, using the gitstate porcelain sampler plus an index check. A projection test shows that every prov_* key reaches Neo4j as its own property, with no escaping and no filtering.**

## Performance

- **Duration:** about 5 min of wall-clock execution after context loading
- **Started:** 2026-09-30T23:29:15Z
- **Completed:** 2026-09-30T23:34Z
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- Added `internal/indexer/source_revision.go`. `stampSourceRevision` returns after one loop unless some node carries `prov_revision_content_id`. It then classifies the file once and stamps every revision-carrying node. `classifySourceRevision` checks, in this order:
  1. git on `PATH`
  2. the symlink-resolved root and `rev-parse --show-toplevel`
  3. `gitstate.NewDirtySampler(top).Sample`
  4. an empty `HeadCommit`
  5. a dirty entry for the worktree-relative slash path (untracked → `not_under_version_control`; anything else → `working_tree_differs_from_head`)
  6. `ls-files --error-unmatch` (ignored or otherwise unindexed → `not_under_version_control`)
- Added exactly one line to `internal/indexer/indexer.go`: `idx.stampSourceRevision(relPath, result)`, directly after `entrypoints.Detect`. The diff against 3238e524 is one added line and no removed lines. `stampSourceRevision` has exactly one caller.
- The tracer acceptance now captures the fixture commit with `gitCommitHash(root)`. The shared helper requires `prov_vcs_commit` to equal that commit with no absence key. The reopened file node is checked the same way.
- Added `TestProjectNode_CobolProvenance`. It covers the 28 program-applicable keys, including `prov_vcs_commit`, and the 4 file-only keys, using string, bool, int, float64, and `[]string` values.

## Task Commits

1. **Task 1 RED: failing VCS revision stamp tests**: `bfcf402a` (test)
2. **Task 1 GREEN: stamp implementation and the applyCoverageDomains hook**: `681ac17a` (feat)
3. **Task 2: tracer VCS assertions and the Neo4j provenance projection test**: `e60cfe59` (test)

No REFACTOR commit was needed.

## Files Created/Modified

- `internal/indexer/source_revision.go`: `stampSourceRevision` and `classifySourceRevision`. No build tag, and no import of the fork.
- `internal/indexer/source_revision_test.go`: `TestSourceRevisionStamp` and its small file-local helpers. It reuses the package's existing `gitInitRepo`, `runGit`, `writeFile`, and `newTestIndexer`.
- `internal/indexer/indexer.go`: the one D-18 call.
- `cmd/gortex/cobol_tracer_test.go`: a `commit` field on `cobolTracerRepo`, a `commit` parameter on `assertCobolTracerProgram`, and file-node VCS checks after the reopen.
- `internal/neo4jprojection/properties_test.go`: `TestProjectNode_CobolProvenance`.

## Decisions Made

- **Sample git on every call.** The alternative was a per-pass cache. With per-call sampling, the stamp shows git state at the moment each file is stamped, and there is no cache to reset between the bulk and incremental paths. Each COBOL file costs three git processes: `rev-parse`, `status`, and `ls-files`. Other languages pay nothing.
- **Quarantined or timed-out files get no stamp on the bulk path.** They skip `applyCoverageDomains`, so they never reach the hook. This is consistent, because such files produce no program node either.
- **Use literal pathspecs and treat timeouts as `git_unavailable`.** Both are hardening beyond the plan's wording, and both prevent false claims. The reasons are in key-decisions above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Security] `--literal-pathspecs` on the ls-files check**
- **Found during:** Task 1
- **Issue:** Git pathspec rules apply to the path even after `--`. A file name containing `*`, `?`, `[`, or a leading `:` could match a different tracked file. The untracked file would then pass the index check and be given a commit (T-02-16).
- **Fix:** Pass `--literal-pathspecs` as a global git option before `ls-files`.
- **Files modified:** `internal/indexer/source_revision.go`
- **Commit:** `681ac17a`

**2. [Rule 1 - Bug] A timeout was being misreported as a repository fact**
- **Found during:** Task 1
- **Issue:** Under the plan's literal mapping, a `rev-parse` failure meant `not_a_git_worktree` and an `ls-files` failure meant `not_under_version_control`. A 30-second timeout would therefore have recorded a false fact about the repository.
- **Fix:** When `ctx.Err()` is set after either call, report `git_unavailable`.
- **Files modified:** `internal/indexer/source_revision.go`
- **Commit:** `681ac17a`

**3. [TDD] The RED commit includes a no-op placeholder method**
- **Found during:** Task 1 RED
- **Issue:** Without the method, the test file does not compile. The tdd.md rules classify a compile failure as INVALID_RED.
- **Fix:** The RED commit includes an empty `stampSourceRevision`, so RED failed on assertions: 9 of 10 subtests failed with expected-value mismatches. `untouched` passed at RED, as it should, because a no-op stamp already leaves unrelated nodes alone. GREEN replaced the placeholder.
- **Commit:** `bfcf402a`, then `681ac17a`

## TDD Gate Compliance

- The RED commit is `bfcf402a` (`test(02-04)`). Every failure was an assertion failure on the target test `TestSourceRevisionStamp`, never a compile, load, or zero-test result. The failing subtests were `clean_committed`, `modified`, `staged_new`, `untracked`, `ignored`, `unborn`, `not_git`, `git_unavailable`, and `subdirectory_root`. Each got `<nil>` where it expected a commit or an absence reason.
- The GREEN commit is `681ac17a` (`feat(02-04)`). All 10 subtests pass under `-race`, and so do the existing `TestStrip*` tests.
- `gsd_run check tdd-red-evidence` parses Node TAP output, and this project uses `go test`. RED was therefore evidenced by hand, as in 02-03.

## Verification

- The Task 1 automated command passed. It checks that the exact test is listed, all 10 subtests PASS under `-race` alongside `TestStrip*`, `indexer.go` differs from 3238e524 by exactly one added line and no removed lines, the hook line is present, and gofmt is clean.
- The Task 2 automated command passed. It checks that the exact projection test is listed, that it passes under `-race` together with `TestProjectionProperties` and `TestProjectionSecretFiltering`, that all five `TestCobolTracer_*` tests pass under `-race`, that the tracer file asserts `prov_vcs_commit`, and that gofmt is clean.
- `grep -c 'prov_' internal/neo4jprojection/properties_test.go` returns 31, above the minimum of 25.
- `GOWORK=off go build ./...` and `go vet ./internal/indexer/` are clean.
- The repo has no other COBOL fixtures (`*.cbl`, `*.cob`, `*.cpy`), so no other indexer test can observe the new Meta keys.

## Issues Encountered

None.

## Known Stubs

None. The RED placeholder was replaced in the GREEN commit.

## Next Phase Readiness

- PROV-04 is complete: content revision plus either a VCS commit or an explicit absence reason. PROV-06 is proven at the projection layer.
- Stores that already exist need a full reindex to gain `prov_vcs_*` keys, because the incremental path only restamps changed files. Phase 4 invalidation should treat a HEAD move as making `prov_vcs_commit` stale on unchanged files. Nothing restamps a clean file whose bytes did not change when HEAD advances.

## Self-Check: PASSED

- FOUND: internal/indexer/source_revision.go, internal/indexer/source_revision_test.go, internal/indexer/indexer.go, cmd/gortex/cobol_tracer_test.go, internal/neo4jprojection/properties_test.go
- FOUND commits: bfcf402a, 681ac17a, e60cfe59
