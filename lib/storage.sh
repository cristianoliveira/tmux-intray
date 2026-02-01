#!/usr/bin/env bash
# Storage library for tmux-intray
# Provides file-based TSV storage with flock locking

# Load core utilities
# shellcheck source=./colors.sh disable=SC1091
# The sourced file exists at runtime but ShellCheck can't resolve it due to relative path/context.
source "$(dirname "${BASH_SOURCE[0]}")/colors.sh"
# shellcheck source=./hooks.sh disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/hooks.sh"

# Default directories
TMUX_INTRAY_STATE_DIR="${TMUX_INTRAY_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/tmux-intray}"
TMUX_INTRAY_CONFIG_DIR="${TMUX_INTRAY_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/tmux-intray}"

# File paths
NOTIFICATIONS_FILE="$TMUX_INTRAY_STATE_DIR/notifications.tsv"
DISMISSED_FILE="$TMUX_INTRAY_STATE_DIR/dismissed.tsv"
LOCK_DIR="$TMUX_INTRAY_STATE_DIR/lock"

# Ensure storage directories exist
storage_init() {
    debug "Initializing storage directories..."
    mkdir -p "$TMUX_INTRAY_STATE_DIR"
    mkdir -p "$TMUX_INTRAY_CONFIG_DIR"

    # Create files if they don't exist
    touch "$NOTIFICATIONS_FILE"
    touch "$DISMISSED_FILE"
    # Lock directory will be created by _with_lock when needed

    # Initialize hooks subsystem
    debug "Initializing hooks subsystem..."
    hooks_init
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

# Internal helper: update tmux status option with active count
_update_tmux_status() {
    # Only update if tmux is running
    if ! tmux has-session 2>/dev/null; then
        return 0
    fi

    local count
    count=$(storage_get_active_count)
    tmux set -g @tmux_intray_active_count "$count" 2>/dev/null || true
}

# Internal helper: simple file locking using mkdir (atomic)
_with_lock() {
    local lock_dir="$1"
    shift
    local timeout=10 # seconds
    local start_time
    start_time=$(date +%s)

    # Try to create lock directory (atomic operation)
    while ! mkdir "$lock_dir" 2>/dev/null; do
        local current_time
        current_time=$(date +%s)
        if ((current_time - start_time > timeout)); then
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
        tac "$file" | awk -F'\t' '!seen[$1]++' | tac
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
    local id="$1" timestamp="$2" state="$3" session="$4" window="$5" pane="$6" message="$7"
    local pane_created="${8:-}" level="${9:-info}"
    printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" "$id" "$timestamp" "$state" "$session" "$window" "$pane" "$message" "$pane_created" "$level" >>"$NOTIFICATIONS_FILE"
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
    debug "Adding notification (level: $level)"
    storage_init

    # Generate ID
    local id
    id=$(_get_next_id)
    debug "Generated notification ID: $id"

    # Use provided timestamp or generate current UTC
    if [[ -z "$timestamp" ]]; then
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    fi
    debug "Using timestamp: $timestamp"

    # Escape message
    local escaped_message
    escaped_message=$(_escape_message "$message")
    debug "Message escaped (original length: ${#message}, escaped length: ${#escaped_message})"

    # Run pre-add hooks
    debug "Running pre-add hooks..."
    hooks_run "pre-add" \
        "NOTIFICATION_ID=$id" \
        "LEVEL=$level" \
        "MESSAGE=$message" \
        "ESCAPED_MESSAGE=$escaped_message" \
        "TIMESTAMP=$timestamp" \
        "SESSION=$session" \
        "WINDOW=$window" \
        "PANE=$pane" \
        "PANE_CREATED=$pane_created"

    # Check if pre-add hooks aborted
    local hooks_rc=$?
    if [[ $hooks_rc -ne 0 ]]; then
        error "Pre-add hook aborted"
        return 1
    fi
    debug "Pre-add hooks completed successfully"

    # Append to TSV file with lock
    debug "Acquiring lock to append notification line"
    _with_lock "$LOCK_DIR" _append_notification_line "$id" "$timestamp" "active" "$session" "$window" "$pane" "$escaped_message" "$pane_created" "$level"
    debug "Notification appended to storage"

    # Update tmux status option
    _update_tmux_status
    debug "Updated tmux status option"

    # Run post-add hooks
    debug "Running post-add hooks..."
    hooks_run "post-add" \
        "NOTIFICATION_ID=$id" \
        "LEVEL=$level" \
        "MESSAGE=$message" \
        "ESCAPED_MESSAGE=$escaped_message" \
        "TIMESTAMP=$timestamp" \
        "SESSION=$session" \
        "WINDOW=$window" \
        "PANE=$pane" \
        "PANE_CREATED=$pane_created"
    debug "Post-add hooks completed"

    echo "$id"
}

# List notifications with optional state and level filters
# Arguments: state_filter (active|dismissed|all) [level_filter]
# Returns: TSV lines (latest version per notification)
storage_list_notifications() {
    local state_filter="${1:-active}"
    local level_filter="${2:-}"

    debug "Listing notifications (state_filter: $state_filter, level_filter: ${level_filter:-none})"
    storage_init

    # Get latest version of each notification
    local latest_lines
    latest_lines=$(_with_lock "$LOCK_DIR" _get_latest_notifications "$NOTIFICATIONS_FILE")
    debug "Retrieved latest notifications (lines: $(echo "$latest_lines" | wc -l))"

    # Build awk filter expression
    local filter_expr=""
    case "$state_filter" in
    active)
        # shellcheck disable=SC2016
        # Awk expression uses single quotes to prevent variable expansion; intentional.
        filter_expr='$3 == "active"'
        ;;
    dismissed)
        # shellcheck disable=SC2016
        # Awk expression uses single quotes to prevent variable expansion; intentional.
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
    debug "Filter expression after state: ${filter_expr:-none}"

    if [[ -n "$level_filter" ]]; then
        if [[ -n "$filter_expr" ]]; then
            filter_expr="${filter_expr} && \$9 == \"$level_filter\""
        else
            filter_expr="\$9 == \"$level_filter\""
        fi
    fi
    debug "Final filter expression: ${filter_expr:-none}"

    if [[ -n "$filter_expr" ]]; then
        awk -F'\t' "$filter_expr" <<<"$latest_lines" || true
    else
        echo "$latest_lines"
    fi
}

