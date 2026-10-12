# Phase 4: Stable Incremental and Cross-Repository Lifecycle - Research

**Researched:** 2026-10-12
**Domain:** Incremental graph reconciliation, deterministic identity, cross-repository resolution scope, and fail-closed scope enforcement in the Gortex Go engine (indexer, resolver, SQLite store, MCP)
**Confidence:** HIGH for the codebase mechanics (every load-bearing claim was read in this session). MEDIUM for the recommended mechanisms (they are designs, not yet prototyped). LOW for a small set of flagged assumptions (see the Assumptions Log).

## Summary

Phase 4 is almost entirely an in-repo engineering problem. There are no new external dependencies: BLAKE3 (`github.com/zeebo/blake3 v0.2.4`) is already a direct dependency, and every needed primitive already exists (`OpenScopedProjectionSnapshot`, `PrefixDiagnostics`, `RepoIndexState`, `merkleSaltFor`, `purgeUnprefixedRepoRows`, `restubIncomingRefs`, `cobolCandidateIndex`). The work is to close specific, already-diagnosed gaps. Phase 3's review and verification documented most of them (CR-01, CR-02, WR-01, WR-10, IN-01..IN-04). Five of those are hard blockers for the equivalence proof (ID-09). Without them the harness cannot pass, not even for a clean build compared with itself:

1. **CR-02 makes clean builds nondeterministic today.** Shared DB2/IDMS/CICS resource nodes take `FilePath`, `Name`, and the per-document `prov_*` Meta from whichever document was written last. The SQLite node upsert overwrites `file_path` and `meta` on conflict. A clean build therefore depends on flush order, and so does any dump that contains file path and meta, which D-01 requires.
2. **CR-01: eviction destroys or falsifies COBOL edges.** `restubIncomingRefs` restubs a bound COBOL CALL to `unresolved::<Name>`. That leaves the owned `cobol::` namespace and lets the generic resolver bind it. Copybook targets are `KindFile` and COPY edges are `EdgeImports`, so both are outside the restub set and are simply deleted when the copybook is evicted.
3. **WR-01: COBOL binding never runs incrementally.** `resolveCobolReferences` runs only inside whole-graph `ResolveAll` and only looks at placeholder edges. A bound edge is never re-checked for exactly-one when a new candidate appears.
4. **The affected-by cap silently truncates** (`defaultAffectedByMax = 200`, Debug log).
5. **Material configuration has no invalidation path**, and the COBOL extractor reads no configuration at all today.

The recommended core mechanism is **one deterministic COBOL reconcile function used by both the full and incremental paths**, so equivalence holds by construction:

- Bound COBOL edges keep their original placeholder in edge Meta.
- On any COBOL mutation, the reconcile recomputes every literal CALL and COPY edge from the affected repository set. The affected set is the changed repo plus every repo whose allowlist names it. The result is a pure function of (placeholder, candidate set, repo policy).
- It garbage-collects orphaned shared resources and re-derives the D-11 alias edges.

`ResolveAll` calls the same function with the full scope. Everything else in the phase is a small, additive, or minimal edit, all at precisely located sites: the material fingerprint, the affected-by escalation, the D-12 ambiguity refusal, the source-read scope gate, single-repo prefixing, ownership health, and the shared scope type.

**Primary recommendation:** Fix shared-resource ownership and edge placeholder retention in the extractor first, bumping to `gortex-cobol-grammar/5`. Then build a single `reconcileCobol(repoSet)` resolver pass that replaces `resolveCobolReferences` and is invoked from the full resolve and every incremental mutation tail. Prove it with a SQLite real-path harness whose canonical dump is read through the generalized D-16 scope snapshot.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Carried forward (do not re-decide)
- Identity shapes are fixed:
  - Programs: `<repo>/<path>::OUTER/INNER#k`. Line numbers never discriminate (P2 D-10).
  - Data items: the full group path from level 01, with `#k` only for true duplicates and FILLERs (P3 D-07).
  - Resources: repo-scoped, with explicit `_unqualified` markers (P3 D-15).
  - Phase 4 makes these identities stable and proves them. It does not redesign them. ID-03 and ID-04 are mostly satisfied already and need proof plus gap fixes.
- Every relationship edge points back to its statement occurrence through a range plus the `KindLocal` statement node (P3 D-05/D-17). This is the basis for ID-05.
- CALL and COPY bind only within the same repo, and only on an exact single match. Unbound references carry `missing` / `ambiguous` reasons with candidate IDs. Names are normalized by `cobol-name-norm/1` (P3 D-09/D-11/D-18). `COPY IDMS` members are never bound (P3 D-19).
- Provenance lives in flat `prov_*` meta, with no new node fields or SQLite columns (P2 D-08). Prefer additive packages over invasive edits to upstream code (fork hygiene).
- Fixtures are invented COBOL only, because the repo is public (P2 D-14).

#### Equivalence proof
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

#### Invalidation and stale facts
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

#### Cross-repo resolution domains
- **D-09:** Explicit allowed scope is a **directed allowlist in workspace config** (for example, repo A may resolve COBOL CALL/COPY against repos B and C). With no entry, resolution is same-repo only. Binding across repos still requires an exact single match within the allowed set. More than one candidate means `ambiguous` with candidate IDs, and nothing is picked. The allowlist is part of the COBOL material fingerprint (D-06), so editing it invalidates bindings.
- **D-10:** **Copybook library search order is an explicit ordered library list per repo in config**, like a SYSLIB concatenation. The first match by order wins and records its search rank plus a versioned rule ID in meta. With no list configured, the P3 exactly-one-match rule stays.
- **D-11:** **Qualified and `_unqualified` resources never merge.** Both IDs stay distinct and stable. When config declares a default schema or subschema for a repo, emit a deterministic link edge from the `_unqualified` node to the qualified node, carrying the rule ID. IDs are never rewritten, so a config change never renames nodes. Claude picks the edge kind, reusing an existing kind. — **Reversibility:** costly. IDs are persisted and projected.
- **D-12:** **Generic cross-repo first-match becomes "ambiguous → refuse" for all languages.** `CrossRepoResolver.resolveFunctionCall` currently takes the first match from a Go map iteration (`cross_repo.go:1038`), which is nondeterministic. When more than one reachable repo matches, bind nothing and record reason `ambiguous` with candidate IDs, mirroring the COBOL rule. This is an accepted behavior change to upstream code, scoped to the minimal edit.

#### Scope and ownership
- **D-13:** **SEC-02 fails closed for source reads only.** When more than one repo is tracked, tools that return source bytes require a resolved scope. These include `read_file`, `get_symbol_source`, editing/smart context source payloads, and the future Phase 7 AI context path. Graph-metadata queries keep today's all-repos default for unbound sessions.
- **D-14:** **Single-repo mode is treated as a workspace of one.** Every node gets an owning repo prefix, and `PrefixDiagnostics` counts any unowned or misprefixed node as a violation. The `Clean()` path that accepts a fully unowned graph goes away for this purpose. Unresolved placeholders are exempt only through an explicit, enumerated namespace list. — **Reversibility:** costly. Changing ID prefixes for single-repo stores means a reindex, and Neo4j reprojection, of existing single-repo stores.
- **D-15:** **SEC-08 enforcement is a test gate plus health status.** The acceptance test fails on any ownership violation. `index_health` reports an unhealthy status with counts but does not block normal queries. Phases 6 and 7 may make export or AI refuse when unhealthy.
- **D-16:** **One shared scope check.** Generalize the Phase 1 `OpenScopedProjectionSnapshot` fail-closed validation into one scope type and validator used by projection, resolution, and source reads. The rule of three is now met. A cross-surface test asserts that the same repo allow-set yields identical node sets on each surface.

### Claude's Discretion
- Incremental COBOL resolution: today `resolveCobolReferences` runs only inside whole-graph `ResolveAll`, and incremental paths skip `cobol::` targets. D-04 requires incremental == clean, so the planner must make the incremental and watcher paths re-run COBOL CALL/COPY resolution for the affected scope (or escalate per D-08). Research picks the mechanism.
- The exact volatile-key allowlist (D-01), dump format, and digest algorithm. Prefer the existing BLAKE3 dependency.
- Config key names and shape for D-09, D-10, and D-11 in `.gortex.yaml` / workspace config. Follow existing workspace config conventions.
- The edge kind for the D-11 qualified link, reusing an existing kind.
- The enumerated placeholder-exemption namespaces (D-14).
- Rollout of D-14 to existing single-repo stores. This likely uses the existing `storeNeedsRebuild` / schema-version path.
- Plan slicing and wave order.

