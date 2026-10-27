---
status: complete
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
source: [02-VERIFICATION.md]
started: 2026-10-01T19:11:34Z
updated: 2026-10-01T20:31:35Z
---

## Current Test

number: 1
name: CI test shards on head a0d2eac9 (run 36909797580)
expected: |
  ubuntu and macos test shards green; windows shard green, or failing for a cause that is diagnosed and recorded
awaiting: none

## Tests

### 1. CI test shards on head a0d2eac9 (run 36909797580)
expected: ubuntu and macos test shards green (ubuntu "Stock COBOL grammar must fail closed" already passed); windows shard green or failing for a diagnosed, recorded cause
result: accepted (owner close-out)

### 2. Windows stub runs and fails closed
expected: As repo owner, the windows-latest job log shows TestCobolGrammarUnsupportedPlatformStub passing and the first '--- FAIL' line (if any) identified; the stub returns errCobolParserUnsupportedPlatform for every COBOL file (D-05, D-17)
result: accepted (owner close-out)

### 3. Accept review-fix deviations from plan text
expected: Accept as overrides (or amend plan/CONTEXT): expanded prov_vcs_commit_absence vocabulary (indexed_bytes_differ_from_file, vcs_query_failed); HEAD blob-ID comparison replacing gitstate.NewDirtySampler; D-17 stub rename/retag to !darwin && !linux with errCobolParserUnsupportedPlatform; and confirm the CR-01, WR-01, WR-05 logic changes
result: accepted (owner close-out)

### 4. Backstop truths
expected: Reindexing the unchanged fixture leaves exactly one DEMOPGM node and one defines edge (TRACE-04); TRACE-07, PROV-01, PROV-02, PROV-04 results are independent of test/package/file order (several clean COBOL files in one pass all get the same HEAD commit)
result: accepted (owner close-out)

### 5. Judgment-tier prohibitions
expected: Each of the ten prohibitions listed in 02-VERIFICATION.md holds (fail-closed attestation, no stock or regex facts, no source in Meta, invented fixtures, no silent degradation, no guessed containment, no false VCS attribution, no private paths in docs)
result: accepted (owner close-out)

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

- Windows: windows-latest test shard still failing on a0d2eac9 (cause undiagnosed). Accepted and deferred by owner 2026-10-01T20:31:35Z; tracked in WINDOWS.md entry 2 and deferred-items.md.
- Items 3–5 accepted by owner close-out without individual review.
