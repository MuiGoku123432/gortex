---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 11
subsystem: parser
tags: [cobol, shared-resources, graphdump, extractor-version, determinism, tdd]

requires:
  - phase: 04-01
    provides: writer-independent resourceNode (D-18) and the graphdump canonical dump helper (D-01, D-02)
  - phase: 04-02
    provides: identity changes covered by the same /5 bump
provides:
  - TestCobolGrammar_SharedResource locking writer-independent DB2/IDMS/CICS resource nodes and qualifier/delimited-name boundaries
  - TestGraphdump_Dump locking the Dump/FirstDiff contract on a seeded SQLite store
  - cobolGrammarExtractorVersion = gortex-cobol-grammar/5 with all six pins updated
affects: [04-09, 04-12, phase-4-gate, neo4j-projection]

actuals:
  tokens: 3640       # chars/4 over the realized diff (14,565 chars)
  tasks: 1
  commits: 2         # MEASURED: git rev-list --count plan_head_before..HEAD
plan_head_before: 5c87e6e9cb4f133b43cddc234912b8848865f46a

tech-stack:
  added: []
  patterns:
    - "Contract-lock unit tests pin a 04-01 outcome at its source (one package-level test per contract)"
    - "Shared-resource writer independence asserted by comparing a {ID, Kind, Name, FilePath, Language, Meta} view across two documents"

key-files:
  created:
    - internal/parser/languages/cobol_grammar_resource_test.go
    - internal/testutil/graphdump/graphdump_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_contract_test.go
    - cmd/gortex/cobol_tracer_test.go
    - cmd/gortex/cobol_breadth_test.go

key-decisions:
  - "One gortex-cobol-grammar/5 bump covers 04-01's shared-resource change and 04-02's identity changes; later Phase 4 extractor changes keep /5 until the phase gate"
  - "extractor_version.go \"cobol\": 2 (and its /4 comment) left untouched for 04-09's stamp-derived fingerprint (D-07)"

patterns-established:
  - "Version bumps under TDD: update the pinned assertions first (RED), then the constant (GREEN)"

requirements-completed: [ID-04, ID-06]

coverage:
  - id: D1
    description: "Shared DB2, IDMS, and CICS resource nodes are identical whichever document (program or procedure copybook) writes them, with per-document cobol_resource_raw on edges"
    requirement: ID-04
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_resource_test.go#TestCobolGrammar_SharedResource/program_and_copybook"
        status: pass
    human_judgment: false
  - id: D2
    description: "Unqualified, two-part, and three-part DB2 names map to separate _unqualified and schema nodes, with the location kept on the edge as sql_location; delimited identifiers keep exact spelling and never merge with the folded name"
    requirement: ID-04
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_resource_test.go#TestCobolGrammar_SharedResource/qualified_and_unqualified"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_resource_test.go#TestCobolGrammar_SharedResource/delimited_name_exact"
        status: pass
    human_judgment: false
  - id: D3
    description: "graphdump.Dump returns sorted rows and a stable digest, drops only justified volatile keys while reporting VolatileVersion, refuses an unversioned or unjustified allowlist, and FirstDiff names the first differing index"
    requirement: ID-06
    verification:
      - kind: unit
        ref: "internal/testutil/graphdump/graphdump_test.go#TestGraphdump_Dump"
        status: pass
    human_judgment: false
  - id: D4
    description: "COBOL mapping versioned gortex-cobol-grammar/5 and every pinned prov_extractor_version assertion agrees"
    verification:
      - kind: unit
        ref: "go test -race ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'; ./internal/parser/languages -run '^Test(Graphdump|Cobol)'; ./internal/indexer -run '^TestCobolLifecycle_'"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 11: Shared-Resource and Dump Contract Locks, COBOL Mapping /5 Summary

**Unit tests pin writer-independent DB2/IDMS/CICS resource nodes (D-18) and the graphdump Dump/FirstDiff contract (D-01, D-02). The COBOL mapping is now `gortex-cobol-grammar/5`, and all six pins match it.**

## Performance

- **Duration:** about 6 min
- **Started:** 2026-10-13T02:04:38Z
- **Completed:** 2026-10-13T02:10:19Z
- **Tasks:** 1 (TDD: RED + GREEN commits)
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- `TestCobolGrammar_SharedResource` has three subtests.
  - `program_and_copybook`: a program and a procedure copybook name one DB2 table, one IDMS record, and one CICS transaction, each in different case. They produce equal resource nodes (`ID`, `Kind`, `Name`, `FilePath`, `Language`, `Meta`). Each edge's `cobol_resource_raw` keeps its own document's spelling (`mysch.orders` / `MySch.Orders`, `emp-rec` / `Emp-Rec`, `'trn1'` / `'Trn1'`).
  - `qualified_and_unqualified`: `ORDERS` and `MYSCH.ORDERS` stay two nodes, `db2::_unqualified/ORDERS` and `db2::MYSCH/ORDERS`. `LOC.MYSCH.ORDERS` maps to `db2::MYSCH/ORDERS` with `sql_location=LOC` on its edge.
  - `delimited_name_exact`: `"MySch"."Orders"` becomes `db2::"MySch"/"Orders"` and keeps its exact spelling. It never shares a node with `MYSCH.ORDERS`. The subtest also probes ASCII-only folding (`Ämp-rec` → `ÄMP-REC`) and `cobolIDSegment` percent-encoding of `% : / #` and control bytes.
