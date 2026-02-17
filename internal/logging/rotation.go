package logging

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// rotate removes the oldest log files in dir when the number of files exceeds maxFiles.
// It only removes files that match the naming pattern "tmux-intray_*.log".
func rotate(dir string, maxFiles int) error {
	if maxFiles <= 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var logFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "tmux-intray_") && strings.HasSuffix(name, ".log") {
			logFiles = append(logFiles, filepath.Join(dir, name))
		}
	}
	if len(logFiles) <= maxFiles {
		return nil
	}
	// Sort by modification time (oldest first)
	sort.Slice(logFiles, func(i, j int) bool {
		info1, err1 := os.Stat(logFiles[i])
		info2, err2 := os.Stat(logFiles[j])
		if err1 != nil || err2 != nil {
			// fallback to lexical sort
			return logFiles[i] < logFiles[j]
		}
		return info1.ModTime().Before(info2.ModTime())
	})
	// Remove oldest files
	for i := 0; i < len(logFiles)-maxFiles; i++ {
		os.Remove(logFiles[i]) // ignore errors
	}
	return nil
}
