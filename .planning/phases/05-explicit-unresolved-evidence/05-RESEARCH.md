# Phase 5: Explicit Unresolved Evidence - Research

**Researched:** 2026-10-19
**Domain:** Deterministic COBOL parse/retrieval findings on the Gortex graph (extractor, resolver, SQLite sidecar, lifecycle ledger)
**Confidence:** HIGH for codebase facts and producer behavior (read and probed this session); MEDIUM for the recommended designs in discretion areas

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Carried forward (do not re-decide)
- Provenance lives in flat `prov_*` and namespaced Meta keys, with no new `graph.Node` fields or SQLite columns on nodes or edges (P2 D-08).
- Never invent coordinates. A gap with no range carries the producer's `AbsenceReason`, surfaced as-is (readiness §4).
- Do not pin the producer's `gaps[].id` or `ledger.configuration_id` in tests (readiness §4).
- Nodes and edges are hard-evicted by `file_path`. The node and edge store holds no inactive rows. Lifecycle state applies to findings only (P4 D-05).
- Line numbers never discriminate identity (P2 D-10, ID-02).
- Unbound edges keep their `cobol_unresolved_reason` vocabulary (P3 D-12). Phase 5 builds findings on top of it and does not replace it.
- Binding follows P3 D-09/D-11, P4 D-09 and P4 D-19:
  - CALL and COPY bind only on an exact single match within the same repository plus the allowed-repository union.
  - `missing` means no candidate in that union.
  - `COPY IDMS` members are never bound (P3 D-19).
- Incremental indexing must equal a clean rebuild, compared on the canonical dump, which includes all deterministic meta (P4 D-01..D-04).
- Ownership health and SEC-08 (P4 D-14, D-15): every new node must be owned or sit in an enumerated exempt namespace.
- Fixtures are invented COBOL only, because the repo is public (P2 D-14).

#### Finding identity
- **D-01:** A finding ID is anchored to:
  - the containing symbol ID
  - the resolution class
  - the gap kind
  - the node kind or expected family
  - a digest of the gap's own original source text

  A `#k` ordinal is added only when identical evidence text appears more than once in the same container. Edits elsewhere in the file and line shifts keep the ID. Editing the gap's own text produces a new ID.

  This deliberately overrides two weaker schemes:
  - the producer's `gapID`, which hashes the whole document's content ID plus byte offsets;
  - the "source revision in the key" direction in `GRAPH-SCHEMA.md`'s Layered Identities table.

  — **Reversibility:** costly — finding IDs are persisted, projected to Neo4j, and will be cited by Phase 8 claims. Changing the scheme closes and reopens every finding in the estate.
- **D-02:** The parser or extractor version is **not** part of the ID. If a new parser version still produces the same gap, the finding keeps its ID and its parser-version fields are re-stamped. Claims handle their own parser-version staleness (REVIEW-05).
- **D-03:** Closed and superseded are distinguished deterministically:
  - **Closed:** the evidence is gone. Either the gap is no longer produced, or the missing target was retrieved or bound.
  - **Superseded:** a different active finding of the same class and kind now sits in the same container. The old record names its successor ID.

  Never match across containers, and never guess.
- **D-04:** There is one `EXTERNAL_UNRESOLVED` finding per reference occurrence (each `CALL 'PGMX'` or `COPY X` statement), with its exact range and containing symbol. Artifact-level counts are aggregated from findings or from the placeholder's in-edges.

#### Storage and history
- **D-05:** Add one new node kind, **`finding`**. This is justified by the `GRAPH-SCHEMA.md` rule, because a generic kind would imply false semantics. Each active finding is:
  - a repository-prefixed node owned by its source file (`file_path` eviction applies);
  - carrying its fields as Meta;
  - searchable, included in the canonical dump, covered by the ownership audit, and projected to Neo4j through the existing projection.

  — **Reversibility:** costly — the kind is persisted in SQLite, projected as a Neo4j label, and consumed by Phase 6 queries.
- **D-06:** Closed and superseded findings do not stay as nodes. Every close or supersede transition is appended to a new sidecar SQLite table, **`finding_history`**. Each row holds:
  - finding ID
  - prior status and new status
  - successor ID (for supersede)
  - evidence and parser stamps
  - repository, workspace, and project scope

  The node store therefore never holds inactive rows. The `file_index_failures` table is the precedent for an additive sidecar table.

  — **Reversibility:** costly — this is a schema migration (v23 or later) plus a new persisted table that later claim lineage reads.
- **D-07:** Finding history is best-effort and not canonical:
  - It survives normal reindexing, incremental updates, and daemon restarts.
  - A store rebuild (a schema bump or a fresh store) starts it empty.
  - It is excluded from the incremental-equals-clean comparison. Only active finding nodes are canonical.
- **D-08:** Users see findings in Phase 5 through existing surfaces only:
  - `get_symbol` by exact ID, through both MCP and CLI (the P2 D-11 acceptance pattern);
  - search by name or kind;
  - edges to the containing symbol and to the placeholder;
  - `index_health`, which reports active finding counts by class.

  A list or filter verb waits for Phase 6 (QUERY-08).

#### What becomes a finding
- **D-09:** `EXTERNAL_UNRESOLVED` covers missing artifacts only:
  - the edge reason `missing` (a CALL, LINK, XCTL, or COPY target with no candidate in the allowed scope);
  - `idms_dictionary_member` (the IDMS dictionary was not retrieved);
  - the producer's `external-reference` gaps (an `-INC` member that was not obtained, or a cyclic one).

  `ambiguous` and `dynamic` stay edge-only reasons. The artifact either exists more than once, or the target cannot be known statically. Neither is "not retrieved". Phase 6 QUERY-03 already distinguishes them through edge reasons.
- **D-10:** `PARSE_UNRESOLVED` comes from the producer's gap list only:
  - `error`, `missing` node, `unparsed-tail`, `lost-capture`, `unmapped-capture`;
  - the document-scope gaps `no-tree`, `budget-exceeded`, and `invocation-failure`;
  - the cascade finding in D-14.

  Gortex's in-source reasons (`label_not_found`, `label_not_in_copybook`, `not_declared_in_*`, `cursor_*`, `scope_role_unknown`, `caller_unowned`) stay edge reasons. Phase 5 makes them scope-accurate, as deferred from Phase 3:
  - **WR-07:** a twice-declared cursor is `ambiguous`, not `cursor_not_declared`.
  - **WR-09:** a procedure copybook's cursor use gets a copybook-scoped reason.
  - **IN-08:** a program nested under unresolved containment does not report a plain `missing`.

  GAP-01 makes every named unparsed tail a finding. That includes the tail every `EXEC SQL` statement leaves under the pinned producer (WINDOWS.md 6), which yields one low-severity finding per SQL statement.
- **D-11:** The GAP-05 placeholder is a **repository-scoped node**, one per (referencing repository, artifact type, normalized key). Artifact types are program, copybook, IDMS dictionary member, and `-INC` member.

  It carries:
  - the normalized key (`cobol-name-norm/1`);
  - the artifact type;
  - a "not obtained" status.

  It carries no source, body, range, or attributes. The as-written spelling stays on the referencing edges, as in P4 D-18.

  Each per-occurrence `EXTERNAL_UNRESOLVED` finding and its reference edge point to the placeholder. When the last reference disappears, the placeholder is removed, following the 04-14 shared-resource eviction pattern. It is owned by the referencing repository, so it is scope-safe (SEC-01) and covered by the audit (SEC-08).

  A placeholder must never satisfy a query that requires parsed implementation (`GRAPH-SCHEMA.md` Required Verification).

  — **Reversibility:** costly — this changes today's unprefixed, edge-only `unresolved::cobol::program|copybook::` targets into persisted, projected, owned nodes.
- **D-12:** Severity comes from the producer's static table (`cobol-handoff-v1`):
  - **high:** document-scope gaps, lost capture, unmapped capture;
  - **medium:** error, missing node, external reference;
  - **low:** unparsed tail.

  Gortex-derived `EXTERNAL_UNRESOLVED` findings (`missing` CALL/COPY, `idms_dictionary_member`) are **medium**, matching the producer's external-reference severity. A versioned severity rule ID is recorded in Meta.

