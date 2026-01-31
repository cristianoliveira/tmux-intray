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
            local id timestamp state session window pane message
            _parse_notification_line "$line" id timestamp state session window pane message
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
    # Ensure migration has happened
    _migrate_if_needed
    # Add to storage (timestamp auto-generated, session/window/pane placeholders)
    storage_add_notification "$item" "" "" "" ""
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
