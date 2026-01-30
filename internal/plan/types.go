package plan

import (
	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/match"
)

// unknownStr is used for unknown enum values.
const unknownStr = "unknown"

// ResolvedMappingPlan is the final output of the resolution pipeline.
// It contains everything needed for code generation.
type ResolvedMappingPlan struct {
	// TypePairs is the list of resolved type pair mappings.
	TypePairs []ResolvedTypePair
	// Diagnostics contains all warnings and errors from resolution.
	Diagnostics Diagnostics
}

// ResolvedTypePair represents a fully resolved mapping between two struct types.
type ResolvedTypePair struct {
	// Source type being converted from.
	SourceType *analyze.TypeInfo
	// Target type being converted to.
	TargetType *analyze.TypeInfo
	// Mappings is the list of resolved field mappings.
	Mappings []ResolvedFieldMapping
	// UnmappedTargets are target fields that could not be mapped.
	UnmappedTargets []UnmappedField
	// NestedPairs tracks nested struct conversions needed.
	NestedPairs []NestedConversion
}

// ResolvedFieldMapping represents a single resolved field mapping.
type ResolvedFieldMapping struct {
	// Target field(s) to populate.
	TargetPaths []mapping.FieldPath
	// Source field(s) to read from.
	SourcePaths []mapping.FieldPath
	// Source specifies the origin of this mapping rule.
	Source MappingSource
	// Cardinality of the mapping (1:1, 1:N, N:1, N:M).
	Cardinality mapping.Cardinality
	// Strategy describes how the conversion should be performed.
	Strategy ConversionStrategy
	// Transform is the name of the transform function (if needed).
	Transform string
	// Default value to use if no source (literal string).
	Default *string
	// Confidence score for auto-matched mappings (0-1).
	Confidence float64
	// Explanation describes why this mapping was chosen.
	Explanation string
	// EffectiveHint is the introspection hint computed for this mapping.
	// Controls whether nested fields are recursively resolved or treated as single units.
	EffectiveHint mapping.IntrospectionHint
}

// MappingSource indicates where a mapping rule originated.
type MappingSource int

const (
	// MappingSourceYAML121 - from YAML 121 shorthand (highest priority).
	MappingSourceYAML121 MappingSource = iota
	// MappingSourceYAMLFields - from YAML explicit fields section.
	MappingSourceYAMLFields
	// MappingSourceYAMLIgnore - from YAML ignore list.
	MappingSourceYAMLIgnore
	// MappingSourceYAMLAuto - from YAML auto section.
	MappingSourceYAMLAuto
	// MappingSourceAutoMatched - auto-matched by best-effort algorithm.
	MappingSourceAutoMatched
)

// String returns a human-readable source name.
func (s MappingSource) String() string {
	switch s {
	case MappingSourceYAML121:
		return "yaml:121"
	case MappingSourceYAMLFields:
		return "yaml:fields"
	case MappingSourceYAMLIgnore:
		return "yaml:ignore"
	case MappingSourceYAMLAuto:
		return "yaml:auto"
	case MappingSourceAutoMatched:
		return "auto"
	default:
		return unknownStr
	}
}

// ConversionStrategy describes how to perform the field conversion.
type ConversionStrategy int

const (
	// StrategyDirectAssign - direct assignment (types are identical or assignable).
	StrategyDirectAssign ConversionStrategy = iota
	// StrategyConvert - explicit Go type conversion.
	StrategyConvert
	// StrategyPointerDeref - dereference pointer with nil check.
	StrategyPointerDeref
	// StrategyPointerWrap - take address to create pointer.
	StrategyPointerWrap
	// StrategySliceMap - map over slice elements.
	StrategySliceMap
	// StrategyNestedCast - call nested caster function.
	StrategyNestedCast
	// StrategyTransform - call custom transform function.
	StrategyTransform
	// StrategyDefault - set default value.
	StrategyDefault
	// StrategyIgnore - explicitly ignored field.
	StrategyIgnore
)

// String returns a human-readable strategy name.
func (s ConversionStrategy) String() string {
	switch s {
	case StrategyDirectAssign:
		return "direct_assign"
	case StrategyConvert:
		return "convert"
	case StrategyPointerDeref:
		return "pointer_deref"
	case StrategyPointerWrap:
		return "pointer_wrap"
	case StrategySliceMap:
		return "slice_map"
	case StrategyNestedCast:
		return "nested_cast"
	case StrategyTransform:
		return "transform"
	case StrategyDefault:
		return "default"
	case StrategyIgnore:
		return "ignore"
	default:
		return unknownStr
	}
}

// UnmappedField represents a target field that couldn't be mapped.
type UnmappedField struct {
	// TargetField is the unmapped field.
	TargetField *analyze.FieldInfo
	// TargetPath is the full path to the field.
	TargetPath mapping.FieldPath
	// Candidates are the ranked potential matches (for suggestions).
	Candidates match.CandidateList
	// Reason explains why it wasn't mapped.
	Reason string
}

