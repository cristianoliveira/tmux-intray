# Go Package Structure for tmux-intray

## Overview

This document describes the Go package organization of tmux-intray. The project uses a layered architecture with clear separation between CLI entry points, shared application use cases, domain logic, and infrastructure.

## Package Tree

```
github.com/cristianoliveira/tmux-intray/
├── cmd/                               # CLI entry point (Cobra)
│   ├── root.go                        # Root command, global flags, initCLI wiring
│   └── tmux-intray/                   # All command definitions (thin wiring only)
│       ├── main.go                    # main() → run() → cobra
│       ├── deps.go                    # DI wiring: storage → core → tui factories
│       ├── add.go, list.go, list_tabs.go, list_recents.go
│       ├── dismiss.go, clear.go, jump.go, mark-read.go
│       ├── follow.go, status.go, cleanup.go, settings.go
│       ├── tui.go, env_helpers.go
│       └── *_test.go                  # One test file per command
├── internal/                          # Private application code
│   ├── app/                           # Shared CLI use cases (thin, no I/O)
│   │   ├── add.go, list.go, clear.go, dismiss.go
│   │   ├── follow.go, jump.go, mark_read.go
│   │   ├── cleanup.go, status.go, settings.go
│   │   ├── display_names.go, stale_filter.go
│   │   └── *_test.go
│   ├── core/                          # Core tmux interaction & notification business logic
│   │   ├── core.go, jump.go
│   │   ├── tmux.go, tmux_navigation.go, tmux_runtime.go
│   │   └── *_test.go
│   ├── domain/                        # Domain types & pure functions (zero deps)
│   │   ├── notification.go            # Notification, NotificationState, NotificationLevel
│   │   ├── filters.go, sorting.go
│   │   ├── grouping.go, session_grouping.go
│   │   ├── repository.go              # NotificationRepository interface (port)
│   │   └── *_test.go
│   ├── ports/                         # Port interfaces for domain boundary
│   │   └── ports.go
│   ├── storage/                       # SQLite storage persistence
│   │   ├── storage.go, factory.go, init.go, lock.go
│   │   ├── interface.go, fields.go
│   │   ├── domain_repository_adapter.go
│   │   ├── sqlite_driver.go
│   │   ├── sqlite/                    # SQLite implementation
│   │   │   ├── storage.go, read.go, cleanup.go, dismiss.go
│   │   │   ├── tmux.go, errors.go, schema_embed.go
│   │   │   ├── sqlcgen/               # sqlc-generated queries
│   │   │   └── *_test.go
│   │   └── *_test.go
│   ├── tmux/                          # Tmux client abstraction
│   │   ├── client.go, config.go, context.go, environment.go
│   │   ├── lists.go, navigation.go, errors.go, mock.go
│   │   └── *_test.go
│   ├── config/                        # Configuration loading
│   │   ├── config.go, dedup_config.go, validators.go
│   │   └── *_test.go
│   ├── settings/                      # Settings persistence & types
│   │   ├── settings.go, manager.go, path.go
│   │   ├── constants.go, tab.go, validation.go
│   │   └── *_test.go
│   ├── dedupconfig/                   # Deduplication configuration options
│   │   ├── options.go
│   │   └── *_test.go
│   ├── search/                        # Search providers
│   │   ├── provider.go, substring.go, token.go, regex.go, mock.go
│   │   └── *_test.go
│   ├── hooks/                         # Hooks subsystem
│   │   ├── hooks.go, api.go, execution.go
│   │   ├── types.go, runtime_config.go
│   │   └── *_test.go
│   ├── colors/                        # Color output utilities
│   │   ├── colors.go, structured.go
│   │   └── *_test.go
│   ├── errors/                        # Error handling (CLI + TUI)
│   │   ├── handler.go, tui_handler.go, colors_output.go
│   │   └── *_test.go
│   ├── format/                        # Output formatting (CLI)
│   │   ├── formatter.go, table.go, notification.go
│   │   ├── status.go, tmuxintray.go
│   │   └── *_test.go
│   ├── formatter/                     # Status presets & template variables
│   │   ├── presets.go, template.go, variables.go
│   │   └── *_test.go
│   ├── notification/                  # Notification conversion helpers
│   │   ├── notification.go, converter.go
│   │   └── *_test.go
│   ├── logging/                       # Structured logging
│   │   ├── logger.go, structured.go
│   │   └── *_test.go
│   ├── version/                       # Version info
│   │   ├── version.go
│   │   └── version_test.go
│   └── tui/                           # TUI (Bubbletea-based)
│       ├── app/                       # TUI entry point & client
│       │   ├── client.go, interfaces.go
│       │   └── *_test.go
│       ├── model/                     # Interface contracts
│       │   ├── ui_state.go, notification_service.go
│       │   ├── repository.go, tree_service.go
│       │   ├── runtime_coordinator.go, controller.go
│       │   └── *_test.go
│       ├── service/                   # Service implementations
│       │   ├── notification_service.go, tree_service.go
│       │   ├── tmux_coordinator.go
│       │   ├── notification_helpers.go, tree_helpers.go
│       │   ├── notification_service_per_session.go
│       │   ├── tree_service_nav.go, tree_service_stats.go
│       │   └── *_test.go
│       ├── state/                     # TUI Model (Bubbletea)
│       │   ├── model.go, ui_state.go
│       │   ├── model_keys.go, model_keys_core.go
│       │   ├── model_key_handlers.go, model_actions.go
│       │   ├── model_movement.go, model_render.go
│       │   ├── model_notifications.go, model_stale.go
│       │   ├── model_settings.go, model_tree_*.go
│       │   ├── settings_service.go, tree_service.go
│       │   ├── messages.go, tree.go
│       │   └── *_test.go
│       ├── render/                    # Pure view rendering
│       │   ├── render.go, group_row.go
│       │   └── *_test.go
│       └── controller/                # Interaction controller (jump targets)
│           ├── interaction_controller.go
│           └── *_test.go
├── docs/                              # Documentation
├── ext/                               # Extensions (Pi agent, OpenCode)
├── scripts/                           # Shell scripts (import graph, linting)
├── Makefile
├── go.mod
└── flake.nix
```

