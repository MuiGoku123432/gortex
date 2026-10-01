---
schema_version: 1
open_count: 3
waived_count: 0
fixed_count: 0
total_count: 3
last_updated: 2026-10-01T17:21:14.000Z
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
  }
]
````
