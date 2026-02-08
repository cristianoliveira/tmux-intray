# SQLite Schema Design for Notifications Storage

## Overview

This document defines the SQLite schema for migrating notification storage from TSV to SQLite.
It preserves current behavior and field compatibility while adding stronger constraints,
better query performance, and a path for future extensibility.

## Goals

- Preserve all current TSV fields and semantics.
- Enforce data integrity with explicit constraints.
- Support current query patterns (`state`, `level`, `session`, timestamp filters).
- Keep migration low-risk and reversible during rollout.

## Current TSV Fields

The current TSV schema stores 10 fields in this order:

1. `id`
2. `timestamp`
3. `state`
4. `session`
5. `window`
6. `pane`
7. `message`
8. `pane_created`
9. `level`
10. `read_timestamp`

## Proposed SQLite Schema

### Primary Table: `notifications`

```sql
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY,
    timestamp TEXT NOT NULL
        CHECK (strftime('%s', timestamp) IS NOT NULL),
    state TEXT NOT NULL
        CHECK (state IN ('active', 'dismissed')),
    session TEXT NOT NULL DEFAULT '',
    window TEXT NOT NULL DEFAULT '',
    pane TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    pane_created TEXT NOT NULL DEFAULT ''
        CHECK (pane_created = '' OR strftime('%s', pane_created) IS NOT NULL),
    level TEXT NOT NULL
        CHECK (level IN ('info', 'warning', 'error', 'critical')),
    read_timestamp TEXT NOT NULL DEFAULT ''
        CHECK (read_timestamp = '' OR strftime('%s', read_timestamp) IS NOT NULL),
    updated_at TEXT NOT NULL
        CHECK (strftime('%s', updated_at) IS NOT NULL)
);
```

### Column Mapping (TSV -> SQLite)

| TSV field | SQLite column | Type | Nullability | Notes |
|-----------|---------------|------|-------------|-------|
| `id` | `id` | `INTEGER` | `NOT NULL` | Primary key. Preserve existing numeric IDs during migration. |
| `timestamp` | `timestamp` | `TEXT` | `NOT NULL` | RFC3339/ISO-8601 UTC timestamp. |
| `state` | `state` | `TEXT` | `NOT NULL` | Allowed values: `active`, `dismissed`. |
| `session` | `session` | `TEXT` | `NOT NULL` | Empty string when unknown. |
| `window` | `window` | `TEXT` | `NOT NULL` | Empty string when unknown. |
| `pane` | `pane` | `TEXT` | `NOT NULL` | Empty string when unknown. |
| `message` | `message` | `TEXT` | `NOT NULL` | Unescaped message body. |
| `pane_created` | `pane_created` | `TEXT` | `NOT NULL` | Empty string or valid timestamp. |
| `level` | `level` | `TEXT` | `NOT NULL` | Allowed values: `info`, `warning`, `error`, `critical`. |
| `read_timestamp` | `read_timestamp` | `TEXT` | `NOT NULL` | Empty string means unread; otherwise valid timestamp. |

Additional SQLite-only column:

| Column | Type | Nullability | Purpose |
|--------|------|-------------|---------|
| `updated_at` | `TEXT` | `NOT NULL` | Last mutation time for each record, for sync/audit/debug workflows. |

## Constraints and Rationale

### State and Level Constraints

- `state` is constrained to `active` or `dismissed` to match runtime filtering and mutation logic.
- `level` is constrained to `info`, `warning`, `error`, `critical` to match accepted inputs.

### Timestamp Validity Constraints

- `timestamp` and `updated_at` must be parseable by SQLite date functions.
- `pane_created` and `read_timestamp` allow empty string (`''`) for backward compatibility with TSV optional fields.
- Non-empty optional timestamps must be parseable by SQLite date functions.

### Suggested Trigger for `updated_at`

```sql
CREATE TRIGGER notifications_set_updated_at
AFTER UPDATE ON notifications
FOR EACH ROW
BEGIN
    UPDATE notifications
    SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
```

Implementation note: if recursion is enabled, use a `BEFORE UPDATE` trigger that assigns
`NEW.updated_at` instead of issuing an `UPDATE` statement.

