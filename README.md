# 🎯 caster-generator

**A magic tool that writes boring Go code for you!**

## 🤔 What Does It Do?

Imagine you have two toy boxes:
- **Box A** has toys with labels like `toy_name`, `toy_color`
- **Box B** needs the same toys but with labels like `Name`, `Color`

**caster-generator** helps you move toys from Box A to Box B automatically! ✨

In programmer talk: it generates code to convert one Go struct to another.

---

## 🚀 Quick Start (5 Steps!)

### Step 1: Build the Tool
```bash
make build
```

### Step 2: Look at Your Structs
```bash
./caster-generator analyze -pkg ./store -pkg ./warehouse
```
This shows you what's inside your "toy boxes"!

### Step 3: Get Magic Suggestions
```bash
./caster-generator suggest -from store.Order -to warehouse.Order -out mapping.yaml
```
This creates a file that says "put THIS toy from Box A into THAT spot in Box B"!

### Step 4: Generate the Code
```bash
./caster-generator gen -mapping mapping.yaml -out ./generated
```
This writes the boring code for you! 🎉

### Step 5: Check Everything is OK
```bash
./caster-generator check -mapping mapping.yaml
```
This makes sure nothing is broken!

---

## 📖 All Commands

### `analyze` - Look Inside Packages
Shows you all the structs (toy boxes) in your packages.

```bash
./caster-generator analyze -pkg ./mypackage
```

| Option | What It Does |
|--------|--------------|
| `-pkg` | Which package to look at (you can use this many times!) |
| `-type` | Only show one specific struct |
| `-verbose` | Show extra details like tags |

**Example:**
```bash
./caster-generator analyze -pkg ./store -pkg ./warehouse -verbose
```

---

### `suggest` - Get Smart Suggestions
Creates a YAML file with mapping suggestions!

```bash
./caster-generator suggest -from store.Order -to warehouse.Order -out mapping.yaml
```

| Option | What It Does |
|--------|--------------|
| `-from` | Source struct (where toys come FROM) |
| `-to` | Target struct (where toys go TO) |
| `-out` | Where to save the YAML file |
| `-pkg` | Extra packages to load (optional) |
| `-min-confidence` | How sure it needs to be (default: 0.7 = 70%) |

**Example:**
```bash
./caster-generator suggest -from store.Customer -to warehouse.Customer -out customer-mapping.yaml
```

---

### `gen` - Generate Code 
Creates actual Go code files!

```bash
./caster-generator gen -mapping mapping.yaml -out ./generated
```

| Option | What It Does |
|--------|--------------|
| `-mapping` | Your YAML mapping file (required!) |
| `-out` | Where to put generated files (default: `./generated`) |
| `-package` | Package name for generated code (default: `casters`) |
| `-pkg` | Extra packages to load (optional) |
| `-strict` | Stop if any field can't be mapped |
| `-write-suggestions` | Save suggestions to another file |

**Example:**
```bash
./caster-generator gen -mapping mapping.yaml -out ./converters -package converters
```

---

### `check` - Validate Mapping
Makes sure your mapping file still works with your code!

```bash
./caster-generator check -mapping mapping.yaml
```

| Option | What It Does |
|--------|--------------|
| `-mapping` | Your YAML mapping file (required!) |
| `-pkg` | Extra packages to load (optional) |
| `-strict` | Fail if any field is unmapped |

**Example:**
```bash
./caster-generator check -mapping mapping.yaml -strict
```

---

## 📝 The Mapping File (YAML)

The mapping file tells the tool how to match fields:

```yaml
version: "1"
mappings:
  - source: store.Order      # FROM this struct
    target: warehouse.Order  # TO this struct
    
    # Simple 1-to-1 mappings (field A goes to field B)
    121:
      ID: OrderID
      CustomerID: CustomerID
    
    # Fields to skip
    ignore:
      - InternalField
      - TempData
```

### YAML Sections Explained:

| Section | What It Does |
|---------|--------------|
| `source` | The struct you're copying FROM |
| `target` | The struct you're copying TO |
| `121` | Simple field mappings (source field → target field) |
| `fields` | Complex mappings (when you need transforms) |
| `ignore` | Fields to skip |
| `auto` | Automatically matched fields |

---

## 🔧 Makefile Commands

```bash
make help     # Show all commands
make build    # Build the tool
make test     # Run tests
make lint     # Check code style
make bench    # Run benchmarks
make cover    # See test coverage
make clean    # Clean up
make all      # Run everything!
```

---

## 🎨 Complete Example

Let's convert a `store.Product` to `warehouse.Product`!

**1. Your source struct (`store/types.go`):**
```go
type Product struct {
    ID          int64  `json:"id"`
    Name        string `json:"name"`
    PriceCents  int64  `json:"price_cents"`
}
```

**2. Your target struct (`warehouse/types.go`):**
```go
type Product struct {
    ProductID   uint   `json:"product_id"`
    ProductName string `json:"product_name"`
    Price       int64  `json:"price"`
}
```

**3. Generate suggestions:**
```bash
./caster-generator suggest -from store.Product -to warehouse.Product -out product.yaml
```

**4. Edit `product.yaml` if needed:**
```yaml
version: "1"
mappings:
  - source: store.Product
    target: warehouse.Product
    121:
      ID: ProductID
      Name: ProductName
      PriceCents: Price
```

**5. Generate the code:**
```bash
./caster-generator gen -mapping product.yaml -out ./generated
```

**6. Use the generated code:**
```go
import "myproject/generated"

func main() {
    storeProduct := store.Product{ID: 1, Name: "Widget", PriceCents: 999}
    warehouseProduct := generated.StoreProductToWarehouseProduct(storeProduct)
}
```

---

## ❓ FAQ

**Q: What if fields have different types?**  
A: The tool handles basic conversions (like `int` to `int64`). For complex conversions, use transforms!

**Q: What if a field doesn't exist in the target?**  
A: It will be marked as "unmapped" in the diagnostics. You can ignore it or map it manually.

**Q: Can I use this with nested structs?**  
A: Yes! It handles nested structs and slices automatically.

**Q: What if I change my structs later?**  
A: Run `check` to find what broke, then regenerate!

---

## 🔥 Advanced Topics

This section covers advanced features for power users who need fine-grained control.

### Sensible Defaults

Commands now have sensible defaults to reduce boilerplate:

| Command | Default Behavior |
|---------|------------------|
| `analyze` | Scans `./...` (current directory tree) if no `-pkg` specified |
| `suggest` | Auto-detects packages from qualified type names (e.g., `store.Order` → `./store`) |
| `gen` | Outputs to `./generated`, package name `casters` |
| `check` | Auto-detects packages from mapping file |

**Examples with defaults:**
```bash
# Just run analyze in your project root - discovers all types
./caster-generator analyze

# Suggest mapping - packages auto-detected from type names
./caster-generator suggest -from store.Order -to warehouse.Order

# Generate with all defaults
./caster-generator gen -mapping mapping.yaml
```

---

### Understanding Confidence Scores

The matching engine uses a combined score (0.0 - 1.0) based on:

| Component | Weight | Description |
|-----------|--------|-------------|
| **Name Similarity** | 60% | Levenshtein distance on normalized names |
| **Type Compatibility** | 40% | How well types can be converted |

**Type Compatibility Levels:**

| Level | Score | Meaning |
|-------|-------|---------|
| `TypeIdentical` | 1.0 | Exact same type |
| `TypeAssignable` | 0.9 | Direct assignment works |
| `TypeConvertible` | 0.7 | Go type conversion works |
| `TypeNeedsTransform` | 0.4 | Custom transform required |
| `TypeIncompatible` | 0.0 | Cannot convert |

**Thresholds:**

