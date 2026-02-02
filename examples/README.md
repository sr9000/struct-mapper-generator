# Examples

This directory contains **multi-stage scenario runners** demonstrating `caster-generator` features.
Each example is an interactive, repeatable workflow that shows the complete `suggest → patch → generate` cycle.

## Quick Start

```bash
# Run any example scenario (interactive mode)
cd examples/arrays
./run.sh

# Run in non-interactive mode (for CI)
CG_NO_PROMPT=1 ./run.sh

# Run with debug output
CG_DEBUG=1 ./run.sh
```

## Directory Structure

```
examples/
├── _scripts/
│   └── common.sh        # Shared helpers for all scenarios
│
├── arrays/              # Fixed-size array mapping with dive hints
│   ├── run.sh           # Multi-stage scenario runner
│   ├── source.go        # APIBox, APIPoint
│   ├── target.go        # DomainBox, DomainPoint
│   └── stages/          # Generated during run (YAML checkpoints)
│
├── basic/               # Simple renames + type conversions
│   ├── run.sh           # Multi-stage scenario runner
│   ├── source.go        # UserDTO, OrderDTO
│   ├── target.go        # User, Order
│   └── transforms.go    # Generated transform functions
│
├── nested/              # Full-feature: requires, transforms, ignore, auto
│   ├── run.sh           # Multi-stage scenario runner
│   ├── source.go        # APIOrder, APICustomer, APIItem, APIAddress
│   ├── target.go        # DomainOrder, DomainCustomer, DomainLineItem, DomainAddress
│   └── transforms.go    # Generated transform functions
│
├── nested-mixed/        # Pointer slices + renames
│   ├── run.sh           # Multi-stage scenario runner
│   ├── source.go        # APIOrder, APIItem (with pointers)
│   └── target.go        # DomainOrder, DomainLine
│
├── pointers/            # Pointer deref + deep field paths
│   ├── run.sh           # Multi-stage scenario runner
│   ├── source.go        # APIOrder, APILineItem (with *int)
│   ├── target.go        # DomainOrder, DomainLineItem
│   └── transforms.go    # Pointer deref transform
│
├── recursive/           # Self-referential types (linked list)
│   ├── run.sh           # Multi-stage scenario runner
│   ├── source.go        # Node (*Node)
│   └── target.go        # NodeDTO (*NodeDTO)
│
├── transforms/          # Type incompatibilities with custom transforms
│   ├── run.sh           # Multi-stage scenario runner
│   ├── source.go        # LegacyProduct, LegacyUser
│   ├── target.go        # ModernProduct, ModernUser
│   └── transforms.go    # Generated transform implementations
│
└── multi-mapping/       # Chained mappings across system boundaries
    ├── run.sh           # Multi-stage scenario runner
    ├── types.go         # External → Internal → Warehouse types
    └── transforms.go    # Generated transform functions
```

## Scenario Overview

### 1. Arrays — Dive Hints for Fixed-Size Arrays

**Goal:** Demonstrate `hint: dive` for array element conversion.

**Stages:**
1. Initial suggest (no hints)
2. Patch YAML: add `hint: dive` for `Corners` field
3. Suggest improve (validate)
4. Generate code
5. Compile check

**Key Concept:** When mapping `[4]APIPoint` → `[4]DomainPoint`, you need `hint: dive` to tell the generator to convert each element.

---

### 2. Basic — Renames and Type Conversions

**Goal:** Show iterative improvement for field renames and numeric type conversions.

**Stages:**
1. Suggest UserDTO → User
2. Suggest OrderDTO → Order
3. Merge & patch with explicit transforms
4. Suggest improve
5. Generate transform stubs
6. Generate casters
7. Compile check

**Key Concept:** When types don't match (e.g., `int64` → `uint`, `float64` → `int64`), you declare transform functions in YAML and implement them in Go.

---

### 3. Nested — Full-Feature Workflow

**Goal:** Demonstrate all YAML features: nesting, `requires`, `transforms`, `ignore`, `auto`.

**Stages:**
1. Initial suggest for root type
2. Patch with dive hints, ignores, requires
3. Suggest improve
4. Generate transforms
5. Generate with suggestions (catches missing transforms)
6. Final generate
7. Compile check

