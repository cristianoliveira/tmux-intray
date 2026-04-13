package hooks

import (
	"os"
	"sync"
	"time"
)

// HookResult contains the result of running hooks with modification support.
type HookResult struct {
	Modified bool
	ExitCode int
	EnvVars  map[string]string
	Rejected bool
}

// Exit codes for hook modification semantics.
const (
	ExitCodeAccept = 0
	ExitCodeReject = 1
	ExitCodeModify = 2
	ExitCodeRoute  = 3
	ExitCodeDefer  = 4
)

// File permission constants.
const (
	FileModeDir    os.FileMode = 0755
	FileModeScript os.FileMode = 0755
)

// HookExecutionResult contains detailed result of a single hook execution.
type HookExecutionResult struct {
	ExitCode int
	Output   string
	Duration time.Duration
	Error    error
}

type hookScript struct {
	path string
	name string
}

var (
	pendingHooks     sync.WaitGroup
	pendingHooksMu   sync.Mutex
	pendingHookCount int
)
