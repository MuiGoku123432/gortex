---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 11
subsystem: parser
tags: [cobol, copybook, preprocessor, fork, sourcemap, grading, provenance]

requires:
  - phase: 02-01
    provides: pushed fork commit f9eaf99 (NewEmbeddedAnalyzer, NewContentID) on gsd/v0.26.0-estate-parse-recovery
provides:
  - "Local fork commit e64234cc493c487393c5e7867adfd7ca92d936ba (parent f9eaf99c34a92db6f5487d316cf9977295a62a7d, NOT pushed) on gsd/v0.26.0-estate-parse-recovery"
  - "transform.RuleCopybookWrapper (copybook-wrapper/v1), transform.CopybookConfigurationID(catalog), transform.RunCopybook(original, catalog)"
  - "grade.Policy.AnchorsByConstruction, grade.PolicyCopybookV1 (cobol-grade-policy-copybook-v1), registered in grade.PolicyFor"
  - "preprocessor.Analyzer.AnalyzeCopybook(ctx, sourceID, original) (handoff.Handoff, error)"
affects: [03-12, 03-13, cobol-extractor, copybook-data-items]

actuals:
  tokens: 6445
  tasks: 2
  commits: 1
plan_head_before: f9eaf99c34a92db6f5487d316cf9977295a62a7d
commits_repo: "fork (tree-sitter-cobol-upgrade); Gortex repo has no code commits for this plan"

tech-stack:
  added: []
  patterns:
    - "Leading occurrences: a caller-supplied occurrence list placed ahead of every rule's occurrences before the stable sort, outside the sequence-separator overlap check"
    - "One private build path (Analyzer.build) parameterized by the transform function and grade policy"

key-files:
  created:
    - $COBOL_UPGRADE_ROOT/preprocessor/transform/copybook.go
    - $COBOL_UPGRADE_ROOT/preprocessor/transform/copybook_test.go
  modified:
    - $COBOL_UPGRADE_ROOT/preprocessor/transform/pipeline.go
    - $COBOL_UPGRADE_ROOT/preprocessor/grade/grade.go
    - $COBOL_UPGRADE_ROOT/preprocessor/grade/grade_test.go
    - $COBOL_UPGRADE_ROOT/preprocessor/analyze.go
    - $COBOL_UPGRADE_ROOT/preprocessor/analyze_test.go

key-decisions:
  - "Fork commit e64234cc493c487393c5e7867adfd7ca92d936ba (parent f9eaf99, 97ac9f1 an ancestor) adds copybook analysis mode; it is local only, and Plan 03-12 owns the user-confirmed push."
  - "The copybook final-newline insert is a second RuleCopybookWrapper occurrence, not a second edit in the header occurrence: run emits occurrences in start order, so one occurrence spanning offset 0 and the member end would overtake every other rule's edit."
  - "analyze(ctx, original) keeps its signature (quality_census.go calls it and is outside the seven-file commit); the transform-parameterized step is analyzeWith, and Analyze/AnalyzeCopybook share Analyzer.build."
  - "Variant selection reuses transform's existing isLevelNumber (01-49, 66, 77, 88) and equalASCIIFold for EXEC on the first code line."

patterns-established:
  - "Copybook handoffs are distinguishable from program handoffs by rule ID, ledger reason, configuration ID, and policy version."

requirements-completed: [DET-01]

