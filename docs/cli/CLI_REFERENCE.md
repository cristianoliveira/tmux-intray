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

*No detailed help available.*

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

*No detailed help available.*

### jump <id>

Jump to the pane of a notification

*No detailed help available.*

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

### help

Show this help message

*No detailed help available.*

### version

Show version information

*No detailed help available.*

