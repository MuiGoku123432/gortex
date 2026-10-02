---
phase: 03-deterministic-cobol-and-mainframe-breadth
reviewed: 2026-10-02T13:49:07Z
depth: standard
files_reviewed: 46
files_reviewed_list:
  - cmd/gortex/cobol_breadth_test.go
  - cmd/gortex/cobol_tracer_test.go
  - docs/languages.md
  - go.mod
  - go.sum
  - internal/indexer/cobol_bom_test.go
  - internal/indexer/coverage_strip_test.go
  - internal/indexer/incremental_resolve_test.go
  - internal/indexer/incremental_resolve.go
  - internal/indexer/indexer.go
  - internal/indexer/source_revision_test.go
  - internal/indexer/source_revision.go
  - internal/neo4jprojection/properties_test.go
  - internal/parser/languages/cobol_grammar_contract_test.go
  - internal/parser/languages/cobol_grammar_data_clauses.go
  - internal/parser/languages/cobol_grammar_data_test.go
  - internal/parser/languages/cobol_grammar_data.go
  - internal/parser/languages/cobol_grammar_dedupe.go
  - internal/parser/languages/cobol_grammar_flow_test.go
  - internal/parser/languages/cobol_grammar_flow.go
  - internal/parser/languages/cobol_grammar_identity.go
  - internal/parser/languages/cobol_grammar_mainframe_test.go
  - internal/parser/languages/cobol_grammar_mainframe.go
  - internal/parser/languages/cobol_grammar_procedure_test.go
  - internal/parser/languages/cobol_grammar_procedure.go
  - internal/parser/languages/cobol_grammar_refs_test.go
  - internal/parser/languages/cobol_grammar_refs.go
  - internal/parser/languages/cobol_grammar_sql_test.go
  - internal/parser/languages/cobol_grammar_sql.go
  - internal/parser/languages/cobol_grammar_statements.go
  - internal/parser/languages/cobol_grammar_structure_test.go
  - internal/parser/languages/cobol_grammar_test.go
  - internal/parser/languages/cobol_grammar_walk.go
  - internal/parser/languages/cobol_grammar.go
  - internal/resolver/cobol_references_test.go
  - internal/resolver/cobol_references.go
  - internal/resolver/cross_repo.go
  - internal/resolver/lsp_resolve.go
  - internal/resolver/resolver.go
  - /Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/analyze.go
  - /Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/analyze_test.go
  - /Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/grade/grade.go
  - /Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/grade/grade_test.go
  - /Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/transform/copybook.go
  - /Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/transform/copybook_test.go
  - /Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/transform/pipeline.go
findings:
  critical: 2
  warning: 7
  info: 5
  total: 14
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-10-02T13:49:07Z
**Depth:** standard
**Files Reviewed:** 46
**Status:** issues_found

## Summary

I reviewed the Phase 3 COBOL breadth extractor (procedure, data, flow, refs, mainframe, SQL, dedupe, walk), the new `resolveCobolReferences` pass and the `cobol::` exclusions in the generic resolvers, the BOM byte-offset alignment in the indexer, and the fork's copybook analysis mode (`RunCopybook`, `AnalyzeCopybook`, `PolicyCopybookV1`).

On a full index, the per-document mapping is careful. IDs are encoded, ordering is deterministic, `Parent` links are bounds-checked, and the exactly-one binding is implemented as designed. The serious problems appear once the graph is modified afterward:

- Graph mutation paths that already exist were never taught about the new `cobol::` namespace or the new shared resource nodes.
- An incremental re-index of a callee hands the deterministic CALL edges back to the generic bare-name resolver, which is exactly what D-09 and D-11 forbid.
- Deleting or editing a COBOL file can silently delete DB2, IDMS, and CICS edges that belong to other programs.

There are also semantic gaps:

- The copybook header heuristic misclassifies common members.
- Copybook mode drops nested COPY edges.
- REDEFINES and RENAMES are resolved across the whole program instead of in their COBOL scope, which produces false "ambiguous" results in ordinary code.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Incremental re-index turns bound COBOL CALL edges into bare-name stubs and drops COPY edges

