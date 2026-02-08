/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/spf13/cobra"
)

// markReadCmd represents the mark-read command
var markReadCmd = &cobra.Command{
	Use:   "mark-read <id>",
	Short: "Mark a notification as read",
	Long: `Mark a notification as read by ID.

USAGE:
    tmux-intray mark-read <id>

OPTIONS:
    -h, --help           Show this help`,
	Args: cobra.ExactArgs(1),
	Run:  runMarkRead,
}

var markReadFunc = func(id string) error {
	return fileStorage.MarkNotificationRead(id)
}

func init() {
	cmd.RootCmd.AddCommand(markReadCmd)
}

func runMarkRead(cmd *cobra.Command, args []string) {
	id := args[0]

	err := markReadFunc(id)
	if err != nil {
		colors.Error(err.Error())
		return
	}

	colors.Success(fmt.Sprintf("Notification %s marked as read", id))
}
