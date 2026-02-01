#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Security-focused ShellCheck scanning
# - Uses -x to follow sourced files (like lint.sh)
# - Sets severity to warning (-S warning) to focus on security-relevant issues
# - Excludes SC2034 (unused variable) as it's not a security concern
# - Runs on all shell scripts (.sh, .bats, .tmux) in the project
echo "Running security-focused ShellCheck on all shell scripts..."

find "$PROJECT_ROOT" -type f \( -name "*.sh" -o -name "*.bats" -o -name "*.tmux" \) -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -not -path "*/.gwt/*" -not -path "*/.direnv/*" -not -path "*/.beads/*" | sort | while read -r file; do
    echo "Checking $file..."

    # Collect all library files for shellcheck to use
    extra_files=()
    if [[ "$file" == *.sh ]]; then
        # Add all lib files
        while IFS= read -r -d '' lib_file; do
            extra_files+=("$lib_file")
        done < <(find "$PROJECT_ROOT/lib" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -not -path "*/.gwt/*" -not -path "*/.direnv/*" -not -path "*/.beads/*" -print0)

        # For command files, also check their modules
        if echo "$file" | grep -q "commands/[^/]*\.sh$"; then
            # Main command files (not in modules/)
            cmd_name=$(basename "$file" .sh)
            modules_dir="$PROJECT_ROOT/commands/$cmd_name/modules"
            if [[ -d "$modules_dir" ]]; then
                while IFS= read -r -d '' module_file; do
                    extra_files+=("$module_file")
                done < <(find "$modules_dir" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -not -path "*/.gwt/*" -not -path "*/.direnv/*" -not -path "*/.beads/*" -print0)
            fi
        elif echo "$file" | grep -q "commands/.*/modules/.*\.sh$"; then
            # Module files (in modules/ subdirectory)
            module_dir=$(dirname "$file")
            while IFS= read -r -d '' module_file; do
                if [[ "$module_file" != "$file" ]]; then
                    extra_files+=("$module_file")
                fi
            done < <(find "$module_dir" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -not -path "*/.gwt/*" -not -path "*/.direnv/*" -not -path "*/.beads/*" -print0)
        fi
    fi

    if [[ ${#extra_files[@]} -eq 0 ]]; then
        if shellcheck -x -S warning --exclude=SC2034 "$file"; then
            echo "✓ $file passed security check"
        else
            echo "✗ $file failed security check"
            exit 1
        fi
    else
        if shellcheck -x -S warning --exclude=SC2034 "$file" "${extra_files[@]}"; then
            echo "✓ $file passed security check"
        else
            echo "✗ $file failed security check"
            exit 1
        fi
    fi
done

echo "All security checks passed!"
