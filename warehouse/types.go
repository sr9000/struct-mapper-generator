package warehouse

import (
	"time"
)

// Address represents a physical or billing/shipping address.
type Address struct {
	ID         uint      `gorm:"primaryKey"  json:"id"`
	Street     string    `json:"street"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	PostalCode string    `json:"postal_code"`
	Country    string    `json:"country"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Customer represents a store customer/user.
type Customer struct {
	ID                       uint       `gorm:"primaryKey"                            json:"id"`
	FirstName                string     `json:"first_name"`
	LastName                 string     `json:"last_name"`
	Email                    string     `gorm:"uniqueIndex"                           json:"email"`
	Phone                    string     `json:"phone"`
	PasswordHash             string     `json:"-"` // omitted in JSON
	DateOfBirth              *time.Time `json:"date_of_birth,omitempty"`
	DefaultBillingAddressID  *uint      `json:"default_billing_address_id,omitempty"`
	DefaultShippingAddressID *uint      `json:"default_shipping_address_id,omitempty"`

	// Relationships
	Addresses []Address `gorm:"foreignKey:CustomerID" json:"addresses,omitempty"`
	Orders    []Order   `gorm:"foreignKey:CustomerID" json:"orders,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Product represents a sellable item in the store.
type Product struct {
	ID          uint    `gorm:"primaryKey"  json:"id"`
	SKU         string  `gorm:"uniqueIndex" json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       int64   `json:"price"` // in cents (minor currency unit)
	Stock       int     `json:"stock"`
	IsActive    bool    `json:"is_active"`
	Weight      float64 `json:"weight"` // in grams, useful for shipping

	// Relationships
	OrderItems []OrderItem `gorm:"foreignKey:ProductID" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Order represents a customer's purchase.
type Order struct {
	ID          uint   `gorm:"primaryKey"    json:"id"`
	CustomerID  uint   `json:"customer_id"`
	OrderNumber string `gorm:"uniqueIndex"   json:"order_number"`
	Status      string `json:"status"`       // e.g. "pending", "paid", "shipped", "cancelled"
	TotalAmount int64  `json:"total_amount"` // in cents
	Currency    string `gorm:"default:'USD'" json:"currency"`

	ShippingAddressID uint `json:"shipping_address_id"`
	BillingAddressID  uint `json:"billing_address_id"`

	// Embedded addresses for snapshot (common denormalization practice)
	ShippingAddress Address `gorm:"embedded;embeddedPrefix:shipping_" json:"shipping_address"`
	BillingAddress  Address `gorm:"embedded;embeddedPrefix:billing_"  json:"billing_address"`

	// Relationships
	Customer Customer    `gorm:"foreignKey:CustomerID" json:"customer"`
	Items    []OrderItem `gorm:"foreignKey:OrderID"    json:"items"`

	PlacedAt    *time.Time `json:"placed_at,omitempty"`
	ShippedAt   *time.Time `json:"shipped_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// OrderItem is a line item within an order.
type OrderItem struct {
	ID         uint  `gorm:"primaryKey"  json:"id"`
	OrderID    uint  `json:"order_id"`
	ProductID  uint  `json:"product_id"`
	Quantity   int   `json:"quantity"`
	UnitPrice  int64 `json:"unit_price"`  // price at time of purchase (in cents)
	TotalPrice int64 `json:"total_price"` // UnitPrice * Quantity

	// Relationships
	Order   Order   `gorm:"foreignKey:OrderID"   json:"-"`
	Product Product `gorm:"foreignKey:ProductID" json:"product"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
