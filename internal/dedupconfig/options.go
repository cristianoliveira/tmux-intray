// Package dedupconfig exposes helpers to read deduplication settings from config.
package dedupconfig

import (
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/dedup"
)

// Load returns deduplication options using current configuration values.
func Load() dedup.Options {
	config.Load()
	criteria := dedup.ParseCriteria(config.Get("dedup.criteria", string(dedup.CriteriaMessage)))
	window := config.GetDuration("dedup.window", 0)
	return dedup.Options{Criteria: criteria, Window: window}
}
