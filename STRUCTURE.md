# Internal Structure Documentation

This document describes the internal architecture of the caster-generator project. It is divided into two parts:

1. **Brief Description and Relations** — high-level overview of modules and their dependencies
2. **Detailed Module Overview** — comprehensive documentation of each module's types, functions, and behaviors

---

## Part 1: Brief Description and Relations

### Module Overview

| Module       | Purpose                                                                           |
|--------------|-----------------------------------------------------------------------------------|
| `common`     | Shared constants and generic utility functions used across all modules            |
| `diagnostic` | Structured error/warning/info collection and reporting                            |
| `analyze`    | Package loading and Go type graph extraction using `go/types`                     |
| `mapping`    | YAML schema, parsing, validation, and transform registry                          |
| `match`      | Name normalization, string similarity, type compatibility, and candidate ranking  |
| `plan`       | Resolution pipeline that converts mappings + auto-match into a deterministic plan |
| `gen`        | Code generation: template rendering, formatting, and file output                  |

### Dependency Graph

```
                    ┌─────────────┐
                    │   common    │  (no internal deps)
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │diagnostic│ │ analyze  │ │  match   │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │            │            │
             │       uses │            │ uses
             │            ▼            │
             │      ┌──────────┐       │
             └─────►│ mapping  │◄──────┘
                    └────┬─────┘
                         │
                         ▼
                    ┌──────────┐
                    │   plan   │
                    └────┬─────┘
                         │
                         ▼
                    ┌──────────┐
                    │   gen    │
                    └──────────┘
```

### Inter-Module Dependencies

| Module       | Depends On                                                      |
|--------------|-----------------------------------------------------------------|
| `common`     | stdlib only (`path`)                                            |
| `diagnostic` | `common`                                                        |
| `analyze`    | `common`, stdlib (`go/types`, `golang.org/x/tools/go/packages`) |
| `mapping`    | `common`, `analyze`, `diagnostic`                               |
| `match`      | `common`, `analyze`, stdlib (`go/types`)                        |
| `plan`       | `common`, `analyze`, `mapping`, `match`, `diagnostic`           |
| `gen`        | `common`, `analyze`, `mapping`, `plan`                          |

### Data Flow

```
┌─────────────────┐     ┌─────────────────┐
│  Go Packages    │     │   YAML File     │
│  (source code)  │     │  (mappings)     │
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│    analyze      │     │    mapping      │
│   (TypeGraph)   │     │  (MappingFile)  │
└────────┬────────┘     └────────┬────────┘
         │                       │
         └───────────┬───────────┘
                     ▼
             ┌───────────────┐
             │     plan      │
             │ (Resolver)    │
             └───────┬───────┘
                     │
                     ▼
          ┌─────────────────────┐
          │ ResolvedMappingPlan │
          └──────────┬──────────┘
                     │
                     ▼
             ┌───────────────┐
             │      gen      │
             │  (Generator)  │
             └───────┬───────┘
                     │
                     ▼
          ┌─────────────────────┐
          │  Generated Go Code  │
          └─────────────────────┘
```

---

## Part 2: Detailed Module Overview

---

### Module: `common`

**Location:** `internal/common/`

**Purpose:** Provides shared constants and generic utility functions used across all other modules. This is a low-level
utility package with minimal dependencies.

#### Constants

| Constant           | Value           | Purpose                                                      |
|--------------------|-----------------|--------------------------------------------------------------|
| `InterfaceTypeStr` | `"interface{}"` | Fallback type string when concrete type cannot be determined |
| `UnknownStr`       | `"unknown"`     | Fallback label for unknown enum values in `String()` methods |

#### Functions

| Function     | Signature                        | Purpose                                                                        |
|--------------|----------------------------------|--------------------------------------------------------------------------------|
| `PkgAlias`   | `(pkgPath string) string`        | Returns the last path element as a package alias; returns `""` for empty input |
| `IsEmpty`    | `[S ~[]E, E any](s S) bool`      | Returns `true` if slice is empty                                               |
| `IsSingle`   | `[S ~[]E, E any](s S) bool`      | Returns `true` if slice has exactly one element                                |
| `IsMultiple` | `[S ~[]E, E any](s S) bool`      | Returns `true` if slice has more than one element                              |
| `First`      | `[S ~[]E, E any](s S) (E, bool)` | Returns first element and `true`, or zero value and `false` if empty           |

#### External Dependencies

- `path` (stdlib) — used by `PkgAlias`

---

### Module: `diagnostic`

**Location:** `internal/diagnostic/`

**Purpose:** Provides structured diagnostic collection (errors, warnings, info messages) for the resolution and
validation pipeline. Enables detailed reporting with context about which type pair and field path caused each
diagnostic.

#### Types

##### `DiagnosticSeverity`

```go
type DiagnosticSeverity int

const (
  DiagnosticInfo DiagnosticSeverity = iota
  DiagnosticWarning
  DiagnosticError
)
```

- **Purpose:** Enum representing severity levels
- **Method:** `String() string` — returns `"info"`, `"warning"`, `"error"`, or `common.UnknownStr`

##### `Diagnostic`

```go
type Diagnostic struct {
  Severity    DiagnosticSeverity
  Code        string // Unique identifier for this diagnostic type
  Message     string // Human-readable description
  TypePair    string   // Context: which type mapping (optional)
  FieldPath   string   // Context: which field (optional)
  Suggestions []string // Potential fixes
}
```

