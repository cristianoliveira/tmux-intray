#!/usr/bin/env bats
# List command with pane filtering tests

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-list-pane"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME

    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
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

@test "list with pane filter shows only matching notifications" {
    source ./lib/storage.sh

    # Add notifications with different pane associations
    storage_add_notification "Message 1" "" "\$1" "@2" "%3" "123"
    storage_add_notification "Message 2" "" "\$1" "@2" "%4" "456"
    storage_add_notification "Message 3" "" "\$5" "@6" "%7" "789"

    # Filter by pane %3
    run storage_list_notifications "all"
    # Manually filter via our test
    local filtered
    filtered=$(echo "$output" | while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local id timestamp state session window pane message pane_created level
            _parse_notification_line "$line" id timestamp state session window pane message pane_created level
            if [[ "$pane" == "%3" ]]; then
                echo "$line"
            fi
        fi
    done)

    [ "$(echo "$filtered" | wc -l)" -eq 1 ]
    [[ "$filtered" == *"Message 1"* ]]
}

@test "list command with --pane filter" {
    # Use tmux to add notifications with pane context
    local session window pane pane_created
    read -r session window pane pane_created <<<"$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{session_id} #{window_id} #{pane_id} #{pane_created}')"

    # Add two notifications with same pane
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'First message'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Second message'"

    # Create another pane and add notification there (simulate by directly writing with different pane ID)
    source ./lib/storage.sh
    storage_add_notification "Other pane message" "" "\$other" "@other" "%other" "999"

    # List with pane filter
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --pane $pane --format=compact"
    [ "$status" -eq 0 ]
    [[ "$output" == *"First message"* ]]
    [[ "$output" == *"Second message"* ]]
    [[ ! "$output" == *"Other pane message"* ]]
}

@test "list table format includes pane column" {
    source ./lib/storage.sh
    storage_add_notification "Test message" "" "\$1" "@2" "%3" "123"

    run ./bin/tmux-intray list --format=table
    [ "$status" -eq 0 ]
    [[ "$output" == *"Pane"* ]]
    [[ "$output" == *"%3"* ]]
}

@test "list with pane filter on non-existent pane shows empty" {
    run ./bin/tmux-intray list --pane %nonexistent --format=compact
    [ "$status" -eq 0 ]
    [[ "$output" == *"No notifications found"* ]] || ! grep -qv '^Loaded' <<<"$output"
}
