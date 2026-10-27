---
phase: 05-explicit-unresolved-evidence
plan: 16
subsystem: cobol-findings
tags: [cobol, placeholder, missing-artifact, namespace, evict-by-id, refusal, gap-closure, d-27, wr-05]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "graph.EvictResourceNodesByIDs kind-guarded by-ID evicter (05-15); identity v2 rules (05-13); settle ledger and CobolFindingChanges drain under the topology writer (05-14); placeholders and the finding-edge sweep rule (05-05)"
provides:
  - "Placeholder ID <callerRepo>::cobol-missing::<type>/<key> with an empty FilePath; cobolPlaceholderTarget is its only builder"
  - "cobolUnresolvedFromPlaceholder cuts at the first '::' (COPY IDMS round trip)"
  - "Sweep candidates: KindResource, cobol_kind missing_artifact, non-empty RepoPrefix, ID under <RepoPrefix>::cobol-missing::; unreferenced ones removed in one sorted graph.EvictResourceNodesByIDs call; no file-path eviction; an unsupported store warns, no fallback"
  - "reconcileCobol refusal: a non-placeholder node at the computed ID keeps the edge on its unresolved target, no placeholder, no finding, one warning per pass (repo, artifact_type, refused)"
  - "cobolOwnerPrefix routes <repo>:: placeholder IDs and <repo>/ finding IDs to their owner in the ledger tracked-prefix loop and syncCobolFindingFTSByRepo"
  - "TestCobolFindings_PlaceholderNamespaceNoFileCollision, TestCobolReconcile_PlaceholderRefusedOverRealNode, TestCobolReconcile_PlaceholderSweepByID"
  - "Audited graph.RepoPrefixOfID caller table and the other ID-splitter rows"
affects: [05-17]

actuals:
  tokens: 8241
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Resolver-made nodes owned by a repository live under <RepoPrefix>:: with an empty FilePath, so no repository path can mint their ID and no file eviction reaches them"
    - "A resolver sweep removes such nodes by ID through a kind-guarded store evicter, passing only IDs it read in the same pass under its own namespace"
    - "A resolver never upserts a placeholder over an ID another node holds; it refuses and logs repository and type only"

key-files:
  created: []
  modified:
    - internal/resolver/cobol_findings.go
    - internal/resolver/cobol_references.go
    - internal/indexer/finding_ledger.go
    - internal/resolver/cobol_reconcile_test.go
    - internal/resolver/cobol_findings_test.go
    - internal/indexer/cobol_findings_lifecycle_test.go
    - "8 more test files (mechanical placeholder literal substitution only)"

key-decisions:
  - "D-27 shipped with the planned grammar <callerRepo>::cobol-missing::<type>/<key> and an empty FilePath; graph.RepoPrefixOfID is unchanged and only the three finding_ledger.go COBOL routing sites use cobolOwnerPrefix"
  - "Guards split by layer: the resolver passes only IDs it read this pass as KindResource missing_artifact nodes under <RepoPrefix>::cobol-missing::; the store applies the kind guard"
  - "Refusal keeps a set of placeholder IDs beside the slice the in-edge read needs, and adds each ID it keeps or creates, so repeated references to one missing target cost one store lookup per pass"
  - "store_vector.go:77 and :114 cannot receive a placeholder ID: no production caller; the production writer ReplaceVectorCorpus persists an explicit, node-verified repo_prefix. Not a rule-4 stop"
  - "New tests never spell a placeholder ID: resolver tests call cobolPlaceholderTarget, the real-path test finds the placeholder through the store, and the sweep test derives the former slash-form path from the helper's output"
  - "GAP-02, GAP-04, GAP-05, GAP-07 addressed but not marked complete; the phase verifier re-runs after 05-17"

patterns-established:
  - "A by-ID sweep test seeds an empty-FilePath bystander (a dependency stub) so a regression to file-path eviction of an empty path fails it"

