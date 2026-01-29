package analyze

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypePath(t *testing.T) {
	// Simple path
	p1 := NewTypePath("Order")
	assert.Equal(t, "Order", p1.String())

	// Field path
	p2 := p1.Field("Items")
	assert.Equal(t, "Order.Items", p2.String())

	// Slice path
	p3 := p2.Slice()
	assert.Equal(t, "Order.Items[]", p3.String())

	// Field in slice element
	p4 := p3.Field("ProductID")
	assert.Equal(t, "Order.Items[].ProductID", p4.String())

	// Pointer
	p5 := NewTypePath("Customer").Field("Address").Pointer()
	assert.Equal(t, "Customer.*Address", p5.String())
}

func TestTypeStringer_TypeString(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store")
	require.NoError(t, err)

	stringer := NewTypeStringer()

	// Test struct type
	orderID := TypeID{PkgPath: "caster-generator/store", Name: "Order"}
	order := graph.GetType(orderID)
	require.NotNil(t, order)
	assert.Equal(t, "Order", stringer.TypeString(order))

	// Test slice field
	var itemsField *FieldInfo
	for i := range order.Fields {
		if order.Fields[i].Name == "Items" {
			itemsField = &order.Fields[i]
			break
		}
	}
	require.NotNil(t, itemsField)
	assert.Equal(t, "[]OrderItem", stringer.TypeString(itemsField.Type))

	// Test alias type
	statusID := TypeID{PkgPath: "caster-generator/store", Name: "OrderStatus"}
	status := graph.GetType(statusID)
	require.NotNil(t, status)
	assert.Equal(t, "OrderStatus", stringer.TypeString(status))
}

func TestTypeStringer_FieldPath(t *testing.T) {
	stringer := NewTypeStringer()

	assert.Equal(t, "Order.ID", stringer.FieldPath("Order", "ID"))
	assert.Equal(t, "Order.Items.ProductID", stringer.FieldPath("Order", "Items", "ProductID"))
}

func TestTypeStringer_BuildFieldPaths(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/store")
	require.NoError(t, err)

	stringer := NewTypeStringer()

	orderID := TypeID{PkgPath: "caster-generator/store", Name: "Order"}
	order := graph.GetType(orderID)
	require.NotNil(t, order)

	paths := stringer.BuildFieldPaths(order, 2)

	// Check direct fields
	assert.Contains(t, paths, "Order.ID")
	assert.Contains(t, paths, "Order.CustomerID")
	assert.Contains(t, paths, "Order.Status")
	assert.Contains(t, paths, "Order.TotalCents")
	assert.Contains(t, paths, "Order.Items")
	assert.Contains(t, paths, "Order.OrderedAt")

	// Check nested fields (Items is a slice of OrderItem)
	assert.Contains(t, paths, "Order.Items[].ProductID")
	assert.Contains(t, paths, "Order.Items[].Name")
	assert.Contains(t, paths, "Order.Items[].Quantity")
	assert.Contains(t, paths, "Order.Items[].UnitPrice")
}

func TestTypeStringer_BuildFieldPathsWithPointer(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.LoadPackages("caster-generator/warehouse")
	require.NoError(t, err)

	stringer := NewTypeStringer()

	customerID := TypeID{PkgPath: "caster-generator/warehouse", Name: "Customer"}
	customer := graph.GetType(customerID)
	require.NotNil(t, customer)

	paths := stringer.BuildFieldPaths(customer, 2)

	// Check direct fields including pointer fields
	assert.Contains(t, paths, "Customer.ID")
	assert.Contains(t, paths, "Customer.Email")
	assert.Contains(t, paths, "Customer.DateOfBirth")
	assert.Contains(t, paths, "Customer.Addresses")
}

func TestTypeStringer_NilType(t *testing.T) {
	stringer := NewTypeStringer()
	assert.Equal(t, "<nil>", stringer.TypeString(nil))
}
