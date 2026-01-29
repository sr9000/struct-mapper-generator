package mapping

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFile loads and parses a YAML mapping file from the given path.
func LoadFile(path string) (*MappingFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file %s: %w", path, err)
	}

	return Parse(data)
}

// Parse parses YAML data into a MappingFile.
func Parse(data []byte) (*MappingFile, error) {
	var mf MappingFile

	err := yaml.Unmarshal(data, &mf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mapping YAML: %w", err)
	}

	// Apply defaults and normalize
	applyDefaults(&mf)

	return &mf, nil
}

// applyDefaults fills in default values for optional fields.
func applyDefaults(mf *MappingFile) {
	if mf.Version == "" {
		mf.Version = "1"
	}

	for i := range mf.Transforms {
		t := &mf.Transforms[i]
		if t.Func == "" {
			t.Func = t.Name
		}
	}
}

// UnmarshalYAML implements custom YAML unmarshaling for StringOrArray.
// Accepts either a single string or an array of strings.
func (s *StringOrArray) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Single string value
		var str string

		err := node.Decode(&str)
		if err != nil {
			return err
		}

		if str != "" {
			*s = StringOrArray{str}
		} else {
			*s = StringOrArray{}
		}

		return nil

	case yaml.SequenceNode:
		// Array of strings
		var arr []string

		err := node.Decode(&arr)
		if err != nil {
			return err
		}

		*s = arr

		return nil

	default:
		return fmt.Errorf("expected string or array, got %v", node.Kind)
	}
}

// MarshalYAML implements custom YAML marshaling for StringOrArray.
// Outputs a single string if length is 1, otherwise an array.
func (s StringOrArray) MarshalYAML() (any, error) {
	if len(s) == 1 {
		return s[0], nil
	}

	return []string(s), nil
}

// First returns the first element or empty string if empty.
func (s StringOrArray) First() string {
	if len(s) == 0 {
		return ""
	}

	return s[0]
}

// IsEmpty returns true if the array is empty.
func (s StringOrArray) IsEmpty() bool {
	return len(s) == 0
}

// IsSingle returns true if the array has exactly one element.
func (s StringOrArray) IsSingle() bool {
	return len(s) == 1
}

// IsMultiple returns true if the array has more than one element.
func (s StringOrArray) IsMultiple() bool {
	return len(s) > 1
}

// Contains returns true if the array contains the given string.
func (s StringOrArray) Contains(str string) bool {
	return slices.Contains(s, str)
}

// UnmarshalYAML implements custom YAML unmarshaling for FieldRefArray.
// Accepts:
//   - Single string: "Name"
//   - Single with hint: {Name: dive}
//   - Array of strings: ["Name", "FullName"]
//   - Array with hints: [{DisplayName: dive}, FullName]
func (f *FieldRefArray) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Single string value: "Name"
		var str string

		err := node.Decode(&str)
		if err != nil {
			return err
		}

		if str != "" {
			*f = FieldRefArray{{Path: str, Hint: HintNone}}
		} else {
			*f = FieldRefArray{}
		}

		return nil

	case yaml.MappingNode:
		// Single map: {Name: dive}
		ref, err := parseFieldRefFromMap(node)
		if err != nil {
			return err
		}

		*f = FieldRefArray{ref}

		return nil

	case yaml.SequenceNode:
		// Array of items
		var refs []FieldRef

		for _, item := range node.Content {
			switch item.Kind {
			case yaml.ScalarNode:
				// String item: "Name"
				var str string

				err := item.Decode(&str)
				if err != nil {
					return err
				}

				refs = append(refs, FieldRef{Path: str, Hint: HintNone})

			case yaml.MappingNode:
				// Map item: {Name: dive}
				ref, err := parseFieldRefFromMap(item)
				if err != nil {
					return err
				}

				refs = append(refs, ref)

			default:
				return fmt.Errorf("expected string or map in array, got %v", item.Kind)
			}
		}

		*f = refs

		return nil

	default:
		return fmt.Errorf("expected string, map, or array, got %v", node.Kind)
	}
}

// parseFieldRefFromMap parses a YAML mapping node like {Name: dive} into a FieldRef.
func parseFieldRefFromMap(node *yaml.Node) (FieldRef, error) {
	if node.Kind != yaml.MappingNode || len(node.Content) != 2 {
		return FieldRef{}, errors.New("expected single key-value map like {Name: dive}")
	}

	var (
		path string
		hint string
	)

	err := node.Content[0].Decode(&path)
	if err != nil {
		return FieldRef{}, fmt.Errorf("invalid field path: %w", err)
	}

	err = node.Content[1].Decode(&hint)
	if err != nil {
		return FieldRef{}, fmt.Errorf("invalid hint value: %w", err)
	}

	h := IntrospectionHint(hint)
	if !h.IsValid() {
		return FieldRef{}, fmt.Errorf("invalid hint %q (expected 'dive' or 'final')", hint)
	}

	return FieldRef{Path: path, Hint: h}, nil
}

