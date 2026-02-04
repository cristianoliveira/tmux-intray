package main

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

func main() {
	stateDir := "/Users/cristianoliveira/other/tmux-intray/.gwt/andy-dev/test_state"
	if len(os.Args) > 1 {
		stateDir = os.Args[1]
	}
	os.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	os.Setenv("TMUX_INTRAY_DEBUG", "true")
	storage.Init()

	// Add some notifications
	id1 := storage.AddNotification("Hello world", "", "session1", "window0", "pane0", "", "info")
	fmt.Printf("Added ID: %s\n", id1)
	id2 := storage.AddNotification("Warning: disk full", "", "session1", "window0", "pane1", "", "warning")
	fmt.Printf("Added ID: %s\n", id2)
	id3 := storage.AddNotification("Error: cannot connect", "", "session2", "window1", "pane2", "", "error")
	fmt.Printf("Added ID: %s\n", id3)

	// List them
	lines := storage.ListNotifications("active", "", "", "", "", "", "")
	fmt.Printf("Lines:\n%s\n", lines)
}
