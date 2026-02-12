package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ = cobra.Command{}

type fakeJumpClient struct {
	ensureTmuxRunningResult   bool
	ensureCalls               int
	getNotificationByIDCalls  []string
	getNotificationByIDResult string
	getNotificationByIDErr    error
	validatePaneExistsCalls   []struct{ session, window, pane string }
	validatePaneExistsResult  bool
	jumpToPaneCalls           []struct{ session, window, pane string }
	jumpToPaneResult          bool
	markNotificationReadCalls []string
	markNotificationReadErr   error
}

func (f *fakeJumpClient) EnsureTmuxRunning() bool {
	f.ensureCalls++
	return f.ensureTmuxRunningResult
}

func (f *fakeJumpClient) GetNotificationByID(id string) (string, error) {
	f.getNotificationByIDCalls = append(f.getNotificationByIDCalls, id)
	return f.getNotificationByIDResult, f.getNotificationByIDErr
}

func (f *fakeJumpClient) ValidatePaneExists(session, window, pane string) bool {
	f.validatePaneExistsCalls = append(f.validatePaneExistsCalls, struct{ session, window, pane string }{session, window, pane})
	return f.validatePaneExistsResult
}

func (f *fakeJumpClient) JumpToPane(session, window, pane string) bool {
	f.jumpToPaneCalls = append(f.jumpToPaneCalls, struct{ session, window, pane string }{session, window, pane})
	return f.jumpToPaneResult
}

func (f *fakeJumpClient) MarkNotificationRead(id string) error {
	f.markNotificationReadCalls = append(f.markNotificationReadCalls, id)
	return f.markNotificationReadErr
}

func TestNewJumpCmdPanicsWhenClientIsNil(t *testing.T) {
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

	NewJumpCmd(nil)
}

func TestJumpCmdArgsValidation(t *testing.T) {
	client := &fakeJumpClient{ensureTmuxRunningResult: true}
	cmd := NewJumpCmd(client)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		wantMsg string
	}{
		{name: "no args returns error", args: []string{}, wantErr: true, wantMsg: "jump: requires a notification id"},
		{name: "one arg returns no error", args: []string{"42"}, wantErr: false},
		{name: "multiple args returns error", args: []string{"42", "extra"}, wantErr: true, wantMsg: "jump: requires a notification id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Args(cmd, tt.args)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantMsg != "" && (err == nil || !strings.Contains(err.Error(), tt.wantMsg)) {
				t.Fatalf("expected error containing %q, got %v", tt.wantMsg, err)
			}
		})
	}
}

func TestJumpRunESuccess(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo",
		validatePaneExistsResult:  true,
		jumpToPaneResult:          true,
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.NoError(t, err)
	assert.Equal(t, 1, client.ensureCalls)
	assert.Equal(t, []string{"42"}, client.getNotificationByIDCalls)
	assert.Equal(t, []string{"42"}, client.markNotificationReadCalls)
	assert.Len(t, client.validatePaneExistsCalls, 1)
	assert.Equal(t, "$0", client.validatePaneExistsCalls[0].session)
	assert.Equal(t, "%0", client.validatePaneExistsCalls[0].window)
	assert.Equal(t, ":0.0", client.validatePaneExistsCalls[0].pane)
	assert.Len(t, client.jumpToPaneCalls, 1)
	assert.Equal(t, "$0", client.jumpToPaneCalls[0].session)
	assert.Equal(t, "%0", client.jumpToPaneCalls[0].window)
	assert.Equal(t, ":0.0", client.jumpToPaneCalls[0].pane)
}

func TestJumpRunETmuxNotRunning(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult: false,
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tmux not running")
	assert.Equal(t, 1, client.ensureCalls)
	assert.Empty(t, client.getNotificationByIDCalls)
}

func TestJumpRunENotificationNotFound(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult: true,
		getNotificationByIDErr:  errors.New("not found"),
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jump: not found")
	assert.Equal(t, 1, client.ensureCalls)
	assert.Equal(t, []string{"42"}, client.getNotificationByIDCalls)
}

func TestJumpRunENoPaneAssociation(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\t2025-02-04T10:00:00Z\tactive\t\t\t\thello\t\tinfo",
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")
	assert.Equal(t, 1, client.ensureCalls)
}

func TestJumpRunEPaneDoesNotExistButWindowSelected(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo",
		validatePaneExistsResult:  false,
		jumpToPaneResult:          true,
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.NoError(t, err)
	assert.Len(t, client.validatePaneExistsCalls, 1)
	assert.Equal(t, "$0", client.validatePaneExistsCalls[0].session)
	assert.Len(t, client.jumpToPaneCalls, 1)
	assert.Equal(t, "$0", client.jumpToPaneCalls[0].session)
	// Should still mark as read (default behavior)
	assert.Equal(t, []string{"42"}, client.markNotificationReadCalls)
}

func TestJumpRunEWindowDoesNotExist(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo",
		validatePaneExistsResult:  true,
		jumpToPaneResult:          false,
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to jump because pane or window does not exist")
	assert.Len(t, client.jumpToPaneCalls, 1)
	assert.Equal(t, "$0", client.jumpToPaneCalls[0].session)
	assert.Empty(t, client.markNotificationReadCalls)
}

func TestJumpRunEInvalidLineFormat(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\tactive",
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid notification line format")
}

func TestJumpRunEMarksNotificationReadOnSuccess(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo",
		validatePaneExistsResult:  true,
		jumpToPaneResult:          true,
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.NoError(t, err)
	assert.Equal(t, []string{"42"}, client.markNotificationReadCalls)
}

func TestJumpRunEDoesNotMarkReadWhenJumpFails(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo",
		validatePaneExistsResult:  true,
		jumpToPaneResult:          false,
	}
	cmd := NewJumpCmd(client)

	err := cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Empty(t, client.markNotificationReadCalls)
}

func TestJumpRunENoMarkReadFlagSkipsMarkRead(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult:   true,
		getNotificationByIDResult: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\thello\t1234567890\tinfo",
		validatePaneExistsResult:  true,
		jumpToPaneResult:          true,
	}
	cmd := NewJumpCmd(client)
	require.NoError(t, cmd.Flags().Set("no-mark-read", "true"))

	err := cmd.RunE(cmd, []string{"42"})
	require.NoError(t, err)
	assert.Empty(t, client.markNotificationReadCalls)
}

func TestJumpRunEInvalidFieldData(t *testing.T) {
	client := &fakeJumpClient{
		ensureTmuxRunningResult: true,
	}
	cmd := NewJumpCmd(client)

	// missing session
	client.getNotificationByIDResult = "42\t2025-02-04T10:00:00Z\tactive\t\t%0\t%1\thello\t1234567890\tinfo"
	err := cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")

	// missing window
	client.getNotificationByIDResult = "42\t2025-02-04T10:00:00Z\tactive\t$0\t\t%1\thello\t1234567890\tinfo"
	err = cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")

	// missing pane
	client.getNotificationByIDResult = "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t\thello\t1234567890\tinfo"
	err = cmd.RunE(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")
}
