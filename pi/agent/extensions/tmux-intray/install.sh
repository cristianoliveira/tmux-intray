#!/usr/bin/env bash

set -euo pipefail

# Tmux Intray Plugin Installer
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
    local dest_dir="${HOME}/.pi/agent/extensions"
    local plugin_dir
    plugin_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local src_dir
    src_dir="$(dirname "$plugin_dir")"

    log_info "Installing tmux-intray extension..."

    # Create destination directory
    mkdir -p "$dest_dir"

    # Copy main plugin file
    log_info "Copying plugin to ${dest_dir}/"
    cp -r "${src_dir}/tmux-intray" "${dest_dir}/"

    # Copy supporting directory (tests, config, etc.)
    if [[ -d "${src_dir}/tmux-intray" ]]; then
        rsync -av --progress \
            --exclude=node_modules \
            --exclude=.tmp \
            "${src_dir}/tmux-intray/" "${dest_dir}/tmux-intray/"
    fi

    log_info "Installation complete!"
    log_info "Plugin installed to: ${dest_dir}"
    echo ""
    log_info "Next steps:"
    echo "  1. pi should automatically detect the plugin"
    echo "  2. /reload pi if needed"
}

# Main
install_plugin "$@"
