#!/usr/bin/env bash
# Example pre-add hook: tmux status bar notification
# Environment variables available:
#   NOTIFICATION_ID, LEVEL, MESSAGE, TIMESTAMP, SESSION, WINDOW, PANE, PANE_CREATED
#
# This hook displays a yellow status notification bar at the bottom of tmux
# when a notification is added. The message is displayed for a few seconds.

set -euo pipefail

# Duration to display the notification in milliseconds (default: 3000ms)
DISPLAY_DURATION="${TMUX_NOTIFICATION_DURATION:-3000}"
NOTIFICATION_MESSAGE="${MESSAGE:-"Test notification"}"

# Display the notification in tmux status bar
tmux display-message -d "$DISPLAY_DURATION" "tmux-intray: $NOTIFICATION_MESSAGE" 2>/dev/null || true