**File:** `internal/indexer/indexer.go:5218-5266` (`restubIncomingRefs`), `internal/indexer/incremental_batch.go:957` (`restubIncomingRefsFromView`), `internal/resolver/cobol_references.go:50`, `internal/graph/graph.go:3275-3297`
**Issue:** When a file is re-indexed, `evictExisting` calls `restubIncomingRefs` and then `Graph.EvictFile`.

- **CALL, LINK, and XCTL edges.** `restubIncomingRefs` re-points every inbound resolvable edge (`EdgeCalls` included) that reaches a referenceable node (`KindFunction` included) to `graph.UnresolvedMarker + n.Name`. A COBOL program node is `KindFunction` with `Name` set to the verbatim PROGRAM-ID. So a cross-file CALL, LINK, or XCTL edge that `resolveCobolReferences` bound under exactly-one becomes `unresolved::PAYCALC`, not `unresolved::cobol::program::PAYCALC`. That target is outside the `cobol::` namespace, so `Resolver.resolveEdge`, `CrossRepoResolver.resolveEdge`, LSP, and `reuseIdentifier` no longer exclude it. It falls through to `resolveFunctionCall`, which binds by bare name with same-file preference (a same-named paragraph in the caller is `KindFunction` too) and cross-repo lift. This is the false deterministic resolution D-09 and D-11 exist to prevent, and `resolveCobolReferences` will never look at the edge again. The edge also keeps `cobol_call=literal` Meta, so it presents as a deterministic literal call.
- **COPY edges.** Copybook file nodes are `KindFile`, which is not referenceable, so they are not restubbed. `EvictFile` then deletes every inbound edge from non-evicted sources (phase 2 of `evictEdgesLocked`). Every program's COPY `imports` edge to a re-indexed or deleted copybook is lost until each of those programs is re-indexed.

**Fix:** Restub COBOL targets into their owned namespace, and include copybook file nodes:
```go
// in restubIncomingRefs / restubIncomingRefsFromView, before the generic stub
stub := graph.UnresolvedMarker + n.Name
if n.Language == "cobol" {
    kind, _ := n.Meta["cobol_kind"].(string)
    key, _ := n.Meta["cobol_name_key"].(string)
    if (kind == "program" || kind == "copybook") && key != "" {
        stub = graph.UnresolvedMarker + "cobol::" + kind + "::" + key
        // also accept EdgeImports here for copybooks, and include KindFile
        // copybook nodes in refIDs
    }
}
```
Then run `resolveCobolReferences`, scoped by `edgeFromInScope`, from the incremental tail (see WR-01) so the restubbed edges rebind under exactly-one. Add a real-path test that re-indexes a callee program and a copybook and asserts the caller's edges are still bound to the same IDs.

### CR-02: Shared DB2, IDMS, and CICS resource nodes are owned by whichever file wrote last; evicting that file deletes other programs' edges

**File:** `internal/parser/languages/cobol_grammar_walk.go:216-229` (`resourceNode`), `internal/parser/languages/cobol_grammar_sql.go:258`, `internal/parser/languages/cobol_grammar_mainframe.go:83,186`
**Issue:** `resourceNode` emits shared nodes with document-scoped IDs: `db2::<schema>/<table>`, `idms::_unqualified/record/<k>`, `cics::transaction/<k>`. Every file that mentions the resource emits the node, each time with `FilePath: w.filePath` and a clone of that file's `docMeta`.

- **Ownership flips.** In the in-memory graph, `addNodeLocked` moves the node into the `byFile` bucket of the last writer. The comment in `cobol_grammar_dedupe.go` notes that SQLite keeps the first insert.
- **Eviction deletes other programs' edges.** When the owning file is deleted, or re-indexed via `evictExisting` → `EvictFile`, `evictFileNodesLocked` removes the shared node. `evictEdgesLocked` then removes every inbound `reads`, `writes`, and `references` edge from other programs. `KindTable` and `KindResource` are not `IsReferenceableSymbol`, so these edges are not even restubbed. They are gone until every other program that touches the table is re-indexed. For a DB2 estate, deleting or editing one program can silently remove the table-impact picture for many others.
- **Provenance depends on backend and order.** The node's provenance Meta (`prov_source_path`, `prov_revision_content_id`, and the `prov_vcs_commit` stamped by `stampSourceRevision`) describes one arbitrary file. Which file depends on the backend and on parallel indexing order, which breaks the phase's determinism goal.

