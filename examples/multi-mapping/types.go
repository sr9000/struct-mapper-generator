package multi

import "time"

// ExternalOrder from partner API.
type ExternalOrder struct {
	ExtOrderID  string          `json:"ext_order_id"`
	BuyerEmail  string          `json:"buyer_email"`
	BuyerName   string          `json:"buyer_name"`
	Items       []ExternalItem  `json:"items"`
	ShipTo      ExternalAddress `json:"ship_to"`
	OrderDate   string          `json:"order_date"` // ISO date string
	TotalAmount float64         `json:"total_amount"`
}

// ExternalItem is a line item from external system.
type ExternalItem struct {
	ItemCode  string  `json:"item_code"`
	ItemName  string  `json:"item_name"`
	Qty       int     `json:"qty"`
	UnitPrice float64 `json:"unit_price"`
}

// ExternalAddress for external orders.
type ExternalAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

// InternalOrder for internal processing.
type InternalOrder struct {
	ID              uint            `json:"id"`
	ExternalRef     string          `json:"external_ref"`
	CustomerEmail   string          `json:"customer_email"`
	CustomerName    string          `json:"customer_name"`
	LineItems       []InternalItem  `json:"line_items"`
	ShippingAddress InternalAddress `json:"shipping_address"`
	PlacedAt        time.Time       `json:"placed_at"`
	TotalCents      int64           `json:"total_cents"`
}

// InternalItem for internal processing.
type InternalItem struct {
	ID         uint   `json:"id"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	Quantity   int    `json:"quantity"`
	PriceCents int64  `json:"price_cents"`
}

// InternalAddress for internal use.
type InternalAddress struct {
	ID         uint   `json:"id"`
	Line1      string `json:"line1"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// WarehouseOrder for fulfillment system.
type WarehouseOrder struct {
	OrderNumber string           `json:"order_number"`
	Recipient   string           `json:"recipient"`
	Items       []WarehouseItem  `json:"items"`
	Address     WarehouseAddress `json:"address"`
	Priority    int              `json:"priority"`
}

// WarehouseItem for fulfillment.
type WarehouseItem struct {
	ProductCode string `json:"product_code"`
	Description string `json:"description"`
	Units       int    `json:"units"`
}

// WarehouseAddress for fulfillment.
type WarehouseAddress struct {
	FullAddress string `json:"full_address"`
	City        string `json:"city"`
	Region      string `json:"region"`
	ZIP         string `json:"zip"`
	CountryCode string `json:"country_code"`
}
