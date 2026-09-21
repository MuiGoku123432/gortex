# Gortex Mainframe Engine

## What This Is

A private fork of [zzet/gortex](https://github.com/zzet/gortex) — the graph-based, multi-language code-intelligence engine (Go, tree-sitter, daemon + MCP/CLI/API) — being evolved into a fully-fledged **graph-based mainframe processing engine**. It ingests a mainframe estate into a queryable graph, layers deterministic analysis on top, then LLM enrichment, and ultimately grows into a digital twin of the estate. Built for one user's modernization work, not for upstream.

## Core Value

A trustworthy graph representation of a mainframe estate that modernization cutover decisions can be made against — deterministic and reproducible first, enriched and simulated later.

## Current Milestone: v1.0 Neo4j Projection and Mainframe Graph Foundation

**Goal:** First provide a general, manually invoked, rebuildable Neo4j projection of Gortex's authoritative SQLite graph; when the enhanced COBOL grammar is ready for production integration, convert its named AST contract into a reproducible, provenance-aware native graph and add optional validated AI claims without changing deterministic facts.

**Target features:**
- A language-agnostic CLI command and MCP tool that materialize an explicitly scoped SQLite graph snapshot into Neo4j
- Stable projection keys, Neo4j constraints, bounded retry-safe upserts, explicit stale-record handling, dry-run planning, progress/results, and cancellation/error behavior
- SQLite as the sole authority: no reverse writes and no synchronous Neo4j dual-write in indexing
- A thin deterministic tracer from one existing COBOL parser node through SQLite persistence and native CLI/MCP query surfaces
- Stable identity, exact source provenance, idempotent incremental lifecycle, and explicit unresolved-gap modeling
- A manually invoked Neo4j projection boundary with SQLite remaining authoritative
- Optional bounded AI enrichment with validated, evidence-linked claims and human review

## Requirements

### Validated

<!-- Inherited from upstream gortex — shipped, working, relied upon. -->

- ✓ Graph indexing engine over 250+ languages (tree-sitter), incremental reindex via daemon — existing
- ✓ Access surfaces: MCP server, CLI, API — existing
- ✓ Graph queries: symbol search, usages, call chains, dataflow edges (`value_flow`/`arg_of`/`returns_to`), AST pattern search, clone detection — existing
- ✓ Analyzer framework (dead code, hotspots, cycles, coverage, routes/models/components, k8s, dbt, cross-repo) — existing
- ✓ Session notes + durable workspace memory layers — existing
- ✓ Token-economy wire formats (GCX1, TOON) and body compression — existing

### Shipped in this fork

<!-- Measured on the DCC pilot corpus; see PythonApps/cobol-kg/HANDOFF.md. -->

- ✓ **COBOL `PROCEDURE DIVISION USING` header** now recognised (`11e79383`) —
  `cobolDivRe` required a period straight after `DIVISION`, so 441 of 506 DCC
  programs had their whole procedure division skipped. Nodes 14,780 → 35,670,
  edges 23,449 → 90,404 in a controlled same-binary comparison
- ✓ **COBOL comment lines no longer scanned** for `COPY`/`CALL` (`028ef49e`) —
  they were 2,242 of 3,768 COPY matches (59%) and 117 of 469 CALL matches
- ✓ **`COPY IDMS [RECORD] <name>`** resolves to the copybook, not the literal
  `IDMS` — 590 statements had collapsed onto one node
- ✓ **Dynamic `CALL <identifier>`** captured in a `dyncall` namespace with
  `OriginASTInferred` — 829 versus 352 quoted literals, so most of the
  inter-program call graph was missing
- ✓ **`cobolprobe`** (`80f0c08e`) — harness measuring the vendored COBOL grammar
  against a real estate; skips without a `-corpus` flag
- ✓ **Enhanced deterministic COBOL parser baseline** — `tree-sitter-cobol-upgrade/main`
  at merge commit `97ac9f1`, including DATA DIVISION, IDMS, CICS, and SQL nodes,
  bounded unparsed tails, 149/149 non-comment corpus assertions, 371 passing NIST
  COBOL-85 tests, and merged gortex cascade/acceptance gates

### Active

<!-- Big-picture staged vision. Detailed per-stage scoping is deliberately deferred to milestone/phase planning. -->

- [ ] Deliver the immediate priority: a general, language-agnostic command available through CLI and MCP that projects a selected workspace/project/repository SQLite snapshot into Neo4j
- [ ] Make projection credentials explicit but non-disclosing, preserve graph scope/provenance/evidence properties, and support constraints, stable keys, bounded batches, retries, stale-record reconciliation, dry-run, progress, cancellation, and partial-failure recovery
- [ ] Keep Neo4j completely outside indexing and all ordinary Gortex availability paths; SQLite remains the sole read/write authority
- [ ] Resume COBOL graph extraction only after the required parser/grammar integration is ready
- [ ] Map the completed parser contract onto gortex's existing node, edge, provenance, identity, and SQLite persistence models
- [ ] Deliver the first vertical slice as a thin deterministic tracer from one existing COBOL parser node through the native graph and CLI/MCP query surfaces
- [ ] Extend deterministic extraction across source-positioned COBOL, IDMS, CICS, SQL, copybook, call, and resource facts
- [ ] Make deterministic facts and findings stable and idempotent across identical and incremental indexing runs
- [ ] Represent parser failures and unavailable artifacts as separate, queryable unresolved gaps before any AI enrichment
- [ ] Validate the user-approved manual Neo4j projection boundary without widening it into continuous synchronization or authority
- [ ] Add optional, policy-gated AI review that emits schema-validated, evidence-linked claims without mutating deterministic facts
- [ ] Support human confirmation, contradiction, and supersession while retaining claim lineage

### Out of Scope

- ~~Upstream contributions / PR parity with zzet/gortex~~ — **needs revisiting.**
  Items `11e79383` and `028ef49e` are generically correct COBOL fixes (wrong for
  any mainframe shop, not Uniti-specific) and the branch is pushed to `origin`.
  Either this exclusion or the upstream framing has to give. A human decision;
  no PR should be opened until it is settled
- Reusing `cobol-ingestor` or `mainframe-viewer` internals — explicit fresh start; those tools stay separate
- Replacing SQLite with Neo4j -- the approved projection is downstream, manually invoked, and rebuildable
- Continuous Neo4j synchronization, index-time dual-write, or reverse writes from Neo4j
- Expanding this milestone into full JCL symbolic resolution or a complete digital twin — both remain later work unless required by the thin tracer or gap contract
- Live-synced twin as a hard commitment — pursued only "if possible and feasible" after static + behavioral stages prove out

## Context

- Upstream base: gortex **v0.63.8-91** (synced 2026-08-27 from v0.61.2, 1,304 commits; the 6 local commits were docs-only and rebased cleanly) — Go, tree-sitter grammars, sqlite-fts5 search, long-running daemon, MCP/CLI/API surfaces. Full codebase map in `.planning/codebase/`.
- `cobol-repo-architect` (sibling project in GoApps) is the **preprocessing front door**: it sets up / pre-processes mainframe source for this engine to ingest.
- Related prior work (`cobol-ingestor`, `mainframe-viewer`) informs the domain but is intentionally not coupled.
- Fork remotes: `origin` = MuiGoku123432/gortex (personal, via `github-personal` SSH), `upstream` = zzet/gortex.
- The engine's existing multi-language graph, dataflow, and analyzer machinery is the substrate the mainframe layers extend.
- The authoritative parser baseline is `tree-sitter-cobol-upgrade/main` at merge commit `97ac9f1`; gortex consumes its `forest-shim/cobol` module through the local `go.work` boundary, and merged acceptance tests protect the integration.
- Gortex's durable graph remains SQLite today. Existing Neo4j support is a manual Cypher export path, not an authoritative or synchronized store.
- The user has approved a general SQLite-to-Neo4j projection command as the immediate v1.0 priority. It must remain language-agnostic and usable independently of COBOL parser readiness.
- COBOL tracer and grammar-dependent phases remain in the milestone but are deferred behind parser/grammar readiness. The completed parser baseline and its history remain authoritative planning evidence.

## Constraints

- **Tech stack**: Go + tree-sitter ecosystem, extending gortex's existing architecture — evolve the engine, don't rewrite it
- **Fork hygiene**: keep the ability to pull upstream improvements — prefer additive packages/analyzers over invasive edits to upstream internals where practical
- **Sequencing**: deterministic layer before LLM enrichment before twin — trust and reproducibility are prerequisites for the later stages
- **Ownership**: personal project under the MuiGoku123432 GitHub account

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Fork zzet/gortex rather than build from scratch | Mature graph engine, daemon, MCP/CLI surfaces, and analyzer framework already exist | — Pending |
| Fresh start vs. existing mainframe tools | Avoid legacy coupling; `cobol-repo-architect` becomes the preprocessing front door instead | — Pending |
| Staged twin: static → behavioral → live-synced | De-risks the vision; each stage has standalone modernization value | — Pending |
| Deterministic before LLM enrichment | LLM interpretation belongs on top of a reproducible substrate, not in place of one | — Pending |
| Fix the existing COBOL/JCL extractors rather than write new ones | Gortex already shipped them, and `cobolStripLine` already had the fixed-format column model right. Every gap was specific and measurable | ✓ Items 0 and 1 shipped 2026-08-27 |
| Hybrid parser: tree-sitter for copybooks, island regex for programs | Superseded by the completed enhanced parser baseline, which now preserves the required DATA DIVISION, IDMS, CICS, and SQL structures under gortex acceptance tests | Superseded 2026-09-15 |
| Parser baseline for graph milestone | Pin planning and acceptance evidence to `tree-sitter-cobol-upgrade/main` merge `97ac9f1`; parser changes are not part of this milestone unless a graph-blocking regression is proven | ✓ Decided 2026-09-15 |
| Native graph before AI enrichment | Deterministic extraction, stable identity, and explicit unresolved gaps must exist before claims can be generated or consumed | ✓ Decided 2026-09-15 |
| General Neo4j projection is the immediate priority | Users need an explicit CLI/MCP path to materialize selected SQLite graph snapshots now; this capability is language-agnostic and does not depend on COBOL grammar readiness | ✓ Decided 2026-09-21 |
| SQLite remains sole authority while Neo4j is a downstream projection | A manual, rebuildable projection provides Neo4j utility without split-brain indexing, reverse writes, or making ordinary Gortex operations depend on Neo4j | ✓ Decided 2026-09-21 |
| Defer COBOL graph phases until parser/grammar readiness | Planning history and baseline `97ac9f1` are preserved, but implementation should not start while required grammars are unavailable | ✓ Decided 2026-09-21 |
| Run this fork's daemon on an isolated `~/.gortex-fork` store | Keeps the official Homebrew install tracking every other repo while the fork is three minor versions ahead. Verified: fork-only commands leave `~/.gortex` byte-identical | ✓ Verified 2026-08-27 |
| Drop `COPY REPLACING` and `COPY x OF y` from scope | Measured **zero** occurrences in the estate; building for them would be speculative generality | ✓ Decided 2026-08-27 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-09-21 -- v1.0 reprioritized to the general Neo4j projection command; COBOL graph phases remain planned but deferred until parser/grammar readiness.*

*Companion docs: `FORK-NOTES.md` (how to run this fork, fix order, upstream
workflow) and `PythonApps/cobol-kg/HANDOFF.md` (session entry point, corpus,
measurements).*
