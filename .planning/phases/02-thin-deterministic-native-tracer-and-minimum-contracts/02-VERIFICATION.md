---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
verified: 2026-10-01T19:30:00Z
status: passed
score: 68/79 must-haves verified (5/5 roadmap success criteria verified on darwin/linux)
covered_files:
  - ".github/workflows/bench-arm.yml"
  - ".github/workflows/ci.yml"
  - ".github/workflows/init-smoke.yml"
  - ".github/workflows/publish-claude-plugin.yml"
  - ".github/workflows/release.yml"
  - ".github/workflows/security.yml"
  - ".github/workflows/skill-drift.yml"
  - ".planning/REQUIREMENTS.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-01-PLAN.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-01-SUMMARY.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-02-PLAN.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-02-SUMMARY.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-03-PLAN.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-03-SUMMARY.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-04-PLAN.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-04-SUMMARY.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-05-PLAN.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-05-SUMMARY.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-06-PLAN.md"
  - ".planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-06-SUMMARY.md"
  - "cmd/gortex/cobol_tracer_test.go"
  - "go.mod"
  - "go.sum"
  - "internal/indexer/indexer.go"
  - "internal/indexer/source_revision.go"
  - "internal/indexer/source_revision_test.go"
  - "internal/neo4jprojection/properties_test.go"
  - "internal/parser/languages/cobol_grammar.go"
  - "internal/parser/languages/cobol_grammar_identity.go"
  - "internal/parser/languages/cobol_grammar_test.go"
  - "internal/parser/languages/cobol_grammar_unsupported.go"
  - "internal/parser/languages/cobol_grammar_unsupported_test.go"
  - "internal/parser/languages/register.go"
covered_digest: "v1:sha256:9b1052d5c3a47a898583b80042c46760a3f43fbcc3face708993b31c04d79c58"
behavior_unverified: 1
overrides_applied: 0
behavior_unverified_items:
  - truth: "02-05: the linux release downloads modules on the host with the credential, then runs goreleaser-cross with the host module cache mounted read-only at /go/pkg/mod, GOPROXY=off, and GOPRIVATE, so the fork credential never enters the container"
    test: "Run release.yml on a throwaway tag (or workflow_dispatch to a scratch release) and watch the linux goreleaser-cross step"
    expected: "The container builds from the read-only host module cache with GOPROXY=off and no COBOL_PARSER_READ_PAT in its environment; the release artifacts build"
    why_human: "The configuration is present (release.yml:220, 302-308) but release.yml has never run since the change (WINDOWS.md entry 3); no test exercises a container build against a read-only module cache"
human_verification:
  - test: "When CI run 36909797580 on head a0d2eac9 finishes, check the three test shards (ubuntu, macos, windows) and benchmark"
    expected: "ubuntu and macos test shards green (the ubuntu 'Stock COBOL grammar must fail closed' step already passed on this head); windows shard either green or failing for a cause that is diagnosed and recorded"
    why_human: "At verification time lint, build-linux-static, build-onnx, security, init-smoke, and skill-drift were green on a0d2eac9 but the test shards were still in progress. The windows shard failed on 54ac27c9 and d4e0b3b7 with an undiagnosed cause (job log needs repo-admin access). The user accepted that as a known exception at the 02-06 checkpoint, but WR-09 changed the build tags since then, so the earlier acceptance covers a different build"
  - test: "As repo owner, open the windows-latest job log and find the first '--- FAIL' line; confirm TestCobolGrammarUnsupportedPlatformStub passes on Windows"
    expected: "The stub test passes on Windows and the stub returns errCobolParserUnsupportedPlatform for every COBOL file, so Windows fails closed (D-05, D-17, SC1 on Windows)"
    why_human: "The stub has only been checked by build-tag selection (`GOOS=windows go list` selects cobol_grammar_unsupported.go and no fork package); it has never been observed to run on Windows (WINDOWS.md entry 2)"
  - test: "Accept or reject the review-fix deviations from the plan text (override candidates below): the expanded prov_vcs_commit_absence vocabulary (adds indexed_bytes_differ_from_file and vcs_query_failed), the replacement of gitstate.NewDirtySampler with a HEAD blob-ID comparison, and the D-17 stub rename and retag to `!darwin && !linux` with errCobolParserUnsupportedPlatform"
    expected: "Each is accepted as an override (or the plan/CONTEXT text is amended), because each preserves or strengthens the original intent"
    why_human: "02-04-PLAN.md, 02-02-PLAN.md, 02-03-PLAN.md, and CONTEXT D-17 still name the superseded vocabulary, key link, error name, and build tag. The code is stricter than the plan, but the plan text is the contract and only the user can accept the change. 02-REVIEW-FIX.md also asks a human to confirm the logic changes in CR-01, WR-01, and WR-05 before verification"
  - test: "Backstop truths with no explicit evidence: (1) TRACE-04 idempotency: reindex the unchanged fixture and count active program nodes and file-to-program defines edges; (2) TRACE-07 ordering; (3) PROV-01 ordering; (4) PROV-02 ordering; (5) PROV-04 ordering (several clean COBOL files in one pass all get the same HEAD commit)"
    expected: "Exactly one DEMOPGM node and one defines edge after a reindex; results independent of test, package, or file order"
    why_human: "These are `verification: backstop` items. The code is stateless per file and consistent with them, but presence and wiring never qualify for a backstop, and no held-out or property test exercises them. The tracer checks the single defines edge only after the first index, not after a reindex"
  - test: "Review the ten judgment-tier prohibitions listed in the report (fail-closed attestation, no stock or regex facts, no source in Meta, invented fixtures, no silent degradation, no guessed containment, no false VCS attribution, no private paths in docs)"
    expected: "Each holds. The LLM-judge verdict below is 'holds' for all ten"
    why_human: "Judgment-tier prohibitions need explicit human resolution; the verifier's verdict is not authoritative (unverified-prohibition — human review recommended)"
