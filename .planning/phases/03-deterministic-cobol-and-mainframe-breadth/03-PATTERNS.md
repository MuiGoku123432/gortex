# Phase 3: Deterministic COBOL and Mainframe Breadth - Pattern Map

**Mapped:** 2026-10-01
**Files analyzed:** 27 new or modified files (20 Gortex, 2 Gortex module files, 5 in the fork)
**Analogs found:** 26 / 27

Privacy: every excerpt below comes from tracked Gortex or fork source, or from invented fixtures. There is no estate content.

All analog paths were checked with `git ls-files` and are tracked source. Gortex analogs are in `/Users/e1001547-mbp-it/repos/mine/GoApps/gortex`. Fork analogs are in `~/repos/mine/devDeps/tree-sitter-cobol-upgrade`, on branch `gsd/v0.26.0-estate-parse-recovery` at HEAD `f9eaf99`, which matches the pin. No gitignored mirror paths are used.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/parser/languages/cobol_grammar.go` (M) | extractor | transform | itself (P2) | exact |
| `internal/parser/languages/cobol_grammar_walk.go` (N) | utility | transform | `cobol_grammar_identity.go` (parentOf, Generated-text read) | role-match |
| `internal/parser/languages/cobol_grammar_procedure.go` (N) | extractor | transform | `cobol_grammar.go` program loop + `java.go` defines/member_of + `go_dataflow.go` KindLocal | exact (composite) |
| `internal/parser/languages/cobol_grammar_data.go` (N) | extractor | transform | `cobol_grammar_identity.go` (stack walk, `#k` ordinals) | role-match |
| `internal/parser/languages/cobol_grammar_refs.go` (N) | extractor | transform | `cobol.go` regex COPY/CALL/dyncall (parity only) + `cobol_grammar.go` meta | role-match |
| `internal/parser/languages/cobol_grammar_mainframe.go` (N) | extractor | transform | `kubernetes.go` KindResource + `go_sql.go` shared-node dedupe | role-match |
| `internal/parser/languages/cobol_grammar_sql.go` (N) | extractor | transform | `go_sql.go` `emitGoSQLEvents` | exact |
| `internal/parser/languages/cobol_grammar_test.go` (M) + sibling `cobol_grammar_*_test.go` (N) | test | — | `cobol_grammar_test.go` | exact |
| `internal/resolver/cobol_references.go` (N) | resolver pass | batch | `internal/resolver/lua_imports.go` | exact |
| `internal/resolver/cobol_references_test.go` (N) | test | — | `internal/resolver/lua_imports_test.go` | exact |
| `internal/resolver/resolver.go` (M: `resolveEdge` case + `ResolveAllContext` call) | resolver | request-response | the `grpc::`/`pyrel::` cases in the same switch | exact |
| `internal/resolver/cross_repo.go` (M) | resolver | request-response | the `import::` case in `CrossRepoResolver.resolveEdge` | exact |
| `internal/resolver/lsp_resolve.go` (M: `identifierFromTarget`) | utility | transform | the same `case` list | exact |
| `internal/indexer/incremental_resolve.go` (M: `reuseIdentifier`) | utility | transform | the same `case` list | exact |
| `internal/indexer/source_revision.go` (M: CR-01 BOM shift) | indexer hook | file-I/O | `stampSourceRevision` in the same file | exact |
| `internal/indexer/source_revision_test.go` (M) or `internal/indexer/cobol_bom_test.go` (N) | test | — | `source_revision_test.go` | exact |
| `cmd/gortex/cobol_tracer_test.go` (M) | test (acceptance) | request-response | itself (P2) | exact |
| `internal/neo4jprojection/properties_test.go` (M, optional 03-09 smoke check) | test | transform | `TestProjectNode_CobolProvenance` (same file, near line 244) | exact |
| `go.mod`, `go.sum` (M, D-16 re-pin) | config | — | P2 02-02 Task 1 step 1 | exact |
| `internal/parser/languages/cobol_grammar.go` const `cobolApprovedPreprocessorModule` (M, D-16) | config | — | line 39 | exact |
| FORK `preprocessor/analyze.go` (M: `AnalyzeCopybook`) | producer API | transform | `Analyze` / `NewEmbeddedAnalyzer` (P2 commit f9eaf99) | exact |
| FORK `preprocessor/analyze_test.go` (M) | test | — | `TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer` (f9eaf99) | exact |
| FORK `preprocessor/transform/pipeline.go` (M: copybook wrapper inserts) | transform | transform | Endevor `-INC` `KindInsert` edit (`endevor.go:219`) + `Run` insert branch | role-match |
| FORK `preprocessor/grade/grade.go` (M: copybook-aware policy) | config/policy | transform | `PolicyV1`, `PolicyFor`, `requiredAlwaysAnchors` | role-match |
| FORK `preprocessor/grade/grade_test.go`, `transform/pipeline_test.go` (M) | test | — | existing tests in those packages | role-match |
| `internal/parser/languages/cobol_grammar_unsupported.go` | stub | — | no change needed (it only exposes the type) | n/a |
| Phase planning: fork commit, push checkpoint, re-pin plan | plan | — | `02-01-PLAN.md` (lines 100-158) and `02-02-PLAN.md` (Task 1 step 1) | exact |

---

## Pattern Assignments

### `internal/parser/languages/cobol_grammar.go` (extractor, transform). Modified by 03-01 and the D-16 last wave

**Analog:** itself (292 lines).

**Build tag and imports** (lines 1-23). Every new `cobol_grammar_*.go` file copies the first line `//go:build darwin || linux`. Note: the 02-02 plan text says `!windows`, but the shipped code is `darwin || linux`, and the stub is `cobol_grammar_unsupported.go` (`//go:build !darwin && !linux`).
```go
//go:build darwin || linux

package languages

import (
	...
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/sourcemap"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/transform"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)
```

