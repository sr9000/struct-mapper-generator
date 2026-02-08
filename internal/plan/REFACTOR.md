# Refactoring Plan: `internal/plan`

This document outlines an incremental plan to simplify and refactor the resolution and export logic in `internal/plan`.

## Scope and constraints

- Depends on `internal/mapping` semantics; avoid changing mapping contracts during plan refactors.
- Keep resolver results stable for existing integration tests.
- Keep suggestion export deterministic (ordering and formatting should be stable).

## Current status

Candidates for refactoring due to size and mixed responsibilities:

- `export.go` (~780 LOC): mixes suggestion modeling and YAML AST construction (comments, ordering).
- `resolver.go` (~1402 LOC): monolithic coordinator (iteration, recursion, automatching, strategy selection, virtual types, dependency ordering).

## Milestone 0 — Lock behavior (recommended first)

Add/confirm unit tests for the most fragile seams:

- Strategy selection:
  - hint-based overrides
  - kind-based determination
  - incompatible/transform-needed paths
- Type walking rules (pointer/leaf behavior) used by `resolveFieldType`.
- Suggestion export determinism (stable ordering).

Acceptance criteria:
- `go test ./...` passes.

## Milestone 1 — Split `resolver.go` by responsibility

Keep external APIs stable; this should primarily be mechanical extraction.

### 1) Strategy selection module (do first)
- **Target file**: `internal/plan/strategy_selector.go` (new)
- **Move**:
  - `determineStrategy*` functions
  - `determineStrategyFromCandidate`
  - strategy explanation constants (e.g. `explSliceMap`, `explPointerWrap`)
  - keep `resolveFieldType` close to these rules

Why first: it’s a well-defined seam and is easy to unit test independently.

### 2) Automatch module
- **Target file**: `internal/plan/automatch.go` (new)
- **Move**:
  - `autoMatchRemainingFields` and candidate scoring/selection

### 3) Virtual types module
- **Target file**: `internal/plan/virtual_types.go` (new)
- **Move**:
  - `preCreateVirtualTypes`, `createVirtualTargetType`, and any purely-virtual helpers

### 4) Dependency ordering module
- **Target file**: `internal/plan/dependencies.go` (new)
- **Move**:
  - `populateExtraTargetDependencies` and related ordering/cycle helpers

### 5) Resolver core left behind
After extraction, `resolver.go` (or `resolver_core.go`) should focus on:
- orchestration (`Resolve`, `resolveTypeMapping`)
- recursion/caching (`resolvedPairs`)
- calling sub-components

Acceptance criteria:
- Integration tests unchanged.
- New unit tests for `strategy_selector` and pointer/leaf rules.

Risk hotspots (add/keep tests here):
- virtual type creation interacting with strategy selection
- dependency ordering affecting assignment order

## Milestone 2 — Split `export.go` into model vs YAML rendering

### 1) Suggestions model
- **Target file**: `internal/plan/suggestions_model.go` (new)
- **Responsibility**: build an intermediate “suggestions model” (could be `mapping.MappingFile` or a dedicated struct) from resolved plan data.

### 2) YAML builder
- **Target file**: `internal/plan/suggestions_yaml.go` (new)
- **Responsibility**: transform the model into `yaml.Node` (comments, formatting, ordering).

Optional improvement:
- Introduce a small `YAMLBuilder` struct to reduce parameter threading and isolate formatting rules.

Acceptance criteria:
- Export remains deterministic.
- Snapshot/golden-style tests exist for ordering (even if minimal).

## Expected outcome

- Resolver becomes readable: orchestration + small focused modules.
- Strategy selection and automatch become unit-testable without running full resolve.
- Export logic becomes two-phase (model then rendering), reducing YAML AST complexity in core logic.
