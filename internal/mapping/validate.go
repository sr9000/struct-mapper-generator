package mapping

import (
	"fmt"

	"caster-generator/internal/analyze"
	"caster-generator/internal/diagnostic"
)

// Validate validates a mapping definition against the given type graph.
// This is a structural validation step only; it doesn't try to prove type
// convertibility beyond what can be checked with the available type info.
func Validate(mf *MappingFile, graph *analyze.TypeGraph) *diagnostic.Diagnostics {
	res := &diagnostic.Diagnostics{}
	if mf == nil {
		res.AddError("mapping_is_nil", "mapping file is nil", "", "")
		return res
	}

	if graph == nil {
		res.AddError("graph_is_nil", "type graph is nil", "", "")
		return res
	}

	// Validate transform defs: detect duplicates (required by tests).
	seenTransforms := map[string]struct{}{}

	for i := range mf.Transforms {
		name := mf.Transforms[i].Name
		if name == "" {
			continue
		}

		if _, ok := seenTransforms[name]; ok {
			res.AddError("duplicate_transform", fmt.Sprintf("duplicate transform %q", name), "", name)
			continue
		}

		seenTransforms[name] = struct{}{}
	}

	for i := range mf.TypeMappings {
		tm := &mf.TypeMappings[i]
		tpStr := fmt.Sprintf("%s->%s", tm.Source, tm.Target)

		srcT := ResolveTypeID(tm.Source, graph)
		if srcT == nil {
			res.AddError("source_type_not_found", fmt.Sprintf("source type %q not found", tm.Source), tpStr, tm.Source)
			continue
		}

		dstT := ResolveTypeID(tm.Target, graph)
		if dstT == nil {
			// If GenerateTarget is true, skip target type validation
			// The target type will be generated during resolution
			if tm.GenerateTarget {
				// Skip field validation against target for generated types
				continue
			}

			res.AddError("target_type_not_found", fmt.Sprintf("target type %q not found", tm.Target), tpStr, tm.Target)

			continue
		}

		// 121 shorthand
		for sp, tp := range tm.OneToOne {
			if err := validatePathAgainstType(sp, srcT); err != nil {
				res.AddError("invalid_source_path", fmt.Sprintf("invalid source path in 121: %v", err), tpStr, sp)
			}

			if err := validatePathAgainstType(tp, dstT); err != nil {
				res.AddError("invalid_target_path", fmt.Sprintf("invalid target path in 121: %v", err), tpStr, tp)
			}
		}

		// fields + auto
		for _, fm := range append(append([]FieldMapping{}, tm.Fields...), tm.Auto...) {
			validateFieldMapping(res, tpStr, srcT, dstT, tm, &fm, seenTransforms)
		}

		// ignore paths
		for _, ig := range tm.Ignore {
			if err := validatePathAgainstType(ig, dstT); err != nil {
				res.AddError("invalid_ignore_path", fmt.Sprintf("invalid ignore path: %v", err), tpStr, ig)
			}
		}
	}

	return res
}

// validateFieldMapping validates a single field mapping within a type mapping.
func validateFieldMapping(
	res *diagnostic.Diagnostics,
	typePairStr string,
	srcT, dstT *analyze.TypeInfo,
	parent *TypeMapping,
	fm *FieldMapping,
	knownTransforms map[string]struct{},
) {
	if fm == nil {
		return
	}

	validateTargets(res, typePairStr, dstT, fm)
	validateSources(res, typePairStr, srcT, parent, fm)
	validateTransform(res, typePairStr, fm, knownTransforms)
	validateExtra(res, typePairStr, srcT, dstT, parent, fm)
}

func validatePathAgainstType(pathStr string, typeInfo *analyze.TypeInfo) error {
	fp, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	current := typeInfo
	for _, seg := range fp.Segments {
		if current == nil {
			return fmt.Errorf("nil type while resolving %q", seg.Name)
		}

		// Auto-deref pointers (matches resolver behavior).
		for current.Kind == analyze.TypeKindPointer {
			current = current.ElemType
			if current == nil {
				return fmt.Errorf("nil pointer element while resolving %q", seg.Name)
			}
		}

		// Resolve field on current struct.
		if current.Kind != analyze.TypeKindStruct {
			return fmt.Errorf("cannot access field %q on non-struct kind %s", seg.Name, current.Kind)
		}

		var fld *analyze.FieldInfo

		for i := range current.Fields {
			if current.Fields[i].Name == seg.Name {
				fld = &current.Fields[i]
				break
			}
		}

		if fld == nil {
			return fmt.Errorf("field %q not found in %s", seg.Name, current.ID)
		}

		if !fld.Exported {
			return fmt.Errorf("field %q is not exported", seg.Name)
		}

		current = fld.Type

		// Apply [] after selecting the field.
		if seg.IsSlice {
			for current.Kind == analyze.TypeKindPointer {
				current = current.ElemType
				if current == nil {
					return fmt.Errorf("nil pointer element while resolving %q", seg.Name)
				}
			}

			if current.Kind != analyze.TypeKindSlice {
				return fmt.Errorf("segment %q uses [] but resolved field is %s", seg.Name, current.Kind)
			}

			current = current.ElemType
			if current == nil {
				return fmt.Errorf("nil slice element while resolving %q", seg.Name)
			}
		}
	}

	return nil
}
