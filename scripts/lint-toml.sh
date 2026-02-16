#!/usr/bin/env bash

# TOML Configuration File Linter
# Enforces snake_case naming convention for all TOML keys and section headers
#
# Exit codes:
#   0 = All TOML files passed validation
#   1 = One or more TOML files have naming violations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Try to get project root via git rev-parse (most accurate)
if git rev-parse --show-toplevel &>/dev/null; then
    PROJECT_ROOT="$(git rev-parse --show-toplevel)"
else
    # Fallback: parent of scripts/ directory
    PROJECT_ROOT="${SCRIPT_DIR%/scripts}"
    if [[ "$PROJECT_ROOT" == "$SCRIPT_DIR" ]]; then
        # scripts dir not in path, try one level up
        PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
    fi
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

# Helper function to validate a single segment (key or section part)
# Arguments:
#   $1 = segment text (without quotes)
#   $2 = line number
#   $3 = context (e.g., "Key", "Section")
validate_segment() {
    local segment="$1"
    local line_num="$2"
    local context="$3"

    # Check if segment matches snake_case pattern
    if [[ "$segment" =~ $SNAKE_CASE_PATTERN ]]; then
        return 0
    fi

    # Determine violation type and suggest correction
    local suggestion=""
    if [[ "$segment" =~ $CAMEL_CASE_PATTERN ]] || [[ "$segment" =~ $PASCAL_CASE_PATTERN ]]; then
        suggestion=" (camelCase/PascalCase detected, use snake_case)"
    elif [[ "$segment" == *"-"* ]]; then
        # Replace hyphens with underscores for suggestion
        local corrected="${segment//-/_}"
        suggestion=" (kebab-case detected, use snake_case: '$corrected')"
    else
        # Other invalid characters (spaces, special chars, etc.)
        suggestion=" (invalid characters, use snake_case: only lowercase letters, digits, underscores)"
    fi

    echo -e "${YELLOW}  Line $line_num: $context '$segment' should use snake_case$suggestion${NC}"
    return 1
}

# Helper to parse a TOML key expression (bare, quoted, or dotted) into segments
# Arguments:
#   $1 = raw key expression (string up to '=')
# Output: prints segments separated by newlines (use process substitution)
parse_key_expression() {
    local expr="$1"
    # Remove leading/trailing whitespace
    expr="$(echo -n "$expr" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    # Initialize array (bash 4+)
    local segments=()
    local segment=""
    local inside_quotes=""
    local quote_char=""

    # Simple state machine to parse bare/quoted segments separated by '.'
    while [[ -n "$expr" ]]; do
        local char="${expr:0:1}"
        local remaining="${expr:1}"

        if [[ -z "$inside_quotes" ]]; then
            if [[ "$char" == "." ]]; then
                # End of segment
                segments+=("$segment")
                segment=""
                expr="$remaining"
                continue
            elif [[ "$char" == "\"" || "$char" == "'" ]]; then
                inside_quotes=1
                quote_char="$char"
                expr="$remaining"
                continue
            else
                segment+="$char"
                expr="$remaining"
                continue
            fi
        else
            if [[ "$char" == "$quote_char" ]]; then
                # Closing quote
                inside_quotes=""
                quote_char=""
                expr="$remaining"
                continue
            else
                segment+="$char"
                expr="$remaining"
                continue
            fi
        fi
    done

    # Add last segment
    if [[ -n "$segment" ]]; then
        segments+=("$segment")
    fi

    # Print each segment on a new line
    for seg in "${segments[@]}"; do
        echo "$seg"
    done
}

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

        # Strip inline comments (simple - assumes '#' not inside strings)
        line="${line%%#*}"
        # Skip empty lines and comments
        [[ -z "${line// /}" ]] && continue

        # Handle [section] headers - must be snake_case
        if [[ "$line" =~ ^[[:space:]]*\[([^]]+)\][[:space:]]*$ ]]; then
            local section="${BASH_REMATCH[1]}"
            # Parse section expression (may contain dots, quoted parts)
            while IFS= read -r segment; do
                if [[ -n "$segment" ]]; then
                    if ! validate_segment "$segment" "$line_num" "Section"; then
                        file_violations=$((file_violations + 1))
                    fi
                fi
            done < <(parse_key_expression "$section")
            continue
        fi

        # Handle key = value assignments - key must be snake_case
        if [[ "$line" =~ ^[[:space:]]*([^=]+)= ]]; then
            local raw_key="${BASH_REMATCH[1]}"
            # Remove trailing whitespace before =
            raw_key="${raw_key%"${raw_key##*[![:space:]]}"}"
            # Parse key expression (may contain dots, quoted parts)
            while IFS= read -r segment; do
                if [[ -n "$segment" ]]; then
                    if ! validate_segment "$segment" "$line_num" "Key"; then
                        file_violations=$((file_violations + 1))
                    fi
                fi
            done < <(parse_key_expression "$raw_key")
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

    # If files are provided as arguments, check only those
    local toml_files_newline
    if [[ $# -gt 0 ]]; then
        # Convert arguments to newline-separated list
        toml_files_newline=$(printf '%s\n' "$@")
    else
        # Find all TOML files in the project
        # Note: We exclude specific dot-directories to avoid excluding the project itself if it's inside .gwt
        toml_files_newline=$(find "$PROJECT_ROOT" \
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
    fi

    if [[ -z "$toml_files_newline" ]]; then
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
    done <<<"$toml_files_newline"

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
