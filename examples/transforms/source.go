package transforms

// LegacyProduct is a product from a legacy system.
type LegacyProduct struct {
	Code         string  `json:"code"`
	Title        string  `json:"title"`
	PriceUSD     float64 `json:"price_usd"`     // dollars as float
	Weight       string  `json:"weight"`        // e.g., "2.5kg"
	Available    string  `json:"available"`     // "Y" or "N"
	LastModified string  `json:"last_modified"` // "2006-01-02 15:04:05"
	Categories   string  `json:"categories"`    // comma-separated: "electronics,gadgets"
}

// LegacyUser represents a user from legacy DB.
type LegacyUser struct {
	UserID      int    `json:"user_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PhoneNumber string `json:"phone_number"` // e.g., "+1-555-123-4567"
	BirthDate   string `json:"birth_date"`   // "1990-05-15"
	Address     string `json:"address"`      // JSON string: {"city":"NYC","zip":"10001"}
}
