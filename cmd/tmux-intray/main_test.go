package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

func captureMainStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read from pipe: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("failed to close pipe reader: %v", err)
	}

	return buf.String()
}

func TestRunNonTUILogsStartupAndCompletion(t *testing.T) {
	colors.EnableStructuredLogging()
	defer colors.EnableStructuredLogging()
	colors.SetDebug(true)
	defer colors.SetDebug(false)

	var exitCode int
	output := captureMainStderr(t, func() {
		exitCode = run([]string{"list"}, func() error { return nil })
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, `"component":"startup"`) {
		t.Fatalf("expected startup structured logs, got %q", output)
	}
	if !strings.Contains(output, `"status":"started"`) {
		t.Fatalf("expected started structured log, got %q", output)
	}
	if !strings.Contains(output, `"status":"completed"`) {
		t.Fatalf("expected completed structured log, got %q", output)
	}
}

func TestRunNonTUILogsFailure(t *testing.T) {
	colors.EnableStructuredLogging()
	defer colors.EnableStructuredLogging()
	colors.SetDebug(true)
	defer colors.SetDebug(false)

	var exitCode int
	output := captureMainStderr(t, func() {
		exitCode = run([]string{"list"}, func() error { return errors.New("boom") })
	})

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(output, `"status":"failed"`) {
		t.Fatalf("expected failed structured log, got %q", output)
	}
}

func TestRunTUISkipsStartupStructuredLogs(t *testing.T) {
	colors.EnableStructuredLogging()
	defer colors.EnableStructuredLogging()
	colors.SetDebug(true)
	defer colors.SetDebug(false)

	var exitCode int
	output := captureMainStderr(t, func() {
		exitCode = run([]string{"tui"}, func() error { return nil })
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if output != "" {
		t.Fatalf("expected no structured logs for tui command, got %q", output)
	}
}
