#!/usr/bin/env bash
# Status command - Show notification status summary

# Source core libraries
COMMAND_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$COMMAND_DIR")"
# shellcheck source=../lib/core.sh
source "$PROJECT_ROOT/lib/core.sh"

# Default configuration
TMUX_INTRAY_STATUS_FORMAT="${TMUX_INTRAY_STATUS_FORMAT:-summary}"

# Helper: get active count
_status_get_active_count() {
    storage_get_active_count
}

# Helper: get counts by level
_status_get_counts_by_level() {
    local lines
    lines=$(storage_list_notifications "active")
    
    local info=0 warning=0 error=0 critical=0
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            case "$level" in
                info)
                    ((info++))
                    ;;
                warning)
                    ((warning++))
                    ;;
                error)
                    ((error++))
                    ;;
                critical)
                    ((critical++))
                    ;;
                *)
                    ((info++))
                    ;;
            esac
        fi
    done <<< "$lines"
    
    echo "info:$info warning:$warning error:$error critical:$critical"
}

# Helper: get counts by pane
_status_get_counts_by_pane() {
    local lines
    lines=$(storage_list_notifications "active")
    
    declare -A pane_counts
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            local pane_key="${session}:${window}:${pane}"
            pane_counts["$pane_key"]=$(( ${pane_counts["$pane_key"]:-0} + 1 ))
        fi
    done <<< "$lines"
    
    for pane_key in "${!pane_counts[@]}"; do
        echo "$pane_key:${pane_counts[$pane_key]}"
    done
}

# Format: summary (default)
_format_summary() {
    local total
    total=$(_status_get_active_count)
    if [[ $total -eq 0 ]]; then
        echo "No active notifications"
        return
    fi
    
    local level_counts
    level_counts=$(_status_get_counts_by_level)
    local info warning error critical
    info=$(echo "$level_counts" | grep -o 'info:[0-9]*' | cut -d: -f2)
    warning=$(echo "$level_counts" | grep -o 'warning:[0-9]*' | cut -d: -f2)
    error=$(echo "$level_counts" | grep -o 'error:[0-9]*' | cut -d: -f2)
    critical=$(echo "$level_counts" | grep -o 'critical:[0-9]*' | cut -d: -f2)
    
    echo "Active notifications: $total"
    echo "  info: $info, warning: $warning, error: $error, critical: $critical"
}

# Format: levels only
_format_levels() {
    local level_counts
    level_counts=$(_status_get_counts_by_level)
    echo "$level_counts" | tr ' ' '\n'
}

# Format: panes only
_format_panes() {
    _status_get_counts_by_pane | while IFS= read -r line; do
        echo "$line"
    done
}

# Format: json (future)
_format_json() {
    error "JSON format not yet implemented"
    return 1
}

status_command() {
    local format="summary"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
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
tmux-intray status - Show notification status summary

USAGE:
    tmux-intray status [OPTIONS]

OPTIONS:
    --format=<format>    Output format: summary, levels, panes, json (default: summary)
    -h, --help           Show this help

EXAMPLES:
    tmux-intray status               # Show summary
    tmux-intray status --format=levels # Show counts by level
    tmux-intray status --format=panes  # Show counts by pane
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
        summary)
            _format_summary
            ;;
        levels)
            _format_levels
            ;;
        panes)
            _format_panes
            ;;
        json)
            _format_json
            ;;
        *)
            error "Unknown format: $format"
            return 1
            ;;
    esac
}