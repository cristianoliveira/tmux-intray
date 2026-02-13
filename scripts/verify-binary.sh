#!/usr/bin/env bash
# Verify that the tmux-intray binary is working correctly

set -euo pipefail

echo "=== Binary Verification for CI ==="
echo

# Check if a custom binary path is set
if [ -n "${TMUX_INTRAY_BIN:-}" ]; then
    echo "✓ TMUX_INTRAY_BIN is set to: $TMUX_INTRAY_BIN"

    # Check if binary exists
    if [ ! -x "$TMUX_INTRAY_BIN" ]; then
        echo "✗ ERROR: Binary does not exist or is not executable: $TMUX_INTRAY_BIN"
        exit 1
    fi
    echo "✓ Binary exists and is executable"

    # Verify Go binary returns proper version
    go_output=$("$TMUX_INTRAY_BIN" --version 2>&1)
    if [[ "$go_output" == *"tmux-intray version"* ]]; then
        echo "✓ Go binary reports valid version information: $go_output"
    else
        echo "✗ ERROR: Go binary version output unexpected: $go_output"
        exit 1
    fi
else
    echo "✓ Using tmux-intray from PATH"

    # Verify tmux-intray command works
    if ! command -v tmux-intray >/dev/null 2>&1; then
        echo "✗ ERROR: tmux-intray not found in PATH"
        exit 1
    fi

    # Verify tmux-intray returns proper version
    go_output=$(tmux-intray --version 2>&1)
    if [[ "$go_output" == *"tmux-intray version"* ]]; then
        echo "✓ tmux-intray reports valid version information: $go_output"
    else
        echo "✗ ERROR: tmux-intray --version output unexpected: $go_output"
        exit 1
    fi
fi

echo
echo "=== Binary Verification Complete ==="
