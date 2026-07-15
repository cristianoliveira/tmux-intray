# TUI Rendering

Render current UI/domain state into terminal text.

- Keep rendering pure: state in, string out.
- Do not load data, call tmux, persist settings, or mutate model state.
- Centralize width, truncation, selection, grouping, and style rules.
- Test exact visible behavior across empty, narrow, selected, grouped, and error states.

Run: `go test ./internal/tui/render`
