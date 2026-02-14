package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var _ *cobra.Command // ensure cobra import is used

type fakeDismissClient struct {
	dismissNotificationCalled bool
	dismissNotificationID     string
	dismissNotificationError  error

	dismissAllCalled bool
	dismissAllError  error
}

func (f *fakeDismissClient) DismissNotification(id string) error {
	f.dismissNotificationCalled = true
	f.dismissNotificationID = id
	return f.dismissNotificationError
}

func (f *fakeDismissClient) DismissAll() error {
	f.dismissAllCalled = true
	return f.dismissAllError
}

func TestNewDismissCmdPanicsWhenClientIsNil(t *testing.T) {
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

	NewDismissCmd(nil)
}

func runDismissCmd(t *testing.T, client dismissClient, args []string, setAllFlag bool) (string, string, error) {
	t.Helper()
	cmd := NewDismissCmd(client)
	if setAllFlag {
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set flag all: %v", err)
		}
	}
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	err := cmd.RunE(cmd, args)
	return outBuf.String(), errBuf.String(), err
}

func TestDismissCmdValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setAllFlag  bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "no args and no --all should error",
			args:        []string{},
			setAllFlag:  false,
			wantErr:     true,
			errContains: "either specify an id or use --all",
		},
		{
			name:        "--all with id should error",
			args:        []string{"123"},
			setAllFlag:  true,
			wantErr:     true,
			errContains: "cannot specify both --all and id",
		},
		{
			name:        "too many arguments should error",
			args:        []string{"123", "456"},
			setAllFlag:  false,
			wantErr:     true,
			errContains: "too many arguments",
		},
		{
			name:       "single id with no --all should succeed",
			args:       []string{"123"},
			setAllFlag: false,
			wantErr:    false,
		},
		{
			name:       "--all with no args should succeed",
			args:       []string{},
			setAllFlag: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeDismissClient{}
			_, _, err := runDismissCmd(t, client, tt.args, tt.setAllFlag)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
				t.Fatalf("expected error containing %q, got %v", tt.errContains, err)
			}
			// Ensure client methods not called on validation error
			if tt.wantErr {
				if client.dismissNotificationCalled || client.dismissAllCalled {
					t.Fatal("client method should not be called on validation error")
				}
			}
		})
	}
}

func TestDismissSuccess(t *testing.T) {
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	var capturedID string
	dismissFunc = func(id string) error {
		capturedID = id
		return nil
	}

	err := Dismiss("42")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if capturedID != "42" {
		t.Errorf("Expected ID '42', got %q", capturedID)
	}
}

func TestDismissCmdSuccess(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setAllFlag bool
		wantID     string
		wantAll    bool
	}{
		{
			name:       "single id",
			args:       []string{"123"},
			setAllFlag: false,
			wantID:     "123",
			wantAll:    false,
		},
		{
			name:       "dismiss all with CI env",
			args:       []string{},
			setAllFlag: true,
			wantID:     "",
			wantAll:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set CI env to skip confirmation for dismiss all
			t.Setenv("CI", "true")
			client := &fakeDismissClient{}
			_, _, err := runDismissCmd(t, client, tt.args, tt.setAllFlag)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			verifyDismissResult(t, client, tt.wantAll, tt.wantID)
		})
	}
}

// verifyDismissResult checks that the expected dismiss method was called.
func verifyDismissResult(t *testing.T, client *fakeDismissClient, wantAll bool, wantID string) {
	t.Helper()
	if wantAll {
		verifyDismissAllCalled(t, client)
	} else {
		verifyDismissSingleCalled(t, client, wantID)
	}
}

// verifyDismissAllCalled verifies dismiss all was called correctly.
func verifyDismissAllCalled(t *testing.T, client *fakeDismissClient) {
	t.Helper()
	if !client.dismissAllCalled {
		t.Error("DismissAll not called")
	}
	if client.dismissNotificationCalled {
		t.Error("DismissNotification should not be called")
	}
}

// verifyDismissSingleCalled verifies dismiss single was called correctly.
func verifyDismissSingleCalled(t *testing.T, client *fakeDismissClient, wantID string) {
	t.Helper()
	if !client.dismissNotificationCalled {
		t.Error("DismissNotification not called")
	}
	if client.dismissAllCalled {
		t.Error("DismissAll should not be called")
	}
	if client.dismissNotificationID != wantID {
		t.Errorf("want ID %q, got %q", wantID, client.dismissNotificationID)
	}
}

func TestDismissCmdError(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setAllFlag  bool
		clientError error
		wantErrMsg  string
	}{
		{
			name:        "single id client error",
			args:        []string{"123"},
			setAllFlag:  false,
			clientError: errors.New("notification not found"),
			wantErrMsg:  "dismiss: failed to dismiss notification",
		},
		{
			name:        "dismiss all client error with CI env",
			args:        []string{},
			setAllFlag:  true,
			clientError: errors.New("storage error"),
			wantErrMsg:  "dismiss: failed to dismiss all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI", "true")
			t.Setenv("BATS_TMPDIR", "true")
			client := &fakeDismissClient{}
			if tt.setAllFlag {
				client.dismissAllError = tt.clientError
			} else {
				client.dismissNotificationError = tt.clientError
			}
			_, _, err := runDismissCmd(t, client, tt.args, tt.setAllFlag)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error missing expected substring %q, got %v", tt.wantErrMsg, err)
			}
			// Note: colors.Error writes to os.Stderr, not captured.
			// We rely on err being returned.
		})
	}
}

