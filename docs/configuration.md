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
| `TMUX_INTRAY_TUI_SETTINGS_PATH` | *unset* (defaults to `$TMUX_INTRAY_CONFIG_DIR/tui.toml`) | Optional override for the TUI settings file location. |
| `TMUX_INTRAY_STORAGE_BACKEND` | `sqlite` | Storage backend (only `sqlite` is supported). |
| `TMUX_INTRAY_MAX_NOTIFICATIONS` | `1000` | Maximum number of notifications to keep (oldest are automatically cleaned up). |
| `TMUX_INTRAY_AUTO_CLEANUP_DAYS` | `30` | Automatically clean up notifications that have been dismissed for more than this many days. |

### Display & Formatting

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_DATE_FORMAT` | `%Y-%m-%d %H:%M:%S` | Date format used in output (see `man date` for format codes). |
| `TMUX_INTRAY_TABLE_FORMAT` | `default` | Table style for `--format=table` output (`default`, `minimal`, `fancy`). |
| `TMUX_INTRAY_LEVEL_COLORS` | `info:green,warning:yellow,error:red,critical:magenta` | Color mapping for notification levels in status bar. Available colors: black, red, green, yellow, blue, magenta, cyan, white. |

### Deduplication

`tmux-intray` can collapse identical notifications when you enable `group_by = "message"` in the TUI or run `tmux-intray list --group-by=message`. Use the `[dedup]` section in `config.toml` (or matching environment variables) to refine how duplicates are detected.

| Key | Env Var | Default | Description |
|-----|---------|---------|-------------|
| `dedup.criteria` | `TMUX_INTRAY_DEDUP__CRITERIA` | `"message"` | Fields used to determine duplicates. Allowed values: `"message"`, `"message_level"`, `"message_source"`, `"exact"`. `message_level` requires both message text and severity to match, `message_source` also includes session/window/pane, and `exact` matches message + level + tmux source + state. |
| `dedup.window` | `TMUX_INTRAY_DEDUP__WINDOW` | *(empty)* | Optional Go-style duration (e.g., `"30s"`, `"5m"`) that limits deduplication to events occurring within the specified time window. Leave empty to combine all matching notifications regardless of age. |

Environment variables that refer to dotted keys use double underscores (`__`) to separate segments. For example, set `TMUX_INTRAY_DEDUP__CRITERIA=message_source` to override `dedup.criteria`.

```toml
[dedup]
criteria = "message_source"
window = "5m"
```

With the configuration above, `tmux-intray` only collapses notifications when they share the same message text and tmux source (session/window/pane) and were emitted within five minutes of each other.

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
| `TMUX_INTRAY_LOGGING_ENABLED` | `false` | Enable structured file logging when set to `true`. See [Structured Logging](./logging.md) for details. |
| `TMUX_INTRAY_LOGGING_LEVEL` | `info` | Minimum log level to record (`debug`, `info`, `warn`, `error`). |
| `TMUX_INTRAY_LOGGING_MAX_FILES` | `10` | Maximum number of log files to retain (older files are rotated out). |

## Sample Configuration File

```toml
# tmux-intray configuration
# This file is parsed as TOML by tmux-intray on startup.

# Storage directories (follow XDG Base Directory Specification)
state_dir = "~/.local/state/tmux-intray"
config_dir = "~/.config/tmux-intray"
storage_backend = "sqlite"

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

# Logging
logging_enabled = false
logging_level = "info"
logging_max_files = 10

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

## SQLite Storage

tmux-intray uses SQLite as its storage backend. Data is stored in `$TMUX_INTRAY_STATE_DIR/notifications.db`.

### sqlc-backed query layer

SQLite queries are defined in `internal/storage/sqlite/queries.sql` and generated with sqlc into `internal/storage/sqlite/sqlcgen/`.

When changing SQLite schema or query files, regenerate and verify generated code:

```bash
make sqlc-generate
make sqlc-check
```

## TUI Settings Persistence

The TUI (Terminal User Interface) automatically saves your preferences when you exit. These settings include column order, sort preferences, active filters, view mode, and grouping preferences.

### Settings File Location

