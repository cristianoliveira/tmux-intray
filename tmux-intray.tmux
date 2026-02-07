#!/usr/bin/env bash
# tmux-intray - Main tmux plugin entry point

# Get the plugin directory
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TMUX_INTRAY=${TMUX_INTRAY_BIN:-"tmux-intray"}

echo "tmux-intray: Version $TMUX_INTRAY_BIN"

# Set up tmux key bindings
set_tmux_bindings() {
    tmux bind-key -T prefix I run-shell "$TMUX_INTRAY follow"
    tmux bind-key -T prefix J run-shell "tmux popup -E -h 50% -w 70% \"$TMUX_INTRAY tui\""
}

# Initialize the plugin
initialize_intray() {
    tmux set-environment -g TMUX_INTRAY_VERSION "0.1.0"
    tmux set-environment -g TMUX_INTRAY_DIR "$PLUGIN_DIR"
    tmux set-environment -g TMUX_INTRAY_BIN "$TMUX_INTRAY"
}

initialize_intray
set_tmux_bindings
