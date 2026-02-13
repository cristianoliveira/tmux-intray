package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkReadSuccess(t *testing.T) {
	client := &fakeMarkReadClient{}
	cmd := NewMarkReadCmd(client)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{"42"})
	require.NoError(t, err)
	require.Equal(t, "42", client.capturedID)
	// Success message is printed via colors.Success; we can't easily capture colors output
}

func TestMarkReadError(t *testing.T) {
	expectedErr := errors.New("notification not found")
	client := &fakeMarkReadClient{err: expectedErr}
	cmd := NewMarkReadCmd(client)

	err := cmd.RunE(cmd, []string{"99"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mark-read:")
	require.Contains(t, err.Error(), expectedErr.Error())
	require.Equal(t, "99", client.capturedID)
}

func TestMarkReadEmptyID(t *testing.T) {
	client := &fakeMarkReadClient{}
	cmd := NewMarkReadCmd(client)

	// cobra.ExactArgs(1) ensures empty string cannot be passed as argument
	// We'll test that the command validates exactly one argument.
	// The actual ID validation is done by storage layer.
	err := cmd.RunE(cmd, []string{""})
	require.NoError(t, err) // empty string ID is allowed for storage to handle
	require.Equal(t, "", client.capturedID)
}

func TestNewMarkReadCmdPanicsWhenClientIsNil(t *testing.T) {
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

	NewMarkReadCmd(nil)
}

func TestMarkReadCmdArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "no args returns error", args: []string{}, wantErr: true},
		{name: "one arg returns no error", args: []string{"42"}, wantErr: false},
		{name: "two args returns error", args: []string{"42", "extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeMarkReadClient{}
			cmd := NewMarkReadCmd(client)
			err := cmd.Args(cmd, tt.args)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestMarkReadVariousIDs(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		err         error
		wantErr     bool
		errContains string
	}{
		{
			name:        "success with numeric ID",
			id:          "42",
			err:         nil,
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "success with zero ID",
			id:          "0",
			err:         nil,
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "success with leading zeros ID",
			id:          "00123",
			err:         nil,
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "notification not found",
			id:          "999",
			err:         errors.New("notification not found"),
			wantErr:     true,
			errContains: "mark-read: notification not found",
		},
		{
			name:        "invalid notification ID",
			id:          "invalid",
			err:         errors.New("invalid notification ID"),
			wantErr:     true,
			errContains: "mark-read: invalid notification ID",
		},
		{
			name:        "storage initialization error",
			id:          "1",
			err:         errors.New("storage not initialized"),
			wantErr:     true,
			errContains: "mark-read: storage not initialized",
		},
		{
			name:        "empty ID passes to storage",
			id:          "",
			err:         nil,
			wantErr:     false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeMarkReadClient{err: tt.err}
			cmd := NewMarkReadCmd(client)
			err := cmd.RunE(cmd, []string{tt.id})
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.id, client.capturedID)
		})
	}
}

type fakeMarkReadClient struct {
	capturedID string
	err        error
}

func (f *fakeMarkReadClient) MarkNotificationRead(id string) error {
	f.capturedID = id
	return f.err
}
