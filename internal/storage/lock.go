// Package storage provides file-based TSV storage with locking.
package storage

import (
	"errors"
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
	for {
		err := os.Mkdir(l.dir, 0755)
		if err == nil {
			return nil
		}
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create lock directory: %w", err)
		}
		// Lock exists, wait and retry
		if time.Since(start) > lockTimeout {
			return errors.New("timeout acquiring lock")
		}
		time.Sleep(lockRetry)
	}
}

// Release releases the lock by removing the directory.
func (l *Lock) Release() error {
	return os.Remove(l.dir)
}

// WithLock executes fn while holding the lock.
func WithLock(dir string, fn func() error) error {
	lock := NewLock(dir)
	if err := lock.Acquire(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer lock.Release()
	return fn()
}
