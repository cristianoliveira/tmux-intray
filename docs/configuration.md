# tmux-intray Configuration

> **Note**: This project follows a minimalist, Unix‑style philosophy. See [Project Philosophy](./philosophy.md) for design principles and rationale.

tmux-intray can be configured via environment variables, which can be set in your shell profile or in a dedicated configuration file. This guide covers all available configuration options.

## Configuration File

The primary way to configure tmux-intray is through the configuration file at `$TMUX_INTRAY_CONFIG_DIR/config.toml` (default: `~/.config/tmux-intray/config.toml`). This file is parsed as a TOML file when tmux-intray starts, allowing you to set configuration options and define custom behavior.

If the file doesn't exist, tmux-intray creates a sample configuration file with default values and helpful comments.

## Environment Variables

All configuration options are controlled by environment variables with the `TMUX_INTRAY_` prefix. You can set these variables in your configuration file or export them in your shell.

### Storage & Paths

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_STATE_DIR` | `$XDG_STATE_HOME/tmux-intray` (`~/.local/state/tmux-intray`) | Directory where notification data is stored. Follows XDG Base Directory Specification. |
| `TMUX_INTRAY_CONFIG_DIR` | `$XDG_CONFIG_HOME/tmux-intray` (`~/.config/tmux-intray`) | Directory for configuration files and hooks. |
| `TMUX_INTRAY_STORAGE_BACKEND` | `tsv` | Storage backend: `tsv`, `sqlite`, or `dual` (TSV primary + SQLite secondary). |
| `TMUX_INTRAY_DUAL_READ_BACKEND` | `sqlite` | When `storage_backend=dual`, selects read backend (`sqlite` or `tsv`). |
| `TMUX_INTRAY_DUAL_VERIFY_ONLY` | `0` | When `storage_backend=dual`, write only to TSV (skip SQLite writes) for verification mode. |
| `TMUX_INTRAY_DUAL_VERIFY_SAMPLE_SIZE` | `25` | Number of records sampled during consistency verification. |
| `TMUX_INTRAY_MAX_NOTIFICATIONS` | `1000` | Maximum number of notifications to keep (oldest are automatically cleaned up). |
| `TMUX_INTRAY_AUTO_CLEANUP_DAYS` | `30` | Automatically clean up notifications that have been dismissed for more than this many days. |

### Display & Formatting

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_DATE_FORMAT` | `%Y-%m-%d %H:%M:%S` | Date format used in output (see `man date` for format codes). |
| `TMUX_INTRAY_TABLE_FORMAT` | `default` | Table style for `--format=table` output (`default`, `minimal`, `fancy`). |
| `TMUX_INTRAY_LEVEL_COLORS` | `info:green,warning:yellow,error:red,critical:magenta` | Color mapping for notification levels in status bar. Available colors: black, red, green, yellow, blue, magenta, cyan, white. |

### Status Bar Integration

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_STATUS_ENABLED` | `1` | Enable (1) or disable (0) the status bar indicator. |
| `TMUX_INTRAY_STATUS_FORMAT` | `compact` | Status panel format (`compact`, `detailed`, `count-only`). |
| `TMUX_INTRAY_SHOW_LEVELS` | `0` | Show level counts (1) or only total count (0) in status bar. |

### Hook System

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_HOOKS_ENABLED` | `1` | Enable (1) or disable (0) hooks globally. |
| `TMUX_INTRAY_HOOKS_FAILURE_MODE` | `warn` | Behavior when a hook fails: `ignore` (silently continue), `warn` (log warning), `abort` (stop operation). |
| `TMUX_INTRAY_HOOKS_ASYNC` | `0` | Run hooks asynchronously (1) or synchronously (0). *Not yet implemented*. |
| `TMUX_INTRAY_HOOKS_DIR` | `$TMUX_INTRAY_CONFIG_DIR/hooks` | Directory containing hook scripts. |
| `TMUX_INTRAY_HOOKS_ENABLED_pre_add` | `1` | Enable/disable pre‑add hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_post_add` | `1` | Enable/disable post‑add hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_pre_dismiss` | `1` | Enable/disable pre‑dismiss hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_post_dismiss` | `1` | Enable/disable post‑dismiss hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_cleanup` | `1` | Enable/disable cleanup hooks (0/1). |
| `TMUX_INTRAY_HOOKS_ENABLED_post_cleanup` | `1` | Enable/disable post‑cleanup hooks (0/1). |

### Debugging & Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_DEBUG` | *unset* | Enable debug output when set to `1`, `true`, `yes`, or `on`. Debug messages are printed to stderr. |
| `TMUX_INTRAY_QUIET` | *unset* | Suppress non‑error output when set to `1`, `true`, `yes`, or `on`. |

