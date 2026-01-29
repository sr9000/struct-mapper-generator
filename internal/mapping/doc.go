// Package mapping provides YAML schema definitions, parsing, validation,
// and the transform registry for explicit field mappings.
//
// YAML is a first-class feature that turns best-effort suggestions
// into deterministic regeneration.
//
// Key capabilities:
//   - Pin explicit field mappings
//   - Ignore target fields
//   - Set defaults
//   - Apply named transforms
//   - Support path expressions for nested shapes
package mapping
