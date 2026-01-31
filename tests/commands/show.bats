#!/usr/bin/env bats
# Show command with modules tests

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test"
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
}

@test "show empty tray" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray show"
    [ "$status" -eq 0 ]
    [[ "$output" == *"empty"* ]]
}

@test "show tray with items" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'item1'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray show"
    [ "$status" -eq 0 ]
    [[ "$output" == *"item1"* ]]
    [[ "$output" == *"Intray Items"* ]]
}

@test "show displays item count" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'item1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'item2'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray show"
    [ "$status" -eq 0 ]
    [[ "$output" == *"2)"* ]]
}
