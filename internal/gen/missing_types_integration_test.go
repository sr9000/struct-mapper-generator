package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/plan"
)

func TestIntegration_VirtualExample(t *testing.T) {
	// This test relies on the existing example in examples/virtual
	rootDir, err := filepath.Abs("../../")
	require.NoError(t, err)

	virtualDir := filepath.Join(rootDir, "examples", "virtual")
	mappingFile := filepath.Join(virtualDir, "stages", "stage1_mapping.yaml")

	// Check if required files exist
	if _, err := os.Stat(virtualDir); os.IsNotExist(err) {
		t.Skip("examples/virtual directory not found, skipping integration test")
	}

	if _, err := os.Stat(mappingFile); os.IsNotExist(err) {
		t.Skip("examples/virtual mapping file not found, skipping integration test")
	}

	// 1. Load Packages
	analyzer := analyze.NewAnalyzer()
	// loading "caster-generator/examples/virtual"
	graph, err := analyzer.LoadPackages("caster-generator/examples/virtual")
	require.NoError(t, err)

	// 2. Load Mapping
	mappingDef, err := mapping.LoadFile(mappingFile)
	require.NoError(t, err)

	// 3. Plan
	config := plan.DefaultConfig()
	resolver := plan.NewResolver(graph, mappingDef, config)
	resolvedPlan, err := resolver.Resolve()
	require.NoError(t, err)

	// 4. Generate
	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(resolvedPlan)
	require.NoError(t, err)

	// 5. Assertions
	missingTypesFileFound := false
	pkgName := "virtual" // package name in examples/virtual

	for _, f := range files {
		// Log filenames for debugging
		t.Logf("Generated file: %s", f.Filename)

		if strings.Contains(f.Filename, "missing_types.go") {
			missingTypesFileFound = true
			content := string(f.Content)

			// Verify package declaration
			assert.Contains(t, content, "package "+pkgName)

			// Verify structs are generated
			assert.Contains(t, content, "type Target struct")
			assert.Contains(t, content, "type TargetItem struct")

			// Verify fields
			assert.Regexp(t, `ID\s+string`, content)
			assert.Regexp(t, `Items\s+\[\]\*TargetItem`, content) // No package prefix for same-package type

			// Verify we are not self-referencing package
			assert.NotContains(t, content, pkgName+".TargetItem")
		} else if strings.HasSuffix(f.Filename, ".go") {
			// Caster file checking
			content := string(f.Content)

			// Should import the virtual package
			assert.Contains(t, content, `"caster-generator/examples/virtual"`)

			// Should return virtual.Target
			assert.Contains(t, content, pkgName+".Target")
		}
	}

	assert.True(t, missingTypesFileFound, "missing_types.go should have been generated")
}

func TestIntegration_MixedGeneratedAndExisting(t *testing.T) {
	// Source1 -> ExistingTarget (not generated)
	// Source2 -> GeneratedTarget (generated)
	src1 := &analyze.TypeInfo{ID: analyze.TypeID{Name: "Source1"}, Kind: analyze.TypeKindStruct}
	tgt1 := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "pkg/existing", Name: "ExistingTarget"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: false,
	}

	src2 := &analyze.TypeInfo{ID: analyze.TypeID{Name: "Source2"}, Kind: analyze.TypeKindStruct}
	tgt2 := &analyze.TypeInfo{
		ID:          analyze.TypeID{PkgPath: "pkg/existing", Name: "GeneratedTarget"},
		Kind:        analyze.TypeKindStruct,
		IsGenerated: true,
	} // Note: Same package "pkg/existing"

	graph := &analyze.TypeGraph{
		Packages: map[string]*analyze.PackageInfo{
			"pkg/existing": {Name: "existing", Dir: "/abs/path/pkg/existing"},
		},
	}

	p := &plan.ResolvedMappingPlan{
		TypePairs: []plan.ResolvedTypePair{
			{SourceType: src1, TargetType: tgt1, IsGeneratedTarget: false},
			{SourceType: src2, TargetType: tgt2, IsGeneratedTarget: true},
		},
		TypeGraph: graph,
	}

	gen := NewGenerator(DefaultGeneratorConfig())
	files, err := gen.Generate(p)
	require.NoError(t, err)

	missingTypeFileCount := 0

	for _, f := range files {
		if strings.Contains(f.Filename, "missing_types.go") {
			missingTypeFileCount++
			content := string(f.Content)

			// Should contain GeneratedTarget
			assert.Contains(t, content, "type GeneratedTarget struct")

			// Should NOT contain ExistingTarget
			assert.NotContains(t, content, "type ExistingTarget struct")
		}
	}

	assert.Equal(t, 1, missingTypeFileCount)
}
