#!/usr/bin/env bash

set -euo pipefail

# OpenCode Tmux Intray Plugin Uninstaller
# Removes the plugin from global or local installation

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
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

# Print usage
usage() {
    cat <<EOF
OpenCode Tmux Intray Plugin Uninstaller
Usage: $0 [OPTIONS]

Options:
  -g, --global        Uninstall from global location ~/.config/opencode/plugins/
  -l, --local         Uninstall from local location \$PWD/.opencode/plugins/
  --force             Skip confirmation prompt
  --dry-run           Perform a dry run without making changes
  --help              Show this help message

Examples:
  $0 --global          # Uninstall globally
  $0 --local           # Uninstall locally
  $0 --global --force  # Force uninstall without confirmation
  $0 --global --dry-run # Dry run uninstall
EOF
}

# Remove plugin files
remove_plugin() {
    local dest_dir="$1"
    local force="$2"
    local dry_run="${3:-false}"

    local dest_js="${dest_dir}/opencode-tmux-intray.js"
    local dest_plugin_dir="${dest_dir}/opencode-tmux-intray"

    # Check if plugin exists
    if [[ ! -e "$dest_js" ]] && [[ ! -e "$dest_plugin_dir" ]]; then
        log_warn "Plugin not found in ${dest_dir}"
        return 0
    fi

    log_info "Found plugin in ${dest_dir}"

    if [[ "$force" != true ]] && [[ "$dry_run" != true ]]; then
        read -r -p "Remove plugin from ${dest_dir}? [y/N] " response
        if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            log_info "Aborted."
            return 0
        fi
    elif [[ "$dry_run" == true ]]; then
        log_info "[dry-run] Would prompt for removal (assuming yes)"
    fi

    # Remove files
    if [[ "$dry_run" != true ]]; then
        if [[ -e "$dest_js" ]]; then
            log_info "Removing ${dest_js}"
            rm -f "$dest_js"
        fi

        if [[ -e "$dest_plugin_dir" ]]; then
            log_info "Removing ${dest_plugin_dir}"
            rm -rf "$dest_plugin_dir"
        fi

        # Try to remove parent directories if empty (optional)
        rmdir "$dest_dir" 2>/dev/null || true
        rmdir "$(dirname "$dest_dir")" 2>/dev/null || true
    else
        log_info "[dry-run] Would remove ${dest_js} (if exists)"
        log_info "[dry-run] Would remove ${dest_plugin_dir} (if exists)"
    fi

    log_info "Plugin removed successfully."
    return 0
}

main() {
    local uninstall_global=false
    local uninstall_local=false
    local force=false
    local dry_run=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
        -g | --global)
            uninstall_global=true
            shift
            ;;
        -l | --local)
            uninstall_local=true
            shift
            ;;
        --force)
            force=true
            shift
            ;;
        --dry-run)
            dry_run=true
            shift
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

    # Validate options
    if [[ "$uninstall_global" == false ]] && [[ "$uninstall_local" == false ]]; then
        log_error "You must specify either --global or --local."
        usage
        exit 1
    fi

    if [[ "$uninstall_global" == true ]] && [[ "$uninstall_local" == true ]]; then
        log_error "Cannot specify both --global and --local. Choose one."
        usage
        exit 1
    fi

    # Determine destination directory
    local dest_dir
    if [[ "$uninstall_global" == true ]]; then
        dest_dir="${HOME}/.config/opencode/plugins"
        log_info "Uninstalling from global location: ${dest_dir}"
    else
        dest_dir="$(pwd)/.opencode/plugins"
        log_info "Uninstalling from local location: ${dest_dir}"
    fi

    # Remove plugin files
    remove_plugin "$dest_dir" "$force" "$dry_run"

    log_info "Uninstallation complete!"
}

main "$@"
