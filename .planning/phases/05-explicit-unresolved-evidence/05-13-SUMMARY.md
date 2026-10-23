---
phase: 05-explicit-unresolved-evidence
plan: 13
subsystem: cobol-findings
tags: [cobol, finding, identity, ledger, blake3, gap-closure, d-21, d-22, d-25, d-26]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "parse findings (05-01), external findings and placeholders (05-05, 05-08), finding ledger (05-06, 05-09), resource evicter (05-15)"
provides:
  - "cobolFindingAnchor: name-stable parse finding anchor (paragraph, section, procedure division, data or environment section, program, else file); document scope always the file node"
  - "cobolAddFinding hashes the anchor into the ID and finding_container (ledger pairing key); member_of keeps the innermost statement"
  - "graph.FindingIDRule gortex-finding-id/2 and graph.FindingEvidenceRule gortex-finding-evidence/2 (each bumped once)"
  - "Corrected FindingEvidenceDigest doc: unkeyed BLAKE3 on purpose, short evidence dictionary-confirmable (D-22 code half)"
  - "External evidence = fixed six fields: edge kind, prov_observation_kind, cics_command, call or copy form, COPY IDMS kind, as-written target (D-25)"
  - "Per-occurrence external findings from cobol_occurrence_count; occurrences 2..n carry prov_range_absence same_line_occurrence and no coordinates (D-26)"
  - "The three VERIFICATION regressions at extractor level and on the real path (MultiIndexer, SQLite, ledger)"
affects: [05-14, 05-16, 05-17]

actuals:
  tokens: 16100
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Finding identity anchors are name-stable; statements are member_of targets only"
    - "External finding evidence has fixed arity; an absent field is the empty string"

key-files:
  created:
    - internal/indexer/cobol_findings_identity_test.go
  modified:
    - internal/graph/finding.go
    - internal/parser/languages/cobol_grammar_findings.go
    - internal/parser/languages/cobol_grammar_findings_test.go
    - internal/resolver/cobol_findings.go
    - internal/resolver/cobol_findings_test.go

key-decisions:
  - "D-21 shipped: parse finding ID and finding_container hash cobolFindingAnchor; document-scope gaps (c < 0, out of range, or uncontained) anchor to the file node with no single-program fallback"
  - "D-01 adjacency trade as shipped: with the paragraph anchor, inserting or deleting a statement that leaves a gap with identical evidence text earlier in the same container shifts the #k ordinal of a later identical-evidence finding and changes its ID; D-01 accepts this, and 05-17 qualifies its user-docs stability sentence from this note"
  - "D-26 mixed-form window: cobolDedupeEdges keys on (From, To, Kind, Line), so CALL literal, EXEC CICS LINK PROGRAM, and EXEC CICS XCTL PROGRAM of one target on one line (all calls edges from the flow source) collapse into one edge with the first occurrence's Meta; all n findings carry the first form's evidence and editing a later form on that line changes no ID. EXEC CICS LOAD (references) and COPY-family statements (imports) collapse only within their own edge kind"
  - "Rule bumps: FindingIDRule and FindingEvidenceRule each bumped exactly once in this plan; extractor version stays gortex-cobol-grammar/6 (Phase 5 convention)"
  - "Tracer gate did not halt: auto mode is off but workflow.human_verify_mode is end-of-phase and mode is yolo; the tracer verify re-ran end-to-end and passed before Task 2"

patterns-established:
  - "Identity regressions are written against unrelated same-kind edits and program-count changes, not only comment-line shifts"

requirements-completed: [GAP-02, GAP-04, GAP-06, GAP-07]