overrides:
  - must_have: "Per D-17, a Windows build compiles a stub NewCobolGrammarExtractor whose Extract returns errCobolParserUnavailableOnWindows"
    reason: "WR-09: the stub now covers every non-darwin/linux platform under //go:build !darwin && !linux (file cobol_grammar_unsupported.go) and returns errCobolParserUnsupportedPlatform; with !windows, freebsd and others selected the real extractor that cannot link"
    accepted_by: "project owner (user)"
    accepted_at: "2026-10-01T20:31:35Z"
  - must_have: "Per D-17, the Windows stub test asserts Extract returns errCobolParserUnavailableOnWindows with the same Language and Extensions as the real extractor"
    reason: "WR-09 rename; TestCobolGrammarUnsupportedPlatformStub asserts the same contract under the new name"
    accepted_by: "project owner (user)"
    accepted_at: "2026-10-01T20:31:35Z"
  - must_have: "Per D-09, prov_vcs_commit_absence is exactly one of git_unavailable, not_a_git_worktree, no_head_commit, not_under_version_control, or working_tree_differs_from_head"
    reason: "CR-01 added indexed_bytes_differ_from_file and WR-05 added vcs_query_failed; the vocabulary stays closed and a commit is claimed only when the extracted bytes hash to HEAD's blob"
    accepted_by: "project owner (user)"
    accepted_at: "2026-10-01T20:31:35Z"
  - must_have: "PROV-04 adjacency: prov_vcs_commit is written only for a file that is tracked in the index and has no dirty entry; any worktree or index difference yields working_tree_differs_from_head"
    reason: "CR-01 replaced the DirtySampler dirty-entry check with a HEAD blob-ID comparison of the exact extracted bytes; transformed or changed-after-read bytes now yield indexed_bytes_differ_from_file"
    accepted_by: "project owner (user)"
    accepted_at: "2026-10-01T20:31:35Z"
  - must_have: "CI green on ubuntu, macos, and windows"
    reason: "User accepted closing Phase 2 with the windows-latest test shard failing (cause undiagnosed; job log needs repo-admin access). ubuntu and macos race suites, lint, static, onnx, benchmark, govulncheck, init-smoke, skill-drift all green on a0d2eac9 (run 36909797580). windows-latest passed on origin/main cbcaf8d5, so the regression is on this branch (Phase 1 or Phase 2 code). Deferred: WINDOWS.md entry 2, deferred-items.md 02-06 item 1."
    accepted_by: "project owner (user)"
    accepted_at: "2026-10-01T20:31:35Z"
human_acceptance: "Closed by the project owner on 2026-10-01T20:31:35Z ('I am fine with phase 2 closing out windows is not my priority right now'). Accepted without item-by-item review: the review-fix override candidates, the 5 backstop truths (TRACE-04 reindex idempotency; TRACE-07/PROV-01/PROV-02/PROV-04 ordering), and the 10 judgment-tier prohibitions (verifier verdict: all hold). Windows is an open, deferred exception."
---

# Phase 2: Thin Deterministic Native Tracer and Minimum Contracts Verification Report

