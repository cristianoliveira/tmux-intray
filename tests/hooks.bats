#!/usr/bin/env bats
# Hook system tests
# shellcheck disable=SC1091,SC2030,SC2031

setup() {
    # Create temporary directories
    TMUX_INTRAY_CONFIG_DIR="$(mktemp -d)"
    export TMUX_INTRAY_CONFIG_DIR
    TMUX_INTRAY_STATE_DIR="$(mktemp -d)"
    export TMUX_INTRAY_STATE_DIR
    export TMUX_INTRAY_HOOKS_ENABLED=1
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="warn"
    export TMUX_INTRAY_HOOKS_ASYNC=0
    # Ensure config_load doesn't create sample config
    touch "$TMUX_INTRAY_CONFIG_DIR/config.sh"
}

teardown() {
    rm -rf "$TMUX_INTRAY_CONFIG_DIR" "$TMUX_INTRAY_STATE_DIR"
}

# Helper to create a hook script
_create_hook_script() {
    local hook_point="$1"
    local script_name="$2"
    local content="$3"
    local dir="$TMUX_INTRAY_CONFIG_DIR/hooks/$hook_point"
    mkdir -p "$dir"
    local script_path="$dir/$script_name"
    cat >"$script_path" <<EOF
#!/usr/bin/env bash
$content
EOF
    chmod +x "$script_path"
    echo "$script_path"
}

@test "hooks_run with no hooks directory returns success" {
    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "hooks_run with empty hooks directory returns success" {
    mkdir -p "$TMUX_INTRAY_CONFIG_DIR/hooks/pre-add"
    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "hooks_run executes a single hook script" {
    _create_hook_script "pre-add" "01-test.sh" "echo 'Hello from hook'"

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Running pre-add hooks"* ]]
    [[ "$output" == *"Hello from hook"* ]]
}

@test "hooks_run passes environment variables to hook script" {
    # shellcheck disable=SC2016
    _create_hook_script "pre-add" "env.sh" 'echo "MESSAGE=$MESSAGE"'

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add" "MESSAGE=test value"
    [ "$status" -eq 0 ]
    [[ "$output" == *"MESSAGE=test value"* ]]
}

@test "hooks_run respects failure mode ignore" {
    _create_hook_script "pre-add" "fail.sh" "exit 1"
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="ignore"

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Running pre-add hooks"* ]]
    # No error message because ignore mode
    [[ "$output" != *"failed"* ]]
}

@test "hooks_run respects failure mode warn" {
    _create_hook_script "pre-add" "fail.sh" "exit 1"
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="warn"

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Warning:"* ]]
    [[ "$output" == *"failed"* ]]
}

@test "hooks_run respects failure mode abort" {
    _create_hook_script "pre-add" "fail.sh" "exit 1"
    export TMUX_INTRAY_HOOKS_FAILURE_MODE="abort"

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Error:"* ]]
    [[ "$output" == *"aborting"* ]]
}

@test "hooks_run executes hooks in alphabetical order" {
    _create_hook_script "pre-add" "02-second.sh" "echo 'second'"
    _create_hook_script "pre-add" "01-first.sh" "echo 'first'"

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    # Check order: first appears before second in output
    local first_pos
    first_pos=$(grep -n "^first$" <<<"$output" | cut -d: -f1)
    local second_pos
    second_pos=$(grep -n "^second$" <<<"$output" | cut -d: -f1)
    [ -n "$first_pos" ]
    [ -n "$second_pos" ]
    [ "$first_pos" -lt "$second_pos" ]
}

@test "hooks_run with per-hook enable variable" {
    _create_hook_script "pre-add" "test.sh" "echo 'executed'"
    export TMUX_INTRAY_HOOKS_ENABLED_pre_add=0

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [[ "$output" != *"executed"* ]]
}

@test "hooks_run with global hooks disabled" {
    _create_hook_script "pre-add" "test.sh" "echo 'executed'"
    export TMUX_INTRAY_HOOKS_ENABLED=0

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [[ "$output" != *"executed"* ]]
}

@test "hooks_run async mode runs script in background" {
    _create_hook_script "pre-add" "sleep.sh" "sleep 0.1; echo 'done'"
    export TMUX_INTRAY_HOOKS_ASYNC=1

    # shellcheck source=../legacy/hooks.sh
    source "$BATS_TEST_DIRNAME/../legacy/hooks.sh"

    run hooks_run "pre-add"
    [ "$status" -eq 0 ]
    [[ "$output" == *"asynchronously"* ]]
    # Script runs in background, we can't capture output easily
    # Give it a moment to finish
    sleep 0.5
}
