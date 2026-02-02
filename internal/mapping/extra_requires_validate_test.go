package mapping_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"caster-generator/internal/analyze"
	"caster-generator/internal/mapping"
)

func TestValidate_ExtraRequiresMustBeDeclared(t *testing.T) {
	// Minimal graph: just enough to validate paths.
	srcID := analyze.TypeID{PkgPath: "src", Name: "Src"}
	dstID := analyze.TypeID{PkgPath: "dst", Name: "Dst"}

	graph := &analyze.TypeGraph{Types: map[analyze.TypeID]*analyze.TypeInfo{}}

	basicString := &analyze.TypeInfo{
		ID:   analyze.TypeID{Name: "string"},
		Kind: analyze.TypeKindBasic,
	}

	srcT := &analyze.TypeInfo{ID: srcID, Kind: analyze.TypeKindStruct}
	srcT.Fields = []analyze.FieldInfo{{
		Name:     "A",
		Exported: true,
		Type:     basicString,
	}}

	dstT := &analyze.TypeInfo{ID: dstID, Kind: analyze.TypeKindStruct}
	dstT.Fields = []analyze.FieldInfo{{
		Name:     "B",
		Exported: true,
		Type:     basicString,
	}}

	graph.Types[srcID] = srcT
	graph.Types[dstID] = dstT

	mf := &mapping.MappingFile{TypeMappings: []mapping.TypeMapping{{
		Source: "src.Src",
		Target: "dst.Dst",
		Fields: []mapping.FieldMapping{{
			Source: mapping.FieldRefArray{{Path: "A"}},
			Target: mapping.FieldRefArray{{Path: "B"}},
			Extra:  mapping.ExtraVals{{Name: "ctx", Def: mapping.ExtraDef{Source: "A"}}},
		}},
	}}}

	res := mapping.Validate(mf, graph)
	require.False(t, res.IsValid())

	found := false
	for _, e := range res.Errors {
		if e.TypePair == "src.Src->dst.Dst" &&
			e.Message == "extra \"ctx\" references an undeclared requires arg; add it under requires: or rename" {
			found = true
			break
		}
	}
	require.True(t, found, "expected undeclared requires error, got: %#v", res.Errors)
}
