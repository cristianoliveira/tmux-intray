#!/usr/bin/env bash

set -euo pipefail

# OpenCode Tmux Intray Plugin Installer
# Installs the plugin globally or locally for OpenCode

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_debug() {
    echo -e "${BLUE}DEBUG:${NC} $*" >&2
}

# Print usage
usage() {
    cat <<EOF
OpenCode Tmux Intray Plugin Installer
Usage: $0 [OPTIONS]

Options:
  -g, --global        Install globally to ~/.config/opencode/plugins/
  -l, --local         Install locally to \$PWD/.opencode/plugins/
  --force             Overwrite existing installation without asking
  --no-deps           Skip npm dependency installation
  --dry-run           Perform a dry run without making changes
  --help              Show this help message

Examples:
  $0 --global          # Install globally
  $0 --local           # Install in current directory
  $0 --global --force  # Force overwrite global installation
  $0 --global --dry-run # Dry run global installation
EOF
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check Node.js and npm availability
check_deps() {
    if ! command_exists node; then
        log_error "Node.js is not installed. Please install Node.js (>=18)."
        return 1
    fi
    if ! command_exists npm; then
        log_error "npm is not installed. Please install npm."
        return 1
    fi
    log_info "Node.js and npm are available."
    return 0
}

# Install npm dependencies in plugin directory
install_deps() {
    local plugin_dir="$1"
    local dry_run="${2:-false}"
    log_info "Installing npm dependencies in ${plugin_dir}..."
    if [[ "$dry_run" != true ]]; then
        if ! npm ci --only=production; then
            log_warn "npm ci failed, trying npm install..."
            if ! npm install --production; then
                log_error "Failed to install npm dependencies."
                return 1
            fi
        fi
        log_info "Dependencies installed successfully."
    else
        log_info "[dry-run] Would run npm ci --only=production (or npm install --production)"
    fi
    return 0
}

# Copy plugin files to destination, excluding node_modules and .tmp directories
copy_plugin() {
    local src_dir="$1"
    local dest_dir="$2"
    local force="$3"
    local dry_run="${4:-false}"

    # Ensure destination directory exists
    if [[ "$dry_run" == true ]]; then
        log_info "[dry-run] Would create directory: ${dest_dir}"
    else
        mkdir -p "$dest_dir"
    fi

    # Check if plugin already exists
    local dest_js="${dest_dir}/opencode-tmux-intray.js"
    local dest_plugin_dir="${dest_dir}/opencode-tmux-intray"

    if [[ -e "$dest_js" ]] || [[ -e "$dest_plugin_dir" ]]; then
        if [[ "$force" == true ]]; then
            log_warn "Overwriting existing plugin files in ${dest_dir}"
        else
            log_warn "Plugin already exists in ${dest_dir}"
            if [[ "$dry_run" == true ]]; then
                log_info "[dry-run] Would prompt for overwrite (assuming yes)"
            else
                read -r -p "Overwrite? [y/N] " response
                if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
                    log_info "Skipping installation."
                    return 1
                fi
            fi
        fi
    fi

    # Copy single JS file
    log_info "Copying ${src_dir}/opencode-tmux-intray.js to ${dest_dir}/"
    if [[ "$dry_run" != true ]]; then
        cp "${src_dir}/opencode-tmux-intray.js" "${dest_dir}/"
    else
        log_info "[dry-run] Would copy ${src_dir}/opencode-tmux-intray.js to ${dest_dir}/"
    fi

    # Copy plugin directory, excluding node_modules and .tmp
    log_info "Copying ${src_dir}/opencode-tmux-intray/ to ${dest_dir}/ (excluding node_modules, .tmp)"
    if [[ "$dry_run" != true ]]; then
        if command_exists rsync; then
            rsync -av --progress \
                --exclude=node_modules \
                --exclude=.tmp \
                --exclude=.git \
                "${src_dir}/opencode-tmux-intray/" "${dest_dir}/opencode-tmux-intray/"
        else
            # Using find and cpio to preserve directory structure and exclude patterns
            (
                cd "${src_dir}/opencode-tmux-intray" || exit 1
                find . -type f \( -name ".*" -prune -o -print \) |
                    grep -E -v '(^\./node_modules|^\./\.tmp|^\./\.git)' |
                    cpio -pdum "${dest_dir}/opencode-tmux-intray/" 2>/dev/null || true
            )
            # Fallback to cp -R if cpio fails
            if [[ ! -d "${dest_dir}/opencode-tmux-intray" ]]; then
                log_warn "Using simple cp -R (exclusions may not work)"
                cp -R "${src_dir}/opencode-tmux-intray/." "${dest_dir}/opencode-tmux-intray/"
            fi
        fi
    else
        log_info "[dry-run] Would copy plugin directory from ${src_dir}/opencode-tmux-intray/ to ${dest_dir}/opencode-tmux-intray/"
    fi

    # Make install script executable
    if [[ "$dry_run" != true ]]; then
        chmod +x "${dest_dir}/opencode-tmux-intray/install.sh" 2>/dev/null || true
    else
        log_info "[dry-run] Would make install script executable"
    fi

    log_info "Plugin files copied successfully."
    return 0
}

# Add npm scripts to package.json in plugin directory
add_npm_scripts() {
    local plugin_dir="$1"
    local dry_run="${2:-false}"
    local package_json="${plugin_dir}/package.json"

    if [[ ! -f "$package_json" ]]; then
        log_warn "package.json not found in plugin directory, skipping npm scripts addition."
        return 0
    fi

    log_info "Adding npm scripts to package.json..."

    if [[ "$dry_run" != true ]]; then
        # Use jq if available, else sed
        if command_exists jq; then
            if ! jq '.scripts += {
                "install-plugin": "./install.sh",
                "uninstall-plugin": "./uninstall.sh"
            }' "$package_json" >"${package_json}.tmp"; then
                log_warn "Failed to update package.json with jq."
                return 0
            fi
            mv "${package_json}.tmp" "$package_json"
            log_info "Added npm scripts: install-plugin, uninstall-plugin"
        else
            log_warn "jq not found, cannot update package.json automatically."
            log_info "Please add the following scripts to ${package_json} manually:"
            cat <<'EOF'
  "scripts": {
    "install-plugin": "./install.sh",
    "uninstall-plugin": "./uninstall.sh"
  }
EOF
        fi
    else
        log_info "[dry-run] Would add npm scripts to package.json"
    fi
}

