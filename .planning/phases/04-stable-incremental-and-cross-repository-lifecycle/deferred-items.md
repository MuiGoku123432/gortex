## Deferred Items

- Stale generic ambiguity stamp when a later pass finds no eligible candidate (found during 04-04)
  status: open
  **What:** `internal/resolver/cross_repo.go` clears `resolution_unresolved_reason` / `resolution_candidate_ids` only when the cross-repository pass binds the edge. If a later pass finds zero eligible candidates (for example, an import or a repository was removed), the edge stays unresolved and keeps the old `ambiguous` stamp.
  **Why deferred:** Outside 04-04's D-12 scope (the plan decides the stamp per pass and documents the master-resolver residual as Pitfall 8). A fix needs a "decided this pass" signal plus a Meta-only clear that is persisted.

- Remote proxy stitch still runs after a local ambiguity refusal (found during 04-04)
  status: open
  **What:** In `CrossRepoResolver.resolveEdge`, `tryRemoteStitch` runs whenever `resolveFunctionCall` leaves `e.To` unchanged, and that includes the new D-12 ambiguity refusal. Its evidence gate refuses bare names. Proxy-edge minting is off by default.
  **Why deferred:** This is behavior that already existed and sits outside D-12's local fallback. Decide whether local ambiguity should block the remote stitch when cross-daemon proxy edges are in scope.
