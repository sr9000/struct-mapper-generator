package analyze

import (
	"go/types"
	"reflect"

	"caster-generator/internal/common"
)

// TypeID uniquely identifies a type by its package path and name.
type TypeID struct {
	PkgPath string // e.g., "caster-generator/store"
	Name    string // e.g., "Order"
}

// String returns a human-readable representation of the TypeID.
func (t TypeID) String() string {
	if t.PkgPath == "" {
		return t.Name
	}

	return t.PkgPath + "." + t.Name
}

// TypeKind represents the kind of a type.
type TypeKind int

const (
	TypeKindUnknown  TypeKind = iota
	TypeKindBasic             // int, string, bool, etc.
	TypeKindStruct            // struct type
	TypeKindPointer           // pointer to another type
	TypeKindSlice             // slice of another type
	TypeKindArray             // array of another type
	TypeKindAlias             // type alias (named type wrapping another)
	TypeKindExternal          // external/opaque type (e.g., time.Time)
)

// String returns a human-readable representation of the TypeKind.
func (k TypeKind) String() string {
	switch k {
	case TypeKindBasic:
		return "basic"
	case TypeKindStruct:
		return "struct"
	case TypeKindPointer:
		return "pointer"
	case TypeKindSlice:
		return "slice"
	case TypeKindArray:
		return "array"
	case TypeKindAlias:
		return "alias"
	case TypeKindExternal:
		return "external"
	default:
		return common.UnknownStr
	}
}

// TypeInfo describes a Go type in the type graph.
type TypeInfo struct {
	ID          TypeID      // Unique identifier (empty for unnamed types like *T or []T)
	Kind        TypeKind    // Kind of type
	Underlying  *TypeInfo   // For named types, the underlying type
	ElemType    *TypeInfo   // For pointers and slices, the element type
	Fields      []FieldInfo // For structs, the list of fields
	GoType      types.Type  // The original go/types.Type (for compatibility checks)
	IsGenerated bool        // True if the type is virtual/generated
}

// IsNamed returns true if this type has a name (TypeID is set).
func (t *TypeInfo) IsNamed() bool {
	return t.ID.Name != ""
}

// FieldInfo describes a struct field.
type FieldInfo struct {
	Name     string            // Go field name
	Exported bool              // Whether the field is exported
	Type     *TypeInfo         // Field type
	Tag      reflect.StructTag // Raw struct tag
	Embedded bool              // Whether the field is embedded (anonymous)
	Index    int               // Field index in the struct
}

// JSONName returns the JSON tag name if present, otherwise the field name.
func (f *FieldInfo) JSONName() string {
	if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
		// Parse first part before comma
		for i := range len(tag) {
			if tag[i] == ',' {
				return tag[:i]
			}
		}

		return tag
	}

	return f.Name
}

// HasTag returns true if the field has the specified tag.
func (f *FieldInfo) HasTag(key string) bool {
	return f.Tag.Get(key) != ""
}

// GetTag returns the value of the specified tag.
func (f *FieldInfo) GetTag(key string) string {
	return f.Tag.Get(key)
}

// TypeGraph holds all analyzed types from loaded packages.
type TypeGraph struct {
	// Types maps TypeID to TypeInfo for all named types.
	Types map[TypeID]*TypeInfo
	// Packages maps package paths to their package info.
	Packages map[string]*PackageInfo
}

// NewTypeGraph creates a new empty TypeGraph.
func NewTypeGraph() *TypeGraph {
	return &TypeGraph{
		Types:    make(map[TypeID]*TypeInfo),
		Packages: make(map[string]*PackageInfo),
	}
}

// GetType returns the TypeInfo for a given TypeID, or nil if not found.
func (g *TypeGraph) GetType(id TypeID) *TypeInfo {
	return g.Types[id]
}

// PackageInfo holds information about a loaded package.
type PackageInfo struct {
	Path  string   // Import path
	Name  string   // Package name
	Types []TypeID // Named types defined in this package
}
