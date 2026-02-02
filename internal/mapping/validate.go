package mapping

import (
	"errors"
	"fmt"
	"strings"

	"caster-generator/internal/analyze"
)

// ValidationResult is returned by Validate.
// It contains a list of errors and warnings.
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationWarning
}

// ValidationError represents a schema+typegraph validation error.
type ValidationError struct {
	TypePair  string
	FieldPath string
	Message   string
}

func (e ValidationError) Error() string {
	var prefix []string
	if e.TypePair != "" {
		prefix = append(prefix, e.TypePair)
	}

	if e.FieldPath != "" {
		prefix = append(prefix, e.FieldPath)
	}

	if len(prefix) > 0 {
		return strings.Join(prefix, " ") + ": " + e.Message
	}

	return e.Message
}

// ValidationWarning represents a non-fatal validation issue.
type ValidationWarning struct {
	TypePair  string
	FieldPath string
	Message   string
}

// IsValid returns true if there are no errors.
func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// Error returns a combined error (or nil when valid).
func (r *ValidationResult) Error() error {
	if r.IsValid() {
		return nil
	}

	var parts []string
	for _, e := range r.Errors {
		parts = append(parts, e.Error())
	}

	return errors.New(strings.Join(parts, "; "))
}

func (r *ValidationResult) addError(typePair, fieldPath, msg string) {
	r.Errors = append(r.Errors, ValidationError{TypePair: typePair, FieldPath: fieldPath, Message: msg})
}

func (r *ValidationResult) addWarning(typePair, fieldPath, msg string) {
	r.Warnings = append(r.Warnings, ValidationWarning{TypePair: typePair, FieldPath: fieldPath, Message: msg})
}

// Validate validates a mapping definition against the given type graph.
// This is a structural validation step only; it doesn't try to prove type
// convertibility beyond what can be checked with the available type info.
func Validate(mf *MappingFile, graph *analyze.TypeGraph) *ValidationResult {
	res := &ValidationResult{}
	if mf == nil {
		res.addError("", "", "mapping file is nil")
		return res
	}

	if graph == nil {
		res.addError("", "", "type graph is nil")
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
			res.addError("", name, fmt.Sprintf("duplicate transform %q", name))
			continue
		}

		seenTransforms[name] = struct{}{}
	}

	for i := range mf.TypeMappings {
		tm := &mf.TypeMappings[i]
		tpStr := fmt.Sprintf("%s->%s", tm.Source, tm.Target)

		srcT := ResolveTypeID(tm.Source, graph)
		if srcT == nil {
			res.addError(tpStr, tm.Source, fmt.Sprintf("source type %q not found", tm.Source))
			continue
		}

		dstT := ResolveTypeID(tm.Target, graph)
		if dstT == nil {
			res.addError(tpStr, tm.Target, fmt.Sprintf("target type %q not found", tm.Target))
			continue
		}

		// 121 shorthand
		for sp, tp := range tm.OneToOne {
			if err := validatePathAgainstType(sp, srcT); err != nil {
				res.addError(tpStr, sp, fmt.Sprintf("invalid source path in 121: %v", err))
			}

			if err := validatePathAgainstType(tp, dstT); err != nil {
				res.addError(tpStr, tp, fmt.Sprintf("invalid target path in 121: %v", err))
			}
		}

		// fields + auto
		for _, fm := range append(append([]FieldMapping{}, tm.Fields...), tm.Auto...) {
			validateFieldMapping(res, tpStr, srcT, dstT, tm, &fm, seenTransforms)
		}

		// ignore paths
		for _, ig := range tm.Ignore {
			if err := validatePathAgainstType(ig, dstT); err != nil {
				res.addError(tpStr, ig, fmt.Sprintf("invalid ignore path: %v", err))
			}
		}
	}

	return res
}

func validateFieldMapping(
	res *ValidationResult,
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
			res.addError(typePairStr, "", "field mapping must specify target")
			continue
		}

		if err := validatePathAgainstType(t.Path, dstT); err != nil {
			res.addError(typePairStr, t.Path, fmt.Sprintf("invalid target path: %v", err))
		}

		if !t.Hint.IsValid() {
			res.addError(typePairStr, t.Path, fmt.Sprintf("invalid hint %q", t.Hint))
		}
	}

	// validate sources (unless default)
	if fm.Default == nil {
		if len(fm.Source) == 0 {
			res.addError(typePairStr, "", "field mapping must specify source (or default)")
		} else {
			for _, s := range fm.Source {
				if s.Path == "" {
					res.addError(typePairStr, "", "field mapping must specify source")
					continue
				}

				if err := validatePathAgainstType(s.Path, srcT); err != nil {
					res.addError(typePairStr, s.Path, fmt.Sprintf("invalid source path: %v", err))
				}

				if !s.Hint.IsValid() {
					res.addError(typePairStr, s.Path, fmt.Sprintf("invalid hint %q", s.Hint))
				}
			}
		}
	}

	// many:1 and many:many require a transform
	if fm.NeedsTransform() && fm.Transform == "" {
		res.addError(typePairStr, "", card.String()+" mapping requires transform")
	}

	// A referenced transform must exist in the registry.
	if fm.Transform != "" {
		if _, ok := knownTransforms[fm.Transform]; !ok {
			res.addError(typePairStr, "", fmt.Sprintf("referenced transform %q is not declared in transforms", fm.Transform))
		}
	}

	// validate extra definitions
	for _, ev := range fm.Extra {
		if ev.Name == "" {
			res.addError(typePairStr, "", "extra entry has empty name")
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
				res.addError(typePairStr, "", fmt.Sprintf("extra %q references an undeclared requires arg; add it under requires: or rename", ev.Name))
			}
		}

		if ev.Def.Source != "" {
			if err := validatePathAgainstType(ev.Def.Source, srcT); err != nil {
				res.addError(typePairStr, ev.Def.Source, fmt.Sprintf("invalid extra.def.source: %v", err))
			}
		}

		if ev.Def.Target != "" {
			if err := validatePathAgainstType(ev.Def.Target, dstT); err != nil {
				res.addError(typePairStr, ev.Def.Target, fmt.Sprintf("invalid extra.def.target: %v", err))
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
