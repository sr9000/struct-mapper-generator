package nested_struct

import "time"

// DomainOrder is the internal order representation.
type DomainOrder struct {
	OrderID      uint             `gorm:"primaryKey"              json:"order_id"`
	Customer     DomainCustomer   `gorm:"embedded"                json:"customer"`
	LineItems    []DomainLineItem `gorm:"foreignKey:OrderID"      json:"line_items"`
	ShippingAddr *DomainAddress   `json:"shipping_addr,omitempty"`
	OrderedAt    time.Time        `json:"ordered_at"`
	OrderStatus  string           `json:"order_status"`
}

// DomainCustomer is the internal customer representation.
type DomainCustomer struct {
	CustomerID   uint   `json:"customer_id"`
	EmailAddress string `json:"email_address"`
	FullName     string `json:"full_name"`
	PhoneNumber  string `json:"phone_number,omitempty"`
}

// DomainLineItem is an order line item in domain model.
type DomainLineItem struct {
	ID          uint   `gorm:"primaryKey"   json:"id"`
	OrderID     uint   `json:"order_id"`
	ProductSKU  string `json:"product_sku"`
	ProductName string `json:"product_name"`
	Qty         int    `json:"qty"`
	PriceCents  int64  `json:"price_cents"`
}

// DomainAddress is an address in the domain model.
type DomainAddress struct {
	ID         uint   `gorm:"primaryKey"        json:"id"`
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`
	CountryISO string `json:"country_iso"`
}
