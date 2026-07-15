# Settings

## Responsibility

Define, validate, load, save, and reset persisted user preferences, including TUI tabs and grouping choices.

## Boundaries

- Keep defaults centralized and validation explicit.
- Separate settings persistence from configuration semantics.
- Do not let TUI state mutate persisted data except through this package's manager contracts.
- Use atomic or otherwise safe persistence behavior.

## Tests

Use temporary paths. Cover defaults, round trips, malformed files, invalid values, and reset behavior.

Run: `go test ./internal/settings`