- **Purpose:** Single diagnostic message with structured context
- **Method:** `String() string` — formats as `[TypePair] FieldPath: [Code] Message`

##### `Diagnostics`

```go
type Diagnostics struct {
  Errors   []Diagnostic
  Warnings []Diagnostic
  Infos    []Diagnostic
}
```

- **Purpose:** Accumulator for all diagnostics during resolution/validation

| Method                                                  | Purpose                                                              |
|---------------------------------------------------------|----------------------------------------------------------------------|
| `AddError(code, message, typePair, fieldPath string)`   | Append an error diagnostic                                           |
| `AddWarning(code, message, typePair, fieldPath string)` | Append a warning diagnostic                                          |
| `AddInfo(code, message, typePair, fieldPath string)`    | Append an info diagnostic                                            |
| `HasErrors() bool`                                      | Returns `true` if any errors exist                                   |
| `IsValid() bool`                                        | Returns `true` if no errors exist                                    |
| `Merge(other Diagnostics)`                              | Concatenates another `Diagnostics` into this one                     |
| `Error() error`                                         | Returns combined error from all error diagnostics, or `nil` if valid |

#### External Dependencies

- `errors`, `fmt`, `strings` (stdlib)
- `common` (internal) — for `UnknownStr`

---

### Module: `analyze`

**Location:** `internal/analyze/`

**Purpose:** Loads Go packages using `golang.org/x/tools/go/packages` and builds an in-memory type graph using
`go/types`. Provides canonical representations of structs, fields, pointers, slices, arrays, aliases, and external
types.

#### Types

##### `TypeID`

```go
type TypeID struct {
  PkgPath string // Import path (e.g., "caster-generator/store")
  Name    string // Type name (e.g., "Order")
}
```

- **Purpose:** Unique identifier for a named type
- **Method:** `String() string` — returns `"pkg.Path.Name"` or just `Name` if `PkgPath` is empty

##### `TypeKind`

```go
type TypeKind int

const (
  TypeKindUnknown TypeKind = iota
  TypeKindBasic
  TypeKindStruct
  TypeKindPointer
  TypeKindSlice
  TypeKindArray
  TypeKindAlias
  TypeKindExternal
)
```

- **Purpose:** Enum of Go type categories
- **Method:** `String() string` — returns human-readable kind name

##### `TypeInfo`

```go
type TypeInfo struct {
  ID         TypeID    // Filled for named types
  Kind       TypeKind  // One of the enum values
  Underlying *TypeInfo // For aliases: the underlying type
  ElemType   *TypeInfo   // For pointers/slices/arrays: element type
  Fields     []FieldInfo // For structs: field list
  GoType     types.Type // Original go/types object
}
```

- **Purpose:** Canonical model node describing a Go type
- **Method:** `IsNamed() bool` — returns `true` if `ID.Name` is non-empty

##### `FieldInfo`

```go
type FieldInfo struct {
  Name     string
  Exported bool
  Type     *TypeInfo
  Tag      reflect.StructTag
  Embedded bool
  Index    int
}
```

- **Purpose:** Describes a struct field with metadata

| Method                      | Purpose                                                                     |
|-----------------------------|-----------------------------------------------------------------------------|
| `JSONName() string`         | Returns JSON tag name (first segment before comma), or field name if no tag |
| `HasTag(key string) bool`   | Returns `true` if tag key exists and is non-empty                           |
| `GetTag(key string) string` | Returns raw tag value via `reflect.StructTag.Get`                           |

##### `TypeGraph`

```go
type TypeGraph struct {
  Types    map[TypeID]*TypeInfo    // Keyed by TypeID
  Packages map[string]*PackageInfo // Keyed by PkgPath
}
```

- **Purpose:** Top-level container of all analyzed types and package metadata
- **Constructor:** `NewTypeGraph() *TypeGraph`
- **Method:** `GetType(id TypeID) *TypeInfo`

##### `PackageInfo`

```go
type PackageInfo struct {
  Path  string
  Name  string
  Dir   string
  Types []TypeID
}
```

- **Purpose:** Per-package metadata

##### `Analyzer`

```go
type Analyzer struct {
  graph     *TypeGraph
  typeCache map[types.Type]*TypeInfo
}
```

- **Purpose:** Orchestrates package loading and recursive type analysis
- **Constructor:** `NewAnalyzer() *Analyzer`

| Method                                                   | Purpose                                           |
|----------------------------------------------------------|---------------------------------------------------|
| `LoadPackages(patterns ...string) (*TypeGraph, error)`   | Load packages matching patterns, build type graph |
| `Graph() *TypeGraph`                                     | Returns the internal type graph                   |
| `GetStruct(pkgPath, typeName string) (*TypeInfo, error)` | Find a named struct by package path and name      |

##### `TypePath`

```go
type TypePath struct {
  parts []string
}
```

- **Purpose:** Compose readable hierarchical type/field path strings

| Method                               | Purpose                                    |
|--------------------------------------|--------------------------------------------|
| `NewTypePath(root string) *TypePath` | Create with root                           |
| `Field(name string) *TypePath`       | Append field part (returns new `TypePath`) |
| `Slice() *TypePath`                  | Append `[]` to last part                   |
| `Pointer() *TypePath`                | Prefix `*` to last part                    |
| `String() string`                    | Join parts with `"."`                      |

##### `TypeStringer`

- **Purpose:** Human-readable type stringification and field path map building
- **Constructor:** `NewTypeStringer() *TypeStringer`

