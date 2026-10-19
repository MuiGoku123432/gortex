---
phase: "5"
slug: "explicit-unresolved-evidence"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-19"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1 |
| **Config file** | none (build tag `darwin \|\| linux` on COBOL tests) |
| **Quick run command** | `GOWORK=off go test -race -count=1 ./internal/graph/... ./internal/resolver ./internal/parser/languages -run 'Finding\|Cobol'` |
| **Full suite command** | `GOWORK=off go test -race ./...` |
| **Estimated runtime** | quick ~60-90 seconds; full suite several minutes |

Measured during research: `TestCobolLifecycle_IdenticalRerun` 10.6s; `./internal/resolver -run Cobol` 5.5s; `./internal/parser/languages -run CobolGrammar` 26.1s (all `-race`).

---

## Sampling Rate

- **After every task commit:** Run the quick command plus `GOWORK=off go test -race -count=1 <package under edit>`
- **After every plan wave:** Run `GOWORK=off go test -race -count=1 -timeout 30m ./internal/graph/... ./internal/resolver ./internal/indexer ./internal/parser/languages ./internal/mcp ./internal/neo4jprojection ./cmd/gortex -run 'Finding|Cobol|PrefixDiagnostics|IndexHealth|ProjectNode|SidecarViewGen|Migration'`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

Requirement → test map (task IDs are filled in by the planner/executor; see 05-RESEARCH.md §Validation Architecture):

