---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
plan: 03
subsystem: parser
tags: [cobol, tree-sitter, identity, provenance, copybook, fail-closed, tdd]
requires:
  - phase: 02-02
    provides: "CobolGrammarExtractor (!windows) with the prov_* Meta contract, approvedGrammarID injection field, Windows stub, and the TestCobolTracer_* acceptance"
provides:
  - "cobolProgramSymbols: END PROGRAM name-stack containment paths (OUTERPGM/INNERPGM), observation-order #k ordinals (k >= 2), containment-unresolved flag, unnamed-program count"
  - "Program IDs relPath::symbol with QualName = symbol; file-node prov_containment_unresolved (only when true) and prov_unnamed_program_count (only when > 0)"
  - "Copybook branch: .cpy/.CPY emit the file node only, with prov_analysis_absence=copybook_standalone_analysis_unsupported and producer-domain prov_revision_content_id; Analyze never called"
  - "13 TestCobolGrammar_* invented-fixture tests plus the windows-only TestCobolGrammarWindowsStub"
  - "Local proof of the BASE-02 build-level negative (stock grammar fails the pin test with the attestation mismatch text)"
affects: [02-04, 02-05, 02-06, phase-3-cobol-calls, phase-4-invalidation, phase-5-parse-unresolved, neo4j-projection]
actuals:
  tokens: 5831
  tasks: 2
  commits: 4
plan_head_before: c818e1857123f3ce04f12fa3d66eaff3cc8c8efb
tech-stack:
  added: []
  patterns:
    - "Containment identity from an END PROGRAM name stack, not from grammar Parent links (which are flat)"
    - "Document-level flags set on docMeta after the per-program maps.Clone so they stay on the file node"
    - "Same-package fault injection through the approvedGrammarID field"
key-files:
  created:
    - internal/parser/languages/cobol_grammar_identity.go
    - internal/parser/languages/cobol_grammar_test.go
    - internal/parser/languages/cobol_grammar_windows_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
key-decisions:
  - "cobolProgramNames (02-02) was folded into cobolProgramSymbols: one pass maps definition names and END PROGRAM names, so there is still exactly one program-name lookup."
  - "prov_extractor_version stays gortex-cobol-grammar/1. The Phase 2 mapping has not been released or persisted outside tests, the plan keeps the 02-02 key contract, and cmd/gortex tracer tests assert /1. Bump it on the first post-phase change."
  - "The '#'-in-ID helper search found no helper that breaks the D-10 shape for KindFunction COBOL programs, so the shape was kept (see Decisions Made for the two owner-hop helpers and why they are harmless today)."
  - "Copybook branch sits after the cached init check and NewSourceID, so a missing or foreign enhanced parser still fails every COBOL file, copybooks included (D-05)."
