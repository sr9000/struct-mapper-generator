package plan

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"caster-generator/internal/mapping"
)

// ExportConfig holds configuration for exporting suggestions.
type ExportConfig struct {
	// MinConfidence is the threshold used during matching (for comments).
	MinConfidence float64
	// MinGap is the gap threshold used during matching (for comments).
	MinGap float64
	// AmbiguityThreshold is the ambiguity threshold used (for comments).
	AmbiguityThreshold float64
	// IncludeRejectedComments adds comments explaining why fields were rejected.
	IncludeRejectedComments bool
}

// DefaultExportConfig returns default export configuration.
func DefaultExportConfig() ExportConfig {
	return ExportConfig{
		MinConfidence:           0.7,
		MinGap:                  0.15,
		AmbiguityThreshold:      0.1,
		IncludeRejectedComments: true,
	}
}

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
	return ExportSuggestionsYAMLWithConfig(plan, DefaultExportConfig())
}

// ExportSuggestionsYAMLWithConfig generates suggested YAML with configuration options.
func ExportSuggestionsYAMLWithConfig(plan *ResolvedMappingPlan, config ExportConfig) ([]byte, error) {
	mf, err := ExportSuggestions(plan)
	if err != nil {
		return nil, err
	}

	// Build YAML with comments using yaml.Node
	root := &yaml.Node{Kind: yaml.MappingNode}

	// Add version
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "version"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: mf.Version},
	)

	// Add mappings
	mappingsKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "mappings"}
	mappingsValue := &yaml.Node{Kind: yaml.SequenceNode}

	for i, tm := range mf.TypeMappings {
		// Find corresponding resolved type pair for comments
		var resolvedTP *ResolvedTypePair
		for j := range plan.TypePairs {
			tp := &plan.TypePairs[j]
			if tp.SourceType.ID.String() == tm.Source && tp.TargetType.ID.String() == tm.Target {
				resolvedTP = tp
				break
			}
		}
		// Also check nested pairs recursively
		if resolvedTP == nil {
			resolvedTP = findResolvedTypePair(plan, tm.Source, tm.Target)
		}

		tmNode := buildTypeMappingNode(&mf.TypeMappings[i], resolvedTP, config)
		mappingsValue.Content = append(mappingsValue.Content, tmNode)
	}

	root.Content = append(root.Content, mappingsKey, mappingsValue)

	// Add transforms if present
	if len(mf.Transforms) > 0 {
		transformsKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "transforms"}
		transformsValue := &yaml.Node{Kind: yaml.SequenceNode}
		for _, td := range mf.Transforms {
			tdNode := &yaml.Node{Kind: yaml.MappingNode}
			tdNode.Content = append(tdNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "name"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: td.Name},
			)
			if td.Func != "" {
				tdNode.Content = append(tdNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "func"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: td.Func},
				)
			}
			transformsValue.Content = append(transformsValue.Content, tdNode)
		}
		root.Content = append(root.Content, transformsKey, transformsValue)
	}

	return yaml.Marshal(root)
}

// findResolvedTypePair recursively finds a resolved type pair by source and target IDs.
func findResolvedTypePair(plan *ResolvedMappingPlan, source, target string) *ResolvedTypePair {
	for i := range plan.TypePairs {
		tp := &plan.TypePairs[i]
		if tp.SourceType.ID.String() == source && tp.TargetType.ID.String() == target {
			return tp
		}
		// Check nested pairs
		for j := range tp.NestedPairs {
			np := &tp.NestedPairs[j]
			if np.ResolvedPair != nil {
				if np.ResolvedPair.SourceType.ID.String() == source && np.ResolvedPair.TargetType.ID.String() == target {
					return np.ResolvedPair
				}
			}
		}
	}
	return nil
}

