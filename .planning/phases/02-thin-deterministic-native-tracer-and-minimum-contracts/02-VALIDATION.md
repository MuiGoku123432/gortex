---
phase: "02"
slug: "thin-deterministic-native-tracer-and-minimum-contracts"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-30"
---

# Phase 02 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1 |
| **Config file** | none (package tests) |
| **Quick run command** | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run 'TestCobolGrammar'` |
| **Full suite command** | `GOWORK=off go test -race ./...` |
| **Estimated runtime** | ~30 seconds quick; several minutes full |

---

## Sampling Rate

- **After every task commit:** Run the quick command plus the touched package
- **After every plan wave:** Run `GOWORK=off go test -race -count=1 ./internal/parser/... ./internal/indexer/ ./internal/neo4jprojection/ ./cmd/gortex/ -run 'Cobol|SourceRevision|ProjectNode'`
- **Before `/gsd-verify-work`:** `GOWORK=off go build -o gortex ./cmd/gortex/ && GOWORK=off go test -race ./...` green, `GOWORK=off go mod tidy` diff-clean, stock-grammar negative step fails as expected
- **Max feedback latency:** 60 seconds (quick)

---

## Per-Task Verification Map

Requirement → test map is authoritative in `02-RESEARCH.md` § Validation Architecture → "Phase Requirements → Test Map". The planner assigns task IDs; this table is filled during `/gsd-validate-phase`.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | BASE-01..03, TRACE-01..07, PROV-01..06, ID-01..02 | see PLAN threat_model | see PLAN | unit / integration / CI step | see RESEARCH test map | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Fork: `NewEmbeddedAnalyzer` + `AttestEmbedded` + tests, pushed (producer prerequisite)
- [ ] SQLite-backed `spinUpDaemonWithConfig` variant + AddBatch-recording wrapper
- [ ] `internal/parser/languages/cobol_grammar_test.go` with invented fixtures
- [ ] `cmd/gortex/cobol_tracer_test.go`
- [ ] `internal/indexer/source_revision_test.go` (temp `git init` repo)
- [ ] `internal/neo4jprojection` provenance projection test

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CI secret `COBOL_PARSER_READ_PAT` provisioned (fine-grained, single repo, Contents read-only) | D-13 | Requires GitHub account action | Create token, add as repo secret, confirm CI green on all three OS shards |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
