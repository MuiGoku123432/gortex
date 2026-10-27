---
phase: 01-general-neo4j-projection-command
plan: 04
subsystem: database
tags: [neo4j, projection, identity, metadata, security]
requires:
  - phase: 01-03
    provides: official-driver projection transport and generation staging
provides:
  - Canonical owner, node, edge, generation, and physical projection identities
  - Closed collision-safe Neo4j labels and relationship types
  - Complete safe native property mapping with reversible metadata and explicit warnings
affects: [01-05, 01-06, 01-07]
actuals:
  tokens: 6362
  tasks: 2
  commits: 4
tech-stack:
  added: []
  patterns: [length-delimited identity hashing, closed generated Cypher tokens, reversible metadata envelope]
key-files:
  created:
    - internal/neo4jprojection/identity.go
    - internal/neo4jprojection/properties.go
    - internal/neo4jprojection/properties_test.go
    - internal/neo4jprojection/testdata/properties.golden.json
  modified:
    - internal/neo4jprojection/service.go
    - internal/neo4jprojection/neo4j.go
key-decisions:
  - "Hash length-delimited identity components so exact scope, provenance, occurrence, and generation boundaries cannot alias through concatenation."
  - "Filter normalized secret-like metadata keys before reversible key encoding, and omit non-finite or unsupported values with deterministic warning counts."
patterns-established:
  - "Mapped transport records are the only source for staged Neo4j labels, relationship types, identities, and properties."
  - "Metadata keeps a canonical key map and per-property JSON encoding map for round-trip interpretation."
requirements-completed: [NEO-04, NEO-05, NEO-06, NEO-13]
coverage:
  - id: D1
    description: "Stable owner, node, edge, generation, and physical keys preserve exact scope and evidence distinctions"
    requirement: NEO-04
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/properties_test.go#TestProjectionIdentity|TestProjectionOwnerIdentity"
        status: pass
    human_judgment: false
  - id: D2
    description: "Closed generated labels and relationship types prevent raw graph values from entering Cypher syntax"
    requirement: NEO-06
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/properties_test.go#TestProjectionTokenSanitization"
        status: pass
    human_judgment: false
  - id: D3
    description: "Complete typed fields and safe metadata survive projection while secrets and unsupported values are counted and omitted"
    requirement: NEO-13
    verification:
      - kind: unit
        ref: "internal/neo4jprojection/properties_test.go#TestProjectionProperties|TestProjectionPropertiesGolden|TestProjectionUnresolved|TestProjectionSecretFiltering"
        status: pass
    human_judgment: false
duration: 8 min
completed: 2026-09-21
status: complete
---

# Phase 1 Plan 4: Deterministic Projection Identity and Properties Summary

**Neo4j staging now uses canonical evidence-preserving identities, closed generated graph tokens, and a complete reversible safe-property envelope with secret filtering.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-21T21:26:19Z
- **Completed:** 2026-09-21T21:34:17Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added length-delimited SHA-256 derivation for exact-scope owners, logical nodes, evidence-occurrence edges, source generations, and generation-specific physical records.
- Added closed collision-safe label and relationship-type generation, including explicit unresolved labeling and original kind preservation.
- Added complete typed node/edge properties, reversible metadata keys, native scalar/list handling, canonical JSON fallback markers, normalized secret filtering, and deterministic omission counts.

## Task Commits

1. **Task 1 RED: Add failing projection identity contract** - `d66a2c99` (test)
2. **Task 1 GREEN: Define canonical projection identities** - `3ea5df48` (feat)
3. **Task 2 RED: Add failing projection property contract** - `0d2db559` (test)
4. **Task 2 GREEN: Project complete safe graph properties** - `98681ec1` (feat)

## Files Created/Modified

- `internal/neo4jprojection/identity.go` - Canonical identities and closed Neo4j tokens.
- `internal/neo4jprojection/properties.go` - Typed node/edge mapping and safe metadata envelope.
- `internal/neo4jprojection/properties_test.go` - Exact identity, token, property, unresolved, and filtering contracts.
- `internal/neo4jprojection/testdata/properties.golden.json` - Byte-stable complete mapped representation.
- `internal/neo4jprojection/service.go` - Generation derivation now uses canonical length-delimited hashing.
- `internal/neo4jprojection/neo4j.go` - Staging groups mapped records by generated label/type and writes mapped properties.

## Decisions Made

- Owner repository inputs are sorted before hashing, while every scope component remains identity-bearing.
- Edge identities bind endpoints, kind, file, line, origin, and persisted occurrence discriminator without including presentation-only fields.
- Metadata uses escaped keys plus an explicit canonical key map; nested and mixed values use canonical JSON with per-property encoding markers.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The unfiltered package suite intentionally includes `TestNeo4jProductionTracer`, which fails outside the disposable-server gate. All exact plan selectors, prerequisite non-disposable contract tests, and the command build passed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-05 can batch and retry mapped records without redefining identity or property behavior.
- Plan 01-06 can reconcile generations using stable exact-owner and physical keys.

## Self-Check: PASSED

- All four created files and two modified transport/service files exist.
- Task commits `d66a2c99`, `3ea5df48`, `0d2db559`, and `98681ec1` exist in order.
- Both exact test-existence guards and focused runs pass.
- The plan-level identity/property selector passes, and the golden file remains byte-identical across repeated runs.
- `GOWORK=off go build ./cmd/gortex/` succeeds.

---
*Phase: 01-general-neo4j-projection-command*
*Completed: 2026-09-21*
