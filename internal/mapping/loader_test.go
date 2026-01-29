package mapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	yaml := `
version: "1"
mappings:
  - source: store.Order
    target: warehouse.Order
    121:
      OrderID: ID
      CustomerName: Customer
    fields:
      - target: Status
        default: "pending"
      - target: Amount
        source: Price
        transform: PriceToAmount
      - target: [FirstName, DisplayName]
        source: Name
      - target: FullName
        source: [FirstName, LastName]
        transform: ConcatNames
    ignore:
      - InternalField
transforms:
  - name: PriceToAmount
    source_type: int
    target_type: float64
    description: Converts price in cents to amount in dollars
  - name: ConcatNames
    source_type: string
    target_type: string
`

	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)
	require.NotNil(t, mf)

	assert.Equal(t, "1", mf.Version)
	require.Len(t, mf.TypeMappings, 1)

	tm := mf.TypeMappings[0]
	assert.Equal(t, "store.Order", tm.Source)
	assert.Equal(t, "warehouse.Order", tm.Target)

	// Check 121 shorthand
	assert.Len(t, tm.OneToOne, 2)
	assert.Equal(t, "ID", tm.OneToOne["OrderID"])
	assert.Equal(t, "Customer", tm.OneToOne["CustomerName"])

	// Check field mappings
	assert.Len(t, tm.Fields, 4)
	assert.Len(t, tm.Ignore, 1)
	assert.Equal(t, "InternalField", tm.Ignore[0])

	// Field with default
	assert.Equal(t, "Status", tm.Fields[0].Target.First())
	require.NotNil(t, tm.Fields[0].Default)
	assert.Equal(t, "pending", *tm.Fields[0].Default)

	// Field with transform
	assert.Equal(t, "Amount", tm.Fields[1].Target.First())
	assert.Equal(t, "Price", tm.Fields[1].Source.First())
	assert.Equal(t, "PriceToAmount", tm.Fields[1].Transform)

	// 1:many mapping
	assert.Len(t, tm.Fields[2].Target, 2)
	assert.Equal(t, "FirstName", tm.Fields[2].Target[0].Path)
	assert.Equal(t, "DisplayName", tm.Fields[2].Target[1].Path)
	assert.Equal(t, "Name", tm.Fields[2].Source.First())
	assert.Equal(t, CardinalityOneToMany, tm.Fields[2].GetCardinality())

	// many:1 mapping
	assert.Equal(t, "FullName", tm.Fields[3].Target.First())
	assert.Len(t, tm.Fields[3].Source, 2)
	assert.Equal(t, "FirstName", tm.Fields[3].Source[0].Path)
	assert.Equal(t, "LastName", tm.Fields[3].Source[1].Path)
	assert.Equal(t, CardinalityManyToOne, tm.Fields[3].GetCardinality())

	// Check transforms
	require.Len(t, mf.Transforms, 2)
	tr := mf.Transforms[0]
	assert.Equal(t, "PriceToAmount", tr.Name)
	assert.Equal(t, "int", tr.SourceType)
	assert.Equal(t, "float64", tr.TargetType)
	assert.Equal(t, "PriceToAmount", tr.Func) // Defaults to Name
}

func TestParseMinimal(t *testing.T) {
	yaml := `
mappings:
  - source: A
    target: B
`

	mf, err := Parse([]byte(yaml))
	require.NoError(t, err)

	assert.Equal(t, "1", mf.Version) // Default version
	require.Len(t, mf.TypeMappings, 1)
	assert.Equal(t, "A", mf.TypeMappings[0].Source)
	assert.Equal(t, "B", mf.TypeMappings[0].Target)
}

