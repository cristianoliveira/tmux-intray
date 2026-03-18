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
| `TMUX_INTRAY_AUTO_CLEANUP_DAYS` | `30` | Automatically clean up notifications that have been dismissed for more than this many days. |

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



### Recents Tab

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_RECENTS_TIME_WINDOW` | `1h` | Time window for the Recents tab. Only notifications from within this time window are shown. Valid values: `5m`, `15m`, `30m`, `1h`, `2h`, `6h`, `12h`, `24h`. |

**Behavior:**

- The Recents tab automatically filters notifications to show only those within the configured time window
- The time window is measured from the current time (when the app is running)
- Changes to this config require restarting tmux-intray
- If an invalid value is provided, the app will warn and use the default (`1h`)

**Examples:**

```bash
# Show only notifications from the last 30 minutes
export TMUX_INTRAY_RECENTS_TIME_WINDOW=30m

# Or in config.toml
recents_time_window = "30m"

# Show notifications from the last 6 hours
export TMUX_INTRAY_RECENTS_TIME_WINDOW=6h
```

### Hook System

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_HOOKS_DIR` | `$TMUX_INTRAY_CONFIG_DIR/hooks` | Directory containing hook scripts. |

### Debugging & Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_LOG_LEVEL` | `info` | Console logging level: `debug`, `info`, `warn`, `error`, or `off`. See [Debugging Guide](./debugging.md) for details. |
| `TMUX_INTRAY_LOGGING_ENABLED` | `false` | Enable structured JSON logging to file. When enabled, logs are written to `$TMUX_INTRAY_STATE_DIR/logs/tmux-intray_YYYY-MM-DDTHH-MM-SS_PID{pid}_{command}.log`. |
| `TMUX_INTRAY_LOGGING_LEVEL` | `info` | File logging level: `debug`, `info`, `warn`, or `error`. This controls the level of detail in structured log files. |
| `TMUX_INTRAY_LOGGING_MAX_FILES` | `10` | Maximum number of log files to keep. Older log files are automatically deleted on startup. |
| `TMUX_INTRAY_LOG_FILE` | *(empty)* | Explicit log file path. If set, overrides the default log file location. Useful for temporary debugging or directing logs to a specific location. |

**Structured Logging:**

tmux-intray supports structured JSON logging that provides detailed, machine-readable logs for debugging and analysis. When enabled:

- Logs are written to a per-run file named `tmux-intray_YYYY-MM-DDTHH-MM-SS_PID{pid}_{command}.log`
- The log file location is `$TMUX_INTRAY_STATE_DIR/logs` (default: `~/.local/state/tmux-intray/logs`)
- Old log files are automatically rotated to keep only the most recent `TMUX_INTRAY_LOGGING_MAX_FILES` files
- Sensitive data (passwords, tokens, keys, etc.) is automatically redacted from logs
- The log file path is printed to stdout on startup when logging is enabled

**Example usage:**

```bash
# Enable structured logging to the default location
export TMUX_INTRAY_LOGGING_ENABLED=true
export TMUX_INTRAY_LOGGING_LEVEL=debug
tmux-intray list

# Specify a custom log file
export TMUX_INTRAY_LOGGING_ENABLED=true
export TMUX_INTRAY_LOG_FILE=/tmp/tmux-intray-debug.log
tmux-intray add "test notification"

# Or use the --log-file flag
tmux-intray --log-file=/tmp/debug.log list
```

**Configuration file example:**

```toml
# Enable structured logging
logging_enabled = true

# Set logging level
logging_level = "info"

# Keep at most 20 log files
logging_max_files = 20

# Optional: explicit log file path
# log_file = "/var/log/tmux-intray/debug.log"
```

For comprehensive debugging guidance including troubleshooting scenarios and examples, see the **[Debugging Guide](./debugging.md)**.

## Sample Configuration File

