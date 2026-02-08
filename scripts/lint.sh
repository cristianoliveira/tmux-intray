#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common library
# shellcheck source=./lib/common.sh disable=SC1091
source "$SCRIPT_DIR/lib/common.sh"

echo "Running ShellCheck on all shell scripts..."

run_shellcheck_on_project "$PROJECT_ROOT" -x

echo "Checking error message format..."

if rg -n --glob "*.go" --glob "!**/.tmp/**" --glob "!**/.gwt/**" 'fmt\.Errorf\("[A-Z]' "$PROJECT_ROOT"; then
    echo "Error: fmt.Errorf messages must start with lower-case text."
    exit 1
fi

if rg -n --glob "*.go" --glob "!**/.tmp/**" --glob "!**/.gwt/**" 'errors\.New\("[A-Z]' "$PROJECT_ROOT"; then
    echo "Error: errors.New messages must start with lower-case text."
    exit 1
fi

if rg -n --glob "*.go" --glob "!**/.tmp/**" --glob "!**/.gwt/**" 'colors\.Error\("[A-Z]' "$PROJECT_ROOT"; then
    echo "Error: colors.Error messages must start with lower-case text."
    exit 1
fi