## Sample Configuration File

```toml
# tmux-intray configuration
# This file is parsed as TOML by tmux-intray on startup.

# Storage directories (follow XDG Base Directory Specification)
state_dir = "~/.local/state/tmux-intray"
config_dir = "~/.config/tmux-intray"
storage_backend = "tsv"

# Dual-write mode (used when storage_backend="dual")
dual_read_backend = "sqlite"
dual_verify_only = false
dual_verify_sample_size = 25

# Storage limits
max_notifications = 1000
auto_cleanup_days = 30

# Display settings
date_format = "%Y-%m-%d %H:%M:%S"
table_format = "default"

# Status bar integration
status_enabled = true
status_format = "compact"
show_levels = false
level_colors = "info:green,warning:yellow,error:red,critical:magenta"

# Hook system
hooks_enabled = true
hooks_failure_mode = "warn"
hooks_async = false
hooks_dir = "~/.config/tmux-intray/hooks"

# Per-hook enable/disable
hooks_enabled_pre_add = true
hooks_enabled_post_add = true
hooks_enabled_pre_dismiss = true
hooks_enabled_post_dismiss = true
hooks_enabled_cleanup = true
hooks_enabled_post_cleanup = true

# Debugging
debug = false
quiet = false
```

## Overriding Configuration

You can also set environment variables directly in your shell, which take precedence over the configuration file. For example:

```bash
export TMUX_INTRAY_DEBUG=1
export TMUX_INTRAY_AUTO_CLEANUP_DAYS=7
tmux-intray list
```

This is useful for temporary debugging or for per‑session customization.

## Dual-Write Mode

Set `TMUX_INTRAY_STORAGE_BACKEND=dual` to enable migration-safe dual writes:

- All mutating operations write to TSV first, then SQLite.
- TSV write failures stop the command.
- SQLite write failures are logged as warnings, and execution continues with TSV as source of truth.
- Reads use `TMUX_INTRAY_DUAL_READ_BACKEND` (`sqlite` by default for performance checks).
- `TMUX_INTRAY_DUAL_VERIFY_ONLY=1` keeps writes TSV-only while allowing consistency verification tooling.

## SQLite Opt-in Rollout (Beta)

> [!WARNING]
> SQLite is opt-in during rollout. `TMUX_INTRAY_STORAGE_BACKEND=tsv` remains the default.

Use these backend values during rollout:

- `tsv` (default): Stable baseline.
- `dual`: Recommended first opt-in step. Writes to TSV and SQLite while preserving TSV as source of truth.
- `sqlite`: Full SQLite mode after successful validation.

### Recommended staged rollout

1. Keep default `tsv` for unchanged behavior.
2. Opt in with `dual` for verification:
   ```bash
   TMUX_INTRAY_STORAGE_BACKEND="dual"
   TMUX_INTRAY_DUAL_READ_BACKEND="sqlite"
   TMUX_INTRAY_DUAL_VERIFY_ONLY=0
   ```
3. Promote to full SQLite:
   ```bash
   TMUX_INTRAY_STORAGE_BACKEND="sqlite"
   ```
4. Roll back immediately if needed:
   ```bash
   TMUX_INTRAY_STORAGE_BACKEND="tsv"
   ```

### Safeguards

- Unknown backend values fall back to TSV.
- If SQLite or dual backend initialization fails, tmux-intray warns and falls back to TSV.
- In dual mode, SQLite write failures are non-fatal; TSV continues as the source of truth.

### sqlc-backed query layer notes

SQLite queries are defined in `internal/storage/sqlite/queries.sql` and generated with sqlc into `internal/storage/sqlite/sqlcgen/`.

When changing SQLite schema or query files, regenerate and verify generated code:

```bash
make sqlc-generate
make sqlc-check
```

## TUI Settings Persistence

The TUI (Terminal User Interface) automatically saves your preferences when you exit. These settings include column order, sort preferences, active filters, view mode, and grouping preferences.

### Settings File Location

Settings are stored at `~/.config/tmux-intray/settings.toml` (or `$XDG_CONFIG_HOME/tmux-intray/settings.toml` if XDG_CONFIG_HOME is set).

### Available Settings

The settings file uses the following TOML schema:

```toml
columns = ["id", "timestamp", "state", "level", "session", "window", "pane", "message"]
sortBy = "timestamp"
sortOrder = "desc"

[filters]
level = ""
state = ""
session = ""
window = ""
pane = ""

viewMode = "grouped"
groupBy = "none"
defaultExpandLevel = 1
expansionState = {}
```

