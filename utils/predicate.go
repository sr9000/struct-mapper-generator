package utils

type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// IsInRange checks if a value is within the specified range, both inclusive.
func IsInRange[T number](min T, value T, max T) bool {
	return min <= value && value <= max
}
