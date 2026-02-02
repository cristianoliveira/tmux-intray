#!/usr/bin/env bash
# List command - Display notifications with various filters and formats

# Source core libraries
COMMAND_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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

# Internal helper: get filtered notification lines
# Arguments: filter pane_filter level_filter session_filter window_filter older_than_cutoff newer_than_cutoff search_pattern search_regex
# Returns: filtered TSV lines (latest version per notification)
_get_filtered_lines() {
    local filter="$1"
    local pane_filter="$2"
    local level_filter="$3"
    local session_filter="$4"
    local window_filter="$5"
    local older_than_cutoff="$6"
    local newer_than_cutoff="$7"
    local search_pattern="$8"
    local search_regex="$9"

    local lines
    lines=$(storage_list_notifications "$filter" "$level_filter" "$session_filter" "$window_filter" "$pane_filter" "$older_than_cutoff" "$newer_than_cutoff")

    # Apply search filter if specified
    if [[ -n "$search_pattern" ]]; then
        local filtered_lines=""
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                local message
                _parse_notification_line "$line" _ _ _ _ _ _ message _ _
                message=$(_unescape_message "$message")
                if [[ "$search_regex" == true ]]; then
                    if [[ "$message" =~ $search_pattern ]]; then
                        filtered_lines="${filtered_lines}${line}\n"
                    fi
                else
                    if [[ "$message" == *"$search_pattern"* ]]; then
                        filtered_lines="${filtered_lines}${line}\n"
                    fi
                fi
            fi
        done <<<"$lines"
        lines=$(echo -e "$filtered_lines" | sed '/^$/d')
    fi

    echo "$lines"
}

# Default format function (legacy compatibility)
_format_legacy() {
    local filter="${1:-active}"
    local pane_filter="${2:-}"
    local level_filter="${3:-}"
    local session_filter="${4:-}"
    local window_filter="${5:-}"
    local older_than_cutoff="${6:-}"
    local newer_than_cutoff="${7:-}"
    local search_pattern="${8:-}"
    local search_regex="${9:-}"
    local group_by="${10:-}"
    local group_count="${11:-false}"

    local lines
    lines=$(_get_filtered_lines "$filter" "$pane_filter" "$level_filter" "$session_filter" "$window_filter" "$older_than_cutoff" "$newer_than_cutoff" "$search_pattern" "$search_regex")

    if [[ -z "$lines" ]]; then
        info "Tray is empty"
        return
    fi

    # Apply grouping if specified (legacy format doesn't support grouping, fall back to compact grouping)
    if [[ -n "$group_by" ]]; then
        _format_compact_with_grouping "$lines" "$group_by" "$group_count"
        return
    fi

    # Items are just messages (legacy compatibility)
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local message
            _parse_notification_line "$line" _ _ _ _ _ _ message _ _
            message=$(_unescape_message "$message")
            echo "$message"
        fi
    done <<<"$lines"
}

# Simple table format
_format_table() {
    local filter="${1:-active}"
    local pane_filter="${2:-}"
    local level_filter="${3:-}"
    local session_filter="${4:-}"
    local window_filter="${5:-}"
    local older_than_cutoff="${6:-}"
    local newer_than_cutoff="${7:-}"
    local search_pattern="${8:-}"
    local search_regex="${9:-}"
    local group_by="${10:-}"
    local group_count="${11:-false}"

    local lines
    lines=$(_get_filtered_lines "$filter" "$pane_filter" "$level_filter" "$session_filter" "$window_filter" "$older_than_cutoff" "$newer_than_cutoff" "$search_pattern" "$search_regex")

    if [[ -z "$lines" ]]; then
        info "No notifications found"
        return
    fi

    # Apply grouping if specified
    if [[ -n "$group_by" ]]; then
        _format_table_with_grouping "$lines" "$group_by" "$group_count"
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
    done <<<"$lines"
}

