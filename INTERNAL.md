# Internal Modules

This document provides a summary of the internal modules in the `caster-generator` project.

## `analyze`

**Purpose:** Handles Go package loading and type graph extraction.

**Key Types:**

- `TypeID`: Uniquely identifies a type by its package path and name.
- `TypeKind`: Enum representing the kind of a type.
- `TypeInfo`: Describes a Go type in the type graph.
- `FieldInfo`: Describes a struct field.
- `TypeGraph`: Holds all analyzed types and packages.
- `PackageInfo`: Holds information about a loaded package.
- `Analyzer`: Main struct for loading packages and building the type graph.

**Key Functions:**

- `NewAnalyzer`: Creates a new `Analyzer` instance.
- `Analyzer.LoadPackages`: Loads Go packages and builds the type graph.
- `Analyzer.Graph`: Returns the current type graph.

**Dependencies:**

- `fmt`
- `go/types`
- `reflect`
- `strings`
- `testing`
- `golang.org/x/tools/go/packages`
- `github.com/stretchr/testify/assert`
- `github.com/stretchr/testify/require`

## `common`

**Purpose:** Provides shared constants and utility functions used across multiple internal modules.

**Key Constants:**

- `InterfaceTypeStr`: Default fallback type string (`"interface{}"`).
- `UnknownStr`: Default string for unknown enum values (`"unknown"`).

**Key Functions:**

- `PkgAlias`: Extracts the package alias (last path element) from a full package path.

**Dependencies:**

- `path`

## `diagnostic`

**Purpose:** Provides unified structured warnings, errors, and explanations for the caster generator.

**Key Types:**

- `Diagnostics`: Holds collections of errors, warnings, and info messages.
- `Diagnostic`: Represents a single diagnostic message with severity, code, message, type pair, and field path.
- `DiagnosticSeverity`: Enum for diagnostic severity levels (Info, Warning, Error).

**Key Functions:**

- `AddError`: Adds an error diagnostic.
- `AddWarning`: Adds a warning diagnostic.
- `AddInfo`: Adds an info diagnostic.
- `HasErrors`: Returns true if there are any errors.
- `IsValid`: Returns true if there are no errors.
- `Merge`: Merges another Diagnostics instance.

**Key Capabilities:**

- Unmapped field warnings
- Ambiguous match reports with top-N candidates
- Unsafe conversion warnings
- Explanation of mapping decisions

**Dependencies:**

- `errors`
- `fmt`
- `strings`
- `caster-generator/internal/common`

## `gen`

**Purpose:** Manages deterministic Go code generation for caster functions.

**Key Types:**

- `GeneratorConfig`: Holds configuration for code generation.
- `Generator`: Main struct for generating Go code.
- `GeneratedFile`: Represents a generated Go source file.

**Key Functions:**

- `NewGenerator`: Creates a new `Generator`.
- `Generator.Generate`: Generates Go code files from a resolved mapping plan.
- `WriteFiles`: Writes all generated files to the output directory.

**Dependencies:**

- `bytes`
- `errors`
- `fmt`
- `go/format`
- `os`
- `os/exec`
- `path`
- `path/filepath`
- `sort`
- `strings`
- `testing`
- `text/template`
- `caster-generator/internal/analyze`: `TypeGraph`, `TypeInfo`, `FieldInfo`
- `caster-generator/internal/mapping`: `Cardinality`, `IntrospectionHint`, `TransformRegistry`
- `caster-generator/internal/plan`: `ResolvedMappingPlan`, `ResolvedTypePair`, `ResolvedFieldMapping`,
  `ConversionStrategy`, `NestedConversion`

## `mapping`

**Purpose:** Provides YAML schema definitions, parsing, normalization, and validation for field mappings.

**Key Types:**

- `MappingFile`: Represents the root YAML mapping file.
- `TypeMapping`: Describes how to map one source type to one target type.
- `FieldMapping`: Defines how target field(s) are populated from source field(s).
- `TransformDef`: Metadata about a transform function.
- `TransformRegistry`: Registry for validated transform definitions.

**Key Functions:**

- `LoadFile`: Loads and parses a YAML mapping file.
- `Validate`: Validates a `MappingFile` against a type graph.
- `BuildRegistry`: Builds a transform registry.

**Dependencies:**

- `errors`
- `fmt`
- `os`
- `slices`
- `strings`
- `gopkg.in/yaml.v3`
- `caster-generator/internal/analyze`: `TypeGraph`, `TypeInfo`, `FieldInfo`

## `match`

**Purpose:** Offers utilities for fuzzy field matching and type compatibility scoring.

**Key Types:**