Settings are stored at `~/.config/tmux-intray/tui.toml` (or `$XDG_CONFIG_HOME/tmux-intray/tui.toml` if XDG_CONFIG_HOME is set). Older releases used `settings.toml`; tmux-intray automatically migrates that legacy file to `tui.toml` on startup.

To move the file elsewhere, set `TMUX_INTRAY_TUI_SETTINGS_PATH` (or the `tui_settings_path` key in `config.toml`).

### Available Settings

The settings file uses the following TOML schema:

```toml
columns = ["id", "timestamp", "state", "level", "session", "window", "pane", "message"]
sortBy = "timestamp"
sortOrder = "desc"
unreadFirst = true

[filters]
level = ""
state = ""
read = ""
session = ""
window = ""
pane = ""

viewMode = "grouped"
groupBy = "none"
defaultExpandLevel = 1
expansionState = {}

[groupHeader]
showTimeRange = true
showLevelBadges = true
showSourceAggregation = false

[groupHeader.badgeColors]
info = "\u001b[0;34m"
warning = "\u001b[1;33m"
error = "\u001b[0;31m"
critical = "\u001b[0;31m"
```

#### Settings Fields

| Field | Type | Description | Default | Valid Values |
|-------|------|-------------|---------|--------------|
| `columns` | array | Column display order | All columns in default order | `["id", "timestamp", "state", "level", "session", "window", "pane", "message", "pane_created"]` |
| `sortBy` | string | Column to sort by | `"timestamp"` | `"id"`, `"timestamp"`, `"state"`, `"level"`, `"session"` |
| `sortOrder` | string | Sort direction | `"desc"` | `"asc"`, `"desc"` |
| `unreadFirst` | bool | Group unread notifications first before applying sort | `true` | `true`, `false` |
| `filters.level` | string | Filter by severity level | `""` (no filter) | `"info"`, `"warning"`, `"error"`, `"critical"`, `""` |
| `filters.state` | string | Filter by state | `""` (no filter) | `"active"`, `"dismissed"`, `""` |
| `filters.read` | string | Filter by read/unread status | `""` (all notifications) | `"read"`, `"unread"`, `""` |
| `filters.session` | string | Filter by tmux session | `""` (no filter) | Session name or `""` |
| `filters.window` | string | Filter by tmux window | `""` (no filter) | Window ID or `""` |
| `filters.pane` | string | Filter by tmux pane | `""` (no filter) | Pane ID or `""` |
| `viewMode` | string | Display layout | `"grouped"` | `"compact"`, `"detailed"`, `"grouped"`, `"search"` |
| `groupBy` | string | Group notifications in the TUI | `"none"` | `"none"`, `"session"`, `"window"`, `"pane"`, `"message"`, `"pane_message"` |
| `defaultExpandLevel` | number | Default grouping expansion depth | `1` | `0`-`3` |
| `expansionState` | object | Explicit expansion overrides by node path | `{}` | Object of string to boolean |
| `groupHeader.showTimeRange` | bool | Show earliest/latest ages in group headers | `true` | `true`, `false` |
| `groupHeader.showLevelBadges` | bool | Show per-level counts as badges | `true` | `true`, `false` |
| `groupHeader.showSourceAggregation` | bool | Show aggregated pane/source info | `false` | `true`, `false` |
| `groupHeader.badgeColors` | table | ANSI color codes per level (`info`, `warning`, `error`, `critical`) | defaults shown above | Strings containing ANSI escape sequences |

`filters.read` lets you persist whether the TUI should show only read, only unread, or all notifications. At runtime you can toggle the same preference with the `:filter-read <read|unread|all>` command; the change is saved back to `tui.toml` automatically.

#### Sorting with Unread-First Grouping

`unreadFirst` controls whether unread notifications are visually separated and displayed first, regardless of the sort order:

- **`true` (default)**: Unread notifications appear first as a group, followed by read notifications. Within each group, notifications are sorted by `sortBy` and `sortOrder`. This is useful for keeping attention on unread items while maintaining a predictable sort order within each read status group.
- **`false`**: All notifications are sorted without grouping by read status, using only the `sortBy` and `sortOrder` settings.

**Example with `unreadFirst = true`** (sort by timestamp descending):
```
[Unread notifications - newest first]
- 14:35 - Error: Connection timeout (unread)
- 14:20 - Warning: High memory usage (unread)

[Read notifications - newest first]
- 13:50 - Info: Backup completed (read)
- 13:10 - Info: Job finished (read)
```

