package app

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// MarkReadClient defines dependencies required to mark notifications read.
type MarkReadClient interface {
	MarkNotificationRead(id string) error
}

// MarkReadUseCase coordinates mark-read behavior.
type MarkReadUseCase struct {
	client MarkReadClient
}

// NewMarkReadUseCase creates a new mark-read use-case.
func NewMarkReadUseCase(client MarkReadClient) *MarkReadUseCase {
	if client == nil {
		panic("NewMarkReadUseCase: client dependency cannot be nil")
	}
	return &MarkReadUseCase{client: client}
}

// Execute runs mark-read use-case preserving CLI behavior.
func (u *MarkReadUseCase) Execute(id string) error {
	if err := u.client.MarkNotificationRead(id); err != nil {
		return fmt.Errorf("mark-read: %w", err)
	}

	colors.Success(fmt.Sprintf("Notification %s marked as read", id))
	return nil
}
