---
phase: 05-explicit-unresolved-evidence
reviewed: 2026-10-21T18:20:27Z
depth: standard
files_reviewed: 55
files_reviewed_list:
  - cmd/gortex/cobol_breadth_test.go
  - cmd/gortex/cobol_findings_test.go
  - cmd/gortex/cobol_tracer_test.go
  - docs/languages.md
  - internal/graph/finding.go
  - internal/graph/finding_ledger.go
  - internal/graph/finding_ledger_test.go
  - internal/graph/finding_node_evict.go
  - internal/graph/finding_node_evict_test.go
  - internal/graph/finding_test.go
  - internal/graph/finding_writers_test.go
  - internal/graph/node.go
  - internal/graph/store_sqlite/finding_node_evict.go
  - internal/graph/store_sqlite/finding_node_evict_test.go
  - internal/graph/store_sqlite/migration_unprefixed_purge_test.go
  - internal/graph/store_sqlite/schema.go
  - internal/graph/store_sqlite/schema_sidecar_view_gen_test.go
  - internal/graph/store_sqlite/schema_version.go
  - internal/graph/store_sqlite/store_finding_history.go
  - internal/graph/store_sqlite/store_finding_history_test.go
  - internal/graph/store_sqlite/store_generation_read_test.go
  - internal/graph/store_sqlite/store_generation_write_test.go
  - internal/graph/store_sqlite/store_purge.go
  - internal/graph/storetest/storetest.go
  - internal/indexer/cobol_cascade_gate_test.go
  - internal/indexer/cobol_findings_equivalence_test.go
  - internal/indexer/cobol_findings_lifecycle_test.go
  - internal/indexer/cobol_lifecycle_crossrepo_test.go
  - internal/indexer/cobol_lifecycle_incremental_test.go
  - internal/indexer/cobol_lifecycle_material_test.go
  - internal/indexer/cobol_reconcile.go
  - internal/indexer/finding_ledger.go
  - internal/indexer/multi.go
  - internal/mcp/index_health_findings_test.go
  - internal/mcp/resources.go
  - internal/mcp/tools_enhancements.go
  - internal/neo4jprojection/properties_test.go
  - internal/parser/languages/cobol_grammar.go
  - internal/parser/languages/cobol_grammar_contract_test.go
  - internal/parser/languages/cobol_grammar_copybook.go
  - internal/parser/languages/cobol_grammar_copybook_test.go
  - internal/parser/languages/cobol_grammar_findings.go
  - internal/parser/languages/cobol_grammar_findings_test.go
  - internal/parser/languages/cobol_grammar_reasons_test.go
  - internal/parser/languages/cobol_grammar_sql.go
  - internal/parser/languages/cobol_grammar_sql_test.go
  - internal/parser/languages/cobol_grammar_structure_test.go
  - internal/parser/languages/cobol_grammar_test.go
  - internal/resolver/cobol_findings.go
  - internal/resolver/cobol_findings_test.go
  - internal/resolver/cobol_reconcile_test.go
  - internal/resolver/cobol_references.go
  - internal/resolver/cobol_references_test.go
  - internal/resolver/resolve_all_pending_test.go
  - internal/resolver/resolver.go
findings:
  critical: 1
  warning: 8
  info: 7
  total: 16
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-10-21T18:20:27Z
**Depth:** standard
**Files Reviewed:** 55
**Status:** issues_found

## Summary

I reviewed every file in `git diff 1e178afd..HEAD` (excluding `.planning/`), focusing on the phase-05 hunks. These parts hold up:

- **SQLite evicter (`finding_node_evict.go`):** the kind guard sits in both the node and edge `DELETE` subqueries, binds are fenced by `view_gen`, the work runs in one transaction, and the chunk size stays under the bind-parameter limit.
- **History sidecar:** generation-keyed, primary key `(view_gen, repo_prefix, seq)`. `seq` is assigned under `writeMu` inside the write transaction, and `beginWrite` enforces the sealed-generation write gate. The sidecar is registered for purge, rekey-drop, orphan scan, and the generation sweep.
- **Placeholder sweep:** ignores exactly the `references` in-edges whose source contains `::@finding/`.
- **Finding IDs:** no line, offset, producer gap ID, or parser version goes into an ID.
- **Finding Meta:** holds no raw source text. Producer reasons and kinds come from static vocabularies; external findings copy only `prov_*` edge keys.

`go test -race` on the finding, reconcile, and evict tests in `internal/graph`, `internal/graph/store_sqlite`, and `internal/resolver` passes.

