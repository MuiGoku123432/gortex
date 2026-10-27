---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 16
subsystem: indexer
tags: [cobol, merkle, incremental, provenance, vcs, graphdump, staleness]

requires:
  - phase: 04-09
    provides: cobolMaterialFingerprint, Indexer.cobolMaterial, _cobol_material staleness, CobolGrammarExtractor.MaterialIdentity
  - phase: 04-01
    provides: COBOL lifecycle harness (newCobolLifecycleEnv, graphdump)
provides:
  - Indexer.merkleSaltFor, which salts COBOL Merkle leaves with |cobol-material@{fingerprint} (D-06)
  - Indexer.restampSourceRevisions and its HEAD-move trigger in reconcileRepoIndexState (D-17)
  - applySourceRevision, the single set-one-delete-the-other revision stamp rule
  - graphdump-volatile/3, in which an emptied Meta renders as null
  - TestCobolLifecycle_VersionChange (mtime and Merkle) and TestCobolLifecycle_HeadMove
affects: [cobol-lifecycle, freshness, provenance, phase-04-verification]

actuals:
  tokens: 7400
  tasks: 2
  commits: 5
plan_head_before: 094fb34b3d2db1d3e9a8d7167f2e7e55f7d8b5d0

tech-stack:
  added: []
  patterns:
    - "Per-repository Merkle salt: an Indexer method wraps the package salt and adds repository material for one language"
    - "Metadata-only provenance refresh: re-decide prov_vcs_commit from transformed bytes and write only the rows that changed"

key-files:
  created:
    - internal/indexer/cobol_material_test.go
    - internal/parser/languages/cobol_grammar_material_test.go
  modified:
    - internal/indexer/cobol_material.go
    - internal/indexer/merkle_reindex.go
    - internal/indexer/source_revision.go
    - internal/indexer/index_state.go
    - internal/indexer/cobol_lifecycle_material_test.go
    - internal/testutil/graphdump/graphdump.go
    - internal/testutil/graphdump/graphdump_test.go

key-decisions:
  - "In Merkle mode the persisted leaf salt is the record of the COBOL material fingerprint. The RepoIndexState _cobol_material is read only on the non-Merkle path, the same as the extractor-version keys, so a Merkle restage does not re-stamp the row. If Merkle is later turned off, the next pass does one extra, conservative COBOL restage."
  - "The HEAD re-stamp groups rows by prov_source_path, the document whose bytes decided the stamp, not by FilePath. Shared DB2/IDMS/CICS resource nodes carry no prov_revision_content_id, so they are untouched."
  - "The re-stamp fires in reconcileRepoIndexState only when a previous row exists and the current HEAD differs from its IndexedSHA, and it runs before the row is re-persisted at the new HEAD."
  - "graphdump renders a Meta that is empty after stripping as null (graphdump-volatile/3). The 04-09 row rewrite is removed, so the config-change tests compare every row exactly."

patterns-established:
  - "Each change-detection mode re-stamps its own record: the RepoIndexState row (mtime) or the Merkle leaf salt (Merkle)"

requirements-completed: [ID-08, ID-09]

coverage:
  - id: D1
    description: "COBOL Merkle leaf salts carry |cobol-material@{fingerprint}, change with index.cobol, and leave other languages' salts unchanged"
    requirement: ID-08
    verification:
      - kind: unit
        ref: "internal/indexer/cobol_material_test.go#TestCobolMaterialFingerprint_MerkleSalt"
        status: pass
    human_judgment: false
  - id: D2
    description: "The fingerprint is stable and depends only on identity and the index.cobol block; staleLangsBetween reports cobol on a missing or different key and never reports the reserved key"
    requirement: ID-08
    verification:
      - kind: unit
        ref: "internal/indexer/cobol_material_test.go#TestCobolMaterialFingerprint_Stable"
        status: pass
      - kind: unit
        ref: "internal/indexer/cobol_material_test.go#TestCobolMaterialFingerprint_ConfigOnly"
        status: pass
      - kind: unit
        ref: "internal/indexer/cobol_material_test.go#TestCobolMaterialFingerprint_StaleLangs"
        status: pass
    human_judgment: false
  - id: D3
    description: "MaterialIdentity equals the provenance values the extractor stamps on a program (D-07)"
    requirement: ID-08
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_material_test.go#TestCobolGrammar_MaterialIdentity"
        status: pass
    human_judgment: false
  - id: D4
    description: "An extractor or parser identity change restages exactly the COBOL files, restores prov_extractor_version, re-stamps the mode's record, and equals the clean build in mtime and Merkle modes; a second pass restages nothing"
    requirement: ID-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_material_test.go#TestCobolLifecycle_VersionChange"
        status: pass
    human_judgment: false
  - id: D5
    description: "A HEAD move with unchanged bytes re-stamps prov_vcs_commit / prov_vcs_commit_absence without re-extraction and equals a clean build at the new HEAD; dirty files stay absent"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_material_test.go#TestCobolLifecycle_HeadMove"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestCobolLifecycle_|^TestSourceRevision|GitWatcher|Poller'"
        status: pass
    human_judgment: false
  - id: D6
    description: "graphdump renders an emptied Meta like a nil Meta (graphdump-volatile/3), and the config-change tests match the clean build on every row"
    verification:
      - kind: unit
        ref: "internal/testutil/graphdump/graphdump_test.go#TestGraphdump_EmptiedMetaIsNull"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_material_test.go#TestCobolLifecycle_ConfigChange"
        status: pass
    human_judgment: false
  - id: D7
    description: "No regression across the indexer package, COBOL grammar tests, MCP freshness, and the build"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -count=1 -timeout 40m ./internal/indexer/"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/mcp/ -run Freshness"
        status: pass
      - kind: other
        ref: "GOWORK=off go build ./..."
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 16: COBOL Merkle Material Salt and HEAD Re-stamp Summary

