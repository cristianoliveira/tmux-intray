#!/usr/bin/env bats
# Test tmux-intray tray management

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME
    
    # Clean up any existing server first
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    
    # Start a tmux server with a session for testing
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
    # Get socket path and set TMUX environment variable so plain tmux commands use our test server
    socket_path=$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{socket_path}' 2>/dev/null)
    export TMUX_SOCKET_PATH="$socket_path"
    # TMUX format: socket_path,client_fd,client_pid
    # We'll fake client_fd and client_pid (not critical for our use)
    export TMUX="$socket_path,12345,0"
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    # Remove socket file if it still exists
    if [[ -n "$TMUX_SOCKET_PATH" && -e "$TMUX_SOCKET_PATH" ]]; then
        rm -f "$TMUX_SOCKET_PATH"
    fi
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
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