## Index Strategy

Recommended indexes:

```sql
CREATE INDEX idx_notifications_state
    ON notifications(state);

CREATE INDEX idx_notifications_level
    ON notifications(level);

CREATE INDEX idx_notifications_session
    ON notifications(session);

CREATE INDEX idx_notifications_timestamp
    ON notifications(timestamp DESC);

CREATE INDEX idx_notifications_read_timestamp
    ON notifications(read_timestamp)
    WHERE read_timestamp <> '';

CREATE INDEX idx_notifications_state_timestamp
    ON notifications(state, timestamp DESC);

CREATE INDEX idx_notifications_session_state_timestamp
    ON notifications(session, state, timestamp DESC);
```

Rationale:

- Single-column indexes satisfy straightforward filter queries.
- `state,timestamp` speeds common "active recent" and "dismissed recent" views.
- `session,state,timestamp` supports tmux-context scoped listing efficiently.
- Partial index on `read_timestamp` keeps the index small while accelerating read/unread queries.

## Optional FTS5 for Message Search

FTS5 is recommended only if message search becomes a frequent workflow.

Suggested design:

- Create virtual table `notifications_fts(message, content='notifications', content_rowid='id')`.
- Keep synchronized via insert/update/delete triggers.
- Query by joining `notifications_fts` with `notifications` on `rowid = id`.

Caveats:

- Adds write amplification and migration complexity.
- Requires trigger maintenance when schema evolves.
- Tokenization behavior should be validated for punctuation-heavy terminal messages.

## Migration Strategy (TSV -> SQLite)

Two rollout options are viable. Prefer phased rollout for lower risk.

### Option A: Phased Migration (Recommended)

1. Add SQLite backend behind a feature flag.
2. Read from TSV, dual-write TSV + SQLite for a stabilization window.
3. Backfill historical TSV records into SQLite in ID order.
4. Validate parity:
   - Row counts by `state` and `level`.
   - Spot-check by `id` for message and timestamps.
   - Verify unread/read semantics (`read_timestamp = ''` handling).
5. Switch reads to SQLite once parity checks pass.
6. Disable TSV writes and keep a temporary TSV rollback path.
7. Remove TSV dependency after one or more successful releases.

### Option B: One-Shot Migration

1. Stop writes (maintenance window or lock-based freeze).
2. Import TSV into SQLite in one transaction.
3. Run validation checks (counts, per-ID checks, timestamp parse checks).
4. Switch to SQLite and archive TSV snapshot.

### Backfill and Validation Details

- Preserve existing `id` values exactly; do not re-number.
- Normalize malformed/short rows before import (current TSV parser pads optional fields).
- Unescape TSV-escaped message content before insert.
- Set `updated_at` during backfill to:
  - migration time for simplicity, or
  - `timestamp` if historical mutation time is not available.
- Record import errors with line number and continue with a reject log; fail fast if reject rate exceeds threshold.

## Future Extensibility

### Tags

Add normalized tag tables:

- `tags(id INTEGER PRIMARY KEY, name TEXT UNIQUE NOT NULL)`
- `notification_tags(notification_id INTEGER NOT NULL, tag_id INTEGER NOT NULL, PRIMARY KEY(notification_id, tag_id), FOREIGN KEY...)`

This allows filtering/grouping without schema churn.

### Custom Metadata

Add flexible metadata support:

- `metadata_json TEXT NOT NULL DEFAULT '{}'` on `notifications`, or
- `notification_metadata(notification_id INTEGER, key TEXT, value TEXT, PRIMARY KEY(notification_id, key))`.

JSON is easier to evolve; key-value table is easier to index by specific keys.

### Additional Operational Fields

Future-safe columns that may be useful:

- `source` (command/hook/system origin)
- `dedupe_key` (coalescing repeated events)
- `expires_at` (auto-expiry policy)

## Summary

This schema preserves all current TSV fields, enforces stronger integrity guarantees, and
supports current query patterns through targeted indexes. The phased migration path reduces
risk while leaving room for FTS search and richer metadata in later iterations.