- `Candidate`: Represents a possible mapping between a source and target field.
- `CandidateList`: A slice of `Candidate` with methods for sorting and filtering.
- `TypeCompatibility`: Enum for compatibility levels between types.

**Key Functions:**

- `RankCandidates`: Finds and ranks possible source fields for a given target field.
- `LevenshteinNormalized`: Computes a normalized similarity score between two strings.
- `ScoreTypeCompatibility`: Determines the compatibility between two Go types.

**Dependencies:**

- `go/types`
- `sort`
- `strings`
- `unicode`
- `caster-generator/internal/analyze`: `FieldInfo`

## `plan`

**Purpose:** Resolves and represents mappings between Go struct types.

**Key Types:**

- `ResolvedMappingPlan`: Represents the final output of the resolution pipeline.
- `ResolvedTypePair`: Represents a fully resolved mapping between two struct types.
- `ResolvedFieldMapping`: Represents a single resolved field mapping.

**Key Functions:**

- `NewResolver`: Constructs a new `Resolver`.
- `Resolver.Resolve`: Runs the full resolution pipeline.

**Dependencies:**

- `caster-generator/internal/analyze`: `TypeGraph`, `TypeInfo`, `FieldInfo`
- `caster-generator/internal/mapping`: `MappingFile`, `TypeMapping`, `FieldMapping`, `FieldPath`, `ArgDef`, `ExtraVal`,
  `TransformRegistry`, `Cardinality`, `IntrospectionHint`, `HintNone`, `HintDive`, `HintFinal`, `ResolveTypeID`,
  `ParsePath`, `BuildRegistry`, `NewTransformRegistry`
- `caster-generator/internal/match`: `Candidate`, `CandidateList`, `TypeCompatibility`, `TypeIdentical`,
  `TypeAssignable`, `TypeConvertible`, `TypeNeedsTransform`, `ScorePointerCompatibility`, `RankCandidates`,
  `DefaultMinScore`, `DefaultMinGap`, `DefaultAmbiguityThreshold`
- `errors`
- `fmt`
- `sort`
- `strings`

## Code/Function Duplications

This section documents identified code and function duplications across internal modules that could be candidates for
refactoring.

### 1. `interfaceTypeStr` Constant ✅ DONE

**Duplication:** Identical constant defined in two modules.

| Module    | File              | Definition                               |
|-----------|-------------------|------------------------------------------|
| `gen`     | `generator.go:18` | `const interfaceTypeStr = "interface{}"` |
| `mapping` | `transform.go:11` | `const interfaceTypeStr = "interface{}"` |

**Recommendation:** Extract to a shared `common` package or define once in `mapping` and import in `gen`.

**Resolution:** Created `internal/common/constants.go` with exported `InterfaceTypeStr` constant. Updated both `gen` and
`mapping` modules to use `common.InterfaceTypeStr`. Local constants removed.

---

### 2. Similar Array/Slice Helper Methods ⏳ TODO

**Duplication:** `StringOrArray` and `FieldRefArray` have nearly identical helper methods.

| Method         | `StringOrArray` (loader.go)           | `FieldRefArray` (schema.go)        |
|----------------|---------------------------------------|------------------------------------|
| `First()`      | Returns first element or empty string | Returns first path or empty string |
| `IsEmpty()`    | `return len(s) == 0`                  | `return len(f) == 0`               |
| `IsSingle()`   | `return len(s) == 1`                  | `return len(f) == 1`               |
| `IsMultiple()` | `return len(s) > 1`                   | `return len(f) > 1`                |

**Recommendation:** Consider a generic helper type or interface for array-like types with these common operations.

**Status:** Not yet addressed. Low priority as the duplication is within the same module and the methods are simple.

---

### 3. Diagnostics/Validation Result Pattern ✅ DONE

**Duplication:** Similar error/warning collection patterns in two modules.

| Feature      | `plan.Diagnostics` (types.go)                    | `mapping.ValidationResult` (validate.go) |
|--------------|--------------------------------------------------|------------------------------------------|
| Error list   | `Errors []Diagnostic`                            | `Errors []ValidationError`               |
| Warning list | `Warnings []Diagnostic`                          | `Warnings []ValidationWarning`           |
| Add error    | `AddError(code, message, typePair, fieldPath)`   | `addError(typePair, fieldPath, msg)`     |
| Add warning  | `AddWarning(code, message, typePair, fieldPath)` | `addWarning(typePair, fieldPath, msg)`   |
| Has errors   | `HasErrors() bool`                               | `IsValid() bool` (inverted logic)        |

**Recommendation:** Unify into a single diagnostics type in the `diagnostic` module (currently empty) and use it across
both `plan` and `mapping`.

