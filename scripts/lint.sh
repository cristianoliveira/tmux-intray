#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common library
# shellcheck source=./lib/common.sh disable=SC1091
source "$SCRIPT_DIR/lib/common.sh"

echo "Running ShellCheck on all shell scripts..."

run_shellcheck_on_project "$PROJECT_ROOT" -x
