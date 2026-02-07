#!/usr/bin/env bash
# Example pre-add hook: log notification addition
# Environment variables available:
#   NOTIFICATION_ID, LEVEL, MESSAGE, TIMESTAMP, SESSION, WINDOW, PANE, PANE_CREATED

LOG_FILE="${HOOK_LOG_FILE:-$HOME/.local/state/tmux-intray/hooks.log}"

mkdir -p "$(dirname "$LOG_FILE")"
{
    echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") [pre-add] ID=$NOTIFICATION_ID level=$LEVEL"
    echo "  message: $MESSAGE"
    echo "  context: ${SESSION:-none}/${WINDOW:-none}/${PANE:-none}"
} >>"$LOG_FILE" 2>/dev/null || true
