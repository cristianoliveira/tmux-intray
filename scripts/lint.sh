#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common library
# shellcheck source=./lib/common.sh disable=SC1091
source "$SCRIPT_DIR/lib/common.sh"

# Check file line length limits
check_file_length() {
    echo "Checking file line counts (max 500 lines)..."
    cd "$PROJECT_ROOT" || return 1
    local max_lines=500
    local warning_lines=300
    local fail=0
    # Grandfathered large files (to be split eventually)
    local exclude_files=(
        "./internal/tui/state/model_tree.go"
        "./internal/storage/sqlite/storage.go"
        "./cmd/tmux-intray/status-panel.go"
    )
    while IFS= read -r -d '' file; do
        # Skip grandfathered large files
        if [[ " ${exclude_files[*]} " == *" $file "* ]]; then
            echo "Info: Skipping grandfathered large file: $file"
            continue
        fi
        lines=$(wc -l <"$file")
        if [[ $lines -gt $max_lines ]]; then
            echo "Error: $file has $lines lines (exceeds maximum $max_lines)"
            fail=1
        elif [[ $lines -gt $warning_lines ]]; then
            echo "Warning: $file has $lines lines (consider splitting, target <= $warning_lines)"
        fi
    done < <(find . -type f -name "*.go" \
        -not -name "*_test.go" \
        -not -path "*/.git/*" \
        -not -path "*/.tmp/*" \
        -not -path "*/.gwt/*" \
        -not -path "*/.bv/*" \
        -not -path "*/.local/*" \
        -not -path "*/tmp/*" \
        -not -path "*/vendor/*" \
        -not -path "*/integration/*" \
        -not -regex ".*/_gen\.go$" \
        -not -regex ".*/integration/.*\.go$" \
        -print0)
    if [[ $fail -eq 1 ]]; then
        echo "File length check failed. Please split large files."
        exit 1
    fi
    echo "File length check passed."
}

echo "Running ShellCheck on all shell scripts..."

run_shellcheck_on_project "$PROJECT_ROOT" -x

echo "Checking file line length limits..."
check_file_length

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
