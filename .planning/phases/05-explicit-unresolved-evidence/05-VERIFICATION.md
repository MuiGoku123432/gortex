---
phase: 05-explicit-unresolved-evidence
verified: 2026-10-26T21:14:27Z
verified_at_commit: e4c591dc
status: gaps_found
score: 4/5 roadmap success criteria; gap-closure plan truths 40/42 (1 by override); 05-01..05-12 truths hold except where CR-01 applies
behavior_unverified: 0
overrides_applied: 1
overrides:
  - must_have: "The full suite `GOWORK=off go test -race -timeout 60m ./...` runs once, after `df -g /System/Volumes/Data` shows at least 10 GiB available, and its log ends with exit=0 and holds no FAIL line (the gap-closure phase gate)."
    reason: "The substance is verified on split evidence; the literal clause cannot be met while any Gortex process on this machine appends to the developer's real query log during the run. Run 1 (05-17-full-suite.run1.log) executed all 156 testable packages fresh (0 cached) with 0 '--- FAIL', 0 'panic:', and 0 'DATA RACE' lines; cmd/gortex printed PASS (m.Run returned 0) and failed only on the pre-existing TestMain fingerprint guard (cmd/gortex/main_test.go:56-61), whose report names only ~/.gortex/cache/query-log.jsonl, written by outside sessions. cmd/gortex passed alone with the guard intact (05-17-cmd-gortex-isolated.log, exit=0), and the guard also stayed intact in this verification's targeted cmd/gortex COBOL run. No non-.planning file changed after 3abe472d. WINDOWS.md entry 8 stays open until a clean full run or a guard fix."
    accepted_by: "cfanch06 (user decision recorded in 05-17-SUMMARY key-decisions and deviation 6, and in WINDOWS.md entry 8)"
    accepted_at: "2026-10-26T18:55:45Z"
re_verification:
  previous_status: gaps_found
  previous_score: "4/5 roadmap success criteria (plan-level truths 95/95 test-backed)"
  previous_report_commit: 211dbe62
  gaps_closed:
    - "SC4 / GAP-06 / D-01 (first review CR-01 and WR-07): parse finding IDs and finding_container (the ledger pairing key) now hash a name-stable anchor and never a statement ordinal; document-scope findings always anchor to the file node; FindingIDRule is gortex-finding-id/2; all three required regressions exist at extractor level and on the real path and pass under -race."
  gaps_remaining: []
  regressions:
    - "New review CR-01 (introduced by 05-16): placeholder IDs are parsed at the first '::', so in a repository whose prefix contains '::' a not-retrieved COPY IDMS member ends with no EXTERNAL_UNRESOLVED finding, and finding and placeholder IDs route to the wrong owner. Reproduced by this verifier on memory, SQLite, and the real path."
  previous_human_verification_resolved:
    - "WR-04 digest posture: decided as D-22 (unkeyed, documented). Verified at internal/graph/finding.go:59-76 and docs/languages.md:279."
    - "WR-01 ledger exclusion: decided as D-23 (fixed). Verified by TestCobolFindings_LedgerExcludesTornMutation (track_tail, master_drain), which passes."
  previous_human_verification_open:
    - "Judgment-tier prohibitions 05-07 (GAP-08) and 05-09 (GAP-07): not deterministically provable; carried forward below."
