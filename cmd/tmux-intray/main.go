package main

import (
	"os"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

func main() {
	exitCode := run(os.Args[1:], cmd.Execute)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func run(args []string, execute func() error) int {
	// Disable structured logging for TUI command to avoid JSON output interfering with display
	isTUICommand := len(args) > 0 && args[0] == "tui"

	if !isTUICommand {
		colors.StructuredDebug("startup", "main", "started", nil, "", nil)
	}

	if err := execute(); err != nil {
		if !isTUICommand {
			colors.StructuredDebug("startup", "main", "failed", err, "", nil)
		}
		return 1
	}

	if !isTUICommand {
		colors.StructuredDebug("startup", "main", "completed", nil, "", nil)
	}

	return 0
}
