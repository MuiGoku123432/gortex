---
phase: 05-explicit-unresolved-evidence
plan: 02
subsystem: graph-store
tags: [finding, eviction, sqlite, storetest, generation-fence, kind-guard]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.KindFinding and the finding ID grammar (05-01)"
provides:
  - "graph.FindingNodeBatchEvicter capability and graph.EvictFindingNodesByIDs dispatcher (supported=false for stores without it, no N+1 fallback)"
  - "(*graph.Graph).EvictFindingNodesByIDs: one-line delegation to evictSyntheticNodesByIDs with the KindFinding kind set"
  - "(*store_sqlite.Store).EvictFindingNodesByIDs: one write transaction, chunked by contractNodeEvictChunkSize, kind and view_gen predicates on every statement"
  - "Generation write-fence case finding_node_evict and capability checklist row"
  - "storetest conformance subtest FindingNodeEvict on both backends"
affects: [05-08, 05-09, 05-12]

actuals:
  tokens: 4667
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Finding eviction is by ID and kind-guarded: only KindFinding rows among the IDs go, with every incident edge; other kinds named in the list survive"
    - "storetest seeds KindFinding only through mkRepoNode, never a &graph.Node{Kind: graph.KindFinding} literal, so the 05-12 D-16 writer inventory stays clean"

key-files:
  created:
    - internal/graph/finding_node_evict.go
    - internal/graph/finding_node_evict_test.go
    - internal/graph/store_sqlite/finding_node_evict.go
    - internal/graph/store_sqlite/finding_node_evict_test.go
  modified:
    - internal/graph/store_sqlite/store_generation_read_test.go
    - internal/graph/store_sqlite/store_generation_write_test.go
    - internal/graph/storetest/storetest.go

key-decisions:
  - "The SQLite body is a deliberate third copy of the config/contract evicter with a 'note: third occurrence' comment; fork hygiene wins over editing two upstream files, and a later upstream-sync phase may extract one helper"
  - "The in-memory side reuses evictSyntheticNodesByIDs, so no new shard-lock loop exists"
  - "Go 1.27 gofmt reflows multi-line composite literals in return statements; store_generation_write_test.go was reformatted (whitespace only) to keep the verify gofmt gate clean"

patterns-established:
  - "A new mutating SQLite capability needs both a writerFamilyFence checklist row and a TestGenerationScopedWriterFamilies case"

requirements-completed: [GAP-02, GAP-07]

coverage:
  - id: D1
    description: "In-memory finding evicter and dispatcher remove only KindFinding nodes among the IDs plus incident edges; duplicate and empty IDs count once or not at all; stores without the capability report unsupported"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/graph/finding_node_evict_test.go#TestEvictFindingNodesByIDs_InMemoryKindGuarded"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_node_evict_test.go#TestEvictFindingNodesByIDs_Unsupported"
        status: pass
    human_judgment: false
  - id: D2
    description: "SQLite finding evicter is kind-guarded, set-oriented, and generation-fenced"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/graph/store_sqlite/finding_node_evict_test.go#TestSQLiteEvictFindingNodesByIDsIsKindGuardedAndSetOriented"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/store_generation_write_test.go#TestGenerationScopedWriterFamilies/finding_node_evict"
        status: pass
      - kind: unit
        ref: "internal/graph/store_sqlite/store_generation_read_test.go#TestGenerationCapabilityChecklistIsComplete"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both backends pass the shared FindingNodeEvict conformance case through graph.EvictFindingNodesByIDs"
    requirement: GAP-02
    verification:
      - kind: integration
        ref: "internal/graph/store_sqlite/store_test.go#TestSQLiteStoreConformance/FindingNodeEvict"
        status: pass
      - kind: integration
        ref: "internal/graph/storetest/memory_conformance_test.go#TestMemoryStoreConformance/FindingNodeEvict"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 02: Finding Node Eviction Summary

**Kind-guarded, ID-set finding eviction on both store backends (`graph.EvictFindingNodesByIDs`): only `finding` rows among the IDs are deleted, with every incident edge, in one bounded SQLite transaction fenced by `view_gen`, ready for the 05-08 resolver's desired-set reconcile.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-10-20T16:00:37Z
- **Completed:** 2026-10-20T16:10:00Z
- **Tasks:** 2
- **Files modified:** 7 (4 created, 3 modified)

## Accomplishments

