---
phase: 05-explicit-unresolved-evidence
verified: 2026-10-26T23:51:21Z
verified_at_commit: 64d20a6e
status: human_needed
score: 5/5 roadmap success criteria; gap-closure plan truths 42/42 (1 by override); 05-01..05-12 truths hold
behavior_unverified: 0
overrides_applied: 1
overrides:
  - must_have: "The full suite `GOWORK=off go test -race -timeout 60m ./...` runs once, after `df -g /System/Volumes/Data` shows at least 10 GiB available, and its log ends with exit=0 and holds no FAIL line (the gap-closure phase gate)."
    reason: "The substance is verified on split evidence; the literal clause cannot be met while any Gortex process on this machine appends to the developer's real query log during the run. Run 1 (05-17-full-suite.run1.log) executed all 156 testable packages fresh (0 cached) with 0 '--- FAIL', 0 'panic:', and 0 'DATA RACE' lines; cmd/gortex printed PASS (m.Run returned 0) and failed only on the pre-existing TestMain fingerprint guard (cmd/gortex/main_test.go:56-61), whose report names only ~/.gortex/cache/query-log.jsonl, written by outside sessions. cmd/gortex passed alone with the guard intact (05-17-cmd-gortex-isolated.log, exit=0), and the guard also stayed intact in this verification's targeted cmd/gortex COBOL run. No non-.planning file changed after 3abe472d. WINDOWS.md entry 8 stays open until a clean full run or a guard fix."
    accepted_by: "cfanch06 (user decision recorded in 05-17-SUMMARY key-decisions and deviation 6, and in WINDOWS.md entry 8)"
    accepted_at: "2026-10-26T18:55:45Z"
re_verification:
  mode: "focused: the two open gaps and review WR-02; everything else is carried forward from c93bce3c unless commits fc0fe9e9..64d20a6e touch it"
  previous_status: gaps_found
  previous_score: "4/5 roadmap success criteria; gap-closure plan truths 40/42 (1 by override)"
  previous_report_commit: c93bce3c
  commits_verified: ["fc0fe9e9", "1750e339", "be0f4f83", "64d20a6e"]
  gaps_closed:
    - "Gap 1, review CR-01 (SC1 / GAP-02; 05-16 truths 5 and 6; 05-08 truths 1 and 9 for '::' prefixes): resolver.ParseCobolPlaceholderID (cobol_findings.go:97-111) finds the namespace marker by its last occurrence, and cobolUnresolvedFromPlaceholder (:117-123) and cobolOwnerPrefix (finding_ledger.go:296-301) both use it. TestCobolReconcile_IDMSColonRepoPrefix (memory and SQLite, three passes) and TestCobolFindings_IDMSColonRepoPrefix (real path) pass. The verifier's earlier a::b probes now keep the finding, and a new nil-base master-path probe proves ledger and search routing to a::b, with a negative control."
    - "Gap 2, review WR-01 (D-24 and clause 2 of the 05-14 GAP-07 test-tier prohibition): writer-free masters skip ResolveAll's COBOL reconcile (DeferCobolReconcile; resolver.go:177-184, :754, :1702) and run ReconcileCobol, the drain, the ledger, and the search sync in one reach-writer section (multi.go:1087-1097). TestCobolFindings_MasterReconcileExcludesLane (both production writer-free entries) and TestCobolReconcile_ResolveAllDefersReconcile pass. A verifier overlay mutation without the writer section fails both subtests in 4.46 s with 'active -> superseded'. docs/languages.md:279 no longer claims timing independence and states what ships."
  warnings_closed:
    - "Review WR-02 (D-26): later occurrences of an imageless edge keep the edge's own absence reason and get no line or coordinate (cobol_findings.go:274). TestCobolReconcile_ExternalImagelessOccurrences passes (4 subtests), and the verifier's earlier probe now shows #2 as no_original_projection."
  gaps_remaining: []
  regressions: []
  previous_human_verification_open:
    - "Judgment-tier prohibitions 05-07 (GAP-08) and 05-09 (GAP-07): still not deterministically provable; carried forward below with refreshed evidence."
human_verification:
  - test: "Judgment-tier prohibition 05-07 (GAP-08): an uncontained cascade never passes silently or behind a flag, never blocks indexing or marks a repository unhealthy, and no later AI output can clear the cascade finding."
    expected: "Confirm. Non-authoritative LLM-judge verdict: HOLDS. TestCobolCascadeGate_{CleanControl,Positive,Negative,Copybook} pass at HEAD 64d20a6e in the default -race run with no flag. TestFindingWriters_Inventory passes at HEAD. TestIndexHealth_FindingsDoNotScore passed at c93bce3c, and internal/mcp is untouched since. The graph exposes no generic remove-by-ID, so a finding leaves the store only through eviction of its file or repository or the fenced resolver call to EvictFindingNodesByIDs. The review-fix commits add no finding writer."
    why_human: "Judgment-tier prohibition (unverified-prohibition: human review recommended). A must-NOT about future AI output cannot be proven by a present-day test."
  - test: "Judgment-tier prohibition 05-09 (GAP-07): no path other than deterministic evidence (source edit, parser or extractor change, retrieval or binding) closes, supersedes, suppresses, or alters a finding."
    expected: "Confirm. Non-authoritative LLM-judge verdict: HOLDS. The D-16 inventory (TestFindingWriters_Inventory, PASS at HEAD) confines KindFinding construction, finding_status writes, AppendFindingTransitions, and EvictFindingNodesByIDs to the extractor, the resolver, and the ledger. The new writer section (multi.go:1087-1097) only calls the existing resolver reconcile and ledger. Residual: the inventory matches literal write shapes, not a Meta write through a computed key. The previous residual (a race could mislabel close versus supersede on writer-free master paths) is closed by be0f4f83."
    why_human: "Judgment-tier prohibition (unverified-prohibition: human review recommended)."
  - test: "Re-run the full race suite on HEAD 64d20a6e with no Gortex sessions active (stop the daemon and every 'gortex mcp' process first): GOWORK=off go test -race -timeout 60m ./..."
    expected: "exit=0, with no '--- FAIL', 'DATA RACE', or 'panic:' line, and cmd/gortex passing with its TestMain guard intact. That re-certifies the 05-17 gate on the code that now ships and closes WINDOWS.md entry 8."
    why_human: "Code changed in internal/resolver and internal/indexer after 3abe472d, the commit the 05-17 override covers. This verification ran the whole resolver package (1075 tests) but only 151 of the 1688 internal/indexer test functions, and not cmd/gortex, whose daemon warmup is the production caller of the changed writer-free master path. cmd/gortex cannot pass while any Gortex session on this machine writes the real query log (guard at cmd/gortex/main_test.go:56-61), and sessions are active now."