**Version bump** (lines 40-42). Change it to `/3`. This literal is also asserted in `cmd/gortex/cobol_tracer_test.go:176`. The value `gortex-cobol-grammar/1` at `internal/neo4jprojection/properties_test.go:244` is an independent fixture and does not need to change.
```go
	cobolGrammarExtractorVersion = "gortex-cobol-grammar/2"
```

**Program node, then structural edge with no prov_* blob** (lines 230-249). This is the template for every new symbol node (paragraph, section, division, data item). Clone `docMeta`, add `cobol_kind`, `prov_observation_kind`, `prov_affected`, stamp the range, then emit `defines` with `Origin`/`Confidence`/`ConfidenceLabel` and **no Meta** (D-17, Assumption A2).
```go
	programs, containmentUnresolved, unnamed := cobolProgramSymbols(h)
	for _, p := range programs {
		o := h.Facts.Observations[p.index]
		meta := maps.Clone(docMeta)
		meta["cobol_kind"] = "program"
		meta["prov_observation_kind"] = o.Kind
		meta["prov_affected"] = o.Affected
		id := filePath + "::" + p.symbol
		node := &graph.Node{
			ID: id, Kind: graph.KindFunction, Name: p.name, QualName: p.symbol,
			FilePath: filePath, Language: "cobol", Meta: meta,
		}
		cobolStampRange(node, o.Original)
		result.Nodes = append(result.Nodes, node)
		result.Edges = append(result.Edges, &graph.Edge{
			From: fileNode.ID, To: id, Kind: graph.EdgeDefines,
			FilePath: filePath, Line: node.StartLine,
			Origin: graph.OriginASTResolved, Confidence: 1.0, ConfidenceLabel: "EXTRACTED",
		})
	}
```
03-01 adds the walker call sites right after this loop, for example `w := newCobolWalk(h)` and then `cobolProcedure(w, ...)`, `cobolData(...)`, and so on. Later plans then touch only their own files, as research's slicing note requires. `programs` (index→symbol) has to be passed to the walkers so they can build `<path>::<prog>/...` IDs.

**Document-level flags written after the loop** (lines 250-257). Use the same "set after cloning" trick for any new document-only key, such as an edge-dedupe count, so it never leaks into node Meta.

**Range stamping** (lines 267-290). Reuse `cobolStampRange` as is for single-observation nodes. Research Pitfall 1 needs a **two-projection extent variant** for paragraph and section extents (header start → last observation end). Put it next to this function or in `cobol_grammar_walk.go`, and keep the same absence vocabulary (`prov_range_exact=false`, `prov_range_absence=...`). The new reason is `paragraph_extent_unmapped`.
```go
func cobolStampRange(node *graph.Node, r *sourcemap.Projection) {
	meta := node.Meta
	switch {
	case r == nil || r.NoImage:
		meta["prov_range_exact"] = false
		meta["prov_range_absence"] = "no_original_projection"
		return
	case r.Synthetic:
		meta["prov_range_exact"] = false
		meta["prov_range_absence"] = "synthetic_projection"
		return
	}
	node.StartLine = int(r.Start.Row) + 1
	...
	meta["prov_start_byte"] = int(r.Start.Byte)
	meta["prov_end_byte"] = int(r.End.Byte)
	meta["prov_range_exact"] = r.Exact
}
```
Relationship edges (PERFORM, CALL, COPY, reads/writes) need the same `prov_*` range keys, but on `edge.Meta`, and `Edge.Line` = start row + 1. The function takes a `*graph.Node`, so the planner should either add an edge-meta variant or factor the meta-writing half into one `func(meta map[string]any, r *sourcemap.Projection) (line int)` helper used by both. Nodes, edges, and extents make three uses, which satisfies the rule of three.

**Copybook branch, replaced in the D-16 last wave** (lines 171-189). Today it short-circuits with `prov_analysis_absence=copybook_standalone_analysis_unsupported`. After the re-pin it calls the new fork entry point under the same slot (lines 190-195), and then runs only the data-item walker with the **copybook file node as owner**:
```go
	var h handoff.Handoff
	func() {
		cobolAnalyzeSlot <- struct{}{}
		defer func() { <-cobolAnalyzeSlot }()
		h, err = e.analyzer.Analyze(context.Background(), sourceID, src)
	}()
	if err != nil {
		return nil, fmt.Errorf("enhanced COBOL analysis of %s: %w", filePath, err)
	}
	if h.Tool.CompiledLanguageID != e.approvedGrammarID { ... errCobolGrammarNotApproved ... }
```
Keep the grammar-ID check for copybooks too. That changes the "wrong pin still succeeds" expectation in `cobol_grammar_test.go:477-503`, which must be rewritten. The research default puts `cobol_kind=copybook` on the copybook file node's meta, and the resolver relies on it. `cobolStampRange` already marks wrapper bytes as `synthetic_projection`.

---

### `internal/parser/languages/cobol_grammar_walk.go` (utility, transform). New in 03-01

**Analog:** `internal/parser/languages/cobol_grammar_identity.go`.

**Bounds-checked parent access** (lines 44-48). Copy this guard into the children index builder. Never index `obs[o.Parent]` without the check.
```go
	obs := h.Facts.Observations
	parentOf := func(i int) (int, bool) {
		p := obs[i].Parent
		return p, p >= 0 && p < len(obs) && p != i
	}
```

**Name from the Generated span** (line 59). This becomes `w.text(i)`. Research Pitfall 2 says paragraph, section, and SQL table names must come from the header or table span text, not from child WORDs.
```go
		name := string(h.Generated[o.Generated.Start:o.Generated.End])
```

**Name folding precedent** (line 77). `cobol-name-norm/1` (D-18) extends this inline closure: upper-fold, strip one quote level, un-double embedded quotes, and strip a `.cpy` suffix for COPY. Refs, mainframe, SQL, and the resolver use it, which is 3+ uses, so it belongs in the walk file as a single `cobolNameNorm(raw string) string` plus a `cobolNameNormRule = "cobol-name-norm/1"` constant. The resolver package cannot import `languages`. The resolver gets candidate names already normalized on the node, or a duplicated one-liner with a `// note:` comment. The planner picks one.
```go
	key := func(name string) string { return strings.ToUpper(strings.Trim(name, `"'`)) }
