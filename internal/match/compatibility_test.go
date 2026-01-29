package match

import (
	"go/types"
	"testing"
)

func TestTypeCompatibility_String(t *testing.T) {
	tests := []struct {
		compat   TypeCompatibility
		expected string
	}{
		{TypeIdentical, "identical"},
		{TypeAssignable, "assignable"},
		{TypeConvertible, "convertible"},
		{TypeNeedsTransform, "needs_transform"},
		{TypeIncompatible, "incompatible"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.compat.String(); got != tt.expected {
				t.Errorf("TypeCompatibility.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeCompatibility_Score(t *testing.T) {
	// Verify ordering
	if TypeIncompatible.Score() >= TypeNeedsTransform.Score() {
		t.Error("TypeIncompatible should have lower score than TypeNeedsTransform")
	}
	if TypeNeedsTransform.Score() >= TypeConvertible.Score() {
		t.Error("TypeNeedsTransform should have lower score than TypeConvertible")
	}
	if TypeConvertible.Score() >= TypeAssignable.Score() {
		t.Error("TypeConvertible should have lower score than TypeAssignable")
	}
	if TypeAssignable.Score() >= TypeIdentical.Score() {
		t.Error("TypeAssignable should have lower score than TypeIdentical")
	}
}

func TestScoreTypeCompatibility(t *testing.T) {
	// Create basic types for testing
	intType := types.Typ[types.Int]
	int64Type := types.Typ[types.Int64]
	stringType := types.Typ[types.String]
	float64Type := types.Typ[types.Float64]

	tests := []struct {
		name     string
		source   types.Type
		target   types.Type
		expected TypeCompatibility
	}{
		{
			name:     "identical int",
			source:   intType,
			target:   intType,
			expected: TypeIdentical,
		},
		{
			name:     "identical string",
			source:   stringType,
			target:   stringType,
			expected: TypeIdentical,
		},
		{
			name:     "int to int64 convertible",
			source:   intType,
			target:   int64Type,
			expected: TypeConvertible,
		},
		{
			name:     "int64 to int convertible",
			source:   int64Type,
			target:   intType,
			expected: TypeConvertible,
		},
		{
			name:     "float64 to int convertible",
			source:   float64Type,
			target:   intType,
			expected: TypeConvertible,
		},
		{
			name:     "int to string convertible",
			source:   intType,
			target:   stringType,
			expected: TypeConvertible, // Go allows int to string conversion (rune)
		},
		{
			name:     "string to int incompatible",
			source:   stringType,
			target:   intType,
			expected: TypeIncompatible,
		},
		{
			name:     "bool to string incompatible",
			source:   types.Typ[types.Bool],
			target:   stringType,
			expected: TypeIncompatible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScoreTypeCompatibility(tt.source, tt.target)
			if result.Compatibility != tt.expected {
				t.Errorf("ScoreTypeCompatibility() = %v, want %v (reason: %s)",
					result.Compatibility, tt.expected, result.Reason)
			}
		})
	}
}

func TestScoreTypeCompatibility_Pointers(t *testing.T) {
	intType := types.Typ[types.Int]
	ptrIntType := types.NewPointer(intType)
	ptrPtrIntType := types.NewPointer(ptrIntType)

	tests := []struct {
		name     string
		source   types.Type
		target   types.Type
		expected TypeCompatibility
	}{
		{
			name:     "identical *int",
			source:   ptrIntType,
			target:   ptrIntType,
			expected: TypeIdentical,
		},
		{
			name:     "*int to int needs transform",
			source:   ptrIntType,
			target:   intType,
			expected: TypeNeedsTransform,
		},
		{
			name:     "int to *int needs transform",
			source:   intType,
			target:   ptrIntType,
			expected: TypeNeedsTransform,
		},
		{
			name:     "**int to *int incompatible",
			source:   ptrPtrIntType,
			target:   ptrIntType,
			expected: TypeIncompatible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScoreTypeCompatibility(tt.source, tt.target)
			if result.Compatibility != tt.expected {
				t.Errorf("ScoreTypeCompatibility() = %v, want %v (reason: %s)",
					result.Compatibility, tt.expected, result.Reason)
			}
		})
	}
}

func TestScoreTypeCompatibility_Slices(t *testing.T) {
	intType := types.Typ[types.Int]
	int64Type := types.Typ[types.Int64]
	stringType := types.Typ[types.String]

	sliceInt := types.NewSlice(intType)
	sliceInt64 := types.NewSlice(int64Type)
	sliceString := types.NewSlice(stringType)

	tests := []struct {
		name     string
		source   types.Type
		target   types.Type
		expected TypeCompatibility
	}{
		{
			name:     "identical []int",
			source:   sliceInt,
			target:   sliceInt,
			expected: TypeIdentical,
		},
		{
			name:     "[]int to []int64 needs transform",
			source:   sliceInt,
			target:   sliceInt64,
			expected: TypeNeedsTransform,
		},
		{
			name:     "[]int to []string needs transform",
			source:   sliceInt,
			target:   sliceString,
			expected: TypeNeedsTransform, // Element-wise conversion possible
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScoreTypeCompatibility(tt.source, tt.target)
			if result.Compatibility != tt.expected {
				t.Errorf("ScoreTypeCompatibility() = %v, want %v (reason: %s)",
					result.Compatibility, tt.expected, result.Reason)
			}
		})
	}
}

func TestScorePointerCompatibility(t *testing.T) {
	intType := types.Typ[types.Int]
	ptrIntType := types.NewPointer(intType)

	tests := []struct {
		name     string
		source   types.Type
		target   types.Type
		expected TypeCompatibility
	}{
		{
			name:     "*int to int needs transform (deref)",
			source:   ptrIntType,
			target:   intType,
			expected: TypeNeedsTransform,
		},
		{
			name:     "int to *int needs transform (addr)",
			source:   intType,
			target:   ptrIntType,
			expected: TypeNeedsTransform,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScorePointerCompatibility(tt.source, tt.target)
			if result.Compatibility != tt.expected {
				t.Errorf("ScorePointerCompatibility() = %v, want %v (reason: %s)",
					result.Compatibility, tt.expected, result.Reason)
			}
		})
	}
}

func TestIsNumericType(t *testing.T) {
	tests := []struct {
		name     string
		typ      types.Type
		expected bool
	}{
		{"int", types.Typ[types.Int], true},
		{"int64", types.Typ[types.Int64], true},
		{"float64", types.Typ[types.Float64], true},
		{"string", types.Typ[types.String], false},
		{"bool", types.Typ[types.Bool], false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNumericType(tt.typ); got != tt.expected {
				t.Errorf("IsNumericType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsStringType(t *testing.T) {
	tests := []struct {
		name     string
		typ      types.Type
		expected bool
	}{
		{"string", types.Typ[types.String], true},
		{"int", types.Typ[types.Int], false},
		{"bool", types.Typ[types.Bool], false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStringType(tt.typ); got != tt.expected {
				t.Errorf("IsStringType() = %v, want %v", got, tt.expected)
			}
		})
	}
}