- `TestGraphdump_Dump` runs on a seeded `store_sqlite` store (two nodes and one edge, scoped by RepoPrefix, WorkspaceID, and ProjectID). It checks that:
  - rows are sorted, and two dumps give the same rows and digest;
  - a key outside the allowlist stays in the rows;
  - a justified volatile key is removed from every row, and `VolatileVersion` is reported;
  - a Volatile with no version, or a key with no justification, returns an error;
  - `FirstDiff` returns `""` for equal rows and `- [1] …` when row 1 differs.
- `cobolGrammarExtractorVersion` is now `gortex-cobol-grammar/5`. Its comment says this is the Phase 4 mapping and stays at /5 until the phase gate. The six pins in `cobol_grammar_contract_test.go` (×2), `cobol_tracer_test.go` (×3) and `cobol_breadth_test.go` (×2 sites; the plan counts six assertions) were updated.

## Task Commits

1. **Task 1 RED:** `21901e02` (test). Adds the contract-lock tests and moves the pins to /5. The pins failed against the /4 constant.
2. **Task 1 GREEN:** `e419efb4` (feat). Bumps the constant to /5.

No REFACTOR commit was needed.

## TDD Gate Compliance

- **RED** (`21901e02`): the new contract-lock tests passed against 04-01's code, as the plan expects, because they lock existing behaviour. The RED gate is the version pins. Every updated assertion failed with `expected: "gortex-cobol-grammar/5"` / `actual: "gortex-cobol-grammar/4"`:
  - `TestCobolGrammar_ProvKeys`: 50 assertion failures.
  - `TestCobolBreadth_StructureAndFlow` and `TestCobolBreadth_CopybookProcedure`.
  - `TestCobolTracer_PersistsThroughAddBatch`, `TestCobolTracer_MCPGetSymbol`, `TestCobolTracer_CLIGetSymbol`, and `TestCobolTracer_ParagraphFlow`.

  These are intentional failures on the planned behaviour. `gsd-tools check tdd-red-evidence` cannot parse `go test` output, so this is the evidence instead.
- **GREEN** (`e419efb4`): the full plan verify command passed under `-race`: graphdump, parser/languages `^Test(Graphdump|Cobol)`, cmd/gortex `^TestCobol(Tracer|Breadth)_`, indexer `^TestCobolLifecycle_`, and a clean gofmt.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar_resource_test.go`: `TestCobolGrammar_SharedResource` (D-11, D-18, ID-04).
- `internal/testutil/graphdump/graphdump_test.go`: `TestGraphdump_Dump` (D-01, D-02, ID-06).
- `internal/parser/languages/cobol_grammar.go`: version constant set to /5, with the Phase 4 comment.
- `internal/parser/languages/cobol_grammar_contract_test.go`, `cmd/gortex/cobol_tracer_test.go`, `cmd/gortex/cobol_breadth_test.go`: pins updated to /5.

## Decisions Made

- One `/5` bump covers both 04-01 and 04-02. Later Phase 4 extractor changes keep `/5` until the phase gate.
- `internal/indexer/extractor_version.go` `"cobol": 2` and its `(gortex-cobol-grammar/4)` comment were not changed. 04-09 replaces the entry with the stamp-derived fingerprint (D-07). Acceptance criterion 2 allows that one comment line to remain.

## Deviations from Plan

None. The plan was executed as written.

One small adjustment: the new test first defined its own `cobolEdgesTo` helper. That collided with the existing `cobolEdgesTo(result, to, kind)` in `cobol_grammar_identity_test.go`. The test now reuses the existing helper (Engineering Standards reuse rule). This happened before the RED commit and is not a code deviation.

## Issues Encountered

None. 04-01's code satisfied both contract-lock tests on the first run, so no 04-01 fix was needed.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- Wave 2 sibling 04-12 and later plans can rely on locked D-18 and dump contracts.
- 04-09 still has to replace `"cobol": 2` with the stamp-derived fingerprint (D-07).

## Self-Check: PASSED

- FOUND: internal/parser/languages/cobol_grammar_resource_test.go
- FOUND: internal/testutil/graphdump/graphdump_test.go
- FOUND: commit 21901e02
- FOUND: commit e419efb4
- `git diff --name-only 5c87e6e9..HEAD` lists exactly the 6 planned files.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*
