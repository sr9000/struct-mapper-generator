package plan

import (
	"fmt"
	"sort"

	"caster-generator/internal/analyze"
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
}

// DefaultConfig returns the default resolution configuration.
func DefaultConfig() ResolutionConfig {
	return ResolutionConfig{
		MinConfidence:      match.DefaultMinScore,
		MinGap:             match.DefaultMinGap,
		AmbiguityThreshold: match.DefaultAmbiguityThreshold,
		StrictMode:         false,
		MaxCandidates:      5,
	}
}

// Resolver performs the resolution pipeline.
type Resolver struct {
	graph      *analyze.TypeGraph
	mappingDef *mapping.MappingFile
	registry   *mapping.TransformRegistry
	config     ResolutionConfig
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
		graph:      graph,
		mappingDef: mappingDef,
		registry:   registry,
		config:     config,
	}
}

// Resolve runs the full resolution pipeline and returns a ResolvedMappingPlan.
func (r *Resolver) Resolve() (*ResolvedMappingPlan, error) {
	plan := &ResolvedMappingPlan{
		TypePairs:   []ResolvedTypePair{},
		Diagnostics: Diagnostics{},
	}

	if r.mappingDef == nil {
		return nil, fmt.Errorf("mapping definition is required")
	}

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

	// In strict mode, fail if there are unresolved targets
	if r.config.StrictMode && plan.Diagnostics.HasErrors() {
		return plan, fmt.Errorf("strict mode: resolution failed with errors")
	}

	return plan, nil
}

// resolveTypeMapping resolves a single type mapping.
func (r *Resolver) resolveTypeMapping(
	tm *mapping.TypeMapping,
	diags *Diagnostics,
) (*ResolvedTypePair, error) {
	// Resolve source and target types
	sourceType := mapping.ResolveTypeID(tm.Source, r.graph)
	if sourceType == nil {
		return nil, fmt.Errorf("source type %q not found", tm.Source)
	}

	targetType := mapping.ResolveTypeID(tm.Target, r.graph)
	if targetType == nil {
		return nil, fmt.Errorf("target type %q not found", tm.Target)
	}

	typePairStr := fmt.Sprintf("%s->%s", sourceType.ID, targetType.ID)

	result := &ResolvedTypePair{
		SourceType:      sourceType,
		TargetType:      targetType,
		Mappings:        []ResolvedFieldMapping{},
		UnmappedTargets: []UnmappedField{},
		NestedPairs:     []NestedConversion{},
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

	// Detect nested struct conversions
	r.detectNestedConversions(result)

	// Sort mappings for deterministic output
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
		tp, err := mapping.ParsePath(t)
		if err != nil {
			return nil, fmt.Errorf("invalid target path %q: %w", t, err)
		}
		targetPaths = append(targetPaths, tp)
	}

	// Handle ignore
	if fm.Ignore {
		return &ResolvedFieldMapping{
			TargetPaths: targetPaths,
			Source:      source,
			Strategy:    StrategyIgnore,
			Explanation: "explicitly ignored",
		}, nil
	}

	// Handle default value
	if fm.Default != nil {
		return &ResolvedFieldMapping{
			TargetPaths: targetPaths,
			Source:      source,
			Strategy:    StrategyDefault,
			Default:     fm.Default,
			Cardinality: mapping.CardinalityOneToOne,
			Explanation: fmt.Sprintf("default value: %s", *fm.Default),
		}, nil
	}

	// Parse source paths
	var sourcePaths []mapping.FieldPath
	for _, s := range fm.Source {
		sp, err := mapping.ParsePath(s)
		if err != nil {
			return nil, fmt.Errorf("invalid source path %q: %w", s, err)
		}
		sourcePaths = append(sourcePaths, sp)
	}

	cardinality := fm.GetCardinality()

	// Determine strategy
	var strategy ConversionStrategy
	var explanation string

	if fm.Transform != "" {
		strategy = StrategyTransform
		explanation = fmt.Sprintf("transform: %s", fm.Transform)
	} else if len(sourcePaths) == 1 && len(targetPaths) == 1 {
		var compat string
		strategy, compat = r.determineStrategy(sourcePaths[0], targetPaths[0], sourceType, targetType)
		explanation = fmt.Sprintf("field mapping: %s (%s)", cardinality, compat)
	} else {
		strategy = StrategyTransform
		explanation = fmt.Sprintf("multi-field mapping: %s", cardinality)
	}

	return &ResolvedFieldMapping{
		TargetPaths: targetPaths,
		SourcePaths: sourcePaths,
		Source:      source,
		Cardinality: cardinality,
		Strategy:    strategy,
		Transform:   fm.Transform,
		Confidence:  1.0,
		Explanation: explanation,
	}, nil
}

