# Project Research Summary

**Project:** Gortex Mainframe Engine -- v1.0 Neo4j Projection and Mainframe Graph Foundation
**Domain:** General scoped graph projection first; provenance-aware mainframe graph after grammar readiness
**Researched:** 2026-09-15; reconciled 2026-09-21
**Confidence:** HIGH for SQLite authority, current scoped/export seams, and Phase 1 boundaries; MEDIUM for intentionally open projection transport/UX details and later AI policy/claim-ledger details

## Executive Summary

Start Phase 1 with a general, manually invoked, explicitly scoped projection of Gortex's authoritative SQLite graph into Neo4j through equivalent CLI and MCP operations. The projection is language-agnostic, downstream, retry-safe, and rebuildable. SQLite remains the sole authority; Neo4j never joins indexing, daemon startup, native queries, reverse writes, or ordinary availability paths.

Ground Phase 1 in current code rather than a new graph model: `internal/exporter/cypher.go` provides language-neutral node/edge serialization behavior; `internal/graph/store_sqlite/scoped_projection.go` provides bounded current-generation scoped reads; current CLI/MCP export surfaces provide entry-point conventions; native IDs and workspace/project/repository fields provide identity/scope; SQLite generations and mutation receipts provide the authoritative snapshot boundary. The current `CREATE` exporter is not the finished apply protocol. The new operation needs stable projected node/relationship keys, target constraints, bounded idempotent transactions, dry-run, redacted progress/results, cancellation, partial-failure honesty, and selected-scope stale-record reconciliation.

This recommendation **supersedes the 2026-09-15 defer-only Neo4j conclusion**. Preserve the reasons against authority replacement, continuous synchronization, and dual-write, but do not use them to delay the approved manual projection. Do not lock discuss-phase gray areas: exact command name, credential syntax, transport/driver choice, or default stale-record policy.

After parser/grammar readiness, continue with the deterministic mainframe sequence: thin COBOL tracer from parser baseline `97ac9f1`, deterministic AST breadth, stable incremental identity, explicit `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED`, then policy-gated AI claims and human review.

## Key Findings

### Recommended Stack

Retain the native stack and add only contracts the milestone actually needs. No parallel graph service, vector database, orchestration framework, or new LLM framework is justified.

**Core technologies:**
- **Go 1.27.0:** existing engine runtime -- preserves daemon, indexing, CLI, MCP, HTTP, and analyzer integration.
- **Tree-sitter Go v0.25.0:** deterministic AST and exact source ranges -- use the enhanced COBOL forest shim at semantic baseline `97ac9f1`.
- **Published or pseudo-versioned COBOL shim:** reproducible parser dependency -- replace reliance on the ignored absolute-path `go.work` before milestone acceptance.
- **SQLite via `modernc.org/sqlite` v1.58.0:** sole authoritative graph store -- already owns atomic mutations, generations, restart recovery, scope, and incremental reconciliation.
- **Existing `graph.Node`/`graph.Edge`, indexer, resolver, and query engine:** native integration substrate -- extend minimally and preserve workspace/project/repository semantics.
- **Existing MCP, CLI, and HTTP surfaces:** user access -- every new fact must be queryable here rather than only through exports.
- **Existing `internal/llm` provider layer:** optional AI transport -- add policy, bounded context, and validated claim contracts instead of another SDK/framework.
- **`jsonschema/v6` v6.0.3:** deterministic AI-output validation -- promote from indirect dependency only when claim validation is implemented.
- **Existing Cypher exporter plus scoped SQLite readers:** Phase 1 foundation -- reuse language-neutral property serialization and bounded current-generation reads, but replace raw `CREATE` replay with a retry-safe apply contract.
- **Neo4j transport/client:** required capability, choice intentionally open -- select during discuss-phase and audit any new package before installation.

**Critical version rule:** CI, clean builds, and `GOWORK=off` must resolve and attest parser grammar content equivalent to merge `97ac9f1`; no silent fallback to the stock COBOL module is acceptable.

### Expected Features

