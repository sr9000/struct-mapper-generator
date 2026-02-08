package plan

import (
	"errors"
	"fmt"
	"sort"

	"caster-generator/internal/analyze"
	"caster-generator/internal/diagnostic"
	"caster-generator/internal/mapping"
	"caster-generator/internal/match"
)

// ResolutionConfig holds configuration for the resolution process.
type ResolutionConfig struct {
	// MinConfidence is the minimum score for auto-accepting a match.
	MinConfidence float64
	// MinGap is the minimum score gap between top candidates for auto-accept.
	MinGap float64
	// AmbiguityThreshold marks pairs as ambiguous if within this difference.
	AmbiguityThreshold float64
	// StrictMode fails on any unresolved target fields.
	StrictMode bool
	// MaxCandidates is the maximum number of candidates to include in suggestions.
	MaxCandidates int
	// RecursiveResolve enables recursive resolution of nested struct/slice types.
	RecursiveResolve bool
	// MaxRecursionDepth limits recursion depth to prevent infinite loops (0 = unlimited).
	MaxRecursionDepth int
}

// DefaultConfig returns the default resolution configuration.
func DefaultConfig() ResolutionConfig {
	return ResolutionConfig{
		MinConfidence:      match.DefaultMinScore,
		MinGap:             match.DefaultMinGap,
		AmbiguityThreshold: match.DefaultAmbiguityThreshold,
		StrictMode:         false,
		MaxCandidates:      5,
		RecursiveResolve:   true,
		MaxRecursionDepth:  10,
	}
}

// Resolver performs the resolution pipeline.
type Resolver struct {
	graph      *analyze.TypeGraph
	mappingDef *mapping.MappingFile
	registry   *mapping.TransformRegistry
	config     ResolutionConfig
	// resolvedPairs caches already-resolved type pairs to prevent infinite recursion
	resolvedPairs map[string]*ResolvedTypePair
}

// NewResolver creates a new Resolver.
func NewResolver(
	graph *analyze.TypeGraph,
	mappingDef *mapping.MappingFile,
	config ResolutionConfig,
) *Resolver {
	var registry *mapping.TransformRegistry
	if mappingDef != nil {
		registry, _ = mapping.BuildRegistry(mappingDef, graph)
	} else {
		registry = mapping.NewTransformRegistry()
	}

	return &Resolver{
		graph:         graph,
		mappingDef:    mappingDef,
		registry:      registry,
		config:        config,
		resolvedPairs: make(map[string]*ResolvedTypePair),
	}
}

// Resolve runs the full resolution pipeline and returns a ResolvedMappingPlan.
func (r *Resolver) Resolve() (*ResolvedMappingPlan, error) {
	plan := &ResolvedMappingPlan{
		TypePairs:          []ResolvedTypePair{},
		Diagnostics:        diagnostic.Diagnostics{},
		TypeGraph:          r.graph,
		OriginalTransforms: r.mappingDef.Transforms,
	}

	if r.mappingDef == nil {
		return nil, errors.New("mapping definition is required")
	}

	// First pass: pre-create all virtual target types so they're available
	// for nested type detection and resolution
	r.preCreateVirtualTypes()

	// Process each type mapping
	for _, tm := range r.mappingDef.TypeMappings {
		resolved, err := r.resolveTypeMapping(&tm, &plan.Diagnostics)
		if err != nil {
			plan.Diagnostics.AddError("resolve_failed", err.Error(),
				fmt.Sprintf("%s->%s", tm.Source, tm.Target), "")

			continue
		}

		plan.TypePairs = append(plan.TypePairs, *resolved)
	}

	// Deduce types for 'requires' arguments from usage context
	r.deduceRequiresTypes(plan)

	// In strict mode, fail if there are unresolved targets
	if r.config.StrictMode && plan.Diagnostics.HasErrors() {
		return plan, errors.New("strict mode: resolution failed with errors")
	}

	return plan, nil
}

