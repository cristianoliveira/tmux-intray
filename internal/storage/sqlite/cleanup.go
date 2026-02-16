// File: cleanup.go
// Purpose: Handles removal of old notifications based on age thresholds with
// optional dry-run support and hook integration.
package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/hooks"
)

// CleanupOldNotifications removes dismissed notifications older than threshold days.
func (s *SQLiteStorage) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	if daysThreshold < 0 {
		return fmt.Errorf("sqlite storage: days threshold must be >= 0")
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -daysThreshold).Format("2006-01-02T15:04:05Z")
	envVars := []string{
		fmt.Sprintf("CLEANUP_DAYS=%d", daysThreshold),
		fmt.Sprintf("CUTOFF_TIMESTAMP=%s", cutoff),
		fmt.Sprintf("DRY_RUN=%t", dryRun),
	}
	if err := hooks.Run("cleanup", envVars...); err != nil {
		return fmt.Errorf("pre-cleanup hook failed: %w", err)
	}

	countCutoff := cutoff
	if daysThreshold == 0 {
		countCutoff = ""
	}

	deletedCount, err := s.queries.CountDismissedForCleanup(context.Background(), countCutoff)
	if err != nil {
		return fmt.Errorf("sqlite storage: count notifications for cleanup: %w", err)
	}
	if deletedCount == 0 {
		postEnv := append(envVars, "DELETED_COUNT=0")
		if err := hooks.Run("post-cleanup", postEnv...); err != nil {
			return fmt.Errorf("post-cleanup hook failed: %w", err)
		}
		return nil
	}

	if dryRun {
		postEnv := append(envVars, fmt.Sprintf("DELETED_COUNT=%d", deletedCount))
		if err := hooks.Run("post-cleanup", postEnv...); err != nil {
			return fmt.Errorf("post-cleanup hook failed: %w", err)
		}
		return nil
	}

	deleteCutoff := cutoff
	if daysThreshold == 0 {
		deleteCutoff = ""
	}

	if err := s.queries.DeleteDismissedForCleanup(context.Background(), deleteCutoff); err != nil {
		return fmt.Errorf("sqlite storage: cleanup old notifications: %w", err)
	}
	postEnv := append(envVars, fmt.Sprintf("DELETED_COUNT=%d", deletedCount))
	if err := hooks.Run("post-cleanup", postEnv...); err != nil {
		return fmt.Errorf("post-cleanup hook failed: %w", err)
	}

	return nil
}
