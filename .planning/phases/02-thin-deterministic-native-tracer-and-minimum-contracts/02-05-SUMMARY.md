---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
plan: 05
subsystem: infra
tags: [github-actions, ci, goprivate, private-module, goreleaser-cross, govulncheck, supply-chain]

# Dependency graph
requires:
  - phase: 02-02
    provides: "go.mod require + replace pinning the private tree-sitter-cobol-upgrade fork (preprocessor, forest-shim/cobol)"
  - phase: 02-03
    provides: "TestCobolGrammar_ApprovedGrammarPinned and the local proof that the stock grammar fails with 'compiled forest language identity mismatch' under GOFLAGS=-mod=mod"
provides:
  - "Workflow-level GOPRIVATE for the fork in all seven Go-building workflows"
  - "Fail-closed 'Configure read access to the private COBOL parser module' step (COBOL_PARSER_READ_PAT -> git insteadOf) right after all 14 actions/setup-go steps"
  - "cache: false on every setup-go step and on golang/govulncheck-action"
  - "ci.yml test job ubuntu step 'Stock COBOL grammar must fail closed' (BASE-02 build-level negative)"
  - "release.yml release job: host setup-go + credential + go mod download, with goreleaser-cross reading the host module cache read-only under GOPROXY=off"
affects: [02-06, release, ci, security]

# Actuals (#2632)
actuals:
  tokens: 5364
  tasks: 2
  commits: 2
plan_head_before: 3eeaff9ebbe0ab7ac38957bb47f464be24b1f3b3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Secret passed only through step env, validated with ::error:: + exit 1, and used only in a global git insteadOf rewrite scoped to the fork URL"
    - "Offline container build: host downloads modules, container mounts GOMODCACHE read-only with GOPROXY=off and receives no fork credential"

key-files:
  created: []
  modified:
    - .github/workflows/ci.yml
    - .github/workflows/security.yml
    - .github/workflows/init-smoke.yml
    - .github/workflows/skill-drift.yml
    - .github/workflows/bench-arm.yml
    - .github/workflows/publish-claude-plugin.yml
    - .github/workflows/release.yml

key-decisions:
  - "The release container mounts the host module cache with :ro on top of the plan's GOPROXY=off, so the container cannot write to or tamper with the host cache. If a real release shows Go needs to write there, the fallback is to drop :ro and keep the rest."
  - "The Neo4j go mod tidy drift was left alone because go.mod is outside this plan's file scope. It is recorded as an open CI risk: the lint job's tidy check will fail on it once the secret exists."
  - "govulncheck-action keeps repo-checkout: false, so the global insteadOf written before it survives into its internal setup-go. It also gets cache: false because its cache input defaults to true at the pinned commit."

patterns-established:
  - "Every new Go-building job adds setup-go with cache: false, followed immediately by the verbatim credential step, and relies on workflow-level GOPRIVATE."

requirements-completed: [BASE-02, TRACE-07]

