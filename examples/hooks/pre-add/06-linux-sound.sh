#!/usr/bin/env bash
# Example pre-add hook: Linux sound notification
# Environment variables available:
#   NOTIFICATION_ID, LEVEL, MESSAGE, TIMESTAMP, SESSION, WINDOW, PANE, PANE_CREATED
#
# This hook plays a sound when a notification is added, without displaying any
# UI notification. It uses paplay (PulseAudio) or aplay (ALSA) to play audio.

set -euo pipefail

# Sound file path for notification
# Default: freedesktop message sound
SOUND_FILE="${LINUX_SOUND_FILE:-/usr/share/sounds/freedesktop/stereo/message.oga}"

# Play notification sound if file exists
if [ -f "$SOUND_FILE" ]; then
    # Try paplay (PulseAudio) first, then fallback to aplay (ALSA)
    paplay "$SOUND_FILE" 2>/dev/null || aplay "$SOUND_FILE" 2>/dev/null || true
fi
