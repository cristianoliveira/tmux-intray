#!/usr/bin/env bash
# Example: Integrating tmux-intray into a CI/CD pipeline

# Check if tmux is running
if ! tmux list-sessions 2>/dev/null | grep -q .; then
    echo "Error: No tmux session running"
    echo "Start tmux first: tmux new-session"
    exit 1
fi

# Check if tmux-intray is available
if ! command -v tmux-intray &>/dev/null; then
    echo "Error: tmux-intray not found"
    echo "Install it first: go install github.com/cristianoliveira/tmux-intray@latest"
    exit 1
fi

PROJECT="${1:-MyProject}"

echo "Running CI pipeline for $PROJECT..."

# Stage 1: Build
echo "Building..."
tmux-intray add "📦 Building $PROJECT..."
sleep 2

# Simulate build result
BUILD_RESULT=$RANDOM
if [[ $BUILD_RESULT -lt 20 ]]; then
    tmux-intray add "🔴 Build FAILED for $PROJECT"
    echo "Build failed!"
    exit 1
else
    tmux-intray add "✅ Build SUCCESS for $PROJECT"
fi

# Stage 2: Tests
echo "Running tests..."
tmux-intray add "🧪 Running tests for $PROJECT..."
sleep 2

# Simulate test result
TEST_RESULT=$RANDOM
if [[ $TEST_RESULT -lt 20 ]]; then
    tmux-intray add "🔴 Tests FAILED for $PROJECT"
    echo "Tests failed!"
    exit 1
else
    tmux-intray add "✅ Tests PASSED for $PROJECT"
fi

echo ""
echo "Pipeline complete! Check your tmux-intray:"
tmux-intray list