gaps:
  - truth: "SC1 / GAP-02, 05-16 truths 5 and 6, and 05-08 truths 1 and 9: a not-retrieved COPY IDMS member keeps its EXTERNAL_UNRESOLVED finding across unchanged passes, and finding and placeholder IDs route to their owning repository, for every repository prefix the configuration accepts."
    status: failed
    reason: "Review CR-01 is real and is a regression from 05-16 (D-27). cobolUnresolvedFromPlaceholder and cobolOwnerPrefix both split an ID at its first '::'. A repository prefix may contain '::': config.ResolvePrefix returns entry.Name or the directory basename verbatim, and neither AddRepo nor Validate rejects it. For prefix a::b, the second pass cannot re-collect the COPY IDMS edge (which carries no cobol_placeholder), so its finding leaves the desired set and is evicted while the placeholder survives. Verifier probes: resolver unit level, memory and SQLite, two unchanged ReconcileCobol(nil) passes: prefix r1 keeps the finding, prefix a::b returns pass1=[finding] then pass2=[]. Real path, MultiIndexer + SQLite cold TrackRepoCtx: prefix fidms has 2 external findings (IDMS and CALL), prefix a::b has 1 (CALL only) while its idms_dictionary_member placeholder exists. Users of such a repository cannot see the unretrieved IDMS member as a finding. Scope is narrow: only COPY IDMS edges parse the ID (CALL, COPY, and -INC keep cobol_placeholder), and Gortex's documented ID grammar <repo_prefix>/<path>::<Symbol> (docs/multi-repo.md:438) is already ambiguous for such prefixes in core helpers outside Phase 5. No placeholder content is fabricated (SC3 holds)."
    artifacts:
      - path: "internal/resolver/cobol_findings.go"
        issue: ":92-99 cobolUnresolvedFromPlaceholder runs strings.Cut(id, \"::\"); for a::b::cobol-missing::idms_dictionary_member/... the CutPrefix fails and it returns \"\"; :284-299 the finding is then evicted as undesired"
      - path: "internal/resolver/cobol_references.go"
        issue: ":285-290 a COPY IDMS in-edge of a placeholder is re-collected only via cobolUnresolvedFromPlaceholder(e.To); \"\" skips add(), so the edge is never a desired entry again"
      - path: "internal/indexer/finding_ledger.go"
        issue: ":297-304 cobolOwnerPrefix returns the text before the first \"::\" when it precedes the first \"/\"; a::b/... and a::b::cobol-missing::... both route to \"a\" (ledger widening :187, search sync :272, :277); 05-09 used graph.RepoPrefixOfID, which returned a::b correctly"
      - path: "internal/config/global.go"
        issue: ":445-450 ResolvePrefix returns entry.Name or filepath.Base verbatim; AddRepo (:552) checks only duplicate paths; Validate (:406) checks only duplicate prefixes and project-name format"
      - path: "docs/languages.md"
        issue: ":279 the reserved repository-name sentence does not mention \"::\""
    missing:
      - "Parse placeholder IDs from the namespace marker, not the first separator (strings.LastIndex(id, \"::\"+cobolMissingNamespace) in cobolUnresolvedFromPlaceholder and cobolOwnerPrefix, keys escape \":\" so the last marker is the real one), or stamp cobol_placeholder on COPY IDMS edges so no ID is parsed, or reject \"::\" in repository prefixes at config/track time and document it with the reserved names."
      - "Resolver regression: prefix a::b, one COPY IDMS edge, two unchanged ReconcileCobol(nil) passes keep the IDMS finding on memory and SQLite."
      - "Real-path regression: track a repository named a::b holding COPY IDMS RECORD EMP-REC; after the cold track the IDMS EXTERNAL_UNRESOLVED finding exists, and the ledger and search sync route its IDs to a::b."
  - truth: "D-24 and the 05-14 GAP-07 prohibition: whether a finding is recorded closed or superseded must not depend on timing (docs/languages.md:279 states that it does not)."
    status: partial
    reason: "Review WR-01 is real by code reading (not probed). The sequential WR-02 scenario is fixed and tested (TestCobolFindings_MasterDrainClosesBeforeLaterEdit). On writer-free master paths (daemon warmup RunPreEnrichResolve and the public deferred catch-up full fallback), ResolveAllContext fires OnComputeDone (markReady) and then reconciles under ResolveMutex and cobolReconcileMu only; the master's ledger takes the reach writer only after it returns. A lane that takes the writer in between commits, reconciles, and runs its own settle ledger first, which can record the master's close plus its own open in the same container as a supersede. No test parks a lane in that window (master_drain parks after the writer is taken). The active finding set is unaffected and history is best-effort (D-07), so this is not a roadmap SC failure; but it contradicts the user-decided D-24 MUST, the second clause of the 05-14 test-tier prohibition (fail-closed per ADR-550), and the user-facing docs sentence."
    artifacts:
      - path: "internal/indexer/multi.go"
        issue: ":1059 master.ResolveAllContext reconciles without the reach writer when topologyHeld is false; :1070 recordCobolSettleLedger takes it only afterwards"
      - path: "internal/resolver/resolver.go"
        issue: ":1201 OnComputeDone (daemon markReady) fires before :1691 reconcileCobol"
      - path: "cmd/gortex/daemon_state.go"
        issue: ":944-949 warmup passes markReady into the writer-free RunPreEnrichResolve"
      - path: "docs/languages.md"
        issue: ":279 \"so whether a finding is closed or superseded does not depend on timing\" is false on these paths"
    missing:
      - "Either run the COBOL reconcile of writer-free master paths after ResolveAllContext returns, in the same writer section as its drain, ledger, and search sync (review WR-01 option 1), plus an interleaving test that parks a lane between the master's reconcile and its ledger;"
      - "or accept the window under D-07 by explicit user decision (an override of D-24 for writer-free master paths), record it with the 05-14 accepted windows, and change the docs sentence to say the guarantee holds only when no edit overlaps a warmup or fallback resolve."
deferred: []
human_verification:
  - test: "Judgment-tier prohibition 05-07 (GAP-08): an uncontained cascade never passes silently or behind a flag, never blocks indexing or marks a repository unhealthy, and no later AI output can clear the cascade finding."
    expected: "Confirm. Non-authoritative LLM-judge verdict: HOLDS. TestCobolCascadeGate_{CleanControl,Positive,Negative,Copybook} pass in the default -race run with no flag; TestIndexHealth_FindingsDoNotScore passes; TestFindingWriters_Inventory passes; the graph exposes no generic remove-by-ID, so a finding leaves the store only through eviction of its file or repository, the fenced resolver call to EvictFindingNodesByIDs, or nothing (the other by-ID evicters are kind-guarded to config, contract, and resource nodes)."
    why_human: "Judgment-tier prohibition (unverified-prohibition: human review recommended). A must-NOT about future AI output cannot be proven by a present-day test."
  - test: "Judgment-tier prohibition 05-09 (GAP-07): no path other than deterministic evidence (source edit, parser or extractor change, retrieval or binding) closes, supersedes, suppresses, or alters a finding."
    expected: "Confirm. Non-authoritative LLM-judge verdict: HOLDS for the active node set. The D-16 inventory confines KindFinding construction, finding_status writes, AppendFindingTransitions, and EvictFindingNodesByIDs to the extractor, the resolver, and the ledger. Residual: the inventory matches literal write shapes, not a Meta write through a computed key, and gap 2 above can still mislabel close versus supersede in history under one race."
    why_human: "Judgment-tier prohibition (unverified-prohibition: human review recommended)."
---

# Phase 5: Explicit Unresolved Evidence Verification Report

**Phase Goal:** Users can see and distinguish source that could not be parsed from referenced artifacts that were not retrieved, without fabricated evidence.
**Verified:** 2026-10-26T21:14:27Z (HEAD `e4c591dc`; no non-`.planning` change since `3abe472d`)
**Status:** gaps_found
**Re-verification:** Yes, after gap closure (plans 05-13 to 05-17). Previous report: `211dbe62`.

## Verdict

- **The previous blocking gap is closed.**
  - Parse finding identity and the ledger pairing key now use a name-stable anchor.
  - Document-scope findings anchor to the file.
  - `FindingIDRule` is bumped to `gortex-finding-id/2`.
  - All three required regressions pass at the extractor level and on the real path.
  - Both earlier human decisions are implemented: WR-04 became D-22, WR-01 became D-23.
