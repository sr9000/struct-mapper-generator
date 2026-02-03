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
	require.Len(t, files, 2) // caster file + missing_transforms.go

	content := string(files[0].Content)
	assert.Contains(t, content, "ConcatNames(in.FirstName, in.LastName)")
}

func TestGenerator_Generate_MissingTransformStubs(t *testing.T) {
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
			{Name: "CustomerID", Exported: true, Type: &analyze.TypeInfo{
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
						TargetPaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "CustomerID"}}}},
						SourcePaths: []mapping.FieldPath{{Segments: []mapping.PathSegment{{Name: "ID"}}}},
						Strategy:    plan.StrategyTransform,
						Transform:   "ID2CustomerID",
						Explanation: "custom transform",
					},
				},
			},
		},
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)

	require.NoError(t, err)
	require.Len(t, files, 2) // caster file + missing_transforms.go

	// First file is the caster
	casterContent := string(files[0].Content)
	assert.Contains(t, casterContent, "out.CustomerID = ID2CustomerID(in.ID)")

	// Second file is the missing transforms
	transformsContent := string(files[1].Content)
	assert.Contains(t, transformsContent, "func ID2CustomerID(v0 int64) string {")
	assert.Contains(t, transformsContent, `panic("transform ID2CustomerID not implemented")`)
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

func TestGenerateMissingTypesFile_Basic(t *testing.T) {
	// Setup Source
	srcType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "testpkg", Name: "Source"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Type: &analyze.TypeInfo{ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic}},
			{Name: "Name", Type: &analyze.TypeInfo{ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic}},
		},
	}

	// Setup Target (Generated)
	tgtID := analyze.TypeID{PkgPath: "testpkg", Name: "Target"}
	tgtType := &analyze.TypeInfo{
		ID:          tgtID,
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Type: &analyze.TypeInfo{ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic}},
			{Name: "Label", Type: &analyze.TypeInfo{ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic}},
		},
	}

	// Mock TypeGraph
	graph := &analyze.TypeGraph{
		Packages: map[string]*analyze.PackageInfo{
			"testpkg": {
				Name: "testpkg",
				Dir:  "/abs/path/to/testpkg",
			},
		},
	}

	// Setup Plan
	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{
				SourceType:        srcType,
				TargetType:        tgtType,
				IsGeneratedTarget: true,
				Mappings:          []plan.ResolvedFieldMapping{},
			},
		},
		TypeGraph: graph,
	}

	// Generate
	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	// Verify
	found := false

	for _, f := range files {
		if strings.Contains(f.Filename, "missing_types.go") {
			found = true
			content := string(f.Content)
			assert.Contains(t, content, "package testpkg")
			assert.Contains(t, content, "type Target struct")
			assert.Regexp(t, `ID\s+string`, content)
			assert.Regexp(t, `Label\s+string`, content)
			// Should NOT contain "testpkg." in struct definition
			assert.NotContains(t, content, "testpkg.")
		}
	}

	assert.True(t, found, "missing_types.go not generated")
}

func TestGenerateMissingTypesFile_MultipleTypes(t *testing.T) {
	// Two targets in same package
	tgt1 := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "testpkg", Name: "Target1"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{{
			Name: "F",
			Type: &analyze.TypeInfo{
				ID:   analyze.TypeID{Name: "int"},
				Kind: analyze.TypeKindBasic}}},
	}
	tgt2 := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "testpkg", Name: "Target2"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{{
			Name: "G",
			Type: &analyze.TypeInfo{
				ID:   analyze.TypeID{Name: "int"},
				Kind: analyze.TypeKindBasic}}},
	}

	src := &analyze.TypeInfo{ID: analyze.TypeID{Name: "Source"}, Kind: analyze.TypeKindStruct}

	graph := &analyze.TypeGraph{
		Packages: map[string]*analyze.PackageInfo{
			"testpkg": {Name: "testpkg", Dir: "/abs/path/to/testpkg"},
		},
	}

	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{SourceType: src, TargetType: tgt1, IsGeneratedTarget: true},
			{SourceType: src, TargetType: tgt2, IsGeneratedTarget: true},
		},
		TypeGraph: graph,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	missingFiles := 0

	for _, f := range files {
		if strings.Contains(f.Filename, "missing_types.go") {
			missingFiles++
			content := string(f.Content)
			assert.Contains(t, content, "package testpkg")
			assert.Contains(t, content, "type Target1 struct")
			assert.Contains(t, content, "type Target2 struct")
		}
	}

	assert.Equal(t, 1, missingFiles)
}

func TestGenerateMissingTypesFile_CrossPackageReference(t *testing.T) {
	// Target has field of type TargetItem (same package)
	itemType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "testpkg", Name: "TargetItem"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
	}

	tgtType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "testpkg", Name: "Target"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{
				Name: "Items",
				Type: &analyze.TypeInfo{
					Kind: analyze.TypeKindSlice,
					ElemType: &analyze.TypeInfo{
						Kind:     analyze.TypeKindPointer,
						ElemType: itemType,
					},
				},
			},
		},
	}

	src := &analyze.TypeInfo{ID: analyze.TypeID{Name: "Source"}, Kind: analyze.TypeKindStruct}

	graph := &analyze.TypeGraph{
		Packages: map[string]*analyze.PackageInfo{
			"testpkg": {Name: "testpkg", Dir: "/abs/path/to/testpkg"},
		},
	}

	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{SourceType: src, TargetType: tgtType, IsGeneratedTarget: true},
			// We don't necessarily need a mapping for TargetItem for this test,
			// just need to check how Target refers to it.
		},
		TypeGraph: graph,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	found := false

	for _, f := range files {
		if strings.Contains(f.Filename, "missing_types.go") {
			found = true
			content := string(f.Content)
			assert.Contains(t, content, "Items []*TargetItem")
			assert.NotContains(t, content, "Items []*testpkg.TargetItem")
		}
	}

	assert.True(t, found)
}