The comment cites "Pitfall 7: shared-node ownership is Phase 4". The consequence is not just ownership metadata, though: other files' edges are lost.
**Fix:** Keep shared resources out of per-file eviction. Either:
- give them a stable synthetic owner path, for example `FilePath: ""` or `<repo>/.cobol/resources`, and remove orphans in a post-resolve GC; or
- emit a per-file anchor node (`<file>::@resource/<id>`) whose edges point at the shared node, so eviction only removes the file's own anchors.

In both cases, strip the per-document `prov_*` keys from the shared node. Only keep `resource_type`, `cobol_name_norm`, `prov_range_absence=shared_resource`, and the extractor version.

## Warnings

### WR-01: `resolveCobolReferences` never runs on the incremental path, and bound edges are never re-checked for exactly-one

**File:** `internal/resolver/resolver.go:1662`, `internal/resolver/resolver.go:2652` (`ResolveFilesAndIncoming`), `internal/resolver/cobol_references.go:59`
**Issue:** The pass is called only from `ResolveAllContext`. The watcher and daemon incremental path (`incrementalWatcherPaths` → `ResolveFilesAndIncoming`) never calls it. As a result:
- a newly saved or edited program's literal CALL and COPY edges stay `unresolved::cobol::…` until some later full resolve;
- adding the only matching copybook does not bind pending COPYs.

The pass also iterates only `EdgesWithUnresolvedTarget()`. Once an edge is bound, adding a second candidate with the same `cobol_name_key` in the same repo never re-stamps it as `ambiguous`. The graph keeps claiming an exactly-one binding that no longer holds. That contradicts the "re-running is idempotent" claim in the doc comment.
**Fix:** Call a scoped `resolveCobolReferences()` from `ResolveFilesAndIncoming`, or from the incremental tail, after the per-file pass. Also include already-bound COBOL edges. For example, keep `cobol_target_key` in edge Meta at emit time so bound edges can be re-evaluated, and revert an edge to its placeholder when the candidate count is no longer 1.

### WR-02: Cross-file CALL can bind to a nested (contained) program in another file

**File:** `internal/resolver/cobol_references.go:54,81-92`
**Issue:** `cobolCandidateIndex` indexes every `cobol_kind=program` node, including contained programs (QualName `OUTER/INNER`). The same-file preference correctly favours a sibling nested program. But when the caller is in another file and the repository has exactly one `INNER`, a `CALL 'INNER'` binds to a program that COBOL scoping makes uncallable from outside its container. Only top-level programs, plus nested programs within their container (COMMON), are visible to a separately compiled caller. The result is a false deterministic binding.
**Fix:** Index nested programs only for same-file lookup:
```go
for _, c := range candidates {
    nested := strings.Contains(c.QualName, "/")
    if c.FilePath == e.FilePath { sameFile = append(sameFile, c) } else if !nested { crossFile = append(crossFile, c) }
}
candidates = sameFile; if len(candidates) == 0 { candidates = crossFile }
```
Alternatively, stamp `cobol_nested=true` on the program node at extraction and filter on that.

### WR-03: REDEFINES and RENAMES are resolved program-wide, giving false "ambiguous" results in ordinary code

**File:** `internal/parser/languages/cobol_grammar_data.go:64-90,111-131`
**Issue:** `cobolDataBind` resolves REDEFINES and RENAMES operands with `idx.lookup(owner, name, qualifiers)`, which searches every item of the program. COBOL fixes the scope of both:
- The REDEFINES object must be the immediately preceding entry at the same level, and it cannot be qualified.
- RENAMES operands must lie inside the 01 record that immediately precedes the 66 entry.

Repeating a child name across records is routine, for example `05 WS-DATE` in several 01 records with `05 WS-DATE-X REDEFINES WS-DATE`. That pattern produces `cobol_unresolved_reason=ambiguous` with candidate lists, even though the language makes the target unique. The edges then point at `unresolved::cobol::data::…` instead of the real item.
**Fix:** In `cobolDataSection`, record for each entry its previous same-level sibling and its enclosing 01 record. Bind REDEFINES to that sibling when the names match, and keep `not_declared_in_program` otherwise. Bind RENAMES through a lookup restricted to items whose ancestor chain ends at that 01 record.

### WR-04: The copybook header heuristic misclassifies common members (fork)