requirements-completed: [BASE-01, BASE-02, BASE-03, TRACE-01, PROV-02, PROV-03, PROV-05, ID-01, ID-02]
coverage:
  - id: D1
    description: "Nested programs get containment-path IDs (src/nest.cbl::OUTERPGM, ::OUTERPGM/INNERPGM) with matching QualName, two defines edges, and OUTERPGM keeps the parser-observed range that stops where INNERPGM begins"
    requirement: ID-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_NestedIDs (go test -race)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Same-container duplicates get observation-order ordinals FIRSTPGM, FIRSTPGM#2, FIRSTPGM#3, never #1, and repeated extraction is identical"
    requirement: ID-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_DuplicateOrdinal (go test -race)"
        status: pass
    human_judgment: false
  - id: D3
    description: "IDs are line-free: five prepended blank/comment lines leave every ID unchanged and shift every prov_start_row by exactly 5"
    requirement: ID-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_IDIgnoresLines (go test -race)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Clean DEMOPGM program node contract (ID, Kind, QualName, defines edge provenance, PROV-03 range ordering, document keys, empty scope fields) and verbatim lowercase names distinct from uppercase"
    requirement: ID-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_ProgramNode and #TestCobolGrammar_NamesVerbatim (go test -race)"
        status: pass
    human_judgment: false
  - id: D5
    description: "An unmatched END PROGRAM sets prov_containment_unresolved without guessing, and a program with no PROGRAM-ID is counted (prov_unnamed_program_count=1, grade red) with no fabricated node"
    requirement: ID-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_ContainmentUnresolved and #TestCobolGrammar_UnnamedProgramCounted (go test -race)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Amber and red documents keep their program nodes with grade, reasons, and handoff Affected as separate keys, and prov_confidence stays 1.0"
    requirement: PROV-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_DegradedAmber (go test -race)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Copybooks (.cpy and .CPY) yield only the file node with the analysis-absence key, source path/ID, NewContentID revision, and extractor version, with no grade or parser keys, even under a wrong grammar pin"
    requirement: PROV-02
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_CopybookSkipsAnalysis (go test -race)"
        status: pass
    human_judgment: false
  - id: D8
    description: "Empty source yields one red file node with the document keys and no program; eight concurrent Extract calls on one extractor return identical IDs"
    requirement: BASE-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_EmptySource and #TestCobolGrammar_ConcurrentExtract (go test -race)"
        status: pass
    human_judgment: false
  - id: D9
    description: "Approved grammar pin: prov_parser_grammar_id equals cobolApprovedGrammarID on every node of two files, identity keys are 64-hex, and parser/extractor version keys match across files; a wrong pin fails with errCobolGrammarNotApproved naming both IDs"
    requirement: BASE-03
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_test.go#TestCobolGrammar_ApprovedGrammarPinned and #TestCobolGrammar_GrammarMismatchFails (go test -race)"
        status: pass
    human_judgment: false
  - id: D10
    description: "Build-level stock-grammar negative: dropping the forest replace in a temporary modfile makes TestCobolGrammar_ApprovedGrammarPinned fail with 'compiled forest language identity mismatch'"
    requirement: BASE-02
    verification:
      - kind: other
        ref: "Task 2 verify command 2 (go mod edit -dropreplace on a temp modfile, GOFLAGS=-mod=mod go test -modfile=..., grep for the mismatch text)"
        status: pass
    human_judgment: false
  - id: D11
    description: "Windows stub test asserts errCobolParserUnavailableOnWindows, a nil result, and the same Language and six-entry Extensions"
    requirement: BASE-02
    verification:
      - kind: other
        ref: "Scratch-module GOOS=windows go vet + go test -c against a minimal parser stand-in (pass); same body run on darwin with the tag stripped (pass)"
        status: pass
    human_judgment: true
    rationale: "No Windows cgo toolchain on this host, so the real package cannot be compiled or run for Windows. The first real run is the Plan 02-05 Windows CI shard."
duration: 12min
completed: 2026-09-30
status: complete
---

# Phase 2 Plan 03: COBOL Identity and Trust Contract Summary

**COBOL program IDs are now `relPath::OUTER/INNER` containment paths, derived from an END PROGRAM name stack, with `#k` observation-order ordinals for same-container duplicates, verbatim names, and no line numbers. Copybooks carry identity and a producer revision without a meaningless red grade. Degraded documents, unapproved grammars, the stock grammar, and Windows each fail visibly, proven by 13 invented-fixture tests plus the build-level stock-grammar negative.**

## Performance

- **Duration:** about 12 min
- **Started:** 2026-09-30T23:13:55Z
- **Completed:** 2026-09-30T23:26Z
- **Tasks:** 2/2
- **Files modified:** 4 (3 created, 1 modified)

## Accomplishments

- Added `cobol_grammar_identity.go` (`!windows`) with `cobolProgramSymbols`:
  - It walks observations with a stack of open programs. Before pushing a program, it closes every open program that has no END PROGRAM marker left.
  - `END PROGRAM N` pops down to the topmost open N. An unmatched N sets `containmentUnresolved` and leaves the stack alone.
  - The k-th program with the same base symbol gets `#k` for k >= 2.
  - A program with no PROGRAM-ID is counted and gets no symbol.
- `Extract` builds IDs as `filePath + "::" + symbol` with `QualName` = symbol and Name = the verbatim name. `prov_containment_unresolved` and `prov_unnamed_program_count` go on the file node only.
- Copybook branch: `.cpy` and `.CPY` return a file-only result with the five identity/revision keys, `prov_revision_content_id` comes from `preprocessor.NewContentID(src)`, and no Analyze call is made.
- Wrote `cobol_grammar_test.go` with 13 `TestCobolGrammar_*` tests, all on invented 7-column fixed-format fixtures (DEMOPGM, OUTERPGM/INNERPGM, FIRSTPGM x2/x3, demopgm, ONEPGM/OTHERPGM, unnamed, AMBERPGM, BADPGM, CUST-REC copybook).
- Wrote `cobol_grammar_windows_test.go` (`//go:build windows`) with `TestCobolGrammarWindowsStub`.
- Proved BASE-02 locally at build level (see D10).

