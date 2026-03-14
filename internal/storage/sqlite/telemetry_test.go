package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLogTelemetryEvent(t *testing.T) {
	s := newTestStorage(t)

	// Test basic event logging
	err := s.LogTelemetryEvent("2026-01-01T12:00:00Z", "test-feature", "cli", `{"key": "value"}`)
	require.NoError(t, err)

	// Test auto-generated timestamp
	err = s.LogTelemetryEvent("", "auto-timestamp-feature", "tui", "{}")
	require.NoError(t, err)

	// Test validation errors
	err = s.LogTelemetryEvent("", "", "cli", "{}")
	require.Error(t, err)
	require.Contains(t, err.Error(), "feature_name cannot be empty")

	err = s.LogTelemetryEvent("", "feature", "invalid", "{}")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid feature_category")
}

func TestGetTelemetryEvents(t *testing.T) {
	s := newTestStorage(t)

	// Add some events with known timestamps
	t1 := "2026-01-01T10:00:00Z"
	t2 := "2026-01-02T11:00:00Z"
	t3 := "2026-01-03T12:00:00Z"

	err := s.LogTelemetryEvent(t1, "feature1", "cli", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent(t2, "feature2", "tui", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent(t3, "feature3", "cli", "{}")
	require.NoError(t, err)

	// Test getting all events
	events, err := s.GetTelemetryEvents("", "")
	require.NoError(t, err)
	require.Len(t, events, 3)

	// Test filtering by start time
	events, err = s.GetTelemetryEvents("2026-01-02T00:00:00Z", "")
	require.NoError(t, err)
	require.Len(t, events, 2)

	// Test filtering by end time
	events, err = s.GetTelemetryEvents("", "2026-01-02T00:00:00Z")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "feature1", events[0].FeatureName)

	// Test filtering by range
	events, err = s.GetTelemetryEvents("2026-01-01T00:00:00Z", "2026-01-03T00:00:00Z")
	require.NoError(t, err)
	require.Len(t, events, 2)

	// Test validation errors
	_, err = s.GetTelemetryEvents("invalid-timestamp", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid start_time format")

	_, err = s.GetTelemetryEvents("", "invalid-timestamp")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid end_time format")
}

func TestGetFeatureUsage(t *testing.T) {
	s := newTestStorage(t)

	// Add events for the same feature
	err := s.LogTelemetryEvent("2026-01-01T10:00:00Z", "feature1", "cli", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent("2026-01-01T11:00:00Z", "feature1", "cli", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent("2026-01-01T12:00:00Z", "feature2", "cli", "{}")
	require.NoError(t, err)

	// Test getting usage count for feature1
	count, err := s.GetFeatureUsage("feature1")
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	// Test getting usage count for feature2
	count, err = s.GetFeatureUsage("feature2")
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	// Test feature that doesn't exist
	count, err = s.GetFeatureUsage("nonexistent")
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	// Test validation error
	_, err = s.GetFeatureUsage("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "feature_name cannot be empty")
}

func TestGetAllFeatures(t *testing.T) {
	s := newTestStorage(t)

	// Add events for different features
	err := s.LogTelemetryEvent("2026-01-01T10:00:00Z", "feature1", "cli", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent("2026-01-01T11:00:00Z", "feature1", "cli", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent("2026-01-01T12:00:00Z", "feature2", "tui", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent("2026-01-01T13:00:00Z", "feature2", "tui", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent("2026-01-01T14:00:00Z", "feature2", "tui", "{}")
	require.NoError(t, err)

	// Test getting all features
	stats, err := s.GetAllFeatures()
	require.NoError(t, err)
	require.Len(t, stats, 2)

	// Verify order (should be sorted by usage count descending)
	require.Equal(t, "feature2", stats[0].FeatureName)
	require.Equal(t, int64(3), stats[0].UsageCount)
	require.Equal(t, "tui", stats[0].FeatureCategory)

	require.Equal(t, "feature1", stats[1].FeatureName)
	require.Equal(t, int64(2), stats[1].UsageCount)
	require.Equal(t, "cli", stats[1].FeatureCategory)

	// Test empty database
	s2 := newTestStorage(t)
	stats, err = s2.GetAllFeatures()
	require.NoError(t, err)
	require.Empty(t, stats)
}

func TestClearTelemetryEvents(t *testing.T) {
	s := newTestStorage(t)

	now := time.Now()

	// Add events at different times
	oldTime := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	recentTime := now.Add(-1 * time.Hour).Format(time.RFC3339)

	err := s.LogTelemetryEvent(oldTime, "old-feature", "cli", "{}")
	require.NoError(t, err)
	err = s.LogTelemetryEvent(recentTime, "recent-feature", "cli", "{}")
	require.NoError(t, err)

	// Count all events
	events, err := s.GetTelemetryEvents("", "")
	require.NoError(t, err)
	require.Len(t, events, 2)

	// Clear events older than 7 days
	deleted, err := s.ClearTelemetryEvents(7)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	// Verify only recent event remains
	events, err = s.GetTelemetryEvents("", "")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "recent-feature", events[0].FeatureName)

	// Clear all events
	deleted, err = s.ClearTelemetryEvents(0)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	events, err = s.GetTelemetryEvents("", "")
	require.NoError(t, err)
	require.Empty(t, events)

	// Test validation error
	_, err = s.ClearTelemetryEvents(-1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "older_than_days must be >= 0")
}

func TestInitTelemetryTable(t *testing.T) {
	s := newTestStorage(t)

	// InitTelemetryTable should be a no-op since the table is created during storage init
	err := s.InitTelemetryTable()
	require.NoError(t, err)

	// Verify we can still use telemetry after calling InitTelemetryTable
	err = s.LogTelemetryEvent("", "test-feature", "cli", "{}")
	require.NoError(t, err)

	events, err := s.GetTelemetryEvents("", "")
	require.NoError(t, err)
	require.Len(t, events, 1)
}

func TestTelemetryEventStorage(t *testing.T) {
	s := newTestStorage(t)

	timestamp := "2026-01-01T12:00:00Z"
	featureName := "my-feature"
	featureCategory := "cli"
	contextData := `{"user": "test", "version": "1.0"}`

	// Log an event
	err := s.LogTelemetryEvent(timestamp, featureName, featureCategory, contextData)
	require.NoError(t, err)

	// Retrieve the event
	events, err := s.GetTelemetryEvents(timestamp, timestamp)
	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, timestamp, event.Timestamp)
	require.Equal(t, featureName, event.FeatureName)
	require.Equal(t, featureCategory, event.FeatureCategory)
	require.Equal(t, contextData, event.ContextData)
	require.Greater(t, event.ID, int64(0))
}

func TestTelemetryIntegration(t *testing.T) {
	s := newTestStorage(t)

	// Simulate real usage: log multiple events for different features
	events := []struct {
		timestamp       string
		featureName     string
		featureCategory string
		contextData     string
	}{
		{"2026-01-01T10:00:00Z", "add-notification", "cli", "{}"},
		{"2026-01-01T11:00:00Z", "list-notifications", "cli", "{}"},
		{"2026-01-01T12:00:00Z", "dismiss-notification", "cli", "{}"},
		{"2026-01-01T13:00:00Z", "add-notification", "tui", "{}"},
		{"2026-01-01T14:00:00Z", "list-notifications", "tui", "{}"},
		{"2026-01-01T15:00:00Z", "add-notification", "cli", "{}"},
	}

	for _, e := range events {
		err := s.LogTelemetryEvent(e.timestamp, e.featureName, e.featureCategory, e.contextData)
		require.NoError(t, err)
	}

	// Check feature usage stats
	stats, err := s.GetAllFeatures()
	require.NoError(t, err)
	// We have 3 distinct feature names, but grouped by category we get 5 entries:
	// add-notification (cli): 2, add-notification (tui): 1
	// list-notifications (cli): 1, list-notifications (tui): 1
	// dismiss-notification (cli): 1
	require.Len(t, stats, 5)

	// Verify stats are sorted by usage count descending
	require.Equal(t, int64(2), stats[0].UsageCount)
	require.Equal(t, "add-notification", stats[0].FeatureName)
	require.Equal(t, "cli", stats[0].FeatureCategory)

	// GetFeatureUsage counts across all categories
	// add-notification should have 3 uses (2 cli + 1 tui)
	addCount, err := s.GetFeatureUsage("add-notification")
	require.NoError(t, err)
	require.Equal(t, int64(3), addCount)

	// list-notifications should have 2 uses (1 cli + 1 tui)
	listCount, err := s.GetFeatureUsage("list-notifications")
	require.NoError(t, err)
	require.Equal(t, int64(2), listCount)

	// dismiss-notification should have 1 use (cli only)
	dismissCount, err := s.GetFeatureUsage("dismiss-notification")
	require.NoError(t, err)
	require.Equal(t, int64(1), dismissCount)

	// Query events by time range
	eventsInRange, err := s.GetTelemetryEvents("2026-01-01T11:00:00Z", "2026-01-01T14:00:00Z")
	require.NoError(t, err)
	require.Len(t, eventsInRange, 4) // 11:00, 12:00, 13:00, 14:00 are all inclusive

	// Clear old events (clear events before "2026-01-01T14:00:00Z")
	deleted, err := s.ClearTelemetryEvents(0) // Clear all events for simplicity
	require.NoError(t, err)
	require.Greater(t, deleted, int64(0))

	// Verify all events are cleared
	allEvents, err := s.GetTelemetryEvents("", "")
	require.NoError(t, err)
	require.Empty(t, allEvents)
}
