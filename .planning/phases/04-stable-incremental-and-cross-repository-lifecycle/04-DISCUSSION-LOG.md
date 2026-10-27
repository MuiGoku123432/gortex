# Phase 4: Stable Incremental and Cross-Repository Lifecycle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-12
**Phase:** 04-stable-incremental-and-cross-repository-lifecycle
**Areas discussed:** Equivalence proof, Invalidation & stale facts, Cross-repo domains, Scope & ownership gate

---

## Equivalence proof

| Question | Options | Selected |
|---|---|---|
| Comparison content | IDs+kinds+edges+det. meta / IDs + edges only / Everything byte-for-byte | IDs+kinds+edges+det. meta ✓ |
| Checker | Canonical dump + diff / Digest only / You decide | Canonical dump + diff ✓ |
| Surface | Test-only helper now / CLI + MCP verb now | Test-only helper now ✓ |
| Sequences | Rerun/edit/delete/restore; Version & config; Cross-file & cross-repo; Interrupted / large fan-out | "Whatever you think is best" → Claude chose all four |

## Invalidation & stale facts

| Question | Options | Selected |
|---|---|---|
| Prior observations on edit/delete | Hard-evict / Tombstone as inactive | Hard-evict ✓ |
| Material configuration | Explicit per-extractor fingerprint / Whole indexConfigHash | Explicit per-extractor fingerprint ✓ |
| Version staleness | Derive from stamped identity / Keep hand-bumped map | Derive from stamped identity ✓ |
| Affected-by > 200 cap | Escalate to full resolve / Remove the cap / Mark index not-exact | Escalate to full resolve ✓ |

## Cross-repo domains

| Question | Options | Selected |
|---|---|---|
| Allowed scope declaration | Workspace config allowlist / No cross-repo binding this phase / Reuse workspace import rules | Workspace config allowlist ✓ |
| Copybook search order | Ordered list in config / Keep exactly-one-match | Ordered list in config ✓ |
| Qualified vs _unqualified | Never merge; link instead / Merge via configured default / Never merge, no link | Never merge; link instead ✓ |
| Generic first-match nondeterminism | Ambiguous → refuse / Deterministic sort only / COBOL only | Ambiguous → refuse ✓ |

**Notes:** cobol-repo-architect's manifest has no library concatenation, and that project is at its Phase 2, so the manifest-fed search order is deferred.

## Scope & ownership gate

| Question | Options | Selected |
|---|---|---|
| SEC-02 fail-closed reach | Source reads only / Everywhere / Mainframe workspaces only | Source reads only ✓ |
| Single-repo unowned nodes | Always prefix + count unowned / Require multi-repo for estates | Always prefix + count unowned ✓ |
| SEC-08 enforcement | Test gate + health status / Block export/projection now / Test gate only | Test gate + health status ✓ |
| SEC-01 same scope | One shared scope check + tests / Per-surface checks + tests | One shared scope check + tests ✓ |

## Claude's Discretion

- Harness sequence set (all four groups).
- Incremental COBOL resolution mechanism; volatile-key allowlist; dump format/digest; config key shapes; D-11 edge kind; placeholder exemption list; D-14 rollout; plan slicing.

## Deferred Ideas

- CLI/MCP graph digest verb (Phase 6).
- Manifest-fed copybook search order (after cobol-repo-architect manifest supports it).
- Runtime export/AI refusal when unhealthy (Phases 6–7).
- Fail-closed scope for graph-metadata queries.
