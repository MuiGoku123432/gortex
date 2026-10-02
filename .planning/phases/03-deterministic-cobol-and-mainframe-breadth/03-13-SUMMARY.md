---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 13
subsystem: parser
tags: [cobol, copybook, preprocessor, re-pin, pseudo-version, data-items, provenance, d-16]

requires:
  - phase: 03-11
    provides: "Fork API transform.RunCopybook, grade.PolicyCopybookV1, Analyzer.AnalyzeCopybook"
  - phase: 03-12
    provides: "Fork commit e64234cc493c487393c5e7867adfd7ca92d936ba on origin gsd/v0.26.0-estate-parse-recovery"
  - phase: 03-05
    provides: "cobolDataSection level walker and the REDEFINES/RENAMES binding pass"
provides:
  - "Gortex preprocessor require pinned at v0.0.0-20261002100409-e64234cc493c (fork e64234c); forest-shim replace unchanged at v0.0.0-20260930215433-f9eaf99c34a9"
  - ".cpy/.CPY files analyzed with Analyzer.AnalyzeCopybook under cobolAnalyzeSlot, behind the same approved-grammar check as programs"
  - "Copybook-owned data items <copybook path>::<01 name>/<child> with the member's own coordinates and the full document prov_* keys"
  - "cobolDataBind: the REDEFINES/RENAMES pass shared by program and copybook owners"
affects: [phase-04-copybook-nesting, cobol-extractor, resolver, verify-work]

actuals:
  tokens: 5669
  tasks: 2
  commits: 3
plan_head_before: 659e3d144eca068c587905a16ba862f7695c4c48

tech-stack:
  added: []
  patterns:
    - "Copybook mode: one shared Extract path (analysis slot, grammar check, docMeta) branching only on which producer entry point runs and which walkers follow"
    - "An empty parentID in cobolDataSection means the copybook file node owns 01/77 items and takes COPY marks"
    - "The copybook walk gets a cloned docMeta, so marks written on the file node never leak into item Meta"

key-files:
  created: []
  modified:
    - go.mod
    - go.sum
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_data.go
    - internal/parser/languages/cobol_grammar_test.go
    - cmd/gortex/cobol_tracer_test.go
    - docs/languages.md

key-decisions:
  - "Preprocessor re-pinned to v0.0.0-20261002100409-e64234cc493c (fork e64234c); forest-shim replace, cobolApprovedGrammarID, and cobolGrammarExtractorVersion gortex-cobol-grammar/3 unchanged."
  - "The REDEFINES/RENAMES loop moved out of cobolDataItems into cobolDataBind (two callers: programs and the copybook branch), so copybook items get the same binding pass (03-05 carry-forward)."
  - "The copybook walk receives maps.Clone(docMeta): COPY marks and the unleveled count land on the file node without being cloned into later items; cobol_kind/cobol_name_key/cobol_name_norm are set on the file node only, after the walk."

patterns-established:
  - "Copybook results contain only the file node and data_item nodes; wrapper program/division/section/paragraph/statement nodes are never emitted"

requirements-completed: [DET-01]

coverage:
  - id: D1
    description: "Only the preprocessor module is re-pinned (go.mod one line, go.sum two preprocessor lines); forest-shim replace and approved grammar ID unchanged; GOWORK=off build succeeds; Windows closure has no fork preprocessor"
    requirement: DET-01
    verification:
      - kind: other
        ref: "Task 1 verify 1 (go.work absent, forest replace at f9eaf99 shim, PV != f9eaf99, constant names PV, grammar ID present, GOWORK=off go build -o /tmp/gortex-phase3 ./cmd/gortex/)"
        status: pass
      - kind: other
        ref: "GOOS=windows CGO_ENABLED=1 GOWORK=off go list -deps ./cmd/gortex/ | ! grep tree-sitter-cobol-upgrade/preprocessor"
        status: pass
    human_judgment: false
  - id: D2
    description: "Copybooks run AnalyzeCopybook, pass the approved-grammar check, and carry full document provenance under cobol-grade-policy-copybook-v1 with no prov_analysis_absence"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_CopybookDataItems"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_CopybookGrammarPinned"
        status: pass
    human_judgment: false
  - id: D3
    description: "Copybook data items owned by the copybook with member rows 0-2 and exact ranges; no wrapper nodes; procedure fragments yield only the file node; pseudo-text members degrade; nested COPY marks group or file node without leaking"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_CopybookDataItems"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_CopybookProcedureFragment"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_CopybookReplacingDegrades"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_CopybookNestedCopy"
        status: pass
    human_judgment: false
  - id: D4
    description: "Real MultiIndexer + SQLite + resolver + MCP path persists and serves <prefix>/copy/custrec.cpy::CUST-REC, and COPY CUSTREC still binds to the copybook"
    requirement: DET-01
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_tracer_test.go#TestCobolTracer_CopybookDataItems"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'"
        status: pass
    human_judgment: false
  - id: D5
    description: "Phase 3 gate: the full repository suite passes under -race"
    requirement: DET-01
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 -timeout 30m ./... (exit 0, 156 ok packages)"
        status: pass
    human_judgment: false

