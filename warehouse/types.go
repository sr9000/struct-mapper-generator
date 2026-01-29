package warehouse

import (
	"time"
)

// Address represents a physical or billing/shipping address
type Address struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Street     string    `json:"street"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	PostalCode string    `json:"postal_code"`
	Country    string    `json:"country"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Customer represents a store customer/user
type Customer struct {
	ID                       uint       `json:"id" gorm:"primaryKey"`
	FirstName                string     `json:"first_name"`
	LastName                 string     `json:"last_name"`
	Email                    string     `json:"email" gorm:"uniqueIndex"`
	Phone                    string     `json:"phone"`
	PasswordHash             string     `json:"-"` // omitted in JSON
	DateOfBirth              *time.Time `json:"date_of_birth,omitempty"`
	DefaultBillingAddressID  *uint      `json:"default_billing_address_id,omitempty"`
	DefaultShippingAddressID *uint      `json:"default_shipping_address_id,omitempty"`

	// Relationships
	Addresses []Address `json:"addresses,omitempty" gorm:"foreignKey:CustomerID"`
	Orders    []Order   `json:"orders,omitempty" gorm:"foreignKey:CustomerID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Product represents a sellable item in the store
type Product struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	SKU         string  `json:"sku" gorm:"uniqueIndex"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       int64   `json:"price"` // in cents (minor currency unit)
	Stock       int     `json:"stock"`
	IsActive    bool    `json:"is_active"`
	Weight      float64 `json:"weight"` // in grams, useful for shipping

	// Relationships
	OrderItems []OrderItem `json:"-" gorm:"foreignKey:ProductID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Order represents a customer's purchase
type Order struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	CustomerID  uint   `json:"customer_id"`
	OrderNumber string `json:"order_number" gorm:"uniqueIndex"`
	Status      string `json:"status"`       // e.g. "pending", "paid", "shipped", "cancelled"
	TotalAmount int64  `json:"total_amount"` // in cents
	Currency    string `json:"currency" gorm:"default:'USD'"`

	ShippingAddressID uint `json:"shipping_address_id"`
	BillingAddressID  uint `json:"billing_address_id"`

	// Embedded addresses for snapshot (common denormalization practice)
	ShippingAddress Address `json:"shipping_address" gorm:"embedded;embeddedPrefix:shipping_"`
	BillingAddress  Address `json:"billing_address" gorm:"embedded;embeddedPrefix:billing_"`

	// Relationships
	Customer Customer    `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	Items    []OrderItem `json:"items" gorm:"foreignKey:OrderID"`

	PlacedAt    *time.Time `json:"placed_at,omitempty"`
	ShippedAt   *time.Time `json:"shipped_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// OrderItem is a line item within an order
type OrderItem struct {
	ID         uint  `json:"id" gorm:"primaryKey"`
	OrderID    uint  `json:"order_id"`
	ProductID  uint  `json:"product_id"`
	Quantity   int   `json:"quantity"`
	UnitPrice  int64 `json:"unit_price"`  // price at time of purchase (in cents)
	TotalPrice int64 `json:"total_price"` // UnitPrice * Quantity

	// Relationships
	Order   Order   `json:"-" gorm:"foreignKey:OrderID"`
	Product Product `json:"product" gorm:"foreignKey:ProductID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
