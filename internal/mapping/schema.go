package mapping

import (
	"errors"
	"fmt"
	"strings"

	"caster-generator/internal/common"
)

// MappingFile represents the root of a YAML mapping definition file.
// This is the authoritative, human-reviewed mapping configuration.
type MappingFile struct {
	// Version of the mapping schema (for future compatibility).
	Version string `yaml:"version,omitempty"`

	// TypeMappings is a list of type pair mappings.
	TypeMappings []TypeMapping `yaml:"mappings"`

	// Transforms defines custom transform functions available for use.
	Transforms []TransformDef `yaml:"transforms,omitempty"`
}

// TypeMapping defines how to map one source type to one target type.
type TypeMapping struct {
	// Source type identifier (e.g., "store.Order" or full path).
	Source string `yaml:"source"`

	// Target type identifier (e.g., "warehouse.Order" or full path).
	Target string `yaml:"target"`

	// Requires lists external variables required by this mapping function.
	// These become additional arguments to the generated function.
	Requires ArgDefArray `yaml:"requires,omitempty"`

	// OneToOne is a simplified mapping syntax where keys are source fields
	// and values are target fields. Supports 1:1 mappings only.
	// Priority: highest (applied first).
	// Example: { "OrderID": "ID", "CustomerName": "Customer" }
	OneToOne map[string]string `yaml:"121,omitempty"`

	// Fields defines explicit field mappings with full control.
	// Supports 1:1, 1:many, many:1, and many:many with transforms.
	// Priority: second highest (after 121).
	Fields []FieldMapping `yaml:"fields,omitempty"`

	// Ignore lists target fields that should not be mapped.
	// Priority: third (after fields).
	Ignore []string `yaml:"ignore,omitempty"`

	// Auto contains auto-matched fields from best-effort matching.
	// This is populated during resolution and has lowest priority.
	// Fields here are overridden by 121, fields, or ignore.
	Auto []FieldMapping `yaml:"auto,omitempty"`
}

// IntrospectionHint indicates how the engine should handle field introspection.
type IntrospectionHint string

const (
	// HintNone means no hint provided; engine decides based on cardinality and types.
	HintNone IntrospectionHint = ""
	// HintDive forces the engine to inspect inner structure of the field (recursively map fields).
	HintDive IntrospectionHint = "dive"
	// HintFinal treats the field as a single unit requiring custom transform (no introspection).
	HintFinal IntrospectionHint = "final"
)

// IsValid returns true if the hint is a recognized value.
func (h IntrospectionHint) IsValid() bool {
	return h == HintNone || h == HintDive || h == HintFinal
}

// FieldRef represents a field path with an optional introspection hint.
// YAML formats supported:
//   - Simple string: "Name"
//   - With hint: {Name: dive} or {Name: final}
type FieldRef struct {
	// Path is the field path (e.g., "Name", "Address.Street", "Items[]").
	Path string
	// Hint is the optional introspection hint for this field.
	Hint IntrospectionHint
}

// String returns the path string.
func (f FieldRef) String() string {
	return f.Path
}

// FieldRefArray is a collection of FieldRef that can be unmarshaled from various YAML formats:
//   - Single string: "Name"
//   - Single with hint: {Name: dive}
//   - Array of strings: ["Name", "FullName"]
//   - Array with hints: [{DisplayName: dive}, FullName]
type FieldRefArray []FieldRef

// Paths returns just the path strings (for backward compatibility).
func (f FieldRefArray) Paths() []string {
	result := make([]string, len(f))
	for i, ref := range f {
		result[i] = ref.Path
	}

	return result
}

// First returns the first element's path or empty string if empty.
func (f FieldRefArray) First() string {
	if len(f) == 0 {
		return ""
	}

	return f[0].Path
}

// FirstRef returns the first element or zero FieldRef if empty.
func (f FieldRefArray) FirstRef() FieldRef {
	if len(f) == 0 {
		return FieldRef{}
	}

	return f[0]
}

// IsEmpty returns true if the array is empty.
func (f FieldRefArray) IsEmpty() bool {
	return len(f) == 0
}

// IsSingle returns true if the array has exactly one element.
func (f FieldRefArray) IsSingle() bool {
	return len(f) == 1
}

// IsMultiple returns true if the array has more than one element.
func (f FieldRefArray) IsMultiple() bool {
	return len(f) > 1
}

// HasAnyHint returns true if any field has a non-empty hint.
func (f FieldRefArray) HasAnyHint() bool {
	for _, ref := range f {
		if ref.Hint != HintNone {
			return true
		}
	}

	return false
}

// HasConflictingHints returns true if there are both dive and final hints.
func (f FieldRefArray) HasConflictingHints() bool {
	hasDive := false
	hasFinal := false

	for _, ref := range f {
		if ref.Hint == HintDive {
			hasDive = true
		}

		if ref.Hint == HintFinal {
			hasFinal = true
		}
	}

	return hasDive && hasFinal
}

