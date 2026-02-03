#!/usr/bin/env bash
# Hook system for tmux-intray

# Load core utilities
# Determine absolute directory of this script
_TMUX_INTRAY_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./colors.sh disable=SC1091
source "$_TMUX_INTRAY_LIB_DIR/colors.sh"
# shellcheck source=./config.sh disable=SC1091
source "$_TMUX_INTRAY_LIB_DIR/config.sh"

# Default configuration values
TMUX_INTRAY_HOOKS_ENABLED="${TMUX_INTRAY_HOOKS_ENABLED:-1}"
TMUX_INTRAY_HOOKS_FAILURE_MODE="${TMUX_INTRAY_HOOKS_FAILURE_MODE:-warn}"
TMUX_INTRAY_HOOKS_ASYNC="${TMUX_INTRAY_HOOKS_ASYNC:-0}"
TMUX_INTRAY_HOOKS_DIR="${TMUX_INTRAY_HOOKS_DIR:-${TMUX_INTRAY_CONFIG_DIR}/hooks}"
TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT="${TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT:-30}"

# Internal tracking of async hook processes
_TMUX_INTRAY_HOOK_PIDS=()
_TMUX_INTRAY_MAX_HOOKS="${TMUX_INTRAY_MAX_HOOKS:-10}"
_TMUX_INTRAY_HOOKS_TRAPS_SET=0

# Async hook process management
_reap_children() {
    local pid
    local status
    # Filter out PIDs that are no longer running and reap zombies
    local new_pids=()
    for pid in "${_TMUX_INTRAY_HOOK_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            # Process exists, check if it's a zombie
            status=$(ps -o stat= -p "$pid" 2>/dev/null || echo '')
            if [[ "$status" == "Z" ]]; then
                # Zombie, reap it
                wait "$pid" 2>/dev/null && debug "Reaped zombie async hook PID $pid"
                # After reaping, the PID will be removed (don't add to new_pids)
            else
                # Still running (or sleeping, etc.)
                new_pids+=("$pid")
            fi
        else
            # Process no longer exists (already terminated and reaped)
            debug "Async hook PID $pid terminated"
        fi
    done
    _TMUX_INTRAY_HOOK_PIDS=("${new_pids[@]}")
}