| Method                                                                | Purpose                                                 |
|-----------------------------------------------------------------------|---------------------------------------------------------|
| `TypeString(t *TypeInfo) string`                                      | Produce readable type string                            |
| `FieldPath(typeName string, fieldNames ...string) string`             | Build field path string                                 |
| `BuildFieldPaths(root *TypeInfo, maxDepth int) map[string]*FieldInfo` | Build map of all field paths to `FieldInfo` up to depth |

#### Internal Helpers

| Function                                                | Purpose                                          |
|---------------------------------------------------------|--------------------------------------------------|
| `analyzeType(t types.Type) *TypeInfo`                   | Recursive analysis of `go/types` into `TypeInfo` |
| `analyzeNamedType(named *types.Named, info *TypeInfo)`  | Fill `TypeInfo` for named types                  |
| `analyzeStructFields(st *types.Struct, info *TypeInfo)` | Extract struct fields                            |
| `processPackage(pkg *packages.Package)`                 | Process a loaded package into the graph          |
| `isExternalPackage(pkgPath string) bool`                | Check if package is external (not in graph)      |

#### External Dependencies

- `golang.org/x/tools/go/packages` — package loading with AST and types
- `go/types` (stdlib) — type inspection
- `reflect` (stdlib) — struct tag parsing
- `fmt`, `strings` (stdlib)
- `common` (internal)

---

### Module: `mapping`

**Location:** `internal/mapping/`

**Purpose:** Defines the YAML schema for explicit field mappings, provides parsing and validation, and manages the
transform registry. Supports 1:1, 1:N, N:1, N:M cardinalities, path expressions, introspection hints, and named
transforms.

#### Schema Types

##### `MappingFile`

```go
type MappingFile struct {
  Version      string        `yaml:"version,omitempty"`
  TypeMappings []TypeMapping `yaml:"mappings"`
  Transforms   []TransformDef `yaml:"transforms,omitempty"`
}
```

- **Purpose:** Top-level YAML mapping file representation

##### `TypeMapping`

```go
type TypeMapping struct {
  Source         string            // Source type name/ID
  Target         string            // Target type name/ID
  Requires       ArgDefArray       // Extra function arguments
  OneToOne       map[string]string // 121 shorthand mappings
  GenerateTarget bool              // Generate target type if missing
  Fields         []FieldMapping    // Explicit field mappings
  Ignore         []string          // Fields to ignore
  Auto           []FieldMapping    // Auto-matched fields with overrides
}
```

##### `FieldMapping`

```go
type FieldMapping struct {
  Source     FieldRefArray // Source field reference(s)
  Target     FieldRefArray // Target field reference(s)
  TargetType string        // Explicit target type (for GenerateTarget)
  Default    *string       // Default value if source absent
  Transform  string        // Transform function name
  Extra      ExtraVals     // Extra value references
}
```

| Method                                 | Purpose                                                     |
|----------------------------------------|-------------------------------------------------------------|
| `GetCardinality() Cardinality`         | Returns 1:1, 1:N, N:1, or N:M based on source/target counts |
| `NeedsTransform() bool`                | Returns `true` for N:1 or N:M (transforms required)         |
| `GetEffectiveHint() IntrospectionHint` | Delegates to `Source.GetEffectiveHint(Target)`              |

##### `Cardinality`

```go
type Cardinality int

const (
  CardinalityOneToOne Cardinality = iota
  CardinalityOneToMany
  CardinalityManyToOne
  CardinalityManyToMany
)
```

- **Method:** `String() string` — returns `"1:1"`, `"1:N"`, `"N:1"`, `"N:M"`

##### `IntrospectionHint`

```go
type IntrospectionHint string

const (
  HintNone  IntrospectionHint = ""
  HintDive  IntrospectionHint = "dive"
  HintFinal IntrospectionHint = "final"
)
```

- **Purpose:** Controls recursive field introspection
- **Method:** `IsValid() bool`

##### `FieldRef`

```go
type FieldRef struct {
  Path string // e.g., "Items[].ProductID"
  Hint IntrospectionHint
}
```

##### `FieldRefArray`

- **Purpose:** Slice of `FieldRef` with YAML marshaling support (accepts string, map, or array)

| Method                                                      | Purpose                                             |
|-------------------------------------------------------------|-----------------------------------------------------|
| `Paths() []string`                                          | Returns slice of path strings                       |
| `First() string`, `FirstRef() FieldRef`                     | Get first element                                   |
| `IsEmpty()`, `IsSingle()`, `IsMultiple()`                   | Cardinality checks                                  |
| `HasAnyHint()`, `HasConflictingHints()`                     | Hint presence checks                                |
| `GetEffectiveHint(targets FieldRefArray) IntrospectionHint` | Determine effective hint based on cardinality rules |

##### `FieldPath` and `PathSegment`

```go
type PathSegment struct {
  Name    string
  IsSlice bool // true for "[]"
}

type FieldPath struct {
  segments []PathSegment
}
```

| Method                         | Purpose                          |
|--------------------------------|----------------------------------|
| `String() string`              | Reconstruct path string          |
| `IsSimple() bool`              | True if single segment, no slice |
| `Root() string`                | First segment name               |
| `IsEmpty() bool`               | True if no segments              |
| `Equals(other FieldPath) bool` | Deep equality                    |

##### `ArgDef` and `ArgDefArray`

```go
type ArgDef struct {
  Name string
  Type string
}
```

