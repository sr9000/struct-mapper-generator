package mapping

import (
	"errors"
	"fmt"
	"strings"
)

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
