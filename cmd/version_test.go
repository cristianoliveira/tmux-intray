package cmd

import (
	"bytes"
	"testing"
)

func TestGetVersion(t *testing.T) {
	// Save original version and restore after test
	origVersion := Version
	defer func() { Version = origVersion }()

	// Test default version
	Version = "0.1.0"
	if got := GetVersion(); got != "0.1.0" {
		t.Errorf("GetVersion() = %q, want %q", got, "0.1.0")
	}

	// Test custom version
	Version = "1.2.3"
	if got := GetVersion(); got != "1.2.3" {
		t.Errorf("GetVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestPrintVersion(t *testing.T) {
	// Save original writer and version
	origWriter := versionOutputWriter
	origVersion := Version
	defer func() {
		versionOutputWriter = origWriter
		Version = origVersion
	}()

	var buf bytes.Buffer
	versionOutputWriter = &buf
	Version = "0.1.0"
	PrintVersion()
	expected := "tmux-intray v0.1.0\n"
	if buf.String() != expected {
		t.Errorf("PrintVersion() printed %q, want %q", buf.String(), expected)
	}
}
