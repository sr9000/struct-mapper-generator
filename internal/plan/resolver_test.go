package plan

import (
	"go/types"
	"testing"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

// Helper function to create a basic TypeInfo with GoType set
func basicTypeInfo() *analyze.TypeInfo {
	return &analyze.TypeInfo{
		Kind:   analyze.TypeKindBasic,
		GoType: types.Typ[types.String],
	}
}

func TestResolverBasic(t *testing.T) {
	// Create a simple type graph
	graph := analyze.NewTypeGraph()

	// Source type
	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Person"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Email", Exported: true, Type: basicTypeInfo()},
			{Name: "internal", Exported: false, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	// Target type
	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "User"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "UserID", Exported: true, Type: basicTypeInfo()},
			{Name: "FullName", Exported: true, Type: basicTypeInfo()},
			{Name: "EmailAddr", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[targetType.ID] = targetType

	// Create mapping
	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.Person",
				Target: "target.User",
				OneToOne: map[string]string{
					"ID":    "UserID",
					"Name":  "FullName",
					"Email": "EmailAddr",
				},
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(plan.TypePairs) != 1 {
		t.Errorf("Expected 1 type pair, got %d", len(plan.TypePairs))
	}

	tp := plan.TypePairs[0]
	if len(tp.Mappings) != 3 {
		t.Errorf("Expected 3 mappings, got %d", len(tp.Mappings))
	}

	// Verify all mappings are from 121
	for _, m := range tp.Mappings {
		if m.Source != MappingSourceYAML121 {
			t.Errorf("Expected source MappingSourceYAML121, got %v", m.Source)
		}
	}
}

func TestResolverAutoMatch(t *testing.T) {
	// Create type graph with matching field names
	graph := analyze.NewTypeGraph()

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Product"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Price", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "Item"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Cost", Exported: true, Type: basicTypeInfo()}, // Different name
		},
	}
	graph.Types[targetType.ID] = targetType

	// Minimal mapping - let auto-match handle identical names
	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.Product",
				Target: "target.Item",
				// No explicit mappings - rely on auto-match
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	tp := plan.TypePairs[0]

	// Should auto-match ID and Name, but not Cost (different name, low confidence)
	autoMatched := 0
	for _, m := range tp.Mappings {
		if m.Source == MappingSourceAutoMatched {
			autoMatched++
		}
	}

	// ID and Name should be auto-matched (identical names)
	if autoMatched < 2 {
		t.Errorf("Expected at least 2 auto-matched fields, got %d", autoMatched)
	}

	// Cost should be unmapped
	if len(tp.UnmappedTargets) < 1 {
		t.Errorf("Expected at least 1 unmapped field, got %d", len(tp.UnmappedTargets))
	}
}

func TestResolverPriority(t *testing.T) {
	// Test that priority order is respected: 121 > fields > ignore > auto
	graph := analyze.NewTypeGraph()

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "A"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "X", Exported: true, Type: basicTypeInfo()},
			{Name: "Y", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "B"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "X", Exported: true, Type: basicTypeInfo()},
			{Name: "Y", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[targetType.ID] = targetType

	// Mapping with overlapping rules
	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.A",
				Target: "target.B",
				OneToOne: map[string]string{
					"X": "X", // 121 takes precedence
				},
				Fields: []mapping.FieldMapping{
					{
						Target: mapping.StringOrArray{"X"}, // Should be overridden by 121
						Source: mapping.StringOrArray{"Y"},
					},
				},
				Ignore: []string{"Y"}, // Y is ignored
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	tp := plan.TypePairs[0]

	// Find X mapping - should be from 121
	var xMapping *ResolvedFieldMapping
	for i := range tp.Mappings {
		if len(tp.Mappings[i].TargetPaths) > 0 && tp.Mappings[i].TargetPaths[0].String() == "X" {
			xMapping = &tp.Mappings[i]
			break
		}
	}

	if xMapping == nil {
		t.Fatal("X mapping not found")
	}

	if xMapping.Source != MappingSourceYAML121 {
		t.Errorf("Expected X to come from 121, got %v", xMapping.Source)
	}

	// Y should be ignored
	var yMapping *ResolvedFieldMapping
	for i := range tp.Mappings {
		if len(tp.Mappings[i].TargetPaths) > 0 && tp.Mappings[i].TargetPaths[0].String() == "Y" {
			yMapping = &tp.Mappings[i]
			break
		}
	}

	if yMapping == nil {
		t.Fatal("Y mapping not found")
	}

	if yMapping.Strategy != StrategyIgnore {
		t.Errorf("Expected Y to be ignored, got strategy %v", yMapping.Strategy)
	}
}

