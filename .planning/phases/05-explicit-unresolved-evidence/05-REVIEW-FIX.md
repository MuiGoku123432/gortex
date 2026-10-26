---
phase: 05-explicit-unresolved-evidence
fixed_at: 2026-10-26T23:21:28Z
review_path: .planning/phases/05-explicit-unresolved-evidence/05-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 05: Code Review Fix Report

**Fixed at:** 2026-10-26T23:21:28Z
**Source review:** `.planning/phases/05-explicit-unresolved-evidence/05-REVIEW.md` (re-review at `e4c591dc`)
**Iteration:** 1
**Scope:** `critical_warning` (CR-01, WR-01, WR-02). IN-01 to IN-06 are out of scope.

**Summary:**
- Findings in scope: 3
- Fixed: 3
- Skipped: 0
- Every fix has a regression test, plus failing-first evidence from a temporary mutation, which was reverted afterwards.
- The D-24 design held. Both STOP checks passed, so no override was needed.

| Finding | Commit | Tests added | Mutation proof |
|---|---|---|---|
| CR-01 | `1750e339` | `TestParseCobolPlaceholderID`, `TestCobolReconcile_IDMSColonRepoPrefix` (memory + SQLite), `TestCobolFindings_IDMSColonRepoPrefix` (real path) | A (resolver parser), B (indexer routing) |
| WR-01 / D-24 | `be0f4f83` | `TestCobolFindings_MasterReconcileExcludesLane` (two subtests), `TestCobolReconcile_ResolveAllDefersReconcile` (memory + SQLite) | D24 (writer section), G1 and G2 (both `ResolveAll` guards) |
| WR-02 | `fc0fe9e9` (prior session) | `TestCobolReconcile_ExternalImagelessOccurrences` (2 reasons x 2 stores) | WR02 (this session) |

## Resumed Pass

This pass resumed after the previous fixer lost its API connection.

- **WR-02** had already been committed as `fc0fe9e9`, together with its regression test. This session added the missing failing-first evidence (mutation WR02).
- **CR-01** was uncommitted in the main checkout. The tree still held the previous agent's in-flight mutation: `ParseCobolPlaceholderID` read `i := strings.Index(id, "::") // MUTATION-A`, which is first-`::` parsing.
  - I restored the `LastIndex` line before doing anything else.
  - I reviewed the diff against the CR-01 spec, re-ran both mutation proofs myself, and then committed.
- **WR-01 / D-24** and this report are new work.

## Fixed Issues

### CR-01: Placeholder IDs are split at the first `::`

**Status:** fixed, test-verified (regressions plus mutation runs A and B)
**Commit:** `1750e339` `fix(05): CR-01 parse placeholder ids at their last namespace marker`

**Files modified:**
- `internal/resolver/cobol_findings.go`
  - `:97-111` adds `ParseCobolPlaceholderID`, the one exported parser of the grammar `cobolPlaceholderTarget` (`:73-86`) builds.
  - `:117-123` makes `cobolUnresolvedFromPlaceholder` use it.
  - `:15-22` corrects the constant doc: it no longer claims that no repository file can ever hold a placeholder ID. It now points to the WINDOW.
- `internal/indexer/finding_ledger.go:296-301`: `cobolOwnerPrefix` returns the parser's owner for placeholder IDs and otherwise calls `graph.RepoPrefixOfID`. The first-`::` heuristic and the `strings` import are gone.
- `internal/resolver/cobol_findings_test.go` and `internal/indexer/cobol_findings_lifecycle_test.go`: the tests below.

**Applied fix:**
- **Locating the owner.** `ParseCobolPlaceholderID` finds `"::" + cobolMissingNamespace` by its last occurrence, and everything before it is the owner prefix. The last marker is always the real one, because the extractor percent-encodes `:` in every key (`cobolIDSegment`, `internal/parser/languages/cobol_grammar_identity.go:214`).
- **What the parser rejects:**
  - an empty owner;
  - an owner that contains `/`, which covers every ID minted from a repository path, file-owned findings included;
  - an empty key, or a key that contains `:`;
  - an artifact type other than the four placeholder types.
