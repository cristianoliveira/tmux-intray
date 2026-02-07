# tmux-intray CLI Reference

Version: v0.1.0

## Overview

tmux-intray is a quiet inbox for things that happen while you're not looking.

## Global Usage

```
tmux-intray v0.1.0

A quiet inbox for things that happen while you're not looking.

USAGE:
    tmux-intray [COMMAND] [OPTIONS]

COMMANDS:
    show            Show all items in the tray (deprecated, use list)
    add <message>   Add a new item to the tray
    list            List notifications with filters and formats
    dismiss <id>    Dismiss a notification
    clear           Clear all items from the tray
    cleanup         Clean up old dismissed notifications
    toggle          Toggle the tray visibility
    jump <id>       Jump to the pane of a notification
    status          Show notification status summary
    status-panel    Status bar indicator script (for tmux status-right)
    follow          Monitor notifications in real-time
    tui             Interactive terminal UI for notifications
    settings        Manage TUI settings
    help            Show this help message
    version         Show version information

OPTIONS:
    -h, --help      Show help message
```

## Commands

### show

Show all items in the tray (deprecated, use list)

*No detailed help available.*

### add <message>

Add a new item to the tray

```
tmux-intray add - Add a new item to the tray

USAGE:
    tmux-intray add [OPTIONS] <message>

OPTIONS:
    --session <id>          Associate with specific session ID
    --window <id>           Associate with specific window ID
    --pane <id>             Associate with specific pane ID
    --pane-created <time>   Pane creation timestamp (seconds since epoch)
    --no-associate          Do not associate with any pane
    --level <level>         Notification level: info, warning, error, critical (default: info)
    -h, --help              Show this help

If no pane association options are provided, automatically associates with
the current tmux pane (if inside tmux). Use --no-associate to skip.
```

### list

List notifications with filters and formats

```
tmux-intray list - List notifications

USAGE:
    tmux-intray list [OPTIONS]

OPTIONS:
    --active             Show active notifications (default)
    --dismissed          Show dismissed notifications
    --all                Show all notifications
    --pane <id>          Filter notifications by pane ID (e.g., %0)
    --level <level>      Filter notifications by level: info, warning, error, critical
    --session <id>       Filter notifications by session ID
    --window <id>        Filter notifications by window ID
    --older-than <days>  Show notifications older than N days
    --newer-than <days>  Show notifications newer than N days
    --search <pattern>   Search messages (substring match)
    --regex              Use regex search with --search
    --group-by <field>   Group notifications by field (session, window, pane, level)
    --group-count        Show only group counts (requires --group-by)
    --format=<format>    Output format: legacy, table, compact, json
    -h, --help           Show this help
```

### dismiss <id>

Dismiss a notification

```
tmux-intray dismiss - Dismiss notifications

USAGE:
    tmux-intray dismiss <id>      Dismiss a specific notification
    tmux-intray dismiss --all     Dismiss all active notifications

OPTIONS:
    -h, --help           Show this help
```

### clear

Clear all items from the tray

```
tmux-intray clear - Clear all notifications

USAGE:
    tmux-intray clear

ALIAS:
    This command is an alias for `tmux-intray dismiss --all`.

EXAMPLES:
    # Clear all active notifications
    tmux-intray clear
```

### cleanup

Clean up old dismissed notifications

```
tmux-intray cleanup - Clean up old dismissed notifications

USAGE:
    tmux-intray cleanup [OPTIONS]

OPTIONS:
    --days N          Clean up notifications dismissed more than N days ago
                      (default: TMUX_INTRAY_AUTO_CLEANUP_DAYS config value)
    --dry-run         Show what would be deleted without actually deleting
    -h, --help        Show this help

Automatically cleans up notifications that have been dismissed and are older
than the configured auto-cleanup days. This helps prevent storage bloat.
```

### toggle

Toggle the tray visibility

```
tmux-intray toggle - Toggle tray visibility

USAGE:
    tmux-intray toggle

DESCRIPTION:
    Toggles the global visibility flag for the tray. When hidden, notifications
    are still stored but may not appear in status bar indicators. This command can be bound to a tmux key binding if desired (previously bound to `prefix+i`).

EXAMPLES:
    # Toggle tray visibility
    tmux-intray toggle
```

### jump <id>

Jump to the pane of a notification

```
tmux-intray jump - Jump to notification source pane

USAGE:
    tmux-intray jump <id>

DESCRIPTION:
    Navigates to the tmux pane where the notification originated. The pane
    must still exist; if it doesn't, the command falls back to the window.

ARGUMENTS:
    <id>    Notification ID (as shown in `tmux-intray list --format=table`)

EXAMPLES:
    # Jump to pane of notification with ID 42
    tmux-intray jump 42
```

### status

Show notification status summary

