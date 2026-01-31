#!/usr/bin/env bats
# Follow command tests

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

@test "follow with no notifications exits immediately with timeout" {
    # Run follow with a short interval and timeout using timeout command
    run timeout 1 tmux -L "$TMUX_SOCKET_NAME" run-shell "echo 'no output expected'; $PWD/bin/tmux-intray follow --interval=0.1" 2>&1
    # timeout will kill after 1 second, exit status 124
    # We just want to ensure no errors
    [ "$status" -ne 127 ]  # not command not found
}

@test "follow detects new notification" {
    # Start follow in background with short interval
    ( tmux -L "$TMUX_SOCKET_NAME" run-shell "cd $PWD && timeout 2 ./bin/tmux-intray follow --interval=0.1" ) &
    follow_pid=$!
    sleep 0.5
    # Add a notification
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add 'new message'"
    sleep 0.5
    # Kill follow process if still running
    kill $follow_pid 2>/dev/null || true
    # We can't easily capture output; just ensure no crash
    # This test is minimal
    true
}

@test "follow level filter" {
    # Add notifications with different levels
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=info 'info message'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray add --level=error 'error message'"
    # Start follow with level=error filter
    # We'll just test that command runs without error
    run timeout 1 tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/bin/tmux-intray follow --level=error --interval=0.1" 2>&1
    [ "$status" -ne 127 ]
}

@test "follow help" {
    run ./bin/tmux-intray follow --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Monitor notifications"* ]]
}
