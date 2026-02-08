package gen

import (
	"fmt"

	"caster-generator/internal/analyze"
	"caster-generator/internal/plan"
)

// buildCollectionMapping is a helper for slice and map mappings.
func (g *Generator) buildCollectionMapping(
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
	mappingKind string,
) string {
	if len(m.SourcePaths) == 0 || len(m.TargetPaths) == 0 {
		return ""
	}

	srcField := "in." + m.SourcePaths[0].String()
	tgtField := "out." + m.TargetPaths[0].String()

	srcType := g.getFieldTypeInfo(pair.SourceType, m.SourcePaths[0].String())
	tgtType := g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())

	if srcType == nil || tgtType == nil {
		return fmt.Sprintf("// TODO: could not determine types for %s mapping %s -> %s",
			mappingKind, m.SourcePaths[0], m.TargetPaths[0])
	}

	// Build extra args string from m.Extra
	extraArgs := g.buildExtraArgsForNestedCall(m.Extra)

	return g.generateCollectionLoop(srcField, tgtField, srcType, tgtType, imports, 0, extraArgs)
}

// generateCollectionLoop generates the loop code for collection mappings.
func (g *Generator) generateCollectionLoop(
	srcField, tgtField string,
	srcType, tgtType *analyze.TypeInfo,
	imports map[string]importSpec,
	depth int,
	extraArgs string,
) string {
	if srcType == nil || tgtType == nil {
		return "// TODO: nil types in loop generation"
	}

	// Handle Slices and Arrays
	if (srcType.Kind == analyze.TypeKindSlice || srcType.Kind == analyze.TypeKindArray) &&
		(tgtType.Kind == analyze.TypeKindSlice || tgtType.Kind == analyze.TypeKindArray) {
		return g.generateSliceArrayLoop(srcField, tgtField, srcType, tgtType, imports, depth, extraArgs)
	}

	// Handle Maps
	if srcType.Kind == analyze.TypeKindMap && tgtType.Kind == analyze.TypeKindMap {
		return g.generateMapLoop(srcField, tgtField, srcType, tgtType, imports, depth, extraArgs)
	}

	return "// TODO: unsupported collection type combination " + srcType.Kind.String() + " -> " + tgtType.Kind.String()
}

// generateSliceArrayLoop generates the loop code for slice/array mappings.
func (g *Generator) generateSliceArrayLoop(
	srcField, tgtField string,
	srcType, tgtType *analyze.TypeInfo,
	imports map[string]importSpec,
	depth int,
	extraArgs string,
) string {
	idxVar := fmt.Sprintf("i_%d", depth)
	srcElem := g.getSliceElementType(srcType)
	tgtElem := g.getSliceElementType(tgtType)

	if srcElem == nil || tgtElem == nil {
		return "// TODO: unknown element types"
	}

	// Initialization
	initStmt := ""
	if tgtType.Kind == analyze.TypeKindSlice {
		initStmt = fmt.Sprintf("%s = make(%s, len(%s))\n",
			tgtField, g.typeRefString(tgtType, imports), srcField)
	}

	// Loop header
	loopHeader := fmt.Sprintf("for %s := range %s {", idxVar, srcField)

	// Inner body
	srcItem := fmt.Sprintf("%s[%s]", srcField, idxVar)
	tgtItem := fmt.Sprintf("%s[%s]", tgtField, idxVar)

	var body string

	// Recursion or conversion
	if g.isCollection(srcElem) && g.isCollection(tgtElem) {
		body = g.generateCollectionLoop(srcItem, tgtItem, srcElem, tgtElem, imports, depth+1, extraArgs)
	} else {
		// Leaf conversion
		tgtElemStr := g.typeRefString(tgtElem, imports)
		expr := g.buildValueConversionWithExtra(srcItem, srcElem, tgtElem, tgtElemStr, extraArgs)
		body = fmt.Sprintf("%s = %s", tgtItem, expr)
	}

	return fmt.Sprintf("%s%s\n\t%s\n}", initStmt, loopHeader, body)
}

