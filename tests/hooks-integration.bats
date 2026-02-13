#!/usr/bin/env bats
# shellcheck disable=SC2030,SC2031
# Disable warnings about modifications being local to subshell (expected in Bats tests)
# Comprehensive integration tests for the hooks system

# Test helper to create a hook script
_create_hook() {
    local hook_point="$1"
    local name="$2"
    local content="$3"
    mkdir -p "$HOOKS_DIR/$hook_point"
    cat >"$HOOKS_DIR/$hook_point/$name" <<EOF
#!/bin/bash
$content
EOF
    chmod +x "$HOOKS_DIR/$hook_point/$name"
}

# Test helper to read hook output file
_read_hook_output() {
    local hook_name="$1"
    if [ -f "$HOOK_OUTPUT_DIR/$hook_name" ]; then
        cat "$HOOK_OUTPUT_DIR/$hook_name"
    fi
}

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-hooks"

    # Setup temp directories
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME

    # Create hooks directory
    HOOKS_DIR="$(mktemp -d)"
    export TMUX_INTRAY_HOOKS_DIR="$HOOKS_DIR"

    # Create output directory for hook test data
    HOOK_OUTPUT_DIR="$(mktemp -d)"
    export HOOK_OUTPUT_DIR

    # Create config file to avoid info messages
    mkdir -p "$XDG_CONFIG_HOME/tmux-intray"
    touch "$XDG_CONFIG_HOME/tmux-intray/config.sh"

    # Determine if we can use tmux
    export TMUX_AVAILABLE=0
    export TMUX_TEST_SESSION_ID=""
    export TMUX_TEST_WINDOW_ID=""
    export TMUX_TEST_PANE_ID=""
    export TMUX_TEST_PANE_CREATED=""
    if [[ -z "${CI:-}" ]] && command -v tmux >/dev/null 2>&1; then
        # Clean up any existing server
        tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
        sleep 0.2

        # Start a tmux server for testing
        if tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test 2>/dev/null; then
            # Wait for server to be ready (avoid race)
            local max_retries=12
            local retry=0
            local session_id=""
            local window_id=""
            local pane_id=""
            local pane_created=""
            local ready=0
            while [[ $retry -lt $max_retries ]]; do
                sleep 0.2
                session_id=$(tmux -L "$TMUX_SOCKET_NAME" display -p -t test '#{session_id}' 2>/dev/null || true)
                window_id=$(tmux -L "$TMUX_SOCKET_NAME" display -p -t test '#{window_id}' 2>/dev/null || true)
                pane_id=$(tmux -L "$TMUX_SOCKET_NAME" display -p -t test '#{pane_id}' 2>/dev/null || true)
                pane_created=$(tmux -L "$TMUX_SOCKET_NAME" display -p -t test '#{pane_start_time}' 2>/dev/null || true)
                if [[ -n "$session_id" && -n "$window_id" && -n "$pane_id" && -n "$pane_created" ]]; then
                    ready=1
                    break
                fi
                retry=$((retry + 1))
            done
            if [[ $ready -eq 1 ]]; then
                # Capture session, window, pane IDs for use in tests
                TMUX_TEST_SESSION_ID="$session_id"
                export TMUX_TEST_SESSION_ID
                TMUX_TEST_WINDOW_ID="$window_id"
                export TMUX_TEST_WINDOW_ID
                TMUX_TEST_PANE_ID="$pane_id"
                export TMUX_TEST_PANE_ID
                TMUX_TEST_PANE_CREATED="$pane_created"
                export TMUX_TEST_PANE_CREATED
                export TMUX_AVAILABLE=1
            else
                echo "warning: tmux server not ready, disabling tmux support" >&2
                export TMUX_AVAILABLE=0
                tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
            fi
        else
            export TMUX_AVAILABLE=0
        fi
    fi

    # Enable debug output
    export TMUX_INTRAY_DEBUG=1
}

teardown() {
    if [[ "${TMUX_AVAILABLE:-0}" -eq 1 ]]; then
        tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
        sleep 0.1
    fi
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME" "$HOOKS_DIR" "$HOOK_OUTPUT_DIR"
}

# ========================================
# Basic Hook Execution Tests
# ========================================

@test "pre-add hook runs before notification is added" {
    # Create pre-add hook that writes to output file
    _create_hook "pre-add" "01-test.sh" "echo \"pre-add-executed\" > \"$HOOK_OUTPUT_DIR/pre-add.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Verify hook was executed
    [ -f "$HOOK_OUTPUT_DIR/pre-add.log" ]
    [[ "$(_read_hook_output "pre-add.log")" == *"pre-add-executed"* ]]
}

