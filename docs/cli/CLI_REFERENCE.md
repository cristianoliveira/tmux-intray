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
  status-panel Status bar indicator script (for tmux status-right)
  tui          Interactive terminal UI for notifications
  version      Show version information

Flags:
  -h, --help      help for tmux-intray
  -v, --version   version for tmux-intray

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
