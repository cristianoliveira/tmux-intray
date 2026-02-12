package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestClearAllSuccess(t *testing.T) {
	originalClearAllFunc := clearAllFunc
	defer func() { clearAllFunc = originalClearAllFunc }()

	called := false
	clearAllFunc = func() error {
		called = true
		return nil
	}

	err := ClearAll()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected clearAllFunc to be called")
	}
}

func TestClearAllError(t *testing.T) {
	originalClearAllFunc := clearAllFunc
	defer func() { clearAllFunc = originalClearAllFunc }()

	expectedErr := errors.New("storage error")
	clearAllFunc = func() error {
		return expectedErr
	}

	err := ClearAll()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestClearAllMultipleCalls(t *testing.T) {
	originalClearAllFunc := clearAllFunc
	defer func() { clearAllFunc = originalClearAllFunc }()

	count := 0
	clearAllFunc = func() error {
		count++
		return nil
	}

	_ = ClearAll()
	_ = ClearAll()
	if count != 2 {
		t.Errorf("Expected clearAllFunc to be called 2 times, got %d", count)
	}
}

type fakeClearClient struct {
	clearCalled bool
	clearErr    error
}

func (f *fakeClearClient) ClearTrayItems() error {
	f.clearCalled = true
	return f.clearErr
}

func TestNewClearCmdPanicsWhenClientIsNil(t *testing.T) {
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

	NewClearCmd(nil)
}

func TestClearCmdRunESuccessWithTmuxlessMode(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	client := &fakeClearClient{}
	cmd := NewClearCmd(client)

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !client.clearCalled {
		t.Fatal("expected ClearTrayItems to be called")
	}
}

func TestClearCmdRunEErrorWithTmuxlessMode(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	expectedErr := errors.New("storage error")
	client := &fakeClearClient{clearErr: expectedErr}
	cmd := NewClearCmd(client)

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "clear: failed to clear tray items") {
		t.Fatalf("expected error to contain 'clear: failed to clear tray items', got %q", err.Error())
	}
	if !client.clearCalled {
		t.Fatal("expected ClearTrayItems to be called")
	}
}

func TestClearCmdRunEWithConfirmationYes(t *testing.T) {
	// Ensure tmuxless mode is false
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
	t.Setenv("CI", "")
	t.Setenv("BATS_TMPDIR", "")
	t.Setenv("TMUX_AVAILABLE", "")

	// Mock stdin with "y\n"
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()

	client := &fakeClearClient{}
	cmd := NewClearCmd(client)

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !client.clearCalled {
		t.Fatal("expected ClearTrayItems to be called")
	}
}

func TestClearCmdRunEWithConfirmationNo(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
	t.Setenv("CI", "")
	t.Setenv("BATS_TMPDIR", "")
	t.Setenv("TMUX_AVAILABLE", "")

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r
	go func() {
		w.Write([]byte("n\n"))
		w.Close()
	}()

	client := &fakeClearClient{}
	cmd := NewClearCmd(client)

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.clearCalled {
		t.Fatal("expected ClearTrayItems not to be called")
	}
}

func TestConfirmClearAllYes(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()

	result := confirmClearAll()
	if !result {
		t.Fatal("expected confirmClearAll to return true for 'y'")
	}
}

func TestConfirmClearAllNo(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r
	go func() {
		w.Write([]byte("n\n"))
		w.Close()
	}()

	result := confirmClearAll()
	if result {
		t.Fatal("expected confirmClearAll to return false for 'n'")
	}
}

func TestConfirmClearAllEmptyInput(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r
	go func() {
		w.Write([]byte("\n"))
		w.Close()
	}()

	result := confirmClearAll()
	if result {
		t.Fatal("expected confirmClearAll to return false for empty input")
	}
}

func TestConfirmClearAllReadError(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close() // Close write side to cause read error
	defer func() { os.Stdin = oldStdin }()

	result := confirmClearAll()
	if result {
		t.Fatal("expected confirmClearAll to return false on read error")
	}
}
