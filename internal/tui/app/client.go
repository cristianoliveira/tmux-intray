// Package app provides TUI application adapters for command wiring.
package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
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
	tmuxClientFactory TmuxClientFactory
	programRunner     ProgramRunner
	settingsLoader    SettingsLoader
}

// NewDefaultClient creates a default TUI client adapter.
// If tmuxClientFactory is nil, a DefaultTmuxClientFactory will be used.
// If programRunner is nil, a DefaultProgramRunner will be used.
// If settingsLoader is nil, a DefaultSettingsLoader will be used.
func NewDefaultClient(tmuxClientFactory TmuxClientFactory, programRunner ProgramRunner, settingsLoader SettingsLoader) *DefaultClient {
	if tmuxClientFactory == nil {
		tmuxClientFactory = NewDefaultTmuxClientFactory()
	}
	if programRunner == nil {
		programRunner = NewDefaultProgramRunner()
	}
	if settingsLoader == nil {
		settingsLoader = NewDefaultSettingsLoader()
	}
	return &DefaultClient{
		tmuxClientFactory: tmuxClientFactory,
		programRunner:     programRunner,
		settingsLoader:    settingsLoader,
	}
}

// LoadSettings loads persisted settings using the injected SettingsLoader.
func (d *DefaultClient) LoadSettings() (*settings.Settings, error) {
	return d.settingsLoader.Load()
}

// CreateModel builds a TUI model implementation.
func (d *DefaultClient) CreateModel() (Model, error) {
	tmuxClient := d.tmuxClientFactory.NewClient()
	return state.NewModel(tmuxClient)
}

// RunProgram starts the bubbletea program using the configured ProgramRunner.
func (d *DefaultClient) RunProgram(model Model) error {
	err := d.programRunner.Run(model)
	if err != nil {
		colors.Error(fmt.Sprintf("Error running TUI: %v", err))
		return err
	}
	return nil
}
