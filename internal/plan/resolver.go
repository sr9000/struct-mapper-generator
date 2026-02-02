package plan

import (
	"errors"
	"fmt"
	"sort"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
	"caster-generator/internal/match"
)

// Strategy explanation constants.
const (
	explSliceMap     = "slice map"
	explNestedStruct = "nested struct"
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
		TypePairs:   []ResolvedTypePair{},
		Diagnostics: Diagnostics{},
	}

	if r.mappingDef == nil {
		return nil, errors.New("mapping definition is required")
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
	diags *Diagnostics,
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

	// Check cache first to prevent infinite recursion
	if cached, exists := r.resolvedPairs[typePairStr]; exists {
		return cached, nil
	}

	result := &ResolvedTypePair{
		SourceType:      sourceType,
		TargetType:      targetType,
		Mappings:        []ResolvedFieldMapping{},
		UnmappedTargets: []UnmappedField{},
		NestedPairs:     []NestedConversion{},
		Requires:        tm.Requires, // Preserve requires
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

// determineStrategy determines the conversion strategy based on source and target types.
func (r *Resolver) determineStrategy(
	sourcePath, targetPath mapping.FieldPath,
	sourceType, targetType *analyze.TypeInfo,
) (ConversionStrategy, string) {
	return r.determineStrategyWithHint(sourcePath, targetPath, sourceType, targetType, mapping.HintNone)
}

// determineStrategyWithHint determines the conversion strategy, respecting introspection hints.
func (r *Resolver) determineStrategyWithHint(
	sourcePath, targetPath mapping.FieldPath,
	sourceType, targetType *analyze.TypeInfo,
	hint mapping.IntrospectionHint,
) (ConversionStrategy, string) {
	// Get the actual field types
	sourceFieldType := r.resolveFieldType(sourcePath, sourceType)
	targetFieldType := r.resolveFieldType(targetPath, targetType)

	if sourceFieldType == nil || targetFieldType == nil {
		return StrategyTransform, "type info unavailable"
	}

	// If hint is "final", always use transform (no introspection)
	if hint == mapping.HintFinal {
		return StrategyTransform, "final (no introspection)"
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

		// Check for pointer-to-pointer struct conversions (e.g., *Node -> *NodeDTO)
		if sourceFieldType.Kind == analyze.TypeKindPointer && targetFieldType.Kind == analyze.TypeKindPointer {
			srcElem := sourceFieldType.ElemType

			tgtElem := targetFieldType.ElemType
			if srcElem != nil && tgtElem != nil &&
				srcElem.Kind == analyze.TypeKindStruct && tgtElem.Kind == analyze.TypeKindStruct {
				return StrategyPointerNestedCast, "pointer nested cast"
			}
		}

		if sourceFieldType.Kind == analyze.TypeKindSlice && targetFieldType.Kind == analyze.TypeKindSlice {
			// For slices, check if hint says dive (introspect elements) or final
			if hint == mapping.HintDive {
				return StrategySliceMap, explSliceMap + " (dive)"
			}

			return StrategySliceMap, explSliceMap
		}

		if sourceFieldType.Kind == analyze.TypeKindArray && targetFieldType.Kind == analyze.TypeKindArray {
			// Arrays are treated like slices for mapping purposes (element-wise).
			if hint == mapping.HintDive {
				return StrategySliceMap, explSliceMap + " (dive, array)"
			}

			return StrategySliceMap, explSliceMap + " (array)"
		}

		if sourceFieldType.Kind == analyze.TypeKindStruct && targetFieldType.Kind == analyze.TypeKindStruct {
			// For structs, check if hint says dive (recursively map fields) or final
			if hint == mapping.HintDive {
				return StrategyNestedCast, explNestedStruct + " (dive)"
			}
			// Default behavior: introspect structs unless marked final
			return StrategyNestedCast, explNestedStruct
		}

		return StrategyTransform, "needs transform"
	default:
		// Check for pointer-to-pointer struct conversions (e.g., *Node -> *NodeDTO)
		if sourceFieldType.Kind == analyze.TypeKindPointer && targetFieldType.Kind == analyze.TypeKindPointer {
			srcElem := sourceFieldType.ElemType

			tgtElem := targetFieldType.ElemType
			if srcElem != nil && tgtElem != nil &&
				srcElem.Kind == analyze.TypeKindStruct && tgtElem.Kind == analyze.TypeKindStruct {
				return StrategyPointerNestedCast, "pointer nested cast"
			}
		}

		// Also check for struct/slice even when marked as incompatible
		if sourceFieldType.Kind == analyze.TypeKindStruct && targetFieldType.Kind == analyze.TypeKindStruct {
			if hint == mapping.HintDive {
				return StrategyNestedCast, explNestedStruct + " (dive)"
			}

			return StrategyNestedCast, explNestedStruct
		}

		if sourceFieldType.Kind == analyze.TypeKindSlice && targetFieldType.Kind == analyze.TypeKindSlice {
			if hint == mapping.HintDive {
				return StrategySliceMap, explSliceMap + " (dive)"
			}

			return StrategySliceMap, explSliceMap
		}

		if sourceFieldType.Kind == analyze.TypeKindArray && targetFieldType.Kind == analyze.TypeKindArray {
			if hint == mapping.HintDive {
				return StrategySliceMap, explSliceMap + " (dive, array)"
			}

			return StrategySliceMap, explSliceMap + " (array)"
		}

		return StrategyTransform, "incompatible"
	}
}

// resolveFieldType resolves the TypeInfo for a field at the given path.
func (r *Resolver) resolveFieldType(path mapping.FieldPath, typeInfo *analyze.TypeInfo) *analyze.TypeInfo {
	current := typeInfo

	for i, seg := range path.Segments {
		if current.Kind != analyze.TypeKindStruct {
			return nil
		}

		var found *analyze.FieldInfo

		for j := range current.Fields {
			if current.Fields[j].Name == seg.Name {
				found = &current.Fields[j]
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

		// Auto-deref pointers only when we're stepping *through* them to reach a deeper field.
		// For leaf fields, we must preserve pointer-ness so strategies like PointerDeref can be selected.
		isLast := i == len(path.Segments)-1
		if !isLast && current.Kind == analyze.TypeKindPointer {
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
		best := candidates.HighConfidence(r.config.MinConfidence, r.config.MinGap)

		// Special case: if no high-confidence match but name matches well and both are structs/slices,
		// allow matching based on structural compatibility
		if best == nil && len(candidates) > 0 {
			topCandidate := &candidates[0]
			// Check if top candidate has high name score (>0.8) and is struct/slice to struct/slice
			if topCandidate.NameScore >= 0.8 && topCandidate.SourceField.Type != nil && topCandidate.TargetField.Type != nil {
				srcKind := topCandidate.SourceField.Type.Kind
				tgtKind := topCandidate.TargetField.Type.Kind

				// Allow struct-to-struct or slice-to-slice with good name match
				if (srcKind == analyze.TypeKindStruct && tgtKind == analyze.TypeKindStruct) ||
					(srcKind == analyze.TypeKindSlice && tgtKind == analyze.TypeKindSlice) ||
					(srcKind == analyze.TypeKindArray && tgtKind == analyze.TypeKindArray) {
					best = topCandidate
				}
			}
		}

		if best != nil {
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

			var reason string

			switch {
			case candidates.IsAmbiguous(r.config.AmbiguityThreshold) && len(candidates) >= 2:
				reason = fmt.Sprintf("ambiguous: top candidates %q (%.2f) and %q (%.2f) are too close",
					candidates[0].SourceField.Name, candidates[0].CombinedScore,
					candidates[1].SourceField.Name, candidates[1].CombinedScore)
			case len(candidates) > 0 && candidates[0].CombinedScore < r.config.MinConfidence:
				reason = fmt.Sprintf("best match %q (%.2f) below threshold %.2f",
					candidates[0].SourceField.Name, candidates[0].CombinedScore, r.config.MinConfidence)
			case len(candidates) == 0:
				reason = "no compatible source fields found"
			default:
				reason = "no high-confidence match"
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

		// Check if source and target are both structs (nested struct conversion)
		if cand.SourceField.Type != nil && cand.TargetField.Type != nil {
			srcKind := cand.SourceField.Type.Kind
			tgtKind := cand.TargetField.Type.Kind

			// Handle struct-to-struct
			if srcKind == analyze.TypeKindStruct && tgtKind == analyze.TypeKindStruct {
				return StrategyNestedCast, "nested struct"
			}

			// Handle slice-to-slice
			if srcKind == analyze.TypeKindSlice && tgtKind == analyze.TypeKindSlice {
				return StrategySliceMap, "slice map"
			}

			// Handle array-to-array
			if srcKind == analyze.TypeKindArray && tgtKind == analyze.TypeKindArray {
				return StrategySliceMap, "array map"
			}
		}

		return StrategyTransform, cand.TypeCompat.Reason
	default:
		// Also check for struct/slice even when marked as incompatible
		// (types might be different named structs which aren't directly compatible)
		if cand.SourceField.Type != nil && cand.TargetField.Type != nil {
			srcKind := cand.SourceField.Type.Kind
			tgtKind := cand.TargetField.Type.Kind

			if srcKind == analyze.TypeKindStruct && tgtKind == analyze.TypeKindStruct {
				return StrategyNestedCast, "nested struct"
			}

			if srcKind == analyze.TypeKindSlice && tgtKind == analyze.TypeKindSlice {
				return StrategySliceMap, "slice map"
			}

			if srcKind == analyze.TypeKindArray && tgtKind == analyze.TypeKindArray {
				return StrategySliceMap, "array map"
			}
		}

		return StrategyTransform, "incompatible"
	}
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

// populateExtraTargetDependencies turns `extra.def.target` references into ordering dependencies.
//
// Semantics: if a mapping for target field B uses extra.def.target = "A" (or "A.Sub")
// then B depends on the assignment for A (or A.Sub).
func (r *Resolver) populateExtraTargetDependencies(pair *ResolvedTypePair, diags *Diagnostics) {
	if pair == nil {
		return
	}

	pairKey := ""
	if pair.SourceType != nil && pair.TargetType != nil {
		pairKey = fmt.Sprintf("%s->%s", pair.SourceType.ID, pair.TargetType.ID)
	}

	// Index which mapping produces which exact target path.
	producer := make(map[string]int)

	for i := range pair.Mappings {
		m := &pair.Mappings[i]
		for _, tp := range m.TargetPaths {
			producer[tp.String()] = i
		}
	}

	for i := range pair.Mappings {
		m := &pair.Mappings[i]
		if len(m.Extra) == 0 {
			continue
		}

		deps := make(map[string]mapping.FieldPath)

		for _, ev := range m.Extra {
			if ev.Def.Target == "" {
				continue
			}

			p, err := mapping.ParsePath(ev.Def.Target)
			if err != nil {
				diags.AddWarning("extra_target_invalid",
					fmt.Sprintf("invalid extra.def.target %q: %v", ev.Def.Target, err),
					pairKey, ev.Def.Target)

				continue
			}

			// Self-dependency is always a cycle.
			for _, tp := range m.TargetPaths {
				if tp.String() == p.String() {
					diags.AddError("extra_dependency_cycle",
						fmt.Sprintf("mapping for %q depends on itself via extra.def.target", p.String()),
						pairKey, p.String())

					continue
				}
			}

			if _, ok := producer[p.String()]; !ok {
				diags.AddError("extra_dependency_missing",
					fmt.Sprintf("extra.def.target %q refers to a target field with no assignment", p.String()),
					pairKey, p.String())

				continue
			}

			deps[p.String()] = p
		}

		if len(deps) == 0 {
			continue
		}

		m.DependsOnTargets = m.DependsOnTargets[:0]
		for _, p := range deps {
			m.DependsOnTargets = append(m.DependsOnTargets, p)
		}

		sort.Slice(m.DependsOnTargets, func(a, b int) bool {
			return m.DependsOnTargets[a].String() < m.DependsOnTargets[b].String()
		})
	}
}

// detectNestedConversions identifies nested struct conversions needed and recursively resolves them.
func (r *Resolver) detectNestedConversions(result *ResolvedTypePair, diags *Diagnostics, depth int) {
	nestedMap := make(map[string]*NestedConversion)

	for _, m := range result.Mappings {
		if m.Strategy == StrategyNestedCast || m.Strategy == StrategySliceMap {
			// Get the source and target types for nested conversion
			if len(m.SourcePaths) > 0 && len(m.TargetPaths) > 0 {
				sourceFieldType := r.resolveFieldType(m.SourcePaths[0], result.SourceType)
				targetFieldType := r.resolveFieldType(m.TargetPaths[0], result.TargetType)

				if sourceFieldType != nil && targetFieldType != nil {
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
						continue
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
			}
		}
	}

	// Recursively resolve nested type pairs
	for key, nc := range nestedMap {
		// Note: if key is already in the cache, we reuse it (cycle-safe).
		if cached, exists := r.resolvedPairs[key]; exists {
			nc.ResolvedPair = cached
			result.NestedPairs = append(result.NestedPairs, *nc)

			continue
		}

		// Check recursion depth
		if r.config.MaxRecursionDepth > 0 && depth >= r.config.MaxRecursionDepth {
			diags.AddWarning("max_recursion_depth",
				"max recursion depth reached for "+key,
				key, "")

			result.NestedPairs = append(result.NestedPairs, *nc)

			continue
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

				continue
			}
		}

		if isRecursiveResolve && isStructPair {
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
