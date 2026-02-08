# Development Guide

## Project Structure

```
tmux-intray/
├── cmd/                      # CLI command implementations (Cobra)
│   └── tmux-intray/         # Main entry point and all commands
│       ├── root.go           # Root command and CLI setup
│       ├── add.go            # Add command
│       ├── list.go           # List command
│       ├── dismiss.go        # Dismiss command
│       ├── clear.go          # Clear command
│       ├── toggle.go         # Toggle command
│       ├── jump.go           # Jump command
│       ├── status.go         # Status command
│       ├── cleanup.go        # Cleanup command
│       └── version.go       # Version command
├── internal/                 # Private application code
│   ├── core/               # Core tmux interaction & tray management
│   ├── storage/            # File-based TSV storage with locking
│   ├── colors/             # Color output utilities
│   ├── config/             # Configuration management
│   ├── hooks/              # Hook subsystem for async operations
│   └── tmuxintray/        # Library initialization and orchestration
├── lib/                      # Shared shell libraries (for integration)
│   ├── storage.sh          # TSV storage helpers
│   ├── config.sh           # Configuration helpers
│   ├── hooks.sh            # Hook system
│   └── colors.sh           # Color utilities
├── tests/                    # Integration tests (Bats)
│   ├── basic.bats          # Basic CLI tests
│   ├── storage.bats        # Storage tests
│   ├── tray.bats           # Tray management tests
│   └── commands/          # Command-specific tests
├── scripts/                  # Helper scripts
│   ├── lint.sh             # ShellCheck linter
│   ├── security-check.sh   # Security-focused ShellCheck
│   └── generate-docs.sh    # Documentation generator
├── tmux-intray.tmux         # Tmux plugin entry point
├── Makefile                  # Build automation
├── go.mod                   # Go module definition
└── flake.nix                # Nix flake for dev environment
```

## Adding a New Command

### Simple Command

1. Create a new command file in `cmd/tmux-intray/`:
   ```bash
   touch cmd/tmux-intray/mycommand.go
   ```

2. Implement the Cobra command:
   ```go
   package cmd

   import (
       "github.com/spf13/cobra"
       "github.com/cristianoliveira/tmux-intray/internal/core"
   )

   var mycommandCmd = &cobra.Command{
       Use:   "mycommand",
       Short: "Description of mycommand",
       Long:  `Longer description of mycommand.`,
       RunE: func(cmd *cobra.Command, args []string) error {
           // Command logic here
           core.DoSomething()
           return nil
       },
   }

   func init() {
       rootCmd.AddCommand(mycommandCmd)
   }
   ```

3. Add tests in `tests/commands/mycommand.bats`:
   ```bash
   #!/usr/bin/env bats
   # My command tests

   @test "mycommand does something" {
       run ./tmux-intray mycommand
       [ "$status" -eq 0 ]
   }
   ```

4. Run tests:
   ```bash
   go test ./...
   bats tests/commands/mycommand.bats
   ```

## Architecture

The CLI follows Go's standard project layout with Cobra framework:

1. **Main entry point** (`cmd/tmux-intray/root.go`):
   - Initializes all commands using Cobra
   - Sets up global flags and configuration
   - Handles command routing

2. **Commands** (`cmd/tmux-intray/*.go`):
   - Each command is a Cobra command
   - Commands delegate to internal packages for business logic
   - Use `RunE` for commands that can error

3. **Internal packages** (`internal/*`):
   - `core/` - Core tmux interaction (context detection, tray management)
   - `storage/` - TSV file storage with locking
   - `colors/` - Terminal color output
   - `config/` - Configuration loading
   - `hooks/` - Hook subsystem
   - `tmuxintray/` - Library initialization

4. **Libraries** (`lib/*.sh`):
   - Shared shell utilities for integration testing
   - TSV storage helpers
   - Configuration helpers

