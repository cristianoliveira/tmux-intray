# Telemetry Architecture

## Overview

This document describes the architecture of the telemetry system in tmux-intray, a local-only feature usage tracking system designed to inform development decisions without compromising user privacy.

## Core Principles

1. **Local-only**: All data is stored locally; no network calls are ever made
2. **Opt-in**: Telemetry is disabled by default; users must explicitly enable it
3. **Privacy-first**: No personal information is collected; only feature usage patterns
4. **User control**: Users can view, export, clear, and disable telemetry at any time
5. **Minimal overhead**: Lightweight implementation with minimal performance impact

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    tmux-intray System                         │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐ │
│  │   CLI Layer  │     │   TUI Layer  │     │  Hook System │ │
│  │              │     │              │     │              │ │
│  │  Commands:   │     │   Views:     │     │  Events:     │ │
│  │  - add       │     │  - detailed  │     │  - pre-add   │ │
│  │  - list      │     │  - grouped   │     │  - post-add  │ │
│  │  - jump      │     │  - search    │     │  - cleanup   │ │
│  │  - dismiss   │     │              │     │              │ │
│  └──────┬───────┘     └──────┬───────┘     └──────┬───────┘ │
│         │                     │                     │        │
│         │                     │                     │        │
│         │   ┌─────────────────┴─────────────────────┐        │
│         │   │                                      │        │
│         │   │    Telemetry Client Interface       │        │
│         │   │                                      │        │
│         │   │  - TrackFeature(feature, category)  │        │
│         │   │  - TrackUIEvent(event, context)     │        │
│         │   │                                      │        │
│         │   └─────────────────┬────────────────────┘        │
│         │                     │                             │
│         │                     ▼                             │
│         │           ┌──────────────┐                        │
│         │           │ Telemetry    │                        │
│         │           │ Storage Port │                        │
│         │           │              │                        │
│         │           │ - Record()   │                        │
│         │           │ - GetEvents()│                        │
│         │           │ - Clear()    │                        │
│         │           └──────┬───────┘                        │
│         │                  │                                │
│         ▼                  ▼                                │
│  ┌─────────────────────────────────┐                       │
│  │      SQLite Storage Layer       │                       │
│  │                                 │                       │
│  │  ┌─────────────────────────┐    │                       │
│  │  │  notifications table   │    │                       │
│  │  └─────────────────────────┘    │                       │
│  │  ┌─────────────────────────┐    │                       │
│  │  │  telemetry_events table │    │                       │
│  │  │                         │    │                       │
│  │  │  - id (PK)              │    │                       │
│  │  │  - timestamp            │    │                       │
│  │  │  - feature_name         │    │                       │
│  │  │  - feature_category     │    │                       │
│  │  │  - context_data         │    │                       │
│  │  └─────────────────────────┘    │                       │
│  └─────────────────────────────────┘                       │
│                         │                                    │
│                         ▼                                    │
│  ┌─────────────────────────────────┐                       │
│  │  File System                    │                       │
│  │                                 │                       │
│  │  ~/.local/state/tmux-intray/    │                       │
│  │  └── notifications.db           │                       │
│  └─────────────────────────────────┘                       │
│                                                               │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Telemetry CLI Commands                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   telemetry  │  │   telemetry  │  │   telemetry  │        │
│  │   show       │  │   export     │  │   clear      │        │
│  │              │  │              │  │              │        │
│  │  - Summary   │  │  - JSONL     │  │  - Delete    │        │
│  │  - By time   │  │  - Backup    │  │  - By age    │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│                                                               │
│  ┌──────────────┐                                             │
│  │   telemetry  │                                             │
│  │   status     │                                             │
│  │              │                                             │
│  │  - Enabled?  │                                             │
│  │  - Events    │                                             │
│  │  - DB size   │                                             │
│  └──────────────┘                                             │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### 1. Telemetry Collection Flow

