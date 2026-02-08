package pointers

type APILineItem struct {
	SKU   string
	Price *int
}

type APIOrder struct {
	ID       string
	LineItem *APILineItem
	Items    []*APILineItem
}