**Example with `unreadFirst = false`** (sort by timestamp descending):
```
- 14:35 - Error: Connection timeout (unread)
- 14:20 - Warning: High memory usage (unread)
- 13:50 - Info: Backup completed (read)
- 13:10 - Info: Job finished (read)
```

To disable unread-first grouping, add this to your `~/.config/tmux-intray/tui.toml`:

```toml
unreadFirst = false
```

Or using the environment variable:

```bash
export TMUX_INTRAY_TUI_UNREAD_FIRST=false
```

`groupBy` controls the depth of grouped view hierarchy:

- `none`: no group rows; notifications are listed directly in grouped view
- `session`: session groups with notifications directly under each session
- `window`: session -> window -> notification
- `pane`: session -> window -> pane -> notification
- `message`: groups notifications by message text (exact match)
- `pane_message`: session -> window -> pane -> message groups (one row per unique message per pane)

#### Message-Based Grouping

Set `groupBy = "message"` (the same value referred to as `group_by = "message"` in issue #223) to collapse identical notifications into a single group headed by the message text. Use `groupBy = "pane_message"` if you want one unique message per pane (no duplicate rows under the group). You can do this in two ways:

1. **Edit the settings file** – add `groupBy = "message"` to `~/.config/tmux-intray/settings.toml` (or your `$XDG_CONFIG_HOME` variant) alongside your other preferences:

    ```toml
    viewMode = "grouped"
    groupBy = "message"
    defaultExpandLevel = 1
    ```

2. **Use the TUI command palette** – run `:group-by message` inside the TUI; the change is saved automatically on exit or when you run `:w`.

Message grouping keeps all other filters intact and works with CLI commands such as `tmux-intray list --group-by=message` and `tmux-intray list --group-by=message --group-count`.

### Default Settings

If the settings file doesn't exist or is corrupted, the TUI uses these defaults:

```toml
columns = ["id", "timestamp", "state", "level", "session", "window", "pane", "message"]
sortBy = "timestamp"
sortOrder = "desc"
unreadFirst = true

[filters]
level = ""
state = ""
read = ""
session = ""
window = ""
pane = ""

viewMode = "grouped"
groupBy = "none"
defaultExpandLevel = 1
expansionState = {}

[groupHeader]
showTimeRange = true
showLevelBadges = true
showSourceAggregation = false

[groupHeader.badgeColors]
info = "\u001b[0;34m"
warning = "\u001b[1;33m"
error = "\u001b[0;31m"
critical = "\u001b[0;31m"
```

### How Settings Are Saved

Settings are saved automatically in these situations:

1. **On TUI exit** (pressing `q`, `:q`, or `Ctrl+C`)
2. **Manual save** (pressing `:w`)

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

This command deletes the `tui.toml` file. The TUI will use defaults on the next launch.

#### Manually Edit Settings

You can edit the settings file directly with any text editor:

```bash
# Open settings file
vim ~/.config/tmux-intray/tui.toml
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
read = "unread"
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
read = ""
session = "work"
window = ""
pane = ""

viewMode = "detailed"
groupBy = "session"
defaultExpandLevel = 2

[expansionState]
"session:work" = true

**Disable unread-first grouping, sort all by timestamp descending:**

```toml
columns = ["timestamp", "level", "message", "session"]
sortBy = "timestamp"
sortOrder = "desc"
unreadFirst = false

[filters]
level = ""
state = ""
read = ""
session = ""
window = ""
pane = ""

viewMode = "grouped"
groupBy = "none"
defaultExpandLevel = 1
expansionState = {}
```

**Customize group headers:**

```toml
[groupHeader]
showTimeRange = false
showLevelBadges = false
showSourceAggregation = true

[groupHeader.badgeColors]
info = "\u001b[0;36m"
warning = "\u001b[1;33m"
error = "\u001b[0;31m"
critical = "\u001b[0;31m"
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
- Use the TUI to change `filters.read` on the fly without editing the file
- Empty or missing `columns` array uses the default column order
- For XDG Base Directory compliance, the file location is `$XDG_CONFIG_HOME/tmux-intray/tui.toml`
