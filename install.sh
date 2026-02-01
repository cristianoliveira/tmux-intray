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
to PREFIX/share/tmux-intray, with a symlink in PREFIX/bin.
EOF
}

# Default installation directory
INSTALL_PREFIX="${HOME}/.local"
INSTALL_BIN_DIR="${INSTALL_PREFIX}/bin"
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

# Download latest release tarball, fall back to main branch if no releases
# If TMUX_INTRAY_LOCAL_TARBALL is set, use that local file instead.
download_latest_release() {
    local temp_dir="$1"

    # If local tarball is provided, use it (for testing)
    if [[ -n "${TMUX_INTRAY_LOCAL_TARBALL:-}" ]]; then
        log_info "Using local tarball: ${TMUX_INTRAY_LOCAL_TARBALL}"
        local tarball_path="${temp_dir}/tmux-intray-local.tar.gz"
        cp "$TMUX_INTRAY_LOCAL_TARBALL" "$tarball_path"
        echo "$tarball_path"
        return 0
    fi

    local api_url="https://api.github.com/repos/cristianoliveira/tmux-intray/releases/latest"

    log_info "Fetching latest release information..."
    # Get latest release tag
    local tag_name
    if ! tag_name=$(curl -fsSL "$api_url" 2>/dev/null | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4); then
        log_warn "No releases found, falling back to main branch"
        tag_name="main"
    fi
    if [[ -z "$tag_name" ]]; then
        log_error "Failed to fetch latest release tag"
        return 1
    fi

    log_info "Using version: ${tag_name}"
    local tarball_url
    local tarball_url2
    if [[ "$tag_name" == "main" ]]; then
        tarball_url="https://github.com/cristianoliveira/tmux-intray/archive/refs/heads/main.tar.gz"
        tarball_url2="https://github.com/cristianoliveira/tmux-intray/archive/main.tar.gz"
    else
        tarball_url="https://github.com/cristianoliveira/tmux-intray/archive/refs/tags/${tag_name}.tar.gz"
        tarball_url2="https://github.com/cristianoliveira/tmux-intray/archive/${tag_name}.tar.gz"
    fi
    local tarball_path="${temp_dir}/tmux-intray-${tag_name}.tar.gz"

    log_info "Downloading ${tarball_url}..."
    if ! curl -fsSL -o "$tarball_path" "$tarball_url"; then
        log_warn "First URL failed, trying alternative..."
        if ! curl -fsSL -o "$tarball_path" "$tarball_url2"; then
            log_error "Failed to download tarball from both URLs"
            return 1
        fi
    fi
    echo "$tarball_path"
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
    INSTALL_SHARE_DIR="${INSTALL_PREFIX}/share/tmux-intray"

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

    # Create temporary directory
    local temp_dir
    temp_dir=$(mktemp -d)
    trap '[[ -n "${temp_dir:-}" ]] && rm -rf "$temp_dir"' EXIT

    # Download latest release
    local tarball
    tarball=$(download_latest_release "$temp_dir") || exit 1

    # Extract tarball
    log_info "Extracting tarball..."
    if [[ "$dry_run" != true ]]; then
        tar -xzf "$tarball" -C "$temp_dir"
    else
        log_info "[dry-run] Would extract tarball"
    fi
    local extracted_dir
    extracted_dir=$(find "$temp_dir" -type d -name "tmux-intray-*" | head -1)
    if [[ -z "$extracted_dir" ]]; then
        # If no top-level directory, assume files were extracted directly
        extracted_dir="$temp_dir"
        log_info "No top-level directory found, using extraction root"
    fi

    # Create installation directories
    if [[ "$dry_run" != true ]]; then
        mkdir -p "$INSTALL_BIN_DIR" "$INSTALL_SHARE_DIR"
    else
        log_info "[dry-run] Would create directories: ${INSTALL_BIN_DIR}, ${INSTALL_SHARE_DIR}"
    fi

    # Copy files to share directory
    log_info "Installing tmux-intray to ${INSTALL_SHARE_DIR}..."
    if [[ "$dry_run" != true ]]; then
        cp -R "$extracted_dir/." "$INSTALL_SHARE_DIR/"
    else
        log_info "[dry-run] Would copy files from ${extracted_dir} to ${INSTALL_SHARE_DIR}"
    fi

    # Make scripts executable
    if [[ "$dry_run" != true ]]; then
        chmod +x "$INSTALL_SHARE_DIR/bin/tmux-intray"
        chmod +x "$INSTALL_SHARE_DIR/scripts/lint.sh" 2>/dev/null || true
        chmod +x "$INSTALL_SHARE_DIR/tmux-intray.tmux" 2>/dev/null || true
    else
        log_info "[dry-run] Would make scripts executable"
    fi

    # Create symlink in bin directory
    local symlink_path="${INSTALL_BIN_DIR}/tmux-intray"
    if [[ -L "$symlink_path" ]] || [[ -f "$symlink_path" ]]; then
        log_warn "Existing tmux-intray found at ${symlink_path}"
        if [[ "$dry_run" != true ]]; then
            read -r -p "Overwrite? [y/N] " response
            if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
                log_info "Skipping symlink creation"
            else
                ln -sf "$INSTALL_SHARE_DIR/bin/tmux-intray" "$symlink_path"
                log_info "Symlink created at ${symlink_path}"
            fi
        else
            log_info "[dry-run] Would prompt to overwrite existing symlink"
        fi
    else
        if [[ "$dry_run" != true ]]; then
            ln -s "$INSTALL_SHARE_DIR/bin/tmux-intray" "$symlink_path"
            log_info "Symlink created at ${symlink_path}"
        else
            log_info "[dry-run] Would create symlink at ${symlink_path}"
        fi
    fi

    # Check PATH
    add_to_path "$INSTALL_BIN_DIR"

    # Verify installation
    log_info "Verifying installation..."
    if [[ "$dry_run" != true ]]; then
        if "$symlink_path" version >/dev/null 2>&1; then
            log_info "tmux-intray installed successfully!"
            log_info "Run 'tmux-intray --help' to get started."
        else
            log_error "Installation verification failed"
            exit 1
        fi
    else
        log_info "[dry-run] Would verify installation"
    fi
}

main "$@"
