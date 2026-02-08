CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY,
    timestamp TEXT NOT NULL CHECK (strftime('%s', timestamp) IS NOT NULL),
    state TEXT NOT NULL CHECK (state IN ('active', 'dismissed')),
    session TEXT NOT NULL DEFAULT '',
    window TEXT NOT NULL DEFAULT '',
    pane TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    pane_created TEXT NOT NULL DEFAULT '' CHECK (pane_created = '' OR strftime('%s', pane_created) IS NOT NULL),
    level TEXT NOT NULL CHECK (level IN ('info', 'warning', 'error', 'critical')),
    read_timestamp TEXT NOT NULL DEFAULT '' CHECK (read_timestamp = '' OR strftime('%s', read_timestamp) IS NOT NULL),
    updated_at TEXT NOT NULL CHECK (strftime('%s', updated_at) IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_notifications_state ON notifications(state);
CREATE INDEX IF NOT EXISTS idx_notifications_level ON notifications(level);
CREATE INDEX IF NOT EXISTS idx_notifications_session ON notifications(session);
CREATE INDEX IF NOT EXISTS idx_notifications_timestamp ON notifications(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_state_timestamp ON notifications(state, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_session_state_timestamp ON notifications(session, state, timestamp DESC);