| Requirement | Behavior | Test Type | Automated Command | File Exists | Status |
|-------------|----------|-----------|-------------------|-------------|--------|
| GAP-01 | tail, ERROR, MISSING, lost/unmapped capture, document gaps become findings with container and range (one per producer tail gap, D-18) | unit (real producer) | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_Findings'` | ❌ W0 | ⬜ pending |
| GAP-01 | producer `-INC` external-reference gap never becomes a finding (D-17) | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_Findings_NoProducerExternal'` | ❌ W0 | ⬜ pending |
| GAP-02 | missing CALL/LINK/XCTL/COPY/-INC and COPY IDMS yield external findings; ambiguous/dynamic/bound/builtin (D-19) do not | unit | `GOWORK=off go test -race -count=1 ./internal/resolver -run 'TestCobolReconcile_(External\|Placeholder)'` | ❌ W0 | ⬜ pending |
| GAP-03 | classes distinct in Meta, ID, and `index_health` counts | unit + MCP | `GOWORK=off go test -race -count=1 ./internal/graph -run TestFindingID ; GOWORK=off go test -race -count=1 ./internal/mcp -run TestIndexHealth_FindingCounts` | ❌ W0 | ⬜ pending |
| GAP-04 | Meta contract, types, no secret-like keys, projection keeps keys | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run TestCobolGrammar_FindingMetaContract ; GOWORK=off go test -race -count=1 ./internal/neo4jprojection -run TestProjectNode_FindingMeta` | ❌ W0 | ⬜ pending |
| GAP-04 | get_symbol by exact ID (MCP + CLI), search by name/kind | integration | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolFindings_'` | ❌ W0 | ⬜ pending |
| GAP-05 | placeholder has no range/body/attributes, never a candidate, swept on last reference, owned (SEC-08) | unit + real path | `GOWORK=off go test -race -count=1 ./internal/resolver -run TestCobolReconcile_Placeholder ; GOWORK=off go test -race -count=1 ./internal/graph/... -run PrefixDiagnostics` | ❌ W0 / ⚠️ update | ⬜ pending |
| GAP-06 | identical rerun and line shift keep IDs; clean x2 and opposite track orders equal | real path | `GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer -run '^TestCobol(Lifecycle\|Findings)_'` | ⚠️ partial (harness exists) | ⬜ pending |
| GAP-07 | fix closes; text edit supersedes with successor; retrieve closes EXTERNAL only; parser re-stamp keeps ID; restore reopens; history survives restart and reindex, cleared by purge; a dependent repository's transition is recorded at the step that caused it | unit + real path | `GOWORK=off go test -race -count=1 ./internal/graph -run TestFindingLedger ; GOWORK=off go test -race -count=1 ./internal/graph/store_sqlite -run TestFindingHistory ; GOWORK=off go test -race -count=1 ./internal/resolver -run TestCobolReconcile_FindingChangesDrain ; GOWORK=off go test -race -count=1 -timeout 30m ./internal/indexer -run 'TestCobolFindings_(History\|CrossRepo)'` | ❌ W0 | ⬜ pending |
| GAP-07 | ledger rows land in SQLite on the shadow path; a shadow-path cold track is re-decided on the durable store (no fabricated not-obtained finding or placeholder, a dependent repository's finding closes, search follows) | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run 'TestCobolFindings_(HistoryShadowPath\|HistoryDependentRepo\|CrossRepoDependentLedgerAndSearch)'` (each sets `GORTEX_SHADOW_MAX_FILES` above corpus size in its shadow subtest and proves the shadow ran from the admissions counter) | ❌ W0 | ⬜ pending |
| GAP-07 | fallback master resolves sync search from their drained change record | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run TestCobolFindings_SearchAfterFallbackMasterResolve` | ❌ W0 | ⬜ pending |
| GAP-08 | positive shapes produce `uncontained-cascade`; negatives keep 20 paragraphs and 20 data items; no flag | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolCascadeGate'` | ❌ W0 | ⬜ pending |
| Deferred P3 | WR-06 identity-only copybook + `analysis-failed` (copybook-only; per D-20 a program whose analysis fails keeps the hard failure, documented in `docs/languages.md`); WR-07 ambiguous cursor; IN-08 distinct reason; WR-09 still green | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_(CopybookAnalysisFailed\|CursorAmbiguous\|NestedContainmentUnresolved\|CopybookProcedureMainframe)' ; GOWORK=off go test -race -count=1 ./internal/resolver -run TestCobolReconcile_ContainmentUnresolved` | ❌ W0 / ✅ WR-09 | ⬜ pending |
| Regression | Phase 2-4 COBOL suites with updated expectations | real path | `GOWORK=off go test -race -count=1 -timeout 30m ./cmd/gortex ./internal/indexer ./internal/resolver -run 'Cobol'` | ✅ (update) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

### Plan-task assignment (filled by the planner, 2026-10-19; revision 1: the original resolver plan split into 05-05 placeholders and 05-08 external findings; 05-06 ledger store and 05-07 cascade gate keep their numbers; revision 2: the ledger plan split into 05-09 (resolver change record, cross-repository ledger tails, history sequences) and 05-11 (search sync, external real-path lifecycle, benchmark); 05-10 keeps its number; the acceptance plan is now 05-12, wave 7; revision 3: the cold `TrackRepoCtx` tail re-decides on the durable store through `reconcileCobolRepos` (05-09 T2), fallback master resolves drain into search sync (05-11 T1), and 05-02 T2's verify is narrowed to its own fence and conformance cases)

| Requirement row | Plan-task | Tests created there |
|-----------------|-----------|---------------------|
| GAP-01 findings | 05-01 T1/T2, 05-03 T1 | `TestCobolGrammar_Findings_*`, `TestCobolFindings_MCPGetSymbol` |
| GAP-01 no producer external (D-17) | 05-01 T2 | `TestCobolGrammar_Findings_NoProducerExternal` |
| GAP-02 placeholders (no false not-obtained) | 05-05 T1, 05-09 T2 | `TestCobolReconcile_PlaceholderEdgeTargets`, `TestCobolReconcile_PlaceholderNotForBoundAmbiguousDynamicBuiltin`, `TestCobolFindings_HistoryDependentRepo` (fdepa-then-fdepb cold track, shadow off and on) |
| GAP-02 external findings | 05-08 T1 | `TestCobolReconcile_External*` |
| GAP-03 class separation | 05-01 T2, 05-04 T1, 05-08 T1 | `TestFindingID_ClassSeparation`, `TestIndexHealth_FindingCounts`, `TestCobolReconcile_ExternalMetaContract` |
| GAP-04 Meta contract / projection | 05-01 T2, 05-04 T2, 05-08 T1-T2 | `TestCobolGrammar_FindingMetaContract`, `TestProjectNode_FindingMeta`, `TestCobolReconcile_ExternalMetaContract`, `TestCobolFindings_ExternalPersisted` |
| GAP-04 MCP/CLI/search | 05-01 T1, 05-12 T1 | `^TestCobolFindings_` (cmd/gortex) |
| GAP-05 placeholder | 05-05 T1-T2, 05-11 T2, 05-12 T1 | `TestCobolReconcile_Placeholder*`, `TestCobolFindings_PlaceholderSweptOnLastReference`, `TestCobolFindings_Placeholder*` (cmd/gortex) |
| GAP-06 identity reuse | 05-01 T2, 05-08 T1, 05-09 T2, 05-12 T2 | `TestCobolGrammar_Findings_LineShiftKeepsIDs`, `TestCobolReconcile_ExternalOrdinalAndLineShift`, `TestCobolFindings_HistoryIdenticalRerun` (identical-byte re-extraction, ledger proven to run), `TestCobolFindings_CleanDeterminism` |
| GAP-07 ledger / history | 05-06 T1-T3, 05-09 T1-T3, 05-11 T2, 05-12 T2 | `TestFindingLedger_*`, `TestFindingHistory_*` (incl. `LatestPerFinding`), `TestCobolReconcile_FindingChangesDrain`, `TestCobolFindings_History*` (incl. `HistoryBatchDeferredTrack`), `TestCobolFindings_ExternalClosesOnRetrieve`, `TestCobolFindings_ExternalCrossContainerNoSupersede` |
| GAP-07 cross-repository ledger and search (D-03, SC4) | 05-09 T1-T2, 05-11 T1 | `TestCobolReconcile_FindingChangesDrain`, `TestCobolFindings_HistoryDependentRepo`, `TestCobolFindings_CrossRepoDependentLedgerAndSearch` (both with both cold-track orders, shadow off and on) |
| GAP-07 search sync (Pitfall 4) | 05-11 T1 | `TestCobolFindings_SearchAfterIncremental`, `TestCobolFindings_SearchAfterFallbackMasterResolve`, `TestCobolFindings_CrossRepoDependentLedgerAndSearch` |
| GAP-07 shadow path | 05-09 T2-T3, 05-11 T1 | `TestCobolFindings_HistoryShadowPath`; the `shadow=on` subtests of `TestCobolFindings_HistoryDependentRepo` and `TestCobolFindings_CrossRepoDependentLedgerAndSearch` |
| GAP-07 ledger scale | 05-06 T2, 05-11 T2 | `TestFindingHistory_LatestPerFinding`; `BenchmarkRecordFindingLedger_LargeRepo` (results in 05-11-SUMMARY, not a default-suite test) |
| GAP-08 cascade gate | 05-03 T1, 05-07 T1-T2 | `TestCobolGrammar_Findings_Cascade*`, `TestCobolCascadeGate_*` |
| Deferred P3 | 05-10 T1-T2, 05-12 T3 (D-20 docs) | `TestCobolGrammar_(CopybookAnalysisFailed\|ProgramAnalysisFailedStillFails\|CursorAmbiguous\|NestedContainmentUnresolved)`, `TestCobolReconcile_ContainmentUnresolved`; D-20 limitation sentence grepped in `docs/languages.md` |
| Evict capability (Wave 0 storetest) | 05-02 T1-T2 | `TestEvictFindingNodesByIDs_*`, storetest `FindingNodeEvict`; 05-02 T2 verify runs only its own fence case (`TestGenerationScopedWriterFamilies/finding_node_evict`), the checklist and fence-name tests, and the `FindingNodeEvict` conformance subtest; the full `Generation\|Conformance` subset runs in 05-02 plan verification |
| D-16 writer inventory | 05-12 T3 | `TestFindingWriters_Inventory` |
| Regression (Phase 2-4 suites) | 05-01, 05-03 T2, 05-05 T3, 05-08 T2 (edited subsets); plan verification of 05-01, 05-03, 05-05, 05-08, 05-09, 05-11, 05-12 (full suites, `-timeout 30m`); 05-06 plan verification runs the whole `store_sqlite` package (`-timeout 30m`) | `GOWORK=off go test -race -timeout 30m ./internal/indexer ./cmd/gortex -run '^TestCobol'`; full `GOWORK=off go test -race ./...` at 05-12 plan verification |

### Per-task latency rule (revision 1, extended in revisions 2 and 3)

Per-task `<automated>` verifies run only the tests the task creates or edits plus the directly affected subset, so each stays inside the 120 s feedback budget. The long real-path suites (`^TestCobolLifecycle_`, `^TestCobolFindings_` in `internal/indexer`, `^TestCobol` in `cmd/gortex`, the cascade gate as a whole) run in each plan's `<verification>` block at the wave merge with `-timeout 30m`, the whole `internal/graph/store_sqlite` package (about 680-725 s with `-race`, measured in Phase 4) runs only in 05-06's plan verification with `-timeout 30m` (its Task 3 verify runs the registry-sensitive subset, about 28 s), and the full `go test -race ./...` runs once, as 05-12's plan verification (the phase gate). Revision 3: 05-02 Task 2's earlier `-run 'EvictFindingNodesByIDs|Generation|Conformance'` measured 148 s with `-race` (94 tests); its verify now selects the new kind-guard test, the `finding_node_evict` and `config_node_evict` fence cases, `TestGenerationCapabilityChecklistIsComplete`, `TestGenerationWriteFenceNamesAreReal`, and the SQLite `FindingNodeEvict` conformance subtest (the same shape measured 4 s with existing names), plus the `internal/graph/storetest` package (about 1 s); the full subset moved to 05-02's plan verification with `-timeout 30m`.

---

## Wave 0 Requirements

- [ ] `internal/graph/finding_test.go` — ID determinism, evidence normalization, ordinals, class separation
- [ ] `internal/graph/finding_ledger_test.go` — open/close/supersede pairing, container and class isolation, reopen
- [ ] `internal/graph/storetest` case for `EvictFindingNodesByIDs` (both backends)
- [ ] `internal/graph/store_sqlite/store_finding_history_test.go` + update `schema_sidecar_view_gen_test.go`
- [ ] `internal/parser/languages/cobol_grammar_findings_test.go` (fixtures from the research probe table)
- [ ] `internal/resolver/cobol_findings_test.go`
- [ ] `internal/indexer/cobol_findings_lifecycle_test.go`, `internal/indexer/cobol_cascade_gate_test.go` (reuse `newCobolLifecycleEnv`)
- [ ] Shadow-path tests (`TestCobolFindings_HistoryShadowPath` and the `shadow=on` cold-track subtests of `TestCobolFindings_HistoryDependentRepo` and `TestCobolFindings_CrossRepoDependentLedgerAndSearch`, each proving the shadow ran from the admissions counter) — Phase 4 harness bypasses the shadow
- [ ] `internal/mcp/index_health_findings_test.go` (pattern: `index_health_ownership_test.go`)
- [ ] `cmd/gortex/cobol_findings_test.go` (pattern: `TestCobolTracer_MCPGetSymbol` / `CLIGetSymbol`)
- [ ] `internal/neo4jprojection` finding Meta case (pattern: `TestProjectNode_CobolBreadthMeta`)
- [ ] Update pinned extractor-version assertions (`gortex-cobol-grammar/5`) and ~50 `missing`-edge-on-`unresolved::` assertions when placeholders land

Framework install: none.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