#### Cascade regression gate (GAP-08)
- **D-13:** Cascades are detected from the producer's grade signals. Gortex does not re-derive them. A document whose grade reasons include the cascade family (`division-scale-recovery`, `retained-structure-collapse`, `missing-required-anchor`, `lost-capture`) gets one **high**-severity, document-scope `PARSE_UNRESOLVED` finding with reason `uncontained-cascade`, carrying the producer's reasons. Its localized gaps are still emitted individually (GAP-01).
- **D-14:** "Regression failure" means a committed fixture gate:
  - It uses invented fixtures and runs on the real path (registered extractor, indexer, SQLite temp store).
  - It runs in the default `go test -race`, with no flag.
  - Injected cascade shapes must produce the `uncontained-cascade` finding.
  - Contained shapes (negative controls) must keep their downstream paragraphs and data items.

  In a real estate, cascades are reported through findings and `index_health` counts. They never block indexing and never mark a repository unhealthy (the same posture as P4 D-15).
- **D-15:** Port the injection cases from `internal/parser/forest/cobolprobe/cascade_test.go` `TestErrorCascade` into the D-14 gate:
  - 20 data items plus paragraphs;
  - IDMS DML;
  - `EXEC CICS` shapes.

  Keep `cobolprobe` as a flag-gated measurement harness for real corpora. It is not the gate.
- **D-16:** A finding closes only through deterministic evidence: a source edit, a parser or extractor change that stops producing it, or retrieval or binding of the target. No API or tool may close or alter a finding in any other way. Phase 8 claims may cite findings but never change their status. There is no `closable_by` field, and nothing AI-specific is built in this phase.

### Claude's Discretion
- **Meta key names** for finding fields. Follow the P2/P3 conventions (`prov_*` for provenance, plus a `finding_*` namespace or similar). Avoid secret-like substrings, because the Neo4j projection filters them.
- **Edge kinds:**
  - finding to containing symbol (reuse `member_of` or similar existing kinds);
  - finding and reference edge to placeholder. Reuse existing kinds. Add `blocked_by` only if research shows a required query is obscured without it.
- **Placeholder node kind and ID grammar.** It must be repository-owned, distinguishable from parsed programs and copybooks, and never mistakable for an implementation. Choose how resolver binding and closure interact with it. The resolver currently rewrites edge `To` on bind, and the edge `To` today is the unprefixed `unresolved::cobol::` ID.
- **Evidence-digest details:**
  - text normalization before hashing;
  - the digest algorithm (prefer the existing BLAKE3 dependency);
  - what to digest when the original range is absent (for example node kind plus absence reason). Never invent coordinates.
- **Containing symbol:**
  - map the producer's `Container` observation index to a Gortex node ID through the existing walk (`cobolWalk.id`, `statementSymbol`);
  - attach document-scope gaps to the file node, or to the program node when exactly one exists.
- **Supersede pairing** when several old and new findings of one class and kind share a container: pair them in source order, then close or open the leftovers.
- **`finding_history`:** schema, migration (`storeNeedsRebuild` or an in-place migration), and indexes. Scope columns follow the `file_index_failures` precedent.
- **Where the extractor and resolver emit findings, and where lifecycle is diffed:**
  - parse findings come from `Extract`;
  - external findings and placeholders come from `reconcileCobol`;
  - lifecycle diffing must see the prior active finding set before eviction, on both the incremental and the clean path.
- **WR-06 representation:** how a copybook whose analysis failed is distinguished from a missing copybook. Expected: a `PARSE_UNRESOLVED` document finding on the present copybook, never an `EXTERNAL_UNRESOLVED`.
- **`index_health` shape** for the finding counts.
- **Extractor version:** bump `prov_extractor_version` past `gortex-cobol-grammar/5`, once for the phase, because the emitted nodes change.
- **Plan slicing and wave order.** Suggested order:
  1. finding kind plus parse findings (tracer);
  2. external findings plus placeholders;
  3. lifecycle and history;
  4. Phase 3 reason fixes;
  5. cascade gate plus incremental-equals-clean coverage.

### Deferred Ideas (OUT OF SCOPE)
- **A `list_findings` MCP/CLI verb** with class and file filters: Phase 6 (QUERY-08).
- **Finding history that survives store rebuilds:** revisit in Phase 8 if claim lineage needs it.
- **Impact-based severity** (for example, a missing called program as high): consider for Phase 6 ranking or monitoring.
- **`ambiguous` and `dynamic` references as findings:** not now. They stay edge reasons, distinguished by Phase 6 QUERY-03.
- **Promoting in-source misses** (`label_not_found`, `not_declared_*`) to `PARSE_UNRESOLVED` when they fall inside an Affected cascade region: possible later precision work.
- **An explicit `closable_by` field on findings:** Phase 8, if claim validation needs it.
- **Marking a repository unhealthy, or failing indexing, on an uncontained cascade:** rejected for now (D-14). Phases 6 and 7 may revisit for export or AI gating.
- **Findings for VB.NET and other languages:** Phase 6.1 or later, reusing the language-neutral `finding` kind.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GAP-01 | Named unparsed tail, localized `ERROR`, or localized `MISSING` with surviving context produces a persisted `PARSE_UNRESOLVED` finding | Producer `handoff.Gap` list is already on every handoff and unused by Gortex; probe shows exact shapes (Finding Emission pattern, Code Examples) |
| GAP-02 | A reference whose target was not retrieved produces an `EXTERNAL_UNRESOLVED` finding | `reconcileCobol` decides `missing` at `cobol_references.go:392`; IDMS edges carry `idms_dictionary_member`; producer `-INC` gap is NOT a usable signal (Pitfall 1) |
| GAP-03 | The two classes stay distinct in storage and query results | `prov_evidence_class` carries the class; class segment in the ID; separate creators (extractor vs resolver); index_health counts by class |
| GAP-04 | Stable ID, scope, exact range, containing symbol, class, reason, severity, parser version, lifecycle status | Finding Meta contract below; `cobolStampMeta` for range/absence; member_of edge for container |
| GAP-05 | Placeholder for the observed target without fabricated content | Placeholder node shape (KindResource, `synthetic`, `artifact_status=not_obtained`, no range) created/swept in the resolver |
| GAP-06 | Unchanged evidence reuses the finding identity | D-01 content-anchored ID with fixed-format normalization; canonical dump comparisons in the Phase 4 harness |
| GAP-07 | Close/supersede when source, parser, or retrieved-artifact evidence changes | Ledger diff over `finding_history` (open set) vs active nodes, at durable indexer points |
| GAP-08 | Uncontained cascade stays a deterministic regression failure | Producer grade reasons; probe-verified positive and negative fixture shapes on the real handoff path |
</phase_requirements>

## Project Constraints (from AGENTS.md / CLAUDE.md)

- Build and test: `go build -o gortex ./cmd/gortex/` (CGO required); `go test -race ./...` must pass. Phase 4 convention prefixes `GOWORK=off`.
- COBOL code and tests carry `//go:build darwin || linux` (Windows stub exists).
- Engineering standards (`CLAUDE.md`): single-use code stays inline; rule of three before extracting shared helpers; no speculative generality; target < 300 lines per module; reuse existing symbols before adding new ones; no new dependencies without stated justification.
- Conventions: exported symbols get doc comments starting with the name; `PURPOSE` / `RATIONALE` / `KEYWORDS` comment blocks on non-obvious functions (the COBOL files follow this).
- Fork hygiene (`.claude/CLAUDE.md`): prefer additive packages/files over invasive edits to upstream internals.
- Fixtures must be invented COBOL; the repository is public (P2 D-14, readiness §7).
- Git: conventional commits, lowercase, no Co-Authored-By trailers; never commit feature work to `main`/`master`. The orchestrator commits; this phase stays on `planning/ai-enhanced-cobol-graph`.
- GSD workflow enforcement: edits only through GSD execute flows.
- Agents working this repo use Gortex graph tools (MCP or `gortex` CLI) rather than raw grep/read on indexed source.

## Summary

The producer handoff already carries everything `PARSE_UNRESOLVED` needs. `handoff.Handoff.Gaps` is re-derived and validated by the pinned module, and each gap has `ResolutionClass`, `Kind`, `Reason`, `Severity`, `Scope`, `Generated`/`Original` spans, `NodeKind`, `ExpectedFamily`, `Container` (an observation index), and `AbsenceReason` [VERIFIED: preprocessor `handoff/gaps.go:93-107`]. Gortex consumes none of it today. Phase 5 is mostly mapping work in three existing seams (extractor `Extract`, resolver `reconcileCobol`, SQLite sidecar) plus one new lifecycle ledger. A throwaway probe against the pinned producer (`e64234c`) confirmed the concrete gap shapes, the cascade grade signals, and three facts that change the plan (see Pitfalls 1-3).