- **The gap cycle introduced one new blocking defect: CR-01 of the new review (05-16).**
  - In a repository whose prefix contains `::`, a not-retrieved `COPY IDMS` member ends with no EXTERNAL_UNRESOLVED finding.
  - Finding and placeholder IDs also route to the wrong owner for ledger and search.
  - I reproduced it on memory, SQLite, and the real path.
  - The configuration accepts such prefixes, and nothing rejects them.
  - It is narrow: only `COPY IDMS` references, and only repositories with `::` in their prefix.
- **One partial gap: WR-01 of the new review (D-24).**
  - Close versus supersede in `finding_history` can still depend on timing on writer-free master paths.
  - The user docs claim the opposite.
  - The active finding set is unaffected (history is best-effort under D-07).
  - Close it with a fix or with an explicit user override.
- **WR-02 (imageless occurrence label) is a warning, not a gap.** Bundle its five-line fix with CR-01; both live in the same resolver file.
- **The full-suite gate passes by override.** The split evidence holds up, and the user accepted it on 2026-10-26. WINDOWS.md entry 8 stays open.

## Re-verification: Previous Gap Items

Each item is from the previous report's `gaps[0].missing` list.

| Required item | Status | Evidence |
|---|---|---|
| Anchor the ID and `finding_container` to a name-stable container, never a statement ordinal; the statement may stay the `member_of` target | CLOSED | `cobol_grammar_findings.go:177` (anchor), `:183` (ID), `:203` (`finding_container`), `:218-222` (`member_of` keeps the innermost statement). `cobolFindingAnchor` (`:237-274`) climbs procedure containers, then sections, then the program, then the root, then the file, and has no statement level. `w.container` only ever holds division, section, or paragraph QualNames (`cobol_grammar_procedure.go:81`, `:149`; `cobol_grammar_statements.go:86` copies one of those). |
| Document-scope findings always anchor to the file node | CLOSED | `:238-240` (`c < 0` or out of range returns `w.filePath`). No non-comment `len(w.programs)` remains. |
| Bump `graph.FindingIDRule` | CLOSED | `finding.go:32` `gortex-finding-id/2`; `:33` `gortex-finding-evidence/2` (D-25). Both writers use the constants. |
| Regression 1: insert a same-kind statement | CLOSED | `TestCobolGrammar_Findings_SameKindInsertKeepsIDs` and `TestCobolFindings_SameKindInsertKeepsIDs`: every pre-existing ID kept; only `"" -> active` rows for the new ones; incremental equals clean. PASS under -race. |
| Regression 2: delete an earlier same-kind statement | CLOSED | `..._SameKindDeleteKeepsIDs` and `TestCobolFindings_SameKindDeleteClosesOnlyItsOwn`: exactly one `active -> closed` row, no superseded row. PASS. |
| Regression 3: add and remove a second program | CLOSED | `..._DocumentScopeAnchorsToFile` (zero, one, and two programs give one ID) and `TestCobolFindings_SecondProgramKeepsDocumentFinding` (no row for the cascade finding). PASS. |

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Named unparsed tails and localized ERROR or MISSING nodes produce stable, source-positioned PARSE_UNRESOLVED findings; unavailable targets produce separate EXTERNAL_UNRESOLVED findings | FAILED (narrow, gap 1) | Emission holds everywhere else: 17 `TestCobolGrammar_Findings*` tests, the cascade gate, `TestCobolReconcile_External*`, and `TestCobolFindings_ExternalPersisted` pass. CR-01: in a repository whose prefix contains `::`, the `COPY IDMS` EXTERNAL_UNRESOLVED finding is absent after a cold track (real-path probe). |
| 2 | Each finding exposes stable identity, exact scope and range, containing symbol, resolution class, normalized reason, severity, parser version, and lifecycle status | VERIFIED (warning WR-02) | `TestCobolGrammar_FindingMetaContract`, `TestCobolReconcile_ExternalMetaContract`, and the MCP and CLI `get_symbol` tests pass. WR-02: occurrences 2..n of a collapsed imageless edge get a false `same_line_occurrence` absence reason. |
| 3 | Missing-target placeholders preserve the observed target reference without fabricated source, body, or attributes | VERIFIED | `<repo>::cobol-missing::<type>/<key>`, empty FilePath, D-11 Meta only. `TestCobolReconcile_PlaceholderShape`, `_NeverCandidate`, `_RefusedOverRealNode`, `_SweepByID`, `TestCobolFindings_PlaceholderNamespaceNoFileCollision`, and `_PlaceholderSearch` pass. CR-01 leaves the placeholder intact and fabricates nothing. |
| 4 | Unchanged evidence reuses finding identities; corrected evidence or retrieved artifacts close or supersede only the applicable finding class | VERIFIED (gap 2 is against D-24, not this wording) | The previous gap is closed (table above). D-23: `TestCobolFindings_LedgerExcludesTornMutation` passes (both subtests). D-24 (sequential): `TestCobolFindings_MasterDrainClosesBeforeLaterEdit` passes. All 31 `TestCobolFindings_*` real-path tests pass under -race. The supersede in gap 2 stays within the applicable class. |
| 5 | Destructive parser-recovery cascades remain deterministic regression failures regardless of later AI output | VERIFIED (judgment prohibition flagged) | `TestCobolCascadeGate_{CleanControl,Positive,Negative,Copybook}` pass on the real path with no flag. `TestFindingWriters_Inventory` passes. The AI clause is routed to human review. |

**Score:** 4/5 roadmap success criteria.

### Gap-Closure Plan Truths (05-13 to 05-17)

