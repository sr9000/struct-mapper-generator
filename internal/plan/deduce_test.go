package plan

import (
	"go/types"
	"testing"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"

	"github.com/stretchr/testify/assert"
)

func TestDeduceRequiresTypes(t *testing.T) {
	// Setup generic Resolver (empty config)
	resolver := &Resolver{}

	// Setup types
	intType := types.Typ[types.Int]
	intTypeInfo := &analyze.TypeInfo{
		Kind:   analyze.TypeKindBasic,
		GoType: intType,
	}

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{Name: "Source"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{
				Name: "ID",
				Type: intTypeInfo,
			},
		},
	}

	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{Name: "Target"},
		Kind: analyze.TypeKindStruct,
	}

	nestedSourceType := &analyze.TypeInfo{ID: analyze.TypeID{Name: "NestedSource"}, Kind: analyze.TypeKindStruct}
	nestedTargetType := &analyze.TypeInfo{ID: analyze.TypeID{Name: "NestedTarget"}, Kind: analyze.TypeKindStruct}

	// Child Pair (The one with 'requires')
	childPair := &ResolvedTypePair{
		SourceType: nestedSourceType,
		TargetType: nestedTargetType,
		Requires: []mapping.ArgDef{
			{Name: "ReqID", Type: "interface{}"},
			{Name: "Other", Type: "string"}, // Should not change
		},
	}

	// Parent Pair (The one providing 'extra')
	// We simulate that 'Nested' field mapping triggers valid usage
	targetPath := mapping.FieldPath{Segments: []mapping.PathSegment{{Name: "Nested"}}}

	parentPair := &ResolvedTypePair{
		SourceType: sourceType,
		TargetType: targetType,
		Mappings: []ResolvedFieldMapping{
			{
				TargetPaths: []mapping.FieldPath{targetPath},
				Extra: []mapping.ExtraVal{
					{
						Name: "ReqID",
						Def:  mapping.ExtraDef{Source: "ID"},
					},
				},
			},
		},
		NestedPairs: []NestedConversion{
			{
				ResolvedPair: childPair,
				ReferencedBy: []mapping.FieldPath{targetPath},
			},
		},
	}

	// Plan
	plan := &ResolvedMappingPlan{
		TypePairs: []ResolvedTypePair{*parentPair},
	}

	// Run deduction
	resolver.deduceRequiresTypes(plan)

	// Verify
	assert.Equal(t, "int", childPair.Requires[0].Type, "Should deduce type 'int' from source field 'ID'")
	assert.Equal(t, "string", childPair.Requires[1].Type, "Should not change existing explicit type")
}

func TestDeduceRequiresConflict(t *testing.T) {
	resolver := &Resolver{}

	intTypeInfo := &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Int]}
	stringTypeInfo := &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.String]}

	// Source 1 has int ID
	source1 := &analyze.TypeInfo{
		ID:     analyze.TypeID{Name: "Source1"},
		Kind:   analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{{Name: "ID", Type: intTypeInfo}},
	}

	// Source 2 has string ID
	source2 := &analyze.TypeInfo{
		ID:     analyze.TypeID{Name: "Source2"},
		Kind:   analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{{Name: "ID", Type: stringTypeInfo}},
	}

	childPair := &ResolvedTypePair{
		SourceType: &analyze.TypeInfo{ID: analyze.TypeID{Name: "Child"}},
		TargetType: &analyze.TypeInfo{ID: analyze.TypeID{Name: "ChildTarget"}},
		Requires: []mapping.ArgDef{
			{Name: "ReqID", Type: "interface{}"},
		},
	}

	targetPath := mapping.FieldPath{Segments: []mapping.PathSegment{{Name: "Nested"}}}

	parent1 := &ResolvedTypePair{
		SourceType: source1,
		TargetType: &analyze.TypeInfo{ID: analyze.TypeID{Name: "Target1"}},
		Mappings: []ResolvedFieldMapping{
			{
				TargetPaths: []mapping.FieldPath{targetPath},
				Extra:       []mapping.ExtraVal{{Name: "ReqID", Def: mapping.ExtraDef{Source: "ID"}}},
			},
		},
		NestedPairs: []NestedConversion{{ResolvedPair: childPair, ReferencedBy: []mapping.FieldPath{targetPath}}},
	}

	parent2 := &ResolvedTypePair{
		SourceType: source2,
		TargetType: &analyze.TypeInfo{ID: analyze.TypeID{Name: "Target2"}},
		Mappings: []ResolvedFieldMapping{
			{
				TargetPaths: []mapping.FieldPath{targetPath},
				Extra:       []mapping.ExtraVal{{Name: "ReqID", Def: mapping.ExtraDef{Source: "ID"}}},
			},
		},
		NestedPairs: []NestedConversion{{ResolvedPair: childPair, ReferencedBy: []mapping.FieldPath{targetPath}}},
	}

	plan := &ResolvedMappingPlan{
		TypePairs: []ResolvedTypePair{*parent1, *parent2},
	}

	resolver.deduceRequiresTypes(plan)

	// Should conflict and fallback to interface{}
	assert.Equal(t, "interface{}", childPair.Requires[0].Type)
	assert.True(t, len(plan.Diagnostics.Warnings) > 0)
	assert.Contains(t, plan.Diagnostics.Warnings[0].Code, "conflict")
}
