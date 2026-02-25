package app

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// AddClient defines dependencies required to add notifications.
type AddClient interface {
	EnsureTmuxRunning() bool
	AddTrayItem(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error)
}

// AddInput represents add command inputs after flag parsing.
type AddInput struct {
	Args          []string
	Session       string
	Window        string
	Pane          string
	PaneCreated   string
	NoAssociate   bool
	Level         string
	AllowTmuxless func() bool
}

// AddUseCase coordinates add notification behavior.
type AddUseCase struct {
	client AddClient
}

// NewAddUseCase creates a new add use-case.
func NewAddUseCase(client AddClient) *AddUseCase {
	if client == nil {
		panic("NewAddUseCase: client dependency cannot be nil")
	}
	return &AddUseCase{client: client}
}

// Execute runs add use-case preserving CLI behavior.
func (u *AddUseCase) Execute(input AddInput) error {
	session := strings.TrimSpace(input.Session)
	window := strings.TrimSpace(input.Window)
	pane := strings.TrimSpace(input.Pane)
	noAssociate := input.NoAssociate

	needsAutoAssociation := !noAssociate && session == "" && window == "" && pane == ""
	if needsAutoAssociation && !u.client.EnsureTmuxRunning() {
		if input.AllowTmuxless != nil && input.AllowTmuxless() {
			colors.Warning("tmux not running; adding notification without pane association")
			noAssociate = true
		} else {
			return fmt.Errorf("tmux not running")
		}
	}

	message := strings.Join(input.Args, " ")
	if err := ValidateAddMessage(message); err != nil {
		return err
	}

	level := input.Level
	if level == "" {
		level = "info"
	}

	_, err := u.client.AddTrayItem(message, session, window, pane, input.PaneCreated, noAssociate, level)
	if err != nil {
		return fmt.Errorf("add: failed to add tray item: %w", err)
	}

	colors.Success("added")
	return nil
}

// ValidateAddMessage checks message length and emptiness.
func ValidateAddMessage(message string) error {
	if len(message) > 1000 {
		return fmt.Errorf("add: message too long (max 1000 characters)")
	}

	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("add: message cannot be empty")
	}

	return nil
}
