# Search

## Responsibility

Provide interchangeable substring, token, and regular-expression matching strategies over notification and tmux display data.

## Boundaries

- Implement the shared provider contract; callers select strategy through dependency wiring.
- Keep matching deterministic and free of I/O.
- Return regex compilation or input errors explicitly.
- Centralize case-sensitivity and display-name options rather than duplicating matching logic.

## Tests

Use table-driven tests for matches, misses, empty input, case behavior, names, and invalid regexes.

Run: `go test ./internal/search`
