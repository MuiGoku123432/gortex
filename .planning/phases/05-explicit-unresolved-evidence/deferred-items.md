# Phase 05 Deferred Items

Out-of-scope discoveries logged by plan executors. Not fixed in the plan that found them.

## From Plan 05-05

- `internal/resolver/csharp_verbatim_type_ref_test.go` is not gofmt-clean at HEAD (last touched by commit `76658f96`, unrelated to Phase 5). `gofmt -l internal/resolver/` lists it. Whitespace only; no 05-05 verify command covers it.