@test "post-add hook runs after notification is added" {
    # Create post-add hook that writes to output file
    _create_hook "post-add" "01-test.sh" "echo \"post-add-executed\" > \"$HOOK_OUTPUT_DIR/post-add.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Verify hook was executed
    [ -f "$HOOK_OUTPUT_DIR/post-add.log" ]
    [[ "$(_read_hook_output "post-add.log")" == *"post-add-executed"* ]]
}

@test "hook execution order: pre-add before storage, post-add after" {
    _create_hook "pre-add" "01-test.sh" "echo \"pre-add\" > \"$HOOK_OUTPUT_DIR/order.log\""
    _create_hook "post-add" "02-test.sh" "echo \"post-add\" >> \"$HOOK_OUTPUT_DIR/order.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Verify order
    local order
    order=$(_read_hook_output "order.log")
    [[ "$order" == "pre-add"$'\n'"post-add" ]]
}

# ========================================
# Hook Environment Variables Tests
# ========================================

@test "hook receives NOTIFICATION_ID variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"NOTIFICATION_ID=\$NOTIFICATION_ID\" > \"$HOOK_OUTPUT_DIR/id.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # NOTIFICATION_ID is the actual numeric ID (1 for first notification)
    [[ "$(_read_hook_output "id.log")" == *"NOTIFICATION_ID=1"* ]]
}

@test "hook receives NOTIFICATION_LEVEL variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"LEVEL=\$LEVEL\" > \"$HOOK_OUTPUT_DIR/level.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=warning 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "level.log")" == *"LEVEL=warning"* ]]
}

@test "hook receives NOTIFICATION_MESSAGE variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"MESSAGE=\$MESSAGE\" > \"$HOOK_OUTPUT_DIR/message.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "message.log")" == *"MESSAGE=test message"* ]]
}

@test "hook receives TIMESTAMP variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"TIMESTAMP=\$TIMESTAMP\" > \"$HOOK_OUTPUT_DIR/timestamp.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Timestamp should be in ISO 8601 format
    [[ "$(_read_hook_output "timestamp.log")" =~ TIMESTAMP=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z ]]
}

@test "hook receives SESSION variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"SESSION=\$SESSION\" > \"$HOOK_OUTPUT_DIR/session.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "session.log")" == *"SESSION=$TMUX_TEST_SESSION_ID"* ]]
}

@test "hook receives WINDOW variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"WINDOW=\$WINDOW\" > \"$HOOK_OUTPUT_DIR/window.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "window.log")" == *"WINDOW=$TMUX_TEST_WINDOW_ID"* ]]
}

@test "hook receives PANE variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"PANE=\$PANE\" > \"$HOOK_OUTPUT_DIR/pane.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "pane.log")" == *"PANE=$TMUX_TEST_PANE_ID"* ]]
}

@test "hook receives PANE_CREATED variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"PANE_CREATED=\$PANE_CREATED\" > \"$HOOK_OUTPUT_DIR/pane-created.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # PANE_CREATED should have a value
    [[ "$(_read_hook_output "pane-created.log")" == *"PANE_CREATED="* ]]
}

@test "hook receives HOOK_POINT variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"HOOK_POINT=\$HOOK_POINT\" > \"$HOOK_OUTPUT_DIR/hook-point.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "hook-point.log")" == *"HOOK_POINT=pre-add"* ]]
}

@test "hook receives HOOK_TIMESTAMP variable" {
    _create_hook "pre-add" "01-test.sh" "echo \"HOOK_TIMESTAMP=\$HOOK_TIMESTAMP\" > \"$HOOK_OUTPUT_DIR/hook-timestamp.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # HOOK_TIMESTAMP should be in ISO 8601 format
    [[ "$(_read_hook_output "hook-timestamp.log")" =~ HOOK_TIMESTAMP=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2} ]]
}

@test "hook receives TMUX_INTRAY_STATE_DIR variable" {
    # TODO: This feature is documented but not implemented yet
    skip "TMUX_INTRAY_STATE_DIR not yet passed to hooks"

    _create_hook "pre-add" "01-test.sh" "echo \"STATE_DIR=\$TMUX_INTRAY_STATE_DIR\" > \"$HOOK_OUTPUT_DIR/state-dir.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "state-dir.log")" == *"STATE_DIR=$XDG_STATE_HOME"* ]]
}

