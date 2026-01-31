#!/usr/bin/env bats
# Jump command tests

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-jump"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME
    
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
    # Create a second pane to jump to
    tmux -L "$TMUX_SOCKET_NAME" split-window -h -t test:0
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

@test "jump requires an id" {
    run ./bin/tmux-intray jump
    [ "$status" -eq 1 ]
    [[ "$output" == *"requires a notification ID"* ]]
}

@test "jump with invalid id fails" {
    run ./bin/tmux-intray jump 999
    [ "$status" -eq 1 ]
    [[ "$output" == *"not found"* ]]
}

@test "jump to pane with association" {
    # Add notification with current pane association
    # We need to run inside tmux to get pane context
    local session window pane pane_created
    read -r session window pane pane_created <<< "$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{session_id} #{window_id} #{pane_id} #{pane_created}')"
    
    # Use tmux run-shell to add notification within the tmux server context
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'test message'"
    # Get the ID (output includes ID and success message)
    local id
    id=$(tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'another message' 2>&1 | head -n1")
    
    # Jump to pane (should succeed)
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray jump $id"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Jumped to pane"* ]]
}

@test "jump to dismissed notification still works" {
    local session window pane pane_created
    read -r session window pane pane_created <<< "$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{session_id} #{window_id} #{pane_id} #{pane_created}')"
    
    local id
    id=$(tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'test message' 2>&1 | head -n1")
    
    # Dismiss notification
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray dismiss $id"
    
    # Jump should still work (with warning)
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray jump $id"
    [ "$status" -eq 0 ]
    [[ "$output" == *"dismissed"* ]]
    [[ "$output" == *"Jumped to pane"* ]]
}

@test "jump fails when pane no longer exists" {
    # Create a notification with a fake pane association
    # We'll directly write to storage to simulate pane that doesn't exist
    source ./lib/storage.sh
    local id
    id=$(storage_add_notification "Test" "" "\$none" "@none" "%none" "123")
    
    # Try to jump
    run ./bin/tmux-intray jump "$id"
    [ "$status" -eq 1 ]
    [[ "$output" == *"does not exist"* ]]
}