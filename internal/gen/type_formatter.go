package gen

import (
	"slices"
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/common"
	"caster-generator/internal/mapping"
)

// typeRef is a reference to a type with optional package qualifier.
type typeRef struct {
	Package   string // Package alias (empty if same package or builtin)
	Name      string // Type name
	IsPointer bool
	IsSlice   bool
	ElemRef   *typeRef // For slice/pointer, the element type
}

// String returns the full type string (e.g., "store.Order", "*warehouse.Product", "[]int").
func (t typeRef) String() string {
	var sb strings.Builder

	if t.IsPointer {
		sb.WriteString("*")
	}

	if t.IsSlice {
		sb.WriteString("[]")

		if t.ElemRef != nil {
			sb.WriteString(t.ElemRef.String())

			return sb.String()
		}
	}

	if t.Package != "" {
		sb.WriteString(t.Package)
		sb.WriteString(".")
	}

	sb.WriteString(t.Name)

	return sb.String()
}

// importSpec represents an import statement.
type importSpec struct {
	Alias string
	Path  string
}

// getPkgName returns the package name for a given package path.
// It tries to look up the name from the type graph, falling back to the path base alias.
func (g *Generator) getPkgName(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}

	if g.graph != nil {
		if pkgInfo, ok := g.graph.Packages[pkgPath]; ok {
			return pkgInfo.Name
		}
	}

	return common.PkgAlias(pkgPath)
}

// addImport adds an import to the imports map.
func (g *Generator) addImport(imports map[string]importSpec, pkgPath string) {
	if pkgPath == "" {
		return
	}

	alias := g.getPkgName(pkgPath)
	imports[pkgPath] = importSpec{
		Alias: alias,
		Path:  pkgPath,
	}
}

// typeRefString returns the string representation of a type for use in generated code.
func (g *Generator) typeRefString(t *analyze.TypeInfo, imports map[string]importSpec) string {
	if t == nil {
		return common.InterfaceTypeStr
	}

	switch t.Kind {
	case analyze.TypeKindBasic:
		return t.ID.Name

	case analyze.TypeKindPointer:
		if t.ElemType != nil {
			return "*" + g.typeRefString(t.ElemType, imports)
		}

		return "*" + common.InterfaceTypeStr

	case analyze.TypeKindSlice:
		if t.ElemType != nil {
			return "[]" + g.typeRefString(t.ElemType, imports)
		}

		return "[]" + common.InterfaceTypeStr

	case analyze.TypeKindMap:
		key := common.InterfaceTypeStr
		val := common.InterfaceTypeStr

		if t.KeyType != nil {
			key = g.typeRefString(t.KeyType, imports)
		}

		if t.ElemType != nil {
			val = g.typeRefString(t.ElemType, imports)
		}

		return "map[" + key + "]" + val

	case analyze.TypeKindArray:
		// Keep length information by using go/types' string.
		// This avoids having to store the array length explicitly in TypeInfo.
		return t.GoType.String()

	case analyze.TypeKindStruct, analyze.TypeKindExternal, analyze.TypeKindAlias:
		// If the type has a package path, use it for import and qualification.
		// Even if IsGenerated is true, if PkgPath is set, we treat it as a cross-package reference
		// unless we are generating into that same package.
		if t.ID.PkgPath != "" {
			// If we are generating code IN the same package as the type, omit prefix/import.
			if t.ID.PkgPath == g.contextPkgPath {
				return t.ID.Name
			}

			g.addImport(imports, t.ID.PkgPath)

			return g.getPkgName(t.ID.PkgPath) + "." + t.ID.Name
		}

		return t.ID.Name

	default:
		return common.InterfaceTypeStr
	}
}

// getFieldTypeString returns the type string for a field path.
func (g *Generator) getFieldTypeString(
	typeInfo *analyze.TypeInfo,
	fieldPath string,
	imports map[string]importSpec,
) string {
	ft := g.getFieldTypeInfo(typeInfo, fieldPath)
	if ft == nil {
		return common.InterfaceTypeStr
	}

	return g.typeRefString(ft, imports)
}

// wrapConversion wraps an expression with type conversion.
func (g *Generator) wrapConversion(
	expr string,
	targetType *analyze.TypeInfo,
	imports map[string]importSpec,
) string {
	typeStr := g.typeRefString(targetType, imports)

	return typeStr + "(" + expr + ")"
}

// zeroValue returns the zero value for a target type.
func (g *Generator) zeroValue(typeInfo *analyze.TypeInfo, paths []mapping.FieldPath) string {
	if len(paths) == 0 {
		return `""`
	}

	ft := g.getFieldTypeInfo(typeInfo, paths[0].String())
	if ft == nil {
		return `""`
	}

	return g.zeroValueForType(ft)
}

// zeroValueForType returns the zero value for a given type.
func (g *Generator) zeroValueForType(ft *analyze.TypeInfo) string {
	switch ft.Kind {
	case analyze.TypeKindBasic:
		return g.zeroValueForBasicType(ft.ID.Name)

	case analyze.TypeKindPointer, analyze.TypeKindSlice, analyze.TypeKindArray, analyze.TypeKindMap:
		return "nil"

	case analyze.TypeKindStruct:
		return g.typeRefString(ft, nil) + "{}"

	default:
		return `""`
	}
}

