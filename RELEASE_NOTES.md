# Release Notes

## Unreleased

### SQLite storage backend (stable)

- SQLite is now the only storage backend. TSV backend has been removed.
- All notification data is stored in `$TMUX_INTRAY_STATE_DIR/notifications.db`.
- Automatic database migration on first run with existing data.
- Improved data integrity with SQLite transactions.