_cleanup_async_hooks() {
    # Wait for all remaining async hooks, kill them if they exceed timeout
    if [[ ${#_TMUX_INTRAY_HOOK_PIDS[@]} -gt 0 ]]; then
        info "Waiting for ${#_TMUX_INTRAY_HOOK_PIDS[@]} async hook(s) to complete..."
        local pid
        for pid in "${_TMUX_INTRAY_HOOK_PIDS[@]}"; do
            # Wait with a small timeout to avoid hanging forever
            # Use kill to check if still alive
            if kill -0 "$pid" 2>/dev/null; then
                debug "Waiting for async hook PID $pid"
                if wait "$pid" 2>/dev/null; then
                    debug "Hook process $pid exited"
                else
                    debug "Hook process $pid already exited"
                fi
            fi
        done
        _TMUX_INTRAY_HOOK_PIDS=()
        debug "All async hooks cleaned up"
    fi
}

_setup_hooks_traps() {
    # Set up traps for SIGCHLD, EXIT, INT, TERM only once
    if [[ $_TMUX_INTRAY_HOOKS_TRAPS_SET -eq 0 ]]; then
        trap '_reap_children' CHLD
        trap '_cleanup_async_hooks' EXIT INT TERM
        _TMUX_INTRAY_HOOKS_TRAPS_SET=1
        debug "Async hook traps set"
    fi
}

_execute_async_hook_with_timeout() {
    local script="$1"
    local env_array_name="$2"
    local -n env="$env_array_name"
    local timeout="${TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT:-30}"

    # Check if we have too many pending hooks
    if [[ ${#_TMUX_INTRAY_HOOK_PIDS[@]} -ge $_TMUX_INTRAY_MAX_HOOKS ]]; then
        warning "Too many async hooks pending (max: $_TMUX_INTRAY_MAX_HOOKS), skipping $script"
        return 1
    fi

    # Ensure traps are set
    _setup_hooks_traps

    # Run script with timeout inside a subshell to isolate environment
    # Note: stdout is redirected to stderr (1>&2) to avoid blocking on pipe while still
    # allowing hook output to appear in logs.
    (
        # Redirect stdout to stderr so output goes to terminal/logs, not pipe
        exec 1>&2
        # Export environment (isolated to this subshell)
        for key in "${!env[@]}"; do
            export "$key"="${env[$key]}"
        done
        # Execute with timeout if available, otherwise execute directly
        if command -v timeout >/dev/null 2>&1; then
            timeout "$timeout" "$script" 2>&1
        else
            if [[ "$timeout" != "30" ]]; then
                warning "timeout command not found, ignoring TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT=$timeout"
            fi
            exec "$script" 2>&1
        fi
    ) &

    local pid=$!

    # Track the PID
    _TMUX_INTRAY_HOOK_PIDS+=("$pid")
    debug "Started async hook $script with PID $pid (timeout: ${timeout}s)"

    return 0
}

# Initialize hooks subsystem
hooks_init() {
    debug "Initializing hooks subsystem"
    # Ensure configuration loaded
    config_load

    # Set up traps for async hook cleanup
    _setup_hooks_traps

    # Create hooks directory if it doesn't exist
    mkdir -p "$TMUX_INTRAY_HOOKS_DIR"
    debug "Hooks directory: $TMUX_INTRAY_HOOKS_DIR"
}

# Run hooks for a specific hook point
# Arguments: hook_point [var1=value1 var2=value2 ...]
hooks_run() {
    local hook_point="$1"
    shift
    local env_vars=("$@")

    # Check if hooks are enabled globally and for this hook point
    local hook_point_var="TMUX_INTRAY_HOOKS_ENABLED_${hook_point//-/_}"
    local hook_point_enabled="${!hook_point_var:-$TMUX_INTRAY_HOOKS_ENABLED}"
    if [[ "$hook_point_enabled" != "1" ]]; then
        debug "Hooks disabled for $hook_point (hook_point_enabled=$hook_point_enabled)"
        return 0
    fi
    debug "Hooks enabled for $hook_point (hook_point_enabled=$hook_point_enabled)"

    # Ensure hooks directory exists
    mkdir -p "$TMUX_INTRAY_HOOKS_DIR"
    debug "Hooks directory: $TMUX_INTRAY_HOOKS_DIR"

    # Get hook scripts for this point
    local hook_dir="$TMUX_INTRAY_HOOKS_DIR/$hook_point"
    if [[ ! -d "$hook_dir" ]]; then
        debug "No hooks directory for $hook_point ($hook_dir)"
        return 0
    fi
    debug "Hook directory exists: $hook_dir"

    local scripts=()
    if [[ -d "$hook_dir" ]]; then
        # Find executable scripts, sorted by name (portable)
        local nullglob_state
        nullglob_state=$(shopt -p nullglob)
        shopt -s nullglob
        local script
        for script in "$hook_dir"/*.sh; do
            if [[ -f "$script" && -x "$script" ]]; then
                scripts+=("$script")
            fi
        done
        eval "$nullglob_state"
    fi
    debug "Found ${#scripts[@]} hook script(s) for $hook_point"

    if [[ ${#scripts[@]} -eq 0 ]]; then
        debug "No hook scripts to execute"
        return 0
    fi

    # Prepare base environment
    # shellcheck disable=SC2034  # hook_env is used via nameref in _hook_execute_script
    local -A hook_env
    # Add default env vars
    hook_env["HOOK_POINT"]="$hook_point"
    hook_env["TMUX_INTRAY_HOOKS_FAILURE_MODE"]="$TMUX_INTRAY_HOOKS_FAILURE_MODE"
    # Add passed environment variables
    for var in "${env_vars[@]}"; do
        if [[ "$var" =~ ^([a-zA-Z_][a-zA-Z0-9_]*)=(.*)$ ]]; then
            # shellcheck disable=SC2034  # hook_env is used via nameref in _hook_execute_script
            hook_env["${BASH_REMATCH[1]}"]="${BASH_REMATCH[2]}"
        fi
    done
    debug "Prepared environment with keys: ${!hook_env[*]}"

    # Log hook execution only if there are scripts to run
    if [[ ${#scripts[@]} -gt 0 ]]; then
        log_info "Running $hook_point hooks (${#scripts[@]} script(s))"
    fi

    # Execute each script
    local script
    local hook_result=0
    for script in "${scripts[@]}"; do
        _hook_execute_script "$script" hook_env
        hook_result=$?
        if [[ $hook_result -ne 0 ]]; then
            # Abort further hooks (based on failure mode)
            if [[ "$TMUX_INTRAY_HOOKS_FAILURE_MODE" == "abort" ]]; then
                return $hook_result
            fi
            # For warn/ignore, continue with other scripts
        fi
    done
}

# Execute a single hook script
# Arguments: script_path env_associative_array_name
_hook_execute_script() {
    local script="$1"
    local env_array_name="$2"
    local -n env="$env_array_name" # nameref to associative array

    # Log script execution
    log_info "  Executing hook: $(basename "$script")"
    debug "  Hook script path: $script"

    # Build environment for script

    # Run script
    debug "  Hook execution mode: async=$TMUX_INTRAY_HOOKS_ASYNC"
    if [[ "$TMUX_INTRAY_HOOKS_ASYNC" == "1" ]]; then
        # Run asynchronously with timeout and cleanup
        info "  Starting hook asynchronously: $(basename "$script")"
        _execute_async_hook_with_timeout "$script" "$env_array_name"
        return $?
    fi

    # Synchronous execution
    local start_time
    start_time=$(date +%s.%N)
    debug "  Running hook script synchronously"

    local output
    local exit_code=0
    output=$(
        for key in "${!env[@]}"; do
            export "$key"="${env[$key]}"
        done
        "$script" 2>&1
    )
    exit_code=$?
    # Print hook output to stderr (so it appears in logs)
    if [[ -n "$output" ]]; then
        echo "$output" >&2
    fi
    local end_time
    end_time=$(date +%s.%N)
    local duration
    duration=$(awk "BEGIN {printf \"%.2f\", $end_time - $start_time}")

    debug "  Environment variables passed to hook script"

    # Handle script result based on failure mode
    case "$TMUX_INTRAY_HOOKS_FAILURE_MODE" in
    ignore)
        # Do nothing, not even logging
        ;;
    warn)
        if [[ $exit_code -ne 0 ]]; then
            warning "Hook script $(basename "$script") failed with exit code $exit_code (ignored)"
            warning "Output: $output"
        else
            log_info "  Hook completed in ${duration}s"
        fi
        ;;
    abort)
        if [[ $exit_code -ne 0 ]]; then
            error "Hook script $(basename "$script") failed with exit code $exit_code (aborting)"
            error "Output: $output"
            return $exit_code
        else
            log_info "  Hook completed in ${duration}s"
        fi
        ;;
    *)
        warning "Unknown failure mode: $TMUX_INTRAY_HOOKS_FAILURE_MODE, defaulting to warn"
        if [[ $exit_code -ne 0 ]]; then
            warning "Hook script $(basename "$script") failed with exit code $exit_code"
            warning "Output: $output"
        fi
        ;;
    esac
    debug "  Hook script execution finished"
}
