#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Running ShellCheck on all shell scripts..."

find "$PROJECT_ROOT" -type f \( -name "*.sh" -o -name "*.bats" -o -name "*.tmux" \) -not -path "*/.git/*" | while read -r file; do
    echo "Checking $file..."
    if shellcheck -x "$file"; then
        echo "✓ $file passed"
    else
        echo "✗ $file failed"
        exit 1
    fi
done

echo "All checks passed!"
