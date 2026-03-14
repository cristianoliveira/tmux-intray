// File: telemetry.go
// Purpose: Implements telemetry event storage for feature usage tracking.
package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/ports"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite/sqlcgen"
)

// Local type aliases for convenience within sqlite package.
// The canonical types are defined in internal/ports package.
type TelemetryEventType = ports.TelemetryEventType
type FeatureUsage = ports.FeatureUsage

// LogTelemetryEvent logs a telemetry event to storage.
func (s *SQLiteStorage) LogTelemetryEvent(timestamp, featureName, featureCategory, contextData string) error {
	if err := validateTelemetryInputs(timestamp, featureName, featureCategory, contextData); err != nil {
		return err
	}
	if timestamp == "" {
		timestamp = utcNow()
	}
	id, err := s.nextTelemetryEventID()
	if err != nil {
		return err
	}

	err = s.queries.InsertTelemetryEvent(context.Background(), sqlcgen.InsertTelemetryEventParams{
		ID:              id,
		Timestamp:       timestamp,
		FeatureName:     featureName,
		FeatureCategory: featureCategory,
		ContextData:     contextData,
	})
	if err != nil {
		return fmt.Errorf("sqlite storage: insert telemetry event: %w", err)
	}

	return nil
}

// GetTelemetryEvents retrieves telemetry events within the specified time range.
// Empty strings for startTime or endTime mean no bound on that side.
func (s *SQLiteStorage) GetTelemetryEvents(startTime, endTime string) ([]TelemetryEventType, error) {
	if startTime != "" {
		if _, err := time.Parse(time.RFC3339, startTime); err != nil {
			return nil, fmt.Errorf("validation error: invalid start_time format '%s', expected RFC3339 format", startTime)
		}
	}
	if endTime != "" {
		if _, err := time.Parse(time.RFC3339, endTime); err != nil {
			return nil, fmt.Errorf("validation error: invalid end_time format '%s', expected RFC3339 format", endTime)
		}
	}

	rows, err := s.queries.ListTelemetryEventsByTimeRange(context.Background(), sqlcgen.ListTelemetryEventsByTimeRangeParams{
		StartTime: startTime,
		EndTime:   endTime,
	})
	if err != nil {
		return nil, fmt.Errorf("sqlite storage: list telemetry events: %w", err)
	}

	events := make([]TelemetryEventType, 0, len(rows))
	for _, row := range rows {
		events = append(events, TelemetryEventType{
			ID:              row.ID,
			Timestamp:       row.Timestamp,
			FeatureName:     row.FeatureName,
			FeatureCategory: row.FeatureCategory,
			ContextData:     row.ContextData,
		})
	}

	return events, nil
}

// GetFeatureUsage returns the usage count for a specific feature.
func (s *SQLiteStorage) GetFeatureUsage(featureName string) (int64, error) {
	if featureName == "" {
		return 0, fmt.Errorf("validation error: feature_name cannot be empty")
	}

	count, err := s.queries.CountFeatureUsage(context.Background(), featureName)
	if err != nil {
		return 0, fmt.Errorf("sqlite storage: count feature usage: %w", err)
	}

	return count, nil
}

// GetAllFeatures returns usage statistics for all features.
func (s *SQLiteStorage) GetAllFeatures() ([]FeatureUsage, error) {
	rows, err := s.queries.GetFeatureUsageStats(context.Background())
	if err != nil {
		return nil, fmt.Errorf("sqlite storage: get feature usage stats: %w", err)
	}

	stats := make([]FeatureUsage, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, FeatureUsage{
			FeatureName:     row.FeatureName,
			FeatureCategory: row.FeatureCategory,
			UsageCount:      int64(row.UsageCount),
		})
	}

	return stats, nil
}

// ClearTelemetryEvents removes telemetry events older than the specified number of days.
// Returns the number of events deleted.
func (s *SQLiteStorage) ClearTelemetryEvents(olderThanDays int) (int64, error) {
	if olderThanDays < 0 {
		return 0, fmt.Errorf("validation error: older_than_days must be >= 0")
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -olderThanDays).Format("2006-01-02T15:04:05Z")

	// Count events before deletion
	count, err := s.queries.CountTelemetryEventsOlderThan(context.Background(), cutoff)
	if err != nil {
		return 0, fmt.Errorf("sqlite storage: count telemetry events for cleanup: %w", err)
	}

	if count == 0 {
		return 0, nil
	}

	// Delete events
	if err := s.queries.DeleteTelemetryEventsOlderThan(context.Background(), cutoff); err != nil {
		return 0, fmt.Errorf("sqlite storage: clear telemetry events: %w", err)
	}

	return count, nil
}

// InitTelemetryTable ensures the telemetry_events table exists.
// This is a no-op since the table is created by schemaSQL during init(),
// but it provides a public method for explicit initialization if needed.
func (s *SQLiteStorage) InitTelemetryTable() error {
	// The table is created automatically in init() via schemaSQL
	// This method is kept for API completeness and explicit initialization scenarios
	return nil
}

func validateTelemetryInputs(timestamp, featureName, featureCategory, contextData string) error {
	if featureName == "" {
		return fmt.Errorf("validation error: feature_name cannot be empty")
	}
	if featureCategory == "" {
		return fmt.Errorf("validation error: feature_category cannot be empty")
	}
	if !validFeatureCategories[featureCategory] {
		return fmt.Errorf("validation error: invalid feature_category '%s', must be one of: cli, tui", featureCategory)
	}
	if timestamp != "" {
		if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
			return fmt.Errorf("validation error: invalid timestamp format '%s', expected RFC3339 format", timestamp)
		}
	}
	if contextData == "" {
		// Default to empty JSON object
		contextData = "{}"
	}
	return nil
}

func (s *SQLiteStorage) nextTelemetryEventID() (int64, error) {
	id, err := s.queries.NextTelemetryEventID(context.Background())
	if err != nil {
		return 0, fmt.Errorf("sqlite storage: get next telemetry event id: %w", err)
	}
	return id, nil
}
