# ROBUSTNESS PLAN

Focus: increase robustness for the main workflow:

`suggest → patch → suggest → gen`

This document tracks robustness work as a checkbox list, similar to `PLAN.md`.

---

## Goals

- Make suggestions explainable (why accepted / why rejected)
- Make the interactive workflow smoother (repeatable examples + automation)
- Cover hard shapes: nested structs, pointers, slices/arrays, recursive structs
- Treat `extra` as dependency edges between assignments (implicit requires/order)

---

## Checklist

### CLI / Configurability

- [x] Add CLI flags to control matching thresholds in `suggest`
  - `-min-confidence`
  - `-min-gap`
  - `-ambiguity-threshold`
  - `-max-candidates`

### Explainability: rejected fields + thresholds

- [x] Include threshold values in exported suggestion YAML (comments)
- [x] Include per-field rejection reasons + top candidate summaries in YAML (comments on `ignore:` entries)
- [x] Include per-field confidence + strategy summaries in YAML (comments on `auto:` items)

### Validation robustness (keep CLI usable)

- [x] Restore / implement `internal/mapping.Validate()` so `gen` and `check` can validate YAML against code
- [x] Provide a shared `mapping.ResolveTypeID()` used by both validation and transform registry
- [x] Expand validation to cover:
  - [x] transforms referenced by mappings exist in registry
  - [x] `requires` referenced by `extra` are declared

### Workflow automation: examples + script

- [x] Add more example fixtures under `examples/`:
  - [x] pointers
  - [x] slices vs arrays (where supported)
  - [ ] recursive structs (cycle safety)
  - [x] nested structs with mixed pointer/slice
- [x] Add shell runner `examples/<NAME>/run.sh` that executes a basic end-to-end compile check:
  - [ ] `suggest` (generate mapping)
  - [ ] apply a committed patch (or validate expected diff)
  - [ ] `suggest` again
  - [x] `gen`
  - [x] compile the example package (via `go test ./examples/<NAME> -run '^$'`)

### `extra` dependency semantics

> Requirement: `extra` fields implicitly create dependencies between assigned fields.

- [ ] During planning/resolution, treat `extra` defs as dependency edges so that when target field B depends on target field A via `extra.def.target`, the plan respects ordering
- [ ] Ensure generator and/or planner uses topological ordering for dependent assignments
- [ ] Improve diagnostics when dependencies are unsatisfied/cyclic

### Corner cases

- [x] Nested structs (already partially supported) — add targeted examples + tests
- [x] Pointers (already partially supported) — add targeted examples + tests
- [x] Arrays — add initial end-to-end support (analyzer + resolution + codegen) + example
- [ ] Recursive structs:
  - [ ] prevent infinite recursion during resolution
  - [ ] prevent infinite recursion during codegen
  - [ ] add example demonstrating safe behavior

### Advanced feature (future)

- [ ] Generate missing target structs (DTO → domain, repo → domain):
  - [ ] propose new type definitions based on source shape
  - [ ] write generated types to a separate folder/package
  - [ ] optionally generate mapping + casters targeting those new types

---

## Completed work (so far)

- `cmd/caster-generator/main.go`
  - Added `suggest` flags for matching thresholds: `-min-confidence`, `-min-gap`, `-ambiguity-threshold`, `-max-candidates`.
  - Plumbed these into `plan.Resolver` config.
  - Plumbed these into YAML export config.

- `internal/plan/export.go`
  - Added `ExportSuggestionsYAMLWithConfig(...)` to emit YAML with comments using `yaml.Node`.
  - Added comments for:
    - threshold values
    - rejected fields (reason + top candidates)
    - accepted auto fields (confidence + strategy)

- `internal/mapping/validate.go`
  - Implemented `mapping.Validate(...)` and `ValidationResult` (required by `gen` and `check`).
  - Implemented `mapping.ResolveTypeID(...)` and switched transform registry to use it.
  - Validation now errors if a field mapping references an undeclared `transform:`.

- `examples/pointers` + `internal/gen/pointers_integration_test.go`
  - Added a pointer-focused example package and mapping fixture.
  - Added an integration test that runs `gen` for the fixture and compiles the example package.
  - Fixed codegen for pointer deref to emit safe nil-checks (and write an `.unformatted.go` sidecar on formatter failure to aid debugging).

- `examples/nested-mixed` + `internal/gen/nested_mixed_integration_test.go`
  - Added an example package that combines nested structs + pointers + slice mapping.
  - Added an integration test that runs `gen` and compiles the example package.

- Arrays support (initial end-to-end)
  - `internal/analyze`: added `TypeKindArray` and recognition of `*types.Array`.
  - `internal/plan`: treat arrays like slices for element-wise mapping and nested conversion detection.
  - `internal/gen`: generate fixed-length loops for array-to-array mapping; keep array length via `GoType.String()`.
  - `examples/arrays` + `internal/gen/arrays_integration_test.go`.

---

## Notes / assumptions

- Array support currently focuses on array-to-array mapping. We intentionally treat arrays as "collection-like" for nested mapping, but we don’t yet support array↔slice bridging.

- Reject/ignore comments are exported for unmapped target fields only (fields that end up in `ignore:`). If we want to comment on *actively rejected candidates* for mapped fields too, we’ll need to persist more ranking metadata in the plan.