# Helper: format table with grouping
_format_table_with_grouping() {
    local lines="$1"
    local group_by="$2"
    local group_count="$3"

    # Determine field index based on group_by
    local field_index
    case "$group_by" in
    session) field_index=4 ;;
    window) field_index=5 ;;
    pane) field_index=6 ;;
    level) field_index=9 ;;
    *)
        error "Invalid group-by field: $group_by"
        return 1
        ;;
    esac

    # Use awk to group lines by field
    local grouped_output
    grouped_output=$(awk -F'\t' -v field="$field_index" -v count_only="$group_count" '
        {
            group = $field
            groups[group] = groups[group] $0 "\n"
            counts[group]++
        }
        END {
            for (group in groups) {
                if (count_only == "true") {
                    printf "Group: %s (%d)\n", group, counts[group]
                } else {
                    printf "=== %s (%d) ===\n", group, counts[group]
                    printf "%s", groups[group]
                }
            }
        }
    ' <<<"$lines")

    # Now output grouped lines with table formatting
    while IFS= read -r line; do
        if [[ -z "$line" ]]; then
            continue
        fi
        # Check if line is a group header (starts with "===" or "Group:")
        if [[ "$line" == "==="* ]] || [[ "$line" == "Group:"* ]]; then
            echo "$line"
        else
            # It's a TSV line, parse and format as table row
            local id timestamp pane level message
            _parse_notification_line "$line" id timestamp _ _ _ pane message _ level
            message=$(_unescape_message "$message")
            local display_msg
            if [[ ${#message} -gt 35 ]]; then
                display_msg="${message:0:32}..."
            else
                display_msg="$message"
            fi
            printf "%-4s  %-25s  %-6s  %-6s  %s\n" "$id" "$timestamp" "$pane" "$level" "$display_msg"
        fi
    done <<<"$grouped_output"
}

# Compact format (just messages)
_format_compact() {
    local filter="${1:-active}"
    local pane_filter="${2:-}"
    local level_filter="${3:-}"
    local session_filter="${4:-}"
    local window_filter="${5:-}"
    local older_than_cutoff="${6:-}"
    local newer_than_cutoff="${7:-}"
    local search_pattern="${8:-}"
    local search_regex="${9:-}"
    local group_by="${10:-}"
    local group_count="${11:-false}"

    local lines
    lines=$(_get_filtered_lines "$filter" "$pane_filter" "$level_filter" "$session_filter" "$window_filter" "$older_than_cutoff" "$newer_than_cutoff" "$search_pattern" "$search_regex")

    if [[ -z "$lines" ]]; then
        return
    fi

    # Apply grouping if specified
    if [[ -n "$group_by" ]]; then
        _format_compact_with_grouping "$lines" "$group_by" "$group_count"
        return
    fi

    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local message
            _parse_notification_line "$line" _ _ _ _ _ _ message _ _
            message=$(_unescape_message "$message")
            echo "$message"
        fi
    done <<<"$lines"
}

# Helper: format compact with grouping
_format_compact_with_grouping() {
    local lines="$1"
    local group_by="$2"
    local group_count="$3"

    local field_index
    case "$group_by" in
    session) field_index=4 ;;
    window) field_index=5 ;;
    pane) field_index=6 ;;
    level) field_index=9 ;;
    *)
        error "Invalid group-by field: $group_by"
        return 1
        ;;
    esac

    awk -F'\t' -v field="$field_index" -v count_only="$group_count" '
        {
            group = $field
            groups[group] = groups[group] $0 "\n"
            counts[group]++
        }
        END {
            for (group in groups) {
                if (count_only == "true") {
                    printf "Group: %s (%d)\n", group, counts[group]
                } else {
                    printf "=== %s (%d) ===\n", group, counts[group]
                    # Print each line in group
                    split(groups[group], group_lines, "\n")
                    for (i in group_lines) {
                        if (group_lines[i] == "") continue
                        # Extract message field (field 7)
                        split(group_lines[i], fields, "\t")
                        message = fields[7]
                        # Unescape message (simplified: replace \\n with newline etc.)
                        gsub(/\\\\/, "\\", message)
                        gsub(/\\t/, "\t", message)
                        gsub(/\\n/, "\n", message)
                        printf "%s\n", message
                    }
                }
            }
        }
    ' <<<"$lines"
}

