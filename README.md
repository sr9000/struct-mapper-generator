# caster-generator

A Go codegen tool that **analyzes two Go domain models** (for example `store.Order` and `warehouse.Order`) and **generates “caster” functions** to convert values between them.

This repo currently contains two example domains:

- `store/` — lightweight JSON/API domain types
- `warehouse/` — persistence/warehouse domain types (GORM-style, denormalized snapshots)

The generator’s job is to bridge these domains safely and repeatably.

---

## Decomposed abilities (grouped by domain)

### Store → Warehouse

**Goal:** Take `store.*` values and produce `warehouse.*` values suitable for persistence/fulfillment.

Capabilities:

- **Type casting / conversion generation**
  - Generates cast functions like `StoreOrderToWarehouseOrder(store.Order) warehouse.Order`.
  - Handles numeric widening/narrowing where safe (ex: `int64` → `uint`) with configurable guards.
  - Supports pointer lifting (ex: `time.Time` → `*time.Time`) and defaulting.

- **Snapshot + denormalization support**
  - Supports generating “snapshot” fields in the target (example: `warehouse.Order.ShippingAddress` embedded fields).
  - Allows mapping a single source value into multiple target fields (1 → many).

- **Collection and line-item conversion**
  - Converts slices/arrays (ex: `[]store.OrderItem` → `[]warehouse.OrderItem`).
  - Supports nested conversion and propagating parent keys where required.

### Warehouse → Store

**Goal:** Take `warehouse.*` values and produce `store.*` values suitable for APIs/UI.

Capabilities:

- **Type casting / conversion generation**
  - Generates cast functions like `WarehouseOrderToStoreOrder(warehouse.Order) store.Order`.
  - Handles embedded structs → flattened fields when needed.

- **Aggregation and projection**
  - Supports mapping many source fields into a single target field (many → 1)
    - Example: `FirstName` + `LastName` → `FullName`.

- **Filtering / shaping output**
  - Allows ignoring fields that don’t make sense to expose (e.g. `PasswordHash`).

---

## Matching & mapping engine

### Field and type matching (fuzzy)

When a mapping isn’t explicitly provided, the tool **suggests** candidate field mappings using a fuzzy match pipeline:

1. **Field-name normalization**
   - Normalizes identifiers before comparing.
   - Typical normalization includes:
     - case folding (`CreatedAt` → `createdat`)
     - removing separators/underscores/dashes (`postal_code` → `postalcode`)
     - optional suffix/prefix trimming (`ID`, `Ids`, `At`, etc.)

2. **Levenshtein distance scoring**
   - Uses **Levenshtein distance** (edit distance) between normalized names to rank candidates.
   - Lower distance = better match.

3. **Type-compatibility scoring**
   - Candidate mappings are adjusted based on type compatibility (exact match, convertible, needs transform).

4. **Conflict detection**
   - Detects ambiguous matches and reports them so they can be resolved via YAML overrides.

### Cardinality-aware mapping

The mapping model supports different “shapes”, not only 1:1:

- **1 → many**: one source field populates multiple target fields
  - Example: `store.Customer.FullName` → `warehouse.Customer.FirstName`, `warehouse.Customer.LastName`.
- **many → 1**: multiple source fields combine into one target field
  - Example: `warehouse.Customer.FirstName` + `warehouse.Customer.LastName` → `store.Customer.FullName`.
- **many ↔ many**: transform between collections/records
  - Example: `store.Order.Items` ↔ `warehouse.Order.Items` with per-item conversion.

---

## YAML mapping definitions

YAML mappings are a **first-class** part of the tool.

They let you:

- pin exact source→target field mappings (authoritative overrides)
- define 1→many / many→1 / many↔many transformations
- ignore fields
- configure defaults
- declare custom transforms (by name) used during code generation

A typical mapping file expresses:

- the **source type** (e.g. `store.Order`)
- the **target type** (e.g. `warehouse.Order`)
- field rules (direct map, ignore, default, transform)

(Exact schema may evolve; the intent is stable: YAML drives deterministic codegen and resolves fuzzy-match ambiguities.)

---

## Generate a new domain type (derived)

Beyond casting, the tool can generate a **new derived type** based on an existing one.

Use-cases:

- create a “view model” / “DTO” subset of a large domain type
- generate a denormalized snapshot type for persistence
- produce compatibility types while refactoring

Capabilities:

- pick a base type (e.g. `warehouse.Order`)
- select/rename fields
- apply tags (json/gorm)
- emit a new struct + optional caster to/from the base type

---

## What this repository demonstrates

These example domains intentionally differ so the generator must handle:

- different ID types (`int64` in `store` vs `uint` in `warehouse`)
- enum-like status (`store.OrderStatus`) vs string status (`warehouse.Order.Status`)
- denormalized address snapshots (`warehouse.Order` embeds addresses)
- nested relationships (`Order` → `Items`)

---

## Non-goals

- Not a runtime reflection mapper. Output is intended to be **generated Go code**.
- Not an ORM. It only converts data shapes.