The three facts the planner must absorb: (1) Gortex builds the analyzer with an empty `-INC` catalog, so the producer emits an `external-reference` / `target-not-obtained` gap for every `-INC` member, even members Gortex later binds to a present copybook; consuming those gaps would fabricate "not obtained" evidence. `-INC` findings must come from the resolver's `missing` decision instead. (2) Unparsed tails are one gap per tail token node, not one per SQL statement (`INSERT ... SELECT A, A FROM T2` yields five). (3) On the cold-index shadow path the resolver runs against an in-memory shadow graph whose sidecars are not drained, so a lifecycle ledger written from inside the resolver would be lost in production while every Phase 4 harness test (which disables the shadow) passes.

The recommended architecture: parse findings are file-owned nodes emitted at the end of `Extract`; external findings and repository-owned placeholders are created, kept, and evicted inside `reconcileCobol` (one creator per class); lifecycle is a ledger diff (`finding_history` open set versus active finding nodes) run at the indexer level against the durable store after each settle point. That ledger design removes the "read the prior active set before eviction" plumbing entirely, because the prior active set is the ledger itself.

**Primary recommendation:** Build a language-neutral finding core in `internal/graph` (kind, ID/evidence helpers, ledger diff, two store capabilities), then add extractor parse findings, resolver external findings/placeholders, and an indexer-level ledger pass, with the D-14 cascade gate and incremental-equals-clean coverage on the existing `cobolLifecycleEnv` harness.

## Architectural Responsibility Map

| Capability | Primary Tier (package) | Secondary Tier | Rationale |
|------------|------------------------|----------------|-----------|
| `KindFinding`, finding ID/evidence digest, Meta key constants, severity rule ID, ledger diff (pure) | Graph core (`internal/graph`) | -- | Language-neutral; both extractor and resolver need the same ID function; graph is imported by both |
| `PARSE_UNRESOLVED` findings from producer gaps, cascade finding, WR-06 identity-only copybook | Extractor (`internal/parser/languages`) | Graph core | Only `Extract` holds the handoff and the observation-to-node walk |
| `EXTERNAL_UNRESOLVED` findings, placeholders, placeholder sweep, IN-08 reason | Resolver (`internal/resolver`, `reconcileCobol`) | Graph core | `missing` is decided only here, on full and incremental paths (P4 D-08) |
| WR-07 ambiguous cursor reason, IN-08 containment stamp | Extractor | Resolver (reads stamp) | Reasons originate in the walker; resolver only reads flags |
| `finding_history` table, v23 migration, sidecar registries, `EvictFindingNodesByIDs` | Store (`internal/graph/store_sqlite`) | In-memory `graph.Graph` | Persistence capabilities follow the `file_index_failures` and `EvictConfigNodesByIDs` precedents |
| Ledger pass (open/close/supersede rows) on the durable store; FTS sync for resolver-created nodes | Indexer orchestration (`internal/indexer`) | Resolver | Only the indexer knows when the durable graph is settled (shadow drain, incremental tails, warm restart) |
| Finding counts by class | MCP (`internal/mcp`, `index_health`) | CLI via `gortex call` | Existing health payload; must not change `health_score` (D-14) |
| `get_symbol` / search acceptance | MCP + CLI (`cmd/gortex`) | -- | P2 D-11 acceptance pattern |
| Finding/placeholder projection | Projection (`internal/neo4jprojection`) | -- | Automatic by kind; verify Meta survives the secret-key filter |
| Canonical comparison | Test utility (`internal/testutil/graphdump`) | -- | Active nodes are compared; history is not |

## Standard Stack

No new external dependencies. Everything needed is already in `go.mod`.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor` | `v0.0.0-20261002100409-e64234cc493c` (pinned) | Source of `handoff.Gap`, `grade.Reason*`, `parsefacts` | Already the only COBOL producer; pin enforced at `cobol_grammar.go` `cobolApprovedPreprocessorModule` [VERIFIED: go.mod:6, cobol_grammar.go:38] |
| `github.com/zeebo/blake3` | `v0.2.4` | Finding ID and evidence digests | Already used by `internal/indexer/cobol_material.go:39`, `merkle.go:169`, `graphdump` [VERIFIED: go.mod:280] |
| `modernc.org/sqlite` (via `store_sqlite`) | existing | `finding_history` sidecar | SQLite is sole authority (STORE-01) |
| Go `testing` + `testify` | existing | All tests | Project standard |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| BLAKE3 for IDs | SHA-256 (`neo4jprojection.projectionDigest` framing) | CONTEXT prefers BLAKE3; reuse the 8-byte big-endian length framing from `projectionDigest` (identity.go:53-61) for alias safety |
| Ledger diff over `finding_history` | Read prior file-owned findings before each eviction | Eviction happens in many store paths (file, files, repo, shadow drain); the ledger needs no hook in any of them |
| `KindResource` placeholder | New `placeholder` kind; `KindModule` (external_call precedent) | D-05 adds exactly one kind; `KindModule` is read as a dependency by SBOM/module passes; `KindResource` is already the COBOL "named by statements, not parsed" kind |

**Installation:** none.

## Package Legitimacy Audit

No packages are installed by this phase. `zeebo/blake3 v0.2.4` and the pinned preprocessor are existing `go.mod` requirements [VERIFIED: go.mod:6, go.mod:280].

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
 COBOL source (.cbl/.cob/.cpy)
        |
        v
 CobolGrammarExtractor.Extract ---- Analyze / AnalyzeCopybook (pinned producer)
        |                                   |  handoff: Observations, Gaps[], Assessment
        |   breadth walk (programs, procedure, data, refs, mainframe, SQL)
        |   NEW cobolFindings(w): Gaps[] (PARSE only) -> finding nodes + member_of
        |   NEW cascade finding when Assessment.Reasons hits the cascade family
        |   NEW WR-06: Analyze error on .cpy -> identity-only file node + analysis-failed finding
        v
 Indexer: applyRepoPrefix -> AddBatch (file generation; hard evict by file_path)
        |
        v
 Resolver.reconcileCobol(repos)  (full: ResolveAll; incremental: ReconcileCobol)
        |  decide each literal CALL/LINK/XCTL/COPY/-INC/SQL INCLUDE edge + COPY IDMS edge
        |    bind ------------> To = candidate          (no finding)
        |    ambiguous/dynamic/caller_unowned/containment_unresolved -> To = unresolved:: (no finding)
        |    missing / idms --> To = <repo>/cobol-missing::<type>/<key> placeholder
        |                       + EXTERNAL finding (owned by edge.FilePath) -member_of-> edge.From
        |                       + finding -references-> placeholder
        |  evict undesired EXTERNAL findings (EvictFindingNodesByIDs)
        |  sweep placeholders with no code in-edges (EvictFile on own virtual path)
        v
 Indexer settle point (durable store only: after shadow drain / incremental tail / warm restart)
        |  NEW ledger pass: open set from finding_history vs active finding nodes
        |     new ID -> row(active); gone ID -> row(superseded w/ successor | closed)
        |  NEW FTS upsert/delete for resolver-created nodes (incremental tails)
        v
 SQLite: nodes/edges (canonical) + finding_history (non-canonical, view_gen-keyed)
        |
        +--> get_symbol / search_symbols / index_health (MCP + CLI)
        +--> neo4j_push projection (label `finding`, placeholders GortexUnresolved)
```

### Recommended File Layout (additive, one responsibility per file)
```
internal/graph/
  finding.go                  # KindFinding doc, Meta key consts, FindingID, evidence normalize, severity rule
  finding_ledger.go           # pure diff: open records + active nodes -> transitions; FindingHistory* interfaces
  finding_node_evict.go       # FindingNodeBatchEvicter + in-memory impl (mirror config_node_evict.go)
internal/graph/store_sqlite/
  store_finding_history.go    # table body, create fn, append/open-set/read
  finding_node_evict.go       # EvictFindingNodesByIDs (mirror config_node_evict.go)
internal/parser/languages/
  cobol_grammar_findings.go   # cobolFindings(w), container ladder, cascade, WR-06 node
internal/resolver/
  cobol_findings.go           # placeholder + external finding desired-set, evict, sweep
internal/indexer/
  finding_ledger.go           # durable ledger pass + FTS sync for resolver-created nodes
```

