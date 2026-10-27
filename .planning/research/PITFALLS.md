# Domain Pitfalls

**Domain:** Scoped SQLite-to-Neo4j projection, then deterministic COBOL graph extraction with optional AI enrichment
**Milestone:** v1.0 Neo4j Projection and Mainframe Graph Foundation
**Researched:** 2026-09-15; reconciled 2026-09-21
**Overall confidence:** HIGH for repository-specific and manual-projection risks; MEDIUM for intentionally open projection transport/UX details and any future continuous synchronization design

The central failure mode is a graph that looks precise while mixing different identities, revisions, repositories, evidence classes, or storage generations. This milestone should optimize for falsifiability: every active fact must say what produced it, from which source revision and parser build, and why it is still active.

## Critical Pitfalls

### Pitfall 1: Identity collisions collapse distinct COBOL artifacts

**What goes wrong:** Programs, paragraphs, copybooks, DB2 tables, CICS resources, IDMS records, unresolved targets, findings, or claims that share a short name become one node. The current COBOL extractor demonstrates the danger: its local ID is `filePath::name`, its `seen` map drops repeated names within a file, paragraphs and programs both use `KindFunction`, and unresolved calls/imports use name-only namespaces. A globally unique-looking hash does not repair an underspecified semantic key.

**Why it happens:** Mainframe identity is contextual. A member name can repeat across repositories and libraries; paragraph names repeat across programs; tables can repeat across schemas; CICS names can vary by region; a source occurrence is not the same thing as the domain artifact it mentions. Case folding, hyphen normalization, aliases, generated source, and copybook expansion add more collision paths.

**Consequences:** False joins, lost definitions, edges attached to the wrong target, cross-repository contamination, unstable IDs after moves, and silent overwrite during SQLite upsert or Neo4j load. Human review cannot recover facts that were merged before provenance was retained.

**Prevention:**
- Define separate canonical-key contracts before broad extraction: repository/retrieval identity, source occurrence, domain artifact, deterministic relationship, unresolved finding, AI claim, and review.
- Extend Gortex's repo-prefixed identity instead of introducing an unrelated ID system. Include the minimum namespace required by each domain: program owner for paragraphs, library/member for copybooks, subsystem/schema for resources, and workspace/project/repo for source-owned facts.
- Keep canonical domain identity separate from occurrence identity. A source range may move while the referenced program remains the same.
- Normalize names once with an explicit dialect policy, while preserving the raw spelling and exact source range.
- Never use mutable attributes such as line number, confidence, review status, or display name as the sole durable identity.

**Detection:**
- Collision corpus containing duplicate program/member/paragraph/resource names across files, libraries, repositories, schemas, and case variants.
- Assert no extraction candidate is discarded only because another node has the same short name.
- Compare expected occurrence count, canonical-entity count, and collision groups separately.
- Run identical indexing twice and compare sorted active node/edge identity sets, not only aggregate counts.

### Pitfall 2: Incremental indexing leaves stale facts active

**What goes wrong:** Editing or deleting a COBOL statement removes the source evidence but the previous node, edge, unresolved finding, derived edge, AI claim, or Neo4j row remains active. Re-indexing can also duplicate the new generation instead of superseding the old one.

**Why it happens:** Gortex already has several lifecycle paths: file eviction, scoped global passes, deferred enrichment, cross-repo resolution, watcher reconciliation, and durable pending-enrichment markers. New mainframe facts or claims can bypass one path. Parser, extractor, preprocessing, configuration, and retrieval revisions can invalidate facts even when source bytes do not change.

**Consequences:** Modernization queries return relationships that no longer exist, resolved references coexist with missing-artifact findings, contradicted claims remain in trusted views, and identical reruns grow the graph.

