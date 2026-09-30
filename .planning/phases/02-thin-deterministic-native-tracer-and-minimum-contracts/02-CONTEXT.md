# Phase 2: Thin Deterministic Native Tracer and Minimum Contracts - Context

**Gathered:** 2026-09-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Trace one named COBOL program from the enhanced grammar (via the `preprocessor` handoff, `cobol-handoff-v1`) through the registered production `cobol` extractor, `ExtractionResult`, repository prefixing, the indexer lifecycle, and the SQLite `AddBatch` transaction, emitting it as a native node plus a native source-file edge. The persisted node must carry stable repository-prefixed identity and the minimum provenance contract (scope, path, exact original range, revision, parser/extractor versions, evidence class, origin, confidence, document grade), and must be retrievable through an existing CLI query and an existing MCP query — with AI providers disabled, no Neo4j, and SQLite as the sole authority.

Out of scope here (later phases): paragraphs/sections/PERFORM/CALL/COPY/data items/IDMS/CICS/SQL from the grammar (Phase 3), incremental lifecycle keyed on content/tool IDs (Phase 4), `PARSE_UNRESOLVED` / `EXTERNAL_UNRESOLVED` findings from `Gaps` (Phase 5), BASE-04 breadth acceptance (Phase 6).

</domain>

<decisions>
## Implementation Decisions

