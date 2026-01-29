package gen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/plan"
)

func TestGenerator_Generate_SimpleTypePair(t *testing.T) {
	// Setup a simple source type
	srcType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/store", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "Name", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	// Setup a simple target type
	tgtType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "Name", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	// Setup resolved mappings
	resolvedPlan := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType: srcType,
				TargetType: tgtType,
				Mappings: []plan.ResolvedFieldMapping{
					{
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						SourcePaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						Strategy:    plan.StrategyDirectAssign,
						Explanation: "exact match",
					},
					{
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "Name"}}}},
						SourcePaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "Name"}}}},
						Strategy:    plan.StrategyDirectAssign,
						Explanation: "exact match",
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 1)

	content := string(files[0].Content)

	// Check file name
	assert.Equal(t, "store_order_to_warehouse_order.go", files[0].Filename)

	// Check generated content contains expected elements
	assert.Contains(t, content, "package casters")
	assert.Contains(t, content, "func StoreOrderToWarehouseOrder")
	assert.Contains(t, content, "out.ID = in.ID")
	assert.Contains(t, content, "out.Name = in.Name")
	assert.Contains(t, content, "return out")
}

func TestGenerator_Generate_WithTypeConversion(t *testing.T) {
	// Source has int, target has int64
	srcType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/store", Name: "Product"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	tgtType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/warehouse", Name: "Product"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	resolvedPlan := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType: srcType,
				TargetType: tgtType,
				Mappings: []plan.ResolvedFieldMapping{
					{
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						SourcePaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						Strategy:    plan.StrategyConvert,
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 1)

	content := string(files[0].Content)
	assert.Contains(t, content, "int64(in.ID)")
}

func TestGenerator_Generate_WithSliceMapping(t *testing.T) {
	elemSrcType := &analyze.TypeInfo{
		ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
	}
	elemTgtType := &analyze.TypeInfo{
		ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
	}

	srcType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/store", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Tags", Exported: true, Type: &analyze.TypeInfo{
				Kind:     analyze.TypeKindSlice,
				ElemType: elemSrcType,
			}},
		},
	}

	tgtType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Tags", Exported: true, Type: &analyze.TypeInfo{
				Kind:     analyze.TypeKindSlice,
				ElemType: elemTgtType,
			}},
		},
	}

	resolvedPlan := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType: srcType,
				TargetType: tgtType,
				Mappings: []plan.ResolvedFieldMapping{
					{
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "Tags"}}}},
						SourcePaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "Tags"}}}},
						Strategy:    plan.StrategySliceMap,
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 1)

	content := string(files[0].Content)
	assert.Contains(t, content, "make([]string, len(in.Tags))")
	assert.Contains(t, content, "for i := range in.Tags")
}

func TestGenerator_Generate_WithUnmappedTODOs(t *testing.T) {
	srcType := &analyze.TypeInfo{
		ID:     analyze.TypeID{PkgPath: "example/store", Name: "Order"},
		Kind:   analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{},
	}

	tgtType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Status", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	resolvedPlan := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType: srcType,
				TargetType: tgtType,
				Mappings:   []plan.ResolvedFieldMapping{},
				UnmappedTargets: []plan.UnmappedField{
					{
						TargetPath: mapping.FieldPath{Segments: []mapping.PathSegment{{Name: "Status"}}},
						Reason:     "no matching source field",
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 1)

	content := string(files[0].Content)
	assert.Contains(t, content, "// TODO: Status - no matching source field")
}

func TestGenerator_Generate_IgnoredMappings(t *testing.T) {
	srcType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/store", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	tgtType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	resolvedPlan := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType: srcType,
				TargetType: tgtType,
				Mappings: []plan.ResolvedFieldMapping{
					{
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						Strategy:    plan.StrategyIgnore,
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 1)

	content := string(files[0].Content)
	// Should not contain assignment for ignored field
	assert.NotContains(t, content, "out.ID = in.ID")
}

func TestGenerator_Generate_WithTransform(t *testing.T) {
	srcType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/store", Name: "Person"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "FirstName", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "LastName", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	tgtType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/warehouse", Name: "Person"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "FullName", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	resolvedPlan := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType: srcType,
				TargetType: tgtType,
				Mappings: []plan.ResolvedFieldMapping{
					{
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "FullName"}}}},
						SourcePaths: []mapping.FieldPath{
							{Segments: []mapping.PathSegment{{Name: "FirstName"}}},
							{Segments: []mapping.PathSegment{{Name: "LastName"}}},
						},
						Strategy:  plan.StrategyTransform,
						Transform: "ConcatNames",
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 1)

	content := string(files[0].Content)
	assert.Contains(t, content, "ConcatNames(in.FirstName, in.LastName)")
}

func TestTypeRef_String(t *testing.T) {
	tests := []struct {
		name     string
		ref      typeRef
		expected string
	}{
		{
			name:     "simple type",
			ref:      typeRef{Name: "string"},
			expected: "string",
		},
		{
			name:     "package qualified type",
			ref:      typeRef{Package: "store", Name: "Order"},
			expected: "store.Order",
		},
		{
			name:     "pointer type",
			ref:      typeRef{Package: "store", Name: "Order", IsPointer: true},
			expected: "*store.Order",
		},
		{
			name: "slice type",
			ref: typeRef{
				IsSlice: true,
				ElemRef: &typeRef{Name: "string"},
			},
			expected: "[]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ref.String())
		})
	}
}

func TestGenerator_filename(t *testing.T) {
	gen := NewGenerator(DefaultGeneratorConfig())

	pair := &plan.ResolvedTypePair{
		SourceType: &analyze.TypeInfo{
			ID: analyze.TypeID{PkgPath: "example/store", Name: "Order"},
		},
		TargetType: &analyze.TypeInfo{
			ID: analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		},
	}

	filename := gen.filename(pair)
	assert.Equal(t, "store_order_to_warehouse_order.go", filename)
}

func TestGenerator_functionName(t *testing.T) {
	gen := NewGenerator(DefaultGeneratorConfig())

	pair := &plan.ResolvedTypePair{
		SourceType: &analyze.TypeInfo{
			ID: analyze.TypeID{PkgPath: "example/store", Name: "Order"},
		},
		TargetType: &analyze.TypeInfo{
			ID: analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		},
	}

	funcName := gen.functionName(pair)
	assert.Equal(t, "StoreOrderToWarehouseOrder", funcName)
}

func TestGenerator_Generate_FormattedOutput(t *testing.T) {
	srcType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/store", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	tgtType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	resolvedPlan := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType: srcType,
				TargetType: tgtType,
				Mappings: []plan.ResolvedFieldMapping{
					{
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						SourcePaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						Strategy:    plan.StrategyDirectAssign,
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 1)

	content := string(files[0].Content)

	// Check that output is properly formatted (no double newlines except intended)
	assert.True(t, strings.HasPrefix(content, "// Code generated by caster-generator"))
	assert.Contains(t, content, "package casters")
}
