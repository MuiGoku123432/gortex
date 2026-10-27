---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 02
subsystem: parser
tags: [cobol, tree-sitter, identity, id-encoding, edge-meta, resolver-prereq]

requires:
  - phase: 03-deterministic-cobol-relationships
    provides: cobolIDSegment, cobolNameNorm, literal CALL/COPY edges, cobolDedupeEdges occurrence collapse
provides:
  - Program ID components encoded with cobolIDSegment (ID-03, IN-03)
  - Edge Meta cobol_placeholder on literal CALL/LINK/XCTL and non-IDMS COPY/-INC/SQL INCLUDE edges (D-18)
  - Edge Meta cobol_copy_library_key = cobolNameNorm(OF/IN library) (D-10)
  - TestCobolGrammar_ProgramIDEncoding, TestCobolGrammar_Placeholder, TestCobolGrammar_Occurrences
affects: [04-06 reconcile pass, 04-08 CR-01 restub, 04-10 sensitive-metadata screen, 04-11 mapping version bump]

actuals:
  tokens: 3654
  tasks: 2
  commits: 4
plan_head_before: ef81beed4b23dbecccd68ab5253ecd9cc95a2e82

tech-stack:
  added: []
  patterns:
    - "Bindable COBOL edges carry their original owned-namespace target in cobol_placeholder; consumers read it instead of re-normalizing names"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_identity_test.go
  modified:
    - internal/parser/languages/cobol_grammar_identity.go
    - internal/parser/languages/cobol_grammar_refs.go
    - internal/parser/languages/cobol_grammar_test.go

key-decisions:
  - "Literal PROGRAM-ID quotes stay in the ID component: the pinned grammar keeps them in program_name, so 'A/B' becomes {path}::'A%2FB' (not {path}::A%2FB); IDs keep verbatim names per D-20 and only the ID metacharacters are encoded"
  - "Occurrence provenance (ID-05) is proven, not changed: same-line duplicates already resolve to distinct KindLocal statement nodes with their own start columns, so IN-04 dedupe stays as is"

patterns-established:
  - "cobol_placeholder: set only where an edge can later be bound (literal program, non-IDMS copybook); its absence marks a never-bound edge"

requirements-completed: [ID-03, ID-05]

coverage:
  - id: D1
    description: "Every program ID component passes through cobolIDSegment, so literal PROGRAM-IDs 'A/B' and 'X#2' never alias a nested path or a #k ordinal; paragraph and container IDs follow the encoded program symbol"
    requirement: ID-03
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_identity_test.go#TestCobolGrammar_ProgramIDEncoding"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobol'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Literal CALL/LINK/XCTL and COPY/-INC/SQL INCLUDE edges carry cobol_placeholder equal to their unresolved::cobol:: target; COPY IDMS and dynamic CALL carry none; OF/IN library adds cobol_copy_library_key"
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_identity_test.go#TestCobolGrammar_Placeholder"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/resolver/ -run 'Cobol'"
        status: pass
    human_judgment: false
  - id: D3
    description: "Repeated literal CALLs keep distinguishable provenance: distinct lines give distinct edges and statement symbols; same-line duplicates collapse with cobol_occurrence_count 2 and two distinct KindLocal statement symbols with their own columns"
    requirement: ID-05
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_identity_test.go#TestCobolGrammar_Occurrences"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 02: COBOL Program ID Encoding and Edge Placeholders Summary

**Literal PROGRAM-IDs now go through `cobolIDSegment` in every program ID component, so `'A/B'` is `{path}::'A%2FB'` and can no longer pose as a nested path. Every bindable CALL and COPY edge now carries its original target in `cobol_placeholder`, and a COPY with an OF/IN library also gets a normalized `cobol_copy_library_key`.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-10-13T01:25:24Z
- **Completed:** 2026-10-13T01:37:33Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- ID-03 / IN-03: program symbols are built from `cobolIDSegment(name)` at both the top level and the nested level. `#k` ordinals count encoded symbols. `Node.Name`, `cobol_name_key`, and END PROGRAM matching still use the verbatim name.
- D-18: literal CALL edges (including CICS LINK and XCTL, which reuse `cobolCallTarget`) and copy, -INC, and SQL INCLUDE edges now carry `cobol_placeholder` equal to their extraction-time `To`. COPY IDMS and dynamic edges have no placeholder, so they can never be bound.
- D-10: a COPY with `OF`/`IN` keeps `cobol_copy_library` as written and adds `cobol_copy_library_key = cobolNameNorm(library)`.
- ID-05: a test now proves that repeated occurrences stay distinguishable. No dedupe change was needed.

## Task Commits

1. **Task 1: Encode program ID components**
   - RED: `f91a41e5` (test). `TestCobolGrammar_ProgramIDEncoding` failed on the encoding assertions; `check tdd-red-evidence` returned `RED_EVIDENCE_OK`, reason `target_test_failed`.
   - GREEN: `82d58e3d` (feat)
