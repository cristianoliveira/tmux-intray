/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
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

const (
	settingsCommandLong = `Manage TUI settings.

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
    tmux-intray settings show`
	resetCommandLong = `Reset TUI settings to defaults by deleting the settings file.

USAGE:
    tmux-intray settings reset [OPTIONS]

OPTIONS:
    --force    Reset without confirmation
    -h, --help Show this help

EXAMPLES:
    # Reset settings with confirmation
    tmux-intray settings reset

    # Reset settings without confirmation
    tmux-intray settings reset --force`
	showCommandLong = `Display current TUI settings in JSON format.

USAGE:
    tmux-intray settings show

EXAMPLES:
    # Show current settings
    tmux-intray settings show`
)

// NewSettingsCmd creates the settings command with explicit dependencies.
func NewSettingsCmd(client settingsClient) *cobra.Command {
	if client == nil {
		panic("NewSettingsCmd: client dependency cannot be nil")
	}

	settingsCmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage TUI settings",
		Long:  settingsCommandLong,
	}

	resetCmd := newResetCmd(client)
	showCmd := newShowCmd(client)

	// Add subcommands to parent
	settingsCmd.AddCommand(resetCmd)
	settingsCmd.AddCommand(showCmd)

	return settingsCmd
}

// newResetCmd creates the reset subcommand.
func newResetCmd(client settingsClient) *cobra.Command {
	var resetForce bool
	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset TUI settings to defaults",
		Long:  resetCommandLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResetCmd(client, resetForce)
		},
	}
	resetCmd.Flags().BoolVar(&resetForce, "force", false, "Reset without confirmation")
	return resetCmd
}

// newShowCmd creates the show subcommand.
func newShowCmd(client settingsClient) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current settings",
		Long:  showCommandLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowCmd(client)
		},
	}
}

// runResetCmd executes the reset subcommand.
func runResetCmd(client settingsClient, force bool) error {
	// Skip confirmation if --force flag is set or running in CI/test environment
	if !force && os.Getenv("CI") == "" && os.Getenv("BATS_TMPDIR") == "" {
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
}

// runShowCmd executes the show subcommand.
func runShowCmd(client settingsClient) error {
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
