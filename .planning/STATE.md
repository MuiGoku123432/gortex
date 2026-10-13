---
gsd_state_version: "1.0"
milestone: v1.0
current_phase: 04
current_phase_name: Stable Incremental and Cross-Repository Lifecycle
status: executing
stopped_at: Completed 04-20-PLAN.md
last_updated: "2026-10-13T21:31:55.760Z"
last_activity: 2026-10-13
last_activity_desc: Phase 04 execution started
state_head: 13d9d1eb6e66c9801fcf3016d6b93f66e1b21476
progress:
  total_phases: 10
  completed_phases: 2
  total_plans: 52
  completed_plans: 50
  percent: 20
milestone_name: Neo4j Projection and Mainframe Graph Foundation
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-21)

**Core value:** A trustworthy graph representation of a mainframe estate that modernization cutover decisions can be made against -- deterministic and reproducible first, enriched and simulated later.
**Current focus:** Phase 04 — Stable Incremental and Cross-Repository Lifecycle

## Current Position

Phase: 04 (Stable Incremental and Cross-Repository Lifecycle) — EXECUTING
Plan: 3 of 22
Status: Ready to execute
Last activity: 2026-10-13 — Phase 04 execution started

Progress: [██░░░░░░░░] 20%

## Performance Metrics

**Velocity:**

- Total plans completed: 13
- Average duration: -
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 7 | - | - |
| 02 | 6 | - | - |

**Recent Trend:**

- Last 5 plans: none
- Trend: No execution data

