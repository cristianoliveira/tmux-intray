# Split concerns: CLI/TUI adapters vs shared business logic

## Goal
Make `internal/app/*` the single source of truth for shared business behavior, while keeping:
- `cmd/*` as the CLI adapter
- `internal/tui/*` as the TUI adapter
- `internal/domain/*` for domain rules
- infra packages for implementation details

## Problem
The current risk is not that the project has two views.
The risk is that the CLI may bypass or duplicate shared business logic that ideally belongs in `internal/app/*`.

## Concerns split

### 1. Inventory and classify current flows
- [ ] Trace `add` flow across `cmd/tmux-intray/*`, `internal/app/*`, `internal/domain/*`, and storage/tmux/config packages
- [ ] Trace `list` flow across the same layers
- [ ] Trace `status` flow across the same layers
- [ ] For each flow, classify code as one of:
  - adapter/presentation
  - application/use-case orchestration
  - domain/business rule
  - infrastructure/IO
- [ ] Record duplication points where `cmd/*` reimplements behavior from `internal/app/*`

### 2. Define target boundaries
- [ ] Write a short boundary note for each layer:
  - `cmd/*`: parse flags, invoke use cases, render CLI output, map errors to exit codes
  - `internal/tui/*`: UI state, events, rendering, invoke use cases
  - `internal/app/*`: shared use cases and orchestration
  - `internal/domain/*`: pure business rules and entities
  - infra packages: tmux, storage, config, hooks, filesystem, sqlite
- [ ] Identify what data shape should cross each boundary
- [ ] Decide which interfaces are canonical and which are duplicates to retire

### 3. Move shared behavior into `internal/app/*`
- [ ] Extract business decisions from `cmd/tmux-intray/add.go` into `internal/app/add.go`
- [ ] Extract business decisions from `cmd/tmux-intray/list.go` into `internal/app/list.go`
- [ ] Extract business decisions from `cmd/tmux-intray/status.go` into `internal/app/status.go`
- [ ] Keep only CLI-specific concerns in `cmd/*`:
  - Cobra wiring
  - flags/args parsing
  - formatting/output
  - exit code mapping
- [ ] Ensure TUI uses the same application-level behavior where appropriate

### 4. Clean dependency injection and composition
- [ ] Identify hidden dependency creation inside use-case/service code
- [ ] Move default dependency construction to the composition root (`cmd/tmux-intray/deps.go` or equivalent)
- [~] Inject config/time/tmux/search dependencies instead of resolving them inside shared logic
  - `internal/app/list.go` no longer creates a tmux client directly; search provider creation now happens in the CLI adapter (`cmd/tmux-intray/list.go`)
  - follow-up: move that factory again into `cmd/tmux-intray/deps.go` so command wiring stays thin
- [ ] Remove or reduce package-level globals/default clients where they cross into business flow

### 5. Simplify overlapping abstractions
- [ ] Compare `internal/domain`, `internal/ports`, and `internal/storage` interfaces for overlap
- [ ] Choose one canonical repository contract per boundary
- [ ] Remove shallow duplicate interfaces that do not add value
- [ ] Keep output-oriented formats (like TSV/string formatting) at adapter edges, not inside shared logic

### 6. Lock behavior with tests first
- [ ] Add characterization tests for current `add` behavior before moving code
- [ ] Add characterization tests for current `list` behavior before moving code
- [ ] Add characterization tests for current `status` behavior before moving code
- [ ] Add adapter-level tests proving CLI delegates to shared application logic
- [ ] Add/adjust TUI tests only where integration with shared logic changes
- [ ] Replace time-sensitive test behavior with injectable clocks/tickers where needed

### 7. Refactor incrementally
- [ ] Refactor one flow at a time, starting with `list` or `add`
- [ ] After each flow move, remove dead/duplicated code immediately
- [ ] Re-run targeted tests after each step
- [ ] Run full validation once all three flows are aligned

### 8. Documentation and follow-up
- [ ] Update architecture docs to explicitly describe CLI/TUI as two adapters over shared use cases
- [ ] Add a small diagram showing adapter → app → domain → infra relationships
- [ ] Capture any leftover follow-up work as explicit tasks in `plans/todo/` or `bd`

## Suggested execution order
1. Characterize `list`
2. Move `list` shared logic into `internal/app/list.go`
3. Make CLI a thin adapter for `list`
4. Repeat for `add`
5. Repeat for `status`
6. Simplify interfaces and DI after behavior is unified

## Done when
- `cmd/*` no longer contains shared business decisions for `add`, `list`, and `status`
- `internal/app/*` is the canonical use-case layer for both CLI and TUI
- domain rules live in `internal/domain/*`
- infra details are injected, not created inside use cases
- tests prove adapters delegate correctly and behavior remains stable
