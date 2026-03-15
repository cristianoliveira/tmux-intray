/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/telemetry"
)

// logCLICommand logs a CLI command invocation with sanitized arguments.
func logCLICommand(command string, args []string) {
	sanitizedArgs := sanitizeArgs(args)
	telemetry.LogCLICommand(command, sanitizedArgs, nil)
}

// logCLICommandWithContext logs a CLI command with additional context.
func logCLICommandWithContext(command string, args []string, context map[string]interface{}) {
	sanitizedArgs := sanitizeArgs(args)
	telemetry.LogCLICommand(command, sanitizedArgs, context)
}

// sanitizeArgs removes or redacts sensitive argument values.
func sanitizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	safeFlags := map[string]bool{
		"--level":        true,
		"--format":       true,
		"--state":        true,
		"--status":       true,
		"--all":          true,
		"--dismissed":    true,
		"--active":       true,
		"--dryrun":       true,
		"--dry-run":      true,
		"--days":         true,
		"--interval":     true,
		"--no-associate": true,
		"--no-mark-read": true,
		"--group-by":     true,
		"--group-count":  true,
		"--filter":       true,
		"--regex":        true,
		"--older-than":   true,
		"--newer-than":   true,
		"--search":       true,
		"--pane":         true,
		"--session":      true,
		"--window":       true,
		"--help":         true,
		"-h":             true,
	}

	redactFlags := map[string]bool{
		"--message":      true,
		"--pane-created": true,
	}

	sanitized := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "--") || (strings.HasPrefix(arg, "-") && !isNumeric(arg)) {
			sanitized = append(sanitized, arg)

			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagName := arg
				value := args[i+1]

				if redactFlags[flagName] {
					sanitized = append(sanitized, "[value]")
				} else if safeFlags[flagName] {
					sanitized = append(sanitized, value)
				} else {
					sanitized = append(sanitized, "[value]")
				}
				i++
			}
		} else {
			if arg == "" {
				sanitized = append(sanitized, "[empty]")
			} else if len(arg) > 100 {
				sanitized = append(sanitized, "[long_text]")
			} else if strings.TrimSpace(arg) == "" {
				sanitized = append(sanitized, "[whitespace]")
			} else {
				sanitized = append(sanitized, "[message]")
			}
		}
	}

	return sanitized
}

// isNumeric checks if a string starts with a digit.
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}