## Package Descriptions

### `cmd/` — CLI Entry Point

- `cmd/root.go` — Root Cobra command, global flags, `initCLI()` wiring
- `cmd/tmux-intray/main.go` — `main()` and `run()` thin wrappers
- `cmd/tmux-intray/deps.go` — DI factory wiring: `storage → core → tui → cliDeps`
- `cmd/tmux-intray/*.go` — One file per command, thin wiring only. Delegates to `internal/app/` for use cases.

Rules:
- Commands are wiring only. No business logic.
- Dependency injection via `cliDepsFactories` (mockable in tests).
- One test file per command.

### `internal/app/` — Shared CLI Use Cases

Thin orchestration layer between commands and core logic. Each file mirrors a CLI command:
- Decouples CLI wiring from business logic
- Testable without tmux or SQLite
- Uses interfaces for all dependencies

### `internal/core/` — Core Business Logic

Tmux interaction, notification lifecycle, jump-to-pane.
- `core.go` — Notification add, list, dismiss, mark-read operations
- `jump.go` — Jump navigation
- `tmux.go`, `tmux_navigation.go`, `tmux_runtime.go` — Tmux command execution

### `internal/domain/` — Domain Types (zero infrastructure deps)

Pure Go: structs, enums, filters, sorting, grouping. No I/O.
- `notification.go` — `Notification` struct with `Validate()`, `MarkRead()`, `Dismiss()`, `MatchesFilter()`
- `filters.go` — Filter logic
- `sorting.go` — Sort strategies
- `grouping.go`, `session_grouping.go` — Group by session/window/pane
- `repository.go` — `NotificationRepository` interface (port)

### `internal/ports/` — Port Interfaces

Domain boundary interfaces. Used by domain and implemented by infrastructure.
- `ports.go` — `NotificationRepository`

### `internal/storage/` — SQLite Persistence

SQLite backend with sqlc-generated type-safe queries.
- `factory.go` — Storage factory from config
- `sqlite/` — SQLite implementation: CRUD, cleanup, tmux field tracking
- `sqlite/sqlcgen/` — sqlc-generated query code
- `domain_repository_adapter.go` — Adapts storage to `NotificationRepository` interface

### `internal/tmux/` — Tmux Client

Tmux command execution abstraction.
- `client.go` — Main client with mockable runner
- `context.go` — Current session/window/pane detection
- `lists.go` — List sessions, windows, panes with name resolution
- `navigation.go` — Jump, select operations

### `internal/config/` — Configuration

TOML-based config loading from `~/.config/tmux-intray/config.toml`.
- Supports environment variable overrides
- Deduplication config in `dedup_config.go`
- Validation in `validators.go`

### `internal/settings/` — Settings Persistence

TUI settings (tab preferences, group headers), persisted as JSON.
- `manager.go` — Load/save/reset
- `constants.go` — Default values
- `tab.go`, `validation.go` — Tab types and validation

### `internal/dedupconfig/` — Dedup Options

Configuration types for notification deduplication.

### `internal/search/` — Search Providers

Pluggable search strategies: substring, token, regex.
- `provider.go` — Search interface
- `substring.go`, `token.go`, `regex.go` — Implementations
- `mock.go` — Test double

### `internal/hooks/` — Hooks Subsystem

