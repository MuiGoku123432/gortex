---
status: testing
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
source: [02-VERIFICATION.md]
started: 2026-10-01T19:11:34Z
updated: 2026-10-01T19:11:34Z
---

## Current Test

number: 1
name: CI test shards on head a0d2eac9 (run 36909797580)
expected: |
  ubuntu and macos test shards green; windows shard green, or failing for a cause that is diagnosed and recorded
awaiting: user response

## Tests

### 1. CI test shards on head a0d2eac9 (run 36909797580)
expected: ubuntu and macos test shards green (ubuntu "Stock COBOL grammar must fail closed" already passed); windows shard green or failing for a diagnosed, recorded cause
result: [pending]

### 2. Windows stub runs and fails closed
expected: As repo owner, the windows-latest job log shows TestCobolGrammarUnsupportedPlatformStub passing and the first '--- FAIL' line (if any) identified; the stub returns errCobolParserUnsupportedPlatform for every COBOL file (D-05, D-17)
result: [pending]

### 3. Accept review-fix deviations from plan text
expected: Accept as overrides (or amend plan/CONTEXT): expanded prov_vcs_commit_absence vocabulary (indexed_bytes_differ_from_file, vcs_query_failed); HEAD blob-ID comparison replacing gitstate.NewDirtySampler; D-17 stub rename/retag to !darwin && !linux with errCobolParserUnsupportedPlatform; and confirm the CR-01, WR-01, WR-05 logic changes
result: [pending]

### 4. Backstop truths
expected: Reindexing the unchanged fixture leaves exactly one DEMOPGM node and one defines edge (TRACE-04); TRACE-07, PROV-01, PROV-02, PROV-04 results are independent of test/package/file order (several clean COBOL files in one pass all get the same HEAD commit)
result: [pending]

### 5. Judgment-tier prohibitions
expected: Each of the ten prohibitions listed in 02-VERIFICATION.md holds (fail-closed attestation, no stock or regex facts, no source in Meta, invented fixtures, no silent degradation, no guessed containment, no false VCS attribution, no private paths in docs)
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps
