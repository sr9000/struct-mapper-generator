package mapping

import (
	"fmt"
	"strings"

	"caster-generator/internal/analyze"
)

// TransformRegistry holds validated transform definitions and provides lookup.
type TransformRegistry struct {
	transforms map[string]*ValidatedTransform
}

// ValidatedTransform represents a transform with resolved type information.
type ValidatedTransform struct {
	Def        *TransformDef
	SourceType *analyze.TypeInfo // Resolved source type (may be nil for basic types)
	TargetType *analyze.TypeInfo // Resolved target type (may be nil for basic types)
}

// NewTransformRegistry creates a new empty transform registry.
func NewTransformRegistry() *TransformRegistry {
	return &TransformRegistry{
		transforms: make(map[string]*ValidatedTransform),
	}
}

// BuildRegistry builds a transform registry from a MappingFile, validating
// transform signatures against the type graph.
func BuildRegistry(mf *MappingFile, graph *analyze.TypeGraph) (*TransformRegistry, []error) {
	registry := NewTransformRegistry()

	var errs []error

	for i := range mf.Transforms {
		def := &mf.Transforms[i]

		// Resolve source type
		var sourceType *analyze.TypeInfo
		if def.SourceType != "" && !isBasicTypeName(def.SourceType) {
			sourceType = resolveTypeID(def.SourceType, graph)
			if sourceType == nil {
				errs = append(errs, fmt.Errorf(
					"transform %q: source type %q not found",
					def.Name, def.SourceType))
			}
		}

		// Resolve target type
		var targetType *analyze.TypeInfo
		if def.TargetType != "" && !isBasicTypeName(def.TargetType) {
			targetType = resolveTypeID(def.TargetType, graph)
			if targetType == nil {
				errs = append(errs, fmt.Errorf(
					"transform %q: target type %q not found",
					def.Name, def.TargetType))
			}
		}

		registry.transforms[def.Name] = &ValidatedTransform{
			Def:        def,
			SourceType: sourceType,
			TargetType: targetType,
		}
	}

	return registry, errs
}

// Add adds a transform to the registry.
func (r *TransformRegistry) Add(def *TransformDef) {
	r.transforms[def.Name] = &ValidatedTransform{
		Def: def,
	}
}

// Get returns a validated transform by name, or nil if not found.
func (r *TransformRegistry) Get(name string) *ValidatedTransform {
	return r.transforms[name]
}

// Has returns true if a transform with the given name exists.
func (r *TransformRegistry) Has(name string) bool {
	_, exists := r.transforms[name]
	return exists
}

// All returns all transforms in the registry.
func (r *TransformRegistry) All() []*ValidatedTransform {
	result := make([]*ValidatedTransform, 0, len(r.transforms))
	for _, t := range r.transforms {
		result = append(result, t)
	}

	return result
}

// Names returns all transform names.
func (r *TransformRegistry) Names() []string {
	names := make([]string, 0, len(r.transforms))
	for name := range r.transforms {
		names = append(names, name)
	}

	return names
}

// FuncCall returns the fully qualified function call expression for a transform.
// Example: "transforms.PriceToAmount" or "mypkg.CustomTransform".
func (t *ValidatedTransform) FuncCall() string {
	if t.Def.Package != "" {
		return t.Def.Package + "." + t.Def.Func
	}

	return t.Def.Func
}

// isBasicTypeName checks if a type name refers to a Go basic type.
var basicTypes = map[string]bool{
	"bool":       true,
	"string":     true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uintptr":    true,
	"byte":       true,
	"rune":       true,
	"float32":    true,
	"float64":    true,
	"complex64":  true,
	"complex128": true,
}

// IsBasicTypeName returns true if the name refers to a Go basic type.
func IsBasicTypeName(name string) bool {
	return basicTypes[name]
}

func isBasicTypeName(name string) bool {
	return basicTypes[name]
}

// GenerateTransformName generates a unique transform name for a field mapping.
// Used for auto-generating transforms for many:1 or many:many mappings.
func GenerateTransformName(sources, targets StringOrArray) string {
	var parts []string

	// Add source field names
	for _, s := range sources {
		// Extract just the field name from path
		name := extractFieldName(s)
		parts = append(parts, name)
	}

	parts = append(parts, "To")

	// Add target field names
	for _, t := range targets {
		name := extractFieldName(t)
		parts = append(parts, name)
	}

	return strings.Join(parts, "")
}

// extractFieldName extracts the last field name from a path.
// "Items[].ProductID" -> "ProductID".
func extractFieldName(path string) string {
	// Remove slice notation
	path = strings.ReplaceAll(path, "[]", "")

	// Get last segment
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return path
	}

	return parts[len(parts)-1]
}

// GenerateStub generates a stub transform function signature for a transform definition.
// This helps users implement missing transforms.
func GenerateStub(def *TransformDef) string {
	sourceType := def.SourceType
	if sourceType == "" {
		sourceType = "interface{}"
	}

	targetType := def.TargetType
	if targetType == "" {
		targetType = "interface{}"
	}

	comment := "// " + def.Func + " transforms a value from source to target type."
	if def.Description != "" {
		comment = "// " + def.Func + " " + def.Description
	}

	return fmt.Sprintf(`%s
func %s(src %s) %s {
	// TODO: Implement transform
	panic("not implemented")
}`, comment, def.Func, sourceType, targetType)
}

// GenerateMultiSourceStub generates a stub for a transform with multiple source fields.
func GenerateMultiSourceStub(def *TransformDef, sourceFields []string) string {
	targetType := def.TargetType
	if targetType == "" {
		targetType = "interface{}"
	}

	// Build parameter list
	var params []string
	for _, field := range sourceFields {
		paramName := strings.ToLower(extractFieldName(field)[:1]) + extractFieldName(field)[1:]
		params = append(params, paramName+" interface{}")
	}

	comment := "// " + def.Func + " combines multiple source fields into a single target value."
	if def.Description != "" {
		comment = "// " + def.Func + " " + def.Description
	}

	return fmt.Sprintf(`%s
func %s(%s) %s {
	// TODO: Implement transform
	panic("not implemented")
}`, comment, def.Func, strings.Join(params, ", "), targetType)
}
