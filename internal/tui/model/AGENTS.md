# TUI Contracts

Define narrow interfaces shared by TUI state, services, rendering, and controllers.

- Keep contracts presentation-focused and implementation-independent.
- Add methods only for demonstrated consumer needs.
- Avoid concrete storage, tmux, or Bubble Tea implementation types where a domain value suffices.
- Interface changes require updating fakes and all implementations.

Verify consumers with: `go test ./internal/tui/...`
