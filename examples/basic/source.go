package basic

import "time"

// UserDTO represents a user from an external API.
type UserDTO struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

// OrderDTO represents an order from an external system.
type OrderDTO struct {
	OrderID    string  `json:"order_id"`
	CustomerID int64   `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
}
