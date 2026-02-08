# Refactoring Plan: `internal/mapping`

This document outlines a safe, incremental plan to simplify and refactor the core mapping types and logic in `internal/mapping`.

## Scope and constraints

- **Must stay compatible** with the existing YAML schema (inputs, defaults, normalization).
- **Must remain deterministic** when marshaling mapping files.
- **Must not change semantics** of path validation and type walking unless explicitly intended and tested.

## Current status ✅ COMPLETED

All milestones have been completed. Final file structure:

| File | Lines | Description |
|------|-------|-------------|
| `loader.go` | ~95 | I/O orchestration: LoadFile, Parse, Marshal, WriteFile, Normalize |
| `schema.go` | ~521 | Type definitions and light helper methods |
| `yaml_types.go` | ~406 | All YAML marshal/unmarshal implementations |
| `path.go` | ~113 | Path parsing: ParsePath, ParsePaths, isValidIdent |
| `validate.go` | ~177 | Main Validate entry point and validatePathAgainstType |
| `validate_helpers.go` | ~172 | Decomposed validators: validateTargets, validateSources, validateTransform, validateExtra |
| `type_resolver.go` | ~63 | ResolveTypeID function |

Original candidates and their reduction:
- ~~`validate.go` (~350 LOC)~~ → split into `validate.go` (~177) + `validate_helpers.go` (~172) + `type_resolver.go` (~63)
- ~~`loader.go` (~421 LOC)~~ → split into `loader.go` (~95) + `yaml_types.go` (partial) + `path.go` (~113)
- ~~`schema.go` (~689 LOC)~~ → reduced to `schema.go` (~521) + `yaml_types.go` (partial)

## Milestone 0 — Lock behavior ✅ COMPLETED

Existing tests confirmed to pass before any code motion.

Acceptance criteria:
- ✅ `go test ./...` passes.

## Milestone 1 — Move-only extractions ✅ COMPLETED

### 1) Extract YAML serialization logic ✅
Moved custom `UnmarshalYAML` / `MarshalYAML` implementations to a dedicated file.

- **Target file**: `internal/mapping/yaml_types.go` ✅
- **Moved**:
  - ✅ `StringOrArray` methods from `loader.go`
  - ✅ `FieldRefArray` methods (and helpers like `parseFieldRefFromMap`) from `loader.go`
  - ✅ `ExtraVals` methods from `schema.go`
  - ✅ `StringArray` methods from `schema.go`
  - ✅ `FieldRefOrString` methods from `schema.go`
  - ✅ `ArgDefArray` methods from `schema.go`

### 2) Extract path parsing logic ✅
Path parsing is now isolated and testable.

- **Target file**: `internal/mapping/path.go` ✅
- **Moved**:
  - ✅ `ParsePath`, `ParsePaths`, `ParsePathsFromRefs` from `loader.go`
  - ✅ `isValidIdent` and related helpers from `loader.go`

### 3) Simplify `loader.go` ✅
After steps (1) and (2), `loader.go` focuses on orchestration:
- ✅ `LoadFile` / `WriteFile` (I/O)
- ✅ `Parse` (YAML entry point)
- ✅ `Normalize...` functions (preparation logic)

Acceptance criteria:
- ✅ No behavior changes; diffs are mechanical "move only".
- ✅ `go test ./...` passes.

## Milestone 2 — Reduce complexity in `schema.go` and `validate.go` ✅ COMPLETED

### 4) Refactor schema logic (optional split) — SKIPPED
The `GetEffectiveHint` method is closely tied to `FieldRefArray` type, so moving it would break the type's encapsulation. Kept in place.

### 5) Decompose `validate.go` ✅
Broke down the monolithic validation functions into smaller validators.

- **Target file**: `internal/mapping/validate_helpers.go` ✅
- ✅ Keep `Validate(...)` as the entrypoint in `validate.go`
- ✅ Refactored `validateFieldMapping` (removed `//nolint:gocyclo`) into helpers:
  - ✅ `validateTargets(...)`
  - ✅ `validateSources(...)` (includes `isRequiredArg` helper)
  - ✅ `validateTransform(...)`
  - ✅ `validateExtra(...)`

### 6) Relocate `ResolveTypeID` ✅
Moved to dedicated file as the low-risk option.

- **Target file**: `internal/mapping/type_resolver.go` ✅

Acceptance criteria:
- ✅ Validation semantics unchanged for existing integration tests.
- ✅ All tests pass, including integration tests.

## Expected outcome ✅ ACHIEVED

- ✅ `loader.go`: reduced from ~421 to ~95 LOC, focused on orchestration.
- ✅ `schema.go`: reduced from ~689 to ~521 LOC, closer to "types + light helpers".
- ✅ `validate.go`: reduced from ~350 to ~177 LOC, composed of small, testable functions.
