# Feature Landscape

**Milestone:** v1.0 Deterministic COBOL Graph Extraction and AI Enrichment Foundation
**Domain:** Trustworthy mainframe code knowledge graph
**Researched:** 2026-09-15
**Overall confidence:** HIGH for current-code and parser-baseline findings; MEDIUM for later AI workflow scope

## Product Principle

The milestone is not an AI code-understanding product with a parser attached. It is a deterministic, queryable COBOL graph that remains useful when source is incomplete, with AI added only as an optional and separately labeled review layer.

The required delivery order is strict:

1. Thin deterministic tracer through the existing native graph and query surfaces.
2. Deterministic breadth across the accepted COBOL parser contract.
3. Stable identity and incremental lifecycle.
4. Explicit `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` modeling.
5. Optional, bounded AI enrichment and human review.

This order prevents later features from depending on unstable identities, silent parser loss, or inferred facts disguised as parser truth.

## Current Baseline and Gap

The enhanced parser is an accepted evidence source, not yet the production extractor. The active `CobolExtractor` remains regex-based and emits programs, divisions, sections, paragraphs, `PERFORM`/`GO TO`, copy references, literal calls, and unresolved dynamic-call identifiers. It does not yet walk the enhanced Tree-sitter AST for data items, statements, IDMS, CICS, SQL, bounded unparsed tails, `ERROR`, or `MISSING` nodes.

The existing graph provides useful substrate: file and generic symbol nodes, source line/column fields, edges with origin/confidence/provenance, unresolved target namespaces, SQLite persistence, incremental indexing, multi-repository identity prefixes, and CLI/MCP query surfaces. It does not currently provide the milestone's first-class mainframe finding/claim lifecycle or explicit `PARSE_UNRESOLVED` versus `EXTERNAL_UNRESOLVED` distinction.

Parser acceptance evidence is strong but has an integration caveat:

- Enhanced parser baseline: `tree-sitter-cobol-upgrade/main` merge `97ac9f1`.
- Reported acceptance: 149/149 non-comment corpus assertions; NIST COBOL-85 371 successes, zero failures, unchanged 11-test skip list; zero DCC/estate differential reclassification or conversion.
- Gortex's `TestErrorCascade` was executed during this research with `-enhanced-parser` and passed, preserving all 20 paragraphs and 20 data items for supported IDMS, CICS, and SQL injections while retaining an invalid-placement negative control.
- The enhanced parser is activated through the repository's local `go.work` replacement. `go.mod` still points to the stock forest module, and `cobolprobe` explicitly says no Tree-sitter COBOL extractor is registered yet.
- The regex COBOL test suite could not be rerun in this environment because dependency retrieval for `golang.org/x/image@v0.46.0` returned HTTP 403. Existing test source directly covers the current extractor behavior, but this run is not fresh executable evidence.

## Table Stakes

Missing any of these means the v1.0 graph is not trustworthy enough for modernization queries.

