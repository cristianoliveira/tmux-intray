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

// settingsCmd represents the settings command
var settingsCmd = &cobra.Command{
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

// resetCmd represents the settings reset command
var resetCmd = &cobra.Command{
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
	Run: runSettingsReset,
}

var resetForce bool

// showCmd represents the settings show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current settings",
	Long: `Display current TUI settings in JSON format.

USAGE:
    tmux-intray settings show

EXAMPLES:
    # Show current settings
    tmux-intray settings show`,
	Run: runSettingsShow,
}

// resetSettingsFunc is the function used to reset settings. Can be changed for testing.
var resetSettingsFunc = func() (*settings.Settings, error) {
	return settings.Reset()
}

// loadSettingsFunc is the function used to load settings. Can be changed for testing.
var loadSettingsFunc = func() (*settings.Settings, error) {
	return settings.Load()
}

func init() {
	cmd.RootCmd.AddCommand(settingsCmd)

	// Add subcommands
	settingsCmd.AddCommand(resetCmd)
	settingsCmd.AddCommand(showCmd)

	// Add flags
	resetCmd.Flags().BoolVar(&resetForce, "force", false, "Reset without confirmation")
}

// runSettingsReset executes the settings reset command.
func runSettingsReset(cmd *cobra.Command, args []string) {
	// Skip confirmation if --force flag is set or running in CI/test environment
	if !resetForce && os.Getenv("CI") == "" && os.Getenv("BATS_TMPDIR") == "" {
		if !confirmReset() {
			colors.Info("Operation cancelled")
			return
		}
	}

	// Reset settings
	_, err := resetSettingsFunc()
	if err != nil {
		colors.Error(fmt.Sprintf("Failed to reset settings: %v", err))
		os.Exit(1)
	}

	colors.Success("Settings reset to defaults")
}

// runSettingsShow executes the settings show command.
func runSettingsShow(cmd *cobra.Command, args []string) {
	// Load settings
	currentSettings, err := loadSettingsFunc()
	if err != nil {
		colors.Error(fmt.Sprintf("Failed to load settings: %v", err))
		os.Exit(1)
	}

	// Marshal settings to JSON with indentation
	data, err := json.MarshalIndent(currentSettings, "", "  ")
	if err != nil {
		colors.Error(fmt.Sprintf("Failed to marshal settings: %v", err))
		os.Exit(1)
	}

	// Display settings
	colors.Info(string(data))
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