// MarshalYAML implements custom YAML marshaling for FieldRefArray.
// Outputs:
//   - Single string if length is 1 and no hint
//   - Single map if length is 1 with hint
//   - Array otherwise
func (f FieldRefArray) MarshalYAML() (any, error) {
	if len(f) == 0 {
		return nil, nil
	}

	if len(f) == 1 {
		if f[0].Hint == HintNone {
			return f[0].Path, nil
		}
		// Return map for single item with hint
		return map[string]string{f[0].Path: string(f[0].Hint)}, nil
	}

	// Array of items
	result := make([]any, len(f))

	for i, ref := range f {
		if ref.Hint == HintNone {
			result[i] = ref.Path
		} else {
			result[i] = map[string]string{ref.Path: string(ref.Hint)}
		}
	}

	return result, nil
}

// ParsePath parses a field path string into a FieldPath.
// Supports: "Field", "Nested.Field", "Items[]", "Items[].ProductID".
func ParsePath(path string) (FieldPath, error) {
	if path == "" {
		return FieldPath{}, errors.New("empty path")
	}

	var segments []PathSegment

	parts := strings.SplitSeq(path, ".")

	for part := range parts {
		if part == "" {
			return FieldPath{}, fmt.Errorf("invalid path %q: empty segment", path)
		}

		isSlice := false
		name := part

		// Check for slice notation
		if strings.HasSuffix(part, "[]") {
			isSlice = true
			name = strings.TrimSuffix(part, "[]")

			if name == "" {
				return FieldPath{}, fmt.Errorf("invalid path %q: slice without field name", path)
			}
		}

		// Validate identifier (basic check)
		if !isValidIdent(name) {
			return FieldPath{}, fmt.Errorf("invalid path %q: invalid identifier %q", path, name)
		}

		segments = append(segments, PathSegment{
			Name:    name,
			IsSlice: isSlice,
		})
	}

	return FieldPath{Segments: segments}, nil
}

// ParsePaths parses multiple field paths from a StringOrArray.
func ParsePaths(paths StringOrArray) ([]FieldPath, error) {
	result := make([]FieldPath, 0, len(paths))

	for _, p := range paths {
		fp, err := ParsePath(p)
		if err != nil {
			return nil, err
		}

		result = append(result, fp)
	}

	return result, nil
}

// ParsePathsFromRefs parses multiple field paths from a FieldRefArray.
func ParsePathsFromRefs(refs FieldRefArray) ([]FieldPath, error) {
	result := make([]FieldPath, 0, len(refs))

	for _, ref := range refs {
		fp, err := ParsePath(ref.Path)
		if err != nil {
			return nil, err
		}

		result = append(result, fp)
	}

	return result, nil
}

// isValidIdent checks if a string is a valid Go identifier.
func isValidIdent(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		if i == 0 {
			// First character must be letter or underscore
			if !isLetter(r) && r != '_' {
				return false
			}
		} else {
			// Subsequent characters can be letter, digit, or underscore
			if !isLetter(r) && !isDigit(r) && r != '_' {
				return false
			}
		}
	}

	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// Marshal serializes a MappingFile to YAML.
func Marshal(mf *MappingFile) ([]byte, error) {
	return yaml.Marshal(mf)
}

// WriteFile writes a MappingFile to the given path.
func WriteFile(mf *MappingFile, path string) error {
	data, err := Marshal(mf)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write mapping file %s: %w", path, err)
	}

	return nil
}

// NormalizeTypeMapping expands 121 shorthand into Fields entries
// and ensures all mappings are in canonical form.
func NormalizeTypeMapping(tm *TypeMapping) {
	// Convert 121 shorthand to explicit field mappings
	// These are prepended to Fields so they have higher effective priority
	// (when resolving, 121 entries will be checked first)
	if len(tm.OneToOne) > 0 {
		expanded := make([]FieldMapping, 0, len(tm.OneToOne))

		for source, target := range tm.OneToOne {
			expanded = append(expanded, FieldMapping{
				Source: FieldRefArray{{Path: source, Hint: HintNone}},
				Target: FieldRefArray{{Path: target, Hint: HintNone}},
			})
		}

		// 121 has highest priority, so it goes first
		tm.Fields = append(expanded, tm.Fields...)
	}
}

// NormalizeMappingFile normalizes all type mappings in a file.
func NormalizeMappingFile(mf *MappingFile) {
	for i := range mf.TypeMappings {
		NormalizeTypeMapping(&mf.TypeMappings[i])
	}
}
