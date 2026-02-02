// Package main provides the CLI entrypoint for caster-generator.
//
// caster-generator is a semi-automated Go codegen tool that:
//   - Parses Go packages (AST + go/types) to understand structs and relationships
//   - Suggests field mappings best-effort
//   - Lets humans review + lock mappings via YAML
//   - Generates fast caster functions
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/diagnostic"
	"caster-generator/internal/gen"
	"caster-generator/internal/mapping"
	"caster-generator/internal/plan"
)

const (
	version = "0.1.0"
	usage   = `caster-generator - a semi-automated Go struct mapping codegen tool

Usage:
  caster-generator <command> [options]

Commands:
  analyze   Print discovered structs and fields from packages (debug)
  suggest   Generate a suggested YAML mapping for a type pair
  gen       Generate casters using YAML mapping
  check     Validate YAML against current code; fail on drift

Global Options:
  -help     Show help for a command
  -version  Print version information

Examples:
  # Analyze packages to see available types
  caster-generator analyze -pkg ./store -pkg ./warehouse

  # Generate suggested mapping YAML for a type pair
  caster-generator suggest -from store.Order -to warehouse.Order -out mapping.yaml

  # Generate casters from YAML mapping
  caster-generator gen -mapping mapping.yaml -out ./generated

  # Validate existing mapping against code
  caster-generator check -mapping mapping.yaml

Run 'caster-generator <command> -help' for more information on a command.
`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "-help", "--help", "help":
		fmt.Print(usage)
		os.Exit(0)
	case "-version", "--version", "version":
		fmt.Printf("caster-generator version %s\n", version)
		os.Exit(0)
	case "analyze":
		runAnalyze(os.Args[2:])
	case "suggest":
		runSuggest(os.Args[2:])
	case "gen":
		runGen(os.Args[2:])
	case "check":
		runCheck(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		fmt.Print(usage)
		os.Exit(1)
	}
}

// StringSliceFlag is a flag that can be specified multiple times.
type StringSliceFlag []string

func (s *StringSliceFlag) String() string {
	return strings.Join(*s, ", ")
}