| Setting | Default | Purpose |
|---------|---------|---------|
| `MinConfidence` | 0.7 | Minimum score for auto-accept |
| `MinGap` | 0.15 | Gap between top 2 candidates |
| `AmbiguityThreshold` | 0.1 | Marks pairs as ambiguous |

Lower confidence with `-min-confidence 0.5` to see more suggestions (but review carefully!).

---

### YAML Mapping Deep Dive

#### Priority Order

When resolving mappings, the tool applies rules in this order:

1. **`121`** - Simple 1:1 explicit mappings (highest priority)
2. **`fields`** - Complex field mappings with transforms
3. **`ignore`** - Explicit ignores
4. **`auto`** - Auto-generated suggestions (lowest priority)

#### Field Path Syntax

```yaml
# Simple field
target: Name

# Nested field (dot notation)
target: Address.City

# Array element access (not yet supported, planned)
target: Items[0].Name
```

#### Introspection Hints

Control how the engine handles nested structures:

```yaml
fields:
  # "dive" - recursively map inner fields
  - target: { Address: dive }
    source: { ShippingAddr: dive }
  
  # "final" - treat as single unit, needs transform
  - target: { Config: final }
    source: { Settings: final }
    transform: ConvertConfig
```

**When to use:**
- `dive`: Nested structs that share similar field structure
- `final`: Complex types requiring custom conversion logic

#### Cardinality Mappings

```yaml
fields:
  # 1:1 - one source to one target
  - target: FullName
    source: Name
  
  # 1:N - one source to many targets
  - target: [FirstName, DisplayName]
    source: Name
  
  # N:1 - many sources to one target (requires transform)
  - target: FullName
    source: [FirstName, LastName]
    transform: ConcatNames
  
  # N:M - many to many (requires transform)
  - target: [FullAddress, City]
    source: [Street, City, State]
    transform: FormatAddress
```

#### Transform Functions

Define custom transforms in the YAML:

```yaml
transforms:
  - name: ConcatNames
    signature: "func(first, last string) string"
    # Implementation goes in your codebase
    
  - name: ParseWeight
    signature: "func(s string) (int, error)"

mappings:
  - source: legacy.Product
    target: modern.Product
    fields:
      - target: FullName
        source: [FirstName, LastName]
        transform: ConcatNames
```

---

## 📚 Step-by-Step Examples

### Example 1: Basic 1:1 Mapping

**Scenario:** Map simple DTOs between API and domain layers.

```bash
cd examples/basic

# Step 1: Analyze available types (current directory)
../../caster-generator analyze

# Step 2: Generate suggested mapping (specify -pkg . since types are in current dir)
../../caster-generator suggest -pkg . -from basic.UserDTO -to basic.User -out user-mapping.yaml

# Step 3: Review and edit the generated YAML
cat user-mapping.yaml

# Step 4: Generate the converter code
../../caster-generator gen -mapping user-mapping.yaml -out ./generated

# Step 5: Validate everything works
../../caster-generator check -mapping user-mapping.yaml
```

**Sample mapping file (`user-mapping.yaml`):**
```yaml
version: "1"
mappings:
  - source: basic.UserDTO
    target: basic.User
    121:
      ID: UserID
      FullName: Name
      IsActive: Active
    auto:
      - target: Email
        source: Email
      - target: CreatedAt
        source: CreatedAt
```

---

### Example 2: Nested Struct Mapping

**Scenario:** Convert deeply nested API response to domain model.

```bash
cd examples/nested

# Step 1: See the complex type structure
../../caster-generator analyze -verbose

# Step 2: Generate mapping for the main type (nested types auto-discovered)
../../caster-generator suggest -pkg . -from nested.APIOrder -to nested.DomainOrder -out order-mapping.yaml

# Step 3: Review - notice nested type mappings are created automatically
cat order-mapping.yaml

# Step 4: Customize nested mappings if needed
# Edit order-mapping.yaml to adjust APIAddress -> DomainAddress mappings

# Step 5: Generate all converters
../../caster-generator gen -mapping order-mapping.yaml -out ./generated -strict
```

