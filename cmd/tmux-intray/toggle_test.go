package main

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/core"
)

func TestToggleCommand(t *testing.T) {
	// Skip if tmux not running (use core abstraction)
	if !core.EnsureTmuxRunning() {
		t.Skip("tmux not running, skipping toggle test")
	}

	// Test the toggle logic directly rather than through the binary
	// to avoid race conditions with external state

	// Test GetCurrentVisibility and Toggle functions
	tests := []struct {
		name     string
		initial  string
		expected bool
		msg      string
	}{
		{
			name:     "visible to hidden",
			initial:  "1",
			expected: false,
			msg:      "Tray hidden",
		},
		{
			name:     "hidden to visible",
			initial:  "0",
			expected: true,
			msg:      "Tray visible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock functions
			origGet := toggleGetVisibilityFunc
			origSet := toggleSetVisibilityFunc

			// Restore original functions after test
			defer func() {
				toggleGetVisibilityFunc = origGet
				toggleSetVisibilityFunc = origSet
			}()

			// Mock get visibility to return our test value
			toggleGetVisibilityFunc = func() (string, error) {
				return tt.initial, nil
			}

			// Mock set visibility to capture the value
			var setVisibilityCalled bool
			var capturedVisibility bool
			toggleSetVisibilityFunc = func(visible bool) error {
				setVisibilityCalled = true
				capturedVisibility = visible
				return nil
			}

			// Test Toggle function
			result, err := Toggle()
			if err != nil {
				t.Fatalf("Toggle() failed: %v", err)
			}

			if !setVisibilityCalled {
				t.Error("SetVisibility was not called")
			}

			if capturedVisibility != tt.expected {
				t.Errorf("Expected SetVisibility to be called with %v, got %v",
					tt.expected, capturedVisibility)
			}

			if result != tt.expected {
				t.Errorf("Expected Toggle to return %v, got %v",
					tt.expected, result)
			}
		})
	}
}
