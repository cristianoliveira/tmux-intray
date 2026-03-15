// File: retention.go
// Purpose: Manages telemetry data retention policy configuration and enforcement.
package sqlite

import (
	"sync"
)

// retentionConfig holds the configured retention period for telemetry data.
// This is a thread-safe wrapper to avoid circular import dependency on the config package.
type retentionConfig struct {
	mu   sync.RWMutex
	days int
}

var (
	// globalRetentionConfig stores the configured retention period.
	// Default is 90 days, can be initialized by SetRetentionDays().
	globalRetentionConfig = &retentionConfig{days: 90}
)

// SetRetentionDays updates the global retention period configuration.
// This should be called during application initialization after config.Load().
// value must be between 7 and 365 days (as validated by config validators).
func SetRetentionDays(days int) {
	globalRetentionConfig.mu.Lock()
	defer globalRetentionConfig.mu.Unlock()

	if days < 7 || days > 365 {
		// Silently ignore invalid values, keep current setting
		return
	}

	globalRetentionConfig.days = days
}

// GetRetentionDays returns the currently configured retention period in days.
func GetRetentionDays() int {
	globalRetentionConfig.mu.RLock()
	defer globalRetentionConfig.mu.RUnlock()
	return globalRetentionConfig.days
}
