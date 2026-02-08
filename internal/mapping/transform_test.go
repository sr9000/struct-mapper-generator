package mapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"caster-generator/internal/analyze"
)

func TestBuildRegistry(t *testing.T) {
	yaml := `
mappings: []
transforms:
  - name: PriceToAmount
    source_type: int
    target_type: float64
    description: Converts price in cents to amount in dollars
  - name: StringToID
    source_type: string
    target_type: string
    package: mypackage
    func: ConvertStringToID
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := analyze.NewTypeGraph()
	registry, errs := BuildRegistry(mf, graph)

	assert.Empty(t, errs)
	assert.NotNil(t, registry)

	// Check PriceToAmount
	pt := registry.Get("PriceToAmount")
	require.NotNil(t, pt)
	assert.Equal(t, "int", pt.Def.SourceType)
	assert.Equal(t, "float64", pt.Def.TargetType)
	assert.Equal(t, "PriceToAmount", pt.FuncCall())

	// Check StringToID
	st := registry.Get("StringToID")
	require.NotNil(t, st)
	assert.Equal(t, "mypackage.ConvertStringToID", st.FuncCall())
}

func TestBuildRegistry_WithCustomTypes(t *testing.T) {
	yaml := `
mappings: []
transforms:
  - name: OrderTransform
    source_type: store.Order
    target_type: warehouse.Order
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	registry, errs := BuildRegistry(mf, graph)

	assert.Empty(t, errs)

	ot := registry.Get("OrderTransform")
	require.NotNil(t, ot)
	assert.NotNil(t, ot.SourceType)
	assert.NotNil(t, ot.TargetType)
	assert.Equal(t, "Order", ot.SourceType.ID.Name)
	assert.Equal(t, "Order", ot.TargetType.ID.Name)
}

func TestBuildRegistry_MissingTypes(t *testing.T) {
	yaml := `
mappings: []
transforms:
  - name: BadTransform
    source_type: nonexistent.Type
    target_type: warehouse.Order
`
	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	graph := buildTestTypeGraph()
	registry, errs := BuildRegistry(mf, graph)

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "nonexistent.Type")

	// Registry should still be built with partial info
	assert.NotNil(t, registry)
	bt := registry.Get("BadTransform")
	require.NotNil(t, bt)
	assert.Nil(t, bt.SourceType)
}

func TestTransformRegistry_Has(t *testing.T) {
	registry := NewTransformRegistry()
	registry.transforms["test"] = &ValidatedTransform{
		Def: &TransformDef{Name: "test"},
	}

	assert.True(t, registry.Has("test"))
	assert.False(t, registry.Has("nonexistent"))
}

func TestTransformRegistry_Add(t *testing.T) {
	registry := NewTransformRegistry()
	def := &TransformDef{
		Name:       "TestTransform",
		SourceType: "string",
		TargetType: "int",
	}

	registry.Add(def)

	assert.True(t, registry.Has("TestTransform"))
	vt := registry.Get("TestTransform")
	assert.Equal(t, "string", vt.Def.SourceType)
}

func TestTransformRegistry_All(t *testing.T) {
	registry := NewTransformRegistry()
	registry.transforms["a"] = &ValidatedTransform{Def: &TransformDef{Name: "a"}}
	registry.transforms["b"] = &ValidatedTransform{Def: &TransformDef{Name: "b"}}

	all := registry.All()
	assert.Len(t, all, 2)
}

func TestTransformRegistry_Names(t *testing.T) {
	registry := NewTransformRegistry()
	registry.transforms["alpha"] = &ValidatedTransform{Def: &TransformDef{Name: "alpha"}}
	registry.transforms["beta"] = &ValidatedTransform{Def: &TransformDef{Name: "beta"}}

	names := registry.Names()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
}

func TestGenerateStub(t *testing.T) {
	def := &TransformDef{
		Name:        "PriceToAmount",
		Func:        "PriceToAmount",
		SourceType:  "int",
		TargetType:  "float64",
		Description: "converts cents to dollars",
	}

	stub := GenerateStub(def)

	assert.Contains(t, stub, "func PriceToAmount(src int) float64")
	assert.Contains(t, stub, "converts cents to dollars")
	assert.Contains(t, stub, "panic")
}

func TestGenerateStub_DefaultTypes(t *testing.T) {
	def := &TransformDef{
		Name: "GenericTransform",
		Func: "GenericTransform",
	}

	stub := GenerateStub(def)

	assert.Contains(t, stub, "func GenericTransform(src interface{}) interface{}")
}

func TestGenerateMultiSourceStub(t *testing.T) {
	def := &TransformDef{
		Name:       "ConcatNames",
		Func:       "ConcatNames",
		TargetType: "string",
	}

	stub := GenerateMultiSourceStub(def, []string{"FirstName", "LastName"})

	assert.Contains(t, stub, "func ConcatNames(")
	assert.Contains(t, stub, "firstName interface{}")
	assert.Contains(t, stub, "lastName interface{}")
	assert.Contains(t, stub, ") string")
}

func TestGenerateTransformName(t *testing.T) {
	tests := []struct {
		sources  StringOrArray
		targets  StringOrArray
		expected string
	}{
		{
			sources:  StringOrArray{"FirstName", "LastName"},
			targets:  StringOrArray{"FullName"},
			expected: "FirstNameLastNameToFullName",
		},
		{
			sources:  StringOrArray{"Price"},
			targets:  StringOrArray{"Amount", "Total"},
			expected: "PriceToAmountTotal",
		},
		{
			sources:  StringOrArray{"Items[].ProductID"},
			targets:  StringOrArray{"ProductIDs"},
			expected: "ProductIDToProductIDs",
		},
		{
			sources:  StringOrArray{"Address.Street", "Address.City"},
			targets:  StringOrArray{"FullAddress"},
			expected: "StreetCityToFullAddress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GenerateTransformName(tt.sources, tt.targets)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFieldName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"Name", "Name"},
		{"Address.Street", "Street"},
		{"Items[]", "Items"},
		{"Items[].ProductID", "ProductID"},
		{"Orders[].Items[].Name", "Name"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractFieldName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBasicTypeName(t *testing.T) {
	assert.True(t, IsBasicTypeName("int"))
	assert.True(t, IsBasicTypeName("string"))
	assert.True(t, IsBasicTypeName("float64"))
	assert.True(t, IsBasicTypeName("bool"))
	assert.True(t, IsBasicTypeName("byte"))
	assert.True(t, IsBasicTypeName("rune"))

	assert.False(t, IsBasicTypeName("Order"))
	assert.False(t, IsBasicTypeName("store.Order"))
	assert.False(t, IsBasicTypeName(""))
}
