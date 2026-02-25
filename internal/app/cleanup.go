package app

import (
	"errors"
	"fmt"
	"io"
)

// CleanupClient defines dependencies required by cleanup command.
type CleanupClient interface {
	EnsureTmuxRunning() bool
	CleanupOldNotifications(days int, dryRun bool) error
}

// CleanupUseCase coordinates cleanup behavior.
type CleanupUseCase struct {
	client CleanupClient
}

// NewCleanupUseCase creates a cleanup use-case.
func NewCleanupUseCase(client CleanupClient) *CleanupUseCase {
	if client == nil {
		panic("NewCleanupUseCase: client dependency cannot be nil")
	}

	return &CleanupUseCase{client: client}
}

// CleanupInput holds parsed cleanup options.
type CleanupInput struct {
	Days         int
	DryRun       bool
	Output       io.Writer
	LoadConfig   func()
	GetConfigInt func(key string, defaultValue int) int
}

// Execute runs cleanup behavior preserving output and validation.
func (u *CleanupUseCase) Execute(input CleanupInput) error {
	if input.LoadConfig != nil {
		input.LoadConfig()
	}

	if !u.client.EnsureTmuxRunning() {
		return errors.New("no tmux session running")
	}

	days := input.Days
	if days == 0 && input.GetConfigInt != nil {
		days = input.GetConfigInt("auto_cleanup_days", 30)
	}

	if days <= 0 {
		return fmt.Errorf("days must be a positive integer")
	}

	if input.Output != nil {
		_, _ = fmt.Fprintf(input.Output, "Starting cleanup of notifications dismissed more than %d days ago\n", days)
	}

	err := u.client.CleanupOldNotifications(days, input.DryRun)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	if input.Output != nil {
		_, _ = fmt.Fprintln(input.Output, "Cleanup completed")
	}

	return nil
}
