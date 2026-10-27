---
phase: 03-deterministic-cobol-and-mainframe-breadth
reviewed: 2026-10-12T16:08:03Z
depth: standard
review_kind: incremental (diff_base 78f40016, gap-closure plans 03-14..03-17)
files_reviewed: 18
files_reviewed_list:
  - cmd/gortex/cobol_breadth_test.go
  - docs/languages.md
  - internal/parser/languages/cobol_grammar_copybook_test.go
  - internal/parser/languages/cobol_grammar_copybook.go
  - internal/parser/languages/cobol_grammar_data_refs.go
  - internal/parser/languages/cobol_grammar_data.go
  - internal/parser/languages/cobol_grammar_flow.go
  - internal/parser/languages/cobol_grammar_identity.go
  - internal/parser/languages/cobol_grammar_mainframe.go
  - internal/parser/languages/cobol_grammar_procedure.go
  - internal/parser/languages/cobol_grammar_refs.go
  - internal/parser/languages/cobol_grammar_sql.go
  - internal/parser/languages/cobol_grammar_statements.go
  - internal/parser/languages/cobol_grammar_test.go
  - internal/parser/languages/cobol_grammar_walk.go
  - internal/parser/languages/cobol_grammar.go
  - internal/resolver/cobol_references_test.go
  - internal/resolver/cobol_references.go
findings:
  critical: 2
  warning: 7
  info: 8
  total: 17
status: issues_found
---

# Phase 03: Code Review Report (incremental re-review)

**Reviewed:** 2026-10-12T16:08:03Z
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

This is an incremental re-review of the gap-closure plans 03-14 through 03-17, against `78f40016..HEAD`:

- 03-14 stamps `cobol_nested` on program nodes and makes the resolver consider nested programs only as same-file candidates.
- 03-15 emits COPY edges inside copybooks. It adds `cobolWalk.root` and `w.id`.
- 03-16 maps procedure copybooks: their structure, flow, and CALLs, and adds the `label_not_in_copybook` reason.
- 03-17 maps IDMS, CICS, and SQL edges in copybooks, adds the `not_declared_in_copybook` reason, splits the data-reference code into its own file, and records the WR-04 window.

**Program documents: no regression found.** For a program document, `w.root` is empty, so `w.id(sym)` is still `<filePath>::<sym>`. Every converted call site produces the same IDs and placeholders as before:
- `cobolPerform` and `cobolGoTo` (sources and `unresolved::cobol::label::<file>::<prog>::…`);
- `cobolCallTarget` and `cobolCICS` (program and owner IDs);
- `cobolSQL` (`programID`);
- `cobolCopy` (`from`);
- `cobolStatements` (`member_of`).

The new `parentID != w.filePath` guard in `cobolProcedure.finish` can never fire for a program document, because `divID` there is always a division node ID. The two copybook-only reason rewrites (`label_not_in_copybook`, `not_declared_in_copybook`) are gated on `w.root != ""`. The test move from `cobol_grammar_test.go` to `cobol_grammar_copybook_test.go` keeps every removed test function. These suites pass locally:
- `go test ./internal/parser/languages -run Cobol`
- `go test ./internal/resolver -run Cobol`
- `go test ./cmd/gortex -run TestCobolBreadth`

**Status of prior findings:**
- **Resolved:** WR-02 for the cross-file case it described, and WR-05.
- **Recorded window:** WR-04. The wrapper and absence stamps are in place and the limit is documented.
- **Still open:** CR-01, CR-02, WR-01, WR-03, WR-06, WR-07, and all Info items. Neither the indexer, the graph, nor `resolver.go` changed in this range.
- **Wider scope:** CR-01, CR-02, and WR-07. Copybooks now emit shared DB2, IDMS, and CICS resource nodes, copybook-to-copybook imports edges, and cursor uses.

**New findings:**
- **WR-08:** the same-file nested-program preference ignores COBOL containment and the COMMON attribute, so it can still bind a CALL falsely within a file.
- **WR-09:** a procedure copybook reports `cursor_not_declared` for cursors its including program declares.
- **WR-10:** a missing `cobol_nested` key fails open, and the unchanged extractor version hides stale stores.
- **IN-06 to IN-08:** minor consistency and edge-case gaps.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Incremental re-index turns bound COBOL CALL edges into bare-name stubs and drops COPY edges

