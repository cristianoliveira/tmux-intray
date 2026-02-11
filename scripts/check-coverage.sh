#!/usr/bin/env bash

set -euo pipefail

# Check coverage threshold
#
# Usage: check-coverage.sh [threshold]
#   threshold: minimum coverage percentage (default: 65)
#
# Exits with 0 if coverage >= threshold, 1 otherwise

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COVERAGE_FILE="${COVERAGE_FILE:-$PROJECT_ROOT/coverage.out}"

# Default threshold
THRESHOLD="${1:-65}"

# Validate threshold is a number
if ! [[ "$THRESHOLD" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
    echo "Error: Threshold must be a number, got '$THRESHOLD'" >&2
    exit 1
fi

# Check if coverage file exists
if [[ ! -f "$COVERAGE_FILE" ]]; then
    echo "Error: Coverage file not found: $COVERAGE_FILE" >&2
    echo "Run 'make go-cover' to generate coverage data." >&2
    exit 1
fi

# Get total coverage percentage using go tool cover
if ! COVERAGE_OUTPUT=$(go tool cover -func="$COVERAGE_FILE" 2>/dev/null); then
    echo "Error: Failed to parse coverage file" >&2
    exit 1
fi

# Extract total coverage percentage from the last line
# Last line format: "total:                                              (statements)                68.5%"
TOTAL_LINE=$(echo "$COVERAGE_OUTPUT" | tail -n 1)
if [[ -z "$TOTAL_LINE" ]]; then
    echo "Error: Could not extract total coverage from output" >&2
    exit 1
fi

# Extract percentage using awk: find the last field that ends with %
PERCENTAGE=$(echo "$TOTAL_LINE" | awk '{ for (i=NF; i>0; i--) if ($i ~ /%$/) { sub(/%/, "", $i); print $i; exit } }')
if [[ -z "$PERCENTAGE" ]]; then
    echo "Error: Could not parse percentage from line: $TOTAL_LINE" >&2
    exit 1
fi

# Compare using bc (supports floating point)
if (($(echo "$PERCENTAGE >= $THRESHOLD" | bc -l))); then
    echo "✓ Coverage: $PERCENTAGE% >= $THRESHOLD%"
    exit 0
else
    echo "✗ Coverage: $PERCENTAGE% < $THRESHOLD%"
    exit 1
fi
