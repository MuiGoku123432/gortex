---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
plan: 01
subsystem: parser
tags: [cobol, tree-sitter, go-modules, grammar-attestation, fork, provenance]
requires:
  - phase: 01
    provides: completed Neo4j projection milestone foundation; Phase 2 parser-readiness decisions D-01..D-05, D-09, D-20
provides:
  - "Pushed fork commit f9eaf99c34a92db6f5487d316cf9977295a62a7d on origin branch gsd/v0.26.0-estate-parse-recovery (the value Plan 02-02 pins)"
  - "tsadapter.AttestEmbedded() (Identity, error): compiled-grammar attestation with no fork checkout on disk"
  - "preprocessor.NewEmbeddedAnalyzer(catalog transform.Catalog) (Analyzer, error)"
  - "preprocessor.NewContentID(original []byte) ContentID, identical to the handoff OriginalContentID"
affects: [02-02, 02-06, cobol-extractor, parser-provenance]
actuals:
  tokens: 2140
  tasks: 2
  commits: 1
plan_head_before: f19029f11cd0e4000e5e443c2b17a9383be17ee1
commits_repo: "fork (tree-sitter-cobol-upgrade); Gortex repo has no code commits for this plan"
tech-stack:
  added: []
  patterns: [attestation step injected as a func into shared build-info/ToolIdentity validation, verbatim mismatch error string shared by checkout and embedded paths]
key-files:
  created: []
  modified:
    - $COBOL_UPGRADE_ROOT/preprocessor/internal/tsadapter/parser.go
    - $COBOL_UPGRADE_ROOT/preprocessor/internal/tsadapter/parser_test.go
    - $COBOL_UPGRADE_ROOT/preprocessor/pipeline.go
    - $COBOL_UPGRADE_ROOT/preprocessor/analyze.go
    - $COBOL_UPGRADE_ROOT/preprocessor/analyze_test.go
key-decisions:
  - "Fork commit f9eaf99c34a92db6f5487d316cf9977295a62a7d (parent f19029f, floor 97ac9f1 is an ancestor) is pushed to origin and is the commit Plan 02-02 pins by pseudo-version."
  - "AttestEmbedded reports the generated expectedCheckoutArtifactID as the checkout identity because go.sum pins the shim module bytes; the compiled-language comparison is kept in full."
  - "dependencyVersion is unchanged, so ToolID and CompiledLanguageID match the checkout path; Gortex records the replacement module path and pseudo-version itself (prov_parser_module, Plan 02-02)."
patterns-established:
  - "The embedded and checkout attestation paths share one validation helper and differ only in the attestation func passed in."
requirements-completed: [BASE-01, BASE-02, BASE-03]
coverage:
  - id: D1
    description: "AttestEmbedded matches the checkout Attest identity and rejects an unavailable COBOL language pointer with a zero Identity"
    requirement: BASE-02
    verification:
      - kind: unit
        ref: "preprocessor/internal/tsadapter/parser_test.go#TestAttestEmbeddedMatchesCheckout|TestAttestEmbeddedRejectsUnavailableLanguage (go test -race)"
        status: pass
    human_judgment: false
  - id: D2
    description: "NewEmbeddedAnalyzer produces a green handoff whose ToolIdentity equals validateSelectedShim(repoRoot)"
    requirement: BASE-03
    verification:
      - kind: unit
        ref: "preprocessor/analyze_test.go#TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer|TestTracerAnalyzeDocument (go test -race)"
        status: pass
    human_judgment: false
  - id: D3
    description: "NewContentID equals the OriginalContentID Analyze records and changes when the bytes change"
    requirement: BASE-03
    verification:
      - kind: unit
        ref: "preprocessor/analyze_test.go#TestNewContentIDMatchesHandoff (go test -race)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Commit f9eaf99 is a fast-forward from f19029f on origin gsd/v0.26.0-estate-parse-recovery, touching only the five planned files"
    requirement: BASE-01
    verification:
      - kind: other
        ref: "git ls-remote origin refs/heads/gsd/v0.26.0-estate-parse-recovery == HEAD; parent == f19029f; merge-base --is-ancestor 97ac9f1 HEAD"
        status: pass
    human_judgment: false
duration: 30min
completed: 2026-09-30
status: complete
---

# Phase 2 Plan 01: Embedded Grammar Attestation in the Fork Preprocessor Summary

**The fork's `preprocessor` module can now attest the linked COBOL grammar against its compiled-in identity with no checkout on disk (`AttestEmbedded`, `NewEmbeddedAnalyzer`), and exposes `NewContentID` for copybook revision identity. It ships as pushed commit `f9eaf99c34a92db6f5487d316cf9977295a62a7d`.**

## Pinned Commit (for Plan 02-02)

| Field | Value |
|-------|-------|
| Repository | `github.com/MuiGoku123432/tree-sitter-cobol-upgrade` |
| Branch | `gsd/v0.26.0-estate-parse-recovery` (on origin) |
| **Full commit hash** | **`f9eaf99c34a92db6f5487d316cf9977295a62a7d`** |
| Parent (accepted v0.26.0 line) | `f19029f11cd0e4000e5e443c2b17a9383be17ee1` |
| Behavioral floor (ancestor) | `97ac9f1` |
| Committer time (UTC) | 2026-09-30 21:54:33 |
| Expected pseudo-version | `v0.0.0-20260930215433-f9eaf99c34a9`. Plan 02-02 must confirm it with `go list -m -json`. |