---

# Phase 5: Explicit Unresolved Evidence Verification Report

**Phase Goal:** Users can see and distinguish source that could not be parsed from referenced artifacts that were not retrieved, without fabricated evidence.
**Verified:** 2026-10-26T23:51:21Z (HEAD `64d20a6e`)
**Status:** human_needed
**Re-verification:** Yes, focused, after the review fixes `fc0fe9e9`, `1750e339`, `be0f4f83`, and `64d20a6e`. Previous report: `c93bce3c` (gaps_found, 4/5).

## Verdict

- **Gap 1 (CR-01) is closed.**
  - Placeholder IDs are parsed at their last namespace marker, and both parsers use the one helper.
  - In a repository whose prefix contains `::`, the `COPY IDMS` finding now survives unchanged passes, on memory, SQLite, and the real path.
  - Ledger and search routing go to the full prefix. A new verifier probe with a negative control proves the ledger route, which the committed real-path test does not discriminate on its own.
- **Gap 2 (WR-01, D-24) is closed.**
  - Writer-free masters reconcile COBOL and record it in one reach-writer section.
  - The interleaving test covers both production entries. It is deterministic and fails fast: a verifier mutation reproduced the old misclassification in 4.46 s.
  - The docs no longer claim timing independence. They state what ships.
- **WR-02 is closed** for imageless occurrences.
  - One INFO residual remains: later same-line occurrences of an imaged edge keep `EndLine = StartLine`. It does not affect SC2.
- **No regressions** in the targeted runs. Everything carried forward from `c93bce3c` still holds where these commits touch it.
- **Why human_needed and not passed:**
  - The two judgment-tier prohibitions stay human items.
  - The full-suite gate passes by override, but its evidence predates these four commits. Re-running the full race suite on HEAD with no Gortex sessions active is human item 3. Per instructions, it is not a gap.

## Focused Checks

### 1. Gap 1: CR-01 (`missing` items of the previous report)

| Required item | Status | Evidence |
|---|---|---|
| Parse placeholder IDs at the namespace marker, not the first separator, in `cobolUnresolvedFromPlaceholder` and `cobolOwnerPrefix` | CLOSED | `ParseCobolPlaceholderID` (`cobol_findings.go:97-111`) takes the last `::cobol-missing::` (`:98`). It rejects an empty owner or one containing `/` (`:99`), an empty key or one containing `:` (`:103`), and any type outside the four (`:106-110`). Keys never contain `:`: every CALL, COPY, `-INC`, IDMS, and CICS key passes `cobolIDSegment`, which percent-encodes `:`, `/`, `#`, and `%` (`cobol_grammar_identity.go:214`; `cobol_grammar_refs.go:62`, `:144`, `:152`; `cobol_grammar_mainframe.go:187`). The last marker is therefore the real one. `cobolUnresolvedFromPlaceholder` uses the helper (`:117-123`). `cobolOwnerPrefix` uses it and otherwise falls back to `graph.RepoPrefixOfID` (`finding_ledger.go:296-301`). A grep for `cobol-missing` and `cobolMissingNamespace` finds no other parser. The sweep-candidate check (`:219`) compares against the node's own `RepoPrefix`, so it is prefix-safe. |
| Resolver regression: prefix `a::b`, one COPY IDMS edge, two unchanged `ReconcileCobol(nil)` passes keep the finding on memory and SQLite | CLOSED | `TestCobolReconcile_IDMSColonRepoPrefix` (`cobol_findings_test.go:235`) runs one creating pass and two unchanged ones. Every pass, on both stores, has exactly one finding and one placeholder, with the finding's `references` edge on that placeholder. PASS. `TestParseCobolPlaceholderID` (`:187`) round-trips all four types for `repo`, `a::b`, `x::y::z`, and `x::cobol-missing::program`. It rejects file-owned finding IDs and file paths that spell the namespace. PASS. Verifier probe `TestVerifierProbe_IDMSColonRepo` (re-run): `a::b` now gives pass1 = pass2 = [the IDMS finding] on memory and SQLite. It gave pass2 = [] before. |
| Real-path regression: repository `a::b` with `COPY IDMS RECORD EMP-REC`; after the cold track the finding exists, and the ledger and search sync route its IDs to `a::b` | CLOSED | `TestCobolFindings_IDMSColonRepoPrefix` (`cobol_findings_lifecycle_test.go:945`) checks after the cold track and after an identical re-index. The finding exists with a stable ID and has exactly one `"" -> active` row under `a::b` and none under `a`. It is in symbol FTS. PASS. **Discrimination:** the FTS check covers search routing (fixer mutation B). The history check alone cannot catch a ledger misroute, because both settle points there ledger `a::b` through their base. Verifier probe `TestVerifierProbe_MasterLedgerColonPrefix` closes this. It runs a writer-free master with nil scope, so there is no ledger base and `cobolOwnerPrefix` is the only route. The master records `[NOPGM finding, active, closed]` under `a::b` at its own step and removes the finding from FTS. With the old first-`::` body swapped in by overlay, `a::b` records `[]` and keeps the FTS row, while the `fcolon` control passes. Verifier probe `TestVerifierProbe_RealPathColonPrefix` (re-run): `a::b` now has 2 external findings (IDMS and CALL), the same as `fidms`. It had 1 before. |

### 2. Gap 2: WR-01 / D-24 (review option 1 taken)

