---
phase: 05-explicit-unresolved-evidence
plan: 07
subsystem: cobol-graph
tags: [cobol, finding, cascade, gate, regression, sqlite, copybook]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "uncontained-cascade PARSE_UNRESOLVED document finding and copybook parse findings (05-03, gortex-cobol-grammar/6)"
provides:
  - "internal/indexer/cobol_cascade_gate_test.go: the D-14/D-15 cascade gate in the default go test -race (no flag, skip, or short-mode guard)"
  - "cobolCascadeControl / cobolCascadeInject: the TestErrorCascade 20/20 control and its three injection points (before data, after data item 1, after paragraph 1)"
  - "cobolCascadeCounts / cobolCascadeTrack: per-document data-item, paragraph, and uncontained-cascade counts read back from SQLite"
affects: [05-10, 05-11, 05-12, phase-06]

actuals:
  tokens: 2200
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Real-path regression gate: one one-file repository per case through newCobolLifecycleEnv, counts read from SQLite by Meta cobol_kind"
    - "A gate over existing behavior proves its RED by mutating the producer (empty cobolCascadeFamily) and reverting, instead of a feat commit"

key-files:
  created:
    - internal/indexer/cobol_cascade_gate_test.go
  modified: []

key-decisions:
  - "05-07: the cascade gate is test-only; its RED was proven by emptying cobolCascadeFamily (all six Positive subtests and the Copybook test fail with 0 findings) and reverting the file, so there is no feat commit"
  - "05-07: the copybook control is CUST-REC with two 05 children (3 data items); the injected variant adds 05 CUST-ID PIC X(8) WHATEVER. as the first child and grades as a cascade on the real path"

patterns-established:
  - "cobolCascadeTrack(t, rel, src) returns the store plus counts so a case can also read edges"

requirements-completed: [GAP-08]

coverage:
  - id: D1
    description: "The clean 20-data-item, 20-paragraph control tracks to exactly 20 data items and 20 paragraphs in SQLite with no uncontained-cascade finding"
    requirement: GAP-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_cascade_gate_test.go#TestCobolCascadeGate_CleanControl"
        status: pass
    human_judgment: false
  - id: D2
    description: "Six program cascade shapes each track without error, keep their file node, and persist exactly one high PARSE_UNRESOLVED uncontained-cascade finding with producer reasons"
    requirement: GAP-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_cascade_gate_test.go#TestCobolCascadeGate_Positive"
        status: pass
    human_judgment: false
  - id: D3
    description: "Eight contained shapes (four EXEC CICS forms, IDMS OBTAIN, EXEC SQL SELECT after paragraph 1, SCHEMA SECTION before data, EXEC SQL INCLUDE SQLCA in WORKING-STORAGE) keep 20 data items and 20 paragraphs and have no cascade finding"
    requirement: GAP-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_cascade_gate_test.go#TestCobolCascadeGate_Negative"
        status: pass
    human_judgment: false
  - id: D4
    description: "A copybook cascade attaches one finding to the copybook (finding_container copy/cust.cpy, one member_of edge to the file node); the uninjected member keeps all 3 data items and has no cascade finding"
    requirement: GAP-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_cascade_gate_test.go#TestCobolCascadeGate_Copybook"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 07: COBOL Cascade Gate Summary

**A committed real-path gate in the default `go test -race`: six injected cascade shapes and one copybook cascade must persist an `uncontained-cascade` finding in SQLite and still index, and eight contained shapes must keep the clean control's 20 data items and 20 paragraphs.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-10-20T18:02:06Z
- **Completed:** 2026-10-20T18:10:05Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- `internal/indexer/cobol_cascade_gate_test.go` (`//go:build darwin || linux`) ports `cobolprobe` `TestErrorCascade` onto the production path: the registered COBOL extractor, a MultiIndexer, and a SQLite temp store through `newCobolLifecycleEnv`. Each case tracks its own one-file repository `gate`.
- The gate has no flag, no `testing.Short()` guard, and no skip. The acceptance grep prints 0.
- `TestCobolCascadeGate_CleanControl` pins 20 data items and 20 paragraphs and no cascade finding.
- `TestCobolCascadeGate_Positive` covers six shapes:
  - OBTAIN in WORKING-STORAGE
  - EXEC SQL DECLARE CURSOR in WORKING-STORAGE
  - EXEC SQL DECLARE ... WITH HOLD in procedure
  - `FROBNICATE THE WIDGET NOW.`
  - `01 BROKEN PIC X(10) GARBAGE CLAUSE HERE.`
  - DATA DIVISION without its period

  Each one tracks without error and persists its file node. Each persists exactly one finding with `finding_kind` uncontained-cascade, `prov_evidence_class` PARSE_UNRESOLVED, severity high, and non-empty `finding_producer_reasons`.