// resolveTypePairRecursive resolves a nested type pair.
// It first checks if there's an explicit YAML mapping for this type pair,
// and falls back to auto-matching if not.
func (r *Resolver) resolveTypePairRecursive(
	sourceType, targetType *analyze.TypeInfo,
	diags *diagnostic.Diagnostics,
	depth int,
) (*ResolvedTypePair, error) {
	if sourceType == nil || targetType == nil {
		return nil, errors.New("source or target type is nil")
	}

	typePairKey := fmt.Sprintf("%s->%s", sourceType.ID, targetType.ID)

	// Check cache first
	if cached, exists := r.resolvedPairs[typePairKey]; exists {
		return cached, nil
	}

	// Check if there's an explicit YAML mapping for this nested type pair
	if r.mappingDef != nil {
		for i := range r.mappingDef.TypeMappings {
			tm := &r.mappingDef.TypeMappings[i]
			yamlSource := mapping.ResolveTypeID(tm.Source, r.graph)
			yamlTarget := mapping.ResolveTypeID(tm.Target, r.graph)

			if yamlSource != nil && yamlTarget != nil &&
				yamlSource.ID == sourceType.ID && yamlTarget.ID == targetType.ID {
				// Found an explicit YAML mapping - use resolveTypeMapping
				return r.resolveTypeMapping(tm, diags)
			}
		}
	}

	result := &ResolvedTypePair{
		SourceType:      sourceType,
		TargetType:      targetType,
		Mappings:        []ResolvedFieldMapping{},
		UnmappedTargets: []UnmappedField{},
		NestedPairs:     []NestedConversion{},
		Requires:        nil, // No explicit requirements for auto-matched nested types
	}

	// Pre-cache to prevent infinite recursion for cyclic types
	r.resolvedPairs[typePairKey] = result

	mappedTargets := make(map[string]bool)

	// Only do auto-matching for nested types (no YAML rules available)
	r.autoMatchRemainingFields(result, sourceType, targetType, mappedTargets, diags, typePairKey)

	// Recursively detect and resolve nested conversions
	r.detectNestedConversions(result, diags, depth)

	// Sort for determinism
	r.sortMappings(result)

	return result, nil
}

