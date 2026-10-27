---
status: complete
phase: 05-explicit-unresolved-evidence
source: [05-VERIFICATION.md]
started: 2026-10-26T23:55:52Z
updated: 2026-10-27T00:37:15Z
---

## Current Test

[testing complete]

## Tests

### 1. Judgment-tier prohibition 05-07 (GAP-08): an uncontained cascade never passes silently or behind a flag, never blocks indexing or marks a repository unhealthy, and no later AI output can clear the cascade finding.
expected: Confirm. Non-authoritative LLM-judge verdict: HOLDS. TestCobolCascadeGate_{CleanControl,Positive,Negative,Copybook} pass at HEAD 64d20a6e in the default -race run with no flag. TestFindingWriters_Inventory passes at HEAD. TestIndexHealth_FindingsDoNotScore passed at c93bce3c, and internal/mcp is untouched since. The graph exposes no generic remove-by-ID, so a finding leaves the store only through eviction of its file or repository or the fenced resolver call to EvictFindingNodesByIDs. The review-fix commits add no finding writer.
result: pass
note: "User sign-off 2026-10-27: 'lets verify run it and I think were good'. The 2026-10-27 full race suite (test 3) re-ran TestCobolCascadeGate_* and TestFindingWriters_Inventory green on the shipped code."

### 2. Judgment-tier prohibition 05-09 (GAP-07): no path other than deterministic evidence (source edit, parser or extractor change, retrieval or binding) closes, supersedes, suppresses, or alters a finding.
expected: Confirm. Non-authoritative LLM-judge verdict: HOLDS. The D-16 inventory (TestFindingWriters_Inventory, PASS at HEAD) confines KindFinding construction, finding_status writes, AppendFindingTransitions, and EvictFindingNodesByIDs to the extractor, the resolver, and the ledger. The new writer section (multi.go:1087-1097) only calls the existing resolver reconcile and ledger. Residual: the inventory matches literal write shapes, not a Meta write through a computed key. The previous residual (a race could mislabel close versus supersede on writer-free master paths) is closed by be0f4f83.
result: pass
note: "User sign-off 2026-10-27: 'lets verify run it and I think were good'. Accepted residual: the inventory matches literal write shapes only."

### 3. Re-run the full race suite on HEAD 64d20a6e with no Gortex sessions active (stop the daemon and every 'gortex mcp' process first): GOWORK=off go test -race -timeout 60m ./...
expected: exit=0, with no '--- FAIL', 'DATA RACE', or 'panic:' line, and cmd/gortex passing with its TestMain guard intact. That re-certifies the 05-17 gate on the code that now ships and closes WINDOWS.md entry 8.
result: pass
source: automated
evidence: "05-UAT-full-suite.log (commit 0de1b7d2): HEAD b69bee7d (code identical to 64d20a6e), 2026-10-27T00:15:16Z to 00:36:16Z, 142 GiB free, exit=0; 157 ok (115 cached from unchanged packages, 42 executed, including internal/indexer 1193.6s, internal/mcp 544.6s, cmd/gortex 154.1s, internal/resolver 70.0s), 40 no test files, 0 FAIL lines, 0 '--- FAIL', 0 DATA RACE, 0 panic. The query-log and the other four guarded files were unchanged across the run (quiet since 23:25:34Z)."

## Summary

total: 3
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