### Pattern 1: Finding identity (D-01)
**What:** `ID = <source path>::@finding/<parse|external>/<hex32>[#k]`, repo-relative in the extractor (prefixed later by `applyRepoPrefix`), built prefixed in the resolver from `edge.FilePath`.
**Hash input (length-delimited, BLAKE3-256 truncated to 16 bytes):** `"gortex-finding-id/1"`, container ID with the repository prefix stripped, resolution class, gap kind, node kind or expected family, evidence digest.
**Ordinal:** group by the unsuffixed ID; order by generated start then end (extractor) or `prov_start_byte` then `Line` (resolver); `#k` only for k >= 2 (matches the COBOL `#k` convention, `cobol_grammar_identity.go:45`).
**Why `@finding`:** `@` cannot begin a COBOL word (same rationale as `cobolCopybookRoot = "@proc"`, `cobol_grammar_copybook.go:15-21`), so no written name can collide.
**Why a class segment:** makes GAP-03 visible in the ID itself and lets counters avoid Meta decode.

### Pattern 2: Evidence digest (D-01, discretion)
- Evidence bytes: `src[Original.Start.Byte:Original.End.Byte]` when `Original != nil && !NoImage && !Synthetic`; else `h.Generated[Generated.Start:Generated.End]` when `Generated != nil`; else empty.
- Normalize: per physical line, drop bytes at 0-based byte columns < 6 (sequence area) and >= 72 (identification area); collapse runs of space/tab/CR/LF to one space; trim. Do not case-fold.
- Rationale: the producer treats input as fixed-format with a column-73 zone boundary [VERIFIED: preprocessor `transform/fixed.go:1-3,26-28`]. ISPF renumbering rewrites columns 1-6 file-wide and change tools stamp 73-80, which would otherwise churn every multi-line ERROR finding (probe ERROR spans reach 20+ lines).
- Empty evidence (zero-width MISSING `.`, document-scope gaps): the digest still differs by kind and node kind; for document scope hash `kind + AbsenceReason` (producer gaps) or the constant reason (`uncontained-cascade`, `analysis-failed`). Never include producer reasons in the ID (they are re-stamped Meta), so a persisting cascade keeps its ID.
- Record `finding_evidence_rule = "gortex-finding-evidence/1"` so a future rule change is visible.

### Pattern 3: Finding node Meta contract (GAP-04)
Clone the document provenance the walker already uses (`w.docMeta`), then set:

| Field (GAP-04) | Meta key | Value |
|---|---|---|
| resolution class | `prov_evidence_class` | `PARSE_UNRESOLVED` or `EXTERNAL_UNRESOLVED` (overrides `DETERMINISTIC`, `cobol_grammar.go:236`; key is stamped but never read today) |
| kind | `finding_kind` | producer `Kind` verbatim, or `uncontained-cascade`, `analysis-failed`, `external-reference` |
| normalized reason | `finding_reason` | producer `Reason` verbatim; edge reason verbatim for external (`missing`, `idms_dictionary_member`) |
| severity | `finding_severity`, `finding_severity_rule` | producer `Severity` verbatim; Gortex additions per D-12; rule `gortex-finding-severity/1` |
| scope | `finding_scope` | producer `Scope` (`localized`/`document`/`unavailable`); `document` for Gortex-derived document findings |
| parser node type / family | `finding_node_kind`, `finding_expected_family` | producer `NodeKind`, `ExpectedFamily`; artifact type for external |
| containing symbol | `finding_container` + `member_of` edge | repo-relative ID in Meta (same convention as `cobol_container`, `cobol_grammar.go:228-230`); prefixed endpoint on the edge |
| exact range | `prov_start_row`..`prov_end_byte`, `prov_range_exact` | via `cobolStampMeta` (`cobol_grammar_walk.go:268`) |
| absent range | `prov_range_exact=false`, `prov_range_absence` | producer `AbsenceReason` verbatim when set; otherwise `cobolStampMeta`'s `no_original_projection` / `synthetic_projection`; `document_scope` for Gortex-derived |
| parser version | existing `prov_parser_tool_id`, `prov_parser_grammar_id`, `prov_extractor_version` | from docMeta (parse) or the referencing edge Meta (external) |
| lifecycle status | `finding_status` | constant `active` (closed/superseded live only in history) |
| evidence | `finding_evidence_digest`, `finding_evidence_rule` | hex digest, rule ID |
| cascade reasons | `finding_producer_reasons` | `[]string` copy of `h.Assessment.Reasons` (cascade finding only) |
| external target | `finding_artifact_type`, `finding_target_key` | `program`/`copybook`/`inc_member`/`idms_dictionary_member`; normalized key from the placeholder target |

Allowed Meta value types are `string, bool, int, float64, []string` (`assertCobolMetaContract`, `cobol_grammar_contract_test.go:99-113`). Keys must not contain `password, passwd, secret, token, credential, privatekey, apikey, authorization` after normalization [VERIFIED: cobol_grammar_contract_test.go:96; neo4jprojection/properties.go:174]. Do not store raw gap source text in Meta (digest + range only); the range lets scoped tools read source under the 04-17 source gate.

### Pattern 4: Container ladder (discretion)
Map `Gap.Container` (observation index, or -1) in this order, walking `Parent` links when a level is empty:
1. innermost statement node (`w.statementSymbol(i)`, `cobol_grammar_walk.go:140`);
2. procedure membership container (`w.container[i]`: paragraph, else section, else procedure division);
3. a data/environment section node in `w.sections`;
4. owning program (`w.programSymbol(i)`), else the walk root (copybook file node via `w.id`);
5. `Container == -1` or a `start` container: the program node when exactly one named program exists, else the file node.
Probe containers seen: `sql_select_query`, `sql_insert_statement`, `program_definition`, `start`, `identification_division`, `procedure_division`, `WORD`, `level_number` [VERIFIED: probe this session]. All resolve through the ladder.

### Pattern 5: Placeholder (GAP-05, discretion)
- Kind `graph.KindResource`; `ID == FilePath == <callerRepo>/cobol-missing::<type>/<key>`; `RepoPrefix`, `WorkspaceID`, `ProjectID` copied from the caller node (`r.cachedGetNode(e.From)`, resolver.go:5757-5777); `Language "cobol"`; `Name = <key>`.
- Meta: `cobol_kind="missing_artifact"`, `artifact_type`, `artifact_key`, `artifact_status="not_obtained"`, `synthetic=true`, `prov_evidence_class="EXTERNAL_UNRESOLVED"`, `prov_range_exact=false`, `prov_range_absence="not_obtained"`, `cobol_name_norm="cobol-name-norm/1"`. No parser stamps, no as-written spelling (writer-independent, as 04-01 learned for shared resources).
- Never a candidate: the candidate index reads only `KindFunction` (`cobol_kind=program`) and `KindFile` (`cobol_kind=copybook`) (`cobol_references.go:127-128`).
- `synthetic: true` gives the Neo4j `GortexUnresolved` label (`neo4jprojection/properties.go:54`), the same marker the resolver's own external-call placeholders use (`resolver/external_calls.go:417-433`).
- Type mapping from the extractor's target: `unresolved::cobol::program::K` -> `program/K`; `...::copybook::K` with `cobol_copy_form=inc` -> `inc_member/K`, otherwise `copybook/K`; `...::copybook::idms/<kind>/<K>` -> `idms_dictionary_member/<kind>/<K>`. Pure string mapping; the resolver must not normalize names itself (D-18).
- Sweep: a placeholder with no in-edge other than from a finding is evicted via `EvictFile(own path)`, exactly like `reconcileCobolResources` (cobol_resources.go:38-79). Evict undesired findings before sweeping.

### Pattern 6: Ledger lifecycle (GAP-06, GAP-07, D-03, D-06, D-07)
- `finding_history` rows record every transition, including opens (`'' -> active`). The current open set is "latest row per finding is `active`". This is a superset of D-06 (which requires close/supersede rows) and is what lets the diff run without pre-eviction reads.
- Diff per repository in scope: active IDs not open -> append `active`; open IDs not active -> superseded when an unpaired newly-opened finding has the same (container, class, kind), pairing old and new in source order (`start_byte`, then ID); leftovers -> `closed`. Never pair across containers or classes.
- Identical rerun writes zero rows (assert this; it is the cheapest spurious-history detector).
- Parser re-stamp with the same ID (D-02) writes no row.
- Restore after delete reopens the same ID (new `active` row after a `closed` row).
- Runs on the durable store only (Pitfall 3).