- **Other splitters.** The CR-01 sweep (`grep` for `cobolMissingNamespace` and `cobol-missing`) found no other site that splits placeholder IDs at the first `::`. The sweep-candidate check (`cobol_findings.go:219`) compares the ID against the node's own `RepoPrefix + "::" + namespace`, so it is prefix-safe and unchanged.

**Tests:**
- `TestParseCobolPlaceholderID` (`cobol_findings_test.go:187`):
  - placeholder IDs of all four artifact types round-trip for the prefixes `repo`, `a::b`, `x::y::z`, and `x::cobol-missing::program` (a prefix that spells the marker);
  - the IDMS ID maps back to `unresolved::cobol::copybook::idms/RECORD/EMP-REC`, and a program ID maps to "";
  - for each prefix, file-owned finding IDs (with and without `#2`) and file paths that spell the namespace do not parse;
  - IDs outside the grammar do not parse.
- `TestCobolReconcile_IDMSColonRepoPrefix` (`:235`), on memory and SQLite, with prefix `a::b` and one never-obtained COPY IDMS member. Over one creating pass and two unchanged `ReconcileCobol(nil)` passes, it asserts:
  - exactly one EXTERNAL_UNRESOLVED finding;
  - exactly one placeholder;
  - the finding's `references` edge points at that placeholder.
- `TestCobolFindings_IDMSColonRepoPrefix` (`cobol_findings_lifecycle_test.go:945`), the real path: MultiIndexer + SQLite, cold `TrackRepoCtx` of a repository named `a::b`. After the track, and again after an identical re-index:
  - the COPY IDMS finding exists and keeps its ID;
  - it has exactly one `"" -> active` history row under `a::b`, and `a` has no history;
  - it is in symbol FTS (the `missing <name>` tier).

**Failing-first evidence:**
- **Mutation A:** restore first-`::` parsing in `ParseCobolPlaceholderID`.
  - `TestParseCobolPlaceholderID` FAILS: every `a::b` ID parses to `("", "", "")`.
  - `TestCobolReconcile_IDMSColonRepoPrefix` FAILS on both stores at `pass 2: the one IDMS finding`. This is the review's probe exactly: the finding exists after pass 1 and is gone after pass 2.
  - `TestCobolFindings_IDMSColonRepoPrefix` FAILS after the track: `"[]" should have 1 item(s)` for `RECORD/EMP-REC` in `a::b`.
  - The `r1` control, `TestCobolReconcile_PlaceholderIDMSSecondPass`, still passes.
- **Mutation B:** restore the old first-`::` body of `cobolOwnerPrefix`, with the resolver parser fixed.
  - `TestCobolFindings_IDMSColonRepoPrefix` FAILS at `identical re-index: a::b/cobol/main.cbl::@finding/external/cbb2d713... is in symbol FTS`.
  - Search sync routed the re-created finding to `a`, which has no Indexer.
  - This proves the real-path test also covers the indexer routing on its own.

### WR-01: D-24 timing on writer-free master paths

**Status:** fixed, test-verified (interleaving test plus mutation runs D24, G1, and G2)
**Commit:** `be0f4f83` `fix(05): WR-01 reconcile writer-free master cobol under the writer`

**Files modified:**
- `internal/resolver/resolver.go`
  - `:177-184` adds the `DeferCobolReconcile` field, documenting why deferral is safe.
  - Both `ResolveAll` reconcile sites honour it: the nothing-pending early return (`:754`) and the settle window (`:1702`).
- `internal/indexer/multi.go`
  - `:1063`: `runMasterResolveHookedContext` sets `master.DeferCobolReconcile = !topologyHeld`.
  - `:1085-1098`: with the writer held, the code is unchanged (`recordCobolSettleLedger(nil, true, master)`).
  - Without the writer, and only when `err == nil && ctx.Err() == nil`, it:
    1. takes `reach.BeginTopologyMutation`;
    2. runs `master.ReconcileCobol(cobolScope)`, the public path, over the scope the pass applied (`SetScope` stores the map verbatim, so the scope is identical);
    3. fires the test hook stage `master_reconciled`;
    4. runs `recordCobolSettleLedger(nil, true, master)`;
    5. releases the writer with `finish(false)` (deferred).
  - `:159-167` documents the new hook stage.