coverage:
  - id: D1
    description: "All seven Go-building workflows declare GOPRIVATE, have no pull_request_target trigger, set cache: false on every setup-go and on govulncheck-action, and follow each of the 14 setup-go steps with the fail-closed credential step"
    requirement: "BASE-02"
    verification:
      - kind: other
        ref: "02-05-PLAN.md 'CI structural checker' run over the 7 workflow files (prints OK)"
        status: pass
      - kind: other
        ref: "yq '[.. | select(has(\"uses\")) | select(.uses | test(\"^actions/setup-go@\")) | .with.cache] | unique' ci.yml -> [false]; grep -c credential step: ci 5, security 1, init-smoke 1, skill-drift 1, bench-arm 2, publish-claude-plugin 1, release 3"
        status: pass
    human_judgment: false
  - id: D2
    description: "The ci.yml 'Stock COBOL grammar must fail closed' step body drops the forest replace in a temp modfile and passes only on 'compiled forest language identity mismatch'"
    requirement: "BASE-02"
    verification:
      - kind: other
        ref: "Task 1 verify: yq-extracted step body run under GOWORK=off (exit 0, printed 'stock COBOL grammar rejected: compiled forest language identity mismatch')"
        status: pass
    human_judgment: false
  - id: D3
    description: "The credential step body writes an insteadOf rewrite, never prints the secret, and fails with ::error:: when the secret is empty"
    verification:
      - kind: other
        ref: "Task 1 verify: extracted body run with HOME=temp and a dummy value (insteadof written, output empty); with an empty value it exits non-zero with '::error::COBOL_PARSER_READ_PAT is not configured'"
        status: pass
    human_judgment: false
  - id: D4
    description: "The linux release builds offline in goreleaser-cross from the host-filled, read-only module cache, with no fork credential in the container"
    verification:
      - kind: other
        ref: "Checker release.yml rules (go mod download before goreleaser-cross; GOPROXY=off, /go/pkg/mod, GOMODCACHE, -e GOPRIVATE, -e GITHUB_TOKEN present; step env == {GITHUB_TOKEN}; no COBOL_PARSER_READ_PAT in the step)"
        status: pass
    human_judgment: true
    rationale: "Structure is proven, but the container and a real tag push cannot run locally. Only a real (or throwaway-tag) Release run proves the offline module-cache handoff and the :ro mount (Task 2 human-check)."
  - id: D5
    description: "CI runs the COBOL tracer acceptance like any other test, with no Neo4j service and no LLM provider in any workflow. The Windows shard gives TestCobolGrammarWindowsStub its first real run."
    requirement: "TRACE-07"
    verification:
      - kind: other
        ref: "grep -n -i 'neo4j|llm|pull_request_target' over ci.yml and security.yml -> no matches"
        status: pass
    human_judgment: true
    rationale: "Green CI on real runners needs the user to provision COBOL_PARSER_READ_PAT. That checkpoint is Plan 02-06."

# Metrics
duration: 4min
completed: 2026-09-30
status: complete
---

# Phase 02 Plan 05: CI Private-Module Access and Stock-Grammar Negative Summary

**All seven Go-building workflows now resolve the private tree-sitter-cobol-upgrade fork. Each job gets a fail-closed, step-scoped `COBOL_PARSER_READ_PAT` git rewrite, and no job caches module source. ci.yml gains an ubuntu-only BASE-02 step that passes only when the stock grammar fails with `compiled forest language identity mismatch`. The linux release now downloads modules on the host and hands goreleaser-cross a read-only module cache with `GOPROXY=off`, so the fork credential never enters the container.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-09-30T23:52:47Z
- **Completed:** 2026-09-30T23:56:30Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Added workflow-level `GOPRIVATE: github.com/MuiGoku123432/tree-sitter-cobol-upgrade` to ci, security, init-smoke, skill-drift, bench-arm, publish-claude-plugin, and release.
- Placed the `Configure read access to the private COBOL parser module` step immediately after all 14 `actions/setup-go` steps. It uses `shell: bash`, takes the secret from step env, fails with `::error::` when the secret is empty (no skip), and writes only the fork's `insteadOf` rewrite.
- Set `cache: false` on every setup-go step. This flips the former `cache: true` in security, init-smoke, bench-arm, and publish-claude-plugin. `golang/govulncheck-action` also gets `cache: false`.
- Added the ci.yml `Stock COBOL grammar must fail closed` step on ubuntu after `Build`. It uses a `-dropreplace` temp modfile and `GOFLAGS=-mod=mod`, and fails if the pin test passes or if it fails for any reason other than the attestation mismatch.
- Changed the release.yml `release` job to run pinned setup-go (cache off), then the credential step, then `go mod download` on the host. The `docker run` adds `-v "$(go env GOMODCACHE)":/go/pkg/mod:ro`, `-e GOPRIVATE`, `-e GOMODCACHE=/go/pkg/mod`, and `-e GOPROXY=off`. The step env still holds only `GITHUB_TOKEN`. `.goreleaser.yml` is unchanged relative to `3238e524`.

## Task Commits

1. **Task 1: Credential step, disabled caches, and the stock-grammar negative in ci.yml and security.yml**: `5e986b75` (ci)
2. **Task 2: Credential plumbing for the remaining Go-building workflows and a credential-free release container**: `738c165e` (ci)

## Files Created/Modified
- `.github/workflows/ci.yml`: GOPRIVATE, credential step + `cache: false` in test/build-linux-static/lint/build-onnx/benchmark, and the BASE-02 stock step.
- `.github/workflows/security.yml`: GOPRIVATE, credential step, `cache: true` changed to `false`, and govulncheck-action `cache: false`.
- `.github/workflows/init-smoke.yml`, `skill-drift.yml`, `bench-arm.yml` (both jobs), `publish-claude-plugin.yml`: GOPRIVATE, `cache: false`, and the credential step.
- `.github/workflows/release.yml`: GOPRIVATE, credential plumbing in build-darwin and release-windows, and the host download + offline container handoff in `release`.

