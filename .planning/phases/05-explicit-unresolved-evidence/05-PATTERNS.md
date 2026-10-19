# Phase 5: Explicit Unresolved Evidence - Pattern Map

**Mapped:** 2026-10-19
**Files analyzed:** 38 (8 new source files, 10 new test files, 20 modified source/test files)
**Analogs found:** 36 / 38. Two pieces have no in-repo analog: the ledger pairing algorithm, and the fixed-format evidence normalization. See "No Analog Found".

All paths are relative to `/Users/e1001547-mbp-it/repos/mine/GoApps/gortex` and were confirmed with `git ls-files`. None is a mirror path from `gortex-phase4-sql`. Line numbers were read on 2026-10-19.

Post-research amendments D-17, D-18 and D-19 from CONTEXT.md apply throughout:
- No producer `external-reference` gap becomes a finding.
- There is one tail finding per token.
- No finding or placeholder is created for `sql_include_builtin=true`.

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/graph/node.go` (MOD: `KindFinding`) | model (schema enum) | -- | `KindContractBridge` + `validNodeKinds` (`node.go:265-308`) | exact |
| `internal/graph/finding.go` (NEW) | utility (ID, digest, Meta keys) | transform | `neo4jprojection/identity.go:53-62` (framing) + `indexer/cobol_material.go:36-43` (BLAKE3) + `cobol_grammar_identity.go:115-119,152-154` (ordinal, rule ID) | composite |
| `internal/graph/finding_ledger.go` (NEW) | model + pure diff + capability ifaces | transform | `internal/graph/file_index_failures.go:1-60` (struct + Reader/Writer ifaces + `var _`) | role-match (diff has no analog) |
| `internal/graph/finding_node_evict.go` (NEW) | store capability (in-memory) | CRUD (delete) | `internal/graph/config_node_evict.go:1-57` + existing `evictSyntheticNodesByIDs` (`derived_contract_replace.go:103`) + dispatcher `contract_node_evict.go:11-24` | exact |
| `internal/graph/store_sqlite/finding_node_evict.go` (NEW) | store (SQLite) | CRUD (delete, tx) | `internal/graph/store_sqlite/config_node_evict.go:16-103` | exact (3rd copy, see rule-of-three flag) |
| `internal/graph/store_sqlite/store_finding_history.go` (NEW) | store sidecar | CRUD (append/read) | `internal/graph/store_sqlite/store_file_index_failures.go:1-76` | exact |
| `internal/graph/store_sqlite/schema_version.go` (MOD: v23) | migration | batch | itself `:37`, `:120`, `:125` | exact |
| `internal/graph/store_sqlite/schema.go` (MOD: registry) | schema registry | -- | `viewGenSidecars` entry `:1242-1247` | exact |
| `internal/graph/store_sqlite/store_purge.go` (MOD: table lists) | store admin | batch | `purgeSidecarTables :43-65`, `orphanScanTables :152-163`, `rekeyDropTables :250-265` | exact |
| `internal/parser/languages/cobol_grammar_findings.go` (NEW) | extractor | transform | `cobol_grammar_walk.go` builders (`symbolNode :165-180`, `memberOf :187-197`, `cobolStampMeta :263-287`) + `cobol_grammar_copybook.go:5-14` imports | exact |
| `internal/parser/languages/cobol_grammar.go` (MOD: call site, `/6`, WR-06, IN-08 stamp) | extractor | transform | itself `:40-46`, `:213-218`, `:255-298` | exact |
| `internal/parser/languages/cobol_grammar_copybook.go` (MOD: call site) | extractor | transform | itself `:65-85` | exact |
| `internal/parser/languages/cobol_grammar_sql.go` (MOD: WR-07) | extractor | transform | itself `:149-170` | exact |
| `internal/resolver/cobol_findings.go` (NEW) | service (resolver pass) | batch | `internal/resolver/cobol_resources.go:13-128` + `external_calls.go:403-435` | exact |
| `internal/resolver/cobol_references.go` (MOD: placeholder `To`, in-edge visit, IDMS, IN-08) | service (resolver pass) | batch | itself `:124-289`, `:317-400` | exact |
| `internal/indexer/finding_ledger.go` (NEW) | indexer orchestration | batch / event-driven tail | `internal/indexer/cobol_reconcile.go:49-82` + `indexer/file_index_failures.go:239-245` + `incremental_batch.go:1029-1057` | composite |
| `internal/indexer/cobol_reconcile.go`, `multi.go`, `incremental_watcher_batch.go`, `repository_untrack.go` (MOD: ledger hook points) | indexer tails | event-driven | `multi.go:1614-1616`, `:2915-2926`, `:3441-3450`; `incremental_watcher_batch.go:425-439`; `repository_untrack.go:243-249` | exact |
| `internal/mcp/tools_enhancements.go` (MOD: finding counts) | controller (`index_health`) | request-response | itself `:3206-3237` (scan loop) + `:3483-3510` (`repo_ownership` block) | exact |
| `internal/mcp/resources.go` (MOD, optional: schema text) | resource text | -- | `handleResourceSchema :195-216` (`contract_bridge` entry) | exact |
| `internal/graph/finding_test.go` (NEW) | test (unit) | -- | `internal/graph/scope_test.go:1-42` | role-match |
| `internal/graph/finding_ledger_test.go` (NEW) | test (unit) | -- | `internal/graph/scope_test.go` | role-match |
| `internal/graph/store_sqlite/store_finding_history_test.go` (NEW) | test (store) | -- | `store_file_index_failures_test.go:12-150` | exact |
| `internal/graph/store_sqlite/finding_node_evict_test.go` (NEW) + `store_generation_read_test.go` / `store_generation_write_test.go` (MOD) | test (store) | -- | `repo_contract_projections_test.go:64-86`; checklist `:1711`; write fence `:672-691` | exact |
| `internal/graph/store_sqlite/schema_sidecar_view_gen_test.go` (MOD) | test (migration) | -- | itself `:285-290`, `:425-431` | exact |
| `internal/parser/languages/cobol_grammar_findings_test.go` (NEW) | test (real producer) | -- | `cobol_grammar_test.go:95-158` helpers + `cobol_grammar_contract_test.go:22-113` | exact |
| `internal/resolver/cobol_findings_test.go` (NEW) | test (unit) | -- | `cobol_resources_test.go:13-67` + `cobol_references_test.go:15-58` | exact |
| `internal/indexer/cobol_findings_lifecycle_test.go` (NEW) | test (real path) | batch | `cobol_lifecycle_test.go:92-192` + `cobol_lifecycle_incremental_test.go:64-154,366-398` | exact |
| `internal/indexer/cobol_cascade_gate_test.go` (NEW) | test (real path gate) | batch | `parser/forest/cobolprobe/cascade_test.go:52-244` (cases) + `newCobolLifecycleEnv` (harness) | composite |
| `internal/mcp/index_health_findings_test.go` (NEW) | test (MCP) | request-response | `internal/mcp/index_health_ownership_test.go:19-68` | exact |
| `cmd/gortex/cobol_findings_test.go` (NEW) | test (integration MCP + CLI) | request-response | `cmd/gortex/cobol_tracer_test.go:115-182,302-476` | exact |
| `internal/neo4jprojection/properties_test.go` (MOD: finding case) | test (unit) | transform | `TestProjectNode_CobolBreadthMeta :321-397` | exact |
| Existing tests pinning `missing` on `unresolved::cobol::...`, `/5`, and node counts (MOD) | test | -- | themselves (see "Tests That Will Change") | exact |

---

## Pattern Assignments

### `internal/graph/node.go` (MOD: add `KindFinding`, D-05)

**Analog:** the `KindContractBridge` declaration and its `validNodeKinds` entry.

**Declaration shape** (`node.go:265-282`): a long doc comment that states the ID convention, the Meta it carries, and the edges that link it.
```go
	// KindContractBridge represents one matched provider↔consumer
	// contract group — ...
	// ID convention:
	// `bridge::<contract-id>` where contract-id is the canonical
	// contract key ...
	KindContractBridge NodeKind = "contract_bridge"
)
```

**Registration** (`node.go:292-308`). This must be added, or `IsValidNodeKind("finding")` is false and `search_symbols kind:finding` fails the check at `mcp/field_query.go:413`.
```go
var validNodeKinds = map[NodeKind]bool{
	...
	KindMacro:     true, KindAgent: true, KindContractBridge: true,
}
```

**Parity fence:** `internal/graph/parity_audit_test.go:41-91` fails when a kind is not referenced outside its declaring file. The extractor and the resolver both reference `KindFinding`, so declare it in `node.go` with every other kind. Do not declare it in `finding.go`: the existing convention keeps one kind block.

**Do NOT** add `KindFinding` to `OwnershipExemptKinds` (`prefix_diagnostics.go:116-132`). Findings are repository-owned and prefixed, so the SEC-08 audit must see them.

---

### `internal/graph/finding.go` (NEW: utility, transform; D-01, D-02, D-12)

**Analogs:** three sources, combined. Keep the file language-neutral: `internal/graph` must not import the preprocessor.

**1. Length-delimited digest framing.** Copy the alias-safe framing from `neo4jprojection/identity.go:53-62` and swap SHA-256 for BLAKE3:
```go
func projectionDigest(prefix string, values ...string) string {
	h := sha256.New()
	var size [8]byte
	for _, value := range values {
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = h.Write(size[:])
		_, _ = h.Write([]byte(value))
	}
	return prefix + "-" + hex.EncodeToString(h.Sum(nil)[:16])
}
```

**2. BLAKE3 usage idiom** (`indexer/cobol_material.go:8`, `:39`). The import is `"github.com/zeebo/blake3"`. Use `blake3.New()` with the framing above, or `blake3.Sum256`, then truncate to 16 bytes (RESEARCH Pattern 1). `internal/graph` does not import blake3 yet; the module is already in `go.mod:280`, so this adds no new dependency.

**3. Versioned rule-ID constant** (`cobol_grammar_identity.go:152-154`):
```go
// cobolNameNormRule identifies the versioned label and name matching rule
// cobolNameNorm implements (D-18). Record it wherever a match depends on it.
const cobolNameNormRule = "cobol-name-norm/1"
```
Mirror it for `gortex-finding-id/1`, `gortex-finding-evidence/1` and `gortex-finding-severity/1`.

**`#k` ordinal convention** (`cobol_grammar_identity.go:115-119`): the first occurrence carries no suffix, and `#k` is added only for k >= 2.
```go
			occurrences[base]++
			symbol := base
			if k := occurrences[base]; k >= 2 {
				symbol += "#" + strconv.Itoa(k)
			}
```

