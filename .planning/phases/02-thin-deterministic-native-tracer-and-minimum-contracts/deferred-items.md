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
