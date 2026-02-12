package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddCmdArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "no args returns error", args: []string{}, wantErr: true},
		{name: "one arg returns no error", args: []string{"hello"}, wantErr: false},
		{name: "multiple args returns no error", args: []string{"hello", "world"}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runAddArgsSafely(t, tt.args)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestAddCmdArgsNilAndEmptySlicesAreStable(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "nil args", args: nil},
		{name: "empty args", args: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runAddArgsSafely(t, tt.args)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func runAddArgsSafely(t *testing.T, args []string) (err error) {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.SetErr(io.Discard)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("addCmd.Args panicked with args %v: %v", args, r)
		}
	}()

	return addCmd.Args(cmd, args)
}

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantErr     bool
		errContains string
	}{
		{name: "empty", message: "", wantErr: true, errContains: "message cannot be empty"},
		{name: "whitespace only", message: " \n\t ", wantErr: true, errContains: "message cannot be empty"},
		{name: "single character", message: "a", wantErr: false},
		{name: "trimmed but valid", message: "  hello world  ", wantErr: false},
		{name: "exactly max length", message: strings.Repeat("a", 1000), wantErr: false},
		{name: "over max length", message: strings.Repeat("a", 1001), wantErr: true, errContains: "message too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessage(tt.message)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
				t.Fatalf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestAddRunEAutoAssociationRequiresTmux(t *testing.T) {
	resetAddStateForTests()

	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
	t.Setenv("BATS_TMPDIR", "")
	t.Setenv("CI", "")
	t.Setenv("TMUX_AVAILABLE", "")

	originalEnsure := addEnsureTmuxRunningFunc
	originalAdd := addTrayItemFunc
	t.Cleanup(func() {
		addEnsureTmuxRunningFunc = originalEnsure
		addTrayItemFunc = originalAdd
	})

	addEnsureTmuxRunningFunc = func() bool { return false }
	addCalled := false
	addTrayItemFunc = func(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error) {
		addCalled = true
		return "", nil
	}

	sessionFlag = "   "
	windowFlag = "\t"
	paneFlag = "\n"

	err := addCmd.RunE(addCmd, []string{"hello"})
	if err == nil || !strings.Contains(err.Error(), "tmux not running") {
		t.Fatalf("expected tmux not running error, got %v", err)
	}
	if addCalled {
		t.Fatalf("expected AddTrayItem not to be called")
	}
	if sessionFlag != "" || windowFlag != "" || paneFlag != "" {
		t.Fatalf("expected context flags to be trimmed to empty, got session=%q window=%q pane=%q", sessionFlag, windowFlag, paneFlag)
	}
}

func TestAddRunEAllowsTmuxlessAndStillValidatesMessage(t *testing.T) {
	resetAddStateForTests()

	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	t.Setenv("BATS_TMPDIR", "")
	t.Setenv("CI", "")
	t.Setenv("TMUX_AVAILABLE", "")

	originalEnsure := addEnsureTmuxRunningFunc
	originalAdd := addTrayItemFunc
	t.Cleanup(func() {
		addEnsureTmuxRunningFunc = originalEnsure
		addTrayItemFunc = originalAdd
	})

	addEnsureTmuxRunningFunc = func() bool { return false }
	addCalled := false
	addTrayItemFunc = func(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error) {
		addCalled = true
		return "", nil
	}

	err := addCmd.RunE(addCmd, []string{strings.Repeat("a", 1001)})
	if err == nil || !strings.Contains(err.Error(), "message too long") {
		t.Fatalf("expected message length validation error, got %v", err)
	}
	if !noAssociateFlag {
		t.Fatalf("expected noAssociateFlag to be enabled in tmuxless mode")
	}
	if addCalled {
		t.Fatalf("expected AddTrayItem not to be called")
	}
}

func TestAddRunENoAssociateSkipsTmuxAndWrapsAddError(t *testing.T) {
	resetAddStateForTests()

	originalEnsure := addEnsureTmuxRunningFunc
	originalAdd := addTrayItemFunc
	t.Cleanup(func() {
		addEnsureTmuxRunningFunc = originalEnsure
		addTrayItemFunc = originalAdd
	})

	ensureCalls := 0
	addEnsureTmuxRunningFunc = func() bool {
		ensureCalls++
		return false
	}

	noAssociateFlag = true
	sessionFlag = " sess-1 "
	windowFlag = " win-2 "
	paneFlag = " pane-3 "
	paneCreatedFlag = "1700000000"
	levelFlag = ""

	var captured struct {
		message     string
		session     string
		window      string
		pane        string
		paneCreated string
		noAssociate bool
		level       string
	}
	addTrayItemFunc = func(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error) {
		captured.message = item
		captured.session = session
		captured.window = window
		captured.pane = pane
		captured.paneCreated = paneCreated
		captured.noAssociate = noAssociate
		captured.level = level
		return "", errors.New("boom")
	}

	err := addCmd.RunE(addCmd, []string{"hello", "world"})
	if err == nil || !strings.Contains(err.Error(), "add: failed to add tray item: boom") {
		t.Fatalf("expected wrapped add error, got %v", err)
	}
	if ensureCalls != 0 {
		t.Fatalf("expected EnsureTmuxRunning not to be called, got %d calls", ensureCalls)
	}
	if captured.message != "hello world" {
		t.Fatalf("expected joined message, got %q", captured.message)
	}
	if captured.session != "sess-1" || captured.window != "win-2" || captured.pane != "pane-3" {
		t.Fatalf("expected trimmed context flags, got session=%q window=%q pane=%q", captured.session, captured.window, captured.pane)
	}
	if captured.paneCreated != "1700000000" {
		t.Fatalf("expected paneCreated to pass through, got %q", captured.paneCreated)
	}
	if !captured.noAssociate {
		t.Fatalf("expected noAssociate=true to be forwarded")
	}
	if captured.level != "info" {
		t.Fatalf("expected default level info, got %q", captured.level)
	}
}

func resetAddStateForTests() {
	sessionFlag = ""
	windowFlag = ""
	paneFlag = ""
	paneCreatedFlag = ""
	noAssociateFlag = false
	levelFlag = "info"
}
