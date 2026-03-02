// Package app provides TUI application adapters for command wiring.
package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/state"
)

// Model defines the narrow TUI model surface used by command wiring.
type Model interface {
	tea.Model
	SetLoadedSettings(loadedSettings *settings.Settings)
	FromState(settingsState settings.TUIState) error
}

// Client defines dependencies needed by the tui command.
type Client interface {
	LoadSettings() (*settings.Settings, error)
	CreateModel() (Model, error)
	RunProgram(model Model) error
}

// DefaultClient is the default adapter-based implementation used by CLI wiring.
type DefaultClient struct {
	tmuxClient tmux.TmuxClient
}

// NewDefaultClient creates a default TUI client adapter.
func NewDefaultClient(tmuxClient tmux.TmuxClient) *DefaultClient {
	return &DefaultClient{tmuxClient: tmuxClient}
}

// LoadSettings loads persisted settings.
func (d *DefaultClient) LoadSettings() (*settings.Settings, error) {
	return settings.Load()
}

// CreateModel builds a TUI model implementation.
func (d *DefaultClient) CreateModel() (Model, error) {
	if d.tmuxClient == nil {
		d.tmuxClient = tmux.NewDefaultClient()
	}
	coreInstance := core.NewCore(d.tmuxClient, nil)
	return state.NewModel(d.tmuxClient, coreInstance)
}

// RunProgram starts the bubbletea program.
func (d *DefaultClient) RunProgram(model Model) error {
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	if err != nil {
		colors.Error(fmt.Sprintf("Error running TUI: %v", err))
		os.Exit(1)
	}
	return nil
}
