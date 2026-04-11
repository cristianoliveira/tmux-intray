#!/usr/bin/env bash
# Example: Using tmux-intray from a script

# Check if tmux is running
if ! tmux list-sessions 2>/dev/null | grep -q .; then
    echo "Error: No tmux session running"
    echo "Start tmux first: tmux new-session"
    exit 1
fi

# Check if tmux-intray is available
if ! command -v tmux-intray &>/dev/null; then
    echo "Error: tmux-intray not found"
    echo "Install it first: go install github.com/cristianoliveira/tmux-intray@latest"
    exit 1
fi

# Add a simple notification
echo "Adding notification to tmux-intray..."
tmux-intray add "Hello from example script!"

# Wait a moment
sleep 1

# List all items
echo ""
echo "Current tray items:"
tmux-intray list --format=table
