# TUI Guidelines and Resilience Standards

## Overview

This document defines the TUI architecture boundaries, package layout, and
resiliency practices for `tmux-intray`. The goal is to preserve a clean
separation between state, rendering, input, and side effects while keeping the
TUI easy to test and safe to evolve.

## Previous TUI Structure

The earlier implementation is concentrated in a small number of command files:

- **`cmd/tmux-intray/tui.go`**: UI model, rendering, input handling, and control
  flow in one file
- **`cmd/tmux-intray/follow.go`**: Follow mode orchestration tied to TUI behavior

## Current TUI Implementation

The project currently exposes a single TUI entry point that mixes concerns:

- `cmd/tmux-intray/tui.go` renders views, handles input, updates state, and
  coordinates follow behavior
- `cmd/tmux-intray/follow.go` integrates with the TUI loop and shares logic with
  the UI state

## Proposed TUI Package Structure

```
github.com/cristianoliveira/tmux-intray/
├── cmd/
│   ├── tui.go                  # TUI command entry point
│   └── follow.go               # Follow command entry point
├── internal/
│   └── tui/
│       ├── state/              # UI state model and reducers
│       ├── render/             # Pure view rendering
│       ├── input/              # Keybindings and event mapping
│       ├── runtime/            # TUI loop, lifecycle, error handling
│       └── follow/             # Follow orchestration and integration
```

## Package Descriptions

### `cmd/` (TUI Commands)

The command layer should remain a thin wrapper:
- Parse CLI flags and configuration
- Assemble dependencies
- Invoke `internal/tui/runtime` or `internal/tui/follow` entry points

### `internal/tui/state`

State model and reducers:
- Defines view state and derived values
- Exposes action types for user input and side effects
- Updates state via pure reducer-style functions

### `internal/tui/render`

Rendering helpers:
- Pure view construction (state in, view out)
- No file, network, or tmux I/O
- Rendering decisions are deterministic and testable

### `internal/tui/input`

Input handling:
- Defines keybindings and event mapping
- Translates terminal events into state actions
- Avoids direct state mutation

### `internal/tui/runtime`

Lifecycle wiring:
- Runs the event loop and orchestrates state, input, and rendering
- Owns cancellation, shutdown, and recovery behavior
- Centralizes error handling and user-facing failure states

### `internal/tui/follow`

Follow mode orchestration:
- Integrates follow behavior with the runtime via a narrow interface
- Avoids direct access to internal UI state where possible
- Keeps follow logic isolated from view rendering

### Command Implementation in `cmd/`

The command files should be small and declarative:
- Each command defines flags and invokes a single runtime entry point
- Business logic remains inside `internal/tui/*` packages
- Errors are surfaced through consistent, user-facing messages

## Design Principles

1. **Separation of Concerns**: State, rendering, input, and effects live in
   dedicated packages.
2. **Pure Rendering**: Rendering functions are deterministic and side-effect
   free.
3. **Resilience**: Runtime handles cancellations, errors, and cleanup in one
   place.
4. **Testability**: State transitions and render output are directly testable.
5. **Minimal Command Layer**: CLI files are wiring only, not logic.

## Migration Strategy

1. **Phase 1**: Extract state and reducers into `internal/tui/state`.
2. **Phase 2**: Move rendering helpers into `internal/tui/render`.
3. **Phase 3**: Isolate input mapping into `internal/tui/input`.
4. **Phase 4**: Introduce `internal/tui/runtime` and rewire the TUI loop.
5. **Phase 5**: Move follow orchestration into `internal/tui/follow`.

## Implementation Notes

- Prefer explicit error propagation and surface failures via colors.Error.
- Keep runtime shutdown idempotent; always release resources on exit.
- Use context cancellation for long-running operations and follow mode.
- Avoid direct tmux calls from render/input packages.

## Next Steps

1. Define the `state` model and action types
2. Extract render helpers and add snapshot-style tests
3. Map keybindings in `input` and validate action dispatch
4. Build a runtime wrapper with centralized error handling
5. Move follow-mode orchestration behind a narrow interface

## References

- `cmd/tmux-intray/tui.go`
- `cmd/tmux-intray/follow.go`