**Must have (table stakes):**
- General manually invoked SQLite-to-Neo4j projection through equivalent CLI/MCP operations with explicit fail-closed workspace/project/repository scope.
- Stable projected node/relationship keys, target constraints, bounded idempotent apply, safe retry/partial failure, dry-run, redacted progress/results, cancellation, selected-scope stale reconciliation, and unchanged SQLite.
- Neo4j-absent indexing, daemon startup, native query, and non-projection behavior.
- Thin parser-node-to-SQLite-to-CLI/MCP tracer after grammar readiness, with exact range, stable ID, parser fingerprint, scope, and AI disabled.
- Deterministic program, division, section, paragraph, statement, data-item, control-flow, call, and copybook graph facts.
- Source-positioned IDMS record/set, CICS program/transaction/map/mapset, and SQL read/write relationships without turning aliases, CTEs, or dynamic sources into false physical resources.
- Separate identity rules for domain entities, source occurrences, edges, findings, claims, and revisions.
- Idempotent incremental lifecycle where repeated runs do not duplicate active facts and changed/deleted inputs close or supersede stale state.
- Explicit `PARSE_UNRESOLVED` findings for present-but-uninterpreted source and `EXTERNAL_UNRESOLVED` placeholders/findings for unavailable artifacts.
- Evidence-class filtering separate from origin and confidence; default trusted views exclude unreviewed AI.
- Native user queries that disclose unresolved boundaries and lower-bound results.
- Full deterministic operation with every LLM provider disabled.

**Should have (differentiators):**
- Honest incomplete-estate graph with impact-ranked parser and retrieval gaps.
- Evidence-preserving lineage across source, parser, extractor, model, and review changes.
- One bounded AI review workflow producing schema-valid, evidence-linked `AI_INFERRED` claims.
- Human confirmation, contradiction, and supersession without rewriting deterministic facts.
- Policy-aware source handling and a feedback loop that graduates recurring, proven gaps into deterministic parser/extractor support.

**Defer (v2+ or decision-gated):**
- Continuous/live Neo4j synchronization -- Phase 1 is an explicit manual snapshot projection; benchmark needs and require a separate ADR before change capture, checkpoints, lag semantics, or automatic refresh.
- Full JCL PROC/include/symbolic/library-order resolution.
- Behavioral or live-synchronized digital twin capabilities.
- Broad business-concept and end-to-end transaction inference.
- Automatic promotion of AI output, parsing every dialect, UI-first workflows, and a vector database.

### Architecture Approach

Use downstream projection plus two later one-way write paths over one authority. Phase 1 is CLI/MCP adapter -> explicit scope resolver -> authoritative SQLite snapshot/generation -> bounded scoped readers -> projection planner -> Neo4j apply boundary -> redacted progress/result. The later deterministic path is parser adapter -> observation mapper -> existing index lifecycle/resolver -> SQLite -> native query surfaces. The later enrichment path reads bounded scoped evidence from SQLite -> applies redaction/provider policy -> invokes an existing provider -> validates a closed claim schema -> writes append-only claims/reviews. Neither Neo4j nor AI writes authoritative deterministic rows.

**Major components:**
1. **Parser adapter** -- runs the pinned grammar and exposes named nodes, failures, exact ranges, and grammar fingerprint.
2. **COBOL observation mapper** -- maps AST observations to native nodes/edges and deterministic metadata without owning persistence or resolution.
3. **Indexer and deterministic resolver** -- apply repository/workspace/project identity, file ownership, eviction, reconciliation, and evidence-based binding.
4. **Finding normalizer** -- creates stable, source-owned parse-gap and missing-artifact records while preserving unresolved references.
5. **SQLite graph store and native queries** -- own active authoritative facts, scope enforcement, search, traversal, and gap queries.
6. **AI context/policy/validation boundary** -- selects minimal allowed evidence, redacts it, invokes an approved provider, and rejects malformed or out-of-scope claims.
7. **Claim ledger and review projection** -- retain immutable proposals and review lineage separately from file-owned deterministic facts.
8. **Projection scope/planner/apply service** -- resolves explicit scope/generation, streams bounded authoritative rows, plans stable keys/constraints/stale ownership, applies idempotently, and reports complete/incomplete state.
9. **CLI/MCP projection adapters** -- expose equivalent behavior without duplicating projection business logic.
10. **Cypher exporter** -- remains a serializer/interchange surface; it owns no authority or reverse-write path.

**Key patterns:** language-agnostic manual projection first; shared CLI/MCP service; fail-closed scoped snapshots; stable projected keys and constraints; bounded idempotent apply; then walking-skeleton tracer, layered native identities, file-owned reconciliation, explicit epistemic boundaries, and schema-validated claim envelopes.

### Critical Pitfalls