5. **Tests** (`tests/**/*.bats`):
   - Integration tests using Bats
   - Test CLI behavior end-to-end
   - Mock tmux environment

This structure makes the codebase:
- ✅ Type-safe with Go
- ✅ Easy to maintain (clear separation of concerns)
- ✅ Easy to extend (add new commands without touching existing ones)
- ✅ Easy to test (unit tests in Go, integration tests in Bats)
- ✅ Well-organized (standard Go layout)

### Example: Add Command Structure

```
cmd/tmux-intray/
├── add.go          # Cobra command definition and business logic
└── ...
```

**add.go** delegates to internal packages:
```go
package cmd

import (
    "github.com/cristianoliveira/tmux-intray/internal/core"
    "github.com/cristianoliveira/tmux-intray/internal/storage"
)

var addCmd = &cobra.Command{
    Use:   "add [message]",
    Short: "Add a notification to the tray",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Parse arguments
        message := args[0]

        // Get tmux context (auto-detection)
        ctx := core.GetCurrentTmuxContext()

        // Add to storage
        id := storage.AddNotification(message, "", ctx.Session, ctx.Window, ctx.Pane, ctx.PaneCreated, "info")

        // Output success
        colors.Success("Added notification: " + id)
        return nil
    },
}
```

## Key Patterns

### Storage Layer (internal/storage)
- TSV file format: `id\ttimestamp\tstate\tsession\twindow\tpane\tmessage\tpaneCreated\tlevel`
- File locking for concurrent access
- State values: "active" or "dismissed"
- Level values: "info", "warning", "error"
- Field indices defined as package constants

### Tmux Interaction (internal/core)
- Mock `tmuxRunner` for testing
- Use `GetCurrentTmuxContext()` for auto context detection
- Escape special characters in messages

#### Using the Tmux Abstraction Layer

The `internal/core` package provides a clean abstraction over tmux commands. **Never call `exec.Command("tmux", ...)` directly** from command implementations. Always use the core package functions.

##### Core Functions Available

**Tmux Context & Navigation:**
```go
import "github.com/cristianoliveira/tmux-intray/internal/core"

// Check if tmux is running
if !core.EnsureTmuxRunning() {
    colors.Error("tmux not running")
    return fmt.Errorf("tmux not available")
}

// Get current tmux context (auto-detection)
ctx := core.GetCurrentTmuxContext()
// ctx.SessionID   // e.g., "$3"
// ctx.WindowID    // e.g., "@16"
// ctx.PaneID      // e.g., "%21"
// ctx.PaneCreated // e.g., "8443"

// Validate a pane exists
if core.ValidatePaneExists(sessionID, windowID, paneID) {
    // Pane exists
}

// Jump to a specific pane
if core.JumpToPane(sessionID, windowID, paneID) {
    colors.Success("Jumped to pane")
} else {
    colors.Error("Failed to jump to pane")
}
```

**Visibility Management:**
```go
// Get current visibility (returns "0" or "1")
visible := core.GetVisibility()
if visible == "1" {
    // Tray is visible
}

// Set visibility
if err := core.SetVisibility(true); err != nil {
    colors.Error("Failed to set visibility: " + err.Error())
}
```

**Tray Management:**
```go
// Get tray items
items := core.GetTrayItems("active") // "active" or "dismissed"

// Add a tray item with auto-context detection
id, err := core.AddTrayItem("My notification", "", "", "", "", false, "info")
if err != nil {
    colors.Error("Failed to add: " + err.Error())
}

// Clear all active tray items
if err := core.ClearTrayItems(); err != nil {
    colors.Error("Failed to clear: " + err.Error())
}
```

##### Example: Complete Command Using Core Abstraction

