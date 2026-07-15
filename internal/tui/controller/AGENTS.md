# TUI Interaction Controller

Resolve user interactions into explicit navigation and jump decisions.

- Keep interaction decisions separate from rendering and storage.
- Depend on contracts and domain identifiers, not concrete TUI state internals.
- Make stale or missing targets explicit outcomes.
- Test valid targets, absent context, stale panes, and collaborator failures.

Run: `go test ./internal/tui/controller`