func TestParseStringOrArray(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected FieldRefArray
	}{
		{
			name: "single string",
			yaml: `
mappings:
  - source: A
    target: B
    fields:
      - target: Name
        source: Name
`,
			expected: FieldRefArray{{Path: "Name"}},
		},
		{
			name: "array",
			yaml: `
mappings:
  - source: A
    target: B
    fields:
      - target: [First, Second]
        source: Value
`,
			expected: FieldRefArray{{Path: "First"}, {Path: "Second"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mf, err := Parse([]byte(tt.yaml))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, mf.TypeMappings[0].Fields[0].Target)
		})
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		input    string
		expected FieldPath
		wantErr  bool
	}{
		{
			input: "Name",
			expected: FieldPath{
				Segments: []PathSegment{{Name: "Name", IsSlice: false}},
			},
		},
		{
			input: "Address.Street",
			expected: FieldPath{
				Segments: []PathSegment{
					{Name: "Address", IsSlice: false},
					{Name: "Street", IsSlice: false},
				},
			},
		},
		{
			input: "Items[]",
			expected: FieldPath{
				Segments: []PathSegment{{Name: "Items", IsSlice: true}},
			},
		},
		{
			input: "Items[].ProductID",
			expected: FieldPath{
				Segments: []PathSegment{
					{Name: "Items", IsSlice: true},
					{Name: "ProductID", IsSlice: false},
				},
			},
		},
		{
			input: "Orders[].Items[].Name",
			expected: FieldPath{
				Segments: []PathSegment{
					{Name: "Orders", IsSlice: true},
					{Name: "Items", IsSlice: true},
					{Name: "Name", IsSlice: false},
				},
			},
		},
		{
			input:   "",
			wantErr: true,
		},
		{
			input:   ".",
			wantErr: true,
		},
		{
			input:   "Field.",
			wantErr: true,
		},
		{
			input:   ".Field",
			wantErr: true,
		},
		{
			input:   "[]",
			wantErr: true,
		},
		{
			input:   "123Invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParsePath(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFieldPathString(t *testing.T) {
	tests := []struct {
		path     FieldPath
		expected string
	}{
		{
			path: FieldPath{
				Segments: []PathSegment{{Name: "Name", IsSlice: false}},
			},
			expected: "Name",
		},
		{
			path: FieldPath{
				Segments: []PathSegment{
					{Name: "Items", IsSlice: true},
					{Name: "ProductID", IsSlice: false},
				},
			},
			expected: "Items[].ProductID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.path.String())
		})
	}
}

func TestFieldPathIsSimple(t *testing.T) {
	simple := FieldPath{Segments: []PathSegment{{Name: "Name", IsSlice: false}}}
	assert.True(t, simple.IsSimple())

	nested := FieldPath{Segments: []PathSegment{
		{Name: "Address", IsSlice: false},
		{Name: "Street", IsSlice: false},
	}}
	assert.False(t, nested.IsSimple())

	slice := FieldPath{Segments: []PathSegment{{Name: "Items", IsSlice: true}}}
	assert.False(t, slice.IsSimple())
}

func TestMarshal(t *testing.T) {
	defaultVal := "pending"
	mf := &MappingFile{
		Version: "1",
		TypeMappings: []TypeMapping{
			{
				Source: "store.Order",
				Target: "warehouse.Order",
				Fields: []FieldMapping{
					{Target: FieldRefArray{{Path: "ID"}}, Source: FieldRefArray{{Path: "OrderID"}}},
					{Target: FieldRefArray{{Path: "Status"}}, Default: &defaultVal},
				},
			},
		},
	}

	data, err := Marshal(mf)
	require.NoError(t, err)
	assert.Contains(t, string(data), "store.Order")
	assert.Contains(t, string(data), "warehouse.Order")

	// Verify round-trip
	parsed, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, mf.Version, parsed.Version)
	assert.Equal(t, len(mf.TypeMappings), len(parsed.TypeMappings))
}

func TestMarshalStringOrArray(t *testing.T) {
	// Single value should marshal as string
	single := StringOrArray{"Name"}
	data, err := single.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, "Name", data)

	// Multiple values should marshal as array
	multi := StringOrArray{"First", "Second"}
	data, err = multi.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, []string{"First", "Second"}, data)
}

func TestNormalizeTypeMapping(t *testing.T) {
	tm := TypeMapping{
		Source: "A",
		Target: "B",
		OneToOne: map[string]string{
			"Src1": "Tgt1",
			"Src2": "Tgt2",
		},
		Fields: []FieldMapping{
			{Target: FieldRefArray{{Path: "Existing"}}, Source: FieldRefArray{{Path: "ExistingSrc"}}},
		},
	}

	NormalizeTypeMapping(&tm)

	// 121 should be expanded and prepended to Fields
	assert.Len(t, tm.Fields, 3) // 2 from 121 + 1 existing
	// Original field should still be there
	found := false
	for _, f := range tm.Fields {
		if f.Target.First() == "Existing" {
			found = true
			break
		}
	}
	assert.True(t, found, "existing field should be preserved")
}

