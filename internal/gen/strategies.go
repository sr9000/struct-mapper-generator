package gen

import (
	"fmt"
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/plan"
)

// applyConversionStrategy applies the conversion strategy to the assignment.
func (g *Generator) applyConversionStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	switch m.Strategy {
	case plan.StrategyDirectAssign:
		// Simple assignment, nothing extra needed

	case plan.StrategyConvert:
		g.applyConvertStrategy(assignment, m, pair, imports)

	case plan.StrategyPointerDeref:
		g.applyPointerDerefStrategy(assignment, m, pair)

	case plan.StrategyPointerWrap:
		g.applyPointerWrapStrategy(assignment, m, pair, imports)

	case plan.StrategySliceMap:
		assignment.IsSlice = true
		assignment.SliceElemVar = "i"
		assignment.SliceBody = g.buildSliceMapping(m, pair, imports)

	case plan.StrategyMap:
		assignment.IsMap = true
		assignment.MapBody = g.buildMapMapping(m, pair, imports)

	case plan.StrategyPointerNestedCast:
		g.applyPointerNestedCastStrategy(assignment, m, pair, imports)

	case plan.StrategyNestedCast:
		g.applyNestedCastStrategy(assignment, m, pair)

	case plan.StrategyTransform:
		g.applyTransformStrategy(assignment, m, pair)

	case plan.StrategyDefault:
		if m.Default != nil {
			assignment.SourceExpr = *m.Default
		}

	case plan.StrategyIgnore:
		// Already handled above
	}
}

// applyConvertStrategy applies the type conversion strategy.
func (g *Generator) applyConvertStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	if len(m.TargetPaths) > 0 {
		targetType := g.getFieldType(pair.TargetType, m.TargetPaths[0].String())
		if targetType != nil {
			assignment.SourceExpr = g.wrapConversion(assignment.SourceExpr, targetType, imports)
		}
	}
}

// applyPointerDerefStrategy applies the pointer dereference strategy.
func (g *Generator) applyPointerDerefStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
) {
	assignment.NeedsNilCheck = true
	// Keep the original pointer expression for the nil-check; use a dereferenced
	// expression for the actual assignment.
	assignment.NilDefault = g.zeroValue(pair.TargetType, m.TargetPaths)

	if len(m.TargetPaths) > 0 {
		ft := g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())
		if ft != nil && ft.Kind == analyze.TypeKindStruct {
			assignment.NilDefault += " /* FIXME: zero value used for nil pointer */"
		}
	}

	assignment.NilCheckExpr = assignment.SourceExpr
	assignment.SourceExpr = "*" + assignment.SourceExpr
}

// applyPointerWrapStrategy applies the pointer wrap strategy.
func (g *Generator) applyPointerWrapStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	if len(m.SourcePaths) > 0 {
		typeStr := g.getFieldTypeString(pair.SourceType, m.SourcePaths[0].String(), imports)
		srcExpr := g.sourceFieldExpr(m.SourcePaths, m, pair)
		assignment.SourceExpr = fmt.Sprintf("func() *%s { v := %s; return &v }()", typeStr, srcExpr)
	}
}

// applyPointerNestedCastStrategy applies the pointer nested cast strategy.
func (g *Generator) applyPointerNestedCastStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	if len(m.SourcePaths) == 0 || len(m.TargetPaths) == 0 {
		return
	}

	srcType := g.getFieldTypeInfo(pair.SourceType, m.SourcePaths[0].String())
	tgtType := g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())

	if srcType == nil || tgtType == nil {
		return
	}

	// Get the element types (what the pointers point to)
	srcElem := srcType
	tgtElem := tgtType

	if srcType.Kind == analyze.TypeKindPointer && srcType.ElemType != nil {
		srcElem = srcType.ElemType
	}

	if tgtType.Kind == analyze.TypeKindPointer && tgtType.ElemType != nil {
		tgtElem = tgtType.ElemType
	}

	casterName := g.nestedFunctionName(srcElem, tgtElem)
	tgtElemStr := g.typeRefString(tgtElem, imports)

	// Generate: func() *TargetType { if src == nil { return nil }; v := Caster(*src); return &v }()
	assignment.SourceExpr = fmt.Sprintf(
		"func() *%s { if %s == nil { return nil }; v := %s(*%s); return &v }()",
		tgtElemStr, assignment.SourceExpr, casterName, assignment.SourceExpr,
	)
}

// applyNestedCastStrategy applies the nested cast strategy.
func (g *Generator) applyNestedCastStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
) {
	if len(m.SourcePaths) == 0 {
		return
	}

	srcType := g.getFieldTypeInfo(pair.SourceType, m.SourcePaths[0].String())
	tgtType := g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())

	if srcType != nil && tgtType != nil {
		casterName := g.nestedFunctionName(srcType, tgtType)
		assignment.NestedCaster = casterName
		// Always call the nested caster with the resolved source expression.
		assignment.SourceExpr = fmt.Sprintf("%s(%s)", casterName, assignment.SourceExpr)
	}
}

// applyTransformStrategy applies the transform function call strategy.
func (g *Generator) applyTransformStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
) {
	if m.Transform == "" {
		return
	}

	args := g.buildTransformArgs(m.SourcePaths, pair)

	// Append extras after explicit source paths (stable order as specified in YAML).
	// Extras can reference either source fields or target fields.
	if len(m.Extra) > 0 {
		var extraArgs []string

		for _, ev := range m.Extra {
			// Prefer explicit source/target, else fallback to the extra name.
			if ev.Def.Source != "" {
				extraArgs = append(extraArgs, "in."+ev.Def.Source)
				continue
			}

			if ev.Def.Target != "" {
				extraArgs = append(extraArgs, "out."+ev.Def.Target)
				continue
			}

			// If Name matches a required arg, it should be passed verbatim.
			isReq := false

			for _, req := range pair.Requires {
				if req.Name == ev.Name {
					isReq = true
					break
				}
			}

			if isReq {
				extraArgs = append(extraArgs, ev.Name)
			} else {
				extraArgs = append(extraArgs, "in."+ev.Name)
			}
		}

		if args == "" {
			args = strings.Join(extraArgs, ", ")
		} else {
			args = args + ", " + strings.Join(extraArgs, ", ")
		}
	}

	assignment.SourceExpr = fmt.Sprintf("%s(%s)", m.Transform, args)
}

// buildSliceMapping generates the slice mapping code.
func (g *Generator) buildSliceMapping(
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) string {
	return g.buildCollectionMapping(m, pair, imports, "slice")
}

// buildMapMapping generates the map mapping code.
func (g *Generator) buildMapMapping(
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) string {
	return g.buildCollectionMapping(m, pair, imports, "map")
}

// buildExtraArgsForNestedCall builds the extra arguments string for a nested caster call.
func (g *Generator) buildExtraArgsForNestedCall(extra []mapping.ExtraVal) string {
	if len(extra) == 0 {
		return ""
	}

	var args []string

	for _, ev := range extra {
		switch {
		case ev.Def.Target != "":
			// If the extra has a target definition, use "out.<target>"
			args = append(args, "out."+ev.Def.Target)
		case ev.Def.Source != "":
			// If the extra has a source definition, use "in.<source>"
			args = append(args, "in."+ev.Def.Source)
		default:
			// Just use the name directly (for requires args passed through)
			args = append(args, ev.Name)
		}
	}

	return strings.Join(args, ", ")
}
