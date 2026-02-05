#!/usr/bin/env bash
# Verify that CI is actually testing the correct binary

set -euo pipefail

echo "=== Binary Verification for CI ==="
echo

# Check if we're supposed to be testing Go binary
if [ -n "${TMUX_INTRAY_BIN:-}" ]; then
    echo "✓ TMUX_INTRAY_BIN is set to: $TMUX_INTRAY_BIN"

    # Check if the binary exists
    if [ ! -x "$TMUX_INTRAY_BIN" ]; then
        echo "✗ ERROR: Binary does not exist or is not executable: $TMUX_INTRAY_BIN"
        exit 1
    fi
    echo "✓ Binary exists and is executable"

    # Check if running actually executes the Go binary
    echo "Testing if Go binary is actually being executed..."
    output=$(./bin/tmux-intray version 2>&1)

    if [[ "$output" == *"Executing $TMUX_INTRAY_BIN"* ]]; then
        echo "✓ Verified: Go binary is being executed"
    else
        echo "✗ ERROR: Go binary is NOT being executed!"
        echo "Expected to find 'Executing $TMUX_INTRAY_BIN' in output"
        echo "Output was: $output"
        exit 1
    fi
else
    echo "✓ TMUX_INTRAY_BIN is not set - using bash wrapper"

    # Verify we're using the bash wrapper
    output=$(./bin/tmux-intray version 2>&1)

    if [[ "$output" == *"Executing"* ]]; then
        echo "✗ ERROR: Binary is being executed despite TMUX_INTRAY_BIN not being set!"
        echo "Output was: $output"
        exit 1
    fi
    echo "✓ Verified: Bash wrapper is being used"
fi

echo
echo "=== Binary Verification Complete ==="
