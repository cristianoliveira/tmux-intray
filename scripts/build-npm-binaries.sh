#!/usr/bin/env bash
#
# Build all platform binaries for npm package
# This script cross-compiles tmux-intray for all supported platforms
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}INFO:${NC} $*"
}

log_error() {
    echo -e "${RED}ERROR:${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}WARN:${NC} $*"
}

# Get version from git tag or default
get_version() {
    local version
    version=$(git describe --tags --exact-match 2>/dev/null || echo "")
    if [[ -n "$version" ]]; then
        # Remove leading 'v'
        echo "${version#v}"
    else
        # Default version for development builds
        echo "0.0.0-dev"
    fi
}

# Build binary for a specific platform
build_binary() {
    local goos="$1"
    local goarch="$2"
    local version="$3"
    local output="${DIST_DIR}/tmux-intray_${goos}_${goarch}"

    if [[ "$goos" == "windows" ]]; then
        output="${output}.exe"
    fi

    log_info "Building for ${goos}/${goarch}..."

    cd "$PROJECT_ROOT"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
        -ldflags="-X github.com/cristianoliveira/tmux-intray/internal/version.Version=${version} -X github.com/cristianoliveira/tmux-intray/internal/version.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
        -o "$output" \
        ./cmd/tmux-intray

    if [[ -f "$output" ]]; then
        local size
        size=$(du -h "$output" | cut -f1)
        log_info "  ✓ Built: $(basename "$output") ($size)"
    else
        log_error "  ✗ Failed to build: $(basename "$output")"
        return 1
    fi
}

main() {
    local version
    version=$(get_version)

    log_info "Building tmux-intray binaries for npm package"
    log_info "Version: ${version}"
    log_info "Output directory: ${DIST_DIR}"
    echo ""

    # Create dist directory
    mkdir -p "$DIST_DIR"

    # Supported platforms (matching release.yml)
    local platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )

    local failed=0
    for platform in "${platforms[@]}"; do
        local goos="${platform%/*}"
        local goarch="${platform#*/}"
        if ! build_binary "$goos" "$goarch" "$version"; then
            failed=$((failed + 1))
        fi
    done

    echo ""
    if [[ $failed -eq 0 ]]; then
        log_info "Successfully built all ${#platforms[@]} platform binaries"
        log_info "Ready for npm publish"
    else
        log_error "Failed to build $failed binary(ies)"
        exit 1
    fi
}

main "$@"
