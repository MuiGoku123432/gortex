---
phase: 03-deterministic-cobol-and-mainframe-breadth
verified: 2026-10-12T16:14:27Z
status: gaps_found
score: 16/17 must-haves verified
covered_files:
  - .planning/REQUIREMENTS.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-01-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-01-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-02-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-02-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-03-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-03-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-04-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-04-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-05-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-05-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-06-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-06-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-07-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-07-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-08-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-08-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-09-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-09-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-10-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-10-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-11-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-11-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-12-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-12-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-13-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-13-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-14-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-14-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-15-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-15-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-16-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-16-SUMMARY.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-17-PLAN.md
  - .planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-17-SUMMARY.md
  - cmd/gortex/cobol_breadth_test.go
  - docs/languages.md
  - go.mod
  - internal/indexer/extractor_version.go
  - internal/indexer/incremental_resolve.go
  - internal/indexer/source_revision.go
  - internal/parser/languages/cobol_grammar.go
  - internal/parser/languages/cobol_grammar_copybook.go
  - internal/parser/languages/cobol_grammar_data.go
  - internal/parser/languages/cobol_grammar_data_clauses.go
  - internal/parser/languages/cobol_grammar_data_refs.go
  - internal/parser/languages/cobol_grammar_dedupe.go
  - internal/parser/languages/cobol_grammar_flow.go
  - internal/parser/languages/cobol_grammar_identity.go
  - internal/parser/languages/cobol_grammar_mainframe.go
  - internal/parser/languages/cobol_grammar_procedure.go
  - internal/parser/languages/cobol_grammar_refs.go
  - internal/parser/languages/cobol_grammar_sql.go
  - internal/parser/languages/cobol_grammar_statements.go
  - internal/parser/languages/cobol_grammar_unsupported.go
  - internal/parser/languages/cobol_grammar_walk.go
  - internal/resolver/cobol_references.go
  - internal/resolver/cross_repo.go
  - internal/resolver/resolver.go
covered_digest: "v1:sha256:018f509a36dab243597a9f2c71788210bcb890946864ed188ff319eab4fab401"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 11/13
  gaps_closed:
    - "DET-05 / SC3: copybook and include references keep the referenced member name and source occurrence, including COPY statements inside an analyzed copybook (03-15)"
    - "Human item 1: procedure-fragment copybooks now emit sections, paragraphs, statements, flow, CALL, COPY, IDMS, CICS, and SQL edges (03-16, 03-17; user chose fix now)"
    - "Gap 1, cross-file part: a CALL from another file never binds to a nested program (03-14)"
  gaps_remaining:
    - "Gap 1, same-file part: a literal CALL is bound only to a program a separately compiled caller can actually reach. A top-level program in a multi-program (batch) file still binds to a program nested inside a different top-level program of the same file (review WR-08)"
  regressions: []
gaps:
  - truth: "Phase goal clause 'without false deterministic resolution' (carried from previous gap 1): a literal CALL is bound only to a program a separately compiled caller can actually reach"
    status: partial
    reason: "03-14 closed the cross-file case. The same-file branch of resolveCobolReferences still treats every program in the caller's file as visible, whatever its containment. Non-mutating overlay probe through the real MultiIndexer, SQLite store, and production resolver: in cobol/batch.cbl (top-level A containing UTIL, then top-level B), B's CALL 'UTIL' binds to cobol/batch.cbl::A/UTIL with no cobol_unresolved_reason. That happens even though a top-level UTIL exists in cobol/util.cbl: the non-empty same-file set shadows it. COBOL scoping makes A/UTIL unreachable from B. In a second case, P contains X and Y (X not COMMON), and Y's CALL 'X' binds to P/X. Both are deterministic bindings the phase goal forbids (review WR-08)."
    artifacts:
      - path: "internal/resolver/cobol_references.go"
        issue: "lines 92-106: every candidate with c.FilePath == e.FilePath goes into sameFile, regardless of cobol_nested or containment, and sameFile replaces the top-level set whenever it is non-empty"
      - path: "internal/parser/languages/cobol_grammar.go"
        issue: "program nodes carry cobol_nested but not their container, so the resolver cannot test visibility. CALL edges do not record the caller program"
    missing:
      - "Decide visibility from containment, not file equality. A nested candidate is visible only to its direct container. Bind a sibling only when COMMON is proven; if the producer does not expose IS COMMON, a sibling stays unresolved with a distinct reason, never bound. Either stamp cobol_container on program nodes and the caller program on CALL edges, or bind visible nested callees in the extractor and keep nested programs out of the resolver index entirely"
      - "Same-file top-level programs join the ordinary top-level candidate set (other files included), so a real top-level target is never shadowed"
      - "Tests: a batch file (A contains UTIL; top-level B calls UTIL; top-level UTIL in another file) where B's CALL binds to the other file's UTIL, or is missing or ambiguous, and never binds to A/UTIL. A non-COMMON sibling call is not bound. Both a resolver unit test and a TestCobolBreadth real-path case"