```go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/cristianoliveira/tmux-intray/internal/core"
    "github.com/cristianoliveira/tmux-intray/internal/colors"
)

var jumpCmd = &cobra.Command{
    Use:   "jump <id>",
    Short: "Jump to notification source pane",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Validate tmux is running first
        if !core.EnsureTmuxRunning() {
            return fmt.Errorf("tmux not running")
        }

        // Get notification details from storage
        id := args[0]
        notif, err := storage.GetNotificationByID(id)
        if err != nil {
            return fmt.Errorf("notification not found: %w", err)
        }

        // Parse notification fields
        fields := strings.Split(notif, "\t")
        sessionID := fields[3] // session
        windowID := fields[4]  // window
        paneID := fields[5]    // pane

        // Jump to the pane using core abstraction
        if !core.JumpToPane(sessionID, windowID, paneID) {
            return fmt.Errorf("failed to jump to pane")
        }

        colors.Success("Jumped to notification " + id)
        return nil
    },
}
```

##### Testing with Tmux Abstraction

The `tmuxRunner` variable in `internal/core/tmux.go` can be replaced for testing:

```go
func TestJumpToPane(t *testing.T) {
    // Save original runner
    origRunner := core.tmuxRunner
    defer func() { core.tmuxRunner = origRunner }()

    // Replace with mock
    core.tmuxRunner = func(args ...string) (string, string, error) {
        switch args[0] {
        case "list-panes":
            // Mock panes list
            return "%21", "", nil
        case "select-window":
            return "", "", nil
        case "select-pane":
            return "", "", nil
        default:
            return "", "", fmt.Errorf("unexpected command: %v", args)
        }
    }

    // Test the function
    result := core.JumpToPane("$3", "@16", "%21")
    require.True(t, result)
}
```

### Colors Output (internal/colors)
- `Error(msg)` - Output to stderr in red
- `Success(msg)` - Output to stdout in green
- `Warning(msg)` - Output to stdout in yellow
- `Info(msg)` - Output to stdout in blue
- `Debug(msg)` - Output to stderr in cyan (only when TMUX_INTRAY_DEBUG set)

### Error Message Format
- Use lower-case messages with no trailing punctuation.
- When adding context, use the format `<component>: <message>` (component is a short, lower-case command, package, or function name).
- Prefer `id` over `ID` in messages unless part of a literal identifier.
- Wrap underlying errors with `%w` and keep the outer message lower-case.

### Hooks Subsystem (internal/hooks)
- Async background operations
- `Init()` to start, `Shutdown()` to stop
- `WaitForPendingHooks()` to flush on exit

## Development Workflow

```bash
# Enter dev environment with tools (Go, bats, shellcheck)
nix develop

# Run all tests (Go + Bats)
make tests

# Run Go tests only
go test ./...

# Run single Go test
go test -v ./internal/core -run TestAddTrayItem

# Run specific Bats test file
bats tests/basic.bats

# Run linter
make lint

# Run security check
make security-check

# Run both tests and lint
make all

# Format all code
make fmt

# Regenerate SQLite sqlc code
make sqlc-generate

# Verify generated sqlc output is up to date
make sqlc-check

# Run deterministic storage benchmarks and refresh baseline artifact
make benchmarks

# Run quick benchmark sanity check during development
make benchmarks-quick
```

## CI/CD Pipeline

tmux-intray uses GitHub Actions for continuous integration and deployment. For detailed documentation on the CI/CD pipeline, see [CI/CD Documentation](docs/ci-cd.md).

Key workflows:
- **CI**: Runs Go tests, Bats tests, linting, security checks, format checks, and install verification on every push and pull request.
- **Release**: Automates release creation and binary building when tags are pushed.

## Further Reading

- [Go Package Structure](./docs/design/go-package-structure.md)
- [Testing Strategy](./docs/testing/testing-strategy.md)
- [Configuration Guide](./docs/configuration.md)
- [Storage Benchmarks](./docs/storage-benchmarks.md)
- [CLI Reference](./docs/cli/CLI_REFERENCE.md)
- [Hooks Documentation](./docs/hooks.md)
- [Troubleshooting Guide](./docs/troubleshooting.md)