```

The proposed `cobolWalk` skeleton is in 03-RESEARCH.md §Code Examples ("Shared walk helper"). Use it as written. `parsefacts.Validate` guarantees `Parent < i`.

---

### `internal/parser/languages/cobol_grammar_procedure.go` (extractor, transform). 03-01 tracer, 03-03 structure, 03-05 control flow

**Analog A, symbol plus containment:** `internal/parser/languages/java.go:767-774`. File `defines` → symbol, and symbol `member_of` → container. This is the research containment choice; `EdgeContains` is not used.
```go
		result.Nodes = append(result.Nodes, node)
		result.Edges = append(result.Edges, &graph.Edge{
			From: fileID, To: id, Kind: graph.EdgeDefines, FilePath: filePath, Line: startLine1,
		})
		classID := filePath + "::" + className
		result.Edges = append(result.Edges, &graph.Edge{
			From: id, To: classID, Kind: graph.EdgeMemberOf, FilePath: filePath, Line: startLine1,
		})
```
Add `Origin: graph.OriginASTResolved, Confidence: 1.0, ConfidenceLabel: "EXTRACTED"` to match `cobol_grammar.go:244-248`.

**Analog B, statement nodes (`KindLocal`, `member_of` only, no `defines`):** `internal/parser/languages/go_dataflow.go:91-116` (D-17):
```go
	w.result.Nodes = append(w.result.Nodes, &graph.Node{
		ID:        id,
		Kind:      graph.KindLocal,
		Name:      name,
		FilePath:  w.filePath,
		StartLine: line,
		EndLine:   line,
		Language:  "go",
	})
	w.result.Edges = append(w.result.Edges, &graph.Edge{
		From:     id,
		To:       w.ownerID,
		Kind:     graph.EdgeMemberOf,
		FilePath: w.filePath,
		Line:     line,
		Origin:   graph.OriginASTResolved,
	})
```
COBOL statement nodes carry the full cloned `docMeta` plus `cobol_kind=statement` and `cobol_statement_kind=<obs kind>`, but their `member_of` edge has no prov blob.

**Analog C, `#k` ordinal for same-path duplicates:** `cobol_grammar_identity.go:82-111`. The same scheme applies to paragraphs (same-path duplicates), statements (`@<kind>#k` for k ≥ 2), and the nested-program vs 01-record collision fallback.
```go
	occurrences := make(map[string]int) // base symbol -> programs seen
	...
			occurrences[base]++
			symbol := base
			if k := occurrences[base]; k >= 2 {
				symbol += "#" + strconv.Itoa(k)
			}
```

**Paragraph flow edge** (`calls` + `cobol_flow`). The edge shape is `cobol_grammar.go:244-248` with `Kind: graph.EdgeCalls` and Meta = cloned `docMeta` plus range keys plus `cobol_flow`, `cobol_statement_symbol` (QualName only, Pitfall 5), and `prov_affected`. Label placeholders use `unresolved::cobol::label::...` with `cobol_unresolved_reason`. The PERFORM/THRU child-walk skeleton is in 03-RESEARCH.md §Code Examples ("Procedure headers and PERFORM targets").

**Edge dedupe (Pitfall 4).** No in-repo extractor dedupes by `(From,To,Kind,Line)` with meta merge. The closest precedent is `go_sql.go:189-190,239-241` (a `seen` map keyed by `callerID + "->" + strID`). Copy that `map[key]struct{}` idiom with key `From|To|Kind|Line`. The first occurrence's meta wins, and later ones append to `cobol_statement_symbols` and bump `cobol_occurrence_count`.

**Regex parity checklist only (never re-registered):** `cobol.go:121-125` (sections as `KindMethod`, superseded), plus its second-walk paragraph/PERFORM section after line 163.

---

### `internal/parser/languages/cobol_grammar_data.go` (extractor, transform). 03-04, then copybook reuse in the D-16 wave

**Analog:** the open-stack walk in `cobol_grammar_identity.go:83-135`, adapted to level numbers (pop while `top.level >= level`, with the 01/77/66/88 rules in RESEARCH §Data items):
```go
	type frame struct {
		cobolProgramSymbol
		hasChildren bool // another program was opened inside this one
	}
	var open []frame
	for i, o := range obs {
		switch o.Kind {
		case "program_definition":
			...
			for len(open) > 0 && remaining[key(open[len(open)-1].name)] == 0 {
				...
				open = open[:len(open)-1]
			}
			base := name
			if len(open) > 0 {
				base = open[len(open)-1].symbol + "/" + name
```
Kinds: 01/77 → `graph.KindVariable`, 02-49/66/88/FILLER → `graph.KindField`. Edges: `defines` from the file, `member_of` → parent item or data section (the java.go pattern above). REDEFINES/RENAMES → `graph.EdgeReferences` with `cobol_ref_role`. The walker must take an **owner ID prefix** parameter (program symbol vs copybook file ID), because the D-16 wave reuses it with the copybook file node as owner (D-16: copybook-owned items are not duplicated per program).

---

### `internal/parser/languages/cobol_grammar_refs.go` (extractor, transform). 03-06

