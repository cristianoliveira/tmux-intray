#!/usr/bin/env bash
# Color utilities for tmux-intray

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

error() {
    echo -e "${RED}Error: $*${NC}" >&2
}

success() {
    echo -e "${GREEN}✓${NC} $*"
}

warning() {
    echo -e "${YELLOW}Warning: $*${NC}" >&2
}

# User-facing info message (stdout)
info() {
    echo -e "${BLUE}$*${NC}"
}

# Logging info message (stderr)
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
