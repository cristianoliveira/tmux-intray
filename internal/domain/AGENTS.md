# Domain Model

## Responsibility

Define pure notification types, states, levels, filters, sorting, grouping, and repository contracts.

## Boundaries

- Keep this package free of I/O, CLI, TUI, SQLite, configuration, and tmux dependencies.
- Express invariants in domain types or pure functions.
- Prefer typed states and levels over raw strings at boundaries.
- Avoid importing infrastructure to make a domain operation convenient.

## Tests

Use table-driven deterministic tests for invariants and transformations. Test invalid values and empty collections.

Run: `go test ./internal/domain`