deferred:
  - truth: "Incremental re-index keeps bound COBOL CALL/COPY edges in the cobol:: namespace and does not drop inbound COPY edges, now including copybook-to-copybook imports and copybook CALLs (review CR-01, WR-01)"
    addressed_in: "Phase 4"
    evidence: "Phase 4 SC3 'Edit, delete, restore ... changes remove or invalidate stale owned observations', SC4 'Incremental reconciliation and a clean rebuild produce the same canonical active deterministic subgraph', SC5 'Resolution ... fail closed ... reject unsupported same-name ... binding'"
  - truth: "Shared DB2/IDMS/CICS resource nodes survive eviction of the writing file, and their per-document prov_* fields do not flip between program and copybook writers (review CR-02, scope widened by 03-17)"
    addressed_in: "Phase 4"
    evidence: "Phase 4 SC1 'namespaced resources ... retain stable, non-colliding identities and distinguishable provenance', SC3, SC4"
  - truth: "Unprefixed per-program unresolved placeholders (dynamic/label/data) cannot collide across repositories with the same file path"
    addressed_in: "Phase 4"
    evidence: "Phase 4 SC5 'report zero unowned or misprefixed active nodes for the acceptance corpus'"
  - truth: "COBOL mapping changes invalidate stale stores: the extractor version is bumped past gortex-cobol-grammar/3, .cbl/.cob/.cpy carry an extractor-version salt, and a program node with no cobol_nested key is never a cross-file candidate (review WR-10)"
    addressed_in: "Phase 4"
    evidence: "Phase 4 SC3 '... parser-version, extractor-version ... changes remove or invalidate stale owned observations' (ID-08); SC5 'Resolution and all downstream surfaces fail closed on missing scope'"
  - truth: "Copybook analysis failure is distinguishable from a missing copybook (review WR-06), and SQL unparsed tails, WORKING-STORAGE DECLARE CURSOR, and DECLARE ... WITH HOLD produce explicit parse findings (WINDOWS.md 4-6)"
    addressed_in: "Phase 5"
    evidence: "Phase 5 goal 'distinguish source that could not be parsed from referenced artifacts that were not retrieved'; SC1 'Named unparsed tails and localized ERROR or MISSING nodes produce ... PARSE_UNRESOLVED findings'"
  - truth: "Unresolved reasons are scope-accurate: a procedure copybook's cursor use gets a copybook-scoped reason instead of cursor_not_declared (WR-09), a twice-declared cursor is ambiguous, not cursor_not_declared (WR-07), and a program nested under unresolved containment does not report a plain missing (IN-08)"
    addressed_in: "Phase 5"
    evidence: "Phase 5 SC2 'Each finding exposes ... resolution class, normalized reason'; SC3 'Missing-target placeholders preserve the observed target reference without fabricated ... attributes'"
---

# Phase 3: Deterministic COBOL and Mainframe Breadth Verification Report

**Phase Goal:** Users can inspect source-positioned COBOL structure, control flow, data, calls, copybooks, IDMS, CICS, and SQL without false deterministic resolution.
**Verified:** 2026-10-12T16:14:27Z
**Status:** gaps_found
**Re-verification:** Yes, after gap closure (plans 03-14 to 03-17)

## Goal Achievement

