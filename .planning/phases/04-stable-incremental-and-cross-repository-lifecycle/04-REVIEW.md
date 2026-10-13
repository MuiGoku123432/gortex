---
phase: 04-stable-incremental-and-cross-repository-lifecycle
reviewed: 2026-10-13T00:00:00Z
depth: standard
files_reviewed: 77
files_reviewed_list:
  - cmd/gortex/daemon_state.go
  - cmd/gortex/mcp.go
  - internal/config/config.go
  - internal/config/manager.go
  - internal/graph/edge.go
  - internal/graph/prefix_diagnostics.go
  - internal/graph/scope.go
  - internal/graph/scoped_projection.go
  - internal/graph/store_sqlite/migration_unprefixed_purge.go
  - internal/graph/store_sqlite/prefix_diagnostics.go
  - internal/graph/store_sqlite/schema_version.go
  - internal/graph/store_sqlite/scoped_projection.go
  - internal/graph/storetest/storetest.go
  - internal/indexer/affected_by.go
  - internal/indexer/cobol_material.go
  - internal/indexer/cobol_reconcile.go
  - internal/indexer/extractor_version.go
  - internal/indexer/incremental_batch.go
  - internal/indexer/incremental_watcher_batch.go
  - internal/indexer/index_state.go
  - internal/indexer/indexer.go
  - internal/indexer/merkle_reindex.go
  - internal/indexer/multi.go
  - internal/indexer/source_revision.go
  - internal/indexer/workspace_resolve.go
  - internal/mcp/change_contract.go
  - internal/mcp/overlay.go
  - internal/mcp/source_scope.go
  - internal/mcp/tools_coding.go
  - internal/mcp/tools_enhancements.go
  - internal/mcp/tools_fileops.go
  - internal/neo4jprojection/service.go
  - internal/parser/languages/cobol_grammar_identity.go
  - internal/parser/languages/cobol_grammar_mainframe.go
  - internal/parser/languages/cobol_grammar_refs.go
  - internal/parser/languages/cobol_grammar_sql.go
  - internal/parser/languages/cobol_grammar_walk.go
  - internal/parser/languages/cobol_grammar.go
  - internal/resolver/cobol_policy.go
  - internal/resolver/cobol_references.go
  - internal/resolver/cobol_resources.go
  - internal/resolver/cross_repo.go
  - internal/resolver/resolver.go
  - internal/testutil/graphdump/graphdump.go
  - cmd/gortex/cobol_breadth_test.go
  - cmd/gortex/cobol_scope_test.go
  - cmd/gortex/cobol_tracer_test.go
  - cmd/gortex/mcp_index_test.go
  - internal/config/cobol_config_test.go
  - internal/graph/prefix_diagnostics_test.go
  - internal/graph/scope_test.go
  - internal/graph/store_sqlite/coverage_spelling_purge_test.go
  - internal/graph/store_sqlite/migration_unprefixed_purge_test.go
  - internal/indexer/affected_by_e2e_test.go
  - internal/indexer/cobol_lifecycle_acceptance_test.go
  - internal/indexer/cobol_lifecycle_crossrepo_test.go
  - internal/indexer/cobol_lifecycle_incremental_test.go
  - internal/indexer/cobol_lifecycle_material_test.go
  - internal/indexer/cobol_lifecycle_test.go
  - internal/indexer/cobol_material_test.go
  - internal/mcp/index_health_ownership_test.go
  - internal/mcp/index_health_path_liveness_test.go
  - internal/mcp/source_scope_test.go
  - internal/mcp/tools_fileops_worktree_test.go
  - internal/neo4jprojection/normalize_scope_test.go
  - internal/neo4jprojection/properties_phase4_test.go
  - internal/parser/languages/cobol_grammar_contract_test.go
  - internal/parser/languages/cobol_grammar_identity_test.go
  - internal/parser/languages/cobol_grammar_mainframe_test.go
  - internal/parser/languages/cobol_grammar_material_test.go
  - internal/parser/languages/cobol_grammar_resource_test.go
  - internal/parser/languages/cobol_grammar_sql_test.go
  - internal/parser/languages/cobol_grammar_test.go
  - internal/resolver/cobol_reconcile_test.go
  - internal/resolver/cobol_resources_test.go
  - internal/resolver/cross_repo_test.go
  - internal/testutil/graphdump/graphdump_test.go