**Parity analog (behavior only):** `internal/parser/languages/cobol.go:127-161`. That regex extractor emitted COPY as `EdgeImports` → `unresolved::import::` and dynamic CALL as `unresolved::dyncall::<ident>` with `OriginASTInferred`. Phase 3 **replaces both namespaces** with `unresolved::cobol::copybook::<NORM>`, `unresolved::cobol::program::<NORM>`, and `unresolved::cobol::dynamic::program::<callerProgramID>::<IDENT>`. Pitfall 6 says never emit bare `unresolved::<NAME>`.
```go
		result.Edges = append(result.Edges, &graph.Edge{
			From: fileNode.ID, To: "unresolved::import::" + mod,
			Kind: graph.EdgeImports, FilePath: filePath, Line: line,
		})
	...
		result.Edges = append(result.Edges, &graph.Edge{
			From: fileNode.ID, To: "unresolved::dyncall::" + string(src[m[2]:m[3]]),
			Kind: graph.EdgeCalls, FilePath: filePath, Line: line,
			Origin: graph.OriginASTInferred,
		})
```
Phase 3 differences:
- From = the innermost procedure container, not the file node. COPY outside any program uses the file node (Pitfall 3).
- `Origin: graph.OriginASTResolved`, Confidence 1.0.
- Full prov Meta.
- `COPY IDMS` → `unresolved::cobol::copybook::idms/<KIND|_>/<NAME>` with reason `idms_dictionary_member`, never bound (D-19).

**Prefixing behaviour to rely on:** `internal/indexer/indexer.go:2314-2321`. Edges whose `To` starts with `unresolved::` are left unprefixed, and every other `To` gets `<repo>/` prepended. In-file targets (paragraph, data item) and shared resource IDs therefore become repo-scoped automatically.
```go
		if strings.HasPrefix(e.To, unresolvedMarker) {
			e.To = intern.String(e.To)
		} else {
			e.To = intern.String(prefix + e.To)
		}
```

**Literal-segment encoding (Pitfall 9).** No analog exists. Percent-encode `:`, `/`, `#`, `%`, and control bytes in literal-derived segments only, and keep `cobol_target_raw`.

---

### `internal/parser/languages/cobol_grammar_mainframe.go` (extractor, transform). 03-07 IDMS + CICS

**Analog A, `KindResource` emission:** `internal/parser/languages/kubernetes.go:87-97`:
```go
	result.Nodes = append(result.Nodes, &graph.Node{
		ID: resID, Kind: graph.KindResource, Name: name,
		QualName: kind + "/" + name,
		FilePath: filePath, StartLine: resLine, EndLine: resLine,
		Language: "yaml",
		Meta:     resMeta,
	})
```
For COBOL, `resMeta` carries `resource_type=idms_record|idms_set|cics_transaction|cics_map|cics_mapset` (precedent: `dbt.go:145` uses the same key), plus `cobol_name_norm`. Do **not** emit a file→resource `defines` the way kubernetes does: resources are shared, and only relationship edges point at them (RESEARCH kind map).

**Analog B, shared node first sighting plus per-result dedupe:** `internal/parser/languages/go_sql.go:255-277`:
```go
		for _, ref := range e.tables {
			tableID := sql.TableNodeID(dialect, ref.Schema, ref.Table)
			if _, ok := seenNodes[tableID]; !ok {
				seenNodes[tableID] = struct{}{}
				...
				result.Nodes = append(result.Nodes, &graph.Node{
					ID:       tableID,
					Kind:     graph.KindTable,
					Name:     ref.Table,
					FilePath: filePath, // first sighting; not authoritative
					...
```
IDs follow RESEARCH §ID grammar: `idms::<SS|_unqualified>/record/<NAME>`, `cics::map/<MAPSET|_unqualified>/<MAP>`, and so on. CICS LINK/XCTL reuse the 03-06 program and dynamic placeholder builders from `cobol_grammar_refs.go`; there is no second implementation.

---

### `internal/parser/languages/cobol_grammar_sql.go` (extractor, transform). 03-08

**Analog:** `internal/parser/languages/go_sql.go:182-292` (`emitGoSQLEvents`). Table nodes use `graph.KindTable` with Meta keys `table`, `dialect`, and `schema` (only when present), which the existing orphan/unreferenced-table analyzers read (lines 262-270):
```go
				meta := map[string]any{
					"table":   ref.Table,
					"dialect": dialect,
				}
				if ref.Schema != "" {
					meta["schema"] = ref.Schema
				}
```
Differences from go_sql:
- `dialect=db2`.
- The ID is `db2::<SCHEMA|_unqualified>/<TABLE>`, not `sql.TableNodeID` (whose scheme is `db::<dialect>::...`). The planner should check whether reusing `sql.TableNodeID` breaks D-15, which needs an explicit `_unqualified`.
- Edges are `graph.EdgeReads`/`EdgeWrites`/`EdgeReferences` (not `EdgeQueries`) with `Origin: graph.OriginASTResolved` (not `OriginTextMatched`) and `sql_verb`, `sql_alias(es)`, and `sql_via_cursor` meta.
- Do **not** copy go_sql's `KindString` registry node.

---

### Tests: `internal/parser/languages/cobol_grammar_test.go` (modified) and sibling `cobol_grammar_{structure,data,flow,refs,mainframe,sql}_test.go` (new)

**Analog:** `internal/parser/languages/cobol_grammar_test.go` (593 lines, already large, so new behaviours belong in siblings, one per walker).