**Key features demonstrated:**
- Automatic nested struct discovery
- Slice type handling (`[]APIItem` → `[]DomainLineItem`)
- Pointer handling (`*APIAddress` → `*DomainAddress`)

---

### Example 3: Using Transforms

**Scenario:** Legacy data with strings needs conversion to typed fields.

```bash
cd examples/transforms

# Step 1: Analyze types
../../caster-generator analyze -verbose

# Step 2: Generate initial mapping (use -pkg . for current directory)
../../caster-generator suggest -pkg . -from transforms.LegacyProduct -to transforms.ModernProduct -out product-mapping.yaml -min-confidence 0.5

# Step 3: Notice many fields need transforms
# Edit product-mapping.yaml to add transforms
```

**Sample mapping with transforms (`product-mapping.yaml`):**
```yaml
version: "1"

transforms:
  - name: DollarsTooCents
    signature: "func(dollars float64) int64"
  - name: ParseWeight
    signature: "func(weight string) int"
  - name: YNToBool
    signature: "func(yn string) bool"
  - name: ParseDateTime
    signature: "func(s string) time.Time"
  - name: SplitCSV
    signature: "func(s string) []string"

mappings:
  - source: transforms.LegacyProduct
    target: transforms.ModernProduct
    121:
      Code: SKU
      Title: Name
    fields:
      - target: PriceCents
        source: PriceUSD
        transform: DollarsTooCents
      - target: WeightGrams
        source: Weight
        transform: ParseWeight
      - target: IsActive
        source: Available
        transform: YNToBool
      - target: UpdatedAt
        source: LastModified
        transform: ParseDateTime
      - target: Tags
        source: Categories
        transform: SplitCSV
```

**Then create the transform implementations (`transforms.go`):**
```go
package transforms

import (
    "strconv"
    "strings"
    "time"
)

func DollarsTooCents(dollars float64) int64 {
    return int64(dollars * 100)
}

func ParseWeight(weight string) int {
    // "2.5kg" -> 2500 grams
    weight = strings.TrimSuffix(weight, "kg")
    f, _ := strconv.ParseFloat(weight, 64)
    return int(f * 1000)
}

func YNToBool(yn string) bool {
    return strings.ToUpper(yn) == "Y"
}

func ParseDateTime(s string) time.Time {
    t, _ := time.Parse("2006-01-02 15:04:05", s)
    return t
}

func SplitCSV(s string) []string {
    return strings.Split(s, ",")
}
```

---

### Example 4: Multi-Mapping Pipeline

**Scenario:** External order → Internal order → Warehouse order (chain of conversions).

```bash
cd examples/multi-mapping

# Step 1: Analyze all types
../../caster-generator analyze

# Step 2: Create mappings for each conversion step (use -pkg . for current directory)
../../caster-generator suggest -pkg . -from multi.ExternalOrder -to multi.InternalOrder -out external-to-internal.yaml
../../caster-generator suggest -pkg . -from multi.InternalOrder -to multi.WarehouseOrder -out internal-to-warehouse.yaml

# Step 3: Combine into single mapping file (or keep separate)
# Edit to create all-mappings.yaml with both conversions

# Step 4: Generate all converters at once
../../caster-generator gen -mapping all-mappings.yaml -out ./generated
```

**Combined mapping file (`all-mappings.yaml`):**
```yaml
version: "1"
mappings:
  # Step 1: External API -> Internal Domain
  - source: multi.ExternalOrder
    target: multi.InternalOrder
    121:
      ExtOrderID: ExternalRef
      BuyerEmail: CustomerEmail
      BuyerName: CustomerName
      Items: LineItems
      ShipTo: ShippingAddress
    fields:
      - target: PlacedAt
        source: OrderDate
        transform: ParseISODate
      - target: TotalCents
        source: TotalAmount
        transform: DollarsToCents

  # Step 2: Internal Domain -> Warehouse Fulfillment
  - source: multi.InternalOrder
    target: multi.WarehouseOrder
    121:
      ExternalRef: OrderNumber
      CustomerName: Recipient
      LineItems: Items
      ShippingAddress: Address
    fields:
      - target: Priority
        default: "1"
```