2. **Task 2: Placeholder, library key, and occurrence proof**
   - RED: `e8b59135` (test). `TestCobolGrammar_Placeholder` failed on the missing Meta keys; `check tdd-red-evidence` returned `RED_EVIDENCE_OK`.
   - GREEN: `1b479675` (feat)

## Files Created/Modified
- `internal/parser/languages/cobol_grammar_identity.go`: encodes program symbol components and updates the symbol field and function docs.
- `internal/parser/languages/cobol_grammar_refs.go`: adds `cobol_placeholder` in `cobolCallTarget` (literal branch) and `cobolCopy` (all forms except idms), plus `cobol_copy_library_key`.
- `internal/parser/languages/cobol_grammar_identity_test.go`: new file with the three tests and their invented fixtures.
- `internal/parser/languages/cobol_grammar_test.go`: `TestCobolGrammar_ProgramNestedStamp/literal_slash_name` now expects the encoded QualName `"A%2FB"`.

## Decisions Made
- **Literal quotes stay in IDs.** A probe confirmed that the pinned grammar does name `PROGRAM-ID. 'A/B'.`, but it keeps the quotes in `program_name`. The encoded IDs are therefore `{path}::'A%2FB'` and `{path}::'X%232'`, not the plan's `{path}::A%2FB` and `{path}::X%232`. Stripping the quotes would change the ID shape that D-20 and CONTEXT fix (verbatim names). The forgery fix itself is the same: no `/` or `#` from a name can reach an ID.
- **IN-04 left unchanged.** Same-line duplicates already list two distinct statement symbols, and each resolves to a `KindLocal` node with its own start column (11 and 26 in the probe).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Existing test pinned the unencoded literal-slash QualName**
- **Found during:** Task 1 (GREEN verify)
- **Issue:** `TestCobolGrammar_ProgramNestedStamp/literal_slash_name` in `cobol_grammar_test.go` expected the QualName `"A/B"`. The planned encoding intentionally changes it to `"A%2FB"`.
- **Fix:** Updated the expectation and added a one-line comment. The test still asserts the substance (`cobol_nested` false).
- **Files modified:** `internal/parser/languages/cobol_grammar_test.go` (a fourth file outside `files_modified`)
- **Verification:** `go test -race ./internal/parser/languages/ -run '^TestCobol'` passes.
- **Committed in:** `82d58e3d`

**2. [Plan text vs actual ID shape] Test expectations adjusted to real ID shapes**
- **Found during:** Task 1 (RED probe, as the plan's action asked)
- **Issue:** The plan's `<behavior>` text assumed unquoted literal IDs (`{path}::A%2FB`) and paragraph IDs of the form `{path}::P1/MAIN-PARA`. The real shapes are `{path}::'A%2FB'` (quotes kept by the grammar) and `{path}::P1/@proc/MAIN-PARA` (the existing P3 shape).
- **Fix:** The tests assert the real shapes. No ID shape was changed.
- **Verification:** All four `ProgramIDEncoding` subtests pass.

---

**Total deviations:** 2 (1 auto-fixed test pin, 1 test expectation aligned to existing ID shapes)
**Impact on plan:** None on scope. ID-03 and ID-05 are closed as intended.

## TDD Gate Compliance

- Task 1: RED `test(04-02)` `f91a41e5` precedes GREEN `feat(04-02)` `82d58e3d`.
- Task 2: RED `test(04-02)` `e8b59135` precedes GREEN `feat(04-02)` `1b479675`.
- `TestCobolGrammar_Occurrences` passed at RED. This is expected: it proves existing behavior (ID-05 flagged assumption, "prove first, fix only if indistinguishable"). The RED target for Task 2 was `TestCobolGrammar_Placeholder`.
- No REFACTOR commits; none were needed.

## Issues Encountered
- One combined multi-package `go test` run reported `FAIL ./cmd/gortex` with no failing test named. Run on its own, `./cmd/gortex` passes, and so does every other package with COBOL tests: cfg, embedding, mcp, neo4jprojection, parser, forest, cobolprobe, languages, resolver, and indexer with `-run '(?i)cobol'`. This looks like a resource or timeout flake in the combined run, not this change.

## Threat Flags

None. The new Meta keys (`cobol_placeholder`, `cobol_copy_library_key`) are the planned surface; T-04-07 (projection screen) is asserted by 04-10.

## User Setup Required

None. No external service configuration required.

## Next Phase Readiness
- 04-06 (reconcile) and 04-08 (CR-01 restub) can read `cobol_placeholder` directly.
- Output changes here (encoded literal program IDs, new edge Meta) ship under `gortex-cobol-grammar/5`, which 04-11 sets; this plan did not bump the version.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*

## Self-Check: PASSED
