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

// ensureTmuxRunningFunc is the function used to ensure tmux is running. Can be changed for testing.
var ensureTmuxRunningFunc = func() bool {
	return core.EnsureTmuxRunning()
}

// getNotificationLineFunc is the function used to get notification line by ID.
// Uses optimized retrieval to improve performance with large datasets.
var getNotificationLineFunc = func(id string) (string, error) {
	// Use the optimized function that directly retrieves by ID
	return storageStore.GetNotificationByID(id)
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
		return nil, fmt.Errorf("tmux not running")
	}

	line, err := getNotificationLineFunc(id)
	if err != nil {
		return nil, err
	}

	fields := strings.Split(line, "\t")
	// Ensure at least 7 fields (up to pane)
	if len(fields) <= storage.FieldPane {
		return nil, fmt.Errorf("jump: invalid notification line format")
	}
	state := fields[storage.FieldState]
	session := fields[storage.FieldSession]
	window := fields[storage.FieldWindow]
	pane := fields[storage.FieldPane]
	message := fields[storage.FieldMessage]

	if session == "" || window == "" || pane == "" {
		// Build a detailed error message showing which fields are missing
		var missingFields []string
		if session == "" {
			missingFields = append(missingFields, "session")
		}
		if window == "" {
			missingFields = append(missingFields, "window")
		}
		if pane == "" {
			missingFields = append(missingFields, "pane")
		}

		return nil, fmt.Errorf(
			"jump: notification %s missing required fields:\n"+
				"  missing: %s\n"+
				"  required fields: session, window, pane\n"+
				"  hint: notifications must be created from within an active tmux session for jump to work",
			id, strings.Join(missingFields, ", "))
	}

	// Validate pane exists
	paneExists := validatePaneExistsFunc(session, window, pane)

	// Jump to pane - returns false only if it completely fails (window selection failed)
	// Returns true even if pane doesn't exist (falls back to window)
	if !jumpToPaneFunc(session, window, pane) {
		return nil, fmt.Errorf("jump: failed to jump because pane or window does not exist")
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
		colors.Error("jump: requires a notification id")
		fmt.Fprintf(os.Stderr, "Usage: tmux-intray jump <id>\n")
		return
	}
	id := args[0]

	// Ensure tmux is running (mirror bash script behavior)
	if !core.EnsureTmuxRunning() {
		colors.Error("tmux not running")
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

	// Display appropriate message based on whether pane selection succeeded
	if result.PaneExists {
		// Pane exists and was selected
		colors.Success(fmt.Sprintf("Jumped to session %s, window %s, pane %s", result.Session, result.Window, result.Pane))
	} else {
		// Pane doesn't exist, fell back to window selection
		colors.Warning(fmt.Sprintf("Pane %s no longer exists (jumped to window %s:%s instead)", result.Pane, result.Session, result.Window))
	}
}
