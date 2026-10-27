---
phase: 05-explicit-unresolved-evidence
reviewed: 2026-10-26T20:46:09Z
depth: standard
reviewed_diff_base: 33d287d4
reviewed_head: 3525a1fe
supersedes: 3040addf
files_reviewed: 33
files_reviewed_list:
  - cmd/gortex/cobol_breadth_test.go
  - cmd/gortex/cobol_findings_test.go
  - docs/languages.md
  - internal/graph/finding.go
  - internal/graph/resource_node_evict.go
  - internal/graph/resource_node_evict_test.go
  - internal/graph/store_sqlite/finding_node_evict.go
  - internal/graph/store_sqlite/resource_node_evict.go
  - internal/graph/store_sqlite/resource_node_evict_test.go
  - internal/graph/store_sqlite/store_generation_read_test.go
  - internal/graph/store_sqlite/store_generation_write_test.go
  - internal/graph/storetest/storetest.go
  - internal/indexer/cobol_findings_equivalence_test.go
  - internal/indexer/cobol_findings_identity_test.go
  - internal/indexer/cobol_findings_ledger_exclusion_test.go
  - internal/indexer/cobol_findings_lifecycle_test.go
  - internal/indexer/cobol_lifecycle_crossrepo_test.go
  - internal/indexer/cobol_lifecycle_incremental_test.go
  - internal/indexer/cobol_lifecycle_material_test.go
  - internal/indexer/cobol_reconcile.go
  - internal/indexer/finding_ledger.go
  - internal/indexer/incremental_watcher_batch.go
  - internal/indexer/multi.go
  - internal/indexer/repository_untrack.go
  - internal/neo4jprojection/properties_test.go
  - internal/parser/languages/cobol_grammar_findings.go
  - internal/parser/languages/cobol_grammar_findings_test.go
  - internal/reach/reach.go
  - internal/resolver/cobol_findings.go
  - internal/resolver/cobol_findings_test.go
  - internal/resolver/cobol_reconcile_test.go
  - internal/resolver/cobol_references.go
  - internal/resolver/cobol_references_test.go
findings:
  critical: 1
  warning: 2
  info: 6
  total: 9
status: issues_found
---

# Phase 05: Code Review Report (gap-closure re-review, 05-13 to 05-17)

**Reviewed:** 2026-10-26T20:46:09Z
**Depth:** standard
**Scope:** `git diff 33d287d4..HEAD -- . ':!.planning/'` at HEAD `3525a1fe` (33 files, +1980/-311)
**Supersedes:** the initial review at `3040addf`
**Status:** issues_found

## Summary

The gap-closure plans fix what they targeted. Specifically:

- **Identity (05-13):** CR-01 and WR-07 are closed. The anchor ladder in `cobolFindingAnchor` never reaches a statement, document scope always lands on the file, and the regressions insert and delete same-kind statements on both the extractor and the real path.
- **Ledger (05-14):** D-23 holds for every lane path I traced. The deadlock audit found no self-acquisition of the writer:
  - every `reconcileCobolRepos` caller is outside a writer section;
  - the batch branch is reachable only under a test-only batch configuration;
  - untrack, the incremental lanes, and the warm closure use the held variants.
- **Evicter (05-15):** sound. The kind and `view_gen` predicates are on every statement, there is one transaction, and conformance passes on both backends.
- **Placeholders (05-16):** the namespace, the by-ID sweep, and the refusal work for ordinary repository names.

Three defects remain. All three are new findings: CR-01 is a regression introduced by 05-16, WR-02 is a defect in new 05-13 code, and WR-01 is what remains of prior WR-02:

| ID | Severity | What breaks |
|---|---|---|
| CR-01 | BLOCKER | The D-27 helpers parse placeholder IDs by cutting at the **first** `::`. In a repository whose prefix contains `::`, every COPY IDMS EXTERNAL_UNRESOLVED finding is deleted on the next unchanged reconcile pass. **Proven by probe** on both stores. Ledger and search routing for that repository is also misattributed. |
| WR-01 | WARNING | D-24 is only half closed. A writer-free master's reconcile runs inside `ResolveAllContext`, after `markReady`, but its ledger takes the writer only after the call returns. A lane started after ready therefore lands its whole commit, reconcile, and ledger in between and records the master's close as a supersede. The docs claim this cannot depend on timing. |
| WR-02 | WARNING | D-26 relabels the absence reason of collapsed **imageless** edges (Line 0) as `same_line_occurrence`. **Proven by probe.** |

