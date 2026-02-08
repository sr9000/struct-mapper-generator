package mapping

import (
	"errors"
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"

	"caster-generator/internal/common"
)

// --- StringOrArray YAML methods ---

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
	if v, ok := common.First(s); ok {
		return v
	}

	return ""
}

// IsEmpty returns true if the array is empty.
func (s StringOrArray) IsEmpty() bool {
	return common.IsEmpty(s)
}

// IsSingle returns true if the array has exactly one element.
func (s StringOrArray) IsSingle() bool {
	return common.IsSingle(s)
}

// IsMultiple returns true if the array has more than one element.
func (s StringOrArray) IsMultiple() bool {
	return common.IsMultiple(s)
}

// Contains returns true if the array contains the given string.
func (s StringOrArray) Contains(str string) bool {
	return slices.Contains(s, str)
}

// --- FieldRefArray YAML methods ---

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

// --- ExtraVals YAML methods ---

// UnmarshalYAML implements yaml.Unmarshaler for ExtraVals.
func (e *ExtraVals) UnmarshalYAML(unmarshal func(any) error) error {
	// Try list of full ExtraVal objects (explicit syntax)
	var objects []ExtraVal
	if err := unmarshal(&objects); err == nil {
		*e = objects
		return nil
	}

	// Try list of strings first
	var list []string
	if err := unmarshal(&list); err == nil {
		result := make([]ExtraVal, len(list))
		for i, s := range list {
			result[i] = ExtraVal{Name: s, Def: ExtraDef{Source: s}}
		}

		*e = result

		return nil
	}

	// Try single string
	var single string
	if err := unmarshal(&single); err == nil {
		*e = []ExtraVal{{Name: single, Def: ExtraDef{Source: single}}}
		return nil
	}

	// Try map
	var m map[string]any
	if err := unmarshal(&m); err == nil {
		var result []ExtraVal

		for k, v := range m {
			// v can be string (implied source) or object
			switch val := v.(type) {
			case string:
				result = append(result, ExtraVal{Name: k, Def: ExtraDef{Source: val}})
			case map[string]any:
				def := ExtraDef{}
				if src, ok := val["source"].(string); ok {
					def.Source = src
				}

				if tgt, ok := val["target"].(string); ok {
					def.Target = tgt
				}

				result = append(result, ExtraVal{Name: k, Def: def})
			default:
				return fmt.Errorf("invalid extra definition for %s", k)
			}
		}
		// Sort for determinism? Use Slice sort later if needed. Map iteration is random.
		*e = result

		return nil
	}

	return errors.New("expected string, list of strings, or map for extra (MODIFIED)")
}

// --- StringArray YAML methods ---

// UnmarshalYAML implements yaml.Unmarshaler for StringArray.
func (s *StringArray) UnmarshalYAML(unmarshal func(any) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*s = []string{single}
		return nil
	}

	var multi []string
	if err := unmarshal(&multi); err == nil {
		*s = multi
		return nil
	}

	return errors.New("expected string or list of strings")
}

// --- FieldRefOrString YAML methods ---

// UnmarshalYAML implements the yaml.Unmarshaler interface for FieldRefOrString.
func (f *FieldRefOrString) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		f.Path = s
		f.Hint = HintNone

		return nil
	}

	var ref FieldRef
	if err := unmarshal(&ref); err == nil {
		*f = FieldRefOrString{ref}
		return nil
	}

	// Try map for hint structure {Path: Hint}
	var m map[string]string
	if err := unmarshal(&m); err == nil && len(m) == 1 {
		for k, v := range m {
			f.Path = k
			f.Hint = IntrospectionHint(v)
		}

		return nil
	}

	return errors.New("expected string or field reference")
}

// --- ArgDefArray YAML methods ---

// UnmarshalYAML implements yaml.Unmarshaler for ArgDefArray.
func (a *ArgDefArray) UnmarshalYAML(unmarshal func(any) error) error {
	var list []any
	if err := unmarshal(&list); err != nil {
		return err
	}

	result := make([]ArgDef, 0, len(list))
	for _, item := range list {
		switch v := item.(type) {
		case string:
			result = append(result, ArgDef{Name: v, Type: "interface{}"})
		case map[string]any:
			// Check if this is an explicit object definition (has "name" key)
			if nameVal, hasName := v["name"]; hasName {
				name, ok := nameVal.(string)
				if !ok {
					return errors.New("invalid argument name, expected string")
				}

				typeStr := "interface{}"

				if typeVal, hasType := v["type"]; hasType {
					if ts, ok := typeVal.(string); ok {
						typeStr = ts
					} else {
						return errors.New("invalid argument type, expected string")
					}
				}

				result = append(result, ArgDef{Name: name, Type: typeStr})

				continue
			}

			// Fallback to Key-Value definition: { "paramName": "paramType" }
			if len(v) != 1 {
				return errors.New("invalid argument definition, expected {name: type} or {name: ..., type: ...}")
			}

			for k, val := range v {
				typeStr, ok := val.(string)
				if !ok {
					return fmt.Errorf("invalid argument type for %s, expected string", k)
				}

				result = append(result, ArgDef{Name: k, Type: typeStr})
			}
		default:
			return errors.New("expected string or map for argument definition")
		}
	}

	*a = result

	return nil
}
