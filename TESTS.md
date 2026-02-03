# Test Cases for Missing Types Generation Feature

This document outlines the test cases needed for the feature that generates missing target types in their respective packages (via `missing_types.go`) instead of embedding them in the caster package.

## Overview

When a mapping specifies `generate_target: true` and the target type has a package path (e.g., `caster-generator/examples/virtual.Target`), the generator should:

1. Create the struct definition in `missing_types.go` in the **target package's directory** (not the caster output directory)
2. Correctly import the target package in the caster file
3. Use unqualified type names within `missing_types.go` (no self-referential package prefixes)
4. Use qualified type names (e.g., `virtual.Target`) in the caster file

---

## Unit Tests

### Location: `internal/gen/generator_test.go`

#### 1. Test: `TestGenerateMissingTypesFile_Basic`

**Purpose:** Verify that a single generated target type is written to `missing_types.go` in the correct package directory.

**Setup:**
- Create a mock `ResolvedMappingPlan` with:
  - Source: `testpkg.Source` (struct with fields `ID string`, `Name string`)
  - Target: `testpkg.Target` (generated, `generate_target: true`)
  - Mapping: `ID -> ID`, `Name -> Label`

**Assertions:**
- `GeneratedFile` list contains a file with path containing `missing_types.go`
- The file content has `package testpkg`
- The file contains `type Target struct`
- The file contains field `ID string`
- The file contains field `Label string`
- No `testpkg.` prefix in the struct definition

---

#### 2. Test: `TestGenerateMissingTypesFile_MultipleTypes`

**Purpose:** Verify that multiple generated target types in the same package are combined into one `missing_types.go`.

**Setup:**
- Create a mock plan with:
  - `Source1 -> Target1` (generated)
  - `Source2 -> Target2` (generated)
  - Both targets in the same package `testpkg`

**Assertions:**
- Only one `missing_types.go` file is generated
- File contains both `type Target1 struct` and `type Target2 struct`
- Both structs are in `package testpkg`

---

#### 3. Test: `TestGenerateMissingTypesFile_CrossPackageReference`

**Purpose:** Verify that when a generated type references another generated type in the same package, no import or prefix is added.

**Setup:**
- `Source` has field `Items []*SourceItem`
- `Target` (generated) has field `Items []*TargetItem`
- `TargetItem` is also generated in the same package

**Assertions:**
- `missing_types.go` contains `Items []*TargetItem` (not `[]*testpkg.TargetItem`)
- No self-import of `testpkg`

---

#### 4. Test: `TestGenerateMissingTypesFile_ExternalTypeReference`

**Purpose:** Verify that when a generated type references an external type, the import is added to `missing_types.go`.

**Setup:**
- `Source` has field `CreatedAt time.Time`
- `Target` (generated) has field `CreatedAt time.Time`

**Assertions:**
- `missing_types.go` contains `import "time"`
- Field is `CreatedAt time.Time`

---

#### 5. Test: `TestGenerateMissingTypesFile_DifferentPackages`

**Purpose:** Verify that generated types in different packages produce separate `missing_types.go` files.

**Setup:**
- `pkg1.Source -> pkg1.Target` (generated)
- `pkg2.Source -> pkg2.Target` (generated)

**Assertions:**
- Two `missing_types.go` files are generated
- One has `package pkg1`, the other has `package pkg2`
- Each file is in the correct directory

---

#### 6. Test: `TestGenerateMissingTypesFile_NoPackagePath`

**Purpose:** Verify that when target has no package path (empty), the struct is embedded in the caster file (original behavior).

**Setup:**
- `Source -> Target` with `generate_target: true`
- Target has empty `PkgPath`

**Assertions:**
- No `missing_types.go` file is generated
- Caster file contains `type Target struct` inline

---

#### 7. Test: `TestCasterFile_ImportsGeneratedType`

**Purpose:** Verify that the caster file correctly imports the package containing the generated type.

**Setup:**
- `testpkg.Source -> testpkg.Target` (generated)
- Target struct is in `missing_types.go`

**Assertions:**
- Caster file has `import testpkg "path/to/testpkg"`
- Caster function uses `testpkg.Target` as return type
- Caster function creates `out := testpkg.Target{}`

---

#### 8. Test: `TestTypeRefString_ContextPackagePath`

**Purpose:** Verify that `typeRefString` omits package prefix when `contextPkgPath` matches the type's package.

**Setup:**
- Set `g.contextPkgPath = "my/pkg"`
- Call `typeRefString` for type with `PkgPath = "my/pkg"`

**Assertions:**
- Returns `"TypeName"` (no prefix)
- No import added

**Setup 2:**
- Set `g.contextPkgPath = "my/pkg"`
- Call `typeRefString` for type with `PkgPath = "other/pkg"`

**Assertions:**
- Returns `"other.TypeName"`
- Import added for `other/pkg`

---

### Location: `internal/analyze/loader_test.go`

#### 9. Test: `TestPackageInfo_Dir`

**Purpose:** Verify that `PackageInfo.Dir` is populated with the physical directory path.

**Setup:**
- Load a known test package

**Assertions:**
- `PackageInfo.Dir` is not empty
- `PackageInfo.Dir` is an absolute path
- Directory contains `.go` files

---

## Integration Tests

### Location: `internal/gen/missing_types_integration_test.go` (new file)

#### 10. Test: `TestIntegration_VirtualExample`

**Purpose:** End-to-end test using the `examples/virtual` scenario.

