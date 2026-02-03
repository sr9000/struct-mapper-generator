package plan

import (
	"caster-generator/internal/analyze"
	"caster-generator/internal/common"
	"caster-generator/internal/diagnostic"
	"caster-generator/internal/mapping"
	"caster-generator/internal/match"
)

// ResolvedMappingPlan is the final output of the resolution pipeline.
// It contains everything needed for code generation.
type ResolvedMappingPlan struct {
	// TypePairs is the list of resolved type pair mappings.
	TypePairs []ResolvedTypePair
	// TypeGraph holds all analyzed types and packages to allow looking up package names.
	TypeGraph *analyze.TypeGraph
	// Diagnostics contains all warnings and errors from resolution.
	Diagnostics diagnostic.Diagnostics
}

// ArgDef represents a function argument definition.
type ArgDef struct {
	Name string
	Type string
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
	// Requires lists external variables required by this mapping function.
	Requires []mapping.ArgDef
	// IsGeneratedTarget is true if the target type is generated from the mapping.
	IsGeneratedTarget bool
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
	// Default value to use if source is empty.
	Default *string
	// Confidence score for auto-matched mappings (0-1).
	Confidence float64
	// Explanation describes why this mapping was chosen.
	Explanation string
	// EffectiveHint is the introspection hint computed for this mapping.
	// Controls whether nested fields are recursively resolved or treated as single units.
	EffectiveHint mapping.IntrospectionHint
	// Extra lists additional info field paths from the source type.
	Extra []mapping.ExtraVal
	// DependsOnTargets lists target field paths that must be assigned before this mapping.
	// Derived from extra.def.target references (and potentially other implicit dependencies).
	DependsOnTargets []mapping.FieldPath
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
		return common.UnknownStr
	}
}

// ConversionStrategy describes how to perform the field conversion.
type ConversionStrategy int

const (
	// StrategyDirectAssign - direct assignment (types are VerdictIdentical or VerdictAssignable).
	StrategyDirectAssign ConversionStrategy = iota
	// StrategyConvert - explicit Go type conversion.
	StrategyConvert
	// StrategyPointerDeref - dereference pointer with nil check.
	StrategyPointerDeref
	// StrategyPointerWrap - take address to create pointer.
	StrategyPointerWrap
	// StrategySliceMap - map over slice elements.
	StrategySliceMap
	// StrategyPointerNestedCast - call nested caster on pointer with nil check.
	StrategyPointerNestedCast
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
	case StrategyPointerNestedCast:
		return "pointer_nested_cast"
	case StrategyNestedCast:
		return "nested_cast"
	case StrategyTransform:
		return "transform"
	case StrategyDefault:
		return "default"
	case StrategyIgnore:
		return "ignore"
	default:
		return common.UnknownStr
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

// UnusedRequires returns a list of required arguments that are not used in any mapping.
func (p *ResolvedTypePair) UnusedRequires() []string {
	if len(p.Requires) == 0 {
		return nil
	}

	used := make(map[string]bool)

	for _, m := range p.Mappings {
		// Use struct Name for ExtraVal
		for _, extra := range m.Extra {
			used[extra.Name] = true
		}
		// Also transforms might use them implicitly?
		// Currently only Extra tracks usage of these external args as passed to transforms.
		// If a transform uses a required arg directly without listing it in 'extra',
		// we wouldn't know unless we parse the transform code (which we don't).
		// But in the generated code, we would pass 'extra' args to transforms.
		// Wait, 'Extra' in mapping is currently defined as "lists additional info field paths from the source type".
		// But the user request implies `requires` (external vars) should also be usable.
		// If `Extra` lists something that matches a `Requires` name, it's used.
	}

	var unused []string

	for _, req := range p.Requires {
		if !used[req.Name] {
			// Check if name conflicts with source type fields?
			// The user asked for "push warning if requires conflicts with source field"
			unused = append(unused, req.Name)
		}
	}

	return unused
}

// CheckRequireConflicts checks if any required argument conflicts with a source field name.
func (p *ResolvedTypePair) CheckRequireConflicts() []string {
	if p.SourceType == nil || len(p.Requires) == 0 {
		return nil
	}

	var conflicts []string

	for _, req := range p.Requires {
		for _, field := range p.SourceType.Fields {
			if field.Name == req.Name {
				conflicts = append(conflicts, req.Name)
				break
			}
		}
	}

	return conflicts
}
