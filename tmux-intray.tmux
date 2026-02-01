#!/usr/bin/env bash
# tmux-intray - Main tmux plugin entry point

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Get the project root directory
PROJECT_ROOT="$(dirname "$CURRENT_DIR")"

# Source the main library
source "$PROJECT_ROOT/lib/tmux-intray.sh"

# Set up tmux key bindings
set_tmux_bindings() {
    tmux bind-key -T prefix i run-shell "$PROJECT_ROOT/bin/tmux-intray toggle"
    tmux bind-key -T prefix I run-shell "$PROJECT_ROOT/bin/tmux-intray list"
}

# Initialize the plugin
initialize_intray() {
    tmux set-environment -g TMUX_INTRAY_VERSION "0.1.0"
    tmux set-environment -g TMUX_INTRAY_DIR "$PROJECT_ROOT"
}

# Update tmux status option with current notification count
update_tmux_status_option() {
    # Source storage library to get update function
    # shellcheck source=../lib/storage.sh disable=SC1091
    source "$PROJECT_ROOT/lib/storage.sh"
    _update_tmux_status
}

initialize_intray
update_tmux_status_option
set_tmux_bindings
