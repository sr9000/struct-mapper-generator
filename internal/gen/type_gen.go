package gen

import (
	"errors"
	"fmt"
	"strings"

	"caster-generator/internal/analyze"
	"caster-generator/internal/plan"
)

// GenerateStruct generates the Go struct definition for a generated target type.
func (g *Generator) GenerateStruct(pair *plan.ResolvedTypePair, imports map[string]importSpec) (string, error) {
	if !pair.IsGeneratedTarget {
		return "", nil
	}

	t := pair.TargetType
	if t == nil {
		return "", errors.New("target type is nil")
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("type %s struct {\n", t.ID.Name))

	for _, f := range t.Fields {
		typeStr := g.typeStringForStruct(f.Type, imports)
		jsonTag := lowerFirst(f.Name)
		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", f.Name, typeStr, jsonTag))
	}

	sb.WriteString("}\n")

	return sb.String(), nil
}

// typeStringForStruct resolves type string using imports.
func (g *Generator) typeStringForStruct(t *analyze.TypeInfo, imports map[string]importSpec) string {
	if t == nil {
		return "interface{}"
	}

	// Use typeRefString helper if possible, but we need typeRef first?
	// Generator has typeRefString which takes *TypeInfo.
	// But typeRefString is private in generator.go.
	// Since we are in same package `gen`, we can call it!
	return g.typeRefString(t, imports)
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}

	return strings.ToLower(s[:1]) + s[1:]
}
