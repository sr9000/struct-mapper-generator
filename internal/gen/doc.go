// Package gen provides deterministic Go code generation for caster functions.
//
// Generation approach uses text/template + go/format for readable,
// allocation-light Go code.
//
// Codegen patterns:
//   - Direct assignment
//   - Numeric conversion with optional guards
//   - Pointer lift/deref with nil checks
//   - Slice mapping (make, loop, per-element conversion)
//   - Nested struct calls (composed generated casters)
//   - Transform function calls
package gen
