/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new item to the tray",
	Long: `Add a new item to the tray.

USAGE:
    tmux-intray add [OPTIONS] <message>

OPTIONS:
    --session <id>          Associate with specific session ID
    --window <id>           Associate with specific window ID
    --pane <id>             Associate with specific pane ID
    --pane-created <time>   Pane creation timestamp (seconds since epoch)
    --no-associate          Do not associate with any pane
    --level <level>         Notification level: info, warning, error, critical (default: info)
    -h, --help              Show this help

If no pane association options are provided, automatically associates with
the current tmux pane (if inside tmux). Use --no-associate to skip.`,
	Run: runAdd,
}

var (
	addSession     string
	addWindow      string
	addPane        string
	addPaneCreated string
	addNoAssociate bool
	addLevel       string
)

var addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
	return core.AddTrayItem(item, session, window, pane, paneCreated, noAuto, level)
}

type AddOptions struct {
	Message     string
	Session     string
	Window      string
	Pane        string
	PaneCreated string
	NoAuto      bool
	Level       string
}

func Add(opts AddOptions) string {
	return addFunc(opts.Message, opts.Session, opts.Window, opts.Pane, opts.PaneCreated, opts.NoAuto, opts.Level)
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Local flags
	addCmd.Flags().StringVar(&addSession, "session", "", "Associate with specific session ID")
	addCmd.Flags().StringVar(&addWindow, "window", "", "Associate with specific window ID")
	addCmd.Flags().StringVar(&addPane, "pane", "", "Associate with specific pane ID")
	addCmd.Flags().StringVar(&addPaneCreated, "pane-created", "", "Pane creation timestamp (seconds since epoch)")
	addCmd.Flags().BoolVar(&addNoAssociate, "no-associate", false, "Do not associate with any pane")
	addCmd.Flags().StringVar(&addLevel, "level", "info", "Notification level: info, warning, error, critical")
}

func runAdd(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		colors.Error("'add' requires a message")
		cmd.PrintErrln("Usage: tmux-intray add [OPTIONS] <message>")
		return
	}
	message := ""
	// Join all remaining arguments as message (preserving spaces)
	for i, arg := range args {
		if i > 0 {
			message += " "
		}
		message += arg
	}

	opts := AddOptions{
		Message:     message,
		Session:     addSession,
		Window:      addWindow,
		Pane:        addPane,
		PaneCreated: addPaneCreated,
		NoAuto:      addNoAssociate,
		Level:       addLevel,
	}
	id := Add(opts)
	if id == "" {
		colors.Error("Failed to add notification")
		return
	}
	colors.Success("Item added to tray (ID: " + id + ")")
}