*Updated after each plan completion*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P01 | 6 min | 2 tasks | 5 files |
| Phase 01 P02 | 10 min | 2 tasks | 5 files |
| Phase 01 P03 | 13 min | 2 tasks | 8 files |
| Phase 01 P04 | 8 min | 2 tasks | 6 files |
| Phase 01 P05 | 17 min | 2 tasks | 8 files |
| Phase 01 P06 | 38 min | 2 tasks | 7 files |
| Phase 01 P07 | 37 min | 2 tasks | 11 files |
| Phase 02 P01 | 30 min | 2 tasks | 5 files |
| Phase 02 P02 | 23 min | 2 tasks | 6 files |
| Phase 02 P03 | 12min | 2 tasks | 4 files |
| Phase 02 P04 | 5min | 2 tasks | 5 files |
| Phase 02 P05 | 4min | 2 tasks | 7 files |
| Phase 02 P06 | ~17h (checkpoint wait) | 3 tasks | 14 files |
| Phase 03 P01 | 8 min | 2 tasks | 6 files |
| Phase 03 P02 | 10min | 2 tasks | 3 files |
| Phase 03 P03 | 11 min | 2 tasks | 5 files |
| Phase 03 P04 | 9 min | 2 tasks | 7 files |
| Phase 03 P05 | 14 min | 2 tasks | 6 files |
| Phase 03 P06 | 9 min | 2 tasks | 3 files |
| Phase 03 P07 | 10 min | 2 tasks | 4 files |
| Phase 03 P08 | 11 min | 2 tasks | 5 files |
| Phase 03 P09 | 17 min | 2 tasks | 4 files |
| Phase 03 P10 | 25 min | 2 tasks | 7 files |
| Phase 03 P11 | 8 min | 2 tasks | 7 files |
| Phase 03 P12 | 1 min | 2 tasks | 0 files |
| Phase 03 P13 | 24 min | 2 tasks | 7 files |
| Phase 03 P14 | 9 min | 2 tasks | 6 files |
| Phase 03 P15 | 7 min | 2 tasks | 7 files |
| Phase 03 P16 | 9 min | 2 tasks | 6 files |
| Phase 03 P17 | 26 min | 3 tasks | 8 files |
| Phase 04 P01 | 25 min | 1 tasks | 8 files |
| Phase 04 P02 | 12 min | 2 tasks | 4 files |
| Phase 04 P03 | 5 min | 1 tasks | 6 files |
| Phase 04 P04 | 6 min | 2 tasks | 2 files |
| Phase 04 P06 | 6min | 1 tasks | 4 files |
| Phase 04 P11 | 6 min | 1 tasks | 6 files |
| Phase 04 P12 | 19 min | 1 tasks | 7 files |
| Phase 04 P05 | 49 min | 2 tasks | 12 files |
| Phase 04 P14 | 9min | 2 tasks | 6 files |
| Phase 04 P07 | 13 min | 2 tasks | 7 files |
| Phase 04 P13 | 6 min | 1 tasks | 2 files |
| Phase 04 P08 | 13 min | 1 tasks | 7 files |
| Phase 04 P15 | 23 min | 2 tasks | 7 files |
| Phase 04 P09 | 15 min | 1 tasks | 5 files |
| Phase 04 P16 | 20 min | 2 tasks | 9 files |
| Phase 04 P10 | 28min | 3 tasks | 6 files |
| Phase 04 P17 | 45min | 2 tasks | 8 files |
| Phase 04 P18 | 24min | 2 tasks | 6 files |
| Phase 04 P19 | 19min | 3 tasks | 3 files |
| Phase 04 P20 | 35min | 2 tasks | 5 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Phase 1]: The immediate priority is a general, language-agnostic CLI/MCP operation that projects an explicitly scoped SQLite snapshot into Neo4j.
- [Phase 1]: SQLite remains sole authority; projection is manual and rebuildable, with no index-time dual-write, reverse writes, or Neo4j dependency for ordinary Gortex operations.
- [Phase 2]: Parser behavior must be reproducibly equivalent to baseline `97ac9f1`; the tracer resumes only when parser/grammar integration is ready.
- [Phase 5]: `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` remain distinct deterministic findings.
- [Phases 7-9]: AI-facing work requires the AI-SPEC workflow before implementation planning; existing artifacts must be renumbered by the orchestrator.
- [Phase 01]: Keep graphfixture independent of store_sqlite so downstream same-package tests can open the fixture without an import cycle.
- [Phase 01]: Register future real-server scenarios now, and make the final mandatory gate fail while any remain skipped.
- [Phase ?]: Keep resolved Neo4j credentials invocation-local and excluded from YAML serialization.
- [Phase ?]: Add a separate immutable snapshot opener capability instead of widening ScopedProjectionSequencer.
- [Phase 01]: Keep the official Neo4j driver invocation-local behind a narrow transport lifecycle; ordinary Gortex startup remains Neo4j-free.
- [Phase 01]: Stage generation-specific physical records and make only one exact-owner manifest pointer switch visible.
- [Phase ?]: Hash length-delimited identity components so exact scope, provenance, occurrence, and generation boundaries cannot alias through concatenation.
- [Phase ?]: Filter normalized secret-like metadata keys before reversible key encoding, and omit non-finite or unsupported values with deterministic warning counts.
- [Phase ?]: Expose callback-based snapshot pages so SQLite cursors close before transport work while one read transaction preserves a coherent source view.
- [Phase ?]: Count and stage nodes before edges in batches capped at 500 records, with progress emitted only after successful transport calls.
- [Phase 01]: Treat manifest activation as logical completion, then report physical reconciliation independently.
- [Phase 01]: Use operation plus pending generation as the exact-owner idempotent lock identity.
- [Phase 01]: Normalize CLI and MCP input through one neo4jprojection.Request contract before profile resolution, snapshot reads, or driver construction.
- [Phase 01]: Classify neo4j_push as an external write while retaining dry-run without a second confirmation field.
- [Phase 02]: Fork commit f9eaf99c34a92db6f5487d316cf9977295a62a7d (parent f19029f, floor 97ac9f1) is pushed and is the commit Plan 02-02 pins; it adds AttestEmbedded, NewEmbeddedAnalyzer, and NewContentID.
- [Phase 02]: Embedded attestation drops only the on-disk checkout hash read; compiled-language identity comparison and ToolID are unchanged.
- [Phase 02]: 02-02: Pinned fork preprocessor and forest-shim/cobol replace at v0.0.0-20260930215433-f9eaf99c34a9 (commit f9eaf99); prov_parser_module records the replacement path@version from build info.
- [Phase 02]: 02-02: CobolGrammarExtractor (register.go:119) emits file + KindFunction program (cobol_kind=program) + EdgeDefines with the prov_* contract; unapproved grammar or failed attestation fails extraction with no fallback.
- [Phase 02]: 02-02: Pre-existing go mod tidy drift (neo4j-go-driver indirect to direct) and the BASE-02 CI step's missing stock go.sum lines are deferred to Plan 02-05 (deferred-items.md).
- [Phase 02]: 02-03: COBOL program IDs are relPath::OUTER/INNER with #k (k>=2) ordinals from an END PROGRAM name stack; grammar Parent links are flat; line numbers never enter an ID
- [Phase 02]: 02-03: Copybooks (.cpy/.CPY) skip Analyze and carry only identity + NewContentID revision + prov_analysis_absence=copybook_standalone_analysis_unsupported
- [Phase 02]: 02-03: prov_extractor_version stays gortex-cobol-grammar/1 for the unreleased Phase 2 mapping; bump on first post-phase change
- [Phase 02]: 02-03: '#'-in-ID owner helpers are harmless for KindFunction COBOL programs today; graph.NodeIsTest owner hop is latent (same-file is_test)
- [Phase 02]: 02-04: VCS revision is sampled per stamp call (accuracy over reuse); ls-files uses --literal-pathspecs; timeouts map to git_unavailable; stamp deletes the opposite key so both never coexist
- [Phase 02]: 02-05: Every Go-building workflow uses GOPRIVATE, setup-go cache: false, and a fail-closed COBOL_PARSER_READ_PAT insteadOf step; the release container reads a host-filled module cache read-only with GOPROXY=off and never receives the fork credential
- [Phase 02]: 02-05: Neo4j go mod tidy drift left open (go.mod out of scope); ci.yml lint tidy check will fail until the one-line tidy result is committed (deferred-items From Plan 02-05)
- [Phase 02]: 02-06: D-13 CI checkpoint accepted with exception — 12/13 checks green on d4e0b3b7; windows-latest test shard fails with undiagnosed cause (log needs repo admin); D-17 Windows stub remains unverified
- [Phase 02]: 02-06: BASE-02 marked complete (OS-independent text, proven by ubuntu stock-grammar fail-closed step); pre-existing lint findings and three dropped-error bugs fixed (9f21e855, d4e0b3b7)
- [Phase 03]: Phase 3 breadth runs before the document-flag block so document-only flags stay on the file node
- [Phase 03]: PERFORM edges record cobol_name_norm=cobol-name-norm/1; labels bind only on exactly one same-program match
- [Phase 03]: Paragraph and section extents end at the last procedure-division child before the ending header
- [Phase 03]: 03-02: alignProvByteOffsets corrects provenance byte offsets in the D-18 hook (+3 only on an exact UTF-8 BOM plus indexed bytes match; any other mismatch marks prov_range_exact=false with indexed_bytes_differ_from_file); rows and columns stay BOM-excluded
- [Phase 03]: 03-02: stampSourceRevision stamps prov_vcs_commit or prov_vcs_commit_absence on relationship edges carrying prov_revision_content_id, same value as the file's nodes
- [Phase 03]: 03-03: cobolDedupeEdges moved to cobol_grammar_dedupe.go to keep cobol_grammar_walk.go under 300 lines
- [Phase 03]: 03-03: the procedure division observation carries container=division node and flowFrom=program, the fallback for span-placed COPY
- [Phase 03]: 03-03: copy_statement is always placed by generated span; a COPY outside every procedure division span gets no statement node
- [Phase 03]: 03-04: unresolved::cobol:: is an owned namespace; resolveCobolReferences binds program/copybook targets only to exactly one same-repo candidate (same-file first for programs), else stamps missing or ambiguous with sorted cobol_candidate_ids
- [Phase 03]: 03-04: resolver compares cobol_name_key by exact string equality; extractor plans must emit keys already normalized under cobol-name-norm/1
- [Phase 03]: 03-05: Data levels outside D-06 (78, 50, ...) and CONSTANT entries are counted in cobol_unleveled_entry_count, never placed by guess
- [Phase 03]: 03-05: A data section's COPY window runs to the next non-COPY observation, because the grammar parents a trailing COPY to the division or program
- [Phase 03]: 03-05: Data names resolve only through cobolDataReference (owner-local, cobol-name-norm/1, IN/OF qualifiers as an inner-to-outer ancestor subsequence); item #k skips program-reserved symbols
- [Phase 03]: 03-06: cobolFlow(w) rebuilds the per-division label table from header observations (w.container of a header is its node symbol) instead of adding a cobolWalk field; own section is tracked by label index; THRU unresolved edges report the first failing endpoint; qualified labels never fall back to a program-wide match
- [Phase 03]: 03-07: COPY IDMS detection mirrors the grammar's _COPY_IDMS token (COPY [ \t]+ IDMS, ASCII fold); EXEC SQL INCLUDE member is read only from sql_include_name because INCLUDE is aliased as a WORD child
- [Phase 03]: 03-07: cobolCallTarget returns every edge it emits; FUNCTION targets are dynamic, keyed by text, with no data reference; one span-based From rule covers all COPY-family forms
- [Phase 03]: 03-08: IDMS/CICS resources are shared KindResource nodes emitted once per document via w.resourceNode (prov_range_absence=shared_resource, no structural edge); qualifiers come only from source (IDMS always _unqualified, CICS map qualified only by a same-statement literal MAPSET)
- [Phase 03]: 03-08: IDMS edges classified by verb observation kind with idms_verb kept as written; CICS PROGRAM operands reuse cobolCallTarget, data-name TRANSID/MAP/MAPSET go to per-program dynamic targets plus cics_<role> data references, never bound
- [Phase 03]: 03-08: cobolNameNorm/cobolIDSegment/cobolNameNormRule moved to cobol_grammar_identity.go and cobolUnquote split out of cobolNameNorm (walk line budget; literal resource names keep their case)
- [Phase 03]: 03-09: DB2 tables are KindTable db2::<schema|_unqualified>/<table> named by sql_source_candidate only; aliases, CTE names, derived tables, cursors, and dynamic sources never become tables
- [Phase 03]: 03-09: SQL cursor names are read from statement text after the verb (underscore identifiers are hidden tokens), skipping orientation keywords, host variables, and numbers
- [Phase 03]: 03-09: the pinned grammar admits no EXEC SQL statement in WORKING-STORAGE and leaves INSERT...SELECT sources in the unparsed tail; neither is guessed
- [Phase 03]: COBOL embedded-SQL KindTable nodes (Language cobol) survive the default-off sql coverage gate like migration and ORM tables; without this every DB2 table and SQL edge was deleted before SQLite
- [Phase 03]: serveCobolTestDaemon is the shared daemon-over-relay helper for COBOL tracer and breadth tests
- [Phase 03]: 03-11: Local fork commit e64234cc493c487393c5e7867adfd7ca92d936ba (parent f9eaf99, NOT pushed) adds transform.RunCopybook/CopybookConfigurationID (rule copybook-wrapper/v1), grade.PolicyCopybookV1 (cobol-grade-policy-copybook-v1, AnchorsByConstruction), and Analyzer.AnalyzeCopybook; Plan 03-12 owns the push
- [Phase 03]: 03-11: Copybook final-newline insert is a separate RuleCopybookWrapper occurrence; analyze() keeps its signature for quality_census and analyzeWith carries the transform parameter
- [Phase 03]: Phase 03-12: User authorized the D-16 fork push (verbatim "push"); e64234cc493c487393c5e7867adfd7ca92d936ba is on origin gsd/v0.26.0-estate-parse-recovery as a fast-forward from f9eaf99, no force
- [Phase 03]: 03-13: Gortex preprocessor re-pinned to v0.0.0-20261002100409-e64234cc493c (fork e64234c); forest-shim replace and cobolApprovedGrammarID unchanged; .cpy files run AnalyzeCopybook behind the same grammar check
- [Phase 03]: 03-13: copybooks own their data items (<path>::<01>/<child>, member coordinates); the copybook walk uses a cloned docMeta so file-node COPY marks never leak; REDEFINES/RENAMES pass shared via cobolDataBind
- [Phase 03]: 03-14: COBOL nested-ness is an explicit cobol_nested Meta stamp from END PROGRAM containment; resolveCobolReferences offers nested programs only to same-file callers and never infers nesting from ID/QualName shape
- [Phase 03]: A copybook walk registers the producer's wrapper program as its root (cobolCopybookRoot "@proc"); cobolWalk.id maps it to the file node, so cobolReferences/cobolCopy emit COPY edges from the copybook file node unchanged
- [Phase 03]: 03-16: Copybook procedure IDs are rooted at @proc (<path>::@proc/<PARA>); cobolProcedure root mode emits no wrapper division and the copybook file node is the procedure container
- [Phase 03]: 03-16: A label a copybook does not define goes to unresolved::cobol::label::<path>::<key> with label_not_in_copybook (only label_not_found is rewritten), never bound across files
- [Phase 03]: 03-17: IDMS, CICS, and DB2 walkers build From/owner IDs through w.id; cobolCopybook runs cobolMainframe and cobolSQL, so copybook statements emit edges From the paragraph or the copybook file node
- [Phase 03]: 03-17: a data name a copybook does not declare is not_declared_in_copybook (keyed on w.root); program documents keep not_declared_in_program
- [Phase 03]: 03-17: WR-04 stays a producer window; the copybook file node records cobol_copybook_wrapper and, when a non-green analysis mapped nothing, prov_analysis_absence=copybook_content_unmapped
- [Phase 04]: 04-01: Shared COBOL resource nodes keep only nine writer-independent prov keys (cobolSharedResourceKeys); FilePath is the node ID and Name the normalized key; the as-written spelling rides on edges as cobol_resource_raw (D-18) — Program and copybook documents write the same tool ID and handoff schema but a different grade policy and transform config, and the last-writer-wins node upsert made clean builds nondeterministic
- [Phase 04]: 04-01: graphdump DeterministicVolatile (graphdump-volatile/1) ships empty; add a key only when a harness diff proves it volatile — Two clean builds and opposite track orders matched with no exclusions (D-01)
- [Phase 04]: 04-02: literal PROGRAM-ID quotes stay in IDs; only ID metacharacters are cobolIDSegment-encoded ('A/B' -> {path}::'A%2FB')
- [Phase 04]: 04-02: cobol_placeholder marks bindable COBOL edges (literal CALL/LINK/XCTL, non-IDMS COPY); its absence marks never-bound edges
- [Phase 04]: 04-03: graph.Scope is the one fail-closed scope type (D-16); ProjectionScope is an alias; Allows/AllowsRepo reject an empty RepoPrefix, the zero Scope, and an exact WorkspaceID/ProjectID mismatch (no RepoPrefix fallback)
- [Phase 04]: 04-03: neo4jprojection.NormalizeRequest fails closed on a blank repository entry instead of dropping it; OpenScopedProjectionSnapshot validates via graph.NewScope + RequireWorkspaceProject with the original error substrings
- [Phase 04]: 04-04: generic cross-repo fallback refuses multi-repository matches via pickCrossRepoCandidate (lowest ID within one repo, synthetic no-repo first); resolution_* keys cleared centrally in resolveEdge on bind
- [Phase 04]: 04-06: reconcileCobol(changed) is the single COBOL binding pass (full via r.scope, incremental via ReconcileCobol); re-decides bound edges from cobol_placeholder, candidates through per-caller graph.Scope, empty RepoPrefix caller -> caller_unowned, diff-only writes
- [Phase 04]: 04-06: cobolPolicyFor carries //nolint:unused until 04-14 reads ResolveFrom; 04-14 must remove it
- [Phase 04]: gortex-cobol-grammar/5 is the single Phase 4 COBOL mapping bump (04-01 shared resources + 04-02 identity); later Phase 4 extractor changes keep /5 until the phase gate
- [Phase 04]: 04-12: source gate resolves scope with IntentReach (bound sessions keep workspace-wide source); a resolved WorkspaceID with empty RepoAllow expands to that workspace's repos
- [Phase 04]: 04-12: resolveFilePath/resolveNodePath/resolveGraphPath and committed-view reads carry requireSourceScope; graphRelPath and workspace-edit ownership use ungated resolveConfinedFilePath (graph keys only); rename_symbol refuses whole when any site is gated
- [Phase 04]: 04-05: Ownership exemptions are three enumerated Go lists (OwnershipExemptKinds/ExemptPathPrefixes/GlobalIDNamespaces) that also generate the SQLite predicates; strict Clean fails on any unowned node (D-14)
- [Phase 04]: 04-05: purgeUnprefixedRepoRows (v7, re-run as v22) deletes only the audit's unowned file-backed rows so external::-path modules and workspace rationale survive
- [Phase 04]: 04-14: COBOL cross-repository binding is a directed ResolveFrom allowlist with one candidate union (same-repository never shadows an allowed repository); copybook libraries rank by first matching library and record cobol_copy_search_rank
- [Phase 04]: 04-14: reconcileCobolResources evicts shared resources (FilePath == ID) with none or alias-only in-edges, then keeps one cobol-default-qualifier/1 alias per _unqualified resource to an existing qualified node; default qualifiers compare as configured, so 04-07 must pass them normalized
- [Phase 04]: 04-07: COBOL policy lookup treats a repository as tracked when registered or holding a live mutation lane, read per call; TrackRepoCtx registers after its own resolve pass, so a registered-only check loses crossa->crossb bindings
- [Phase 04]: 04-07: index.cobol library names and default qualifiers must already be cobol-name-norm/1 keys; an invalid index.cobol block rejects the whole .gortex.yaml read and keeps the last good parse
- [Phase 04]: 04-13: embedded gortex mcp --index indexes through MultiIndexer.TrackRepoCtx (indexEmbeddedRepo) and watches through mi.GetIndexer(prefix); no watcher without an indexed repo; nil MultiIndexer is an error (D-14, SEC-08) — The embedded path was the last writer of unowned nodes; a workspace of one keeps the strict 04-05 audit clean
- [Phase 04]: 04-15: ReconcileRepoCtx reconciles COBOL unconditionally at the end of every warm restart, so an interrupted reindex (mtimes already persisted) converges
- [Phase 04]: 04-15: Resolver.reconcileCobol runs DetectCrossRepoEdgesForReindexes on its rebound batch, so a cross-repository COBOL CALL carries cross_repo_calls on clean and incremental paths alike
- [Phase 04]: 04-15 (D-08): above AffectedByReresolveMax both affected-by sites log 'affected-by: cap exceeded, resolving full affected set' at Info and resolve the full set
- [Phase 04]: 04-09: COBOL material fingerprint = 48-bit BLAKE3 of CobolGrammarExtractor.MaterialIdentity plus the index.cobol JSON, stored as reserved key _cobol_material in RepoIndexState ExtractorVersions (no schema migration). Inequality marks cobol stale on the next full-root reconcile. The cobol extractorVersions entry is removed. The MCP freshness rider keeps the global snapshot. — D-06/D-07: the stamp and the invalidation trigger share one source, and only index.cobol (not the whole config hash) invalidates COBOL
- [Phase 04]: 04-16: In Merkle mode the persisted leaf salt (|cobol-material@fp) is the record of the COBOL material fingerprint; the RepoIndexState _cobol_material row is read and re-stamped only on the non-Merkle path
- [Phase 04]: 04-16: HEAD-move re-stamp (D-17) runs in reconcileRepoIndexState when HEAD differs from the stored IndexedSHA, grouping rows by prov_source_path and deciding from transforms.run bytes; graphdump-volatile/3 renders an emptied Meta as null
- [Phase 04]: 04-10: TestCobolScope_CrossSurface calls get_symbol_source from the directory above both checkouts so only the explicit repo argument narrows the read; from repository A's checkout the session binding alone already excluded B
- [Phase 04]: 04-10: callCobolTracerTool delegates to callCobolToolResult (text, isError) so tests read tool-error results without a second relay helper
- [Phase 04]: 04-10: SEC-03 placeholder collision closed by proof - the shared unresolved dynamic CALL target has no node and each edge keeps its repository-prefixed From
- [Phase 04]: 04-17: every source surface admits paths only via requirePathSourceScope (lexical + physical owner); gate is independent of the 04-18 walk fix because pre-tracked duplicate nodes persist
- [Phase 04]: 04-18: Nested tracked roots are pruned in shouldExclude, admitWalkFileKnownType's regular-file fast path, and Watcher.isExcluded; config roots seeded once per Indexer, registered roots read live under mi.mu.RLock (audited: no caller holds mi.mu)
- [Phase 04]: 04-18: Kept the live RLock read over pushing roots to Indexers at publication points — the watcher needs later-tracked roots on already-built Indexers
- [Phase 04]: 04-19: explore refuses unbound multi-repo artifact-probe requests from task text alone (D-13) and admits lane matches through filterTextMatchesByResolvedScope (D-16)
- [Phase 04]: 04-19: source-channel inventory is a parse-level test (Grep* calls + filesystem walks keyed by enclosing function)
- [Phase 04]: 04-20: nested-root pruning reads only live MultiIndexer state (mi.repos + reference-counted mi.pendingRoots declared by indexMultiRepo and the daemon warmup via DeclarePendingRoots) under one mi.mu.RLock; the per-Indexer config copy is removed (CR-01, WR-01)

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 1]: Exact command name, connection/driver syntax, and default stale-record deletion policy remain discuss-phase decisions; requirements define behavior, not those details.
- [Phase 1]: Projection design must prove stable relationship keys, Neo4j constraint support, bounded retry behavior, and scope-safe stale-record reconciliation.
- [Phase 2]: COBOL parser/grammar readiness now available (2026-09-30) from tree-sitter-cobol-upgrade v0.26.0 -- see .planning/research/COBOL-PARSER-READINESS.md for setup (go.work is currently stale and must be recreated), the handoff API, and the Gortex mapping. Phase 2 can be planned.
- [Phase 4]: Copybook/library search order and canonical cross-repository resolution domains need corpus evidence.
- [Phases 7-9]: Provider authorization, retention, claim-ledger, reviewer authority, and evaluation thresholds remain policy/design decisions.
- [Phase 2]: windows-latest CI test shard fails (runs 36877578916, 36889132850); cause unknown pending job-log access — see 02 deferred-items From Plan 02-06

### Roadmap Evolution

- Phase 6.1 inserted after Phase 6: Deterministic VB.NET extraction via pinned MuiGoku123432/tree-sitter-vbnet grammar; placed after the deterministic gate, before AI phases

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Storage | Continuous Neo4j synchronization or Neo4j authority | Deferred; v1.0 includes only explicit scoped projection | v1.0 reprioritization |
| COBOL graph | Tracer and grammar-dependent extraction phases | Deferred until parser/grammar readiness | v1.0 reprioritization |
| Mainframe breadth | Full JCL symbolic resolution and behavioral/live twin | Future milestone | v1.0 planning |

## Session Continuity

Last session: 2026-10-13T21:31:55.702Z
Stopped at: Completed 04-20-PLAN.md
Resume file: None
