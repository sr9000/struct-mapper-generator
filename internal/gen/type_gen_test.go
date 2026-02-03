package gen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"caster-generator/internal/analyze"
	"caster-generator/internal/plan"
)

func TestGenerator_GenerateStruct_Basic(t *testing.T) {
	// Setup a simple generated target type
	targetType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "Name", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	pair := &plan.ResolvedTypePair{
		TargetType:        targetType,
		IsGeneratedTarget: true,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	imports := make(map[string]importSpec)

	result, err := gen.GenerateStruct(pair, imports)

	require.NoError(t, err)
	assert.Contains(t, result, "type Order struct {")
	assert.Contains(t, result, "ID int64 `json:\"iD\"`")
	assert.Contains(t, result, "Name string `json:\"name\"`")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(result), "}"))
}

func TestGenerator_GenerateStruct_WithPointerField(t *testing.T) {
	// Setup a target type with pointer fields
	targetType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "Product"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "Price", Exported: true, Type: &analyze.TypeInfo{
				Kind: analyze.TypeKindPointer,
				ElemType: &analyze.TypeInfo{
					ID: analyze.TypeID{Name: "float64"}, Kind: analyze.TypeKindBasic,
				},
			}},
		},
	}

	pair := &plan.ResolvedTypePair{
		TargetType:        targetType,
		IsGeneratedTarget: true,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	imports := make(map[string]importSpec)

	result, err := gen.GenerateStruct(pair, imports)

	require.NoError(t, err)
	assert.Contains(t, result, "type Product struct {")
	assert.Contains(t, result, "ID string")
	assert.Contains(t, result, "Price *float64")
}

func TestGenerator_GenerateStruct_WithSliceField(t *testing.T) {
	// Setup a target type with slice field
	targetType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "Container"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{Name: "Items", Exported: true, Type: &analyze.TypeInfo{
				Kind: analyze.TypeKindSlice,
				ElemType: &analyze.TypeInfo{
					ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
				},
			}},
		},
	}

	pair := &plan.ResolvedTypePair{
		TargetType:        targetType,
		IsGeneratedTarget: true,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	imports := make(map[string]importSpec)

	result, err := gen.GenerateStruct(pair, imports)

	require.NoError(t, err)
	assert.Contains(t, result, "type Container struct {")
	assert.Contains(t, result, "Items []string")
}

func TestGenerator_GenerateStruct_NotGenerated(t *testing.T) {
	// Setup a non-generated target type (should return empty)
	targetType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "Order"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: false,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "int64"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	pair := &plan.ResolvedTypePair{
		TargetType:        targetType,
		IsGeneratedTarget: false,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	imports := make(map[string]importSpec)

	result, err := gen.GenerateStruct(pair, imports)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGenerator_GenerateStruct_NilTargetType(t *testing.T) {
	pair := &plan.ResolvedTypePair{
		TargetType:        nil,
		IsGeneratedTarget: true,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	imports := make(map[string]importSpec)

	result, err := gen.GenerateStruct(pair, imports)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "target type is nil")
	assert.Empty(t, result)
}

func TestGenerator_GenerateStruct_WithNestedStruct(t *testing.T) {
	// Setup a target type with nested struct field
	nestedType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "Address"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: false,
	}

	targetType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "Customer"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{Name: "Name", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "Address", Exported: true, Type: nestedType},
		},
	}

	pair := &plan.ResolvedTypePair{
		TargetType:        targetType,
		IsGeneratedTarget: true,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	imports := make(map[string]importSpec)

	result, err := gen.GenerateStruct(pair, imports)

	require.NoError(t, err)
	assert.Contains(t, result, "type Customer struct {")
	assert.Contains(t, result, "Name string")
	assert.Contains(t, result, "Address")
}

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"A", "a"},
		{"Name", "name"},
		{"ID", "iD"},
		{"firstName", "firstName"},
		{"URL", "uRL"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := lowerFirst(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerator_GenerateStruct_SliceOfPointerToGenerated(t *testing.T) {
	// The element type (TargetItem) is also generated
	elemType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "TargetItem"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{Name: "Name", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}

	// Target type with slice of pointers to generated type
	targetType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "example/warehouse", Name: "Target"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "Items", Exported: true, Type: &analyze.TypeInfo{
				Kind:        analyze.TypeKindSlice,
				IsGenerated: true,
				ElemType: &analyze.TypeInfo{
					Kind:        analyze.TypeKindPointer,
					IsGenerated: true,
					ElemType:    elemType,
				},
			}},
		},
	}

	pair := &plan.ResolvedTypePair{
		TargetType:        targetType,
		IsGeneratedTarget: true,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	gen.contextPkgPath = "example/warehouse"
	imports := make(map[string]importSpec)

	result, err := gen.GenerateStruct(pair, imports)

	require.NoError(t, err)
	assert.Contains(t, result, "type Target struct {")
	assert.Contains(t, result, "ID string")
	// The Items field should be []*TargetItem without package prefix
	assert.Contains(t, result, "Items []*TargetItem")
	assert.NotContains(t, result, "warehouse.TargetItem")
}
