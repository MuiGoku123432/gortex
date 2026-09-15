# v1.0 Threat Model

**Milestone:** v1.0 Deterministic COBOL Graph Extraction and AI Enrichment Foundation  
**Status:** Planning security contract  
**Date:** 2026-09-15  
**Method:** Asset/trust-boundary analysis with release-blocking verification gates

## Security Objective

Protect proprietary mainframe source and preserve graph truth while allowing optional, bounded AI enrichment. Deterministic indexing and native queries must remain safe and complete with AI disabled. No provider capability, model confidence, export convenience, or parser output may bypass authorization, scope, evidence, or lifecycle controls.

## Assets

- Proprietary COBOL, copybooks, embedded SQL/CICS/IDMS source, literals, comments, paths, and generated/preprocessed artifacts.
- Repository, workspace, project, retrieval, library, subsystem, and revision metadata.
- Deterministic graph facts, unresolved findings, source ranges, parser diagnostics, and dataflow.
- AI contexts, prompts, outputs, claims, review decisions, and model/provider lineage.
- Provider credentials, endpoint configuration, custom headers, subprocess environment, and local model paths.
- SQLite database, query/conversation logs, caches, temporary files, exported Cypher, and projected databases.
- Trust labels: evidence class, origin, confidence, lifecycle status, scope, and reviewer identity.

## Trust Boundaries

```text
retrieved source
  -> preprocessing/parser boundary
  -> deterministic mapper/indexer
  -> authoritative SQLite graph
  -> native scoped queries
  -> context minimization/redaction policy
  -> approved provider boundary
  -> closed-schema validator
  -> append-only claim/review ledger
  -> optional scoped snapshot export
```

Crossing each arrow requires explicit validation. The provider and exported graph are outside deterministic authority. CLI/subprocess providers are external boundaries unless their actual transport, retention, and execution environment are approved.

## Current Security-Relevant Evidence

| Current evidence | Security implication |
|---|---|
| `internal/llm/provider/provider.go::New` supports local, hosted, CLI/subprocess, and custom providers | Technical availability is not organizational authorization. Provider policy must precede dispatch. |
| `internal/llm/svc/service.go::Service.RunAgent` constructs graph tools under an `opts.Scope`, invokes the provider, and can record framed prompts and final answers | Existing scoped tools are reusable, but free-form agent execution is not a claim-write API and conversation logging is a disclosure surface. |
| `internal/graph/node.go::Node` and `internal/indexer/indexer.go::Indexer.applyRepoPrefix` carry workspace/project/repository boundaries | These fields are necessary but not sufficient; context, logs, claims, and exports must apply the same allow-set. |
| `internal/mcp/query_log.go::queryLogger.append` creates `0644` JSONL logs, rotates one backup, and swallows failures | Current defaults are unsuitable as proof of a confidential AI audit trail. Raw source must not enter this path by default. |
| `internal/exporter/cypher.go::WriteCypher` emits graph snapshots | Export is an exfiltration boundary and must preserve scope/evidence filtering. |
| `internal/parser/languages/cobol.go::CobolExtractor.Extract` is the registered regex baseline, while enhanced parser acceptance currently depends on a local workspace selection | Parser/supply-chain drift can silently change graph truth and finding closure. |
| `.planning/research/PITFALLS.md` records 104 misprefixed nodes in current health evidence | Scope leakage is demonstrated risk, not hypothetical hardening. AI and projection acceptance must require clean ownership health. |

## Threats, Mitigations, and Gates

### T1. Proprietary Source Disclosure

**Threat:** Raw source, literals, paths, comments, graph snippets, prompts, responses, or diagnostics reach an unauthorized provider, log, cache, temporary file, evaluation artifact, or export.

**Mitigations:**
- AI off by default; deterministic indexing has no provider dependency.
- Classify data before context construction and apply least-content selection.
- Redact secrets, credentials, protected literals, customer identifiers, and prohibited paths before request serialization.
- Store hashes/IDs and policy metadata rather than raw contexts by default.
- Use restrictive file permissions and bounded retention for any approved persisted AI artifact.
- Treat errors, retries, tracing, crash reports, transcripts, and rotated logs as disclosure surfaces.

