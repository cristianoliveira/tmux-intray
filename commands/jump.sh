#!/usr/bin/env bash
# Jump command - Navigate to the pane of a notification

# Source core libraries
# shellcheck source=../lib/core.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$(dirname "${BASH_SOURCE[0]}")/../lib/core.sh"

jump_command() {
    if [[ $# -ne 1 ]]; then
        error "'jump' requires a notification ID"
        echo "Usage: tmux-intray jump <id>" >&2
        exit 1
    fi
    
    local id="$1"
    
    ensure_tmux_running
    
    # Get latest line for this ID
    local line
    line=$(_with_lock "$LOCK_DIR" _get_latest_line_for_id "$id")
    if [[ -z "$line" ]]; then
        error "Notification with ID $id not found"
        exit 1
    fi
    
    # Parse line
    local state session window pane
    _parse_notification_line "$line" _ _ state session window pane _ _ _
    
    if [[ "$state" == "dismissed" ]]; then
        info "Notification $id is dismissed, but jumping anyway"
    fi
    
    if [[ -z "$session" ]] || [[ -z "$window" ]] || [[ -z "$pane" ]]; then
        error "Notification $id has no pane association"
        exit 1
    fi
    
    # Jump to pane
    if jump_to_pane "$session" "$window" "$pane"; then
        success "Jumped to pane ${session}:${window}:${pane}"
    else
        error "Failed to jump to pane (maybe pane no longer exists)"
        exit 1
    fi
}