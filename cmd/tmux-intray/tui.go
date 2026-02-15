/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/state"
	"github.com/spf13/cobra"
)

type tuiClient interface {
	LoadSettings() (*settings.Settings, error)
	CreateModel() (*state.Model, error)
	RunProgram(model *state.Model) error
}

// NewTUICmd creates the tui command with explicit dependencies.
func NewTUICmd(client tuiClient) *cobra.Command {
	if client == nil {
		panic("NewTUICmd: client dependency cannot be nil")
	}

	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive terminal UI for notifications",
		Long: `Interactive terminal UI for notifications.

USAGE:
    tmux-intray tui

KEY BINDINGS:
    j/k         Move up/down in the list
    /           Enter search mode

    v           Cycle view mode (compact/detailed/grouped)
    ESC         Exit search mode, or quit TUI
    d           Dismiss selected notification
    r           Mark selected notification as read
    u           Mark selected notification as unread
    Enter       Jump to pane
    :w          Save settings
    q           Quit TUI`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load settings from disk (use defaults if missing/corrupted)
			loadedSettings, err := client.LoadSettings()
			if err != nil {
				colors.Warning(fmt.Sprintf("Failed to load settings, using defaults: %v", err))
				loadedSettings = settings.DefaultSettings()
			}
			colors.Debug("Loaded settings for TUI")

			// Create TUI model
			model, err := client.CreateModel()
			if err != nil {
				colors.Error(fmt.Sprintf("Failed to create TUI model: %v", err))
				os.Exit(1)
			}

			// Store loaded settings reference
			model.SetLoadedSettings(loadedSettings)

			// Apply loaded settings to model
			st := settings.FromSettings(loadedSettings)
			if err := model.FromState(st); err != nil {
				colors.Warning(fmt.Sprintf("Failed to apply settings to TUI model: %v", err))
				// Continue with default settings
			}
			colors.Debug("Applied settings to TUI model")

			// Run the program
			return client.RunProgram(model)
		},
	}
}

// defaultTUIClient is the default implementation.
type defaultTUIClient struct {
	tmuxClient tmux.TmuxClient
}

func (d *defaultTUIClient) LoadSettings() (*settings.Settings, error) {
	return settings.Load()
}

func (d *defaultTUIClient) CreateModel() (*state.Model, error) {
	if d.tmuxClient == nil {
		d.tmuxClient = tmux.NewDefaultClient()
	}
	return state.NewModel(d.tmuxClient)
}

func (d *defaultTUIClient) RunProgram(model *state.Model) error {
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Start the program
	_, err := p.Run()
	if err != nil {
		colors.Error(fmt.Sprintf("Error running TUI: %v", err))
		os.Exit(1)
	}
	return nil
}

// tuiCmd represents the tui command
var tuiCmd = NewTUICmd(&defaultTUIClient{})

func init() {
	cmd.RootCmd.AddCommand(tuiCmd)
}