func (s *StringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// runAnalyze implements the 'analyze' command.
func runAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: caster-generator analyze [options]

Print discovered structs and fields from packages (debug).

Options:
`)
		fs.PrintDefaults()
	}

	var packages StringSliceFlag

	fs.Var(&packages, "pkg", "Package path to analyze (can be specified multiple times, default: ./...)")
	verbose := fs.Bool("verbose", false, "Show detailed field information including tags")
	typeFilter := fs.String("type", "", "Filter to show only a specific type")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Default to current directory if no packages specified
	if len(packages) == 0 {
		packages = append(packages, "./...")
	}

	// Load packages
	analyzer := analyze.NewAnalyzer()

	graph, err := analyzer.LoadPackages(packages...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading packages: %v\n", err)
		os.Exit(1)
	}

	// Print discovered types
	stringer := analyze.NewTypeStringer()

	for _, pkgInfo := range graph.Packages {
		fmt.Printf("\nPackage: %s (%s)\n", pkgInfo.Name, pkgInfo.Path)
		fmt.Println(strings.Repeat("-", 60))

		for _, typeID := range pkgInfo.Types {
			typeInfo := graph.GetType(typeID)
			if typeInfo == nil {
				continue
			}

			// Apply type filter if specified
			if *typeFilter != "" && typeID.Name != *typeFilter {
				continue
			}

			fmt.Printf("\n  %s (%s)\n", typeID.Name, typeInfo.Kind)

			if typeInfo.Kind == analyze.TypeKindStruct {
				for _, field := range typeInfo.Fields {
					if !field.Exported {
						continue
					}

					typeStr := stringer.TypeString(field.Type)
					fmt.Printf("    - %s: %s\n", field.Name, typeStr)

					if *verbose && field.Tag != "" {
						fmt.Printf("      tags: %s\n", field.Tag)
					}
				}
			}
		}
	}

	fmt.Println()
}

// runSuggest implements the 'suggest' command.
func runSuggest(args []string) {
	fs := flag.NewFlagSet("suggest", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: caster-generator suggest [options]

Generate a suggested YAML mapping for a type pair.

Options:
`)
		fs.PrintDefaults()
	}

	var packages StringSliceFlag

	fs.Var(&packages, "pkg", "Package path to analyze (auto-detected from type names if not specified)")
	mappingFile := fs.String("mapping", "", "Path to existing YAML mapping file to improve")
	fromType := fs.String("from", "", "Source type (e.g., store.Order) - required if no mapping file")
	toType := fs.String("to", "", "Target type (e.g., warehouse.Order) - required if no mapping file")
	outFile := fs.String("out", "", "Output YAML file (default: stdout)")
	minConfidence := fs.Float64("min-confidence", 0.7, "Minimum confidence for auto-matching (0.0-1.0)")
	minGap := fs.Float64("min-gap", 0.15, "Minimum score gap between top candidates for auto-accept")
	ambiguityThreshold := fs.Float64("ambiguity-threshold", 0.1, "Score difference threshold for marking ambiguity")
	maxCandidates := fs.Int("max-candidates", 5, "Maximum number of candidates to include in suggestions")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Auto-detect packages from type names if not specified
	if len(packages) == 0 {
		fromPkg := extractPackage(*fromType)
		toPkg := extractPackage(*toType)

		if fromPkg != "" {
			packages = append(packages, "./"+fromPkg)
		}

		if toPkg != "" && toPkg != fromPkg {
			packages = append(packages, "./"+toPkg)
		}
	}

	// Try to load existing mapping file
	var mappingDef *mapping.MappingFile

	// First try -mapping flag
	if *mappingFile != "" {
		if existingDef, err := mapping.LoadFile(*mappingFile); err == nil {
			mappingDef = existingDef

			fmt.Printf("Loaded existing mapping from %s\n", *mappingFile)

			// Auto-detect packages from existing mapping if not specified
			if len(packages) == 0 {
				packages = extractPackagesFromMapping(mappingDef)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error loading mapping file: %v\n", err)
			os.Exit(1)
		}
	} else if *outFile != "" {
		// Then try -out file if it exists
		if existingDef, err := mapping.LoadFile(*outFile); err == nil {
			mappingDef = existingDef

			fmt.Printf("Loaded existing mapping from %s\n", *outFile)

			// Auto-detect packages from existing mapping if not specified
			if len(packages) == 0 {
				packages = extractPackagesFromMapping(mappingDef)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Note: could not load existing mapping from %s: %v\n", *outFile, err)
		}
	}

	// If no existing mapping, create a minimal one
	if mappingDef == nil {
		if *fromType == "" || *toType == "" {
			fmt.Fprintln(os.Stderr, "Error: -from and -to flags are required when no existing mapping file")
			fs.Usage()
			os.Exit(1)
		}

		mappingDef = &mapping.MappingFile{
			Version: "1",
			TypeMappings: []mapping.TypeMapping{
				{
					Source: *fromType,
					Target: *toType,
				},
			},
		}
	}

	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "Error: cannot auto-detect packages. "+
			"Use qualified type names (e.g., store.Order) or specify -pkg flags")
		fs.Usage()
		os.Exit(1)
	}

	// Load packages
	analyzer := analyze.NewAnalyzer()

	graph, err := analyzer.LoadPackages(packages...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading packages: %v\n", err)
		os.Exit(1)
	}

	// Run resolution with auto-matching
	config := plan.DefaultConfig()
	config.MinConfidence = *minConfidence
	config.MinGap = *minGap
	config.AmbiguityThreshold = *ambiguityThreshold
	config.MaxCandidates = *maxCandidates
	resolver := plan.NewResolver(graph, mappingDef, config)

	resolvedPlan, err := resolver.Resolve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving mappings: %v\n", err)
		os.Exit(1)
	}

	// Export suggestions as YAML with threshold info in comments
	exportConfig := plan.ExportConfig{
		MinConfidence:           *minConfidence,
		MinGap:                  *minGap,
		AmbiguityThreshold:      *ambiguityThreshold,
		IncludeRejectedComments: true,
	}

	yamlData, err := plan.ExportSuggestionsYAMLWithConfig(resolvedPlan, exportConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting suggestions: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if *outFile != "" {
		err := os.WriteFile(*outFile, yamlData, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Suggested mapping written to %s\n", *outFile)
	} else {
		fmt.Print(string(yamlData))
	}

	// Print diagnostics summary
	printDiagnostics(&resolvedPlan.Diagnostics)

	// Warn about incomplete mappings that were fixed with placeholders
	incompleteMappings := resolvedPlan.FindIncompleteMappings()
	if len(incompleteMappings) > 0 {
		fmt.Fprintln(os.Stderr, "\nNote: The following mappings have incompatible types "+
			"and were moved to 'fields' with placeholder transforms:")

		for _, im := range incompleteMappings {
			fmt.Fprintf(os.Stderr, "  - %s -> %s (in %s)\n", im.SourcePath, im.TargetPath, im.TypePair)
			fmt.Fprintf(os.Stderr, "    reason: %s\n", im.Explanation)
		}

		fmt.Fprintln(os.Stderr, "\nPlease implement the TODO_* transform functions "+
			"or rename them to your actual function names.")
	}
}

// runGen implements the 'gen' command.
func runGen(args []string) {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: caster-generator gen [options]

Generate casters using YAML mapping.

Options:
`)
		fs.PrintDefaults()
	}

	var packages StringSliceFlag

	fs.Var(&packages, "pkg", "Package path to analyze (can be specified multiple times)")
	mappingFile := fs.String("mapping", "", "Path to YAML mapping file (required)")
	outDir := fs.String("out", "./generated", "Output directory for generated files")
	pkgName := fs.String("package", "casters", "Package name for generated code")
	strict := fs.Bool("strict", false, "Fail on any unresolved target fields")
	writeSuggestions := fs.String("write-suggestions", "", "Write suggested mapping YAML to this file")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *mappingFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -mapping flag is required")
		fs.Usage()
		os.Exit(1)
	}

	// Load mapping file
	mappingDef, err := mapping.LoadFile(*mappingFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mapping file: %v\n", err)
		os.Exit(1)
	}

	// Auto-detect packages from mapping if not specified
	if len(packages) == 0 {
		packages = extractPackagesFromMapping(mappingDef)
	}

	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one -pkg flag is required, or mapping must use qualified type names")
		fs.Usage()
		os.Exit(1)
	}

	// Load packages
	analyzer := analyze.NewAnalyzer()

	graph, err := analyzer.LoadPackages(packages...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading packages: %v\n", err)
		os.Exit(1)
	}

	// Validate mapping against type graph
	if result := mapping.Validate(mappingDef, graph); !result.IsValid() {
		fmt.Fprintln(os.Stderr, "Mapping validation errors:")

		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}

		os.Exit(1)
	}

	// Run resolution
	config := plan.DefaultConfig()
	config.StrictMode = *strict
	resolver := plan.NewResolver(graph, mappingDef, config)

	resolvedPlan, err := resolver.Resolve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving mappings: %v\n", err)
		os.Exit(1)
	}

	// Print diagnostics
	printDiagnostics(&resolvedPlan.Diagnostics)

	// Check for incomplete mappings (types that need transforms but don't have them)
	incompleteMappings := resolvedPlan.FindIncompleteMappings()
	if len(incompleteMappings) > 0 {
		fmt.Fprintln(os.Stderr, "\nError: Found mappings with incompatible types that require custom transform functions:")

		for _, im := range incompleteMappings {
			fmt.Fprintf(os.Stderr, "  - %s -> %s (in %s)\n", im.SourcePath, im.TargetPath, im.TypePair)
			fmt.Fprintf(os.Stderr, "    reason: %s\n", im.Explanation)
			fmt.Fprintf(os.Stderr, "    source: %s\n", im.Source)
		}

		fmt.Fprintln(os.Stderr, "\nTo fix this:")
		fmt.Fprintln(os.Stderr, "  1. Move these mappings from '121' to 'fields' section in your YAML")
		fmt.Fprintln(os.Stderr, "  2. Add a 'transform' function name for each")
		fmt.Fprintln(os.Stderr, "  3. Implement the transform functions in your code")
		fmt.Fprintln(os.Stderr, "\nOr run 'suggest' command to auto-generate updated YAML with placeholders.")
		os.Exit(1)
	}

	// Write suggestions if requested
	if *writeSuggestions != "" {
		yamlData, err := plan.ExportSuggestionsYAML(resolvedPlan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting suggestions: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*writeSuggestions, yamlData, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing suggestions file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Suggested mapping written to %s\n", *writeSuggestions)
	}

	// Generate code
	generator := gen.NewGenerator(gen.GeneratorConfig{
		PackageName:          *pkgName,
		OutputDir:            *outDir,
		GenerateComments:     true,
		IncludeUnmappedTODOs: true,
	})

	files, err := generator.Generate(resolvedPlan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
		os.Exit(1)
	}

	// Write files
	if err := gen.WriteFiles(files, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing generated files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d file(s) in %s\n", len(files), *outDir)

	for _, f := range files {
		fmt.Printf("  - %s\n", f.Filename)
	}
}

// runCheck implements the 'check' command.
func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: caster-generator check [options]

Validate YAML against current code; fail on drift.

Options:
`)
		fs.PrintDefaults()
	}

	var packages StringSliceFlag

	fs.Var(&packages, "pkg", "Package path to analyze (can be specified multiple times)")
	mappingFile := fs.String("mapping", "", "Path to YAML mapping file (required)")
	strict := fs.Bool("strict", false, "Fail on any unresolved target fields")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *mappingFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -mapping flag is required")
		fs.Usage()
		os.Exit(1)
	}

	// Load mapping file
	mappingDef, err := mapping.LoadFile(*mappingFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mapping file: %v\n", err)
		os.Exit(1)
	}

	// Auto-detect packages from mapping if not specified
	if len(packages) == 0 {
		packages = extractPackagesFromMapping(mappingDef)
	}

	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one -pkg flag is required, or mapping must use qualified type names")
		fs.Usage()
		os.Exit(1)
	}

	// Load packages
	analyzer := analyze.NewAnalyzer()

	graph, err := analyzer.LoadPackages(packages...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading packages: %v\n", err)
		os.Exit(1)
	}

	// Validate mapping against type graph
	validationResult := mapping.Validate(mappingDef, graph)
	if !validationResult.IsValid() {
		fmt.Fprintln(os.Stderr, "Mapping validation errors:")

		for _, e := range validationResult.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}

		os.Exit(1)
	}

	// Run resolution to check for issues
	config := plan.DefaultConfig()
	config.StrictMode = *strict
	resolver := plan.NewResolver(graph, mappingDef, config)

	resolvedPlan, err := resolver.Resolve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving mappings: %v\n", err)
		os.Exit(1)
	}

	// Print diagnostics
	printDiagnostics(&resolvedPlan.Diagnostics)

	// Check for issues
	hasIssues := false

	for _, tp := range resolvedPlan.TypePairs {
		if len(tp.UnmappedTargets) > 0 {
			hasIssues = true

			fmt.Printf("\nUnmapped targets in %s -> %s:\n", tp.SourceType.ID, tp.TargetType.ID)

			for _, um := range tp.UnmappedTargets {
				fmt.Printf("  - %s: %s\n", um.TargetPath, um.Reason)
			}
		}
	}

	if resolvedPlan.Diagnostics.HasErrors() {
		hasIssues = true
	}

	if hasIssues {
		fmt.Fprintln(os.Stderr, "\nCheck failed: mapping has issues")
		os.Exit(1)
	}

	fmt.Println("Check passed: mapping is valid")
}

