package mapping

import (
	"fmt"
	"os"
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
	if err := yaml.Unmarshal(data, &mf); err != nil {
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
		if err := node.Decode(&str); err != nil {
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
		if err := node.Decode(&arr); err != nil {
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
func (s StringOrArray) MarshalYAML() (interface{}, error) {
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
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// ParsePath parses a field path string into a FieldPath.
// Supports: "Field", "Nested.Field", "Items[]", "Items[].ProductID".
func ParsePath(path string) (FieldPath, error) {
	if path == "" {
		return FieldPath{}, fmt.Errorf("empty path")
	}

	var segments []PathSegment

	parts := strings.Split(path, ".")

	for _, part := range parts {
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
				Source: StringOrArray{source},
				Target: StringOrArray{target},
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
