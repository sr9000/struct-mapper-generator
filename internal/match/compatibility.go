package match

import (
	"go/types"
)

// TypeCompatibility represents the level of compatibility between two types.
type TypeCompatibility int

const (
	// TypeIncompatible means the types cannot be converted.
	TypeIncompatible TypeCompatibility = iota
	// TypeNeedsTransform means conversion requires a custom transform function.
	TypeNeedsTransform
	// TypeConvertible means types are convertible using Go's type conversion.
	TypeConvertible
	// TypeAssignable means the source type can be directly assigned to the target.
	TypeAssignable
	// TypeIdentical means the types are exactly the same.
	TypeIdentical
)

const (
	VerdictIdentical      = "identical"
	VerdictAssignable     = "assignable"
	VerdictConvertible    = "convertible"
	VerdictNeedsTransform = "needs_transform"
	VerdictIncompatible   = "incompatible"
)

// String returns a human-readable name for the compatibility level.
func (c TypeCompatibility) String() string {
	switch c {
	case TypeIdentical:
		return VerdictIdentical
	case TypeAssignable:
		return VerdictAssignable
	case TypeConvertible:
		return VerdictConvertible
	case TypeNeedsTransform:
		return VerdictNeedsTransform
	case TypeIncompatible:
		return VerdictIncompatible
	default:
		return "unknown"
	}
}

// Score returns a numeric score for sorting (higher is better).
func (c TypeCompatibility) Score() int {
	return int(c)
}

// TypeCompatibilityResult contains detailed information about type compatibility.
type TypeCompatibilityResult struct {
	Compatibility TypeCompatibility
	Reason        string // Human-readable explanation
	SourceType    string // String representation of source type
	TargetType    string // String representation of target type
}

// ScoreTypeCompatibility determines the compatibility between a source and target type.
// Uses go/types for accurate type analysis.
func ScoreTypeCompatibility(source, target types.Type) TypeCompatibilityResult {
	sourceStr := source.String()
	targetStr := target.String()

	// Check for identical types
	if types.Identical(source, target) {
		return TypeCompatibilityResult{
			Compatibility: TypeIdentical,
			Reason:        "types are identical",
			SourceType:    sourceStr,
			TargetType:    targetStr,
		}
	}

	// Check for assignability (includes identical and interface satisfaction)
	if types.AssignableTo(source, target) {
		return TypeCompatibilityResult{
			Compatibility: TypeAssignable,
			Reason:        "source is assignable to target",
			SourceType:    sourceStr,
			TargetType:    targetStr,
		}
	}

	// Check for convertibility (numeric conversions, string/[]byte, etc.)
	if types.ConvertibleTo(source, target) {
		return TypeCompatibilityResult{
			Compatibility: TypeConvertible,
			Reason:        "source is convertible to target",
			SourceType:    sourceStr,
			TargetType:    targetStr,
		}
	}

	// Check for special cases that might need transforms
	if needsTransform(source, target) {
		return TypeCompatibilityResult{
			Compatibility: TypeNeedsTransform,
			Reason:        "types require a transform function",
			SourceType:    sourceStr,
			TargetType:    targetStr,
		}
	}

	return TypeCompatibilityResult{
		Compatibility: TypeIncompatible,
		Reason:        "types are not compatible",
		SourceType:    sourceStr,
		TargetType:    targetStr,
	}
}

// needsTransform checks for cases where types might be convertible via a transform.
func needsTransform(source, target types.Type) bool {
	// Unwrap named types to check underlying structure
	sourceUnderlying := source.Underlying()
	targetUnderlying := target.Underlying()

	// Pointer to non-pointer or vice versa (might be liftable)
	sourcePtr, sourceIsPtr := source.(*types.Pointer)
	targetPtr, targetIsPtr := target.(*types.Pointer)

	if sourceIsPtr && !targetIsPtr {
		// *T -> T (dereference possible if not nil)
		if types.Identical(sourcePtr.Elem(), target) ||
			types.AssignableTo(sourcePtr.Elem(), target) ||
			types.ConvertibleTo(sourcePtr.Elem(), target) {
			return true
		}
	}

	if !sourceIsPtr && targetIsPtr {
		// T -> *T (take address)
		if types.Identical(source, targetPtr.Elem()) ||
			types.AssignableTo(source, targetPtr.Elem()) ||
			types.ConvertibleTo(source, targetPtr.Elem()) {
			return true
		}
	}

	// Slice to slice with different element types
	sourceSlice, sourceIsSlice := sourceUnderlying.(*types.Slice)
	targetSlice, targetIsSlice := targetUnderlying.(*types.Slice)

	if sourceIsSlice && targetIsSlice {
		// Check if element types are compatible
		elemCompat := ScoreTypeCompatibility(sourceSlice.Elem(), targetSlice.Elem())
		if elemCompat.Compatibility >= TypeNeedsTransform {
			return true
		}
	}

	// Struct to struct (might have compatible fields)
	_, sourceIsStruct := sourceUnderlying.(*types.Struct)
	_, targetIsStruct := targetUnderlying.(*types.Struct)

	if sourceIsStruct && targetIsStruct {
		return true
	}

	return false
}

// ScorePointerCompatibility checks compatibility considering pointer wrapping/unwrapping.
func ScorePointerCompatibility(source, target types.Type) TypeCompatibilityResult {
	result := ScoreTypeCompatibility(source, target)
	if result.Compatibility >= TypeConvertible {
		return result
	}

	// Try unwrapping source pointer
	if ptr, ok := source.(*types.Pointer); ok {
		innerResult := ScoreTypeCompatibility(ptr.Elem(), target)
		if innerResult.Compatibility >= TypeConvertible {
			return TypeCompatibilityResult{
				Compatibility: TypeNeedsTransform,
				Reason:        "requires pointer dereference",
				SourceType:    source.String(),
				TargetType:    target.String(),
			}
		}
	}

	// Try wrapping source as pointer
	if ptr, ok := target.(*types.Pointer); ok {
		innerResult := ScoreTypeCompatibility(source, ptr.Elem())
		if innerResult.Compatibility >= TypeConvertible {
			return TypeCompatibilityResult{
				Compatibility: TypeNeedsTransform,
				Reason:        "requires taking address",
				SourceType:    source.String(),
				TargetType:    target.String(),
			}
		}
	}

	return result
}

// IsNumericType returns true if the type is a numeric basic type.
func IsNumericType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}

	info := basic.Info()

	return info&types.IsNumeric != 0
}

// IsStringType returns true if the type is a string.
func IsStringType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}

	return basic.Kind() == types.String
}
