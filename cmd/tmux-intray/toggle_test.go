package main

import (
	"errors"
	"strings"
	"testing"
)

type fakeToggleClient struct {
	ensureTmuxRunningResult bool
	getVisibilityResult     string
	getVisibilityError      error
	setVisibilityError      error
	runHookError            error

	ensureCalls   int
	getVisCalls   int
	setVisCalls   int
	setVisCapture bool
	runHookCalls  int
}

func (f *fakeToggleClient) EnsureTmuxRunning() bool {
	f.ensureCalls++
	return f.ensureTmuxRunningResult
}

func (f *fakeToggleClient) GetVisibility() (string, error) {
	f.getVisCalls++
	return f.getVisibilityResult, f.getVisibilityError
}

func (f *fakeToggleClient) SetVisibility(visible bool) error {
	f.setVisCalls++
	f.setVisCapture = visible
	return f.setVisibilityError
}

func (f *fakeToggleClient) RunHook(name string, envVars ...string) error {
	f.runHookCalls++
	return f.runHookError
}

func TestNewToggleCmdPanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "client dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil dependency, got %q", msg)
		}
	}()

	NewToggleCmd(nil)
}

func TestToggleCmdSuccess(t *testing.T) {
	tests := []struct {
		name              string
		initialVisibility string
		expectedSet       bool
		expectedMsg       string
	}{
		{
			name:              "visible to hidden",
			initialVisibility: "1",
			expectedSet:       false,
			expectedMsg:       "Tray hidden",
		},
		{
			name:              "hidden to visible",
			initialVisibility: "0",
			expectedSet:       true,
			expectedMsg:       "Tray visible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeToggleClient{
				ensureTmuxRunningResult: true,
				getVisibilityResult:     tt.initialVisibility,
			}
			cmd := NewToggleCmd(client)

			err := cmd.RunE(cmd, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if client.ensureCalls != 1 {
				t.Errorf("expected EnsureTmuxRunning to be called once, got %d", client.ensureCalls)
			}
			if client.getVisCalls != 1 {
				t.Errorf("expected GetVisibility to be called once, got %d", client.getVisCalls)
			}
			if client.setVisCalls != 1 {
				t.Errorf("expected SetVisibility to be called once, got %d", client.setVisCalls)
			}
			if client.setVisCapture != tt.expectedSet {
				t.Errorf("expected SetVisibility(%v), got %v", tt.expectedSet, client.setVisCapture)
			}
			if client.runHookCalls != 2 {
				t.Errorf("expected RunHook to be called twice (pre and post), got %d", client.runHookCalls)
			}
		})
	}
}

func TestToggleCmdTmuxNotRunning(t *testing.T) {
	// Test without tmuxless mode
	t.Run("error when tmux not running", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
		t.Setenv("BATS_TMPDIR", "")
		t.Setenv("CI", "")

		client := &fakeToggleClient{
			ensureTmuxRunningResult: false,
		}
		cmd := NewToggleCmd(client)

		err := cmd.RunE(cmd, nil)
		if err == nil {
			t.Fatal("expected error when tmux not running")
		}
		if !strings.Contains(err.Error(), "tmux not running") {
			t.Errorf("expected 'tmux not running' error, got %v", err)
		}
	})

	// Test with tmuxless mode
	t.Run("skips when tmuxless mode", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")

		client := &fakeToggleClient{
			ensureTmuxRunningResult: false,
		}
		cmd := NewToggleCmd(client)

		err := cmd.RunE(cmd, nil)
		if err != nil {
			t.Fatalf("expected no error in tmuxless mode, got %v", err)
		}
		if client.getVisCalls != 0 {
			t.Error("expected GetVisibility not to be called when skipping")
		}
	})
}

func TestToggleCmdGetVisibilityError(t *testing.T) {
	expectedErr := errors.New("get visibility failed")
	client := &fakeToggleClient{
		ensureTmuxRunningResult: true,
		getVisibilityError:      expectedErr,
	}
	cmd := NewToggleCmd(client)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestToggleCmdSetVisibilityError(t *testing.T) {
	expectedErr := errors.New("set visibility failed")
	client := &fakeToggleClient{
		ensureTmuxRunningResult: true,
		getVisibilityResult:     "0",
		setVisibilityError:      expectedErr,
	}
	cmd := NewToggleCmd(client)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestToggleCmdPreHookError(t *testing.T) {
	expectedErr := errors.New("hook failed")
	client := &fakeToggleClient{
		ensureTmuxRunningResult: true,
		getVisibilityResult:     "0",
		runHookError:            expectedErr,
	}
	cmd := NewToggleCmd(client)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	// SetVisibility should not be called if pre-hook fails
	if client.setVisCalls != 0 {
		t.Error("expected SetVisibility not to be called when pre-hook fails")
	}
}

func TestToggleCommandLegacy(t *testing.T) {
	// Skip if tmux not running (use core abstraction)
	// This tests the legacy Toggle function

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
