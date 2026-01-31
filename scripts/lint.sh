#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Running ShellCheck on all shell scripts..."

find "$PROJECT_ROOT" -type f \( -name "*.sh" -o -name "*.bats" -o -name "*.tmux" \) -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" | sort | while read -r file; do
    echo "Checking $file..."

    # Collect all library files for shellcheck to use
    extra_files=()
    if [[ "$file" == *.sh ]]; then
        # Add all lib files
        while IFS= read -r -d '' lib_file; do
            extra_files+=("$lib_file")
        done < <(find "$PROJECT_ROOT/lib" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0)

        # For command files, also check their modules
        if echo "$file" | grep -q "commands/[^/]*\.sh$"; then
            # Main command files (not in modules/)
            cmd_name=$(basename "$file" .sh)
            modules_dir="$PROJECT_ROOT/commands/$cmd_name/modules"
            if [[ -d "$modules_dir" ]]; then
                while IFS= read -r -d '' module_file; do
                    extra_files+=("$module_file")
                done < <(find "$modules_dir" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0)
            fi
        elif echo "$file" | grep -q "commands/.*/modules/.*\.sh$"; then
            # Module files (in modules/ subdirectory)
            module_dir=$(dirname "$file")
            while IFS= read -r -d '' module_file; do
                if [[ "$module_file" != "$file" ]]; then
                    extra_files+=("$module_file")
                fi
            done < <(find "$module_dir" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0)
        fi
    fi

    if [[ ${#extra_files[@]} -eq 0 ]]; then
        if shellcheck -x "$file"; then
            echo "✓ $file passed"
        else
            echo "✗ $file failed"
            exit 1
        fi
    else
        if shellcheck -x "$file" "${extra_files[@]}"; then
            echo "✓ $file passed"
        else
            echo "✗ $file failed"
            exit 1
        fi
    fi
done

echo "All checks passed!"
