---
phase: 05-explicit-unresolved-evidence
plan: 10
subsystem: cobol-graph
tags: [cobol, finding, parse-unresolved, copybook, cursor, nested-program, resolver, wr-06, wr-07, wr-09, in-08]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "cobolAddFinding and the document-scope finding path, gortex-cobol-grammar/6 (05-03); missing-artifact placeholders (05-05); EXTERNAL_UNRESOLVED findings only for missing and IDMS (05-08)"
provides:
  - "(*CobolGrammarExtractor).cobolCopybookAnalysisFailed: identity-only copybook file node (prov_analysis_absence=copybook_analysis_failed) plus one high analysis-failed PARSE_UNRESOLVED document finding (reason analysis-error)"
  - "Extract copybook analysis errors return the WR-06 result; program analysis, init, and grammar errors still fail (D-20)"
  - "Cursor use reason ambiguous when the cursor is declared two or more times in the program (WR-07)"
  - "Program Meta cobol_nested_unresolved=true on nested programs when END PROGRAM containment is unresolved (IN-08)"
  - "Resolver metaCobolNestedUnresolved and edge reason containment_unresolved (no placeholder, no finding)"
affects: [05-11, 05-12, phase-06]

actuals:
  tokens: 6900
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "A Gortex-derived document finding without a handoff is built by running cobolAddFinding over a walk of an empty handoff.Handoff; cobolFindingContainer then attaches it to the file node"
    - "Resolver reasons that mean 'the artifact exists' (ambiguous, containment_unresolved) keep To on the unresolved:: target and never reach reconcileCobolFindings"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_reasons_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_findings.go
    - internal/parser/languages/cobol_grammar_sql.go
    - internal/parser/languages/cobol_grammar_sql_test.go
    - internal/resolver/cobol_references.go
    - internal/resolver/cobol_references_test.go

key-decisions:
  - "WR-06 finding Meta is the pre-identity docMeta clone, so the finding carries the prov stamps but no cobol_kind, cobol_name_key, or prov_analysis_absence, like 05-03 copybook findings"
  - "WR-06 file node stamps prov_parser_grammar_id with the approved grammar ID (the pin the extractor enforces) and invents no handoff-derived key (tool ID, parse/transform config, schema, grade)"
  - "IN-08 counts a dropped candidate as guessed whether it was dropped as a non-container nested program in the caller repository or as a nested program of another allowed repository; containment_unresolved fires only when the candidate list ends up empty"
  - "WR-07: a copybook that declares a cursor twice also gets ambiguous; cursor_not_in_copybook stays only for zero declarations"

patterns-established:
  - "Extractor reasons that keep an artifact present (WR-06 identity node, IN-08 stamp) are the guard against false EXTERNAL_UNRESOLVED findings; the resolver adds no finding for them"

requirements-completed: [GAP-01, GAP-02, GAP-03]

coverage:
  - id: D1
    description: "A copybook whose analysis fails yields exactly the identity file node and one high analysis-failed document finding with a member_of edge; the error text never reaches Name or Meta; the ID is stable across reruns"
    requirement: GAP-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_reasons_test.go#TestCobolGrammar_CopybookAnalysisFailed"
        status: pass
    human_judgment: false
  - id: D2
    description: "A program whose analysis fails still returns an error and no result (D-20)"
    requirement: GAP-03
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_reasons_test.go#TestCobolGrammar_ProgramAnalysisFailedStillFails"
        status: pass
    human_judgment: false
  - id: D3
    description: "A twice-declared cursor is ambiguous, an undeclared one cursor_not_declared, a copybook's undeclared one cursor_not_in_copybook (WR-07, WR-09)"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_reasons_test.go#TestCobolGrammar_CursorAmbiguous"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_sql_test.go#TestCobolGrammar_SQL/cursor_unknown"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_copybook_test.go#TestCobolGrammar_CopybookProcedureMainframe"
        status: pass
    human_judgment: false
  - id: D4
    description: "Nested programs under unresolved END PROGRAM containment carry cobol_nested_unresolved=true; top-level programs and well-nested documents carry no key"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_reasons_test.go#TestCobolGrammar_NestedContainmentUnresolved"
        status: pass
    human_judgment: false
  - id: D5
    description: "A CALL whose only candidate is a stamped nested program gets containment_unresolved, keeps its unresolved:: target, and creates no placeholder or finding; an unstamped one stays missing; a later stamp moves a missing CALL off its placeholder and removes its finding (memory and SQLite)"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReconcile_ContainmentUnresolved"
        status: pass
    human_judgment: false
  - id: D6
    description: "A COPY of a copybook whose analysis failed binds to its identity node, with no EXTERNAL_UNRESOLVED finding or placeholder (memory and SQLite)"
    requirement: GAP-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReconcile_AnalysisFailedCopybookBinds"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 10: Scope-Accurate Unresolved Reasons Summary

