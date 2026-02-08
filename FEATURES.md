# All Features

`caster-generator` is a semi-automated Go struct mapping codegen tool.

## Commands and Args

### `analyze` — Inspect packages

Print discovered structs and fields from packages (debug/exploration).

```bash
caster-generator analyze [options]
```

**Options:**

| Flag           | Description                             | Default |
|----------------|-----------------------------------------|---------|
| `-pkg <path>`  | Package path to analyze (repeatable)    | `./...` |
| `-verbose`     | Show detailed field info including tags | `false` |
| `-type <name>` | Filter to show only a specific type     | (all)   |

**Example:**

```bash
caster-generator analyze -pkg ./store -pkg ./warehouse -type Order
```

---

### `suggest` — Generate mapping suggestions

Generate a suggested YAML mapping for type pairs, with auto-matching of fields.

```bash
caster-generator suggest [options]
```

**Options:**

| Flag                           | Description                                             | Default                |
|--------------------------------|---------------------------------------------------------|------------------------|
| `-pkg <path>`                  | Package path (auto-detected from type names if omitted) | (auto)                 |
| `-mapping <file>`              | Path to existing YAML to improve                        | (none)                 |
| `-from <type>`                 | Source type (e.g., `store.Order`)                       | required if no mapping |
| `-to <type>`                   | Target type (e.g., `warehouse.Order`)                   | required if no mapping |
| `-out <file>`                  | Output YAML file                                        | stdout                 |
| `-min-confidence <float>`      | Minimum confidence for auto-matching                    | `0.7`                  |
| `-min-gap <float>`             | Minimum score gap between top candidates                | `0.15`                 |
| `-ambiguity-threshold <float>` | Score threshold for marking ambiguity                   | `0.1`                  |
| `-max-candidates <int>`        | Max candidates in suggestions                           | `5`                    |

**Examples:**

```bash
# Generate initial suggestion
caster-generator suggest -from store.Order -to warehouse.Order -out mapping.yaml

# Improve existing mapping with auto-matching
caster-generator suggest -mapping mapping.yaml -out mapping.yaml
```

---

### `gen` — Generate caster code

Generate Go caster functions from a YAML mapping file.

```bash
caster-generator gen [options]
```

**Options:**

| Flag                        | Description                          | Default             |
|-----------------------------|--------------------------------------|---------------------|
| `-pkg <path>`               | Package path to analyze (repeatable) | (auto from mapping) |
| `-mapping <file>`           | Path to YAML mapping file            | **required**        |
| `-out <dir>`                | Output directory for generated files | `./generated`       |
| `-package <name>`           | Package name for generated code      | `casters`           |
| `-strict`                   | Fail on any unresolved target fields | `false`             |
| `-write-suggestions <file>` | Write suggested mapping YAML         | (none)              |

**Example:**

```bash
caster-generator gen -mapping mapping.yaml -out ./generated -package casters
```

---

### `check` — Validate mapping

Validate YAML mapping against current code; fail on drift.

```bash
caster-generator check [options]
```

**Options:**

| Flag              | Description                          | Default             |
|-------------------|--------------------------------------|---------------------|
| `-pkg <path>`     | Package path to analyze (repeatable) | (auto from mapping) |
| `-mapping <file>` | Path to YAML mapping file            | **required**        |
| `-strict`         | Fail on any unresolved target fields | `false`             |

**Example:**

```bash
caster-generator check -mapping mapping.yaml
```

---

## YAML Mapping Schema

### Basic Structure

```yaml
version: "1"

mappings:
  - source: pkg.SourceType
    target: pkg.TargetType
    # ... mapping configuration

transforms:
  - name: MyTransform
    source_type: string
    target_type: int
```

### Type Mapping Options

| Field             | Type              | Description                                      |
|-------------------|-------------------|--------------------------------------------------|
| `source`          | string            | Source type identifier (e.g., `store.Order`)     |
| `target`          | string            | Target type identifier (e.g., `warehouse.Order`) |
| `requires`        | ArgDefArray       | Extra function arguments (context passing)       |
| `121`             | map[string]string | Simple 1:1 field name mappings                   |
| `fields`          | []FieldMapping    | Explicit field mappings with full control        |
| `ignore`          | []string          | Target fields to skip                            |
| `auto`            | []FieldMapping    | Auto-matched fields (lowest priority)            |
| `generate_target` | bool              | Generate target type if missing                  |

**Priority order:** `121` > `fields` > `ignore` > `auto`