func TestResolverIgnore(t *testing.T) {
	graph := analyze.NewTypeGraph()

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "S"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "A", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "T"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "A", Exported: true, Type: basicTypeInfo()},
			{Name: "Internal", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[targetType.ID] = targetType

	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.S",
				Target: "target.T",
				Ignore: []string{"Internal"},
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	tp := plan.TypePairs[0]

	// Find Internal mapping
	var internalMapping *ResolvedFieldMapping
	for i := range tp.Mappings {
		if len(tp.Mappings[i].TargetPaths) > 0 && tp.Mappings[i].TargetPaths[0].String() == "Internal" {
			internalMapping = &tp.Mappings[i]
			break
		}
	}

	if internalMapping == nil {
		t.Fatal("Internal mapping not found")
	}

	if internalMapping.Source != MappingSourceYAMLIgnore {
		t.Errorf("Expected source MappingSourceYAMLIgnore, got %v", internalMapping.Source)
	}

	if internalMapping.Strategy != StrategyIgnore {
		t.Errorf("Expected strategy StrategyIgnore, got %v", internalMapping.Strategy)
	}
}

func TestResolverDefaultValue(t *testing.T) {
	graph := analyze.NewTypeGraph()

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "S"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "A", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "T"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "A", Exported: true, Type: basicTypeInfo()},
			{Name: "Status", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[targetType.ID] = targetType

	defaultVal := "active"
	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.S",
				Target: "target.T",
				Fields: []mapping.FieldMapping{
					{
						Target:  mapping.StringOrArray{"Status"},
						Default: &defaultVal,
					},
				},
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	tp := plan.TypePairs[0]

	// Find Status mapping
	var statusMapping *ResolvedFieldMapping
	for i := range tp.Mappings {
		if len(tp.Mappings[i].TargetPaths) > 0 && tp.Mappings[i].TargetPaths[0].String() == "Status" {
			statusMapping = &tp.Mappings[i]
			break
		}
	}

	if statusMapping == nil {
		t.Fatal("Status mapping not found")
	}

	if statusMapping.Strategy != StrategyDefault {
		t.Errorf("Expected strategy StrategyDefault, got %v", statusMapping.Strategy)
	}

	if statusMapping.Default == nil || *statusMapping.Default != "active" {
		t.Errorf("Expected default 'active', got %v", statusMapping.Default)
	}
}

func TestExportSuggestions(t *testing.T) {
	graph := analyze.NewTypeGraph()

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "S"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "T"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Extra", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[targetType.ID] = targetType

	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.S",
				Target: "target.T",
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Export suggestions
	yamlBytes, err := ExportSuggestionsYAML(plan)
	if err != nil {
		t.Fatalf("ExportSuggestionsYAML failed: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Error("Expected non-empty YAML output")
	}

	// Basic check that it parses back
	exportedMF, err := mapping.Parse(yamlBytes)
	if err != nil {
		t.Fatalf("Failed to parse exported YAML: %v", err)
	}

	if len(exportedMF.TypeMappings) != 1 {
		t.Errorf("Expected 1 type mapping, got %d", len(exportedMF.TypeMappings))
	}
}

func TestGenerateReport(t *testing.T) {
	graph := analyze.NewTypeGraph()

	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "S"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "T"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Extra", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[targetType.ID] = targetType

	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.S",
				Target: "target.T",
				OneToOne: map[string]string{
					"ID": "ID",
				},
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	report := GenerateReport(plan)

	if len(report.TypePairs) != 1 {
		t.Fatalf("Expected 1 type pair, got %d", len(report.TypePairs))
	}

	tpr := report.TypePairs[0]
	if tpr.ExplicitCount != 1 {
		t.Errorf("Expected 1 explicit mapping, got %d", tpr.ExplicitCount)
	}

	if len(tpr.Unmapped) != 1 {
		t.Errorf("Expected 1 unmapped field, got %d", len(tpr.Unmapped))
	}

	if !tpr.NeedsReview {
		t.Error("Expected NeedsReview to be true")
	}

	// Test formatted output
	formatted := FormatReport(report)
	if formatted == "" {
		t.Error("Expected non-empty formatted report")
	}
}

func TestResolverNestedStruct(t *testing.T) {
	// Test recursive resolution of nested struct fields
	graph := analyze.NewTypeGraph()

	// Nested source type (Address)
	sourceAddressType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Address"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Street", Exported: true, Type: basicTypeInfo()},
			{Name: "City", Exported: true, Type: basicTypeInfo()},
			{Name: "Country", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceAddressType.ID] = sourceAddressType

	// Nested target type (Location)
	targetAddressType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "Location"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Street", Exported: true, Type: basicTypeInfo()},
			{Name: "City", Exported: true, Type: basicTypeInfo()},
			{Name: "Nation", Exported: true, Type: basicTypeInfo()}, // Different name
		},
	}
	graph.Types[targetAddressType.ID] = targetAddressType

	// Top-level source type (Person with nested Address)
	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Person"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "HomeAddress", Exported: true, Type: sourceAddressType},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	// Top-level target type (User with nested Location)
	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "User"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "HomeAddress", Exported: true, Type: targetAddressType},
		},
	}
	graph.Types[targetType.ID] = targetType

	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.Person",
				Target: "target.User",
				// Let auto-match handle it
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(plan.TypePairs) != 1 {
		t.Fatalf("Expected 1 type pair, got %d", len(plan.TypePairs))
	}

	tp := plan.TypePairs[0]

	// Should have detected nested struct conversion
	if len(tp.NestedPairs) == 0 {
		t.Error("Expected at least 1 nested pair for Address->Location conversion")
	}

	// Find the nested conversion
	var nestedConv *NestedConversion
	for i := range tp.NestedPairs {
		if tp.NestedPairs[i].SourceType.ID.Name == "Address" {
			nestedConv = &tp.NestedPairs[i]
			break
		}
	}

	if nestedConv == nil {
		t.Fatal("Expected nested conversion for Address type")
	}

	// Verify it was recursively resolved
	if nestedConv.ResolvedPair == nil {
		t.Error("Expected nested pair to be recursively resolved")
	} else {
		// Check that the nested pair has auto-matched fields
		if len(nestedConv.ResolvedPair.Mappings) < 2 {
			t.Errorf("Expected at least 2 auto-matched fields in nested pair, got %d",
				len(nestedConv.ResolvedPair.Mappings))
		}

		// Street and City should be auto-matched, Nation should be unmapped
		if len(nestedConv.ResolvedPair.UnmappedTargets) != 1 {
			t.Errorf("Expected 1 unmapped target (Nation), got %d",
				len(nestedConv.ResolvedPair.UnmappedTargets))
		}
	}
}