duration: 24min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 13: Copybook-Mode Re-pin and Copybook-Owned Data Items Summary

**Gortex now pins the fork preprocessor at `v0.0.0-20261002100409-e64234cc493c` and analyzes `.cpy` members with `Analyzer.AnalyzeCopybook`. Each copybook owns its data items (`<path>::CUST-REC/CUST-ID`) with the member's own coordinates and the copybook grade policy. The forest-shim grammar pin and `cobolApprovedGrammarID` are unchanged.**

## Performance

- **Duration:** about 24 min (about 13 of them were the full `-race` suite)
- **Started:** 2026-10-02T13:08:59Z
- **Completed:** 2026-10-02T13:32:59Z
- **Tasks:** 2 of 2
- **Files modified:** 7

## Pinned Producer

- Preprocessor: `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor v0.0.0-20261002100409-e64234cc493c`, which is fork commit e64234cc493c487393c5e7867adfd7ca92d936ba. Resolved with `go list -m -json ...@e64234c` under `GOWORK=off`, `GOPRIVATE`, and a per-command `GIT_CONFIG_*` SSH rewrite. Nothing was written to global git config.
- Forest-shim replace: unchanged at `v0.0.0-20260930215433-f9eaf99c34a9`.
- `git diff c62d8f75^ c62d8f75 -- go.mod go.sum` changes only the preprocessor require line and the two preprocessor go.sum lines. `go mod tidy` added no other hunks.

## Accomplishments

- `Extract` decides copybook versus program up front. Both paths share the analysis slot, the approved-grammar check, and one `docMeta` builder. A wrong grammar pin now fails copybooks with `errCobolGrammarNotApproved`, which mitigates T-03-41.
- The copybook branch runs only the data-section walker (T-03-43). It calls `cobolDataSection(w, idx, filePath, "", "", s)` for each data-division section, then `cobolDataBind`, then `cobolDedupeEdges`. It never emits the wrapper's program, divisions, or sections.
- In `cobolDataSection`, an empty `parentID` makes the copybook file node stand in for the section node. 01 and 77 items get only the file `defines` edge. A COPY with no open item marks the file node with `cobol_contains_copy`.
- The copybook file node has lost `prov_analysis_absence`. It now carries every document `prov_*` key plus `cobol_kind=copybook`, `cobol_name_key`, and `cobol_name_norm`, and COPY binding still resolves through those keys.
- The `docs/languages.md` COBOL row now says copybooks are analyzed in the producer's copybook mode and own their data items. The Windows sentence is kept.

## Task Commits

1. **Task 1 RED:** `581cca76` (test). Added `TestCobolTracer_CopybookDataItems` and moved the `prov_preprocessor_module` expectation to the new pseudo-version.
2. **Task 1 GREEN:** `c62d8f75` (feat). The re-pin, the copybook branch, the copybook mode in `cobolDataSection`, and the `cobolDataBind` extraction.
3. **Task 2:** `2323ecab` (test). Replaced `TestCobolGrammar_CopybookSkipsAnalysis` with the four planned tests plus `TestCobolGrammar_CopybookNestedCopy`, and updated the docs row.

**Plan metadata:** docs commit (this SUMMARY, STATE.md, ROADMAP.md, REQUIREMENTS.md).

## TDD Evidence

- **Task 1 RED (valid):** before the re-pin, `TestCobolTracer_CopybookDataItems` failed on its target assertion `copybook data item repo/copy/custrec.cpy::CUST-REC missing`. `TestCobolTracer_PersistsThroughAddBatch` failed on `prov_preprocessor_module` (expected the e64234c pseudo-version, got the f9eaf99 one). **GREEN:** every `TestCobolTracer_*` and `TestCobolBreadth_*` test passed under `-race`.
- **Task 2:** its implementation shipped in Task 1 by plan design, so the new unit tests passed as soon as they were written. To show they can fail, I ran two mutations. Passing the uncloned `docMeta` to the walk, and dropping the file-node fallback in `cobolDataSection`, each made `TestCobolGrammar_CopybookNestedCopy` fail. Both mutations were reverted, and `git diff HEAD` was clean afterwards.
- `gsd check tdd-red-evidence` cannot read Go test output (the same limitation 03-11 hit), so the evidence above comes from `go test` output.