The defects are concentrated in **identity stability** and **ledger determinism**:

- **CR-01 (empirically verified):** inserting one unrelated statement of the same kind earlier in a paragraph re-IDs existing parse findings. The ledger then records them as closed and opened, not kept. This directly violates D-01/GAP-06, and the ID scheme is costly to change after release.
- **Ledger correctness depends on unstated outside exclusion:** the ledger relies on exclusion it does not take itself (WR-01). It also skips some master drains (WR-02).
- **Evidence and container choices let IDs drift:** CALL, LINK, and XCTL produce the same evidence (WR-03), and document-scope findings change container with the program count (WR-07).

Verification method for CR-01: I compiled a probe test into `internal/parser/languages` through `go test -overlay`, so no repository file was modified. It ran the phase's own `cobolFindingShiftFixture` with one extra `EXEC SQL` statement inserted above the existing ones.

## Critical Issues

### CR-01: Parse finding IDs are anchored to ordinal statement IDs, so unrelated edits re-ID unchanged findings

**File:** `internal/parser/languages/cobol_grammar_findings.go:168-174`, `:220-229`; root cause `internal/parser/languages/cobol_grammar_statements.go:100-104`

**Issue:** `cobolFindingContainer` returns the innermost **statement** node first. Statement IDs are `<container>/@<kind>#k`, where `k` is the statement's ordinal among same-kind statements in its paragraph. `cobolAddFinding` hashes that container into the finding ID (`graph.FindingSymbol(container, ...)`) and writes it to `finding_container`.

Inserting any same-kind statement earlier in the paragraph shifts `k` for every later statement, and so changes the ID of every finding inside those statements. The finding's own evidence text is unchanged.

Probe result on `cobolFindingShiftFixture`, with `EXEC SQL SELECT AAA FROM APP.BBB END-EXEC.` inserted as the first statement of `MAIN-PARA`:
- Only 1 of 3 finding IDs survived.
- The `ZQXCOL` tail moved from `MAIN-PARA/@exec_sql_statement` to `…#2`, with a new ID.
- The `ZQXOTH` tail moved from `#2` to `#3`, with a new ID.

`FindingLedgerTransitions` pairs only on `(Container, Class, Kind)`, so these become `closed` plus `opened` rows, not even `superseded`. With D-18 per-token SQL tails, a single inserted `EXEC SQL` churns every tail finding below it in the paragraph.

This breaks:
- D-01: "Edits elsewhere in the file and line shifts keep the ID".
- GAP-06: unchanged evidence keeps its ID.
- The "accuracy without churn" preference.

The scheme is persisted, projected to Neo4j, and will be cited by Phase 8 claims, so it must be fixed before release.

The existing stability tests miss it because they only insert comment lines:
- `cobol_grammar_findings_test.go:283-317`
- `cobol_findings_equivalence_test.go:193-208`

**Fix:** anchor identity and the pairing key to a name-stable container, and keep the statement only as the structural parent.

```go
// identity anchor: paragraph/section/data section/program/file — never a statement ordinal
func cobolFindingAnchor(w *cobolWalk, c int) string { /* cobolFindingContainer minus the w.statement level */ }

anchor := cobolFindingAnchor(w, g.Container)
symbol := graph.FindingSymbol(anchor, graph.FindingClassParse, string(g.Kind), family, digest)
meta[graph.MetaFindingContainer] = anchor
w.memberOf(node.ID, cobolFindingContainer(w, g.Container), node.StartLine) // statement stays the member_of target
```

Add a regression test that inserts a same-kind statement above existing gaps in the same paragraph. Assert that the IDs are unchanged and that the ledger appends no row.

## Warnings

### WR-01: The finding ledger does not exclude concurrent finding mutations, so it can record false transitions

**File:** `internal/indexer/finding_ledger.go:14-17`, `:53-72`; `internal/indexer/multi.go:2931-2950`; `internal/resolver/cobol_findings.go:241`, `:292`, `:310`

**Issue:** `findingLedgerMu` only serializes ledgers against each other (the `TestCobolFindings_LedgerSerialized` scope). Finding mutations are guarded by different locks:
- the resolver reconcile, under `cobolReconcileMu`;
- file commits, under the repository lane and the global reach topology writer.

The resolver's own mutation is also three separate commits: `EvictFindingNodesByIDs`, then `AddBatch`, then the placeholder `EvictFile`.

