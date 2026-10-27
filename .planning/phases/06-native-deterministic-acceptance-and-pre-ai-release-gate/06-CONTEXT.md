# Phase 6: Native Deterministic Acceptance and Pre-AI Release Gate - Context

**Gathered:** 2026-10-27
**Status:** Ready for planning, after the user pushes the branch (D-13)

<domain>
## Phase Boundary

Users can answer the required native graph questions about a COBOL estate through the existing CLI and MCP surfaces, with exact evidence and an honest statement when a result is a lower bound:

- programs and copybooks (QUERY-01)
- paragraph execution paths (QUERY-02)
- callers of a program (QUERY-03)
- programs depending on a copybook (QUERY-04)
- data items by container (QUERY-05)
- DB2 reads and writes (QUERY-06)
- IDMS and CICS usage (QUERY-07)
- unresolved findings, filtered and aggregated (QUERY-08)
- unavailable artifacts ranked by impact (QUERY-09)

The deterministic system then proves, with AI disabled, that it meets its reproducibility, lifecycle, monitoring, and unresolved-boundary invariants. A fixture-based gate runs locally and in fork CI on all three operating systems before Phase 7 begins.

Requirements: BASE-04, QUERY-01..QUERY-10, EVAL-01, EVAL-02, EVAL-03, EVAL-06.

Also in scope (carried defects, decided in this discussion):
- **P5 review WR-08 / P3 WR-08:** COBOL CALL visibility. Programs hidden by containment are reported as "not obtained" today (D-08).
- **P5 D-20:** a program whose analysis fails can make its callers resolve to `missing` (D-09).
- **P4 WR-06 and WR-07:** lifecycle failure paths (D-10).
- **CI blockers on the gate path:** the Windows script tests (D-18), the `cmd/gortex` TestMain guard (D-17), and whatever the pre-planning push surfaces (D-13).

Out of this phase:
- QUERY-11 (AI-inferred, human-confirmed, contradicted, superseded outcomes): Phase 8.
- AI policy, context, claims, review, and AI evaluation: Phases 7-9.
- Estate-scale thresholds as a gate condition (D-19).
- Phase 6.1 (VB.NET), full JCL, behavioral or live twin.

Depends on Phase 5. That precondition is met: Phase 5 completed 2026-10-26, verification passed, UAT 3/3.

</domain>

<decisions>
## Implementation Decisions

### Carried forward (do not re-decide)
- Fixtures are invented COBOL only, because the repo is public (P2 D-14). No estate content, program names, or paths are committed anywhere, including the estate baseline.
- "Canonical active deterministic subgraph" and the dump are defined by P4 D-01..D-04: `internal/testutil/graphdump`, with a versioned volatile allowlist. Restore reproduces byte-identical rows.
- Source reads fail closed on missing scope (P4 D-13). Query evidence in this phase carries no source text, so it is outside that gate (D-02).
- Health posture (P4 D-15, P5 D-14):
  - Ownership violations make `index_health` unhealthy.
  - Findings and parser cascades never do.
  - Nothing blocks queries.
- `cobolprobe` is a flag-gated measurement harness, not the gate (P5 D-15).
- A finding closes only through deterministic evidence; no tool alters one (P5 D-16).
- One `PARSE_UNRESOLVED` finding per producer tail token (P5 D-18). Counts are sized to per-token volume.
- Finding classes, statuses, Meta keys, and rule IDs live in `internal/graph/finding.go`. Queries filter on those keys and never parse IDs.
- Unbound COBOL edges keep their `cobol_unresolved_reason` vocabulary (P3 D-12). `ambiguous` and `dynamic` stay edge reasons, not findings (P5 D-09).
- Binding is an exact single match in the same repository plus the allowed-repository union (P3 D-09/D-11, P4 D-09/D-19).
- One COBOL extractor version bump per phase. The current version is `gortex-cobol-grammar/6`. If Phase 6 changes emitted COBOL Meta, bump to `/7` once.