Pre/post notification hooks, custom script execution.
- `hooks.go` — Hook runner
- `api.go` — Public API
- `execution.go` — Script execution
- `types.go` — Hook types and point constants
- `runtime_config.go` — Runtime configuration

### `internal/colors/` — Color Output

ANSI color helpers for terminal output.
- `colors.go` — Simple colored output
- `structured.go` — Structured debug logging

### `internal/errors/` — Error Handling

CLI and TUI error handlers.
- `handler.go` — CLI error messages
- `tui_handler.go` — TUI inline error display
- `colors_output.go` — Colored error output

### `internal/format/` — Output Formatting (CLI)

Table, simple, compact, JSON, and template formatting for CLI output.
- `formatter.go` — Base formatting layer
- `table.go` — Table view format
- `notification.go` — Notification output
- `status.go` — Status output
- `tmuxintray.go` — Format-specific helpers

### `internal/formatter/` — Status Presets & Templates

Status command presets: compact, summary, detailed, legacy.
- `presets.go` — Built-in presets
- `template.go` — Template engine
- `variables.go` — Template variable substitution

### `internal/notification/` — Notification Conversion

Converters between domain `Notification` and string representations.

### `internal/logging/` — Structured Logging

JSON-structured logging to file, with log rotation.

### `internal/version/` — Version Info

Build version information injected at compile time.

### `internal/tui/` — Terminal UI (Bubbletea)

Interactive TUI with five sub-packages:

#### `tui/app/` — TUI Entry Point
Client wrapper, TUI interfaces.

#### `tui/model/` — Interface Contracts
Pure Go interfaces defining TUI component boundaries: `UIState`, `NotificationService`, `TreeService`, `RuntimeCoordinator`, `InteractionController`.

#### `tui/service/` — Service Implementations
Concrete implementations of model interfaces.
- `notification_service.go` — Filter, search, group notifications
- `tree_service.go` — Hierarchical tree build and navigation
- `tmux_coordinator.go` — Tmux runtime integration
- `notification_service_per_session.go` — Per-session notification logic
- `tree_service_nav.go`, `tree_service_stats.go` — Tree helpers

#### `tui/state/` — TUI Model (Bubbletea)
The core Bubbletea `Model` struct and 18 supporting files.
- `model.go` — `NewModel()`, `Init()`, `Update()`
- `model_keys.go`, `model_keys_core.go` — Key bindings
- `model_key_handlers.go` — Key handlers (enter, backspace, runes)
- `model_actions.go` — Dismiss, jump, mark-read actions
- `model_movement.go` — Cursor movement
- `model_render.go` — View rendering dispatch
- `model_notifications.go` — Notification loading
- `model_stale.go` — Stale target detection
- `model_settings.go` — Settings tab
- `model_tree_*.go` — Tree build, expansion, navigation, view
- `ui_state.go` — UI state management
- `messages.go` — Bubbletea message types
- `tree.go` — Tree node struct
- `settings_service.go`, `tree_service.go` — Service wrappers

#### `tui/render/` — View Rendering
Pure rendering functions (state in, string out). No I/O.
- `render.go` — Main view construction
- `group_row.go` — Grouped notification rows

#### `tui/controller/` — Interaction Controller
Handles jump target resolution and navigation decisions.

## Design Principles

1. **Separation of Concerns**: Each package has a single responsibility.
2. **Dependency Injection**: Shared dependencies passed via interfaces through factories (`deps.go`).
3. **Testability**: All packages independently testable with mocked dependencies.
4. **Domain at the Center**: `internal/domain/` has zero infrastructure dependencies.
5. **Thin Commands**: CLI commands (`cmd/tmux-intray/`) are wiring only — business logic in `internal/app/` and `internal/core/`.

## Architecture Layers

See [Import Layering Map](./import-layering-map.md) for the explicit dependency edges between layers:

1. **CLI** (`cmd/`) — Entry points, wiring
2. **Presentation** (`internal/tui/`, `internal/format/`, `internal/formatter/`) — Output, rendering
3. **Application** (`internal/app/`, `internal/core/`) — Use cases, business logic
4. **Domain** (`internal/domain/`, `internal/ports/`, `internal/notification/`, `internal/search/`, `internal/dedup/`) — Pure types
5. **Infrastructure** (`internal/storage/`, `internal/tmux/`, `internal/config/`, `internal/hooks/`, `internal/settings/`, `internal/colors/`, `internal/errors/`, `internal/logging/`, `internal/version/`, `internal/dedupconfig/`) — External dependencies

## References

- [Import Layering Map](./import-layering-map.md)
- [TUI Guidelines](./tui/tui-guidelines.md)
- [DEVELOPMENT.md](../../DEVELOPMENT.md)
- [CLI Reference](../cli/CLI_REFERENCE.md)
