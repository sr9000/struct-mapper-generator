# IMPLEMENTATION PLAN

Build a semi-automated Go codegen tool that:

- **Parses Go packages (AST + go/types)** to understand structs and relationships.
- **Suggests field mappings best-effort** (80% of the work for ~20% effort).
- Lets a human **review + lock mappings via YAML**.
- **Generates fast caster functions** (mapping is constructed once at generation time; runtime conversion is just Go code).

This tool is intentionally **human-supervised**: it prefers to be helpful and explainable over being “magical”. When it’s unsure, it should surface ambiguity instead of guessing silently.

---

## 0) Success criteria (contract)

**Inputs**
- Source package(s) + type (e.g. `store.Order`)
- Target package(s) + type (e.g. `warehouse.Order`)
- Optional YAML mapping definition(s)

**Outputs**
- Generated Go file(s) with caster functions (e.g. `StoreOrderToWarehouseOrder`)
- (Optional) exported “suggested mapping YAML” for review
- Diagnostics report (unmapped fields, ambiguous matches, unsafe conversions)

**Constraints / philosophy**
- Slow analysis is OK. Generation is an offline step.
- Generated casters should be small, readable, deterministic, and fast.
- Best-effort suggestions are a feature. The tool’s job is to reduce tedious mapping work.

---

## 1) Milestones (vertical slices)

### Milestone A — 1:1 struct mapping for a single pair
Goal: `store.Order` → `warehouse.Order` with simple compatible fields.

- [ ] Load both packages with types info
- [ ] Extract struct field lists and tags
- [ ] Generate a caster with direct assignments + basic type conversions
- [ ] Emit diagnostics for unmapped target fields

### Milestone B — Fuzzy suggestions + YAML overrides
Goal: tool can propose mappings and allow humans to pin overrides.

- [ ] Implement normalization + Levenshtein scoring
- [ ] Print a ranked candidate list per target field
- [ ] Add YAML schema to lock chosen mappings and ignores
- [ ] Regenerate deterministically from YAML

### Milestone C — Collections + nested structs
Goal: support `[]T`, pointers, and nested conversions.

- [ ] Slice mapping generator (`make`, loop, per-element conversion)
- [ ] Pointer lift/deref strategies with nil guards
- [ ] Nested struct conversions by composed generated casters

### Milestone D — Cardinality mappings + transforms
Goal: 1→many, many→1, and many↔many with named transforms.

- [ ] YAML rules for 1→many and many→1
- [ ] Named transform registry support (validated signatures)
- [ ] Codegen emits transform calls

---

## 2) Proposed repo layout

Keep the public surface tiny; put most logic under `internal/`.

- `cmd/caster-generator/`
  - CLI entrypoint
- `internal/analyze/`
  - Package loading + type graph extraction (AST + `go/types`)
- `internal/match/`
  - Name normalization, Levenshtein, scoring, candidate ranking
- `internal/mapping/`
  - YAML schema, parsing, validation, transform registry
- `internal/plan/`
  - Resolution pipeline: YAML pins + best-effort suggestions + diagnostics
- `internal/gen/`
  - Deterministic Go code generation + `go/format`
- `internal/diagnostic/`
  - Structured warnings/errors, “why this mapped” explanations

Decisions to make early:
- Where generated code lives: `./generated/` vs `./casters/`
- Generated package name: same as target, or dedicated `casters` package

---

## 3) Static analysis (AST + go/types): build a type graph once

### Requirements
The analyzer should produce a canonical in-memory model, so later phases don’t need to keep consulting AST.

### Implementation details
- Use `golang.org/x/tools/go/packages` with:
  - `NeedName`, `NeedFiles`, `NeedSyntax`, `NeedTypes`, `NeedTypesInfo`, `NeedImports`
- From the loaded packages, build:
  - `TypeID` = package import path + type name
  - `TypeInfo` describing:
    - kind: struct / basic / alias / pointer / slice / external(opaque)
    - underlying type information
  - `FieldInfo` describing:
    - Go field name (and whether exported)
    - field type: pointer/slice nesting, element types
    - tags (json/gorm), plus helper accessors (e.g. JSON name)
    - whether embedded / anonymous

### Edge-case policy (explicit 80/20)
Start with:
- exported struct fields only
- treat external types (e.g. `time.Time`) as “opaque” values (assign/convert only)
- no maps, no interfaces, no generics (reject with clear diagnostics)

Checklist
- [ ] Package loader + caching
- [ ] Struct field extraction (include tags)
- [ ] Type stringer for readable paths (`Order.Items[].ProductID`)

---

## 4) Matching & suggestions (best-effort engine)

### Field-name normalization
Used before fuzzy matching.

Suggested normalization pipeline:
- case-fold to lower
- strip separators (`_`, `-`, spaces)
- optionally strip common suffix/prefix tokens (`id`, `ids`, `at`, `utc`, etc.)
- optionally tokenize CamelCase for better matching (later improvement)

