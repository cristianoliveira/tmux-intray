#!/usr/bin/env bash
# Example pre-add hook: macOS sound notification
# Environment variables available:
#   NOTIFICATION_ID, LEVEL, MESSAGE, TIMESTAMP, SESSION, WINDOW, PANE, PANE_CREATED
#
# This hook plays a sound when a notification is added, without displaying any
# UI notification. It uses afplay to play system sounds or custom audio files.

set -euo pipefail

# Sound file path for notification (default: macOS Ping system sound)
# You can specify a custom file path like: /path/to/your/sound.mp3
# Or use a system sound name like: Ping, Glass, Purr, Sosumi, etc.
SOUND_FILE="${MACOS_SOUND_FILE:-/System/Library/Sounds/Ping.aiff}"

# Play the sound using afplay
if [ -f "$SOUND_FILE" ]; then
    afplay "$SOUND_FILE" 2>/dev/null || true
else
    # Fallback: use osascript beep if sound file not found
    osascript -e 'beep 1' 2>/dev/null || true
fi
