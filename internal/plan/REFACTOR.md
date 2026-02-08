# Refactoring Plan: `internal/plan`

This document outlines a plan to simplify and refactor the resolution and export logic in `internal/plan`.

## Current Status

The following files identify as candidates for refactoring due to size and monolithic design:

- **`internal/plan/export.go`** (780 lines): Handles both data conversion and detailed YAML AST construction for suggestions.
- **`internal/plan/resolver.go`** (1402 lines): A monolithic coordinator managing iteration, recursion, matching, and strategy selection.

## Objectives

1. **Decompose Monoliths**: Break down the large `Resolver` and `Export` logic into focused components.
2. **Encapsulation**: Hide complex YAML AST manipulation and matching heuristics behind simpler interfaces.
3. **Improve Testability**: Make individual components like `AutoMatcher` and `StrategySelector` independently testable.

## Proposed Steps

### 1. Refactor `internal/plan/export.go` (Suggestion Generation)
The export logic is complex because it builds YAML nodes with comments based on match quality.

* **Target File**: `internal/plan/suggestion_generator.go` (new)
* **Actions**:
    * Extract `GenerateSuggestions(plan)` which returns a `mapping.MappingFile` or similar intermediate structure.
    * Create a `SuggestionBuilder` or `YAMLBuilder` struct to encapsulate the verbose `yaml.Node` construction.
    * Separate logic like `appendIgnore` (which formats rejection reasons) from the raw YAML production.

### 2. Refactor `internal/plan/resolver.go` (Automatching)
Automatching is a large chunk of the resolver.

* **Target File**: `internal/plan/automatch.go` (new)
* **Actions**:
    * Extract `AutoMatcher` struct.
    * Move `autoMatchRemainingFields` logic here. 
    * Responsibility: Given source/target types and existing mappings, identify auto-match candidates using `internal/match`.

### 3. Refactor `internal/plan/resolver.go` (Strategy Selection)
Determining *how* to convert one type to another is a distinct concern.

* **Target File**: `internal/plan/strategy_selector.go` (new)
* **Actions**:
    * Move `determineStrategyFromCandidate` and strategy-related constants (e.g., `explSliceMap`, `explPointerWrap`).
    * Responsibility: Map `match.Candidate` results to `ConversionStrategy`.

### 4. Refactor `internal/plan/resolver.go` (Virtual Types)
Managing target type generation.

* **Target File**: `internal/plan/virtual_types.go` (new)
* **Actions**:
    * Move `preCreateVirtualTypes` and `createVirtualTargetType`.
    * Responsibility: Populate the `TypeGraph` with generated types before or during resolution.

## Expected Outcome

* **`resolver.go`**: Reduced significantly by delegating to specialized components. The core loop will become much clearer.
* **`export.go`**: Cleanly separated into suggestion logic and YAML output logic.
* **Testability**: Heuristics for matching and strategy selection can be tested without running the full resolution pipeline.
