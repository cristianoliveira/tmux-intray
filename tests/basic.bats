#!/usr/bin/env bats
# Test basic tmux-intray functionality

@test "tmux-intray shows help" {
    run ./bin/tmux-intray help
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray"* ]] || [[ "$output" == *"USAGE:"* ]]

    # If TMUX_INTRAY_BIN is set, verify we're using the Go binary
    if [ -n "${TMUX_INTRAY_BIN:-}" ]; then
        # Check for either the Executing message or the Go help format
        [[ "$output" == *"Executing ${TMUX_INTRAY_BIN}"* ]] || [[ "$output" == *"USAGE:"* ]] || {
            echo "ERROR: Expected to find 'Executing ${TMUX_INTRAY_BIN}' or 'USAGE:' in output"
            echo "Output was: $output"
            exit 1
        }
    fi
}

@test "tmux-intray shows version" {
    run ./bin/tmux-intray version
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray"* ]]

    # If TMUX_INTRAY_BIN is set, verify we're using the Go binary
    if [ -n "${TMUX_INTRAY_BIN:-}" ]; then
        # Check for either the Executing message or the Go version format
        [[ "$output" == *"Executing ${TMUX_INTRAY_BIN}"* ]] || [[ "$output" == *"tmux-intray v"* ]] || {
            echo "ERROR: Expected to find 'Executing ${TMUX_INTRAY_BIN}' or 'tmux-intray v' in output"
            echo "Output was: $output"
            exit 1
        }
    fi
}
