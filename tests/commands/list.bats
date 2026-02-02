#!/usr/bin/env bats
# Test list command

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-list"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME

    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
    # Capture session, window, pane IDs for use in tests
    TMUX_TEST_SESSION_ID=$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{session_id}')
    export TMUX_TEST_SESSION_ID
    TMUX_TEST_WINDOW_ID=$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{window_id}')
    export TMUX_TEST_WINDOW_ID
    TMUX_TEST_PANE_ID=$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{pane_id}')
    export TMUX_TEST_PANE_ID
    # Enable debug output for tests
    export TMUX_INTRAY_DEBUG=1
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
}

@test "list command shows help" {
    run ./bin/tmux-intray list --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray list"* ]]
}

@test "list with empty tray" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list"
    [ "$status" -eq 0 ]
    [[ "$output" == *"empty"* ]]
}

@test "list shows active notifications" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Test message 1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Test message 2'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Test message 1"* ]]
    [[ "$output" == *"Test message 2"* ]]
}

@test "list --format=table" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Test message'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --format=table"
    [ "$status" -eq 0 ]
    [[ "$output" == *"ID"* ]]
    [[ "$output" == *"Timestamp"* ]]
    [[ "$output" == *"Message"* ]]
}

@test "list --format=compact" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Test message'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --format=compact"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Test message"* ]]
}

@test "list --dismissed shows dismissed only" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Active'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'To dismiss'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray dismiss 2"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --dismissed"
    [ "$status" -eq 0 ]
    [[ "$output" == *"To dismiss"* ]]
    [[ "$output" != *"Active"* ]]
}

@test "list --all shows all notifications" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Active'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'To dismiss'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray dismiss 2"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --all"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Active"* ]]
    [[ "$output" == *"To dismiss"* ]]
}

@test "list --session filters by session" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Msg1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --session wrong_session 'Msg2'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --session=wrong_session"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Msg2"* ]]
    [[ "$output" != *"Msg1"* ]]
}

@test "list --window filters by window" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Msg1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --window wrong_window 'Msg2'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --window=wrong_window"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Msg2"* ]]
    [[ "$output" != *"Msg1"* ]]
}

@test "list --older-than filters older notifications" {
    # Add a notification now (should be newer than 1 day ago)
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Recent'"
    # Older than 0 days (older than now) should include recent notifications (since they are older than now)
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --older-than=0"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Recent"* ]]
}

@test "list --newer-than filters newer notifications" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Recent'"
    # Newer than 1 day ago should include recent
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --newer-than=1"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Recent"* ]]
}

@test "list --search filters by substring" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Hello world'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Goodbye world'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --search='Hello'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Hello"* ]]
    [[ "$output" != *"Goodbye"* ]]
}

@test "list --search with --regex filters by regex" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Error 123'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'Warning 456'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --search='Error.*' --regex"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Error"* ]]
    [[ "$output" != *"Warning"* ]]
}

@test "list --group-by groups notifications" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=warning 'Warning 1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=warning 'Warning 2'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=error 'Error 1'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --group-by=level --format=table"
    [ "$status" -eq 0 ]
    [[ "$output" == *"=== warning"* ]]
    [[ "$output" == *"=== error"* ]]
    [[ "$output" == *"Warning 1"* ]]
    [[ "$output" == *"Warning 2"* ]]
    [[ "$output" == *"Error 1"* ]]
}

@test "list --group-by with --group-count shows only counts" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=warning 'Warning 1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=warning 'Warning 2'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=error 'Error 1'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --group-by=level --group-count --format=table"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Group: warning (2)"* ]]
    [[ "$output" == *"Group: error (1)"* ]]
    [[ "$output" != *"Warning 1"* ]]
    [[ "$output" != *"Error 1"* ]]
}

@test "list combined filters work together" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=warning 'Test warning'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=error 'Test error'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=info 'Test info'"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray list --level=warning --search='Test' --format=compact"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Test warning"* ]]
    [[ "$output" != *"Test error"* ]]
    [[ "$output" != *"Test info"* ]]
}
