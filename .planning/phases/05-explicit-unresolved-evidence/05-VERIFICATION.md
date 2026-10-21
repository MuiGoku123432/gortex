---
phase: 05-explicit-unresolved-evidence
verified: 2026-10-21T18:39:34Z
verified_at_commit: 3040addf
status: gaps_found
score: 4/5 roadmap success criteria verified (plan-level truths 95/95 test-backed, but the D-01 stability truths were narrowed to comment-line shifts)
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "SC4 / GAP-06 / D-01: Unchanged evidence reuses finding identities; corrected source/parser evidence or retrieved artifacts close or supersede only the applicable finding class."
    status: failed
    reason: >-
      PARSE_UNRESOLVED finding IDs, and the ledger pairing key, are anchored to the innermost
      statement node ID. Statement IDs carry a per-kind ordinal within the paragraph
      (`<para>/@exec_sql_statement#k`). Inserting or deleting an unrelated same-kind statement
      earlier in the same paragraph therefore re-IDs every later finding whose own evidence text
      is unchanged. The ledger then records false lifecycle rows. Reproduced by the verifier on
      the phase's own fixture (cobolFindingShiftFixture, via `go test -overlay`, no repo change).
      (a) One clean `EXEC SQL SELECT AAA FROM APP.BBB END-EXEC.` inserted at the top of MAIN-PARA:
      1 of 3 IDs kept. ZQXCOL moved from `@exec_sql_statement` to `#2`, and ZQXOTH from `#2` to `#3`.
      FindingLedgerTransitions writes "ZQXCOL superseded by the new AAA tail" and "ZQXOTH superseded
      by the re-IDed ZQXCOL", which is wrong successor lineage for unchanged findings.
      (b) Deleting the first EXEC SQL: the ZQXOTH tail, with unchanged evidence, gets a new ID and a
      false `superseded` row.
      (c) WR-07, confirmed: appending a second program to a file moves its `uncontained-cascade`
      finding from container `OUTERPGM` to the file node. The ID changes although the evidence is
      the constant `uncontained-cascade`.
      This violates D-01's guarantee ("Edits elsewhere in the file and line shifts keep the ID")
      and the locked "accuracy without churn" preference. D-01 marks the scheme as costly to
      reverse: IDs are persisted, projected to Neo4j, and cited by Phase 8 claims. Every per-token
      SQL tail (D-18) sits in an EXEC SQL statement, so this pattern is common in DB2 COBOL.
      Phase 6 SC4 (incremental equals clean) does not catch it, because the churned IDs are
      deterministic.
    artifacts:
      - path: "internal/parser/languages/cobol_grammar_findings.go"
        issue: >-
          :168 container := cobolFindingContainer(...); :174 FindingSymbol(container, ...);
          :194 finding_container = container. :220-231 the statement level is tried first.
          :261-266 a document-scope gap goes to the single program when exactly one exists,
          else to the file.
      - path: "internal/parser/languages/cobol_grammar_statements.go"
        issue: ":102-105 the statement symbol is `<container>/@<kind>` plus `#k`, an ordinal among same-kind statements in the paragraph"
      - path: "internal/graph/finding_ledger.go"
        issue: ":77, :98, :105 pair on (Container, Class, Kind), so a re-anchored container yields false supersede or close-plus-open rows"
      - path: "internal/parser/languages/cobol_grammar_findings_test.go"
        issue: ":283-317 TestCobolGrammar_Findings_LineShiftKeepsIDs inserts only comment lines, so it cannot detect ordinal drift"
      - path: "internal/indexer/cobol_findings_equivalence_test.go"
        issue: "TestCobolFindings_LineShiftKeepsIDs (around :193-208) inserts only comment lines"
    missing:
      - "Anchor the parse finding ID and finding_container (the ledger pairing key) to a name-stable container: paragraph, section, data or environment section, program, or file. Never use a statement ordinal. The statement can stay the member_of target."
      - "Anchor document-scope findings (no-tree, budget-exceeded, invocation-failure, uncontained-cascade, analysis-failed) to the file node always, never to 'the single program when exactly one exists'."
      - "Bump graph.FindingIDRule (for example gortex-finding-id/2), because persisted IDs change."
      - "Regression test 1: insert a same-kind statement (EXEC SQL) above existing gaps in the same paragraph. Assert every pre-existing finding ID is unchanged and the ledger appends no row for them."
      - "Regression test 2: delete an earlier same-kind statement. Assert only the deleted statement's findings close and no superseded row is written for unchanged ones."
      - "Regression test 3: add or remove a second program in a file with a document-scope finding. Assert its ID is unchanged."