**File:** `/Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/preprocessor/transform/copybook.go:77-94`
**Issue:** `opensWithDataDescription` picks the data header for any member whose first token is `EXEC`. Procedure-division members very often open with `EXEC CICS …` or `EXEC SQL SELECT/UPDATE/OPEN …`. Those get wrapped under `WORKING-STORAGE SECTION`, so they parse with errors and degrade, and their statements are lost.

The opposite case also fails. Data members whose first code line is not a level number select the procedure header and fail to parse:
- `FD`/`SD` file descriptions (FILE SECTION copybooks);
- a nested `COPY X.`;
- an Endevor `-INC` line. `ln.code` only looks at columns 8-72, so the `-INC` token is never seen.

The tests (`copybook_test.go:109-151`) cover only `EXEC SQL DECLARE`.
**Fix:**
- Treat `EXEC SQL DECLARE`, `EXEC SQL INCLUDE`, and `EXEC SQL BEGIN DECLARE SECTION` as data, and every other `EXEC` as procedure.
- Treat `FD` and `SD` as data, ideally with a `FILE SECTION` header variant.
- Skip `COPY`, `-INC`, and `++INCLUDE` lines and decide on the next code line.
- Add test cases for `EXEC CICS`, `FD`, and a leading `COPY`.

### WR-05: Copybook mode emits no reference edges, so nested COPY and procedure members vanish

**File:** `internal/parser/languages/cobol_grammar.go:220-241`
**Issue:** The copybook branch runs only `cobolDataSection` and `cobolDataBind`.
- A `COPY Y` inside copybook `X.cpy` only sets `cobol_contains_copy=true` on the file node. No `imports` edge to `unresolved::cobol::copybook::Y` is emitted, so the copybook-to-copybook dependency chain is missing from the graph.
- For members that take the procedure header, nothing beyond the file node is emitted: no paragraphs, CALLs, PERFORMs, EXEC SQL, or CICS. There is also no `prov_analysis_absence` reason saying why.

`docs/languages.md` says copybooks are "analyzed in the producer's copybook mode" without stating either limit.
**Fix:** In the copybook branch, call `cobolCopy(w, i)` for every `copy_statement` and `exec_sql_include` observation. `cobolCopy` already falls back to `from = w.filePath` when there is no program. For procedure-header members, stamp `docMeta["prov_analysis_absence"] = "copybook_procedure_member_unsupported"`, or extend extraction to them. Document the limit in `docs/languages.md`.

### WR-06: A copybook analysis error now drops the copybook's file node, so every COPY of it reports `missing`

**File:** `internal/parser/languages/cobol_grammar.go:184-189`
**Issue:** Before this phase, a `.cpy` file always produced an identity file node. Now any `AnalyzeCopybook` error, such as an exceeded parse budget, the wall-clock ceiling, or a transform failure, makes `Extract` return `nil, err`. The copybook disappears from the graph. `resolveCobolReferences` then stamps every program's COPY of it `cobol_unresolved_reason=missing`, which is false, because the member exists in the repository. Large DCLGEN members are the most likely to hit the budget.
**Fix:** On copybook analysis failure, return the identity-only file node, as Phase 2 did, with the binding keys and an absence reason:
```go
if err != nil && copybook {
    meta := map[string]any{ /* prov_source_path, prov_source_id, prov_revision_content_id, prov_extractor_version */ 
        "prov_analysis_absence": "copybook_analysis_failed",
        "cobol_kind": "copybook", "cobol_name_key": key, "cobol_name_norm": cobolNameNormRule}
    return &parser.ExtractionResult{Nodes: []*graph.Node{fileNode(meta)}}, nil
}
```

### WR-07: A cursor declared twice is labeled `cursor_not_declared`

**File:** `internal/parser/languages/cobol_grammar_sql.go:151-156`
**Issue:** `if len(decls) != 1` stamps `cobol_unresolved_reason=cursor_not_declared` for both zero and two-or-more DECLAREs of the same cursor in one program. That breaks the D-12 reason vocabulary: ambiguity is reported as absence, and the candidate DECLAREs are not recorded. A related gap: DB2 programs often put `EXEC SQL DECLARE … CURSOR` in WORKING-STORAGE, which the pinned grammar does not admit (see the comment at lines 90-91). Every OPEN and FETCH of such a cursor is silently reported as "not declared", with no indication that the grammar is the cause.
**Fix:** Split the case:
```go
switch len(decls) {
case 0: n.Meta["cobol_unresolved_reason"] = "cursor_not_declared"
case 1: /* emit reads */
default:
    n.Meta["cobol_unresolved_reason"] = "ambiguous"
    n.Meta["cobol_candidate_symbols"] = /* sorted statement QualNames of decls */
}
```

