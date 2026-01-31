#!/usr/bin/env bash
# Example: Simulating a long-running process with notifications

set -euo pipefail

TMUX_INTRAY_BIN="./bin/tmux-intray"
PROJECT_NAME="Example Build"

echo "Starting $PROJECT_NAME..."

# Notify start
"$TMUX_INTRAY_BIN" add "🚀 Starting $PROJECT_NAME..."

# Simulate some work
for step in {1..5}; do
    echo "Step $step/5 in progress..."
    sleep 2
done

# Notify completion
"$TMUX_INTRAY_BIN" add "✅ $PROJECT_NAME completed successfully!"

echo ""
echo "Tray contents:"
"$TMUX_INTRAY_BIN" show