requirements-completed: []  # Plan requirements are [GAP-02, GAP-04, GAP-05, GAP-07]; left unmarked per orchestrator instruction (phase verifier after 05-17).

coverage:
  - id: D1
    description: "Placeholder grammar <repo>::cobol-missing::<type>/<key>, empty FilePath, IDMS round trip; every existing placeholder test passes with the new IDs"
    requirement: GAP-05
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderShape"
        status: pass
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderIDMSSecondPass"
        status: pass
      - kind: unit
        ref: "GOWORK=off go test -race -count=1 ./internal/resolver/"
        status: pass
    human_judgment: false
  - id: D2
    description: "WR-05 real path: a real COBOL file at cobol-missing::program/ZQXHOST.CBL and its HOSTPGM node survive placeholder creation and sweep; incremental equals clean after each step"
    requirement: GAP-05
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_PlaceholderNamespaceNoFileCollision"
        status: pass
    human_judgment: false
  - id: D3
    description: "Sweep removes placeholders by ID only; an empty-FilePath stub, a file at the old slash-form path, and its child survive; one resource eviction, no file eviction"
    requirement: GAP-07
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderSweepByID"
        status: pass
    human_judgment: false
  - id: D4
    description: "Refusal over an occupied placeholder ID: node unchanged, edge unresolved with reason missing, no placeholder or finding, one warning with repo and type only, second pass writes nothing"
    requirement: GAP-05
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_PlaceholderRefusedOverRealNode"
        status: pass
    human_judgment: false
  - id: D5
    description: "Placeholder and finding IDs route to their owner's ledger and search index (cobolOwnerPrefix)"
    requirement: GAP-02
    verification:
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_SearchAfterIncremental"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_lifecycle_test.go#TestCobolFindings_CrossRepoDependentLedgerAndSearch"
        status: pass
      - kind: e2e
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_PlaceholderSearch"
        status: pass
      - kind: e2e
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_PlaceholderMCPGetSymbol"
        status: pass
    human_judgment: false
  - id: D6
    description: "RepoPrefixOfID caller audit and other ID-splitter dispositions"
    verification: []
    human_judgment: true
    rationale: "A code-reading argument per call site, not an executable check; the verifier should confirm the dispositions, especially the vector trace and the guard-rule fallback."
  - id: D7
    description: "WINDOW (review WR-05) paragraph in the reconcileCobol doc"
    verification: []
    human_judgment: true
    rationale: "Doc accuracy is judged by review; the acceptance greps passed but cannot judge the wording."

duration: 50min
completed: 2026-10-26
status: complete
---

# Phase 5 Plan 16: Placeholder ID Namespace and By-ID Sweep Summary

**Missing-artifact placeholders now live at `<repo>::cobol-missing::<type>/<key>` with no file path. No repository file can share their ID. The sweep removes them by ID through the kind-guarded resource evicter. A node already holding a computed placeholder ID is never overwritten. Review WR-05 is closed on the real path.**

## Performance

- **Duration:** ~50 min
- **Started:** 2026-10-26T14:06:15Z
- **Completed:** 2026-10-26T14:55Z
- **Tasks:** 2 of 2
- **Files modified:** 14 source and test files (no new files), plus planning artifacts

## Accomplishments

- **Grammar:** `cobolPlaceholderTarget` joins the caller repository and the namespace with `::`. New placeholders have an empty FilePath. `cobolUnresolvedFromPlaceholder` cuts at the first `::`.
- **Sweep:** candidates must be under their own `<RepoPrefix>::cobol-missing::` prefix. Unreferenced ones go in one sorted `graph.EvictResourceNodesByIDs` call. Evicted IDs enter the change record only when the store supports the call; otherwise the sweep logs one warning, with no fallback. `EvictFile` is gone from `cobol_findings.go`.
- **Refusal:** `reconcileCobol` refuses a placeholder ID held by a non-placeholder node. The `WINDOW (review WR-05)` doc paragraph covers the residual window.
- **Routing:** `cobolOwnerPrefix` routes the three `finding_ledger.go` COBOL sites, so placeholders stay searchable and are ledgered under their owner.
- **Tests:** one real-path regression and two unit tests, each shown failing first. 13 test files took the mechanical `::` substitution.

