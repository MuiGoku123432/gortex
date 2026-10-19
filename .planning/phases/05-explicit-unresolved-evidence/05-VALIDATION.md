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
- **After every plan wave:** Run `GOWORK=off go test -race -count=1 ./internal/graph/... ./internal/resolver ./internal/indexer ./internal/parser/languages ./internal/mcp ./internal/neo4jprojection ./cmd/gortex -run 'Finding|Cobol|PrefixDiagnostics|IndexHealth|ProjectNode|SidecarViewGen|Migration'`
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
| GAP-06 | identical rerun and line shift keep IDs; clean x2 and opposite track orders equal | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobol(Lifecycle\|Findings)_'` | ⚠️ partial (harness exists) | ⬜ pending |
| GAP-07 | fix closes; text edit supersedes with successor; retrieve closes EXTERNAL only; parser re-stamp keeps ID; restore reopens; history survives restart and reindex, cleared by purge | unit + real path | `GOWORK=off go test -race -count=1 ./internal/graph -run TestFindingLedger ; GOWORK=off go test -race -count=1 ./internal/graph/store_sqlite -run TestFindingHistory ; GOWORK=off go test -race -count=1 ./internal/indexer -run TestCobolFindings_History` | ❌ W0 | ⬜ pending |
| GAP-07 | ledger rows land in SQLite on the shadow path | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run TestCobolFindings_HistoryShadowPath` (set `GORTEX_SHADOW_MAX_FILES` above corpus size) | ❌ W0 | ⬜ pending |
| GAP-08 | positive shapes produce `uncontained-cascade`; negatives keep 20 paragraphs and 20 data items; no flag | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolCascadeGate'` | ❌ W0 | ⬜ pending |
| Deferred P3 | WR-06 identity-only copybook + `analysis-failed`; WR-07 ambiguous cursor; IN-08 distinct reason; WR-09 still green | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_(CopybookAnalysisFailed\|CursorAmbiguous\|NestedContainmentUnresolved\|CopybookProcedureMainframe)' ; GOWORK=off go test -race -count=1 ./internal/resolver -run TestCobolReconcile_ContainmentUnresolved` | ❌ W0 / ✅ WR-09 | ⬜ pending |
| Regression | Phase 2-4 COBOL suites with updated expectations | real path | `GOWORK=off go test -race -count=1 ./cmd/gortex ./internal/indexer ./internal/resolver -run 'Cobol'` | ✅ (update) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/graph/finding_test.go` — ID determinism, evidence normalization, ordinals, class separation
- [ ] `internal/graph/finding_ledger_test.go` — open/close/supersede pairing, container and class isolation, reopen
- [ ] `internal/graph/storetest` case for `EvictFindingNodesByIDs` (both backends)
- [ ] `internal/graph/store_sqlite/store_finding_history_test.go` + update `schema_sidecar_view_gen_test.go`
- [ ] `internal/parser/languages/cobol_grammar_findings_test.go` (fixtures from the research probe table)
- [ ] `internal/resolver/cobol_findings_test.go`
- [ ] `internal/indexer/cobol_findings_lifecycle_test.go`, `internal/indexer/cobol_cascade_gate_test.go` (reuse `newCobolLifecycleEnv`)
- [ ] Shadow-path history test (`TestCobolFindings_HistoryShadowPath`) — Phase 4 harness bypasses the shadow
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
