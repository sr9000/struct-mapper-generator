package plan

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"caster-generator/internal/mapping"
)

// ExportSuggestions generates a suggested YAML mapping file from a resolved plan.
// This allows users to review and approve auto-matched mappings.
func ExportSuggestions(plan *ResolvedMappingPlan) (*mapping.MappingFile, error) {
	mf := &mapping.MappingFile{
		Version:      "1",
		TypeMappings: []mapping.TypeMapping{},
		Transforms:   []mapping.TransformDef{},
	}

	// Track already exported type pairs to avoid duplicates
	exported := make(map[string]bool)

	for _, tp := range plan.TypePairs {
		exportTypePairWithNested(&tp, mf, exported)
	}

	return mf, nil
}

// exportTypePairWithNested exports a type pair and recursively exports its nested pairs.
func exportTypePairWithNested(tp *ResolvedTypePair, mf *mapping.MappingFile, exported map[string]bool) {
	key := tp.SourceType.ID.String() + "->" + tp.TargetType.ID.String()
	if exported[key] {
		return
	}

	exported[key] = true

	tm := exportTypePairSuggestions(tp)
	mf.TypeMappings = append(mf.TypeMappings, tm)

	// Recursively export nested pairs
	for _, np := range tp.NestedPairs {
		if np.ResolvedPair != nil {
			exportTypePairWithNested(np.ResolvedPair, mf, exported)
		}
	}
}

// ExportSuggestionsYAML generates suggested YAML as a byte slice.
func ExportSuggestionsYAML(plan *ResolvedMappingPlan) ([]byte, error) {
	mf, err := ExportSuggestions(plan)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(mf)
}

// exportTypePairSuggestions exports a single type pair as a TypeMapping.
func exportTypePairSuggestions(tp *ResolvedTypePair) mapping.TypeMapping {
	tm := mapping.TypeMapping{
		Source:   tp.SourceType.ID.String(),
		Target:   tp.TargetType.ID.String(),
		Requires: tp.Requires, // Preserve requires
		OneToOne: make(map[string]string),
		Fields:   []mapping.FieldMapping{},
		Ignore:   []string{},
		Auto:     []mapping.FieldMapping{},
	}

	// Restore Requires field from original mapping if available
	// Note: The current plan model doesn't store the original 'requires' field.
	// We need to update the plan model or the resolver to pass this through.
	// For now, if we had access to the original mapping definition, we could copy it.

	// TODO: Pass Requires and Extra fields through ResolvedTypePair/ResolvedFieldMapping

	for _, m := range tp.Mappings {
		switch m.Source {
		case MappingSourceYAML121:
			// Check if this 121 mapping has incompatible types (needs transform)
			if m.Strategy == StrategyTransform && m.Transform == "" {
				// Move to fields section with a placeholder transform
				fm := exportFieldMapping(&m)
				fm.Transform = generatePlaceholderTransformName(m.SourcePaths, m.TargetPaths)
				tm.Fields = append(tm.Fields, fm)
			} else {
				// Preserve as 121 mappings
				if len(m.SourcePaths) == 1 && len(m.TargetPaths) == 1 {
					tm.OneToOne[m.SourcePaths[0].String()] = m.TargetPaths[0].String()
				}
			}

		case MappingSourceYAMLFields:
			// Preserve explicit fields
			fm := exportFieldMapping(&m)
			// If this field mapping needs a transform but doesn't have one, add a placeholder
			if m.Strategy == StrategyTransform && m.Transform == "" && fm.Transform == "" {
				fm.Transform = generatePlaceholderTransformName(m.SourcePaths, m.TargetPaths)
			}
			tm.Fields = append(tm.Fields, fm)

		case MappingSourceYAMLIgnore, MappingSourceYAMLAuto:
			// Keep these as-is
			if m.Strategy == StrategyIgnore {
				for _, tp := range m.TargetPaths {
					tm.Ignore = append(tm.Ignore, tp.String())
				}
			} else {
				fm := exportFieldMapping(&m)
				tm.Auto = append(tm.Auto, fm)
			}

		case MappingSourceAutoMatched:
			// Put auto-matched into the auto section with comments
			fm := exportFieldMapping(&m)
			// Add comment with confidence info
			fm.Transform = "" // Clear any generated transform name
			tm.Auto = append(tm.Auto, fm)
		}
	}

	// Add unmapped fields as ignored - user can review and move to fields if needed
	for _, um := range tp.UnmappedTargets {
		tm.Ignore = append(tm.Ignore, um.TargetPath.String())
	}

	return tm
}

