# Caster Generator Refactoring Plan

This document coordinates refactoring efforts across the codebase. Package-level refactor plans live alongside the code (see links below), but this file defines the overall sequencing, invariants, and acceptance criteria.

## Refactor Contract (must not change)

These are the safety rails for all refactors in this repo:

- **YAML compatibility**: The mapping YAML schema stays compatible. Loading, defaults, and normalization must not break existing mapping files.
- **Determinism**:
  - Suggestion export (`internal/plan`) must be stable: ordering of types/fields/suggestions is deterministic.
  - Code generation (`internal/gen`) must be stable: file set, file names, import ordering/aliasing, and emitted code ordering are deterministic.
- **Type-walking semantics**: Pointer/leaf behavior must remain consistent between mapping validation and plan resolution (or be intentionally different with tests proving the contract).

If a change intentionally modifies behavior or output, the PR must:
1) state the intended diff, and 2) add/adjust tests to lock it in.

## Dependency Ordering (do this first → last)

1. `internal/mapping` (schema, YAML codecs, loader, validation)
2. `internal/plan` (resolver decomposition + suggestion export)
3. `internal/gen` (generator decomposition)

Rationale: `plan` depends on `mapping` primitives/semantics, and `gen` depends on the resolved plan shape and strategy semantics.

## Milestones (PR-sized)

### Milestone 0 — Lock behavior with tests
Acceptance criteria:
- `go test ./...` passes.
- Add/confirm tests focusing on determinism and seam boundaries (path parsing, hint resolution, strategy selection, export ordering, generator determinism).

### Milestone 1 — Refactor `internal/mapping` (move-only first)
Acceptance criteria:
- No output diffs in examples (except whitespace/comments if explicitly intended).
- Validation behavior unchanged (especially path validation + pointer deref rules).

### Milestone 2 — Refactor `internal/plan` resolver/export
Acceptance criteria:
- Resolver results unchanged for existing integration tests.
- Suggestion export ordering remains deterministic.
- Unit tests exist for strategy selection and pointer/leaf rules.

### Milestone 3 — Refactor `internal/gen` generator
Acceptance criteria:
- Generated code remains deterministic and compiles across existing examples/tests.
- Import aliasing/type formatting behavior unchanged (unless explicitly changed with tests).

### Milestone 4 — Optional cleanups
Only after milestones 0–3 are green:
- Consider de-duplicating type-walk helpers (likely via `internal/analyze` with explicit options).
- Consider relocating string-type-ID resolution into a more appropriate package if it becomes a stable API.

## Refactoring Modules (detailed package plans)

- `internal/mapping`: [internal/mapping/REFACTOR.md](internal/mapping/REFACTOR.md)
- `internal/plan`: [internal/plan/REFACTOR.md](internal/plan/REFACTOR.md)
- `internal/gen`: [internal/gen/REFACTOR.md](internal/gen/REFACTOR.md)