**Prevention:**
- Specify ownership for every fact: exact source file/range or a named synthetic generation whose reconciliation owns all emitted rows.
- Reconcile each changed ownership unit as a set: compute the desired generation, atomically activate it, and close/supersede facts absent from it. Do not implement add-only upserts.
- Include source content/retrieval revision, parser grammar content ID, extractor schema version, and material configuration in the extraction generation fingerprint.
- Invalidate dependent findings and AI claims when any cited evidence or context version changes; preserve lineage but remove them from active/trusted views.
- Treat watcher health and reconciliation as correctness signals. The existing janitor bounds missed filesystem events, but a COBOL lifecycle test must exercise both watch-style updates and cold rebuilds.

**Detection:**
- Golden lifecycle sequence: index A, identical A, edit A→B, delete B, restore A, parser-version change without source change, and interrupted retry.
- After every step assert exact active sets, one active generation per owner, no dangling edges, and correct supersession/closure records.
- Compare an incremental result against a clean rebuild from the same inputs. Any active-set difference is a release blocker.
- Query for active claims whose evidence node/range/revision is inactive or absent.

### Pitfall 3: Confidence is treated as evidence

**What goes wrong:** A high numerical confidence or label promotes an AI inference, heuristic dynamic call, or weak resolver result into the deterministic graph view. Conversely, a certain reference to a missing copybook may be treated as low quality because the target is unknown.

**Why it happens:** The existing edge model and Cypher exporter expose `confidence`, `confidence_label`, and `origin` as adjacent properties, which invites consumers to rank evidence classes on one axis. They are orthogonal: evidence answers how a claim was established; confidence estimates uncertainty within that class.

**Consequences:** Hallucinations become operational truth, deterministic unresolved references disappear from priority lists, and downstream consumers cannot reconstruct why a relationship exists.

**Prevention:**
- Make evidence class mandatory and typed: deterministic, AI-inferred, external-unresolved, parse-unresolved, human-confirmed, contradicted, superseded.
- Define trusted views by evidence/status policy, never by confidence threshold alone. Unreviewed AI claims stay excluded by default regardless of score.
- Keep confidence optional and class-specific. Do not compare parser certainty, heuristic confidence, and model self-confidence as if calibrated on one scale.
- Require supporting graph IDs/source ranges and unavailable-context declarations for material AI claims.

**Detection:**
- Contract tests prove that `AI_INFERRED confidence=1.0` remains outside deterministic and human-confirmed views.
- Contract tests prove that a certain reference to an unavailable target remains `EXTERNAL_UNRESOLVED`, not `PARSE_UNRESOLVED` or low-confidence AI.
- Audit every edge/node returned by trusted queries for a non-empty evidence class and valid provenance.

### Pitfall 4: AI mutates deterministic truth

**What goes wrong:** Model output overwrites parser nodes, retargets deterministic edges, closes parser errors, invents missing artifact contents, or is inserted directly into canonical domain entities.

**Why it happens:** Reusing generic upsert paths is convenient, and a first-class claim layer can look like overhead. Structured output validates shape, not truth. Human confirmation also tempts implementations to rewrite history rather than add a reviewed assertion.

**Consequences:** Indexing ceases to be reproducible, AI-off mode yields a different substrate, model/provider changes alter graph truth, and contradictory evidence cannot be audited.

**Prevention:**
- Put AI behind a one-way claim boundary. The context builder reads deterministic facts/findings; the validator writes claim/review records only.
- Give claims their own stable IDs, evidence links, model/prompt/context versions, status, and supersession/contradiction links.
- Human confirmation creates a `HUMAN_CONFIRMED` assertion while retaining the originating AI claim. It does not rewrite the parser's fact.
- Permit deterministic grammar/extractor improvements to close findings; never permit an AI response to make parser regression gates pass.
- Keep indexing and native queries fully functional with AI disabled.

**Detection:**
- Snapshot the deterministic subgraph before and after AI review and require byte-equivalent identities/provenance.
- Deny AI-write transactions that target deterministic node/edge tables or evidence classes.
- Run the same corpus with AI off and on; deterministic active sets must match exactly.
- Test contradictory model outputs and require separate claims rather than last-write-wins mutation.

### Pitfall 5: Missing artifacts are confused with parse gaps