#### Settings Fields

| Field | Type | Description | Default | Valid Values |
|-------|------|-------------|---------|--------------|
| `columns` | array | Column display order | All columns in default order | `["id", "timestamp", "state", "level", "session", "window", "pane", "message", "pane_created"]` |
| `sortBy` | string | Column to sort by | `"timestamp"` | `"id"`, `"timestamp"`, `"state"`, `"level"`, `"session"` |
| `sortOrder` | string | Sort direction | `"desc"` | `"asc"`, `"desc"` |
| `filters.level` | string | Filter by severity level | `""` (no filter) | `"info"`, `"warning"`, `"error"`, `"critical"`, `""` |
| `filters.state` | string | Filter by state | `""` (no filter) | `"active"`, `"dismissed"`, `""` |
| `filters.session` | string | Filter by tmux session | `""` (no filter) | Session name or `""` |
| `filters.window` | string | Filter by tmux window | `""` (no filter) | Window ID or `""` |
| `filters.pane` | string | Filter by tmux pane | `""` (no filter) | Pane ID or `""` |
| `viewMode` | string | Display layout | `"grouped"` | `"compact"`, `"detailed"`, `"grouped"` |
| `groupBy` | string | Group notifications in the TUI | `"none"` | `"none"`, `"session"`, `"window"`, `"pane"` |
| `defaultExpandLevel` | number | Default grouping expansion depth | `1` | `0`-`3` |
| `expansionState` | object | Explicit expansion overrides by node path | `{}` | Object of string to boolean |

`groupBy` controls the depth of grouped view hierarchy:

- `none`: no group rows; notifications are listed directly in grouped view
- `session`: session groups with notifications directly under each session
- `window`: session -> window -> notification
- `pane`: session -> window -> pane -> notification

### Default Settings

If the settings file doesn't exist or is corrupted, the TUI uses these defaults:

```toml
columns = ["id", "timestamp", "state", "level", "session", "window", "pane", "message"]
sortBy = "timestamp"
sortOrder = "desc"

[filters]
level = ""
state = ""
session = ""
window = ""
pane = ""

viewMode = "grouped"
groupBy = "none"
defaultExpandLevel = 1
expansionState = {}
```

### How Settings Are Saved

Settings are saved automatically in these situations:

1. **On TUI exit** (pressing `q`, `:q`, or `Ctrl+C`)
2. **Manual save** (pressing `:w` in command mode)

The save operation uses atomic writes to prevent file corruption.

### Managing Settings

#### View Current Settings

Display your current settings in TOML format:

```bash
tmux-intray settings show
```

#### Reset Settings to Defaults

Reset all TUI settings to their default values:

```bash
# Reset with confirmation prompt
tmux-intray settings reset

# Reset without confirmation (use with caution)
tmux-intray settings reset --force
```

This command deletes the `settings.toml` file. The TUI will use defaults on the next launch.

#### Manually Edit Settings

You can edit the settings file directly with any text editor:

```bash
# Open settings file
vim ~/.config/tmux-intray/settings.toml
```

After editing, the TUI will load the new settings on the next launch.

### Example Settings

Here are some example settings configurations:

**Show only active errors, sorted by timestamp ascending:**

```toml
columns = ["level", "message", "session", "timestamp"]
sortBy = "timestamp"
sortOrder = "asc"

[filters]
level = "error"
state = "active"
session = ""
window = ""
pane = ""

viewMode = "compact"
groupBy = "none"
defaultExpandLevel = 1
expansionState = {}
```

**Show warnings and critical messages from specific session:**

```toml
columns = ["level", "message", "window", "pane", "timestamp"]
sortBy = "level"
sortOrder = "desc"

[filters]
level = "critical"
state = ""
session = "work"
window = ""
pane = ""

viewMode = "detailed"
groupBy = "session"
defaultExpandLevel = 2

[expansionState]
"session:work" = true
```

### Error Handling

If the settings file is corrupted (invalid TOML), the TUI will:
1. Log a warning message to stderr
2. Fall back to default settings
3. Continue operating normally

This ensures that a corrupted settings file doesn't prevent the TUI from running.

### Notes

- Settings are stored in TOML format with 2-space indentation for readability
- The settings directory (`~/.config/tmux-intray`) is created automatically if it doesn't exist
- File locking is used to prevent concurrent writes when multiple TUI instances are running
- Empty string values for filters mean "no filter" (show all)
- Empty or missing `columns` array uses the default column order
- For XDG Base Directory compliance, the file location is `$XDG_CONFIG_HOME/tmux-intray/settings.toml`