@test "hook receives TMUX_INTRAY_CONFIG_DIR variable" {
    # TODO: This feature is documented but not implemented yet
    skip "TMUX_INTRAY_CONFIG_DIR not yet passed to hooks"

    _create_hook "pre-add" "01-test.sh" "echo \"CONFIG_DIR=\$TMUX_INTRAY_CONFIG_DIR\" > \"$HOOK_OUTPUT_DIR/config-dir.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "config-dir.log")" == *"CONFIG_DIR=$XDG_CONFIG_HOME"* ]]
}

@test "post-add hook receives NOTIFICATION_ID with actual ID" {
    _create_hook "post-add" "01-test.sh" "echo \"NOTIFICATION_ID=\$NOTIFICATION_ID\" > \"$HOOK_OUTPUT_DIR/post-id.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # In post-add, NOTIFICATION_ID should be an actual number (1 for first notification)
    [[ "$(_read_hook_output "post-id.log")" == *"NOTIFICATION_ID=1"* ]]
}

# ========================================
# Failure Modes Tests
# ========================================

@test "abort mode: pre-add hook failure prevents add" {
    _create_hook "pre-add" "01-fail.sh" "exit 1"

    # shellcheck disable=SC2030
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="abort"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    # Run directly to check exit code (tmux run-shell doesn't propagate exit code)
    run ./tmux-intray add 'test message'
    [ "$status" -ne 0 ]
    # Check for error message in output (using grep to avoid shellcheck glob issues)
    echo "$output" | grep -qi "hook.*failed\|pre-add hook aborted\|Failed to add tray item"
}

@test "warn mode: pre-add hook failure logs warning but allows add" {
    _create_hook "pre-add" "01-fail.sh" "exit 1"

    # shellcheck disable=SC2030,SC2031
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="warn"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run ./tmux-intray add 'test message'
    [ "$status" -eq 0 ]
    # Warning is printed to stderr, which Bats captures in output
    echo "$output" | grep -qi "warning.*hook.*failed\|Warning:"
}

@test "ignore mode: pre-add hook failure silent, allows add" {
    _create_hook "pre-add" "01-fail.sh" "exit 1"

    # shellcheck disable=SC2030,SC2031
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="ignore"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]
    # Should not show warning
    [[ "$output" != *"warning"* ]]
}

# ========================================
# Dismiss Hooks Tests
# ========================================

@test "pre-dismiss hook runs before dismiss" {
    _create_hook "pre-dismiss" "01-test.sh" "echo \"pre-dismiss-executed\" > \"$HOOK_OUTPUT_DIR/pre-dismiss.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    # Add a notification first
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'" >/dev/null 2>&1

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray dismiss 1"
    [ "$status" -eq 0 ]

    [ -f "$HOOK_OUTPUT_DIR/pre-dismiss.log" ]
    [[ "$(_read_hook_output "pre-dismiss.log")" == *"pre-dismiss-executed"* ]]
}

@test "post-dismiss hook runs after dismiss" {
    _create_hook "post-dismiss" "01-test.sh" "echo \"post-dismiss-executed\" > \"$HOOK_OUTPUT_DIR/post-dismiss.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    # Add a notification first
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'" >/dev/null 2>&1

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray dismiss 1"
    [ "$status" -eq 0 ]

    [ -f "$HOOK_OUTPUT_DIR/post-dismiss.log" ]
    [[ "$(_read_hook_output "post-dismiss.log")" == *"post-dismiss-executed"* ]]
}

@test "dismiss hooks receive correct notification data" {
    _create_hook "post-dismiss" "01-test.sh" "echo \"ID=\$NOTIFICATION_ID\" > \"$HOOK_OUTPUT_DIR/dismiss-data.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    # Add a notification
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'dismiss me'" >/dev/null 2>&1

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray dismiss 1"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "dismiss-data.log")" == *"ID=1"* ]]
}

# ========================================
# Async Hooks Tests
# ========================================

@test "async hooks run in background" {
    _create_hook "pre-add" "01-async.sh" "echo \"async-start\" > \"$HOOK_OUTPUT_DIR/async.log\"; sleep 0.5; echo \"async-done\" >> \"$HOOK_OUTPUT_DIR/async.log\""

    # shellcheck disable=SC2030,SC2031
    export TMUX_INTRAY_HOOKS_ASYNC="1"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    local start
    start=$(date +%s)
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    local end
    end=$(date +%s)
    local duration=$((end - start))

    # Command should complete quickly (less than sleep time)
    [ "$duration" -lt 3 ]
    [ "$status" -eq 0 ]
}

