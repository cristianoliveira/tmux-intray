# Recents | All tabs and jump navigation breakdown

## Scope and outcome

This plan tracks feature `tmux-intray-6r4` and converging lanes through `tmux-intray-i5r` (cross-lane tests/docs gate).

Implemented outcome:
- Recents/All tab contract is strongly typed and persisted with safe fallback to Recents.
- Dataset pipeline is tab-scoped: `select dataset by active tab -> apply filters/search -> apply sort -> render`.
- Recents and All operate on active notifications only in this phase.
- Jump keeps existing Enter behavior: pane jump when pane target exists, explicit window jump when pane target is unavailable.
- No telemetry fields or events were introduced in this phase.

## Lane sequencing and status

1. `tmux-intray-ow7` (tab contract + persistence) - completed
2. `tmux-intray-d93` (tab-scoped filtering semantics) - completed
3. `tmux-intray-6g2` (tab rendering + `r`/`a` keymaps) - completed
4. `tmux-intray-7qq` (explicit window jump, preserve pane jump) - completed
5. `tmux-intray-i5r` (cross-lane tests/docs gate) - this lane validates integration and docs completion

## Cross-lane acceptance checklist

- [x] Default active tab resolves to Recents when session starts.
- [x] `r` and `a` switch tabs and immediately refresh visible list.
- [x] Recents dataset is limited while All shows full active dataset.
- [x] Enter triggers pane jump when pane target is present.
- [x] Enter triggers explicit window jump when pane target is missing.
- [x] Constraints documented: active-only All semantics, no telemetry additions.

## Validation map (tests)

Primary test coverage in:
- `internal/settings/tab_test.go`
- `internal/settings/persistence_test.go`
- `internal/tui/service/notification_service_test.go`
- `internal/tui/render/render_test.go`
- `internal/tui/state/model_test.go`
- `internal/tui/state/model_integration_test.go`
- `internal/core/jump_test.go`

Cross-lane gate specifically validates:
- Recents default + Recents/All switching behavior.
- Jump behavior across both modes (pane and explicit window target).

## Manual verification checklist

- [x] Open TUI and confirm Recents is active by default.
- [x] Press `a` and confirm list expands to All active notifications.
- [x] Press `r` and confirm list returns to Recents slice.
- [x] Select entry with pane context and press Enter (pane jump).
- [x] Select entry without pane context and press Enter (window jump fallback).
- [x] Confirm no telemetry-related files changed in this feature.

## Notes

- This phase intentionally keeps All scoped to active notifications only (no dismissed expansion).
- Explicit window jump is provided through target-resolution behavior, without introducing new global keybindings.