// generatePlaceholderTransformName creates a placeholder transform function name
// based on the source and target field names.
func generatePlaceholderTransformName(sourcePaths []mapping.FieldPath, targetPaths []mapping.FieldPath) string {
	sourceName := "Source"
	targetName := "Target"

	if len(sourcePaths) > 0 {
		sourceName = sourcePaths[0].String()
	}
	if len(targetPaths) > 0 {
		targetName = targetPaths[0].String()
	}

	// Create a descriptive placeholder name
	return fmt.Sprintf("TODO_%sTo%s", sourceName, targetName)
}

// exportFieldMapping converts a ResolvedFieldMapping to a mapping.FieldMapping.
func exportFieldMapping(m *ResolvedFieldMapping) mapping.FieldMapping {
	fm := mapping.FieldMapping{}

	// Set sources with hints
	if len(m.SourcePaths) > 0 {
		sources := make(mapping.FieldRefArray, len(m.SourcePaths))

		for i, sp := range m.SourcePaths {
			hint := mapping.HintNone
			// Apply effective hint to the first source for 1:N or 1:1
			if i == 0 && m.EffectiveHint != mapping.HintNone {
				if m.Cardinality == mapping.CardinalityOneToMany || m.Cardinality == mapping.CardinalityOneToOne {
					hint = m.EffectiveHint
				}
			}

			sources[i] = mapping.FieldRef{Path: sp.String(), Hint: hint}
		}

		fm.Source = sources
	}

	// Set targets with hints
	targets := make(mapping.FieldRefArray, len(m.TargetPaths))

	for i, tp := range m.TargetPaths {
		hint := mapping.HintNone
		// Apply effective hint to the first target (following cardinality rules)
		if i == 0 && m.EffectiveHint != mapping.HintNone {
			// For N:1, hint goes on target; for others, we'll put it on first item
			if m.Cardinality == mapping.CardinalityManyToOne || m.Cardinality == mapping.CardinalityOneToOne {
				hint = m.EffectiveHint
			}
		}

		targets[i] = mapping.FieldRef{Path: tp.String(), Hint: hint}
	}

	fm.Target = targets

	// Set default
	if m.Default != nil {
		fm.Default = m.Default
	}

	// Set transform
	if m.Transform != "" {
		fm.Transform = m.Transform
	}

	// Set extra
	if len(m.Extra) > 0 {
		fm.Extra = m.Extra
	}

	return fm
}

// SuggestionReport generates a human-readable report of suggestions.
type SuggestionReport struct {
	TypePairs []TypePairReport
}

// TypePairReport contains suggestions for a single type pair.
type TypePairReport struct {
	Source        string
	Target        string
	AutoMatched   []MatchReport
	Unmapped      []UnmappedReport
	ExplicitCount int
	IgnoredCount  int
	NeedsReview   bool
}

// MatchReport describes an auto-matched field.
type MatchReport struct {
	SourceField string
	TargetField string
	Confidence  float64
	Strategy    string
	Explanation string
}

// UnmappedReport describes an unmapped field with suggestions.
type UnmappedReport struct {
	TargetField string
	Reason      string
	Candidates  []CandidateReport
}

// CandidateReport describes a potential match candidate.
type CandidateReport struct {
	SourceField string
	Score       float64
	TypeCompat  string
}

