#!/usr/bin/env bats
# Status command integration tests for format extension

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-status"
    XDG_STATE_HOME="$(mktemp -d)"
    export XDG_STATE_HOME
    XDG_CONFIG_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME

    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.2
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.2
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
}

@test "status preset formats work end-to-end" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --session alpha --window win-a --pane pane-1 --level=info 'info-message' >/dev/null"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --session alpha --window win-a --pane pane-2 --level=warning 'warning-message' >/dev/null"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --session beta --window win-b --pane pane-3 --level=critical 'critical-message' >/dev/null"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=compact"
    [ "$status" -eq 0 ]
    [ "$output" = "[3] critical-message" ]

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=detailed"
    [ "$status" -eq 0 ]
    [ "$output" = "3 unread, 0 read | Latest: critical-message" ]

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=json"
    [ "$status" -eq 0 ]
    [ "$output" = '{"unread":3,"total":3,"message":"critical-message"}' ]

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=count-only"
    [ "$status" -eq 0 ]
    [ "$output" = "3" ]

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=levels"
    [ "$status" -eq 0 ]
    [ "$output" = "Severity: 1 | Unread: 3" ]

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=panes"
    [ "$status" -eq 0 ]
    [ "$output" = "alpha:win-a:pane-1,alpha:win-a:pane-2,beta:win-b:pane-3 (3)" ]
}

@test "status resolves all 13 template variables" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --session alpha --window win-a --pane pane-1 --level=info 'msg-one' >/dev/null"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --session beta --window win-b --pane pane-2 --level=warning 'msg-two' >/dev/null"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --session beta --window win-c --pane pane-3 --level=critical 'msg-three' >/dev/null"
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray dismiss 1 >/dev/null"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${unread-count}|\${total-count}|\${read-count}|\${active-count}|\${dismissed-count}|\${latest-message}|\${has-unread}|\${has-active}|\${has-dismissed}|\${highest-severity}|\${session-list}|\${window-list}|\${pane-list}'"
    [ "$status" -eq 0 ]
    [ "$output" = "2|2|1|2|1|msg-three|true|true|true|1|beta|win-b,win-c|beta:win-b:pane-2,beta:win-c:pane-3" ]
}

@test "status booleans render as true false strings" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='\${has-unread}|\${has-active}|\${has-dismissed}'"
    [ "$status" -eq 0 ]
    [ "$output" = "false|false|false" ]
}

@test "status custom template with multiple variables" {
    tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray add --level=critical 'critical-message' >/dev/null"

    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format='critical=\${critical-count} unread=\${unread-count} latest=\${latest-message}'"
    [ "$status" -eq 0 ]
    [ "$output" = "critical=1 unread=1 latest=critical-message" ]
}

@test "status invalid variable returns helpful error" {
    # shellcheck disable=SC2016
    run ./tmux-intray status --format='${unknown-var}' 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" == *"unknown variable"* ]]
    [[ "$output" == *"unknown-var"* ]]
    [[ "$output" == *"supported"* ]]
}

@test "status unknown preset returns helpful error" {
    run ./tmux-intray status --format=not-a-preset 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" == *"unknown format or template"* ]]
    [[ "$output" == *"not-a-preset"* ]]
}

@test "status exit codes for success and error paths" {
    run tmux -L "$TMUX_SOCKET_NAME" run-shell "$PWD/tmux-intray status --format=count-only"
    [ "$status" -eq 0 ]

    # shellcheck disable=SC2016
    run ./tmux-intray status --format='${critical_count}' 2>&1
    [ "$status" -eq 1 ]
}