coverage:
  - id: D1
    description: "RunCopybook wraps a member in a data or procedure header recorded as KindInsert segments, keeps the member's content ID, records CopybookConfigurationID, and appends a final newline insert when needed"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "preprocessor/transform/copybook_test.go#TestRunCopybookDataWrapper"
        status: pass
      - kind: unit
        ref: "preprocessor/transform/copybook_test.go#TestRunCopybookProcedureWrapper"
        status: pass
      - kind: unit
        ref: "preprocessor/transform/copybook_test.go#TestRunCopybookVariantSelection"
        status: pass
    human_judgment: false
  - id: D2
    description: "PolicyCopybookV1 is registered in PolicyFor and skips only missing-required-anchor; PolicyV1 is unchanged"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "preprocessor/grade/grade_test.go#TestPolicyForCopybook"
        status: pass
      - kind: unit
        ref: "preprocessor/grade/grade_test.go#TestAssessCopybookPolicy"
        status: pass
    human_judgment: false
  - id: D3
    description: "AnalyzeCopybook yields validated handoffs with member-true coordinates (data items at rows 0-2 on their level numbers, paragraph at row 0), copybook policy grading, and degraded grades with gaps for REPLACING pseudo-text members"
    requirement: DET-01
    verification:
      - kind: integration
        ref: "preprocessor/analyze_test.go#TestAnalyzeCopybookDataMember"
        status: pass
      - kind: integration
        ref: "preprocessor/analyze_test.go#TestAnalyzeCopybookProcedureMember"
        status: pass
      - kind: integration
        ref: "preprocessor/analyze_test.go#TestAnalyzeCopybookReplacingDegrades"
        status: pass
    human_judgment: false
  - id: D4
    description: "Program mode unchanged: existing grade, transform, handoff suites and the anchored tracer/embedded/content-ID tests pass; commit touches exactly seven files, grammar untouched, nothing pushed"
    requirement: DET-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./preprocessor/grade/ ./preprocessor/transform/ ./preprocessor/handoff/"
        status: pass
      - kind: other
        ref: "Task 2 verify 2 (parent f9eaf99, 97ac9f1 ancestor, seven-file list, clean tree) under LC_ALL=C; ls-remote still f9eaf99"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 11: Copybook Analysis Mode in the Fork Preprocessor Summary

**`Analyzer.AnalyzeCopybook` wraps a standalone copybook member in a synthetic data or procedure header recorded as `copybook-wrapper/v1` KindInsert segments, grades it under the new `cobol-grade-policy-copybook-v1`, and keeps the member's own coordinates and content ID. Committed locally in the fork as `e64234cc493c487393c5e7867adfd7ca92d936ba`, not pushed.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-10-02T09:57:42Z
- **Completed:** 2026-10-02T10:05:30Z
- **Tasks:** 2
- **Files modified:** 7 (fork), plus this SUMMARY in Gortex

## Accomplishments

- `transform.RunCopybook` picks the header variant from the member's first code line. A level number or `EXEC` selects the data header (IDENTIFICATION, PROGRAM-ID CPYWRAP, DATA DIVISION, WORKING-STORAGE SECTION). Anything else selects the procedure header. An empty member gets the data header. The ledger records the choice as the reason `copybook-data-header` or `copybook-procedure-header`.
- `transform.Run` and `ConfigurationID` behave as before. `Run` delegates to an unexported `run(original, catalog, leading, configurationID)`, so program-mode ledgers keep their configuration ID.
- `grade.PolicyCopybookV1` is PolicyV1 plus `AnchorsByConstruction: true`. It skips only the missing-required-anchor check. `handoff.Validate` accepts copybook handoffs because `PolicyFor` knows the new version.
- Member coordinates hold: the data items in the invented 3-line member project exactly (not synthetic) to rows 0, 1, 2 at their level-number bytes, and the paragraph header to row 0.

## Task Commits

The plan requires one fork commit, so Tasks 1 and 2 share it:

1. **Task 1: Copybook wrapper transform and the copybook grade policy**, and
2. **Task 2: Analyzer.AnalyzeCopybook and the local fork commit**: fork commit `e64234cc493c487393c5e7867adfd7ca92d936ba` (feat), parent `f9eaf99c34a92db6f5487d316cf9977295a62a7d`, branch `gsd/v0.26.0-estate-parse-recovery`, **not pushed** (`ls-remote` still shows f9eaf99).

**Plan metadata:** Gortex docs commit (this SUMMARY, STATE.md, ROADMAP.md).

## TDD Evidence

The plan locks one commit, so there are no separate test/feat commits. RED was observed before each implementation step:

- **Task 1 RED:** with declaration-only stubs, all five targets failed on behavior assertions. `TestPolicyForCopybook` failed with "ok = false, want true". `TestAssessCopybookPolicy` got red `[missing-required-anchor]` where green was wanted. `TestRunCopybookDataWrapper` and `TestRunCopybookProcedureWrapper` failed because Generated did not begin with the header. Every `TestRunCopybookVariantSelection` subcase failed with "ledger has no entries". **GREEN:** all five passed, and so did the full grade, transform, and handoff suites under `-race`.
- **Task 2 RED:** with `AnalyzeCopybook` stubbed to `Analyze`, `TestAnalyzeCopybookDataMember` failed (PolicyVersion `cobol-grade-policy-v1`) and `TestAnalyzeCopybookProcedureMember` failed (no paragraph_header observation). `TestAnalyzeCopybookReplacingDegrades` already passed, because program mode also degrades a bare member, so it acts as a regression guard. **GREEN:** all three passed, plus the three existing anchored tests under `-race`.
- The GSD `check tdd-red-evidence` classifier only parses Node TAP output, so it cannot read Go test output. The evidence above is from `go test -v`.

