# v1.0 Graph Schema Evidence and Direction

**Milestone:** v1.0 Deterministic COBOL Graph Extraction and AI Enrichment Foundation
**Status:** Planning contract; candidate vocabulary remains subject to thin-tracer evidence
**Date:** 2026-09-15

## Purpose

Define how candidate mainframe concepts fit Gortex's current graph without creating a parallel model. This document fixes semantic boundaries, identity layers, provenance distinctions, lifecycle, scope, and reconciliation direction. It does not authorize broad product implementation or lock every candidate kind before the thin tracer proves the native path.

## Governing Rules

1. Deterministic parser and resolver output establishes facts.
2. Missing or uninterpretable evidence remains explicit and queryable.
3. AI emits claims, never mutations of deterministic facts.
4. Existing repository/workspace/project identity and native SQLite lifecycle remain the substrate.
5. New kinds are additive and justified by required filtering, identity, or relationship semantics.
6. Confidence is not evidence, and origin is not lifecycle state.

## Current Primitive Evidence

- `internal/graph/node.go::Node` already carries `ID`, `Kind`, logical names, file path, line/column range, language, metadata, `RepoPrefix`, `WorkspaceID`, and `ProjectID`.
- `internal/graph/edge.go::Edge` already carries endpoints, `EdgeKind`, source file/line, numeric and labeled confidence, origin, provenance tier, cross-repository marker, and internal metadata.
- `internal/indexer/indexer.go::Indexer.applyRepoPrefix` prefixes local IDs and file paths, stamps workspace/project identity, and intentionally leaves `unresolved::` targets unprefixed for later resolution.
- `internal/parser/languages/cobol.go::CobolExtractor.Extract` shows the current compatibility baseline: files, programs/paragraphs as generic functions, divisions as types, sections as methods, `defines`, `imports`, `calls`, and unresolved target namespaces. It is regex-based and too lossy to define the final schema.
- `internal/graph/store_sqlite/store.go::Store.AddBatch` is the transactional persistence seam.
- `internal/indexer/incremental_batch.go::Indexer.commitStructuralIncrementalBatch` and `internal/resolver/cross_repo_incremental.go::CrossRepoResolver.ResolveFilesAndIncoming` are the lifecycle and re-resolution seams.

## Candidate Concept Mapping

The mapping favors reuse where semantics already match and additive vocabulary only where generic kinds would make required queries ambiguous.

| Candidate concept | Current primitive / v1.0 direction | Identity and evidence notes |
|---|---|---|
| Estate | Metadata or project/workspace aggregation; no new node initially | Do not duplicate `WorkspaceID`/`ProjectID` until a user query requires an estate entity. |
| Repository | Existing repository prefix and repository graph records | Scope boundary, not a COBOL domain kind. |
| Retrieval revision | Generation/metadata record direction | Provenance input, never embedded in stable declaration IDs. |
| Source file | Existing `KindFile` | Repository-relative path plus repository identity. |
| COBOL program | Thin tracer uses `KindFunction` plus `cobol_kind=program`; strong candidate for additive `program` kind after tracer | Logical declaration identity; exact occurrence range is provenance. |
| Division | Existing containment node plus typed `cobol_kind=division` unless required queries justify a kind | Owned by program/file. |
| Section | Existing containment node plus typed metadata initially | Scope-qualified under program/division. |
| Paragraph | Strong candidate for additive `paragraph` kind | Qualified by owning program; do not identify by short name or line alone. |
| Statement | Existing statement/expression primitive if queryable; otherwise source occurrence metadata/node only for evidence-bearing statements | Occurrence identity includes revision/range/content family. Avoid one node per token. |
| Data item | Strong candidate for additive `data_item` kind | Qualified by containing record/group and source owner. Preserve level number and raw name as metadata. |
| Copybook/include member | Strong candidate for additive `copybook` or existing module/file plus role metadata | Identity requires repository plus library/member resolution domain, not short name alone. |
| Called program | Existing resolved node plus `calls`; absent target remains `unresolved::` placeholder | Static literal call can be deterministic; dynamic value-derived candidate retains inferred origin. |
| DB2 physical table | Reuse `KindTable`; use existing `reads`/`writes` where semantics match | Include schema/resolution domain. Alias and CTE are not physical tables. |
| SQL alias / CTE | Occurrence or scoped symbol metadata/node only when lineage queries require it | Scoped to statement/query; never merge with physical table. |
| Dynamic SQL source | Unresolved occurrence/finding, optionally linked by dataflow | Do not invent a physical target. |
| IDMS record / set | Additive resource kinds only if generic resource cannot support direct filtering | Qualified by schema/subsystem context. |
| CICS program / transaction / map / mapset | Generic resource plus typed role initially; additive kinds if required native queries demand them | Qualify by applicable subsystem/region/configuration context. |
| Parser finding | Additive first-class `finding` node or typed finding record projected as a node | Source-owned, deterministic, exact range, active lifecycle. |
| Missing artifact | Existing unresolved placeholder plus finding | Placeholder proves reference existence, not target contents. |
| Source range | Typed node/edge range fields plus evidence metadata; separate node only when multiple claims must cite one occurrence | Include byte/row/column and source revision. |
| AI/human claim | First-class claim ledger record; optional graph projection | Never share file-eviction semantics with deterministic rows. |
| Review | Append-only review event linked to claim | Reviewer identity and policy scope required. |