### Anti-Patterns to Avoid
- **Consuming the producer's `external-reference` gaps:** every `-INC` gets one with an empty catalog (Pitfall 1).
- **Writing history from inside `reconcileCobol`:** lost on the shadow path (Pitfall 3).
- **Emitting findings before docMeta document flags are frozen correctly:** see Pitfall 5.
- **Using `cobol_name_key` / `cobol_kind=copybook|program` on placeholders or findings:** risks future candidate confusion.
- **Adding finding_history to the repo-predicate eviction:** `EvictRepo` clears `file_index_failures` on reindex (file_batch_evict.go:113-118); history must survive reindex (D-07).
- **Filtering or merging producer gaps:** GAP-01/D-10 say every gap is a finding; grouping tail tokens would be invention.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Range / absence stamping | New range code | `cobolStampMeta`, `cobolStampRange` | Already encodes no-image, synthetic, exact flags (P2 D-09) |
| Cascade detection | Re-derive from gaps | `h.Assessment.Reasons` (`grade.Reason*`) | D-13; producer is authoritative |
| Severity | Gortex table for producer kinds | `Gap.Severity` (producer `severityTable`, cobol-handoff-v1) | D-12 |
| Bounded node removal by ID | Generic delete | Mirror `EvictConfigNodesByIDs` (store_sqlite/config_node_evict.go:16, graph/config_node_evict.go:6-12) as `EvictFindingNodesByIDs` | Same transaction shape, both backends |
| Sidecar table plumbing | Ad-hoc table | `file_index_failures` precedent: body const, `viewGenSidecars`, v-numbered in-place migration, `purgeSidecarTables`, `orphanScanTables`, `rekeyDropTables` | Registry tests enforce consistency |
| Orphan placeholder GC | New sweeper | Extend the `reconcileCobolResources` pattern | FilePath==ID eviction already proven (04-14) |
| Canonical comparison | New diff tool | `graphdump.Dump` + `requireSameDump` | Phase 4 harness |
| FTS tokens | New tokenizer | `ftsTokensFor`, `shouldIndexForSearch`, `BatchUpsertSymbolFTS`/`BatchDeleteSymbolFTS` | Same corpus as every other writer |

**Key insight:** every storage, eviction, provenance, and comparison primitive this phase needs already exists with a COBOL or sidecar precedent; the new code is mapping logic plus one pure diff.

## Runtime State Inventory

This phase migrates persisted shape (edge targets, schema, node set), so the inventory applies.

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | Existing SQLite stores hold literal CALL/COPY edges with `To = unresolved::cobol::...` and reason `missing`, and COPY IDMS edges on `unresolved::cobol::copybook::idms/...` | No data migration task: the extractor version bump (`/5` -> `/6`) changes the COBOL material fingerprint, so every COBOL file re-extracts, and `ReconcileRepoCtx` reconciles COBOL unconditionally at warm restart (04-15), which rewrites `missing` edges to placeholders. Verify with a warm-restart test. |
| Stored data | No `finding_history` table in v22 stores | v23 in-place additive migration (precedent v21 `createFileIndexFailuresTable`, schema_version.go:120). History starts empty (D-07). |
| Live service config | Neo4j projections made before Phase 5 contain synthetic `unresolved` nodes for missing COBOL targets | Manual re-run of the explicit projection; Phase 1 stale-record reconciliation removes old records (NEO-14). Document only. |
| OS-registered state | None -- verified: no launchd/systemd/scheduler artifacts reference COBOL targets | None |
| Secrets/env vars | None -- no new config keys or env vars | None |
| Build artifacts | Running daemons and installed `gortex` binaries carry the `/5` mapping | Rebuild binary and restart daemon; the material fingerprint triggers re-extraction automatically |

## Common Pitfalls

### Pitfall 1: The producer's `-INC` gap fabricates "not obtained"
**What goes wrong:** `cobol_grammar.go` builds the analyzer with `transform.NewCatalog(nil)`; the Endevor rule appends `ReasonIncMemberUnresolved` whenever `!catalog.Resolve(member)` [VERIFIED: preprocessor `transform/endevor.go:221-228`]. Probe: `-INC CUSTREC` yields `EXTERNAL_UNRESOLVED kind=external-reference reason=target-not-obtained target="CUSTREC" container=WORD` on a green document [VERIFIED: probe]. Meanwhile Gortex emits an `imports` edge with `cobol_copy_form=inc` that the resolver may bind to a present copybook (`cobol_grammar_refs.go:136-138,148,177`).
**How to avoid:** do not map producer `external-reference` gaps to findings. `-INC` findings come only from the resolver when the `inc` edge is `missing`; `cyclic` cannot fire with an empty catalog (`Catalog.Cyclic` needs elements).
**Warning signs:** an `EXTERNAL_UNRESOLVED` finding on a file whose `-INC` edge is bound.
**Note:** this narrows D-09's third bullet; see Assumptions A1.

### Pitfall 2: Tail volume is per token
**What goes wrong:** `SELECT ID FROM APP.CUSTOMER` leaves one tail (`ID`); `INSERT INTO T1 SELECT A FROM T2` leaves four (`SELECT`, `A`, `FROM`, `T2`); `SELECT A, A` leaves two identical `A` tails in one container [VERIFIED: probe]. CONTEXT expected one per statement.
**How to avoid:** emit one finding per gap (GAP-01), rely on `#k` for duplicate text in a container, and size tests and `index_health` accordingly. See Assumptions A2.

### Pitfall 3: Shadow path drops resolver-time sidecar writes
**What goes wrong:** during a cold index the indexer points the resolver at an in-memory shadow (`idx.resolver.SetGraph(inMemShadow)`, indexer.go:2909); only explicitly transferred sidecars reach disk (`persistShadowCompactSidecars`, sidecar_batch.go:102-123). `newCobolLifecycleEnv` sets `GORTEX_SHADOW_MAX_FILES=0`, so every Phase 4 harness test bypasses the shadow.
**How to avoid:** placeholders and external findings may be created on the shadow (nodes drain), but the ledger pass and FTS sync run at the indexer level on the durable store. Add one test with the shadow enabled that asserts `finding_history` rows exist in SQLite after a track.

### Pitfall 4: Resolver-created nodes are not searchable after incremental changes
**What goes wrong:** symbol FTS is written by the indexer (`upsertIncrementalFTS`, incremental_batch.go:1037; `populateSymbolFTS`, indexer.go:797), not by `AddBatch`. A full non-shadow index repopulates FTS from all store nodes after resolution, but incremental reconciles do not, and `EvictFindingNodesByIDs` would leave stale FTS rows.
**How to avoid:** have `reconcileCobol` report created and evicted finding/placeholder IDs; the indexer COBOL tails (`reconcileCobolAfterMutation`, `reconcileCobolRepos`, cobol_reconcile.go:55-83) upsert/delete their FTS rows with `ftsTokensFor`. Test search by name for an external finding/placeholder after an incremental edit.

### Pitfall 5: docMeta mutation order in `Extract`
**What goes wrong:** for programs, `w.docMeta` is the file node's map; `prov_containment_unresolved` and `prov_unnamed_program_count` are set after the walk (cobol_grammar.go:292-295) and would leak into cloned finding Meta if findings were built later. For copybooks, `cobolCopybook` sets `cobol_kind=copybook`, `cobol_name_key` on `docMeta` after the walk (copybook.go:72-74), and its `copybook_content_unmapped` check is `len(result.Nodes) == 1 && len(result.Edges) == 0` (copybook.go:83), which finding nodes would break.
**How to avoid:** call `cobolFindings(w)` right after `cobolSQL` and before the document flags in `Extract`; in `cobolCopybook` call it with the cloned walk docMeta and after the unmapped check (or count only non-finding nodes). Add a regression test for both.

### Pitfall 6: `missing` decided for an artifact that exists
**What goes wrong:** IN-08: under `containmentUnresolved`, children keep `nested: true` (cobol_grammar_identity.go:105-146) and the resolver drops nested candidates from other files (cobol_references.go:350-357), returning `missing`. Legacy nodes without `cobol_nested` also fail closed to `missing`. WR-06: a copybook analysis error returns `nil, err` (cobol_grammar.go:215), so every COPY of it is `missing`.
**How to avoid:** stamp the affected programs (for example `cobol_nested_unresolved=true`) and return a distinct reason (for example `containment_unresolved`) with no finding; emit an identity-only copybook node plus an `analysis-failed` document finding. Without these, Phase 5 turns false reasons into false `EXTERNAL_UNRESOLVED` findings.

