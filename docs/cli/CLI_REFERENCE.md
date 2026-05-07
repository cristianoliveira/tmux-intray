# tmux-intray CLI Reference

Version: 1.0.0+fb02412

## Overview

tmux-intray is a quiet inbox for things that happen while you're not looking.

## Global Usage

```
A quiet inbox for things that happen while you're not looking.

Usage:
  tmux-intray [flags]
  tmux-intray [command]

Available Commands:
  add         Add a new item to the tray
  cleanup     Clean up old dismissed notifications
  clear       Clear all items from the tray
  completion  Generate the autocompletion script for the specified shell
  dismiss     Dismiss a notification
  follow      Monitor notifications in real-time
  help        Help about any command
  jump        Jump to the pane of a notification
  list        List notifications with filters and formats
  mark-read   Mark a notification as read
  settings    Manage TUI settings
  status      Show notification status summary
  tui         Interactive terminal UI for notifications

Flags:
  -h, --help              help for tmux-intray
      --log-file string   explicit log file path (overrides config)
  -v, --version           version for tmux-intray

Use "tmux-intray [command] --help" for more information about a command.
```

## Commands

### add

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

Usage:
  tmux-intray add [OPTIONS] <message> [flags]

Flags:
  -h, --help                  help for add
      --level string          Notification level: info, warning, error, critical (default "info")
      --no-associate          Do not associate with any pane
      --pane string           Associate with specific pane ID
      --pane-created string   Pane creation timestamp (seconds since epoch)
      --session string        Associate with specific session ID
      --window string         Associate with specific window ID

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### cleanup

Clean up old dismissed notifications

*No detailed help available.*

### clear

Clear all items from the tray

```
Clear all active notifications from the tray.

This command dismisses all active notifications, running pre-clear and per-notification
hooks, and updates the tmux status option.

USAGE:
    tmux-intray clear

ALIAS:
    This command is an alias for 'tmux-intray dismiss --all'.

EXAMPLES:
    # Clear all active notifications
    tmux-intray clear

Usage:
  tmux-intray clear [flags]

Flags:
  -h, --help   help for clear

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### completion

Generate the autocompletion script for the specified shell

*No detailed help available.*

### dismiss

Dismiss a notification

```
Dismiss a specific notification by ID or all active notifications.

USAGE:
    tmux-intray dismiss <id>      Dismiss a specific notification
    tmux-intray dismiss --all     Dismiss all active notifications

OPTIONS:
    -h, --help           Show this help

Usage:
  tmux-intray dismiss [ID] [flags]

Flags:
      --all    Dismiss all active notifications
  -h, --help   help for dismiss

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### follow

Monitor notifications in real-time

```
Monitor notifications in real-time.

USAGE:
    tmux-intray follow [OPTIONS]

OPTIONS:
    --all              Show all notifications (not just active)
    --dismissed        Show only dismissed notifications
    --level <level>   Filter by level (error, warning, info)
    --pane <id>       Filter by pane ID
    --interval <secs>  Poll interval (default: 1)
    -h, --help         Show this help

Usage:
  tmux-intray follow [flags]

Flags:
      --all              Show all notifications (not just active)
      --dismissed        Show only dismissed notifications
  -h, --help             help for follow
      --interval float   Poll interval in seconds (default: 1) (default 1)
      --level string     Filter by level (error, warning, info)
      --pane string      Filter by pane ID

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### help

Help about any command

*No detailed help available.*

### jump

Jump to the pane of a notification

```
Jump to the pane of a notification.

USAGE:
    tmux-intray jump <id>

DESCRIPTION:
    Navigates to the tmux pane where the notification originated. The pane
    must still exist; if it doesn't, the command falls back to the window.
    By default, a successful jump automatically marks the notification as read.
    Use --no-mark-read to disable this behavior.

ARGUMENTS:
    <id>    Notification ID (as shown in 'tmux-intray list --format=table')

OPTIONS:
    --no-mark-read    Do not mark the notification as read after a successful jump

EXAMPLES:
    # Jump to pane of notification with ID 42
    tmux-intray jump 42

    # Jump without marking notification as read
    tmux-intray jump --no-mark-read 42

Usage:
  tmux-intray jump [flags]

Flags:
  -h, --help           help for jump
      --no-mark-read   do not mark notification as read after successful jump

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### list

List notifications with filters and formats

