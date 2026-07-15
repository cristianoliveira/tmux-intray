# Development Guide

This project follows minimalist, Unix‑style principles focused on quiet notifications, persistent storage, and composable design. When making design decisions, see [Project Philosophy](docs/philosophy.md) for the guiding principles.

## Project Structure

```
tmux-intray/
├── cmd/                      # CLI entry point (Cobra)
│   ├── root.go               # Root command, global flags, initCLI wiring
│   └── tmux-intray/          # Command definitions (thin wiring only)
│       ├── main.go           # main() and run()
│       ├── deps.go           # DI wiring: storage → core → tui factories
│       ├── add.go, list.go, list_tabs.go, list_recents.go
│       ├── dismiss.go, clear.go, jump.go, mark-read.go
│       ├── follow.go, status.go, cleanup.go, settings.go
│       ├── tui.go, env_helpers.go
│       └── *_test.go         # One test file per command
├── internal/                 # Private application code
│   ├── app/                # Shared CLI use cases (thin orchestration)
│   ├── core/               # Core tmux interaction & notification business logic
│   ├── domain/             # Domain types (Notification, filters, sorting, grouping)
│   ├── ports/              # Port interfaces (NotificationRepository)
│   ├── storage/            # SQLite storage backend with sqlc-generated queries
│   ├── tmux/               # Tmux client abstraction
│   ├── config/             # Configuration management (TOML)
│   ├── settings/           # Settings persistence & types
│   ├── dedupconfig/        # Deduplication configuration
│   ├── search/             # Search providers (substring, token, regex)
│   ├── hooks/              # Hooks subsystem
│   ├── colors/             # Color output utilities
│   ├── errors/             # Error handling (CLI + TUI)
│   ├── format/             # Output formatting (table, simple, compact, JSON)
│   ├── formatter/          # Status presets & template variables
│   ├── notification/       # Notification conversion helpers
│   ├── logging/            # Structured logging
│   ├── version/            # Version info
│   └── tui/                # Terminal UI (Bubbletea)
│       ├── app/            # TUI entry point & client
│       ├── model/          # Interface contracts
│       ├── service/        # Service implementations
│       ├── state/          # Bubbletea Model (18 files)
│       ├── render/         # Pure view rendering
│       └── controller/     # Interaction controller
├── scripts/                  # Helper scripts (import graph, linting)
├── docs/                     # Documentation
├── Makefile                  # Build automation
├── go.mod                   # Go module definition
└── flake.nix                # Nix flake for dev environment
```

For detailed package descriptions and architecture layers, see [Go Package Structure](./docs/design/go-package-structure.md).

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

The CLI follows a layered architecture with Cobra framework and dependency injection:

1. **Main entry point** (`cmd/root.go` + `cmd/tmux-intray/main.go`):
   - `initCLI()` loads config, builds dependencies, registers commands
   - `main()` / `run()` are thin: parse args, execute Cobra, handle exit codes

2. **Commands** (`cmd/tmux-intray/*.go`):
   - Thin wiring only — no business logic
   - Each command receives dependencies via `cliDeps` struct
   - Delegate to `internal/app/` for use cases

3. **Shared use cases** (`internal/app/`):
   - Orchestration layer between commands and core logic
   - Testable without tmux or SQLite

4. **Internal packages** (`internal/*`):
   - `domain/` — Pure types: Notification, filters, sorting, grouping (zero deps)
   - `core/` — Business logic: add, dismiss, jump, tmux operations
   - `storage/` — SQLite persistence with sqlc-generated queries
   - `tmux/` — Tmux client abstraction
   - `config/` — Configuration loading (TOML)
   - `settings/` — Settings persistence (JSON)
   - `hooks/` — Hook subsystem
   - `search/` — Pluggable search providers
   - `format/` — CLI output formatting (table, simple, compact, JSON)
   - `formatter/` — Status presets and template variables
   - `tui/` — Bubbletea TUI (model, service, state, render, controller)
   - Support packages: `colors/`, `errors/`, `logging/`, `notification/`, `version/`, `dedupconfig/`

See [Go Package Structure](./docs/design/go-package-structure.md) for the full tree.

