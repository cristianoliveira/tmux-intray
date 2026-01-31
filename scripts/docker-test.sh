#!/usr/bin/env bash
# Docker test runner for tmux-intray
# Builds a Docker image with all dependencies and runs tests or other commands

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="tmux-intray-test"

# Default command if none provided
DEFAULT_CMD=("make" "tests")

usage() {
    cat <<EOF
Usage: $0 [COMMAND] [ARGS...]

Run tmux-intray tests or other commands in an isolated Docker container.

If no command is provided, runs 'make tests'.

Examples:
  $0                    # Run all tests
  $0 make lint         # Run linter
  $0 bats tests        # Run bats directly
  $0 bash              # Start interactive shell in container
EOF
}

main() {
    cd "$PROJECT_ROOT"

    echo "Building Docker image: $IMAGE_NAME"
    docker build -t "$IMAGE_NAME" .

    echo "Running container with command: ${*:-${DEFAULT_CMD[*]}}"
    docker run --rm \
        -v "$PROJECT_ROOT:/home/tester/tmux-intray" \
        -w /home/tester/tmux-intray \
        "$IMAGE_NAME" "$@"
}

if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    usage
    exit 0
fi

main "$@"