**Status:** open (deferred to Phase 4/5). Scope is wider: 03-15 and 03-16 add more inbound edges that this drops.
**File:** `internal/indexer/indexer.go:5218-5266` (`restubIncomingRefs`), `internal/indexer/incremental_batch.go:957` (`restubIncomingRefsFromView`), `internal/resolver/cobol_references.go:61`, `internal/graph/graph.go:3275-3297`
**Issue:** There are no changes to the indexer or graph since the prior review.

`restubIncomingRefs` still re-points a bound COBOL CALL, LINK, or XCTL edge to `unresolved::<PROGRAM-ID>`. That is outside the `cobol::` namespace, so the generic bare-name resolver can rebind it. This is the false deterministic resolution that D-09 and D-11 forbid.

`EvictFile` still deletes every inbound COPY `imports` edge to a re-indexed or deleted copybook. Since 03-15, those inbound edges include copybook-to-copybook imports (`copy/custall.cpy -> copy/custrec.cpy`). Since 03-16, they include literal CALLs from procedure copybooks to programs. Both are lost or restubbed the same way.

**Fix:** This is unchanged from the prior review. Restub COBOL program and copybook targets into `unresolved::cobol::<kind>::<cobol_name_key>`, and include `KindFile` copybook nodes and `EdgeImports` in the restub set. Then run a scoped `resolveCobolReferences` from the incremental tail (see WR-01).
```go
stub := graph.UnresolvedMarker + n.Name
if n.Language == "cobol" {
    kind, _ := n.Meta["cobol_kind"].(string)
    key, _ := n.Meta["cobol_name_key"].(string)
    if (kind == "program" || kind == "copybook") && key != "" {
        stub = graph.UnresolvedMarker + "cobol::" + kind + "::" + key
    }
}
```

### CR-02: Shared DB2, IDMS, and CICS resource nodes are owned by whichever file wrote last; evicting that file deletes other programs' edges

**Status:** open (deferred to Phase 4/5). Scope is wider: copybooks are now writers too.
**File:** `internal/parser/languages/cobol_grammar_walk.go:230-243` (`resourceNode`), `internal/parser/languages/cobol_grammar_copybook.go:66-68`, `internal/parser/languages/cobol_grammar_sql.go:248`, `internal/parser/languages/cobol_grammar_mainframe.go:83,186`
**Issue:** `resourceNode` is unchanged. It emits the shared `db2::…`, `idms::…`, and `cics::…` nodes with `FilePath: w.filePath` and a clone of the writing document's `docMeta`.

Since 03-17, `cobolCopybook` also runs `cobolMainframe` and `cobolSQL`, so procedure copybooks emit these same shared nodes. `TestCobolBreadth_CopybookProcedureReferences` shows `copy/valpara.cpy` and `cobol/ordpgm.cbl` both writing `<prefixA>/db2::MYSCH/ORDERS`. As a result:
- Ownership, `prov_source_path`, `prov_document_grade`, and `prov_grade_policy` now flip between a program document and a copybook. `cobol-grade-policy-copybook-v1` can appear on a table node first seen by a program.
- Re-indexing or deleting one copybook can delete the table's inbound `reads` and `writes` edges from every program in the repository.

**Fix:** This is unchanged from the prior review. Keep shared resources out of per-file eviction, either through a synthetic owner path plus a post-resolve GC, or through per-file anchor nodes. Strip the per-document `prov_*` keys from shared nodes.

## Warnings

### WR-01: `resolveCobolReferences` never runs on the incremental path, and bound edges are never re-checked for exactly-one

**Status:** open (deferred to Phase 4/5)
**File:** `internal/resolver/resolver.go:1662`, `internal/resolver/resolver.go:2652` (`ResolveFilesAndIncoming`), `internal/resolver/cobol_references.go:70`
**Issue:** This is unchanged. The pass is still reached only from `ResolveAllContext` and still iterates only `EdgesWithUnresolvedTarget()`. Two consequences:
- A watcher save leaves the new copybook COPY and CALL edges unresolved until the next full resolve.
- An edge bound under exactly-one is never re-stamped `ambiguous` when a second candidate appears.

The 03-14 nested filter adds one more stale case. If a program is edited so that it becomes contained in another program (`cobol_nested` flips to true), cross-file edges already bound to it stay bound, although the new rule says they must not be.
**Fix:** This is unchanged from the prior review. Call a scoped pass from the incremental tail, and re-evaluate already-bound COBOL edges. For example, keep the placeholder key in edge Meta so a bound edge can be reverted when its candidate count is no longer 1.

