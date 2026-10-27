---
phase: 05-explicit-unresolved-evidence
plan: 12
subsystem: cobol-graph
tags: [cobol, finding, placeholder, mcp, cli, search, graphdump, canonical, finding_history, d-16, inventory, go-ast, docs, windows-ledger, phase-gate]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "index_health findings counts and the Neo4j label Finding (05-04); missing-artifact placeholders (05-05); the cascade gate (05-07); EXTERNAL_UNRESOLVED findings (05-08); the durable finding ledger (05-09); scope-accurate reasons and the D-20 program hard failure (05-10); symbol FTS sync at every settle point (05-11)"
provides:
  - "Four real-path cmd tests: the NOPGM external finding by exact ID through MCP and CLI get_symbol, the placeholder through MCP get_symbol, and placeholder search (T-05-36)"
  - "internal/indexer/cobol_findings_equivalence_test.go: cobolFindingsCorpus (two independent repositories with every finding source) and five canonical-graph tests"
  - "internal/graph/finding_writers_test.go: TestFindingWriters_Inventory, the D-16 go/ast writer inventory (T-05-35)"
  - "docs/languages.md COBOL row: parse and external findings, placeholders, index_health counts, finding_history, the Neo4j Finding label, and the D-20 program analysis-failure limitation"
  - ".planning/WINDOWS.md rows 4, 5, 6 fixed with notes naming the explicit PARSE_UNRESOLVED findings"
  - "Phase gate: GOWORK=off go test -race -timeout 60m ./... green (157 ok, 0 FAIL, 0 cached)"
affects: [phase-06, phase-06.1, phase-08]

actuals:
  tokens: 9600
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "A canonical-graph test decodes graphdump N rows (\\x1f fields, JSON Meta last) to assert which finding and placeholder rows a dump holds"
    - "Writer fences are go/ast inventories over internal/ and cmd/ non-test files with a per-rule allowlist plus a required-hit list, so a scanner that stops matching fails"

key-files:
  created:
    - internal/indexer/cobol_findings_equivalence_test.go
    - internal/graph/finding_writers_test.go
  modified:
    - cmd/gortex/cobol_findings_test.go
    - docs/languages.md
    - .planning/WINDOWS.md

key-decisions:
  - "search_symbols relaxes a kind clause that filters every candidate out and flags filters_relaxed: true; PlaceholderSearch therefore asserts the placeholder appears under kind:function only through that declared fallback and only as kind resource, never as a function"
  - "The D-16 inventory requires each allowed writer file to still hold its rule (except the storetest allowlist entry), so a silent scanner regression fails the gate"
  - "WINDOWS.md fix notes live in each entry's reason field, the only free-text column the ledger schema has"

patterns-established:
  - "Prove a tests-only plan's tests are not vacuous with a throwaway production mutation that is reverted before commit (placeholder kind, ledger pairing key, a probe writer file)"

requirements-completed: [GAP-02, GAP-04, GAP-05, GAP-06, GAP-07]

coverage:
  - id: D1
    description: "The EXTERNAL_UNRESOLVED finding of CALL 'NOPGM' is returned by exact ID through MCP and CLI get_symbol with the full Pattern 3 Meta, a member_of edge to the caller, and a references edge to its placeholder"
    requirement: GAP-04
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_ExternalMCPGetSymbol"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_ExternalCLIGetSymbol"
        status: pass
    human_judgment: false
  - id: D2
    description: "The not-obtained placeholder is served through MCP with no line, range, or parser stamp, with the code calls edge and the finding's references edge as in-edges, and never as a function in search"
    requirement: GAP-05
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_PlaceholderMCPGetSymbol"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_PlaceholderSearch"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two clean builds and opposite track orders of the findings corpus give byte-identical dumps holding both finding classes and six placeholders; ambiguous, dynamic, and SQLCA references get none"
    requirement: GAP-06
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_equivalence_test.go#TestCobolFindings_CleanDeterminism"
        status: pass
    human_judgment: false
  - id: D4
    description: "A three-line shift keeps every finding ID; an added missing CALL, a deleted copybook, and the restore each equal a clean build, and the restore equals the original"
    requirement: GAP-06
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_equivalence_test.go#TestCobolFindings_LineShiftKeepsIDs"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_equivalence_test.go#TestCobolFindings_IncrementalEqualsClean"
        status: pass
    human_judgment: false
  - id: D5
    description: "A finding_history append leaves the canonical dump digest unchanged (D-07)"
    requirement: GAP-06
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_equivalence_test.go#TestCobolFindings_HistoryNotCanonical"
        status: pass
    human_judgment: false
  - id: D6
    description: "Moving CALL 'NOPGM' from A-PARA to B-PARA in one edit appends exactly one active -> closed and one '' -> active external row and no superseded row; the placeholder stays (D-03)"
    requirement: GAP-07
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_equivalence_test.go#TestCobolFindings_ExternalCrossContainerNoSupersede"
        status: pass
    human_judgment: false
  - id: D7
    description: "Only the extractor and resolver build findings or write finding_status, only finding_ledger.go appends history, and only the resolver and internal/graph evict findings (D-16)"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/graph/finding_writers_test.go#TestFindingWriters_Inventory"
        status: pass
    human_judgment: false
  - id: D8
    description: "docs/languages.md describes the findings, placeholders, index_health counts, history, the Neo4j Finding label, and the D-20 limitation; WINDOWS rows 4-6 read fixed"
    verification:
      - kind: other
        ref: "Task 3 verify (grep EXTERNAL_UNRESOLVED, cobol-missing::, 'A COBOL program whose analysis fails', file_index_failures; awk rows 4-6 fixed = 3)"
        status: pass
    human_judgment: true
    rationale: "Grep proves the required sentences exist; whether the COBOL row reads clearly to a user is a judgment call"
  - id: D9
    description: "Phase gate: the full race suite passes"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -timeout 60m ./..."
        status: pass
    human_judgment: false

