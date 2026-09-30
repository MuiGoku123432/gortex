# Phase 02 Deferred Items

Out-of-scope discoveries logged during execution. They were not fixed in the plan that found them.

## From Plan 02-02

1. **Existing `go mod tidy` drift for the Neo4j driver.** At `94ebade0`, before any 02-02 change, `GOWORK=off go mod tidy` moves `github.com/neo4j/neo4j-go-driver/v6 v6.3.0` from the indirect `require` block to the direct one, because Phase 1 imports it directly. Plan 02-02 reverted that hunk to keep its go.mod diff to the planned lines. The drift will fail the `ci.yml` lint job's `go mod tidy` diff check. Owner: Plan 02-05 (CI), or a one-line chore.
2. **The BASE-02 stock-grammar CI step needs the stock go.sum lines.** Under the replace, `go mod tidy` drops the two `github.com/alexaandru/go-sitter-forest/cobol v1.9.1` go.sum lines. A `-modfile` copy made with `-dropreplace` therefore lacks the stock module's sums. `! go test -modfile=…` would then pass vacuously on a "missing go.sum entry" error instead of on `compiled forest language identity mismatch`. Plan 02-05's step should restore the sums, for example with `go mod download` against the temporary modfile, and assert the mismatch text rather than any failure.
