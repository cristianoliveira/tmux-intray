#!/usr/bin/env bats
# Test migration from environment variables to file storage

setup() {
    export TMUX_SOCKET_NAME="tmux-intray-test-migration"
    export XDG_STATE_HOME="$(mktemp -d)"
    export XDG_CONFIG_HOME="$(mktemp -d)"
    
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    tmux -L "$TMUX_SOCKET_NAME" new-session -d -s test
    sleep 0.1
}

teardown() {
    tmux -L "$TMUX_SOCKET_NAME" kill-server 2>/dev/null || true
    sleep 0.1
    rm -rf "$XDG_STATE_HOME" "$XDG_CONFIG_HOME"
}

@test "migration from empty environment" {
    source ./lib/storage.sh
    
    # Ensure no env variable
    tmux set-environment -g TMUX_INTRAY_ITEMS ""
    
    run storage_migrate_from_env
    [ "$status" -eq 0 ]
    
    # No notifications should be created
    [ "$(storage_get_active_count)" -eq 0 ]
}

@test "migration with single item" {
    source ./lib/storage.sh
    
    # Set up environment variable with old format
    tmux set-environment -g TMUX_INTRAY_ITEMS "[2025-01-31 12:34:56] Test message"
    
    run storage_migrate_from_env
    [ "$status" -eq 0 ]
    
    # Should have one notification
    [ "$(storage_get_active_count)" -eq 1 ]
    
    # Environment variable should be cleared
    local env_value
    env_value=$(tmux show-environment -g TMUX_INTRAY_ITEMS 2>/dev/null)
    [[ "$env_value" == *"=\"\"" || "$env_value" == "" ]]
    
    # Verify notification content
    local line
    line=$(storage_list_notifications "active")
    IFS=$'\t' read -r _ _ _ _ _ _ message _ _ <<< "$line"
    
    # Message should be unescaped
    [[ "$message" == *"Test message"* ]]
}

@test "migration with multiple colon-separated items" {
    source ./lib/storage.sh
    
    tmux set-environment -g TMUX_INTRAY_ITEMS "[2025-01-31 12:34:56] First:[2025-01-31 12:35:00] Second"
    
    storage_migrate_from_env
    
    [ "$(storage_get_active_count)" -eq 2 ]
}

@test "migration preserves timestamps" {
    source ./lib/storage.sh
    
    local original_timestamp="2025-01-31 12:34:56"
    tmux set-environment -g TMUX_INTRAY_ITEMS "[${original_timestamp}] Test"
    
    storage_migrate_from_env
    
    local line
    line=$(storage_list_notifications "active")
    IFS=$'\t' read -r _ timestamp _ _ _ _ _ _ _ <<< "$line"
    
    # Timestamp should be converted to ISO format
    [[ "$timestamp" == *"2025-01-31T12:34:56Z"* ]]
}

@test "migration of items without timestamp" {
    source ./lib/storage.sh
    
    # Items might be malformed (no timestamp)
    tmux set-environment -g TMUX_INTRAY_ITEMS "Plain message"
    
    storage_migrate_from_env
    
    [ "$(storage_get_active_count)" -eq 1 ]
}

@test "migration only happens once" {
    source ./lib/storage.sh
    
    tmux set-environment -g TMUX_INTRAY_ITEMS "[2025-01-31 12:34:56] Test"
    
    # First migration
    storage_migrate_from_env
    local first_count=$(storage_get_active_count)
    
    # Second migration should not add duplicates
    storage_migrate_from_env
    local second_count=$(storage_get_active_count)
    
    [ "$first_count" -eq "$second_count" ]
}