duration: 61min
completed: 2026-10-21
status: complete
---

# Phase 5 Plan 12: Findings End to End and the Phase Gate Summary

**COBOL findings and not-obtained placeholders are now proven on the real path. A user can fetch the `NOPGM` external finding and its placeholder by exact ID through MCP and the CLI, and a `kind:function` search never returns the placeholder as an implementation. Findings and placeholders converge across clean, reordered, line-shifted, and edit/delete/restore builds, while `finding_history` stays outside the canonical dump. A go/ast inventory fences every finding write to its deterministic writer. The docs now state the D-20 program limitation, WINDOWS rows 4-6 are closed, and the full race suite is green.**

## Performance

- **Duration:** about 61 min, most of it test wall time. The full race suite took 1737 s and the two `^TestCobol` verification runs took 63 s and 295 s.
- **Started:** 2026-10-21T16:49:07Z
- **Completed:** 2026-10-21T17:49:58Z
- **Tasks:** 3 of 3
- **Files modified:** 5 (2 created, 3 modified)
- **Disk:** free space stayed between 41 and 53 GiB, well above the 10 GiB stop threshold.

## Accomplishments

- **MCP, CLI, and search (D-08, GAP-04, GAP-05).**
  - `cobolFindingsNOPGM` finds three IDs through the store, never by recomputing a hash: the finding by `finding_target_key`, the placeholder, and the caller (the From of the one `calls` edge into the placeholder).
  - `assertCobolExternalFinding` checks every Pattern 3 key. `finding_container` must equal the `member_of` target without its prefix, and `references` must point at `<prefix>/cobol-missing::program/NOPGM`.
  - The placeholder test checks four things:
    - kind `resource`, `artifact_status` `not_obtained`, `synthetic` true;
    - no `start_line`, no range keys, no `prov_parser_*` key;
    - a `calls` in-edge from the caller;
    - a `references` in-edge from the finding.
- **Canonical graph (GAP-06, D-01, D-07, P4 D-04).**
  - `cobolFindingsCorpus` holds every finding source in two independent repositories.
  - A clean build yields these findings:

    | Class | Findings |
    |---|---|
    | `PARSE_UNRESOLVED` | 2 unparsed-tail, 2 error, 1 missing, 2 lost-capture, 1 uncontained-cascade |
    | `EXTERNAL_UNRESOLVED` | 6 external-reference |

  - It also yields six placeholders: program, copybook, inc_member, and idms_dictionary_member in `fcora`, and program and copybook in `fcorb`.
  - The ambiguous `DUPPGM`, the dynamic `WS-PGM`, and `SQLCA` get no finding and no placeholder.
- **Cross-container ledger (D-03, GAP-07).** On the real path, moving `CALL 'NOPGM'` from `A-PARA` to `B-PARA` produces a close plus an open, never a supersede.
- **D-16 writer inventory (T-05-35).** A walk over the `internal/` and `cmd/` non-test files enforces four rules, each with its own allow list:

  | Rule | Allowed files |
  |---|---|
  | KindFinding construction | the extractor and resolver writers, plus the allowlisted `storetest.go` |
  | `finding_status` write | the two writers |
  | `AppendFindingTransitions` call | `internal/indexer/finding_ledger.go` |
  | `EvictFindingNodesByIDs` call | `internal/resolver/cobol_findings.go` and `internal/graph/` |

  - Reads and constant declarations never match.
  - Each failure names the file, line, and rule.
