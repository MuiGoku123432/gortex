# AI-enhanced COBOL graph milestone handoff

**Status:** Ready for GSD milestone planning
**Date:** 2026-09-14
**Project:** Gortex Mainframe Engine
**Implementation repo:** `/Users/e1001547-mbp-it/repos/mine/GoApps/gortex`
**Parser repo:** `/Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade`
**Parser baseline:** `main` at merge commit `97ac9f1`
**Target store:** Gortex's existing persistent graph first; Neo4j integration must be decided during planning

---

## 1. How to use this document

Use this file as the primary context document when starting the next GSD milestone in the gortex
repository.

Suggested starting request:

```text
Start a new GSD milestone for the AI-enhanced COBOL graph described in
`docs/ai-enhanced-cobol-graph-handoff.md`. Treat the completed Tree-sitter COBOL parser as the
deterministic evidence baseline. First reconcile this handoff with gortex's existing graph schema,
SQLite persistence, provenance model, COBOL extractor, LLM providers, and multi-repo architecture.
Do not assume Neo4j replaces the existing graph store. Research and decide the integration boundary
before creating the roadmap, and run the AI integration design workflow for AI-facing phases.
```

The intended workflow is:

1. Run the GSD new-milestone workflow in gortex.
2. Use this document as milestone input and update `.planning/PROJECT.md` where it is stale.
3. Research the existing graph, COBOL extraction, provenance, persistence, and LLM extension points.
4. Decide the storage and synchronization architecture before roadmap creation.
5. Run the GSD AI integration phase workflow before planning AI implementation phases.
6. Plan deterministic graph extraction before AI enrichment.

---

## 2. Objective

Build a detailed, trustworthy graph of a mainframe estate from available COBOL and related
artifacts. The completed Tree-sitter COBOL parser supplies deterministic, source-positioned facts.
Gortex converts those facts into graph entities and relationships, records unresolved parser and
retrieval gaps, and uses AI to propose separately labeled enrichments where deterministic analysis
cannot parse, resolve, or verify the code.

The system must remain useful when the retrieved estate is incomplete. Missing copybooks, called
programs, JCL, DB2 metadata, CICS definitions, IDMS metadata, generated source, and preprocessor
outputs must become explicit graph gaps rather than silent omissions or invented facts.

The governing rule is:

> Deterministic extraction establishes facts. Missing evidence remains explicit. AI proposes
> evidence-linked claims, but never rewrites parser truth or turns unavailable source into a
> verified fact.

---

## 3. Completed parser baseline

The parser milestone is complete in `tree-sitter-cobol-upgrade`:

- 25 of 25 implementation plans completed.
- Delivery, IDMS, CICS, and SQL phases completed.
- Phase 4 verification passed 7 of 7 requirements.
- Phase 4 security review reported `SECURED` with zero open threats.
- All 149 non-comment Tree-sitter corpus assertions passed.
- Gortex cascade tests preserve 20 of 20 paragraphs and data items for procedure and DATA DIVISION
  SQL cases.
- NIST COBOL-85 passed with 371 successes, zero failures, and the unchanged 11-test skip list.
- DCC and estate differentials reported zero reclassification and zero conversion.
- The final parser and queries are merged to parser `main` at `97ac9f1` and materialized in its
  `forest-shim/cobol` module, which gortex consumes through its ignored local `go.work` integration
  boundary.

The deterministic parser exposes named, source-positioned nodes for:

- COBOL programs, divisions, sections, paragraphs, statements, and data declarations
- IDMS DML statements, record names, and set names
- CICS transaction, program, map, and mapset operands
- SQL statements, physical tables, aliases, CTEs, includes, and dynamic sources
- bounded `idms_unparsed_tail`, `cics_unparsed_tail`, and `sql_unparsed_tail` regions
- localized Tree-sitter errors and missing nodes where the grammar cannot recover fully

Parser source documents:

- `/Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/docs/ai-enhanced-cobol-graph.md`
- `/Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/docs/spec/SPEC-001-nodes-for-edges.md`
- `/Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/docs/baseline.md`
- `/Users/e1001547-mbp-it/repos/mine/devDeps/tree-sitter-cobol-upgrade/docs/sql-tail-census.md`

These files are supporting evidence. This handoff is self-contained enough to begin planning if the
parser repository is unavailable.

---

## 4. Existing gortex substrate

Planning must extend existing gortex architecture rather than inventing a parallel engine.

Relevant existing capabilities include:

- Tree-sitter extraction through language-specific extractors.
- A persistent SQLite graph store written during indexing.
- Incremental re-indexing and cross-repository resolution.
- Existing node and edge types, metadata, provenance, confidence, and origin concepts.
- Existing dataflow edges such as `value_flow`, `arg_of`, and `returns_to`.
- Existing COBOL extraction and `cobolprobe` measurement infrastructure.
- Existing optional LLM providers and graph-grounded `ask` agent.
- MCP, CLI, HTTP, and UI surfaces for querying the graph.

Canonical references:

- `docs/architecture.md`
- `docs/llm.md`
- `docs/multi-repo.md`
- `docs/contracts.md`
- `docs/server.md`
- `internal/parser/forest/cobolprobe/`

Important architectural correction:

> Gortex already uses SQLite as its durable graph store. Neo4j must not be assumed to replace it.
> Planning must choose between native gortex storage only, Neo4j export/materialization, or a
> synchronized secondary projection. The choice must be evidence-based and preserve gortex's
> incremental indexing and query behavior.

---

## 5. Intended system boundary

```text
retrieved and preprocessed mainframe artifacts
                    |
                    v
       deterministic Tree-sitter parsing
                    |
                    v
      deterministic gortex graph extraction
                    |
          +---------+----------+
          |                    |
          v                    v
 verified graph facts   unresolved findings
          |                    |
          |                    v
          |            bounded AI review
          |                    |
          +---------+----------+
                    |
                    v
       provenance-aware active graph
                    |
          +---------+----------+
          |                    |
          v                    v
 native gortex queries   optional Neo4j projection
```

The deterministic graph must work with AI disabled. AI enrichment is an optional layer and may not
be required for indexing, persistence, or basic graph queries.

---

## 6. Evidence classes

Every material graph claim must be distinguishable by evidence class.

| Evidence class | Meaning | Active fact? |
|---|---|---|
| `DETERMINISTIC` | Directly established by parser or deterministic analysis | Yes |
| `AI_INFERRED` | Proposed by AI from bounded code and graph context | No, unless a consumer opts in |
| `EXTERNAL_UNRESOLVED` | A deterministic reference points to an artifact not obtained | Reference is factual; target details are unknown |
| `PARSE_UNRESOLVED` | Source exists, but parsing or deterministic verification failed | Only the source and failure are factual |
| `HUMAN_CONFIRMED` | A reviewer accepted an inference | Yes, with retained provenance |
| `CONTRADICTED` | Later evidence disproved a claim | No; retained for audit |
| `SUPERSEDED` | A newer extraction or claim replaced it | No; retained for lineage |

Confidence is not evidence. A high-confidence AI result remains `AI_INFERRED`. A deterministic
reference to a missing program can be certain even when the target implementation is unavailable.

Before introducing new fields, planning must map these classes onto gortex's current origin,
confidence, metadata, and temporal-history mechanisms.

---

## 7. Deterministic graph scope

The first milestone slice should extract and preserve deterministic entities and relationships from
the completed parser contract.

Candidate entities:

- estate, repository, retrieval revision, and source file
- COBOL program, division, section, paragraph, statement, and data item
- copybook and include member
- external or missing program
- CICS transaction, program, map, and mapset
- DB2 table, alias, CTE, include, and dynamic SQL source
- IDMS record and set
- parser finding, missing artifact, source range, and claim

Candidate relationships:

```text
(PROGRAM)-[:CONTAINS]->(PARAGRAPH)
(PARAGRAPH)-[:PERFORMS]->(PARAGRAPH)
(PROGRAM)-[:CALLS]->(PROGRAM)
(PROGRAM)-[:MAY_CALL]->(PROGRAM)
(PROGRAM)-[:USES_COPYBOOK]->(COPYBOOK)
(STATEMENT)-[:READS_TABLE]->(DB2_TABLE)
(STATEMENT)-[:WRITES_TABLE]->(DB2_TABLE)
(STATEMENT)-[:USES_RECORD]->(IDMS_RECORD)
(STATEMENT)-[:USES_SET]->(IDMS_SET)
(STATEMENT)-[:LINKS_PROGRAM]->(CICS_PROGRAM)
(STATEMENT)-[:USES_TRANSACTION]->(CICS_TRANSACTION)
(STATEMENT)-[:USES_MAP]->(CICS_MAP)
(FINDING)-[:LOCATED_AT]->(SOURCE_RANGE)
(FINDING)-[:BLOCKED_BY]->(MISSING_ARTIFACT)
(CLAIM)-[:SUPPORTED_BY]->(SOURCE_RANGE)
```

These names are candidates, not locked schema. GSD research must reconcile them with existing
`graph.NodeKind`, `graph.EdgeKind`, metadata, external-node, symbol-ID, and cross-repo conventions.

---

## 8. Parser and retrieval gap contract

A parser miss should produce a bounded finding whenever surrounding structure survives.

Preferred outcomes:

1. Fully recognized AST node and deterministic graph facts.
2. Named unparsed tail inside a known construct with exact source range.
3. Localized Tree-sitter `ERROR` or missing node with known containing structure.
4. Uncontained recovery cascade that removes downstream declarations or paragraphs.

Outcomes 2 and 3 are AI-review candidates. Outcome 4 remains a high-severity deterministic defect;
AI may recover hypotheses but cannot make the regression gate pass.

Missing source and parser failure are separate classes:

- `PARSE_UNRESOLVED`: source exists, but deterministic interpretation failed.
- `EXTERNAL_UNRESOLVED`: a reference exists, but the target artifact was not obtained.

A deterministic reference to a missing target should still create a placeholder entity. The graph
must be able to answer which missing programs, copybooks, JCL members, schemas, or subsystem
artifacts block the most verification.

---

## 9. Finding contract

Unresolved regions should be normalized before any AI call. The finding itself is deterministic and
must be persisted even when AI is disabled.

Minimum fields:

```json
{
  "finding_id": "stable-content-derived-id",
  "repo_id": "tracked-repository-id",
  "source_path": "application/program.cbl",
  "source_revision": "commit-or-retrieval-revision",
  "range": {
    "start": {"row": 42, "column": 7},
    "end": {"row": 46, "column": 18}
  },
  "containing_symbol_id": "existing-gortex-symbol-id",
  "parser_node_type": "ERROR",
  "expected_family": "exec_sql_statement",
  "resolution_class": "PARSE_UNRESOLVED",
  "reason": "unsupported_or_ambiguous_syntax",
  "severity": "medium",
  "parser_version": "grammar-content-id",
  "review_status": "pending"
}
```

Stable IDs must support idempotent reruns, closure when a parser improves, and supersession when
source or parser versions change.

---

## 10. AI context and output contract

The context builder should deterministically select the smallest useful package:

- unresolved source range
- containing program, division, section, and paragraph
- nearby deterministic AST and graph nodes
- relevant data declarations, assignments, and value-flow edges
- available copybooks
- callers, callees, and related cross-repo paths
- similar successfully parsed constructs
- compiler, dialect, subsystem, and preprocessing metadata
- parser diagnostics and recovery behavior
- explicit unavailable-artifact list
- redaction metadata

AI may:

- classify unsupported syntax or preprocessing needs
- infer dynamic call candidates and resource names
- connect aliases and naming variants
- propose business concepts, data lineage, and transaction flows
- summarize programs and paragraphs
- identify recurring parser gaps and recommend deterministic work
- state what missing artifact would verify an inference

