// Package storage provides storage backend selection and implementations.
package storage

import (
	"fmt"
	"os"
	"time"
)

const (
	lockTimeout = 10 * time.Second
	lockRetry   = 100 * time.Millisecond
)

// Lock represents a directory-based lock.
type Lock struct {
	dir string
}

// NewLock creates a new lock at the given directory path.
func NewLock(dir string) *Lock {
	return &Lock{dir: dir}
}

// Acquire attempts to acquire the lock, retrying until timeout.
func (l *Lock) Acquire() error {
	start := time.Now()
	for time.Since(start) <= lockTimeout {
		// Use MkdirAll to ensure parent directories exist
		err := os.MkdirAll(l.dir, FileModeDir)
		if err == nil {
			return nil
		}
		// If directory already exists, that's fine
		if os.IsExist(err) {
			return nil
		}
		if time.Since(start) > lockTimeout {
			return fmt.Errorf("failed to acquire lock for %s after timeout: %w", l.dir, err)
		}
		// Retry after a short delay
		time.Sleep(lockRetry)
	}
	// Timeout reached - this check ensures we timeout even if operation hangs without error
	elapsed := time.Since(start)
	return fmt.Errorf("failed to acquire lock after %v (timeout: %v)", elapsed, lockTimeout)
}

// Release releases the lock by removing the directory.
func (l *Lock) Release() error {
	return os.Remove(l.dir)
}

// WithLock executes fn while holding the lock.
func WithLock(dir string, fn func() error) error {
	lock := NewLock(dir)
	if err := lock.Acquire(); err != nil {
		return fmt.Errorf("with lock: failed to acquire lock for %s: %w", dir, err)
	}
	defer lock.Release()
	return fn()
}
