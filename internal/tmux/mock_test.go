// Package tmux provides a unified abstraction layer for tmux operations.
package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockClientGetCurrentContext demonstrates basic MockClient usage for GetCurrentContext.
func TestMockClientGetCurrentContext(t *testing.T) {
	mockClient := new(MockClient)

	// Configure the mock to return a specific context
	expectedContext := TmuxContext{
		SessionID: "$1",
		WindowID:  "@1",
		PaneID:    "%1",
		PanePID:   "1234",
	}
	mockClient.On("GetCurrentContext").Return(expectedContext, nil)

	// Call the method
	ctx, err := mockClient.GetCurrentContext()

	// Verify the results
	require.NoError(t, err)
	assert.Equal(t, "$1", ctx.SessionID)
	assert.Equal(t, "@1", ctx.WindowID)
	assert.Equal(t, "%1", ctx.PaneID)
	assert.Equal(t, "1234", ctx.PanePID)

	// Assert that the method was called as expected
	mockClient.AssertCalled(t, "GetCurrentContext")
	mockClient.AssertExpectations(t)
}

// TestMockClientValidatePaneExists demonstrates MockClient usage for ValidatePaneExists.
func TestMockClientValidatePaneExists(t *testing.T) {
	// Configure the mock for different scenarios
	tests := []struct {
		name      string
		sessionID string
		windowID  string
		paneID    string
		exists    bool
		wantErr   bool
	}{
		{
			name:      "existing pane",
			sessionID: "$1",
			windowID:  "@1",
			paneID:    "%1",
			exists:    true,
			wantErr:   false,
		},
		{
			name:      "non-existent pane",
			sessionID: "$1",
			windowID:  "@1",
			paneID:    "%999",
			exists:    false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			mockClient.On("ValidatePaneExists", tt.sessionID, tt.windowID, tt.paneID).
				Return(tt.exists, nil)

			exists, err := mockClient.ValidatePaneExists(tt.sessionID, tt.windowID, tt.paneID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exists, exists)
			mockClient.AssertExpectations(t)
		})
	}
}

// TestMockClientJumpToPane demonstrates MockClient usage for JumpToPane.
func TestMockClientJumpToPane(t *testing.T) {
	mockClient := new(MockClient)

	// Configure the mock
	mockClient.On("JumpToPane", "$1", "@1", "%1").Return(true, nil)

	// Call the method
	success, err := mockClient.JumpToPane("$1", "@1", "%1")

	// Verify
	require.NoError(t, err)
	assert.True(t, success)
	mockClient.AssertCalled(t, "JumpToPane", "$1", "@1", "%1")
	mockClient.AssertExpectations(t)
}

// TestMockClientEnvironment demonstrates MockClient usage for SetEnvironment and GetEnvironment.
func TestMockClientEnvironment(t *testing.T) {
	mockClient := new(MockClient)

	// Configure SetEnvironment
	mockClient.On("SetEnvironment", "TEST_VAR", "test_value").Return(nil)

	// Configure GetEnvironment
	mockClient.On("GetEnvironment", "TEST_VAR").Return("test_value", nil)

	// Test SetEnvironment
	err := mockClient.SetEnvironment("TEST_VAR", "test_value")
	require.NoError(t, err)
	mockClient.AssertCalled(t, "SetEnvironment", "TEST_VAR", "test_value")

	// Test GetEnvironment
	value, err := mockClient.GetEnvironment("TEST_VAR")
	require.NoError(t, err)
	assert.Equal(t, "test_value", value)
	mockClient.AssertCalled(t, "GetEnvironment", "TEST_VAR")

	mockClient.AssertExpectations(t)
}

// TestMockClientHasSession demonstrates MockClient usage for HasSession.
func TestMockClientHasSession(t *testing.T) {
	mockClient := new(MockClient)

	// Configure the mock to return true (tmux running)
	mockClient.On("HasSession").Return(true, nil)

	// Call the method
	running, err := mockClient.HasSession()

	// Verify
	require.NoError(t, err)
	assert.True(t, running)
	mockClient.AssertCalled(t, "HasSession")
	mockClient.AssertExpectations(t)
}

