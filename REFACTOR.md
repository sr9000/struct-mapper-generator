# Refactoring Plan: `internal/mapping`

This document outlines a plan to simplify and refactor the core mapping types and logic in `internal/mapping`.

## Current Status

The following files identify as candidates for refactoring due to size, complexity, and mixed responsibilities:

-   **`internal/mapping/validate.go`** (350 lines): Contains structural validation logic. `validateFieldMapping` has high cyclomatic complexity.
-   **`internal/mapping/loader.go`** (421 lines): Mixes file I/O, YAML parsing orchestration, custom YAML unmarshaling logic for multiple types, and path parsing logic.
-   **`internal/mapping/schema.go`** (689 lines): Contains struct definitions, methods with significant business logic (e.g., `GetEffectiveHint`), and more custom YAML logic (`ExtraVals`).

## Objectives

1.  **Single Responsibility Principle**: Separate YAML mechanics, data structure definitions, and business rules.
2.  **Reduce Complexity**: Break down large functions (especially in validation).
3.  **Improve Readability**: Make the codebase easier to navigate.

## Proposed Steps

### 1. Extract YAML Serialization Logic
Move all custom `UnmarshalYAML` and `MarshalYAML` implementations to a dedicated file. This declutters the schema and loader files.

*   **Target File**: `internal/mapping/yaml_types.go` (new)
*   **Actions**:
    *   Move `StringOrArray` methods from `loader.go`.
    *   Move `FieldRefArray` methods from `loader.go`.
    *   Move `ExtraVals` methods from `schema.go`.
    *   Move `StringArray` methods from `schema.go`.

### 2. Extract Path Parsing Logic
The logic for parsing string paths (e.g. `Items[].ID`) is a distinct domain concept.

*   **Target File**: `internal/mapping/path.go` (new)
*   **Actions**:
    *   Move `ParsePath` and `ParsePaths` family of functions from `loader.go`.
    *   Move `isValidIdent` and helper constraints from `loader.go`.

### 3. Simplify `loader.go`
After steps 1 and 2, `loader.go` should only contain:
*   `LoadFile` / `WriteFile` (I/O)
*   `Parse` (YAML entry point)
*   `Normalize...` functions (Preparation logic)

### 4. Refactor `schema.go` Logic
Splitting definitions from heavy logic makes the schema easier to read.

*   **Target File**: `internal/mapping/hints.go` (new) or `internal/mapping/schema_logic.go`
*   **Actions**:
    *   Move `GetEffectiveHint` (logic-heavy) out of `schema.go`.
    *   Move `Cardinality` logic if it grows.

### 5. Decompose `validate.go`
Break down the monolithic validation functions.

*   **Actions**:
    *   Refactor `validateFieldMapping` (which is currently marked with `//nolint:gocyclo`) into sub-validators:
        *   `validateTargets(...)`
        *   `validateSources(...)`
        *   `validateExtra(...)`
    *   **Move Component**: `ResolveTypeID` is strictly an operation on the `TypeGraph`. consider moving it to `internal/analyze/graph.go` or similar, making it `graph.ResolveType(id)`.
    *   **Move Component**: `validatePathAgainstType` validates a `FieldPath` against a `TypeInfo`. This could reside in `internal/analyze` or a new `internal/mapping/validator_helpers.go`.

## Expected Outcome

*   **`loader.go`**: Reduced from ~420 lines to ~100 lines.
*   **`schema.go`**: Reduced from ~690 lines to ~400 lines (pure definitions).
*   **`validate.go`**: Similar size but composed of many small, testable functions instead of massive loops; cleaner dependencies if graph helpers are moved.