`TrackRepoCtx`'s tail runs a master `ReconcileCobol` plus a ledger after `trackRepoSourceCtx` has returned. That puts it outside the topology writer. It is reachable from MCP `track_repository` (`internal/mcp/tools_multi.go:532`, `internal/mcp/auto_index.go:64`) while a watcher lane is mid-mutation in a repository the ledger covers (its base, or a `resolve_from` dependent drained from a resolver).

Two failure cases:
- **Unchanged evidence:** a ledger that reads between a lane's file commit (which evicts the file-owned external findings) and its reconcile (which re-adds them) appends `active→closed`. The lane's own ledger then appends `closed→active` for unchanged evidence.
- **Edited reference:** it records closed plus opened, not D-03's deterministic `superseded`.

**Fix:** make the ledger hold the same exclusion as the mutations it observes. Either:
- run `recordCobolFindingLedger` under the reach topology writer and the repository lanes of every repository it ledgers; or
- move the `TrackRepoCtx` tail inside `trackRepoSourceCtx`'s topology/lane section, and document plus assert (for example with a `held` flag check) that every caller holds it.

Add a concurrency test that interleaves a ledger with a reconcile and a file commit.

### WR-02: The master-resolve drain discards reconciled repositories without a ledger, so later transitions merge

**File:** `internal/indexer/multi.go:1027-1032`

**Issue:** `runMasterResolveHookedContext` drains `master.CobolFindingChanges()` only to sync search, and drops `repos`. The master is ephemeral, so this drain is the only record of which repositories it changed. The covered paths are:
- the incremental no-frontier fallback;
- `resolveDeferredMutations`' full fallback;
- `RunPreEnrichResolve`.

The next ledger runs at an unrelated settle point and records only the net diff. `TestCobolFindings_SearchAfterFallbackMasterResolve` (`cobol_findings_lifecycle_test.go:627`) pins this "no ledger at the master drain" behavior.

Example of the resulting misclassification:
1. The master closes Φ1, because its target was retrieved.
2. Before the next settle point, an edit adds a different missing CALL to the same paragraph.
3. The next ledger records "Φ1 superseded by Φ2".

Φ1 was actually closed by retrieval. The D-03 classification therefore depends on timing.

**Fix:** in `runMasterResolveHookedContext`, call `recordCobolFindingLedger(mi.graph, nil, tracked, mi.logger, master)` instead of only `CobolFindingChanges` plus the search sync. Alternatively, accumulate the drained repositories on `mi` for the next settle point. Update the test to expect the row at the master step.

### WR-03: External finding evidence does not distinguish CALL, LINK, XCTL, or LOAD

**File:** `internal/resolver/cobol_findings.go:379-390`; `internal/parser/languages/cobol_grammar_refs.go:79-81`; `internal/parser/languages/cobol_grammar_mainframe.go:164-169`

**Issue:** the evidence is `cobol_call` + IDMS kind + raw target. `cobol_call` is `literal` or `dynamic`, not the verb. `EXEC CICS LINK`, `XCTL`, and `LOAD PROGRAM` reuse `cobolCallTarget`, which stamps the same `literal`. The edge kind is not hashed either.

Consequences:
- `CALL 'X'` and `EXEC CICS XCTL PROGRAM('X')` in the same paragraph produce the same base ID and differ only by `#k`.
- Editing one into the other keeps the finding ID.

This contradicts D-01 ("editing the gap's own text produces a new ID") and the function's doc ("the edge's call or copy form").

**Fix:** add the edge `Kind`, `prov_observation_kind` (`call_statement` or `exec_cics_statement`), and the CICS command (stamp it on the edge if it is absent) to the evidence before hashing. Bump the evidence rule (`gortex-finding-evidence/2`).

### WR-04: The evidence digest is unsalted, so single-token evidence is recoverable by dictionary attack

**File:** `internal/graph/finding.go:55-72`, `:104-120`; `internal/parser/languages/cobol_grammar_findings.go:192`

**Issue:** `FindingEvidenceDigest` is an unkeyed BLAKE3 of a public rule string plus the evidence. Under D-18 each unparsed-tail finding hashes exactly one token, such as a host variable, table, column, or literal. The digest is stored in `finding_evidence_digest`, is folded into the ID, is searchable, and is projected to Neo4j. The projection's secret filter (`neo4jprojection/properties.go:174`) inspects key names only.