**Key Concepts:**
- `hint: dive` for slice/array fields with nested types
- `requires` to pass context from parent to nested casters
- `ignore` for fields you don't want to map
- `auto` for fields the tool should auto-match

---

### 4. Nested-Mixed — Pointer Slices

**Goal:** Handle `[]*SourceType` → `[]*TargetType` with renames.

**Stages:**
1. Initial suggest
2. Patch with dive hint
3. Suggest improve
4. Generate code
5. Compile check

**Key Concept:** Pointer slices work like regular slices with `hint: dive`.

---

### 5. Pointers — Pointer Deref and Deep Paths

**Goal:** Map pointer fields and deep source paths (`LineItem.Price`).

**Stages:**
1. Initial suggest
2. Patch with deep paths and deref transforms
3. Generate pointer deref transform
4. Suggest improve
5. Generate code
6. Compile check

**Key Concept:** For `*int` → `int`, you need a transform that handles nil safely.

---

### 6. Recursive — Self-Referential Types

**Goal:** Generate casters for recursive structures (linked lists, trees).

**Stages:**
1. Analyze types
2. Initial suggest
3. Verify mapping
4. Suggest improve
5. Generate code
6. Compile check

**Key Concept:** The generator automatically handles recursive calls.

---

### 7. Transforms — Custom Transform Functions

**Goal:** Canonical example for type incompatibilities with many transform patterns.

**Stages:**
1. Suggest with low confidence
2. Create explicit mapping with transforms
3. Suggest improve
4. Generate transform stubs **on the fly**
5. Generate casters
6. Compile check

**Transform Patterns Demonstrated:**
- Numeric: `float64` → `int64` (dollars to cents)
- Parsing: `string` → `time.Time`, `string` → `int` (weight)
- Boolean: `"Y"/"N"` → `bool`
- Splitting: `"a,b,c"` → `[]string`
- Multi-source: `FirstName` + `LastName` → `FullName`
- JSON extraction: `{"city":"NYC"}` → `City`, `PostalCode`

---

### 8. Multi-Mapping — Chained Conversions

**Goal:** Map types across system boundaries: External API → Internal → Warehouse.

**Stages:**
1. Suggest External → Internal
2. Suggest Internal → Warehouse
3. Create merged mapping
4. Generate transforms
5. Validate mapping
6. Generate casters
7. Compile check

**Key Concept:** A single YAML file can contain multiple mapping chains.

---

## Environment Variables

| Variable       | Description                                      |
|----------------|--------------------------------------------------|
| `CG_NO_PROMPT` | Set to `1` to skip interactive prompts (for CI) |
| `CG_DEBUG`     | Set to `1` to show executed commands            |
| `CG_VERBOSE`   | Set to `1` for more detailed output             |

## Workflow Pattern

All scenarios follow this pattern:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   suggest   │ ──▶ │ patch YAML  │ ──▶ │   suggest   │
│  (initial)  │     │  (manual)   │     │  (improve)  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   compile   │ ◀── │     gen     │ ◀── │  generate   │
│    check    │     │             │     │  transforms │
└─────────────┘     └─────────────┘     └─────────────┘
```

## Transform Generation "On The Fly"

When the generator encounters type incompatibilities, it can:

1. **Detect the need:** Move incompatible `121` mappings to `fields` section
2. **Generate placeholder names:** `FieldNameToTargetField` style
3. **Export YAML with signatures:** `transforms:` section with `signature:`

The scenario scripts then:
1. Parse the YAML for transform declarations
2. Generate Go stub files with proper signatures
3. User implements the logic (or uses generated implementations)
4. Re-run generation

## See Also

- Main [README.md](../README.md) for CLI reference
- [STRUCTURE.md](../STRUCTURE.md) for project structure

## Known Issues

Some generated code may have compilation issues:

1. **Duplicate transform functions**: When transforms are defined in both YAML and transforms.go, the generator may inline them causing redeclaration errors. Workaround: Remove duplicates from generated files or transforms.go.

2. **Requires parameter mismatch**: When using `requires` section, the generated caster may have different signature than expected. This is a generator bug being tracked.

3. **Multi-mapping dive hints**: Complex nested struct mappings with `hint: dive` may not generate code. The mapping YAML structure is demonstrated but generation support is incomplete.
