# Notification Grouping and Aggregation Design

## Overview

This document describes the design for notification grouping and aggregation in the tmux-intray system. The goal is to provide a robust mechanism for deduplicating and grouping similar notifications based on configurable criteria.

## Data Model

### NotificationGroup Struct

The `NotificationGroup` struct represents a group of notifications that are considered duplicates or similar:

```go
type NotificationGroup struct {
    ID             string
    Criteria       GroupingCriteria
    Representative *Notification   // The representative notification for this group
    Notifications  []*Notification // All notifications in this group
    Count          int             // Total number of notifications in the group
    FirstSeen      time.Time       // Timestamp of the first notification in the group
    LastSeen       time.Time       // Timestamp of the last notification in the group
    Aggregated     bool            // Whether this group has been aggregated for display
}
```

### GroupingCriteria

The system supports multiple grouping criteria:

```go
type GroupingCriteria string

const (
    GroupByExactMatch      GroupingCriteria = "exact_match"
    GroupByMessageAndLevel GroupingCriteria = "message_and_level"
    GroupByMessageAndSource GroupingCriteria = "message_and_source"
    GroupByTimeWindow      GroupingCriteria = "time_window"
)
```

### GroupingConfig

Configuration for the grouping behavior:

```go
type GroupingConfig struct {
    Criteria          GroupingCriteria
    TimeWindow        time.Duration // For time window grouping
    MaxGroupSize      int           // Maximum number of notifications to keep in a group
    EnableAggregation bool          // Whether to aggregate groups for display
}
```

## Integration Points

### 1. Storage Integration

The grouping system works with the existing TSV storage format. Notifications are loaded from storage, grouped, and then can be displayed or processed.

### 2. Existing Notification Structure

The system extends the existing `Notification` struct without modifying it. The grouping is done on top of the existing notification data.

### 3. Grouping Functions

Key functions for grouping:

- `GroupNotificationsByCriteria(notifications []*Notification, config GroupingConfig)`: Main grouping function
- `GetGroupedNotificationsByExactMatch(notifications []*Notification)`: Group by exact message match
- `GetGroupedNotificationsByMessageAndLevel(notifications []*Notification)`: Group by message and level
- `GetGroupedNotificationsByMessageAndSource(notifications []*Notification)`: Group by message and source
- `GetGroupedNotificationsByTimeWindow(notifications []*Notification, window time.Duration)`: Group by time window

## Matching Algorithms

### Exact Match
Groups notifications with identical messages.

### Message and Level
Groups notifications with the same message and severity level.

### Message and Source
Groups notifications with the same message and source (session, window, pane).

### Time Window
Groups notifications within a specified time window that have similar content.

## Aggregation

The system supports aggregation of large groups:
- When a group exceeds `MaxGroupSize`, it can be marked as aggregated
- Aggregated groups show a summary instead of individual notifications
- Configuration controls whether aggregation is enabled

## Performance Considerations

1. **Efficient Grouping**: Uses hash maps for O(1) lookups during grouping
2. **Memory Efficiency**: Stores pointers to notifications rather than copies
3. **Time-Based Sorting**: Groups are sorted by first seen time for display
4. **Configurable Limits**: Prevents excessive memory usage with `MaxGroupSize`

## Usage Example

```go
// Create grouping configuration
config := GroupingConfig{
    Criteria:          GroupByMessageAndLevel,
    TimeWindow:        5 * time.Minute,
    MaxGroupSize:      10,
    EnableAggregation: true,
}

// Group notifications
groups, err := GroupNotificationsByCriteria(notifications, config)
if err != nil {
    // handle error
}

// Process grouped notifications
for _, group := range groups.Groups {
    fmt.Printf("Group: %s, Count: %d\n", group.GetDisplayName(), group.Count)
    for _, notif := range group.Notifications {
        // process individual notifications
    }
}
```

## Extension Points

The design allows for easy extension:
1. Add new `GroupingCriteria` values
2. Implement custom matching algorithms
3. Extend `GroupingConfig` with additional parameters
4. Customize aggregation behavior

This design provides a flexible and efficient solution for notification grouping and deduplication while maintaining compatibility with the existing system architecture.