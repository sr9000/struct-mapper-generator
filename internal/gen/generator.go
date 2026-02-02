package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"path"
	"sort"
	"strings"
	"text/template"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/plan"
)

// interfaceTypeStr is used as a fallback type string.
const interfaceTypeStr = "interface{}"

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
	var files []GeneratedFile

	for _, pair := range p.TypePairs {
		file, err := g.generateTypePair(&pair)
		if err != nil {
			return nil, fmt.Errorf("generating %s->%s: %w",
				pair.SourceType.ID, pair.TargetType.ID, err)
		}

		files = append(files, *file)
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

// MissingTransform represents a missing transform function that needs stub generation.
type MissingTransform struct {
	Name       string
	Args       []string
	ReturnType string
}

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
}

type extraArg struct {
	Name string
	Type string
}

// importSpec represents an import statement.
type importSpec struct {
	Alias string
	Path  string
}

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
	srcPkgAlias := g.pkgAlias(pair.SourceType.ID.PkgPath)
	tgtPkgAlias := g.pkgAlias(pair.TargetType.ID.PkgPath)

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
	g.addImport(imports, pair.TargetType.ID.PkgPath)

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
	// Pass imports so that type resolution knows about aliases
	data.MissingTransforms = g.identifyMissingTransforms(pair, imports)

	// Convert imports map to sorted slice
	for _, imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	sort.Slice(data.Imports, func(i, j int) bool {
		return data.Imports[i].Path < data.Imports[j].Path
	})

	return data
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
	// That’s true for current generator behavior.
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
				Package: g.pkgAlias(nested.SourceType.ID.PkgPath),
				Name:    nested.SourceType.ID.Name,
			},
			TargetType: typeRef{
				Package: g.pkgAlias(nested.TargetType.ID.PkgPath),
				Name:    nested.TargetType.ID.Name,
			},
		}
		// Add imports for nested types
		g.addImport(imports, nested.SourceType.ID.PkgPath)
		g.addImport(imports, nested.TargetType.ID.PkgPath)

		data.NestedCasters = append(data.NestedCasters, nestedRef)
	}
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

// applyConversionStrategy applies the conversion strategy to the assignment.
func (g *Generator) applyConversionStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	switch m.Strategy {
	case plan.StrategyDirectAssign:
		// Simple assignment, nothing extra needed

	case plan.StrategyConvert:
		g.applyConvertStrategy(assignment, m, pair, imports)

	case plan.StrategyPointerDeref:
		g.applyPointerDerefStrategy(assignment, m, pair)

	case plan.StrategyPointerWrap:
		g.applyPointerWrapStrategy(assignment, m, pair, imports)

	case plan.StrategySliceMap:
		assignment.IsSlice = true
		assignment.SliceElemVar = "i"
		assignment.SliceBody = g.buildSliceMapping(m, pair, imports)

	case plan.StrategyPointerNestedCast:
		g.applyPointerNestedCastStrategy(assignment, m, pair, imports)

	case plan.StrategyNestedCast:
		g.applyNestedCastStrategy(assignment, m, pair)

	case plan.StrategyTransform:
		g.applyTransformStrategy(assignment, m, pair)

	case plan.StrategyDefault:
		if m.Default != nil {
			assignment.SourceExpr = *m.Default
		}

	case plan.StrategyIgnore:
		// Already handled above
	}
}

func (g *Generator) applyConvertStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	if len(m.TargetPaths) > 0 {
		targetType := g.getFieldType(pair.TargetType, m.TargetPaths[0].String())
		if targetType != nil {
			assignment.SourceExpr = g.wrapConversion(assignment.SourceExpr, targetType, imports)
		}
	}
}

func (g *Generator) applyPointerDerefStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
) {
	assignment.NeedsNilCheck = true
	// Keep the original pointer expression for the nil-check; use a dereferenced
	// expression for the actual assignment.
	assignment.NilDefault = g.zeroValue(pair.TargetType, m.TargetPaths)
	assignment.NilCheckExpr = assignment.SourceExpr
	assignment.SourceExpr = "*" + assignment.SourceExpr
}

func (g *Generator) applyPointerWrapStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	if len(m.SourcePaths) > 0 {
		typeStr := g.getFieldTypeString(pair.SourceType, m.SourcePaths[0].String(), imports)
		srcExpr := g.sourceFieldExpr(m.SourcePaths, m, pair)
		assignment.SourceExpr = fmt.Sprintf("func() *%s { v := %s; return &v }()", typeStr, srcExpr)
	}
}