// determineStrategy determines the conversion strategy based on source and target types.
func (r *Resolver) determineStrategy(
	sourcePath, targetPath mapping.FieldPath,
	sourceType, targetType *analyze.TypeInfo,
) (ConversionStrategy, string) {
	// Get the actual field types
	sourceFieldType := r.resolveFieldType(sourcePath, sourceType)
	targetFieldType := r.resolveFieldType(targetPath, targetType)

	if sourceFieldType == nil || targetFieldType == nil {
		return StrategyTransform, "type info unavailable"
	}

	// Check type compatibility
	compat := match.ScorePointerCompatibility(sourceFieldType.GoType, targetFieldType.GoType)

	switch compat.Compatibility {
	case match.TypeIdentical:
		return StrategyDirectAssign, "identical"
	case match.TypeAssignable:
		return StrategyDirectAssign, "assignable"
	case match.TypeConvertible:
		return StrategyConvert, "convertible"
	case match.TypeNeedsTransform:
		// Determine more specific strategy
		if sourceFieldType.Kind == analyze.TypeKindPointer && targetFieldType.Kind != analyze.TypeKindPointer {
			return StrategyPointerDeref, "pointer deref"
		}
		if sourceFieldType.Kind != analyze.TypeKindPointer && targetFieldType.Kind == analyze.TypeKindPointer {
			return StrategyPointerWrap, "pointer wrap"
		}
		if sourceFieldType.Kind == analyze.TypeKindSlice && targetFieldType.Kind == analyze.TypeKindSlice {
			return StrategySliceMap, "slice map"
		}
		if sourceFieldType.Kind == analyze.TypeKindStruct && targetFieldType.Kind == analyze.TypeKindStruct {
			return StrategyNestedCast, "nested struct"
		}
		return StrategyTransform, "needs transform"
	default:
		return StrategyTransform, "incompatible"
	}
}

// resolveFieldType resolves the TypeInfo for a field at the given path.
func (r *Resolver) resolveFieldType(path mapping.FieldPath, typeInfo *analyze.TypeInfo) *analyze.TypeInfo {
	current := typeInfo

	for _, seg := range path.Segments {
		if current.Kind != analyze.TypeKindStruct {
			return nil
		}

		var found *analyze.FieldInfo
		for i := range current.Fields {
			if current.Fields[i].Name == seg.Name {
				found = &current.Fields[i]
				break
			}
		}

		if found == nil {
			return nil
		}

		current = found.Type

		if seg.IsSlice && current.Kind == analyze.TypeKindSlice {
			current = current.ElemType
		}

		// Auto-deref pointers for nested access
		if current.Kind == analyze.TypeKindPointer {
			current = current.ElemType
		}
	}

	return current
}