**Meta key constant block style** (`resolver/cobol_references.go:17-42`): one grouped `const (...)`, with a comment above each group that names who writes and who reads the keys. Use it for the `finding_*` keys listed in RESEARCH Pattern 3.

**`@finding` root rationale** (`cobol_grammar_copybook.go:16-21`): `'@' cannot begin a COBOL word, so no written name collides with it.` Reuse this wording.

**Doc style** (`internal/graph/scope.go:9-22`): `PURPOSE — / RATIONALE — / KEYWORDS —` on the exported type or function.

**Discretion flag for the planner:** RESEARCH places "evidence normalize" in `graph/finding.go`. The column 1-6 and 73+ stripping rule is a COBOL fixed-format rule, and the extractor is its only caller (external findings digest edge Meta, not source columns). Two placements fit the convention:
- `FindingID` takes an already-normalized evidence `[]byte` or digest, and the normalization stays private in `cobol_grammar_findings.go`. This is the single-use-stays-local rule.
- The normalization lives in `graph`, as RESEARCH suggests.

Pick one explicitly.

---

### `internal/graph/finding_ledger.go` (NEW: model + pure diff + capability interfaces; D-03, D-06)

**Analog for the types and capability interfaces:** `internal/graph/file_index_failures.go:1-25`.
```go
// FileIndexFailure records a file that could not be indexed, ...
type FileIndexFailure struct {
	Path             string `json:"path"`
	Error            string `json:"error"`
	PermissionDenied bool   `json:"permission_denied,omitempty"`
	RepoPrefix       string `json:"repo_prefix"`
	WorkspaceID      string `json:"workspace_id,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
}

// FileIndexFailureReader is an optional, generation-scoped health capability.
type FileIndexFailureReader interface {
	FileIndexFailuresForRepo(repoPrefix string) ([]FileIndexFailure, error)
}

// FileIndexFailureWriter replaces one repository's complete failure snapshot.
type FileIndexFailureWriter interface {
	ReplaceFileIndexFailures(repoPrefix string, failures []FileIndexFailure) error
}
```
Model a `FindingTransition` (or similarly named) struct with these fields:
- `FindingID`, `PriorStatus`, `NewStatus`, `SuccessorID`;
- the evidence and parser stamps;
- the `RepoPrefix`, `WorkspaceID` and `ProjectID` scope columns.

Add two capabilities: a reader for the open set and an appender.

**In-memory backend (optional):** the precedent implements both capabilities on `*Graph` with a mutex and a map field (`graph.go:553-554`, `file_index_failures.go:27-60`). Adding fields to `Graph` is an invasive upstream edit. Prefer SQLite-only, with the indexer's capability probe a no-op on in-memory stores, unless a pure-unit ledger test needs it. The pure diff function can be tested without any store.

**Diff algorithm:** no analog. Use RESEARCH Pattern 6. Pair within one (container, class, kind) in `start_byte`-then-ID order. Never pair across containers or classes. An identical rerun must produce zero transitions.

---

### `internal/graph/finding_node_evict.go` (NEW: in-memory capability)

**Analog for the interface and dispatcher:** `internal/graph/contract_node_evict.go:3-24`.
```go
type ContractNodeBatchEvicter interface {
	EvictContractNodesByIDs(ids []string) (nodesRemoved, edgesRemoved int)
}

