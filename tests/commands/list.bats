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
