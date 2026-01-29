package analyze

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer_LoadPackages(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store", "caster-generator/warehouse")
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Check that packages were loaded
	assert.Contains(t, graph.Packages, "caster-generator/store")
	assert.Contains(t, graph.Packages, "caster-generator/warehouse")

	// Check that types were extracted
	storeOrder := TypeID{PkgPath: "caster-generator/store", Name: "Order"}
	assert.Contains(t, graph.Types, storeOrder)

	warehouseOrder := TypeID{PkgPath: "caster-generator/warehouse", Name: "Order"}
	assert.Contains(t, graph.Types, warehouseOrder)
}

func TestAnalyzer_StoreOrderFields(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store")
	require.NoError(t, err)

	orderID := TypeID{PkgPath: "caster-generator/store", Name: "Order"}
	order := graph.GetType(orderID)
	require.NotNil(t, order)
	assert.Equal(t, TypeKindStruct, order.Kind)

	// Check expected fields exist
	fieldNames := make(map[string]bool)
	for _, f := range order.Fields {
		fieldNames[f.Name] = true
	}

	assert.True(t, fieldNames["ID"], "Order should have ID field")
	assert.True(t, fieldNames["CustomerID"], "Order should have CustomerID field")
	assert.True(t, fieldNames["Status"], "Order should have Status field")
	assert.True(t, fieldNames["TotalCents"], "Order should have TotalCents field")
	assert.True(t, fieldNames["Items"], "Order should have Items field")
	assert.True(t, fieldNames["OrderedAt"], "Order should have OrderedAt field")
}

func TestAnalyzer_FieldTags(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store")
	require.NoError(t, err)

	productID := TypeID{PkgPath: "caster-generator/store", Name: "Product"}
	product := graph.GetType(productID)
	require.NotNil(t, product)

	// Find the SKU field
	var skuField *FieldInfo
	for i := range product.Fields {
		if product.Fields[i].Name == "SKU" {
			skuField = &product.Fields[i]
			break
		}
	}
	require.NotNil(t, skuField)

	// Check JSON tag
	assert.Equal(t, "sku", skuField.JSONName())
	assert.True(t, skuField.HasTag("json"))
}

func TestAnalyzer_SliceField(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store")
	require.NoError(t, err)

	orderID := TypeID{PkgPath: "caster-generator/store", Name: "Order"}
	order := graph.GetType(orderID)
	require.NotNil(t, order)

	// Find Items field
	var itemsField *FieldInfo
	for i := range order.Fields {
		if order.Fields[i].Name == "Items" {
			itemsField = &order.Fields[i]
			break
		}
	}
	require.NotNil(t, itemsField)

	// Items should be a slice
	assert.Equal(t, TypeKindSlice, itemsField.Type.Kind)
	assert.NotNil(t, itemsField.Type.ElemType)
	assert.Equal(t, TypeKindStruct, itemsField.Type.ElemType.Kind)
}

func TestAnalyzer_PointerField(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store")
	require.NoError(t, err)

	customerID := TypeID{PkgPath: "caster-generator/store", Name: "Customer"}
	customer := graph.GetType(customerID)
	require.NotNil(t, customer)

	// Find Address field (which is *string)
	var addressField *FieldInfo
	for i := range customer.Fields {
		if customer.Fields[i].Name == "Address" {
			addressField = &customer.Fields[i]
			break
		}
	}
	require.NotNil(t, addressField)

	// Address should be a pointer
	assert.Equal(t, TypeKindPointer, addressField.Type.Kind)
	assert.NotNil(t, addressField.Type.ElemType)
	assert.Equal(t, TypeKindBasic, addressField.Type.ElemType.Kind)
}

func TestAnalyzer_TypeAlias(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store")
	require.NoError(t, err)

	statusID := TypeID{PkgPath: "caster-generator/store", Name: "OrderStatus"}
	status := graph.GetType(statusID)
	require.NotNil(t, status)

	// OrderStatus should be an alias for string
	assert.Equal(t, TypeKindAlias, status.Kind)
}

func TestTypeID_String(t *testing.T) {
	id := TypeID{PkgPath: "caster-generator/store", Name: "Order"}
	assert.Equal(t, "caster-generator/store.Order", id.String())

	// Empty package path
	idNoPkg := TypeID{Name: "int"}
	assert.Equal(t, "int", idNoPkg.String())
}

func TestTypeKind_String(t *testing.T) {
	assert.Equal(t, "basic", TypeKindBasic.String())
	assert.Equal(t, "struct", TypeKindStruct.String())
	assert.Equal(t, "pointer", TypeKindPointer.String())
	assert.Equal(t, "slice", TypeKindSlice.String())
	assert.Equal(t, "alias", TypeKindAlias.String())
	assert.Equal(t, "external", TypeKindExternal.String())
	assert.Equal(t, "unknown", TypeKindUnknown.String())
}

func TestFieldInfo_JSONName(t *testing.T) {
	// Test with simple tag
	f1 := FieldInfo{Name: "MyField", Tag: `json:"my_field"`}
	assert.Equal(t, "my_field", f1.JSONName())

	// Test with options
	f2 := FieldInfo{Name: "MyField", Tag: `json:"my_field,omitempty"`}
	assert.Equal(t, "my_field", f2.JSONName())

	// Test with no tag
	f3 := FieldInfo{Name: "MyField", Tag: ""}
	assert.Equal(t, "MyField", f3.JSONName())

	// Test with "-" (ignored in JSON)
	f4 := FieldInfo{Name: "MyField", Tag: `json:"-"`}
	assert.Equal(t, "MyField", f4.JSONName())
}
