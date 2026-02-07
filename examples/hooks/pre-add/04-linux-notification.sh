#!/usr/bin/env bash
# Example pre-add hook: Linux desktop notification (visual only)
# Environment variables available:
#   NOTIFICATION_ID, LEVEL, MESSAGE, TIMESTAMP, SESSION, WINDOW, PANE, PANE_CREATED
#
# This hook triggers a Linux desktop notification when a notification is added.
# It displays a visual notification without any sound.
# For sound-only notifications, see 06-linux-sound.sh

set -euo pipefail

# Display desktop notification using notify-send (no sound)
notify-send "tmux-intray" "$MESSAGE" \
    --icon=dialog-information \
    --urgency=normal \
    --app-name="tmux-intray" 2>/dev/null || true
