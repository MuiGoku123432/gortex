# Phase 6: Native Deterministic Acceptance and Pre-AI Release Gate - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md -- this log preserves the alternatives considered.

**Date:** 2026-10-26 (areas 1-3, saved in a checkpoint) and 2026-10-27 (area 4, resumed)
**Phase:** 06-native-deterministic-acceptance-and-pre-ai-release-gate
**Areas discussed:** Query surface shape, Lower-bound disclosure, Carried defects, Estate gate & monitoring

Areas 1-3 were recorded in `06-DISCUSS-CHECKPOINT.json`. That checkpoint kept option labels only, so those tables have no option descriptions.

---

## Query surface shape

### How should the Phase 6 native queries be exposed?

| Option | Description | Selected |
|--------|-------------|----------|
| Extend existing | | ✓ |
| One new cobol tool | | |
| One verb per question | | |

**User's choice:** Extend existing. Traversal goes through `get_symbol` / `get_callers` / `get_call_chain` / `find_usages`; new COBOL `analyze` kinds are added on both the MCP `analyze` tool and the `gortex analyze` CLI.

### What does "exact evidence" in a query result contain?

| Option | Description | Selected |
|--------|-------------|----------|
| IDs + ranges, no text | | ✓ |
| Opt-in source snippet | | |
| Always inline text | | |

**User's choice:** IDs, original ranges, statement node ID, and `prov_*`/rule stamps, with no source text. Results are metadata only, outside the P4 D-13 source gate.

### QUERY-09 ranking primary key

| Option | Description | Selected |
|--------|-------------|----------|
| Affected programs | | ✓ |
| Inbound references | | |
| Transitive impact | | |

**User's choice:** Distinct affected programs, then raw reference count, then normalized key.

### QUERY-02 fall-through

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit + disclose | | ✓ |
| Derive at query time | | |

**User's choice:** Explicit transfers only. Every path result discloses "fall-through not modeled" as a boundary.

---

## Lower-bound disclosure

### Disclosure shape

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse lower_bound + boundaries | | ✓ |
| Per-row markers only | | |
| Both | | |

**User's choice:** Reuse the upstream `lower_bound` flag and `boundaries[]` (site ID, reason, edge kind), with COBOL `BoundaryReason` values.

### Which unresolved spots make a floor

| Option | Description | Selected |
|--------|-------------|----------|
| Per-query relevance | | ✓ |
| Strict: anything | | |
| Severity threshold | | |

**User's choice:** A versioned per-query relevance table, filled in by research. Examples: for callers, dynamic CALLs in scope, an ambiguous reference naming the target, and medium or high findings in caller containers; for DB2, SQL tails.

### Placeholders in results

| Option | Description | Selected |
|--------|-------------|----------|
| Tagged row + boundary | | ✓ |
| Boundary only | | |

**User's choice:** A row tagged `resolution=missing` in reference-style results, also listed in `boundaries[]`. Never in present-artifact lists or implementation counts.

---

## Carried defects

### WR-08 nested/sibling CALL visibility

| Option | Description | Selected |
|--------|-------------|----------|
| Fix in Phase 6 | | ✓ |
| Accept + document | | |
| Mark, don't fix | | |

**User's choice:** Fix in Phase 6. Visibility is containment-based; a sibling binds only when COMMON is proven, otherwise the call stays unresolved with a distinct reason.

### D-20 failed program analysis

| Option | Description | Selected |
|--------|-------------|----------|
| Disclose as boundary | | ✓ |
| Lexical PROGRAM-ID node | | |
| Leave as documented | | |

**User's choice:** Disclose as a boundary. Failed-analysis files in scope make caller and missing-artifact results lower bounds; QUERY-09 flags possible presence; monitoring counts them. No name guessing.

### Phase 4 human items

| Option | Description | Selected |
|--------|-------------|----------|
| Lifecycle in, refactor out | | ✓ |
| Both in | | |
| Both out | | |

**User's choice:** Lifecycle WR-06/WR-07 are fixed in Phase 6 (SC4 interrupted retry). The refactor-tool scope policy goes to the backlog.

### Phase 5 re-verification relation

| Option | Description | Selected |
|--------|-------------|----------|
| Hard precondition | | ✓ |
| Fold residuals in | | |

**User's choice:** Hard precondition. Phase 6 is planned only after 05-17 completes and Phase 5 re-verifies as passed; new Phase 5 gaps are fixed in Phase 5. (Met on 2026-10-26.)

---

## Estate gate & monitoring

### Does the pre-AI release gate depend on a run against the real DCC code?

| Option | Description | Selected |
|--------|-------------|----------|
| Fixtures gate, estate baseline | Pass/fail rests only on committed invented fixtures. One AI-off estate run must complete, and its aggregate-only monitoring report is recorded as the Phase 7 baseline. | ✓ |
| Estate thresholds gate too | The gate also fails on estate conditions (zero unowned nodes, zero program analysis failures, zero uncontained cascades); ties release to a private corpus no CI can run. | |
| Fixtures only | No estate run in Phase 6. | |

**User's choice:** Fixtures gate, estate baseline.

### What counts as "all deterministic release gates pass" before Phase 7?

| Option | Description | Selected |
|--------|-------------|----------|
| Local + CI, all OS | Local race suite plus green fork CI on ubuntu, macOS, and Windows (adds lint, tidy, BASE-02, and the static build); requires pushing the branch. | ✓ |
| Local + CI, Windows tracked | Same, but Windows stays an open window outside the gate. | |
| Local suite only | As in Phase 5; no push or CI requirement. | |

**User's choice:** Local + CI, all OS.