# Dismiss a notification by ID
storage_dismiss_notification() {
    local id="$1"

    debug "Dismissing notification ID: $id"
    storage_init

    # Get latest line for this ID
    local line
    line=$(_with_lock "$LOCK_DIR" _get_latest_line_for_id "$id")
    debug "Retrieved latest line for ID $id"

    if [[ -z "$line" ]]; then
        error "Notification with ID $id not found"
        return 1
    fi

    # Check current state
    local timestamp state session window pane message pane_created level
    _parse_notification_line "$line" dummy timestamp state session window pane message pane_created level
    debug "Current state: $state, level: $level"

    if [[ "$state" == "dismissed" ]]; then
        error "Notification $id is already dismissed"
        return 1
    fi

    # Unescape message for hooks
    local unescaped_message
    unescaped_message=$(_unescape_message "$message")

    # Run pre-dismiss hooks
    debug "Running pre-dismiss hooks..."
    hooks_run "pre-dismiss" \
        "NOTIFICATION_ID=$id" \
        "LEVEL=$level" \
        "MESSAGE=$unescaped_message" \
        "TIMESTAMP=$timestamp" \
        "SESSION=$session" \
        "WINDOW=$window" \
        "PANE=$pane" \
        "PANE_CREATED=$pane_created"

    # Check if pre-dismiss hooks aborted
    local hooks_rc=$?
    if [[ $hooks_rc -ne 0 ]]; then
        error "Pre-dismiss hook aborted"
        return 1
    fi
    debug "Pre-dismiss hooks completed successfully"

    # Add dismissed version (preserve level)
    debug "Acquiring lock to append dismissed version"
    _with_lock "$LOCK_DIR" _append_notification_line "$id" "$timestamp" "dismissed" "$session" "$window" "$pane" "$message" "$pane_created" "$level"
    debug "Dismissed version appended"

    # Update tmux status option
    _update_tmux_status
    debug "Updated tmux status option"

    # Run post-dismiss hooks
    debug "Running post-dismiss hooks..."
    hooks_run "post-dismiss" \
        "NOTIFICATION_ID=$id" \
        "LEVEL=$level" \
        "MESSAGE=$unescaped_message" \
        "TIMESTAMP=$timestamp" \
        "SESSION=$session" \
        "WINDOW=$window" \
        "PANE=$pane" \
        "PANE_CREATED=$pane_created"
    debug "Post-dismiss hooks completed"
}

# Dismiss all active notifications
storage_dismiss_all() {
    debug "Dismissing all active notifications"
    storage_init

    # Get latest active notifications
    local active_lines
    active_lines=$(_with_lock "$LOCK_DIR" _get_latest_active_lines)
    debug "Retrieved active notifications (count: $(echo "$active_lines" | wc -l))"

    # Process each active notification
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            debug "Processing notification ID: $id"

            # Unescape message for hooks
            local unescaped_message
            unescaped_message=$(_unescape_message "$message")

            # Run pre-dismiss hooks
            debug "Running pre-dismiss hooks for ID $id"
            hooks_run "pre-dismiss" \
                "NOTIFICATION_ID=$id" \
                "LEVEL=$level" \
                "MESSAGE=$unescaped_message" \
                "TIMESTAMP=$timestamp" \
                "SESSION=$session" \
                "WINDOW=$window" \
                "PANE=$pane" \
                "PANE_CREATED=$pane_created"

            # Check if pre-dismiss hooks aborted
            local hooks_rc=$?
            if [[ $hooks_rc -ne 0 ]]; then
                error "Pre-dismiss hook aborted for notification $id"
                return 1
            fi
            debug "Pre-dismiss hooks completed for ID $id"

            # Add dismissed version (preserve level)
            debug "Appending dismissed version for ID $id"
            _with_lock "$LOCK_DIR" _append_notification_line "$id" "$timestamp" "dismissed" "$session" "$window" "$pane" "$message" "$pane_created" "$level"

            # Run post-dismiss hooks
            debug "Running post-dismiss hooks for ID $id"
            hooks_run "post-dismiss" \
                "NOTIFICATION_ID=$id" \
                "LEVEL=$level" \
                "MESSAGE=$unescaped_message" \
                "TIMESTAMP=$timestamp" \
                "SESSION=$session" \
                "WINDOW=$window" \
                "PANE=$pane" \
                "PANE_CREATED=$pane_created"
        fi
    done <<<"$active_lines"
    debug "All active notifications dismissed"

    # Update tmux status option
    _update_tmux_status
    debug "Updated tmux status option"
}