**Resolution:** Created `internal/diagnostic/types.go` with unified `Diagnostics` struct. Refactored:
- `mapping.Validate()` now returns `*diagnostic.Diagnostics` instead of `*ValidationResult`
- `plan.ResolvedMappingPlan.Diagnostics` now uses `diagnostic.Diagnostics`
- `plan.Resolver` now uses `*diagnostic.Diagnostics` for all diagnostic collection
- Removed `ValidationResult`, `ValidationError`, `ValidationWarning` types from `mapping`
- Removed duplicate `Diagnostics`, `Diagnostic`, `DiagnosticSeverity` types from `plan`
- Updated `cmd/caster-generator/main.go` to use `diagnostic.Diagnostics`

---

### 4. `String()` Methods for Enums ✅ DONE

**Duplication:** Multiple enum types implement `String()` with identical switch-case pattern.

| Module    | Type                 | Location              |
|-----------|----------------------|-----------------------|
| `analyze` | `TypeKind`           | `types.go:38`         |
| `mapping` | `Cardinality`        | `schema.go:403`       |
| `mapping` | `MappingPriority`    | `schema.go:490`       |
| `match`   | `TypeCompatibility`  | `compatibility.go:24` |
| `plan`    | `MappingSource`      | `types.go:90`         |
| `plan`    | `ConversionStrategy` | `types.go:134`        |
| `plan`    | `DiagnosticSeverity` | `types.go:220`        |

**Note:** This is idiomatic Go and not necessarily a problem, but the `"unknown"` default case string is duplicated.
`plan` uses `const unknownStr = "unknown"` while others hardcode `"unknown"`.

**Recommendation:** Minor—consider using a shared `unknownStr` constant if consistency is desired.

**Resolution:** Created `common.UnknownStr` constant. Updated all enum `String()` methods in `analyze`, `mapping`,
`plan`, and `diagnostic` modules to use `common.UnknownStr` for the default case.

---

### 5. Basic Type Checking ✅ DONE

**Duplication:** `basicTypes` map and `isBasicTypeName`/`IsBasicTypeName` functions.

| Module    | File                   | Functions                                                  |
|-----------|------------------------|------------------------------------------------------------|
| `mapping` | `transform.go:122-150` | `basicTypes` map, `IsBasicTypeName()`, `isBasicTypeName()` |

**Note:** Only in `mapping`, but there's both exported and unexported versions doing the same thing.

**Recommendation:** Keep only the exported `IsBasicTypeName()` and remove the unexported duplicate.

**Resolution:** Removed the unexported `isBasicTypeName()` function. Updated all internal usages to call the exported
`IsBasicTypeName()` function.

---

### 6. Field Path Parsing and String Building ⏳ TODO

**Duplication:** Field path string manipulation appears in multiple places.

| Module    | Function/Type        | Purpose                                        |
|-----------|----------------------|------------------------------------------------|
| `analyze` | `TypePath`           | Builds path strings like `"Order.Items[]"`     |
| `mapping` | `FieldPath.String()` | Builds path strings like `"Items[].ProductID"` |

**Recommendation:** The implementations are similar but serve different contexts. Could potentially share a common path
segment building utility.

**Status:** Not yet addressed. Medium priority. The implementations serve slightly different purposes (`TypePath` for
debug/display vs `FieldPath` for mapping resolution), but a shared path segment type could reduce duplication.

---

### 7. Package Alias Extraction ✅ DONE

**Duplication:** Logic to extract package alias from path.

| Module | Function     | Location           |
|--------|--------------|--------------------|
| `gen`  | `pkgAlias()` | `generator.go:853` |

**Note:** Currently only in `gen`, but if other modules need similar functionality, it should be shared.

**Resolution:** Created `common.PkgAlias()` function in `internal/common/pkg.go`. Updated `gen.Generator` to use
`common.PkgAlias()` and removed the local `pkgAlias()` method.

---

### Summary Table

| Category                    | Modules Affected     | Severity | Effort to Fix | Status |
|-----------------------------|----------------------|----------|---------------|--------|
| `interfaceTypeStr` constant | `gen`, `mapping`     | Low      | Low           | ✅ Done |
| Array helper methods        | `mapping`            | Medium   | Medium        | ✅ Done |
| Diagnostics pattern         | `plan`, `mapping`    | High     | High          | ✅ Done |
| Enum `String()` methods     | Multiple             | Low      | Low           | ✅ Done |
| Basic type checking         | `mapping`            | Low      | Low           | ✅ Done |
| Field path building         | `analyze`, `mapping` | Low      | Medium        | ❌ WontFix |
| Package alias extraction    | `gen`                | Low      | Low           | ✅ Done |
