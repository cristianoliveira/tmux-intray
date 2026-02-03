package colors

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestError(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	Error("something went wrong")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Errorf("Error output missing 'Error:' prefix: %q", output)
	}
	if !strings.Contains(output, "something went wrong") {
		t.Errorf("Error output missing message: %q", output)
	}
	if !strings.Contains(output, "\033[0;31m") {
		t.Errorf("Error output missing red color code: %q", output)
	}
}

func TestSuccess(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	Success("operation completed")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Errorf("Success output missing checkmark: %q", output)
	}
	if !strings.Contains(output, "operation completed") {
		t.Errorf("Success output missing message: %q", output)
	}
	if !strings.Contains(output, "\033[0;32m") {
		t.Errorf("Success output missing green color code: %q", output)
	}
}

func TestWarning(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	Warning("this is a warning")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "Warning:") {
		t.Errorf("Warning output missing 'Warning:' prefix: %q", output)
	}
	if !strings.Contains(output, "this is a warning") {
		t.Errorf("Warning output missing message: %q", output)
	}
	if !strings.Contains(output, "\033[1;33m") {
		t.Errorf("Warning output missing yellow color code: %q", output)
	}
}

func TestInfo(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	Info("informational message")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "informational message") {
		t.Errorf("Info output missing message: %q", output)
	}
	if !strings.Contains(output, "\033[0;34m") {
		t.Errorf("Info output missing blue color code: %q", output)
	}
}

func TestLogInfo(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	LogInfo("log message")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "log message") {
		t.Errorf("LogInfo output missing message: %q", output)
	}
	if !strings.Contains(output, "\033[0;34m") {
		t.Errorf("LogInfo output missing blue color code: %q", output)
	}
}

func TestDebugEnabled(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Enable debug
	SetDebug(true)
	defer SetDebug(false)

	Debug("debug message")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "Debug:") {
		t.Errorf("Debug output missing 'Debug:' prefix: %q", output)
	}
	if !strings.Contains(output, "debug message") {
		t.Errorf("Debug output missing message: %q", output)
	}
	if !strings.Contains(output, "\033[0;36m") {
		t.Errorf("Debug output missing cyan color code: %q", output)
	}
}

func TestDebugDisabled(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	SetDebug(false)
	Debug("debug message")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if output != "" {
		t.Errorf("Debug output should be empty when disabled, got: %q", output)
	}
}

func TestEnvironmentDebugEnabled(t *testing.T) {
	// Temporarily set environment variable
	if err := os.Setenv("TMUX_INTRAY_DEBUG", "true"); err != nil {
		t.Fatal(err)
	}
	// Reset the package-level variable by re-initializing
	// Since init() already ran, we need to manually set
	// We'll just test that SetDebug works; environment variable is read in init()
	// but we can't easily re-run init. We'll skip this test for now.
	// Instead we'll test SetDebug separately.
	t.Skip("environment variable test requires package reload")
}

func TestMultipleArguments(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	Info("multiple", "arguments", "joined")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	expected := "multiple arguments joined"
	if !strings.Contains(output, expected) {
		t.Errorf("Info output missing joined arguments: got %q, want substring %q", output, expected)
	}
}

func TestColorConstants(t *testing.T) {
	// Ensure constants are non-empty
	if Red == "" || Green == "" || Yellow == "" || Blue == "" || Cyan == "" || Reset == "" {
		t.Error("Color constants should not be empty")
	}
}
