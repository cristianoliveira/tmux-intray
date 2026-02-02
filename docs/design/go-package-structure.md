# Go Package Structure for tmux-intray

## Overview

This document describes the Go package organization that preserves the modular architecture of the current Bash implementation. The goal is to provide a clear, maintainable structure that mirrors the existing separation of concerns while leveraging Go's type safety and standard library.

## Current Bash Modular Architecture

The current Bash implementation consists of:

- **`bin/tmux-intray`**: Main entry point, command dispatch
- **`lib/`**: Shared libraries (core, storage, colors, config, hooks)
- **`commands/`**: Individual command implementations (add, list, dismiss, etc.)

Each command sources the necessary libraries and implements a `{command}_command` function.

## Proposed Go Package Structure

```
github.com/cristianoliveira/tmux-intray/
├── cmd/
│   └── tmux-intray/           # CLI entry point (main)
│       └── main.go
├── internal/                  # Private application code
│   ├── core/                  # Core tmux interaction & tray management
│   │   ├── core.go
│   │   └── tmux.go
│   ├── storage/               # File-based TSV storage with locking
│   │   ├── storage.go
│   │   └── lock.go
│   ├── colors/                # Color output utilities
│   │   └── colors.go
│   ├── config/                # Configuration loading
│   │   └── config.go
│   ├── hooks/                 # Hooks subsystem
│   │   └── hooks.go
│   └── commands/              # Command implementations
│       ├── add/
│       │   └── add.go
│       ├── list/
│       │   └── list.go
│       ├── dismiss/
│       │   └── dismiss.go
│       ├── clear/
│       │   └── clear.go
│       ├── toggle/
│       │   └── toggle.go
│       ├── jump/
│       │   └── jump.go
│       ├── status/
│       │   └── status.go
│       ├── follow/
│       │   └── follow.go
│       ├── status-panel/
│       │   └── status-panel.go
│       ├── help/
│       │   └── help.go
│       └── version/
│           └── version.go
├── pkg/                       # Public APIs (if any)
│   └── api/                   # External integration APIs
└── go.mod                     # Go module definition
```

## Package Descriptions

### `cmd/tmux-intray`

The main CLI entry point that:
- Parses command-line arguments
- Dispatches to appropriate command implementation
- Handles global flags and version/help commands
- Initializes configuration and storage

### `internal/core`

Provides core tmux interaction and tray management:
- Tmux session/window/pane detection and validation
- Jump-to-pane functionality
- Tray visibility management (via tmux environment variables)
- Notification item abstraction

### `internal/storage`

File-based TSV storage with advisory locking:
- Atomic operations using directory-based locks (`flock` equivalent)
- TSV file format: `id\ttimestamp\tstate\tsession\twindow\tpane\tmessage\tpane_created\tlevel`
- Functions for add, list, dismiss, count, cleanup
- Handles concurrent access from multiple tmux sessions

### `internal/colors`

Color output utilities for terminal:
- ANSI color codes for different notification levels (info, warning, error)
- Consistent color themes across commands

### `internal/config`

Configuration management:
- Loads from `~/.config/tmux-intray/config.toml` (or `.json`/`.yaml`)
- Default values for storage paths, colors, hooks
- Environment variable overrides

### `internal/hooks`

Hook subsystem for extensibility:
- Pre/post notification hooks
- Custom script execution
- Event-driven architecture

### `internal/commands`

Each command is implemented as a separate package following the Command pattern:
- Each package exports a `Run(args []string) error` function
- Commands receive parsed arguments and have access to shared dependencies (storage, config, etc.)
- Command-specific logic and validation

## Design Principles

1. **Separation of Concerns**: Each package has a single responsibility.
2. **Dependency Injection**: Shared dependencies (storage, config) are passed to commands via interfaces.
3. **Testability**: Packages are independently testable with mocked dependencies.
4. **Backward Compatibility**: The Go implementation should maintain the same CLI interface and behavior as the Bash version.
5. **Gradual Migration**: The Go binary can initially embed Bash scripts, then gradually replace components.

## Migration Strategy

1. **Phase 1**: Maintain existing embed wrapper while designing package structure.
2. **Phase 2**: Implement core packages (storage, config) and switch to Go-based storage.
3. **Phase 3**: Implement command packages one by one, maintaining compatibility.
4. **Phase 4**: Remove Bash script dependencies and fully transition to Go.

## Implementation Notes

- Use `cobra` or `urfave/cli` for CLI framework (decision pending)
- Use `github.com/golang/go` standard library for file operations and locking
- Consider `github.com/sevlyar/go-daemon` for background processes (follow command)
- Use `github.com/spf13/viper` for configuration if needed

## Next Steps

1. Create scaffold directories and placeholder `.go` files
2. Define interfaces for storage, config, etc.
3. Implement storage package with tests
4. Implement core tmux integration
5. Implement first command (e.g., `add`) as proof-of-concept

## References

- [Current Bash Implementation](../implementation/tmux-intray-implementation-plan.md)
- [Go Standard Project Layout](https://github.com/golang-standards/project-layout)
- [Go Package Design](https://rakyll.org/style-packages/)