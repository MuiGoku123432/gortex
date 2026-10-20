---
phase: 05-explicit-unresolved-evidence
plan: 04
subsystem: mcp-health-and-projection
tags: [finding, index_health, neo4j, projection, schema, gap-03, gap-04]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.KindFinding, FindingClassParse/External, MetaEvidenceClass and finding Meta key constants (05-01)"
provides:
  - "index_health payload key findings = {PARSE_UNRESOLVED: n, EXTERNAL_UNRESOLVED: m}, active KindFinding nodes counted by prov_evidence_class over the file scan's repository set; both keys always present"
  - "gortex://schema node-kind entry for finding (ID convention, Meta keys, member_of and references edges)"
  - "TestProjectNode_FindingMeta: finding and missing-artifact placeholder Meta proven projection-safe; Neo4j labels Finding and GortexUnresolved pinned"
  - "assertProjectedMeta test helper in internal/neo4jprojection/properties_test.go"
affects: [05-05, 05-08, 05-11, 05-12, phase-06]

actuals:
  tokens: 4500
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "index_health reported-never-enforced blocks are added after result is built and never touch healthScore or recommendation"
    - "Projection key-by-key Meta proof goes through assertProjectedMeta"

key-files:
  created:
    - internal/mcp/index_health_findings_test.go
  modified:
    - internal/mcp/tools_enhancements.go
    - internal/mcp/resources.go
    - internal/neo4jprojection/properties_test.go

key-decisions:
  - "index_health findings counts read prov_evidence_class, never the ID segment, and only count nodes whose RepoPrefix the file scan collected; any other class value is ignored"
  - "The Neo4j kind label for findings is Finding (sanitizeLabel capitalizes); Phase 6 and export consumers must match :Finding, not :finding"
  - "No projection change was needed: every finding_*, artifact_*, and prov_* key passes the secret filter and key encoding unchanged"

patterns-established:
  - "assertProjectedMeta(t, node) is the one key-by-key projection check; new Meta contracts add a node and call it"

requirements-completed: [GAP-03, GAP-04]

coverage:
  - id: D1
    description: "index_health reports active finding counts by class over the payload's repository set; a finding outside that set and a node with another class value are not counted; both keys are 0 with no findings"
    requirement: GAP-03
    verification:
      - kind: unit
        ref: "internal/mcp/index_health_findings_test.go#TestIndexHealth_FindingCounts"
        status: pass
    human_judgment: false
  - id: D2
    description: "The same store with and without findings yields the same health_score and recommendation (D-14, P4 D-15)"
    requirement: GAP-03
    verification:
      - kind: unit
        ref: "internal/mcp/index_health_findings_test.go#TestIndexHealth_FindingsDoNotScore"
        status: pass
    human_judgment: false
  - id: D3
    description: "A finding node with every 05-01 key and a placeholder KindResource node project every Meta key as an equal meta_<key> property with zero secret and zero unsupported warnings; labels Finding and GortexUnresolved"
    requirement: GAP-04
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/properties_test.go#TestProjectNode_FindingMeta"
        status: pass
    human_judgment: false
  - id: D4
    description: "gortex://schema lists the finding node kind with its ID convention and edges"
    verification:
      - kind: other
        ref: "grep -c -- '- finding ' internal/mcp/resources.go == 1; GOWORK=off go test -race ./internal/mcp/ -run 'Schema|CatalogHelp|Resource'"
        status: pass
    human_judgment: false

duration: 16min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 04: Finding Visibility in index_health and Projection Summary

**`index_health` now carries `findings = {PARSE_UNRESOLVED, EXTERNAL_UNRESOLVED}` counted from `prov_evidence_class` over its own repository set, without touching the health score or recommendation; finding and placeholder Meta are proven to survive Neo4j projection intact (labels `Finding` and `GortexUnresolved`), and `gortex://schema` documents the `finding` kind.**

## Performance

- **Duration:** ~16 min
- **Started:** 2026-10-20T16:27:13Z
- **Completed:** 2026-10-20T16:43:14Z
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `buildIndexHealthBasePayloadCtx` walks `NodesByKind(graph.KindFinding)` after the file scan, with the same `healthScanCancelStride` context check and nil check. It counts only nodes whose `RepoPrefix` the file scan collected, keyed by `Meta[prov_evidence_class]`.
- `result["findings"]` is set after `result` is built, next to `repo_ownership`. Its comment records D-08 visibility and D-14/P4 D-15 (reported, never enforced). Score and recommendation code is untouched.
- `TestProjectNode_FindingMeta` projects a finding that carries all 16 finding keys plus the docMeta and range `prov_*` keys, and a placeholder with the 05-05 shape. Both have zero secret and zero unsupported warnings, and every key appears unchanged as `meta_<key>`.
- `handleResourceSchema` lists `finding` after `contract_bridge`. The entry gives the ID convention, the class and finding Meta keys, and the `member_of` and `references` edges.

## Task Commits

1. **Task 1: index_health reports active finding counts by class and never scores them** - `fdd4c2b9` (test, RED), `13116579` (feat, GREEN)
2. **Task 2: Projection keeps finding and placeholder Meta; schema lists the finding kind** - `81c61f18` (test), `345240d3` (feat)

