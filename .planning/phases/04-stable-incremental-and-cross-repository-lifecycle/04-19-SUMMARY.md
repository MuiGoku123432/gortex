---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 19
subsystem: mcp
tags: [source-scope, explore, artifact-lane, sec-01, sec-02, d-13, d-16, inventory, go-parser]

requires:
  - phase: 04-12
    provides: requireSourceScope / errSourceScopeRequired fail-closed source gate (D-13)
  - phase: 04-17
    provides: requirePathSourceScope physical-owner gate and filterTextMatchesByResolvedScope (D-16)
provides:
  - handleExplore refuses an unbound multi-repository request before an artifact lane with text probes runs (decided from task text only)
  - gatherExploreArtifactLane admits probe matches only through filterTextMatchesByResolvedScope (physical owner in scope) before snippets and scoring
  - TestSourceScope_SourceChannelsInventoried — parse-level inventory of every Grep* call and filesystem walk in internal/mcp, each listed with its gate
  - TestSourceScope_SearchSymbolsContentFallbackGated — content fallback returns no line unbound, returns it beta-scoped
  - explore cases in TestSourceScope_UnboundMultiRepoRefused and both TestSourceScope_NestedCheckoutRefused states; IN-04 beta-scoped positive controls (find_declaration, batch_symbols)
  - Source-surface audit of every Grep*, walk, os.ReadFile, os.Open and readLinesForCtx site (below)
affects: [phase-04-verification, source-scope, explore, 04-22]

actuals:
  tokens: 2405
  tasks: 3
  commits: 3
plan_head_before: 6740ad6606d58c22569cbe2f398b0af78de73e2b
plan_head_after: 9900a27be3d2c0eeb47227cc513828203b878625

tech-stack:
  added: []
  patterns:
    - "Source channels are inventoried structurally: go/parser over non-test internal/mcp files, keyed by enclosing FuncDecl, compared to a function→gate map"
    - "Unbound refusal for a text-search surface is decided from request text alone (never from matches), so it is not an existence oracle"

key-files:
  created: []
  modified:
    - internal/mcp/tools_explore.go
    - internal/mcp/explore_artifact_intent.go
    - internal/mcp/source_scope_test.go

key-decisions:
  - "Explore refuses (rather than dropping snippets) when the artifact intent has probes and no scope resolves: a dropped snippet could still raise a file's score and reveal the text exists. Path-only intents and ordinary code tasks are not refused (D-13 metadata default)."
  - "Reused filterTextMatchesByResolvedScope in the lane with a ResolvedScope literal built inline from QueryOptions; no new helper and no lane signature change."
  - "The source-channel inventory keys on the enclosing function name, so a new call site inside an already-listed function is not flagged; adding a new function with a Grep*/walk call is."

patterns-established:
  - "Inventory-as-test: a channel call site cannot ship without a listed gate (TestSourceScope_SourceChannelsInventoried)"

requirements-completed: [SEC-01, SEC-02]

coverage:
  - id: D1
    description: "Unbound two-repository explore artifact request is refused with the scope-required text, no repo name, no deploy.yaml bytes; beta-scoped explore returns the line; unbound ordinary task is not refused"
    requirement: SEC-02
    verification:
      - kind: integration
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_UnboundMultiRepoRefused/explore"
        status: pass
    human_judgment: false
  - id: D2
    description: "Explore artifact lane admits matches by physical owner; alpha-scoped explore never returns nested beta's deploy.yaml in either index state"
    requirement: SEC-01
    verification:
      - kind: integration
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_NestedCheckoutRefused/{tracked_together,outer_indexed_first}/explore"
        status: pass
    human_judgment: false
  - id: D3
    description: "Structural source-channel inventory (Grep* calls and filesystem walks) enforced by a parse-level test"
    requirement: SEC-02
    verification:
      - kind: unit
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_SourceChannelsInventoried"
        status: pass
    human_judgment: false
  - id: D4
    description: "search_symbols content fallback returns no source line to an unbound multi-repo request and returns it beta-scoped"
    requirement: SEC-02
    verification:
      - kind: integration
        ref: "internal/mcp/source_scope_test.go#TestSourceScope_SearchSymbolsContentFallbackGated"
        status: pass
    human_judgment: false

duration: 19min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 19: Explore artifact lane source gate Summary

**Explore's artifact lane now refuses unbound multi-repository requests from the task text alone, and passes its matches through the same physical-owner filter search_text uses. A parse-level test lists every text-search and filesystem-walk call site with its gate. This closes VERIFICATION gaps[0] (SEC-01, SEC-02, SC-5).**

