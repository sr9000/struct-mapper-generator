package mapping

import (
	"fmt"
	"strings"

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

	card := fm.GetCardinality()

	// validate target
	for _, t := range fm.Target {
		if t.Path == "" {
			res.AddError("missing_target_path", "field mapping must specify target", typePairStr, "")
			continue
		}

		if err := validatePathAgainstType(t.Path, dstT); err != nil {
			res.AddError("invalid_target_path", fmt.Sprintf("invalid target path: %v", err), typePairStr, t.Path)
		}

		if !t.Hint.IsValid() {
			res.AddError("invalid_hint", fmt.Sprintf("invalid hint %q", t.Hint), typePairStr, t.Path)
		}
	}

	// validate sources (unless default)
	if fm.Default == nil {
		if len(fm.Source) == 0 {
			res.AddError("missing_source", "field mapping must specify source (or default)", typePairStr, "")
		} else {
			for _, s := range fm.Source {
				if s.Path == "" {
					res.AddError("empty_source_path", "field mapping must specify source", typePairStr, "")
					continue
				}

				if err := validatePathAgainstType(s.Path, srcT); err != nil {
					res.AddError("invalid_source_path", fmt.Sprintf("invalid source path: %v", err), typePairStr, s.Path)
				}

				if !s.Hint.IsValid() {
					res.AddError("invalid_hint", fmt.Sprintf("invalid hint %q", s.Hint), typePairStr, s.Path)
				}
			}
		}
	}

	// many:1 and many:many require a transform
	if fm.NeedsTransform() && fm.Transform == "" {
		res.AddError("missing_transform", card.String()+" mapping requires transform", typePairStr, "")
	}

	// A referenced transform must exist in the registry.
	if fm.Transform != "" {
		if _, ok := knownTransforms[fm.Transform]; !ok {
			res.AddError("unknown_transform",
				fmt.Sprintf("referenced transform %q is not declared in transforms", fm.Transform),
				typePairStr, "")
		}
	}

	// validate extra definitions
	for _, ev := range fm.Extra {
		if ev.Name == "" {
			res.AddError("empty_extra_name", "extra entry has empty name", typePairStr, "")
			continue
		}

		// If this extra is intended to satisfy a required argument, ensure it's declared.
		if parent != nil {
			declared := false

			for i := range parent.Requires {
				if parent.Requires[i].Name == ev.Name {
					declared = true
					break
				}
			}

			if !declared {
				res.AddError("undeclared_extra_arg",
					fmt.Sprintf("extra %q references an undeclared requires arg; add it under requires: or rename", ev.Name),
					typePairStr, "")
			}
		}

		if ev.Def.Source != "" {
			if err := validatePathAgainstType(ev.Def.Source, srcT); err != nil {
				res.AddError("invalid_extra_source", fmt.Sprintf("invalid extra.def.source: %v", err), typePairStr, ev.Def.Source)
			}
		}

		if ev.Def.Target != "" {
			if err := validatePathAgainstType(ev.Def.Target, dstT); err != nil {
				res.AddError("invalid_extra_target", fmt.Sprintf("invalid extra.def.target: %v", err), typePairStr, ev.Def.Target)
			}
		}
	}
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

// ResolveTypeID resolves a type ID string like:
// - "store.Order" (short)
// - "caster-generator/store.Order" (full)
// - "Order" (name only).
func ResolveTypeID(typeIDStr string, graph *analyze.TypeGraph) *analyze.TypeInfo {
	if graph == nil {
		return nil
	}

	// Name-only: best-effort match by type name.
	if !strings.Contains(typeIDStr, ".") {
		name := typeIDStr
		if name == "" {
			return nil
		}

		for id, t := range graph.Types {
			if id.Name == name {
				return t
			}
		}

		return nil
	}

	lastDot := strings.LastIndex(typeIDStr, ".")
	if lastDot < 0 {
		return nil
	}

	pkgStr := typeIDStr[:lastDot]

	name := typeIDStr[lastDot+1:]
	if pkgStr == "" || name == "" {
		return nil
	}

	// 1) exact match (for fully qualified import path)
	if t := graph.GetType(analyze.TypeID{PkgPath: pkgStr, Name: name}); t != nil {
		return t
	}

	// 2) suffix match (for short forms like "store.Order" vs "caster-generator/store.Order")
	for id, t := range graph.Types {
		if id.Name != name {
			continue
		}

		if id.PkgPath == pkgStr || strings.HasSuffix(id.PkgPath, "/"+pkgStr) {
			return t
		}
	}

	return nil
}
