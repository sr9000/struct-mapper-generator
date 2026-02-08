package plan

import (
	"fmt"
	"sort"

	"caster-generator/internal/diagnostic"
	"caster-generator/internal/mapping"
)

// populateExtraTargetDependencies turns `extra.def.target` references into ordering dependencies.
//
// Semantics: if a mapping for target field B uses extra.def.target = "A" (or "A.Sub")
// then B depends on the assignment for A (or A.Sub).
func (r *Resolver) populateExtraTargetDependencies(pair *ResolvedTypePair, diags *diagnostic.Diagnostics) {
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
