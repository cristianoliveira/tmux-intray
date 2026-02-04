#!/usr/bin/env bash
# Color utilities for tmux-intray
#
# This library provides colorized output functions for user-facing messages
# and internal logging. Functions follow these conventions:
#
# - User-facing messages: stdout, no prefix, colored for visibility
# - Internal logging: stderr, colored but may include prefixes
# - Debug messages: stderr only when TMUX_INTRAY_DEBUG is enabled
#
# Usage guidelines:
# 1. info(): For messages that should be displayed to the user as part of
#    normal operation (e.g., status updates, results). Outputs to stdout.
#    Example: info "Notification added with ID 123"
#
# 2. log_info(): For internal logging/debugging information that should go
#    to stderr (e.g., progress steps, diagnostic info). Use when you want to
#    log something but not show it to the user in normal operation.
#    Currently unused in the codebase but available for future logging needs.
#
# 3. debug(): For debug-level logging only when TMUX_INTRAY_DEBUG is enabled.
#    Use for development/troubleshooting.
#
# 4. success(): For positive confirmation messages (stdout).
#
# 5. warning(): For warning messages that should be visible but non-fatal (stderr).
#
# 6. error(): For error messages that indicate failures (stderr).
#
# Note: stdout vs stderr distinction is important for scripts that need to
# capture clean output (e.g., command output should be only stdout user messages).
# Logging and errors should go to stderr to avoid polluting output.

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Output error message to stderr with red "Error:" prefix
error() {
    echo -e "${RED}Error: $*${NC}" >&2
}

# Output success message to stdout with green checkmark
success() {
    echo -e "${GREEN}✓${NC} $*"
}

# Output warning message to stderr with yellow "Warning:" prefix
warning() {
    echo -e "${YELLOW}Warning: $*${NC}" >&2
}

# User-facing info message (stdout)
# Use for informational messages that should be shown to the user.
info() {
    echo -e "${BLUE}$*${NC}"
}

# Logging info message (stderr)
# Use for internal logging/debugging information.
# Currently unused but available for future logging needs.
log_info() {
    echo -e "${BLUE}$*${NC}" >&2
}

# Debug logging (stderr) when TMUX_INTRAY_DEBUG is enabled
debug() {
    case "${TMUX_INTRAY_DEBUG:-}" in
    1 | true | TRUE | yes | YES | on | ON)
        echo -e "${CYAN}Debug: $*${NC}" >&2
        ;;
    esac
}