| Plan | # | Truth (short) | Status | Evidence |
|---|---|---|---|---|
| 05-13 | 1 | ID and `finding_container` use the name-stable anchor; `member_of` targets the innermost statement | VERIFIED | Code cited above; `assertCobolFindingMember` |
| 05-13 | 2 | Regression 1 (insert), extractor and real path | VERIFIED | Two tests PASS |
| 05-13 | 3 | Regression 2 (delete) closes only ZQXCOL, no supersede | VERIFIED | Two tests PASS |
| 05-13 | 4 | Regression 3: document scope on the file for 0, 1, or 2 programs | VERIFIED | Two tests PASS |
| 05-13 | 5 | Both rule constants bumped once, written to Meta | VERIFIED | `finding.go:32-33`; `cobol_grammar_findings.go:206-207`; `cobol_findings.go:479-480` |
| 05-13 | 6 | D-25 six-field evidence; CALL, LINK, XCTL, LOAD distinct; XCTL to LINK supersedes only its own | VERIFIED | `cobol_findings.go:455-459`; `TestCobolReconcile_ExternalReferenceFormEvidence`, `TestCobolFindings_ReferenceFormEditSupersedes` |
| 05-13 | 7 | D-26 n findings per collapsed edge; later occurrences carry no coordinates and `same_line_occurrence` | VERIFIED as written (WR-02) | `cobol_findings.go:236-255`; `TestCobolReconcile_ExternalSameLineOccurrences`, `TestCobolFindings_SameLineCallsPerOccurrence` |
| 05-13 | 8 | Edge probe: adjacency | VERIFIED | `TestCobolGrammar_Findings_SameEvidenceAnchors`, `_UnparsedTailPerToken` |
| 05-13 | 9 | Edge probe: empty | VERIFIED | `_MissingZeroWidth` asserts `FindingEvidenceDigest("")`; `_DocumentScopeAnchorsToFile/synthetic` |
| 05-13 | 10 | Edge probe: ordering, byte-identical reruns | VERIFIED | `cobol_grammar_findings.go:60-71`; `cobol_findings.go:257-270`; `TestCobolFindings_CleanDeterminism` |
| 05-13 | 11 | D-22 code doc corrected | VERIFIED | `finding.go:59-76` (unkeyed, dictionary; no "never leaks") |
| 05-13 | 12 | Suites pass under -race | VERIFIED | This verification's runs (below) and run 1 |
| 05-14 | 1 | Every MultiIndexer ledger runs under the writer and `cobolReconcileMu`; one lock order | VERIFIED | `cobol_reconcile.go:167-181`; `finding_ledger.go:159-162`; `cobol_references.go:177` |
| 05-14 | 2 | Taking and held variants at the right sites; no self-deadlock | VERIFIED | `cobol_reconcile.go:102-106`, `:117`; grep counts; the nested-root and topology-batch tests executed in run 1 (`internal/indexer` ok, 1283.7 s, no timeout) |
| 05-14 | 3 | Cold-track gap documented as benign | VERIFIED (documentation truth) | 05-14-SUMMARY "Cold-Track Gap"; residual corner accepted under T-05-73 |
| 05-14 | 4 | Concurrency probe (track_tail, master_drain) | VERIFIED | Both subtests PASS; mutation failures recorded in the SUMMARY |
| 05-14 | 5 | D-24: master drain ledgers at its own step; writer state threaded | VERIFIED | `multi.go:1036-1084`, `:1070`, `:1013-1022`, `:914` |
| 05-14 | 6 | D-24 WR-02 scenario (sequential): close, never a supersede | VERIFIED | `TestCobolFindings_MasterDrainClosesBeforeLaterEdit` PASS. The concurrent window is gap 2. |
| 05-14 | 7 | Edge probe: idempotency, zero rows | VERIFIED | `_SearchAfterFallbackMasterResolve`, `_HistoryIdenticalRerun` |
| 05-14 | 8 | `TopologyWriterActive` and the held-writer warning | VERIFIED | `cobol_reconcile.go:171-174` |
| 05-14 | 9 | Standalone Indexer tail passes `CobolReconcileLock` | VERIFIED | `cobol_reconcile.go:86-87` |
| 05-14 | 10 | Existing suites and the whole indexer package pass | VERIFIED | Run 1; this verification's runs |
| 05-15 | 1-4 | Kind-guarded `EvictResourceNodesByIDs` on both backends, shared helper, fence, conformance | VERIFIED | `resource_node_evict.go:19-36`; `store_sqlite/finding_node_evict.go:29+`; 7 named tests and subtests PASS |
| 05-16 | 1 | Placeholder ID shape, empty FilePath, audit exemption; no repository file can mint it | VERIFIED | `cobol_findings.go:72-85`, `:308-322`; `_PlaceholderShape`, `_PlaceholderPrefixDiagnostics`. A crafted prefix is handled by the documented WINDOW plus refusal. |
| 05-16 | 2 | Sweep by ID only; unsupported store warns with no fallback | VERIFIED | `cobol_findings.go:349-374`; 0 non-comment `.EvictFile(`; `_PlaceholderSweepByID` |
| 05-16 | 3 | WR-05 real path: a repository file at a placeholder-like path survives | VERIFIED | `TestCobolFindings_PlaceholderNamespaceNoFileCollision` PASS |
| 05-16 | 4 | Refusal over an existing node | VERIFIED | `cobol_references.go:327`; `_PlaceholderRefusedOverRealNode` |
| 05-16 | 5 | COPY IDMS round trip from a `<repo>::cobol-missing::idms_dictionary_member/...` ID | FAILED (gap 1) | Holds for prefix `r1`; fails for `a::b` (probes) |
| 05-16 | 6 | `cobolOwnerPrefix` routes both `<repo>/...` and `<repo>::...` | FAILED (gap 1) | Returns `a` for `a::b/...` and `a::b::...` (`finding_ledger.go:297-304`) |
| 05-16 | 7 | `RepoPrefixOfID` caller audit with dispositions | VERIFIED | 05-16-SUMMARY table (24 sites plus splitters); its (d) rows assume no `::` in a prefix |
| 05-16 | 8 | WINDOW paragraph | VERIFIED | `cobol_references.go:148-158` |
| 05-17 | 1-5 | Docs for D-22, D-27, D-21, D-25, and D-26 with the mixed-form window | VERIFIED | Every Task 1 grep holds; the slash form and the old parenthetical are gone. IN-05: the window sentence understates the collapse. |
| 05-17 | 6 | Docs for D-23 and D-24 history timing | VERIFIED as written | Both phrases are present. The appended "does not depend on timing" is false on writer-free master paths (gap 2). |
| 05-17 | 7 | D-20 kept; WR-08 and IN-01..07 not documented; inventory passes | VERIFIED | grep; `TestFindingWriters_Inventory` PASS |
| 05-17 | 8 | Full-suite gate log ends `exit=0` with no FAIL | PASSED (override) | Override: split evidence verified; accepted by cfanch06 on 2026-10-26 (see frontmatter) |

