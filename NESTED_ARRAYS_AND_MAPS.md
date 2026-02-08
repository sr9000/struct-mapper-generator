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

- [ ] **Step 1: Create Reproductions**
    - Create `examples/nested` with failing `run.sh`.
    - Create `examples/maps` with failing `run.sh`.
    - Create `examples/nested-mixed` with failing `run.sh`.

- [ ] **Step 2: Map Support (Level 1)**
    - Modify `internal/gen/generator.go` to recognize `TypeKindMap`.
    - Implement simple `map` copying loop.

- [ ] **Step 3: Refactor Loop Generation**
    - Rename `generateSliceLoopCode` to `generateCollectionLoop`.
    - Add `depth` parameter for unique variable names (`i_0`, `k_0`, etc.).
    - Allow handling both Maps and Slices in one consistent structure.

- [ ] **Step 4: Recursive Nesting**
    - Update `generateCollectionLoop` to recurse if the element is also a collection.
    - Handles `[][]T` and `map[K][]V`.

- [ ] **Step 5: Struct Conversion in Maps/Slices**
    - Ensure `buildElementConversion` correctly delegates to `nestedFunctionName` or recursive calls.

- [ ] **Step 6: Verification**
    - Run all example scripts covering all scenarios.

## Step 1: Create Reproductions

### What to do
Create three new directories in `examples/`: `nested`, `maps`, and `nested-mixed`.
For each directory, create:
1.  `source.go`: Defining the source types with nested arrays/maps.
2.  `target.go`: Defining the target types.
3.  `map.yaml`: The mapping configuration.
4.  `run.sh`: A script to run the generator and verify the output.

### Definition of Done
- Directories `examples/nested`, `examples/maps`, `examples/nested-mixed` exist.
- `run.sh` in each directory executes the generator.
- The generation likely fails or produces incorrect code (this is expected at this stage).

### Suggested files set to work
- `examples/nested/*`
- `examples/maps/*`
- `examples/nested-mixed/*`

### Ideas what functions/structs needs to be patched/created
- Copy headers/boilerplate from `examples/basic`.
- Ensure `map.yaml` points to the correct package names.

## Step 2: Map Support (Level 1)

### What to do
Extend the generator to handle `map` types, which are currently ignored or malformed.
Implement a basic strategy for `Plan.StrategyMap` (or reuse reuse `SliceMap` if appropriate, but likely need a distinction).
Ensure the `analyze` package correctly identifies `TypeKindMap`.

### Definition of Done
- `examples/maps` generates compilable code for `map[string]string` -> `map[string]string`.
- The generated code uses `make(map[...])` and iterates using `for k, v := range ...`.

### Suggested files set to work
- `internal/gen/generator.go`

### Ideas what functions/structs needs to be patched/created
- Patch `applyConversionStrategy` in `internal/gen/generator.go`:
    - Add logic for `analyze.TypeKindMap`.
- Create `buildMapMapping` (similar to `buildSliceMapping`).
- Verify `internal/analyze/types.go` has `TypeKindMap`.

## Step 3: Refactor Loop Generation

### What to do
Refactor the existing `generateSliceLoopCode` and the new map logic into a unified `generateCollectionLoop`.
Introduce a `depth` parameter to handle variable naming in nested loops (e.g., `i_0` for outer, `i_1` for inner).
This step prepares the codebase for recursion.

### Definition of Done
- `examples/arrays` (existing) still passes.
- `examples/maps` still passes.
- Source code in `internal/gen` uses `generateCollectionLoop` for both.

### Suggested files set to work
- `internal/gen/generator.go`

### Ideas what functions/structs needs to be patched/created
- Create `func (g *Generator) generateCollectionLoop(srcField, tgtField string, srcType, tgtType *analyze.TypeInfo, imports map[string]importSpec, depth int) string`
- Use `fmt.Sprintf("i_%d", depth)` for loop variables.

## Step 4: Recursive Nesting

### What to do
Implement recursion in `generateCollectionLoop`.
When the element type of the collection is itself a Slice or Map, call `generateCollectionLoop` again with `depth + 1`.

### Definition of Done
- `examples/nested` (`[][]int`) compiles and runs correctly.
- `examples/nested-mixed` (`map[string][]Inner`) generates correct nested loops.

### Suggested files set to work
- `internal/gen/generator.go`

### Ideas what functions/structs needs to be patched/created
- Update `generateCollectionLoop`:
    - Logic: `if isCollection(srcElem) { body = g.generateCollectionLoop(..., depth+1) }`

## Step 5: Struct Conversion in Maps/Slices

### What to do
Ensure that when the recursion hits a leaf node that is a Struct (not a primitive), it correctly calls the conversion function (e.g., `APIBoxToDomainBox`).
This might require updating `buildElementConversion` to be aware of the context or just function correctly within the loop body.

### Definition of Done
- `examples/nested-mixed` fully works including struct conversion.
- `generated/` code shows `Target = Convert(Source)` calls inside loops.

### Suggested files set to work
- `internal/gen/generator.go`

### Ideas what functions/structs needs to be patched/created
- Patch `buildElementConversion`:
    - It must handle `TypeKindStruct` by looking up the generated matching function from the Plan/Registry.

## Step 6: Verification

### What to do
Run all integration tests and ensure no regressions.
Check if any new unit tests are needed in `internal/gen`.

### Definition of Done
- `make test` passes.
- All `examples/**/run.sh` pass.

### Suggested files set to work
- `internal/gen/generator_test.go`
- `examples/`

### Ideas what functions/structs needs to be patched/created
- Add specific unit tests for `generateCollectionLoop` edge cases (e.g. nil maps, empty slices).