**What goes wrong:** A missing copybook, called program, JCL member, DB2 catalog, CICS definition, IDMS schema, or preprocessor output is recorded as a parser defect. The inverse is equally harmful: source exists but an `ERROR`, missing node, or unparsed tail is labeled as an unavailable artifact.

**Why it happens:** Both conditions produce an unresolved edge. The parser baseline also proves that low recall can have different causes: `SCHEMA SECTION` is assigned to preprocessing, bare copybooks need fragment handling, and localized IDMS/CICS/SQL tails are grammar coverage signals.

**Consequences:** Teams fix the wrong subsystem, AI is asked to invent unavailable content, parser regressions are hidden as retrieval problems, and gap-impact ranking becomes meaningless.

**Prevention:**
- Decide gap class from evidence availability before interpreting syntax: `EXTERNAL_UNRESOLVED` means a deterministic reference exists but the target artifact is not present; `PARSE_UNRESOLVED` means bytes are present but deterministic interpretation failed.
- Add a third operational reason dimension for decode/preprocess/unsupported-fragment failures without collapsing the two evidence classes.
- Preserve exact source range, containing symbol, expected construct family, parser version, retrieval manifest, and attempted resolution scope.
- Create placeholders for missing targets without inventing target attributes.

**Detection:**
- Intentionally incomplete fixtures where a target is absent, present-but-invalid, excluded by policy, and supplied after re-index.
- Assert that supplying the artifact closes `EXTERNAL_UNRESOLVED`; improving the parser closes `PARSE_UNRESOLVED`; neither action closes the other class.
- Report gap counts and blast radius by class and reason, never as one generic unresolved total.

### Pitfall 6: Cross-repository scope leakage

**What goes wrong:** Same-named artifacts from another repository satisfy a reference, appear in AI context, leak through Neo4j export, or become visible to a session outside its workspace/project scope.

**Why it happens:** Gortex intentionally stores multiple repositories in one shared graph and reach/analyze tools often default to workspace breadth. Some analyses are workspace-bound but not repo-narrowed. Current health evidence also reports 104 misprefixed nodes, proving ownership drift is not theoretical. Filesystem access is bounded to tracked roots, not to the session workspace.

**Consequences:** False dependency edges, proprietary source disclosure between projects, contaminated AI claims, and exports that weaken the native query boundary.

**Prevention:**
- Carry workspace, project, repo, retrieval revision, and library/search-order context through extraction, resolution, claim context, export, and projection.
- Fail closed on absent or ambiguous scope. A short COBOL member name must not resolve across all tracked repos by default.
- Treat cross-repo links as explicit derived relationships with both endpoints' scope/provenance, not as canonical identity merging.
- Apply the same allow-set to source reads, graph traversal, AI context assembly, logs, and Neo4j projection. Do not assume a query-layer clamp protects exporter or logger paths.

**Detection:**
- Two-workspace adversarial fixture with identical program/copybook/resource names and unique canary text.
- Assert no canary appears in queries, contexts, logs, claims, or exports outside its authorized scope.
- Make repo-ownership health (`unowned`/`misprefixed`) a gate before AI or projection phases.
- Test parent-directory sessions and explicit widening/narrowing because their scope semantics differ from sessions opened inside one repo.

### Pitfall 7: Proprietary source leaks through providers or logs

**What goes wrong:** Raw estate source, literals, paths, prompts, model output, or graph snippets reach an unapproved external provider or persist in query logs, provider logs, subprocess history, exported files, caches, traces, or evaluation artifacts.

**Why it happens:** Provider configuration is flexible and global/project config merges per field. The current query logger is enabled by default, stores query text and errors, creates world-readable `0644` files, rotates one backup, and can store full responses when `GORTEX_QUERY_LOG_RESPONSES=1`. CLI/subprocess providers may apply their own retention. Redaction after dispatch is already too late.

**Consequences:** Intellectual-property or regulated-data breach, unbounded retention, and inability to prove which source was sent where.

