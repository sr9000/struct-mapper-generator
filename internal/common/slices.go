package common

// IsEmpty returns true if the slice is empty.
func IsEmpty[S ~[]E, E any](s S) bool {
	return len(s) == 0
}

// IsSingle returns true if the slice has exactly one element.
func IsSingle[S ~[]E, E any](s S) bool {
	return len(s) == 1
}

// IsMultiple returns true if the slice has more than one element.
func IsMultiple[S ~[]E, E any](s S) bool {
	return len(s) > 1
}

// First returns the first element of the slice and true, or the zero value and false if empty.
func First[S ~[]E, E any](s S) (E, bool) {
	if len(s) == 0 {
		var zero E
		return zero, false
	}

	return s[0], true
}
