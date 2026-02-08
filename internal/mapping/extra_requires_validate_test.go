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

	// Test 1: Extra without definition - should require declaration in requires
	t.Run("extra without def must be declared", func(t *testing.T) {
		mf := &mapping.MappingFile{TypeMappings: []mapping.TypeMapping{{
			Source: "src.Src",
			Target: "dst.Dst",
			Fields: []mapping.FieldMapping{{
				Source: mapping.FieldRefArray{{Path: "A"}},
				Target: mapping.FieldRefArray{{Path: "B"}},
				// Extra without Source/Target def - must be declared in requires
				Extra: mapping.ExtraVals{{Name: "ctx"}},
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
	})

	// Test 2: Extra with source definition - does NOT need to be declared in requires
	t.Run("extra with source def is valid without requires", func(t *testing.T) {
		mf := &mapping.MappingFile{TypeMappings: []mapping.TypeMapping{{
			Source: "src.Src",
			Target: "dst.Dst",
			Fields: []mapping.FieldMapping{{
				Source: mapping.FieldRefArray{{Path: "A"}},
				Target: mapping.FieldRefArray{{Path: "B"}},
				// Extra with Source def - creates value from source field, no requires needed
				Extra: mapping.ExtraVals{{Name: "ctx", Def: mapping.ExtraDef{Source: "A"}}},
			}},
		}}}

		res := mapping.Validate(mf, graph)
		require.True(t, res.IsValid(), "validation errors: %#v", res.Errors)
	})

	// Test 3: Extra with target definition - does NOT need to be declared in requires
	t.Run("extra with target def is valid without requires", func(t *testing.T) {
		mf := &mapping.MappingFile{TypeMappings: []mapping.TypeMapping{{
			Source: "src.Src",
			Target: "dst.Dst",
			Fields: []mapping.FieldMapping{{
				Source: mapping.FieldRefArray{{Path: "A"}},
				Target: mapping.FieldRefArray{{Path: "B"}},
				// Extra with Target def - creates value from target field, no requires needed
				Extra: mapping.ExtraVals{{Name: "ctx", Def: mapping.ExtraDef{Target: "B"}}},
			}},
		}}}

		res := mapping.Validate(mf, graph)
		require.True(t, res.IsValid(), "validation errors: %#v", res.Errors)
	})
}
