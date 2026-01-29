package match

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		// Identical strings
		{"", "", 0},
		{"a", "a", 0},
		{"hello", "hello", 0},

		// Empty vs non-empty
		{"", "abc", 3},
		{"abc", "", 3},

		// Single character operations
		{"a", "b", 1},    // substitution
		{"a", "ab", 1},   // insertion
		{"ab", "a", 1},   // deletion
		{"abc", "ab", 1}, // deletion
		{"ab", "abc", 1}, // insertion

		// Multiple operations
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"algorithm", "altruistic", 6},

		// Case-sensitive
		{"ABC", "abc", 3},
		{"Hello", "hello", 1},

		// Real-world field name examples
		{"orderid", "orderid", 0},
		{"customerid", "customerID", 2}, // case difference
		{"pricecents", "totalcents", 5},
		{"createdat", "updatedat", 3}, // c->u, r->p, e->d
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := Levenshtein(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}

			// Verify symmetry
			resultReverse := Levenshtein(tt.b, tt.a)
			if result != resultReverse {
				t.Errorf("Levenshtein symmetry failed: (%q, %q) = %d, (%q, %q) = %d",
					tt.a, tt.b, result, tt.b, tt.a, resultReverse)
			}
		})
	}
}

func TestLevenshteinNormalized(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected float64
	}{
		// Identical strings
		{"", "", 1.0},
		{"hello", "hello", 1.0},

		// Completely different
		{"abc", "xyz", 0.0},

		// Partial matches
		{"kitten", "sitting", 1.0 - 3.0/7.0}, // ~0.571
		{"abc", "ab", 1.0 - 1.0/3.0},         // ~0.667
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := LevenshteinNormalized(tt.a, tt.b)
			// Allow small floating point tolerance
			if diff := result - tt.expected; diff < -0.001 || diff > 0.001 {
				t.Errorf("LevenshteinNormalized(%q, %q) = %f, want %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestNormalizedLevenshteinScore(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		minScore float64 // minimum expected score
	}{
		// Exact match after normalization
		{"OrderID", "orderid", 1.0},
		{"order_id", "orderID", 1.0},
		{"CustomerName", "customer_name", 1.0},

		// Similar names
		{"TotalCents", "TotalAmount", 0.5}, // Some similarity
		{"CreatedAt", "UpdatedAt", 0.5},    // Similar suffix

		// Different names
		{"Email", "Password", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := NormalizedLevenshteinScore(tt.a, tt.b)
			if result < tt.minScore {
				t.Errorf("NormalizedLevenshteinScore(%q, %q) = %f, want >= %f",
					tt.a, tt.b, result, tt.minScore)
			}
		})
	}
}

func TestNormalizedLevenshteinScoreWithSuffixStrip(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected float64
	}{
		// ID suffix stripping should make these match
		{"CustomerID", "Customer", 1.0},
		{"OrderID", "Order", 1.0},

		// At suffix stripping
		{"CreatedAt", "Created", 1.0},
		{"UpdatedAt", "Updated", 1.0},

		// Combined normalization + suffix strip
		{"customer_id", "CustomerID", 1.0},
		{"created_at", "CreatedAt", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := NormalizedLevenshteinScoreWithSuffixStrip(tt.a, tt.b)
			if diff := result - tt.expected; diff < -0.001 || diff > 0.001 {
				t.Errorf("NormalizedLevenshteinScoreWithSuffixStrip(%q, %q) = %f, want %f",
					tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkLevenshtein(b *testing.B) {
	a := "algorithm"
	bStr := "altruistic"
	for i := 0; i < b.N; i++ {
		Levenshtein(a, bStr)
	}
}

func BenchmarkNormalizedLevenshteinScore(b *testing.B) {
	a := "CustomerOrderID"
	bStr := "customer_order_id"
	for i := 0; i < b.N; i++ {
		NormalizedLevenshteinScore(a, bStr)
	}
}