---

### `121` — Simple 1:1 Mappings

Quick shorthand for direct field renames:

```yaml
"121":
  OrderID: ID
  CustomerName: Name
  EmailAddress: Email
```

---

### `fields` — Explicit Field Mappings

Full control over field mappings:

```yaml
fields:
  # Simple rename
  - source: OldName
    target: NewName

  # With transform function
  - source: PriceDollars
    target: PriceCents
    transform: DollarsToCents

  # Many-to-one (requires transform)
  - source: [ FirstName, LastName ]
    target: FullName
    transform: ConcatNames

  # Default value (no source)
  - target: Status
    default: "pending"

  # With introspection hints
  - source:
      Items: dive    # Force recursive introspection
    target:
      LineItems: dive

  # Extra context passing
  - source: Items
    target: LineItems
    extra:
      - name: OrderID
        def:
          target: OrderID  # Reference to already-assigned target field
```

---

### `ignore` — Skip Target Fields

Target fields that should not be mapped:

```yaml
ignore:
  - InternalID
  - CreatedAt
  - UpdatedAt
```

---

### `requires` — Context Passing

Pass extra arguments to the generated caster function:

```yaml
requires:
  - name: OrderID
    type: uint
  - name: UserContext
    type: "*context.Context"
```

Generated function signature:

```go
func APIItemToDomainLineItem(in APIItem, OrderID uint, UserContext *context.Context) DomainLineItem
```

---

### Transforms

#### Declaring Transforms

```yaml
transforms:
  - name: DollarsToCents
    source_type: float64
    target_type: int64
    package: myproject/transforms  # Optional: where the function lives
    func: ConvertDollarsToCents    # Optional: actual function name
    description: "Converts dollar amount to cents"
```

#### Using Transforms

```yaml
fields:
  - source: Price
    target: PriceCents
    transform: DollarsToCents
```

#### Transform Patterns

| Pattern            | Example Source  | Example Target | Transform        |
|--------------------|-----------------|----------------|------------------|
| Numeric conversion | `float64`       | `int64`        | `DollarsToCents` |
| String parsing     | `string`        | `time.Time`    | `ParseDateTime`  |
| Boolean conversion | `"Y"/"N"`       | `bool`         | `YNToBool`       |
| CSV splitting      | `"a,b,c"`       | `[]string`     | `CSVToSlice`     |
| Multi-source       | `[First, Last]` | `FullName`     | `ConcatNames`    |
| Pointer deref      | `*int`          | `int`          | `DerefInt`       |

#### Auto-Generated Transform Stubs

When incompatible types are detected, the tool can generate placeholder transforms:

```yaml
transforms:
  - name: TODO_Price_to_Amount
    source_type: float64
    target_type: int64
    auto_generated: true
```

---

### Introspection Hints

Control how nested types are handled:

| Hint    | Description                                          |
|---------|------------------------------------------------------|
| `dive`  | Force recursive introspection of inner struct fields |
| `final` | Treat as single unit requiring custom transform      |
| (none)  | Engine decides based on types and cardinality        |

**Usage:**

```yaml
fields:
  - source:
      Address: dive    # Recursively map Address fields
    target:
      ShippingAddress: dive

  - source:
      Metadata: final  # Don't introspect, use transform
    target:
      Meta: final
    transform: ConvertMetadata
```

---

### Nested and Recursive Maps

#### Nested Structs

When mapping types with nested struct fields:

```yaml
mappings:
  # Root type
  - source: APIOrder
    target: DomainOrder
    fields:
      - source:
          Customer: dive
        target:
          Customer: dive

  # Nested type (auto-detected and resolved)
  - source: APICustomer
    target: DomainCustomer
    "121":
      Name: FullName
```

#### Recursive/Self-Referential Types

The generator handles recursive types automatically:

```go
// Source
type Node struct {
    Value int
    Next  *Node
}

// Target
type NodeDTO struct {
    Val  int
    Next *NodeDTO
}
```

```yaml
mappings:
  - source: pkg.Node
    target: pkg.NodeDTO
    "121":
      Value: Val
    auto:
      - source: Next
        target: Next
```

#### Slices and Arrays

```yaml
# Slice of structs
fields:
  - source:
      Items: dive
    target:
      LineItems: dive

# Fixed-size array
fields:
  - source:
      Corners: dive   # [4]APIPoint -> [4]DomainPoint
    target:
      Corners: dive
```

---

### Suggestions

