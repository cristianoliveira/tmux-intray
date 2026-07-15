# TUI Application

Construct and run Bubble Tea with configured repositories, services, settings, and tmux collaborators.

- Treat this package as TUI composition, not business logic.
- Pass configured clients into services and state.
- Keep terminal startup/shutdown errors explicit.
- Test construction and run behavior through injected interfaces; avoid real terminal or tmux dependencies.

Run: `go test ./internal/tui/app`