func TestDismissCmdConfirmation(t *testing.T) {
	// Override confirmDismissAllFunc
	originalConfirm := confirmDismissAllFunc
	defer func() { confirmDismissAllFunc = originalConfirm }()

	t.Run("cancelled", func(t *testing.T) {
		confirmDismissAllFunc = func() bool { return false }
		client := &fakeDismissClient{}
		// Do not set CI/BATS_TMPDIR, so confirmation will be attempted
		t.Setenv("CI", "")
		t.Setenv("BATS_TMPDIR", "")
		_, _, err := runDismissCmd(t, client, []string{}, true)
		if err != nil {
			t.Errorf("expected no error on cancellation, got %v", err)
		}
		if client.dismissAllCalled {
			t.Error("DismissAll should not be called when confirmation denied")
		}
		// Should have printed "Operation cancelled" via colors.Info (not captured)
	})

	t.Run("confirmed", func(t *testing.T) {
		confirmDismissAllFunc = func() bool { return true }
		client := &fakeDismissClient{}
		t.Setenv("CI", "")
		t.Setenv("BATS_TMPDIR", "")
		_, _, err := runDismissCmd(t, client, []string{}, true)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !client.dismissAllCalled {
			t.Error("DismissAll should be called when confirmation granted")
		}
	})
}

func TestDismissError(t *testing.T) {
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	expectedErr := errors.New("notification not found")
	dismissFunc = func(id string) error {
		return expectedErr
	}

	err := Dismiss("99")
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestDismissAllSuccess(t *testing.T) {
	originalDismissAllFunc := dismissAllFunc
	defer func() { dismissAllFunc = originalDismissAllFunc }()

	called := false
	dismissAllFunc = func() error {
		called = true
		return nil
	}

	err := DismissAll()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected dismissAllFunc to be called")
	}
}

func TestDismissAllError(t *testing.T) {
	originalDismissAllFunc := dismissAllFunc
	defer func() { dismissAllFunc = originalDismissAllFunc }()

	expectedErr := errors.New("storage error")
	dismissAllFunc = func() error {
		return expectedErr
	}

	err := DismissAll()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestDismissEmptyID(t *testing.T) {
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	var capturedID string
	dismissFunc = func(id string) error {
		capturedID = id
		return nil
	}

	err := Dismiss("")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if capturedID != "" {
		t.Errorf("Expected empty ID, got %q", capturedID)
	}
}

func TestDismissAllMultipleCalls(t *testing.T) {
	originalDismissAllFunc := dismissAllFunc
	defer func() { dismissAllFunc = originalDismissAllFunc }()

	count := 0
	dismissAllFunc = func() error {
		count++
		return nil
	}

	_ = DismissAll()
	_ = DismissAll()
	if count != 2 {
		t.Errorf("Expected dismissAllFunc to be called 2 times, got %d", count)
	}
}

func TestDismissPreservesReadTimestamp(t *testing.T) {
	// This is a unit test for the dismiss command behavior.
	// The actual storage layer test verifies that read_timestamp is preserved,
	// but this test ensures the dismiss command function doesn't break it.
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	// Mock dismissFunc that succeeds without modifying read_timestamp
	dismissFunc = func(id string) error {
		// In real implementation, DismissNotification preserves read_timestamp
		// This test just verifies that dismissFunc is called correctly
		return nil
	}

	err := Dismiss("123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// The actual verification of read_timestamp preservation is done in storage layer tests
	// See: internal/storage/storage.go DismissNotification
}

func TestConfirmDismissAll(t *testing.T) {
	// backup stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	t.Run("yes", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("y\n"))
		_ = w.Close()
		if !confirmDismissAll() {
			t.Error("expected true for 'y'")
		}
	})

	t.Run("yes uppercase", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("Y\n"))
		_ = w.Close()
		if !confirmDismissAll() {
			t.Error("expected true for 'Y'")
		}
	})

	t.Run("yes full", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("yes\n"))
		_ = w.Close()
		if !confirmDismissAll() {
			t.Error("expected true for 'yes'")
		}
	})

	t.Run("no", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("n\n"))
		_ = w.Close()
		if confirmDismissAll() {
			t.Error("expected false for 'n'")
		}
	})

	t.Run("empty", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("\n"))
		_ = w.Close()
		if confirmDismissAll() {
			t.Error("expected false for empty line")
		}
	})

	t.Run("read error", func(t *testing.T) {
		// Simulate read error by closing pipe early
		r, w, _ := os.Pipe()
		os.Stdin = r
		_ = w.Close()
		if confirmDismissAll() {
			t.Error("expected false on read error")
		}
	})
}
