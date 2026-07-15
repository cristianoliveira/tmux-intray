# TUI Services

Implement notification filtering/grouping, tree operations, and tmux runtime coordination behind `internal/tui/model` contracts.

- Keep services independent of Bubble Tea message dispatch.
- Reuse domain filters, grouping, sorting, and search providers.
- Isolate tmux effects in runtime coordination.
- Keep tree transformations deterministic and test navigation boundaries.

Run: `go test ./internal/tui/service`
