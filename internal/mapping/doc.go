// Package mapping provides YAML schema definitions, parsing, validation,
// and the transform registry for explicit field mappings.
//
// YAML is a first-class feature that turns best-effort suggestions
// into deterministic regeneration.
//
// # Key capabilities
//
//   - Pin explicit field mappings (1:1, 1:many, many:1, many:many)
//   - Simplified "121" shorthand for 1:1 mappings
//   - Ignore target fields
//   - Set defaults
//   - Apply named transforms
//   - Support path expressions for nested shapes (e.g., "Items[].ProductID")
//   - Priority-based conflict resolution (121 > fields > ignore > auto)
//
// # Schema Overview
//
// The mapping file has the following structure:
//
//	version: "1"
//	mappings:
//	  - source: store.Order
//	    target: warehouse.Order
//	    # Simplified 1:1 mappings (highest priority)
//	    121:
//	      OrderID: ID
//	      CustomerName: Customer
//	    # Full field mappings with all options
//	    fields:
//	      - target: Status
//	        default: "pending"
//	      - target: [DisplayName, FullName]  # 1:many
//	        source: Name
//	      - target: Address                  # many:1 (requires transform)
//	        source: [Street, City, State]
//	        transform: ConcatAddress
//	    # Fields to ignore
//	    ignore:
//	      - InternalField
//	    # Auto-matched fields (populated during resolution, lowest priority)
//	    auto:
//	      - target: Amount
//	        source: Price
//	transforms:
//	  - name: ConcatAddress
//	    source_type: string
//	    target_type: string
//
// # Priority Order
//
// When resolving field mappings, conflicts are resolved using this priority:
//  1. "121" shorthand mappings (highest)
//  2. "fields" explicit mappings
//  3. "ignore" list
//  4. "auto" best-effort matches (lowest)
//
// # Cardinality Support
//
//   - 1:1 - Single source to single target (auto-resolvable for primitives)
//   - 1:N - Single source to multiple targets (no transform required)
//   - N:1 - Multiple sources to single target (transform required)
//   - N:M - Multiple sources to multiple targets (transform required)
//
// # Path Syntax
//
// Field paths support:
//   - Simple fields: "Name"
//   - Nested fields: "Address.Street"
//   - Slice elements: "Items[]"
//   - Nested slice fields: "Items[].ProductID"
//
// # Transform Registry
//
// Transforms are referenced by name in field mappings. The registry validates
// that referenced transforms exist and have compatible type signatures.
// For N:1 and N:M mappings, transforms are required. For unspecified transforms,
// unique names are auto-generated (e.g., "FirstNameLastNameToFullName").
package mapping
