package tutorial

import "time"

// =============================================================================
// Source Types (simulating external API models)
// =============================================================================

// APIProduct represents a product from an external API.
type APIProduct struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	PriceUSD    float64 `json:"price_usd"`
	InStock     int     `json:"in_stock"`
	Category    string  `json:"category"`
}

// APICustomer represents a customer from an external API.
type APICustomer struct {
	CustomerID string `json:"customer_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Active     string `json:"active"` // "Y" or "N"
}

// APIOrder represents an order from an external API.
type APIOrder struct {
	OrderID     string         `json:"order_id"`
	CustomerID  string         `json:"customer_id"`
	Customer    *APICustomer   `json:"customer"`
	Items       []APIOrderItem `json:"items"`
	TotalUSD    float64        `json:"total_usd"`
	OrderDate   string         `json:"order_date"` // ISO date string
	Status      string         `json:"status"`
	ShippingQty *int           `json:"shipping_qty"`
}

// APIOrderItem represents an order line item from an external API.
type APIOrderItem struct {
	ItemID    string  `json:"item_id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// APIAddress represents an address from an external API.
type APIAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

// =============================================================================
// Target Types (simulating internal domain models)
// =============================================================================

// DomainProduct represents a product in the internal domain.
type DomainProduct struct {
	ID          uint   `gorm:"primaryKey"`
	SKU         string `gorm:"uniqueIndex"`
	Name        string
	PriceCents  int64
	Stock       int
	CategoryTag string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DomainCustomer represents a customer in the internal domain.
type DomainCustomer struct {
	ID        uint   `gorm:"primaryKey"`
	FullName  string // Combined from FirstName + LastName
	Email     string `gorm:"uniqueIndex"`
	Phone     string
	IsActive  bool // Converted from "Y"/"N"
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DomainOrder represents an order in the internal domain.
type DomainOrder struct {
	ID            uint             `gorm:"primaryKey"`
	ExternalRef   string           // Original OrderID
	CustomerID    uint             // Foreign key
	Customer      *DomainCustomer  // Nested struct
	LineItems     []DomainLineItem // Renamed from Items
	TotalCents    int64            // Converted from dollars
	PlacedAt      time.Time        // Parsed from string
	OrderStatus   string           // Renamed from Status
	ShippingCount int              // Dereferenced from pointer
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DomainLineItem represents an order line item in the internal domain.
type DomainLineItem struct {
	ID          uint `gorm:"primaryKey"`
	OrderID     uint // Foreign key - needs context passing!
	ProductSKU  string
	ProductName string
	Qty         int
	PriceCents  int64
}

// DomainAddress represents an address in the internal domain.
type DomainAddress struct {
	Street     string
	City       string
	PostalCode string // Renamed from ZipCode
	Country    string
}

// =============================================================================
// Recursive Types (for Step 13)
// =============================================================================

// TreeNode represents a tree node (source).
type TreeNode struct {
	Value    string
	Children []*TreeNode
}

// TreeNodeDTO represents a tree node (target).
type TreeNodeDTO struct {
	Data     string
	Children []*TreeNodeDTO
}

// =============================================================================
// Collection Types (for Step 11)
// =============================================================================

// APIPoint represents a 2D point (source).
type APIPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DomainPoint represents a 2D point (target).
type DomainPoint struct {
	X int
	Y int
}

// APIBox represents a box with 4 corners (source).
type APIBox struct {
	Name    string
	Corners [4]APIPoint
}

// DomainBox represents a box with 4 corners (target).
type DomainBox struct {
	Label   string
	Corners [4]DomainPoint
}

// APIInventory represents product inventory (source).
type APIInventory struct {
	Products map[string]APIProduct // map by product ID
}

// DomainInventory represents product inventory (target).
type DomainInventory struct {
	Products map[string]DomainProduct // map by SKU
}