### Pitfall 7: SQLCA/SQLDA include noise
**What goes wrong:** `EXEC SQL INCLUDE SQLCA` follows the copybook path with `cobol_placeholder` and `sql_include_builtin=true` (cobol_grammar_refs.go:185-188); with no SQLCA member it is `missing`, so every DB2 program would get a "not obtained" finding and SQLCA would top Phase 6's blocker ranking.
**How to avoid:** skip findings and placeholders for `sql_include_builtin=true` edges (edge keeps its reason). See Assumptions A3.

### Pitfall 8: `graphHasLanguage("cobol")` early return
**What goes wrong:** `reconcileCobol` returns immediately when the graph holds no COBOL (cobol_references.go:125). Deleting a repository's last COBOL file must still close its findings in the ledger and sweep its placeholders.
**How to avoid:** the indexer-level ledger pass is not gated on COBOL presence (it compares the open set for the repo even when no finding nodes remain).

### Pitfall 9: Tests that will change
- ~50 assertions in resolver/indexer/cmd tests check `missing` edges on `unresolved::cobol::...` (counted: `cobol_references_test.go` 18, `cobol_reconcile_test.go` 14, `cobol_lifecycle_crossrepo_test.go` 10, `cobol_lifecycle_incremental_test.go` 7, `cobol_breadth_test.go` 4, `cobol_lifecycle_material_test.go` 3). Extractor tests keep `unresolved::` targets.
- `gortex-cobol-grammar/5` is asserted in `cobol_grammar_contract_test.go` (2), `cobol_tracer_test.go` (3), `cobol_breadth_test.go` (2).
- Node-count assertions: `cobol_grammar_copybook_test.go:280,627`, `cobol_grammar_test.go:521`.
- `schema_sidecar_view_gen_test.go:285-290,425-431` special-cases `file_index_failures` as "introduced at v21"; `finding_history` needs the same.

### Pitfall 10: Generic "unresolved" metrics undercount COBOL
**What goes wrong:** once `missing` edges point at placeholder nodes they no longer appear in `EdgesWithUnresolvedTarget` or `IsUnresolvedTarget`-based counts. `reconcileCobol` itself must also visit in-edges of placeholder nodes, or a placeholder-bound edge is never re-decided when its target arrives.
**How to avoid:** add placeholder IDs to the in-edge visit in `reconcileCobol` (beside `candidateIDs`, cobol_references.go:196-215); document that COBOL unresolved counts come from findings and edge reasons.

## Code Examples

### Producer gap and severity vocabulary (verbatim)
```go
// Source: preprocessor@e64234c handoff/gaps.go:34-53, 64-68  [VERIFIED: read this session]
ResolutionParseUnresolved    ResolutionClass = "PARSE_UNRESOLVED"
ResolutionExternalUnresolved ResolutionClass = "EXTERNAL_UNRESOLVED"
KindBudgetExceeded    Kind = "budget-exceeded"
KindNoTree            Kind = "no-tree"
KindInvocationFailure Kind = "invocation-failure"
KindError             Kind = "error"
KindMissing           Kind = "missing"
KindUnparsedTail      Kind = "unparsed-tail"
KindLostCapture       Kind = "lost-capture"
KindUnmappedCapture   Kind = "unmapped-capture"
KindExternalReference Kind = "external-reference"
SeverityHigh   Severity = "high"
SeverityMedium Severity = "medium"
SeverityLow    Severity = "low"
ScopeLocalized   Scope = "localized"
ScopeDocument    Scope = "document"
ScopeUnavailable Scope = "unavailable"

var severityTable = map[Kind]Severity{
	KindBudgetExceeded: SeverityHigh, KindNoTree: SeverityHigh, KindInvocationFailure: SeverityHigh,
	KindError: SeverityMedium, KindMissing: SeverityMedium, KindExternalReference: SeverityMedium,
	KindUnparsedTail: SeverityLow, KindLostCapture: SeverityHigh, KindUnmappedCapture: SeverityHigh,
}
```
`Gap` fields: `ID, ResolutionClass, Kind, Reason, Severity, Scope, Generated *sourcemap.Span, Original *sourcemap.Projection, NodeKind, ExpectedFamily, Container int, Target, AbsenceReason` [VERIFIED: gaps.go:93-107]. Document gaps use `Container: -1` and `AbsenceReason: kr.reason` (gaps.go:117).

### Cascade family reasons (verbatim)
```go
// Source: preprocessor@e64234c grade/grade.go:108-122  [VERIFIED]
ReasonParseTimeout              = "parse-timeout"
ReasonNoTree                    = "no-tree"
ReasonInvocationFailure         = "invocation-failure"
ReasonInvalidProvenance         = "invalid-provenance"
ReasonUnmappedCapture           = "unmapped-capture"
ReasonMissingRequiredAnchor     = "missing-required-anchor"
ReasonLostCapture               = "lost-capture"
ReasonDivisionScaleRecovery     = "division-scale-recovery"
ReasonRetainedStructureCollapse = "retained-structure-collapse"
ReasonErrorNode                 = "error-node"
ReasonMissingNode               = "missing-node"
ReasonUnparsedTail              = "unparsed-tail"
ReasonAffectedCapture           = "affected-capture"
```
D-13 family = `division-scale-recovery`, `retained-structure-collapse`, `missing-required-anchor`, `lost-capture`. Use the `grade` constants, not string literals.

### Extractor finding emission (skeleton, names are recommendations)
```go
// internal/parser/languages/cobol_grammar_findings.go (new)
// Called in Extract right after cobolSQL(w, data), before the document flags.
func cobolFindings(w *cobolWalk) {
	for _, g := range w.h.Gaps {
		if g.ResolutionClass != handoff.ResolutionParseUnresolved {
			continue // Pitfall 1: never consume producer external-reference gaps
		}
		// container ladder (Pattern 4) -> container ID (repo-relative)
		// evidence bytes + normalize (Pattern 2) -> digest
		// graph.FindingID(...) + ordinal (Pattern 1)
		// meta := maps.Clone(w.docMeta); set Pattern 3 keys; cobolStampMeta or absence
		// append node (Kind graph.KindFinding) + w.memberOf(findingID, containerID, line)
	}
	// D-13: one document finding when w.h.Assessment.Reasons intersects the cascade family
}
```

### Placeholder bound in the resolver (skeleton)
```go
// internal/resolver/cobol_findings.go (new); called from reconcileCobol
// d := r.decideCobol(...)  // unchanged decision
// if d.reason == "missing" (and !sql_include_builtin) -> to = placeholderID(callerRepo, e)
// ensure placeholder node; desired external finding keyed by FindingID(edge.From, ...)
// after the edge loop: EvictFindingNodesByIDs(undesired); AddBatch(new nodes, new edges);
// then sweep placeholders whose only in-edges come from findings.
```

### Probe-verified fixture shapes (invented COBOL; real handoff path)
| Shape (injected into the 20-data/20-paragraph control) | Grade / reasons | Gaps | Gate role |
|---|---|---|---|
| clean control | green, [] | 0 | baseline 20/20 |
| `EXEC CICS SEND MAP('M') END-EXEC.` after para 1 (and RETURN, SEND ERASE, multi-line) | green | 0 | negative (contained) |
| `OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.` after para 1 | green | 0 | negative |
| `EXEC SQL SELECT ID FROM APP.CUSTOMER END-EXEC.` after para 1 | amber [unparsed-tail] | 1 tail | negative for cascade, positive GAP-01 |
| `SCHEMA SECTION.` + `DB DCCSS000 WITHIN CAMSC000.` before data | green | 0 | negative (the transform handles it; the raw-grammar test logged 0/20) |
| `EXEC SQL INCLUDE SQLCA END-EXEC.` in WORKING-STORAGE | green | 0 | negative |
| `COMPUTE WS-B = (1 + 2.` | amber [error-node affected-capture] | 1 ERROR under `procedure_division` | localized ERROR, no cascade |
| `PROGRAM-ID. T` (no period) | amber [missing-node] | 1 MISSING `.` zero-width | localized MISSING |
| `OBTAIN ...` in WORKING-STORAGE | red [lost-capture division-scale-recovery error-node affected-capture] | 1 ERROR + 19 lost `data-item` | positive cascade (data 1/20) |
| `EXEC SQL DECLARE C1 CURSOR FOR SELECT A FROM T1 END-EXEC.` in WORKING-STORAGE (WINDOWS 4) | red [same] | lost `exec-sql` + ERROR + 19 lost | positive cascade |
| `EXEC SQL DECLARE C2 CURSOR WITH HOLD FOR ...` in procedure | red [same] | lost `exec-sql` + ERROR (inexact original) | positive cascade (paras 1/20) |
| `FROBNICATE THE WIDGET NOW.` in procedure | red [same] | ERROR + 19 lost `paragraph` | positive cascade |
| `01  BROKEN PIC X(10) GARBAGE CLAUSE HERE.` | red [same] | ERROR + 19 lost | positive cascade |
| `DATA DIVISION` missing period | red [missing-required-anchor ...] | ERROR + lost anchors | positive cascade |
| copybook `05 CUST-ID PIC X(8) WHATEVER.` | red [lost-capture division-scale-recovery ...] | ERROR + 1 lost | positive (copybook policy) |
| `-INC CUSTREC` | green | 1 EXTERNAL `target-not-obtained` | Pitfall 1 regression fixture |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Missing targets are edge-only `unresolved::cobol::...` IDs | Repository-owned placeholder nodes plus per-occurrence findings | Phase 5 (D-11) | Edge `To` changes for `missing` and COPY IDMS; resolver must visit placeholder in-edges |
| Producer `gapID` (document content ID + offsets) | Content-anchored Gortex finding ID (D-01) | Phase 5 | Line shifts and unrelated edits keep IDs |
| Raw-grammar `TestErrorCascade` behind `-enhanced-parser` | Real-path gate in default `go test -race` | Phase 5 (D-14/D-15) | `SCHEMA SECTION` and EXEC SQL INCLUDE are contained through the transform |