**Verification gates:**
- Canary secrets and proprietary markers do not appear in outbound captures, query/conversation/daemon logs, rotations, SQLite claim records, temp files, errors, or Cypher exports unless an explicit test policy allows that exact sink.
- AI-off corpus indexing performs no network/provider process invocation.
- Denied requests record metadata-only policy decisions without echoing denied content.

### T2. Provider Authorization Bypass

**Threat:** A configured hosted, custom, or subprocess provider receives source without approval for the workspace, model, endpoint, region, retention/training terms, and data class.

**Mitigations:**
- Separate provider construction from authorization. `provider.New` success does not grant use.
- Require an explicit allow-list decision keyed by workspace/project, provider, model/deployment, endpoint/region, data class, and retention mode.
- Fail closed when policy is absent, ambiguous, or overridden by broader global configuration.
- Record provider/model/policy version and source scope for every allowed or denied attempt.
- Prefer local or approved enterprise deployments for raw source.

**Verification gates:**
- Matrix tests cover global versus repository policy, custom endpoints, routing model changes, and every CLI/subprocess provider.
- Unauthorized provider tests prove zero request bytes leave the policy boundary.
- Routing cannot select a model/provider outside the approved set.

**Unresolved policy:** Approved providers, deployments, regions, training/retention terms, and authorization owner remain organization decisions.

### T3. Excessive or Unredacted Context

**Threat:** The model receives entire files, unrelated repositories, secrets, or unnecessary graph neighborhoods when one finding needs a narrow evidence set.

**Mitigations:**
- Deterministically build the smallest context: finding range, containing hierarchy, nearby AST/graph nodes, relevant declarations/dataflow, explicitly available dependencies, and missing-context list.
- Apply scope before retrieval and redaction before serialization.
- Emit a context manifest containing evidence IDs, byte/token budget, redaction classes/counts, and omissions without retaining raw payload by default.
- Reject requests exceeding policy budget rather than silently widening context.

**Verification gates:**
- Golden fixtures assert exact selected evidence IDs and maximum context size.
- Redaction runs before provider and logging adapters.
- Adversarial neighboring files/repositories containing canaries never enter context.

### T4. Logging and Retention Leakage

**Threat:** Existing query/conversation logging persists prompts, outputs, snippets, paths, or claims too broadly, with permissive modes, silent failure, or undefined retention.

**Mitigations:**
- Milestone AI logging defaults to metadata only: request ID, provider/model, policy, scope IDs, evidence hashes, schema result, usage, timing, and decision.
- No raw prompt/context/response in query logs by default, regardless of general response-logging configuration.
- Use owner-only permissions for confidential audit artifacts and explicit retention/deletion schedules.
- Provide audit-health status; do not treat fail-silent operational logging as a compliance control.

**Verification gates:**
- File mode, rotation, retention, deletion, disk-full, and revoked-handle tests.
- Static/dynamic canary scan across current and rotated logs.
- Security acceptance fails when required audit records cannot be written, while ordinary telemetry may remain fail-silent.

**Unresolved policy:** Retention periods, encryption-at-rest requirement, audit owner, legal hold, and whether any raw context may ever be retained.

### T5. Prompt Injection from Source or Graph Content

**Threat:** Source comments, copybook text, identifiers, issue text, prior claims, or retrieved graph content instruct the model to ignore policy, invoke broader tools, reveal other source, or emit trusted-looking claims.

**Mitigations:**
- Treat all repository and graph content as untrusted data, never instructions.
- Use a dedicated structured-review operation with no arbitrary write tools and the minimum read-only scoped retrieval capability.
- Delimit evidence records, label provenance, and exclude unrelated prior model text from authoritative context.
- Enforce policy, scope, and output validation outside the model; prompts are not security controls.
- Never execute model-produced commands, paths, queries, or provider changes as part of claim ingestion.

