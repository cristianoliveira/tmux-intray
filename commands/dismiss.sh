#!/usr/bin/env bash
# Dismiss command - Dismiss notifications

# Source core libraries
# shellcheck source=../lib/core.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$(dirname "${BASH_SOURCE[0]}")/../lib/core.sh"

dismiss_command() {
    local dismiss_all=false
    local id=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
        --all)
            dismiss_all=true
            shift
            ;;
        --help | -h)
            cat <<EOF
tmux-intray dismiss - Dismiss notifications

USAGE:
    tmux-intray dismiss <id>      Dismiss a specific notification
    tmux-intray dismiss --all     Dismiss all active notifications

OPTIONS:
    -h, --help           Show this help

EOF
            return 0
            ;;
        *)
            # Assume it's an ID
            if [[ -n "$id" ]]; then
                error "Multiple IDs specified: $id and $1"
                return 1
            fi
            if ! [[ "$1" =~ ^[0-9]+$ ]]; then
                error "Invalid notification ID: $1 (must be a number)"
                return 1
            fi
            id="$1"
            shift
            ;;
        esac
    done

    ensure_tmux_running

    if [[ "$dismiss_all" == true ]]; then
        if [[ -n "$id" ]]; then
            error "Cannot specify both --all and ID"
            return 1
        fi
        storage_dismiss_all
        success "All active notifications dismissed"
    elif [[ -n "$id" ]]; then
        if storage_dismiss_notification "$id"; then
            success "Notification $id dismissed"
        else
            # Error already printed by storage_dismiss_notification
            return 1
        fi
    else
        error "Either specify an ID or use --all"
        echo "Usage: tmux-intray dismiss <id> | tmux-intray dismiss --all" >&2
        return 1
    fi
}
