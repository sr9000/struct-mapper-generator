package mapping

import (
	"fmt"
	"strings"

	"caster-generator/internal/analyze"
)

// ValidationError represents a validation error with context.
type ValidationError struct {
	Path    string // Path in the YAML (e.g., "mappings[0].fields[2].target")
	Message string // Human-readable error message
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationResult contains all validation errors and warnings.
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
}

// IsValid returns true if there are no errors.
func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// AddError adds an error to the result.
func (r *ValidationResult) AddError(path, message string) {
	r.Errors = append(r.Errors, ValidationError{Path: path, Message: message})
}

// AddWarning adds a warning to the result.
func (r *ValidationResult) AddWarning(path, message string) {
	r.Warnings = append(r.Warnings, ValidationError{Path: path, Message: message})
}

// Error returns a combined error message if there are errors.
func (r *ValidationResult) Error() error {
	if r.IsValid() {
		return nil
	}

	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, e.Error())
	}

	return fmt.Errorf("validation failed:\n  %s", strings.Join(msgs, "\n  "))
}

// Validate validates a MappingFile against a TypeGraph.
// It checks that:
//   - Source and target types exist
//   - Field paths are valid
//   - Transform references are valid
//   - Cardinality rules are respected (many:1 needs transform).
func Validate(mf *MappingFile, graph *analyze.TypeGraph) *ValidationResult {
	result := &ValidationResult{}

	// Build transform registry for lookup
	transforms := make(map[string]*TransformDef)

	for i := range mf.Transforms {
		t := &mf.Transforms[i]
		path := fmt.Sprintf("transforms[%d]", i)

		if t.Name == "" {
			result.AddError(path+".name", "transform name is required")
			continue
		}

		if _, exists := transforms[t.Name]; exists {
			result.AddError(path+".name", fmt.Sprintf("duplicate transform name %q", t.Name))
			continue
		}

		transforms[t.Name] = t
	}

	// Validate each type mapping
	for i, tm := range mf.TypeMappings {
		basePath := fmt.Sprintf("mappings[%d]", i)
		validateTypeMapping(basePath, &tm, graph, transforms, result)
	}

	return result
}

// validateTypeMapping validates a single TypeMapping.
func validateTypeMapping(
	basePath string,
	tm *TypeMapping,
	graph *analyze.TypeGraph,
	transforms map[string]*TransformDef,
	result *ValidationResult,
) {
	// Validate source type
	sourceType := resolveTypeID(tm.Source, graph)
	if sourceType == nil {
		result.AddError(basePath+".source", fmt.Sprintf("type %q not found in type graph", tm.Source))
	}

	// Validate target type
	targetType := resolveTypeID(tm.Target, graph)
	if targetType == nil {
		result.AddError(basePath+".target", fmt.Sprintf("type %q not found in type graph", tm.Target))
	}

	// If either type is missing, skip field validation
	if sourceType == nil || targetType == nil {
		return
	}

	// Validate 121 shorthand mappings
	for source, target := range tm.OneToOne {
		path := fmt.Sprintf("%s.121[%s]", basePath, source)

		// Validate source path
		sourcePath, parseErr := ParsePath(source)
		if parseErr != nil {
			result.AddError(path+".source", parseErr.Error())
		} else {
			fieldErr := validateFieldPath(sourcePath, sourceType, "source")
			if fieldErr != nil {
				result.AddError(path+".source", fieldErr.Error())
			}
		}

		// Validate target path
		targetPath, parseErr := ParsePath(target)
		if parseErr != nil {
			result.AddError(path+".target", parseErr.Error())
		} else {
			fieldErr := validateFieldPath(targetPath, targetType, "target")
			if fieldErr != nil {
				result.AddError(path+".target", fieldErr.Error())
			}
		}
	}

	// Validate ignored fields
	for j, ignored := range tm.Ignore {
		path := fmt.Sprintf("%s.ignore[%d]", basePath, j)

		fp, parseErr := ParsePath(ignored)
		if parseErr != nil {
			result.AddError(path, parseErr.Error())
			continue
		}

		fieldErr := validateFieldPath(fp, targetType, "target")
		if fieldErr != nil {
			result.AddError(path, fieldErr.Error())
		}
	}

	// Validate explicit field mappings
	for j := range tm.Fields {
		fm := &tm.Fields[j]
		fieldPath := fmt.Sprintf("%s.fields[%d]", basePath, j)
		validateFieldMapping(fieldPath, fm, sourceType, targetType, transforms, result)
	}

	// Validate auto field mappings
	for j := range tm.Auto {
		fm := &tm.Auto[j]
		fieldPath := fmt.Sprintf("%s.auto[%d]", basePath, j)
		validateFieldMapping(fieldPath, fm, sourceType, targetType, transforms, result)
	}
}

