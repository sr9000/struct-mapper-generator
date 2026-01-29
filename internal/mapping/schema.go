package mapping

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

// FieldMapping defines how target field(s) are populated from source field(s).
// Supports all cardinalities: 1:1, 1:many, many:1, many:many.
type FieldMapping struct {
	// Target is the target field path(s). Can be a single string or array.
	// Examples: "Name", ["FirstName", "DisplayName"], "Address.Street"
	Target StringOrArray `yaml:"target"`

	// Source is the source field path(s). Can be a single string or array.
	// Examples: "Name", ["FirstName", "LastName"], "Addr.Street"
	// If empty, the field is set from Default or via Transform.
	Source StringOrArray `yaml:"source,omitempty"`

	// Default is a literal value to assign if Source is empty.
	// Supports basic types: strings (quoted), numbers, booleans.
	Default *string `yaml:"default,omitempty"`

	// Transform is the name of a transform function to apply.
	// Required for many:1 mappings. For many:many, a unique transform
	// name is auto-generated if not specified.
	Transform string `yaml:"transform,omitempty"`

	// Ignore can be set to true to explicitly skip this target field.
	Ignore bool `yaml:"ignore,omitempty"`
}

// StringOrArray is a type that can be unmarshaled from either a string or an array of strings.
// This allows YAML fields to accept both "field" and ["field1", "field2"].
type StringOrArray []string

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
		return "unknown"
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
		return "unknown"
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

	for i, seg := range p.Segments {
		if i > 0 {
			result += "."
		}

		result += seg.Name
		if seg.IsSlice {
			result += "[]"
		}
	}

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
