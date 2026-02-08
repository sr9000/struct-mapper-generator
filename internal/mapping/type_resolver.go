package mapping

import (
	"strings"

	"caster-generator/internal/analyze"
)

// ResolveTypeID resolves a type ID string like:
// - "store.Order" (short)
// - "caster-generator/store.Order" (full)
// - "Order" (name only).
func ResolveTypeID(typeIDStr string, graph *analyze.TypeGraph) *analyze.TypeInfo {
	if graph == nil {
		return nil
	}

	// Name-only: best-effort match by type name.
	if !strings.Contains(typeIDStr, ".") {
		name := typeIDStr
		if name == "" {
			return nil
		}

		for id, t := range graph.Types {
			if id.Name == name {
				return t
			}
		}

		return nil
	}

	lastDot := strings.LastIndex(typeIDStr, ".")
	if lastDot < 0 {
		return nil
	}

	pkgStr := typeIDStr[:lastDot]

	name := typeIDStr[lastDot+1:]
	if pkgStr == "" || name == "" {
		return nil
	}

	// 1) exact match (for fully qualified import path)
	if t := graph.GetType(analyze.TypeID{PkgPath: pkgStr, Name: name}); t != nil {
		return t
	}

	// 2) suffix match (for short forms like "store.Order" vs "caster-generator/store.Order")
	for id, t := range graph.Types {
		if id.Name != name {
			continue
		}

		if id.PkgPath == pkgStr || strings.HasSuffix(id.PkgPath, "/"+pkgStr) {
			return t
		}
	}

	return nil
}