### WR-02: Cross-file CALL can bind to a nested (contained) program in another file

**Status:** resolved (03-14)
**File:** `internal/resolver/cobol_references.go:92-106`, `internal/parser/languages/cobol_grammar_identity.go:116`, `internal/parser/languages/cobol_grammar.go:233`
**Resolution:** `cobolProgramSymbols` sets `nested: len(open) > 0`, and `Extract` stamps it as `cobol_nested`. The resolver now uses only top-level (`cobol_nested` not true) candidates for callers in other files, and same-file candidates otherwise. Nesting is read from the flag, never from the shape of a QualName. `TestCobolBreadth_NestedProgramScope` covers this on the persisted SQLite graph.

Two residual issues remain, filed as new findings:
- Same-file candidates are still not scoped by containment (WR-08).
- A missing flag fails open (WR-10).

### WR-03: REDEFINES and RENAMES are resolved program-wide, giving false "ambiguous" results in ordinary code

**Status:** open (deferred to Phase 4/5)
**File:** `internal/parser/languages/cobol_grammar_data_refs.go:14-41,62-82` (moved out of `cobol_grammar_data.go` in 03-17, logic unchanged)
**Issue:** `cobolDataBind` still resolves REDEFINES and RENAMES operands through `idx.lookup(owner, …)` over every item that has the same owner. That owner is now the whole program, or the whole copybook. A routine repeated child name, such as `05 WS-DATE` under two 01 records followed by `REDEFINES WS-DATE`, still gives `ambiguous`. COBOL makes the target unique: for REDEFINES it is the preceding same-level entry, and for RENAMES it lies within the preceding 01 record.
**Fix:** This is unchanged. Bind REDEFINES to the previous same-level sibling, and restrict the RENAMES lookup to the enclosing 01 record.

### WR-04: The copybook header heuristic misclassifies common members (fork)

**Status:** recorded window. This is not counted as open. The producer fix is deferred, and Gortex records the producer's choice.
**File:** `internal/parser/languages/cobol_grammar_copybook.go:73-85`, `docs/languages.md:279`, fork `preprocessor/transform/copybook.go:77-94`
**Resolution:** The file node now records:
- `cobol_copybook_wrapper` (`data` or `procedure`), read from the wrapper program's first data or procedure division;
- `prov_analysis_absence=copybook_content_unmapped` when the grade is not green and nothing beyond the file node was mapped.

`docs/languages.md` documents the window, listing `EXEC CICS`, non-declarative `EXEC SQL`, `FD`/`SD`, a leading `COPY`, and `-INC`. `TestCobolGrammar_CopybookWrapperWindow` covers it.

**Residual gap (accepted with the window):** the absence stamp fires only when the copybook mapped nothing at all. A member that parses with the wrong wrapper but still maps some fragments carries no stamp. Only the amber or red `prov_document_grade` signals that statements were lost. The producer fix (classify `EXEC` by sub-verb, treat `FD`/`SD` as data, skip a leading `COPY`/`-INC`) remains the real remedy.

### WR-05: Copybook mode emits no reference edges, so nested COPY and procedure members vanish

**Status:** resolved (03-15, 03-16, 03-17)
**File:** `internal/parser/languages/cobol_grammar_copybook.go:45-86`, `internal/parser/languages/cobol_grammar_procedure.go:47-64`, `internal/parser/languages/cobol_grammar_refs.go:155-161`
**Resolution:** `cobolCopybook` now runs the full pipeline with the producer's wrapper program as the walk root, and `w.id(root)` maps that root to the copybook file node:
- `cobolProcedure`: sections, paragraphs, statements, PERFORM, and GO TO;
- `cobolReferences`: COPY, `-INC`, COPY IDMS, SQL INCLUDE, and CALL;
- `cobolMainframe` and `cobolSQL`.

The wrapper's program and divisions are never emitted, and the tests assert that. `docs/languages.md` documents the behaviour and states that COPY is not expanded into including programs. Copybook-to-copybook imports bind on the real SQLite path (`TestCobolBreadth_CopybookNestedCopy`).

### WR-06: A copybook analysis error drops the copybook's file node, so every COPY of it reports `missing`

