#!/usr/bin/env bats
# Add command with modules tests

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

@test "add requires a message" {
    run ./tmux-intray add
    [ "$status" -eq 1 ]
    [[ "$output" == *"requires a message"* ]]
}

@test "add item to tray" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"added"* ]]
}

@test "add empty message fails" {
    run ./tmux-intray add ""
    [ "$status" -eq 1 ]
    [[ "$output" == *"empty"* ]]
}

@test "add long message fails (>1000 chars)" {
    local long_message
    long_message=$(printf 'a%.0s' {1..1001})
    run ./tmux-intray add "$long_message"
    [ "$status" -eq 1 ]
    [[ "$output" == *"too long"* ]]
}