# Clean up old dismissed notifications
# Arguments: days_threshold [dry_run]
storage_cleanup_old_notifications() {
    local days_threshold="$1"
    local dry_run="${2:-false}"

    debug "Cleaning up notifications older than $days_threshold days (dry_run: $dry_run)"
    storage_init

    # Calculate cutoff timestamp (UTC)
    local cutoff_timestamp
    cutoff_timestamp=$(date -u -d "$days_threshold days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v-"${days_threshold}"d +"%Y-%m-%dT%H:%M:%SZ")
    debug "Cutoff timestamp: $cutoff_timestamp"

    info "Cleaning up notifications dismissed before $cutoff_timestamp"

    # Run pre-cleanup hooks
    hooks_run "cleanup" \
        "CLEANUP_DAYS=$days_threshold" \
        "CUTOFF_TIMESTAMP=$cutoff_timestamp" \
        "DRY_RUN=$dry_run"

    # Get latest version of each notification
    local latest_lines
    latest_lines=$(_with_lock "$LOCK_DIR" _get_latest_notifications "$NOTIFICATIONS_FILE")
    debug "Retrieved latest notifications (lines: $(echo "$latest_lines" | wc -l))"

    # Collect IDs of dismissed notifications older than cutoff
    local ids_to_delete=()
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            if [[ "$state" == "dismissed" ]] && [[ "$timestamp" < "$cutoff_timestamp" ]]; then
                ids_to_delete+=("$id")
            fi
        fi
    done <<<"$latest_lines"

    local deleted_count=${#ids_to_delete[@]}
    debug "Found $deleted_count notification(s) to delete"

    if [[ $deleted_count -eq 0 ]]; then
        info "No old dismissed notifications to clean up"
        # Run post-cleanup hooks with zero count
        hooks_run "post-cleanup" \
            "CLEANUP_DAYS=$days_threshold" \
            "CUTOFF_TIMESTAMP=$cutoff_timestamp" \
            "DELETED_COUNT=0"
        return 0
    fi

    info "Found $deleted_count notification(s) to clean up"

    if [[ "$dry_run" == true ]]; then
        info "Dry run: would delete notifications with IDs: ${ids_to_delete[*]}"
        # Run post-cleanup hooks with dry run
        hooks_run "post-cleanup" \
            "CLEANUP_DAYS=$days_threshold" \
            "CUTOFF_TIMESTAMP=$cutoff_timestamp" \
            "DRY_RUN=true" \
            "DELETED_COUNT=$deleted_count"
        return 0
    fi

    # Filter out all lines whose ID is in ids_to_delete
    debug "Deleting notifications with IDs: ${ids_to_delete[*]}"
    _with_lock "$LOCK_DIR" _filter_out_ids "$NOTIFICATIONS_FILE" "${ids_to_delete[@]}"
    debug "Successfully deleted $deleted_count notification(s)"

    # Run post-cleanup hooks
    hooks_run "post-cleanup" \
        "CLEANUP_DAYS=$days_threshold" \
        "CUTOFF_TIMESTAMP=$cutoff_timestamp" \
        "DELETED_COUNT=$deleted_count"

    info "Successfully cleaned up $deleted_count notification(s)"
}

# Internal helper: filter out lines with given IDs from notifications file
# Arguments: file id1 [id2 ...]
_filter_out_ids() {
    local file="$1"
    shift
    local ids=("$@")

    # Build awk pattern to exclude lines where first field matches any ID
    local pattern=""
    for id in "${ids[@]}"; do
        pattern="${pattern:+$pattern && }\$1 != \"$id\""
    done

    # Create temporary file
    local temp_file
    temp_file=$(mktemp)

    # Filter using awk
    awk -F'\t' "$pattern" "$file" >"$temp_file"

    # Replace original file
    mv "$temp_file" "$file"
}

# Get count of active notifications
storage_get_active_count() {
    debug "Getting active notification count"
    storage_list_notifications "active" | wc -l
}
