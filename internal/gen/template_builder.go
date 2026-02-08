package gen

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/plan"
)

// templateData holds all data needed for the caster template.
type templateData struct {
	PackageName       string
	Filename          string
	Imports           []importSpec
	FunctionName      string
	SourceType        typeRef
	TargetType        typeRef
	Assignments       []assignmentData
	UnmappedTODOs     []string
	GenerateComments  bool
	NestedCasters     []nestedCasterRef
	MissingTransforms []MissingTransform
	ExtraArgs         []extraArg
	StructDef         string
}

// extraArg represents an additional argument to a caster function.
type extraArg struct {
	Name string
	Type string
}

// assignmentData represents a single field assignment in the caster.
type assignmentData struct {
	TargetField string
	SourceExpr  string
	Comment     string
	Strategy    plan.ConversionStrategy
	// For slice mapping
	IsSlice      bool
	SliceElemVar string
	SliceBody    string
	// For map mapping
	IsMap   bool
	MapBody string
	// For nested caster
	NestedCaster string
	// For nil check wrapper
	NeedsNilCheck bool
	NilDefault    string
	// For pointer nil check
	NilCheckExpr string
}

// nestedCasterRef tracks a nested caster function that needs to be called.
type nestedCasterRef struct {
	FunctionName string
	SourceType   typeRef
	TargetType   typeRef
}

// buildTemplateData constructs the template data from a resolved type pair.
func (g *Generator) buildTemplateData(pair *plan.ResolvedTypePair) *templateData {
	srcPkgAlias := g.getPkgName(pair.SourceType.ID.PkgPath)
	tgtPkgAlias := g.getPkgName(pair.TargetType.ID.PkgPath)

	// For generated targets, don't use package prefix (type is generated in same package)
	if pair.IsGeneratedTarget {
		tgtPkgAlias = ""
	}

	data := &templateData{
		PackageName:      g.config.PackageName,
		Filename:         g.filename(pair),
		FunctionName:     g.functionName(pair),
		GenerateComments: g.config.GenerateComments,
		SourceType: typeRef{
			Package: srcPkgAlias,
			Name:    pair.SourceType.ID.Name,
		},
		TargetType: typeRef{
			Package: tgtPkgAlias,
			Name:    pair.TargetType.ID.Name,
		},
	}

	// Add Requires as extra args
	if len(pair.Requires) > 0 {
		for _, req := range pair.Requires {
			data.ExtraArgs = append(data.ExtraArgs, extraArg{
				Name: req.Name,
				Type: req.Type,
			})
		}
	}

	// Collect imports
	imports := make(map[string]importSpec)
	g.addImport(imports, pair.SourceType.ID.PkgPath)
	// Don't add import for generated target types
	if !pair.IsGeneratedTarget {
		g.addImport(imports, pair.TargetType.ID.PkgPath)
	}

	// Generate struct definition if needed
	g.processStructDefinition(data, pair, imports)

	// Process mappings
	for _, m := range pair.Mappings {
		assignment := g.buildAssignment(&m, pair, imports)
		if assignment != nil {
			data.Assignments = append(data.Assignments, *assignment)
		}
	}

	// Reorder assignments based on implicit dependencies (e.g., extra.def.target).
	g.orderAssignmentsByDependencies(data, pair)

	// Add TODO comments for unmapped fields
	if g.config.IncludeUnmappedTODOs {
		for _, unmapped := range pair.UnmappedTargets {
			todo := fmt.Sprintf("TODO: %s - %s", unmapped.TargetPath, unmapped.Reason)
			data.UnmappedTODOs = append(data.UnmappedTODOs, todo)
		}
	}

	// Collect nested casters
	g.collectNestedCasters(data, pair, imports)

	// Identify missing transforms
	g.identifyMissingTransforms(pair)

	// Convert imports map to sorted slice
	for _, imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	sort.Slice(data.Imports, func(i, j int) bool {
		return data.Imports[i].Path < data.Imports[j].Path
	})

	return data
}

