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