- **Purpose:** Define extra function arguments (for `Requires`)
- `ArgDefArray` has custom YAML unmarshaling (accepts string, map, or list)

##### `ExtraDef`, `ExtraVal`, `ExtraVals`

```go
type ExtraDef struct {
  Source string
  Target string
}

type ExtraVal struct {
  Name string
  Def  ExtraDef
}
```

- **Purpose:** Reference extra values in field mappings
- `ExtraVals` has custom YAML unmarshaling

#### Transform Types

##### `TransformDef`

```go
type TransformDef struct {
  Name          string
  SourceType    string
  TargetType    string
  Package       string // optional
  Func          string // optional
  Description   string
  AutoGenerated bool
}
```

- **Purpose:** Declares a transform function

##### `ValidatedTransform`

```go
type ValidatedTransform struct {
  *TransformDef
  SourceType *analyze.TypeInfo
  TargetType *analyze.TypeInfo
}
```

- **Method:** `FuncCall() string` — returns fully-qualified function expression

##### `TransformRegistry`

- **Constructor:** `NewTransformRegistry() *TransformRegistry`

| Method                                                                                   | Purpose                               |
|------------------------------------------------------------------------------------------|---------------------------------------|
| `BuildRegistry(mf *MappingFile, graph *analyze.TypeGraph) (*TransformRegistry, []error)` | Resolve transforms against type graph |
| `Add(def *TransformDef)`                                                                 | Register a transform                  |
| `Get(name string) *ValidatedTransform`                                                   | Lookup by name                        |
| `Has(name string) bool`                                                                  | Check existence                       |
| `All() []*ValidatedTransform`                                                            | Return all transforms                 |
| `Names() []string`                                                                       | Return all names                      |

#### Transform Helpers

| Function                                                                   | Purpose                               |
|----------------------------------------------------------------------------|---------------------------------------|
| `IsBasicTypeName(name string) bool`                                        | Check if name is a builtin basic type |
| `GenerateTransformName(sources, targets StringOrArray) string`             | Generate unique transform name        |
| `GenerateStub(def *TransformDef) string`                                   | Generate single-source transform stub |
| `GenerateMultiSourceStub(def *TransformDef, sourceFields []string) string` | Generate multi-source transform stub  |

#### Validation

##### `Validate(mf *MappingFile, graph *analyze.TypeGraph) *diagnostic.Diagnostics`

- **Purpose:** Structural validation of mapping file against type graph
- **Checks:** Duplicate transforms, type existence, path validity, hint validity, transform references

##### `validateFieldMapping(...)`

- **Purpose:** Validate a single `FieldMapping` against source/target types

##### `validatePathAgainstType(pathStr string, typeInfo *analyze.TypeInfo) error`

- **Purpose:** Walk a path through `TypeInfo` to verify field existence and exportedness

##### `ResolveTypeID(typeIDStr string, graph *analyze.TypeGraph) *analyze.TypeInfo`

- **Purpose:** Resolve a type identifier string to `TypeInfo`
- **Accepts:** Name-only, `pkg.Name`, or fully-qualified paths

#### Loader Functions

| Function                                        | Purpose                       |
|-------------------------------------------------|-------------------------------|
| `LoadFile(path string) (*MappingFile, error)`   | Load YAML file from path      |
| `Parse(data []byte) (*MappingFile, error)`      | Parse YAML bytes              |
| `Marshal(mf *MappingFile) ([]byte, error)`      | Marshal to YAML bytes         |
| `WriteFile(mf *MappingFile, path string) error` | Write YAML file               |
| `NormalizeTypeMapping(tm *TypeMapping)`         | Normalize a type mapping      |
| `NormalizeMappingFile(mf *MappingFile)`         | Normalize entire mapping file |

#### Path Parsing

| Function                                                      | Purpose                  |
|---------------------------------------------------------------|--------------------------|
| `ParsePath(path string) (FieldPath, error)`                   | Parse single path string |
| `ParsePaths(paths StringOrArray) ([]FieldPath, error)`        | Parse multiple paths     |
| `ParsePathsFromRefs(refs FieldRefArray) ([]FieldPath, error)` | Parse paths from refs    |

#### External Dependencies

- `errors`, `fmt`, `strings` (stdlib)
- `gopkg.in/yaml.v3` (implied by YAML marshaling)
- `common`, `analyze`, `diagnostic` (internal)

---

### Module: `match`

**Location:** `internal/match/`

**Purpose:** Provides name normalization, fuzzy string similarity (Levenshtein distance), type compatibility scoring
using `go/types`, and candidate ranking for automatic field matching.

#### Type Compatibility

##### `TypeCompatibility`

```go
type TypeCompatibility int

const (
  TypeIncompatible TypeCompatibility = iota
  TypeNeedsTransform
  TypeConvertible
  TypeAssignable
  TypeIdentical
)
```

| Method            | Purpose                            |
|-------------------|------------------------------------|
| `String() string` | Human-readable label               |
| `Score() int`     | Numeric ordering (higher = better) |

##### `TypeCompatibilityResult`

```go
type TypeCompatibilityResult struct {
  Compatibility TypeCompatibility
  Reason        string
  SourceType    string
  TargetType    string
}
```

#### Compatibility Functions

