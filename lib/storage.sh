#!/usr/bin/env bash
# Storage library for tmux-intray
# Provides file-based TSV storage with flock locking

# Load core utilities
# shellcheck source=./colors.sh
source "$(dirname "${BASH_SOURCE[0]}")/colors.sh"

# Default directories
TMUX_INTRAY_STATE_DIR="${TMUX_INTRAY_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/tmux-intray}"
TMUX_INTRAY_CONFIG_DIR="${TMUX_INTRAY_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/tmux-intray}"

# File paths
NOTIFICATIONS_FILE="$TMUX_INTRAY_STATE_DIR/notifications.tsv"
DISMISSED_FILE="$TMUX_INTRAY_STATE_DIR/dismissed.tsv"
LOCK_DIR="$TMUX_INTRAY_STATE_DIR/lock"

# Ensure storage directories exist
storage_init() {
    mkdir -p "$TMUX_INTRAY_STATE_DIR"
    mkdir -p "$TMUX_INTRAY_CONFIG_DIR"
    
    # Create files if they don't exist
    touch "$NOTIFICATIONS_FILE"
    touch "$DISMISSED_FILE"
    # Lock directory will be created by _with_lock when needed
}

# Internal helper: escape tabs and newlines in message field
_escape_message() {
    local msg="$1"
    # Escape backslashes first, then tabs, then newlines
    # Using bash parameter expansion for safety
    msg="${msg//\\/\\\\}"
    msg="${msg//$'\t'/\\t}"
    msg="${msg//$'\n'/\\n}"
    echo "$msg"
}

# Internal helper: unescape message field
_unescape_message() {
    local msg="$1"
    # Unescape in reverse order
    msg="${msg//\\n/$'\n'}"
    msg="${msg//\\t/$'\t'}"
    msg="${msg//\\\\/\\}"
    echo "$msg"
}

# Internal helper: simple file locking using mkdir (atomic)
_with_lock() {
    local lock_dir="$1"
    shift
    local timeout=10  # seconds
    local start_time
    start_time=$(date +%s)
    
    # Try to create lock directory (atomic operation)
    while ! mkdir "$lock_dir" 2>/dev/null; do
        local current_time
        current_time=$(date +%s)
        if (( current_time - start_time > timeout )); then
            error "Timeout acquiring lock on $lock_dir"
            return 1
        fi
        sleep 0.1
    done
    
    # Execute command with lock held
    (
        # Ensure lock is removed on exit
        trap 'rmdir "$lock_dir" 2>/dev/null || true' EXIT
        "$@"
    )
}

# Internal helper: get next notification ID
_get_next_id() {
    local last_id=0
    if [[ -f "$NOTIFICATIONS_FILE" && -s "$NOTIFICATIONS_FILE" ]]; then
        # Get last line, extract first field (ID)
        last_id=$(tail -n 1 "$NOTIFICATIONS_FILE" | cut -f1)
        # Ensure numeric
        if ! [[ "$last_id" =~ ^[0-9]+$ ]]; then
            last_id=0
        fi
    fi
    echo $((last_id + 1))
}

# Internal helper: get latest version of each notification (last line per ID)
_get_latest_notifications() {
    local file="$1"
    # Use awk to keep only the last occurrence of each ID
    # Since file is append-only, we can reverse, keep first occurrence, reverse back
    if [[ -f "$file" && -s "$file" ]]; then
        tac "$file" | awk -F '\t' '!seen[$1]++' | tac
    fi
}

# Internal helper: get latest line for a specific notification ID
_get_latest_line_for_id() {
    local id="$1"
    _get_latest_notifications "$NOTIFICATIONS_FILE" | awk -F'\t' -v id="$id" '$1 == id' || true
}

# Internal helper: get latest active notifications
_get_latest_active_lines() {
    _get_latest_notifications "$NOTIFICATIONS_FILE" | awk -F'\t' '$3 == "active"' || true
}

# Internal helper: append a notification line to file
_append_notification_line() {
    local id="$1" timestamp="$2" state="$3" session="$4" window="$5" pane="$6" message="$7" pane_created="${8:-}" level="${9:-info}"
    printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" "$id" "$timestamp" "$state" "$session" "$window" "$pane" "$message" "$pane_created" "$level" >> "$NOTIFICATIONS_FILE"
}

