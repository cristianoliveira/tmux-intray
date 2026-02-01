#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Create a temporary directory for test installation
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TEST_DIR"' EXIT

# Create a tarball mimicking a release
echo "Creating test tarball..."
cd "$PROJECT_ROOT"
tar -czf "$TEST_DIR/tmux-intray-test.tar.gz" \
    --exclude='.git' \
    --exclude='dist' \
    --exclude='.tmp' \
    --exclude='.bv' \
    --exclude='.local' \
    --exclude='tmp' \
    --exclude='*.swp' \
    .

# Create a mock install.sh that uses our local tarball
cat >"$TEST_DIR/install.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Override download_latest_release to use local tarball
download_latest_release() {
    local temp_dir="$1"
    local tarball_path="$temp_dir/tmux-intray-test.tar.gz"
    cp "$TEST_TARBALL" "$tarball_path"
    echo "$tarball_path"
}

# Source the real install.sh after overriding the function
# We'll extract the original install.sh and source it
EOF

# We need to inject the function into the real install.sh
# Simpler: run install.sh with a fake curl that returns our tarball
# Let's create a wrapper script that overrides curl
cat >"$TEST_DIR/curl-fake" <<'EOF'
#!/usr/bin/env bash
if [[ "$*" == *"api.github.com"* ]]; then
    # Return empty JSON (no releases)
    echo '{}'
elif [[ "$*" == *"archive/refs/heads/main.tar.gz"* ]]; then
    # Serve the local tarball
    cat "$TEST_TARBALL"
else
    # Real curl (should not happen)
    /usr/bin/curl "$@"
fi
EOF
chmod +x "$TEST_DIR/curl-fake"

export PATH="$TEST_DIR:$PATH"
export TEST_TARBALL="$TEST_DIR/tmux-intray-test.tar.gz"

# Run the real install.sh with a test prefix
echo "Running install.sh with test prefix..."
INSTALL_PREFIX="$TEST_DIR/install-prefix"
cd "$TEST_DIR"
"$PROJECT_ROOT/install.sh" --prefix "$INSTALL_PREFIX"

# Verify installation
echo "Verifying installation..."
"$INSTALL_PREFIX/bin/tmux-intray" version
echo "Installation test passed!"
