#!/usr/bin/env bash
# Follow command - Monitor notifications in real-time

set -euo pipefail

# Source core libraries
# shellcheck source=../lib/core.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$(dirname "${BASH_SOURCE[0]}")/../lib/core.sh"

follow_command() {
    local poll_interval=1 # seconds
    local filter="active"
    local level_filter=""
    local pane_filter=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
        --all)
            filter="all"
            shift
            ;;
        --dismissed)
            filter="dismissed"
            shift
            ;;
        --level)
            if [[ -z "${2:-}" ]]; then
                error "--level requires an argument (error, warning, info)"
                return 1
            fi
            level_filter="$2"
            shift 2
            ;;
        --pane)
            if [[ -z "${2:-}" ]]; then
                error "--pane requires a pane ID"
                return 1
            fi
            pane_filter="$2"
            shift 2
            ;;
        --interval)
            if [[ -z "${2:-}" ]]; then
                error "--interval requires a number (seconds)"
                return 1
            fi
            poll_interval="$2"
            shift 2
            ;;
        --help | -h)
            cat <<EOF
tmux-intray follow - Monitor notifications in real-time

USAGE:
    tmux-intray follow [OPTIONS]

OPTIONS:
    --all              Show all notifications (not just active)
    --dismissed        Show only dismissed notifications
    --level <level>   Filter by level (error, warning, info)
    --pane <id>       Filter by pane ID
    --interval <secs>  Poll interval (default: 1)
    -h, --help         Show this help

EOF
            return 0
            ;;
        *)
            error "Unknown option: $1"
            return 1
            ;;
        esac
    done

    ensure_tmux_running

    local current_line_count=0

    # Clear screen
    clear

    info "Monitoring notifications (Ctrl+C to stop)..."
    echo ""

    # Main monitoring loop
    while true; do
        # Get notifications
        local lines
        lines=$(storage_list_notifications "$filter" "$level_filter")

        # Apply pane filter if specified
        if [[ -n "$pane_filter" ]]; then
            lines=$(echo "$lines" | awk -F'\t' -v pane="$pane_filter" '$6 == pane || $6 == ""')
        fi

        # Check if new notifications appeared
        local new_line_count
        new_line_count=$(echo "$lines" | wc -l | tr -d ' ')

        if [[ "$new_line_count" -gt "$current_line_count" ]]; then
            # New notification(s)
            local new_lines
            new_lines=$(echo "$lines" | tail -n $((new_line_count - current_line_count)))

            while IFS= read -r line; do
                if [[ -n "$line" ]]; then
                    # shellcheck disable=SC2034
                    # Variables are used later in the loop; ShellCheck can't see usage across lines.
                    local timestamp message pane level
                    timestamp=$(echo "$line" | cut -f2)
                    message=$(echo "$line" | cut -f7)
                    pane=$(echo "$line" | cut -f6)
                    level=$(echo "$line" | cut -f9)

                    # Format timestamp
                    local display_time
                    display_time=$(echo "$timestamp" | sed 's/T/ /; s/Z$//')

                    # Format message
                    local formatted_message
                    formatted_message=$(printf "[%s] [%s] %s" "$display_time" "$level" "$message")

                    # Color based on level
                    case "$level" in
                    error)
                        echo -e "\033[0;31m$formatted_message\033[0m"
                        ;;
                    warning)
                        echo -e "\033[1;33m$formatted_message\033[0m"
                        ;;
                    info)
                        echo "$formatted_message"
                        ;;
                    esac

                    # Show pane info if available
                    if [[ -n "$pane" ]]; then
                        echo "  └─ From pane: $pane"
                    fi
                fi
            done <<<"$new_lines"
        fi

        current_line_count="$new_line_count"
        sleep "$poll_interval"
    done
}