**Steps:**
1. Load packages `caster-generator/examples/virtual`
2. Load mapping from `examples/virtual/stages/stage1_mapping.yaml`
3. Run generator
4. Verify output files

**Assertions:**
- `missing_types.go` is generated in `examples/virtual/`
- Contains `type Target struct` and `type TargetItem struct`
- Caster files import `virtual` package
- All files compile successfully

---

#### 11. Test: `TestIntegration_MixedGeneratedAndExisting`

**Purpose:** Test scenario where some targets are generated and some exist.

**Setup:**
- `Source1 -> ExistingTarget` (not generated)
- `Source2 -> GeneratedTarget` (generated)

**Assertions:**
- Only `GeneratedTarget` appears in `missing_types.go`
- `ExistingTarget` is referenced normally in caster

---

## Shell-Based Example Tests

### Location: `examples/` directory

These are runnable shell scripts that serve as both examples and integration tests.

---

### Example 1: `examples/virtual/` (existing, enhanced)

**File:** `examples/virtual/run.sh`

**Purpose:** Demonstrate basic generated target type in same package as source.

**Verifications:**
- [x] `missing_types.go` created in `examples/virtual/`
- [x] Contains `type Target struct` with correct fields
- [x] Contains `type TargetItem struct` with correct fields
- [x] No `virtual.` prefix in struct definitions
- [x] Caster files use `virtual.Target` and `virtual.TargetItem`
- [x] Project compiles

---

### Example 2: `examples/virtual-cross-pkg/` (new)

**Purpose:** Demonstrate generated target type in a different package than source.

**Structure:**
```
examples/virtual-cross-pkg/
├── source/
│   └── types.go          # Source types
├── target/
│   └── (empty initially, missing_types.go generated here)
├── generated/
│   └── (caster files)
├── stages/
│   └── mapping.yaml
└── run.sh
```

**Mapping:**
```yaml
version: "1"
mappings:
  - source: .../source.Input
    target: .../target.Output
    generate_target: true
    fields:
      - source: Data
        target: Data
```

**Verifications:**
- `missing_types.go` created in `examples/virtual-cross-pkg/target/`
- Contains `package target`
- Caster imports both `source` and `target` packages
- All files compile

---

### Example 3: `examples/virtual-with-external/` (new)

**Purpose:** Demonstrate generated target type that references external types (e.g., `time.Time`).

**Source:**
```go
type Event struct {
    ID        string
    Timestamp time.Time
    Data      []byte
}
```

**Mapping:**
```yaml
version: "1"
mappings:
  - source: .../Event
    target: .../EventDTO
    generate_target: true
    auto: all
```

**Verifications:**
- `missing_types.go` has `import "time"`
- Field `Timestamp time.Time` present
- No invalid imports
- Compiles successfully

---

### Example 4: `examples/virtual-nested/` (new)

**Purpose:** Demonstrate generated types with nested struct references.

**Source:**
```go
type Order struct {
    ID    string
    Items []OrderItem
}

type OrderItem struct {
    SKU string
    Qty int
}
```

**Mapping:**
```yaml
version: "1"
mappings:
  - source: .../Order
    target: .../OrderDTO
    generate_target: true
    auto: all
  
  - source: .../OrderItem
    target: .../OrderItemDTO
    generate_target: true
    auto: all
```

**Verifications:**
- `missing_types.go` contains both `OrderDTO` and `OrderItemDTO`
- `Items []OrderItemDTO` (no package prefix)
- Nested caster function generated
- Compiles successfully

---

### Example 5: `examples/virtual-pointer-fields/` (new)

**Purpose:** Demonstrate generated types preserve pointer semantics.

**Source:**
```go
type Person struct {
    Name    string
    Age     *int
    Address *Address
}

type Address struct {
    City string
}
```

**Mapping with renames:**
```yaml
version: "1"
mappings:
  - source: .../Person
    target: .../PersonDTO
    generate_target: true
    fields:
      - source: Name
        target: FullName
      - source: Age
        target: Age
      - source: Address
        target: Location
  
  - source: .../Address
    target: .../AddressDTO
    generate_target: true
    fields:
      - source: City
        target: CityName
```

**Verifications:**
- `PersonDTO` has `Age *int` and `Location *AddressDTO`
- `AddressDTO` has `CityName string`
- Pointer nil-check logic in caster
- Compiles successfully

---

## Test Execution

### Unit Tests

```bash
# Run all generator tests
go test -v ./internal/gen/...

# Run specific test
go test -v ./internal/gen/... -run TestGenerateMissingTypesFile
```

### Integration Tests

```bash
# Run all integration tests
go test -v ./internal/gen/... -run TestIntegration

# Run with verbose output
go test -v ./internal/gen/... -run TestIntegration -count=1
```

### Shell Examples

```bash
# Run all examples
make examples

# Run specific example
./examples/virtual/run.sh

# Run without prompts (CI mode)
CG_NO_PROMPT=1 ./examples/virtual/run.sh
```

---

## CI Integration

Add to `.github/workflows/test.yml` or equivalent:

```yaml
- name: Run unit tests
  run: go test -v ./...

- name: Run shell examples
  run: |
    export CG_NO_PROMPT=1
    for script in examples/*/run.sh; do
      echo "Running $script"
      bash "$script"
    done
```

---

## Coverage Goals

| Area | Target Coverage |
|------|-----------------|
| `generateMissingTypesFiles` | 90% |
| `typeRefString` with `contextPkgPath` | 100% |
| `addMissingType` | 100% |
| `PackageInfo.Dir` population | 100% |
| End-to-end virtual example | Pass |
