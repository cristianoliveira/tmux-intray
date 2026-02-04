package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPrintHelp(t *testing.T) {
	// Create a root command with some subcommands
	rootCmd := &cobra.Command{
		Use:     "tmux-intray",
		Short:   "Test root",
		Long:    "Test root long",
		Version: "0.1.0",
	}
	// Add subcommands in the order expected by printHelp
	addCmd := &cobra.Command{Use: "add", Short: "Add a new item to the tray"}
	listCmd := &cobra.Command{Use: "list", Short: "List notifications with filters and formats"}
	dismissCmd := &cobra.Command{Use: "dismiss ID", Short: "Dismiss a notification"}
	clearCmd := &cobra.Command{Use: "clear", Short: "Clear all items from the tray"}
	cleanupCmd := &cobra.Command{Use: "cleanup", Short: "Clean up old dismissed notifications"}
	toggleCmd := &cobra.Command{Use: "toggle", Short: "Toggle the tray visibility"}
	jumpCmd := &cobra.Command{Use: "jump", Short: "Jump to the pane of a notification"}
	statusCmd := &cobra.Command{Use: "status", Short: "Show notification status summary"}
	statusPanelCmd := &cobra.Command{Use: "status-panel", Short: "Status bar indicator script (for tmux status-right)"}
	followCmd := &cobra.Command{Use: "follow", Short: "Monitor notifications in real-time"}
	helpCmd := &cobra.Command{Use: "help", Short: "Show this help message"}
	versionCmd := &cobra.Command{Use: "version", Short: "Show version information"}

	rootCmd.AddCommand(addCmd, listCmd, dismissCmd, clearCmd, cleanupCmd, toggleCmd, jumpCmd, statusCmd, statusPanelCmd, followCmd, helpCmd, versionCmd)

	// Capture output
	var buf bytes.Buffer
	outputWriter = &buf
	defer func() { outputWriter = nil }()

	PrintHelp(rootCmd)
	output := buf.String()

	// Basic assertions
	if !strings.Contains(output, "tmux-intray v0.1.0") {
		t.Error("Help output should contain version")
	}
	if !strings.Contains(output, "A quiet inbox for things that happen while you're not looking.") {
		t.Error("Help output should contain description")
	}
	if !strings.Contains(output, "USAGE:") {
		t.Error("Help output should contain USAGE section")
	}
	if !strings.Contains(output, "COMMANDS:") {
		t.Error("Help output should contain COMMANDS section")
	}
	if !strings.Contains(output, "OPTIONS:") {
		t.Error("Help output should contain OPTIONS section")
	}
	// Check that each command appears
	for _, cmd := range []string{"add", "list", "dismiss", "clear", "cleanup", "toggle", "jump", "status", "status-panel", "follow", "help", "version"} {
		if !strings.Contains(output, cmd) {
			t.Errorf("Help output should contain command %q", cmd)
		}
	}
}
