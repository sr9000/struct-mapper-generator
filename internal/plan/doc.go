// Package plan provides the resolution pipeline that produces a final
// ResolvedMappingPlan consumed by code generation.
//
// Resolution pipeline:
//  1. Analyze packages → type graph
//  2. Load YAML (optional) → validate
//  3. For each requested type pair:
//     - Apply YAML-pinned rules first
//     - For remaining target fields, compute candidates via fuzzy matcher
//     - Auto-accept only on high confidence, otherwise mark "needs review"
//  4. Emit diagnostics (unmapped targets, ambiguity lists, unsafe conversions)
package plan