AI must not:

- overwrite deterministic nodes or edges
- silently replace parser errors
- invent missing artifact contents
- promote confidence into evidence
- hide contradictory evidence
- make parser regression gates pass
- omit unavailable-context declarations

Every AI output must pass a deterministic schema validator before graph insertion.

---

## 11. Claim and provenance requirements

A material AI inference must retain:

- stable claim ID
- model provider and identifier
- model, prompt-template, and context-builder versions
- timestamp
- confidence and optional calibrated band
- concise rationale
- supporting source ranges and graph IDs
- missing-context declaration
- review status and reviewer identity
- contradiction and supersession links

Planning must decide whether claims are represented as:

- first-class claim nodes
- evidence-bearing graph relationships
- a hybrid based on claim complexity

First-class claims are the safer default for material assertions because multiple models and humans
can disagree without duplicating domain entities or mutating deterministic facts.

---

## 12. Stable identity and idempotency

Repeated indexing of identical source and versions must produce the same active deterministic
subgraph and findings.

Identity needs separate rules for:

- domain artifacts such as programs, copybooks, tables, and maps
- source occurrences and exact ranges
- deterministic relationships
- unresolved findings
- AI claims and reviews
- retrieval revisions and parser versions

Gortex's current multi-repo ID form, `<repo_prefix>/<path>::<Symbol>`, must be the starting point.
Do not create a second incompatible identity system without proving why the existing form cannot be
extended.

Incremental indexing must close or supersede stale findings and claims without deleting audit
history. If Neo4j is used, synchronization must be idempotent and resilient to partial retries.

---

## 13. Neo4j decision gate

The long-term target includes Neo4j, but gortex already has a production graph model and SQLite
persistence. The first planning phase must compare:

| Option | Description |
|---|---|
| Native only | Extend gortex's SQLite graph and expose all mainframe/AI queries through existing surfaces |
| Export | Keep SQLite authoritative and periodically materialize a Neo4j projection |
| Synchronized projection | Stream deterministic and reviewed graph changes into Neo4j incrementally |
| Replace store | Make Neo4j authoritative instead of SQLite |

Replacing the store should be presumed high-risk because it cuts across indexing, incremental
reconciliation, daemon behavior, query tools, tests, packaging, and the zero-dependency design.
It must not be selected without strong evidence.

The likely starting hypothesis is SQLite authoritative plus a Neo4j projection, but this is not a
locked decision. GSD research must evaluate actual modernization queries, scale, deployment,
consistency, and operational requirements.

---

## 14. Security and data-governance constraints

Mainframe source and estate metadata may be proprietary or regulated.

Requirements:

- AI features remain optional and off by default.
- Local or approved enterprise models are preferred for raw source.
- External providers require explicit configuration and authorization.
- Context is bounded and redacted where practical.
- Secrets, credentials, and protected literals are excluded from model context.
- Prompt, context, output, and claim retention are configurable.
- Logs do not retain raw proprietary snippets by default.
- Neo4j exports do not weaken repository or project scoping.
- Every external AI call is attributable to provider, model, policy, and source scope.

Existing gortex provider behavior in `docs/llm.md` should be extended rather than bypassed.

---

## 15. Proposed milestone

**Name:** Deterministic COBOL Graph Extraction and AI Enrichment Foundation

**Goal:** Convert the completed COBOL parser's named AST contract into a reproducible,
provenance-aware graph; represent parser and retrieval gaps explicitly; then add optional,
validated AI claims without changing deterministic facts.

Suggested phases:

1. **Architecture and schema decision**
   Reconcile the parser contract with existing gortex graph types, provenance, identity,
   persistence, and Neo4j options. Produce an accepted schema and storage ADR.
2. **Deterministic COBOL graph extraction**
   Extract source-positioned IDMS, CICS, SQL, program, paragraph, data, copybook, call, and resource
   facts into the native graph.