// resolveTypeMapping resolves a single type mapping.
func (r *Resolver) resolveTypeMapping(
	tm *mapping.TypeMapping,
	diags *diagnostic.Diagnostics,
) (*ResolvedTypePair, error) {
	// Resolve source and target types
	sourceType := mapping.ResolveTypeID(tm.Source, r.graph)
	if sourceType == nil {
		return nil, fmt.Errorf("source type %q not found", tm.Source)
	}

	targetType := mapping.ResolveTypeID(tm.Target, r.graph)
	isGeneratedTarget := false

	if targetType == nil {
		if tm.GenerateTarget {
			// Create virtual target type
			targetType = r.createVirtualTargetType(tm, sourceType)
			isGeneratedTarget = true
		} else {
			return nil, fmt.Errorf("target type %q not found", tm.Target)
		}
	} else if tm.GenerateTarget {
		// Target type was pre-created in preCreateVirtualTypes
		isGeneratedTarget = true
	}

	typePairStr := fmt.Sprintf("%s->%s", sourceType.ID, targetType.ID)

	// Check cache first to prevent infinite recursion
	if cached, exists := r.resolvedPairs[typePairStr]; exists {
		return cached, nil
	}

	result := &ResolvedTypePair{
		SourceType:        sourceType,
		TargetType:        targetType,
		Mappings:          []ResolvedFieldMapping{},
		UnmappedTargets:   []UnmappedField{},
		NestedPairs:       []NestedConversion{},
		Requires:          tm.Requires, // Preserve requires
		IsGeneratedTarget: isGeneratedTarget,
	}

	// Pre-cache to prevent infinite recursion for cyclic types
	r.resolvedPairs[typePairStr] = result

	// Check for requires conflicts
	if conflicts := result.CheckRequireConflicts(); len(conflicts) > 0 {
		for _, conflict := range conflicts {
			diags.AddWarning("requires_conflict",
				fmt.Sprintf("required variable %q conflicts with source field", conflict),
				typePairStr, "")
		}
	}

	// Track which target fields have been mapped
	mappedTargets := make(map[string]bool)

	// Priority 1: Process 121 shorthand mappings (highest priority)
	for sourcePath, targetPath := range tm.OneToOne {
		resolved, err := r.resolve121Mapping(sourcePath, targetPath, sourceType, targetType)
		if err != nil {
			diags.AddWarning("121_mapping_error", err.Error(), typePairStr, targetPath)
			continue
		}

		result.Mappings = append(result.Mappings, *resolved)
		// Mark all target paths as mapped
		for _, tp := range resolved.TargetPaths {
			mappedTargets[tp.String()] = true
		}
	}

	// Priority 2: Process explicit field mappings
	for _, fm := range tm.Fields {
		resolved, err := r.resolveFieldMapping(&fm, sourceType, targetType, MappingSourceYAMLFields)
		if err != nil {
			diags.AddWarning("field_mapping_error", err.Error(), typePairStr, fm.Target.First())
			continue
		}
		// Check for conflicts with higher priority mappings
		for _, tp := range resolved.TargetPaths {
			if mappedTargets[tp.String()] {
				diags.AddWarning("mapping_override",
					fmt.Sprintf("field %q already mapped by higher priority rule", tp.String()),
					typePairStr, tp.String())

				continue
			}

			mappedTargets[tp.String()] = true
		}

		result.Mappings = append(result.Mappings, *resolved)
	}

	// Priority 3: Process ignore list
	for _, ignorePath := range tm.Ignore {
		if mappedTargets[ignorePath] {
			continue // Already handled by higher priority
		}

		fp, err := mapping.ParsePath(ignorePath)
		if err != nil {
			diags.AddWarning("ignore_parse_error", err.Error(), typePairStr, ignorePath)
			continue
		}

		resolved := ResolvedFieldMapping{
			TargetPaths: []mapping.FieldPath{fp},
			SourcePaths: nil,
			Source:      MappingSourceYAMLIgnore,
			Strategy:    StrategyIgnore,
			Explanation: "explicitly ignored",
		}
		result.Mappings = append(result.Mappings, resolved)
		mappedTargets[ignorePath] = true
	}

	// Priority 4: Process YAML auto mappings
	for _, fm := range tm.Auto {
		resolved, err := r.resolveFieldMapping(&fm, sourceType, targetType, MappingSourceYAMLAuto)
		if err != nil {
			diags.AddWarning("auto_mapping_error", err.Error(), typePairStr, fm.Target.First())
			continue
		}
		// Check for conflicts
		for _, tp := range resolved.TargetPaths {
			if mappedTargets[tp.String()] {
				continue // Already handled by higher priority
			}

			mappedTargets[tp.String()] = true
		}

		result.Mappings = append(result.Mappings, *resolved)
	}

	// Priority 5: Auto-match remaining target fields
	r.autoMatchRemainingFields(result, sourceType, targetType, mappedTargets, diags, typePairStr)

	// Detect nested struct conversions (with recursive resolution)
	r.detectNestedConversions(result, diags, 0)

	// Derive dependency edges from `extra.def.target` references.
	r.populateExtraTargetDependencies(result, diags)

	// Sort for determinism
	r.sortMappings(result)

	return result, nil
}

// resolve121Mapping resolves a 1:1 shorthand mapping.
func (r *Resolver) resolve121Mapping(
	sourcePath, targetPath string,
	sourceType, targetType *analyze.TypeInfo,
) (*ResolvedFieldMapping, error) {
	sp, err := mapping.ParsePath(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("invalid source path %q: %w", sourcePath, err)
	}

	tp, err := mapping.ParsePath(targetPath)
	if err != nil {
		return nil, fmt.Errorf("invalid target path %q: %w", targetPath, err)
	}

	// Determine conversion strategy based on types
	strategy, compat := r.determineStrategy(sp, tp, sourceType, targetType)

	return &ResolvedFieldMapping{
		TargetPaths: []mapping.FieldPath{tp},
		SourcePaths: []mapping.FieldPath{sp},
		Source:      MappingSourceYAML121,
		Cardinality: mapping.CardinalityOneToOne,
		Strategy:    strategy,
		Confidence:  1.0, // Explicit mappings have full confidence
		Explanation: fmt.Sprintf("explicit 121 mapping: %s -> %s (%s)", sourcePath, targetPath, compat),
	}, nil
}