### Query surface (QUERY-01..QUERY-07, QUERY-09)
- **D-01:** Extend existing surfaces; add no new tool.
  - Traversal questions (callers, paragraph paths, copybook dependents, usages) go through `get_symbol`, `get_callers`, `get_call_chain`, and `find_usages`.
  - Questions without an existing verb become new COBOL `analyze` kinds. Those questions are present programs and copybooks, data items by container, DB2 reads and writes, IDMS and CICS usage, findings filter and aggregate, and missing-artifact ranking.
  - Each kind is registered on both the MCP `analyze` tool and the `gortex analyze` CLI, with one shared handler and one result shape.
  - — **Reversibility:** costly — kind names and result fields become a CLI/MCP contract that Phases 7-9 and user scripts consume.
- **D-02:** "Exact evidence" in a result has these parts:
  - stable IDs
  - original source ranges
  - the statement node ID
  - the `prov_*` and rule-ID stamps

  It never includes source text. Results are metadata only.
- **D-03:** QUERY-09 ranks unavailable artifacts by distinct affected programs (descending), then raw inbound reference count (descending), then normalized key (ascending).
- **D-04:** QUERY-02 paragraph paths follow explicit transfers only (`PERFORM`, `THRU`, `GO TO` edges). Every path result discloses "fall-through not modeled" as a boundary.

### Lower-bound disclosure (QUERY-10, SC3)
- **D-05:** Reuse the upstream `lower_bound` flag and `boundaries[]` list from `internal/graph/epistemic_boundary.go`. Each boundary carries the site ID, reason, and edge kind, and the list is capped at `maxBoundaries` (50). Add COBOL `BoundaryReason` values; research names them.
  - Examples: dynamic call, ambiguous target, missing artifact, parse finding in scope, failed analysis, fall-through not modeled.
  - — **Reversibility:** costly — reason strings become part of every query result contract.
- **D-06:** Which unresolved spots make a result a lower bound is a **per-query relevance table**. The table is versioned (rule ID, like other rules), and research fills it in. A spot that cannot affect a query's answer must not set `lower_bound` on that query.
  - Callers: dynamic CALLs in scope, an `ambiguous` reference naming the target, and medium or high findings in caller containers.
  - DB2: SQL unparsed tails.
- **D-07:** A placeholder (missing artifact) appears only in reference-style results (callers, dependents, usages):
  - as a row tagged `resolution=missing`;
  - listed in `boundaries[]`.

  It never appears in present-artifact lists (QUERY-01) or in implementation counts.

### Carried defects
- **D-08:** Fix COBOL CALL visibility in Phase 6. History:
  - Commit `9f6c1973` (2026-10-12) fixed the false binding P3 WR-08 found: nested candidates now bind only to the program that directly contains them (`internal/resolver/cobol_references.go`, the `contained` / `sameFile` split in `decideCobol`).
  - Phase 3 has not been re-verified since that commit.
  - The remaining defect is P5 review WR-08: a program that exists but is hidden by visibility resolves `missing`, which gets an `EXTERNAL_UNRESOLVED` finding and a "not obtained" placeholder.

  Decision:
  - Containment-based visibility stays.
  - A sibling (nested in the same container) binds only when `COMMON` is proven from producer evidence.
  - Any other hidden program resolves unresolved with a distinct reason. It is never `missing`, gets no placeholder, and gets no `EXTERNAL_UNRESOLVED` finding. The existing `containment_unresolved` (IN-08) is the precedent for a distinct reason.
  - If the pinned producer exposes no `COMMON` observation, sibling calls fail closed with that distinct reason. The extractor has no `COMMON` handling today.
- **D-09:** P5 D-20 (program analysis failure) is disclosed as a boundary:
  - A failed-analysis file in scope makes caller and missing-artifact results lower bounds.
  - QUERY-09 flags an affected missing artifact as possibly present.
  - Monitoring counts these failures (D-22, D-23).
  - No program identity is guessed from a file name.