// buildTypeMappingNode builds a yaml.Node for a TypeMapping with comments.
func buildTypeMappingNode(tm *mapping.TypeMapping, resolvedTP *ResolvedTypePair, config ExportConfig) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}

	// source
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "source"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: tm.Source},
	)

	// target
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "target"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: tm.Target},
	)

	// requires
	if len(tm.Requires) > 0 {
		requiresKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "requires"}
		requiresValue := &yaml.Node{Kind: yaml.SequenceNode}
		for _, req := range tm.Requires {
			reqNode := &yaml.Node{Kind: yaml.MappingNode}
			reqNode.Content = append(reqNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "name"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: req.Name},
			)
			if req.Type != "" {
				reqNode.Content = append(reqNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "type"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: req.Type},
				)
			}
			requiresValue.Content = append(requiresValue.Content, reqNode)
		}
		node.Content = append(node.Content, requiresKey, requiresValue)
	}

	// 121
	if len(tm.OneToOne) > 0 {
		oneToOneKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "121"}
		oneToOneValue := &yaml.Node{Kind: yaml.MappingNode}
		for src, tgt := range tm.OneToOne {
			oneToOneValue.Content = append(oneToOneValue.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: src},
				&yaml.Node{Kind: yaml.ScalarNode, Value: tgt},
			)
		}
		node.Content = append(node.Content, oneToOneKey, oneToOneValue)
	}

	// fields
	if len(tm.Fields) > 0 {
		fieldsKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "fields"}
		fieldsValue := &yaml.Node{Kind: yaml.SequenceNode}
		for _, fm := range tm.Fields {
			fmNode := buildFieldMappingNode(&fm)
			fieldsValue.Content = append(fieldsValue.Content, fmNode)
		}
		node.Content = append(node.Content, fieldsKey, fieldsValue)
	}

	// ignore - add comments for why fields were ignored/rejected
	if len(tm.Ignore) > 0 || (resolvedTP != nil && len(resolvedTP.UnmappedTargets) > 0 && config.IncludeRejectedComments) {
		ignoreKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "ignore"}
		ignoreValue := &yaml.Node{Kind: yaml.SequenceNode}

		// Add header comment with threshold info
		if config.IncludeRejectedComments && resolvedTP != nil && len(resolvedTP.UnmappedTargets) > 0 {
			ignoreKey.HeadComment = fmt.Sprintf("# Thresholds: min_confidence=%.2f, min_gap=%.2f, ambiguity=%.2f",
				config.MinConfidence, config.MinGap, config.AmbiguityThreshold)
		}

		for _, ignorePath := range tm.Ignore {
			ignoreNode := &yaml.Node{Kind: yaml.ScalarNode, Value: ignorePath}

			// Find the corresponding unmapped field for comment
			if config.IncludeRejectedComments && resolvedTP != nil {
				for _, um := range resolvedTP.UnmappedTargets {
					if um.TargetPath.String() == ignorePath {
						// Build comment with rejection reason and candidates
						var commentParts []string
						commentParts = append(commentParts, um.Reason)

						if len(um.Candidates) > 0 {
							commentParts = append(commentParts, "Candidates:")
							for i, c := range um.Candidates {
								if i >= 3 { // Limit to top 3 in comment
									break
								}
								commentParts = append(commentParts,
									fmt.Sprintf("  %d. %s (score=%.2f, type=%s)",
										i+1, c.SourceField.Name, c.CombinedScore, c.TypeCompat.Compatibility.String()))
							}
						}
						ignoreNode.LineComment = "# " + strings.Join(commentParts, "; ")
						break
					}
				}
			}

			ignoreValue.Content = append(ignoreValue.Content, ignoreNode)
		}
		node.Content = append(node.Content, ignoreKey, ignoreValue)
	}

	// auto - add comments with confidence scores
	if len(tm.Auto) > 0 {
		autoKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "auto"}
		autoValue := &yaml.Node{Kind: yaml.SequenceNode}
		for _, fm := range tm.Auto {
			fmNode := buildFieldMappingNode(&fm)

			// Add confidence comment if we have the resolved mapping info
			if resolvedTP != nil {
				for _, m := range resolvedTP.Mappings {
					if m.Source == MappingSourceAutoMatched &&
						len(m.SourcePaths) > 0 && len(m.TargetPaths) > 0 &&
						len(fm.Source) > 0 && m.SourcePaths[0].String() == fm.Source[0].Path {
						fmNode.LineComment = fmt.Sprintf("# confidence=%.2f, strategy=%s",
							m.Confidence, m.Strategy.String())
						break
					}
				}
			}

			autoValue.Content = append(autoValue.Content, fmNode)
		}
		node.Content = append(node.Content, autoKey, autoValue)
	}

	return node
}

