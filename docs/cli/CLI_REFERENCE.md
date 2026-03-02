# tmux-intray CLI Reference

Version: development

## Overview

tmux-intray is a quiet inbox for things that happen while you're not looking.

## Global Usage

```
A quiet inbox for things that happen while you're not looking.

Usage:
  tmux-intray [flags]
  tmux-intray [command]

Available Commands:
  add          Add a new item to the tray
  cleanup      Clean up old dismissed notifications
  clear        Clear all items from the tray
  dismiss      Dismiss a notification
  follow       Monitor notifications in real-time
  help         Help about any command
  jump         Jump to the pane of a notification
  list         List notifications with filters and formats
  mark-read    Mark a notification as read
  settings     Manage TUI settings
  status       Show notification status summary
  tui          Interactive terminal UI for notifications
  version      Show version information

Flags:
  -h, --help               help for tmux-intray
  -v, --version            version for tmux-intray
      --log-file string    log file path (default empty, logs to stderr)

Use "tmux-intray [command] --help" for more information about a command.
```

## Commands

### list

```
tmux-intray list [flags]
```

Lists notifications with filter, grouping, and formatting flags. Common grouping flags:

- `--group-by <field>` – field can be `session`, `window`, `pane`, `level`, or `message`. The new `message` option collapses identical notification text so you can review duplicates once.
- `--group-count` – when paired with `--group-by`, only emit group headers and counts.

Use `--group-by=message --group-count` to summarize duplicate notifications quickly:

```
tmux-intray list --group-by=message --group-count
```

The CLI shares its grouping implementation with the TUI, so any value that works in one place (including `message`) works in the other.

### tui

```
tmux-intray tui
```

Launches the interactive notifications UI.

#### Keybindings

- `r` - switch to Recents tab
- `a` - switch to All tab
- `R` - mark selected notification as read
- `u` - mark selected notification as unread
- `Enter` - jump to selected notification target (pane when available, window fallback)

### status

```
tmux-intray status [flags]
```

Show notification status summary with template-based formatting. Supports 13 template variables and 6 built-in presets, plus custom templates for flexible output.

#### Presets (Built-in Templates)

| Preset | Template | Description |
|--------|----------|-------------|
| `compact` | `[%{unread-count}] %{latest-message}` | Count and latest message (default) |
| `detailed` | `%{unread-count} unread, %{read-count} read \| Latest: %{latest-message}` | Full breakdown with message |
| `json` | `{"unread":%{unread-count},"total":%{total-count},"message":"%{latest-message}"}` | JSON format for scripting |
| `count-only` | `%{unread-count}` | Just the notification count |
| `levels` | `Severity: %{highest-severity} \| Unread: %{unread-count}` | Severity level + count |
| `panes` | `%{pane-list} (%{unread-count})` | Panes with count |

#### Template Variables (13 Total)

**Count Variables**:
- `%{unread-count}` – Number of active notifications
- `%{active-count}` – Alias for unread-count
- `%{total-count}` – Alias for unread-count
- `%{read-count}` – Number of dismissed notifications
- `%{dismissed-count}` – Number of dismissed notifications

**Severity Count Variables**:
- `%{critical-count}` – Number of critical notifications
- `%{error-count}` – Number of error notifications
- `%{warning-count}` – Number of warning notifications
- `%{info-count}` – Number of info notifications

**Content Variables**:
- `%{latest-message}` – Text of most recent active notification

**Boolean Variables** (return "true" or "false"):
- `%{has-unread}` – True if any active notifications exist
- `%{has-active}` – Alias for has-unread
- `%{has-dismissed}` – True if any dismissed notifications exist

**Severity Variable**:
- `%{highest-severity}` – Ordinal (1=critical, 2=error, 3=warning, 4=info)

**Session/Window/Pane Variables** (reserved for future):
- `%{session-list}` – Sessions with active notifications
- `%{window-list}` – Windows with active notifications
- `%{pane-list}` – Panes with active notifications

#### Flags

- `--format=<format>` – Preset name (`compact`, `detailed`, `json`, etc.) or custom template using `%{variable}` syntax (default: `compact`)

#### Examples

```bash
# Default compact format
tmux-intray status
# Output: [3] Build completed successfully

# Show detailed format
tmux-intray status --format=detailed
# Output: 3 unread, 2 read | Latest: Build completed successfully

# JSON output for scripting
tmux-intray status --format=json
# Output: {"unread":3,"total":3,"message":"Build completed successfully"}

# Custom template - severity summary
tmux-intray status --format='C:%{critical-count} E:%{error-count} W:%{warning-count}'
# Output: C:1 E:2 W:3

# Just the count (for status bar)
tmux-intray status --format=count-only
# Output: 3

# Custom message with icon
tmux-intray status --format='📬 %{unread-count} notifications'
# Output: 📬 3 notifications
```

#### Environment Variables

- `TMUX_INTRAY_STATUS_FORMAT` – Default format (same as `--format`, CLI flag takes precedence)

#### Integration Example

For tmux status bar in `.tmux.conf`:

```bash
# Compact status showing count and message
set -g status-right "#(tmux-intray status --format=compact) %H:%M"

# Or just the count
set -g status-right "Inbox: #(tmux-intray status --format=count-only) %H:%M"
```

#### Exit Codes

- `0` - Success
- `1` - Error (tmux not running, invalid template, or database error)

#### Full Documentation

See [Status Command Guide](../status-command-guide.md) for:
- Detailed variable descriptions and use cases
- All 6 preset definitions with examples
- Real-world use cases (status bar integration, scripts, etc.)
- Troubleshooting section
- Advanced examples
