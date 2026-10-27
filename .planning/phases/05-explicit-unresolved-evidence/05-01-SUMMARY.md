---
phase: 05-explicit-unresolved-evidence
plan: 01
subsystem: cobol-graph
tags: [cobol, finding, parse-unresolved, blake3, identity, mcp, sqlite]

requires:
  - phase: 04-stable-incremental-and-cross-repository-lifecycle
    provides: "gortex-cobol-grammar/5 extractor walk (cobolWalk, cobolStampRange), repository prefixing, SQLite AddBatch path, serveCobolTestDaemon"
provides:
  - "graph.KindFinding (\"finding\") registered in validNodeKinds"
  - "graph.FindingSymbol and graph.FindingEvidenceDigest (D-01 identity, gortex-finding-id/1, gortex-finding-evidence/1)"
  - "Finding class, status, rule, and Meta key constants in internal/graph/finding.go"
  - "cobolFindings(w): every producer PARSE_UNRESOLVED gap of a program document becomes a KindFinding node with a member_of edge"
  - "Acceptance tests: MCP get_symbol, CLI gortex call get_symbol, kind:finding search on a persisted finding"
affects: [05-02, 05-03, 05-04, 05-05, 05-06, 05-08, 05-09, 05-11, 05-12, phase-06]

actuals:
  tokens: 11637
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Finding identity: length-framed BLAKE3 over (rule, container, class, kind, family, evidence digest), first 16 bytes hex"
    - "Container ladder: innermost statement, then procedure membership container, then data/environment section (each by Parent walk), then owning program, then walk root, then single program or file node"
    - "Finding Meta = cloned docMeta + finding_* keys; Name is the gap kind, never source text"

key-files:
  created:
    - internal/graph/finding.go
    - internal/graph/finding_test.go
    - internal/parser/languages/cobol_grammar_findings.go
    - internal/parser/languages/cobol_grammar_findings_test.go
    - cmd/gortex/cobol_findings_test.go
  modified:
    - internal/graph/node.go
    - internal/parser/languages/cobol_grammar.go
    - internal/parser/languages/cobol_grammar_test.go
    - internal/parser/languages/cobol_grammar_contract_test.go
    - internal/parser/languages/cobol_grammar_structure_test.go

key-decisions:
  - "Container ladder runs three separate ancestor walks (statement, then membership container, then section) because w.container is set on every procedure observation; a per-level check would return the paragraph before reaching the enclosing statement"
  - "Tracer feedback gate: with workflow.human_verify_mode=end-of-phase and a fully automated <verify>, the tracer verify was re-run green after its commit and execution continued; the human look at the slice is deferred to end-of-phase UAT"
  - "Fixed-format evidence normalization stays private in cobol_grammar_findings.go (single caller); internal/graph exposes only FindingEvidenceDigest and FindingSymbol"

patterns-established:
  - "Finding writers call graph.FindingSymbol with the repo-relative container ID and append #k (k>=2) per identical symbol in generated-span order"
  - "Tests written before findings existed exclude KindFinding nodes and edges whose From is a finding via cobolFindingIDs"

requirements-completed: [GAP-01, GAP-03, GAP-04, GAP-06]

