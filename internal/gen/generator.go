package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"caster-generator/internal/analyze"
	"caster-generator/internal/plan"
)

// GeneratorConfig holds configuration for code generation.
type GeneratorConfig struct {
	// PackageName is the name of the generated package.
	PackageName string
	// OutputDir is the directory where generated files are written.
	OutputDir string
	// GenerateComments enables generation of explanatory comments.
	GenerateComments bool
	// IncludeUnmappedTODOs generates TODO comments for unmapped fields.
	IncludeUnmappedTODOs bool
	// DeclaredTransforms is a set of transform names declared in the mapping file.
	// Transforms in this set won't have stubs generated.
	DeclaredTransforms map[string]bool
}

// DefaultGeneratorConfig returns the default generator configuration.
func DefaultGeneratorConfig() GeneratorConfig {
	return GeneratorConfig{
		PackageName:          "casters",
		OutputDir:            "./generated",
		GenerateComments:     true,
		IncludeUnmappedTODOs: true,
	}
}

// Generator generates Go code from a resolved mapping plan.
type Generator struct {
	config GeneratorConfig
	graph  *analyze.TypeGraph
	// missingTransforms stores info about missing transforms across all files.
	// Key is the function name.
	missingTransforms map[string]MissingTransformInfo

	// missingTypes stores info about missing target types that need to be generated
	// in their respective packages.
	// Key is the directory path.
	missingTypes map[string][]MissingTypeInfo

	// contextPkgPath is the package path currently being generated into.
	// Used to suppress package prefixes for types in the same package.
	contextPkgPath string
}

// MissingTransformInfo represents a missing transform function info.
// Used for internal deduplication.
type MissingTransformInfo struct {
	Name       string
	Args       []*analyze.TypeInfo
	ReturnType *analyze.TypeInfo
}

// MissingTypeInfo represents a missing type definition.
type MissingTypeInfo struct {
	PkgName   string
	StructDef string
	Imports   []importSpec
}

// MissingTransform represents a missing transform function that needs stub generation.
type MissingTransform struct {
	Name       string
	Args       []string
	ReturnType string
}

// NewGenerator creates a new Generator with the given configuration.
func NewGenerator(config GeneratorConfig) *Generator {
	return &Generator{config: config}
}

// GeneratedFile represents a generated Go source file.
type GeneratedFile struct {
	// Filename is the name of the file (e.g., "store_order_to_warehouse_order.go").
	Filename string
	// Content is the formatted Go source code.
	Content []byte
}

// Generate generates Go code from a ResolvedMappingPlan.
// Returns a list of generated files.
func (g *Generator) Generate(p *plan.ResolvedMappingPlan) ([]GeneratedFile, error) {
	g.graph = p.TypeGraph

	var files []GeneratedFile

	// Reset missing transforms for this run
	g.missingTransforms = make(map[string]MissingTransformInfo)
	g.missingTypes = make(map[string][]MissingTypeInfo)

	for _, pair := range p.TypePairs {
		file, err := g.generateTypePair(&pair)
		if err != nil {
			return nil, fmt.Errorf("generating %s->%s: %w",
				pair.SourceType.ID, pair.TargetType.ID, err)
		}

		files = append(files, *file)
	}

	// Generate missing transforms file if needed
	if len(g.missingTransforms) > 0 {
		file, err := g.generateMissingTransformsFile()
		if err != nil {
			return nil, fmt.Errorf("generating missing transforms: %w", err)
		}

		files = append(files, *file)
	}

	// Generate missing types files
	if len(g.missingTypes) > 0 {
		missingFiles, err := g.generateMissingTypesFiles()
		if err != nil {
			return nil, fmt.Errorf("generating missing types: %w", err)
		}

		files = append(files, missingFiles...)
	}

	return files, nil
}

// generateTypePair generates code for a single type pair.
func (g *Generator) generateTypePair(pair *plan.ResolvedTypePair) (*GeneratedFile, error) {
	data := g.buildTemplateData(pair)

	var buf bytes.Buffer
	if err := casterTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Best-effort: write unformatted code to a sidecar file to aid debugging.
		// This is intentionally non-fatal for the write attempt.
		if g.config.OutputDir != "" {
			_ = writeDebugUnformatted(g.config.OutputDir, data.Filename, buf.Bytes())
		}
		// Return unformatted code for debugging
		return &GeneratedFile{
			Filename: data.Filename,
			Content:  buf.Bytes(),
		}, fmt.Errorf("formatting code: %w (unformatted code returned)", err)
	}

	return &GeneratedFile{
		Filename: data.Filename,
		Content:  formatted,
	}, nil
}

