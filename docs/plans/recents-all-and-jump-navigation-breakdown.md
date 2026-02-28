# Recents/All and Jump Navigation Breakdown

## Status

- Phase: implementation completion gate
- Lane: `tmux-intray-i5r`
- Scope: cross-lane test/docs closure for Recents/All and jump behavior

## Implemented Sequencing and Outcomes

1. Startup defaults
   - TUI state initializes with `active_tab = recents`.
   - First filtered dataset shows active notifications only.

2. Recents/All switching
   - `Tab` cycles `recents -> all -> recents`.
   - Search/filter pipeline is reapplied on each switch.
   - Current phase behavior: both tabs operate over the active-only dataset.

3. Jump behavior
   - `Enter` in non-grouped views executes jump for selected notification.
   - Jump requires session, window, and pane identifiers.
   - Successful jump uses explicit `window` and `pane` targets and marks the notification read.
   - Grouped view keeps `Enter` toggle-first on group rows; jumps occur on notification rows.

## Constraints for This Phase

- `all` tab is intentionally active-only in this phase.
- No telemetry additions are included in this phase.

## Verification Coverage Added

- `internal/tui/state/model_test.go`
  - validates Recents default + Recents/All tab cycling with active-only results
  - validates Enter-driven jump passes explicit session/window/pane
- `internal/tui/service/notification_service_test.go`
  - codifies phase constraint that `TabAll` remains active-only

## CLI Reference Impact

- No user-visible keybinding changes were made for this gate.
- `docs/cli/CLI_REFERENCE.md` remains unchanged.
