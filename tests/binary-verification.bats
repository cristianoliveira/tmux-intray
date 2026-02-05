#!/usr/bin/env bats
# Test to verify that CI is actually testing the correct binary

@test "verify tests-go uses Go binary" {
    # Build the Go binary first
    run make go-build
    [ "$status" -eq 0 ]
    
    # Run with TMUX_INTRAY_BIN set like tests-go does
    run bash -c "TMUX_INTRAY_BIN=./bin/tmux-intray-go ./bin/tmux-intray version 2>&1"
    
    [ "$status" -eq 0 ]
    # The wrapper should indicate it's executing the Go binary
    [[ "$output" == *"Executing ./bin/tmux-intray-go"* ]] || {
        echo "ERROR: tests-go target is NOT actually running the Go binary!"
        echo "Expected to find 'Executing ./bin/tmux-intray-go' in output"
        echo "Output was: $output"
        exit 1
    }
}

@test "verify tests uses bash wrapper" {
    # Run without TMUX_INTRAY_BIN like tests does
    # Clear any TMUX_INTRAY_BIN that might be set
    run bash -c "unset TMUX_INTRAY_BIN && ./bin/tmux-intray version 2>&1"
    
    [ "$status" -eq 0 ]
    # Should NOT show the "Executing" message
    if [[ "$output" == *"Executing"* ]]; then
        echo "ERROR: tests target is unexpectedly executing a binary!"
        echo "Output was: $output"
        exit 1
    fi
}

@test "verify no fallback to bash when Go binary is specified" {
    # Build the Go binary
    run make go-build
    [ "$status" -eq 0 ]
    
    # Run with TMUX_INTRAY_BIN
    run bash -c "TMUX_INTRAY_BIN=./bin/tmux-intray-go ./bin/tmux-intray --help 2>&1"
    
    [ "$status" -eq 0 ]
    
    # Must show the execution message
    [[ "$output" == *"Executing ./bin/tmux-intray-go"* ]] || {
        echo "ERROR: Binary verification failed!"
        echo "The Go binary is not being executed despite TMUX_INTRAY_BIN being set"
        exit 1
    }
}