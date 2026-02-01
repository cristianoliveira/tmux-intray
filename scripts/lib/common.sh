#!/usr/bin/env bash
# Common shellcheck utilities for lint and security scripts

set -euo pipefail

# Find all shell script files in the project
# Excludes common temporary and hidden directories
find_shell_scripts() {
    local root_dir="$1"
    find "$root_dir" -type f \( -name "*.sh" -o -name "*.bats" -o -name "*.tmux" \) \
        -not -path "*/.git/*" \
        -not -path "*/.tmp/*" \
        -not -path "*/.bv/*" \
        -not -path "*/.local/*" \
        -not -path "*/tmp/*" \
        -not -path "*/tmp*/*" \
        -not -path "*/.gwt" \
        -not -path "*/.gwt-main" \
        -not -path "*/.direnv/*" \
        -not -path "*/.beads/*" |
        sort
}

# Get extra library files for a given shell script file
# Used to provide shellcheck with sourced files for context
get_extra_files() {
    local file="$1"
    local root_dir="$2"
    local extra_files=()

    if [[ "$file" == *.sh ]]; then
        # Add all lib files
        while IFS= read -r -d '' lib_file; do
            extra_files+=("$lib_file")
        done < <(find "$root_dir/lib" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -not -path "*/.gwt" -not -path "*/.direnv/*" -not -path "*/.beads/*" -print0)

        # For command files, also check their modules
        if echo "$file" | grep -q "commands/[^/]*\.sh$"; then
            # Main command files (not in modules/)
            local cmd_name
            cmd_name=$(basename "$file" .sh)
            local modules_dir="$root_dir/commands/$cmd_name/modules"
            if [[ -d "$modules_dir" ]]; then
                while IFS= read -r -d '' module_file; do
                    extra_files+=("$module_file")
                done < <(find "$modules_dir" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -not -path "*/.gwt" -not -path "*/.direnv/*" -not -path "*/.beads/*" -print0)
            fi
        elif echo "$file" | grep -q "commands/.*/modules/.*\.sh$"; then
            # Module files (in modules/ subdirectory)
            local module_dir
            module_dir=$(dirname "$file")
            while IFS= read -r -d '' module_file; do
                if [[ "$module_file" != "$file" ]]; then
                    extra_files+=("$module_file")
                fi
            done < <(find "$module_dir" -name "*.sh" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -not -path "*/.gwt" -not -path "*/.direnv/*" -not -path "*/.beads/*" -print0)
        fi
    fi

    # Return extra files by printing them null-delimited (safe for spaces)
    if [[ ${#extra_files[@]} -gt 0 ]]; then
        printf '%s\0' "${extra_files[@]}"
    fi
}

# Run shellcheck on all shell scripts with given arguments
# Arguments: root_dir shellcheck_args...
run_shellcheck_on_project() {
    local root_dir="$1"
    shift
    local shellcheck_args=("$@")

    local script_files
    script_files=$(find_shell_scripts "$root_dir")

    while IFS= read -r file; do
        echo "Checking $file..."

        # Get extra files for this script
        local extra_files=()
        while IFS= read -r -d '' extra_file; do
            extra_files+=("$extra_file")
        done < <(get_extra_files "$file" "$root_dir")

        # Run shellcheck
        if [[ ${#extra_files[@]} -eq 0 ]]; then
            if shellcheck "${shellcheck_args[@]}" "$file"; then
                echo "✓ $file passed"
            else
                echo "✗ $file failed"
                return 1
            fi
        else
            if shellcheck "${shellcheck_args[@]}" "$file" "${extra_files[@]}"; then
                echo "✓ $file passed"
            else
                echo "✗ $file failed"
                return 1
            fi
        fi
    done <<<"$script_files"

    echo "All checks passed!"
}
