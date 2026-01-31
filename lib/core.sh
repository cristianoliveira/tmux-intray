#!/usr/bin/env bash
# Core functions for tmux-intray

ensure_tmux_running() {
    if ! tmux has-session 2>/dev/null; then
        error "No tmux session running"
        exit 1
    fi
}

get_tray_items() {
    tmux show-environment -g TMUX_INTRAY_ITEMS 2>/dev/null || echo ""
}

add_tray_item() {
    local item="$1"
    local existing_items
    existing_items=$(get_tray_items)

    if [[ -z "$existing_items" ]]; then
        tmux set-environment -g TMUX_INTRAY_ITEMS "$item"
    else
        tmux set-environment -g TMUX_INTRAY_ITEMS "${existing_items}:${item}"
    fi
}

clear_tray_items() {
    tmux set-environment -g TMUX_INTRAY_ITEMS ""
}

get_visibility() {
    tmux show-environment -g TMUX_INTRAY_VISIBLE 2>/dev/null || echo "0"
}

set_visibility() {
    local visible="$1"
    tmux set-environment -g TMUX_INTRAY_VISIBLE "$visible"
}