---

### Example 5: CI/CD Integration

**Validate mappings in CI pipeline:**

```yaml
# .github/workflows/mapping-check.yml
name: Mapping Validation
on: [push, pull_request]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build caster-generator
        run: make build
      
      - name: Validate all mappings
        run: |
          for f in mappings/*.yaml; do
            echo "Checking $f..."
            ./caster-generator check -mapping "$f" -strict
          done
      
      - name: Regenerate and verify no drift
        run: |
          ./caster-generator gen -mapping mappings/all.yaml -out ./generated
          git diff --exit-code ./generated
```

---

### Example 6: Debugging Suggestions

When auto-suggestions aren't what you expect:

```bash
# Lower confidence to see all candidates
./caster-generator suggest -from store.Order -to warehouse.Order -min-confidence 0.3

# The warnings section shows why fields weren't matched:
# - "below threshold 0.70" - increase min-confidence to see it
# - "ambiguous: top candidates X and Y are too close" - manual mapping needed

# Check specific type in detail
./caster-generator analyze -pkg ./store -type Order -verbose
./caster-generator analyze -pkg ./warehouse -type Order -verbose

# Compare field names side by side to understand matching issues
```

---

## 🛠️ Configuration Reference

### Full YAML Schema

```yaml
version: "1"  # Schema version

# Custom transform function definitions
transforms:
  - name: TransformName
    signature: "func(input Type) (output Type, error)"

# Type pair mappings
mappings:
  - source: package.SourceType     # Source struct (required)
    target: package.TargetType     # Target struct (required)
    
    # Simple 1:1 mappings (highest priority)
    121:
      SourceField: TargetField
      AnotherField: AnotherTarget
    
    # Complex field mappings
    fields:
      - target: FieldName           # Single field
        source: SourceField         # Single source
        transform: TransformFunc    # Optional transform
        
      - target: [Field1, Field2]    # Multiple targets
        source: SingleSource        # 1:N mapping
        
      - target: SingleTarget        # N:1 mapping
        source: [Src1, Src2]
        transform: CombineFunc      # Required for N:1
        
      - target: Field
        default: "value"            # Literal default
    
    # Fields to skip (source fields to ignore)
    ignore:
      - InternalField
      - TempData
    
    # Auto-generated (lowest priority, managed by tool)
    auto:
      - target: Field
        source: MatchedField
```

### CLI Quick Reference

```bash
# Analyze - discover types
caster-generator analyze                           # All types in ./...
caster-generator analyze -pkg ./mypackage          # Specific package
caster-generator analyze -type Order -verbose      # Single type with tags

# Suggest - generate mapping YAML
caster-generator suggest -from pkg.Src -to pkg.Dst          # To stdout
caster-generator suggest -from pkg.Src -to pkg.Dst -out f.yaml  # To file
caster-generator suggest ... -min-confidence 0.5            # Lower threshold

# Generate - create Go code
caster-generator gen -mapping m.yaml                        # Defaults: ./generated, casters
caster-generator gen -mapping m.yaml -out ./conv -package converters
caster-generator gen -mapping m.yaml -strict                # Fail on unmapped
caster-generator gen -mapping m.yaml -write-suggestions s.yaml

# Check - validate mapping
caster-generator check -mapping m.yaml                      # Basic validation
caster-generator check -mapping m.yaml -strict              # Strict mode
```

---

## 📜 License

MIT - do whatever you want! 🎉

---

**Made with ❤️ to save you from writing boring code!**
