package nested

import "time"

// APIOrder is an order from external API with nested structures.
type APIOrder struct {
	ID        string      `json:"id"`
	Customer  APICustomer `json:"customer"`
	Items     []APIItem   `json:"items"`
	Shipping  *APIAddress `json:"shipping,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	Status    string      `json:"status"`
}

// APICustomer represents customer info in API response.
type APICustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
}

// APIItem represents an order line item.
type APIItem struct {
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// APIAddress represents a shipping/billing address.
type APIAddress struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}