func (g *Generator) applyPointerNestedCastStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) {
	if len(m.SourcePaths) == 0 || len(m.TargetPaths) == 0 {
		return
	}

	srcType := g.getFieldTypeInfo(pair.SourceType, m.SourcePaths[0].String())
	tgtType := g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())

	if srcType == nil || tgtType == nil {
		return
	}

	// Get the element types (what the pointers point to)
	srcElem := srcType
	tgtElem := tgtType

	if srcType.Kind == analyze.TypeKindPointer && srcType.ElemType != nil {
		srcElem = srcType.ElemType
	}

	if tgtType.Kind == analyze.TypeKindPointer && tgtType.ElemType != nil {
		tgtElem = tgtType.ElemType
	}

	casterName := g.nestedFunctionName(srcElem, tgtElem)
	tgtElemStr := g.typeRefString(tgtElem, imports)

	// Generate: func() *TargetType { if src == nil { return nil }; v := Caster(*src); return &v }()
	assignment.SourceExpr = fmt.Sprintf(
		"func() *%s { if %s == nil { return nil }; v := %s(*%s); return &v }()",
		tgtElemStr, assignment.SourceExpr, casterName, assignment.SourceExpr,
	)
}

func (g *Generator) applyNestedCastStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
) {
	if len(m.SourcePaths) == 0 {
		return
	}

	srcType := g.getFieldTypeInfo(pair.SourceType, m.SourcePaths[0].String())
	tgtType := g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())

	if srcType != nil && tgtType != nil {
		casterName := g.nestedFunctionName(srcType, tgtType)
		assignment.NestedCaster = casterName
		// Always call the nested caster with the resolved source expression.
		assignment.SourceExpr = fmt.Sprintf("%s(%s)", casterName, assignment.SourceExpr)
	}
}