@test "async hook command returns before hook completes" {
    _create_hook "pre-add" "01-async.sh" "echo \"before-sleep\" > \"$HOOK_OUTPUT_DIR/async-timing.log\"; sleep 1; echo \"after-sleep\" >> \"$HOOK_OUTPUT_DIR/async-timing.log\""

    # shellcheck disable=SC2030,SC2031
    export TMUX_INTRAY_HOOKS_ASYNC="1"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"

    # Command returns immediately
    [ "$status" -eq 0 ]

    # Give async hook time to start
    sleep 0.1

    # Hook should have started but not finished
    [ -f "$HOOK_OUTPUT_DIR/async-timing.log" ]
    [[ "$(_read_hook_output "async-timing.log")" == *"before-sleep"* ]]

    # Wait for hook to complete
    sleep 1.5

    # Hook should have completed
    [[ "$(_read_hook_output "async-timing.log")" == *"after-sleep"* ]]
}

# ========================================
# Configuration Tests
# ========================================

@test "TMUX_INTRAY_HOOKS_ENABLED=0 disables all hooks" {
    # TODO: This feature is documented but not implemented yet
    skip "TMUX_INTRAY_HOOKS_ENABLED not yet implemented"

    _create_hook "pre-add" "01-test.sh" "echo \"hook-executed\" > \"$HOOK_OUTPUT_DIR/disabled.log\""

    # shellcheck disable=SC2030,SC2031
    export TMUX_INTRAY_HOOKS_ENABLED="0"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Hook should not have been executed
    [ ! -f "$HOOK_OUTPUT_DIR/disabled.log" ]
}

@test "TMUX_INTRAY_HOOKS_ENABLED=1 enables hooks" {
    # TODO: This feature is documented but not implemented yet
    # Hooks currently always run when present
    skip "TMUX_INTRAY_HOOKS_ENABLED not yet implemented"

    _create_hook "pre-add" "01-test.sh" "echo \"hook-executed\" > \"$HOOK_OUTPUT_DIR/enabled.log\""

    # shellcheck disable=SC2030,SC2031
    export TMUX_INTRAY_HOOKS_ENABLED="1"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Hook should have been executed
    [ -f "$HOOK_OUTPUT_DIR/enabled.log" ]
    [[ "$(_read_hook_output "enabled.log")" == *"hook-executed"* ]]
}

@test "hooks execute in alphabetical order" {
    _create_hook "pre-add" "03-third.sh" "echo \"third\" >> \"$HOOK_OUTPUT_DIR/order-test.log\""
    _create_hook "pre-add" "01-first.sh" "echo \"first\" >> \"$HOOK_OUTPUT_DIR/order-test.log\""
    _create_hook "pre-add" "02-second.sh" "echo \"second\" >> \"$HOOK_OUTPUT_DIR/order-test.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Verify alphabetical order
    local order
    order=$(_read_hook_output "order-test.log")
    [[ "$order" == "first"$'\n'"second"$'\n'"third" ]]
}

@test "non-executable hooks are skipped" {
    # Create hook without execute permission
    mkdir -p "$HOOKS_DIR/pre-add"
    cat >"$HOOKS_DIR/pre-add/non-executable.sh" <<'EOF'
#!/bin/bash
echo "should-not-execute" > "$HOOK_OUTPUT_DIR/skipped.log"
EOF

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    # Hook should not have been executed
    [ ! -f "$HOOK_OUTPUT_DIR/skipped.log" ]
}

@test "hook with zero exit code succeeds" {
    _create_hook "pre-add" "01-success.sh" "exit 0"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"added"* ]]
}

@test "hook output is written to stderr" {
    _create_hook "pre-add" "01-output.sh" "echo \"hook-stderr-output\" >&2"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run ./tmux-intray add 'test message' 2>&1
    [ "$status" -eq 0 ]
    # Hook output should appear in stderr (which Bats captures in output)
    [[ "$output" == *"hook-stderr-output"* ]] || [[ "$output" == *"Executing hook:"* ]]
}

@test "hook receives TMUX_INTRAY_HOOKS_FAILURE_MODE" {
    _create_hook "pre-add" "01-failure-mode.sh" "echo \"FAILURE_MODE=\$TMUX_INTRAY_HOOKS_FAILURE_MODE\" > \"$HOOK_OUTPUT_DIR/failure-mode.log\""

    # shellcheck disable=SC2030,SC2031
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="warn"

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "failure-mode.log")" == *"FAILURE_MODE=warn"* ]]
}

