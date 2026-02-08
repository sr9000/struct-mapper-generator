package match

import (
	"strings"
	"unicode"
)

// NormalizeIdent normalizes an identifier for fuzzy matching.
// The normalization pipeline:
// 1. Case-fold to lower.
// 2. Strip separators (_, -, spaces).
// 3. Tokenize CamelCase.
// 4. Optionally strip common suffix/prefix tokens.
func NormalizeIdent(s string) string {
	// First expand CamelCase before lowercasing
	tokens := tokenizeCamelCase(s)

	// Join, lowercase, and strip separators
	joined := strings.Join(tokens, "")
	joined = strings.ToLower(joined)
	joined = stripSeparators(joined)

	return joined
}

// NormalizeIdentWithSuffixStrip normalizes and strips common suffixes/prefixes.
// Common tokens to strip: id, ids, at, utc, timestamp.
// Note: We avoid stripping short suffixes like "ts" as they're too aggressive.
func NormalizeIdentWithSuffixStrip(s string) string {
	normalized := NormalizeIdent(s)

	// Strip common suffixes (ordered from longer to shorter to avoid partial matches)
	suffixes := []string{"timestamp", "ids", "utc", "id", "at"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(normalized, suffix) && len(normalized) > len(suffix) {
			normalized = strings.TrimSuffix(normalized, suffix)

			break
		}
	}

	return normalized
}

// tokenizeCamelCase splits a CamelCase or camelCase string into tokens.
// Examples:
//   - "OrderID" -> ["Order", "ID"]
//   - "customerName" -> ["customer", "Name"]
//   - "XMLParser" -> ["XML", "Parser"]
//   - "getHTTPResponse" -> ["get", "HTTP", "Response"]
func tokenizeCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	var tokens []string

	var current strings.Builder

	runes := []rune(s)
	for i := range runes {
		r := runes[i]

		// Handle separators - start a new token
		if isSeparator(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

			continue
		}

		if i == 0 {
			current.WriteRune(r)

			continue
		}

		if shouldStartNewToken(runes, i) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// isSeparator returns true if the rune is a common separator.
func isSeparator(r rune) bool {
	return r == '_' || r == '-' || r == ' '
}

// shouldStartNewToken determines if a new token should start at position i.
func shouldStartNewToken(runes []rune, i int) bool {
	r := runes[i]
	prevRune := runes[i-1]
	isUpper := unicode.IsUpper(r)
	isPrevUpper := unicode.IsUpper(prevRune)
	isPrevSep := isSeparator(prevRune)

	// Transition from lowercase to uppercase: start new token
	// e.g., "orderID" -> split before 'I'
	if isUpper && !isPrevUpper && !isPrevSep {
		return true
	}

	// End of acronym: check if next character is lowercase
	// e.g., "XMLParser" -> "XML" + "Parser", split before 'P'
	hasNextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
	if isUpper && isPrevUpper && hasNextLower {
		return true
	}

	return false
}

// stripSeparators removes common separators from a string.
func stripSeparators(s string) string {
	var result strings.Builder

	result.Grow(len(s))

	for _, r := range s {
		if !isSeparator(r) {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// TokenizeIdent splits an identifier into normalized lowercase tokens.
// Useful for more advanced matching algorithms.
func TokenizeIdent(s string) []string {
	tokens := tokenizeCamelCase(s)
	for i, t := range tokens {
		tokens[i] = strings.ToLower(t)
	}

	return tokens
}
