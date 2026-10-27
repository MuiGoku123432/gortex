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
  status: resolved (04-16)
  **What:** The incremental patch stamps the `source_*_fingerprint` delta-probe cache keys into every file node's Meta (`stampExtractionGraphFingerprints`, `stampDerivedFingerprints`). `graphdump` strips them as volatile (`graphdump-volatile/2`, 04-08), so a file node whose Meta held nothing else dumps as `{}`, and a clean build of the same file dumps `null`. This applies to any non-COBOL file (reproduced with `conf/x.yaml` and `pkg/a.go`). COBOL file nodes always carry `prov_*` keys, so the existing COBOL harness never hit it. 04-09's config-change tests add `.gortex.yaml` incrementally, and `requireConfigCleanEquivalent` normalizes only that one row.
  **Why deferred:** The fix is in the shared dump tool: render an emptied Meta as `null` and version the policy as `graphdump-volatile/3`. That is outside 04-09's files and changes the measuring tool every Phase 4 lifecycle test relies on.
  **Resolution (04-16):** `graphdump.canonicalMeta` renders a Meta that is empty after stripping as `null`, the same as a nil Meta, under `graphdump-volatile/3` (`TestGraphdump_EmptiedMetaIsNull`). The `.gortex.yaml` row rewrite in `requireConfigCleanEquivalent` is removed; the config-change tests call `requireCleanEquivalent`, so every row must match exactly.

- `internal/indexer/file_delta.go` is not gofmt-clean (found during 04-18)
  status: open
  **What:** `gofmt -l internal/indexer/` lists `file_delta.go`: the multi-value `return fileDeltaFingerprints{...}, derivedFingerprints{...}` near line 586 is indented one level deeper than gofmt wants. Last touched by upstream commit 7aae60eb.
  **Why deferred:** 04-18 does not modify this file, and reformatting an upstream file outside the plan's scope widens the fork diff for no behavior change.

- Lifecycle failure-path and concurrency warnings (accepted in 04-UAT test 2)
  status: open
  **What:** WR-06: when a declared repository fails to register, declared roots over-prune and nothing re-admits the nested files (probe: `TestVerifierR3_WR06`); only an explicit `IncrementalReindexRepo` restores them. WR-07: enclosing-reconcile failures are only logged (`internal/indexer/multi.go` near 2988-2993), so track/untrack report success. Also carried: REVIEW-FIX WR-01 (`cobolReconcileMu`, deferred reconcile), re-review WR-01/WR-02 (untrack restub under contention; reconcile sites outside the reach gate), and IN-05 (the "no `mi.mu` holder walks" rule is not enforced; hidden `mi.mu.RLock` on the walk path).
  **Why deferred:** WR-06 and WR-07 trigger only after an indexing or reconcile failure that is already logged, and a clean rebuild recovers. The concurrency items depend on timing a deterministic test cannot reliably reach. Fix candidates: re-admit declared roots on registration failure (WR-06); surface reconcile errors in the track/untrack result (WR-07); a lock-order assertion or a lint for IN-05.