### Parser baseline and resolution
- **D-01:** The approved parser line is the accepted **v0.26.0** "Estate Parse Recovery" line of `tree-sitter-cobol-upgrade` (branch `gsd/v0.26.0-estate-parse-recovery`, accepted at `f19029f` on 2026-09-30), consumed through its `preprocessor` handoff. `97ac9f1` remains the behavioral floor. BASE-01 and BASE-03 wording in `.planning/REQUIREMENTS.md` (and Phase 2 success criterion 1 in ROADMAP.md) must be amended in this phase to name the pinned v0.26.0 commit instead of `97ac9f1`. — **Reversibility:** costly — changing the baseline again means re-pinning, re-attesting, and re-running acceptance.
- **D-02:** Resolve reproducibly under `GOWORK=off` via `go.mod`: `require github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor` at a pseudo-version of the pinned commit, and `replace github.com/alexaandru/go-sitter-forest/cobol => github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol <pseudo-version>` (the shim intentionally declares the upstream module path). `go.sum` pins both. Builds need `GOPRIVATE` covering the fork repo. This **supersedes** the 2026-09-15 checkpoint answers "Pinned fork module" / "Fork-owned path", which assumed published modules. The stale local `go.work` (points at nonexistent `./forest-shim/cobol`, `./preprocessor`) must not be what makes builds pass.
- **D-03:** Grammar attestation must not require a fork checkout on disk at runtime. The fork adds an embedded-attestation constructor (attests against a grammar hash compiled into the module); Gortex calls it instead of `preprocessor.NewAnalyzer(repoRoot, …)`. This is a **producer-side prerequisite** for execution — planning must schedule/flag it (a fork change in `~/repos/mine/devDeps/tree-sitter-cobol-upgrade`), not work around it.
- **D-04 (carried from 2026-09-15):** Each extraction records parser identity as **commit + grammar content hash** (map onto the handoff's `ToolID` / `Tool`, `ParseConfigurationID`, `Ledger.ConfigurationID`).
- **D-05 (carried from 2026-09-15):** If the enhanced parser is unavailable or its fingerprint/attestation differs, **COBOL extraction fails** — no skip, no regex fallback, no silent stock grammar.

### Extractor transition
- **D-06:** Direction is **grammar-only**: the regex `CobolExtractor` is not used as a fallback or hybrid; COBOL facts come from the grammar handoff. User criterion: "whatever gets us the best and most accurate result" — research finalizes the exact transition shape (including how the regex-derived paragraphs/calls/COPY edges are handled until Phase 3 rebuilds them from `Observations`) and must justify it against accuracy. — **Reversibility:** costly — removes currently shipped DCC extraction until Phase 3.
- **D-07:** Amber/red documents and `Affected=true` observations are **emitted, marked degraded**: document grade and the affected flag are persisted as separately queryable provenance; nothing is silently dropped.

### Provenance contract
- **D-08:** Provenance dimensions live in **namespaced flat `Meta` keys** (e.g. `prov.*`: scope, revision, parser tool/config IDs, extractor version, evidence class, origin, confidence, document grade, affected). No new `graph.Node` fields or SQLite columns (fork hygiene); keys must round-trip through SQLite and the Phase 1 Neo4j projection and be individually queryable (PROV-06). Evidence class uses the handoff §6 vocabulary, mapped onto existing origin/confidence mechanisms before inventing new ones. — **Reversibility:** costly — key names become a contract for queries, Neo4j properties, and later phases.
- **D-09:** Revision (PROV-04) = the handoff's `OriginalContentID` **always**, plus the VCS commit when the file is under version control; when there is no VCS revision, record an explicit absence reason rather than omitting it.
- **D-10:** Program identity extends `<repo_prefix>/<path>::NAME`. Nested programs use the containment path (`::OUTER/INNER`); same-level duplicates get a deterministic occurrence ordinal (`::NAME#2`). Line numbers are never a discriminator (ID-02). All persisted ranges are **original** coordinates via the handoff `Map`. — **Reversibility:** costly — IDs are persisted in SQLite, projected to Neo4j, and referenced by later phases.

### Acceptance surface
- **D-11:** Retrieval is proven with MCP `get_symbol` and its matching `gortex` CLI verb, fetching by the exact repository-prefixed ID and asserting identity plus every provenance key; plus a `search_symbols` by-name hit for discoverability.
- **D-12:** The acceptance test is **in-process on the real path**: real registered extractor → indexer lifecycle → SQLite `AddBatch` into a temp store, then the real CLI command and MCP handler against it; AI providers forced off, no Neo4j; runs under `go test -race` (follow the Phase 1 `graphfixture` / SQLite test pattern).
- **D-13:** CI gets **read access** to the private fork (`GOPRIVATE` + read-only deploy key or fine-grained token as a CI secret) so the tracer runs in CI like any other test — no skip-when-missing. Provisioning the secret is a user action the plan must surface.
- **D-14:** Fixtures are **invented COBOL only** (repo is public): one clean/green program as the tracer, plus guard cases — a nested program (containment-path ID), a degraded/amber program (degraded marking), and a parser-mismatch case that must fail.

### Claude's Discretion
- Final extractor transition shape under D-06 (research decides, criterion: accuracy).
- Native node kind for the program (keep `KindFunction` vs new `KindProgram`) — research decides, weighing existing query compatibility and fork hygiene against schema fidelity (`GRAPH-SCHEMA.md`).
- Exact `prov.*` key names, confidence value for deterministic facts, extractor version string.
- COPY/INCLUDE catalog for the tracer (readiness doc suggests `transform.NewCatalog(nil)`), parse concurrency/memory limits (readiness doc §6: one file at a time or low concurrency on 16 GB hosts), exact CLI verb name.
- Whether the local `go.work` is deleted or kept only as an optional developer override for iterating on the fork.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Parser readiness and handoff (primary)
- `.planning/research/COBOL-PARSER-READINESS.md` — delivery modules, API (`NewAnalyzer`, `NewSourceID`, `Analyze`), handoff field → Gortex mapping, boundary rules D10, "do not pin `ledger.configuration_id` / `gaps[].id`", producer limitations, privacy rules
- `~/repos/mine/devDeps/tree-sitter-cobol-upgrade` (local fork checkout; `docs/vendoring.md`, `preprocessor/handoff/testdata/consumer-v1.golden.json`) — producer contract; never copy estate content from it

### Milestone contract
- `docs/ai-enhanced-cobol-graph-handoff.md` §4 (existing substrate), §6 (evidence classes), §12 (stable identity), §18 (non-goals)
- `.planning/REQUIREMENTS.md` — BASE-01..03, TRACE-01..07, PROV-01..06, ID-01..02 (BASE-01/03 to be amended per D-01)
- `.planning/ROADMAP.md` — Phase 2 goal and success criteria
- `.planning/PROJECT.md` — constraints (fork hygiene, sequencing), Key Decisions

### Graph schema, storage, architecture
- `.planning/research/GRAPH-SCHEMA.md` — node/edge mapping (input to node-kind decision)
- `.planning/research/STORAGE-ADR.md` — SQLite sole authority
- `.planning/research/ARCHITECTURE.md`, `.planning/research/STACK.md`
- `.planning/phases/01-general-neo4j-projection-command/01-CONTEXT.md` — projection of node Meta/labels that D-08 keys must survive

### Existing COBOL tooling
- `internal/parser/forest/cobolprobe/README.md` — grammar probe harness

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/parser/languages/cobol.go` — regex `CobolExtractor` (`NewCobolExtractor`, `Extract`); program nodes today are `KindFunction` with ID `filePath::NAME` and an `EdgeDefines` edge from the file node. Being superseded per D-06.
- `internal/parser/languages/register.go:119` — `reg.Register(NewCobolExtractor())`, the registration point for the production extractor.
- `parser.ExtractionResult` / `graph.Node.Meta` — carrier for nodes, edges, and `prov.*` keys.
- `internal/testutil/graphfixture` (Phase 1) — SQLite fixture pattern for the in-process acceptance test.

### Established Patterns
- Multi-repo IDs `<repo_prefix>/<path>::<Symbol>`; repository prefixing applied by the indexer, not the extractor.
- Edge `Origin` evidence tiers (e.g. `graph.OriginASTInferred`); `Node.Origin` is currently used only for cross-daemon proxy nodes — do not overload it without checking.

### Integration Points
- `go.mod:28` — `github.com/alexaandru/go-sitter-forest/cobol v1.9.1` (stock); the D-02 `replace` targets this.
- `internal/parser/forest/dump_kinds_test.go` imports `cobolforest` — will pick up the shim grammar after D-02; check it still passes.
- Stale `go.work` in repo root (git-ignored) — must not mask a broken `GOWORK=off` build.
- `.github/workflows/ci.yml` — needs `GOPRIVATE` + fork read credential (D-13).

</code_context>

<specifics>
## Specific Ideas

- User intent: rely on the grammar, not the regex parser ("i imagine we dont use the regex parser at all and rely on the grammar").
- Accuracy over convenience is the tie-breaker for anything research decides.
- Readiness doc sequencing: fix resolution first, then register an extractor that calls `Analyze` on one invented fixture.

</specifics>

<deferred>
## Deferred Ideas

- Exported INCLUDE/COPY catalog loader from the fork (readiness doc §2 follow-up) — needed for Phase 3/4 breadth, not the tracer.
- Fork publishing tagged modules (`preprocessor/vX`, `forest-shim/cobol/vX`) to replace pseudo-versions — nice-to-have follow-up.

</deferred>

---

*Phase: 02-thin-deterministic-native-tracer-and-minimum-contracts*
*Context gathered: 2026-09-30*
