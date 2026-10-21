---
schema_version: 1
open_count: 3
waived_count: 0
fixed_count: 4
total_count: 7
last_updated: 2026-10-21T17:03:38.047Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 01 | stub | internal/neo4jprojection/neo4j_integration_test.go | 27 | Real-server scenario bodies are registered for Plans 01-03 and 01-06 and fail in required-scenarios mode until implemented | open |  | 2026-09-21T20:54:43.978Z |  |
| 2 | 02 | unrun-verify | internal/parser/languages/cobol_grammar_windows_test.go |  | TestCobolGrammarWindowsStub still unverified on Windows. First real runs happened in Plan 02-06 (CI run 36877578916 on 54ac27c9 and run 36889132850 on d4e0b3b7): the windows-latest test shard failed both times in step "Test (windows, no race detector)" (go test -v ./..., exit 1). The failing test and cause are unknown because the job log needs repo-admin access; the failure may be anywhere in the full suite, not necessarily this stub. See deferred-items From Plan 02-06 | open |  | 2026-09-30T23:24:38.722Z |  |
| 3 | 02 | unrun-verify | .github/workflows/release.yml |  | 02-05 Task 2 human-check not run: goreleaser-cross reading the host module cache read-only with GOPROXY=off is unproven until a real (or throwaway) tag release. COBOL_PARSER_READ_PAT was provisioned in Plan 02-06 and the ci/security/init-smoke/skill-drift jobs got past the credential step on d4e0b3b7, but release.yml has not run yet | open |  | 2026-09-30T23:56:08.041Z |  |
| 4 | 03 | unrun-verify | internal/parser/languages/cobol_grammar_sql.go |  | 03-09: data-division EXEC SQL (DECLARE CURSOR in WORKING-STORAGE) program-node flow source is unreachable under the pinned grammar (parses as ERROR, grades red); the fallback is untested | fixed | Phase 5 (05-12): no longer silent. DECLARE CURSOR in WORKING-STORAGE now persists one PARSE_UNRESOLVED finding per producer gap, and the document gets one high uncontained-cascade finding, gated by 05-07 case declare_cursor_in_working_storage. The flow-source fallback stays unreachable under the pinned grammar. | 2026-10-02T08:58:47.249Z | 2026-10-21T17:03:37.871Z |
| 5 | 03 | deviation | internal/parser/languages/cobol_grammar_sql.go |  | 03-09: INSERT INTO T1 SELECT ... FROM T2 reads of T2 are not emitted; the producer leaves the SELECT in sql_unparsed_tail (Phase 5 producer gap, no guess) | fixed | Phase 5 (05-12): the INSERT ... SELECT tail is now explicit evidence. Each tail token is one low PARSE_UNRESOLVED unparsed-tail finding (05-01, D-18). Reads of T2 are still not emitted; Gortex does not guess them. | 2026-10-02T08:58:47.347Z | 2026-10-21T17:03:37.962Z |
| 6 | 03 | deviation | internal/parser/languages/cobol_grammar_sql.go |  | 03-10: every EXEC SQL statement leaves an sql_unparsed_tail under the pinned producer, so any SQL-bearing COBOL document grades amber (unparsed-tail); the 03-10 acceptance program is amber, not green. Phase 5 producer gap | fixed | Phase 5 (05-12): every EXEC SQL tail token is now one low PARSE_UNRESOLVED unparsed-tail finding (05-01, D-18), searchable and counted in index_health (05-04). The amber grade remains; the gap is no longer silent. | 2026-10-02T09:29:09.876Z | 2026-10-21T17:03:38.047Z |
| 7 | 04 | deviation | internal/indexer/cobol_lifecycle_material_test.go |  | 04-09 config-change tests normalize the incrementally added .gortex.yaml file-node Meta ({} vs null); graphdump empty-Meta gap deferred (deferred-items.md) | fixed |  | 2026-10-13T05:02:39.021Z | 2026-10-13T05:09:07.178Z |

