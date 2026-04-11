package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HookExecutionResult contains detailed result of a single hook execution.
type HookExecutionResult struct {
	ExitCode int
	Output   string
	Duration time.Duration
	Error    error
}

// runSyncHook executes a hook script synchronously.
func runSyncHook(scriptPath, scriptName string, envMap map[string]string, failureMode string) error {
	result := runSyncHookWithResult(scriptPath, scriptName, envMap, failureMode)
	return result.Error
}

// runSyncHookWithResult executes a hook script synchronously and returns detailed result.
func runSyncHookWithResult(scriptPath, scriptName string, envMap map[string]string, failureMode string) HookExecutionResult {
	start := time.Now()
	cmd := exec.Command(scriptPath)
	cmd.Env = os.Environ()
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Print hook output to stderr (so it appears in logs)
	if len(output) > 0 {
		_, _ = os.Stderr.Write(output)
	}

	result := HookExecutionResult{
		ExitCode: exitCode,
		Output:   string(output),
		Duration: duration,
	}

	if err != nil {
		switch failureMode {
		case "abort":
			result.Error = fmt.Errorf("hooks.Run: hook '%s' failed after %.2fs: %v, output: %s", scriptName, duration.Seconds(), err, output)
		case "warn":
			if isHooksVerbose() {
				fmt.Fprintf(os.Stderr, "warning: hook %s failed after %.2fs: %v, output: %s\n", scriptName, duration.Seconds(), err, output)
			}
			result.Error = nil
		case "ignore":
			result.Error = nil
		}
	} else if isHooksVerbose() {
		fmt.Fprintf(os.Stderr, "  Hook completed in %.2fs\n", duration.Seconds())
	}
	return result
}

// runAsyncHook executes a hook script asynchronously with timeout.
func runAsyncHook(scriptPath, scriptName string, envMap map[string]string, failureMode string) {
	timeout := getAsyncTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Env = os.Environ()
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		cancel()
		if failureMode != "ignore" && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: async hook %s failed to start: %v\n", scriptName, err)
		}
		pendingHooksMu.Lock()
		pendingHookCount--
		pendingHooksMu.Unlock()
		pendingHooks.Done()
		return
	}
	startTime := time.Now()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if isHooksVerbose() {
					fmt.Fprintf(os.Stderr, "error: async hook %s panicked: %v\n", scriptName, r)
				}
			}
			pendingHooksMu.Lock()
			pendingHookCount--
			pendingHooksMu.Unlock()
			pendingHooks.Done()
			cancel()
		}()

		err := cmd.Wait()
		duration := time.Since(startTime)

		if ctx.Err() == context.DeadlineExceeded && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: async hook %s timed out after %.2fs\n", scriptName, duration.Seconds())
		}

		if err != nil && failureMode != "ignore" && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: async hook %s failed: %v (duration: %.2fs)\n", scriptName, err, duration.Seconds())
		} else if err == nil && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "  async hook %s completed in %.2fs\n", scriptName, duration.Seconds())
		}
	}()
}

// parseModifications extracts environment variable modifications from hook output.
func parseModifications(output string) map[string]string {
	modifications := make(map[string]string)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "export ") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.TrimPrefix(parts[0], "export "))
		value := parts[1]

		if !isValidEnvVarName(key) {
			continue
		}

		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		} else if strings.Contains(value, " ") {
			continue
		}

		modifications[key] = value
	}

	return modifications
}

// isValidEnvVarName checks if a string is a valid environment variable name.
func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		if i == 0 {
			if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && c != '_' {
				return false
			}
		} else if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
