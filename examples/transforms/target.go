package transforms

import "time"

// ModernProduct is the new product model.
type ModernProduct struct {
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	PriceCents  int64     `json:"price_cents"`  // price in cents
	WeightGrams int       `json:"weight_grams"` // weight in grams
	IsActive    bool      `json:"is_active"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags"` // categories as slice
}

// ModernUser is the new user model.
type ModernUser struct {
	ID          uint       `json:"id"`
	FullName    string     `json:"full_name"` // combined first + last
	Phone       string     `json:"phone"`     // normalized: "15551234567"
	DateOfBirth *time.Time `json:"date_of_birth"`
	City        string     `json:"city"`
	PostalCode  string     `json:"postal_code"`
}
