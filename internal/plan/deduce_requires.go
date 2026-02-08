package plan

import (
	"fmt"
	"go/types"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

// DeducedType represents a type deduced from usage context.
type DeducedType struct {
	TypeStr string
	Source  string // Description of where this deduction came from
}

type pairUsage struct {
	Parent       *ResolvedTypePair
	ReferencedBy []mapping.FieldPath
}

// deduceRequiresTypes attempts to infer types for 'requires' arguments based on 'extra' definitions
// in the mappings that use the nested types.
func (r *Resolver) deduceRequiresTypes(plan *ResolvedMappingPlan) {
	// 1. Build a graph of usages: which type pairs use which nested type pairs?
	uniquePairs := make(map[string]*ResolvedTypePair)
	usages := make(map[string][]pairUsage)

	// Helper to traverse recursively
	var traverse func(pair *ResolvedTypePair)

	traverse = func(pair *ResolvedTypePair) {
		if pair == nil {
			return
		}

		key := getPairKey(pair)
		if _, seen := uniquePairs[key]; seen {
			return
		}

		uniquePairs[key] = pair

		for _, nested := range pair.NestedPairs {
			if nested.ResolvedPair == nil {
				continue
			}

			childKey := getPairKey(nested.ResolvedPair)

			// Record usage
			usages[childKey] = append(usages[childKey], pairUsage{
				Parent:       pair,
				ReferencedBy: nested.ReferencedBy,
			})

			traverse(nested.ResolvedPair)
		}
	}

	// Start traversal from top-level pairs in the plan
	for i := range plan.TypePairs {
		traverse(&plan.TypePairs[i])
	}

	// 2. For each pair, check its requires
	for key, pair := range uniquePairs {
		for i := range pair.Requires {
			req := &pair.Requires[i]
			// Only deduce if type is explicitly interface{} or empty
			// (which defaults to interface{} in loader but let's check string)
			if req.Type != "interface{}" && req.Type != "" {
				continue
			}

			// Attempt to deduce
			candidates := []DeducedType{}
			pairUsages := usages[key]

			for _, usage := range pairUsages {
				// Find the field mapping in Parent that triggers this usage
				// We look for a mapping whose TargetPaths overlap with ReferencedBy
				var matchingMapping *ResolvedFieldMapping

				for idx := range usage.Parent.Mappings {
					m := &usage.Parent.Mappings[idx]
					if pathsIntersect(m.TargetPaths, usage.ReferencedBy) {
						matchingMapping = m
						break
					}
				}

				if matchingMapping == nil {
					continue
				}

				// Check Extra in matching mapping for the current requirement name
				for _, extra := range matchingMapping.Extra {
					if extra.Name == req.Name {
						var (
							fieldType  *analyze.TypeInfo
							originDesc string
						)

						if extra.Def.Source != "" {
							// Try to resolve from source
							srcPath, err := mapping.ParsePath(extra.Def.Source)
							if err == nil {
								fieldType = r.resolveFieldType(srcPath, usage.Parent.SourceType)
								originDesc = fmt.Sprintf("mapping %s->%s source field %s",
									usage.Parent.SourceType.ID, usage.Parent.TargetType.ID, extra.Def.Source)
							}
						} else if extra.Def.Target != "" {
							// Try to resolve from target
							tgtPath, err := mapping.ParsePath(extra.Def.Target)
							if err == nil {
								fieldType = r.resolveFieldType(tgtPath, usage.Parent.TargetType)
								originDesc = fmt.Sprintf("mapping %s->%s target field %s",
									usage.Parent.SourceType.ID, usage.Parent.TargetType.ID, extra.Def.Target)
							}
						}

						if fieldType == nil || fieldType.GoType == nil {
							continue
						}

						typeStr := types.TypeString(fieldType.GoType, nil)
						candidates = append(candidates, DeducedType{
							TypeStr: typeStr,
							Source:  originDesc,
						})
					}
				}
			}

			// 3. Resolve candidates
			if len(candidates) > 0 {
				first := candidates[0]
				conflict := false

				for _, c := range candidates[1:] {
					if c.TypeStr != first.TypeStr {
						conflict = true

						plan.Diagnostics.AddWarning("requires_type_conflict",
							fmt.Sprintf("Conflicting deduced types for required variable %q: "+
								"%s (from %s) vs %s (from %s). Keeping interface{}.",
								req.Name, first.TypeStr, first.Source, c.TypeStr, c.Source),
							key, "")
					}
				}

				if !conflict {
					req.Type = first.TypeStr
				}
			}
		}
	}
}

func getPairKey(p *ResolvedTypePair) string {
	if p == nil || p.SourceType == nil || p.TargetType == nil {
		return "unknown"
	}

	return fmt.Sprintf("%s->%s", p.SourceType.ID, p.TargetType.ID)
}

func pathsIntersect(a, b []mapping.FieldPath) bool {
	for _, pa := range a {
		strA := pa.String()
		for _, pb := range b {
			if strA == pb.String() {
				return true
			}
		}
	}

	return false
}