// buildFieldMappingNode builds a yaml.Node for a FieldMapping.
func buildFieldMappingNode(fm *mapping.FieldMapping) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}

	// source
	if len(fm.Source) > 0 {
		sourceKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "source"}
		var sourceValue *yaml.Node
		if len(fm.Source) == 1 && fm.Source[0].Hint == mapping.HintNone {
			sourceValue = &yaml.Node{Kind: yaml.ScalarNode, Value: fm.Source[0].Path}
		} else {
			sourceValue = buildFieldRefArrayNode(fm.Source)
		}
		node.Content = append(node.Content, sourceKey, sourceValue)
	}

	// target
	if len(fm.Target) > 0 {
		targetKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "target"}
		var targetValue *yaml.Node
		if len(fm.Target) == 1 && fm.Target[0].Hint == mapping.HintNone {
			targetValue = &yaml.Node{Kind: yaml.ScalarNode, Value: fm.Target[0].Path}
		} else {
			targetValue = buildFieldRefArrayNode(fm.Target)
		}
		node.Content = append(node.Content, targetKey, targetValue)
	}

	// transform
	if fm.Transform != "" {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "transform"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: fm.Transform},
		)
	}

	// default
	if fm.Default != nil {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "default"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: *fm.Default},
		)
	}

	// extra
	if len(fm.Extra) > 0 {
		extraKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "extra"}
		extraValue := &yaml.Node{Kind: yaml.SequenceNode}
		for _, ev := range fm.Extra {
			evNode := &yaml.Node{Kind: yaml.MappingNode}
			evNode.Content = append(evNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "name"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: ev.Name},
			)

			// mapping.ExtraDef is a value type (not a pointer). Only emit `def` if something is set.
			if ev.Def.Source != "" || ev.Def.Target != "" {
				defNode := &yaml.Node{Kind: yaml.MappingNode}
				if ev.Def.Source != "" {
					defNode.Content = append(defNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "source"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: ev.Def.Source},
					)
				}
				if ev.Def.Target != "" {
					defNode.Content = append(defNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "target"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: ev.Def.Target},
					)
				}
				evNode.Content = append(evNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "def"},
					defNode,
				)
			}

			extraValue.Content = append(extraValue.Content, evNode)
		}
		node.Content = append(node.Content, extraKey, extraValue)
	}

	return node
}

// buildFieldRefArrayNode builds a yaml.Node for FieldRefArray.
func buildFieldRefArrayNode(refs mapping.FieldRefArray) *yaml.Node {
	if len(refs) == 1 {
		if refs[0].Hint == mapping.HintNone {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: refs[0].Path}
		}
		// Single ref with hint: {Path: hint}
		node := &yaml.Node{Kind: yaml.MappingNode}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: refs[0].Path},
			&yaml.Node{Kind: yaml.ScalarNode, Value: string(refs[0].Hint)},
		)
		return node
	}

	// Multiple refs
	node := &yaml.Node{Kind: yaml.SequenceNode}
	for _, ref := range refs {
		if ref.Hint == mapping.HintNone {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: ref.Path})
		} else {
			refNode := &yaml.Node{Kind: yaml.MappingNode}
			refNode.Content = append(refNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: ref.Path},
				&yaml.Node{Kind: yaml.ScalarNode, Value: string(ref.Hint)},
			)
			node.Content = append(node.Content, refNode)
		}
	}
	return node
} // exportTypePairSuggestions exports a single type pair as a TypeMapping.
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
