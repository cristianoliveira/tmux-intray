package main

import (
	"os"
	"strings"
)

// allowTmuxlessMode returns true when tmux-related commands should skip
// hard failures (e.g. running inside CI or when explicitly requested).
func allowTmuxlessMode() bool {
	if envBool("TMUX_INTRAY_ALLOW_NO_TMUX") {
		return true
	}

	// CI providers usually set CI=true
	if os.Getenv("CI") != "" {
		return true
	}

	// Bats integration tests expose this variable
	if os.Getenv("BATS_TMPDIR") != "" {
		return true
	}

	// Test harness exports TMUX_AVAILABLE so honor it as well
	if strings.TrimSpace(os.Getenv("TMUX_AVAILABLE")) == "0" {
		return true
	}

	return false
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
