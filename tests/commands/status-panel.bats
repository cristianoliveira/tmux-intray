#!/usr/bin/env bats
# Test status-panel command

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-status-panel"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME

    # Determine if we can use tmux
    export TMUX_AVAILABLE=0
    if [[ -z "${CI:-}" ]] && command -v tmux >/dev/null 2>&1; then
        # Clean up any existing server
        tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
        sleep 0.2

        # Start a tmux server for testing
        if tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test 2>/dev/null; then
            # Wait for server to be ready and socket path to exist
            local max_retries=5
            local retry=0
            local socket_path=""
            while [[ $retry -lt $max_retries ]]; do
                sleep 0.2
                socket_path=$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{socket_path}' 2>/dev/null)
                if [[ -n "$socket_path" && -S "$socket_path" ]]; then
                    break
                fi
                retry=$((retry + 1))
            done
            if [[ -n "$socket_path" && -S "$socket_path" ]]; then
                export TMUX_AVAILABLE=1
            else
                echo "warning: tmux socket path missing or not a socket, disabling tmux support" >&2
                export TMUX_AVAILABLE=0
                tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
            fi
        else
            export TMUX_AVAILABLE=0
        fi
    fi
}

teardown() {
    if [[ "${TMUX_AVAILABLE:-0}" -eq 1 ]]; then
        tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
        sleep 0.1
    fi
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
}

@test "status-panel shows help" {
    run ./tmux-intray status-panel --help 2>/dev/null
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray status-panel"* ]]
}

@test "status-panel compact format with zero notifications" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status-panel --format=compact" 2>/dev/null
    [ "$status" -eq 0 ]
    # Should output empty line (no indicator)
    [ -z "$output" ]
}

@test "status-panel compact format with one notification" {
    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'Test message' 2>&1 >/dev/null"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status-panel --format=compact" 2>/dev/null
    [ "$status" -eq 0 ]
    # Should contain bell icon and count
    [[ "$output" == *"🔔"* ]]
    [[ "$output" == *"1"* ]]
}

@test "status-panel detailed format with multiple levels" {
    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=info 'Info' 2>&1 >/dev/null"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=warning 'Warning' 2>&1 >/dev/null"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status-panel --format=detailed" 2>/dev/null
    [ "$status" -eq 0 ]
    [[ "$output" == *"i:1"* ]]
    [[ "$output" == *"w:1"* ]]
}

@test "status-panel count-only format" {
    [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]] && skip "tmux not available"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'Test' 2>&1 >/dev/null"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status-panel --format=count-only" 2>/dev/null
    [ "$status" -eq 0 ]
    [ "$output" = "1" ]
}

@test "status-panel disabled via --enabled=0" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'Test' 2>&1 >/dev/null"
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status-panel --enabled=0" 2>/dev/null
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "status-panel updates tmux option @tmux_intray_active_count" {
    # Ensure tmux option is set after adding notification
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'Test' 2>&1 >/dev/null"
    run tmux -L "$TMUX_SOCKET_NAME" show-options -g @tmux_intray_active_count 2>/dev/null
    [ "$status" -eq 0 ]
    [ "$output" = "@tmux_intray_active_count 1" ]
}

@test "status-panel tmux option updates after dismiss" {
    # Skip due to color codes interfering with ID capture
    skip "ID capture issue with color codes"
    id=$(tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'Test' 2>&1 | tail -1")
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray dismiss $id 2>&1 >/dev/null"
    run tmux -L "$TMUX_SOCKET_NAME" show-options -g @tmux_intray_active_count 2>/dev/null
    [ "$status" -eq 0 ]
    [ "$output" = "@tmux_intray_active_count 0" ]
}