- **Docs and ledger.**
  - The COBOL row of `docs/languages.md` covers parse and external findings, placeholders, edge-only references, IDs, `index_health` counts, `finding_history`, the `unresolved::` change, the stale Neo4j projection, and the D-20 sentence.
  - WINDOWS rows 4-6 are marked `fixed`, each with a note.

## Task Commits

1. **Task 1: External finding and placeholder through MCP, CLI, and search.** `ecec2d04` (test)
2. **Task 2: Findings and placeholders are canonical; history is not.** `d27d5f5a` (test)
3. **Task 3: D-16 writer inventory, docs (with the D-20 limitation), and windows ledger.** `ef329c28` (test)

**Plan metadata:** the `docs(05-12)` SUMMARY commit, then the `docs(05-12)` STATE/ROADMAP/REQUIREMENTS commit.

## Files Created/Modified

- `cmd/gortex/cobol_findings_test.go`:
  - helpers `cobolFindingsNOPGM` and `assertCobolExternalFinding`;
  - tests `TestCobolFindings_ExternalMCPGetSymbol`, `_ExternalCLIGetSymbol`, `_PlaceholderMCPGetSymbol`, and `_PlaceholderSearch`.
- `internal/indexer/cobol_findings_equivalence_test.go`:
  - fixtures `cobolFindingsMain`, `cobolFindingsDup`, `cobolFindingsCorpus`, and the `cobolCrossContainer` pair;
  - dump decoders `cobolDumpNodes`, `cobolDumpIDs`, and `cobolDumpFindingIDs`;
  - env helper `newCobolFindingsEnv`;
  - the five tests.
- `internal/graph/finding_writers_test.go`:
  - rule tables `findingWriterRules` and `findingWriterRequired`;
  - `TestFindingWriters_Inventory`;
  - helpers `findingWriterHits`, `isFindingName`, and `isFindingStatusKey`;
  - it reuses `findRepoRoot`, `fenceSkippedDirs`, and `nestedGitCheckout` from the package's existing fence tests.
- `docs/languages.md`: the COBOL row (line 279), with its four table pipes unchanged.
- `.planning/WINDOWS.md`: rows 4-6 set to `fixed` with notes; counts are now 3 open and 4 fixed.

## Decisions Made

- **Search relaxation.** `PlaceholderSearch` asserts that under `kind:function` the placeholder appears only through the `filters_relaxed` fallback, and only as kind `resource`. See Deviation 1.
- **No vacuous inventory.** The inventory also requires each listed writer file to still match its rule. The `storetest.go` entry is exempt: it seeds through `mkRepoNode`, so it matches nothing today and stays allowed only to tolerate a later conformance case.
- **Where the notes go.** WINDOWS fix notes go in the `reason` field, because `gsd-tools windows fixed` takes no note argument.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test expectation] A `kind:function` search does not return an empty set**
- **Found during:** Task 1, on the first run of `TestCobolFindings_PlaceholderSearch`.
- **Issue:** The plan expected `kind:function NOPGM` to return no result with the placeholder ID. Instead, `handleSearchSymbols` (`internal/mcp/tools_core.go:2113-2124`) retries on the free text alone when a kind clause filters out every candidate. It returns that set with `filters_relaxed: true`, so the placeholder and the finding came back. This is a documented, flagged MCP fallback, not a bug in the 05-05 or 05-08 Meta or edges. Changing search relaxation is out of this plan's scope.
- **Fix:** The test decodes `filters_relaxed` and each hit's `kind`. If the placeholder appears under `kind:function`, the response must say the clause was relaxed, and the hit must be kind `resource`. The plain `NOPGM` search must still return the placeholder.
- **Verification:** A throwaway mutation that set the placeholder's kind to `KindFunction` in `internal/resolver/cobol_findings.go` made both placeholder tests fail. The mutation was reverted with `git checkout --` on that one file.
- **Committed in:** `ecec2d04`.

**2. [Rule 3 - Blocking] `gsd-tools windows fixed` dropped ledger fields and cannot record a note**
- **Found during:** Task 3.
- **Issue:** The ledger command rewrote `.planning/WINDOWS.md` without the `"milestone": "v1.0"` field. It dropped the field from all six entries that carried it. The command also takes no note argument.
- **Fix:** The command marked rows 4-6 fixed and wrote the counts and timestamps. A script then restored each `milestone` field from HEAD and wrote the fix note into the `reason` field of each of rows 4-6, in both the JSON block and the table.
- **Verification:**
  - `gsd-tools windows status` parses the file: `ok`, 3 open, 4 fixed.
  - The diff against HEAD touches only rows 4-6 and the frontmatter.
  - The plan's awk check prints 3.
- **Committed in:** `ef329c28`.

---

