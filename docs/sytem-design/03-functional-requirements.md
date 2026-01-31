# Functional Requirements

## FR-1 Emit notifications
- Must allow emitting notifications from any pane
- Single tmux command
- Automatically captures session, window, and pane

## FR-2 Persist notifications
- Notifications persist for tmux lifetime
- Survive window/pane changes and detach/attach

## FR-3 Notification lifecycle
- Notifications have states: active, dismissed
- State transitions are explicit

## FR-4 Status indicator
- Status bar shows presence/count of notifications
- Must be non-intrusive
- O(1) access (no scanning logs)

## FR-5 List notifications
- Command or keybinding opens list
- Shows message, source, timestamp

## FR-6 Jump to source
- Selecting notification switches to correct session/window/pane
- User-initiated only

## FR-7 Dismiss notifications
- Dismiss individual notifications
- Clear all notifications
- Updates status indicator immediately

## Nice-to-haves
- Notification levels (info/warn/error)
- Hooks and automation
- Filtering and grouping
- Optional OS notification bridge
