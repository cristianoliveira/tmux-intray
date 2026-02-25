package app

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// DismissClient defines dependencies required for dismiss operations.
type DismissClient interface {
	DismissNotification(id string) error
	DismissAll() error
}

// DismissInput represents dismiss command inputs after flag parsing.
type DismissInput struct {
	Args          []string
	All           bool
	ConfirmAll    func() bool
	IsCIOrTestEnv func() bool
}

// DismissUseCase coordinates dismiss behavior.
type DismissUseCase struct {
	client DismissClient
}

// NewDismissUseCase creates a new dismiss use-case.
func NewDismissUseCase(client DismissClient) *DismissUseCase {
	if client == nil {
		panic("NewDismissUseCase: client dependency cannot be nil")
	}
	return &DismissUseCase{client: client}
}

// Execute runs dismiss use-case preserving CLI behavior.
func (u *DismissUseCase) Execute(input DismissInput) error {
	if input.All && len(input.Args) > 0 {
		return fmt.Errorf("dismiss: cannot specify both --all and id")
	}
	if !input.All && len(input.Args) == 0 {
		return fmt.Errorf("dismiss: either specify an id or use --all")
	}
	if len(input.Args) > 1 {
		return fmt.Errorf("dismiss: too many arguments")
	}

	if input.All {
		return u.dismissAllWithConfirmation(input)
	}
	return u.dismissSingle(input.Args[0])
}

func (u *DismissUseCase) dismissAllWithConfirmation(input DismissInput) error {
	isCIOrTest := input.IsCIOrTestEnv
	if isCIOrTest == nil {
		isCIOrTest = func() bool {
			return os.Getenv("CI") != "" || os.Getenv("BATS_TMPDIR") != ""
		}
	}

	if !isCIOrTest() {
		confirm := input.ConfirmAll
		if confirm != nil && !confirm() {
			colors.Info("Operation cancelled")
			return nil
		}
	} else {
		colors.Debug("skipping confirmation due to CI/test environment")
	}

	if err := u.client.DismissAll(); err != nil {
		return fmt.Errorf("dismiss: failed to dismiss all: %w", err)
	}

	colors.Success("All active notifications dismissed")
	return nil
}

func (u *DismissUseCase) dismissSingle(id string) error {
	if err := u.client.DismissNotification(id); err != nil {
		return fmt.Errorf("dismiss: failed to dismiss notification: %w", err)
	}

	colors.Success("Notification " + id + " dismissed")
	return nil
}