// resolveFieldMapping resolves a FieldMapping from YAML.
func (r *Resolver) resolveFieldMapping(
	fm *mapping.FieldMapping,
	sourceType, targetType *analyze.TypeInfo,
	source MappingSource,
) (*ResolvedFieldMapping, error) {
	// Parse target paths
	var targetPaths []mapping.FieldPath

	for _, t := range fm.Target {
		tp, err := mapping.ParsePath(t.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid target path %q: %w", t.Path, err)
		}

		targetPaths = append(targetPaths, tp)
	}

	// Handle default value
	if fm.Default != nil {
		return &ResolvedFieldMapping{
			TargetPaths: targetPaths,
			Source:      source,
			Strategy:    StrategyDefault,
			Default:     fm.Default,
			Cardinality: mapping.CardinalityOneToOne,
			Explanation: "default value: " + *fm.Default,
			Extra:       fm.Extra,
		}, nil
	}

	// Parse source paths
	var sourcePaths []mapping.FieldPath

	for _, s := range fm.Source {
		sp, err := mapping.ParsePath(s.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid source path %q: %w", s.Path, err)
		}

		sourcePaths = append(sourcePaths, sp)
	}

	// If a transform is explicitly specified, keep StrategyTransform.
	// Otherwise, derive the strategy from source/target types so YAML field
	// mappings behave the same as auto-matched ones (pointer deref/wrap/etc).
	strategy := StrategyDirectAssign
	explanation := "field mapping: 1:1"
	cardinality := mapping.CardinalityOneToOne
	// Default hint is none; for field mappings we currently only use the first source's hint.
	hint := mapping.HintNone
	if len(fm.Source) > 0 {
		hint = fm.Source[0].Hint
	}

	if fm.Transform != "" {
		strategy = StrategyTransform
		explanation = "field mapping: 1:1 (transform)"
	} else if len(sourcePaths) > 0 && len(targetPaths) > 0 {
		st, expl := r.determineStrategyWithHint(
			sourcePaths[0],
			targetPaths[0],
			sourceType,
			targetType,
			hint,
		)
		strategy = st
		explanation = "field mapping: 1:1 (" + expl + ")"
	}

	return &ResolvedFieldMapping{
		SourcePaths:   sourcePaths,
		TargetPaths:   targetPaths,
		Source:        source,
		Cardinality:   cardinality,
		Strategy:      strategy,
		Transform:     fm.Transform,
		Confidence:    1.0,
		Explanation:   explanation,
		EffectiveHint: hint,
		Extra:         fm.Extra,
	}, nil
}

// collectionElem returns the element type for a slice or array, if applicable.
func (r *Resolver) collectionElem(t *analyze.TypeInfo) *analyze.TypeInfo {
	if t == nil {
		return nil
	}

	if (t.Kind == analyze.TypeKindSlice || t.Kind == analyze.TypeKindArray) && t.ElemType != nil {
		return t.ElemType
	}

	return nil
}

// detectNestedConversions identifies nested struct conversions needed and recursively resolves them.
func (r *Resolver) detectNestedConversions(result *ResolvedTypePair, diags *diagnostic.Diagnostics, depth int) {
	nestedMap := make(map[string]*NestedConversion)

	for _, m := range result.Mappings {
		r.analyzeMappingForNestedConversion(&m, result, nestedMap)
	}

	// Recursively resolve nested type pairs
	for key, nc := range nestedMap {
		r.resolveNestedConversion(key, nc, result, diags, depth)
	}
}

func (r *Resolver) analyzeMappingForNestedConversion(
	m *ResolvedFieldMapping,
	result *ResolvedTypePair,
	nestedMap map[string]*NestedConversion,
) {
	if m.Strategy != StrategyNestedCast && m.Strategy != StrategySliceMap {
		return
	}

	// Get the source and target types for nested conversion
	if len(m.SourcePaths) == 0 || len(m.TargetPaths) == 0 {
		return
	}

	sourceFieldType := r.resolveFieldType(m.SourcePaths[0], result.SourceType)
	targetFieldType := r.resolveFieldType(m.TargetPaths[0], result.TargetType)

	if sourceFieldType == nil || targetFieldType == nil {
		return
	}

	// For slice/array mappings, get the element types
	isSlice := m.Strategy == StrategySliceMap
	actualSourceType := sourceFieldType
	actualTargetType := targetFieldType

	if isSlice {
		if elem := r.collectionElem(sourceFieldType); elem != nil {
			actualSourceType = elem
		}

		if elem := r.collectionElem(targetFieldType); elem != nil {
			actualTargetType = elem
		}
	}

	// Handle pointer element types
	if actualSourceType.Kind == analyze.TypeKindPointer && actualSourceType.ElemType != nil {
		actualSourceType = actualSourceType.ElemType
	}

	if actualTargetType.Kind == analyze.TypeKindPointer && actualTargetType.ElemType != nil {
		actualTargetType = actualTargetType.ElemType
	}

	// Only process struct-to-struct conversions
	if actualSourceType.Kind != analyze.TypeKindStruct || actualTargetType.Kind != analyze.TypeKindStruct {
		return
	}

	key := fmt.Sprintf("%s->%s", actualSourceType.ID, actualTargetType.ID)
	if existing, ok := nestedMap[key]; ok {
		existing.ReferencedBy = append(existing.ReferencedBy, m.TargetPaths[0])
	} else {
		nestedMap[key] = &NestedConversion{
			SourceType:     actualSourceType,
			TargetType:     actualTargetType,
			ReferencedBy:   []mapping.FieldPath{m.TargetPaths[0]},
			IsSliceElement: isSlice,
		}
	}
}