**Phase Goal:** A user can retrieve one trustworthy named COBOL program observation through existing native Gortex surfaces after it traverses the real production indexing and SQLite path.
**Verified:** 2026-10-01T19:30:00Z
**Status:** passed (accepted with exceptions by the project owner; see `overrides` and `human_acceptance` in frontmatter)
**Re-verification:** No. This is the first verification. No earlier 02-VERIFICATION.md existed.
**Code baseline:** HEAD `a0d2eac9` (after the review-fix commits 90cbc134..53d88fd3). The working tree has no source changes, only `.planning/config.json` and `.planning/state.json`.
**Platform scope:** Every behavioral result below was observed on darwin (local). It also covers linux through CI on d4e0b3b7 and the ubuntu stock-grammar step on a0d2eac9. Nothing was observed running on Windows.

## Goal Achievement

The goal is achieved on darwin and linux. The decisive evidence is a real-path, in-process acceptance that I ran myself under `-race`. It goes through the registered production extractor (`RegisterAll` → `NewCobolGrammarExtractor`), `MultiIndexer.TrackRepoCtx`, `applyRepoPrefix`, and the indexer lifecycle with the shadow disabled. A mutex-guarded recorder wraps the real `*store_sqlite.Store` and sees the prefixed ID in `AddBatch`/`AddBatchChecked`. After that the test closes the store and reopens it from disk, then retrieves the node through the real daemon relay with MCP `get_symbol` (detail=full), the pre-existing `gortex call get_symbol` CLI verb (`runCall`), and MCP `search_symbols`.

### Roadmap Success Criteria

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A clean or `GOWORK=off` build uses the pinned commit `f9eaf99c34a9` (floor `97ac9f1`), records its grammar content identifier, or fails instead of silently selecting the stock grammar | ✓ VERIFIED (darwin/linux) | `go.mod:6` requires `.../preprocessor v0.0.0-20260930215433-f9eaf99c34a9`. `go.mod:375` replaces `go-sitter-forest/cobol` with `.../forest-shim/cobol` at the same pseudo-version, and go.sum pins both. No `go.work` or `go.work.sum` exists. `GOWORK=off go build ./cmd/gortex/` exits 0. In the fork, `f9eaf99c` has `f19029f` as its only parent, `97ac9f1` is an ancestor, and the commit is on `origin/gsd/v0.26.0-estate-parse-recovery`. The fork tests `TestAttestEmbedded*`, `TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer`, and `TestNewContentIDMatchesHandoff` pass. The grammar ID is persisted as `prov_parser_grammar_id` (tracer asserts it equals `f97452e1…503d`). I ran the stock-grammar negative locally (temp modfile, `-dropreplace`). `TestCobolGrammar_ApprovedGrammarPinned` failed with `compiled forest language identity mismatch`, as required. The same CI step passed on ubuntu at a0d2eac9. On Windows only build-tag selection was checked (stub, no fork package). See Human Verification |
| 2 | Indexing the tracer fixture emits one named program as a native node and its source-file relationship as a native edge through the registered extractor, ExtractionResult, repository prefixing, indexer lifecycle, and SQLite AddBatch | ✓ VERIFIED | `register.go:119` is the only register.go change from `3238e524` (`NewCobolExtractor` → `NewCobolGrammarExtractor`). `TestCobolTracer_PersistsThroughAddBatch` PASS: the recorder saw `{prefix}/cobol/demopgm.cbl::DEMOPGM` and the file ID before any retrieval. After reopen, `GetNode` returns the program, `GetInEdges` holds exactly one `defines` edge from the file node, and `graphfixture.Canonical` shows a SQLite node row. `TestCobolGrammar_ProgramNode` shows exactly one program node and exactly one edge (D-15) |
| 3 | The traced observation exposes stable repository-prefixed identity, exact source range, scope, revision, parser/extractor versions, evidence class, origin, and confidence without line number as its sole identity discriminator | ✓ VERIFIED | `assertCobolTracerProgram` checks the stored node, the reopened node, the MCP payload, and the CLI payload. It covers kind, name, qual_name, prefixed file_path, `prov_source_path`, `prov_parser_grammar_id`, `prov_parser_module` (shim replace path), `prov_preprocessor_module` (pinned), `prov_extractor_version=gortex-cobol-grammar/2`, `prov_evidence_class=DETERMINISTIC`, `prov_origin=ast_resolved`, `prov_confidence=1.0`, grade/policy/schema, `prov_range_exact=true`, row/line consistency, five 64-hex identity keys, `prov_vcs_commit` equal to the fixture HEAD, and no source text in prov_ values. Scope: RepoPrefix, WorkspaceID, and ProjectID are non-empty and equal the file node's values. ID shape is `filePath::containment[#k]` (`cobol_grammar_identity.go`). `TestCobolGrammar_IDIgnoresLines` shows that prepended lines shift rows and leave IDs unchanged |
| 4 | A user can retrieve the same persisted node through an existing CLI query and an existing MCP query | ✓ VERIFIED | `TestCobolTracer_MCPGetSymbol` PASS: daemon relay `tools/call get_symbol` with detail=full returns the exact ID, the full contract, and an in-edge `defines` from the file. `TestCobolTracer_CLIGetSymbol` PASS through `runCall` (`gortex call`, present in `cmd/gortex/call.go` at baseline `3238e524`). `TestCobolTracer_SearchSymbolsByName` PASS |
| 5 | Passes with AI providers disabled and no Neo4j installation, SQLite sole authority, no deterministic dual-write | ✓ VERIFIED | `TestCobolTracer_NoAINoNeo4j` PASS: `srv.LLMService()` is nil, `GORTEX_LLM_PROVIDER` is empty, `ask` is not registered, and get_symbol still serves the node. `go list -deps` finds 0 `neo4j-go-driver` packages in `internal/indexer`, `internal/parser/languages`, and `internal/graph/store_sqlite`, so the indexing path cannot dual-write. No workflow configures Neo4j or an LLM provider |