**Prevention:**
- AI enrichment must be off by default and require an explicit policy decision per provider, model, workspace, data class, and retention mode.
- Enforce provider allow-listing, bounded context, secret/protected-literal scanning, and redaction before request construction and before every persistence/logging seam.
- Default milestone-specific logs to metadata-only, restrictive permissions, bounded retention, and no raw prompt/context/response. Do not rely on `GORTEX_QUERY_LOG_RESPONSES` remaining unset.
- Record an audit envelope containing provider/model/policy/source scope and hashes, not proprietary payloads.
- Require local or approved enterprise endpoints for raw source; subprocess/CLI providers are external unless their actual transport and retention policy is approved.

**Detection:**
- Canary secrets and proprietary markers tested against outbound request capture, query log, daemon log, rotated log, claim store, temp files, crash/error paths, and Neo4j export.
- Negative tests for global-provider configuration overridden by repo policy and vice versa.
- File-mode and retention tests for every persisted AI artifact.
- Provider dispatch must emit a policy decision record even when denied, without echoing denied source.

### Pitfall 8: Parser-version drift hides behind ignored `go.work`

**What goes wrong:** A developer runs the accepted parser baseline while CI, another machine, a release build, or `GOWORK=off` uses the older module dependency. Graph output then differs although the gortex commit and source corpus are identical.

**Why it happens:** The current `go.work` is ignored by `.gitignore`, contains an absolute path to `tree-sitter-cobol-upgrade/forest-shim/cobol`, and is the mechanism selecting the completed parser at commit `97ac9f1`. Official Go behavior makes all `use` modules main modules, so the local shim overrides normal module resolution. The parser repo root is not itself a Go module (`go test ./...` from its root fails), making naive verification misleading.

**Consequences:** False reproducibility claims, passing local acceptance gates with failing release behavior, unexplained graph churn, and stale parser-gap findings.

**Prevention:**
- Make the parser grammar content ID and generated shim revision explicit build inputs, not facts inferred from an untracked workspace file.
- Add a committed, machine-independent dependency pin or a reproducible bootstrap/CI assertion that proves the selected module directory and parser hash equal baseline `97ac9f1`.
- Run acceptance in both intended workspace mode and `GOWORK=off`; fail if either silently resolves an unapproved parser.
- Store parser content ID on extraction generations/findings so a parser-only change triggers reconciliation.

**Detection:**
- Gate prints and asserts `go env GOWORK`, `go list -m -json` directory/version/replacement, parser generated-file hash, and parser git SHA where available.
- Golden AST/query and graph fingerprint must match on a clean machine without the developer's absolute path.
- Deliberately remove/rename the local parser checkout and verify the build fails closed rather than falling back unnoticed.

### Pitfall 9: Manual projection is implemented as raw export replay or mistaken for synchronization

**What goes wrong:** The approved Phase 1 operation replays current `CREATE` output into a non-empty target, loads the whole graph unbounded, accepts ambiguous scope, collapses evidence-bearing relationships, leaves stale selected-scope records active, or reports completion after partial failure. A worse variant joins indexing and creates split-brain authority.

**Why it happens:** Current Neo4j support is a file/inline snapshot serializer, not a database projection service. `WriteCypher` snapshots through `CREATE`; current MCP export filtering is repository-oriented; the scoped SQLite readers and mutation generations are separate lower-level seams. There is no atomic transaction spanning SQLite and Neo4j, and target idempotency requires stable application keys plus constraints.

**Consequences:** Split-brain graph, duplicate relationships, resurrected stale facts, projection of unauthorized repositories, and operational complexity that undermines SQLite's current authority.

