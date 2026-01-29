# Examples

This directory contains step-by-step examples demonstrating `caster-generator` features.

## Directory Structure

```
examples/
├── basic/           # Simple 1:1 struct mapping
│   ├── source.go    # Source DTOs (UserDTO, OrderDTO)
│   └── target.go    # Target domain models (User, Order)
│
├── nested/          # Nested struct and slice mapping
│   ├── source.go    # API types with nested structures
│   └── target.go    # Domain types with different nesting
│
├── transforms/      # Custom transform functions
│   ├── source.go    # Legacy types with string fields
│   └── target.go    # Modern types with typed fields
│
└── multi-mapping/   # Multiple related type conversions
    └── types.go     # External → Internal → Warehouse chain
```

## Quick Start

Each example can be run from its directory. Use `-pkg .` when types are in the current directory.

### Basic Example

```bash
cd basic
../../caster-generator analyze
../../caster-generator suggest -pkg . -from basic.UserDTO -to basic.User
```

### Nested Example

```bash
cd nested
../../caster-generator analyze -verbose
../../caster-generator suggest -pkg . -from nested.APIOrder -to nested.DomainOrder -out mapping.yaml
```

### Transforms Example

```bash
cd transforms
../../caster-generator suggest -pkg . -from transforms.LegacyProduct -to transforms.ModernProduct -min-confidence 0.4
```

### Multi-Mapping Example

```bash
cd multi-mapping
../../caster-generator suggest -pkg . -from multi.ExternalOrder -to multi.InternalOrder
../../caster-generator suggest -pkg . -from multi.InternalOrder -to multi.WarehouseOrder
```

## See Also

Refer to the main [README.md](../README.md) for detailed explanations of each scenario.