coverage:
  - id: D1
    description: "KindFinding is a registered node kind and FindingSymbol/FindingEvidenceDigest implement the D-01 identity (class segment, length framing, empty-evidence domain)"
    requirement: GAP-03
    verification:
      - kind: unit
        ref: "internal/graph/finding_test.go#TestFindingID_Deterministic"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_test.go#TestFindingID_ClassSeparation"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_test.go#TestFindingID_LengthFramed"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_test.go#TestFindingEvidenceDigest_Domain"
        status: pass
      - kind: unit
        ref: "internal/graph/parity_audit_test.go#TestSchemaParityAudit"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every producer PARSE_UNRESOLVED gap of a program document is one finding (per-token tails, error, missing, lost-capture, document-scope synthetic kinds); producer EXTERNAL_UNRESOLVED gaps are never findings"
    requirement: GAP-01
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_UnparsedTailPerToken"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_ErrorContainer"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_MissingZeroWidth"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_LostCapture"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_DocumentScopeSynthetic"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_NoProducerExternal"
        status: pass
    human_judgment: false
  - id: D3
    description: "A persisted, repository-prefixed unparsed-tail finding carries the GAP-04 Meta contract and is returned by MCP get_symbol, CLI gortex call get_symbol, and kind:finding search, with no source text in Name, ID, or Meta"
    requirement: GAP-04
    verification:
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_MCPGetSymbol"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_CLIGetSymbol"
        status: pass
      - kind: integration
        ref: "cmd/gortex/cobol_findings_test.go#TestCobolFindings_SearchByNameAndKind"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_FindingMetaContract"
        status: pass
    human_judgment: false
  - id: D4
    description: "Finding IDs survive line shifts; editing one gap's own text changes only that finding's ID"
    requirement: GAP-06
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_LineShiftKeepsIDs"
        status: pass
    human_judgment: false
  - id: D5
    description: "Tracer slice reviewed by a human (interactive tracer feedback gate)"
    verification:
      - kind: integration
        ref: "GOWORK=off go test -race -count=1 ./cmd/gortex/ -run '^TestCobolFindings_|^TestCobolTracer_'"
        status: pass
    human_judgment: true
    rationale: "The interactive tracer gate was deferred to end-of-phase UAT under workflow.human_verify_mode=end-of-phase; a human should confirm the finding payload shape from get_symbol is what Phase 6 consumers expect"

duration: 26min
completed: 2026-10-20
status: complete
---

# Phase 5 Plan 01: Parse Findings Tracer Summary

**Every producer PARSE_UNRESOLVED gap of a COBOL program now persists as a `finding` node with a line-free BLAKE3 ID (`<path>::@finding/parse/<32 hex>[#k]`), a member_of edge to its container, and the GAP-04 Meta contract, retrievable through MCP get_symbol, CLI get_symbol, and kind:finding search.**

## Performance

- **Duration:** ~26 min
- **Started:** 2026-10-20T15:33:00Z
- **Completed:** 2026-10-20T15:59:00Z
- **Tasks:** 2
- **Files modified:** 10 (5 created, 5 modified)

## Accomplishments

- `graph.KindFinding` registered; `internal/graph/finding.go` holds the class, status, rule, and Meta key constants plus `FindingSymbol` and `FindingEvidenceDigest` (length-framed BLAKE3, no line, offset, gap ID, or version hashed).
- `cobolFindings(w)` maps every producer parse gap (sorted by generated span) onto a finding: container ladder, fixed-format evidence normalization (columns 7-72, whitespace collapse), `#k` ordinals, verbatim severity/scope/absence reason, `member_of` edge. Producer `-INC` external gaps are skipped (D-17).
- Called in `Extract` right after `cobolSQL(w, data)`, before the document flags, so they never leak into finding Meta.
- Real-path acceptance: the tail finding on `ZQXCOL` reaches SQLite through `AddBatch` and is returned by MCP `get_symbol` (detail=full), `gortex call get_symbol`, and `search_symbols "kind:finding unparsed"`; neither its Name, ID, nor any Meta value carries the token.
- 12 new unit tests pin identity and the program-path mapping on the real producer; pre-existing empty-source, ProvKeys, and DegradedBreadth tests are finding-aware.

## Task Commits

1. **Task 1 (tracer): One program parse finding reaches SQLite and is returned by MCP get_symbol, CLI get_symbol, and search** - `76ad151d` (feat)
2. **Task 2: Pin the D-01 identity and the program-path gap mapping on the real producer** - `979d723b` (test)

**Plan metadata:** recorded in the docs(05-01) commit that adds this file.

## Files Created/Modified

- `internal/graph/node.go` - `KindFinding` declaration with ID-convention doc comment; `validNodeKinds` entry
- `internal/graph/finding.go` - finding constants, `FindingEvidenceDigest`, `FindingSymbol`, private `findingDigest`
- `internal/graph/finding_test.go` - determinism, class separation, length framing, digest domain
- `internal/parser/languages/cobol_grammar_findings.go` - `cobolFindings`, `cobolFindingContainer`, `cobolFindingEvidence`, `cobolFixedEvidenceText`
- `internal/parser/languages/cobol_grammar.go` - `cobolFindings(w)` call site after `cobolSQL`
- `internal/parser/languages/cobol_grammar_findings_test.go` - eight `TestCobolGrammar_Finding*` tests plus `cobolFindingNodes`, `cobolFindingIDs`, `cobolProducerGaps`, `assertCobolFindingMember`
- `internal/parser/languages/cobol_grammar_test.go` - `TestCobolGrammar_EmptySource` counts non-finding nodes
- `internal/parser/languages/cobol_grammar_structure_test.go` - empty-source tail of `TestCobolGrammar_Statements` counts non-finding nodes
- `internal/parser/languages/cobol_grammar_contract_test.go` - ProvKeys and DegradedBreadth skip findings and their member_of edges
- `cmd/gortex/cobol_findings_test.go` - `indexCobolFindingsRepo` fixture plus the three `TestCobolFindings_*` acceptance tests

