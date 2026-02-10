# Go Package Structure for tmux-intray

## Overview

This document describes the Go package organization that preserves the modular architecture of the current Bash implementation. The goal is to provide a clear, maintainable structure that mirrors the existing separation of concerns while leveraging Go's type safety and standard library.

## Previous Bash Modular Architecture

The previous Bash implementation consisted of:

- **`bin/tmux-intray`**: Main entry point, command dispatch (removed)
- **`scripts/lib/`**: Shell libraries for tmux integration scripts only (legacy)
- **`commands/`**: Individual command implementations (add, list, dismiss, etc.)

## Current Go Implementation

The project now uses a pure Go implementation:

- **`tmux-intray`**: Go binary with all commands implemented natively
- **`cmd/tmux-intray/`**: Go command implementations

Each command sources the necessary libraries and implements a `{command}_command` function.

## Proposed Go Package Structure

```
github.com/cristianoliveira/tmux-intray/
├── cmd/                       # CLI command implementations
│   ├── add.go                 # Add command
│   ├── list.go                # List command
│   ├── dismiss.go             # Dismiss command
│   ├── clear.go               # Clear command
│   ├── toggle.go              # Toggle command
│   ├── jump.go                # Jump command
│   ├── status.go              # Status command
│   ├── status-panel.go        # Status-panel command
│   ├── follow.go              # Follow command
│   ├── help.go                # Help command
│   ├── version.go             # Version command
│   ├── cleanup.go             # Cleanup command
│   ├── root.go                # Root command and CLI entry point
│   └── wrapper/               # Wrapper for Bash migration
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
│   └── tmuxintray/            # Tmux integration utilities
├── pkg/                       # Public APIs (if any)
│   └── api/                   # External integration APIs
└── go.mod                     # Go module definition
```

## Package Descriptions

### `cmd/` (CLI Entry Point and Commands)

The CLI is structured as a flat set of command files in the `cmd/` directory:
- `root.go` defines the root command and handles command dispatch
- Each command file (e.g., `add.go`, `list.go`) contains Cobra command definition and business logic
- Commands delegate core functionality to internal packages (`core`, `storage`, etc.)
- The wrapper subdirectory contains the temporary wrapper for Bash migration

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

### Command Implementation in `cmd/`

Each command is implemented as a Cobra command defined directly in `cmd/*.go` files, combining CLI definition with business logic:
- Each command file contains the Cobra command definition, flag parsing, and a `Run` function
- Business logic delegates to internal packages (`core`, `storage`, `config`, etc.)
- This simpler structure reduces package nesting while maintaining clear separation between CLI and core logic

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