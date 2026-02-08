package plan

import (
	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/match"
)

// Strategy explanation constants.
const (
	explSliceMap          = "slice map"
	explNestedStruct      = "nested struct"
	explPointerNestedCast = "pointer nested cast"
	explPointerDeref      = "pointer deref"
	explPointerWrap       = "pointer wrap"
	explMap               = "map copy"
)

// determineStrategy determines the conversion strategy based on source and target types.
func (r *Resolver) determineStrategy(
	sourcePath, targetPath mapping.FieldPath,
	sourceType, targetType *analyze.TypeInfo,
) (ConversionStrategy, string) {
	return r.determineStrategyWithHint(sourcePath, targetPath, sourceType, targetType, mapping.HintNone)
}

// determineStrategyWithHint determines the conversion strategy, respecting introspection hints.
func (r *Resolver) determineStrategyWithHint(
	sourcePath, targetPath mapping.FieldPath,
	sourceType, targetType *analyze.TypeInfo,
	hint mapping.IntrospectionHint,
) (ConversionStrategy, string) {
	// Get the actual field types
	sourceFieldType := r.resolveFieldType(sourcePath, sourceType)
	targetFieldType := r.resolveFieldType(targetPath, targetType)

	if sourceFieldType == nil || targetFieldType == nil {
		return StrategyTransform, "type info unavailable"
	}

	// If hint is "final", always use transform (no introspection)
	if hint == mapping.HintFinal {
		return StrategyTransform, "final (no introspection)"
	}

	// For generated types, we can't use Go type compatibility check
	// Instead, use structural matching based on Kind
	if sourceFieldType.IsGenerated || targetFieldType.IsGenerated ||
		sourceFieldType.GoType == nil || targetFieldType.GoType == nil {
		return r.determineStrategyByKind(sourceFieldType, targetFieldType, hint)
	}

	// Check type compatibility
	compat := match.ScorePointerCompatibility(sourceFieldType.GoType, targetFieldType.GoType)

	switch compat.Compatibility {
	case match.TypeIdentical:
		return StrategyDirectAssign, match.VerdictIdentical
	case match.TypeAssignable:
		return StrategyDirectAssign, match.VerdictAssignable
	case match.TypeConvertible:
		return StrategyConvert, match.VerdictConvertible
	case match.TypeNeedsTransform:
		return r.determineNeedsTransformStrategy(sourceFieldType, targetFieldType, hint)
	default:
		return r.determineIncompatibleStrategy(sourceFieldType, targetFieldType, hint)
	}
}

// determineStrategyByKind determines conversion strategy based on type kinds
// (used for generated types where GoType is not available).
func (r *Resolver) determineStrategyByKind(
	sourceFieldType, targetFieldType *analyze.TypeInfo,
	hint mapping.IntrospectionHint,
) (ConversionStrategy, string) {
	srcKind := sourceFieldType.Kind
	tgtKind := targetFieldType.Kind

	// Same kind - direct assign or compatible
	if srcKind == tgtKind {
		switch srcKind {
		default:
			return StrategyDirectAssign, "same kind"
		case analyze.TypeKindBasic:
			// For basic types with same name, direct assign
			if sourceFieldType.ID.Name == targetFieldType.ID.Name {
				return StrategyDirectAssign, "identical"
			}

			return StrategyConvert, "convertible"
		case analyze.TypeKindStruct:
			if hint == mapping.HintDive {
				return StrategyNestedCast, "nested struct (dive)"
			}

			return StrategyNestedCast, explNestedStruct
		case analyze.TypeKindSlice, analyze.TypeKindArray:
			if hint == mapping.HintDive {
				return StrategySliceMap, "slice map (dive)"
			}

			return StrategySliceMap, explSliceMap
		case analyze.TypeKindMap:
			if hint == mapping.HintDive {
				return StrategyMap, "map copy (dive)"
			}

			return StrategyMap, explMap
		case analyze.TypeKindPointer:
			// Check element types
			if sourceFieldType.ElemType != nil && targetFieldType.ElemType != nil {
				if sourceFieldType.ElemType.Kind == analyze.TypeKindStruct &&
					targetFieldType.ElemType.Kind == analyze.TypeKindStruct {
					return StrategyPointerNestedCast, explPointerNestedCast
				}
			}

			return StrategyDirectAssign, "pointer"
		}
	}

	// Different kinds - handle common cases
	if srcKind == analyze.TypeKindPointer && tgtKind != analyze.TypeKindPointer {
		return StrategyPointerDeref, explPointerDeref
	}

	if srcKind != analyze.TypeKindPointer && tgtKind == analyze.TypeKindPointer {
		return StrategyPointerWrap, explPointerWrap
	}

	return StrategyTransform, "incompatible kinds"
}

