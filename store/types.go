package store

import (
	"time"
)

// 1. Product represents an individual item available for sale.
// We use int64 for Price to represent cents (lowest currency unit) to avoid floating-point errors.
type Product struct {
	ID          int64     `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	PriceCents  int64     `json:"price_cents"`
	Inventory   int       `json:"inventory_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// 2. Customer represents the user placing orders.
type Customer struct {
	ID       int64   `json:"id"`
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Address  *string `json:"address"` // In a complex app, this might be its own struct
	IsActive bool    `json:"is_active"`
}

// 3. Order represents a transaction made by a customer.
type Order struct {
	ID         int64       `json:"id"`
	CustomerID int64       `json:"customer_id"`
	Status     OrderStatus `json:"status"`
	TotalCents int64       `json:"total_cents"`
	Items      []OrderItem `json:"items"` // Has-Many relationship
	OrderedAt  time.Time   `json:"ordered_at"`
}

// 4. OrderItem represents a specific product line within an order.
// It snapshots the price at the time of purchase.
type OrderItem struct {
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"` // Redundant but useful for history if product name changes
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

// 5. OrderStatus is a custom type for type-safe status handling.
type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusPaid      OrderStatus = "PAID"
	StatusShipped   OrderStatus = "SHIPPED"
	StatusCancelled OrderStatus = "CANCELLED"
)
