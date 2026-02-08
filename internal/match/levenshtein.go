package match

// Levenshtein computes the Levenshtein distance (edit distance) between two strings.
// The distance is the minimum number of single-character edits (insertions, deletions,
// or substitutions) required to transform one string into the other.
//
// Time complexity: O(len(a) * len(b))
// Space complexity: O(min(len(a), len(b))).
func Levenshtein(a, b string) int {
	if a == b {
		return 0
	}

	if len(a) == 0 {
		return len(b)
	}

	if len(b) == 0 {
		return len(a)
	}

	// Ensure a is the shorter string for space optimization
	if len(a) > len(b) {
		a, b = b, a
	}

	// Use two rows instead of full matrix for space optimization
	prev := make([]int, len(a)+1)
	curr := make([]int, len(a)+1)

	// Initialize first row
	for i := range prev {
		prev[i] = i
	}

	// Fill in the rest of the matrix
	for j := 1; j <= len(b); j++ {
		curr[0] = j

		for i := 1; i <= len(a); i++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			curr[i] = min3(
				prev[i]+1,      // deletion
				curr[i-1]+1,    // insertion
				prev[i-1]+cost, // substitution
			)
		}

		prev, curr = curr, prev
	}

	return prev[len(a)]
}

// LevenshteinNormalized computes a normalized similarity score between 0 and 1.
// 1.0 means identical strings, 0.0 means completely different.
// The score is: 1 - (distance / max(len(a), len(b))).
func LevenshteinNormalized(a, b string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}

	maxLen := max(len(b), len(a))

	distance := Levenshtein(a, b)

	return 1.0 - float64(distance)/float64(maxLen)
}

// NormalizedLevenshteinScore computes the similarity score between two identifiers
// after normalizing them. This is the primary function for name matching.
func NormalizedLevenshteinScore(a, b string) float64 {
	normA := NormalizeIdent(a)
	normB := NormalizeIdent(b)

	return LevenshteinNormalized(normA, normB)
}

// NormalizedLevenshteinScoreWithSuffixStrip computes the similarity score
// with additional suffix stripping for common patterns.
func NormalizedLevenshteinScoreWithSuffixStrip(a, b string) float64 {
	normA := NormalizeIdentWithSuffixStrip(a)
	normB := NormalizeIdentWithSuffixStrip(b)

	return LevenshteinNormalized(normA, normB)
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}

		return c
	}

	if b < c {
		return b
	}

	return c
}