This structure makes the codebase:
- ✅ Type-safe with Go
- ✅ Easy to maintain (clear separation of concerns)
- ✅ Easy to extend (add new commands without touching existing ones)
- ✅ Easy to test (unit tests with mocked factories)
- ✅ Well-organized (layered architecture)

### Example: Add Command Structure

The add command is split across three layers:

**cmd/tmux-intray/add.go** — thin wiring:
```go
func NewAddCmd(coreClient cliCore) *cobra.Command {
    return &cobra.Command{
        Use: "add [message]",
        RunE: func(cmd *cobra.Command, args []string) error {
            return app.Add(coreClient, args, flagNoAssociate, flagLevel)
        },
    }
}
```

**internal/app/add.go** — use case orchestration:
```go
func Add(client CoreClient, args []string, noAssociate bool, level string) error {
    // Validate, auto-detect context, delegate to core
    id, err := client.AddTrayItem(message, session, window, pane, paneCreated, noAssociate, level)
    // Handle errors, output
}
```

**internal/core/core.go** — business logic:
```go
func (c *Core) AddTrayItem(...) (string, error) {
    // Validate, create Notification, persist via storage
}
```

## Key Patterns

### Storage Layer (internal/storage)
- SQLite backend with sqlc-generated queries in `internal/storage/sqlite/sqlcgen/`
- Domain adapter (`domain_repository_adapter.go`) maps to `NotificationRepository` interface
- State values: `domain.NotificationState` enum (active, dismissed, read)
- Level values: `domain.NotificationLevel` enum (info, warning, error)
- Factory pattern via `storage.NewFromConfig()` for DI

### Tmux Interaction (internal/tmux)

The `internal/tmux` package provides a clean client abstraction over tmux commands. The client is mockable for testing via interfaces in `internal/tmux/client.go`.

```go
import "github.com/cristianoliveira/tmux-intray/internal/tmux"

client := tmux.NewDefaultClient()

// Check if tmux is running
running := client.IsRunning()

// Get current tmux context
ctx, _ := client.GetCurrentContext()
// ctx.SessionID, ctx.WindowID, ctx.PaneID

// List sessions/windows/panes with names
sessions, _ := client.ListSessions()
windows, _ := client.ListWindows()
panes, _ := client.ListPanes()

// Jump to a pane
client.JumpToPane(sessionID, windowID, paneID)
```

The `internal/core` package wraps tmux operations with business logic (validation, error handling). Use `internal/tmux` directly for low-level tmux operations, and `internal/core` for notification-related tmux workflows.

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

### TOML Configuration Naming Conventions
- All TOML configuration files must use **snake_case** for keys and section names
- Examples: `config_dir`, `max_notifications`, `auto_cleanup_days`, not `configDir` or `config-dir`
- This convention is enforced by the automated linting script `scripts/lint-toml.sh`
- The linting script is automatically run as part of `make lint` and CI/CD pipeline
- To manually check TOML files for naming violations, run: `./scripts/lint-toml.sh` or `./scripts/lint-toml.sh path/to/file.toml`
- The linter detects and reports:
  - **camelCase** violations (e.g., `configDir` → should be `config_dir`)
  - **PascalCase** violations (e.g., `ConfigDir` → should be `config_dir`)
  - **kebab-case** violations (e.g., `config-dir` → should be `config_dir`)
  - Violations in both bare and quoted keys
- For detailed information, see [Configuration Guide](./docs/configuration.md)

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
```

## CI/CD Pipeline

tmux-intray uses GitHub Actions for continuous integration and deployment. For detailed documentation on the CI/CD pipeline, see [CI/CD Documentation](docs/ci-cd.md).

Key workflows:
- **CI**: Runs Go tests, linting, import graph validation, format checks on every push and pull request.
- **Release**: Automates release creation and binary building when tags are pushed.

## Further Reading

- [Go Package Structure](./docs/design/go-package-structure.md)
- [Import Layering Map](./docs/design/import-layering-map.md)
- [TUI Guidelines](./docs/design/tui/tui-guidelines.md)
- [Configuration Guide](./docs/configuration.md)
- [CLI Reference](./docs/cli/CLI_REFERENCE.md)
- [Hooks Documentation](./docs/hooks.md)
- [Troubleshooting Guide](./docs/troubleshooting.md)
