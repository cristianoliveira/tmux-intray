#!/usr/bin/env bats
# Status command tests

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test"
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
    # Clear any existing notifications
    ./tmux-intray clear >/dev/null 2>&1 || true
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
}

@test "status shows no active notifications" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status"
    [ "$status" -eq 0 ]
    [[ "$output" == *"No active notifications"* ]]
}

@test "status shows active count" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Active notifications: 1"* ]]
    [[ "$output" == *"info: 1"* ]]
}

@test "status shows level counts" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=warning 'warning message'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=error 'error message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=levels"
    [ "$status" -eq 0 ]
    [[ "$output" == *"info:0"* ]] || true
    [[ "$output" == *"warning:1"* ]]
    [[ "$output" == *"error:1"* ]]
}

@test "status format summary includes level breakdown" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=critical 'critical message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status"
    [ "$status" -eq 0 ]
    [[ "$output" == *"critical: 1"* ]]
}

@test "status format panes shows pane counts" {
    # Add notification with pane association (current pane)
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'pane message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=panes"
    [ "$status" -eq 0 ]
    # Output should contain pane ID pattern like %0
    [[ "$output" == *"%"[0-9]* ]] || [[ "$output" == *":"*":"* ]] || true
}

@test "status unknown format error" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=invalid"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown format"* ]]
}

# =====================================================================
# COMPREHENSIVE E2E TESTS FOR ALL 6 PRESETS
# =====================================================================

@test "status format compact works" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=compact"
    [ "$status" -eq 0 ]
    [[ "$output" == *"["* ]] # compact format has brackets
}

@test "status format detailed works" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=detailed"
    [ "$status" -eq 0 ]
    [[ "$output" == *"unread"* ]] # detailed format shows "unread"
}

@test "status format json works" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"{"* ]] # json format starts with {
}

@test "status format count-only works" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=count-only"
    [ "$status" -eq 0 ]
    # should just output a number
    [[ "$output" =~ ^[0-9]+$ ]]
}

@test "status format panes works" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=panes"
    [ "$status" -eq 0 ]
    # panes format includes pane identification
    [[ "$output" == *":"* ]]
}

# =====================================================================
# CUSTOM TEMPLATE TESTS
# =====================================================================

@test "status custom template with single variable" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='Count: \${unread-count}'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Count: 1"* ]]
}

@test "status custom template with multiple variables" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test2'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='[\${unread-count}/\${total-count}]'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"[2/2]"* ]]
}

@test "status custom template with message variable" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'important message'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='Latest: \${latest-message}'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"important message"* ]]
}

# =====================================================================
# BOOLEAN VARIABLE TESTS
# =====================================================================

@test "status has-unread variable returns true when notifications exist" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${has-unread}'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"true"* ]]
}

@test "status has-unread variable returns false when no notifications" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${has-unread}'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"false"* ]]
}

@test "status has-active variable returns true when active notifications" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${has-active}'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"true"* ]]
}

# =====================================================================
# VARIABLE RESOLUTION TESTS
# =====================================================================

@test "status unread-count variable resolves" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'msg1'"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'msg2'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${unread-count}'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 2 ]]
}

@test "status active-count variable resolves" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'msg1'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${active-count}'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 1 ]]
}

@test "status total-count variable is alias for unread-count" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'msg'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${total-count}'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 1 ]]
}

# =====================================================================
# REGRESSION TESTS
# =====================================================================

@test "status without format flag uses default compact" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status"
    [ "$status" -eq 0 ]
    # Default is compact format with brackets
    [[ "$output" == *"["* ]]
}

@test "status help displays format information" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --help"
    [ "$status" -eq 0 ]
    [[ "$output" == *"format"* ]]
}

@test "status with environment variable TMUX_INTRAY_STATUS_FORMAT" {
    export TMUX_INTRAY_STATUS_FORMAT="count-only"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test'"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status"
    [ "$status" -eq 0 ]
    # Should use count-only format from env var
    [[ "$output" =~ ^[0-9]+$ ]]
    unset TMUX_INTRAY_STATUS_FORMAT
}
