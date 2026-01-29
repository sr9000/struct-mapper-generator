package match

import (
	"testing"
)

func TestNormalizeIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic cases
		{"OrderID", "orderid"},
		{"order_id", "orderid"},
		{"order-id", "orderid"},
		{"orderId", "orderid"},
		{"ORDERID", "orderid"},

		// CamelCase variations
		{"customerName", "customername"},
		{"CustomerName", "customername"},
		{"XMLParser", "xmlparser"},
		{"getHTTPResponse", "gethttpresponse"},
		{"totalCents", "totalcents"},
		{"TotalCents", "totalcents"},

		// With underscores
		{"price_cents", "pricecents"},
		{"PRICE_CENTS", "pricecents"},
		{"Price_Cents", "pricecents"},

		// Edge cases
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"ID", "id"},
		{"id", "id"},

		// Mixed separators
		{"order_item-ID", "orderitemid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeIdent(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeIdent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeIdentWithSuffixStrip(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Stripping ID suffix
		{"OrderID", "order"},
		{"CustomerID", "customer"},
		{"productId", "product"},

		// Stripping IDs suffix (plural)
		{"OrderIDs", "order"},
		{"customerIds", "customer"},

		// Stripping At suffix
		{"CreatedAt", "created"},
		{"UpdatedAt", "updated"},
		{"orderedAt", "ordered"},

		// Stripping UTC suffix
		{"CreatedUTC", "created"},
		{"timestampUTC", "timestamp"},

		// Stripping timestamp suffix
		{"CreatedTimestamp", "created"},

		// Should not strip if result would be empty
		{"ID", "id"},
		{"At", "at"},

		// No suffix to strip
		{"CustomerName", "customername"},
		{"totalCents", "totalcents"},
		{"PriceCents", "pricecents"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeIdentWithSuffixStrip(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeIdentWithSuffixStrip(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenizeCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"OrderID", []string{"Order", "ID"}},
		{"customerName", []string{"customer", "Name"}},
		{"XMLParser", []string{"XML", "Parser"}},
		{"getHTTPResponse", []string{"get", "HTTP", "Response"}},
		{"order_id", []string{"order", "id"}},
		{"ALLCAPS", []string{"ALLCAPS"}},
		{"lowercase", []string{"lowercase"}},
		{"", nil},
		{"a", []string{"a"}},
		{"AB", []string{"AB"}},
		{"AbC", []string{"Ab", "C"}},
		{"ABcD", []string{"A", "Bc", "D"}},
		{"URLParser", []string{"URL", "Parser"}},
		{"parseURL", []string{"parse", "URL"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := tokenizeCamelCase(tt.input)
			if !stringSliceEqual(result, tt.expected) {
				t.Errorf("tokenizeCamelCase(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenizeIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"OrderID", []string{"order", "id"}},
		{"customerName", []string{"customer", "name"}},
		{"XMLParser", []string{"xml", "parser"}},
		{"order_id", []string{"order", "id"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TokenizeIdent(tt.input)
			if !stringSliceEqual(result, tt.expected) {
				t.Errorf("TokenizeIdent(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