- `TestCobolCascadeGate_Negative` re-pins the clean control first. It then checks eight shapes against it, using the analog's `t.Fatalf` style:
  - EXEC CICS with a paren option, no option, a bare option, and multi-line
  - IDMS OBTAIN after paragraph 1
  - EXEC SQL SELECT after paragraph 1
  - SCHEMA SECTION before data
  - EXEC SQL INCLUDE SQLCA in WORKING-STORAGE
- `TestCobolCascadeGate_Copybook` tracks an invented `copy/cust.cpy`. The control keeps all 3 data items and has no cascade finding. The variant with `05 CUST-ID PIC X(8) WHATEVER.` as the first child persists one cascade finding with `finding_container` `copy/cust.cpy` and one `member_of` edge to `gate/copy/cust.cpy`.
- `internal/parser/forest/cobolprobe` is unchanged.

## Task Commits

1. **Task 1: Real-path gate for program cascade shapes and contained negatives** - `eac33b3c` (test)
2. **Task 2: Copybook cascade case under the copybook grade policy** - `e8423e2c` (test)

**Plan metadata:** recorded in the docs(05-07) commit that adds this file.

## Files Created/Modified

- `internal/indexer/cobol_cascade_gate_test.go` - the control builder, the injection helper, the SQLite count reader, the track helper, and the four `TestCobolCascadeGate_*` tests

## Decisions Made

- The gate is test-only, because the behavior it guards already exists (05-03). Task 2 generalized `cobolCascadeTrack` to take the repository-relative path and return the store, so the copybook case can read the finding's `member_of` edge. That needed no second track helper.
- The copybook control has two `05` children (3 data items). It still grades as a cascade on the real path with the probe's injected child.

## Deviations from Plan

None. The plan executed exactly as written.

## TDD Gate Compliance

Both tasks are `tdd="true"`, but they only gate behavior that 05-03 already shipped. The gate tests pass on first run because the feature already exists, so the RED gate was proven by mutation instead:

- **Mutation:** `cobolCascadeFamily` was set to an empty list in `internal/parser/languages/cobol_grammar_findings.go`.
- **Result:** All six `TestCobolCascadeGate_Positive` subtests failed. `TestCobolCascadeGate_Copybook` also failed. Each reported `"[]" should have 1 item(s), but has 0`.
- **Revert:** The file was restored with a single-file checkout, and `git diff --quiet` confirmed the restore.

No production code changed, so there is no `feat(05-07)` commit. Both commits are `test(05-07)`.

## Issues Encountered

None. Every listed shape graded on the real path as the research probe table predicts, so no shape was reclassified.

## Verification

- Task 1 verify passed: the three tests are listed, pass under `-race`, `cobolprobe` has no diff, and gofmt is clean.
- Task 1 acceptance passed:

  | Check | Result |
  |---|---|
  | `enhanced-parser\|testing.Short()\|t.Skip` | 0 |
  | `newCobolLifecycleEnv(` | 1 |
  | `git diff --quiet -- internal/parser/forest/cobolprobe` | exit 0 |

- Task 2 verify passed: the test is listed, passes under `-race`, and gofmt is clean.
- Plan verification:
  - `GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestCobolCascadeGate_'` passes all four tests together in about 30 s.
  - `GOWORK=off go build ./...` passes.
  - `git diff --name-only cc415e4b..HEAD` lists only `internal/indexer/cobol_cascade_gate_test.go`.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- GAP-08 is a deterministic regression failure. A change that drops the cascade finding fails the gate in the default race suite.
- Re-pinning the producer must re-run the gate. If a shape grades differently, the gate fails and the shape must not be silently reclassified.

## Self-Check: PASSED

- FOUND: internal/indexer/cobol_cascade_gate_test.go
- FOUND commits: eac33b3c, e8423e2c
- Both task verify commands, every acceptance criterion, and the plan-level verification passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
