---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
plan: 06
subsystem: infra
tags: [requirements, docs, ci, github-actions, golangci-lint, cobol, private-module, phase-gate]

# Dependency graph
requires:
  - phase: 02-04
    provides: "VCS source-revision stamping and the completed tracer extraction contract"
  - phase: 02-05
    provides: "Credential step that fails when COBOL_PARSER_READ_PAT is missing, GOPRIVATE in every Go-building workflow, and the 'Stock COBOL grammar must fail closed' ubuntu step"
provides:
  - "BASE-01, BASE-03, the Source baseline line, the Out of Scope parser line, and ROADMAP Phase 2 criterion 1 now name pinned commit f9eaf99c34a9 (accepted line f19029f, behavioral floor 97ac9f1)"
  - "PROJECT.md: context for the go.mod pin with embedded attestation, a reindex-after-upgrade note, and Key Decisions rows for D-01..D-05, D-06/D-15, and D-19"
  - "Scrubbed cobolprobe README (no private paths; documents the go.mod pin, -enhanced-parser, and fork iteration through a temporary -modfile) and an accurate COBOL row in docs/languages.md"
  - "COBOL_PARSER_READ_PAT provisioned by the user. 12 of 13 CI checks are green on d4e0b3b7; windows-latest is the accepted exception"
  - "Lint fixes, including three real dropped-error bugs in Phase 1 code"
affects: [phase-02-verification, phase-03, ci, release]

# Actuals (#2632)
actuals:
  tokens: 7800    # chars/4 over the realized diff (81b56bf2..HEAD plus ledger edits), not counting this SUMMARY
  tasks: 3
  commits: 4      # measured: git rev-list --count 81b56bf2..HEAD when this SUMMARY was written
plan_head_before: 81b56bf201a6098a93728ccaf6b73e149d7c0999

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Requirement text cites the commit derived from the go.mod pseudo-version, and a verify grep ties the two together so they cannot drift"
    - "Iterate on the fork with a temporary -modfile copy and local-path replaces, never go.work"

key-files:
  created: []
  modified:
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md
    - .planning/PROJECT.md
    - internal/parser/forest/cobolprobe/README.md
    - docs/languages.md
    - internal/neo4jprojection/neo4j.go
    - internal/neo4jprojection/neo4j_test.go
    - internal/neo4jprojection/service_test.go
    - internal/parser/forest/cobolprobe/hypo_test.go
    - internal/graph/store_sqlite/scoped_projection.go
    - internal/testutil/graphfixture/fixture.go
    - cmd/gortex/neo4j.go
    - .planning/WINDOWS.md
    - .planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/deferred-items.md

key-decisions:
  - "The user accepted the D-13 CI checkpoint with one known exception: windows-latest fails and its cause has not been diagnosed. This is recorded as an exception, not as green CI."
  - "BASE-02 is marked complete. Its text ('a clean build with GOWORK=off MUST either resolve the approved parser baseline or fail without silently falling back to the stock COBOL grammar') does not depend on Windows. The ubuntu 'Stock COBOL grammar must fail closed' step proves it on CI. The Windows stub (D-17) stays a separate, unverified item."
  - "Pre-existing lint findings in Phase 1 and pre-milestone code were fixed in this plan (user-approved deviation), because they kept the newly credentialed lint job from going green."

patterns-established:
  - "Any new CI exception is recorded in both WINDOWS.md and deferred-items.md, with the job URL and what evidence is still missing"

requirements-completed: [BASE-01, BASE-02, BASE-03, TRACE-07]

