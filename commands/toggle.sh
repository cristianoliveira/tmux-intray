#!/usr/bin/env bash
# Toggle command - Toggle tray visibility

toggle_command() {
    ensure_tmux_running

    local visible
    visible=$(get_visibility)

    if [[ "$visible" == "1" ]]; then
        set_visibility "0"
        info "Tray hidden"
    else
        set_visibility "1"
        info "Tray visible"
    fi
}