// extractPackage extracts the package path from a qualified type name.
// Handles both short forms (e.g., "store.Order") and full import paths
// (e.g., "caster-generator/store.Product").
func extractPackage(typeName string) string {
	// Find the last dot which separates package from type name
	lastDot := strings.LastIndex(typeName, ".")
	if lastDot == -1 {
		return ""
	}

	return typeName[:lastDot]
}

// extractPackagesFromMapping extracts package paths from mapping type names.
func extractPackagesFromMapping(mf *mapping.MappingFile) []string {
	pkgSet := make(map[string]bool)

	for _, tm := range mf.TypeMappings {
		if pkg := extractPackage(tm.Source); pkg != "" {
			// Check if it's a full import path or relative
			if strings.Contains(pkg, "/") {
				pkgSet[pkg] = true
			} else {
				pkgSet["./"+pkg] = true
			}
		}

		if pkg := extractPackage(tm.Target); pkg != "" {
			// Check if it's a full import path or relative
			if strings.Contains(pkg, "/") {
				pkgSet[pkg] = true
			} else {
				pkgSet["./"+pkg] = true
			}
		}
	}

	var packages []string
	for pkg := range pkgSet {
		packages = append(packages, pkg)
	}

	return packages
}

// printDiagnostics prints diagnostic information to stderr.
func printDiagnostics(diags *diagnostic.Diagnostics) {
	if len(diags.Warnings) > 0 {
		fmt.Fprintln(os.Stderr, "\nWarnings:")

		for _, w := range diags.Warnings {
			fmt.Fprintf(os.Stderr, "  [%s] %s\n", w.Code, w.Message)

			if w.TypePair != "" {
				fmt.Fprintf(os.Stderr, "    type pair: %s\n", w.TypePair)
			}

			if w.FieldPath != "" {
				fmt.Fprintf(os.Stderr, "    field: %s\n", w.FieldPath)
			}
		}
	}

	if len(diags.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "\nErrors:")

		for _, e := range diags.Errors {
			fmt.Fprintf(os.Stderr, "  [%s] %s\n", e.Code, e.Message)

			if e.TypePair != "" {
				fmt.Fprintf(os.Stderr, "    type pair: %s\n", e.TypePair)
			}

			if e.FieldPath != "" {
				fmt.Fprintf(os.Stderr, "    field: %s\n", e.FieldPath)
			}
		}
	}
}