**Score:** 40/42 gap-closure truths (one by override). Both failures share root cause CR-01.

### Plans 05-01 to 05-12 (Regression)

- **96 truths.** Nine were superseded in wording by user-confirmed D-21, D-24, D-25, and D-27. Their behavior is re-tested in the new form by the updated tests, which pass:

| Plan:line | Superseded wording | Superseded by |
|---|---|---|
| 05-01:31 | `gortex-finding-id/1` | D-21 (`/2`) |
| 05-05:27, 05-05:31, 05-11:26, 05-11:28 | `<repo>/cobol-missing::...` placeholder spelling, FilePath equal to the ID | D-27 (`<repo>::cobol-missing::...`, empty FilePath) |
| 05-08:30 | External evidence of form, IDMS kind, and raw target | D-25 (six fields) |
| 05-09:26, 05-11:22 | Ownership routing via `graph.RepoPrefixOfID` | D-27 (`cobolOwnerPrefix`) |
| 05-11:23 | "no ledger runs there" at the master drain | D-24 |

- **The rest hold.** Evidence:
  - Run 1 executed every package fresh on the current code.
  - This verification's targeted runs pass: 17 parser, 25 resolver, 31 indexer findings, 4 cascade-gate, and 24 cmd/gortex COBOL tests, plus graph, store, MCP, and projection tests.
- **Exception:** 05-08 truth 1 (one finding per `COPY IDMS` occurrence) and truth 9 (an unchanged second reconcile writes nothing) also fail for `::` prefixes. That is the same root cause as gap 1.

### Assessment of the Orchestrator's Questions

1. **Full-suite gate (05-17 Task 2).**
   - The split evidence substantively supports "the full race suite is green" on the code at HEAD.
     - Run 1 executed all 156 testable packages fresh (0 cached), with 0 `--- FAIL`, 0 `panic:`, and 0 `DATA RACE` lines.
     - cmd/gortex printed `PASS` before the guard report. `m.Run()` therefore returned 0, and the guard (`main_test.go:56-61`) turned it into exit 1.
     - The guard names only the query log, which any outside Gortex call appends to.
     - cmd/gortex alone passed with the guard intact (`05-17-cmd-gortex-isolated.log`, exit=0).
     - My targeted cmd/gortex COBOL run also passed with the guard intact.
     - `git diff --name-only 3abe472d..HEAD -- . ':!.planning'` is empty.
   - One limit: the log does not echo the command. `-race` is inferred from package times (`internal/indexer` took 1283.7 s, against 1153.7 s in the 05-14 race run).
   - **Recorded as an override**, because the user's acceptance is documented and the substance checks out. WINDOWS.md entry 8 stays open until a literal clean run or a guard fix.
2. **CR-01 (new review): goal gap (gap 1).**
   - **Can prefixes contain `::`? Yes.**
     - `ResolvePrefix` (`global.go:445-450`) returns `entry.Name` or the basename verbatim.
     - `AddRepo` checks only duplicate paths, and `Validate` checks only duplicate prefixes and project names.
     - `TrackRepoCtx` accepted `Name: "a::b"` in my probe.
   - **Which criterion?** It falls under SC1 and GAP-02: missing evidence, not fabricated evidence. Its mechanism also violates SC4 and GAP-06, since an unchanged pass deletes the finding. It does not fall under SC3: the placeholder is intact and carries no invented content.
   - **Mitigating context:** Gortex's documented ID grammar (`docs/multi-repo.md:438`) is already ambiguous for such prefixes, and core helpers cut node IDs at the first `::`:
     - `graph.IDFile` (`overlay.go:369`);
     - `graph.EnclosingFromID` (`node.go:501`);
     - stub-kind detection (`stub.go:84`);
     - MCP rename (`rename_apply.go:32`).
   - That argues for rejecting `::` at track time, which also protects those helpers, or for an override that declares such prefixes unsupported.
3. **WR-01 (new review): partial gap (gap 2).**
   - **It is not a roadmap SC failure.** The active set is correct, the supersede stays within the class, and history is best-effort under D-07.
   - **But it is not yet an accepted window.**
     - D-24 is a user-decided MUST: "must not depend on timing".
     - The 05-14 test-tier prohibition has no wired enforcement for this window, so it fails closed under ADR-550.
     - The docs sentence is false today.
   - **It is reachable.** Daemon warmup passes `markReady` into the writer-free master (`daemon_state.go:944-949`), and the reconcile runs after ready (`resolver.go:1201` then `:1691`).
   - **Close it** with review option 1, or with a recorded user override plus the docs correction.