findings:
  critical: 3
  warning: 4
  info: 6
  total: 13
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-10-13T00:00:00Z
**Depth:** standard
**Files Reviewed:** 77 (44 source, 33 test; tests reviewed for reliability only)
**Status:** issues_found

## Narrative Findings (AI reviewer)

## Summary

The review covered the Phase 4 changes (`git diff bf119200^..HEAD`) to the source gate, scope type, ownership audit and v22 migration, COBOL reconcile and policy, material fingerprint, D-12 cross-repo ambiguity, D-17 re-stamp, the embedded single-repo path, and the dump helper. Where the risk needed it, call chains were followed into `scope_resolve.go`, `guardSymlinkWithinRepo`, `tools_search_text.go`, `tools_find_declaration.go`, `repository_untrack.go`, `store_purge.go`, `checkout_lifecycle.go`, and the daemon warmup loop.

The COBOL decision logic in `decideCobol` and `cobolLibraryRank` holds up. It is deterministic: the candidate union is sorted by `Scope`, ambiguous IDs are sorted, and the same decision is never written twice. `pickCrossRepoCandidate` also detects multi-repo ambiguity correctly. The problems are at the edges of the phase:

- **The fail-closed source gate (SEC-02) has two bypasses.**
  - It assigns a path to a repository by its lexical repo prefix, but path confinement allows a path inside any tracked root. A symlink or a nested checkout therefore reads or writes another repository's bytes.
  - Two tools return source lines from the trigram index and never reach the gate's choke points.
- **Untracking a repository deletes other repositories' COBOL CALL/COPY edges that were bound into it.** It does not return them to their placeholders, so the incremental result differs from a clean build and the edges are permanently lost.
- **The COBOL reconcile has no serialization across passes,** although the codebase serializes the analogous contract reconcile.

The four items already in `deferred-items.md` are not repeated here: the stale generic ambiguity stamp, remote stitch after a refusal, the stale `cross_repo_calls` parallel edge, and the resolved `{}`/`null` dump item.

## Critical Issues

### CR-01: The source gate assigns a repository from the lexical path while confinement allows any tracked root, so a symlink or nested checkout bypasses the scope

**File:** `internal/mcp/tools_fileops.go:72-86` (with `resolveConfinedFilePath` :93-107, `guardRepoRoots` :468-490), `internal/mcp/source_scope.go:60-96`

**Issue:** `resolveFilePath` decides which repository owns a request by calling `matchedRepoPrefix(s.multiIndexer, relPath)`. Here `relPath` is the caller's lexical, repo-prefixed spelling: `resolveConfinedFilePath` returns `rawPath` verbatim at line 95 of the body. The only check on the symlink-resolved location is `guardSymlinkWithinRepo`, and it accepts any path inside **any** tracked root (`guardRepoRoots` collects every `RepoPrefixes()` root). This gives two bypasses:

- **Symlink.** Repo `alpha` contains `vendor/shared -> ../../beta/src`. A session scoped to `alpha` (or a request with `repo:"alpha"`) calls `read_file`, `edit_file`, or `write_file` on `alpha/vendor/shared/x.go`. The gate checks `alpha`, which is allowed, and the confinement guard sees that the real path lies inside beta's root, which is also allowed. Beta's bytes are returned or overwritten.
- **Nested checkouts.** Beta's root is a subdirectory of alpha's root, and the two have different prefix names. The path `alpha/<subdir>/beta-file` is attributed to `alpha`.

Either way a caller reads or writes source in a repository outside its resolved scope. That breaks D-13/D-20 and the file's own claim that "a repository outside the resolved scope answers exactly like a missing file". Cross-repo symlinks to shared copybook directories are a realistic layout in a mainframe estate.

