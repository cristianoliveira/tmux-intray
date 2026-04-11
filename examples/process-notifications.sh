#!/usr/bin/env bash
# Example: Simulating a long-running process with notifications

set -euo pipefail

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

PROJECT_NAME="Example Build"

echo "Starting $PROJECT_NAME..."

# Notify start
tmux-intray add "🚀 Starting $PROJECT_NAME..."

# Simulate some work
for step in {1..5}; do
    echo "Step $step/5 in progress..."
    sleep 2
done

# Notify completion
tmux-intray add "✅ $PROJECT_NAME completed successfully!"

echo ""
echo "Tray contents:"
tmux-intray list
