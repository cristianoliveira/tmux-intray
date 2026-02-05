/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// jumpCmd represents the jump command
var jumpCmd = &cobra.Command{
	Use:   "jump",
	Short: "Jump to the pane of a notification",
	Long: `Jump to the pane of a notification.

USAGE:
    tmux-intray jump <id>

DESCRIPTION:
    Navigates to the tmux pane where the notification originated. The pane
    must still exist; if it doesn't, the command falls back to the window.

ARGUMENTS:
    <id>    Notification ID (as shown in 'tmux-intray list --format=table')

EXAMPLES:
    # Jump to pane of notification with ID 42
    tmux-intray jump 42`,
	Run: runJump,
}

// Field indices matching storage package
const (
	fieldID        = 0
	fieldTimestamp = 1
	fieldState     = 2
	fieldSession   = 3
	fieldWindow    = 4
	fieldPane      = 5
	fieldMessage   = 6
	// fieldPaneCreated = 7
	// fieldLevel       = 8
)

// ensureTmuxRunningFunc is the function used to ensure tmux is running. Can be changed for testing.
var ensureTmuxRunningFunc = func() bool {
	return core.EnsureTmuxRunning()
}

// getNotificationLineFunc is the function used to get notification line by ID.
// Uses optimized retrieval to improve performance with large datasets.
var getNotificationLineFunc = func(id string) (string, error) {
	// Use the optimized function that directly retrieves by ID
	return storage.GetNotificationByID(id)
}

// validatePaneExistsFunc is the function used to validate pane exists.
var validatePaneExistsFunc = func(session, window, pane string) bool {
	return core.ValidatePaneExists(session, window, pane)
}

// jumpToPaneFunc is the function used to jump to pane.
var jumpToPaneFunc = func(session, window, pane string) bool {
	return core.JumpToPane(session, window, pane)
}

// JumpResult holds the result of a jump operation.
type JumpResult struct {
	ID         string
	Session    string
	Window     string
	Pane       string
	State      string
	Message    string
	PaneExists bool
}

// Jump jumps to the pane of the notification with the given ID.
// Returns jump result and error.
func Jump(id string) (*JumpResult, error) {
	if !ensureTmuxRunningFunc() {
		return nil, fmt.Errorf("tmux is not running")
	}

	line, err := getNotificationLineFunc(id)
	if err != nil {
		return nil, err
	}

	fields := strings.Split(line, "\t")
	// Ensure at least 7 fields (up to pane)
	if len(fields) <= fieldPane {
		return nil, fmt.Errorf("invalid notification line format")
	}
	state := fields[fieldState]
	session := fields[fieldSession]
	window := fields[fieldWindow]
	pane := fields[fieldPane]
	message := fields[fieldMessage]

	if session == "" || window == "" || pane == "" {
		return nil, fmt.Errorf("notification %s has no pane association", id)
	}

	// Validate pane exists
	paneExists := validatePaneExistsFunc(session, window, pane)

	// Jump to pane
	if !jumpToPaneFunc(session, window, pane) {
		// JumpToPane returns true even if pane doesn't exist (window selected).
		// So we treat false as failure to select window.
		return nil, fmt.Errorf("failed to jump to pane (maybe window no longer exists)")
	}

	return &JumpResult{
		ID:         id,
		Session:    session,
		Window:     window,
		Pane:       pane,
		State:      state,
		Message:    message,
		PaneExists: paneExists,
	}, nil
}

func init() {
	cmd.RootCmd.AddCommand(jumpCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// jumpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will run when this command
	// is called directly, e.g.:
	// jumpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runJump(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		colors.Error("'jump' requires a notification ID")
		fmt.Fprintf(os.Stderr, "Usage: tmux-intray jump <id>\n")
		return
	}
	id := args[0]

	// Ensure tmux is running (mirror bash script behavior)
	if !core.EnsureTmuxRunning() {
		colors.Error("tmux is not running")
		return
	}

	// Jump to pane
	result, err := Jump(id)
	if err != nil {
		colors.Error(err.Error())
		return
	}

	// Display result
	if result.State == "dismissed" {
		colors.Info(fmt.Sprintf("Notification %s is dismissed, but jumping anyway", id))
	}
	colors.Success(fmt.Sprintf("Jumped to pane %s:%s.%s", result.Session, result.Window, result.Pane))
}