**Prevention:**
- Keep SQLite authoritative without qualification and keep Neo4j outside indexing and ordinary availability paths.
- Resolve explicit workspace/project/repository scope and authoritative SQLite snapshot/generation before planning writes; fail closed on absence or ambiguity.
- Derive stable node and relationship projection keys from authoritative identity plus material occurrence/provenance distinctions, and establish required target constraints before upserts.
- Stream bounded scoped rows and use retry-safe idempotent transactions; stop promptly on cancellation/error and mark incomplete work honestly.
- Define selected-scope stale ownership/reconciliation explicitly without affecting other scopes. The capability is required, but the default stale action remains a discuss-phase decision.
- Never use raw `CREATE` exporter output as the apply protocol and never add reverse writes.
- Reserve outbox/change capture, checkpoints, and lag semantics for a later continuous-synchronization ADR.

**Detection:**
- Disposable-Neo4j or protocol-seam tests cover empty and non-empty targets, repeated apply, mid-batch error, cancellation, retry, stale selected-scope records, and neighboring-scope canaries.
- Reapply each batch multiple times and assert one projected node/relationship per stable key while preserving distinct evidence-bearing occurrences.
- Compare dry-run expected counts/scope with final result and assert incomplete work is never labeled complete.
- Snapshot SQLite before/after success and failure and assert no authoritative mutation.
- Run indexing, daemon startup, native queries, and non-projection tests with Neo4j absent.

## Moderate Pitfalls

### Pitfall 10: Source positions are mistaken for durable identity

**What goes wrong:** Inserting lines above a paragraph changes every finding/claim ID, while an actual semantic rewrite at the same line reuses an old ID.

**Prevention:** Use source range as provenance and occurrence disambiguation, but combine it with owner identity and content/construct fingerprint. Define move, edit, split, and merge semantics before claim lineage is added.

**Detection:** Move-only and whitespace-only fixtures should preserve domain identity while updating occurrence provenance; content changes should supersede the old occurrence.

### Pitfall 11: Aggregate parser counts mask structural loss

**What goes wrong:** Node totals improve while downstream paragraphs/data items disappear through recovery cascades, or duplicate captures inflate counts. The parser baseline already showed four `ERROR` nodes could hide most paragraphs in a 5,352-line program.

**Prevention:** Gate on source-positioned expected constructs, containment continuity, and graph edges, not parse-error count or total nodes alone. Preserve the existing corpus/NIST/differential tests and add graph-level assertions.

**Detection:** Per-file sentinel nodes after every risky construct, clean-node reclassification diff, duplicate-range audit, and before/after edge coverage on the same corpus.

### Pitfall 12: Placeholder nodes masquerade as obtained artifacts

**What goes wrong:** An unresolved call target or copybook placeholder acquires attributes/edges that make clients treat it as indexed source.

**Prevention:** Give placeholders an explicit unresolved kind/status, no fabricated source range/body, and a resolution link when the artifact arrives. Do not merge placeholder and real entity solely by short name.

**Detection:** Trusted queries must expose availability/evidence state; tests ensure placeholders cannot satisfy queries requiring parsed implementation.

### Pitfall 13: Generic node kinds erase mainframe meaning

**What goes wrong:** Programs and paragraphs both appear as functions, divisions as types, and sections as methods. Generic analyzers then apply language assumptions that are false for COBOL.

**Prevention:** Decide where a new kind is essential versus where typed metadata plus dedicated edges is sufficient. Require every mainframe concept to be queryable without parsing display names.

**Detection:** Schema contract queries distinguish program, paragraph, section, division, statement, data item, resource, finding, and claim directly.

## Minor Pitfalls

### Pitfall 14: Case and dialect normalization destroys source truth

**What goes wrong:** Uppercasing aids resolution but loses raw spelling, quoting, schema qualifiers, or dialect-sensitive distinctions.

**Prevention:** Store raw lexeme and canonical lookup key separately; version the normalization policy.

### Pitfall 15: Line-only provenance is insufficient

**What goes wrong:** Multiple operands/statements on one line become indistinguishable, especially in fixed-format COBOL and embedded SQL/CICS blocks.

**Prevention:** Persist byte/row/column ranges from the named AST and source revision. Keep line-only provenance only for legacy facts.

### Pitfall 16: Silent logging failures create false audit confidence