- `docs/languages.md:279`: see Docs below.
- `internal/indexer/cobol_findings_ledger_exclusion_test.go` and `internal/resolver/cobol_findings_test.go`: the tests below.

**STOP checks (both passed):**
1. **The master holds no lock that a lane takes after the writer when the new writer section starts.**
   - `ResolveAllContext` releases the store `ResolveMutex` with its deferred unlock (`resolver.go:653`). The reconcile was skipped, so `cobolReconcileMu` was never taken.
   - The writer-free callers hold nothing either:
     - `RunPreEnrichResolve` runs after `BeginParallelBatch` has released the batch gate (its unlock is deferred).
     - `FinishTailResult` calls `resolveDeferredMutationsMode` after `finishOwnedContractTails` has released the per-repo lanes.
   - Inside the section the order is: writer, then `ResolveMutex` (`ReconcileCobol`, `cobol_references.go:74`), then `cobolReconcileMu`, released; then for the ledger, `cobolReconcileMu`, `cobolFindingMu` (drain), `findingLedgerMu`. This is the binding order.
2. **Nothing after `ResolveAll`'s reconcile step depends on COBOL reconcile results.** After `:1702` the pass runs:

   | Pass | Why it is independent of COBOL reconcile output |
   |---|---|
   | `resolveGodotResPaths` | Gated on godot/gdscript; matches `res://` targets only |
   | `resolveRazorUsings` | Gated on razor; matches `unresolved::razor_using::` targets only |
   | `attributeNonGoModuleImports` | Gated on python/dart; matches `external::` targets only |
   | Java and PHP override dispatch | `ScanOverrideDispatchCalls` admits only `java` or `php` callers |
   | Terminal stamping | `identifierFromTarget` returns "" for `cobol::` targets, so both the per-edge and the bulk classifier treat them as non-terminal and only unstamp edges they marked. The existing `generic` subtest asserts "COBOL target is never classified terminal". |
   | `compactDeferredLSPRetries` | Touches LSP retry records only, and COBOL targets never enter them |

   - After `ResolveAllContext` returns, `reconcileRetargetedTestCalls` still runs after the writer section, so the order relative to the COBOL reconcile is unchanged.
   - The cross-repo resolver (`runCrossRepoResolveContext`) never reaches `reconcileCobol`.

**Tests:**
- `TestCobolFindings_MasterReconcileExcludesLane` (`cobol_findings_ledger_exclusion_test.go:128`) has two subtests, one per production writer-free entry:
  - `pre_enrich_resolve`: `RunPreEnrichResolve`, the daemon warmup;
  - `deferred_fallback`: `resolveDeferredMutations(nil, true, scope, false)`, the deferred catch-up full fallback.
- Its helper `requireCobolMasterCloseBeforeLane` (`:160`) works as follows:
  - NOPGM is retrieved through an interrupted reindex, so the master's reconcile closes NOPGM's finding.
  - The master parks at `master_reconciled`, and the test reads `TopologyWriterActive()` once.
  - The lane then adds `CALL 'NEWPGM'` to the same paragraph. When the writer is held, the master is released first. When it is not, the lane runs to completion first.
  - The test requires exactly two rows, in order: NOPGM `active -> closed`, then NEWPGM `"" -> active`.
  - It uses hooks and channels only, with no sleep. Every wait is a `select` on a 2-minute `context.WithTimeout` that calls `t.Fatal`, and a cleanup releases a parked master.
- `TestCobolReconcile_ResolveAllDefersReconcile` (`cobol_findings_test.go:625`, memory + SQLite) pins the flag at both `ResolveAll` sites:
  - a pass with pending edges leaves the unresolved edge undecided;
  - a pass with nothing pending leaves a placeholder edge unbound;
  - in both cases a later `ReconcileCobol` makes the decision.
  - The interleaving test cannot catch a dropped guard, because with the guard gone the in-section reconcile is a no-op. This test closes that gap.

