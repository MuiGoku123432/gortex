---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 01
subsystem: testing
tags: [cobol, graphdump, blake3, sqlite, determinism, shared-resources, db2, idms, cics]

requires:
  - phase: 03-deterministic-cobol-and-mainframe-breadth
    provides: COBOL grammar extractor with shared DB2/IDMS/CICS resource nodes (D-15) and cobol-name-norm/1
provides:
  - internal/testutil/graphdump (Volatile, DeterministicVolatile graphdump-volatile/1, Result, Dump, FirstDiff)
  - internal/indexer/cobol_lifecycle_test.go D-04 harness (cobolLifecycleEnv, newCobolLifecycleEnv, track, dump, cleanCobolDump, requireSameDump, cobolLifecycleRepoA/B)
  - TestCobolLifecycle_CleanDeterminism (rebuild, track_order, shared_resources, no_stash_keys)
  - writer-independent shared resource nodes (Name = normalized key, FilePath = own ID, cobolSharedResourceKeys only)
  - edge Meta cobol_resource_raw on every edge into a shared resource
affects: [04-02, 04-06, 04-07, 04-08, 04-09, 04-10, 04-11, 04-14, 04-15, 04-16, neo4j-projection]

actuals:
  tokens: 7028
  tasks: 1
  commits: 2
plan_head_before: 320c44cf82419d43e7a24cc0a0382773b5378b5b

tech-stack:
  added: []
  patterns:
    - "Canonical dump through graph.ScopedProjectionSnapshotOpener: sorted N/E rows, 0x1F separators, json.Marshal Meta minus a versioned, justified Volatile allowlist, BLAKE3 hex digest"
    - "Shared resource nodes own a virtual path (FilePath = ID) and carry only writer-independent provenance"

key-files:
  created:
    - internal/testutil/graphdump/graphdump.go
    - internal/indexer/cobol_lifecycle_test.go
  modified:
    - internal/parser/languages/cobol_grammar_walk.go
    - internal/parser/languages/cobol_grammar_mainframe.go
    - internal/parser/languages/cobol_grammar_sql.go
    - internal/parser/languages/cobol_grammar_mainframe_test.go
    - internal/parser/languages/cobol_grammar_sql_test.go
    - internal/parser/languages/cobol_grammar_contract_test.go

key-decisions:
  - "Shared resource Meta keeps exactly nine document keys (cobolSharedResourceKeys): prov_parser_tool_id, prov_parser_grammar_id, prov_parser_module, prov_preprocessor_module, prov_handoff_schema, prov_extractor_version, prov_evidence_class, prov_origin, prov_confidence. The harness showed program and copybook documents write the same values for all nine (tool ID and handoff schema included), and different values for grade policy and transform config, which are dropped."
  - "DeterministicVolatile ships with no keys: two clean builds and opposite track orders produced identical dumps without excluding any Meta key, so nothing was allowlisted."
  - "DB2 node Meta table and schema now hold the cobolSQLName-normalized values, matching Name; the as-written spelling is cobol_resource_raw on the edge (first spelling in a statement wins, with the to-plus-kind dedupe)."

patterns-established:
  - "Lifecycle harness: newCobolLifecycleEnv writes invented repos, opens SQLite plus MultiIndexer, tracks with Workspace/Project phase4, and dumps with graphdump; later Phase 4 plans add their own test files on top of it"
  - "assertCobolSharedProvenance is the single shared-resource provenance check used by the IDMS/CICS, DB2, and ProvKeys tests"

requirements-completed: [ID-06, ID-04]

coverage:
  - id: D1
    description: "Test-only canonical dump helper with versioned, justified volatile allowlist, BLAKE3 digest, and FirstDiff (D-01, D-02, D-03)"
    requirement: ID-06
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_test.go#TestCobolLifecycle_CleanDeterminism"
        status: pass
      - kind: other
        ref: "GOWORK=off go vet ./internal/testutil/graphdump/ and the no-production-importer grep"
        status: pass
    human_judgment: false
  - id: D2
    description: "Two clean builds and two opposite track orders of an invented two-repository COBOL corpus give one digest on the real path (extractor, MultiIndexer.TrackRepoCtx, SQLite) under -race"
    requirement: ID-06
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_test.go#TestCobolLifecycle_CleanDeterminism/rebuild,track_order,no_stash_keys"
        status: pass
    human_judgment: false
  - id: D3
    description: "Shared DB2/IDMS/CICS nodes are writer-independent: Name is the normalized key, FilePath is the node ID, Meta has only writer-independent keys, and every in-edge records cobol_resource_raw (CR-02, D-18)"
    requirement: ID-04
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_test.go#TestCobolLifecycle_CleanDeterminism/shared_resources"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race ./internal/parser/languages/ -run '^TestCobol'"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 01: Canonical Dump and Deterministic Clean COBOL Builds Summary