deferred: []
human_verification:
  - test: "Judgment-tier prohibition 05-07 (GAP-08): an uncontained cascade never passes silently or behind a flag, never blocks indexing or marks a repository unhealthy, and no later AI output can clear the cascade finding."
    expected: "Confirm. The non-authoritative LLM-judge verdict is HOLDS. TestCobolCascadeGate_* runs in the default -race suite with no flag, positive shapes still persist their file node, findings never enter health_score (TestIndexHealth_FindingsDoNotScore), and the TestFindingWriters_Inventory go/ast fence admits only deterministic writers."
    why_human: "Judgment-tier prohibition (unverified-prohibition: human review recommended). A must-NOT about future AI behavior cannot be proven by a present-day test."
  - test: "Judgment-tier prohibition 05-09 (GAP-07): no path other than deterministic evidence (source edit, parser or extractor change, retrieval or binding) closes, supersedes, suppresses, or alters a finding."
    expected: "Confirm. The LLM-judge verdict is HOLDS for the active node set: TestFindingWriters_Inventory confines KindFinding construction, finding_status writes, AppendFindingTransitions, and EvictFindingNodesByIDs to the two writer files plus the ledger. Note that the CR-01 gap above and WR-01/WR-02 below can make history rows misattribute transitions."
    why_human: "Judgment-tier prohibition (unverified-prohibition: human review recommended)."
  - test: "WR-04 decision: the finding_evidence_digest is unsalted BLAKE3 of public-rule plus evidence, and under D-18 each unparsed-tail digest covers exactly one token. The digest is stored, searchable, projected to Neo4j, and folded into the ID."
    expected: "Decide now whether to switch to a keyed digest (per-workspace key) or to accept and document the dictionary-recovery residual risk. Literal source text is never stored, which satisfies the 05-01 privacy prohibition as written."
    why_human: "This is a security-posture decision. Changing the digest later re-IDs every finding, the same costly reversal D-01 warns about, so it should be decided together with the CR-01 identity fix."
  - test: "WR-01 decision: recordFindingLedger runs under findingLedgerMu only. It does not hold cobolReconcileMu (internal/resolver/cobol_references.go:150) or the repository lane/topology writer. A TrackRepoCtx tail ledger (internal/indexer/multi.go:2949) widened to a dependent repository can read between that repository's file commit, which evicts its external findings, and its reconcile, which re-adds them."
    expected: "Decide whether best-effort history (D-07) tolerates a possible false active->closed->active pair under that interleaving, or whether the ledger must take the same exclusion as the mutations. No concurrency test currently exercises the interleaving."
    why_human: "The race window needs a purpose-built concurrency test or a design decision. The active (canonical) finding set is unaffected, and only non-canonical history rows can be wrong."
---

# Phase 5: Explicit Unresolved Evidence Verification Report

**Phase Goal:** Users can see and distinguish source that could not be parsed from referenced artifacts that were not retrieved, without fabricated evidence.
**Verified:** 2026-10-21T18:39:34Z (HEAD `3040addf`; no non-`.planning` change since `ef329c28`)
**Status:** gaps_found
**Re-verification:** No. This is the initial verification.

## Verdict

- **Most of the phase works.**
  - Both finding classes exist and are persisted in SQLite.
  - They are distinguishable, carry the full field contract, and are served through MCP, the CLI, search, and `index_health`.
  - Placeholders are not fabricated.
  - The cascade gate is a default-suite regression test.