**COBOL observations no longer go stale while bytes stay the same. In Merkle mode, COBOL leaf salts now include the 04-09 material fingerprint, so an index.cobol edit or an extractor or parser identity change restages exactly the COBOL files. A HEAD move re-stamps `prov_vcs_commit` from the same transformed bytes indexing uses, without re-extraction. In both cases the result equals a clean build.**

## Performance

- **Duration:** about 20 min
- **Started:** 2026-10-13T05:04:59Z
- **Completed:** 2026-10-13T05:24:29Z (the full indexer regression run ended about 6 minutes later)
- **Tasks:** 2 (TDD), plus the orchestrator-added graphdump fix
- **Files modified:** 9 source and test files, plus 2 planning files

## Accomplishments

- `Indexer.merkleSaltFor` adds `|cobol-material@{fingerprint}` to `.cob/.cbl/.cpy` leaf salts. Both `merkle.Build` and `merkle.BuildWithKnownFiles` call sites use it (D-06).
- The version-change sequence plays the store an older binary left behind and converges to the clean build in both modes (D-07). The stale node is built at runtime from the current stamped version, so no old mapping literal lands in source.
- `restampSourceRevisions` re-decides the revision stamp per `prov_source_path` document. It skips files that are stale in the mtime ledger and writes only rows whose stamp changed. `reconcileRepoIndexState` calls it when HEAD differs from the stored `IndexedSHA`. That function is reached by the git watcher, the poller, and the version-restage tail (D-17).
- `applySourceRevision` is now the single rule that sets one stamp and deletes the other. `stampSourceRevision` and the re-stamp both use it.
- graphdump-volatile/3: an emptied Meta renders as `null`. The 04-09 `.gortex.yaml` row rewrite is removed.

## Task Commits

0. **Orchestrator scope: graphdump emptied-Meta fix:** `803dfd78` (fix)
1. **Task 1 RED: Merkle salt and version-change tests:** `ae3ff540` (test)
2. **Task 1 GREEN: salt COBOL Merkle leaves with the material fingerprint:** `79bd16e7` (feat)
3. **Task 2 RED: HEAD-move lifecycle tests:** `d53a5b6c` (test)
4. **Task 2 GREEN: re-stamp the VCS commit on HEAD move:** `8a217d60` (feat)

No REFACTOR commits were needed.

## TDD Gate Compliance

- Task 1 RED `ae3ff540`: `TestCobolMaterialFingerprint_MerkleSalt` failed on its planned assertions. The salt `_post_extraction_policy@2` lacked the fingerprint suffix and did not change with `SetCobolConfig`. `TestCobolLifecycle_VersionChange/merkle` failed with `"_post_extraction_policy@2" does not contain "|cobol-material@…"`. Five targets passed already in RED because they lock behaviour 04-09 shipped: Stable, ConfigOnly, StaleLangs, MaterialIdentity, and VersionChange/mtime.
- Task 1 GREEN `79bd16e7`: every Task 1 target passes under `-race`, and so does every existing `Merkle` indexer test.
- Task 2 RED `d53a5b6c`: all four HeadMove subtests failed on assertions. MAINPGM's `prov_vcs_commit` was the old commit, or it still carried `working_tree_differs_from_head`. The dump digest differed from the clean build at the new HEAD. `no_reextraction` re-stamped 0 rows.
- Task 2 GREEN `8a217d60`: all four subtests pass under `-race`, and so do every `TestCobolLifecycle_`, `TestSourceRevision`, `GitWatcher`, and `Poller` test.

## Files Created/Modified