coverage:
  - id: D1
    description: "Same-kind EXEC SQL inserted above existing gaps keeps every pre-existing parse finding ID and the ledger only opens the inserted statement's findings (VERIFICATION regression 1)"
    requirement: GAP-06
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_SameKindInsertKeepsIDs"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_identity_test.go#TestCobolFindings_SameKindInsertKeepsIDs"
        status: pass
    human_judgment: false
  - id: D2
    description: "Deleting the ZQXCOL EXEC SQL closes only its tail (closed, no successor); ZQXOTH and ERROR keep IDs, no superseded row (VERIFICATION regression 2)"
    requirement: GAP-06
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_SameKindDeleteKeepsIDs"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_identity_test.go#TestCobolFindings_SameKindDeleteClosesOnlyItsOwn"
        status: pass
    human_judgment: false
  - id: D3
    description: "Document-scope findings anchor to the file node for zero, one, or two programs; adding and removing a second program keeps the uncontained-cascade ID and writes no row (VERIFICATION regression 3, WR-07)"
    requirement: GAP-06
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_DocumentScopeAnchorsToFile"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_identity_test.go#TestCobolFindings_SecondProgramKeepsDocumentFinding"
        status: pass
    human_judgment: false
  - id: D4
    description: "CALL, CICS LINK, LOAD, and XCTL of one target are four unsuffixed findings; COPY and -INC differ; an XCTL edited to LINK supersedes only its finding (D-25)"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalReferenceFormEvidence"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_identity_test.go#TestCobolFindings_ReferenceFormEditSupersedes"
        status: pass
    human_judgment: false
  - id: D5
    description: "A collapsed edge with cobol_occurrence_count n yields base..base#n; later occurrences carry same_line_occurrence and no coordinates; unchanged rerun writes nothing; removing the second CALL closes only base#2 (D-26)"
    requirement: GAP-02
    verification:
      - kind: unit
        ref: "internal/resolver/cobol_findings_test.go#TestCobolReconcile_ExternalSameLineOccurrences"
        status: pass
      - kind: integration
        ref: "internal/indexer/cobol_findings_identity_test.go#TestCobolFindings_SameLineCallsPerOccurrence"
        status: pass
    human_judgment: false
  - id: D6
    description: "GAP-04 edge probes: identical evidence in one anchor separates into base and base#2, two anchors keep two unsuffixed IDs; zero-width MISSING digest is FindingEvidenceDigest(\"\")"
    requirement: GAP-04
    verification:
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_SameEvidenceAnchors"
        status: pass
      - kind: unit
        ref: "internal/parser/languages/cobol_grammar_findings_test.go#TestCobolGrammar_Findings_MissingZeroWidth"
        status: pass
    human_judgment: false
  - id: D7
    description: "FindingEvidenceDigest doc no longer claims source cannot escape; states unkeyed BLAKE3 by design and dictionary-confirmable short evidence (D-22 code half)"
    verification: []
    human_judgment: true
    rationale: "Doc wording is judged by review; the acceptance greps (dictionary, unkeyed, no 'never leaks') passed but cannot judge accuracy"

duration: 30min
completed: 2026-10-23
status: complete
---

# Phase 5 Plan 13: Name-Stable Finding Identity Summary

**Parse finding IDs and the ledger pairing key now hash a name-stable anchor (paragraph/section/program/file, never a statement ordinal); external findings carry the reference form and one finding per same-line occurrence; both rule constants bumped once.**

## Performance

- **Duration:** 30 min
- **Started:** 2026-10-23T15:25:23Z
- **Completed:** 2026-10-23T15:55:32Z
- **Tasks:** 3 (plus one optional plan-checker test)
- **Files modified:** 6 (1 created)

## Accomplishments

- `cobolFindingAnchor` replaces the statement-first ladder (D-21). The ID and `finding_container` hash the anchor; `member_of` targets the innermost statement node when one exists, else the anchor. The `len(w.programs) == 1` document-scope fallback is gone (WR-07).
- `FindingIDRule` = `gortex-finding-id/2`, `FindingEvidenceRule` = `gortex-finding-evidence/2`, each bumped once. Every parse and external finding writes them via the constants.
- `FindingEvidenceDigest` doc corrected (D-22 code half): no "never leaks" claim; unkeyed BLAKE3 for reproducible IDs; single-token evidence is dictionary-confirmable.
- External evidence is the fixed six-field NUL-joined list (D-25). New resolver constants `metaCICSCommand`, `metaObservationKind`, `metaCobolOccurrences`. No extractor change.
- `reconcileCobolFindings` expands each desired edge into `cobol_occurrence_count` occurrences, sorted after ID, start, Line, edge Kind by occurrence index (D-26, D-04).
- All three VERIFICATION regressions exist at extractor level and on the real path, and each was shown failing on the pre-change code.