### Where should EVAL-06 monitoring live?

| Option | Description | Selected |
|--------|-------------|----------|
| Extend index_health | By-reason and by-severity counts under the findings block, plus a classified extraction-failure summary; `gortex status` shows the same; the QUERY-08 kind is the drill-down. | ✓ |
| QUERY-08 kind is the report | No new health fields; the QUERY-08 aggregate is the monitoring report. | |
| Health summary + ledger deltas | Option 1 plus opened/closed/superseded since the last index, from the best-effort ledger. | |

**User's choice:** Extend index_health.

### How should monitoring give extraction failures a class and reason?

| Option | Description | Selected |
|--------|-------------|----------|
| Reason code at record | A versioned reason set stored next to the free-text Error in the `file_index_failures` sidecar (schema change). | ✓ |
| Classify at read time | Map the stored Error text to reasons when reporting; brittle if the wording changes. | |
| By language only | Count by language or kind and keep the free-text samples. | |

**User's choice:** Reason code at record.

### What does the estate baseline contain, and where does it live?

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated baseline file | Committed `06-ESTATE-BASELINE.md`: the monitoring block plus run identity (commit, grammar ID, extractor version, file counts, node/edge totals, wall time); aggregate only. | ✓ |
| Inside VERIFICATION only | The same numbers as gate evidence inside `06-VERIFICATION.md`. | |
| Monitoring block only | The raw `index_health` block, without run identity. | |

**User's choice:** Dedicated baseline file.

### How does the gate prove it ran with AI disabled?

| Option | Description | Selected |
|--------|-------------|----------|
| Isolate config + assert none | Isolated config home with no `llm` block, `GORTEX_LLM_*` cleared, and a runtime assertion that no provider was constructed. | ✓ |
| Keep the env convention | Each test sets `GORTEX_LLM_PROVIDER=""`, as today. | |

**User's choice:** Isolate config + assert none.

### How is the pre-AI gate run and enforced?

| Option | Description | Selected |
|--------|-------------|----------|
| Registry gate test | A Go test lists every Phase 6 acceptance case and fails if any is missing or skipped. | ✓ |
| Registry test + make target | Same, plus a `make pre-ai-gate` target with an activity pre-check. | |
| Documented checklist | The verifier runs the listed commands, as in Phase 5. | |

**User's choice:** Registry gate test.

### How are the SC1 acceptance fixtures organized?

| Option | Description | Selected |
|--------|-------------|----------|
| Golden dumps per category | One table-driven suite, one committed fixture per SC1 category under testdata, compared to a committed golden canonical dump; `-update` flag. | ✓ |
| Extend existing tests | Add missing cases to the per-phase tests with inline fixtures; the registry maps categories to test names. | |

**User's choice:** Golden dumps per category.

### When does the branch get pushed?

| Option | Description | Selected |
|--------|-------------|----------|
| You push before planning | The user pushes before `/gsd-plan-phase 6`; CI failures become research input. | ✓ |
| Push as Phase 6's first plan | Plan 06-01 pushes, with authorization, and records CI. | |
| Push only at the gate | One push at the end of Phase 6. | |

**User's choice:** You push before planning.

### What does Phase 6 do about the cmd/gortex TestMain query-log guard?

| Option | Description | Selected |
|--------|-------------|----------|
| Narrow the guard | Check upstream first; otherwise fail only on query-log lines pointing into the test's temp/sandbox dirs. | ✓ |
| Keep the no-activity rule | Upstream's position; every local gate run needs all Gortex sessions stopped. | |
| Drop the query log check | One-line diff; weakest check. | |

**User's choice:** Narrow the guard.

### How should the Windows script test failure be fixed?

| Option | Description | Selected |
|--------|-------------|----------|
| Skip on Windows | `TestNeo4jScript*` skip on Windows with a stated reason (bash + Docker developer harness). | ✓ |
| Make it portable | Run through Git Bash with converted paths and `.sh` fakes. | |

**User's choice:** Skip on Windows.

### If the DCC baseline run turns up defects, which ones does Phase 6 fix?

| Option | Description | Selected |
|--------|-------------|----------|
| Fix completion blockers only | Crash, hang, or a non-finishing index is fixed with an invented reproducing fixture; count-level concerns go to a ranked follow-up list. | ✓ |
| Fix every confirmed defect | Every confirmed defect is fixed in Phase 6. | |

**User's choice:** Fix completion blockers only.

**Notes:**
- In the resumed session, the question "Should findings or failures change a repo's health status?" was listed but replaced. P5 D-14 and P4 D-15 already decide it.
- At wrap-up the user chose "I'm ready for context" over exploring the remaining open items. Those are recorded as undecided in CONTEXT.md.

---

## the agent's Discretion

- Names and grouping of the new `analyze` kinds and their result fields
- COBOL `BoundaryReason` names and the per-query relevance table contents
- The distinct reason name for visibility-hidden programs, and how `COMMON` is detected
- The failure reason set and migration mechanics
- How the registry gate test detects missing or skipped cases
- Layout of the extended `index_health` block and the `gortex status` summary
- How the golden `-update` flag is wired
- Whether the SC1 suite also asserts the SEC-08 ownership audit

## Deferred Ideas

- Query-time fall-through derivation (behavioral twin stage)
- Ranking missing artifacts by transitive impact
- Opt-in source snippets in COBOL query results
- Refactor-tool out-of-scope policy for `move_symbol` / `inline_symbol` / `rename_symbol` (Phase 4 WR-04/IN-06/WR-08): backlog
- A `graph digest` CLI/MCP verb (Phase 4 deferred): not needed for Phase 6 monitoring
