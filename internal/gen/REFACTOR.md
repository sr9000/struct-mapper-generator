# Refactoring Plan: `internal/gen`

This document outlines an incremental plan to simplify and refactor the code generation logic in `internal/gen`.

## Scope and constraints

- `internal/gen` depends on `internal/plan` output shape and strategy semantics; refactor `plan` first.
- Generated output must remain:
  - **deterministic** (stable files, stable import aliases/order, stable statement order)
  - **compilable** across all examples/tests

## Current status

Primary refactor candidate:

- `generator.go` (~1854 LOC): template data preparation, strategy application, collection loop generation, type formatting/imports, and file I/O are all mixed.

## Milestone 0 — Lock behavior (recommended first)

Add/confirm tests focused on generator stability:

- Deterministic imports and type formatting.
- Stable output across representative example cases (pointers, nested structs, collections).

Acceptance criteria:
- `go test ./...` passes.

## Milestone 1 — Extract type formatting + import management (high leverage)

Type formatting/import aliasing tends to cause subtle regressions; isolating it early reduces risk.

- **Target file**: `internal/gen/type_formatter.go` (new)
- **Move**:
  - `typeRef`, `typeRefString`, `getPkgName`
  - any import alias management helpers

Acceptance criteria:
- Generated files compile with identical imports (ordering/aliases stable).

## Milestone 2 — Extract template/view-model building

- **Target file**: `internal/gen/template_builder.go` (new)
- **Move**:
  - `buildTemplateData`, `buildAssignment`, `collectNestedCasters`

Goal: keep this layer as pure as possible (no filesystem, minimal formatting), to enable unit tests.

Acceptance criteria:
- No generated output diffs.

## Milestone 3 — Extract strategy application

- **Target file**: `internal/gen/strategies.go` (new)
- **Move**:
  - `applyConversionStrategy` and all strategy-specific helpers (pointer, nested casts, transform calls, etc.)

Acceptance criteria:
- Strategy selection behavior unchanged; emit identical code for existing tests.

## Milestone 4 — Extract collection generation

- **Target file**: `internal/gen/collections.go` (new)
- **Move**:
  - slice/map/array conversion helpers
  - recursive loop generation helpers (`generateCollectionLoop`, value conversion helpers)

Acceptance criteria:
- Nested collection examples remain stable and compile.

## Milestone 5 — Simplify `generator.go` orchestration

After milestones 1–4, `generator.go` should focus on:
- `Generator` struct/config
- `Generate(plan)` entry point
- file emission loop + gofmt
- missing types/transforms emission (optional extraction later)

Acceptance criteria:
- `generator.go` is primarily orchestration (< ~400 LOC target is reasonable).

## Expected outcome

- Core generator logic becomes modular and testable.
- Determinism is easier to preserve (type formatting/imports are isolated).
- `generator.go` becomes a readable coordinator instead of the single place where everything happens.