**Total deviations:** 2 auto-fixed (1 test expectation, 1 tooling blocker).
**Impact on plan:** No production code changed. `git diff --name-only 15da5371..HEAD` lists exactly the 5 planned files.

## TDD Gate Compliance

Every task is `tdd="true"`, but each adds tests over behavior that 05-01 to 05-11 already shipped. Each test passed against existing code, except the search expectation in Deviation 1. As in 05-07 and 05-11, there is no `feat` commit. RED was proven by throwaway production mutations, each reverted before its commit:

| Task | Mutation | Tests that failed |
|---|---|---|
| 1 | placeholder kind set to `KindFunction` | `PlaceholderMCPGetSymbol` and `PlaceholderSearch` |
| 2 | ledger pairing key with the container blanked (`internal/graph/finding_ledger.go`) | `ExternalCrossContainerNoSupersede`, which reported `Should not be: "superseded"` |
| 3 | a probe file in `internal/mcp` with one write per rule plus reads | `TestFindingWriters_Inventory` |

For Task 3, the inventory reported seven offenders on the correct lines, and it did not flag the probe's reads.

## Verification

- **Task 1 verify** (exact command): exit 0. All eight `TestCobolFindings_*` tests are listed and pass under `-race` (25.9 s); gofmt is clean.
- **Task 2 verify** (exact command): exit 0. All five tests are listed and pass under `-race` (39.4 s); gofmt is clean.
- **Task 3 verify** (exact command): exit 0.
  - Acceptance greps:

    | Pattern | File | Count |
    |---|---|---|
    | `AssignStmt` | `finding_writers_test.go` | 1 |
    | `MetaFindingStatus` | `finding_writers_test.go` | 3 |
    | `PARSE_UNRESOLVED` | `docs/languages.md` | 1 |
    | `EXTERNAL_UNRESOLVED` | `docs/languages.md` | 1 |
    | `A COBOL program whose analysis fails` | `docs/languages.md` | 1 |
    | `file_index_failures` | `docs/languages.md` | 1 |
    | `guess a program identity from a filename` | `docs/languages.md` | 1 |

  - WINDOWS rows 4-6 all read `fixed`.
- **Plan verification:**
  - `GOWORK=off go test -race -count=1 -timeout 30m ./cmd/gortex/ -run '^TestCobol'`: ok (63.2 s).
  - `GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer/ -run '^TestCobol'`: ok (295.1 s). This covers every tracer, breadth, scope, findings, lifecycle, and cascade-gate test.
  - **Phase gate** `GOWORK=off go test -race -timeout 60m ./...`: exit 0 in 1737 s.
    - 157 packages ok, 40 with no test files, 0 FAIL, 0 cached.
    - The largest packages:

      | Package | Time |
      |---|---|
      | `internal/indexer` | 1629.3 s |
      | `internal/graph/store_sqlite` | 1125.3 s |
      | `internal/mcp` | 821.1 s |
      | `cmd/gortex` | 343.3 s |
      | `internal/parser/languages` | 128.6 s |
      | `internal/resolver` | 114.1 s |
      | `internal/graphview` | 98.3 s |
      | `internal/graph` | 35.2 s |
      | `internal/neo4jprojection` | 10.6 s |

    - No pre-existing failure surfaced, so `deferred-items.md` gains no entry.
  - `git diff --name-only 15da5371..HEAD` lists only the 5 planned files. The pre-existing `.planning/config.json`, `.planning/state.json`, `milestone.lock`, and `06.1-*` changes were left unstaged.

## Issues Encountered

- `internal/resolver/csharp_verbatim_type_ref_test.go` is still not gofmt-clean at HEAD. This is pre-existing and already in `deferred-items.md`.

## Known Stubs

None.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- Phase 5 is complete: 12 of 12 plans have SUMMARYs and the phase gate is green. It is ready for `/gsd-verify-work 05` and phase verification.
- Phase 6 (QUERY-08..10, EVAL-06) can rely on these properties:
  - Active findings and placeholders are canonical.
  - The kind label is `Finding` in Neo4j and `finding` in SQLite.
  - The D-16 inventory fails if a list or filter verb ever writes a finding.
  - The D-20 limitation (a present program whose analysis fails yields a false not-obtained placeholder for its callers) is documented and left for Phase 6.

## Self-Check: PASSED

- FOUND: cmd/gortex/cobol_findings_test.go, internal/indexer/cobol_findings_equivalence_test.go, internal/graph/finding_writers_test.go, docs/languages.md, .planning/WINDOWS.md
- FOUND commits: ecec2d04, d27d5f5a, ef329c28
- All three task verify commands, every acceptance criterion, both `^TestCobol` runs, and the full race suite passed.

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-21*
