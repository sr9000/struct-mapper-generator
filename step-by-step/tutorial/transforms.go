package tutorial

import (
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Transform Functions for Tutorial Examples
// =============================================================================

// StringToUint converts a string ID to uint.
func StringToUint(s string) uint {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(v)
}

// DollarsToCents converts a dollar amount (float64) to cents (int64).
func DollarsToCents(dollars float64) int64 {
	return int64(dollars * 100)
}

// Float64ToInt converts float64 to int (truncating decimals).
func Float64ToInt(f float64) int {
	return int(f)
}

// YNToBool converts "Y"/"N" string to boolean.
func YNToBool(yn string) bool {
	return strings.ToUpper(yn) == "Y"
}

// ParseISODate parses an ISO date string to time.Time.
func ParseISODate(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try simpler format
		t, err = time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// ConcatNames combines first and last name.
func ConcatNames(first, last string) string {
	return strings.TrimSpace(first + " " + last)
}

// IntPtrToInt dereferences an int pointer with a default of 0.
func IntPtrToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// PassOrderID is a pass-through transform for context passing.
// Used when requires parameter is passed to child caster.
func PassOrderID(orderID uint) uint {
	return orderID
}
