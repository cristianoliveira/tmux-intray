# TUI State

Own Bubble Tea `Model`, messages, key handling, cursor/tree state, settings state, and update transitions.

- Keep `Update` shallow by dispatching typed messages to focused handlers.
- Return commands for effects; do not execute storage or tmux I/O directly in transitions.
- Keep state changes deterministic and rendering outside this package where practical.
- Add tests for key paths, boundaries, empty data, errors, and asynchronous result messages.

Run: `go test ./internal/tui/state`
