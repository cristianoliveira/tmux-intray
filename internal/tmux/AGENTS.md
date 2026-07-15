# Tmux Adapter

## Responsibility

Wrap tmux process execution, context discovery, target listing, and pane navigation behind a mockable client.

## Boundaries

- Keep command construction and output parsing here.
- Return structured context and explicit errors; do not print user-facing messages.
- Do not add notification lifecycle or CLI flag behavior.
- Inject command runners in tests; never require a live tmux server for unit tests.

## Tests

Test exact arguments, parsing, missing-server behavior, malformed output, and runner failures.

Run: `go test ./internal/tmux`
