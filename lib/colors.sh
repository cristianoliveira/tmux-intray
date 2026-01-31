#!/usr/bin/env bash
# Color utilities for tmux-intray

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

error() {
    echo -e "${RED}Error: $*${NC}" >&2
}

success() {
    echo -e "${GREEN}✓${NC} $*"
}

warning() {
    echo -e "${YELLOW}Warning: $*${NC}"
}

info() {
    echo -e "${BLUE}$*${NC}"
}