**Status:** open (deferred to Phase 4/5)
**File:** `internal/parser/languages/cobol_grammar.go:185-186`
**Issue:** This is unchanged. `case err != nil && copybook: return nil, …` still drops the file node. Every program's COPY of that copybook then becomes `cobol_unresolved_reason=missing`, although the member exists.

The cost is now higher. A copybook that fails analysis also loses its own outbound COPY, CALL, and SQL edges from 03-15 through 03-17, and the copybooks it includes lose their inbound import from it.
**Fix:** This is unchanged. On a copybook analysis error, return the identity-only file node with `cobol_kind=copybook`, `cobol_name_key`, `cobol_name_norm`, and `prov_analysis_absence=copybook_analysis_failed`.

### WR-07: A cursor declared twice is labeled `cursor_not_declared`

**Status:** open (deferred to Phase 4/5). Scope is wider: see WR-09.
**File:** `internal/parser/languages/cobol_grammar_sql.go:151-156`
**Issue:** This is unchanged. `if len(decls) != 1` stamps `cursor_not_declared` both when there are zero DECLAREs and when there are two or more. Ambiguity is therefore reported as absence, and the candidate DECLAREs are not recorded.
**Fix:**
```go
switch len(decls) {
case 0:
    n.Meta["cobol_unresolved_reason"] = "cursor_not_declared"
case 1:
    // emit reads
default:
    n.Meta["cobol_unresolved_reason"] = "ambiguous"
    n.Meta["cobol_candidate_symbols"] = /* sorted DECLARE statement QualNames */
}
```

### WR-08: Same-file program candidates ignore COBOL containment, so a CALL can still bind to an invisible nested program, or turn ambiguous falsely, within one file

**Status:** new (residual of WR-02)
**File:** `internal/resolver/cobol_references.go:92-106`
**Issue:** For a program target, every candidate whose `FilePath` equals the edge's `FilePath` is put in `sameFile`, whatever its containment. `sameFile` then replaces all other candidates. COBOL visibility does not work that way. A contained program is callable only by its directly containing program, and by its siblings when it is `COMMON`. Concrete false results:
- **Batch file with two top-level programs.** In `A (contains UTIL) … END PROGRAM A. B. … CALL 'UTIL'`, B's `CALL 'UTIL'` binds deterministically to `A/UTIL`. A has no visibility relationship with B. If a top-level `UTIL` exists in another file, that real target is never considered, because the non-empty `sameFile` shadows it.
- **Sibling that is not COMMON.** If A contains X and Y, and Y calls X where X is not COMMON, the call binds to `A/X`. COBOL scoping forbids that call.
- **Two containers with same-named children.** If A contains `UTIL`, B contains `UTIL`, and A calls `UTIL`, the result is `ambiguous` with both candidates. COBOL makes `A/UTIL` the unique target.

