package analyze

import (
	"strings"
)

// TypePath builds a readable path string for a type.
// Examples:
//   - "Order" for a simple struct
//   - "Order.Items" for a nested field
//   - "Order.Items[]" for a slice field
//   - "Order.Items[].ProductID" for a field within slice elements
type TypePath struct {
	parts []string
}

// NewTypePath creates a new TypePath from a root type name.
func NewTypePath(root string) *TypePath {
	return &TypePath{
		parts: []string{root},
	}
}

// Field appends a field name to the path.
func (p *TypePath) Field(name string) *TypePath {
	return &TypePath{
		parts: append(append([]string{}, p.parts...), name),
	}
}

// Slice appends a slice indicator "[]" to the path.
func (p *TypePath) Slice() *TypePath {
	if len(p.parts) == 0 {
		return &TypePath{parts: []string{"[]"}}
	}
	newParts := make([]string, len(p.parts))
	copy(newParts, p.parts)
	newParts[len(newParts)-1] = newParts[len(newParts)-1] + "[]"
	return &TypePath{parts: newParts}
}

// Pointer appends a pointer indicator "*" to the path.
func (p *TypePath) Pointer() *TypePath {
	if len(p.parts) == 0 {
		return &TypePath{parts: []string{"*"}}
	}
	newParts := make([]string, len(p.parts))
	copy(newParts, p.parts)
	newParts[len(newParts)-1] = "*" + newParts[len(newParts)-1]
	return &TypePath{parts: newParts}
}

// String returns the full path string.
func (p *TypePath) String() string {
	return strings.Join(p.parts, ".")
}

// TypeStringer provides methods for creating readable type path strings.
type TypeStringer struct{}

// NewTypeStringer creates a new TypeStringer.
func NewTypeStringer() *TypeStringer {
	return &TypeStringer{}
}

// TypeString returns a human-readable string representation of a TypeInfo.
func (s *TypeStringer) TypeString(t *TypeInfo) string {
	if t == nil {
		return "<nil>"
	}

	switch t.Kind {
	case TypeKindBasic:
		return t.GoType.String()

	case TypeKindStruct:
		if t.IsNamed() {
			return t.ID.Name
		}
		return "struct{...}"

	case TypeKindPointer:
		if t.ElemType != nil {
			return "*" + s.TypeString(t.ElemType)
		}
		return "*<unknown>"

	case TypeKindSlice:
		if t.ElemType != nil {
			return "[]" + s.TypeString(t.ElemType)
		}
		return "[]<unknown>"

	case TypeKindAlias:
		if t.IsNamed() {
			return t.ID.Name
		}
		return s.TypeString(t.Underlying)

	case TypeKindExternal:
		if t.IsNamed() {
			return t.ID.String()
		}
		return t.GoType.String()

	default:
		return t.GoType.String()
	}
}

// FieldPath returns a path string for a field within a type.
// Example: Order, Items -> "Order.Items"
func (s *TypeStringer) FieldPath(typeName string, fieldNames ...string) string {
	path := NewTypePath(typeName)
	for _, fn := range fieldNames {
		path = path.Field(fn)
	}
	return path.String()
}

// BuildFieldPaths recursively builds all field paths for a struct type.
// Returns a map of path string to FieldInfo.
func (s *TypeStringer) BuildFieldPaths(root *TypeInfo, maxDepth int) map[string]*FieldInfo {
	result := make(map[string]*FieldInfo)
	if root == nil || root.Kind != TypeKindStruct {
		return result
	}

	rootName := root.ID.Name
	if rootName == "" {
		rootName = "root"
	}

	s.buildFieldPathsRecursive(root, NewTypePath(rootName), result, 0, maxDepth)
	return result
}

func (s *TypeStringer) buildFieldPathsRecursive(t *TypeInfo, path *TypePath, result map[string]*FieldInfo, depth, maxDepth int) {
	if depth > maxDepth || t == nil {
		return
	}

	for i := range t.Fields {
		field := &t.Fields[i]
		fieldPath := path.Field(field.Name)

		// Store the field at this path
		result[fieldPath.String()] = field

		// Recursively process nested types
		fieldType := field.Type
		s.processNestedType(fieldType, fieldPath, result, depth+1, maxDepth)
	}
}

func (s *TypeStringer) processNestedType(t *TypeInfo, path *TypePath, result map[string]*FieldInfo, depth, maxDepth int) {
	if t == nil || depth > maxDepth {
		return
	}

	switch t.Kind {
	case TypeKindStruct:
		s.buildFieldPathsRecursive(t, path, result, depth, maxDepth)

	case TypeKindPointer:
		// Process the element type
		if t.ElemType != nil {
			s.processNestedType(t.ElemType, path, result, depth, maxDepth)
		}

	case TypeKindSlice:
		// Add [] notation and process element type
		slicePath := path.Slice()
		if t.ElemType != nil {
			s.processNestedType(t.ElemType, slicePath, result, depth, maxDepth)
		}

	case TypeKindBasic, TypeKindAlias, TypeKindExternal, TypeKindUnknown:
		// Terminal types - nothing to recurse into
	}
}