| Check | Status | Evidence |
|---|---|---|
| The master's COBOL reconcile and its settle ledger share one writer section on the writer-free paths | VERIFIED | `multi.go:1063` sets `master.DeferCobolReconcile = !topologyHeld`. `ResolveAll` skips both reconcile sites under the flag: nothing pending (`resolver.go:754`) and the settle window (`:1702`). Without the writer, and only when `err == nil && ctx.Err() == nil`, `multi.go:1087-1097` runs one section: `reach.BeginTopologyMutation` (`:1089`), `master.ReconcileCobol(cobolScope)` (`:1091`), the test hook (`:1092-1094`), then `recordCobolSettleLedger(nil, true, master)` (`:1095`), which drains, ledgers, and syncs search (`cobol_reconcile.go:167-181`). The writer is released by `defer finish(false)` (`:1090`). `SetScope` stores the map verbatim (`resolver.go:568-570`), so `cobolScope` is exactly the scope `ResolveAll` would have used. Production writer-free entries: (a) the daemon warmup `RunPreEnrichResolve` (`daemon_state.go:949` to `multi.go:1210`, `topologyHeld` false); (b) the deferred catch-up full fallback (`BeginDeferredPasses` at `daemon_state.go:909` or `RunDeferredPassesAllResult` at `:1001`, then `FinishTailResult` `multi.go:837`, then `:928`). Only `finishColdDeferredPasses` sets `topologyHeld` true (`:622`). Neither entry runs under a writer: the topology batch at `daemon_state.go:638` returns before `:803`. Every row of the fixer's D-24 call-site table matches a grep of the callers. |
| The topologyHeld path is unchanged | VERIFIED | With `topologyHeld` true, `DeferCobolReconcile` is false, so `ResolveAll` reconciles inside the pass as before. `recordCobolSettleLedger(nil, true, master)` (`:1086`) is identical to the previous `(nil, topologyHeld, master)`. Callers: the incremental no-frontier fallback (`incremental_watcher_batch.go:413`), IndexAll pre-enrich (`multi.go:2486`), and the IndexAll cold deferred tail (`:622`). `DeferCobolReconcile` is written only at `multi.go:1063` (grep), so per-repository resolvers keep reconciling inside `ResolveAll`. |
| Lock order: batch gate, writer, resolve mutex / `cobolReconcileMu`, `findingLedgerMu`, `cobolFindingMu` | VERIFIED | Inside the new section: writer (`reach.go:289`, process-global, non-reentrant), then `ResolveMutex` (`cobol_references.go:74`), then `cobolReconcileMu` (`:176-179`; the master's lock is `&mi.cobolReconcileMu`, `multi.go:1012`), then `cobolFindingMu` (`cobol_findings.go:405`, leaf), all released. Then `recordCobolFindingLedger` takes `cobolReconcileMu` (`finding_ledger.go:158-161`), drains under the leaf `cobolFindingMu` (`cobol_findings.go:449`), and takes `findingLedgerMu` (`finding_ledger.go:53`). The writer is not re-taken (`cobol_reconcile.go:168`). `cobolFindingMu` is a documented leaf (`resolver.go:506-509`) and is never nested with `findingLedgerMu`, so the stated order cannot be inverted. On entry the master holds nothing a lane takes after the writer: `ResolveAllContext` released `ResolveMutex` (`resolver.go:653-654`); the deferred reconcile never took `cobolReconcileMu`; `BeginParallelBatch` released `batchMutationGate` (`multi.go:542-543`); `FinishTailResult` released the lanes (`finishOwnedContractTails`); and warmup enrichment applies stay parked behind `applyGate` until `RunPreEnrichResolve` returns. The 151 indexer tests run here completed with no deadlock or timeout. |
| The interleaving subtests are deterministic and fail fast | VERIFIED | `TestCobolFindings_MasterReconcileExcludesLane` (`cobol_findings_ledger_exclusion_test.go:128`) has one subtest per production entry: `pre_enrich_resolve` and `deferred_fallback`. It uses hooks and channels only, with no sleep. The single `TopologyWriterActive` read happens while the master is parked in the hook. The COBOL tests are not `t.Parallel`; the package's three parallel test files run after the sequential tests. Every wait is a `select` on a 2-minute context that calls `t.Fatal` (`:177-188`, `:206-210`). A cleanup releases a parked master (`:192-194`). The row order is the same whether or not the lane reached the writer before the release. The guard pin `TestCobolReconcile_ResolveAllDefersReconcile` (`cobol_findings_test.go:625`, both stores) covers what the interleaving test cannot: with a dropped guard, the in-section reconcile is a no-op. **Fail-fast proof (verifier, overlay-only):** with the writer section removed and the ledger taking the writer itself, both subtests FAIL in 2.23 s each (4.46 s total, no timeout). NOPGM is recorded `active -> superseded`, not `active -> closed`, with `writer held: false`. |
| `docs/languages.md:279` claims exactly what ships | VERIFIED | "does not depend on timing" is gone. The text now says a full resolve outside indexing (the daemon warmup resolve or a fallback catch-up) "reconciles and records its transitions in one step that no edit can interleave", and that a finding is recorded as superseded only when the same step removed it and added a same-class, same-kind finding in the same container. That matches the code: writer-free masters use the new section; writer-held masters keep the caller's writer across the pass; lanes hold the writer from commit through ledger (D-23); supersede is the D-03 pairing within one ledger diff. Note (INFO): if an edit's own step runs before a writer-free master's section and makes both the retrieval close and a new open in the same container, that step records a supersede. That is legitimate under D-03 ("a different active finding of the same class and kind now sits in the same container") and is exactly what the sentence says. Clause 2 of the 05-14 prohibition ("MUST NOT let close versus supersede depend on when a later settle point happens to run") concerns a later step recording another step's change, which can no longer happen on these paths. |

**Behavior changes on writer-free paths** (verified by code reading):
- `OnComputeDone` (`markReady`) still fires inside the pass, so readiness is not delayed.
- The COBOL reconcile now runs after the refinement tail. Nothing in that tail reads COBOL bindings:
  - terminal stamping never classifies `cobol::` targets (`TestCobolNamespaceExclusions/generic` PASS);
  - godot, razor, attribution, and override dispatch are gated on other languages.
- A failed or cancelled pass reconciles and records nothing. After the old reconcile step, every error return is a `ctx.Err()` check (`resolver.go:1702-1859`). Earlier errors never reached the reconcile before the fix either.

### 3. WR-02

| Check | Status | Evidence |
|---|---|---|
| Imageless occurrences keep their own absence reason | VERIFIED | The relabel runs only when `i > 0 && d.e.Line > 0` (`cobol_findings.go:274`). Otherwise every occurrence is `newCobolExternalFinding(d)`, which copies the edge's `prov_*` keys verbatim (`:493-497`). `TestCobolReconcile_ExternalImagelessOccurrences` (`cobol_findings_test.go:1141`) covers `no_original_projection` and `synthetic_projection` on both stores: `base` and `base#2` both keep the edge's reason and `prov_range_exact=false`. PASS (4 subtests). Verifier probe `TestVerifierProbe_NoImageOccurrence` (re-run): `#2` is now `no_original_projection`. It was `same_line_occurrence` before. |
| No invented coordinates or lines | VERIFIED | `cobolStampMeta` stamps no coordinates and returns Line 0 for no-image and synthetic projections (`cobol_grammar_walk.go:268-277`). So an imageless edge has nothing to copy, and `EndLine` stays 0 because `prov_end_row` is absent (`cobol_findings.go:517-519`). The test asserts no coordinate key and `[StartLine, EndLine] = [0, 0]`. |
| Residual: later occurrences on a real line get `EndLine = StartLine` (`:281`) | INFO, does not affect SC2 | The occurrence carries `prov_range_exact=false`, `prov_range_absence=same_line_occurrence`, and no `prov_*` coordinate. That meets SC2's documented "exact range or an absence reason" contract. `StartLine` is true, because the dedupe key includes the start line. `EndLine` is a lower bound that understates only a later occurrence continued onto the next line, such as `CALL 'X'. CALL 'X'` followed by `USING A.`. It is pinned on purpose (`cobol_findings_test.go:1099`, "spans the shared line"). Follow-up, not a gap: set `EndLine = 0`, or document it as a lower bound. |

### 4. Targeted tests (this verification; all `GOWORK=off`, `-race -count=1`, explicit `-timeout`)

| Run | Command (abridged) | Result |
|---|---|---|
| Resolver, whole package | `-timeout 600s -v ./internal/resolver/` | ok, 58.3 s. 1073 top-level PASS (1437 with subtests); 2 SKIP (flag-gated `TestFrameworkCensusProbe`, `TestFrameworkSynthesisScopedProbe`); 0 FAIL, 0 DATA RACE, 0 panic. Includes all four new tests. |
| Indexer, requested selection | `-timeout 1500s -run '^TestCobolFindings_\|^TestCobolLifecycle_\|^TestNestedTrackedRoot_\|^TestRunRepositoryTopologyBatch'` | ok, 266.2 s. 69 top-level PASS: 33 `TestCobolFindings_` (31 plus the 2 new), 23 `TestCobolLifecycle_`, 11 `TestNestedTrackedRoot_`, 2 `TestRunRepositoryTopologyBatch*`. 119 with subtests; 0 FAIL, race, panic, or timeout. |
| Indexer, every caller of the changed master path | `-timeout 1200s`. It selects every top-level test in the 11 other indexer test files that call `RunPreEnrichResolve`, `runPreEnrichResolve`, `runMasterResolve*`, `resolveDeferredMutations*`, `RunDeferredPassesAllResult`, `BeginDeferredPasses`, or `FinishTailResult` | ok, 31.6 s. 78 PASS; 0 FAIL, race, or panic. A 38-test subset matching the fixer's selection also passed separately (14.1 s). |
| Cascade gate (SC5) | `-timeout 600s -run '^TestCobolCascadeGate_'` | ok, 30.2 s. 4/4 PASS |
| Writer inventory (D-16) | `-timeout 300s -run '^TestFindingWriters_Inventory$' ./internal/graph/` | PASS |
| Static | `go vet ./internal/resolver/ ./internal/indexer/`; `gofmt -l` on the 7 changed Go files; debt-marker grep on the 8 changed files | clean; clean; none |

- **Not run, per instructions:** the full suite and `cmd/gortex`.
- **Is the targeted evidence enough?**
  - It is enough to verify the three fixes and every direct caller of the changed code in `internal/indexer`.
  - It does not re-certify the full-suite gate on HEAD: 151 of 1688 indexer test functions ran, and the daemon warmup in `cmd/gortex` did not run. That is human item 3.

### 5. Verifier probes (overlay-only via `go test -overlay`; files in `$TMPDIR/opencode/verify05/`; the repository is untouched, `git status` clean outside `.planning`)

| Probe | Result |
|---|---|
| P1 `TestVerifierProbe_IDMSColonRepo` (resolver, re-run) | `r1` and `a::b`, memory and SQLite: pass1 = pass2 = [IDMS finding]; placeholder kept |
| P2 `TestVerifierProbe_RealPathColonPrefix` (indexer, re-run) | `fidms` and `a::b`: 2 external findings each, both placeholders, and two `"" -> active` rows under the repository's own prefix |
| P3 `TestVerifierProbe_NoImageOccurrence` (resolver, re-run) | `base` and `#2`: `no_original_projection`, `exact=false`, start 0, end 0 |
| P4 `TestVerifierProbe_OwnerPrefixColon` (new) | `cobolOwnerPrefix` returns the full prefix for 7 ID shapes (3 finding, 4 placeholder) under `r1`, `a::b`, `x::y::z`, and `x::cobol-missing::program`. PASS. With the old body: 21 failures (`a`, `x`) |
| P5 `TestVerifierProbe_MasterLedgerColonPrefix` (new) | Writer-free warmup master, nil scope: `fcolon` and `a::b` each record exactly `[NOPGM, active, closed]` at the master step, drop the finding from FTS, and write nothing under `a`. PASS. With the old body: `a::b` records `[]` and keeps the FTS row; `fcolon` passes |
| P6 D24 mutation of `multi.go` (overlay) | `TestCobolFindings_MasterReconcileExcludesLane` fails 2/2 in 4.46 s with `active -> superseded`, `writer held: false` |

### 6. Carry-forward

- **The 05-17 full-suite override is kept verbatim.** Its last factual clause ("No non-.planning file changed after 3abe472d") was true when it was accepted. Four commits have since changed 7 code files and the docs. The override stands as accepted for the code at `3abe472d`. Human item 3 re-certifies HEAD.
- **Both judgment-tier human items are kept.** Neither can be proven deterministically: one concerns future AI output, the other all conceivable paths. Their evidence is refreshed at HEAD.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Named unparsed tails and localized ERROR or MISSING nodes produce stable, source-positioned PARSE_UNRESOLVED findings; unavailable targets produce separate EXTERNAL_UNRESOLVED findings | VERIFIED | The previous FAIL (CR-01) is closed (check 1). This run passed the whole resolver package (`TestCobolReconcile_External*`, the IDMS tests), the 33 real-path `TestCobolFindings_*` tests including `_IDMSColonRepoPrefix`, and the cascade gate (4/4). Extractor emission (17 `TestCobolGrammar_Findings*` tests) is carried from `c93bce3c`; no extractor file changed since. |
| 2 | Each finding exposes stable identity, exact scope and range, containing symbol, resolution class, normalized reason, severity, parser version, and lifecycle status | VERIFIED (INFO residual) | WR-02 is closed (check 3). `TestCobolReconcile_ExternalMetaContract` and `_ExternalImagelessOccurrences` pass in this run. The extractor meta contract and the MCP and CLI `get_symbol` tests are carried from `c93bce3c`; those files are untouched, and `cmd/gortex` was not run, per instructions. The `EndLine` residual is INFO. |
| 3 | Missing-target placeholders preserve the observed target reference without fabricated source, body, or attributes | VERIFIED | Placeholder construction is unchanged (`cobol_findings.go:337-351`: D-11 Meta only, empty FilePath, no range). The CR-01 fix only parses IDs. The shape, never-candidate, refusal, and sweep-by-ID tests pass in the resolver run. `TestCobolFindings_PlaceholderNamespaceNoFileCollision` and `_PlaceholderSearch` pass in the indexer run. |
| 4 | Unchanged evidence reuses finding identities; corrected evidence or retrieved artifacts close or supersede only the applicable finding class | VERIFIED | These pass in this run: the identity regressions (`_SameKind*`, `_SecondProgram*`, `_ReferenceFormEditSupersedes`, `_SameLineCallsPerOccurrence`); D-23 (`_LedgerExcludesTornMutation`, both subtests); and D-24 including the concurrent window (`_MasterReconcileExcludesLane`, both subtests, and `_MasterDrainClosesBeforeLaterEdit`). For `::` prefixes, unchanged passes keep the IDMS finding. |
| 5 | Destructive parser-recovery cascades remain deterministic regression failures regardless of later AI output | VERIFIED (judgment prohibition flagged) | `TestCobolCascadeGate_{CleanControl,Positive,Negative,Copybook}` pass at HEAD with no flag. `TestFindingWriters_Inventory` passes at HEAD. The AI clause is human item 1. |

**Score:** 5/5 roadmap success criteria.

### Gap-Closure Plan Truths (05-13 to 05-17)

Rows changed by this re-verification are marked **(updated)**. All other rows are carried from `c93bce3c`.

| Plan | # | Truth (short) | Status | Evidence |
|---|---|---|---|---|
| 05-13 | 1 | ID and `finding_container` use the name-stable anchor; `member_of` targets the innermost statement | VERIFIED | `cobol_grammar_findings.go:177`, `:183`, `:203`, `:218-222`; extractor untouched since |
| 05-13 | 2 | Regression 1 (insert), extractor and real path | VERIFIED | Real-path `TestCobolFindings_SameKindInsertKeepsIDs` PASS (this run); extractor test carried |
| 05-13 | 3 | Regression 2 (delete) closes only ZQXCOL, no supersede | VERIFIED | `TestCobolFindings_SameKindDeleteClosesOnlyItsOwn` PASS (this run) |
| 05-13 | 4 | Regression 3: document scope on the file for 0, 1, or 2 programs | VERIFIED | `TestCobolFindings_SecondProgramKeepsDocumentFinding` PASS (this run) |
| 05-13 | 5 | Both rule constants bumped once, written to Meta | VERIFIED | `finding.go:32-33`; `cobol_findings.go:508-509` |
| 05-13 | 6 | D-25 six-field evidence; CALL, LINK, XCTL, LOAD distinct | VERIFIED | `cobol_findings.go:484-488`; `TestCobolReconcile_ExternalReferenceFormEvidence`, `TestCobolFindings_ReferenceFormEditSupersedes` PASS |
| 05-13 | 7 | D-26 n findings per collapsed edge; later occurrences carry no coordinates and `same_line_occurrence` **(updated)** | VERIFIED | `cobol_findings.go:268-284`. WR-02 closed: an imageless edge keeps its own reason. `TestCobolReconcile_ExternalSameLineOccurrences`, `_ExternalImagelessOccurrences`, `TestCobolFindings_SameLineCallsPerOccurrence` PASS. `EndLine` residual is INFO |
| 05-13 | 8-11 | Edge probes (adjacency, empty, ordering) and the D-22 doc | VERIFIED | Carried; files untouched |
| 05-13 | 12 | Suites pass under -race **(updated)** | VERIFIED | This verification's runs |
| 05-14 | 1 | Every MultiIndexer ledger runs under the writer and `cobolReconcileMu`; one lock order **(updated)** | VERIFIED | `cobol_reconcile.go:167-181`; `finding_ledger.go:158-161`; plus the new master section `multi.go:1087-1097` (check 2) |
| 05-14 | 2 | Taking and held variants at the right sites; no self-deadlock **(updated)** | VERIFIED | `TestNestedTrackedRoot_*` (11) and `TestRunRepositoryTopologyBatch*` (2) PASS in this run; 78 master-path callers PASS |
| 05-14 | 3 | Cold-track gap documented as benign | VERIFIED (documentation truth) | Carried; T-05-73 accepted |
| 05-14 | 4 | Concurrency probe (track_tail, master_drain) | VERIFIED | Both subtests PASS (this run) |
| 05-14 | 5 | D-24: master drain ledgers at its own step; writer state threaded **(updated)** | VERIFIED | `multi.go:1063`, `:1085-1097`; `topologyHeld` threaded from `:837`, `:928`, `:1210`, `:1219`, `:2486` |
| 05-14 | 6 | D-24: close, never a supersede **(updated)** | VERIFIED | Sequential: `TestCobolFindings_MasterDrainClosesBeforeLaterEdit` PASS. Concurrent window closed: `TestCobolFindings_MasterReconcileExcludesLane` PASS, mutation-proven |
| 05-14 | 7-9 | Idempotency edge probe; `TopologyWriterActive` and held-writer warning; standalone tail lock | VERIFIED | Carried; `_SearchAfterFallbackMasterResolve` PASS (this run) |
| 05-14 | 10 | Existing suites and the whole indexer package pass | VERIFIED (whole package at `3abe472d`) | Whole package from run 1. At HEAD, the requested selection plus every master-path caller passes (151 tests). The HEAD full re-run is human item 3 |
| 05-15 | 1-4 | Kind-guarded `EvictResourceNodesByIDs` on both backends | VERIFIED | Carried; files untouched |
| 05-16 | 1 | Placeholder ID shape, empty FilePath, audit exemption; no repository file can mint it **(updated)** | VERIFIED | `cobol_findings.go:73-86`, `:337-351`. The constant doc (`:15-22`) no longer overclaims; it points to the WINDOW |
| 05-16 | 2-4 | Sweep by ID only; WR-05 real path; refusal | VERIFIED | `TestCobolReconcile_PlaceholderSweepByID`, `_PlaceholderRefusedOverRealNode`, `TestCobolFindings_PlaceholderNamespaceNoFileCollision` PASS |
| 05-16 | 5 | COPY IDMS round trip from a `<repo>::cobol-missing::idms_dictionary_member/...` ID **(updated)** | VERIFIED (was FAILED) | Holds for `r1`, `a::b`, `x::y::z`, and a prefix that spells the marker (check 1) |
| 05-16 | 6 | `cobolOwnerPrefix` routes both `<repo>/...` and `<repo>::...` **(updated)** | VERIFIED (was FAILED) | `finding_ledger.go:296-301`; probes P4 and P5 with negative controls |
| 05-16 | 7 | `RepoPrefixOfID` caller audit with dispositions | VERIFIED | Its (d) rows assumed no `::` in a prefix. Both COBOL parsers no longer depend on that. The MCP scope sites are IN-02 (accepted fail-closed) |
| 05-16 | 8 | WINDOW paragraph | VERIFIED | `cobol_references.go:148-158` |
| 05-17 | 1-5 | Docs for D-22, D-27, D-21, D-25, D-26 | VERIFIED | Carried; IN-05 wording note stays INFO |
| 05-17 | 6 | Docs for D-23 and D-24 history timing **(updated)** | VERIFIED | `docs/languages.md:279` reworded at `be0f4f83` (check 2) |
| 05-17 | 7 | D-20 kept; inventory passes | VERIFIED | `TestFindingWriters_Inventory` PASS at HEAD |
| 05-17 | 8 | Full-suite gate log ends `exit=0` with no FAIL | PASSED (override) | Override accepted by cfanch06 on 2026-10-26 for the code at `3abe472d`. The HEAD re-run is human item 3 |

**Score:** 42/42 gap-closure truths. One is by override, and two (05-14 #10 and 05-17 #8) rest on whole-suite evidence from `3abe472d`.

### Plans 05-01 to 05-12 (Regression)

- **96 truths.** Nine were superseded in wording by user-confirmed D-21, D-24, D-25, and D-27. Their behavior is re-tested in the new form:

| Plan:line | Superseded wording | Superseded by |
|---|---|---|
| 05-01:31 | `gortex-finding-id/1` | D-21 (`/2`) |
| 05-05:27, 05-05:31, 05-11:26, 05-11:28 | `<repo>/cobol-missing::...` placeholder spelling, FilePath equal to the ID | D-27 (`<repo>::cobol-missing::...`, empty FilePath) |
| 05-08:30 | External evidence of form, IDMS kind, and raw target | D-25 (six fields) |
| 05-09:26, 05-11:22 | Ownership routing via `graph.RepoPrefixOfID` | D-27 (`cobolOwnerPrefix`: placeholder parser first, then `graph.RepoPrefixOfID`) |
| 05-11:23 | "no ledger runs there" at the master drain | D-24 |

- **The previous exception is resolved.** 05-08 truth 1 (one finding per `COPY IDMS` occurrence) and truth 9 (an unchanged second reconcile writes nothing) now hold for `::` prefixes:
  - `TestCobolReconcile_IDMSColonRepoPrefix` holds identical sets over three passes.
  - By code reading, an unchanged pass issues no rebind, eviction, or `AddBatch` (`cobol_references.go:348-352`; `cobol_findings.go:313-374`).

## Required Artifacts (changed by the review fixes)

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/resolver/cobol_findings.go` | `ParseCobolPlaceholderID`; IDMS recovery through it; WR-02 guard | VERIFIED | `:97-111`, `:117-123`, `:274`; wired from `reconcileCobol` (`cobol_references.go:287`) and `cobolOwnerPrefix` |
| `internal/indexer/finding_ledger.go` | `cobolOwnerPrefix` via the parser | VERIFIED | `:296-301`; used at `:186` (ledger widening), `:271`, and `:276` (search sync) |
| `internal/resolver/resolver.go` | `DeferCobolReconcile`, honoured at both reconcile sites | VERIFIED | `:177-184`, `:754`, `:1702`; set only at `multi.go:1063` |
| `internal/indexer/multi.go` | Writer section for writer-free masters; hook stage | VERIFIED | `:1063`, `:1085-1097`; hook doc `:159-167` |
| `internal/resolver/cobol_findings_test.go` | Four new tests | VERIFIED | `:187`, `:235`, `:625`, `:1141`; all PASS |
| `internal/indexer/cobol_findings_lifecycle_test.go` | `TestCobolFindings_IDMSColonRepoPrefix` | VERIFIED | `:945`; PASS |
| `internal/indexer/cobol_findings_ledger_exclusion_test.go` | `TestCobolFindings_MasterReconcileExcludesLane`, bounded waits | VERIFIED | `:128-238`; PASS; mutation-proven |
| `docs/languages.md:279` | D-24 sentence states what ships | VERIFIED | Check 2. IN-01 and IN-05 wording notes stay INFO |

The other gap-closure artifacts (extractor anchor, rule constants, resource evicter, reach writer probe, settle-ledger helper) are untouched and carried as VERIFIED.

## Key Link Verification (changed links)

| From | To | Via | Status |
|---|---|---|---|
| `runMasterResolveHookedContext` | `Resolver.DeferCobolReconcile` | `multi.go:1063`; read at `resolver.go:754`, `:1702` | WIRED |
| `runMasterResolveHookedContext` | `ReconcileCobol` and `recordCobolSettleLedger` | `multi.go:1091` and `:1095`, both inside the `BeginTopologyMutation` section at `:1089` | WIRED (atomic; previously non-atomic) |
| `reconcileCobol` (IDMS in-edge recovery) | `ParseCobolPlaceholderID` | `cobol_references.go:287` to `cobol_findings.go:118` | WIRED (previously misparsed `::` prefixes) |
| `cobolOwnerPrefix` | `resolver.ParseCobolPlaceholderID`, then `graph.RepoPrefixOfID` | `finding_ledger.go:297`, `:300` | WIRED |
| Ledger widening and search sync | `cobolOwnerPrefix` | `finding_ledger.go:186`, `:271`, `:276` | WIRED (previously misrouted `::` prefixes) |

Unchanged links from `c93bce3c` are carried as WIRED:
- extractor anchor to `graph.FindingSymbol`;
- ledger transition to `MetaFindingContainer`;
- reconcile to `cobol_occurrence_count`;
- settle ledger to `cobolReconcileMu`;
- evicter to `evictSyntheticNodesByIDs`;
- sweep to `EvictResourceNodesByIDs`;
- docs to `finding.go` rule IDs.

## Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|---|---|---|---|---|
| `index_health` `findings` | Per-class counts | `NodesByKind(KindFinding)` | Yes | FLOWING (carried; `TestIndexHealth_FindingCounts` PASS at `c93bce3c`, `internal/mcp` untouched) |
| MCP and CLI `get_symbol` on a finding or placeholder | Meta and edges | SQLite after a real index | Yes | FLOWING (carried; `cmd/gortex` not run, per instructions) |
| `search_symbols kind:finding` | FTS rows | `syncCobolFindingFTSByRepo` at each settle point, routed by `cobolOwnerPrefix` | Yes, now including `::` prefixes | FLOWING (`_IDMSColonRepoPrefix`, `_PlaceholderSearch`, `_SearchAfterIncremental`, `_SearchAfterFallbackMasterResolve` PASS; probe P5) |
| `finding_history` | Transition rows | `recordCobolFindingLedger` at each settle point, including the writer-free master section | Yes | FLOWING (`_MasterReconcileExcludesLane`, `_IDMSColonRepoPrefix`, probe P5) |

### Probe Execution

- **Step 7c: SKIPPED.** No `scripts/*/tests/probe-*.sh` exists, and no plan, summary, or the fix report declares one. The verifier's own overlay probes are listed under Focused Check 5.

## Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|---|---|---|---|---|
| GAP-01 | 05-01, 05-03, 05-10 | Unparsed tail, ERROR, or MISSING produces a persisted PARSE_UNRESOLVED finding | SATISFIED | Carried; cascade gate PASS at HEAD |
| GAP-02 | 05-02, 05-05, 05-08..05-10, 05-12, 05-13, 05-16, 05-17 | A not-retrieved target produces an EXTERNAL_UNRESOLVED finding | SATISFIED (was BLOCKED, partial) | CR-01 closed. `COPY IDMS` findings survive for `::` prefixes (check 1) |
| GAP-03 | 05-01, 05-04, 05-08, 05-10 | The two classes stay distinct in storage and query | SATISFIED | Carried |
| GAP-04 | 05-01, 05-03, 05-04, 05-08, 05-12, 05-13, 05-16, 05-17 | Full field contract | SATISFIED (INFO residual) | WR-02 closed; the `EndLine` lower bound is INFO |
| GAP-05 | 05-05, 05-11, 05-12, 05-15, 05-16, 05-17 | Placeholder without fabricated content | SATISFIED | Placeholder tests PASS; construction unchanged |
| GAP-06 | 05-01, 05-06, 05-08, 05-09, 05-11..05-14, 05-17 | Unchanged evidence reuses identity | SATISFIED | Identity regressions PASS; unchanged passes keep the IDMS finding for `::` prefixes |
| GAP-07 | 05-02, 05-05, 05-06, 05-08, 05-09, 05-11..05-17 | Close or supersede when evidence resolves or replaces it | SATISFIED (judgment clause flagged) | The history timing window is closed (check 2). The judgment clause is human item 2 |
| GAP-08 | 05-03, 05-07 | Cascade remains a deterministic regression failure | SATISFIED (judgment clause flagged) | Cascade gate PASS at HEAD; human item 1 |

- **Orphaned requirements:** none. All 8 IDs map to Phase 5 in REQUIREMENTS.md, and each appears in at least one plan's `requirements:`.
- **REQUIREMENTS.md** still shows all 8 unchecked, with "Gaps Found" in traceability. The orchestrator should update it after the human items resolve. This verifier does not edit it.

## Prohibitions (ADR-550)

| Plan | Requirement | Tier | Statement (short) | Disposition |
|---|---|---|---|---|
| 05-01 | GAP-01 privacy | test | No raw gap text in Name, QualName, ID, or Meta | VERIFIED (carried). D-22 residual documented |
| 05-01 | GAP-01 transparency | test | No dropping or aggregating of producer gaps | VERIFIED (carried) |
| 05-05 | GAP-02 transparency | test | No placeholder for present, builtin, ambiguous, dynamic, or bound targets | VERIFIED (resolver run) |
| 05-05 | GAP-05 safety | test | No fabricated placeholder content; never an implementation | VERIFIED |
| 05-07 | GAP-08 safety | judgment | Cascade never silent and never blocking; AI cannot clear it | FLAGGED (human item 1). LLM-judge verdict: HOLDS |
| 05-08 | GAP-02 transparency | test | EXTERNAL_UNRESOLVED means exactly "never obtained" | VERIFIED |
| 05-09 | GAP-07 safety | judgment | Only deterministic evidence changes a finding | FLAGGED (human item 2). LLM-judge verdict: HOLDS |
| 05-13 | GAP-06 transparency | test | No ID change or row for unchanged evidence after unrelated edits | VERIFIED |
| 05-13 | GAP-02 transparency | test | No merging of occurrences; no shared evidence across reference forms | VERIFIED, with the documented mixed-form same-line window. WR-02 closed |
| 05-14 | GAP-07 transparency | test | (1) No closed or superseded row while evidence exists; (2) close vs supersede never depends on when a later settle point runs | (1) VERIFIED. (2) **VERIFIED (was FLAGGED)**: wired enforcement is `TestCobolFindings_MasterReconcileExcludesLane`, failing first under the fixer's mutation D24 and this verifier's overlay mutation |
| 05-16 | GAP-05 safety | test | A placeholder never occupies, overwrites, or evicts a real node | VERIFIED |
| 05-17 | GAP-04 transparency | test | Docs never claim the digest prevents recovery or overstate reference-form edits | VERIFIED |

## Anti-Patterns Found

- **Debt markers:** none in the 8 files changed since `c93bce3c`. `gofmt` and `go vet` are clean. No `MUTATION` leftovers.
- **Resolved since `c93bce3c`:**
  - both BLOCKER rows (first-`::` parsing in `cobol_findings.go` and `finding_ledger.go`);
  - both D-24 WARNING rows (`multi.go`, and the docs timing claim);
  - the WR-02 WARNING row.

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `internal/resolver/cobol_findings.go` | 281 | `EndLine = StartLine` for later same-line occurrences of an imaged edge | INFO | WR-02 secondary note. The provenance contract stays honest, and a test pins the behavior (`cobol_findings_test.go:1099`) |
| `internal/indexer/cobol_findings_lifecycle_test.go` | 945 | The real-path CR-01 test does not discriminate ledger routing on its own (both its settle points ledger `a::b` through their base) | INFO | Covered by verifier probe P5. Suggest committing a nil-scope master assertion |
| `docs/languages.md` | 279 | The mixed-form window understates the collapse (spelling variants, imageless references) | INFO | IN-05 (carried) |
| `internal/indexer/cobol_findings_ledger_exclusion_test.go` | 94, 104 | The two older subtests still wait without a bound | INFO | IN-03 (carried); the new subtests are bounded |
| `internal/indexer/cobol_reconcile.go` | 95-96 | The comment claims `finish(false)` keeps the reach index valid | INFO | IN-06 (carried). The new section also releases with `finish(false)` (`multi.go:1090`) |
| `internal/indexer/repository_topology_batch.go` | 39-80 | In-batch exclusion relies on `deferResolve` | INFO | IN-04 (carried) |
| `internal/mcp/scope_resolve.go` | 466, 561 | Placeholders drop out of workspace-clamped views | INFO | IN-02 (carried). These sites could now use `ParseCobolPlaceholderID` |
| `docs/languages.md` | 279 | The old-store note omits that old external findings stay active beside new ones | INFO | IN-01 (carried) |
| Gortex-wide | n/a | A repository prefix containing `/` is unsupported: `graph.RepoPrefixOfID` cuts at the first `/`, and the config does not reject it. `::` prefixes are still misparsed by core helpers (`graph.IDFile`, `graph.EnclosingFromID`, stub detection, MCP rename) | INFO | Pre-existing and outside Phase 5. Phase 5's own parsers now handle `::` |

## Human Verification Required

### 1. Judgment prohibition 05-07 (GAP-08)

- **Test:** Review that a cascade can never pass silently or be cleared by AI, and never blocks indexing.
- **Expected:** Confirm HOLDS. Evidence:
  - the cascade gate passes at HEAD in the default suite, with no flag;
  - `TestFindingWriters_Inventory` passes at HEAD;
  - `TestIndexHealth_FindingsDoNotScore` passed at `c93bce3c` (`internal/mcp` untouched);
  - the graph has no generic remove-by-ID.
- **Why human:** Judgment-tier prohibition about future AI output.

### 2. Judgment prohibition 05-09 (GAP-07)

- **Test:** Review that no MCP, CLI, or AI path can close or alter a finding.
- **Expected:** Confirm HOLDS.
  - Residual: the inventory matches literal write shapes, not computed-key Meta writes.
  - The previous close-versus-supersede race on writer-free master paths is closed.
- **Why human:** Judgment-tier prohibition.

### 3. Full race suite on HEAD

- **Test:**
  1. Stop the Gortex daemon and every `gortex mcp` process.
  2. Run `GOWORK=off go test -race -timeout 60m ./...` once on `64d20a6e`, after `df -g /System/Volumes/Data` shows at least 10 GiB free.
- **Expected:**
  - `exit=0`, with no `--- FAIL`, `DATA RACE`, or `panic:` line;
  - `cmd/gortex` passes with its TestMain guard intact.
  - That re-certifies the 05-17 gate on HEAD and closes WINDOWS.md entry 8.
- **Why human:**
  - Code changed after `3abe472d`, the commit the override covers.
  - This verification ran 151 of 1688 indexer test functions and not `cmd/gortex`, whose daemon warmup is the production caller of the changed writer-free path.
  - `cmd/gortex` cannot pass while Gortex sessions write the real query log, and sessions are active on this machine now.

## Gaps Summary

- **No gaps remain.**
  - Gap 1 (CR-01) and gap 2 (WR-01, D-24) are closed with code, committed regressions, and mutation evidence. This verifier independently reproduced the negative controls for both.
  - WR-02 is closed.
- **No regressions** appeared in the targeted runs:
  - the whole resolver package;
  - 151 indexer tests, including every caller of the changed master path;
  - the cascade gate;
  - the writer inventory.
- **Status is human_needed:**
  - two carried judgment-tier prohibitions;
  - a full-suite re-run on HEAD with no Gortex sessions active, because the override's evidence predates these four commits.
- **Optional follow-ups (INFO, not gaps):**
  - set `EndLine = 0` for later same-line occurrences, or document it as a lower bound;
  - commit the nil-scope master-path ledger-routing assertion for a `::` prefix (verifier probe P5);
  - move IN-02's MCP sites to `ParseCobolPlaceholderID`.

---

_Verified: 2026-10-26T23:51:21Z_
_Verifier: the agent (gsd-verifier)_