## Minimal Additive Direction

The thin tracer should add no new kind: map a named `PROGRAM-ID` observation to existing `KindFunction`, `EdgeDefines`, exact range, and `cobol_kind=program`. This proves parser, prefixing, SQLite, and CLI/MCP visibility.

After the tracer, the smallest likely additive vocabulary is:

- Node kinds: `program`, `paragraph`, `data_item`, `copybook`, `finding`, `claim`.
- Edge kinds: `performs`, `uses_record`, `uses_set`, `uses_transaction`, `uses_map`, `supported_by`, `blocked_by`, `contradicts`, `supersedes`.
- Typed fields or constants: evidence class and lifecycle status before claims become queryable.

This is a direction, not a mandate to add all listed kinds. Reuse `defines`, `contains`/`member_of`, `calls`, `imports`, `reads`, `writes`, `value_flow`, `arg_of`, and `returns_to` whenever their current semantics are exact. A new kind is justified only when metadata scanning would obscure a required query or a generic kind would imply false language semantics.

## Layered Identities

There is no single universal key. Each layer has different stability requirements.

| Layer | Canonical key direction | Lifecycle behavior |
|---|---|---|
| Repository/source owner | tracked repository identity plus normalized repository-relative path | Stable across content edits; path/repo changes create a new owner. |
| Domain declaration | `<repo-prefix>/<path>::<scope-qualified-logical-name>` as the starting grammar | Stable across line movement; revision and range are not part of declaration identity. |
| External logical artifact | typed namespace plus normalized qualified name and explicit resolution domain | Remains unresolved when library/schema/subsystem context is insufficient. |
| Source occurrence | digest of repository, path, source revision, parser-node family, exact byte/point range, and normalized payload | Changes when evidence occurrence changes; may supersede a prior occurrence. |
| Deterministic relationship occurrence | endpoints, kind, source owner, exact occurrence discriminator, and derivation origin | Multiple source occurrences must not collapse when each is material evidence. Current edge dedup behavior must be verified. |
| Finding | source owner/revision, parser fingerprint, exact range, resolution class, reason, and expected family | Same evidence reruns idempotently; source/parser change closes or supersedes. |
| Claim | proposed assertion, ordered evidence-set digest, model, prompt/context/schema versions, and policy scope | Identical inputs are idempotent; changed evidence/model/schema creates a new revision. |
| Review | claim ID, reviewer identity, decision, timestamp, and review-policy version | Append-only; never rewrites the originating claim. |

Raw lexemes and normalized lookup keys must both be retained. Case folding, hyphen handling, aliases, and dialect rules must be versioned. Line numbers alone are never durable declaration identity.

## Provenance Vocabulary

These dimensions are orthogonal and must not be collapsed:

| Dimension | Question answered | Examples |
|---|---|---|
| Evidence class | What epistemic category is this record? | `DETERMINISTIC`, `PARSE_UNRESOLVED`, `EXTERNAL_UNRESOLVED`, `AI_INFERRED`, `HUMAN_CONFIRMED`. |
| Origin | How was it produced? | `ast_resolved`, `ast_inferred`, resolver/LSP/heuristic origin. Existing `Edge.Origin` is the starting point. |
| Confidence | How uncertain is this derivation within its class? | Numeric/label confidence. It cannot promote an AI claim or demote a certain missing reference. |
| Evidence | Which exact observations support it? | Source range IDs, graph IDs, parser diagnostics, retrieval manifest entries. |
| Provenance | Which reproducible inputs and transformations produced it? | Repository/revision, parser fingerprint, extractor version, configuration, model/prompt/context/schema versions. |
| Lifecycle status | Is it currently eligible for a view? | active, pending review, confirmed, contradicted, stale, closed, superseded. |

`CONTRADICTED` and `SUPERSEDED` are lifecycle states, not evidence classes. `HUMAN_CONFIRMED` is a reviewed assertion derived from a retained claim; it does not rewrite parser evidence.

## Finding Lifecycle

A finding is deterministic and exists even when AI is disabled.