```
tmux-intray status - Show notification status summary

USAGE:
    tmux-intray status [OPTIONS]

OPTIONS:
    --format=<format>    Output format: summary, levels, panes, json (default: summary)
    -h, --help           Show this help

EXAMPLES:
    tmux-intray status               # Show summary
    tmux-intray status --format=levels # Show counts by level
    tmux-intray status --format=panes  # Show counts by pane
```

### status-panel

Status bar indicator script (for tmux status-right)

```
tmux-intray status-panel - Status bar indicator script

USAGE:
    tmux-intray status-panel [OPTIONS]

OPTIONS:
    --format=<format>    Output format: compact, detailed, count-only (default: compact)
    --enabled=<0|1>      Enable/disable status indicator (default: 1)
    -h, --help           Show this help

DESCRIPTION:
    This script is designed to be used in tmux status-right configuration.
    Example: set -g status-right "#(tmux-intray status-panel) %H:%M"

    The script outputs a formatted string showing notification counts.
    When clicked, it can trigger the list command (via tmux bindings).
```

### settings

Manage TUI settings

```
tmux-intray settings - Manage TUI settings

USAGE:
    tmux-intray settings <subcommand>

SUBCOMMANDS:
    reset    Reset TUI settings to defaults
    show     Display current settings in JSON format

DESCRIPTION:
    The settings command manages TUI user preferences that are persisted
    to disk. Settings include column order, sort preferences, active filters,
    and view mode.

    Settings are stored at ~/.config/tmux-intray/settings.json
    (or $XDG_CONFIG_HOME/tmux-intray/settings.json if XDG_CONFIG_HOME is set).

EXAMPLES:
    # Show current settings
    tmux-intray settings show

    # Reset settings with confirmation
    tmux-intray settings reset

    # Reset settings without confirmation
    tmux-intray settings reset --force
```

#### settings reset

Reset TUI settings to defaults by deleting the settings file.

```
tmux-intray settings reset - Reset TUI settings to defaults

USAGE:
    tmux-intray settings reset [OPTIONS]

OPTIONS:
    --force       Reset without confirmation prompt
    -h, --help    Show this help

DESCRIPTION:
    Deletes the settings file (~/.config/tmux-intray/settings.json).
    The TUI will use default settings on the next launch.

    By default, this command prompts for confirmation before deleting.
    Use the --force flag to skip the confirmation prompt (use with caution).

    The confirmation prompt is automatically skipped in CI/test environments
    (when CI or BATS_TMPDIR environment variables are set).

EXAMPLES:
    # Reset with confirmation prompt
    tmux-intray settings reset
    > Are you sure you want to reset all settings to defaults? (y/N): y
    Settings reset to defaults

    # Reset without confirmation
    tmux-intray settings reset --force
    Settings reset to defaults
```

#### settings show

Display current TUI settings in JSON format.

```
tmux-intray settings show - Display current settings

USAGE:
    tmux-intray settings show

DESCRIPTION:
    Outputs the current TUI settings in formatted JSON.
    This is useful for inspecting your current configuration or for
    scripting purposes.

EXAMPLES:
    # Show current settings
    tmux-intray settings show
    {
      "columns": ["id", "timestamp", "state", "level", "session", "window", "pane", "message"],
      "sortBy": "timestamp",
      "sortOrder": "desc",
      "filters": {
        "level": "",
        "state": "",
        "session": "",
        "window": "",
        "pane": ""
      },
      "viewMode": "compact"
    }

    # Pipe to jq for filtering
    tmux-intray settings show | jq '.sortBy'
    "timestamp"
```

### follow

Monitor notifications in real-time

```
tmux-intray follow - Monitor notifications in real-time

USAGE:
    tmux-intray follow [OPTIONS]

OPTIONS:
    --all              Show all notifications (not just active)
    --dismissed        Show only dismissed notifications
    --level <level>   Filter by level (error, warning, info)
    --pane <id>       Filter by pane ID
    --interval <secs>  Poll interval (default: 1)
    -h, --help         Show this help
```

### tui

Interactive terminal UI for notifications

```
tmux-intray tui - Interactive terminal UI for notifications

USAGE:
    tmux-intray tui

KEY BINDINGS:
    j/k         Move up/down in the list
    /           Enter search mode
    :           Enter command mode
    ESC         Exit search/command mode, or quit TUI
    d           Dismiss selected notification
    Enter       Jump to pane (or execute command in command mode)
    :w          Save settings manually
    q           Quit TUI

SETTINGS:
    Settings are automatically saved on TUI exit and loaded on startup.
    Settings include column order, sort preferences, active filters,
    and view mode. Use the :w command to save manually without exiting.
    Settings file: ~/.config/tmux-intray/settings.json

    To view current settings: tmux-intray settings show
    To reset to defaults: tmux-intray settings reset
```

### help

Show this help message

*No detailed help available.*

### version

Show version information

*No detailed help available.*