**Deprecated/outdated:**
- WR-09 is already fixed (`cursor_not_in_copybook`, cobol_grammar_sql.go:155-162; test cobol_grammar_copybook_test.go:505, commit 9f6c1973). Only verification remains.
- WR-07 is still open: `if len(decls) != 1` stamps `cursor_not_declared` for two or more DECLAREs (cobol_grammar_sql.go:155-157). The reason lands on the statement node Meta, not an edge.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Producer `external-reference` gaps are not converted to findings; `-INC` findings come from the resolver `missing` decision (narrows D-09 bullet 3). **CONFIRMED by user 2026-10-19 -> CONTEXT D-17.** | Pitfall 1 | If kept literally, every `-INC` member gets a false "not obtained" finding |
| A2 | One finding per unparsed-tail gap (per token), not per SQL statement. **CONFIRMED by user 2026-10-19 -> CONTEXT D-18.** | Pitfall 2 | If the user wants per-statement grouping, that is Gortex-invented aggregation and needs a decision |
| A3 | `sql_include_builtin=true` (SQLCA/SQLDA) gets no external finding or placeholder. **CONFIRMED by user 2026-10-19 -> CONTEXT D-19.** | Pitfall 7 | If kept, SQLCA dominates missing-artifact counts and rankings |
| A4 | Placeholder reuses `KindResource` with `synthetic=true` and `cobol_kind=missing_artifact`, ID `<repo>/cobol-missing::<type>/<key>` | Pattern 5 | Phase 6 QUERY-09 must filter resources by Meta; a later dedicated kind is a costly change |
| A5 | `finding_history` records opens as well as closes/supersedes (ledger design) | Pattern 6 | If only close/supersede rows are allowed, the diff needs pre-eviction reads in every eviction path |
| A6 | Evidence normalization drops byte columns 1-6 and 73+ and collapses whitespace (fixed-format assumption) | Pattern 2 | Free-format sources would lose code text from the digest (only causes extra ordinals, not collisions) |
| A7 | WR-06 fallback applies to copybooks only; program analysis errors keep today's hard failure + `file_index_failures`. **CONFIRMED by user 2026-10-19 as an accepted, documented limitation -> CONTEXT D-20.** | Pitfall 6 | A program that fails analysis still yields `missing` for its callers |
| A8 | Lost-capture evidence is the anchor span text (literal D-01), accepting ordinal-based lineage among identical anchors such as `01` | Pattern 2 | History may attribute a closure to the last `#k` rather than the removed occurrence |

## Open Questions (RESOLVED)

1. **Where does lifecycle diffing run?**
   - RESOLVED: indexer-level ledger pass on the durable store at the end of `reconcileCobolAfterMutation`, `MultiIndexer.reconcileCobolRepos` (covers incremental, batch end, warm restart, config change, untrack), and after `TrackRepoCtx` completes indexing and resolution (the track path has no indexer-level COBOL tail today; add one). Not inside `reconcileCobol` (Pitfall 3).
2. **How does the resolver find edges already moved to a placeholder?**
   - RESOLVED: include placeholder node IDs in the existing in-edge visit (`GetInEdgesByNodeIDs`); literal edges carry `cobol_placeholder`; IDMS edges are recognized by `cobol_copy_form=idms` and their key is recovered from the placeholder ID, without re-normalizing names.
3. **Does `get_symbol` serve a node whose FilePath is virtual?**
   - RESOLVED: yes. `handleGetSymbol` reads the node and its edges only; `ensureFresh` on the `::`-split prefix is a no-op for a non-file path (tools_core.go:1709-1754). Still add MCP and CLI tests for both a finding and a placeholder.
4. **How are findings counted in `index_health` without decoding every Meta?**
   - RESOLVED: iterate `NodesByKind(graph.KindFinding)` in the background payload builder (same cancel stride as the file loop) and count by the class segment of the ID or `prov_evidence_class`; report `{"PARSE_UNRESOLVED": n, "EXTERNAL_UNRESOLVED": m}` scoped to the payload's repositories; never feed `health_score` or `recommendation` (D-14).
5. **Edge kinds?**
   - RESOLVED: `member_of` (finding -> container, structural, no Meta) and `references` (finding -> placeholder). `blocked_by` is not needed: QUERY-09 counts code in-edges (`calls`/`imports`) to placeholders, which edge kind already separates from finding edges.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build/tests | yes | go1.27.1 darwin/arm64 (go.mod `go 1.27.0`) | -- |
| CGO C toolchain | tree-sitter / COBOL tests | yes (COBOL tests built and ran this session) | -- | -- |
| Pinned preprocessor + forest-shim in module cache | COBOL extraction | yes | e64234c / f9eaf99 | -- |
| SQLite (pure Go) | store tests | yes | modernc v1.54.0 | -- |
| Neo4j server | not required | n/a | -- | Projection tests use property-level unit tests |

Measured: `TestCobolLifecycle_IdenticalRerun` 10.6s; `./internal/resolver -run Cobol` 5.5s; `./internal/parser/languages -run CobolGrammar` 26.1s (all `-race`, this session).

**Missing dependencies with no fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + testify v1.11.1 |
| Config file | none; COBOL tests use `//go:build darwin \|\| linux` |
| Quick run command | `GOWORK=off go test -race -count=1 ./internal/graph/... ./internal/resolver ./internal/parser/languages -run 'Finding\|Cobol'` |
| Full suite command | `GOWORK=off go test -race ./...` |
| Estimated runtime | quick ~60-90s; full suite several minutes |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GAP-01 | tail, ERROR, MISSING, lost/unmapped capture, document gaps become findings with container and range | unit (real producer) | `GOWORK=off go test -race -count=1 ./internal/parser/languages -run 'TestCobolGrammar_Findings'` | No, Wave 0 |
| GAP-01 | producer `-INC` gap never becomes a finding | unit | `... ./internal/parser/languages -run 'TestCobolGrammar_Findings_NoProducerExternal'` | No |
| GAP-02 | missing CALL/LINK/XCTL/COPY/-INC and COPY IDMS yield external findings; ambiguous/dynamic/bound/builtin do not | unit | `GOWORK=off go test -race -count=1 ./internal/resolver -run 'TestCobolReconcile_(External\|Placeholder)'` | No |
| GAP-03 | classes distinct in Meta, ID, and `index_health` counts | unit + MCP | `... ./internal/graph -run TestFindingID ; ... ./internal/mcp -run TestIndexHealth_FindingCounts` | No |
| GAP-04 | Meta contract, types, no secret-like keys, projection keeps keys | unit | `... ./internal/parser/languages -run TestCobolGrammar_FindingMetaContract ; ... ./internal/neo4jprojection -run TestProjectNode_FindingMeta` | No |
| GAP-04 | get_symbol by exact ID (MCP + CLI), search by name/kind | integration | `GOWORK=off go test -race -count=1 ./cmd/gortex -run '^TestCobolFindings_'` | No |
| GAP-05 | placeholder has no range/body/attributes, never a candidate, swept on last reference, owned (SEC-08) | unit + real path | `... ./internal/resolver -run TestCobolReconcile_Placeholder ; ... ./internal/graph/... -run PrefixDiagnostics` | No / update |
| GAP-06 | identical rerun and line shift keep IDs; clean x2 and opposite track orders equal | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobol(Lifecycle\|Findings)_'` | partial (harness exists) |
| GAP-07 | fix closes; text edit supersedes with successor; retrieve closes EXTERNAL only; parser re-stamp keeps ID; restore reopens; history survives restart and reindex, cleared by purge | unit + real path | `... ./internal/graph -run TestFindingLedger ; ... ./internal/graph/store_sqlite -run TestFindingHistory ; ... ./internal/indexer -run TestCobolFindings_History` | No |
| GAP-07 | ledger rows land in SQLite on the shadow path | real path | `... ./internal/indexer -run TestCobolFindings_HistoryShadowPath` (set `GORTEX_SHADOW_MAX_FILES` above corpus size) | No |
| GAP-08 | positive shapes produce `uncontained-cascade`; negatives keep 20 paragraphs and 20 data items; no flag | real path | `GOWORK=off go test -race -count=1 ./internal/indexer -run '^TestCobolCascadeGate'` | No |
| Deferred P3 | WR-06 identity-only copybook + `analysis-failed`; WR-07 ambiguous cursor; IN-08 distinct reason; WR-09 still green | unit | `... ./internal/parser/languages -run 'TestCobolGrammar_(CopybookAnalysisFailed\|CursorAmbiguous\|NestedContainmentUnresolved\|CopybookProcedureMainframe)' ; ... ./internal/resolver -run TestCobolReconcile_ContainmentUnresolved` | No / WR-09 yes |
| Regression | Phase 2-4 COBOL suites with updated expectations | real path | `GOWORK=off go test -race -count=1 ./cmd/gortex ./internal/indexer ./internal/resolver -run 'Cobol'` | yes (update) |

