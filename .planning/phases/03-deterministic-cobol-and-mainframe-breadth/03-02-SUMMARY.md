---
phase: 03-deterministic-cobol-and-mainframe-breadth
plan: 02
subsystem: indexer
tags: [cobol, provenance, bom, prov-03, vcs, indexer-hook]

requires:
  - phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
    provides: stampSourceRevision D-18 indexer hook, prov_* range contract, always-on BOM strip transform
  - phase: 03-deterministic-cobol-and-mainframe-breadth (03-01)
    provides: byte-bearing breadth nodes and relationship edges that carry cloned docMeta (prov_revision_content_id)
provides:
  - Indexer.alignProvByteOffsets - BOM shift (+3 on int prov_start_byte/prov_end_byte) and indexed-bytes mismatch marking on nodes and edges
  - stampSourceRevision stamps prov_vcs_commit / prov_vcs_commit_absence on relationship edges carrying prov_revision_content_id
  - prov_range_absence value indexed_bytes_differ_from_file
  - TestSourceRevisionBOMShift, TestSourceRevisionStampEdges, TestCobolBOMByteOffsets
affects: [03-03, 03-05, 03-06, 03-07, 03-08, 03-09, 03-10, 03-13]

actuals:
  tokens: 2837
  tasks: 2
  commits: 3
plan_head_before: 88a64df50781092f2941bab5b94785ca4d0f85f7

tech-stack:
  added: []
  patterns:
    - "Provenance byte offsets are corrected to file offsets in the indexer hook, never in the extractor; rows and columns stay BOM-excluded"
    - "A byte offset is shifted only on an exact BOM-plus-indexed-bytes match; any other mismatch is marked prov_range_exact=false with a reason"
    - "Every revision stamp applies to nodes and relationship edges alike"

key-files:
  created:
    - internal/indexer/cobol_bom_test.go
  modified:
    - internal/indexer/source_revision.go
    - internal/indexer/source_revision_test.go

key-decisions:
  - "alignProvByteOffsets runs at the end of stampSourceRevision, after its nodes-only gate, so results without prov_revision_content_id still return early and indexer.go is untouched"
  - "On an indexed/disk mismatch, indexed_bytes_differ_from_file overwrites any earlier prov_range_absence (for example paragraph_extent_unmapped) on byte-bearing entries, because the bytes are not file offsets at all"
  - "Only int byte values are shifted; rows, columns, StartLine/EndLine, and edge Line never move (Assumption A6)"

patterns-established:
  - "Indexer-side provenance correction: the hook that knows rootPath and the indexed bytes owns any file-relative adjustment"

requirements-completed: [DET-01]

coverage:
  - id: D1
    description: "BOM-prefixed files carry true file byte offsets on nodes and edges; rows, columns, and lines unchanged"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionBOMShift/bom"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_bom_test.go#TestCobolBOMByteOffsets"
        status: pass
    human_judgment: false
  - id: D2
    description: "Indexed bytes that differ from the file in any other way mark every byte-bearing node and edge inexact with indexed_bytes_differ_from_file; equal or unreadable files are left as written"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionBOMShift/differ"
        status: pass
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionBOMShift/equal"
        status: pass
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionBOMShift/unreadable"
        status: pass
    human_judgment: false
  - id: D3
    description: "Relationship edges carrying prov_revision_content_id get exactly one of prov_vcs_commit or prov_vcs_commit_absence, matching their file's nodes"
    requirement: DET-01
    verification:
      - kind: unit
        ref: "internal/indexer/source_revision_test.go#TestSourceRevisionStampEdges"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-10-02
status: complete
---

# Phase 3 Plan 02: BOM Byte-Offset Alignment and Edge Revision Stamp Summary

**The D-18 indexer hook now turns COBOL provenance byte offsets into true file offsets (+3 when the file is exactly a UTF-8 BOM plus the indexed bytes, marked inexact with `indexed_bytes_differ_from_file` on any other mismatch) for nodes and edges, and stamps the VCS revision on relationship edges, closing the Phase 2 CR-01 residual**

## Performance

- **Duration:** about 10 min
- **Started:** 2026-10-02T05:52:11Z
- **Completed:** 2026-10-02T06:02:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- `alignProvByteOffsets` in `internal/indexer/source_revision.go`, called at the end of `stampSourceRevision`: one `os.ReadFile` per file that carries byte offsets; equal bytes or an unreadable file leave ranges alone; an exact BOM-plus-src match shifts int `prov_start_byte`/`prov_end_byte` by 3 on every node and edge; any other difference sets `prov_range_exact=false`, `prov_range_absence=indexed_bytes_differ_from_file`.
- `stampSourceRevision` applies the same commit/absence assignment to `result.Edges`, so the relationship edges 03-01 introduced carry `prov_vcs_commit` or `prov_vcs_commit_absence`, never both. The nodes-only gate and its early return are unchanged.
- `TestCobolBOMByteOffsets` proves the shift through `Indexer.Index` and the real grammar extractor: `bom/demo.cbl::DEMOPGM` reports bytes 10..128 against 7..125 for the plain copy, with equal rows, lines, and `prov_range_exact=true`.
- `indexer.go`, `transform.go`, and the extractor are unmodified.