**Fix:** Assign ownership from the resolved real path, and require both the lexical repository and the physical repository to be in scope:
```go
absPath, relPath, err = s.resolveConfinedFilePath(ctx, rawPath)
if err != nil || s.multiIndexer == nil {
    return absPath, relPath, err
}
lexical := matchedRepoPrefix(s.multiIndexer, relPath)
if view := requestViewPathRoot(ctx); lexical == "" && view.contains(absPath) {
    lexical = view.repoPrefix
}
real, rerr := filepath.EvalSymlinks(absPath)
if rerr != nil {
    real = resolveNearestExistingAncestor(absPath)
}
physical := s.multiIndexer.RepoForFile(real) // longest tracked root containing the real path
for _, repo := range []string{lexical, physical} {
    if err := s.requireSourceScope(ctx, repo); err != nil {
        return "", "", err
    }
}
```
Apply the same treatment to the absolute-path arm of `resolveGraphPath` (:588-598). Add a `source_scope_test` case with a symlink from alpha into beta.

### CR-02: `search_text` and `find_declaration` return source lines from every tracked repository with no scope

**File:** `internal/mcp/tools_search_text.go:77-107`, `internal/mcp/tools_find_declaration.go:160-164,120`

**Issue:** D-13 requires that any tool returning source bytes refuse an unbound multi-repo request. 04-RESEARCH explicitly listed "`search_text` match lines" as covered by the choke points. They are not covered: both tools read `trigram.Match.Text`, the matching source line (`internal/search/trigram/searcher.go:20`), straight from the trigram index and never call `resolveFilePath` or `resolveNodePath`.

- **`search_text`.** For an unbound session `resolved.RepoAllow` is nil, so `scopedMultiGrep` is false and the call uses `s.multiIndexer.GrepText(query, limit)` across every repository. `filterTextMatchesByResolvedScope` keeps everything. An unbound caller can therefore read arbitrary source in every tracked repo, line by line. A regexp of `.` with a high `limit` dumps whole files.
- **`find_declaration`.** It calls `s.multiIndexer.GrepText(useSite, 0)` with no scope resolution at all, so even a bound session gets `use_sites[].text` from repositories outside its workspace.

`TestSourceScope_UnboundMultiRepoRefused` does not include either tool, which is why the gap passed.

**Fix:** Gate both tools on the shared rule before returning `Text`. The smallest change keeps match metadata and refuses or strips text:
```go
// search_text, after filterTextMatchesByResolvedScope:
if s.multiIndexer != nil && len(s.multiIndexer.RepoPrefixes()) > 1 {
    kept := matches[:0]
    for _, m := range matches {
        err := s.requireSourceScope(ctx, matchedRepoPrefix(s.multiIndexer, m.Path))
        if errors.Is(err, errSourceScopeRequired) {
            return mcp.NewToolResultError(err.Error()), nil
        }
        if err == nil {
            kept = append(kept, m)
        }
    }
    matches = kept
}
```
Apply the same filter to `find_declaration`'s `raw` before grouping. Add both tools to the unbound-refusal inventory test.

### CR-03: Untracking a repository deletes other repositories' bound COBOL CALL/COPY edges instead of returning them to their placeholders

**File:** `internal/indexer/repository_untrack.go:205-213` → `internal/graph/store_sqlite/store_purge.go:113`; interacts with `internal/resolver/cobol_references.go:115-276`

**Issue:** D-09 lets repository A bind COBOL CALL/COPY edges into an allowed repository B. When B is untracked, `purgeRepositoryPayload` calls `PurgeRepo(B)`, which runs `deleteByIDColumnsTx(tx, "edges", []string{"from_id", "to_id"}, ids)`. That deletes every edge whose **target** is a B node, including A's statement-occurrence edges bound across repositories.

File eviction does not have this problem: `restubIncomingRefs` and `restubIncomingRefsFromView` put the edge back on its `cobol_placeholder`. Untrack has no equivalent step and runs no reconcile for A. As a result:

- A's CALL/COPY statement loses its relationship edge entirely, contrary to ID-05 (every relationship edge points back to its statement occurrence).
- A clean build of A without B tracked keeps the edge on its placeholder with `cobol_unresolved_reason=missing`, so incremental is no longer equal to clean (ID-09).
- Re-tracking B later cannot rebind the edge. `reconcileCobolRepos({B})` expands to A but finds no edge to decide. The loss persists until A's file is re-extracted.

The 04-15 deferred item only covers the stale parallel edge on unbind, not deletion of the primary edge.

**Fix:** Before `PurgeRepo`, return inbound COBOL edges from other repositories to their placeholders, then reconcile the dependents after the purge:
```go
// in purgeRepositoryPayload, before PurgeRepo(repoPrefix):
ids := /* B's COBOL program and copybook node IDs */
var batch []graph.EdgeReindex
for _, edges := range mi.graph.GetInEdgesByNodeIDs(ids) {
    for _, e := range edges {
        ph, _ := e.Meta["cobol_placeholder"].(string)
        if ph == "" || graph.RepoPrefixOfID(e.From) == repoPrefix || e.To == ph {
            continue
        }
        old := e.To
        e.To, e.CrossRepo = ph, false
        batch = append(batch, graph.EdgeReindex{Edge: e, OldTo: old})
    }
}
mi.graph.ReindexEdges(batch)
// after the purge and detach:
mi.reconcileCobolRepos(map[string]struct{}{repoPrefix: {}}) // expands to every repo that resolves from B
```
Add a D-04 sequence: track A and B, bind A→B, untrack B, then compare with a clean build of A.

## Warnings

### WR-01: COBOL reconcile passes are not serialized against each other, and the warm-restart pass ignores `deferResolve` during parallel warmup

**File:** `internal/indexer/cobol_reconcile.go:76-82`, `internal/indexer/multi.go:3249-3253`, `internal/indexer/incremental_watcher_batch.go:428-438`, `internal/resolver/cobol_references.go:48-52`

**Issue:** `ReconcileCobol` protects itself with `r.mu`, but `reconcileCobolRepos` builds a **new** master resolver (`mi.newMasterResolver(false)`) on every call, so `r.mu` excludes nothing across callers. Callers run concurrently:

- the per-repository warm-restart path `ReconcileRepoCtx`, which runs inside the parallel warmup batch at `cmd/gortex/daemon_state.go:705`;
- the watcher's `resolveIncrementalRepoMutationMode` tail for each repository;
- each per-repository resolver's `ResolveAll`.

Each pass grows its repository set by the dependents (`cobolReconcileRepos`), so two passes routinely decide the **same** edges. Each runs read → decide → `ReindexEdges`/`PersistEdgeAttributesBatch` from its own snapshot, and `reconcileCobolResources` can `EvictFile` a shared resource that another pass is about to link. On the in-memory store the passes also write `e.To`/`e.Meta` on live shared edges at the same time. The codebase already solved this exact shape for contract edges with `mi.reconcileMu` ("two overlapping reconciles can interleave evict and mint and persist a stale generation").

Separately, the new unconditional call at `multi.go:3253` runs even when `batchMode.deferResolve` is set. The neighbouring tails deliberately defer under that flag. During parallel warmup it decides A's edges while B is not yet registered: `tracked(B)` is false, so A's binding into B is rewritten to `missing`, and the result depends on registration order.

**Fix:** Add a dedicated MultiIndexer mutex and take it in `reconcileCobolRepos`, matching `reconcileMu`. Skip the warm-restart call when `batchMode.deferResolve` is set and run one COBOL reconcile in the global resolve phase instead:
```go
func (mi *MultiIndexer) reconcileCobolRepos(repos map[string]struct{}) {
    mi.cobolReconcileMu.Lock()
    defer mi.cobolReconcileMu.Unlock()
    if master := mi.newMasterResolver(false); master != nil {
        master.ReconcileCobol(repos)
    }
}
```

### WR-02: The v22 migration's sidecar sweep deletes every empty-prefix sidecar row on every v21 store, which is broader than the node predicate it claims to mirror