func (r *Resolver) determineNeedsTransformStrategy(
	sourceFieldType, targetFieldType *analyze.TypeInfo,
	hint mapping.IntrospectionHint,
) (ConversionStrategy, string) {
	// Determine more specific strategy
	if sourceFieldType.Kind == analyze.TypeKindPointer && targetFieldType.Kind != analyze.TypeKindPointer {
		return StrategyPointerDeref, explPointerDeref
	}

	if sourceFieldType.Kind != analyze.TypeKindPointer && targetFieldType.Kind == analyze.TypeKindPointer {
		return StrategyPointerWrap, explPointerWrap
	}

	// Check for pointer-to-pointer struct conversions (e.g., *Node -> *NodeDTO)
	if sourceFieldType.Kind == analyze.TypeKindPointer && targetFieldType.Kind == analyze.TypeKindPointer {
		srcElem := sourceFieldType.ElemType

		tgtElem := targetFieldType.ElemType
		if srcElem != nil && tgtElem != nil &&
			srcElem.Kind == analyze.TypeKindStruct && tgtElem.Kind == analyze.TypeKindStruct {
			return StrategyPointerNestedCast, explPointerNestedCast
		}
	}

	if sourceFieldType.Kind == analyze.TypeKindSlice && targetFieldType.Kind == analyze.TypeKindSlice {
		// For slices, check if hint says dive (introspect elements) or final
		if hint == mapping.HintDive {
			return StrategySliceMap, explSliceMap + " (dive)"
		}

		return StrategySliceMap, explSliceMap
	}

	if sourceFieldType.Kind == analyze.TypeKindArray && targetFieldType.Kind == analyze.TypeKindArray {
		// Arrays are treated like slices for mapping purposes (element-wise).
		if hint == mapping.HintDive {
			return StrategySliceMap, explSliceMap + " (dive, array)"
		}

		return StrategySliceMap, explSliceMap + " (array)"
	}

	if sourceFieldType.Kind == analyze.TypeKindStruct && targetFieldType.Kind == analyze.TypeKindStruct {
		// For structs, check if hint says dive (recursively map fields) or final
		if hint == mapping.HintDive {
			return StrategyNestedCast, explNestedStruct + " (dive)"
		}
		// Default behavior: introspect structs unless marked final
		return StrategyNestedCast, explNestedStruct
	}

	return StrategyTransform, "needs transform"
}

func (r *Resolver) determineIncompatibleStrategy(
	sourceFieldType, targetFieldType *analyze.TypeInfo,
	hint mapping.IntrospectionHint,
) (ConversionStrategy, string) {
	// Check for pointer-to-pointer struct conversions (e.g., *Node -> *NodeDTO)
	if sourceFieldType.Kind == analyze.TypeKindPointer && targetFieldType.Kind == analyze.TypeKindPointer {
		srcElem := sourceFieldType.ElemType

		tgtElem := targetFieldType.ElemType
		if srcElem != nil && tgtElem != nil &&
			srcElem.Kind == analyze.TypeKindStruct && tgtElem.Kind == analyze.TypeKindStruct {
			return StrategyPointerNestedCast, explPointerNestedCast
		}
	}

	// Also check for struct/slice even when marked as incompatible
	if sourceFieldType.Kind == analyze.TypeKindStruct && targetFieldType.Kind == analyze.TypeKindStruct {
		if hint == mapping.HintDive {
			return StrategyNestedCast, explNestedStruct + " (dive)"
		}

		return StrategyNestedCast, explNestedStruct
	}

	if sourceFieldType.Kind == analyze.TypeKindSlice && targetFieldType.Kind == analyze.TypeKindSlice {
		if hint == mapping.HintDive {
			return StrategySliceMap, explSliceMap + " (dive)"
		}

		return StrategySliceMap, explSliceMap
	}

	if sourceFieldType.Kind == analyze.TypeKindArray && targetFieldType.Kind == analyze.TypeKindArray {
		if hint == mapping.HintDive {
			return StrategySliceMap, explSliceMap + " (dive, array)"
		}

		return StrategySliceMap, explSliceMap + " (array)"
	}

	return StrategyTransform, "incompatible"
}

