# Phase 05 Deferred Items

Out-of-scope discoveries logged by plan executors. Not fixed in the plan that found them.

## From Plan 05-05

- `internal/resolver/csharp_verbatim_type_ref_test.go` is not gofmt-clean at HEAD (last touched by commit `76658f96`, unrelated to Phase 5). `gofmt -l internal/resolver/` lists it. Whitespace only; no 05-05 verify command covers it.

## From Plan 05-16

- `internal/resolver/cobol_resources.go` `reconcileCobolResources` keeps the WR-05 shape for shared DB2 and IDMS resources. Those nodes use ID = FilePath = `<repo>/<system>::<qualifier>/<rest>` (`cobol_grammar_walk.go`, `cobolQualifiedID`), and an unreferenced one is removed with `EvictFile(n.FilePath)`. A repository file at a path such as `db2::SCHEMA/TABLE` could share that ID, and the eviction would then take every node of that file. D-27 and WR-05 cover only missing-artifact placeholders, so 05-16 left it unchanged. The fix would mirror 05-16: a `<repo>::` namespace, an empty FilePath, and removal by ID through `graph.EvictResourceNodesByIDs`.
