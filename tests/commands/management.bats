#!/usr/bin/env bash
# Clear and toggle command tests

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

@test "clear tray" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray clear"
    [ "$status" -eq 0 ]
    [[ "$output" == *"cleared"* ]]
}

@test "toggle tray visibility" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray toggle"
    [ "$status" -eq 0 ]
}