**File:** `internal/graph/store_sqlite/migration_unprefixed_purge.go:81-85`, `internal/graph/store_sqlite/schema_version.go:121-125`

**Issue:** 04-05 narrowed the node delete to "the audit's unowned file-backed rows" because v22 runs on every current store, where `external::`-path modules, rationale nodes, and contracts legitimately carry `repo_prefix=''`. The sidecar loop was not narrowed. It still runs `DELETE FROM <table> WHERE repo_prefix = ''` over all 19 `rekeyMoveTables` and `rekeyDropTables` entries (`repo_index_state`, `contract_state`, `enrichment_state`, `symbol_fts`/`symbol_fts_rowid`, `ref_facts`, `constant_values`, `semantic_binding_types`, `file_index_failures`, …), whether or not any node was purged.

Sidecar rows that belong to the empty-prefix population the migration promises to keep are therefore deleted on every v21 store. The only persistent stores v22 actually needs to clean are the few that held embedded unprefixed rows. The migration cannot be undone, and `TestMigrationV22_PurgesUnprefixedRows` asserts node survival but not sidecar survival.

**Fix:** Sweep sidecars only when the node delete actually removed unowned file-backed rows, and assert sidecar survival in the test:
```go
res, err := tx.Exec(`DELETE FROM nodes WHERE ` + unprefixedFileBackedNodePredicate)
if err != nil { return fmt.Errorf("delete unprefixed nodes: %w", err) }
if n, _ := res.RowsAffected(); n == 0 {
    return nil // a clean current store: leave legitimately empty-prefix sidecars alone
}
for _, table := range ... { /* existing sweep */ }
```

### WR-03: `daemon reload` changes the live COBOL policy, but nothing restages or reconciles existing bindings

**File:** `internal/indexer/multi.go:2462-2492`, `internal/indexer/checkout_lifecycle.go:1151`, `internal/indexer/workspace_resolve.go:720-757`

**Issue:** `RefreshRepoConfigs` now calls `idx.SetCobolConfig`, and `cobolPolicyLookup` reads `configMgr.GetRepoConfig` on every pass, so the new `index.cobol` policy takes effect at once for the **next** reconcile. Neither `ApplyReload` nor `realController.Reload` triggers that reconcile or the D-06 fingerprint restage. Existing bindings keep reflecting the old allowlist, library order, or default qualifiers until one of two things happens:

- an unrelated COBOL mutation in that repository re-decides every edge, under a policy the user never saw applied; or
- a daemon restart compares `_cobol_material`.

The material-config acceptance test hides this because it calls `IncrementalReindexRepo(prefix, nil)` by hand after `RefreshRepoConfigs` (04-RESEARCH Pitfall 6).

**Fix:** In `RefreshRepoConfigs`, record the prefixes whose `cobolMaterialFingerprint` changed, and after the loop run `mi.reconcileCobolRepos(changed)`. Alternatively, schedule the same full-root incremental reindex the harness uses, so the stored `_cobol_material` is also re-persisted.

### WR-04: The `rename_symbol` refusal reveals that out-of-scope references exist

**File:** `internal/mcp/tools_coding.go:2860-2866`

**Issue:** `source_scope.go` promises that an out-of-scope refusal "does not reveal whether the target exists". `rename_symbol` returns a distinct message when any reference falls outside scope: "some references lie outside the resolved scope; widen it with repo, project, or workspace". A caller scoped to `alpha` can rename an `alpha` symbol with `dry_run:true` and learn whether any other repository references it. That is a cross-repository existence oracle, and the gate exists to prevent it.

**Fix:** Refusing the rename is right; the message is not. Return text that is not specific to whether references exist, for example the generic `errSourceScopeRequired` guidance ("pass repo, project, or workspace…"). Alternatively, return the same text that `get_symbol_source` uses for an out-of-scope target.

## Info

### IN-01: `cobolPolicyLookup` logs a Warn for each repository on every reconcile pass

**File:** `internal/indexer/workspace_resolve.go:748-754`