The gap-closure work is real and wired. COPY inside copybooks (previous gap 2) and procedure copybooks (previous human item 1) are closed and proven on the persisted SQLite graph. The cross-file part of previous gap 1 is closed.

One false deterministic binding remains. The same-file rule from 03-14 ignores containment, so in a multi-program file a top-level program's CALL binds to a program nested inside a different top-level program. It also shadows the real top-level target in another file. A non-mutating overlay probe confirmed this on the real indexing path (review WR-08). It is the same class of defect as previous gap 1 and fails the same truth, so it stays a blocker.

### Previous Gap Status

| Previous item | Status now | Evidence |
|---|---|---|
| Gap 1: cross-file CALL binds to a nested program | ⚠️ Partially closed | Cross-file: closed. `cobol_nested` is stamped on every program node (`cobol_grammar.go:233`, `identity.go` `nested: len(open) > 0`). The resolver excludes nested programs from other files. `TestCobolBreadth_NestedProgramScope` and 4 `nested_*` resolver subtests pass. Same-file: still false (WR-08, see gap) |
| Gap 2: COPY inside a copybook loses member name (DET-05) | ✓ Closed | `cobolCopybook` calls `cobolReferences(w, idx)`, and `w.id(root)` maps the wrapper to the file node. `TestCobolBreadth_CopybookNestedCopy`: `copy/custall.cpy -> copy/custrec.cpy` imports edge bound, with `cobol_copy_member_raw=CUSTREC`, `prov_start_row=1`, `prov_range_exact`, and no reason. `TestCobolGrammar_CopybookNestedCopy/IncludeForms` pass |
| Human item 1: procedure copybooks emit nothing | ✓ Closed (user chose fix now) | `TestCobolBreadth_CopybookProcedure`: `copy/valpara.cpy::@proc/VALIDATE-PARA` and `CHECK-PARA` persist at member rows 0 and 5. PERFORM is bound. The foreign `PERFORM MAIN-PARA` stays `label_not_in_copybook`. CALL 'SUBPGM' is bound. MCP `get_callers` returns VALIDATE-PARA. There is no program or division node and no wrapper name. `TestCobolBreadth_CopybookProcedureReferences`: reads edge to `db2::MYSCH/ORDERS` at row 4 |

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: source-positioned programs, copybooks, divisions, sections, paragraphs, statements, and data items from named parser nodes | ✓ VERIFIED | Unchanged from the previous verification. The procedure-copybook caveat is closed: copybook paragraphs and statements have member coordinates (truth 15) |
| 2 | SC2: PERFORM, THRU, and GO TO paragraph flow without unsupported resolution | ✓ VERIFIED | Unchanged for programs (`TestCobolGrammar_ControlFlow`). Copybooks resolve labels member-locally with `label_not_in_copybook` (`TestCobolGrammar_CopybookProcedureFlow`, real path) |
| 3 | SC3: literal calls, dynamic identifiers, copybook references, missing, and ambiguous stay distinguishable | ✓ VERIFIED | `TestCobolBreadth_CallsAndCopy` and `TestCobolGrammar_CopybookProcedureCalls` (dynamic CALL in a copybook goes to `unresolved::cobol::dynamic::program::<path>::<key>` with reason dynamic) pass |
| 4 | SC4: IDMS record/set and CICS program/transaction/map/mapset keep exact source evidence | ✓ VERIFIED | Unchanged for programs. Copybooks: `TestCobolGrammar_CopybookProcedureMainframe` |
| 5 | SC5: DB2 reads/writes keep statement evidence; aliases, CTEs, includes, and dynamic sources stay distinct | ✓ VERIFIED | Unchanged. Copybook SQL adds no table: repository A's KindTable set is unchanged (03-17 real-path test) |
| 6 | Goal clause: a literal CALL binds only to a program the caller can actually reach (no false deterministic resolution) | ✗ FAILED (partial) | Cross-file: fixed. Same-file: the overlay probe shows `batch.cbl::B` CALL 'UTIL' binding to `batch.cbl::A/UTIL` (reason nil) while top-level `util.cbl::UTIL` exists, and `sibling.cbl::P/Y` CALL 'X' binding to non-COMMON `P/X`. `cobol_references.go:92-106` puts every same-file candidate in `sameFile` |
| 7 | DET-05: copybook/include references keep member name and occurrence, including inside analyzed copybooks | ✓ VERIFIED | Gap 2 closed (see table above) |
| 8 | 03-02 CR-01: BOM byte offsets aligned on nodes and edges; VCS stamp on edges | ✓ VERIFIED | `TestSourceRevision*`, `TestCobolBreadth_BOMEdgeOffsets` pass |
| 9 | 03-04: owned `cobol::` namespace; same-repo exactly-one; dynamic/label/data/idms_scope/COPY IDMS never rebound | ✓ VERIFIED | `TestCobolNamespaceExclusions`, `TestCobolReferences_*`, `TestReuseIdentifierSkipsCobolNamespace` pass. COPY IDMS skip still at `cobol_references.go` (`cobolIDMSCopyTargetPrefix`) |
| 10 | Provenance contract: full prov_* on analyzed nodes and edges, restricted Meta types, no secret-like keys, extractor version recorded | ✓ VERIFIED | `TestCobolGrammar_ProvKeys` passes, and the copybook fixtures pass `assertCobolMetaContract`. Warning: the version is still `/3` after mapping changes (WR-10, deferred to Phase 4) |
| 11 | Amber and red documents still emit breadth with grade keys and prov_affected | ✓ VERIFIED | `TestCobolGrammar_DegradedBreadth` passes |
| 12 | Neo4j projection keeps Phase 3 node Meta keys | ✓ VERIFIED | `TestProjectNode_CobolBreadthMeta` passes |
| 13 | 03-11/12/13: fork copybook mode pinned; copybooks via AnalyzeCopybook under the grammar check; no wrapper nodes | ✓ VERIFIED | `go.mod`/`go.sum` show no diff since `78f40016`. `TestCobolGrammar_CopybookGrammarPinned` passes. The real-path test asserts no `CPYWRAP` ID and no program or division node in copybooks |
| 14 | 03-14: every program node carries `cobol_nested`; the resolver reads the flag and never infers nesting from ID shape | ✓ VERIFIED | Probe: `A` and `B` give `nested=false`; `A/UTIL`, `P/X`, and `P/Y` give `nested=true`. `TestCobolGrammar_ProgramNestedStamp` passes. The resolver uses only `metaCobolNested` |
| 15 | 03-16: procedure copybooks own sections, paragraphs, statements, PERFORM/THRU/GO TO, CALL, and COPY with member coordinates; no wrapper program or division | ✓ VERIFIED | `TestCobolGrammar_CopybookProcedureRoot/Flow/Calls/Fragment` and `TestCobolBreadth_CopybookProcedure` (SQLite plus MCP relay) pass |
| 16 | 03-17: copybook IDMS, CICS, and SQL edges; `not_declared_in_copybook` for undeclared data names | ✓ VERIFIED | `TestCobolGrammar_CopybookProcedureMainframe`, `TestCobolGrammar_CopybookDataNames`, and `TestCobolBreadth_CopybookProcedureReferences` pass. `cobol_grammar_data_refs.go` gates the reason on `w.root != ""` |
| 17 | 03-17: WR-04 window recorded (`cobol_copybook_wrapper`, `copybook_content_unmapped`) and documented in docs/languages.md | ✓ VERIFIED | `cobol_grammar_copybook.go` stamps both, `TestCobolGrammar_CopybookWrapperWindow` passes, and the docs row has every required sentence plus the Windows sentence |

