#!/usr/bin/env bats
# Test tmux-intray tray management

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test"
    
    # Clean up any existing server first
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    
    # Start a tmux server with a session for testing
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
}

@test "show tray when empty" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray show"
    [ "$status" -eq 0 ]
    [[ "$output" == *"empty"* ]]
}

@test "clear tray" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'test'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"added"* ]]
    
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray clear"
    [ "$status" -eq 0 ]
    [[ "$output" == *"cleared"* ]]
}

@test "toggle tray visibility" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray toggle"
    [ "$status" -eq 0 ]
}