## Decisions Made

- Container ladder is three separate ancestor walks (statement, membership container, section). `w.container` is set on every procedure-division observation, so checking all three per level would attach SQL tails to the paragraph instead of their `@exec_sql_statement`.
- Tracer feedback gate: config has `auto_advance=false` but `human_verify_mode=end-of-phase`, and the tracer `<verify>` is fully automated. The verify was re-run green after commit `76ad151d`, then Task 2 proceeded. The human look at the slice is deferred to end-of-phase UAT (coverage D5).
- Three test call sites needed a "finding ID set", so `cobolFindingIDs` was added (rule of three) in the findings test file.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pre-existing gofmt drift in internal/graph/node.go**
- **Found during:** Task 1 verify
- **Issue:** At HEAD, `node.go` misaligned the `Meta` and `RepoPrefix` fields of `Node`, so the verify step `test -z "$(gofmt -l ... internal/graph/node.go ...)"` failed even though the new code was formatted.
- **Fix:** `gofmt -w internal/graph/node.go`, as the plan's action instructs. Whitespace only.
- **Files modified:** internal/graph/node.go
- **Verification:** `gofmt -l` prints nothing; build and tests pass
- **Committed in:** 76ad151d

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Whitespace only. The wave-merge check (`./internal/indexer/ ./cmd/gortex/ -run '^TestCobol'`) passed with no count-only test fixes, so no other test files changed.

## TDD Gate Compliance

Task 2 (`tdd="true"`) pins behavior the tracer task had to ship first. Its `test(05-01)` commit therefore follows the `feat(05-01)` commit, and its tests passed on first run. No mapping bug surfaced. Task 1 did have a real RED run: the three `TestCobolFindings_*` tests failed with zero findings before `cobolFindings(w)` was wired into `Extract`.

## Issues Encountered

- The empty source carries no producer gaps today, so `TestCobolGrammar_EmptySource` and the `TestCobolGrammar_Statements` tail see zero findings. Their finding-aware assertions already admit the 05-03 `uncontained-cascade` finding without another edit.
- Lost-capture gaps for sibling data items share one evidence image in one container. Probe: two lost `data-item` captures in `LOSTPGM` got `<hex>` and `<hex>#2`. Their IDs depend on ordinal position, so inserting an earlier lost capture shifts later ordinals. This follows D-01 as written, but 05-06/05-09 supersede pairing should expect it.

## User Setup Required

None. No external service configuration is required.

## Next Phase Readiness

- 05-02 (finding eviction), 05-03 (copybook findings, cascade, `/6` bump), 05-04 (index_health counts), 05-05/05-08 (placeholders and external findings via `graph.FindingSymbol`) can build on this kind, ID function, and Meta contract.
- Concern for the ledger plans: lost-capture ordinals among sibling data items (see Issues Encountered).

## Self-Check: PASSED

- FOUND: internal/graph/finding.go, internal/graph/finding_test.go, internal/parser/languages/cobol_grammar_findings.go, internal/parser/languages/cobol_grammar_findings_test.go, cmd/gortex/cobol_findings_test.go
- FOUND commits: 76ad151d, 979d723b
- Task 1 and Task 2 verify commands passed; every acceptance criterion grep passed (1, 1, 2, 1, 1, 1; 0 and 2 for Task 2)
- Plan verification passed: `go test -race ./internal/graph/... ./internal/parser/languages ./cmd/gortex -run 'Finding|Cobol|Parity'` and `go test -race -timeout 30m ./internal/indexer/ ./cmd/gortex/ -run '^TestCobol'`

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-20*