func TestCardinality(t *testing.T) {
	tests := []struct {
		name     string
		source   FieldRefArray
		target   FieldRefArray
		expected Cardinality
	}{
		{"1:1", FieldRefArray{{Path: "A"}}, FieldRefArray{{Path: "B"}}, CardinalityOneToOne},
		{"1:N", FieldRefArray{{Path: "A"}}, FieldRefArray{{Path: "B"}, {Path: "C"}}, CardinalityOneToMany},
		{"N:1", FieldRefArray{{Path: "A"}, {Path: "B"}}, FieldRefArray{{Path: "C"}}, CardinalityManyToOne},
		{"N:M", FieldRefArray{{Path: "A"}, {Path: "B"}}, FieldRefArray{{Path: "C"}, {Path: "D"}}, CardinalityManyToMany},
		{"empty source 1:1", FieldRefArray{}, FieldRefArray{{Path: "B"}}, CardinalityOneToOne},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := FieldMapping{Source: tt.source, Target: tt.target}
			assert.Equal(t, tt.expected, fm.GetCardinality())
		})
	}
}

func TestNeedsTransform(t *testing.T) {
	tests := []struct {
		name     string
		source   FieldRefArray
		target   FieldRefArray
		expected bool
	}{
		{"1:1 no transform", FieldRefArray{{Path: "A"}}, FieldRefArray{{Path: "B"}}, false},
		{"1:N no transform", FieldRefArray{{Path: "A"}}, FieldRefArray{{Path: "B"}, {Path: "C"}}, false},
		{"N:1 needs transform", FieldRefArray{{Path: "A"}, {Path: "B"}}, FieldRefArray{{Path: "C"}}, true},
		{"N:M needs transform", FieldRefArray{{Path: "A"}, {Path: "B"}}, FieldRefArray{{Path: "C"}, {Path: "D"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := FieldMapping{Source: tt.source, Target: tt.target}
			assert.Equal(t, tt.expected, fm.NeedsTransform())
		})
	}
}

func TestStringOrArrayMethods(t *testing.T) {
	empty := StringOrArray{}
	assert.True(t, empty.IsEmpty())
	assert.False(t, empty.IsSingle())
	assert.False(t, empty.IsMultiple())
	assert.Equal(t, "", empty.First())

	single := StringOrArray{"one"}
	assert.False(t, single.IsEmpty())
	assert.True(t, single.IsSingle())
	assert.False(t, single.IsMultiple())
	assert.Equal(t, "one", single.First())
	assert.True(t, single.Contains("one"))
	assert.False(t, single.Contains("two"))

	multi := StringOrArray{"one", "two"}
	assert.False(t, multi.IsEmpty())
	assert.False(t, multi.IsSingle())
	assert.True(t, multi.IsMultiple())
	assert.Equal(t, "one", multi.First())
	assert.True(t, multi.Contains("one"))
	assert.True(t, multi.Contains("two"))
}

func TestFieldPathEquals(t *testing.T) {
	path1 := FieldPath{Segments: []PathSegment{{Name: "A", IsSlice: false}}}
	path2 := FieldPath{Segments: []PathSegment{{Name: "A", IsSlice: false}}}
	path3 := FieldPath{Segments: []PathSegment{{Name: "B", IsSlice: false}}}
	path4 := FieldPath{Segments: []PathSegment{{Name: "A", IsSlice: true}}}

	assert.True(t, path1.Equals(path2))
	assert.False(t, path1.Equals(path3))
	assert.False(t, path1.Equals(path4))
}

func TestFieldRefArrayWithHints(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected FieldRefArray
	}{
		{
			name: "single string",
			yaml: `
mappings:
  - source: A
    target: B
    fields:
      - target: Name
        source: Name
`,
			expected: FieldRefArray{{Path: "Name", Hint: HintNone}},
		},
		{
			name: "single with dive hint",
			yaml: `
mappings:
  - source: A
    target: B
    fields:
      - target: {Address: dive}
        source: Addr
`,
			expected: FieldRefArray{{Path: "Address", Hint: HintDive}},
		},
		{
			name: "single with final hint",
			yaml: `
mappings:
  - source: A
    target: B
    fields:
      - target: {Metadata: final}
        source: Meta
`,
			expected: FieldRefArray{{Path: "Metadata", Hint: HintFinal}},
		},
		{
			name: "array with mixed hints",
			yaml: `
mappings:
  - source: A
    target: B
    fields:
      - target: [{DisplayName: dive}, FullName]
        source: Name
`,
			expected: FieldRefArray{
				{Path: "DisplayName", Hint: HintDive},
				{Path: "FullName", Hint: HintNone},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mf, err := Parse([]byte(tt.yaml))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, mf.TypeMappings[0].Fields[0].Target)
		})
	}
}