```toml
# tmux-intray configuration
# This file is parsed as TOML by tmux-intray on startup.

# Storage directories (follow XDG Base Directory Specification)
state_dir = "~/.local/state/tmux-intray"
config_dir = "~/.config/tmux-intray"
storage_backend = "sqlite"

# Storage limits
auto_cleanup_days = 30

# Hook system
hooks_dir = "~/.config/tmux-intray/hooks"

# Console logging (see docs/debugging.md for details)
# Options: debug, info, warn, error, off
# Default: info
log_level = "info"

# Structured JSON logging to file
# Options: true, false
# Default: false
logging_enabled = false

# File logging level: debug, info, warn, error
# Default: info
logging_level = "info"

# Maximum number of log files to keep
# Default: 10
logging_max_files = 10

# Optional: explicit log file path (overrides default location)
# log_file = "/var/log/tmux-intray/debug.log"

# Recents tab configuration
# Valid values: 5m, 15m, 30m, 1h, 2h, 6h, 12h, 24h
# Default: 1h
recents_time_window = "1h"
```

## Overriding Configuration

You can also set environment variables directly in your shell, which take precedence over the configuration file. For example:

```bash
# Enable debug logging
export TMUX_INTRAY_LOG_LEVEL=debug

# Adjust notification retention
export TMUX_INTRAY_AUTO_CLEANUP_DAYS=7

# Run command
tmux-intray list
```

For detailed debugging examples and troubleshooting, see the [Debugging Guide](./debugging.md).

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
sort_by = "timestamp"
sort_order = "desc"
unread_first = true

[filters]
level = ""
state = ""
read = ""
session = ""
window = ""
pane = ""

view_mode = "grouped"
group_by = "none"
default_expand_level = 1
expansion_state = {}

[group_header]
show_time_range = true
show_level_badges = true
show_source_aggregation = false

[group_header.badge_colors]
info = "\u001b[0;34m"
warning = "\u001b[1;33m"
error = "\u001b[0;31m"
critical = "\u001b[0;31m"
```

#### Settings Fields

| Field | Type | Description | Default | Valid Values |
|-------|------|-------------|---------|--------------|
| `columns` | array | Column display order | All columns in default order | `["id", "timestamp", "state", "level", "session", "window", "pane", "message", "pane_created"]` |
| `sort_by` | string | Column to sort by | `"timestamp"` | `"id"`, `"timestamp"`, `"state"`, `"level"`, `"session"` |
| `sort_order` | string | Sort direction | `"desc"` | `"asc"`, `"desc"` |
| `unread_first` | bool | Group unread notifications first before applying sort | `true` | `true`, `false` |
| `filters.level` | string | Filter by severity level | `""` (no filter) | `"info"`, `"warning"`, `"error"`, `"critical"`, `""` |
| `filters.state` | string | Filter by state | `""` (no filter) | `"active"`, `"dismissed"`, `""` |
| `filters.read` | string | Filter by read/unread status | `""` (all notifications) | `"read"`, `"unread"`, `""` |
| `filters.session` | string | Filter by tmux session | `""` (no filter) | Session name or `""` |
| `filters.window` | string | Filter by tmux window | `""` (no filter) | Window ID or `""` |
| `filters.pane` | string | Filter by tmux pane | `""` (no filter) | Pane ID or `""` |
| `view_mode` | string | Display layout | `"grouped"` | `"detailed"`, `"grouped"`, `"search"` (note: `compact` is deprecated for migration only) |
| `group_by` | string | Group notifications in the TUI | `"none"` | `"none"`, `"session"`, `"window"`, `"pane"`, `"message"`, `"pane_message"` |
| `default_expand_level` | number | Default grouping expansion depth | `1` | `0`-`3` |
| `expansion_state` | object | Explicit expansion overrides by node path | `{}` | Object of string to boolean |
| `group_header.show_time_range` | bool | Show earliest/latest ages in group headers | `true` | `true`, `false` |
| `group_header.show_level_badges` | bool | Show per-level counts as badges | `true` | `true`, `false` |
| `group_header.show_source_aggregation` | bool | Show aggregated pane/source info | `false` | `true`, `false` |
| `group_header.badge_colors` | table | ANSI color codes per level (`info`, `warning`, `error`, `critical`) | defaults shown above | Strings containing ANSI escape sequences |

`filters.read` lets you persist whether the TUI should show only read, only unread, or all notifications. At runtime you can toggle the same preference with the `:filter-read <read|unread|all>` command; the change is saved back to `tui.toml` automatically.

#### View Mode Migration

The `compact` view mode is **deprecated** but remains supported for migration purposes:

- Legacy configuration files with `view_mode = "compact"` are automatically migrated to `view_mode = "detailed"` on TUI startup
- The `compact` mode is no longer part of the active view mode cycle (`detailed → grouped → search → detailed`)
- Use `detailed` mode for full notification details in a single-line format

#### Sorting with Unread-First Grouping

`unread_first` controls whether unread notifications are visually separated and displayed first, regardless of the sort order:

- **`true` (default)**: Unread notifications appear first as a group, followed by read notifications. Within each group, notifications are sorted by `sort_by` and `sort_order`. This is useful for keeping attention on unread items while maintaining a predictable sort order within each read status group.
- **`false`**: All notifications are sorted without grouping by read status, using only the `sort_by` and `sort_order` settings.

**Example with `unread_first = true`** (sort by timestamp descending):
```
[Unread notifications - newest first]
- 14:35 - Error: Connection timeout (unread)
- 14:20 - Warning: High memory usage (unread)