## Task Commits

1. **Task 1 (tracer): name-stable parse finding anchor (D-21, D-22)** - `6fe6f0bc` (fix)
2. **Task 2: reference-form evidence and per-occurrence external findings (D-25, D-26)** - `79155e82` (fix)
3. **Task 3: real-path identity regressions** - `8688c750` (test)
4. **Optional plan-checker test: same evidence under one vs two anchors** - `e5563500` (test)

## RED and Mutation Runs

| Test | Run before production change (or mutation) | Failure |
|------|--------------------------------------------|---------|
| `TestCobolGrammar_Findings_SameKindInsertKeepsIDs` | RED, pre-change | 2 base IDs missing; 5 ledger rows incl. 2 false `superseded` (ZQXCOL `@exec_sql_statement` -> AAA tail, ZQXOTH `#2` -> re-IDed ZQXCOL) |
| `TestCobolGrammar_Findings_SameKindDeleteKeepsIDs` | RED, pre-change | ZQXOTH re-IDed; 3 rows, incl. false `superseded` |
| `TestCobolGrammar_Findings_DocumentScopeAnchorsToFile` | RED, pre-change | container `src/syn.cbl::SYNPGM`; one-program vs no-program IDs differ; cascade ID changed on second program (`::T` -> file) |
| `TestCobolGrammar_Findings_DocumentScopeSynthetic` (updated) | RED, pre-change | container `src/syn.cbl::SYNPGM`, not the file |
| `TestCobolFindings_SameKindInsertKeepsIDs` | RED, pre-change (real path) | 2 stored IDs missing; history rows `active -> superseded` for pre-existing IDs |
| `TestCobolReconcile_ExternalReferenceFormEvidence` | RED, pre-change | IDs `...#2`, `#3`, `#4` for the four forms (one evidence) |
| `TestCobolReconcile_ExternalSameLineOccurrences` | RED, pre-change | 1 finding instead of 3 |
| `TestCobolReconcile_External{MissingCall,MissingCopyAndInc,IDMS,OrdinalAndLineShift,ContainerSeparation,MetaContract}` (helper moved to six fields) | RED, pre-change | expected six-field IDs absent |
| `TestCobolFindings_SameKindDeleteClosesOnlyItsOwn` | Mutation A (statement anchor first + single-program fallback restored) | ZQXOTH missing; 3 rows, one false `superseded` |
| `TestCobolFindings_SecondProgramKeepsDocumentFinding` | Mutation A | container `cobol/t.cbl::T`; ID changed; `active -> closed` and `closed -> active` rows for the cascade |
| `TestCobolFindings_ReferenceFormEditSupersedes` | Mutation B (pre-D-25 evidence `form, raw`) | XCTL finding is `...#2`; LINK edit keeps the ID |
| `TestCobolFindings_SameLineCallsPerOccurrence` | Mutation C (occurrence count forced to 1) | 1 external finding instead of 2 |
| `TestCobolGrammar_Findings_SameEvidenceAnchors` | Mutation A (in-range statement anchor) | `one_paragraph`: second tail unsuffixed in `@exec_sql_statement#2` instead of `base#2` |

Every mutation was reverted with `git checkout -- <file>` on the committed file and re-verified (`MUTATION` marker count 0).

## Files Created/Modified

