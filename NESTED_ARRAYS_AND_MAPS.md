# Nested Arrays and Maps

This document describes the plan to implement support for nested arrays, slices, and maps.

## Feature Description
The generator should support arbitrary nesting of collections:
- `[]T`, `[][]T`, `[][][]T`
- `map[K]V`
- `map[K]map[K2]V`
- `[]map[K]V`, `map[K][]V`

It should handle type conversion for leaf elements (e.g. `[]APIStruct` -> `[]DomainStruct`).

## New Feature & Implementation Logic

### Logic Updates
The current `generateSliceLoopCode` in `internal/gen/generator.go` produces a standard `for` loop string. It is not recursive and doesn't handle Maps.

We need to introduce a recursive generation strategy.

### Sub Functions
New methods in `Generator` struct (`internal/gen/generator.go`):

1.  `buildCollectionMapping(m, pair, imports)`: Replacement/Extension of `buildSliceMapping`. Determines if we are dealing with slices or maps and initiates the loop generation.
2.  `generateCollectionLoop(srcVar, tgtVar, srcType, tgtType, depth, ...)`:
    - **Recursion**: If elements are collections, it recursively generates an inner loop.
    - **Initialization**: Handles `make([]T, len)` or `make(map[K]V, len)`.
    - **Loop Variables**: Uses `depth` to generate unique index variables (e.g. `i_0`, `i_1`, `k_0, v_0`).
    - **Assignment**: Generates the assignment `tgtVar[index] = convertedValue`.
3.  `buildElementConversion(...`: Updated to handle simple conversions and call `nestedFunctionName` for structs. For collections, it shouldn't return a string, but the `generateCollectionLoop` should handle the assignment logic itself.

## Tests

### Integration Tests
New examples will act as integration tests.
- `examples/nested/`: Focus on `[][]T`.
- `examples/maps/`: Focus on `map[string]T`.
- `examples/nested-mixed/`: Focus on `map[string][]T`.

### Unit Tests
- `internal/gen/generator_test.go`: Add test cases for `buildCollectionMapping` with nested types.

## New Examples with run.sh

We will create the following structures in `examples/`:

### 1. Nested Arrays (`examples/nested`)
**Source:**
```go
type Source struct {
    Grid [][]int
}
```
**Target:**
```go
type Target struct {
    Grid [][]int
}
```

### 2. Maps (`examples/maps`)
**Source:**
```go
type Source struct {
    Tags map[string]string
}
```
**Target:**
```go
type Target struct {
    Tags map[string]string
}
```

### 3. Deep Nesting (`examples/nested-mixed`)
**Source:**
```go
type Source struct {
    Data map[string][]Inner
}
type Inner struct { Name string }
```

## Step-by-Step Implementation Plan

- [ ] **Step 1: Map Support (Level 1)**
    - Modify `internal/gen/generator.go` to recognize `TypeKindMap`.
    - Implement simple `map` copying loop.

- [ ] **Step 2: Refactor Loop Generation**
    - Create `generateCollectionLoop` to handle both slices and maps.
    - Implement variable naming strategy based on depth (`i0`, `i1`, etc.).

- [ ] **Step 3: Recursive Nesting**
    - Update `generateCollectionLoop` to detect if the element/value is a collection.
    - If collection, recurse `generateCollectionLoop` for the inner element.

- [ ] **Step 4: Struct Conversion in Maps/Slices**
    - Ensure `buildElementConversion` works correctly when invoked from deep recursion.

- [ ] **Step 5: Create Reproductions**
    - Create `examples/nested` with failing `run.sh` (generation produces invalid code or TODOs).
    - Create `examples/maps` with failing `run.sh`.

- [ ] **Step 6: Verification**
    - Run all example scripts (`run.sh`).
    - Verify `internal/gen` tests passes.
