# Refactoring Plan: `internal/gen`

This document outlines a plan to simplify and refactor the code generation logic in `internal/gen`.

## Current Status

The primary candidate for refactoring is:

- **`internal/gen/generator.go`** (1854 lines): A monolithic file handling template data preparation, strategy application, collection loop generation, type reference formatting, and file I/O.

## Objectives

1. **Decompose Monoliths**: Break down `Generator` into specialized components.
2. **Isolate Template Logic**: Separate data preparation from code generation mechanics.
3. **Encapsulate Domain Logic**: Move strategies (slice/map/pointer logic) into dedicated handlers.

## Proposed Steps

### 1. Extract Template Data Building
The logic for converting a `ResolvedTypePair` into `templateData` is complex and mixes type analysis with view logic.

* **Target File**: `internal/gen/template_builder.go` (new)
* **Actions**:
    * Move `buildTemplateData`, `buildAssignment`, and `collectNestedCasters` here.
    * Responsibility: Prepare the data structure required by the Go templates.

### 2. Extract Strategy Application
The `switch` statement in `applyConversionStrategy` handles many disparate logic paths (pointers, nested casts, transforms).

* **Target File**: `internal/gen/strategies.go` (new)
* **Actions**:
    * Move `applyConversionStrategy` and all specific strategy methods (`applyConvertStrategy`, `applyPointerDerefStrategy`, etc.).
    * Responsibility: Mutate the `assignmentData` based on the chosen conversion strategy.

### 3. Extract Collection Generation
Generating loops for slices and maps involves recursive logic for nested collections.

* **Target File**: `internal/gen/collections.go` (new)
* **Actions**:
    * Move `buildSliceMapping`, `buildMapMapping`, and `generateCollectionLoop`.
    * Move `buildValueConversion` helper methods.
    * Responsibility: Generate Go code strings for iterating and converting collection types.

### 4. Extract Type Formatting
Formatting type strings (handling pointers, slices, packages imports) is a utility concern.

* **Target File**: `internal/gen/type_formatter.go` (new) or `internal/gen/types.go`
* **Actions**:
    * Move `typeRef`, `typeRefString`, `getPkgName`, and import management logic.
    * Responsibility: consistently format Go type representations and manage import aliases.

### 5. Simplify `generator.go`
After steps 1-4, `generator.go` should focus on the high-level orchestration.

* **Remaining content**:
    * `Generator` struct and config.
    * `Generate(plan)` entry point.
    * File generation loop.
    * Handling of missing types/transforms (which could also be extracted later).

## Expected Outcome

* **`generator.go`**: Reduced from ~1850 lines to < 400 lines (orchestration only).
* **`strategies.go`**: Encapsulates the specific "how-to" of code generation for each mapping type.
* **`collections.go`**: Isolates the complex recursive loop generation logic.
* **`template_builder.go`**: clear separation between the "Plan" model and the "View" model.