### Levenshtein distance
- Compute edit distance between normalized names
- Use it as the **primary name similarity score**
- Lower distance = better match

### Type compatibility scoring
For each candidate mapping, compute a type score:
- exact type match
- assignable in Go
- convertible in Go (`go/types` Convertibility)
- requires transform (allowed only if YAML specifies a transform)
- impossible

Combine scores into a ranked list. Always store “why”:
- normalized names
- Levenshtein distance
- type relation (“assignable”, “convertible”, “needs transform”)

### Ambiguity rules
- If top-1 and top-2 candidates are within a small threshold, mark as **ambiguous**.
- If many plausible candidates exist, prefer to output a **suggestion block** for human review.

Checklist
- [ ] `NormalizeIdent` + tests
- [ ] Levenshtein implementation + tests
- [ ] Type compatibility scorer using `go/types`
- [ ] Candidate ranking with deterministic tie-breakers

---

## 5) YAML mapping definitions (authoritative, human-reviewed)

YAML is a first-class feature. It turns best-effort suggestions into deterministic regeneration.

### MVP YAML capabilities
- pin explicit field mappings
- ignore target fields
- set defaults
- apply named transforms
- support path expressions for nested shapes

### Path syntax (MVP)
- `Field`
- `Nested.Field`
- `Items[]` for slice element
- `Items[].ProductID`

### Cardinality rules
Support via YAML (explicit):
- 1 → many (one source feeds multiple targets)
- many → 1 (multiple sources feed one target) — typically requires transform
- many ↔ many (collection conversions) — typically element-level mapping + transform

### Transform registry
YAML references transforms by name. The generator validates:
- transform exists (or generates a stub)
- signature matches expected types

Checklist
- [ ] Define YAML schema structs and load with `yaml.v3`
- [ ] Validate field paths against analyzed type graph
- [ ] Implement transform registry concept (names + signature checks)

---

## 6) Resolution pipeline (YAML wins, then suggestions)

Resolution produces a final `ResolvedMappingPlan` consumed by codegen.

Pipeline:
1. Analyze packages → type graph
2. Load YAML (optional) → validate
3. For each requested type pair:
   - apply YAML-pinned rules first
   - for remaining target fields:
     - compute candidates via fuzzy matcher
     - auto-accept only on high confidence
     - otherwise mark “needs review”
4. Emit diagnostics:
   - unmapped targets
   - ambiguity lists with top-N candidates
   - unsafe conversions

Optional artifact:
- Write `mapping.suggested.yaml` so humans can edit/approve and then promote to authoritative YAML.

Checklist
- [ ] Implement `ResolvedMappingPlan` model
- [ ] Confidence thresholds + deterministic ordering
- [ ] Suggestion export format

---

## 7) Code generation (fast casters)

Goal: generate readable, allocation-light Go.

### Generation approach
- Use either:
  - `text/template` + `go/format` (simple)
  - or build Go AST nodes + `format.Node` (more structured)

### Codegen patterns
- direct assignment
- numeric conversion with optional guards
- pointer lift/deref with nil checks
- slice mapping:
  - `out := make([]T, len(in))`
  - index loop for speed
- nested struct calls (call other generated caster)
- transforms: call named transform functions

### Determinism
- stable file naming
- stable field ordering
- stable import ordering

Checklist
- [ ] Implement emitters for assign/convert/pointer/slice/nested
- [ ] Generate comments/TODOs for unresolved target fields
- [ ] Run `go/format` and ensure `go test ./...` stays clean

---

## 8) CLI workflows (semi-automated)

Suggested commands:
- `analyze` — print discovered structs/fields (debug)
- `suggest` — generate a suggested YAML mapping for a type pair
- `gen` — generate casters using YAML + auto-accept policy
- `check` — validate YAML against current code; fail on drift

Suggested flags:
- `-fromType store.Order`
- `-toType warehouse.Order`
- `-mapping ./mapping.yaml`
- `-out ./generated`
- `-writeSuggestions ./mapping.suggested.yaml`
- `-strict` (fail on any unresolved)

Checklist
- [ ] Single shared pipeline used by all commands
- [ ] Clear, structured console output for ambiguities + “why”

---

## 9) 80/20 boundaries (explicit non-goals for v1)

Start strict and expand only if required:
- no generics, no maps, no interfaces
- no deep embedding flattening (later: support `embeddedPrefix` patterns)
- many→1 requires explicit transform rule
- narrow numeric conversions require explicit allow-policy

---

## Draft checklist (paste-friendly)

- [ ] Implement package loading + type graph extraction (`internal/analyze`)
- [ ] Implement normalization + Levenshtein + scoring (`internal/match`)
- [ ] Define YAML schema + validation + transform registry (`internal/mapping`)
- [ ] Implement mapping resolution + diagnostics + suggestion export (`internal/plan`)
- [ ] Implement deterministic caster generator (`internal/gen`)
- [ ] Add CLI commands: analyze/suggest/gen/check (`cmd/caster-generator`)
- [ ] Document workflow: suggest → review YAML → gen → commit