### Deferred Ideas (OUT OF SCOPE)
- A CLI/MCP `graph digest` / canonical-dump verb, for Phase 6 monitoring if needed.
- Feeding the copybook search order from the `cobol-repo-architect` retrieval manifest. Its manifest has no library concatenation yet, and that project is at its Phase 2.
- Runtime refusal of export or AI context when ownership health is unhealthy (Phases 6 and 7).
- Strict fail-closed scope for graph-metadata queries, beyond source reads.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ID-03 | Paragraph identity MUST include its owning program context. | Already `<path>::PROG/PARA` (P3). Gap: program-name components are not passed through `cobolIDSegment` (IN-03), so a literal PROGRAM-ID containing `/` or `#` can alias a nested path or an ordinal. Fix in the extractor and bump to `/5`. See "ID gap analysis". |
| ID-04 | Resource identity MUST include the minimum namespace to avoid merging distinct same-named artifacts. | Already repo-scoped `db2::<SCHEMA\|_unqualified>/<T>` etc. Gap: CR-02 (shared node owned by its last writer; Name as first written). Fix: resource `FilePath` = its own ID, writer-independent Name/Meta, GC. D-11 alias edge via `EdgeAliases`. IN-01 (second DB2 scheme `db::db2::S.T`) is recorded, not merged. |
| ID-05 | Repeated occurrences of one relationship keep distinguishable provenance. | Edge identity is `(from_id, to_id, kind, file_path, line, view_gen)`, so occurrences on different lines are distinct edges. Same-line duplicates collapse to one edge with `cobol_occurrence_count` and `cobol_statement_symbols` (IN-04). Prove both cases; decide whether IN-04's flow-union fix is in scope. |
| ID-06 | Reindexing identical inputs produces the same active identities. | Requires CR-02 to be fixed first. Harness: clean build twice, track order A,B vs B,A, and identical rerun, all with equal dumps. |
| ID-07 | A changed or deleted file's stale observations are removed. | D-05 hard-evict already exists. Gaps: CR-01 (COBOL restub branch), resource GC, and the `restub_prev_*` stash keys must not leak onto COBOL edges. |
| ID-08 | Parser, extractor, retrieval, or material-config changes invalidate observations even with unchanged bytes. | D-06/D-07 fingerprint via an optional extractor capability plus `index.cobol` config, salted into Merkle and persisted in `RepoIndexState`. Today `prov_extractor_version` is never compared, and the COBOL extractor reads no config. Retrieval: no retrieval input exists yet (deferred), so record it as absent. |
| ID-09 | Incremental == clean canonical active subgraph. | The single `reconcileCobol` function used by both paths, the D-02 dump helper, and the D-04 harness. |
| SEC-01 | One scope enforced across extraction, resolution, query, AI, claims, logging, and export. | D-16 shared `graph.Scope` and validator. Consumers: projection snapshot, COBOL resolution allow-set, MCP source-read gate. A cross-surface test. |
| SEC-02 | Ambiguous or absent scope fails closed for source reads and AI context. | D-13 gate at the source-byte choke points (`resolveNodePath`, `resolveFilePath`, `readViewFile`) when more than one repo is tracked and the session is unbound or unnarrowed. |
| SEC-03 | No cross-repo binding of same-named programs, copybooks, or resources without allowed scope and deterministic evidence. | COBOL: D-09 directed allowlist inside `reconcileCobol`. Generic: D-12 ambiguity refusal in `resolveFunctionCall` (and the identical pattern in `resolveMethodCall`). Resources: never cross-repo (IDs are repo-scoped). |
| SEC-08 | Ownership health reports zero unowned/misprefixed nodes for the acceptance corpus. | D-14 strict `PrefixDiagnostics`, embedded single-repo path prefixed, v22 in-place purge reusing `purgeUnprefixedRepoRows`, and D-15 `index_health.repo_ownership.status`. |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

These come from `./CLAUDE.md`, `./.claude/CLAUDE.md`, and the user's global CLAUDE.md. Treat them like locked decisions.

- **Build and test:** `go build -o gortex ./cmd/gortex/` (CGO is required). `go test -race ./...`: every test package must pass. Phase 2/3 run tests with `GOWORK=off`.
- **Simplicity:** write the simplest code that solves the task. Keep single-use code inline. **Rule of three**: extract only at three or more uses, and leave two occurrences duplicated with a `// note:` comment. No speculative generality, YAGNI, and no new dependencies, patterns, or layers without stated justification.
- **Reuse before create:** query for an existing equivalent before adding any function or type. Never fork an existing implementation; extend it.
- **File size:** target under 150 lines per component and under 300 per module, with one responsibility per file. Follow the neighboring folder conventions.
- **Fork hygiene:** prefer additive packages and analyzers over invasive edits to upstream internals. Keep upstream pull-ability.
- **Go conventions:** every exported function gets a doc comment starting with its name. Errors are wrapped with `%w` and context. Use early returns. Test names follow `TestFunctionName_Case`. Exported enums are typed constants. `PURPOSE —` / `RATIONALE —` / `KEYWORDS —` doc blocks are the house style for non-trivial functions.
- **Config:** `.gortex.yaml` via Viper with `mapstructure` + `yaml` tags. Validation lives in `validateWorkspaceSchema` and concatenates errors.
- **Security (org instructions):** OWASP Top 10, least privilege, no secrets in logs. Meta keys must avoid the projection's sensitive substrings (see Security Domain).
- **Process:** the work goes through GSD execute-phase. Prefer Gortex graph tools for indexed Go source. `*_test.go` files are not in the graph index, so read them with grep. Ignore the sibling `GoApps/gortex-phase4-sql` checkout.

## Scout Corrections (verify-don't-trust results)

| Scout claim (04-CONTEXT `<code_context>`) | Finding this session |
|---|---|
| "`.cbl/.cob/.cpy` not in the extractor salt; extractor version `/3`" (P3 WR-10) | **Stale.** Commit `9f6c1973` already maps `".cob": "cobol"`, `".cbl": "cobol"`, `".cpy": "cobol"` and sets `"cobol": 2` [VERIFIED: internal/indexer/extractor_version.go:48,101-103]. The extractor constant is `cobolGrammarExtractorVersion = "gortex-cobol-grammar/4"` [VERIFIED: internal/parser/languages/cobol_grammar.go:42]. D-07 still applies: the int `2` is hand-bumped and is not derived from the stamp. |
| "No incremental-vs-full equivalence harness" | **Partly wrong.** `TestIncrementalReindex_ConvergesToFullIndex` (internal/indexer/incremental_reindex_test.go:122) exists, but it covers only an in-memory Go graph, and its `canonicalGraph` renders only node identity and edge `From\|To\|Kind`, with no Meta, no SQLite, and no COBOL. Reuse its `bumpMtime` pattern and its shape. Nothing exists for SQLite, meta, COBOL, or cross-repo. |
| "First-match at `cross_repo.go:1038`" | The nondeterministic order comes from `for r := range reachableRepos` in `scopedCandidates` (cross_repo.go:1036). The first-match bind is in `resolveFunctionCall` step 2 (cross_repo.go:1297-1322), and `resolveMethodCall` has the same pattern. |
| "Single-repo prefixing needs a store rebuild" | The daemon already prefixes every repo, including a lone repo (multi.go:2393-2403, 2876-2883), and migration v7 already purged pre-flip unprefixed rows [VERIFIED: schema_version.go:97-103]. What remains unprefixed: the embedded `gortex mcp --index` path (`idx.Index(mcpIndex)` with no `SetRepoPrefix`, cmd/gortex/mcp.go:491), plus throwaway `init`/`eval_*` stores. |
| "`indexConfigHash` not in the primary-index salt" | Confirmed. It feeds only view-generation build identity (checkout_coordinator.go:1820, ref_views.go:258). |
| "`prov_extractor_version` never compared on reindex" | Confirmed: there is no reader. |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Stable COBOL identity (IN-03 encoding, resource owner path, placeholder retention) | Extractor (`internal/parser/languages/cobol_grammar_*`) | — | IDs and Meta are minted at extraction. The indexer only prefixes them. |
| Material fingerprint / staleness (D-06/D-07) | Indexer (`internal/indexer/extractor_version.go`, `index_state.go`, `merkle_reindex.go`) | Extractor (declares identity via an optional interface), Config (`index.cobol`) | Staleness is an indexer decision. The extractor only declares what it reads. |
| COBOL CALL/COPY binding, allowlist, search order, alias edges, resource GC | Resolver (`internal/resolver/cobol_references.go` → reconcile) | Indexer tails (trigger), MultiIndexer (policy lookup) | Binding is resolution. The trigger and the per-repo config snapshot live where the mutation is known. |
| Restub of COBOL in-edges before eviction (CR-01) | Indexer (`restubIncomingRefs`, `restubIncomingRefsFromView`) | — | This must happen pre-eviction, where the prior nodes are visible. |
| Affected-by escalation (D-08) | Indexer (`affected_by.go`, `incremental_batch.go`) | — | The cap lives here. |
| Generic cross-repo ambiguity (D-12) | Resolver (`cross_repo.go`) | — | — |
| Shared scope type and validator (D-16) | Graph (`internal/graph/scoped_projection.go` → `scope.go`) | Store (`store_sqlite/scoped_projection.go`), MCP, Resolver | `graph` already owns `ProjectionScope`, and every tier imports `graph`. |
| Source-read fail-closed (D-13) | MCP (`internal/mcp/tools_fileops.go`, `view_files.go`) | — | The source bytes leave the system here. |
| Ownership health (D-14/D-15) | Graph (`prefix_diagnostics.go`) + Store (SQLite predicates) | MCP (`index_health`), Store migration (v22) | — |
| Canonical dump + digest (D-02/D-03) | Test helper (`internal/testutil/graphdump`) | Store (scoped snapshot) | It is test-only in this phase. |

## Standard Stack

