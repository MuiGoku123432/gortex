# Phase 3: Deterministic COBOL and Mainframe Breadth - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-01
**Phase:** 03-deterministic-cobol-and-mainframe-breadth
**Areas discussed:** Graph vocabulary, Statement & data granularity, Unresolved & ambiguous targets, Mainframe resource modeling

---

## Graph vocabulary

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse kinds + cobol_kind | Existing kinds tagged with cobol_kind; tools work immediately; no upstream schema edits | ✓ |
| Additive COBOL kinds | New KindProgram/KindParagraph/KindDataItem/KindCopybook | |
| Hybrid | Reuse where semantics match, add only where no honest fit | |

| Option | Description | Selected |
|--------|-------------|----------|
| EdgeCalls + cobol_flow meta | Reuse calls; call-chain tools trace paragraph flow | ✓ |
| New `performs` edge kind | Separate paragraph flow from inter-program CALL | |
| calls for PERFORM, new kind for GO TO | Model jump-without-return separately | |

| Option | Description | Selected |
|--------|-------------|----------|
| Expand to every paragraph in range | One edge per paragraph A..C in source order | ✓ |
| Edges to A and C only | Endpoints plus range meta | |
| Edge to A only + meta | Minimal | |

| Option | Description | Selected |
|--------|-------------|----------|
| No — explicit transfers only | PERFORM/THRU/GO TO only; ordinals kept for later derivation | ✓ |
| Yes — fallthrough edges | Needs terminator detection | |
| You decide | | |

**User's choice:** All recommended options.

---

## Statement & data granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Evidence-bearing only | Nodes only for CALL/PERFORM/GO TO/COPY/EXEC/IDMS DML | |
| No statement nodes | Occurrence on edges only | |
| All statements | Every named statement a node | (leaning) |

**User's choice:** Free text: "I am feeling all statements but evidence bearing makes sense to let research decide but I am leaning all."
**Notes:** Research decides, starting from "all", with volume evidence.

| Option | Description | Selected |
|--------|-------------|----------|
| Everything, with level/role meta | 01-49, 66, 77, 88, FILLER | ✓ |
| Named items only | Skip FILLER | |
| Records + elementary, no 88s | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Research the fork's options first | Determine whether preprocessor can parse copybooks; flag a fork prerequisite if needed | ✓ (with report-back) |
| Synthetic wrapper in Gortex | | |
| Defer copybook contents | | |

**User's choice:** Free text: "Yes research as mentioned above if it can't do it let me know I think it and or the grammar may be able to just let me know."

| Option | Description | Selected |
|--------|-------------|----------|
| Full group path + ordinal | <owner>::PROG/REC-A/WS-DATE | ✓ |
| Owner + name + ordinal | Shorter; ordinal churn | |
| You decide | | |

---

## Unresolved & ambiguous targets

| Option | Description | Selected |
|--------|-------------|----------|
| unresolved:: + existing resolver | Same-repo, exactly-one candidate binding | ✓ |
| Extractor-local only | | |
| Resolver + cross-repo | Conflicts with scope contract | |

| Option | Description | Selected |
|--------|-------------|----------|
| Distinct dyncall placeholder, never bound | Scoped to calling program, edge to data item | ✓ |
| Also try literal VALUE resolution | | |
| You decide | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Same-repo unique-name only | 0 → missing, 2+ → ambiguous with candidates | ✓ |
| Configured search order now | | |
| No binding in Phase 3 | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Placeholders + reason meta now; findings in P5 | | ✓ |
| Also emit basic finding nodes now | | |
| Ignore until Phase 5 | | |

---

## Mainframe resource modeling

| Option | Description | Selected |
|--------|-------------|----------|
| KindResource + resource_type meta | CICS programs via call path | ✓ |
| New kinds per resource | | |
| Hybrid (IDMS records → KindTable) | | |

| Option | Description | Selected |
|--------|-------------|----------|
| reads/writes/references by verb | Verb kept in meta | ✓ |
| references for all + verb meta | | |
| New uses_* edges | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Repo-scoped + explicit 'unqualified' | Qualifiers only from source | ✓ |
| Repo-scoped, name only | | |
| Workspace-global names | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Resolve aliases away, keep others distinct | | |
| Nodes for aliases too | | |
| You decide | | ✓ |

---

## Claude's Discretion

- SQL alias/CTE/include/dynamic-source representation (constraints: never a physical table, never invented)
- Final statement granularity (user leans "all")
- Exact existing kind per construct, placeholder namespaces, meta key names, containment edge kind, plan slicing

## Deferred Ideas

- Implicit fall-through / terminator analysis — behavioral twin
- Dynamic CALL value-flow resolution — later phase
- Copybook search order / cobol-repo-architect manifest — Phase 4
- Cross-repo CALL/COPY binding — Phase 4
- Merging qualified and unqualified same-named resources — Phase 4