```
List notifications with filters and formats.

USAGE:
    tmux-intray list [OPTIONS]

OPTIONS:
    --tab <tab>          Show special tab view: recents, sessions, all
    --active             Show active notifications (default)
    --dismissed          Show dismissed notifications
    --all                Show all notifications
    --pane <id|title>    Filter notifications by pane ID or pane title
    --level <level>      Filter notifications by level: info, warning, error, critical
    --session <id|name>  Filter notifications by session ID or session name
    --window <id|name>   Filter notifications by window ID or window name
    --ids                Show raw tmux session/window/pane IDs instead of resolved names
    --show-stale         Include notifications whose tmux session/window/pane no longer exists
    --older-than <days>  Show notifications older than N days
    --newer-than <days>  Show notifications newer than N days
    --search <pattern>   Search messages (substring match)
    --regex              Use regex search with --search
    --group-by <field>   Group notifications by field (session, window, pane, level, message)
    --group-count        Show only group counts (requires --group-by)
    --filter <status>    Filter notifications by read status: read, unread
    --format=<format>    Output format: simple (default), legacy, table, compact, json

TAB VIEWS:
    --tab=recents        Show recent unread notifications (max 1 per session, last hour)
    --tab=sessions       Show unique sessions with notifications
    --tab=all            Show all notifications (same as --all)

ORDERING:
    Unread notifications are listed first, then read notifications.
    Relative order remains unchanged within each group.
    -h, --help           Show this help

Usage:
  tmux-intray list [flags]

Flags:
      --active            Show active notifications (default)
      --all               Show all notifications
      --dismissed         Show dismissed notifications
      --filter string     Filter notifications by read status: read, unread
      --format string     Output format: simple (default), legacy, table, compact, json (default "simple")
      --group-by string   Group notifications by field (session, window, pane, level, message)
      --group-count       Show only group counts (requires --group-by)
  -h, --help              help for list
      --ids               Show raw tmux session/window/pane IDs instead of resolved names
      --json              Output in JSON format
      --level string      Filter notifications by level: info, warning, error, critical
      --newer-than int    Show notifications newer than N days
      --older-than int    Show notifications older than N days
      --pane string       Filter notifications by pane ID or pane title
      --regex             Use regex search with --search
      --search string     Search messages (substring match)
      --session string    Filter notifications by session ID or session name
      --show-stale        Include notifications whose tmux session/window/pane no longer exists
      --tab string        Show special tab view: recents, sessions, all
      --window string     Filter notifications by window ID or window name

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### mark-read

Mark a notification as read

```
Mark a notification as read by ID.

USAGE:
    tmux-intray mark-read <id>

OPTIONS:
    -h, --help           Show this help

Usage:
  tmux-intray mark-read <id> [flags]

Flags:
  -h, --help   help for mark-read

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### settings

Manage TUI settings

```
Manage TUI settings.

USAGE:
    tmux-intray settings <subcommand>

SUBCOMMANDS:
    reset    Reset settings to defaults
    show     Display current settings

EXAMPLES:
    # Reset settings with confirmation
    tmux-intray settings reset

    # Reset settings without confirmation
    tmux-intray settings reset --force

    # Show current settings
    tmux-intray settings show

Usage:
  tmux-intray settings [command]

Available Commands:
  reset       Reset TUI settings to defaults
  show        Display current settings

Flags:
  -h, --help   help for settings

Global Flags:
      --log-file string   explicit log file path (overrides config)

Use "tmux-intray settings [command] --help" for more information about a command.
```

### status

Show notification status summary

```
Show notification status summary with template-based formatting.

USAGE:
    tmux-intray status [OPTIONS]

OPTIONS:
    --format=<format>    Output format: preset name or custom template (default: compact)

PRESETS / FORMATS (6):
    compact      [{{unread-count}}] {{latest-message}}
    detailed     {{unread-count}} unread, {{read-count}} read | Latest: {{latest-message}}
    json         Special JSON output with counts and pane breakdown
    count-only   {{unread-count}}
    levels       Special multi-line severity count output
    panes        Special pane-count output

VARIABLES (13):
    {{unread-count}}      Number of active notifications
    {{active-count}}      Alias for unread-count
    {{total-count}}       Alias for unread-count
    {{read-count}}        Number of dismissed notifications
    {{dismissed-count}}   Number of dismissed notifications
    {{latest-message}}    Text of most recent active notification
    {{has-unread}}        true/false if any active exist
    {{has-active}}        true/false if any active exist
    {{has-dismissed}}     true/false if any dismissed exist
    {{highest-severity}}  Severity level (1=critical, 2=error, 3=warning, 4=info)
    {{session-list}}      Sessions with active notifications
    {{window-list}}       Windows with active notifications
    {{pane-list}}         Panes with active notifications

LEVEL VARIABLES (4):
    {{critical-count}}    Number of critical notifications
    {{error-count}}       Number of error notifications
    {{warning-count}}     Number of warning notifications
    {{info-count}}        Number of info notifications

EXAMPLES:
    tmux-intray status                    # compact: [0] message
    tmux-intray status --format=detailed  # detailed: 0 unread, 0 read | Latest: ...
    tmux-intray status --format=json      # JSON: {"unread":0,"total":0,...}
    tmux-intray status --format='Alerts: {{critical-count}}'
    tmux-intray status --format='{{unread-count}} new messages'
    tmux-intray status --format='C:{{critical-count}} E:{{error-count}} W:{{warning-count}}'
    tmux-intray status --format='Level {{highest-severity}}'

See docs/status-guide.md for detailed documentation and more examples.

Usage:
  tmux-intray status [flags]

Flags:
      --format string   Output format: preset name or custom template (default "compact")
  -h, --help            help for status

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

### tui

Interactive terminal UI for notifications

```
Interactive terminal UI for notifications.

USAGE:
    tmux-intray tui

KEY BINDINGS:
    j/k         Move up/down in the list
    r/a         Switch to Recents / All tabs
    Ctrl+s      Switch to Sessions tab
    /           Enter search mode
    Ctrl+v      Cycle view mode (detailed/grouped/search)
    ESC         Exit search mode, or quit TUI
    d           Dismiss selected notification
    R           Mark selected notification as read
    u           Mark selected notification as unread
    Enter       Jump to pane/window target
    q           Quit TUI

OPTIONS:
    --show-stale Include notifications whose tmux session/window/pane no longer exists

NOTES:
    - Settings are saved automatically on quit.
    - Up/Down arrows are supported in search contexts.

Usage:
  tmux-intray tui [flags]

Flags:
  -h, --help         help for tui
      --show-stale   Include notifications whose tmux session/window/pane no longer exists

Global Flags:
      --log-file string   explicit log file path (overrides config)
```