func (r *Resolver) resolveNestedConversion(
	key string,
	nc *NestedConversion,
	result *ResolvedTypePair,
	diags *diagnostic.Diagnostics,
	depth int,
) {
	// Note: if key is already in the cache, we reuse it (cycle-safe).
	if cached, exists := r.resolvedPairs[key]; exists {
		nc.ResolvedPair = cached
		result.NestedPairs = append(result.NestedPairs, *nc)

		return
	}

	// Check recursion depth
	if r.config.MaxRecursionDepth > 0 && depth >= r.config.MaxRecursionDepth {
		diags.AddWarning("max_recursion_depth",
			"max recursion depth reached for "+key,
			key, "")

		result.NestedPairs = append(result.NestedPairs, *nc)

		return
	}

	// Recursively resolve if enabled
	isRecursiveResolve := r.config.RecursiveResolve
	isStructPair := nc.SourceType.Kind == analyze.TypeKindStruct &&
		nc.TargetType.Kind == analyze.TypeKindStruct

	// If we end up trying to resolve the exact same type pair as our parent, skip
	// and let cache/self-reference handle it.
	if isRecursiveResolve && isStructPair {
		parentKey := ""
		if result.SourceType != nil && result.TargetType != nil {
			parentKey = fmt.Sprintf("%s->%s", result.SourceType.ID, result.TargetType.ID)
		}

		if parentKey != "" && parentKey == key {
			diags.AddInfo("recursive_pair_self_reference",
				"detected self-referential nested struct pair; skipping recursive resolve to avoid infinite recursion",
				key, "")

			result.NestedPairs = append(result.NestedPairs, *nc)

			return
		}

		nestedResult, err := r.resolveTypePairRecursive(nc.SourceType, nc.TargetType, diags, depth+1)
		if err != nil {
			diags.AddWarning("nested_resolve_error", err.Error(), key, "")
		} else {
			nc.ResolvedPair = nestedResult
			// Cache the result
			r.resolvedPairs[key] = nestedResult
		}
	}

	result.NestedPairs = append(result.NestedPairs, *nc)
}

// sortMappings sorts mappings for deterministic output.
func (r *Resolver) sortMappings(result *ResolvedTypePair) {
	sort.Slice(result.Mappings, func(i, j int) bool {
		// First by source priority (higher priority first)
		if result.Mappings[i].Source != result.Mappings[j].Source {
			return result.Mappings[i].Source < result.Mappings[j].Source
		}
		// Then by target path alphabetically
		if len(result.Mappings[i].TargetPaths) > 0 && len(result.Mappings[j].TargetPaths) > 0 {
			return result.Mappings[i].TargetPaths[0].String() < result.Mappings[j].TargetPaths[0].String()
		}

		return false
	})

	// Sort unmapped fields
	sort.Slice(result.UnmappedTargets, func(i, j int) bool {
		return result.UnmappedTargets[i].TargetPath.String() < result.UnmappedTargets[j].TargetPath.String()
	})

	// Sort nested pairs
	sort.Slice(result.NestedPairs, func(i, j int) bool {
		iKey := fmt.Sprintf("%s->%s", result.NestedPairs[i].SourceType.ID, result.NestedPairs[i].TargetType.ID)
		jKey := fmt.Sprintf("%s->%s", result.NestedPairs[j].SourceType.ID, result.NestedPairs[j].TargetType.ID)

		return iKey < jKey
	})
}
