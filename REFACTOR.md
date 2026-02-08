# Caster Generator Refactoring Plan

This document coordinates the refactoring efforts across the codebase. Refactoring plans are separated by package to maintain focus and clarity.

## Refactoring Modules

### 1. `internal/mapping`
Focuses on simplifying the YAML schema, loader, and validation logic.
- **Detailed Plan**: [internal/mapping/REFACTOR.md](internal/mapping/REFACTOR.md)
- **Key Goals**: Separate YAML mechanics from domain types, reduce validation complexity.

### 2. `internal/plan`
Focuses on decomposing the monolithic resolver and suggestion export logic.
- **Detailed Plan**: [internal/plan/REFACTOR.md](internal/plan/REFACTOR.md)
- **Key Goals**: Extract AutoMatch, Strategy Selection, and Virtual Type management into focused components.

### 3. `internal/gen`
Focuses on decomposing the monolithic generator and template data preparation logic.
- **Detailed Plan**: [internal/gen/REFACTOR.md](internal/gen/REFACTOR.md)
- **Key Goals**: Separate template data building, strategy application, and collection loop generation.

## Overall Objectives

- **Maintainability**: Reduce the size of monolithic files (`resolver.go`, `schema.go`).
- **Separation of Concerns**: Ensure each component has a single, clear responsibility.
- **Readability**: Make the complex matching and resolution logic easier for new contributors to follow.