3. **Stable IDs and incremental lifecycle**
   Prove identical reruns are idempotent and source/parser changes close or supersede observations.
4. **Gap and missing-artifact model**
   Emit `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` findings without requiring AI.
5. **Neo4j projection foundation**
   Implement the selected projection or synchronization boundary with retry-safe identity and
   provenance.
6. **AI context and structured review**
   Build bounded context packages, provider-policy enforcement, schemas, and output validation.
7. **AI claims and human review**
   Persist inferred claims separately and support confirmation, contradiction, and supersession.
8. **Evaluation and end-to-end validation**
   Validate deterministic repeatability, missing-source behavior, inference quality, security, and
   modernization queries against intentionally incomplete artifact sets.

GSD may split or reorder these phases after research. Deterministic extraction and gap modeling must
precede AI claim insertion.

---

## 16. Milestone success criteria

The milestone is successful when:

1. Identical available source, parser, extractor, and configuration produce the same active
   deterministic graph.
2. IDMS, CICS, SQL, program, paragraph, data, copybook, and call relationships carry exact source
   provenance.
3. Unsupported parser regions and unavailable artifacts are queryable findings, not silent losses.
4. Incremental re-indexing closes or supersedes stale observations without duplicate active facts.
5. AI can be disabled without breaking deterministic indexing or queries.
6. Every AI claim is schema-valid, evidence-linked, confidence-scored, reviewable, and replaceable.
7. Deterministic and human-confirmed views exclude unreviewed AI claims by default.
8. Proprietary source cannot reach an unapproved provider or persistent log accidentally.
9. The chosen Neo4j boundary is idempotent, scoped, and does not undermine gortex's native graph.
10. End-to-end queries distinguish known, inferred, unresolved, contradicted, and missing evidence.

---

## 17. Questions GSD must answer

1. Which existing gortex node and edge kinds can represent COBOL domain concepts unchanged?
2. Which mainframe concepts require new kinds versus metadata on existing kinds?
3. What provenance and confidence machinery already exists and should be reused?
4. How are parser version, retrieval revision, and source range represented?
5. How should copybooks and duplicate member names resolve across libraries and repositories?
6. How should dynamic calls and SQL dynamic sources use existing dataflow analysis?
7. What is the authoritative store, and what exact purpose does Neo4j serve?
8. Which queries require Neo4j rather than existing graph traversal?
9. Which model providers are approved for proprietary source?
10. What context and claim data may be retained?
11. What confidence calibration and human-review thresholds are required?
12. Which downstream tools may consume `AI_INFERRED` claims?
13. How are claims invalidated when source, parser, prompt, or model versions change?
14. What reference dataset will measure AI classification and inference quality?
15. What parser-gap frequency or graph impact justifies moving an AI-handled pattern into the
    deterministic grammar?

---

## 18. Non-goals for the first milestone

- Parsing every COBOL dialect or every mainframe language.
- Full behavioral simulation or a live-synchronized digital twin.
- Replacing deterministic parsing with AI.
- Treating all AI output as graph truth.
- Replacing gortex's SQLite store without an accepted architecture decision.
- Sending proprietary estate source to public AI providers by default.
- Reimplementing existing gortex graph, dataflow, LLM-provider, or query infrastructure.
- Solving every missing artifact through inference.

---

## 19. Deliverables expected from planning

Before implementation begins, GSD should produce:

- updated `.planning/PROJECT.md`
- milestone requirements with explicit deterministic and AI requirement IDs
- domain and architecture research grounded in the current gortex codebase
- an AI design specification covering model contract, evaluations, guardrails, and monitoring
- a Neo4j/storage ADR
- canonical graph schema and identity rules
- phased roadmap with requirement coverage
- per-phase verification criteria
- threat model for source handling, external providers, and graph trust boundaries

The first implementation plan should be a thin deterministic tracer from one existing COBOL parser
node through gortex's native graph and query surfaces. It should not begin with an AI call or a
Neo4j migration.
