package basic

import "time"

// User represents a user in the internal domain model.
type User struct {
	UserID    uint      `gorm:"primaryKey"  json:"user_id"`
	Email     string    `gorm:"uniqueIndex" json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

// Order represents an order in the internal domain model.
type Order struct {
	ID         uint   `gorm:"primaryKey"  json:"id"`
	CustomerID uint   `json:"customer_id"`
	TotalCents int64  `json:"total_cents"`
	Status     string `json:"status"`
}
