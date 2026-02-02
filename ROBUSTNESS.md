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

- [ ] Add more example fixtures under `examples/`:
  - [ ] pointers
  - [ ] slices vs arrays (where supported)
  - [ ] recursive structs (cycle safety)
  - [ ] nested structs with mixed pointer/slice
- [ ] Add shell runner `examples/<NAME>/run.sh` that executes the full loop:
  - [ ] `suggest` (generate mapping)
  - [ ] apply a committed patch (or validate expected diff)
  - [ ] `suggest` again
  - [ ] `gen`
  - [ ] execute `go test ./...` for the repo when possible

### `extra` dependency semantics

> Requirement: `extra` fields implicitly create dependencies between assigned fields.

- [ ] During planning/resolution, treat `extra` defs as dependency edges so that when target field B depends on target field A via `extra.def.target`, the plan respects ordering
- [ ] Ensure generator and/or planner uses topological ordering for dependent assignments
- [ ] Improve diagnostics when dependencies are unsatisfied/cyclic

### Corner cases

- [ ] Nested structs (already partially supported) — add targeted examples + tests
- [ ] Pointers (already partially supported) — add targeted examples + tests
- [ ] Arrays (current analyzer models slices only; decide support)
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

---

## Notes / assumptions

- Analyzer currently models slices (`TypeKindSlice`) but not arrays as a separate kind. "array" robustness work may require either:
  - teaching the analyzer to treat arrays as slices, or
  - adding a `TypeKindArray`.

- Reject/ignore comments are exported for unmapped target fields only (fields that end up in `ignore:`). If we want to comment on *actively rejected candidates* for mapped fields too, we’ll need to persist more ranking metadata in the plan.