### Core (all already in `go.mod`; nothing to install)
| Library / package | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/zeebo/blake3` | v0.2.4 | Dump digest and material fingerprint | Already used by `internal/indexer/file_meta.go` (`blake3.Sum256`) and `internal/indexer/merkle/merkle.go` (`blake3.New()`) [VERIFIED: grep + go.mod:280] |
| `encoding/json` (stdlib) | Go 1.27 | Canonical Meta rendering | `json.Marshal` of `map[string]any` sorts keys, so it is a deterministic rendering for free [ASSUMED, standard Go behavior] |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions | House standard |
| `go.uber.org/zap/zaptest/observer` | (zap v1.28.0) | Asserting Info escalation logs (D-08) | Already used by `TestAffectedBy_CapBoundsFanout` |
| `modernc.org/sqlite` via `store_sqlite` | v1.54.0 | Real-path temp store | P2 D-12 pattern |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| BLAKE3 digest | `crypto/sha256` | `graphfixture` uses sha256. CONTEXT asks to prefer BLAKE3, and both are fine for test equality. |
| Reading the dump via `OpenScopedProjectionSnapshot` | Raw SQL like `graphfixture.Canonical` | Raw SQL renders `hex(meta)` from an encoded blob and includes edge rowids, which reflect insertion order. That breaks equivalence. The snapshot returns decoded nodes and edges, and using it also exercises the D-16 scope. |
| New column for the material fingerprint | Reserved key inside the `ExtractorVersions` JSON map | See Pattern 4. The column is cleaner, and the reserved key avoids a migration but needs inequality semantics hacked into `staleLangsBetween`. |

**Installation:** none.

## Package Legitimacy Audit

This phase installs no external packages. Every library listed above is already in `go.mod`/`go.sum`.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| (none new) | — | — | — | — | — | — |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
 file edit / delete / restore / config reload / version bump
            │
            ▼
 ┌─────────────────────────┐   stale? (mtime ledger | Merkle leaf salt)
 │ Indexer reconcile       │◄── D-06/D-07: COBOL material fingerprint
 │ (incrementalReindex...) │    (extractor identity + index.cobol config)
 └──────────┬──────────────┘
            │ for each stale/deleted file (pre-eviction)
            ▼
 ┌─────────────────────────┐  CR-01 COBOL branch: bound CALL/COPY in-edges
 │ restubIncomingRefs*     │──► To = e.Meta["cobol_placeholder"]  (no stash keys)
 └──────────┬──────────────┘
            ▼
 ┌─────────────────────────┐  D-05 hard-evict by file_path
 │ EvictFiles + AddBatch   │  shared resources are NOT evicted (FilePath = own ID)
 └──────────┬──────────────┘
            ▼
 ┌─────────────────────────┐  generic same-repo + cross-repo tails (unchanged)
 │ resolver tails          │  D-08: affected-by > cap ⇒ resolve ALL affected files
 │                         │  D-12: cross-repo >1 eligible repo ⇒ ambiguous, no bind
 └──────────┬──────────────┘
            ▼  (any COBOL path in the mutation)
 ┌───────────────────────────────────────────────────────────┐
 │ reconcileCobol(S)  S = changed repos ∪ repos allow-listing│
 │  1 candidate index (programs, copybooks) over allowed     │
 │  2 edges = placeholders + bound edges (via in-edges of    │
 │    candidates), From repo ∈ S                             │
 │  3 decide(placeholder, policy(callerRepo)) →              │
 │    bind | missing | ambiguous | rank (D-10)               │
 │  4 write only diffs (ReindexEdges / PersistEdgeAttributes)│
 │  5 alias edges _unqualified→qualified (D-11, EdgeAliases) │
 │  6 GC shared resources with no in-edges (CR-02)           │
 └──────────┬────────────────────────────────────────────────┘
            ▼
       SQLite store ──► graph.Scope (D-16) ──► projection snapshot / dump (D-02)
                                       └────► MCP source-read gate (D-13)
                                       └────► COBOL allow-set (D-09)
 index_health ◄── PrefixDiagnostics (D-14 strict, D-15 status)
```

### Recommended file layout (additive where possible)
```
internal/graph/scope.go                          # NEW: Scope type + NewScope validator + Allows (D-16)
internal/graph/scoped_projection.go              # ProjectionScope becomes an alias of Scope
internal/graph/prefix_diagnostics.go             # strict Clean / exemption list (D-14)
internal/resolver/cobol_references.go            # reconcileCobol replaces resolveCobolReferences body
internal/resolver/cobol_policy.go                # NEW: per-repo COBOL policy (allowlist, libraries, defaults) + lookup setter
internal/resolver/cobol_resources.go             # NEW: alias edges (D-11) + orphan resource GC (CR-02)
internal/resolver/cross_repo.go                  # D-12 minimal edit
internal/indexer/cobol_material.go               # NEW: COBOL material fingerprint (D-06/D-07)
internal/indexer/extractor_version.go            # salt method / stale check hook
internal/indexer/affected_by.go, incremental_batch.go   # D-08
internal/indexer/indexer.go, incremental_batch.go       # CR-01 COBOL restub branch
internal/config/config.go                        # index.cobol block + validation
internal/parser/languages/cobol_grammar*.go      # IN-03, placeholder meta, resource owner, MaterialIdentity
internal/mcp/source_scope.go                     # NEW: requireSourceScope (D-13)
internal/graph/store_sqlite/schema_version.go    # v22 (and v23 if the fingerprint column is chosen)
internal/testutil/graphdump/graphdump.go         # NEW: canonical dump + BLAKE3 digest + first-diff (D-02/D-03)
internal/indexer/cobol_lifecycle_test.go         # NEW: D-04 harness (needs package hooks; build tag darwin||linux)
cmd/gortex/cobol_scope_test.go                   # NEW: D-16 cross-surface + D-13 via MCP relay
```

### Pattern 1: One reconcile function for full and incremental COBOL binding (Claude's Discretion item)

**What:** Replace the body of `resolveCobolReferences` with `reconcileCobol(repos map[string]struct{})`. It recomputes the binding decision for every literal CALL and COPY edge whose source repo is in `repos`, from (placeholder, candidate set, policy). It writes only the diffs. `ResolveAll` calls it with the scope expanded by reverse allowlist. The incremental tails call it with `{changed repo} ∪ {repos whose allowlist names it}`.