main() {
    local install_global=false
    local install_local=false
    local force=false
    local install_deps=true
    local dry_run=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
        -g | --global)
            install_global=true
            shift
            ;;
        -l | --local)
            install_local=true
            shift
            ;;
        --force)
            force=true
            shift
            ;;
        --no-deps)
            install_deps=false
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
    if [[ "$install_global" == false ]] && [[ "$install_local" == false ]]; then
        log_error "You must specify either --global or --local."
        usage
        exit 1
    fi

    if [[ "$install_global" == true ]] && [[ "$install_local" == true ]]; then
        log_error "Cannot specify both --global and --local. Choose one."
        usage
        exit 1
    fi

    # Determine source directory (where this script is located)
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SRC_DIR="$(dirname "$SCRIPT_DIR")"

    if [[ ! -f "${SRC_DIR}/opencode-tmux-intray.js" ]] || [[ ! -d "${SRC_DIR}/opencode-tmux-intray" ]]; then
        log_error "Source plugin files not found. Make sure you're running from the correct location."
        exit 1
    fi

    # Determine destination directory
    local dest_dir
    if [[ "$install_global" == true ]]; then
        dest_dir="${HOME}/.config/opencode/plugins"
        log_info "Installing globally to ${dest_dir}"
    else
        dest_dir="$(pwd)/.opencode/plugins"
        log_info "Installing locally to ${dest_dir}"
    fi

    # Check dependencies (node, npm)
    if [[ "$install_deps" == true ]]; then
        check_deps || exit 1
    fi

    # Copy plugin files
    copy_plugin "$SRC_DIR" "$dest_dir" "$force" "$dry_run" || exit 1

    # Install npm dependencies in plugin directory
    if [[ "$install_deps" == true ]]; then
        local plugin_path="${dest_dir}/opencode-tmux-intray"
        if [[ "$dry_run" != true ]]; then
            (cd "$plugin_path" && install_deps "$plugin_path" "$dry_run") || exit 1
        else
            install_deps "$plugin_path" "$dry_run" || exit 1
        fi
    fi

    # Add npm scripts to package.json
    add_npm_scripts "${dest_dir}/opencode-tmux-intray" "$dry_run"

    log_info "Installation complete!"
    log_info "Plugin installed to: ${dest_dir}"

    if [[ "$install_global" == true ]]; then
        cat <<'EOF'

Next steps:
1. OpenCode should automatically detect the plugin in ~/.config/opencode/plugins/
2. Configure the plugin by creating ~/.config/opencode-tmux-intray/opencode-config.json
3. Restart OpenCode if needed

For more information, see the plugin README.
EOF
    else
        cat <<'EOF'

Next steps:
1. OpenCode should detect the plugin in ./.opencode/plugins/
2. Configure the plugin by creating ~/.config/opencode-tmux-intray/opencode-config.json
3. Restart OpenCode if needed

Note: The plugin is installed locally to this directory.
EOF
    fi
}

main "$@"
