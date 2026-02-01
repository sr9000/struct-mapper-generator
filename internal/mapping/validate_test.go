package mapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"caster-generator/internal/analyze"
)

// buildTestTypeGraph creates a simple type graph for testing validation.
func buildTestTypeGraph() *analyze.TypeGraph {
	graph := analyze.NewTypeGraph()

	// Create store.Order type
	storeOrderID := analyze.TypeID{PkgPath: "caster-generator/store", Name: "Order"}
	stringType := &analyze.TypeInfo{Kind: analyze.TypeKindBasic}
	intType := &analyze.TypeInfo{Kind: analyze.TypeKindBasic}

	itemType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "caster-generator/store", Name: "Item"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ProductID", Exported: true, Type: stringType, Index: 0},
			{Name: "Quantity", Exported: true, Type: intType, Index: 1},
		},
	}
	graph.Types[itemType.ID] = itemType

	itemSliceType := &analyze.TypeInfo{
		Kind:     analyze.TypeKindSlice,
		ElemType: itemType,
	}

	storeOrder := &analyze.TypeInfo{
		ID:   storeOrderID,
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "OrderID", Exported: true, Type: stringType, Index: 0},
			{Name: "CustomerName", Exported: true, Type: stringType, Index: 1},
			{Name: "Price", Exported: true, Type: intType, Index: 2},
			{Name: "Items", Exported: true, Type: itemSliceType, Index: 3},
			{Name: "internal", Exported: false, Type: stringType, Index: 4},
			{Name: "FirstName", Exported: true, Type: stringType, Index: 5},
			{Name: "LastName", Exported: true, Type: stringType, Index: 6},
		},
	}
	graph.Types[storeOrderID] = storeOrder

	// Create warehouse.Order type
	warehouseOrderID := analyze.TypeID{PkgPath: "caster-generator/warehouse", Name: "Order"}
	warehouseOrder := &analyze.TypeInfo{
		ID:   warehouseOrderID,
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: stringType, Index: 0},
			{Name: "Customer", Exported: true, Type: stringType, Index: 1},
			{Name: "Amount", Exported: true, Type: intType, Index: 2},
			{Name: "Status", Exported: true, Type: stringType, Index: 3},
			{Name: "DisplayName", Exported: true, Type: stringType, Index: 4},
			{Name: "FullName", Exported: true, Type: stringType, Index: 5},
		},
	}
	graph.Types[warehouseOrderID] = warehouseOrder

	return graph
}

func TestValidate_ValidMapping(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: ID
        source: OrderID
      - target: Customer
        source: CustomerName
transforms: []
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.True(t, result.IsValid(), "expected valid mapping, got errors: %v", result.Errors)
}

func TestValidate_121Shorthand(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    121:
      OrderID: ID
      CustomerName: Customer
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.True(t, result.IsValid(), "expected valid mapping, got errors: %v", result.Errors)
}

func TestValidate_Invalid121Shorthand(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    121:
      NonExistent: ID
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	assert.Contains(t, result.Error().Error(), "NonExistent")
}

func TestValidate_MissingSourceType(t *testing.T) {
	yaml := `
mappings:
  - source: nonexistent.Type
    target: warehouse.Order
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Message, "nonexistent.Type")
}

func TestValidate_MissingTargetType(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: nonexistent.Type
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Message, "nonexistent.Type")
}

func TestValidate_InvalidSourceField(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: ID
        source: NonExistentField
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	assert.Contains(t, result.Error().Error(), "NonExistentField")
}

func TestValidate_InvalidTargetField(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: NonExistentField
        source: OrderID
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	assert.Contains(t, result.Error().Error(), "NonExistentField")
}

func TestValidate_UnexportedField(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: ID
        source: internal
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	assert.Contains(t, result.Error().Error(), "not exported")
}

func TestValidate_NestedPath(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: ID
        source: Items[].ProductID
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	// This should validate successfully - path exists
	assert.True(t, result.IsValid(), "errors: %v", result.Errors)
}

func TestValidate_AllowMissingTransform(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: ID
        source: OrderID
        transform: NonExistentTransform
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.True(t, result.IsValid())
}

func TestValidate_DuplicateTransform(t *testing.T) {
	yaml := `
mappings: []
transforms:
  - name: Convert
    source_type: int
    target_type: string
  - name: Convert
    source_type: float64
    target_type: string
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	assert.Contains(t, result.Error().Error(), "duplicate transform")
}

func TestValidate_FieldMappingWithIgnore(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    ignore:
      - Status
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.True(t, result.IsValid())
}

func TestValidate_FieldMappingRequiresSource(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: ID
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.False(t, result.IsValid())
	assert.Contains(t, result.Error().Error(), "must specify")
}

func TestValidate_IgnoreList(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    ignore:
      - Status
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.True(t, result.IsValid())
}

func TestValidate_TypeResolution(t *testing.T) {
	tests := []struct {
		name   string
		source string
		target string
		valid  bool
	}{
		{"full path", "caster-generator/store.Order", "caster-generator/warehouse.Order", true},
		{"short name", "store.Order", "warehouse.Order", true},
		{"type name only", "Order", "Order", true}, // Ambiguous but resolves
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
mappings:
  - source: ` + tt.source + `
    target: ` + tt.target + `
`
			mf, err := Parse([]byte(yaml))
			require.NoError(t, err)

			graph := buildTestTypeGraph()
			result := Validate(mf, graph)

			if tt.valid {
				assert.True(t, result.IsValid(), "errors: %v", result.Errors)
			} else {
				assert.False(t, result.IsValid())
			}
		})
	}
}

func TestValidate_OneToMany(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: [DisplayName, FullName]
        source: CustomerName
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	// 1:many doesn't require transform
	assert.True(t, result.IsValid(), "errors: %v", result.Errors)
}

func TestValidate_ManyToOneRequiresTransform(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: FullName
        source: [FirstName, LastName]
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	// many:1 requires transform
	assert.False(t, result.IsValid())
	assert.Contains(t, result.Error().Error(), "N:1")
	assert.Contains(t, result.Error().Error(), "transform")
}

func TestValidate_ManyToOneWithTransform(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    fields:
      - target: FullName
        source: [FirstName, LastName]
        transform: ConcatNames
transforms:
  - name: ConcatNames
    source_type: string
    target_type: string
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.True(t, result.IsValid(), "errors: %v", result.Errors)
}

func TestValidate_AutoMappings(t *testing.T) {
	yaml := `
mappings:
  - source: store.Order
    target: warehouse.Order
    auto:
      - target: ID
        source: OrderID
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	result := Validate(mf, graph)

	assert.True(t, result.IsValid(), "errors: %v", result.Errors)
}
