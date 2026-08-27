# Gortex Mainframe Engine

## What This Is

A private fork of [zzet/gortex](https://github.com/zzet/gortex) — the graph-based, multi-language code-intelligence engine (Go, tree-sitter, daemon + MCP/CLI/API) — being evolved into a fully-fledged **graph-based mainframe processing engine**. It ingests a mainframe estate into a queryable graph, layers deterministic analysis on top, then LLM enrichment, and ultimately grows into a digital twin of the estate. Built for one user's modernization work, not for upstream.

## Core Value

A trustworthy graph representation of a mainframe estate that modernization cutover decisions can be made against — deterministic and reproducible first, enriched and simulated later.

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

### Active

<!-- Big-picture staged vision. Detailed per-stage scoping is deliberately deferred to milestone/phase planning. -->

- [ ] **Stage 1 — Deterministic mainframe processing:** ingest mainframe artifacts into the graph (COBOL + copybooks first-class; aspirationally "everything" — JCL, DB2, CICS, etc.), consuming pre-processed input from `cobol-repo-architect`
- [ ] **Register the vendored COBOL tree-sitter grammar for `.cpy`** — the next
  concrete step. `go-sitter-forest/cobol` v1.9.1 (vendoring
  `yutaro-sakamoto/tree-sitter-cobol`, MIT) is **already a `go.mod` dependency
  and already compiled in**, with no extractor registered because `cobol.go`
  claims `.cbl`/`.cpy`. Measured recall on 958 DCC copybooks: **93%** of 24,478
  fields once wrapped in a synthetic program shell, against **0** from the regex
  extractor. This is the DATA DIVISION, i.e. the field-level lineage everything
  else is for
- [ ] **Extend the grammar for `EXEC CICS`/`EXEC SQL` and IDMS DML** — programs
  sit at 27% recall because grammar errors **cascade to the end of their
  division**: one `EXEC CICS` leaves 1 of 20 following paragraphs, one IDMS
  `SCHEMA SECTION` leaves 0 of 20 data items. Upstreamable to `@yutaro-sakamoto`
- [ ] **JCL symbolic resolution** (`SET` / `INCLUDE` / `JCLLIB ORDER`) — absent
  entirely, so `DSN=&DCC1XN..EXTRACT` never resolves and the job→dataset→job
  graph never forms. Unmeasured; needs its own baseline pass
- [ ] **Stage 1 — Deterministic analyses** over the mainframe graph (impact analysis, lineage, batch flow — exact capability set TBD at phase planning)
- [ ] **Stage 2 — LLM enrichment layer** on top of the deterministic graph (interpretation, summarization, business-rule extraction — scoped later)
- [ ] **Stage 3 — Digital twin, staged:** static structural twin → behavioral simulation → live-synced twin (feasibility-gated), with data integration
- [ ] Twin outputs that directly support **modernization cutover** (the end goal all stages serve)

### Out of Scope

- ~~Upstream contributions / PR parity with zzet/gortex~~ — **needs revisiting.**
  Items `11e79383` and `028ef49e` are generically correct COBOL fixes (wrong for
  any mainframe shop, not Uniti-specific) and the branch is pushed to `origin`.
  Either this exclusion or the upstream framing has to give. A human decision;
  no PR should be opened until it is settled
- Reusing `cobol-ingestor` or `mainframe-viewer` internals — explicit fresh start; those tools stay separate
- Committing to a v1 artifact list or deterministic feature set now — user chose to keep planning big-picture; details land in phase planning
- Live-synced twin as a hard commitment — pursued only "if possible and feasible" after static + behavioral stages prove out

## Context

- Upstream base: gortex **v0.63.8-91** (synced 2026-08-27 from v0.61.2, 1,304 commits; the 6 local commits were docs-only and rebased cleanly) — Go, tree-sitter grammars, sqlite-fts5 search, long-running daemon, MCP/CLI/API surfaces. Full codebase map in `.planning/codebase/`.
- `cobol-repo-architect` (sibling project in GoApps) is the **preprocessing front door**: it sets up / pre-processes mainframe source for this engine to ingest.
- Related prior work (`cobol-ingestor`, `mainframe-viewer`) informs the domain but is intentionally not coupled.
- Fork remotes: `origin` = MuiGoku123432/gortex (personal, via `github-personal` SSH), `upstream` = zzet/gortex.
- The engine's existing multi-language graph, dataflow, and analyzer machinery is the substrate the mainframe layers extend.

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
| Hybrid parser: tree-sitter for copybooks, island regex for programs | Measured, not assumed — 93% field recall on `.cpy` versus 27% on `.cbl`, because grammar errors cascade to end-of-division. Same conclusion Koopa reached for real COBOL | ✓ Decided 2026-08-27 |
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
*Last updated: 2026-08-27 — items 0 and 1 shipped, tree-sitter grammar measured.*

*Companion docs: `FORK-NOTES.md` (how to run this fork, fix order, upstream
workflow) and `PythonApps/cobol-kg/HANDOFF.md` (session entry point, corpus,
measurements).*
