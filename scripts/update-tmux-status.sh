#!/usr/bin/env bash
# Update tmux status option with current notification count

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source storage library
# shellcheck source=../lib/storage.sh disable=SC1091
source "$PROJECT_ROOT/lib/storage.sh"

# Update tmux status option
_update_tmux_status
