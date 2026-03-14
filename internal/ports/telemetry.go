// Package ports defines application boundary interfaces used by core services.
package ports

// TelemetryEventType represents a telemetry event returned from storage.
type TelemetryEventType struct {
	ID              int64
	Timestamp       string
	FeatureName     string
	FeatureCategory string
	ContextData     string
}

// FeatureUsage represents feature usage statistics.
type FeatureUsage struct {
	FeatureName     string
	FeatureCategory string
	UsageCount      int64
}

// TelemetryStorage defines the interface for telemetry storage operations.
// This interface bridges cmd and storage layers without requiring cmd to import concrete adapters.
type TelemetryStorage interface {
	GetFeatureUsage(featureName string) (int64, error)
	GetAllFeatures() ([]FeatureUsage, error)
	GetTelemetryEvents(startTime, endTime string) ([]TelemetryEventType, error)
	ClearTelemetryEvents(olderThanDays int) (int64, error)
}
