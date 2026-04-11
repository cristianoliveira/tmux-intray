package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func buildHookEnv(hookPoint string, envVars []string) map[string]string {
	envMap := make(map[string]string)
	envMap["HOOK_POINT"] = hookPoint
	envMap["TMUX_INTRAY_HOOKS_FAILURE_MODE"] = getFailureMode()
	envMap["HOOK_TIMESTAMP"] = time.Now().Format(time.RFC3339)

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

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].name < scripts[j].name
	})

	return scripts
}

// hookScript holds information about a hook script.
type hookScript struct {
	path string
	name string
}
