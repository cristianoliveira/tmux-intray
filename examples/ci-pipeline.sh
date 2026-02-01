#!/usr/bin/env bash
# Example: Integrating tmux-intray into a CI/CD pipeline

TMUX_INTRAY_BIN="./bin/tmux-intray"
PROJECT="${1:-MyProject}"

echo "Running CI pipeline for $PROJECT..."

# Stage 1: Build
echo "Building..."
"$TMUX_INTRAY_BIN" add "📦 Building $PROJECT..."
sleep 2

# Simulate build result
BUILD_RESULT=$RANDOM
if [[ $BUILD_RESULT -lt 20 ]]; then
    "$TMUX_INTRAY_BIN" add "🔴 Build FAILED for $PROJECT"
    echo "Build failed!"
    exit 1
else
    "$TMUX_INTRAY_BIN" add "✅ Build SUCCESS for $PROJECT"
fi

# Stage 2: Tests
echo "Running tests..."
"$TMUX_INTRAY_BIN" add "🧪 Running tests for $PROJECT..."
sleep 2

# Simulate test result
TEST_RESULT=$RANDOM
if [[ $TEST_RESULT -lt 20 ]]; then
    "$TMUX_INTRAY_BIN" add "🔴 Tests FAILED for $PROJECT"
    echo "Tests failed!"
    exit 1
else
    "$TMUX_INTRAY_BIN" add "✅ Tests PASSED for $PROJECT"
fi

echo ""
echo "Pipeline complete! Check your tmux-intray:"
"$TMUX_INTRAY_BIN" list
