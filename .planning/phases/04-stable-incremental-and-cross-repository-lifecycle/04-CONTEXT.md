# Phase 4: Stable Incremental and Cross-Repository Lifecycle - Context

**Gathered:** 2026-10-12
**Status:** Ready for planning

<domain>
## Phase Boundary

For identical final inputs, users get the same canonical, scoped, deterministic graph, whether it came from reruns, edits, deletes, restores, version or config changes, or a clean rebuild. Stale owned observations are removed when source, parser, extractor, retrieval, or material configuration changes. Cross-repository resolution binds only inside explicitly allowed scope with deterministic evidence. Source-reading surfaces fail closed on missing scope. Ownership health reports zero unowned or misprefixed nodes for the acceptance corpus.

Requirements: ID-03..ID-09, SEC-01, SEC-02, SEC-03, SEC-08.

Out of this phase:
- Finding nodes and finding lifecycle (Phase 5).
- Native query answers, monitoring, and the pre-AI release gate (Phase 6).
- AI context enforcement beyond the shared scope check it will reuse (Phase 7).

Depends on Phase 3, which is currently in verification (03-17 complete).

</domain>

<decisions>
## Implementation Decisions

### Carried forward (do not re-decide)
- Identity shapes are fixed:
  - Programs: `<repo>/<path>::OUTER/INNER#k`. Line numbers never discriminate (P2 D-10).
  - Data items: the full group path from level 01, with `#k` only for true duplicates and FILLERs (P3 D-07).
  - Resources: repo-scoped, with explicit `_unqualified` markers (P3 D-15).
  - Phase 4 makes these identities stable and proves them. It does not redesign them. ID-03 and ID-04 are mostly satisfied already and need proof plus gap fixes.
- Every relationship edge points back to its statement occurrence through a range plus the `KindLocal` statement node (P3 D-05/D-17). This is the basis for ID-05.
- CALL and COPY bind only within the same repo, and only on an exact single match. Unbound references carry `missing` / `ambiguous` reasons with candidate IDs. Names are normalized by `cobol-name-norm/1` (P3 D-09/D-11/D-18). `COPY IDMS` members are never bound (P3 D-19).
- Provenance lives in flat `prov_*` meta, with no new node fields or SQLite columns (P2 D-08). Prefer additive packages over invasive edits to upstream code (fork hygiene).
- Fixtures are invented COBOL only, because the repo is public (P2 D-14).

### Equivalence proof
- **D-01:** "Canonical active deterministic subgraph" means:
  - Nodes: ID, kind, name, file path, and range.
  - Edges: the (from, to, kind) triple plus edge file and line.
  - All deterministic meta (`cobol_*`, `prov_*`, resolution reasons).
  - Excluded: an **explicit, versioned allowlist of volatile keys**, such as index timestamps, mtimes, and cache-only fingerprints. Every excluded key must be justified in the allowlist.
