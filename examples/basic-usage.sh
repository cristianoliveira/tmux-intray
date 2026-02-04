#!/usr/bin/env bash
# Example: Using tmux-intray from a script

TMUX_INTRAY_BIN="./bin/tmux-intray"

# Check if tmux is running
if ! tmux has-session 2>/dev/null; then
    echo "Error: No tmux session running"
    echo "Start tmux first: tmux new-session"
    exit 1
fi

# Add a simple notification
echo "Adding notification to tmux-intray..."
"$TMUX_INTRAY_BIN" add "Hello from example script!"

# Wait a moment
sleep 1

# List all items
echo ""
echo "Current tray items:"
"$TMUX_INTRAY_BIN" list --format=table