- Header and imports: lines 1-21 (`//go:build darwin || linux`, testify `assert`/`require`, fork `preprocessor`/`handoff`/`sourcemap`).
- Fixture style: lines 23-31. A comment states the fixtures are invented, and each fixture is a 7-column fixed-format raw string.
- Helpers to reuse, not duplicate: `extractCobolGrammar` (95-101), `cobolProgramNodes` (103-111; filtering on `Meta["cobol_kind"]` generalizes to a by-`cobol_kind` filter), `cobolFileNode` (121-131), `cobolTestHandoff` (134-142; compare persisted values with the producer's observations), `assertCobolRange` (145-158; applies to every new node), `findEdgeTo` (`cobol_test.go:287`), and `cobolDocumentKeys` (86-93; assert on every new node and relationship edge).
- Test shape: `TestCobolGrammar_ProgramNode` (160-198) shows how to assert ID, Kind, QualName, `cobol_kind`, empty scope fields, `assertCobolRange`, and the defines edge with `Origin`/`Confidence`/`ConfidenceLabel`. Note line 177, `require.Len(t, result.Edges, 1)`: that test **breaks** once 03-01 adds paragraph edges and must be loosened to filter by kind.
- `TestCobolGrammar_CopybookSkipsAnalysis` (477-503) is rewritten in the D-16 wave.
- `TestCobolGrammar_DegradedAmber` (439-475) is the pattern for "Affected observations emitted and marked".

---

### `internal/resolver/cobol_references.go` (resolver pass, batch). 03-06

**Analog:** `internal/resolver/lua_imports.go` (127 lines), a whole-graph post-pass that binds an owned placeholder namespace.

**Language gate, candidate index, edge sweep, reindex batch** (lines 25-80):
```go
func (r *Resolver) resolveLuaRequires() {
	if !r.graphHasLanguage("lua") && !r.graphHasLanguage("luau") {
		return
	}
	...
	for file := range graph.FileLanguageNodesSeq(r.graph) { ... }

	var reindexBatch []graph.EdgeReindex
	for e := range r.graph.EdgesByKind(graph.EdgeImports) {
		if e == nil || !strings.HasPrefix(e.To, "unresolved::import::") {
			continue
		}
		if !r.edgeFromInScope(e.From) {
			continue
		}
		...
		oldTo := e.To
		e.To = resolved
		e.Origin = graph.OriginASTResolved
		reindexBatch = append(reindexBatch, graph.EdgeReindex{Edge: e, OldTo: oldTo})
	}
	if len(reindexBatch) > 0 {
		r.noteImportEdgeReindexes(reindexBatch)
		r.graph.ReindexEdges(reindexBatch)
	}
}
```
**Ambiguity refusal precedent:** `resolveLuaRobloxRequire` (lines 116-127). It returns `""` on a second distinct candidate. COBOL must also record `cobol_candidate_ids` (sorted) and `cobol_unresolved_reason=ambiguous`, not just refuse.

Seams to use (verified):
- `r.graphHasLanguage("cobol")`: `internal/resolver/language_gate.go:15-20`.
- `r.nodesByKindLang(graph.KindFunction, "cobol")` and `r.nodesByKindLang(graph.KindFile, "cobol")`, defined directly below in `language_gate.go`. **Prefer these over `FileLanguageNodesSeq`**, which yields only `{ID, Language}` with no Meta, so it cannot see `cobol_kind=program|copybook` or `RepoPrefix`.
- `r.callerRepoPrefix(e)`: `resolver.go:5741-5751`.
- `r.edgeFromInScope(e.From)`: `resolver.go:1876-1882`.
- `r.graph.EdgesWithUnresolvedTarget()`: `internal/graph/graph.go:1613`. Use it, or iterate `EdgesByKind(EdgeCalls)` plus `EdgesByKind(EdgeImports)` the way Lua does.
- Meta-only stamps (missing/ambiguous): `graph.EdgeMetaBatchPersister` (`internal/graph/store.go:1565-1567`), used as in `terminal.go:285-291`:
```go
	if len(changed) > 0 {
		if p, ok := r.graph.(graph.EdgeTerminalStampPersister); ok {
			p.PersistEdgeTerminalStamps(changed)
		} else if p, ok := r.graph.(graph.EdgeMetaBatchPersister); ok {
			p.PersistEdgeAttributesBatch(changed)
		}
	}
```
The in-memory graph does not implement the persister. In-place Meta mutation is already durable there (store.go:1560-1564 comment).

**Call site:** `internal/resolver/resolver.go:1657-1660`. Add `r.resolveCobolReferences()` right after `r.resolveLuaRequires()`, inside the same settle window:
```go
	r.resolveLuaRequires()
	// Godot `res://` binding ...
	r.resolveGodotResPaths()
```

### `internal/resolver/cobol_references_test.go` (test). 03-06

**Analog:** `internal/resolver/lua_imports_test.go`. The pattern is an in-memory `graph.New()`, a `seedFile` helper (`module_attribution_test.go:15-20`), an edge helper that `g.AddEdge`s a placeholder, `r := New(g)`, a direct call to the pass method, and assertions on `e.To`/`e.Origin`:
```go
func TestResolveLuaRequires_RobloxAmbiguousStaysExternal(t *testing.T) {
	g := graph.New()
	seedFile(g, "main.luau", "luau")
	seedFile(g, "a/Foo.luau", "luau")
	seedFile(g, "b/Foo.luau", "luau")
	e := luaRequire(g, "main.luau", "Foo", "script.Parent.Foo")

	r := New(g)
	r.resolveLuaRequires()

	assert.Equal(t, "unresolved::import::Foo", e.To, "ambiguous Roblox module stays external")
}
```
`seedFile` sets no `RepoPrefix` and no Meta. COBOL tests need their own seeding, with `RepoPrefix`, `Language: "cobol"`, and `Meta["cobol_kind"]` on program and copybook nodes, to cover two repos, duplicate copybooks, nested same-file programs, and the case where `COPY IDMS` is never bound.

---

### `internal/resolver/resolver.go` `resolveEdge` (M), `internal/resolver/cross_repo.go` (M), `internal/resolver/lsp_resolve.go` (M), `internal/indexer/incremental_resolve.go` (M)

Each change is one `case` line beside existing owned-namespace cases:

- `resolver.go:3361-3376`. Add `case strings.HasPrefix(target, "cobol::"): return oldTo, false` next to:
```go
	case strings.HasPrefix(target, "grpc::"):
		// ... Leave the edge untouched.
		return oldTo, false
	case strings.HasPrefix(target, "pyrel::"):
		...
		return oldTo, false
```
- `cross_repo.go:1107-1121`. `CrossRepoResolver.resolveEdge` currently has no owned-namespace exit (only `import::`, `*.`, and type-hierarchy kinds). Add a `cobol::` case that does `stats.Unresolved++` and returns before `default:` → `cr.resolveFunctionCall`, mirroring the `EdgeExtends` branch (lines 1112-1120).
- `lsp_resolve.go:517-522` and `incremental_resolve.go:450-453`. Append `strings.HasPrefix(target, "cobol::")` to the existing list:
```go
	case strings.HasPrefix(target, "import::"),
		strings.HasPrefix(target, "pyrel::"),
		strings.HasPrefix(target, "grpc::"):
		return ""
```
Fork-hygiene note: these are the four "minimal, justified" upstream edits. The rest of the logic lives in the additive `cobol_references.go`.

---

### `internal/indexer/source_revision.go` (indexer hook, file-I/O). 03-02 CR-01

**Analog:** `stampSourceRevision` in the same file (lines 43-77). Copy its gate-then-loop: return early unless some node carries `prov_revision_content_id`, then mutate Meta in place.
```go
func (idx *Indexer) stampSourceRevision(relPath string, src []byte, result *parser.ExtractionResult) {
	if result == nil {
		return
	}
	carries := false
	for _, n := range result.Nodes {
		if n == nil {
			continue
		}
		if _, ok := n.Meta["prov_revision_content_id"]; ok {
			carries = true
			break
		}
	}
	if !carries {
		return
	}
	...
```
**Disk-compare precedent** (lines 170-171):
```go
	if disk, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath))); err != nil || !bytes.Equal(disk, src) {
		return "", "indexed_bytes_differ_from_file"
	}