// zeroValueForBasicType returns the zero value for a basic type.
func (g *Generator) zeroValueForBasicType(name string) string {
	switch name {
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "0"
	case "bool":
		return "false"
	default:
		return `""`
	}
}

// typesIdentical checks if two types are identical.
func (g *Generator) typesIdentical(a, b *analyze.TypeInfo) bool {
	if a == nil || b == nil {
		return false
	}

	if a.Kind != b.Kind {
		return false
	}

	kinds := []analyze.TypeKind{analyze.TypeKindPointer, analyze.TypeKindSlice, analyze.TypeKindArray}
	if slices.Contains(kinds, a.Kind) {
		return g.typesIdentical(a.ElemType, b.ElemType)
	}

	return a.ID == b.ID
}

// typesConvertible checks if type a can be converted to type b.
func (g *Generator) typesConvertible(a, b *analyze.TypeInfo) bool {
	if a == nil || b == nil {
		return false
	}

	// Basic types are usually convertible among numeric types
	if a.Kind == analyze.TypeKindBasic && b.Kind == analyze.TypeKindBasic {
		return true
	}

	// For pointer, slice, array types - both types must be the same kind
	// and element types must be convertible
	if a.Kind == b.Kind {
		kinds := []analyze.TypeKind{analyze.TypeKindPointer, analyze.TypeKindSlice, analyze.TypeKindArray}
		if slices.Contains(kinds, a.Kind) {
			return g.typesConvertible(a.ElemType, b.ElemType)
		}
	}

	// Same named types
	if a.ID == b.ID && a.ID.Name != "" {
		return true
	}

	// Handle Aliases: if underlying types are convertible, then aliases are convertible
	if a.Kind == analyze.TypeKindAlias {
		return g.typesConvertible(a.Underlying, b)
	}

	if b.Kind == analyze.TypeKindAlias {
		return g.typesConvertible(a, b.Underlying)
	}

	return false
}

// typeInfoFromString creates a TypeInfo from a type string (e.g., "uint", "string", "*int").
func (g *Generator) typeInfoFromString(typeStr string) *analyze.TypeInfo {
	if typeStr == "" || typeStr == common.InterfaceTypeStr {
		return nil
	}

	// Handle pointer types
	if strings.HasPrefix(typeStr, "*") {
		elemType := g.typeInfoFromString(typeStr[1:])

		return &analyze.TypeInfo{
			ID:       analyze.TypeID{Name: typeStr},
			Kind:     analyze.TypeKindPointer,
			ElemType: elemType,
		}
	}

	// Handle slice types
	if strings.HasPrefix(typeStr, "[]") {
		elemType := g.typeInfoFromString(typeStr[2:])

		return &analyze.TypeInfo{
			ID:       analyze.TypeID{Name: typeStr},
			Kind:     analyze.TypeKindSlice,
			ElemType: elemType,
		}
	}

	// Handle map types (basic parsing)
	if strings.HasPrefix(typeStr, "map[") {
		// For simplicity, return as basic type - the string will be preserved
		return &analyze.TypeInfo{
			ID:   analyze.TypeID{Name: typeStr},
			Kind: analyze.TypeKindBasic,
		}
	}

	// Check if it's a qualified type (has a dot for package)
	if dotIdx := strings.LastIndex(typeStr, "."); dotIdx > 0 {
		pkgPath := typeStr[:dotIdx]
		name := typeStr[dotIdx+1:]

		return &analyze.TypeInfo{
			ID:   analyze.TypeID{PkgPath: pkgPath, Name: name},
			Kind: analyze.TypeKindExternal,
		}
	}

	// Basic type
	return &analyze.TypeInfo{
		ID:   analyze.TypeID{Name: typeStr},
		Kind: analyze.TypeKindBasic,
	}
}

// getFieldTypeInfo returns the TypeInfo for a field at a given path.
func (g *Generator) getFieldTypeInfo(typeInfo *analyze.TypeInfo, fieldPath string) *analyze.TypeInfo {
	if typeInfo == nil {
		return nil
	}

	parts := strings.Split(fieldPath, ".")
	current := typeInfo

	for _, part := range parts {
		// Remove slice notation if present
		part = strings.TrimSuffix(part, "[]")

		if current.Kind == analyze.TypeKindPointer && current.ElemType != nil {
			current = current.ElemType
		}

		if current.Kind != analyze.TypeKindStruct {
			return nil
		}

		current = g.findFieldInStruct(current, part)
		if current == nil {
			return nil
		}
	}

	return current
}

// findFieldInStruct finds a field by name in a struct type.
func (g *Generator) findFieldInStruct(structType *analyze.TypeInfo, fieldName string) *analyze.TypeInfo {
	for _, field := range structType.Fields {
		if field.Name == fieldName {
			return field.Type
		}
	}

	return nil
}

// getFieldType returns the TypeInfo for a field at a given path (alias for getFieldTypeInfo).
func (g *Generator) getFieldType(typeInfo *analyze.TypeInfo, fieldPath string) *analyze.TypeInfo {
	return g.getFieldTypeInfo(typeInfo, fieldPath)
}
