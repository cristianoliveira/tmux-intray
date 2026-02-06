#!/usr/bin/env bash
# tmux-intray - Main tmux plugin entry point

# Get the plugin directory
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Set up tmux key bindings
set_tmux_bindings() {
    # Use tmux-intray from PATH if available, otherwise use local build
    if command -v tmux-intray >/dev/null 2>&1; then
        TMUX_INTRAY="tmux-intray"
    else
        TMUX_INTRAY="go run ./cmd/tmux-intray"
    fi

    tmux bind-key -T prefix I run-shell "$TMUX_INTRAY follow"
    tmux bind-key -T prefix J run-shell "tmux popup -E -h 80% -w 80% \"$TMUX_INTRAY tui\""
}

# Initialize the plugin
initialize_intray() {
    tmux set-environment -g TMUX_INTRAY_VERSION "0.1.0"
    tmux set-environment -g TMUX_INTRAY_DIR "$PLUGIN_DIR"
}

initialize_intray
set_tmux_bindings