| Function                                                                       | Purpose                                                |
|--------------------------------------------------------------------------------|--------------------------------------------------------|
| `ScoreTypeCompatibility(source, target types.Type) TypeCompatibilityResult`    | Score compatibility between two `go/types.Type` values |
| `ScorePointerCompatibility(source, target types.Type) TypeCompatibilityResult` | Wrapper that tries pointer unwrapping/wrapping         |
| `needsTransform(source, target types.Type) bool`                               | Heuristic for cases requiring custom transforms        |
| `IsNumericType(t types.Type) bool`                                             | Check if basic numeric type                            |
| `IsStringType(t types.Type) bool`                                              | Check if string type                                   |

#### String Similarity

| Function                                                         | Purpose                                                     |
|------------------------------------------------------------------|-------------------------------------------------------------|
| `Levenshtein(a, b string) int`                                   | Classic edit distance (O(n*m) time, O(min(n,m)) space)      |
| `LevenshteinNormalized(a, b string) float64`                     | Normalized similarity in [0,1]                              |
| `NormalizedLevenshteinScore(a, b string) float64`                | Similarity after identifier normalization                   |
| `NormalizedLevenshteinScoreWithSuffixStrip(a, b string) float64` | With common suffix removal (`id`, `at`, `utc`, `timestamp`) |

#### Name Normalization

| Function                                         | Purpose                                                        |
|--------------------------------------------------|----------------------------------------------------------------|
| `NormalizeIdent(s string) string`                | Canonicalize: tokenize CamelCase, lowercase, remove separators |
| `NormalizeIdentWithSuffixStrip(s string) string` | Normalize + strip common suffixes                              |
| `TokenizeIdent(s string) []string`               | Tokenize identifier string                                     |
| `tokenizeCamelCase(s string) []string`           | Split CamelCase preserving acronyms                            |
| `stripSeparators(s string) string`               | Remove `_`, `-`, space                                         |

#### Candidate Ranking

##### `Candidate`

```go
type Candidate struct {
  SourceField   *analyze.FieldInfo
  CombinedScore float64
  TypeCompat    TypeCompatibilityResult
}
```

##### `CandidateList`

- **Type:** Slice of `Candidate`

| Method                                                | Purpose                                   |
|-------------------------------------------------------|-------------------------------------------|
| `Best() *Candidate`                                   | Highest-ranked candidate or `nil`         |
| `Top(n int) CandidateList`                            | First n candidates (bounds-safe)          |
| `IsAmbiguous(threshold float64) bool`                 | True if top two scores within threshold   |
| `AboveThreshold(th float64) CandidateList`            | All candidates with score >= threshold    |
| `HighConfidence(minScore, minGap float64) *Candidate` | Single high-confidence candidate or `nil` |

##### `RankCandidates(target *analyze.FieldInfo, sources []analyze.FieldInfo) CandidateList`

- **Purpose:** Create ranked candidates for a target field
- **Behavior:**
    - Filters unexported sources
    - Computes name similarity score
    - Computes type compatibility
    - Combines into `CombinedScore`
    - Sorts descending by score (deterministic tie-breaker: alphabetical by name)

##### `calculateCombinedScore(nameScore float64, typeCompat TypeCompatibility) float64`

- **Purpose:** Mix name similarity and type compatibility into single score

#### External Dependencies

- `strings`, `unicode` (stdlib) — normalization
- `go/types` (stdlib) — type comparison
- `sort`, `math` (stdlib)
- `common`, `analyze` (internal)

---

### Module: `plan`

**Location:** `internal/plan/`

**Purpose:** Resolution pipeline that converts YAML mappings and automatic matches into a deterministic, typed
`ResolvedMappingPlan` used for code generation. Handles nested conversions, ordering dependencies, and diagnostics.
Includes virtual type creation and suggestion exporting.

#### Core Types

##### `ResolvedMappingPlan`

```go
type ResolvedMappingPlan struct {
  TypePairs          []ResolvedTypePair
  TypeGraph          *analyze.TypeGraph
  Diagnostics        diagnostic.Diagnostics
  OriginalTransforms []mapping.TransformDef
}
```

| Method                                             | Purpose                                           |
|----------------------------------------------------|---------------------------------------------------|
| `FindIncompleteMappings() []IncompleteMappingInfo` | Find mappings needing transform but none provided |
| `HasIncompleteMappings() bool`                     | Convenience wrapper                               |

##### `ResolvedTypePair`

```go
type ResolvedTypePair struct {
  SourceType        *analyze.TypeInfo
  TargetType        *analyze.TypeInfo
  Mappings          []ResolvedFieldMapping
  UnmappedTargets   []UnmappedField
  NestedPairs       []NestedConversion
  Requires          []mapping.ArgDef
  IsGeneratedTarget bool
}
```

| Method                             | Purpose                                                    |
|------------------------------------|------------------------------------------------------------|
| `UnusedRequires() []string`        | Returns `Requires` not referenced via `Extra`              |
| `CheckRequireConflicts() []string` | Check if `Requires` names conflict with source field names |

##### `ResolvedFieldMapping`

```go
type ResolvedFieldMapping struct {
  TargetPaths      []mapping.FieldPath
  SourcePaths      []mapping.FieldPath
  Source           MappingSource
  Cardinality      mapping.Cardinality
  Strategy         ConversionStrategy
  Transform        string
  Default          *string
  Extra            []mapping.ExtraVal
  DependsOnTargets []mapping.FieldPath
  Confidence       float64
  Explanation      string
  EffectiveHint    mapping.IntrospectionHint
}
```

##### `MappingSource`

