# Split concerns: CLI/TUI adapters vs shared business logic

## Status
Done on 2026-04-28.

## Outcome
`internal/app/*` is now the shared application/use-case layer for the main CLI business flows.
The CLI side in `cmd/tmux-intray/*` acts as a thinner adapter focused on:

- Cobra wiring
- flags/args parsing
- writer selection
- command-specific adapter behavior

Key results:

- `add` behavior flows through `internal/app/add.go`
- `list` behavior flows through `internal/app/list.go`
- `status` behavior flows through `internal/app/status.go`
- default dependency construction lives in `cmd/tmux-intray/deps.go`
- package-level list test hooks/globals were removed from `cmd/tmux-intray/list.go`
- list adapter tests now use explicit injected clients/writers
- architecture map updated in `.ast.map.json`

## Boundary summary
- `cmd/*`: CLI adapter, flag parsing, Cobra integration, output destination, exit semantics
- `internal/tui/*`: TUI state, rendering, interaction orchestration
- `internal/app/*`: shared use cases and orchestration
- `internal/domain/*`: domain entities/rules
- infra packages: tmux, storage, config, hooks, settings, formatter, search

## Verification
Validated with:

```bash
go test ./cmd/tmux-intray ./internal/app
```

## Notes
This plan is considered complete for the adapter/app split around the main command flows.
Any future architecture cleanup should be tracked as a new plan or `bd` issue rather than extending this completed one.