4. **WR-02 (new review): warning and follow-up, not a goal gap.**
   - Reproduced on memory and SQLite: an imageless edge with `cobol_occurrence_count=2` yields `#2` with `prov_range_absence=same_line_occurrence` in place of the true `no_original_projection`.
   - Counts, IDs, and classes stay correct, and imageless relationship edges need preprocessor-generated reference text, which is rare.
   - It does conflict with the carried-forward "never invent; absence reasons as-is" rule. `EndLine = StartLine` also understates a multi-line later occurrence.
   - Bundle the fix with gap 1: same file, about five lines.
5. **Previous human items.**
   - WR-04 and WR-01 are resolved and verified (D-22, D-23).
   - The two judgment-tier prohibitions cannot be proven deterministically. The inventory fences known write shapes, not every conceivable path, and "regardless of later AI output" concerns future code. They stay human items, with refreshed evidence.

## Required Artifacts (gap-closure)

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/parser/languages/cobol_grammar_findings.go` | `cobolFindingAnchor`, anchor vs `member_of` split | VERIFIED | Exists, substantive, wired from `Extract` |
| `internal/graph/finding.go` | Rule constants `/2`, D-22 doc | VERIFIED | `:32-33`, `:59-76` |
| `internal/resolver/cobol_findings.go` | Six-field evidence, occurrence expansion, `::` namespace, by-ID sweep | VERIFIED, defect at `:92-99` (gap 1) | Wired from `reconcileCobol` |
| `internal/indexer/cobol_findings_identity_test.go` | Five real-path identity tests | VERIFIED | All PASS |
| `internal/indexer/cobol_reconcile.go` | `recordCobolSettleLedger`, taking and held variants | VERIFIED | Wired at every settle point |
| `internal/indexer/finding_ledger.go` | `reconcileMu`, `cobolOwnerPrefix` | VERIFIED, defect at `:297-304` (gap 1) | Wired |
| `internal/reach/reach.go` | `TopologyWriterActive` | VERIFIED | Used at `cobol_reconcile.go:171` and in the exclusion test |
| `internal/indexer/cobol_findings_ledger_exclusion_test.go` | Deterministic interleaving test | VERIFIED | No sleep and no timed wait; both subtests PASS |
| `internal/graph/resource_node_evict.go`, `store_sqlite/resource_node_evict.go` | Kind-guarded resource evicter | VERIFIED | Conformance on both backends |
| `docs/languages.md:279` | D-21 to D-27 documented | VERIFIED, two inaccuracies | Timing claim (gap 2); occurrence and reserved-name wording (WR-02, IN-05, gap 1) |

## Key Link Verification (gap-closure)

| From | To | Via | Status |
|---|---|---|---|
| `cobol_grammar_findings.go` | `graph.FindingSymbol` | `cobolFindingAnchor` result hashed and written to `finding_container` (`:177`, `:183`, `:203`) | WIRED |
| `finding_ledger.go` `findingTransitionOf` | parse-finding Meta | reads `MetaFindingContainer` (`:105`) | WIRED |
| resolver `reconcileCobolFindings` | extractor dedupe | reads `cobol_occurrence_count` (`:51`, `:239`) | WIRED |
| `cobol_reconcile.go` | `reach.BeginTopologyMutation` | `:103`, `:169` | WIRED |
| `recordCobolSettleLedger` | `recordCobolFindingLedger` | `&mi.cobolReconcileMu` (`:179`) | WIRED |
| `runMasterResolveHookedContext` | `recordCobolSettleLedger` | `multi.go:1070` | WIRED (non-atomic with `:1059`; gap 2) |
| `(*Graph).EvictResourceNodesByIDs` | `evictSyntheticNodesByIDs` | `resource_node_evict.go:35` | WIRED |
| resolver sweep | `graph.EvictResourceNodesByIDs` | `cobol_findings.go:368` | WIRED |
| `finding_ledger.go` | `cobolOwnerPrefix` | `:187`, `:272`, `:277` | WIRED; parses wrongly for `::` prefixes (gap 1) |
| `docs/languages.md` | `internal/graph/finding.go` | "unkeyed", `gortex-finding-id/2` | WIRED |

## Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|---|---|---|---|---|
| `index_health` `findings` | Per-class counts | `NodesByKind(KindFinding)` | Yes | FLOWING (`TestIndexHealth_FindingCounts` PASS) |
| MCP and CLI `get_symbol` on a finding or placeholder | Meta and edges | SQLite after a real index | Yes | FLOWING (`TestCobolFindings_MCPGetSymbol`, `_CLIGetSymbol`, `_ExternalMCPGetSymbol`, `_PlaceholderMCPGetSymbol` PASS) |
| `search_symbols kind:finding` | FTS rows | `syncCobolFindingFTSByRepo` at each settle point | Yes; misrouted for `::` prefixes (gap 1) | FLOWING (`_SearchByNameAndKind`, `_PlaceholderSearch`, `_SearchAfterIncremental` PASS) |

## Behavioral Spot-Checks (run by the verifier, `-race -count=1`, each under `-timeout 300s`)

| Behavior | Command (abridged) | Result | Status |
|---|---|---|---|
| Extractor findings, identity regressions, meta contract | `./internal/parser/languages/ -run 'TestCobolGrammar_Findings_\|TestCobolGrammar_FindingMetaContract'` | 17 PASS | PASS |
| Real-path identity regressions | `./internal/indexer/ -run '^TestCobolFindings_(SameKind...\|SecondProgram...\|ReferenceForm...\|SameLine...)$'` | 5 PASS | PASS |
| All real-path findings tests (31), in three batches | `./internal/indexer/ -run '^TestCobolFindings_...'` | 31 PASS, including both `LedgerExcludesTornMutation` subtests | PASS |
| Cascade gate | `./internal/indexer/ -run '^TestCobolCascadeGate_'` | 4 PASS | PASS |
| Resolver COBOL suite | `./internal/resolver/ -run 'Cobol'` | ok (24.0 s) | PASS |
| Ledger diff, digest, writer inventory | `./internal/graph/ -run 'TestFindingWriters_Inventory\|TestFindingLedger...'` | 10 PASS | PASS |
| Resource evicter | `./internal/graph/`, `store_sqlite`, `storetest` named tests | 7 PASS (subtests confirmed) | PASS |
| `index_health` and projection | `./internal/mcp/ -run '...Finding...'`, `./internal/neo4jprojection/` | PASS | PASS |
| MCP and CLI acceptance | `./cmd/gortex/ -run '^TestCobolFindings_\|^TestCobolTracer\|^TestCobolBreadth'` | 24 PASS, guard intact | PASS |
| CR-01, resolver | Overlay probe `TestVerifierProbe_IDMSColonRepo`, memory and SQLite | `r1`: finding kept. `a::b`: pass1=[finding], pass2=[] | FAIL (gap 1) |
| CR-01, real path | Overlay probe `TestVerifierProbe_RealPathColonPrefix` (cold TrackRepoCtx) | `fidms`: 2 external findings. `a::b`: 1 (no IDMS finding; placeholder present) | FAIL (gap 1) |
| WR-02 | Overlay probe `TestVerifierProbe_NoImageOccurrence` | `#2` absence = `same_line_occurrence` (was `no_original_projection`) | Warning |
| Static checks on the 33 files changed in `33d287d4..HEAD` | gofmt, `go vet` (4 packages), debt-marker grep | clean | PASS |

