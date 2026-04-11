package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Run executes hooks for a hook point with environment variables.
func Run(hookPoint string, envVars ...string) error {
	hookDir := filepath.Join(getHooksDir(), hookPoint)
	files, err := os.ReadDir(hookDir)
	if err != nil {
		return nil
	}

	envMap := buildHookEnv(hookPoint, envVars)
	scripts := collectHookScripts(hookDir, files)
	if len(scripts) == 0 {
		return nil
	}

	if isHooksVerbose() {
		fmt.Fprintf(os.Stderr, "Running %s hooks (%d script(s))\n", hookPoint, len(scripts))
	}

	failureMode := getFailureMode()
	return executeHooks(scripts, envMap, failureMode, getAsyncEnabled(), getMaxAsyncHooks())
}

// RunWithModification executes hooks for a hook point and returns modification results.
func RunWithModification(hookPoint string, envVars ...string) (HookResult, error) {
	if getAsyncEnabled() {
		hookDir := filepath.Join(getHooksDir(), hookPoint)
		files, err := os.ReadDir(hookDir)
		if err != nil {
			return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
		}

		envMap := buildHookEnv(hookPoint, envVars)
		scripts := collectHookScripts(hookDir, files)
		if len(scripts) == 0 {
			return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
		}

		failureMode := getFailureMode()
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "Running %s hooks (async, %d script(s))\n", hookPoint, len(scripts))
		}

		_ = executeHooks(scripts, envMap, failureMode, true, getMaxAsyncHooks())
		return HookResult{ExitCode: 0, Modified: false, EnvVars: make(map[string]string)}, nil
	}

	hookDir := filepath.Join(getHooksDir(), hookPoint)
	files, err := os.ReadDir(hookDir)
	if err != nil {
		return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
	}

	envMap := buildHookEnv(hookPoint, envVars)
	scripts := collectHookScripts(hookDir, files)
	if len(scripts) == 0 {
		return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
	}

	if isHooksVerbose() {
		fmt.Fprintf(os.Stderr, "Running %s hooks (%d script(s))\n", hookPoint, len(scripts))
	}

	failureMode := getFailureMode()
	result := HookResult{EnvVars: make(map[string]string), ExitCode: 0, Modified: false}

	for _, script := range scripts {
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "  Executing hook: %s\n", script.name)
		}

		execResult := runSyncHookWithResult(script.path, script.name, envMap, failureMode)
		result.ExitCode = execResult.ExitCode

		if execResult.ExitCode == ExitCodeModify || execResult.ExitCode == ExitCodeRoute || execResult.ExitCode == ExitCodeDefer {
			mods := parseModifications(execResult.Output)
			for k, v := range mods {
				result.EnvVars[k] = v
			}
			result.Modified = true
		}

		if execResult.ExitCode == ExitCodeReject {
			result.Rejected = true
			return result, fmt.Errorf("hook '%s' rejected notification", script.name)
		}

		if execResult.Error != nil && failureMode == "abort" {
			return result, execResult.Error
		}
	}

	return result, nil
}

func buildHookEnv(hookPoint string, envVars []string) map[string]string {
	envMap := map[string]string{
		"HOOK_POINT":                     hookPoint,
		"TMUX_INTRAY_HOOKS_FAILURE_MODE": getFailureMode(),
		"HOOK_TIMESTAMP":                 time.Now().Format(time.RFC3339),
	}

	if tmuxIntrayPath := resolveTmuxIntrayPath(); tmuxIntrayPath != "" {
		envMap["TMUX_INTRAY_BINARY"] = tmuxIntrayPath
	}

	for _, v := range envVars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	return envMap
}

func resolveTmuxIntrayPath() string {
	var tmuxIntrayPath string
	if exe, err := os.Executable(); err == nil {
		tmuxIntrayPath = exe
	}

	if len(os.Args) > 0 && os.Args[0] != "" {
		if filepath.IsAbs(os.Args[0]) {
			tmuxIntrayPath = os.Args[0]
		} else if path, err := exec.LookPath(os.Args[0]); err == nil {
			tmuxIntrayPath = path
		}
	}

	if tmuxIntrayPath != "" {
		return tmuxIntrayPath
	}

	home, _ := os.UserHomeDir()
	commonPaths := []string{
		filepath.Join(home, ".local", "bin", "tmux-intray"),
		"/usr/local/bin/tmux-intray",
		"/usr/bin/tmux-intray",
	}
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func collectHookScripts(hookDir string, files []os.DirEntry) []hookScript {
	scripts := []hookScript{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		scriptPath := filepath.Join(hookDir, f.Name())
		info, err := os.Stat(scriptPath)
		if err != nil || info.Mode()&0111 == 0 {
			continue
		}
		scripts = append(scripts, hookScript{path: scriptPath, name: f.Name()})
	}

	sort.Slice(scripts, func(i, j int) bool { return scripts[i].name < scripts[j].name })
	return scripts
}

func executeHooks(scripts []hookScript, envMap map[string]string, failureMode string, asyncEnabled bool, maxAsync int) error {
	for _, script := range scripts {
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "  Executing hook: %s\n", script.name)
		}
		if asyncEnabled {
			tryStartAsyncHook(script, envMap, failureMode, maxAsync)
			continue
		}
		if err := runSyncHookAndCheckAbort(script, envMap, failureMode); err != nil {
			return err
		}
	}
	return nil
}

func runSyncHookAndCheckAbort(script hookScript, envMap map[string]string, failureMode string) error {
	if err := runSyncHook(script.path, script.name, envMap, failureMode); err != nil && failureMode == "abort" {
		return err
	}
	return nil
}

func tryStartAsyncHook(script hookScript, envMap map[string]string, failureMode string, maxAsync int) bool {
	pendingHooksMu.Lock()
	defer pendingHooksMu.Unlock()

	if pendingHookCount >= maxAsync {
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: Too many async hooks pending (max: %d), skipping %s\n", maxAsync, script.name)
		}
		return false
	}
	pendingHookCount++
	pendingHooks.Add(1)
	if isHooksVerbose() {
		fmt.Fprintf(os.Stderr, "  Starting hook asynchronously: %s\n", script.name)
	}
	go runAsyncHook(script.path, script.name, envMap, failureMode)
	return true
}

// ResetForTesting resets internal state for testing.
func ResetForTesting() {
	pendingHooksMu.Lock()
	defer pendingHooksMu.Unlock()
	if pendingHookCount > 0 {
		panic(fmt.Sprintf("hooks.ResetForTesting: called with %d pending hooks. Call hooks.WaitForPendingHooks() first.", pendingHookCount))
	}
	pendingHookCount = 0
	pendingHooks = sync.WaitGroup{}
}

// WaitForPendingHooks waits for all pending async hooks to complete.
func WaitForPendingHooks() {
	pendingHooks.Wait()
}

// Shutdown gracefully shuts down the hooks subsystem.
func Shutdown() {
	WaitForPendingHooks()
}