list_command() {
    local filter="active"
    local format="legacy"
    local pane_filter=""
    local level_filter=""
    local session_filter=""
    local window_filter=""
    local older_than_days=""
    local newer_than_days=""
    local older_than_cutoff=""
    local newer_than_cutoff=""
    local search_pattern=""
    local search_regex=false
    local group_by=""
    local group_count=false

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
        --session=*)
            session_filter="${1#*=}"
            shift
            ;;
        --session)
            session_filter="$2"
            shift 2
            ;;
        --window=*)
            window_filter="${1#*=}"
            shift
            ;;
        --window)
            window_filter="$2"
            shift 2
            ;;
        --older-than=*)
            older_than_days="${1#*=}"
            shift
            ;;
        --older-than)
            older_than_days="$2"
            shift 2
            ;;
        --newer-than=*)
            newer_than_days="${1#*=}"
            shift
            ;;
        --newer-than)
            newer_than_days="$2"
            shift 2
            ;;
        --search=*)
            search_pattern="${1#*=}"
            shift
            ;;
        --search)
            search_pattern="$2"
            shift 2
            ;;
        --regex)
            search_regex=true
            shift
            ;;
        --group-by=*)
            group_by="${1#*=}"
            shift
            ;;
        --group-by)
            group_by="$2"
            shift 2
            ;;
        --group-count)
            group_count=true
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
        --help | -h)
            cat <<EOF
tmux-intray list - List notifications

USAGE:
    tmux-intray list [OPTIONS]

OPTIONS:
    --active             Show active notifications (default)
    --dismissed          Show dismissed notifications
    --all                Show all notifications
    --pane <id>          Filter notifications by pane ID (e.g., %0)
    --level <level>      Filter notifications by level: info, warning, error, critical
    --session <id>       Filter notifications by session ID
    --window <id>        Filter notifications by window ID
    --older-than <days>  Show notifications older than N days
    --newer-than <days>  Show notifications newer than N days
    --search <pattern>   Search messages (substring match)
    --regex              Use regex search with --search
    --group-by <field>   Group notifications by field (session, window, pane, level)
    --group-count        Show only group counts (requires --group-by)
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

    # Compute cutoff timestamps if needed
    if [[ -n "$older_than_days" ]]; then
        older_than_cutoff=$(date -u -d "$older_than_days days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v-"${older_than_days}"d +"%Y-%m-%dT%H:%M:%SZ")
    fi
    if [[ -n "$newer_than_days" ]]; then
        newer_than_cutoff=$(date -u -d "$newer_than_days days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v-"${newer_than_days}"d +"%Y-%m-%dT%H:%M:%SZ")
    fi

    # Validate group-by field
    if [[ -n "$group_by" ]]; then
        case "$group_by" in
        session | window | pane | level)
            # valid
            ;;
        *)
            error "Invalid group-by field: $group_by (must be session, window, pane, level)"
            return 1
            ;;
        esac
    fi

    case "$format" in
    legacy)
        # For backward compatibility with scripts using old show command
        _format_legacy "$filter" "$pane_filter" "$level_filter" "$session_filter" "$window_filter" "$older_than_cutoff" "$newer_than_cutoff" "$search_pattern" "$search_regex" "$group_by" "$group_count"
        ;;
    table)
        _format_table "$filter" "$pane_filter" "$level_filter" "$session_filter" "$window_filter" "$older_than_cutoff" "$newer_than_cutoff" "$search_pattern" "$search_regex" "$group_by" "$group_count"
        ;;
    compact)
        _format_compact "$filter" "$pane_filter" "$level_filter" "$session_filter" "$window_filter" "$older_than_cutoff" "$newer_than_cutoff" "$search_pattern" "$search_regex" "$group_by" "$group_count"
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
