#!/usr/bin/env bats
# Test one-click installer

setup() {
    # Create a temporary directory for test
    TEST_DIR="$(mktemp -d)"
    export TEST_DIR
    # Create a staged copy of the project to avoid tar races
    local stage_dir="$TEST_DIR/stage"
    mkdir -p "$stage_dir"
    cd "$BATS_TEST_DIRNAME/.." || exit
    # Use rsync if available for efficient copying with exclusions
    if command -v rsync >/dev/null 2>&1; then
        rsync -a \
            --exclude='.git' \
            --exclude='dist' \
            --exclude='.tmp' \
            --exclude='.bv' \
            --exclude='.local' \
            --exclude='tmp' \
            --exclude='*.swp' \
            --exclude='bin' \
            . "$stage_dir/"
    else
        # Fallback: copy everything and remove excluded directories
        cp -R . "$stage_dir/"
        rm -rf "$stage_dir/.git" "$stage_dir/dist" "$stage_dir/.tmp" \
            "$stage_dir/.bv" "$stage_dir/.local" "$stage_dir/tmp" 2>/dev/null || true
        find "$stage_dir" -name '*.swp' -delete 2>/dev/null || true
    fi
    # Ensure bin directory exists with the binary (required by installer)
    mkdir -p "$stage_dir/bin"
    # Build and copy the Go binary
    make go-build
    cp -p tmux-intray "$stage_dir/bin/"
    # Create tarball from staged copy
    tar -czf "$TEST_DIR/tmux-intray-main.tar.gz" -C "$stage_dir" .
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
    # shellcheck disable=SC2030,SC2031
    PATH="$TEST_DIR:$PATH"
    # Run install.sh with dry-run
    run ./install.sh --dry-run --prefix "$TEST_DIR/install-prefix"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Dry run enabled"* ]]
}

@test "install.sh actually installs" {
    # shellcheck disable=SC2030,SC2031
    PATH="$TEST_DIR:$PATH"
    run ./install.sh --prefix "$TEST_DIR/install-prefix"
    [ "$status" -eq 0 ]
    [[ "$output" == *"installed successfully"* ]]
    # Verify binary exists and works
    "$TEST_DIR/install-prefix/bin/tmux-intray" --version
    # Clean up
    rm -rf "$TEST_DIR/install-prefix"
}