| Feature | User outcome | Why expected | Complexity | Delivery order / acceptance note |
|---------|--------------|--------------|------------|----------------------------------|
| Thin deterministic vertical tracer | A user can select one accepted COBOL parser node, index it, retrieve its native graph node/edge through CLI and MCP, and navigate back to exact source | Proves the real parser-to-extractor-to-SQLite-to-query path before schema breadth multiplies risk | Medium | **First.** Use one existing parser construct and existing storage/query machinery; no AI and no Neo4j dependency |
| Parser baseline pin and runtime verification | A user can tell which grammar produced a fact and can detect accidental fallback to the stock parser | Current acceptance depends on local `go.work`; silent fallback would make graph results incomparable | Medium | Gate the tracer on a parser content/version identity and an acceptance fixture that fails against the stock grammar |
| Exact source provenance | Every material fact answers "which repository, revision, file, range, parser, and extractor produced this?" | Trustworthy impact and modernization decisions require evidence, not names alone | Medium | Reuse source line/column and provenance fields; add only missing revision/version metadata |
| Deterministic COBOL structure | Users can query programs, divisions, sections, paragraphs, statements, and data items and navigate containment | These are the minimum semantic units for understanding a COBOL estate | High | **Second.** Walk named AST nodes rather than extending regex coverage |
| Deterministic control flow and calls | Users can answer "what paragraphs can this paragraph execute?", "who calls this program?", and "which calls remain dynamic?" | Call and paragraph flow are foundational impact-analysis queries | High | Preserve `PERFORM`, `THRU`, `GO TO`, literal calls, and dynamic call identifiers; distinguish resolved from unresolved targets |
| Copybook/include usage | Users can answer "which programs use this copybook/include?" and "which missing members block verification?" | Shared record layouts dominate blast radius in COBOL estates | High | Resolve only with available evidence and library/repository context; do not guess duplicate member identity |
| DATA DIVISION model | Users can locate data declarations and trace program/resource references to relevant data items | Current regex extractor produces zero data items, so lineage and dynamic-call resolution are blocked | High | Build from accepted `data_description` nodes and source ranges |
| IDMS facts | Users can find statements using each record or set and identify affected programs/paragraphs | The parser now exposes named IDMS statements, record names, and set names | High | Extract source-positioned record/set relationships without inferring unavailable schema metadata |
| CICS facts | Users can find programs linked, transactions used, and maps/mapsets referenced by each statement or program | CICS dependencies are core online-transaction modernization inputs | High | Support accepted operand shapes including no-option, bare-option, parenthesized, and multiline forms |
| SQL facts | Users can answer which statements read or write DB2 tables and distinguish physical tables, aliases, CTEs, includes, and dynamic sources | SQL lineage is table stakes for data-impact analysis; aliases and CTEs must not become false physical tables | High | Preserve statement source ranges and explicit dynamic-source uncertainty |
| Stable graph identity | Identical source, parser, extractor, and configuration produce the same active nodes, edges, and findings | Diffing, review, caching, and external projection all fail if identity drifts | High | **Third.** Extend existing repo-prefixed IDs; define separate rules for artifacts, occurrences, edges, and findings |
| Idempotent incremental lifecycle | Reindexing does not duplicate active facts; changed source/parser closes or supersedes stale observations while retaining audit lineage | A graph that accumulates stale duplicates becomes less trustworthy over time | High | Must precede durable findings and AI claims; verify identical reruns and controlled source/parser changes |
| `PARSE_UNRESOLVED` findings | Users can ask "where did source exist but deterministic parsing or verification fail?" and receive bounded ranges, containing symbols, reason, severity, and parser version | Tree-sitter exposes queryable `ERROR` and zero-width `MISSING` nodes; the enhanced grammar also emits bounded unparsed tails | High | **Fourth.** Persist findings even with AI disabled; outcome remains a deterministic fact about failure, not a recovered semantic fact |
| `EXTERNAL_UNRESOLVED` placeholders/findings | Users can ask "which unavailable programs, copybooks, JCL members, schemas, or subsystem artifacts block the most verification?" | Missing source is not a parser failure and must not disappear into generic unresolved edges | High | **Fourth.** Keep the observed reference factual while leaving target internals unknown; do not conflate with dynamic dispatch |
| Evidence-class filtering | Queries can distinguish deterministic facts, external gaps, parse gaps, AI inferences, human-confirmed claims, contradicted claims, and superseded records | Confidence alone cannot express whether evidence exists | High | Default operational views include deterministic and human-confirmed facts, not unreviewed AI claims |
| Native user-query outcomes | Users can reach the new graph through existing CLI/MCP surfaces without custom SQL or Neo4j | A persisted graph is not a product feature until users can answer concrete questions | Medium | Required outcomes listed below; query behavior must disclose unresolved boundaries and lower-bound counts |
| AI-off operation | Indexing, persistence, lifecycle, unresolved-gap queries, and deterministic navigation work with all LLM providers disabled | Optional AI is a hard trust, security, cost, and availability boundary | Medium | Exercise in end-to-end acceptance before AI work begins |

## Required User-Query Outcomes

The roadmap should specify observable queries rather than merely schema creation. By the end of the deterministic portion, a user must be able to answer:

| Question | Required answer characteristics |
|----------|---------------------------------|
| What COBOL programs and copybooks are present? | Stable identities, repository/revision scope, exact source locations |
| What paragraphs can this paragraph execute? | `PERFORM`, `THRU`, and `GO TO` paths; unresolved boundaries explicitly reported |
| Who calls this program? | Resolved literal calls separated from dynamic call identifiers and missing targets |
| Which programs depend on this copybook? | Cross-file/repository usages plus unresolved duplicate/member-library ambiguity |
| Which data items belong to this program or copybook? | Containment and exact declaration ranges, not regex-derived approximations |
| Which DB2 tables does this program read or write? | Statement-level evidence; aliases/CTEs separated from physical tables; dynamic SQL called out |
| Which IDMS records and sets does this program use? | Source-positioned statement-to-record/set relationships |
| Which CICS programs, transactions, maps, and mapsets does this program reference? | Operand-level evidence and unresolved targets preserved |
| Where did parsing fail or remain partial? | `PARSE_UNRESOLVED` by construct family, severity, range, containing symbol, parser version, and status |
| Which unavailable artifacts block the most verification? | Ranked `EXTERNAL_UNRESOLVED` placeholders by inbound references and affected programs |
| What changed after reindexing or a parser upgrade? | Added, closed, superseded, and unchanged facts/findings without duplicate active records |
| Which statements are facts versus inferences? | Evidence class, confidence, source support, model/version where applicable, and review state |