// validateFieldMapping validates a single FieldMapping.
func validateFieldMapping(
	basePath string,
	fm *FieldMapping,
	sourceType *analyze.TypeInfo,
	targetType *analyze.TypeInfo,
	transforms map[string]*TransformDef,
	result *ValidationResult,
) {
	// Target is always required
	if fm.Target.IsEmpty() {
		result.AddError(basePath+".target", "target field path is required")
		return
	}

	// Validate all target paths
	for i, target := range fm.Target {
		targetPath, err := ParsePath(target.Path)
		if err != nil {
			result.AddError(fmt.Sprintf("%s.target[%d]", basePath, i), err.Error())
			continue
		}

		if err := validateFieldPath(targetPath, targetType, "target"); err != nil {
			result.AddError(fmt.Sprintf("%s.target[%d]", basePath, i), err.Error())
		}

		// Validate hint if present
		if target.Hint != HintNone && !target.Hint.IsValid() {
			result.AddError(fmt.Sprintf("%s.target[%d]", basePath, i),
				fmt.Sprintf("invalid hint %q (expected 'dive' or 'final')", target.Hint))
		}
	}

	// If ignore is set, no other fields should be set
	if fm.Ignore {
		if !fm.Source.IsEmpty() || fm.Default != nil || fm.Transform != "" {
			result.AddWarning(basePath, "ignore is set; source, default, and transform are ignored")
		}

		return
	}

	// Count how many value sources are specified
	hasSource := !fm.Source.IsEmpty()
	hasDefault := fm.Default != nil

	if !hasSource && !hasDefault && fm.Transform == "" {
		result.AddError(basePath, "must specify either source, default, or transform")
	}

	if hasSource && hasDefault {
		result.AddWarning(basePath, "both source and default specified; source takes precedence")
	}

	// Validate all source paths
	for i, source := range fm.Source {
		sourcePath, err := ParsePath(source.Path)
		if err != nil {
			result.AddError(fmt.Sprintf("%s.source[%d]", basePath, i), err.Error())
			continue
		}

		if err := validateFieldPath(sourcePath, sourceType, "source"); err != nil {
			result.AddError(fmt.Sprintf("%s.source[%d]", basePath, i), err.Error())
		}

		// Validate hint if present
		if source.Hint != HintNone && !source.Hint.IsValid() {
			result.AddError(fmt.Sprintf("%s.source[%d]", basePath, i),
				fmt.Sprintf("invalid hint %q (expected 'dive' or 'final')", source.Hint))
		}
	}

	// Validate cardinality rules
	card := fm.GetCardinality()

	// many:1 and many:many require transform
	if fm.NeedsTransform() && fm.Transform == "" {
		result.AddError(basePath, fmt.Sprintf(
			"%s mapping requires a transform function", card))
	}

	// Validate transform reference
	if fm.Transform != "" {
		if _, exists := transforms[fm.Transform]; !exists {
			result.AddError(basePath+".transform", fmt.Sprintf("transform %q not defined", fm.Transform))
		}
	}
}

// validateFieldPath validates that a field path exists in the given type.
func validateFieldPath(path FieldPath, typeInfo *analyze.TypeInfo, context string) error {
	current := typeInfo

	for i, seg := range path.Segments {
		// Must be a struct to access fields
		if current.Kind != analyze.TypeKindStruct {
			return fmt.Errorf("%s field path segment %q: expected struct, got %s",
				context, seg.Name, current.Kind)
		}

		// Find the field
		var field *analyze.FieldInfo

		for j := range current.Fields {
			if current.Fields[j].Name == seg.Name {
				field = &current.Fields[j]
				break
			}
		}

		if field == nil {
			return fmt.Errorf("%s field %q not found in type %s",
				context, seg.Name, current.ID)
		}

		if !field.Exported {
			return fmt.Errorf("%s field %q is not exported",
				context, seg.Name)
		}

		// Move to the field's type
		current = field.Type

		// Handle slice notation
		if seg.IsSlice {
			if current.Kind != analyze.TypeKindSlice {
				return fmt.Errorf("%s field path segment %q: expected slice, got %s",
					context, seg.Name+"[]", current.Kind)
			}
			// Move into slice element type
			current = current.ElemType
		}

		// Handle pointer dereferencing (auto-deref)
		if current.Kind == analyze.TypeKindPointer && i < len(path.Segments)-1 {
			current = current.ElemType
		}
	}

	return nil
}

// resolveTypeID finds a type in the graph by a short or full identifier.
// Supports: "Order", "store.Order", "caster-generator/store.Order".
func resolveTypeID(name string, graph *analyze.TypeGraph) *analyze.TypeInfo {
	// Try exact match first
	for id, info := range graph.Types {
		if id.String() == name {
			return info
		}
	}

	// Try package.Type format
	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		pkgShort := parts[0]
		typeName := parts[1]

		for id, info := range graph.Types {
			if id.Name == typeName {
				// Check if package path ends with the short name
				if strings.HasSuffix(id.PkgPath, "/"+pkgShort) || id.PkgPath == pkgShort {
					return info
				}
				// Also check if the package name (not path) matches
				// This handles cases where package name differs from directory name
				if pkgInfo := graph.Packages[id.PkgPath]; pkgInfo != nil && pkgInfo.Name == pkgShort {
					return info
				}
			}
		}
	}

	// Try just type name (ambiguous but useful for simple cases)
	for id, info := range graph.Types {
		if id.Name == name {
			return info
		}
	}

	return nil
}

// ResolveTypeID is exported for use by other packages.
func ResolveTypeID(name string, graph *analyze.TypeGraph) *analyze.TypeInfo {
	return resolveTypeID(name, graph)
}
