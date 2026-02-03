// Package hooks provides a hook subsystem for extensibility.
package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Init initializes the hooks subsystem.
func Init() {
	// Ensure hooks directory exists
	dir := getHooksDir()
	os.MkdirAll(dir, 0755)
}

// getHooksDir returns the hooks directory path.
func getHooksDir() string {
	if dir := os.Getenv("TMUX_INTRAY_HOOKS_DIR"); dir != "" {
		return dir
	}
	// Default: $XDG_CONFIG_HOME/tmux-intray/hooks
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "tmux-intray", "hooks")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tmux-intray", "hooks")
}

// getFailureMode returns the failure mode (abort, warn, ignore).
func getFailureMode() string {
	if mode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE"); mode != "" {
		return mode
	}
	return "warn"
}

// Run executes hooks for a hook point with environment variables.
func Run(hookPoint string, envVars ...string) error {
	// Check if hooks are enabled globally
	if enabled := os.Getenv("TMUX_INTRAY_HOOKS_ENABLED"); enabled == "0" {
		return nil
	}
	// Hook point specific enable variable
	hookPointVar := fmt.Sprintf("TMUX_INTRAY_HOOKS_ENABLED_%s", strings.ReplaceAll(strings.ToUpper(hookPoint), "-", "_"))
	if enabled := os.Getenv(hookPointVar); enabled == "0" {
		return nil
	}

	hookDir := filepath.Join(getHooksDir(), hookPoint)
	files, err := os.ReadDir(hookDir)
	if err != nil {
		// Directory doesn't exist -> no hooks
		return nil
	}

	// Build environment map
	envMap := make(map[string]string)
	envMap["HOOK_POINT"] = hookPoint
	for _, v := range envVars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	failureMode := getFailureMode()
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".sh") {
			continue
		}
		scriptPath := filepath.Join(hookDir, f.Name())
		info, err := os.Stat(scriptPath)
		if err != nil || info.Mode()&0111 == 0 {
			// Not executable
			continue
		}
		cmd := exec.Command(scriptPath)
		cmd.Env = os.Environ()
		for k, v := range envMap {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			switch failureMode {
			case "abort":
				return fmt.Errorf("hook %s failed: %v, output: %s", f.Name(), err, output)
			case "warn":
				fmt.Fprintf(os.Stderr, "warning: hook %s failed: %v, output: %s\n", f.Name(), err, output)
			case "ignore":
				// do nothing
			}
		}
	}
	return nil
}
