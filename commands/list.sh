#!/usr/bin/env bash
# List command - Display notifications with various filters and formats

# Source core libraries
COMMAND_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$COMMAND_DIR")"
# shellcheck source=../lib/core.sh
source "$PROJECT_ROOT/lib/core.sh"

# Source local modules if they exist
if [[ -d "$COMMAND_DIR/list/modules" ]]; then
    for module in "$COMMAND_DIR/list/modules"/*.sh; do
        # shellcheck source=/dev/null
        source "$module"
    done
fi

# Default format function (legacy compatibility)
_format_legacy() {
    local items
    items=$(get_tray_items)
    
    if [[ -z "$items" ]]; then
        info "Tray is empty"
        return
    fi
    
    # Convert colon-separated items to lines
    echo "$items" | tr ':' '\n'
}

# Simple table format
_format_table() {
    local lines
    lines=$(storage_list_notifications "${1:-active}")
    
    if [[ -z "$lines" ]]; then
        info "No notifications found"
        return
    fi
    
    echo "ID    Timestamp                 Message"
    echo "----  ------------------------  -------"
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message
            _parse_notification_line "$line" id timestamp state session window pane message
            message=$(_unescape_message "$message")
            # Truncate message for display
            local display_msg
            if [[ ${#message} -gt 50 ]]; then
                display_msg="${message:0:47}..."
            else
                display_msg="$message"
            fi
            printf "%-4s  %-25s  %s\n" "$id" "$timestamp" "$display_msg"
        fi
    done <<< "$lines"
}

# Compact format (just messages)
_format_compact() {
    local lines
    lines=$(storage_list_notifications "${1:-active}")
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message
            _parse_notification_line "$line" id timestamp state session window pane message
            message=$(_unescape_message "$message")
            echo "$message"
        fi
    done <<< "$lines"
}

list_command() {
    local filter="active"
    local format="legacy"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --active)
                filter="active"
                shift
                ;;
            --dismissed)
                filter="dismissed"
                shift
                ;;
            --all)
                filter="all"
                shift
                ;;
            --format=*)
                format="${1#*=}"
                shift
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            --help|-h)
                cat << EOF
tmux-intray list - List notifications

USAGE:
    tmux-intray list [OPTIONS]

OPTIONS:
    --active             Show active notifications (default)
    --dismissed          Show dismissed notifications
    --all                Show all notifications
    --format=<format>    Output format: legacy, table, compact, json
    -h, --help           Show this help

EOF
                return 0
                ;;
            *)
                error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    ensure_tmux_running
    
    case "$format" in
        legacy)
            # For backward compatibility with scripts using old show command
            _format_legacy
            ;;
        table)
            _format_table "$filter"
            ;;
        compact)
            _format_compact "$filter"
            ;;
        json)
            error "JSON format not yet implemented"
            return 1
            ;;
        *)
            error "Unknown format: $format"
            return 1
            ;;
    esac
}