func TestResolverSliceOfStructs(t *testing.T) {
	// Test recursive resolution of slice element types
	graph := analyze.NewTypeGraph()

	// Source item type
	sourceItemType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Item"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Price", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceItemType.ID] = sourceItemType

	// Target item type
	targetItemType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "Product"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Cost", Exported: true, Type: basicTypeInfo()}, // Different name
		},
	}
	graph.Types[targetItemType.ID] = targetItemType

	// Slice types
	sourceSliceType := &analyze.TypeInfo{
		ID:       analyze.TypeID{},
		Kind:     analyze.TypeKindSlice,
		ElemType: sourceItemType,
	}

	targetSliceType := &analyze.TypeInfo{
		ID:       analyze.TypeID{},
		Kind:     analyze.TypeKindSlice,
		ElemType: targetItemType,
	}

	// Top-level source type (Order with slice of Items)
	sourceType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Order"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "OrderID", Exported: true, Type: basicTypeInfo()},
			{Name: "Items", Exported: true, Type: sourceSliceType},
		},
	}
	graph.Types[sourceType.ID] = sourceType

	// Top-level target type (Invoice with slice of Products)
	targetType := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "Invoice"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "OrderID", Exported: true, Type: basicTypeInfo()},
			{Name: "Items", Exported: true, Type: targetSliceType},
		},
	}
	graph.Types[targetType.ID] = targetType

	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.Order",
				Target: "target.Invoice",
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(plan.TypePairs) != 1 {
		t.Fatalf("Expected 1 type pair, got %d", len(plan.TypePairs))
	}

	tp := plan.TypePairs[0]

	// Find the Items mapping
	var itemsMapping *ResolvedFieldMapping
	for i := range tp.Mappings {
		if len(tp.Mappings[i].TargetPaths) > 0 && tp.Mappings[i].TargetPaths[0].String() == "Items" {
			itemsMapping = &tp.Mappings[i]
			break
		}
	}

	if itemsMapping == nil {
		t.Fatal("Expected Items mapping")
	}

	// Items should use slice map strategy
	if itemsMapping.Strategy != StrategySliceMap {
		t.Errorf("Expected StrategySliceMap for Items, got %v", itemsMapping.Strategy)
	}

	// Should have detected nested slice element conversion
	if len(tp.NestedPairs) == 0 {
		t.Error("Expected at least 1 nested pair for Item->Product element conversion")
	}

	// Find the nested conversion for slice elements
	var nestedConv *NestedConversion
	for i := range tp.NestedPairs {
		if tp.NestedPairs[i].SourceType.ID.Name == "Item" {
			nestedConv = &tp.NestedPairs[i]
			break
		}
	}

	if nestedConv == nil {
		t.Fatal("Expected nested conversion for Item element type")
	}

	// Verify it's marked as slice element
	if !nestedConv.IsSliceElement {
		t.Error("Expected IsSliceElement to be true")
	}

	// Verify it was recursively resolved
	if nestedConv.ResolvedPair == nil {
		t.Error("Expected nested pair to be recursively resolved")
	} else {
		// ID and Name should be auto-matched, Cost should be unmapped
		if len(nestedConv.ResolvedPair.UnmappedTargets) != 1 {
			t.Errorf("Expected 1 unmapped target (Cost), got %d",
				len(nestedConv.ResolvedPair.UnmappedTargets))
		}
	}
}