func TestFieldRefArrayMarshal(t *testing.T) {
	// Single without hint should marshal as string
	single := FieldRefArray{{Path: "Name"}}
	data, err := single.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, "Name", data)

	// Single with hint should marshal as map
	singleHint := FieldRefArray{{Path: "Address", Hint: HintDive}}
	data, err = singleHint.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Address": "dive"}, data)

	// Array should marshal as array
	multi := FieldRefArray{
		{Path: "DisplayName", Hint: HintDive},
		{Path: "FullName"},
	}
	data, err = multi.MarshalYAML()
	require.NoError(t, err)
	expected := []interface{}{
		map[string]string{"DisplayName": "dive"},
		"FullName",
	}
	assert.Equal(t, expected, data)
}

func TestFieldRefArrayMethods(t *testing.T) {
	empty := FieldRefArray{}
	assert.True(t, empty.IsEmpty())
	assert.False(t, empty.IsSingle())
	assert.False(t, empty.IsMultiple())
	assert.Equal(t, "", empty.First())
	assert.False(t, empty.HasAnyHint())

	single := FieldRefArray{{Path: "Name"}}
	assert.False(t, single.IsEmpty())
	assert.True(t, single.IsSingle())
	assert.False(t, single.IsMultiple())
	assert.Equal(t, "Name", single.First())
	assert.False(t, single.HasAnyHint())

	singleHint := FieldRefArray{{Path: "Name", Hint: HintDive}}
	assert.True(t, singleHint.HasAnyHint())

	multi := FieldRefArray{{Path: "A"}, {Path: "B", Hint: HintFinal}}
	assert.True(t, multi.IsMultiple())
	assert.True(t, multi.HasAnyHint())
	assert.Equal(t, []string{"A", "B"}, multi.Paths())
}

func TestGetEffectiveHint(t *testing.T) {
	tests := []struct {
		name     string
		source   FieldRefArray
		target   FieldRefArray
		expected IntrospectionHint
	}{
		{
			name:     "no hints",
			source:   FieldRefArray{{Path: "A"}},
			target:   FieldRefArray{{Path: "B"}},
			expected: HintNone,
		},
		{
			name:     "1:1 source hint",
			source:   FieldRefArray{{Path: "A", Hint: HintDive}},
			target:   FieldRefArray{{Path: "B"}},
			expected: HintDive,
		},
		{
			name:     "1:1 target hint",
			source:   FieldRefArray{{Path: "A"}},
			target:   FieldRefArray{{Path: "B", Hint: HintFinal}},
			expected: HintFinal,
		},
		{
			name:     "1:N source hint applies",
			source:   FieldRefArray{{Path: "A", Hint: HintDive}},
			target:   FieldRefArray{{Path: "B"}, {Path: "C"}},
			expected: HintDive,
		},
		{
			name:     "N:1 target hint applies",
			source:   FieldRefArray{{Path: "A"}, {Path: "B"}},
			target:   FieldRefArray{{Path: "C", Hint: HintDive}},
			expected: HintDive,
		},
		{
			name:   "N:M defaults to final",
			source: FieldRefArray{{Path: "A"}, {Path: "B"}},
			target: FieldRefArray{{Path: "C"}, {Path: "D"}},
			// N:M without hints defaults to final
			expected: HintFinal,
		},
		{
			name:     "conflicting hints resolve to final",
			source:   FieldRefArray{{Path: "A", Hint: HintDive}},
			target:   FieldRefArray{{Path: "B", Hint: HintFinal}},
			expected: HintFinal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.source.GetEffectiveHint(tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}
