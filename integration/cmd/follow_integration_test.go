//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

func TestFollowIntegration(t *testing.T) {
	// Setup temporary state directory
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Reset storage state and initialize
	storage.Reset()
	storage.Init()

	// Add a notification
	_, err := storage.AddNotification("integration test", "", "", "", "", "", "info")
	if err != nil {
		t.Fatalf("Failed to add notification: %v", err)
	}

	// Create tick channel
	tickChan := make(chan time.Time)
	defer close(tickChan)

	// Capture output
	var buf bytes.Buffer
	opts := cmd.FollowOptions{
		State:    "active",
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Follow(ctx, opts)
	}()

	// Trigger tick
	tickChan <- time.Now()
	// Wait for processing
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	output := buf.String()
	if !strings.Contains(output, "integration test") {
		t.Errorf("Expected output to contain 'integration test', got: %s", output)
	}
	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected output to contain level info")
	}
}

func TestFollowIntegrationWithPane(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")
	storage.Reset()
	storage.Init()

	// Add notification with pane
	_, err := storage.AddNotification("pane test", "", "sess", "win", "%123", "", "warning")
	if err != nil {
		t.Fatalf("Failed to add notification: %v", err)
	}

	tickChan := make(chan time.Time)
	defer close(tickChan)
	var buf bytes.Buffer
	opts := cmd.FollowOptions{
		State:    "active",
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Follow(ctx, opts)
	}()

	tickChan <- time.Now()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	output := buf.String()
	if !strings.Contains(output, "pane test") {
		t.Errorf("Expected output to contain 'pane test'")
	}
	if !strings.Contains(output, "[warning]") {
		t.Errorf("Expected output to contain level warning")
	}
	if !strings.Contains(output, "└─ From pane: %123") {
		t.Errorf("Expected pane info line")
	}
}
