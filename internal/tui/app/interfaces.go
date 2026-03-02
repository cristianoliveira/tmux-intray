// Package app provides TUI application adapters for command wiring.
package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

// ProgramRunner defines the interface for running a bubbletea program.
// This abstraction allows for easier testing and swapping of implementations.
type ProgramRunner interface {
	// Run starts the bubbletea program with the given model.
	Run(model tea.Model) error
}

// DefaultProgramRunner is the default implementation of ProgramRunner
// that wraps tea.NewProgram with standard options.
type DefaultProgramRunner struct{}

// NewDefaultProgramRunner creates a new DefaultProgramRunner.
func NewDefaultProgramRunner() *DefaultProgramRunner {
	return &DefaultProgramRunner{}
}

// Run starts a bubbletea program with the given model.
// It uses tea.WithAltScreen and tea.WithMouseCellMotion options by default.
func (r *DefaultProgramRunner) Run(model tea.Model) error {
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}

// SettingsLoader defines the interface for loading settings.
// This abstraction allows for easier testing and swapping of implementations.
type SettingsLoader interface {
	// Load loads and returns the settings.
	Load() (*settings.Settings, error)
}

// DefaultSettingsLoader is the default implementation of SettingsLoader
// that wraps settings.Load for production use.
type DefaultSettingsLoader struct{}

// NewDefaultSettingsLoader creates a new DefaultSettingsLoader.
func NewDefaultSettingsLoader() *DefaultSettingsLoader {
	return &DefaultSettingsLoader{}
}

// Load loads settings using the settings package's Load function.
func (l *DefaultSettingsLoader) Load() (*settings.Settings, error) {
	return settings.Load()
}

// TmuxClientFactory defines the interface for creating tmux.TmuxClient instances.
// This abstraction allows for dependency injection and easier testing.
type TmuxClientFactory interface {
	// NewClient creates a new tmux.TmuxClient instance.
	NewClient() tmux.TmuxClient
}

// DefaultTmuxClientFactory is the default implementation of TmuxClientFactory
// that wraps tmux.NewDefaultClient.
type DefaultTmuxClientFactory struct{}

// NewDefaultTmuxClientFactory creates a new DefaultTmuxClientFactory.
func NewDefaultTmuxClientFactory() *DefaultTmuxClientFactory {
	return &DefaultTmuxClientFactory{}
}

// NewClient creates a new tmux.TmuxClient instance using tmux.NewDefaultClient.
func (f *DefaultTmuxClientFactory) NewClient() tmux.TmuxClient {
	return tmux.NewDefaultClient()
}