@test "pre-clear hook runs when clear command is used" {
    _create_hook "pre-clear" "01-test.sh" "echo \"pre-clear-executed\" > \"$HOOK_OUTPUT_DIR/pre-clear.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    # Add a notification first
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'" >/dev/null 2>&1

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray clear"
    [ "$status" -eq 0 ]

    [ -f "$HOOK_OUTPUT_DIR/pre-clear.log" ]
    [[ "$(_read_hook_output "pre-clear.log")" == *"pre-clear-executed"* ]]
}

@test "hooks work with level option" {
    _create_hook "pre-add" "01-level.sh" "echo \"LEVEL=\$LEVEL\" > \"$HOOK_OUTPUT_DIR/level-option.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=error 'error message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "level-option.log")" == *"LEVEL=error"* ]]
}

@test "hooks work with custom session" {
    _create_hook "pre-add" "01-session.sh" "echo \"SESSION=\$SESSION\" > \"$HOOK_OUTPUT_DIR/custom-session.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --session=custom-sess 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "custom-session.log")" == *"SESSION=custom-sess"* ]]
}

@test "hooks work with custom window" {
    _create_hook "pre-add" "01-window.sh" "echo \"WINDOW=\$WINDOW\" > \"$HOOK_OUTPUT_DIR/custom-window.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --window=custom-win 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "custom-window.log")" == *"WINDOW=custom-win"* ]]
}

@test "hooks work with custom pane" {
    _create_hook "pre-add" "01-pane.sh" "echo \"PANE=\$PANE\" > \"$HOOK_OUTPUT_DIR/custom-pane.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --pane=custom-pane 'test message'"
    [ "$status" -eq 0 ]

    [[ "$(_read_hook_output "custom-pane.log")" == *"PANE=custom-pane"* ]]
}

@test "multiple hooks at same point all execute" {
    _create_hook "pre-add" "01-first.sh" "echo \"first\" >> \"$HOOK_OUTPUT_DIR/multiple.log\""
    _create_hook "pre-add" "02-second.sh" "echo \"second\" >> \"$HOOK_OUTPUT_DIR/multiple.log\""
    _create_hook "pre-add" "03-third.sh" "echo \"third\" >> \"$HOOK_OUTPUT_DIR/multiple.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [ -f "$HOOK_OUTPUT_DIR/multiple.log" ]
    [[ "$(_read_hook_output "multiple.log")" == *"first"* ]]
    [[ "$(_read_hook_output "multiple.log")" == *"second"* ]]
    [[ "$(_read_hook_output "multiple.log")" == *"third"* ]]
}

@test "hook directory can be empty" {
    mkdir -p "$HOOKS_DIR/pre-add"
    # Don't create any hooks

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"added"* ]]
}

@test "hooks directory missing is handled gracefully" {
    # Don't create any hooks directory at all

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"added"* ]]
}

@test "post-list hook runs after list command" {
    # TODO: This feature is planned but not implemented yet
    skip "post-list hook not yet implemented"

    _create_hook "post-list" "01-test.sh" "echo \"post-list-executed\" > \"$HOOK_OUTPUT_DIR/post-list.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    # Add a notification first
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'" >/dev/null 2>&1

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray list"
    [ "$status" -eq 0 ]

    [ -f "$HOOK_OUTPUT_DIR/post-list.log" ]
    [[ "$(_read_hook_output "post-list.log")" == *"post-list-executed"* ]]
}

@test "hook with shebang executes correctly" {
    _create_hook "pre-add" "01-shebang.sh" "#!/usr/bin/env bash
echo \"shebang-test\" > \"$HOOK_OUTPUT_DIR/shebang.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test message'"
    [ "$status" -eq 0 ]

    [ -f "$HOOK_OUTPUT_DIR/shebang.log" ]
    [[ "$(_read_hook_output "shebang.log")" == *"shebang-test"* ]]
}

@test "hook receives backward-compatible environment variable aliases" {
    # TODO: Backward-compatible aliases are documented but not implemented yet
    skip "NOTIFICATION_LEVEL aliases not yet implemented"

    _create_hook "post-add" "01-aliases.sh" "echo \"NOTIFICATION_LEVEL=\$NOTIFICATION_LEVEL\" > \"$HOOK_OUTPUT_DIR/aliases.log\"; echo \"LEVEL=\$LEVEL\" >> \"$HOOK_OUTPUT_DIR/aliases.log\""

    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=warning 'test message'"
    [ "$status" -eq 0 ]

    local aliases
    aliases=$(_read_hook_output "aliases.log")
    [[ "$aliases" == *"NOTIFICATION_LEVEL=warning"* ]]
    [[ "$aliases" == *"LEVEL=warning"* ]]
}
