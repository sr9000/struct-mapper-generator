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
