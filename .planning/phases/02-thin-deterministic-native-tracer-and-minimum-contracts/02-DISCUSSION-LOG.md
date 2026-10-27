# Phase 2: Thin Deterministic Native Tracer and Minimum Contracts - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-15 (session 1), 2026-09-30 (session 2, resumed from checkpoint)
**Phase:** 02-thin-deterministic-native-tracer-and-minimum-contracts
**Areas discussed:** Parser pinning, Parser pinning (revisited), Extractor transition, Provenance contract, Acceptance surface

---

## Parser pinning (session 1, 2026-09-15)

| Question | Options | Selected |
|----------|---------|----------|
| How to reproducibly consume baseline 97ac9f1? | Pinned fork module / Vendor shim in Gortex / Pinned bootstrap checkout | Pinned fork module (superseded 2026-09-30) |
| Upstream forest import path or fork-owned path? | Fork-owned path / Upstream path + replace / Research decides | Fork-owned path (superseded 2026-09-30) |
| Parser identity per extraction? | Commit + grammar hash / Module version only / Grammar hash only | Commit + grammar hash |
| Parser unavailable or fingerprint differs? | Fail COBOL extraction / Skip COBOL files / Use regex with warning | Fail COBOL extraction |

---

## Parser pinning — revisited (session 2)

Reopened because `.planning/research/COBOL-PARSER-READINESS.md` (2026-09-30) showed the modules are unpublished, go.work-only, and the shim impersonates the upstream path.

| Option | Description | Selected |
|--------|-------------|----------|
| v0.26.0 handoff line | Pin accepted v0.26.0 commit; amend BASE-01/03; 97ac9f1 as floor | ✓ |
| Stay on 97ac9f1 | Walk raw tree-sitter output | |

| Option | Description | Selected |
|--------|-------------|----------|
| go.mod require + replace | Preprocessor pseudo-version + replace forest/cobol => fork shim; GOPRIVATE | ✓ |
| go.work only + loud guard | GOWORK=off can only fail loudly | |
| Ask fork to publish tags first | Block execution on tagged modules | |

| Option | Description | Selected |
|--------|-------------|----------|
| Fork adds embedded attestation | Constructor attesting against compiled-in grammar hash | ✓ |
| Configured checkout path | Env/config path passed to NewAnalyzer | |
| Skip NewAnalyzer attestation | CheckParser + ToolID compare | |

---

## Extractor transition

| Option | Description | Selected |
|--------|-------------|----------|
| Hybrid in one extractor | Program from handoff, rest regex until Phase 3 | |
| Full replace, tracer only | Only program + file edge from handoff | |
| Config-gated switch | Regex default, handoff opt-in | |

**User's choice:** "What ever gets us the best and most accurate result Id finalize this in the research phase"; follow-up: "i imagine we dont use the regex parser at all and rely on the grammar".

| Option | Description | Selected |
|--------|-------------|----------|
| Keep KindFunction | Existing queries find it | |
| New KindProgram | Semantically honest, upstream edit | |
| Research decides | Weigh compatibility vs schema fidelity | ✓ |

| Option | Description | Selected |
|--------|-------------|----------|
| Emit, marked degraded | Grade + Affected as queryable provenance | ✓ |
| Emit green only | Skip non-green | |
| Research decides | | |

---

## Provenance contract

| Question | Selected | Alternatives |
|----------|----------|--------------|
| Storage | Namespaced Meta keys | First-class Node fields; Sidecar provenance table |
| Revision | Content ID + optional VCS rev | Git commit only; Research decides |
| Same-named programs | Name + containment path (+ ordinal) | Plain path::NAME first wins; Research decides |

---

## Acceptance surface

| Question | Selected | Alternatives |
|----------|----------|--------------|
| Query surfaces | get_symbol by exact ID (+ search_symbols hit) | search_symbols only; Research decides |
| Harness | In-process real path | Built-binary end-to-end; Both |
| CI access | Grant CI read access | Local-only for now; Research decides |
| Fixture | One green + guard cases | Single program only |

---

## Claude's Discretion

- Final extractor transition shape (research, accuracy criterion)
- Program node kind (research)
- prov.* key names, deterministic confidence value, extractor version, catalog, concurrency, CLI verb, go.work handling

## Deferred Ideas

- Exported INCLUDE/COPY catalog loader from the fork
- Fork publishing tagged modules
