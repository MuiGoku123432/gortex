---
phase: 04-stable-incremental-and-cross-repository-lifecycle
plan: 09
subsystem: indexer
tags: [cobol, incremental, staleness, blake3, repo-index-state, provenance]

requires:
  - phase: 04-07
    provides: live index.cobol config (Indexer.cobolConfig, SetCobolConfig, MultiIndexer.RefreshRepoConfigs)
  - phase: 04-11
    provides: extractor version gortex-cobol-grammar/5; the "cobol" extractorVersions entry was left for this plan to replace
  - phase: 04-08
    provides: COBOL reconcile on every incremental tail, so restaged COBOL files converge to a clean build
  - phase: 04-15
    provides: warm-restart reconcile and graphdump-volatile/2
provides:
  - CobolGrammarExtractor.MaterialIdentity (D-07), the stamped extractor identity
  - cobolMaterialKey "_cobol_material", cobolMaterialFingerprint, Indexer.cobolMaterial, Indexer.repoExtractorVersionsSnapshot
  - staleLangsBetween marks cobol stale when the material fingerprint is missing or different
  - "cobol" removed from extractorVersions
  - TestCobolLifecycle_ConfigChange and TestCobolLifecycle_UnrelatedConfig
affects: [04-16, cobol-lifecycle, freshness]

actuals:
  tokens: 4300
  tasks: 1
  commits: 2
plan_head_before: fa3e3340a308cfd526cf1b071c8e8125c50400b8

tech-stack:
  added: []
  patterns:
    - "Reserved non-language keys in the RepoIndexState ExtractorVersions JSON (_post_extraction_policy is an epoch, _cobol_material is an identity compared by inequality)"
    - "Optional extractor capability asserted ad hoc through an anonymous interface (interface{ MaterialIdentity() string })"

key-files:
  created:
    - internal/indexer/cobol_material.go
    - internal/indexer/cobol_lifecycle_material_test.go
  modified:
    - internal/parser/languages/cobol_grammar.go
    - internal/indexer/extractor_version.go
    - internal/indexer/index_state.go

key-decisions:
  - "The COBOL material fingerprint is the first 6 bytes (big-endian) of BLAKE3(MaterialIdentity + newline + JSON of index.cobol), stored as the reserved integer key _cobol_material in the existing ExtractorVersions JSON. No schema migration and no retrieval manifest input."
  - "Only the repository's own staleness check (extractorVersionStaleLangSet) compares against the per-repository snapshot. ExtractorVersionStaleLangs, the MCP freshness rider, keeps the global snapshot, so material staleness is not shown there."
  - "MaterialIdentity runs the extractor's lazy init. On failure it returns unavailable:preprocessor-not-approved or unavailable:parser, so a broken toolchain still gets a distinct fingerprint."
  - "The config-change tests normalize only the incrementally added .gortex.yaml file-node row (Meta {} vs null). The cause is a gap in graphdump that predates this plan, so it is deferred and the measuring tool is unchanged."

patterns-established:
  - "Material invalidation: an identity-valued reserved key whose inequality marks one language stale on the next full-root non-Merkle reconcile, re-stamped by the clean pass's reconcileRepoIndexState"

requirements-completed: [ID-08, ID-09]

coverage:
  - id: D1
    description: "An index.cobol db2_default_schema edit with unchanged COBOL bytes restages every COBOL file and matches the clean build, including the aliases edge"
    requirement: ID-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_material_test.go#TestCobolLifecycle_ConfigChange/default_schema"
        status: pass
    human_judgment: false
  - id: D2
    description: "An index.cobol resolve_from edit changes cross-repository bindings exactly as a clean build with that config does"
    requirement: ID-09
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_material_test.go#TestCobolLifecycle_ConfigChange/allowlist"
        status: pass
    human_judgment: false
  - id: D3
    description: "A clean pass re-stamps _cobol_material to the current fingerprint, and a second reindex restages nothing"
    requirement: ID-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_material_test.go#TestCobolLifecycle_ConfigChange/restamped"
        status: pass
    human_judgment: false
  - id: D4
    description: "Editing an unrelated index key restages no COBOL file and leaves the stored fingerprint unchanged"
    requirement: ID-08
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_lifecycle_material_test.go#TestCobolLifecycle_UnrelatedConfig"
        status: pass
    human_judgment: false
  - id: D5
    description: "Existing extractor-version, stale-language, and MCP freshness behaviour is unchanged after the cobol entry is removed"
    verification:
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestCobolLifecycle_|ExtractorVersion|StaleLangs'"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/mcp/ -run 'Freshness'"
        status: pass
      - kind: integration
        ref: "GOWORK=off go test -count=1 -timeout 40m ./internal/indexer/"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 09: COBOL Material Fingerprint Summary

**Editing the index.cobol config now restages COBOL files even when their bytes are unchanged. A 48-bit BLAKE3 fingerprint of the extractor's own stamped identity plus the repository's index.cobol block is stored as `_cobol_material` in RepoIndexState. It differs from the current value after a config edit, so the next full-root reconcile marks cobol stale, and the clean pass re-stamps it.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-10-13T04:47:03Z
- **Completed:** 2026-10-13T05:02:34Z
- **Tasks:** 1 (tracer, TDD)
- **Files modified:** 5

