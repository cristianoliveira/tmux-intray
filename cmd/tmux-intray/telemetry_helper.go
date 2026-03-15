/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/telemetry"
)

// logCLICommand logs a CLI command invocation with sanitized arguments.
// This function:
// - Logs the command name and sanitized arguments
// - Sanitizes sensitive flag values (e.g., --level, --format)
// - Adds context data like success status (when available)
//
// Example usage in a command:
//
//	logCLICommand("add", []string{"--level", "error", "test message"})
func logCLICommand(command string, args []string) {
	sanitizedArgs := sanitizeArgs(args)
	telemetry.LogCLICommand(command, sanitizedArgs, nil)
}

// logCLICommandWithContext logs a CLI command with additional context.
// This is useful for tracking command outcomes, execution metrics, etc.
//
// Example usage:
//
//	logCLICommandWithContext("add", args, map[string]interface{}{
//	  "success": true,
//	  "duration_ms": 125,
//	})
func logCLICommandWithContext(command string, args []string, context map[string]interface{}) {
	sanitizedArgs := sanitizeArgs(args)
	telemetry.LogCLICommand(command, sanitizedArgs, context)
}

// sanitizeArgs removes or redacts sensitive argument values.
// This prevents logging user data, credentials, or sensitive configuration.
//
// Strategy:
// - For flags with sensitive values (--message, etc.), keep the flag but omit value
// - For positional arguments that might contain user content, use a placeholder
// - For safe flags (--level, --format, etc.), keep both flag and value
//
// Examples:
// - ["--level", "error", "test message"] -> ["--level", "error", "[message]"]
// - ["--session", "work", "--window", "1"] -> ["--session", "work", "--window", "1"]
func sanitizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	// Flags that are safe to log completely (no sensitive values)
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

	// Flags whose values should be redacted (usually contain user data)
	redactFlags := map[string]bool{
		"--message":      true,
		"--pane-created": true,
	}

	sanitized := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if it's a flag
		if strings.HasPrefix(arg, "--") || (strings.HasPrefix(arg, "-") && !isNumeric(arg)) {
			// This is a flag
			sanitized = append(sanitized, arg)

			// Check if this flag has a value following it
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagName := arg
				value := args[i+1]

				if redactFlags[flagName] {
					// Redact the value
					sanitized = append(sanitized, "[value]")
				} else if safeFlags[flagName] {
					// Keep the value
					sanitized = append(sanitized, value)
				} else {
					// Unknown flag - play it safe and redact
					sanitized = append(sanitized, "[value]")
				}
				i++ // Skip the next argument since we processed it as a value
			}
		} else {
			// This is a positional argument (not a flag)
			// For positional arguments, use a placeholder as they often contain user data
			if arg == "" {
				sanitized = append(sanitized, "[empty]")
			} else if len(arg) > 100 {
				sanitized = append(sanitized, "[long_text]")
			} else if strings.TrimSpace(arg) == "" {
				sanitized = append(sanitized, "[whitespace]")
			} else {
				// For positional args, use a generic placeholder
				sanitized = append(sanitized, "[message]")
			}
		}
	}

	return sanitized
}

// isNumeric checks if a string starts with a digit (used to distinguish negative numbers from flags)
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}