**Verification gates:**
- Injection corpus in comments, literals, file names, findings, and prior claims cannot widen scope, change provider, bypass redaction, or write deterministic rows.
- Tool-call allow-list and argument scope are enforced independently of model output.

### T6. Malformed or Semantically Invalid AI Output

**Threat:** Syntactically plausible model output contains unknown fields, invalid IDs, out-of-range confidence, unsupported predicates, evidence outside scope, or attempted deterministic promotion.

**Mitigations:**
- Require a closed, versioned schema with no unknown fields.
- Validate types, enums, bounds, maximum sizes/counts, claim identity inputs, and missing-context declaration.
- Resolve every subject/object/evidence ID against the authorized active snapshot.
- Reject deterministic evidence classes, direct graph mutations, invented target contents, and unsupported lifecycle transitions.
- Store only validated envelopes; retain rejection metadata without raw sensitive payload by default.

**Verification gates:**
- Fuzz and negative fixtures cover malformed JSON, duplicate IDs, oversized rationale, unknown evidence, cross-scope IDs, stale evidence, and `confidence=1.0` promotion attempts.
- Validator failure causes no claim or graph mutation.

### T7. Repository and Project Scope Leakage

**Threat:** Same-named artifacts from another repository resolve a reference, enter AI context, appear in a claim, leak into a log, or are included in an export.

**Mitigations:**
- Use one explicit allow-set across graph read, source read, resolution, context assembly, validation, logging, claim storage, and export.
- Fail closed on missing/ambiguous workspace, project, repository, retrieval, or library scope.
- Keep cross-repository links explicit and evidence-bearing; do not merge identity by name.
- Gate AI and export on zero unowned/misprefixed active records for the selected scope.

**Verification gates:**
- Two-workspace fixtures with duplicate names and unique canaries prove isolation through queries, contexts, claims, logs, and Cypher.
- Parent-directory and explicitly widened sessions receive separate authorization tests.
- Current ownership-health anomalies must be resolved before AI/provider or projection acceptance.

### T8. Claims Contaminate Deterministic Truth

**Threat:** Model output overwrites parser nodes, retargets edges, closes findings, invents missing artifacts, or appears in trusted default views based on confidence.

**Mitigations:**
- Separate deterministic write path from append-only claim/review ledger.
- Make evidence class mandatory and independent of origin, confidence, and review status.
- Exclude unreviewed `AI_INFERRED` claims from deterministic/default views.
- Human confirmation creates a retained reviewed assertion; it does not rewrite parser facts.
- Only deterministic source/parser/resolver changes close deterministic findings.

**Verification gates:**
- AI-on and AI-off runs yield identical deterministic active snapshots.
- Database permissions/API tests deny AI writes to deterministic rows.
- Conflicting model claims coexist; no last-write-wins mutation.
- Claims with inactive evidence become stale/ineligible while audit history remains.

### T9. Projection and Export Leakage

**Threat:** Cypher exports include unauthorized repositories, raw source metadata, unresolved evidence, unreviewed claims, or stale data; downstream consumers mistake a snapshot for authoritative synchronized truth.

**Mitigations:**
- Export from SQLite only, under explicit workspace/project/repository and evidence/status filters.
- Default to deterministic active facts; inclusion of findings or reviewed claims is explicit.
- Attach snapshot generation/time, scope, policy, and filter manifest.
- Prohibit reverse writes and document rebuild/disposal semantics.
- Keep credentials and source bodies out of export unless a separately approved use case requires them.

**Verification gates:**
- Scope/evidence canaries are absent from unauthorized snapshots.
- Export failure leaves SQLite unchanged.
- Documentation and interfaces call current Cypher output a snapshot, never synchronization.
- Rebuild from the same authoritative generation produces the same canonical projection.

### T10. Supply-Chain and Parser Drift

**Threat:** CI or release builds silently use a different COBOL grammar than local acceptance because the enhanced parser is selected through an ignored absolute-path `go.work`; parser packages or generated code drift and alter facts/findings.

