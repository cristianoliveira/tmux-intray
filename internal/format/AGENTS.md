# CLI Formatting

## Responsibility

Transform notifications and status data into stable table, simple, compact, JSON, or template-oriented CLI output.

## Boundaries

- Keep formatting pure where possible: values in, text out.
- Do not query storage or tmux from formatters.
- Preserve machine-readable output contracts; JSON changes require compatibility tests.
- Keep ANSI color decisions explicit and avoid colors in structured output.

## Tests

Use exact-output and JSON semantic assertions. Cover empty values, special characters, and each supported format.

Run: `go test ./internal/format`
