package match

import (
	"go/types"
	"sort"

	"caster-generator/internal/analyze"
)

// Candidate represents a potential mapping from a source field to a target field.
type Candidate struct {
	SourceField *analyze.FieldInfo
	TargetField *analyze.FieldInfo

	// Scoring components
	NameScore  float64                 // Normalized Levenshtein similarity (0-1)
	TypeCompat TypeCompatibilityResult // Type compatibility result

	// Combined score for ranking (higher is better)
	CombinedScore float64

	// Metadata for debugging/explanation
	NormalizedSourceName string
	NormalizedTargetName string
}

// CandidateList is a list of candidates with ranking functionality.
type CandidateList []Candidate

// RankCandidates finds and ranks potential source field matches for a target field.
// Returns candidates sorted by combined score (descending).
func RankCandidates(
	targetField *analyze.FieldInfo,
	sourceFields []analyze.FieldInfo,
) CandidateList {
	var candidates CandidateList

	targetNorm := NormalizeIdent(targetField.Name)
	targetNormStripped := NormalizeIdentWithSuffixStrip(targetField.Name)

	for i := range sourceFields {
		sourceField := &sourceFields[i]

		// Skip unexported fields
		if !sourceField.Exported {
			continue
		}

		sourceNorm := NormalizeIdent(sourceField.Name)
		sourceNormStripped := NormalizeIdentWithSuffixStrip(sourceField.Name)

		// Calculate name similarity (use max of regular and suffix-stripped)
		nameScore := LevenshteinNormalized(sourceNorm, targetNorm)
		nameScoreStripped := LevenshteinNormalized(sourceNormStripped, targetNormStripped)
		if nameScoreStripped > nameScore {
			nameScore = nameScoreStripped
		}

		// Check type compatibility
		var typeCompat TypeCompatibilityResult
		if sourceField.Type != nil && sourceField.Type.GoType != nil &&
			targetField.Type != nil && targetField.Type.GoType != nil {
			typeCompat = ScorePointerCompatibility(
				sourceField.Type.GoType,
				targetField.Type.GoType,
			)
		} else {
			typeCompat = TypeCompatibilityResult{
				Compatibility: TypeIncompatible,
				Reason:        "type information unavailable",
			}
		}

		// Calculate combined score
		combinedScore := calculateCombinedScore(nameScore, typeCompat.Compatibility)

		candidates = append(candidates, Candidate{
			SourceField:          sourceField,
			TargetField:          targetField,
			NameScore:            nameScore,
			TypeCompat:           typeCompat,
			CombinedScore:        combinedScore,
			NormalizedSourceName: sourceNorm,
			NormalizedTargetName: targetNorm,
		})
	}

	// Sort by combined score (descending), then by name for determinism
	sort.Sort(candidates)

	return candidates
}

// RankCandidatesWithTypes ranks candidates using types.Type directly
// (useful when you don't have full analyze.FieldInfo).
func RankCandidatesWithTypes(
	targetName string,
	targetType types.Type,
	sourceFields []struct {
		Name string
		Type types.Type
	},
) CandidateList {
	var candidates CandidateList

	targetNorm := NormalizeIdent(targetName)
	targetNormStripped := NormalizeIdentWithSuffixStrip(targetName)

	for _, source := range sourceFields {
		sourceNorm := NormalizeIdent(source.Name)
		sourceNormStripped := NormalizeIdentWithSuffixStrip(source.Name)

		// Calculate name similarity
		nameScore := LevenshteinNormalized(sourceNorm, targetNorm)
		nameScoreStripped := LevenshteinNormalized(sourceNormStripped, targetNormStripped)
		if nameScoreStripped > nameScore {
			nameScore = nameScoreStripped
		}

		// Check type compatibility
		var typeCompat TypeCompatibilityResult
		if source.Type != nil && targetType != nil {
			typeCompat = ScorePointerCompatibility(source.Type, targetType)
		} else {
			typeCompat = TypeCompatibilityResult{
				Compatibility: TypeIncompatible,
				Reason:        "type information unavailable",
			}
		}

		combinedScore := calculateCombinedScore(nameScore, typeCompat.Compatibility)

		candidates = append(candidates, Candidate{
			SourceField: &analyze.FieldInfo{
				Name:     source.Name,
				Exported: true,
			},
			TargetField: &analyze.FieldInfo{
				Name:     targetName,
				Exported: true,
			},
			NameScore:            nameScore,
			TypeCompat:           typeCompat,
			CombinedScore:        combinedScore,
			NormalizedSourceName: sourceNorm,
			NormalizedTargetName: targetNorm,
		})
	}

	sort.Sort(candidates)

	return candidates
}

