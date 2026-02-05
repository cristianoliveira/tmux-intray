#!/usr/bin/env bash

set -euo pipefail

# tmux-intray one-click installer
# Downloads the latest release from GitHub and installs it locally

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
# shellcheck disable=SC2034
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_error() {
    echo -e "${RED}ERROR:${NC} $*" >&2
}

log_info() {
    echo -e "${GREEN}INFO:${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}WARN:${NC} $*" >&2
}

usage() {
    cat <<EOF
tmux-intray one-click installer
Usage: $0 [OPTIONS]

Options:
  --dry-run        Perform a dry run without making changes
  --prefix DIR     Set installation prefix (default: ~/.local)
  --help           Show this help message

Example:
  $0 --dry-run
  $0 --prefix /usr/local

The installer downloads the latest release from GitHub and installs tmux-intray
to PREFIX/bin/tmux-intray.

The installer downloads the pre-built Go binary for optimal performance.
EOF
}

# Default installation directory
INSTALL_PREFIX="${HOME}/.local"
INSTALL_BIN_DIR="${INSTALL_PREFIX}/bin"
# shellcheck disable=SC2034
INSTALL_SHARE_DIR="${INSTALL_PREFIX}/share/tmux-intray"

# Detect platform
detect_platform() {
    case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *) echo "unknown" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
    x86_64 | amd64) echo "amd64" ;;
    arm64 | aarch64) echo "arm64" ;;
    *) echo "unknown" ;;
    esac
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Add directory to PATH in shell config if not already present
add_to_path() {
    local dir="$1"
    if [[ ":$PATH:" != *":${dir}:"* ]]; then
        log_warn "${dir} is not in your PATH."
        local shell_rc
        shell_rc="${HOME}/.$(basename "${SHELL:-bash}")rc"
        if [[ -f "${shell_rc}" ]]; then
            log_info "You can add ${dir} to your PATH by adding the following line to ${shell_rc}:"
            echo "  export PATH=\"${dir}:\$PATH\"" >&2
        fi
    fi
}

# Build from source if pre-built binary is not available
build_from_source() {
    local temp_dir="$1"

    log_info "Building tmux-intray from source..."

    # Check if Go is installed
    if ! command_exists go; then
        log_error "Go is required for building from source. Please install Go first."
        log_error "Visit https://golang.org/dl/ for installation instructions."
        return 1
    fi

    # Get source code
    local source_url="https://github.com/cristianoliveira/tmux-intray/archive/refs/heads/main.tar.gz"
    local source_path="${temp_dir}/source.tar.gz"
    local extract_dir="${temp_dir}/source"

    log_info "Downloading source code..."
    if ! curl -fsSL -o "$source_path" "$source_url"; then
        log_error "Failed to download source code"
        return 1
    fi

    # Extract source
    mkdir -p "$extract_dir"
    if ! tar -xzf "$source_path" -C "$extract_dir" --strip-components=1; then
        log_error "Failed to extract source code"
        return 1
    fi

    # Build the binary
    log_info "Building binary..."
    cd "$extract_dir"
    if ! go build -o tmux-intray ./cmd/tmux-intray; then
        log_error "Failed to build binary"
        return 1
    fi

    # Return path to built binary
    echo "${extract_dir}/tmux-intray"
}

# Download binary from GitHub release
download_binary() {
    local temp_dir="$1"
    local platform="$2"
    local arch="$3"

    # Get latest release info
    local api_url="https://api.github.com/repos/cristianoliveira/tmux-intray/releases/latest"

    log_info "Fetching latest release information..."
    local release_info
    if ! release_info=$(curl -fsSL "$api_url" 2>/dev/null); then
        log_error "Failed to fetch release information from GitHub"
        return 1
    fi

    # Get tag name
    local tag_name
    if ! tag_name=$(echo "$release_info" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4); then
        log_error "Failed to extract tag name from release"
        return 1
    fi

    log_info "Found latest release: ${tag_name}"

    # Determine binary name based on platform
    local binary_name="tmux-intray_${platform}_${arch}"
    if [[ "$platform" == "windows" ]]; then
        binary_name="${binary_name}.exe"
    fi

    # Find the download URL for our binary
    local download_url
    download_url=$(echo "$release_info" | grep -o "\"browser_download_url\": *\"[^\"]*${binary_name}[^\"]*\"" | cut -d'"' -f4)

    if [[ -z "$download_url" ]]; then
        log_error "Binary ${binary_name} not found in release artifacts"
        log_error "Available artifacts:"
        echo "$release_info" | grep -o '"browser_download_url": *"[^"]*"' | cut -d'"' -f4 | sed 's/^/  - /' >&2
        return 1
    fi

    # Download the binary
    local binary_path="${temp_dir}/${binary_name}"
    log_info "Downloading ${binary_name} from ${download_url}..."
    if ! curl -fsSL -o "$binary_path" "$download_url"; then
        log_error "Failed to download binary"
        return 1
    fi

    # Make binary executable (except on Windows)
    if [[ "$platform" != "windows" ]]; then
        chmod +x "$binary_path"
    fi

    echo "$binary_path"
}