### Plan Must-Have Truths (grouped)

| Plan | Truths | Verified | Not verified | Notes |
|------|--------|----------|--------------|-------|
| 02-01 (fork embedded attestation) | 5 | 5 | 0 | Fork tests pass at f9eaf99. `AttestEmbedded` keeps the verbatim `compiled forest language identity mismatch` comparison. Lineage and origin checked |
| 02-02 (pin, extractor, tracer) | 21 + 3 backstop | 20 | 1 superseded + 3 backstop | D-17 truth names `errCobolParserUnavailableOnWindows` and a `!windows` tag. WR-09 replaced them with `errCobolParserUnsupportedPlatform` and `!darwin && !linux`: stricter, and the intent holds. Override candidate. Backstops TRACE-04 idempotency, TRACE-07 ordering, and PROV-01 ordering have no explicit evidence |
| 02-03 (identity, copybooks, guards) | 21 + 3 backstop | 20 + 2 backstop | 1 superseded + 1 backstop | Same D-17 rename (stub test now `TestCobolGrammarUnsupportedPlatformStub`). Backstop PROV-03 empty is VERIFIED by `TestCobolGrammar_RangeNeverInvented` (missing/no_image/synthetic write no coordinates). Backstop PROV-05 ordering is VERIFIED by `TestCobolGrammar_ApprovedGrammarPinned` (identity keys are equal across files in one process). Backstop PROV-02 ordering has no explicit evidence |
| 02-04 (VCS revision, projection) | 6 + 1 backstop | 4 | 2 superseded + 1 backstop | Truth 2 lists exactly five absence reasons. CR-01/WR-05 added `indexed_bytes_differ_from_file` and `vcs_query_failed`. The PROV-04 adjacency truth's "any worktree or index difference yields working_tree_differs_from_head" now has a BOM/transform case that yields `indexed_bytes_differ_from_file` instead. Both are stricter, and override candidates. Backstop PROV-04 ordering has no explicit evidence |
| 02-05 (CI credentials) | 7 | 6 | 1 present, behavior unverified | Structure checked in all 7 workflows: setup-go count equals credential-step count, `cache: false` everywhere, `GOPRIVATE` at workflow level, no `pull_request_target`, `.goreleaser.yml` unchanged. The release container handoff has never run |
| 02-06 (docs, CI, gate) | 7 | 6 | 1 | REQUIREMENTS/ROADMAP/PROJECT/cobolprobe/languages.md amendments present. Gate items re-run locally: build, vet, tidy-clean, focused race suite, cobolprobe `-enhanced-parser` gate PASS, stock negative, and a Windows closure without the fork. The full-suite result is recorded under Behavioral Spot-Checks. "CI green on ubuntu, macos, **and windows**" is not met: windows failed on d4e0b3b7 (user-accepted exception) and is in progress on a0d2eac9 |

**Score:** 68/79 must-haves verified (5 roadmap SCs + 74 plan truths; plan truths that restate an SC are counted once, in the plan). Not verified: 4 superseded-wording truths (override candidates), 1 CI truth pending/excepted, 1 present-behavior-unverified (release container), and 5 backstops (insufficient_spec). The remaining 2 backstops have explicit test evidence. No truth is FAILED for a missing implementation.

