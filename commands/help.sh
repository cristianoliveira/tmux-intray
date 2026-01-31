#!/usr/bin/env bash
# Help command - Show help message

help_command() {
    cat <<EOF
tmux-intray v${VERSION}

A quiet inbox for things that happen while you're not looking.

USAGE:
    tmux-intray [COMMAND] [OPTIONS]

COMMANDS:
    show            Show all items in the tray (deprecated, use list)
    add <message>   Add a new item to the tray
    list            List notifications with filters and formats
    dismiss <id>    Dismiss a notification
    clear           Clear all items from the tray
    toggle          Toggle the tray visibility
    help            Show this help message
    version         Show version information

OPTIONS:
    -h, --help      Show help message

EOF
}
