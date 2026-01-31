#!/usr/bin/env bash
# Clear command - Clear all items from the tray

clear_command() {
    ensure_tmux_running
    clear_tray_items
    success "Tray cleared"
}
