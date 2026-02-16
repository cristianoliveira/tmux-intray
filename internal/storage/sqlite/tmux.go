// File: tmux.go
// Purpose: Provides tmux client integration for syncing notification status to
// tmux environment variables.
package sqlite

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

var tmuxClient tmux.TmuxClient = tmux.NewDefaultClient()

// SetTmuxClient sets the tmux client used for status updates.
func SetTmuxClient(client tmux.TmuxClient) {
	if client == nil {
		return
	}
	tmuxClient = client
}

func (s *SQLiteStorage) syncTmuxStatusOption() {
	if err := s.updateTmuxStatusOption(s.GetActiveCount()); err != nil {
		colors.Error(fmt.Sprintf("failed to update tmux status: %v", err))
	}
}

func (s *SQLiteStorage) updateTmuxStatusOption(count int) error {
	running, err := tmuxClient.HasSession()
	if err != nil {
		return fmt.Errorf("updateTmuxStatusOption: tmux not available: %w", err)
	}
	if !running {
		return fmt.Errorf("updateTmuxStatusOption: tmux not running")
	}
	if err := tmuxClient.SetStatusOption("@tmux_intray_active_count", fmt.Sprintf("%d", count)); err != nil {
		return fmt.Errorf("updateTmuxStatusOption: failed to set @tmux_intray_active_count to %d: %w", count, err)
	}
	return nil
}