```go
type MappingSource int

const (
  MappingSourceYAML121 MappingSource = iota // Highest priority
  MappingSourceYAMLFields
  MappingSourceYAMLIgnore
  MappingSourceYAMLAuto
  MappingSourceAutoMatched // Lowest priority
)
```

- **Method:** `String() string`

##### `ConversionStrategy`

```go
type ConversionStrategy int

const (
  StrategyDirectAssign ConversionStrategy = iota
  StrategyConvert
  StrategyPointerDeref
  StrategyPointerWrap
  StrategySliceMap
  StrategyPointerNestedCast
  StrategyNestedCast
  StrategyTransform
  StrategyDefault
  StrategyIgnore
)
```

- **Method:** `String() string`

##### `UnmappedField`

```go
type UnmappedField struct {
  TargetField *analyze.FieldInfo
  TargetPath  mapping.FieldPath
  Candidates  match.CandidateList
  Reason      string
}
```

##### `NestedConversion`

```go
type NestedConversion struct {
  SourceType    *analyze.TypeInfo
  TargetType    *analyze.TypeInfo
  ReferencedBy  []mapping.FieldPath
  IsSliceElement bool
  ResolvedPair  *ResolvedTypePair
}
```

##### `IncompleteMappingInfo`

```go
type IncompleteMappingInfo struct {
  TypePair    string
  SourcePath  string
  TargetPath  string
  Source      MappingSource
  Explanation string
}
```

##### `DeducedType` and `pairUsage`

Types to support `deduce_requires` logic for dependency analysis across transforms.

#### Suggestions Model

Types for suggestion reporting and export:

- `SuggestionReport`
- `TypePairReport`
- `MatchReport`
- `UnmappedReport`
- `CandidateReport`
- `ExportConfig`

#### Resolver Functions

| Function                                        | Purpose                                           |
|-------------------------------------------------|---------------------------------------------------|
| `DefaultConfig() ResolutionConfig`              | Returns default resolution configuration          |
| `NewResolver(...)`                              | Create new resolver instance                      |
| `Resolve() (*ResolvedMappingPlan, error)`       | Main entry point for resolution                   |
| `resolveTypePairRecursive(...)`                 | Recursively resolve type pairs                    |
| `resolveTypeMapping(...)`                       | Resolve a single type mapping                     |
| `resolve121Mapping(...)`                        | Resolve YAML 1:1 shorthand mapping                |
| `resolveFieldMapping(...)`                      | Resolve explicit YAML field mapping               |
| `determineStrategy(...)`                        | Pick `ConversionStrategy` for source/target paths |
| `determineStrategyWithHint(...)`                | Strategy selection with introspection hint        |
| `determineStrategyByKind(...)`                  | Low-level strategy selection based on types       |
| `resolveFieldType(...)`                         | Walk path to get field's `TypeInfo`               |
| `autoMatchRemainingFields(...)`                 | Auto-match unmapped target fields                 |
| `determineStrategyFromCandidate(...)`           | Map candidate compatibility to strategy           |
| `populateExtraTargetDependencies(...)`          | Build `DependsOnTargets` from `Extra` references  |
| `detectNestedConversions(...)`                  | Scan for nested type conversions                  |
| `analyzeMappingForNestedConversion(...)`        | Analyze mapping for nested conversion needs       |
| `resolveNestedConversion(...)`                  | Resolve details for nested conversion             |
| `sortMappings(...)`                             | Deterministic ordering for outputs                |
| `collectionElem(...)`                           | Get element type for slice/array                  |
| `deduceRequiresTypes(...)`                      | Infer types for `Requires` arguments              |
| `DefaultExportConfig() ExportConfig`            | Returns default export configuration              |
| `ExportSuggestions(...)`                        | Generate YAML with suggestion                     |
| `ExportSuggestionsYAML(...)`                    | Render suggestions to YAML bytes                  |
| `ExportSuggestionsYAMLWithConfig(...)`          | Render suggestions to YAML with config options    |
| `GenerateReport(...)`                           | Detailed suggestion report                        |
| `FormatReport(report *SuggestionReport) string` | Format report as human-readable string            |

#### Virtual Types

The resolver can generate "virtual" target types for simple compositions (tuples, inline structs) to bridge structural gaps.

| Function                       | Purpose                                                      |
|--------------------------------|--------------------------------------------------------------|
| `preCreateVirtualTypes()`      | Initialize virtual type registry                             |
| `createVirtualTargetType(...)` | Create a virtual struct that helps map N sources to 1 target |
| `remapToGeneratedType(...)`    | Resolve virtual types back to generated real types if needed |

#### Resolver Configuration

##### `ResolutionConfig`

```go
type ResolutionConfig struct {
  MinConfidence      float64
  MinGap             float64
  AmbiguityThreshold float64
  StrictMode         bool
  MaxCandidates      int
  RecursiveResolve   bool
  MaxRecursionDepth  int
}
```

- **Constructor:** `DefaultConfig() ResolutionConfig`

Configuration fields:
- `MinConfidence` — minimum score for auto-accept
- `MinGap` — minimum gap between top candidates
- `AmbiguityThreshold` — threshold for ambiguity detection
- `StrictMode` — fails on any unresolved target fields
- `MaxCandidates` — max candidates to report
- `MaxRecursionDepth` — depth limit for nested resolution
- `RecursiveResolve` — enable/disable recursive nested resolution

##### `Resolver`