**Failing-first evidence:**
- **Mutation D24:** remove the writer section. The reconcile runs without the writer, and the ledger takes the writer itself, which is the pre-fix shape.
  - Both subtests FAIL with `[nopgm, active, superseded]` in place of `[nopgm, active, closed]` (`writer held: false`).
  - That is WR-01's misclassification exactly. The failure is the assertion at about 4.7 s, not a timeout.
- **Mutation G1:** ignore the flag at the settle-window site. The test FAILS on both stores: `a pass with pending edges defers the decision` (the edge is already on the placeholder).
- **Mutation G2:** ignore the flag at the nothing-pending site. The test FAILS on both stores: `a pass with nothing pending defers the binding` (the edge is already bound).

**Docs:** `docs/languages.md:279` previously said that a fallback full resolve "records its transitions at the step that made them, so whether a finding is closed or superseded does not depend on timing". It now states exactly what ships:
- A full resolve outside indexing (the daemon warmup resolve or a fallback catch-up) reconciles and records its transitions in one step that no edit can interleave.
- Every transition is therefore recorded by the step that made it.
- A finding is recorded as superseded only when the same step removed it and added a finding of the same class and kind in the same container.

The blanket "does not depend on timing" was dropped on purpose. If an edit's own step runs first and makes both changes, its supersede is legitimate under the D-03 pairing rule (`graph/finding_ledger.go`, pairing by `(Container, Class, Kind)` within one diff), so the label depends on which step made the change, not on interleaving.

**Behaviour changes on writer-free master paths only:**
- The master's COBOL reconcile now runs after `ResolveAll`'s refinement tail rather than inside it (check 2), and it waits for the reach writer.
- `OnComputeDone` (`markReady`) still fires inside the pass, before the reconcile, so readiness is not delayed.
- A failed or cancelled pass no longer reconciles COBOL. Before the fix, a cancellation after the reconcile step had already reconciled. The next settle point or resolve now reconciles instead, and the master's change record is empty, so the skipped ledger loses nothing.

### WR-02: Collapsed imageless occurrences relabelled `same_line_occurrence`

**Status:** fixed (prior session); the regression test was confirmed and mutation-proven in this session
**Commit:** `fc0fe9e9` `fix(05): WR-02 keep the absence reason of collapsed imageless occurrences`

**Files modified:**
- `internal/resolver/cobol_findings.go:274`: occurrences 2..n are relabelled only when the edge has a source line (`if i > 0 && d.e.Line > 0`). Otherwise they keep the edge's own absence reason and have no line or coordinates.
- The doc is at `:166-170`.
- `internal/resolver/cobol_findings_test.go`: the test below.

**Test:** `TestCobolReconcile_ExternalImagelessOccurrences` (`cobol_findings_test.go:1141`). For each of `no_original_projection` and `synthetic_projection`, on both stores, with a Line-0 edge and `cobol_occurrence_count=2`:
- two findings, `base` and `base#2`;
- both keep the edge's absence reason and `prov_range_exact=false`;
- neither has a coordinate key or a line.

**Failing-first evidence (mutation WR02, this session):** drop `&& d.e.Line > 0`. All four subtests FAIL: `#2` gets `prov_range_absence=same_line_occurrence` in place of `no_original_projection` or `synthetic_projection`. The control `TestCobolReconcile_ExternalSameLineOccurrences` still passes.

## D-24 Call-Site and Writer-State Table (HEAD `be0f4f83`)

Every caller of `runMasterResolveHookedContext` is listed. "Deferred" means that `ResolveAll` skips the reconcile, which then runs with the ledger in the new writer section (`multi.go:1087-1097`).