- `internal/indexer/cobol_material.go`: `Indexer.merkleSaltFor`.
- `internal/indexer/merkle_reindex.go`: both Merkle build sites pass `idx.merkleSaltFor`.
- `internal/indexer/source_revision.go`: `applySourceRevision` and `restampSourceRevisions`.
- `internal/indexer/index_state.go`: the HEAD-move trigger in `reconcileRepoIndexState`.
- `internal/indexer/cobol_material_test.go`: the four `TestCobolMaterialFingerprint_*` tests.
- `internal/parser/languages/cobol_grammar_material_test.go`: `TestCobolGrammar_MaterialIdentity`.
- `internal/indexer/cobol_lifecycle_material_test.go`: `TestCobolLifecycle_VersionChange`, `TestCobolLifecycle_HeadMove`, and the git-fixture helpers. The workaround helper is removed.
- `internal/testutil/graphdump/graphdump.go` and `graphdump_test.go`: volatile/3 and `TestGraphdump_EmptiedMetaIsNull`.

## Decisions Made

See key-decisions in the frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Orchestrator-added scope, D-01/D-02] graphdump renders an emptied Meta as null**
- **Found during:** 04-09, assigned to this plan by the orchestrator
- **Issue:** A node whose Meta was empty after volatile stripping dumped as `{}`, while a clean build's nil Meta dumped as `null`. Any non-COBOL file added on the incremental path therefore never byte-matched the clean build.
- **Fix:** `canonicalMeta` returns `null` for an empty map. The version is bumped to `graphdump-volatile/3`. `TestGraphdump_EmptiedMetaIsNull` was added and the version expectation updated. The `.gortex.yaml` row rewrite in `requireConfigCleanEquivalent` was removed. With the rewrite gone, that helper was identical to `requireCleanEquivalent`, so it was deleted and its three callers use `requireCleanEquivalent`. The deferred item is marked resolved and WINDOWS entry 7 is fixed.
- **Files modified:** `internal/testutil/graphdump/graphdump.go`, `internal/testutil/graphdump/graphdump_test.go`, `internal/indexer/cobol_lifecycle_material_test.go`, `deferred-items.md`, `.planning/WINDOWS.md`
- **Verification:** the RED run failed on the version string and on the `{}` row. After the fix, graphdump and every `TestCobolLifecycle_` test pass under `-race`.
- **Committed in:** `803dfd78`

**2. [Rule 3 - Blocking] RED stub so the Task 1 RED commit compiles**
- **Issue:** The tests call `idx.merkleSaltFor`, which did not exist.
- **Fix:** The RED commit carried a stub that returned the package salt unchanged. GREEN replaced it.
- **Committed in:** `ae3ff540`

**3. [Rule 1 - Test correctness] Merkle mode re-stamps the leaf salt, not the RepoIndexState row**
- **Found during:** Task 1 GREEN
- **Issue:** In Merkle mode the version sequence restaged and converged, but the stored `_cobol_material` row stayed at the rewritten value. The incremental path re-stamps the row only for a non-Merkle version restage (`extractorStaleLangs`), and Merkle mode never reads the row.
- **Fix:** The test now asserts each mode's own record: the row in mtime mode, the leaf salt in Merkle mode. In both modes it also asserts that a second pass restages nothing. Production code is unchanged. See key-decisions.
- **Files modified:** `internal/indexer/cobol_lifecycle_material_test.go`
- **Committed in:** `79bd16e7`

**4. Merkle mode is enabled in the test through GORTEX_MERKLE=1**
- `GORTEX_MERKLE` is the documented override of `index.merkle` in `merkleEnabled`. The test asserts `idx.merkleEnabled()` after track, so it never runs in the wrong mode silently.

---

**Total deviations:** 1 orchestrator-added scope item, 1 blocking stub, 1 test correctness fix, and 1 test mechanism note.
**Impact on plan:** No production scope beyond the plan and the assigned graphdump fix.

## Issues Encountered

- `restampSourceRevisions` runs `classifySourceRevision` once per COBOL document after each HEAD move. That is up to 3 git calls per file, through the shared gitcmd limiter. The cost per file matches a clean stamp. A very large estate pays it once per commit observed, and it was not benchmarked.
- `internal/indexer/source_revision.go` is now about 370 lines, above the 300-line module target. The re-stamp shares the stamp rule and the classifier with the code already in that file, so it was not split out.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- The COBOL invalidation work for unchanged bytes is complete. That covers config (04-09), extractor and parser identity, Merkle mode, and HEAD moves, each proven against clean builds.
- `prov_vcs_commit` stays compared in the dump and is not volatile.

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*

## Self-Check: PASSED

- All 9 key files exist on disk.
- Commits 803dfd78, ae3ff540, 79bd16e7, d53a5b6c, and 8a217d60 are present in git log.
- Both task verify commands passed. The full `./internal/indexer/` package passed (359s).
