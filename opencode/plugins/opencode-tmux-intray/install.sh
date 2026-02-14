#!/usr/bin/env bash

set -euo pipefail

# OpenCode Tmux Intray Plugin Installer
# Simple installation script

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}INFO:${NC} $*" >&2
}

log_error() {
    echo -e "${RED}ERROR:${NC} $*" >&2
}

# Install plugin to global location
install_plugin() {
    local dest_dir="${HOME}/.config/opencode/plugins"
    local plugin_dir
    plugin_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local src_dir
    src_dir="$(dirname "$plugin_dir")"

    log_info "Installing opencode-tmux-intray plugin..."

    # Create destination directory
    mkdir -p "$dest_dir"

    # Copy main plugin file
    log_info "Copying plugin to ${dest_dir}/"
    cp "${src_dir}/opencode-tmux-intray.js" "${dest_dir}/"

    # Copy supporting directory (tests, config, etc.)
    if [[ -d "${src_dir}/opencode-tmux-intray" ]]; then
        rsync -av --progress \
            --exclude=node_modules \
            --exclude=.tmp \
            "${src_dir}/opencode-tmux-intray/" "${dest_dir}/opencode-tmux-intray/"
    fi

    log_info "Installation complete!"
    log_info "Plugin installed to: ${dest_dir}"
    echo ""
    log_info "Next steps:"
    echo "  1. OpenCode should automatically detect the plugin"
    echo "  2. Restart OpenCode if needed"
}

# Main
install_plugin "$@"
