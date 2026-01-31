#!/usr/bin/env bash
# Add command - Add a new item to the tray
# This is an example of a command with its own sub-modules

# Source local modules
COMMAND_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./commands/add/modules/validators.sh
source "$COMMAND_DIR/add/modules/validators.sh"
# shellcheck source=./commands/add/modules/formatters.sh
source "$COMMAND_DIR/add/modules/formatters.sh"

add_command() {
    local session="" window="" pane="" pane_created="" no_associate=false level="info"

    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
        --session=*)
            session="${1#*=}"
            shift
            ;;
        --session)
            session="$2"
            shift 2
            ;;
        --window=*)
            window="${1#*=}"
            shift
            ;;
        --window)
            window="$2"
            shift 2
            ;;
        --pane=*)
            pane="${1#*=}"
            shift
            ;;
        --pane)
            pane="$2"
            shift 2
            ;;
        --pane-created=*)
            pane_created="${1#*=}"
            shift
            ;;
        --pane-created)
            pane_created="$2"
            shift 2
            ;;
        --no-associate)
            no_associate=true
            shift
            ;;
        --level=*)
            level="${1#*=}"
            shift
            ;;
        --level)
            level="$2"
            shift 2
            ;;
        --help | -h)
            cat <<EOF
tmux-intray add - Add a new item to the tray

USAGE:
    tmux-intray add [OPTIONS] <message>

OPTIONS:
    --session <id>          Associate with specific session ID
    --window <id>           Associate with specific window ID
    --pane <id>             Associate with specific pane ID
    --pane-created <time>   Pane creation timestamp (seconds since epoch)
    --no-associate          Do not associate with any pane
    --level <level>         Notification level: info, warning, error, critical (default: info)
    -h, --help              Show this help

If no pane association options are provided, automatically associates with
the current tmux pane (if inside tmux). Use --no-associate to skip.
EOF
            return 0
            ;;
        --*)
            error "Unknown option: $1"
            echo "Usage: tmux-intray add [OPTIONS] <message>" >&2
            exit 1
            ;;
        *)
            # First non-option argument is the start of the message
            break
            ;;
        esac
    done

    if [[ $# -eq 0 ]]; then
        error "'add' requires a message"
        echo "Usage: tmux-intray add [OPTIONS] <message>" >&2
        exit 1
    fi

    ensure_tmux_running

    local message="$*"

    # Use local module functions
    validate_message "$message"
    local formatted_message
    formatted_message=$(format_message "$message")

    if [[ "$no_associate" == true ]]; then
        # Pass empty strings and disable auto-detection
        add_tray_item "$formatted_message" "" "" "" "" "true" "$level"
    else
        add_tray_item "$formatted_message" "$session" "$window" "$pane" "$pane_created" "false" "$level"
    fi
    success "Item added to tray"
}
