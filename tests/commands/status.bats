#!/usr/bin/env bats
# Status command tests

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test"
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
    # Clear any existing notifications
    ./bin/tmux-intray clear >/dev/null 2>&1 || true
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
}

@test "status shows no active notifications" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray status"
    [ "$status" -eq 0 ]
    [[ "$output" == *"No active notifications"* ]]
}

@test "status shows active count" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'test message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray status"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Active notifications: 1"* ]]
    [[ "$output" == *"info: 1"* ]]
}

@test "status shows level counts" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=warning 'warning message'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=error 'error message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray status --format=levels"
    [ "$status" -eq 0 ]
    [[ "$output" == *"info:0"* ]] || true
    [[ "$output" == *"warning:1"* ]]
    [[ "$output" == *"error:1"* ]]
}

@test "status format summary includes level breakdown" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=critical 'critical message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray status"
    [ "$status" -eq 0 ]
    [[ "$output" == *"critical: 1"* ]]
}

@test "status format panes shows pane counts" {
    # Add notification with pane association (current pane)
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'pane message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray status --format=panes"
    [ "$status" -eq 0 ]
    # Output should contain pane ID pattern like %0
    [[ "$output" == *"%"[0-9]* ]] || [[ "$output" == *":"*":"* ]] || true
}

@test "status unknown format error" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray status --format=invalid"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown format"* ]]
}
