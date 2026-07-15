# Configuration

## Responsibility

Load, merge, expose, and validate application configuration from TOML and supported environment overrides.

## Boundaries

- TOML keys and sections must use `snake_case`.
- Keep defaults and precedence explicit and centralized.
- Do not perform application workflows while loading configuration.
- Validate user input at the configuration boundary with actionable errors.

## Tests

Use temporary config directories and isolated environment variables. Cover defaults, overrides, malformed TOML, and invalid values.

Run: `go test ./internal/config && ./scripts/lint-toml.sh`
