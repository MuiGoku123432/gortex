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
| 03-01-T1 | 03-01 | 1 | DET-01, DET-02 (tracer, real path) | T-03-03 | labels bind only on exactly one same-program match | integration | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolTracer_'` (includes `TestCobolTracer_ParagraphFlow`) | ❌ W0 | ⬜ pending |
| 03-01-T2 | 03-01 | 1 | DET-01 (divisions, sections, paragraphs), DET-02 (PERFORM) | T-03-01 | name-derived ID segments percent-encoded | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_(Divisions|Paragraphs|PerformEdge|EdgeDedupe|NameNorm)$'` | ❌ W0 | ⬜ pending |
| 03-02-T1 | 03-02 | 1 | CR-01 (PROV-03 residual), DET-01 | T-03-06 | unequal indexed bytes never claimed exact | unit | `GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestSourceRevision'` | ❌ W0 | ⬜ pending |
| 03-02-T2 | 03-02 | 1 | CR-01 (real extractor) | T-03-06 | N/A | integration | `GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestCobolBOMByteOffsets$'` | ❌ W0 | ⬜ pending |
| 03-03-T1 | 03-03 | 2 | DET-01 (sections, statements) | T-03-09 | no statement text in Meta | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_(Structure|Statements)$'` | ❌ W0 | ⬜ pending |
| 03-03-T2 | 03-03 | 2 | DET-01 (statement occurrence on edges) | — | N/A | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_StatementSymbolOnEdges$'` | ❌ W0 | ⬜ pending |
| 03-04-T1 | 03-04 | 2 | DET-03, DET-04 | T-03-13 | generic resolver passes never bind cobol:: targets | unit | `GOWORK=off go test -race -count=1 ./internal/resolver/ -run '^TestCobolNamespaceExclusions$'` | ❌ W0 | ⬜ pending |
| 03-04-T2 | 03-04 | 2 | DET-06 (+DET-03) | T-03-12 | same-repo exactly-one binding only | unit | `GOWORK=off go test -race -count=1 ./internal/resolver/ -run '^TestCobolReferences_'` ; `GOWORK=off go test -race -count=1 ./internal/indexer/ -run '^TestReuseIdentifierSkipsCobolNamespace$'` | ❌ W0 | ⬜ pending |
| 03-05-T1 | 03-05 | 3 | DET-01 (data items) | T-03-16 | data-item ID segments encoded | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_DataItems$'` | ❌ W0 | ⬜ pending |
| 03-05-T2 | 03-05 | 3 | DET-01 (REDEFINES, RENAMES, COPY in group) | T-03-18 | N/A | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_Data(References|ContainsCopy)$'` | ❌ W0 | ⬜ pending |
| 03-06-T1, T2 | 03-06 | 3 | DET-02 | T-03-19 | no partial THRU expansion, no cross-program labels | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_ControlFlow$'` | ❌ W0 | ⬜ pending |
| 03-07-T1 | 03-07 | 4 | DET-03, DET-04, DET-05 | T-03-22 | literal-derived ID segments encoded | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_(Calls|Copy|CopybookKey)$'` | ❌ W0 | ⬜ pending |
| 03-07-T2 | 03-07 | 4 | DET-05, DET-06 (real path) | T-03-23 | owned namespaces only | integration | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolTracer_'` (includes `TestCobolTracer_CallAndCopyBinding`) | ❌ W0 | ⬜ pending |
| 03-08-T1 | 03-08 | 5 | DET-07, DET-08 | T-03-25 | qualifiers only from source | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_IDMS$'` | ❌ W0 | ⬜ pending |
| 03-08-T2 | 03-08 | 5 | DET-09, DET-10 | T-03-26 | data-named operands never bound | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_CICS$'` | ❌ W0 | ⬜ pending |
| 03-09-T1, T2 | 03-09 | 6 | DET-11, DET-12 | T-03-29 | no physical table invented | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_SQL$'` | ❌ W0 | ⬜ pending |
| 03-10-T1 | 03-10 | 7 | DET-01..DET-12 (real path, two repos) | T-03-33 | no cross-repo binding by name | integration | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Breadth|Tracer)_'` | ❌ W0 | ⬜ pending |
| 03-10-T2 | 03-10 | 7 | cross-cutting (prov keys, value types, degraded, version `/3`) | T-03-32 | no Meta key contains a secret-like substring | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobolGrammar_(ProvKeys|DegradedBreadth)$'` ; `GOWORK=off go test -race -count=1 ./internal/neo4jprojection/ -run '^TestProjectNode_Cobol'` | partial | ⬜ pending |
| 03-11-T1 | 03-11 | 8 | D-16 copybook mode (fork transform and grade) | T-03-34 | member coordinates kept; program mode unchanged | unit (fork) | fork: `go test -race -count=1 ./preprocessor/grade/ ./preprocessor/transform/ ./preprocessor/handoff/` | ❌ W0 | ⬜ pending |
| 03-11-T2 | 03-11 | 8 | D-16 copybook mode (fork analyzer) | T-03-36 | invented members only | unit (fork) | fork: `go test -race -count=1 ./preprocessor/ -run '^(TestTracerAnalyzeDocument|TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer|TestNewContentIDMatchesHandoff|TestAnalyzeCopybookDataMember|TestAnalyzeCopybookProcedureMember|TestAnalyzeCopybookReplacingDegrades)$'` | ❌ W0 | ⬜ pending |
| 03-12-T1, T2 | 03-12 | 9 | D-16 push | T-03-38 | user-confirmed fast-forward only | manual decision + check | `git -C "$FORK" ls-remote origin refs/heads/gsd/v0.26.0-estate-parse-recovery` equals HEAD with parent f9eaf99 | n/a | ⬜ pending |
| 03-13-T1 | 03-13 | 10 | D-16 re-pin, DET-01 (copybook-owned data items, real path) | T-03-41 | copybooks pass the approved-grammar check | integration | re-pin `go list -m` checks ; `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Tracer|Breadth)_'` | ❌ W0 | ⬜ pending |
| 03-13-T2 | 03-13 | 10 | D-16 copybook unit tests, phase gate | T-03-43 | wrapper never surfaces as nodes | unit + full | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run '^TestCobol'` ; `GOWORK=off go test -race -count=1 ./...` | partial | ⬜ pending |

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
| Fork copybook-mode commit pushed to origin | D-16 | Outward action needing user confirmation | Plan 03-12 Task 1 `checkpoint:decision` ("push" or "hold"); after "push", Task 2 verifies `git -C "$COBOL_UPGRADE_ROOT" ls-remote origin` matches local HEAD with parent f9eaf99 |
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
