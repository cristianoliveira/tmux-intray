package main

import (
	"github.com/cristianoliveira/tmux-intray/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