| # | Entry (`multi.go`) | Production caller | `topologyHeld` | Writer while the master runs | COBOL reconcile | Ledger |
|---|---|---|---|---|---|---|
| 1 | `runMasterResolve` (`:1016`) | none (tests only) | false | none | deferred | same writer section |
| 2 | `runMasterResolveTopologyHeld` (`:1023`) | incremental no-frontier fallback (`incremental_watcher_batch.go:413`) | true | the lane's writer | inside `ResolveAll` (unchanged) | inherits the writer (unchanged) |
| 3 | `resolveDeferredMutationsMode` full fallback (`:928`), via `FinishTailResult` (`:837`) | `BeginDeferredPasses` (`daemon_state.go:909`); `RunDeferredPassesAllResult` (`:612`, `daemon_state.go:1001`) | false | none | deferred | same writer section |
| 4 | same, via `finishColdDeferredPasses` (`:622`, IndexAll `:2493`) | IndexAll cold tail | true | IndexAll's writer | unchanged | unchanged |
| 5 | `resolveDeferredMutations`, 4-argument (`:910`) | none (`deferred_receipt*_test.go`) | false | none | deferred | same writer section |
| 6 | `runPreEnrichResolve` from IndexAll (`:2486`) | IndexAll | true | IndexAll's writer | unchanged | unchanged |
| 7 | `RunPreEnrichResolve` (`:1209`, call at `:1219`) | daemon warmup (`daemon_state.go:949`) | false | none | deferred | same writer section |
| - | `runMasterResolveFiles` (`:1114`), `runMasterResolveNames` (`:1150`) | receipt-exact catch-up | n/a | n/a | none (no `ResolveAll`) | none (unchanged) |

- **Lanes:** the MCP sites the review cites (`tools_fileops.go:785`, `tools_core.go:1578`) are lanes (`IncrementalReindexRepo`), not master callers. They take the writer from commit through ledger, so they now wait for a writer-free master's reconcile and ledger section.
- **Lock order** is unchanged everywhere: batch gate, then reach topology writer, then resolve mutex and `cobolReconcileMu`, then `findingLedgerMu`, then `cobolFindingMu`.

## Test Results

All tests ran in the **main checkout** (see Deviations), `-race -count=1`, each with an explicit `-timeout`. None timed out.

| Command | Result | Duration |
|---|---|---|
| `go build ./...` | ok | 15 s |
| `go vet ./internal/resolver/ ./internal/indexer/` | ok | n/a |
| `gofmt -l` on all changed files | clean | n/a |
| CR-01 targeted: resolver `TestParseCobolPlaceholderID`, `TestCobolReconcile_IDMSColonRepoPrefix`, `TestCobolReconcile_PlaceholderIDMSSecondPass` (`-timeout 300s`) | ok | 3.9 s |
| CR-01 targeted: indexer `TestCobolFindings_IDMSColonRepoPrefix` (`-timeout 900s`) | ok | 5.2 s |
| WR-01 targeted: indexer `MasterReconcileExcludesLane`, `LedgerExcludesTornMutation`, `MasterDrainClosesBeforeLaterEdit` (`-timeout 900s`) | ok | 16.7 s |
| WR-01 targeted: resolver `ResolveAllDefersReconcile`, `PlaceholderResolveAllNoPending` (`-timeout 300s`) | ok | 4.5 s |
| `go test -race -timeout 300s ./internal/resolver/` (whole package) | ok; 0 FAIL, 0 DATA RACE | 62.5 s (wall 68 s) |
| `go test -race -timeout 900s ./internal/indexer/ -run '^TestCobolFindings_\|^TestCobolLifecycle_\|^TestCobolCascadeGate_\|^TestNestedTrackedRoot_\|^TestRunRepositoryTopologyBatch'` | ok; 73 top-level tests PASS (137 RUN lines with subtests); 0 FAIL, race, or panic | 299.9 s (wall 306 s) |
| Indexer master and deferred selection (`-timeout 900s`): `TestBeginDeferredPassesOverlap*`, `TestColdDeferredCohortRetainsRegisteredUniverseScope`, `TestCrossRepoResolveReconcilesRetargetedTestCall`, `TestDeferredEnrichmentReleasesEveryRepoStateAfterDrain`, `TestFinishTailResult*`, `TestMasterResolveWholeGraphReconcilesRetargetedTestCalls`, `TestMultiIndexer_RunPreEnrichResolve_*`, `TestMutationReceiptEvict*`, `TestResolveDeferredMutations*`, `TestRunDeferredPasses*`, `TestRunPreEnrichResolve*` | ok; 32 top-level tests PASS, 0 FAIL, 0 SKIP, 0 race | 9.5 s (wall 15 s) |

**Not run, per instructions:** the full suite, and `cmd/gortex` (its guard trips on outside Gortex activity, and no `cmd/gortex` file changed).