- **D-10:** Fix P4 WR-06 and WR-07 in Phase 6, as part of SC4's interrupted-retry proof:
  - **WR-06:** declared roots over-prune when a declared repository fails to register, and nothing re-admits them.
  - **WR-07:** an enclosing-reconcile failure is only logged, while track and untrack report success.

  P4 human item 1 (refactor-tool policy for out-of-scope references in `rename_symbol` / `move_symbol` / `inline_symbol`, i.e. P4 WR-04/IN-06/WR-08) goes to the backlog.
- **D-11:** Phase 5 completion was a hard precondition, and it is met. If a new Phase 5 gap surfaces, it is fixed in Phase 5, not folded into Phase 6.

### Release gate (SC1, SC4, SC5)
- **D-12:** "All deterministic release gates pass" means both of these, on the same code:
  - the local `GOWORK=off go test -race ./...` (the Phase 5 command);
  - a green fork CI run on ubuntu, macOS, and Windows. CI adds lint, the `go.mod` tidy check, the BASE-02 stock-grammar fail-closed step, and the static build. None of these has run on the Phase 3-5 code.
- **D-13:** The user pushes `planning/ai-enhanced-cobol-graph` to `origin` **before** `/gsd-plan-phase 6`. The branch was 362 commits ahead at discussion time, and fork CI last ran 2026-10-01. Failures from that CI run become research input and get their own fix plans. Agents never push without explicit authorization, including for the final gate run.
- **D-14:** A **registry gate test** enforces the gate. It lists every Phase 6 acceptance case (SC1-SC5) and fails if any listed case is missing or skipped, so a plain `go test -race ./...` enforces the gate locally and in CI. It reuses the Phase 1 pattern of failing while any required scenario is skipped.
- **D-15:** SC1 fixtures are **golden dumps per category**:
  - One table-driven Phase 6 acceptance suite holds one committed invented fixture per SC1 category, as `.cbl`/`.cpy` files under `testdata`.
  - Categories: COBOL structure, DATA DIVISION, IDMS, CICS, SQL, bounded unparsed tail, `ERROR`, `MISSING`, missing artifact.
  - Fixtures are indexed on the real path (registered extractor, indexer, SQLite temp store).
  - Each result is compared to a committed golden canonical dump covering IDs and provenance (EVAL-02).
  - Goldens regenerate through an `-update` flag, so each change shows up as a reviewable diff.
- **D-16:** The gate proves AI is off instead of assuming it:
  - acceptance runs with an isolated Gortex config home that has no `llm` block;
  - all `GORTEX_LLM_*` variables are cleared;
  - a runtime assertion checks that no LLM provider was constructed (for example, `ask` stays unregistered).

  This replaces relying on `GORTEX_LLM_PROVIDER=""` alone (the TRACE-07 convention in `cmd/gortex/cobol_tracer_test.go`).
- **D-17:** Narrow the `cmd/gortex` TestMain guard (`cmd/gortex/main_test.go`). First check upstream `zzet/gortex` for an existing fix.
  - Fail on the query log only when lines added during the run point their `repo` or `corpus` fields into the test's temp or sandbox directories.
  - Every other real per-user path keeps today's fingerprint check.

  Goal: agents can keep using Gortex while the gate runs. The 05-17 gate tripped on this three times.
- **D-18:** The `TestNeo4jScript*` tests in `internal/neo4jprojection/neo4j_integration_test.go` skip on Windows, with a stated reason: `scripts/test-neo4j.sh` is a bash + Docker developer harness, and the argument handling those tests check still runs on Linux and macOS. This was the only known Windows CI failure (run 36909797580, exit 127). Anything else the push surfaces on Windows gets fixed.

### Estate run
- **D-19:** Pass/fail rests only on committed invented fixtures (EVAL-01). One AI-off index of the DCC slice must complete (5,169 files; local location in `PythonApps/cobol-kg/HANDOFF.md`). Completion is the only estate-dependent gate condition, and nothing is thresholded on the run's counts.
- **D-20:** Only completion blockers from the estate run are fixed in Phase 6: a crash, a hang, or an index that won't finish. Each fix comes with an invented fixture that reproduces it, and no estate content is committed. Concerns visible only in the counts go into the baseline file as a ranked follow-up list.
- **D-21:** The estate baseline is a committed `06-ESTATE-BASELINE.md` in this phase directory. It contains:
  - the `index_health` monitoring block as captured (D-22);
  - run identity: gortex commit, grammar ID, extractor version, file counts by language, node and edge totals by kind, wall time.

  It is aggregate only: no names, paths, or source. Phases 7-9 compare against it.

