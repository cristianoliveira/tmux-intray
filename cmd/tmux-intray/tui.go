/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/state"
	"github.com/spf13/cobra"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI for notifications",
	Long: `Interactive terminal UI for notifications.

USAGE:
    tmux-intray tui

KEY BINDINGS:
    j/k         Move up/down in the list
    /           Enter search mode
    :           Enter command mode
    ESC         Exit search/command mode, or quit TUI
    d           Dismiss selected notification
    Enter       Jump to pane (or execute command in command mode)
    :w          Save settings
    q           Quit TUI`,
	Run: runTUI,
}

func init() {
	cmd.RootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) {
	// Initialize storage
	storage.Init()

	// Create TmuxClient
	client := tmux.NewDefaultClient()

	// Load settings from disk (use defaults if missing/corrupted)
	loadedSettings, err := settings.Load()
	if err != nil {
		colors.Warning(fmt.Sprintf("Failed to load settings, using defaults: %v", err))
		loadedSettings = settings.DefaultSettings()
	}
	colors.Debug("Loaded settings for TUI")

	// Create TUI model
	model, err := state.NewModel(client)
	if err != nil {
		colors.Error(fmt.Sprintf("Failed to create TUI model: %v", err))
		os.Exit(1)
	}

	// Store loaded settings reference
	model.SetLoadedSettings(loadedSettings)

	// Apply loaded settings to model
	state := settings.FromSettings(loadedSettings)
	if err := model.FromState(state); err != nil {
		colors.Warning(fmt.Sprintf("Failed to apply settings to TUI model: %v", err))
		// Continue with default settings
	}
	colors.Debug("Applied settings to TUI model")

	// Create and run the bubbletea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Start the program
	if _, err := p.Run(); err != nil {
		colors.Error(fmt.Sprintf("Error running TUI: %v", err))
		os.Exit(1)
	}
}