**Issue:** The lookup runs once per repository per pass, and every COBOL watcher mutation and warm restart is a pass. A repository whose `resolve_from` names an untracked repository therefore logs the same Warn on every save.

**Fix:** Log once per (repo, untracked-set) change, for example with a `sync.Map` keyed by `repo+"\x00"+strings.Join(dropped, ",")`, or demote repeats to Debug.

### IN-02: Untrimmed `lib.Repo`/`lib.Path` in `cobolLibraryRank` disagree with the trimmed validation and tracking checks

**File:** `internal/resolver/cobol_references.go:407-411`

**Issue:** `CobolConfig.Validate` and `cobolPolicyLookup` trim `lib.Repo`, but `cobolLibraryRank` compares the raw `lib.Repo` with `c.RepoPrefix`, and uses `path.Clean(lib.Path)` without `filepath.ToSlash`. On Windows that means backslashes. A quoted `" shared"` or `copy\prod` passes validation and then never ranks, so every COPY silently resolves to `missing`.

**Fix:** Normalize once in `cobolPolicyFromConfig`: `Repo: strings.TrimSpace(lib.Repo)` and `Path: path.Clean(filepath.ToSlash(strings.TrimSpace(lib.Path)))`.

### IN-03: The material fingerprint truncates on 32-bit builds and depends on list order

**File:** `internal/indexer/cobol_material.go:38-42,68`

**Issue:** `int(fp)` truncates a 48-bit value where `int` is 32 bits, so different fingerprints can collide. Separately, `json.Marshal(cfg)` makes `resolve_from: [B, C]` and `[C, B]` produce different fingerprints, which spuriously restages every COBOL file. `resolve_from` is a set; library order is meaningful and should stay ordered.

**Fix:** Sort a clone of `ResolveFrom` before hashing. Store the fingerprint as a JSON string, or keep it in `int64`, if 32-bit targets matter.

### IN-04: The canonical dump omits edge and node fields that the reconcile pass mutates

**File:** `internal/testutil/graphdump/graphdump.go:101,115`

**Issue:** Edge rows omit `Origin`, `CrossRepo`, and `Confidence`, and node rows omit `RepoPrefix`, `WorkspaceID`, `ProjectID`, and `Language`. `reconcileCobol` writes `e.Origin` and `e.CrossRepo`, and the restub paths set `To` without resetting `CrossRepo`. A history-dependent value in those fields would pass the incremental-equals-clean proof. This is D-01-compliant, so it is not a defect in the decision, but it is a blind spot in the proof.

**Fix:** Append `Origin`, `CrossRepo`, and `RepoPrefix` to the rows under a new `graphdump-volatile/4` version.

### IN-05: Bare unprefixed `Indexer` paths no longer bind any COBOL CALL or COPY

**File:** `internal/indexer/indexer.go:598-603`, `internal/resolver/cobol_references.go:305-308`

**Issue:** `decideCobol` now returns `caller_unowned` whenever the caller has no repo prefix. Before Phase 4, an unprefixed caller matched unprefixed candidates. `gortex eval-server --index` (`cmd/gortex/eval_server.go:66`) and the `init` temporary store (`cmd/gortex/init.go:493`) still index through a bare `indexer.New`, so every literal CALL and COPY there now stays unbound. This is consistent with D-14 and D-16, but the change is silent.

**Fix:** Route those paths through `indexEmbeddedRepo` / MultiIndexer the way `gortex mcp` now does, or document the behavior.

### IN-06: `get_editing_context` gates the anchor's repository but reads view bytes from the view's repository

**File:** `internal/mcp/tools_coding.go:571-590`

**Issue:** The `compress_bodies` path checks `anchor.RepoPrefix` but then reads through `refViewFilesFor(ctx)` / `readViewFile`, which serves `files.repoPrefix`. `handleReadFile` (`tools_fileops.go:1447`) gates `files.repoPrefix`. The two surfaces apply different keys to the same committed-view read.

**Fix:** In the view branch, also call `s.requireSourceScope(ctx, files.repoPrefix)` before `readViewFile`.

---

_Reviewed: 2026-10-13T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