**Plan metadata:** recorded in the docs(05-04) commit that adds this file.

## Files Created/Modified

- `internal/mcp/index_health_findings_test.go` - `readIndexHealth` (the ownership test's closure as a file-local function), two-repository fixture builders, `TestIndexHealth_FindingCounts` (`counts_by_class`, `none`), `TestIndexHealth_FindingsDoNotScore`
- `internal/mcp/tools_enhancements.go` - finding count loop after the file scan; `result["findings"]` block after `repo_ownership`
- `internal/neo4jprojection/properties_test.go` - `TestProjectNode_FindingMeta`; `assertProjectedMeta` extracted and reused by `TestProjectNode_CobolProvenance` and `TestProjectNode_CobolBreadthMeta`
- `internal/mcp/resources.go` - `finding` node-kind entry in the schema text

## Decisions Made

- Counts read `prov_evidence_class`, not the ID segment. A malformed ID cannot shift a count, and a node with any other class value (tested with `RESOLVED`) is ignored.
- The repository restriction is tested directly: a finding in `r9`, which has no file node, is not counted (T-05-11).
- The DoNotScore fixture holds a `defines` edge in both stores. Otherwise the findings' `member_of` edges would flip `edges_ok` and change the recommendation for a reason that has nothing to do with findings.
- The Neo4j label is `Finding`, because `sanitizeLabel` capitalizes. The test pins that literal, since it is what Neo4j queries must match.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - CLAUDE.md convention] Extracted assertProjectedMeta on its third use**
- **Found during:** Task 2
- **Issue:** The key-by-key projection check existed twice already. `TestProjectNode_CobolBreadthMeta` had a `note: second occurrence` comment. The new test would add a third copy, and the project rule of three requires extraction at three.
- **Fix:** Added a test-local `assertProjectedMeta(t, node) projectedNode` in `properties_test.go`. All three tests now call it, and it returns the projection for label checks. The check logic is unchanged.
- **Files modified:** internal/neo4jprojection/properties_test.go
- **Verification:** `TestProjectNode_CobolProvenance`, `TestProjectNode_CobolBreadthMeta`, and `TestProjectNode_FindingMeta` pass under `-race`.
- **Committed in:** 81c61f18

---

**Total deviations:** 1 auto-fixed (1 convention-driven)
**Impact on plan:** Test-file refactor only, inside a planned file. No scope creep.

## TDD Gate Compliance

- Task 1 ran RED then GREEN. In `fdd4c2b9`, all three subtests failed with "index_health must carry a findings block". `13116579` made them pass.
- Task 2 is a proof task. The plan expected no production change for the projection. `TestProjectNode_FindingMeta` failed on its first run, but only on my own wrong label assumption (`finding` vs. the projected `Finding`). The test was fixed, not the projection, and then it passed. The schema entry's RED was its grep (count 0 before `345240d3`), because `catalog_help_test.go` is outside this plan's file list.

## Issues Encountered

- Kind labels project capitalized: `finding` becomes `Finding`, `resource` becomes `Resource`. The plan's wording ("the `finding` kind label") reads as the raw kind. The behavior is pre-existing and consistent across kinds, so nothing changed in production.

## Verification

- Task 1 verify: both tests are listed; `go test -race ./internal/mcp/ -run IndexHealth` passes; gofmt is clean. Acceptance greps print 1 and 1.
- Task 2 verify: the test is listed; projection tests pass under `-race`; MCP `Schema|CatalogHelp|Resource` pass; the schema grep prints 1; gofmt is clean.
- Plan verification:
  - `go build ./...` passes.
  - `GOWORK=off go test -race -count=1 ./internal/mcp/ ./internal/neo4jprojection/` passes (544 s and 9 s).
  - `git diff --name-only d3494164..HEAD` lists exactly the 4 planned files.

## Threat Mitigations

- T-05-11: counts are restricted to `repoPrefixes` and only integers are reported (`counts_by_class` r9 case).
- T-05-12: no finding, artifact, or provenance key trips the secret filter (`TestProjectNode_FindingMeta`, zero secret warnings).
- T-05-13: counts are added after `result` and never feed `healthScore` or `recommendation` (`TestIndexHealth_FindingsDoNotScore`).

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 05-05 and 05-08 (placeholders and external findings) will show up in `index_health` `EXTERNAL_UNRESOLVED` with no further change, as long as they set `prov_evidence_class` and a scanned `RepoPrefix`.
- The placeholder shape in `TestProjectNode_FindingMeta` is the one listed for 05-05. If 05-05 adds Meta keys, add them to that node.
- Phase 6 Cypher must match the `:Finding` label.

## Self-Check: PASSED

- FOUND: internal/mcp/index_health_findings_test.go, internal/mcp/tools_enhancements.go, internal/mcp/resources.go, internal/neo4jprojection/properties_test.go
- FOUND commits: fdd4c2b9, 13116579, 81c61f18, 345240d3
- Both task verify commands, every acceptance criterion, and the plan-level checks passed

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