The `suggest` command produces YAML with:

1. **Auto-matched fields** in `auto:` section
2. **Confidence scores** as comments
3. **Unmapped fields** listed for review
4. **Candidate suggestions** for ambiguous matches

**Example output:**

```yaml
mappings:
  - source: store.Order
    target: warehouse.Order
    auto:
      - source: ID
        target: OrderID
        # confidence: 0.85, strategy: direct
      - source: CustomerName
        target: Name
        # confidence: 0.72, strategy: direct
    # unmapped targets:
    #   - ShippingAddress (no match found)
    #   - InternalCode (no match found, candidates: [Code: 0.45])
```

---

## Virtual Types

Generate target types on-the-fly when they don't exist.

### Usage

```yaml
mappings:
  - source: external.APIResponse
    target: internal.CleanResponse
    generate_target: true
    fields:
      - source: ID
        target: ID
      - source: RawData
        target: Data
      - source: Timestamp
        target: CreatedAt
```

### Features

- **Type inference**: Target field types inferred from source
- **Explicit types**: Use `target_type` for control

```yaml
fields:
  - source: Value
    target: Amount
    target_type: "int64"  # Explicit type
```

- **Nested virtual types**: Generate hierarchies

```yaml
mappings:
  - source: pkg.Container
    target: pkg.TargetContainer
    generate_target: true
    auto:
      - source: Items
        target: Items

  - source: pkg.Item
    target: pkg.TargetItem
    generate_target: true
    "121":
      Name: Name
```

- **Cross-package**: Generate types in different packages

### Generated Output

Virtual types are written to `missing_types.go` in the output directory:

```go
// Code generated by caster-generator. DO NOT EDIT.
package casters

type CleanResponse struct {
	ID        string
	Data      []byte
	CreatedAt time.Time
}
```

---

## Cardinality Support

| Cardinality | Source          | Target          | Transform Required         |
|-------------|-----------------|-----------------|----------------------------|
| 1:1         | single field    | single field    | only if types incompatible |
| 1:N         | single field    | multiple fields | no (cloning)               |
| N:1         | multiple fields | single field    | **yes**                    |
| N:M         | multiple fields | multiple fields | **yes**                    |

**Example N:1:**

```yaml
fields:
  - source: [ FirstName, LastName ]
    target: FullName
    transform: ConcatNames
```

---

## Conversion Strategies

The generator automatically selects conversion strategies:

| Strategy       | Description              | Example                   |
|----------------|--------------------------|---------------------------|
| `Direct`       | Types match exactly      | `string` → `string`       |
| `Convert`      | Basic type conversion    | `int32` → `int64`         |
| `PointerDeref` | Dereference pointer      | `*int` → `int`            |
| `PointerWrap`  | Wrap in pointer          | `int` → `*int`            |
| `NestedCast`   | Call nested caster       | `APIItem` → `DomainItem`  |
| `Transform`    | Apply transform function | `float64` → `int64`       |
| `SliceMap`     | Map over slice elements  | `[]A` → `[]B`             |
| `MapConvert`   | Convert map entries      | `map[K1]V1` → `map[K2]V2` |

---

## Extra Value Passing

Pass additional values to nested casters:

```yaml
fields:
  - source:
      Items: dive
    target:
      LineItems: dive
    extra:
      - name: OrderID
        def:
          source: ID           # From source field
      - name: ContextValue
        def:
          target: ProcessedID  # From already-assigned target field
```

The nested caster receives these as additional arguments (via `requires` on the nested mapping).

---

## Environment Variables

| Variable       | Description                       |
|----------------|-----------------------------------|
| `CG_NO_PROMPT` | Skip interactive prompts (for CI) |
| `CG_DEBUG`     | Show executed commands            |
| `CG_VERBOSE`   | More detailed output              |

---

## Workflow Pattern

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   suggest   │ ──▶ │ patch YAML  │ ──▶ │   suggest   │
│  (initial)  │     │  (manual)   │     │  (improve)  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   compile   │ ◀── │     gen     │ ◀── │  implement  │
│    check    │     │             │     │  transforms │
└─────────────┘     └─────────────┘     └─────────────┘
```

1. **Suggest**: Auto-generate initial YAML mapping
2. **Patch**: Review and adjust (add hints, ignores, transforms)
3. **Suggest improve**: Re-run to fill gaps
4. **Implement transforms**: Write custom conversion functions
5. **Generate**: Produce caster code
6. **Compile check**: Verify generated code compiles