```
**Hook call site:** `internal/indexer/indexer.go:1721`, `idx.stampSourceRevision(relPath, src, result)` inside `applyCoverageDomains` (line 1664). Call the new sibling from the same place, or from inside `stampSourceRevision`. The BOM shift must walk `result.Edges` too, not just nodes. Use the PURPOSE/RATIONALE/KEYWORDS doc-comment style shown at lines 21-42.

### `internal/indexer/source_revision_test.go` (test). 03-02

**Analog:** the same file. Use `sourceRevisionResult` (20-28) to build a minimal COBOL-shaped `ExtractionResult` (file and program nodes with `prov_revision_content_id`; add an edge with `prov_start_byte`). Use `writeRepoFile` (40-45), `stampAt`/`stampBytes` (58-75), and `requireGit` (32-37). Research says to also run one BOM-prefixed invented program through the real indexer, because the extractor alone never sees the BOM. The harness for that is `indexCobolTracerRepo` in `cmd/gortex/cobol_tracer_test.go`.

---

### `cmd/gortex/cobol_tracer_test.go` (acceptance test). 03-01, 03-06, 03-09, D-16 re-pin

**Analog:** itself.
- Fixture constant: lines 36-40. Extend it with a second paragraph and a `PERFORM`. Stay invented.
- `indexCobolTracerRepo`: lines 96-152 (git-init repo, SQLite store, recorder, MultiIndexer `TrackRepoCtx`, prefixed IDs from `res.RepoPrefix`). Add `paragraphID` fields following `programID: res.RepoPrefix + "/cobol/demopgm.cbl::DEMOPGM"`.
- MCP tool call pattern: lines 374-392 (`startCobolTracerDaemon`, `callCobolTracerTool(t, socket, repo.root, "get_symbol", map[string]any{...,"format":"json"})`, json decode). `get_callers` on a paragraph (D-02) copies this call shape with tool `get_callers`.
- **Pinned literals that must change:** line 174 (`prov_preprocessor_module` equals `...preprocessor@v0.0.0-20260930215433-f9eaf99c34a9`, which changes in the D-16 re-pin) and line 176 (`gortex-cobol-grammar/2` → `/3`, in 03-01).

---

### `go.mod` / `go.sum` / `cobolApprovedPreprocessorModule` (config). D-16 re-pin

**Analog:** `02-02-PLAN.md` Task 1, step 1 (line 196) and `02-02-SUMMARY.md` §"Pinned Pseudo-Version" (lines 117-126).
- Module-fetch environment, copied verbatim: `GOWORK=off`, `GOPRIVATE=github.com/MuiGoku123432/tree-sitter-cobol-upgrade`, and the per-command `GIT_CONFIG_COUNT=1` / `GIT_CONFIG_KEY_0=url.git@github-personal:MuiGoku123432/tree-sitter-cobol-upgrade.insteadOf` / `GIT_CONFIG_VALUE_0=https://github.com/MuiGoku123432/tree-sitter-cobol-upgrade`. Never write global git config.
- Resolve the PV with `go list -m -json github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor@<H>`.
- Current pin locations, all to update together: `go.mod:6` (require), `go.sum:11-12`, `internal/parser/languages/cobol_grammar.go:39`, and `cmd/gortex/cobol_tracer_test.go:174`. `cobol_grammar_test.go:484,556,571,590` reference the constant, not the literal, so they follow automatically.
- **Differences from P2:** D-16 re-pins **only the preprocessor**. Leave the forest-shim `replace` at `go.mod:375` and `go.sum:9-10` on `v0.0.0-20260930215433-f9eaf99c34a9`. The fork's `validateShim` (`preprocessor/pipeline.go:478-503`) only checks that the forest module is present and that the go-tree-sitter version matches. It never compares the forest version to the preprocessor version, so mixed pseudo-versions are valid. **Drop** the 02-02 verify assertion that the two pseudo-versions are equal (`02-02-PLAN.md:214`). `cobolApprovedGrammarID` (`cobol_grammar.go:32`) stays unchanged because the grammar is unchanged.
- Carry over the P2 deviation: `go mod tidy` promotes `neo4j-go-driver` to a direct require. Revert that hunk (see `deferred-items.md`).

---

### FORK `preprocessor/analyze.go` (producer API). D-16 fork commit

**Analog:** the P2 commit `f9eaf99` diff to this file. It is additive: a new exported entry point with a doc comment that mirrors an existing one, and no change to existing signatures.

