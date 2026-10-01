---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
fixed_at: 2026-10-01T18:15:11Z
review_path: .planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-REVIEW.md
iteration: 1
findings_in_scope: 12
fixed: 9
skipped: 3
status: partial
---

# Phase 2: Code Review Fix Report

**Fixed at:** 2026-10-01T18:15:11Z
**Source review:** `.planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 12 (CR-01, WR-01 to WR-09, FK-01, FK-02)
- Fixed: 9 (CR-01, WR-01, WR-02, WR-03, WR-05, WR-06, WR-07, WR-08, WR-09)
- Skipped: 3 (WR-04 partly: documentation only. FK-01 and FK-02: deferred)
- Out of scope (Info): IN-01 to IN-08, FK-03, FK-04

Where the work ran: every edit, commit, and verification ran in the **main checkout** (`/Users/e1001547-mbp-it/repos/mine/GoApps/gortex`, branch `planning/ai-enhanced-cobol-graph`), not in an isolated worktree. The orchestrator's root-pin guard (`rootpin_bound.sh`) requires the git top level to equal the pinned main checkout and halts otherwise, so a worktree could not be used. No push was made.

Three findings change logic, so a human should confirm them before verification: CR-01, WR-01, and WR-05.

## Fixed Issues

### CR-01: `prov_vcs_commit` is claimed for bytes HEAD does not hold (BOM strip, transforms, TOCTOU)

**Files modified:** `internal/indexer/source_revision.go`, `internal/indexer/source_revision_test.go`, `internal/indexer/indexer.go` (the single D-18 hook call only)
**Commit:** 90cbc134
**Status:** fixed: requires human verification (logic change)
**Applied fix:** `stampSourceRevision` now takes `src`, the exact bytes that were extracted and hashed. The only change in `indexer.go` is that `src` is passed to the existing hook call. `classifySourceRevision` resolves HEAD to one commit and reads that commit's blob ID for the path with `git --literal-pathspecs ls-tree -z <commit> -- <rel>`. It claims the commit only when the git blob ID of `src` matches (SHA-1, or SHA-256 when object IDs are 64 hex digits). On a mismatch it records the new `indexed_bytes_differ_from_file` when `src` is not the file on disk (BOM strip, command transform, file changed after reading), and `working_tree_differs_from_head` otherwise. A path HEAD does not hold falls back to `ls-files --error-unmatch`, which keeps the existing `staged_new` result (`working_tree_differs_from_head`) and the untracked/ignored result (`not_under_version_control`). The full-repo `DirtySampler` scan is gone. New subtests: `bom_stripped` (uses the indexer's real BOM transform), `reverted_after_hashing`, `assume_unchanged`, and `sha256_repository`. The first three fail on the old code.
**Vocabulary change:** added `indexed_bytes_differ_from_file`. No existing reason was removed. Not addressed here: D-10's `prov_start_byte`/`prov_end_byte` are still in post-BOM-strip coordinates for BOM files. That is the coordinate half of CR-01 point 1, and the requested fix covered only the commit claim.

### WR-01: The END PROGRAM lookahead produces wrong containment IDs without flagging them

**Files modified:** `internal/parser/languages/cobol_grammar_identity.go`, `internal/parser/languages/cobol_grammar_test.go`
**Commit:** d1199e31
**Status:** fixed: requires human verification (logic change)
**Applied fix:** Each open stack frame records whether a program was opened inside it. `prov_containment_unresolved` is now set in three cases: an END PROGRAM pop skips frames above its match, the walk pops a frame that has children after its marker is used up, or a frame with children is still open at end of file. IDs are unchanged, but they are no longer reported as resolved. `TestCobolGrammar_StolenEndMarkerUnresolved` runs both reviewer sequences as invented fixtures through the real grammar. Both cases fail on the old code.

### WR-02: END PROGRAM matching is case-sensitive, so legal COBOL loses its nesting

**Files modified:** `internal/parser/languages/cobol_grammar_identity.go`, `internal/parser/languages/cobol_grammar_test.go`
**Commit:** 59f97f57
**Applied fix:** Marker counting and stack matching use the key `strings.ToUpper(strings.Trim(name, "\"'"))`. Symbols still keep the name as written (D-20, and `TestCobolGrammar_NamesVerbatim` still passes). `TestCobolGrammar_EndProgramMatchesCaseInsensitively` covers `OuterPgm` closed by `END PROGRAM OUTERPGM` and the literal `"LitOuter"` closed by `END PROGRAM LITOUTER`. Both fail on the old code. A probe showed the grammar keeps the quotes in `program_name` for a literal name. That partly answers IN-02, which stays out of scope.

### WR-03: Coordinates are emitted for no-image and synthetic projections

**Files modified:** `internal/parser/languages/cobol_grammar.go`, `internal/parser/languages/cobol_grammar_test.go`
**Commit:** 0413fb1f
**Applied fix:** The range mapping moves into the private `cobolStampRange`, which writes coordinates only for a projection that images original bytes. A missing or `NoImage` projection gets `prov_range_absence=no_original_projection`. A wholly `Synthetic` projection gets `synthetic_projection`, a new value. Both get `prov_range_exact=false`. `TestCobolGrammar_RangeNeverInvented` uses synthetic `sourcemap.Projection` values, and its no_image and synthetic cases fail on the old condition.

### WR-05: Revision absence reasons are misclassified under load and in common repo setups

**Files modified:** `internal/indexer/source_revision.go`, `internal/indexer/source_revision_test.go`
**Commit:** 3dde1838
**Status:** fixed: requires human verification (logic change)
**Applied fix:** A single `rev-parse --show-toplevel -q --verify HEAD^{commit}`, run from the file's own directory, replaces the separate top-level and HEAD queries. A stamp now costs 2 git processes, plus `ls-files` only when HEAD lacks the path. Before CR-01 it cost 4, one of them a full-repo `status --untracked-files=all`. Because the query runs from the file's directory, a file inside a submodule resolves to the submodule's own repository and HEAD. Each reason now has one source:
- `not_a_git_worktree` only for git's "not a git repository" answer.
- `no_head_commit` only for exit status 1 with a top level printed.
- `not_under_version_control` only for `ls-files` exit status 1.
- `vcs_query_failed` (new) for timeouts and every other git failure, for example a corrupt repository or a safe.directory refusal.
- `git_unavailable` now means only that no git binary is on PATH.

New subtests `corrupt_repository` and `submodule` both fail on the CR-01 code.
**Vocabulary change:** added `vcs_query_failed`. `git_unavailable` no longer covers timeouts. STATE.md line 112 ("timeouts map to git_unavailable") and 02-04-PLAN.md describe the old mapping. Not done: caching git state once per index pass. That needs state on `Indexer`, which is beyond D-18's single hook.

### WR-06: The fork PAT stays in `~/.gitconfig` for the rest of every job

**Files modified:** `.github/workflows/ci.yml`, `release.yml`, `bench-arm.yml`, `security.yml`, `init-smoke.yml`, `skill-drift.yml`, `publish-claude-plugin.yml`
**Commit:** aa15ba11
**Applied fix:** All 14 credential steps now `export GIT_CONFIG_COUNT/GIT_CONFIG_KEY_0/GIT_CONFIG_VALUE_0` with the insteadOf rewrite and run `go mod download` in the same step. git reads those variables from the environment, so nothing is written to disk. In `publish-claude-plugin.yml` the step runs with `working-directory: gortex`. Later steps, including every third-party action, build from the warm module cache and never see the token. `go mod download -json` confirmed that the no-argument form downloads both fork modules, the shim through its replace. The release job's separate "Download Go modules for the release container" step became redundant and was folded into its credential step. The goreleaser-cross container is unchanged: read-only cache, `GOPROXY=off`, no PAT. `cache: false` is unchanged. Step names stay the same, so the 02-05 structural checker still applies. 02-05-PLAN.md's T-02-24 "accept" note about `~/.gitconfig` is now out of date.

### WR-07: Fork PRs and Dependabot can no longer pass CI

**Files modified:** `.github/workflows/ci.yml`, `.github/dependabot.yml`, `.planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/deferred-items.md`
**Commit:** 53d88fd3
**Applied fix:** Documentation only. A comment at the top of `ci.yml` records that fork PR runs and Dependabot runs never get `COBOL_PARSER_READ_PAT` and fail at the credential step by design (D-13). It also gives the maintainer path for external contributions: review, including `.github/`, then push the branch to this repository. The gomod section of `dependabot.yml` documents the Dependabot secret and the `registries: cobol-fork` entry that would restore Go updates. That entry stays commented out until the user provisions the Dependabot secret, so the config never references a missing secret. The follow-up is an open item in deferred-items.md.

### WR-08: The approved-grammar pin and provenance do not cover the preprocessor module

**Files modified:** `internal/parser/languages/cobol_grammar.go`, `internal/parser/languages/cobol_grammar_test.go`, `cmd/gortex/cobol_tracer_test.go`
**Commit:** 507714dd
**Applied fix:** `init()` reads the build-info entry for `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor`, honouring replace. It fails with `errCobolPreprocessorNotApproved` unless that entry equals the new `cobolApprovedPreprocessorModule` pin (`…/preprocessor@v0.0.0-20260930215433-f9eaf99c34a9`), so no file is parsed under an unapproved producer. Every analyzed node records `prov_preprocessor_module`. Copybooks fail closed under a wrong pin as well. `cobolGrammarExtractorVersion` moves to `gortex-cobol-grammar/2`, and the tracer test asserts the new version and key. New test: `TestCobolGrammar_PreprocessorMismatchFails`. The comment about which pin backs BASE-01 and which backs BASE-02 was corrected. The CI stock-grammar negative step still fails for the attestation reason, because attestation runs before the new check. The step was run locally and passed. Future fork bumps must update this constant together with go.mod.

### WR-09: The `!windows` build tag breaks every non-darwin/linux build

**Files modified:** `internal/parser/languages/cobol_grammar.go`, `cobol_grammar_identity.go`, `cobol_grammar_test.go`, `cmd/gortex/cobol_tracer_test.go`, plus the renamed `cobol_grammar_windows.go` → `cobol_grammar_unsupported.go` and `cobol_grammar_windows_test.go` → `cobol_grammar_unsupported_test.go`
**Commit:** 3334258a
**Applied fix:** The real extractor and its tests use `//go:build darwin || linux`. The stub and its test use `//go:build !darwin && !linux`. The rename was required, not cosmetic: a `_windows.go` suffix is itself a GOOS=windows constraint, so retagging alone would still have left freebsd and other platforms with neither file. The error is now `errCobolParserUnsupportedPlatform`, its text no longer names Windows only, and the test is now `TestCobolGrammarUnsupportedPlatformStub`. A build-tag check (`go list` per GOOS) shows darwin and linux select `cobol_grammar.go` and `cobol_grammar_identity.go`, while windows, freebsd, openbsd, and illumos select only `cobol_grammar_unsupported.go`. With the old tags, freebsd selected the real extractor. The stub could not be compiled for those platforms on this host, because the cgo tree-sitter dependencies need a cross toolchain. `.planning/WINDOWS.md` entry 2 still points at the old test path. It is a gsd-tools ledger and was not edited.