## Performance

- **Duration:** about 30 min, including the wait for the human push
- **Started:** 2026-09-30T21:49Z
- **Completed:** 2026-09-30T22:20Z
- **Tasks:** 2/2
- **Files modified:** 5 (fork repo)

## Accomplishments

- Added `tsadapter.AttestEmbedded()`. It compares `CompiledIdentity()` with the generated `expectedCompiledLanguageID` and returns the verbatim `compiled forest language identity mismatch` error when they differ. When the language pointer is unavailable it returns an error with a zero Identity.
- Moved the build-info checks and the `ToolIdentity` assembly into one shared helper in `pipeline.go`. `validateSelectedShim(repoRoot)` works exactly as before. The new `validateEmbeddedShim()` swaps in only the attestation call.
- Added `preprocessor.NewEmbeddedAnalyzer(catalog)`, which mirrors `NewAnalyzer`, and `preprocessor.NewContentID(original)`, which wraps `contract.NewContentID`.
- Pushed the commit to origin as a fast-forward. No force push was used.

## Task Commits

1. **Task 1: Add embedded grammar attestation and a content-identity wrapper**: `f9eaf99c34a92db6f5487d316cf9977295a62a7d` (feat, fork repo) `feat(preprocessor): add embedded grammar attestation for module consumers`
2. **Task 2: Push the fork commit to origin**: no commit. The user ran `git push origin gsd/v0.26.0-estate-parse-recovery`. The pre-push estate guard passed (3 checks), commit separability passed, and the push went `f19029f..f9eaf99`. The re-run verify printed `VERIFIED f9eaf99c34a92db6f5487d316cf9977295a62a7d`.

## Files Created/Modified (fork repo, `$COBOL_UPGRADE_ROOT`)

- `preprocessor/internal/tsadapter/parser.go`: adds `AttestEmbedded()`.
- `preprocessor/internal/tsadapter/parser_test.go`: adds `TestAttestEmbeddedMatchesCheckout` and `TestAttestEmbeddedRejectsUnavailableLanguage`.
- `preprocessor/pipeline.go`: adds the shared validation helper and `validateEmbeddedShim()`.
- `preprocessor/analyze.go`: adds `NewEmbeddedAnalyzer` and `NewContentID`.
- `preprocessor/analyze_test.go`: adds `TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer` and `TestNewContentIDMatchesHandoff`. Both use the invented `tracerProgram()`.

`expected_identity.go`, `Attest`, `CompiledIdentity`, `CheckoutIdentity`, and `dependencyVersion` are unchanged.

## Decisions Made

- The embedded path drops only the on-disk checkout hash read. The compiled-language identity comparison is not relaxed (BASE-02 prohibition).
- `dependencyVersion` / `ForestModuleVersion` are left unchanged so `ToolID` stays stable for existing fork run envelopes.

## TDD Gate Compliance

- **RED run 1:** compile failure from undefined symbols, which the plan allows.
- **RED run 2:** temporary wrong-result stubs, never committed, produced assertion-level RED. `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for `TestAttestEmbeddedMatchesCheckout`, `TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer`, and `TestNewContentIDMatchesHandoff`. `TestAttestEmbeddedRejectsUnavailableLanguage` passed against the erroring stub, which is expected for a rejection test. Evidence is in the session scratchpad (`red_*.json`, `red_tap.txt`).
- **GREEN:** all four tests pass under `-race`. So do `TestTracerAnalyzeDocument` and the whole tsadapter package. This was re-confirmed at the pushed commit during continuation.
- **Plan-mandated gate exception:** there is one `feat` commit and no separate `test(...)` commit. Task 1 verify 2 requires HEAD's only parent to be `f19029f` and a fixed commit message, so a RED test commit followed by a GREEN commit would fail the plan's own verification.

## Deviations from Plan

### Verify-command issue (not a code deviation)

**1. Task 1 verify 2 is locale-sensitive**
- **Found during:** Task 1
- **Issue:** The expected file-list string assumes C-locale `sort` order (`analyze.go` before `analyze_test.go`). Under the default UTF-8 locale, `sort` orders the names differently, so the check fails even though the commit is correct.
- **Resolution:** Verify 2 passes under `LC_ALL=C`. No code change was made. Future plans that compare a sorted file list should pin `LC_ALL=C`.

---

**Total deviations:** 0 code deviations; 1 verify-command note.
**Impact on plan:** None. The commit touches exactly the five planned files.

## Issues Encountered

- The unfiltered `./preprocessor/` suite ran for more than 10 minutes during planning, so the plan's anchored test selectors were used.

## User Setup Required

None. The one human step, the push, is done.

## Next Phase Readiness

- Plan 02-02 can resolve the pseudo-version for `f9eaf99c34a92db6f5487d316cf9977295a62a7d` with `GOWORK=off` and `GOPRIVATE` covering the fork, pin `preprocessor` plus the `forest-shim/cobol` replace, and call `NewEmbeddedAnalyzer` and `NewContentID`.

## Self-Check: PASSED

- FOUND: fork commit f9eaf99c34a92db6f5487d316cf9977295a62a7d (local HEAD and origin branch head)
- FOUND: parent f19029f11cd0e4000e5e443c2b17a9383be17ee1; 97ac9f1 is an ancestor
- FOUND: exactly the five planned files in the commit; no tracked changes left in the fork
- PASS: tsadapter package and the three anchored preprocessor tests under `-race`; `go vet` clean
