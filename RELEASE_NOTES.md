# Release Notes

## Unreleased

### SQLite storage backend (stable)

- SQLite is now the only storage backend. TSV backend has been removed.
- All notification data is stored in `$TMUX_INTRAY_STATE_DIR/notifications.db`.
- Automatic database migration on first run with existing data.
- Improved data integrity with SQLite transactions.

### TUI filtering

- New `filters.read` setting persists whether the TUI should show read, unread, or all notifications. It is documented in `docs/configuration.md`.
- Added the `:filter-read <read|unread|all>` command and a footer indicator so you can toggle the filter at runtime; the preference is saved automatically between sessions.
