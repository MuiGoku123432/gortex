---
phase: 05-explicit-unresolved-evidence
plan: 03
subsystem: cobol-graph
tags: [cobol, finding, parse-unresolved, cascade, copybook, extractor-version]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "cobolFindings(w), graph.FindingSymbol, finding Meta key constants (05-01)"
provides:
  - "cobolCascadeFamily (grade.ReasonDivisionScaleRecovery, ReasonRetainedStructureCollapse, ReasonMissingRequiredAnchor, ReasonLostCapture)"
  - "One uncontained-cascade PARSE_UNRESOLVED document finding per cascaded document (severity high, scope document, prov_range_absence=document_scope, finding_producer_reasons)"
  - "cobolAddFinding: the one private node-building path shared by producer gaps and the cascade finding"
  - "Standalone copybook parse findings: cobolFindings(w) runs last in cobolCopybook, after the WR-04 copybook_content_unmapped check"
  - "cobolGrammarExtractorVersion = gortex-cobol-grammar/6 (Phase 5 mapping)"
affects: [05-04, 05-05, 05-07, 05-10, 05-11, 05-12, phase-06]

actuals:
  tokens: 7900
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Gortex-derived document findings are built as a handoff.Gap value (Container -1, AbsenceReason document_scope) passed through the same cobolAddFinding path, with a constant evidence string"
    - "Cascade detection reads only h.Assessment.Reasons against the producer grade constants; nothing is re-derived from gaps"

key-files:
  created: []
  modified:
    - internal/parser/languages/cobol_grammar_findings.go
    - internal/parser/languages/cobol_grammar_copybook.go
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_findings_test.go
    - internal/parser/languages/cobol_grammar_copybook_test.go
    - internal/parser/languages/cobol_grammar_contract_test.go
    - cmd/gortex/cobol_tracer_test.go
    - cmd/gortex/cobol_breadth_test.go

key-decisions:
  - "The cascade finding goes through the same private builder as producer gaps (a handoff.Gap with Kind/Reason uncontained-cascade, SeverityHigh, ScopeDocument, Container -1, AbsenceReason document_scope); its evidence is the constant uncontained-cascade, so producer reasons never reach the ID"
  - "An empty source grades red [missing-required-anchor], so it now carries one uncontained-cascade finding on the file node; D-13 is applied as written with no empty-input special case, and the 05-01 finding-aware EmptySource assertions already admit it"
  - "The four cascade-family tests carry their own family list (cobolTestCascadeReasons) so a wrong extractor list fails them"

patterns-established:
  - "A Gortex-derived COBOL finding is a synthetic handoff.Gap passed to cobolAddFinding with a constant evidence string"
  - "Copybook tests that predate findings count non-finding nodes and require every remaining edge to come from a finding"

requirements-completed: [GAP-01, GAP-04, GAP-08]

coverage:
  - id: D1
    description: "Each probe-verified cascade shape on the 20/20 control gets exactly one uncontained-cascade finding (high, document scope, document_scope absence, producer reasons copied) attached to the single program, plus one finding per localized producer parse gap; each family reason alone triggers it"
    requirement: GAP-08
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_CascadePositive"
        status: pass
    human_judgment: false
  - id: D2
    description: "Contained shapes (clean, EXEC SQL SELECT tail, unbalanced COMPUTE, EXEC CICS SEND MAP, OBTAIN after a paragraph) and red documents without a family reason get no uncontained-cascade finding"
    requirement: GAP-08
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_CascadeNegative"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two cascade shapes with different producer reasons in the same container share one uncontained-cascade finding ID"
    requirement: GAP-04
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_CascadeIDStable"
        status: pass
    human_judgment: false
  - id: D4
    description: "Standalone copybooks emit one finding per producer parse gap plus the cascade finding, all member_of the copybook file node, without cobol_kind; the WR-04 unmapped flag is still decided on member nodes only"
    requirement: GAP-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_Copybook"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_CopybookUnmappedFlag"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookWrapperWindow"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every COBOL node and edge is stamped gortex-cobol-grammar/6 and the material fingerprint tests still pass"
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_contract_test.go#TestCobolGrammar_ProvKeys"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/indexer/ -run 'CobolMaterial|^TestCobolLifecycle_Material'"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex/ -run '^TestCobol(Tracer|Breadth|Findings)_'"
        status: pass
    human_judgment: false

duration: 14min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 03: Cascade and Copybook Parse Findings Summary

**A COBOL document graded with a D-13 cascade-family reason now carries one high-severity, document-scope `uncontained-cascade` finding whose ID ignores the producer reasons, standalone copybooks emit the same parse findings as programs without disturbing the WR-04 unmapped flag, and the extractor is stamped `gortex-cobol-grammar/6` so existing stores re-extract.**

## Performance

- **Duration:** ~14 min
- **Started:** 2026-10-20T16:11:35Z
- **Completed:** 2026-10-20T16:25:06Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- `cobolCascadeFamily` holds the four producer grade constants (no literals). At the end of `cobolFindings`, a document whose `h.Assessment.Reasons` holds any of them gets one finding: kind and reason `uncontained-cascade`, severity `high` with `gortex-finding-severity/1`, scope `document`, `prov_range_exact=false`, `prov_range_absence=document_scope`, `finding_producer_reasons` a fresh copy of the reasons, container by the document-scope rule (single named program, else file node, copybook root maps to the file node).
- The gap loop and the cascade share one private builder, `cobolAddFinding`. The cascade is passed as a `handoff.Gap` value with the constant evidence `uncontained-cascade`, so its ID depends only on container, class, kind, and that constant.
- `cobolCopybook` calls `cobolFindings(w)` as its last statement, after the `copybook_content_unmapped` check. The walk's docMeta is the pre-walk clone, so copybook findings carry no `cobol_kind`, `cobol_name_key`, or `prov_analysis_absence`.
- `cobolGrammarExtractorVersion` is `gortex-cobol-grammar/6`. Its comment records the Phase 5 mapping and that 05-10 keeps `/6`. All seven pinned `/5` assertions now read `/6`.