// generateMapLoop generates the loop code for map mappings.
func (g *Generator) generateMapLoop(
	srcField, tgtField string,
	srcType, tgtType *analyze.TypeInfo,
	imports map[string]importSpec,
	depth int,
	extraArgs string,
) string {
	keyVar := fmt.Sprintf("k_%d", depth)
	valVar := fmt.Sprintf("v_%d", depth)

	srcVal := g.getMapValueType(srcType)
	tgtVal := g.getMapValueType(tgtType)

	srcKey := g.getMapKeyType(srcType)
	tgtKey := g.getMapKeyType(tgtType)

	if srcVal == nil || tgtVal == nil || srcKey == nil || tgtKey == nil {
		return "// TODO: unknown map types"
	}

	// Initialization
	initStmt := fmt.Sprintf("%s = make(%s, len(%s))\n",
		tgtField, g.typeRefString(tgtType, imports), srcField)

	loopHeader := fmt.Sprintf("for %s, %s := range %s {", keyVar, valVar, srcField)

	tgtKeyStr := g.typeRefString(tgtKey, imports)
	keyExpr := g.buildValueConversion(keyVar, srcKey, tgtKey, tgtKeyStr)

	tgtItem := fmt.Sprintf("%s[%s]", tgtField, keyExpr)

	var body string

	if g.isCollection(srcVal) && g.isCollection(tgtVal) {
		// For nested collections, we might need a block not just a string statement
		body = g.generateCollectionLoop(valVar, tgtItem, srcVal, tgtVal, imports, depth+1, extraArgs)
	} else {
		tgtValStr := g.typeRefString(tgtVal, imports)
		expr := g.buildValueConversionWithExtra(valVar, srcVal, tgtVal, tgtValStr, extraArgs)
		body = fmt.Sprintf("%s = %s", tgtItem, expr)
	}

	return fmt.Sprintf("%s%s\n\t%s\n}", initStmt, loopHeader, body)
}

// isCollection checks if a type is a collection (slice, array, or map).
func (g *Generator) isCollection(t *analyze.TypeInfo) bool {
	if t == nil {
		return false
	}

	return t.Kind == analyze.TypeKindSlice || t.Kind == analyze.TypeKindArray || t.Kind == analyze.TypeKindMap
}

// getSliceElementType returns the element type of a slice or array.
func (g *Generator) getSliceElementType(t *analyze.TypeInfo) *analyze.TypeInfo {
	if t.Kind == analyze.TypeKindSlice && t.ElemType != nil {
		return t.ElemType
	}

	if t.Kind == analyze.TypeKindArray && t.ElemType != nil {
		return t.ElemType
	}

	return nil
}

// getMapKeyType returns the key type of a map.
func (g *Generator) getMapKeyType(t *analyze.TypeInfo) *analyze.TypeInfo {
	if t.Kind == analyze.TypeKindMap && t.KeyType != nil {
		return t.KeyType
	}

	return nil
}

// getMapValueType returns the value type of a map.
func (g *Generator) getMapValueType(t *analyze.TypeInfo) *analyze.TypeInfo {
	if t.Kind == analyze.TypeKindMap && t.ElemType != nil {
		return t.ElemType
	}

	return nil
}

