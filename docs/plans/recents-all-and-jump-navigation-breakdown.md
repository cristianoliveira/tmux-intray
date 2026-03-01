# Recents | All tabs and Jump Navigation Breakdown

## Problem statement

Currently the application lacks a clear, discoverable, and keyboard-friendly way for users to navigate recently used windows and panes. Users need to quickly filter between "Recents" (recently active) and "All" sessions, and then jump directly to a target window or pane. Without a consistent tab state, predictable rendering, keyboard bindings, and jump target controls, workflows are interrupted, causing friction for power users who rely on rapid context switching.

This document defines the product direction, user stories, acceptance criteria, phased implementation plan (with file paths), risks, test plan, and definition of done for implementing Recents|All tabs and jump navigation controls.

## Architecture Reference

The tabs implementation follows the architecture defined in [Tabs Architecture](../design/tabs-architecture.md), which establishes tabs as data views with a clear pipeline: `select dataset by tab → apply search/filters → apply sort → render`.

## Product direction

Goal: Provide a lightweight, performant, keyboard-first UI that lets users toggle between Recents and All views, filter and search within those views, and jump to a selected window or pane.

Design principles:
- Minimal UI surface that fits the project's minimalist philosophy
- Keyboard-first: support mnemonics and consistent keybindings
- Predictable state: tab state should be persisted during session interactions
- Fast rendering: only necessary elements are rendered/updated

## User stories and acceptance criteria

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

## Phased implementation plan

Phase 1: Tab state and filtering (priority: high)
- Implement tab state machine and filtering logic
- Files to modify/add:
  - cmd/tmux-intray/main.go - add CLI flags/config for recents list size (if applicable)
  - internal/recents/state.go - new file: tab state, activeTab enum, filtering utilities
  - internal/recents/filter.go - new file: filter and search helpers
  - internal/store/recents_store.go - integrate recents limit and retrieval
- Acceptance criteria:
  - Unit tests for state transitions and filtering
  - Manual test: toggle tabs and observe filtered results

Phase 2: Tab rendering and keybindings (priority: high)
- Render tabs in the UI, highlight active tab, add keybindings to switch
- Files to modify/add:
  - internal/ui/recents_view.go - render tabs and list container
  - internal/ui/keybindings.go - register tab keybindings (`r`, `a`, `/`, Enter)
  - assets/styles.go or similar - small visual indicators for active tab
- Acceptance criteria:
  - UI shows tabs and responds to key presses
  - Visual indicator for active tab
  - Integration tests for keybinding handling

Phase 3: Jump window/pane actions (priority: medium)
- Implement actions that perform the actual jump to window/pane
- Files to modify/add:
  - internal/actions/jump.go - logic to resolve and perform jump (window/pane)
  - internal/tmux/client.go - expose functions for focusing/attaching windows and panes
  - internal/ui/recents_view.go - call actions on Enter
- Acceptance criteria:
  - Jump actions succeed for valid targets
  - Errors displayed in UI for invalid targets
  - Integration/e2e tests (mock tmux client)

Phase 4: Tests and docs updates (priority: medium)
- Add tests, update docs, create usage examples
- Files to modify/add:
  - internal/recents/*_test.go - unit tests for filtering and state
  - internal/actions/jump_test.go - unit tests with mock client
  - docs/usage/recents.md - user-facing docs
  - docs/plans/recents-all-and-jump-navigation-breakdown.md - this file (implementation plan)
- Acceptance criteria:
  - All tests pass locally
  - Documentation updated with examples and keybindings

## Risks

- Incomplete tmux client API: may require expansion or mocks for tests
- Performance regressions if list rendering is naive for large "All" lists
- Edge cases where targets are stale (closed windows/panes)
- Keyboard shortcut collisions with existing bindings

## Test plan

Unit tests:
- Tab state transitions (activate/deactivate)
- Filtering and search behavior (match/no match, case-insensitive)
- Actions: resolve target IDs and handle missing targets

Integration tests:
- UI keybindings: simulate key events and assert view changes
- Jump actions with a mocked tmux client that verifies commands invoked

Manual acceptance tests:
- Toggle between Recents and All with keyboard
- Search for an item and jump to window or pane
- Attempt to jump to a removed target and confirm error handling

## Definition of Done

- Code implements tab state/filtering, UI rendering, keybindings, and jump actions
- Unit and integration tests cover core functionality and pass
- Documentation updated (this plan + user docs)
- bd issues created and linked for tracking
- No regressions in existing functionality

