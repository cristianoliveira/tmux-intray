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
    # Run a simple command that doesn't require tmux
    run "$OLDPWD/bin/tmux-intray" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray v"* ]]
    # Cleanup
    cd "$OLDPWD"
    rmdir "$tmpdir"
}