The docs line "A nested (contained) program is a CALL target only from its own file" describes the implementation, but the binding is still claimed under exactly-one, which is the false deterministic resolution that D-09 forbids.
**Fix:** Decide visibility from containment instead of file equality.
1. Stamp `cobol_container` (the containing program's symbol) and `cobol_common` (from the PROGRAM-ID `IS COMMON` clause) on program nodes in `Extract`.
2. In the resolver, derive the caller's program symbol from the edge's `From` QualName prefix. Better, stamp `cobol_caller_program` on CALL edges in `cobolCallTarget`.
3. Keep a same-file candidate only when its container is the caller, or when it is COMMON and shares the caller's container. Otherwise it is not a candidate, and the top-level cross-file set is used.

```go
for _, c := range candidates {
    nested, _ := c.Meta["cobol_nested"].(bool)
    switch {
    case !nested && c.FilePath != e.FilePath:
        topLevel = append(topLevel, c)
    case nested && c.FilePath == e.FilePath && cobolVisible(c, callerProg):
        sameFile = append(sameFile, c)
    case !nested && c.FilePath == e.FilePath:
        topLevel = append(topLevel, c) // a separately compiled program in a batch file
    }
}
```

Alternatively, bind visible nested callees in the extractor, where containment is known exactly, and keep nested programs out of the resolver's index entirely. Add tests for the batch-file and two-container cases.

### WR-09: A procedure copybook reports `cursor_not_declared` for a cursor its including program declares

**Status:** new (03-17)
**File:** `internal/parser/languages/cobol_grammar_sql.go:92,114,151-156`, `internal/parser/languages/cobol_grammar_copybook.go:68`
**Issue:** In a copybook, `programID` is `w.id("@proc")`, which is the copybook path. `declared` therefore holds only DECLAREs written in the member itself. A very common pattern is a program that DECLAREs a cursor and a procedure copybook that does `OPEN`/`FETCH`/`CLOSE` on it. In that case the copybook's statement node gets `cobol_unresolved_reason=cursor_not_declared`, and no `reads` edge is emitted.

That reason is false: the cursor is declared, in the including program, which the copybook cannot see. Plans 03-16 and 03-17 introduced copybook-specific reasons for exactly this situation (`label_not_in_copybook` in `cobolFlowEdge`, `not_declared_in_copybook` in `cobolDataReference`). The cursor path was missed, so the D-12 vocabulary is inconsistent, and consumers filtering on `cursor_not_declared` get false positives from copybooks.
**Fix:** Mirror the label and data rewrite:
```go
if len(decls) == 0 {
    reason := "cursor_not_declared"
    if w.root != "" {
        reason = "cursor_not_in_copybook" // an including program may declare it
    }
    n.Meta["cobol_unresolved_reason"] = reason
}
```
Add a `TestCobolGrammar_CopybookProcedureMainframe` case with `EXEC SQL OPEN C1` and no DECLARE, and update `docs/languages.md` alongside `label_not_in_copybook` and `not_declared_in_copybook`.

### WR-10: A missing `cobol_nested` key fails open, and the unchanged extractor version hides stores that lack it

**Status:** new (03-14 through 03-17)
**File:** `internal/resolver/cobol_references.go:98`, `internal/parser/languages/cobol_grammar.go:40-42`, `internal/indexer/extractor_version.go:37-95`
**Issue:**
- **Fail-open.** `nested, _ := c.Meta[metaCobolNested].(bool)` treats an absent key as "top-level". Any program node written before 03-14, or by any other writer, is therefore still a cross-file candidate, and the WR-02 false binding comes back silently.
- **Version contract broken.** The constant's contract says "Bump it whenever the emitted nodes, edges, or prov_* keys change." Plans 03-14 through 03-17 added `cobol_nested`, `cobol_copybook_wrapper`, `prov_analysis_absence=copybook_content_unmapped`, new copybook node families (`@proc/…`), and new edge families. Yet `prov_extractor_version` stays `gortex-cobol-grammar/3`, and `TestCobolBreadth_CopybookProcedure` pins that value. Two different mappings now carry the same provenance identity, and nothing can tell a stale store from a current one. 03-14-SUMMARY concedes that "existing stores need a reindex before `cobol_nested` appears", but nothing triggers or detects that reindex.
- **No re-extraction.** `.cbl`, `.cob`, and `.cpy` are absent from `extractorSaltExtLang`. The extractor-version Merkle salt therefore cannot re-flag unchanged COBOL files when this mapping changes.
**Fix:**
- Bump `cobolGrammarExtractorVersion` to `gortex-cobol-grammar/4` and update the pinned test assertions.
- Map `.cbl`, `.cob`, and `.cpy` to a `"cobol"` key in `extractorSaltExtLang`, with a `"cobol": 2` entry in `extractorVersions`, so unchanged members re-extract.
- Fail closed in the resolver:
```go
nested, known := c.Meta[metaCobolNested].(bool)
if !known || nested { continue } // unknown nesting is never a cross-file candidate
```

## Info

### IN-01: DB2 table IDs use a second identity scheme

**Status:** open (deferred to Phase 4/5)
**File:** `internal/parser/languages/cobol_grammar_sql.go:248`
**Issue:** This is unchanged. COBOL emits `db2::<SCHEMA>/<TABLE>`, while migration DDL uses `db::db2::SCHEMA.TABLE` (`gortexsql.TableNodeID`). Procedure copybooks now emit the COBOL scheme too.
**Fix:** Record the divergence for ID-04, or add a `same_as` bridge in Phase 4.

### IN-02: Copybook detection relies only on the `.cpy` extension, and extension handling is asymmetric

**Status:** open (deferred to Phase 4/5)
**File:** `internal/parser/languages/cobol_grammar.go:173`, `internal/parser/languages/cobol_grammar_copybook.go:72`, `internal/parser/languages/cobol_grammar_refs.go:135`
**Issue:** This is unchanged. The copybook file key strips any extension (`filepath.Ext`). A COPY key strips only `.CPY`. Only `.cpy` members get copybook mode.
**Fix:** Strip the same extension set on both sides, and document that only `.cpy` members are copybook candidates.

### IN-03: Program symbols are embedded in IDs and placeholders without `cobolIDSegment`

**Status:** open (deferred to Phase 4/5)
**File:** `internal/parser/languages/cobol_grammar_identity.go:106-116`
**Issue:** This is unchanged. `base := name` / `open[...].symbol + "/" + name` still uses the verbatim `program_name`.
**Fix:** Encode each program-name component with `cobolIDSegment`. This is an ID change, so bump the extractor version (see WR-10).

### IN-04: Dedupe discards the second occurrence's semantics

**Status:** open (deferred to Phase 4/5)
**File:** `internal/parser/languages/cobol_grammar_dedupe.go:26-46`
**Issue:** This is unchanged. The copybook branch now feeds procedure flow edges through `cobolDedupeEdges` as well (`cobol_grammar.go:222`), so the same collapse of a PERFORM and a GO TO on one line applies to copybooks.
**Fix:** This is unchanged: union the `cobol_flow` values, and pad `cobol_statement_symbols` so it stays index-aligned.

### IN-05: The projection test fixture uses a different type than the extractor

**Status:** open (deferred to Phase 4/5)
**File:** `internal/neo4jprojection/properties_test.go:341`, `internal/parser/languages/cobol_grammar_flow.go:288-289`
**Issue:** This is unchanged. The fixture sets `cobol_renames` to a string, but the extractor writes `[]string`. `cobol_depending_positions` stores integers as `[]string`.
**Fix:** This is unchanged.

### IN-06: Copybook statements get `member_of` the file node, but copybook paragraphs and sections do not

**Status:** new (03-16)
**File:** `internal/parser/languages/cobol_grammar_procedure.go:96-98`, `internal/parser/languages/cobol_grammar_statements.go:110`
**Issue:** In a copybook, `finish` deliberately skips `member_of` when the parent is the file node, leaving top-level paragraphs and sections with only `defines`. `cobolStatements` does `w.memberOf(node.ID, w.id(w.container[i]), …)`, and for a statement before any paragraph that resolves to the file node. So statements get a `member_of` to a `KindFile` node, while the paragraphs at the same level do not. Containment queries over copybooks therefore see two shapes. The `cobolStatements` doc comment ("which is the file node in a copybook") records the behaviour but not why it differs.
**Fix:** Pick one rule. Either skip `member_of` to `w.filePath` in `cobolStatements` too, or emit it for top-level paragraphs and sections as well. Then assert it in `TestCobolGrammar_CopybookProcedureRoot`.

### IN-07: A copybook that COPYs itself binds a self-import

**Status:** new (03-15)
**File:** `internal/resolver/cobol_references.go:92,108-115`, `internal/parser/languages/cobol_grammar_refs.go:155-163`
**Issue:** Copybook COPY edges now come from the copybook file node. The copybook index includes that same node. So `COPY X` inside `X.cpy` (recursive COPY, which compilers reject) binds `X.cpy -> X.cpy` as an exactly-one deterministic import. Cycle analyses then report a self-loop as a real dependency.
**Fix:** In `resolveCobolReferences`, when `candidates[0].ID == e.From`, stamp a reason such as `cobol_unresolved_reason=self_copy` instead of binding.

### IN-08: `cobol_nested=true` is asserted even when the file's containment is unresolved

**Status:** new (03-14)
**File:** `internal/parser/languages/cobol_grammar_identity.go:100-116`, `internal/parser/languages/cobol_grammar.go:250-252`
**Issue:** When a program with children is popped without a matching END PROGRAM, `containmentUnresolved` is set. The file node gets `prov_containment_unresolved`, but the children keep `nested: true` from the guessed containment. The resolver trusts that flag and drops them from cross-file candidates. The program may in fact be a separately compiled program in a malformed batch file, so a cross-file CALL to it becomes a false `missing`, with no link to the file-level flag.
**Fix:** When `containmentUnresolved` is true, stamp `cobol_nested_unresolved=true` on the affected programs, or leave `cobol_nested` unset. Fail-closed handling (WR-10) then reports those calls with a distinct reason instead of `missing`.

---

_Reviewed: 2026-10-12T16:08:03Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