// generateMissingTransformsFile generates a shared file for missing transforms.
func (g *Generator) generateMissingTransformsFile() (*GeneratedFile, error) {
	data := &templateData{
		PackageName: g.config.PackageName,
		Filename:    "missing_transforms.go",
	}

	imports := make(map[string]importSpec)

	// Convert g.missingTransforms to slice for template
	var missing []MissingTransform

	// Sorted iteration to ensure deterministic output
	var keys []string
	for k := range g.missingTransforms {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, name := range keys {
		info := g.missingTransforms[name]

		var argTypes []string
		for _, argInfo := range info.Args {
			argTypes = append(argTypes, g.typeRefString(argInfo, imports))
		}

		returnType := g.typeRefString(info.ReturnType, imports)

		missing = append(missing, MissingTransform{
			Name:       info.Name,
			Args:       argTypes,
			ReturnType: returnType,
		})
	}

	data.MissingTransforms = missing

	// Convert imports map to sorted slice
	for _, imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	sort.Slice(data.Imports, func(i, j int) bool {
		return data.Imports[i].Path < data.Imports[j].Path
	})

	var buf bytes.Buffer
	if err := missingTransformsTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		if g.config.OutputDir != "" {
			_ = writeDebugUnformatted(g.config.OutputDir, data.Filename, buf.Bytes())
		}

		return &GeneratedFile{
			Filename: data.Filename,
			Content:  buf.Bytes(),
		}, fmt.Errorf("formatting code: %w", err)
	}

	return &GeneratedFile{
		Filename: data.Filename,
		Content:  formatted,
	}, nil
}

// generateMissingTypesFiles generates missing_types.go files in respective directories.
func (g *Generator) generateMissingTypesFiles() ([]GeneratedFile, error) {
	var files []GeneratedFile

	for dir, infos := range g.missingTypes {
		if len(infos) == 0 {
			continue
		}

		// Group import specs
		imports := make(map[string]importSpec)

		var structDefs []string

		pkgName := infos[0].PkgName

		for _, info := range infos {
			structDefs = append(structDefs, info.StructDef)
			for _, imp := range info.Imports {
				// Don't import the package we are generating code in
				if imp.Path == "" {
					continue
				}
				// We need to check if the import path matches the package path of THIS file
				// But we don't have easy access to THIS package path, only dir.
				// However, regular Go imports rules apply.
				// If a struct refers to "virtual.TargetItem" and we are IN "virtual", we should NOT import "virtual".

				// Checking if import path ends with pkgName is a weak heuristic, but
				// checking if known PkgPath maps to 'dir' is better.
				if pkgInfo, ok := g.graph.Packages[imp.Path]; ok && pkgInfo.Dir == dir {
					continue
				}

				imports[imp.Path] = imp
			}
		}

		var sortedImports []importSpec
		for _, imp := range imports {
			sortedImports = append(sortedImports, imp)
		}

		sort.Slice(sortedImports, func(i, j int) bool {
			return sortedImports[i].Path < sortedImports[j].Path
		})

		data := &MissingTypesTemplateData{
			PackageName: pkgName,
			Imports:     sortedImports,
			StructDefs:  structDefs,
		}

		var buf bytes.Buffer
		if err := missingTypesTemplate.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("executing missing types template for %s: %w", dir, err)
		}

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			if g.config.OutputDir != "" {
				_ = writeDebugUnformatted(g.config.OutputDir, "missing_types_debug.go", buf.Bytes())
			}

			return nil, fmt.Errorf("formatting missing types code for %s: %w", dir, err)
		}

		relPath, relErr := filepath.Rel(g.config.OutputDir, dir)
		if relErr != nil {
			// Fallback to absolute path and hope caller handles it or Writer is updated
			relPath = filepath.Join(dir, "missing_types.go")
		} else {
			relPath = filepath.Join(relPath, "missing_types.go")
		}

		files = append(files, GeneratedFile{
			Filename: relPath, // This will be joined with OutputDir
			Content:  formatted,
		})
	}

	return files, nil
}

// MissingTypesTemplateData holds data for the missing types template.
type MissingTypesTemplateData struct {
	PackageName string
	Imports     []importSpec
	StructDefs  []string
}

// addMissingType records a struct definition that needs to be generated in a specific package.
func (g *Generator) addMissingType(dir, pkgName, structDef string, imports []importSpec) {
	if g.missingTypes == nil {
		g.missingTypes = make(map[string][]MissingTypeInfo)
	}

	g.missingTypes[dir] = append(g.missingTypes[dir], MissingTypeInfo{
		PkgName:   pkgName,
		StructDef: structDef,
		Imports:   imports,
	})
}

// Helper functions for naming

