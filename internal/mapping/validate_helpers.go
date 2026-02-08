package mapping

import (
	"fmt"
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/diagnostic"
)

// validateTargets validates the target field references in a field mapping.
func validateTargets(
	res *diagnostic.Diagnostics,
	typePairStr string,
	dstT *analyze.TypeInfo,
	fm *FieldMapping,
) {
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
}

// validateSources validates the source field references in a field mapping.
func validateSources(
	res *diagnostic.Diagnostics,
	typePairStr string,
	srcT *analyze.TypeInfo,
	parent *TypeMapping,
	fm *FieldMapping,
) {
	// Skip validation if using default value
	if fm.Default != nil {
		return
	}

	if len(fm.Source) == 0 {
		res.AddError("missing_source", "field mapping must specify source (or default)", typePairStr, "")
		return
	}

	for _, s := range fm.Source {
		if s.Path == "" {
			res.AddError("empty_source_path", "field mapping must specify source", typePairStr, "")
			continue
		}

		// Check if this source path starts with a required argument
		isReq := isRequiredArg(s.Path, parent)

		if !isReq {
			if err := validatePathAgainstType(s.Path, srcT); err != nil {
				res.AddError("invalid_source_path", fmt.Sprintf("invalid source path: %v", err), typePairStr, s.Path)
			}
		}

		if !s.Hint.IsValid() {
			res.AddError("invalid_hint", fmt.Sprintf("invalid hint %q", s.Hint), typePairStr, s.Path)
		}
	}
}

// isRequiredArg checks if a path starts with a required argument name.
func isRequiredArg(path string, parent *TypeMapping) bool {
	if parent == nil {
		return false
	}

	// Extract first segment of path
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return false
	}

	for _, req := range parent.Requires {
		if req.Name == parts[0] {
			return true
		}
	}

	return false
}

// validateTransform validates the transform reference in a field mapping.
func validateTransform(
	res *diagnostic.Diagnostics,
	typePairStr string,
	fm *FieldMapping,
	knownTransforms map[string]struct{},
) {
	card := fm.GetCardinality()

	// many:1 and many:many require a transform
	if fm.NeedsTransform() && fm.Transform == "" {
		res.AddError("missing_transform", card.String()+" mapping requires transform", typePairStr, "")
	}

	// A referenced transform must exist in the registry, unless it's a simple name
	// (without package prefix) which will have a stub generated.
	if fm.Transform != "" {
		if _, ok := knownTransforms[fm.Transform]; !ok {
			// Allow simple transform names without package prefix - stubs will be generated
			if strings.Contains(fm.Transform, ".") {
				res.AddError("unknown_transform",
					fmt.Sprintf("referenced transform %q is not declared in transforms", fm.Transform),
					typePairStr, "")
			}
		}
	}
}

// validateExtra validates the extra definitions in a field mapping.
func validateExtra(
	res *diagnostic.Diagnostics,
	typePairStr string,
	srcT, dstT *analyze.TypeInfo,
	parent *TypeMapping,
	fm *FieldMapping,
) {
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
				// Relax validation: if the extra arg has a definition (source or target),
				// it's creating a new value, not just forwarding a required arg.
				isDefinition := ev.Def.Source != "" || ev.Def.Target != ""

				if !isDefinition {
					res.AddError("undeclared_extra_arg",
						fmt.Sprintf("extra %q references an undeclared requires arg; add it under requires: or rename", ev.Name),
						typePairStr, "")
				}
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
