#!/usr/bin/env bats
# Test to verify that we're running the Go binary directly

@test "verify Go binary exists and works" {
    # Build the Go binary first
    run make go-build
    [ "$status" -eq 0 ]

    # Run the Go binary directly
    run ./tmux-intray version
    [ "$status" -eq 0 ]

    # Should show version information
    [[ "$output" == *"tmux-intray"* ]] || {
        echo "ERROR: tmux-intray binary doesn't show version info!"
        echo "Output was: $output"
        exit 1
    }
}

@test "verify Go binary help works" {
    # Build the Go binary first
    run make go-build
    [ "$status" -eq 0 ]

    # Run the Go binary with help
    run ./tmux-intray --help
    [ "$status" -eq 0 ]

    # Should show help information
    [[ "$output" == *"Usage"* ]] || {
        echo "ERROR: tmux-intray binary doesn't show help!"
        echo "Output was: $output"
        exit 1
    }
}
