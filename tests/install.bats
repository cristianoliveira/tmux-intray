#!/usr/bin/env bats
# Test one-click installer

setup() {
    # Create a temporary directory for test
    TEST_DIR="$(mktemp -d)"
    export TEST_DIR
    # Create a fake tarball of the project
    cd "$BATS_TEST_DIRNAME/.."
    tar -czf "$TEST_DIR/tmux-intray-main.tar.gz" \
        --exclude='.git' \
        --exclude='dist' \
        --exclude='.tmp' \
        --exclude='.bv' \
        --exclude='.local' \
        --exclude='tmp' \
        --exclude='*.swp' .
    # Create a mock curl that returns our tarball
    cat >"$TEST_DIR/curl" <<'EOF'
#!/usr/bin/env bash
# Mock curl that handles -o flag
output_file=""
url=""
# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        -o)
            output_file="$2"
            shift 2
            ;;
        -f|-s|-S|-L)
            # Ignore common curl flags
            shift
            ;;
        *)
            # Assume this is the URL
            url="$1"
            shift
            ;;
    esac
done

if [[ "$url" == *"api.github.com/repos/cristianoliveira/tmux-intray/releases/latest"* ]]; then
    # Return empty JSON (no releases)
    if [[ -n "$output_file" ]]; then
        echo '{}' > "$output_file"
    else
        echo '{}'
    fi
elif [[ "$url" == *"archive/refs/heads/main.tar.gz"* ]]; then
    # Serve the local tarball
    if [[ -n "$output_file" ]]; then
        cat "$TEST_DIR/tmux-intray-main.tar.gz" > "$output_file"
    else
        cat "$TEST_DIR/tmux-intray-main.tar.gz"
    fi
else
    # Real curl (should not happen)
    exec /usr/bin/curl "$@"
fi
EOF
    chmod +x "$TEST_DIR/curl"
}

teardown() {
    rm -rf "$TEST_DIR"
}

@test "install.sh dry-run works" {
    # Add mock curl to PATH
    PATH="$TEST_DIR:$PATH"
    # Run install.sh with dry-run
    run ./install.sh --dry-run --prefix "$TEST_DIR/install-prefix"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Dry run enabled"* ]]
}

@test "install.sh actually installs" {
    PATH="$TEST_DIR:$PATH"
    run ./install.sh --prefix "$TEST_DIR/install-prefix"
    [ "$status" -eq 0 ]
    [[ "$output" == *"installed successfully"* ]]
    # Verify binary exists and works
    "$TEST_DIR/install-prefix/bin/tmux-intray" version
    # Clean up
    rm -rf "$TEST_DIR/install-prefix"
}
