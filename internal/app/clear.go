package app

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// ClearClient defines dependencies required for clear operations.
type ClearClient interface {
	ClearTrayItems() error
}

// ClearInput represents clear command contextual inputs.
type ClearInput struct {
	AllowTmuxlessMode func() bool
	ConfirmAll        func() bool
}

// ClearUseCase coordinates clear behavior.
type ClearUseCase struct {
	client ClearClient
}

// NewClearUseCase creates a new clear use-case.
func NewClearUseCase(client ClearClient) *ClearUseCase {
	if client == nil {
		panic("NewClearUseCase: client dependency cannot be nil")
	}
	return &ClearUseCase{client: client}
}

// Execute runs clear use-case preserving CLI behavior.
func (u *ClearUseCase) Execute(input ClearInput) error {
	if input.AllowTmuxlessMode != nil && input.AllowTmuxlessMode() {
		if err := u.client.ClearTrayItems(); err != nil {
			return fmt.Errorf("clear: failed to clear tray items: %w", err)
		}
		colors.Success("cleared")
		return nil
	}

	if input.ConfirmAll != nil && !input.ConfirmAll() {
		colors.Info("Operation cancelled")
		return nil
	}

	if err := u.client.ClearTrayItems(); err != nil {
		return fmt.Errorf("clear: failed to clear tray items: %w", err)
	}

	colors.Success("Tray cleared")
	return nil
}