// buildValueConversionWithExtra builds a value conversion expression, appending extra args for nested caster calls.
func (g *Generator) buildValueConversionWithExtra(
	srcExpr string,
	srcType, tgtType *analyze.TypeInfo,
	tgtTypeStr string,
	extraArgs string,
) string {
	if g.typesIdentical(srcType, tgtType) {
		return srcExpr
	}

	if g.typesConvertible(srcType, tgtType) {
		return fmt.Sprintf("%s(%s)", tgtTypeStr, srcExpr)
	}

	if srcType.Kind == analyze.TypeKindStruct && tgtType.Kind == analyze.TypeKindStruct {
		casterName := g.nestedFunctionName(srcType, tgtType)
		if extraArgs != "" {
			return fmt.Sprintf("%s(%s, %s)", casterName, srcExpr, extraArgs)
		}

		return fmt.Sprintf("%s(%s)", casterName, srcExpr)
	}

	// Handle pointer-to-struct element conversion (both pointers)
	if srcType.Kind == analyze.TypeKindPointer && tgtType.Kind == analyze.TypeKindPointer {
		srcInner := srcType.ElemType
		tgtInner := tgtType.ElemType

		if srcInner != nil && tgtInner != nil &&
			srcInner.Kind == analyze.TypeKindStruct && tgtInner.Kind == analyze.TypeKindStruct {
			casterName := g.nestedFunctionName(srcInner, tgtInner)

			casterCall := casterName + "(*" + srcExpr + ")"
			if extraArgs != "" {
				casterCall = fmt.Sprintf("%s(*%s, %s)", casterName, srcExpr, extraArgs)
			}

			return fmt.Sprintf("func() %s { if %s == nil { return nil }; v := %s; return &v }()",
				tgtTypeStr, srcExpr, casterCall)
		}
	}

	// Handle pointer-to-struct -> struct element conversion (dereference + convert)
	if srcType.Kind == analyze.TypeKindPointer && tgtType.Kind == analyze.TypeKindStruct {
		srcInner := srcType.ElemType

		if srcInner != nil && srcInner.Kind == analyze.TypeKindStruct {
			casterName := g.nestedFunctionName(srcInner, tgtType)

			casterCall := fmt.Sprintf("%s(*%s)", casterName, srcExpr)
			if extraArgs != "" {
				casterCall = fmt.Sprintf("%s(*%s, %s)", casterName, srcExpr, extraArgs)
			}

			return fmt.Sprintf(
				"func() %s {"+
					" if %s == nil { return %s{} /* FIXME: zero value used for nil pointer */ };"+
					" return %s }()",
				tgtTypeStr, srcExpr, tgtTypeStr, casterCall)
		}
	}

	// Handle struct -> pointer-to-struct element conversion (convert + reference)
	if srcType.Kind == analyze.TypeKindStruct && tgtType.Kind == analyze.TypeKindPointer {
		tgtInner := tgtType.ElemType

		if tgtInner != nil && tgtInner.Kind == analyze.TypeKindStruct {
			casterName := g.nestedFunctionName(srcType, tgtInner)

			casterCall := fmt.Sprintf("%s(%s)", casterName, srcExpr)
			if extraArgs != "" {
				casterCall = fmt.Sprintf("%s(%s, %s)", casterName, srcExpr, extraArgs)
			}

			return fmt.Sprintf("func() %s { v := %s; return &v }()", tgtTypeStr, casterCall)
		}
	}

	// Fallback - hope for the best
	return srcExpr
}

// buildValueConversion builds a value conversion expression without extra args.
func (g *Generator) buildValueConversion(
	srcExpr string,
	srcType, tgtType *analyze.TypeInfo,
	tgtTypeStr string,
) string {
	if g.typesIdentical(srcType, tgtType) {
		return srcExpr
	}

	if g.typesConvertible(srcType, tgtType) {
		return fmt.Sprintf("%s(%s)", tgtTypeStr, srcExpr)
	}

	if srcType.Kind == analyze.TypeKindStruct && tgtType.Kind == analyze.TypeKindStruct {
		casterName := g.nestedFunctionName(srcType, tgtType)

		return fmt.Sprintf("%s(%s)", casterName, srcExpr)
	}

	// Handle pointer-to-struct element conversion (both pointers)
	if srcType.Kind == analyze.TypeKindPointer && tgtType.Kind == analyze.TypeKindPointer {
		srcInner := srcType.ElemType
		tgtInner := tgtType.ElemType

		if srcInner != nil && tgtInner != nil &&
			srcInner.Kind == analyze.TypeKindStruct && tgtInner.Kind == analyze.TypeKindStruct {
			casterName := g.nestedFunctionName(srcInner, tgtInner)

			return fmt.Sprintf("func() %s { if %s == nil { return nil }; v := %s(*%s); return &v }()",
				tgtTypeStr, srcExpr, casterName, srcExpr)
		}
	}

	// Handle pointer-to-struct -> struct element conversion (dereference + convert)
	if srcType.Kind == analyze.TypeKindPointer && tgtType.Kind == analyze.TypeKindStruct {
		srcInner := srcType.ElemType

		if srcInner != nil && srcInner.Kind == analyze.TypeKindStruct {
			casterName := g.nestedFunctionName(srcInner, tgtType)

			return fmt.Sprintf(
				"func() %s {"+
					" if %s == nil { return %s{} /* FIXME: zero value used for nil pointer */ };"+
					" return %s(*%s) }()",
				tgtTypeStr, srcExpr, tgtTypeStr, casterName, srcExpr)
		}
	}

	// Handle struct -> pointer-to-struct element conversion (convert + reference)
	if srcType.Kind == analyze.TypeKindStruct && tgtType.Kind == analyze.TypeKindPointer {
		tgtInner := tgtType.ElemType

		if tgtInner != nil && tgtInner.Kind == analyze.TypeKindStruct {
			casterName := g.nestedFunctionName(srcType, tgtInner)

			return fmt.Sprintf("func() %s { v := %s(%s); return &v }()", tgtTypeStr, casterName, srcExpr)
		}
	}

	// Fallback - hope for the best
	return srcExpr
}
