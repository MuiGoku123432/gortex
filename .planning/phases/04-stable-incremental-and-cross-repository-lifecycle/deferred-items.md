## Deferred Items

- Stale generic ambiguity stamp when a later pass finds no eligible candidate (found during 04-04)
  status: open
  **What:** `internal/resolver/cross_repo.go` clears `resolution_unresolved_reason` / `resolution_candidate_ids` only when the cross-repository pass binds the edge. If a later pass finds zero eligible candidates (for example, an import or a repository was removed), the edge stays unresolved and keeps the old `ambiguous` stamp.
  **Why deferred:** Outside 04-04's D-12 scope (the plan decides the stamp per pass and documents the master-resolver residual as Pitfall 8). A fix needs a "decided this pass" signal plus a Meta-only clear that is persisted.

- Remote proxy stitch still runs after a local ambiguity refusal (found during 04-04)
  status: open
  **What:** In `CrossRepoResolver.resolveEdge`, `tryRemoteStitch` runs whenever `resolveFunctionCall` leaves `e.To` unchanged, and that includes the new D-12 ambiguity refusal. Its evidence gate refuses bare names. Proxy-edge minting is off by default.
  **Why deferred:** This is behavior that already existed and sits outside D-12's local fallback. Decide whether local ambiguity should block the remote stitch when cross-daemon proxy edges are in scope.

- Stale cross_repo_calls parallel edge when a cross-repository COBOL CALL unbinds without eviction (found during 04-15)
  status: open
  **What:** 04-15 makes `Resolver.reconcileCobol` run `DetectCrossRepoEdgesForReindexes` on its rebound batch, so a CALL bound across repositories gets its `cross_repo_calls` edge on both the clean and incremental paths. When such a CALL later returns to its placeholder and neither endpoint file is evicted, the parallel edge stays. For example, crossa adds its own XPGM, so the CALL becomes ambiguous. The parallel layer has always relied on `EvictFile` to drop stale edges, and generic cross-repository rebinds behave the same way.
  **Why deferred:** No D-04 sequence in 04-15 reaches this case. Deleting the target evicts the parallel edge. A fix needs a store-level removal of the old `(From, Line, cross_repo_calls, OldTo)` edge on unbind. It belongs with the Phase 6 generic cross-repository re-check (research Open Question 4).

- Non-COBOL file nodes added on the incremental path dump as `{}` where a clean build dumps `null` (found during 04-09)
  status: open
  **What:** The incremental patch stamps the `source_*_fingerprint` delta-probe cache keys into every file node's Meta (`stampExtractionGraphFingerprints`, `stampDerivedFingerprints`). `graphdump` strips them as volatile (`graphdump-volatile/2`, 04-08), so a file node whose Meta held nothing else dumps as `{}`, and a clean build of the same file dumps `null`. This applies to any non-COBOL file (reproduced with `conf/x.yaml` and `pkg/a.go`). COBOL file nodes always carry `prov_*` keys, so the existing COBOL harness never hit it. 04-09's config-change tests add `.gortex.yaml` incrementally, and `requireConfigCleanEquivalent` normalizes only that one row.
  **Why deferred:** The fix is in the shared dump tool: render an emptied Meta as `null` and version the policy as `graphdump-volatile/3`. That is outside 04-09's files and changes the measuring tool every Phase 4 lifecycle test relies on.