````json
[
  {
    "id": 1,
    "kind": "stub",
    "phase": "01",
    "file": "internal/neo4jprojection/neo4j_integration_test.go",
    "line": 27,
    "description": "Real-server scenario bodies are registered for Plans 01-03 and 01-06 and fail in required-scenarios mode until implemented",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-21T20:54:43.978Z",
    "resolved_at": null
  },
  {
    "id": 2,
    "kind": "unrun-verify",
    "phase": "02",
    "file": "internal/parser/languages/cobol_grammar_windows_test.go",
    "line": null,
    "description": "TestCobolGrammarWindowsStub still unverified on Windows. First real runs happened in Plan 02-06 (CI run 36877578916 on 54ac27c9 and run 36889132850 on d4e0b3b7): the windows-latest test shard failed both times in step \"Test (windows, no race detector)\" (go test -v ./..., exit 1). The failing test and cause are unknown because the job log needs repo-admin access; the failure may be anywhere in the full suite, not necessarily this stub. See deferred-items From Plan 02-06",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-30T23:24:38.722Z",
    "resolved_at": null,
    "milestone": "v1.0"
  },
  {
    "id": 3,
    "kind": "unrun-verify",
    "phase": "02",
    "file": ".github/workflows/release.yml",
    "line": null,
    "description": "02-05 Task 2 human-check not run: goreleaser-cross reading the host module cache read-only with GOPROXY=off is unproven until a real (or throwaway) tag release. COBOL_PARSER_READ_PAT was provisioned in Plan 02-06 and the ci/security/init-smoke/skill-drift jobs got past the credential step on d4e0b3b7, but release.yml has not run yet",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-30T23:56:08.041Z",
    "resolved_at": null,
    "milestone": "v1.0"
  },
  {
    "id": 4,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "internal/parser/languages/cobol_grammar_sql.go",
    "line": null,
    "description": "03-09: data-division EXEC SQL (DECLARE CURSOR in WORKING-STORAGE) program-node flow source is unreachable under the pinned grammar (parses as ERROR, grades red); the fallback is untested",
    "status": "fixed",
    "reason": "Phase 5 (05-12): no longer silent. DECLARE CURSOR in WORKING-STORAGE now persists one PARSE_UNRESOLVED finding per producer gap, and the document gets one high uncontained-cascade finding, gated by 05-07 case declare_cursor_in_working_storage. The flow-source fallback stays unreachable under the pinned grammar.",
    "recorded_at": "2026-10-02T08:58:47.249Z",
    "resolved_at": "2026-10-21T17:03:37.871Z",
    "milestone": "v1.0"
  },
  {
    "id": 5,
    "kind": "deviation",
    "phase": "03",
    "file": "internal/parser/languages/cobol_grammar_sql.go",
    "line": null,
    "description": "03-09: INSERT INTO T1 SELECT ... FROM T2 reads of T2 are not emitted; the producer leaves the SELECT in sql_unparsed_tail (Phase 5 producer gap, no guess)",
    "status": "fixed",
    "reason": "Phase 5 (05-12): the INSERT ... SELECT tail is now explicit evidence. Each tail token is one low PARSE_UNRESOLVED unparsed-tail finding (05-01, D-18). Reads of T2 are still not emitted; Gortex does not guess them.",
    "recorded_at": "2026-10-02T08:58:47.347Z",
    "resolved_at": "2026-10-21T17:03:37.962Z",
    "milestone": "v1.0"
  },
  {
    "id": 6,
    "kind": "deviation",
    "phase": "03",
    "file": "internal/parser/languages/cobol_grammar_sql.go",
    "line": null,
    "description": "03-10: every EXEC SQL statement leaves an sql_unparsed_tail under the pinned producer, so any SQL-bearing COBOL document grades amber (unparsed-tail); the 03-10 acceptance program is amber, not green. Phase 5 producer gap",
    "status": "fixed",
    "reason": "Phase 5 (05-12): every EXEC SQL tail token is now one low PARSE_UNRESOLVED unparsed-tail finding (05-01, D-18), searchable and counted in index_health (05-04). The amber grade remains; the gap is no longer silent.",
    "recorded_at": "2026-10-02T09:29:09.876Z",
    "resolved_at": "2026-10-21T17:03:38.047Z",
    "milestone": "v1.0"
  },
  {
    "id": 7,
    "kind": "deviation",
    "phase": "04",
    "file": "internal/indexer/cobol_lifecycle_material_test.go",
    "line": null,
    "description": "04-09 config-change tests normalize the incrementally added .gortex.yaml file-node Meta ({} vs null); graphdump empty-Meta gap deferred (deferred-items.md)",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-10-13T05:02:39.021Z",
    "resolved_at": "2026-10-13T05:09:07.178Z",
    "milestone": "v1.0"
  }
]
````
