package main

import (
	"os"

	"github.com/cristianoliveira/tmux-intray/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
