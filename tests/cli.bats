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
