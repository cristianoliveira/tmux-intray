#!/usr/bin/env bats
# Test basic tmux-intray functionality

@test "tmux-intray shows help" {
    run ./bin/tmux-intray help
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray"* ]]
}

@test "tmux-intray shows version" {
    run ./bin/tmux-intray version
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux-intray"* ]]
}
