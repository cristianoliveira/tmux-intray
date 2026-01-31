#!/usr/bin/env bats
# Test storage library functionality

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-storage"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME

    # Clean up any existing server
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1

    # Start a tmux server for migration tests
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
    # Get socket path and set TMUX environment variable so plain tmux commands use our test server
    socket_path=$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{socket_path}' 2>/dev/null)
    # TMUX format: socket_path,client_fd,client_pid
    # We'll fake client_fd and client_pid (not critical for our use)
    export TMUX="$socket_path,12345,0"
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
}

@test "storage_init creates directories" {
    source ./lib/storage.sh

    storage_init

    [ -d "$TMUX_INTRAY_STATE_DIR" ]
    [ -d "$TMUX_INTRAY_CONFIG_DIR" ]
    [ -f "$NOTIFICATIONS_FILE" ]
    [ -f "$DISMISSED_FILE" ]
}

@test "storage_add_notification adds entry" {
    source ./lib/storage.sh

    local id
    id=$(storage_add_notification "Test message")

    [ -n "$id" ]
    [ "$id" -eq 1 ]

    # Verify file contains entry
    local line_count
    line_count=$(wc -l <"$NOTIFICATIONS_FILE")
    [ "$line_count" -eq 1 ]

    # Verify fields
    local line
    line=$(head -n 1 "$NOTIFICATIONS_FILE")
    local id_field timestamp state session window pane message pane_created level
    _parse_notification_line "$line" id_field timestamp state session window pane message pane_created level

    [ "$id_field" -eq 1 ]
    [[ "$timestamp" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]
    [ "$state" = "active" ]
    [ -z "$session" ]
    [ -z "$window" ]
    [ -z "$pane" ]
    # Message is escaped, but should contain original
    [[ "$message" == *"Test message"* ]]
}

@test "storage_list_notifications filters by state" {
    source ./lib/storage.sh

    storage_add_notification "Active 1"
    storage_add_notification "Active 2"

    # Dismiss first notification
    storage_dismiss_notification 1

    local active_count dismissed_count all_count
    active_count=$(storage_list_notifications "active" | wc -l)
    dismissed_count=$(storage_list_notifications "dismissed" | wc -l)
    all_count=$(storage_list_notifications "all" | wc -l)

    [ "$active_count" -eq 1 ]
    [ "$dismissed_count" -eq 1 ]
    [ "$all_count" -eq 2 ] # Latest version per notification (id1 dismissed, id2 active)
}

@test "storage_dismiss_notification updates state" {
    source ./lib/storage.sh

    storage_add_notification "Test"

    storage_dismiss_notification 1

    local line
    line=$(storage_list_notifications "dismissed")
    [ -n "$line" ]

    IFS=$'\t' read -r id timestamp state _ _ _ _ _ _ <<<"$line"
    [ "$state" = "dismissed" ]
}

@test "storage_dismiss_all dismisses all active" {
    source ./lib/storage.sh

    storage_add_notification "Test 1"
    storage_add_notification "Test 2"

    storage_dismiss_all

    local active_count
    active_count=$(storage_get_active_count)

    [ "$active_count" -eq 0 ]
}

@test "storage_get_active_count returns correct number" {
    source ./lib/storage.sh

    [ "$(storage_get_active_count)" -eq 0 ]

    storage_add_notification "Test 1"
    [ "$(storage_get_active_count)" -eq 1 ]

    storage_add_notification "Test 2"
    [ "$(storage_get_active_count)" -eq 2 ]

    storage_dismiss_notification 1
    [ "$(storage_get_active_count)" -eq 1 ]
}

@test "escape and unescape roundtrip" {
    source ./lib/storage.sh

    local original="Test	tab
newline"
    local escaped
    escaped=$(_escape_message "$original")

    # Should not contain actual tabs or newlines
    [[ ! "$escaped" =~ $'\t' ]]
    [[ ! "$escaped" =~ $'\n' ]]

    local unescaped
    unescaped=$(_unescape_message "$escaped")

    [ "$unescaped" = "$original" ]
}