```
User Action
   │
   ▼
Command Execution (CLI) / UI Event (TUI)
   │
   ▼
Check: Is telemetry enabled?
   │
   ├─ No → Skip collection
   │
   └─ Yes → Continue
         │
         ▼
    TrackFeature(feature_name, feature_category, context)
         │
         ▼
    Create Telemetry Event:
    - timestamp (now)
    - feature_name
    - feature_category
    - context_data (JSON)
         │
         ▼
    TelemetryStorage.Record(event)
         │
         ▼
    SQLite INSERT into telemetry_events
         │
         ▼
    [Complete]
```

### 2. Telemetry Query Flow (CLI Commands)

```
CLI Command: telemetry show
   │
   ▼
Check: Is telemetry enabled?
   │
   ├─ No → Show "telemetry disabled" message
   │
   └─ Yes → Continue
         │
         ▼
    TelemetryStorage.GetEvents(start_time, end_time)
         │
         ▼
    SQLite SELECT from telemetry_events
    - Filter by timestamp if --days specified
    - Group by feature_name and feature_category
    - Count usage per feature
         │
         ▼
    Format and display results
         │
         ▼
    [Complete]
```

### 3. Telemetry Export Flow

```
CLI Command: telemetry export --output FILE
   │
   ▼
Check: Is telemetry enabled?
   │
   ├─ No → Show "telemetry disabled" message
   │
   └─ Yes → Continue
         │
         ▼
    TelemetryStorage.GetEvents("", "")
         │
         ▼
    SQLite SELECT * FROM telemetry_events
         │
         ▼
    For each event:
    - Marshal to JSON
    - Write to FILE (JSONL format: one JSON per line)
         │
         ▼
    Show success message with event count
         │
         ▼
    [Complete]
```

### 4. Telemetry Clear Flow

```
CLI Command: telemetry clear --days N
   │
   ▼
Check: Is telemetry enabled?
   │
   ├─ No → Show "telemetry disabled" message
   │
   └─ Yes → Continue
         │
         ▼
    Calculate cutoff timestamp (now - N days)
         │
         ▼
    Confirm with user (unless forced)
         │
   └─ Yes → Continue
         │
         ▼
    TelemetryStorage.ClearEvents(cutoff)
         │
         ▼
    SQLite DELETE FROM telemetry_events
    WHERE timestamp < cutoff
         │
         ▼
    Show number of deleted events
         │
         ▼
    [Complete]
```