### Override Candidates (intentional deviations, not yet accepted)

**This looks intentional.** Each item below comes from a code-review fix that is stricter than the plan text. To accept them, add to this file's frontmatter:

```yaml
overrides:
  - must_have: "Per D-17, a Windows build compiles a stub NewCobolGrammarExtractor whose Extract returns errCobolParserUnavailableOnWindows"
    reason: "WR-09: the stub now covers every non-darwin/linux platform under //go:build !darwin && !linux (file cobol_grammar_unsupported.go) and returns errCobolParserUnsupportedPlatform; with !windows, freebsd and others selected the real extractor that cannot link"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
  - must_have: "Per D-17, the Windows stub test asserts Extract returns errCobolParserUnavailableOnWindows with the same Language and Extensions as the real extractor"
    reason: "WR-09 rename; TestCobolGrammarUnsupportedPlatformStub asserts the same contract under the new name"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
  - must_have: "Per D-09, prov_vcs_commit_absence is exactly one of git_unavailable, not_a_git_worktree, no_head_commit, not_under_version_control, or working_tree_differs_from_head"
    reason: "CR-01 added indexed_bytes_differ_from_file and WR-05 added vcs_query_failed; the vocabulary stays closed and a commit is claimed only when the extracted bytes hash to HEAD's blob"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
  - must_have: "PROV-04 adjacency: prov_vcs_commit is written only for a file that is tracked in the index and has no dirty entry; any worktree or index difference yields working_tree_differs_from_head"
    reason: "CR-01 replaced the DirtySampler dirty-entry check with a HEAD blob-ID comparison of the exact extracted bytes; transformed or changed-after-read bytes now yield indexed_bytes_differ_from_file"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` / `go.sum` | require + replace at one pseudo-version | ✓ VERIFIED | Lines 6 and 375; four go.sum lines; `go mod tidy -diff` clean |
