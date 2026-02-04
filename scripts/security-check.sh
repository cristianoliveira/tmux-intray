#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common library
# shellcheck source=./lib/common.sh disable=SC1091
source "$SCRIPT_DIR/lib/common.sh"

# Security-focused ShellCheck scanning
# - Uses -x to follow sourced files (like lint.sh)
# - Sets severity to info (-S info) to include info-level security issues
# - Excludes SC2034 (unused variable) as it's not a security concern
# - Runs on all shell scripts (.sh, .bats, .tmux) in the project
echo "Running security-focused ShellCheck on all shell scripts..."

run_shellcheck_on_project "$PROJECT_ROOT" -x -S info --exclude=SC2034