## Task Commits

1. **Task 1: Uncontained-cascade document finding and copybook findings** - `09c02024` (test, RED), `30451994` (feat, GREEN)
2. **Task 2: Bump the COBOL extractor version to /6 once for the phase** - `ad07d3e8` (feat)

**Plan metadata:** recorded in the docs(05-03) commit that adds this file.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar_findings.go` - `cobolCascadeKind`, `cobolCascadeFamily`, the cascade branch in `cobolFindings`, and `cobolAddFinding` (the gap loop body moved here unchanged)
- `internal/parser/languages/cobol_grammar_copybook.go` - `cobolFindings(w)` as the last statement; doc comment names the findings
- `internal/parser/languages/cobol_grammar.go` - version `/6` and its comment
- `internal/parser/languages/cobol_grammar_findings_test.go` - the 20/20 control builder, six positive shapes, the copybook fixture, and the five `TestCobolGrammar_Findings_*` tests with their helpers
- `internal/parser/languages/cobol_grammar_copybook_test.go` - the procedure-fragment node count and the unmapped-member node and edge checks exclude findings
- `internal/parser/languages/cobol_grammar_contract_test.go`, `cmd/gortex/cobol_tracer_test.go`, `cmd/gortex/cobol_breadth_test.go` - `/5` to `/6` only

## Decisions Made

- The cascade finding reuses the gap builder through a `handoff.Gap` value instead of a second builder. Every Meta key, the `#k` counter, the range absence override, and the `member_of` edge therefore stay identical for both paths.
- The tests list the cascade family independently (`cobolTestCascadeReasons`). They also add synthetic subtests: each family reason alone triggers the finding, and a red document whose reasons are `parse-timeout`, `no-tree`, or `unmapped-capture` (plus amber codes) gets none.
- Producer observations from a probe on this build, kept for later plans:
  - The six program shapes grade red, five with `[lost-capture division-scale-recovery error-node affected-capture]`. `DATA DIVISION` without its period adds `missing-required-anchor` first. Each shape has 21-23 findings.
  - The copybook `05 CUST-ID PIC X(8) WHATEVER.` member gets 3 findings and still maps member nodes.
  - The `EXEC CICS RETURN` member grades red `[division-scale-recovery ...]`. It gets 2 findings and stays unmapped.

## Deviations from Plan

None - plan executed exactly as written. The wave-merge check passed with no count-only test fixes.

## TDD Gate Compliance

Task 1 followed RED then GREEN. In `09c02024`, all five new tests ran and failed on missing findings, and their preconditions (grade, cascade reasons, single program) passed. `30451994` made them pass. Task 2 is not a TDD task.

## Issues Encountered

- An empty source grades red `[missing-required-anchor]`. It now gets one `uncontained-cascade` finding on the file node, because there is no program to attach it to. This follows the plan's flagged assumption: D-13 applies as written, with no special case for empty input. `TestCobolGrammar_EmptySource` and the empty tail of `TestCobolGrammar_Statements` already count only non-finding nodes and require every edge to come from a finding, so both pass unchanged.

## Verification

- Task 1 verify: all five tests listed; `go test -race ./internal/parser/languages/ -run CobolGrammar` passes; gofmt is clean on all four files.
- Task 1 acceptance:

  | Check | Result |
  |---|---|
  | `grade.Reason*` lines | 4 |
  | `"division-scale-recovery"` literal | 0 |
  | `cobolFindings(w)` in the last 5 lines of the copybook file | 1 |
  | `assert.Empty(t, result.Edges)` | 0 |

- Task 2 verify:
  - No `/5` literal remains in `internal` or `cmd`.
  - The COBOL grammar, material (`CobolMaterial|^TestCobolLifecycle_Material`), and `cmd/gortex` COBOL (`Tracer|Breadth|Findings`) suites pass under `-race`.
  - gofmt is clean.
  - The version-constant grep prints 1.
- Plan verification:
  - `go build ./...` passes.
  - `GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer/ ./cmd/gortex/ -run '^TestCobol'` passes (146 s and 45 s).
  - An extra `./internal/resolver -run Cobol` run passes.
  - `git diff --name-only b7962680..HEAD` lists exactly the 8 planned files.

## User Setup Required

None. No external service configuration is required. Running daemons and installed binaries carry `/5` until rebuilt; the material fingerprint then re-extracts every COBOL file.

## Next Phase Readiness

- 05-07 can gate on the `uncontained-cascade` finding on the real path; the six shapes and the 20/20 builder in `cobol_grammar_findings_test.go` are the extractor-level reference.
- 05-04 `index_health` counts should expect one cascade finding per cascaded document, including empty or anchorless sources.
- 05-10 must keep `/6` for WR-06, WR-07, and IN-08.

## Self-Check: PASSED

- FOUND: all 8 modified files on disk
- FOUND commits: 09c02024, 30451994, ad07d3e8
- Both task verify commands, every acceptance criterion, and the plan-level wave-merge check passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
