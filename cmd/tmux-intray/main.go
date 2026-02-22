package main

import (
	"os"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

func main() {
	// Disable structured logging for TUI command to avoid JSON output interfering with display
	args := os.Args[1:]
	isTUICommand := len(args) > 0 && args[0] == "tui"

	if !isTUICommand {
		colors.StructuredInfo("startup", "main", "started", nil, "", nil)
	}

	if err := cmd.Execute(); err != nil {
		if !isTUICommand {
			colors.StructuredError("startup", "main", "failed", err, "", nil)
		}
		os.Exit(1)
	}

	if !isTUICommand {
		colors.StructuredInfo("startup", "main", "completed", nil, "", nil)
	}
}