func (g *Generator) filename(pair *plan.ResolvedTypePair) string {
	src := strings.ToLower(pair.SourceType.ID.Name)
	tgt := strings.ToLower(pair.TargetType.ID.Name)
	srcPkg := g.getPkgName(pair.SourceType.ID.PkgPath)
	tgtPkg := g.getPkgName(pair.TargetType.ID.PkgPath)

	// For generated targets with no package path, use the output package name
	if tgtPkg == "" && pair.IsGeneratedTarget {
		tgtPkg = g.config.PackageName
	}

	return fmt.Sprintf("%s_%s_to_%s_%s.go", srcPkg, src, tgtPkg, tgt)
}

func (g *Generator) functionName(pair *plan.ResolvedTypePair) string {
	srcPkg := g.capitalize(g.getPkgName(pair.SourceType.ID.PkgPath))
	tgtPkg := g.capitalize(g.getPkgName(pair.TargetType.ID.PkgPath))

	// For generated targets with no package path, use the output package name
	if tgtPkg == "" && pair.IsGeneratedTarget {
		tgtPkg = g.capitalize(g.config.PackageName)
	}

	return fmt.Sprintf("%s%sTo%s%s",
		srcPkg, pair.SourceType.ID.Name,
		tgtPkg, pair.TargetType.ID.Name)
}

func (g *Generator) nestedFunctionName(src, tgt *analyze.TypeInfo) string {
	srcPkg := g.capitalize(g.getPkgName(src.ID.PkgPath))
	tgtPkg := g.capitalize(g.getPkgName(tgt.ID.PkgPath))

	// For generated targets with no package path, use the output package name
	if tgtPkg == "" && tgt.IsGenerated {
		tgtPkg = g.capitalize(g.config.PackageName)
	}

	return fmt.Sprintf("%s%sTo%s%s", srcPkg, src.ID.Name, tgtPkg, tgt.ID.Name)
}

func (g *Generator) capitalize(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

// Templates

var casterTemplate = template.Must(template.New("caster").Parse(`// Code generated by caster-generator. DO NOT EDIT.

package {{.PackageName}}

{{if .Imports}}
import (
{{range .Imports}}	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{end}})
{{end}}
{{if .StructDef}}
// Generated target type
{{.StructDef}}
{{end}}
// {{.FunctionName}} converts {{.SourceType}} to {{.TargetType}}.
func {{.FunctionName}}(in {{.SourceType}}{{range .ExtraArgs}}, {{.Name}} {{.Type}}{{end}}) {{.TargetType}} {
	out := {{.TargetType}}{}
{{range .Assignments}}
{{if .Comment}}	// {{.Comment}}
{{end}}{{if .IsSlice}}	{{.SliceBody}}
{{else if .IsMap}}	{{.MapBody}}
{{else if .NeedsNilCheck}}	if ({{if .NilCheckExpr}}{{.NilCheckExpr}}{{else}}{{.SourceExpr}}{{end}}) != nil {
		{{.TargetField}} = {{.SourceExpr}}
	} else {
		{{.TargetField}} = {{.NilDefault}}
	}
{{else}}	{{.TargetField}} = {{.SourceExpr}}
{{end}}{{end}}
{{if .UnmappedTODOs}}
{{range .UnmappedTODOs}}	// {{.}}
{{end}}{{end}}
	return out
}

{{if .MissingTransforms}}
// Missing transforms. Ideally, these should be implemented in your project or defined as transforms in map.yaml
{{range .MissingTransforms}}func {{.Name}}({{range $index, $arg := .Args}}{{if $index}}, {{end}}v{{$index}} {{$arg}}{{end}}) {{.ReturnType}} {
	panic("transform {{.Name}} not implemented")
}

{{end}}{{end}}
`))

var missingTransformsTemplate = template.Must(template.New("missing").Parse(`// Code generated by caster-generator. DO NOT EDIT.

package {{.PackageName}}

{{if .Imports}}
import (
{{range .Imports}}	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{end}})
{{end}}

// Missing transforms. Ideally, these should be implemented in your project or defined as transforms in map.yaml

{{range .MissingTransforms}}func {{.Name}}({{range $index, $arg := .Args}}{{if $index}}, {{end}}v{{$index}} {{$arg}}{{end}}) {{.ReturnType}} {
	panic("transform {{.Name}} not implemented")
}

{{end}}
`))

var missingTypesTemplate = template.Must(
	template.New("missing_types").
		Parse(`// Code generated by caster-generator. DO NOT EDIT.

package {{.PackageName}}

{{if .Imports}}
import (
{{range .Imports}}	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{end}})
{{end}}

{{range .StructDefs}}
{{.}}
{{end}}
`))