- `internal/graph/finding.go` - rule constants /2, `FindingSymbol` anchor doc, `FindingEvidenceDigest` D-22 doc.
- `internal/parser/languages/cobol_grammar_findings.go` - `cobolFindingAnchor`, anchor vs member_of split in `cobolAddFinding`, docs; `cobolFindingEvidence` doc names `graph.FindingEvidenceRule`.
- `internal/parser/languages/cobol_grammar_findings_test.go` - `cobolFindingRows`, `cobolFindingTail`, `cobolFindingSecondProgram`; updated `assertCobolFindingMember`, `DocumentScopeSynthetic`, `CascadePositive`; new `SameKindInsertKeepsIDs`, `SameKindDeleteKeepsIDs`, `DocumentScopeAnchorsToFile`, `SameEvidenceAnchors`.
- `internal/resolver/cobol_findings.go` - six-field evidence, occurrence expansion, docs.
- `internal/resolver/cobol_findings_test.go` - `cobolExternalID(..., [6]string)`, `cobolCallEvidence`; corrected `ExternalMissingCall` comment; `ExternalMetaContract` pins the six-field digest of its exact edge; new `ExternalReferenceFormEvidence`, `ExternalSameLineOccurrences`.
- `internal/indexer/cobol_findings_identity_test.go` (new) - `cobolIdentityFixture`, `cobolTransitions`, and the five real-path tests.

## Decisions Made

- See key-decisions. Fixture for the second program (both levels): `IDENTIFICATION DIVISION. / PROGRAM-ID. ZQXSECND. / PROCEDURE DIVISION. / STOP RUN.` appended with no `END PROGRAM`; the producer sees two programs and `cobolCascadeShapes[0]` (obtain in working storage) keeps its cascade grade, so no alternate shape was needed.
- Stored `finding_container` is repository-relative without the repo prefix (`cobol/t.cbl`), while node IDs carry it (`fsec/cobol/t.cbl::...`).
- Developer stores indexed between 05-01 and this plan keep old parse finding IDs for unchanged files until re-extracted (touch or clean re-index), because the extractor version stays `/6`; external findings re-ID at the next reconcile. No production store holds Phase 5 findings yet.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] ReferenceFormEvidence edit step re-seeds CALL and LINK**
- **Found during:** Task 2
- **Issue:** `RemoveEdge(from, to, kind)` deletes every matching edge on SQLite (first match on memory), so the line-4 XCTL `calls` edge cannot be removed alone.
- **Fix:** remove all `calls` edges of the paragraph, re-seed CALL and LINK unchanged, add the LOAD on line 4 (byte 400, after the original LOAD at 300); evictions and additions asserted through `CobolFindingChanges`.
- **Committed in:** `79155e82`

**2. [Plan-checker optional] Added `TestCobolGrammar_Findings_SameEvidenceAnchors`**
- Same tail token in two paragraphs -> two unsuffixed IDs; in one paragraph -> base and base#2. Witness of the adjacency trade. Committed in `e5563500`.

**Total deviations:** 1 auto-fixed (test mechanics), 1 optional test added. **Impact:** none on production scope.

## Issues Encountered

- Disk: free space on `/System/Volumes/Data` fell from 16 GiB at start to 5.2 GiB after the `-race` indexer and cmd/gortex suites (Go build/test cache). Per instructions nothing was deleted and no further heavy runs were started after verification; flag for the orchestrator before 05-14.

## Verification

- Task 1, 2, 3 verify commands: pass.
- `go test -race -count=1 ./internal/graph/ ./internal/parser/languages/ ./internal/resolver/ -run 'Finding|Cobol'`: ok.
- `go test -race -count=1 -timeout 30m ./internal/indexer/ -run '^TestCobol'`: ok (285 s).
- `go test -race -count=1 -timeout 30m ./cmd/gortex/ -run '^TestCobol'`: ok (58 s).
- `go test -race -count=1 ./internal/mcp/ ./internal/neo4jprojection/ -run 'Finding|IndexHealth|ProjectNode'`: ok.
- `go build ./...`: ok.
- `git diff --name-only 215c9660..HEAD` (before this SUMMARY): exactly the six planned files.

## Next Phase Readiness

- 05-14 (ledger exclusion) and 05-16 (placeholder namespace) can proceed; new tests use no slash-form placeholder literals.
- 05-17 must document the adjacency trade and the mixed-form same-line window recorded above.

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-23*

## Self-Check: PASSED

- All six key files exist; commits 6fe6f0bc, 79155e82, 8688c750, e5563500, 3c129bce found in git log.