**The deferred Phase 3 items are closed so no finding inherits a false reason. A copybook whose analysis fails is now a present copybook with a high `analysis-failed` `PARSE_UNRESOLVED` finding (WR-06). A twice-declared cursor is `ambiguous` (WR-07), and WR-09's `cursor_not_in_copybook` is still green. A CALL that only a guessed nesting hid is `containment_unresolved`, not `missing`, so it gets no placeholder or `EXTERNAL_UNRESOLVED` finding (IN-08).**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-10-20T19:05:40Z
- **Completed:** 2026-10-20T19:20:29Z
- **Tasks:** 2
- **Files modified:** 7 (1 created)

## Accomplishments

- **WR-06.** `Extract`'s copybook error branch returns `cobolCopybookAnalysisFailed(filePath, src, sourceID)`.
  - The file node has `KindFile`, lines 1 to the line count + 1, and the copybook binding keys (`cobol_kind`, `cobol_name_key`, `cobol_name_norm`).
  - It also carries the stamps known without a handoff: source path and ID, `NewContentID` revision, approved grammar ID, both modules, extractor version `/6`, evidence class, origin, and confidence.
  - It sets `prov_analysis_absence=copybook_analysis_failed`.
  - One finding goes through the existing `cobolAddFinding`: kind `analysis-failed`, reason `analysis-error`, severity `high`, scope `document`, and `prov_range_absence=document_scope`. It has a `member_of` edge to the file node.
  - The producer's error text is dropped (T-05-32).
- **D-20.** Program analysis errors, init errors, and the grammar check still fail. The program branch keeps `enhanced COBOL analysis of %s`.
- **WR-07.** In `cobol_grammar_sql.go`, a cursor with two or more same-program DECLAREs is now stamped `ambiguous` on its use statement. The `cursor_unknown` C2 case expects it. C9 keeps `cursor_not_declared`, and the copybook WR-09 test still passes.
- **IN-08 stamp.** The extractor sets `cobol_nested_unresolved=true` only on nested programs of a document whose containment is unresolved. Well-nested dumps are unchanged.
- **IN-08 decision.** `decideCobol` reads `metaCobolNestedUnresolved`. It tracks a candidate dropped only for its nesting while the stamp is set. If the program candidate list ends up empty and such a candidate was dropped, it returns `containment_unresolved`. The edge keeps its `unresolved::` target. `reconcileCobolFindings` acts only on missing and IDMS, so it creates nothing.

## Task Commits

1. **Task 1: Extractor reasons (WR-06, WR-07, IN-08 stamp)**
   - `53346946` (test, RED): CopybookAnalysisFailed, CursorAmbiguous, NestedContainmentUnresolved, and SQL/cursor_unknown failed. ProgramAnalysisFailedStillFails passed, because it is a D-20 guard on existing behavior.
   - `947bfaac` (feat, GREEN)
2. **Task 2: Resolver containment_unresolved**
   - `00bc2da6` (test, RED): the `stamped` and `missing_then_stamped` subtests failed on both stores. `unstamped_missing` and AnalysisFailedCopybookBinds passed: they guard behavior the resolver already has.
   - `e2396544` (feat, GREEN)