WR-06 test seam: inside package `languages`, call `e.once.Do(e.init)` then set `e.analyzer = preprocessor.Analyzer{}`; the zero analyzer returns "analyzer has no attested tool identity" from `build` [VERIFIED: preprocessor analyze.go:126-130]. Grammar-mismatch and init failures must still fail extraction (BASE-02); only analysis errors fall back.

Document-scope gaps (`no-tree`, `budget-exceeded`, `invocation-failure`, `unmapped-capture`) cannot be produced by fixtures; unit-test the mapping by appending synthetic `handoff.Gap` values to a real handoff before calling the mapper (Gortex never calls `handoff.Validate`).

### Sampling Rate
- **Per task commit:** quick command plus `GOWORK=off go test -race -count=1 <package under edit>`
- **Per wave merge:** `GOWORK=off go test -race -count=1 ./internal/graph/... ./internal/resolver ./internal/indexer ./internal/parser/languages ./internal/mcp ./internal/neo4jprojection ./cmd/gortex -run 'Finding|Cobol|PrefixDiagnostics|IndexHealth|ProjectNode|SidecarViewGen|Migration'`
- **Phase gate:** full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/graph/finding_test.go` -- ID determinism, evidence normalization, ordinals, class separation
- [ ] `internal/graph/finding_ledger_test.go` -- open/close/supersede pairing, container and class isolation, reopen
- [ ] `internal/graph/storetest` case for `EvictFindingNodesByIDs` (both backends)
- [ ] `internal/graph/store_sqlite/store_finding_history_test.go` + update `schema_sidecar_view_gen_test.go`
- [ ] `internal/parser/languages/cobol_grammar_findings_test.go` (fixtures from the probe table)
- [ ] `internal/resolver/cobol_findings_test.go`
- [ ] `internal/indexer/cobol_findings_lifecycle_test.go`, `internal/indexer/cobol_cascade_gate_test.go` (reuse `newCobolLifecycleEnv`)
- [ ] `internal/mcp/index_health_findings_test.go` (pattern: `index_health_ownership_test.go`)
- [ ] `cmd/gortex/cobol_findings_test.go` (pattern: `TestCobolTracer_MCPGetSymbol` / `CLIGetSymbol`)
- [ ] `internal/neo4jprojection` finding Meta case (pattern: `TestProjectNode_CobolBreadthMeta`)
- Framework install: none

## Security Domain

Security enforcement is enabled (ASVS level 1, block on high).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | yes | Findings and placeholders carry the referencing repository's `RepoPrefix`/workspace/project; `get_symbol` filters with `resolvedScopeAllowsNode`; `index_health` counts only in-scope repositories (SEC-01); ownership audit stays clean (SEC-08) |
| V5 Input Validation | yes | Finding IDs are hashes, so source text cannot forge an ID or path segment; placeholder keys reuse the extractor's `cobolIDSegment`-encoded key; Meta types restricted to the contract set |
| V6 Cryptography | no (identity hashing only) | BLAKE3 from the existing dependency; never hand-roll |
| V7 Error Handling and Logging | yes | Do not persist raw analysis error text in node Meta or history (record a coarse class only); history stores digests, not source |
| V8 Data Protection | yes | No raw gap source text in finding Meta, history, or projection; ranges let scoped tools read source under the 04-17 gate |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Crafted COBOL names forging IDs or namespaces | Tampering | Hash-based finding IDs; `@finding` root cannot be a COBOL word; encoded placeholder keys |
| Cross-repository placeholder sharing (pre-Phase-4 SEC-03 class) | Information disclosure | One placeholder per referencing repository; never unprefixed |
| Protected source leaking into Neo4j export or logs via findings | Information disclosure | Digest + range only; secret-key filter compatibility test |
| Pathological file producing very many findings (lost captures, tail tokens) | Denial of service | Bounded by producer observations per file; COBOL analysis already serialized (`cobolAnalyzeSlot`); `index_health` counting is background and cancellable |
| Unbounded `finding_history` growth | Denial of service | Rows only on transitions; zero rows on identical reruns; purged with the repository |
| Closing findings through a tool or AI path | Tampering / repudiation | D-16: only extractor, resolver, and ledger code paths write `KindFinding` nodes or history; consider a parse-level inventory test |

## Sources

### Primary (HIGH confidence)
- Pinned producer module `preprocessor@v0.0.0-20261002100409-e64234cc493c` in the Go module cache: `handoff/gaps.go`, `handoff/handoff.go`, `parsefacts/facts.go`, `grade/grade.go`, `analyze.go`, `transform/endevor.go`, `transform/fixed.go`, `sourcemap/sourcemap.go` (read this session)
- Probe program against the pinned producer, run this session (outputs summarized in the fixture table)
- Gortex source read this session: `cobol_grammar.go`, `cobol_grammar_walk.go`, `cobol_grammar_copybook.go`, `cobol_grammar_refs.go`, `cobol_grammar_sql.go`, `cobol_grammar_identity.go`, `resolver/cobol_references.go`, `resolver/cobol_resources.go`, `resolver/external_calls.go`, `graph/prefix_diagnostics.go`, `graph/stub.go`, `graph/node.go`, `graph/store.go`, `graph/config_node_evict.go`, `store_sqlite/{schema_version,schema,store_file_index_failures,store_purge,file_batch_evict,config_node_evict,store_fts}.go`, `indexer/{indexer,cobol_reconcile,incremental_batch,source_revision,sidecar_batch,multi}.go`, `mcp/{tools_core,tools_enhancements}.go`, `neo4jprojection/{properties,identity,neo4j}.go`, `testutil/graphdump/graphdump.go`, Phase 4 lifecycle tests
- Planning: `05-CONTEXT.md`, `REQUIREMENTS.md`, `STATE.md`, `GRAPH-SCHEMA.md`, `COBOL-PARSER-READINESS.md`, `docs/ai-enhanced-cobol-graph-handoff.md` §6, §8, §9, §12, `03-REVIEW.md` (WR-06/07/09, IN-08), `03-VERIFICATION.md` deferred list, `WINDOWS.md` 4-6, `04-VALIDATION.md`, `04-VERIFICATION.md`

### Secondary (MEDIUM confidence)
- None needed. No external documentation lookups were required: the phase adds no dependencies, and all web/doc providers are disabled in `.planning/config.json`.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new packages; versions read from `go.mod` and the module cache
- Architecture: MEDIUM-HIGH -- seams and precedents verified in code; placement of the ledger pass and the placeholder kind are recommendations inside discretion areas
- Pitfalls: HIGH -- Pitfalls 1-3 and 5-10 reproduced by probe or confirmed by reading the exact lines

**Research date:** 2026-10-19
**Valid until:** 2026-11-18 (stable while the producer pin `e64234c` is unchanged; re-check Pitfalls 1-2 on any re-pin)
