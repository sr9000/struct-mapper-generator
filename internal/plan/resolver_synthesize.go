package plan

import (
	"fmt"
	"go/types"
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

// synthesizeTargetType creates a virtual TypeInfo for a missing target type based on the mapping.
func (r *Resolver) synthesizeTargetType(tm *mapping.TypeMapping, sourceType *analyze.TypeInfo) (*analyze.TypeInfo, error) {
	// Parse package and name: "pkg.Name"
	// For V1 we accept simply "Name" (default package?) or "pkg.Name".
	// But generated types usually need a package.
	// We use the full string as name if no dot, or split.
	var typeName string
	var pkgPath string

	if idx := strings.LastIndex(tm.Target, "."); idx != -1 {
		pkgPath = tm.Target[:idx]
		typeName = tm.Target[idx+1:]
	} else {
		typeName = tm.Target
		// fallback pkgPath? Empty means same package or unknown.
	}

	// Create TypeInfo
	typeInfo := &analyze.TypeInfo{
		ID: analyze.TypeID{
			PkgPath: pkgPath,
			Name:    typeName,
		},
		Kind:        analyze.TypeKindStruct,
		Fields:      []analyze.FieldInfo{},
		IsGenerated: true,
	}

	fields := make(map[string]*analyze.FieldInfo) // Map to prevent duplicates
	// Helper to add field
	addField := func(name string, t *analyze.TypeInfo) {
		if _, exists := fields[name]; !exists {
			fields[name] = &analyze.FieldInfo{
				Name:     name,
				Exported: true, // Generated fields are exported
				Type:     t,
				// Index is irrelevant for now or set at end
			}
		}
	}

	// 1. Process OneToOne
	for srcField, tgtField := range tm.OneToOne {
		sFi := findField(sourceType, srcField)
		if sFi != nil {
			addField(tgtField, sFi.Type)
		} else {
			// If source field not found, we can't infer type easily.
			// Maybe error?
			return nil, fmt.Errorf("source field %q not found for generated target field %q", srcField, tgtField)
		}
	}

	// 2. Process Fields
	for _, fm := range tm.Fields {
		if len(fm.Target) == 0 {
			continue
		}

		// Use the first target path segment as the field name
		// Support simple mappings only for generation V1
		fullPath := fm.Target.First()
		if fullPath == "" {
			continue
		}

		// If path is "Address.Street", we are adding "Address" field to THIS type.
		// Its type should be a struct that has "Street".
		// For simplicity, we only handle top-level fields of the generated struct here.
		// If nested generation is needed, it would be recursive.
		fieldName := fullPath
		if idx := strings.Index(fullPath, "."); idx != -1 {
			fieldName = fullPath[:idx]
		}

		if _, exists := fields[fieldName]; exists {
			continue
		}

		var fType *analyze.TypeInfo

		// Determine type
		if fm.TargetType != "" {
			fType = r.resolveOrBasicType(fm.TargetType)
		} else {
			// Fallback to source type
			// deduceSourceType logic
			srcType := deduceSourceType(sourceType, fm.Source, r)
			if srcType != nil {
				fType = srcType
			} else {
				// Unknown type
				fType = &analyze.TypeInfo{Kind: analyze.TypeKindUnknown}
			}
		}

		addField(fieldName, fType)
	}

	// Convert map to slice
	fieldList := make([]analyze.FieldInfo, 0, len(fields))
	for _, fi := range fields {
		fieldList = append(fieldList, *fi)
	}
	// TODO: Sort fields?

	typeInfo.Fields = fieldList

	// Add to graph
	r.graph.Types[typeInfo.ID] = typeInfo

	return typeInfo, nil
}

// findField looks up a field in a TypeInfo by name.
func findField(t *analyze.TypeInfo, name string) *analyze.FieldInfo {
	if t.Kind != analyze.TypeKindStruct {
		return nil
	}
	for _, f := range t.Fields {
		if f.Name == name {
			return &f
		}
	}
	return nil
}

// deduceSourceType tries to find the type of the source field.
func deduceSourceType(root *analyze.TypeInfo, ref mapping.FieldRefArray, r *Resolver) *analyze.TypeInfo {
	if len(ref) == 0 {
		return nil
	}
	pathStr := ref.First() // simplified
	// Split path
	parts := strings.Split(pathStr, ".")

	// Helper to decode path
	path := mapping.FieldPath{
		Segments: make([]mapping.PathSegment, len(parts)),
	}
	for i, p := range parts {
		path.Segments[i] = mapping.PathSegment{Name: p}
	}

	// Reuse existing resolveFieldType?
	// resolveFieldType takes FieldPath which requires parsing logic.
	// But resolveFieldType verifies existence.
	return r.resolveFieldType(path, root)
}

// resolveOrBasicType returns TypeInfo for basic types or named types.
func (r *Resolver) resolveOrBasicType(name string) *analyze.TypeInfo {
	// check basic types
	switch name {
	case "string":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.String]}
	case "int":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Int]}
	case "int64":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Int64]}
	case "int32":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Int32]}
	case "float64":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Float64]}
	case "float32":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Float32]}
	case "bool":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Bool]}
	case "byte":
		return &analyze.TypeInfo{Kind: analyze.TypeKindBasic, GoType: types.Typ[types.Byte]}
		// Pointer types? "*int"
		// Handling generic syntax is hard here.
	}

	// Check if pointer
	if strings.HasPrefix(name, "*") {
		elem := r.resolveOrBasicType(name[1:])
		return &analyze.TypeInfo{
			Kind:        analyze.TypeKindPointer,
			ElemType:    elem,
			GoType:      nil, // Simplified, as we might lack GoType for elem
			IsGenerated: isGenerated(elem),
		}
	}

	// Check if slice
	if strings.HasPrefix(name, "[]") {
		elem := r.resolveOrBasicType(name[2:])
		return &analyze.TypeInfo{
			Kind:        analyze.TypeKindSlice,
			ElemType:    elem,
			GoType:      nil,
			IsGenerated: isGenerated(elem),
		}
	}

	// Try resolving as named type in graph
	t := mapping.ResolveTypeID(name, r.graph)
	if t != nil {
		return t
	}

	// Not found, assume it will be generated or external
	// For generated, we might find it if we already generated it.
	// If name is "pkg.Type", try constructing ID.
	// Otherwise fallback to unknown.
	return &analyze.TypeInfo{Kind: analyze.TypeKindUnknown} // Or named type stub?
}
