#!/usr/bin/env bash
# Status panel script for tmux status bar
# Outputs formatted notification count with optional level colors

# Source core libraries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
# shellcheck source=../lib/colors.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$PROJECT_ROOT/lib/colors.sh"
# shellcheck source=../lib/core.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$(dirname "${BASH_SOURCE[0]}")/../lib/core.sh"

# Default configuration
TMUX_INTRAY_STATUS_FORMAT="${TMUX_INTRAY_STATUS_FORMAT:-compact}"
TMUX_INTRAY_STATUS_ENABLED="${TMUX_INTRAY_STATUS_ENABLED:-1}"
TMUX_INTRAY_LEVEL_COLORS="${TMUX_INTRAY_LEVEL_COLORS:-info:green,warning:yellow,error:red,critical:magenta}"
TMUX_INTRAY_SHOW_LEVELS="${TMUX_INTRAY_SHOW_LEVELS:-0}"

# Parse command line arguments
parse_args() {
    local format="$TMUX_INTRAY_STATUS_FORMAT"
    local enabled="$TMUX_INTRAY_STATUS_ENABLED"

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
        --enabled=*)
            enabled="${1#*=}"
            shift
            ;;
        --enabled)
            enabled="$2"
            shift 2
            ;;
        --help | -h)
            cat <<EOF
tmux-intray status-panel - Status bar indicator script

USAGE:
    tmux-intray status-panel [OPTIONS]

OPTIONS:
    --format=<format>    Output format: compact, detailed, count-only (default: compact)
    --enabled=<0|1>      Enable/disable status indicator (default: 1)
    -h, --help           Show this help

DESCRIPTION:
    This script is designed to be used in tmux status-right configuration.
    Example: set -g status-right "#(tmux-intray status-panel) %H:%M"

    The script outputs a formatted string showing notification counts.
    When clicked, it can trigger the list command (via tmux bindings).
EOF
            return 1
            ;;
        *)
            error "Unknown argument: $1"
            exit 1
            ;;
        esac
    done

    echo "$format" "$enabled"
}

# Get active notification count
get_active_count() {
    storage_get_active_count
}

# Get counts by level
get_counts_by_level() {
    local lines
    lines=$(storage_list_notifications "active")

    local info=0 warning=0 error=0 critical=0
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local level
            _parse_notification_line "$line" _ _ _ _ _ _ _ _ level
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
    done <<<"$lines"

    echo "$info $warning $error $critical"
}

# Get color for level
get_level_color() {
    local level="$1"
    local color=""
    # Parse TMUX_INTRAY_LEVEL_COLORS
    IFS=',' read -ra pairs <<<"$TMUX_INTRAY_LEVEL_COLORS"
    for pair in "${pairs[@]}"; do
        if [[ "$pair" == "$level:"* ]]; then
            color="${pair#*:}"
            break
        fi
    done
    echo "$color"
}

# Format: compact (icon + count)
format_compact() {
    local total="$1"
    local info="$2" warning="$3" error="$4" critical="$5"

    if [[ $total -eq 0 ]]; then
        echo ""
        return
    fi

    # Determine highest severity level present
    local highest_level="info"
    if [[ $critical -gt 0 ]]; then
        highest_level="critical"
    elif [[ $error -gt 0 ]]; then
        highest_level="error"
    elif [[ $warning -gt 0 ]]; then
        highest_level="warning"
    fi

    local color
    color=$(get_level_color "$highest_level")
    local icon="🔔"

    if [[ -n "$color" ]]; then
        # Use tmux color codes
        echo "#[fg=$color]$icon $total#[default]"
    else
        echo "$icon $total"
    fi
}

# Format: detailed (counts per level)
format_detailed() {
    local total="$1"
    local info="$2" warning="$3" error="$4" critical="$5"

    if [[ $total -eq 0 ]]; then
        echo ""
        return
    fi

    local output=""
    if [[ $info -gt 0 ]]; then
        local color
        color=$(get_level_color "info")
        if [[ -n "$color" ]]; then
            output+="#[fg=$color]i:$info#[default] "
        else
            output+="i:$info "
        fi
    fi
    if [[ $warning -gt 0 ]]; then
        color=$(get_level_color "warning")
        if [[ -n "$color" ]]; then
            output+="#[fg=$color]w:$warning#[default] "
        else
            output+="w:$warning "
        fi
    fi
    if [[ $error -gt 0 ]]; then
        color=$(get_level_color "error")
        if [[ -n "$color" ]]; then
            output+="#[fg=$color]e:$error#[default] "
        else
            output+="e:$error "
        fi
    fi
    if [[ $critical -gt 0 ]]; then
        color=$(get_level_color "critical")
        if [[ -n "$color" ]]; then
            output+="#[fg=$color]c:$critical#[default] "
        else
            output+="c:$critical "
        fi
    fi
    # Trim trailing space
    echo "${output% }"
}

# Format: count-only
format_count_only() {
    local total="$1"
    if [[ $total -eq 0 ]]; then
        echo ""
    else
        echo "$total"
    fi
}

main() {
    # Parse arguments
    local output
    if ! output=$(parse_args "$@"); then
        # parse_args returned non-zero, which indicates help printed
        echo -n "$output"
        exit 0
    fi
    read -r format enabled <<<"$output"

    # Check if status is enabled
    if [[ "$enabled" != "1" ]]; then
        exit 0
    fi

    # Ensure tmux is running (silently fail if not)
    if ! tmux has-session 2>/dev/null; then
        exit 0
    fi

    # Get counts
    local total
    total=$(get_active_count)
    read -r info warning error critical < <(get_counts_by_level)

    # Format output
    case "$format" in
    compact)
        format_compact "$total" "$info" "$warning" "$error" "$critical"
        ;;
    detailed)
        format_detailed "$total" "$info" "$warning" "$error" "$critical"
        ;;
    count-only)
        format_count_only "$total"
        ;;
    *)
        error "Unknown format: $format"
        exit 1
        ;;
    esac
}

# Command function for CLI
status_panel_command() {
    main "$@"
}

# Run main only if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
