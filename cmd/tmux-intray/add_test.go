package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddCmdArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		wantMsg string
	}{
		{name: "no args returns error", args: []string{}, wantErr: true, wantMsg: "add requires a message"},
		{name: "one arg returns no error", args: []string{"hello"}, wantErr: false},
		{name: "multiple args returns no error", args: []string{"hello", "world"}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, stderr := runAddArgsSafely(t, tt.args)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantMsg != "" && !strings.Contains(stderr, tt.wantMsg) {
				t.Fatalf("expected stderr to contain %q, got %q", tt.wantMsg, stderr)
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
			err, _ := runAddArgsSafely(t, tt.args)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestNewAddCmdPanicsWhenClientIsNil(t *testing.T) {
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

	NewAddCmd(nil)
}

func runAddArgsSafely(t *testing.T, args []string) (err error, stderr string) {
	t.Helper()

	client := &fakeAddClient{}
	add := NewAddCmd(client)
	errBuffer := &bytes.Buffer{}
	add.SetErr(errBuffer)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("addCmd.Args panicked with args %v: %v", args, r)
		}
	}()

	err = add.Args(add, args)
	return err, errBuffer.String()
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
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "")
	t.Setenv("BATS_TMPDIR", "")
	t.Setenv("CI", "")
	t.Setenv("TMUX_AVAILABLE", "")

	client := &fakeAddClient{ensureTmuxRunningResult: false}
	add := NewAddCmd(client)
	setFlag(t, add, "session", "   ")
	setFlag(t, add, "window", "\t")
	setFlag(t, add, "pane", "\n")

	err := add.RunE(add, []string{"hello"})
	if err == nil || !strings.Contains(err.Error(), "tmux not running") {
		t.Fatalf("expected tmux not running error, got %v", err)
	}
	if client.addCalled {
		t.Fatalf("expected AddTrayItem not to be called")
	}
	if client.ensureCalls != 1 {
		t.Fatalf("expected EnsureTmuxRunning to be called once, got %d", client.ensureCalls)
	}
}

func TestAddRunEAllowsTmuxlessAndStillValidatesMessage(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	t.Setenv("BATS_TMPDIR", "")
	t.Setenv("CI", "")
	t.Setenv("TMUX_AVAILABLE", "")

	client := &fakeAddClient{ensureTmuxRunningResult: false}
	add := NewAddCmd(client)

	err := add.RunE(add, []string{strings.Repeat("a", 1001)})
	if err == nil || !strings.Contains(err.Error(), "message too long") {
		t.Fatalf("expected message length validation error, got %v", err)
	}
	if client.ensureCalls != 1 {
		t.Fatalf("expected EnsureTmuxRunning to be called once, got %d", client.ensureCalls)
	}
	if client.addCalled {
		t.Fatalf("expected AddTrayItem not to be called")
	}
}

func TestAddRunETmuxlessFallbackPassesNoAssociateToClient(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	t.Setenv("BATS_TMPDIR", "")
	t.Setenv("CI", "")
	t.Setenv("TMUX_AVAILABLE", "")

	client := &fakeAddClient{ensureTmuxRunningResult: false}
	add := NewAddCmd(client)

	err := add.RunE(add, []string{"hello"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.ensureCalls != 1 {
		t.Fatalf("expected EnsureTmuxRunning to be called once, got %d", client.ensureCalls)
	}
	if !client.addCalled {
		t.Fatalf("expected AddTrayItem to be called")
	}
	if !client.captured.noAssociate {
		t.Fatalf("expected noAssociate=true in tmuxless fallback")
	}
}

func TestAddRunENoAssociateSkipsTmuxAndWrapsAddError(t *testing.T) {
	client := &fakeAddClient{
		ensureTmuxRunningResult: false,
		addErr:                  errors.New("boom"),
	}
	add := NewAddCmd(client)
	setFlag(t, add, "no-associate", "true")
	setFlag(t, add, "session", " sess-1 ")
	setFlag(t, add, "window", " win-2 ")
	setFlag(t, add, "pane", " pane-3 ")
	setFlag(t, add, "pane-created", "1700000000")
	setFlag(t, add, "level", "")

	err := add.RunE(add, []string{"hello", "world"})
	if err == nil || !strings.Contains(err.Error(), "add: failed to add tray item: boom") {
		t.Fatalf("expected wrapped add error, got %v", err)
	}
	if client.ensureCalls != 0 {
		t.Fatalf("expected EnsureTmuxRunning not to be called, got %d calls", client.ensureCalls)
	}
	if client.captured.message != "hello world" {
		t.Fatalf("expected joined message, got %q", client.captured.message)
	}
	if client.captured.session != "sess-1" || client.captured.window != "win-2" || client.captured.pane != "pane-3" {
		t.Fatalf("expected trimmed context flags, got session=%q window=%q pane=%q", client.captured.session, client.captured.window, client.captured.pane)
	}
	if client.captured.paneCreated != "1700000000" {
		t.Fatalf("expected paneCreated to pass through, got %q", client.captured.paneCreated)
	}
	if !client.captured.noAssociate {
		t.Fatalf("expected noAssociate=true to be forwarded")
	}
	if client.captured.level != "info" {
		t.Fatalf("expected default level info, got %q", client.captured.level)
	}
}

type fakeAddClient struct {
	ensureTmuxRunningResult bool
	ensureCalls             int
	addCalled               bool
	addErr                  error
	captured                struct {
		message     string
		session     string
		window      string
		pane        string
		paneCreated string
		noAssociate bool
		level       string
	}
}

func (f *fakeAddClient) EnsureTmuxRunning() bool {
	f.ensureCalls++
	return f.ensureTmuxRunningResult
}

func (f *fakeAddClient) AddTrayItem(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error) {
	f.addCalled = true
	f.captured.message = item
	f.captured.session = session
	f.captured.window = window
	f.captured.pane = pane
	f.captured.paneCreated = paneCreated
	f.captured.noAssociate = noAssociate
	f.captured.level = level
	return "", f.addErr
}

func setFlag(t *testing.T, command *cobra.Command, name, value string) {
	t.Helper()
	if err := command.Flags().Set(name, value); err != nil {
		t.Fatalf("set flag %q: %v", name, err)
	}
}