## Task Commits

1. **Task 1 (tracer): placeholder namespace, by-ID sweep, routing** - `c58e3e4e` (test, RED), `4e8d4a9e` (fix, GREEN)
2. **Task 2: refusal and by-ID sweep proof** - `5a4002a6` (test, RED), `ab03a310` (fix, GREEN plus the sweep test)

**Plan metadata:** the docs(05-16) commits that add this file and update STATE/ROADMAP.

## RED and Mutation Runs

| Test | Run | Failure |
|------|-----|---------|
| `TestCobolFindings_PlaceholderNamespaceNoFileCollision` | RED on the pre-change code | After track the placeholder took `fcoll/cobol-missing::program/ZQXHOST.CBL`: the file node's kind became `resource`, its FilePath was the ID, and it was audited. After the CALL was removed the real file node was gone (`require.NotNil` failed in the second `requireHost`). |
| same | Mutation: slash restored in `cobolPlaceholderTarget` on the new code | Placeholder at the file's ID; file node `resource` with empty FilePath; never swept (outside the namespace check); incremental-vs-clean dump hash mismatch |
| `TestCobolReconcile_PlaceholderRefusedOverRealNode` | RED on Task 1 code (memory and SQLite) | The seeded KindFunction became `resource` with FilePath "", RepoPrefix r1, D-11 Meta; the CALL moved onto it; one placeholder and one finding existed; no warning |
| `TestCobolReconcile_PlaceholderSweepByID` | Mutation: sweep calls `EvictFile(n.FilePath)` again | Both stores: `dep::zqx::Helper` (empty FilePath) evicted with the placeholder; SQLite: 0 resource evictions, 1 `EvictFile` |

Each mutation was reverted by an edit, and `git diff -- internal cmd` was empty afterward (MUTATION markers: 0).

## RepoPrefixOfID Caller Audit

Fresh grep at 4e8d4a9e: 24 non-test call sites, plus the definition at `graph/repo_prefix.go:23` and the new fallback inside `cobolOwnerPrefix`. Categories: (a) keyed on an edge's source, which can never be a placeholder because a placeholder has no out-edges; (b) RepoPrefix-first; (c) only ever passed file IDs; (d) switched to `cobolOwnerPrefix`; (e) workspace clamp keyed on the ID alone; (f) persisted per-repository key.