// GenerateReport creates a suggestion report from a resolved plan.
func GenerateReport(plan *ResolvedMappingPlan) *SuggestionReport {
	report := &SuggestionReport{
		TypePairs: []TypePairReport{},
	}

	for _, tp := range plan.TypePairs {
		tpr := TypePairReport{
			Source:      tp.SourceType.ID.String(),
			Target:      tp.TargetType.ID.String(),
			AutoMatched: []MatchReport{},
			Unmapped:    []UnmappedReport{},
		}

		for _, m := range tp.Mappings {
			switch m.Source {
			case MappingSourceYAML121, MappingSourceYAMLFields, MappingSourceYAMLAuto:
				tpr.ExplicitCount++
			case MappingSourceYAMLIgnore:
				tpr.IgnoredCount++
			case MappingSourceAutoMatched:
				if len(m.SourcePaths) > 0 && len(m.TargetPaths) > 0 {
					tpr.AutoMatched = append(tpr.AutoMatched, MatchReport{
						SourceField: m.SourcePaths[0].String(),
						TargetField: m.TargetPaths[0].String(),
						Confidence:  m.Confidence,
						Strategy:    m.Strategy.String(),
						Explanation: m.Explanation,
					})
				}
			}
		}

		for _, um := range tp.UnmappedTargets {
			umr := UnmappedReport{
				TargetField: um.TargetPath.String(),
				Reason:      um.Reason,
				Candidates:  []CandidateReport{},
			}

			for _, c := range um.Candidates {
				umr.Candidates = append(umr.Candidates, CandidateReport{
					SourceField: c.SourceField.Name,
					Score:       c.CombinedScore,
					TypeCompat:  c.TypeCompat.Compatibility.String(),
				})
			}

			tpr.Unmapped = append(tpr.Unmapped, umr)
		}

		tpr.NeedsReview = len(tpr.Unmapped) > 0

		report.TypePairs = append(report.TypePairs, tpr)
	}

	return report
}

// FormatReport formats a suggestion report as human-readable text.
func FormatReport(report *SuggestionReport) string {
	var result string

	var (
		resultSb250 strings.Builder
		resultSb258 strings.Builder
		resultSb259 strings.Builder
	)

	for _, tp := range report.TypePairs {
		resultSb250.WriteString(fmt.Sprintf("\n=== %s -> %s ===\n", tp.Source, tp.Target))
		resultSb250.WriteString(fmt.Sprintf("Explicit: %d, Ignored: %d, Auto-mapped: %d, Unmapped: %d\n",
			tp.ExplicitCount, tp.IgnoredCount, len(tp.AutoMatched), len(tp.Unmapped)))

		if len(tp.AutoMatched) > 0 {
			resultSb250.WriteString("\nAuto-mapped fields:\n")

			var resultSb257 strings.Builder
			for _, m := range tp.AutoMatched {
				resultSb257.WriteString(fmt.Sprintf("  ✓ %s -> %s (%.0f%%, %s)\n",
					m.SourceField, m.TargetField, m.Confidence*100, m.Strategy))
			}

			resultSb258.WriteString(resultSb257.String())
		}

		if len(tp.Unmapped) > 0 {
			resultSb250.WriteString("\nUnmapped target fields (need review):\n")

			var (
				resultSb265 strings.Builder
				resultSb276 strings.Builder
			)

			for _, um := range tp.Unmapped {
				resultSb265.WriteString(fmt.Sprintf("  ✗ %s: %s\n", um.TargetField, um.Reason))

				if len(um.Candidates) > 0 {
					resultSb265.WriteString("    Suggestions:\n")

					var resultSb269 strings.Builder
					for i, c := range um.Candidates {
						resultSb269.WriteString(fmt.Sprintf("      %d. %s (%.0f%%, %s)\n",
							i+1, c.SourceField, c.Score*100, c.TypeCompat))
					}

					resultSb276.WriteString(resultSb269.String())
				}
			}

			resultSb259.WriteString(resultSb276.String())

			resultSb258.WriteString(resultSb265.String())
		}

		if tp.NeedsReview {
			resultSb250.WriteString("\n⚠ This type pair needs manual review.\n")
		} else {
			resultSb250.WriteString("\n✓ All target fields mapped.\n")
		}
	}

	result += resultSb259.String()

	result += resultSb258.String()

	result += resultSb250.String()

	return result
}
