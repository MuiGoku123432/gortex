---
phase: 05-explicit-unresolved-evidence
plan: 15
subsystem: graph-store
tags: [resource, placeholder, eviction, sqlite, storetest, generation-fence, kind-guard, gap-closure]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.FindingNodeBatchEvicter, the SQLite finding evicter, and the finding_node_evict fence and conformance pattern (05-02)"
provides:
  - "graph.ResourceNodeBatchEvicter capability and graph.EvictResourceNodesByIDs dispatcher (supported=false for stores without it, no point-delete fallback)"
  - "(*graph.Graph).EvictResourceNodesByIDs: one-line delegation to evictSyntheticNodesByIDs with the KindResource kind set"
  - "(*store_sqlite.Store).EvictResourceNodesByIDs: one write transaction, chunked by contractNodeEvictChunkSize, kind and view_gen predicates on every statement"
  - "private (*store_sqlite.Store).evictNodesOfKindByIDs shared by the finding and resource evicters"
  - "Generation write-fence case resource_node_evict and capability checklist row"
  - "storetest conformance subtest ResourceNodeEvict on both backends"
affects: [05-16]

actuals:
  tokens: 3000
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Placeholder eviction is by ID and kind-guarded: only KindResource rows among the IDs go, with every incident edge; a node with an empty FilePath is evicted like any other resource"
    - "The Phase 5 SQLite evicters (finding, resource) are one-line delegations to evictNodesOfKindByIDs(kind, ids); the upstream config and contract evicters keep their own copies"

key-files:
  created:
    - internal/graph/resource_node_evict.go
    - internal/graph/resource_node_evict_test.go
    - internal/graph/store_sqlite/resource_node_evict.go
    - internal/graph/store_sqlite/resource_node_evict_test.go
  modified:
    - internal/graph/store_sqlite/finding_node_evict.go
    - internal/graph/store_sqlite/store_generation_read_test.go
    - internal/graph/store_sqlite/store_generation_write_test.go
    - internal/graph/storetest/storetest.go

key-decisions:
  - "Rule of three applied: the fourth kind-guarded SQLite evicter triggered extraction of evictNodesOfKindByIDs(kind, ids) in finding_node_evict.go; config_node_evict.go and contract_node_evict.go stay untouched for fork hygiene"
  - "The capability is language-neutral (KindResource); the COBOL missing_artifact and placeholder-namespace guard belongs to the 05-16 resolver, the store applies only the kind and generation guard"
  - "Tracer gate did not halt: auto mode is off but workflow.human_verify_mode is end-of-phase and the plan is autonomous, and the tracer verify is fully automated and passed"

patterns-established:
  - "A by-ID evicter for a new node kind is a public method that delegates to evictNodesOfKindByIDs, plus a writerFamilyFence checklist row, a TestGenerationScopedWriterFamilies case, and a storetest conformance case"

requirements-completed: [GAP-05, GAP-07]