main() {
    local dry_run=false
    local prefix="${HOME}/.local"

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
        --dry-run)
            dry_run=true
            shift
            ;;
        --prefix)
            if [[ -z "$2" ]]; then
                log_error "Missing argument for --prefix"
                usage
                exit 1
            fi
            prefix="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
        esac
    done

    # Update installation directories based on prefix
    INSTALL_PREFIX="$prefix"
    INSTALL_BIN_DIR="${INSTALL_PREFIX}/bin"

    log_info "Starting tmux-intray installation..."
    log_info "Installation prefix: ${INSTALL_PREFIX}"
    if [[ "$dry_run" == true ]]; then
        log_info "Dry run enabled - no changes will be made"
    fi

    # Check for curl
    if ! command_exists curl; then
        log_error "curl is required but not installed. Please install curl first."
        exit 1
    fi

    # Detect platform and architecture
    local platform
    local arch
    platform=$(detect_platform)
    arch=$(detect_arch)

    if [[ "$platform" == "unknown" ]]; then
        log_error "Unsupported platform: $(uname -s)"
        exit 1
    fi

    if [[ "$arch" == "unknown" ]]; then
        log_error "Unsupported architecture: $(uname -m)"
        exit 1
    fi

    log_info "Detected platform: ${platform}/${arch}"

    # Create temporary directory
    local temp_dir
    temp_dir=$(mktemp -d)
    trap '[[ -n "${temp_dir:-}" ]] && rm -rf "$temp_dir"' EXIT

    # Download binary
    local binary_path
    if [[ "$dry_run" != true ]]; then
        if ! binary_path=$(download_binary "$temp_dir" "$platform" "$arch"); then
            log_info "Pre-built binary not found. Attempting to build from source..."
            binary_path=$(build_from_source "$temp_dir") || exit 1
        fi
    else
        log_info "[dry-run] Would download binary for ${platform}/${arch}"
        binary_path="${temp_dir}/tmux-intray_${platform}_${arch}"
    fi

    # Create installation directory
    if [[ "$dry_run" != true ]]; then
        mkdir -p "$INSTALL_BIN_DIR"
    else
        log_info "[dry-run] Would create directory: ${INSTALL_BIN_DIR}"
    fi

    # Install binary
    local install_path="${INSTALL_BIN_DIR}/tmux-intray"
    if [[ "$platform" == "windows" ]]; then
        install_path="${install_path}.exe"
    fi

    log_info "Installing tmux-intray to ${install_path}..."
    if [[ "$dry_run" != true ]]; then
        if [[ -f "$install_path" ]]; then
            log_warn "Existing tmux-intray found at ${install_path}"
            read -r -p "Overwrite? [y/N] " response
            if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
                log_info "Skipping installation"
                exit 0
            fi
        fi
        cp "$binary_path" "$install_path"
    else
        log_info "[dry-run] Would copy binary to ${install_path}"
    fi

    # Check PATH
    add_to_path "$INSTALL_BIN_DIR"

    # Verify installation
    log_info "Verifying installation..."
    if [[ "$dry_run" != true ]]; then
        if [[ -x "$install_path" ]]; then
            # Test the installed binary
            if "$install_path" version >/dev/null 2>&1; then
                log_info "tmux-intray installed successfully!"
                log_info "Run 'tmux-intray --help' to get started."
            else
                log_error "Binary verification failed"
                exit 1
            fi
        else
            log_error "Binary not found at ${install_path}"
            exit 1
        fi
    else
        log_info "[dry-run] Would verify installation"
    fi
}

main "$@"
