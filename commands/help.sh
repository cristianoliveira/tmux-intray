#!/usr/bin/env bash
# Help command - Show help message

help_command() {
    cat <<EOF
tmux-intray v${VERSION}

A quiet inbox for things that happen while you're not looking.

USAGE:
    tmux-intray [COMMAND] [OPTIONS]

COMMANDS:
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

EOF
}