**Existing `Analyze`** (lines 101-123). `AnalyzeCopybook(ctx, sourceID, original, ...)` copies this body and swaps `a.analyze` for a copybook-mode transform plus `handoff.Build` with a copybook policy:
```go
func (a Analyzer) Analyze(ctx context.Context, sourceID SourceID, original []byte) (handoff.Handoff, error) {
	if a.tool == (ToolIdentity{}) {
		return handoff.Handoff{}, errors.New("analyzer has no attested tool identity")
	}

	an, err := a.analyze(ctx, original)
	if err != nil {
		return handoff.Handoff{}, err
	}

	return handoff.Build(handoff.Input{
		SourceID:             sourceID,
		Original:             original,
		Result:               an.result,
		Tool:                 a.tool,
		ParseConfigurationID: a.budget.ConfigurationID(),
		Facts:                an.facts,
		Policy:               grade.PolicyV1,
	})
}
```
**The private `analyze`** (lines 79-99) calls `transform.Run(original, a.catalog)`, then `tsadapter.ParseBudgeted`, then `parsefacts.Extract(raw, result.Map, original, result.Generated)`. The copybook variant differs only in the transform call. To avoid forking the body, add a transform-function parameter, or pass in a `transform.Result` produced by the caller. Either keeps the logic in one place.

**Doc-comment style for the new export** (from f9eaf99, lines 47-50):
```go
// NewEmbeddedAnalyzer builds an Analyzer attested against the grammar
// identity compiled into this module instead of a fork checkout on disk. It
// fails when the linked forest COBOL grammar is not the grammar this module
// was generated against (for example the stock go-sitter-forest grammar).
```
**Identity invariant this gives you for free.** `handoff.Build` (`handoff/handoff.go:108`) computes `OriginalContentID` from `in.Original`, and `Validate` (lines 148-156) requires the Map and Ledger original IDs to match it. As long as the wrapper is *inserted* (`KindInsert`) and `Original` is the member bytes, the D-16 requirement that `OriginalContentID` stays the member's own holds automatically.

### FORK `preprocessor/transform/pipeline.go` (transform). Copybook wrapper inserts

**Analog A, insert edit:** `transform/endevor.go:218-220`. A rule emits `edit{kind: sourcemap.KindInsert, start: X, end: X, text: ..., reason: ...}`:
```go
			if !form.hasPeriod {
				edits = append(edits, edit{kind: sourcemap.KindInsert, start: absorbEnd, end: absorbEnd, text: []byte("."), reason: reasonIncPeriodInsert})
			}
			occurrences = append(occurrences, occurrence{rule: RuleEndevorInc, edits: edits})
```
**Analog B, how `Run` applies inserts** (`pipeline.go:144-148`) and the Builder constraints (`sourcemap/builder.go:85-107`). The insert anchor must equal the current cursor, `text` must be non-empty, and the rule ID must be 1-64 printable ASCII characters with no spaces (`sourcemap.go:145-155`). A leading wrapper at offset 0 and a trailing wrapper at `len(src)` both satisfy these constraints.

**Rule constants and configuration identity** (`pipeline.go:16-31`):
```go
const (
	RuleFixedFormat    = "fixed-format/v1"
	RuleEndevorInc     = "endevor-inc/v1"
	RuleIDMSStructural = "idms-structural/v1"
	ConfigVersion      = "cobol-fixed-transform-v6"
)

func ConfigurationID(catalog Catalog) string {
	return contract.HashFields(contract.ConfigDomain, ConfigVersion, LedgerSchemaVersion, RuleFixedFormat, RuleEndevorInc, RuleIDMSStructural, catalog.ID())
}
```
**Gotcha for the planner:** adding a new rule ID such as `copybook-wrapper/v1` to `ConfigurationID`, or bumping `ConfigVersion`, changes `h.Ledger.ConfigurationID`, and therefore `prov_transform_config_id`, for **every program**, not just copybooks. Gortex tests never pin it (they only regex-check 64 hex characters), but it is a visible provenance change. Decide explicitly whether the copybook rule enters the program configuration ID or only a copybook-mode ID. `Run(original, catalog)` (line 66) has no options today, so copybook mode needs a sibling entry point, for example `RunCopybook(original, catalog, variant)`, that shares the occurrence/Builder loop rather than forking it. The wrapper text itself must be valid 7-column fixed format, because the zone check rejects disagreeing lines.

### FORK `preprocessor/grade/grade.go` (policy). Copybook-aware anchor rule

**Analog:** `PolicyV1` and `PolicyFor` (lines 66-80) plus the required-anchor families (lines 109-113, 223-237):
```go
var PolicyV1 = Policy{
	Version:                "cobol-grade-policy-v1",
	CollapseNumerator:      1,
	CollapseDenominator:    2,
	DivisionScaleAncestors: []string{"start", "program_definition"},
}

func PolicyFor(version string) (Policy, bool) {
	if version == PolicyV1.Version {
		return PolicyV1, true
	}
	return Policy{}, false
}
...
	requiredAlwaysAnchors     = []string{transform.AnchorIdentificationDivision, transform.AnchorProgramID}
	requiredIfExpectedAnchors = []string{transform.AnchorDataDivision, transform.AnchorProcedureDivision}
```
**Gotchas:**
1. `handoff.Validate` re-derives the policy with `grade.PolicyFor(h.Assessment.PolicyVersion)` (`handoff/handoff.go:199-202`). A new copybook policy version **must** be registered in `PolicyFor`, or every copybook handoff fails validation as `ErrMalformed`.
2. Anchors are computed lexically from **generated** bytes (`parsefacts/extract.go:143`, `transform.ExpectedAnchors(generated)`), so the wrapper's IDENTIFICATION DIVISION and PROGRAM-ID anchors exist, but they sit on synthetic bytes. The copybook rule has to decide whether those count. The likely answer is that they are satisfied by construction, and the rule should not hide genuine body errors.

### FORK tests (`analyze_test.go`, `grade/grade_test.go`, `transform/pipeline_test.go`)

