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

## 📜 License

MIT - do whatever you want! 🎉

---

**Made with ❤️ to save you from writing boring code!**
