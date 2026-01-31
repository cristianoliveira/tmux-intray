#!/usr/bin/env bash
# Add command - Add a new item to the tray
# This is an example of a command with its own sub-modules

# Source local modules
COMMAND_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# shellcheck source=./commands/add/modules/validators.sh
source "$COMMAND_DIR/add/modules/validators.sh"
# shellcheck source=./commands/add/modules/formatters.sh
source "$COMMAND_DIR/add/modules/formatters.sh"

add_command() {
    if [[ $# -eq 0 ]]; then
        error "'add' requires a message"
        echo "Usage: tmux-intray add <message>" >&2
        exit 1
    fi

    ensure_tmux_running

    local message="$*"
    
    # Use local module functions
    validate_message "$message"
    local formatted_message
    formatted_message=$(format_message "$message")
    
    add_tray_item "$formatted_message"
    success "Item added to tray"
}