// NestedConversion tracks a required nested struct conversion.
type NestedConversion struct {
	// SourceType is the nested source struct type.
	SourceType *analyze.TypeInfo
	// TargetType is the nested target struct type.
	TargetType *analyze.TypeInfo
	// ReferencedBy tracks which field mappings need this conversion.
	ReferencedBy []mapping.FieldPath
	// IsSliceElement indicates this conversion is for slice elements.
	IsSliceElement bool
	// ResolvedPair contains the recursively resolved mapping for this nested pair.
	// May be nil if resolution was deferred or failed.
	ResolvedPair *ResolvedTypePair
}

// Diagnostics holds all diagnostic information from resolution.
type Diagnostics struct {
	Errors   []Diagnostic
	Warnings []Diagnostic
}

// Diagnostic represents a single diagnostic message.
type Diagnostic struct {
	// Severity of the diagnostic.
	Severity DiagnosticSeverity
	// Code is a unique identifier for this type of diagnostic.
	Code string
	// Message is the human-readable description.
	Message string
	// TypePair identifies which type mapping this relates to (if any).
	TypePair string
	// FieldPath identifies which field this relates to (if any).
	FieldPath string
	// Suggestions are potential fixes or alternatives.
	Suggestions []string
}

// DiagnosticSeverity represents the severity level of a diagnostic.
type DiagnosticSeverity int

const (
	DiagnosticInfo DiagnosticSeverity = iota
	DiagnosticWarning
	DiagnosticError
)

// String returns a human-readable severity name.
func (s DiagnosticSeverity) String() string {
	switch s {
	case DiagnosticInfo:
		return "info"
	case DiagnosticWarning:
		return "warning"
	case DiagnosticError:
		return "error"
	default:
		return "unknown"
	}
}

// AddError adds an error diagnostic.
func (d *Diagnostics) AddError(code, message, typePair, fieldPath string) {
	d.Errors = append(d.Errors, Diagnostic{
		Severity:  DiagnosticError,
		Code:      code,
		Message:   message,
		TypePair:  typePair,
		FieldPath: fieldPath,
	})
}

// AddWarning adds a warning diagnostic.
func (d *Diagnostics) AddWarning(code, message, typePair, fieldPath string) {
	d.Warnings = append(d.Warnings, Diagnostic{
		Severity:  DiagnosticWarning,
		Code:      code,
		Message:   message,
		TypePair:  typePair,
		FieldPath: fieldPath,
	})
}

// AddInfo adds an info diagnostic.
func (d *Diagnostics) AddInfo(code, message, typePair, fieldPath string) {
	d.Warnings = append(d.Warnings, Diagnostic{
		Severity:  DiagnosticInfo,
		Code:      code,
		Message:   message,
		TypePair:  typePair,
		FieldPath: fieldPath,
	})
}

// HasErrors returns true if there are any error-level diagnostics.
func (d *Diagnostics) HasErrors() bool {
	return len(d.Errors) > 0
}

// Merge combines diagnostics from another Diagnostics instance.
func (d *Diagnostics) Merge(other *Diagnostics) {
	d.Errors = append(d.Errors, other.Errors...)
	d.Warnings = append(d.Warnings, other.Warnings...)
}

// IncompleteMappingInfo describes a mapping that requires a transform but doesn't have one.
type IncompleteMappingInfo struct {
	TypePair    string
	SourcePath  string
	TargetPath  string
	Source      MappingSource
	Explanation string
}

// FindIncompleteMappings returns all mappings that have StrategyTransform but no Transform function defined.
// These mappings cannot be generated and need user intervention.
func (p *ResolvedMappingPlan) FindIncompleteMappings() []IncompleteMappingInfo {
	var incomplete []IncompleteMappingInfo

	for _, tp := range p.TypePairs {
		typePairStr := tp.SourceType.ID.String() + "->" + tp.TargetType.ID.String()

		for _, m := range tp.Mappings {
			if m.Strategy == StrategyTransform && m.Transform == "" {
				info := IncompleteMappingInfo{
					TypePair:    typePairStr,
					Explanation: m.Explanation,
					Source:      m.Source,
				}
				if len(m.SourcePaths) > 0 {
					info.SourcePath = m.SourcePaths[0].String()
				}
				if len(m.TargetPaths) > 0 {
					info.TargetPath = m.TargetPaths[0].String()
				}
				incomplete = append(incomplete, info)
			}
		}
	}

	return incomplete
}

// HasIncompleteMappings returns true if there are any mappings that need transforms but don't have them.
func (p *ResolvedMappingPlan) HasIncompleteMappings() bool {
	return len(p.FindIncompleteMappings()) > 0
}