**Plan metadata:** the docs(05-10) commit that adds this file.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar.go`:
  - the copybook error branch returns the WR-06 result;
  - the program Meta loop stamps `cobol_nested_unresolved`;
  - the `Extract` doc names the fallback.
- `internal/parser/languages/cobol_grammar_findings.go`: `cobolAnalysisFailedKind` and `cobolAnalysisFailedReason`, plus `(*CobolGrammarExtractor).cobolCopybookAnalysisFailed`.
- `internal/parser/languages/cobol_grammar_sql.go`: the `ambiguous` cursor reason and its doc sentence.
- `internal/parser/languages/cobol_grammar_reasons_test.go` (new): the four `TestCobolGrammar_*` tests and the `cobolFailingAnalyzer` seam.
- `internal/parser/languages/cobol_grammar_sql_test.go`: `cursor_unknown` now expects C2 `ambiguous`.
- `internal/resolver/cobol_references.go`:
  - the `metaCobolNestedUnresolved` const;
  - the `guessed` tracking and the `containment_unresolved` return;
  - the D-12 vocabulary comment and the `reconcileCobol` doc sentence.
- `internal/resolver/cobol_references_test.go`: `seedCobolNode`, `TestCobolReconcile_ContainmentUnresolved`, and `TestCobolReconcile_AnalysisFailedCopybookBinds`.

## Decisions Made

- **WR-06 finding Meta.** The finding clones the docMeta from before the identity keys were added. It carries no `cobol_kind` or `prov_analysis_absence`, matching how 05-03 copybook findings avoid file-node keys.
- **Copybook binding keys.** `cobolCopybookAnalysisFailed` repeats the three keys `cobolCopybook` writes, with a `// note:` to keep them in step. Two occurrences stay duplicated (rule of three).
- **IN-08 scope.** Both drop sites count as guessed: a same-repository nested program the caller does not contain, and a nested program of another allowed repository. If the nesting were wrong, either could be a top-level candidate. The reason fires only when no candidate remains. A remaining bind or ambiguity stands, per the plan.
- **WR-07 in a copybook.** A copybook that declares a cursor twice also gets `ambiguous`. `cursor_not_in_copybook` stays for zero declarations only.
- **Extractor version.** `prov_extractor_version` stays `gortex-cobol-grammar/6`.

## Deviations from Plan

None. The plan executed as written. The diff covers exactly the 7 planned files.

## TDD Gate Compliance

- **Task 1:** RED `53346946`, then GREEN `947bfaac`.
- **Task 2:** RED `00bc2da6`, then GREEN `e2396544`.
- **Tests green before implementation:** `TestCobolGrammar_ProgramAnalysisFailedStillFails` and `TestCobolReconcile_AnalysisFailedCopybookBinds` passed at RED. Both guard behavior that must not change: D-20 hard failure, and the resolver binding any `KindFile` copybook. In each RED commit, the other new tests failed for the intended reason.

## Issues Encountered

None.

## Verification

- **Task 1 verify:**
  - All five listed tests exist.
  - `go test -race ./internal/parser/languages/ -run CobolGrammar` is ok.
  - gofmt is clean.
  - Acceptance greps: `cobolCopybookAnalysisFailed(` 1, `enhanced COBOL analysis of %s` 1, `"ambiguous"` 1, `cursor_not_in_copybook` 2, and `cobol_nested_unresolved` in `cobol_grammar.go` 1.
- **Task 2 verify:**
  - Both tests are listed.
  - `go test -race ./internal/resolver/` (whole package) is ok.
  - `go test -race ./cmd/gortex/ -run '^TestCobolBreadth_'` is ok.
  - gofmt is clean.
  - Acceptance greps: `"containment_unresolved"` 1, `metaCobolNestedUnresolved` 2.
- **Plan verification:**
  - `go build ./...` is ok.
  - `GOWORK=off go test -race -count=1 -timeout 30m ./internal/parser/languages ./internal/resolver ./internal/indexer -run 'Cobol'` is ok (40 s, 28 s, 212 s).
  - Extra: `./cmd/gortex/ -run '^TestCobol'` is ok (45 s).
  - `git diff --name-only b9f58755..HEAD` lists only the 7 planned files.

## User Setup Required

None. No external service configuration is required. Running daemons pick up the new reasons when rebuilt. The extractor version did not change, so existing stores re-extract a file only when it changes or when the material fingerprint moves for another reason.

## Next Phase Readiness

- 05-11 and 05-12 can rely on these meanings:
  - `missing` and `EXTERNAL_UNRESOLVED` mean "not obtained", except for D-20 (a program whose analysis fails).
  - `analysis-failed` is the copybook parse-failure finding.
- 05-12 Task 3 documents D-20 in `docs/languages.md`. It can also list `containment_unresolved` in the reason vocabulary.

## Self-Check: PASSED

- FOUND: all 7 planned files on disk (1 created, 6 modified)
- FOUND commits: 53346946, 947bfaac, 00bc2da6, e2396544
- Both task verify commands, every acceptance criterion, and the plan-level verification passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