| # | Call site | Can it receive a placeholder ID? | Disposition |
|---|-----------|----------------------------------|-------------|
| 1 | graph/override_dispatch_projection.go:74 | No: `edge.From` of unresolved calls edges | (a) |
| 2 | graph/placeholder_sources.go:63 | No: `r.Edge.From` of a reindexed edge | (a) |
| 3 | analyzer/synthesizers.go:98 | No: `e.From` of a synthesized edge | (a) |
| 4 | indexer/repository_untrack.go:292 | No: `e.From` of an in-edge of a COBOL program or copybook | (a) |
| 5 | resolver/terminal.go:199 | No: `e.From` | (a) |
| 6 | resolver/framework_synth.go:1686 | No: `edge.From` | (a) |
| 7 | resolver/framework_synth.go:1688 | No: `edge.FilePath` is the source file (a finding's references edge carries the finding's file) | (a) |
| 8 | resolver/repo_language_lookup.go:119 | No: `edge.From` fallback | (a) |
| 9 | resolver/resolver.go:1123 | No: guard job `edge.From` | (a) |
| 10 | resolver/resolver.go:1881 | No: `e.From` | (a) |
| 11 | resolver/resolver.go:1911 | No: all nine `edgeFromInScope` callers pass `e.From` | (a) |
| 12 | resolver/cobol_references.go:188 | No: COBOL code edge `e.From` | (a) |
| 13 | resolver/resolve_all_pending.go:102 | No: `edge.From`, overridden by the hydrated source's RepoPrefix | (a) |
| 14 | graph/stub.go:224 | Only if RepoPrefix is empty; a placeholder's is set | (b) |
| 15 | resolver/resolver.go:1935 | No: `dirIndex` holds KindFile identities, RepoPrefix-first | (b) |
| 16 | resolver/cross_pkg_guard.go:432 | No: KindFile identity IDs and import-edge sources | (c) |
| 17 | indexer/finding_ledger.go:187 | Yes: drained added and evicted IDs | (d) `cobolOwnerPrefix` |
| 18 | indexer/finding_ledger.go:272 | Yes | (d) |
| 19 | indexer/finding_ledger.go:277 | Yes | (d) |
| 20 | mcp/scope_resolve.go:466 (community members) | Yes | (e) yields `<repo>::cobol-missing::<type>`, which is in no workspace repo set, so the member drops out of a clamped view. It fails closed like `<repo>::unresolved::` IDs. Accepted |
| 21 | mcp/scope_resolve.go:475 (community files) | No in practice: an empty FilePath adds no file entry | (e) accepted |
| 22 | mcp/scope_resolve.go:561 (process steps) | Yes: as a leaf step | (e) a placeholder has no out-edges, so dropping it never re-parents a step. Fails closed. Accepted |
| 23 | graph/store_sqlite/store_vector.go:77 (`UpsertEmbedding`) | No: only store_sqlite tests call it | (f) cannot receive |
| 24 | graph/store_sqlite/store_vector.go:114 (`BulkUpsertEmbeddings`) | No: only store_sqlite tests call it | (f) cannot receive |

Vector trace for (f):
- `collectEmbedTexts` (indexer.go:5366) skips only KindFile, KindImport, content nodes, and configured SkipEmbed kinds. A placeholder can therefore be embedded.
- It is embedded only through `ReplaceVectorCorpus` (indexer/vector_plan.go:241, :419). That call gets the plan's explicit `repoPrefix`, whose nodes come from `GetRepoNonContentNodes(repoPrefix)`, and `validateVectorCorpusOwnershipTx` checks each node's stored `repo_prefix`.
- So no vector row's repository is derived from a placeholder ID.

### Other ID splitters

| Splitter (call sites) | Result for a placeholder ID | Disposition |
|-----------------------|-----------------------------|-------------|
| mcp/tools_analysis.go:265 `repoPrefixOf` | "" (the text before the first `::` has no slash) | - |
| `majorityRepoPrefix` (tools_analysis.go:281, used at :162) | the placeholder counts toward no repository | fails closed, display only |
| `uniqueRepoPrefixesFromSteps` (tools_analysis.go:303, used at :251) | the placeholder contributes no repository | fails closed, display only |
| `repoScopeResolver.prefixOf` (guard_rules.go:74, via :98, :117, :139 and tools_inspections.go:443) | "" becomes the sole tracked repository (its owner in single-repo mode); with several repositories, `guardRulesFor("")` returns the global and server rules | accepted: read-only advisory evaluation over the session's scoped nodes; nothing persisted; same as other `<repo>::` and stub IDs |
| `graph.StubRepoPrefix` (terminal.go:204, resolver.go:1890, external_call_attribution.go:117) | "" (`cobol-missing` is not in `stubKinds`), the same as the old slash form | fails closed, unchanged |
| `graph.UnresolvedRepoPrefix` (terminal.go:202, resolver.go:1888) | "" (no `::unresolved::` marker), the same as before | fails closed, unchanged |
| graph/store_sqlite/prefix_diagnostics.go:62 SQL predicate | `file_path = ''` and `instr(id, repo_prefix \|\| '::') = 1`, so exempt | exempt as an owned node (SQL twin of `IsAuditableRepoSourceNode`); `TestCobolReconcile_PlaceholderPrefixDiagnostics` asserts owned = 8, unowned 0, misprefixed 0 |

