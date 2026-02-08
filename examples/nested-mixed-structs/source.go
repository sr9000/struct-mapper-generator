package nestedmixed

type APIOrder struct {
	ID    string
	Items []*APIItem
}

type APIItem struct {
	SKU      string
	Quantity int
	Note     *string
}
