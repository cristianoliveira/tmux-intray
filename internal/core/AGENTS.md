# Core Application Logic

## Responsibility

Own notification lifecycle and tmux-aware application behavior: add, list, dismiss, read state, cleanup, context association, and navigation.

## Boundaries

- Depend on ports and injected collaborators; do not construct CLI commands.
- Keep storage SQL in `internal/storage` and low-level tmux execution in `internal/tmux`.
- Reuse domain types and validation instead of parallel representations.
- Preserve notification context: session, window, pane, and pane creation identity.

## Tests

Prefer fakes and temporary storage. Cover associated and tmuxless behavior plus dependency failures.

Run: `go test ./internal/core`
