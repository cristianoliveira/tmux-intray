# CLI Commands

## Responsibility

This package is the composition root and Cobra transport layer. It owns flags, argument binding, dependency factories, command registration, and process exit behavior.

## Boundaries

- Keep commands thin; delegate reusable workflows to `internal/app` or `internal/core`.
- Wire concrete storage, core, search, settings, and TUI dependencies in `deps.go`.
- Do not add domain rules, persistence queries, or tmux process logic here.
- Preserve command output and exit-code compatibility unless change is intentional.

## Tests

Add or update adjacent `*_test.go` files. Inject factories or narrow command interfaces rather than invoking real tmux or user storage.

Run: `go test ./cmd/tmux-intray`
