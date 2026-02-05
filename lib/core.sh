#!/usr/bin/env bash
# Core functions for tmux-intray

# Determine absolute directory of this script
_TMUX_INTRAY_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load storage and configuration
# shellcheck source=./storage.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$_TMUX_INTRAY_LIB_DIR/storage.sh"
# shellcheck source=./config.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$_TMUX_INTRAY_LIB_DIR/config.sh"

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
    # Note: pane_created is not a standard tmux format variable, so we exclude it
    local format="#{session_id} #{window_id} #{pane_id}"
    local result
    if ! result=$(tmux display -p "$format" 2>/dev/null); then
        error "Failed to get tmux context"
        return 1
    fi
    # Split by space
    IFS=' ' read -r current_session current_window current_pane current_pane_created <<<"$result"
    # Set pane_created to empty since it's not available in tmux format variables
    current_pane_created=""
}

# Validate that a pane exists given session, window, pane IDs
# Arguments: session_id window_id pane_id
# Returns: 0 if pane exists, 1 otherwise
validate_pane_exists() {
    local session="$1" window="$2" pane="$3"
    # List panes in target window and check if pane ID matches
    local pane_list
    if ! pane_list=$(tmux list-panes -t "${session}:${window}" -F "#{pane_id}" 2>/dev/null); then
        return 1 # session or window doesn't exist
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
        warning "Pane ${session}:${window}.${pane} does not exist, jumping to window instead"
        if ! tmux select-window -t "${session}:${window}"; then
            error "Window ${session}:${window} does not exist"
            return 1
        fi
        return 0
    fi
    if ! tmux select-window -t "${session}:${window}"; then
        error "Window ${session}:${window} does not exist"
        return 1
    fi
    tmux select-pane -t "${session}:${window}.${pane}"
}

get_tray_items() {
    # Return newline-separated items from storage
    # Each notification already includes a timestamp in the message
    local state_filter="${1:-active}"
    local items=""
    local lines
    lines=$(storage_list_notifications "$state_filter")

    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            # shellcheck disable=SC2034
            # Variables are used indirectly via printf -v assignment in _parse_notification_line.
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            # Unescape message
            message=$(_unescape_message "$message")
            items="${items:+$items$'\n'}$message"
        fi
    done <<<"$lines"

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
