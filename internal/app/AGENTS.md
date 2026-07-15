# Application Use Cases

## Responsibility

Orchestrate CLI-facing workflows between parsed inputs and injected clients. This layer coordinates validation, defaults, calls, and user-facing outcomes without owning infrastructure.

## Boundaries

- Model each workflow with a small input and injected interface.
- Keep Cobra types, SQLite details, and direct tmux command execution out.
- Put reusable domain rules in `internal/domain`; put notification lifecycle behavior in `internal/core`.
- Fail explicitly and wrap dependency errors with operation context.

## Tests

Use fakes for interfaces. Cover successful orchestration and each dependency/error branch.

Run: `go test ./internal/app`
