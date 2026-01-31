#!/usr/bin/env bash
# List command - Display notifications with various filters and formats

# Source core libraries
COMMAND_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# shellcheck source=../lib/core.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$(dirname "$COMMAND_DIR")/lib/core.sh"

# Source local modules if they exist
if [[ -d "$COMMAND_DIR/list/modules" ]]; then
    for module in "$COMMAND_DIR/list/modules"/*.sh; do
        # shellcheck source=/dev/null
        # Module files may not exist; we check existence before sourcing.
        source "$module"
    done
fi

# Default format function (legacy compatibility)
_format_legacy() {
    local filter="${1:-active}"
    local pane_filter="${2:-}"
    local level_filter="${3:-}"
    local items
    items=$(get_tray_items "$filter")
    
    if [[ -z "$items" ]]; then
        info "Tray is empty"
        return
    fi
    
    # Items are already newline-separated
    echo "$items"
}

# Simple table format
_format_table() {
    local filter="${1:-active}"
    local pane_filter="${2:-}"
    local level_filter="${3:-}"
    local lines
    lines=$(storage_list_notifications "$filter" "$level_filter")
    
    # Filter by pane if specified
    if [[ -n "$pane_filter" ]]; then
        local filtered_lines=""
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                local pane
                _parse_notification_line "$line" _ _ _ _ _ pane _ _ _
                if [[ "$pane" == "$pane_filter" ]]; then
                    filtered_lines="${filtered_lines}${line}\n"
                fi
            fi
        done <<< "$lines"
        lines=$(echo -e "$filtered_lines" | sed '/^$/d')
    fi
    
    if [[ -z "$lines" ]]; then
        info "No notifications found"
        return
    fi
    
    echo "ID    Timestamp                 Pane    Level   Message"
    echo "----  ------------------------  ------  ------  -------"
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp pane level message
            _parse_notification_line "$line" id timestamp _ _ _ pane message _ level
            message=$(_unescape_message "$message")
            # Truncate message for display
            local display_msg
            if [[ ${#message} -gt 35 ]]; then
                display_msg="${message:0:32}..."
            else
                display_msg="$message"
            fi
            printf "%-4s  %-25s  %-6s  %-6s  %s\n" "$id" "$timestamp" "$pane" "$level" "$display_msg"
        fi
    done <<< "$lines"
}

# Compact format (just messages)
_format_compact() {
    local filter="${1:-active}"
    local pane_filter="${2:-}"
    local level_filter="${3:-}"
    local lines
    lines=$(storage_list_notifications "$filter" "$level_filter")
    
    # Filter by pane if specified
    if [[ -n "$pane_filter" ]]; then
        local filtered_lines=""
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                local pane
                _parse_notification_line "$line" _ _ _ _ _ pane _ _ _
                if [[ "$pane" == "$pane_filter" ]]; then
                    filtered_lines="${filtered_lines}${line}\n"
                fi
            fi
        done <<< "$lines"
        lines=$(echo -e "$filtered_lines" | sed '/^$/d')
    fi
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local message
            _parse_notification_line "$line" _ _ _ _ _ _ message _ _
            message=$(_unescape_message "$message")
            echo "$message"
        fi
    done <<< "$lines"
}

list_command() {
    local filter="active"
    local format="legacy"
    local pane_filter=""
    local level_filter=""
    
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
            --pane=*)
                pane_filter="${1#*=}"
                shift
                ;;
            --pane)
                pane_filter="$2"
                shift 2
                ;;
            --level=*)
                level_filter="${1#*=}"
                shift
                ;;
            --level)
                level_filter="$2"
                shift 2
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
    --pane <id>          Filter notifications by pane ID (e.g., %0)
    --level <level>      Filter notifications by level: info, warning, error, critical
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
            _format_legacy "$filter" "$pane_filter" "$level_filter"
            ;;
        table)
            _format_table "$filter" "$pane_filter" "$level_filter"
            ;;
        compact)
            _format_compact "$filter" "$pane_filter" "$level_filter"
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