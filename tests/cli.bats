#!/usr/bin/env bats
# Test tmux-intray command line interface

@test "unknown command returns error" {
    run ./bin/tmux-intray unknown
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown command"* ]]
}

@test "add without message returns error" {
    run ./bin/tmux-intray add
    [ "$status" -eq 1 ]
    [[ "$output" == *"requires a message"* ]]
}

@test "commands work when invoked from different working directory" {
    # Create a temporary directory outside the project
    tmpdir="$(mktemp -d)"
    cd "$tmpdir" || exit 1
    # Copy both wrapper and Go binary to test directory
    cp "$OLDPWD/bin/tmux-intray" .
    if [[ -x "$OLDPWD/bin/tmux-intray-go" ]]; then
        cp "$OLDPWD/bin/tmux-intray-go" .
    fi
    # Run a simple command that doesn't require tmux
    run ./tmux-intray --help
    [ "$status" -eq 0 ]
    # Check for either Go or bash output format
    [[ "$output" == *"tmux-intray v"* ]] || [[ "$output" == *"tmux-intray"* ]] || [[ "$output" == *"Usage:"* ]]
    # Cleanup
    cd "$OLDPWD"
    rm -rf "$tmpdir"
}