**What goes wrong:** The current query logger intentionally fails silent, so an operator may assume an AI audit trail exists when disk/path errors disabled it.

**Prevention:** Security audit records need an explicit health signal and fail-closed policy where required; retrieval telemetry can remain fail-silent.

## Phase-Specific Warnings

| Phase topic | Likely pitfall | Required prevention gate | Required detection gate |
|-------------|----------------|--------------------------|-------------------------|
| 1. General Neo4j projection | Raw export replay, ambiguous scope, duplicate relationships, cross-scope stale deletion, or hidden partial failure | Shared language-agnostic CLI/MCP service; explicit scope/generation; stable keys/constraints; bounded idempotent batches; dry-run/progress/cancellation/incomplete result; selected-scope stale ownership | Disposable target/protocol tests for retry, stale records, cancellation, scope canaries, property preservation, and unchanged SQLite |
| 2. Thin deterministic tracer | Existing regex IDs and generic kinds are copied into the new path | Trace one named AST node with exact range, parser/extractor version, repo/workspace identity, and deterministic evidence class through SQLite and CLI/MCP | Same input twice yields identical active IDs; duplicate-name fixture does not collapse occurrences |
| 3. Deterministic COBOL graph extraction | Aggregate coverage hides cascades or collisions | Expand by construct family only after identity/provenance contract is stable; preserve raw and canonical names | Corpus differential checks source-positioned expected nodes/edges, clean reclassification, collision groups, and downstream sentinels |
| 4. Stable IDs and incremental lifecycle | Add-only updates leave stale facts/findings | Set-based reconcile per owner/generation; parser/config/retrieval versions participate in invalidation | Incremental-vs-clean-rebuild equivalence across edit/delete/restore/parser-change/interrupted-retry sequence |
| 5. Gap and missing-artifact model | Retrieval, preprocessing, and parse failures become one generic unresolved bucket | Classify availability before syntax interpretation; placeholders carry no invented target facts | Intentionally incomplete corpus proves each gap closes only through its correct remedy |
| Cross-cutting projection boundary | Manual projection is widened into continuous synchronization or authority | Prohibit index-time dual-write/reverse writes; require a later ADR for change capture/checkpoints/lag | Static/code-path review plus Neo4j-absent ordinary-operation acceptance |
| 7. AI context and structured review | Source exfiltration and confidence promotion | Policy gate and redaction before context/provider/logging; schema requires evidence class, citations, missing context, and versions | Canary scan across outbound traffic and all persistence; `confidence=1.0` AI remains untrusted |
| 8. AI claims and human review | Claims mutate facts or lose disagreement history | First-class append-only claims/reviews with contradiction and supersession; deterministic tables are not AI-writable | AI-on/off deterministic snapshots match; conflicting models coexist; stale evidence deactivates dependent claims |
| 9. End-to-end validation | Developer-only `go.work` makes results irreproducible | Pin and attest parser baseline independent of an ignored absolute local workspace path | Clean-machine and `GOWORK=off` gates verify selected parser/hash or fail closed |
| Cross-cutting multi-repo validation | Wrong repo data leaks into resolution, AI, logs, or Neo4j | One scope contract shared by graph read, file read, context builder, logger, claim store, and exporter | Canary repositories plus zero misprefixed/unowned node gate |

## Release-Blocking Invariants

1. Identical source, retrieval revision, parser content ID, extractor version, and configuration produce the same active deterministic identity set.
2. An incremental graph equals a clean rebuild for the same inputs.
3. No AI operation changes a deterministic node or edge.
4. Evidence class controls trust; confidence never promotes evidence.
5. Missing source and present-but-unparsed source remain separately queryable.
6. Every active claim cites active evidence from an allowed scope and declares missing context.
7. Raw proprietary source cannot reach an unapproved provider, query log, rotated log, export, or temp artifact.
8. A build cannot silently fall back from parser baseline `97ac9f1` because `go.work` is absent.
9. SQLite remains the sole read/write authority; replacement is outside v1.0.
10. The manual Neo4j projection is idempotent, explicitly scoped, rebuildable, and honest about incomplete work; continuous lag semantics are out of scope.