## Differentiators

These features create value beyond a conventional syntax index, but only after table stakes are stable.

| Feature | Value proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Honest incomplete-estate graph | Makes absent source and parser limits queryable, so users know when counts and paths are lower bounds | High | Strongest milestone differentiator; extends Gortex's existing unresolved-boundary concept into durable mainframe findings |
| Parser-gap impact ranking | Prioritizes grammar/preprocessing work by number of blocked facts, callers, resources, and programs rather than raw error count | Medium | Deterministic analytics only; no AI required |
| Evidence-preserving temporal lineage | Shows why a fact or claim was closed, contradicted, or superseded across source/parser/model changes | High | Valuable for audit and confidence during modernization |
| Optional bounded AI review | Uses the smallest deterministic context package to classify a parse gap, propose dynamic targets, aliases, concepts, or summaries | High | **Fifth.** Off by default; never required for deterministic indexing or querying |
| First-class AI claims | Allows multiple models and humans to disagree without mutating domain entities or deterministic edges | High | Schema-valid, evidence-linked, versioned, confidence-scored, reviewable, and replaceable |
| Human confirmation and contradiction | Promotes reviewed claims into trusted views while retaining original evidence and dissent history | High | Confirmation changes evidence class; confidence alone never promotes a claim |
| Deterministic-to-AI improvement loop | Aggregates recurring gap families so proven patterns can graduate into parser/extractor work | Medium | AI proposes priorities; deterministic tests decide when support is real |
| Policy-aware source handling | Allows local or approved enterprise models while recording provider/model/policy/source scope and excluding secrets/raw snippets from logs | High | Mandatory for any AI release that sees proprietary estate source |

## Anti-Features

| Anti-feature | Why avoid | What to do instead |
|--------------|-----------|-------------------|
| AI-first extraction | Makes graph truth nondeterministic and hides whether the parser actually recognized source | Build and validate the deterministic tracer, breadth, lifecycle, and gap model first |
| Treating confidence as evidence | A high-confidence guess is still a guess; a missing target can be deterministically known to be missing | Store evidence class separately from confidence and default queries to verified classes |
| AI mutation of deterministic facts | Erases provenance and prevents reproducible reruns | Store AI output as separate claims linked to evidence |
| Generic unresolved bucket | Conflates parser failure, missing source, dynamic dispatch, and ordinary unresolved resolution | Model `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` explicitly; retain dynamic-call semantics separately |
| Silent dropping of `ERROR`, `MISSING`, or unparsed tails | Creates false completeness and unsafe impact answers | Emit bounded findings with containing context and exact range |
| New parallel graph engine | Duplicates persistence, incremental reconciliation, scoping, and query behavior | Extend the existing Gortex graph, SQLite store, resolver, and CLI/MCP surfaces |
| Neo4j as a v1 prerequisite or authoritative replacement | Adds synchronization and deployment risk before proving a query need; current support is manual Cypher export | Keep SQLite authoritative; defer projection until measured queries justify it |
| Full JCL symbolic resolution | Expands scope into PROC/include/symbolic/library-order semantics before the COBOL graph contract is proven | Represent referenced but unavailable JCL as `EXTERNAL_UNRESOLVED`; plan full JCL later |
| Full behavioral or live digital twin | Requires runtime events, schedules, datasets, subsystem metadata, simulation semantics, and operational synchronization far beyond this foundation | Deliver static deterministic graph and evidence-aware claims; defer behavioral and live stages |
| Inventing missing artifact contents | Converts absence into false certainty | Record the missing artifact and what evidence would resolve it |
| Parsing every dialect in v1 | Encourages broad, weak support and destabilizes the accepted baseline | Support measured constructs; persist unsupported regions as findings; prioritize by impact |
| Automatic external-model use | Risks proprietary source leakage, cost surprises, and non-reproducible indexing | Keep AI off by default with explicit provider authorization, bounded context, and retention policy |
| Building UI-specific workflows first | Couples the schema to one presentation before CLI/MCP semantics stabilize | Prove native query outcomes first; UI can consume the same evidence-aware API later |