[Read notifications - newest first]
- 13:50 - Info: Backup completed (read)
- 13:10 - Info: Job finished (read)
```

**Example with `unread_first = false`** (sort by timestamp descending):
```
- 14:35 - Error: Connection timeout (unread)
- 14:20 - Warning: High memory usage (unread)
- 13:50 - Info: Backup completed (read)
- 13:10 - Info: Job finished (read)
```

To disable unread-first grouping, add this to your `~/.config/tmux-intray/tui.toml`:

```toml
unread_first = false
```

Or using the environment variable:

```bash
export TMUX_INTRAY_TUI_UNREAD_FIRST=false
```

`group_by` controls the depth of grouped view hierarchy:

- `none`: no group rows; notifications are listed directly in grouped view
- `session`: session groups with notifications directly under each session
- `window`: session -> window -> notification
- `pane`: session -> window -> pane -> notification
- `message`: groups notifications by message text (exact match)
- `pane_message`: session -> window -> pane -> message groups (one row per unique message per pane)

#### Message-Based Grouping

Set `group_by = "message"` to collapse identical notifications into a single group headed by the message text. Use `group_by = "pane_message"` if you want one unique message per pane (no duplicate rows under the group). You can do this in two ways:

1. **Edit the settings file** – add `group_by = "message"` to `~/.config/tmux-intray/settings.toml` (or your `$XDG_CONFIG_HOME` variant) alongside your other preferences:

    ```toml
    view_mode = "grouped"
    group_by = "message"
    default_expand_level = 1
    ```

2. **Use the TUI command palette** – run `:group-by message` inside the TUI; the change is saved automatically on exit or when you run `:w`.

Message grouping keeps all other filters intact and works with CLI commands such as `tmux-intray list --group-by=message` and `tmux-intray list --group-by=message --group-count`.

### Default Settings

If the settings file doesn't exist or is corrupted, the TUI uses these defaults:

```toml
columns = ["id", "timestamp", "state", "level", "session", "window", "pane", "message"]
sort_by = "timestamp"
sort_order = "desc"
unread_first = true

[filters]
level = ""
state = ""
read = ""
session = ""
window = ""
pane = ""

view_mode = "grouped"
group_by = "none"
default_expand_level = 1
expansion_state = {}

[group_header]
show_time_range = true
show_level_badges = true
show_source_aggregation = false

[group_header.badge_colors]
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
sort_by = "timestamp"
sort_order = "asc"

[filters]
level = "error"
state = "active"
read = "unread"
session = ""
window = ""
pane = ""

view_mode = "detailed"
group_by = "none"
default_expand_level = 1
expansion_state = {}
```

**Show warnings and critical messages from specific session:**

```toml
columns = ["level", "message", "window", "pane", "timestamp"]
sort_by = "level"
sort_order = "desc"

[filters]
level = "critical"
state = ""
read = ""
session = "work"
window = ""
pane = ""

view_mode = "detailed"
group_by = "session"
default_expand_level = 2

[expansion_state]
"session:work" = true
```

**Disable unread-first grouping, sort all by timestamp descending:**

```toml
columns = ["timestamp", "level", "message", "session"]
sort_by = "timestamp"
sort_order = "desc"
unread_first = false

[filters]
level = ""
state = ""
read = ""
session = ""
window = ""
pane = ""

view_mode = "grouped"
group_by = "none"
default_expand_level = 1
expansion_state = {}
```

**Customize group headers:**

```toml
[group_header]
show_time_range = false
show_level_badges = false
show_source_aggregation = true

[group_header.badge_colors]
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
