# Phase 2: Thin Deterministic Native Tracer and Minimum Contracts - Research

**Researched:** 2026-09-30
**Domain:** Go module resolution of a private fork, tree-sitter COBOL grammar handoff, Gortex extractor/indexer/SQLite/MCP/CLI plumbing, provenance contract
**Confidence:** HIGH for module mechanics, attestation, indexer path, and retrieval surfaces (proven with probes this session). MEDIUM for the VCS-revision stamp design and the SQLite-backed daemon test harness (designed, not yet built).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Parser baseline and resolution
- **D-01:** The approved parser line is the accepted **v0.26.0** "Estate Parse Recovery" line of `tree-sitter-cobol-upgrade` (branch `gsd/v0.26.0-estate-parse-recovery`, accepted at `f19029f` on 2026-09-30), consumed through its `preprocessor` handoff. `97ac9f1` remains the behavioral floor. BASE-01 and BASE-03 wording in `.planning/REQUIREMENTS.md` (and Phase 2 success criterion 1 in ROADMAP.md) must be amended in this phase to name the pinned v0.26.0 commit instead of `97ac9f1`. — **Reversibility:** costly — changing the baseline again means re-pinning, re-attesting, and re-running acceptance.
- **D-02:** Resolve reproducibly under `GOWORK=off` via `go.mod`: `require github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor` at a pseudo-version of the pinned commit, and `replace github.com/alexaandru/go-sitter-forest/cobol => github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol <pseudo-version>` (the shim intentionally declares the upstream module path). `go.sum` pins both. Builds need `GOPRIVATE` covering the fork repo. This **supersedes** the 2026-09-15 checkpoint answers "Pinned fork module" / "Fork-owned path", which assumed published modules. The stale local `go.work` (points at nonexistent `./forest-shim/cobol`, `./preprocessor`) must not be what makes builds pass.
- **D-03:** Grammar attestation must not require a fork checkout on disk at runtime. The fork adds an embedded-attestation constructor (attests against a grammar hash compiled into the module); Gortex calls it instead of `preprocessor.NewAnalyzer(repoRoot, …)`. This is a **producer-side prerequisite** for execution — planning must schedule/flag it (a fork change in `~/repos/mine/devDeps/tree-sitter-cobol-upgrade`), not work around it.
- **D-04 (carried from 2026-09-15):** Each extraction records parser identity as **commit + grammar content hash** (map onto the handoff's `ToolID` / `Tool`, `ParseConfigurationID`, `Ledger.ConfigurationID`).
- **D-05 (carried from 2026-09-15):** If the enhanced parser is unavailable or its fingerprint/attestation differs, **COBOL extraction fails** — no skip, no regex fallback, no silent stock grammar.

#### Extractor transition
- **D-06:** Direction is **grammar-only**: the regex `CobolExtractor` is not used as a fallback or hybrid; COBOL facts come from the grammar handoff. User criterion: "whatever gets us the best and most accurate result" — research finalizes the exact transition shape (including how the regex-derived paragraphs/calls/COPY edges are handled until Phase 3 rebuilds them from `Observations`) and must justify it against accuracy. — **Reversibility:** costly — removes currently shipped DCC extraction until Phase 3.
- **D-07:** Amber/red documents and `Affected=true` observations are **emitted, marked degraded**: document grade and the affected flag are persisted as separately queryable provenance; nothing is silently dropped.

#### Provenance contract
- **D-08:** Provenance dimensions live in **namespaced flat `Meta` keys** (e.g. `prov.*`: scope, revision, parser tool/config IDs, extractor version, evidence class, origin, confidence, document grade, affected). No new `graph.Node` fields or SQLite columns (fork hygiene); keys must round-trip through SQLite and the Phase 1 Neo4j projection and be individually queryable (PROV-06). Evidence class uses the handoff §6 vocabulary, mapped onto existing origin/confidence mechanisms before inventing new ones. — **Reversibility:** costly — key names become a contract for queries, Neo4j properties, and later phases.
- **D-09:** Revision (PROV-04) = the handoff's `OriginalContentID` **always**, plus the VCS commit when the file is under version control; when there is no VCS revision, record an explicit absence reason rather than omitting it.
- **D-10:** Program identity extends `<repo_prefix>/<path>::NAME`. Nested programs use the containment path (`::OUTER/INNER`); same-level duplicates get a deterministic occurrence ordinal (`::NAME#2`). Line numbers are never a discriminator (ID-02). All persisted ranges are **original** coordinates via the handoff `Map`. — **Reversibility:** costly — IDs are persisted in SQLite, projected to Neo4j, and referenced by later phases.

#### Acceptance surface
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

### Deferred Ideas (OUT OF SCOPE)
- Exported INCLUDE/COPY catalog loader from the fork (readiness doc §2 follow-up) — needed for Phase 3/4 breadth, not the tracer.
- Fork publishing tagged modules (`preprocessor/vX`, `forest-shim/cobol/vX`) to replace pseudo-versions — nice-to-have follow-up.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BASE-01 | Production COBOL extraction uses parser behavior reproducibly equivalent to the approved baseline (to be amended to the pinned v0.26.0 commit per D-01) | Q3 (replace works, probe-proven), Q4 (embedded attestation), Gortex-side approved grammar pin (Pattern 2) |
| BASE-02 | `GOWORK=off` build resolves the approved baseline or fails, never silently stock | Q3 negative probe: stock grammar makes `NewAnalyzer` return `compiled forest language identity mismatch`; Pattern 2 fail-closed extractor |
| BASE-03 | Each extraction records a comparable grammar content identifier | `prov_parser_grammar_id` = `Tool.CompiledLanguageID` (Q5 key table) |
| TRACE-01 | One named program-definition node traced from the enhanced parser through the registered production extractor | Q1 (transition shape), register.go:119 swap |
| TRACE-02 | Native node + native source-file edge | `KindFunction` + `EdgeDefines` (Q2) |
| TRACE-03 | Through `ExtractionResult`, repository prefixing, indexer lifecycle | Q6 (prefix applied by `applyRepoPrefix` after extraction); MultiIndexer path always prefixes |
| TRACE-04 | Through SQLite `AddBatch` | Q7: set `GORTEX_SHADOW_MAX_FILES=0` or the cold index bypasses AddBatch via the in-memory shadow + BulkLoad; counting wrapper proves AddBatch |
| TRACE-05 | Retrievable through an existing CLI query | `gortex call get_symbol --arg id=… --arg detail=full` and `gortex query symbol <name>` (Q7) |
| TRACE-06 | Retrievable through an existing MCP query | MCP `get_symbol` with `detail:"full"` (brief omits Meta), plus `search_symbols` (Q7) |
| TRACE-07 | Passes with AI providers disabled and no Neo4j | In-process MCP server never sets `llmService`; no Neo4j profile configured (Q7) |
| PROV-01 | Repository, workspace, project scope | Native `RepoPrefix`/`WorkspaceID`/`ProjectID` stamped by `applyRepoPrefix` (Q5) |
| PROV-02 | Normalized repository-relative path | `prov_source_path` = unprefixed `relPath` the extractor receives (Q5) |
| PROV-03 | Exact start/end row and column from the parser | Native 1-based line/0-based column fields plus verbatim 0-based `prov_*` projection fields (Q5) |
| PROV-04 | Revision that supplied the bytes | `prov_revision_content_id` always + `prov_vcs_commit` or `prov_vcs_commit_absence` (Q5, Pattern 4) |
| PROV-05 | Parser and extractor version identifiers | `prov_parser_tool_id`, `prov_parser_grammar_id`, `prov_parser_module`, `prov_parse_config_id`, `prov_transform_config_id`, `prov_extractor_version` (Q5) |
| PROV-06 | Evidence class, extraction origin, confidence separately queryable | Three separate keys: `prov_evidence_class`, `prov_origin`, `prov_confidence` (Q5) |
| ID-01 | Extend repository-prefixed identity convention | `<prefix>/<path>::<PROGRAM>` via extractor `relPath::NAME` + indexer prefixing (Q6) |
| ID-02 | Line number never sole discriminator | Containment path + occurrence ordinal; probe shows grammar parent links cannot express nesting, so an END PROGRAM stack is required (Q6) |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- Build/test contract: `go build -o gortex ./cmd/gortex/` (CGO required) and `go test -race ./...` must pass. [VERIFIED: ./CLAUDE.md]
- Gortex graph tools are the mandated navigation path for indexed source; this session used shell reads because no Gortex MCP functions were mounted. Executors should use `search_symbols` / `get_symbol_source` / `get_editing_context` / `edit_file` when available. [VERIFIED: ./CLAUDE.md]
- Engineering standards: simplest change; single-use code stays inline; rule of three before extracting shared code; no speculative config/abstractions; no new dependencies without stated justification and confirmation; files < 300 lines per module where practical; follow existing folder conventions; self-review diff. [VERIFIED: ./.claude/CLAUDE.md, ~/.claude/CLAUDE.md]
- Fork hygiene: prefer additive packages/files over invasive edits to upstream internals. [VERIFIED: ./.claude/CLAUDE.md]
- GSD workflow enforcement: repo edits go through GSD commands. [VERIFIED: ./.claude/CLAUDE.md]
- Error handling: wrap with `%w`; never silently suppress. Doc comments on exported symbols. [VERIFIED: ./.claude/CLAUDE.md]
- Org policy: no customer/estate data in outputs; least-privilege credentials; never store/echo secrets. [VERIFIED: organization instructions]
- Privacy (this public repo): invented COBOL only; never paste estate source, member/program names, private root paths, or run IDs. [VERIFIED: .planning/research/COBOL-PARSER-READINESS.md:7, :94-97]

## Summary

The whole D-02 resolution path works today and was proven this session in a scratch module and through `go test -modfile` against the real Gortex tree: under `GOWORK=off` with `GOPRIVATE` set, `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/{preprocessor,forest-shim/cobol}@f19029f` resolve to pseudo-version `v0.0.0-20260930153412-f19029f11cd0`. The module-path replace is accepted because the shim's `go.mod` declares the replaced path. `go.sum` pins both modules. `NewAnalyzer` succeeds and produces a green handoff under `go test -race`. Without the replace, attestation fails closed with `compiled forest language identity mismatch`. The fork's `docs/vendoring.md` reasons for choosing `go.work` do not block D-02. Its first reason was a relative path outside the repo, and D-02 uses a module path instead. Its second reason was go.sum bookkeeping, which D-02 accepts on purpose. [VERIFIED: probes this session]

Three findings change the plan's shape. **(1)** `preprocessor` does not compile on Windows: its `internal/securefs` package has only `darwin || linux` files. Gortex CI's Windows job and Windows release builds therefore break unless the grammar extractor is build-tagged `!windows`, with a Windows stub that fails extraction. **(2)** A cold first index into SQLite does not use `AddBatch`. It swaps to an in-memory shadow graph and dumps through `BulkLoad`. TRACE-04 therefore needs `GORTEX_SHADOW_MAX_FILES=0`, an existing test knob, plus an AddBatch-counting store wrapper. **(3)** The grammar does not express nested programs through parent links. Every `program_definition` hangs off `start`, and both `END PROGRAM` markers of a nested pair attach to the inner program. D-10's `::OUTER/INNER` must therefore come from an `END PROGRAM` name stack, not from `Observation.Parent`. [VERIFIED: probes this session]

**Primary recommendation:** Two plans have to land first, in order. First, the fork adds `NewEmbeddedAnalyzer(catalog)` and pushes it. Second, Gortex pins that commit via require+replace, deletes the stale `go.work`, adds GOPRIVATE and a read credential to every Go-building CI job, and swaps `register.go:119` to a new grammar-only extractor. That extractor is `internal/parser/languages/cobol_grammar.go` with a Windows stub. It emits file node → `EdgeDefines` → `KindFunction` program nodes carrying `cobol_kind=program` and the `prov_*` key set below. Acceptance is an in-process test in `cmd/gortex` on SQLite with the shadow disabled, retrieving through `gortex call get_symbol` and MCP `get_symbol` (`detail:"full"`).

## Delegated Research Questions — Answers

### Q1. Extractor transition shape under D-06 (needs user confirmation: temporary regression)

**What the regex extractor emits today, and what Phase 2 grammar-only drops** [VERIFIED: internal/parser/languages/cobol.go:108-169, read via shell]:

| Fact | Regex emission today (verbatim) | Phase 2 recommendation |
|------|------|------|
| Program | `add(name, graph.KindFunction, line, len(lines), nil)` + `EdgeDefines` from file | **Kept.** Grammar `program_definition` → `KindFunction`, `cobol_kind=program` |
| Division | `name := string(src[m[2]:m[3]]) + "-DIVISION"` as `graph.KindType` | Dropped until Phase 3 |
| Section | `+ "-SECTION"` as `graph.KindMethod` | Dropped until Phase 3 |
| Paragraph | `graph.KindFunction` with `map[string]any{"cobol_kind": "paragraph"}` | Dropped until Phase 3 |
| COPY | `To: "unresolved::import::" + mod, Kind: graph.EdgeImports` | Dropped until Phase 3 |
| Static CALL | `To: "unresolved::" + target, Kind: graph.EdgeCalls` | Dropped until Phase 3 |
| Dynamic CALL | `To: "unresolved::dyncall::" + …`, `Origin: graph.OriginASTInferred` | Dropped until Phase 3 |
| PERFORM/THRU/GO TO | `EdgeCalls` paragraph→`filePath+"::"+name` | Dropped until Phase 3 |

Magnitude, per aggregate counts already written in the upstream code comments (no estate data added here): one corpus had "590 such statements" of `COPY IDMS`, dynamic calls "outnumbered literal ones 829 to 352", and before the comment filter "2,242 of 3,768 COPY matches and 117 of 469 CALL matches" were false positives. [VERIFIED: cobol.go:28-38, :250-259] Every COBOL file therefore loses all intra-program structure and every inter-program CALL/COPY edge until Phase 3.

**Options weighed against accuracy:**
- **A — Program + file edge only (RECOMMENDED).** This is exactly the ROADMAP/TRACE scope. It persists only facts whose identity contract is settled in this phase (ID-01/ID-02). Paragraph identity (ID-03) and resource identity (ID-04) are Phase 4 requirements. Deriving paragraphs, CALL, and COPY from `Observations` is Phase 3 scope, and Phase 3 still needs grammar kinds such as `paragraph_header` and `call_statement` [VERIFIED: consumer-v1.golden.json kinds]. For a trust-first graph, a documented absence is more accurate than regex facts that are known to be heuristic: the area-A paragraph test accepts any indent ≤ 11 [VERIFIED: cobol.go:302-315], and comment-line filtering was needed to remove a majority of false COPY matches.
- **B — Grammar extractor also emits paragraphs/sections/CALL/COPY now.** This is more complete, but it pulls Phase 3 work in and persists paragraph IDs of the form `file::PARA` that ID-03 will change. Rejected for scope and ID churn.
- **C/D — Regex fallback or hybrid.** Forbidden by D-06.

**Swap mechanics:** replace `reg.Register(NewCobolExtractor())` at `internal/parser/languages/register.go:119` [VERIFIED: register.go:119 reads `reg.Register(NewCobolExtractor())`] with the grammar extractor. Leave `cobol.go` and `cobol_test.go` unedited, because they are upstream-owned and editing them invites merge conflicts. The 9 regex unit tests keep passing because they construct `NewCobolExtractor()` directly. See Open Question 2 for keep-vs-delete.

**USER MUST CONFIRM:** COBOL paragraphs, sections, divisions, PERFORM/GO TO, CALL, and COPY edges disappear from the graph from the Phase 2 swap until Phase 3 lands. `docs`/`README` claims of "COBOL paragraphs" become temporarily false.

### Q2. Native node kind: keep `graph.KindFunction`

- GRAPH-SCHEMA is explicit: "The thin tracer should add no new kind: map a named `PROGRAM-ID` observation to existing `KindFunction`, `EdgeDefines`, exact range, and `cobol_kind=program`." [VERIFIED: .planning/research/GRAPH-SCHEMA.md:60]. It lists `program` as a "strong candidate for additive `program` kind after tracer" [VERIFIED: GRAPH-SCHEMA.md:39].
- `KindFunction NodeKind = "function"` [VERIFIED: internal/graph/node.go:29]. The Neo4j projection derives a label from the kind (`labels := []string{"GortexNode", sanitizeLabel(node.Kind)}`) [VERIFIED: internal/neo4jprojection/properties.go:52]. Existing tools such as search, get_symbol, and FTS already treat function nodes as first-class symbols. The regex path used the same kind, so JCL/other consumers that name programs see no kind change.
- A new kind now would touch node.go (upstream-owned), analyzers keyed on kinds, and projection labels, and it would be a second costly contract. **Decision: `KindFunction` + `Meta["cobol_kind"]="program"`.** Revisit in Phase 3 when paragraphs arrive.

### Q3. Go module mechanics for D-02 (probe-proven)

| Question | Answer | Evidence |
|---|---|---|
| Does a module-path replace to a nested-module subdirectory of a private repo work? | Yes. `go list -m` resolved `Path: …/forest-shim/cobol`, `Origin.Subdir: "forest-shim/cobol"`, `TagPrefix: "forest-shim/cobol/"`. The build ran `NewAnalyzer` successfully | [VERIFIED: probe `go list -m -json …@f19029f`, `go run`] |
| Does the replacement's go.mod path mismatch break it? | No. The shim declares `module github.com/alexaandru/go-sitter-forest/cobol`, which equals the *replaced* path. That is what Go requires of a replacement | [VERIFIED: forest-shim/cobol/go.mod:1; probe built] |
| Pseudo-version form | `v0.0.0-20260930153412-f19029f11cd0` for both modules. There are no `forest-shim/cobol/v*` or `preprocessor/v*` tags, and repo tags `0.1.0`/`0.1.1` are not semver-prefixed, so v0.0.0 applies | [VERIFIED: probe output `"Version": "v0.0.0-20260930153412-f19029f11cd0"`] |
| Is the branch pushed? | Yes: `git ls-remote origin` shows `f19029f11cd0e4000e5e443c2b17a9383be17ee1 refs/heads/gsd/v0.26.0-estate-parse-recovery` | [VERIFIED: ls-remote] |
| Does preprocessor require the forest path, so the replace must apply transitively? | Yes. `preprocessor/go.mod` has `require github.com/alexaandru/go-sitter-forest/cobol v1.9.1`, and `internal/tsadapter` imports `cobol "github.com/alexaandru/go-sitter-forest/cobol"`. A main-module replace applies build-wide; build info shows `dep …/cobol v1.9.1 replace=&{…/forest-shim/cobol v0.0.0-20260930153412-f19029f11cd0 h1:MLlg8…}` | [VERIFIED: preprocessor/go.mod; tsadapter/parser.go imports; probe] |
| go.sum | Adds `…/forest-shim/cobol v0.0.0-… h1:` + `/go.mod h1:` and the same pair for `…/preprocessor`. The stock `cobol v1.9.1` lines remain | [VERIFIED: probe go.sum] |
| GOPRIVATE/GONOSUMDB | `GOPRIVATE=github.com/MuiGoku123432/tree-sitter-cobol-upgrade` implies GONOPROXY+GONOSUMDB, so there is no proxy.golang.org or sum.golang.org lookup for the private path. Git over HTTPS needs a credential. The dev machine uses an `insteadOf` rewrite to the `github-personal` SSH alias | [VERIFIED: probe used GOPRIVATE + GIT_CONFIG insteadOf] |
| Does `docs/vendoring.md` block D-02? | No. Reason 1 was "a relative path *outside* the repository"; D-02 uses a module path. Reason 2 was go.sum participation; D-02 wants go.sum pinning | [VERIFIED: fork docs/vendoring.md §3] |
| Do transitive versions change? | `golang.org/x/sys v0.35.0` (preprocessor) < Gortex `v0.48.0`, so MVS keeps v0.48.0. `mattn/go-pointer` is already replaced in-tree. `go-tree-sitter v0.25.0` is identical. No new public modules | [VERIFIED: go.mod:259,283,359] |
| Does it work under `go test -race`? | Yes. Build info deps are present in test binaries, and `NewAnalyzer` returned nil error | [VERIFIED: probe `go test -race`] |
| Stale `go.work` | Currently **breaks every non-`GOWORK=off` go command** in the Gortex checkout: `go: cannot load module forest-shim/cobol listed in go.work file` | [VERIFIED: `go list -m` in repo] |

**Recommendation for the `go.work` discretion item:** delete the local `go.work` and `go.work.sum`. Both are git-ignored (`.gitignore:23-24`), so this is a user-local action. For fork iteration, use a temporary `-modfile` with a local-path replace instead of go.work. This avoids go.work silently overriding the go.mod pin, and it avoids the ToolID drift described under Pitfall 6. [VERIFIED for `-modfile` with a module-path replace; ASSUMED for a local-path replace]

**Behavioral floor:** `97ac9f1` is an ancestor of `f19029f` (402 commits). Gortex's existing `cobolprobe` IDMS/CICS/SQL parity gate passes against the shim: `go test ./internal/parser/forest/cobolprobe/ -enhanced-parser` → `--- PASS: TestErrorCascade`. [VERIFIED: `git merge-base --is-ancestor`; `-modfile` probe]

### Q4. D-03 embedded attestation — fork-side prerequisite spec

**How it attests today** [VERIFIED: fork preprocessor/analyze.go, pipeline.go:461-490, internal/tsadapter/parser.go, expected_identity.go]:
- `NewAnalyzer(repoRoot, catalog)` calls `validateSelectedShim(repoRoot)`. That function (a) reads `debug.ReadBuildInfo()` and requires the forest module in `Deps` plus `goTreeSitterVersion = "v0.25.0"`, then (b) calls `tsadapter.Attest(filepath.Join(repoRoot, "forest-shim", "cobol"))`.
- `Attest` computes `CompiledIdentity()`: a SHA-256 over the *linked* language (ABI, node kinds, fields, every parse state's lookahead/next-state, four fixed invented parses, `cobol.Info()`, and the embedded query sources). It also computes `CheckoutIdentity(shimRoot)`, a SHA-256 over the shim files on disk. It compares them with generated constants: `expectedCompiledLanguageID = "f97452e11a2b80b92acb7edf776131470bc2e1ffd7c077f4ade4ce3c3a47503d"` and `expectedCheckoutArtifactID = "1403095cbdf3a88f60e4fd03f1bbbc7aea3b6d43058996f23c786554c0c5e0ac"` [VERIFIED: expected_identity.go:6-7].
- Only the checkout hash needs files on disk. Measured cost: ~80–140 ms per `NewAnalyzer`.

**Minimal fork change (schedule as the first plan, fork repo, pushed before Gortex pins):**
```go
// preprocessor/internal/tsadapter/parser.go
// AttestEmbedded attests the linked COBOL language against the identity
// generated into this module, without reading a shim checkout. The checkout
// artifact identity is the generated constant: the shim bytes it describes are
// the module content go.sum already pins.
func AttestEmbedded() (Identity, error) {
	compiled, err := CompiledIdentity()
	if err != nil {
		return Identity{}, err
	}
	if compiled != expectedCompiledLanguageID {
		return Identity{}, errors.New("compiled forest language identity mismatch")
	}
	return Identity{CompiledLanguageID: compiled, CheckoutArtifactID: expectedCheckoutArtifactID}, nil
}

// preprocessor/analyze.go
// NewEmbeddedAnalyzer builds an Analyzer attested against the grammar identity
// compiled into this module instead of a fork checkout on disk. It fails when
// the linked forest COBOL grammar is not the one this module was generated
// against (for example the stock go-sitter-forest grammar).
func NewEmbeddedAnalyzer(catalog transform.Catalog) (Analyzer, error)
```
- `pipeline.go`: reuse the build-info half of `validateSelectedShim` (forest dep present, go-tree-sitter `v0.25.0`) and swap only the attestation call. Parameterize the attest step or split the function; that is the fork's choice.
- **Recommended, fork's call:** when `dependency.Replace != nil`, set `ForestModuleVersion` from the replacement (`Replace.Path@Replace.Version`). Under the replace, today's `dependencyVersion` returns `v1.9.1`, the same string the stock grammar would report [VERIFIED: probe `ForestModuleVersion:v1.9.1`]. That is misleading provenance, although `CompiledLanguageID` still distinguishes the grammars.
- Fork tests: embedded attestation succeeds, and `ToolIdentity` equals the one from `NewAnalyzer(repoRoot)` on an unmodified checkout. For the negative case, stub `languagePointer`, which is a package var.
- **Consequence for D-01:** the pinned commit becomes the fork commit that adds the constructor, not `f19029f`. Grammar and shim files are unchanged, so `CompiledLanguageID` stays `f97452e1…` and behavior is equivalent to `f19029f`. BASE-01/ROADMAP amendments should name the new commit and cite `f19029f` as the accepted v0.26.0 line.

### Q5. Provenance mapping (D-08/D-09) — proposed `prov_*` contract

**Separator: use `prov_` (underscore), not `prov.`** The Phase 1 projection encodes any byte outside `[A-Za-z0-9_]` as `x<hex>` [VERIFIED: internal/neo4jprojection/properties.go:154-169], so `prov.origin` would become Neo4j property `meta_provx2eorigin`, while `prov_origin` becomes `meta_prov_origin`. D-08 said "e.g. `prov.*`" and left names to discretion, so underscore is still namespaced and projects cleanly. **Also:** the projection silently drops keys whose normalized name contains `"password", "passwd", "secret", "token", "credential", "privatekey", "apikey", "authorization"` [VERIFIED: properties.go:171-180]. None of the keys below contain those substrings. Never name a provenance key `*_token*`.

| Handoff / source | Gortex mechanism | Key (Meta unless native) | Value type | On |
|---|---|---|---|---|
| repo / workspace / project | Native fields stamped by `applyRepoPrefix` | `RepoPrefix`, `WorkspaceID`, `ProjectID` (native; not duplicated in Meta because the extractor cannot know them) | string | all |
| relPath given to `Extract` | normalized repo-relative path | `prov_source_path` | string | all |
| `SourceID` | stable source identity | `prov_source_id` | 64-hex | all |
| `OriginalContentID` | revision, always (D-09) | `prov_revision_content_id` | 64-hex | all |
| git HEAD, when the file is clean | VCS revision (D-09) | `prov_vcs_commit` **or** `prov_vcs_commit_absence` ∈ {`not_a_git_worktree`, `no_head_commit`, `working_tree_differs_from_head`, `git_unavailable`} | string | all |
| `ToolID` | parser build identity | `prov_parser_tool_id` | 64-hex | all |
| `Tool.CompiledLanguageID` | BASE-03 grammar content identifier | `prov_parser_grammar_id` | 64-hex | all |
| build info `Replace.Path@Version` | parser commit (D-04 "commit") | `prov_parser_module` | string e.g. `…/forest-shim/cobol@v0.0.0-2026…-f19029f11cd0` | all |
| `ParseConfigurationID` | parse budget identity | `prov_parse_config_id` | 64-hex | all |
| `Ledger.ConfigurationID` | transform config identity (never pin in tests) | `prov_transform_config_id` | 64-hex | all |
| `SchemaVersion` | handoff schema | `prov_handoff_schema` | `"cobol-handoff-v1"` | all |
| Gortex extractor | extractor version (bump on mapping change; Phase 4 invalidation key) | `prov_extractor_version` | `"gortex-cobol-grammar/1"` | all |
| §6 class | evidence class, §6 vocabulary verbatim | `prov_evidence_class` | `"DETERMINISTIC"` | all |
| Edge origin tier vocabulary | extraction origin (reuses `OriginASTResolved = "ast_resolved"`; `Node.Origin` stays reserved for proxy nodes) | `prov_origin` | `"ast_resolved"` | all |
| Gortex-owned confidence | confidence (producer supplies none, per boundary rule D10) | `prov_confidence` | `1.0` (float64) | all |
| `Assessment.Grade` / `Reasons` / `PolicyVersion` | document trust (D-07) | `prov_document_grade`, `prov_document_grade_reasons`, `prov_grade_policy` | string, []string, string | all |
| `Observation.Kind` | observation kind | `prov_observation_kind` | `"program_definition"` | program |
| `Observation.Affected` | degraded marker (D-07) | `prov_affected` | bool | program |
| `Observation.Original` (0-based, verbatim) | exact original range (PROV-03) | `prov_start_row`, `prov_start_column`, `prov_end_row`, `prov_end_column`, `prov_start_byte`, `prov_end_byte`, `prov_range_exact` | int ×6, bool | program |
| same, native | tool compatibility | `StartLine = Start.Row+1`, `StartColumn = Start.Column`, `EndLine = End.Row+1`, `EndColumn = End.Column` (native) | int | program |
| containment path | ID symbol part | `QualName` (native), e.g. `OUTERPGM/INNERPGM` | string | program |
| schema convention | COBOL role | `cobol_kind` = `"program"` | string | program |

Notes:
- Confidence: `1.0` for every deterministic observation. Degradation is expressed through `prov_document_grade` and `prov_affected`, never by lowering confidence, because handoff §6 says "Confidence is not evidence". [CITED: docs/ai-enhanced-cobol-graph-handoff.md §6] The `EdgeDefines` file→program edge sets `Origin: graph.OriginASTResolved`, `Confidence: 1.0`, and `ConfidenceLabel: "EXTRACTED"`, matching `DefaultOriginFor`/`ConfidenceLabelFor` for `EdgeDefines`. [VERIFIED: edge.go:835-836, :1146-1168]
- Native `StartColumn`/`EndColumn` carry `json:"start_column,omitempty"`, so a 0 column disappears from JSON [VERIFIED: node.go:322-323]. That is why the verbatim `prov_*` range ints exist.
- **SQLite round-trip:** Meta is written through `encodeMeta`. The flat codec has exact type tags for string/bool/int/float64/[]string (`metaTagString`, `metaTagBool`, `metaTagInt`, `metaTagFloat64`, `metaTagStrSlice`) and falls back to JSON otherwise [VERIFIED: store_sqlite/meta_json.go:335-347, :409-422]. Only the listed promoted keys (`signature`, `visibility`, `doc`, … `section_text`) move into columns [VERIFIED: meta_json.go:771-793]. `prov_*` stays in the blob and round-trips exactly. Convert handoff `uint64` rows/columns to `int` before storing.
- **`get_symbol` visibility:** `detail:"brief"` returns `node.Brief()`, which carries no Meta. `detail:"full"` returns `"node": s.withAbsPath(ctx, node)`, whose JSON includes `meta` [VERIFIED: internal/mcp/tools_core.go:1741-1749; node.go:325 `Meta map[string]any json:"meta,omitempty"`]. **Acceptance must pass `detail=full`.**
- **Neo4j projection:** every Meta key becomes `meta_<encoded>` plus a `gortex_meta_key_map` entry [VERIFIED: properties.go:77-150]. A unit test on `projectNode` can prove it without Neo4j.
- The indexer also adds retrieval keys (`search_signature`, `search_qual_name`, `search_doc`) via `normalizeExtractionMetadata`. Assert the `prov_*` subset, not exact Meta equality. [VERIFIED: indexer/metadata_normalize.go:87-92]

### Q6. Identity (D-10)

- **Prefixing is the indexer's job.** The extractor emits `ID: relPath` for the file and `relPath + "::" + symbol` for programs, as regex `cobol.go:78,89` does. `applyRepoPrefix` rewrites `n.ID = intern.String(prefix + n.ID)`, `n.FilePath`, `n.RepoPrefix`, and edge endpoints, but skips `unresolved::` targets. It also stamps `WorkspaceID`/`ProjectID` [VERIFIED: internal/indexer/indexer.go:2255-2320]. Single-repo `Indexer` has an empty prefix, so no prefix is applied. `MultiIndexer` "Always stamp[s] the repo prefix, even when this is the only tracked repo" [VERIFIED: internal/indexer/multi.go:2395-2406]. The acceptance must therefore go through `MultiIndexer`.
- **Probe finding: `Observation.Parent` cannot build `::OUTER/INNER`.** The grammar is `start: repeat(program_definition)` and `program_definition: seq(identification_division, …, repeat($.end_program))` [VERIFIED: fork grammar.js:36-56]. For an invented nested fixture, both `program_definition` observations have `parent=0`. INNER's definition contains *both* `END PROGRAM INNERPGM.` and `END PROGRAM OUTERPGM.`, and OUTER's observed span stops where INNER begins. [VERIFIED: probe output]
- **Algorithm (deterministic, line-free):**
  1. Walk `Facts.Observations` in order. For each `program_definition` D, its name is the `program_name` whose parent is the `identification_division` whose parent is D. Take it from `Generated[span.Start:span.End]`, per the readiness doc [VERIFIED: COBOL-PARSER-READINESS.md:65].
  2. Precompute, for each name, how many `end_program` markers remain. The marker's name is its child `program_name`.
  3. On program D: pop every open program that has no remaining `END PROGRAM`, because a program without a marker cannot contain another. Then D's container path is the open stack. Push D.
  4. On `end_program` N: pop down to the matching N. If N is not open, record `prov_containment_unresolved=true` on the file node and do not guess.
  5. Symbol = containment path joined by `/`. For a same-parent duplicate, append `#k` (k ≥ 2) in observation order. The probe of two sibling `FIRSTPGM` programs needs `::FIRSTPGM` and `::FIRSTPGM#2`.
  - Line numbers never enter the ID (ID-02). Guard test: prepend blank/comment lines to a fixture and assert identical IDs.
- **Name normalization (open, see Open Question 4):** recommend keeping the ID component verbatim as it appears in `Generated`, with no case folding. Folding is a cross-program resolution concern for Phase 3.
- `/` after `::` is safe for existing helpers: `IDFile` splits at the first `::`, and `RepoPrefixOfID` splits at the first `/` before `::`. [VERIFIED: graph/overlay.go:369-377, graph/repo_prefix.go:23-28]

### Q7. Acceptance (D-11/D-12)

- **MCP:** `get_symbol`, registered at `internal/mcp/tools_core.go:1180` and handled by `handleGetSymbol` at :1709, requires `id`. Use `detail:"full"` (see Q5). Discoverability uses `search_symbols` with `query:"<PROGRAM>"`. Both are in the `core` preset [VERIFIED: internal/mcp/tool_presets.go:72].
- **CLI:** every `gortex query …` and `gortex node` verb relays to a running daemon (`requireDaemonTool` → `daemonRequiredErr` when absent) [VERIFIED: cmd/gortex/query.go:45-94; node.go]. The CLI verb that matches `get_symbol` is `gortex call get_symbol --arg id=<id> --arg detail=full`. `runCall` sends legacy names through `callDaemonTool = requireDaemonTool` with a JSON format [VERIFIED: cmd/gortex/call.go:32, :95-146]. By-name discovery uses `gortex query symbol <NAME>`, which relays to `search_symbols` [VERIFIED: query.go:216-225].
- **In-process real daemon harness exists:** `spinUpDaemonWithConfig` in `cmd/gortex/daemon_integration_test.go:44-128`. It sets `GORTEX_DAEMON_SOCKET`/`GORTEX_DAEMON_PIDFILE`, runs `languages.RegisterAll(reg)`, `indexer.NewMultiIndexer`, `mi.TrackRepoCtx`, `gortexmcp.NewServer`, `daemon.New(socket…)` + `newMCPDispatcher`, and `d.Listen(); go d.Serve()`, then MCP frames go through `daemon.DialTo`. It uses `graph.New()` (in-memory). **Phase 2 needs a variant on `store_sqlite.Open(temp)`.** `internal/indexer/multi_reconcile_sqlite_test.go` already runs `NewMultiIndexer(graph.Store(s), …)` on SQLite with a `GlobalConfig{Repos: {Path, Name: "repo"}}` config. [VERIFIED: both files, read via shell]
- **Phase 1 `graphfixture` is not the real path.** It hand-creates tables such as `CREATE TABLE nodes (owner_key …)` [VERIFIED: internal/testutil/graphfixture/fixture.go:52-56]. Reuse only its `Canonical(path)`: its non-`owner_key` branch reads the real store schema and can assert SQLite rows independently of Gortex APIs.
- **AddBatch path (TRACE-04):** a first index on a `BulkLoader` store below the shadow thresholds swaps to an in-memory shadow and dumps through `BulkLoad` ("dump the final state to the disk backend via one BulkLoad cycle") [VERIFIED: indexer.go:2660-2690]. `GORTEX_SHADOW_MAX_FILES=0` "disables the shadow entirely (always run against the disk store directly)" [VERIFIED: indexer/shadow_threshold.go:38-51], and the existing SQLite test uses it (`t.Setenv("GORTEX_SHADOW_MAX_FILES", "0") // exercise the direct SQLite cold path`). Direct writes go through `parseGraphBatch.flushLocked` → `b.store.AddBatch(nodes, edges)` [VERIFIED: indexer/parse_graph_batch.go:122-131]. To *prove* AddBatch, wrap the store in a struct embedding `*store_sqlite.Store` that overrides `AddBatch` to record IDs. Embedding a pointer keeps `BulkLoader`/`FileMtimeWriter`/`SymbolSearcher` assertions working; check for concrete-type assertions in a Wave-0 spike. Then close the store, **reopen**, and assert `GetNode(id)` Meta, proving persistence rather than memory.
- **AI off (TRACE-07):** the in-process `gortexmcp.NewServer(...)` harness never sets `llmService`, so `ask` stays unregistered. Also clear `GORTEX_LLM_PROVIDER` with `t.Setenv` and assert `ask` is absent from `tools/list`. No embedder is set on the indexer. No Neo4j profile or config is created, and nothing in the index path dual-writes.
- **Parser-mismatch negative (D-14):** use two layers.
  1. **In-process (always runs):** the extractor holds its approved grammar ID as an unexported struct field, set from the constant `cobolApprovedGrammarID = "f97452e1…"`. A test constructs it with a wrong value and indexes the tracer fixture. Assert that `IndexResult` errors name the mismatch, `GetNode(programID) == nil`, and the file stays visible as a parse-failed skip node, because the indexer records `parseFailedFiles` for a nil result with an error [VERIFIED: indexer.go:3595-3606].
  2. **Build-level (BASE-02 proof, CI ubuntu only):** copy `go.mod`/`go.sum` to a temp modfile, run `go mod edit -modfile=… -dropreplace github.com/alexaandru/go-sitter-forest/cobol`, and run the grammar-pin test. It must fail with `compiled forest language identity mismatch` [VERIFIED: stock-grammar probe printed `NewAnalyzer err: compiled forest language identity mismatch`].

### Q8. CI (D-13) — every Go-compiling job needs the credential, not just `ci.yml` test

`ci.yml` jobs that compile or resolve modules: `test` (ubuntu/macos/windows), `build-linux-static`, `lint` (golangci-lint + `go mod tidy` diff), `build-onnx`, `benchmark` [VERIFIED: .github/workflows/ci.yml]. Other workflows that also run `go build`/`go test`: `init-smoke.yml`, `skill-drift.yml`, `security.yml` (govulncheck), `bench-arm.yml`, `release.yml`, `publish-claude-plugin.yml` (setup-go). [VERIFIED: grep of workflows] Unmodified jobs fail at module download once go.mod requires the private module.

Per-job step (bash shell so the Windows runner works too):
```yaml
env:
  GOPRIVATE: github.com/MuiGoku123432/tree-sitter-cobol-upgrade
# … in each job, after actions/checkout and before any go command:
      - name: Configure read access to the private COBOL parser module
        shell: bash
        env:
          COBOL_PARSER_READ_PAT: ${{ secrets.COBOL_PARSER_READ_PAT }}
        run: |
          test -n "$COBOL_PARSER_READ_PAT" || { echo "::error::COBOL_PARSER_READ_PAT is not configured"; exit 1; }
          git config --global \
            url."https://x-access-token:${COBOL_PARSER_READ_PAT}@github.com/MuiGoku123432/tree-sitter-cobol-upgrade".insteadOf \
            "https://github.com/MuiGoku123432/tree-sitter-cobol-upgrade"
```
- **User action:** create a fine-grained PAT scoped to *only* `tree-sitter-cobol-upgrade` with Contents: Read-only, or a read-only deploy key, and store it as a repo secret. [ASSUMED: GitHub fine-grained PAT scoping] Prefer the PAT because it is cross-platform HTTPS; a deploy key needs SSH agent setup on Windows.
- Windows test job: see Pitfall 1. The grammar tests are `!windows`, so the Windows job still resolves modules (needs the credential) but compiles the stub.
- Add a ubuntu-only "stock grammar must fail" step (Q7 layer 2).
- **Actions cache exposure:** `actions/setup-go` caches the module cache, which will then hold the private source. [ASSUMED] Caches created on the default branch are restorable by PR runs, including fork PRs. Set `cache: false` on setup-go in jobs that download the private module, or accept the risk explicitly (see Security).

### Q9. Existing tests whose behavior changes

| Test | Change under shim + swap | Evidence |
|---|---|---|
| `internal/parser/forest/dump_kinds_test.go` (`TestDumpGrammarKinds/cobol`) | Research helper with no assertions beyond parse success. Passes under the shim (prints `ERROR ×1, comment_entry ×5, start ×1`) | [VERIFIED: `-modfile` probe] |
| `internal/parser/forest/cobolprobe` `TestHypothesis*` | Pass under the shim | [VERIFIED: probe] |
| `cobolprobe` `TestErrorCascade` | Skips without `-enhanced-parser`. With the flag it passes under the shim; after D-02 the shim is always linked | [VERIFIED: probe] |
| `cobolprobe/README.md` | Describes the go.work activation and names `97ac9f1`. It also embeds absolute developer-home and corpus paths, an existing privacy leak in a public repo. Update it to the go.mod pin and scrub the paths | [VERIFIED: README read] |
| `internal/parser/languages/cobol_test.go` (9 regex tests) | Unaffected if `cobol.go` is kept, since they call `NewCobolExtractor()` directly | [VERIFIED: file read] |
| `internal/mcp/parse_gate_test.go` (asserts `parseErrorCount("cobol", …)` returns not-ok) | Unaffected: `astquery.DefaultLanguageResolver("cobol")` still has no COBOL | [VERIFIED: parse_gate.go:65-72] |
| `internal/cfg/cfg_test.go:701-705` ("cobol must not be supported") | Unaffected | [VERIFIED] |
| `ci.yml` static-link step `go test -tags netgo,osusergo,static_link ./internal/parser/languages/` | Now also runs the grammar extractor tests; needs the credential | [VERIFIED: ci.yml] |
| Windows `go test ./...` | **Breaks at compile** if `languages` imports `preprocessor` unconditionally | [VERIFIED: `GOOS=windows go list` → "build constraints exclude all Go files in …/internal/securefs"] |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|---|---|---|---|
| Grammar build + attestation | Producer module (fork `preprocessor`, `forest-shim/cobol`) | Gortex go.mod pin | Boundary rule D10: producer emits facts; Gortex owns graph conversion |
| Handoff → native nodes/edges + `prov_*` | Parser tier (`internal/parser/languages`) | — | The extractor has the bytes and handoff; no repo context |
| Repo prefix, workspace/project scope | Indexer (`applyRepoPrefix`) | — | Existing convention; the extractor must not prefix |
| VCS revision stamp | Indexer (`applyCoverageDomains` hook) | `internal/gitstate` | Only the indexer knows `rootPath`; one hook covers bulk and incremental |
| Persistence | Database/Storage (`store_sqlite.AddBatch`) | — | SQLite is sole authority (STORAGE-ADR) |
| Retrieval | API/Backend (MCP `get_symbol`/`search_symbols`) | CLI relay (`gortex call`, `gortex query symbol`) | Two front doors, one handler |
| Projection of `prov_*` | Phase 1 `neo4jprojection` (manual) | — | No dual-write; verify by unit test only |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---|---|---|---|
| `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor` | `v0.0.0-<ts>-<commit adding NewEmbeddedAnalyzer>` (today `v0.0.0-20260930153412-f19029f11cd0`) | Transform, budgeted parse, facts, grade, `cobol-handoff-v1` | The approved producer (D-01) [VERIFIED: probe] |
| `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol` (replaces `github.com/alexaandru/go-sitter-forest/cobol v1.9.1`) | same pseudo-version | Enhanced grammar under the upstream module path | D-02 [VERIFIED: probe] |
| `github.com/tree-sitter/go-tree-sitter` | v0.25.0 (unchanged) | cgo runtime; the fork requires exactly this | [VERIFIED: go.mod:259; pipeline.go:32] |
| `internal/gitstate.SampleDirty` (in-tree) | — | HEAD commit + dirty set for D-09 | Existing helper; don't hand-roll git status parsing [VERIFIED: gitstate/dirty.go:268] |

### Supporting
| Library | Purpose | When to Use |
|---|---|---|
| `store_sqlite.Open`, `indexer.NewMultiIndexer` | Real path in tests | Acceptance |
| `graphfixture.Canonical` | Store-independent row check | Optional SQLite assertion |
| `transform.NewCatalog(nil)` | Empty INCLUDE catalog | Phase 2 tracer: "`-INC` rewrites are symbolic… catalog's ID does feed the configuration identity" [VERIFIED: readiness doc §2] |

**Installation:**
```bash
export GOPRIVATE=github.com/MuiGoku123432/tree-sitter-cobol-upgrade
GOWORK=off go get github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor@<commit>
GOWORK=off go mod edit -replace github.com/alexaandru/go-sitter-forest/cobol=github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol@<same pseudo-version>
GOWORK=off go mod tidy
```

## Package Legitimacy Audit

The `gsd-tools package-legitimacy` seam supports only npm/pypi/crates (`Usage: … --ecosystem <npm|pypi|crates>`), so Go modules were verified by direct provenance instead.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---|---|---|---|---|---|---|
| `…/tree-sitter-cobol-upgrade/preprocessor` | Go (private, direct VCS) | branch head 2026-09-30 | n/a (private) | github.com/MuiGoku123432/tree-sitter-cobol-upgrade (user-owned) | OK — user's own repo, commit verified via `git ls-remote`, h1 pinned in go.sum | Approved (new dependency, locked by D-02) |
| `…/tree-sitter-cobol-upgrade/forest-shim/cobol` | Go (private) | same | n/a | same | OK — same | Approved (replace, locked by D-02) |

**Packages removed due to [SLOP] verdict:** none. **Packages flagged [SUS]:** none. No new public modules enter the graph: transitive `golang.org/x/sys` resolves to Gortex's existing `v0.48.0`.

## Architecture Patterns

### System Architecture Diagram

```
 COBOL file bytes (.cbl/.cob)          .cpy (copybook)
        │                                   │
        ▼                                   ▼
 indexer parse worker ──► registry.GetByLanguage("cobol") ──► CobolGrammarExtractor.Extract(relPath, src)
                                                              │
                     first call: sync.Once ─► preprocessor.NewEmbeddedAnalyzer(NewCatalog(nil))
                                              │ attest fails ─► every Extract returns error (D-05)
                                              ▼
                     semaphore(1) ─► an.Analyze(ctx, NewSourceID(relPath), src) ─► handoff.Handoff
                                              │ h.Tool.CompiledLanguageID != approved ─► error (BASE-01/02)
                                              ▼
                     map: file node + program nodes (END PROGRAM stack ids) + EdgeDefines + prov_* keys
                     (.cpy: file node only, prov_analysis_absence; no Analyze)
                                              ▼
 indexer: stampParseErrors / normalizeExtractionMetadata ─► applyCoverageDomains (+ VCS revision stamp)
                                              ▼
                     applyRepoPrefix (IDs, FilePath, RepoPrefix, WorkspaceID, ProjectID)
                                              ▼
       shadow disabled? ── yes ─► parseGraphBatch ─► store_sqlite.AddBatch (TRACE-04)
                        └─ no ──► in-memory shadow ─► BulkLoad (daemon cold path)
                                              ▼
                              SQLite nodes.meta blob (flat codec)
                                              ▼
       MCP get_symbol(detail=full) / search_symbols   ◄──  gortex call get_symbol / gortex query symbol (daemon relay)
                                              ▼
                  (manual, Phase 1) neo4jprojection.projectNode ─► meta_prov_* properties
```

### Recommended Project Structure
```
internal/parser/languages/
├── cobol_grammar.go          # //go:build !windows — handoff → ExtractionResult, IDs, prov_* (target < 300 lines)
├── cobol_grammar_windows.go  # //go:build windows — same constructor; Extract returns "enhanced COBOL parser unavailable on windows"
├── cobol_grammar_test.go     # //go:build !windows — unit + guard fixtures (invented COBOL, inline strings)
├── cobol.go / cobol_test.go  # untouched (upstream-owned); unregistered
└── register.go               # line 119: NewCobolExtractor() → NewCobolGrammarExtractor()
internal/indexer/
├── source_revision.go        # VCS revision stamp (only touches nodes that carry prov_revision_content_id)
└── indexer.go                # one call added inside applyCoverageDomains
cmd/gortex/
└── cobol_tracer_test.go      # //go:build !windows — SQLite-backed daemon acceptance
```

### Pattern 1: Lazy, fail-closed analyzer (D-03/D-05)
Attest once on first use rather than in `RegisterAll`. `RegisterAll` runs for every `gortex` command, and attestation costs ~80–140 ms. Cache the error, so every later `Extract` returns it.
```go
// Source: pattern derived from fork analyze.go API (NewAnalyzer/Analyze) + Q4 spec
type CobolGrammarExtractor struct {
	approvedGrammarID string // cobolApprovedGrammarID in production; tests inject a wrong value
	once              sync.Once
	analyzer          preprocessor.Analyzer
	initErr           error
}
```

### Pattern 2: Gortex-side approved grammar pin (BASE-01/03)
The embedded attestation proves "linked grammar == what this preprocessor module expects". The Gortex constant proves "this preprocessor module is the approved one". Without the constant, a go.mod bump to another fork commit would silently change the baseline. Compare `h.Tool.CompiledLanguageID` with `cobolApprovedGrammarID` on every handoff, and fail on mismatch. The constant equals today's `expectedCompiledLanguageID` (`f97452e1…`). [VERIFIED: expected_identity.go:6; probe tool output]

### Pattern 3: Serialize parses (readiness §6)
Use a package-level `chan struct{}` of capacity 1 around `Analyze`. This follows "Parse one file at a time, or with low concurrency, on 16 GB hosts" [VERIFIED: readiness doc §6]. Other languages keep full parallelism. Revisit in Phase 3/6.

### Pattern 4: VCS revision stamp (D-09) — MEDIUM confidence design
The `Extractor` interface is `Extract(filePath string, src []byte)` with a relative path and no repo context [VERIFIED: internal/parser/extractor.go:12-16]. The indexer is the only place that knows `rootPath`. `applyCoverageDomains(relPath, lang, src, result)` is "Called from both the bulk index worker pool (IndexCtx) and the incremental indexFile path" [VERIFIED: indexer.go:1655-1664; call sites file_delta.go:161, indexer.go:3625, :4658].
- Add one call there, `idx.stampSourceRevision(relPath, result)`. It returns immediately unless some node has `prov_revision_content_id`, so other languages are untouched.
- Take the snapshot with `gitstate.SampleDirty(ctx, idx.rootPath)`, which returns `HeadCommit` and `Entries` including `DirtyUntracked`/`DirtyModified` [VERIFIED: gitstate/dirty.go]. Cache it once per bulk pass; recompute per incremental call.
- Map porcelain paths, which are relative to the worktree top-level, to `relPath`, which is relative to `rootPath`.
- Clean path → `prov_vcs_commit = HeadCommit`. Otherwise set `prov_vcs_commit_absence` from {`not_a_git_worktree`, `git_unavailable`, `no_head_commit`, `working_tree_differs_from_head`}.
- Caveat: `idx.transforms.run(relPath, src)` (BOM strip, etc.) runs before extraction [VERIFIED: indexer.go:3570-3572], so `prov_revision_content_id` hashes the bytes that were parsed, which may differ from the disk bytes.

### Anti-Patterns to Avoid
- **Walking raw tree-sitter output of unpreprocessed source.** Readiness doc: "Do not walk raw tree-sitter output of unpreprocessed source."
- **Using `Observation.Parent` for program nesting.** It is flat by grammar construction (Q6).
- **Pinning `ledger.configuration_id` / `gaps[].id` in tests.** They "change on every producer transform-config bump" [VERIFIED: readiness doc §4]. Assert shape (64-hex) instead.
- **Pinning `ToolID` in tests.** It folds `ForestModuleVersion`, which differs between go.work (`(devel)`) and go.mod builds. Pin `CompiledLanguageID` instead.
- **Overloading `Node.Origin`.** It marks cross-daemon proxy nodes: "`""` on every locally-indexed node" [VERIFIED: node.go:349-355].
- **Storing `Generated`, `Ledger.Entries`, or source snippets in Meta.** Store identities and codes only (size and estate-content leakage).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| Grammar identity/attestation | Own hash of parser.c | `NewEmbeddedAnalyzer` + `Tool.CompiledLanguageID` | Producer fingerprints linked language, states, queries |
| Original coordinates | Recomputing rows/cols from bytes | `Observation.Original` (`sourcemap.Projection`) | Map already validated by `handoff.Validate` |
| Source/content IDs | sha256 in Gortex | `h.SourceID`, `h.OriginalContentID` | Domain-separated producer identities |
| Git state | Parsing `git status` | `internal/gitstate.SampleDirty` | Existing, NUL-safe porcelain parser |
| Daemon test harness | New IPC fake | `spinUpDaemonWithConfig` pattern + SQLite store | Exercises real dispatcher/relay |
| Meta encoding / Neo4j keys | Custom serialization | `encodeMeta` / `appendMetadata` | Exact type round-trip; collision-aware key map |

## Runtime State Inventory

The extractor swap and baseline change leave state behind that no repo edit fixes.

| Category | Items Found | Action Required |
|---|---|---|
| Stored data | Existing SQLite stores (`.gortex/` or daemon store) hold regex-era COBOL nodes (`file::PARA`, `-DIVISION`, `-SECTION`, `unresolved::import::*`) | Full reindex / `gortex track` again after upgrade. The incremental path only reindexes changed files, so unchanged COBOL files keep regex nodes until touched. Document it; Phase 4 owns invalidation keyed on `prov_extractor_version` |
| Live service config | A Neo4j database previously projected (Phase 1) contains regex-era COBOL nodes | Re-run the manual projection after reindex; no automatic sync exists (by design) |
| OS-registered state | A running `gortex` daemon binary built before the swap keeps the regex extractor | Restart the daemon after rebuild (`gortex daemon` restart) |
| Secrets/env vars | New: `GOPRIVATE` (dev machine, CI env), CI secret `COBOL_PARSER_READ_PAT`, dev git `insteadOf` rewrite | User provisions; plan surfaces as `checkpoint:human-action` |
| Build artifacts | Stale git-ignored `go.work`/`go.work.sum` in repo root (breaks all non-`GOWORK=off` commands); module cache gains private sources; the fork's own `go.work` is unaffected | Delete Gortex's local `go.work`/`go.work.sum` (user-local, not a commit) |

## Common Pitfalls

### Pitfall 1: Windows build break
**What goes wrong:** `go build`/`go test` on Windows fails: "build constraints exclude all Go files in …/preprocessor/internal/securefs".
**Why:** securefs files are `//go:build darwin || linux` only [VERIFIED: probe].
**How to avoid:** build-tag the extractor and its tests `!windows`. The Windows stub keeps the same constructor name and fails every Extract (D-05). Alternatively, the fork adds Windows support as a second producer change.
**Warning signs:** the Windows CI shard is red at compile.

### Pitfall 2: Cold index bypasses AddBatch
**What goes wrong:** the test passes but the program was persisted via `BulkLoad`, so TRACE-04 is unproven.
**How to avoid:** `t.Setenv("GORTEX_SHADOW_MAX_FILES", "0")` plus the AddBatch-recording wrapper. Optionally add a second subtest on the default path to prove Meta survives BulkLoad too.

### Pitfall 3: `get_symbol` brief hides provenance
**What goes wrong:** the `detail` default is `"brief"`, and `Brief()` has no Meta.
**How to avoid:** always send `detail=full` and assert `node.meta.prov_*`.

### Pitfall 4: Nested program range truncation
**What goes wrong:** the grammar ends OUTER's `program_definition` where INNER begins, so the persisted range for OUTER does not reach `END PROGRAM OUTER`.
**How to avoid:** persist the parser-observed range (PROV-03 "from the parser"). Report the limitation to the fork, and don't synthesize a range. Assert the nested fixture by ID, not by OUTER's end line.

### Pitfall 5: Copybooks graded red
**What goes wrong:** `Analyze` on a standalone copybook gives grade `red` with reasons `[missing-required-anchor division-scale-recovery error-node affected-capture]` [VERIFIED: probe], which marks every copybook "red" for a meaningless reason.
**How to avoid (recommend, confirm):** for `.cpy`/`.CPY`, emit the file node only with `prov_analysis_absence="copybook_standalone_analysis_unsupported"`, and skip Analyze.

### Pitfall 6: go.work silently overriding the pin
**What goes wrong:** with a go.work `use` of the fork, the build links fork-local sources, and `ForestModuleVersion` becomes `(devel)`, which changes ToolID.
**How to avoid:** delete go.work; always run verify commands with `GOWORK=off`; never pin ToolID.

### Pitfall 7: Neo4j secret filter and key encoding
**What goes wrong:** a key containing `token`/`secret`/etc. is silently dropped from the projection, and dots become `x2e`.
**How to avoid:** use the `prov_` underscore keys from Q5, and add a projection unit test asserting zero `Secret` warnings.

### Pitfall 8: Private module fetch without GOPRIVATE
**What goes wrong:** `go` queries proxy.golang.org and sum.golang.org, which fails, and the private module path is sent to public services.
**How to avoid:** set GOPRIVATE in the dev shell (`go env -w GOPRIVATE=…`) and in CI `env`.

### Pitfall 9: Crash-isolation subprocess route
**What goes wrong:** with `index.crash_isolation` / `GORTEX_PARSER_ISOLATION=1`, extraction runs in worker subprocesses [VERIFIED: config.go:687-695; off by default]. Attestation then runs per worker, and Meta crosses a serialization boundary that was not verified in this session.
**How to avoid:** keep acceptance on the default in-process route. Note it as a Phase 6 check.

## Code Examples

### Program-name lookup from observations
```go
// Source: probe this session against cobol-handoff-v1 (kinds program_definition,
// identification_division, program_name, end_program)
name := func(h handoff.Handoff, def int) (string, bool) {
	for i, o := range h.Facts.Observations {
		if o.Kind != "identification_division" || o.Parent != def {
			continue
		}
		for _, c := range h.Facts.Observations[i+1:] {
			if c.Kind == "program_name" && c.Parent == i {
				return string(h.Generated[c.Generated.Start:c.Generated.End]), true
			}
		}
	}
	return "", false
}
```

### Parser module identity from build info (D-04 "commit")
```go
// Source: probe — debug.ReadBuildInfo in go test -race showed
// dep github.com/alexaandru/go-sitter-forest/cobol v1.9.1 replace=&{Path:…/forest-shim/cobol Version:v0.0.0-20260930153412-f19029f11cd0 …}
func cobolParserModule() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, d := range info.Deps {
		if d.Path == "github.com/alexaandru/go-sitter-forest/cobol" && d.Replace != nil {
			return d.Replace.Path + "@" + d.Replace.Version
		}
	}
	return ""
}
```

### Invented fixtures that produced the probe results
```text
tracer (green):  IDENTIFICATION DIVISION. / PROGRAM-ID. DEMOPGM. / PROCEDURE DIVISION. / MAIN-PARA. / STOP RUN.
nested (green):  OUTERPGM … IDENTIFICATION DIVISION. PROGRAM-ID. INNERPGM. … END PROGRAM INNERPGM. END PROGRAM OUTERPGM.
duplicate:       two sibling FIRSTPGM programs each closed by END PROGRAM FIRSTPGM.
amber:           DATA DIVISION / WORKING-STORAGE SECTION / 01 WS-A PIC X(   → grade=amber reasons=[missing-node]
seq-area:        cols 1-6 digits and 73-80 "SEQ0000n" → green, name SEQPGM
```
All fixtures use 7-column fixed format (lines start with 7 spaces or a 6-digit sequence plus a space).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| Regex `CobolExtractor` | Grammar handoff `cobol-handoff-v1` | Phase 2 | Program facts gain exact ranges and provenance; other COBOL facts pause until Phase 3 |
| go.work local pipe to fork | go.mod require + replace pseudo-version | Phase 2 (D-02) | Reproducible under `GOWORK=off`; CI needs a credential |
| `NewAnalyzer(repoRoot)` checkout attestation | `NewEmbeddedAnalyzer` (fork change) | Phase 2 prerequisite | No fork checkout at runtime |

**Deprecated/outdated:** `cobolprobe/README.md` go.work activation instructions; REQUIREMENTS "Out of Scope: Changing parser baseline `97ac9f1`…" (line 193); PROJECT.md:96/118/122 baseline statements. All should be amended alongside BASE-01 (D-01).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|---|---|---|
| A1 | A fine-grained PAT can be scoped to one repo with Contents: Read-only and used as the HTTPS password with any username | Q8 | CI auth step needs a different form (deploy key) |
| A2 | Actions caches from the default branch are restorable by fork PR runs, exposing cached private module source | Q8/Security | If wrong, `cache: false` is unnecessary but harmless |
| A3 | A `-modfile` local-path replace works as the developer fork-iteration override | Q3 | Dev override needs go.work instead |
| A4 | Program names should not be case-folded in IDs in Phase 2 | Q6 | ID churn in Phase 3 if resolution requires folding |
| A5 | Embedding `*store_sqlite.Store` in a test wrapper doesn't trip concrete-type assertions in the MultiIndexer/checkout lifecycle | Q7 | Need a log-observer or different proof of AddBatch |
| A6 | The SQLite-backed variant of `spinUpDaemonWithConfig` works with `realController` + checkout lifecycle on a non-git temp dir | Q7 | Wave-0 spike may need a git-initialized fixture dir |
| A7 | Skipping Analyze for `.cpy` is preferable to recording a red grade | Pitfall 5 | Copybook provenance semantics differ from user intent |
| A8 | Gortex release binaries embedding the fork's `grammar.json` + `*.scm` (shim `//go:embed grammar.json *.scm`) is acceptable to the user | Security | Private grammar work published in public release assets |

## Open Questions (items needing user confirmation)

1. **Temporary regression (D-06):** accept loss of COBOL paragraphs/sections/divisions/CALL/COPY/PERFORM edges from the Phase 2 swap until Phase 3? *Recommendation: yes (Option A).*
2. **Regex code disposition:** keep `cobol.go`/`cobol_test.go` untouched and unregistered (fork hygiene, zero upstream conflict), or delete them (no dead code)? *Recommendation: keep.*
3. **Windows:** accept a Windows stub that fails COBOL extraction (D-05-consistent), or require a second fork change for Windows securefs? *Recommendation: stub now.*
4. **Name normalization in IDs:** keep verbatim (recommended) vs uppercase-fold.
5. **Copybooks:** skip analysis with an absence reason (recommended) vs record a red grade.
6. **VCS stamp placement:** approve the small indexer hook in `applyCoverageDomains` (needed because extractors have no repo context). The fallback, content ID plus a fixed absence reason in Phase 2, would under-deliver D-09.
7. **Public release binaries:** the shim embeds `grammar.json` and query `.scm` sources into every Gortex binary; `release.yml` would publish them. Confirm acceptable or gate releases.
8. **CI scope:** credential in all Go-building workflows (recommended), or only `ci.yml` with the others disabled/allowed to fail?

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| Go toolchain | build/test | ✓ | go1.27.1 darwin/arm64 (go.mod `go 1.27.0`) | — |
| C compiler (cgo) | tree-sitter shim | ✓ (builds succeeded) | — | — |
| git + read access to private fork | module fetch | ✓ via `github-personal` SSH alias + `insteadOf` | — | — |
| `gh` CLI | secret provisioning | ✓ but signed in to a work account without evident access to the private fork | — | User provisions secret in the GitHub UI |
| Fork checkout | producer change (D-03) | ✓ `$COBOL_UPGRADE_ROOT`, branch up to date with origin | head `f19029f` | — |
| Neo4j | not required | not needed | — | — |

**Missing dependencies with no fallback:** CI secret `COBOL_PARSER_READ_PAT` (user action). **Missing with fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|---|---|
| Framework | Go `testing` + testify v1.11.1 |
| Config file | none (package tests) |
| Quick run command | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run 'TestCobolGrammar'` |
| Full suite command | `GOWORK=off go test -race ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | Failing signal | File Exists? |
|---|---|---|---|---|---|
| BASE-01, BASE-03 | Handoff grammar ID equals pinned approved ID; `prov_parser_grammar_id` present | unit | `GOWORK=off go test -race -count=1 ./internal/parser/languages/ -run TestCobolGrammar_ApprovedGrammarPinned` | mismatch error / missing key | ❌ Wave 0 |
| BASE-02 | Stock grammar build fails closed | CI step (modfile dropreplace) | `cp go.mod $T/stock.mod && cp go.sum $T/stock.sum && GOWORK=off go mod edit -modfile=$T/stock.mod -dropreplace github.com/alexaandru/go-sitter-forest/cobol && ! GOWORK=off go test -modfile=$T/stock.mod -count=1 ./internal/parser/languages/ -run TestCobolGrammar_ApprovedGrammarPinned` | the test **passing** under stock = failure | ❌ Wave 0 |
| BASE-02 | Wrong approved ID → extraction error, no program node, file visible | unit | `… -run TestCobolGrammar_GrammarMismatchFails` | node present or no error | ❌ |
| TRACE-01/02, ID-01 | Program `KindFunction` + `EdgeDefines` from file, `cobol_kind=program` | unit | `… -run TestCobolGrammar_ProgramNode` | wrong kind/ID/edge | ❌ |
| ID-02 | Nested `::OUTERPGM/INNERPGM`, duplicate `#2`, line-shift invariance | unit | `… -run 'TestCobolGrammar_(NestedIDs\|DuplicateOrdinal\|IDIgnoresLines)'` | ID differs | ❌ |
| D-07 | Amber fixture emitted with `prov_document_grade=amber`, reasons, `prov_affected` | unit | `… -run TestCobolGrammar_DegradedAmber` | node dropped / key missing | ❌ |
| PROV-03 | Verbatim 0-based range + native 1-based lines | unit | `… -run TestCobolGrammar_ProgramNode` | range mismatch | ❌ |
| PROV-04 | VCS commit on clean git file; absence reason on non-git / modified | unit (temp git repo) | `GOWORK=off go test -race -count=1 ./internal/indexer/ -run TestSourceRevisionStamp` | missing commit / absence | ❌ |
| TRACE-03/04, PROV-01/02/05/06 | Real MultiIndexer → SQLite AddBatch → reopen → Meta intact, prefixed ID, scope fields | integration | `GOWORK=off go test -race -count=1 ./cmd/gortex/ -run TestCobolTracer_PersistsThroughAddBatch` | AddBatch never saw ID / key missing after reopen | ❌ |
| TRACE-05 | `gortex call get_symbol --arg id=… --arg detail=full` returns node + prov keys | integration | `… ./cmd/gortex/ -run TestCobolTracer_CLIGetSymbol` | not found / key missing | ❌ |
| TRACE-06, D-11 | MCP `get_symbol` full + `search_symbols` by name | integration | `… ./cmd/gortex/ -run TestCobolTracer_MCP` | not found | ❌ |
| TRACE-07 | No `ask` tool, no LLM env, no Neo4j | integration | `… ./cmd/gortex/ -run TestCobolTracer_NoAINoNeo4j` | `ask` listed | ❌ |
| D-08 | `prov_*` project as `meta_prov_*`, zero secret drops | unit | `GOWORK=off go test -race -count=1 ./internal/neo4jprojection/ -run TestProjectNode_CobolProvenance` | encoded key / Secret>0 | ❌ |
| Regression | Existing grammar probes still pass | unit | `GOWORK=off go test -race -count=1 ./internal/parser/forest/... -enhanced-parser -run 'TestErrorCascade\|TestHypothesis\|TestDumpGrammarKinds'` | FAIL | ✅ |

Note: `-enhanced-parser` is a flag of the cobolprobe package only; run it as `GOWORK=off go test -race ./internal/parser/forest/cobolprobe/ -enhanced-parser` separately from `./internal/parser/forest/`.

### Sampling Rate
- **Per task commit:** the quick command plus the package touched.
- **Per wave merge:** `GOWORK=off go test -race -count=1 ./internal/parser/... ./internal/indexer/ ./internal/neo4jprojection/ ./cmd/gortex/ -run 'Cobol|SourceRevision|ProjectNode'`
- **Phase gate:** `GOWORK=off go build -o gortex ./cmd/gortex/ && GOWORK=off go test -race ./...` green, `GOWORK=off go mod tidy` diff-clean, the stock-grammar negative step failing as expected, and CI green on all three OS shards.

### Wave 0 Gaps
- [ ] Fork: `NewEmbeddedAnalyzer` + `AttestEmbedded` + tests, pushed (producer prerequisite)
- [ ] Spike: SQLite-backed `spinUpDaemonWithConfig` variant + AddBatch-recording wrapper (A5, A6)
- [ ] `internal/parser/languages/cobol_grammar_test.go` fixtures (invented)
- [ ] `cmd/gortex/cobol_tracer_test.go`
- [ ] `internal/indexer/source_revision_test.go` (temp `git init` repo)
- [ ] `internal/neo4jprojection` provenance projection test

## Security Domain

### Applicable ASVS Categories (Level 1)

| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | no (no user auth added) | — |
| V3 Session Management | no | — |
| V4 Access Control | yes (retrieval) | Existing `resolveScope` / `resolvedScopeAllowsNode` in `handleGetSymbol`; no bypass |
| V5 Input Validation | yes | Untrusted COBOL bytes go only through `Analyze` (transform, budgeted parse, `handoff.Validate`); `NewSourceID` rejects absolute/non-local locators |
| V6 Cryptography | yes (identities) | Producer SHA-256 IDs; never hand-roll |
| V7 Errors/Logging | yes | Producer errors "never carry source bytes or offsets"; wrap with `%w`, don't log source |
| V10/V14 Config & Dependencies | yes | go.sum h1 pinning, GOPRIVATE, least-privilege CI credential |

### Known Threat Patterns for this phase

| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| CI credential over-scope or leak | Information disclosure / Elevation | Fine-grained PAT, single repo, Contents read-only; pass via step `env`, never `echo`; never use `pull_request_target`; rotate on exposure |
| Private module source restored from Actions cache by fork PRs (A2) | Information disclosure | `cache: false` on setup-go in jobs that download it, or accept explicitly |
| Private module path sent to public proxy/sumdb | Information disclosure | GOPRIVATE in dev and CI |
| Supply-chain swap of fork content (force-push) | Tampering | go.sum h1 pins; build fails closed on mismatch; Gortex approved-grammar pin |
| Silent stock-grammar fallback | Tampering / Repudiation | Embedded attestation + approved ID pin; extraction error, never regex fallback |
| Estate data in public repo via fixtures/goldens | Information disclosure | Invented COBOL only; assert structure/kinds/ranges, not estate names; scrub `cobolprobe/README.md` absolute paths |
| Estate content in Meta/provenance | Information disclosure | `prov_*` holds hashes, IDs, grade codes, and names only; never `Generated`/ledger entries/snippets. Note `search_doc` may capture comment lines above a program (existing indexer behavior); local SQLite only |
| Secret-looking provenance keys dropped from projection | Repudiation (silent loss) | Key names avoid filter substrings; projection test asserts `Secret == 0` |
| Pathological parse memory spike (DoS) | Denial of service | Serialize `Analyze` (semaphore 1); producer 2-min ceiling; crash isolation remains opt-in |
| Private grammar/queries published in release binaries (A8) | Information disclosure | User decision; gate `release.yml` if not acceptable |

Regulatory note: no customer PII or CPNI is processed. The estate source is proprietary code, handled locally only.

## Sources

### Primary (HIGH confidence)
- Probes this session (scratch module and `-modfile` runs against the Gortex tree): pseudo-version resolution, replace acceptance, go.sum entries, build info under `go test -race`, stock-grammar negative, nested/duplicate/amber/seq-area/copybook handoffs, Windows `go list` failure, attestation timing, cobolprobe/dump-kinds under the shim.
- Fork source: `preprocessor/analyze.go`, `pipeline.go:26-33,461-503`, `internal/tsadapter/parser.go`, `expected_identity.go`, `internal/contract/contract.go`, `handoff/handoff.go`, `sourcemap/sourcemap.go`, `parsefacts/facts.go`, `grade/grade.go`, `grammar.js:36-56,1378-1382`, `forest-shim/cobol/{go.mod,binding.go,plugin.go}`, `docs/vendoring.md`, `go.work`, `preprocessor/go.mod`.
- Gortex source: `internal/parser/languages/{cobol.go,register.go}`, `internal/parser/extractor.go`, `internal/graph/{node.go,edge.go,overlay.go,repo_prefix.go}`, `internal/graph/store_sqlite/{store.go,meta_json.go,schema.go}`, `internal/indexer/{indexer.go,multi.go,parse_graph_batch.go,shadow_threshold.go,crash_isolation.go,metadata_normalize.go}`, `internal/gitstate/dirty.go`, `internal/mcp/{tools_core.go,tool_presets.go,parse_gate.go}`, `internal/neo4jprojection/properties.go`, `cmd/gortex/{query.go,call.go,node.go,daemon_integration_test.go}`, `internal/indexer/multi_reconcile_sqlite_test.go`, `internal/testutil/graphfixture/fixture.go`, `.github/workflows/*.yml`, `go.mod`.
- Planning: `02-CONTEXT.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`, `COBOL-PARSER-READINESS.md`, `GRAPH-SCHEMA.md`, `docs/ai-enhanced-cobol-graph-handoff.md` §4/§6/§12/§18.

### Secondary (MEDIUM confidence)
- Design inferences: VCS stamp hook placement, SQLite daemon harness variant.

### Tertiary (LOW confidence)
- A1, A2 (GitHub platform behaviors from training knowledge).

## Metadata

**Confidence breakdown:**
- Standard stack / module mechanics: HIGH (end-to-end probes, including the negative case)
- Architecture (extractor, IDs, prov mapping, retrieval): HIGH for the facts, MEDIUM for the VCS stamp and the test-harness variant
- Pitfalls: HIGH (each reproduced or read in source)

**Research date:** 2026-09-30
**Valid until:** 2026-10-30, or until the fork pushes the D-03 commit (re-derive the pseudo-version then)
