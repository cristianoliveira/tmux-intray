#!/usr/bin/env bats
# Test tmux-intray tray management

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME
    # Create empty config to avoid sample config messages
    mkdir -p "$XDG_CONFIG_HOME/tmux-intray"
    touch "$XDG_CONFIG_HOME/tmux-intray/config.sh"

    # Determine if we can use tmux
    export TMUX_AVAILABLE=0
    export TMUX_SOCKET_PATH=""
    if [[ -z "${CI:-}" ]] && command -v tmux >/dev/null 2>&1; then
        # Clean up any existing server first
        tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
        sleep 1

        # Start a tmux server with a session for testing
        if tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test 2>/dev/null; then
            # Wait for server to be ready and socket path to exist
            local max_retries=5
            local retry=0
            local socket_path=""
            while [[ $retry -lt $max_retries ]]; do
                sleep 0.5
                socket_path=$(tmux -L "$TMUX_SOCKET_NAME" display -p '#{socket_path}' 2>/dev/null)
                if [[ -n "$socket_path" && -S "$socket_path" ]]; then
                    break
                fi
                retry=$((retry + 1))
            done
            if [[ -n "$socket_path" && -S "$socket_path" ]]; then
                export TMUX_SOCKET_PATH="$socket_path"
                # TMUX format: socket_path,client_fd,client_pid
                # We'll fake client_fd and client_pid (not critical for our use)
                export TMUX="$socket_path,12345,0"
                export TMUX_AVAILABLE=1
            else
                echo "warning: tmux socket path missing or not a socket, disabling tmux support" >&2
                export TMUX_AVAILABLE=0
                export TMUX_SOCKET_PATH=""
                tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
            fi
        else
            export TMUX_AVAILABLE=0
            export TMUX_SOCKET_PATH=""
        fi
    fi
}

teardown() {
    if [[ "${TMUX_AVAILABLE:-0}" -eq 1 ]]; then
        tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
        sleep 1
        # Remove socket file if it still exists
        if [[ -n "$TMUX_SOCKET_PATH" && -e "$TMUX_SOCKET_PATH" ]]; then
            rm -f "$TMUX_SOCKET_PATH"
        fi
    fi
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
}

@test "clear tray" {
    if [[ -n "${CI:-}" ]] || [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]]; then
        run "$PWD/tmux-intray" add 'test' 2>&1
    else
        run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add 'test' 2>&1"
    fi
    [ "$status" -eq 0 ]
    [[ "$output" == *"added"* ]]

    if [[ -n "${CI:-}" ]] || [[ "${TMUX_AVAILABLE:-0}" -ne 1 ]]; then
        run "$PWD/tmux-intray" clear 2>&1
    else
        run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray clear 2>&1"
    fi
    [ "$status" -eq 0 ]
    [[ "$output" == *"cleared"* ]]
}
