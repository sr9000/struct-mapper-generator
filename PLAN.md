# Plan: Generate Missing Types for Cast

This feature allows `caster-generator` to generate Go struct definitions for missing target types based on the mapping configuration. This is useful for initializing a domain layer from an API layer (Schema-First approach).

## 1. Schema Extensions (`internal/mapping`)

We need to allow the user to specify that a target type should be generated, and optionally specify types for fields.

### `internal/mapping/schema.go`
- **Modify `TypeMapping` struct**:
  - Add `GenerateTarget bool` (yaml: `generate_target,omitempty`). If true, the target type is expected to be missing and will be generated.
- **Modify `FieldMapping` struct**:
  - Add `TargetType string` (yaml: `target_type,omitempty`). Allows explicitly defining the type of the generated target field (e.g., "string", "*int", "float64").

## 2. Analysis & Type Graph (`internal/analyze`)

The analysis phase needs to handle "virtual" types that don't exist in the parsed Go code.

### `internal/analyze/types.go`
- **Modify `TypeInfo` struct**:
  - Add `IsGenerated bool`.
  - Ensure methods handle cases where `GoType` (the underlying `go/types.Type`) might be nil for generated types.

## 3. Resolution Logic (`internal/plan`)

The resolver currently fails if the target type is missing. We need to intercept this.

### `internal/plan/resolver.go`
- **Modify `resolveTypeMapping`**:
  - If `tm.Source` resolves but `tm.Target` does not:
    - Check if `tm.GenerateTarget` is true.
    - If yes, **create a virtual `TypeInfo`** for the target.
      - Parse package and name from `tm.Target`.
      - mark `IsGenerated = true`.
      - Add it to the `TypeGraph` or a local cache.
    - **Synthesize Fields**:
      - Iterate over `tm.OneToOne`:
        - Create `FieldInfo` for target.
        - Type: Same as source field.
      - Iterate over `tm.Fields`:
        - Create `FieldInfo` for target.
        - Type determination:
          - If `TargetType` is set in mapping -> resolve basic type (string, int, etc.) or simple named type.
          - If `Method` (Transform) is used -> infer from transform's return type (if available).
          - Else -> use Source field's type.
  - **Skip Compatibility Checks**:
    - When validating the mapping (Source -> Target), if Target is `IsGenerated`, we skip strict `go/types` compatibility checks and assume the mapping is valid (since we are defining the target to be valid).

### `internal/match/compatibility.go` (or usage in `plan`)
- Ensure compatibility scoring handles `nil` `GoType` if passed, or ensure the resolver doesn't call it for generated types.

## 4. Generation Phase (`internal/gen`)

We need a new generator component to write the `type Struct struct { ... }` definition.

### `internal/gen/generator.go`
- Add a new phase `GenerateTypes` before or alongside `GenerateConversions`.
- Identify types in the `ResolvedMappingPlan` that are marked `IsGenerated` (or have a flag in `ResolvedTypePair`).

### `internal/gen/type_gen.go` (New File)
- Implement `TypeGenerator` or functions to render struct definitions.
- iterate fields of the fabricated `TypeInfo`.
- Write:
  ```go
  type TypeName struct {
      FieldName FieldType `json:"fieldName"` // Maybe default tags?
  }
  ```
- Important: We need to decide *where* to write these types.
  - For V1, we will write them to the same output file as the converters (assuming the user directs output to `domain/types.go` or similar).

## 5. Structs & Interfaces

### New/Modified Structs

**`internal/mapping/schema.go`**
```go
type TypeMapping struct {
    // ... existing ...
    GenerateTarget bool `yaml:"generate_target,omitempty"`
}

type FieldMapping struct {
    // ... existing ...
    TargetType string `yaml:"target_type,omitempty"`
}
```

**`internal/analyze/types.go`**
```go
type TypeInfo struct {
    // ... existing ...
    IsGenerated bool
}
```

**`internal/plan/types.go`**
```go
type ResolvedTypePair struct {
    // ... existing ...
    IsGeneratedTarget bool
}
```

## 6. Execution Steps (Implementation Order)

1.  Modify `mapping/schema.go` to support new YAML fields.
2.  Update `analyze/types.go` to support generated type flags.
3.  Update `plan/resolver.go` to logic to synthesize target types from mapping.
4.  Implement `gen/type_gen.go` and hook it into `gen/generator.go`.
5.  Verification with a new example test case.