### Mutation Runs

Every mutation was applied with an Edit and reverted with an Edit. After each revert, `grep -rn MUTATION internal cmd` was empty, and the tree was clean except for the fix being committed.

| Mutation | Test | Result |
|---|---|---|
| A: `ParseCobolPlaceholderID` cuts at the first `::` | the three CR-01 tests | FAIL. Resolver: `a::b` IDs unparsed; `pass 2: the one IDMS finding` on memory and SQLite. Real path: no IDMS finding after the cold track. |
| B: old first-`::` `cobolOwnerPrefix` body | `TestCobolFindings_IDMSColonRepoPrefix` | FAIL: after the identical re-index the finding is not in symbol FTS (routed to `a`) |
| WR02: drop `&& d.e.Line > 0` | `TestCobolReconcile_ExternalImagelessOccurrences` | FAIL 4 of 4: `#2` gets `same_line_occurrence` |
| D24: writer section removed; the ledger takes the writer itself (the pre-fix shape) | `TestCobolFindings_MasterReconcileExcludesLane` | FAIL 2 of 2: NOPGM `active -> superseded`, `writer held: false` |
| G1: settle-window guard ignored | `TestCobolReconcile_ResolveAllDefersReconcile` | FAIL on memory and SQLite |
| G2: nothing-pending guard ignored | `TestCobolReconcile_ResolveAllDefersReconcile` | FAIL on memory and SQLite |

## Deviations

1. **No worktree.**
   - `workflow.use_worktrees` is unset, so the agent default would have created one.
   - The user's global AGENTS.md forbids worktrees, and the uncommitted CR-01 work lived in the main checkout. All edits, commits, and verification therefore ran in the main checkout. No worktree, temporary branch, or recovery sentinel was created, so there was no cleanup tail.
   - The numbers above are reproducible from this checkout.
2. **Mutations reverted with Edit, never `git checkout --`.** The run constraints forbid `git checkout --`. Each revert was confirmed by re-reading the code and by the marker grep.
3. **Commits used plain `git commit`** with explicit paths and hooks enabled, as the run constraints require, rather than `gsd-tools query commit`. Never staged: `.planning/config.json`, `.planning/state.json`, `.planning/milestone.lock`, and the phase 06 and 06.1 directories.
4. **Two tests beyond the spec:**
   - `TestCobolReconcile_ResolveAllDefersReconcile`, the guard pin explained under WR-01;
   - a second interleaving subtest, so that both production writer-free entries (warmup and deferred fallback) are covered, not just one.
5. **Fix and tests share one commit per finding.** RED comes from the recorded mutation runs instead of separate test-first commits, as the run instructions specified.

## Residuals (not fixed; out of scope or info-tier)

- **WR-02 secondary note.** `fc0fe9e9` keeps `EndLine = StartLine` for later occurrences of an edge that has a source line. The review's note about a multi-line later occurrence is therefore not addressed. The run marked WR-02 done, so it was not changed.
- **IN-06.** The new writer section releases with `finish(false)`, matching `reconcileCobolRepos`. The reach index is not invalidated when the reconcile rebinds.
- **IN-03.** The two existing `LedgerExcludesTornMutation` subtests still use unbounded waits. Only the new subtests use bounded ones.
- **IN-02.** `mcp/scope_resolve.go` still derives a placeholder's repository with `graph.RepoPrefixOfID`. It could now use `ParseCobolPlaceholderID`.
- **Other first-`::` splitters.** Core helpers that cut node IDs at the first `::` (`graph.IDFile`, `graph.EnclosingFromID`, stub detection, MCP rename) are not placeholder parsers and are untouched. The verifier noted them as pre-existing, outside Phase 5.
- **gofmt.** Two pre-existing, unrelated unformatted files are untouched: `internal/resolver/csharp_verbatim_type_ref_test.go` and `internal/indexer/file_delta.go`.
- **Not done:** STATE.md, ROADMAP.md, and REQUIREMENTS.md were not updated, per instructions.

---

_Fixed: 2026-10-26T23:21:28Z_
_Fixer: the agent (gsd-code-fixer), resumed after an API disconnect_
_Iteration: 1_
