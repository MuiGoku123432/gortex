---
phase: "4"
slug: "stable-incremental-and-cross-repository-lifecycle"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-12"
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1 + zap observer |
| **Config file** | none (build tag `darwin \|\| linux` on COBOL tests) |
| **Quick run command** | `GOWORK=off go test -race -count=1 ./internal/resolver -run 'Cobol\|CrossRepo'` |
| **Full suite command** | `GOWORK=off go test -race ./...` |
| **Estimated runtime** | ~60 seconds quick-per-package; full suite several minutes |

---

## Sampling Rate

- **After every task commit:** Run the quick command plus `GOWORK=off go test -race -count=1 <package under edit>`
- **After every plan wave:** Run `GOWORK=off go test -race -count=1 ./internal/resolver ./internal/indexer ./internal/graph/... ./internal/parser/languages ./cmd/gortex -run 'Cobol|CrossRepo|PrefixDiagnostics|Scope|AffectedBy'`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

Requirement → test map (task IDs are filled in by the planner/executor; see 04-RESEARCH.md §Validation Architecture):

| Requirement | Behavior | Test Type | Automated Command | File Exists | Status |
|-------------|----------|-----------|-------------------|-------------|--------|
| ID-03 | Program ID component encoding; no aliasing | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_ProgramIDEncoding'` | ❌ W0 | ⬜ pending |
| ID-04 | Shared resource writer-independent identity; alias edge; GC | unit + real path | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_SharedResource'` | ❌ | ⬜ pending |
| ID-05 | Distinct occurrence provenance | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_Occurrences'` | ❌ | ⬜ pending |
| ID-06 | Clean×2 and order-swapped digests equal | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolLifecycle_CleanDeterminism'` | ❌ W0 | ⬜ pending |
| ID-07 | Edit/delete/restore == clean | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolLifecycle_(Edit\|Delete\|Restore)'` | ❌ | ⬜ pending |
| ID-08 | Version/config changes restage unchanged bytes | real path + unit | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolLifecycle_(Version\|Config)\|TestCobolMaterialFingerprint'` | ❌ | ⬜ pending |
| ID-09 | All D-04 sequences == clean | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolLifecycle_'` | ❌ | ⬜ pending |
| SEC-01 | Same allow-set → same node set across surfaces | integration | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolScope_CrossSurface'` | ❌ | ⬜ pending |
| SEC-02 | Unbound multi-repo source read/write refused (D-13/D-20) | integration | `GOWORK=off go test -race -count=1 ./internal/mcp -run 'TestSourceScope_'` | ❌ | ⬜ pending |
| SEC-03 | No cross-repo bind without allowlist; union ambiguity; generic refuse | unit | `GOWORK=off go test -race -count=1 ./internal/resolver -run 'TestCobolReconcile_Allowlist\|TestCrossRepoResolveAll_Ambiguous'` | ❌ | ⬜ pending |
| SEC-08 | Zero unowned/misprefixed; health status | unit + real path | `GOWORK=off go test -race -count=1 ./internal/graph/... -run 'PrefixDiagnostics'` | ⚠️ update | ⬜ pending |
| D-08 | Over-cap escalates | unit | `GOWORK=off go test -race -count=1 ./internal/indexer -run 'TestAffectedBy_Cap'` | ⚠️ rewrite | ⬜ pending |
| Regression | Phase 2/3 suites green | real path | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Tracer\|Breadth)_'` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/testutil/graphdump/graphdump.go` + `graphdump_test.go` — Dump, Volatile allowlist, FirstDiff, BLAKE3 digest
- [ ] `internal/indexer/cobol_lifecycle_test.go` — invented two-repo fixtures, >cap copybook generator, clean-determinism test (RED until CR-02 fix)
- [ ] Spike: interrupted reindex produces real partial state then full retry
- [ ] Spike: confirm which tail sites a COBOL save reaches
- [ ] Update pinned extractor-version assertions when the extractor bumps

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