**Mitigations:**
- Pin or reproducibly attest the enhanced parser/shim equivalent to baseline `97ac9f1`.
- Persist grammar-content fingerprint and extractor/schema version on extraction generations and findings.
- Verify checksums/module provenance and review dependency updates.
- Treat parsers as untrusted input processors: bounded resources, corpus regression, malformed-input tests, and fail-closed selection.
- A parser-only change triggers deterministic reconciliation and claim staleness.

**Verification gates:**
- Clean-machine and `GOWORK=off` builds identify the exact parser module/directory/hash or fail closed.
- Existing 149/149 corpus assertions, 371 NIST successes, cascade tests, and graph-level sentinel tests remain green.
- Dependency update tests compare source-positioned graph output, not aggregate counts only.
- Malformed/adversarial inputs cannot escape tracked roots, execute source content, or exhaust unbounded resources.

## Abuse Cases

| Abuse case | Required outcome |
|---|---|
| Source comment says to reveal another repository | Treated as data; no scope widening or tool authorization change. |
| User config names a technically supported but unapproved provider | Dispatch denied before context serialization. |
| Model cites a valid ID from another workspace | Schema may parse, but semantic validation rejects it. |
| Model asserts missing copybook contents with high confidence | Stored nowhere or retained only as rejected output metadata; never a fact. |
| Human confirms a claim, then parser/source changes | Confirmation remains auditable but becomes stale/version-ineligible until reviewed under the new evidence. |
| Export requests all repositories without explicit authorization | Fail closed or require explicit authorized scope; never infer permission from graph visibility. |
| Parser workspace disappears | Build/index fails closed rather than silently using an older grammar. |

## Release-Blocking Gates

1. AI is disabled by default and deterministic indexing/querying passes with no provider configured.
2. Provider authorization is explicit and tested before any source/context serialization.
3. Context selection and redaction pass canary and size-budget tests.
4. Raw proprietary content is absent from default logs, rotations, errors, temp files, claim rows, and exports.
5. Closed-schema plus semantic scope/evidence validation rejects malformed or unauthorized claims without mutation.
6. AI-on/off deterministic graph snapshots are identical.
7. Scope canaries prove isolation across queries, resolver, AI, logs, claims, and export.
8. Active graph ownership health has zero unowned/misprefixed records in the release scope.
9. Parser baseline is reproducible without developer-local workspace state.
10. Cypher remains a labeled, scoped, rebuildable snapshot and cannot write back.

## Residual and Unresolved Risk

- No provider is approved merely by this document. Authorization, retention, region, and training terms remain unresolved policy placeholders.
- Redaction cannot guarantee removal of every business-sensitive value; raw-source provider approval is still required.
- Human confirmation can be wrong; lineage and version scoping reduce but do not eliminate that risk.
- The exact confidential-audit retention/encryption design remains unresolved.
- Dependency compromise beyond pinned checksums requires broader organizational supply-chain controls.
- Future Neo4j synchronization requires a new threat-model review covering credentials, network path, tenancy, lag, tombstones, and rebuild.

## Sources

- `.planning/PROJECT.md`
- `docs/ai-enhanced-cobol-graph-handoff.md`, sections 10, 11, 14, 16, and 18
- `.planning/research/SUMMARY.md`
- `.planning/research/ARCHITECTURE.md`
- `.planning/research/PITFALLS.md`, especially pitfalls 3 through 9 and 16
- `internal/llm/provider/provider.go::New`
- `internal/llm/svc/service.go::Service.RunAgent`
- `internal/graph/node.go::Node`
- `internal/indexer/indexer.go::Indexer.applyRepoPrefix`
- `internal/mcp/query_log.go::queryLogger.append`
- `internal/exporter/cypher.go::WriteCypher`
- `internal/parser/languages/cobol.go::CobolExtractor.Extract`
- `internal/parser/forest/cobolprobe/cascade_test.go::TestErrorCascade`