coverage:
  - id: D1
    description: "In-memory resource evicter and dispatcher remove only KindResource nodes among mixed-kind IDs plus incident edges; duplicate and empty IDs count once or not at all; stores without the capability report unsupported; nil store or empty list is (0, 0, true)"
    requirement: GAP-05
    verification:
      - kind: unit
        ref: "internal/graph/resource_node_evict_test.go#TestEvictResourceNodesByIDs_InMemoryKindGuarded"
        status: pass
      - kind: unit
        ref: "internal/graph/resource_node_evict_test.go#TestEvictResourceNodesByIDs_Unsupported"
        status: pass
    human_judgment: false
  - id: D2
    description: "SQLite resource evicter is kind-guarded, set-oriented, and generation-fenced through the shared helper"
    requirement: GAP-05
    verification:
      - kind: unit
        ref: "internal/graph/store_sqlite/resource_node_evict_test.go#TestSQLiteEvictResourceNodesByIDsIsKindGuardedAndSetOriented"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/store_generation_write_test.go#TestGenerationScopedWriterFamilies/resource_node_evict"
        status: pass
      - kind: unit
        ref: "internal/graph/store_sqlite/store_generation_read_test.go#TestGenerationCapabilityChecklistIsComplete"
        status: pass
    human_judgment: false
  - id: D3
    description: "Finding eviction is unchanged after the helper extraction"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/graph/store_sqlite/finding_node_evict_test.go#TestSQLiteEvictFindingNodesByIDsIsKindGuardedAndSetOriented"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/store_generation_write_test.go#TestGenerationScopedWriterFamilies/finding_node_evict"
        status: pass
      - kind: integration
        ref: "internal/graph/store_sqlite/store_test.go#TestSQLiteStoreConformance/FindingNodeEvict"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both backends pass the shared ResourceNodeEvict conformance case through graph.EvictResourceNodesByIDs"
    requirement: GAP-05
    verification:
      - kind: integration
        ref: "internal/graph/store_sqlite/store_test.go#TestSQLiteStoreConformance/ResourceNodeEvict"
        status: pass
      - kind: integration
        ref: "internal/graph/storetest/memory_conformance_test.go#TestMemoryStoreConformance/ResourceNodeEvict"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-10-23
status: complete
---

# Phase 5 Plan 15: Resource Node Eviction Summary

**Kind-guarded, by-ID resource eviction on both store backends (`graph.EvictResourceNodesByIDs`), with the SQLite finding and resource evicters now sharing one private `evictNodesOfKindByIDs` helper. The D-27 placeholder sweep (05-16) can remove missing-artifact placeholders by ID, never by file path.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-10-23T15:10:34Z
- **Completed:** 2026-10-23T15:20:30Z
- **Tasks:** 2
- **Files modified:** 8 (4 created, 4 modified)

## Accomplishments

- `graph.ResourceNodeBatchEvicter`, the `EvictResourceNodesByIDs` dispatcher (nil store or empty IDs give `(0, 0, true)`, a store without the capability gives `(0, 0, false)`), and the `*Graph` method as a single `evictSyntheticNodesByIDs` delegation with `KindResource`.
- `(*store_sqlite.Store).evictNodesOfKindByIDs(kind, ids)`: the former finding evicter body, binding `string(kind)`. It dedupes IDs, runs one write transaction in chunks of `contractNodeEvictChunkSize`, and puts the kind and `view_gen` predicates on both edge deletes and the node delete. `EvictFindingNodesByIDs` and the new `EvictResourceNodesByIDs` are one-line delegations to it.
- Generation fence: `resource_node_evict` writer-family case (an empty-FilePath placeholder plus one `calls` edge in both generations, evicted through the derived handle) and the `ResourceNodeBatchEvicter` checklist row.
- Conformance: `ResourceNodeEvict` runs on the memory and SQLite backends. A finding named in the same ID list survives, the finding's own `member_of` edge survives, and both edges incident to the resource go.

## Task Commits

1. **Task 1 (tracer): EvictResourceNodesByIDs through the dispatcher on both backends** - `e967546f` (test, RED), `aafce4d6` (feat, GREEN)
2. **Task 2: Generation checklist, write fence, and storetest conformance** - `f206d251` (test, RED: fence and conformance cases, checklist failing), `1c935ef7` (test, GREEN: checklist row)

**Plan metadata:** recorded in the docs(05-15) commit that adds this file.

## Files Created/Modified

- `internal/graph/resource_node_evict.go` - capability interface, dispatcher, `*Graph` delegation, compile-time assertion
- `internal/graph/resource_node_evict_test.go` - in-memory kind-guard and unsupported-store tests
- `internal/graph/store_sqlite/finding_node_evict.go` - private `evictNodesOfKindByIDs`; `EvictFindingNodesByIDs` delegates; third-occurrence note replaced
- `internal/graph/store_sqlite/resource_node_evict.go` - `(*Store).EvictResourceNodesByIDs` and its compile-time assertion
- `internal/graph/store_sqlite/resource_node_evict_test.go` - SQLite kind-guard and set-orientation test
- `internal/graph/store_sqlite/store_generation_read_test.go` - capability checklist row
- `internal/graph/store_sqlite/store_generation_write_test.go` - `genWriteResource` constant and `resource_node_evict` case
- `internal/graph/storetest/storetest.go` - `ResourceNodeEvict` registration and `testResourceNodeEvict`

