/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/spf13/cobra"
)

type toggleClient interface {
	EnsureTmuxRunning() bool
	GetVisibility() (string, error)
	SetVisibility(visible bool) error
	RunHook(name string, envVars ...string) error
}

// NewToggleCmd creates the toggle command with explicit dependencies.
func NewToggleCmd(client toggleClient) *cobra.Command {
	if client == nil {
		panic("NewToggleCmd: client dependency cannot be nil")
	}

	return &cobra.Command{
		Use:   "toggle",
		Short: "Toggle tray visibility",
		Long: `Toggle the visibility of the tmux-intray tray.

This command shows or hides the tray by setting the global tmux environment
variable TMUX_INTRAY_VISIBLE to "1" (visible) or "0" (hidden).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure tmux is running
			if !client.EnsureTmuxRunning() {
				if allowTmuxlessMode() {
					colors.Warning("tmux not running; skipping toggle")
					return nil
				}
				return fmt.Errorf("tmux not running")
			}

			// Get current visibility
			oldVisible, err := client.GetVisibility()
			if err != nil {
				colors.Error(fmt.Sprintf("Failed to get current visibility: %v", err))
				return err
			}

			var newVisible bool
			var msg string
			if oldVisible == "1" {
				newVisible = false
				msg = "Tray hidden"
			} else {
				newVisible = true
				msg = "Tray visible"
			}
			newVisibleStr := "0"
			if newVisible {
				newVisibleStr = "1"
			}

			// Run pre-toggle hooks
			envVars := []string{
				"OLD_VISIBLE=" + oldVisible,
				"VISIBLE=" + newVisibleStr,
			}
			if err := client.RunHook("pre-toggle", envVars...); err != nil {
				colors.Error(err.Error())
				return err
			}

			// Set new visibility
			if err := client.SetVisibility(newVisible); err != nil {
				colors.Error(err.Error())
				return err
			}

			// Run post-toggle hooks
			if err := client.RunHook("post-toggle", envVars...); err != nil {
				colors.Error(err.Error())
				return err
			}

			colors.Info(msg)
			return nil
		},
	}
}

// toggleGetVisibilityFunc is the function used to get visibility. Can be changed for testing.
var toggleGetVisibilityFunc = func() (string, error) {
	return core.GetVisibility()
}

// toggleSetVisibilityFunc is the function used to set visibility. Can be changed for testing.
var toggleSetVisibilityFunc = func(visible bool) error {
	return core.SetVisibility(visible)
}

// toggleEnsureTmuxRunningFunc is the function used to check tmux. Can be changed for testing.
var toggleEnsureTmuxRunningFunc = func() bool {
	return core.EnsureTmuxRunning()
}

// toggleRunHookFunc is the function used to run hooks. Can be changed for testing.
var toggleRunHookFunc = func(name string, envVars ...string) error {
	return hooks.Run(name, envVars...)
}

// GetCurrentVisibility returns the current visibility as a boolean.
// Returns true if tray is visible, false if hidden.
func GetCurrentVisibility() bool {
	visible, _ := toggleGetVisibilityFunc()
	return visible == "1"
}

// Toggle toggles tray visibility and returns the new visibility state.
// Returns true if tray is now visible, false if hidden, and any error.
func Toggle() (bool, error) {
	visible, err := toggleGetVisibilityFunc()
	if err != nil {
		return false, err
	}
	newVisible := visible != "1"
	err = toggleSetVisibilityFunc(newVisible)
	return newVisible, err
}

// defaultToggleClient is the default implementation using core package.
type defaultToggleClient struct{}

func (d *defaultToggleClient) EnsureTmuxRunning() bool {
	return toggleEnsureTmuxRunningFunc()
}

func (d *defaultToggleClient) GetVisibility() (string, error) {
	return toggleGetVisibilityFunc()
}

func (d *defaultToggleClient) SetVisibility(visible bool) error {
	return toggleSetVisibilityFunc(visible)
}

func (d *defaultToggleClient) RunHook(name string, envVars ...string) error {
	return toggleRunHookFunc(name, envVars...)
}

// toggleCmd represents the toggle command
var toggleCmd = NewToggleCmd(&defaultToggleClient{})

func init() {
	cmd.RootCmd.AddCommand(toggleCmd)
}