Other consumers checked:
- **Untrack:** purges by the `repo_prefix` column on SQLite and by the `byRepo` bucket in memory, so placeholders still go.
- **Search:** admission (`shouldIndexForSearch`) does not read FilePath; only the path tokens disappear.
- **Neo4j:** the `GortexUnresolved` label comes from `Meta["synthetic"]`.

## Files Created/Modified

- `internal/resolver/cobol_findings.go`: grammar, IDMS cut, empty FilePath, namespace-checked candidates, by-ID sweep, rewritten constants doc and PURPOSE/RATIONALE.
- `internal/resolver/cobol_references.go`: refusal, warning, `WINDOW (review WR-05)` paragraph, `zap` import.
- `internal/indexer/finding_ledger.go`: `cobolOwnerPrefix` and three call sites; docs.
- `internal/resolver/cobol_reconcile_test.go`: `resourceEvicts` counter and `EvictResourceNodesByIDs` on `cobolWriteCounter`.
- `internal/resolver/cobol_findings_test.go`:
  - two new tests;
  - PlaceholderShape expects an empty FilePath;
  - PrefixDiagnostics pins owned = 8 and per-placeholder exemption;
  - two idempotency tests assert no resource eviction.
- `internal/indexer/cobol_findings_lifecycle_test.go`: the WR-05 real-path test and its corpus.
- `internal/neo4jprojection/properties_test.go`: placeholder fixture without FilePath.
- Literal substitution only: `cobol_references_test.go`, `cobol_findings_equivalence_test.go`, `cobol_lifecycle_crossrepo_test.go`, `cobol_lifecycle_incremental_test.go`, `cobol_lifecycle_material_test.go`, `cmd/gortex/cobol_breadth_test.go`, `cmd/gortex/cobol_findings_test.go`.
- `cobol_findings_identity_test.go` was planned but holds no slash-form literal, so it is unchanged.

## Decisions Made

See key-decisions. Also:
- `PrefixDiagnostics` now pins the exact owned count, 8: two callers plus six findings. Findings are audited, so the old `>= 8` would have passed vacuously once placeholders left the audit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Test integrity] Idempotency tests also assert no resource eviction**
- **Found during:** Task 1.
- **Issue:** the sweep no longer calls `EvictFile`, so "no EvictFile" in `PlaceholderIdempotent` and `ExternalIdempotent` would hold vacuously.
- **Fix:** `cobolWriteCounter.writes()` includes `resourceEvicts`, and both tests assert it is zero.
- **Committed in:** `4e8d4a9e`.

**2. [Orchestrator rule] No hand-spelled placeholder IDs in the new tests**
- **Issue:** the plan said to build the sweep test's old slash-form path by joining strings, and to locate the placeholder at a literal ID.
- **Fix:** the sweep test derives the path from `cobolPlaceholderTarget` output. The real-path test finds the placeholder by `artifact_key` in the store and asserts `fcoll::` ownership, an empty FilePath, audit exemption, and the CALL in-edge.
- **Committed in:** `c58e3e4e`, `ab03a310`.

**3. [Rule 2 - Doc accuracy] The WINDOW paragraph also names the stale code edge**
- **Issue:** in the occupied-ID window, the code edge already on that ID is not revisited either: it is not unresolved, not on a candidate, and not on a placeholder.
- **Fix:** the paragraph says that finding and that edge stay until the referencing file is re-extracted or a clean re-index.
- **Committed in:** `ab03a310`.

**Total deviations:** 3 (test integrity, the single-helper rule, doc accuracy). **Impact:** none on production scope. No unplanned files.

## TDD Gate Compliance

- **Task 1:** RED `c58e3e4e` (the real-path test fails on the pre-change code), GREEN `4e8d4a9e`, plus the plan's mutation run.
- **Task 2:** RED `5a4002a6` (refusal fails on Task 1 code), GREEN `ab03a310`. The sweep test passes on Task 1 code; its mutation run is recorded above.

## Tracer Gate

