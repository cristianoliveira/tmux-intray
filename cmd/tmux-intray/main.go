package main

import (
	"os"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

func main() {
	colors.StructuredInfo("startup", "main", "started", nil, "", nil)
	if err := cmd.Execute(); err != nil {
		colors.StructuredError("startup", "main", "failed", err, "", nil)
		os.Exit(1)
	}
	colors.StructuredInfo("startup", "main", "completed", nil, "", nil)
}