- **The probes were overlay-only.** The probe files live in `$TMPDIR/opencode/verify05/`, outside the repository; no repository file was changed.
- **The full suite was not re-run**, as instructed.

### Probe Execution

- **Step 7c: SKIPPED.** No `scripts/*/tests/probe-*.sh` files exist, and no plan or summary declares one.

## Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|---|---|---|---|---|
| GAP-01 | 05-01, 05-03, 05-10 | Unparsed tail, ERROR, or MISSING produces a persisted PARSE_UNRESOLVED finding | SATISFIED | Per-gap emission tests; cascade gate |
| GAP-02 | 05-02, 05-05, 05-08..05-10, 05-12, 05-13, 05-16, 05-17 | A not-retrieved target produces an EXTERNAL_UNRESOLVED finding | BLOCKED (partial, gap 1) | Satisfied for every reference form except `COPY IDMS` in repositories whose prefix contains `::` |
| GAP-03 | 05-01, 05-04, 05-08, 05-10 | The two classes stay distinct in storage and query | SATISFIED | `prov_evidence_class`, separate ID segments, `index_health` by class |
| GAP-04 | 05-01, 05-03, 05-04, 05-08, 05-12, 05-13, 05-16, 05-17 | Full field contract | SATISFIED (warning WR-02) | Meta-contract tests |
| GAP-05 | 05-05, 05-11, 05-12, 05-15, 05-16, 05-17 | Placeholder without fabricated content | SATISFIED | Shape, refusal, sweep-by-ID, and namespace tests |
| GAP-06 | 05-01, 05-06, 05-08, 05-09, 05-11..05-14, 05-17 | Unchanged evidence reuses identity | SATISFIED (previous gap closed) | Six identity regressions; equivalence tests. CR-01's deletion of the IDMS finding is counted under GAP-02. |
| GAP-07 | 05-02, 05-05, 05-06, 05-08, 05-09, 05-11..05-17 | Close or supersede when evidence resolves or replaces it | SATISFIED for the active set; history timing partial (gap 2) | History, ledger-exclusion, and master-drain tests |
| GAP-08 | 05-03, 05-07 | Cascade remains a deterministic regression failure | SATISFIED (judgment clause flagged) | Cascade gate in the default suite |

- **Orphaned requirements:** none. All 8 IDs map to Phase 5 in REQUIREMENTS.md, and each appears in at least one plan's `requirements:`.
- **REQUIREMENTS.md** still shows all 8 unchecked, with "Gaps Found" in traceability. That is consistent with this result.

## Prohibitions (ADR-550)

| Plan | Requirement | Tier | Statement (short) | Disposition |
|---|---|---|---|---|
| 05-01 | GAP-01 privacy | test | No raw gap text in Name, QualName, ID, or Meta | VERIFIED (meta-contract tests). D-22 residual documented. |
| 05-01 | GAP-01 transparency | test | No dropping or aggregating of producer gaps | VERIFIED |
| 05-05 | GAP-02 transparency | test | No placeholder for present, builtin, ambiguous, dynamic, or bound targets | VERIFIED |
| 05-05 | GAP-05 safety | test | No fabricated placeholder content; never an implementation | VERIFIED |
| 05-07 | GAP-08 safety | judgment | Cascade never silent and never blocking; AI cannot clear it | FLAGGED (human review). LLM-judge verdict: HOLDS. |
| 05-08 | GAP-02 transparency | test | EXTERNAL_UNRESOLVED means exactly "never obtained" | VERIFIED |
| 05-09 | GAP-07 safety | judgment | Only deterministic evidence changes a finding | FLAGGED (human review). LLM-judge verdict: HOLDS. |
| 05-13 | GAP-06 transparency | test | No ID change or row for unchanged evidence after unrelated edits | VERIFIED (six regressions). The D-01 adjacency trade is documented. |
| 05-13 | GAP-02 transparency | test | No merging of occurrences; no shared evidence across reference forms | VERIFIED, with the documented mixed-form same-line window |
| 05-14 | GAP-07 transparency | test | (1) No closed or superseded row while evidence exists; (2) close vs supersede never timing-dependent | (1) VERIFIED. (2) FLAGGED: fails closed on writer-free master paths (gap 2). |
| 05-16 | GAP-05 safety | test | A placeholder never occupies, overwrites, or evicts a real node | VERIFIED |
| 05-17 | GAP-04 transparency | test | Docs never claim the digest prevents recovery or overstate reference-form edits | VERIFIED |

