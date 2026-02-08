package plan

import (
	"fmt"

	"caster-generator/internal/analyze"
	"caster-generator/internal/diagnostic"
	"caster-generator/internal/mapping"
	"caster-generator/internal/match"
)

// autoMatchRemainingFields uses best-effort matching for unmapped target fields.
func (r *Resolver) autoMatchRemainingFields(
	result *ResolvedTypePair,
	sourceType, targetType *analyze.TypeInfo,
	mappedTargets map[string]bool,
	diags *diagnostic.Diagnostics,
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