## Skipped Issues

### WR-04: `cobolAnalyzeSlot` neither bounds memory under crash isolation nor stays out of the timeout budget

**File:** `internal/parser/languages/cobol_grammar.go:43-46`
**Reason:** partial. Commit 43c2e807 rewrote the `cobolAnalyzeSlot` comment to state both limits: the slot bounds one process only, so each crash-isolation worker has its own, and queued COBOL files spend their `max_extract_millis` budget while waiting. The behavioural fix was not made. It needs a deadline-aware slot and a context for `Analyze`, but `Extract(filePath, src)` cannot see the budget. The budget timer lives in `internal/indexer/skip_telemetry.go` (`extractWithTimeoutDone` → `parser.Extract`), which is upstream indexer internals beyond D-18's single approved hook. Sending COBOL to one designated worker under crash isolation is also a design decision. Both need a user decision.
**Original issue:** under crash isolation N workers run N COBOL analyses at once. Under `max_extract_millis`, files queued for the slot time out while waiting, and each abandoned goroutine still takes the slot and runs `Analyze` for a discarded result.

### FK-01: `AttestEmbedded` reports a checkout identity it never checked

**File:** `<fork>/preprocessor/internal/tsadapter/parser.go`, `<fork>/preprocessor/analyze.go`
**Reason:** deferred: requires a fork commit and a user-approved push. The fork repository was not edited.
**Original issue:** embedded attestation returns the generated `expectedCheckoutArtifactID` constant as `CheckoutArtifactID` without fingerprinting any shim artifact, so its `ToolID` cannot be told apart from a real checkout attestation.