**Analog:** `TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer` and `TestNewContentIDMatchesHandoff` from f9eaf99 (`analyze_test.go:165-240`). Each one builds `transform.NewCatalog(...)` and `NewEmbeddedAnalyzer`, calls the entry point on an invented fixture (`tracerProgram()`, lines 20-33, is the fixture style), and asserts `h.Assessment.Grade`, `h.ToolID == h.Tool.ID()`, and `NewContentID(original) == h.OriginalContentID`. The copybook test asserts the member's own content ID, `data_description` observations present, wrapper projections `Synthetic`, and a `REPLACING ==:TAG:==` member degrading with explicit gaps rather than erroring. The fork's tests use stdlib `t.Fatalf`, not testify. Match that.

### Planning analog: fork commit, then push checkpoint, then re-pin

**Analog:** `02-01-PLAN.md`:
- line 100: a precondition that pins the fork HEAD, branch, and clean tree;
- line 120: stage exact paths only (the fork has untracked `.gsd/`, `.planning/milestone.lock`, and `.planning/research/.cache/`), commit, record the hash, **do not push in this task**;
- lines 138-158: `<task type="checkpoint:human-action" gate="blocking">`, which is the push (fast-forward only, never `--force`). Its automated verification is `git ls-remote origin refs/heads/gsd/v0.26.0-estate-parse-recovery` equal to HEAD, with HEAD's parent the recorded base, and `<resume-signal>` "pushed" / "push it".

For D-16, the base parent is `f9eaf99c34a92db6f5487d316cf9977295a62a7d` (the current fork HEAD). Then the Gortex re-pin plan (02-02 analog above), then copybook data-item extraction.

---

## Shared Patterns

### Provenance Meta contract (every new node and every relationship edge)
**Source:** `internal/parser/languages/cobol_grammar.go:204-222` (`docMeta`) plus lines 233-236.
**Apply to:** all walker files.
```go
		meta := maps.Clone(docMeta)
		meta["cobol_kind"] = "program"
		meta["prov_observation_kind"] = o.Kind
		meta["prov_affected"] = o.Affected
```
Value types are limited to string, bool, int, float64, and []string, so the SQLite flat codec round-trips them (02-02 key contract). Structural `defines`/`member_of` edges carry **no** Meta (D-17).

### Structural edge literal
**Source:** `cobol_grammar.go:244-248`.
**Apply to:** every `defines` and `member_of` edge.
`Origin: graph.OriginASTResolved, Confidence: 1.0, ConfidenceLabel: "EXTRACTED"`.

### Repo-relative IDs only
**Source:** `cobol_grammar.go:237` (`id := filePath + "::" + p.symbol`) and `indexer.go:2300-2321` (`applyRepoPrefix`).
**Apply to:** all extractor IDs and resource IDs. Never build `<repo>/` in the extractor. Node IDs stored in Meta are not prefixed (Pitfall 5), so store the QualName (`cobol_statement_symbol`).

### Owned placeholder namespace
**Source:** the `grpc::`/`pyrel::` precedent in `resolver.go:3362-3376`, `lsp_resolve.go:517-522`, and `incremental_resolve.go:450-453`.
**Apply to:** every COBOL placeholder: `unresolved::cobol::{program,copybook,dynamic,label,data,idms_scope}::...`.

### Neo4j-safe meta key names
**Source:** `internal/neo4jprojection/properties.go:171-180` (`sensitiveMetadataKey`). It lowercases the key and strips `-`, `_`, `.`, and spaces before substring-matching `password|passwd|secret|token|credential|privatekey|apikey|authorization`.
**Apply to:** every new `cobol_*`, `sql_*`, `idms_*`, and `cics_*` key. Because underscores are stripped, check the joined form too.

### Error wrapping, no source text
**Source:** `cobol_grammar.go:196-201` (`fmt.Errorf("enhanced COBOL analysis of %s: %w", filePath, err)`) and the fork's `errTransformFailed` (`transform/pipeline.go:59`, "message that carries no source bytes or offsets").
**Apply to:** the extractor, resolver, and the fork copybook entry point.

### Invented fixtures only
**Source:** `cobol_grammar_test.go:23-24` and `cobol_tracer_test.go:34-35` comments, plus fork `analyze_test.go:17-19`.
**Apply to:** every new test in both repos.

## No Analog Found

| File / Concern | Role | Data Flow | Reason |
|---|---|---|---|
| Edge dedupe with meta merge (`From,To,Kind,Line` → `cobol_statement_symbols`, `cobol_occurrence_count`) | utility | transform | No extractor merges duplicate-edge meta. `go_sql.go` only drops duplicates. Use RESEARCH §Edge identity. |
| Percent-encoding of literal-derived ID segments (Pitfall 9) | utility | transform | No existing ID-segment encoder in `internal/parser/languages`. Implement it per RESEARCH Pitfall 9 and keep the raw value in meta. |
| Two-projection extent stamping (Pitfall 1) | utility | transform | `cobolStampRange` takes one projection. Extend it per RESEARCH Pitfall 1. |

## Metadata

**Analog search scope:** `internal/parser/languages/` (cobol*, go_sql, go_dataflow, java, kubernetes, dbt), `internal/resolver/` (lua_imports, resolver, cross_repo, lsp_resolve, terminal, language_gate, module_attribution_test), `internal/indexer/` (indexer, source_revision, incremental_resolve), `internal/graph/` (store, graph, resolver_projection, stub), `internal/neo4jprojection/properties.go`, `cmd/gortex/cobol_tracer_test.go`, fork `preprocessor/{analyze,pipeline}.go`, `transform/{pipeline,endevor}.go`, `sourcemap/{builder,sourcemap}.go`, `grade/grade.go`, `handoff/handoff.go`, and `.planning/phases/02-*/02-0{1,2}-{PLAN,SUMMARY}.md`.
**Files scanned:** about 35.
**Pattern extraction date:** 2026-10-01.
