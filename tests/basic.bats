#!/usr/bin/env bats
# Test basic tmux-intray functionality

@test "tmux-intray shows help" {
    # Ensure the Go binary exists
    if [ ! -f "./tmux-intray" ]; then
        skip "tmux-intray binary not found (run 'make go-build' first)"
    fi

    run ./tmux-intray help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]] || {
        echo "ERROR: Expected to find 'Usage:' in help output"
        echo "Output was: $output"
        exit 1
    }
}

@test "tmux-intray shows version" {
    # Ensure the Go binary exists
    if [ ! -f "./tmux-intray" ]; then
        skip "tmux-intray binary not found (run 'make go-build' first)"
    fi

    run ./tmux-intray version
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray"* ]] || {
        echo "ERROR: Expected to find 'tmux-intray' in version output"
        echo "Output was: $output"
        exit 1
    }
}