**Score:** 16/17 truths verified (0 present, behavior-unverified)

### Judgment on New Review Findings

| Finding | Verdict | Reasoning |
|---|---|---|
| WR-08 batch file: a top-level caller binds to a program nested in another top-level program, shadowing a real top-level target | 🛑 Goal gap (blocker) | Confirmed on the real path. It is a deterministic binding to an unreachable program, the exact class the phase goal and previous gap 1 forbid. No later phase names same-file containment visibility (Phase 4 SC5 covers cross-repository binding only) |
| WR-08 non-COMMON sibling binds | 🛑 Part of the same gap | Confirmed by the probe. It is a false binding unless COMMON is proven. If the producer cannot expose `IS COMMON` (no COMMON observation found in the pinned producer module), the call should fail closed |
| WR-08 two containers each holding UTIL: false `ambiguous` | ⚠️ Warning | Conservative: nothing is bound. The containment fix above resolves it as a side effect |
| WR-09 copybook cursor stamped `cursor_not_declared` | ⚠️ Warning, deferred to Phase 5 | Confirmed by the probe (`curpara.cpy::@proc/CUR-PARA/@exec_sql_statement` and `#2`). Nothing is bound and no table is invented, so this is not a false resolution. It is an inaccurate reason, which belongs to Phase 5's normalized-reason contract. No plan truth required a copybook cursor reason. It would be cheap to fix with the gap above |
| WR-10 fail-open on a missing `cobol_nested` key | ⚠️ Warning, deferred to Phase 4 | On a fresh full index every program node carries the key (probe and real-path test). Only stale stores or foreign writers lack it, which is Phase 4 SC5 "fail closed on missing scope" |
| WR-10 extractor version `/3` not bumped; COBOL not in `extractorSaltExtLang` | ⚠️ Warning, deferred to Phase 4 | Confirmed: `cobol_grammar.go:42` is `gortex-cobol-grammar/3`, and `extractor_version.go` has no `.cbl/.cob/.cpy` entry. PROV-05 (an identifier is recorded) still holds. Stale-store invalidation on extractor change is Phase 4 SC3/ID-08. Recommendation: bump to `/4` when the WR-08 fix lands, since that changes emitted Meta again |
| IN-07 self-COPY binds a self-import | ℹ️ Info | Confirmed (`selfcpy.cpy -imports-> selfcpy.cpy`). It mirrors what is written, and compilers reject recursive COPY. Not a false resolution of a valid program |
| IN-08 `cobol_nested=true` under unresolved containment gives false `missing` | ⚠️ Warning, deferred to Phase 5 | Conservative: nothing is bound. The file node carries `prov_containment_unresolved` |
| CR-01, CR-02, WR-01, WR-06 | Deferred (unchanged) | No indexer, graph, or `resolver.go` change in `78f40016..HEAD`. The full-index path is unaffected: all real-path tests pass on a full track. CR-02's provenance flip between program and copybook writers is a shared-node ownership issue, which is Phase 4 SC1 |

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Incremental restub and COPY-edge drops, now including copybook-origin edges (CR-01, WR-01) | Phase 4 | SC3, SC4, SC5 |
| 2 | Shared resource ownership and prov flip between program and copybook writers (CR-02) | Phase 4 | SC1, SC3, SC4 |
| 3 | Cross-repository placeholder collisions | Phase 4 | SC5 |
| 4 | Extractor version bump, COBOL salt, fail-closed `cobol_nested` (WR-10) | Phase 4 | SC3 (ID-08), SC5 |
| 5 | Copybook analysis failure vs missing (WR-06); SQL producer windows (WINDOWS.md 4-6) | Phase 5 | Goal, SC1 |
| 6 | Reason accuracy: copybook cursor (WR-09), duplicate DECLARE (WR-07), unresolved containment (IN-08) | Phase 5 | SC2, SC3 |