## Performance

- **Duration:** 19 min
- **Started:** 2026-10-13T20:47:01Z
- **Completed:** 2026-10-13T21:05:39Z
- **Tasks:** 3/3
- **Files modified:** 3

## Accomplishments

- `handleExplore` (internal/mcp/tools_explore.go): when `artifactIntent.active` and the intent has probes, it calls `requireSourceScope(ctx, "")` after `resolveScope` and before `gatherExploreArtifactLane`. `errSourceScopeRequired` becomes a tool error. This mirrors search_text and find_declaration (D-13, SEC-02).
- `gatherExploreArtifactLane` (internal/mcp/explore_artifact_intent.go): each probe's matches go through `filterTextMatchesByResolvedScope` before `enrichTextMatchesContext`. A match whose physical owner is out of scope adds no snippet and no score (D-16, SEC-01).
- Tests (internal/mcp/source_scope_test.go):
  - `exploreSecret` / `exploreSecretTask` and the `writeBetaDeploySecret` fixture
  - an explore case, a beta-scoped positive control and a no-over-refusal check in UnboundMultiRepoRefused
  - `deploy.yaml` in `newNestedCheckoutServer`, plus an `explore` sub-subtest in both nested states
  - IN-04 beta-scoped positive controls for find_declaration (literal and regex) and batch_symbols
  - `TestSourceScope_SourceChannelsInventoried` and `TestSourceScope_SearchSymbolsContentFallbackGated`

## Task Commits

1. **Task 1 (tracer): unbound explore artifact refusal**: `43846c32` (fix). RED showed the unbound explore returning `ZEBRA_SECRET_TOKEN: hunter2zebraxyz` from `beta/deploy.yaml`. GREEN after the refusal. The tracer gate re-ran `<verify>` end to end and it passed.
2. **Task 2: physical-owner filter in the lane**: `93248d33` (fix). RED showed `outer_indexed_first/explore` returning the secret for an alpha-scoped request through `alpha/nested/deploy.yaml`, while tracked_together passed. GREEN after the filter.
3. **Task 3: structural inventory + content fallback test**: `9900a27b` (test). These are guard tests and passed on HEAD plus Tasks 1-2. Guard-teeth check below.

## TDD / Guard-teeth record

- Task 1 RED: `TestSourceScope_UnboundMultiRepoRefused/explore` failed. The response was not IsError and contained the secret.
- Task 2 RED: `TestSourceScope_NestedCheckoutRefused/outer_indexed_first/explore` failed because the alpha-scoped response contained `hunter2zebraxyz`.
- Task 3 teeth: I temporarily deleted the `gatherExploreArtifactLane` entry from `textGates`. The test failed with `text-search call sites changed: put the new site behind requireSourceScope or requirePathSourceScope (or filterTextMatchesByResolvedScope) and list it here with its gate`, and the diff showed `+ "gatherExploreArtifactLane"`. I restored the file from a scratch copy and the test passes again.

## Source-surface audit

Method: `grep -nE '\.Grep[A-Za-z]*\('`, `grep -nE '(filepath|fs)\.Walk(Dir)?\('` and `grep -nE 'os\.ReadFile\(|os\.Open\(|readLinesForCtx\('` over non-test `internal/mcp/*.go`, with a go/ast pass mapping each site to its enclosing function. Each path was traced back to where it came from.

### Text-search channels (all match the expected set)

| Function | Sites | Gate |
|---|---|---|
| handleSearchText | tools_search_text.go:100,103,105,111,114,116 | unbound refusal + filterTextMatchesByResolvedScope (04-12, 04-17) |
| searchTextInView | view_search_text.go:55 (GrepCheckout) | reached only from handleSearchText after its refusal; same filter |
| findUseSiteMatches | tools_find_declaration.go:174,176,191,193 | handleFindDeclaration unbound refusal + requirePathSourceScope per path (04-17; filter-before-limit 04-22) |
| gatherExploreArtifactLane | explore_artifact_intent.go:390,392,394 | handleExplore unbound refusal + filterTextMatchesByResolvedScope (this plan) |
| searchExploreSourceLiteralDurable | explore_source_literal.go:1062,1079 | single repository only (searchExploreSourceLiteral refuses multi-repo fan-out); explore emits no text; search_symbols content fallback filters through filterTextMatchesByResolvedScope |

### Filesystem walks (all match the expected set)

