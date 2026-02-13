/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/spf13/cobra"
)

type settingsClient interface {
	ResetSettings() (*settings.Settings, error)
	LoadSettings() (*settings.Settings, error)
}

// NewSettingsCmd creates the settings command with explicit dependencies.
func NewSettingsCmd(client settingsClient) *cobra.Command {
	if client == nil {
		panic("NewSettingsCmd: client dependency cannot be nil")
	}

	var resetForce bool

	// Parent command
	settingsCmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage TUI settings",
		Long: `Manage TUI settings.

USAGE:
    tmux-intray settings <subcommand>

SUBCOMMANDS:
    reset    Reset settings to defaults
    show     Display current settings

EXAMPLES:
    # Reset settings with confirmation
    tmux-intray settings reset

    # Reset settings without confirmation
    tmux-intray settings reset --force

    # Show current settings
    tmux-intray settings show`,
	}

	// Subcommand: reset
	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset TUI settings to defaults",
		Long: `Reset TUI settings to defaults by deleting the settings file.

USAGE:
    tmux-intray settings reset [OPTIONS]

OPTIONS:
    --force    Reset without confirmation
    -h, --help Show this help

EXAMPLES:
    # Reset settings with confirmation
    tmux-intray settings reset

    # Reset settings without confirmation
    tmux-intray settings reset --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Skip confirmation if --force flag is set or running in CI/test environment
			if !resetForce && os.Getenv("CI") == "" && os.Getenv("BATS_TMPDIR") == "" {
				if !confirmReset() {
					colors.Info("Operation cancelled")
					return nil
				}
			}

			// Reset settings
			_, err := client.ResetSettings()
			if err != nil {
				return fmt.Errorf("failed to reset settings: %w", err)
			}

			colors.Success("Settings reset to defaults")
			return nil
		},
	}

	// Subcommand: show
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Display current settings",
		Long: `Display current TUI settings in JSON format.

USAGE:
    tmux-intray settings show

EXAMPLES:
    # Show current settings
    tmux-intray settings show`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load settings
			currentSettings, err := client.LoadSettings()
			if err != nil {
				return fmt.Errorf("failed to load settings: %w", err)
			}

			// Marshal settings to JSON with indentation
			data, err := json.MarshalIndent(currentSettings, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal settings: %w", err)
			}

			// Display settings
			colors.Info(string(data))
			return nil
		},
	}

	// Add flags
	resetCmd.Flags().BoolVar(&resetForce, "force", false, "Reset without confirmation")

	// Add subcommands to parent
	settingsCmd.AddCommand(resetCmd)
	settingsCmd.AddCommand(showCmd)

	return settingsCmd
}

// settingsCmd represents the settings command
var settingsCmd = NewSettingsCmd(coreClient)

func init() {
	cmd.RootCmd.AddCommand(settingsCmd)
}

// confirmReset asks the user for confirmation before resetting settings.
func confirmReset() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure you want to reset all settings to defaults? (y/N): ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read, assume no
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