Phase 4 must close items 1, 2, and 4 before COBOL is used under a watcher or on a store indexed before 03-14.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/parser/languages/cobol_grammar_identity.go` | `nested` from the END PROGRAM stack | ✓ VERIFIED | `nested: len(open) > 0` |
| `internal/parser/languages/cobol_grammar.go` | `cobol_nested` on program nodes; copybook branch delegates | ✓ VERIFIED | `meta["cobol_nested"] = p.nested`; `cobolCopybook(h, src, filePath, docMeta, result)` |
| `internal/parser/languages/cobol_grammar_copybook.go` | copybook pipeline, wrapper root, WR-04 stamps | ✓ VERIFIED | 86 lines. Calls `cobolProcedure`, `cobolDataSection`, `cobolDataBind`, `cobolReferences`, `cobolMainframe`, and `cobolSQL`, then stamps `grade.Green` absence |
| `internal/parser/languages/cobol_grammar_walk.go` | `root`, `id()` | ✓ VERIFIED | `func (w *cobolWalk) id(symbol string)` maps the root to the file node |
| `internal/parser/languages/cobol_grammar_data_refs.go` | data-name resolution plus `not_declared_in_copybook` | ✓ VERIFIED | 114 lines. Logic moved from `cobol_grammar_data.go` |
| `internal/resolver/cobol_references.go` | `metaCobolNested`, cross-file nested exclusion | ⚠️ VERIFIED with defect | Cross-file rule correct. Same-file rule ignores containment (gap) |
| `cmd/gortex/cobol_breadth_test.go` | nest, custall, and valpara fixtures and tests | ✓ VERIFIED | 8 `TestCobolBreadth_*` pass |
| `internal/parser/languages/cobol_grammar_copybook_test.go` | copybook unit tests | ✓ VERIFIED | 13 `TestCobolGrammar_Copybook*` pass |
| `docs/languages.md` | COBOL row updates | ✓ VERIFIED | Nested-scope, copybook COPY, procedure copybook, not-expanded, WR-04, and Windows sentences present |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| identity.go | cobol_grammar.go | `p.nested` into Meta | ✓ WIRED | |
| cobol_grammar.go | resolver/cobol_references.go | `cobol_nested` read as `metaCobolNested` | ✓ WIRED | Probe shows the flag on persisted nodes |
| cobol_grammar.go | cobol_grammar_copybook.go | `cobolCopybook(` | ✓ WIRED | |
| cobol_grammar_copybook.go | cobol_grammar_refs.go | `cobolReferences(w, idx)` | ✓ WIRED | Bound copybook-to-copybook edge on SQLite |
| cobol_grammar_copybook.go | cobol_grammar_procedure.go | `cobolProcedure(w)` | ✓ WIRED | |
| cobol_grammar_copybook.go | cobol_grammar_sql.go / mainframe.go | `cobolSQL(w, idx)`, `cobolMainframe(w, idx)` | ✓ WIRED | Persisted reads edge to `db2::MYSCH/ORDERS` |
| cobol_grammar_data_refs.go | cobol_grammar_walk.go | `w.root` selects `not_declared_in_copybook` | ✓ WIRED | |
| cobol_grammar_copybook.go | producer `grade` | `h.Assessment.Grade != grade.Green` | ✓ WIRED | |
| cobol_grammar_flow.go | resolver | `unresolved::cobol::label::` never bound | ✓ WIRED | Copybook label placeholder persists with its reason |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| Persisted COBOL graph (SQLite) | copybook nodes and edges | handoff → `cobolCopybook` → MultiIndexer → store_sqlite | Yes. Read back via `GetNode`, `GetOutEdges`, and MCP `get_callers` | ✓ FLOWING |
| Resolver stamps | `cobol_nested` → candidate filter | extractor Meta → `resolveCobolReferences` | Yes. The cross-file result is `missing` on SQLite | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Extractor, resolver, indexer, and projection COBOL suites | `GOWORK=off go test -count=1 -run 'Cobol\|TestReuseIdentifierSkipsCobolNamespace\|TestSourceRevision' ./internal/parser/languages/ ./internal/resolver/ ./internal/indexer/ ./internal/neo4jprojection/` | 4 × ok | ✓ PASS |
| Real-path two-repo acceptance | `GOWORK=off go test -count=1 -v -run 'TestCobolBreadth\|TestCobolTracer' ./cmd/gortex/` | 16/16 PASS | ✓ PASS |
| Build and vet | `GOWORK=off go build ./...`; `go vet` on the languages and resolver packages | exit 0, no findings | ✓ PASS |
| Windows dependency closure | `GOOS=windows go list -deps ./cmd/gortex \| grep -c preprocessor` | 0 | ✓ PASS |
| go.mod/go.sum unchanged during gap closure | `git diff --stat 78f40016..HEAD -- go.mod go.sum` | empty | ✓ PASS |
| Batch file: top-level B CALL 'UTIL' (overlay probe, no tree change) | `go test -overlay … -run TestVerifierProbe_Phase3 ./cmd/gortex/` | `B/@proc/B-PARA -calls-> batch.cbl::A/UTIL reason=<nil>` while `util.cbl::UTIL` exists | ✗ FAIL (false binding) |
| Non-COMMON sibling CALL (same probe) | same | `P/Y/@proc/Y-PARA -calls-> sibling.cbl::P/X reason=<nil>` | ✗ FAIL (false binding) |
| Copybook cursor reason (same probe) | same | `curpara.cpy::@proc/CUR-PARA/@exec_sql_statement reason=cursor_not_declared` | ⚠️ WR-09 confirmed (deferred) |
| Self-COPY (same probe) | same | `selfcpy.cpy -imports-> selfcpy.cpy` | ℹ️ IN-07 confirmed |
| Full suite | `go test -race -timeout 40m ./...` | orchestrator: 156 ok through wave 13; 03-17 executor reports a pass on final HEAD | relied on, not re-run |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared or present for this phase. Step 7c: SKIPPED. The verifier's own overlay probe is listed under spot-checks. It was overlaid as `cmd/gortex/zz_verifier_probe_test.go` from the scratchpad, and the working tree was not modified.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| DET-01 | 03-01,02,03,05,10,11,12,13,16,17 | source-positioned program/division/section/paragraph/statement/data-item observations | ✓ SATISFIED | Truths 1, 13, 15, 17 |
| DET-02 | 03-01,06,10,16 | PERFORM/THRU/GO TO control flow | ✓ SATISFIED | Truth 2 |
| DET-03 | 03-04,07,10,14,16 | literal calls separate from dynamic identifiers | ✓ SATISFIED | Separation holds. The same-file false binding is tracked under the goal clause (truth 6), not DET-03's separation wording |
| DET-04 | 03-04,07,10,14,16,17 | dynamic identifiers stay unresolved | ✓ SATISFIED | Copybook dynamic CALL, CICS, and PREPARE stay dynamic |
| DET-05 | 03-07,10,15,16 | copybook/include references keep member name and occurrence | ✓ SATISFIED | Truth 7; previous gap 2 closed |
| DET-06 | 03-04,07,10,15 | copybook ambiguity preserved | ✓ SATISFIED | The unchanged exactly-one rule applies to copybook-origin COPYs |
| DET-07 | 03-08,10,17 | IDMS record references | ✓ SATISFIED | Truths 4, 16 |
| DET-08 | 03-08,10,17 | IDMS set references | ✓ SATISFIED | Truths 4, 16 |
| DET-09 | 03-08,10,17 | CICS program/transaction references | ✓ SATISFIED | Truths 4, 16 |
| DET-10 | 03-08,10,17 | CICS map/mapset references | ✓ SATISFIED | Truths 4, 16 |
| DET-11 | 03-09,10,17 | SQL read/write to deterministically identified tables | ✓ SATISFIED | Truths 5, 16 |
| DET-12 | 03-09,10,17 | aliases/CTEs/includes/dynamic sources distinguishable | ✓ SATISFIED | Truth 5 |

No orphaned requirements. DET-01 to DET-12 are all claimed, and REQUIREMENTS.md maps no other ID to Phase 3. REQUIREMENTS.md marks all twelve `[x] Complete`. Every requirement is satisfied as worded, but the phase as a whole is not complete until the goal-clause gap closes.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (all files changed in 78f40016..HEAD) | – | TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER | none found | Debt-marker gate clean |
| internal/resolver/cobol_references.go | 92-106 | same-file candidates ignore containment | 🛑 Blocker | False CALL binding in multi-program files (gap) |
| internal/parser/languages/cobol_grammar_sql.go | 151-156 | copybook cursor use and duplicate DECLARE both stamped `cursor_not_declared` | ⚠️ Warning | Misleading reason (WR-07, WR-09). Deferred to Phase 5 |
| internal/parser/languages/cobol_grammar.go | 42 | extractor version unchanged after mapping changes | ⚠️ Warning | Stale stores undetectable (WR-10). Deferred to Phase 4 |
| internal/resolver/cobol_references.go | 98 | missing `cobol_nested` treated as top-level | ⚠️ Warning | Fail-open on stale stores (WR-10). Deferred to Phase 4 |
| internal/parser/languages/cobol_grammar_data_refs.go | 14-41, 62-82 | REDEFINES/RENAMES looked up owner-wide | ⚠️ Warning | False `ambiguous` (WR-03). Conservative, not a binding |
| internal/parser/languages/cobol_grammar_procedure.go / statements.go | 96-98 / 110 | copybook statements `member_of` the file; paragraphs do not | ℹ️ Info | Two containment shapes (IN-06) |
| internal/resolver/cobol_references.go | 108-115 | self-COPY binds | ℹ️ Info | IN-07 |
| procedure copybook CALL binding | – | a copybook's literal CALL binds to the repository's sole top-level program, though an including program may contain a same-named nested program | ℹ️ Info | The copybook cannot see its includers, and COPY is not expanded (documented). Rare. Note for the WR-08 fix design |

### Human Verification Required

Status is `gaps_found`, so these items do not set the status. Resolve them alongside gap closure.

#### Judgment-tier prohibitions (non-authoritative LLM-judge verdicts, flagged: unverified-prohibition, human review recommended)

The 20 prohibitions from 03-01 to 03-13 keep their previous verdicts (all hold; see the prior report). New prohibitions from 03-14 to 03-17:

- 03-14: no cross-file binding to a nested program, and no inferring nesting from ID shape. Judge: holds. The resolver reads only `metaCobolNested`. The same-file false binding is the separate gap above.
- 03-15: no copybook-reference binding other than exactly-one, and no COPY IDMS binding. Judge: holds (copybook path unchanged; `cobolIDMSCopyTargetPrefix` skip).
- 03-15 and 03-16: no wrapper program, division, or wrapper-byte range emitted. Judge: holds. The real-path test asserts no program or division node and no `CPYWRAP` ID, and the copybook root maps to the file node.
- 03-16: no PERFORM or GO TO bound outside the member, and no fall-through. Judge: holds (`label_not_in_copybook` real-path assertion).
- 03-16 and 03-17: no fork change, no go.mod/go.sum change. Judge: holds (empty diff since `78f40016`).
- 03-17: no KindTable for an alias, CTE, include, derived table, or dynamic source from a copybook, and no binding of a copybook data name to an includer's item. Judge: holds (table set unchanged; `not_declared_in_copybook`).
- 03-17 transparency: no degraded empty copybook presented as complete. Judge: holds as scoped. A degraded member that mapped something carries no absence stamp, and only its amber or red grade signals the loss. The plan accepted this explicitly.
- All gap plans: no estate names in fixtures. Judge: holds as far as visible (invented names: CUSTALL, VALPARA, NESTCALL, OUTER/INNER).

### Gaps Summary

Previous gap 2 and human item 1 are closed with real-path evidence. Previous gap 1 is closed for callers in other files but not for the same file. `resolveCobolReferences` puts every same-file program in the preferred candidate set, whatever its containment. As a result:

- In a multi-program (batch) source file, a top-level program's literal CALL binds to a program nested inside a different top-level program.
- That same-file set shadows the real top-level target in another file.
- A non-COMMON sibling also binds.

This is the false deterministic resolution the phase goal forbids, confirmed through the real MultiIndexer, SQLite, and resolver path.

The fix is narrow. Program nodes need their container (or the extractor binds visible nested callees itself), and same-file top-level programs join the ordinary candidate set. When this lands, bump the extractor version to `/4` (WR-10). Optionally also give copybook cursor uses a copybook-scoped reason (WR-09).

**Possible override (not recommended).** D-09 says literally "Nested and contained programs in the same file resolve within the file first", and 03-14 implemented that wording. If you decide multi-program batch files are out of scope for Phase 3, record an override and add the item to the Phase 4 roadmap:

```yaml
overrides:
  - must_have: "Phase goal clause 'without false deterministic resolution' (carried from previous gap 1): a literal CALL is bound only to a program a separately compiled caller can actually reach"
    reason: "Same-file containment visibility (multi-program batch files, non-COMMON siblings) deferred; Phase 4 roadmap updated with a success criterion for it"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

---

_Verified: 2026-10-12T16:14:27Z_
_Verifier: Claude (gsd-verifier)_