- **D-02:** Equality uses a **canonical, sorted, deterministic dump** of the scoped active subgraph, plus a digest of that dump. Tests diff the incremental and clean-rebuild dumps and print the first differing rows, so a failure is diagnosable.
- **D-03:** The dump and digest are a **test-only or internal helper** in this phase. There is no CLI or MCP verb. Phase 6 may promote it for monitoring.
- **D-04 (Claude's choice, user delegated):** The acceptance harness proves incremental == clean rebuild for all of these sequences:
  - Identical rerun, edit, delete, and restore. Restore must reproduce the identical IDs.
  - Extractor-version bump, parser-identity change, and material-config change with unchanged bytes.
  - Cross-file: a change in file B re-resolves CALL/COPY edges from unchanged file A.
  - Cross-repo: a multi-repo fixture with same-named programs and copybooks that must not bind without an allowlist.
  - An interrupted reindex followed by a retry.
  - A change whose affected set exceeds the 200-file affected-by cap.
  - The harness runs in-process on the real path (registered extractor → indexer → SQLite temp store), under `go test -race`, following the P2 D-12 pattern.

### Invalidation and stale facts
- **D-05:** **Hard-evict.** Keep today's `file_path` eviction: stale nodes and edges owned by a changed or deleted file are removed. "Restore" re-emits the same IDs because identity is derived from content and position, not history. There are no tombstones or inactive rows in the node and edge store. Phase 5 adds lifecycle status only for findings. — **Reversibility:** costly. Adding tombstones later means every query, projection, and analyzer must filter inactive rows.
- **D-06:** **Material configuration is an explicit per-extractor fingerprint.** Each extractor declares the config inputs it actually reads. For COBOL that includes:
  - the approved grammar ID and preprocessor module pin
  - the name-norm rule ID
  - the copybook library search order (D-10)
  - the cross-repo allowlist (D-09)
  - the default-qualifier config (D-11)
  - any retrieval manifest digest

  Their hash is salted into that language's Merkle salt and persisted in `RepoIndexState`, so unrelated config edits don't trigger a reindex. The whole `indexConfigHash` is **not** used as a salt.
- **D-07:** **Version staleness is derived from the stamped identity.** The COBOL invalidation salt is computed from the same values written to `prov_extractor_version` and the parser tool/config IDs, so the stamp and the trigger cannot drift. The hand-bumped `extractorVersions` map stays for other languages.
- **D-08:** **When the affected-by set exceeds the cap, escalate to a full resolve** of the affected scope instead of silently truncating at `defaultAffectedByMax = 200`. Log at Info with counts. Correctness wins over speed.

### Cross-repo resolution domains
- **D-09:** Explicit allowed scope is a **directed allowlist in workspace config** (for example, repo A may resolve COBOL CALL/COPY against repos B and C). With no entry, resolution is same-repo only. Binding across repos still requires an exact single match within the allowed set. More than one candidate means `ambiguous` with candidate IDs, and nothing is picked. The allowlist is part of the COBOL material fingerprint (D-06), so editing it invalidates bindings.
- **D-10:** **Copybook library search order is an explicit ordered library list per repo in config**, like a SYSLIB concatenation. The first match by order wins and records its search rank plus a versioned rule ID in meta. With no list configured, the P3 exactly-one-match rule stays.
- **D-11:** **Qualified and `_unqualified` resources never merge.** Both IDs stay distinct and stable. When config declares a default schema or subschema for a repo, emit a deterministic link edge from the `_unqualified` node to the qualified node, carrying the rule ID. IDs are never rewritten, so a config change never renames nodes. Claude picks the edge kind, reusing an existing kind. — **Reversibility:** costly. IDs are persisted and projected.
- **D-12:** **Generic cross-repo first-match becomes "ambiguous → refuse" for all languages.** `CrossRepoResolver.resolveFunctionCall` currently takes the first match from a Go map iteration (`cross_repo.go:1038`), which is nondeterministic. When more than one reachable repo matches, bind nothing and record reason `ambiguous` with candidate IDs, mirroring the COBOL rule. This is an accepted behavior change to upstream code, scoped to the minimal edit.

### Scope and ownership
- **D-13:** **SEC-02 fails closed for source reads only.** When more than one repo is tracked, tools that return source bytes require a resolved scope. These include `read_file`, `get_symbol_source`, editing/smart context source payloads, and the future Phase 7 AI context path. Graph-metadata queries keep today's all-repos default for unbound sessions.
- **D-14:** **Single-repo mode is treated as a workspace of one.** Every node gets an owning repo prefix, and `PrefixDiagnostics` counts any unowned or misprefixed node as a violation. The `Clean()` path that accepts a fully unowned graph goes away for this purpose. Unresolved placeholders are exempt only through an explicit, enumerated namespace list. — **Reversibility:** costly. Changing ID prefixes for single-repo stores means a reindex, and Neo4j reprojection, of existing single-repo stores.
- **D-15:** **SEC-08 enforcement is a test gate plus health status.** The acceptance test fails on any ownership violation. `index_health` reports an unhealthy status with counts but does not block normal queries. Phases 6 and 7 may make export or AI refuse when unhealthy.
- **D-16:** **One shared scope check.** Generalize the Phase 1 `OpenScopedProjectionSnapshot` fail-closed validation into one scope type and validator used by projection, resolution, and source reads. The rule of three is now met. A cross-surface test asserts that the same repo allow-set yields identical node sets on each surface.

### Post-research confirmations (2026-10-12, user-confirmed after 04-RESEARCH.md)
- **D-17:** **When git HEAD moves with unchanged bytes, re-stamp `prov_vcs_commit`.** Treat it as a metadata-only refresh, with no re-extraction. `prov_vcs_commit` stays deterministic provenance and **is compared** in the canonical dump. It is not on the volatile allowlist.
- **D-18:** **A shared DB2/IDMS/CICS resource node's Name is the normalized key** under `cobol-name-norm/1`, so it doesn't depend on which file wrote it. Each referencing edge keeps the as-written spelling in meta. This narrows the P2 D-20 "names as written" rule for shared resources only. — **Reversibility:** costly. Names are persisted and projected.
- **D-19:** **D-09 candidates are the union of same-repo and allowed-repo candidates.** More than one candidate across that union means `ambiguous` with candidate IDs, and nothing binds. A same-repo candidate never silently shadows one in an allowed repo.
- **D-20:** **The D-13 scope gate also covers write tools** (`edit_file` / `write_file` and any tool sharing `resolveFilePath`). Any tool that reads or writes source bytes requires a resolved scope when more than one repo is tracked.

### Claude's Discretion
- Research recommendations adopted by default, unless the planner finds a blocker:
  - Fix CR-02 shared-resource ownership first.
  - One `reconcileCobol(repoSet)` used by both full and incremental paths.
  - `cobol_placeholder` meta on bound edges.
  - `MaterialIdentity()` extractor interface.
  - `index.cobol` config namespace.
  - `ProjectionScope` → `Scope` with an alias.
  - `EdgeAliases` for D-11.
  - The D-14 rollout reuses `purgeUnprefixedRepoRows` as migration v22.
  - Embedded single-repo mode goes through MultiIndexer, after measuring its blast radius.
  - The ID-09 equivalence corpus is limited to COBOL. Already-bound generic cross-repo edges are not re-checked incrementally.
  - The D-10 library entry shape and how a `COPY OF/IN` qualifier narrows the search follow the research recommendation.
- Incremental COBOL resolution: today `resolveCobolReferences` runs only inside whole-graph `ResolveAll`, and incremental paths skip `cobol::` targets. D-04 requires incremental == clean, so the planner must make the incremental and watcher paths re-run COBOL CALL/COPY resolution for the affected scope (or escalate per D-08). Research picks the mechanism.
- The exact volatile-key allowlist (D-01), dump format, and digest algorithm. Prefer the existing BLAKE3 dependency.
- Config key names and shape for D-09, D-10, and D-11 in `.gortex.yaml` / workspace config. Follow existing workspace config conventions.
- The edge kind for the D-11 qualified link, reusing an existing kind.
- The enumerated placeholder-exemption namespaces (D-14).
- Rollout of D-14 to existing single-repo stores. This likely uses the existing `storeNeedsRebuild` / schema-version path.
- Plan slicing and wave order.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` §"Phase 4: Stable Incremental and Cross-Repository Lifecycle": goal and the 5 success criteria.
- `.planning/REQUIREMENTS.md`: ID-03..ID-09, SEC-01, SEC-02, SEC-03, SEC-08, and the Out of Scope list.
- `.planning/PROJECT.md` §Key Decisions: SQLite is the sole authority, fork hygiene, and the parser pin.

### Prior phase decisions (identity, provenance, resolution)
- `.planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-CONTEXT.md`: D-08 (provenance meta), D-09 (revision), D-10 (identity), D-12 (in-process acceptance pattern), D-14 (invented fixtures).
- `.planning/phases/03-deterministic-cobol-and-mainframe-breadth/03-CONTEXT.md`: D-07 (data-item IDs), D-09/D-11 (CALL/COPY binding), D-15 (resource IDs), D-17/D-18/D-19, and the deferred items this phase picks up.
- `.planning/phases/01-general-neo4j-projection-command/01-CONTEXT.md`: fail-closed scope and generation-owned reconciliation in the projection.

### Codebase maps
- `.planning/codebase/ARCHITECTURE.md`: the indexer, resolver, and graph layers, and the incremental reindex error handling.
- `.planning/codebase/CONCERNS.md`: resolver panics on scope mismatch (`framework_scoped_store.go`) and goroutine pools in cross-repo resolution.
- `.planning/codebase/TESTING.md`: test conventions (`graph/storetest`, race).

No external specs. The requirements are fully captured in the decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

Scouted 2026-10-12. The graph IDs are under `gortex/...`. A sibling checkout, `GoApps/gortex-phase4-sql`, is also indexed and duplicates many symbols, so don't confuse the two.

### Reusable Assets
- `store_sqlite/scoped_projection.go:22` `OpenScopedProjectionSnapshot`: the fail-closed scope validation to generalize (D-16). It rejects an empty allow-set, `""` entries, or an empty workspace or project.
- `graph/prefix_diagnostics.go:30` `PrefixDiagnostics`: counts unowned and misprefixed nodes and is surfaced in `index_health` (`mcp/tools_enhancements.go:3141`). This is the base for D-14 and D-15.
- `indexer/extractor_version.go`: the `extractorVersions` map, `merkleSaltFor` (:130), and `ExtractorVersionStaleLangs` (:150). This is where the D-06/D-07 salts plug in.
- `graph/index_state.go:13` `RepoIndexState` (IndexedSHA, WorkspaceFP, ExtractorVersions): where the material fingerprint is persisted.
- `indexer/file_delta.go:22,67`: the semantic, metadata, and core fingerprints used for skip-unchanged.
- `indexer/derived_invalidation.go`: `DerivedInvalidationPlan` and `ContractGroupFrontier`, which already isolate by workspace and project.
- `graph/storetest/storetest.go`: store-contract tests (`testReindexEdges` :848, `testPrefixDiagnostics` :204).

### Established Patterns
- Eviction is by `file_path` (`store_sqlite/file_batch_evict.go:13`, `Graph.EvictFile` graph.go:3220). Evicting a file drops every edge with an endpoint in it. Inbound edges are restored by affected-by re-resolution (`affected_by.go:544`) and `ResolveFilesAndIncoming` (`resolver.go:2652`).
- Repo prefixing happens in `Indexer.applyRepoPrefix` (indexer.go:2265). Unresolved targets stay unprefixed. `FindNodesByNameInRepo` deliberately lets empty-prefix nodes through (graph.go:3063).
- Schema rebuilds use `PRAGMA user_version` (`currentSchemaVersion = 21`) plus `storeNeedsRebuild` (cmd/gortex/daemon_state.go:1436).

### Known gaps this phase must close
- `defaultAffectedByMax = 200` silently truncates, logging only at Debug (`affected_by.go:558`). Addressed by D-08.
- `resolveCobolReferences` (`resolver/cobol_references.go:68`) runs only in whole-graph `ResolveAll`. Incremental paths skip `cobol::` targets (resolver.go:3379, cross_repo.go:1121, incremental_resolve.go:453). See Claude's Discretion.
- `prov_extractor_version` is never compared on reindex, and `indexConfigHash` is not part of the primary-index salt. Addressed by D-06 and D-07.
- Cross-repo first-match uses map iteration order (`cross_repo.go:1038`). Addressed by D-12.
- Unbound MCP sessions fail open to all repos (`mcp/tools_core.go:638`, `mcp/scope_resolve.go:117/204`). Addressed by D-13.
- There is no incremental-vs-full equivalence harness. Addressed by D-02 and D-04. Test files are not in the graph index, so confirm this with a grep of `*_test.go`.

### Integration Points
- Incremental entry points:
  - `MultiIndexer.IncrementalReindexRepo` (multi.go:2497)
  - `Indexer.IncrementalReindexPaths` (indexer.go:6014/6119)
  - the watcher executors (`incremental_watcher_batch.go:229-320`)
  - MCP `handleReindexRepository` / `reindexFile`
- Cross-repo resolution: `MultiIndexer.runCrossRepoResolveContext` (workspace_resolve.go:513) → `CrossRepoResolver`.

</code_context>

<specifics>
## Specific Ideas

- The copybook search order should mimic a mainframe SYSLIB concatenation: ordered, first match wins, and the rank is recorded.
- "Restore" must reproduce byte-identical canonical dump rows, not merely equivalent ones.

</specifics>

<deferred>
## Deferred Ideas

- A CLI/MCP `graph digest` / canonical-dump verb, for Phase 6 monitoring if needed.
- Feeding the copybook search order from the `cobol-repo-architect` retrieval manifest. Its manifest has no library concatenation yet, and that project is at its Phase 2.
- Runtime refusal of export or AI context when ownership health is unhealthy (Phases 6 and 7).
- Strict fail-closed scope for graph-metadata queries, beyond source reads.

</deferred>

---

*Phase: 04-stable-incremental-and-cross-repository-lifecycle*
*Context gathered: 2026-10-12*
