#!/usr/bin/env bash
# Example pre-add hook: macOS UI notification (visual only)
# Environment variables available:
#   NOTIFICATION_ID, LEVEL, MESSAGE, TIMESTAMP, SESSION, WINDOW, PANE, PANE_CREATED
#
# This hook triggers a macOS notification when a notification is added.
# It displays a visual notification without any sound.
# For sound-only notifications, see 05-macos-sound.sh

set -euo pipefail

# Display notification using osascript (no sound)
osascript -e "display notification \"Message: $MESSAGE\" with title \"tmux-intray\"" 2>/dev/null || true