- Auto mode is off (`_auto_chain_active` and `auto_advance` are false). `human_verify_mode` is end-of-phase, mode is yolo, and the plan is autonomous, so the run did not halt (05-13 and 05-15 precedent).
- The Task 1 `<verify>` re-ran verbatim end to end after the mutation revert and passed before Task 2 started.

## Issues Encountered

- **Gortex edit timeout:** one `gortex_edit` batch hit its 59 s deadline but completed in the background. The diff was inspected before continuing, and every change had landed exactly once.
- **Disk:** free space fell from 12 GiB to 6.5 GiB over the `-race` runs. Nothing was deleted, and no ENOSPC occurred.
- **Residual repository-prefix edge cases (user-controlled, accepted by the plan):**
  - A repository prefix that spells another repository's placeholder namespace, such as `r1::cobol-missing::program`, hits the refusal path.
  - A repository named like a global namespace (`unresolved`, `dep`, `module`, `stdlib`, `builtin`, `external`, `string`) makes every `<repo>::` ID read as global. A repository named `unresolved` would give placeholders that `graph.IsUnresolvedTarget` accepts.
  - 05-17 may document both cases.
- **Deferred:** shared DB2 and IDMS resources in `reconcileCobolResources` keep the WR-05 shape (ID = FilePath, `EvictFile`). Logged in `deferred-items.md`, not fixed: out of D-27 scope.

## Verification

| Command | Result |
|---------|--------|
| Task 1 `<verify>` (verbatim, after the mutation revert) | exit 0 (resolver 23 s, indexer 27 s, cmd/gortex 14 s, neo4j, graph, gofmt) |
| Task 2 `<verify>` (verbatim) | exit 0 (resolver `Cobol` 24 s; six named tests listed) |
| `GOWORK=off go test -race -count=1 ./internal/resolver/` | ok (54 s) |
| `GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer/ -run '^TestCobol'` | ok (272 s), no timeout |
| `GOWORK=off go test -race -count=1 -timeout 30m ./cmd/gortex/ -run '^TestCobol'` | ok (52 s) |
| `GOWORK=off go test -race -count=1 ./internal/neo4jprojection/ ./internal/graph/ ./internal/mcp/ -run 'ProjectNode\|PrefixDiagnostics\|Finding\|IndexHealth'` | ok |
| `GOWORK=off go build ./...` | ok |
| `git diff --name-only 437c13de..HEAD` | 14 of the 15 planned files; nothing unplanned |
| Acceptance greps | prod `/cobol-missing::` 0; `.EvictFile(` 0; `EvictResourceNodesByIDs` 3; `func cobolOwnerPrefix(` 1; `cobolOwnerPrefix(` 4; non-comment `RepoPrefixOfID(` in finding_ledger.go 1; `WINDOW (review WR-05)` 1; `clean re-index` 1 |

## Threat Flags

None. The only new output is the refusal warning, which names the repository and artifact type only (T-05-72, accepted).

## Known Stubs

None.

## User Setup Required

None. Developer stores indexed before this plan keep old-grammar placeholders until a clean re-index; there is no legacy migration, per the plan.

## Next Phase Readiness

- 05-17 should document D-27: the grammar, the empty FilePath, the by-ID sweep, the refusal, and the WINDOW (stale finding and edge until the file is re-extracted or a clean re-index).
- It should also give the rebuild note for pre-gap-closure stores, and may add the repository-name edge cases above.
- GAP-02, GAP-04, GAP-05, and GAP-07 stay pending for the phase verifier.

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-26*

## Self-Check: PASSED

- FOUND: internal/resolver/cobol_findings.go, internal/resolver/cobol_references.go, internal/indexer/finding_ledger.go, internal/indexer/cobol_findings_lifecycle_test.go, internal/resolver/cobol_findings_test.go, internal/resolver/cobol_reconcile_test.go
- FOUND commits: c58e3e4e, 4e8d4a9e, 5a4002a6, ab03a310
- Both task verify commands and every plan-level verification command passed; the source tree is clean after the mutation runs.