## Task Commits

1. **Task 1 RED: containment-path identity tests**: `91f7e951` (test)
2. **Task 1 GREEN: containment-path IDs with ordinals**: `84fe5df2` (feat)
3. **Task 2 RED: degraded, copybook, empty, concurrency, guard, and Windows tests**: `ec35e5a4` (test)
4. **Task 2 GREEN: copybook branch**: `4474fe15` (feat)

**Plan metadata:** recorded in the docs commit that follows this SUMMARY.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar_identity.go`: `cobolProgramSymbol` and `cobolProgramSymbols` (115 lines).
- `internal/parser/languages/cobol_grammar.go`: uses `cobolProgramSymbols`, sets the file-level flags, adds the copybook branch, and drops `cobolProgramNames` (now 227 lines, under the 300-line target).
- `internal/parser/languages/cobol_grammar_test.go`: the 13 tests and their shared helpers (`extractCobolGrammar`, `cobolProgramNodes`, `cobolProgramIDs`, `cobolFileNode`, `cobolTestHandoff`, `assertCobolRange`). It reuses the existing `findEdgeTo` from `cobol_test.go`.
- `internal/parser/languages/cobol_grammar_windows_test.go`: `TestCobolGrammarWindowsStub`.

## Decisions Made

- **`#` helper search (Task 1 step 3).** I grepped `internal/` and `cmd/` for `Index`/`IndexByte`/`Cut`/`Split`/`Contains`/`HasPrefix` on `"#"` and `'#'`. Only three helpers derive an owner from the text before a `#` in a node ID:
  - `graph.EnclosingFromID` (`internal/graph/node.go:496`) applies only to `KindClosure`, `KindMethod`, `KindField`, and `KindEnumMember`. COBOL programs are `KindFunction`, so it returns no owner.
  - `resolver.enclosingFunctionForBinding` (`internal/resolver/bare_name_scope_bind.go:205`) is used only to index and bind `KindLocal`, `KindParam`, and `KindGenericParam` candidates (Go and Rust). COBOL emits none of these, so no candidate is ever owned by a COBOL program.
  - `graph.NodeIsTest` (`internal/graph/test_class.go:39`) does treat `FIRSTPGM` as the owner of `FIRSTPGM#2` and would copy its `is_test` stamp. Both programs live in the same file, and COBOL has no per-symbol test stamping, so both already get the same answer. This is a latent owner hop, not a behavior change today.

  The rest (`#param:`, `#local:`, and `#closure@` matchers, the C# `#param:`, and dataflow refine) match specific suffixes that a COBOL ID never contains. Since nothing breaks the D-10 shape, the shape was kept. The finding is stored as a pinned Gortex memory so Phase 3 re-checks it if it adds COBOL locals or per-program test stamping.
- **One name-lookup pass.** `cobolProgramNames` was removed, and its lookup now lives inside `cobolProgramSymbols`, together with the END PROGRAM name map.
- **Extractor version kept at `/1`** (see key-decisions).
- **Copybook placement** is after `e.once.Do(e.init)`, the `initErr` return, and `NewSourceID`, so D-05 still fails copybooks when the parser is unavailable.

## Deviations from Plan

### Auto-fixed Issues

None. No bugs, missing functionality, or blocking issues came up.

### Notes

**1. [Verify environment] The BASE-02 negative was run with the per-command git rewrite.**
- **Found during:** Task 2 verify command 2
- **Detail:** I ran the plan's command body verbatim, plus the orchestrator's `GIT_CONFIG_COUNT/KEY/VALUE` rewrite environment, so `-mod=mod` could reach the private fork if needed.
- **Result:** `-mod=mod` re-obtained stock `github.com/alexaandru/go-sitter-forest/cobol v1.9.1` through the public proxy and wrote its two sum lines into the temporary `stock.sum`. The target test failed with `enhanced COBOL parser unavailable: compiled forest language identity mismatch`, not with "missing go.sum entry". The repo's `go.mod` and `go.sum` were untouched.
- **For Plan 02-05:** the deferred-items concern (vacuous pass on missing sums) does not arise when `GOFLAGS=-mod=mod` is set, as it is in this command. CI still needs network access to the Go proxy for that step.

**2. [Verify environment] The Windows test could not run natively.**
- **Detail:** The real package cannot be cross-compiled here: the forest grammars need cgo, and there is no mingw toolchain.
- **Partial proof:** I copied the stub and the test into a scratch module with a minimal `parser` stand-in. `GOOS=windows GOARCH=amd64 go vet` and `go test -c` both passed, and the same test body passed on darwin with the tag stripped.
- **Tracking:** Recorded as open `unrun-verify` in `.planning/WINDOWS.md` for the Plan 02-05 Windows CI shard.

---

**Total deviations:** 0 auto-fixes and 2 verification-environment notes. **Impact:** none on the planned contract or file set.

## TDD Gate Compliance

Both tasks are `tdd="true"` in a `type: execute` plan. Each task produced a `test(02-03)` commit before its `feat(02-03)` commit (`91f7e951` → `84fe5df2`, `ec35e5a4` → `4474fe15`).

The formal `check tdd-red-evidence` classifier parses Node TAP output, and this project uses `go test`, so RED was evidenced by hand. Every RED failure was a target-test assertion failure, never a compile error, load error, or zero-test run:

- **Task 1 RED:** 4 of 7 tests failed on identity assertions:
  - `NestedIDs` got `INNERPGM` instead of `OUTERPGM/INNERPGM`.
  - `DuplicateOrdinal` got `FIRSTPGM` twice instead of `FIRSTPGM` and `FIRSTPGM#2`.
  - `ContainmentUnresolved` found no `prov_containment_unresolved` key.
  - `UnnamedProgramCounted` found no `prov_unnamed_program_count` key.

  `ProgramNode`, `NamesVerbatim`, and `IDIgnoresLines` passed at RED. They characterize 02-02 behavior that was already correct: the clean single program, verbatim names, and the fact that 02-02 IDs were already line-free, just not collision-free.
- **Task 2 RED:** only `CopybookSkipsAnalysis` failed. A copybook was still analyzed, graded red, and given `prov_parser_*` keys. The other five tests characterize the 02-02 contract, as the plan predicted. No fixes were needed.

## Issues Encountered

- `go test -list` and the `-race` runs pass. The full `internal/parser/languages` package passes under `-race` (30s), `cmd/gortex` `TestCobolTracer_*` pass under `-race`, `go build ./...` succeeds, and the `.cpy`-referencing `cobolprobe` and `config` tests still pass.

## Known Stubs

None. The Windows `CobolGrammarExtractor` is an intentional fail-closed stand-in (D-05, D-17).

## Threat Flags

None. Threats T-02-11 through T-02-15 are each mitigated by the named tests. No new endpoints, auth paths, or trust-boundary schema were added.

## User Setup Required

None.

## Next Phase Readiness

- **Plan 02-04:** the single top-level DEMOPGM ID is unchanged. Copybooks carry `prov_revision_content_id`, so the VCS stamp applies to them too.
- **Plan 02-05:** can wire the BASE-02 negative as written with `GOFLAGS=-mod=mod`, which needs proxy network access. It also owns the Windows-shard run of `TestCobolGrammarWindowsStub` and the open WINDOWS.md entry.
- **Phase 3:** re-check the `NodeIsTest` owner hop before adding per-program test stamping or COBOL locals.

## Self-Check: PASSED

- FOUND: internal/parser/languages/cobol_grammar_identity.go, cobol_grammar.go, cobol_grammar_test.go, cobol_grammar_windows_test.go
- FOUND: commits 91f7e951, 84fe5df2, ec35e5a4, 4474fe15 on planning/ai-enhanced-cobol-graph
- PASS: Task 1 verify, Task 2 verify 1 and verify 2 (stock-grammar negative), plan verification (`go test -race ./internal/parser/languages/ ./cmd/gortex -run '^TestCobol'`), and every acceptance grep (build tags, `func cobolProgramSymbols(` once, use in cobol_grammar.go, one `copybook_standalone_analysis_unsupported`, one `preprocessor.NewContentID`, Windows test references `errCobolParserUnavailableOnWindows`)