**Why this mechanism:** A placeholder-only pass (today's) can never re-check a bound edge (WR-01). A key-scoped pass needs the keys of deleted definitions, captured before eviction and carried across three layers. A repo-scoped recompute needs neither, and it is literally the same function as the clean build, so incremental equals clean by construction. Cost per COBOL mutation is O(programs + copybooks + CALL/COPY edges in S). That is linear and bounded by the repo set, and "correctness wins over speed" (D-08 spirit).

**Required preconditions (extractor, version `/5`):**
- Every literal CALL and every COPY/INC/SQL INCLUDE edge carries `cobol_placeholder` = its exact original `unresolved::cobol::…` target, so a bound edge can be decided again and restubbed without a second normalization rule (D-18). Today a bound edge loses its placeholder: `e.To = candidates[0].ID` overwrites it [VERIFIED: cobol_references.go, read this session]. The name `cobol_placeholder` is a recommendation [ASSUMED name].
- Program symbols are encoded with `cobolIDSegment` (IN-03).

**Edge enumeration (no new store API is required, but one optional fast path is recommended):**
- Placeholders: today's `r.graph.EdgesWithUnresolvedTarget()` filtered by the `cobol::program::` / `cobol::copybook::` prefixes (excluding `cobol::copybook::idms/`) and by source repo ∈ S. Optional additive capability: an SQLite range scan `to_id >= 'unresolved::cobol::' AND to_id < 'unresolved::cobol:;'` to avoid a whole-graph unresolved scan per save.
- Bound edges: `r.graph.GetInEdgesByNodeIDs(candidateIDs)`, filtered to edges carrying `cobol_placeholder`, with source repo ∈ S. `GetInEdgesByNodeIDs` exists on `graph.Store` and SQLite [VERIFIED: graph/graph.go:3200, store_sqlite/store_lookups.go:245].

**Decision rule (extends the P3 rule; keep it in one function):**
```go
// Source: extends internal/resolver/cobol_references.go (read this session).
// allowed = {callerRepo} ∪ policy(callerRepo).ResolveFrom   (D-09, directed)
// candidates = index[repo+"\x00"+key] for repo in allowed
// programs: keep today's contained → sameFile → otherFile preference INSIDE callerRepo;
//           a candidate from another allowed repo joins the top-level set (no nesting visibility);
//           unknown cobol_nested ⇒ never a candidate (WR-10, already implemented)
// copybooks: if policy(callerRepo).CopybookLibraries non-empty (D-10):
//              rank(c) = index of first library whose (repo, dir) contains c.FilePath; unranked dropped;
//              keep min-rank candidates; if exactly 1 ⇒ bind + cobol_copy_search_rank, cobol_copy_search_rule
//            else: P3 exactly-one rule
// 1 candidate ⇒ To = id, Origin ASTResolved, delete reason/candidates
// 0 ⇒ To = placeholder, reason "missing";  ≥2 ⇒ To = placeholder, reason "ambiguous", sorted candidate IDs
// caller RepoPrefix == "" ⇒ fail closed: leave placeholder (SC5 "fail closed on missing scope")
```

**Where to invoke incrementally.** Use the outer mutation tails, not the resolver entry points. Reason: `incrementalResolutionFrontier` returns `needed=false` when a receipt says `!ResolutionRelevant` (incremental_watcher_batch.go:72-74), so a resolver-entry hook can be skipped entirely. Trigger when any stale, deleted, or restaged path has a COBOL extension (`ExtractorLangForFile(rel) == "cobol"`). There are four tail sites, which meets the rule of three, so one shared helper is justified:
1. `MultiIndexer.resolveIncrementalRepoMutationMode` (incremental_watcher_batch.go:376), which covers multi incremental, watcher, and evict.
2. `Indexer.incrementalWatcherPaths` (incremental_watcher_batch.go:290s).
3. `Indexer.incrementalEvictWatcherPath` (incremental_watcher_batch.go:236).
4. The full paths through `ResolveAll` (resolver.go:1662 call site). Expand the scope by reverse allowlist so that track order A,B vs B,A gives the same result.

The per-file legacy path `Indexer.indexFile` (indexer.go:4446 → `ResolveFileAndIncoming` at 4766) and `reresolveFileScopedRaw` (5172) should also call it if COBOL can reach them. Verify in Wave 0 with the `incrementalCatchupHook`.

### Pattern 2: CR-01 COBOL restub branch (minimal edit at two sites)

`restubIncomingRefs` (indexer.go:5218) and `restubIncomingRefsFromView` (incremental_batch.go:957) both:
- skip non-`IsReferenceableSymbol` nodes, which excludes `KindFile` copybooks;
- skip non-`IsResolvableRefEdge` kinds, which excludes `EdgeImports`;
- restub to `graph.UnresolvedMarker + n.Name`;
- call `graph.StashRestubProvenance(e)`, which writes `restub_prev_to`, `restub_prev_origin`, `restub_prev_tier`, and `restub_prev_conf` [VERIFIED: graph/restub_provenance.go:20-25].

Add a COBOL branch to both. For any in-edge (any kind) whose Meta has `cobol_placeholder`, set `To` to that value. Do **not** stash: the stash keys would survive on an edge that stays unresolved and make incremental differ from clean. Leave `Origin` as the extractor's value. Include COBOL `KindFile` copybook nodes in the fetched in-edge set. Two sites means a `// note:` duplicate, or one package-private helper `cobolRestubTarget(e) (string, bool)` in `indexer`. A private helper in the same package is allowed.

### Pattern 3: CR-02 shared resource ownership (extractor + reconcile GC)

**The defect:** `resourceNode` emits `FilePath: w.filePath`, `Name: name` (as first written in this document), and `maps.Clone(w.docMeta)` [VERIFIED: cobol_grammar_walk.go:230-243]. The SQLite node upsert sets `file_path=excluded.file_path` … `meta=excluded.meta` on conflict [VERIFIED: store_sqlite/add_batch_set.go:54-67], so the last writer wins. Evicting the owning file deletes every other program's edges into the resource.

**Fix:**
- Resource `FilePath` = the resource's own (unprefixed) ID. `applyRepoPrefix` prefixes ID and FilePath identically, so `PrefixDiagnostics` stays clean, and a real-file eviction never touches it.
- `Name` = the normalized key. The as-written spelling goes onto the edge (for example `cobol_resource_raw`).
- Meta = only writer-independent keys: `cobol_*` resource keys, `prov_range_exact=false`, `prov_range_absence=shared_resource`, and binary-constant `prov_*` (grammar ID, parser/preprocessor module, handoff schema, extractor version, evidence class, origin, confidence). **Drop** `prov_source_path`, `prov_source_id`, `prov_revision_content_id` (which also stops the VCS stamp), `prov_document_grade*`, `prov_grade_policy`, `prov_parse_config_id`, and `prov_transform_config_id`. Copybook mode uses a different transform config and grade policy, so these differ by writer [CITED: 03-REVIEW CR-02].
- GC in `reconcileCobol`: for every COBOL `KindResource`/`KindTable` marked `prov_range_absence=shared_resource` in repos S, if it has no in-edges other than `EdgeAliases`, call `EvictFile(n.FilePath)`. That path holds exactly that one node. `graph.Store` has no node-delete primitive (only `RemoveEdge`, `EvictFile`, `EvictRepo`) [VERIFIED: graph/store.go:160-237].
- Only `KindFile` nodes face the path-liveness ("orphan files") audit in `index_health`, so the virtual path does not trip it [VERIFIED: mcp/tools_enhancements.go:3207-3236].

Name-as-written vs. normalized is a decision that touches P2 D-20. See Open Question 2.

### Pattern 4: D-06/D-07 material fingerprint

- **Extractor declares its identity** through an optional capability, an additive Go interface idiom used throughout this codebase:
  ```go
  // in parser (or checked ad hoc in indexer): interface{ MaterialIdentity() string }
  func (e *CobolGrammarExtractor) MaterialIdentity() string {
      return cobolApprovedGrammarID + "|" + cobolApprovedPreprocessorModule + "|" +
          cobolGrammarExtractorVersion + "|" + cobolNameNormRule
  }
  ```
  The same constants are written into `prov_parser_grammar_id`, `prov_preprocessor_module`, `prov_extractor_version`, and `cobol_name_norm`. `init()` refuses any other preprocessor module and any other compiled grammar, so `prov_parser_tool_id`/`prov_parse_config_id` are determined by these pins (D-07). Values [VERIFIED: cobol_grammar.go:32,39,42; cobol_grammar_identity.go:152]: `cobolApprovedGrammarID = "f97452e11a2b80b92acb7edf776131470bc2e1ffd7c077f4ade4ce3c3a47503d"`, `cobolApprovedPreprocessorModule = cobolPreprocessorModulePath + "@v0.0.0-20261002100409-e64234cc493c"`, `cobolGrammarExtractorVersion = "gortex-cobol-grammar/4"`, `const cobolNameNormRule = "cobol-name-norm/1"`. The `!darwin && !linux` stub (`cobol_grammar_unsupported.go`) must implement it too, or the indexer must tolerate its absence.
- **The indexer looks it up** with `idx.registry.GetByLanguage("cobol")` (parser/registry.go:56). That requires no `indexer → languages` import. Verified: `languages` does not depend on `indexer` (`go list -deps`), but the capability route avoids the edge anyway.
- **Fingerprint** = `blake3(MaterialIdentity() + "\n" + canonicalJSON(idx.config.Cobol))`, hex-truncated (for example 16 bytes). It includes only the inputs actually read. There is no retrieval manifest today, so leave it out (YAGNI) and say so in a comment.
- **Merkle salt:** `merkleSaltFor` is a free function passed to `merkle.Build(…, merkleSaltFor)` (merkle_reindex.go:92,118). Change those two call sites to a closure or method that appends `|cobol-material@<fp>` for COBOL paths. Drop `"cobol": 2` from `extractorVersions`: D-07 says the hand-bumped map stays *for other languages*. Do keep the `.cob`/`.cbl`/`.cpy` mapping, which carries the global policy salt.
- **Non-Merkle (default) path:** `Merkle bool` defaults to off (config.go:703). Staleness then comes only from `extractorVersionStaleLangSet()` on full-root passes (indexer.go:6181). Persist the fingerprint in `RepoIndexState` and compare it there.
  - **Recommended:** add `MaterialFingerprints string` (JSON lang→hex) with an in-place `ALTER TABLE repo_index_state ADD COLUMN` migration. This is freshness state, not node provenance, so P2 D-08's "no SQLite columns" (which is about provenance) does not apply. Flagged as A3.
  - **Alternative with no migration:** a reserved key in the `ExtractorVersions` map whose value is a 53-bit fingerprint integer, compared with `!=` in `staleLangsBetween` (alongside the existing `postExtractionPolicySnapshotKey` special case).
- **Config:** put the COBOL block under `index:` (`IndexConfig.Cobol`). Three reasons: (1) every per-repo Indexer already receives `cfg.Index` (multi.go:2387, 2883); (2) `indexConfigHash` covers it for view generations automatically, which is correct; (3) `RefreshRepoConfigs` (multi.go:2457) needs one added line to push it into the live Indexer. Today it pushes only `Exclude`, so without the change a reload would not reach the fingerprint.

### Pattern 5: D-08 affected-by escalation

There are two truncation sites with identical logic [VERIFIED: affected_by.go:556-563; incremental_batch.go:1494-1499]:
```go
if maxFiles := idx.affectedByMaxFiles(); len(files) > maxFiles {
    idx.logger.Debug("affected-by: re-resolve set truncated", ... zap.Int("dropped", len(files)-maxFiles))
    files = files[:maxFiles]
}
```
**Minimal correct escalation:** above the cap, log at **Info** (`"affected-by: cap exceeded, resolving full affected set"` with `affected` and `cap`) and **do not truncate**. The files dropped today are exactly the callers whose edges `restubIncomingRefs` already parked on `unresolved::<Name>` stubs. Resolving the full list with the same batched `ResolveFilesAndIncoming` + `SynthesizeExternalCallsForFiles` + `persistRefFactsForFiles` is the "full resolve of the affected scope". A scoped `ResolveAll` would also rebind pending stubs, but it costs more and does not refresh ref facts per file. `TestAffectedBy_CapBoundsFanout` (affected_by_e2e_test.go:207) pins the truncation and must be rewritten. Update the doc comment on `IndexConfig.AffectedByReresolveMax` (config.go:762-772), which says "truncated". `SparseGenerationBuilder.builderClosureCap` (builder_closure.go:85-91) reuses the cap but already publishes `ClosureTruncated` honestly. It is out of D-04's path, so leave it.

### Pattern 6: D-12 minimal edit

In `resolveFunctionCall` step 2 (cross_repo.go:1305-1322), collect the eligible candidates instead of returning the first. If they span more than one repo, do not bind: stamp `resolution_unresolved_reason="ambiguous"` and sorted `resolution_candidate_ids` (key names are a recommendation [ASSUMED]), and persist them with the cross-repo pass's batch. Otherwise bind the single repo's candidate. If one repo holds several, pick the lowest ID, mirroring the existing `lowestIDQualNameCandidate` precedent in `pickQualNameCandidate` (cross_repo.go:324-352). That function already refuses more than one eligible foreign candidate (`return nil, true`). Apply the same edit to the identical step 2 in `resolveMethodCall`. D-12 names only `resolveFunctionCall`, but its intent is "for all languages" (Open Question 5). Clear the two keys whenever the cross-repo pass binds the edge. Blast radius: the baseline `go test ./internal/resolver -run CrossRepo` is green (2.3 s, this session). The likely-affected test is `TestCrossRepoResolveAll_CrossRepoFallback`.

### Pattern 7: D-16 shared scope type

Make `graph.ProjectionScope` (graph/scoped_projection.go:19-23: `Workspace string; Project string; Repositories []string`) the shared type. Rename it to `graph.Scope` and keep `type ProjectionScope = Scope`, so the projection consumers do not churn. Add:
```go
// NewScope normalizes and validates; fails closed. Source: generalizes
// store_sqlite/scoped_projection.go:22-50 (read this session).
func NewScope(workspace, project string, repos []string) (Scope, error) // trims, rejects empty set / blank entry, dedupes, sorts
func (s Scope) RequireWorkspaceProject() error                         // projection-only rule (today's line 28 check)
func (s Scope) Allows(n *Node) bool  // n.RepoPrefix != "" && in set; and Workspace/Project match when set
```
Consumers:
- `OpenScopedProjectionSnapshot` and `neo4jprojection.NormalizeRequest` (service.go:140-180) call `NewScope` (+`RequireWorkspaceProject`).
- `reconcileCobol` builds a repos-only `Scope` per caller repo (D-09 allow-set) and filters candidates with `Allows`.
- The MCP source gate converts `ResolvedScope.RepoAllow` to a `Scope`. A nil or empty set is refused when more than one repo is tracked.

Unlike `query.QueryOptions.ScopeAllows`, which *admits* empty-prefix nodes (query/subgraph.go:306-318), `Scope.Allows` must reject them. That is the D-14 world. The cross-surface test indexes two repos and, for an allow-set of {A}, asserts set-equality of node IDs from: (a) projection `ReadNodePages`; (b) the reconcile candidate filter; (c) the IDs for which the MCP source gate permits `get_symbol_source`.

### Pattern 8: D-13 source-read gate

`nodeInSessionScope` returns `true` for an unbound session (mcp/server.go:2456-2465). An unbound session (no cwd or no multi-indexer, mcp/server.go:2259-2263) gets `RepoAllow == nil`, meaning all repos, from `resolveUnboundScope`. That is the fail-open.

Add `s.requireSourceScope(ctx, repoPrefix string) error`:
- No-op when `multiIndexer == nil` or `len(RepoPrefixes()) <= 1`.
- Otherwise require one of: a bound session, an explicit `repo`/`project`/`workspace`/`scope` narrowing, an active project, or a server-default workspace. Also require `repoPrefix` ∈ the resolved allow-set.

Call it at the three source-byte choke points: `resolveNodePath` (tools_fileops.go:474, 28 callers), `resolveFilePath` (tools_fileops.go:68, 22 callers), and `readViewFile` (view_files.go:203, 4 callers). Gating at the choke points covers tools that never call `nodeInSessionScope`: `read_file`, `get_symbol_source`, `batch_symbols include_source`, `get_editing_context compress_bodies`, `smart_context`, `explore`, `export_context`, `search_text` match lines, `diff_context`, `pr_review_context`, and `compare_with_overlay`. Write tools also pass through `resolveFilePath`. Gating them too is the safer reading of SEC-01 (Open Question 6). Inventory test: a table test that calls every source-returning tool from an unbound session with two tracked repos and expects an error.

### Pattern 9: D-14/D-15 ownership

- Strictness: `Clean()` currently "tolerates a wholly-unowned graph" (prefix_diagnostics.go:70-74). Make unowned > 0 a violation. Flip the storetest fence `testPrefixDiagnostics` and the unit test "wholly unprefixed is single-repo mode, not a defect" (prefix_diagnostics_test.go:115-121).
- **Enumerated exemption list.** It already exists implicitly, and the storetest fence pins it [VERIFIED: storetest.go:211-224; prefix_diagnostics.go:117-149]: FilePath `""` (global `dep::…`), FilePath prefix `external::`, FilePath prefix `external-call::`, and kinds `KindContract`, `KindContractBridge`, `KindTopic`, `KindRationale`. Turn it into a named, documented list (one place, tested), and make an empty-FilePath node exempt only when its ID namespace is enumerated (today: `dep::`). `unresolved::` targets are edge strings, not nodes, so they need no node exemption.
- Single-repo: the embedded path (`cmd/gortex/mcp.go:491` `idx.Index(mcpIndex)`) must run as a workspace of one. Recommended: track via the `MultiIndexer` that `shared_server.go:577-579` already builds, so one code path prefixes. Blast radius: the `resolveNodePath` single-repo fallback joins `root + node.FilePath` without stripping the prefix (tools_fileops.go:510-512).
- Rollout: append `{version: 22, name: "purge unprefixed solo-repo rows (embedded)", inPlace: purgeUnprefixedRepoRows}`. The function is idempotent, and its own doc explains why in-place beats `rebuild: true` (migration_unprefixed_purge.go:23-30). `currentSchemaVersion = 21` [VERIFIED: schema_version.go:37] becomes 22. Neo4j: Phase 1 projection reconciliation is generation-owned, so a re-push replaces old IDs. Document it as a manual step.
- D-15: add `"status": "healthy"|"unhealthy"` plus counts to `index_health.repo_ownership`. Today the block has `owned_code_nodes`, `unowned_code_nodes`, `consistent`, `misprefixed_*`, and `mixed`/`unowned_samples` [VERIFIED: mcp/tools_enhancements.go:3483-3500]. Always surface `unowned_samples` once unowned counts as a defect.

### Anti-Patterns to Avoid
- **Hooking COBOL reconcile only at resolver entry points.** A `!ResolutionRelevant` receipt skips the resolver entirely.
- **Calling `StashRestubProvenance` for COBOL restubs.** The `restub_prev_*` keys leak into the dump.
- **Re-normalizing names in the resolver** (D-18). Use the retained `cobol_placeholder`.
- **Removing the cap without logging** (D-08 requires an Info log with counts).
- **Reading the dump with raw SQL** (`hex(meta)` and edge rowids are not canonical). Use the decoded scoped snapshot and sort.
- **Making `Scope.Allows` admit empty prefixes.** That reintroduces fail-open.
- **Putting COBOL config at the top level** where the per-repo Indexer cannot see it (`New(g, reg, cfg.Index, …)`).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Unprefixed-row rollout | A new purge or a rebuild rung | `purgeUnprefixedRepoRows` as v22 `inPlace` | Already handles edges, vectors, nodes, and sidecars, idempotent |
| Scoped read of the active subgraph | New SQL | `OpenScopedProjectionSnapshot` `ReadNodePages`/`ReadEdgePages` | Already scoped, generation-bound, paged, and decoded |
| Deterministic ID tie-break | A new sort helper | `lowestIDQualNameCandidate` pattern (cross_repo.go) | Existing precedent |
| Persisting Meta-only stamps | Ad hoc writes | `graph.EdgeMetaBatchPersister.PersistEdgeAttributesBatch`, `ReindexEdges` | What `resolveCobolReferences` already uses |
| Per-repo config snapshot inside a resolver pass | A global config read | The `SetCrossWorkspaceDepLookup(mi.crossWorkspacePassLookup())` pattern (workspace_resolve.go:492-559) | Frozen per pass, with reload visibility |
| Hashing | sha/fnv | `blake3.Sum256` / `blake3.New()` | Already a dependency |
| Sensitive meta key screening | A new filter | `neo4jprojection.sensitiveMetadataKey` rules | New keys must pass them |

**Key insight:** every hard part of this phase already has an in-repo precedent. The risk is in the wiring order and in leaked state (stash keys, writer-dependent Meta, truncation), not in algorithms.

## Runtime State Inventory

This phase changes IDs and Meta for COBOL (version `/5`), resource FilePaths, and single-repo prefixes, so it qualifies.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | (1) SQLite stores with COBOL nodes from `/4` and earlier: resource nodes owned by real file paths, programs with unencoded names, bound edges without `cobol_placeholder`. (2) Embedded single-repo stores with `repo_prefix=''` rows. (3) `repo_index_state` rows without a material fingerprint. | (1) The extractor `/5` plus the fingerprint mismatch restages every COBOL file (code edit, automatic re-extraction). Stale orphan resources under old real-file paths are evicted when their file restages. Verify in the harness. (2) v22 in-place purge (data migration). (3) A missing fingerprint is treated as stale → one restage (code edit). |
| Live service config | Neo4j projections made from Phase 3 graphs hold the old resource FilePaths, Names, and Meta, and any IN-03 ID changes. | Manual re-run of the scoped projection (generation-owned reconciliation replaces records). Document it. Nothing is in git. |
| OS-registered state | None. Verified: no launchd/systemd/Task Scheduler entries embed COBOL IDs or prefixes. The daemon registration is unaffected by graph contents. | None |
| Secrets/env vars | None renamed. `GORTEX_MERKLE` / `index.merkle` select the salt path. The new `index.cobol` keys are not secrets. | None |
| Build artifacts | None. The Go build cache is unaffected. The private fork module is already pinned and cached (`TestCobolBreadth_CallsAndCopy` ran green in 5.5 s this session). | None |

## Common Pitfalls

### Pitfall 1: Clean builds are not deterministic until CR-02 is fixed
**What goes wrong:** two clean builds of the same corpus give different dumps (resource `file_path`, `name`, and `prov_*` from different writers).
**How to avoid:** make "clean build twice / track order swapped ⇒ equal dumps" the first harness test, and fix the extractor before anything else.
**Warning signs:** dump diffs only on `db2::`, `idms::`, or `cics::` rows.

### Pitfall 2: The mtime ledger misses same-second edits
**What goes wrong:** an edit is not detected, so "incremental" silently does nothing.
**How to avoid:** use `bumpMtime` (write, then `os.Chtimes` to now+2s), as `incremental_reindex_test.go` does. Also bump on restore.

### Pitfall 3: `prov_vcs_commit` drifts when HEAD moves without content change
**What goes wrong:** `stampSourceRevision` writes the HEAD commit only when the bytes equal HEAD's blob (source_revision.go:21-30). Unchanged files are not re-stamped when HEAD moves, but a clean rebuild stamps the new HEAD.
**How to avoid:** keep HEAD fixed for the whole harness sequence (edit the working tree, don't commit). Decide the policy for HEAD movement (Open Question 1).

### Pitfall 4: Stash and terminal keys leak into the dump
**What goes wrong:** `restub_prev_*` survive on unresolved COBOL edges. Also, `resolve_terminal` / `resolve_terminal_reason` [VERIFIED: graph/edge_terminal_stamps.go:6,9] are stamped only by full unscoped passes. COBOL placeholders are never classified terminal, because `identifierFromTarget` returns "" for `cobol::` (lsp_resolve.go:517-523), but other languages' edges would be.
**How to avoid:** no stash for COBOL. Put `resolve_terminal` and `resolve_terminal_reason` on the volatile allowlist with that justification, and only add them if the harness observes them.

### Pitfall 5: Allowlist directionality and reverse fan-out
**What goes wrong:** repo B adds a program that A's allowlist can reach, but only B's edges are recomputed, so A stays bound to the old candidate or stays `missing`.
**How to avoid:** S = changed repo ∪ {repos whose allowlist names it}. Test it with track order A,B vs B,A.

### Pitfall 6: Config reload never reaches the live Indexer
**What goes wrong:** `RefreshRepoConfigs` pushes only `Exclude` (multi.go:2476-2482), so the fingerprint never changes in a running daemon.
**How to avoid:** push `cfg.Index.Cobol` too. The harness calls `mi.RefreshRepoConfigs()` and then a full-root `IncrementalReindexRepo(prefix, nil)`.

### Pitfall 7: `IncrementalReindexRepo` has no context parameter
**What goes wrong:** it uses `context.Background()` internally (multi.go:2497-2523), so the harness cannot cancel it to simulate an interruption.
**How to avoid:** simulate interruption inside the `indexer` package. Set `idx.incrementalResolveFilesHook = func([]string) {}` (indexer.go:526-529), so parse, evict, and AddBatch commit but the resolver tail does not. Then "retry" with a fresh `MultiIndexer` over the same SQLite file via `ReconcileRepoCtx` (the warm-restart path) and compare with a clean build. This is why the D-04 harness lives in `internal/indexer`.

### Pitfall 8: The generic ambiguity reason goes stale
**What goes wrong:** a later per-repo bind leaves `resolution_unresolved_reason` on a now-bound edge.
**How to avoid:** clear the keys on bind in the cross-repo pass. Document that readers must consult them only when `To` is unresolved. Generic-language incremental re-checking of already-bound cross-repo edges is out of D-04's corpus (Open Question 4).

### Pitfall 9: Restoring a file without a fresh `#k`
**What goes wrong:** `#k` is positional within a document, so restore reproduces IDs only if the content is byte-identical.
**How to avoid:** the harness restores the original bytes exactly and asserts the dump rows for that file are byte-identical to the pre-edit dump (CONTEXT specifics).

### Pitfall 10: New meta keys trip the projection's secret screen
`sensitiveMetadataKey` lower-cases the key, strips `-_. `, and rejects keys containing `password`, `passwd`, `secret`, `token`, `credential`, `privatekey`, `apikey`, or `authorization` [VERIFIED: neo4jprojection/properties.go:171-177]. Check every new key, for example `cobol_placeholder`, `cobol_copy_search_rank`, `cobol_copy_search_rule`, `cobol_qualifier_rule`, `cobol_resource_raw`, and `resolution_candidate_ids`.

## Code Examples

### Canonical dump (test helper, D-02/D-03)
```go
// internal/testutil/graphdump/graphdump.go (NEW, test-only; mirrors graphfixture's "production must not import")
// Reads through graph.ScopedProjectionSnapshotOpener so store_sqlite tests can use it without a cycle.
type Volatile map[string]string // key -> justification (D-01: every exclusion justified)

func Dump(ctx context.Context, o graph.ScopedProjectionSnapshotOpener, scope graph.ProjectionScope, v Volatile) ([]string, string, error) {
	snap, err := o.OpenScopedProjectionSnapshot(ctx, scope)
	if err != nil { return nil, "", err }
	defer snap.Close()
	var rows []string
	err = snap.ReadNodePages(ctx, func(ns []*graph.Node) error {
		for _, n := range ns {
			rows = append(rows, fmt.Sprintf("N\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d:%d-%d:%d\x1f%s",
				n.ID, n.Kind, n.Name, n.FilePath, n.StartLine, n.StartColumn, n.EndLine, n.EndColumn, canonMeta(n.Meta, v)))
		}
		return nil
	})
	if err == nil {
		err = snap.ReadEdgePages(ctx, func(es []graph.ScopedEdgeRow) error {
			for _, r := range es {
				e := r.Edge
				rows = append(rows, fmt.Sprintf("E\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d\x1f%s",
					e.From, e.To, e.Kind, e.FilePath, e.Line, canonMeta(e.Meta, v)))
			}
			return nil
		})
	}
	if err != nil { return nil, "", err }
	sort.Strings(rows) // edge rowids are insertion-ordered; never use them
	sum := blake3.Sum256([]byte(strings.Join(rows, "\n")))
	return rows, hex.EncodeToString(sum[:]), nil
}
// canonMeta: copy minus volatile keys, json.Marshal (sorted keys). FirstDiff(a, b, n) prints first n differing rows.
```
Scope for the dump: `OpenScopedProjectionSnapshot` requires workspace and project [VERIFIED: store_sqlite/scoped_projection.go:28-30]. The harness should read `WorkspaceID`/`ProjectID` from an indexed node of each repo rather than assume the defaults (A6).

### Volatile allowlist (start minimal; evidence-driven)
```go
var phase4Volatile = graphdump.Volatile{
	// Add an entry ONLY when the harness shows a diff caused by it, with the reason.
	"resolve_terminal":        "resolver work-avoidance stamp written only by full unscoped passes (resolver/terminal.go)",
	"resolve_terminal_reason": "companion of resolve_terminal",
}
// NOT volatile (must never appear; assert absent): restub_prev_to, restub_prev_origin, restub_prev_tier, restub_prev_conf
```

### D-04 harness skeleton (internal/indexer, `//go:build darwin || linux`)
```go
// Clean: fresh SQLite + MultiIndexer.TrackRepoCtx for each repo (both orders) → Dump.
// Incremental: one store; Track; then mutate with bumpMtime/os.Remove/config edit;
//   mi.RefreshRepoConfigs(); mi.IncrementalReindexRepo(prefix, nil) → Dump.
// require.Equal(t, cleanDigest, incDigest, graphdump.FirstDiff(clean, inc, 20))
// Sequences: rerun, edit, delete, restore (byte-identical rows), extractor bump / parser-identity change
// (simulate by rewriting the persisted fingerprint in RepoIndexState and the stamped prov_* on one node,
// then assert the restage removed it), material-config change, cross-file B→A, cross-repo with/without
// allowlist, interrupted (incrementalResolveFilesHook no-op) + ReconcileRepoCtx retry,
// affected-by > cap (index.affected_by_reresolve_max = 1 via config).
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Unprefixed single-repo mode | Every repo prefixed, including a lone repo, in the daemon | v7 migration (`purgeUnprefixedRepoRows`) | D-14 is mostly done for the daemon. The embedded path remains. |
| COBOL not version-salted | `.cob/.cbl/.cpy` → `cobol@2` salt | Commit `9f6c1973` (2026-10-12) | D-07 replaces the hand bump with a stamp-derived fingerprint. |
| `resolveCobolReferences` placeholder-only pass | (recommended) `reconcileCobol(S)` recompute | Phase 4 | Fixes WR-01 and makes equivalence hold by construction. |

**Deprecated/outdated:** the `PrefixDiagnostics.Clean()` doc "tolerates a wholly-unowned graph" and `logRepoOwnershipAudit`'s "A wholly-unowned graph is single-repo mode, not a defect" (cmd/gortex/daemon_state.go:1277) both become wrong under D-14.

## ID-03 / ID-04 / ID-05 Gap Analysis

| Req | Current state (verified) | Gap | Fix |
|---|---|---|---|
| ID-03 | Program symbol `base := name`; nested `container + "/" + name`; `#k` for the k≥2 duplicate (cobol_grammar_identity.go:106-116). Paragraphs extend the program symbol. Copybook paragraphs use `<path>::@proc/<PARA>`. | The program name is not passed through `cobolIDSegment` (IN-03). A literal PROGRAM-ID `'A/B'` aliases nested `A/B`, and `'X#2'` aliases the 2nd `X`. | Encode each component with `cobolIDSegment` (which already escapes `% : / #` and control bytes, cobol_grammar_identity.go:208-226). Bump `/5`. Add a collision test. |
| ID-04 | Repo-scoped `db2::<schema\|_unqualified>/<t>`, `idms::_unqualified/<role>/<k>`, `cics::transaction/…` (cobol_grammar_sql.go:263, cobol_grammar_mainframe.go:85,170-188). | CR-02 ownership and Name. IN-01: a second DB2 scheme `db::db2::S.T` from DDL (sql.go:832-841). | Pattern 3. Record IN-01 as a known divergence (no merge, which D-11 forbids). |
| ID-05 | Edge key `(from_id, to_id, kind, file_path, line, view_gen)` (store_sqlite/schema.go:953-956; graph.go:80-81). Same-line duplicates collapse with `cobol_occurrence_count` and `cobol_statement_symbols` (cobol_grammar_dedupe.go). | IN-04: the survivor keeps only the first occurrence's `cobol_flow` and range. | Prove distinct lines give distinct edges, and same line gives distinct `cobol_statement_symbols`. Fix IN-04 (union `cobol_flow`) only if the planner judges "distinguishable provenance" unmet (Open Question 3). |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Meta key names `cobol_placeholder`, `cobol_resource_raw`, `cobol_copy_search_rank`, `cobol_copy_search_rule` (`cobol-copy-search/1`), `cobol_qualifier_rule` (`cobol-default-qualifier/1`), `resolution_unresolved_reason`, `resolution_candidate_ids` | Patterns 1, 3, 6 | Only naming. They become a projection contract (P2 D-08 reversibility). |
| A2 | Config shape `index.cobol: {resolve_from: [repo…], copybook_libraries: [{repo?, path}], db2_default_schema, idms_default_subschema}`, validated in `validateWorkspaceSchema` (no wildcards, repo-relative paths, no `..`) | Pattern 4 | Wrong placement means the per-repo Indexer cannot see it. The D-09 wording says "workspace config", and per-repo `.gortex.yaml` matches the `cross_workspace_deps` precedent. |
| A3 | Persist the fingerprint as a new `repo_index_state` column with an in-place migration (vs. a reserved key in `ExtractorVersions`) | Pattern 4 | Schema version churn. Either option works. |
| A4 | Union semantics for D-09: a same-repo candidate plus an allowed-repo candidate is `ambiguous` (no same-repo-first preference across repos) | Pattern 1 | A different but still deterministic result. Needs user confirmation. |
| A5 | D-10 library entries are directories (optionally in an allowed repo). A COPY with `OF/IN <lib>` restricts the search to libraries whose name matches. Candidates outside every listed library are invisible. | Pattern 1 | Mainframe semantics mismatch. Needs user confirmation. |
| A6 | Default `WorkspaceID`/`ProjectID` for a tracked repo are non-empty, so the projection snapshot accepts the dump scope | Code Examples | The dump needs another scope path. Read them from the nodes. |
| A7 | `json.Marshal` sorts map keys, so rendering is deterministic | Code Examples | Standard Go behavior. Low risk. |
| A8 | The D-11 alias edge is emitted only when the qualified node exists, and never creates a node | Pattern 3 | The alternative invents nodes, which DET rules forbid. |
| A9 | Resource Name becomes the normalized key, with the as-written spelling moved to the edge | Pattern 3 | Touches P2 D-20 "names as written". Needs user confirmation. |

## Open Questions (RESOLVED)

All seven questions were closed after research. Three were closed by the user-confirmed decisions D-17, D-18, and D-20 in 04-CONTEXT.md (D-19 closed Assumption A4). One was closed by D-12's "for all languages" wording. Three were closed by adopting the recommendation under CONTEXT "Claude's Discretion". Each item keeps its original text and adds where it was decided and which plan implements it.

1. **HEAD movement vs. `prov_vcs_commit`.** What we know: a clean rebuild after a commit stamps the new HEAD on unchanged files, and incremental does not. What is unclear: whether HEAD is a "final input" for ID-09. Recommendation: either re-stamp revision Meta on HEAD move in `reconcileRepoIndexState`, or allowlist `prov_vcs_commit` and `prov_vcs_commit_absence` with the justification "derived from repository HEAD, not file content". Ask the user. In the meantime the harness keeps HEAD fixed.
   **RESOLVED by D-17 (user-confirmed):** HEAD is a final input. A HEAD move with unchanged bytes re-stamps `prov_vcs_commit` as a metadata-only refresh. `prov_vcs_commit` stays compared in the canonical dump and is not on the volatile allowlist. Implemented by 04-16 Task 2 (`restampSourceRevisions`, `TestCobolLifecycle_HeadMove`). The other harness sequences keep HEAD fixed (04-08, 04-15).
2. **Shared resource Name (A9).** Recommendation: use the normalized key, and keep the raw spelling on edges.
   **RESOLVED by D-18 (user-confirmed):** a shared DB2, IDMS, or CICS resource node is named by its `cobol-name-norm/1` key, and each referencing edge keeps the as-written spelling in `cobol_resource_raw`. Implemented by 04-01 Task 1 and locked by 04-11 (`TestCobolGrammar_SharedResource`).
3. **IN-04 in scope?** Recommendation: prove ID-05 with the current behavior. Fix IN-04 only if a test shows two semantic occurrences become indistinguishable.
   **RESOLVED (recommendation adopted, CONTEXT "Claude's Discretion"):** ID-05 is proven with the current dedupe behavior, and IN-04 stays unchanged unless that proof fails. Implemented by 04-02 Task 2 (`TestCobolGrammar_Occurrences`, see its flagged assumptions).
4. **Generic-language incremental equivalence.** D-12 makes the decision deterministic per pass, but bound generic cross-repo edges are not re-checked when a new candidate appears. Recommendation: scope the ID-09 acceptance corpus to COBOL (plus the D-12 unit tests) and record the generic gap for Phase 6.
   **RESOLVED (recommendation adopted, CONTEXT "Claude's Discretion"):** the ID-09 equivalence corpus is COBOL only, and already-bound generic cross-repo edges are not re-checked incrementally. The D-12 per-pass determinism is proven by 04-04. The generic gap is recorded as a flagged assumption in 04-08 and 04-15, for Phase 6.
5. **`resolveMethodCall`.** Should it get the same D-12 edit? Recommendation: yes, because it has the identical first-match pattern and D-12 says "for all languages".
   **RESOLVED (D-12 "for all languages"; recommendation adopted):** all three cross-repo loops of `resolveMethodCall` get the same refuse-on-ambiguity edit. Implemented by 04-04 Task 2 (`TestCrossRepoResolveAll_AmbiguousMethod`).
6. **Source gate on write tools** (`resolveFilePath` is shared with `edit_file`/`write_file`). Recommendation: gate them too (fail closed), and confirm with the user.
   **RESOLVED by D-20 (user-confirmed):** any tool that reads or writes source bytes, including `edit_file`, `write_file`, and every other `resolveFilePath` caller, requires a resolved scope when more than one repo is tracked. Implemented by 04-12 (`TestSourceScope_WriteToolsGated`).
7. **Embedded single-repo via `MultiIndexer`** vs. `SetRepoPrefix` on the bare Indexer. Recommendation: `MultiIndexer` (one code path). Measure the `internal/mcp` test blast radius first: 64 test files construct `indexer.New(`.
   **RESOLVED (recommendation adopted, CONTEXT "Claude's Discretion": embedded single-repo mode goes through MultiIndexer after its blast radius is measured):** implemented by 04-13, which records the before and after blast-radius counts in its SUMMARY.

Assumptions A4 and A9 in the Assumptions Log were confirmed by the same post-research decisions: A4 by D-19 (union semantics), A9 by D-18 (normalized resource Name).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | all | ✓ | go1.27.1 (go.mod `go 1.27.0`) | — |
| C compiler (CGO, tree-sitter) | build/tests | ✓ | Apple clang 21.0.0 | — |
| git | fixtures (`gitInitRepo`) + VCS stamp | ✓ | 2.55.0 | — |
| Private COBOL fork modules (cached) | COBOL extractor | ✓ | preprocessor `v0.0.0-20261002100409-e64234cc493c`; forest-shim `v0.0.0-20260930215433-f9eaf99c34a9` | — (CI needs `COBOL_PARSER_READ_PAT`, P2 D-13) |
| Neo4j | — | not needed | — | Projection is not exercised by this phase's tests |

**Missing dependencies with no fallback:** none. **Probe:** `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolBreadth_CallsAndCopy$'` → `ok … 5.516s`, and `GOWORK=off go test -count=1 ./internal/resolver -run 'CrossRepo'` → `ok … 2.329s` (both this session).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + testify v1.11.1 + zap observer |
| Config file | none (build tag `darwin \|\| linux` on COBOL tests) |
| Quick run command | `GOWORK=off go test -race -count=1 ./internal/resolver -run 'Cobol\|CrossRepo'` |
| Full suite command | `GOWORK=off go test -race ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ID-03 | Encoded program components; no alias between `'A/B'` and nested `A/B`, or `'X#2'` and the 2nd `X` | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_ProgramIDEncoding'` | ❌ Wave 0 |
| ID-04 | Shared resource FilePath = ID, writer-independent Meta/Name; alias edge; GC | unit + real path | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_SharedResource'` ; `... ./internal/resolver -run 'TestCobolReconcile_(Alias\|ResourceGC)'` | ❌ |
| ID-05 | Distinct lines give distinct edges; same line gives distinct `cobol_statement_symbols` | unit | `... ./internal/parser/languages -run 'TestCobolGrammar_Occurrences'` | ❌ |
| ID-06 | Clean×2 and order A,B vs B,A give equal digests | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolLifecycle_CleanDeterminism'` | ❌ |
| ID-07 | Edit, delete, restore: equal to clean; no `restub_prev_*`; byte-identical restore rows | real path | `... ./internal/indexer -run '^TestCobolLifecycle_(Edit\|Delete\|Restore)'` | ❌ |
| ID-08 | Extractor/parser identity change and config change restage unchanged bytes; stale `prov_*` removed | real path + unit | `... ./internal/indexer -run '^TestCobolLifecycle_(Version\|Config)\|TestCobolMaterialFingerprint'` | ❌ |
| ID-09 | All D-04 sequences equal clean (incl. cross-file, cross-repo, interrupted, > cap) | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolLifecycle_'` | ❌ |
| SEC-01 | One allow-set gives the same node set across projection, resolution, and source read | integration (MCP relay) | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolScope_CrossSurface'` ; `... ./internal/graph -run 'TestScope'` | ❌ |
| SEC-02 | Unbound multi-repo source reads are refused for every source tool | integration | `GOWORK=off go test -race -count=1 ./internal/mcp -run 'TestSourceScope_'` | ❌ |
| SEC-03 | No cross-repo COBOL bind without an allowlist; ambiguous within the allowed set; D-12 generic refuse | unit | `... ./internal/resolver -run 'TestCobolReconcile_Allowlist\|TestCrossRepoResolveAll_Ambiguous'` | ❌ |
| SEC-08 | Zero unowned/misprefixed in the acceptance corpus; strict `Clean`; `index_health` status | unit + store contract + real path | `... ./internal/graph/... -run 'PrefixDiagnostics'` ; `... ./internal/indexer -run '^TestCobolLifecycle_Ownership'` | ⚠️ update existing |
| D-08 | > cap escalates (Info log, all files resolved) | unit | `GOWORK=off go test -race -count=1 ./internal/indexer -run 'TestAffectedBy_Cap'` | ⚠️ rewrite `TestAffectedBy_CapBoundsFanout` |
| Regression | Phase 2/3 real-path suites still green | real path | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobol(Tracer\|Breadth)_'` | ✅ (assertions pin `/4` → update to `/5`) |

### Sampling Rate
- **Per task commit:** the quick command plus the package under edit.
- **Per wave merge:** `GOWORK=off go test -race -count=1 ./internal/resolver ./internal/indexer ./internal/graph/... ./internal/parser/languages ./cmd/gortex -run 'Cobol|CrossRepo|PrefixDiagnostics|Scope|AffectedBy'`
- **Phase gate:** `GOWORK=off go test -race ./...` must be fully green before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `internal/testutil/graphdump/graphdump.go` + `graphdump_test.go`: Dump, Volatile, FirstDiff, BLAKE3 digest
- [ ] `internal/indexer/cobol_lifecycle_test.go`: fixtures (invented COBOL, two repos, a copybook included by > cap programs generated in a loop), clean-determinism test. It is expected RED until the CR-02 fix, which documents the baseline.
- [ ] Spike: confirm `incrementalResolveFilesHook` + `ReconcileRepoCtx` produce a real partial state then a full retry (Pitfall 7).
- [ ] Spike: confirm which tail sites a COBOL save actually reaches (`incrementalCatchupHook`).
- [ ] Update pinned `/4` assertions in `cmd/gortex/cobol_tracer_test.go`, `cobol_breadth_test.go`, and `internal/parser/languages/cobol_grammar_contract_test.go` when the extractor bumps.

## Security Domain

`security_enforcement` is enabled (ASVS L1, block on high).

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Architecture | yes | One scope type (D-16) used at every trust boundary |
| V2 Authentication | no | Unchanged (daemon socket/HTTP auth is out of scope) |
| V3 Session Management | partial | MCP session binding (`sessionScope`) decides bound vs. unbound. Do not change binding. |
| V4 Access Control | **yes** | D-13 fail-closed source gate. `Scope.Allows` rejects unowned nodes. The D-09 allowlist is directed and has no wildcards. |
| V5 Input Validation | **yes** | `validateWorkspaceSchema` for `index.cobol`: repo prefixes must be concrete (no `*`), library paths repo-relative with no `..` and not absolute, defaults non-empty after trim. `cobolIDSegment` for every name-derived ID segment (IN-03). |
| V6 Cryptography | limited | BLAKE3 is used only for fingerprints and digests (integrity/equality, not secrecy). Never hand-roll. |
| V7 Error/Logging | yes | Info logs carry counts and repo prefixes only, never source text. Refusal messages must not reveal out-of-scope node existence (reuse "symbol not found" parity, tools_coding.go:959-962). |
| V12 Files/Resources | yes | The source gate sits before `guardSymlinkWithinRepo`. Library paths are confined to the repo root. |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unbound MCP session reads another repo's source (fail-open RepoAllow nil) | Information disclosure | D-13 gate at `resolveNodePath`/`resolveFilePath`/`readViewFile`. Inventory test over every source tool. |
| Same-named program/copybook in an unrelated repo silently bound | Tampering / Spoofing | D-09 directed allowlist, exactly-one inside the allowed set, reverse-fan-out recompute. D-12 refuse for generic. |
| Literal PROGRAM-ID forging an ID path segment or ordinal | Spoofing / Tampering | `cobolIDSegment` on every program component (IN-03) |
| Restub to a bare name rebinds a COBOL call through the generic resolver (CR-01) | Tampering | COBOL restub to the retained `cobol_placeholder`; the `cobol::` namespace stays owned |
| Stale observations survive version or config change | Repudiation / integrity | D-06/D-07 fingerprint plus restage; the harness asserts removal |
| Wildcard or untracked allowlist entry widens scope | Elevation of privilege | Config validation rejects wildcards. Unknown repo prefixes are ignored with a Warn and never treated as "all". |
| Escalation removes the cap, so a huge re-resolve can be triggered | DoS | Accepted by D-08 (correctness over speed). Log at Info with counts so it is observable. |
| New meta keys leaking secret-looking names into Neo4j | Information disclosure | Check them against `sensitiveMetadataKey` (Pitfall 10) |

## Sources

### Primary (HIGH confidence; files read in this session)
- `internal/indexer/extractor_version.go` (salt, stale langs, `"cobol": 2`, ext map)
- `internal/parser/languages/cobol_grammar.go`, `cobol_grammar_identity.go`, `cobol_grammar_refs.go`, `cobol_grammar_walk.go`, `cobol_grammar_dedupe.go`
- `internal/resolver/cobol_references.go`, `cross_repo.go`, `resolver.go` (ResolveAll settle window, `ResolveFilesAndIncoming`), `terminal.go`
- `internal/indexer/affected_by.go`, `incremental_batch.go`, `incremental_watcher_batch.go`, `indexer.go` (applyRepoPrefix, restub, ResolveAll), `multi.go`, `workspace_resolve.go`, `index_state.go`, `source_revision.go`, `incremental_reindex_test.go`, `affected_by_e2e_test.go`
- `internal/graph/prefix_diagnostics.go`, `index_state.go`, `restub_provenance.go`, `edge_terminal_stamps.go`, `edge.go`, `stub.go`, `scoped_projection.go`, `storetest/storetest.go`
- `internal/graph/store_sqlite/scoped_projection.go`, `schema_version.go`, `migration_unprefixed_purge.go`, `add_batch_set.go`, `file_batch_evict.go`, `meta_json.go`
- `internal/mcp/scope_resolve.go`, `tools_core.go`, `tools_coding.go`, `tools_fileops.go`, `server.go`, `tools_enhancements.go`, `tools_neo4j.go`
- `internal/config/config.go`, `internal/serverstack/shared_server.go`, `cmd/gortex/mcp.go`, `cmd/gortex/daemon_state.go`, `cmd/gortex/cobol_tracer_test.go`, `cobol_breadth_test.go`
- `internal/neo4jprojection/service.go`, `properties.go`; `internal/testutil/graphfixture/fixture.go`
- Planning: 04-CONTEXT.md, REQUIREMENTS.md, STATE.md, 03-CONTEXT.md, 03-VERIFICATION.md, 03-REVIEW.md, 02-CONTEXT.md, 02 deferred-items.md

### Secondary (MEDIUM)
- Test probes run this session (COBOL breadth test, resolver cross-repo tests); `go list -deps` import check.

### Tertiary (LOW)
- None. No web sources were needed. Every recommendation not backed by a quoted file is tagged [ASSUMED] in the Assumptions Log.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH. Nothing new, and every dependency is verified in `go.mod` or in code.
- Architecture: MEDIUM. The mechanisms are grounded in read code but not prototyped. Two spikes are listed in Wave 0.
- Pitfalls: HIGH. Each is tied to a specific line read this session.

**Research date:** 2026-10-12
**Valid until:** 2026-11-11 (the codebase moves fast; re-verify line numbers if upstream is merged)