```go
type Resolver struct {
  graph         *analyze.TypeGraph
  mappingDef    *mapping.MappingFile
  registry      *mapping.TransformRegistry
  config        ResolutionConfig
  resolvedPairs map[string]*ResolvedTypePair
}
```

- **Constructor:** `NewResolver(graph, mappingDef, config) *Resolver`

#### External Dependencies

- `fmt`, `sort` (stdlib)
- `common`, `analyze`, `mapping`, `match`, `diagnostic` (internal)

---

### Module: `gen`

**Location:** `internal/gen/`

**Purpose:** Generates Go caster functions from a `ResolvedMappingPlan`. Uses `text/template` for rendering, `go/format`
for formatting, and provides utilities for topological sorting of assignments and file output.

#### Core Types

##### `GeneratorConfig`

```go
type GeneratorConfig struct {
  PackageName          string
  OutputDir            string
  GenerateComments     bool
  IncludeUnmappedTODOs bool
}
```

- **Constructor:** `DefaultGeneratorConfig()` — returns sensible defaults

##### `Generator`

```go
type Generator struct {
  config            GeneratorConfig
  graph             *analyze.TypeGraph
  missingTransforms map[string]MissingTransformInfo
  missingTypes      map[string][]MissingTypeInfo
  contextPkgPath    string
}
```

- **Constructor:** `NewGenerator(config GeneratorConfig) *Generator`
- **Method:** `Generate(p *plan.ResolvedMappingPlan) ([]GeneratedFile, error)`

##### `GeneratedFile`

```go
type GeneratedFile struct {
  Filename string
  Content  []byte
}
```

##### `MissingTransformInfo`, `MissingTypeInfo`

Structured data for reporting elements that need to be generated but are missing.

##### `MissingTypesTemplateData`

Data passed to templates for generating missing type definitions.

#### Template Data Types

##### `templateData`

```go
type templateData struct {
  Package           string
  Filename          string
  Imports           []importSpec
  FunctionName      string
  SourceType        typeRef
  TargetType        typeRef
  Assignments       []assignmentData
  TODOs             []string
  NestedCasters     []nestedCasterRef
  MissingTransforms []MissingTransform
  ExtraArgs         []extraArg
  GenerateComments  bool
}
```

##### `importSpec`

```go
type importSpec struct {
  Alias string
  Path  string
}
```

##### `typeRef`

```go
type typeRef struct {
  Package   string
  Name      string
  IsPointer bool
  IsSlice   bool
  ElemRef   *typeRef
}
```

- **Method:** `String() string` — compose full type string

##### `assignmentData`

```go
type assignmentData struct {
  TargetField    string
  SourceExpr     string
  Comment        string
  Strategy       plan.ConversionStrategy
  IsSlice        bool
  SliceBody      string
  NestedCaster   string
  NeedsNilCheck  bool
  NilCheckExpr   string
  NilDefault     string
}
```

##### `nestedCasterRef`

```go
type nestedCasterRef struct {
  Name       string
  SourceType typeRef
  TargetType typeRef
}
```

##### `MissingTransform`

```go
type MissingTransform struct {
  Name       string
  ArgTypes   []string
  ReturnType string
}
```

##### `extraArg`

```go
type extraArg struct {
  Name string
  Type string
}
```

#### Generation Functions

| Function                                                                          | Purpose                                              |
|-----------------------------------------------------------------------------------|------------------------------------------------------|
| `generateTypePair(pair *plan.ResolvedTypePair) (*GeneratedFile, error)`           | Build template data, execute template, format output |
| `generateMissingTransformsFile()`                                                 | Generate stubs for missing transforms                |
| `generateMissingTypesFiles()`                                                     | Generate definitions for missing types               |
| `buildTemplateData(pair *plan.ResolvedTypePair) *templateData`                    | Convert `ResolvedTypePair` to template data          |
| `orderAssignmentsByDependencies(data *templateData, pair *plan.ResolvedTypePair)` | Topologically sort assignments                       |
| `collectNestedCasters(...)`                                                       | Convert `NestedPairs` to `nestedCasterRef` entries   |
| `buildAssignment(m *plan.ResolvedFieldMapping, ...) *assignmentData`              | Create `assignmentData` from resolved mapping        |
| `applyConversionStrategy(...)`                                                    | Dispatch to strategy-specific handling               |
| `filename(pair)`, `functionName(pair)`                                            | Helpers for naming artifacts                         |

#### Strategy Helpers

| Function                              | Strategies Handled          |
|---------------------------------------|-----------------------------|
| `applyConvertStrategy(...)`           | `StrategyConvert`           |
| `applyPointerDerefStrategy(...)`      | `StrategyPointerDeref`      |
| `applyPointerWrapStrategy(...)`       | `StrategyPointerWrap`       |
| `applyNestedCastStrategy(...)`        | `StrategyNestedCast`        |
| `applyPointerNestedCastStrategy(...)` | `StrategyPointerNestedCast` |
| `applyTransformStrategy(...)`         | `StrategyTransform`         |

#### Expression Builders

| Function                                                                    | Purpose                               |
|-----------------------------------------------------------------------------|---------------------------------------|
| `targetFieldExpr(paths []mapping.FieldPath) string`                         | Build target expression (`out.Field`) |
| `sourceFieldExpr(paths []mapping.FieldPath, ...) string`                    | Build source expression (`in.Field`)  |
| `wrapConversion(expr string, targetType *analyze.TypeInfo, imports) string` | Produce `Type(expr)`                  |

