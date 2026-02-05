#!/usr/bin/env bats
# Test basic tmux-intray functionality

@test "tmux-intray shows help" {
    run ./bin/tmux-intray help
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray"* ]]
    
    # If TMUX_INTRAY_BIN is set, verify we're using the Go binary
    if [ -n "${TMUX_INTRAY_BIN:-}" ]; then
        [[ "$output" == *"Executing ${TMUX_INTRAY_BIN}"* ]] || {
            echo "ERROR: Expected to find 'Executing ${TMUX_INTRAY_BIN}' in output"
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
        [[ "$output" == *"Executing ${TMUX_INTRAY_BIN}"* ]] || {
            echo "ERROR: Expected to find 'Executing ${TMUX_INTRAY_BIN}' in output"
            echo "Output was: $output"
            exit 1
        }
    fi
}
