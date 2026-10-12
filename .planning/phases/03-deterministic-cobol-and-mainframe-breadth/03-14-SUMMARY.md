---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 14
subsystem: cobol-resolver
tags: [cobol, resolver, nested-programs, call-binding, gap-closure, tdd]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: "03-04 resolveCobolReferences exactly-one binding pass; 03-07 END PROGRAM containment IDs; 03-13 breadth real-path harness"
provides:
  - "cobolProgramSymbol.nested set from the END PROGRAM containment stack"
  - "cobol_nested (bool) Meta on every COBOL program node"
  - "resolveCobolReferences offers a nested program only to callers in its own file"
  - "Resolver, extractor, and persisted real-path tests for nested CALL scope"
affects: [phase-04-incremental-cobol, cobol-call-binding, neo4j-projection]

actuals:
  tokens: 3700
  tasks: 2
  commits: 3
plan_head_before: 8f9b1aea5e2a6b31b1406346a192bb503744486d

tech-stack:
  added: []
  patterns:
    - "Scope facts the resolver needs are stamped as explicit node Meta at extraction; the resolver never parses ID or QualName shape"

key-files:
  created: []
  modified:
    - internal/parser/languages/cobol_grammar_identity.go
    - internal/parser/languages/cobol_grammar.go
    - internal/resolver/cobol_references.go
    - internal/resolver/cobol_references_test.go
    - internal/parser/languages/cobol_grammar_test.go
    - cmd/gortex/cobol_breadth_test.go

key-decisions:
  - "Nested-ness is an explicit cobol_nested Meta stamp derived from END PROGRAM containment (len(open) > 0), not a QualName '/' check: a top-level literal PROGRAM-ID such as \"A/B\" already has '/' in its verbatim symbol"
  - "Program candidates: same-file first (nested or not, D-09 unchanged); otherwise only top-level programs from other files. A sole nested cross-file match falls to missing (fail closed)"
  - "An edge with empty FilePath gets no same-file list, so nested programs are excluded for it rather than bound"

patterns-established:
  - "Resolver scope filters read extractor-stamped Meta flags, never node-ID shape"

requirements-completed: [DET-03, DET-04]

coverage:
  - id: D1
    description: "A cross-file literal CALL whose only same-repository match is a nested program stays on unresolved::cobol::program::<key> with reason missing and no candidate IDs"
    requirement: DET-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReferences_ProgramBinding/nested_cross_file_missing"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_NestedProgramScope"
        status: pass
    human_judgment: false
  - id: D2
    description: "Cross-file binding prefers the top-level program, and ambiguous candidate IDs never list a nested program from another file"
    requirement: DET-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReferences_ProgramBinding/nested_cross_file_ignored"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReferences_ProgramBinding/nested_excluded_from_candidates"
        status: pass
    human_judgment: false
  - id: D3
    description: "Same-file nested binding unchanged: OUTER's CALL 'INNER' binds to OUTER/INNER, in memory and on the persisted store"
    requirement: DET-03
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReferences_ProgramBinding/nested_same_file"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_references_test.go#TestCobolReferences_ProgramBinding/same_file_first"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_breadth_test.go#TestCobolBreadth_NestedProgramScope"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every program node carries cobol_nested from END PROGRAM containment, including false for a top-level literal name containing '/'"
    requirement: DET-03
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_ProgramNestedStamp"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_NestedIDs"
        status: pass
    human_judgment: false
  - id: D5
    description: "Dynamic, label, data, idms_scope, and COPY IDMS targets untouched; every existing TestCobol* test in resolver, languages, and cmd/gortex still passes"
    requirement: DET-04
    verification:
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/resolver/ -run '^TestCobol'"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobol(Grammar_|Extractor_)'"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 14: Nested COBOL Program CALL Scope Summary

**A separately compiled caller can no longer bind to a contained COBOL program: the extractor stamps `cobol_nested` from END PROGRAM containment, and `resolveCobolReferences` offers nested programs only to callers in their own file, proven on the persisted two-repository graph.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-10-12T14:18:19Z
- **Completed:** 2026-10-12T14:27:35Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- `cobolProgramSymbol.nested` is set from `len(open) > 0` after the close loop, the same condition that adds the container segment to the symbol. Every program node carries `cobol_nested` (bool) in Meta.
- `resolveCobolReferences` splits program candidates into same-file (nested or not) and other-file top-level (`cobol_nested` not true). It uses the same-file list when that list is non-empty, and otherwise the top-level list. A nested program in another file is never a candidate, so a cross-file CALL whose only match is nested stays `missing`.
- On the real MultiIndexer, SQLite store, and production resolver path, NESTCALL's `CALL 'INNER'` persists on `unresolved::cobol::program::INNER` with reason `missing` and no candidate IDs. OUTER's same-file call still binds to `OUTER/INNER`. Persisted `OUTER/INNER` has `cobol_nested=true` and `OUTER` has `false`.
- Closes VERIFICATION `gaps[0]` (review WR-02): both `missing` items (the candidate rule and the cross-file resolver test) are delivered.

## Task Commits

1. **Task 1 RED: failing nested-scope tests** - `a70a797b` (test)
2. **Task 1 GREEN: nested flag, Meta stamp, resolver filter** - `8d3996fa` (feat)
3. **Task 2: pin cobol_nested at the extractor** - `2d864984` (test)

No REFACTOR commit was needed.

## TDD Gate Compliance

