package plan

import (
	"testing"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

func TestPopulateExtraTargetDependencies_BuildsDepsAndDetectsMissing(t *testing.T) {
	pair := &ResolvedTypePair{
		SourceType: &analyze.TypeInfo{ID: analyze.TypeID{PkgPath: "p", Name: "S"}},
		TargetType: &analyze.TypeInfo{ID: analyze.TypeID{PkgPath: "p", Name: "T"}},
		Mappings: []ResolvedFieldMapping{
			{TargetPaths: []mapping.FieldPath{mustPath(t, "A")}},
			{TargetPaths: []mapping.FieldPath{mustPath(t, "B")}, Extra: []mapping.ExtraVal{{Name: "x", Def: mapping.ExtraDef{Target: "A"}}}},
			{TargetPaths: []mapping.FieldPath{mustPath(t, "C")}, Extra: []mapping.ExtraVal{{Name: "y", Def: mapping.ExtraDef{Target: "Missing"}}}},
		},
	}

	r := &Resolver{}
	diags := &Diagnostics{}
	r.populateExtraTargetDependencies(pair, diags)

	if len(pair.Mappings[1].DependsOnTargets) != 1 || pair.Mappings[1].DependsOnTargets[0].String() != "A" {
		t.Fatalf("expected B to depend on A, got %#v", pair.Mappings[1].DependsOnTargets)
	}

	if len(diags.Errors) == 0 {
		t.Fatalf("expected an error for missing dependency")
	}
}

func mustPath(t *testing.T, s string) mapping.FieldPath {
	t.Helper()

	p, err := mapping.ParsePath(s)
	if err != nil {
		t.Fatalf("parse path %q: %v", s, err)
	}

	return p
}
