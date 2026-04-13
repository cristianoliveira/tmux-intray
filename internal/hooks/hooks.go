// Package hooks provides a hook subsystem for extensibility.
package hooks

import (
	"fmt"
	"os"
	"sync"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
)

var (
	manager *hookManager
	once    sync.Once
)

type hookManager struct {
	mu          sync.Mutex
	shutdown    chan struct{}
	initialized bool
}

func getManager() *hookManager {
	once.Do(func() {
		manager = &hookManager{
			shutdown: make(chan struct{}),
		}
	})
	return manager
}

// Init initializes the hooks subsystem.
func Init() error {
	config.Load()
	m := getManager()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.initialized {
		return nil
	}

	dir := getHooksDir()
	if err := os.MkdirAll(dir, FileModeDir); err != nil {
		colors.Error(fmt.Sprintf("hooks.Init: failed to create hooks directory %s: %v", dir, err))
		return fmt.Errorf("hooks.Init: failed to create hooks directory %s: %w", dir, err)
	}
	m.initialized = true
	return nil
}