### FK-02: `ToolIdentity.ForestModuleVersion` ignores `replace`, so it records the stock version

**File:** `<fork>/preprocessor/pipeline.go` (`validateShim` → `dependencyVersion`)
**Reason:** deferred: requires a fork commit and a user-approved push. The fork repository was not edited. On the Gortex side, `prov_parser_module` already records the replace target, and WR-08 now adds `prov_preprocessor_module`.
**Original issue:** `dependencyVersion` returns `dependency.Version` and ignores `dependency.Replace`, so the handoff's `Tool`/`ToolID` records the stock `v1.9.1` and does not change when the replace target moves.

## Verification

All of these ran in the main checkout, under `GOWORK=off`, after the last fix commit.

- `go build ./...`: OK
- `go vet ./internal/indexer/ ./internal/parser/languages/ ./cmd/gortex/`: OK
- `golangci-lint@v2.13.1 run --timeout=10m ./...`: 0 issues
- `go test -race -count=1 -timeout 30m ./internal/indexer/`: ok (732s)
- `go test -race -count=1 ./internal/parser/languages/ ./cmd/gortex/ -run 'Cobol|SourceRevision'`: ok, ok
- `git diff 8a12927c HEAD -- go.mod go.sum`: empty
- 02-05 structural checker (`ci-check` from 02-05-PLAN.md) over all 7 workflows: `OK`. Every workflow parses with `yq`, and `.goreleaser.yml` is unchanged since `3238e524`.
- Credential step body (from `ci.yml`), run locally with a dummy secret and a temporary HOME: it exits 0, writes no `~/.gitconfig`, never prints the secret, git inside the step sees the insteadOf rewrite, and with an empty secret it fails with `::error::`.
- The `Stock COBOL grammar must fail closed` step body, run locally, still reports `compiled forest language identity mismatch`.
- Fail-before checks: each new behavioural test was run against the pre-fix code and failed (CR-01: 3 subtests; WR-01: 2; WR-02: 2; WR-03: 2; WR-05: 2).
- Real CI was not run (no push).

---

_Fixed: 2026-10-01T18:15:11Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
