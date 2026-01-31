#!/usr/bin/env bash
# Core functions for tmux-intray

# Load storage and configuration
# shellcheck source=./storage.sh
source "$(dirname "${BASH_SOURCE[0]}")/storage.sh"
# shellcheck source=./config.sh
source "$(dirname "${BASH_SOURCE[0]}")/config.sh"

# Initialize configuration
config_load

ensure_tmux_running() {
    if ! tmux has-session 2>/dev/null; then
        error "No tmux session running"
        exit 1
    fi
}

# Get current tmux context (session, window, pane IDs and pane creation time)
# Sets variables: current_session, current_window, current_pane, current_pane_created
get_current_tmux_context() {
    if ! ensure_tmux_running >/dev/null 2>&1; then
        return 1
    fi
    # Use tmux display -p to get formatted strings
    local format="#{session_id} #{window_id} #{pane_id} #{pane_created}"
    local result
    result=$(tmux display -p "$format" 2>/dev/null)
    if [[ $? -ne 0 ]]; then
        error "Failed to get tmux context"
        return 1
    fi
    # Split by space
    IFS=' ' read -r current_session current_window current_pane current_pane_created <<< "$result"
}

# Validate that a pane exists given session, window, pane IDs
# Arguments: session_id window_id pane_id
# Returns: 0 if pane exists, 1 otherwise
validate_pane_exists() {
    local session="$1" window="$2" pane="$3"
    # List panes in target window and check if pane ID matches
    local pane_list
    pane_list=$(tmux list-panes -t "${session}:${window}" -F "#{pane_id}" 2>/dev/null)
    if [[ $? -ne 0 ]]; then
        return 1  # session or window doesn't exist
    fi
    # Check if pane ID is in the list
    if [[ "$pane_list" == *"$pane"* ]]; then
        return 0
    else
        return 1
    fi
}

# Jump to a specific pane
# Arguments: session_id window_id pane_id
jump_to_pane() {
    local session="$1" window="$2" pane="$3"
    if ! validate_pane_exists "$session" "$window" "$pane"; then
        error "Pane ${session}:${window}:${pane} does not exist"
        return 1
    fi
    tmux select-pane -t "${session}:${window}.${pane}"
}

# Migration check: if environment variables exist, migrate to storage
# This should be called once when needed
_migrate_if_needed() {
    local env_items
    env_items=$(tmux show-environment -g TMUX_INTRAY_ITEMS 2>/dev/null || echo "")
    env_items="${env_items#TMUX_INTRAY_ITEMS=}"
    
    if [[ -n "$env_items" ]]; then
        storage_migrate_from_env
    fi
}

get_tray_items() {
    # For backward compatibility, return colon-separated items from storage
    # Format each notification as "[timestamp] message"
    local items=""
    local lines
    lines=$(storage_list_notifications "active")
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            # Unescape message
            message=$(_unescape_message "$message")
            # Format similar to old format: [timestamp] message
            # Old format used simple timestamp; we'll use ISO timestamp trimmed maybe
            # For compatibility, keep same format: [timestamp] message
            # Extract date part only (YYYY-MM-DD HH:MM:SS)
            local display_time
            display_time=$(echo "$timestamp" | sed 's/T/ /; s/Z$//')
            items="${items:+$items:}[$display_time] $message"
        fi
    done <<< "$lines"
    
    echo "$items"
}

add_tray_item() {
    local item="$1"
    local session="${2:-}"
    local window="${3:-}"
    local pane="${4:-}"
    local pane_created="${5:-}"
    local no_auto="${6:-false}"
    local level="${7:-info}"
    # Ensure migration has happened
    _migrate_if_needed
    # If pane is empty but session/window/pane not provided, attempt to get current context
    if [[ "$no_auto" != "true" ]] && [[ -z "$session" ]] && [[ -z "$window" ]] && [[ -z "$pane" ]]; then
        if get_current_tmux_context 2>/dev/null; then
            session="$current_session"
            window="$current_window"
            pane="$current_pane"
            pane_created="$current_pane_created"
        fi
    fi
    # Add to storage
    storage_add_notification "$item" "" "$session" "$window" "$pane" "$pane_created" "$level"
}

clear_tray_items() {
    # Dismiss all active notifications
    storage_dismiss_all
}

get_visibility() {
    tmux show-environment -g TMUX_INTRAY_VISIBLE 2>/dev/null || echo "0"
}

set_visibility() {
    local visible="$1"
    tmux set-environment -g TMUX_INTRAY_VISIBLE "$visible"
}