// TestMockClientSetStatusOption demonstrates MockClient usage for SetStatusOption.
func TestMockClientSetStatusOption(t *testing.T) {
	mockClient := new(MockClient)

	// Configure the mock
	mockClient.On("SetStatusOption", "status-left", "value").Return(nil)

	// Call the method
	err := mockClient.SetStatusOption("status-left", "value")

	// Verify
	require.NoError(t, err)
	mockClient.AssertCalled(t, "SetStatusOption", "status-left", "value")
	mockClient.AssertExpectations(t)
}

// TestMockClientListSessions demonstrates MockClient usage for ListSessions.
func TestMockClientListSessions(t *testing.T) {
	mockClient := new(MockClient)

	// Configure the mock to return a map of sessions
	expectedSessions := map[string]string{
		"$1": "session1",
		"$2": "session2",
	}
	mockClient.On("ListSessions").Return(expectedSessions, nil)

	// Call the method
	sessions, err := mockClient.ListSessions()

	// Verify
	require.NoError(t, err)
	assert.Equal(t, expectedSessions, sessions)
	assert.Equal(t, 2, len(sessions))
	assert.Equal(t, "session1", sessions["$1"])
	mockClient.AssertCalled(t, "ListSessions")
	mockClient.AssertExpectations(t)
}

// TestMockClientRun demonstrates MockClient usage for Run method.
func TestMockClientRun(t *testing.T) {
	// Configure the mock for different commands
	tests := []struct {
		name    string
		args    []string
		stdout  string
		stderr  string
		wantErr bool
	}{
		{
			name:    "list sessions",
			args:    []string{"list-sessions", "-F", "#{session_name}"},
			stdout:  "session1\nsession2",
			stderr:  "",
			wantErr: false,
		},
		{
			name:    "invalid command",
			args:    []string{"invalid-command"},
			stdout:  "",
			stderr:  "unknown command",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			mockClient.On("Run", tt.args).Return(tt.stdout, tt.stderr,
				func() error {
					if tt.wantErr {
						return assert.AnError
					}
					return nil
				}())

			stdout, stderr, err := mockClient.Run(tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.stdout, stdout)
			assert.Equal(t, tt.stderr, stderr)
			mockClient.AssertExpectations(t)
		})
	}
}

// TestMockClientErrorHandling demonstrates MockClient usage for error scenarios.
func TestMockClientErrorHandling(t *testing.T) {
	mockClient := new(MockClient)

	// Configure the mock to return an error
	mockClient.On("GetCurrentContext").Return(TmuxContext{}, ErrTmuxNotRunning)

	// Call the method
	ctx, err := mockClient.GetCurrentContext()

	// Verify
	assert.Error(t, err)
	assert.Equal(t, ErrTmuxNotRunning, err)
	assert.Equal(t, "", ctx.SessionID)
	mockClient.AssertCalled(t, "GetCurrentContext")
	mockClient.AssertExpectations(t)
}

// TestMockClientMultipleCalls demonstrates setting up a mock to be called multiple times.
func TestMockClientMultipleCalls(t *testing.T) {
	mockClient := new(MockClient)

	// Configure the mock to be called 3 times with the same return value
	mockClient.On("GetCurrentContext").Return(TmuxContext{
		SessionID: "$1",
		WindowID:  "@1",
		PaneID:    "%1",
		PanePID:   "1234",
	}, nil).Times(3)

	// Call the method 3 times
	for i := 0; i < 3; i++ {
		ctx, err := mockClient.GetCurrentContext()
		require.NoError(t, err)
		assert.Equal(t, "$1", ctx.SessionID)
	}

	// Assert that the method was called 3 times
	mockClient.AssertNumberOfCalls(t, "GetCurrentContext", 3)
	mockClient.AssertExpectations(t)
}

// TestMockClientNotCalled demonstrates verifying a method was not called.
func TestMockClientNotCalled(t *testing.T) {
	mockClient := new(MockClient)

	// Configure a method that may or may not be called (using Maybe)
	mockClient.On("GetCurrentContext").Return(TmuxContext{}, nil).Maybe()

	// Call a different method
	mockClient.On("HasSession").Return(true, nil)
	mockClient.HasSession()

	// Assert that GetCurrentContext was not called
	mockClient.AssertNotCalled(t, "GetCurrentContext")

	// Assert that HasSession was called
	mockClient.AssertCalled(t, "HasSession")
	mockClient.AssertExpectations(t)
}