1. Normalize a named unparsed tail, localized `ERROR`/`MISSING`, or unavailable target into a stable finding.
2. Classify availability first: present-but-uninterpreted is `PARSE_UNRESOLVED`; referenced-but-not-obtained is `EXTERNAL_UNRESOLVED`.
3. Attach exact source range, containing symbol, parser fingerprint, retrieval revision, expected construct family, reason, severity, scope, and attempted resolution domain.
4. Repeated identical extraction reuses the active finding identity.
5. A successful parser improvement closes a parse finding. Arrival/resolution of the target closes an external finding. One remedy must not close the other class.
6. Changed source/parser/retrieval evidence creates a successor and closes or supersedes the old active finding while retaining audit lineage.
7. AI review may cite a finding but cannot close it.

## Claim Lifecycle

1. A policy-approved context builder selects active scoped evidence and records a context manifest plus missing-context declaration.
2. Provider output must match a closed, versioned schema.
3. Deterministic validation resolves every evidence ID and proposed endpoint within the request scope.
4. Accepted output is stored as `AI_INFERRED` in an append-only claim ledger, excluded from deterministic views by default.
5. Human review appends confirmation, contradiction, or rejection with reviewer and policy version.
6. Source, parser, context, prompt, schema, or cited-evidence changes mark dependent claims stale or superseded; history remains.
7. Conflicting claims coexist and are linked by contradiction/supersession rather than last-write-wins mutation.
8. Human-confirmed projections remain distinct from deterministic parser facts.

## Scope Contract

Every source-owned node, edge, finding, claim, review, context manifest, and export record must carry or inherit:

- workspace ID
- project ID
- repository identity/prefix
- normalized repository-relative path where applicable
- retrieval/source revision
- explicit cross-repository status for derived links
- mainframe resolution domain where applicable, such as copybook library order, DB2 schema, CICS region, or IDMS schema

The same allow-set must govern graph reads, source reads, resolution, AI context, logs, claims, and export. Missing or ambiguous scope fails closed. Name equality never authorizes cross-repository resolution.

## Incremental Reconciliation

Deterministic records are file/source-owner managed and reconciled as a set:

1. Fingerprint source/retrieval revision, parser grammar content, extractor schema, and material configuration.
2. Compute the complete desired observations/findings for the changed owner.
3. In one native lifecycle, evict or deactivate prior file-owned active rows, insert the replacement generation, and rerun affected local/incoming resolution.
4. Close findings absent from the replacement set and invalidate dependent derived state.
5. Preserve append-only claim/review history, but remove stale claims from eligible active views.
6. Require canonical active deterministic equality between incremental reconciliation and a clean rebuild.

The current `AddBatch`, structural incremental commit, unresolved placeholder convention, and incoming cross-repo resolver are seams to extend. A separate COBOL reconciliation engine is rejected.

## Required Verification

- Duplicate short names across files, programs, libraries, schemas, and repositories do not collapse.
- Multiple evidence-bearing occurrences are preserved or explicitly modeled.
- Exact row/column/byte range survives SQLite round-trip.
- Identical reruns produce identical active IDs and no duplicates.
- Edit/delete/restore/parser-change sequences equal clean rebuilds.
- `AI_INFERRED` at confidence 1.0 remains outside deterministic views.
- Missing target placeholders never satisfy queries requiring parsed implementation.
- `PARSE_UNRESOLVED` and `EXTERNAL_UNRESOLVED` close only through their correct remedies.
- Scope canaries never cross query, context, claim, log, or export boundaries.

## Unresolved Placeholders

These must be answered by the relevant implementation phase, not guessed here:

- Exact enhanced-parser API and persisted grammar fingerprint equivalent to baseline `97ac9f1`.
- Copybook library concatenation/search order and retrieval manifest format.
- Canonical qualifiers for DB2, CICS, and IDMS resources.
- Whether current edge identity preserves distinct same-line/source occurrences; if not, whether occurrence/evidence nodes are required.
- Final subset of additive node/edge kinds based on named query ergonomics.
- Claim-ledger schema, encryption, retention, and raw-context policy.
- Who may confirm claims and whether confirmation is source/parser-version scoped.
- Default eligibility of human-confirmed claims in native and exported views.

## Sources

- `.planning/PROJECT.md`
- `docs/ai-enhanced-cobol-graph-handoff.md`, sections 6 through 12
- `.planning/research/SUMMARY.md`
- `.planning/research/ARCHITECTURE.md`, especially "Canonical Schema Direction"
- `.planning/research/PITFALLS.md`, especially pitfalls 1 through 6 and 10 through 15
- `internal/graph/node.go::Node`
- `internal/graph/edge.go::Edge`
- `internal/parser/languages/cobol.go::CobolExtractor.Extract`
- `internal/indexer/indexer.go::Indexer.applyRepoPrefix`
- `internal/graph/store_sqlite/store.go::Store.AddBatch`
- `internal/indexer/incremental_batch.go::Indexer.commitStructuralIncrementalBatch`
- `internal/resolver/cross_repo_incremental.go::CrossRepoResolver.ResolveFilesAndIncoming`