// autoMatchRemainingFields uses best-effort matching for unmapped target fields.
func (r *Resolver) autoMatchRemainingFields(
	result *ResolvedTypePair,
	sourceType, targetType *analyze.TypeInfo,
	mappedTargets map[string]bool,
	diags *Diagnostics,
	typePairStr string,
) {
	// Get all source fields for matching
	sourceFields := sourceType.Fields

	// Process each unmapped target field
	for i := range targetType.Fields {
		targetField := &targetType.Fields[i]

		// Skip if already mapped or unexported
		if mappedTargets[targetField.Name] || !targetField.Exported {
			continue
		}

		// Rank candidates
		candidates := match.RankCandidates(targetField, sourceFields)

		// Try to auto-match with high confidence
		if best := candidates.HighConfidence(r.config.MinConfidence, r.config.MinGap); best != nil {
			// Successful auto-match
			strategy, compat := r.determineStrategyFromCandidate(best)

			targetPath := mapping.FieldPath{
				Segments: []mapping.PathSegment{{Name: targetField.Name}},
			}
			sourcePath := mapping.FieldPath{
				Segments: []mapping.PathSegment{{Name: best.SourceField.Name}},
			}

			resolved := ResolvedFieldMapping{
				TargetPaths: []mapping.FieldPath{targetPath},
				SourcePaths: []mapping.FieldPath{sourcePath},
				Source:      MappingSourceAutoMatched,
				Cardinality: mapping.CardinalityOneToOne,
				Strategy:    strategy,
				Confidence:  best.CombinedScore,
				Explanation: fmt.Sprintf("auto-matched: %s -> %s (score: %.2f, %s)",
					best.SourceField.Name, targetField.Name, best.CombinedScore, compat),
			}

			result.Mappings = append(result.Mappings, resolved)
			mappedTargets[targetField.Name] = true
		} else {
			// Add to unmapped with candidates for suggestions
			targetPath := mapping.FieldPath{
				Segments: []mapping.PathSegment{{Name: targetField.Name}},
			}

			reason := "no high-confidence match"
			if candidates.IsAmbiguous(r.config.AmbiguityThreshold) && len(candidates) >= 2 {
				reason = fmt.Sprintf("ambiguous: top candidates %q (%.2f) and %q (%.2f) are too close",
					candidates[0].SourceField.Name, candidates[0].CombinedScore,
					candidates[1].SourceField.Name, candidates[1].CombinedScore)
			} else if len(candidates) > 0 && candidates[0].CombinedScore < r.config.MinConfidence {
				reason = fmt.Sprintf("best match %q (%.2f) below threshold %.2f",
					candidates[0].SourceField.Name, candidates[0].CombinedScore, r.config.MinConfidence)
			} else if len(candidates) == 0 {
				reason = "no compatible source fields found"
			}

			result.UnmappedTargets = append(result.UnmappedTargets, UnmappedField{
				TargetField: targetField,
				TargetPath:  targetPath,
				Candidates:  candidates.Top(r.config.MaxCandidates),
				Reason:      reason,
			})

			diags.AddWarning("unmapped_field",
				fmt.Sprintf("target field %q: %s", targetField.Name, reason),
				typePairStr, targetField.Name)
		}
	}
}

// determineStrategyFromCandidate determines the conversion strategy from a candidate match.
func (r *Resolver) determineStrategyFromCandidate(cand *match.Candidate) (ConversionStrategy, string) {
	switch cand.TypeCompat.Compatibility {
	case match.TypeIdentical:
		return StrategyDirectAssign, "identical"
	case match.TypeAssignable:
		return StrategyDirectAssign, "assignable"
	case match.TypeConvertible:
		return StrategyConvert, "convertible"
	case match.TypeNeedsTransform:
		// Check for specific strategies based on reason
		if cand.TypeCompat.Reason == "requires pointer dereference" {
			return StrategyPointerDeref, "pointer deref"
		}
		if cand.TypeCompat.Reason == "requires taking address" {
			return StrategyPointerWrap, "pointer wrap"
		}
		return StrategyTransform, cand.TypeCompat.Reason
	default:
		return StrategyTransform, "incompatible"
	}
}

// detectNestedConversions identifies nested struct conversions needed.
func (r *Resolver) detectNestedConversions(result *ResolvedTypePair) {
	nestedMap := make(map[string]*NestedConversion)

	for _, m := range result.Mappings {
		if m.Strategy == StrategyNestedCast || m.Strategy == StrategySliceMap {
			// Get the source and target types for nested conversion
			if len(m.SourcePaths) > 0 && len(m.TargetPaths) > 0 {
				sourceFieldType := r.resolveFieldType(m.SourcePaths[0], result.SourceType)
				targetFieldType := r.resolveFieldType(m.TargetPaths[0], result.TargetType)

				if sourceFieldType != nil && targetFieldType != nil {
					key := fmt.Sprintf("%s->%s", sourceFieldType.ID, targetFieldType.ID)
					if existing, ok := nestedMap[key]; ok {
						existing.ReferencedBy = append(existing.ReferencedBy, m.TargetPaths[0])
					} else {
						nestedMap[key] = &NestedConversion{
							SourceType:   sourceFieldType,
							TargetType:   targetFieldType,
							ReferencedBy: []mapping.FieldPath{m.TargetPaths[0]},
						}
					}
				}
			}
		}
	}

	for _, nc := range nestedMap {
		result.NestedPairs = append(result.NestedPairs, *nc)
	}
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