coverage:
  - id: D1
    description: "Planning contract names the pinned parser commit f9eaf99c34a9 with accepted line f19029f and floor 97ac9f1, and records the D-15 and D-19 decisions"
    requirement: "BASE-01"
    verification:
      - kind: other
        ref: "Task 1 <verify> grep chain (re-run at SUMMARY time: TASK1_OK f9eaf99c34a9)"
        status: pass
    human_judgment: false
  - id: D2
    description: "cobolprobe README and docs/languages.md are accurate and contain no absolute or home-relative paths"
    requirement: "BASE-01"
    verification:
      - kind: other
        ref: "Task 2 <verify> doc greps (re-run at SUMMARY time: DOCS_OK)"
        status: pass
    human_judgment: false
  - id: D3
    description: "With no AI provider and no Neo4j, the tracer and the full race suite pass on real ubuntu and macos runners"
    requirement: "TRACE-07"
    verification:
      - kind: integration
        ref: "ci.yml test (ubuntu-latest, 1.27) and test (macos-latest, 1.27): go test -race -timeout=45m ./... on d4e0b3b7"
        status: pass
    human_judgment: false
  - id: D4
    description: "On CI, a GOWORK=off build with the stock grammar fails closed with 'compiled forest language identity mismatch'"
    requirement: "BASE-02"
    verification:
      - kind: integration
        ref: "ci.yml test (ubuntu-latest) step 'Stock COBOL grammar must fail closed' on d4e0b3b7"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Windows test shard (including the D-17 TestCobolGrammarWindowsStub) is green"
    verification:
      - kind: integration
        ref: "https://github.com/MuiGoku123432/gortex/actions/runs/36889132850/job/110460015885"
        status: fail
    human_judgment: true
    rationale: "The job log needs repo-admin access to read, so the failing test is unknown. The user accepted the checkpoint with this exception, and it stays open in WINDOWS.md entry 2 and deferred-items From Plan 02-06."

# Metrics
duration: ~17h wall clock (2026-09-30 evening to 2026-10-01 midday, mostly the human checkpoint wait)
completed: 2026-10-01
status: complete
---

# Phase 2 Plan 06: Baseline Contract, Doc Scrub, and Credentialed CI Summary

**The requirements, roadmap, and project decisions now name the pinned parser commit `f9eaf99c34a9`. The COBOL docs are scrubbed and accurate. The private-fork CI credential is live: 12 of 13 checks are green on `d4e0b3b7`, including ubuntu and macos race suites and the stock-grammar fail-closed step. The windows-latest shard still fails for a cause nobody has diagnosed, and the user accepted that as an exception.**

## Performance

- **Duration:** about 17h wall clock. Tasks 1 and 2 ran on the evening of 2026-09-30. Task 3 waited on the user (secret provisioning, PR, two CI runs, a lint deviation) until 2026-10-01.
- **Started:** 2026-09-30 (first task commit `5ec79441` at 19:34 -05:00)
- **Completed:** 2026-10-01
- **Tasks:** 3 of 3. Task 3 was accepted with an exception.
- **Files modified:** 14, counting ledger files

## Accomplishments

