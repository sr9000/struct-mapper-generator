package pointers

type DomainLineItem struct {
	SKU   string
	Price int
}

type DomainOrder struct {
	ID string

	// LineItemPrice maps from APIOrder.LineItem.Price (*int -> int).
	LineItemPrice int

	LineItem DomainLineItem
	Items    []DomainLineItem
}
