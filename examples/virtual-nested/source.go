package nested

type Order struct {
	ID    string
	Items []OrderItem
}

type OrderItem struct {
	SKU string
	Qty int
}
