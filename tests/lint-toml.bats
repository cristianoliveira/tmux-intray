#!/usr/bin/env bats
# Test TOML linting script

setup() {
    # Ensure lint script is executable
    chmod +x ./scripts/lint-toml.sh 2>/dev/null || true
}

@test "lint-toml.sh passes for valid TOML files" {
    run ./scripts/lint-toml.sh
    echo "Output: $output"
    [ "$status" -eq 0 ]
    [[ "$output" == *"passed"* ]] || [[ "$output" == *"No TOML files found"* ]]
}

@test "lint-toml.sh detects camelCase violations" {
    # Run script on invalid_camelcase.toml fixture
    run ./scripts/lint-toml.sh tests/fixtures/toml/invalid_camelcase.toml
    [ "$status" -eq 1 ]
    [[ "$output" == *"should use snake_case"* ]]
    [[ "$output" == *"camelCase/PascalCase detected"* ]]
}

@test "lint-toml.sh detects PascalCase violations" {
    run ./scripts/lint-toml.sh tests/fixtures/toml/invalid_pascalcase.toml
    [ "$status" -eq 1 ]
    [[ "$output" == *"should use snake_case"* ]]
    [[ "$output" == *"camelCase/PascalCase detected"* ]]
}

@test "lint-toml.sh detects kebab-case violations" {
    run ./scripts/lint-toml.sh tests/fixtures/toml/invalid_kebabcase.toml
    [ "$status" -eq 1 ]
    [[ "$output" == *"should use snake_case"* ]]
    [[ "$output" == *"kebab-case detected"* ]]
}

@test "lint-toml.sh detects quoted key violations" {
    run ./scripts/lint-toml.sh tests/fixtures/toml/invalid_quoted.toml
    [ "$status" -eq 1 ]
    [[ "$output" == *"should use snake_case"* ]]
    # Should detect kebab-case within quotes
    [[ "$output" == *"kebab-case detected"* ]] || [[ "$output" == *"camelCase/PascalCase detected"* ]]
}

@test "lint-toml.sh validates dotted keys per segment" {
    # Use the kebab-case fixture which contains dotted keys with hyphens
    run ./scripts/lint-toml.sh tests/fixtures/toml/invalid_kebabcase.toml
    [ "$status" -eq 1 ]
    # Should mention badge-colors and default-theme
    [[ "$output" == *"badge-colors"* ]]
    [[ "$output" == *"default-theme"* ]]
}

@test "lint-toml.sh passes for valid config files" {
    run ./scripts/lint-toml.sh tests/fixtures/toml/valid_config.toml
    [ "$status" -eq 0 ]
    [[ "$output" == *"passed"* ]] || [[ "$output" == *"No TOML files found"* ]]
}

@test "lint-toml.sh passes for valid settings file" {
    run ./scripts/lint-toml.sh tests/fixtures/toml/valid_settings.toml
    [ "$status" -eq 0 ]
    [[ "$output" == *"passed"* ]] || [[ "$output" == *"No TOML files found"* ]]
}
