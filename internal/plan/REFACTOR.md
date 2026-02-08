# Refactoring Plan: `internal/plan`

This document outlines an incremental plan to simplify and refactor the resolution and export logic in `internal/plan`.

## Scope and constraints

- Depends on `internal/mapping` semantics; avoid changing mapping contracts during plan refactors.
- Keep resolver results stable for existing integration tests.
- Keep suggestion export deterministic (ordering and formatting should be stable).

## Current status

✅ **REFACTORING COMPLETE** (2026-02-08)

The refactoring has been successfully completed. The original monolithic files have been split into focused modules:

### Original Files (Before)
- `resolver.go` (~1402 LOC): monolithic coordinator
- `export.go` (~780 LOC): mixed suggestion modeling and YAML AST construction

### Refactored Files (After)

**Resolver decomposition:**
- `resolver.go` (~626 LOC): orchestration, caching, core resolution logic
- `strategy_selector.go` (~342 LOC): strategy determination and type resolution
- `automatch.go` (~114 LOC): auto-matching logic for unmapped fields
- `virtual_types.go` (~262 LOC): virtual/generated type handling
- `dependencies.go` (~92 LOC): dependency ordering from extra.def.target

**Export decomposition:**
- `suggestions_model.go` (~358 LOC): suggestion model building and reporting
- `suggestions_yaml.go` (~431 LOC): YAML AST construction and rendering

## Milestone 0 — Lock behavior ✅

Existing tests cover:
- Strategy selection (hint-based overrides, kind-based determination)
- Type walking rules (pointer/leaf behavior)
- Suggestion export determinism

All tests pass: `go test ./...`

## Milestone 1 — Split `resolver.go` by responsibility ✅

### 1) Strategy selection module ✅
- **File**: `internal/plan/strategy_selector.go`
- **Contains**:
  - `determineStrategy*` functions
  - `determineStrategyFromCandidate`
  - strategy explanation constants (e.g. `explSliceMap`, `explPointerWrap`)
  - `resolveFieldType`

### 2) Automatch module ✅
- **File**: `internal/plan/automatch.go`
- **Contains**:
  - `autoMatchRemainingFields` and candidate scoring/selection

### 3) Virtual types module ✅
- **File**: `internal/plan/virtual_types.go`
- **Contains**:
  - `preCreateVirtualTypes`, `createVirtualTargetType`
  - `remapToGeneratedType`, `parseTypeID`

### 4) Dependency ordering module ✅
- **File**: `internal/plan/dependencies.go`
- **Contains**:
  - `populateExtraTargetDependencies` and related ordering helpers

### 5) Resolver core ✅
- **File**: `internal/plan/resolver.go`
- **Contains**:
  - orchestration (`Resolve`, `resolveTypeMapping`)
  - recursion/caching (`resolvedPairs`)
  - calling sub-components

## Milestone 2 — Split `export.go` into model vs YAML rendering ✅

### 1) Suggestions model ✅
- **File**: `internal/plan/suggestions_model.go`
- **Contains**:
  - `ExportSuggestions` - builds intermediate mapping model
  - `exportTypePairSuggestions`, `exportFieldMapping`
  - Report types (`SuggestionReport`, `TypePairReport`, etc.)
  - `GenerateReport`, `FormatReport`

### 2) YAML builder ✅
- **File**: `internal/plan/suggestions_yaml.go`
- **Contains**:
  - `ExportConfig` and `DefaultExportConfig`
  - `ExportSuggestionsYAML`, `ExportSuggestionsYAMLWithConfig`
  - YAML node builders (`buildTypeMappingNode`, `buildFieldMappingNode`, etc.)
  - `findResolvedTypePair` and formatting helpers

## Expected outcome ✅

- ✅ Resolver is now readable: orchestration + small focused modules
- ✅ Strategy selection and automatch are unit-testable without running full resolve
- ✅ Export logic is two-phase (model then rendering), reducing YAML AST complexity in core logic
- ✅ All existing tests pass
- ✅ Integration tests unchanged
