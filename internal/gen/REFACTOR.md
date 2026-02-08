# Refactoring Plan: `internal/gen`

This document outlines an incremental plan to simplify and refactor the code generation logic in `internal/gen`.

## Scope and constraints

- `internal/gen` depends on `internal/plan` output shape and strategy semantics; refactor `plan` first.
- Generated output must remain:
  - **deterministic** (stable files, stable import aliases/order, stable statement order)
  - **compilable** across all examples/tests

## Current status

**COMPLETED** - All milestones have been achieved. The refactoring was completed on 2026-02-08.

### Summary of changes:
- `generator.go` reduced from ~1854 LOC to ~471 LOC (primarily orchestration)
- Created 4 new focused files:
  - `type_formatter.go` (~378 LOC): type formatting, import management, zero values
  - `template_builder.go` (~498 LOC): template data building, assignments, dependencies
  - `strategies.go` (~269 LOC): conversion strategy application
  - `collections.go` (~334 LOC): collection loop generation

## Milestone 0 — Lock behavior ✅ COMPLETE

Tests confirmed stable before and after refactoring:
- `go test ./...` passes
- All 16 example scenarios compile and pass

## Milestone 1 — Extract type formatting + import management ✅ COMPLETE

- **Created file**: `internal/gen/type_formatter.go`
- **Moved**:
  - `typeRef`, `typeRefString`, `getPkgName`
  - `addImport`, `getFieldTypeString`, `wrapConversion`
  - `zeroValue`, `zeroValueForType`, `zeroValueForBasicType`
  - `typesIdentical`, `typesConvertible`, `typeInfoFromString`
  - `getFieldTypeInfo`, `findFieldInStruct`, `getFieldType`
  - `importSpec` type definition

## Milestone 2 — Extract template/view-model building ✅ COMPLETE

- **Created file**: `internal/gen/template_builder.go`
- **Moved**:
  - `buildTemplateData`, `buildAssignment`, `collectNestedCasters`
  - `orderAssignmentsByDependencies`, `processStructDefinition`
  - `targetFieldExpr`, `sourceFieldExpr`, `buildTransformArgs`
  - `getRequiredArgType`, `identifyMissingTransforms`
  - Type definitions: `templateData`, `extraArg`, `assignmentData`, `nestedCasterRef`

## Milestone 3 — Extract strategy application ✅ COMPLETE

- **Created file**: `internal/gen/strategies.go`
- **Moved**:
  - `applyConversionStrategy` and all strategy-specific helpers:
    - `applyConvertStrategy`, `applyPointerDerefStrategy`
    - `applyPointerWrapStrategy`, `applyPointerNestedCastStrategy`
    - `applyNestedCastStrategy`, `applyTransformStrategy`
  - `buildSliceMapping`, `buildMapMapping`, `buildExtraArgsForNestedCall`

## Milestone 4 — Extract collection generation ✅ COMPLETE

- **Created file**: `internal/gen/collections.go`
- **Moved**:
  - `buildCollectionMapping`, `generateCollectionLoop`
  - `generateSliceArrayLoop`, `generateMapLoop`
  - `isCollection`, `getSliceElementType`, `getMapKeyType`, `getMapValueType`
  - `buildValueConversionWithExtra`, `buildValueConversion`

## Milestone 5 — Simplify `generator.go` orchestration ✅ COMPLETE

`generator.go` now contains only:
- `Generator` struct/config types
- `NewGenerator`, `DefaultGeneratorConfig`
- `Generate(plan)` entry point
- `generateTypePair`, `generateMissingTransformsFile`, `generateMissingTypesFiles`
- Naming helpers: `filename`, `functionName`, `nestedFunctionName`, `capitalize`
- Template definitions

Final line count: ~471 LOC (target was < ~400 LOC, close enough considering templates)

## Expected outcome ✅ ACHIEVED

- Core generator logic is now modular and testable.
- Determinism is preserved (all tests pass, all examples compile).
- `generator.go` is a readable coordinator instead of a monolithic file.
