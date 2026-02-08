# 🎯 caster-generator

**A magic tool that writes boring Go code for you!**

## 🤔 What Does It Do?

Imagine you have two toy boxes:

- **Box A** has toys with labels like `toy_name`, `toy_color`
- **Box B** needs the same toys but with labels like `Name`, `Color`
- **caster-generator** helps you move toys from Box A to Box B automatically! ✨

In programmer talk: it generates code to convert one Go struct to another. It handles:
- **Nested structs** (assigning `struct A` to `struct B`)
- **Collections** (Slices, Arrays, Maps, including nested ones like `map[K][]V`)
- **Pointers** (auto-dereferencing, nil checks)
- **Recursive types** (safe cycle detection)
- **Type mismatch** (via custom transform functions)

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

This creates a YAML file that says "put THIS toy from Box A into THAT spot in Box B"!

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

## 🎓 Examples Directory

The `examples/` directory contains runnable scenarios. Use them to learn by doing!

| Example                                  | Description                                               |
|------------------------------------------|-----------------------------------------------------------|
| [Basic](examples/basic/)                 | Simple renames and numeric type conversions               |
| [Nested](examples/nested-struct/)        | Handling nested structs and collections                   |
| [Transforms](examples/transforms/)       | Custom logic for type mismatches (e.g. `string` -> `int`) |
| [Arrays](examples/arrays/)               | Working with fixed-size arrays                            |
| [Recursion](examples/recursive-struct/)  | Handling self-referencing types                           |
| [Multi-Mapping](examples/multi-mapping/) | Chained conversions (A -> B -> C)                         |

Run any example:

```bash
cd examples/basic
./run.sh
```

---

## 📖 CLI Reference

### `analyze` - Look Inside Packages

Shows you all the structs in your packages.

```bash
./caster-generator analyze -pkg ./mypackage
```

### `suggest` - Get Smart Suggestions

Creates or improves a mapping file.

```bash
./caster-generator suggest -from store.Order -to warehouse.Order -out mapping.yaml
```

*Options:*

- `-min-confidence`: Threshold for auto-match (default 0.7)
- `-min-gap`: Required score gap between candidates (default 0.15)
- `-pkg`: Specify packages (auto-detected usually)

### `gen` - Generate Code

Creates actual Go code files.

```bash
./caster-generator gen -mapping mapping.yaml -out ./generated
```

*Options:*

- `-strict`: Stops if any field is missing
- `-package`: Custom package name (default `casters`)
- `-write-suggestions`: Dumps ignored fields to a sidecar file

### `check` - Validate Mapping

Validates your mapping against the current code.

```bash
./caster-generator check -mapping mapping.yaml -strict
```

---

## 🔧 Makefile Commands

```bash
make help     # Show all commands
make build    # Build the tool
make test     # Run tests
make lint     # Check code style
make bench    # Run benchmarks
make clean    # Clean up
make all      # Run everything!
```
---

# 🧠 Advanced Usage & Configuration

Everything below is for power users who need fine-grained control over the generation process.

## 1. The Mapping File Structure

The `mapping.yaml` file tells the tool how to match fields.

```yaml
version: "1"
# Define reusable transforms here (optional)
transforms:
  - name: StringToInt
    signature: "func(s string) (int, error)"
mappings:
  - source: pkg.SourceType
    target: pkg.TargetType
    # Priority 1: Simple Renames
    121:
      ID: OrderID
    # Priority 2: Advanced Field Mapping
    fields:
      - source: CreatedAt
        target: Timestamp
        transform: TimeToEpoch
    # Priority 3: Ignore Fields
    ignore:
      - InternalID
    # Priority 4: Auto-matched (managed by tool)
    auto:
      - source: Name
        target: Name
```

## 2. Advanced Field Mappings (`fields`)

The `fields` section allows fine-grained control.

### Transformations

When types don't match (e.g. `string` to `int`), provide a `transform` function name. You must implement these functions
in your Go code (or let the generator create stubs).

```yaml
fields:
  - source: TotalPrice
    target: PriceCents
    transform: DollarsToCents  # func(float64) int64
```

### Multi-Source Mapping (N:1)

Combine multiple source fields into one target field.

```yaml
fields:
  - source: [ FirstName, LastName ]
    target: FullName
    transform: ConcatNames     # func(string, string) string
```

### Deep Source Selection

Map a field from deep inside a nested structure to a flat field on the target.

```yaml
fields:
  - source:
      Meta:
        Details:
          Tag: value
    target: Tag
```

### Dive Hints for Collections

When mapping slices or maps of complex types, use `hint: dive` to tell the generator to look inside the collection.

```yaml
fields:
  # map[string]SourceItem -> map[string]TargetItem
  - source: Items
    target: LineItems
    hint: dive
```

## 3. Context Passing (`requires`)

Use `requires` and `extra` to pass external data (like IDs or config) into nested converters.
**Step 1: Declare requirement on the nested type**

```yaml
- source: store.Item
  target: warehouse.LineItem
  requires:
    - name: OrderID
      type: string
```

**Step 2: Pass the value from the parent**

```yaml
- source: store.Order
  target: warehouse.Order
  fields:
    - source: Items
      target: LineItems
      hint: dive
      extra:
        - name: OrderID    # Matches 'requires' name above
          def:
            target: ID     # Uses 'ID' field from warehouse.Order
```

## 4. Understanding Confidence Scores

The matching engine uses a score (0.0 - 1.0) to decide matches.

- **Name Similarity (60%)**: Levenshtein distance.
- **Type Compatibility (40%)**:
    - `1.0`: Identical types
    - `0.9`: Assignable types
    - `0.7`: Convertible types (e.g. `int` -> `float`)
    - `0.4`: Needs transform
    - `0.0`: Incompatible
      Adjust thresholds if suggestions are too loose or too strict:

```bash
./caster-generator suggest ... -min-confidence 0.8
```

---

## 📚 Detailed Use Cases

### Case A: Handling Transforms & Stubs

If you use a custom transform in your YAML:

1. **Declare it** in the `transforms` section (optional but recommended).
2. **Generate code**.
3. **Check output**: The generator creates `missing_transforms.go` with stubs for any functions it can't find.
4. **Implement**: Move the function to your own file and implement the logic. The generator will see it next time and
   remove the stub.

```go
// generated/missing_transforms.go
func DollarsToCents(v0 float64) int64 {
    panic("transform DollarsToCents not implemented")
}
```

### Case B: CI/CD Integration

You can use `check` to ensure your mappings are always up to date with your code.

```yaml
# .github/workflows/check.yml
steps:
  - run: make build
  - run: ./caster-generator check -mapping mapping.yaml -strict
```

If a developer changes a struct field but forgets to update the mapping, this step will fail!
---

## ❓ FAQ

**Q: What if fields have different types?**  
A: The tool handles basic conversions (like `int` to `int64`). For complex conversions, use transforms!
**Q: What if a field doesn't exist in the target?**  
A: It will be marked as "unmapped" in the diagnostics. You can ignore it or map it manually.
**Q: Can I use this with nested structs?**  
A: Yes! It handles nested structs, slices, arrays, and even recursive types automatically.
**Q: What if I change my structs later?**  
A: Run `check` to find what broke, then regenerate!
---
**Made with ❤️ to save you from writing boring code!**