1. **Identity collisions collapse distinct artifacts** -- define entity-specific canonical keys, preserve raw and normalized names, include repository/library/program/schema context, and test adversarial duplicates.
2. **Incremental indexing leaves stale or duplicate active facts** -- assign every observation an owner/generation, reconcile desired sets atomically, invalidate dependents, and require clean-rebuild equivalence.
3. **Confidence is treated as evidence or AI mutates truth** -- use typed evidence classes and separate append-only claims; prove AI-on and AI-off deterministic snapshots are identical.
4. **Parse gaps, missing artifacts, and placeholders are conflated** -- classify evidence availability before syntax interpretation; never fabricate missing target contents; verify each class closes only through its correct remedy.
5. **Scope or proprietary source leaks across repositories/providers/logs** -- fail closed on ambiguous scope, use one allow-set across reads/context/logs/export, gate providers explicitly, redact before dispatch, and use canary tests.
6. **Parser drift hides behind ignored `go.work`** -- commit a reproducible shim pin, attest grammar content, test clean-machine and `GOWORK=off` builds, and fail rather than silently falling back.
7. **Raw Cypher export is mistaken for the approved projection or widened into synchronization** -- require explicit scope/generation, stable keys/constraints, bounded idempotent apply, stale selected-scope ownership, cancellation and incomplete-result honesty; reserve outbox/checkpoints/lag for later continuous synchronization.

## Implications for Roadmap

Based on the superseding decision and dependency graph, use nine phases. Keep Phase 1 language-agnostic and executable; do not turn it into COBOL schema or continuous-synchronization work.

### Phase 1: General Neo4j Projection Command
**Rationale:** The immediate user priority is safe, explicit Neo4j utility over the graph already authoritative in SQLite, independent of parser readiness.
**Delivers:** Equivalent CLI/MCP operations; fail-closed workspace/project/repository scope; authoritative snapshot/generation selection; stable projected keys and constraints; bounded idempotent apply; dry-run, redacted progress/results, cancellation, partial-failure/retry safety, and selected-scope stale reconciliation; Neo4j-absent ordinary operation.
**Addresses:** NEO-01 through NEO-17 and STORE-01 through STORE-05.
**Avoids:** Raw `CREATE` replay, unbounded loads, ambiguous scope, duplicate evidence, cross-scope deletion, secret disclosure, split-brain authority, and grammar coupling.
**Open for discuss-phase:** exact command name, credential syntax, driver/transport choice, and default stale-record policy.

### Phase 2: Thin Deterministic Native Tracer and Minimum Contracts
**Rationale:** Prove the real production seam before multiplying parser-node mappings or adding schema kinds.
**Delivers:** One named program/`PROGRAM-ID` node through the registered Tree-sitter extractor, `ExtractionResult`, repo prefixing, SQLite, and one CLI plus one MCP/native query; exact range and parser/extractor provenance; repeated and changed-file checks.
**Addresses:** Thin tracer, native user-query outcome, AI-off operation.
**Uses:** Existing `KindFunction`, `EdgeDefines`, `AddBatch`, indexer, query engine, CLI/MCP.
**Avoids:** Parallel graph architecture, broad schema-first development, and unlocalized integration failures.

### Phase 3: Deterministic COBOL Graph Breadth
**Rationale:** Once the walking skeleton is proven, expand in vertical construct families while each result remains directly queryable.
**Delivers:** Minimal justified domain schema and AST mapping for structure, paragraphs/control flow, calls, copybooks, data items, IDMS, CICS, and SQL, with exact source evidence and unresolved dynamic forms preserved.
**Addresses:** All deterministic structure/resource table stakes and required modernization queries.
**Avoids:** Generic-kind semantic loss, aggregate-count false confidence, aliases/CTEs as physical tables, and invented resolution.

### Phase 4: Incremental, Cross-Repository, and Identity Lifecycle
**Rationale:** Findings and claims must not be built atop graph churn or stale active state.
**Delivers:** Idempotent reruns; edit/delete/restore/parser-change reconciliation; clean-build equivalence; scoped duplicate-member and cross-repo resolution behavior; closure/supersession primitives.
**Addresses:** Stable graph identity and idempotent lifecycle.
**Avoids:** Stale facts, line-number identity, wrong-library binding, dangling derived edges, and scope leakage.

### Phase 5: Explicit Unresolved Evidence and Impact Queries
**Rationale:** AI context and honest user answers require deterministic knowledge of whether evidence is malformed or absent.
**Delivers:** Source-owned `PARSE_UNRESOLVED` findings from tails/`ERROR`/`MISSING`; `EXTERNAL_UNRESOLVED` placeholders/findings; closure rules; blockage and impact ranking; lower-bound query disclosure.
**Addresses:** Honest incomplete-estate graph, parser-gap ranking, missing-artifact queries, evidence filtering.
**Avoids:** Generic unresolved buckets, silent parser loss, and placeholders masquerading as obtained source.