## Files Created/Modified

- `go.mod`, `go.sum`: the preprocessor re-pin only.
- `internal/parser/languages/cobol_grammar.go`: the re-pinned `cobolApprovedPreprocessorModule`, the shared analysis path with `AnalyzeCopybook`, and the copybook data branch (299 lines).
- `internal/parser/languages/cobol_grammar_data.go`: `cobolDataBind`, plus the copybook owner fallback in `cobolDataSection` (312 lines).
- `internal/parser/languages/cobol_grammar_test.go`: the copybook tests and invented fixtures.
- `cmd/gortex/cobol_tracer_test.go`: `TestCobolTracer_CopybookDataItems` and the `cobolTracerApprovedPreprocessorModule` constant.
- `docs/languages.md`: the COBOL row's copybook sentence.

## Decisions Made

See key-decisions in the frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The copybook walk gets a cloned docMeta**
- **Found during:** Task 1
- **Issue:** In copybook mode, the file node's Meta is `docMeta`, and `symbolNode` clones `docMeta` into every item. So a COPY mark (or the unleveled count) written on the file node partway through the walk would leak into every later item's Meta.
- **Fix:** `newCobolWalk(h, src, filePath, maps.Clone(docMeta), nil, result)`.
- **Files modified:** internal/parser/languages/cobol_grammar.go
- **Verification:** `TestCobolGrammar_CopybookNestedCopy` asserts that only `CUST-REC/CUST-ADDR` and the file node are marked, and the mutation check shows it fails without the clone.
- **Committed in:** c62d8f75

**2. [Rule 2 - Missing critical] The REDEFINES/RENAMES pass runs for copybook owners**
- **Found during:** Task 1 (03-05 carry-forward note)
- **Issue:** The binding pass lived inside `cobolDataItems`, and the copybook branch does not call that function.
- **Fix:** I extracted the pass as `cobolDataBind(w, idx)` in the same file and called it from both `cobolDataItems` and the copybook branch.
- **Files modified:** internal/parser/languages/cobol_grammar_data.go, internal/parser/languages/cobol_grammar.go
- **Verification:** every existing REDEFINES/RENAMES test in `TestCobol*` still passes under `-race`.
- **Committed in:** c62d8f75

**3. [Rule 2 - Missing critical] Added `TestCobolGrammar_CopybookNestedCopy`**
- **Found during:** Task 2
- **Issue:** must-have truth 6 (a COPY inside a copybook marks the innermost open item, else the file node) had no planned test.
- **Fix:** I added an invented fixture and a test for it. The plan's `-list` check only requires the five named tests to exist, so the extra test does not conflict with it.
- **Committed in:** 2323ecab

---

**Total deviations:** 3 auto-fixed (1 bug, 2 missing critical).
**Impact on plan:** No scope change. The grammar pin, the extractor version, and the plan's APIs are as specified.

## Issues Encountered

- Line budgets: `cobol_grammar.go` is at 299 lines and `cobol_grammar_data.go` at 312. The data file goes past the 300-line signal by 12 lines because `cobolDataBind` has its own doc comment and signature. The next real seam is data-name resolution (`qualifiedName`, `lookup`, `cobolDataReference`, `cobolDataBind`), which could move into its own file if this file grows again.
- `gofmt -l internal/parser/languages/` lists pre-existing unformatted non-COBOL files. They are out of scope and untouched. Every file this plan changed is gofmt-clean.

## Known Stubs

None.

## Threat Flags

None. No new surface beyond the plan's threat model: T-03-41 (`CopybookGrammarPinned`), T-03-42 (env-only GOPRIVATE and GIT_CONFIG_*), T-03-43 (data walker only, no wrapper nodes), and T-03-SC (diff limited to the preprocessor lines) were all mitigated as planned.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- Phase 3 gate: `GOWORK=off go test -race -count=1 -timeout 30m ./...` exited 0, the `GOWORK=off` binary builds, and the Windows closure has no fork preprocessor. Ready for `/gsd-verify-work 03`.
- Deferred to Phase 4, as the plan flagged: copybook-to-copybook import edges for nested COPY, library search order, and paragraphs and statements owned by procedure-fragment copybooks.

## Self-Check: PASSED

- FOUND: go.mod, go.sum, internal/parser/languages/cobol_grammar.go, internal/parser/languages/cobol_grammar_data.go, internal/parser/languages/cobol_grammar_test.go, cmd/gortex/cobol_tracer_test.go, docs/languages.md
- FOUND commits: 581cca76, c62d8f75, 2323ecab (3 commits since plan_head_before 659e3d14, measured)

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
