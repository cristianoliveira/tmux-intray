#!/usr/bin/env bash
# Show command - Display all items in the tray
# This is an example of a command with its own sub-modules

# Source local modules
COMMAND_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# shellcheck source=./commands/show/modules/filters.sh
source "$COMMAND_DIR/show/modules/filters.sh"
# shellcheck source=./commands/show/modules/display.sh
source "$COMMAND_DIR/show/modules/display.sh"

show_command() {
    ensure_tmux_running

    local items
    items=$(get_tray_items)

    if [[ -z "$items" ]]; then
        info "Tray is empty"
        return
    fi

    # Use local module functions to process and display
    local formatted_items
    formatted_items=$(format_items "$items")
    echo "$formatted_items"
}