// EvictContractNodesByIDs dispatches only to the set-oriented capability.
// Production Graph and SQLite stores implement it; adapter stores report
// unsupported instead of hiding an N+1 point-delete fallback.
func EvictContractNodesByIDs(store Store, ids []string) (nodesRemoved, edgesRemoved int, supported bool) {
	if store == nil || len(ids) == 0 {
		return 0, 0, true
	}
	evicter, ok := store.(ContractNodeBatchEvicter)
	if !ok {
		return 0, 0, false
	}
	nodesRemoved, edgesRemoved = evicter.EvictContractNodesByIDs(ids)
	return nodesRemoved, edgesRemoved, true
}
```

**In-memory body: reuse, do not copy.** `config_node_evict.go:12-55` and `contract_node_evict.go:28+` each inline the shard-lock loop. A kind-parameterized private helper already exists in `derived_contract_replace.go:103`:
```go
func (g *Graph) evictSyntheticNodesByIDs(ids []string, kinds map[NodeKind]struct{}) (nodesRemoved, edgesRemoved int) {
```
It is already called with one kind at `derived_contract_replace.go:57-60`. The `*Graph` method should therefore be one line:
```go
return g.evictSyntheticNodesByIDs(ids, map[NodeKind]struct{}{KindFinding: {}})
```
End the file with `var _ FindingNodeBatchEvicter = (*Graph)(nil)` (as at `config_node_evict.go:57`).

---

### `internal/graph/store_sqlite/finding_node_evict.go` (NEW: SQLite tx delete)

**Analog:** `internal/graph/store_sqlite/config_node_evict.go:16-103`. Copy the whole transaction shape, with the kind bound as an argument:
```go
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if !s.invalidateAnalysisBeforeMutationLocked() {
		return 0, 0
	}
	tx, err := s.beginWrite()
	if err != nil {
		panicOnFatal(err)
		return 0, 0
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for start := 0; start < len(unique); start += contractNodeEvictChunkSize {
		...
		args = append(args, string(graph.KindConfigKey))       // -> graph.KindFinding
		...
		configIDs := `SELECT id FROM nodes WHERE kind = ? AND id IN (` + placeholders + `) AND view_gen = ?`
		for _, column := range []string{"from_id", "to_id"} {
			result, execErr := tx.Exec(`DELETE FROM edges WHERE `+column+` IN (`+configIDs+`) AND view_gen = ?`, edgeArgs...)
		...
	}
	...
	committed = true
	changed := nodesRemoved > 0 || edgesRemoved > 0
	s.finishAnalysisMutationLocked(changed)
	if changed {
		s.markMutationReceiptsIncompleteLocked()
	}
```
Reuse `contractNodeEvictChunkSize` (`contract_node_evict.go:9-11`, which is 800 IDs plus bindings).

**Rule-of-three flag (planner decision):** this would be the third near-identical SQLite body. The other two are `contract_node_evict.go:21-105` and `config_node_evict.go:16-101`. The CLAUDE.md rule of three says to extract a private `evictNodesOfKindByIDs(kind, ids)` in `store_sqlite` and make all three delegate. That edits two upstream files, against fork hygiene. Choose one option and record the reason:
- extract the helper;
- copy, with a `// note: third occurrence` comment.

**Registration in the generation fences (required, or registry tests fail):**
- `store_generation_read_test.go:1711`. Add a sibling line:
  ```go
  {iface: (*graph.ConfigNodeBatchEvicter)(nil), skip: skipWrite, writeFence: writerFamilyFence("config_node_evict")},
  ```
- `store_generation_write_test.go:672-691`. Add a `finding_node_evict` case that copies the `config_node_evict` seed/disturb pair, with a `KindFinding` node and one incident edge.

**Kind-guard test analog:** `store_sqlite/repo_contract_projections_test.go:64-86` (`TestSQLiteEvictConfigNodesByIDsIsKindGuardedAndSetOriented`). It seeds a non-target kind under the same ID list, passes a duplicate ID, and asserts `(1,1)`.

---

### `internal/graph/store_sqlite/store_finding_history.go` (NEW: sidecar table; D-06, D-07)

**Analog:** `internal/graph/store_sqlite/store_file_index_failures.go:1-76`.

**Table body plus create function** (`:9-23`):
```go
const fileIndexFailuresTableBody = ` (
    view_gen          INTEGER NOT NULL DEFAULT 0,
    repo_prefix       TEXT NOT NULL,
    file_path         TEXT NOT NULL,
    error             TEXT NOT NULL,
    permission_denied INTEGER NOT NULL DEFAULT 0,
    workspace_id      TEXT NOT NULL DEFAULT '',
    project_id        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (view_gen, repo_prefix, file_path)
) WITHOUT ROWID`

func createFileIndexFailuresTable(tx *sql.Tx) error {
	_, err := tx.Exec("CREATE TABLE IF NOT EXISTS file_index_failures" + fileIndexFailuresTableBody)
	return err
}
```
For a ledger of transitions:
- `view_gen` must lead the primary key (the `viewGenSidecars` contract, `schema.go:1235-1241`).
- The key also needs a per-row discriminator, for example `(view_gen, repo_prefix, finding_id, seq)`, because one finding can have several rows.

**Read pattern** (`:25-41`): `s.db.Query(... WHERE view_gen = ? AND repo_prefix = ? ORDER BY ...)`, a typed scan, and return `rows.Err()`.

**Write pattern** (`:43-71`): `s.writeMu.Lock()`, then `s.beginWrite()`, `defer tx.Rollback() //nolint:errcheck`, chunked multi-row `INSERT` bounded by `fileMetaChunk` (comment: "N parameters per row"), and `tx.Commit()`. An append-only ledger drops the leading `DELETE`. Writers return `error`; the `panicOnFatal` style is only for graph-core evicters.

**Compile-time assertions** (`:73-76`):
```go
var (
	_ graph.FileIndexFailureReader = (*Store)(nil)
	_ graph.FileIndexFailureWriter = (*Store)(nil)
)
```

**Registries to update. Every one is enforced by an existing test.**

| Registry | Location | Action | Why |
|---|---|---|---|
| `viewGenSidecars` | `schema.go:1242-1247` | add `{table: "finding_history", body: ..., columns: ...}` | feeds the fresh-store DDL (`sidecarSchemaSQL`, `schema.go:1474-1488`) and the generation sweep (`payload_generation.go:583-592`) |
| `currentSchemaVersion` | `schema_version.go:37` | `22` -> `23` | `schema_version.go:932` refuses a bump without a migration |
| `schemaMigrations` | `schema_version.go:120,125` | append `{version: 23, name: "persist finding history", inPlace: createFindingHistoryTable}` | v21 precedent; idempotent `IF NOT EXISTS` |
| `purgeSidecarTables` | `store_purge.go:43-65` | add | untrack (`PurgeRepo`) removes history (RESEARCH: "purged with the repository") |
| `orphanScanTables` | `store_purge.go:152-163` | add | history rows can outlive every node of a repository |
| `rekeyDropTables` | `store_purge.go:250-265` | add | finding IDs embed the repository prefix |
| `EvictRepo` sidecar sweep | `file_batch_evict.go:113-119` | **do NOT add** | that path clears `file_index_failures` on reindex; history must survive reindex (D-07) |
| capability checklist | `store_generation_read_test.go:1745-1746` | add two `skip:` lines like the v21 ones | the checklist fails on an unlisted interface |

**Migration test special cases** (`schema_sidecar_view_gen_test.go:285-290`, `:425-431`) currently single out `file_index_failures`:
```go
		if sidecar.table == "file_index_failures" {
			// This sidecar was introduced at v21; v14 had no table or rows.
			execDDL(t, db, `DROP TABLE file_index_failures`)
			continue
		}
```
Extend both checks to `finding_history` ("introduced at v23"). A small set or a `switch` is enough; do not build a registry for two entries.

---

### `internal/parser/languages/cobol_grammar_findings.go` (NEW: extractor, transform; GAP-01, D-13, D-18)

**Analog:** the `cobolWalk` builders in `cobol_grammar_walk.go`.

**File header and imports** (`cobol_grammar_copybook.go:1-14` already imports both `grade` and `handoff`):
```go
//go:build darwin || linux

package languages

import (
	"maps"
	...
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/grade"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"

	"github.com/zzet/gortex/internal/graph"
)
```

**Node construction** (copy `symbolNode`, `cobol_grammar_walk.go:165-180`). Clone `w.docMeta`, set Meta, and append to `w.result.Nodes`:
```go
	meta := maps.Clone(w.docMeta)
	meta["cobol_kind"] = cobolKind
	...
	node := &graph.Node{
		ID: w.filePath + "::" + symbol, Kind: kind, Name: name, QualName: symbol,
		FilePath: w.filePath, Language: "cobol", Meta: meta,
	}
	w.result.Nodes = append(w.result.Nodes, node)
```
A finding uses `graph.KindFinding`, and its ID is `w.filePath + "::@finding/parse/<hex>[#k]"`. **Do not set `cobol_kind`** to `program` or `copybook`, and do not set `cobol_name_key`. RESEARCH Anti-Patterns: the candidate index (`cobol_references.go:486-504`) keys on exactly those.

**Container edge** (`cobol_grammar_walk.go:187-197`). Structural, with no Meta (D-17):
```go
func (w *cobolWalk) memberOf(child, parent string, line int) {
	w.structural(child, parent, graph.EdgeMemberOf, line)
}
```

**Range or absence stamping** (`cobol_grammar_walk.go:263-287`). Pass `gap.Original`. When the producer set `AbsenceReason`, overwrite `prov_range_absence` with it verbatim:
```go
func cobolStampMeta(meta map[string]any, r *sourcemap.Projection) int {
	switch {
	case r == nil || r.NoImage:
		meta["prov_range_exact"] = false
		meta["prov_range_absence"] = "no_original_projection"
		return 0
	case r.Synthetic:
		meta["prov_range_exact"] = false
		meta["prov_range_absence"] = "synthetic_projection"
		return 0
	}
	meta["prov_start_row"] = int(r.Start.Row)
	...
	meta["prov_range_exact"] = r.Exact
	return int(r.Start.Row) + 1
}
```

**Container ladder inputs** (RESEARCH Pattern 4) come from existing walk state:

| Ladder step | Walk member | Location |
|---|---|---|
| innermost statement | `w.statementSymbol(i)` (returns a QualName, never prefixed) | `cobol_grammar_walk.go:137-154` |
| paragraph, section or procedure division | `w.container[i]` | fields `:41-47` |
| data or environment section | `w.sections[i]` | `:52-54` |
| owning program | `w.programSymbol(i)` | `:117-125` |
| QualName to node ID (copybook root becomes file node) | `w.id(symbol)` | `:127-135` |
| Parent walk | `w.parent(i)` (bounds-checked) | `:103-108` |

**Producer gap struct** (verified at `preprocessor@e64234c/handoff/gaps.go:93-107`): `ResolutionClass, Kind, Reason, Severity, Scope, Generated *sourcemap.Span, Original *sourcemap.Projection, NodeKind, ExpectedFamily, Container int, Target, AbsenceReason`. Filter `g.ResolutionClass == handoff.ResolutionParseUnresolved` only (D-17).

**Call sites (Pitfall 5: Meta mutation order).**
- Programs, `cobol_grammar.go:283-296`: insert after `cobolSQL(w, data)` (`:288`) and **before** the document flags at `:291-296`:
  ```go
	cobolSQL(w, data)
	// Set after the loop so these document-level flags stay on the file
	// node and are not cloned into program Meta.
	if containmentUnresolved {
		docMeta["prov_containment_unresolved"] = true
	}
  ```
- Copybooks, `cobol_grammar_copybook.go:68-85`. The walk already holds a cloned docMeta (`:53`). The unmapped check at `:83` counts nodes:
  ```go
	if len(result.Nodes) == 1 && len(result.Edges) == 0 && h.Assessment.Grade != grade.Green {
		docMeta["prov_analysis_absence"] = "copybook_content_unmapped"
	}
  ```
  Either call `cobolFindings(w)` after `:83-85`, or make the check count only non-finding nodes. Add a regression test for each path.

**Cascade (D-13):** read `w.h.Assessment.Reasons` (stamped today at `cobol_grammar.go:239-240`). Compare against `grade.ReasonDivisionScaleRecovery`, `grade.ReasonRetainedStructureCollapse`, `grade.ReasonMissingRequiredAnchor` and `grade.ReasonLostCapture` (the constants, not literals, per RESEARCH).

**Severity (D-12):** use `g.Severity` verbatim. Gortex-derived findings (`uncontained-cascade`, `analysis-failed`) are set by Gortex, with `finding_severity_rule`.

---

### `internal/parser/languages/cobol_grammar.go` (MOD: four changes)

**1. Extractor version** (`:40-46`). Bump `/5` to `/6` once for the phase and update the comment in the same voice:
```go
	// cobolGrammarExtractorVersion identifies this handoff-to-graph mapping.
	// Bump it whenever the emitted nodes, edges, or prov_* keys change.
	// /5 is the Phase 4 mapping: ...
	cobolGrammarExtractorVersion = "gortex-cobol-grammar/5"
```
`MaterialIdentity` (`:174-186`) folds the version into the material fingerprint, so every COBOL file re-extracts automatically. No data migration is needed (RESEARCH Runtime State Inventory).

**2. WR-06 copybook fallback** (`:213-218`). Today:
```go
	switch {
	case err != nil && copybook:
		return nil, fmt.Errorf("enhanced COBOL copybook analysis of %s: %w", filePath, err)
	case err != nil:
		return nil, fmt.Errorf("enhanced COBOL analysis of %s: %w", filePath, err)
	}
```
The copybook branch becomes an identity-only file node plus an `analysis-failed` document finding. The candidate identity keys must match `cobol_grammar_copybook.go:71-74` exactly, or the resolver's candidate index (`cobol_references.go:486-504`) will not see the node:
```go
	base := filepath.Base(filePath)
	docMeta["cobol_kind"] = "copybook"
	docMeta["cobol_name_key"] = cobolIDSegment(cobolNameNorm(strings.TrimSuffix(base, filepath.Ext(base))))
	docMeta["cobol_name_norm"] = cobolNameNormRule
```
Keep these as hard failures: init errors (`:193-195`), grammar mismatch (`:219-222`) and program analysis errors (A7). Do not persist raw error text in Meta (RESEARCH Security V7).

**3. IN-08 stamp** (`:255-265`). The per-program Meta loop already stamps `cobol_nested` and `cobol_container`, and `containmentUnresolved` is in scope from `:255`. Add the new flag to the same `meta` map, only on nested programs:
```go
	programs, containmentUnresolved, unnamed := cobolProgramSymbols(h)
	for _, p := range programs {
		...
		meta["cobol_nested"] = p.nested
		if p.nested {
			meta["cobol_container"] = filePath + "::" + p.container
		}
```

**4. Call site for `cobolFindings(w)`**: see the previous section.

---

### `internal/parser/languages/cobol_grammar_sql.go` (MOD: WR-07)

**Change site** (`:154-163`):
```go
		decls := declared[u.programID+"\x00"+cobolSQLName(u.cursor)]
		if len(decls) != 1 {
			if n != nil {
				reason := "cursor_not_declared"
				if len(decls) == 0 && w.root != "" {
					// A copybook does not declare it; an including program may.
					reason = "cursor_not_in_copybook"
				}
				n.Meta["cobol_unresolved_reason"] = reason
			}
			continue
		}
```
When `len(decls) > 1`, set the reason to `"ambiguous"`. WR-09 (`cursor_not_in_copybook`) is already fixed; keep it green (`cobol_grammar_copybook_test.go:505`).

**Test that flips:** `cobol_grammar_sql_test.go:272-282` (`cursor_unknown`) loops over `"C9", "C2"` and asserts `cursor_not_declared` for both. C2 is declared twice (`:279-281`), so its expectation becomes `ambiguous`.

---

### `internal/resolver/cobol_findings.go` (NEW: resolver pass; GAP-02, GAP-05, D-04, D-11, D-19)

**Primary analog:** `internal/resolver/cobol_resources.go:13-128` (`reconcileCobolResources`). It has the same structure: collect owned synthetic nodes by kind and language, read in-edges in one batch, evict orphans through `EvictFile` on their own virtual path, and add missing edges with one `AddBatch`.

**Doc block** (`:13-37`): copy the `PURPOSE — / RATIONALE — / KEYWORDS —` layout, including the "writes nothing when ... a second run is a no-op" sentence. Identical reruns must write nothing (ID-07).

**Collect and filter by repository set** (`:39-58`):
```go
	for _, kind := range []graph.NodeKind{graph.KindResource, graph.KindTable} {
		for n := range r.nodesByKindLang(kind, "cobol") {
			if n == nil || n.RepoPrefix == "" || n.FilePath != n.ID {
				continue
			}
			if absence, _ := n.Meta["prov_range_absence"].(string); absence != "shared_resource" {
				continue
			}
			if repos != nil {
				if _, ok := repos[n.RepoPrefix]; !ok {
					continue
				}
			}
			shared = append(shared, n)
```
For placeholders, change the discriminator to `cobol_kind == "missing_artifact"` (RESEARCH Pattern 5). Keep the `FilePath == ID` guard (ID-07: never delete a node owned by a real file).

**Orphan sweep** (`:64-79`). The test is "no in-edge other than from a finding":
```go
	inEdges := r.graph.GetInEdgesByNodeIDs(ids)
	for _, n := range shared {
		referenced := false
		for _, e := range inEdges[n.ID] {
			if e != nil && e.Kind != graph.EdgeAliases {
				referenced = true
				break
			}
		}
		if !referenced {
			r.graph.EvictFile(n.FilePath)
			continue
		}
```
Evict undesired findings **before** this sweep (RESEARCH Pattern 5).

**Batch add** (`:125-127`): `r.graph.AddBatch(nil, added)`. Use the same call with new finding and placeholder nodes plus their `member_of` and `references` edges.

**Synthetic placeholder node builder** (`external_calls.go:403-435`):
```go
func externalCallNodeID(ecosystem, importPath string) string {
	return externalCallPrefix + ecosystem + "::" + importPath
}

func newExternalCallNode(nodeID, ecosystem, importPath, callerLang string) *graph.Node {
	return &graph.Node{
		ID:       nodeID,
		Kind:     graph.KindModule,
		...
		Meta: map[string]any{
			"synthetic":     true,
			"external_call": true,
			...
		},
	}
}
```
For placeholders:
- use `graph.KindResource` and `ID == FilePath == <callerRepo>/cobol-missing::<type>/<key>`;
- set Meta `synthetic: true`, which gives the Neo4j `GortexUnresolved` label (`neo4jprojection/properties.go:54-55`).

**Scope fields.** Copy them from the caller node (`resolver.go:5756-5779`):
```go
func (r *Resolver) callerRepoPrefix(e *graph.Edge) string {
	fromNode := r.cachedGetNode(e.From)
	if fromNode != nil {
		return fromNode.RepoPrefix
	}
	return ""
}
```
Copy `RepoPrefix`, `WorkspaceID` and `ProjectID` from `r.cachedGetNode(e.From)` onto both the placeholder and the external finding. Without them, SEC-01 scope filters and the canonical dump (`graphdump`, scoped by repository and workspace) drop the nodes.

**Writer-independent Meta.** Shared synthetic nodes keep only writer-independent keys (`cobol_grammar_walk.go:223-261`, `cobolSharedResourceKeys`). Placeholders carry no parser stamps and no as-written spelling (D-11, P4 D-18).

---

### `internal/resolver/cobol_references.go` (MOD: integrate placeholders)

| Change | Site | Current code (excerpt) |
|---|---|---|
| Pitfall 8: the ledger must still run when no COBOL is left. The early return can stay because the ledger runs at the indexer level, but placeholder sweep and finding eviction for a repository whose last COBOL file was deleted must not depend on this pass | `:125-127` | `if !r.graphHasLanguage("cobol") { return }` |
| COPY IDMS edges are skipped today; D-09 makes them external findings and moves them onto placeholders | `:167-169` | `case strings.HasPrefix(target, cobolIDMSCopyTargetPrefix): return` |
| Visit in-edges of placeholder nodes too (Pitfall 10). IDMS edges carry no `cobol_placeholder` (`cobol_grammar_refs.go:176-178`), so recognize them by `cobol_copy_form=idms` and recover the key from the placeholder ID | `:194-215` | `for _, edges := range r.graph.GetInEdgesByNodeIDs(candidateIDs) { ... if placeholder, _ := e.Meta[metaCobolPlaceholder].(string); placeholder != "" { add(e, placeholder) } }` |
| On `d.reason == "missing"` with `sql_include_builtin != true` (D-19), set `to` to the placeholder node ID before the compare-and-skip check, so identical reruns still write nothing | `:228-237` | `to, origin, crossRepo := ref.placeholder, e.Origin, false` ... `if e.To == to && e.Origin == origin && ... { continue }` |
| Call the new findings and placeholder pass next to the resources pass | `:288` | `r.reconcileCobolResources(repos, policies)` |
| IN-08: return a distinct reason with no finding when the candidate program carries the new containment-unresolved stamp | `:347-369` (`decideCobol` program branch) | `nested, known := c.Meta[metaCobolNested].(bool)` |

Add the IN-08 Meta key to the "Program node Meta read by this pass" group (`:35-39`). Keep the D-12 reason vocabulary comment at `:22-24` in sync.

**Persistence of edge moves** (`:276-287`): placeholder rebinding is a `To` change, so it flows through `graph.EdgeReindex` / `r.graph.ReindexEdges(rebound)`. No new persistence path is needed.

**Signature-change flag (Pitfall 4):** RESEARCH asks `reconcileCobol` to report created and evicted finding and placeholder IDs for FTS sync. `ReconcileCobol` has 2 production callers (`indexer/cobol_reconcile.go:70`, `:81`) and about 35 test callers, and `ResolveAll` calls `reconcileCobol` at `resolver.go:1674`. Run `gortex change verify` before changing the exported signature. The alternative is an additive accessor so the 37 call sites stay unchanged.

---

### `internal/indexer/finding_ledger.go` (NEW: durable ledger pass + FTS sync; GAP-06, GAP-07, Pitfalls 3, 4, 8)

**Analogs:**

**1. Tail structure** (`cobol_reconcile.go:49-82`). This is where the ledger hooks in:
```go
func (idx *Indexer) reconcileCobolAfterMutation(result *IndexResult, receipt *graph.MutationReceipt) {
	if idx == nil || !cobolMutationTouched(result, receipt) {
		return
	}
	idx.observeIncrementalCatchup("cobol", cobolMutationPaths(result, receipt))
	if idx.incrementalResolveFilesHook != nil || idx.resolver == nil {
		return
	}
	...
	idx.resolver.ReconcileCobol(repos)
}

func (mi *MultiIndexer) reconcileCobolRepos(repos map[string]struct{}) {
	master := mi.newMasterResolver(false)
	if master == nil {
		return
	}
	master.ReconcileCobol(repos)
}
```
Run the ledger pass on `idx.graph` / `mi.graph` (the durable store) right after `ReconcileCobol`. Never run it inside the resolver: the cold path points the resolver at an in-memory shadow (`indexer.go:2909`) whose sidecars are not drained (`sidecar_batch.go:99-123`).

**2. Optional-capability probe with a warning on failure** (`indexer/file_index_failures.go:239-245`):
```go
		if writer, ok := idx.graph.(graph.FileIndexFailureWriter); ok {
			if err := writer.ReplaceFileIndexFailures(idx.repoPrefix, rows); err != nil {
				idx.logger.Warn("indexer: persisting file failures failed", zap.String("repo", idx.repoPrefix), zap.Error(err))
				return
			}
```
History is best-effort (D-07): log and continue, never fail the index.

**3. FTS upsert and delete for resolver-created nodes** (`incremental_batch.go:1029-1057`):
```go
func (idx *Indexer) deleteSymbolFTS(nodeIDs []string) {
	if deleter, ok := idx.graph.(graph.SymbolFTSBatchDeleter); ok {
		if err := deleter.BatchDeleteSymbolFTS(nodeIDs); err != nil {
			idx.logger.Debug("indexer: backend FTS batch delete failed", zap.Error(err))
		}
	}
}

func (idx *Indexer) upsertIncrementalFTS(stages []*incrementalBatchStage) {
	batcher, ok := idx.graph.(graph.SymbolFTSBatchUpserter)
	...
			if !idx.shouldIndexForSearch(node) {
				continue
			}
			items = append(items, graph.SymbolFTSItem{
				NodeID: node.ID, Tokens: ftsTokensFor(node, idx.projectName),
			})
```
Reuse `deleteSymbolFTS` as-is. Use `ftsTokensFor` (`indexer.go:748-764`) and `shouldIndexForSearch` (`indexer.go:890-933`) for the upserts. `shouldIndexForSearch` does not exclude `KindFinding` or `KindResource`, so both are searchable once upserted.

**Reading active findings per repository** (`graph/scoped_projection.go:56`):
```go
func NodesInScopeSeq(s Store, repoPrefixes, filePaths []string, kinds ...NodeKind) iter.Seq[*Node]
```
Call it as `graph.NodesInScopeSeq(store, repos, nil, graph.KindFinding)`. The call is bounded and streaming on SQLite, and the fallback works on `*Graph`.

**Hook points** (production call sites of the COBOL reconcile):

| Path | Site | Note |
|---|---|---|
| standalone incremental | `cobol_reconcile.go:70` | inside `reconcileCobolAfterMutation` |
| multi incremental / config / untrack | `cobol_reconcile.go:81` (`reconcileCobolRepos`) | one sink for `multi.go:1615` (EndBatch), `:1652`, `:2597`, `:3449` (warm restart), `incremental_watcher_batch.go:437`, `repository_untrack.go:246` |
| cold track | `multi.go:2915-2926` (`TrackRepoCtx`) | **no COBOL tail exists today** (`ResolveAll` reconciles inside the resolver, `resolver.go:1674`). Add one after `trackRepoSourceCtx` returns, beside the `reconcileEnclosingRepos` tail at `:2920-2924` |
| deferred batch | `multi.go:3441-3450` | warm restart under `deferResolve` defers to `EndBatch`; the ledger follows the same deferral |

Pitfall 8: the pass is **not** gated on COBOL presence. It compares the open set for each repository even when no finding nodes remain.

---

### `internal/mcp/tools_enhancements.go` (MOD: finding counts in `index_health`; D-08, D-14)

**Scan-loop analog** (`:3206-3237`): the same cancel stride and the same nil checks, collected per repository:
```go
	scanned = 0
	for n := range s.graph.NodesByKind(graph.KindFile) {
		if scanned%healthScanCancelStride == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		scanned++
		if n == nil {
			continue
		}
		repoPrefixes[n.RepoPrefix] = struct{}{}
```
Iterate `s.graph.NodesByKind(graph.KindFinding)` the same way. Count by `prov_evidence_class`, or by the class segment of the ID (RESEARCH Open Question 4), restricted to `repoPrefixes`.

**Payload block analog** (`:3483-3510`, the D-15 "reported, never enforced" comment):
```go
	if prefixAuditOK {
		// D-15: status is reported, never enforced — an unhealthy store still
		// answers every query. Phases 6 and 7 decide what to refuse.
		...
		result["repo_ownership"] = ownership
	}
```
Add the block, for example `result["findings"] = map[string]any{"PARSE_UNRESOLVED": n, "EXTERNAL_UNRESOLVED": m}`, **after** `result` is built (`:3409-3425`).

**Must not:**
- feed `healthScore` (`:3168-3171`, `:3244-3246`);
- append to `recommendation` (`:3278-3326`).

This is D-14: cascades never mark a repository unhealthy.

---

### `internal/neo4jprojection/properties_test.go` (MOD: finding and placeholder Meta)

**Analog:** `TestProjectNode_CobolBreadthMeta` (`:321-397`). Append a `KindFinding` node and a placeholder `KindResource` node to the `nodes` slice. The key-by-key loop already handles `int` and `[]string`:
```go
	for _, node := range nodes {
		projected, warnings, err := projectNode("owner", "generation", node)
		...
		if warnings.Secret != 0 || warnings.Unsupported != 0 {
			t.Fatalf("%s: warnings = %+v, want no secret and no unsupported", node.ID, warnings)
		}
		for key, want := range node.Meta {
			...
			got, ok := projected.Properties["meta_"+key]
```
Also assert that the placeholder gets the `GortexUnresolved` label (pattern at `properties_test.go:109`; rule at `properties.go:54`).

**Secret filter** (`properties.go:171-180`): no `finding_*`, `artifact_*` or `prov_*` key may contain `password|passwd|secret|token|credential|privatekey|apikey|authorization` after stripping `- _ .` and spaces. Check new key names against this list.

---

### `cmd/gortex/cobol_findings_test.go` (NEW: get_symbol by exact ID through MCP and CLI, plus search; D-08, P2 D-11)

**Analog:** `cmd/gortex/cobol_tracer_test.go`.

| Piece | Location | Reuse |
|---|---|---|
| fixture repo, git commit, SQLite store, `TrackRepoCtx`, shadow off | `:115-182` (`indexCobolTracerRepo`) | copy the shape with a fixture holding a missing `CALL`, a missing `COPY` and a parse gap |
| in-process daemon over the real relay | `:312-359` (`serveCobolTestDaemon`) | **call directly**; it is already shared ("The COBOL tracer and breadth tests share it") |
| MCP `get_symbol` with `detail: "full"` (brief detail omits Meta) | `:420-440` | same call with a finding ID and a placeholder ID |
| CLI `gortex call get_symbol` | `:442-456` (`newCallTestCmd`, `callIndex`, `callArgs`, `runCall`) | same |
| `search_symbols` by name | `:458-476` | search by finding name and placeholder name |

```go
	text := callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": repo.programID, "detail": "full", "format": "json"})
```

---

### `internal/indexer/cobol_findings_lifecycle_test.go` (NEW: real-path lifecycle; GAP-06, GAP-07)

**Analog:** the Phase 4 harness. Reuse its helpers; do not copy them.

| Helper | Location |
|---|---|
| `newCobolLifecycleEnv(t, repos)`; sets `GORTEX_SHADOW_MAX_FILES=0` | `cobol_lifecycle_test.go:103-137` |
| `env.track`, `env.dump`, `cleanCobolDump`, `requireSameDump` | `cobol_lifecycle_test.go:139-192` |
| `env.write`, `env.remove`, `env.reindex`, `env.requireCleanEquivalent` | `cobol_lifecycle_incremental_test.go:64-95` |
| `env.restart`, `env.warmReconcile` | `cobol_lifecycle_incremental_test.go:366-398` |
| identical-rerun pattern (zero ledger rows is the analogue) | `cobol_lifecycle_incremental_test.go:141-154` |
| delete then restore sequence (reopen the same ID) | `cobol_lifecycle_incremental_test.go:167-219` |

**Shadow-path test (Pitfall 3):** call `t.Setenv("GORTEX_SHADOW_MAX_FILES", "<above corpus size>")` **after** `newCobolLifecycleEnv`, which sets it to `0` at `:108`. A later `t.Setenv` overrides it. Then assert that `finding_history` rows exist in SQLite.

**Fixture rule** (`cobol_lifecycle_test.go:25-27`): "every fixture below is invented fixed-format COBOL. It must never be replaced with estate-derived source."

---

### `internal/indexer/cobol_cascade_gate_test.go` (NEW: D-14 and D-15 gate in the default `go test -race`)

**Fixture analog:** `internal/parser/forest/cobolprobe/cascade_test.go:57-139`. Port the case construction verbatim (20 data items, 20 paragraphs, injection after paragraph 1 or before the first data item):
```go
	var data, proc strings.Builder
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&data, "       01  WS-FIELD-%02d          PIC X(10).\n", i)
	}
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&proc, "       %04d-PARA.\n           MOVE 'X' TO WS-FIELD-01.\n", i)
	}
	head := "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n"
	mid := "       PROCEDURE DIVISION.\n"
	...
	lines := strings.SplitN(proc.String(), "\n", 3)
	injectedProc := lines[0] + "\n" + lines[1] + "\n           EXEC CICS SEND MAP('M') END-EXEC.\n" + lines[2]
```

**Assertion style** (`:168-244`): every comparison is against the same clean control, and the control is pinned first.
```go
	if d != 20 || pa != 20 {
		t.Fatalf("clean control moved: got %d data items and %d paragraphs, want 20 and 20", d, pa)
	}
```

**Differences from the analog:**
- There is no `-enhanced-parser` flag skip (`:53-55`).
- Counts come from the real registered extractor through `newCobolLifecycleEnv` and SQLite, not from the raw grammar.
- Positive shapes (RESEARCH fixture table) assert one `uncontained-cascade` finding. Negative shapes assert 20 paragraphs and 20 data items and no cascade finding.

Keep `cobolprobe` unchanged (D-15).

---

### `internal/parser/languages/cobol_grammar_findings_test.go` (NEW: unit tests on the real producer)

| Helper | Location |
|---|---|
| `extractCobolGrammar` | `cobol_grammar_test.go:95-101` |
| `cobolFileNode` | `cobol_grammar_test.go:121-130` |
| `cobolTestHandoff` (program `Analyze`; a copybook needs `AnalyzeCopybook`) | `cobol_grammar_test.go:132-142` |
| `assertCobolRange` | `cobol_grammar_test.go:144-158` |
| `assertCobolMetaContract` (value types and secret-key words) | `cobol_grammar_contract_test.go:92-113` |
| fixed-format fixture style | `cobol_grammar_contract_test.go:22-90` |

- **Synthetic document-scope gaps** (`no-tree`, `budget-exceeded`, `invocation-failure`, `unmapped-capture`): build a real handoff with `cobolTestHandoff`, append `handoff.Gap` values, then build `newCobolWalk(...)` (`cobol_grammar_walk.go:63`) and call the mapper directly. Gortex never calls `handoff.Validate`.
- **WR-06 seam:** inside package `languages`, call `e.once.Do(e.init)` and then `e.analyzer = preprocessor.Analyzer{}` (RESEARCH Validation Architecture).
- **IN-08 fixture:** reuse the `cobolProgram(name, end)` builder (`cobol_grammar_test.go:305-317`) and the stolen-marker cases (`:321-348`).

---

### `internal/resolver/cobol_findings_test.go` (NEW)

| Helper | Location |
|---|---|
| `seedCobol(g, repo, id, kind, cobolKind, key)` | `cobol_references_test.go:15-33` |
| `cobolEdge(g, from, line, kind, to, meta)` | `cobol_references_test.go:35-46` |
| `cobolEdgeAt` | `cobol_references_test.go:48-58` |
| `seedSharedResource` and subtest layout (`orphan`, `referenced_kept`, ...) | `cobol_resources_test.go:13-67` |

Name tests `TestCobolReconcile_External*` / `TestCobolReconcile_Placeholder*` / `TestCobolReconcile_ContainmentUnresolved`, matching `TestCobolReconcile_ResourceGC` and the RESEARCH test map. Run every case on both `graph.New()` and a SQLite store where both are cheap. `cobol_references_test.go:12` already imports `store_sqlite` for that.

---

### `internal/mcp/index_health_findings_test.go` (NEW)

**Analog:** `index_health_ownership_test.go:19-68`. Copy the `readOwnership` closure (graph, then indexer, then `NewServer`, then `buildIndexHealthBasePayloadCtx`, then seed the cache, then `handleIndexHealth`) and read `payload["findings"]`. Add one case that proves `health_score` is unchanged with findings present (D-14).

---

### `internal/graph/finding_test.go` and `finding_ledger_test.go` (NEW)

**Analog:** `internal/graph/scope_test.go:1-42`: `package graph`, plain `testing`, `t.Fatalf` with got and want, no testify. Cover:
- ID determinism;
- line-shift invariance (feed the same evidence at different offsets);
- the `#k` ordinal, only for k >= 2;
- class separation (same evidence, different class, different ID);
- for the ledger: pairing order, container and class isolation, reopen after close, and zero transitions on an identical input.

---

## Shared Patterns

### Doc comments
**Source:** `resolver/cobol_references.go:63-123`, `resolver/cobol_resources.go:13-37`, `parser/languages/cobol_grammar_copybook.go:23-44`.
**Apply to:** every new non-obvious function (`cobolFindings`, the resolver pass, the ledger diff, the indexer ledger pass).
```go
// reconcileCobolResources collects orphaned COBOL shared resources and ...
//
// PURPOSE — ...
//
// RATIONALE — ...
//
// KEYWORDS — cobol, shared resource, orphan, gc, evict, ...
```
Exported symbols get a doc comment that starts with the name. A known gap gets a `WINDOW (review WR-xx):` paragraph (`cobol_grammar_copybook.go:37-43`).

### Build tags
**Source:** `cobol_grammar.go:1`, `cobol_lifecycle_test.go:1`.
**Apply to:** every file under `internal/parser/languages/cobol_grammar_*` and every indexer or cmd test that runs the real COBOL extractor: `//go:build darwin || linux`. Files in `internal/graph`, `internal/resolver` and `store_sqlite` take no tag. The Windows stub (`cobol_grammar_unsupported.go`) needs no change.

### Optional store capability plus compile-time assertion
**Source:** `graph/file_index_failures.go:16-25,57-60`; `store_sqlite/store_file_index_failures.go:73-76`; `graph/contract_node_evict.go:11-24`.
**Apply to:** `FindingNodeBatchEvicter` and the finding-history reader and appender. The interface lives in `internal/graph`. Backends assert `var _ graph.X = (*Store)(nil)`. Callers probe with `if x, ok := store.(graph.X); ok { ... }`. A dispatcher returns `supported=false` rather than falling back to N+1 point deletes.

### Meta contract
**Source:** `cobol_grammar_contract_test.go:96-113`; `neo4jprojection/properties.go:171-180`.
**Apply to:** every finding and placeholder Meta key.
- Value types are only `string, bool, int, float64, []string`.
- No secret-like substring may appear in a key.
- Never store raw gap source text (RESEARCH Security V8); store the digest and the range only.
- Repo-relative IDs in Meta follow `cobol_container` (`cobol_grammar.go:263-265`); prefixed IDs go on edge endpoints.

### Versioned rule IDs in Meta
**Source:** `cobolNameNormRule` (`cobol_grammar_identity.go:152-154`) written to `cobol_name_norm`; `cobolCopySearchRule` written to `cobol_copy_search_rule` (`cobol_references.go:259-260`).
**Apply to:** `finding_evidence_rule`, `finding_severity_rule` and the finding ID rule.

### Compare before write (idempotent reruns)
**Source:** `cobol_references.go:232-237`.
```go
		reason, _ := e.Meta[metaCobolUnresolvedReason].(string)
		if e.To == to && e.Origin == origin && e.CrossRepo == crossRepo &&
			reason == d.reason && cobolIDsEqual(e.Meta[metaCobolCandidateIDs], d.ids) &&
			cobolRankEqual(e.Meta, d) {
			continue
		}
```
**Apply to:**
- placeholder `To` rewrites;
- the "desired versus existing" external-finding set: add only missing nodes and evict only undesired ones;
- the ledger (zero rows on an identical rerun).

Disk stores decode `[]string` as `[]any` and ints as `int64` or `float64`. Compare with the `cobolIDsEqual` / `cobolRankEqual` tolerance (`:439-484`).

### Owned synthetic node on its own virtual path
**Source:** `cobol_grammar_walk.go:234-261` (`resourceNode`: `FilePath: id`), `cobol_resources.go:28-33` (evict only when `FilePath == ID`).
**Apply to:** placeholders, so file eviction never removes them and the resolver sweep collects them. Findings are the opposite case: their `FilePath` is the real source file, so `file_path` eviction removes them (D-05).

### Error handling by layer

| Layer | Pattern | Source |
|---|---|---|
| graph-core evicter (SQLite) | `panicOnFatal(err)`, deferred rollback unless committed | `store_sqlite/config_node_evict.go:41-94` |
| sidecar writer (SQLite) | return `error`, `defer tx.Rollback() //nolint:errcheck` | `store_file_index_failures.go:43-71` |
| indexer best-effort sidecar | probe the capability, `idx.logger.Warn(...)`, continue | `indexer/file_index_failures.go:239-245` |
| FTS sync | `idx.logger.Debug(...)` on failure | `incremental_batch.go:1029-1057` |
| extractor | hard error for init, grammar and program analysis failures; copybook analysis failure becomes an identity node plus a finding (WR-06 only) | `cobol_grammar.go:193-222` |

### Test commands
**Source:** RESEARCH Validation Architecture. All runs use `GOWORK=off go test -race -count=1 ...`. COBOL suites take about 5-30 seconds per package.

---

## Tests That Will Change (Pitfall 9)

| File | What changes |
|---|---|
| `internal/resolver/cobol_references_test.go` (about 18), `cobol_reconcile_test.go` (about 14) | `missing` edges move from `unresolved::cobol::...` to `<repo>/cobol-missing::...` placeholder nodes; COPY IDMS edges move too |
| `internal/indexer/cobol_lifecycle_crossrepo_test.go` (about 10), `cobol_lifecycle_incremental_test.go` (about 7; e.g. `:176-189` asserts `copyEdge[2] == placeholder` and `missing`), `cobol_lifecycle_material_test.go` (3) | same, plus new finding and placeholder rows in every dump |
| `cmd/gortex/cobol_breadth_test.go` (4 `missing`; `/5` at `:332`, `:450`) | placeholder target, `/6` |
| `cmd/gortex/cobol_tracer_test.go` (`/5` at `:206`, `:533`, `:550`) | `/6` |
| `internal/parser/languages/cobol_grammar_contract_test.go` (`/5`, two sites) | `/6`; `TestCobolGrammar_DegradedBreadth` / `ProvKeys` see finding nodes |
| `internal/parser/languages/cobol_grammar_copybook_test.go:280,627`, `cobol_grammar_test.go:521` | node counts include finding nodes |
| `internal/parser/languages/cobol_grammar_sql_test.go:272-282` | C2 (declared twice) becomes `ambiguous` (WR-07) |
| `internal/graph/store_sqlite/schema_sidecar_view_gen_test.go:285-290,425-431` | treat `finding_history` like `file_index_failures` |
| `internal/graph/store_sqlite/store_generation_read_test.go` (`:1711`, `:1745`), `store_generation_write_test.go` (`:672-691`) | new capability rows and write-fence case |

Extractor-level tests keep `unresolved::` targets. Only post-resolver assertions move to placeholders.

---

## No Analog Found

| File / piece | Role | Data Flow | Reason | Planner source |
|---|---|---|---|---|
| Ledger pairing diff in `internal/graph/finding_ledger.go` | pure diff | transform | No open-set versus active-set transition differ exists in the codebase | RESEARCH Pattern 6 |
| Fixed-format evidence normalization (drop byte columns < 6 and >= 72, collapse whitespace) | utility | transform | Gortex has no column-aware text rule; the only one is in the producer (`transform/fixed.go`) | RESEARCH Pattern 2 |

Everything else maps onto an existing COBOL, sidecar or capability precedent.

---

## Engineering-Standards Flags for the Planner

1. **Rule of three, SQLite kind-guarded evict.** `EvictFindingNodesByIDs` would be the third copy, after `config_node_evict.go` and `contract_node_evict.go`. Either extract a private `store_sqlite` helper, which edits two upstream files, or copy with a `// note:` comment. Decide and record the reason.
2. **Reuse, in-memory evict.** Delegate to the existing `(*Graph).evictSyntheticNodesByIDs` (`derived_contract_replace.go:103`). Do not write a fourth shard-lock loop.
3. **Single-use code stays local.** The container ladder, the cascade family check and the evidence normalization have one caller each, the extractor, so keep them private in `cobol_grammar_findings.go`. Only the finding ID function, the Meta key constants and the ledger types need `internal/graph`, because two packages must agree byte for byte.
4. **Exported signature change.** If `ReconcileCobol` gains return values, run `verify_change` first (2 production and about 35 test callers).
5. **Existing shared helpers to call, not copy:**
   - `serveCobolTestDaemon` (`cmd/gortex/cobol_tracer_test.go:315`);
   - the `cobolLifecycleEnv` helpers (`internal/indexer`);
   - `deleteSymbolFTS` (`incremental_batch.go:1029`);
   - `ftsTokensFor`;
   - `graph.NodesInScopeSeq`.

---

## Metadata

**Analog search scope:** `internal/graph` (and `store_sqlite`, `storetest`), `internal/parser/languages`, `internal/parser/forest/cobolprobe`, `internal/resolver`, `internal/indexer`, `internal/mcp`, `internal/neo4jprojection`, `internal/testutil/graphdump`, `cmd/gortex`, plus the pinned preprocessor module cache (`handoff/gaps.go`).
**Files scanned:** about 45 read in full or in targeted ranges.
**Pattern extraction date:** 2026-10-19
