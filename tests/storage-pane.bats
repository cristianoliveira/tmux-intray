#!/usr/bin/env bats
# Test storage library with pane associations

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-pane"
    export XDG_STATE_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME="$(mktemp -d)"
    
    # Clean up any existing server
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    
    # Start a tmux server for migration tests
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
}

@test "storage_add_notification with pane association" {
    source ./lib/storage.sh
    
    local id
    id=$(storage_add_notification "Test message" "" "\$1" "@2" "%3" "1234567890")
    
    [ -n "$id" ]
    
    # Verify fields
    local line
    line=$(tail -n 1 "$NOTIFICATIONS_FILE")
    IFS=$'\t' read -r id_field timestamp state session window pane message pane_created level <<< "$line"
    
    [ "$id_field" -eq 1 ]
    [ "$state" = "active" ]
    [ "$session" = "\$1" ]
    [ "$window" = "@2" ]
    [ "$pane" = "%3" ]
    [ "$pane_created" = "1234567890" ]
    [[ "$message" == *"Test message"* ]]
}

@test "storage_add_notification with empty pane association" {
    source ./lib/storage.sh
    
    storage_add_notification "Test message"
    
    local line
    line=$(tail -n 1 "$NOTIFICATIONS_FILE")
    IFS=$'\t' read -r id_field timestamp state session window pane message pane_created level <<< "$line"
    
    [ -z "$session" ]
    [ -z "$window" ]
    [ -z "$pane" ]
    [ -z "$pane_created" ]
}

@test "storage_dismiss_notification preserves pane association" {
    source ./lib/storage.sh
    
    local id
    id=$(storage_add_notification "Test message" "" "\$1" "@2" "%3" "1234567890")
    
    storage_dismiss_notification "$id"
    
    # Get dismissed line
    local line
    line=$(storage_list_notifications "dismissed")
    IFS=$'\t' read -r id_field timestamp state session window pane message pane_created level <<< "$line"
    
    [ "$state" = "dismissed" ]
    [ "$session" = "\$1" ]
    [ "$window" = "@2" ]
    [ "$pane" = "%3" ]
    [ "$pane_created" = "1234567890" ]
}

@test "storage_dismiss_all preserves pane association" {
    source ./lib/storage.sh
    
    storage_add_notification "Test 1" "" "\$1" "@2" "%3" "123"
    storage_add_notification "Test 2" "" "\$4" "@5" "%6" "456"
    
    storage_dismiss_all
    
    local dismissed_lines
    dismissed_lines=$(storage_list_notifications "dismissed")
    local line_count
    line_count=$(echo "$dismissed_lines" | wc -l)
    [ "$line_count" -eq 2 ]
    
    # Verify pane associations are present
    while IFS=$'\t' read -r id_field timestamp state session window pane message pane_created level; do
        if [ "$id_field" -eq 1 ]; then
            [ "$session" = "\$1" ]
            [ "$pane_created" = "123" ]
        elif [ "$id_field" -eq 2 ]; then
            [ "$session" = "\$4" ]
            [ "$pane_created" = "456" ]
        fi
    done <<< "$dismissed_lines"
}