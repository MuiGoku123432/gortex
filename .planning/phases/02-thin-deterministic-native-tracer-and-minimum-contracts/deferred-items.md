# Phase 02 Deferred Items

Out-of-scope discoveries logged during execution. They were not fixed in the plan that found them.

## From Plan 02-02

1. **Existing `go mod tidy` drift for the Neo4j driver.** At `94ebade0`, before any 02-02 change, `GOWORK=off go mod tidy` moves `github.com/neo4j/neo4j-go-driver/v6 v6.3.0` from the indirect `require` block to the direct one, because Phase 1 imports it directly. Plan 02-02 reverted that hunk to keep its go.mod diff to the planned lines. The drift will fail the `ci.yml` lint job's `go mod tidy` diff check. Owner: Plan 02-05 (CI), or a one-line chore.
   status: resolved (commit 81b56bf2; see From Plan 02-05 item 1)
2. **The BASE-02 stock-grammar CI step needs the stock go.sum lines.** Under the replace, `go mod tidy` drops the two `github.com/alexaandru/go-sitter-forest/cobol v1.9.1` go.sum lines. A `-modfile` copy made with `-dropreplace` therefore lacks the stock module's sums. `! go test -modfile=…` would then pass vacuously on a "missing go.sum entry" error instead of on `compiled forest language identity mismatch`. Plan 02-05's step should restore the sums, for example with `go mod download` against the temporary modfile, and assert the mismatch text rather than any failure.
   status: resolved
   **Resolution (Plan 02-05):** the `Stock COBOL grammar must fail closed` step in `ci.yml` runs the pin test with `GOFLAGS=-mod=mod`, which fetches the stock sums into the temporary `stock.sum`, and it passes only when the output contains `compiled forest language identity mismatch`. It was run locally and passed.

## From Plan 02-05

1. **Item 1 above (the Neo4j tidy drift) is still open, and the lint job will now reach it.** `GOWORK=off go mod tidy -diff` at `738c165e` still shows only that move of `neo4j-go-driver/v6 v6.3.0` from indirect to direct, with no go.sum change. Plan 02-05's file scope is the seven workflows, so go.mod was left untouched. Once `COBOL_PARSER_READ_PAT` exists, the `ci.yml` `lint` job's `go.mod is tidy` step will fail on this until someone commits the one-line `go mod tidy` result. Owner: a one-line chore before or during Plan 02-06's green-CI checkpoint.
   status: resolved
   **Resolution (commit 81b56bf2):** the orchestrator committed the `go mod tidy` result with the user's approval, moving `neo4j-go-driver/v6` to the direct `require` block. `GOWORK=off go mod tidy -diff` is now clean, and Plan 02-06's gate re-ran `go mod tidy` with no go.mod or go.sum diff.

## From Plan 02-06

1. **The windows-latest CI test shard fails, and the cause has not been diagnosed.** With `COBOL_PARSER_READ_PAT` provisioned, 12 of 13 checks passed on head `d4e0b3b78779b11206e2b90ecb3e2ccbc5065daa`. `test (windows-latest, 1.27)` failed in step `Test (windows, no race detector)` (`go test -v -timeout=45m ./...`, exit 1). It also failed in the same step on the first run at `54ac27c9`. The only public annotation is "Process completed with exit code 1.", and reading the job log needs repo-admin access, which the local `gh` session (an Enterprise Managed User account) does not have. So it is unknown whether the failure is in `TestCobolGrammarWindowsStub` (D-17), in another Phase 1 or 2 test, or in an upstream test that already failed on Windows. The user accepted Plan 02-06's CI checkpoint with this known exception.
   - Second run (head `d4e0b3b7`): https://github.com/MuiGoku123432/gortex/actions/runs/36889132850/job/110460015885
   - First run (head `54ac27c9`): https://github.com/MuiGoku123432/gortex/actions/runs/36877578916/job/110420852834
   - Next step: as the repo owner, open the job log, find the first `--- FAIL` line, and decide whether it is phase-caused (fix it) or pre-existing (re-run on upstream `main` and record it). WINDOWS.md entry 2 stays open until this is resolved.
   - Owner: a follow-up before Phase 2 verification signs off D-17, or `/gsd-debug`.
   status: open
2. **Pre-existing golangci-lint findings blocked the lint job.** The first credentialed run failed `lint` on 7 findings in Phase 1 code and the pre-milestone cobolprobe test.
   status: resolved
   **Resolution (commits 9f21e855, d4e0b3b7, user-approved):** fixed the unused, ineffassign, and errcheck findings, plus three real dropped-error bugs (`rows.Err()` in `graphfixture.Canonical` and in the scoped-projection endpoint read, and the `neo4j push --json` stdout write). `lint` is green on `d4e0b3b7`.

## From code review (02-REVIEW.md WR-07)

1. **Fork PRs and Dependabot cannot pass CI, by design of D-13.** `pull_request` runs from forks get no repository secrets, and Dependabot-triggered runs get only Dependabot secrets, so `COBOL_PARSER_READ_PAT` is empty and every Go-building job stops at `::error::COBOL_PARSER_READ_PAT is not configured`. D-13 chose "no skip-when-missing", and this is its recorded consequence. For a public repository it means an external contribution can only be tested after a maintainer reviews it (including `.github/`) and pushes the branch to this repository. Separately, Dependabot's gomod updater cannot resolve the private fork, so Go dependency updates stop. The behaviour is documented at the top of `.github/workflows/ci.yml` and in the gomod section of `.github/dependabot.yml`.
   - User action to restore Dependabot: create a Dependabot secret `COBOL_PARSER_READ_PAT` (same fine-grained, single-repo, Contents read-only token), then add the `registries: cobol-fork` entry shown in `.github/dependabot.yml` and reference it from the gomod update. The entry was not added yet, so the config never references a secret that does not exist.
   - Owner: user (secret provisioning), then a one-line config follow-up.
   status: open

## From code review (02-REVIEW.md), deferred 2026-10-01 (user-approved)

- [ ] **WR-04 — COBOL analyze slot is per-process; slot wait counts against `max_extract_millis`.** Needs the indexer's extraction budget plumbed into the extractor (upstream `skip_telemetry.go` beyond the D-18 hook) and a crash-isolation design decision. Target: Phase 4 (lifecycle). Documented in the `cobolAnalyzeSlot` comment (43c2e807).
- [ ] **FK-01 (fork) — `AttestEmbedded` reports `CheckoutArtifactID` from a generated constant without verifying it.** Producer-side; needs a fork commit, user-approved push, and a Gortex re-pin. Gortex trust is unaffected (compiled-language ID is still verified; Gortex pins both modules).
- [ ] **FK-02 (fork) — `dependencyVersion` ignores `replace`, so `ToolIdentity.ForestModuleVersion` records stock `v1.9.1`.** Same fix path as FK-01. Gortex's own `prov_parser_module` records the replaced shim version correctly.
- [ ] **CR-01 residual — `prov_start_byte`/`prov_end_byte` are off by the BOM length for BOM-prefixed files.** PROV-03 edge case; fixtures have no BOM. Target: Phase 3.