- **One roadmap contract fails: success criterion 4 (GAP-06).**
  - A parse finding's identity is anchored to an ordinal statement ID.
  - Unrelated same-kind statement inserts or deletes in the same paragraph re-ID findings whose evidence is unchanged.
  - The ledger then writes false `superseded` lineage for those findings.
- **This is CR-01 from the advisory review, and it is real.** I reproduced it independently.
  - It is a gap, not a nit: D-01 explicitly guarantees ID stability across "edits elsewhere in the file".
  - D-01 also flags the ID scheme as costly to change after Phase 8 starts citing IDs.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Named unparsed tails and localized ERROR/MISSING nodes produce stable, source-positioned PARSE_UNRESOLVED findings; unavailable targets produce separate EXTERNAL_UNRESOLVED findings | VERIFIED (emission, positioning, class separation). The stability defect is counted under #4. | `cobolFindings` (`cobol_grammar_findings.go:49-87`) maps every producer PARSE_UNRESOLVED gap, one per tail token (D-18), and skips producer EXTERNAL gaps (D-17). The resolver builds EXTERNAL_UNRESOLVED via `newCobolExternalFinding` (`resolver/cobol_findings.go:373+`). Tests: `TestCobolGrammar_Findings_UnparsedTailPerToken`, `_ErrorContainer`, `_MissingZeroWidth`, `_NoProducerExternal`, `TestCobolReconcile_ExternalMissingCall`, `TestCobolFindings_ExternalPersisted`. All passed under -race. |
| 2 | Each finding exposes stable identity, exact scope and range, containing symbol, resolution class, normalized reason, severity, parser version, lifecycle status | VERIFIED (fields). Identity stability is counted under #4. | `cobolAddFinding` Meta (`:184-205`) plus `cobolStampRange`; external Meta copies `prov_*` from the edge. Tests: `TestCobolGrammar_FindingMetaContract`, `TestCobolReconcile_ExternalMetaContract`, `TestCobolFindings_ExternalMCPGetSymbol`, `_ExternalCLIGetSymbol`. All passed. |
| 3 | Missing-target placeholders preserve the observed target reference without fabricated source, body, or attributes | VERIFIED | `<repo>/cobol-missing::<type>/<key>` KindResource with Meta restricted to key, type, `not_obtained`, and `synthetic`. It has no range, parser stamp, or raw spelling, and it is never a binding candidate. Tests: `TestCobolReconcile_PlaceholderShape`, `_PlaceholderNeverCandidate`, `TestCobolFindings_PlaceholderMCPGetSymbol`, `_PlaceholderSearch`. All passed. |
| 4 | Unchanged evidence reuses finding identities; corrected evidence or retrieved artifacts close or supersede only the applicable finding class | FAILED (BLOCKER) | Positive paths pass: an ERROR fix closes only that finding, a token edit supersedes, retrieval closes only that COPY's finding, restore reopens, and a re-stamp writes no row (`TestCobolFindings_History*`, `_ExternalClosesOnRetrieve`). An **unrelated same-kind statement edit re-IDs unchanged findings** and writes false supersede rows; see the probe below. Document-scope findings re-anchor when the program count changes (WR-07, reproduced). |
| 5 | Destructive parser-recovery cascades remain deterministic regression failures regardless of later AI output | VERIFIED. The judgment prohibition is flagged for human review. | `TestCobolCascadeGate_{CleanControl,Positive,Negative,Copybook}` run on the real path (MultiIndexer plus SQLite) with no flag, and passed under -race. `TestFindingWriters_Inventory` fences writers to deterministic code. |

**Score:** 4/5 roadmap success criteria. Plan frontmatter truths: 95/95 are backed by passing named tests. However, the D-01 stability truths in 05-01 and 05-12 were written as "inserting comment lines keeps every finding ID", a narrowing of SC4 that structurally cannot detect ordinal drift.

### CR-01 Reproduction (verifier-run, overlay probe, no repository change)

The probe was `go test -overlay` with a temporary test file in `internal/parser/languages`. Fixture: `cobolFindingShiftFixture`, which holds 2 SQL tails (`ZQXCOL`, `ZQXOTH`) and 1 ERROR.

