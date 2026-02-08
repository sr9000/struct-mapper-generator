package plan

import (
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

// preCreateVirtualTypes creates stub TypeInfo entries for all virtual target types
// before resolution begins. This ensures they're available for nested type detection.
func (r *Resolver) preCreateVirtualTypes() {
	if r.mappingDef == nil {
		return
	}

	for _, tm := range r.mappingDef.TypeMappings {
		if !tm.GenerateTarget {
			continue
		}

		// Check if source type exists
		sourceType := mapping.ResolveTypeID(tm.Source, r.graph)
		if sourceType == nil {
			continue
		}

		// Check if target type already exists (shouldn't for generate_target)
		targetID := parseTypeID(tm.Target)
		if r.graph.GetType(targetID) != nil {
			continue
		}

		// Create the virtual type (full structure will be populated in resolveTypeMapping)
		r.createVirtualTargetType(&tm, sourceType)
	}
}

// createVirtualTargetType creates a virtual TypeInfo for a generated target type.
// It synthesizes the target structure from the mapping definition.
func (r *Resolver) createVirtualTargetType(tm *mapping.TypeMapping, sourceType *analyze.TypeInfo) *analyze.TypeInfo {
	// Parse target type ID from string
	targetID := parseTypeID(tm.Target)

	// Create virtual type
	targetType := &analyze.TypeInfo{
		ID:          targetID,
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
		Fields:      []analyze.FieldInfo{},
	}

	// Build field index for source type
	sourceFields := make(map[string]*analyze.FieldInfo)
	for i := range sourceType.Fields {
		sourceFields[sourceType.Fields[i].Name] = &sourceType.Fields[i]
	}

	// Track which fields we've added
	addedFields := make(map[string]bool)

	// Helper to potentially remap a type if there's a generated target mapping for it
	remapType := func(srcType *analyze.TypeInfo) *analyze.TypeInfo {
		if srcType == nil {
			return srcType
		}

		return r.remapToGeneratedType(srcType)
	}

	// Process 121 mappings
	for sourcePath, targetPath := range tm.OneToOne {
		if addedFields[targetPath] {
			continue
		}

		if srcField, ok := sourceFields[sourcePath]; ok {
			targetType.Fields = append(targetType.Fields, analyze.FieldInfo{
				Name:     targetPath,
				Exported: true,
				Type:     remapType(srcField.Type),
				Index:    len(targetType.Fields),
			})
			addedFields[targetPath] = true
		}
	}

	// Process explicit field mappings
	for _, fm := range tm.Fields {
		for _, t := range fm.Target {
			targetName := t.Path
			if addedFields[targetName] {
				continue
			}
			// Try to infer type from source
			var fieldType *analyze.TypeInfo

			for _, s := range fm.Source {
				if srcField, ok := sourceFields[s.Path]; ok {
					fieldType = srcField.Type
					break
				}
			}

			if fieldType == nil {
				// Default to interface{} if we can't infer
				fieldType = &analyze.TypeInfo{
					ID:   analyze.TypeID{Name: "interface{}"},
					Kind: analyze.TypeKindBasic,
				}
			}

			targetType.Fields = append(targetType.Fields, analyze.FieldInfo{
				Name:     targetName,
				Exported: true,
				Type:     remapType(fieldType),
				Index:    len(targetType.Fields),
			})
			addedFields[targetName] = true
		}
	}

	// Process auto mappings
	for _, fm := range tm.Auto {
		for _, t := range fm.Target {
			targetName := t.Path
			if addedFields[targetName] {
				continue
			}
			// Try to infer type from source
			var fieldType *analyze.TypeInfo

			for _, s := range fm.Source {
				if srcField, ok := sourceFields[s.Path]; ok {
					fieldType = srcField.Type
					break
				}
			}

			if fieldType == nil {
				fieldType = &analyze.TypeInfo{
					ID:   analyze.TypeID{Name: "interface{}"},
					Kind: analyze.TypeKindBasic,
				}
			}

			targetType.Fields = append(targetType.Fields, analyze.FieldInfo{
				Name:     targetName,
				Exported: true,
				Type:     remapType(fieldType),
				Index:    len(targetType.Fields),
			})
			addedFields[targetName] = true
		}
	}

	// Add to graph for future lookups
	r.graph.Types[targetID] = targetType

	return targetType
}

// remapToGeneratedType checks if there's a generated target type mapping for the given source type
// and returns the corresponding target type reference. For slices/pointers, it recursively remaps the element type.
func (r *Resolver) remapToGeneratedType(srcType *analyze.TypeInfo) *analyze.TypeInfo {
	if srcType == nil || r.mappingDef == nil {
		return srcType
	}

	// Handle pointer types - recursively remap element
	if srcType.Kind == analyze.TypeKindPointer && srcType.ElemType != nil {
		remappedElem := r.remapToGeneratedType(srcType.ElemType)
		if remappedElem != srcType.ElemType {
			return &analyze.TypeInfo{
				Kind:        analyze.TypeKindPointer,
				ElemType:    remappedElem,
				IsGenerated: true,
			}
		}

		return srcType
	}

	// Handle slice types - recursively remap element
	if srcType.Kind == analyze.TypeKindSlice && srcType.ElemType != nil {
		remappedElem := r.remapToGeneratedType(srcType.ElemType)
		if remappedElem != srcType.ElemType {
			return &analyze.TypeInfo{
				Kind:        analyze.TypeKindSlice,
				ElemType:    remappedElem,
				IsGenerated: true,
			}
		}

		return srcType
	}

	// Handle array types - recursively remap element
	if srcType.Kind == analyze.TypeKindArray && srcType.ElemType != nil {
		remappedElem := r.remapToGeneratedType(srcType.ElemType)
		if remappedElem != srcType.ElemType {
			return &analyze.TypeInfo{
				Kind:        analyze.TypeKindArray,
				ElemType:    remappedElem,
				IsGenerated: true,
			}
		}

		return srcType
	}

	// For struct types, look for a matching generate_target mapping
	if srcType.Kind == analyze.TypeKindStruct && srcType.ID.Name != "" {
		for _, otherTM := range r.mappingDef.TypeMappings {
			if !otherTM.GenerateTarget {
				continue
			}
			// Check if this mapping's source matches our type
			otherSource := mapping.ResolveTypeID(otherTM.Source, r.graph)
			if otherSource != nil && otherSource.ID == srcType.ID {
				// Found a matching mapping - return a reference to the generated target type
				targetID := parseTypeID(otherTM.Target)
				// Check if we already have this type in the graph
				if existing := r.graph.GetType(targetID); existing != nil {
					return existing
				}
				// Create the virtual type and add it to the graph
				// This ensures all references use the same type object
				otherSourceType := mapping.ResolveTypeID(otherTM.Source, r.graph)
				if otherSourceType != nil {
					return r.createVirtualTargetType(&otherTM, otherSourceType)
				}
				// Fallback: create a stub type reference
				return &analyze.TypeInfo{
					ID:          targetID,
					Kind:        analyze.TypeKindStruct,
					IsGenerated: true,
				}
			}
		}
	}

	return srcType
}

// parseTypeID parses a type ID string into TypeID struct.
func parseTypeID(typeIDStr string) analyze.TypeID {
	// Handle name-only case
	if !strings.Contains(typeIDStr, ".") {
		return analyze.TypeID{Name: typeIDStr}
	}

	lastDot := strings.LastIndex(typeIDStr, ".")
	if lastDot < 0 {
		return analyze.TypeID{Name: typeIDStr}
	}

	return analyze.TypeID{
		PkgPath: typeIDStr[:lastDot],
		Name:    typeIDStr[lastDot+1:],
	}
}
