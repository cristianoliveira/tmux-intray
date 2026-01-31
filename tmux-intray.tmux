#!/usr/bin/env bash
# tmux-intray - Main tmux plugin entry point

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Get the project root directory
PROJECT_ROOT="$(dirname "$CURRENT_DIR")"

# Source the main library
source "$PROJECT_ROOT/lib/tmux-intray.sh"

# Set up tmux key bindings
set_tmux_bindings() {
    tmux bind-key -T prefix i run-shell "$PROJECT_ROOT/bin/tmux-intray toggle"
    tmux bind-key -T prefix I run-shell "$PROJECT_ROOT/bin/tmux-intray show"
}

# Initialize the plugin
initialize_intray() {
    tmux set-environment -g TMUX_INTRAY_VERSION "0.1.0"
    tmux set-environment -g TMUX_INTRAY_DIR "$PROJECT_ROOT"
}

initialize_intray
set_tmux_bindings