func TestGenerateMissingTypesFile_ExternalTypeReference(t *testing.T) {
	// Target has field of type time.Time
	tgtType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "testpkg", Name: "Target"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{
			{
				Name: "CreatedAt",
				Type: &analyze.TypeInfo{
					ID:   analyze.TypeID{PkgPath: "time", Name: "Time"},
					Kind: analyze.TypeKindStruct,
				},
			},
		},
	}
	src := &analyze.TypeInfo{ID: analyze.TypeID{Name: "Source"}, Kind: analyze.TypeKindStruct}

	graph := &analyze.TypeGraph{
		Packages: map[string]*analyze.PackageInfo{
			"testpkg": {Name: "testpkg", Dir: "/abs/path/to/testpkg"},
			"time":    {Name: "time", Dir: ""}, // External, dir empty?
		},
	}

	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{SourceType: src, TargetType: tgtType, IsGeneratedTarget: true},
		},
		TypeGraph: graph,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	found := false

	for _, f := range files {
		if strings.Contains(f.Filename, "missing_types.go") {
			found = true
			content := string(f.Content)
			assert.Contains(t, content, `import (`)
			assert.Contains(t, content, `"time"`)
			assert.Contains(t, content, "CreatedAt time.Time")
		}
	}

	assert.True(t, found)
}

func TestGenerateMissingTypesFile_DifferentPackages(t *testing.T) {
	tgt1 := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "pkg1", Name: "Target"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
	}
	tgt2 := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "pkg2", Name: "Target"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
	}
	src := &analyze.TypeInfo{ID: analyze.TypeID{Name: "Source"}, Kind: analyze.TypeKindStruct}

	graph := &analyze.TypeGraph{
		Packages: map[string]*analyze.PackageInfo{
			"pkg1": {Name: "pkg1", Dir: "/path/to/pkg1"},
			"pkg2": {Name: "pkg2", Dir: "/path/to/pkg2"},
		},
	}

	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{SourceType: src, TargetType: tgt1, IsGeneratedTarget: true},
			{SourceType: src, TargetType: tgt2, IsGeneratedTarget: true},
		},
		TypeGraph: graph,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	missingFiles := 0

	for _, f := range files {
		if strings.Contains(f.Filename, "missing_types.go") {
			missingFiles++

			content := string(f.Content)
			if strings.Contains(content, "package pkg1") {
				assert.Contains(t, f.Filename, "pkg1")
			} else if strings.Contains(content, "package pkg2") {
				assert.Contains(t, f.Filename, "pkg2")
			}
		}
	}

	assert.Equal(t, 2, missingFiles)
}

func TestGenerateMissingTypesFile_NoPackagePath(t *testing.T) {
	// Target has empty PkgPath -> should be embedded in caster file
	tgtType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "", Name: "Target"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields: []analyze.FieldInfo{{
			Name: "F",
			Type: &analyze.TypeInfo{
				ID:   analyze.TypeID{Name: "int"},
				Kind: analyze.TypeKindBasic}}},
	}
	src := &analyze.TypeInfo{ID: analyze.TypeID{Name: "Source"}, Kind: analyze.TypeKindStruct}

	// Empty graph ok?
	graph := &analyze.TypeGraph{Packages: map[string]*analyze.PackageInfo{}}

	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{SourceType: src, TargetType: tgtType, IsGeneratedTarget: true},
		},
		TypeGraph: graph,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotContains(t, f.Filename, "missing_types.go")

		if strings.HasSuffix(f.Filename, ".go") {
			content := string(f.Content)
			assert.Contains(t, content, "type Target struct")
		}
	}
}

func TestCasterFile_ImportsGeneratedType(t *testing.T) {
	tgtType := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "testpkg", Name: "Target"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
	}
	src := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "srcpkg", Name: "Source"},
		Kind: analyze.TypeKindStruct,
	}

	graph := &analyze.TypeGraph{
		Packages: map[string]*analyze.PackageInfo{
			"testpkg": {Name: "testpkg", Dir: "/path/to/testpkg"},
			"srcpkg":  {Name: "srcpkg", Dir: "/path/to/srcpkg"},
		},
	}

	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{SourceType: src, TargetType: tgtType, IsGeneratedTarget: true},
		},
		TypeGraph: graph,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	for _, f := range files {
		if !strings.Contains(f.Filename, "missing_types.go") {
			// This is the caster file
			content := string(f.Content)
			assert.Contains(t, content, `import (`)
			// Should import testpkg
			assert.Contains(t, content, `"testpkg"`)
			// Function signature return type
			assert.Contains(t, content, "testpkg.Target")
			// Instantiation
			assert.Contains(t, content, "out := testpkg.Target{}")
		}
	}
}

func TestTypeRefString_ContextPackagePath(t *testing.T) {
	g := &Generator{}
	imports := make(map[string]importSpec)

	// Case 1: Matching context -> no prefix
	g.contextPkgPath = "my/pkg"
	typMatched := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "my/pkg", Name: "Foo"},
		Kind: analyze.TypeKindStruct,
	}
	assert.Equal(t, "Foo", g.typeRefString(typMatched, imports))
	assert.Empty(t, imports)

	// Case 2: Different context -> prefix + import
	typOther := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "other/pkg", Name: "Bar"},
		Kind: analyze.TypeKindStruct,
	}
	assert.Equal(t, "pkg.Bar", g.typeRefString(typOther, imports))
	assert.Contains(t, imports, "other/pkg")
}
