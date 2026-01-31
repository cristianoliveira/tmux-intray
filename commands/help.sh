#!/usr/bin/env bash
# Help command - Show help message

help_command() {
    cat << EOF
tmux-intray v${VERSION}

A quiet inbox for things that happen while you're not looking.

USAGE:
    tmux-intray [COMMAND] [OPTIONS]

COMMANDS:
    show            Show all items in the tray
    add <message>   Add a new item to the tray
    clear           Clear all items from the tray
    toggle          Toggle the tray visibility
    help            Show this help message
    version         Show version information

OPTIONS:
    -h, --help      Show help message

EOF
}