#### Collection Mapping (Slices, Arrays, Maps)

| Function                                                                       | Purpose                                                    |
|--------------------------------------------------------------------------------|------------------------------------------------------------|
| `buildSliceMapping(...)`                                                       | Orchestrate slice/array mapping code                       |
| `buildMapMapping(...)`                                                         | Orchestrate map mapping code                               |
| `generateCollectionLoop(srcField, tgtField, srcType, tgtType, imports, depth)` | Unified recursive loop generation for all collection types |
| `generateSliceArrayLoop(...)`, `generateMapLoop(...)`                          | Specific loop generators                                   |
| `isCollection(t *analyze.TypeInfo) bool`                                       | Check if type is a slice, array, or map                    |
| `getSliceElementType(t *analyze.TypeInfo) *analyze.TypeInfo`                   | Get element type for slice/array                           |
| `getMapKeyType(t *analyze.TypeInfo) *analyze.TypeInfo`                         | Get key type for map                                       |
| `getMapValueType(t *analyze.TypeInfo) *analyze.TypeInfo`                       | Get value type for map                                     |
| `buildValueConversion(srcExpr, srcType, tgtType, tgtTypeStr) string`           | Per-element/value conversion logic                         |

**`generateCollectionLoop` Details:**
- **Recursion:** If elements are also collections (e.g., `[][]T` or `map[K][]V`), recursively generates nested loops
- **Unique Variables:** Uses `depth` parameter to generate unique loop variables (`i_0`, `k_1`, `v_1`) avoiding shadowing
- **Initialization:** Generates `make()` calls with proper type and capacity for slices and maps
- **Struct Conversion:** Delegates to `buildValueConversion` which calls nested casters for struct elements

#### Type Helpers

| Function                                                     | Purpose                             |
|--------------------------------------------------------------|-------------------------------------|
| `getFieldType(typeInfo, fieldPath) *analyze.TypeInfo`        | Alias for `getFieldTypeInfo`        |
| `getFieldTypeInfo(typeInfo, fieldPath) *analyze.TypeInfo`    | Navigate dotted path to field type  |
| `findFieldInStruct(structType, fieldName) *analyze.TypeInfo` | Find field in struct by name        |
| `getFieldTypeString(typeInfo, fieldPath, imports) string`    | String representation of field type |
| `typeRefString(t *analyze.TypeInfo, imports) string`         | Go type string for `TypeInfo`       |
| `getPkgName(pkgPath string) string`                          | Get short package name              |
| `addImport(imports, pkgPath)`                                | Manage import aliases               |

#### Zero Value Helpers

| Function                                        | Purpose                                         |
|-------------------------------------------------|-------------------------------------------------|
| `zeroValue(typeInfo, paths) string`             | Determine zero/default literal for target field |
| `zeroValueForType(ft *analyze.TypeInfo) string` | Zero value for a `TypeInfo`                     |
| `zeroValueForBasicType(name string) string`     | Zero value for basic type name                  |

#### Type Comparison

| Function                                        | Purpose                                  |
|-------------------------------------------------|------------------------------------------|
| `typesIdentical(a, b *analyze.TypeInfo) bool`   | Compare ID and Kind                      |
| `typesConvertible(a, b *analyze.TypeInfo) bool` | Check basic-to-basic or same named types |

#### Transform Helpers

| Function                                                      | Purpose                                 |
|---------------------------------------------------------------|-----------------------------------------|
| `buildTransformArgs(paths, pair) string`                      | Build comma-separated arg list          |
| `identifyMissingTransforms(pair, imports) []MissingTransform` | Collect unresolved transform references |

#### Topological Sort

##### `topoSortAssignments(n int, depsFn func(i int) []int) ([]int, error)`

- **Purpose:** Deterministic topological sort of assignments
- **Behavior:** Uses indegree/adjacency, deterministic ready queue (sorted), returns error on cycle

#### File Writing

| Constant/Function                                                      | Purpose                                             |
|------------------------------------------------------------------------|-----------------------------------------------------|
| `dirPerm = 0o755`                                                      | Directory permissions                               |
| `filePerm = 0o644`                                                     | File permissions                                    |
| `WriteFiles(files []GeneratedFile, outputDir string) error`            | Create dir and write all generated files            |
| `writeDebugUnformatted(outDir, filename string, content []byte) error` | Best-effort write of unformatted code for debugging |

#### Template

- `casterTemplate` (`var`) — `text/template` that renders:
    - Package declaration and imports
    - Function signature with `typeRef.String`
    - Assignments using `assignmentData` fields
    - Missing transform stubs

#### External Dependencies

- `bytes`, `fmt`, `go/format`, `path`, `path/filepath`, `sort`, `strings`, `text/template`, `os`, `errors` (stdlib)
- `common`, `analyze`, `mapping`, `plan` (internal)

---

## Summary

The caster-generator codebase follows a clean layered architecture:

1. **Foundation Layer** (`common`, `diagnostic`) — shared utilities and error handling
2. **Analysis Layer** (`analyze`) — Go type introspection
3. **Matching Layer** (`match`) — fuzzy field matching algorithms
4. **Schema Layer** (`mapping`) — YAML configuration and validation
5. **Planning Layer** (`plan`) — resolution pipeline combining all inputs
6. **Generation Layer** (`gen`) — final code output

Each layer depends only on layers below it, maintaining clear separation of concerns and enabling independent testing of
each module.