## Sources

### Repository and baseline evidence -- HIGH confidence

- `.planning/PROJECT.md` -- milestone goal, sequencing constraints, SQLite authority, parser baseline `97ac9f1`.
- `docs/ai-enhanced-cobol-graph-handoff.md` -- evidence classes, finding/claim contracts, gap distinction, identity/idempotency and security requirements.
- `tree-sitter-cobol-upgrade/docs/baseline.md` -- measured recall, error-cascade behavior, preprocessing boundary, corpus and regression evidence.
- `tree-sitter-cobol-upgrade/docs/spec/SPEC-001-nodes-for-edges.md` -- named-node contract, proprietary-source warning, integration boundary and parser constraints.
- `internal/parser/languages/cobol.go` -- current regex IDs, duplicate suppression, generic kinds, unresolved target namespaces, comment and dynamic-call behavior.
- `internal/graph/node.go` and `internal/graph/graph.go::hashEdgeIdentity` -- current identity/scope fields and origin-sensitive edge identity.
- `internal/indexer/deferred_enrich_scope.go::Indexer.deferredEnrichFrontiers` and `internal/indexer/multi.go::MultiIndexer.runGlobalGraphPassesTopologyHeld` -- current eviction, scoped enrichment and cross-repo derived-pass lifecycle.
- `internal/mcp/query_log.go` -- default-on metadata logging, optional full responses, permissions, rotation, and fail-silent behavior.
- `docs/llm.md` -- provider matrix, configuration precedence and optional/off-by-default LLM surfaces.
- `docs/multi-repo.md` -- workspace/project boundaries, query intent defaults, filesystem-scope caveat, watcher/reconcile behavior.
- `internal/exporter/cypher.go` and `internal/mcp/tools_export.go` -- current manual snapshot export uses `CREATE`, has no DDL, and can export all repos by default.
- Direct verification on 2026-09-15: parser repo HEAD equals `97ac9f1`; gortex `go.work` is ignored and resolves the local parser shim by absolute path; `go test ./...` at the parser repository root is not a valid baseline command because that root is not a Go module.
- Gortex `index_health` on 2026-09-15 reported 104 misprefixed nodes, which makes cross-repo ownership validation a current prerequisite rather than a hypothetical hardening task.

### External documentation -- supporting confidence LOW per research confidence classifier

- Go official workspace tutorial: https://go.dev/doc/tutorial/workspaces -- `go.work` lists modules treated as main modules and local workspace code is used for builds.
- Go Modules Reference: https://go.dev/ref/mod#workspaces -- workspace/module selection and reproducibility semantics.
- Neo4j Cypher `MERGE`: https://neo4j.com/docs/cypher-manual/current/clauses/merge/ -- exact-pattern behavior, constraints recommendation, and concurrency uniqueness caveat.
- Neo4j constraints: https://neo4j.com/docs/cypher-manual/current/constraints/managing-constraints/ -- uniqueness/key constraints and index-backed enforcement.
- OWASP LLM02:2025 Sensitive Information Disclosure: https://genai.owasp.org/llmrisk/llm022025-sensitive-information-disclosure/ -- sanitization, least privilege, restricted sources, redaction, and transparent retention policy.

## Research Gaps Requiring Phase-Specific Work

- The exact mainframe canonical-key rules depend on library concatenation/search order, subsystem qualifiers, preprocessing manifests, and retrieval metadata not yet represented in the native schema.
- Confidence calibration is intentionally unresolved. The roadmap should not block deterministic extraction on it; calibration belongs in the AI evaluation phase.
- The manual Neo4j projection is approved independently of query benchmarks. Measured query/scale evidence is still required before accepting continuous synchronization, index-time integration, or replacement.
- Provider-specific retention, training, regional processing, and enterprise approval cannot be inferred from provider names. Treat each deployment as unapproved until policy evidence is supplied.