| `internal/parser/languages/cobol_grammar.go` | Grammar extractor: lazy attestation, serialized Analyze, grammar and preprocessor pins, prov_* keys | ✓ VERIFIED | 292 lines, `darwin \|\| linux`; `cobolApprovedGrammarID`, `cobolApprovedPreprocessorModule`, `cobolStampRange` |
| `internal/parser/languages/cobol_grammar_identity.go` | END PROGRAM stack containment, ordinals, unresolved flags | ✓ VERIFIED | Case-insensitive marker key (WR-02), stolen-marker flags (WR-01) |
| `internal/parser/languages/cobol_grammar_unsupported.go` (planned as `cobol_grammar_windows.go`) | Fail-closed stub | ✓ VERIFIED (build-tag only) | `!darwin && !linux`; `GOOS=windows`/`freebsd` `go list` selects it and no fork package |
| `internal/parser/languages/register.go` | Registers the grammar extractor | ✓ VERIFIED | Single-line diff from `3238e524`; `cobol.go`/`cobol_test.go` byte-identical (D-16) and regex tests still pass |
| `internal/indexer/source_revision.go` | `stampSourceRevision` | ✓ VERIFIED | Signature now `(relPath, src, result)` (CR-01) |
| `internal/indexer/indexer.go` | One additive hook call | ✓ VERIFIED | Diff from `3238e524` is exactly 1 line (`:1721`); `applyCoverageDomains` is called from bulk (`:3626`, `:4659`) and incremental (`file_delta.go:161`) paths |
| `cmd/gortex/cobol_tracer_test.go` | Real-path acceptance | ✓ VERIFIED | 5 tests PASS under `-race` |
| `internal/neo4jprojection/properties_test.go` | `TestProjectNode_CobolProvenance` | ✓ VERIFIED (drift, see Anti-Patterns) | PASS. Fixture predates WR-08 (`prov_extractor_version` "/1", no `prov_preprocessor_module`) |
| `.github/workflows/*.yml` (7) | Credential, cache, GOPRIVATE, stock negative | ✓ VERIFIED | See 02-05 row |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| register.go | cobol_grammar.go | `NewCobolGrammarExtractor()` | ✓ WIRED | register.go:119 |
| cobol_grammar.go | fork preprocessor | `preprocessor.NewEmbeddedAnalyzer` | ✓ WIRED | `init()`, under `sync.Once` |
| cobol_grammar.go | cobol_grammar_identity.go | `cobolProgramSymbols(` | ✓ WIRED | Extract loop |
| fork analyze.go | fork pipeline.go → tsadapter | `validateEmbeddedShim()` → `tsadapter.AttestEmbedded` | ✓ WIRED | pipeline.go:472-474 at f9eaf99 |
| cobol_tracer_test.go | store_sqlite | recorder overrides `AddBatch`/`AddBatchChecked` | ✓ WIRED | |
| cobol_tracer_test.go | mcp get_symbol | relay `tools/call` with `"detail": "full"` | ✓ WIRED | |
| indexer.go | source_revision.go | `idx.stampSourceRevision(relPath, src, result)` | ✓ WIRED | Pattern changed (adds `src`) per CR-01 |
| source_revision.go | gitstate/dirty.go | `gitstate.NewDirtySampler` | ⚠️ REPLACED | CR-01 removed it in favor of `rev-parse` + `ls-tree` blob-ID comparison via `gitcmd`. Intent holds; override candidate |
| ci.yml | cobol_grammar_test.go | `TestCobolGrammar_ApprovedGrammarPinned` under dropped replace | ✓ WIRED | ci.yml:70-90; passed on ubuntu at a0d2eac9 |
| release.yml | go.mod | host `go mod download` | ✓ WIRED (config) | Behavior unexercised |
| REQUIREMENTS.md | go.mod | BASE-01/03 cite `f9eaf99c34a9` | ✓ WIRED | REQUIREMENTS.md:5, 35, 37, 193 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| Program node Meta | `docMeta` | `h` from the real `Analyze` handoff and build info | Yes. The tracer asserts real hex IDs and the HEAD commit from a real git repo | ✓ FLOWING |
| `prov_vcs_commit` | `classifySourceRevision` | real `git rev-parse`/`ls-tree` | Yes, equals `gitCommitHash(root)` in the tracer | ✓ FLOWING |
| MCP/CLI payload | get_symbol result | reopened-capable SQLite store via query engine | Yes, the same ID and contract as the store | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| GOWORK=off build | `GOWORK=off go build ./cmd/gortex/` | exit 0 | ✓ PASS |
| Tracer acceptance | `GOWORK=off go test -race -count=1 -v -run 'TestCobolTracer_' ./cmd/gortex/` | 5/5 PASS, 0 SKIP | ✓ PASS |
| Grammar unit and guard suite | `... -run 'TestCobolGrammar\|TestCobol' ./internal/parser/languages/` | 18 TestCobolGrammar_* (+subtests) and 9 regex TestCobolExtractor_* PASS | ✓ PASS |
| VCS revision stamp | `... -run TestSourceRevisionStamp ./internal/indexer/` | 16/16 subtests PASS (incl. bom_stripped, reverted_after_hashing, assume_unchanged, sha256_repository, corrupt_repository, submodule) | ✓ PASS |
| Neo4j projection of prov keys | `... -run TestProjectNode_CobolProvenance ./internal/neo4jprojection/` | PASS | ✓ PASS |
| Stock grammar fails closed | ci.yml step body run locally | test FAIL with `enhanced COBOL parser unavailable: compiled forest language identity mismatch` | ✓ PASS |
| cobolprobe enhanced gate | `go test -race ./internal/parser/forest/cobolprobe/ -run TestErrorCascade -enhanced-parser` | PASS | ✓ PASS |
| Fork producer tests | `go test -run TestAttestEmbedded ./preprocessor/internal/tsadapter/` and `-run 'TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer\|TestNewContentIDMatchesHandoff' ./preprocessor/` (fork @ f9eaf99) | ok, ok | ✓ PASS |
| Neo4j driver absent from indexing path | `go list -deps` on indexer/parser/store_sqlite | 0/0/0 | ✓ PASS |
| Non-darwin/linux selects stub, no fork | `GOOS=windows go list -deps ./cmd/gortex \| grep -c tree-sitter-cobol-upgrade`; `GOOS={windows,freebsd} go list -f '{{.GoFiles}}'` | 0; only `cobol_grammar_unsupported.go` | ✓ PASS |
| tidy / vet | `go mod tidy -diff`; `go vet` on changed packages | clean; ok | ✓ PASS |
| Phase 1 regression | `go test -race ./internal/neo4jprojection/ ./internal/testutil/...`; `-run 'Scoped\|Projection\|Neo4j' ./internal/graph/store_sqlite/ ./cmd/gortex/` | ok (all) | ✓ PASS |
| Full suite (darwin) | `GOWORK=off go test -race -count=1 -timeout=45m ./...` | exit 0: 156 packages ok, 0 FAIL (darwin; run after all review-fix commits) | ✓ PASS |
| CI on a0d2eac9 | `gh run view 36909797580` | lint, build-linux-static, build-onnx, benchmark, security, init-smoke, skill-drift success; ubuntu "Stock COBOL grammar must fail closed" success; ubuntu/macos/windows test shards still in progress | ? PENDING |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared or present for this phase. The cobolprobe gate is a Go test and ran above. Step 7c: no shell probes to run.

