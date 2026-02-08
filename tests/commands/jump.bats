#!/usr/bin/env bats
# Jump command integration tests - verify pane jumping behavior
# shellcheck disable=SC1091,SC2016

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
    tmux -L "$TMUX_SOCKET_NAME" split-window -h -t test
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
    run ./tmux-intray jump
    [[ "$output" == *"requires a notification ID"* ]]
}

@test "jump with invalid id fails" {
    run ./tmux-intray jump 999
    [[ "$output" == *"not found"* ]]
}

@test "jump to pane with association succeeds" {
    # Tiger Style Integration Test: ASSERTION 1 - Jump command should succeed when pane exists
    # Tiger Style Integration Test: ASSERTION 2 - Output should show success message

    source ./lib/storage.sh
    local session window pane pane_created
    read -r session window pane pane_created <<<"$(tmux -L "$TMUX_SOCKET_NAME" display -p -t test:0 '#{session_id} #{window_id} #{pane_id} #{pane_created}')"

    local id
    id=$(storage_add_notification "test message" "" "$session" "$window" "$pane" "$pane_created")

    run ./tmux-intray jump "$id"
    [[ "$output" == *"Jumped to session"* ]]
}

@test "jump to dismissed notification still works" {
    # Tiger Style Integration Test: ASSERTION - Should succeed even if notification is dismissed

    source ./lib/storage.sh
    local session window pane pane_created
    read -r session window pane pane_created <<<"$(tmux -L "$TMUX_SOCKET_NAME" display -p -t test:0 '#{session_id} #{window_id} #{pane_id} #{pane_created}')"

    local id
    id=$(storage_add_notification "test message" "" "$session" "$window" "$pane" "$pane_created")

    ./tmux-intray dismiss "$id" >/dev/null 2>&1

    run ./tmux-intray jump "$id"
    [[ "$output" == *"dismissed"* ]]
    [[ "$output" == *"Jumped to session"* ]]
}

@test "jump fails when pane no longer exists" {
    # Tiger Style Integration Test: ASSERTION - Should fail gracefully with error when pane/window invalid

    source ./lib/storage.sh
    local id
    id=$(storage_add_notification "Test" "" '$none' '@none' '%none' "123")

    run ./tmux-intray jump "$id"
    [[ "$output" == *"does not exist"* ]]
}

@test "jump error handling shows error instead of success" {
    # Tiger Style Integration Test: Bug Fix Verification
    # ASSERTION: When jump fails, error message shown (not success message)
    # This verifies the fix for error swallowing in the original code

    source ./lib/storage.sh
    local id
    id=$(storage_add_notification "Error test" "" '$invalid' '@invalid' '%invalid' "123")

    run ./tmux-intray jump "$id"
    # Should NOT show success message when it fails
    [[ ! "$output" == *"Jumped to session"* ]]
}