| Function | Site | Gate |
|---|---|---|
| handleMoveSymbol | tools_move_inline.go:251 (filepath.Walk modRoot) | ungated per entry; root from gated resolveNodePath; 04-22 adds per-entry requirePathSourceScope |
| handleInlineSymbol | tools_move_inline.go:1135 | as above (04-22) |
| handleGenerateSkill | tools_generate_skill.go:145 (filepath.WalkDir) | ungated per entry; dir from resolveFilePath; 04-22 |

### File reads (os.ReadFile / os.Open / readLinesForCtx)

| Site | Function | Path source | Classification |
|---|---|---|---|
| manifest.go:255 | manifestSymbolSource | resolveNodePath | gated |
| localization_source_window.go:151 | localizationSourceWindowForHit | resolveFilePath | gated |
| dynamic_boundary.go:126 | dynamicBoundariesForSymbol | resolveNodePath | gated |
| tools_analyze_temporal_verify.go:95 | fileBody | resolveNodePath (caller) | gated |
| tools_artifacts.go:135 | handleGetArtifact | resolveFilePath + guardSymlinkWithinRepo | gated |
| tools_cfg.go:68 | buildSymbolCFG | resolveNodePath | gated |
| tools_dataflow.go:214 | dataflowRefiner | resolveNodePath | gated |
| tools_nav.go:381 | navRead | resolveNodePath | gated |
| tools_coding.go:599 | handleGetEditingContext | requireSourceScope + readViewFile / resolveNodePath | gated |
| tools_coding.go:1036 | handleGetSymbolSource | readViewFile / resolveNodePath + guardSymlinkWithinRepo | gated |
| tools_coding.go:1352 | handleBatchSymbols | requireSourceScope + resolveNodePath | gated |
| tools_coding.go:1715,1763,1793 | handleSuggestPattern | resolveNodePath | gated |
| tools_coding.go:2256 | handleSmartContext | requireSourceScope + resolveNodePath | gated |
| tools_coding.go:3014 | handleEditSymbol | resolveNodePath | gated |
| tools_coding.go:1154 | readLines (helper) | callers: readLinesForCtx; tools_enhancements.go:738, 2931 (handleDiffContext), tools_search_assist.go:544 (extractBodyForVerify), all resolveNodePath | gated via callers |
| tools_coding.go:3203 | readSingleLineAt (helper) | callers: handleRenameSymbol (resolveGraphPath closure), buildPropagationPlan (resolveNodePath) | gated via callers |
| tools_fileops.go:894 | handleEditFile | resolveFilePath | gated |
| tools_fileops.go:1097,1113,1139 | handleWriteFile | resolveFilePath | gated |
| tools_fileops.go:1546 | handleReadFile | resolveFilePath + guardSymlinkWithinRepo (installed-skill branch reads skill files, not repo source) | gated |
| physical_evidence_open_windows.go:11 | openPhysicalEvidenceFile | handleReadFile physical-evidence branch after resolveFilePath | gated via caller |
| tools_fileops.go:1795 | detectLanguageForPath | callers pass gated paths; language sniff only, no bytes returned | non-returning |
| savings_retrieval.go:216 | fileHeadSample | resolveGraphPath; token estimate only | non-returning |
| rename_apply.go:57 | unindexedRenameRecovery | resolveFilePath | gated |
| rename_apply.go:358 | planRenameWrites | resolveGraphPath | gated |
| safe_delete_propagate.go:114 | applyRemoveLinePatches | p.abs from buildPropagationPlan → resolveNodePath | gated via caller |
| tools_safe_delete.go:203 | handleSafeDeleteSymbol | resolveNodePath | gated |
| tools_safe_delete.go:353 | applyPendingDeletes | pending.abs from resolveNodePath (target + cascade) | gated via caller |
| tools_enhancements.go:4196 | applyBatchSymbolEdit | resolveNodePath | gated |
| tools_enhancements.go:4324 | applyBatchFileEdit | resolveFilePath | gated |
| tools_enhancements.go:2702 | handleScaffold (write branch) | `s.indexer` root only (single-repository indexer); edits from gated scaffoldReader (resolveGraphPath) | gated (single repo) |
| batch_transaction.go:586 | batchPathBytes | readBatchBuffers plans: resolveFilePath / resolveNodePath | gated via caller |
| batch_transaction_journal.go:62,199,211 | readBatchManifest / readBatchBackup / classifyBatchFiles | transaction journal + receipt paths recorded from gated plans | non-source (journal) |
| batch_transaction_durability.go:132 | syncBatchDirectory | directory fsync | non-source |
| tools_analyze_retrieval_log.go:215 | readQueryLogTail | query log file | non-source (log) |
| tools_overlay_branch.go:334,355 | diskWriteForMerge | resolveFilePath | gated |
| tools_review.go:1405 | identityEditForSymbol | resolveOverlayAbsPath (ungated) | bytes feed buildSimulation only; response carries counts (`touched=... broken_callers=...`) |
| tools_simulate_io.go:15 | readSimFile | callers contentForSim / simulateDiagnosticsAtStep via resolveOverlayAbsPath (ungated) | bytes feed the simulation and LSP diagnostics; response carries graph diff, counts, diagnostics; no raw lines |
| overlay.go:360 | overlaySHAMatches | resolveOverlayAbsPath | SHA comparison only, non-returning |
| tools_move_inline.go:119,275,605,631,658,1083,1169 | handleMoveSymbol / handleInlineSymbol / Go helpers | resolveNodePath / resolveFilePath roots; walk entries ungated | owned by 04-22 |
| tools_generate_skill.go:179 | handleGenerateSkill | walk entries | owned by 04-22 |