Anyone holding the projection can confirm or recover tokens from a name dictionary. A literal credential inside an unparsed `EXEC SQL … USING 'pw'` tail becomes a fast, unsalted hash of that secret. The doc comment's claim that "protected estate source never leaks through a finding's ID … or Neo4j projection" is false for low-entropy evidence.

**Fix:** use a keyed digest, `blake3.NewKeyed(key)`, with a per-workspace key persisted alongside the workspace config. Use it for both the evidence digest and `findingDigest`, so IDs stay deterministic within an estate. Alternatively, drop `finding_evidence_digest` from projected Meta and correct the privacy claim. At minimum, document the residual risk.

### WR-05: A placeholder ID can collide with a real repository path, and the sweep then evicts real file nodes

**File:** `internal/resolver/cobol_findings.go:150-163`, `:250-270`, `:310`

**Issue:** placeholders use ID = FilePath = `<repo>/cobol-missing::<type>/<key>`. `:` is legal in POSIX file names, and `cobolIDSegment` does not escape `.`.

A repository file at `cobol-missing::program/X.CBL`, referenced by a missing `CALL 'X.CBL'`, has a file node with that exact ID. Two things then go wrong:
- `AddBatch` upserts the placeholder over the real file node.
- The sweep calls `EvictFile(n.FilePath)`, which deletes every node with that `file_path`, including the real file's programs and paragraphs.

The kind and Meta guard at `:156` only inspects the node at the ID, not what `EvictFile` removes. That contradicts the T-05-17 claim that "no node owned by a real file is ever removed". The likelihood is low, but the input is attacker-controllable through repository content.

**Fix:** place placeholders in a namespace that no repository path can produce, for example a `<repo>::cobol-missing/<type>/<key>` ID with an empty or virtual FilePath (the audit already admits owned `<RepoPrefix>::` IDs). Then evict by ID with a kind and `cobol_kind=missing_artifact` guard, not `EvictFile`. Refuse to create a placeholder when a non-placeholder node already exists at the ID.

### WR-06: Same-line duplicate references collapse into one EXTERNAL_UNRESOLVED finding

**File:** `internal/parser/languages/cobol_grammar_dedupe.go:26-37`; `internal/graph/store_sqlite/schema.go:871`; `internal/resolver/cobol_findings.go:196-224`

**Issue:** edges are deduplicated on `From|To|Kind|Line`, and the edge table is UNIQUE on `(from_id, to_id, kind, file_path, line, view_gen)`. So `CALL 'X'. CALL 'X'.` on one line in one paragraph, or a CALL and a CICS LINK of the same program on one line, become a single edge with `cobol_occurrence_count=2`.

`reconcileCobolFindings` emits one finding per desired edge and never reads `cobol_occurrence_count`. That violates D-04 (one finding per reference occurrence) and undercounts the artifact-level aggregates D-04 says are derived from findings.

**Fix:** when building `findings`, expand an edge with `cobol_occurrence_count = n` into `n` occurrences. Order them deterministically from `cobol_statement_symbols`, so the existing `#k` ordinal logic assigns `#2..#n`. Alternatively, stop collapsing literal CALL/COPY relation edges.

### WR-07: Document-scope findings change container (and ID) with the program count

**File:** `internal/parser/languages/cobol_grammar_findings.go:261-268`

**Issue:** document-scope findings attach to the single program when exactly one exists, else to the file node. This covers:
- `no-tree`, `budget-exceeded`, and `invocation-failure`;
- `uncontained-cascade` and `analysis-failed`;
- gaps outside every program.

Adding a nested program elsewhere in the file, or a cascade that hides or recovers a `PROGRAM-ID` while the user edits toward a fix, therefore moves the container. That re-IDs the finding, and the ledger logs closed plus opened (different container), not "unchanged". This is the same D-01 "edits elsewhere" violation as CR-01, through a different path.

**Fix:** always anchor document-scope findings to the file node, which is stable for the document's lifetime. Keep the single-program association, if wanted, as a non-identity `member_of` edge or a Meta field.

### WR-08: Programs hidden by visibility rules are reported as "not obtained"

**File:** `internal/resolver/cobol_references.go:430-445`, `:459-463`, `:478`, `:290`

**Issue:** `decideCobol` silently drops existing in-scope candidates in three cases:
- a nested program called from another file;
- a sibling under resolved containment (COMMON is not observed);
- a legacy candidate with no `cobol_nested` flag (`!known`).

It then returns `missing`. Line `:290` turns every `missing` into a `not_obtained` placeholder plus an EXTERNAL_UNRESOLVED finding. IN-08 exempts only *unresolved* containment.