// calculateCombinedScore computes a combined score from name similarity and type compatibility.
// Weights:
//   - Name similarity: 60% (0.0-0.6)
//   - Type compatibility: 40% (0.0-0.4)
func calculateCombinedScore(nameScore float64, typeCompat TypeCompatibility) float64 {
	const (
		nameWeight = 0.6
		typeWeight = 0.4
	)

	// Normalize type compatibility to 0-1 range
	var typeScore float64
	switch typeCompat {
	case TypeIdentical:
		typeScore = 1.0
	case TypeAssignable:
		typeScore = 0.9
	case TypeConvertible:
		typeScore = 0.7
	case TypeNeedsTransform:
		typeScore = 0.4
	case TypeIncompatible:
		typeScore = 0.0
	}

	return nameScore*nameWeight + typeScore*typeWeight
}

// Len implements sort.Interface.
func (c CandidateList) Len() int { return len(c) }

// Swap implements sort.Interface.
func (c CandidateList) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

// Less implements sort.Interface.
// Sorts by combined score descending, then by source field name for determinism.
func (c CandidateList) Less(i, j int) bool {
	// Higher score comes first
	if c[i].CombinedScore != c[j].CombinedScore {
		return c[i].CombinedScore > c[j].CombinedScore
	}
	// Tie-breaker: alphabetical by source field name
	return c[i].SourceField.Name < c[j].SourceField.Name
}

// Top returns the top n candidates.
func (c CandidateList) Top(n int) CandidateList {
	if n >= len(c) {
		return c
	}
	return c[:n]
}

// Best returns the best candidate, or nil if no candidates.
func (c CandidateList) Best() *Candidate {
	if len(c) == 0 {
		return nil
	}
	return &c[0]
}

// IsAmbiguous returns true if the top two candidates are within the threshold.
func (c CandidateList) IsAmbiguous(threshold float64) bool {
	if len(c) < 2 {
		return false
	}
	diff := c[0].CombinedScore - c[1].CombinedScore
	return diff < threshold
}

// AboveThreshold returns candidates with combined score above the threshold.
func (c CandidateList) AboveThreshold(threshold float64) CandidateList {
	var result CandidateList
	for _, cand := range c {
		if cand.CombinedScore >= threshold {
			result = append(result, cand)
		}
	}
	return result
}

// HighConfidence returns the best candidate if it's significantly better than alternatives.
// Returns nil if no clear winner exists.
func (c CandidateList) HighConfidence(minScore, minGap float64) *Candidate {
	if len(c) == 0 {
		return nil
	}
	best := &c[0]

	// Must meet minimum score threshold
	if best.CombinedScore < minScore {
		return nil
	}

	// Must be compatible (at least needs transform)
	if best.TypeCompat.Compatibility < TypeNeedsTransform {
		return nil
	}

	// If there's a second candidate, must have sufficient gap
	if len(c) > 1 {
		gap := c[0].CombinedScore - c[1].CombinedScore
		if gap < minGap {
			return nil
		}
	}

	return best
}

// Confidence thresholds for auto-accepting matches.
const (
	// DefaultMinScore is the minimum combined score for auto-acceptance.
	DefaultMinScore = 0.7
	// DefaultMinGap is the minimum score gap between top candidates.
	DefaultMinGap = 0.15
	// DefaultAmbiguityThreshold is the score difference that marks ambiguity.
	DefaultAmbiguityThreshold = 0.1
)