## Task Commits

1. **Task 1: Align provenance byte offsets and stamp the VCS revision on relationship edges** - `8adeaa9d` (test, RED), `1ad4a981` (feat, GREEN)
2. **Task 2: Prove the BOM shift through the real grammar extractor and Indexer.Index** - `1a1ba85c` (test)

## Files Created/Modified
- `internal/indexer/source_revision.go` - edge stamping in `stampSourceRevision`; new `alignProvByteOffsets` with PURPOSE/RATIONALE/KEYWORDS doc
- `internal/indexer/source_revision_test.go` - `sourceRevisionResult` gains a revision-carrying edge and a bare structural edge; `TestSourceRevisionStampEdges`, `byteRangeResult`, `TestSourceRevisionBOMShift`
- `internal/indexer/cobol_bom_test.go` - `//go:build darwin || linux`; `TestCobolBOMByteOffsets` with an invented DEMOPGM fixture

## TDD Evidence
- **Task 1 RED:** `GOWORK=off go test -count=1 ./internal/indexer/ -run '^TestSourceRevision'` exited 1. Only the target subtests failed, on their own assertions: `TestSourceRevisionStampEdges/clean_committed` and `/modified` (edge carried no commit or absence), `TestSourceRevisionBOMShift/bom` (expected 13/43, actual 10/40), `/differ` (expected `prov_range_exact=false` and the absence reason, actual true/nil). `equal`, `unreadable`, and every pre-existing `TestSourceRevisionStamp` subtest passed. **GREEN:** `1ad4a981`; the plan's verify command passes under `-race`.
- **Task 2:** the behavior already existed after Task 1, so this test cannot fail on HEAD. To show it discriminates, it was run once against the pre-fix `source_revision.go` (from `8adeaa9d`, swapped in temporarily and restored byte-identical): it failed on the byte-shift assertions only (`prov_start_byte` expected 10 actual 7; `prov_end_byte` expected 128 actual 125). On the fixed code it passes under `-race`.
- `gsd-tools check tdd-red-evidence` parses node:test/TAP output, not `go test` output, so the RED records above were captured by hand (same as 03-01).

## Decisions Made
- The new sibling runs inside `stampSourceRevision` after its gate, not from `applyCoverageDomains`, so the hook stays a single call site and `indexer.go` is not touched.
- On a mismatch, `indexed_bytes_differ_from_file` overwrites an earlier range absence such as `paragraph_extent_unmapped` on byte-bearing entries. The plan specifies this, and it is the more fundamental reason: those bytes are not file offsets.
- Only `int` values are shifted. The extractor writes `int(r.Start.Byte)`, and the hook runs before persistence, so no other numeric type reaches it.

## Deviations from Plan

None - plan executed exactly as written. Task 2 has no separate GREEN commit because Task 1 already implemented the behavior; its RED was demonstrated against the pre-fix code instead (see TDD Evidence).

## TDD Gate Compliance
- RED `test(03-02)`: `8adeaa9d` precedes GREEN `feat(03-02)`: `1ad4a981`. No refactor commit was needed.
- Task 2's `1a1ba85c` is a test-only commit after GREEN, as described above.

## Issues Encountered
None.

## Known Stubs
None.

## Threat Flags
None. The only new surface is the file re-read covered by T-03-07 (path is `idx.rootPath` joined with the tracked relative path; bytes are never logged or returned). T-03-06 is enforced by the exact BOM-plus-src match. Fixtures are invented.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Every later Phase 3 plan that adds byte-bearing nodes or relationship edges inherits file-true offsets and the edge revision stamp with no extra work, as long as its entries carry `prov_start_byte` and `prov_revision_content_id`.
- Existing SQLite stores need a reindex to pick up corrected offsets on BOM-prefixed files (no version bump; the extractor output is unchanged).

## Self-Check: PASSED
- FOUND: internal/indexer/cobol_bom_test.go, internal/indexer/source_revision.go, internal/indexer/source_revision_test.go
- FOUND commits: 8adeaa9d, 1ad4a981, 1a1ba85c
- Plan verification: diff since 88a64df5 lists only the three planned files; `go test ./internal/indexer/` and `go test ./cmd/gortex -run '^TestCobolTracer_'` pass.

---
*Phase: 03-deterministic-cobol-and-mainframe-breadth*
*Completed: 2026-10-02*
