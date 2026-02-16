#!/usr/bin/env bash

# TOML Configuration File Linter
# Enforces snake_case naming convention for all TOML keys and section headers
#
# Exit codes:
#   0 = All TOML files passed validation
#   1 = One or more TOML files have naming violations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# PROJECT_ROOT is the parent of scripts/ directory
# Handle both normal case (scripts is a subdirectory) and special cases
PROJECT_ROOT="${SCRIPT_DIR%/scripts}"
if [[ "$PROJECT_ROOT" == "$SCRIPT_DIR" ]]; then
    # scripts dir not in path, try one level up
    PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
fi

# Colors for output (matching project style)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Track if any violations were found
VIOLATIONS_FOUND=0

# snake_case pattern: lowercase letters, digits, and underscores only
# Cannot start with underscore or digit, cannot end with underscore
# Must contain at least one character
SNAKE_CASE_PATTERN='^[a-z][a-z0-9_]*[a-z0-9]$|^[a-z]$'

# camelCase pattern: starts lowercase, has uppercase letters
CAMEL_CASE_PATTERN='[a-z][a-z0-9]*[A-Z]'

# PascalCase pattern: starts uppercase
PASCAL_CASE_PATTERN='^[A-Z]'

# check_toml_file validates a single TOML file for naming conventions
# Arguments:
#   $1 = path to TOML file
check_toml_file() {
    local file="$1"
    local file_violations=0

    # Check if file exists and is readable
    if [[ ! -f "$file" ]] || [[ ! -r "$file" ]]; then
        echo -e "${RED}✗ Error: Cannot read file: $file${NC}" >&2
        VIOLATIONS_FOUND=1
        return 1
    fi

    # Parse TOML for keys (lines with = assignments or [section] headers)
    # This is a simple pattern-based approach, not a full TOML parser
    while IFS= read -r line_with_num; do
        # Extract line number and content (grep -n outputs "linenum:content")
        if [[ "$line_with_num" =~ ^([0-9]+):(.*)$ ]]; then
            local line_num="${BASH_REMATCH[1]}"
            local line="${BASH_REMATCH[2]}"
        else
            continue
        fi

        # Skip empty lines and comments
        [[ -z "${line// /}" ]] && continue
        [[ "$line" =~ ^[[:space:]]*# ]] && continue

        # Handle [section] headers - must be snake_case
        if [[ "$line" =~ ^[[:space:]]*\[([a-zA-Z0-9_\.\-]+)\][[:space:]]*$ ]]; then
            local section="${BASH_REMATCH[1]}"
            # Replace dots with underscores for nested sections (group_header becomes valid)
            local section_normalized="${section//./_}"

            # Check if section follows snake_case (after normalizing dots)
            if ! [[ "$section_normalized" =~ $SNAKE_CASE_PATTERN ]]; then
                if [[ "$section_normalized" =~ $CAMEL_CASE_PATTERN ]] || [[ "$section_normalized" =~ $PASCAL_CASE_PATTERN ]]; then
                    echo -e "${YELLOW}  Line $line_num: Section '[$section]' should use snake_case${NC}"
                    file_violations=$((file_violations + 1))
                fi
            fi
            continue
        fi

        # Handle key = value assignments - key must be snake_case
        if [[ "$line" =~ ^[[:space:]]*([a-zA-Z_][a-zA-Z0-9_\-]*)[[:space:]]*= ]]; then
            local key="${BASH_REMATCH[1]}"

            # Check if key follows snake_case
            if ! [[ "$key" =~ $SNAKE_CASE_PATTERN ]]; then
                if [[ "$key" =~ $CAMEL_CASE_PATTERN ]] || [[ "$key" =~ $PASCAL_CASE_PATTERN ]]; then
                    echo -e "${YELLOW}  Line $line_num: Key '$key' should use snake_case${NC}"
                    file_violations=$((file_violations + 1))
                fi
            fi
        fi
    done < <(grep -n . "$file")

    if [[ $file_violations -gt 0 ]]; then
        echo -e "${RED}✗ $file: Found $file_violations naming violation(s)${NC}" >&2
        VIOLATIONS_FOUND=1
        return 1
    fi

    return 0
}

# Main linting logic
main() {
    echo "Checking TOML file naming conventions..."

    # Find all TOML files in the project
    # Note: We exclude specific dot-directories to avoid excluding the project itself if it's inside .gwt
    local toml_files
    toml_files=$(find "$PROJECT_ROOT" \
        -type f \
        -name "*.toml" \
        -not -path "*/.git/*" \
        -not -path "*/.tmp/*" \
        -not -path "*/_tmp/*" \
        -not -path "*/.bv/*" \
        -not -path "*/.local/*" \
        -not -path "*/tmp/*" \
        -not -path "*/vendor/*" \
        -not -path "*/.direnv/*" \
        -not -path "*/.beads/*" \
        -not -path "*/tests/fixtures/*" \
        2>/dev/null || true)

    if [[ -z "$toml_files" ]]; then
        echo -e "${GREEN}✓ No TOML files found to check${NC}"
        return 0
    fi

    local files_checked=0
    local files_passed=0

    while IFS= read -r toml_file; do
        # Skip empty lines
        [[ -z "$toml_file" ]] && continue
        files_checked=$((files_checked + 1))

        if check_toml_file "$toml_file"; then
            echo -e "${GREEN}✓ $toml_file${NC}"
            files_passed=$((files_passed + 1))
        fi
    done <<<"$toml_files"

    echo ""
    if [[ $files_checked -eq 0 ]]; then
        echo -e "${GREEN}✓ No TOML files to check${NC}"
        return 0
    fi

    if [[ $VIOLATIONS_FOUND -eq 0 ]]; then
        echo -e "${GREEN}✓ TOML naming check passed ($files_checked file(s) checked)${NC}"
        return 0
    else
        echo -e "${RED}✗ TOML naming check failed ($files_passed/$files_checked file(s) passed)${NC}"
        return 1
    fi
}

main "$@"