## Feature Dependencies and Mandatory Ordering

```text
Accepted parser baseline (97ac9f1) + parser-version verification
  -> Thin deterministic tracer
     -> Native SQLite persistence
     -> CLI/MCP retrieval with exact source provenance

Thin deterministic tracer
  -> Deterministic COBOL structural breadth
     -> paragraphs/control flow/calls/copybooks/data items
     -> IDMS/CICS/SQL/resource relationships

Deterministic breadth
  -> Stable identity rules
     -> Idempotent identical reruns
     -> Incremental close/supersede lifecycle

Stable identity/lifecycle
  -> PARSE_UNRESOLVED findings
  -> EXTERNAL_UNRESOLVED placeholders/findings
     -> unresolved-impact ranking
     -> honest lower-bound query results

Deterministic facts + stable lifecycle + explicit gaps
  -> bounded AI context builder
     -> schema validation and provider policy
     -> AI_INFERRED claim persistence
     -> human confirmation/contradiction/supersession

All deterministic outcomes proven
  -> optional Neo4j projection decision
  -> later JCL resolution
  -> later behavioral/digital-twin work
```

Critical dependency rules:

- Do not broaden extraction before the tracer proves one end-to-end path. Otherwise failures are impossible to localize across parser, extractor, persistence, and query layers.
- Do not add durable findings before stable identity and lifecycle semantics. Otherwise every reindex creates duplicate gaps and AI review work.
- Implement both unresolved classes before AI. AI context selection and evaluation depend on knowing whether evidence is malformed or absent.
- Do not expose AI claims in default deterministic queries. Opt-in views and human confirmation must be explicit.
- Do not make Neo4j, JCL completion, or digital-twin behavior dependencies of v1 acceptance.

## Milestone Slice Recommendation

### Phase 1: Thin Deterministic Tracer

Deliver one parser-backed COBOL construct end to end using the existing graph store and CLI/MCP query surfaces. Include exact source range, parser identity, deterministic evidence class, and a failing fallback check proving the enhanced parser is active.

**Exit outcome:** A user can index a small fixture and query the same source-backed fact by stable ID through native surfaces with AI disabled.

### Phase 2: Deterministic Breadth

Expand AST walking across program structure, paragraphs/control flow, calls, copybooks, data items, IDMS, CICS, and SQL. Preserve distinctions such as literal versus dynamic calls and physical SQL tables versus aliases/CTEs/dynamic sources.

**Exit outcome:** The required deterministic user queries work on representative accepted fixtures without regex-only blind spots.

### Phase 3: Stable Identity and Lifecycle

Specify and test identities for domain artifacts, source occurrences, edges, findings, revisions, and parser versions. Prove identical reruns do not duplicate active facts and changed inputs close or supersede stale records.

**Exit outcome:** Users can compare runs and trust that changes reflect evidence changes rather than ID churn.

### Phase 4: Explicit Unresolved Evidence

Persist bounded `PARSE_UNRESOLVED` findings from named unparsed tails, `ERROR`, and `MISSING` nodes. Persist `EXTERNAL_UNRESOLVED` placeholders for observed references whose artifacts were not retrieved. Add impact and blockage queries.

**Exit outcome:** Users can distinguish parser limitations from missing-estate limitations and rank both by graph impact with AI disabled.

### Phase 5: Optional AI Enrichment Foundation

Build deterministic context packages, provider policy gates, strict output schemas, first-class `AI_INFERRED` claims, and human confirmation/contradiction/supersession. Evaluate against a curated reference set and ensure default deterministic views exclude unreviewed claims.

**Exit outcome:** AI adds reviewable hypotheses without changing deterministic facts or becoming an indexing dependency.

## MVP Recommendation

Prioritize:

1. One thin parser-to-SQLite-to-CLI/MCP deterministic tracer with source provenance and parser-version guard.
2. Deterministic AST breadth for the accepted COBOL, DATA DIVISION, IDMS, CICS, and SQL contracts.
3. Stable IDs and idempotent lifecycle before any durable unresolved or AI records.
4. Explicit `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` findings with user-facing impact queries.
5. One narrow optional AI workflow, preferably classification of an existing parse finding into a schema-valid `AI_INFERRED` claim, followed by human confirmation or rejection.

Defer:

- **Full JCL parsing, PROC expansion, symbolic resolution, and execution modeling:** separate mainframe language/runtime problem; only missing-JCL references belong in v1 gap reporting.
- **Behavioral and live-synchronized digital twin:** requires trustworthy static identity plus runtime/scheduler/data feeds not established here.
- **Neo4j synchronization:** keep current SQLite authority and manual export unless a measured modernization query cannot be served acceptably by native traversal.
- **Broad AI business-concept and end-to-end transaction inference:** first prove one bounded claim lifecycle and an evaluation dataset.
- **Automatic promotion of AI claims:** human confirmation and deterministic evidence remain separate trust transitions.

## Confidence Assessment

| Finding area | Confidence | Basis |
|--------------|------------|-------|
| Existing COBOL extractor scope and limitations | HIGH | Direct inspection of `CobolExtractor.Extract`, `extractProcedure`, and extractor tests |
| Enhanced parser recovery in Gortex integration | HIGH | Fresh passing execution of `TestErrorCascade -enhanced-parser` plus test-source inspection |
| Full parser acceptance totals | MEDIUM-HIGH | Consistent project and handoff records tied to merge `97ac9f1`; not all upstream parser suites were rerun in this task |
| Existing graph provenance/unresolved substrate | HIGH | Direct inspection of graph node fields, edge origin usage, unresolved namespaces, and epistemic-boundary code |
| Need for explicit finding and AI-claim models | HIGH | Current-code searches found no milestone-specific review-status or AI-origin model; handoff requires semantics generic unresolved edges cannot express |
| Optional AI workflow shape | MEDIUM | Strong product constraints and existing provider substrate, but evaluation thresholds and approved providers remain phase-specific decisions |

## Sources

### Project and code evidence

- `.planning/PROJECT.md` -- milestone scope, constraints, decisions, shipped extractor fixes, parser baseline.
- `docs/ai-enhanced-cobol-graph-handoff.md` -- evidence classes, graph candidates, finding/claim contracts, security constraints, non-goals.
- `internal/parser/languages/cobol.go` -- current regex extractor and unresolved call/copy behavior.
- `internal/parser/languages/cobol_test.go` -- current expected program, paragraph, call, dynamic-call, copybook, and comment behavior.
- `internal/parser/forest/cobolprobe/README.md` -- acceptance harness, baseline measurements, `go.work` integration boundary, and explicit statement that no Tree-sitter COBOL extractor is registered yet.
- `internal/parser/forest/cobolprobe/cascade_test.go` -- executable parity gates for IDMS, four CICS shapes, SQL in procedure/data contexts, and negative controls.
- `go.work` -- local replacement to `tree-sitter-cobol-upgrade/forest-shim/cobol`.
- `internal/graph/node.go` -- current node vocabulary and source-position fields.
- `internal/graph/edge.go` -- current edge provenance/origin/confidence substrate.
- `internal/graph/epistemic_boundary.go` -- current query-time unresolved/external boundary reporting.
- Fresh command evidence, 2026-09-15: `go test ./internal/parser/forest/cobolprobe -run TestErrorCascade -count=1 -enhanced-parser` passed.
- Fresh command caveat, 2026-09-15: `go test ./internal/parser/languages -run 'TestCobolExtractor_' -count=1` was blocked during dependency setup by HTTP 403 for `golang.org/x/image@v0.46.0`; this is not a COBOL test failure.

### External documentation

- Tree-sitter, "Query Syntax -- ERROR and MISSING nodes": https://tree-sitter.github.io/tree-sitter/using-parsers/queries/1-syntax.html -- official documentation confirms both failure forms are queryable. Confidence classified LOW by the required provider seam, so this source supports but does not independently establish Gortex behavior.
- Tree-sitter, "Basic Parsing -- Syntax Nodes": https://tree-sitter.github.io/tree-sitter/using-parsers/2-basic-parsing.html -- official documentation confirms node byte and row/column positions. Confidence classified LOW by the required provider seam; exact Gortex persistence remains a local acceptance responsibility.

## Open Questions for Phase Planning

- Which mainframe concepts need new node/edge kinds versus generic kinds plus metadata? Current generic kinds are insufficiently expressive, but schema proliferation should be avoided.
- How will parser content identity be made portable beyond the ignored local `go.work` boundary so CI and users cannot silently run the stock grammar?
- How should duplicate copybook names resolve across libraries and repositories without encoding environment-specific search order too early?
- What exact storage representation gives findings and claims temporal history without overloading ordinary graph nodes?
- Which approved model/provider policy applies to proprietary source, and what reference dataset and thresholds determine whether the narrow AI slice is acceptable?
