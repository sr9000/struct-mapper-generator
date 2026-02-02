package plan

import (
	"testing"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

func TestResolveFieldMapping_YAMLPointerDerefStrategy(t *testing.T) {
	an := analyze.NewAnalyzer()

	graph, err := an.LoadPackages("caster-generator/examples/pointers")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}

	src := mapping.ResolveTypeID("caster-generator/examples/pointers.APIOrder", graph)
	if src == nil {
		t.Fatalf("source type not found")
	}

	tgt := mapping.ResolveTypeID("caster-generator/examples/pointers.DomainOrder", graph)
	if tgt == nil {
		t.Fatalf("target type not found")
	}

	mf := &mapping.MappingFile{Version: "1"}
	r := NewResolver(graph, mf, DefaultConfig())

	fm := &mapping.FieldMapping{
		Source: mapping.FieldRefArray{{Path: "LineItem.Price"}},
		Target: mapping.FieldRefArray{{Path: "LineItemPrice"}},
	}

	resolved, err := r.resolveFieldMapping(fm, src, tgt, MappingSourceYAMLFields)
	if err != nil {
		t.Fatalf("resolveFieldMapping: %v", err)
	}

	if resolved.Strategy != StrategyPointerDeref {
		t.Fatalf("expected StrategyPointerDeref, got %v (explanation=%q)", resolved.Strategy, resolved.Explanation)
	}
}
