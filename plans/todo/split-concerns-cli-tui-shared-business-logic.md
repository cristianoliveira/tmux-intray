# Split concerns: CLI/TUI adapters vs shared business logic

## Goal
Make `internal/app/*` the single source of truth for shared business behavior, while keeping:
- `cmd/*` as CLI adapters
- `internal/tui/*` as TUI adapters
- `internal/domain/*` for domain rules
- infra packages for implementation details

## Current state
Completed since the original plan:
- `add` behavior is coordinated by `internal/app/add.go`; `cmd/tmux-intray/add.go` is mostly Cobra wiring plus adapter glue.
- `list` behavior is coordinated by `internal/app/list.go`; tmux display-name loading is injected from `cmd/tmux-intray/deps.go`.
- `status` behavior is coordinated by `internal/app/status.go`; preset lookup is injected from `cmd/tmux-intray/deps.go`.
- Hidden tmux/search/preset construction was moved out of app use cases.
- Package-level list command test hooks were removed.
- Human-readable list output work is complete and no longer belongs in `plans/todo/`.

## Remaining risks
- Some CLI files still expose thin compatibility/helper wrappers around app behavior.
- It is not yet proven that TUI paths reuse the same app behavior everywhere they should.
- `internal/domain`, `internal/ports`, and storage interfaces may overlap.
- Architecture docs do not yet clearly describe adapter → app → domain → infra boundaries.

## TODO

### 1. Audit remaining adapter leakage
- [ ] Review `cmd/tmux-intray/add.go` for leftover wrappers such as `validateMessage` and decide whether tests still need them.
- [ ] Review `cmd/tmux-intray/list.go` for adapter helpers that could be narrowed or moved.
- [ ] Review `cmd/tmux-intray/status.go` wrappers such as `countByLevel` and `paneCounts`.
- [ ] Record any `cmd/*` logic that is more than flags, argument parsing, rendering, or error mapping.

### 2. Verify TUI/app alignment
- [ ] Trace add/list/status-equivalent TUI flows through `internal/tui/*`.
- [ ] Identify where TUI duplicates app behavior instead of calling use cases.
- [ ] Decide which app use cases should be shared with TUI and which TUI behavior is legitimately UI-specific.
- [ ] Add adapter-level tests where TUI or CLI delegation is not covered.

### 3. Simplify boundaries and interfaces
- [ ] Compare `internal/domain`, `internal/ports`, and `internal/storage` interfaces for overlap.
- [ ] Choose one canonical repository contract per boundary.
- [ ] Remove shallow duplicate interfaces that do not add value.
- [ ] Keep output-oriented formats at adapter edges, not inside shared logic.

### 4. Document target architecture
- [ ] Add a short boundary note for each layer:
  - `cmd/*`: parse flags, invoke use cases, render CLI output, map errors to exit codes
  - `internal/tui/*`: UI state, events, rendering, invoke use cases
  - `internal/app/*`: shared use cases and orchestration
  - `internal/domain/*`: pure business rules and entities
  - infra packages: tmux, storage, config, hooks, filesystem, sqlite
- [ ] Add a small adapter → app → domain → infra diagram.
- [ ] Update docs to name `internal/app/*` as the canonical use-case layer.

### 5. Validate after each cleanup
- [ ] Run targeted tests for touched packages.
- [ ] Run `go test ./...`.
- [ ] Run `make lint`.

## Done when
- `cmd/*` contains only adapter concerns for add/list/status.
- TUI either reuses app use cases or has documented UI-specific reasons not to.
- Domain rules live in `internal/domain/*`.
- Infra details are injected, not created inside use cases.
- Tests prove adapters delegate correctly and behavior remains stable.
