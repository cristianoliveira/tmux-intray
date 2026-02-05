#!/usr/bin/env bats
# Test tmux-intray command line interface

@test "unknown command returns error" {
    # Ensure the Go binary exists
    if [ ! -f "./tmux-intray" ]; then
        skip "tmux-intray binary not found (run 'make go-build' first)"
    fi

    run ./tmux-intray unknown
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown command"* ]]
}

@test "add without message returns error" {
    # Ensure the Go binary exists
    if [ ! -f "./tmux-intray" ]; then
        skip "tmux-intray binary not found (run 'make go-build' first)"
    fi

    run ./tmux-intray add
    [ "$status" -eq 1 ]
    [[ "$output" == *"requires a message"* ]]
}

@test "commands work when invoked from different working directory" {
    # Ensure the Go binary exists
    if [ ! -f "./tmux-intray" ]; then
        skip "tmux-intray binary not found (run 'make go-build' first)"
    fi

    # Create a temporary directory outside the project
    tmpdir="$(mktemp -d)"
    cd "$tmpdir" || exit 1
    # Copy the Go binary to test directory
    cp "$OLDPWD/tmux-intray" .
    # Run a simple command that doesn't require tmux
    run ./tmux-intray --help
    [ "$status" -eq 0 ]
    # Check for Go output format
    [[ "$output" == *"Usage:"* ]] || {
        echo "ERROR: Expected to find 'Usage:' in help output"
        echo "Output was: $output"
        exit 1
    }
    # Cleanup
    cd "$OLDPWD"
    rm -rf "$tmpdir"
}
