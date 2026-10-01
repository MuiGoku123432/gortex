---
phase: "3"
slug: "deterministic-cobol-and-mainframe-breadth"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-01"
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Derived from `03-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1 |
| **Config file** | none (COBOL grammar files carry `darwin \|\| linux` build tags; Windows stub per P2 D-17) |
| **Quick run command** | `go test -race ./internal/parser/languages/ -run 'TestCobolGrammar' -count=1` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~60 seconds quick; full suite several minutes |

Package runs: resolver `go test -race ./internal/resolver/ -run 'TestCobol' -count=1`; indexer (BOM) `go test -race ./internal/indexer/ -run 'TestCobol|TestStampSourceRevision' -count=1`; acceptance `go test -race ./cmd/gortex/ -run 'TestCobolTracer' -count=1`. Fork-side (copybook mode, D-16) tests run in `$COBOL_UPGRADE_ROOT` (default `~/repos/mine/devDeps/tree-sitter-cobol-upgrade`) with `go test -race ./preprocessor/...`.

---

## Sampling Rate

- **After every task commit:** Run the quick run command for the touched package
- **After every plan wave:** Run `go test -race ./internal/parser/languages/ ./internal/resolver/ ./internal/indexer/ ./cmd/gortex/ -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map is keyed by requirement and filled in per task during planning/execution.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | — | DET-01 (structure) | — | N/A | unit | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_Structure -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-01 (data items) | — | N/A | unit | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_DataItems -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-02 | — | N/A | unit | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_ControlFlow -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-02 (real path) | — | N/A | integration | `go test -race ./cmd/gortex/ -run TestCobolTracer_ParagraphFlow -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-03/04 | ID injection | literal-derived ID segments encoded | unit + resolver | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_Calls -count=1` ; `go test -race ./internal/resolver/ -run TestCobolReferences -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-05/06 | false cross-repo binding | same-repo exactly-one only | unit + resolver | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_Copy -count=1` ; `go test -race ./internal/resolver/ -run TestCobolReferences_Copybook -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-07/08 | — | N/A | unit | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_IDMS -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-09/10 | — | N/A | unit | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_CICS -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | DET-11/12 | — | no physical table invented | unit | `go test -race ./internal/parser/languages/ -run TestCobolGrammar_SQL -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | CR-01 (PROV-03) | — | N/A | integration | `go test -race ./internal/indexer/ -run TestCobolBOMByteOffsets -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | D-16 copybook mode | — | N/A | unit (fork) + unit | fork `go test -race ./preprocessor/ -run Copybook` ; `go test -race ./internal/parser/languages/ -run TestCobolGrammar_Copybook -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | cross-cutting (degraded, version `/3`, prov keys) | secret-like keys | no meta key contains secret-like substrings | unit | `go test -race ./internal/parser/languages/ -run 'TestCobolGrammar_(Degraded|ProvKeys)' -count=1` | partial | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Shared invented COBOL fixtures (multi-construct program; sections + nested programs; IDMS; CICS; SQL; BOM variant; standalone copybook) as test constants in `internal/parser/languages/cobol_grammar_test.go` or a sibling `_test.go` — invented only
- [ ] `internal/resolver/cobol_references_test.go` — in-memory graph with two repos, duplicate copybooks, nested same-file programs
- [ ] Extend `cmd/gortex/cobol_tracer_test.go` with paragraph-flow and call-binding assertions on the real path
- Framework install: none — existing infrastructure

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Fork copybook-mode commit pushed to origin | D-16 | Outward action needing user confirmation | Confirm at the checkpoint; then verify `git -C "$COBOL_UPGRADE_ROOT" ls-remote origin` matches local HEAD |
| Aggregate-only corpus sanity (optional) | D-05 | Private estate not reachable from CI; privacy rule | If `PREPROCESSOR_CORPUS_ROOT` is set locally, run the probe and record aggregate counts only |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