- BASE-01, BASE-03, the `**Source baseline:**` line, the Out of Scope parser line, and ROADMAP Phase 2 criterion 1 all cite `f9eaf99c34a9`, the commit derived from the go.mod `preprocessor` pseudo-version. BASE-01 keeps `97ac9f1` as the behavioral floor.
- PROJECT.md describes the go.mod require-plus-replace pin with embedded attestation (not go.work). It adds Key Decisions rows for D-01..D-05, D-06/D-15 (the temporary loss of COBOL structure), and D-19 (release binaries embed the fork's `grammar.json` and `.scm` queries), plus the note to reindex after an upgrade.
- The cobolprobe README has no private paths. It documents `-enhanced-parser`, uses `$COBOL_CORPUS_ROOT`/`$COBOL_UPGRADE_ROOT` placeholders, explains fork iteration through a temporary `-modfile`, and names `cobol_grammar.go` as the production extractor. In docs/languages.md, COBOL has left the regex list, and its row says program-only extraction, temporary structure gaps, and no Windows support.
- `COBOL_PARSER_READ_PAT` was provisioned by the user, and the phase branch ran CI on real runners.

## CI Evidence (Task 3)

**Head commit:** `d4e0b3b78779b11206e2b90ecb3e2ccbc5065daa` (CI run 36889132850). The user opened the PR to `MuiGoku123432/gortex` main, but no PR URL was supplied.

| Check | Result |
|-------|--------|
| test (ubuntu-latest, 1.27), including "Stock COBOL grammar must fail closed" and `go test -race ./...` | success |
| test (macos-latest, 1.27), `go test -race ./...` | success |
| test (windows-latest, 1.27), "Test (windows, no race detector)" | **failure** (exit 1) |
| lint | success (after the deviations below) |
| build-linux-static, build-onnx, benchmark | success |
| govulncheck, trivy-fs, Trivy | success |
| dry-run (init-smoke), skill-drift, validate | success |

- Failing job: https://github.com/MuiGoku123432/gortex/actions/runs/36889132850/job/110460015885
- First credentialed run (head `54ac27c9`, run 36877578916): `lint` failed on 7 pre-existing findings, and windows failed in the same step: https://github.com/MuiGoku123432/gortex/actions/runs/36877578916/job/110420852834
- The secret works: every job, including the failing Windows job, got past the "Configure read access to the private COBOL parser module" step, which fails when the secret is missing. Nobody could list the secret's scope directly, because the local `gh` session is an Enterprise Managed User account and gets HTTP 403. The single-repo, read-only scope is therefore as the user reported, not independently verified.
- **Outcome: accepted with an exception.** The user said: "The only one that faild that time was the windows test so I consider that good". This plan does **not** claim that CI is fully green or that the Windows stub is verified. The windows step runs `go test -v ./...` over the whole repo, so the failure could be in the D-17 stub, in other phase code, or in an upstream test that already failed on Windows. The only public annotation is "Process completed with exit code 1."

## Task Commits

1. **Task 1: Amend requirements, roadmap, and project decisions to the pinned parser commit:** `5ec79441` (docs)
2. **Task 2: Scrub the COBOL docs and run the full phase gate:** `54ac27c9` (docs)
3. **Task 3: Provision the CI read credential and confirm CI on real runners:** the user did this; deviation commits `9f21e855`, `d4e0b3b7` (fix)

The orchestrator pushed `54ac27c9` and then `54ac27c9..d4e0b3b7` to origin with user authorization. This executor did not push.

## Files Created/Modified

- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/PROJECT.md`: baseline amended to the pinned commit, with the D-15/D-19 decisions
- `internal/parser/forest/cobolprobe/README.md`, `docs/languages.md`: scrubbed and accurate COBOL docs
- `internal/neo4jprojection/neo4j.go`, `neo4j_test.go`, `service_test.go`, `internal/parser/forest/cobolprobe/hypo_test.go`: lint fixes (unused, ineffassign, errcheck)
- `internal/graph/store_sqlite/scoped_projection.go`, `internal/testutil/graphfixture/fixture.go`, `cmd/gortex/neo4j.go`: dropped-error bug fixes
- `.planning/WINDOWS.md`, `deferred-items.md`: the Windows exception and the release follow-up

## Decisions Made

- **Requirements:** BASE-01, BASE-02, BASE-03, and TRACE-07 are marked complete. BASE-01/03 have the pinned commit in their text and are covered by the provenance tests, which pass on ubuntu and macos. TRACE-07 is shown by the ubuntu and macos race suites passing with no AI provider and no Neo4j. BASE-02's text is OS-independent ("a clean build with GOWORK=off MUST either resolve the approved parser baseline or fail without silently falling back") and is shown by the ubuntu stock-grammar step. Windows unavailability (D-17) is a separate decision that is **unverified**, and it is tracked in WINDOWS.md entry 2.

## Deviations from Plan

### Auto-fixed / user-approved Issues

**1. [Rule 3 - Blocking] go.mod tidy drift for neo4j-go-driver**
- **Found during:** Plans 02-02 and 02-05 (deferred). It would have failed the lint job's `go.mod is tidy` step.
- **Fix:** committed the `go mod tidy` result, moving `neo4j-go-driver/v6` from indirect to direct. No go.sum change.
- **Commit:** `81b56bf2`. The orchestrator made it before this plan's ledger base, with the user's approval.

**2. [Rule 3 - Blocking, user-approved] Pre-existing golangci-lint findings (unused / ineffassign / errcheck)**
- **Found during:** Task 3, the first credentialed CI run (`54ac27c9`)
- **Issue:** 7 findings in Phase 1 code and the pre-milestone cobolprobe test failed `lint`. They had never surfaced because lint could not run without the private-module credential.
- **Fix:** removed the dead `physicalKey` helper and the unused `batchTransport.dryRuns` field, built the hypothesis source once, and asserted that the stale-attempt `Abort` returns nil.
- **Files modified:** `internal/neo4jprojection/neo4j.go`, `neo4j_test.go`, `service_test.go`, `internal/parser/forest/cobolprobe/hypo_test.go`
- **Commit:** `9f21e855`

**3. [Rule 1 - Bug, user-approved] Dropped errors that lint surfaced**
- **Found during:** Task 3, the same lint run
- **Issue:** `rows.Err()` was never checked in the scoped-projection endpoint read (`readEndpointNodes`) or in `graphfixture.Canonical`, so a SQLite iteration error could produce a silently truncated page or fingerprint. The `neo4j push --json` stdout write error was also discarded.
- **Fix:** check `rows.Err()` and return the write error.
- **Files modified:** `internal/graph/store_sqlite/scoped_projection.go`, `internal/testutil/graphfixture/fixture.go`, `cmd/gortex/neo4j.go`
- **Verification:** local golangci-lint v2.13.1 reported `0 issues.` The touched packages pass under `-race`. CI `lint` and the ubuntu and macos race suites are green on `d4e0b3b7`.
- **Commit:** `d4e0b3b7`

**4. [Checkpoint exception] Task 3 acceptance criterion "all three OS test shards green" not met**
- windows-latest fails and its cause has not been diagnosed. The user accepted the checkpoint anyway. This is recorded, not fixed (see Deferred Issues).

---

**Total deviations:** 3 fixes (1 before the ledger base, 2 in plan) and 1 accepted checkpoint exception.
**Impact on plan:** The fixes are small and needed to get lint green on CI. Fix 3 corrects real data-integrity bugs in Phase 1 projection code. The Windows exception leaves D-17 unverified.

## Deferred Issues

- **windows-latest test shard failure.** The cause is unknown and the job log needs repo-admin access. Diagnosis steps and both job URLs are in `deferred-items.md` (From Plan 02-06, item 1). WINDOWS.md entry 2 (`TestCobolGrammarWindowsStub`) stays **open** with the updated description.
- **Release container offline build.** WINDOWS.md entry 3 stays **open**. The credential now works on ci, security, init-smoke, and skill-drift, but `release.yml` has not run.

## Issues Encountered

- `gh pr create` and secret listing failed with HTTP 403. The local `gh` session is an Enterprise Managed User account without access to the personal repositories. The user opened the PR, and the CI evidence came from the public check-runs API.

## Phase Gate (Task 2)

The previous executor ran the Task 2 gate before the checkpoint: build, vet, tidy, the focused Cobol/SourceRevision/ProjectNode race suite, the cobolprobe `-enhanced-parser` gate, the stock-grammar step body, and the Windows dependency closure without the fork. When this SUMMARY was written, the Task 1 and Task 2 doc greps were re-run and passed. `git diff --stat 3238e524 -- internal/parser/languages/cobol.go internal/parser/languages/cobol_test.go` is empty, so D-16 held. CI's ubuntu and macos `go test -race -timeout=45m ./...` on `d4e0b3b7` stand in for the full local race suite and are green.

## Known Stubs

None added by this plan.

## Next Phase Readiness

- Phase 2's planning contract, docs, and CI credential are in place. Ubuntu and macos prove the tracer with no AI and no Neo4j.
- Phase 2 verification should treat D-17 (Windows stub) as **unverified** until the windows-latest log is read and the failure is either fixed or shown to be pre-existing.

## Self-Check: PASSED

- FOUND: .planning/phases/02-thin-deterministic-native-tracer-and-minimum-contracts/02-06-SUMMARY.md
- FOUND: commits 5ec79441, 54ac27c9, 9f21e855, d4e0b3b7, 81b56bf2 in `git log`
- FOUND: Task 1 verify (TASK1_OK f9eaf99c34a9), Task 2 doc verify (DOCS_OK), D-16 diff empty
- WINDOWS.md entries 2 and 3 open with updated descriptions; deferred-items From Plan 02-06 written
