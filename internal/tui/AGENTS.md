# Terminal UI

## Responsibility

Provide interactive Bubble Tea presentation while keeping state transitions, services, rendering, and external interactions separated.

## Package map

- `app/`: construct and run TUI.
- `model/`: contracts shared across TUI components.
- `state/`: Bubble Tea model and update transitions.
- `service/`: notification, tree, and tmux coordination.
- `render/`: pure terminal views.
- `controller/`: interaction and jump decisions.

## Boundaries

- State owns transitions, not storage or tmux implementation details.
- Renderers must not perform I/O or mutate state.
- Put external effects behind model interfaces and services.
- Preserve keyboard behavior and inline error feedback with focused tests.

Run: `go test ./internal/tui/...`
