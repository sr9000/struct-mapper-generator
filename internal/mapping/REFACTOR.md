# Refactoring Plan: `internal/mapping`

This document outlines a safe, incremental plan to simplify and refactor the core mapping types and logic in `internal/mapping`.

## Scope and constraints

- **Must stay compatible** with the existing YAML schema (inputs, defaults, normalization).
- **Must remain deterministic** when marshaling mapping files.
- **Must not change semantics** of path validation and type walking unless explicitly intended and tested.

## Current status

Candidates for refactoring due to size/complexity and mixed responsibilities:

- `validate.go` (~350 LOC): structural validation logic; `validateFieldMapping` is high cyclomatic complexity.
- `loader.go` (~421 LOC): mixes file I/O, YAML parsing orchestration, custom YAML marshal/unmarshal, and path parsing.
- `schema.go` (~689 LOC): struct definitions plus logic-heavy methods and additional custom YAML logic.

## Milestone 0 — Lock behavior (recommended first)

Add/confirm unit tests around the seam boundaries before code motion:

- Path parsing: `ParsePath` / `ParsePaths`.
- Hint resolution: `FieldRefArray.GetEffectiveHint`.
- YAML round-trips for custom YAML types (where ordering matters).
- Validation edge cases:
  - pointer deref rules
  - slice/map path segments
  - transform-name validation rules

Acceptance criteria:
- `go test ./...` passes.

## Milestone 1 — Move-only extractions (lowest risk)

### 1) Extract YAML serialization logic
Move custom `UnmarshalYAML` / `MarshalYAML` implementations to a dedicated file.

- **Target file**: `internal/mapping/yaml_types.go` (new)
- **Move**:
  - `StringOrArray` methods from `loader.go`
  - `FieldRefArray` methods (and helpers like `parseFieldRefFromMap`) from `loader.go`
  - `ExtraVals` methods from `schema.go`
  - `StringArray` methods from `schema.go`

### 2) Extract path parsing logic
Path parsing is its own domain problem; keep it isolated and testable.

- **Target file**: `internal/mapping/path.go` (new)
- **Move**:
  - `ParsePath`, `ParsePaths`, `ParsePathsFromRefs` from `loader.go`
  - `isValidIdent` and related helpers from `loader.go`

### 3) Simplify `loader.go`
After steps (1) and (2), `loader.go` should focus on orchestration:
- `LoadFile` / `WriteFile` (I/O)
- `Parse` (YAML entry point)
- `Normalize...` functions (preparation logic)

Acceptance criteria:
- No behavior changes; diffs should be mechanical “move only”.
- `go test ./...` passes.

## Milestone 2 — Reduce complexity in `schema.go` and `validate.go`

### 4) Refactor schema logic (optional split)
Split definitions from heavier business logic.

- **Target file**: `internal/mapping/schema_logic.go` (new) or `hints.go`
- **Move**:
  - `GetEffectiveHint` and other logic-heavy helpers out of `schema.go`

### 5) Decompose `validate.go`
Break down the monolithic validation functions into smaller validators.

- Keep `Validate(...)` as the entrypoint.
- Refactor `validateFieldMapping` (currently `//nolint:gocyclo`) into helpers like:
  - `validateTargets(...)`
  - `validateSources(...)`
  - `validateExtra(...)`

Design note:
- `validatePathAgainstType` is performance- and semantics-critical. If you move it, keep the exact pointer/leaf handling and add regression tests.

### 6) Consider relocating `ResolveTypeID`
`ResolveTypeID` is fundamentally “string id -> type info” and may fit better in `internal/analyze`.

- **Low-risk option**: move to `internal/mapping/type_resolver.go` first.
- **Architectural option**: move into `internal/analyze` (e.g., `TypeGraph.ResolveTypeID`) and keep a wrapper in `mapping` for compatibility.

Acceptance criteria:
- Validation semantics unchanged for existing integration tests.
- Add targeted unit tests for edge cases in the extracted helpers.

## Expected outcome

- `loader.go`: reduced substantially and focused on orchestration.
- `schema.go`: closer to “types + light helpers”, with heavy logic extracted.
- `validate.go`: composed of small, testable functions.