## Decisions Made
- **Read-only module-cache mount (`:ro`).** This goes beyond the plan's literal `-v …:/go/pkg/mod`, following the orchestrator's least-privilege note. Go reads a complete module cache without writing to it, and a miss fails loudly under `GOPROXY=off` anyway. If the first real release shows a write attempt, drop `:ro` and keep everything else.
- **Tidy drift not fixed here.** go.mod is outside this plan's `files_modified`. See Issues Encountered.
- **govulncheck-action.** Its `cache` input defaults to `true` at `032d4551…`, so `cache: false` was added. `repo-checkout: false` was already set, so the action does not re-checkout, and the global git rewrite from the preceding step stays in effect.

## Deviations from Plan

**1. [Rule 2 - Least privilege] Read-only module-cache mount in the release container**
- **Found during:** Task 2
- **Issue:** The plan's `-v "$(go env GOMODCACHE)":/go/pkg/mod` would let root in the container write into the host module cache.
- **Fix:** Mounted it `:ro`. Every checker token (`/go/pkg/mod`, `GOMODCACHE`, `GOPROXY=off`, `-e GOPRIVATE`, `-e GITHUB_TOKEN`) is still present.
- **Files modified:** .github/workflows/release.yml
- **Verification:** The CI structural checker prints OK. Runtime proof waits on the first real release (D4).
- **Committed in:** 738c165e

---

**Total deviations:** 1 auto-applied (Rule 2).
**Impact on plan:** Hardening only. No scope creep, and nothing changes in what releases publish.

## Issues Encountered
- **Known CI risk: lint job tidy check.** `GOWORK=off go mod tidy -diff` still moves `github.com/neo4j/neo4j-go-driver/v6 v6.3.0` from indirect to direct, with no go.sum change. This pre-dates Phase 2. Once the secret exists, the `ci.yml` `lint` → `go.mod is tidy` step will fail until the one-line tidy result is committed. It is logged as open in `deferred-items.md` (From Plan 02-05). The earlier stock-go.sum deferred item is marked resolved, because `-mod=mod` fetches the sums.
- **actionlint was not run.** It is not installed, and installing a tool is excluded from auto-fix. Validation relied on yq parsing all seven files, the plan's structural checker, and running the extracted step bodies.

## Known Stubs

None.

## Threat Flags

None. The only new surface is the `COBOL_PARSER_READ_PAT` secret consumption and the container mount. Both are covered by T-02-20..T-02-25 in the plan's threat model.

## User Setup Required

Plan 02-06 owns this. The user must create a fine-grained PAT scoped only to `MuiGoku123432/tree-sitter-cobol-upgrade` with Contents: Read-only, and store it as the repository secret `COBOL_PARSER_READ_PAT`. Until then, every Go-building job fails by design at the credential step with `::error::COBOL_PARSER_READ_PAT is not configured`.

## Next Phase Readiness
- Plan 02-06 can provision the secret and confirm green CI. The Windows `test` shard will be the first real compile and run of the Windows stub and `TestCobolGrammarWindowsStub` (WINDOWS.md #2 stays open until then).
- Before or at that checkpoint, commit the one-line `go mod tidy` fix or the lint job stays red.
- The release offline handoff is unproven until a tag release runs (WINDOWS.md unrun-verify entry added).

---
*Phase: 02-thin-deterministic-native-tracer-and-minimum-contracts*
*Completed: 2026-09-30*

## Self-Check: PASSED

- FOUND: all 7 modified workflow files; `.goreleaser.yml` unchanged vs 3238e524
- FOUND: commits 5e986b75, 738c165e (`git rev-list --count 3eeaff9e..HEAD` = 2 before this SUMMARY)
- PASS: Task 1 verify (checker OK, stock step exit 0 with the mismatch text, credential body checks), Task 1 acceptance (cache values `[false]`, grep counts 5 and 1), Task 2 verify (checker OK over 7 workflows, .goreleaser.yml unchanged, YAML valid), Task 2 acceptance (container tokens present, step env GITHUB_TOKEN only)