// GetEffectiveHint returns the effective hint for resolution based on cardinality rules.
// For 1:N mappings, uses the source (first type) hint.
// For N:1 mappings, uses the target (last type) hint.
// Returns HintNone if no hints or conflicting hints.
func (f FieldRefArray) GetEffectiveHint(targets FieldRefArray) IntrospectionHint {
	// Check for conflicting hints across both arrays
	allRefs := append(append([]FieldRef{}, f...), targets...)
	hasDive := false
	hasFinal := false

	for _, ref := range allRefs {
		if ref.Hint == HintDive {
			hasDive = true
		}

		if ref.Hint == HintFinal {
			hasFinal = true
		}
	}

	if hasDive && hasFinal {
		// Conflicting hints - treat as final (no introspection)
		return HintFinal
	}

	srcCount := len(f)
	tgtCount := len(targets)

	// 1:N - introspect source (first type)
	if srcCount <= 1 && tgtCount > 1 {
		if srcCount == 1 && f[0].Hint != HintNone {
			return f[0].Hint
		}
		// Check targets for hints
		for _, t := range targets {
			if t.Hint != HintNone {
				return t.Hint
			}
		}

		return HintNone
	}

	// N:1 - introspect target (last type)
	if srcCount > 1 && tgtCount <= 1 {
		if tgtCount == 1 && targets[0].Hint != HintNone {
			return targets[0].Hint
		}
		// Check sources for hints
		for _, s := range f {
			if s.Hint != HintNone {
				return s.Hint
			}
		}

		return HintNone
	}

	// 1:1 - any hint applies
	if srcCount <= 1 && tgtCount <= 1 {
		if srcCount == 1 && f[0].Hint != HintNone {
			return f[0].Hint
		}

		if tgtCount == 1 && targets[0].Hint != HintNone {
			return targets[0].Hint
		}

		return HintNone
	}

	// N:M - conflicting by default, needs explicit hint or final
	if hasDive {
		return HintDive
	}

	if hasFinal {
		return HintFinal
	}

	return HintFinal // Default to final for N:M
}

// FieldMapping defines how target field(s) are populated from source field(s).
// Supports all cardinalities: 1:1, 1:many, many:1, many:many.
//
// Field paths can include optional introspection hints:
//   - Simple: "Name" or ["Name", "FullName"]
//   - With hint: {Name: dive} or [{DisplayName: dive}, FullName]
//
// Hints control recursive resolution:
//   - "dive": force recursive introspection of inner struct fields
//   - "final": treat as single unit requiring custom transform (no introspection)
//
// Introspection rules by cardinality:
//   - 1:N - introspects the source (first) type, not cloning into all targets
//   - N:1 - introspects the target (last) type
//   - 1:1 - possibly introspects if both are structs (unless hint says final)
//   - N:M - treated as final unless explicit dive hint
type FieldMapping struct {
	// Source is the source field path(s) with optional hints.
	// Examples: "Name", ["FirstName", "LastName"], {Addr: dive}
	// If empty, the field is set from Default or via Transform.
	Source FieldRefArray `yaml:"source,omitempty"`

	// Target is the target field path(s) with optional hints.
	// Examples: "Name", ["FirstName", "DisplayName"], {Address: dive}
	Target FieldRefArray `yaml:"target"`

	// Default is a literal value to assign if Source is empty.
	// Supports basic types: strings (quoted), numbers, booleans.
	Default *string `yaml:"default,omitempty"`

	// Transform is the name of a transform function to apply.
	// Required for many:1 mappings. For many:many, a unique transform
	// name is auto-generated if not specified.
	Transform string `yaml:"transform,omitempty"`

	// Extra lists additional info field paths from the source type (or parent scope)
	// that should be passed to the mapping/transform/caster.
	Extra ExtraVals `yaml:"extra,omitempty"`
}

// ExtraDef represents an extra value definition.
type ExtraDef struct {
	Source string `yaml:"source,omitempty"`
	Target string `yaml:"target,omitempty"`
}

// ExtraVals represents the "extra" field which can be a list of strings or a map of definitions.
type ExtraVals []ExtraVal

// ExtraVal is a single extra value definition with a name.
type ExtraVal struct {
	Name string   `yaml:"name"`
	Def  ExtraDef `yaml:"def"`
}

// UnmarshalYAML implements yaml.Unmarshaler.
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

// StringOrArray is a type that can be unmarshaled from either a string or an array of strings.
// This allows YAML fields to accept both "field" and ["field1", "field2"].
type StringOrArray []string

// StringArray is a string slice that can be unmarshaled from a single string or a list.
type StringArray []string

// UnmarshalYAML implements yaml.Unmarshaler.
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

// Cardinality represents the mapping cardinality.
type Cardinality int

const (
	CardinalityOneToOne   Cardinality = iota // 1:1 - single source to single target
	CardinalityOneToMany                     // 1:N - single source to multiple targets
	CardinalityManyToOne                     // N:1 - multiple sources to single target
	CardinalityManyToMany                    // N:M - multiple sources to multiple targets
)

// String returns a human-readable representation of the cardinality.
func (c Cardinality) String() string {
	switch c {
	case CardinalityOneToOne:
		return "1:1"
	case CardinalityOneToMany:
		return "1:N"
	case CardinalityManyToOne:
		return "N:1"
	case CardinalityManyToMany:
		return "N:M"
	default:
		return common.UnknownStr
	}
}

