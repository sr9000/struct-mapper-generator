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
- `caster-generator/internal/plan`: `ResolvedMappingPlan`, `ResolvedTypePair`, `ResolvedFieldMapping`, `ConversionStrategy`, `NestedConversion`

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
- `caster-generator/internal/mapping`: `MappingFile`, `TypeMapping`, `FieldMapping`, `FieldPath`, `ArgDef`, `ExtraVal`, `TransformRegistry`, `Cardinality`, `IntrospectionHint`, `HintNone`, `HintDive`, `HintFinal`, `ResolveTypeID`, `ParsePath`, `BuildRegistry`, `NewTransformRegistry`
- `caster-generator/internal/match`: `Candidate`, `CandidateList`, `TypeCompatibility`, `TypeIdentical`, `TypeAssignable`, `TypeConvertible`, `TypeNeedsTransform`, `ScorePointerCompatibility`, `RankCandidates`, `DefaultMinScore`, `DefaultMinGap`, `DefaultAmbiguityThreshold`
- `errors`
- `fmt`
- `sort`
- `strings`

## `diagnostic`

**Purpose:** Provides structured warnings, errors, and explanations for the caster generator.

**Key Capabilities:**
- Unmapped field warnings
- Ambiguous match reports with top-N candidates
- Unsafe conversion warnings
- Explanation of mapping decisions

**Dependencies:**
- None