// processStructDefinition handles struct definition generation and placement.
func (g *Generator) processStructDefinition(
	data *templateData,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	// Decide if we are moving the struct definition
	moveStruct := false
	targetPkgPath := pair.TargetType.ID.PkgPath

	// Logic: If the target has a defined package path that is NOT empty,
	// and we can find its physical directory, we move it there.
	if pair.IsGeneratedTarget && targetPkgPath != "" {
		if pkgInfo, ok := g.graph.Packages[targetPkgPath]; ok && pkgInfo.Dir != "" {
			moveStruct = true
		}
	}

	// Set context package path for struct generation to ensure correct type references
	if moveStruct {
		g.contextPkgPath = targetPkgPath
	} else {
		g.contextPkgPath = ""
	}

	// Use a temporary map to capture imports for the struct
	structImports := make(map[string]importSpec)
	if structDef, err := g.GenerateStruct(pair, structImports); err == nil {
		if moveStruct {
			// Add import to caster file manually (since we removed struct def from here)
			g.addImport(imports, targetPkgPath)

			// We also need to ensure tgtPkgAlias is set in templateData
			tgtPkgAlias := g.getPkgName(targetPkgPath)
			data.TargetType.Package = tgtPkgAlias

			// Convert structImports map to slice for storage
			var importedSpecs []importSpec
			for _, spec := range structImports {
				importedSpecs = append(importedSpecs, spec)
			}

			// Store struct def for later generation in the target package
			dir := g.graph.Packages[targetPkgPath].Dir
			pkgName := g.graph.Packages[targetPkgPath].Name
			g.addMissingType(dir, pkgName, structDef, importedSpecs)
		} else {
			// Merge structImports into imports for the current file
			maps.Copy(imports, structImports)

			data.StructDef = structDef
		}
	}

	// Reset context package path
	g.contextPkgPath = ""
}

// buildAssignment creates an assignment data from a resolved field mapping.
func (g *Generator) buildAssignment(
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) *assignmentData {
	// Skip ignored mappings
	if m.Strategy == plan.StrategyIgnore {
		return nil
	}

	targetField := g.targetFieldExpr(m.TargetPaths)
	sourceExpr := g.sourceFieldExpr(m.SourcePaths, m, pair)

	comment := ""
	if g.config.GenerateComments && m.Explanation != "" {
		comment = m.Explanation
	}

	assignment := &assignmentData{
		TargetField: targetField,
		SourceExpr:  sourceExpr,
		Comment:     comment,
		Strategy:    m.Strategy,
	}

	g.applyConversionStrategy(assignment, m, pair, imports)

	return assignment
}

// collectNestedCasters adds nested caster references to the template data.
func (g *Generator) collectNestedCasters(
	data *templateData,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	for _, nested := range pair.NestedPairs {
		nestedRef := nestedCasterRef{
			FunctionName: g.nestedFunctionName(nested.SourceType, nested.TargetType),
			SourceType: typeRef{
				Package: g.getPkgName(nested.SourceType.ID.PkgPath),
				Name:    nested.SourceType.ID.Name,
			},
			TargetType: typeRef{
				Package: g.getPkgName(nested.TargetType.ID.PkgPath),
				Name:    nested.TargetType.ID.Name,
			},
		}
		// Add imports for nested types
		g.addImport(imports, nested.SourceType.ID.PkgPath)
		g.addImport(imports, nested.TargetType.ID.PkgPath)

		data.NestedCasters = append(data.NestedCasters, nestedRef)
	}
}

// orderAssignmentsByDependencies topologically sorts assignments based on
// ResolvedFieldMapping.DependsOnTargets.
func (g *Generator) orderAssignmentsByDependencies(data *templateData, pair *plan.ResolvedTypePair) {
	if data == nil || pair == nil {
		return
	}

	if len(data.Assignments) == 0 || len(pair.Mappings) == 0 {
		return
	}

	// Assume buildAssignment produced 1 assignment per mapping, in the same order.
	// That's true for current generator behavior.
	n := min(len(pair.Mappings), len(data.Assignments))

	// Build index by exact target field expr, using the assignment list.
	byTarget := make(map[string]int, n)
	for i := range n {
		t := data.Assignments[i].TargetField
		if t != "" {
			byTarget[t] = i
		}
	}

	order, err := topoSortAssignments(n, func(i int) []int {
		m := pair.Mappings[i]
		if len(m.DependsOnTargets) == 0 {
			return nil
		}

		var deps []int

		for _, dep := range m.DependsOnTargets {
			depExpr := "out." + dep.String()

			j, ok := byTarget[depExpr]
			if !ok {
				// Missing dependency should have been flagged by planner; ignore here.
				continue
			}

			deps = append(deps, j)
		}

		// Deterministic.
		sort.Ints(deps)

		return deps
	})
	if err != nil {
		// Best-effort: keep original order.
		return
	}

	reordered := make([]assignmentData, 0, n)
	for _, idx := range order {
		reordered = append(reordered, data.Assignments[idx])
	}

	// Keep any tail assignments (shouldn't exist today, but stay safe).
	if len(data.Assignments) > n {
		reordered = append(reordered, data.Assignments[n:]...)
	}

	data.Assignments = reordered
}