### Requirements Coverage

All 18 phase IDs are claimed by at least one plan and mapped to Phase 2 in REQUIREMENTS.md. No orphans.

| Requirement | Source Plan | Status | Evidence |
|-------------|-------------|--------|----------|
| BASE-01 | 02-01, 02-02, 02-03, 02-06 | ✓ SATISFIED | go.mod pin at f9eaf99c34a9; preprocessor module pin (WR-08); REQUIREMENTS amended |
| BASE-02 | 02-01..02-03, 02-05, 02-06 | ✓ SATISFIED (darwin/linux) | Stock negative fails with the mismatch text; grammar and preprocessor mismatch tests; no fallback. Windows: stub, build-tag only |
| BASE-03 | 02-01..02-03, 02-06 | ✓ SATISFIED | `prov_parser_grammar_id` is 64-hex, equal to the pin, and identical across files |
| TRACE-01 | 02-02, 02-03 | ✓ SATISFIED | Registered production extractor emits DEMOPGM |
| TRACE-02 | 02-02 | ✓ SATISFIED | KindFunction node + EdgeDefines |
| TRACE-03 | 02-02 | ✓ SATISFIED | MultiIndexer → applyRepoPrefix → lifecycle; extractor emits unprefixed IDs |
| TRACE-04 | 02-02 | ✓ SATISFIED | AddBatch recorder + reopen (shadow disabled; see Info) |
| TRACE-05 | 02-02 | ✓ SATISFIED | `gortex call get_symbol` |
| TRACE-06 | 02-02 | ✓ SATISFIED | MCP get_symbol detail=full |
| TRACE-07 | 02-02, 02-05, 02-06 | ✓ SATISFIED | NoAINoNeo4j test; dependency closure |
| PROV-01 | 02-02 | ✓ SATISFIED | RepoPrefix/WorkspaceID/ProjectID non-empty, equal to file |
| PROV-02 | 02-02, 02-03 | ✓ SATISFIED | `prov_source_path` unprefixed on every node incl. copybooks |
| PROV-03 | 02-02, 02-03 | ✓ SATISFIED | Rows/columns/bytes from the original projection; never invented. BOM byte offset residual deferred to Phase 3 |
| PROV-04 | 02-04 | ✓ SATISFIED | `prov_revision_content_id` always; `prov_vcs_commit` or an absence reason |
| PROV-05 | 02-02, 02-03 | ✓ SATISFIED | tool/grammar/module/preprocessor/config/extractor version keys |
| PROV-06 | 02-02, 02-04 | ✓ SATISFIED | Three separate keys; each projects as its own `meta_*` property |
| ID-01 | 02-02, 02-03 | ✓ SATISFIED | `{prefix}/cobol/demopgm.cbl::DEMOPGM` |
| ID-02 | 02-03 | ✓ SATISFIED | Containment path + ordinal; lines never in IDs |

### Prohibitions (judgment tier; LLM-judge verdicts, non-authoritative)

All ten are `verification: judgment` and flagged **unverified-prohibition — human review recommended**.

