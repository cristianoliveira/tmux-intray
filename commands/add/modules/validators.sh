#!/usr/bin/env bash
# Validators for the add command

validate_message() {
    local message="$1"
    
    # Check if message is too long
    if [[ ${#message} -gt 1000 ]]; then
        error "Message too long (max 1000 characters)"
        exit 1
    fi
    
    # Check if message is empty after stripping
    local stripped
    stripped=$(echo "$message" | xargs)
    if [[ -z "$stripped" ]]; then
        error "Message cannot be empty"
        exit 1
    fi
}

validate_timestamp() {
    local timestamp="$1"
    
    # Basic timestamp validation (YYYY-MM-DD HH:MM:SS)
    if [[ ! "$timestamp" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}\ [0-9]{2}:[0-9]{2}:[0-9]{2}$ ]]; then
        error "Invalid timestamp format"
        exit 1
    fi
}
