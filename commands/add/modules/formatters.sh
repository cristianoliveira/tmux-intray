#!/usr/bin/env bash
# Formatters for the add command

# shellcheck source=./commands/add/modules/validators.sh

format_message() {
    local message="$1"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Validate timestamp
    validate_timestamp "$timestamp"
    
    # Format: [timestamp] message
    echo "[${timestamp}] ${message}"
}

format_with_source() {
    local message="$1"
    local source="${2:-unknown}"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo "[${timestamp}] [${source}] ${message}"
}

format_with_priority() {
    local message="$1"
    local priority="${2:-normal}" # normal, high, low
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    local priority_symbol
    case "$priority" in
        high)   priority_symbol="🔴" ;;
        normal) priority_symbol="⚪" ;;
        low)    priority_symbol="🟢" ;;
    esac
    
    echo "[${timestamp}] ${priority_symbol} ${message}"
}