## Info

### IN-01: DB2 table IDs use a second identity scheme

**File:** `internal/parser/languages/cobol_grammar_sql.go:258`
**Issue:** COBOL tables are `db2::<SCHEMA>/<TABLE>`, while the repo's existing table identity is `gortexsql.TableNodeID(dialect, schema, table)`, which gives `db::db2::SCHEMA.TABLE` (`internal/sql/sql.go:832`). It is used by migration DDL. The same physical DB2 table seen in DDL and in COBOL will never be the same node. D-15 chose the scheme, but the divergence should be recorded so Phase 4 (ID-04) can reconcile it.
**Fix:** Note the divergence in D-15 or in `03-PATTERNS.md`. Alternatively, add a `same_as` bridge in Phase 4.

### IN-02: Copybook detection relies only on the `.cpy` extension, and extension handling is asymmetric

**File:** `internal/parser/languages/cobol_grammar.go:110,173,235-237`, `internal/parser/languages/cobol_grammar_refs.go:134`
**Issue:**
- `.cbl` and `.cob` members used as copybooks, `.copy`, and mixed-case `.Cpy` are never `cobol_kind=copybook`. `Extensions()` also omits `.Cpy`, `.cPy`, and similar spellings that `EqualFold` would otherwise accept.
- The file key strips any extension, but a COPY key strips only `.CPY`, so `COPY 'X.cbl'` can never match `X.cbl`.

**Fix:** Strip the same set of extensions on both sides, and document that only `.cpy` members are copybook candidates.

### IN-03: Program symbols are embedded in IDs and placeholders without `cobolIDSegment`

**File:** `internal/parser/languages/cobol_grammar_identity.go:102-112`, `internal/parser/languages/cobol_grammar_refs.go:55`, `internal/parser/languages/cobol_grammar_flow.go:203,271`, `internal/parser/languages/cobol_grammar_data.go:141`
**Issue:** The program symbol is the verbatim `program_name`, and a literal PROGRAM-ID may contain `:`, `/`, or `#`. It now prefixes every breadth node ID and every `unresolved::cobol::{dynamic,data,label}::<file>::<prog>::…` placeholder. T-03-22 and Pitfall 9 encode every other name-derived segment, but this one is not encoded, so source text can still add path segments or `::` markers.
**Fix:** Encode each program-name component with `cobolIDSegment` when building `symbol` in `cobolProgramSymbols`. This is an ID change, so bump `cobolGrammarExtractorVersion`.

### IN-04: Dedupe discards the second occurrence's semantics

**File:** `internal/parser/languages/cobol_grammar_dedupe.go:26-46`
**Issue:**
- A `PERFORM P` and a `GO TO P` on the same line are both `EdgeCalls` with the same From, To, and Line. They collapse into one edge that keeps only the first `cobol_flow` and the first edge's `cobol_thru_*` and `cobol_goto_depending*` keys.
- When the survivor has no `cobol_statement_symbol` but the duplicate has one, `cobol_statement_symbols` gets one entry while `cobol_occurrence_count` is 2, so the two lists no longer line up.

**Fix:** When merging, union the differing `cobol_flow` values into a `cobol_flows` list. Pad `cobol_statement_symbols` with `""` so it stays index-aligned with the occurrence count.

### IN-05: The projection test fixture uses a different type than the extractor

**File:** `internal/neo4jprojection/properties_test.go:345`, `internal/parser/languages/cobol_grammar_flow.go:281-282`
**Issue:**
- The fixture sets `cobol_renames` to a string, but `cobolDataClauses` writes `[]string`, so the test does not cover the real projected type.
- `cobol_depending_positions` stores integers as `[]string`. That is harmless, but consumers have to parse it.

**Fix:** Use `[]string{"WS-A"}` in the fixture. Consider storing the positions as `[]int`, which `gob.Register([]int{})` already supports.

---

_Reviewed: 2026-10-02T13:49:07Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
