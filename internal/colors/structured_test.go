package colors

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStderr(t *testing.T, fn func()) string {
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

func TestStructuredDebugIsGatedByDebugMode(t *testing.T) {
	EnableStructuredLogging()
	defer EnableStructuredLogging()
	SetDebug(false)
	defer SetDebug(false)

	output := captureStderr(t, func() {
		StructuredDebug("colors", "debug_disabled", "skipped", nil, "", nil)
	})

	if output != "" {
		t.Fatalf("expected no structured output when debug disabled, got %q", output)
	}

	SetDebug(true)
	output = captureStderr(t, func() {
		StructuredDebug("colors", "debug_enabled", "written", nil, "", nil)
	})

	if !strings.Contains(output, `"level":"debug"`) {
		t.Fatalf("expected structured debug output, got %q", output)
	}
}

func TestStructuredLoggingCanBeDisabled(t *testing.T) {
	SetDebug(true)
	defer SetDebug(false)
	DisableStructuredLogging()
	defer EnableStructuredLogging()

	output := captureStderr(t, func() {
		StructuredInfo("colors", "disabled", "skipped", nil, "", nil)
	})

	if output != "" {
		t.Fatalf("expected no structured output when disabled, got %q", output)
	}
}
