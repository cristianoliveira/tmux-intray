# Go Package Design for tmux-intray Migration

## Purpose & Scope

Direct migration of Bash libraries to Go with backward compatibility:
- TSV storage format identical (tab-separated, same fields, escaping rules)
- Hook execution preserved (when implemented)
- CLI flags and behavior unchanged
- Environment variable precedence (env > config file > defaults)
- tmux integration unchanged (same tmux options, status updates)

## Package Map Table

| Bash module | Go package (path) | Key responsibilities | API surface (functions/structs) | Dependencies | Compatibility notes |
|-------------|-------------------|----------------------|---------------------------------|--------------|---------------------|
| storage.sh | `internal/storage` | TSV file storage with locking, notification CRUD, active count, tmux status updates | `type Storage struct`, `AddNotification`, `ListNotifications`, `DismissNotification`, `DismissAll`, `GetActiveCount` | internal/config, internal/tmux, internal/tsv, internal/types | TSV format identical, locking via `flock` or directory atomicity, same file paths (XDG state dir) |
| config.sh | `internal/config` | Load configuration from environment, config file, defaults; sample config creation | `type Config struct`, `Load()`, `Get()`, `SaveSample()` | none | Same env var names, config file format (shell script) parsed as key-value, defaults identical |
| (domain) | `internal/types` | Notification struct, enums (State, Level), validation helpers | `type Notification struct`, `type State`, `type Level`, `ValidateNotification()` | none | Struct fields match TSV columns; zero values safe |
| hooks.sh (missing) | `internal/hooks` | Execute pre/post hooks for notifications; manage hook scripts | `type HookManager struct`, `ExecutePreAdd`, `ExecutePostAdd` | internal/config, internal/types | Hook scripts remain Bash for compatibility; execution sync/async TBD |
| core.sh | `internal/core` | Tmux context retrieval, pane validation, jump, tray operations, visibility | `type Core struct`, `GetTmuxContext`, `ValidatePane`, `JumpToPane`, `AddItem`, `GetItems`, `ClearItems`, `GetVisibility`, `SetVisibility` | internal/storage, internal/config, internal/tmux, internal/types | Same tmux display -p format, same fallback behavior |
| colors.sh | `internal/log` | Colored output for errors, warnings, info, success; logging levels | `Error()`, `Warning()`, `Info()`, `Success()`, `SetLevel()` | none | Output colors match existing; optional structured logging |
| tmux-intray.sh (placeholder) | `cmd/tmux-intray` | Main CLI entry point; command routing | `main()`, command subcommands | internal/core, internal/config, internal/storage, internal/log | Same subcommands (show, status, toggle, dismiss, help) |
| commands/*.sh | `cmd/tmux-intray` (subcommands) | Individual command implementations | `runShow()`, `runStatus()`, `runToggle()`, `runDismiss()`, `runHelp()` | internal/core, internal/log | Same CLI flags, output format, exit codes |

## Detailed Package Briefs

### internal/config

**Overview & inputs/outputs**
- Loads configuration from environment variables (`TMUX_INTRAY_*`), config file (`$TMUX_INTRAY_CONFIG_DIR/config.sh`), and defaults.
- Provides typed access to configuration values.
- Creates sample config file if missing.

**Public interfaces**
```go
type Config struct { ... }
func Load() (*Config, error)
func (c *Config) Get(key string) string
func (c *Config) Int(key string, default int) int
func (c *Config) Bool(key string, default bool) bool
func SaveSample(path string) error
```

**Errors/logging approach**
- Errors returned from Load if config file exists but cannot be parsed; fallback to defaults.
- Log via internal/log at info level when config loaded.

**Testing notes**
- Unit tests: loading from env vars, file, precedence.
- Golden fixtures for config file parsing.
- Integration test with temporary XDG directories.

### internal/types

**Overview & inputs/outputs**
- Defines domain types: Notification, State, Level.
- Provides validation and conversion helpers.
- Shared across storage, hooks, core.

**Public interfaces**
```go
type Notification struct {
    ID string
    Timestamp time.Time
    State State
    Session string
    Window string
    Pane string
    Message string
    PaneCreated string
    Level Level
}
type State string // "active", "dismissed"
type Level string // "info", "warning", "error", "critical"
func ValidateNotification(n Notification) error
func ParseState(s string) (State, error)
func ParseLevel(s string) (Level, error)
```

**Errors/logging approach**
- Validation errors returned as error.
- No logging.

**Testing notes**
- Unit tests for validation and parsing.
- Golden compatibility with Bash string values.

### internal/storage

**Overview & inputs/outputs**
- Manages TSV files in `TMUX_INTRAY_STATE_DIR` with atomic locking (directory-based lock).
- Provides notification operations: add, list, dismiss, count.
- Updates tmux option `@tmux_intray_active_count` via internal/tmux.
- Escapes/unescapes tabs and newlines in message field.

**Public interfaces**
```go
// Notification type imported from internal/types
type Notification = types.Notification
type Storage struct { ... }
func NewStorage(config *config.Config, tmux *tmux.Client) (*Storage, error)
func (s *Storage) AddNotification(msg string, opts AddOptions) (id string, error)
func (s *Storage) ListNotifications(filter Filter) ([]types.Notification, error)
func (s *Storage) DismissNotification(id string) error
func (s *Storage) DismissAll() error
func (s *Storage) GetActiveCount() (int, error)
```

**Errors/logging approach**
- Errors returned for invalid IDs, lock timeouts, I/O errors.
- Log debug for lock acquisition, file writes.

**Testing notes**
- Unit tests with temporary directory.
- Integration tests with actual tmux (optional).
- Golden TSV fixtures for compatibility.
- Concurrency tests for locking.

### internal/hooks (future)

**Overview & inputs/outputs**
- Executes hook scripts defined in config (pre-add, post-add, pre-dismiss, etc.)
- Passes notification data as environment variables.
- Captures stdout/stderr, exit codes.

**Public interfaces**
```go
type HookManager struct { ... }
func NewHookManager(config *config.Config) *HookManager
func (h *HookManager) Execute(event string, data map[string]string) error
```

**Errors/logging approach**
- Hook non-zero exit codes logged as warnings; do not abort operation unless configured.
- Errors returned only for fatal failures (hook not found, permission denied).

**Testing notes**
- Unit tests with mock scripts.
- Integration tests with real hook scripts.

### internal/tmux

**Overview & inputs/outputs**
- Low-level tmux client interaction via `tmux` command execution.
- Parses tmux display -p output.
- Sets tmux options.

**Public interfaces**
```go
type Client struct { ... }
func NewClient() (*Client, error)
func (c *Client) HasSession() bool
func (c *Client) Display(format string) (string, error)
func (c *Client) SetOption(option, value string) error
func (c *Client) SelectPane(session, window, pane string) error
```

**Errors/logging approach**
- Errors returned when tmux command fails.
- Log debug for tmux calls.

**Testing notes**
- Unit tests with mocked command execution.
- Integration tests require tmux server.

### internal/core

**Overview & inputs/outputs**
- High-level tray operations using storage and tmux.
- Retrieves current tmux context (session, window, pane, pane creation time).
- Validates pane existence.
- Jumps to pane (with fallback to window).
- Adds tray items with automatic context detection.
- Gets visibility state from tmux environment variable.

**Public interfaces**
```go
type Core struct { ... }
func NewCore(storage *storage.Storage, tmux *tmux.Client, config *config.Config) (*Core, error)
func (c *Core) GetTmuxContext() (session, window, pane, paneCreated string, err error)
func (c *Core) ValidatePane(session, window, pane string) bool
func (c *Core) JumpToPane(session, window, pane string) error
func (c *Core) AddItem(msg string, opts AddItemOptions) (id string, error)
func (c *Core) GetItems(stateFilter string) ([]string, error)
func (c *Core) ClearItems() error
func (c *Core) GetVisibility() (bool, error)
func (c *Core) SetVisibility(visible bool) error
```

**Errors/logging approach**
- Errors propagated from dependencies.
- Log info for visibility changes, jump actions.

**Testing notes**
- Unit tests with mocked storage and tmux.
- Integration tests with real tmux (optional).

### internal/log

**Overview & inputs/outputs**
- Provides colored output to terminal (stderr/stdout).
- Supports log levels (error, warning, info, success).
- Optional structured logging for future.

**Public interfaces**
```go
func Error(format string, args ...interface{})
func Warning(format string, args ...interface{})
func Info(format string, args ...interface{})
func Success(format string, args ...interface{})
func SetLevel(level LogLevel)
func SetOutput(w io.Writer)
```

**Errors/logging approach**
- No errors returned (writes to stderr).
- Color detection based on terminal capabilities.

**Testing notes**
- Unit tests with captured output.
- Color disabled in CI.

### internal/tsv

**Overview & inputs/outputs**
- TSV encoding/decoding with tab and newline escaping.
- Used by storage package.

**Public interfaces**
```go
func EncodeRecord(fields []string) string
func DecodeRecord(line string) ([]string, error)
func EscapeField(s string) string
func UnescapeField(s string) string
```

**Errors/logging approach**
- Errors on malformed escaping (unlikely).

**Testing notes**
- Unit tests for edge cases (empty fields, tabs, newlines).
- Golden compatibility with Bash implementation.

### cmd/tmux-intray

**Overview & inputs/outputs**
- Main CLI entry point parsing subcommands.
- Routes to appropriate command implementations.
- Sets up configuration, logging, and core dependencies.

**Public interfaces**
```go
func main()
func runShow(cmd *cobra.Command, args []string)
func runStatus(cmd *cobra.Command, args []string)
func runToggle(cmd *cobra.Command, args []string)
func runDismiss(cmd *cobra.Command, args []string)
func runHelp(cmd *cobra.Command, args []string)
```

**Errors/logging approach**
- Command errors printed via internal/log.
- Exit codes match Bash script.

**Testing notes**
- Integration tests for each subcommand.
- Shadow testing alongside Bash version.

## Cross-cutting Concerns

**Logging**
- Use internal/log for all user-facing messages.
- Debug logging only when `TMUX_INTRAY_DEBUG=1`.

**Config loading order**
1. Default values (hardcoded)
2. Config file values (`$TMUX_INTRAY_CONFIG_DIR/config.sh`)
3. Environment variables (`TMUX_INTRAY_*`)
Later overrides earlier.

**TSV storage compatibility rules**
- File format: 9 fields separated by tabs.
- Field order: ID, timestamp, state, session, window, pane, message, pane_created, level.
- Timestamp format: RFC3339 UTC (`2006-01-02T15:04:05Z`).
- Message escaping: backslashes → `\\`, tabs → `\t`, newlines → `\n`.
- Level defaults to "info" if missing.

**Hook execution rules**
- Hooks are optional scripts in `$TMUX_INTRAY_CONFIG_DIR/hooks/`.
- Execute synchronously (blocking) to maintain ordering.
- Exit code 0 = success; non-zero logs warning but continues (configurable).
- Environment variables: `TMUX_INTRAY_EVENT`, `TMUX_INTRAY_ID`, `TMUX_INTRAY_MESSAGE`, etc.

**Color handling**
- Respect `NO_COLOR` environment variable.
- Auto-disable colors when output is not a terminal.
- Use same color codes as Bash (red, green, yellow, blue, magenta, cyan, white).

## Migration/Implementation Order

**Phase 1 – Foundation (Small)**
- internal/log (1-2 days)
- internal/config (2-3 days)
- internal/tsv (1 day)

**Phase 2 – Storage (Medium)**
- internal/tmux (2 days)
- internal/storage (3-4 days)

**Phase 3 – Core (Medium)**
- internal/core (3-4 days)

**Phase 4 – Commands (Large)**
- cmd/tmux-intray with subcommands (5-7 days)

**Phase 5 – Hooks & Integration (Medium)**
- internal/hooks (2-3 days)
- Integration testing (2 days)

**Phase 6 – Polish & Switchover (Small)**
- Performance benchmarks, final compatibility tests, switchover (2-3 days)

Total estimated effort: ~20-30 days.

## Open Questions/Risks

1. **Hooks design**: Should hooks be synchronous? Bash scripts may expect immediate execution. Risk: blocking UI.
2. **Tmux command compatibility**: Different tmux versions may have different output formats. Need to test across versions.
3. **TSV parsing performance**: Large files (1000+ lines) need efficient parsing; Bash uses awk/tail. Go implementation must be similarly fast.
4. **Locking strategy**: Directory-based locking may not be portable on all filesystems; consider `flock` syscall.
5. **Backward compatibility**: Must ensure existing notifications remain readable after migration; thorough testing needed.
6. **Configuration file format**: Currently a shell script sourced; parsing shell assignments is non-trivial. May need to maintain shell compatibility or migrate to TOML/YAML (breaking change).
7. **Dependency injection**: Should we use interfaces for tmux and storage to ease testing? Yes.
8. **Error handling**: Need to decide whether to log and continue or abort for non-critical errors.

## Assumptions

- Go 1.24+ is available on target systems.
- Tmux is installed and accessible in PATH.
- Existing Bash scripts will be replaced gradually; dual-runner phase will allow side-by-side comparison.
- Users have XDG directories or fallback to `~/.local/state` and `~/.config`.