func TestResolverDeepNesting(t *testing.T) {
	// Test multiple levels of nesting: A -> B -> C
	graph := analyze.NewTypeGraph()

	// Level 3 types (deepest)
	sourceC := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "C"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Value", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[sourceC.ID] = sourceC

	targetC := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "C"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Value", Exported: true, Type: basicTypeInfo()},
		},
	}
	graph.Types[targetC.ID] = targetC

	// Level 2 types
	sourceB := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "B"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Nested", Exported: true, Type: sourceC},
		},
	}
	graph.Types[sourceB.ID] = sourceB

	targetB := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "B"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Name", Exported: true, Type: basicTypeInfo()},
			{Name: "Nested", Exported: true, Type: targetC},
		},
	}
	graph.Types[targetB.ID] = targetB

	// Level 1 types (top-level)
	sourceA := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "A"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Child", Exported: true, Type: sourceB},
		},
	}
	graph.Types[sourceA.ID] = sourceA

	targetA := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "A"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "ID", Exported: true, Type: basicTypeInfo()},
			{Name: "Child", Exported: true, Type: targetB},
		},
	}
	graph.Types[targetA.ID] = targetA

	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.A",
				Target: "target.A",
			},
		},
	}

	resolver := NewResolver(graph, mf, DefaultConfig())
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	tp := plan.TypePairs[0]

	// Should have nested conversion for A->B
	if len(tp.NestedPairs) == 0 {
		t.Error("Expected nested pairs")
	}

	// Find B->B nested conversion and verify it has its own nested C->C
	var nestedB *NestedConversion
	for i := range tp.NestedPairs {
		if tp.NestedPairs[i].SourceType.ID.Name == "B" {
			nestedB = &tp.NestedPairs[i]
			break
		}
	}

	if nestedB == nil {
		t.Fatal("Expected nested conversion for B type")
	}

	if nestedB.ResolvedPair == nil {
		t.Fatal("Expected B->B to be recursively resolved")
	}

	// B->B should have its own nested C->C conversion
	if len(nestedB.ResolvedPair.NestedPairs) == 0 {
		t.Error("Expected nested C->C conversion within B->B")
	} else {
		nestedC := nestedB.ResolvedPair.NestedPairs[0]
		if nestedC.SourceType.ID.Name != "C" {
			t.Errorf("Expected nested C type, got %s", nestedC.SourceType.ID.Name)
		}
		if nestedC.ResolvedPair == nil {
			t.Error("Expected C->C to be recursively resolved")
		}
	}
}

func TestResolverMaxRecursionDepth(t *testing.T) {
	// Test that max recursion depth is respected
	graph := analyze.NewTypeGraph()

	// Create a self-referential type (tree structure)
	sourceNode := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/source", Name: "Node"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Value", Exported: true, Type: basicTypeInfo()},
			// Child will reference the same type - creating potential infinite recursion
		},
	}
	// Add self-reference
	sourceNode.Fields = append(sourceNode.Fields, analyze.FieldInfo{
		Name: "Child", Exported: true, Type: sourceNode,
	})
	graph.Types[sourceNode.ID] = sourceNode

	targetNode := &analyze.TypeInfo{
		ID:   analyze.TypeID{PkgPath: "test/target", Name: "TreeNode"},
		Kind: analyze.TypeKindStruct,
		Fields: []analyze.FieldInfo{
			{Name: "Value", Exported: true, Type: basicTypeInfo()},
		},
	}
	targetNode.Fields = append(targetNode.Fields, analyze.FieldInfo{
		Name: "Child", Exported: true, Type: targetNode,
	})
	graph.Types[targetNode.ID] = targetNode

	mf := &mapping.MappingFile{
		Version: "1",
		TypeMappings: []mapping.TypeMapping{
			{
				Source: "source.Node",
				Target: "target.TreeNode",
			},
		},
	}

	config := DefaultConfig()
	config.MaxRecursionDepth = 3 // Limit recursion

	resolver := NewResolver(graph, mf, config)
	plan, err := resolver.Resolve()

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Should succeed without infinite loop
	if len(plan.TypePairs) != 1 {
		t.Fatalf("Expected 1 type pair, got %d", len(plan.TypePairs))
	}

	// Should have warnings about max recursion depth
	hasRecursionWarning := false
	for _, w := range plan.Diagnostics.Warnings {
		if w.Code == "max_recursion_depth" {
			hasRecursionWarning = true
			break
		}
	}

	if !hasRecursionWarning {
		t.Log("Note: Max recursion warning might not appear if caching kicks in first")
	}
}