| # | Plan | Statement (short) | Verdict | Basis |
|---|------|-------------------|---------|-------|
| 1 | 02-01 | Do not relax the compiled-language identity comparison | holds | `AttestEmbedded` performs the same `compiled != expectedCompiledLanguageID` check |
| 2 | 02-01 | No estate content in fork tests/commit | holds | f9eaf99 diff has no private paths and no new PROGRAM-ID fixtures |
| 3 | 02-02 | No COBOL facts from the stock grammar or regex | holds | Regex unregistered; init/attest/grammar/preprocessor checks return errors, with no fallback |
| 4 | 02-02 | No Generated/ledger/map/source in Meta | holds | `h.Generated` is read only for the program-name span; tracer asserts no source text in prov_ values |
| 5 | 02-02 | Invented fixtures only | holds | DEMOPGM, OUTERPGM, INNERPGM, FIRSTPGM, ONEPGM, AMBERPGM, BADPGM |
| 6 | 02-03 | No silent drop of degraded observations | holds | `TestCobolGrammar_DegradedAmber` (amber/red) emits program with grade keys, confidence 1.0 |
| 7 | 02-03 | No merged IDs or guessed containment | holds | WR-01/WR-02 flags; `StolenEndMarkerUnresolved`, `ContainmentUnresolved` tests |
| 8 | 02-03 | Invented fixtures only | holds | As #5 |
| 9 | 02-04 | No VCS commit for modified/staged/untracked/ignored bytes | holds | Blob-ID equality gate; modified/staged_new/untracked/ignored/assume_unchanged/bom_stripped subtests |
| 10 | 02-06 | No estate names, home paths, tokens in edited docs | holds | No `/Users/` or `/home/` in edited docs; corpus paths use `$COBOL_CORPUS_ROOT` |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (all phase-modified files) | n/a | TBD/FIXME/XXX/TODO/placeholder | none found | — |
| `internal/neo4jprojection/properties_test.go` | 228-270 | Contract fixture drift: says it "must stay identical" to the contract, but still uses `prov_extractor_version` "/1" and lacks `prov_preprocessor_module` (added by WR-08) | ⚠️ Warning | Low. Projection is generic per key, and `prov_preprocessor_module` contains no sensitive substring (`properties.go:174`), so it projects. The PROV-06 contract test no longer covers the full key set |
| `docs/languages.md` | 279 | Says "COBOL extraction is unavailable on Windows"; after WR-09 it is unavailable on every non-darwin/linux platform | ℹ️ Info | Doc accuracy |
| `.planning/WINDOWS.md` | entry 2 | Points at the old `cobol_grammar_windows_test.go` path | ℹ️ Info | Ledger accuracy |
| `cmd/gortex/cobol_tracer_test.go` | 107 | Tracer forces `GORTEX_SHADOW_MAX_FILES=0`. The default daemon cold path (in-memory shadow → BulkLoad) is not exercised for COBOL Meta | ℹ️ Info | SC2/TRACE-04 explicitly name AddBatch, so the criterion is met. Meta survival through BulkLoad is unproven by this phase |
| `cmd/gortex/cobol_tracer_test.go` | 247-277 | Daemon wiring is duplicated from `spinUpDaemonWithConfig` in the test, not the production daemon bootstrap | ℹ️ Info | The MCP handler, relay, and CLI `runCall` are real. Production composition is assumed equivalent |

### Accepted / Deferred (not gaps)

These were checked against the user-accepted list and deferred-items.md. They do not affect status: D-15 (temporary loss of paragraph/CALL/COPY edges; docs and PROJECT.md state it), D-16 (regex `cobol.go` unregistered, byte-identical), WR-04 (per-process analyze slot, documented, Phase 4), FK-01, FK-02 (fork), CR-01 residual BOM byte offsets (Phase 3), WR-07 fork-PR/Dependabot consequence (open user action).

### Human Verification Required

1. **CI on a0d2eac9.** When run 36909797580 finishes, confirm the ubuntu and macos test shards are green and record the windows result. The earlier user acceptance of the windows failure covered d4e0b3b7, before WR-09 changed the build tags.
2. **Windows stub actually runs.** Read the windows job log as repo owner. Confirm `TestCobolGrammarUnsupportedPlatformStub` passes and identify the first `--- FAIL`. This closes SC1 and D-17 on Windows (WINDOWS.md entry 2).
3. **Accept the review-fix deviations** (override candidates above). Also confirm the CR-01, WR-01, and WR-05 logic changes, which 02-REVIEW-FIX.md marks as needing a human.
4. **Backstop truths** (insufficient_spec): TRACE-04 reindex idempotency, and the ordering truths for TRACE-07, PROV-01, PROV-02, and PROV-04.
5. **Release container handoff** (present, behavior unverified): run release.yml on a throwaway tag (WINDOWS.md entry 3).
6. **Judgment-tier prohibitions:** confirm the ten verdicts above.

### Gaps Summary

No blocking gap was found. On darwin and linux the codebase achieves the phase goal. A named COBOL program from the pinned enhanced grammar goes through the registered extractor, prefixing, the indexer lifecycle, and SQLite AddBatch. It persists across a reopen with its full provenance contract, and it is retrievable by exact ID through MCP `get_symbol` and the CLI `gortex call get_symbol`, with AI and Neo4j absent. I ran every decisive test myself under `-race` and none skipped. The stock-grammar negative fails closed for the right reason.

Status is `human_needed` rather than `passed` for four reasons:
- Windows has only build-tag evidence, and CI on the post-review head is still running.
- Four plan truths name vocabulary, mechanisms, or identifiers that review fixes replaced with stricter ones. They need user-accepted overrides.
- Five backstop truths have no explicit evidence.
- The release-container path and the judgment-tier prohibitions need human sign-off.

---

_Verified: 2026-10-01T19:30:00Z_
_Verifier: Claude (gsd-verifier)_
