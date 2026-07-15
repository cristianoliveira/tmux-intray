# Status Templates

Own status presets, template parsing, and variable substitution.

- Keep template expansion deterministic and independent of storage/tmux I/O.
- Define variables and presets in one source of truth.
- Return invalid template or unknown preset errors explicitly.
- Preserve existing preset output unless change is intentional and tested.

Run: `go test ./internal/formatter`
