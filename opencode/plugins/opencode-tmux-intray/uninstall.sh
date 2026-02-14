#!/usr/bin/env bash

set -euo pipefail

# OpenCode Tmux Intray Plugin Uninstaller

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

# Uninstall plugin from global location
uninstall_plugin() {
    local dest_dir="${HOME}/.config/opencode/plugins"
    local dest_js="${dest_dir}/opencode-tmux-intray.js"
    local dest_plugin_dir="${dest_dir}/opencode-tmux-intray"

    log_info "Uninstalling opencode-tmux-intray plugin..."

    # Check if plugin exists
    if [[ ! -e "$dest_js" ]] && [[ ! -e "$dest_plugin_dir" ]]; then
        log_info "Plugin not found in ${dest_dir}"
        return 0
    fi

    # Confirm removal
    read -r -p "Remove plugin from ${dest_dir}? [y/N] " response
    if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        log_info "Aborted."
        return 0
    fi

    # Remove files
    if [[ -e "$dest_js" ]]; then
        rm -f "$dest_js"
        log_info "Removed ${dest_js}"
    fi

    if [[ -e "$dest_plugin_dir" ]]; then
        rm -rf "$dest_plugin_dir"
        log_info "Removed ${dest_plugin_dir}"
    fi

    log_info "Uninstallation complete!"
}

# Main
uninstall_plugin "$@"