- `graph.FindingNodeBatchEvicter`, the `EvictFindingNodesByIDs` dispatcher (nil store or empty IDs give `(0, 0, true)`, a store without the capability gives `(0, 0, false)`), and the `*Graph` method as a single `evictSyntheticNodesByIDs` delegation.
- `(*store_sqlite.Store).EvictFindingNodesByIDs`: dedupes IDs, one write transaction, chunks of `contractNodeEvictChunkSize`, `kind = 'finding'` and `view_gen` on every edge and node delete, analysis invalidation and receipt marking like its siblings.
- Generation fence: `finding_node_evict` write-fence case (finding plus one `member_of` edge in both generations, evicted through one) and the `FindingNodeBatchEvicter` checklist row.
- Conformance: `FindingNodeEvict` subtest runs on the memory and SQLite backends; a resource named in the same ID list and a program survive, the program's own reference edge survives.

## Task Commits

1. **Task 1: In-memory finding evicter and dispatcher** - `0efa3dee` (test, RED), `8fa30826` (feat, GREEN)
2. **Task 2: SQLite finding evicter with generation fence and two-backend conformance** - `f1fae480` (test, RED), `2b4d16d6` (feat, GREEN)

**Plan metadata:** recorded in the docs(05-02) commit that adds this file.

## Files Created/Modified

- `internal/graph/finding_node_evict.go` - capability interface, dispatcher, `*Graph` delegation
- `internal/graph/finding_node_evict_test.go` - in-memory kind-guard and unsupported-store tests
- `internal/graph/store_sqlite/finding_node_evict.go` - SQLite transactional evicter (third-occurrence copy)
- `internal/graph/store_sqlite/finding_node_evict_test.go` - SQLite kind-guard and set-orientation test
- `internal/graph/store_sqlite/store_generation_read_test.go` - capability checklist row
- `internal/graph/store_sqlite/store_generation_write_test.go` - `genWriteFinding` constant and `finding_node_evict` case; gofmt reflow
- `internal/graph/storetest/storetest.go` - `FindingNodeEvict` registration and `testFindingNodeEvict`

## Decisions Made

- Kept the SQLite body as a copy with the `// note: third occurrence` comment (plan flagged assumption): extracting a shared helper would edit two upstream files.
- The storetest case also asserts a non-incident edge survives (`EdgeCount() == 1`), which proves the edge delete is limited to edges incident to evicted findings.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pre-existing gofmt drift in store_generation_write_test.go**
- **Found during:** Task 2 (RED run)
- **Issue:** At HEAD, Go 1.27 `gofmt -l` already flagged the file: three multi-line composite literals in `return` statements (the `config_node_evict` and `derived_contract_replace` seeds and `generationWriteContractMaterial`) used the older indentation. The task verify requires `gofmt -l` to print nothing for this file.
- **Fix:** `gofmt -w` on the file, as the plan's action instructs. Whitespace only.
- **Files modified:** internal/graph/store_sqlite/store_generation_write_test.go
- **Verification:** `gofmt -l` prints nothing; the fence cases pass
- **Committed in:** f1fae480

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Whitespace only. No behavior change.

## TDD Gate Compliance

Both tasks followed RED then GREEN: `0efa3dee` and `f1fae480` failed to build against the missing methods; `8fa30826` and `2b4d16d6` made them pass. The memory backend's `FindingNodeEvict` conformance subtest passed at the RED commit because Task 1 had already shipped the in-memory capability; the SQLite side was the RED target.

## Issues Encountered

None.

## Verification

- Task 1 verify: both tests listed and passing under `-race`, with `EvictConfig|EvictContract`; gofmt clean. Acceptance greps: delegation 1, `lockAllWrite` 0.
- Task 2 verify: kind-guard test, `finding_node_evict` and `config_node_evict` fence cases, checklist and fence-name tests, SQLite `FindingNodeEvict` all PASS under `-race`; `internal/graph/storetest` passes; gofmt clean. Acceptance greps: 1, 1, 1, 1, 0, 1.
- Plan verification: `GOWORK=off go test -race -count=1 -timeout 30m ./internal/graph/store_sqlite/ -run 'EvictFindingNodesByIDs|Generation|Conformance'` passed (150 s). `go build ./...` and `go test -race ./internal/graph/` pass.
- `git diff --name-only 3b6eab13..HEAD` lists exactly the 7 planned files; both `config_node_evict.go` and `contract_node_evict.go` (both backends) are unchanged.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 05-08 can call `graph.EvictFindingNodesByIDs(store, undesired)` for its desired-set reconcile and must treat `supported == false` as an error, not fall back to point deletes.
- GAP-02 and GAP-07 stay pending in REQUIREMENTS.md: sibling plans also declare them.

## Self-Check: PASSED

- FOUND: internal/graph/finding_node_evict.go, internal/graph/finding_node_evict_test.go, internal/graph/store_sqlite/finding_node_evict.go, internal/graph/store_sqlite/finding_node_evict_test.go
- FOUND commits: 0efa3dee, 8fa30826, f1fae480, 2b4d16d6
- Both task verify commands and the plan verification passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