## Files Created/Modified (fork)

- `preprocessor/transform/copybook.go`: RuleCopybookWrapper, the header constants and reasons, CopybookConfigurationID, RunCopybook, and opensWithDataDescription.
- `preprocessor/transform/copybook_test.go`: the three RunCopybook tests (invented members).
- `preprocessor/transform/pipeline.go`: `Run` delegates to `run`. Leading occurrences are prepended after the separator filter and before the stable sort, and the ledger records the configuration ID it is passed.
- `preprocessor/grade/grade.go`: AnchorsByConstruction, PolicyCopybookV1, PolicyFor switch, and the guarded required-anchor check in Assess.
- `preprocessor/grade/grade_test.go`: TestPolicyForCopybook and TestAssessCopybookPolicy.
- `preprocessor/analyze.go`: analyzeWith (transform-parameterized), analyze (program mode), the shared build path, and AnalyzeCopybook.
- `preprocessor/analyze_test.go`: three AnalyzeCopybook tests and one small helper.

## Decisions Made

See key-decisions in the frontmatter. In short: the final-newline insert is its own occurrence, `analyze` keeps its signature for quality_census, and variant selection reuses the existing token helpers.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Kept `analyze(ctx, original)` and added `analyzeWith` instead of changing `analyze`'s signature**
- **Found during:** Task 2
- **Issue:** The plan says `analyze` should take the transform function as a parameter. But `quality_census.go:116,124` calls `analyzer.analyze(ctx, data)`, and that file is outside the seven-file commit that Task 2's verify enforces.
- **Fix:** `analyze` stays as the program-mode entry point and delegates to `analyzeWith(ctx, original, transform.Run)`. `Analyze` and `AnalyzeCopybook` share `build(ctx, sourceID, original, run, policy)`.
- **Files modified:** preprocessor/analyze.go
- **Verification:** Task 2 verify 1 passes, and `go vet ./preprocessor/` is clean.
- **Committed in:** e64234c

**2. [Rule 1 - Bug] The final-newline insert is a separate occurrence, not a second edit of the header occurrence**
- **Found during:** Task 1
- **Issue:** The plan describes one occurrence holding both the offset-0 header insert and the end-of-member newline. `run` emits occurrences in sorted start order and moves its cursor to each edit's end. So one occurrence spanning 0 and len(original) would move the cursor past every other rule's edits, and the Builder would fail with ErrOverlap whenever any other rule fired.
- **Fix:** RunCopybook passes two leading occurrences under RuleCopybookWrapper. The trailing one sorts after every occurrence that starts before the member end.
- **Files modified:** preprocessor/transform/copybook.go
- **Verification:** TestRunCopybookDataWrapper (unterminated case) passes, and so does assertLedgerCoversMap.
- **Committed in:** e64234c

**3. [Verify-command locale] Task 2 verify 2 needs `LC_ALL=C`**
- **Found during:** Task 2
- **Issue:** The expected file-list string assumes C-locale `sort` order. Under this shell's en_US.UTF-8 collation, `analyze_test.go` sorts before `analyze.go`, so the check fails even though the file set is exactly the seven planned files.
- **Fix:** I re-ran the identical command under `LC_ALL=C`, and it passed (TASK2_VERIFY2_PASS). No code change.
- **Committed in:** n/a

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug), plus 1 verify-command locale note.
**Impact on plan:** No scope change. Every planned API, name, and file is as specified.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None. The push of e64234c is Plan 03-12 and needs explicit user confirmation.

## Next Phase Readiness

- Plan 03-12 can push `e64234cc493c487393c5e7867adfd7ca92d936ba` to origin `gsd/v0.26.0-estate-parse-recovery` once the user confirms. Plan 03-13 then re-pins Gortex to the new pseudo-version and calls `AnalyzeCopybook` for `.cpy` members.
- The grammar is unchanged, so the compiled grammar ID Gortex pins stays the same.

## Self-Check: PASSED

- FOUND: preprocessor/transform/copybook.go, copybook_test.go (fork, in commit e64234c)
- FOUND: fork commit e64234cc493c487393c5e7867adfd7ca92d936ba (parent f9eaf99), touching exactly the seven planned files
- NOT PUSHED: ls-remote origin refs/heads/gsd/v0.26.0-estate-parse-recovery = f9eaf99

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