// determineStrategyFromCandidate determines the conversion strategy from a candidate match.
func (r *Resolver) determineStrategyFromCandidate(cand *match.Candidate) (ConversionStrategy, string) {
	switch cand.TypeCompat.Compatibility {
	case match.TypeIdentical:
		return StrategyDirectAssign, match.TypeIdentical.String()
	case match.TypeAssignable:
		return StrategyDirectAssign, match.TypeAssignable.String()
	case match.TypeConvertible:
		return StrategyConvert, match.TypeConvertible.String()
	case match.TypeNeedsTransform:
		// Check for specific strategies based on reason
		if cand.TypeCompat.Reason == "requires pointer dereference" {
			return StrategyPointerDeref, explPointerDeref
		}

		if cand.TypeCompat.Reason == "requires taking address" {
			return StrategyPointerWrap, explPointerWrap
		}

		// Check if source and target are both structs (nested struct conversion)
		if cand.SourceField.Type != nil && cand.TargetField.Type != nil {
			srcKind := cand.SourceField.Type.Kind
			tgtKind := cand.TargetField.Type.Kind

			// Handle struct-to-struct
			if srcKind == analyze.TypeKindStruct && tgtKind == analyze.TypeKindStruct {
				return StrategyNestedCast, "nested struct"
			}

			// Handle slice-to-slice
			if srcKind == analyze.TypeKindSlice && tgtKind == analyze.TypeKindSlice {
				return StrategySliceMap, "slice map"
			}

			// Handle array-to-array
			if srcKind == analyze.TypeKindArray && tgtKind == analyze.TypeKindArray {
				return StrategySliceMap, "array map"
			}

			// Handle map-to-map
			if srcKind == analyze.TypeKindMap && tgtKind == analyze.TypeKindMap {
				return StrategyMap, "map copy"
			}
		}

		return StrategyTransform, cand.TypeCompat.Reason
	default:
		// Also check for struct/slice even when marked as incompatible
		// (types might be different named structs which aren't directly compatible)
		if cand.SourceField.Type != nil && cand.TargetField.Type != nil {
			srcKind := cand.SourceField.Type.Kind
			tgtKind := cand.TargetField.Type.Kind

			if srcKind == analyze.TypeKindStruct && tgtKind == analyze.TypeKindStruct {
				return StrategyNestedCast, "nested struct"
			}

			if srcKind == analyze.TypeKindSlice && tgtKind == analyze.TypeKindSlice {
				return StrategySliceMap, "slice map"
			}

			if srcKind == analyze.TypeKindArray && tgtKind == analyze.TypeKindArray {
				return StrategySliceMap, "array map"
			}

			if srcKind == analyze.TypeKindMap && tgtKind == analyze.TypeKindMap {
				return StrategyMap, "map copy"
			}
		}

		return StrategyTransform, "incompatible"
	}
}

// resolveFieldType resolves the TypeInfo for a field at the given path.
func (r *Resolver) resolveFieldType(path mapping.FieldPath, typeInfo *analyze.TypeInfo) *analyze.TypeInfo {
	current := typeInfo

	for i, seg := range path.Segments {
		if current.Kind != analyze.TypeKindStruct {
			return nil
		}

		var found *analyze.FieldInfo

		for j := range current.Fields {
			if current.Fields[j].Name == seg.Name {
				found = &current.Fields[j]
				break
			}
		}

		if found == nil {
			return nil
		}

		current = found.Type

		if seg.IsSlice && current.Kind == analyze.TypeKindSlice {
			current = current.ElemType
		}

		// Auto-deref pointers only when we're stepping *through* them to reach a deeper field.
		// For leaf fields, we must preserve pointer-ness so strategies like PointerDeref can be selected.
		isLast := i == len(path.Segments)-1
		if !isLast && current.Kind == analyze.TypeKindPointer {
			current = current.ElemType
		}
	}

	return current
}