## Storage Schema

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS telemetry_events (
    id INTEGER PRIMARY KEY,
    timestamp TEXT NOT NULL CHECK (strftime('%s', timestamp) IS NOT NULL),
    feature_name TEXT NOT NULL,
    feature_category TEXT NOT NULL CHECK (feature_category IN ('cli', 'tui')),
    context_data TEXT NOT NULL DEFAULT '{}'
);
```

### Indexes

```sql
-- Optimized for common query patterns
CREATE INDEX IF NOT EXISTS idx_telemetry_events_timestamp
    ON telemetry_events(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_feature_name
    ON telemetry_events(feature_name);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_feature_category
    ON telemetry_events(feature_category);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_feature_name_timestamp
    ON telemetry_events(feature_name, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_feature_category_timestamp
    ON telemetry_events(feature_category, timestamp DESC);
```

### Schema Rationale

- **`id`**: Primary key for efficient queries and foreign key references (future)
- **`timestamp`**: ISO 8601 format for sorting and time-based filtering; CHECK constraint ensures valid timestamps
- **`feature_name`**: The specific feature used (e.g., "add", "list", "jump", "tui:navigate")
- **`feature_category`**: High-level category ("cli" or "tui") for analysis; CHECK constraint ensures valid values
- **`context_data`**: JSON string for additional context; allows flexible data structure without schema changes

### Index Rationale

- **`timestamp DESC`**: Most common query pattern is to show recent events
- **`feature_name`**: Frequently used for filtering by specific feature
- **``feature_category`**: Used for category-based analysis
- **Composite indexes**: Optimize common filter combinations (feature_name + timestamp, feature_category + timestamp)

## Component Architecture

### 1. Telemetry Client (`cmd/tmux-intray/telemetry.go`)

The telemetry client is the primary interface for tracking telemetry events and executing telemetry commands.

**Responsibilities:**

- Track feature usage events
- Implement CLI telemetry commands (show, export, clear, status)
- Format and display telemetry data
- Handle user confirmation for destructive operations

**Key Functions:**

```go
type TelemetryClient interface {
    TrackFeature(name, category string, context map[string]interface{})
    ShowEvents(days int) error
    ExportEvents(outputPath string) error
    ClearEvents(days int) error
    GetStatus() (*TelemetryStatus, error)
}
```

### 2. Telemetry Storage Port (`internal/ports/telemetry.go`)

The storage port defines the interface for telemetry data operations, following the repository pattern.

**Responsibilities:**

- Define contract for telemetry data operations
- Abstract storage implementation from business logic
- Enable testability and future storage backend changes

**Interface Definition:**

```go
type TelemetryStorage interface {
    RecordTelemetryEvent(event *TelemetryEvent) error
    GetTelemetryEvents(startTime, endTime string) ([]TelemetryEvent, error)
    GetFeatureUsageStats() ([]FeatureUsage, error)
    CountTelemetryEventsOlderThan(cutoff string) (int64, error)
    ClearTelemetryEvents(cutoff string) (int64, error)
}
```

### 3. SQLite Storage Implementation (`internal/storage/sqlite/`)

The SQLite storage layer implements the TelemetryStorage interface using sqlc-generated queries.

**Responsibilities:**

- Implement TelemetryStorage interface
- Handle SQLite-specific operations
- Manage database transactions
- Provide efficient data access

**Key Components:**

- `schema.sql`: Database schema definition
- `queries.sql`: SQL queries for telemetry operations
- `sqlcgen/`: sqlc-generated Go code

**Generated Queries:**

- `InsertTelemetryEvent`: Insert new telemetry event
- `ListTelemetryEventsByTimeRange`: Query events by time range
- `GetFeatureUsageStats`: Aggregate usage statistics
- `CountTelemetryEventsOlderThan`: Count events older than cutoff
- `DeleteTelemetryEventsOlderThan`: Delete old events

### 4. Configuration Layer (`internal/config/`)

The configuration layer provides telemetry configuration settings.

**Responsibilities:**

- Read and validate telemetry configuration
- Provide configuration to components that need it
- Handle environment variable overrides

**Configuration Options:**

```toml
telemetry_enabled = false
```

Environment variable: `TMUX_INTRAY_TELEMETRY_ENABLED`

### 5. CLI Command Layer (`cmd/tmux-intray/`)

The CLI command layer exposes telemetry functionality through the `tmux-intray telemetry` command.

**Responsibilities:**

- Parse command-line arguments
- Execute telemetry subcommands (show, export, clear, status)
- Format output for user
- Handle errors and user feedback

**Subcommands:**

- `telemetry show [--days N]`: Display feature usage summary
- `telemetry export --output FILE`: Export data to JSONL
- `telemetry clear [--days N]`: Clear old telemetry data
- `telemetry status`: Show telemetry status

## Integration Points

### 1. CLI Commands Integration

All CLI commands can optionally track their execution:

```go
cmd := &cobra.Command{
    Use:   "add <message>",
    Short: "Add a new item to the tray",
    Run: func(cmd *cobra.Command, args []string) {
        // Track telemetry if enabled
        telemetry.TrackFeature("add", "cli", nil)

        // Command implementation
        // ...
    },
}
```

### 2. TUI Integration

TUI actions and views can be tracked:

```go
// Track TUI view navigation
telemetry.TrackFeature("tui:tab-switch", "tui", map[string]interface{}{
    "from_tab": previousTab,
    "to_tab":   currentTab,
})

// Track TUI actions
telemetry.TrackFeature("tui:dismiss", "tui", map[string]interface{}{
    "view_mode": viewMode,
})
```

### 3. Hooks Integration

Hook system events can be tracked:

```go
// Track hook execution
telemetry.TrackFeature("hook:post-add", "cli", map[string]interface{}{
    "success": success,
    "duration": duration.Milliseconds(),
})
```

## Privacy Guarantees

### 1. Local-Only Storage

All telemetry data is stored locally in `~/.local/state/tmux-intray/notifications.db`. No network calls are ever made to transmit telemetry data.

### 2. Opt-In Model

Telemetry is disabled by default. Users must explicitly enable it by setting:

```bash
export TMUX_INTRAY_TELEMETRY_ENABLED=true
```

### 3. Minimal Data Collection

Only feature usage patterns are collected:

- Feature name
- Feature category
- Timestamp
- Optional context data (no personal information)

No personal information, notification content, or system details are collected.

### 4. User Control

Users have complete control:

- View: `tmux-intray telemetry show`
- Export: `tmux-intray telemetry export --output FILE`
- Clear: `tmux-intray telemetry clear --days N`
- Disable: `export TMUX_INTRAY_TELEMETRY_ENABLED=false`

## Performance Considerations

### 1. Minimal Overhead

- Telemetry tracking is a single database INSERT operation
- No blocking I/O (database writes are fast)
- No network calls (zero latency)
- Optional (can be completely disabled)

### 2. Efficient Queries

- Indexes optimize all common query patterns
- Time-based queries use timestamp index
- Aggregation queries use feature_name and feature_category indexes

### 3. Storage Efficiency

- Lightweight schema (5 columns)
- Minimal data per event (~100 bytes typical)
- JSON context data only when needed
- Optional automatic cleanup of old events

### 4. Concurrency

- SQLite handles concurrent reads and writes
- File locking prevents corruption
- Atomic operations ensure data consistency

## Testing Strategy

### 1. Unit Tests

- Telemetry client logic
- Storage interface implementations
- Configuration parsing
- Data aggregation and formatting

### 2. Integration Tests

- End-to-end telemetry tracking
- CLI command execution
- Data persistence and retrieval
- Export and clear operations

### 3. Performance Tests

- Insert performance under load
- Query performance with large datasets
- Index effectiveness validation

### 4. Privacy Tests

- Verify no network calls are made
- Ensure opt-in behavior works correctly
- Validate that personal information is not collected

## Future Enhancements

### 1. Optional Improvements

- **Data visualization**: Built-in charts and graphs for usage patterns
- **Advanced filtering**: More granular filtering options
- **Custom analytics**: User-defined analytics queries
- **Data encryption**: Optional encryption at rest
- **Integration**: Export to external analytics tools (user-controlled)

### 2. Potential Extensions

- **Feature correlation**: Analyze feature usage patterns and relationships
- **Time-based analysis**: Peak usage times, trends over time
- **User-defined events**: Allow users to track custom events
- **Comparative analysis**: Compare usage across different time periods

### 3. Non-Goals

- Remote data collection or transmission
- User profiling or behavior analysis
- Cross-session or cross-device tracking
- Integration with third-party analytics services
- Monetization or commercial use of telemetry data

## Security Considerations

### 1. File Permissions

- Database file respects standard Unix file permissions
- Only accessible by the owning user
- No setuid or setgid bits

### 2. Input Validation

- All inputs are validated before storage
- SQL injection protection via parameterized queries
- JSON context data is validated

### 3. Error Handling

- Errors are logged but don't crash the application
- Failed telemetry operations don't affect core functionality
- Graceful degradation when telemetry is unavailable

## Troubleshooting

### 1. Telemetry Not Recording

- Check if telemetry is enabled: `tmux-intray telemetry status`
- Verify configuration: `TMUX_INTRAY_TELEMETRY_ENABLED` or `config.toml`
- Check logs: `~/.local/state/tmux-intray/debug.log`

### 2. Performance Issues

- Disable telemetry if experiencing performance problems
- Clear old events: `tmux-intray telemetry clear --days 30`
- Check database size: `tmux-intray telemetry status`

### 3. Data Not Showing

- Verify events exist: `tmux-intray telemetry status`
- Check time range filters: `tmux-intray telemetry show --days 365`
- Export and examine data: `tmux-intray telemetry export --output test.jsonl`

## References

- [Privacy Documentation](../privacy.md) - Detailed privacy policy
- [Configuration Guide](../configuration.md#telemetry) - Configuration options
- [CLI Reference](../cli/CLI_REFERENCE.md#telemetry) - Command documentation
- [Go Package Structure](./go-package-structure.md) - Code organization
- [Project Philosophy](../philosophy.md) - Design principles

---

**Last Updated**: 2026-03-14

This architecture document is part of tmux-intray's commitment to transparency and user privacy.
