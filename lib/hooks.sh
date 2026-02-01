#!/usr/bin/env bash
# Hook system for tmux-intray

# Load core utilities
# shellcheck source=./colors.sh disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/colors.sh"
# shellcheck source=./config.sh disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/config.sh"

# Default configuration values
TMUX_INTRAY_HOOKS_ENABLED="${TMUX_INTRAY_HOOKS_ENABLED:-1}"
TMUX_INTRAY_HOOKS_FAILURE_MODE="${TMUX_INTRAY_HOOKS_FAILURE_MODE:-warn}"
TMUX_INTRAY_HOOKS_ASYNC="${TMUX_INTRAY_HOOKS_ASYNC:-0}"
TMUX_INTRAY_HOOKS_DIR="${TMUX_INTRAY_HOOKS_DIR:-${TMUX_INTRAY_CONFIG_DIR}/hooks}"

# Initialize hooks subsystem
hooks_init() {
    # Ensure configuration loaded
    config_load

    # Create hooks directory if it doesn't exist
    mkdir -p "$TMUX_INTRAY_HOOKS_DIR"
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
        return 0
    fi

    # Ensure hooks directory exists
    mkdir -p "$TMUX_INTRAY_HOOKS_DIR"

    # Get hook scripts for this point
    local hook_dir="$TMUX_INTRAY_HOOKS_DIR/$hook_point"
    if [[ ! -d "$hook_dir" ]]; then
        # No hooks for this point
        return 0
    fi

    local scripts=()
    if [[ -d "$hook_dir" ]]; then
        # Find executable scripts, sorted by name
        while IFS= read -r -d '' script; do
            scripts+=("$script")
        done < <(find "$hook_dir" -type f -name "*.sh" -executable -print0 | sort -z)
    fi

    if [[ ${#scripts[@]} -eq 0 ]]; then
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

    # Log hook execution
    info "Running $hook_point hooks (${#scripts[@]} script(s))"

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
    info "  Executing hook: $(basename "$script")"

    # Build environment for script
    local key
    for key in "${!env[@]}"; do
        export "$key"="${env[$key]}"
    done

    # Run script
    if [[ "$TMUX_INTRAY_HOOKS_ASYNC" == "1" ]]; then
        # Run asynchronously (fire and forget)
        info "  Starting hook asynchronously: $(basename "$script")"
        (
            # Export environment for the child process
            for key in "${!env[@]}"; do
                export "$key"="${env[$key]}"
            done
            # Execute script
            exec "$script"
        ) &
        # Clean up exported variables in parent (they were exported in subshell only)
        return 0
    fi

    # Synchronous execution
    local start_time
    start_time=$(date +%s.%N)

    local output
    local exit_code=0
    output=$("$script" 2>&1)
    exit_code=$?
    # Print hook output to stderr (so it appears in logs)
    if [[ -n "$output" ]]; then
        echo "$output" >&2
    fi
    local end_time
    end_time=$(date +%s.%N)
    local duration
    duration=$(awk "BEGIN {printf \"%.2f\", $end_time - $start_time}")

    # Clean up exported variables (except TMUX_INTRAY_HOOKS_FAILURE_MODE which is global)
    for key in "${!env[@]}"; do
        if [[ "$key" != "TMUX_INTRAY_HOOKS_FAILURE_MODE" ]]; then
            unset "$key"
        fi
    done

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
            info "  Hook completed in ${duration}s"
        fi
        ;;
    abort)
        if [[ $exit_code -ne 0 ]]; then
            error "Hook script $(basename "$script") failed with exit code $exit_code (aborting)"
            error "Output: $output"
            return $exit_code
        else
            info "  Hook completed in ${duration}s"
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
}