## Accomplishments

- `CobolGrammarExtractor.MaterialIdentity()` returns the values Extract stamps (approved grammar ID, parser module, preprocessor module, `gortex-cobol-grammar/5`, `cobol-name-norm/1`). The stamp and the invalidation trigger come from one place, so they cannot drift (D-07).
- `internal/indexer/cobol_material.go` holds `cobolMaterialFingerprint`, `cobolMaterial`, and `repoExtractorVersionsSnapshot`. `persistRepoIndexState` now persists the per-repository snapshot (D-06).
- `staleLangsBetween` marks cobol stale when the stored fingerprint is missing or different (it compares by inequality). It never returns `_cobol_material` as a language and leaves the key out of the policy-epoch fan-out. A shared `decodeExtractorVersions` step decodes the stored JSON for both callers.
- The hand-bumped `"cobol": 2` entry is gone from `extractorVersions`. The `.cob/.cbl/.cpy` extension mapping that carries the global policy salt stays.
- Config-change harness: editing `db2_default_schema` or `resolve_from` restages every COBOL file and matches the clean build. Editing an unrelated key restages only `.gortex.yaml`.

## Task Commits

1. **Task 1 (tracer) RED: failing COBOL material config-change lifecycle tests** - `37847811` (test)
2. **Task 1 (tracer) GREEN: restage COBOL when its material fingerprint changes** - `49420839` (feat)

No REFACTOR commit was needed.

## TDD Gate Compliance

- RED `37847811`: the target tests failed on their planned assertions. `"1" is not greater than or equal to "3"` (only `.gortex.yaml` was restaged). The dump digest differed on the missing `aliases` edge and on the unbound `XCPY`/`XPGM` edges. `COBOL material fingerprint available` and `stored _cobol_material after track` were false.
- GREEN `49420839`: both tests pass under `-race`, and so do the full indexer package, the MCP freshness tests, the COBOL parser tests, and the `cmd/gortex` COBOL tests.

## Files Created/Modified

- `internal/parser/languages/cobol_grammar.go`: adds `MaterialIdentity` (not added to the non-darwin/linux stub).
- `internal/indexer/cobol_material.go`: reserved key, fingerprint, per-repository snapshot.
- `internal/indexer/extractor_version.go`: removes the cobol entry; adds material staleness and the shared decode.
- `internal/indexer/index_state.go`: persists `repoExtractorVersionsSnapshot()`.
- `internal/indexer/cobol_lifecycle_material_test.go`: `TestCobolLifecycle_ConfigChange` (default_schema, allowlist, restamped) and `TestCobolLifecycle_UnrelatedConfig`.

## Decisions Made

See key-decisions in the frontmatter. In short:
- The fingerprint is stored inside the existing ExtractorVersions JSON, so there is no new column.
- The MCP freshness rider is unchanged.
- An init failure still yields a distinct identity.
- On 32-bit `int` platforms the persisted value would be truncated. Stored and current values are truncated the same way, so change detection still works.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] RED stub so the RED commit compiles**
- **Found during:** Task 1 RED
- **Issue:** The tests reference `cobolMaterialKey` and `Indexer.cobolMaterial`, which did not exist, so the RED commit would not compile.
- **Fix:** The RED commit carried a minimal `cobol_material.go` with the key constant and a `cobolMaterial` stub that returned `(0, false)`. GREEN replaced it.
- **Committed in:** `37847811`

**2. [Rule 3 - Blocking] Normalize the incrementally added `.gortex.yaml` row in the clean-equivalence check**
- **Found during:** Task 1 GREEN
- **Issue:** Every COBOL row matched the clean build. The one difference was the `.gortex.yaml` file node: `Meta {}` after the incremental pass, `null` in a clean build. The incremental patch stamps the `source_*_fingerprint` cache keys, and graphdump strips them as volatile, which leaves an empty map. This happens for any non-COBOL file added incrementally. A probe reproduced it with `conf/x.yaml` and `pkg/a.go` and no config change. It predates this plan.
- **Fix:** The test helper `requireConfigCleanEquivalent` rewrites only that repository's `.gortex.yaml` row from `{}` to `null` and compares every other row exactly. The graphdump fix is logged in `deferred-items.md` and `.planning/WINDOWS.md`.
- **Files modified:** `internal/indexer/cobol_lifecycle_material_test.go`
- **Committed in:** `49420839`

---

**Total deviations:** 2 auto-fixed (2 blocking).
**Impact on plan:** No production scope change. The second fix is a narrow, documented normalization in the test, outside the COBOL rows the plan proves.

## Issues Encountered

- `gofmt -l internal/indexer/` lists `file_delta.go`. That file was already unformatted before this plan and was not touched.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 04-16 can salt `cobolMaterial()` into the Merkle leaves (the Merkle path still ignores the material key), prove the identity-change sequence, and add the HEAD re-stamp (D-17).
- Deferred: graphdump renders an emptied Meta as `{}` (fix with `graphdump-volatile/3`).

---
*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Completed: 2026-10-13*

## Self-Check: PASSED