### Monitoring (EVAL-06)
- **D-22:** Extend `index_health`:
  - Its findings block (today counts by class only, `internal/mcp/tools_enhancements.go` near the `parseFindings` / `externalFindings` loop) gains counts by reason and by severity within each class.
  - Add a classified extraction-failure summary, by language or artifact kind and by reason.
  - `gortex status` shows the same summary.
  - The QUERY-08 `analyze` kind is the drill-down (container, parser version, lifecycle filters).
- **D-23:** Extraction failures get a **reason code at record time**: a small, versioned reason set stored next to the free-text `Error` in the `file_index_failures` sidecar (`internal/graph/file_index_failures.go`, `FileIndexFailure`). Research sets the final reason set; candidates are `analysis-error`, `grammar-not-approved`, `unreadable`, `read-error`, `other`. This needs a SQLite schema change, following the in-place migration precedent of `finding_history` (v23).
  - — **Reversibility:** costly — a persisted column and migration; the reason codes become a monitoring contract compared against the estate baseline.

### the agent's Discretion
- Names and grouping of the new `analyze` kinds (one per question or fewer kinds with modes), and their result field names, as long as CLI and MCP share one handler (D-01).
- COBOL `BoundaryReason` names and the contents of the per-query relevance table (D-05, D-06). Research proposes them.
- The distinct unresolved reason name for visibility-hidden programs, and how `COMMON` is detected (D-08).
- The final failure reason set and the migration mechanics (D-23).
- How the registry gate test detects missing or skipped cases (D-14).
- Layout of the extended `index_health` block and the `gortex status` summary (D-22).
- How the golden `-update` flag is wired (D-15).
- Whether the SC1 suite also asserts the SEC-08 ownership audit on its corpus (cheap; recommended, since SEC-08 names "the acceptance corpus").