## Decisions Made

- Extracted the shared helper per the rule of three. Only the two Phase 5 files use it. The `// note:` on the helper records why `config_node_evict.go` and `contract_node_evict.go` keep their copies.
- The unit tests add a fourth edge that does not touch the resource and assert it survives. This proves the edge delete is limited to edges incident to evicted resources.
- `testResourceNodeEvict` seeds the finding through `mkRepoNode`, never a `KindFinding` composite literal, so the D-16 finding-writer inventory stays clean. The new `_test.go` files are outside the inventory scan.

## Deviations from Plan

None. The plan executed as written.

## TDD Gate Compliance

- Task 1: `e967546f` failed to build against the missing `EvictResourceNodesByIDs` symbols on both packages. `aafce4d6` made them pass.
- Task 2: the fence and conformance cases passed at `f206d251` because Task 1 had already shipped the capability. The RED target was `TestGenerationCapabilityChecklistIsComplete`, which failed with "graph.ResourceNodeBatchEvicter is asserted on *Store but missing from the generation checklist". `1c935ef7` added the row and turned it green. Both Task 2 commits are `test(...)` because every changed file is test-only.

## Tracer Gate

Auto mode is off (`_auto_chain_active` and `auto_advance` are false). `workflow.human_verify_mode` is `end-of-phase` and the plan is `autonomous: true`, so the run did not halt mid-flight. The tracer `<verify>` is fully automated and passed end-to-end before Task 2 started.

## Issues Encountered

None.

## Verification

- Task 1 verify: `TestEvictResourceNodesByIDs_InMemoryKindGuarded` listed. `TestEvict(Resource|Finding)NodesByIDs_*` (4 tests) and `TestSQLiteEvict(Resource|Finding)NodesByIDsIsKindGuardedAndSetOriented` all PASS under `-race`. gofmt is clean. Acceptance greps returned 1, 3, 1 and 1.
- Task 2 verify: the `resource_node_evict` and `finding_node_evict` fence cases, `TestGenerationCapabilityChecklistIsComplete`, `TestGenerationWriteFenceNamesAreReal`, and the SQLite and memory `ResourceNodeEvict` and `FindingNodeEvict` conformance subtests all PASS under `-race`. gofmt is clean. Acceptance greps returned 1, 1 and 3.
- Plan verification:
  - `GOWORK=off go test -race -count=1 ./internal/graph/ ./internal/graph/storetest/` passed.
  - `GOWORK=off go test -race -count=1 -timeout 30m ./internal/graph/store_sqlite/ -run 'Evict|Generation|Conformance'` passed (203 s).
  - `GOWORK=off go build ./...` succeeded.
  - `git diff --name-only 33d287d4..HEAD` lists exactly the 8 planned files with no deletions. `config_node_evict.go` and `contract_node_evict.go` are unchanged.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 05-16 can call `graph.EvictResourceNodesByIDs(store, staleIDs)` for the D-27 placeholder sweep. It must treat `supported == false` as an error rather than fall back to point deletes or `EvictFile`. It must pass only IDs it read with `cobol_kind=missing_artifact` in the placeholder namespace during the same reconcile pass, because the store applies only the kind and generation guard.
- GAP-05 and GAP-07 stay pending in REQUIREMENTS.md. The phase verifier re-runs after all gap plans.

## Self-Check: PASSED

- FOUND: internal/graph/resource_node_evict.go, internal/graph/resource_node_evict_test.go, internal/graph/store_sqlite/resource_node_evict.go, internal/graph/store_sqlite/resource_node_evict_test.go, internal/graph/store_sqlite/finding_node_evict.go
- FOUND commits: e967546f, aafce4d6, f206d251, 1c935ef7
- Both task verify commands and the plan verification passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-23*