## Anti-Patterns Found

- **Debt markers:** none. No `TBD`, `FIXME`, or `XXX` in the 33 files changed in `33d287d4..HEAD` outside `.planning`, and no added `TODO`, `HACK`, or `PLACEHOLDER` lines. gofmt and vet are clean.

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `internal/resolver/cobol_findings.go` | 92-99 | ID parsed at the first `::` | BLOCKER | Gap 1 |
| `internal/indexer/finding_ledger.go` | 297-304 | Owner prefix parsed at the first `::` | BLOCKER | Gap 1: ledger and search misrouting |
| `internal/indexer/multi.go` | 1059-1070 | Reconcile and ledger not in one writer section on writer-free master paths | WARNING | Gap 2 (history label; D-24) |
| `docs/languages.md` | 279 | "does not depend on timing" overclaims | WARNING | Gap 2 |
| `internal/resolver/cobol_findings.go` | 245-253 | Unconditional `same_line_occurrence` relabel and `EndLine = StartLine` | WARNING | WR-02; bundle with gap 1 |
| `docs/languages.md` | 279 | The mixed-form window understates the collapse (spelling variants, imageless references) | INFO | IN-05 |
| `internal/indexer/cobol_findings_ledger_exclusion_test.go` | 90, 100 | A regression hangs until the package timeout instead of failing | INFO | IN-03 |
| `internal/indexer/cobol_reconcile.go` | 95-96 | Comment claims `finish(false)` keeps the reach index valid | INFO | IN-06 (pre-existing behavior) |
| `internal/indexer/repository_topology_batch.go` | 39-80 | In-batch exclusion relies on `deferResolve` | INFO | IN-04; unreachable in production today |
| `internal/mcp/scope_resolve.go` | 466, 561 | Placeholders drop out of workspace-clamped views | INFO | IN-02; accepted by the 05-16 audit as fail-closed |
| `docs/languages.md` | 279 | Old-store note omits that old external findings stay active beside new ones | INFO | IN-01; developer stores only |

## Human Verification Required

### 1. Judgment prohibition 05-07 (GAP-08)

- **Test:** Review that a cascade can never pass silently or be cleared by AI, and never blocks indexing.
- **Expected:** Confirm HOLDS. Evidence:
  - the cascade gate passes in the default suite;
  - `TestIndexHealth_FindingsDoNotScore` passes;
  - the D-16 writer inventory passes;
  - the graph has no generic remove-by-ID.
- **Why human:** Judgment-tier prohibition about future AI output.

### 2. Judgment prohibition 05-09 (GAP-07)

- **Test:** Review that no MCP, CLI, or AI path can close or alter a finding.
- **Expected:** Confirm HOLDS. Note the residuals:
  - the inventory matches literal write shapes, not computed-key Meta writes;
  - gap 2 can mislabel close versus supersede in history under one race.
- **Why human:** Judgment-tier prohibition.

## Gaps Summary

There are two gaps: one blocking and one partial.

- **Gap 1 (BLOCKER, CR-01, regression from 05-16).**
  - **Defect:** placeholder and finding IDs are parsed at the first `::`.
  - **Effect:** in a repository whose prefix contains `::`, which the configuration accepts:
    - a not-retrieved `COPY IDMS` member keeps its placeholder but loses its EXTERNAL_UNRESOLVED finding on the second unchanged reconcile pass, so it is absent after any cold track;
    - findings and placeholders route to the wrong repository for the ledger and search.
  - **Fix options**, any one of:
    - parse from the last `::cobol-missing::` marker;
    - stamp `cobol_placeholder` on `COPY IDMS` edges;
    - reject `::` in repository prefixes at track time, which also covers the core ID helpers that already misparse such prefixes.
  - **Tests to add:** two regressions, at resolver and real-path level.
  - **Override path (not recommended):** declare `::` prefixes unsupported:

    ```yaml
    overrides:
      - must_have: "a not-retrieved COPY IDMS member keeps its EXTERNAL_UNRESOLVED finding, and finding and placeholder IDs route to their owning repository, for every repository prefix"
        reason: "Repository prefixes containing '::' are unsupported Gortex-wide: the ID grammar <repo_prefix>/<path>::<Symbol> is ambiguous for them and graph.IDFile, graph.EnclosingFromID, stub detection, and MCP rename already misparse them"
        accepted_by: "<name>"
        accepted_at: "<ISO timestamp>"
    ```

    If you take this path, still add `::` to the reserved-name sentence in `docs/languages.md`, or reject it at track time.
- **Gap 2 (partial, WR-01, D-24).**
  - **Defect:** on writer-free master paths (daemon warmup, public deferred catch-up), the master reconciles before taking the reach writer for its ledger. An overlapping edit's settle ledger can then record the master's close as a supersede.
  - **Impact:** the active set and SC4 are unaffected, but the user-decided D-24 MUST and the 05-14 prohibition are not met, and the docs claim the opposite.
  - **Close it** in one of two ways:
    - move the reconcile into the master's writer section and add the interleaving test (review WR-01 option 1);
    - or record an explicit user override of D-24 for these paths and qualify the docs sentence.
- **Bundle with gap 1 (non-blocking):** WR-02, a five-line change in the same resolver function: keep the edge's own absence reason unless it has an exact range, and do not set `EndLine` for later occurrences.
- **Everything else holds:**
  - the previous gap is closed;
  - D-22, D-23, D-25, D-26 (counts), and D-27 (collision, refusal, sweep) are implemented and tested;
  - the cascade gate holds;
  - MCP, CLI, search, and `index_health` exposure work;
  - the full suite passes by override, with WINDOWS.md entry 8 open.

---

_Verified: 2026-10-26T21:14:27Z_
_Verifier: the agent (gsd-verifier)_