### Open items (not decided; raise at plan time)
The user chose to proceed without discussing these. Planning must surface each one rather than silently include or drop it.
- **Open WINDOWS.md entries:** #1 (Neo4j real-server scenarios registered, still listed open) and #3 (`release.yml` has never run). Does the gate require them closed?
- **Phase 4 deferred item (04-15):** a stale `cross_repo_calls` parallel edge remains when a cross-repository COBOL CALL unbinds without eviction. This is a risk to SC4 equivalence; research should check whether the SC4 sequences hit it.
- **Phase 5 deferred item (05-16):** shared DB2/IDMS resource nodes still use ID = FilePath and are swept with `EvictFile`, the WR-05 hazard that D-27 removed for placeholders only.
- **Refusing export or AI context on unhealthy ownership:** P4 deferred this to "Phases 6 and 7". It is not in Phase 6's success criteria. The recommendation is Phase 7 (AI context) and Phase 9 (export); the user has not decided.
- **Phase 3 and 4 status (found while writing this context, not raised in the discussion):**
  - Phase 3 is still `gaps_found` and has not been re-verified since `9f6c1973`.
  - Phase 4 is `human_needed`.
  - REQUIREMENTS.md still lists DET-01..12, ID-03..09, SEC-01..03, and SEC-08 as "Gaps Found".
  - Open question: does the pre-AI gate include re-verifying Phases 3 and 4 after the Phase 6 fixes, and updating that traceability?

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` §"Phase 6: Native Deterministic Acceptance and Pre-AI Release Gate": goal and the five success criteria
- `.planning/REQUIREMENTS.md`: BASE-04, QUERY-01..QUERY-10, EVAL-01, EVAL-02, EVAL-03, EVAL-06; SEC-08 (acceptance-corpus ownership); Out of Scope list
- `.planning/PROJECT.md`: product principle and the Key Decisions table

### Prior decisions and carried defects
- `.planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-VERIFICATION.md`: the original P3 WR-08 gap, its probe results, and the deferred-items table
- `.planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-CONTEXT.md`: D-01..D-04 (canonical dump), D-13 (source gate), D-15 (health posture)
- `.planning/phases/04-stable-incremental-and-cross-repository-lifecycle/04-VERIFICATION.md`: human items (WR-06, WR-07, refactor-tool policy)
- `.planning/phases/04-stable-incremental-and-cross-repository-lifecycle/deferred-items.md`: the stale `cross_repo_calls` edge (04-15), and `internal/indexer/file_delta.go` not gofmt-clean
- `.planning/phases/05-explicit-unresolved-evidence/05-CONTEXT.md`: D-09, D-14..D-16, D-18, D-20, and the gap-closure amendments (P5 WR-08 deferred to Phase 6)
- `.planning/phases/05-explicit-unresolved-evidence/05-REVIEW.md`: WR-08 ("visibility-hidden programs reported as not obtained")
- `.planning/phases/05-explicit-unresolved-evidence/05-VERIFICATION.md`: Phase 5 passed (the D-11 precondition)
- `.planning/phases/05-explicit-unresolved-evidence/deferred-items.md`: shared-resource ID hazard (05-16), TestMain guard analysis (05-17), `csharp_verbatim_type_ref_test.go` not gofmt-clean
- `.planning/WINDOWS.md`: open entries #1, #2 (Windows CI), #3
- `.planning/STATE.md` §Decisions: Phase 2-5 implementation decisions (COBOL IDs, nesting stamps, resolver rules, finding writers)

### Research background
- `.planning/research/COBOL-PARSER-READINESS.md`: producer handoff API, and the baseline behaviors BASE-04 must verify
- `.planning/research/GRAPH-SCHEMA.md`: concept mapping and provenance vocabulary
- `.planning/research/PITFALLS.md`: Pitfall 2 (stale facts), Pitfall 5 (missing vs parse gaps), Pitfall 6 (scope leakage)
- `.planning/research/THREAT-MODEL.md`: T1 and T4. No source in monitoring output, logs, or the estate baseline.
- `docs/languages.md`: the COBOL row (documented limitations, including P5 D-20 and the mixed-form same-line window)
- `internal/parser/forest/cobolprobe/README.md`: how to run the measurement harness against a local corpus

### Code entry points
- `internal/graph/epistemic_boundary.go`: `EpistemicBoundary`, `BoundaryReason`, `maxBoundaries`, `ClassifyDroppedTarget`
- `internal/mcp/analyze_kinds.go`: the `analyze` kind registry
- `cmd/gortex/analyze.go`: the CLI `analyze` command
- `internal/mcp/tools_enhancements.go`: `index_health`, including the findings count loop
- `internal/mcp/index_file_failures.go`: `indexFileFailureSummary`, `readIndexFileFailures`
- `internal/graph/file_index_failures.go`: `FileIndexFailure` and the sidecar interfaces; SQLite in `internal/graph/store_sqlite/store_file_index_failures.go`; recording in `internal/indexer/file_index_failures.go`
- `internal/graph/finding.go`: finding classes, statuses, Meta keys, rule IDs
- `internal/resolver/cobol_references.go`: `decideCobol`, the program visibility candidate split (D-08)
- `internal/indexer/multi.go`: the enclosing-reconcile error path (P4 WR-07) and declared-root handling (P4 WR-06)
- `internal/testutil/graphdump/graphdump.go`: `Dump`, `Volatile`
- `cmd/gortex/main_test.go`: the TestMain guard (`realUserStatePaths`, `diffUserState`)
- `internal/neo4jprojection/neo4j_integration_test.go`: the `TestNeo4jScript*` tests
- `cmd/gortex/status.go`, `cmd/gortex/daemon_index_health.go`: the `gortex status` surface
- `.github/workflows/ci.yml`: the CI matrix and steps the gate depends on

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`graphdump.Dump` with its `Volatile` allowlist:** golden dumps for SC1 (D-15) and the SC4 lifecycle equivalence comparisons.
- **`EpistemicBoundary` / `BoundaryReason`:** already drive `lower_bound` and `boundaries[]` in the impact, explore, and core tools. COBOL adds reasons rather than a new shape.
- **`index_health` findings loop:** counts by `prov_evidence_class` today. Extend it in place with reason and severity (D-22).
- **`file_index_failures` sidecar:** exists across the graph, SQLite, overlay, and graphview layers. D-23 adds one field through those layers.
- **Existing COBOL acceptance tests:** many existing SC1/SC4 cases can be referenced by the registry (D-14) instead of rewritten.
  - `cmd/gortex/cobol_{tracer,breadth,findings,scope}_test.go`
  - `internal/indexer/cobol_lifecycle_*_test.go`
  - `internal/indexer/cobol_findings_*_test.go`
  - `internal/indexer/cobol_cascade_gate_test.go`
- **`serveCobolTestDaemon`:** the shared daemon-over-relay helper for COBOL CLI/MCP tests.
- **The Phase 1 required-scenario gate in `internal/neo4jprojection`:** the precedent for "fail while any required case is skipped" (D-14).
- **The `resolution_outcomes` analyze kind:** an existing generic why-unresolved taxonomy, a pattern reference for the QUERY-08 kind.

### Established Patterns
- One request/result contract shared by CLI and MCP (Phase 1 `neo4jprojection.NormalizeRequest`).
- Versioned rule IDs recorded in Meta for every derived rule (`gortex-finding-id/2`, `cobol-name-norm/1`, severity and evidence rules). The D-06 relevance table and the D-23 reason set follow this.
- One COBOL extractor-version bump per phase.
- In-place SQLite migrations (`finding_history`, v23).
- Real-path tests: registered extractor, indexer, SQLite temp store, and `AddBatch` (not `BulkLoad`), with AI providers off.
- Fork hygiene: keep diffs to upstream files minimal and prefer additive files. This applies to D-17 (an upstream test file) and D-18.

### Integration Points
- MCP `analyze` dispatcher plus `cmd/gortex/analyze.go`, for the new COBOL kinds.
- Result envelopes of `get_callers`, `get_call_chain`, and `find_usages`, for `lower_bound`, `boundaries[]`, and `resolution=missing` rows.
- The `index_health` handler and `gortex status`, for monitoring.
- `Indexer.flushFileIndexFailures` and the failure-recording path, for reason codes.
- `Resolver.decideCobol`, for program visibility (D-08).
- MultiIndexer track/untrack/declared-root lifecycle, for WR-06 and WR-07.

</code_context>

<specifics>
## Specific Ideas

- The 05-17 gate was accepted on split evidence because outside Gortex activity tripped the TestMain guard. D-17 exists so that Phase 6's gate passes literally.
- Expect the pre-planning push to fail at least on:
  - Windows (`TestNeo4jScript*`);
  - lint (two files are not gofmt-clean: `internal/indexer/file_delta.go` and `internal/resolver/csharp_verbatim_type_ref_test.go`).
- On real code, dynamic CALLs outnumbered literal ones (829 vs 352 in the regex-era DCC measurement). Lower-bound disclosure on caller results will therefore be common on the estate. That is expected, not a defect.
- The estate baseline is a monitoring reference. AI-on/off equality (EVAL-09, Phase 9) compares canonical dumps, not the baseline.

</specifics>

<deferred>
## Deferred Ideas

- Query-time fall-through derivation for paragraph paths: belongs to the behavioral stage (FUT-04).
- Ranking missing artifacts by transitive impact.
- Opt-in source snippets in COBOL query results.
- Refactor-tool policy for out-of-scope references in `rename_symbol` / `move_symbol` / `inline_symbol` (P4 WR-04/IN-06/WR-08): backlog.
- A CLI/MCP `graph digest` verb (P4 deferred it "for Phase 6 monitoring if needed"). It is not needed: monitoring goes through `index_health` (D-22), and golden dumps stay test-only (D-15).

</deferred>

---

*Phase: 06-native-deterministic-acceptance-and-pre-ai-release-gate*
*Context gathered: 2026-10-27*