- **RED (Task 1):** I converted the `go test -json` output to TAP and checked it with `gsd-tools check tdd-red-evidence`. Both records returned `RED_EVIDENCE_OK / target_test_failed`.
  - `TestCobolReferences_ProgramBinding/nested_cross_file_missing` failed on its assertion: expected `unresolved::cobol::program::INNER`, actual `a/nest.cbl::OUTER/INNER`. `nested_cross_file_ignored` and `nested_excluded_from_candidates` also failed on assertions. `nested_same_file` passed as intended, since it pins behavior that does not change.
  - `TestCobolBreadth_NestedProgramScope` failed because `cobol_nested` was absent (nil) and no `NESTCALL-PARA -> unresolved::cobol::program::INNER` edge existed (the edge was bound to OUTER/INNER).
- **GREEN (Task 1):** `8d3996fa`. All four `nested_*` subtests and the breadth test pass under `-race`.
- **Task 2 (pinning tests):** The tests pin behavior that Task 1 already implemented, so they passed on first run. The plan called for a mutation check to prove they bite:
  - **Mutation check:** I temporarily set `nested: strings.Contains(symbol, "/")` in `cobolProgramSymbols`. `TestCobolGrammar_ProgramNestedStamp/literal_slash_name` then failed (expected `{"\"A/B\"": false}`, actual `true`), and the other three subtests still passed.
  - I restored `len(open) > 0`. `git diff --stat internal/parser/languages/cobol_grammar_identity.go` was empty, so the file is unchanged from the Task 1 commit.
- **Tracer feedback gate (Task 1, `type="tracer"`):** The run was interactive, mode was `end-of-phase`, and `<verify>` was automated-only. The re-run verify passed, so execution continued to Task 2 without a checkpoint.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar_identity.go`: `nested` field on `cobolProgramSymbol`, set from the containment stack
- `internal/parser/languages/cobol_grammar.go`: `meta["cobol_nested"] = p.nested` on every program node (now 302 lines; Plan 03-15 brings it back under 300)
- `internal/resolver/cobol_references.go`: `metaCobolNested` constant, same-file-only rule for nested candidates, updated PURPOSE/RATIONALE doc
- `internal/resolver/cobol_references_test.go`: `nested_cross_file_missing`, `nested_cross_file_ignored`, `nested_same_file`, `nested_excluded_from_candidates`
- `internal/parser/languages/cobol_grammar_test.go`: `TestCobolGrammar_ProgramNestedStamp` (top_level, literal_slash_name, stolen_marker, siblings) and `cobol_nested` assertions in `TestCobolGrammar_NestedIDs`
- `cmd/gortex/cobol_breadth_test.go`: invented `cobolBreadthNestFixture` (`cobol/nest.cbl`) and `cobolBreadthNestCallFixture` (`cobol/nestcall.cbl`) in repository A, plus `TestCobolBreadth_NestedProgramScope`

## Decisions Made

- I used an explicit extraction stamp rather than a QualName `/` check. The rationale is the plan's: program symbols are verbatim `program_name` bytes (IN-03), so a top-level literal `PROGRAM-ID. "A/B".` already contains `/`. The `literal_slash_name` mutation check confirms this.
- Same-file binding is unchanged (D-09 "same file first"). This plan does not narrow same-file candidates further, for example a sibling top-level program calling another program's contained program.
- When a file carries `prov_containment_unresolved`, the nested flag and the ID share the same containment guess. Excluding such a program from cross-file candidates fails closed (`missing`).

## Deviations from Plan

None. The plan executed as written.

## Issues Encountered

- The Gortex MCP tools named in `<read_first>` (`smart_context`, `surface_memories`, `get_editing_context`, `query_notes`) were not mounted in this executor session. I read the source files with shell reads instead. This had no effect on the outcome.
- `gsd-tools check tdd-red-evidence` parses node TAP output. I converted the Go test JSON into TAP lines (one `ok`/`not ok` per test plus a `# tests/pass/fail` summary) so the checker could classify the RED runs.

## Verification

- Task 1 verify 1: `TestCobolBreadth_NestedProgramScope` is listed, and `go test -race ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'` passed (all 13 tests).
- Task 1 verify 2: the `nested_` PASS count is 4. `-race` resolver `^TestCobol` and languages `^TestCobol(Grammar_|Extractor_)` passed, `gofmt -l` is clean, and `go vet` is clean.
- Task 2 verify: both test names are listed, `-race` languages passed, and `gofmt` is clean.
- Acceptance greps: `cobol_nested` appears once in `cobol_grammar.go` and `metaCobolNested` twice in `cobol_references.go`.
- Extra checks: full `./internal/resolver/` and `./internal/parser/languages/` packages pass. `./internal/neo4jprojection/` passes, `./internal/indexer -run Cobol` passes, and `go build ./...` succeeds.
- `git diff --name-only 8f9b1aea..HEAD` lists exactly the six `files_modified`.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- VERIFICATION gap 1 is closed. Gap 2 (COPY edges inside an analyzed copybook) is planned in 03-15/03-16.
- `cobolGrammarExtractorVersion` stays `gortex-cobol-grammar/3` (Phase 3 is unreleased). Existing stores need a reindex before `cobol_nested` appears on their program nodes.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-12*

## Self-Check: PASSED

- All six modified files exist on disk.
- Commits `a70a797b`, `8d3996fa`, and `2d864984` are present in `git log`.