### Phase 6: Native Deterministic Acceptance and Pre-AI Release Gate
**Rationale:** Native query outcomes, reproducibility, lifecycle equivalence, monitoring, and unresolved-boundary disclosure must be proven before AI is introduced.
**Delivers:** Representative deterministic fixtures; CLI/MCP query acceptance; incremental-versus-clean equivalence; unresolved lower-bound disclosure; deterministic monitoring with AI disabled.
**Addresses:** Native query and deterministic evaluation requirements.
**Avoids:** AI masking parser/storage defects or incomplete results presented as complete.

### Phase 7: AI Contract, Policy, and Evaluation Foundation
**Rationale:** Provider invocation is unsafe and unmeasurable until context, redaction, retention, schema, and evaluation contracts are fixed.
**Delivers:** AI design specification; approved-provider policy; bounded deterministic context manifest; pre-dispatch redaction; versioned output schema; evidence-ID/scope validation. AI remains off by default.
**Addresses:** Policy-aware source handling and validated optional enrichment foundation.
**Avoids:** Proprietary-source leakage, free-form `ask` as a write API, unvalidated structured output, and confidence promotion.

### Phase 8: Evidence-Linked Claims and Human Review
**Rationale:** Persist model output only after stable evidence and policy gates exist.
**Delivers:** Append-only first-class claims/reviews; evidence links; model/prompt/context/schema lineage; confirmation, contradiction, staleness, and supersession; explicit opt-in claim queries.
**Addresses:** AI claims, bounded review, human trust transitions, temporal lineage.
**Avoids:** AI mutation of deterministic truth, last-write-wins disagreement, claim deletion during file eviction, and default-query contamination.

### Phase 9: End-to-End AI Evaluation and Projection Safety
**Rationale:** Close the milestone by proving AI release invariants and every external-disclosure boundary without widening Phase 1 into synchronization.
**Delivers:** Curated AI evaluation; AI-on/off deterministic equality; source/scope canaries across providers, logs, claims, temporary files, and projection; confirmation that projection remains manual and one-way.
**Addresses:** End-to-end AI quality, security, and storage-authority invariants.
**Avoids:** Developer-only success, source leakage, split-brain writes, and accidental continuous synchronization.

### Phase Ordering Rationale

- General projection comes first because it operates on the existing language-agnostic SQLite graph and does not require COBOL grammar readiness.
- Identity/evidence and parser reproducibility remain prerequisites for later COBOL breadth, but implementation proof must immediately follow so contracts are tested rather than overdesigned.
- Deterministic breadth precedes lifecycle stress because representative relationships are needed, while lifecycle must stabilize before durable findings or claims.
- Both unresolved classes precede AI because context selection and evaluation depend on whether bytes are present or artifacts are absent.
- Policy/schema/evaluation precede claim persistence; otherwise the first AI implementation establishes unsafe defaults.
- Manual Neo4j projection is Phase 1. Only continuous synchronization, index-time coupling, reverse writes, or replacement remain behind later measurements and a separate ADR.

### Research Flags

Phases likely needing `/gsd-plan-phase --research-phase <N>`:
- **Phase 1:** research the shared projection service boundary, authoritative snapshot/scope contract, projected relationship occurrence key, Neo4j constraint/upsert plan, stale ownership, cancellation/retry semantics, and disposable integration-test seam. Discuss command naming, credentials, driver choice, and stale default before locking them.
- **Phase 2:** freeze the exact enhanced-parser Go API/fingerprint, portable module pin, canonical keys, edge occurrence identity, and minimal new kinds.
- **Phase 3:** research construct-family mappings and query semantics, especially SQL alias/CTE handling, CICS operands, IDMS resources, and copybook preprocessing context.
- **Phase 4:** research actual library concatenation/search order, duplicate member resolution, and cross-repository lifecycle behavior.
- **Phase 7:** use the AI integration design workflow; approved providers, retention, redaction, logging, evaluation set, and thresholds are unresolved policy decisions.
- **Phase 8:** design claim-ledger retention/encryption, human-confirmation scope, and active-view projection semantics.
- **Phase 9:** validate AI quality and projection disclosure boundaries; any continuous synchronization proposal requires a new researched phase and separate ADR rather than being smuggled into validation.

