// Package version implements the version command.
package version

import "fmt"

// Version is the version of tmux-intray.
const Version = "0.1.0"

// Run executes the version command.
func Run(args []string) error {
	fmt.Printf("tmux-intray v%s\n", Version)
	return nil
}
