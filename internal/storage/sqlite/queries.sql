-- name: NextNotificationID :one
SELECT COALESCE(MAX(id), 0) + 1 AS next_id
FROM notifications;

-- name: CreateNotification :exec
INSERT INTO notifications (
    id,
    timestamp,
    state,
    session,
    window,
    pane,
    message,
    pane_created,
    level,
    read_timestamp,
    updated_at
)
VALUES (?, ?, 'active', ?, ?, ?, ?, ?, ?, '', ?);

-- name: GetNotificationLineByID :one
SELECT id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp
FROM notifications
WHERE id = ?;

-- name: GetNotificationForHooksByID :one
SELECT id, timestamp, state, session, window, pane, message, pane_created, level
FROM notifications
WHERE id = ?;

-- name: ListActiveNotificationsForHooks :many
SELECT id, timestamp, state, session, window, pane, message, pane_created, level
FROM notifications
WHERE state = 'active'
ORDER BY id ASC;

-- name: ListNotifications :many
SELECT id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp
FROM notifications
WHERE (sqlc.arg(state_filter) = '' OR sqlc.arg(state_filter) = 'all' OR state = sqlc.arg(state_filter))
  AND (sqlc.arg(level_filter) = '' OR level = sqlc.arg(level_filter))
  AND (sqlc.arg(session_filter) = '' OR session = sqlc.arg(session_filter))
  AND (sqlc.arg(window_filter) = '' OR window = sqlc.arg(window_filter))
  AND (sqlc.arg(pane_filter) = '' OR pane = sqlc.arg(pane_filter))
  AND (sqlc.arg(older_than_cutoff) = '' OR timestamp < sqlc.arg(older_than_cutoff))
  AND (sqlc.arg(newer_than_cutoff) = '' OR timestamp > sqlc.arg(newer_than_cutoff))
ORDER BY id ASC;

-- name: DismissNotificationByID :execresult
UPDATE notifications
SET state = 'dismissed', updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: UpdateReadTimestampByID :execresult
UPDATE notifications
SET read_timestamp = sqlc.arg(read_timestamp), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: CountDismissedForCleanup :one
SELECT COUNT(1)
FROM notifications
WHERE state = 'dismissed'
  AND (sqlc.arg(cutoff) = '' OR timestamp < sqlc.arg(cutoff));

-- name: DeleteDismissedForCleanup :exec
DELETE FROM notifications
WHERE state = 'dismissed'
  AND (sqlc.arg(cutoff) = '' OR timestamp < sqlc.arg(cutoff));

-- name: CountActiveNotifications :one
SELECT COUNT(1)
FROM notifications
WHERE state = 'active';

-- name: UpsertNotification :exec
INSERT INTO notifications (
    id,
    timestamp,
    state,
    session,
    window,
    pane,
    message,
    pane_created,
    level,
    read_timestamp,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    timestamp = excluded.timestamp,
    state = excluded.state,
    session = excluded.session,
    window = excluded.window,
    pane = excluded.pane,
    message = excluded.message,
    pane_created = excluded.pane_created,
    level = excluded.level,
    read_timestamp = excluded.read_timestamp,
    updated_at = excluded.updated_at;
