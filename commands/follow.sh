#!/usr/bin/env bash
# Follow command - Monitor notifications in real-time

# Source core libraries
COMMAND_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$COMMAND_DIR")"
# shellcheck source=../lib/core.sh
source "$PROJECT_ROOT/lib/core.sh"

follow_command() {
    local poll_interval=1  # seconds
    local filter="active"
    local level_filter=""
    local pane_filter=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --interval=*)
                poll_interval="${1#*=}"
                shift
                ;;
            --interval)
                poll_interval="$2"
                shift 2
                ;;
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
            --level=*)
                level_filter="${1#*=}"
                shift
                ;;
            --level)
                level_filter="$2"
                shift 2
                ;;
            --pane=*)
                pane_filter="${1#*=}"
                shift
                ;;
            --pane)
                pane_filter="$2"
                shift 2
                ;;
            --help|-h)
                cat << EOF
tmux-intray follow - Monitor notifications in real-time

USAGE:
    tmux-intray follow [OPTIONS]

OPTIONS:
    --interval <sec>     Polling interval in seconds (default: 1)
    --active             Show active notifications (default)
    --dismissed          Show dismissed notifications
    --all                Show all notifications
    --level <level>      Filter notifications by level
    --pane <id>          Filter notifications by pane ID
    -h, --help           Show this help

EXAMPLES:
    tmux-intray follow                 # Follow active notifications
    tmux-intray follow --level=error   # Follow only error notifications
    tmux-intray follow --interval=5    # Poll every 5 seconds
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
    
    # Get initial set of notification IDs
    declare -A last_seen_ids
    local lines
    lines=$(storage_list_notifications "$filter" "$level_filter")
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            # Filter by pane if specified
            if [[ -n "$pane_filter" ]] && [[ "$pane" != "$pane_filter" ]]; then
                continue
            fi
            last_seen_ids["$id"]=1
        fi
    done <<< "$lines"
    
    info "Following notifications (press Ctrl+C to stop)..."
    
    while true; do
        local lines
        lines=$(storage_list_notifications "$filter" "$level_filter")
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                local id timestamp state session window pane message pane_created level
                _parse_notification_line "$line" id timestamp state session window pane message pane_created level
                # Filter by pane if specified
                if [[ -n "$pane_filter" ]] && [[ "$pane" != "$pane_filter" ]]; then
                    continue
                fi
                # If ID not seen before, it's new
                if [[ -z "${last_seen_ids["$id"]:-}" ]]; then
                    last_seen_ids["$id"]=1
                    # Output notification
                    message=$(_unescape_message "$message")
                    local display_time
                    display_time=$(echo "$timestamp" | sed 's/T/ /; s/Z$//')
                    echo "[$display_time] [$level] $message"
                fi
            fi
        done <<< "$lines"
        sleep "$poll_interval"
    done
}