// targetFieldExpr builds the target field expression (e.g., "out.Name", "out.Address.Street").
func (g *Generator) targetFieldExpr(paths []mapping.FieldPath) string {
	if len(paths) == 0 {
		return ""
	}
	// For 1:N mappings, we'd need multiple assignments; for now handle the first
	return "out." + paths[0].String()
}

// sourceFieldExpr builds the source field expression.
func (g *Generator) sourceFieldExpr(
	paths []mapping.FieldPath,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
) string {
	if len(paths) == 0 {
		if m.Default != nil {
			return *m.Default
		}

		return ""
	}

	// Check if this path refers to a required argument
	firstSegment := paths[0].Segments[0].Name
	for _, req := range pair.Requires {
		if req.Name == firstSegment {
			return paths[0].String()
		}
	}

	return "in." + paths[0].String()
}

// buildTransformArgs builds the argument list for a transform function call.
func (g *Generator) buildTransformArgs(paths []mapping.FieldPath, pair *plan.ResolvedTypePair) string {
	args := make([]string, 0, len(paths))

	for _, p := range paths {
		// Check if this path refers to a required argument
		isReq := false

		if len(p.Segments) > 0 {
			firstSegment := p.Segments[0].Name
			for _, req := range pair.Requires {
				if req.Name == firstSegment {
					isReq = true
					break
				}
			}
		}

		if isReq {
			args = append(args, p.String())
		} else {
			args = append(args, "in."+p.String())
		}
	}

	return strings.Join(args, ", ")
}

// getRequiredArgType returns the TypeInfo for a required argument by name, or nil if not found.
func (g *Generator) getRequiredArgType(pair *plan.ResolvedTypePair, name string) *analyze.TypeInfo {
	for _, req := range pair.Requires {
		if req.Name == name {
			return g.typeInfoFromString(req.Type)
		}
	}

	return nil
}

// identifyMissingTransforms finds referenced transforms that are not imported or defined.
func (g *Generator) identifyMissingTransforms(
	pair *plan.ResolvedTypePair,
) {
	if g.missingTransforms == nil {
		g.missingTransforms = make(map[string]MissingTransformInfo)
	}

	seen := make(map[string]bool)

	for _, m := range pair.Mappings {
		if m.Transform == "" {
			continue
		}

		// If transform contains a dot, it's likely a package call (or method)
		if strings.Contains(m.Transform, ".") {
			continue
		}

		// If transform is declared in the mapping file, skip it
		if g.config.DeclaredTransforms != nil && g.config.DeclaredTransforms[m.Transform] {
			continue
		}

		if !seen[m.Transform] {
			// Check if we already have this transform in the global map
			if _, exists := g.missingTransforms[m.Transform]; exists {
				seen[m.Transform] = true
				continue
			}

			// Determine argument types
			var argInfos []*analyze.TypeInfo

			for _, sp := range m.SourcePaths {
				// First check if this source path refers to a required argument
				var info *analyze.TypeInfo
				if len(sp.Segments) > 0 {
					info = g.getRequiredArgType(pair, sp.Segments[0].Name)
				}

				// If not a required arg, look up from source type
				if info == nil {
					info = g.getFieldTypeInfo(pair.SourceType, sp.String())
				}

				argInfos = append(argInfos, info)
			}

			// Also add 'extra' types if any
			for _, exp := range m.Extra {
				var info *analyze.TypeInfo

				// First check if the extra matches a required argument
				info = g.getRequiredArgType(pair, exp.Name)
				if info != nil {
					argInfos = append(argInfos, info)
					continue
				}

				switch {
				case exp.Def.Source != "":
					// Check if source refers to a required arg
					info = g.getRequiredArgType(pair, exp.Def.Source)
					if info == nil {
						info = g.getFieldTypeInfo(pair.SourceType, exp.Def.Source)
					}
				case exp.Def.Target != "":
					// Reference to target type field
					info = g.getFieldTypeInfo(pair.TargetType, exp.Def.Target)
				default:
					// Fallback - check if name matches a required arg
					info = g.getRequiredArgType(pair, exp.Name)
				}

				argInfos = append(argInfos, info)
			}

			// Determine return type
			var returnInfo *analyze.TypeInfo
			if len(m.TargetPaths) > 0 {
				returnInfo = g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())
			}

			g.missingTransforms[m.Transform] = MissingTransformInfo{
				Name:       m.Transform,
				Args:       argInfos,
				ReturnType: returnInfo,
			}
			seen[m.Transform] = true
		}
	}
}