# Internal helper: parse notification line into variables
# Usage: _parse_notification_line line id_var timestamp_var state_var session_var window_var pane_var message_var [pane_created_var] [level_var]
_parse_notification_line() {
    local line="$1"
    shift
    local id_var="$1" timestamp_var="$2" state_var="$3" session_var="$4" window_var="$5" pane_var="$6" message_var="$7"
    local pane_created_var="${8:-}"
    local level_var="${9:-}"
    
    # Use awk to split by tab and assign to named variables via eval
    # Read into temporary array - up to 9 fields
    local -a fields
    mapfile -t fields < <(echo "$line" | awk -F'\t' '{
        for(i=1;i<=9;i++) print $i
    }')
    
    # Ensure we have at least 9 fields (pad with empty)
    while [[ ${#fields[@]} -lt 9 ]]; do
        fields+=("")
    done
    
    printf -v "$id_var" "%s" "${fields[0]}"
    printf -v "$timestamp_var" "%s" "${fields[1]}"
    printf -v "$state_var" "%s" "${fields[2]}"
    printf -v "$session_var" "%s" "${fields[3]}"
    printf -v "$window_var" "%s" "${fields[4]}"
    printf -v "$pane_var" "%s" "${fields[5]}"
    printf -v "$message_var" "%s" "${fields[6]}"
    if [[ -n "$pane_created_var" ]]; then
        printf -v "$pane_created_var" "%s" "${fields[7]}"
    fi
    if [[ -n "$level_var" ]]; then
        # If field 8 is empty, default to "info"
        printf -v "$level_var" "%s" "${fields[8]:-info}"
    fi
}

# Add a notification to storage
# Arguments: message [timestamp] [session] [window] [pane] [pane_created] [level]
# Returns: notification ID
storage_add_notification() {
    local message="$1"
    local timestamp="${2:-}"
    local session="${3:-}"
    local window="${4:-}"
    local pane="${5:-}"
    local pane_created="${6:-}"
    local level="${7:-info}"
    
    # Ensure storage initialized
    storage_init
    
    # Generate ID
    local id
    id=$(_get_next_id)
    
    # Use provided timestamp or generate current UTC
    if [[ -z "$timestamp" ]]; then
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    fi
    
    # Escape message
    local escaped_message
    escaped_message=$(_escape_message "$message")
    
    # Append to TSV file with lock
    _with_lock "$LOCK_DIR" _append_notification_line "$id" "$timestamp" "active" "$session" "$window" "$pane" "$escaped_message" "$pane_created" "$level"
    
    echo "$id"
}

# List notifications with optional state and level filters
# Arguments: state_filter (active|dismissed|all) [level_filter]
# Returns: TSV lines (latest version per notification)
storage_list_notifications() {
    local state_filter="${1:-active}"
    local level_filter="${2:-}"
    
    storage_init
    
    # Get latest version of each notification
    local latest_lines
    latest_lines=$(_with_lock "$LOCK_DIR" _get_latest_notifications "$NOTIFICATIONS_FILE")
    
    # Build awk filter expression
    local filter_expr=""
    case "$state_filter" in
        active)
            filter_expr='$3 == "active"'
            ;;
        dismissed)
            filter_expr='$3 == "dismissed"'
            ;;
        all)
            # No state filter
            ;;
        *)
            error "Invalid state filter: $state_filter"
            return 1
            ;;
    esac
    
    if [[ -n "$level_filter" ]]; then
        if [[ -n "$filter_expr" ]]; then
            filter_expr="${filter_expr} && \$9 == \"$level_filter\""
        else
            filter_expr="\$9 == \"$level_filter\""
        fi
    fi
    
    if [[ -n "$filter_expr" ]]; then
        awk -F'\t' "$filter_expr" <<< "$latest_lines" || true
    else
        echo "$latest_lines"
    fi
}

# Dismiss a notification by ID
storage_dismiss_notification() {
    local id="$1"
    
    storage_init
    
    # Get latest line for this ID
    local line
    line=$(_with_lock "$LOCK_DIR" _get_latest_line_for_id "$id")
    
    if [[ -z "$line" ]]; then
        error "Notification with ID $id not found"
        return 1
    fi
    
    # Check current state
    local timestamp state session window pane message pane_created level
    _parse_notification_line "$line" dummy timestamp state session window pane message pane_created level

    if [[ "$state" == "dismissed" ]]; then
        error "Notification $id is already dismissed"
        return 1
    fi
    
    # Add dismissed version (preserve level)
    _with_lock "$LOCK_DIR" _append_notification_line "$id" "$timestamp" "dismissed" "$session" "$window" "$pane" "$message" "$pane_created" "$level"
}

# Dismiss all active notifications
storage_dismiss_all() {
    storage_init
    
    # Get latest active notifications
    local active_lines
    active_lines=$(_with_lock "$LOCK_DIR" _get_latest_active_lines)
    
    # Add dismissed version for each
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            _with_lock "$LOCK_DIR" _append_notification_line "$id" "$timestamp" "dismissed" "$session" "$window" "$pane" "$message" "$pane_created" "$level"
        fi
    done <<< "$active_lines"
}

# Get count of active notifications
storage_get_active_count() {
    storage_list_notifications "active" | wc -l
}

# Migrate from environment variables to file storage
storage_migrate_from_env() {
    # Check if TMUX_INTRAY_ITEMS exists and has content
    local env_items
    env_items=$(tmux show-environment -g TMUX_INTRAY_ITEMS 2>/dev/null || echo "")
    # Format: TMUX_INTRAY_ITEMS=item1:item2:item3
    env_items="${env_items#TMUX_INTRAY_ITEMS=}"
    
    if [[ -z "$env_items" ]]; then
        return 0  # Nothing to migrate
    fi
    
    # Split by colon
    local migrated=0
    IFS=':' read -ra items <<< "$env_items"
    
    for item in "${items[@]}"; do
        if [[ -n "$item" ]]; then
            # Parse existing formatted message: [timestamp] message
            # Extract timestamp between brackets at start
            local timestamp message
            if [[ "$item" =~ ^\[([^]]+)\]\ (.*)$ ]]; then
                timestamp="${BASH_REMATCH[1]}"
                message="${BASH_REMATCH[2]}"
                # Convert timestamp to ISO 8601 UTC if possible
                # For now keep as is, but convert to ISO format
                # We'll parse with date command
                local iso_timestamp
                if iso_timestamp=$(date -u -d "$timestamp" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null); then
                    timestamp="$iso_timestamp"
                else
                    # Fallback: use current time
                    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
                fi
            else
                # No timestamp, use current time
                timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
                message="$item"
            fi
            
            # Add to storage with original timestamp (default level: info)
            storage_add_notification "$message" "$timestamp" "" "" "" "" "info"
            ((migrated++))
        fi
    done
    
    # Clear environment variable after migration
    tmux set-environment -g TMUX_INTRAY_ITEMS ""
    
    info "Migrated $migrated items from environment variables to file storage"
}