Probe method: `go test -overlay` with a throwaway test file in `$TMPDIR`. No repository file was modified. Commands were targeted, with `-timeout 300s`. The full suite was not run.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Placeholder IDs are split at the first `::`, so a repository prefix containing `::` loses its COPY IDMS findings and misroutes ownership

**Files:**
- `internal/resolver/cobol_findings.go:92-99` (`cobolUnresolvedFromPlaceholder`)
- `internal/indexer/finding_ledger.go:297-304` (`cobolOwnerPrefix`)
- callers:
  - `internal/resolver/cobol_references.go:285-291`
  - `internal/resolver/cobol_references.go:210-213`
  - `internal/indexer/finding_ledger.go:187`, `:272`, `:277`

**Issue:** D-27 moved placeholders to `<repo>::cobol-missing::<type>/<key>`. Both new parsers assume the repository prefix contains no `::`, and nothing validates that:
- `config.ResolvePrefix` (`internal/config/global.go:445-450`) returns `entry.Name` or the directory basename unchanged.
- `AddRepo` checks only for duplicate paths.

The old code cut at the first `/`, which a prefix cannot contain, so both breaks are regressions.

1. **Findings are deleted (GAP-02 / D-09).** `cobolUnresolvedFromPlaceholder` runs `strings.Cut(id, "::")`.
   - For repository `a::b`, the remainder is `b::cobol-missing::idms_dictionary_member/...`, so the `CutPrefix` fails and the function returns `""`.
   - A COPY IDMS edge carries no `cobol_placeholder`, so `reconcileCobol` passes `""` to `add`, which returns at `:210-213`.
   - The edge is never collected again. Its finding drops out of `wanted` and is evicted as undesired at `cobol_findings.go:284-295`.
   - The placeholder survives, because the code edge still targets it.
   - A cold track runs two passes (the per-repository `ResolveAll`, then the tail's master `ReconcileCobol`). So the finding exists only between them: the history gets an open and then a false close, and the reference ends with no finding.

   Probe `TestReviewProbe_IDMSColonRepo` (repository `a::b`, one COPY IDMS, two unchanged `ReconcileCobol(nil)` passes):

   ```
   pass1 findings=[a::b/main.cbl::@finding/external/dbc4ebf1...] placeholders=[a::b::cobol-missing::idms_dictionary_member/RECORD/EMP-REC]
   pass2 findings=[]                                              placeholders=[a::b::cobol-missing::idms_dictionary_member/RECORD/EMP-REC]
   ```

   The memory and SQLite runs give the same result.

2. **Ownership is misrouted.** `cobolOwnerPrefix` returns the text before the first `::` whenever that comes before the first `/`.
   - Both `a::b/cobol/x.cbl::@finding/...` and `a::b::cobol-missing::...` therefore route to `a`.
   - **Ledger:** the tracked-prefix widening (`:184-195`) silently skips `a::b`.
   - **Search sync:** `syncCobolFindingFTSByRepo` (`:268-288`) either skips `a::b`'s IDs, or, when a repository `a` is tracked, writes them through `a`'s Indexer with `a`'s project tokens.
   - For finding IDs this is also a regression: the previous `graph.RepoPrefixOfID` returned `a::b` correctly.

The constant comment at `cobol_findings.go:15-21` says "no repository file, program, or paragraph can produce a placeholder ID". That overclaims for the same reason; the WINDOW paragraph at `cobol_references.go:148-158` admits the case. The `docs/languages.md:279` reserved-name sentence does not mention `::`.

**Fix:** parse from the namespace marker, not the first separator. Use `LastIndex` so that a prefix which itself spells the marker still parses: keys escape `:` (`cobolIDSegment`), so the last marker is always the real one.

```go
// resolver
func cobolUnresolvedFromPlaceholder(id string) string {
	i := strings.LastIndex(id, "::"+cobolMissingNamespace)
	if i < 0 {
		return ""
	}
	rest, ok := strings.CutPrefix(id[i+2:], cobolMissingNamespace+cobolArtifactIDMSMember+"/")
	if !ok || rest == "" {
		return ""
	}
	return graph.UnresolvedMarker + cobolIDMSCopyTargetPrefix + rest
}

// indexer (or a shared graph helper; see IN-02)
func cobolOwnerPrefix(id string) string {
	if i := strings.LastIndex(id, "::cobol-missing::"); i > 0 {
		return id[:i]
	}
	return graph.RepoPrefixOfID(id) // file-owned finding IDs: "<repo>/<path>::..."
}
```

Then add resolver and indexer regressions with a `::`-containing prefix:
- two unchanged passes keep the IDMS finding;
- the ledger and search sync route to `a::b`.

Alternatives:
- stamp `cobol_placeholder` on COPY IDMS edges, so nothing parses IDs at all;
- reject `::` in repository prefixes at track time and add it to the docs' reserved-name sentence.

## Warnings

### WR-01: D-24 is timing-dependent on writer-free master paths: a lane can ledger between the master's reconcile and the master's ledger

**Files:**
- `internal/indexer/multi.go:1059-1070`
- `internal/resolver/resolver.go:645`, `:1202`, `:1691`
- `cmd/gortex/daemon_state.go:949`, `:999-1001`
- `internal/mcp/tools_fileops.go:785`, `internal/mcp/tools_core.go:1578`
- `docs/languages.md:279`

**Issue:** `runMasterResolveHookedContext` now ledgers its own drain (D-24). With `topologyHeld == false`, that ledger is not atomic with the reconcile it records. This covers the daemon warmup `RunPreEnrichResolve` and the public deferred catch-up full fallback.

`ResolveAllContext` holds the store `ResolveMutex` (`r.mu = g.ResolveMutex()`, `resolver.go:645`) and runs these steps in order:
1. it fires `OnComputeDone`, which is the warmup's `markReady` (`:1202`);
2. it runs `reconcileCobol`, which closes Φ1 when its target was retrieved (`:1691`);
3. it continues with the rest of the pass;
4. it returns, and only then does `recordCobolSettleLedger` request the reach writer (`multi.go:1070`).

An agent edit after ready interleaves as follows:
1. The edit (`reindexFile` → `IncrementalReindexRepo`) takes the writer and commits `CALL 'NEWPGM'` into Φ1's paragraph.
2. It blocks on `ResolveMutex` for its resolver tails.
3. When the master returns, the lane runs first, because the master is now waiting for the writer the lane holds. The lane reconciles and runs its own settle ledger.
4. That ledger diffs the latest rows (Φ1 active) against the store (Φ1 gone, Φ2 open, same container) and writes "Φ1 superseded by Φ2".
5. The master's ledger then finds nothing.

This is exactly WR-02's misclassification. It persists whenever a lane overlaps a writer-free master.

Neither 05-14 test exercises this window:
- `TestCobolFindings_MasterDrainClosesBeforeLaterEdit` is sequential.
- `LedgerExcludesTornMutation/master_drain` parks after the writer is taken.

`docs/languages.md:279` states: "a fallback full resolve records its transitions at the step that made them, so whether a finding is closed or superseded does not depend on timing". That is not true on these paths.

**Fix:** pick one of:
1. On writer-free master paths, suppress `reconcileCobol` inside `ResolveAllContext` (a resolver flag), and run `mi.reconcileCobolRepos(scope)` after it returns. Reconcile, drain, ledger, and search sync then share one writer section, as every other settle point does.
2. Accept the residual under D-07, record it in the 05-14 accepted windows, and change the docs sentence to say the guarantee holds only when no edit overlaps a fallback or warmup resolve.

Either way, add an interleaving test that parks a lane between the master's reconcile and its ledger. That needs a hook after `ResolveAllContext` returns, before `BeginTopologyMutation`.

### WR-02: D-26 overwrites the absence reason of collapsed imageless edges with a false `same_line_occurrence`

**Files:**
- `internal/resolver/cobol_findings.go:243-253`
- root cause:
  - `internal/parser/languages/cobol_grammar_dedupe.go:26-31`
  - `internal/parser/languages/cobol_grammar_walk.go:268-277`
- `docs/languages.md:279`

**Issue:** the expansion assumes every collapsed edge stands for references on one real line. That does not hold for references with no image:
- `cobolStampMeta` returns Line 0 for a `NoImage` or `Synthetic` projection, and stamps `prov_range_absence=no_original_projection` or `synthetic_projection`.
- `cobolDedupeEdges` keys on `From|To|Kind|Line`.

So two imageless references to one target from one container collapse whatever their real lines are. For occurrences 2..n, the expansion then:
- replaces the edge's true absence reason with `same_line_occurrence`, a claim nothing established;
- sets `EndLine = StartLine`.

This contradicts the carried-forward rule that absence reasons are surfaced as-is.

Probe `TestReviewProbe_NoImageOccurrence` (edge with Line 0, `prov_range_absence=no_original_projection`, `cobol_occurrence_count=2`, memory and SQLite):

```
...c41d    absence=no_original_projection exact=false start=0 end=0
...c41d#2  absence=same_line_occurrence   exact=false start=0 end=0
```

A secondary problem also affects imaged edges. `EndLine = StartLine` invents an end line for a later occurrence that spans lines, for example `CALL 'X'. CALL 'X'` followed by `USING A.` on the next line.

**Fix:** relabel only when the edge has an exact range. Otherwise keep the edge's own absence reason:

```go
if i > 0 {
	for _, k := range rangeKeys { delete(node.Meta, k) }
	node.Meta["prov_range_exact"] = false
	if exact, _ := d.e.Meta["prov_range_exact"].(bool); exact && d.e.Line > 0 {
		node.Meta["prov_range_absence"] = "same_line_occurrence"
	} // else keep the edge's prov_range_absence verbatim
	node.EndLine = 0 // the occurrence's end line is unknown; do not invent one
}
```

If consumers require `EndLine >= StartLine`, document that `EndLine` is a lower bound for these findings. Add the imageless case to `TestCobolReconcile_ExternalSameLineOccurrences`.

## Info

### IN-01: The old-store note omits that pre-D-27 external findings stay active beside their replacements

**File:** `docs/languages.md:279`; `internal/resolver/cobol_findings.go:193`, `:206-221`

**Issue:** the docs say a store indexed before the gap closure keeps its old placeholders and the old parse finding IDs.

The reconcile now builds `owned`, and therefore `present`, only from placeholders under the new namespace. So an old external finding:
- is never evicted, because its `references` edge targets an old-shape placeholder;
- sits next to the new finding for the same reference.

Probe `TestReviewProbe_OldStoreExternal`: after two passes, one CALL has two active EXTERNAL_UNRESOLVED findings and two placeholders on both stores. `index_health` and search therefore double-count until the referencing file is re-extracted. No production store holds Phase 5 findings yet, so the impact is limited to developer stores.

**Fix:** extend the sentence, for example: "...and the old external findings of every unchanged file, which are counted next to the new ones". Alternatively, sweep old-shape (`<repo>/cobol-missing::`) placeholders and their finding in-edges once.

### IN-02: Placeholders drop out of workspace-clamped community and process views

**File:** `internal/mcp/scope_resolve.go:466`, `:561` (05-16 audit rows 20 and 22, accepted as fail-closed)

**Issue:** `graph.RepoPrefixOfID` returns `<repo>::cobol-missing::<type>` for a placeholder. That value is in no workspace repository set, so workspace-bound and `repo_allow` sessions lose owned, legitimate placeholder nodes from `clusters`, `concepts`, `suggest_boundaries`, and process steps. The corrected owner-prefix logic lives only in package `indexer`.

**Fix:** move one corrected helper into `internal/graph`, with the CR-01 `LastIndex` form, and use it at both MCP sites and at `finding_ledger.go`.

### IN-03: The D-23 interleaving test turns a regression into a package-timeout hang

**File:** `internal/indexer/cobol_findings_ledger_exclusion_test.go:90`, `:100`

**Issue:** if a settle point stops emitting the `ledger` stage, `<-aParked` blocks until `go test -timeout` panics the whole package. The 05-17 gate uses 60m. The same happens with `<-bCommitted` in the mutation branch. The 05-14 SUMMARY acknowledges this.

**Fix:** a bounded `select` on a generous deadline is deterministic and is not a sleep:

```go
select { case <-aParked: case <-ctx.Done(): t.Fatal("settle point never reached its ledger stage") }
```

Here `ctx` comes from `context.WithTimeout(t.Context(), 2*time.Minute)`.

### IN-04: Inside a repository topology batch, D-23 exclusion rests on `deferResolve`, not on the writer

**File:** `internal/indexer/repository_topology_batch.go:39-44`, `:68-80`; `internal/indexer/multi.go:3012-3013`, `:3546`

**Issue:** `RunRepositoryTopologyBatch` holds one writer for all members, and members run in parallel. The held branches therefore give batch members no mutual exclusion: one member's inline settle ledger could read another member between its commit and its reconcile. Today this is unreachable in production. The only batch (`cmd/gortex/daemon_state.go:638`) runs under `BeginParallelBatch`, so `deferResolve` is true and both tails defer to `EndBatch`. Nothing enforces that.

**Fix:** document the precondition at both held branches, and fail loudly in tests (or log) when `activeRepositoryTopologyBatch(ctx, mi) != nil && !deferResolve` reaches an inline settle ledger.

### IN-05: The mixed-form window understates the collapse

**File:** `docs/languages.md:279`; `internal/parser/languages/cobol_grammar_dedupe.go:26-31`

**Issue:** the dedupe key uses the normalized `To`. So two spellings of one target on one line also collapse, and every finding carries the first spelling. Editing the second spelling then changes no ID. Example: `CALL 'pgmx'. CALL 'PGMX'.`

The docs name only statement forms. Imageless references (WR-02) collapse across lines as well.

**Fix:** widen the window sentence to cover "any two literal references to the same target that share a container, edge kind, and line (or that both lack a source position)".

### IN-06: The `reconcileCobolRepos` comment claims `finish(false)` keeps the reach index valid

**File:** `internal/indexer/cobol_reconcile.go:91-101`, `:103-104`; `internal/indexer/multi.go:2649-2651`; `internal/reach/reach.go:1002-1015`

**Issue:** the reconcile can rebind CALL and COPY edges. Examples:
- a config reload re-decides bindings through `RefreshRepoConfigs`;
- the cold-track tail applies its cross-repository refinement.

`reach.InvalidateIndex` says it must run "whenever the graph mutates in a way that could change reach sets", and `finish(false)` skips it. This is pre-existing behavior (the pass took no writer before 05-14), but the new comment asserts the opposite.

**Fix:** either pass `true` when the reconcile rebound or removed edges (via a returned flag), or correct the comment to say the reach index is not invalidated here.

## Prior Findings Status

Status of the initial review's findings (`git show 3040addf:.planning/phases/05-explicit-unresolved-evidence/05-REVIEW.md`):

| Prior | Status | Evidence |
|---|---|---|
| CR-01 (statement-ordinal anchor) | **Resolved** | `cobol_grammar_findings.go:177` (anchor), `:183` (ID), `:203` (`finding_container`), `:218-222` (`member_of` keeps the statement), `:237-276` (`cobolFindingAnchor` has no statement level). Tests: `TestCobolGrammar_Findings_SameKindInsertKeepsIDs` / `_SameKindDeleteKeepsIDs`, and on the real path `TestCobolFindings_SameKindInsertKeepsIDs` / `_SameKindDeleteClosesOnlyItsOwn`. `FindingIDRule` is `/2` (`finding.go:32`). |
| WR-01 (ledger exclusion) | **Resolved** | Every MultiIndexer ledger goes through `recordCobolSettleLedger`, which takes or inherits the writer (`cobol_reconcile.go:167-179`). `recordCobolFindingLedger` holds `cobolReconcileMu` across drain, read, diff, and append (`finding_ledger.go:159-162`). The `TrackRepoCtx` tail is batch-aware (`multi.go:3012-3016`). `TestCobolFindings_LedgerExcludesTornMutation` covers it. The cold-track gap is accepted (05-14 SUMMARY). See IN-04 for the in-batch precondition. |
| WR-02 (master drain without ledger) | **Resolved for the sequential case** | `multi.go:1070`; `TestCobolFindings_MasterDrainClosesBeforeLaterEdit`. The concurrent interleaving on writer-free master paths remains and is reported as **WR-01** above. |
| WR-03 (reference form not in evidence) | **Resolved** | Six-field evidence at `cobol_findings.go:455-458`; `TestCobolReconcile_ExternalReferenceFormEvidence`, `TestCobolFindings_ReferenceFormEditSupersedes`. `FindingEvidenceRule` is `/2` (`finding.go:33`). The mixed-form same-line window is documented (see IN-05). |
| WR-04 (unkeyed digest) | **Accepted / documented (D-22)** | The doc comment is corrected at `finding.go:62-68`. The residual risk is documented in `docs/languages.md:279` ("anyone holding the graph or a Neo4j projection can confirm a guessed short evidence value..."). |
| WR-05 (placeholder / file collision) | **Resolved for missing-artifact placeholders** | Grammar `cobol_findings.go:84`; namespace-checked candidates `:193`; by-ID kind-guarded sweep `:366-374`; refusal `cobol_references.go:327-345`; `TestCobolFindings_PlaceholderNamespaceNoFileCollision`, `TestCobolReconcile_PlaceholderSweepByID`, `_PlaceholderRefusedOverRealNode`. Shared DB2/IDMS/CICS resources keep the old shape (`cobol_resources.go:42`, `:77`), as recorded in `deferred-items.md` (05-16). The new parsing introduces **CR-01** above. |
| WR-06 (same-line duplicates collapse) | **Resolved** | `cobol_findings.go:236-255`; `TestCobolReconcile_ExternalSameLineOccurrences`, `TestCobolFindings_SameLineCallsPerOccurrence`. The imageless-edge residual is **WR-02** above. |
| WR-07 (document scope follows program count) | **Resolved** | `cobol_grammar_findings.go:237-239` (Container < 0 goes to the file); `TestCobolGrammar_Findings_DocumentScopeAnchorsToFile`, `TestCobolFindings_SecondProgramKeepsDocumentFinding`. |
| WR-08 (visibility-hidden programs reported as not obtained) | **Out of scope** | Per CONTEXT gap-closure amendments; revisit in Phase 6. |
| IN-01 (v23 migration test cannot fail) | **Out of scope** | Per CONTEXT. |
| IN-02 (history generation checklist skipped) | **Out of scope** | Per CONTEXT. |
| IN-03 (evicted IDs recorded regardless of counts) | **Out of scope** | Per CONTEXT. The same pattern now also covers placeholders (`cobol_findings.go:368-369`). |
| IN-04 (per-indexer FTS routing) | **Out of scope** | Per CONTEXT; `cobol_reconcile.go:88` is unchanged. |
| IN-05 (adaptive node counts in extractor tests) | **Out of scope** | Per CONTEXT. |
| IN-06 (`LOAD` missing from docs) | **Out of scope; incidentally resolved** | `docs/languages.md:279` now lists `LOAD`. |
| IN-07 (gofmt churn) | **Out of scope** | Per CONTEXT. |

## Verification Notes

- **Locking audit:**
  - Lock order observed everywhere: batch gate, reach writer, store `ResolveMutex` (`r.mu`), `cobolReconcileMu`, `findingLedgerMu`. `cobolFindingMu` is a leaf, taken before `findingLedgerMu` and never nested with it.
  - Every `reconcileCobolRepos` caller is writer-free: `EndBatch`, `ResetBatch`, `RefreshRepoConfigs`, and the `TrackRepoCtx` tail outside a batch.
  - Every held-variant caller holds the writer: incremental lanes, the warm closure, untrack's refresh closure, and the IndexAll pre-enrich and cold deferred tail.
  - `TrackRepoCtx` callers (MCP, auto-index, checkout promote and register, the `ReconcileRepoCtx` fallback) hold no writer.
- **Probes** (overlay only, `GOWORK=off go test -count=1 -timeout 300s -run TestReviewProbe_ ./internal/resolver/`):
  - `IDMSColonRepo`: **FAIL** (CR-01).
  - `NoImageOccurrence`: shows the relabel (WR-02).
  - `OldStoreExternal`: shows the duplicates (IN-01).
- No repository file was modified, and nothing was committed.

---

_Reviewed: 2026-10-26T20:46:09Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
