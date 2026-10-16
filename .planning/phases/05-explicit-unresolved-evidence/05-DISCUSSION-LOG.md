# Phase 5: Explicit Unresolved Evidence - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md -- this log preserves the alternatives considered.

**Date:** 2026-10-14
**Phase:** 05-explicit-unresolved-evidence
**Areas discussed:** Finding identity, Storage and history, What becomes a finding, Cascade regression gate

---

## Finding identity

**What should anchor a finding's stable ID (GAP-06)?**

| Option | Description | Selected |
|--------|-------------|----------|
| Container + evidence text | Containing symbol ID + class + kind + node kind/expected family + digest of the gap's own source text; #k only for identical text in one container | ✓ |
| Container + ordinal | Containing symbol ID + class + kind + node kind + source-order #k; fixing one gap renumbers later ones | |
| Per-file content hash | Producer gapID scheme; any file edit supersedes every finding in it | |

**When the parser/extractor version changes but the same gap persists?**

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse, re-stamp version | Parser version not in the ID; keep ID, update parser_version | ✓ |
| Supersede on parser change | Parser version in the ID; every bump rewrites every finding | |

**How to tell closed from superseded (GAP-07)?**

| Option | Description | Selected |
|--------|-------------|----------|
| Closed vs superseded by anchor | Closed = evidence gone; superseded = different finding of same class+kind at the same container, successor recorded | ✓ |
| Close only, never link | Every disappearance closes; no lineage | |

**EXTERNAL_UNRESOLVED granularity (PGMX missing, called from 40 places)?**

| Option | Description | Selected |
|--------|-------------|----------|
| Per reference occurrence | One finding per referencing statement, exact range and container | ✓ |
| Per program and target | One per (program, target), ranges listed in meta | |
| Per repo and target | One per missing artifact per repository | |

**User's choice:** Recommended option on all four.

---

## Storage and history

**Where should active findings live?**

| Option | Description | Selected |
|--------|-------------|----------|
| New `finding` node kind | Repo-prefixed, file-owned node; searchable, canonical, audited, projected | ✓ |
| Existing kind + meta | Reuse KindDoc/KindTodo + cobol_kind=finding; false semantics | |
| Separate table only | Outside the graph; new read path for every surface | |

**How are closed and superseded findings retained?**

| Option | Description | Selected |
|--------|-------------|----------|
| Active nodes + history table | Active findings as nodes; transitions appended to sidecar `finding_history` | ✓ |
| Inactive finding nodes | Closed/superseded stay as nodes with lifecycle meta; every query filters | |
| No history | Closed findings disappear | |

**How durable is finding history?**

| Option | Description | Selected |
|--------|-------------|----------|
| Best-effort, not canonical | Survives reindex/restart; reset by store rebuild; excluded from incremental==clean | ✓ |
| Durable across rebuilds | Exported and re-imported across schema rebuilds | |

**How should users see findings in Phase 5?**

| Option | Description | Selected |
|--------|-------------|----------|
| Existing tools + health counts | get_symbol/search/edges + index_health counts by class; verbs in Phase 6 | ✓ |
| Add a minimal list verb now | list_findings MCP/CLI filterable by class and file | |

**User's choice:** Recommended option on all four. (User said "continue" after this area; treated as advancing to the next area.)

---

## What becomes a finding

**Which unbound references produce EXTERNAL_UNRESOLVED findings?**

| Option | Description | Selected |
|--------|-------------|----------|
| Missing artifacts only | `missing` + `idms_dictionary_member` + producer -INC gaps; ambiguous/dynamic edge-only | ✓ |
| Also count ambiguous | ambiguous also an EXTERNAL finding | |
| Every unbound reference | Includes dynamic (829 vs 352 literal CALLs on DCC) | |

**Which in-source signals produce PARSE_UNRESOLVED findings?**

| Option | Description | Selected |
|--------|-------------|----------|
| Parser gaps only | Producer gap list only; in-source edge reasons made scope-accurate (WR-07/WR-09/IN-08) | ✓ |
| Also in-source misses | label_not_found, not_declared_* etc. become PARSE findings | |
| Only when in a cascade | In-source miss is a finding only inside an Affected region | |

**What should the GAP-05 placeholder be?**

| Option | Description | Selected |
|--------|-------------|----------|
| Repo-scoped node | One per (referencing repo, artifact type, normalized key); no fabricated content; removed with last reference | ✓ |
| Global shared node | One per normalized key across repos; crosses scope | |
| Edge target only | No node; finding carries the observed target | |

**How is severity decided?**

| Option | Description | Selected |
|--------|-------------|----------|
| Adopt producer table | Producer static table; Gortex-derived EXTERNAL = medium; versioned rule ID | ✓ |
| Gortex impact-based policy | Own policy, e.g. missing program = high | |

**User's choice:** Recommended option on all four.

---

## Cascade regression gate

**How is an uncontained cascade detected?**

| Option | Description | Selected |
|--------|-------------|----------|
| Use parser grade signals | Cascade-family grade reasons -> one high document-scope `uncontained-cascade` finding | ✓ |
| Gortex structure check | Independent expected-structure comparison | |
| Fixtures only, no finding | Test failures only; no runtime marker | |

**What makes a cascade a regression failure (GAP-08)?**

| Option | Description | Selected |
|--------|-------------|----------|
| Default test gate, runtime reports | Real-path fixture gate in default `go test -race`; runtime findings + health counts, never blocks | ✓ |
| Also mark repo unhealthy | index_health unhealthy on any cascade | |
| Fail indexing | Refuse to publish a repo with a cascade | |

**What happens to the existing cobolprobe TestErrorCascade?**

| Option | Description | Selected |
|--------|-------------|----------|
| Port cases, keep probe | Port injection cases to the real-path gate; keep probe as flag-gated measurement | ✓ |
| Leave it, new fixtures | Two suites on different parse paths | |
| Port and retire probe | Lose the -corpus measurement path | |

**How to enforce "even if AI proposes a recovery" now?**

| Option | Description | Selected |
|--------|-------------|----------|
| Deterministic closure only | Invariant: only source/parser/retrieval evidence closes findings | ✓ |
| Add explicit closable_by field | Machine-checkable remedy list on every finding | |

**User's choice:** Recommended option on all four.

---

## the agent's Discretion

- Meta key names, edge kinds (finding->container, ->placeholder), placeholder kind and ID grammar
- Evidence-digest normalization and algorithm (prefer BLAKE3); digest input when the range is absent
- Container mapping from producer observation indexes; document-scope attachment
- Supersede pairing within one container
- `finding_history` schema and migration
- Where findings are emitted (Extract vs reconcileCobol) and where lifecycle is diffed
- WR-06 representation, index_health count shape, extractor version bump, plan slicing

## Deferred Ideas

- list_findings verb -> Phase 6 QUERY-08
- History durable across store rebuilds -> revisit in Phase 8
- Impact-based severity -> Phase 6 ranking/monitoring
- ambiguous/dynamic as findings -> not now (Phase 6 QUERY-03 uses edge reasons)
- In-cascade promotion of in-source misses -> later precision work
- closable_by field -> Phase 8 if needed
- Unhealthy/fail-indexing on cascade -> rejected; Phases 6-7 may revisit for export/AI gating
- VB.NET and other-language findings -> Phase 6.1 or later
