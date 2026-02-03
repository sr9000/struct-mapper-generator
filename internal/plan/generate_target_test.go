package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

func TestGenerateTarget_SingleMapping(t *testing.T) {
	yamlContent := `
version: "1"
mappings:
  - source: test/source.Source
    target: test/target.Target
    generate_target: true
    fields:
      - source: ID
        target: ID
`
	mf, err := mapping.Parse([]byte(yamlContent))
	require.NoError(t, err)

	assert.True(t, mf.TypeMappings[0].GenerateTarget)

	// Create minimal graph
	graph := analyze.NewTypeGraph()
	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Source"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	resolver := NewResolver(graph, mf, DefaultConfig())
	result, err := resolver.Resolve()
	require.NoError(t, err)

	require.Len(t, result.TypePairs, 1)
	tp := result.TypePairs[0]

	assert.Equal(t, "Source", tp.SourceType.ID.Name)
	assert.Equal(t, "Target", tp.TargetType.ID.Name)
	assert.True(t, tp.IsGeneratedTarget, "IsGeneratedTarget should be true")
	assert.True(t, tp.TargetType.IsGenerated, "TargetType.IsGenerated should be true")
}

func TestGenerateTarget_NestedTypes(t *testing.T) {
	yamlContent := `
version: "1"
mappings:
  - source: test/source.Source
    target: test/target.Target
    generate_target: true
    fields:
      - source: ID
        target: ID
    auto:
      - source: Items
        target: Items
  - source: test/source.Item
    target: test/target.Item
    generate_target: true
    fields:
      - source: Name
        target: Name
`
	mf, err := mapping.Parse([]byte(yamlContent))
	require.NoError(t, err)

	// Create minimal graph with nested types
	graph := analyze.NewTypeGraph()

	itemType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Item"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Name", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
		},
	}
	graph.Types[itemType.ID] = itemType

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Source"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: &analyze.TypeInfo{
				ID: analyze.TypeID{Name: "string"}, Kind: analyze.TypeKindBasic,
			}},
			{Name: "Items", Exported: true, Type: &analyze.TypeInfo{
				Kind:     analyze.TypeKindSlice,
				ElemType: itemType,
			}},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	resolver := NewResolver(graph, mf, DefaultConfig())
	result, err := resolver.Resolve()
	require.NoError(t, err)

	require.Len(t, result.TypePairs, 2)

	// First pair: Source -> Target
	tp1 := result.TypePairs[0]
	assert.Equal(t, "Source", tp1.SourceType.ID.Name)
	assert.Equal(t, "Target", tp1.TargetType.ID.Name)
	assert.True(t, tp1.IsGeneratedTarget, "First pair IsGeneratedTarget should be true")

	// Check that Items field references the generated target type
	itemsField := findField(tp1.TargetType.Fields, "Items")
	require.NotNil(t, itemsField)
	require.NotNil(t, itemsField.Type)
	assert.Equal(t, analyze.TypeKindSlice, itemsField.Type.Kind)

	if itemsField.Type.ElemType != nil {
		// The element should reference test/target.Item, not test/source.Item
		assert.Equal(t, "Item", itemsField.Type.ElemType.ID.Name)
		assert.Equal(t, "test/target", itemsField.Type.ElemType.ID.PkgPath)
	}

	// Second pair: Item -> Item
	tp2 := result.TypePairs[1]
	assert.Equal(t, "Item", tp2.SourceType.ID.Name)
	assert.Equal(t, "Item", tp2.TargetType.ID.Name)
	assert.True(t, tp2.IsGeneratedTarget, "Second pair IsGeneratedTarget should be true")
}

func findField(fields []analyze.FieldInfo, name string) *analyze.FieldInfo {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}

	return nil
}