Phases with established patterns where separate research can usually be skipped:
- **Phase 2:** existing extractor/indexer/SQLite/query paths and walking-skeleton acceptance are well documented.
- **Phase 5:** gap classes and required lifecycle/query behavior are already strongly specified, assuming Phase 4 identity decisions are complete.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Current export/scoped-read/SQLite seams and authority boundary are verified. Phase 1 transport/driver selection remains intentionally open; future continuous synchronization remains MEDIUM and separately gated. |
| Features | HIGH | Deterministic table stakes derive from the accepted parser contract, current extractor gaps, and concrete user queries. Broad AI scope remains intentionally deferred. |
| Architecture | HIGH | Existing indexer, store, resolver, query, scope, LLM, and exporter behavior directly supports the two-path architecture and thin tracer. Claim-ledger tables remain MEDIUM pending retention/query needs. |
| Pitfalls | HIGH | Most risks are demonstrated by current IDs, lifecycle paths, logging defaults, scope health, exporter behavior, and `go.work`; provider-specific policy and future projection failure modes remain MEDIUM. |

**Overall confidence:** HIGH for roadmap ordering and deterministic architecture; MEDIUM for phase-specific AI governance and any future synchronized projection.

### Gaps to Address

- **Portable parser identity:** decide the exact tagged/pseudo-versioned shim and persisted grammar-content fingerprint equivalent to `97ac9f1`.
- **Canonical mainframe keys:** obtain real preprocessing manifests, copybook/library search order, subsystem qualifiers, and retrieval metadata before locking resolution identities.
- **Edge occurrence semantics:** verify whether current edge deduplication preserves multiple evidence-bearing occurrences or requires occurrence/evidence nodes.
- **Minimal domain vocabulary:** settle which concepts require first-class node/edge kinds based on required query ergonomics, not candidate lists.
- **Current scope health:** investigate and clear the reported misprefixed/unowned-node condition before AI context or export acceptance.
- **AI governance:** identify approved deployments and retention/training/region/log policies; provider support does not imply authorization.
- **Claim ledger:** define retention, encryption, raw-context storage, review authority, stale-evidence behavior, and whether human confirmation is version-scoped.
- **AI evaluation:** define the reference corpus, quality thresholds, confidence calibration approach, and rule for graduating recurring gaps into deterministic support.
- **Projection discussion choices:** settle exact CLI/MCP naming, credential syntax, transport/driver, and default stale-record policy without weakening fixed scope, authority, retry, and reconciliation requirements.
- **Continuous-synchronization need:** collect freshness/query/scale evidence before adding change capture, automatic refresh, checkpoints, or lag semantics. This does not block Phase 1 manual projection.
- **Fresh baseline test:** rerun the regex extractor suite when dependency retrieval is available; its prior failure was an HTTP 403 setup issue, not a COBOL assertion failure.

## Sources

### Primary (HIGH confidence)
- `.planning/PROJECT.md` -- milestone goal, constraints, accepted parser baseline, shipped extractor fixes, and SQLite authority.
- `docs/ai-enhanced-cobol-graph-handoff.md` -- governing evidence model, gap/claim contracts, security constraints, non-goals, and mandatory thin tracer.
- `.planning/research/STACK.md` -- dependency, persistence, provider, and projection analysis.
- `.planning/research/FEATURES.md` -- table stakes, differentiators, anti-features, user-query outcomes, and feature dependencies.
- `.planning/research/ARCHITECTURE.md` -- verified current seams, component boundaries, data flows, identity direction, and build order.
- `.planning/research/PITFALLS.md` -- repository-specific failure modes, release invariants, and phase gates.
- Current repository implementation cited by the research: graph node/edge models, SQLite `AddBatch` and mutation receipts, indexer incremental paths, resolver placeholder reconciliation, scoped query engine, LLM provider/service layer, COBOL extractor/probe, and Cypher exporter.
- Parser acceptance evidence tied to `tree-sitter-cobol-upgrade/main` merge `97ac9f1`: 149/149 non-comment corpus assertions, 371 NIST successes with zero failures and unchanged skips, zero DCC/estate differential regressions, and passing Gortex error-cascade integration.

### Supporting official documentation
- Go Modules Reference and workspace documentation -- module/workspace selection and the need for durable reproducible dependency inputs.
- Tree-sitter parsing/query documentation -- exact node positions plus queryable `ERROR` and `MISSING` nodes.
- Neo4j Cypher and Go driver manuals -- `MERGE`/constraint requirements, transaction retries, and idempotency implications.
- OWASP LLM02:2025 Sensitive Information Disclosure -- pre-dispatch minimization, redaction, source restriction, and retention transparency.

### Evidence caveats
- Future Neo4j design is deliberately unproven until named native queries are benchmarked.
- AI provider approval, retention, and quality thresholds require organization-specific evidence.
- Full upstream parser suites were not rerun during every research pass, though project acceptance records and a fresh Gortex cascade test support the baseline.
- A fresh regex COBOL extractor test run was blocked by dependency-download HTTP 403 and should be repeated when the environment permits.

---
*Research completed: 2026-09-15*
*Ready for roadmap: yes*