The result is that a program indexed in the same file gets a finding claiming it was never obtained. That is broader than the phase's "the artifact was not obtained" contract and D-09's "covers missing artifacts only". `cobol_references_test.go:277` now pins the placeholder for a sibling program that exists.

**Fix:** set a `hidden` flag whenever a candidate is dropped for visibility. When the candidate list ends up empty and `hidden` is set, return a distinct reason (for example `not_callable`) that produces neither a placeholder nor a finding, as `containment_unresolved` already does.

## Info

### IN-01: The v23 migration test cannot fail on the migration step

**File:** `internal/graph/store_sqlite/store_finding_history_test.go:197-205`; `internal/graph/store_sqlite/store.go:563`, `:624`

**Issue:** `Open` runs `schemaSQL`, which already includes `CREATE TABLE IF NOT EXISTS finding_history` from `sidecarSchemaSQL`, before `applyInPlaceMigrations`. So the table and index exist again whether or not `createFindingHistoryTable` does anything, and the test passes even if the v23 step is a no-op. The idempotency loop only runs against a store that already has the table.

**Fix:** call `createFindingHistoryTable` in a transaction against a raw database that lacks the table, and assert that it creates it. Otherwise, acknowledge that the step is redundant.

### IN-02: History reader and appender skip the generation-isolation checklist

**File:** `internal/graph/store_sqlite/store_generation_read_test.go:1761-1762`

**Issue:** both capabilities are skipped as "covered by TestFindingHistory_*", but no `TestFindingHistory_*` test uses a derived generation. `view_gen` fencing of history reads and appends is therefore untested.

**Fix:** add a base plus derived handle case: append through each, and assert that each reads only its own rows.

### IN-03: Evicted IDs are recorded even when the store removed nothing

**File:** `internal/resolver/cobol_findings.go:241-243`

**Issue:** `evicted = undesired` ignores the `(nodes, edges)` counts. If the SQLite evicter removes fewer rows (for example, a row inherited by a derived generation, which the `view_gen = ?` delete cannot reach), search sync still deletes the FTS rows of a node that remains visible.

**Fix:** record only the IDs whose nodes are gone afterward (`GetNodesByIDs`), or skip the FTS delete when `nodesRemoved < len(undesired)`.

### IN-04: The per-indexer settle point syncs other repositories' IDs through its own indexer

**File:** `internal/indexer/cobol_reconcile.go:80-81`

**Issue:** `idx.resolver.ReconcileCobol` widens to the `resolve_from` dependents, but `idx.syncCobolFindingFTS` upserts every drained ID with `idx.projectName` tokens. The MultiIndexer path routes each ID by prefix (`syncCobolFindingFTSByRepo`); this path does not.

**Fix:** route by `graph.RepoPrefixOfID` when an owner MultiIndexer exists, or restrict the sync to `idx.repoPrefix`.

### IN-05: Extractor tests size node counts by the emitted findings, so spurious findings on clean source pass

**File:** `internal/parser/languages/cobol_grammar_copybook_test.go:280`, `:630`; `internal/parser/languages/cobol_grammar_structure_test.go:243`; `internal/parser/languages/cobol_grammar_test.go:527`

**Issue:** assertions such as `Len(nodes, 3+len(cobolFindingNodes(result, "")))` accept any number of findings. A regression that emits findings on green fixtures stays green.

**Fix:** for fixtures known to be green, assert `len(findings) == 0`. Keep the adaptive form only for degraded fixtures.

### IN-06: Docs and D-09 omit `EXEC CICS LOAD PROGRAM` as an EXTERNAL_UNRESOLVED source

**File:** `docs/languages.md:279`

**Issue:** a literal `LOAD PROGRAM('X')` is a `cobol::program::` references edge, so it also gets a placeholder and a finding (`cobol_findings_test.go` `code_references_kept`). The docs list only `CALL`, `LINK`, `XCTL`, `COPY`, `-INC`, and `EXEC SQL INCLUDE`.

**Fix:** document `LOAD`, or exclude it deliberately.

### IN-07: Unrelated gofmt reindentation churn in a phase test

**File:** `internal/graph/store_sqlite/store_generation_write_test.go:675-690`, `:746-770`, `:936-948`

**Issue:** whitespace-only reindentation of pre-existing seed blocks inflates the phase diff and obscures the one real addition (the `finding_node_evict` case).

**Fix:** split it into its own `style:` commit, or revert it.

---

_Reviewed: 2026-10-21T18:20:27Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