**A BLAKE3-digested canonical dump of the scoped SQLite subgraph shows that two clean builds of a real two-repository COBOL corpus, in either track order, are identical. Shared DB2/IDMS/CICS nodes were made writer-independent to get there: FilePath is the node's own ID, Name is the normalized key, Meta keeps only writer-independent provenance, and the spelling as written is kept on each edge in `cobol_resource_raw`.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-10-13T00:57:19Z
- **Completed:** 2026-10-13T01:23:13Z
- **Tasks:** 1 (tracer, TDD)
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- `internal/testutil/graphdump`: `Dump` reads only through `graph.ScopedProjectionSnapshotOpener`. It renders sorted `N`/`E` rows (ID, kind, name, file path, full range; from, to, kind, file, line), each with canonical JSON Meta minus a versioned, justified `Volatile` allowlist, and returns a BLAKE3 hex digest. `Dump` refuses an unversioned allowlist or an unjustified key. `FirstDiff` reports the first differing rows. The package doc forbids production imports.
- `internal/indexer/cobol_lifecycle_test.go`: the D-04 harness (`cobolLifecycleEnv`, `newCobolLifecycleEnv`, `track`, `dump`, `cleanCobolDump`, `requireSameDump`, fixtures `cobolLifecycleRepoA`/`B`) and `TestCobolLifecycle_CleanDeterminism` with the subtests `rebuild`, `track_order`, `shared_resources`, and `no_stash_keys`.
- CR-02 fix (D-18): shared resource nodes no longer depend on whichever program or copybook wrote them last. Before the fix, the RED run showed rebuild digests differing only on `db2::`/`idms::` rows, with FilePath `lifea/cobol/main.cbl` in one build and `lifea/copy/valpara.cpy` in the other.

## Task Commits

1. **Task 1 (tracer): canonical dump, harness, CR-02 fix**
   - RED: `bf119200` (test) - harness and graphdump; the target test failed on shared-resource assertions (`RED_EVIDENCE_OK`, reason `target_test_failed`)
   - GREEN: `482707c0` (feat) - writer-independent `resourceNode`, normalized names, `cobol_resource_raw`, updated extractor test helpers
   - REFACTOR: none needed

## Files Created/Modified

- `internal/testutil/graphdump/graphdump.go` - test-only canonical dump, digest, Volatile allowlist, FirstDiff
- `internal/indexer/cobol_lifecycle_test.go` - D-04 harness and the clean-determinism test
- `internal/parser/languages/cobol_grammar_walk.go` - `cobolSharedResourceKeys` and the new `resourceNode` contract
- `internal/parser/languages/cobol_grammar_mainframe.go` - IDMS and CICS literal names pass `cobolNameNorm`; edges carry `cobol_resource_raw`
- `internal/parser/languages/cobol_grammar_sql.go` - `cobolSQLTable.raw`; normalized table and schema Name/Meta; edges carry `cobol_resource_raw`
- `internal/parser/languages/cobol_grammar_mainframe_test.go` - `cobolDocumentOnlyKeys`, `assertCobolSharedProvenance`, updated `assertCobolResource`
- `internal/parser/languages/cobol_grammar_sql_test.go` - `assertCobolTable` uses the shared-provenance check
- `internal/parser/languages/cobol_grammar_contract_test.go` - `TestCobolGrammar_ProvKeys` checks shared nodes against the retained key set

## Decisions Made

- The retained shared-resource provenance keys were confirmed from real output, not assumed. Both analysis modes write the same tool ID `cf00f13c…` and handoff schema `cobol-handoff-v1`. Grade policy (`cobol-grade-policy-v1` vs `cobol-grade-policy-copybook-v1`) and transform config differ, so they are dropped. None of the nine planned keys had to be removed.
- `DeterministicVolatile` stays empty. No harness diff showed a volatile key, and per D-01 nothing is allowlisted without that evidence.
- The `/4` extractor-version pins are unchanged. 04-11 bumps them to `gortex-cobol-grammar/5`.

## Deviations from Plan

### Auto-fixed Issues

None.

### Minor implementation choices

- The D-01 allowlist refusal checks (empty Version, empty justification) run inline at the end of `TestCobolLifecycle_CleanDeterminism` instead of in a fifth subtest, which keeps the planned four subtests. 04-11's `TestGraphdump_Dump` locks this behavior in its own test.
- `FirstDiff` prints the 0x1F separator as ` | ` so failure output is readable. The rows and digest are unchanged.
- `assertCobolSharedProvenance` is a new test helper used from three test files (rule of three).

**Total deviations:** 0 auto-fixed. **Impact:** none; the plan executed as written.

## Issues Encountered

- `go test -race ./internal/indexer/` as a whole package exceeds Go's default 10-minute test timeout on this machine. The goroutine dump was in the unrelated `TestRefViewCoalescesConcurrentSelections`. This is pre-existing package runtime, not a regression: the full package passes without `-race` (`ok … 306s`), and the plan's scoped `-race` runs pass.

## TDD Gate Compliance

- RED `test(04-01)` commit `bf119200` precedes GREEN `feat(04-01)` commit `482707c0`. The RED evidence was verified by `check tdd-red-evidence`: `RED_EVIDENCE_OK`.

## Verification

- Plan verify command: exit 0 (`-list` finds `TestCobolLifecycle_CleanDeterminism`; `TestCobolLifecycle_*`, `TestCobol*` in languages, and `TestCobol(Tracer|Breadth)_*` pass under `-race`; `go vet` is clean; `gofmt -l` is empty for all 8 files).
- `TestCobolLifecycle_*` passed 4 times in a row under `-race` (`-count=4`), so the result is stable.
- Acceptance greps: `func Dump(`, `func FirstDiff(`, and `graphdump-volatile/1` each match once. No production importer of `internal/testutil/graphdump`.
- `./internal/neo4jprojection/` passes under `-race`. `cobol_resource_raw` passes the `sensitiveMetadataKey` screen (no sensitive substrings).

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for 04-02 and the other wave-1 plans. Later plans add their own `cobol_lifecycle_*_test.go` files on top of `newCobolLifecycleEnv`, `cleanCobolDump`, and `requireSameDump`.
- Orphaned shared resource nodes (no remaining in-edges) are not collected yet. That GC is in the resolver reconcile (04-06/04-14), as the new `resourceNode` doc comment states.

## Self-Check: PASSED
