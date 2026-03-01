# Recents/All and Jump Navigation Breakdown

## Status

- Phase: implementation completion gate
- Lane: `tmux-intray-i5r`
- Scope: cross-lane test/docs closure for Recents/All and jump behavior

## Problem Statement

Currently the application lacks a clear, discoverable, and keyboard-friendly way for users to navigate recently used windows and panes. Users need to quickly filter between "Recents" (recently active) and "All" sessions, and then jump directly to a target window or pane. Without a consistent tab state, predictable rendering, keyboard bindings, and jump target controls, workflows are interrupted, causing friction for power users who rely on rapid context switching.

## Architecture Reference

The tabs implementation follows the architecture defined in [Tabs Architecture](../design/tabs-architecture.md), which establishes tabs as data views with a clear pipeline: `select dataset by tab → apply search/filters → apply sort → render`.

## Implemented Sequencing and Outcomes

1. Startup defaults
   - TUI state initializes with `active_tab = recents`.
   - First filtered dataset shows active notifications only.

2. Recents/All switching
   - `r` switches to `recents`; `a` switches to `all`.
   - Search/filter pipeline is reapplied on each switch.
   - Current phase behavior: both tabs operate over the active-only dataset.
   - Read-mark action moved to `R` so `r` remains dedicated to tab selection.

3. Jump behavior
   - `Enter` in non-grouped views executes jump for selected notification.
   - Jump requires session + window; pane is optional.
   - If pane exists, jump targets pane; if pane is missing/empty, jump falls back to explicit window target.
   - Successful jump marks the notification read.
   - Grouped view keeps `Enter` toggle-first on group rows; jumps occur on notification rows.

## Constraints for This Phase

- `all` tab is intentionally active-only in this phase.
- No telemetry additions are included in this phase.

## Verification Coverage Added

- `internal/tui/state/model_test.go`
  - validates Recents default + `r`/`a` tab switching with active-only results
  - validates Enter-driven pane jump and explicit window fallback jump
- `internal/tui/service/notification_service_test.go`
  - codifies phase constraint that `TabAll` remains active-only

## Original Product Direction (for reference)

Goal: Provide a lightweight, performant, keyboard-first UI that lets users toggle between Recents and All views, filter and search within those views, and jump to a selected window or pane.

Design principles:
- Minimal UI surface that fits the project's minimalist philosophy
- Keyboard-first: support mnemonics and consistent keybindings
- Predictable state: tab state should be persisted during session interactions
- Fast rendering: only necessary elements are rendered/updated

## Original User Stories and Acceptance Criteria (for reference)

1) As a user, I want to switch between "Recents" and "All" so I can focus on recently used items or see all available sessions.
   - Acceptance criteria:
     - There are two tab controls labeled "Recents" and "All" in the recents UI
     - Pressing a key (e.g., `r` for Recents and `a` for All) switches the active tab
     - The active tab visually differs (highlight/underline)
     - Filtering and search apply within the active tab

2) As a user, I want the recents list to be filtered to show only recently active windows/panes for "Recents" and every tracked session for "All".
   - Acceptance criteria:
     - "Recents" list shows N most recently active sessions (configurable default, e.g., 20)
     - "All" shows the complete list grouped by session/host
     - Switching tabs updates the list content immediately

3) As a user, I want to jump to a selected window or pane from the list.
   - Acceptance criteria:
     - When an item is focused, pressing Enter triggers a jump action
     - Jump actions support two targets: window and pane
     - The jump action is validated and errors are handled gracefully (e.g., target missing)

4) As a user, I want keyboard navigation (up/down, page up/down) and search within the list.
   - Acceptance criteria:
     - Arrow keys move focus between list items
     - PageUp/PageDown scrolls the list
     - `/` focuses a search input and typing filters list

## CLI Reference Impact

- User-visible keybindings are documented in `docs/cli/CLI_REFERENCE.md` (`r`/`a` tab switch, `R` mark read).
