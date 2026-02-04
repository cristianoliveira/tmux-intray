// Command tmux-intray is a Go wrapper that embeds the bash script and its dependencies.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	assets "github.com/cristianoliveira/tmux-intray"
)

// Version is set via ldflags during build
var Version = "dev"

func main() {
	// Create a temporary directory to extract the project
	tmpDir, err := os.MkdirTemp("", "tmux-intray-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Extract embedded files
	err = extractFS(assets.FS, tmpDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to extract embedded files: %v\n", err)
		os.Exit(1)
	}

	// Path to the bash script
	scriptPath := filepath.Join(tmpDir, "bin", "tmux-intray")
	// Make script executable
	err = os.Chmod(scriptPath, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to make script executable: %v\n", err)
		os.Exit(1)
	}

	// Prepare command
	cmd := exec.Command(scriptPath, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Set environment variable to indicate we're running from temp directory
	cmd.Env = append(os.Environ(), "TMUX_INTRAY_TEMP_ROOT="+tmpDir)

	// Run the command
	err = cmd.Run()
	if err != nil {
		// If the command exited with a non-zero status, propagate it
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Failed to execute tmux-intray: %v\n", err)
		os.Exit(1)
	}
}

// extractFS recursively copies the embedded filesystem to the target directory.
func extractFS(srcFS fs.FS, targetDir string) error {
	return fs.WalkDir(srcFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destPath := filepath.Join(targetDir, path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		data, err := fs.ReadFile(srcFS, path)
		if err != nil {
			return err
		}
		// Preserve executable bit for scripts (simplify: check file extension)
		mode := fs.FileMode(0644)
		if filepath.Ext(path) == ".sh" || filepath.Ext(path) == ".tmux" || filepath.Base(path) == "tmux-intray" {
			mode = 0755
		}
		return os.WriteFile(destPath, data, mode)
	})
}