func (g *Generator) applyTransformStrategy(
	assignment *assignmentData,
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
) {
	if m.Transform == "" {
		return
	}

	args := g.buildTransformArgs(m.SourcePaths, pair)

	// Append extras after explicit source paths (stable order as specified in YAML).
	// Extras can reference either source fields or target fields.
	if len(m.Extra) > 0 {
		var extraArgs []string

		for _, ev := range m.Extra {
			// Prefer explicit source/target, else fallback to the extra name.
			if ev.Def.Source != "" {
				extraArgs = append(extraArgs, "in."+ev.Def.Source)
				continue
			}

			if ev.Def.Target != "" {
				extraArgs = append(extraArgs, "out."+ev.Def.Target)
				continue
			}

			// If Name matches a required arg, it should be passed verbatim.
			isReq := false

			for _, req := range pair.Requires {
				if req.Name == ev.Name {
					isReq = true
					break
				}
			}

			if isReq {
				extraArgs = append(extraArgs, ev.Name)
			} else {
				extraArgs = append(extraArgs, "in."+ev.Name)
			}
		}

		if args == "" {
			args = strings.Join(extraArgs, ", ")
		} else {
			args = args + ", " + strings.Join(extraArgs, ", ")
		}
	}

	assignment.SourceExpr = fmt.Sprintf("%s(%s)", m.Transform, args)
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

// wrapConversion wraps an expression with type conversion.
func (g *Generator) wrapConversion(
	expr string,
	targetType *analyze.TypeInfo,
	imports map[string]importSpec,
) string {
	typeStr := g.typeRefString(targetType, imports)

	return fmt.Sprintf("%s(%s)", typeStr, expr)
}

// buildSliceMapping generates the slice mapping code.
func (g *Generator) buildSliceMapping(
	m *plan.ResolvedFieldMapping,
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) string {
	if len(m.SourcePaths) == 0 || len(m.TargetPaths) == 0 {
		return ""
	}

	srcField := "in." + m.SourcePaths[0].String()
	tgtField := "out." + m.TargetPaths[0].String()

	srcType := g.getFieldTypeInfo(pair.SourceType, m.SourcePaths[0].String())
	tgtType := g.getFieldTypeInfo(pair.TargetType, m.TargetPaths[0].String())

	if srcType == nil || tgtType == nil {
		return fmt.Sprintf("// TODO: could not determine types for slice mapping %s -> %s",
			m.SourcePaths[0], m.TargetPaths[0])
	}

	return g.generateSliceLoopCode(srcField, tgtField, srcType, tgtType, imports)
}

func (g *Generator) generateSliceLoopCode(
	srcField, tgtField string,
	srcType, tgtType *analyze.TypeInfo,
	imports map[string]importSpec,
) string {
	srcElem := g.getSliceElementType(srcType)
	tgtElem := g.getSliceElementType(tgtType)

	if srcElem == nil || tgtElem == nil {
		return "// TODO: could not determine element types for slice mapping"
	}

	tgtElemStr := g.typeRefString(tgtElem, imports)
	elemConv := g.buildElementConversion(srcField, srcElem, tgtElem, tgtElemStr)

	// Arrays are fixed size; slices are variable.
	if srcType.Kind == analyze.TypeKindArray && tgtType.Kind == analyze.TypeKindArray {
		return fmt.Sprintf(`for i := range %s {
		%s[i] = %s
	}`, srcField, tgtField, elemConv)
	}

	return fmt.Sprintf(`%s = make([]%s, len(%s))
	for i := range %s {
		%s[i] = %s
	}`, tgtField, tgtElemStr, srcField, srcField, tgtField, elemConv)
}

func (g *Generator) getSliceElementType(t *analyze.TypeInfo) *analyze.TypeInfo {
	if t.Kind == analyze.TypeKindSlice && t.ElemType != nil {
		return t.ElemType
	}

	if t.Kind == analyze.TypeKindArray && t.ElemType != nil {
		return t.ElemType
	}

	return nil
}

func (g *Generator) buildElementConversion(
	srcField string,
	srcElem, tgtElem *analyze.TypeInfo,
	tgtElemStr string,
) string {
	if g.typesIdentical(srcElem, tgtElem) {
		return srcField + "[i]"
	}

	if g.typesConvertible(srcElem, tgtElem) {
		return fmt.Sprintf("%s(%s[i])", tgtElemStr, srcField)
	}

	if srcElem.Kind == analyze.TypeKindStruct && tgtElem.Kind == analyze.TypeKindStruct {
		casterName := g.nestedFunctionName(srcElem, tgtElem)

		return fmt.Sprintf("%s(%s[i])", casterName, srcField)
	}

	// Fallback - hope for the best
	return srcField + "[i]"
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

// identifyMissingTransforms finds referenced transforms that are not imported or defined.
func (g *Generator) identifyMissingTransforms(
	pair *plan.ResolvedTypePair,
	imports map[string]importSpec,
) []MissingTransform {
	var missing []MissingTransform

	seen := make(map[string]bool)

	for _, m := range pair.Mappings {
		if m.Transform == "" {
			continue
		}

		// If transform contains a dot, it's likely a package call (or method)
		if strings.Contains(m.Transform, ".") {
			continue
		}

		if !seen[m.Transform] {
			// Determine argument types
			var argTypes []string

			for _, sp := range m.SourcePaths {
				typ := g.getFieldTypeString(pair.SourceType, sp.String(), imports)
				argTypes = append(argTypes, typ)
			}

			// Also add 'extra' types if any
			for _, exp := range m.Extra {
				var typ string
				if exp.Def.Source != "" {
					typ = g.getFieldTypeString(pair.SourceType, exp.Def.Source, imports)
				} else if exp.Def.Target != "" {
					// Reference to target type field?
					typ = g.getFieldTypeString(pair.TargetType, exp.Def.Target, imports)
				} else {
					// Fallback
					typ = "interface{}"
				}

				argTypes = append(argTypes, typ)
			}

			// Determine return type
			returnType := "interface{}"
			if len(m.TargetPaths) > 0 {
				returnType = g.getFieldTypeString(pair.TargetType, m.TargetPaths[0].String(), imports)
			}

			missing = append(missing, MissingTransform{
				Name:       m.Transform,
				Args:       argTypes,
				ReturnType: returnType,
			})
			seen[m.Transform] = true
		}
	}

	sort.Slice(missing, func(i, j int) bool {
		return missing[i].Name < missing[j].Name
	})

	return missing
}

// Helper functions

func (g *Generator) filename(pair *plan.ResolvedTypePair) string {
	src := strings.ToLower(pair.SourceType.ID.Name)
	tgt := strings.ToLower(pair.TargetType.ID.Name)
	srcPkg := path.Base(pair.SourceType.ID.PkgPath)
	tgtPkg := path.Base(pair.TargetType.ID.PkgPath)

	return fmt.Sprintf("%s_%s_to_%s_%s.go", srcPkg, src, tgtPkg, tgt)
}

func (g *Generator) functionName(pair *plan.ResolvedTypePair) string {
	srcPkg := g.capitalize(path.Base(pair.SourceType.ID.PkgPath))
	tgtPkg := g.capitalize(path.Base(pair.TargetType.ID.PkgPath))

	return fmt.Sprintf("%s%sTo%s%s",
		srcPkg, pair.SourceType.ID.Name,
		tgtPkg, pair.TargetType.ID.Name)
}

func (g *Generator) nestedFunctionName(src, tgt *analyze.TypeInfo) string {
	srcPkg := g.capitalize(path.Base(src.ID.PkgPath))
	tgtPkg := g.capitalize(path.Base(tgt.ID.PkgPath))

	return fmt.Sprintf("%s%sTo%s%s", srcPkg, src.ID.Name, tgtPkg, tgt.ID.Name)
}

func (g *Generator) pkgAlias(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}

	return path.Base(pkgPath)
}

func (g *Generator) addImport(imports map[string]importSpec, pkgPath string) {
	if pkgPath == "" {
		return
	}

	alias := g.pkgAlias(pkgPath)
	imports[pkgPath] = importSpec{
		Alias: alias,
		Path:  pkgPath,
	}
}

func (g *Generator) capitalize(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *Generator) getFieldType(typeInfo *analyze.TypeInfo, fieldPath string) *analyze.TypeInfo {
	return g.getFieldTypeInfo(typeInfo, fieldPath)
}

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

func (g *Generator) findFieldInStruct(structType *analyze.TypeInfo, fieldName string) *analyze.TypeInfo {
	for _, field := range structType.Fields {
		if field.Name == fieldName {
			return field.Type
		}
	}

	return nil
}

func (g *Generator) getFieldTypeString(
	typeInfo *analyze.TypeInfo,
	fieldPath string,
	imports map[string]importSpec,
) string {
	ft := g.getFieldTypeInfo(typeInfo, fieldPath)
	if ft == nil {
		return interfaceTypeStr
	}

	return g.typeRefString(ft, imports)
}

func (g *Generator) typeRefString(t *analyze.TypeInfo, imports map[string]importSpec) string {
	if t == nil {
		return interfaceTypeStr
	}

	switch t.Kind {
	case analyze.TypeKindBasic:
		return t.ID.Name

	case analyze.TypeKindPointer:
		if t.ElemType != nil {
			return "*" + g.typeRefString(t.ElemType, imports)
		}

		return "*" + interfaceTypeStr

	case analyze.TypeKindSlice:
		if t.ElemType != nil {
			return "[]" + g.typeRefString(t.ElemType, imports)
		}

		return "[]" + interfaceTypeStr

	case analyze.TypeKindArray:
		// Keep length information by using go/types' string.
		// This avoids having to store the array length explicitly in TypeInfo.
		return t.GoType.String()

	case analyze.TypeKindStruct, analyze.TypeKindExternal, analyze.TypeKindAlias:
		if t.ID.PkgPath != "" {
			g.addImport(imports, t.ID.PkgPath)

			return g.pkgAlias(t.ID.PkgPath) + "." + t.ID.Name
		}

		return t.ID.Name

	default:
		return "interface{}"
	}
}

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

func (g *Generator) zeroValueForType(ft *analyze.TypeInfo) string {
	switch ft.Kind {
	case analyze.TypeKindBasic:
		return g.zeroValueForBasicType(ft.ID.Name)

	case analyze.TypeKindPointer, analyze.TypeKindSlice, analyze.TypeKindArray:
		return "nil"

	case analyze.TypeKindStruct:
		return g.typeRefString(ft, nil) + "{}"

	default:
		return `""`
	}
}

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

func (g *Generator) typesIdentical(a, b *analyze.TypeInfo) bool {
	if a == nil || b == nil {
		return false
	}

	return a.ID == b.ID && a.Kind == b.Kind
}

func (g *Generator) typesConvertible(a, b *analyze.TypeInfo) bool {
	if a == nil || b == nil {
		return false
	}

	// Basic types are usually convertible among numeric types
	if a.Kind == analyze.TypeKindBasic && b.Kind == analyze.TypeKindBasic {
		return true
	}

	// Same named types
	if a.ID == b.ID {
		return true
	}

	return false
}

// Template for the caster file

var casterTemplate = template.Must(template.New("caster").Parse(`// Code generated by caster-generator. DO NOT EDIT.

package {{.PackageName}}

{{if .Imports}}
import (
{{range .Imports}}	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{end}})
{{end}}

// {{.FunctionName}} converts {{.SourceType}} to {{.TargetType}}.
func {{.FunctionName}}(in {{.SourceType}}{{range .ExtraArgs}}, {{.Name}} {{.Type}}{{end}}) {{.TargetType}} {
	out := {{.TargetType}}{}
{{range .Assignments}}
{{if .Comment}}	// {{.Comment}}
{{end}}{{if .IsSlice}}	{{.SliceBody}}
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