| Edit | IDs kept | Ledger rows (FindingLedgerTransitions) |
|------|----------|----------------------------------------|
| Insert `EXEC SQL SELECT AAA FROM APP.BBB END-EXEC.` as the first MAIN-PARA statement | 1/3 (only the ERROR, anchored at `@procedure_division`) | `ZQXCOL(@exec_sql_statement)` superseded by AAA's tail; `ZQXOTH(#2)` superseded by the re-IDed ZQXCOL tail; ZQXOTH opened as new at `#3`. The lineage is wrong. |
| Delete the first `EXEC SQL` (ZQXCOL) | 1/3 | ZQXCOL closed (correct). Unchanged ZQXOTH superseded by its own re-IDed self (wrong). |
| Insert `MOVE 1 TO WS-B.` above (different kind) | 3/3 | none |
| Insert a clean `COMPUTE` above | 3/3 | none |
| Insert a new paragraph with EXEC SQL above MAIN-PARA | 3/3 (+1 new) | open only |
| Append a second program to a file with an `uncontained-cascade` finding (WR-07) | the cascade finding re-IDed (`OUTERPGM` to the file container) | close plus open of the "same" cascade |

- **Root cause:** `cobolFindingContainer` returns `w.statement[i].ID` first (`cobol_grammar_findings.go:224`). Statement symbols are `<container>/@<kind>#k` (`cobol_grammar_statements.go:102-105`).
- **External findings are not affected.** Their container is the edge `From`, which is the paragraph (`resolver/cobol_findings.go:391`).
- **Correction to the review:** the review predicted close-plus-open rows. The actual rows are false `superseded` rows, because the shifted ordinals collide with the old container keys. That is arguably worse for Phase 8 claim lineage.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/graph/finding.go` | FindingSymbol, digest, constants | VERIFIED | `gsd verify.artifacts`: 4/4 for 05-01 |
| `internal/graph/node.go` | KindFinding registered | VERIFIED | `:295`, `:322` |
| `internal/parser/languages/cobol_grammar_findings.go` | parse findings, cascade, analysis-failed | VERIFIED (exists, substantive, wired) | Called at `cobol_grammar.go:298`; copybook path `:219` |
| `internal/graph/finding_node_evict.go` + `store_sqlite/finding_node_evict.go` | kind-guarded eviction | VERIFIED | Delegates to `evictSyntheticNodesByIDs(...KindFinding)` (`:34`) |
| `internal/mcp/tools_enhancements.go` | `findings` counts in index_health | VERIFIED | `:3245` NodesByKind loop; `:3540` result |
| `internal/mcp/resources.go` | schema lists `finding` | VERIFIED | `:218-224` |
| `internal/resolver/cobol_findings.go` | placeholders, external findings, change drain | VERIFIED | Called from `cobol_references.go:359`; `EvictFindingNodesByIDs` `:241`; sweep `EvictFile` `:310` |
| `internal/graph/finding_ledger.go` | diff: open/close/supersede/reopen | VERIFIED (exists, wired). The pairing key inherits the CR-01 anchor. | Used by `indexer/finding_ledger.go:67` |
| `internal/graph/store_sqlite/store_finding_history.go` + schema v23 | sidecar table, latest-row index | VERIFIED | `schema.go:1252`; `store_purge.go:45,155,256` |
| `internal/indexer/finding_ledger.go` | ledger, FTS sync | VERIFIED | `cobol_reconcile.go:80,109`; `multi.go:1031-1032` |
| `internal/indexer/cobol_cascade_gate_test.go` | default-suite gate | VERIFIED | No skip or flag; reuses `newCobolLifecycleEnv` |
| `internal/graph/finding_writers_test.go` | D-16 go/ast inventory | VERIFIED | Passed |
| `docs/languages.md` | findings, placeholders, D-20 limitation | VERIFIED | `:279` |
| `.planning/WINDOWS.md` rows 4-6 | closed by findings | VERIFIED | All three rows read `fixed` |

### Key Link Verification

The gsd `verify.key-links` tool could not parse most `from:` values, which are descriptive rather than plain paths, and it rejected double-escaped regexes. Each link was checked manually instead:

| From | To | Status | Evidence |
|------|----|--------|----------|
| `cobol_grammar.go` Extract | `cobolFindings` | WIRED | `:297-298` `cobolSQL(w, data)` then `cobolFindings(w)` |
| `cobol_grammar_findings.go` / `resolver/cobol_findings.go` | `graph.FindingSymbol` | WIRED | `:174` / `:392` |
| `finding_node_evict.go` | `evictSyntheticNodesByIDs` | WIRED | `:34` |
| findings | grade Reason constants | WIRED | `:155-158` |
| `MaterialIdentity` | `cobolGrammarExtractorVersion` (`/6`) | WIRED | `cobol_grammar.go:46`, `:184` |
| `buildIndexHealthBasePayloadCtx` | KindFinding | WIRED | `tools_enhancements.go:3245` |
| `reconcileCobol` | `reconcileCobolFindings` | WIRED | `cobol_references.go:359` (before `reconcileCobolResources` `:360`) |
| placeholder sweep | `EvictFile` | WIRED | `resolver/cobol_findings.go:310` |
| `viewGenSidecars`, purge, orphan, rekey | `finding_history` | WIRED | `schema.go:1252`; `store_purge.go` |
| settle points | `recordCobolFindingLedger` | WIRED | `cobol_reconcile.go:80`, `:109` |
| `TrackRepoCtx` | `reconcileCobolRepos` / `batchCobolRepos` | WIRED | `multi.go:2942`, `:2949` |
| ledger | `CobolFindingChanges` drain | WIRED | `finding_ledger.go:156`; master `multi.go:1031` (FTS only, no ledger; see WR-02) |
| FTS sync | `SymbolFTSBatchUpserter` | WIRED | `finding_ledger.go:213` |
| copybook error branch | `cobolCopybookAnalysisFailed` | WIRED | `cobol_grammar.go:219` |
| `cobol_nested_unresolved` stamp | `decideCobol` | WIRED | `cobol_grammar.go:272`; `cobol_references.go:45`, `:459-463` |
| equivalence tests | `requireSameDump` / `requireCleanEquivalent` | WIRED | `cobol_findings_equivalence_test.go:153-212` |

### Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|----------|------|--------|-----------|--------|
| index_health `findings` | per-class counts | `NodesByKind(KindFinding)` over the store | yes | FLOWING (`TestIndexHealth_FindingCounts`) |
| MCP/CLI `get_symbol` on a finding | Meta and edges | SQLite store after a real index | yes | FLOWING (`cmd/gortex` `TestCobolFindings_*`) |
| `search_symbols kind:finding` | FTS rows | `syncCobolFindingFTS` at each settle point | yes | FLOWING (`TestCobolFindings_SearchAfterIncremental`, `_PlaceholderSearch`) |

### Behavioral Spot-Checks (run by verifier, -race, -count=1)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Finding core, ledger diff, eviction, history sidecar, writer inventory, extractor findings, reasons, resolver placeholders and external findings, index_health, projection | `GOWORK=off go test -race -count=1 -timeout 600s -run 'TestFinding\|TestEvictFindingNodesByIDs\|TestSQLiteEvictFindingNodesByIDs\|TestCobolGrammar_Findings\|TestCobolGrammar_FindingMetaContract\|TestCobolGrammar_CopybookAnalysisFailed\|...\|TestCobolReconcile_\|TestIndexHealth_Finding\|TestProjectNode_FindingMeta' ./internal/graph/ ./internal/graph/store_sqlite/ ./internal/parser/languages/ ./internal/resolver/ ./internal/mcp/ ./internal/neo4jprojection/` | all 6 packages ok | PASS |
| Real-path cascade gate, lifecycle, ledger, history, search sync, equivalence | `GOWORK=off go test -race -count=1 -timeout 1500s -run 'TestCobolCascadeGate_\|TestCobolFindings_' ./internal/indexer/` | ok (115 s) | PASS |
| MCP/CLI acceptance plus tracer and breadth regressions | `GOWORK=off go test -race -count=1 -timeout 1500s -run 'TestCobolFindings_\|TestCobolTracer\|TestCobolBreadth' ./cmd/gortex/` | ok (54 s) | PASS |
| SC4 identity under an unrelated same-kind statement edit | overlay probe `TestProbeCR01`, `TestProbeCR01Ledger`, `TestProbeCR01LedgerDelete` | 1/3 IDs kept; false supersede rows | FAIL (gap) |
| WR-07 document-scope container | overlay probe `TestProbeWR07` | cascade finding re-IDed | FAIL (part of the same gap) |

- **Full suite:** not re-run, per the orchestrator. The 05-12 SUMMARY reports a cold full `-race` run at `ef329c28` with 157 ok and 0 FAIL. `git diff ef329c28..HEAD` outside `.planning/` is empty.

### Probe Execution

- **Step 7c: SKIPPED.** There are no `scripts/*/tests/probe-*.sh` files, and no plan declares a probe.

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|--------------|-------------|--------|----------|
| GAP-01 | 05-01, 05-03, 05-10 | Unparsed tail, ERROR, or MISSING produces a persisted PARSE_UNRESOLVED finding | SATISFIED | Per-gap emission tests; cascade gate |
| GAP-02 | 05-02, 05-05, 05-08, 05-09, 05-10, 05-12 | A not-retrieved target produces an EXTERNAL_UNRESOLVED finding | SATISFIED (see WR-06) | `TestCobolReconcile_External*`, `TestCobolFindings_ExternalPersisted` |
| GAP-03 | 05-01, 05-04, 05-08, 05-10 | The two classes are distinct in storage and query | SATISFIED | `prov_evidence_class`; separate ID namespaces `@finding/parse` and `@finding/external`; index_health counts by class |
| GAP-04 | 05-01, 05-03, 05-04, 05-08, 05-12 | Stable ID, scope, range, container, class, reason, severity, parser version, status | SATISFIED for the field contract. ID stability is BLOCKED under GAP-06. | Meta-contract tests |
| GAP-05 | 05-05, 05-11, 05-12 | Placeholder without fabricated source, body, or attributes | SATISFIED | Placeholder shape and search tests |
| GAP-06 | 05-01, 05-06, 05-08, 05-09, 05-11, 05-12 | Unchanged evidence reuses the active finding identity | BLOCKED (partial) | Holds for identical rerun, re-stamp, comment or line shifts, and clean-build order. Fails for an unrelated same-kind statement edit in the same paragraph and a program-count change (see gap). REQUIREMENTS.md marks it `[x]`, which is premature. |
| GAP-07 | 05-02, 05-05, 05-06, 05-08, 05-09, 05-11, 05-12 | Close or supersede when evidence resolves or replaces it | SATISFIED (the must-close direction), with an over-trigger caveat | Real-path history tests pass. The CR-01 gap makes it also supersede findings whose evidence did NOT change. |
| GAP-08 | 05-03, 05-07 | Cascade remains a deterministic regression failure | SATISFIED | Cascade gate in the default suite |

- **Orphaned requirements:** none. All 8 IDs map to Phase 5 in REQUIREMENTS.md, and each appears in at least one plan.

### Prohibitions (ADR-550)

| Plan | Requirement | Tier | Statement (short) | Disposition |
|------|-------------|------|-------------------|-------------|
| 05-01 | GAP-01 privacy | test | No raw gap text in Name, QualName, ID, or Meta | VERIFIED: `TestCobolGrammar_FindingMetaContract` asserts the absence of the `ZQXCOL` token, and `TestCobolReconcile_ExternalMetaContract` asserts the raw spelling is never stored. Residual risk WR-04 is routed to human decision. |
| 05-01 | GAP-01 transparency | test | No dropping or aggregating of producer parse gaps | VERIFIED: `_UnparsedTailPerToken`, `assertCobolFindingsPerGap` |
| 05-05 | GAP-02 transparency | test | No placeholder for a present, builtin, ambiguous, dynamic, or bound -INC target | VERIFIED: `TestCobolReconcile_PlaceholderNotForBoundAmbiguousDynamicBuiltin` (D-20 is an accepted, documented limitation) |
| 05-05 | GAP-05 safety | test | No fabricated content on a placeholder; never bound as an implementation | VERIFIED: `_PlaceholderShape`, `_PlaceholderNeverCandidate`, `_PlaceholderSearch` |
| 05-07 | GAP-08 safety | judgment | Cascade never silent, never blocks; AI cannot clear it | FLAGGED (unverified-prohibition: human review recommended). The LLM verdict, which is not authoritative, is HOLDS. |
| 05-08 | GAP-02 transparency | test | EXTERNAL_UNRESOLVED means exactly "never obtained" | VERIFIED: `_ExternalNotForBoundAmbiguousDynamicBuiltin` (see WR-08 note) |
| 05-09 | GAP-07 safety | judgment | Only deterministic evidence changes a finding's status | FLAGGED (unverified-prohibition: human review recommended). The LLM verdict, which is not authoritative, is HOLDS: `TestFindingWriters_Inventory`. |

### Anti-Patterns Found

- **Debt markers:** none. No `TBD`/`FIXME`/`XXX` in the 55 files changed `1e178afd..HEAD` outside `.planning`, and no added `TODO`/`HACK`/`PLACEHOLDER` lines.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/parser/languages/cobol_grammar_findings.go` | 220-266 | Identity anchored to an ordinal statement ID or a program-count-dependent container | BLOCKER | SC4 / GAP-06 (gap above) |
| `internal/indexer/multi.go` | 1027-1032 | Master drain syncs FTS but writes no ledger (WR-02) | WARNING | The history close-vs-supersede classification can depend on timing. Intentional per the 05-11 plan; history is best-effort (D-07). |
| `internal/indexer/finding_ledger.go` | 53-72 | Ledger exclusion is narrower than the mutations it observes (WR-01) | WARNING | Possible false active->closed->active rows under concurrent track plus watcher. Routed to human decision. |
| `internal/resolver/cobol_findings.go` | 196-224, 379-390 | Ignores `cobol_occurrence_count` (WR-06); evidence omits the verb or edge kind (WR-03) | WARNING | Same-line duplicate CALLs to the same missing target in one paragraph produce one finding, not one per occurrence (D-04). A CALL<->XCTL edit keeps the ID. Edge cases only. |
| `internal/resolver/cobol_findings.go` | 150-163, 310 | Placeholder ID is also its FilePath, so a repository path `cobol-missing::program/X` could collide (WR-05) | WARNING | Low likelihood. The sweep's `EvictFile` could then remove a real file's nodes. |
| `internal/graph/finding.go` | 55-72 | Unsalted evidence digest over single tokens (WR-04) | WARNING | Dictionary-recoverable tokens through the projection. Routed to human decision. |
| `internal/resolver/cobol_references.go` | 430-463 | Visibility-hidden programs return `missing` (WR-08) | INFO | Defensible under COBOL semantics: a CALL to a non-visible nested name resolves to a separately compiled program, which was indeed not obtained. Not a fabrication. |

### Assessment of the Orchestrator's Specific Questions

- **CR-01 (statement-ordinal anchor):**
  - It is real, and I reproduced it independently.
  - It is a gap against SC4, D-01, and GAP-06, not merely advisory, for three reasons:
    1. D-01's text promises stability across "edits elsewhere in the file".
    2. The CONTEXT "accuracy without churn" preference forbids exactly this.
    3. The plans narrowed the stability truth to comment-line inserts, so no test could catch it.
  - The defect also corrupts the ledger. It writes false `superseded` rows with wrong successor IDs, not just close-plus-open rows. That affects the lineage Phase 8 claims will cite.
  - It is not deferred: Phase 6 SC4 checks incremental-equals-clean, which deterministic churn satisfies.
- **WR-07:**
  - It is real and reproduced. It has the same root cause class as CR-01 (an unstable identity anchor), so it is folded into the same gap. It is lower frequency, since most COBOL files have one program, but cascade edits that hide or recover a `PROGRAM-ID` hit it.
- **WR-01 and WR-02:**
  - Both are real design limits, and both affect only non-canonical history (D-07 "best-effort").
  - The canonical active finding set stays correct and is proven by the equivalence tests.
  - WR-02's no-ledger-at-master-drain is a documented plan decision (05-11 truth 2).
  - I classify both as WARNING, not BLOCKER, against SC4: SC4's observable contract is met on every deterministic settle-point path the tests exercise.
  - WR-01 is routed to a human decision because no concurrency test exists to settle it.

### Human Verification Required

#### 1. Judgment prohibition 05-07 (GAP-08)

- **Test:** Review that a cascade can never pass silently or be cleared by AI, and never blocks indexing.
- **Expected:** Confirm HOLDS. Evidence: the cascade gate is in the default suite, the D-16 writer inventory, and `TestIndexHealth_FindingsDoNotScore`.
- **Why human:** This is a judgment-tier prohibition.

#### 2. Judgment prohibition 05-09 (GAP-07)

- **Test:** Review that no MCP, CLI, or AI path can close or alter a finding.
- **Expected:** Confirm HOLDS. Evidence: `TestFindingWriters_Inventory` go/ast fence.
- **Why human:** This is a judgment-tier prohibition.

#### 3. WR-04 digest posture

- **Test:** Decide between a keyed digest and documenting the residual risk.
- **Expected:** A recorded decision, made together with the CR-01 fix, because both change persisted IDs.
- **Why human:** This is a security-posture trade-off with a costly-to-reverse ID impact.

#### 4. WR-01 ledger exclusion

- **Test:** Decide whether false history pairs under concurrent track plus watcher are acceptable for best-effort history, or require the ledger to share the mutation exclusion.
- **Expected:** A recorded decision, plus a concurrency test if the ledger is tightened.
- **Why human:** No test exercises the interleaving.

### Gaps Summary

There is one blocking gap, with a single root cause.

- **What fails:**
  - PARSE_UNRESOLVED finding IDs, and the ledger's `(container, class, kind)` pairing key, are anchored to the innermost statement node.
  - That node's ID carries a per-kind ordinal (`@exec_sql_statement#k`).
  - Document-scope findings anchor to "the single program, else the file".
  - So adding or removing an unrelated same-kind statement earlier in a paragraph, or changing a file's program count, re-IDs findings whose own evidence did not change.
  - The ledger then records false `superseded` or `closed` lineage.
- **Why it matters now:**
  - It breaks roadmap SC4 and GAP-06, and it violates D-01 and the locked "accuracy without churn" preference.
  - Every per-token SQL tail lives in an EXEC SQL statement, so the pattern is common in DB2 COBOL.
  - D-01 rates the ID scheme as costly to change once Neo4j projections and Phase 8 claims depend on it.
- **The fix is contained:**
  1. Anchor identity and the pairing key to the paragraph, section, program, or file, and keep the statement only as the `member_of` target.
  2. Always anchor document-scope findings to the file node.
  3. Bump `FindingIDRule`.
  4. Add the three regression tests listed in the gap.
- **Everything else holds:**
  - Emission, the field contract, placeholders, the cascade gate, MCP/CLI/search/index_health exposure, the writer fence, and incremental-equals-clean.
  - All targeted tests pass under -race.

**Override path, if the deviation is intended (not recommended):** add the following to this file's frontmatter. This is not recommended because D-01 marks the scheme costly to reverse after Phase 8 starts citing IDs.

```yaml
overrides:
  - must_have: "Unchanged evidence reuses finding identities; corrected source/parser evidence or retrieved artifacts close or supersede only the applicable finding class."
    reason: "<why statement-ordinal anchoring is acceptable>"
    accepted_by: "<name>"
    accepted_at: "<ISO timestamp>"
```

---

_Verified: 2026-10-21T18:39:34Z_
_Verifier: the agent (gsd-verifier)_