// GetCardinality returns the cardinality of this field mapping.
func (fm *FieldMapping) GetCardinality() Cardinality {
	sourceCount := len(fm.Source)
	targetCount := len(fm.Target)

	if sourceCount <= 1 && targetCount <= 1 {
		return CardinalityOneToOne
	}

	if sourceCount <= 1 && targetCount > 1 {
		return CardinalityOneToMany
	}

	if sourceCount > 1 && targetCount <= 1 {
		return CardinalityManyToOne
	}

	return CardinalityManyToMany
}

// NeedsTransform returns true if this mapping requires a transform function.
// Many:1 always requires transform. Many:many requires transform.
// 1:1 with incompatible types may need transform (checked during validation).
func (fm *FieldMapping) NeedsTransform() bool {
	card := fm.GetCardinality()
	return card == CardinalityManyToOne || card == CardinalityManyToMany
}

// GetEffectiveHint returns the effective introspection hint for this mapping.
// Delegates to FieldRefArray.GetEffectiveHint with source/target arrays.
func (fm *FieldMapping) GetEffectiveHint() IntrospectionHint {
	return fm.Source.GetEffectiveHint(fm.Target)
}

// TransformDef defines metadata about a transform function.
// The actual implementation lives in user code; this validates usage.
type TransformDef struct {
	// Name is the transform identifier used in field mappings.
	Name string `yaml:"name"`

	// SourceType is the expected input type (e.g., "string", "int", "store.Price").
	// For many:1, this describes the combined input pattern.
	SourceType string `yaml:"source_type"`

	// TargetType is the expected output type (e.g., "string", "float64", "warehouse.Amount").
	TargetType string `yaml:"target_type"`

	// Package is the import path where the transform function is defined.
	// If empty, assumes the transform is in the generated casters package.
	Package string `yaml:"package,omitempty"`

	// Func is the actual function name. Defaults to Name if not specified.
	Func string `yaml:"func,omitempty"`

	// Description is an optional human-readable description.
	Description string `yaml:"description,omitempty"`

	// AutoGenerated indicates this transform was auto-generated during resolution.
	AutoGenerated bool `yaml:"auto_generated,omitempty"`
}

// MappingPriority represents the priority level of a mapping rule.
type MappingPriority int

const (
	PriorityAuto     MappingPriority = iota // Lowest: auto-matched by best-effort
	PriorityIgnore                          // Third: explicitly ignored
	PriorityFields                          // Second: explicit field mappings
	PriorityOneToOne                        // Highest: 121 shorthand mappings
)

// String returns a human-readable representation of the priority.
func (p MappingPriority) String() string {
	switch p {
	case PriorityOneToOne:
		return "121"
	case PriorityFields:
		return "fields"
	case PriorityIgnore:
		return "ignore"
	case PriorityAuto:
		return "auto"
	default:
		return common.UnknownStr
	}
}

// PathSegment represents a parsed segment of a field path.
type PathSegment struct {
	// Name is the field name.
	Name string

	// IsSlice indicates this segment accesses slice elements (e.g., "Items[]").
	IsSlice bool
}

// FieldPath represents a parsed field path like "Items[].ProductID".
type FieldPath struct {
	Segments []PathSegment
}

// String returns the path as a string.
func (p FieldPath) String() string {
	var result string

	var resultSb396 strings.Builder

	for i, seg := range p.Segments {
		if i > 0 {
			resultSb396.WriteString(".")
		}

		resultSb396.WriteString(seg.Name)

		if seg.IsSlice {
			resultSb396.WriteString("[]")
		}
	}

	result += resultSb396.String()

	return result
}

// IsSimple returns true if this is a simple single-field path (no nesting, no slices).
func (p FieldPath) IsSimple() bool {
	return len(p.Segments) == 1 && !p.Segments[0].IsSlice
}

// Root returns the first segment's field name.
func (p FieldPath) Root() string {
	if len(p.Segments) == 0 {
		return ""
	}

	return p.Segments[0].Name
}

// IsEmpty returns true if the path has no segments.
func (p FieldPath) IsEmpty() bool {
	return len(p.Segments) == 0
}

// Equals returns true if two paths are equal.
func (p FieldPath) Equals(other FieldPath) bool {
	if len(p.Segments) != len(other.Segments) {
		return false
	}

	for i, seg := range p.Segments {
		if seg.Name != other.Segments[i].Name || seg.IsSlice != other.Segments[i].IsSlice {
			return false
		}
	}

	return true
}

// FieldRefOrString handles unmarshalling either a simple string or a FieldRef object.
type FieldRefOrString struct {
	FieldRef
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
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

// ArgDef represents an argument definition (name and type).
// Can be simpler string "name" (type is inferred or interface{}) or complex {name: type}.
type ArgDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// ArgDefArray unmarshals a list of arguments.
type ArgDefArray []ArgDef

// UnmarshalYAML implements yaml.Unmarshaler.
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