Planning-time results confirmed: identityEditForSymbol returns counts only. The handleScaffold write branch is single-repository only. fileHeadSample only estimates tokens. The three walks are ungated per entry and 04-22 gates them. The audit found no surface outside 04-22's files that returns or writes source bytes without a gate.

## Files Created/Modified

- `internal/mcp/tools_explore.go`: `errors` import and the unbound artifact-probe refusal in handleExplore
- `internal/mcp/explore_artifact_intent.go`: filterTextMatchesByResolvedScope on probe matches
- `internal/mcp/source_scope_test.go`: explore fixtures and cases, IN-04 positive controls, the inventory test, and the content-fallback test

## Decisions Made

See key-decisions in the frontmatter.

## Deviations from Plan

None. The plan executed as written. No existing explore test depended on an unbound multi-repository artifact snippet, so no test expectation changed.

## Issues Encountered

- I did not mark SEC-01 and SEC-02 Complete in REQUIREMENTS.md. `requirements.mark-complete` ran, and I reverted its edit. 04-22 also lists both requirements and still has to gate the three walks. Commit d3b60325 had already reverted an earlier premature Complete. Both stay "Gaps Found" until phase re-verification.

## Threat Flags

None new. T-04-65 and T-04-66 are mitigated by Tasks 1 and 2. T-04-67 is mitigated because the refusal is decided from the task text and dropped matches are removed before scoring. T-04-68 is mitigated by the inventory test. Residual, already noted at planning time: the simulation reads (identityEditForSymbol, readSimFile) resolve through ungated `resolveOverlayAbsPath` but return no raw source lines. LSP diagnostic messages from preview_edit/simulate_chain could in principle quote fragments of a simulated file. That is outside this plan.

## Verification

- Task 1 verify: `-list` found the test; `--- PASS: TestSourceScope_UnboundMultiRepoRefused/explore` under `-race`; `-run 'Explore|^TestSourceScope_'` race run ok (48s); gofmt clean.
- Task 2 verify: all 16 `TestSourceScope_NestedCheckoutRefused/<state>/<surface>` PASS lines present under `-race`; `-run 'Explore|^TestSourceScope_|^TestFilterTextMatches|^TestSearchText|^TestSearchSymbols_ContentFallback'` race run ok (59s); gofmt clean.
- Task 3 verify: all four TestSourceScope_ names listed; `-run '^TestSourceScope_|^TestAnalyzeKinds_|^TestSearchSymbols_ContentFallback'` race run ok (16s); gofmt clean.
- `GOWORK=off go build ./... && go vet ./internal/mcp/`: ok.
- `GOWORK=off go test -race -count=1 ./internal/mcp/ -timeout 45m`: ok (517s).
- `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolScope_' -timeout 20m`: ok.
- `git diff --name-only 6740ad66..HEAD` (code commits): only the three `files_modified` files.

## Next Phase Readiness

- 04-22 can gate the three walks per entry. The inventory already lists them with the "04-22" gate text, so that plan needs no inventory change unless it moves a walk into a new function.

## Self-Check: PASSED

- Files: tools_explore.go, explore_artifact_intent.go, source_scope_test.go, 04-19-SUMMARY.md all present.
- Commits: 43846c32, 93248d33, 9900a27b are ancestors of HEAD.
