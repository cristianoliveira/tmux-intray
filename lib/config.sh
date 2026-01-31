#!/usr/bin/env bash
# Configuration management for tmux-intray

# Load core utilities
# shellcheck source=./colors.sh
source "$(dirname "${BASH_SOURCE[0]}")/colors.sh"

# Default configuration values
TMUX_INTRAY_STATE_DIR="${TMUX_INTRAY_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/tmux-intray}"
TMUX_INTRAY_CONFIG_DIR="${TMUX_INTRAY_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/tmux-intray}"
TMUX_INTRAY_MAX_NOTIFICATIONS="${TMUX_INTRAY_MAX_NOTIFICATIONS:-1000}"
TMUX_INTRAY_AUTO_CLEANUP_DAYS="${TMUX_INTRAY_AUTO_CLEANUP_DAYS:-30}"
TMUX_INTRAY_DATE_FORMAT="${TMUX_INTRAY_DATE_FORMAT:-%Y-%m-%d %H:%M:%S}"
TMUX_INTRAY_TABLE_FORMAT="${TMUX_INTRAY_TABLE_FORMAT:-default}"
# Status bar integration
TMUX_INTRAY_STATUS_ENABLED="${TMUX_INTRAY_STATUS_ENABLED:-1}"
TMUX_INTRAY_STATUS_FORMAT="${TMUX_INTRAY_STATUS_FORMAT:-compact}"
TMUX_INTRAY_SHOW_LEVELS="${TMUX_INTRAY_SHOW_LEVELS:-0}"
TMUX_INTRAY_LEVEL_COLORS="${TMUX_INTRAY_LEVEL_COLORS:-info:green,warning:yellow,error:red,critical:magenta}"

# Load user configuration if exists
config_load() {
    local config_file="$TMUX_INTRAY_CONFIG_DIR/config.sh"
    
    if [[ -f "$config_file" ]]; then
        # shellcheck source=/dev/null
        source "$config_file"
        info "Loaded configuration from $config_file"
    else
        # Create directory and sample config file
        mkdir -p "$TMUX_INTRAY_CONFIG_DIR"
        _create_sample_config "$config_file"
    fi
}

# Create sample configuration file
_create_sample_config() {
    local config_file="$1"
    
    cat > "$config_file" << 'EOF'
# tmux-intray configuration
# This file is sourced by tmux-intray on startup.

# Storage directories (follow XDG Base Directory Specification)
# TMUX_INTRAY_STATE_DIR="$HOME/.local/state/tmux-intray"
# TMUX_INTRAY_CONFIG_DIR="$HOME/.config/tmux-intray"

# Storage limits
# Maximum number of notifications to keep (oldest are automatically cleaned up)
# TMUX_INTRAY_MAX_NOTIFICATIONS=1000

# Auto-cleanup: dismiss notifications older than N days
# TMUX_INTRAY_AUTO_CLEANUP_DAYS=30

# Display settings
# Date format for display (see 'man date' for format codes)
# TMUX_INTRAY_DATE_FORMAT="%Y-%m-%d %H:%M:%S"

# Table format style: default, minimal, fancy
# TMUX_INTRAY_TABLE_FORMAT="default"

# Status bar integration
# Enable/disable status indicator (0=disabled, 1=enabled)
# TMUX_INTRAY_STATUS_ENABLED=1
# Status format: compact, detailed, count-only
# TMUX_INTRAY_STATUS_FORMAT="compact"
# Show level counts in status (0=only total, 1=show levels)
# TMUX_INTRAY_SHOW_LEVELS=0
# Level colors for status bar (format: level:color,level:color)
# Available colors: black, red, green, yellow, blue, magenta, cyan, white
# TMUX_INTRAY_LEVEL_COLORS="info:green,warning:yellow,error:red,critical:magenta"
EOF
    
    info "Created sample configuration at $config_file"
    info "Edit this file to customize tmux-intray behavior."
}

# Get configuration value with default
# Usage: config_get <key> <default>
config_get() {
    local key="$1"
    local default="$2"
    
    # Use indirect variable reference
    local value="${!key:-}"
    
    if [[ -z "$value" ]]; then
        echo "$default"
    else
        echo "$value"
    fi
}