package app

import (
	"errors"
	"strings"
	"testing"
)

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

func TestNewAddUseCasePanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "client dependency cannot be nil") {
			t.Fatalf("unexpected panic message: %q", msg)
		}
	}()

	NewAddUseCase(nil)
}

func TestValidateAddMessage(t *testing.T) {
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
			err := ValidateAddMessage(tt.message)
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

func TestAddUseCaseExecuteRejectsEmptyMessage(t *testing.T) {
	client := &fakeAddClient{ensureTmuxRunningResult: true}
	useCase := NewAddUseCase(client)

	err := useCase.Execute(AddInput{Args: []string{}})
	if err == nil || !strings.Contains(err.Error(), "message cannot be empty") {
		t.Fatalf("expected validation error, got %v", err)
	}
	if client.addCalled {
		t.Fatalf("expected AddTrayItem not to be called")
	}
}

func TestAddUseCaseExecuteRequiresTmuxWhenAutoAssociating(t *testing.T) {
	client := &fakeAddClient{ensureTmuxRunningResult: false}
	useCase := NewAddUseCase(client)

	err := useCase.Execute(AddInput{
		Args:    []string{"hello"},
		Session: "   ",
		Window:  "\t",
		Pane:    "\n",
	})
	if err == nil || !strings.Contains(err.Error(), "tmux not running") {
		t.Fatalf("expected tmux not running error, got %v", err)
	}
	if client.ensureCalls != 1 {
		t.Fatalf("expected EnsureTmuxRunning once, got %d", client.ensureCalls)
	}
	if client.addCalled {
		t.Fatalf("expected AddTrayItem not to be called")
	}
}

func TestAddUseCaseExecuteAllowsTmuxlessFallback(t *testing.T) {
	client := &fakeAddClient{ensureTmuxRunningResult: false}
	useCase := NewAddUseCase(client)

	err := useCase.Execute(AddInput{
		Args: []string{"hello"},
		AllowTmuxless: func() bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.ensureCalls != 1 {
		t.Fatalf("expected EnsureTmuxRunning once, got %d", client.ensureCalls)
	}
	if !client.addCalled {
		t.Fatalf("expected AddTrayItem to be called")
	}
	if !client.captured.noAssociate {
		t.Fatalf("expected noAssociate=true in tmuxless fallback")
	}
}

func TestAddUseCaseExecuteSkipsTmuxForNoAssociateAndWrapsError(t *testing.T) {
	client := &fakeAddClient{ensureTmuxRunningResult: false, addErr: errors.New("boom")}
	useCase := NewAddUseCase(client)

	err := useCase.Execute(AddInput{
		Args:        []string{"hello", "world"},
		Session:     " sess-1 ",
		Window:      " win-2 ",
		Pane:        " pane-3 ",
		PaneCreated: "1700000000",
		NoAssociate: true,
	})
	if err == nil || !strings.Contains(err.Error(), "add: failed to add tray item: boom") {
		t.Fatalf("expected wrapped add error, got %v", err)
	}
	if client.ensureCalls != 0 {
		t.Fatalf("expected EnsureTmuxRunning not to be called, got %d", client.ensureCalls)
	}
	if client.captured.message != "hello world" {
		t.Fatalf("expected joined message, got %q", client.captured.message)
	}
	if client.captured.session != "sess-1" || client.captured.window != "win-2" || client.captured.pane != "pane-3" {
		t.Fatalf("expected trimmed context, got session=%q window=%q pane=%q", client.captured.session, client.captured.window, client.captured.pane)
	}
	if client.captured.paneCreated != "1700000000" {
		t.Fatalf("expected paneCreated to pass through, got %q", client.captured.paneCreated)
	}
	if !client.captured.noAssociate {
		t.Fatalf("expected noAssociate=true")
	}
	if client.captured.level != "info" {
		t.Fatalf("expected default level info, got %q", client.captured.level)
	}
}

func TestAddUseCaseExecuteUsesExplicitLevel(t *testing.T) {
	client := &fakeAddClient{ensureTmuxRunningResult: true}
	useCase := NewAddUseCase(client)

	err := useCase.Execute(AddInput{
		Args:  []string{"hello"},
		Level: "warning",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.captured.level != "warning" {
		t.Fatalf("expected warning level, got %q", client.captured.level)
	}
}
