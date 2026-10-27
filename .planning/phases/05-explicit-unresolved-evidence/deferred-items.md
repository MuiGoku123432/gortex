# Phase 05 Deferred Items

Out-of-scope discoveries logged by plan executors. Not fixed in the plan that found them.

## From Plan 05-05

- `internal/resolver/csharp_verbatim_type_ref_test.go` is not gofmt-clean at HEAD (last touched by commit `76658f96`, unrelated to Phase 5). `gofmt -l internal/resolver/` lists it. Whitespace only; no 05-05 verify command covers it.

## From Plan 05-16

- `internal/resolver/cobol_resources.go` `reconcileCobolResources` keeps the WR-05 shape for shared DB2 and IDMS resources. Those nodes use ID = FilePath = `<repo>/<system>::<qualifier>/<rest>` (`cobol_grammar_walk.go`, `cobolQualifiedID`), and an unreferenced one is removed with `EvictFile(n.FilePath)`. A repository file at a path such as `db2::SCHEMA/TABLE` could share that ID, and the eviction would then take every node of that file. D-27 and WR-05 cover only missing-artifact placeholders, so 05-16 left it unchanged. The fix would mirror 05-16: a `<repo>::` namespace, an empty FilePath, and removal by ID through `graph.EvictResourceNodesByIDs`.

## From Plan 05-17

- `cmd/gortex/main_test.go` `TestMain` fails the whole `cmd/gortex` package when any of eight real per-user paths changes during the run (listed by `realUserStatePaths`, compared by `diffUserState` after `m.Run()`). One of them is the developer's real Gortex query log, `~/.gortex/cache/query-log.jsonl` (line 74), hashed by content. `TestMain` sets `GORTEX_QUERY_LOG_DISABLE=1` for the package's own processes, but any Gortex tool call made elsewhere on the machine during the run (another agent session, or an earlier call still finishing in the daemon) appends to that file. So any Gortex use anywhere on the machine during `go test ./...` fails the package even when every test passes. The guard's false-positive hint mentions only a daemon that tracked a repository; any tool call is enough.
- The 05-17 phase gate (`GOWORK=off go test -race -timeout 60m ./...` on `3abe472d`) tripped on this three times, on three different outside writers. Each time every `cmd/gortex` test passed, the log had 0 `--- FAIL` lines, and the guard report named only the query log:
  - run 1 (started 16:25:36Z): the previous 05-17 executor's own timed-out Gortex `change detect` call, which finished in the daemon and wrote a `detect_changes` entry at 16:29:34Z;
  - run 2 (started 18:13:28Z): a `detect_changes` call from a pricing-engine-service session at 18:15:09Z;
  - run 3 (started 18:40:12Z): `explore` at 18:41:38Z and `search_text` at 18:41:43Z from a kinetic-sales-quest session.
  Run alone with `-count=1` and the guard intact, `cmd/gortex` passed twice with no outside write.
- Pre-existing test-harness issue: the guard dates from `34e1f482` (2026-08-08), before Phase 5, and is outside Phase 5 scope, so it is not fixed here. A fix has to tell a write by the test from a write by another session, for example by dropping the query log from the content guard now that `TestMain` disables query telemetry for the package. Until then, run the full suite with no Gortex activity on the machine, or confirm a `cmd/gortex` guard failure by re-running that package alone.
